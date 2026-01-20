package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/assert"
)

func TestProcessWithLogger(t *testing.T) {
	// Create test handler
	testData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testData)
	handler := JSON(userCache)
	wrappedHandler := ProcessWithLogger(handler)

	// Create test request
	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	// Execute handler
	wrappedHandler.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code, "状态码应该是200")
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"), "Content-Type应该是application/json")
}

func TestProcessWithLogger_LoggingMiddleware(t *testing.T) {
	// Test if logging middleware is correctly wrapped
	testData := []define.AllowListUser{}
	userCache := cache.NewSafeUserCache()
	userCache.Set(testData)
	handler := JSON(userCache)
	wrappedHandler := ProcessWithLogger(handler)

	// Verify returned type is http.Handler
	assert.NotNil(t, wrappedHandler, "包装后的处理器不应该为nil")
	assert.Implements(t, (*http.Handler)(nil), wrappedHandler, "应该实现http.Handler接口")
}

func TestProcessWithLogger_RequestID(t *testing.T) {
	// Test Request-ID header
	testData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testData)
	handler := JSON(userCache)
	wrappedHandler := ProcessWithLogger(handler)

	req := httptest.NewRequest("GET", "/", http.NoBody)
	req.Header.Set("Request-Id", "test-request-id-123")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProcessWithLogger_UserAgent(t *testing.T) {
	// Test User-Agent header
	testData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testData)
	handler := JSON(userCache)
	wrappedHandler := ProcessWithLogger(handler)

	req := httptest.NewRequest("GET", "/", http.NoBody)
	req.Header.Set("User-Agent", "test-agent/1.0")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProcessWithLogger_Referer(t *testing.T) {
	// Test Referer header
	testData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testData)
	handler := JSON(userCache)
	wrappedHandler := ProcessWithLogger(handler)

	req := httptest.NewRequest("GET", "/", http.NoBody)
	req.Header.Set("Referer", "http://example.com")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProcessWithLogger_RemoteAddr(t *testing.T) {
	// Test RemoteAddr
	testData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testData)
	handler := JSON(userCache)
	wrappedHandler := ProcessWithLogger(handler)

	req := httptest.NewRequest("GET", "/", http.NoBody)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProcessWithLogger_DifferentMethods(t *testing.T) {
	// Test different HTTP methods
	testData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testData)
	handler := JSON(userCache)
	wrappedHandler := ProcessWithLogger(handler)

	// Test allowed method (GET)
	t.Run("GET", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", http.NoBody)
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "GET方法应该返回200")
	})

	// Test disallowed methods (should return 405)
	disallowedMethods := []string{"POST", "PUT", "DELETE", "OPTIONS", "PATCH"}
	for _, method := range disallowedMethods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/", http.NoBody)
			w := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(w, req)

			// Disallowed methods should return 405 Method Not Allowed
			assert.Equal(t, http.StatusMethodNotAllowed, w.Code, "方法%s应该返回405", method)
		})
	}
}

func TestProcessWithLogger_Concurrent(t *testing.T) {
	// Test concurrency safety
	testData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testData)
	handler := JSON(userCache)
	wrappedHandler := ProcessWithLogger(handler)

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/", http.NoBody)
			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)
			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Test passes if no panic occurs
	assert.True(t, true, "并发请求应该安全")
}
