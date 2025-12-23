// Copyright (c) 2025 Alibaba Group Holding Ltd.
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

package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"ai-data-masking/config"
	"ai-data-masking/lib"
	"ai-data-masking/wlog"

	"github.com/google/uuid"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

func main() {}

func init() {
	wrapper.SetCtx(
		"ai-data-masking",
		wrapper.ParseConfig(parseConfig),
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
		wrapper.ProcessRequestBody(onHttpRequestBody),
		wrapper.ProcessResponseHeaders(onHttpResponseHeaders),
		wrapper.ProcessResponseBody(onHttpResponseBody),
		wrapper.ProcessStreamingResponseBody(onHttpStreamingResponseBody),
		wrapper.WithRebuildAfterRequests[config.AiDataMaskingConfig](1000),
	)
}

const (
	pluginName = "ai-data-masking"
)

func parseConfig(json gjson.Result, cfg *config.AiDataMaskingConfig) error {
	// 设置默认值
	cfg.DenyOpenAI = json.Get("deny_openai").Bool()
	if !json.Get("deny_openai").Exists() {
		cfg.DenyOpenAI = true // 默认值
	}

	cfg.DenyRaw = json.Get("deny_raw").Bool()
	cfg.SystemDeny = json.Get("system_deny").Bool()

	// 解析 deny_code
	if json.Get("deny_code").Exists() {
		cfg.DenyCode = uint32(json.Get("deny_code").Int())
	} else {
		cfg.DenyCode = 200 // 默认值
	}

	// 解析 deny_message
	cfg.DenyMessage = json.Get("deny_message").String()
	if cfg.DenyMessage == "" {
		cfg.DenyMessage = "提问或回答中包含敏感词，已被屏蔽"
	}

	// 解析 deny_raw_message
	cfg.DenyRawMessage = json.Get("deny_raw_message").String()
	if cfg.DenyRawMessage == "" {
		cfg.DenyRawMessage = `{"errmsg":"提问或回答中包含敏感词，已被屏蔽"}`
	}

	// 解析 deny_content_type
	cfg.DenyContentType = json.Get("deny_content_type").String()
	if cfg.DenyContentType == "" {
		cfg.DenyContentType = "application/json"
	}

	// 解析 deny_jsonpath
	for _, item := range json.Get("deny_jsonpath").Array() {
		path := item.String()
		if path != "" {
			cfg.DenyJSONPath = append(cfg.DenyJSONPath, path)
		}
	}

	// 解析 deny_words
	for _, item := range json.Get("deny_words").Array() {
		word := strings.TrimSpace(item.String())
		if word != "" {
			cfg.DenyWords = append(cfg.DenyWords, word)
		}
	}

	// 解析 replace_roles
	for _, item := range json.Get("replace_roles").Array() {
		rule := config.Rule{
			Regex:   item.Get("regex").String(),
			Type:    item.Get("type").String(),
			Restore: item.Get("restore").Bool(),
			Value:   item.Get("value").String(),
		}

		// 编译正则表达式（支持 GROK 模式）
		if rule.Regex != "" {
			pattern := convertGrokToRegex(rule.Regex)
			compiled, err := regexp.Compile(pattern)
			if err != nil {
				proxywasm.LogWarnf("failed to compile regex %s: %v", rule.Regex, err)
				continue
			}
			rule.CompiledRegex = compiled
		}

		cfg.ReplaceRoles = append(cfg.ReplaceRoles, rule)
	}

	return nil
}

