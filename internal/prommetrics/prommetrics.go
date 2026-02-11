// Package prommetrics provides Prometheus metrics collection functionality.
// Includes HTTP request metrics, cache metrics, background task metrics, etc.
package prommetrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	metricskit "github.com/soulteary/metrics-kit"
)

var (
	// Registry is the Prometheus registry for Warden metrics
	Registry *metricskit.Registry

	// Cache holds cache-related metrics
	Cache *metricskit.CacheMetrics

	// RateLimit holds rate limiting metrics
	RateLimit *metricskit.RateLimitMetrics

	// HTTPRequestTotal records total number of HTTP requests
	HTTPRequestTotal *prometheus.CounterVec

	// HTTPRequestDuration records HTTP request latency
	HTTPRequestDuration *prometheus.HistogramVec

	// CacheSize records number of users in cache (alias to Cache.Size)
	CacheSize prometheus.Gauge

	// CacheHits records number of cache hits (alias to Cache.Hits)
	CacheHits prometheus.Counter

	// CacheMisses records number of cache misses (alias to Cache.Misses)
	CacheMisses prometheus.Counter

	// BackgroundTaskTotal records total number of background tasks executed
	BackgroundTaskTotal prometheus.Counter

	// BackgroundTaskDuration records background task execution time
	BackgroundTaskDuration prometheus.Histogram

	// BackgroundTaskErrors records number of background task errors
	BackgroundTaskErrors prometheus.Counter

	// RateLimitHits records number of rate limit hits (legacy, uses ip label)
	RateLimitHits *prometheus.CounterVec
)

func init() {
	Init()
}

// Init initializes all Warden metrics using metrics-kit
func Init() {
	Registry = metricskit.NewRegistry("warden")
	cm := metricskit.NewCommonMetrics(Registry)

	// Cache metrics with "user" subsystem for user cache
	Cache = cm.NewCacheMetrics("user")

	// Rate limit metrics
	RateLimit = cm.NewRateLimitMetrics()

	// HTTP metrics using builder pattern (keep endpoint label for backward compatibility)
	HTTPRequestTotal = Registry.Counter("http_requests_total").
		Help("Total number of HTTP requests").
		Labels("method", "endpoint", "status").
		BuildVec()

	HTTPRequestDuration = Registry.Histogram("http_request_duration_seconds").
		Help("HTTP request duration in seconds").
		Labels("method", "endpoint").
		Buckets(metricskit.HTTPDurationBuckets()).
		BuildVec()

	// Setup cache variable aliases for backward compatibility
	CacheSize = Cache.Size
	CacheHits = Cache.Hits
	CacheMisses = Cache.Misses

	// Background task metrics (create manually to avoid conflict with CommonMetrics)
	BackgroundTaskTotal = Registry.Counter("background_task_total").
		Help("Total number of background tasks executed").
		Build()

	BackgroundTaskDuration = Registry.Histogram("background_task_duration_seconds").
		Help("Background task duration in seconds").
		Buckets(metricskit.DefaultBuckets()).
		Build()

	BackgroundTaskErrors = Registry.Counter("background_task_errors_total").
		Help("Total number of background task errors").
		Build()

	// Rate limit legacy alias (uses ip label instead of scope for backward compatibility)
	RateLimitHits = Registry.Counter("rate_limit_hits_legacy_total").
		Help("Total number of rate limit hits (legacy, by IP)").
		Labels("ip").
		BuildVec()
}

// Handler returns Prometheus metrics endpoint handler
func Handler() http.Handler {
	return metricskit.HandlerFor(Registry)
}

// RecordHTTPRequest records an HTTP request
func RecordHTTPRequest(method, endpoint, status string, duration time.Duration) {
	HTTPRequestTotal.WithLabelValues(method, endpoint, status).Inc()
	HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

// RecordCacheHit records a cache hit
func RecordCacheHit() {
	Cache.RecordHit()
}

// RecordCacheMiss records a cache miss
func RecordCacheMiss() {
	Cache.RecordMiss()
}

// SetCacheSize sets the current cache size
func SetCacheSize(size float64) {
	Cache.SetSize(size)
}

// RecordBackgroundTask records a background task execution
func RecordBackgroundTask(duration time.Duration, success bool) {
	BackgroundTaskTotal.Inc()
	BackgroundTaskDuration.Observe(duration.Seconds())
	if !success {
		BackgroundTaskErrors.Inc()
	}
}

// RecordRateLimitHit records a rate limit hit by IP (legacy)
func RecordRateLimitHit(ip string) {
	RateLimitHits.WithLabelValues(ip).Inc()
	// Also record in the new metrics with "ip" scope
	RateLimit.RecordHit("ip")
}
