package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

// TestSecurityHeadersMiddleware 测试安全响应头设置
func TestSecurityHeadersMiddleware(t *testing.T) {
	middleware := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	}))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// 验证所有安全响应头都被设置
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"), "应该设置 X-Content-Type-Options")
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"), "应该设置 X-Frame-Options")
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"), "应该设置 X-XSS-Protection")
	assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"), "应该设置 Referrer-Policy")
	assert.Contains(t, w.Header().Get("Content-Security-Policy"), "default-src 'self'", "应该设置 Content-Security-Policy")
}

// TestSecurityHeadersMiddleware_AllHeaders 测试所有安全头的内容
func TestSecurityHeadersMiddleware_AllHeaders(t *testing.T) {
	middleware := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	headers := w.Header()

	// 验证 X-Content-Type-Options
	assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"), "X-Content-Type-Options 应该为 nosniff")

	// 验证 X-Frame-Options
	assert.Equal(t, "DENY", headers.Get("X-Frame-Options"), "X-Frame-Options 应该为 DENY")

	// 验证 X-XSS-Protection
	assert.Equal(t, "1; mode=block", headers.Get("X-XSS-Protection"), "X-XSS-Protection 应该为 1; mode=block")

	// 验证 Referrer-Policy
	assert.Equal(t, "strict-origin-when-cross-origin", headers.Get("Referrer-Policy"), "Referrer-Policy 应该为 strict-origin-when-cross-origin")

	// 验证 Content-Security-Policy
	csp := headers.Get("Content-Security-Policy")
	assert.NotEmpty(t, csp, "Content-Security-Policy 不应该为空")
	assert.Contains(t, csp, "default-src 'self'", "CSP 应该包含 default-src 'self'")
	assert.Contains(t, csp, "script-src 'self'", "CSP 应该包含 script-src 'self'")
	assert.Contains(t, csp, "style-src 'self' 'unsafe-inline'", "CSP 应该包含 style-src 'self' 'unsafe-inline'")
}

// TestSecurityHeadersMiddleware_DifferentStatusCodes 测试不同状态码下的安全头
func TestSecurityHeadersMiddleware_DifferentStatusCodes(t *testing.T) {
	statusCodes := []int{
		http.StatusOK,
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusNotFound,
		http.StatusInternalServerError,
	}

	for _, statusCode := range statusCodes {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			middleware := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(statusCode)
			}))

			req := httptest.NewRequest("GET", "/", http.NoBody)
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			// 验证所有状态码都应该设置安全头
			assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"), "所有状态码都应该设置安全头")
			assert.Equal(t, statusCode, w.Code, "状态码应该正确设置")
		})
	}
}

// TestSecurityHeadersMiddleware_Concurrent 测试并发安全性
func TestSecurityHeadersMiddleware_Concurrent(t *testing.T) {
	middleware := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	}))

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/", http.NoBody)
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
			done <- true
		}()
	}

	// 等待所有请求完成
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestSecurityHeadersMiddleware_DoesNotOverride 测试不会覆盖已有的响应头
func TestSecurityHeadersMiddleware_DoesNotOverride(t *testing.T) {
	middleware := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 先设置一些响应头
		w.Header().Set("X-Custom-Header", "custom-value")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// 验证自定义头仍然存在
	assert.Equal(t, "custom-value", w.Header().Get("X-Custom-Header"), "自定义响应头应该保留")
	// 验证安全头也被设置
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"), "安全头也应该被设置")
}

// TestSecurityHeadersMiddleware_CSPContent 测试 CSP 内容完整性
func TestSecurityHeadersMiddleware_CSPContent(t *testing.T) {
	middleware := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	csp := w.Header().Get("Content-Security-Policy")

	// 验证 CSP 包含所有必要的指令
	expectedDirectives := []string{
		"default-src 'self'",
		"script-src 'self'",
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data:",
		"font-src 'self' data:",
	}

	for _, directive := range expectedDirectives {
		assert.Contains(t, csp, directive, "CSP 应该包含指令: %s", directive)
	}
}
