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
	"strings"
	"transformer-vnet/config"
	"transformer-vnet/wlog"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
)

// DoubaoRequest doubao-seedream-4.5 实际请求格式
type DoubaoRequest struct {
	Model                            string                            `json:"model"`
	Prompt                           string                            `json:"prompt"`
	Size                             string                            `json:"size,omitempty"`
	Watermark                        bool                              `json:"watermark,omitempty"`
	Seed                             int64                             `json:"seed,omitempty"`
	SequentialImageGeneration        string                            `json:"sequential_image_generation,omitempty"`
	SequentialImageGenerationOptions *SequentialImageGenerationOptions `json:"sequential_image_generation_options,omitempty"`
	ResponseFormat                   string                            `json:"response_format,omitempty"`
	Stream                           bool                              `json:"stream,omitempty"`
	N                                int                               `json:"n,omitempty"`
}

// SequentialImageGenerationOptions 序列图片生成选项
type SequentialImageGenerationOptions struct {
	MaxImages int `json:"max_images"`
}

// DoubaoResponse doubao-seedream-4.5 实际响应格式
type DoubaoResponse struct {
	Model   string        `json:"model"`
	Created int64         `json:"created"`
	Data    []DoubaoImage `json:"data"`
	Usage   DoubaoUsage   `json:"usage,omitempty"`
	Error   *DoubaoError  `json:"error,omitempty"`
}

// DoubaoImage doubao 图片格式
type DoubaoImage struct {
	URL           string `json:"url,omitempty"`
	B64JSON       string `json:"b64_json,omitempty"`
	Size          string `json:"size,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

// DoubaoUsage doubao 使用情况格式
type DoubaoUsage struct {
	GeneratedImages int64 `json:"generated_images"`
	OutputTokens    int64 `json:"output_tokens,omitempty"`
	TotalTokens     int64 `json:"total_tokens,omitempty"`
}

// DoubaoError doubao 错误格式
type DoubaoError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param,omitempty"`
	Code    string `json:"code,omitempty"`
}

// ToDoubaoRequest 将统一请求格式转换为 doubao-seedream-4.5 请求格式
func ToDoubaoRequest(unified *UnifiedRequest) (*DoubaoRequest, error) {
	if unified == nil {
		return nil, nil
	}

	doubao := &DoubaoRequest{
		Model: unified.Model,
	}

	// 1. 提取 prompt: 从 input.messages[0].content[0].text 提取
	if len(unified.Input.Messages) > 0 {
		message := unified.Input.Messages[0]
		if len(message.Content) > 0 {
			doubao.Prompt = message.Content[0].Text
		}
	}

	// 2. 提取 parameters 中的参数（除 n 外）
	if unified.Parameters != nil {
		if size, ok := unified.Parameters["size"].(string); ok {
			// 统一处理 size 参数：将 qwen 格式（宽*高）映射到 doubao 格式（2K/4K 或像素格式）
			doubao.Size = mapSizeFromQwenToDoubao(size)
		}
		if watermark, ok := unified.Parameters["watermark"].(bool); ok {
			doubao.Watermark = watermark
		}
		if seed, ok := unified.Parameters["seed"].(float64); ok {
			doubao.Seed = int64(seed)
		}
	}

	// 3. 映射 n 到 max_images: 将 parameters.n 映射到 sequential_image_generation_options.max_images
	if unified.Parameters != nil {
		if n, ok := unified.Parameters["n"].(float64); ok && n > 0 {
			doubao.SequentialImageGeneration = "auto"
			doubao.SequentialImageGenerationOptions = &SequentialImageGenerationOptions{
				MaxImages: int(n),
			}
		}
	}

	// 4. 提取 extra_body.doubao_seedream 中的参数
	if unified.ExtraBody != nil {
		if doubaoSeedream, ok := unified.ExtraBody["doubao_seedream"].(map[string]interface{}); ok {
			if sequentialImageGen, ok := doubaoSeedream["sequential_image_generation"].(string); ok {
				doubao.SequentialImageGeneration = sequentialImageGen
			}
			if responseFormat, ok := doubaoSeedream["response_format"].(string); ok {
				doubao.ResponseFormat = responseFormat
			}
			if stream, ok := doubaoSeedream["stream"].(bool); ok {
				doubao.Stream = stream
			}
		}
	}

	wlog.LogWithLine("[%s] Converted unified request to doubao-seedream format", config.PluginName)
	return doubao, nil
}

