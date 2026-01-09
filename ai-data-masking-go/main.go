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
	"embed"
	"encoding/json"
	"fmt"
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

//go:embed resources/*
var resourcesFS embed.FS

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
	// 解析基础配置
	cfg.SystemDeny = json.Get("system_deny").Bool()

	// 解析 deny_code
	if json.Get("deny_code").Exists() {
		cfg.DenyCode = uint32(json.Get("deny_code").Int())
	} else {
		cfg.DenyCode = 200 // 默认值
	}

	// 解析 deny_content_type
	cfg.DenyContentType = json.Get("deny_content_type").String()
	if cfg.DenyContentType == "" {
		cfg.DenyContentType = "application/json"
	}

	// 解析 deny_message
	cfg.DenyMessage = json.Get("deny_message").String()
	if cfg.DenyMessage == "" {
		cfg.DenyMessage = "提问或回答中包含敏感词，已被屏蔽"
	}

	// 解析 deny_words
	for _, item := range json.Get("deny_words").Array() {
		word := strings.TrimSpace(item.String())
		if word != "" {
			cfg.DenyWords = append(cfg.DenyWords, word)
		}
	}

	// 解析 deny_plot
	denyPlotJson := json.Get("deny_plot")
	cfg.DenyPlot.Plot = denyPlotJson.Get("plot").String()
	if cfg.DenyPlot.Plot == "" {
		cfg.DenyPlot.Plot = "stop" // 默认值
	}
	cfg.DenyPlot.Value = denyPlotJson.Get("value").String()

	// 解析 request_deny
	if json.Get("request_deny").Exists() {
		cfg.RequestDeny = json.Get("request_deny").Bool()
	} else {
		cfg.RequestDeny = true // 默认开启请求拦截
	}

	// 解析 response_deny
	if json.Get("response_deny").Exists() {
		cfg.ResponseDeny = json.Get("response_deny").Bool()
	} else {
		cfg.ResponseDeny = true // 默认开启响应拦截
	}

	// 解析 match_format
	matchFormatJson := json.Get("match_format")
	cfg.MatchFormat.Type = matchFormatJson.Get("type").String()
	if cfg.MatchFormat.Type == "" {
		cfg.MatchFormat.Type = "openai" // 默认值
	}

	// 解析 request_deny_jsonpath
	for _, item := range matchFormatJson.Get("request_deny_jsonpath").Array() {
		path := item.String()
		if path != "" {
			cfg.MatchFormat.RequestDenyJSONPath = append(cfg.MatchFormat.RequestDenyJSONPath, path)
		}
	}

	// 解析 response_deny_jsonpath
	for _, item := range matchFormatJson.Get("response_deny_jsonpath").Array() {
		path := item.String()
		if path != "" {
			cfg.MatchFormat.ResponseDenyJSONPath = append(cfg.MatchFormat.ResponseDenyJSONPath, path)
		}
	}

	if cfg.SystemDeny {
		// 从资源文件加载系统敏感词
		systemWords, err := config.LoadSystemDenyWords(resourcesFS)
		if err != nil {
			wlog.LogWithLine("[%s] Failed to load system deny words: %v", pluginName, err)
		} else {
			config.SystemDenyWords = systemWords
			wlog.LogWithLine("[%s] Loaded %d system deny words from resources", pluginName, len(config.SystemDenyWords))
		}

		// 初始化系统敏感词批次（在启动时初始化，避免重复初始化）
		lib.InitSystemDenyWordsBatches(config.SystemDenyWords)

		// 计算最长敏感词长度（在启动时计算，避免流处理时重复计算）
		config.MaxSensitiveWordLength = lib.CalculateMaxSensitiveWordLength(cfg)
	}
	MaxBufferChunkCount := json.Get("max_buffer_chunk_count").Uint()
	if MaxBufferChunkCount == 0 {
		cfg.MaxBufferChunkCount = config.DefaultMaxBufferChunkCount
	} else {
		cfg.MaxBufferChunkCount = uint32(MaxBufferChunkCount)
	}

	MaxStreamChunkBufferLen := json.Get("max_stream_chunk_buffer_len").Uint()
	if MaxStreamChunkBufferLen == 0 {
		cfg.MaxStreamChunkBufferLen = config.DefaultMaxStreamChunkBufferLen
	} else {
		cfg.MaxStreamChunkBufferLen = uint32(MaxStreamChunkBufferLen)
	}
	// 打印所有配置的 JSON（使用 gjson 的 Raw 字段获取原始 JSON）

	wlog.LogWithLine("[%s] Configuration:\n%s", pluginName, string(lib.PrintConfig(cfg)))
	wlog.LogWithLine("[%s] 最大返回敏感词重叠边界长度: %d", pluginName, config.MaxSensitiveWordLength)
	wlog.LogWithLine("[%s] 最长返回敏感词检测chunk个数: %d", pluginName, cfg.MaxBufferChunkCount)
	wlog.LogWithLine("[%s] 最长返回敏感词检测chunk大小: %d", pluginName, cfg.MaxStreamChunkBufferLen)

	return nil
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

	// 检查是否开启请求拦截
	if !cfg.RequestDeny {
		wlog.LogWithLine("[%s] Request deny is disabled, skipping request body processing", pluginName)
		return types.ActionContinue
	}

	// 根据 match_format.type 处理不同格式的请求
	var modified bool
	var denied bool

	switch cfg.MatchFormat.Type {
	case "openai":
		// 处理 OpenAI 格式
		modified, denied = lib.ProcessOpenAIRequest(ctx, pluginCtx, body)
		if denied {
			pluginCtx.RequestDenyModifyType = config.DenyModifyTypeOpenAI
			return handleRequestDeny(ctx, pluginCtx, &cfg)
		}
		if modified {
			pluginCtx.IsModified = true
			pluginCtx.RequestDenyModifyType = config.DenyModifyTypeOpenAI
			proxywasm.ReplaceHttpRequestBody(body)
		}

	case "anthropic":
		// TODO: 处理 Anthropic 格式（暂时使用 OpenAI 格式处理）
		wlog.LogWithLine("[%s] Anthropic format not fully implemented, using OpenAI format", pluginName)
		modified, denied = lib.ProcessOpenAIRequest(ctx, pluginCtx, body)
		if denied {
			pluginCtx.RequestDenyModifyType = config.DenyModifyTypeOpenAI
			return handleRequestDeny(ctx, pluginCtx, &cfg)
		}
		if modified {
			pluginCtx.IsModified = true
			pluginCtx.RequestDenyModifyType = config.DenyModifyTypeOpenAI
			proxywasm.ReplaceHttpRequestBody(body)
		}

	case "custom":
		// 自定义格式：根据 request_deny_jsonpath 处理
		if len(cfg.MatchFormat.RequestDenyJSONPath) > 0 {
			// 使用自定义的 JSONPath 进行处理
			modified, denied = lib.ProcessCustomJSONPathRequest(ctx, pluginCtx, body, cfg.MatchFormat.RequestDenyJSONPath)
			if denied {
				pluginCtx.RequestDenyModifyType = config.DenyModifyTypeJSONPath
				return handleRequestDeny(ctx, pluginCtx, &cfg)
			}
			if modified {
				pluginCtx.IsModified = true
				pluginCtx.RequestDenyModifyType = config.DenyModifyTypeJSONPath
				proxywasm.ReplaceHttpRequestBody(body)
			}
		}

	default:
		wlog.LogWithLine("[%s] Unknown match_format.type: %s, using openai as default", pluginName, cfg.MatchFormat.Type)
		modified, denied = lib.ProcessOpenAIRequest(ctx, pluginCtx, body)
		if denied {
			pluginCtx.RequestDenyModifyType = config.DenyModifyTypeOpenAI
			return handleRequestDeny(ctx, pluginCtx, &cfg)
		}
		if modified {
			pluginCtx.IsModified = true
			pluginCtx.RequestDenyModifyType = config.DenyModifyTypeOpenAI
			proxywasm.ReplaceHttpRequestBody(body)
		}
	}

	return types.ActionContinue
}

