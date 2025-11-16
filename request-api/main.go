package main

import (
	"net/http"
	"path"

	"request-api/config"

	"fmt"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
)

func main() {}

func init() {
	wrapper.SetCtx(
		"request-api",
		wrapper.ParseConfig(config.ParseConfig),
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
		wrapper.ProcessRequestBody(onHttpRequestBody),
		wrapper.ProcessResponseHeaders(onHttpResponseHeaders),
	)
}

const (
	HeaderAuthorization    = "authorization"
	HeaderFailureModeAllow = "x-envoy-auth-failure-mode-allowed"
)

// 目前，x-forwarded-xxx 头部仅在 forward_auth 模式下使用
const (
	HeaderOriginalMethod   = "x-original-method"
	HeaderOriginalUri      = "x-original-uri"
	HeaderXForwardedProto  = "x-forwarded-proto"
	HeaderXForwardedMethod = "x-forwarded-method"
	HeaderXForwardedUri    = "x-forwarded-uri"
	HeaderXForwardedHost   = "x-forwarded-host"
)

func onHttpRequestHeaders(ctx wrapper.HttpContext, config config.ExtAuthConfig) types.Action {
	// 禁用路由重新计算，因为插件可能会修改与所选路由相关的一些头部
	ctx.DisableReroute()

	// 如果 withRequestBody 为 true 且 HTTP 请求包含请求体，
	// 将在 onHttpRequestBody 阶段处理
	if wrapper.HasRequestBody() && config.HttpService.AuthorizationRequest.WithRequestBody {
		ctx.SetRequestBodyBufferLimit(config.HttpService.AuthorizationRequest.MaxRequestBodyBytes)
		// 请求包含请求体，需要延迟头部传输直到缓存未命中时再发送头部
		return types.HeaderStopIteration
	}

	ctx.DontReadRequestBody()
	return checkExtAuth(ctx, config, nil, types.HeaderStopAllIterationAndWatermark)
}

func onHttpRequestBody(ctx wrapper.HttpContext, config config.ExtAuthConfig, body []byte) types.Action {
	if config.HttpService.AuthorizationRequest.WithRequestBody {
		return checkExtAuth(ctx, config, body, types.DataStopIterationAndBuffer)
	}
	return types.ActionContinue
}

func checkExtAuth(ctx wrapper.HttpContext, cfg config.ExtAuthConfig, body []byte, pauseAction types.Action) types.Action {
	httpServiceConfig := cfg.HttpService

	extAuthReqHeaders := buildExtAuthRequestHeaders(ctx, cfg)

	// 根据 endpoint_mode 设置 requestMethod 和 requestPath
	requestMethod := httpServiceConfig.RequestMethod
	requestPath := httpServiceConfig.Path
	if httpServiceConfig.EndpointMode == config.EndpointModeEnvoy {
		requestMethod = ctx.Method()
		requestPath = path.Join(httpServiceConfig.PathPrefix, ctx.Path())
	}

	// 调用外部服务
	headersArray := convertHeadersToArray(extAuthReqHeaders)
	log.Errorf("requestMethod: %s, requestPath: %s", requestMethod, requestPath)
	log.Errorf("HTTP call start, cluster: %s, requestPath: %s", httpServiceConfig.Client.ClusterName(), requestPath)
	log.Errorf("config: %v", cfg)
	err := httpServiceConfig.Client.Call(requestMethod, requestPath, headersArray, body,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			if statusCode >= 200 && statusCode < 300 {
				// 使用 SetUserAttribute 存储外部服务响应数据，在响应头处理阶段添加
				ctx.SetUserAttribute("X-External-Status", fmt.Sprintf("%d", statusCode))
				// 如果响应体太大，只存储前 1024 字节，避免超出限制
				responseBodyStr := string(responseBody)
				if len(responseBodyStr) > 1024 {
					responseBodyStr = responseBodyStr[:1024] + "...(truncated)"
				}
				ctx.SetUserAttribute("X-External-Response", responseBodyStr)
				proxywasm.ResumeHttpRequest()

			} else {
				log.Errorf("failed to call ext auth server, status: %d", statusCode)
				callExtAuthServerErrorHandler(cfg, statusCode, responseHeaders, responseBody)
				return
			}

		}, httpServiceConfig.Timeout)

	if err != nil {

		log.Errorf("failed to call ext auth server: %v", err)
		// 由于调用错误和 HTTP 状态码 500 的处理逻辑相同，这里直接使用 500
		callExtAuthServerErrorHandler(cfg, http.StatusInternalServerError, nil, nil)
		return types.ActionContinue
	}
	return pauseAction
}

