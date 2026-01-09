package config

import (
	"bufio"
	"io/fs"
	"strings"
)

const (
	DEFAULT_MAX_BODY_BYTES         uint32 = 100 * 1024 * 1024
	DefaultMaxBufferChunkCount     uint32 = 30
	DefaultMaxStreamChunkBufferLen uint32 = 2048
)

const (
	DEFAULT_SCHEMA                    = "defaultSchema"
	HTTP_STATUS_OK                    = uint32(200)
	HTTP_STATUS_INTERNAL_SERVER_ERROR = uint32(500)
)
const (
	FINISH_REASON_STOP = "stop"
)

var MaxSensitiveWordLength int = 0 // 最长敏感词长度（字节数），在配置解析时计算

const (
	MAX_SYSTEM_DENY_WORDS_BATCH_SIZE = 200
)

// AiDataMaskingConfig 插件配置
type AiDataMaskingConfig struct {
	SystemDeny              bool        `json:"system_deny"`                 // 开启内置拦截规则
	DenyCode                uint32      `json:"deny_code"`                   // 拦截时http状态码
	DenyContentType         string      `json:"deny_content_type"`           // 拦截时返回content_type头
	DenyMessage             string      `json:"deny_message"`                // 拦截时ai返回消息
	DenyWords               []string    `json:"deny_words"`                  // 自定义敏感词列表
	DenyPlot                DenyPlot    `json:"deny_plot"`                   // 拒绝处理方式
	RequestDeny             bool        `json:"request_deny"`                // 是否开启请求拦截
	ResponseDeny            bool        `json:"response_deny"`               // 是否开启响应拦截
	MatchFormat             MatchFormat `json:"match_format"`                // 匹配格式配置
	MaxBufferChunkCount     uint32      `json:"max_buffer_chunk_count"`      // 最长敏感词检测chunk个数
	MaxStreamChunkBufferLen uint32      `json:"max_stream_chunk_buffer_len"` // 最长敏感词检测chunk大小
	DenyPunctuation         []string    `json:"deny_punctuation"`            // 敏感词检测忽略的标点符号
}

// DenyPlot 拒绝处理方式
type DenyPlot struct {
	Plot  string `json:"plot"`  // replace, stop, rollback 默认stop
	Value string `json:"value"` // 如果是 replace，则替换为value
}

// MatchFormat 匹配格式配置
type MatchFormat struct {
	Type                 string   `json:"type"`                   // 格式类型：custom, openai, anthropic
	RequestDenyJSONPath  []string `json:"request_deny_jsonpath"`  // 请求拦截的 JSONPath（仅 type=custom 时有效）
	ResponseDenyJSONPath []string `json:"response_deny_jsonpath"` // 响应拦截的 JSONPath（仅 type=custom 时有效）
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
	IsRequestModified  bool // 是否是请求阶段修改，暂时保留，请求阶段不做替换处理
	IsResponseModified bool // 是否是响应阶段修改
	IsModified         bool // 是否拒绝敏感词后被修改
	Step               Step // 处理步骤

	MaxBufferChunkCount     uint32 // 最长敏感词检测chunk个数
	MaxStreamChunkBufferLen uint32 // 最长敏感词检测chunk大小
	// 流式响应缓冲区（滑动窗口）
	StreamContentBuffer         string       // content 缓冲区（用于敏感词检查）
	StreamReasoningBuffer       string       // reasoning 缓冲区（用于敏感词检查）
	StreamContentBufferOffset   int          // content 缓冲区的偏移量（用于处理跨窗口边界）
	StreamReasoningBufferOffset int          // reasoning 缓冲区的偏移量（用于处理跨窗口边界）
	StreamDenied                bool         // 是否已拒绝（用于标记后续不再处理）
	StreamSeq                   int          // SSE 流序号，用于标识每个数据块
	StreamRollbackSentSeqs      map[int]bool // 已发送rollback的seq集合（防止重复回退相同的chunk）
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
	Seq            int    // SSE 流序号，用于标识每个数据块（rollback 策略使用）
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
	// 系统敏感词库（从资源文件加载）
	SystemDenyWords = []string{}
)

// LoadSystemDenyWords 从资源文件加载系统敏感词
func LoadSystemDenyWords(resourcesFS fs.FS) ([]string, error) {
	var words []string
	wordMap := make(map[string]bool) // 用于去重

	// 加载中文敏感词文件
	chineseFile, err := resourcesFS.Open("resources/sensitive_word_dict.txt")
	if err == nil {
		defer chineseFile.Close()
		scanner := bufio.NewScanner(chineseFile)
		for scanner.Scan() {
			word := strings.TrimSpace(scanner.Text())
			if word != "" && !wordMap[word] {
				words = append(words, word)
				wordMap[word] = true
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}

	// 加载英文敏感词文件
	englishFile, err := resourcesFS.Open("resources/sensitive_word_dict_en.txt")
	if err == nil {
		defer englishFile.Close()
		scanner := bufio.NewScanner(englishFile)
		for scanner.Scan() {
			word := strings.TrimSpace(scanner.Text())
			if word != "" && !wordMap[word] {
				words = append(words, word)
				wordMap[word] = true
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}

	// 加载合并的敏感词文件
	mergedFile, err := resourcesFS.Open("resources/sensitive_word_dict_merged.txt")
	if err == nil {
		defer mergedFile.Close()
		scanner := bufio.NewScanner(mergedFile)
		for scanner.Scan() {
			word := strings.TrimSpace(scanner.Text())
			if word != "" && !wordMap[word] {
				words = append(words, word)
				wordMap[word] = true
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}

	return words, nil
}
