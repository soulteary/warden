package metrics

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

	// 创建一个测试请求
	req, err := http.NewRequest("GET", "/metrics", nil)
	require.NoError(t, err)

	// 创建响应记录器
	rr := httptest.NewRecorder()

	// 调用处理器
	handler.ServeHTTP(rr, req)

	// 验证响应状态码
	assert.Equal(t, http.StatusOK, rr.Code)

	// 验证响应体包含Prometheus指标格式
	body := rr.Body.String()
	assert.Contains(t, body, "# HELP", "响应应该包含Prometheus指标说明")
	assert.Contains(t, body, "# TYPE", "响应应该包含Prometheus指标类型")
}

func TestHandler_MultipleRequests(t *testing.T) {
	handler := Handler()
	require.NotNil(t, handler)

	// 发送多个请求
	for i := 0; i < 5; i++ {
		req, err := http.NewRequest("GET", "/metrics", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	}
}

func TestHandler_ConcurrentRequests(t *testing.T) {
	handler := Handler()
	require.NotNil(t, handler)

	// 并发发送请求
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			req, err := http.NewRequest("GET", "/metrics", nil)
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

	// 等待所有请求完成
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

	req, err := http.NewRequest("GET", "/metrics", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// 验证响应头
	assert.Equal(t, http.StatusOK, rr.Code)
	// Prometheus metrics端点通常返回text/plain
	assert.Contains(t, rr.Header().Get("Content-Type"), "text/plain", "Content-Type应该是text/plain")
}

func TestHandler_InvalidMethod(t *testing.T) {
	handler := Handler()
	require.NotNil(t, handler)

	// 测试POST请求（虽然Prometheus通常只使用GET）
	req, err := http.NewRequest("POST", "/metrics", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Prometheus handler通常允许POST，但这里我们只验证不会panic
	assert.NotPanics(t, func() {
		handler.ServeHTTP(rr, req)
	})
}

func TestMetrics_VariablesInitialized(t *testing.T) {
	// 验证所有指标变量都已初始化
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
