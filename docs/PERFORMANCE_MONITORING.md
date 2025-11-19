# Performance Monitoring Guide

## Mock Provider Tick Generation Rate

### Calculation

The mock provider generates ticks at a fixed interval:
- **Interval**: 100ms (0.1 seconds) per tick cycle
- **Per Symbol**: 1 tick per cycle per symbol
- **Formula**: `Ticks per second = Number of symbols × 10`

### Examples

| Symbols | Ticks/Second | Ticks/Minute | Ticks/Hour |
|---------|--------------|--------------|------------|
| 1       | 10           | 600          | 36,000     |
| 3       | 30           | 1,800        | 108,000    |
| 5       | 50           | 3,000        | 180,000    |
| 10      | 100          | 6,000        | 360,000    |
| 20      | 200          | 12,000       | 720,000    |
| 50      | 500          | 30,000       | 1,800,000  |

### Code Reference

From `internal/data/mock_provider.go`:
```go
ticker := time.NewTicker(100 * time.Millisecond) // Generate ticks every 100ms
// Generates 1 tick per symbol per 100ms = 10 ticks/second per symbol
```

## Measuring Ingestion Performance

### 1. Prometheus Metrics

The ingest service exposes several metrics for performance monitoring:

#### Key Metrics

**Publish Rate**:
```promql
# Total ticks published
stream_publish_total

# Publish rate (ticks per second)
rate(stream_publish_total[1m]) * 60

# Publish rate by partition
rate(stream_publish_total[1m]) by (partition)
```

**Publish Errors**:
```promql
# Total publish errors
stream_publish_errors_total

# Error rate
rate(stream_publish_errors_total[1m])
```

**Publish Latency**:
```promql
# Average publish latency
stream_publish_latency_seconds

# P95 latency
histogram_quantile(0.95, stream_publish_latency_seconds_bucket)

# P99 latency
histogram_quantile(0.99, stream_publish_latency_seconds_bucket)
```

**Batch Size**:
```promql
# Average batch size
stream_publish_batch_size
```

### 2. Query Metrics via HTTP

```bash
# Get all metrics
curl http://localhost:8081/metrics

# Get publish metrics only
curl http://localhost:8081/metrics | grep stream_publish

# Get current publish total
curl -s http://localhost:8081/metrics | grep "^stream_publish_total" | head -1
```

### 3. Real-Time Monitoring Scripts

#### Monitor Publish Rate

```bash
# Watch publish rate (updates every second)
watch -n 1 'curl -s http://localhost:8081/metrics | grep "^stream_publish_total" | head -1'
```

#### Monitor Publish Rate with Calculation

```bash
#!/bin/bash
# monitor_publish_rate.sh

PREV_TOTAL=0
PREV_TIME=$(date +%s)

while true; do
    CURRENT_TOTAL=$(curl -s http://localhost:8081/metrics | grep "^stream_publish_total" | head -1 | awk '{print $2}')
    CURRENT_TIME=$(date +%s)
    
    if [ -n "$CURRENT_TOTAL" ] && [ "$PREV_TOTAL" -gt 0 ]; then
        DELTA=$((CURRENT_TOTAL - PREV_TOTAL))
        DELTA_TIME=$((CURRENT_TIME - PREV_TIME))
        RATE=$((DELTA / DELTA_TIME))
        
        echo "$(date '+%H:%M:%S') - Total: $CURRENT_TOTAL, Rate: $RATE ticks/sec"
    fi
    
    PREV_TOTAL=$CURRENT_TOTAL
    PREV_TIME=$CURRENT_TIME
    sleep 1
done
```

### 4. Redis Stream Monitoring

#### Check Stream Length

```bash
# Current stream length
docker exec stock-scanner-redis redis-cli XLEN ticks

# Watch stream length
watch -n 1 'docker exec stock-scanner-redis redis-cli XLEN ticks'
```

#### Check Stream Growth Rate

```bash
#!/bin/bash
# monitor_stream_growth.sh

PREV_LENGTH=0
PREV_TIME=$(date +%s)

while true; do
    CURRENT_LENGTH=$(docker exec stock-scanner-redis redis-cli XLEN ticks 2>/dev/null | tail -1)
    CURRENT_TIME=$(date +%s)
    
    if [ -n "$CURRENT_LENGTH" ] && [ "$PREV_LENGTH" -gt 0 ]; then
        DELTA=$((CURRENT_LENGTH - PREV_LENGTH))
        DELTA_TIME=$((CURRENT_TIME - PREV_TIME))
        RATE=$((DELTA / DELTA_TIME))
        
        echo "$(date '+%H:%M:%S') - Length: $CURRENT_LENGTH, Growth: $RATE msgs/sec"
    fi
    
    PREV_LENGTH=$CURRENT_LENGTH
    PREV_TIME=$CURRENT_TIME
    sleep 1
done
```

#### Read Recent Messages

```bash
# Read last 10 messages
docker exec stock-scanner-redis redis-cli XREAD COUNT 10 STREAMS ticks 0

# Read messages from last 5 seconds
docker exec stock-scanner-redis redis-cli XREAD COUNT 100 STREAMS ticks $(($(date +%s) - 5) * 1000)
```

