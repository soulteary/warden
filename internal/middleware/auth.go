// Package middleware provides HTTP middleware functionality.
// Includes rate limiting, compression, request body limiting, metrics collection and other middleware.
package middleware

import (
	// Standard library
	"net/http"
	"strings"

	// Third-party libraries
	"github.com/rs/zerolog/hlog"

	// Internal packages
	"github.com/soulteary/warden/internal/i18n"
)

// AuthMiddleware creates API Key authentication middleware
//
// This middleware verifies whether requests are authorized by checking the X-API-Key in request headers.
// If API Key is empty, all requests will be rejected (API Key should be set in production environment).
// If API Key is not empty, only requests providing the correct API Key can pass.
//
// Parameters:
//   - apiKey: API Key value, if empty then authentication is disabled (not recommended for production use)
//
// Returns:
//   - func(http.Handler) http.Handler: HTTP middleware function
//
// Usage example:
//
//	authMiddleware := AuthMiddleware("your-api-key-here")
//	handler := authMiddleware(protectedHandler)
func AuthMiddleware(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If API Key is empty, should reject all requests in production environment
			// But may allow in development environment (controlled via environment variables)
			if apiKey == "" {
				// Check if it's development environment (determined via environment variables)
				// In production environment, API Key should be set
				hlog.FromRequest(r).Warn().
					Msg(i18n.T(r, "error.api_key_not_configured"))
				http.Error(w, i18n.T(r, "http.unauthorized"), http.StatusUnauthorized)
				return
			}

			// Get API Key from request headers
			// Supports both X-API-Key and Authorization: Bearer <key> methods
			providedKey := r.Header.Get("X-API-Key")
			if providedKey == "" {
				// Try to get from Authorization header
				authHeader := r.Header.Get("Authorization")
				if strings.HasPrefix(authHeader, "Bearer ") {
					providedKey = strings.TrimPrefix(authHeader, "Bearer ")
				}
			}

			// Verify API Key
			if providedKey == "" || providedKey != apiKey {
				hlog.FromRequest(r).Warn().
					Str("ip", getClientIP(r)).
					Str("path", r.URL.Path).
					Str("method", r.Method).
					Msg(i18n.T(r, "error.auth_failed"))
				http.Error(w, i18n.T(r, "http.unauthorized"), http.StatusUnauthorized)
				return
			}

			// Authentication successful, continue processing request
			next.ServeHTTP(w, r)
		})
	}
}

// OptionalAuthMiddleware creates optional API Key authentication middleware
//
// Unlike AuthMiddleware, if API Key is empty, this middleware will not reject requests.
// This is suitable for scenarios where certain endpoints require optional authentication.
//
// Parameters:
//   - apiKey: API Key value, if empty then no authentication is performed
//
// Returns:
//   - func(http.Handler) http.Handler: HTTP middleware function
func OptionalAuthMiddleware(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If API Key is empty, do not perform authentication
			if apiKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Execute authentication logic (same as AuthMiddleware)
			providedKey := r.Header.Get("X-API-Key")
			if providedKey == "" {
				authHeader := r.Header.Get("Authorization")
				if strings.HasPrefix(authHeader, "Bearer ") {
					providedKey = strings.TrimPrefix(authHeader, "Bearer ")
				}
			}

			if providedKey == "" || providedKey != apiKey {
				hlog.FromRequest(r).Warn().
					Str("ip", getClientIP(r)).
					Str("path", r.URL.Path).
					Str("method", r.Method).
					Msg(i18n.T(r, "error.auth_failed"))
				http.Error(w, i18n.T(r, "http.unauthorized"), http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
