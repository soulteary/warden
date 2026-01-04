// Package metrics 提供了 Prometheus 指标收集功能。
// 包括 HTTP 请求指标、缓存指标、后台任务指标等。
package metrics

import (
	// 标准库
	"net/http"

	// 第三方库
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HTTPRequestTotal 记录 HTTP 请求总数
	HTTPRequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	// HTTPRequestDuration 记录 HTTP 请求延迟
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	// CacheSize 记录缓存中的用户数量
	CacheSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "cache_size",
			Help: "Number of users in cache",
		},
	)

	// BackgroundTaskTotal 记录后台任务执行总数
	BackgroundTaskTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "background_task_total",
			Help: "Total number of background tasks executed",
		},
	)

	// BackgroundTaskDuration 记录后台任务执行时间
	BackgroundTaskDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "background_task_duration_seconds",
			Help:    "Background task duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	// BackgroundTaskErrors 记录后台任务错误数
	BackgroundTaskErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "background_task_errors_total",
			Help: "Total number of background task errors",
		},
	)

	// CacheHits 记录缓存命中次数
	CacheHits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total number of cache hits",
		},
	)

	// CacheMisses 记录缓存未命中次数
	CacheMisses = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total number of cache misses",
		},
	)

	// RateLimitHits 记录速率限制触发次数
	RateLimitHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rate_limit_hits_total",
			Help: "Total number of rate limit hits",
		},
		[]string{"ip"},
	)
)

// Handler 返回 Prometheus metrics 端点处理器
func Handler() http.Handler {
	return promhttp.Handler()
}
