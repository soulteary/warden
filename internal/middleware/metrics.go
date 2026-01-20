// Package middleware provides HTTP middleware functionality.
// Includes rate limiting, compression, request body limiting, metrics collection and other middleware.
package middleware

import (
	// Standard library
	"net/http"
	"strconv"
	"time"

	// Internal packages
	"github.com/soulteary/warden/internal/metrics"
)

// MetricsMiddleware creates Prometheus metrics collection middleware
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap ResponseWriter to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		// Record metrics
		duration := time.Since(start).Seconds()
		endpoint := r.URL.Path
		if endpoint == "" {
			endpoint = "/"
		}

		status := strconv.Itoa(rw.statusCode)

		metrics.HTTPRequestTotal.WithLabelValues(r.Method, endpoint, status).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(r.Method, endpoint).Observe(duration)
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
