# AI 数据脱敏插件 - 数据处理流程文档

## 一、判断逻辑

### 请求阶段 (onHttpRequestBody)

#### 1. OpenAI 格式检测
- **条件**: `deny_openai = true` 且请求体包含 `messages[0].content` 字段
- **非流式请求** (`stream = false`):
  - 检测到敏感词 → 直接返回拒绝响应（JSON 格式）
  - 未检测到敏感词 → 继续传递请求
- **流式请求** (`stream = true`):
  - 检测到敏感词 → 直接返回拒绝响应（SSE 格式，包含 `data: [DONE]`）
  - 未检测到敏感词 → 继续传递请求

#### 2. JSONPath 格式检测
- **条件**: `deny_jsonpath` 配置项不为空
- 检测到敏感词 → 直接返回拒绝响应（JSON 格式）
- 未检测到敏感词 → 继续传递请求

#### 3. Raw 格式检测
- **条件**: `deny_raw = true`
- 检测到敏感词 → 直接返回拒绝响应（JSON 格式）
- 未检测到敏感词 → 继续传递请求

### 响应阶段 (onHttpResponseBody / onHttpStreamingResponseBody)

#### 1. 非流式响应处理
- **条件**: `Content-Type` 不是 `text/event-stream`
- 检测到敏感词 → 返回拒绝响应（OpenAI JSON 格式）
- 未检测到敏感词 → 原样返回响应

#### 2. 流式响应处理
- **条件**: `Content-Type` 包含 `text/event-stream`
- 使用缓冲区机制进行增量检测
- 支持跨 chunk 敏感词检测
- 检测到敏感词 → 替换所有相关 chunk 为自定义回复

## 二、流程处理逻辑

### 请求阶段流程

```
onHttpRequestHeaders
  ↓
onHttpRequestBody
  ↓
检测到敏感词？
  ├─ 是 → 调用 DenyHandler
  │        ↓
  │      SendHttpResponseWithDetail (发送响应)
  │        ↓
  │      设置 response_sent_in_request = "true"
  │        ↓
  │      返回 ActionPause
  │
  └─ 否 → 继续传递请求到上游
```

### 响应阶段流程（请求阶段未拒绝）

```
onHttpResponseHeaders
  ↓
检查 IsResponseFromUpstream() → true
  ↓
检查 response_sent_in_request → false
  ↓
判断响应类型
  ├─ 流式响应 (SSE)
  │    ↓
  │  设置 is_streaming_response = true
  │    ↓
  │  不缓冲响应体
  │    ↓
  │  onHttpStreamingResponseBody (逐 chunk 处理)
  │
  └─ 非流式响应
       ↓
      缓冲响应体
       ↓
      onHttpResponseBody (一次性处理)
```

### 响应阶段流程（请求阶段已拒绝）

```
onHttpResponseHeaders
  ↓
检查 IsResponseFromUpstream() → false
  ↓
检查 response_sent_in_request → true
  ↓
跳过处理，直接返回 ActionContinue ✅
  ↓
onHttpResponseBody (被调用)
  ↓
检查 IsResponseFromUpstream() → false
  ↓
检查 response_sent_in_request → true
  ↓
跳过处理，直接返回 ActionContinue ✅
```

## 三、流式响应处理详细流程

### 3.1 流式响应检测机制

```
onHttpStreamingResponseBody (每个 chunk)
  ↓
解析 SSE 事件 (按 \n\n 分割)
  ↓
提取 content 和 reasoning 增量
  ↓
添加到累积缓冲区
  ├─ StreamContentBuffer (content 累积)
  └─ StreamReasoningBuffer (reasoning 累积)
  ↓
记录 chunk 信息到 StreamChunkBuffer
  ├─ Data: 原始 chunk 数据
  ├─ ContentStart/End: 在 StreamContentBuffer 中的位置
  └─ ReasoningStart/End: 在 StreamReasoningBuffer 中的位置
  ↓
缓冲区满或流结束？
  ├─ 是 → 进行敏感词检测
  └─ 否 → 返回 nil，等待更多数据
```

### 3.2 跨 Chunk 敏感词检测

#### 检测算法
1. **累积缓冲区检测**: 使用 `FindSensitiveWordMatches` 在累积缓冲区中查找所有敏感词匹配位置
2. **位置映射**: 根据敏感词在缓冲区中的位置（StartPos, EndPos），找到所有涉及的 chunk
3. **Chunk 标记**: 标记所有与敏感词位置重叠的 chunk

#### 重叠判断逻辑
```go
// 如果敏感词的任何部分在 chunk 范围内，就标记这个 chunk
if (match.StartPos >= chunkStart && match.StartPos < chunkEnd) ||
   (match.EndPos > chunkStart && match.EndPos <= chunkEnd) ||
   (match.StartPos <= chunkStart && match.EndPos >= chunkEnd) {
    deniedChunkIndices[i] = true
}
```

