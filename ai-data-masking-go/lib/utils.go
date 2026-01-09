package lib

import (
	"ai-data-masking/config"
	"encoding/json"
	"strings"
	"unicode/utf8"
)

// ReplaceSensitiveWordsWithValue 使用指定的 value 替换敏感词，保持字符数相等
// 如果 value 长度不够，会重复 value 直到达到敏感词的长度
// 如果 value 长度超过敏感词，会截断 value
func ReplaceSensitiveWordsWithValue(text string, config *config.AiDataMaskingConfig, systemDenyWords []string, replaceValue string) string {
	if text == "" {
		return text
	}

	// 如果 replaceValue 为空，使用 "*" 作为默认值
	if replaceValue == "" {
		replaceValue = "*"
	}

	result := text
	textBytes := []byte(text)

	// 检查自定义敏感词
	if len(config.DenyWords) > 0 {
		matcher := getOrBuildCustomMatcher(config.DenyWords)
		matches := matcher.Match(textBytes)
		if len(matches) > 0 {
			// 使用 map 去重，避免重复处理同一个敏感词
			processedWords := make(map[int]bool, len(matches))
			for _, wordIdx := range matches {
				if processedWords[wordIdx] {
					continue
				}
				processedWords[wordIdx] = true
				matchedWord := config.DenyWords[wordIdx]

				// 计算敏感词的字符数（不是字节数）
				wordRuneCount := utf8.RuneCountInString(matchedWord)
				replaceValueRuneCount := utf8.RuneCountInString(replaceValue)

				// 生成替换字符串，保持字符数相等
				var replacement string
				if replaceValueRuneCount == wordRuneCount {
					// 长度相等，直接使用
					replacement = replaceValue
				} else if replaceValueRuneCount < wordRuneCount {
					// value 长度不够，重复 value 直到达到敏感词的长度
					repeatCount := (wordRuneCount + replaceValueRuneCount - 1) / replaceValueRuneCount // 向上取整
					replacement = strings.Repeat(replaceValue, repeatCount)
					// 截断到精确长度
					replacementRunes := []rune(replacement)
					replacement = string(replacementRunes[:wordRuneCount])
				} else {
					// value 长度超过敏感词，截断 value
					replaceValueRunes := []rune(replaceValue)
					replacement = string(replaceValueRunes[:wordRuneCount])
				}

				// 替换所有匹配的敏感词
				result = strings.ReplaceAll(result, matchedWord, replacement)
			}
		}
	}

	// 检查系统敏感词
	if config.SystemDeny && len(systemDenyWords) > 0 {
		matcher := getOrBuildSystemMatcher(systemDenyWords)
		matches := matcher.Match([]byte(result))
		if len(matches) > 0 {
			// 使用 map 去重，避免重复处理同一个敏感词
			processedWords := make(map[int]bool, len(matches))
			for _, wordIdx := range matches {
				if processedWords[wordIdx] {
					continue
				}
				processedWords[wordIdx] = true
				matchedWord := systemDenyWords[wordIdx]

				// 计算敏感词的字符数（不是字节数）
				wordRuneCount := utf8.RuneCountInString(matchedWord)
				replaceValueRuneCount := utf8.RuneCountInString(replaceValue)

				// 生成替换字符串，保持字符数相等
				var replacement string
				if replaceValueRuneCount == wordRuneCount {
					// 长度相等，直接使用
					replacement = replaceValue
				} else if replaceValueRuneCount < wordRuneCount {
					// value 长度不够，重复 value 直到达到敏感词的长度
					repeatCount := (wordRuneCount + replaceValueRuneCount - 1) / replaceValueRuneCount // 向上取整
					replacement = strings.Repeat(replaceValue, repeatCount)
					// 截断到精确长度
					replacementRunes := []rune(replacement)
					replacement = string(replacementRunes[:wordRuneCount])
				} else {
					// value 长度超过敏感词，截断 value
					replaceValueRunes := []rune(replaceValue)
					replacement = string(replaceValueRunes[:wordRuneCount])
				}

				// 替换所有匹配的敏感词
				result = strings.ReplaceAll(result, matchedWord, replacement)
			}
		}
	}

	return result
}

// calculateMaxSensitiveWordLength 计算最长敏感词的长度（字节数）
// 用于确定需要保留多少历史数据以检测跨窗口边界的敏感词
func CalculateMaxSensitiveWordLength(cfg *config.AiDataMaskingConfig) int {
	maxLen := 0
	maxWordLen := 0
	// 检查自定义敏感词
	if len(cfg.DenyWords) > 0 {
		for _, word := range cfg.DenyWords {
			if utf8.RuneCountInString(word) > maxWordLen {
				maxWordLen = utf8.RuneCountInString(word)
			}
		}
	}

	// 检查系统敏感词
	// if cfg.SystemDeny && len(config.SystemDenyWords) > 0 {
	// 	for _, word := range config.SystemDenyWords {
	// 		if utf8.RuneCountInString(word) > maxWordLen {
	// 			maxWordLen = utf8.RuneCountInString(word)
	// 		}
	// 	}
	// }

	// 如果没有任何敏感词，返回一个默认值（比如 20 字节）
	// 这样可以保留一些历史数据，以防万一
	if maxLen == 0 {
		maxLen = maxWordLen * 3 * 2 // byte 中文占3个字节，英文占1个字节，2倍冗余
	}

	return maxLen
}

func PrintConfig(cfg *config.AiDataMaskingConfig) []byte {
	b, _ := json.MarshalIndent(cfg, "", "  ")
	return b
}
