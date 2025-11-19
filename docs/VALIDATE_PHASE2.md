# Phase 2: Indicator Engine - Validation Guide

This guide provides comprehensive instructions for validating that Phase 2 (Indicator Engine) is working correctly when running Docker services.

## Prerequisites

- Docker and Docker Compose installed
- `jq` installed (optional, for JSON parsing)
- Services started via `make docker-up-all` or `docker-compose up -d`

## Quick Validation

Run the automated validation script:

```bash
./scripts/validate_phase2.sh
```

This script performs all validation checks automatically and provides a summary.

## Manual Validation Steps

### Step 1: Start Docker Services

```bash
# Start all services (infrastructure + Go services)
make docker-up-all

# Or start infrastructure first, then services
make docker-up
# Then build and run services
make build
make run-ingest  # In one terminal
make run-bars    # In another terminal
make run-indicator  # In another terminal
```

### Step 2: Verify Service Health

Check that the indicator service is running and healthy:

```bash
# Check health endpoint
curl http://localhost:8085/health | jq .

# Expected response:
# {
#   "status": "UP",
#   "timestamp": "2025-11-19T...",
#   "checks": {
#     "consumer": {
#       "status": "ok",
#       "running": true,
#       "stats": { ... }
#     },
#     "engine": {
#       "status": "ok",
#       "symbol_count": 5
#     },
#     "publisher": {
#       "status": "ok",
#       "running": true
#     }
#   }
# }
```

**What to verify:**
- ✅ `status` should be `"UP"`
- ✅ `checks.consumer.running` should be `true`
- ✅ `checks.publisher.running` should be `true`
- ✅ `checks.engine.symbol_count` should be `> 0` after bars are processed

### Step 3: Check Service Logs

Monitor the indicator service logs to see real-time activity:

```bash
# View indicator service logs
docker logs -f stock-scanner-indicator

# Or using Makefile
make docker-logs-service SERVICE=indicator
```

**What to look for:**
- ✅ `"Starting indicator engine service"`
- ✅ `"Starting bar consumer"`
- ✅ `"Indicator engine service started"`
- ✅ `"Published indicators"` (debug logs when indicators are computed)
- ❌ Any error messages

### Step 4: Verify Data Flow

#### 4.1 Check Finalized Bars Stream

The indicator service consumes from the `bars.finalized` stream. Verify bars are being finalized:

```bash
# Check stream length
docker exec stock-scanner-redis redis-cli XLEN bars.finalized

# Should show > 0 after a minute boundary passes
# Example output: (integer) 15
```

**Note:** If this returns `0`, wait 1-2 minutes for the first minute boundary to occur.

#### 4.2 Check Consumer Group

Verify the indicator service is consuming from the stream:

```bash
# Check consumer group info
docker exec stock-scanner-redis redis-cli XINFO GROUPS bars.finalized

# Should show "indicator-engine" consumer group with:
# - name: indicator-engine
# - consumers: 1
# - pending: number of unprocessed messages
# - last-delivered-id: last message ID
```

#### 4.3 Verify Indicators Are Published to Redis

Check that indicators are being stored in Redis:

```bash
# List all indicator keys
docker exec stock-scanner-redis redis-cli KEYS "ind:*"

# Should show keys like:
# 1) "ind:AAPL"
# 2) "ind:MSFT"
# 3) "ind:GOOGL"
# ...
```

**Get indicator data for a specific symbol:**

```bash
# Get indicator data for AAPL
docker exec stock-scanner-redis redis-cli GET "ind:AAPL" | jq .

# Expected structure:
# {
#   "symbol": "AAPL",
#   "timestamp": "2025-11-19T10:57:00Z",
#   "values": {
#     "rsi_14": 65.5,
#     "ema_20": 150.2,
#     "ema_50": 148.9,
#     "ema_200": 145.8,
#     "sma_20": 149.8,
#     "sma_50": 147.2,
#     "sma_200": 144.5,
#     "vwap_5m": 150.1,
#     "vwap_15m": 149.9,
#     "vwap_1h": 149.5,
#     "volume_avg_5m": 1250000.0,
#     "volume_avg_15m": 1200000.0,
#     "volume_avg_1h": 1150000.0,
#     "price_change_1m_pct": 0.5,
#     "price_change_5m_pct": 1.2,
#     "price_change_15m_pct": 2.1
#   }
# }
```

**What to verify:**
- ✅ Keys exist for symbols that have received finalized bars
- ✅ Each key contains a JSON object with `symbol`, `timestamp`, and `values`
- ✅ `values` object contains multiple indicator types
- ✅ Indicator values are numeric (not null/undefined)

#### 4.4 Check Pub/Sub Channel

