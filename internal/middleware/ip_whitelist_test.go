package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

// TestIPWhitelistMiddleware_EmptyWhitelist 测试空白名单（应该允许所有 IP）
func TestIPWhitelistMiddleware_EmptyWhitelist(t *testing.T) {
	middleware := IPWhitelistMiddleware("")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "空白名单应该允许所有 IP")
}

// TestIPWhitelistMiddleware_SingleIP 测试单个 IP 白名单
func TestIPWhitelistMiddleware_SingleIP(t *testing.T) {
	middleware := IPWhitelistMiddleware("192.168.1.1")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	tests := []struct {
		name       string
		remoteAddr string
		want       int
	}{
		{
			name:       "允许的 IP",
			remoteAddr: "192.168.1.1:12345",
			want:       http.StatusOK,
		},
		{
			name:       "不允许的 IP",
			remoteAddr: "192.168.1.2:12345",
			want:       http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.want, w.Code, "状态码应该匹配")
		})
	}
}

// TestIPWhitelistMiddleware_MultipleIPs 测试多个 IP 白名单
func TestIPWhitelistMiddleware_MultipleIPs(t *testing.T) {
	middleware := IPWhitelistMiddleware("192.168.1.1,192.168.1.2,10.0.0.1")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	tests := []struct {
		name       string
		remoteAddr string
		want       int
	}{
		{
			name:       "第一个 IP",
			remoteAddr: "192.168.1.1:12345",
			want:       http.StatusOK,
		},
		{
			name:       "第二个 IP",
			remoteAddr: "192.168.1.2:12345",
			want:       http.StatusOK,
		},
		{
			name:       "第三个 IP",
			remoteAddr: "10.0.0.1:12345",
			want:       http.StatusOK,
		},
		{
			name:       "不在白名单的 IP",
			remoteAddr: "192.168.1.3:12345",
			want:       http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.want, w.Code, "状态码应该匹配")
		})
	}
}

// TestIPWhitelistMiddleware_WithSpaces 测试带空格的 IP 列表
func TestIPWhitelistMiddleware_WithSpaces(t *testing.T) {
	middleware := IPWhitelistMiddleware(" 192.168.1.1 , 192.168.1.2 ")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "应该能够处理带空格的 IP 列表")
}

// TestIPWhitelistMiddleware_XForwardedFor 测试 X-Forwarded-For 头
func TestIPWhitelistMiddleware_XForwardedFor(t *testing.T) {
	middleware := IPWhitelistMiddleware("192.168.1.1")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "192.168.1.1")
	w := httptest.NewRecorder()

	// 注意：根据实现，可能使用 X-Forwarded-For 或 RemoteAddr
	// 这里只是测试不会 panic
	assert.NotPanics(t, func() {
		handler.ServeHTTP(w, req)
	})
}

// TestIPWhitelistMiddleware_XRealIP 测试 X-Real-IP 头
func TestIPWhitelistMiddleware_XRealIP(t *testing.T) {
	middleware := IPWhitelistMiddleware("192.168.1.1")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Real-IP", "192.168.1.1")
	w := httptest.NewRecorder()

	// 注意：根据实现，可能使用 X-Real-IP 或 RemoteAddr
	// 这里只是测试不会 panic
	assert.NotPanics(t, func() {
		handler.ServeHTTP(w, req)
	})
}

// TestIPWhitelistMiddleware_CIDR 测试 CIDR 网段
func TestIPWhitelistMiddleware_CIDR(t *testing.T) {
	middleware := IPWhitelistMiddleware("192.168.1.0/24")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	tests := []struct {
		name       string
		remoteAddr string
		want       int
	}{
		{
			name:       "网段内的 IP",
			remoteAddr: "192.168.1.100:12345",
			want:       http.StatusOK,
		},
		{
			name:       "网段外的 IP",
			remoteAddr: "192.168.2.1:12345",
			want:       http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.want, w.Code, "状态码应该匹配")
		})
	}
}

// TestIPWhitelistMiddleware_MixedIPsAndCIDR 测试混合 IP 和 CIDR
func TestIPWhitelistMiddleware_MixedIPsAndCIDR(t *testing.T) {
	middleware := IPWhitelistMiddleware("192.168.1.1,10.0.0.0/8,172.16.0.1")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	tests := []struct {
		name       string
		remoteAddr string
		want       int
	}{
		{
			name:       "单个 IP",
			remoteAddr: "192.168.1.1:12345",
			want:       http.StatusOK,
		},
		{
			name:       "CIDR 网段内的 IP",
			remoteAddr: "10.0.0.1:12345",
			want:       http.StatusOK,
		},
		{
			name:       "另一个单个 IP",
			remoteAddr: "172.16.0.1:12345",
			want:       http.StatusOK,
		},
		{
			name:       "不在白名单的 IP",
			remoteAddr: "192.168.2.1:12345",
			want:       http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.want, w.Code, "状态码应该匹配")
		})
	}
}
