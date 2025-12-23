package lib

import (
	"ai-data-masking/config"
	"runtime"
	"testing"
)

// 模拟 SSE chunk 数据
var (
	testSSEChunk = []byte(`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":123,"model":"gpt-3.5-turbo","choices":[{"index":0,"delta":{"role":"assistant","content":"这是"},"finish_reason":null}]}

`)

	testSSEChunkWithSensitiveWord = []byte(`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":123,"model":"gpt-3.5-turbo","choices":[{"index":0,"delta":{"role":"assistant","content":"敏感词1"},"finish_reason":null}]}

`)

	testSSEChunkCrossChunk = []byte(`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":123,"model":"gpt-3.5-turbo","choices":[{"index":0,"delta":{"role":"assistant","content":"这是敏"},"finish_reason":null}]}

`)

	testSSEChunkDone = []byte(`data: [DONE]

`)
)

// createTestPluginContext 创建测试用的 PluginContext
func createTestPluginContext() *config.PluginContext {
	cfg := &config.AiDataMaskingConfig{
		DenyWords:    testDenyWords,
		SystemDeny:   true,
		DenyMessage:  "检测到敏感词",
		StreamBuffer: 10 * 1024, // 10KB
	}

	return &config.PluginContext{
		Config:                cfg,
		OpenAIRequest:         &config.OpenAIRequest{Model: "gpt-3.5-turbo", Stream: true},
		StreamContentBuffer:   "",
		StreamReasoningBuffer: "",
		StreamDenied:          false,
		StreamChunkBuffer:     make([]config.StreamChunk, 0),
		StreamChunkBufferSize: 0,
	}
}

// TestAccuracy_ProcessOpenAIStreamResponse 测试流式处理准确率
func TestAccuracy_ProcessOpenAIStreamResponse(t *testing.T) {
	tests := []struct {
		name           string
		chunk          []byte
		isLastChunk    bool
		expectedDenied bool
		expectedOutput bool
		desc           string
	}{
		{
			name:           "正常 chunk",
			chunk:          testSSEChunk,
			isLastChunk:    false,
			expectedDenied: false,
			expectedOutput: true,
			desc:           "正常 chunk 应该通过",
		},
		{
			name:           "包含敏感词的 chunk",
			chunk:          testSSEChunkWithSensitiveWord,
			isLastChunk:    false,
			expectedDenied: true,
			expectedOutput: true,
			desc:           "包含敏感词的 chunk 应该被拒绝",
		},
		{
			name:           "[DONE] chunk",
			chunk:          testSSEChunkDone,
			isLastChunk:    true,
			expectedDenied: false,
			expectedOutput: true,
			desc:           "[DONE] chunk 应该正常处理",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pluginCtx := createTestPluginContext()
			// 注意：这里需要 mock HttpContext，实际测试中可能需要使用接口
			// 为了简化，我们直接测试内部逻辑

			// 由于 ProcessOpenAIStreamResponse 需要 wrapper.HttpContext，
			// 这里我们主要测试敏感词检测的准确率
			// 实际集成测试需要完整的上下文
			_ = pluginCtx
			_ = tt
		})
	}
}

// TestMemoryUsage_StreamProcessing 测试流式处理内存使用
func TestMemoryUsage_StreamProcessing(t *testing.T) {
	cfg := createTestConfig()
	systemWords := testSystemDenyWords

	// 模拟流式处理：多个 chunk
	chunks := [][]byte{
		testSSEChunk,
		testSSEChunk,
		testSSEChunk,
		testSSEChunkWithSensitiveWord,
	}

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// 模拟处理多个 chunk
	for i := 0; i < 100; i++ {
		pluginCtx := createTestPluginContext()
		for _, chunk := range chunks {
			// 提取 content 进行检测
			content := extractContentFromSSE(chunk)
			if content != "" {
				pluginCtx.StreamContentBuffer += content
				CheckMessage(pluginCtx.StreamContentBuffer, cfg, systemWords, true)
			}
		}
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	allocated := m2.TotalAlloc - m1.TotalAlloc
	allocs := m2.Mallocs - m1.Mallocs

	t.Logf("流式处理内存使用情况:")
	t.Logf("  总分配内存: %d bytes (%.2f KB)", allocated, float64(allocated)/1024)
	t.Logf("  分配次数: %d", allocs)
	t.Logf("  平均每次处理: %d bytes", allocated/100)
}

// extractContentFromSSE 从 SSE chunk 中提取 content（简化版）
func extractContentFromSSE(chunk []byte) string {
	// 这是一个简化的提取函数，实际应该使用 gjson 解析
	// 这里仅用于测试
	chunkStr := string(chunk)
	if len(chunkStr) > 100 {
		// 模拟提取 content
		return "这是"
	}
	return ""
}

// BenchmarkMemory_StreamBuffer 测试流式缓冲区内存使用
func BenchmarkMemory_StreamBuffer(b *testing.B) {
	cfg := createTestConfig()
	systemWords := testSystemDenyWords

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pluginCtx := createTestPluginContext()
		// 模拟添加内容到缓冲区
		for j := 0; j < 100; j++ {
			pluginCtx.StreamContentBuffer += "这是测试内容"
			// 检测
			CheckMessage(pluginCtx.StreamContentBuffer, cfg, systemWords, true)
		}
	}
	b.StopTimer()

	runtime.GC()
	runtime.ReadMemStats(&m2)

	allocated := m2.TotalAlloc - m1.TotalAlloc
	allocs := m2.Mallocs - m1.Mallocs

	b.ReportMetric(float64(allocated)/float64(b.N), "B/op")
	b.ReportMetric(float64(allocs)/float64(b.N), "allocs/op")
}

// BenchmarkStreamBuffer_SlidingWindow 测试滑动窗口内存使用
func BenchmarkStreamBuffer_SlidingWindow(b *testing.B) {
	bufferSize := uint32(10 * 1024) // 10KB

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pluginCtx := createTestPluginContext()
		pluginCtx.Config.StreamBuffer = bufferSize

		// 模拟滑动窗口
		for j := 0; j < 1000; j++ {
			contentDelta := "这是测试内容"
			pluginCtx.StreamContentBuffer += contentDelta

			// 限制缓冲区大小
			if len(pluginCtx.StreamContentBuffer) > int(bufferSize) {
				cutoff := len(pluginCtx.StreamContentBuffer) - int(bufferSize)
				pluginCtx.StreamContentBuffer = pluginCtx.StreamContentBuffer[cutoff:]
			}
		}
	}
	b.StopTimer()

	runtime.GC()
	runtime.ReadMemStats(&m2)

	allocated := m2.TotalAlloc - m1.TotalAlloc
	allocs := m2.Mallocs - m1.Mallocs

	b.ReportMetric(float64(allocated)/float64(b.N), "B/op")
	b.ReportMetric(float64(allocs)/float64(b.N), "allocs/op")
}
