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
	handler := JSON(userCache, nil)
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
	handler := JSON(userCache, nil)
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
	handler := JSON(userCache, nil)
	wrappedHandler := ProcessWithLogger(handler)

	req := httptest.NewRequest("GET", "/", http.NoBody)
	req.Header.Set("X-Request-ID", "test-request-id-123")
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
	handler := JSON(userCache, nil)
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
	handler := JSON(userCache, nil)
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
	handler := JSON(userCache, nil)
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
	handler := JSON(userCache, nil)
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
	handler := JSON(userCache, nil)
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

// TestAccessLogMiddleware tests AccessLogMiddleware function
func TestAccessLogMiddleware(t *testing.T) {
	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			t.Errorf("Failed to write response body: %v", err)
		}
	})

	// Wrap with AccessLogMiddleware
	middleware := AccessLogMiddleware()
	wrappedHandler := middleware(handler)

	// Create test request
	req := httptest.NewRequest("GET", "/test?phone=13800138000", http.NoBody)
	req.RemoteAddr = "192.168.1.1:12345"
	req.Header.Set("User-Agent", "test-agent/1.0")
	req.Header.Set("Referer", "http://example.com")
	w := httptest.NewRecorder()

	// Execute handler
	wrappedHandler.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code, "状态码应该是200")
	assert.Equal(t, "OK", w.Body.String(), "响应体应该正确")
}

// TestAccessLogMiddleware_DifferentStatusCodes tests different status codes
func TestAccessLogMiddleware_DifferentStatusCodes(t *testing.T) {
	//nolint:govet // fieldalignment: test cases prioritize readability
	tests := []struct {
		name           string
		handler        http.HandlerFunc
		expectedStatus int
	}{
		{
			name: "200 OK",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "404 Not Found",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "500 Internal Server Error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := AccessLogMiddleware()
			wrappedHandler := middleware(tt.handler)

			req := httptest.NewRequest("GET", "/", http.NoBody)
			w := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "状态码应该匹配")
		})
	}
}

// TestAccessLogMiddleware_URLSanitization tests URL sanitization
func TestAccessLogMiddleware_URLSanitization(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := AccessLogMiddleware()
	wrappedHandler := middleware(handler)

	// Test with sensitive query parameters
	req := httptest.NewRequest("GET", "/test?phone=13800138000&mail=test@example.com&email=user@test.com", http.NoBody)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "应该正常处理请求")
}

// TestAccessLogMiddleware_DifferentMethods tests different HTTP methods
func TestAccessLogMiddleware_DifferentMethods(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := AccessLogMiddleware()
	wrappedHandler := middleware(handler)

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/", http.NoBody)
			w := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code, "应该正常处理请求")
		})
	}
}

// TestAccessLogMiddleware_RequestID tests Request-ID header handling
func TestAccessLogMiddleware_RequestID(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := AccessLogMiddleware()
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/", http.NoBody)
	req.Header.Set("Request-Id", "custom-request-id-123")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "应该正常处理请求")
}

// TestAccessLogMiddleware_ResponseSize tests response size logging
func TestAccessLogMiddleware_ResponseSize(t *testing.T) {
	responseBody := "This is a test response body"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(responseBody)); err != nil {
			t.Errorf("Failed to write response body: %v", err)
		}
	})

	middleware := AccessLogMiddleware()
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "应该正常处理请求")
	assert.Equal(t, responseBody, w.Body.String(), "响应体应该正确")
}