// convertGrokToRegex 将 GROK 模式转换为正则表达式（简化版）
func convertGrokToRegex(grokPattern string) string {
	// 这里实现 GROK 到正则的转换
	// 简化实现，支持常见的 GROK 模式
	patterns := map[string]string{
		"%{MOBILE}":                            `\d{8,11}`,
		"%{IDCARD}":                            `\d{17}[0-9xX]|\d{15}`,
		"%{IP}":                                `\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`,
		"%{EMAILLOCALPART}":                    `[a-zA-Z0-9._%+-]+`,
		"%{HOSTNAME:domain}":                   `([a-zA-Z0-9.-]+)`,
		"%{EMAILLOCALPART}@%{HOSTNAME:domain}": `[a-zA-Z0-9._%+-]+@([a-zA-Z0-9.-]+)`,
	}

	// 检查是否有预定义的模式
	if pattern, ok := patterns[grokPattern]; ok {
		return pattern
	}

	// 简单的 GROK 模式替换
	result := grokPattern
	for grok, regex := range patterns {
		result = strings.ReplaceAll(result, grok, regex)
	}

	// 如果没有匹配，返回原始字符串（可能是标准正则）
	return result
}

// getOrCreatePluginContext 获取或创建插件上下文
func getOrCreatePluginContext(ctx wrapper.HttpContext, cfg *config.AiDataMaskingConfig) *config.PluginContext {
	contextKey := pluginName + "_context"
	value := ctx.GetContext(contextKey)
	if value != nil {
		if pluginCtx, ok := value.(*config.PluginContext); ok {
			return pluginCtx
		}
	}

	pluginCtx := &config.PluginContext{
		Config:                cfg,
		MaskMap:               make(map[string]*string),
		OpenAIRequest:         &config.OpenAIRequest{},
		StreamContentBuffer:   "", // 初始化流式响应缓冲区
		StreamReasoningBuffer: "", // 初始化流式响应缓冲区
		StreamDenied:          false,
		StreamChunkBuffer:     make([]config.StreamChunk, 0), // 初始化 chunk 缓冲区
		StreamChunkBufferSize: 0,                             // 初始化缓冲区大小
	}
	ctx.SetContext(contextKey, pluginCtx)
	return pluginCtx
}
func onHttpRequestHeaders(ctx wrapper.HttpContext, cfg config.AiDataMaskingConfig) types.Action {
	// 禁用重路由
	ctx.DisableReroute()
	pluginCtx := getOrCreatePluginContext(ctx, &cfg)
	pluginCtx.Step = config.StepRequestHeader
	wlog.LogWithLine("[%s] Process Step: %s", pluginName, pluginCtx.Step.String())
	// 检查是否有请求体
	contentLength, err := proxywasm.GetHttpRequestHeader("content-length")
	if err == nil && contentLength != "0" && contentLength != "" {
		// 移除 Content-Length，让 Envoy 重新计算
		proxywasm.RemoveHttpRequestHeader("content-length")
		return types.ActionContinue
	}
	if err != nil {
		// proxywasm.LogErrorf("failed to get content-length: %v", err)
		return types.ActionContinue
	}

	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, cfg config.AiDataMaskingConfig, body []byte) types.Action {
	pluginCtx := getOrCreatePluginContext(ctx, &cfg)
	pluginCtx.Step = config.StepRequestBody
	wlog.LogWithLine("[%s] Process Step: %s", pluginName, pluginCtx.Step.String())
	ctx.SetRequestBodyBufferLimit(config.DEFAULT_MAX_BODY_BYTES)
	// 如果配置了OpenAI拒绝，则处理OpenAI请求
	if cfg.DenyOpenAI {
		var modified bool
		var denied bool
		// 请求体阶段处理OpenAI请求
		modified, denied = lib.ProcessOpenAIRequest(ctx, pluginCtx, body)
		// 如果匹配到敏感词
		if denied {
			pluginCtx.IsDeny = true
			pluginCtx.IsRequestDeny = true
			pluginCtx.RequestDenyModifyType = config.DenyModifyTypeOpenAI
			ctx.SetUserAttribute("x-ai-data-masking", string(pluginCtx.RequestDenyModifyType))
			ctx.SetUserAttribute("deny_step", pluginCtx.Step.String())
			ctx.SetUserAttribute("deny_code", fmt.Sprintf("%d", cfg.DenyCode))

			// 设置标志，表示响应已在请求阶段发送，响应阶段的回调应该跳过处理
			ctx.SetUserAttribute("response_sent_in_request", "true")

			var openaiResponseJson []byte
			// 根据是否为流式请求构造不同的响应格式
			if pluginCtx.OpenAIRequest.Stream {
				// 流式响应：使用 SSE 格式
				streamResponse := config.OpenAIStreamCompletionResponse{
					Id:      uuid.New().String(),
					Object:  "chat.completion.chunk",
					Created: 123,
					Model:   pluginCtx.OpenAIRequest.Model,
					Choices: []config.OpenAIStreamChoice{
						{
							Index: 0,
							Delta: &config.OpenAIMessage{
								Role:    "assistant",
								Content: cfg.DenyMessage,
							},
							FinishReason: config.FINISH_REASON_STOP,
						},
					},
				}
				streamJson, _ := json.Marshal(streamResponse)
				// SSE 格式：data: {...}\n\ndata:[DONE]\n\n
				openaiResponseJson = []byte(fmt.Sprintf("data: %s\n\ndata: [DONE]\n\n", string(streamJson)))
			} else {
				// 非流式响应
				openaiResponse := config.OpenAICompletionResponse{
					Id:      uuid.New().String(),
					Object:  "chat.completion",
					Created: 123,
					Model:   pluginCtx.OpenAIRequest.Model,
					Choices: []config.OpenAICompletionChoice{
						{
							Index: 0,
							Message: &config.OpenAIMessage{
								Role:    "assistant",
								Content: cfg.DenyMessage,
							},
						},
					},
					Usage: &config.OpenAIUsage{
						PromptTokens:     0,
						CompletionTokens: 0,
						TotalTokens:      0,
					},
				}
				openaiResponseJson, _ = json.Marshal(openaiResponse) // []byte
			}
			ctx.SetUserAttribute("deny_message", openaiResponseJson)
			wlog.LogWithLine("[%s] onHttpRequestBody: pluginCtx.OpenAIRequest.Model=%s", pluginName, pluginCtx.OpenAIRequest.Model)
			wlog.LogWithLine("[%s] onHttpRequestBody DenyModifyType:%s Stream:%v deny() called: deny_message=%s",
				pluginName, pluginCtx.RequestDenyModifyType, pluginCtx.OpenAIRequest.Stream, cfg.DenyMessage)

			return lib.DenyHandler(ctx, pluginCtx)
		}

		if modified {
			pluginCtx.IsModified = true
			pluginCtx.RequestDenyModifyType = config.DenyModifyTypeOpenAI
			pluginCtx.IsRequestModified = true
			proxywasm.ReplaceHttpRequestBody(body)
		}
	}

	// 如果配置了JSONPath拒绝，则处理JSONPath请求
	if len(cfg.DenyJSONPath) > 0 {
		var modified bool
		var denied bool

		modified, denied = lib.ProcessJSONPathRequest(ctx, pluginCtx, body)
		if denied {
			pluginCtx.IsDeny = true
			pluginCtx.IsRequestDeny = true
			pluginCtx.RequestDenyModifyType = config.DenyModifyTypeJSONPath

			ctx.SetUserAttribute("x-ai-data-masking", string(pluginCtx.RequestDenyModifyType))
			ctx.SetUserAttribute("deny_step", pluginCtx.Step.String())
			ctx.SetUserAttribute("deny_code", fmt.Sprintf("%d", cfg.DenyCode))

			jsonPathResponse := config.JSONPathResponse{
				Code:    cfg.DenyCode,
				Message: cfg.DenyMessage,
				Data:    map[string]interface{}{},
			}
			jsonPathResponseJson, _ := json.Marshal(jsonPathResponse)
			ctx.SetUserAttribute("deny_message", []byte(jsonPathResponseJson))
			// 设置标志，表示响应已在请求阶段发送，响应阶段的回调应该跳过处理
			ctx.SetUserAttribute("response_sent_in_request", "true")
			wlog.LogWithLine("[%s] onHttpRequestBody JSONPath:%b deny() called: deny_message=%s", pluginName, pluginCtx.RequestDenyModifyType, cfg.DenyMessage)

			return lib.DenyHandler(ctx, pluginCtx)
		}
		if modified {
			pluginCtx.IsModified = true
			pluginCtx.RequestDenyModifyType = config.DenyModifyTypeJSONPath
			pluginCtx.IsRequestModified = true
			proxywasm.ReplaceHttpRequestBody(body)
		}
	}
	// 如果配置了Raw拒绝，则处理Raw请求
	if cfg.DenyRaw {
		var modified bool
		var denied bool
		modified, denied = lib.ProcessRawRequest(ctx, pluginCtx, body)
		if denied {
			pluginCtx.IsDeny = true
			pluginCtx.IsRequestDeny = true
			pluginCtx.RequestDenyModifyType = config.DenyModifyTypeRaw
			ctx.SetUserAttribute("x-ai-data-masking", string(pluginCtx.RequestDenyModifyType))
			ctx.SetUserAttribute("deny_step", pluginCtx.Step.String())
			ctx.SetUserAttribute("deny_code", fmt.Sprintf("%d", cfg.DenyCode))
			rawResponse := config.RawResponse{
				Code:    cfg.DenyCode,
				Message: cfg.DenyMessage,
				Data:    map[string]interface{}{},
			}
			rawResponseJson, _ := json.Marshal(rawResponse)
			ctx.SetUserAttribute("deny_message", []byte(rawResponseJson))
			// 设置标志，表示响应已在请求阶段发送，响应阶段的回调应该跳过处理
			ctx.SetUserAttribute("response_sent_in_request", "true")
			wlog.LogWithLine("[%s] onHttpRequestBody Raw:%b deny() called: deny_message=%s", pluginName, pluginCtx.RequestDenyModifyType, cfg.DenyMessage)
			return lib.DenyHandler(ctx, pluginCtx)
		}
		if modified {
			pluginCtx.IsModified = true
			pluginCtx.RequestDenyModifyType = config.DenyModifyTypeRaw
			pluginCtx.IsRequestModified = true
			proxywasm.ReplaceHttpRequestBody(body)
		}
	}
	// 同步处理完成，继续传递请求到下游
	return types.ActionContinue
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, cfg config.AiDataMaskingConfig) types.Action {
	pluginCtx := getOrCreatePluginContext(ctx, &cfg)
	pluginCtx.Step = config.StepRespHeader

	wlog.LogWithLine("[%s] Process Step: %s", pluginName, pluginCtx.Step.String())
	// 检查响应是否来自上游（如果是在请求阶段通过 SendHttpResponse 发送的，则不是来自上游）
	if !wrapper.IsResponseFromUpstream() {
		// 响应不是来自上游（可能是我们在请求阶段发送的），直接跳过处理
		wlog.LogWithLine("[%s] onHttpResponseHeaders: response not from upstream, skipping processing", pluginName)
		ctx.DontReadResponseBody()
		return types.ActionContinue
	}

	// 检查是否在请求阶段已经发送了响应
	if responseSent, ok := ctx.GetUserAttribute("response_sent_in_request").(string); ok && responseSent == "true" {
		wlog.LogWithLine("[%s] onHttpResponseHeaders: response already sent in request phase, skipping processing", pluginName)
		ctx.DontReadResponseBody()
		return types.ActionContinue
	}

	// 检查响应头，判断是否为流式响应
	transferEncoding, _ := proxywasm.GetHttpResponseHeader("transfer-encoding")
	contentType, _ := proxywasm.GetHttpResponseHeader("content-type")

	// Envoy 会根据以下条件判断是否为流式响应：
	// 1. Transfer-Encoding: chunked 存在
	// 2. Content-Length 不存在或为 0
	// 3. Content-Type 为 text/event-stream (SSE)
	isChunked := transferEncoding == "chunked"
	isSSE := strings.Contains(contentType, "text/event-stream")
	isStreaming := isSSE

	// 设置流式响应标志，供 onHttpStreamingResponseBody 使用
	ctx.SetUserAttribute("is_streaming_response", fmt.Sprintf("%v", isStreaming))
	pluginCtx.RespIsSSE = isSSE

	wlog.LogWithLine("[%s] onHttpResponseHeaders: Transfer-Encoding=%s, Content-Type=%s, isChunked=%v, isSSE=%v, isStreaming=%v",
		pluginName, transferEncoding, contentType, isChunked, isSSE, isStreaming)

	// 如果不是流式响应，需要缓冲响应体，这样 wrapper 会调用 onHttpResponseBody 而不是 onHttpStreamingResponseBody
	if !isStreaming {
		ctx.BufferResponseBody() //防止直接进入onHttpStreamingResponseBody
		ctx.SetResponseBodyBufferLimit(config.DEFAULT_MAX_BODY_BYTES)
	}

	// 停止继续处理响应头，停止往onHttpResponseBody 传递响应头，onHttpResponseBody 可能会修改响应头
	return types.HeaderStopIteration
}

