# Testing and Deployment Guide

This guide covers testing and deploying the implemented services (Ingest and Bars) using Docker.

## Prerequisites

- Docker and Docker Compose installed
- Go 1.24+ (for local testing)
- `make` (optional, for convenience commands)

## Quick Start

### 1. Set Up Environment

```bash
# Copy example environment file
cp config/env.example .env

# Edit .env with your configuration
# For testing with mock provider, set:
# MARKET_DATA_PROVIDER=mock
# MARKET_DATA_API_KEY=test-key
# MARKET_DATA_SYMBOLS=AAPL,MSFT,GOOGL
```

### 2. Start Infrastructure

```bash
# Start only infrastructure (Redis, TimescaleDB, Prometheus, Grafana)
make docker-up

# Or manually:
docker-compose -f config/docker-compose.yaml up -d redis timescaledb prometheus grafana
```

### 3. Run Database Migrations

```bash
# Wait for TimescaleDB to be ready (about 10 seconds)
sleep 10

# Run migrations
make migrate-up

# Or manually:
psql -h localhost -U postgres -d stock_scanner -f scripts/migrations/001_create_bars_table.sql
```

### 4. Build and Start Services

#### Option A: Docker Compose (Recommended)

```bash
# Build and start all services
make docker-up-all

# Or manually:
docker-compose -f config/docker-compose.yaml up -d --build ingest bars
```

#### Option B: Local Testing (Without Docker)

```bash
# Build services
make build

# Start infrastructure first
make docker-up

# In separate terminals, run services:
make run-ingest
make run-bars
```

## Testing Services

### 1. Test Ingest Service

#### Check Health
```bash
curl http://localhost:8081/health | jq .
```

Expected response:
```json
{
  "status": "healthy",
  "checks": {
    "provider": {
      "connected": true,
      "provider": "mock"
    }
  }
}
```

#### Check Metrics
```bash
curl http://localhost:8081/metrics | grep stream_publish
```

#### Verify Ticks Are Being Published
```bash
# Connect to Redis
docker exec -it stock-scanner-redis redis-cli

# Check stream length
XLEN ticks

# Read messages
XREAD COUNT 10 STREAMS ticks 0
```

### 2. Test Bars Service

#### Check Health
```bash
curl http://localhost:8083/health | jq .
```

Expected response:
```json
{
  "status": "UP",
  "checks": {
    "consumer": {
      "status": "ok",
      "running": true
    },
    "aggregator": {
      "status": "ok",
      "symbol_count": 3
    },
    "publisher": {
      "status": "ok",
      "running": true
    },
    "database": {
      "status": "ok",
      "running": true
    }
  }
}
```

#### Check Metrics
```bash
curl http://localhost:8083/metrics | grep -E "(timescale|stream_consume)"
```

#### Verify Live Bars
```bash
# Connect to Redis
docker exec -it stock-scanner-redis redis-cli

# Get live bar for a symbol
GET livebar:AAPL
```

#### Verify Finalized Bars Stream
```bash
# Check finalized bars stream
XLEN bars.finalized

# Read finalized bars
XREAD COUNT 10 STREAMS bars.finalized 0
```

#### Verify Bars in TimescaleDB
```bash
# Connect to TimescaleDB
docker exec -it stock-scanner-timescaledb psql -U postgres -d stock_scanner

# Query bars
SELECT * FROM bars_1m ORDER BY timestamp DESC LIMIT 10;

# Query by symbol
SELECT * FROM bars_1m WHERE symbol = 'AAPL' ORDER BY timestamp DESC LIMIT 10;
```

### 3. End-to-End Test

#### Automated Test Script
```bash
# Run the automated test script
./scripts/test_services.sh
```

#### Manual End-to-End Verification

1. **Start Services**
   ```bash
   make docker-up-all
   ```

2. **Wait for Services to Start** (about 30 seconds)
   ```bash
   # Check service status
   docker-compose -f config/docker-compose.yaml ps
   ```

3. **Verify Data Flow**
   ```bash
   # Check ingest service is publishing ticks
   docker exec -it stock-scanner-redis redis-cli XLEN ticks
   
   # Wait 1-2 minutes for bars to be finalized
   sleep 120
   
   # Check bars service has processed ticks
   docker exec -it stock-scanner-redis redis-cli XLEN bars.finalized
   
   # Check bars in database
   docker exec -it stock-scanner-timescaledb psql -U postgres -d stock_scanner \
     -c "SELECT COUNT(*) FROM bars_1m;"
   ```

