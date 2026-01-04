// Package middleware 提供了 HTTP 中间件功能。
// 包括速率限制、压缩、请求体限制、指标收集等中间件。
package middleware

import (
	// 标准库
	"net/http"

	// 第三方库
	"github.com/rs/zerolog/hlog"

	// 项目内部包
	"soulteary.com/soulteary/warden/internal/define"
)

// BodyLimitMiddleware 创建请求体大小限制中间件
// 限制请求体大小，防止恶意请求
// 注意：http.MaxBytesReader 会在读取时自动检查大小，超过限制会返回错误
func BodyLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 对于 GET/HEAD 请求，通常没有请求体，直接通过
		if r.Method == "GET" || r.Method == "HEAD" {
			next.ServeHTTP(w, r)
			return
		}

		// 检查 Content-Length 头
		if r.ContentLength > define.MAX_REQUEST_BODY_SIZE {
			hlog.FromRequest(r).Warn().
				Int64("content_length", r.ContentLength).
				Int("max_size", define.MAX_REQUEST_BODY_SIZE).
				Msg("请求体大小超过限制")
			http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
			return
		}

		// 限制请求体大小（MaxBytesReader 会在读取时检查）
		// 如果超过限制，会在后续读取时返回错误
		r.Body = http.MaxBytesReader(w, r.Body, define.MAX_REQUEST_BODY_SIZE)

		next.ServeHTTP(w, r)
	})
}
