## WASM Demo 使用说明

### 1. 功能概述
- `main.go` 展示 Higress WASM 插件的基本骨架：注册 `ParseConfig`、请求头/体、响应头/体处理函数，并在每个阶段打印带序号的日志，方便跟踪执行顺序。
- 插件通过配置项 `mockEnable` 控制是否直接返回模拟响应；关闭 mock 时请求会透传到 `httpbin` 后端。

### 2. 执行顺序
插件在一次 HTTP 交互中按以下顺序触发（与 `main.go` 中日志 `[1]~[5]` 对应）：
1. **[1] parseConfig**：启动时解析 YAML→JSON 配置，记录 `mockEnable`。
2. **[2] onHttpRequestHeaders**：添加请求头 `hello: world`；若 `mockEnable=true`，立即返回 `200 hello world`。
3. **[3] onHttpRequestBody**：打印请求体内容（如有），然后 `ActionContinue`。
4. **[4] onHttpResponseHeaders**：添加响应头 `x-wasm-demo: enabled`，mock 模式下再加 `x-wasm-mock: true`。
5. **[5] onHttpResponseBody**：打印响应体内容（如有），继续下游。

### 3. `types.Action` 常用含义
- `types.HeaderContinue`：继续处理头部（请求/响应阶段）。
- `types.HeaderStopIteration`：暂时停止头部迭代，等待宿主随后恢复，常用于需异步完成后再继续处理请求/响应头。
- `types.ActionContinue`：继续传递当前方向的数据（Body、Trailers）。
- `types.ActionPause`：暂停流程，常用于等待异步结果或直接返回自定义响应。

### 4. 构建与运行
```bash
# 生成 main.wasm、复制到 deploy_dev，并启动本地 Envoy + httpbin
./dev.sh

# 观察插件日志（含序号）
docker-compose -f deploy_dev/docker-compose.yml logs -f envoy

# 使用 curl 触发请求
curl -i http://127.0.0.1:10000/get
curl -i http://127.0.0.1:10000/post -d '{"foo":"bar"}' -H 'Content-Type: application/json'
```

如需切换 mock 行为，修改 `deploy_dev/envoy.yaml` 中的 `mockEnable` 值并重新执行 `dev.sh`。

