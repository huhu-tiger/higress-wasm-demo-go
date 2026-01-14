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
	"transformer-vnet/config"
	"transformer-vnet/wlog"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
)

// QwenRequest qwen-image-plus 实际请求格式（与统一格式相同）
// 注意：qwen-image 的请求格式就是统一格式，无需转换
type QwenRequest struct {
	Model      string                 `json:"model"`
	Input      UnifiedInput           `json:"input"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// QwenResponse qwen-image-plus 实际响应格式（与统一格式相同）
// 注意：qwen-image 的响应格式就是统一格式，无需转换
type QwenResponse struct {
	RequestID  string        `json:"request_id"`
	Output     UnifiedOutput `json:"output"`
	Usage      UnifiedUsage  `json:"usage"`
	Code       string        `json:"code,omitempty"`
	Message    string        `json:"message,omitempty"`
	StatusCode int           `json:"status_code,omitempty"`
}

// ToQwenRequest 将统一格式转换为 qwen-image 格式
// 注意：qwen-image 的格式就是统一格式，所以直接返回统一格式即可
func ToQwenRequest(unified *UnifiedRequest) (*QwenRequest, error) {
	if unified == nil {
		return nil, nil
	}

	// qwen-image 的格式就是统一格式，直接转换
	qwen := &QwenRequest{
		Model:      unified.Model,
		Input:      unified.Input,
		Parameters: unified.Parameters,
	}

	wlog.LogWithLine("[%s] Converted unified request to qwen-image format (no conversion needed)", config.PluginName)
	return qwen, nil
}

// ToQwenRequestFromJSON 从 JSON 字节数组转换为 qwen-image 请求格式
func ToQwenRequestFromJSON(body []byte) (*QwenRequest, error) {
	// qwen-image 的格式就是统一格式，直接解析为统一格式即可
	var unified UnifiedRequest
	if err := json.Unmarshal(body, &unified); err != nil {
		return nil, err
	}

	return ToQwenRequest(&unified)
}

// ToUnifiedResponse 将 qwen-image 响应转换为统一格式
// 注意：qwen-image 的格式就是统一格式，所以直接转换即可
func ToUnifiedResponseFromQwen(qwen *QwenResponse) (*UnifiedResponse, error) {
	if qwen == nil {
		return nil, nil
	}

	// qwen-image 的格式就是统一格式，直接转换
	unified := &UnifiedResponse{
		RequestID:  qwen.RequestID,
		Output:     qwen.Output,
		Usage:      qwen.Usage,
		Code:       qwen.Code,
		Message:    qwen.Message,
		StatusCode: qwen.StatusCode,
	}

	wlog.LogWithLine("[%s] Converted qwen-image response to unified format (no conversion needed)", config.PluginName)
	return unified, nil
}

// ToUnifiedResponseFromQwenJSON 从 JSON 字节数组转换为统一响应格式
func ToUnifiedResponseFromQwenJSON(body []byte) (*UnifiedResponse, error) {
	// 先解析为 qwen 格式
	var qwen QwenResponse
	if err := json.Unmarshal(body, &qwen); err != nil {
		return nil, err
	}

	// 转换为统一格式
	return ToUnifiedResponseFromQwen(&qwen)
}

// transformQwenRequest 将统一格式转换为 qwen-image 格式（实际无需转换）
func transformQwenRequest(_ []byte) ([]byte, error) {
	// qwen-image 的格式就是统一格式，无需转换，直接返回 nil 表示不需要转换
	wlog.LogWithLine("[%s] Qwen-image request format is already unified, no conversion needed", config.PluginName)
	return nil, nil
}

// transformQwenResponse 将 qwen-image 响应转换为统一格式（实际无需转换）
func transformQwenResponse(_ []byte) ([]byte, error) {
	// qwen-image 的格式就是统一格式，无需转换，直接返回 nil 表示不需要转换
	wlog.LogWithLine("[%s] Qwen-image response format is already unified, no conversion needed", config.PluginName)
	return nil, nil
}

// isQwenStreamingResponse 判断 qwen 模型是否为流式响应
// qwen 模型：不支持流式响应，始终返回 false
// 即使响应是 chunked 传输，也应该缓冲响应体进行转换
func isQwenStreamingResponse(isChunked, hasStreamParam bool) bool {
	// qwen 模型不支持流式响应，即使使用 chunked 传输，也需要缓冲响应体进行格式转换
	return false
}

// processQwenResponseHeaders 处理 qwen 模型的响应头
// qwen 模型：如果不是流式请求，删除 transfer-encoding header
func processQwenResponseHeaders(transferEncoding string, hasStreamRequest bool) {
	if !hasStreamRequest && transferEncoding == "chunked" {
		proxywasm.RemoveHttpResponseHeader("transfer-encoding")
		wlog.LogWithLine("[%s] Removed transfer-encoding header (qwen, not a stream request)", config.PluginName)
	}
}
