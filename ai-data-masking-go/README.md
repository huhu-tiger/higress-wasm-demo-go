# AI 数据脱敏插件 (Golang 版本)

这是 AI 数据脱敏插件的 Golang 实现版本，功能与 Rust 版本相同。

## 功能说明

对请求/返回中的敏感词拦截、替换

### 处理数据范围
- openai协议：请求/返回对话内容
- jsonpath：只处理指定字段
- raw：整个请求/返回body

### 敏感词拦截
- 处理数据范围中出现敏感词直接拦截，返回预设错误信息
- 支持系统内置敏感词库和自定义敏感词
- 系统敏感词库：从 `resources/` 目录下的资源文件自动加载（中文和英文敏感词）
- 通过 `system_deny=true` 启用系统敏感词过滤功能

### 敏感词替换
- 将请求数据中出现的敏感词替换为脱敏字符串，传递给后端服务。可保证敏感数据不出域
- 部分脱敏数据在后端服务返回后可进行还原
- 自定义规则支持标准正则和grok规则，替换字符串支持变量替换

## 处理阶段说明

插件在请求阶段和响应阶段都会进行敏感词检测和处理，但处理方式有所不同：

### 请求阶段处理（onHttpRequestBody）

**检测时机**：在请求发送到后端服务之前

**处理逻辑**：
1. **敏感词拦截**：如果检测到敏感词（通过 `deny_words` 或 `system_deny`），**直接拒绝请求**，返回拒绝响应，**不会发送到后端服务**
   - 系统敏感词（`system_deny=true`）会在所有请求格式中进行检测：
     - OpenAI 格式：检测 `messages[*].content` 和 `messages[*].reasoning_content`
     - JSONPath 格式：检测配置的 JSONPath 路径对应的字段
     - Raw 格式：检测整个请求体
2. **敏感词替换**：如果没有检测到敏感词，但匹配到 `replace_roles` 规则，则**替换敏感词后继续发送到后端服务**

**响应格式**：
- **非流式请求**（`stream: false`）：返回 JSON 格式的拒绝响应
- **流式请求**（`stream: true`）：返回 SSE 格式的拒绝响应（包含 `data: [DONE]`）

**示例**：
```json
// 请求包含敏感词 "敏感词"
{
  "model": "gpt-3.5-turbo",
  "messages": [{"role": "user", "content": "这是一个敏感词测试"}]
}

// 直接返回拒绝响应，不会发送到后端
{
  "id": "chatcmpl-xxx",
  "choices": [{
    "message": {
      "role": "assistant",
      "content": "提问或回答中包含敏感词，已被屏蔽"
    }
  }]
}
```

### 响应阶段处理（onHttpResponseBody / onHttpStreamingResponseBody）

**检测时机**：在后端服务返回响应之后

**处理逻辑**：根据 `response_deny_plot.plot` 配置的策略进行处理

#### 非流式响应处理

**触发条件**：`Content-Type` 不是 `text/event-stream`

**处理方式**：
- **stop**（默认）：检测到敏感词后，返回拒绝消息，替换整个响应体
- **replace**：检测到敏感词后，将敏感词替换为 `response_deny_plot.value`，继续返回响应

**示例**：
```json
// 后端返回的响应包含敏感词
{
  "choices": [{
    "message": {
      "content": "这是一个敏感词测试"
    }
  }]
}

// stop 模式：返回拒绝消息
{
  "choices": [{
    "message": {
      "content": "提问或回答中包含敏感词，已被屏蔽"
    }
  }]
}

// replace 模式（value: "***"）：替换敏感词
{
  "choices": [{
    "message": {
      "content": "这是一个***测试"
    }
  }]
}
```

#### 流式响应处理

**触发条件**：`Content-Type` 包含 `text/event-stream`

**处理方式**：
- **stop**：检测到敏感词后，立即停止后续 chunk 传输，返回拒绝消息
- **replace**：在流式传输过程中，实时将敏感词替换为 `response_deny_plot.value`，继续传输
- **rollback**：发送 rollback 事件，通知客户端回退到指定序列号（仅流式响应支持）

**示例**：
```
// 流式响应事件流
data: {"id":"chatcmpl-xxx","choices":[{"delta":{"content":"正常"}}],"seq":1}

data: {"id":"chatcmpl-xxx","choices":[{"delta":{"content":"敏感"}}],"seq":2}

data: {"id":"chatcmpl-xxx","choices":[{"delta":{"content":"词"}}],"seq":3}

// stop 模式：立即停止，返回拒绝消息
data: {"id":"chatcmpl-xxx","choices":[{"delta":{"content":"提问或回答中包含敏感词，已被屏蔽"}}]}

data: [DONE]

// replace 模式（value: "***"）：替换敏感词，继续传输
data: {"id":"chatcmpl-xxx","choices":[{"delta":{"content":"正常"}}],"seq":1}

data: {"id":"chatcmpl-xxx","choices":[{"delta":{"content":"***"}}],"seq":2}

data: {"id":"chatcmpl-xxx","choices":[{"delta":{"content":""}}],"seq":3}

// rollback 模式：发送回退事件
data: {"id":"chatcmpl-xxx","choices":[{"delta":{"content":"正常"}}],"seq":1}

data: {"id":"chatcmpl-xxx","choices":[{"delta":{"content":"敏感"}}],"seq":2}

data: {"id":"chatcmpl-xxx","choices":[{"delta":{"content":"词"}}],"seq":3}

data: {"type":"rollback","rollback_to":[2,3],"reason":"content_safety"}
```

