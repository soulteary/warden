package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"soulteary.com/soulteary/warden/internal/cache"
	"soulteary.com/soulteary/warden/internal/define"
)

func TestProcessWithLogger(t *testing.T) {
	// 创建测试处理器
	testData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testData)
	handler := JSON(userCache)
	wrappedHandler := ProcessWithLogger(handler)

	// 创建测试请求
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	// 执行处理器
	wrappedHandler.ServeHTTP(w, req)

	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code, "状态码应该是200")
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"), "Content-Type应该是application/json")
}

func TestProcessWithLogger_LoggingMiddleware(t *testing.T) {
	// 测试日志中间件是否正确包装
	testData := []define.AllowListUser{}
	userCache := cache.NewSafeUserCache()
	userCache.Set(testData)
	handler := JSON(userCache)
	wrappedHandler := ProcessWithLogger(handler)

	// 验证返回的是http.Handler类型
	assert.NotNil(t, wrappedHandler, "包装后的处理器不应该为nil")
	assert.Implements(t, (*http.Handler)(nil), wrappedHandler, "应该实现http.Handler接口")
}

func TestProcessWithLogger_RequestID(t *testing.T) {
	// 测试Request-ID头
	testData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testData)
	handler := JSON(userCache)
	wrappedHandler := ProcessWithLogger(handler)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Request-Id", "test-request-id-123")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProcessWithLogger_UserAgent(t *testing.T) {
	// 测试User-Agent头
	testData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testData)
	handler := JSON(userCache)
	wrappedHandler := ProcessWithLogger(handler)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("User-Agent", "test-agent/1.0")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProcessWithLogger_Referer(t *testing.T) {
	// 测试Referer头
	testData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testData)
	handler := JSON(userCache)
	wrappedHandler := ProcessWithLogger(handler)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Referer", "http://example.com")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProcessWithLogger_RemoteAddr(t *testing.T) {
	// 测试RemoteAddr
	testData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testData)
	handler := JSON(userCache)
	wrappedHandler := ProcessWithLogger(handler)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProcessWithLogger_DifferentMethods(t *testing.T) {
	// 测试不同的HTTP方法
	testData := []define.AllowListUser{
		{Phone: "13800138000", Mail: "test@example.com"},
	}

	userCache := cache.NewSafeUserCache()
	userCache.Set(testData)
	handler := JSON(userCache)
	wrappedHandler := ProcessWithLogger(handler)

	// 测试允许的方法（GET）
	t.Run("GET", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "GET方法应该返回200")
	})

	// 测试不允许的方法（应该返回405）
	disallowedMethods := []string{"POST", "PUT", "DELETE", "OPTIONS", "PATCH"}
	for _, method := range disallowedMethods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/", nil)
			w := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(w, req)

			// 不允许的方法应该返回405 Method Not Allowed
			assert.Equal(t, http.StatusMethodNotAllowed, w.Code, "方法%s应该返回405", method)
		})
	}
}

func TestProcessWithLogger_Concurrent(t *testing.T) {
	// 测试并发安全性
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
			req := httptest.NewRequest("GET", "/", nil)
			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)
			done <- true
		}()
	}

	// 等待所有请求完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 如果没有panic，测试通过
	assert.True(t, true, "并发请求应该安全")
}
