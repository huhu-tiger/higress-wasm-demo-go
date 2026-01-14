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

import (
	"encoding/json"
	"strconv"
	"transformer-vnet/config"
	"transformer-vnet/wlog"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
)

// GeminiRequest gemini-3-image 实际请求格式
type GeminiRequest struct {
	Model      string          `json:"model"`
	Messages   []GeminiMessage `json:"messages"`
	Modalities []string        `json:"modalities"`
	Provider   *GeminiProvider `json:"provider,omitempty"`
}

// GeminiProvider gemini 提供商过滤格式
type GeminiProvider struct {
	Only []string `json:"only,omitempty"`
}

// GeminiMessage gemini 消息格式
type GeminiMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // 可以是字符串或数组
}

// GeminiResponse gemini-3-image 实际响应格式
type GeminiResponse struct {
	ID       string         `json:"id"`
	Provider string         `json:"provider,omitempty"`
	Model    string         `json:"model"`
	Object   string         `json:"object,omitempty"`
	Created  int64          `json:"created"`
	Choices  []GeminiChoice `json:"choices"`
	Usage    *GeminiUsage   `json:"usage,omitempty"`
	Error    *GeminiError   `json:"error,omitempty"`
}

// GeminiChoice gemini 选择格式
type GeminiChoice struct {
	Logprobs           interface{}       `json:"logprobs,omitempty"`
	FinishReason       string            `json:"finish_reason"`
	NativeFinishReason string            `json:"native_finish_reason,omitempty"`
	Index              int               `json:"index"`
	Message            GeminiMessageResp `json:"message"`
}

// GeminiMessageResp gemini 消息响应格式
type GeminiMessageResp struct {
	Role             string        `json:"role"`
	Content          string        `json:"content"`
	Refusal          interface{}   `json:"refusal,omitempty"`
	Reasoning        string        `json:"reasoning,omitempty"`
	ReasoningDetails []interface{} `json:"reasoning_details,omitempty"`
	Annotations      []interface{} `json:"annotations,omitempty"`
	Images           []GeminiImage `json:"images,omitempty"`
}

// GeminiImage gemini 图片格式
type GeminiImage struct {
	Type     string         `json:"type"`
	ImageURL GeminiImageURL `json:"image_url"`
	Index    int            `json:"index,omitempty"`
}

// GeminiImageURL gemini 图片 URL 格式
type GeminiImageURL struct {
	URL string `json:"url"`
}

// GeminiUsage gemini 使用情况格式
type GeminiUsage struct {
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	Cost             float64 `json:"cost,omitempty"`
	IsByok           bool    `json:"is_byok,omitempty"`
}

// GeminiError gemini 错误格式
type GeminiError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param,omitempty"`
	Code    string `json:"code,omitempty"`
}

