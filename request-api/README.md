# HTTP Call WASM 插件说明文档

## 1. 功能概述

本插件演示了如何在 Higress WASM 插件中发起外部 HTTP 调用。插件在请求处理阶段会：
- 向外部服务发起 HTTP POST 请求
- 将外部服务的响应添加到请求头中
- 继续处理原始请求

## 2. Envoy Cluster 配置设计说明

### 2.1 为什么需要在 Envoy 中配置 Cluster？

**核心原因：Envoy 的架构设计要求所有上游服务连接必须通过 Cluster 配置定义。**

#### 2.1.1 Envoy 的架构设计

Envoy 使用 **Cluster** 来管理所有上游服务的连接。Cluster 定义了：
- 目标服务的地址和端口
- 连接超时、负载均衡策略
- 健康检查、熔断等高级配置

#### 2.1.2 WASM 插件的 HTTP 调用流程

当 WASM 插件调用 `config.client.Post()` 时，实际执行流程如下：

```
WASM 插件代码
    ↓
生成 Cluster 名称 (例如: outbound|8000||172.22.220.21.static)
    ↓
调用 proxywasm.DispatchHttpCall(clusterName, ...)
    ↓
Envoy 根据 clusterName 查找配置中的 Cluster 定义
    ↓
如果找到 → 使用 Cluster 配置建立连接 → 发送请求
如果未找到 → 返回错误: "bad argument"
```

#### 2.1.3 Cluster 名称生成规则

根据不同的 Cluster 类型，名称生成规则如下：

**StaticIpCluster** (静态 IP 集群):
```go
// 格式: outbound|{Port}||{ServiceName}.static
// 示例: outbound|8000||172.22.220.21.static
func (c StaticIpCluster) ClusterName() string {
    return fmt.Sprintf("outbound|%d||%s.static", c.Port, c.ServiceName)
}
```

**FQDNCluster** (FQDN 集群):
```go
// 格式: outbound|{Port}||{FQDN}
// 示例: outbound|80||httpbin.org
func (c FQDNCluster) ClusterName() string {
    return fmt.Sprintf("outbound|%d||%s", c.Port, c.FQDN)
}
```

**K8sCluster** (Kubernetes 集群):
```go
// 格式: outbound|{Port}|{Version}|{ServiceName}.{Namespace}.svc.cluster.local
// 示例: outbound|80|v1|my-service.default.svc.cluster.local
func (c K8sCluster) ClusterName() string {
    return fmt.Sprintf("outbound|%d|%s|%s.%s.svc.cluster.local",
        c.Port, c.Version, c.ServiceName, namespace)
}
```

#### 2.1.4 为什么必须预先配置？

1. **Envoy 不会动态创建 Cluster**
   - WASM 插件只能通过名称引用已存在的 Cluster
   - 这是 Envoy 的配置驱动设计原则

2. **统一管理和监控**
   - 所有上游服务连接都通过配置定义
   - 便于统一管理、监控和追踪

3. **安全性考虑**
   - 避免插件随意连接未知服务
   - 所有外部连接都需要管理员显式配置

#### 2.1.5 如果未配置会发生什么？

如果 Envoy 配置中没有对应的 Cluster，会出现以下错误：

```
HTTP call failed: error status returned by host: bad argument
```

因为 Envoy 找不到对应的 Cluster 定义，无法建立连接。

### 2.2 如何配置 Cluster？

在 `envoy.yaml` 的 `clusters` 部分添加对应的 Cluster 配置：

#### 示例 1: StaticIpCluster 配置

**代码中的配置：**
```go
config.client = wrapper.NewClusterClient(wrapper.StaticIpCluster{
    ServiceName: "172.22.220.21",
    Port:        int64(8000),
    Host:        "172.22.220.21",
})
```

**生成的 Cluster 名称：** `outbound|8000||172.22.220.21.static`

**envoy.yaml 中的配置：**
```yaml
clusters:
  - name: outbound|8000||172.22.220.21.static
    connect_timeout: 30s
    type: STATIC
    dns_lookup_family: V4_ONLY
    lb_policy: ROUND_ROBIN
    load_assignment:
      cluster_name: outbound|8000||172.22.220.21.static
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: 172.22.220.21
                port_value: 8000
```

#### 示例 2: FQDNCluster 配置

**代码中的配置：**
```go
config.client = wrapper.NewClusterClient(wrapper.FQDNCluster{
    FQDN: "httpbin.org",
    Port: int64(80),
    Host: "httpbin.org",
})
```

**生成的 Cluster 名称：** `outbound|80||httpbin.org`

**envoy.yaml 中的配置：**
```yaml
clusters:
  - name: outbound|80||httpbin.org
    connect_timeout: 30s
    type: LOGICAL_DNS
    dns_lookup_family: V4_ONLY
    lb_policy: ROUND_ROBIN
    load_assignment:
      cluster_name: outbound|80||httpbin.org
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: httpbin.org
                port_value: 80
```

### 2.3 Cluster 配置参数说明

| 参数 | 说明 | 示例值 |
|------|------|--------|
| `name` | Cluster 名称，必须与代码生成的名称完全一致 | `outbound|8000||172.22.220.21.static` |
| `type` | Cluster 类型：`STATIC`（静态IP）、`LOGICAL_DNS`（DNS解析） | `STATIC` |
| `connect_timeout` | 连接超时时间 | `30s` |
| `dns_lookup_family` | DNS 查询类型：`V4_ONLY`、`V6_ONLY`、`AUTO` | `V4_ONLY` |
| `lb_policy` | 负载均衡策略：`ROUND_ROBIN`、`LEAST_REQUEST` 等 | `ROUND_ROBIN` |
| `address` | 目标服务地址（IP 或域名） | `172.22.220.21` |
| `port_value` | 目标服务端口 | `8000` |

