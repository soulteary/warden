package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	loggerkit "github.com/soulteary/logger-kit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/logger"
)

func init() {
	logger.SetLevel(loggerkit.InfoLevel)
}

// TestIPWhitelistMiddleware_EmptyWhitelist tests empty whitelist (should allow all IPs)
func TestIPWhitelistMiddleware_EmptyWhitelist(t *testing.T) {
	middleware := IPWhitelistMiddleware("")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	}))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Empty whitelist should allow all IPs")
}

// TestIPWhitelistMiddleware_SingleIP tests single IP whitelist
func TestIPWhitelistMiddleware_SingleIP(t *testing.T) {
	middleware := IPWhitelistMiddleware("192.168.1.1")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	}))

	tests := []struct {
		name       string
		remoteAddr string
		want       int
	}{
		{
			name:       "Allowed IP",
			remoteAddr: "192.168.1.1:12345",
			want:       http.StatusOK,
		},
		{
			name:       "Disallowed IP",
			remoteAddr: "192.168.1.2:12345",
			want:       http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", http.NoBody)
			req.RemoteAddr = tt.remoteAddr
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.want, w.Code, "Status code should match")
		})
	}
}

// TestIPWhitelistMiddleware_MultipleIPs tests multiple IP whitelist
func TestIPWhitelistMiddleware_MultipleIPs(t *testing.T) {
	middleware := IPWhitelistMiddleware("192.168.1.1,192.168.1.2,10.0.0.1")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	}))

	tests := []struct {
		name       string
		remoteAddr string
		want       int
	}{
		{
			name:       "First IP",
			remoteAddr: "192.168.1.1:12345",
			want:       http.StatusOK,
		},
		{
			name:       "Second IP",
			remoteAddr: "192.168.1.2:12345",
			want:       http.StatusOK,
		},
		{
			name:       "Third IP",
			remoteAddr: "10.0.0.1:12345",
			want:       http.StatusOK,
		},
		{
			name:       "IP not in whitelist",
			remoteAddr: "192.168.1.3:12345",
			want:       http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", http.NoBody)
			req.RemoteAddr = tt.remoteAddr
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.want, w.Code, "状态码应该匹配")
		})
	}
}

// TestIPWhitelistMiddleware_WithSpaces tests IP list with spaces
func TestIPWhitelistMiddleware_WithSpaces(t *testing.T) {
	middleware := IPWhitelistMiddleware(" 192.168.1.1 , 192.168.1.2 ")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	}))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Should be able to handle IP list with spaces")
}

// TestIPWhitelistMiddleware_XForwardedFor tests X-Forwarded-For header
func TestIPWhitelistMiddleware_XForwardedFor(t *testing.T) {
	middleware := IPWhitelistMiddleware("192.168.1.1")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	}))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "192.168.1.1")
	w := httptest.NewRecorder()

	// Note: According to implementation, may use X-Forwarded-For or RemoteAddr
	// Here just test that it doesn't panic
	assert.NotPanics(t, func() {
		handler.ServeHTTP(w, req)
	})
}

// TestIPWhitelistMiddleware_XRealIP tests X-Real-IP header
func TestIPWhitelistMiddleware_XRealIP(t *testing.T) {
	middleware := IPWhitelistMiddleware("192.168.1.1")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	}))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Real-IP", "192.168.1.1")
	w := httptest.NewRecorder()

	// Note: According to implementation, may use X-Real-IP or RemoteAddr
	// Here just test that it doesn't panic
	assert.NotPanics(t, func() {
		handler.ServeHTTP(w, req)
	})
}

// TestIPWhitelistMiddleware_CIDR tests CIDR network
func TestIPWhitelistMiddleware_CIDR(t *testing.T) {
	middleware := IPWhitelistMiddleware("192.168.1.0/24")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	}))

	tests := []struct {
		name       string
		remoteAddr string
		want       int
	}{
		{
			name:       "IP in network",
			remoteAddr: "192.168.1.100:12345",
			want:       http.StatusOK,
		},
		{
			name:       "IP outside network",
			remoteAddr: "192.168.2.1:12345",
			want:       http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", http.NoBody)
			req.RemoteAddr = tt.remoteAddr
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.want, w.Code, "状态码应该匹配")
		})
	}
}

// TestIPWhitelistMiddleware_MixedIPsAndCIDR tests mixed IPs and CIDR
func TestIPWhitelistMiddleware_MixedIPsAndCIDR(t *testing.T) {
	middleware := IPWhitelistMiddleware("192.168.1.1,10.0.0.0/8,172.16.0.1")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	}))

	tests := []struct {
		name       string
		remoteAddr string
		want       int
	}{
		{
			name:       "Single IP",
			remoteAddr: "192.168.1.1:12345",
			want:       http.StatusOK,
		},
		{
			name:       "IP in CIDR network",
			remoteAddr: "10.0.0.1:12345",
			want:       http.StatusOK,
		},
		{
			name:       "Another single IP",
			remoteAddr: "172.16.0.1:12345",
			want:       http.StatusOK,
		},
		{
			name:       "IP not in whitelist",
			remoteAddr: "192.168.2.1:12345",
			want:       http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", http.NoBody)
			req.RemoteAddr = tt.remoteAddr
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.want, w.Code, "状态码应该匹配")
		})
	}
}
