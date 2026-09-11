// Package middleware provides HTTP middleware functionality.
// Includes rate limiting, compression, request body limiting, metrics collection and other middleware.
package middleware

import (
	// Standard library
	"net/http"
	"strconv"
	"time"

	// Internal packages
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/prommetrics"
)

// MetricsMiddleware creates Prometheus metrics collection middleware
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap ResponseWriter to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		// Record metrics.
		//
		// BOTH label values are normalized against an allowlist instead of using the raw
		// request. This middleware runs BEFORE authentication, so an unauthenticated caller
		// sending arbitrary paths — or arbitrary method tokens, which net/http accepts and
		// hands to the handler verbatim — would otherwise create a new, permanent
		// Prometheus time series per request.
		duration := time.Since(start).Seconds()
		endpoint := define.NormalizeEndpointLabel(r.URL.Path)
		method := define.NormalizeMethodLabel(r.Method)

		status := strconv.Itoa(rw.statusCode)

		prommetrics.HTTPRequestTotal.WithLabelValues(method, endpoint, status).Inc()
		prommetrics.HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(duration)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
