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
	"runtime"
	"strings"
	"transformer-vnet/config"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/tidwall/pretty"
)

// printJSON 格式化并打印 JSON，带缩进（在一条日志中打印）
// 使用 runtime.Caller 获取调用者的位置信息，而不是 printJSON 的位置
// prefix: 可选的前缀文字，会在 JSON 前面显示，默认为空
func printJSON(pluginName string, data []byte, indent string, prefix ...string) {
	// 获取调用者的信息（跳过 printJSON 这一层）
	pc, file, line, _ := runtime.Caller(1)
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
	uniqueID := fmt.Sprintf("[%s:%s:L%d]", file, funcName, line)

	// 处理前缀参数（默认为空）
	prefixStr := ""
	if len(prefix) > 0 && prefix[0] != "" {
		prefixStr = prefix[0] + " "
	}

	if len(data) == 0 {
		proxywasm.LogWarnf("%s [%s]%s%s(empty)", uniqueID, pluginName, indent, prefixStr)
		return
	}

	// 尝试解析 JSON 并格式化
	var jsonObj interface{}
	if err := json.Unmarshal(data, &jsonObj); err != nil {
		// 如果不是有效的 JSON，直接打印原始内容（截断到 1K）
		dataStr := string(data)
		if len(dataStr) > config.MaxPrintSize {
			dataStr = dataStr[:config.MaxPrintSize] + fmt.Sprintf("... (truncated, total size: %d bytes)", len(data))
		}
		proxywasm.LogWarnf("%s [%s]%s%s(invalid JSON): %s", uniqueID, pluginName, indent, prefixStr, dataStr)
		return
	}

	// 重新编码为格式化的 JSON
	formattedJSON, err := json.MarshalIndent(jsonObj, indent, "  ")
	if err != nil {
		// 如果格式化失败，使用 pretty 包
		formattedJSON = pretty.Pretty(data)
	}

	// 将格式化的 JSON 作为一条日志打印
	// 如果超过 1K (1024 字节)，只打印前 1K
	formattedStr := string(formattedJSON)
	if len(formattedStr) > config.MaxPrintSize {
		formattedStr = formattedStr[:config.MaxPrintSize] + fmt.Sprintf("\n... (truncated, total size: %d bytes)", len(formattedJSON))
	}
	proxywasm.LogWarnf("%s [%s]%s%s%s", uniqueID, pluginName, indent, prefixStr, formattedStr)
}