#### 示例场景
假设敏感词 "敏感词" 跨越两个 chunk:
- **Chunk 1**: "这是敏" (ContentStart: 0, ContentEnd: 6)
- **Chunk 2**: "感词内容" (ContentStart: 6, ContentEnd: 18)

检测结果:
- 敏感词位置: StartPos=3, EndPos=9
- Chunk 1 重叠: 3 >= 0 && 3 < 6 → **标记**
- Chunk 2 重叠: 9 > 6 && 9 <= 18 → **标记**

### 3.3 流式响应替换逻辑

```
检测到敏感词
  ↓
标记所有涉及的 chunk (deniedChunkIndices)
  ↓
构建返回结果
  ├─ 第一个被标记的 chunk → 替换为自定义回复
  ├─ 其他被标记的 chunk → 跳过（不输出）
  ├─ 未被标记的 chunk → 原样输出
  └─ [DONE] chunk → 跳过，最后统一添加
  ↓
如果流已结束，添加 data: [DONE]\n\n
  ↓
返回处理后的 chunk
```

### 3.4 缓冲区管理

#### 滑动窗口机制
- **默认缓冲区大小**: 10KB (可通过 `stream_buffer` 配置)
- **溢出处理**: 当缓冲区超过限制时，保留最新的 `bufferSize` 字节
- **位置调整**: 同步调整所有 chunk 的位置信息，确保位置映射正确

#### 缓冲区触发条件
- 缓冲区满 (`StreamChunkBufferSize >= bufferSize`)
- 流结束 (`data: [DONE]` 标记)

## 四、敏感词检测机制

### 4.1 检测算法
- **算法**: Aho-Corasick 自动机
- **优势**: 一次遍历可同时匹配多个敏感词
- **缓存机制**: 匹配器构建后缓存，避免重复构建

### 4.2 检测范围
- **自定义敏感词**: `deny_words` 配置项
- **系统敏感词**: `system_deny = true` 时启用

### 4.3 检测模式
- **非流式**: 直接匹配完整文本
- **流式**: 使用累积缓冲区进行增量匹配

## 五、状态标志管理

### 5.1 请求阶段标志
- `response_sent_in_request`: 标记响应是否已在请求阶段发送
- `x-ai-data-masking`: 拒绝类型（OpenAI/JSONPath/Raw）
- `deny_step`: 拒绝发生的阶段
- `deny_code`: HTTP 状态码

### 5.2 流式响应标志
- `StreamDenied`: 标记流式响应是否已拒绝（后续 chunk 直接跳过）
- `RespIsSSE`: 标记响应是否为 SSE 格式
- `is_streaming_response`: 标记是否为流式响应

### 5.3 上下文状态
- `IsDeny`: 是否被拒绝
- `IsRequestDeny`: 是否在请求阶段拒绝
- `IsResponseDeny`: 是否在响应阶段拒绝
- `IsModified`: 是否被修改
- `IsRequestModified`: 是否在请求阶段修改
- `IsResponseModified`: 是否在响应阶段修改

## 六、特殊处理

### 6.1 [DONE] 标记处理
- **问题**: 避免重复输出 `[DONE]`
- **解决方案**: 
  - 在输出循环中跳过 `IsDone = true` 的 chunk
  - 最后统一添加一个 `data: [DONE]\n\n`
- **结果**: 无论是否有敏感词，都只输出一个 `[DONE]`

### 6.2 响应来源检查
- **IsResponseFromUpstream()**: 检查响应是否来自上游
- **用途**: 区分请求阶段发送的响应和上游返回的响应

### 6.3 响应头处理
- **HeaderStopIteration**: 停止响应头的迭代处理
- **用途**: 防止响应头被多次处理

## 七、性能优化

### 7.1 匹配器缓存
- 自定义敏感词匹配器缓存
- 系统敏感词匹配器缓存
- 使用读写锁保证并发安全

### 7.2 缓冲区管理
- 滑动窗口机制，限制内存使用
- 及时清理已处理的 chunk

### 7.3 早期退出
- 流式响应一旦检测到敏感词，后续 chunk 直接跳过
- 请求阶段检测到敏感词，立即返回，不继续处理

## 八、错误处理

### 8.1 JSON 解析错误
- 解析失败时跳过该事件，继续处理下一个

### 8.2 缓冲区溢出
- 使用滑动窗口机制，自动清理旧数据

### 8.3 流中断
- 通过 `isLastChunk` 参数判断流是否结束
- 流结束时统一处理缓冲区中的剩余数据
