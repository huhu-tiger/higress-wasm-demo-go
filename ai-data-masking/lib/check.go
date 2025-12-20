package lib

import (
	"ai-data-masking/config"
	"sync"

	"github.com/cloudflare/ahocorasick"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
)

const pluginName = "ai-data-masking"

var (
	// 缓存已构建的匹配器，避免重复构建
	customMatcherCache *ahocorasick.Matcher
	customWordsCache   []string
	systemMatcherCache *ahocorasick.Matcher
	systemWordsCache   []string
	cacheMutex         sync.RWMutex
)

// CheckMessage 检查消息中是否包含敏感词
// isStream: true 表示流式处理，false 表示非流式处理
func CheckMessage(message string, config *config.AiDataMaskingConfig, systemDenyWords []string, isStream bool) bool {
	if message == "" {
		return false
	}

	// 非流式处理：直接匹配完整文本
	if !isStream {
		return checkNonStream(message, config, systemDenyWords)
	}

	// 流式处理：使用增量匹配
	return checkStream(message, config, systemDenyWords)
}

// checkNonStream 非流式处理：一次性匹配完整文本
func checkNonStream(message string, config *config.AiDataMaskingConfig, systemDenyWords []string) bool {
	messageBytes := []byte(message)

	// 检查自定义敏感词
	if len(config.DenyWords) > 0 {
		matcher := getOrBuildCustomMatcher(config.DenyWords)
		matches := matcher.Match(messageBytes)
		if len(matches) > 0 {
			// matches 返回的是匹配的字典索引，我们需要找到对应的敏感词
			matchedWord := config.DenyWords[matches[0]]
			proxywasm.LogWarnf("[%s] checkNonStream custom deny word %s matched from %s", pluginName, matchedWord, message)
			return true
		}
	}

	// 检查系统敏感词
	if config.SystemDeny && len(systemDenyWords) > 0 {
		matcher := getOrBuildSystemMatcher(systemDenyWords)
		matches := matcher.Match(messageBytes)
		if len(matches) > 0 {
			// matches 返回的是匹配的字典索引
			matchedWord := systemDenyWords[matches[0]]
			proxywasm.LogWarnf("[%s] system deny word %s matched from %s", pluginName, matchedWord, message)
			return true
		}
	}

	return false
}

// checkStream 流式处理：支持增量匹配
// 对于流式数据，我们需要检查当前 chunk 以及可能跨越 chunk 的敏感词
func checkStream(chunk string, config *config.AiDataMaskingConfig, systemDenyWords []string) bool {
	// 流式处理时，直接检查当前 chunk
	// 注意：如果敏感词可能跨越多个 chunk，需要在调用方维护缓冲区
	// 这里假设每个 chunk 都是相对完整的文本片段
	chunkBytes := []byte(chunk)

	// 检查自定义敏感词
	if len(config.DenyWords) > 0 {
		matcher := getOrBuildCustomMatcher(config.DenyWords)
		matches := matcher.Match(chunkBytes)
		if len(matches) > 0 {
			// matches 返回的是匹配的字典索引
			matchedWord := config.DenyWords[matches[0]]
			proxywasm.LogWarnf("[%s] [stream] custom deny word %s matched from chunk: %s", pluginName, matchedWord, chunk)
			return true
		}
	}

	// 检查系统敏感词
	if config.SystemDeny && len(systemDenyWords) > 0 {
		matcher := getOrBuildSystemMatcher(systemDenyWords)
		matches := matcher.Match(chunkBytes)
		if len(matches) > 0 {
			// matches 返回的是匹配的字典索引
			matchedWord := systemDenyWords[matches[0]]
			proxywasm.LogWarnf("[%s] [stream] system deny word %s matched from chunk: %s", pluginName, matchedWord, chunk)
			return true
		}
	}

	return false
}

// MatchResult 敏感词匹配结果
type MatchResult struct {
	MatchedWord string // 匹配到的敏感词
	StartPos    int    // 匹配开始位置（字节位置）
	EndPos      int    // 匹配结束位置（字节位置）
}

