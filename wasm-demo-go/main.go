package main

import (
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
	if config.mockEnable {
		proxywasm.SendHttpResponse(200, nil, []byte("hello world"), -1)
		log.Info("[2] onHttpRequestHeaders -> HeaderStopIteration: mock mode stops header iteration并直接回包")
		return types.HeaderStopIteration
	}
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
	proxywasm.AddHttpResponseHeader("x-wasm-demo", "enabled")
	if config.mockEnable {
		proxywasm.AddHttpResponseHeader("x-wasm-mock", "true")
	}
	log.Info("[4] onHttpResponseHeaders -> HeaderContinue: 响应头继续发送给客户端")
	return types.HeaderContinue
}

func onHttpResponseBody(ctx wrapper.HttpContext, config MyConfig, body []byte) types.Action {
	if len(body) > 0 {
		log.Infof("[5] onHttpResponseBody received body: %s", string(body))
	} else {
		log.Info("[5] onHttpResponseBody no body data")
	}
	log.Info("[5] onHttpResponseBody -> ActionContinue: 响应体继续发送")
	return types.ActionContinue
}
