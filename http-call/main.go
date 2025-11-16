package main

import (
	"fmt"
	"net/http"

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
	client     wrapper.HttpClient
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

	// 静态IP
	// config.client = wrapper.NewClusterClient(wrapper.StaticIpCluster{
	// 	ServiceName: "172.22.220.21",
	// 	Port:        int64(8000),
	// 	Host:        "172.22.220.21",
	// })
	// 域名
	config.client = wrapper.NewClusterClient(wrapper.FQDNCluster{
		FQDN: "jsonplaceholder.typicode.com",
		Port: int64(443),
		Host: "jsonplaceholder.typicode.com",
	})

	log.Infof("HTTP call start, cluster: %s, requestPath: %s", config.client.ClusterName(), "/echo/post")
	err := config.client.Post("/posts", [][2]string{}, []byte(`{"message": "hello from wasm"}`), func(statusCode int, responseHeaders http.Header, responseBody []byte) {
		log.Infof("HTTP call response: status=%d, body=%s", statusCode, string(responseBody))

		// 在回调函数中添加响应头（此时响应头处理阶段还未结束，可以添加）
		if statusCode >= 200 && statusCode < 300 {
			proxywasm.AddHttpResponseHeader("X-External-Response", "HTTP call success")
			proxywasm.AddHttpResponseHeader("X-External-Status", fmt.Sprintf("%d", statusCode))
		} else {
			proxywasm.AddHttpResponseHeader("X-External-Response", "HTTP call failed")
			proxywasm.AddHttpResponseHeader("X-External-Status", fmt.Sprintf("%d", statusCode))
		}

		// ResumeHttpRequest() 的作用是恢复主请求的处理流程。
		// 当我们发起外部 HTTP 调用后，会暂停主请求（即 Envoy/WASM 处理流程被挂起），
		// 等 HTTP 回调处理完后调用 ResumeHttpRequest 让主请求继续下游传递。
		proxywasm.ResumeHttpRequest()
	}, uint32(5000)) // 5 second timeout
	if err != nil {
		log.Errorf("HTTP call failed: %v", err)
		return types.ActionContinue
	} else {
		// 将 HTTP call 成功信息存储到 ctx 中，以便在响应头处理阶段使用
		ctx.SetUserAttribute("X-External-Response", "HTTP call success")
	}
	return types.HeaderStopAllIterationAndWatermark // 暂停主请求的处理流程，等待HTTP回调处理完成
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
	proxywasm.AddHttpResponseHeader("x-wasm-demo", "this is http-call plugin")
	if config.mockEnable {
		proxywasm.AddHttpResponseHeader("x-wasm-mock", "true")
	}
	// 从 ctx 中获取 X-External-Response 的值
	externalResponse, ok := ctx.GetUserAttribute("X-External-Response").(string)
	if ok && externalResponse != "" {
		proxywasm.AddHttpResponseHeader("X-External-Response", externalResponse)
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
