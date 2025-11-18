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

## Testing the Ingest Service

The ingest service can be tested with a mock provider (no external API keys needed).

### Quick Test

```bash
# 1. Set up environment for mock provider
cp config/env.example .env
# Edit .env: MARKET_DATA_PROVIDER=mock, MARKET_DATA_API_KEY=test-key

# 2. Start Redis
make docker-up

# 3. Run automated test script
./scripts/test_ingest.sh

# Or manually:
# 3. Build and run
go build -o ingest ./cmd/ingest
./ingest

# 4. In another terminal, check health
curl http://localhost:8081/health | jq .

# 5. Check if ticks are being published (wait 10-15 seconds)
redis-cli XLEN ticks
redis-cli XREAD COUNT 10 STREAMS ticks 0
```

For detailed testing instructions, see:
- [Quick Test Guide](docs/QUICK_TEST.md)
- [Full Testing Guide](docs/testing_ingest_service.md)

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