// handleRequestDeny 处理请求阶段的拒绝
func handleRequestDeny(ctx wrapper.HttpContext, pluginCtx *config.PluginContext, cfg *config.AiDataMaskingConfig) types.Action {
	pluginCtx.IsDeny = true
	pluginCtx.IsRequestDeny = true

	ctx.SetUserAttribute("x-ai-data-masking", string(pluginCtx.RequestDenyModifyType))
	ctx.SetUserAttribute("deny_step", pluginCtx.Step.String())
	ctx.SetUserAttribute("deny_code", fmt.Sprintf("%d", cfg.DenyCode))
	ctx.SetUserAttribute("response_sent_in_request", "true")

	var denyMessageBytes []byte

	switch pluginCtx.RequestDenyModifyType {
	case config.DenyModifyTypeOpenAI:
		// 根据是否为流式请求构造不同的响应格式
		if pluginCtx.OpenAIRequest != nil && pluginCtx.OpenAIRequest.Stream {
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
			denyMessageBytes = []byte(fmt.Sprintf("data: %s\n\ndata: [DONE]\n\n", string(streamJson)))
		} else {
			// 非流式响应
			openaiResponse := config.OpenAICompletionResponse{
				Id:      uuid.New().String(),
				Object:  "chat.completion",
				Created: 123,
				Model: func() string {
					if pluginCtx.OpenAIRequest != nil {
						return pluginCtx.OpenAIRequest.Model
					}
					return "unknown"
				}(),
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
			denyMessageBytes, _ = json.Marshal(openaiResponse)
		}

	case config.DenyModifyTypeJSONPath:
		jsonPathResponse := config.JSONPathResponse{
			Code:    cfg.DenyCode,
			Message: cfg.DenyMessage,
			Data:    map[string]interface{}{},
		}
		denyMessageBytes, _ = json.Marshal(jsonPathResponse)

	default:
		denyMessageBytes = []byte(cfg.DenyMessage)
	}

	ctx.SetUserAttribute("deny_message", denyMessageBytes)

	wlog.LogWithLine("[%s] onHttpRequestBody DenyModifyType:%s deny() called: deny_message=%s",
		pluginName, pluginCtx.RequestDenyModifyType, cfg.DenyMessage)

	return lib.DenyHandler(ctx, pluginCtx)
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

	// 检查是否开启响应拦截
	if !cfg.ResponseDeny {
		wlog.LogWithLine("[%s] Response deny is disabled, skipping response body processing", pluginName)
		return types.ActionContinue
	}

	return processNonStreamResponse(ctx, cfg, body)
}

// processNonStreamResponse 处理非流式响应
func processNonStreamResponse(ctx wrapper.HttpContext, cfg config.AiDataMaskingConfig, body []byte) types.Action {
	pluginCtx := getOrCreatePluginContext(ctx, &cfg)
	bodyStr := string(body)
	wlog.LogWithLine("[%s] processNonStreamResponse: body length=%d, MatchFormat.Type=%v, RespIsSSE=%v",
		pluginName, len(body), pluginCtx.Config.MatchFormat.Type, pluginCtx.RespIsSSE)

	var modified, denied bool

	// 根据 match_format.type 处理不同格式的响应
	switch cfg.MatchFormat.Type {
	case "openai":
		// 处理 OpenAI 格式响应
		if pluginCtx.OpenAIRequest != nil {
			wlog.LogWithLine("[%s] processNonStreamResponse: processing OpenAI response", pluginName)
			modified, denied = lib.ProcessOpenAIResponse(ctx, pluginCtx, bodyStr, body)

			if denied {
				pluginCtx.ResponseDenyModifyType = config.DenyModifyTypeOpenAI
				return handleResponseDenyNonStream(ctx, pluginCtx, &cfg, bodyStr)
			}
			if modified {
				pluginCtx.IsModified = true
				pluginCtx.ResponseDenyModifyType = config.DenyModifyTypeOpenAI
				proxywasm.ReplaceHttpResponseBody(body)
			}
		}

	case "anthropic":
		// TODO: 处理 Anthropic 格式响应（暂时使用 OpenAI 格式处理）
		wlog.LogWithLine("[%s] Anthropic format not fully implemented, using OpenAI format", pluginName)
		if pluginCtx.OpenAIRequest != nil {
			modified, denied = lib.ProcessOpenAIResponse(ctx, pluginCtx, bodyStr, body)

			if denied {
				pluginCtx.ResponseDenyModifyType = config.DenyModifyTypeOpenAI
				return handleResponseDenyNonStream(ctx, pluginCtx, &cfg, bodyStr)
			}
			if modified {
				pluginCtx.IsModified = true
				pluginCtx.ResponseDenyModifyType = config.DenyModifyTypeOpenAI
				proxywasm.ReplaceHttpResponseBody(body)
			}
		}

	case "custom":
		// 自定义格式：根据 response_deny_jsonpath 处理
		if len(cfg.MatchFormat.ResponseDenyJSONPath) > 0 {
			// 使用自定义的 JSONPath 进行处理
			modified, denied = lib.ProcessCustomJSONPathResponse(ctx, pluginCtx, bodyStr, cfg.MatchFormat.ResponseDenyJSONPath)

			if denied {
				pluginCtx.ResponseDenyModifyType = config.DenyModifyTypeJSONPath
				return handleResponseDenyNonStream(ctx, pluginCtx, &cfg, bodyStr)
			}
			if modified {
				pluginCtx.IsModified = true
				pluginCtx.ResponseDenyModifyType = config.DenyModifyTypeJSONPath
				proxywasm.ReplaceHttpResponseBody(body)
			}
		}

	default:
		wlog.LogWithLine("[%s] Unknown match_format.type: %s, using openai as default", pluginName, cfg.MatchFormat.Type)
		if pluginCtx.OpenAIRequest != nil {
			modified, denied = lib.ProcessOpenAIResponse(ctx, pluginCtx, bodyStr, body)

			if denied {
				pluginCtx.ResponseDenyModifyType = config.DenyModifyTypeOpenAI
				return handleResponseDenyNonStream(ctx, pluginCtx, &cfg, bodyStr)
			}
			if modified {
				pluginCtx.IsModified = true
				pluginCtx.ResponseDenyModifyType = config.DenyModifyTypeOpenAI
				proxywasm.ReplaceHttpResponseBody(body)
			}
		}
	}

	wlog.LogWithLine("[%s] processNonStreamResponse: all checks passed, returning ActionContinue", pluginName)
	return types.ActionContinue
}

// handleResponseDenyNonStream 处理非流式响应的拒绝
func handleResponseDenyNonStream(ctx wrapper.HttpContext, pluginCtx *config.PluginContext, cfg *config.AiDataMaskingConfig, bodyStr string) types.Action {
	// 根据拒绝策略处理
	denyPlot := cfg.DenyPlot.Plot
	if denyPlot == "" {
		denyPlot = "stop" // 默认值
	}

	// 设置 deny 相关标志和属性
	pluginCtx.IsDeny = true
	pluginCtx.IsResponseDeny = true

	ctx.SetUserAttribute("x-ai-data-masking", string(pluginCtx.ResponseDenyModifyType))
	ctx.SetUserAttribute("deny_step", pluginCtx.Step.String())
	ctx.SetUserAttribute("deny_code", fmt.Sprintf("%d", cfg.DenyCode))
	ctx.SetUserAttribute("deny_plot", denyPlot)

	if denyPlot == "replace" {
		wlog.LogWithLine("[%s] handleResponseDenyNonStream: replaced sensitive words with value, continuing", pluginName)
		return lib.DenyHandlerResponseReplaceNonStream(ctx, pluginCtx, bodyStr)
	}

	// stop 策略：返回拒绝消息（默认行为）
	switch pluginCtx.ResponseDenyModifyType {
	case config.DenyModifyTypeOpenAI:
		openaiResponse := config.OpenAICompletionResponse{
			Id:      uuid.New().String(),
			Object:  "chat.completion",
			Created: 123,
			Model: func() string {
				if pluginCtx.OpenAIRequest != nil {
					return pluginCtx.OpenAIRequest.Model
				}
				return "unknown"
			}(),
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

	case config.DenyModifyTypeJSONPath:
		jsonPathResponse := config.JSONPathResponse{
			Code:    cfg.DenyCode,
			Message: cfg.DenyMessage,
			Data:    map[string]interface{}{},
		}
		jsonPathResponseJson, _ := json.Marshal(jsonPathResponse)
		ctx.SetUserAttribute("deny_message", jsonPathResponseJson)

	default:
		ctx.SetUserAttribute("deny_message", []byte(cfg.DenyMessage))
	}

	wlog.LogWithLine("[%s] handleResponseDenyNonStream: Response Denied (stop strategy)", pluginName)
	return lib.DenyHandler(ctx, pluginCtx)
}

func onHttpStreamingResponseBody(ctx wrapper.HttpContext, cfg config.AiDataMaskingConfig, chunk []byte, isLastChunk bool) []byte {
	pluginCtx := getOrCreatePluginContext(ctx, &cfg)
	pluginCtx.Step = config.StepStreamRespBody
	// wlog.LogWithLine("[%s] Process Step: %s", pluginName, pluginCtx.Step.String())

	// 检查是否开启响应拦截
	if !cfg.ResponseDeny {
		// wlog.LogWithLine("[%s] Response deny is disabled, skipping streaming response body processing", pluginName)
		return chunk
	}

	// 根据拒绝策略处理
	denyPlot := cfg.DenyPlot.Plot
	if denyPlot == "" {
		denyPlot = "stop" // 默认值
	}
	if denyPlot == "" {
		denyPlot = "stop" // 默认值
	}
	if denyPlot == "rollback" && (cfg.MatchFormat.Type == "openai" || cfg.MatchFormat.Type == "anthropic") && pluginCtx.OpenAIRequest != nil {
		processedChunk := lib.ProcessOpenAIStreamRollbackResponse(ctx, pluginCtx, chunk, isLastChunk)
		if processedChunk != nil {
			wlog.LogWithLine("[%s] onHttpStreamingResponseBody: processing OpenAI rollback response, chunk:%s, processedChunk:%s",
				pluginName, string(chunk), string(processedChunk))
			return processedChunk
		}
		return chunk
	}

	if denyPlot == "replace" && (cfg.MatchFormat.Type == "openai" || cfg.MatchFormat.Type == "anthropic") && pluginCtx.OpenAIRequest != nil {
		processedChunk := lib.ProcessOpenAIStreamReplaceResponse(ctx, pluginCtx, chunk, isLastChunk)
		wlog.LogWithLine("[%s] onHttpStreamingResponseBody: processing OpenAI response, chunk:%s, processedChunk:%s",
			pluginName, string(chunk), string(processedChunk))
		return processedChunk
	}

	if denyPlot == "stop" {
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

		// 处理 OpenAI/Anthropic 格式的流式响应
		if (cfg.MatchFormat.Type == "openai" || cfg.MatchFormat.Type == "anthropic") && pluginCtx.OpenAIRequest != nil {
			processedChunk, denied := lib.ProcessOpenAIStreamDenyResponse(ctx, pluginCtx, chunk, isLastChunk)
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
			}
			// 没有 deny，返回处理后的 chunk（可能是原样或修改后的）
			if processedChunk != nil {
				wlog.LogWithLine("[%s] onHttpStreamingResponseBody: processing OpenAI response, processedChunk=%s", pluginName, string(processedChunk))
				return processedChunk
			}
		}
	}
	return []byte(": HIGRESS AI DATA PROCESSING \n\n")
}