func onHttpResponseBody(ctx wrapper.HttpContext, cfg config.AiDataMaskingConfig, body []byte) types.Action {
	pluginCtx := getOrCreatePluginContext(ctx, &cfg)
	pluginCtx.Step = config.StepRespBody
	ctx.SetResponseBodyBufferLimit(config.DEFAULT_MAX_BODY_BYTES)
	wlog.LogWithLine("[%s] Process Step: %s", pluginName, pluginCtx.Step.String())
	// 检查响应是否来自上游（如果是在请求阶段通过 SendHttpResponse 发送的，则不是来自上游）
	if !wrapper.IsResponseFromUpstream() {
		// 响应不是来自上游（可能是我们在请求阶段发送的），直接跳过处理
		wlog.LogWithLine("[%s] onHttpResponseBody: response not from upstream, skipping processing", pluginName)
		return types.ActionContinue
	}

	// 检查是否在请求阶段已经发送了响应
	if responseSent, ok := ctx.GetUserAttribute("response_sent_in_request").(string); ok && responseSent == "true" {
		wlog.LogWithLine("[%s] onHttpResponseBody: response already sent in request phase, skipping processing", pluginName)
		return types.ActionContinue
	}

	return processNonStreamResponse(ctx, cfg, body)
}

// processNonStreamResponse 处理非流式响应
func processNonStreamResponse(ctx wrapper.HttpContext, cfg config.AiDataMaskingConfig, body []byte) types.Action {
	pluginCtx := getOrCreatePluginContext(ctx, &cfg)
	bodyStr := string(body)
	wlog.LogWithLine("[%s] processNonStreamResponse: body length=%d, RequestDenyType=%v, RespIsSSE=%v, DenyOpenAI=%v, DenyRaw=%v",
		pluginName, len(body), pluginCtx.RequestDenyModifyType, pluginCtx.RespIsSSE, pluginCtx.Config.DenyOpenAI, pluginCtx.Config.DenyRaw)

	// 先处理 OpenAI JSON 响应（如果启用）,并且请求阶段是openai格式
	if pluginCtx.Config.DenyOpenAI && pluginCtx.OpenAIRequest != nil {
		wlog.LogWithLine("[%s] processNonStreamResponse: processing OpenAI response", pluginName)
		modified, denied := lib.ProcessOpenAIResponse(ctx, pluginCtx, bodyStr, body)

		if denied {
			pluginCtx.IsDeny = true
			pluginCtx.IsResponseDeny = true
			pluginCtx.ResponseDenyModifyType = config.DenyModifyTypeOpenAI
			ctx.SetUserAttribute("x-ai-data-masking", string(pluginCtx.ResponseDenyModifyType))
			ctx.SetUserAttribute("deny_step", pluginCtx.Step.String())
			ctx.SetUserAttribute("deny_code", fmt.Sprintf("%d", pluginCtx.Config.DenyCode))

			// 非流式响应
			openaiResponse := config.OpenAICompletionResponse{
				Id:      uuid.New().String(),
				Object:  "chat.completion",
				Created: 123,
				Model:   pluginCtx.OpenAIRequest.Model,
				Choices: []config.OpenAICompletionChoice{
					{
						Index: 0,
						Message: &config.OpenAIMessage{
							Role:    "assistant",
							Content: cfg.DenyMessage,
						},
					},
				},
				Usage: &config.OpenAIUsage{
					PromptTokens:     0,
					CompletionTokens: 0,
					TotalTokens:      0,
				},
			}
			openaiResponseJson, _ := json.Marshal(openaiResponse)
			ctx.SetUserAttribute("deny_message", openaiResponseJson)

			wlog.LogWithLine("[%s] processNonStreamResponse: OpenAI Response Denied, denied=%v", pluginName, denied)

			return lib.DenyHandler(ctx, pluginCtx)
		}
		if modified {
			pluginCtx.IsModified = true
			pluginCtx.ResponseDenyModifyType = config.DenyModifyTypeOpenAI
			pluginCtx.IsResponseModified = true
			proxywasm.ReplaceHttpResponseBody(body)
		}
	}

	// // 再处理 Raw 响应体（如果启用）
	// if pluginCtx.Config.DenyRaw {
	// 	wlog.LogWithLine("[%s] processNonStreamResponse: processing Raw response", pluginName)
	// 	action := lib.ProcessRawResponse(ctx, pluginCtx, bodyStr)
	// 	if action != types.ActionContinue {
	// 		wlog.LogWithLine("[%s] processNonStreamResponse: Raw Response Denied, action=%v", pluginName, action)
	// 		return action
	// 	}
	// 	wlog.LogWithLine("[%s] processNonStreamResponse: Raw response processed, continuing", pluginName)
	// }

	// wlog.LogWithLine("[%s] processNonStreamResponse: all checks passed, returning ActionContinue", pluginName)
	return types.ActionContinue
}