// FindSensitiveWordMatches 查找文本中所有敏感词匹配的位置
// 返回所有匹配的位置信息（按字节位置）
func FindSensitiveWordMatches(text string, config *config.AiDataMaskingConfig, systemDenyWords []string) []MatchResult {
	if text == "" {
		return nil
	}

	var results []MatchResult
	textBytes := []byte(text)

	// 检查自定义敏感词
	if len(config.DenyWords) > 0 {
		matcher := getOrBuildCustomMatcher(config.DenyWords)
		matches := matcher.Match(textBytes)
		if len(matches) > 0 {
			// 使用 map 去重，避免重复处理同一个敏感词
			processedWords := make(map[int]bool)
			for _, wordIdx := range matches {
				if processedWords[wordIdx] {
					continue
				}
				processedWords[wordIdx] = true
				matchedWord := config.DenyWords[wordIdx]
				// 在文本中查找所有该敏感词的位置
				wordBytes := []byte(matchedWord)
				start := 0
				for {
					pos := findBytes(textBytes[start:], wordBytes)
					if pos == -1 {
						break
					}
					actualPos := start + pos
					results = append(results, MatchResult{
						MatchedWord: matchedWord,
						StartPos:    actualPos,
						EndPos:      actualPos + len(wordBytes),
					})
					start = actualPos + 1
					if start >= len(textBytes) {
						break
					}
				}
			}
		}
	}

	// 检查系统敏感词
	if config.SystemDeny && len(systemDenyWords) > 0 {
		matcher := getOrBuildSystemMatcher(systemDenyWords)
		matches := matcher.Match(textBytes)
		if len(matches) > 0 {
			// 使用 map 去重，避免重复处理同一个敏感词
			processedWords := make(map[int]bool)
			for _, wordIdx := range matches {
				if processedWords[wordIdx] {
					continue
				}
				processedWords[wordIdx] = true
				matchedWord := systemDenyWords[wordIdx]
				// 在文本中查找所有该敏感词的位置
				wordBytes := []byte(matchedWord)
				start := 0
				for {
					pos := findBytes(textBytes[start:], wordBytes)
					if pos == -1 {
						break
					}
					actualPos := start + pos
					results = append(results, MatchResult{
						MatchedWord: matchedWord,
						StartPos:    actualPos,
						EndPos:      actualPos + len(wordBytes),
					})
					start = actualPos + 1
					if start >= len(textBytes) {
						break
					}
				}
			}
		}
	}

	return results
}

// findBytes 在字节切片中查找子切片的位置
func findBytes(haystack, needle []byte) int {
	if len(needle) == 0 {
		return 0
	}
	if len(needle) > len(haystack) {
		return -1
	}
	for i := 0; i <= len(haystack)-len(needle); i++ {
		match := true
		for j := 0; j < len(needle); j++ {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// getOrBuildCustomMatcher 获取或构建自定义敏感词匹配器
func getOrBuildCustomMatcher(words []string) *ahocorasick.Matcher {
	cacheMutex.RLock()
	if customMatcherCache != nil && wordsEqual(customWordsCache, words) {
		matcher := customMatcherCache
		cacheMutex.RUnlock()
		return matcher
	}
	cacheMutex.RUnlock()

	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	// 双重检查
	if customMatcherCache != nil && wordsEqual(customWordsCache, words) {
		return customMatcherCache
	}

	// 构建新的匹配器
	matcher := ahocorasick.NewStringMatcher(words)
	customMatcherCache = matcher
	customWordsCache = make([]string, len(words))
	copy(customWordsCache, words)

	return matcher
}

// getOrBuildSystemMatcher 获取或构建系统敏感词匹配器
func getOrBuildSystemMatcher(words []string) *ahocorasick.Matcher {
	cacheMutex.RLock()
	if systemMatcherCache != nil && wordsEqual(systemWordsCache, words) {
		matcher := systemMatcherCache
		cacheMutex.RUnlock()
		return matcher
	}
	cacheMutex.RUnlock()

	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	// 双重检查
	if systemMatcherCache != nil && wordsEqual(systemWordsCache, words) {
		return systemMatcherCache
	}

	// 构建新的匹配器
	matcher := ahocorasick.NewStringMatcher(words)
	systemMatcherCache = matcher
	systemWordsCache = make([]string, len(words))
	copy(systemWordsCache, words)

	return matcher
}

// wordsEqual 比较两个字符串切片是否相等
func wordsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
