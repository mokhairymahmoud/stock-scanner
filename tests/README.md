# Test Suite Documentation

This directory contains comprehensive tests for the Stock Scanner system, organized by test type and purpose.

## Test Organization

Tests are organized into subdirectories by type:

```
tests/
├── component_e2e/      # Component-level E2E tests
├── api_e2e/            # API-based E2E tests (recommended)
├── pipeline_e2e/       # Internal pipeline E2E tests
└── performance/        # Performance and stress tests
```

### 📦 Unit Tests
Fast, isolated tests for individual components. These use mocks and don't require external services.

**Location:** `internal/*/` (co-located with source code)
- **`internal/data/normalizer_test.go`** - Tests data normalization from different provider formats
- **`internal/data/provider_test.go`** - Tests mock provider and provider factory
- **`internal/pubsub/stream_publisher_test.go`** - Tests Redis stream publishing with batching and partitioning
- **`internal/wsgateway/websocket_test.go`** - Tests WebSocket client connection and reconnection logic

**Run unit tests:**
```bash
go test ./internal/... -v -run "TestNormalizer|TestProvider|TestStreamPublisher|TestWebSocket"
```

### 🔗 Integration Tests
Tests that verify components work together, using mocks for external dependencies.

**Location:** `internal/*/` (co-located with source code)
- **`internal/bars/aggregator_test.go`** - Tests bar aggregation service integration
- **`internal/bars/publisher_test.go`** - Tests bar publishing integration

**Run integration tests:**
```bash
go test ./internal/... -v -run "TestBars|TestIngest"
```

### 🧪 Component E2E Tests
End-to-end tests for specific components using mocks.

**Location:** `tests/component_e2e/`
- **`scanner_e2e_test.go`** - Tests scanner worker component end-to-end (state management, rule evaluation, alerts)

**Run component E2E tests:**
```bash
go test ./tests/component_e2e -v -run TestScannerE2E
```

### 🌐 API-Based E2E Tests (Recommended)
**Real user workflow tests** - Deploys all services via Docker and tests via HTTP/WebSocket APIs.

**Location:** `tests/api_e2e/`
- **`e2e_api_test.go`** - Complete API-based E2E tests
- **`e2e_test_helper.go`** - Helper utilities for E2E tests

**Tests included:**
- Service health checks
- Rule management (CRUD operations)
- Symbol listing and search
- WebSocket connections for real-time alerts
- Complete user workflows

**Run API E2E tests:**
```bash
# Auto-starts Docker services if needed
go test ./tests/api_e2e -v -run TestE2E -timeout 10m

# Skip Docker setup (if services already running)
SKIP_DOCKER_SETUP=true go test ./tests/api_e2e -v -run TestE2E
```

**Prerequisites:**
- Docker and Docker Compose installed
- Ports available: 8090 (API), 8088 (WebSocket), 8081-8093 (services)

### 🔧 Internal Pipeline E2E Tests
Low-level tests that verify the internal data pipeline (Redis streams, etc.).

**Location:** `tests/pipeline_e2e/`
- **`full_pipeline_e2e_test.go`** - Tests complete pipeline from ingestion to alerts via internal components

**Run pipeline E2E tests:**
```bash
# Requires Redis and TimescaleDB running
go test ./tests/pipeline_e2e -v -run TestFullPipelineE2E
```

### ⚡ Performance Tests
Tests for system performance, scalability, and resource usage.

**Location:** `tests/performance/`
- **`load_test.go`** - Load tests with 2000, 5000, 10000 symbols
- **`stress_test.go`** - Stress tests (tick bursts, high rule counts, connection pools)
- **`chaos_test.go`** - Chaos engineering tests (failures, partitions, recovery)
- **`stability_test.go`** - Long-running stability and memory leak tests

**Run performance tests:**
```bash
# Load tests
go test ./tests/performance -v -run TestLoad

# Stress tests
go test ./tests/performance -v -run TestStress

# Chaos tests
go test ./tests/performance -v -run TestChaos

# Stability tests (long-running)
go test ./tests/performance -v -run TestStability -timeout 30m
```

## Quick Start

### Run All Tests

```bash
# Unit and integration tests (fast, no external dependencies)
go test ./tests -v -short

# All tests including E2E (requires Docker)
go test ./tests -v -timeout 10m
```

### Run Tests by Category

```bash
# Unit tests only (in internal packages)
go test ./internal/... -v -run "TestNormalizer|TestProvider|TestStreamPublisher|TestWebSocket"

# Integration tests only (in internal packages)
go test ./internal/... -v -run "TestBars|TestIngest"

# Component E2E tests
go test ./tests/component_e2e -v -run TestScannerE2E

# E2E tests (API-based, recommended)
go test ./tests/api_e2e -v -run TestE2E -timeout 10m

# Pipeline E2E tests
go test ./tests/pipeline_e2e -v -run TestFullPipelineE2E

# Performance tests
go test ./tests/performance -v -run "TestLoad|TestStress|TestChaos|TestStability"
```

## Test Prerequisites

### Unit & Integration Tests
- ✅ No prerequisites - use mocks
- ✅ Fast execution
- ✅ Can run in CI/CD without Docker

### E2E Tests (API-Based)
- ✅ Docker and Docker Compose
- ✅ Ports: 8090, 8088, 8081-8093, 6379, 5432
- ✅ Services can be auto-started by tests

### Pipeline E2E Tests
- ✅ Redis running on localhost:6379
- ✅ TimescaleDB running on localhost:5432
- ✅ Can use Docker Compose: `docker-compose -f config/docker-compose.yaml up -d`

