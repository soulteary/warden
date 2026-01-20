// Package metrics provides Prometheus metrics collection functionality.
// Includes HTTP request metrics, cache metrics, background task metrics, etc.
package metrics

import (
	// Standard library
	"net/http"

	// Third-party libraries
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HTTPRequestTotal records total number of HTTP requests
	HTTPRequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	// HTTPRequestDuration records HTTP request latency
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	// CacheSize records number of users in cache
	CacheSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "cache_size",
			Help: "Number of users in cache",
		},
	)

	// BackgroundTaskTotal records total number of background tasks executed
	BackgroundTaskTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "background_task_total",
			Help: "Total number of background tasks executed",
		},
	)

	// BackgroundTaskDuration records background task execution time
	BackgroundTaskDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "background_task_duration_seconds",
			Help:    "Background task duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	// BackgroundTaskErrors records number of background task errors
	BackgroundTaskErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "background_task_errors_total",
			Help: "Total number of background task errors",
		},
	)

	// CacheHits records number of cache hits
	CacheHits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total number of cache hits",
		},
	)

	// CacheMisses records number of cache misses
	CacheMisses = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total number of cache misses",
		},
	)

	// RateLimitHits records number of rate limit hits
	RateLimitHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rate_limit_hits_total",
			Help: "Total number of rate limit hits",
		},
		[]string{"ip"},
	)
)

// Handler returns Prometheus metrics endpoint handler
func Handler() http.Handler {
	return promhttp.Handler()
}
