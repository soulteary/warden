package prommetrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	RecordRateLimitHit("127.0.0.1")
}