// ToGeminiRequest 将统一请求格式转换为 gemini-3-image 请求格式
func ToGeminiRequest(unified *UnifiedRequest) (*GeminiRequest, error) {
	if unified == nil {
		return nil, nil
	}

	gemini := &GeminiRequest{
		Model: unified.Model,
	}

	// 1. 构建 messages 结构：从 input.messages[0] 提取 role 和 content[0].text
	if len(unified.Input.Messages) > 0 {
		unifiedMsg := unified.Input.Messages[0]
		geminiMsg := GeminiMessage{
			Role: unifiedMsg.Role,
		}

		// 提取 content[0].text 作为字符串
		if len(unifiedMsg.Content) > 0 {
			geminiMsg.Content = unifiedMsg.Content[0].Text
		} else {
			geminiMsg.Content = ""
		}

		gemini.Messages = []GeminiMessage{geminiMsg}
	}

	// 2. 提取 extra_body.gemini_3_image.modalities 到顶层 modalities
	// 3. 提取 extra_body.gemini_3_image.provider 到顶层 provider
	if unified.ExtraBody != nil {
		if gemini3Image, ok := unified.ExtraBody["gemini_3_image"].(map[string]interface{}); ok {
			// 提取 modalities
			if modalities, ok := gemini3Image["modalities"].([]interface{}); ok {
				gemini.Modalities = make([]string, 0, len(modalities))
				for _, m := range modalities {
					if str, ok := m.(string); ok {
						gemini.Modalities = append(gemini.Modalities, str)
					}
				}
			}

			// 提取 provider
			if providerData, ok := gemini3Image["provider"].(map[string]interface{}); ok {
				gemini.Provider = &GeminiProvider{}
				if only, ok := providerData["only"].([]interface{}); ok {
					gemini.Provider.Only = make([]string, 0, len(only))
					for _, o := range only {
						if str, ok := o.(string); ok {
							gemini.Provider.Only = append(gemini.Provider.Only, str)
						}
					}
				}
			}
		}
	}

	// 如果没有设置 modalities，使用默认值 ["image", "text"]
	if len(gemini.Modalities) == 0 {
		gemini.Modalities = []string{"image", "text"}
	}

	wlog.LogWithLine("[%s] Converted unified request to gemini-3-image format", config.PluginName)
	return gemini, nil
}

// ToGeminiRequestFromJSON 从 JSON 字节数组转换为 gemini-3-image 请求格式
func ToGeminiRequestFromJSON(body []byte) (*GeminiRequest, error) {
	// 先解析为统一格式
	var unified UnifiedRequest
	if err := json.Unmarshal(body, &unified); err != nil {
		return nil, err
	}

	return ToGeminiRequest(&unified)
}

// ToUnifiedResponseFromGemini 将 gemini-3-image 响应格式转换为统一响应格式
func ToUnifiedResponseFromGemini(gemini *GeminiResponse) (*UnifiedResponse, error) {
	if gemini == nil {
		return nil, nil
	}

	// 如果有错误，直接返回错误响应
	if gemini.Error != nil {
		return &UnifiedResponse{
			RequestID:  gemini.ID,
			Code:       gemini.Error.Code,
			Message:    gemini.Error.Message,
			StatusCode: 400,
		}, nil
	}

	// 1. 构建 content 数组：从 choices[0].message.images 提取图片
	contentArray := make([]UnifiedContent, 0)
	imageCount := 0

	if len(gemini.Choices) > 0 {
		choice := gemini.Choices[0]
		if len(choice.Message.Images) > 0 {
			for _, img := range choice.Message.Images {
				if img.ImageURL.URL != "" {
					contentArray = append(contentArray, UnifiedContent{
						Image: img.ImageURL.URL,
					})
					imageCount++
					// wlog.LogWithLine("[%s] Added image URL to content array: %s", "config.PluginName", img.ImageURL.URL)
				}
			}
		}
	}

	// 如果没有 content，记录警告
	if len(contentArray) == 0 {
		wlog.LogWithLine("[%s] Warning: No images found in gemini response", config.PluginName)
	}

	// 2. 提取 id 作为 request_id
	requestID := gemini.ID
	if requestID == "" {
		// 如果没有 id，使用 created 时间戳
		if gemini.Created > 0 {
			requestID = strconv.FormatInt(gemini.Created, 10)
		}
	}

	// 3. 提取 usage 信息
	imageCountFromUsage := imageCount
	if gemini.Usage != nil && gemini.Usage.CompletionTokens > 0 {
		// 如果有 usage 信息，优先使用它
		// 注意：gemini 的 usage 可能不包含 image_count，所以使用实际提取的图片数量
	}

	// 4. 构建统一格式的响应
	unified := &UnifiedResponse{
		RequestID: requestID,
		Output: UnifiedOutput{
			Choices: []UnifiedChoice{
				{
					FinishReason: "stop",
					Message: UnifiedMessage{
						Role:    "assistant",
						Content: contentArray,
					},
				},
			},
			TaskMetric: UnifiedTaskMetric{
				Failed:    0,
				Succeeded: imageCountFromUsage,
				Total:     imageCountFromUsage,
			},
		},
		Usage: UnifiedUsage{
			ImageCount: imageCountFromUsage,
		},
	}

	// 如果有 usage 信息，提取 token 统计
	if gemini.Usage != nil {
		unified.Usage.TotalTokens = gemini.Usage.TotalTokens
		unified.Usage.PromptTokens = gemini.Usage.PromptTokens
		unified.Usage.CompletionTokens = gemini.Usage.CompletionTokens
	}

	wlog.LogWithLine("[%s] Converted gemini-3-image response to unified format", config.PluginName)
	return unified, nil
}

