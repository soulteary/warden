// Package middleware provides HTTP middleware functionality.
// Includes rate limiting, compression, request body limiting, metrics collection and other middleware.
package middleware

import (
	// Standard library
	"net/http"
)

// SecurityHeadersMiddleware creates security response headers middleware
//
// This middleware adds security-related HTTP response headers to improve application security.
// Includes:
// - X-Content-Type-Options: prevents MIME type sniffing
// - X-Frame-Options: prevents clickjacking
// - X-XSS-Protection: enables browser XSS filter
// - Referrer-Policy: controls referrer information
// - Content-Security-Policy: content security policy (optional)
//
// Returns:
//   - func(http.Handler) http.Handler: HTTP middleware function
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// Enable browser XSS filter
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Control referrer information (don't leak source)
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy (CSP) - adjust according to actual needs
		// Uses a more relaxed policy here, can be adjusted if stricter security is needed
		csp := "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self' data:;"
		w.Header().Set("Content-Security-Policy", csp)

		// Continue processing request
		next.ServeHTTP(w, r)
	})
}
