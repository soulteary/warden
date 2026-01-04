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
	"soulteary.com/soulteary/warden/internal/logger"
)

// ProcessWithLogger 为 HTTP 处理器添加日志记录中间件
//
// 该函数使用 alice 中间件链为处理器添加以下功能：
// - 请求日志记录：记录请求方法、URL、状态码、响应大小和耗时
// - 远程地址记录：记录客户端 IP 地址
// - 用户代理记录：记录客户端 User-Agent
// - 引用来源记录：记录 HTTP Referer 头
// - 请求 ID 生成：为每个请求生成唯一 ID（从 Request-Id 头读取或自动生成）
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

	c = c.Append(hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
		hlog.FromRequest(r).Info().
			Str("method", r.Method).
			Stringer("url", r.URL).
			Int("status", status).
			Int("size", size).
			Dur("duration", duration).
			Msg("")
	}))
	c = c.Append(hlog.RemoteAddrHandler("ip"))
	c = c.Append(hlog.UserAgentHandler("user_agent"))
	c = c.Append(hlog.RefererHandler("referer"))
	c = c.Append(hlog.RequestIDHandler("req_id", "Request-Id"))

	return c.Then(http.HandlerFunc(handler))
}
