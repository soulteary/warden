package prommetrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/define"
)

func TestHandler(t *testing.T) {
	handler := Handler()
	require.NotNil(t, handler)

	// Create a test request
	req, err := http.NewRequest("GET", "/metrics", http.NoBody)
	require.NoError(t, err)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call handler
	handler.ServeHTTP(rr, req)

	// Verify response status code
	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify response body contains Prometheus metrics format
	body := rr.Body.String()
	assert.Contains(t, body, "# HELP", "响应应该包含Prometheus指标说明")
	assert.Contains(t, body, "# TYPE", "响应应该包含Prometheus指标类型")
}

func TestHandler_MultipleRequests(t *testing.T) {
	handler := Handler()
	require.NotNil(t, handler)

	// Send multiple requests
	for i := 0; i < 5; i++ {
		req, err := http.NewRequest("GET", "/metrics", http.NoBody)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	}
}

func TestHandler_ConcurrentRequests(t *testing.T) {
	handler := Handler()
	require.NotNil(t, handler)

	// Send requests concurrently
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			req, err := http.NewRequest("GET", "/metrics", http.NoBody)
			if err != nil {
				done <- false
				return
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code == http.StatusOK {
				done <- true
			} else {
				done <- false
			}
		}()
	}

	// Wait for all requests to complete
	successCount := 0
	for i := 0; i < 10; i++ {
		if <-done {
			successCount++
		}
	}

	assert.Equal(t, 10, successCount, "所有并发请求应该成功")
}

func TestHandler_ResponseHeaders(t *testing.T) {
	handler := Handler()
	require.NotNil(t, handler)

	req, err := http.NewRequest("GET", "/metrics", http.NoBody)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Verify response headers
	assert.Equal(t, http.StatusOK, rr.Code)
	// Prometheus metrics endpoint usually returns text/plain
	assert.Contains(t, rr.Header().Get("Content-Type"), "text/plain", "Content-Type应该是text/plain")
}

func TestHandler_InvalidMethod(t *testing.T) {
	handler := Handler()
	require.NotNil(t, handler)

	// Test POST request (although Prometheus usually only uses GET)
	req, err := http.NewRequest("POST", "/metrics", http.NoBody)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Prometheus handler usually allows POST, but here we only verify it doesn't panic
	assert.NotPanics(t, func() {
		handler.ServeHTTP(rr, req)
	})
}

func TestMetrics_VariablesInitialized(t *testing.T) {
	// Verify all metrics variables are initialized
	assert.NotNil(t, HTTPRequestTotal, "HTTPRequestTotal应该已初始化")
	assert.NotNil(t, HTTPRequestDuration, "HTTPRequestDuration应该已初始化")
	assert.NotNil(t, CacheSize, "CacheSize应该已初始化")
	assert.NotNil(t, BackgroundTaskTotal, "BackgroundTaskTotal应该已初始化")
	assert.NotNil(t, BackgroundTaskDuration, "BackgroundTaskDuration应该已初始化")
	assert.NotNil(t, BackgroundTaskErrors, "BackgroundTaskErrors应该已初始化")
	assert.NotNil(t, CacheHits, "CacheHits应该已初始化")
	assert.NotNil(t, CacheMisses, "CacheMisses应该已初始化")
	assert.NotNil(t, RateLimitHits, "RateLimitHits应该已初始化")
}

// TestRecordFunctions covers RecordHTTPRequest, RecordCacheHit, RecordCacheMiss,
// SetCacheSize, RecordBackgroundTask, RecordRateLimitHit (no panic, metrics increment)
func TestRecordHTTPRequest(t *testing.T) {
	RecordHTTPRequest("GET", "/api", "200", 0)
	RecordHTTPRequest("POST", "/user", "201", 0)
}

func TestRecordCacheHit(t *testing.T) {
	RecordCacheHit()
}

func TestRecordCacheMiss(t *testing.T) {
	RecordCacheMiss()
}

