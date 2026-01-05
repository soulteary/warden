// Package middleware 提供了 HTTP 中间件功能。
// 包括速率限制、压缩、请求体限制、指标收集等中间件。
package middleware

import (
	// 标准库
	"encoding/json"
	"net/http"
	"os"

	// 第三方库
	"github.com/rs/zerolog/hlog"
)

// ErrorResponse 错误响应结构
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// ErrorHandlerMiddleware 创建错误处理中间件
//
// 该中间件在生产环境隐藏详细的错误信息，只返回通用错误消息。
// 详细错误信息仅记录在日志中，不返回给客户端。
//
// 参数:
//   - appMode: 应用模式（"production" 或 "prod" 表示生产环境）
//
// 返回:
//   - func(http.Handler) http.Handler: HTTP 中间件函数
func ErrorHandlerMiddleware(appMode string) func(http.Handler) http.Handler {
	isProduction := appMode == "production" || appMode == "prod"

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 使用自定义的 ResponseWriter 来捕获错误
			rw := &errorResponseWriter{
				ResponseWriter: w,
				isProduction:   isProduction,
				request:        r,
			}
			next.ServeHTTP(rw, r)
		})
	}
}

// errorResponseWriter 自定义 ResponseWriter，用于捕获和修改错误响应
type errorResponseWriter struct {
	http.ResponseWriter
	isProduction bool
	request      *http.Request
	statusCode   int
	body         []byte
}

// WriteHeader 捕获状态码
func (rw *errorResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write 捕获响应体
func (rw *errorResponseWriter) Write(b []byte) (int, error) {
	// 如果是错误响应（4xx 或 5xx），在生产环境可能需要隐藏详细信息
	if rw.statusCode >= 400 && rw.isProduction {
		// 尝试解析 JSON 错误响应
		var errResp ErrorResponse
		if err := json.Unmarshal(b, &errResp); err == nil {
			// 在生产环境，只返回通用错误消息
			genericResp := ErrorResponse{
				Error: getGenericErrorMessage(rw.statusCode),
			}
			// 记录详细错误到日志
			hlog.FromRequest(rw.request).Error().
				Int("status_code", rw.statusCode).
				Str("original_error", errResp.Error).
				Str("original_message", errResp.Message).
				Str("original_code", errResp.Code).
				Msg("错误响应（详细信息已隐藏）")

			// 重新编码通用错误响应
			if newBody, err := json.Marshal(genericResp); err == nil {
				b = newBody
			}
		} else {
			// 如果不是 JSON 格式，也记录原始响应
			hlog.FromRequest(rw.request).Error().
				Int("status_code", rw.statusCode).
				Str("original_response", string(b)).
				Msg("错误响应（详细信息已隐藏）")
			// 返回通用错误消息
			genericResp := ErrorResponse{
				Error: getGenericErrorMessage(rw.statusCode),
			}
			if newBody, err := json.Marshal(genericResp); err == nil {
				b = newBody
			}
		}
	}

	return rw.ResponseWriter.Write(b)
}

// getGenericErrorMessage 根据状态码返回通用错误消息
func getGenericErrorMessage(statusCode int) string {
	switch {
	case statusCode >= 500:
		return "内部服务器错误，请稍后重试"
	case statusCode == 404:
		return "请求的资源不存在"
	case statusCode == 403:
		return "访问被拒绝"
	case statusCode == 401:
		return "未授权访问"
	case statusCode == 400:
		return "请求参数无效"
	case statusCode == 429:
		return "请求过于频繁，请稍后重试"
	default:
		return "请求处理失败"
	}
}

// SafeError 安全地返回错误响应（根据环境决定是否隐藏详细信息）
func SafeError(w http.ResponseWriter, r *http.Request, statusCode int, err error, detailMessage string) {
	appMode := os.Getenv("MODE")
	isProduction := appMode == "production" || appMode == "prod"

	// 记录详细错误到日志
	hlog.FromRequest(r).Error().
		Int("status_code", statusCode).
		Err(err).
		Str("detail", detailMessage).
		Msg("处理请求时发生错误")

	// 构建错误响应
	var resp ErrorResponse
	if isProduction {
		// 生产环境：只返回通用错误消息
		resp = ErrorResponse{
			Error: getGenericErrorMessage(statusCode),
		}
	} else {
		// 开发环境：返回详细错误信息
		resp = ErrorResponse{
			Error:   getGenericErrorMessage(statusCode),
			Message: detailMessage,
		}
		if err != nil {
			resp.Message = err.Error()
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		hlog.FromRequest(r).Error().
			Err(err).
			Msg("编码错误响应失败")
	}
}