## Monitoring

### Prometheus
- URL: http://localhost:9090
- View metrics from all services
- Query examples:
  - `stream_publish_total` - Total ticks published
  - `timescale_write_total` - Total bars written to DB
  - `stream_consume_total` - Total ticks consumed

### Grafana
- URL: http://localhost:3000
- Login: admin/admin
- Add Prometheus as data source: http://prometheus:9090

### RedisInsight
- URL: http://localhost:8001
- Redis GUI for inspecting data, streams, and keys
- Connect to Redis using:
  - Host: `redis` (from within Docker network) or `localhost` (from host)
  - Port: `6379`
  - No password (unless configured)

### Service Logs

```bash
# View all logs
make docker-logs

# View specific service logs
make docker-logs-service SERVICE=ingest
make docker-logs-service SERVICE=bars

# Or manually:
docker-compose -f config/docker-compose.yaml logs -f ingest
docker-compose -f config/docker-compose.yaml logs -f bars
```

## Troubleshooting

### Services Won't Start

1. **Check Infrastructure**
   ```bash
   # Verify Redis is running
   docker exec -it stock-scanner-redis redis-cli ping
   
   # Verify TimescaleDB is running
   docker exec -it stock-scanner-timescaledb pg_isready -U postgres
   ```

2. **Check Environment Variables**
   ```bash
   # Verify .env file exists
   cat .env
   
   # Check service environment
   docker exec stock-scanner-ingest env | grep MARKET_DATA
   ```

3. **Check Logs**
   ```bash
   docker-compose -f config/docker-compose.yaml logs ingest
   docker-compose -f config/docker-compose.yaml logs bars
   ```

### No Ticks Being Published

1. **Check Provider Connection**
   ```bash
   curl http://localhost:8081/health | jq .checks.provider
   ```

2. **Check Symbols Configuration**
   ```bash
   docker exec stock-scanner-ingest env | grep MARKET_DATA_SYMBOLS
   ```

3. **Check Redis Connection**
   ```bash
   docker exec stock-scanner-ingest env | grep REDIS
   ```

### No Bars Being Created

1. **Check Consumer Status**
   ```bash
   curl http://localhost:8083/health | jq .checks.consumer
   ```

2. **Check Redis Stream**
   ```bash
   docker exec -it stock-scanner-redis redis-cli XLEN ticks
   ```

3. **Check Database Connection**
   ```bash
   curl http://localhost:8083/health | jq .checks.database
   ```

4. **Check Logs**
   ```bash
   docker-compose -f config/docker-compose.yaml logs bars | grep -i error
   ```

### Database Connection Issues

1. **Verify Database is Ready**
   ```bash
   docker exec -it stock-scanner-timescaledb pg_isready -U postgres
   ```

2. **Check Connection String**
   ```bash
   docker exec stock-scanner-bars env | grep DB_
   ```

3. **Test Connection Manually**
   ```bash
   docker exec -it stock-scanner-timescaledb psql -U postgres -d stock_scanner -c "SELECT 1;"
   ```

## Performance Testing

### Load Test with Mock Provider

The mock provider can generate high volumes. To test:

1. **Increase Symbol Count**
   ```bash
   # Edit .env
   MARKET_DATA_SYMBOLS=AAPL,MSFT,GOOGL,TSLA,AMZN,NVDA,META,NFLX,AMD,INTC
   ```

2. **Restart Services**
   ```bash
   docker-compose -f config/docker-compose.yaml restart ingest bars
   ```

3. **Monitor Metrics**
   ```bash
   # Watch publish rate
   watch -n 1 'curl -s http://localhost:8081/metrics | grep stream_publish_total'
   
   # Watch consume rate
   watch -n 1 'curl -s http://localhost:8083/metrics | grep stream_consume_total'
   
   # Watch database write rate
   watch -n 1 'curl -s http://localhost:8083/metrics | grep timescale_write_total'
   ```

## Cleanup

### Stop All Services
```bash
make docker-down
```

### Remove All Data (Volumes)
```bash
docker-compose -f config/docker-compose.yaml down -v
```

### Rebuild Services
```bash
# Rebuild images
make docker-build

# Restart services
make docker-restart
```

## Next Steps

Once services are running and tested:
1. Verify data is flowing: Ingest → Bars → TimescaleDB
2. Check metrics in Prometheus
3. Set up Grafana dashboards
4. Proceed with Phase 2: Indicator Engine

