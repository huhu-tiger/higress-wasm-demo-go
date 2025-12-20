package lib

import (
	"encoding/json"
	"fmt"
	"strings"

	"ai-data-masking/config"
	"ai-data-masking/wlog"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// processOpenAIRequest 处理 OpenAI 格式请求（使用 gjson 解析）
func ProcessOpenAIRequest(ctx wrapper.HttpContext, pluginCtx *config.PluginContext, body []byte) (bool, bool) {
	bodyStr := string(body)

	// 使用 gjson 解析基础字段
	root := gjson.Parse(bodyStr)
	if !root.Exists() {
		return false, false
	}

	// 检查是否是 OpenAI 请求
	contentResult := gjson.Get(bodyStr, "messages.0.content")

	if !contentResult.Exists() {
		// pluginCtx.RequestDenyType = config.DenyTypeOpenAI // 不是openai格式，不设置拒绝类型，返回false,false
		return false, false
	}

	// 初始化 OpenAIRequest（如果为 nil）
	if pluginCtx.OpenAIRequest == nil {
		pluginCtx.OpenAIRequest = &config.OpenAIRequest{}
	}

	stream := root.Get("stream").Bool()
	pluginCtx.OpenAIRequest.Stream = stream
	pluginCtx.OpenAIRequest.Model = root.Get("model").String()

	messages := root.Get("messages")
	if !messages.Exists() || messages.Type != gjson.JSON {
		return false, false
	}

	modified := false
	denied := false

	// 遍历 messages 数组
	messages.ForEach(func(key, v gjson.Result) bool {
		idx := key.Int()

		content := v.Get("content").String()
		reasoningContent := v.Get("reasoning_content").String()

		// 先做命中检查（请求阶段，非流式）
		if CheckMessage(content, pluginCtx.Config, config.SystemDenyWords, false) || CheckMessage(reasoningContent, pluginCtx.Config, config.SystemDenyWords, false) {
			// 命中直接拒绝，不再继续遍历
			denied = true
			return false
		}

		// 替换敏感词
		newContent := ReplaceMessage(content, pluginCtx)
		newReasoningContent := ReplaceMessage(reasoningContent, pluginCtx)

		// 如果有变更，用 sjson 回写
		basePath := fmt.Sprintf("messages.%d.", idx)

		if newContent != content {
			var err error
			bodyStr, err = sjson.Set(bodyStr, basePath+"content", newContent)
			if err == nil {
				modified = true
			}
		}
		if newReasoningContent != reasoningContent {
			var err error
			bodyStr, err = sjson.Set(bodyStr, basePath+"reasoning_content", newReasoningContent)
			if err == nil {
				modified = true
			}
		}

		return true
	})

	return modified, denied
}

// processJSONPathRequest 处理 JSONPath 请求
func ProcessJSONPathRequest(ctx wrapper.HttpContext, pluginCtx *config.PluginContext, body []byte) (bool, bool) {
	// 简化实现：使用 gjson 解析 JSONPath
	// 注意：这里需要完整的 JSONPath 实现
	bodyStr := string(body)
	modified := false
	denied := false
	for _, path := range pluginCtx.Config.DenyJSONPath {
		result := gjson.Get(bodyStr, path)
		if !result.Exists() {
			continue
		}
		pluginCtx.RequestDenyModifyType = config.DenyModifyTypeJSONPath
		// 1) 直接是字符串
		if result.Type == gjson.String {
			content := result.String()
			if CheckMessage(content, pluginCtx.Config, config.SystemDenyWords, false) {
				denied = true
				return modified, denied
			}

			newContent := ReplaceMessage(content, pluginCtx)
			if newContent != content {
				oldJson, _ := json.Marshal(content)
				newJson, _ := json.Marshal(newContent)
				bodyStr = strings.ReplaceAll(bodyStr, string(oldJson), string(newJson))
				modified = true
			}
			continue
		}

		// 2) 是数组（例如 messages.#.content 或 input.messages.#.content.#.text 等）
		if result.IsArray() {
			for _, item := range result.Array() {
				if item.Type != gjson.String {
					continue
				}
				content := item.String()
				if CheckMessage(content, pluginCtx.Config, config.SystemDenyWords, false) {
					denied = true
					return modified, denied
				}

				newContent := ReplaceMessage(content, pluginCtx)
				if newContent != content {
					oldJson, _ := json.Marshal(content)
					newJson, _ := json.Marshal(newContent)
					bodyStr = strings.ReplaceAll(bodyStr, string(oldJson), string(newJson))
					modified = true
				}
			}
		}
	}

	return modified, denied
}

// processRawRequest 处理原始请求，示例：
func ProcessRawRequest(ctx wrapper.HttpContext, pluginCtx *config.PluginContext, body []byte) (bool, bool) {
	bodyStr := string(body)
	modified := false
	denied := false

	if CheckMessage(bodyStr, pluginCtx.Config, config.SystemDenyWords, false) {
		denied = true
		return modified, denied
	}

	newBody := ReplaceMessage(bodyStr, pluginCtx)
	if newBody != bodyStr {
		modified = true
	}

	return modified, denied
}

// ProcessOpenAIResponse 处理 OpenAI 非流式 JSON 响应，返回动作和新的 body 字符串
// 参考 ProcessOpenAIRequest 的实现，使用 gjson 和 sjson 进行精确的 JSON 处理
func ProcessOpenAIResponse(ctx wrapper.HttpContext, pluginCtx *config.PluginContext, bodyStr string, body []byte) (bool, bool) {

	modified := false
	denied := false
	// 使用 gjson 解析基础字段
	root := gjson.Parse(bodyStr)
	if !root.Exists() {
		return false, false
	}
	// 检查是否是 OpenAI 响应格式
	contentResult := gjson.Get(bodyStr, "choices.0.message")

	if !contentResult.Exists() || contentResult.Type != gjson.JSON {
		// pluginCtx.RequestDenyType = config.DenyTypeOpenAI // 不是openai格式，不设置拒绝类型，返回false,false
		return false, false
	}

	//

	choices := gjson.Get(bodyStr, "choices")

	// 遍历 choices 数组
	choices.ForEach(func(key, choice gjson.Result) bool {
		idx := key.Int()

		content := choice.Get("message.content").String()
		reasoning := choice.Get("message.reasoning").String()

		// 先做命中检查（响应阶段，非流式）
		if CheckMessage(content, pluginCtx.Config, config.SystemDenyWords, false) ||
			CheckMessage(reasoning, pluginCtx.Config, config.SystemDenyWords, false) {
			// 命中直接拒绝，不再继续遍历
			denied = true
			return false // 停止遍历
		}

		// 替换敏感词（与请求阶段保持一致，使用 ReplaceMessage）
		newContent := ReplaceMessage(content, pluginCtx)
		newReasoningContent := ReplaceMessage(reasoning, pluginCtx)

		// 构建基础路径
		basePath := fmt.Sprintf("choices.%d.message.", idx)

		// 如果有变更，用 sjson 回写
		if newContent != content {
			var err error
			bodyStr, err = sjson.Set(bodyStr, basePath+"content", newContent)
			if err == nil {
				modified = true
			}
		}
		if newReasoningContent != reasoning {
			var err error
			bodyStr, err = sjson.Set(bodyStr, basePath+"reasoning", newReasoningContent)
			if err == nil {
				modified = true
			}
		}

		return true // 继续处理下一个 choice
	})

	// 如果检测到敏感词，调用 DenyHandler
	if denied {
		wlog.LogWithLine("[%s] ProcessOpenAIResponse: sensitive word detected, calling deny() - isStream=%v",
			pluginName, pluginCtx.OpenAIRequest.Stream)

		return modified, denied
	}

	// 如有修改，写回响应体
	if modified {
		proxywasm.ReplaceHttpResponseBody([]byte(bodyStr))
	}

	return modified, denied
}

// handleRawResponse 处理非 OpenAI 的原始响应体
func ProcessRawResponse(ctx wrapper.HttpContext, pluginCtx *config.PluginContext, bodyStr string) types.Action {
	// 命中敏感词直接拒绝（非流式响应）
	if CheckMessage(bodyStr, pluginCtx.Config, config.SystemDenyWords, false) {
		wlog.LogWithLine("[%s] ProcessRawResponse: sensitive word detected, calling deny() - isStream=%v, isOpenAI=%v",
			pluginName, pluginCtx.OpenAIRequest.Stream, pluginCtx.RequestDenyModifyType)
		action := DenyHandler(ctx, pluginCtx)
		wlog.LogWithLine("[%s] ProcessRawResponse: deny() returned action=%v", pluginName, action)
		return action
	}

	// 还原脱敏数据
	newBody := RestoreMessage(bodyStr, pluginCtx)
	if newBody != bodyStr {
		proxywasm.ReplaceHttpResponseBody([]byte(newBody))
	}

	return types.ActionContinue
}

// ProcessOpenAIStreamResponse 处理 OpenAI 流式 JSON 响应，使用滑动窗口缓冲区机制
// 返回处理后的 chunk 和是否拒绝的标识
func ProcessOpenAIStreamResponse(ctx wrapper.HttpContext, pluginCtx *config.PluginContext, chunk []byte, isLastChunk bool) ([]byte, bool) {
	// 如果已经拒绝，直接返回空，不再处理后续 chunk
	if pluginCtx.StreamDenied {
		return nil, true
	}

	// 初始化 chunk 缓冲区
	if pluginCtx.StreamChunkBuffer == nil {
		pluginCtx.StreamChunkBuffer = make([]config.StreamChunk, 0)
		pluginCtx.StreamChunkBufferSize = 0
	}

	// 获取缓冲区大小，默认为 10KB
	bufferSize := pluginCtx.Config.StreamBuffer
	if bufferSize == 0 {
		bufferSize = 10 * 1024 // 默认 10KB
	}

	// 使用 wrapper.UnifySSEChunk 统一处理 SSE 格式
	unifiedChunk := wrapper.UnifySSEChunk(chunk)

	// 按 \n\n 分割 SSE 事件（每个事件可能包含多行）
	events := strings.Split(strings.TrimSpace(string(unifiedChunk)), "\n\n")

	streamEnded := false // 标记流是否已结束

	// 处理当前 chunk 中的所有事件
	for _, eventStr := range events {
		if eventStr == "" {
			continue
		}

		// 检查是否是 [DONE] 标记
		if strings.Contains(eventStr, "data: [DONE]") {
			streamEnded = true
			// 添加 [DONE] chunk
			pluginCtx.StreamChunkBuffer = append(pluginCtx.StreamChunkBuffer, config.StreamChunk{
				Data:   []byte(eventStr + "\n\n"),
				IsDone: true,
			})
			pluginCtx.StreamChunkBufferSize += len(eventStr) + 2
			break
		}

		// 解析事件，提取 content 和 reasoning 增量
		lines := strings.Split(eventStr, "\n")
		contentStart := len(pluginCtx.StreamContentBuffer)
		reasoningStart := len(pluginCtx.StreamReasoningBuffer)

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// 处理 SSE 格式：data: {...} 或 data:{...}
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			jsonStr := strings.TrimSpace(strings.TrimPrefix(line, "data:"))

			// 解析 JSON
			root := gjson.Parse(jsonStr)
			if !root.Exists() {
				continue
			}

			// 检查是否是 OpenAI 流式响应格式
			choices := root.Get("choices")
			if !choices.Exists() || !choices.IsArray() {
				continue
			}

			// 处理每个 choice
			choices.ForEach(func(key, choice gjson.Result) bool {
				// 提取 delta 中的 content 和 reasoning 增量
				delta := choice.Get("delta")
				if !delta.Exists() {
					return true
				}

				contentDelta := delta.Get("content").String()
				reasoningDelta := delta.Get("reasoning").String()

				// 将增量添加到缓冲区（滑动窗口）
				if contentDelta != "" {
					pluginCtx.StreamContentBuffer += contentDelta
					// 限制缓冲区大小
					if len(pluginCtx.StreamContentBuffer) > int(bufferSize) {
						// 保留最新的 bufferSize 字节，滑动窗口
						pluginCtx.StreamContentBuffer = pluginCtx.StreamContentBuffer[len(pluginCtx.StreamContentBuffer)-int(bufferSize):]
						// 调整所有 chunk 的 content 位置
						for i := range pluginCtx.StreamChunkBuffer {
							if pluginCtx.StreamChunkBuffer[i].ContentStart >= len(pluginCtx.StreamContentBuffer)-int(bufferSize) {
								pluginCtx.StreamChunkBuffer[i].ContentStart -= len(pluginCtx.StreamContentBuffer) - int(bufferSize)
								pluginCtx.StreamChunkBuffer[i].ContentEnd -= len(pluginCtx.StreamContentBuffer) - int(bufferSize)
							} else {
								pluginCtx.StreamChunkBuffer[i].ContentStart = 0
								pluginCtx.StreamChunkBuffer[i].ContentEnd = 0
							}
						}
						contentStart = len(pluginCtx.StreamContentBuffer) - len(contentDelta)
					}
				}

				if reasoningDelta != "" {
					pluginCtx.StreamReasoningBuffer += reasoningDelta
					// 限制缓冲区大小
					if len(pluginCtx.StreamReasoningBuffer) > int(bufferSize) {
						// 保留最新的 bufferSize 字节，滑动窗口
						pluginCtx.StreamReasoningBuffer = pluginCtx.StreamReasoningBuffer[len(pluginCtx.StreamReasoningBuffer)-int(bufferSize):]
						// 调整所有 chunk 的 reasoning 位置
						for i := range pluginCtx.StreamChunkBuffer {
							if pluginCtx.StreamChunkBuffer[i].ReasoningStart >= len(pluginCtx.StreamReasoningBuffer)-int(bufferSize) {
								pluginCtx.StreamChunkBuffer[i].ReasoningStart -= len(pluginCtx.StreamReasoningBuffer) - int(bufferSize)
								pluginCtx.StreamChunkBuffer[i].ReasoningEnd -= len(pluginCtx.StreamReasoningBuffer) - int(bufferSize)
							} else {
								pluginCtx.StreamChunkBuffer[i].ReasoningStart = 0
								pluginCtx.StreamChunkBuffer[i].ReasoningEnd = 0
							}
						}
						reasoningStart = len(pluginCtx.StreamReasoningBuffer) - len(reasoningDelta)
					}
				}

				return true
			})
		}

		// 记录 chunk 信息
		contentEnd := len(pluginCtx.StreamContentBuffer)
		reasoningEnd := len(pluginCtx.StreamReasoningBuffer)

		pluginCtx.StreamChunkBuffer = append(pluginCtx.StreamChunkBuffer, config.StreamChunk{
			Data:           []byte(eventStr + "\n\n"),
			ContentStart:   contentStart,
			ContentEnd:     contentEnd,
			ReasoningStart: reasoningStart,
			ReasoningEnd:   reasoningEnd,
			IsDone:         false,
		})
		pluginCtx.StreamChunkBufferSize += len(eventStr) + 2
	}

	// 检查是否需要处理缓冲区（缓冲区满或流结束）
	shouldProcess := streamEnded || pluginCtx.StreamChunkBufferSize >= int(bufferSize)

	if !shouldProcess {
		// 缓冲区未满且流未结束，暂不返回，等待更多数据
		return nil, false
	}

	// 处理缓冲区：进行敏感词检查
	denied := false
	deniedChunkIndices := make(map[int]bool) // 记录包含敏感词的 chunk 索引

	// 检查累积缓冲区中是否包含敏感词，并获取所有匹配的位置
	// 这样可以识别跨越多个 chunk 的敏感词
	contentMatches := FindSensitiveWordMatches(pluginCtx.StreamContentBuffer, pluginCtx.Config, config.SystemDenyWords)
	reasoningMatches := FindSensitiveWordMatches(pluginCtx.StreamReasoningBuffer, pluginCtx.Config, config.SystemDenyWords)

	// 根据匹配位置，标记所有涉及的 chunk
	// 处理 content 缓冲区的匹配
	for _, match := range contentMatches {
		denied = true
		// 找到所有与这个敏感词位置重叠的 chunk
		for i, streamChunk := range pluginCtx.StreamChunkBuffer {
			if streamChunk.IsDone {
				continue
			}

			chunkStart := streamChunk.ContentStart
			chunkEnd := streamChunk.ContentEnd

			// 检查敏感词位置是否与 chunk 位置重叠
			// 如果敏感词的任何部分在 chunk 范围内，就标记这个 chunk
			if (match.StartPos >= chunkStart && match.StartPos < chunkEnd) ||
				(match.EndPos > chunkStart && match.EndPos <= chunkEnd) ||
				(match.StartPos <= chunkStart && match.EndPos >= chunkEnd) {
				deniedChunkIndices[i] = true
				wlog.LogWithLine("[%s] ProcessOpenAIStreamResponse: sensitive word '%s' detected in content, marking chunk %d (pos: %d-%d, chunk: %d-%d)",
					pluginName, match.MatchedWord, i, match.StartPos, match.EndPos, chunkStart, chunkEnd)
			}
		}
	}

	// 处理 reasoning 缓冲区的匹配
	for _, match := range reasoningMatches {
		denied = true
		// 找到所有与这个敏感词位置重叠的 chunk
		for i, streamChunk := range pluginCtx.StreamChunkBuffer {
			if streamChunk.IsDone {
				continue
			}

			chunkStart := streamChunk.ReasoningStart
			chunkEnd := streamChunk.ReasoningEnd

			// 检查敏感词位置是否与 chunk 位置重叠
			// 如果敏感词的任何部分在 chunk 范围内，就标记这个 chunk
			if (match.StartPos >= chunkStart && match.StartPos < chunkEnd) ||
				(match.EndPos > chunkStart && match.EndPos <= chunkEnd) ||
				(match.StartPos <= chunkStart && match.EndPos >= chunkEnd) {
				deniedChunkIndices[i] = true
				wlog.LogWithLine("[%s] ProcessOpenAIStreamResponse: sensitive word '%s' detected in reasoning, marking chunk %d (pos: %d-%d, chunk: %d-%d)",
					pluginName, match.MatchedWord, i, match.StartPos, match.EndPos, chunkStart, chunkEnd)
			}
		}
	}

	// 构建返回结果
	var result strings.Builder
	if denied {
		// 有敏感词：删除包含敏感词的 chunk，用自定义内容替换
		pluginCtx.StreamDenied = true

		// 替换所有被标记的 chunk
		replaced := false
		for i, streamChunk := range pluginCtx.StreamChunkBuffer {
			// 跳过 [DONE] chunk，最后统一处理
			if streamChunk.IsDone {
				continue
			}

			if deniedChunkIndices[i] {
				// 只替换第一个被标记的 chunk，后续的被标记 chunk 直接跳过（不输出）
				if !replaced {
					// 用自定义内容替换第一个被标记的 chunk
					denyMessage := pluginCtx.Config.DenyMessage
					if denyMessage == "" {
						denyMessage = "提问或回答中包含敏感词，已被屏蔽"
					}

					// 构造替换的 SSE 事件
					replacementChunk := fmt.Sprintf("data: {\"id\":\"chatcmpl-deny\",\"object\":\"chat.completion.chunk\",\"created\":123,\"model\":\"%s\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"%s\"},\"finish_reason\":null}]}\n\n",
						pluginCtx.OpenAIRequest.Model, denyMessage)
					result.WriteString(replacementChunk)
					replaced = true
				}
				// 其他被标记的 chunk 不输出，直接跳过
			} else {
				// 不包含敏感词的 chunk 原样返回
				result.Write(streamChunk.Data)
			}
		}

		// 如果流已结束，添加 [DONE]
		if streamEnded {
			result.WriteString("data: [DONE]\n\n")
		}
	} else {
		// 没有敏感词：原样返回所有 chunk
		for _, streamChunk := range pluginCtx.StreamChunkBuffer {
			result.Write(streamChunk.Data)
		}
	}

	// 清空缓冲区，准备处理下一批数据
	pluginCtx.StreamChunkBuffer = pluginCtx.StreamChunkBuffer[:0]
	pluginCtx.StreamChunkBufferSize = 0

	resultBytes := []byte(result.String())
	if len(resultBytes) == 0 {
		return nil, denied
	}

	return resultBytes, denied
}

