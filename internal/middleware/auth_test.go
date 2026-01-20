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
	// Initialize zerolog to avoid panic
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

// TestAuthMiddleware_ValidAPIKey tests valid API Key
func TestAuthMiddleware_ValidAPIKey(t *testing.T) {
	apiKey := "test-api-key-123"
	middleware := AuthMiddleware(apiKey)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	}))

	tests := []struct {
		name   string
		header string
		value  string
		want   int
	}{
		{
			name:   "X-API-Key header",
			header: "X-API-Key",
			value:  apiKey,
			want:   http.StatusOK,
		},
		{
			name:   "Authorization Bearer header",
			header: "Authorization",
			value:  "Bearer " + apiKey,
			want:   http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", http.NoBody)
			req.Header.Set(tt.header, tt.value)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.want, w.Code, "Should return 200")
		})
	}
}

// TestAuthMiddleware_InvalidAPIKey tests invalid API Key
func TestAuthMiddleware_InvalidAPIKey(t *testing.T) {
	apiKey := "test-api-key-123"
	middleware := AuthMiddleware(apiKey)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name   string
		header string
		value  string
	}{
		{
			name:   "Wrong X-API-Key",
			header: "X-API-Key",
			value:  "wrong-key",
		},
		{
			name:   "Wrong Authorization Bearer",
			header: "Authorization",
			value:  "Bearer wrong-key",
		},
		{
			name:   "Empty X-API-Key",
			header: "X-API-Key",
			value:  "",
		},
		{
			name:   "Missing header",
			header: "",
			value:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", http.NoBody)
			if tt.header != "" {
				req.Header.Set(tt.header, tt.value)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code, "Should return 401")
		})
	}
}

// TestAuthMiddleware_EmptyAPIKey tests empty API Key (should reject all requests)
func TestAuthMiddleware_EmptyAPIKey(t *testing.T) {
	middleware := AuthMiddleware("")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "Empty API Key should reject all requests")
}

// TestOptionalAuthMiddleware_ValidAPIKey tests optional authentication middleware (valid API Key)
func TestOptionalAuthMiddleware_ValidAPIKey(t *testing.T) {
	apiKey := "test-api-key-123"
	middleware := OptionalAuthMiddleware(apiKey)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	}))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	req.Header.Set("X-API-Key", apiKey)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Valid API Key should pass")
}

// TestOptionalAuthMiddleware_InvalidAPIKey tests optional authentication middleware (invalid API Key)
func TestOptionalAuthMiddleware_InvalidAPIKey(t *testing.T) {
	apiKey := "test-api-key-123"
	middleware := OptionalAuthMiddleware(apiKey)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	req.Header.Set("X-API-Key", "wrong-key")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "Invalid API Key should return 401")
}

// TestOptionalAuthMiddleware_EmptyAPIKey tests optional authentication middleware (empty API Key, should allow pass)
func TestOptionalAuthMiddleware_EmptyAPIKey(t *testing.T) {
	middleware := OptionalAuthMiddleware("")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	}))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Should allow pass when API Key is empty (optional authentication)")
}

// TestAuthMiddleware_AuthorizationHeaderPrecedence tests Authorization header precedence
func TestAuthMiddleware_AuthorizationHeaderPrecedence(t *testing.T) {
	apiKey := "test-api-key-123"
	middleware := AuthMiddleware(apiKey)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	}))

	// Provide both X-API-Key and Authorization, X-API-Key should take precedence
	req := httptest.NewRequest("GET", "/", http.NoBody)
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Authorization", "Bearer wrong-key")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "X-API-Key should take precedence over Authorization")
}

// TestAuthMiddleware_BearerTokenFormat tests Bearer token format
func TestAuthMiddleware_BearerTokenFormat(t *testing.T) {
	apiKey := "test-api-key-123"
	middleware := AuthMiddleware(apiKey)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name  string
		value string
		want  int
	}{
		{
			name:  "Correct Bearer format",
			value: "Bearer test-api-key-123",
			want:  http.StatusOK,
		},
		{
			name:  "Extra space after Bearer",
			value: "Bearer  test-api-key-123",
			want:  http.StatusUnauthorized, // Should fail because of extra space
		},
		{
			name:  "Lowercase bearer",
			value: "bearer test-api-key-123",
			want:  http.StatusUnauthorized, // Should fail because it's not "Bearer "
		},
		{
			name:  "No Bearer prefix",
			value: "test-api-key-123",
			want:  http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", http.NoBody)
			req.Header.Set("Authorization", tt.value)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.want, w.Code, "Status code should match")
		})
	}
}
