# AI 数据脱敏插件 Golang 实现说明

## 实现概述

已使用 Golang 重新实现了 ai-data-masking 插件，功能与 Rust 版本基本一致。

## 已实现功能

### ✅ 核心功能

1. **配置解析**
   - 支持所有配置字段
   - 默认值处理
   - GROK 模式转换（简化版）

2. **敏感词检测**
   - 自定义敏感词列表
   - 系统敏感词库（基础实现）
   - 字符串包含匹配

3. **OpenAI 协议支持**
   - 请求拦截和替换
   - 响应拦截和还原
   - 流式和非流式格式

4. **JSONPath 支持**
   - 使用 gjson 库解析 JSONPath
   - 支持数组和嵌套字段

5. **Raw Body 支持**
   - 原始请求/响应 body 检查
   - 敏感词替换

6. **流式响应支持**
   - SSE 格式检测
   - 流式响应中的敏感词检查
   - 错误消息发送

7. **敏感词替换**
   - Replace 类型替换
   - Hash 类型替换（SHA256）
   - 数据还原功能

## 与 Rust 版本的差异

### 1. 敏感词检测

**Rust 版本**：
- 使用 jieba 分词库
- 基于分词的精确匹配

**Golang 版本**：
- 当前使用简单的字符串包含匹配
- 可以集成 gojieba 库改进

### 2. GROK 支持

**Rust 版本**：
- 完整的 GROK 模式解析
- 支持嵌套 GROK 模式

**Golang 版本**：
- 简化实现，支持常见模式
- 可以集成 grok 库完善

### 3. 系统敏感词库

**Rust 版本**：
- 从资源文件加载
- 使用 rust-embed

**Golang 版本**：
- 当前是硬编码的示例
- 需要从资源文件加载（可以使用 embed）

### 4. 流式响应处理

**Rust 版本**：
- 完整的 SSE 事件解析
- 消息窗口管理
- 增量内容处理

**Golang 版本**：
- 基础实现
- 需要完善 SSE 事件解析

## 代码结构

```
ai-data-masking/
├── main.go          # 主实现文件
├── go.mod           # Go 模块定义
├── README.md        # 使用文档
├── VERSION          # 版本号
└── IMPLEMENTATION_NOTES.md  # 实现说明（本文件）
```

## 主要函数

### 配置和初始化
- `parseConfig()` - 解析插件配置
- `convertGrokToRegex()` - GROK 模式转换
- `getOrCreatePluginContext()` - 获取插件上下文

### 请求处理
- `onHttpRequestHeaders()` - 请求头处理
- `onHttpRequestBody()` - 请求体处理
- `processOpenAIRequest()` - OpenAI 格式请求处理
- `processJSONPathRequest()` - JSONPath 请求处理
- `processRawRequest()` - 原始请求处理

### 响应处理
- `onHttpResponseHeaders()` - 响应头处理
- `onHttpResponseBody()` - 非流式响应处理
- `onHttpStreamingResponseBody()` - 流式响应处理
- `processNonStreamResponse()` - 非流式响应处理
- `processStreamResponse()` - 流式响应处理

### 核心功能
- `checkMessage()` - 敏感词检测
- `deny()` - 拒绝请求/响应
- `replaceMessage()` - 敏感词替换
- `restoreMessage()` - 数据还原
- `msgToResponse()` - 消息格式转换

## 改进建议

### 1. 集成分词库

```go
import "github.com/yanyiwu/gojieba"

func checkMessageWithJieba(message string, config *AiDataMaskingConfig, log log.Log) bool {
    jieba := gojieba.NewJieba()
    defer jieba.Free()
    
    words := jieba.Cut(message, true)
    for _, word := range words {
        // 检查敏感词
    }
}
```

### 2. 完善 GROK 支持

可以使用现有的 GROK 库：
```go
import "github.com/vjeantet/grok"
```

### 3. 从资源文件加载敏感词

使用 Go 1.16+ 的 embed 功能：
```go
//go:embed res/sensitive_word_dict.txt
var sensitiveWordDict string
```

### 4. 完善流式响应处理

实现完整的 SSE 事件解析，参考 Rust 版本的 `msg_win_openai.rs`。

## 测试

### 构建插件

```bash
cd /data/work/higress/plugins/wasm-go
PLUGIN_NAME=ai-data-masking make build
```

### 测试配置

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
    deny_openai: true
    deny_words:
      - "政治"
    deny_code: 403
  url: oci://your-registry/ai-data-masking:1.0.0
  phase: AUTHN
  priority: 991
```

## 已知限制

1. **敏感词检测**：当前使用简单字符串匹配，可能误匹配
2. **GROK 支持**：只支持常见模式，复杂模式可能不工作
3. **流式响应**：SSE 解析需要进一步完善
4. **系统敏感词库**：需要从资源文件加载

## 后续工作

- [ ] 集成 gojieba 进行分词
- [ ] 完善 GROK 模式支持
- [ ] 从资源文件加载系统敏感词库
- [ ] 完善流式响应 SSE 解析
- [ ] 添加单元测试
- [ ] 性能优化

## 参考

- Rust 版本实现：`plugins/wasm-rust/extensions/ai-data-masking/`
- Golang 插件框架：`plugins/wasm-go/pkg/wrapper/`
- 其他 Golang 插件示例：`plugins/wasm-go/extensions/`

