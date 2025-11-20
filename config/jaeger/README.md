# Jaeger Distributed Tracing

This directory contains configuration and documentation for Jaeger distributed tracing infrastructure.

## Overview

Jaeger is used for distributed tracing across the Stock Scanner microservices. It helps track requests as they flow through the system, from tick ingestion to alert delivery.

## Architecture

Jaeger uses the all-in-one deployment which includes:
- **Jaeger Agent**: Receives traces from applications
- **Jaeger Collector**: Collects and processes traces
- **Jaeger Query**: Query service and UI
- **Storage Backend**: Badger (in-memory/file for development)

## Access

### Jaeger UI
- **URL**: http://localhost:16686
- **Features**:
  - Search traces by service, operation, tags
  - View trace timeline and spans
  - Analyze trace dependencies
  - View service map

### API Endpoints

- **OTLP gRPC**: `localhost:4317`
- **OTLP HTTP**: `localhost:4318`
- **Jaeger HTTP**: `localhost:14268`
- **Zipkin**: `localhost:9411`

## Integration

### OpenTelemetry (Recommended)

Jaeger supports OpenTelemetry Protocol (OTLP). Configure your OpenTelemetry exporter:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/sdk/trace"
)

func initTracing() error {
    exporter, err := otlptracegrpc.New(
        context.Background(),
        otlptracegrpc.WithEndpoint("jaeger:4317"),
        otlptracegrpc.WithInsecure(),
    )
    if err != nil {
        return err
    }

    tp := trace.NewTracerProvider(
        trace.WithBatcher(exporter),
        trace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String("stock-scanner-ingest"),
        )),
    )
    otel.SetTracerProvider(tp)
    return nil
}
```

### Jaeger Client (Legacy)

For direct Jaeger integration:

```go
import (
    "github.com/jaegertracing/jaeger-client-go"
    "github.com/jaegertracing/jaeger-client-go/config"
)

func initTracing() (jaeger.Tracer, error) {
    cfg := config.Configuration{
        ServiceName: "stock-scanner-ingest",
        Sampler: &config.SamplerConfig{
            Type:  jaeger.SamplerTypeConst,
            Param: 1.0, // 100% sampling for development
        },
        Reporter: &config.ReporterConfig{
            LogSpans:            true,
            LocalAgentHostPort:  "jaeger:6831",
        },
    }
    tracer, _, err := cfg.NewTracer()
    return tracer, err
}
```

## Trace Context Propagation

### HTTP Headers

When making HTTP requests between services, propagate trace context:

```go
import (
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/trace"
)

// Extract trace context from incoming request
propagator := propagation.TraceContext{}
ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

// Inject trace context into outgoing request
req, _ := http.NewRequest("GET", url, nil)
propagator.Inject(ctx, propagation.HeaderCarrier(req.Header))
```

### Redis Streams

For async message processing (Redis Streams), include trace context in message metadata:

```go
// When publishing
traceID := trace.SpanFromContext(ctx).SpanContext().TraceID().String()
spanID := trace.SpanFromContext(ctx).SpanContext().SpanID().String()

message := map[string]interface{}{
    "data": data,
    "trace_id": traceID,
    "span_id": spanID,
}

// When consuming
ctx := context.Background()
if traceID, ok := msg["trace_id"].(string); ok {
    // Create new span linked to parent trace
    ctx = trace.ContextWithSpanContext(ctx, ...)
}
```

## Key Spans to Instrument

### Data Pipeline
1. **Tick Ingestion** (`ingest.tick.receive`)
2. **Bar Aggregation** (`bars.aggregate`)
3. **Bar Finalization** (`bars.finalize`)
4. **Indicator Computation** (`indicator.compute`)
5. **Indicator Publishing** (`indicator.publish`)

### Scanner Worker
1. **Tick Processing** (`scanner.tick.process`)
2. **Indicator Update** (`scanner.indicator.update`)
3. **Rule Evaluation** (`scanner.rule.evaluate`)
4. **Alert Generation** (`scanner.alert.generate`)

### Alert Service
1. **Alert Consumption** (`alert.consume`)
2. **Deduplication** (`alert.deduplicate`)
3. **Filtering** (`alert.filter`)
4. **Persistence** (`alert.persist`)

### WebSocket Gateway
1. **Alert Delivery** (`ws-gateway.alert.deliver`)
2. **Connection Management** (`ws-gateway.connection`)

### API Service
1. **Request Handling** (`api.request.handle`)
2. **Rule Management** (`api.rule.manage`)
3. **Database Operations** (`api.db.query`)

## Sampling

### Development
- **100% sampling** - Capture all traces for debugging

### Production
- **1% sampling** - Reduce overhead while maintaining visibility
- **Error sampling** - Always sample errors (100%)

```go
sampler := trace.TraceIDRatioBased(0.01) // 1% sampling
if span.IsRecording() && span.Status().Code == codes.Error {
    sampler = trace.AlwaysSample() // Always sample errors
}
```

## Querying Traces

### Via Jaeger UI

1. Go to http://localhost:16686
2. Select service from dropdown
3. Select operation (optional)
4. Set time range
5. Click "Find Traces"

### Via API

```bash
# Search traces
curl "http://localhost:16686/api/traces?service=scanner&limit=20"

