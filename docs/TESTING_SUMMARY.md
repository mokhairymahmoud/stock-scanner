# Testing and Deployment Summary

## Quick Reference

### One-Command Deployment
```bash
make docker-deploy
```

### One-Command Testing
```bash
make docker-test
```

### One-Command Verification
```bash
make docker-verify
```

## Step-by-Step Guide

### 1. Initial Setup (One Time)

```bash
# Copy environment file
cp config/env.example .env

# Edit .env for testing with mock provider:
# MARKET_DATA_PROVIDER=mock
# MARKET_DATA_API_KEY=test-key
# MARKET_DATA_SYMBOLS=AAPL,MSFT,GOOGL
```

### 2. Deploy Services

```bash
# Automated deployment
make docker-deploy

# This will:
# - Start infrastructure (Redis, TimescaleDB, Prometheus, Grafana)
# - Run database migrations
# - Build Docker images
# - Start Ingest and Bars services
# - Wait for health checks
```

### 3. Test Services

```bash
# Automated testing
make docker-test

# This will:
# - Check service health endpoints
# - Verify provider connections
# - Check metrics
# - Verify data flow
```

### 4. Verify Deployment

```bash
# Quick verification
make docker-verify

# Or manually check:
curl http://localhost:8081/health | jq .
curl http://localhost:8083/health | jq .
```

## Service Endpoints

### Ingest Service
- **Health**: http://localhost:8081/health
- **Metrics**: http://localhost:8081/metrics
- **Port**: 8080 (main), 8081 (health)

### Bars Service
- **Health**: http://localhost:8083/health
- **Metrics**: http://localhost:8083/metrics
- **Port**: 8082 (main), 8083 (health)

### Infrastructure
- **Redis**: localhost:6379
- **TimescaleDB**: localhost:5432
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/admin)

## Data Flow Verification

### 1. Check Ticks Stream (after 10-15 seconds)
```bash
docker exec -it stock-scanner-redis redis-cli XLEN ticks
```

### 2. Check Live Bars (after 10-15 seconds)
```bash
docker exec -it stock-scanner-redis redis-cli GET livebar:AAPL | jq .
```

### 3. Check Finalized Bars (after 1-2 minutes)
```bash
docker exec -it stock-scanner-redis redis-cli XLEN bars.finalized
```

### 4. Check Database (after 1-2 minutes)
```bash
docker exec -it stock-scanner-timescaledb psql -U postgres -d stock_scanner \
  -c "SELECT symbol, timestamp, open, close, volume FROM bars_1m ORDER BY timestamp DESC LIMIT 5;"
```

## Monitoring

### View Logs
```bash
# All services
make docker-logs

# Specific service
make docker-logs-service SERVICE=ingest
make docker-logs-service SERVICE=bars
```

### View Metrics
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000

### Key Metrics to Monitor
- `stream_publish_total` - Ticks published by ingest service
- `stream_consume_total` - Ticks consumed by bars service
- `timescale_write_total` - Bars written to database
- `bar_finalize_total` - Bars finalized

## Troubleshooting

### Services Not Starting
```bash
# Check service status
docker-compose -f config/docker-compose.yaml ps

# Check logs
docker-compose -f config/docker-compose.yaml logs ingest
docker-compose -f config/docker-compose.yaml logs bars
```

### No Data Flow
```bash
# Check provider connection
curl http://localhost:8081/health | jq .checks.provider

# Check consumer status
curl http://localhost:8083/health | jq .checks.consumer

# Check Redis stream
docker exec -it stock-scanner-redis redis-cli XLEN ticks
```

### Database Issues
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

# Remove all data (volumes)
docker-compose -f config/docker-compose.yaml down -v
```

## Documentation

- **Full Testing Guide**: [TESTING_AND_DEPLOYMENT.md](TESTING_AND_DEPLOYMENT.md)
- **Quick Start**: [QUICK_START.md](QUICK_START.md)
- **Deployment Checklist**: [DEPLOYMENT_CHECKLIST.md](DEPLOYMENT_CHECKLIST.md)

