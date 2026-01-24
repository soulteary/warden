// Package tracing provides OpenTelemetry tracing functionality for HTTP requests.
// It includes middleware for automatic span creation and context propagation.
package tracing

import (
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// Middleware creates an HTTP middleware for OpenTelemetry tracing
func Middleware(next http.Handler) http.Handler {
	propagator := otel.GetTextMapPropagator()
	tracer := GetTracer()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract trace context from request headers
		ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		// Determine span name from route path
		spanName := r.URL.Path
		if spanName == "" {
			spanName = r.Method + " " + r.URL.Path
		}

		// Start span
		ctx, span := tracer.Start(
			ctx,
			spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPMethod(r.Method),
				semconv.HTTPURL(r.URL.String()),
				attribute.String("http.user_agent", r.UserAgent()),
				attribute.String("http.remote_addr", r.RemoteAddr),
			),
		)
		defer span.End()

		// Create response writer wrapper to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Process request with context
		next.ServeHTTP(rw, r.WithContext(ctx))

		// Set span status and attributes based on response
		span.SetAttributes(
			semconv.HTTPStatusCode(rw.statusCode),
			attribute.Int("http.response.size", rw.responseSize),
		)

		if rw.statusCode >= 400 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", rw.statusCode))
		} else {
			span.SetStatus(codes.Ok, "")
		}

		// Inject trace context into response headers
		propagator.Inject(ctx, propagation.HeaderCarrier(rw.Header()))
	})
}

// responseWriter wraps http.ResponseWriter to capture status code and response size
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	responseSize int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.responseSize += len(b)
	return rw.ResponseWriter.Write(b)
}
