# Real-Time Trading Scanner

A high-performance, real-time stock market scanner that processes market data, computes technical indicators, and evaluates trading rules to generate alerts with sub-second latency.

## Architecture

The system consists of multiple microservices:
- **Ingest Service**: Connects to market data providers and publishes ticks
- **Bar Aggregator**: Aggregates ticks into 1-minute bars
- **Indicator Engine**: Computes technical indicators (RSI, EMA, VWAP, etc.)
- **Scanner Worker**: Evaluates trading rules and generates alerts
- **WebSocket Gateway**: Delivers alerts to clients in real-time
- **REST API**: Manages rules, alerts, and user preferences

## Prerequisites

- Go 1.21 or later
- Docker and Docker Compose
- Make (optional, for convenience commands)

## Quick Start

### 1. Clone and Setup

```bash
git clone <repository-url>
cd stock-scanner
```

### 2. Configure Environment

```bash
cp config/env.example .env
# Edit .env with your configuration (especially MARKET_DATA_API_KEY and MARKET_DATA_API_SECRET)
```

### 3. Deploy and Test Services

#### Quick Deployment (Recommended)

```bash
# Automated deployment (starts infrastructure, runs migrations, builds and starts services)
make docker-deploy

# Or use the deployment script directly
./scripts/deploy.sh
```

This will:
- Start infrastructure (Redis, TimescaleDB, Prometheus, Grafana)
- Run database migrations
- Build Docker images for Ingest and Bars services
- Start both services
- Wait for health checks

#### Test Services

```bash
# Automated testing
make docker-test

# Or use the test script directly
./scripts/test_services.sh
```

#### Manual Start (Alternative)

```bash
# Start infrastructure only
make docker-up

# Wait for infrastructure to be ready
sleep 15

# Run migrations
make migrate-up

# Start all services
make docker-up-all

# View logs
make docker-logs
# Or for a specific service:
make docker-logs-service SERVICE=ingest
```

#### Manual Testing (For Development/Debugging)

```bash
# Start infrastructure
make docker-up

# Build all services
make build

# Run individual services (in separate terminals)
make run-ingest
make run-bars
```

### 4. Verify Services

```bash
# Check Ingest Service
curl http://localhost:8081/health | jq .

# Check Bars Service
curl http://localhost:8083/health | jq .

# Check metrics
curl http://localhost:8081/metrics | grep stream_publish
curl http://localhost:8083/metrics | grep timescale_write
```

### 5. Verify Data Flow

```bash
# Check ticks stream (wait 10-15 seconds first)
docker exec -it stock-scanner-redis redis-cli XLEN ticks

# Check finalized bars (wait 1-2 minutes for minute boundary)
docker exec -it stock-scanner-redis redis-cli XLEN bars.finalized

# Check database
docker exec -it stock-scanner-timescaledb psql -U postgres -d stock_scanner \
  -c "SELECT COUNT(*) FROM bars_1m;"
```

## Development

### Project Structure

```
/cmd          - Service entry points
/internal     - Internal packages (not exported)
/pkg          - Public packages
/scripts      - Utility scripts and migrations
/config       - Configuration files
/docs         - Documentation
/tests        - Integration and E2E tests
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/models

# Run integration tests
go test ./tests/... -v
```

## Comprehensive Testing Guide

This section provides step-by-step instructions to test all functionalities of the system.

### Prerequisites for Testing

```bash
# Ensure all required tools are installed
docker --version
docker-compose --version
curl --version
jq --version  # Optional but recommended
redis-cli --version  # Optional but recommended

# Ensure ports are available: 6379, 5432, 8080-8091, 9090, 3000, 8001
```

### Initial Setup

```bash
# 1. Clone and navigate to project
cd stock-scanner

# 2. Create environment file
cp config/env.example .env

# 3. Edit .env for testing (use mock provider for testing)
# MARKET_DATA_PROVIDER=mock
# MARKET_DATA_SYMBOLS=AAPL,MSFT,GOOGL,AMZN,TSLA
# MARKET_DATA_API_KEY=test-key
# MARKET_DATA_API_SECRET=test-secret

# 4. Start infrastructure
make docker-up

# 5. Wait for infrastructure to be ready (15-20 seconds)
sleep 20

# 6. Run database migrations
make migrate-up
```

