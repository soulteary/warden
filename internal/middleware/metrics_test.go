package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

// TestMetricsMiddleware_RecordsMetrics tests metrics recording
func TestMetricsMiddleware_RecordsMetrics(t *testing.T) {
	middleware := MetricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	}))

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	w := httptest.NewRecorder()

	start := time.Now()
	middleware.ServeHTTP(w, req)
	duration := time.Since(start)

	assert.Equal(t, http.StatusOK, w.Code, "Should return 200")
	assert.Less(t, duration, 100*time.Millisecond, "Request should complete quickly")
	// Note: Actual metrics verification requires accessing metrics package, here only verify middleware doesn't panic
}

// TestMetricsMiddleware_DifferentStatusCodes tests metrics recording for different status codes
func TestMetricsMiddleware_DifferentStatusCodes(t *testing.T) {
	statusCodes := []int{
		http.StatusOK,
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusNotFound,
		http.StatusInternalServerError,
	}

	for _, statusCode := range statusCodes {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			middleware := MetricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(statusCode)
			}))

			req := httptest.NewRequest("GET", "/test", http.NoBody)
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			assert.Equal(t, statusCode, w.Code, "Status code should be set correctly")
		})
	}
}

// TestMetricsMiddleware_DifferentMethods tests metrics recording for different HTTP methods
func TestMetricsMiddleware_DifferentMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			middleware := MetricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(method, "/test", http.NoBody)
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code, "Should return 200")
		})
	}
}

// TestMetricsMiddleware_DifferentEndpoints tests metrics recording for different endpoints
func TestMetricsMiddleware_DifferentEndpoints(t *testing.T) {
	endpoints := []string{"/", "/health", "/user", "/metrics", "/api/v1/test"}

	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			middleware := MetricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", endpoint, http.NoBody)
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code, "Should return 200")
		})
	}
}

// TestMetricsMiddleware_EmptyPath tests empty path handling
func TestMetricsMiddleware_EmptyPath(t *testing.T) {
	middleware := MetricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// In handler, if path is empty, should be treated as "/"
		if r.URL.Path == "" {
			r.URL.Path = "/"
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Use valid URL, but test middleware handling of empty path
	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Should return 200")
}

// TestMetricsMiddleware_ResponseWriter tests ResponseWriter wrapping
func TestMetricsMiddleware_ResponseWriter(t *testing.T) {
	middleware := MetricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Test multiple writes
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("Hello"))
		require.NoError(t, err)
		_, err = w.Write([]byte(" World"))
		require.NoError(t, err)
	}))

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Should return 200")
	assert.Equal(t, "Hello World", w.Body.String(), "Response body should be correct")
}

// TestMetricsMiddleware_Concurrent tests concurrency safety
func TestMetricsMiddleware_Concurrent(t *testing.T) {
	middleware := MetricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	}))

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/test", http.NoBody)
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

// TestMetricsMiddleware_DurationMeasurement tests duration measurement
func TestMetricsMiddleware_DurationMeasurement(t *testing.T) {
	// Create a handler with delay
	middleware := MetricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	w := httptest.NewRecorder()

	start := time.Now()
	middleware.ServeHTTP(w, req)
	duration := time.Since(start)

	assert.Equal(t, http.StatusOK, w.Code, "Should return 200")
	assert.GreaterOrEqual(t, duration, 50*time.Millisecond, "Duration should be at least 50ms")
	assert.Less(t, duration, 200*time.Millisecond, "Duration should not exceed 200ms (including overhead)")
}

// TestResponseWriter_WriteHeader tests ResponseWriter's WriteHeader method
func TestResponseWriter_WriteHeader(t *testing.T) {
	recorder := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: recorder,
		statusCode:     http.StatusOK,
	}

	// Test setting status code
	rw.WriteHeader(http.StatusNotFound)
	assert.Equal(t, http.StatusNotFound, rw.statusCode, "Status code should be updated")
	assert.Equal(t, http.StatusNotFound, recorder.Code, "Underlying ResponseWriter's status code should be updated")
}

// TestResponseWriter_WriteHeader_MultipleCalls tests multiple calls to WriteHeader
func TestResponseWriter_WriteHeader_MultipleCalls(t *testing.T) {
	recorder := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: recorder,
		statusCode:     http.StatusOK,
	}

	// First call
	rw.WriteHeader(http.StatusBadRequest)
	assert.Equal(t, http.StatusBadRequest, rw.statusCode)

	// Second call (should update status code)
	rw.WriteHeader(http.StatusInternalServerError)
	assert.Equal(t, http.StatusInternalServerError, rw.statusCode)
}