func TestSetCacheSize(t *testing.T) {
	SetCacheSize(0)
	SetCacheSize(100)
}

func TestRecordBackgroundTask(t *testing.T) {
	RecordBackgroundTask(0, true)
	RecordBackgroundTask(0, false)
}

func TestRecordRateLimitHit(t *testing.T) {
	value := func(c prometheus.Counter) float64 {
		metric := &dto.Metric{}
		require.NoError(t, c.Write(metric))
		return metric.GetCounter().GetValue()
	}
	hits := func(scope string) float64 {
		return value(RateLimit.Hits.WithLabelValues(scope))
	}
	ipBefore, credentialBefore, legacyBefore := hits("ip"), hits("credential"), value(RateLimitHits)

	RecordRateLimitHit("ip")
	RecordRateLimitHit("credential")
	// Anything else, such as a client address, must not become a label value.
	RecordRateLimitHit("127.0.0.1")

	if got := hits("ip") - ipBefore; got != 2 {
		t.Errorf("ip scope increased by %v, want 2", got)
	}
	if got := hits("credential") - credentialBefore; got != 1 {
		t.Errorf("credential scope increased by %v, want 1", got)
	}
	if got := value(RateLimitHits) - legacyBefore; got != 3 {
		t.Errorf("legacy counter increased by %v, want 3", got)
	}
	series := make(chan prometheus.Metric, 8)
	RateLimit.Hits.Collect(series)
	if got := len(series); got != 2 {
		t.Errorf("rate limit hit series = %d, want 2 (ip, credential)", got)
	}
}

// TestRecordHTTPRequest_NormalizesLabels ensures the exported helper cannot be used to
// bypass the label allowlist. It currently has no production caller, which is exactly why
// the invariant has to live inside it rather than at a single call site.
func TestRecordHTTPRequest_NormalizesLabels(t *testing.T) {
	const junkPath = "warden-record-probe-path-1f7a"
	const junkMethod = "WARDENRECORDPROBE"

	RecordHTTPRequest(junkMethod, "/"+junkPath, "404", time.Millisecond)

	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody))
	body := rec.Body.String()

	assert.NotContains(t, body, junkPath, "请求路径不得进入指标标签")
	assert.NotContains(t, body, junkMethod, "请求方法不得进入指标标签")
	assert.Contains(t, body, `endpoint="`+define.LABEL_OTHER+`"`)
	assert.Contains(t, body, `method="`+define.LABEL_OTHER+`"`)
}

func TestRedisSnapshotResultLabelsAreBounded(t *testing.T) {
	Init()
	RecordRedisSnapshotResult("dataset-secret", "payload-secret", "raw-error-secret")
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody))
	require.Contains(t, rec.Body.String(), `warden_redis_snapshot_operations_total{mode="unknown",operation="unknown",result="unknown"} 1`)
	require.NotContains(t, rec.Body.String(), "secret")
}

// The deprecation counter is labelled by feature, and the replay counter
// counts each rejection once.
func TestRecordDeprecationAndReplayRejection(t *testing.T) {
	value := func(c prometheus.Counter) float64 {
		metric := &dto.Metric{}
		require.NoError(t, c.Write(metric))
		return metric.GetCounter().GetValue()
	}
	v1Before := value(DeprecationTotal.WithLabelValues("hmac_v1"))
	legacyBefore := value(DeprecationTotal.WithLabelValues("encryption_legacy"))
	replayBefore := value(HMACReplayRejectedTotal)

	RecordDeprecation("hmac_v1")
	RecordDeprecation("hmac_v1")
	RecordHMACReplayRejected()

	require.InDelta(t, 2, value(DeprecationTotal.WithLabelValues("hmac_v1"))-v1Before, 0)
	require.InDelta(t, 0, value(DeprecationTotal.WithLabelValues("encryption_legacy"))-legacyBefore, 0)
	require.InDelta(t, 1, value(HMACReplayRejectedTotal)-replayBefore, 0)
}