### 处理流程对比

| 阶段 | 检测到敏感词 | 处理方式 | 是否发送到后端 |
|------|------------|---------|---------------|
| **请求阶段** | 是 | 直接拒绝，返回拒绝响应 | ❌ 否 |
| **请求阶段** | 否（但匹配 replace_roles） | 替换敏感词后发送 | ✅ 是 |
| **响应阶段（非流式）** | 是 | 根据 `response_deny_plot.plot` 处理 | ✅ 已发送 |
| **响应阶段（流式）** | 是 | 根据 `response_deny_plot.plot` 处理 | ✅ 已发送 |

## 运行属性

插件执行阶段：`认证阶段`
插件执行优先级：`991`

## 配置字段

| 名称 | 数据类型 | 默认值 | 描述 |
| -------- | --------  | -------- | -------- |
| deny_openai | bool | true | 对openai协议进行拦截 |
| deny_jsonpath | array of string | [] | 对指定jsonpath拦截 |
| deny_raw | bool | false | 对原始body拦截 |
| system_deny | bool | false | 开启内置拦截规则（启用后会自动使用 resources/ 目录下的系统敏感词库进行过滤） |
| deny_code | uint32 | 200 | 拦截时http状态码 |
| deny_message | string | 提问或回答中包含敏感词，已被屏蔽 | 拦截时ai返回消息 |
| deny_raw_message | string | {"errmsg":"提问或回答中包含敏感词，已被屏蔽"} | 非openai拦截时返回内容 |
| deny_content_type | string | application/json | 非openai拦截时返回content_type头 |
| deny_words | array of string | [] | 自定义敏感词列表 |
| replace_roles | array | [] | 自定义敏感词正则替换 |
| replace_roles.regex | string | - | 规则正则(内置GROK规则) |
| replace_roles.type | string | - | 替换类型，可选值：replace、hash |
| replace_roles.restore | bool | false | 是否恢复 |
| replace_roles.value | string | - | 替换值（支持正则变量） |
| response_deny_plot | object | - | 响应拒绝处理方式（仅适用于响应阶段，请求阶段检测到敏感词直接拒绝） |
| response_deny_plot.plot | string | stop | 处理方式，可选值：stop（停止）、replace（替换）、rollback（回退，仅流式响应） |
| response_deny_plot.value | string | - | 当plot为replace时，替换为value；当plot为stop或rollback时，不使用此字段 |
| max_buffer_chunk_count | uint32 | 30 | 流式响应中敏感词检测的最大chunk个数 |
| max_stream_chunk_buffer_len | uint32 | 2048 | 流式响应中敏感词检测的最大chunk大小（字节数） |

## response_deny_plot 处理方式说明

**重要说明**：`response_deny_plot.plot` **仅适用于响应阶段**，不适用于请求阶段。

- **请求阶段**：检测到敏感词后，直接拒绝请求，不会发送到后端服务
- **响应阶段**：检测到敏感词后，根据 `response_deny_plot.plot` 配置的策略进行处理

`response_deny_plot.plot` 用于配置当检测到**响应中包含敏感词**时的处理策略。支持三种处理方式：

### 1. stop（停止）- 默认方式

**行为**：检测到敏感词后，立即停止响应，返回拒绝消息。

**适用场景**：需要严格阻止敏感内容返回给用户。

**效果示例**：

- **非流式响应**：
  ```json
  {
    "id": "chatcmpl-xxx",
    "object": "chat.completion",
    "created": 123,
    "model": "gpt-3.5-turbo",
    "choices": [{
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "提问或回答中包含敏感词，已被屏蔽"
      }
    }]
  }
  ```

- **流式响应**：
  - 检测到敏感词后，立即停止后续 chunk 的传输
  - 返回包含拒绝消息的最后一个 chunk，然后结束流

**配置示例**：
```yaml
response_deny_plot:
  plot: "stop"
  value: ""  # stop 模式下 value 不使用
```

### 2. replace（替换）

**行为**：检测到敏感词后，将敏感词替换为配置的 `value`，继续返回响应。

**适用场景**：需要隐藏敏感内容但保持响应完整性，用户体验更好。

**效果示例**：

假设响应内容为：`"这是一个敏感词测试"`，配置 `value: "***"`

