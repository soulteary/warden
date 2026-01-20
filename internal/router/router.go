// Package router 提供了 HTTP 路由处理功能。
// 包括请求日志记录、JSON 响应、健康检查等路由处理器。
package router

import (
	// 标准库
	"net/http"
	"time"

	// 第三方库
	"github.com/justinas/alice"
	"github.com/rs/zerolog/hlog"

	// 项目内部包
	"github.com/soulteary/warden/internal/i18n"
	"github.com/soulteary/warden/internal/logger"
)

// ProcessWithLogger 为 HTTP 处理器添加日志记录中间件
//
// 该函数使用 alice 中间件链为处理器添加以下功能：
// - 远程地址记录：记录客户端 IP 地址
// - 用户代理记录：记录客户端 User-Agent
// - 引用来源记录：记录 HTTP Referer 头
// - 请求 ID 生成：为每个请求生成唯一 ID（从 Request-Id 头读取或自动生成）
//
// 注意：访问日志由外层的 AccessLogMiddleware 统一处理，避免重复记录。
//
// 参数:
//   - handler: HTTP 请求处理函数
//
// 返回:
//   - http.Handler: 包装后的 HTTP 处理器，包含日志记录功能
func ProcessWithLogger(handler func(http.ResponseWriter, *http.Request)) http.Handler {
	logInstance := logger.GetLogger()
	c := alice.New()
	c = c.Append(hlog.NewHandler(logInstance))

	// 添加字段处理器，确保在访问日志中能获取到这些字段
	c = c.Append(hlog.RemoteAddrHandler("ip"))
	c = c.Append(hlog.UserAgentHandler("user_agent"))
	c = c.Append(hlog.RefererHandler("referer"))
	c = c.Append(hlog.RequestIDHandler("req_id", "Request-Id"))

	// 注意：访问日志处理器已移到外层的 AccessLogMiddleware，避免重复记录

	return c.Then(http.HandlerFunc(handler))
}

// AccessLogMiddleware 创建访问日志中间件
//
// 该中间件可以在最外层使用，确保所有请求（包括认证失败的）都能记录访问日志。
// 返回一个可以包装任何 http.Handler 的中间件函数。
//
// 返回:
//   - func(http.Handler) http.Handler: HTTP 中间件函数
func AccessLogMiddleware() func(http.Handler) http.Handler {
	logInstance := logger.GetLogger()
	return func(next http.Handler) http.Handler {
		c := alice.New()
		c = c.Append(hlog.NewHandler(logInstance))

		// 先添加字段处理器，确保在访问日志中能获取到这些字段
		c = c.Append(hlog.RemoteAddrHandler("ip"))
		c = c.Append(hlog.UserAgentHandler("user_agent"))
		c = c.Append(hlog.RefererHandler("referer"))
		c = c.Append(hlog.RequestIDHandler("req_id", "Request-Id"))

		// 然后添加访问日志处理器
		c = c.Append(hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
			// 访问日志使用默认语言，因为这是系统日志
			hlog.FromRequest(r).Info().
				Str("method", r.Method).
				Stringer("url", r.URL).
				Int("status", status).
				Int("size", size).
				Dur("duration", duration).
				Msg("HTTP request")
		}))

		return c.Then(next)
	}
}
