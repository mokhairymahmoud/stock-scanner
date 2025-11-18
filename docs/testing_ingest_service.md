# Testing the Ingest Service

This guide explains how to test the ingest service with different providers.

## Prerequisites

1. **Redis** must be running (via Docker Compose or standalone)
2. **Environment variables** configured (see `.env` file)

## Testing with Mock Provider

The mock provider is the easiest way to test the ingest service without external dependencies.

### Step 1: Set up Environment Variables

Create a `.env` file in the project root:

```bash
# Environment
ENVIRONMENT=development
LOG_LEVEL=debug

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Database (not required for ingest service, but needed for config validation)
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=stock_scanner

# Market Data Configuration
MARKET_DATA_PROVIDER=mock
MARKET_DATA_API_KEY=test-key
MARKET_DATA_SYMBOLS=AAPL,MSFT,GOOGL,TSLA

# Ingest Service Configuration
INGEST_PORT=8080
INGEST_HEALTH_PORT=8081
INGEST_STREAM_NAME=ticks
INGEST_BATCH_SIZE=100
INGEST_BATCH_TIMEOUT=100ms
```

### Step 2: Start Redis

If using Docker Compose:

```bash
make docker-up
```

Or start Redis manually:

```bash
docker run -d -p 6379:6379 redis:7-alpine
```

### Step 3: Run the Ingest Service

```bash
# Build the service
go build -o ingest ./cmd/ingest

# Run the service
./ingest
```

Or use the Makefile:

```bash
make run-ingest
```

### Step 4: Verify the Service is Running

#### Check Health Endpoint

```bash
curl http://localhost:8081/health
```

Expected response:
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

#### Check Readiness

```bash
curl http://localhost:8081/ready
```

Should return: `ready`

#### Check Liveness

```bash
curl http://localhost:8081/live
```

Should return: `alive`

#### Check Metrics

```bash
curl http://localhost:8081/metrics
```

You should see Prometheus metrics including:
- `stream_publish_total` - Total ticks published
- `stream_publish_latency_seconds` - Publish latency
- `stream_publish_batch_size` - Batch sizes

### Step 5: Verify Data Flow

#### Check Redis Stream

Connect to Redis and check the stream:

```bash
# Connect to Redis CLI
redis-cli

# Check stream length
XLEN ticks

# Read messages from stream
XREAD COUNT 10 STREAMS ticks 0

# Read latest messages
XREAD COUNT 10 STREAMS ticks $
```

You should see messages like:
```
1) 1) "ticks"
   2) 1) 1) "1699999999999-0"
         2) 1) "tick"
            2) "{\"symbol\":\"AAPL\",\"price\":150.5,\"size\":100,\"timestamp\":\"2023-11-17T00:00:00Z\",\"type\":\"trade\"}"
```

### Step 6: Monitor Logs

The service logs will show:
- Connection status
- Subscription confirmation
- Tick processing (every 1000 ticks)
- Batch publishing

Example logs:
```
INFO    Starting ingest service    {"port": "8080", "health_port": "8081", "stream": "ticks", "provider": "mock"}
INFO    Connected to Redis    {"host": "localhost", "port": 6379}
INFO    Subscribed to symbols    {"count": 4, "symbols": "[AAPL MSFT GOOGL TSLA]"}
DEBUG   Published batch to stream    {"stream": "ticks", "count": 100, "latency": "2.5ms"}
```

## Testing with Real Providers

### Alpaca Provider (when implemented)

1. Get API credentials from Alpaca
2. Update `.env`:

```bash
MARKET_DATA_PROVIDER=alpaca
MARKET_DATA_API_KEY=your-api-key
MARKET_DATA_API_SECRET=your-api-secret
MARKET_DATA_WS_URL=wss://stream.data.alpaca.markets/v2/iex
MARKET_DATA_SYMBOLS=AAPL,MSFT,GOOGL
```

3. Run the service as above

### Polygon.io Provider (when implemented)

1. Get API key from Polygon.io
2. Update `.env`:

```bash
MARKET_DATA_PROVIDER=polygon
MARKET_DATA_API_KEY=your-api-key
MARKET_DATA_WS_URL=wss://socket.polygon.io/stocks
MARKET_DATA_SYMBOLS=AAPL,MSFT,GOOGL
```

3. Run the service as above

## Integration Testing

### Run Integration Tests

```bash
# Run all integration tests
go test ./tests/... -v -run TestIngestService

# Run specific test
go test ./tests/... -v -run TestIngestService_Integration
```

### Manual Integration Test Script

Create a test script to verify end-to-end flow:

```bash
#!/bin/bash
# test_ingest.sh

echo "Starting Redis..."
docker-compose up -d redis

echo "Waiting for Redis..."
sleep 2

echo "Building ingest service..."
go build -o ingest ./cmd/ingest

echo "Starting ingest service in background..."
./ingest &
INGEST_PID=$!

echo "Waiting for service to start..."
sleep 3

echo "Checking health..."
curl -s http://localhost:8081/health | jq .

echo "Waiting for ticks to be published..."
sleep 5

echo "Checking Redis stream..."
redis-cli XLEN ticks

echo "Reading sample messages..."
redis-cli XREAD COUNT 5 STREAMS ticks 0

echo "Stopping ingest service..."
kill $INGEST_PID

echo "Test complete!"
```

## Troubleshooting

### Service Won't Start

1. **Check Redis connection**:
   ```bash
   redis-cli ping
   ```
   Should return: `PONG`

2. **Check configuration**:
   ```bash
   # Verify environment variables are loaded
   go run ./cmd/ingest --help  # If you add a flag to print config
   ```

3. **Check logs** for specific errors

### No Ticks Being Published

1. **Verify provider is connected**:
   ```bash
   curl http://localhost:8081/health | jq .checks.provider.connected
   ```

2. **Check symbols are subscribed**:
   - Look for log: "Subscribed to symbols"
   - Verify `MARKET_DATA_SYMBOLS` is set correctly

3. **Check batch size**:
   ```bash
   curl http://localhost:8081/health | jq .checks.publisher.batch_size
   ```
   - If batch_size > 0, ticks are queued but not flushed yet
   - Wait for batch timeout or manually flush

4. **Check Redis connection**:
   ```bash
   redis-cli ping
   ```

### Health Check Returns Degraded

If `status: "degraded"`, the provider is not connected. Check:
- Provider configuration (API keys, URLs)
- Network connectivity
- Provider service status

## Performance Testing

### Load Test with Mock Provider

The mock provider can generate high volumes of ticks. To test:

1. Increase symbol count:
   ```bash
   MARKET_DATA_SYMBOLS=AAPL,MSFT,GOOGL,TSLA,AMZN,NVDA,META,NFLX,AMD,INTC
   ```

2. Monitor metrics:
   ```bash
   watch -n 1 'curl -s http://localhost:8081/metrics | grep stream_publish'
   ```

3. Check Redis stream length:
   ```bash
   watch -n 1 'redis-cli XLEN ticks'
   ```

### Benchmark Publishing

```bash
# Run benchmark test
go test ./tests/... -bench=BenchmarkStreamPublisher -benchmem
```

## Next Steps

Once the ingest service is working:
1. Verify ticks are being published to Redis streams
2. Move to Phase 1.2: Bar Aggregator Service
3. The bar aggregator will consume from the same stream

