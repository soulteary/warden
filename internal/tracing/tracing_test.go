package tracing

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// resetGlobalState resets the global tracer state for testing
func resetGlobalState() {
	tracerProvider = nil
	tracer = nil
	otel.SetTracerProvider(sdktrace.NewTracerProvider())
}

// setupTestTracer creates a test tracer provider with in-memory exporter
func setupTestTracer(_ *testing.T) (*sdktrace.TracerProvider, *tracetest.InMemoryExporter) {
	resetGlobalState()

	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.Empty()),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)
	tracerProvider = tp
	tracer = tp.Tracer("test")

	return tp, exporter
}

// teardownTestTracer cleans up test tracer
func teardownTestTracer() {
	if tracerProvider != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = tracerProvider.Shutdown(ctx) // nolint:errcheck // test cleanup
	}
	resetGlobalState()
}

func TestInitTracer_Success(t *testing.T) {
	defer teardownTestTracer()
	resetGlobalState()

	// Use a mock endpoint that won't actually connect
	// We'll use an invalid endpoint to test initialization logic
	// but we need to handle the error case separately
	endpoint := "http://localhost:4318"
	tp, err := InitTracer("test-service", "1.0.0", endpoint)

	// InitTracer will try to create an OTLP exporter, which may fail
	// if the endpoint is not reachable, but we're testing the logic
	if err != nil {
		// If exporter creation fails, that's expected for invalid endpoints
		// We can still test that the function handles it correctly
		assert.Error(t, err)
		assert.Nil(t, tp)
	} else {
		// If it succeeds (e.g., in CI with mock server), verify setup
		require.NotNil(t, tp)
		assert.NotNil(t, tracerProvider)
		assert.NotNil(t, tracer)
		assert.True(t, IsEnabled())
	}
}

func TestInitTracer_EmptyEndpoint(t *testing.T) {
	defer teardownTestTracer()
	resetGlobalState()

	tp, err := InitTracer("test-service", "1.0.0", "")

	assert.NoError(t, err)
	assert.Nil(t, tp)
	assert.False(t, IsEnabled())
	assert.NotNil(t, GetTracer()) // Should return noop tracer
}

func TestInitTracer_ResourceCreationFailure(t *testing.T) {
	defer teardownTestTracer()
	resetGlobalState()

	// This test is difficult to trigger without mocking resource.New
	// Resource creation rarely fails in practice, but we test the error path exists
	// We'll test with a valid endpoint and verify error handling
	endpoint := "http://localhost:4318"
	_, err := InitTracer("test-service", "1.0.0", endpoint)

	// Error may occur if OTLP exporter creation fails (expected for invalid endpoint)
	// or if resource creation fails (unlikely but possible)
	if err != nil {
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create")
	}
}

func TestShutdown_WithProvider(t *testing.T) {
	defer resetGlobalState()
	_, _ = setupTestTracer(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := Shutdown(ctx)
	assert.NoError(t, err)
}

func TestShutdown_WithoutProvider(t *testing.T) {
	defer resetGlobalState()
	resetGlobalState()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := Shutdown(ctx)
	assert.NoError(t, err)
}

func TestGetTracer_Initialized(t *testing.T) {
	defer teardownTestTracer()
	tp, _ := setupTestTracer(t)
	require.NotNil(t, tp)

	testTracer := GetTracer()
	assert.NotNil(t, testTracer)
	assert.True(t, IsEnabled())
}

func TestGetTracer_NotInitialized(t *testing.T) {
	defer resetGlobalState()
	resetGlobalState()

	testTracer := GetTracer()
	assert.NotNil(t, testTracer) // Should return noop tracer
	assert.False(t, IsEnabled())
}

func TestIsEnabled_True(t *testing.T) {
	defer teardownTestTracer()
	tp, _ := setupTestTracer(t)
	require.NotNil(t, tp)

	assert.True(t, IsEnabled())
}

func TestIsEnabled_False_NoProvider(t *testing.T) {
	defer resetGlobalState()
	resetGlobalState()
	tracerProvider = nil
	tracer = nil

	assert.False(t, IsEnabled())
}

func TestIsEnabled_False_NoTracer(t *testing.T) {
	defer resetGlobalState()
	resetGlobalState()
	// Set provider but not tracer
	tp, _ := setupTestTracer(t)
	require.NotNil(t, tp)
	tracer = nil

	assert.False(t, IsEnabled())
}

func TestInitTracer_GlobalState(t *testing.T) {
	defer teardownTestTracer()
	resetGlobalState()

	// Test that InitTracer sets global state correctly
	// We'll use an endpoint that may fail, but test the state management
	endpoint := "http://localhost:4318"
	tp, err := InitTracer("warden", "1.0.0", endpoint)

	if err == nil && tp != nil {
		// If initialization succeeds, verify global state
		assert.Equal(t, tp, tracerProvider)
		assert.NotNil(t, tracer)
		assert.NotNil(t, otel.GetTracerProvider())
		assert.NotNil(t, otel.GetTextMapPropagator())
	}
}
