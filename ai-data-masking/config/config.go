package config

import (
	"regexp"
)

const (
	DEFAULT_MAX_BODY_BYTES uint32 = 100 * 1024 * 1024
)

const (
	DEFAULT_SCHEMA                    = "defaultSchema"
	HTTP_STATUS_OK                    = uint32(200)
	HTTP_STATUS_INTERNAL_SERVER_ERROR = uint32(500)
)
const (
	FINISH_REASON_STOP = "stop"
)

// AiDataMaskingConfig 插件配置
type AiDataMaskingConfig struct {
	DenyOpenAI      bool     `json:"deny_openai"`
	DenyRaw         bool     `json:"deny_raw"`
	DenyJSONPath    []string `json:"deny_jsonpath"`
	SystemDeny      bool     `json:"system_deny"`
	DenyCode        uint32   `json:"deny_code"`
	DenyMessage     string   `json:"deny_message"`
	DenyRawMessage  string   `json:"deny_raw_message"`
	DenyContentType string   `json:"deny_content_type"`
	DenyWords       []string `json:"deny_words"`
	ReplaceRoles    []Rule   `json:"replace_roles"`
	StreamBuffer    uint32   `json:"stream_buffer"`
}

// Rule 替换规则
type Rule struct {
	Regex   string `json:"regex"`
	Type    string `json:"type"` // "replace" or "hash"
	Restore bool   `json:"restore"`
	Value   string `json:"value"`
	// 编译后的正则表达式
	CompiledRegex *regexp.Regexp
}

// PluginContext 插件上下文（与配置解耦）
type PluginContext struct {
	Config                 *AiDataMaskingConfig
	MaskMap                map[string]*string // hash值 -> 原始值，用于还原
	OpenAIRequest          *OpenAIRequest     // OpenAI请求参数
	RequestDenyModifyType  DenyModifyType     // 请求拒绝类型
	ResponseDenyModifyType DenyModifyType     // 响应拒绝类型
	RespIsSSE              bool               // 响应是否是SSE,返回头阶段判断，如果是sse，则分块处理
	// deny
	IsRequestDeny  bool // 是否是请求阶段拒绝
	IsResponseDeny bool // 是否是响应阶段拒绝
	IsDeny         bool // 是否被拒绝
	// modify
	IsRequestModified  bool // 是否是请求阶段修改
	IsResponseModified bool // 是否是响应阶段修改
	IsModified         bool // 是否被修改
	Step               Step // 处理步骤
	// 流式响应缓冲区（滑动窗口）
	StreamContentBuffer   string // content 缓冲区（用于敏感词检查）
	StreamReasoningBuffer string // reasoning 缓冲区（用于敏感词检查）
	StreamDenied          bool   // 是否已拒绝（用于标记后续不再处理）
	// 流式响应 chunk 缓冲区
	StreamChunkBuffer     []StreamChunk // 存储所有 chunk，等待缓冲区满或 [DONE] 时处理
	StreamChunkBufferSize int           // 当前缓冲区大小（字节数）
}

// StreamChunk 流式响应 chunk 结构
type StreamChunk struct {
	Data           []byte // chunk 的原始数据
	ContentStart   int    // 在 StreamContentBuffer 中的起始位置
	ContentEnd     int    // 在 StreamContentBuffer 中的结束位置
	ReasoningStart int    // 在 StreamReasoningBuffer 中的起始位置
	ReasoningEnd   int    // 在 StreamReasoningBuffer 中的结束位置
	IsDone         bool   // 是否是 [DONE] 标记
}

type DenyModifyType string

const (
	DenyModifyTypeOpenAI   DenyModifyType = "OpenAI"
	DenyModifyTypeJSONPath DenyModifyType = "JSONPath"
	DenyModifyTypeRaw      DenyModifyType = "Raw"
)

type OpenAIRequest struct {
	Model    string
	Stream   bool
	Messages []OpenAIMessage
}

// Step 处理步骤枚举
type Step string

const (
	StepRequestHeader  Step = "request_header"   // 请求头处理阶段
	StepRequestBody    Step = "request_body"     // 请求体处理阶段
	StepRespHeader     Step = "resp_header"      // 响应头处理阶段
	StepRespBody       Step = "resp_body"        // 响应体处理阶段
	StepStreamRespBody Step = "stream_resp_body" // 流式响应体处理阶段
)

// String 返回枚举的字符串值
func (s Step) String() string {
	return string(s)
}

// IsValid 检查 Step 是否为有效值
func (s Step) IsValid() bool {
	return s == StepRequestHeader || s == StepRequestBody || s == StepRespHeader ||
		s == StepRespBody || s == StepStreamRespBody
}

var (
	// 系统敏感词库（简化版，实际应该从文件加载）
	SystemDenyWords = []string{
		// 这里可以添加系统敏感词
		// 实际实现应该从资源文件加载
	}
)
