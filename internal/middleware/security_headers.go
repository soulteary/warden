// Package middleware 提供了 HTTP 中间件功能。
// 包括速率限制、压缩、请求体限制、指标收集等中间件。
package middleware

import (
	// 标准库
	"net/http"
)

// SecurityHeadersMiddleware 创建安全响应头中间件
//
// 该中间件添加安全相关的 HTTP 响应头，提高应用的安全性。
// 包括：
// - X-Content-Type-Options: 防止 MIME 类型嗅探
// - X-Frame-Options: 防止点击劫持
// - X-XSS-Protection: 启用浏览器 XSS 过滤器
// - Referrer-Policy: 控制 referrer 信息
// - Content-Security-Policy: 内容安全策略（可选）
//
// 返回:
//   - func(http.Handler) http.Handler: HTTP 中间件函数
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 防止 MIME 类型嗅探
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// 防止点击劫持
		w.Header().Set("X-Frame-Options", "DENY")

		// 启用浏览器 XSS 过滤器
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// 控制 referrer 信息（不泄露来源）
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// 内容安全策略（CSP）- 根据实际需求调整
		// 这里使用较宽松的策略，如果需要更严格的安全，可以调整
		csp := "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self' data:;"
		w.Header().Set("Content-Security-Policy", csp)

		// 继续处理请求
		next.ServeHTTP(w, r)
	})
}
