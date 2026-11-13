package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	middlewarekit "github.com/soulteary/middleware-kit/v3"
	"github.com/stretchr/testify/require"
)

func TestNewTrustedProxyConfig(t *testing.T) {
	clientIP := func(raw, remoteAddr string) string {
		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		req.RemoteAddr = remoteAddr
		req.Header.Set("X-Forwarded-For", "203.0.113.5")
		req.Header.Set("X-Real-IP", "203.0.113.6")
		return middlewarekit.GetClientIP(req, NewTrustedProxyConfig(raw))
	}

	// Empty: forwarded headers are honoured from private and loopback peers only.
	require.Equal(t, "203.0.113.5", clientIP("", "10.0.0.7:1234"))
	require.Equal(t, "203.0.113.5", clientIP("", "127.0.0.1:1234"))
	require.Equal(t, "198.51.100.9", clientIP("", "198.51.100.9:1234"))

	// A list: only its members are trusted.
	require.Equal(t, "203.0.113.5", clientIP("10.0.0.7", "10.0.0.7:1234"))
	require.Equal(t, "10.0.0.8", clientIP("10.0.0.7", "10.0.0.8:1234"))

	// none: the client address is the TCP peer, whatever its address.
	for _, raw := range []string{"none", " None ", "NONE,"} {
		for _, peer := range []string{"10.0.0.7", "172.16.0.1", "192.168.1.1", "127.0.0.1", "198.51.100.9", "::1", "fd00::1", "fe80::1"} {
			remoteAddr := "[" + peer + "]:1234"
			require.Equal(t, peer, clientIP(raw, remoteAddr), "%q peer %s", raw, peer)
		}
	}
}

func TestIPWhitelistMiddlewareTrustedProxyNone(t *testing.T) {
	request := func() int {
		handler := IPWhitelistMiddleware("203.0.113.5")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		req.RemoteAddr = "10.0.0.7:1234"
		req.Header.Set("X-Forwarded-For", "203.0.113.5")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}
	t.Setenv("TRUSTED_PROXY_IPS", "")
	require.Equal(t, http.StatusOK, request())
	t.Setenv("TRUSTED_PROXY_IPS", "none")
	require.Equal(t, http.StatusForbidden, request())
}