The indicator service publishes updates to the `indicators.updated` channel:

```bash
# Monitor the channel in real-time (in a separate terminal)
docker exec -it stock-scanner-redis redis-cli PSUBSCRIBE "indicators.updated"

# You should see messages like:
# 1) "pmessage"
# 2) "indicators.updated"
# 3) "indicators.updated"
# 4) "{\"symbol\":\"AAPL\",\"timestamp\":\"2025-11-19T10:57:00Z\"}"
```

**Note:** Press `Ctrl+C` to stop monitoring.

### Step 5: Verify Metrics

Check Prometheus metrics for indicator-related data:

```bash
# Get all metrics
curl http://localhost:8085/metrics

# Filter for indicator-related metrics
curl http://localhost:8085/metrics | grep -i indicator

# Look for metrics like:
# - indicator_computation_total
# - indicator_publish_total
# - indicator_computation_latency_seconds
# - indicator_engine_symbols_total
```

### Step 6: Use RedisInsight (Visual Validation)

RedisInsight provides a GUI for inspecting Redis data:

1. **Open RedisInsight:**
   - URL: http://localhost:8001
   - Browser should open automatically

2. **Connect to Redis:**
   - Click "Add Redis Database"
   - Host: `redis` (or `localhost` if connecting from host)
   - Port: `6379`
   - Name: `Stock Scanner Redis`
   - Click "Add Redis Database"

3. **Browse Indicator Keys:**
   - Go to "Browser" tab
   - Filter: `ind:*`
   - Click on a key (e.g., `ind:AAPL`)
   - View the JSON structure with all indicator values

4. **Check Streams:**
   - Go to "Browser" tab
   - Find `bars.finalized` stream
   - View messages in the stream
   - Check consumer groups

5. **Monitor Pub/Sub:**
   - Go to "Pub/Sub" tab
   - Subscribe to channel: `indicators.updated`
   - See real-time updates when indicators are published

### Step 7: End-to-End Validation

Verify the complete data flow:

```bash
# 1. Check ticks are being generated
docker exec stock-scanner-redis redis-cli XLEN ticks
# Should be > 0

# 2. Check bars are being finalized
docker exec stock-scanner-redis redis-cli XLEN bars.finalized
# Should be > 0 (after minute boundary)

# 3. Check indicators are being computed
docker exec stock-scanner-redis redis-cli KEYS "ind:*" | wc -l
# Should be > 0 (after bars are processed)

# 4. Check a specific indicator value
docker exec stock-scanner-redis redis-cli GET "ind:AAPL" | jq '.values.rsi_14'
# Should return a number between 0-100
```

### Step 8: Expected Timeline

When starting fresh services, expect this timeline:

1. **0-10 seconds:** Services start up
   - Infrastructure (Redis, TimescaleDB) starts
   - Go services start

2. **10-30 seconds:** Ingest service generates ticks
   - Ticks appear in `ticks` stream
   - Bars service starts consuming

3. **30-60 seconds:** Bars service aggregates ticks
   - Live bars are published to `livebar:{symbol}` keys
   - Waiting for minute boundary

4. **60+ seconds:** First minute boundary occurs
   - Bars are finalized and published to `bars.finalized` stream
   - Indicator service consumes finalized bars

5. **60-90 seconds:** Indicators are computed
   - Some indicators are ready immediately (EMA, SMA after enough bars)
   - RSI needs 15 bars (15 minutes of data)
   - Indicators appear in `ind:{symbol}` keys

6. **90+ seconds:** Full indicator set available
   - All indicators should be computed
   - Pub/sub notifications are sent

### Step 9: Troubleshooting

#### Problem: No indicators appearing

**Check 1: Is the indicator service running?**
```bash
docker ps | grep indicator
curl http://localhost:8085/health
```

**Check 2: Are bars being finalized?**
```bash
docker exec stock-scanner-redis redis-cli XLEN bars.finalized
# If 0, check bars service
curl http://localhost:8083/health
```

**Check 3: Is the consumer group active?**
```bash
docker exec stock-scanner-redis redis-cli XINFO GROUPS bars.finalized
# Should show "indicator-engine" group
```

**Check 4: Check service logs for errors**
```bash
docker logs stock-scanner-indicator 2>&1 | grep -i error
```

#### Problem: Indicators show 0 or null values

**Possible causes:**
- Not enough bars yet (RSI needs 15 bars, some indicators need more)
- Wait 2-3 minutes for enough data to accumulate
- Check that bars are actually being finalized

**Verify:**
```bash
# Check how many bars have been processed
docker exec stock-scanner-redis redis-cli XLEN bars.finalized

# Check symbol count in engine
curl http://localhost:8085/health | jq '.checks.engine.symbol_count'
```

