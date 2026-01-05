// Package middleware 提供了 HTTP 中间件功能。
// 包括速率限制、压缩、请求体限制、指标收集等中间件。
package middleware

import (
	// 标准库
	"net/http"
	"strings"

	// 第三方库
	"github.com/rs/zerolog/hlog"
)

// AuthMiddleware 创建 API Key 认证中间件
//
// 该中间件通过检查请求头中的 X-API-Key 来验证请求是否被授权。
// 如果 API Key 为空，则所有请求都会被拒绝（生产环境应该设置 API Key）。
// 如果 API Key 不为空，则只有提供正确 API Key 的请求才能通过。
//
// 参数:
//   - apiKey: API Key 值，如果为空则禁用认证（不推荐在生产环境使用）
//
// 返回:
//   - func(http.Handler) http.Handler: HTTP 中间件函数
//
// 使用示例:
//
//	authMiddleware := AuthMiddleware("your-api-key-here")
//	handler := authMiddleware(protectedHandler)
func AuthMiddleware(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 如果 API Key 为空，在生产环境应该拒绝所有请求
			// 但在开发环境可能允许（通过环境变量控制）
			if apiKey == "" {
				// 检查是否为开发环境（通过环境变量判断）
				// 在生产环境，应该设置 API Key
				hlog.FromRequest(r).Warn().
					Msg("API Key 未配置，请求被拒绝（生产环境必须配置 API Key）")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// 从请求头获取 API Key
			// 支持 X-API-Key 和 Authorization: Bearer <key> 两种方式
			providedKey := r.Header.Get("X-API-Key")
			if providedKey == "" {
				// 尝试从 Authorization 头获取
				authHeader := r.Header.Get("Authorization")
				if strings.HasPrefix(authHeader, "Bearer ") {
					providedKey = strings.TrimPrefix(authHeader, "Bearer ")
				}
			}

			// 验证 API Key
			if providedKey == "" || providedKey != apiKey {
				hlog.FromRequest(r).Warn().
					Str("ip", getClientIP(r)).
					Str("path", r.URL.Path).
					Str("method", r.Method).
					Msg("认证失败：无效的 API Key")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// 认证成功，继续处理请求
			next.ServeHTTP(w, r)
		})
	}
}

// OptionalAuthMiddleware 创建可选的 API Key 认证中间件
//
// 与 AuthMiddleware 不同，如果 API Key 为空，该中间件不会拒绝请求。
// 这适用于某些端点需要可选认证的场景。
//
// 参数:
//   - apiKey: API Key 值，如果为空则不进行认证
//
// 返回:
//   - func(http.Handler) http.Handler: HTTP 中间件函数
func OptionalAuthMiddleware(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 如果 API Key 为空，不进行认证
			if apiKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			// 执行认证逻辑（与 AuthMiddleware 相同）
			providedKey := r.Header.Get("X-API-Key")
			if providedKey == "" {
				authHeader := r.Header.Get("Authorization")
				if strings.HasPrefix(authHeader, "Bearer ") {
					providedKey = strings.TrimPrefix(authHeader, "Bearer ")
				}
			}

			if providedKey == "" || providedKey != apiKey {
				hlog.FromRequest(r).Warn().
					Str("ip", getClientIP(r)).
					Str("path", r.URL.Path).
					Str("method", r.Method).
					Msg("认证失败：无效的 API Key")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
