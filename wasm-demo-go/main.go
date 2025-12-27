package main

import (
	"errors"
	"strings"

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

// getRouteName 获取路由名称
func getRouteName() (string, error) {
	if raw, err := proxywasm.GetProperty([]string{"route_name"}); err != nil {
		return "-", err
	} else {
		return string(raw), nil
	}
}

// getAPIName 获取 API 名称（从路由名称中解析）
func getAPIName() (string, error) {
	if raw, err := proxywasm.GetProperty([]string{"route_name"}); err != nil {
		return "-", err
	} else {
		parts := strings.Split(string(raw), "@")
		if len(parts) < 3 {
			return "-", errors.New("not api type")
		} else {
			return strings.Join(parts[:3], "@"), nil
		}
	}
}

// getClusterName 获取集群名称（服务信息）
func getClusterName() (string, error) {
	if raw, err := proxywasm.GetProperty([]string{"cluster_name"}); err != nil {
		return "-", err
	} else {
		return string(raw), nil
	}
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config MyConfig) types.Action {
	// 获取所有请求头并打印
	reqHeaders, err := proxywasm.GetHttpRequestHeaders()
	if err == nil {
		log.Warnf("[2] onHttpRequestHeaders -> 请求中的所有 header:")
		for _, header := range reqHeaders {
			if len(header) >= 2 {
				log.Warnf("[2] onHttpRequestHeaders ->  %s: %s", header[0], header[1])
			}
		}
	} else {
		// 错误级别日志：获取请求头失败
		log.Errorf("[2] onHttpRequestHeaders -> 获取请求头失败: %v", err)
	}

	// 获取路由和服务信息并写入请求 header
	route, routeErr := getRouteName()
	if routeErr == nil && route != "" && route != "-" {
		proxywasm.AddHttpRequestHeader("x-route-name", route)
		log.Warnf("[2] onHttpRequestHeaders -> 路由名称: %s", route)
		// 保存到 context 以便后续使用
		ctx.SetContext("route_name", route)
	} else {
		// 警告级别日志：获取路由名称失败（非关键错误）
		log.Warnf("[2] onHttpRequestHeaders -> 获取路由名称失败: %v", routeErr)
	}

	cluster, clusterErr := getClusterName()
	if clusterErr == nil && cluster != "" && cluster != "-" {
		proxywasm.AddHttpRequestHeader("x-cluster-name", cluster)
		log.Warnf("[2] onHttpRequestHeaders -> 集群名称(服务): %s", cluster)
		// 保存到 context 以便后续使用
		ctx.SetContext("cluster_name", cluster)
	} else {
		// 警告级别日志：获取集群名称失败（非关键错误）
		log.Warnf("[2] onHttpRequestHeaders -> 获取集群名称失败: %v", clusterErr)
	}

	api, apiErr := getAPIName()
	if apiErr == nil && api != "" && api != "-" {
		proxywasm.AddHttpRequestHeader("x-api-name", api)
		log.Warnf("[2] onHttpRequestHeaders -> API 名称: %s", api)
		// 保存到 context 以便后续使用
		ctx.SetUserAttribute("api_name", api)
	} else {
		// 调试级别日志：获取 API 名称失败（可能不是 API 类型，这是正常的）
		log.Warnf("[2] onHttpRequestHeaders -> 获取 API 名称失败(可能不是 API 类型): %v", apiErr)
	}

	proxywasm.AddHttpRequestHeader("hello", "world")
	if config.mockEnable {
		proxywasm.SendHttpResponse(200, nil, []byte("hello world"), -1)
		log.Info("[2] onHttpRequestHeaders -> HeaderStopIteration: mock mode stops header iteration并直接回包")
		// 警告级别日志：mock 模式启用，请求被拦截
		log.Warn("[2] onHttpRequestHeaders -> Mock 模式已启用，请求被拦截并返回模拟响应")
		return types.HeaderStopIteration
	}
	log.Info("[2] onHttpRequestHeaders -> HeaderContinue: 继续透传请求头")
	return types.HeaderContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config MyConfig, body []byte) types.Action {
	if len(body) > 0 {
		log.Warnf("[3] onHttpRequestBody received body: %s", string(body))
		// 警告级别日志：请求体过大
		if len(body) > 10*1024*1024 { // 10MB
			log.Warnf("[3] onHttpRequestBody -> 请求体过大: %d bytes，可能影响性能", len(body))
		}
	} else {
		log.Info("[3] onHttpRequestBody no body data")
	}
	log.Info("[3] onHttpRequestBody -> ActionContinue: 请求体继续透传")
	return types.ActionContinue
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config MyConfig) types.Action {
	// 获取所有响应头并打印
	respHeaders, err := proxywasm.GetHttpResponseHeaders()
	if err == nil {
		log.Warnf("[4] onHttpResponseHeaders -> 响应中的所有 header:")
		for _, header := range respHeaders {
			if len(header) >= 2 {
				log.Infof("  %s: %s", header[0], header[1])
			}
		}
	} else {
		// 错误级别日志：获取响应头失败
		log.Errorf("[4] onHttpResponseHeaders -> 获取响应头失败: %v", err)
	}

	// 将路由和服务信息也添加到响应头中，方便客户端查看
	if route, ok := ctx.GetContext("route_name").(string); ok && route != "" {
		proxywasm.AddHttpResponseHeader("x-route-name", route)
		log.Warnf("[4] onHttpResponseHeaders -> 已添加路由信息到响应头: %s", route)
	} else {
		// 警告级别日志：路由信息缺失
		log.Warn("[4] onHttpResponseHeaders -> 路由信息缺失，无法添加到响应头")
	}

	if cluster, ok := ctx.GetContext("cluster_name").(string); ok && cluster != "" {
		proxywasm.AddHttpResponseHeader("x-cluster-name", cluster)
		log.Warnf("[4] onHttpResponseHeaders -> 已添加集群信息到响应头: %s", cluster)
	} else {
		// 警告级别日志：集群信息缺失
		log.Warn("[4] onHttpResponseHeaders -> 集群信息缺失，无法添加到响应头")
	}

	if api, ok := ctx.GetUserAttribute("api_name").(string); ok && api != "" {
		proxywasm.AddHttpResponseHeader("x-api-name", api)
		log.Warnf("[4] onHttpResponseHeaders -> 已添加 API 信息到响应头: %s", api)
	} else {
		log.Warn("[4] onHttpResponseHeaders -> API 信息缺失，无法添加到响应头")
	}

	proxywasm.AddHttpResponseHeader("x-wasm-demo", "hello world2")

	if config.mockEnable {
		proxywasm.AddHttpResponseHeader("x-wasm-mock", "true")
	}
	log.Info("[4] onHttpResponseHeaders -> HeaderContinue: 响应头继续发送给客户端")
	return types.HeaderContinue
}

func onHttpResponseBody(ctx wrapper.HttpContext, config MyConfig, body []byte) types.Action {
	if len(body) > 0 {
		log.Warnf("[5] onHttpResponseBody received body: %s", string(body))
		// 警告级别日志：响应体过大
		if len(body) > 10*1024*1024 { // 10MB
			log.Warnf("[5] onHttpResponseBody -> 响应体过大: %d bytes，可能影响性能", len(body))
		}
		// 错误级别日志示例：检查响应体是否包含错误信息
		if strings.Contains(string(body), "\"error\"") || strings.Contains(string(body), "\"status\":\"error\"") {
			log.Error("[5] onHttpResponseBody -> 检测到响应体包含错误信息")
		}
	} else {
		log.Info("[5] onHttpResponseBody no body data")
	}
	log.Info("[5] onHttpResponseBody -> ActionContinue: 响应体继续发送")
	return types.ActionContinue
}
