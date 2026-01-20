package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func init() {
	// 初始化 zerolog 以避免 panic
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

// TestAuthMiddleware_ValidAPIKey 测试有效的 API Key
func TestAuthMiddleware_ValidAPIKey(t *testing.T) {
	apiKey := "test-api-key-123"
	middleware := AuthMiddleware(apiKey)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	tests := []struct {
		name   string
		header string
		value  string
		want   int
	}{
		{
			name:   "X-API-Key header",
			header: "X-API-Key",
			value:  apiKey,
			want:   http.StatusOK,
		},
		{
			name:   "Authorization Bearer header",
			header: "Authorization",
			value:  "Bearer " + apiKey,
			want:   http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set(tt.header, tt.value)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.want, w.Code, "应该返回 200")
		})
	}
}

// TestAuthMiddleware_InvalidAPIKey 测试无效的 API Key
func TestAuthMiddleware_InvalidAPIKey(t *testing.T) {
	apiKey := "test-api-key-123"
	middleware := AuthMiddleware(apiKey)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name   string
		header string
		value  string
	}{
		{
			name:   "错误的 X-API-Key",
			header: "X-API-Key",
			value:  "wrong-key",
		},
		{
			name:   "错误的 Authorization Bearer",
			header: "Authorization",
			value:  "Bearer wrong-key",
		},
		{
			name:   "空的 X-API-Key",
			header: "X-API-Key",
			value:  "",
		},
		{
			name:   "缺少 header",
			header: "",
			value:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.header != "" {
				req.Header.Set(tt.header, tt.value)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code, "应该返回 401")
		})
	}
}

// TestAuthMiddleware_EmptyAPIKey 测试空的 API Key（应该拒绝所有请求）
func TestAuthMiddleware_EmptyAPIKey(t *testing.T) {
	middleware := AuthMiddleware("")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "空 API Key 应该拒绝所有请求")
}

// TestOptionalAuthMiddleware_ValidAPIKey 测试可选认证中间件（有效 API Key）
func TestOptionalAuthMiddleware_ValidAPIKey(t *testing.T) {
	apiKey := "test-api-key-123"
	middleware := OptionalAuthMiddleware(apiKey)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", apiKey)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "有效 API Key 应该通过")
}

// TestOptionalAuthMiddleware_InvalidAPIKey 测试可选认证中间件（无效 API Key）
func TestOptionalAuthMiddleware_InvalidAPIKey(t *testing.T) {
	apiKey := "test-api-key-123"
	middleware := OptionalAuthMiddleware(apiKey)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "wrong-key")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "无效 API Key 应该返回 401")
}

// TestOptionalAuthMiddleware_EmptyAPIKey 测试可选认证中间件（空 API Key，应该允许通过）
func TestOptionalAuthMiddleware_EmptyAPIKey(t *testing.T) {
	middleware := OptionalAuthMiddleware("")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "空 API Key 时应该允许通过（可选认证）")
}

// TestAuthMiddleware_AuthorizationHeaderPrecedence 测试 Authorization 头的优先级
func TestAuthMiddleware_AuthorizationHeaderPrecedence(t *testing.T) {
	apiKey := "test-api-key-123"
	middleware := AuthMiddleware(apiKey)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	// 同时提供 X-API-Key 和 Authorization，X-API-Key 应该优先
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Authorization", "Bearer wrong-key")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "X-API-Key 应该优先于 Authorization")
}

// TestAuthMiddleware_BearerTokenFormat 测试 Bearer token 格式
func TestAuthMiddleware_BearerTokenFormat(t *testing.T) {
	apiKey := "test-api-key-123"
	middleware := AuthMiddleware(apiKey)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name  string
		value string
		want  int
	}{
		{
			name:  "正确的 Bearer 格式",
			value: "Bearer test-api-key-123",
			want:  http.StatusOK,
		},
		{
			name:  "Bearer 后有多余空格",
			value: "Bearer  test-api-key-123",
			want:  http.StatusUnauthorized, // 应该失败，因为有多余空格
		},
		{
			name:  "小写 bearer",
			value: "bearer test-api-key-123",
			want:  http.StatusUnauthorized, // 应该失败，因为不是 "Bearer "
		},
		{
			name:  "没有 Bearer 前缀",
			value: "test-api-key-123",
			want:  http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Authorization", tt.value)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.want, w.Code, "状态码应该匹配")
		})
	}
}