### Testing All Services

#### 1. Test Ingest Service

```bash
# Build and start ingest service
make build
make run-ingest

# Or use Docker
make docker-build
make docker-up-all

# In another terminal, verify service is running
curl http://localhost:8081/health | jq .
# Expected: {"status":"healthy"}

# Check metrics
curl http://localhost:8081/metrics | grep stream_publish

# Verify ticks are being published (wait 10-15 seconds)
docker exec -it stock-scanner-redis redis-cli XLEN ticks
# Should show increasing count

# Read sample ticks
docker exec -it stock-scanner-redis redis-cli XREAD COUNT 5 STREAMS ticks 0
# Should show tick data with symbol, price, size, timestamp
```

#### 2. Test Bars Service

```bash
# Start bars service (if not already running)
make run-bars

# Verify service health
curl http://localhost:8083/health | jq .
# Expected: {"status":"healthy"}

# Check metrics
curl http://localhost:8083/metrics | grep timescale_write

# Wait for minute boundary (or wait 1-2 minutes)
# Then check finalized bars stream
docker exec -it stock-scanner-redis redis-cli XLEN bars.finalized
# Should show finalized bars

# Check live bars in Redis
docker exec -it stock-scanner-redis redis-cli GET "livebar:AAPL"
# Should show JSON with open, high, low, close, volume

# Verify bars in database (wait for batch write)
docker exec -it stock-scanner-timescaledb psql -U postgres -d stock_scanner \
  -c "SELECT symbol, timestamp, open, high, low, close, volume FROM bars_1m ORDER BY timestamp DESC LIMIT 5;"
```

#### 3. Test Indicator Service

```bash
# Start indicator service
make run-indicator

# Verify service health
curl http://localhost:8085/health | jq .
# Expected: {"status":"healthy"}

# Check metrics
curl http://localhost:8085/metrics | grep indicator_compute

# Wait for indicators to be computed (after bars are finalized)
# Check indicator values in Redis
docker exec -it stock-scanner-redis redis-cli GET "ind:AAPL"
# Should show JSON with indicator values (rsi_14, ema_20, etc.)

# Check indicator stream
docker exec -it stock-scanner-redis redis-cli XLEN indicators
# Should show published indicators
```

#### 4. Test Scanner Service

```bash
# Start scanner service
make run-scanner

# Verify service health
curl http://localhost:8087/health | jq .
# Expected: {"status":"healthy"}

# Check metrics
curl http://localhost:8087/metrics | grep scan_loop

# Create a test rule via API (see REST API testing below)
# Or manually add rule to Redis for testing:
docker exec -it stock-scanner-redis redis-cli SET "rules:test-rule" '{"id":"test-rule","name":"RSI Oversold","conditions":[{"metric":"rsi_14","operator":"<","value":30}],"cooldown":300,"enabled":true}'

# Wait for scanner to pick up rule and scan
# Check for alerts in Redis stream
docker exec -it stock-scanner-redis redis-cli XLEN alerts.raw
# Should show alerts if rules match

# Check scanner state
curl http://localhost:8087/stats | jq .
# Should show symbols being scanned, rules loaded, etc.
```

#### 5. Test Alert Service

```bash
# Start alert service
make run-alert

# Verify service health
curl http://localhost:8089/health | jq .
# Expected: {"status":"healthy"}

# Check metrics
curl http://localhost:8089/metrics | grep alert_processed

# Verify alerts are being processed
# Check filtered alerts stream (for WebSocket Gateway)
docker exec -it stock-scanner-redis redis-cli XLEN alerts.filtered
# Should show processed alerts

# Check alert history in database
docker exec -it stock-scanner-timescaledb psql -U postgres -d stock_scanner \
  -c "SELECT id, rule_id, symbol, timestamp, price, message FROM alert_history ORDER BY timestamp DESC LIMIT 5;"
```

#### 6. Test WebSocket Gateway

