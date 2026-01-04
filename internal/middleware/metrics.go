// Package middleware 提供了 HTTP 中间件功能。
// 包括速率限制、压缩、请求体限制、指标收集等中间件。
package middleware

import (
	// 标准库
	"net/http"
	"strconv"
	"time"

	// 项目内部包
	"soulteary.com/soulteary/warden/internal/metrics"
)

// MetricsMiddleware 创建 Prometheus 指标收集中间件
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 包装 ResponseWriter 以捕获状态码
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		// 记录指标
		duration := time.Since(start).Seconds()
		endpoint := r.URL.Path
		if endpoint == "" {
			endpoint = "/"
		}

		status := strconv.Itoa(rw.statusCode)

		metrics.HTTPRequestTotal.WithLabelValues(r.Method, endpoint, status).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(r.Method, endpoint).Observe(duration)
	})
}

// responseWriter 包装 http.ResponseWriter 以捕获状态码
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
