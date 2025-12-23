package config

// OpenAI 结构体
// OpenAI 非流式响应结构体
type OpenAICompletionResponse struct {
	Id                string                   `json:"id,omitempty"`
	Object            string                   `json:"object,omitempty"`
	Created           int64                    `json:"created,omitempty"`
	Model             string                   `json:"model,omitempty"`
	SystemFingerprint string                   `json:"system_fingerprint,omitempty"`
	Choices           []OpenAICompletionChoice `json:"choices"`
	Usage             *OpenAIUsage             `json:"usage,omitempty"`
}

// OpenAI 流式响应结构体（SSE 格式中的单个事件）
type OpenAIStreamCompletionResponse struct {
	Id                string               `json:"id,omitempty"`
	Object            string               `json:"object,omitempty"`
	Created           int64                `json:"created,omitempty"`
	Model             string               `json:"model,omitempty"`
	SystemFingerprint string               `json:"system_fingerprint,omitempty"`
	Choices           []OpenAIStreamChoice `json:"choices"`
	Usage             *OpenAIUsage         `json:"usage,omitempty"`
}

// OpenAI 非流式响应中的 Choice
type OpenAICompletionChoice struct {
	Index        int                    `json:"index"`
	Message      *OpenAIMessage         `json:"message,omitempty"`
	FinishReason string                 `json:"finish_reason,omitempty"`
	Logprobs     map[string]interface{} `json:"logprobs,omitempty"`
}

// OpenAI 流式响应中的 Choice
type OpenAIStreamChoice struct {
	Index        int                    `json:"index"`
	Delta        *OpenAIMessage         `json:"delta,omitempty"`
	FinishReason string                 `json:"finish_reason,omitempty"`
	Logprobs     map[string]interface{} `json:"logprobs,omitempty"`
}

// OpenAI Message 结构体（用于非流式的 message 和流式的 delta）
type OpenAIMessage struct {
	Role             string                 `json:"role,omitempty"`
	Content          string                 `json:"content,omitempty"`
	Reasoning        string                 `json:"reasoning,omitempty"`         // 流式响应中的 reasoning
	ReasoningContent string                 `json:"reasoning_content,omitempty"` // 非流式响应中的 reasoning_content
	Name             string                 `json:"name,omitempty"`
	ToolCalls        []interface{}          `json:"tool_calls,omitempty"`
	FunctionCall     map[string]interface{} `json:"function_call,omitempty"`
	Refusal          string                 `json:"refusal,omitempty"`
}

// OpenAI Usage 结构体
type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

// JSONPath 返回拒绝响应结构体
type JSONPathResponse struct {
	Code    uint32                 `json:"code"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}

// Raw 返回拒绝响应结构体
type RawResponse struct {
	Code    uint32                 `json:"code"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}
