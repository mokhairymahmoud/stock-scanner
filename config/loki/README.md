# Loki Logging Infrastructure

This directory contains configuration for Loki (log aggregation) and Promtail (log collection) for centralized logging.

## Components

### Loki
Loki is a horizontally-scalable, highly-available log aggregation system inspired by Prometheus. It's designed to be very cost-effective and easy to operate.

**Configuration:** `loki-config.yaml`
- HTTP API on port 3100
- File system storage (for local development)
- 30-day retention period
- 16MB/s ingestion rate limit

### Promtail
Promtail is an agent that ships the contents of local logs to a private Grafana Loki instance or Grafana Cloud.

**Configuration:** `promtail-config.yaml`
- Collects logs from Docker containers
- Parses structured JSON logs from applications
- Extracts labels (service, level, trace_id, etc.)
- Ships logs to Loki

## Setup

### Docker Compose

Loki and Promtail are automatically started with Docker Compose:

```bash
docker-compose -f config/docker-compose.yaml up -d
```

Services:
- **Loki**: http://localhost:3100
- **Promtail**: Runs as a daemon, no direct access needed

### Accessing Logs

#### Via Grafana

1. Access Grafana: http://localhost:3000
2. Go to Explore → Select "Loki" datasource
3. Query logs using LogQL

#### Via Loki API

```bash
# Query logs
curl -G -s "http://localhost:3100/loki/api/v1/query_range" \
  --data-urlencode 'query={service="ingest"}' \
  --data-urlencode 'start=1234567890000000000' \
  --data-urlencode 'end=1234567891000000000' \
  --data-urlencode 'limit=100' | jq

# Get labels
curl http://localhost:3100/loki/api/v1/labels

# Get label values
curl http://localhost:3100/loki/api/v1/label/service/values
```

## LogQL Queries

LogQL is Loki's query language. Examples:

### Basic Queries

```logql
# All logs
{service=~".+"}

# Logs from specific service
{service="ingest"}

# Error logs
{level="error"}

# Logs with trace ID
{trace_id=~".+"}

# Multiple services
{service=~"ingest|bars|scanner"}
```

### Filtering

```logql
# Error logs from scanner
{service="scanner", level="error"}

# Logs containing "failed"
{service=~".+"} |= "failed"

# Logs not containing "debug"
{service=~".+"} != "debug"
```

### Aggregations

```logql
# Count logs per service
sum(count_over_time({service=~".+"}[1m])) by (service)

# Count logs per level
sum(count_over_time({service=~".+"}[1m])) by (level)

# Rate of error logs
sum(rate({level="error"}[5m])) by (service)
```

### Range Queries

```logql
# Logs over time window
{service="scanner"} [5m]

# Rate of logs
rate({service=~".+"}[1m])
```

## Log Labels

Promtail extracts the following labels from application logs:

- `service` - Service name (ingest, bars, scanner, etc.)
- `level` - Log level (debug, info, warn, error)
- `container` - Container name
- `trace_id` - Distributed tracing ID (if present)
- `span_id` - Span ID (if present)

## Log Format

Applications use structured JSON logging (zap logger):

```json
{
  "timestamp": "2024-01-15T10:30:45.123Z",
  "level": "info",
  "msg": "Starting scanner worker service",
  "service": "scanner",
  "worker_id": "worker-1",
  "trace_id": "abc123",
  "caller": "cmd/scanner/main.go:40"
}
```

## Configuration

### Loki Retention

Default retention is 30 days (720h). To change:

1. Edit `loki-config.yaml`:
```yaml
limits_config:
  retention_period: 168h  # 7 days
```

2. Restart Loki:
```bash
docker-compose -f config/docker-compose.yaml restart loki
```

### Promtail Scraping

Promtail automatically discovers Docker containers. To filter specific containers:

Edit `promtail-config.yaml`:
```yaml
relabel_configs:
  - source_labels: ['__meta_docker_container_label_com_docker_compose_project']
    regex: 'stock-scanner'
    action: keep
```

### Log Parsing

Promtail parses structured JSON logs. If your logs have a different format, update the pipeline stages in `promtail-config.yaml`.

## Troubleshooting

### No Logs Appearing

1. **Check Promtail is running:**
```bash
docker logs stock-scanner-promtail
```

2. **Check Loki is receiving logs:**
```bash
curl http://localhost:3100/loki/api/v1/labels
```

3. **Check container labels:**
```bash
docker inspect stock-scanner-ingest | jq '.[0].Config.Labels'
```

4. **Verify log format:**
```bash
docker logs stock-scanner-ingest | head -5
```

### High Memory Usage

Loki can use significant memory with high log volumes. Adjust limits in `loki-config.yaml`:

```yaml
limits_config:
  ingestion_rate_mb: 8  # Reduce from 16
  ingestion_burst_size_mb: 16  # Reduce from 32
```

### Missing Labels

If labels aren't being extracted:

1. Check Promtail pipeline stages match your log format
2. Verify JSON parsing is working
3. Check Promtail logs for errors

### Performance Issues

1. **Reduce retention period** if storage is limited
2. **Increase chunk size** for better compression
3. **Use object storage** (S3, GCS) instead of filesystem for production

## Production Considerations

### Storage

For production, use object storage instead of filesystem:

```yaml
common:
  storage:
    s3:
      bucket: loki-logs
      endpoint: s3.amazonaws.com
      region: us-east-1
```

### High Availability

1. Run multiple Loki instances behind a load balancer
2. Use shared storage (S3, GCS)
3. Configure replication factor > 1

### Security

1. Enable authentication in Loki
2. Use TLS for Loki API
3. Restrict Promtail network access
4. Encrypt logs at rest (object storage encryption)

### Monitoring

Monitor Loki and Promtail:
- Loki ingestion rate
- Query performance
- Storage usage
- Error rates

Add to Prometheus:
```yaml
scrape_configs:
  - job_name: 'loki'
    static_configs:
      - targets: ['loki:3100']
```

## Kubernetes Deployment

For Kubernetes, use:
- [Loki Helm Chart](https://github.com/grafana/helm-charts/tree/main/charts/loki)
- [Promtail DaemonSet](https://github.com/grafana/helm-charts/tree/main/charts/promtail)

Or use the Kubernetes manifests in `k8s/base/` (to be added).

## Resources

- [Loki Documentation](https://grafana.com/docs/loki/latest/)
- [LogQL Documentation](https://grafana.com/docs/loki/latest/logql/)
- [Promtail Documentation](https://grafana.com/docs/loki/latest/clients/promtail/)

