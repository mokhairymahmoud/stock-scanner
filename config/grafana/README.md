# Grafana Dashboards

This directory contains Grafana provisioning configuration and dashboards for monitoring the Stock Scanner application.

## Structure

```
config/grafana/
├── provisioning/
│   ├── datasources/
│   │   └── prometheus.yaml      # Prometheus datasource configuration
│   └── dashboards/
│       └── dashboards.yaml      # Dashboard provisioning configuration
├── dashboards/                  # Dashboard JSON files
│   ├── overview.json            # Overview dashboard (all services)
│   ├── data-pipeline.json       # Data pipeline dashboard (ingest, bars, indicator)
│   ├── scanner.json             # Scanner worker dashboard
│   ├── alerts.json              # Alert service and WebSocket gateway dashboard
│   └── api.json                 # REST API dashboard
└── README.md                    # This file
```

## Dashboards

### 1. Overview Dashboard
High-level view of all services:
- Service health status
- HTTP request rates across all services
- Error rates
- HTTP request duration (p95)
- Active pods count
- CPU and memory usage

### 2. Data Pipeline Dashboard
Monitors the data ingestion and processing pipeline:
- Request rates for ingest, bars, and indicator services
- Database write rates and latency
- Redis stream publish rates
- Database write queue depth
- Write errors

### 3. Scanner Dashboard
Critical metrics for the scanner worker:
- **Scan cycle duration** (p50, p95, p99) - SLO: p95 < 800ms
- Ticks processed rate
- Alerts emitted rate
- Worker queue depth
- Indicators processed rate
- Consumer lag (Redis streams)
- Active symbols, rules, cooldowns, and workers

### 4. Alerts Dashboard
Alert processing and delivery metrics:
- Alerts processed rate
- Alerts delivered rate (WebSocket)
- Active WebSocket connections
- Alert delivery latency (SLO: p95 < 2s)
- Alerts deduplicated rate
- Alert history write rate

### 5. API Dashboard
REST API service metrics:
- API request rate by endpoint
- Request duration (p50, p95, p99)
- Error rates (4xx, 5xx)
- Rule management operations

## Setup

The dashboards are automatically provisioned when Grafana starts via Docker Compose. The provisioning configuration is mounted from `config/grafana/provisioning/`.

### Accessing Grafana

1. Start the services:
```bash
docker-compose -f config/docker-compose.yaml up -d
```

2. Access Grafana at http://localhost:3000
   - Username: `admin`
   - Password: `admin`

3. Dashboards are available in the "Stock Scanner" folder

## Customization

### Adding New Dashboards

1. Create a new JSON file in `config/grafana/dashboards/`
2. Follow the Grafana dashboard JSON format
3. Restart Grafana or wait for auto-reload (10s interval)

### Modifying Existing Dashboards

1. Edit the JSON files in `config/grafana/dashboards/`
2. Changes are automatically picked up (10s refresh interval)
3. Or manually reload in Grafana UI: Dashboard → Settings → JSON Model → Save

### Metrics Reference

The dashboards use the following Prometheus metrics:

#### Common Metrics
- `http_requests_total` - HTTP request counter
- `http_request_duration_seconds` - HTTP request duration histogram
- `errors_total` - Error counter

#### Service-Specific Metrics
- `scan_cycle_seconds` - Scanner cycle duration histogram
- `ticks_processed_total` - Ticks processed counter
- `alerts_emitted_total` - Alerts emitted counter
- `alerts_delivered_total` - Alerts delivered counter
- `websocket_connections` - Active WebSocket connections gauge
- `timescale_write_total` - Database write counter
- `timescale_write_latency` - Database write latency histogram
- `timescale_write_queue_depth` - Database write queue depth gauge
- `publish_total` - Redis stream publish counter
- `consumer_lag` - Redis stream consumer lag gauge

## Troubleshooting

### Dashboards Not Appearing

1. Check Grafana logs:
```bash
docker logs stock-scanner-grafana
```

2. Verify provisioning volumes are mounted correctly in docker-compose.yaml

3. Check Prometheus datasource is configured:
   - Go to Configuration → Data Sources
   - Verify "Prometheus" datasource exists and is accessible

### Metrics Not Showing

1. Verify Prometheus is scraping metrics:
   - Go to Prometheus UI: http://localhost:9090
   - Check Targets: http://localhost:9090/targets

2. Verify metric names match what's exposed:
   - Query Prometheus: http://localhost:9090/graph
   - Search for metric names

3. Check service health endpoints:
   - `http://localhost:8081/metrics` (ingest)
   - `http://localhost:8083/metrics` (bars)
   - `http://localhost:8085/metrics` (indicator)
   - `http://localhost:8087/metrics` (scanner)
   - `http://localhost:8093/metrics` (alert)
   - `http://localhost:8089/metrics` (ws-gateway)
   - `http://localhost:8091/metrics` (api)

### Dashboard Refresh Issues

1. Check dashboard refresh interval (default: 10s)
2. Verify Prometheus scrape interval (default: 15s)
3. Check for time range issues (default: last 1 hour)

## Production Considerations

1. **Authentication**: Set up proper authentication for Grafana
2. **TLS**: Enable HTTPS for Grafana access
3. **Backup**: Export dashboards regularly or use version control
4. **Alerting**: Set up Grafana alerting rules based on dashboard metrics
5. **Performance**: Adjust refresh intervals based on load
6. **Retention**: Configure Prometheus retention policy for long-term storage

