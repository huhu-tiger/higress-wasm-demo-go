package lib

import (
	"ai-data-masking/config"
	"runtime"
	"testing"
)

// 测试数据
var (
	testDenyWords = []string{
		"敏感词1",
		"敏感词2",
		"测试敏感词",
		"违规内容",
		"禁止词汇",
		"不良信息",
		"违法内容",
		"不当言论",
		"恶意攻击",
		"垃圾信息",
	}

	testSystemDenyWords = []string{
		"系统敏感词1",
		"系统敏感词2",
		"系统禁止词",
	}

	// 测试文本：包含敏感词
	testTextWithSensitiveWords = "这是一段包含敏感词1和敏感词2的测试文本，还有测试敏感词和违规内容。"

	// 测试文本：不包含敏感词
	testTextWithoutSensitiveWords = "这是一段正常的测试文本，没有任何敏感内容。"

	// 测试文本：跨 chunk 的敏感词
	testTextCrossChunk = "这是敏" + "感词1" + "的内容"

	// 长文本测试
	longText = generateLongText(10000)
)

// generateLongText 生成长文本用于测试
func generateLongText(length int) string {
	baseText := "这是一段测试文本，用于测试长文本的性能和内存使用情况。"
	result := ""
	for len(result) < length {
		result += baseText
	}
	return result[:length]
}

// createTestConfig 创建测试配置
func createTestConfig() *config.AiDataMaskingConfig {
	return &config.AiDataMaskingConfig{
		DenyWords:    testDenyWords,
		SystemDeny:   true,
		DenyMessage:  "检测到敏感词",
		StreamBuffer: 10 * 1024, // 10KB
	}
}

// BenchmarkCheckMessage_NonStream 测试非流式检测性能
func BenchmarkCheckMessage_NonStream(b *testing.B) {
	cfg := createTestConfig()
	systemWords := testSystemDenyWords

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// 注意：在测试环境中，proxywasm.LogWarnf 会 panic
		// 这里我们只测试检测逻辑，不测试日志输出
		_ = CheckMessage(testTextWithSensitiveWords, cfg, systemWords, false)
	}
}

// BenchmarkCheckMessage_Stream 测试流式检测性能
func BenchmarkCheckMessage_Stream(b *testing.B) {
	cfg := createTestConfig()
	systemWords := testSystemDenyWords

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = CheckMessage(testTextWithSensitiveWords, cfg, systemWords, true)
	}
}

// BenchmarkFindSensitiveWordMatches 测试敏感词位置查找性能
func BenchmarkFindSensitiveWordMatches(b *testing.B) {
	cfg := createTestConfig()
	systemWords := testSystemDenyWords

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = FindSensitiveWordMatches(testTextWithSensitiveWords, cfg, systemWords)
	}
}

// BenchmarkFindSensitiveWordMatches_LongText 测试长文本的敏感词查找性能
func BenchmarkFindSensitiveWordMatches_LongText(b *testing.B) {
	cfg := createTestConfig()
	systemWords := testSystemDenyWords

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = FindSensitiveWordMatches(longText, cfg, systemWords)
	}
}

// BenchmarkCheckMessage_MultipleWords 测试多个敏感词的检测性能
func BenchmarkCheckMessage_MultipleWords(b *testing.B) {
	cfg := createTestConfig()
	cfg.DenyWords = append(cfg.DenyWords, testSystemDenyWords...)
	systemWords := testSystemDenyWords

	text := "这是一段包含" + testDenyWords[0] + "和" + testDenyWords[1] + "以及" + testSystemDenyWords[0] + "的文本"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = CheckMessage(text, cfg, systemWords, false)
	}
}

