package lib

import (
	"ai-data-masking/config"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode/utf8"
)

// replaceMessage 替换消息中的敏感词
func ReplaceMessage(message string, pluginCtx *config.PluginContext) string {
	if len(pluginCtx.Config.ReplaceRoles) == 0 {
		return message
	}

	result := message
	for _, rule := range pluginCtx.Config.ReplaceRoles {
		if rule.CompiledRegex == nil {
			continue
		}

		if rule.Type == "replace" && !rule.Restore {
			// 简单替换，不还原
			result = rule.CompiledRegex.ReplaceAllString(result, rule.Value)
		} else {
			// 需要还原的替换或 hash
			matches := rule.CompiledRegex.FindAllString(result, -1)
			for _, match := range matches {
				var toWord string
				if rule.Type == "hash" {
					// SHA256 hash
					hash := sha256.Sum256([]byte(match))
					toWord = hex.EncodeToString(hash[:])
				} else {
					// 替换
					toWord = rule.CompiledRegex.ReplaceAllString(match, rule.Value)
				}

				// 记录映射关系用于还原
				if rule.Restore && toWord != "" {
					pluginCtx.MaskMap[toWord] = &match
				}

				result = strings.ReplaceAll(result, match, toWord)
			}
		}
	}

	return result
}

// restoreMessage 还原消息中的脱敏数据
func RestoreMessage(message string, pluginCtx *config.PluginContext) string {
	if len(pluginCtx.MaskMap) == 0 {
		return message
	}

	result := message
	for hash, original := range pluginCtx.MaskMap {
		if original != nil {
			result = strings.ReplaceAll(result, hash, *original)
		}
	}

	return result
}

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
