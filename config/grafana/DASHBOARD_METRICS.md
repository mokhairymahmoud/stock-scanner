# Dashboard Metrics Status

This document tracks which metrics are implemented vs. which are referenced in dashboards.

## ✅ Implemented Metrics

These metrics exist and are working:

### Common Metrics (pkg/logger/metrics.go)
- `http_requests_total` - HTTP request counter
- `http_request_duration_seconds` - HTTP request duration histogram
- `errors_total` - Error counter

### Storage Metrics (internal/storage/timescale.go)
- `timescale_write_total` - Write operations counter (labels: `status`)
- `timescale_write_errors_total` - Write errors counter (labels: `error_type`)
- `timescale_write_latency_seconds` - Write latency histogram (labels: `operation`)
- `timescale_write_queue_depth` - Write queue depth gauge
- `timescale_write_batch_size` - Batch size histogram (labels: `operation`)

### Stream Publisher Metrics (internal/pubsub/stream_publisher.go)
- `stream_publish_total` - Messages published counter (labels: `stream`, `partition`)
- `stream_publish_errors_total` - Publish errors counter (labels: `stream`, `partition`)
- `stream_publish_latency_seconds` - Publish latency histogram (labels: `stream`, `partition`)
- `stream_publish_batch_size` - Batch size histogram (labels: `stream`)

## ⚠️ Missing Metrics (Referenced in Dashboards but Not Implemented)

These metrics are used in dashboards but need to be implemented:

### Scanner Metrics
- `scan_cycle_seconds` - Scan cycle duration histogram
- `ticks_processed_total` - Ticks processed counter
- `indicators_processed_total` - Indicators processed counter
- `alerts_emitted_total` - Alerts emitted counter
- `worker_queue_depth` - Worker queue depth gauge
- `consumer_lag` - Consumer lag gauge
- `symbol_count` - Active symbols gauge
- `rules_active_count` - Active rules gauge
- `cooldown_active_count` - Active cooldowns gauge

### Alert Service Metrics
- `alerts_processed_total` - Alerts processed counter
- `alerts_delivered_total` - Alerts delivered counter
- `alerts_deduplicated_total` - Alerts deduplicated counter
- `alert_delivery_latency_seconds` - Alert delivery latency histogram

### WebSocket Gateway Metrics
- `websocket_connections` - Active WebSocket connections gauge

### API Service Metrics
- `rules_created_total` - Rules created counter
- `rules_updated_total` - Rules updated counter
- `rules_deleted_total` - Rules deleted counter

## Dashboard Status

### ✅ Working Dashboards
- **Overview**: Uses `up`, `http_requests_total`, `errors_total`, `http_request_duration_seconds` - All working
- **Data Pipeline**: Uses `stream_publish_total`, `timescale_write_total`, `timescale_write_latency_seconds`, `timescale_write_queue_depth`, `timescale_write_errors_total` - All working

### ⚠️ Partially Working Dashboards
- **Scanner**: Most metrics missing (will show "No data" until implemented)
- **Alerts**: Most metrics missing (will show "No data" until implemented)
- **API**: Rule management metrics missing (will show "No data" until implemented)
- **Logs**: Should work if Loki is collecting logs

## Recommendations

1. **Short term**: The dashboards will show "No data" for missing metrics, which is expected
2. **Long term**: Implement the missing metrics in the respective services
3. **Alternative**: Remove or comment out panels that reference missing metrics until they're implemented

## How to Add Missing Metrics

Example for scanner service:

```go
// In internal/scanner/scan_loop.go
var (
    scanCycleDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "scan_cycle_seconds",
            Help:    "Scan cycle duration in seconds",
            Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
        },
        []string{"worker_id"},
    )
    
    ticksProcessedTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "ticks_processed_total",
            Help: "Total number of ticks processed",
        },
        []string{"worker_id"},
    )
)
```

Then use them in the code:
```go
scanCycleDuration.WithLabelValues(workerID).Observe(scanTime.Seconds())
ticksProcessedTotal.WithLabelValues(workerID).Inc()
```