```bash
# Start WebSocket Gateway
make run-ws-gateway

# Verify service health
curl http://localhost:8089/health | jq .
# Expected: {"status":"healthy"}

# Test WebSocket connection (using wscat or similar tool)
# Install wscat: npm install -g wscat

# Connect to WebSocket
wscat -c ws://localhost:8088/ws

# After connecting, subscribe to symbols:
# Send: {"type":"subscribe","symbols":["AAPL","MSFT"]}

# You should receive alert messages when they occur:
# {"type":"alert","data":{"id":"...","rule_id":"...","symbol":"AAPL",...}}

# Check gateway stats
curl http://localhost:8091/stats | jq .
# Should show active connections, messages sent, etc.
```

#### 7. Test REST API Service

```bash
# Start API service
make run-api

# Verify service health
curl http://localhost:8080/health | jq .
# Expected: {"status":"healthy"}

# Check readiness
curl http://localhost:8080/ready | jq .
# Expected: {"status":"ready"}

# Check metrics
curl http://localhost:8080/metrics | grep http_request
```

**Rule Management Testing:**

```bash
# 1. List all rules
curl http://localhost:8080/api/v1/rules | jq .

# 2. Create a new rule
curl -X POST http://localhost:8080/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "RSI Oversold",
    "description": "Alert when RSI drops below 30",
    "conditions": [
      {"metric": "rsi_14", "operator": "<", "value": 30}
    ],
    "cooldown": 300,
    "enabled": true
  }' | jq .

# Save the rule ID from response (e.g., "rule-123")

# 3. Get specific rule
curl http://localhost:8080/api/v1/rules/rule-123 | jq .

# 4. Update rule
curl -X PUT http://localhost:8080/api/v1/rules/rule-123 \
  -H "Content-Type: application/json" \
  -d '{
    "name": "RSI Oversold Updated",
    "description": "Updated description",
    "conditions": [
      {"metric": "rsi_14", "operator": "<", "value": 25}
    ],
    "cooldown": 600,
    "enabled": true
  }' | jq .

# 5. Validate rule
curl -X POST http://localhost:8080/api/v1/rules/rule-123/validate | jq .
# Expected: {"valid":true}

# 6. Delete rule
curl -X DELETE http://localhost:8080/api/v1/rules/rule-123 | jq .
```

**Alert History Testing:**

```bash
# 1. List all alerts
curl http://localhost:8080/api/v1/alerts | jq .

# 2. List alerts with filters
# Filter by symbol
curl "http://localhost:8080/api/v1/alerts?symbol=AAPL" | jq .

# Filter by rule ID
curl "http://localhost:8080/api/v1/alerts?rule_id=test-rule" | jq .

# Filter by date range
curl "http://localhost:8080/api/v1/alerts?start_time=2024-01-01T00:00:00Z&end_time=2024-12-31T23:59:59Z" | jq .

# Pagination
curl "http://localhost:8080/api/v1/alerts?limit=10&offset=0" | jq .

# 3. Get specific alert
curl http://localhost:8080/api/v1/alerts/alert-123 | jq .
```

**Symbol Management Testing:**

```bash
# 1. List all symbols
curl http://localhost:8080/api/v1/symbols | jq .

# 2. Search symbols
curl "http://localhost:8080/api/v1/symbols?search=AA" | jq .
# Should return symbols matching "AA" (e.g., AAPL)

# 3. Get specific symbol
curl http://localhost:8080/api/v1/symbols/AAPL | jq .
```

**User Management Testing:**

```bash
# 1. Get user profile
curl http://localhost:8080/api/v1/user/profile | jq .

# 2. Update user profile (MVP: not persisted)
curl -X PUT http://localhost:8080/api/v1/user/profile \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test User",
    "email": "test@example.com"
  }' | jq .
```

### End-to-End Flow Testing

Test the complete flow from market data to alerts:

```bash
# 1. Start all services in order
make docker-up          # Infrastructure
make migrate-up         # Database migrations
make run-ingest        # Terminal 1
make run-bars          # Terminal 2
make run-indicator     # Terminal 3
make run-scanner       # Terminal 4
make run-alert         # Terminal 5
make run-ws-gateway    # Terminal 6
make run-api           # Terminal 7

# 2. Create a rule via API
RULE_ID=$(curl -s -X POST http://localhost:8080/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "High Volume Alert",
    "conditions": [
      {"metric": "volume", "operator": ">", "value": 1000000}
    ],
    "cooldown": 60,
    "enabled": true
  }' | jq -r '.id')

echo "Created rule: $RULE_ID"

# 3. Wait for data flow (1-2 minutes)
# - Ticks → Bars → Indicators → Scanner → Alerts

# 4. Verify data at each stage
# Check ticks
docker exec -it stock-scanner-redis redis-cli XLEN ticks

# Check bars
docker exec -it stock-scanner-redis redis-cli XLEN bars.finalized

# Check indicators
docker exec -it stock-scanner-redis redis-cli GET "ind:AAPL"

# Check alerts
curl http://localhost:8080/api/v1/alerts | jq .

# 5. Connect WebSocket and verify real-time alerts
wscat -c ws://localhost:8091/ws
# Send: {"type":"subscribe","symbols":["AAPL"]}
# Wait for alert messages
```

### Monitoring and Verification

```bash
# 1. Check all service health endpoints
for port in 8081 8083 8085 8087 8089 8091 8080; do
  echo "Checking port $port:"
  curl -s http://localhost:$port/health | jq .
done

# 2. View Prometheus metrics
open http://localhost:9090
# Or query metrics directly:
curl http://localhost:8081/metrics | grep -E "(stream_publish|tick_received)"

# 3. View Grafana dashboards (if enabled)
open http://localhost:3000

# 4. View Redis data with RedisInsight
open http://localhost:8001

# 5. Check database
docker exec -it stock-scanner-timescaledb psql -U postgres -d stock_scanner \
  -c "SELECT COUNT(*) FROM bars_1m;"
docker exec -it stock-scanner-timescaledb psql -U postgres -d stock_scanner \
  -c "SELECT COUNT(*) FROM alert_history;"
docker exec -it stock-scanner-timescaledb psql -U postgres -d stock_scanner \
  -c "SELECT COUNT(*) FROM rules;"
```

### Automated Testing

```bash
# Run all unit tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/api/... -v
go test ./internal/rules/... -v
go test ./internal/scanner/... -v

# Run integration tests
go test ./tests/... -v

# Run E2E tests (if available)
./scripts/e2e_test.sh
```

### Troubleshooting

**Service not starting:**
```bash
# Check logs
make docker-logs
# Or for specific service
docker logs stock-scanner-ingest

# Check if ports are in use
lsof -i :8081
```

**No data flowing:**
```bash
# Verify Redis is running
docker exec -it stock-scanner-redis redis-cli PING

# Verify TimescaleDB is running
docker exec -it stock-scanner-timescaledb psql -U postgres -c "SELECT 1;"

# Check service connectivity
curl http://localhost:8081/health
```

**Rules not being picked up:**
```bash
# Verify rule is in database
docker exec -it stock-scanner-timescaledb psql -U postgres -d stock_scanner \
  -c "SELECT * FROM rules WHERE enabled = true;"

# Verify rule is in Redis
docker exec -it stock-scanner-redis redis-cli GET "rules:test-rule"

# Check scanner logs
docker logs stock-scanner-scanner | grep -i rule
```

For more detailed testing instructions, see:
- [E2E Testing Guide](docs/E2E_TESTING_GUIDE.md)
- [Implementation Plan](implementation_plan.md)

### Code Quality

```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run

# Run security scan
gosec ./...
```

## Configuration

Configuration is managed through environment variables. See `config/env.example` for all available options.

Key configuration areas:
- **Database**: TimescaleDB connection settings
- **Redis**: Redis connection and stream configuration
- **Market Data**: Provider credentials and symbols
- **Services**: Ports, timeouts, and scaling parameters

## Monitoring

- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (if enabled)
- **RedisInsight**: http://localhost:8001 (Redis GUI)
- **Health Checks**: Each service exposes `/health` and `/metrics` endpoints

## Documentation

- [Architecture Overview](architecture.md)
- [MVP Specification](mvp_spec.md)
- [Implementation Plan](implementation_plan.md)
- [Scanner Worker Design](scanner_worker_design.md)
- [Testing and Deployment Guide](docs/TESTING_AND_DEPLOYMENT.md)
- [Quick Start Guide](docs/QUICK_START.md)

## License

[Add your license here]

