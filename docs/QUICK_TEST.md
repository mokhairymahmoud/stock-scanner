# Quick Test Guide - Ingest Service

## Quick Start (Mock Provider)

### 1. Create `.env` file

```bash
cp config/env.example .env
```

Edit `.env` and set:
```bash
MARKET_DATA_PROVIDER=mock
MARKET_DATA_API_KEY=test-key
MARKET_DATA_SYMBOLS=AAPL,MSFT,GOOGL
LOG_LEVEL=debug
```

### 2. Start Redis

```bash
# Option 1: Docker Compose (recommended)
make docker-up

# Option 2: Docker directly
docker run -d -p 6379:6379 redis:7-alpine
```

### 3. Run the Service

```bash
# Option 1: Using Makefile
make run-ingest

# Option 2: Build and run manually
go build -o ingest ./cmd/ingest
./ingest
```

### 4. Test the Service

#### Check Health
```bash
curl http://localhost:8081/health | jq .
```

#### Check if Ticks are Being Published
```bash
# Wait 10-15 seconds for ticks to accumulate and flush

# Check stream length
redis-cli XLEN ticks

# Read messages
redis-cli XREAD COUNT 10 STREAMS ticks 0
```

#### Check Metrics
```bash
curl http://localhost:8081/metrics | grep stream_publish
```

### 5. Automated Test Script

```bash
./scripts/test_ingest.sh
```

This script will:
- Check Redis connection
- Build the service
- Start the service
- Test all endpoints
- Verify data is being published
- Show sample messages

## What to Expect

### Logs
You should see:
```
INFO    Starting ingest service
INFO    Connected to Redis
INFO    Subscribed to symbols    {"count": 3, "symbols": "[AAPL MSFT GOOGL]"}
DEBUG   Published batch to stream    {"stream": "ticks", "count": 100, "latency": "2.5ms"}
```

### Health Check Response
```json
{
  "status": "healthy",
  "timestamp": "2023-11-17T00:00:00Z",
  "checks": {
    "provider": {
      "status": "ok",
      "connected": true,
      "provider": "mock"
    },
    "publisher": {
      "status": "ok",
      "batch_size": 0
    }
  }
}
```

### Redis Stream Messages
Each message contains a tick:
```json
{
  "symbol": "AAPL",
  "price": 150.5,
  "size": 100,
  "timestamp": "2023-11-17T00:00:00Z",
  "type": "trade"
}
```

## Troubleshooting

### Service won't start
- Check Redis is running: `redis-cli ping`
- Check `.env` file exists and has required variables
- Check logs for specific errors

### No ticks in stream
- Wait longer (batch flushes every 100ms or when batch size reaches 100)
- Check health endpoint: `curl http://localhost:8081/health`
- Verify provider is connected: `jq .checks.provider.connected <(curl -s http://localhost:8081/health)`

### Provider not connected
- For mock provider: Should always connect
- Check `MARKET_DATA_PROVIDER` is set to `mock`
- Check logs for connection errors

## Next Steps

Once testing is successful:
1. Verify ticks are being published to Redis
2. Move to Phase 1.2: Bar Aggregator Service
3. The bar aggregator will consume from the `ticks` stream

