#!/usr/bin/env python3
"""
自动生成 Envoy Cluster 配置工具

从 envoy.yaml 中的 WASM 配置提取 endpoint 信息，自动生成对应的 cluster 配置。
参考: https://higress.cn/docs/ebook/wasm16/
"""

import yaml
import json
import re
import sys
import os
from typing import Dict, List, Optional, Tuple


def extract_wasm_config(envoy_config: Dict) -> Optional[Dict]:
    """从 Envoy 配置中提取 WASM 插件的 JSON 配置"""
    try:
        listeners = envoy_config.get('static_resources', {}).get('listeners', [])
        for listener in listeners:
            filter_chains = listener.get('filter_chains', [])
            for chain in filter_chains:
                filters = chain.get('filters', [])
                for filter in filters:
                    if filter.get('name') == 'envoy.filters.network.http_connection_manager':
                        typed_config = filter.get('typed_config', {})
                        http_filters = typed_config.get('http_filters', [])
                        for http_filter in http_filters:
                            if http_filter.get('name') == 'wasmdemo':
                                wasm_config = http_filter.get('typed_config', {}).get('value', {})
                                config_value = wasm_config.get('config', {}).get('configuration', {})
                                if config_value.get('@type') == 'type.googleapis.com/google.protobuf.StringValue':
                                    json_str = config_value.get('value', '{}')
                                    return json.loads(json_str)
    except Exception as e:
        print(f"提取 WASM 配置时出错: {e}", file=sys.stderr)
    return None


def generate_cluster_name(service_name: str, service_port: int) -> str:
    """
    生成 Cluster 名称
    根据 Higress WASM SDK 的 FQDNCluster 规则: outbound|{Port}||{FQDN}
    """
    return f"outbound|{service_port}||{service_name}"


def is_static_ip(address: str) -> bool:
    """判断是否为静态 IP 地址"""
    ip_pattern = r'^(\d{1,3}\.){3}\d{1,3}$'
    return bool(re.match(ip_pattern, address))


def generate_cluster_config(service_name: str, service_port: int, service_host: Optional[str] = None) -> Dict:
    """
    生成 Envoy Cluster 配置
    
    Args:
        service_name: 服务名称（FQDN 或 IP）
        service_port: 服务端口
        service_host: 服务主机（可选，默认使用 service_name）
    """
    if service_host is None:
        service_host = service_name
    
    cluster_name = generate_cluster_name(service_name, service_port)
    
    # 判断是否为静态 IP
    if is_static_ip(service_name):
        # 静态 IP 配置
        cluster_name = f"outbound|{service_port}||{service_name}.static"
        cluster_config = {
            'name': cluster_name,
            'connect_timeout': '30s',
            'type': 'STATIC',
            'dns_lookup_family': 'V4_ONLY',
            'lb_policy': 'ROUND_ROBIN',
            'load_assignment': {
                'cluster_name': cluster_name,
                'endpoints': [{
                    'lb_endpoints': [{
                        'endpoint': {
                            'address': {
                                'socket_address': {
                                    'address': service_name,
                                    'port_value': service_port
                                }
                            }
                        }
                    }]
                }]
            }
        }
    else:
        # FQDN 配置
        cluster_config = {
            'name': cluster_name,
            'connect_timeout': '30s',
            'type': 'LOGICAL_DNS',
            'dns_lookup_family': 'V4_ONLY',
            'lb_policy': 'ROUND_ROBIN',
            'load_assignment': {
                'cluster_name': cluster_name,
                'endpoints': [{
                    'lb_endpoints': [{
                        'endpoint': {
                            'address': {
                                'socket_address': {
                                    'address': service_name,
                                    'port_value': service_port
                                }
                            }
                        }
                    }]
                }]
            }
        }
        
        # 如果端口是 443，添加 TLS 配置
        if service_port == 443:
            cluster_config['transport_socket'] = {
                'name': 'envoy.transport_sockets.tls',
                'typed_config': {
                    '@type': 'type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext',
                    'sni': service_name
                }
            }
    
    return cluster_config


def find_existing_cluster(clusters: List[Dict], cluster_name: str) -> Optional[int]:
    """查找已存在的 cluster，返回索引"""
    for i, cluster in enumerate(clusters):
        if cluster.get('name') == cluster_name:
            return i
    return None


def update_envoy_config(envoy_file: str, output_file: Optional[str] = None):
    """
    更新 Envoy 配置文件，自动添加缺失的 cluster 配置
    
    Args:
        envoy_file: Envoy 配置文件路径
        output_file: 输出文件路径（如果为 None，则覆盖原文件）
    """
    if output_file is None:
        output_file = envoy_file
    
    # 读取 Envoy 配置
    with open(envoy_file, 'r', encoding='utf-8') as f:
        envoy_config = yaml.safe_load(f)
    
    # 提取 WASM 配置
    wasm_config = extract_wasm_config(envoy_config)
    if not wasm_config:
        print("未找到 WASM 配置", file=sys.stderr)
        return False
    
    # 提取 endpoint 信息
    http_service = wasm_config.get('http_service', {})
    endpoint = http_service.get('endpoint', {})
    
    service_name = endpoint.get('service_name')
    service_port = endpoint.get('service_port', 80)
    service_host = endpoint.get('service_host')
    
    if not service_name:
        print("未找到 endpoint.service_name 配置", file=sys.stderr)
        return False
    
    # 生成 cluster 配置
    cluster_config = generate_cluster_config(service_name, service_port, service_host)
    cluster_name = cluster_config['name']
    
    # 获取现有的 clusters
    clusters = envoy_config.get('static_resources', {}).get('clusters', [])
    
    # 检查 cluster 是否已存在
    existing_idx = find_existing_cluster(clusters, cluster_name)
    
    if existing_idx is not None:
        print(f"Cluster '{cluster_name}' 已存在，跳过添加")
        # 可以选择更新现有配置
        # clusters[existing_idx] = cluster_config
    else:
        print(f"添加新的 Cluster: {cluster_name}")
        clusters.append(cluster_config)
        envoy_config['static_resources']['clusters'] = clusters
    
    # 写入更新后的配置
    with open(output_file, 'w', encoding='utf-8') as f:
        yaml.dump(envoy_config, f, default_flow_style=False, allow_unicode=True, sort_keys=False)
    
    print(f"配置已更新: {output_file}")
    return True


def main():
    """主函数"""
    if len(sys.argv) < 2:
        print("用法: python3 generate_clusters.py <envoy.yaml> [output.yaml]")
        print("示例: python3 generate_clusters.py deploy_dev/envoy.yaml")
        sys.exit(1)
    
    envoy_file = sys.argv[1]
    output_file = sys.argv[2] if len(sys.argv) > 2 else None
    
    if not os.path.exists(envoy_file):
        print(f"文件不存在: {envoy_file}", file=sys.stderr)
        sys.exit(1)
    
    success = update_envoy_config(envoy_file, output_file)
    sys.exit(0 if success else 1)


if __name__ == '__main__':
    main()