// ToDoubaoRequestFromJSON 从 JSON 字节数组转换为 doubao-seedream 请求格式
func ToDoubaoRequestFromJSON(body []byte) (*DoubaoRequest, error) {
	// 先解析为统一格式
	var unified UnifiedRequest
	if err := json.Unmarshal(body, &unified); err != nil {
		return nil, err
	}

	// 转换为 doubao 格式
	return ToDoubaoRequest(&unified)
}

// ToUnifiedResponse 将 doubao-seedream-4.5 响应格式转换为统一响应格式
func ToUnifiedResponse(doubao *DoubaoResponse) (*UnifiedResponse, error) {
	if doubao == nil {
		return nil, nil
	}

	// 如果有错误，直接返回错误响应
	if doubao.Error != nil {
		return &UnifiedResponse{
			RequestID:  "",
			Code:       doubao.Error.Code,
			Message:    doubao.Error.Message,
			StatusCode: 400,
		}, nil
	}

	// 1. 构建 content 数组
	contentArray := make([]UnifiedContent, 0)
	for _, image := range doubao.Data {
		// 优先使用 url，如果没有则使用 b64_json
		if image.URL != "" {
			contentArray = append(contentArray, UnifiedContent{
				Image: image.URL,
			})
			wlog.LogWithLine("[%s] Added image URL to content array: %s", config.PluginName, image.URL)
		} else if image.B64JSON != "" {
			contentArray = append(contentArray, UnifiedContent{
				Image: image.B64JSON,
			})
			wlog.LogWithLine("[%s] Added image b64_json to content array", config.PluginName)
		}
	}

	// 如果没有 content，记录警告
	if len(contentArray) == 0 {
		wlog.LogWithLine("[%s] Warning: No content found in doubao response data", config.PluginName)
	}

	// 2. 提取 created 作为 request_id（转换为字符串）
	requestID := ""
	if doubao.Created > 0 {
		// 将 int64 转换为字符串
		requestID = strconv.FormatInt(doubao.Created, 10)
	}

	// 3. 提取 usage.generated_images 作为 usage.image_count
	imageCount := int(doubao.Usage.GeneratedImages)
	if imageCount == 0 && len(doubao.Data) > 0 {
		// 如果没有 generated_images，使用 data 数组的长度
		imageCount = len(doubao.Data)
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
				Succeeded: imageCount,
				Total:     imageCount,
			},
		},
		Usage: UnifiedUsage{
			ImageCount: imageCount,
		},
	}

	wlog.LogWithLine("[%s] Converted doubao-seedream response to unified format", config.PluginName)
	return unified, nil
}

// checkDoubaoStreamParameter 检查 doubao 请求中是否有 stream 参数
// 对于 doubao 模型，stream 参数在 extra_body.doubao_seedream.stream 中
func checkDoubaoStreamParameter(body []byte) bool {
	var unified UnifiedRequest
	if err := json.Unmarshal(body, &unified); err != nil {
		return false
	}

	if unified.ExtraBody != nil {
		if doubaoSeedream, ok := unified.ExtraBody["doubao_seedream"].(map[string]interface{}); ok {
			if stream, ok := doubaoSeedream["stream"].(bool); ok && stream {
				return true
			}
		}
	}

	return false
}

// isDoubaoStreamingResponse 判断 doubao 模型是否为流式响应
// doubao 模型：需要同时满足 transfer-encoding chunked 和 stream 参数
func isDoubaoStreamingResponse(isChunked, hasStreamParam bool) bool {
	return isChunked && hasStreamParam
}

// processDoubaoResponseHeaders 处理 doubao 模型的响应头
// doubao 模型：如果不是流式请求，删除 transfer-encoding header
func processDoubaoResponseHeaders(transferEncoding string, hasStreamRequest bool) {
	if !hasStreamRequest && transferEncoding == "chunked" {
		proxywasm.RemoveHttpResponseHeader("transfer-encoding")
		wlog.LogWithLine("[%s] Removed transfer-encoding header (doubao, not a stream request)", config.PluginName)
	}
}

