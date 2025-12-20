package lib

import (
	"ai-data-masking/config"
	"crypto/sha256"
	"encoding/hex"
	"strings"
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