### 5. Performance Benchmarks

#### Test with Different Symbol Counts

```bash
# Test with 3 symbols (30 ticks/sec)
export MARKET_DATA_SYMBOLS=AAPL,MSFT,GOOGL
docker-compose -f config/docker-compose.yaml restart ingest

# Test with 10 symbols (100 ticks/sec)
export MARKET_DATA_SYMBOLS=AAPL,MSFT,GOOGL,TSLA,AMZN,NVDA,META,NFLX,AMD,INTC
docker-compose -f config/docker-compose.yaml restart ingest

# Test with 20 symbols (200 ticks/sec)
export MARKET_DATA_SYMBOLS=AAPL,MSFT,GOOGL,TSLA,AMZN,NVDA,META,NFLX,AMD,INTC,SPY,QQQ,IWM,TLT,GLD,BTC,ETH,COIN,HOOD,SOFI
docker-compose -f config/docker-compose.yaml restart ingest
```

### 6. Grafana Dashboard Queries

#### Publish Rate Dashboard

```promql
# Ticks per second
rate(stream_publish_total[1m]) * 60

# Errors per second
rate(stream_publish_errors_total[1m]) * 60

# Average latency (ms)
stream_publish_latency_seconds * 1000

# P95 latency (ms)
histogram_quantile(0.95, stream_publish_latency_seconds_bucket) * 1000

# Batch size
stream_publish_batch_size
```

### 7. End-to-End Latency Measurement

#### Measure Tick to Redis Latency

```bash
# Get a tick timestamp from Redis
docker exec stock-scanner-redis redis-cli XREAD COUNT 1 STREAMS ticks 0 | \
  jq -r '.[0][1][0][1][1]' | \
  jq -r '.timestamp' | \
  xargs -I {} date -d {} +%s

# Compare with current time
echo "Latency: $(($(date +%s) - $(...))) seconds"
```

### 8. Load Testing

#### Generate High Load

```bash
# Set many symbols for high tick rate
export MARKET_DATA_SYMBOLS=AAPL,MSFT,GOOGL,TSLA,AMZN,NVDA,META,NFLX,AMD,INTC,SPY,QQQ,IWM,TLT,GLD,BTC,ETH,COIN,HOOD,SOFI,ARKK,ARKQ,ARKW,ARKG,ARKF
docker-compose -f config/docker-compose.yaml restart ingest

# Monitor performance
watch -n 1 'curl -s http://localhost:8081/metrics | grep stream_publish'
```

#### Monitor Resource Usage

```bash
# CPU and memory usage
docker stats stock-scanner-ingest --no-stream

# Watch resource usage
watch -n 1 'docker stats stock-scanner-ingest --no-stream'
```

### 9. Performance Targets

Based on the architecture, target performance metrics:

| Metric | Target | Critical |
|--------|--------|----------|
| Publish Rate | > 1000 ticks/sec | > 500 ticks/sec |
| Publish Latency (P95) | < 10ms | < 50ms |
| Publish Latency (P99) | < 50ms | < 100ms |
| Error Rate | < 0.1% | < 1% |
| Batch Size | 50-100 ticks | 10-200 ticks |

### 10. Troubleshooting Performance Issues

#### Low Publish Rate

1. Check provider connection:
   ```bash
   curl http://localhost:8081/health | jq .checks.provider
   ```

2. Check Redis connection:
   ```bash
   docker exec stock-scanner-redis redis-cli ping
   ```

3. Check for errors:
   ```bash
   curl -s http://localhost:8081/metrics | grep stream_publish_errors
   ```

4. Check service logs:
   ```bash
   docker-compose -f config/docker-compose.yaml logs ingest | tail -50
   ```

#### High Latency

1. Check Redis performance:
   ```bash
   docker exec stock-scanner-redis redis-cli --latency
   ```

2. Check batch size:
   ```bash
   curl -s http://localhost:8081/metrics | grep stream_publish_batch_size
   ```

3. Check network:
   ```bash
   docker exec stock-scanner-ingest ping -c 3 redis
   ```

#### High Error Rate

1. Check Redis memory:
   ```bash
   docker exec stock-scanner-redis redis-cli INFO memory | grep used_memory_human
   ```

2. Check stream length (backlog):
   ```bash
   docker exec stock-scanner-redis redis-cli XLEN ticks
   ```

3. Check consumer lag:
   ```bash
   docker exec stock-scanner-redis redis-cli XINFO GROUPS ticks
   ```

## Quick Performance Check

Run this one-liner to get a quick performance snapshot:

```bash
echo "=== Ingestion Performance ===" && \
echo "Publish Total: $(curl -s http://localhost:8081/metrics | grep '^stream_publish_total' | head -1 | awk '{print $2}')" && \
echo "Errors: $(curl -s http://localhost:8081/metrics | grep '^stream_publish_errors_total' | head -1 | awk '{print $2}')" && \
echo "Stream Length: $(docker exec stock-scanner-redis redis-cli XLEN ticks 2>/dev/null | tail -1)" && \
echo "Provider Connected: $(curl -s http://localhost:8081/health | jq -r '.checks.provider.connected')"
```

