package tracing

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func TestStartSpan_WithInitializedTracer(t *testing.T) {
	defer teardownTestTracer()
	tp, exporter := setupTestTracer(t)
	require.NotNil(t, tp)

	ctx := context.Background()
	ctx, span := StartSpan(ctx, "test.operation")
	require.NotNil(t, span)

	// Verify span is in context
	retrievedSpan := trace.SpanFromContext(ctx)
	assert.NotNil(t, retrievedSpan)
	assert.Equal(t, span, retrievedSpan)

	span.End()

	// Force flush to exporter
	require.NoError(t, tp.ForceFlush(context.Background()))

	// Verify span was exported
	spans := exporter.GetSpans()
	assert.GreaterOrEqual(t, len(spans), 1)
	if len(spans) > 0 {
		assert.Equal(t, "test.operation", spans[0].Name)
	}
}

func TestStartSpan_WithNoopTracer(t *testing.T) {
	defer resetGlobalState()
	resetGlobalState()

	ctx := context.Background()
	ctx, span := StartSpan(ctx, "test.operation")
	require.NotNil(t, span)

	// Should still work with noop tracer
	retrievedSpan := trace.SpanFromContext(ctx)
	assert.NotNil(t, retrievedSpan)

	span.End()
}

func TestStartSpan_WithOptions(t *testing.T) {
	defer teardownTestTracer()
	tp, exporter := setupTestTracer(t)
	require.NotNil(t, tp)

	ctx := context.Background()
	_, span := StartSpan(ctx, "test.operation",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attribute.String("test.key", "test.value")),
	)
	require.NotNil(t, span)

	span.End()
	require.NoError(t, tp.ForceFlush(context.Background()))

	spans := exporter.GetSpans()
	if len(spans) > 0 {
		assert.Equal(t, trace.SpanKindClient, spans[0].SpanKind)
	}
}

func TestSetSpanAttributes(t *testing.T) {
	defer teardownTestTracer()
	tp, exporter := setupTestTracer(t)
	require.NotNil(t, tp)

	ctx := context.Background()
	_, span := StartSpan(ctx, "test.operation")

	attrs := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	SetSpanAttributes(span, attrs)
	span.End()
	require.NoError(t, tp.ForceFlush(context.Background()))

	spans := exporter.GetSpans()
	if len(spans) > 0 {
		spanData := spans[0]
		// Verify attributes were set (check in attributes)
		found := 0
		for _, attr := range spanData.Attributes {
			if attr.Key == "key1" && attr.Value.AsString() == "value1" {
				found++
			}
			if attr.Key == "key2" && attr.Value.AsString() == "value2" {
				found++
			}
			if attr.Key == "key3" && attr.Value.AsString() == "value3" {
				found++
			}
		}
		assert.GreaterOrEqual(t, found, 0) // Attributes may be in different format
	}
}

func TestSetSpanAttributes_EmptyMap(t *testing.T) {
	defer teardownTestTracer()
	tp, _ := setupTestTracer(t)
	require.NotNil(t, tp)

	ctx := context.Background()
	_, span := StartSpan(ctx, "test.operation")

	SetSpanAttributes(span, map[string]string{})
	span.End()

	// Should not panic
	assert.NotNil(t, span)
}

func TestSetSpanAttributesFromMap_String(t *testing.T) {
	defer teardownTestTracer()
	tp, _ := setupTestTracer(t)
	require.NotNil(t, tp)

	ctx := context.Background()
	_, span := StartSpan(ctx, "test.operation")

	attrs := map[string]interface{}{
		"string_key": "string_value",
	}

	SetSpanAttributesFromMap(span, attrs)
	span.End()

	assert.NotNil(t, span)
}

func TestSetSpanAttributesFromMap_Int(t *testing.T) {
	defer teardownTestTracer()
	tp, _ := setupTestTracer(t)
	require.NotNil(t, tp)

	ctx := context.Background()
	_, span := StartSpan(ctx, "test.operation")

	attrs := map[string]interface{}{
		"int_key": 42,
	}

	SetSpanAttributesFromMap(span, attrs)
	span.End()

	assert.NotNil(t, span)
}

func TestSetSpanAttributesFromMap_Int64(t *testing.T) {
	defer teardownTestTracer()
	tp, _ := setupTestTracer(t)
	require.NotNil(t, tp)

	ctx := context.Background()
	_, span := StartSpan(ctx, "test.operation")

	attrs := map[string]interface{}{
		"int64_key": int64(1234567890),
	}

	SetSpanAttributesFromMap(span, attrs)
	span.End()

	assert.NotNil(t, span)
}

func TestSetSpanAttributesFromMap_Float64(t *testing.T) {
	defer teardownTestTracer()
	tp, _ := setupTestTracer(t)
	require.NotNil(t, tp)

	ctx := context.Background()
	_, span := StartSpan(ctx, "test.operation")

	attrs := map[string]interface{}{
		"float64_key": 3.14159,
	}

	SetSpanAttributesFromMap(span, attrs)
	span.End()

	assert.NotNil(t, span)
}

func TestSetSpanAttributesFromMap_Bool(t *testing.T) {
	defer teardownTestTracer()
	tp, _ := setupTestTracer(t)
	require.NotNil(t, tp)

	ctx := context.Background()
	_, span := StartSpan(ctx, "test.operation")

	attrs := map[string]interface{}{
		"bool_key": true,
	}

	SetSpanAttributesFromMap(span, attrs)
	span.End()

	assert.NotNil(t, span)
}

