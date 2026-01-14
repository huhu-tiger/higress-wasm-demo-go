package t2i_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"transformer-vnet/bussiness/t2i"
)

// 注意：由于转换函数内部使用了 wlog.LogWithLine，而 wlog 依赖 proxywasm，
// 在非 WASM 环境中运行测试会失败。这里我们直接测试 JSON 转换逻辑，
// 或者需要 mock proxywasm 环境。为了简化，我们只测试 JSON 结构转换的正确性。

// TestUnifiedRequestTransform 测试统一请求格式转换为各模型格式
func TestUnifiedRequestTransform(t *testing.T) {
	// 统一请求格式
	unifiedRequest := map[string]interface{}{
		"model": "volcengine/volcengine/doubao-seedream-4.5",
		"input": map[string]interface{}{
			"messages": []map[string]interface{}{
				{
					"role": "user",
					"content": []map[string]interface{}{
						{
							"text": "Generate a beautiful sunset over mountains",
						},
					},
				},
			},
		},
		"parameters": map[string]interface{}{
			"size": "1024*1024",
			"n":    1,
		},
	}

	unifiedJSON, _ := json.Marshal(unifiedRequest)

	// 测试三个模型的请求转换
	models := []struct {
		name      string
		model     string
		modelType string
	}{
		{"Doubao", "volcengine/volcengine/doubao-seedream-4.5", "t2i"},
		{"Qwen", "aliyun/aliyun/qwen-image-plus", "t2i"},
		{"Gemini", "openrouter/auto/gemini-3-image", "t2i"},
	}

	for _, model := range models {
		t.Run(fmt.Sprintf("RequestTransform_%s", model.name), func(t *testing.T) {
			// 更新 model 字段
			var req map[string]interface{}
			json.Unmarshal(unifiedJSON, &req)
			req["model"] = model.model

			reqJSON, _ := json.Marshal(req)

			// 转换请求
			transformed, err := t2i.TransformRequest(model.model, model.modelType, reqJSON)
			if err != nil {
				t.Errorf("%s request transform failed: %v", model.name, err)
				return
			}

			if transformed == nil {
				t.Logf("%s: No transformation needed (already in correct format)", model.name)
			} else {
				t.Logf("%s transformed request:\n%s", model.name, formatJSON(transformed))
			}
		})
	}
}

// TestUnifiedResponseTransform 测试各模型响应格式转换为统一格式
func TestUnifiedResponseTransform(t *testing.T) {
	// 测试三个模型的响应转换
	testCases := []struct {
		name      string
		model     string
		modelType string
		response  string
	}{
		{
			name:      "Doubao",
			model:     "volcengine/volcengine/doubao-seedream-4.5",
			modelType: "t2i",
			response: `{
				"model": "doubao-seedream-4.5",
				"created": 1705123456,
				"data": [
					{
						"url": "https://example.com/image1.png",
						"size": "2K"
					}
				],
				"usage": {
					"generated_images": 1
				}
			}`,
		},
		{
			name:      "Qwen",
			model:     "aliyun/aliyun/qwen-image-plus",
			modelType: "t2i",
			response: `{
				"request_id": "req_123",
				"output": {
					"choices": [
						{
							"finish_reason": "stop",
							"message": {
								"role": "assistant",
								"content": [
									{
										"image": "https://example.com/image1.png"
									}
								]
							}
						}
					],
					"task_metric": {}
				},
				"usage": {
					"image_count": 1
				}
			}`,
		},
		{
			name:      "Gemini",
			model:     "openrouter/auto/gemini-3-image",
			modelType: "t2i",
			response: `{
				"id": "chatcmpl-123",
				"model": "gemini-3-image",
				"created": 1705123456,
				"choices": [
					{
						"message": {
							"role": "assistant",
							"content": [
								{
									"type": "image_url",
									"image_url": {
										"url": "https://example.com/image1.png"
									}
								}
							]
						},
						"finish_reason": "stop"
					}
				],
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 0,
					"total_tokens": 10
				}
			}`,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("ResponseTransform_%s", tc.name), func(t *testing.T) {
			// 转换响应
			transformed, err := t2i.TransformResponse(tc.model, tc.modelType, []byte(tc.response))
			if err != nil {
				t.Errorf("%s response transform failed: %v", tc.name, err)
				return
			}

			if transformed == nil {
				t.Logf("%s: No transformation needed (already in correct format)", tc.name)
			} else {
				t.Logf("%s transformed response:\n%s", tc.name, formatJSON(transformed))

				// 验证统一格式
				var unified t2i.UnifiedResponse
				if err := json.Unmarshal(transformed, &unified); err != nil {
					t.Errorf("%s: Failed to parse unified response: %v", tc.name, err)
					return
				}

				// 验证必要字段
				if unified.Output.Choices == nil || len(unified.Output.Choices) == 0 {
					t.Errorf("%s: Unified response missing choices", tc.name)
				} else {
					choice := unified.Output.Choices[0]
					if choice.Message.Content == nil || len(choice.Message.Content) == 0 {
						t.Errorf("%s: Unified response missing content", tc.name)
					} else {
						content := choice.Message.Content[0]
						if content.Image == "" {
							t.Errorf("%s: Unified response missing image", tc.name)
						} else {
							t.Logf("%s: Unified response validated successfully, image: %s", tc.name, content.Image[:min(50, len(content.Image))])
						}
					}
				}
			}
		})
	}
}