// deny 拒绝请求/响应
func DenyHandler(ctx wrapper.HttpContext, pluginCtx *config.PluginContext) types.Action {
	cfg := pluginCtx.Config
	inResponseDeny := pluginCtx.IsResponseDeny
	isRequestDeny := pluginCtx.IsRequestDeny

	// 确保 OpenAIRequest 已初始化
	if pluginCtx.OpenAIRequest == nil {
		pluginCtx.OpenAIRequest = &config.OpenAIRequest{}
	}

	wlog.LogWithLine("[%s] deny() called: inResponseDeny=%v, isOpenAI=%v, isStream=%v",
		pluginName, inResponseDeny, pluginCtx.RequestDenyModifyType, pluginCtx.OpenAIRequest.Stream)

	if isRequestDeny {
		// 根据是否为流式请求设置不同的 Content-Type
		contentType := cfg.DenyContentType
		headers := [][2]string{
			{"Content-Type", contentType},
		}

		if pluginCtx.OpenAIRequest != nil && pluginCtx.OpenAIRequest.Stream {
			contentType = "text/event-stream; charset=utf-8"
			headers = [][2]string{
				{"Content-Type", contentType},
			}
			// 注意：不需要手动设置 transfer-encoding: chunked
			// 因为 SendHttpResponseWithDetail 一次性发送完整响应体时，
			// Envoy 会自动设置 Content-Length，而不是使用 chunked encoding
			// 对于 SSE 格式，即使有 Content-Length，客户端也能正常解析
		}

		// 安全地获取用户属性并添加到响应头
		if maskingAttr := ctx.GetUserAttribute("x-ai-data-masking"); maskingAttr != nil {
			if maskingStr, ok := maskingAttr.(string); ok && maskingStr != "" {
				headers = append(headers, [2]string{"x-ai-data-masking", maskingStr})
			}
		}
		if denyStepAttr := ctx.GetUserAttribute("deny_step"); denyStepAttr != nil {
			if denyStepStr, ok := denyStepAttr.(string); ok && denyStepStr != "" {
				headers = append(headers, [2]string{"deny_step", denyStepStr})
			}
		}
		denyMessage := ctx.GetUserAttribute("deny_message").([]byte)

		wlog.LogWithLine("[%s] deny() -> Calling SendHttpResponse: code=%d, headers=%d, contentType=%s, body length=%d",
			pluginName, cfg.DenyCode, len(headers), contentType, len(denyMessage))

		proxywasm.SendHttpResponseWithDetail(cfg.DenyCode, "", headers, denyMessage, -1)

		wlog.LogWithLine("[%s] deny() -> SendHttpResponse completed, returning ActionPause", pluginName)
		return types.ActionContinue
	}

	if inResponseDeny && !pluginCtx.OpenAIRequest.Stream {
		// 非流式响应：替换响应体并设置响应头
		wlog.LogWithLine("[%s] deny() -> NON-STREAMING response detected, will replace response body", pluginName)
		// 设置标志，表示响应已被拒绝，跳过后续处理
		ctx.SetUserAttribute("response_denied", "true")
		// 设置响应头（先移除再添加，确保覆盖）
		proxywasm.RemoveHttpResponseHeader("content-type")
		proxywasm.AddHttpResponseHeader("content-type", cfg.DenyContentType)
		// 如果存在用户属性，添加到响应头
		if maskingAttr := ctx.GetUserAttribute("x-ai-data-masking"); maskingAttr != nil {
			if maskingStr, ok := maskingAttr.(string); ok && maskingStr != "" {
				proxywasm.RemoveHttpResponseHeader("x-ai-data-masking")
				proxywasm.AddHttpResponseHeader("x-ai-data-masking", maskingStr)
			}
		}
		if denyStepAttr := ctx.GetUserAttribute("deny_step"); denyStepAttr != nil {
			if denyStepStr, ok := denyStepAttr.(string); ok && denyStepStr != "" {
				proxywasm.RemoveHttpResponseHeader("deny_step")
				proxywasm.AddHttpResponseHeader("deny_step", denyStepStr)
			}
		}
		denyMessage := ctx.GetUserAttribute("deny_message").([]byte)
		wlog.LogWithLine("[%s] deny() -> ReplaceHttpResponseBody called with denyMessage length=%d", pluginName, len(denyMessage))
		proxywasm.ReplaceHttpResponseBody(denyMessage)
		// 更新 Content-Length 头，这是防止请求卡住和传输错误的关键步骤
		proxywasm.RemoveHttpResponseHeader("content-length")
		proxywasm.AddHttpResponseHeader("content-length", fmt.Sprintf("%d", len(denyMessage)))
		wlog.LogWithLine("[%s] deny() -> ReplaceHttpResponseBody completed, Content-Length updated to %d, returning ActionContinue", pluginName, len(denyMessage))
		return types.ActionContinue
	}

	return types.ActionContinue
}
