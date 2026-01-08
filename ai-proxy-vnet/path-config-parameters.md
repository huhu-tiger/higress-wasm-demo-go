# AI Proxy 插件 - 路径相关配置参数说明

## 概述

AI Proxy 插件提供了多个与路径处理相关的配置参数，用于控制请求路径的转换、重写和映射。

## 路径相关配置参数

### 1. `openaiCustomUrl` (OpenAI 自定义后端 URL)

**类型：** `string`  
**必填：** 否  
**适用 Provider：** `openai`  
**默认值：** 无

**说明：**
- 用于配置基于 OpenAI 协议的自定义后端 URL
- 格式：`域名:端口/路径` 或 `域名:端口`
- 如果路径以 `/completions`、`/embeddings` 等结尾，会被识别为直接路径（`isDirectCustomPath = true`），不会进行路径拼接
- 如果路径不以这些后缀结尾，插件会自动拼接路径

**示例：**
```yaml
provider:
  type: openai
  openaiCustomUrl: "220.181.114.184:30951/api/v1/services/aigc/multimodal-generation/generation"
  # 或者只配置域名
  openaiCustomUrl: "220.181.114.184:30951"
```

**路径处理逻辑：**
- 如果 `openaiCustomUrl = "domain:port/path"`，且 `path` 不以 `/completions` 等结尾
- 插件会将路径拼接：`/path` + `/chat/completions` = `/path/chat/completions`
- 如果 `path` 以 `/completions` 等结尾，直接使用该路径，不拼接

**代码位置：** `provider/openai.go:75-106`

---

### 2. `capabilities` (AI 能力和路径映射)

**类型：** `map[string]string`  
**必填：** 否  
**适用 Provider：** 所有  
**默认值：** 无

**说明：**
- 用于配置 AI 能力和实际 API 路径的映射关系
- Key：厂商协议能力名称（如 `openai/v1/chatcompletions`）
- Value：实际的 API 路径（如 `/v1/chat/completions`）
- 支持的能力类型：
  - `openai/v1/chatcompletions` - 聊天完成
  - `openai/v1/embeddings` - 文本向量
  - `openai/v1/imagegeneration` - 图像生成
  - `openai/v1/audiospeech` - 音频语音
  - `cohere/v1/rerank` - Cohere 重排序

**示例：**
```yaml
provider:
  type: openai
  capabilities:
    "openai/v1/chatcompletions": "/api/v1/services/aigc/multimodal-generation/generation"
    "openai/v1/embeddings": "/api/v1/services/embeddings/text-embedding/text-embedding"
```

**注意事项：**
- 如果同时配置了 `openaiCustomUrl`，`capabilities` 可能会被自动拼接逻辑覆盖
- 建议在不配置 `openaiCustomUrl` 时使用 `capabilities`

**代码位置：** `provider/provider.go:607-624`

---

### 3. `basePath` (基础路径)

**类型：** `string`  
**必填：** 否  
**适用 Provider：** 所有  
**默认值：** 无

**说明：**
- 用于在请求路径中移除或添加统一前缀
- 需要配合 `basePathHandling` 使用
- 如果只配置 `basePath` 而不配置 `basePathHandling`，默认为 `removePrefix`

**示例：**
```yaml
provider:
  type: openai
  basePath: "/api/v1"
  basePathHandling: "removePrefix"
```

**代码位置：** `provider/provider.go:410-413, 625-629`

---

### 4. `basePathHandling` (基础路径处理方式)

**类型：** `string`  
**必填：** 否  
**适用 Provider：** 所有  
**可选值：**
- `removePrefix` - 移除前缀（默认值）
- `prepend` - 添加前缀

**说明：**
- 指定 `basePath` 的处理方式
- `removePrefix`：从请求路径中移除 `basePath` 前缀
- `prepend`：在请求路径前添加 `basePath` 前缀

**示例：**

**移除前缀：**
```yaml
provider:
  type: openai
  basePath: "/api/v1"
  basePathHandling: "removePrefix"
```
- 请求路径：`/api/v1/chat/completions` → 处理后：`/chat/completions`

**添加前缀：**
```yaml
provider:
  type: openai
  basePath: "/api/v1"
  basePathHandling: "prepend"
```
- 请求路径：`/chat/completions` → 处理后：`/api/v1/chat/completions`

**处理时机：**
- `removePrefix`：在 `TransformRequestHeaders` 之前执行
- `prepend`：在 `TransformRequestHeaders` 之后执行

