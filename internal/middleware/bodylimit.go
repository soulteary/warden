// Package middleware provides HTTP middleware functionality.
// Includes rate limiting, compression, request body limiting, metrics collection and other middleware.
package middleware

import (
	// Standard library
	"net/http"

	// Third-party libraries
	"github.com/rs/zerolog/hlog"

	// Internal packages
	"github.com/soulteary/warden/internal/define"
)

// BodyLimitMiddleware creates request body size limiting middleware
// Limits request body size to prevent malicious requests
// Note: http.MaxBytesReader will automatically check size when reading, returns error if limit is exceeded
func BodyLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For GET/HEAD requests, usually no request body, pass through directly
		if r.Method == "GET" || r.Method == "HEAD" {
			next.ServeHTTP(w, r)
			return
		}

		// Check Content-Length header
		if r.ContentLength > define.MAX_REQUEST_BODY_SIZE {
			hlog.FromRequest(r).Warn().
				Int64("content_length", r.ContentLength).
				Int("max_size", define.MAX_REQUEST_BODY_SIZE).
				Msg("Request body size exceeds limit")
			http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
			return
		}

		// Limit request body size (MaxBytesReader will check when reading)
		// If limit is exceeded, will return error on subsequent reads
		r.Body = http.MaxBytesReader(w, r.Body, define.MAX_REQUEST_BODY_SIZE)

		next.ServeHTTP(w, r)
	})
}
