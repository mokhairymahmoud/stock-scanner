# Quick Start Guide - Testing & Deployment

## Prerequisites

- Docker and Docker Compose
- `jq` (optional, for JSON parsing)
- `redis-cli` (optional, for Redis inspection)
- `psql` (optional, for database inspection)

## Quick Deployment (5 Minutes)

### 1. Setup Environment

```bash
# Copy and edit environment file
cp config/env.example .env

# For testing with mock provider (no API keys needed), edit .env:
# MARKET_DATA_PROVIDER=mock
# MARKET_DATA_API_KEY=test-key
# MARKET_DATA_SYMBOLS=AAPL,MSFT,GOOGL
```

### 2. Deploy Everything

```bash
# Automated deployment
make docker-deploy

# Or manually:
./scripts/deploy.sh
```

This will:
- Start infrastructure (Redis, TimescaleDB, Prometheus, Grafana)
- Run database migrations
- Build and start Ingest and Bars services
- Wait for services to be healthy

### 3. Test Services

```bash
# Automated testing
make docker-test

# Or manually:
./scripts/test_services.sh
```

### 4. Verify Data Flow

```bash
# Wait 1-2 minutes for data to accumulate
sleep 120

# Check ticks stream
docker exec -it stock-scanner-redis redis-cli XLEN ticks

# Check finalized bars
docker exec -it stock-scanner-redis redis-cli XLEN bars.finalized

# Check database
docker exec -it stock-scanner-timescaledb psql -U postgres -d stock_scanner \
  -c "SELECT COUNT(*) FROM bars_1m;"
```

## Manual Testing

### Test Ingest Service

```bash
# Health check
curl http://localhost:8081/health | jq .

# Metrics
curl http://localhost:8081/metrics | grep stream_publish

# Check Redis stream
docker exec -it stock-scanner-redis redis-cli XREAD COUNT 5 STREAMS ticks 0
```

### Test Bars Service

```bash
# Health check
curl http://localhost:8083/health | jq .

# Metrics
curl http://localhost:8083/metrics | grep -E "(timescale|consume)"

# Check live bars
docker exec -it stock-scanner-redis redis-cli GET livebar:AAPL

# Check finalized bars stream
docker exec -it stock-scanner-redis redis-cli XREAD COUNT 5 STREAMS bars.finalized 0

# Check database
docker exec -it stock-scanner-timescaledb psql -U postgres -d stock_scanner \
  -c "SELECT symbol, timestamp, open, close, volume FROM bars_1m ORDER BY timestamp DESC LIMIT 5;"
```

## View Logs

```bash
# All services
make docker-logs

# Specific service
make docker-logs-service SERVICE=ingest
make docker-logs-service SERVICE=bars

# Follow logs
docker-compose -f config/docker-compose.yaml logs -f ingest bars
```

## Monitoring

- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/admin)
- **RedisInsight**: http://localhost:8001 (Redis GUI)
- **Service Health**:
  - Ingest: http://localhost:8081/health
  - Bars: http://localhost:8083/health
- **Metrics**:
  - Ingest: http://localhost:8081/metrics
  - Bars: http://localhost:8083/metrics

## Common Issues

### Services Not Starting

```bash
# Check infrastructure
docker-compose -f config/docker-compose.yaml ps

# Check logs
docker-compose -f config/docker-compose.yaml logs ingest
docker-compose -f config/docker-compose.yaml logs bars
```

### No Data Flow

1. Check provider connection:
   ```bash
   curl http://localhost:8081/health | jq .checks.provider
   ```

2. Check consumer status:
   ```bash
   curl http://localhost:8083/health | jq .checks.consumer
   ```

3. Check Redis stream:
   ```bash
   docker exec -it stock-scanner-redis redis-cli XLEN ticks
   ```

### Database Connection Issues

```bash
# Test database connection
docker exec -it stock-scanner-timescaledb psql -U postgres -d stock_scanner -c "SELECT 1;"

# Check database client status
curl http://localhost:8083/health | jq .checks.database
```

## Cleanup

```bash
# Stop all services
make docker-down

# Remove all data
docker-compose -f config/docker-compose.yaml down -v
```

## Next Steps

Once services are running:
1. Verify end-to-end data flow
2. Set up Grafana dashboards
3. Monitor metrics in Prometheus
4. Proceed with Phase 2: Indicator Engine

