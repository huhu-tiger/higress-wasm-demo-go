// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package t2i

// UnifiedRequest 统一请求格式（文生图）
type UnifiedRequest struct {
	Model      string                 `json:"model"`
	Input      UnifiedInput           `json:"input"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	ExtraBody  map[string]interface{} `json:"extra_body,omitempty"`
}

// UnifiedInput 统一输入格式
type UnifiedInput struct {
	Messages []UnifiedMessage `json:"messages"`
}

// UnifiedMessage 统一消息格式
type UnifiedMessage struct {
	Role    string           `json:"role"`
	Content []UnifiedContent `json:"content"`
}

// UnifiedContent 统一内容格式
type UnifiedContent struct {
	Text  string `json:"text,omitempty"`
	Image string `json:"image,omitempty"`
}

// UnifiedResponse 统一响应格式（文生图）
type UnifiedResponse struct {
	RequestID  string        `json:"request_id"`
	Output     UnifiedOutput `json:"output"`
	Usage      UnifiedUsage  `json:"usage"`
	Code       string        `json:"code,omitempty"`
	Message    string        `json:"message,omitempty"`
	StatusCode int           `json:"status_code,omitempty"`
}

// UnifiedOutput 统一输出格式
type UnifiedOutput struct {
	Choices    []UnifiedChoice   `json:"choices"`
	TaskMetric UnifiedTaskMetric `json:"task_metric"`
}

// UnifiedChoice 统一选择格式
type UnifiedChoice struct {
	FinishReason string         `json:"finish_reason"`
	Message      UnifiedMessage `json:"message"`
	Index        int            `json:"index,omitempty"`
}

// UnifiedTaskMetric 统一任务指标格式
type UnifiedTaskMetric struct {
	Failed    int `json:"FAILED"`
	Succeeded int `json:"SUCCEEDED"`
	Total     int `json:"TOTAL"`
}

// UnifiedUsage 统一使用情况格式
type UnifiedUsage struct {
	ImageCount       int `json:"image_count"`
	Height           int `json:"height,omitempty"`
	Width            int `json:"width,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
}

// CheckStreamParameter 检查请求体中是否有 stream 参数（通用方法）
// 根据不同的 model 和 modelType 调用不同的检查方法
func CheckStreamParameter(model, modelType string, body []byte) bool {
	if modelType == "t2i" {
		switch model {
		case "volcengine/volcengine/doubao-seedream-4.5":
			return checkDoubaoStreamParameter(body)
		case "aliyun/aliyun/qwen-image-plus":
			// qwen 模型目前不支持 stream 参数
			return false
		case "openrouter/auto/gemini-3-image", "google/gemini-3-pro-image-preview":
			// gemini 模型目前不支持 stream 参数
			return false
		default:
			// 其他模型默认不支持 stream 参数
			return false
		}
	}
	// 其他 modelType 默认不支持 stream 参数
	return false
}

// TransformRequest 转换请求体（通用方法）
// 根据不同的 model 和 modelType 调用不同的转换方法
func TransformRequest(model, modelType string, body []byte) ([]byte, error) {
	if modelType == "t2i" {
		switch model {
		case "volcengine/volcengine/doubao-seedream-4.5":
			return transformDoubaoRequest(body)
		case "aliyun/aliyun/qwen-image-plus":
			return transformQwenRequest(body)
		case "openrouter/auto/gemini-3-image", "google/gemini-3-pro-image-preview":
			return transformGeminiRequest(body)
		default:
			// 其他模型默认不需要转换
			return nil, nil
		}
	}
	// 其他 modelType 默认不需要转换
	return nil, nil
}

// TransformResponse 转换响应体（通用方法）
// 根据不同的 model 和 modelType 调用不同的转换方法
func TransformResponse(model, modelType string, body []byte) ([]byte, error) {
	if modelType == "t2i" {
		switch model {
		case "volcengine/volcengine/doubao-seedream-4.5":
			return transformDoubaoResponse(body)
		case "aliyun/aliyun/qwen-image-plus":
			return transformQwenResponse(body)
		case "openrouter/auto/gemini-3-image":
			return transformGeminiResponse(body)
		default:
			// 其他模型默认不需要转换
			return nil, nil
		}
	}
	// 其他 modelType 默认不需要转换
	return nil, nil
}

// IsStreamingResponse 判断是否为流式响应（通用方法）
// 根据不同的 model 和 modelType 调用不同的判断方法
// isChunked: 响应 header 的 transfer-encoding 是否为 chunked
// hasStreamParam: 请求参数中是否有 stream 参数
func IsStreamingResponse(model, modelType string, isChunked, hasStreamParam bool) bool {
	if modelType == "t2i" {
		switch model {
		case "volcengine/volcengine/doubao-seedream-4.5":
			return isDoubaoStreamingResponse(isChunked, hasStreamParam)
		case "aliyun/aliyun/qwen-image-plus":
			return isQwenStreamingResponse(isChunked, hasStreamParam)
		case "openrouter/auto/gemini-3-image":
			return isGeminiStreamingResponse(isChunked, hasStreamParam)
		default:
			// 其他模型默认只判断 transfer-encoding
			return isChunked
		}
	}
	// 其他 modelType 默认只判断 transfer-encoding
	return isChunked
}

// ProcessResponseHeaders 处理响应头（通用方法）
// 根据不同的 model 和 modelType 调用不同的处理方法
// 在 onHttpResponseHeaders 阶段调用，用于删除或修改响应头
// transferEncoding: 响应 header 的 transfer-encoding 值
// hasStreamRequest: 请求参数中是否有 stream 参数
func ProcessResponseHeaders(model, modelType, transferEncoding string, hasStreamRequest bool) {
	if modelType == "t2i" {
		switch model {
		case "volcengine/volcengine/doubao-seedream-4.5":
			processDoubaoResponseHeaders(transferEncoding, hasStreamRequest)
		case "aliyun/aliyun/qwen-image-plus":
			processQwenResponseHeaders(transferEncoding, hasStreamRequest)
		case "openrouter/auto/gemini-3-image":
			processGeminiResponseHeaders(transferEncoding, hasStreamRequest)
		default:
			// 其他模型默认处理逻辑：如果不是流式请求，删除 transfer-encoding
			processDefaultResponseHeaders(transferEncoding, hasStreamRequest)
		}
	} else {
		// 其他 modelType 默认处理逻辑
		processDefaultResponseHeaders(transferEncoding, hasStreamRequest)
	}
}

// processDefaultResponseHeaders 默认响应头处理逻辑
// 如果不是流式请求，删除 transfer-encoding header
func processDefaultResponseHeaders(transferEncoding string, hasStreamRequest bool) {
	// 默认逻辑由各模型文件实现，这里只是占位
}