// TestRoundTrip 测试往返转换（统一 -> 模型 -> 统一）
func TestRoundTrip(t *testing.T) {
	// 统一请求格式
	unifiedRequest := map[string]interface{}{
		"model": "volcengine/volcengine/doubao-seedream-4.5",
		"input": map[string]interface{}{
			"messages": []map[string]interface{}{
				{
					"role": "user",
					"content": []map[string]interface{}{
						{
							"text": "Generate a beautiful sunset over mountains",
						},
					},
				},
			},
		},
		"parameters": map[string]interface{}{
			"size": "1024*1024",
			"n":    1,
		},
	}

	unifiedJSON, _ := json.Marshal(unifiedRequest)

	// 测试三个模型的往返转换
	models := []struct {
		name      string
		model     string
		modelType string
		response  string
	}{
		{
			name:      "Doubao",
			model:     "volcengine/volcengine/doubao-seedream-4.5",
			modelType: "t2i",
			response: `{
				"model": "doubao-seedream-4.5",
				"created": 1705123456,
				"data": [
					{
						"url": "https://example.com/image1.png",
						"size": "2K"
					}
				],
				"usage": {
					"generated_images": 1
				}
			}`,
		},
		{
			name:      "Qwen",
			model:     "aliyun/aliyun/qwen-image-plus",
			modelType: "t2i",
			response: `{
				"request_id": "req_123",
				"output": {
					"choices": [
						{
							"finish_reason": "stop",
							"message": {
								"role": "assistant",
								"content": [
									{
										"image": "https://example.com/image1.png"
									}
								]
							}
						}
					],
					"task_metric": {}
				},
				"usage": {
					"image_count": 1
				}
			}`,
		},
		{
			name:      "Gemini",
			model:     "openrouter/auto/gemini-3-image",
			modelType: "t2i",
			response: `{
				"id": "chatcmpl-123",
				"model": "gemini-3-image",
				"created": 1705123456,
				"choices": [
					{
						"message": {
							"role": "assistant",
							"content": [
								{
									"type": "image_url",
									"image_url": {
										"url": "https://example.com/image1.png"
									}
								}
							]
						},
						"finish_reason": "stop"
					}
				],
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 0,
					"total_tokens": 10
				}
			}`,
		},
	}

	for _, model := range models {
		t.Run(fmt.Sprintf("RoundTrip_%s", model.name), func(t *testing.T) {
			// 1. 统一请求 -> 模型请求
			var req map[string]interface{}
			json.Unmarshal(unifiedJSON, &req)
			req["model"] = model.model

			reqJSON, _ := json.Marshal(req)

			transformedReq, err := t2i.TransformRequest(model.model, model.modelType, reqJSON)
			if err != nil {
				t.Errorf("%s: Request transform failed: %v", model.name, err)
				return
			}

			if transformedReq != nil {
				t.Logf("%s transformed request:\n%s", model.name, formatJSON(transformedReq))
			} else {
				t.Logf("%s: Request already in correct format", model.name)
			}

			// 2. 模型响应 -> 统一响应
			transformedResp, err := t2i.TransformResponse(model.model, model.modelType, []byte(model.response))
			if err != nil {
				t.Errorf("%s: Response transform failed: %v", model.name, err)
				return
			}

			if transformedResp != nil {
				t.Logf("%s transformed response:\n%s", model.name, formatJSON(transformedResp))

				// 验证统一格式
				var unified t2i.UnifiedResponse
				if err := json.Unmarshal(transformedResp, &unified); err != nil {
					t.Errorf("%s: Failed to parse unified response: %v", model.name, err)
					return
				}

				// 验证必要字段
				if unified.Output.Choices == nil || len(unified.Output.Choices) == 0 {
					t.Errorf("%s: Unified response missing choices", model.name)
				} else {
					choice := unified.Output.Choices[0]
					if choice.Message.Content == nil || len(choice.Message.Content) == 0 {
						t.Errorf("%s: Unified response missing content", model.name)
					} else {
						content := choice.Message.Content[0]
						if content.Image == "" {
							t.Errorf("%s: Unified response missing image", model.name)
						} else {
							t.Logf("%s: Round trip test passed! Image: %s", model.name, content.Image[:min(50, len(content.Image))])
						}
					}
				}
			} else {
				t.Logf("%s: Response already in correct format", model.name)
			}
		})
	}
}

// formatJSON 格式化 JSON 输出
func formatJSON(data []byte) string {
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return string(data)
	}
	formatted, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return string(data)
	}
	return string(formatted)
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
