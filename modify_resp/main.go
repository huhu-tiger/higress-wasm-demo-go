package main

import (
	"encoding/json"
	"fmt"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

func main() {}

func init() {
	wrapper.SetCtx(
		// 插件名称
		"my-plugin",
		// 为解析插件配置，设置自定义函数
		wrapper.ParseConfig(parseConfig),
		// 为处理请求头，设置自定义函数
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
		// 处理请求体
		wrapper.ProcessRequestBody(onHttpRequestBody),
		// 同时示例注册响应头处理逻辑
		wrapper.ProcessResponseHeaders(onHttpResponseHeaders),
		// 处理响应体
		wrapper.ProcessResponseBody(onHttpResponseBody),
	)
}

// 自定义插件配置
type MyConfig struct {
	mockEnable bool
}

// 在控制台插件配置中填写的yaml配置会自动转换为json，此处直接从json这个参数里解析配置即可
func parseConfig(json gjson.Result, config *MyConfig) error {
	// 解析出配置，更新到config中
	config.mockEnable = json.Get("mockEnable").Bool()
	log.Infof("[1] parseConfig -> mockEnable=%v", config.mockEnable)
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config MyConfig) types.Action {
	proxywasm.AddHttpRequestHeader("hello", "world")
	// 所有请求都继续处理，在响应体阶段统一返回错误信息
	log.Info("[2] onHttpRequestHeaders -> HeaderContinue: 继续透传请求头")
	return types.HeaderContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config MyConfig, body []byte) types.Action {
	if len(body) > 0 {
		log.Infof("[3] onHttpRequestBody received body: %s", string(body))
	} else {
		log.Info("[3] onHttpRequestBody no body data")
	}
	log.Info("[3] onHttpRequestBody -> ActionContinue: 请求体继续透传")
	return types.ActionContinue
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config MyConfig) types.Action {
	proxywasm.AddHttpResponseHeader("x-wasm-demo", "hello world2")

	if config.mockEnable {
		proxywasm.AddHttpResponseHeader("x-wasm-mock", "true")
	}
	log.Info("[4] onHttpResponseHeaders -> HeaderContinue: 响应头继续发送给客户端")
	return types.HeaderStopIteration
}

func onHttpResponseBody(ctx wrapper.HttpContext, config MyConfig, body []byte) types.Action {
	// 定义 JSON 错误响应结构
	type ErrorResponse struct {
		Error string `json:"错误"`
	}

	// 返回指定的错误信息
	errorResponse := ErrorResponse{
		Error: "请求的接口不支持 HTTP/1.1 协议，请开启「兼容 HTTP/2」，并设置「HTTP 连接方式」为 HTTP/2 先验知识",
	}

	// 将错误响应序列化为 JSON
	jsonBody, err := json.Marshal(errorResponse)
	if err != nil {
		log.Errorf("[5] onHttpResponseBody -> JSON marshal error: %v", err)
		return types.ActionContinue
	}

	// 替换响应体为 JSON 格式的错误信息
	// 注意：在响应体阶段不能使用 SendHttpResponse，因为响应头已经发送
	// 必须使用 ReplaceHttpResponseBody + 更新响应头的方式
	proxywasm.ReplaceHttpResponseBody(jsonBody)
	log.Infof("[5] onHttpResponseBody -> 响应体已修改为错误 JSON，新长度: %d", len(jsonBody))

	// 设置 Content-Type 为 application/json
	proxywasm.RemoveHttpResponseHeader("content-type")
	proxywasm.AddHttpResponseHeader("content-type", "application/json")

	// 更新 Content-Length 头，这是防止请求卡住和传输错误的关键步骤
	proxywasm.RemoveHttpResponseHeader("content-length")
	proxywasm.AddHttpResponseHeader("content-length", fmt.Sprintf("%d", len(jsonBody)))

	log.Info("[5] onHttpResponseBody -> ActionContinue: 错误响应体继续发送")
	return types.ActionContinue
}