func onHttpStreamingResponseBody(ctx wrapper.HttpContext, cfg config.AiDataMaskingConfig, chunk []byte, isLastChunk bool) []byte {
	pluginCtx := getOrCreatePluginContext(ctx, &cfg)
	pluginCtx.Step = config.StepStreamRespBody
	// wlog.LogWithLine("[%s] Process Step: %s", pluginName, pluginCtx.Step.String())

	// 如果已经检测到敏感词并拒绝，后续的chunk直接返回 [DONE] 或空，不再处理
	if pluginCtx.StreamDenied {
		// wlog.LogWithLine("[%s] onHttpStreamingResponseBody: stream already denied, returning [DONE]", pluginName)
		// 如果是最后一个chunk，返回 [DONE]，否则返回空（丢弃后续chunk）
		if isLastChunk {
			// return []byte("data: [DONE]\n\n")
			return nil
		}
		return nil
	}

	// 先处理 OpenAI JSON 响应（如果启用）,并且请求阶段是openai格式
	if pluginCtx.Config.DenyOpenAI && pluginCtx.OpenAIRequest != nil {

		processedChunk, denied := lib.ProcessOpenAIStreamResponse(ctx, pluginCtx, chunk, isLastChunk)
		if denied {
			// 检测到敏感词，标记为拒绝并返回截断的响应
			pluginCtx.IsDeny = true
			pluginCtx.IsResponseDeny = true
			pluginCtx.ResponseDenyModifyType = config.DenyModifyTypeOpenAI
			// 返回截断的响应（包含拒绝消息和 [DONE]）
			if processedChunk != nil {
				wlog.LogWithLine("[%s] onHttpStreamingResponseBody: processing OpenAI response,  processedChunk=%s", pluginName, string(processedChunk))
				return processedChunk
			}
			// // 如果没有返回chunk，返回 [DONE] 结束流
			// return []byte("data: [DONE]\n\n")
		}
		// 没有 deny，返回处理后的 chunk（可能是原样或修改后的）
		if processedChunk != nil {
			wlog.LogWithLine("[%s] onHttpStreamingResponseBody: processing OpenAI response, processedChunk=%s", pluginName, string(processedChunk))
			return processedChunk
		}
	}
	return []byte(": HIGRESS AI DATA PROCESSING \n\n")
}
