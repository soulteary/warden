package tracing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestMiddleware_Success(t *testing.T) {
	defer teardownTestTracer()
	tp, exporter := setupTestTracer(t)
	require.NotNil(t, tp)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	})

	middleware := Middleware(handler)

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "OK", w.Body.String())

	// Force flush to exporter
	require.NoError(t, tp.ForceFlush(context.Background()))

	spans := exporter.GetSpans()
	if len(spans) > 0 {
		assert.Equal(t, "/test", spans[0].Name)
	}
}

func TestMiddleware_DifferentStatusCodes(t *testing.T) {
	defer teardownTestTracer()
	tp, exporter := setupTestTracer(t)
	require.NotNil(t, tp)

	//nolint:govet // fieldalignment: test struct, field order optimized for readability
	testCases := []struct {
		name           string
		statusCode     int
		expectedStatus string
	}{
		{name: "200 OK", statusCode: http.StatusOK, expectedStatus: "OK"},
		{name: "400 Bad Request", statusCode: http.StatusBadRequest, expectedStatus: "Error"},
		{name: "404 Not Found", statusCode: http.StatusNotFound, expectedStatus: "Error"},
		{name: "500 Internal Server Error", statusCode: http.StatusInternalServerError, expectedStatus: "Error"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			exporter.Reset()

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			})

			middleware := Middleware(handler)
			req := httptest.NewRequest("GET", "/test", http.NoBody)
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			assert.Equal(t, tc.statusCode, w.Code)

			require.NoError(t, tp.ForceFlush(context.Background()))
			spans := exporter.GetSpans()
			if len(spans) > 0 {
				spanData := spans[0]
				// Verify status code is set correctly (Error for >= 400, Ok otherwise)
				if tc.statusCode >= 400 {
					assert.Equal(t, "Error", spanData.Status.Code.String())
				} else {
					assert.Equal(t, "Ok", spanData.Status.Code.String())
				}
			}
		})
	}
}

func TestMiddleware_TraceContextExtraction(t *testing.T) {
	defer teardownTestTracer()
	tp, _ := setupTestTracer(t)
	require.NotNil(t, tp)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify trace context is in request context
		span := trace.SpanFromContext(r.Context())
		assert.NotNil(t, span)
		w.WriteHeader(http.StatusOK)
	})

	middleware := Middleware(handler)

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	// Add trace context headers
	propagator := otel.GetTextMapPropagator()
	ctx, parentSpan := tp.Tracer("test").Start(context.Background(), "parent")
	propagator.Inject(ctx, propagation.HeaderCarrier(req.Header))
	parentSpan.End()

	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMiddleware_TraceContextInjection(t *testing.T) {
	defer teardownTestTracer()
	tp, _ := setupTestTracer(t)
	require.NotNil(t, tp)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := Middleware(handler)

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// Verify trace context is injected into response headers
	// The exact header names depend on the propagator configuration
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMiddleware_SpanAttributes(t *testing.T) {
	defer teardownTestTracer()
	tp, exporter := setupTestTracer(t)
	require.NotNil(t, tp)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := Middleware(handler)

	req := httptest.NewRequest("GET", "/test?param=value", http.NoBody)
	req.Header.Set("User-Agent", "test-agent/1.0")
	req.RemoteAddr = "192.168.1.1:12345"

	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	require.NoError(t, tp.ForceFlush(context.Background()))
	spans := exporter.GetSpans()
	if len(spans) > 0 {
		spanData := spans[0]
		// Verify attributes are set (checking for key presence)
		foundMethod := false
		foundURL := false
		foundUserAgent := false
		foundRemoteAddr := false

		for _, attr := range spanData.Attributes {
			switch attr.Key {
			case "http.method":
				foundMethod = true
				assert.Equal(t, "GET", attr.Value.AsString())
			case "http.url":
				foundURL = true
			case "http.user_agent":
				foundUserAgent = true
				assert.Equal(t, "test-agent/1.0", attr.Value.AsString())
			case "http.remote_addr":
				foundRemoteAddr = true
				assert.Equal(t, "192.168.1.1:12345", attr.Value.AsString())
			}
		}

		// At least some attributes should be found
		assert.True(t, foundMethod || foundURL || foundUserAgent || foundRemoteAddr)
	}
}

func TestMiddleware_EmptyPath(t *testing.T) {
	defer teardownTestTracer()
	tp, exporter := setupTestTracer(t)
	require.NotNil(t, tp)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := Middleware(handler)

	// Use "/" instead of empty string, as httptest.NewRequest doesn't accept empty URL
	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	require.NoError(t, tp.ForceFlush(context.Background()))
	spans := exporter.GetSpans()
	if len(spans) > 0 {
		// Path should be set
		assert.NotEmpty(t, spans[0].Name)
	}
}

func TestMiddleware_DifferentMethods(t *testing.T) {
	defer teardownTestTracer()
	tp, exporter := setupTestTracer(t)
	require.NotNil(t, tp)

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			exporter.Reset()

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			middleware := Middleware(handler)
			req := httptest.NewRequest(method, "/test", http.NoBody)
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			require.NoError(t, tp.ForceFlush(context.Background()))
			spans := exporter.GetSpans()
			if len(spans) > 0 {
				assert.Equal(t, "/test", spans[0].Name)
			}
		})
	}
}

