// Package middleware provides HTTP middleware functionality.
// Includes rate limiting, compression, request body limiting, metrics collection and other middleware.
package middleware

import (
	// Standard library
	"encoding/json"
	"net/http"
	"os"

	// Third-party libraries
	"github.com/rs/zerolog/hlog"

	// Internal packages
	"github.com/soulteary/warden/internal/i18n"
)

// ErrorResponse error response structure
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// ErrorHandlerMiddleware creates error handling middleware
//
// This middleware hides detailed error information in production environment, only returning generic error messages.
// Detailed error information is only recorded in logs, not returned to clients.
//
// Parameters:
//   - appMode: application mode ("production" or "prod" indicates production environment)
//
// Returns:
//   - func(http.Handler) http.Handler: HTTP middleware function
func ErrorHandlerMiddleware(appMode string) func(http.Handler) http.Handler {
	isProduction := appMode == "production" || appMode == "prod"

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Use custom ResponseWriter to capture errors
			rw := &errorResponseWriter{
				ResponseWriter: w,
				isProduction:   isProduction,
				request:        r,
			}
			next.ServeHTTP(rw, r)
		})
	}
}

// errorResponseWriter custom ResponseWriter for capturing and modifying error responses
type errorResponseWriter struct {
	http.ResponseWriter
	request      *http.Request
	statusCode   int
	isProduction bool
}

// WriteHeader captures status code
func (rw *errorResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write captures response body
func (rw *errorResponseWriter) Write(b []byte) (int, error) {
	// If it's an error response (4xx or 5xx), may need to hide detailed information in production environment
	if rw.statusCode >= 400 && rw.isProduction {
		// Try to parse JSON error response
		var errResp ErrorResponse
		if err := json.Unmarshal(b, &errResp); err == nil {
			// In production environment, only return generic error message
			genericResp := ErrorResponse{
				Error: getGenericErrorMessage(rw.request, rw.statusCode),
			}
			// Record detailed error to log
			hlog.FromRequest(rw.request).Error().
				Int("status_code", rw.statusCode).
				Str("original_error", errResp.Error).
				Str("original_message", errResp.Message).
				Str("original_code", errResp.Code).
				Msg(i18n.T(rw.request, "error.error_response_hidden"))

			// Re-encode generic error response
			if newBody, err := json.Marshal(genericResp); err == nil {
				b = newBody
			}
		} else {
			// If not JSON format, also record original response
			hlog.FromRequest(rw.request).Error().
				Int("status_code", rw.statusCode).
				Str("original_response", string(b)).
				Msg(i18n.T(rw.request, "error.error_response_hidden"))
			// Return generic error message
			genericResp := ErrorResponse{
				Error: getGenericErrorMessage(rw.request, rw.statusCode),
			}
			if newBody, err := json.Marshal(genericResp); err == nil {
				b = newBody
			}
		}
	}

	return rw.ResponseWriter.Write(b)
}

// getGenericErrorMessage returns generic error message based on status code (supports internationalization)
func getGenericErrorMessage(r *http.Request, statusCode int) string {
	var key string
	switch {
	case statusCode >= 500:
		key = "error.internal_server_error"
	case statusCode == 404:
		key = "error.not_found"
	case statusCode == 403:
		key = "error.forbidden"
	case statusCode == 401:
		key = "error.unauthorized"
	case statusCode == 400:
		key = "error.bad_request"
	case statusCode == 429:
		key = "error.too_many_requests"
	default:
		key = "error.request_failed"
	}

	if r != nil {
		return i18n.T(r, key)
	}
	// If no request context, use default language
	return i18n.TWithLang(i18n.LangEN, key)
}

// SafeError safely returns error response (decides whether to hide detailed information based on environment)
func SafeError(w http.ResponseWriter, r *http.Request, statusCode int, err error, detailMessage string) {
	appMode := os.Getenv("MODE")
	isProduction := appMode == "production" || appMode == "prod"

	// Record detailed error to log
	hlog.FromRequest(r).Error().
		Int("status_code", statusCode).
		Err(err).
		Str("detail", detailMessage).
		Msg(i18n.T(r, "error.request_error"))

	// Build error response
	var resp ErrorResponse
	if isProduction {
		// Production environment: only return generic error message
		resp = ErrorResponse{
			Error: getGenericErrorMessage(r, statusCode),
		}
	} else {
		// Development environment: return detailed error information
		resp = ErrorResponse{
			Error:   getGenericErrorMessage(r, statusCode),
			Message: detailMessage,
		}
		if err != nil {
			resp.Message = err.Error()
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		hlog.FromRequest(r).Error().
			Err(err).
			Msg(i18n.T(r, "error.encode_error_response_failed"))
	}
}