### Performance Tests
- ✅ Some require Redis (load/stress tests)
- ✅ Others are pure unit tests (no dependencies)

## Test File Reference

### Tests in `tests/` Directory

| File | Location | Type | Purpose | Dependencies |
|------|----------|------|---------|--------------|
| `scanner_e2e_test.go` | `component_e2e/` | Component E2E | Scanner component | Mocks |
| `e2e_api_test.go` | `api_e2e/` | API E2E | **Real user workflows** | Docker services |
| `e2e_test_helper.go` | `api_e2e/` | Helper | E2E utilities | Docker |
| `full_pipeline_e2e_test.go` | `pipeline_e2e/` | Pipeline E2E | Internal pipeline | Redis, TimescaleDB |
| `load_test.go` | `performance/` | Performance | Load testing | Optional Redis |
| `stress_test.go` | `performance/` | Performance | Stress testing | Optional Redis |
| `chaos_test.go` | `performance/` | Performance | Chaos engineering | Optional Redis |
| `stability_test.go` | `performance/` | Performance | Stability testing | None |

### Tests in `internal/` Directory (Unit & Integration)

| File | Location | Type | Purpose | Dependencies |
|------|----------|------|---------|--------------|
| `normalizer_test.go` | `internal/data/` | Unit | Data normalization | None |
| `provider_test.go` | `internal/data/` | Unit | Provider abstraction | None |
| `stream_publisher_test.go` | `internal/pubsub/` | Unit | Stream publishing | Mock Redis |
| `websocket_test.go` | `internal/wsgateway/` | Unit | WebSocket client | None |
| `aggregator_test.go` | `internal/bars/` | Integration | Bar aggregation | Mock Redis |
| `publisher_test.go` | `internal/bars/` | Integration | Bar publishing | Mock Redis |

## Recommended Test Workflow

### For Development
1. **Run unit tests frequently** (fast feedback):
   ```bash
   go test ./internal/... -v -short
   ```

2. **Run integration tests** before committing:
   ```bash
   go test ./internal/... -v -run "TestBars|TestIngest"
   ```

3. **Run API E2E tests** before pushing:
   ```bash
   go test ./tests/api_e2e -v -run TestE2E -timeout 10m
   ```

### For CI/CD
1. **Always run unit and integration tests**:
   ```bash
   go test ./internal/... -v -short
   ```

2. **Run API E2E tests** (if Docker available):
   ```bash
   docker-compose -f config/docker-compose.yaml up -d
   go test ./tests/api_e2e -v -run TestE2E -timeout 10m
   docker-compose -f config/docker-compose.yaml down
   ```

3. **Run performance tests** (optional, on schedule):
   ```bash
   go test ./tests/performance -v -run "TestLoad|TestStress" -timeout 30m
   ```

## Test Coverage Goals

- **Unit Tests**: > 80% coverage (already achieved)
- **Integration Tests**: Cover all major integration points
- **E2E Tests**: Cover complete user workflows
- **Performance Tests**: Validate performance at scale

## Troubleshooting

### Tests Failing to Connect to Services

```bash
# Check if services are running
docker ps

# Start services manually
docker-compose -f config/docker-compose.yaml up -d

# Check service health
curl http://localhost:8090/health
curl http://localhost:8088/health
```

### Port Conflicts

If ports are in use:
1. Stop existing services: `docker-compose -f config/docker-compose.yaml down`
2. Or modify port mappings in `docker-compose.yaml`
3. Update test constants in test files

### Tests Timing Out

- Increase timeout: `go test ./tests -timeout 30m`
- Check service logs: `docker-compose -f config/docker-compose.yaml logs`
- Verify all services are healthy before running tests

### WebSocket Connection Fails

- Ensure WebSocket gateway service is running
- Check firewall settings
- Verify port 8088 is accessible

## Test Maintenance

### Adding New Tests

1. **Unit tests** → Add to appropriate `*_test.go` file in `internal/*/` directories
2. **Integration tests** → Add to appropriate `*_test.go` file in `internal/*/` directories
3. **Component E2E tests** → Add to `tests/component_e2e/`
4. **API E2E tests** → Add to `tests/api_e2e/e2e_api_test.go` or create new file
5. **Pipeline E2E tests** → Add to `tests/pipeline_e2e/`
6. **Performance tests** → Add to appropriate file in `tests/performance/`

### Test Naming Conventions

- Unit tests: `TestComponent_Method`
- Integration tests: `TestService_Feature`
- E2E tests: `TestE2E_Workflow`
- Performance tests: `TestLoad_Scenario`, `TestStress_Scenario`, etc.

## Related Documentation

- **E2E Testing Guide**: See `README_E2E.md` for detailed E2E testing instructions
- **Implementation Plan**: See `../implementation_plan.md` for test strategy
- **Architecture**: See `../architecture.md` for system design

## Test Statistics

### Tests in `tests/` Directory
- **Component E2E Tests**: 1 file (`component_e2e/`)
- **API E2E Tests**: 2 files (`api_e2e/`)
- **Pipeline E2E Tests**: 1 file (`pipeline_e2e/`)
- **Performance Tests**: 4 files (`performance/`)

### Tests in `internal/` Directory
- **Unit Tests**: Multiple files (co-located with source)
- **Integration Tests**: Multiple files (co-located with source)

### Total
- **Total Test Functions**: 78+

---

**Last Updated**: 2024-11-20
**Maintained By**: Development Team

