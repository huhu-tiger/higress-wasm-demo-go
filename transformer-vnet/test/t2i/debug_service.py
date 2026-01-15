#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
服务调试脚本 - 检查服务状态和连接性
"""

import subprocess
import requests
import time
from pathlib import Path

project_root = Path(__file__).parent.parent.parent

def check_docker_status():
    """检查 Docker 状态"""
    print("🔍 检查 Docker 状态...")
    
    try:
        # 检查 Docker 是否运行
        result = subprocess.run(["docker", "version"], capture_output=True, text=True, timeout=10)
        if result.returncode == 0:
            print("✅ Docker 正在运行")
        else:
            print("❌ Docker 未运行或有问题")
            print(result.stderr)
            return False
    except Exception as e:
        print(f"❌ 无法检查 Docker 状态: {e}")
        return False
    
    return True

def check_compose_status():
    """检查 docker-compose 状态"""
    print("\n🔍 检查 Docker Compose 状态...")
    
    deploy_dir = project_root / "deploy_dev"
    
    try:
        # 检查容器状态
        result = subprocess.run(
            ["docker-compose", "ps"],
            cwd=deploy_dir,
            capture_output=True,
            text=True,
            timeout=10
        )
        
        print("容器状态:")
        print(result.stdout)
        
        if result.stderr:
            print("错误信息:")
            print(result.stderr)
        
        # 检查日志
        print("\n最近的日志:")
        log_result = subprocess.run(
            ["docker-compose", "logs", "--tail=20"],
            cwd=deploy_dir,
            capture_output=True,
            text=True,
            timeout=10
        )
        print(log_result.stdout)
        
        if log_result.stderr:
            print("日志错误:")
            print(log_result.stderr)
            
    except Exception as e:
        print(f"❌ 无法检查 docker-compose 状态: {e}")

def check_port_status():
    """检查端口状态"""
    print("\n🔍 检查端口状态...")
    
    try:
        # 检查端口是否被占用
        result = subprocess.run(
            ["lsof", "-i", ":10000"],
            capture_output=True,
            text=True,
            timeout=10
        )
        
        if result.stdout:
            print("端口 10000 使用情况:")
            print(result.stdout)
        else:
            print("端口 10000 未被占用")
            
    except FileNotFoundError:
        print("lsof 命令未找到，跳过端口检查")
    except Exception as e:
        print(f"❌ 无法检查端口状态: {e}")

def test_connectivity():
    """测试连接性"""
    print("\n🔍 测试服务连接性...")
    
    endpoints = [
        "http://localhost:10000",
        "http://localhost:10000/health",
        "http://localhost:10000/v1/images/generations"
    ]
    
    for endpoint in endpoints:
        try:
            print(f"测试 {endpoint}...")
            response = requests.get(endpoint, timeout=5)
            print(f"  状态码: {response.status_code}")
            print(f"  响应头: {dict(response.headers)}")
            if len(response.text) < 500:
                print(f"  响应内容: {response.text}")
            else:
                print(f"  响应内容长度: {len(response.text)} 字符")
        except requests.exceptions.ConnectionError:
            print(f"  ❌ 连接被拒绝")
        except requests.exceptions.Timeout:
            print(f"  ❌ 连接超时")
        except Exception as e:
            print(f"  ❌ 连接异常: {e}")
        print()

def check_envoy_config():
    """检查 Envoy 配置"""
    print("\n🔍 检查 Envoy 配置...")
    
    deploy_dir = project_root / "deploy_dev"
    compose_file = deploy_dir / "docker-compose.yml"
    
    if compose_file.exists():
        print("docker-compose.yml 存在")
        
        # 读取配置文件，查看当前使用的 envoy 配置
        with open(compose_file, 'r') as f:
            content = f.read()
            
        # 查找 envoy 配置行
        for line in content.split('\n'):
            if 'envoy' in line and '.yaml' in line:
                print(f"当前 Envoy 配置: {line.strip()}")
    else:
        print("❌ docker-compose.yml 不存在")
    
    # 检查 envoy 配置文件
    envoy_configs = [
        "envoy.yaml",
        "envoy-qwen.yaml", 
        "envoy-gemini.yaml"
    ]
    
    for config in envoy_configs:
        config_path = deploy_dir / "t2i" / config
        if config_path.exists():
            print(f"✅ {config} 存在")
        else:
            print(f"❌ {config} 不存在")

def main():
    print("🚀 Transformer-VNet 服务调试脚本")
    print(f"项目根目录: {project_root}")
    print("="*60)
    
    # 检查各个组件
    check_docker_status()
    check_compose_status()
    check_port_status()
    check_envoy_config()
    test_connectivity()
    
    print("\n" + "="*60)
    print("🔧 建议的调试步骤:")
    print("1. 如果 Docker 未运行，请启动 Docker 服务")
    print("2. 如果容器未启动，尝试手动启动: cd deploy_dev && docker-compose up -d")
    print("3. 如果端口被占用，停止占用进程或更改端口")
    print("4. 检查 docker-compose 日志: cd deploy_dev && docker-compose logs")
    print("5. 如果配置文件有问题，检查 Makefile 中的配置切换逻辑")

if __name__ == "__main__":
    main()