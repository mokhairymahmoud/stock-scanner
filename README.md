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

### 3. Start Services

You have two options:

#### Option A: Docker Compose (Recommended)

```bash
# Start infrastructure only (Redis, TimescaleDB, Prometheus, Grafana)
make docker-up

# Or start everything including Go services
make docker-up-all

# View logs
make docker-logs
# Or for a specific service:
make docker-logs-service SERVICE=ingest
```

This starts:
- Redis (port 6379)
- TimescaleDB (port 5432)
- Prometheus (port 9090)
- Grafana (port 3000)
- All Go microservices (ports 8080-8091)

#### Option B: Manual (For Development/Debugging)

```bash
# Start infrastructure
make docker-up

# Build all services
make build

# Run individual services (in separate terminals)
make run-ingest
make run-bars
make run-indicator
make run-scanner
make run-ws-gateway
make run-api
```

### 4. Run Migrations

```bash
# Run database migrations
make migrate-up
# Or manually:
psql -h localhost -U postgres -d stock_scanner -f scripts/migrations/001_create_bars_table.sql
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
```

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

## License

[Add your license here]