// ToUnifiedResponseFromGeminiJSON 从 JSON 字节数组转换为统一响应格式
func ToUnifiedResponseFromGeminiJSON(body []byte) (*UnifiedResponse, error) {
	// 先解析为 gemini 格式
	var gemini GeminiResponse
	if err := json.Unmarshal(body, &gemini); err != nil {
		wlog.LogWithLine("[%s] Failed to unmarshal gemini response: %v", config.PluginName, err)
		return nil, err
	}

	wlog.LogWithLine("[%s] Successfully parsed gemini response, ID: %s, Model: %s, Choices count: %d",
		config.PluginName, gemini.ID, gemini.Model, len(gemini.Choices))

	// 转换为统一格式
	return ToUnifiedResponseFromGemini(&gemini)
}

// transformGeminiRequest 将统一格式转换为 gemini-3-image 格式
func transformGeminiRequest(body []byte) ([]byte, error) {
	// 使用新的转换方法
	geminiReq, err := ToGeminiRequestFromJSON(body)
	if err != nil {
		return nil, err
	}

	if geminiReq == nil {
		return nil, nil
	}

	// 转换为 JSON
	resultJSON, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, err
	}

	wlog.LogWithLine("[%s] Request transformed from unified format to gemini-3-image format", config.PluginName)
	return resultJSON, nil
}

// transformGeminiResponse 将 gemini-3-image 响应转换为统一格式（qwen-image 标准）
func transformGeminiResponse(body []byte) ([]byte, error) {
	wlog.LogWithLine("[%s] Starting gemini response transformation, body length: %d", config.PluginName, len(body))

	// 使用新的转换方法
	unifiedResp, err := ToUnifiedResponseFromGeminiJSON(body)
	if err != nil {
		wlog.LogWithLine("[%s] Failed to convert gemini response: %v", config.PluginName, err)
		return nil, err
	}

	if unifiedResp == nil {
		wlog.LogWithLine("[%s] Unified response is nil, returning nil", config.PluginName)
		return nil, nil
	}

	// 转换为 JSON
	resultJSON, err := json.Marshal(unifiedResp)
	if err != nil {
		wlog.LogWithLine("[%s] Failed to marshal unified response: %v", config.PluginName, err)
		return nil, err
	}

	wlog.LogWithLine("[%s] Response transformed from gemini-3-image format to unified format, result length: %d", config.PluginName, len(resultJSON))
	return resultJSON, nil
}

// isGeminiStreamingResponse 判断 gemini 模型是否为流式响应
// gemini 模型：始终返回 false，强制非流处理
func isGeminiStreamingResponse(isChunked, hasStreamParam bool) bool {
	return false
}

// processGeminiResponseHeaders 处理 gemini 模型的响应头
// gemini 模型：删除 transfer-encoding header，确保非流处理
func processGeminiResponseHeaders(transferEncoding string, hasStreamRequest bool) {
	if transferEncoding == "chunked" {
		proxywasm.RemoveHttpResponseHeader("transfer-encoding")
		wlog.LogWithLine("[%s] Removed transfer-encoding header (gemini, force non-streaming)", config.PluginName)
	}
}
