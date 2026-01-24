# Warden OpenTelemetry Tracing

Warden service supports OpenTelemetry distributed tracing for monitoring and debugging inter-service call chains.

## Features

- **Automatic HTTP Request Tracing**: Automatically creates spans for all HTTP requests
- **User Query Tracing**: Adds detailed tracing information for the `/user` endpoint
- **Context Propagation**: Supports W3C Trace Context standard, seamlessly integrates with Stargate and Herald services
- **Configurable**: Enable/disable via environment variables or configuration files

## Configuration

### Environment Variables

```bash
# Enable OpenTelemetry tracing
OTLP_ENABLED=true

# OTLP endpoint (e.g., Jaeger, Tempo, OpenTelemetry Collector)
OTLP_ENDPOINT=http://localhost:4318
```

### Configuration File (YAML)

```yaml
tracing:
  enabled: true
  endpoint: "http://localhost:4318"
```

## Core Spans

### HTTP Request Span

All HTTP requests automatically create spans with the following attributes:
- `http.method`: HTTP method
- `http.url`: Request URL
- `http.status_code`: Response status code
- `http.user_agent`: User agent
- `http.remote_addr`: Client address

### User Query Span (`warden.get_user`)

Queries to the `/user` endpoint create dedicated spans containing:
- `warden.query.phone`: Queried phone number (masked)
- `warden.query.mail`: Queried email (masked)
- `warden.query.user_id`: Queried user ID
- `warden.user.found`: Whether user was found
- `warden.user.id`: Found user ID

## Usage Examples

### Starting Warden with Tracing Enabled

```bash
export OTLP_ENABLED=true
export OTLP_ENDPOINT=http://localhost:4318
./warden
```

### Using Tracing in Code

```go
import "github.com/soulteary/warden/internal/tracing"

// Create child span
ctx, span := tracing.StartSpan(ctx, "warden.custom_operation")
defer span.End()

// Set attributes
span.SetAttributes(attribute.String("key", "value"))

// Record error
if err != nil {
    tracing.RecordError(span, err)
}
```

## Integration with Stargate and Herald

Warden's tracing automatically integrates with the tracing context of Stargate and Herald services:

1. **Stargate** passes trace context via HTTP headers when calling Warden
2. **Warden** automatically extracts and continues the trace chain
3. Spans from all three services appear in the same trace

## Supported Tracing Backends

- **Jaeger**: `OTLP_ENDPOINT=http://localhost:4318`
- **Tempo**: `OTLP_ENDPOINT=http://localhost:4318`
- **OpenTelemetry Collector**: `OTLP_ENDPOINT=http://localhost:4318`
- **Other OTLP-compatible backends**

## Performance Considerations

- Tracing uses batch export by default, minimizing performance impact
- Trace data volume can be controlled via sampling rate
- Production environments should use sampling strategies (currently full sampling, suitable for development)

## Troubleshooting

### Tracing Not Enabled

Check environment variables:
```bash
echo $OTLP_ENABLED
echo $OTLP_ENDPOINT
```

### Trace Data Not Reaching Backend

1. Check if OTLP endpoint is accessible
2. Check network connection
3. Review error messages in Warden logs

### Missing Spans

Ensure you use `r.Context()` to pass context in request handling, rather than creating a new context.