// transformDoubaoRequest 将统一格式转换为 doubao-seedream 格式
func transformDoubaoRequest(body []byte) ([]byte, error) {
	// 使用新的转换方法
	doubaoReq, err := ToDoubaoRequestFromJSON(body)
	if err != nil {
		return nil, err
	}

	if doubaoReq == nil {
		return nil, nil
	}

	// 转换为 JSON
	resultJSON, err := json.Marshal(doubaoReq)
	if err != nil {
		return nil, err
	}

	wlog.LogWithLine("[%s] Request transformed from unified format to doubao-seedream format", config.PluginName)
	return resultJSON, nil
}

// transformDoubaoResponse 将 doubao-seedream 响应转换为统一格式（qwen-image 标准）
func transformDoubaoResponse(body []byte) ([]byte, error) {
	wlog.LogWithLine("[%s] Starting doubao response transformation, body length: %d", config.PluginName, len(body))

	// 使用新的转换方法
	unifiedResp, err := ToUnifiedResponseFromJSON(body)
	if err != nil {
		wlog.LogWithLine("[%s] Failed to convert doubao response: %v", config.PluginName, err)
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

	wlog.LogWithLine("[%s] Response transformed from doubao-seedream format to unified format, result length: %d", config.PluginName, len(resultJSON))
	return resultJSON, nil
}

// mapSizeFromQwenToDoubao 将 qwen 格式的 size（宽*高）映射到 doubao 格式（2K/4K）
// qwen 格式示例: "1328*1328", "1664*928", "928*1664", "1472*1104", "1104*1472"
// doubao 格式: "2K" 或 "4K"（只接受这两种格式）
// 映射规则：
// - 如果已经是 doubao 格式（2K、4K），直接返回
// - qwen 格式（宽*高）根据总像素数映射：
//   - 总像素数 < 9,000,000（3K x 3K），转换为 "2K"
//   - 总像素数 >= 9,000,000（3K x 3K），转换为 "4K"
func mapSizeFromQwenToDoubao(qwenSize string) string {
	if qwenSize == "" {
		return ""
	}

	// 如果已经是 doubao 格式（2K、4K），直接返回
	upperSize := strings.ToUpper(qwenSize)
	if upperSize == "2K" || upperSize == "4K" {
		return upperSize
	}

	// 如果包含 "x"，可能是像素格式，尝试解析并转换
	if strings.Contains(qwenSize, "x") {
		parts := strings.Split(qwenSize, "x")
		if len(parts) == 2 {
			width, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
			height, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err1 == nil && err2 == nil {
				totalPixels := width * height
				if totalPixels < 9000000 {
					return "2K"
				}
				return "4K"
			}
		}
		// 解析失败，返回原值
		wlog.LogWithLine("[%s] Failed to parse pixel size format: %s, returning as is", config.PluginName, qwenSize)
		return qwenSize
	}

	// 解析 qwen 格式：宽*高
	parts := strings.Split(qwenSize, "*")
	if len(parts) != 2 {
		// 如果格式不正确，返回原值
		wlog.LogWithLine("[%s] Invalid qwen size format: %s, returning as is", config.PluginName, qwenSize)
		return qwenSize
	}

	width, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	height, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil {
		// 如果解析失败，返回原值
		wlog.LogWithLine("[%s] Failed to parse qwen size: %s, returning as is", config.PluginName, qwenSize)
		return qwenSize
	}

	// 计算总像素数
	totalPixels := width * height

	// 根据总像素数映射到 doubao 格式
	// 3K x 3K = 9,000,000 像素作为分界点
	// 小于 9,000,000 像素 -> "2K"
	// 大于等于 9,000,000 像素 -> "4K"
	if totalPixels < 9000000 {
		return "2K"
	}
	return "4K"
}

// abs 返回整数的绝对值
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// ToUnifiedResponseFromJSON 从 JSON 字节数组转换为统一响应格式
func ToUnifiedResponseFromJSON(body []byte) (*UnifiedResponse, error) {
	// 先解析为 doubao 格式
	var doubao DoubaoResponse
	if err := json.Unmarshal(body, &doubao); err != nil {
		wlog.LogWithLine("[%s] Failed to unmarshal doubao response: %v", config.PluginName, err)
		return nil, err
	}

	wlog.LogWithLine("[%s] Successfully parsed doubao response, Model: %s, Created: %d, Data count: %d",
		config.PluginName, doubao.Model, doubao.Created, len(doubao.Data))

	// 转换为统一格式
	return ToUnifiedResponse(&doubao)
}