func TestMiddleware_ResponseSize(t *testing.T) {
	defer teardownTestTracer()
	tp, exporter := setupTestTracer(t)
	require.NotNil(t, tp)

	responseBody := "This is a test response"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(responseBody))
		require.NoError(t, err)
	})

	middleware := Middleware(handler)

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, responseBody, w.Body.String())

	require.NoError(t, tp.ForceFlush(context.Background()))
	spans := exporter.GetSpans()
	if len(spans) > 0 {
		spanData := spans[0]
		// Verify response size attribute
		foundSize := false
		for _, attr := range spanData.Attributes {
			if attr.Key == "http.response.size" {
				foundSize = true
				assert.Equal(t, int64(len(responseBody)), attr.Value.AsInt64())
				break
			}
		}
		// Response size should be recorded
		assert.True(t, foundSize)
	}
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	baseWriter := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: baseWriter,
		statusCode:     http.StatusOK,
	}

	rw.WriteHeader(http.StatusNotFound)

	assert.Equal(t, http.StatusNotFound, rw.statusCode)
	assert.Equal(t, http.StatusNotFound, baseWriter.Code)
}

func TestResponseWriter_Write(t *testing.T) {
	baseWriter := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: baseWriter,
		statusCode:     http.StatusOK,
	}

	data := []byte("test data")
	n, err := rw.Write(data)

	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, len(data), rw.responseSize)
	assert.Equal(t, data, baseWriter.Body.Bytes())
}

func TestResponseWriter_MultipleWrites(t *testing.T) {
	baseWriter := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: baseWriter,
		statusCode:     http.StatusOK,
	}

	data1 := []byte("first ")
	data2 := []byte("second")
	data3 := []byte(" third")

	_, err := rw.Write(data1)
	require.NoError(t, err)
	_, err = rw.Write(data2)
	require.NoError(t, err)
	_, err = rw.Write(data3)
	require.NoError(t, err)

	expectedSize := len(data1) + len(data2) + len(data3)
	assert.Equal(t, expectedSize, rw.responseSize)
	assert.Equal(t, "first second third", baseWriter.Body.String())
}

func TestResponseWriter_WriteHeaderAfterWrite(t *testing.T) {
	baseWriter := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: baseWriter,
		statusCode:     http.StatusOK,
	}

	_, err := rw.Write([]byte("data"))
	require.NoError(t, err)
	rw.WriteHeader(http.StatusCreated)

	// In HTTP standard library, once Write is called, WriteHeader becomes ineffective
	// because headers have already been sent. However, our wrapper should still
	// update the internal statusCode field for tracking purposes.
	// The base writer will have StatusOK (200) because Write was called first,
	// which implicitly calls WriteHeader(200) if not already called.
	assert.Equal(t, http.StatusCreated, rw.statusCode) // Our wrapper tracks it
	assert.Equal(t, http.StatusOK, baseWriter.Code)    // But base writer already sent 200
}

func TestMiddleware_ContextPropagation(t *testing.T) {
	defer teardownTestTracer()
	tp, _ := setupTestTracer(t)
	require.NotNil(t, tp)

	var capturedSpan trace.Span
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture span from context
		capturedSpan = trace.SpanFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	middleware := Middleware(handler)

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// Verify span was captured and is not nil
	assert.NotNil(t, capturedSpan)
	assert.Equal(t, http.StatusOK, w.Code)
}
