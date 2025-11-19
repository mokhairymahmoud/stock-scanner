# End-to-End Testing Guide — Docker Environment

This guide provides comprehensive instructions for testing all system features using Docker. It covers the complete data flow from market data ingestion through rule evaluation and alert emission.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Initial Setup](#initial-setup)
3. [Starting the System](#starting-the-system)
4. [Testing Phase 1: Market Data Ingest](#testing-phase-1-market-data-ingest)
5. [Testing Phase 2: Bar Aggregation](#testing-phase-2-bar-aggregation)
6. [Testing Phase 3: Indicator Engine](#testing-phase-3-indicator-engine)
7. [Testing Phase 4: Scanner Worker](#testing-phase-4-scanner-worker)
8. [Complete End-to-End Flow](#complete-end-to-end-flow)
9. [Testing Rules and Alerts](#testing-rules-and-alerts)
10. [Testing Redis Rule Store](#testing-redis-rule-store)
11. [Monitoring and Observability](#monitoring-and-observability)
12. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Required Software

- **Docker** (version 20.10+)
- **Docker Compose** (version 2.0+)
- **curl** (for API testing)
- **jq** (optional, for JSON parsing)
- **redis-cli** (optional, for Redis inspection)

### Verify Installation

```bash
# Check Docker
docker --version
docker-compose --version

# Check optional tools
curl --version
jq --version  # Optional
redis-cli --version  # Optional
```

### Port Availability

Ensure the following ports are available:
- `6379` - Redis
- `5432` - TimescaleDB
- `8080-8091` - Go services
- `9090` - Prometheus
- `3000` - Grafana
- `8001` - RedisInsight

---

## Initial Setup

### 1. Clone and Navigate to Project

```bash
cd /path/to/stock-scanner
```

### 2. Create Environment File

```bash
# Copy example environment file
cp config/env.example .env

# Edit .env file with your configuration
# For testing, you can use the mock provider:
# MARKET_DATA_PROVIDER=mock
# MARKET_DATA_SYMBOLS=AAPL,MSFT,GOOGL,AMZN,TSLA
```

### 3. Build Services

```bash
# Build all Go services
make build

# Or build Docker images
make docker-build
```

---

## Starting the System

### Option 1: Start All Services (Recommended)

```bash
# Start all services including infrastructure and Go services
make docker-up-all

# Or manually:
docker-compose -f config/docker-compose.yaml up -d --build
```

### Option 2: Start Infrastructure Only

```bash
# Start only infrastructure (Redis, TimescaleDB, Prometheus, Grafana)
make docker-up

# Then run Go services locally:
make run-ingest &
make run-bars &
make run-indicator &
make run-scanner &
```

### Verify Services Are Running

```bash
# Check all containers
docker-compose -f config/docker-compose.yaml ps

# Check service health
make docker-verify

# Or manually check each service:
curl http://localhost:8081/health  # Ingest
curl http://localhost:8083/health   # Bars
curl http://localhost:8085/health   # Indicator
curl http://localhost:8087/health   # Scanner
```

### Service URLs

| Service | Health Check | Metrics | Ports |
|---------|-------------|---------|-------|
| Ingest | http://localhost:8081/health | http://localhost:8081/metrics | 8080, 8081 |
| Bars | http://localhost:8083/health | http://localhost:8083/metrics | 8082, 8083 |
| Indicator | http://localhost:8085/health | http://localhost:8085/metrics | 8084, 8085 |
| Scanner | http://localhost:8087/health | http://localhost:8087/metrics | 8086, 8087 |
| Prometheus | http://localhost:9090/-/healthy | http://localhost:9090/metrics | 9090 |
| Grafana | http://localhost:3000/api/health | - | 3000 |
| RedisInsight | http://localhost:8001/api/health | - | 8001 |

---

## Testing Phase 1: Market Data Ingest

### 1.1 Verify Ingest Service Health

```bash
curl http://localhost:8081/health | jq
```

**Expected Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-01T12:00:00Z",
  "checks": {
    "provider": {
      "status": "ok",
      "connected": true,
      "provider": "mock"
    },
    "publisher": {
      "status": "ok",
      "batch_size": 100
    }
  }
}
```

### 1.2 Check Ticks Are Being Published

```bash
# Wait 10-15 seconds for ticks to accumulate, then check:
docker exec stock-scanner-redis redis-cli XLEN ticks

# View recent ticks
docker exec stock-scanner-redis redis-cli XREAD COUNT 10 STREAMS ticks 0

# Monitor tick stream in real-time
docker exec stock-scanner-redis redis-cli XREAD BLOCK 1000 STREAMS ticks $
```

**Expected:** Stream length should increase over time.

### 1.3 Check Ingest Metrics

```bash
curl http://localhost:8081/metrics | grep stream_publish
```

**Key Metrics to Check:**
- `stream_publish_total` - Total ticks published
- `stream_publish_errors_total` - Publishing errors (should be 0)
- `stream_publish_latency_seconds` - Publishing latency

### 1.4 Verify Tick Format

```bash
# Get a sample tick
docker exec stock-scanner-redis redis-cli XREAD COUNT 1 STREAMS ticks 0 | \
  jq -r '.[0][1][0][1][1]' | jq
```

**Expected Format:**
```json
{
  "symbol": "AAPL",
  "price": 150.25,
  "size": 100,
  "timestamp": "2024-01-01T12:00:00Z",
  "type": "trade"
}
```

---

## Testing Phase 2: Bar Aggregation

### 2.1 Verify Bars Service Health

```bash
curl http://localhost:8083/health | jq
```

**Expected Response:**
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
      "symbol_count": 5
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

### 2.2 Check Live Bars

```bash
# Check live bar for a symbol
docker exec stock-scanner-redis redis-cli GET livebar:AAPL | jq

# Check all live bar keys
docker exec stock-scanner-redis redis-cli KEYS "livebar:*"
```

**Expected:** Live bars should be updated every few seconds.

### 2.3 Check Finalized Bars Stream

```bash
# Wait 1-2 minutes for a minute boundary to pass, then:
docker exec stock-scanner-redis redis-cli XLEN bars.finalized

# View recent finalized bars
docker exec stock-scanner-redis redis-cli XREAD COUNT 5 STREAMS bars.finalized 0
```

**Expected:** Finalized bars appear every minute.

### 2.4 Verify Bars in Database

```bash
# Connect to TimescaleDB and query bars
docker exec -it stock-scanner-timescaledb psql -U postgres -d stock_scanner -c \
  "SELECT symbol, timestamp, open, high, low, close, volume FROM bars_1m ORDER BY timestamp DESC LIMIT 10;"
```

**Expected:** Bars should be written to the database.

### 2.5 Check Bars Metrics

```bash
curl http://localhost:8083/metrics | grep bar
```

**Key Metrics:**
- `bars_finalized_total` - Total finalized bars
- `bars_published_total` - Bars published to Redis
- `bars_written_total` - Bars written to database

---

## Testing Phase 3: Indicator Engine

### 3.1 Verify Indicator Service Health

```bash
curl http://localhost:8085/health | jq
```

**Expected Response:**
```json
{
  "status": "UP",
  "checks": {
    "consumer": {
      "status": "ok",
      "running": true
    },
    "engine": {
      "status": "ok",
      "symbol_count": 5
    },
    "publisher": {
      "status": "ok",
      "running": true
    }
  }
}
```

### 3.2 Check Indicator Keys in Redis

```bash
# Check indicator for a symbol
docker exec stock-scanner-redis redis-cli GET "ind:AAPL" | jq

# List all indicator keys
docker exec stock-scanner-redis redis-cli KEYS "ind:*"
```

**Expected Format:**
```json
{
  "symbol": "AAPL",
  "timestamp": "2024-01-01T12:00:00Z",
  "values": {
    "rsi_14": 50.5,
    "ema_20": 150.25,
    "sma_50": 149.80,
    "vwap_5m": 150.10,
    "volume_avg_5m": 1000.0,
    "price_change_1m_pct": 0.5
  }
}
```

### 3.3 Monitor Indicator Updates via Pub/Sub

```bash
# Subscribe to indicator updates
docker exec stock-scanner-redis redis-cli PSUBSCRIBE "indicators.updated"
```

**Expected:** You should see messages when indicators are updated.

### 3.4 Verify Indicator Computation

```bash
# Check indicator metrics
curl http://localhost:8085/metrics | grep indicator

# Check specific indicators
docker exec stock-scanner-redis redis-cli GET "ind:AAPL" | jq '.values'
```

**Expected Indicators:**
- `rsi_14` - Relative Strength Index (14 period)
- `ema_20`, `ema_50`, `ema_200` - Exponential Moving Averages
- `sma_20`, `sma_50`, `sma_200` - Simple Moving Averages
- `vwap_5m`, `vwap_15m`, `vwap_1h` - Volume Weighted Average Price
- `volume_avg_5m`, `volume_avg_15m`, `volume_avg_1h` - Volume Averages
- `price_change_1m_pct`, `price_change_5m_pct`, `price_change_15m_pct` - Price Changes

---

## Testing Phase 4: Scanner Worker

### 4.1 Verify Scanner Service Health

```bash
curl http://localhost:8087/health | jq
```

**Expected Response:**
```json
{
  "status": "UP",
  "worker": {
    "id": "worker-1",
    "count": 1
  },
  "checks": {
    "state_manager": {
      "status": "ok",
      "symbol_count": 5
    },
    "scan_loop": {
      "status": "ok",
      "running": true,
      "stats": {
        "scan_cycles": 100,
        "symbols_scanned": 500,
        "rules_evaluated": 500,
        "rules_matched": 10,
        "alerts_emitted": 10
      }
    },
    "tick_consumer": {
      "status": "ok",
      "running": true
    },
    "indicator_consumer": {
      "status": "ok",
      "running": true
    },
    "bar_handler": {
      "status": "ok",
      "running": true
    }
  }
}
```

### 4.2 Check Scanner Statistics

```bash
curl http://localhost:8087/stats | jq
```

**Expected:** Detailed statistics about scan cycles, rules evaluated, alerts emitted.

### 4.3 Verify State Rehydration

```bash
# Check scanner logs for rehydration messages
docker logs stock-scanner-scanner | grep -i rehydrat

# Verify state manager has symbols
curl http://localhost:8087/health | jq '.checks.state_manager.symbol_count'
```

**Expected:** Scanner should log "State rehydration complete" on startup.

### 4.4 Monitor Scan Loop Performance

```bash
# Check scan loop metrics
curl http://localhost:8087/metrics | grep scan_cycle

# Check scan cycle time (should be < 800ms)
curl http://localhost:8087/stats | jq '.scan_loop.scan_cycle_time'
```

**Expected:** Scan cycle time should be well under 800ms (target).

---

## Complete End-to-End Flow

### Test Complete Pipeline: Tick → Bar → Indicator → Alert

#### Step 1: Verify Data Flow

```bash
# 1. Check ticks are flowing
docker exec stock-scanner-redis redis-cli XLEN ticks

# 2. Check bars are being finalized
docker exec stock-scanner-redis redis-cli XLEN bars.finalized

# 3. Check indicators are being computed
docker exec stock-scanner-redis redis-cli KEYS "ind:*" | wc -l

# 4. Check scanner is processing
curl http://localhost:8087/stats | jq '.scan_loop.scan_cycles'
```

#### Step 2: Monitor Real-Time Flow

```bash
# Terminal 1: Monitor ticks
watch -n 1 'docker exec stock-scanner-redis redis-cli XLEN ticks'

# Terminal 2: Monitor bars
watch -n 1 'docker exec stock-scanner-redis redis-cli XLEN bars.finalized'

# Terminal 3: Monitor indicators
watch -n 1 'docker exec stock-scanner-redis redis-cli KEYS "ind:*" | wc -l'

# Terminal 4: Monitor scanner stats
watch -n 1 'curl -s http://localhost:8087/stats | jq ".scan_loop"'
```

#### Step 3: Verify End-to-End Latency

```bash
# Get a tick timestamp
TICK_TIME=$(docker exec stock-scanner-redis redis-cli XREAD COUNT 1 STREAMS ticks 0 | \
  jq -r '.[0][1][0][1][1]' | jq -r '.timestamp')

# Wait for processing (30-60 seconds)
# Then check if corresponding bar exists
# Then check if indicators were computed
# Then check if scanner processed it
```

---

## Testing Rules and Alerts

### 5.1 Add a Rule (In-Memory Store)

If using in-memory store (default), rules must be added programmatically. For now, we'll test with Redis store.

### 5.2 Add a Rule (Redis Store)

First, configure scanner to use Redis store:

```bash
# Update .env file
echo "SCANNER_RULE_STORE_TYPE=redis" >> .env

# Restart scanner service
docker-compose -f config/docker-compose.yaml restart scanner
```

Then add a rule via Redis:

```bash
# Create a rule JSON
cat > /tmp/rule.json << 'EOF'
{
  "id": "rule-rsi-oversold",
  "name": "RSI Oversold",
  "description": "Alert when RSI drops below 30",
  "conditions": [
    {
      "metric": "rsi_14",
      "operator": "<",
      "value": 30.0
    }
  ],
  "cooldown": 300,
  "enabled": true
}
EOF

# Store rule in Redis
docker exec -i stock-scanner-redis redis-cli SET "rules:rule-rsi-oversold" "$(cat /tmp/rule.json)" EX 3600

# Add rule ID to set
docker exec stock-scanner-redis redis-cli SADD "rules:ids" "rule-rsi-oversold"

# Verify rule was stored
docker exec stock-scanner-redis redis-cli GET "rules:rule-rsi-oversold" | jq
```

### 5.3 Trigger Rule Reload

The scanner should automatically reload rules. Check logs:

```bash
docker logs stock-scanner-scanner | grep -i "reload"
```

### 5.4 Monitor Alerts

```bash
# Subscribe to alerts channel
docker exec stock-scanner-redis redis-cli PSUBSCRIBE "alerts"

# Or check alerts stream
docker exec stock-scanner-redis redis-cli XLEN alerts

# View recent alerts
docker exec stock-scanner-redis redis-cli XREAD COUNT 10 STREAMS alerts 0
```

**Expected Alert Format:**
```json
{
  "id": "uuid",
  "rule_id": "rule-rsi-oversold",
  "rule_name": "RSI Oversold",
  "symbol": "AAPL",
  "timestamp": "2024-01-01T12:00:00Z",
  "price": 150.25,
  "message": "Rule 'RSI Oversold' matched for AAPL",
  "metadata": {
    "rsi_14": 25.5
  },
  "trace_id": "uuid"
}
```

### 5.5 Verify Cooldown

```bash
# Check scanner stats after an alert
curl http://localhost:8087/stats | jq '.cooldown_tracker.cooldown_count'

# The same rule should not fire again for the cooldown period (300 seconds)
```

### 5.6 Test Multiple Rules

```bash
# Add another rule
cat > /tmp/rule2.json << 'EOF'
{
  "id": "rule-ema-cross",
  "name": "EMA Cross Above",
  "conditions": [
    {
      "metric": "ema_20",
      "operator": ">",
      "value": 150.0
    }
  ],
  "cooldown": 60,
  "enabled": true
}
EOF

docker exec -i stock-scanner-redis redis-cli SET "rules:rule-ema-cross" "$(cat /tmp/rule2.json)" EX 3600
docker exec stock-scanner-redis redis-cli SADD "rules:ids" "rule-ema-cross"
```

### 5.7 Verify Rule Evaluation

```bash
# Check scanner stats
curl http://localhost:8087/stats | jq '.scan_loop'

# Should show:
# - rules_evaluated: increasing
# - rules_matched: when conditions are met
# - alerts_emitted: when alerts are generated
```

---

## Testing Redis Rule Store

### 6.1 Verify Redis Store Configuration

```bash
# Check scanner is using Redis store
docker logs stock-scanner-scanner | grep -i "redis rule store"
```

### 6.2 Test Rule CRUD Operations

```bash
# List all rule IDs
docker exec stock-scanner-redis redis-cli SMEMBERS "rules:ids"

# Get a specific rule
docker exec stock-scanner-redis redis-cli GET "rules:rule-rsi-oversold" | jq

# Update a rule
cat > /tmp/rule-updated.json << 'EOF'
{
  "id": "rule-rsi-oversold",
  "name": "RSI Oversold (Updated)",
  "conditions": [
    {
      "metric": "rsi_14",
      "operator": "<",
      "value": 25.0
    }
  ],
  "cooldown": 600,
  "enabled": true
}
EOF

docker exec -i stock-scanner-redis redis-cli SET "rules:rule-rsi-oversold" "$(cat /tmp/rule-updated.json)" EX 3600

# Disable a rule
docker exec stock-scanner-redis redis-cli GET "rules:rule-rsi-oversold" | \
  jq '.enabled = false' | \
  docker exec -i stock-scanner-redis redis-cli SET "rules:rule-rsi-oversold" "$(cat)" EX 3600

# Delete a rule
docker exec stock-scanner-redis redis-cli DEL "rules:rule-rsi-oversold"
docker exec stock-scanner-redis redis-cli SREM "rules:ids" "rule-rsi-oversold"
```

### 6.3 Test Rule Persistence

```bash
# Add a rule
# ... (as above)

# Restart scanner
docker-compose -f config/docker-compose.yaml restart scanner

# Verify rule still exists after restart
docker exec stock-scanner-redis redis-cli GET "rules:rule-rsi-oversold" | jq
```

---

## Monitoring and Observability

### 7.1 Prometheus Metrics

```bash
# Access Prometheus UI
open http://localhost:9090

# Query metrics
# Example: Total ticks published
curl http://localhost:9090/api/v1/query?query=stream_publish_total

# Example: Scan cycle time
curl http://localhost:9090/api/v1/query?query=scan_cycle_seconds
```

### 7.2 Grafana Dashboards

```bash
# Access Grafana
open http://localhost:3000
# Login: admin / admin

# Create dashboards for:
# - Tick ingestion rate
# - Bar aggregation rate
# - Indicator computation latency
# - Scan cycle performance
# - Alert emission rate
```

### 7.3 RedisInsight

```bash
# Access RedisInsight
open http://localhost:8001

# Explore:
# - Streams (ticks, bars.finalized, alerts)
# - Keys (livebar:*, ind:*, rules:*)
# - Pub/Sub channels (indicators.updated, alerts)
```

### 7.4 Service Logs

```bash
# View all logs
docker-compose -f config/docker-compose.yaml logs -f

# View specific service logs
docker logs -f stock-scanner-ingest
docker logs -f stock-scanner-bars
docker logs -f stock-scanner-indicator
docker logs -f stock-scanner-scanner

# Or use Makefile
make docker-logs-service SERVICE=scanner
```

---

## Troubleshooting

### Issue: Services Not Starting

```bash
# Check Docker Compose status
docker-compose -f config/docker-compose.yaml ps

# Check logs for errors
docker-compose -f config/docker-compose.yaml logs

# Verify environment file
cat .env | grep -v "^#"
```

### Issue: No Ticks Being Published

```bash
# Check ingest service health
curl http://localhost:8081/health

# Check provider connection
curl http://localhost:8081/health | jq '.checks.provider.connected'

# Check ingest logs
docker logs stock-scanner-ingest | tail -50

# Verify Redis connection
docker exec stock-scanner-redis redis-cli ping
```

### Issue: Bars Not Being Finalized

```bash
# Check bars service health
curl http://localhost:8083/health

# Check if ticks are being consumed
curl http://localhost:8083/health | jq '.checks.consumer.stats'

# Wait for minute boundary (bars finalize at :00 of each minute)
# Check bars service logs
docker logs stock-scanner-bars | grep -i "finalize"
```

### Issue: Indicators Not Computing

```bash
# Check indicator service health
curl http://localhost:8085/health

# Verify finalized bars exist
docker exec stock-scanner-redis redis-cli XLEN bars.finalized

# Check indicator logs
docker logs stock-scanner-indicator | tail -50
```

### Issue: Scanner Not Evaluating Rules

```bash
# Check scanner health
curl http://localhost:8087/health

# Verify rules exist
docker exec stock-scanner-redis redis-cli SMEMBERS "rules:ids"

# Check scanner logs for rule loading
docker logs stock-scanner-scanner | grep -i rule

# Verify state manager has symbols
curl http://localhost:8087/stats | jq '.state_manager.symbol_count'
```

### Issue: No Alerts Being Emitted

```bash
# Check if rules match conditions
# 1. Check current indicator values
docker exec stock-scanner-redis redis-cli GET "ind:AAPL" | jq '.values'

# 2. Verify rule conditions
docker exec stock-scanner-redis redis-cli GET "rules:rule-rsi-oversold" | jq '.conditions'

# 3. Check cooldown status
curl http://localhost:8087/stats | jq '.cooldown_tracker'

# 4. Check alert emitter stats
curl http://localhost:8087/stats | jq '.alert_emitter'
```

### Issue: High Scan Cycle Time

```bash
# Check scan cycle metrics
curl http://localhost:8087/metrics | grep scan_cycle

# Check number of symbols
curl http://localhost:8087/stats | jq '.state_manager.symbol_count'

# Check number of rules
docker exec stock-scanner-redis redis-cli SMEMBERS "rules:ids" | wc -l

# If too high, consider:
# - Reducing symbol universe
# - Reducing number of rules
# - Scaling scanner workers
```

### Issue: Redis Connection Errors

```bash
# Verify Redis is running
docker exec stock-scanner-redis redis-cli ping

# Check Redis logs
docker logs stock-scanner-redis

# Verify network connectivity
docker exec stock-scanner-scanner ping redis
```

### Issue: Database Connection Errors

```bash
# Verify TimescaleDB is running
docker exec stock-scanner-timescaledb pg_isready -U postgres

# Check database logs
docker logs stock-scanner-timescaledb

# Verify connection string in .env
grep DB_HOST .env
```

---

## Quick Test Checklist

Use this checklist to quickly verify the system is working:

- [ ] All infrastructure services healthy (Redis, TimescaleDB, Prometheus, Grafana)
- [ ] All Go services healthy (Ingest, Bars, Indicator, Scanner)
- [ ] Ticks are being published to Redis stream
- [ ] Live bars are being updated in Redis
- [ ] Finalized bars are being written to database
- [ ] Indicators are being computed and stored
- [ ] Scanner is consuming ticks, indicators, and bars
- [ ] Rules are loaded and being evaluated
- [ ] Alerts are being emitted when rules match
- [ ] Cooldown is preventing duplicate alerts
- [ ] Metrics are being collected in Prometheus
- [ ] All services are logging correctly

---

## Performance Benchmarks

### Expected Performance Targets

- **Tick Ingestion:** 100+ ticks/second per symbol
- **Bar Finalization:** 1 bar per minute per symbol
- **Indicator Computation:** < 100ms per bar
- **Scan Cycle:** < 800ms for 2000 symbols
- **Alert Emission:** < 50ms per alert

### Load Testing

```bash
# Test with more symbols
export MARKET_DATA_SYMBOLS=AAPL,MSFT,GOOGL,TSLA,AMZN,NVDA,META,NFLX,AMD,INTC
docker-compose -f config/docker-compose.yaml restart ingest

# Monitor performance
watch -n 1 'curl -s http://localhost:8087/stats | jq ".scan_loop.scan_cycle_time"'
```

---

## Cleanup

### Stop All Services

```bash
# Stop all services
make docker-down

# Or manually:
docker-compose -f config/docker-compose.yaml down
```

### Remove All Data

```bash
# Stop services
docker-compose -f config/docker-compose.yaml down

# Remove volumes (WARNING: Deletes all data)
docker-compose -f config/docker-compose.yaml down -v
```

### Reset Specific Service

```bash
# Restart a specific service
docker-compose -f config/docker-compose.yaml restart scanner

# Rebuild and restart
docker-compose -f config/docker-compose.yaml up -d --build scanner
```

---

## Next Steps

After completing these tests:

1. **Phase 4 Testing:** Test Alert Service and WebSocket Gateway (when implemented)
2. **Phase 5 Testing:** Test REST API for rule management (when implemented)
3. **Load Testing:** Test with 1000+ symbols
4. **Chaos Testing:** Test failure scenarios (network interruptions, service restarts)
5. **Integration Testing:** Test with real market data providers

---

## Additional Resources

- **Architecture Documentation:** `architecture.md`
- **Implementation Plan:** `implementation_plan.md`
- **Quick Start Guide:** `docs/QUICK_START.md`
- **Deployment Checklist:** `docs/DEPLOYMENT_CHECKLIST.md`

---

## Support

If you encounter issues:

1. Check service logs: `docker logs <service-name>`
2. Verify health endpoints: `curl http://localhost:<port>/health`
3. Check Redis data: Use RedisInsight or `redis-cli`
4. Review this guide's troubleshooting section
5. Check implementation plan for expected behavior

---

**Last Updated:** 2024-01-01
**Version:** 1.0

