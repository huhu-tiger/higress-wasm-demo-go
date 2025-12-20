package wlog

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
)

// getCallerInfo 获取调用者的信息（文件名、函数名、行号）
func getCallerInfo() (string, string, int) {
	pc, file, line, _ := runtime.Caller(2)
	fn := runtime.FuncForPC(pc)
	funcName := "unknown"
	if fn != nil {
		funcName = fn.Name()
		// 只取函数名，去掉包路径
		if idx := strings.LastIndex(funcName, "."); idx >= 0 {
			funcName = funcName[idx+1:]
		}
	}
	// 只取文件名，去掉路径
	if idx := strings.LastIndex(file, "/"); idx >= 0 {
		file = file[idx+1:]
	}
	return file, funcName, line
}

// logStreamingDecision 流式判断的提醒日志
// 使用 proxywasm.LogWarnf 而不是 log.Warnf，避免框架自动添加 UUID
func LogStreamingDecision(format string, args ...interface{}) {
	file, funcName, line := getCallerInfo()
	uniqueID := fmt.Sprintf("[%s:%s:L%d]", file, funcName, line)
	alert := "🚨 [STREAMING DECISION] 🚨"
	proxywasm.LogWarnf(fmt.Sprintf("%s %s %s", uniqueID, alert, format), args...)
}

// logWithLine 带唯一标识的日志函数
// 使用 proxywasm.LogWarnf 而不是 log.Warnf，避免框架自动添加 UUID
func LogWithLine(format string, args ...interface{}) {
	file, funcName, line := getCallerInfo()
	uniqueID := fmt.Sprintf("[%s:%s:L%d]", file, funcName, line)
	proxywasm.LogWarnf(fmt.Sprintf("%s %s", uniqueID, format), args...)
}
