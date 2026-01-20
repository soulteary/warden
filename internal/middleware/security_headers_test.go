package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

// TestSecurityHeadersMiddleware tests security response headers setting
func TestSecurityHeadersMiddleware(t *testing.T) {
	middleware := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	}))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// Verify all security response headers are set
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"), "Should set X-Content-Type-Options")
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"), "Should set X-Frame-Options")
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"), "Should set X-XSS-Protection")
	assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"), "Should set Referrer-Policy")
	assert.Contains(t, w.Header().Get("Content-Security-Policy"), "default-src 'self'", "Should set Content-Security-Policy")
}

// TestSecurityHeadersMiddleware_AllHeaders tests content of all security headers
func TestSecurityHeadersMiddleware_AllHeaders(t *testing.T) {
	middleware := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	headers := w.Header()

	// Verify X-Content-Type-Options
	assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"), "X-Content-Type-Options should be nosniff")

	// Verify X-Frame-Options
	assert.Equal(t, "DENY", headers.Get("X-Frame-Options"), "X-Frame-Options should be DENY")

	// Verify X-XSS-Protection
	assert.Equal(t, "1; mode=block", headers.Get("X-XSS-Protection"), "X-XSS-Protection should be 1; mode=block")

	// Verify Referrer-Policy
	assert.Equal(t, "strict-origin-when-cross-origin", headers.Get("Referrer-Policy"), "Referrer-Policy should be strict-origin-when-cross-origin")

	// Verify Content-Security-Policy
	csp := headers.Get("Content-Security-Policy")
	assert.NotEmpty(t, csp, "Content-Security-Policy should not be empty")
	assert.Contains(t, csp, "default-src 'self'", "CSP should contain default-src 'self'")
	assert.Contains(t, csp, "script-src 'self'", "CSP should contain script-src 'self'")
	assert.Contains(t, csp, "style-src 'self' 'unsafe-inline'", "CSP should contain style-src 'self' 'unsafe-inline'")
}

// TestSecurityHeadersMiddleware_DifferentStatusCodes tests security headers under different status codes
func TestSecurityHeadersMiddleware_DifferentStatusCodes(t *testing.T) {
	statusCodes := []int{
		http.StatusOK,
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusNotFound,
		http.StatusInternalServerError,
	}

	for _, statusCode := range statusCodes {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			middleware := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(statusCode)
			}))

			req := httptest.NewRequest("GET", "/", http.NoBody)
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			// Verify all status codes should set security headers
			assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"), "All status codes should set security headers")
			assert.Equal(t, statusCode, w.Code, "Status code should be set correctly")
		})
	}
}

// TestSecurityHeadersMiddleware_Concurrent tests concurrency safety
func TestSecurityHeadersMiddleware_Concurrent(t *testing.T) {
	middleware := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	}))

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/", http.NoBody)
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestSecurityHeadersMiddleware_DoesNotOverride tests that it doesn't override existing response headers
func TestSecurityHeadersMiddleware_DoesNotOverride(t *testing.T) {
	middleware := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set some response headers first
		w.Header().Set("X-Custom-Header", "custom-value")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// Verify custom header still exists
	assert.Equal(t, "custom-value", w.Header().Get("X-Custom-Header"), "Custom response header should be preserved")
	// Verify security headers are also set
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"), "Security headers should also be set")
}

// TestSecurityHeadersMiddleware_CSPContent tests CSP content completeness
func TestSecurityHeadersMiddleware_CSPContent(t *testing.T) {
	middleware := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	csp := w.Header().Get("Content-Security-Policy")

	// Verify CSP contains all necessary directives
	expectedDirectives := []string{
		"default-src 'self'",
		"script-src 'self'",
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data:",
		"font-src 'self' data:",
	}

	for _, directive := range expectedDirectives {
		assert.Contains(t, csp, directive, "CSP should contain directive: %s", directive)
	}
}