// TestAccuracy_FindSensitiveWordMatches 测试位置查找准确率（不调用 proxywasm）
func TestAccuracy_FindSensitiveWordMatches(t *testing.T) {
	cfg := createTestConfig()
	systemWords := testSystemDenyWords

	tests := []struct {
		name          string
		text          string
		expectedCount int
		expectedWords []string
		desc          string
	}{
		{
			name:          "单个敏感词",
			text:          "包含敏感词1的文本",
			expectedCount: 1,
			expectedWords: []string{"敏感词1"},
			desc:          "应该找到一个敏感词",
		},
		{
			name:          "多个敏感词",
			text:          testTextWithSensitiveWords,
			expectedCount: 4, // 敏感词1, 敏感词2, 测试敏感词, 违规内容
			expectedWords: []string{"敏感词1", "敏感词2", "测试敏感词", "违规内容"},
			desc:          "应该找到多个敏感词",
		},
		{
			name:          "重复敏感词",
			text:          "敏感词1和敏感词1再次出现",
			expectedCount: 2,
			expectedWords: []string{"敏感词1", "敏感词1"},
			desc:          "应该找到重复的敏感词",
		},
		{
			name:          "无敏感词",
			text:          testTextWithoutSensitiveWords,
			expectedCount: 0,
			expectedWords: []string{},
			desc:          "不应该找到敏感词",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := FindSensitiveWordMatches(tt.text, cfg, systemWords)

			if len(results) != tt.expectedCount {
				t.Errorf("%s: 期望找到 %d 个敏感词, 实际找到 %d 个", tt.desc, tt.expectedCount, len(results))
			}

			// 验证找到的敏感词是否正确
			foundWords := make(map[string]int)
			for _, result := range results {
				foundWords[result.MatchedWord]++
			}

			for _, expectedWord := range tt.expectedWords {
				if foundWords[expectedWord] == 0 {
					t.Errorf("%s: 期望找到敏感词 '%s', 但未找到", tt.desc, expectedWord)
				}
			}

			// 验证位置是否正确
			for _, result := range results {
				if result.StartPos < 0 || result.EndPos <= result.StartPos {
					t.Errorf("%s: 无效的位置范围: StartPos=%d, EndPos=%d", tt.desc, result.StartPos, result.EndPos)
				}
				// 验证位置对应的文本确实是敏感词
				if result.StartPos < len(tt.text) && result.EndPos <= len(tt.text) {
					matchedText := tt.text[result.StartPos:result.EndPos]
					if matchedText != result.MatchedWord {
						t.Errorf("%s: 位置对应的文本不匹配: 期望 '%s', 实际 '%s'", tt.desc, result.MatchedWord, matchedText)
					}
				}
			}
		})
	}
}

// TestMemoryUsage_FindSensitiveWordMatches 测试位置查找的内存使用
func TestMemoryUsage_FindSensitiveWordMatches(t *testing.T) {
	cfg := createTestConfig()
	systemWords := testSystemDenyWords

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// 执行多次查找
	for i := 0; i < 1000; i++ {
		_ = FindSensitiveWordMatches(testTextWithSensitiveWords, cfg, systemWords)
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	allocated := m2.TotalAlloc - m1.TotalAlloc
	allocs := m2.Mallocs - m1.Mallocs

	t.Logf("内存使用情况:")
	t.Logf("  总分配内存: %d bytes (%.2f KB)", allocated, float64(allocated)/1024)
	t.Logf("  分配次数: %d", allocs)
	t.Logf("  平均每次分配: %d bytes", allocated/1000)
}

// BenchmarkMemory_CheckMessage 内存使用 benchmark
func BenchmarkMemory_CheckMessage(b *testing.B) {
	cfg := createTestConfig()
	systemWords := testSystemDenyWords

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CheckMessage(testTextWithSensitiveWords, cfg, systemWords, false)
	}
	b.StopTimer()

	runtime.GC()
	runtime.ReadMemStats(&m2)

	allocated := m2.TotalAlloc - m1.TotalAlloc
	allocs := m2.Mallocs - m1.Mallocs

	b.ReportMetric(float64(allocated)/float64(b.N), "B/op")
	b.ReportMetric(float64(allocs)/float64(b.N), "allocs/op")
}

// BenchmarkMemory_FindSensitiveWordMatches 位置查找内存使用 benchmark
func BenchmarkMemory_FindSensitiveWordMatches(b *testing.B) {
	cfg := createTestConfig()
	systemWords := testSystemDenyWords

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FindSensitiveWordMatches(testTextWithSensitiveWords, cfg, systemWords)
	}
	b.StopTimer()

	runtime.GC()
	runtime.ReadMemStats(&m2)

	allocated := m2.TotalAlloc - m1.TotalAlloc
	allocs := m2.Mallocs - m1.Mallocs

	b.ReportMetric(float64(allocated)/float64(b.N), "B/op")
	b.ReportMetric(float64(allocs)/float64(b.N), "allocs/op")
}

// BenchmarkConcurrent_CheckMessage 并发测试
func BenchmarkConcurrent_CheckMessage(b *testing.B) {
	cfg := createTestConfig()
	systemWords := testSystemDenyWords

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = CheckMessage(testTextWithSensitiveWords, cfg, systemWords, false)
		}
	})
}

// BenchmarkConcurrent_FindSensitiveWordMatches 并发位置查找测试
func BenchmarkConcurrent_FindSensitiveWordMatches(b *testing.B) {
	cfg := createTestConfig()
	systemWords := testSystemDenyWords

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = FindSensitiveWordMatches(testTextWithSensitiveWords, cfg, systemWords)
		}
	})
}
