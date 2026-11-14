package middleware

import (
	"context"
	"net"
	"net/http"
)

type clientIPContextKey struct{}

// WithClientIP returns a copy of r that carries the client address resolved
// through the trusted-proxy configuration.
func WithClientIP(r *http.Request, clientIP string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), clientIPContextKey{}, clientIP))
}

// ClientIP returns the client address the rate limiter resolved for r: the
// forwarded address when the TCP peer is a trusted proxy, the peer otherwise.
// A request that did not pass through the limiter yields the TCP peer address
// without its port, so forwarded headers are never trusted implicitly.
func ClientIP(r *http.Request) string {
	if clientIP, ok := r.Context().Value(clientIPContextKey{}).(string); ok && clientIP != "" {
		return clientIP
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
