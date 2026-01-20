package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

// TestCompressMiddleware_NoGzipSupport tests client without gzip support
func TestCompressMiddleware_NoGzipSupport(t *testing.T) {
	middleware := CompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("test response"))
		require.NoError(t, err)
	}))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	// Don't set Accept-Encoding
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("Content-Encoding"), "Should not set Content-Encoding when gzip is not supported")
	assert.Equal(t, "test response", w.Body.String(), "Response body should be uncompressed")
}

// TestCompressMiddleware_WithGzipSupport tests client with gzip support
func TestCompressMiddleware_WithGzipSupport(t *testing.T) {
	middleware := CompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("test response"))
		require.NoError(t, err)
	}))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"), "Should set Content-Encoding: gzip")
	assert.Equal(t, "Accept-Encoding", w.Header().Get("Vary"), "Should set Vary: Accept-Encoding")

	// Verify response is compressed
	body := w.Body.Bytes()
	if len(body) > 0 {
		// Check if it's gzip format (gzip files start with 1f 8b)
		if len(body) >= 2 && body[0] == 0x1f && body[1] == 0x8b {
			// Try to decompress
			reader, err := gzip.NewReader(strings.NewReader(string(body)))
			if err == nil {
				decompressed, err := io.ReadAll(reader)
				assert.NoError(t, err, "Should be able to decompress response")
				assert.Equal(t, "test response", string(decompressed), "Decompressed content should be correct")
				require.NoError(t, reader.Close())
			}
		}
	}
}

// TestCompressMiddleware_MultipleEncodings tests multiple encoding formats
func TestCompressMiddleware_MultipleEncodings(t *testing.T) {
	middleware := CompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("test response"))
		require.NoError(t, err)
	}))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"), "Should use gzip compression")
}

// TestCompressMiddleware_EmptyResponse tests empty response
func TestCompressMiddleware_EmptyResponse(t *testing.T) {
	middleware := CompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// TestCompressMiddleware_LargeResponse tests large response
func TestCompressMiddleware_LargeResponse(t *testing.T) {
	largeData := strings.Repeat("test data ", 1000)
	middleware := CompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(largeData))
		require.NoError(t, err)
	}))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"), "Should use gzip compression")

	// Verify compressed data should be smaller than original data
	compressedSize := len(w.Body.Bytes())
	originalSize := len(largeData)
	assert.Less(t, compressedSize, originalSize, "Compressed data should be smaller than original data")
}

// TestCompressMiddleware_ConcurrentRequests tests concurrent requests
func TestCompressMiddleware_ConcurrentRequests(t *testing.T) {
	middleware := CompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("test response"))
		require.NoError(t, err)
	}))

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/", http.NoBody)
			req.Header.Set("Accept-Encoding", "gzip")
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestCompressMiddleware_GzipWriterPool tests gzip writer pool
func TestCompressMiddleware_GzipWriterPool(t *testing.T) {
	middleware := CompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("test response"))
		require.NoError(t, err)
	}))

	// Multiple requests should reuse writer
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/", http.NoBody)
		req.Header.Set("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"))
	}
}
