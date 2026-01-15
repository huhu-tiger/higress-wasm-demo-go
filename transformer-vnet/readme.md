# Transformer-VNet

基于 Higress Proxy-WASM 的 AI 模型请求/响应转换插件，用于统一不同 AI 服务提供商的 API 格式。

## 项目简介

Transformer-VNet 是一个运行在 Higress 网关上的 WebAssembly 插件，实现了多个 AI 图像生成服务的请求和响应格式统一转换。通过该插件，可以使用统一的 API 格式调用不同厂商的文生图服务（如豆包、通义千问、Gemini 等），简化多模型集成和切换。

## 核心功能

- **统一 API 格式**: 将不同厂商的 API 格式转换为统一的请求/响应结构
- **多模型支持**: 支持豆包 (Doubao)、通义千问 (Qwen)、Gemini 等主流文生图模型
- **流式响应处理**: 支持流式和非流式响应的自动识别和处理
- **错误统一处理**: 将不同厂商的错误响应转换为统一格式
- **参数智能映射**: 自动处理不同模型间的参数差异（如尺寸格式转换）

## 支持的模型

### 文生图 (Text-to-Image)

| 模型 | Model Header | 文档 |
|------|--------------|------|
| 豆包 SeeDream 4.5 | `volcengine/volcengine/doubao-seedream-4.5` | [doubao-seedream-4.5.md](docs/文生图/doubao-seedream-4.5.md) |
| 通义千问图像生成 | `qwen/qwen-image` | [qwen-image.md](docs/文生图/qwen-image.md) |
| Gemini 3 Image | `gemini/gemini-3-image` | [gemini-3-image.md](docs/文生图/gemini-3-image.md) |

## 快速开始

### 环境要求

- Go 1.24.1+
- TinyGo (用于编译 WASM)
- Higress 网关

### 编译插件

```bash
# 编译 WASM 插件
make build

# 编译后的文件位于 deploy_dev/main.wasm
```

### 部署

1. 将编译好的 `main.wasm` 部署到 Higress 网关
2. 配置 Envoy 路由规则（参考 `deploy_dev/t2i/` 目录下的配置文件）
3. 启动服务

```bash
# 使用 docker-compose 启动开发环境
cd deploy_dev
docker-compose up -d
```

## 使用方式

### 请求格式

使用统一的请求格式调用不同模型，通过 HTTP Header 指定模型和类型：

```bash
curl -X POST http://your-gateway/api/image/generate \
  -H "model: volcengine/volcengine/doubao-seedream-4.5" \
  -H "model_type: t2i" \
  -H "Content-Type: application/json" \
  -d '{
    "input": {
      "messages": [
        {
          "role": "user",
          "content": [
            {
              "text": "生成一幅美丽的风景画"
            }
          ]
        }
      ]
    },
    "parameters": {
      "size": "1024*1024",
      "n": 1
    }
  }'
```

### 响应格式

统一的响应格式：

```json
{
  "request_id": "...",
  "output": {
    "choices": [
      {
        "finish_reason": "stop",
        "message": {
          "role": "assistant",
          "content": [
            {
              "image": "https://..."
            }
          ]
        }
      }
    ]
  },
  "usage": {
    "image_count": 1
  }
}
```

### 错误响应

统一的错误格式：

```json
{
  "code": "400",
  "message": {
    "error": "详细错误信息"
  },
  "data": "higress transformer plugin"
}
```

## 项目结构

```
.
├── bussiness/          # 业务逻辑
│   └── t2i/           # 文生图转换逻辑
│       ├── public.go  # 统一请求/响应结构定义
│       ├── doubao.go  # 豆包模型转换
│       ├── qwen.go    # 通义千问转换
│       └── gemini.go  # Gemini 转换
├── config/            # 配置文件
├── deploy_dev/        # 部署配置
│   ├── t2i/          # 文生图 Envoy 配置
│   └── docker-compose.yml
├── docs/              # 文档
│   └── 文生图/        # 各模型 API 文档
├── test/              # 测试文件
├── wlog/              # 日志工具
├── main.go            # 插件入口
└── utils.go           # 工具函数
```

## 开发指南

### 添加新模型支持

1. 在 `docs/文生图/` 目录下创建模型文档
2. 在 `bussiness/t2i/` 目录下创建对应的转换文件
3. 实现 `TransformRequest` 和 `TransformResponse` 方法
4. 在 `public.go` 中注册新模型

### 日志调试

插件使用自定义日志工具 `wlog`，会输出详细的转换过程：

- 请求转换前后的 JSON 对比
- 响应转换前后的 JSON 对比
- 流式响应的处理过程
- 错误信息和堆栈

### 测试

```bash
# 运行测试
cd test/t2i
go test -v
```

## 配置说明

### 插件配置

在 `config/config.go` 中可以调整：

- `DefaultMaxRequestBodyBytes`: 请求体最大大小（默认 1MB）
- `DefaultMaxResponseBodyBytes`: 响应体最大大小（默认 25MB）
- `MaxPrintSize`: 日志打印最大大小（默认 1KB）

### Envoy 配置

参考 `deploy_dev/t2i/` 目录下的配置文件，可以配置：

- 路由规则
- 上游服务地址
- WASM 插件加载
- 超时设置

## 技术栈

- **语言**: Go 1.24.1
- **框架**: Higress Proxy-WASM Go SDK
- **编译**: TinyGo (WASM target)
- **依赖管理**: Go Modules

## 许可证

Apache License 2.0

## 贡献

欢迎提交 Issue 和 Pull Request！

## 联系方式

如有问题或建议，请通过 Issue 反馈。