- **非流式响应**：
  ```json
  {
    "choices": [{
      "message": {
        "content": "这是一个***测试"
      }
    }]
  }
  ```

- **流式响应**：
  - 在流式传输过程中，实时将敏感词替换为 `value`
  - 继续传输后续内容，不中断流

**配置示例**：
```yaml
response_deny_plot:
  plot: "replace"
  value: "***"  # 替换敏感词的字符串
```

### 3. rollback（回退）- 仅流式响应

**行为**：检测到敏感词后，发送 rollback 事件通知客户端回退到指定序列号，客户端需要根据事件回退已显示的内容。

**适用场景**：需要客户端主动处理敏感内容回退，适用于需要精确控制显示内容的场景。

**效果示例**：

假设流式响应中，序列号 8、9、10 的 chunk 包含敏感词：

- **流式响应事件**：
  ```
  data: {"id":"chatcmpl-xxx","choices":[{"delta":{"content":"正常内容"}}],"seq":8}
  
  data: {"id":"chatcmpl-xxx","choices":[{"delta":{"content":"敏感"}}],"seq":9}
  
  data: {"id":"chatcmpl-xxx","choices":[{"delta":{"content":"词内容"}}],"seq":10}
  
  data: {"type":"rollback","rollback_to":[8,9,10],"reason":"content_safety"}
  ```

- **客户端处理**：
  1. 客户端收到 rollback 事件后，需要回退并移除序列号 8、9、10 对应的内容
  2. 可以选择显示错误提示或继续等待后续内容

**配置示例**：
```yaml
response_deny_plot:
  plot: "rollback"
  value: ""  # rollback 模式下 value 不使用
```

**注意事项**：
- `rollback` 模式仅适用于流式响应（`stream: true`）
- 客户端需要实现 rollback 事件的处理逻辑
- rollback 事件格式：`{"type":"rollback","rollback_to":[seq1,seq2,...],"reason":"content_safety"}`

## 配置示例

```yaml
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: ai-data-masking
  namespace: higress-system
spec:
  selector:
    matchLabels:
      higress: higress-system-higress-gateway
  defaultConfig:
    system_deny: true
    deny_openai: true
    deny_jsonpath:
      - "$.messages[*].content"
    deny_raw: true
    deny_code: 200
    deny_message: "提问或回答中包含敏感词，已被屏蔽"
    deny_raw_message: "{\"errmsg\":\"提问或回答中包含敏感词，已被屏蔽\"}"
    deny_content_type: "application/json"
    deny_words: 
      - "自定义敏感词1"
      - "自定义敏感词2"
    replace_roles:
      - regex: "%{MOBILE}"
        type: "replace"
        value: "****"
      - regex: "%{EMAILLOCALPART}@%{HOSTNAME:domain}"
        type: "replace"
        restore: true
        value: "****@$domain"
      - regex: "%{IP}"
        type: "replace"
        restore: true
        value: "***.***.***.***"
      - regex: "%{IDCARD}"
        type: "replace"
        value: "****"
      - regex: "sk-[0-9a-zA-Z]*"
        restore: true
        type: "hash"
    response_deny_plot:
      plot: "stop"
      value: ""
    max_buffer_chunk_count: 30
    max_stream_chunk_buffer_len: 2048
  url: oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/ai-data-masking:1.0.0
  phase: AUTHN
  priority: 991
```

## 构建

```bash
cd /data/work/higress/plugins/wasm-go
PLUGIN_NAME=ai-data-masking make build
```

## 相关说明

- 流模式中如果脱敏后的词被多个chunk拆分，可能无法进行还原
- 流模式中，如果敏感词语被多个chunk拆分，可能会有敏感词的一部分返回给用户的情况
- grok 内置规则列表 https://help.aliyun.com/zh/sls/user-guide/grok-patterns
- 内置敏感词库数据来源 https://github.com/houbb/sensitive-word-data/tree/main/src/main/resources
- 由于敏感词列表是在文本分词后进行匹配的，所以请将 `deny_words` 设置为单个单词，英文多单词情况如 `hello word` 可能无法匹配

## 与 Rust 版本的差异

1. **敏感词检测**：当前使用简单的字符串包含匹配，Rust 版本使用 jieba 分词
2. **GROK 支持**：当前是简化实现，支持常见模式
3. **流式响应处理**：基础实现，需要进一步完善 SSE 解析
4. **系统敏感词库**：已实现从资源文件加载

## 待完善功能

- [x] 系统敏感词库从资源文件加载,加载resources下的文件，作为资源文件
- [x] system_deny=true 增加系统敏感词过滤敏感词（已实现，系统敏感词库会在程序启动时自动加载，当 system_deny=true 时启用过滤）
- [x] 系统敏感词加入到 请求参数的过滤，请求参数匹配到敏感词直接返回（已实现，在请求阶段，系统敏感词会在所有请求格式（OpenAI、JSONPath、Raw）中进行检测，匹配到敏感词会直接拒绝请求并返回拒绝响应）