**代码位置：** `provider/provider.go:973-984`

---

### 5. `subPath` (子路径)

**类型：** `string`  
**必填：** 否  
**适用 Provider：** 所有  
**默认值：** 无

**说明：**
- 如果配置了 `subPath`，将会先移除请求 path 中该前缀，再进行后续处理
- **注意：** 在代码中未找到 `subPath` 的实现，可能已废弃或计划实现

**示例：**
```yaml
provider:
  type: openai
  subPath: "/api/v1"
```

---

## Provider 特定的路径配置

### OpenAI Provider

- **`openaiCustomUrl`** - 自定义后端 URL

### Qwen Provider

- **`qwenDomain`** - 通义千问服务域名（默认：`dashscope.aliyuncs.com`）

**示例：**
```yaml
provider:
  type: qwen
  qwenDomain: "dashscope-finance.aliyuncs.com"  # 金融云服务
```

### vLLM Provider

- **`vllmCustomUrl`** - vLLM 自定义后端 URL（格式类似 `openaiCustomUrl`）

### Generic Provider

- **`genericHost`** - 用于覆盖请求转发的目标 Host

**示例：**
```yaml
provider:
  type: generic
  genericHost: "backend.example.com"
```

---

## 路径处理流程

```
客户端请求路径
    ↓
1. basePath 处理（removePrefix）
    ↓
2. Provider TransformRequestHeaders（路径转换）
    - 使用 openaiCustomUrl 或 capabilities 进行路径映射
    ↓
3. basePath 处理（prepend）
    ↓
最终转发路径
```

## 配置示例

### 示例 1：使用 openaiCustomUrl（路径会被拼接）

```yaml
provider:
  type: openai
  apiTokens:
    - "YOUR_TOKEN"
  openaiCustomUrl: "220.181.114.184:30951/api/v1/services/aigc/multimodal-generation/generation"
  # 实际路径：/api/v1/services/aigc/multimodal-generation/generation/chat/completions
```

### 示例 2：使用 capabilities（精确控制路径）

```yaml
provider:
  type: openai
  apiTokens:
    - "YOUR_TOKEN"
  capabilities:
    "openai/v1/chatcompletions": "/api/v1/services/aigc/multimodal-generation/generation"
  # 实际路径：/api/v1/services/aigc/multimodal-generation/generation
```

### 示例 3：使用 basePath 移除前缀

```yaml
provider:
  type: openai
  apiTokens:
    - "YOUR_TOKEN"
  basePath: "/api/v1"
  basePathHandling: "removePrefix"
  # 请求：/api/v1/chat/completions → 转发：/chat/completions
```

### 示例 4：使用 basePath 添加前缀

```yaml
provider:
  type: openai
  apiTokens:
    - "YOUR_TOKEN"
  basePath: "/api/v1"
  basePathHandling: "prepend"
  # 请求：/chat/completions → 转发：/api/v1/chat/completions
```

### 示例 5：组合使用

```yaml
provider:
  type: openai
  apiTokens:
    - "YOUR_TOKEN"
  basePath: "/api"
  basePathHandling: "removePrefix"
  capabilities:
    "openai/v1/chatcompletions": "/v1/services/aigc/multimodal-generation/generation"
  # 请求：/api/v1/chat/completions
  # 1. 移除 /api → /v1/chat/completions
  # 2. capabilities 映射 → /v1/services/aigc/multimodal-generation/generation
```

---

## 注意事项

1. **路径拼接问题：**
   - 配置 `openaiCustomUrl` 时，如果路径不以 `/completions` 等结尾，会自动拼接
   - 如果后端不支持拼接后的路径，建议使用 `capabilities` 或 `basePath`

2. **capabilities 覆盖：**
   - 如果配置了 `openaiCustomUrl`，`capabilities` 可能会被自动拼接逻辑覆盖
   - 建议在不配置 `openaiCustomUrl` 时使用 `capabilities`

3. **basePath 处理顺序：**
   - `removePrefix` 在路径转换之前执行
   - `prepend` 在路径转换之后执行

4. **路径匹配：**
   - 插件通过路径后缀匹配识别 API 类型（如 `/chat/completions`、`/embeddings`）
   - 也支持正则模式匹配（如 `/v1/files/{file_id}`）

---

## 相关代码文件

- `provider/provider.go` - ProviderConfig 结构体和路径处理逻辑
- `provider/openai.go` - OpenAI Provider 的路径处理
- `util/http.go` - 路径工具函数
- `main.go` - 路径识别和 API 名称映射