#### Problem: Service health shows DOWN

**Check service logs:**
```bash
docker logs stock-scanner-indicator --tail 50
```

**Common issues:**
- Redis connection failed: Check `REDIS_HOST` environment variable
- Consumer group creation failed: Check Redis permissions
- Port conflict: Check if port 8085 is already in use

#### Problem: Consumer not processing messages

**Check consumer group lag:**
```bash
docker exec stock-scanner-redis redis-cli XINFO GROUPS bars.finalized
# Look at "pending" count - should be low or 0
```

**If pending is high:**
- Consumer may be stuck
- Restart the indicator service: `docker restart stock-scanner-indicator`

### Step 10: Validation Checklist

Use this checklist to ensure everything is working:

- [ ] Indicator service health endpoint returns `UP`
- [ ] Consumer is running (`checks.consumer.running: true`)
- [ ] Publisher is running (`checks.publisher.running: true`)
- [ ] Finalized bars stream has messages (`XLEN bars.finalized > 0`)
- [ ] Consumer group exists and is consuming
- [ ] Indicator keys exist in Redis (`KEYS ind:*` returns keys)
- [ ] Indicator values are populated (not null/empty)
- [ ] Multiple indicator types are computed (RSI, EMA, SMA, VWAP, etc.)
- [ ] Pub/sub channel is receiving updates
- [ ] Metrics are available
- [ ] No errors in service logs

### Step 11: Advanced Validation

#### Test with Specific Symbol

If you know a symbol is being tracked:

```bash
SYMBOL="AAPL"

# Check if indicator exists
docker exec stock-scanner-redis redis-cli EXISTS "ind:$SYMBOL"

# Get all indicators for symbol
docker exec stock-scanner-redis redis-cli GET "ind:$SYMBOL" | jq '.values'

# Check specific indicator
docker exec stock-scanner-redis redis-cli GET "ind:$SYMBOL" | jq '.values.rsi_14'
```

#### Monitor Real-Time Updates

Watch indicators update in real-time:

```bash
# Terminal 1: Monitor pub/sub
docker exec -it stock-scanner-redis redis-cli PSUBSCRIBE "indicators.updated"

# Terminal 2: Watch a specific indicator key
watch -n 1 'docker exec stock-scanner-redis redis-cli GET "ind:AAPL" | jq ".values.rsi_14"'
```

#### Check Indicator Calculation Accuracy

Verify indicators are calculated correctly:

```bash
# Get RSI value
RSI=$(docker exec stock-scanner-redis redis-cli GET "ind:AAPL" | jq -r '.values.rsi_14')
echo "RSI: $RSI"

# RSI should be between 0-100
if (( $(echo "$RSI >= 0 && $RSI <= 100" | bc -l) )); then
    echo "✅ RSI is in valid range"
else
    echo "❌ RSI is out of range"
fi
```

## Quick Reference Commands

```bash
# Health check
curl http://localhost:8085/health | jq .

# Check finalized bars
docker exec stock-scanner-redis redis-cli XLEN bars.finalized

# List indicator keys
docker exec stock-scanner-redis redis-cli KEYS "ind:*"

# Get indicator for symbol
docker exec stock-scanner-redis redis-cli GET "ind:AAPL" | jq .

# Check consumer group
docker exec stock-scanner-redis redis-cli XINFO GROUPS bars.finalized

# View logs
docker logs -f stock-scanner-indicator

# Check metrics
curl http://localhost:8085/metrics | grep indicator

# Monitor pub/sub
docker exec -it stock-scanner-redis redis-cli PSUBSCRIBE "indicators.updated"
```

## Success Criteria

Phase 2 is successfully validated when:

1. ✅ Indicator service is running and healthy
2. ✅ Service is consuming from `bars.finalized` stream
3. ✅ Indicators are being computed for symbols
4. ✅ Indicators are published to Redis (`ind:{symbol}` keys)
5. ✅ Pub/sub notifications are sent (`indicators.updated` channel)
6. ✅ Multiple indicator types are available (RSI, EMA, SMA, VWAP, etc.)
7. ✅ No errors in service logs
8. ✅ Metrics are being collected

## Next Steps

Once Phase 2 is validated:

- **Phase 3:** Rule Engine & Scanner Worker (will consume indicators and evaluate rules)
- Monitor performance: Use `scripts/monitor_performance.sh`
- View dashboards: Grafana at http://localhost:3000

## Additional Resources

- **RedisInsight Guide:** See `docs/REDISINSIGHT_GUIDE.md`
- **Performance Monitoring:** See `docs/PERFORMANCE_MONITORING.md`
- **Testing Guide:** See `docs/TESTING_AND_DEPLOYMENT.md`

