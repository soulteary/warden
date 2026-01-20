package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/define"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

// TestBodyLimitMiddleware_GETRequest tests GET request (should pass)
func TestBodyLimitMiddleware_GETRequest(t *testing.T) {
	middleware := BodyLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	}))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "GET request should pass")
}

// TestBodyLimitMiddleware_HEADRequest tests HEAD request (should pass)
func TestBodyLimitMiddleware_HEADRequest(t *testing.T) {
	middleware := BodyLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("HEAD", "/", http.NoBody)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "HEAD request should pass")
}

// TestBodyLimitMiddleware_ValidSize tests valid size request body
func TestBodyLimitMiddleware_ValidSize(t *testing.T) {
	middleware := BodyLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(body)
		require.NoError(t, err)
	}))

	// Create a request body smaller than limit
	bodySize := define.MAX_REQUEST_BODY_SIZE / 2
	body := bytes.NewReader(make([]byte, bodySize))

	req := httptest.NewRequest("POST", "/", body)
	req.ContentLength = int64(bodySize)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Valid size request body should pass")
}

// TestBodyLimitMiddleware_ExceedsLimit tests request body exceeding limit
func TestBodyLimitMiddleware_ExceedsLimit(t *testing.T) {
	middleware := BodyLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create a request body exceeding limit
	bodySize := define.MAX_REQUEST_BODY_SIZE + 1
	req := httptest.NewRequest("POST", "/", http.NoBody)
	req.ContentLength = int64(bodySize)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code, "Request body exceeding limit should return 413")
}

// TestBodyLimitMiddleware_ExactLimit tests request body exactly at limit
func TestBodyLimitMiddleware_ExactLimit(t *testing.T) {
	middleware := BodyLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	}))

	// Create a request body exactly at limit
	bodySize := define.MAX_REQUEST_BODY_SIZE
	body := bytes.NewReader(make([]byte, bodySize))

	req := httptest.NewRequest("POST", "/", body)
	req.ContentLength = int64(bodySize)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Request body exactly at limit should pass")
}

// TestBodyLimitMiddleware_NoContentLength tests request without Content-Length header
func TestBodyLimitMiddleware_NoContentLength(t *testing.T) {
	middleware := BodyLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	}))

	// Create a request without Content-Length
	body := bytes.NewReader([]byte("test body"))
	req := httptest.NewRequest("POST", "/", body)
	// Don't set ContentLength
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// Should pass because MaxBytesReader will check when reading
	assert.Equal(t, http.StatusOK, w.Code, "Request without Content-Length should pass (will check when reading)")
}

// TestBodyLimitMiddleware_DifferentMethods tests different HTTP methods
func TestBodyLimitMiddleware_DifferentMethods(t *testing.T) {
	middleware := BodyLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	}))

	methods := []string{"POST", "PUT", "PATCH", "DELETE"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			bodySize := define.MAX_REQUEST_BODY_SIZE + 1
			req := httptest.NewRequest(method, "/", http.NoBody)
			req.ContentLength = int64(bodySize)
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code, "%s request exceeding limit should return 413", method)
		})
	}
}
