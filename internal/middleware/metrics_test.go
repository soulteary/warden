package middleware

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	loggerkit "github.com/soulteary/logger-kit/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/logger"
	"github.com/soulteary/warden/internal/prommetrics"
)

func init() {
	logger.SetLevel(loggerkit.InfoLevel)
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

// TestMetricsMiddleware_UnknownPathCollapsesLabel proves end-to-end that an attacker-chosen
// path never reaches the Prometheus endpoint label. This middleware runs before
// authentication, so a raw-path label lets an unauthenticated caller create one permanent
// time series per request.
func TestMetricsMiddleware_UnknownPathCollapsesLabel(t *testing.T) {
	handler := MetricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	const junk = "warden-cardinality-probe-8f21c0"
	for _, suffix := range []string{"a", "b", "c"} {
		req := httptest.NewRequest(http.MethodGet, "/"+junk+"-"+suffix, http.NoBody)
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}

	scrape := httptest.NewRecorder()
	prommetrics.Handler().ServeHTTP(scrape, httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody))
	body := scrape.Body.String()

	assert.NotContains(t, body, junk, "请求路径不得出现在指标标签中")
	assert.Contains(t, body, `endpoint="`+define.LABEL_OTHER+`"`, "未注册路径应归入 other 标签")
}

// TestMetricsMiddleware_KnownPathKeepsLabel confirms normalization did not flatten the
// routes operators actually dashboard on.
func TestMetricsMiddleware_KnownPathKeepsLabel(t *testing.T) {
	handler := MetricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, define.PATH_V1_LOOKUP, http.NoBody)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	scrape := httptest.NewRecorder()
	prommetrics.Handler().ServeHTTP(scrape, httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody))
	assert.Contains(t, scrape.Body.String(), `endpoint="`+define.PATH_V1_LOOKUP+`"`)
}

// TestMetricsMiddleware_UnknownMethodCollapsesLabel drives a real TCP connection so the
// request carries a method token that httptest.NewRequest would not let us forge. net/http
// passes any valid token straight to the handler, so without normalization each bogus
// method mints a permanent time series.
func TestMetricsMiddleware_UnknownMethodCollapsesLabel(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/", MetricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	const junkMethod = "WARDENPROBE"
	addr := strings.TrimPrefix(srv.URL, "http://")
	for i := 0; i < 3; i++ {
		conn, err := net.Dial("tcp", addr)
		require.NoError(t, err)
		_, err = fmt.Fprintf(conn, "%s%d /probe HTTP/1.1\r\nHost: warden.test\r\nConnection: close\r\n\r\n", junkMethod, i)
		require.NoError(t, err)
		status, err := bufio.NewReader(conn).ReadString('\n')
		require.NoError(t, err)
		require.Contains(t, status, "404", "任意方法仍应被服务端处理，从而进入指标中间件")
		require.NoError(t, conn.Close())
	}

	scrape := httptest.NewRecorder()
	prommetrics.Handler().ServeHTTP(scrape, httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody))
	body := scrape.Body.String()

	assert.NotContains(t, body, junkMethod, "请求方法不得出现在指标标签中")
	assert.Contains(t, body, `method="`+define.LABEL_OTHER+`"`, "未知方法应归入 other 标签")
}
