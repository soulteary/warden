// Package prommetrics provides Prometheus metrics collection functionality.
// Includes HTTP request metrics, cache metrics, background task metrics, etc.
package prommetrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	metricskit "github.com/soulteary/metrics-kit/v3"

	"github.com/soulteary/warden/internal/define"
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

	// RateLimitHits records number of rate limit hits. The legacy metric name is
	// kept for dashboard compatibility, but the high-cardinality per-IP label has
	// been removed (see below) to avoid unbounded time series.
	RateLimitHits prometheus.Counter

	// RedisSnapshotOperationsTotal records bounded migration operation outcomes.
	RedisSnapshotOperationsTotal *prometheus.CounterVec

	// SnapshotAgeSeconds records the age of the current rule-set snapshot in seconds.
	SnapshotAgeSeconds prometheus.Gauge

	// SnapshotDegraded is 1 when the current snapshot is degraded (served from
	// local/last-known-good after a remote failure), 0 otherwise.
	SnapshotDegraded prometheus.Gauge

	// RefreshFailuresTotal counts background refresh failures by non-sensitive reason code.
	RefreshFailuresTotal *prometheus.CounterVec

	// RemoteFallbackTotal counts remote-failure fallbacks by merge mode and reason code.
	RemoteFallbackTotal *prometheus.CounterVec

	// DeprecationTotal counts uses of deprecated features by a stable, low-cardinality
	// feature label (e.g. "encryption_legacy", "mode_legacy_env", "hmac_v1").
	DeprecationTotal *prometheus.CounterVec

	// HMACReplayRejectedTotal counts HMAC v2 requests rejected because their nonce was
	// already seen within the timestamp window (replay protection).
	HMACReplayRejectedTotal prometheus.Counter
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

	// Rate limit legacy metric. The historical name is retained but the per-IP label
	// is intentionally dropped: a per-client-IP label is unbounded (one series per
	// source address) and both a cardinality and a mild privacy concern. Aggregate
	// per-scope counting still happens via RateLimit.RecordHit below.
	RateLimitHits = Registry.Counter("rate_limit_hits_legacy_total").
		Help("Total number of rate limit hits (legacy; no per-IP label)").
		Build()

	// Snapshot / fallback observability (low-cardinality labels only).
	RedisSnapshotOperationsTotal = Registry.Counter("redis_snapshot_operations_total").
		Help("Versioned Redis snapshot operations by migration mode and outcome").
		Labels("mode", "operation", "result").BuildVec()
	SnapshotAgeSeconds = Registry.Gauge("snapshot_age_seconds").
		Help("Age in seconds of the current rule-set snapshot").
		Build()

	SnapshotDegraded = Registry.Gauge("snapshot_degraded").
		Help("Whether the current snapshot is degraded (1) or healthy (0)").
		Build()

	RefreshFailuresTotal = Registry.Counter("refresh_failures_total").
		Help("Total number of background refresh failures by reason").
		Labels("reason").
		BuildVec()

	RemoteFallbackTotal = Registry.Counter("remote_fallback_total").
		Help("Total number of remote-failure fallbacks by mode and reason").
		Labels("mode", "reason").
		BuildVec()

	DeprecationTotal = Registry.Counter("deprecation_total").
		Help("Total uses of deprecated features by feature code").
		Labels("feature").
		BuildVec()

	HMACReplayRejectedTotal = Registry.Counter("hmac_replay_rejected_total").
		Help("Total HMAC v2 requests rejected due to nonce replay within the window").
		Build()
}

// Handler returns Prometheus metrics endpoint handler
func Handler() http.Handler {
	return metricskit.HandlerFor(Registry)
}

// RecordHTTPRequest records an HTTP request
func RecordHTTPRequest(method, endpoint, status string, duration time.Duration) {
	// Normalize here as well as in the middleware so the bounded-cardinality invariant
	// belongs to the metric rather than to one call site: method and endpoint both derive
	// from attacker-controlled request data.
	method = define.NormalizeMethodLabel(method)
	endpoint = define.NormalizeEndpointLabel(endpoint)
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

// RecordRateLimitHit records a refused request. scope is "ip" for the
// per-client-address limit and "credential" for the per-credential limit; any
// other value is counted as "ip" so the label set stays fixed. Client addresses
// and credential identifiers are never used as label values.
func RecordRateLimitHit(scope string) {
	if scope != "credential" {
		scope = "ip"
	}
	RateLimitHits.Inc()
	RateLimit.RecordHit(scope)
}

// RecordDeprecation increments the deprecation counter for a stable feature code.
func RecordDeprecation(feature string) {
	DeprecationTotal.WithLabelValues(feature).Inc()
}

// RecordHMACReplayRejected increments the HMAC replay-rejection counter.
func RecordHMACReplayRejected() {
	HMACReplayRejectedTotal.Inc()
}

// RecordRedisSnapshotResult accepts only fixed labels, never keys or payloads.
func RecordRedisSnapshotResult(mode, operation, result string) {
	switch mode {
	case "shadow", "v2":
	default:
		mode = "unknown"
	}
	switch operation {
	case "read", "follow_read", "publish", "renew", "lease_acquire", "lease_release":
	default:
		operation = "unknown"
	}
	switch result {
	case "ineligible", "superseded", "restored", "published", "renewed", "acquired", "released", "busy", "no_lease", "unavailable", "unknown_source_time", "legacy_write_error", "missing", "schema_rejected", "dataset_rejected", "policy_rejected", "timestamp_rejected", "too_large", "invalid_envelope", "contract_rejected", "effective_rejected", "lease_lost", "revision_conflict", "io_error":
	default:
		result = "unknown"
	}
	RedisSnapshotOperationsTotal.WithLabelValues(mode, operation, result).Inc()
}