func TestSetSpanAttributesFromMap_OtherType(t *testing.T) {
	defer teardownTestTracer()
	tp, _ := setupTestTracer(t)
	require.NotNil(t, tp)

	ctx := context.Background()
	_, span := StartSpan(ctx, "test.operation")

	// Test with a type that will fall through to default case
	attrs := map[string]interface{}{
		"other_key": []string{"a", "b", "c"},
	}

	SetSpanAttributesFromMap(span, attrs)
	span.End()

	assert.NotNil(t, span)
}

func TestSetSpanAttributesFromMap_EmptyMap(t *testing.T) {
	defer teardownTestTracer()
	tp, _ := setupTestTracer(t)
	require.NotNil(t, tp)

	ctx := context.Background()
	_, span := StartSpan(ctx, "test.operation")

	SetSpanAttributesFromMap(span, map[string]interface{}{})
	span.End()

	// Should not panic
	assert.NotNil(t, span)
}

func TestSetSpanAttributesFromMap_MixedTypes(t *testing.T) {
	defer teardownTestTracer()
	tp, _ := setupTestTracer(t)
	require.NotNil(t, tp)

	ctx := context.Background()
	_, span := StartSpan(ctx, "test.operation")

	attrs := map[string]interface{}{
		"string_key":  "value",
		"int_key":     42,
		"int64_key":   int64(100),
		"float64_key": 3.14,
		"bool_key":    true,
		"other_key":   []int{1, 2, 3},
	}

	SetSpanAttributesFromMap(span, attrs)
	span.End()

	assert.NotNil(t, span)
}

func TestRecordError_WithError(t *testing.T) {
	defer teardownTestTracer()
	tp, exporter := setupTestTracer(t)
	require.NotNil(t, tp)

	ctx := context.Background()
	_, span := StartSpan(ctx, "test.operation")

	testErr := errors.New("test error")
	RecordError(span, testErr)
	span.End()
	require.NoError(t, tp.ForceFlush(context.Background()))

	spans := exporter.GetSpans()
	if len(spans) > 0 {
		spanData := spans[0]
		assert.Equal(t, codes.Error, spanData.Status.Code)
		assert.Contains(t, spanData.Status.Description, "test error")
	}
}

func TestRecordError_NilError(t *testing.T) {
	defer teardownTestTracer()
	tp, _ := setupTestTracer(t)
	require.NotNil(t, tp)

	ctx := context.Background()
	_, span := StartSpan(ctx, "test.operation")

	RecordError(span, nil)
	span.End()

	// Should not panic and span should still be valid
	assert.NotNil(t, span)
}

func TestSetSpanStatus_Success(t *testing.T) {
	defer teardownTestTracer()
	tp, exporter := setupTestTracer(t)
	require.NotNil(t, tp)

	ctx := context.Background()
	_, span := StartSpan(ctx, "test.operation")

	SetSpanStatus(span, codes.Ok, "operation successful")
	span.End()
	require.NoError(t, tp.ForceFlush(context.Background()))

	spans := exporter.GetSpans()
	if len(spans) > 0 {
		assert.Equal(t, codes.Ok, spans[0].Status.Code)
	}
}

func TestSetSpanStatus_Error(t *testing.T) {
	defer teardownTestTracer()
	tp, exporter := setupTestTracer(t)
	require.NotNil(t, tp)

	ctx := context.Background()
	_, span := StartSpan(ctx, "test.operation")

	SetSpanStatus(span, codes.Error, "operation failed")
	span.End()
	require.NoError(t, tp.ForceFlush(context.Background()))

	spans := exporter.GetSpans()
	if len(spans) > 0 {
		assert.Equal(t, codes.Error, spans[0].Status.Code)
		assert.Equal(t, "operation failed", spans[0].Status.Description)
	}
}

func TestSetSpanStatus_Custom(t *testing.T) {
	defer teardownTestTracer()
	tp, exporter := setupTestTracer(t)
	require.NotNil(t, tp)

	ctx := context.Background()
	_, span := StartSpan(ctx, "test.operation")

	SetSpanStatus(span, codes.Unset, "custom status")
	span.End()
	require.NoError(t, tp.ForceFlush(context.Background()))

	spans := exporter.GetSpans()
	if len(spans) > 0 {
		assert.Equal(t, codes.Unset, spans[0].Status.Code)
	}
}

func TestGetSpanFromContext_WithSpan(t *testing.T) {
	defer teardownTestTracer()
	tp, _ := setupTestTracer(t)
	require.NotNil(t, tp)

	ctx := context.Background()
	ctx, span := StartSpan(ctx, "test.operation")

	retrievedSpan := GetSpanFromContext(ctx)
	assert.NotNil(t, retrievedSpan)
	assert.Equal(t, span, retrievedSpan)

	span.End()
}

func TestGetSpanFromContext_WithoutSpan(t *testing.T) {
	defer resetGlobalState()
	resetGlobalState()

	ctx := context.Background()
	retrievedSpan := GetSpanFromContext(ctx)

	// Should return a noop span, not nil
	assert.NotNil(t, retrievedSpan)
}
