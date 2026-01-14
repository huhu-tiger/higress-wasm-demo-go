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

package main

import (
	"encoding/json"
	"fmt"
	"transformer-vnet/bussiness/t2i"
	"transformer-vnet/config"
	"transformer-vnet/wlog"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

func main() {}

func init() {
	wrapper.SetCtx(
		config.PluginName,
		wrapper.ParseConfig(parseConfig),
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
		wrapper.ProcessRequestBody(onHttpRequestBody),
		wrapper.ProcessResponseHeaders(onHttpResponseHeaders),
		wrapper.ProcessResponseBody(onHttpResponseBody),
		wrapper.ProcessStreamingResponseBody(onHttpStreamingResponseBody),
	)
}

// Config 插件配置
type Config struct {
	// 可以在这里添加配置项
}

func parseConfig(json gjson.Result, cfg *Config) error {
	// 解析配置
	// 注意：此函数会被 Envoy 的多个工作线程调用，因此不在此处打印日志以避免重复
	return nil
}

// getOrCreatePluginContext 获取或创建插件上下文
func getOrCreatePluginContext(ctx wrapper.HttpContext) *PluginContext {
	contextKey := config.PluginName + "_context"
	value := ctx.GetContext(contextKey)
	if value != nil {
		if pluginCtx, ok := value.(*PluginContext); ok {
			return pluginCtx
		}
	}

	pluginCtx := &PluginContext{
		Model:            "",
		ModelType:        "",
		TransformFail:    "",
		StatusCode:       "",
		IsErrorStatus:    false,
		HasStreamRequest: false,
	}
	ctx.SetContext(contextKey, pluginCtx)
	return pluginCtx
}

// PluginContext 插件上下文
type PluginContext struct {
	Model            string // 模型名称，例如: volcengine/volcengine/doubao-seedream-4.5
	ModelType        string // 模型类型，例如: t2i (文生图), i2i (图生图)
	TransformFail    string // 转换失败的阶段: "request" 或 "response"，空字符串表示没有失败
	StatusCode       string // 响应状态码
	IsErrorStatus    bool   // 是否为错误状态码（非 2xx）
	HasStreamRequest bool   // 请求中是否有 stream 参数（doubao 等模型使用）
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, cfg Config) types.Action {
	// 禁用重路由
	ctx.DisableReroute()

	// 移除 Accept-Encoding 头部，防止上游返回压缩响应
	// 如果响应有 content-encoding，wrapper 会认为是二进制响应，跳过 onHttpResponseBody
	proxywasm.RemoveHttpRequestHeader("accept-encoding")
	wlog.LogWithLine("[%s] Removed accept-encoding header to prevent compressed response", config.PluginName)

	pluginCtx := getOrCreatePluginContext(ctx)

	// 读取请求头中的 model 和 model_type
	model, err := proxywasm.GetHttpRequestHeader("model")
	if err == nil && model != "" {
		pluginCtx.Model = model
		wlog.LogWithLine("[%s] Model from header: %s", config.PluginName, model)
	}

	modelType, err := proxywasm.GetHttpRequestHeader("model_type")
	if err == nil && modelType != "" {
		pluginCtx.ModelType = modelType
		wlog.LogWithLine("[%s] Model type from header: %s", config.PluginName, modelType)
	}

	// 检查是否有请求体
	contentLength, err := proxywasm.GetHttpRequestHeader("content-length")
	if err == nil && contentLength != "0" && contentLength != "" {
		// 移除 Content-Length，让 Envoy 重新计算
		proxywasm.RemoveHttpRequestHeader("content-length")
		return types.ActionContinue
	}

	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, cfg Config, body []byte) types.Action {
	pluginCtx := getOrCreatePluginContext(ctx)

	// 设置请求体缓冲区限制
	ctx.SetRequestBodyBufferLimit(config.DefaultMaxRequestBodyBytes)

	if len(body) == 0 {
		wlog.LogWithLine("[%s] Request body is empty, skipping transformation", config.PluginName)
		return types.ActionContinue
	}

	// 根据 model 和 model_type 进行转换
	if pluginCtx.Model == "" || pluginCtx.ModelType == "" {
		wlog.LogWithLine("[%s] Model or ModelType is empty, skipping transformation", config.PluginName)
		return types.ActionContinue
	}

	// 打印原始传入的请求参数
	wlog.LogWithLine("[%s] ========== Request Transformation ==========", config.PluginName)
	wlog.LogWithLine("[%s] Model: %s, ModelType: %s", config.PluginName, pluginCtx.Model, pluginCtx.ModelType)
	wlog.LogWithLine("[%s] [原始传入的请求参数] Original Request Body:", config.PluginName)
	printJSON(config.PluginName, body, "  ")

	var transformedBody []byte
	var err error
	if pluginCtx.ModelType == "t2i" {
		// 检查请求体中是否有 stream 参数（用于判断是否为流式响应）
		if t2i.CheckStreamParameter(pluginCtx.Model, pluginCtx.ModelType, body) {
			pluginCtx.HasStreamRequest = true
			wlog.LogWithLine("[%s] Found stream=true in request body (model: %s)", config.PluginName, pluginCtx.Model)
		}
		// 转换请求体
		transformedBody, err = t2i.TransformRequest(pluginCtx.Model, pluginCtx.ModelType, body)
		if err != nil {
			wlog.LogWithLine("[%s] Failed to transform request: %v", config.PluginName, err)
			// 记录转换失败的阶段
			pluginCtx.TransformFail = "request"
			wlog.LogWithLine("[%s] ============================================", config.PluginName)
			return types.ActionContinue
		}
	}

	if transformedBody != nil {
		// 打印转换后的请求参数
		wlog.LogWithLine("[%s] [转换后的请求参数] Transformed Request Body:", config.PluginName)
		printJSON(config.PluginName, transformedBody, "  ")
		wlog.LogWithLine("[%s] Request transformed successfully", config.PluginName)
		proxywasm.ReplaceHttpRequestBody(transformedBody)
	} else {
		wlog.LogWithLine("[%s] [无需转换] No transformation needed, request body unchanged", config.PluginName)
		wlog.LogWithLine("[%s] [转换后的请求参数] Transformed Request Body (same as original):", config.PluginName)
		printJSON(config.PluginName, body, "  ")
	}
	wlog.LogWithLine("[%s] ============================================", config.PluginName)

	return types.ActionContinue
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, cfg Config) types.Action {
	pluginCtx := getOrCreatePluginContext(ctx)

	// 获取响应状态码
	statusCode, _ := proxywasm.GetHttpResponseHeader(":status")
	pluginCtx.StatusCode = statusCode

	// 检查是否为错误状态码（非 2xx）
	isErrorStatus := false
	if statusCode != "" {
		// 解析状态码
		if len(statusCode) >= 3 {
			firstDigit := statusCode[0]
			if firstDigit != '2' {
				isErrorStatus = true
			}
		}
	}
	pluginCtx.IsErrorStatus = isErrorStatus

	// 检查响应是否来自上游
	isFromUpstream := wrapper.IsResponseFromUpstream()

	if !isFromUpstream {
		wlog.LogWithLine("[%s] Response not from upstream (status: %s)", config.PluginName, statusCode)
		// 响应不来自上游时，添加响应头以便客户端识别（因为不需要缓冲响应体）
		if pluginCtx.Model != "" {
			proxywasm.AddHttpResponseHeader("model", pluginCtx.Model)
			wlog.LogWithLine("[%s] Added response header: model=%s (not from upstream)", config.PluginName, pluginCtx.Model)
		}
		if pluginCtx.ModelType != "" {
			proxywasm.AddHttpResponseHeader("model_type", pluginCtx.ModelType)
			wlog.LogWithLine("[%s] Added response header: model_type=%s (not from upstream)", config.PluginName, pluginCtx.ModelType)
		}
		ctx.DontReadResponseBody()
		return types.ActionContinue
	}

	// 响应来自上游
	if isErrorStatus {
		wlog.LogWithLine("[%s] Response from upstream with error status: %s, will convert to unified error format", config.PluginName, statusCode)
	} else {
		wlog.LogWithLine("[%s] Response from upstream (status: %s), processing normally", config.PluginName, statusCode)
	}

	// 注意：不要在这里添加响应头！添加响应头会导致响应头被提交，BufferResponseBody 会失效
	// 对于来自上游的响应，所有响应头的添加都在 onHttpResponseBody 中进行

	// 判断是否为流式响应
	// 条件1: 响应 header 的 transfer-encoding 为 chunked
	// 条件2: 请求参数中是否有 stream 参数（doubao 有，如果没有此参数只判断条件1）
	contentType, _ := proxywasm.GetHttpResponseHeader("content-type")
	contentLength, _ := proxywasm.GetHttpResponseHeader("content-length")
	transferEncoding, _ := proxywasm.GetHttpResponseHeader("transfer-encoding")
	contentEncoding, _ := proxywasm.GetHttpResponseHeader("content-encoding")

	// 检查 content-encoding，如果存在会导致 wrapper 认为是二进制响应，不调用 onHttpResponseBody
	wlog.LogWithLine("[%s] Content-Encoding: '%s' (if not empty, wrapper will skip onHttpResponseBody!)", config.PluginName, contentEncoding)

	// 如果有 content-encoding，需要移除它，否则 wrapper 会跳过响应体处理
	if contentEncoding != "" {
		proxywasm.RemoveHttpResponseHeader("content-encoding")
		wlog.LogWithLine("[%s] Removed content-encoding header to enable response body processing", config.PluginName)
	}

	isChunked := transferEncoding == "chunked"
	hasStreamParam := pluginCtx.HasStreamRequest
	var isStreaming bool

	if pluginCtx.ModelType == "t2i" {
		// t2i 模块处理响应头（调用各模型文件中的封装方法，删除或修改响应头）
		t2i.ProcessResponseHeaders(pluginCtx.Model, pluginCtx.ModelType, transferEncoding, hasStreamParam)
		// 判断是否为流式响应（调用各模型文件中的封装方法）
		isStreaming = t2i.IsStreamingResponse(pluginCtx.Model, pluginCtx.ModelType, isChunked, hasStreamParam)
	}

	wlog.LogWithLine("[%s] Model: %s, isChunked=%v, hasStreamParam=%v, isStreaming=%v",
		config.PluginName, pluginCtx.Model, isChunked, hasStreamParam, isStreaming)

	wlog.LogWithLine("[%s] Content-Type: %s, Content-Length: %s, Transfer-Encoding: %s, isStreaming: %v",
		config.PluginName, contentType, contentLength, transferEncoding, isStreaming)

	// 检查 Content-Length，如果为空或为 0，可能是分块传输
	if contentLength == "" || contentLength == "0" {
		wlog.LogWithLine("[%s] Content-Length is empty or 0 (may be chunked transfer). Content-Length: '%s'", config.PluginName, contentLength)
	} else {
		wlog.LogWithLine("[%s] Content-Length is %s, response body should exist", config.PluginName, contentLength)
	}

	// 如果不是流式响应，需要缓冲响应体
	// BufferResponseBody 会将分块传输的响应体缓冲起来，然后一次性调用 onHttpResponseBody
	if !isStreaming {
		wlog.LogWithLine("[%s] Not streaming response, calling BufferResponseBody and SetResponseBodyBufferLimit(%d)", config.PluginName, config.DefaultMaxResponseBodyBytes)
		ctx.BufferResponseBody()
		ctx.SetResponseBodyBufferLimit(config.DefaultMaxResponseBodyBytes)
		wlog.LogWithLine("[%s] BufferResponseBody and SetResponseBodyBufferLimit called successfully", config.PluginName)
	} else {
		wlog.LogWithLine("[%s] Streaming response detected, skipping BufferResponseBody (response will be passed through)", config.PluginName)
		// 流式响应不需要缓冲，直接透传
		ctx.DontReadResponseBody()
	}

	wlog.LogWithLine("[%s] Response headers processed, Model: %s, ModelType: %s, returning HeaderStopIteration", config.PluginName, pluginCtx.Model, pluginCtx.ModelType)
	// 返回 HeaderStopIteration，停止继续处理响应头，等待响应体缓冲完成后调用 onHttpResponseBody
	return types.HeaderStopIteration
}

func onHttpResponseBody(ctx wrapper.HttpContext, cfg Config, body []byte) types.Action {
	pluginCtx := getOrCreatePluginContext(ctx)

	wlog.LogWithLine("[%s] ========== onHttpResponseBody called ==========", config.PluginName)
	wlog.LogWithLine("[%s] Body length: %d", config.PluginName, len(body))
	wlog.LogWithLine("[%s] IsResponseFromUpstream: %v", config.PluginName, wrapper.IsResponseFromUpstream())
	wlog.LogWithLine("[%s] Model: %s, ModelType: %s", config.PluginName, pluginCtx.Model, pluginCtx.ModelType)
	wlog.LogWithLine("[%s] IsErrorStatus: %v", config.PluginName, pluginCtx.IsErrorStatus)

	// 添加响应头（model, model_type 等）
	if pluginCtx.Model != "" {
		proxywasm.AddHttpResponseHeader("model", pluginCtx.Model)
		wlog.LogWithLine("[%s] Added response header: model=%s", config.PluginName, pluginCtx.Model)
	}

	if pluginCtx.ModelType != "" {
		proxywasm.AddHttpResponseHeader("model_type", pluginCtx.ModelType)
		wlog.LogWithLine("[%s] Added response header: model_type=%s", config.PluginName, pluginCtx.ModelType)
	}

	// 如果转换失败，在响应头中添加失败的阶段
	if pluginCtx.TransformFail != "" {
		proxywasm.AddHttpResponseHeader("x-transform-fail-stage", pluginCtx.TransformFail)
		wlog.LogWithLine("[%s] Added response header: x-transform-fail-stage=%s", config.PluginName, pluginCtx.TransformFail)
	}

	// 如果不是流式请求，设置正确的 content-type 和清理 transfer-encoding
	if !pluginCtx.HasStreamRequest {
		// 修改或添加 content-type: application/json; charset=utf-8
		proxywasm.RemoveHttpResponseHeader("content-type")
		proxywasm.AddHttpResponseHeader("content-type", "application/json; charset=utf-8")
		wlog.LogWithLine("[%s] Set content-type to application/json; charset=utf-8 (in onHttpResponseBody)", config.PluginName)
	}

	if len(body) == 0 {
		wlog.LogWithLine("[%s] Response body is empty, skipping transformation", config.PluginName)
		return types.ActionContinue
	}

	// 检查响应是否来自上游
	if !wrapper.IsResponseFromUpstream() {
		wlog.LogWithLine("[%s] Response not from upstream, skipping processing", config.PluginName)
		return types.ActionContinue
	}

	// 如果实际服务返回非正常状态，返回统一的错误提示
	if pluginCtx.IsErrorStatus {
		wlog.LogWithLine("[%s] Error status detected, calling handleErrorResponse", config.PluginName)
		return handleErrorResponse(ctx, pluginCtx, body)
	}

	// 根据 model 和 model_type 进行转换
	if pluginCtx.Model == "" || pluginCtx.ModelType == "" {
		wlog.LogWithLine("[%s] Model or ModelType is empty, skipping transformation. Model: '%s', ModelType: '%s'", config.PluginName, pluginCtx.Model, pluginCtx.ModelType)
		return types.ActionContinue
	}

	// 打印转换前的响应体
	wlog.LogWithLine("[%s] ========== Response Transformation ==========", config.PluginName)
	wlog.LogWithLine("[%s] Model: %s, ModelType: %s", config.PluginName, pluginCtx.Model, pluginCtx.ModelType)
	wlog.LogWithLine("[%s] Original Response Body:", config.PluginName)
	printJSON(config.PluginName, body, "  ", "Original Response Body:")

	var transformedBody []byte
	var err error
	if pluginCtx.ModelType == "t2i" {
		// 转换响应体
		wlog.LogWithLine("[%s] Calling TransformResponse with Model: %s, ModelType: %s", config.PluginName, pluginCtx.Model, pluginCtx.ModelType)
		transformedBody, err = t2i.TransformResponse(pluginCtx.Model, pluginCtx.ModelType, body)
		if err != nil {
			wlog.LogWithLine("[%s] Failed to transform response: %v", config.PluginName, err)
			// 记录转换失败的阶段
			pluginCtx.TransformFail = "response"
			return types.ActionContinue
		}
	}

	if transformedBody != nil {
		// 打印转换后的响应体
		wlog.LogWithLine("[%s] Transformed Response Body:", config.PluginName)
		printJSON(config.PluginName, transformedBody, "  ")
		wlog.LogWithLine("[%s] Response transformed successfully", config.PluginName)
		proxywasm.ReplaceHttpResponseBody(transformedBody)
		// 更新 Content-Length 头，防止请求卡住
		proxywasm.RemoveHttpResponseHeader("content-length")
		proxywasm.AddHttpResponseHeader("content-length", fmt.Sprintf("%d", len(transformedBody)))

		// 如果请求不是流式请求，删除 transfer-encoding header（可能在 BufferResponseBody 后被重新添加）
		if !pluginCtx.HasStreamRequest {
			proxywasm.RemoveHttpResponseHeader("transfer-encoding")
			wlog.LogWithLine("[%s] Removed transfer-encoding header in onHttpResponseBody (not a stream request)", config.PluginName)
		}
	} else {
		wlog.LogWithLine("[%s] No transformation needed, response body unchanged", config.PluginName)
		// 即使没有转换，如果不是流式请求，也删除 transfer-encoding header
		if !pluginCtx.HasStreamRequest {
			proxywasm.RemoveHttpResponseHeader("transfer-encoding")
			wlog.LogWithLine("[%s] Removed transfer-encoding header in onHttpResponseBody (not a stream request, no transformation)", config.PluginName)
		}
	}
	wlog.LogWithLine("[%s] ============================================", config.PluginName)

	return types.ActionContinue
}

// onHttpStreamingResponseBody 处理流式响应体
func onHttpStreamingResponseBody(ctx wrapper.HttpContext, cfg Config, chunk []byte, isLastChunk bool) []byte {
	pluginCtx := getOrCreatePluginContext(ctx)

	wlog.LogWithLine("[%s] ========== onHttpStreamingResponseBody called ==========", config.PluginName)
	wlog.LogWithLine("[%s] Entered streaming response body handler", config.PluginName)
	wlog.LogWithLine("[%s] Chunk length: %d", config.PluginName, len(chunk))
	wlog.LogWithLine("[%s] IsLastChunk: %v", config.PluginName, isLastChunk)
	wlog.LogWithLine("[%s] Model: %s, ModelType: %s", config.PluginName, pluginCtx.Model, pluginCtx.ModelType)
	wlog.LogWithLine("[%s] HasStreamRequest: %v", config.PluginName, pluginCtx.HasStreamRequest)
	wlog.LogWithLine("[%s] IsResponseFromUpstream: %v", config.PluginName, wrapper.IsResponseFromUpstream())

	// 打印 chunk 内容（限制长度以避免日志过长）
	chunkPreview := string(chunk)
	if len(chunkPreview) > 200 {
		chunkPreview = chunkPreview[:200] + "..."
	}
	wlog.LogWithLine("[%s] Chunk preview: %s", config.PluginName, chunkPreview)
	wlog.LogWithLine("[%s] ============================================", config.PluginName)

	// 直接透传 chunk，不做任何修改
	return chunk
}

// handleErrorResponse 处理错误响应，转换为统一错误格式
// 格式: { "code": "状态码", "message": "实际错误返回结构体", "data": "higress transformer plugin" }
func handleErrorResponse(ctx wrapper.HttpContext, pluginCtx *PluginContext, body []byte) types.Action {
	wlog.LogWithLine("[%s] ========== Error Response Handling ==========", config.PluginName)
	wlog.LogWithLine("[%s] Status Code: %s", config.PluginName, pluginCtx.StatusCode)
	wlog.LogWithLine("[%s] Original Error Response Body:", config.PluginName)
	printJSON(config.PluginName, body, "  ")

	// 尝试解析实际错误返回结构体（可能是 JSON）
	var errorMessage interface{}
	if len(body) > 0 {
		// 尝试解析为 JSON
		var jsonObj interface{}
		if err := json.Unmarshal(body, &jsonObj); err == nil {
			// 如果成功解析为 JSON，使用解析后的对象
			errorMessage = jsonObj
		} else {
			// 如果不是 JSON，使用原始字符串
			errorMessage = string(body)
		}
	} else {
		errorMessage = ""
	}

	// 构建统一错误格式
	errorResponse := map[string]interface{}{
		"code":    pluginCtx.StatusCode,
		"message": errorMessage,
		"data":    "higress transformer plugin",
	}

	// 转换为 JSON
	errorJSON, err := json.Marshal(errorResponse)
	if err != nil {
		wlog.LogWithLine("[%s] Failed to marshal error response: %v", config.PluginName, err)
		return types.ActionContinue
	}

	// 打印转换后的错误响应
	wlog.LogWithLine("[%s] Unified Error Response:", config.PluginName)
	printJSON(config.PluginName, errorJSON, "  ")

	// 替换响应体
	proxywasm.ReplaceHttpResponseBody(errorJSON)

	// 确保响应头中的 Content-Type 为 application/json
	proxywasm.RemoveHttpResponseHeader("content-type")
	proxywasm.AddHttpResponseHeader("content-type", "application/json")

	// 更新 Content-Length 头，这是防止请求卡住和传输错误的关键步骤
	proxywasm.RemoveHttpResponseHeader("content-length")
	proxywasm.AddHttpResponseHeader("content-length", fmt.Sprintf("%d", len(errorJSON)))

	wlog.LogWithLine("[%s] ============================================", config.PluginName)
	return types.ActionContinue
}