# Get trace by ID
curl "http://localhost:16686/api/traces/{trace-id}"
```

### Via Grafana

Grafana can query Jaeger traces:
1. Add Jaeger datasource in Grafana
2. Use Explore → Jaeger
3. Search and visualize traces

## Service Map

Jaeger automatically generates a service dependency map showing:
- Services and their relationships
- Request flow direction
- Error rates between services

Access via: http://localhost:16686/dependencies

## Performance Considerations

### Storage

For production, use persistent storage:
- **Elasticsearch** (recommended for production)
- **Cassandra** (high throughput)
- **Badger** (development only, in-memory)

Update docker-compose:
```yaml
jaeger:
  environment:
    - SPAN_STORAGE_TYPE=elasticsearch
    - ES_SERVER_URLS=http://elasticsearch:9200
```

### Retention

Default retention depends on storage:
- **Badger**: Limited by disk space
- **Elasticsearch**: Configure index lifecycle
- **Cassandra**: TTL-based retention

### Sampling Impact

High sampling rates increase:
- Storage usage
- Network traffic
- Processing overhead

Recommendations:
- Development: 100%
- Staging: 10%
- Production: 1-5%

## Troubleshooting

### No Traces Appearing

1. **Check service is sending traces:**
```bash
docker logs stock-scanner-jaeger | grep "Received"
```

2. **Verify OTLP endpoint:**
```bash
curl http://localhost:4318/v1/traces
```

3. **Check service configuration:**
   - Verify endpoint URL: `jaeger:4317` (gRPC) or `jaeger:4318` (HTTP)
   - Check sampling rate
   - Verify trace context propagation

### High Memory Usage

1. Reduce sampling rate
2. Use persistent storage (Elasticsearch)
3. Configure retention policies
4. Limit trace size (max spans per trace)

### Slow Query Performance

1. Use time-based queries (narrow time range)
2. Add service/operation filters
3. Use Elasticsearch for large deployments
4. Enable trace indexing

## Production Deployment

### Kubernetes

For Kubernetes, use Jaeger Operator:

```yaml
apiVersion: jaegertracing.io/v1
kind: Jaeger
metadata:
  name: jaeger
spec:
  strategy: production
  storage:
    type: elasticsearch
    elasticsearch:
      nodeCount: 3
      redundancyPolicy: SingleRedundancy
```

### High Availability

1. Run multiple Jaeger collectors
2. Use load balancer for collectors
3. Use shared storage (Elasticsearch cluster)
4. Configure replication

### Security

1. Enable authentication in Jaeger UI
2. Use TLS for OTLP endpoints
3. Restrict network access
4. Encrypt trace data at rest

## Best Practices

1. **Use OpenTelemetry** - Industry standard, vendor-agnostic
2. **Propagate context** - Ensure trace context flows through all services
3. **Meaningful span names** - Use consistent naming: `service.operation`
4. **Add attributes** - Include relevant metadata (symbol, rule_id, etc.)
5. **Sample intelligently** - Balance visibility with overhead
6. **Monitor trace health** - Track trace ingestion rate, errors
7. **Set trace limits** - Prevent runaway traces from breaking system

## Resources

- [Jaeger Documentation](https://www.jaegertracing.io/docs/)
- [OpenTelemetry Go](https://opentelemetry.io/docs/instrumentation/go/)
- [Jaeger Kubernetes Operator](https://github.com/jaegertracing/jaeger-operator)