## 3. 代码示例

### 3.1 基本用法

```go
func onHttpRequestHeaders(ctx wrapper.HttpContext, config MyConfig) types.Action {
    // 创建 Cluster 客户端
    config.client = wrapper.NewClusterClient(wrapper.StaticIpCluster{
        ServiceName: "172.22.220.21",
        Port:        int64(8000),
        Host:        "172.22.220.21",
    })
    
    // 发起 HTTP POST 请求
    err := config.client.Post(
        "/echo/post",                    // 路径
        [][2]string{},                    // 请求头（空）
        []byte(`{"message": "hello"}`),  // 请求体
        func(statusCode int, responseHeaders http.Header, responseBody []byte) {
            // 回调函数：处理响应
            log.Infof("HTTP call response: status=%d, body=%s", statusCode, string(responseBody))
            proxywasm.AddHttpRequestHeader("X-External-Response", string(responseBody))
            proxywasm.ResumeHttpRequest()
        },
        uint32(5000), // 超时时间（毫秒）
    )
    
    if err != nil {
        log.Errorf("HTTP call failed: %v", err)
        return types.ActionContinue
    }
    
    // 暂停请求处理，等待异步 HTTP 调用完成
    return types.HeaderStopAllIterationAndWatermark
}
```

### 3.2 带请求头的调用

```go
headers := [][2]string{
    {"Content-Type", "application/json"},
    {"Authorization", "Bearer token123"},
}

err := config.client.Post(
    "/api/endpoint",
    headers,
    []byte(`{"data": "value"}`),
    callback,
    uint32(5000),
)
```

### 3.3 使用不同的 Cluster 类型

```go
// 方式 1: 静态 IP
client1 := wrapper.NewClusterClient(wrapper.StaticIpCluster{
    ServiceName: "192.168.1.100",
    Port:        int64(8080),
    Host:        "api.example.com",
})

// 方式 2: FQDN
client2 := wrapper.NewClusterClient(wrapper.FQDNCluster{
    FQDN: "httpbin.org",
    Port: int64(80),
    Host: "httpbin.org",
})

// 方式 3: Kubernetes Service
client3 := wrapper.NewClusterClient(wrapper.K8sCluster{
    ServiceName: "my-service",
    Namespace:   "default",
    Port:        int64(8080),
    Version:     "v1",
    Host:        "my-service.default.svc.cluster.local",
})
```

## 4. 构建与运行

### 4.1 构建 WASM 文件

```bash
# 进入项目目录
cd /data/work/go/higress-wasm-demo-go/http-call

# 构建 WASM 文件
tinygo build -o main.wasm -scheduler=none -target=wasi main.go

# 复制到部署目录
cp main.wasm deploy_dev/
```

### 4.2 配置 Envoy

确保 `deploy_dev/envoy.yaml` 中：
1. 配置了 WASM 插件
2. **配置了对应的 Cluster**（重要！）

### 4.3 启动服务

```bash
cd deploy_dev
docker-compose up -d
```

### 4.4 测试

```bash
# 发送请求
curl -i http://127.0.0.1:10000/get

# 查看 Envoy 日志
docker-compose logs -f envoy
```

## 5. 常见问题

### 5.1 错误：`HTTP call failed: error status returned by host: bad argument`

**原因：** Envoy 配置中缺少对应的 Cluster 定义。

**解决方案：**
1. 检查代码中使用的 Cluster 类型和参数
2. 根据 Cluster 名称生成规则，确定应该配置的 Cluster 名称
3. 在 `envoy.yaml` 的 `clusters` 部分添加对应的配置

### 5.2 如何确定 Cluster 名称？

查看 `cluster_wrapper.go` 中对应 Cluster 类型的 `ClusterName()` 方法，或者：
1. 查看代码中使用的 Cluster 类型和参数
2. 根据生成规则手动计算
3. 在日志中查找（如果 Envoy 有相关错误日志）

### 5.3 Cluster 名称不匹配

**症状：** 配置了 Cluster，但仍然报错。

**检查项：**
- Cluster 名称必须完全匹配（包括大小写、特殊字符）
- 检查是否有拼写错误
- 确认端口号和 ServiceName 是否正确

### 5.4 连接超时

**可能原因：**
- 目标服务不可达
- 网络配置问题
- Cluster 配置中的地址或端口错误

**排查步骤：**
1. 检查目标服务是否正常运行
2. 验证 Cluster 配置中的地址和端口
3. 增加 `connect_timeout` 值
4. 检查网络连通性

## 6. 最佳实践

1. **预先规划 Cluster 配置**
   - 在开发阶段就确定需要的外部服务
   - 提前在 Envoy 配置中定义所有 Cluster

2. **使用有意义的 ServiceName**
   - 使用描述性的名称，便于识别和管理
   - 避免使用纯 IP 地址作为 ServiceName（虽然技术上可行）

3. **统一管理 Cluster 配置**
   - 将 Cluster 配置集中管理
   - 使用配置模板或工具生成

4. **监控和日志**
   - 启用 Envoy 的访问日志
   - 监控 Cluster 的健康状态
   - 记录 HTTP 调用的成功率和延迟

5. **错误处理**
   - 在回调函数中处理各种 HTTP 状态码
   - 实现重试机制（如需要）
   - 记录详细的错误日志

## 7. 参考资料

- [Envoy Cluster 配置文档](https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/cluster/v3/cluster.proto)
- [Higress WASM Go SDK](https://github.com/higress-group/proxy-wasm-go-sdk)
- [WASM Go Wrapper](https://github.com/higress-group/wasm-go)