// buildExtAuthRequestHeaders 构建要发送到外部服务的请求头部
func buildExtAuthRequestHeaders(ctx wrapper.HttpContext, cfg config.ExtAuthConfig) http.Header {
	extAuthReqHeaders := http.Header{}

	httpServiceConfig := cfg.HttpService
	requestConfig := httpServiceConfig.AuthorizationRequest
	reqHeaders, _ := proxywasm.GetHttpRequestHeaders()

	// 复制所有请求头到外部服务请求
	for _, header := range reqHeaders {
		extAuthReqHeaders.Set(header[0], header[1])
	}

	// 添加自定义头部
	for key, value := range requestConfig.HeadersToAdd {
		extAuthReqHeaders.Set(key, value)
	}

	// 当 endpoint_mode 为 forward_auth 时添加额外的头部
	if httpServiceConfig.EndpointMode == config.EndpointModeForwardAuth {
		// 兼容旧版本
		extAuthReqHeaders.Set(HeaderOriginalMethod, ctx.Method())
		extAuthReqHeaders.Set(HeaderOriginalUri, ctx.Path())
		// 添加 x-forwarded-xxx 头部
		extAuthReqHeaders.Set(HeaderXForwardedProto, ctx.Scheme())
		extAuthReqHeaders.Set(HeaderXForwardedMethod, ctx.Method())
		extAuthReqHeaders.Set(HeaderXForwardedUri, ctx.Path())
		extAuthReqHeaders.Set(HeaderXForwardedHost, ctx.Host())
	}
	return extAuthReqHeaders
}

func callExtAuthServerErrorHandler(config config.ExtAuthConfig, statusCode int, extAuthRespHeaders http.Header, responseBody []byte) {
	if statusCode >= http.StatusInternalServerError && config.FailureModeAllow {
		if config.FailureModeAllowHeaderAdd {
			_ = proxywasm.ReplaceHttpRequestHeader(HeaderFailureModeAllow, "true")
		}
		proxywasm.ResumeHttpRequest()
		return
	}

	// 如果外部服务不可用或返回 5xx 状态码，则使用 StatusOnError 拒绝客户端请求
	// 否则，使用外部服务返回的状态码来拒绝请求
	statusToUse := statusCode
	if statusCode >= http.StatusInternalServerError {
		statusToUse = int(config.StatusOnError)
	}

	// 将响应头转换为数组格式
	respHeadersArray := convertHeadersToArray(extAuthRespHeaders)
	_ = proxywasm.SendHttpResponse(uint32(statusToUse), respHeadersArray, responseBody, -1)
}

// onHttpResponseHeaders 处理响应头，添加外部服务调用的响应信息
func onHttpResponseHeaders(ctx wrapper.HttpContext, config config.ExtAuthConfig) types.Action {
	// 从 UserAttribute 中获取外部服务响应数据
	if externalStatus, ok := ctx.GetUserAttribute("X-External-Status").(string); ok && externalStatus != "" {
		proxywasm.AddHttpResponseHeader("X-External-Status", externalStatus)
	}
	if externalResponse, ok := ctx.GetUserAttribute("X-External-Response").(string); ok && externalResponse != "" {
		proxywasm.AddHttpResponseHeader("X-External-Response", externalResponse)
	}
	return types.ActionContinue
}

// convertHeadersToArray 将 http.Header 转换为 [][2]string 格式
func convertHeadersToArray(headers http.Header) [][2]string {
	result := make([][2]string, 0)
	for key, values := range headers {
		for _, value := range values {
			result = append(result, [2]string{key, value})
		}
	}
	return result
}
