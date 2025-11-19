# Detailed Implementation Plan — Real-Time Trading Scanner

## Overview
This document provides a comprehensive, phase-by-phase implementation plan for the real-time trading scanner system. Each phase builds upon the previous one, with clear dependencies and deliverables.

---

## Phase 0: Project Setup & Foundation (Week 1) ✅ COMPLETE

### Goals
- ✅ Set up project structure
- ✅ Configure development environment
- ✅ Establish CI/CD basics
- ✅ Set up local infrastructure (Docker Compose)

### Tasks

#### 0.1 Project Structure Setup ✅
- [x] Initialize Go module (`go mod init`)
- [x] Create directory structure per MVP spec:
  ```
  /cmd
    /ingest
    /scanner
    /bars
    /indicator
    /ws_gateway
    /api
  /internal
    /config
    /data
    /rules
    /scanner
    /models
    /pubsub
    /storage
  /pkg
    /indicator
    /timeseries
    /logger
  /scripts
    /migrations
  /config
  /docs
  /tests
  ```
- [x] Add `.gitignore` for Go
- [x] Create `README.md` with setup instructions
- [x] Create placeholder main files for all services
- [x] Create `Makefile` with common commands

#### 0.2 Configuration Management ✅
- [x] Implement config package (`internal/config`)
  - [x] Environment variable loading
  - [x] Config structs for each service
  - [x] Validation logic
  - [x] Default values
- [x] Create `config/env.example` with all required vars
- [x] Add config validation on startup

#### 0.3 Logging & Observability Foundation ✅
- [x] Implement structured logging (`pkg/logger`)
  - [x] JSON structured logs
  - [x] Log levels (debug, info, warn, error)
  - [x] Context propagation
- [x] Set up OpenTelemetry tracing foundation
  - [x] Trace initialization helpers
  - [x] Span creation helpers (placeholders)
- [x] Set up Prometheus metrics foundation
  - [x] Metrics registry
  - [x] Common metric helpers

#### 0.4 Local Infrastructure (Docker Compose) ✅
- [x] Create `config/docker-compose.yaml`:
  - [x] Redis (with persistence)
  - [x] TimescaleDB (PostgreSQL with TimescaleDB extension)
  - [x] Prometheus
  - [x] Grafana (optional for local dev)
  - [x] RedisInsight (Redis GUI for development)
- [x] Create initialization scripts:
  - [x] TimescaleDB schema setup (`scripts/migrations/001_create_bars_table.sql`)
  - [x] Prometheus configuration (`config/prometheus.yml`)
- [x] Document local setup in README

#### 0.5 Data Models & Types ✅
- [x] Define core data structures (`internal/models`):
  - [x] `Tick` struct
  - [x] `Bar1m` struct
  - [x] `LiveBar` struct
  - [x] `SymbolState` struct
  - [x] `Indicator` struct
  - [x] `Rule` struct
  - [x] `Alert` struct
- [x] Add JSON serialization tags
- [x] Add validation methods
- [x] Create unit tests for models (all passing)

#### 0.6 Storage Interfaces ✅
- [x] Define storage interfaces (`internal/storage`):
  - [x] `BarStorage` interface (TimescaleDB)
  - [x] `AlertStorage` interface (ClickHouse/TimescaleDB)
  - [x] `RedisClient` interface (wrappers)
- [x] Create mock implementations for testing

### Phase 0 Completion Summary

**Status:** ✅ Complete

**Deliverables:**
- ✅ Complete project structure with all directories and placeholder main files
- ✅ Configuration management system with environment variable support
- ✅ Structured logging with zap and Prometheus metrics foundation
- ✅ Docker Compose setup with Redis, TimescaleDB, Prometheus, Grafana, and RedisInsight
- ✅ All core data models with validation and unit tests (all passing)
- ✅ Storage interfaces with mock implementations for testing
- ✅ Makefile for common development tasks
- ✅ Comprehensive README with setup instructions

**Verification:**
- All code compiles successfully
- All unit tests pass
- No linter errors
- Dependencies properly managed

**Ready for:** Phase 1 - Core Data Pipeline

---

## Phase 1: Core Data Pipeline (Weeks 2-3)

### Goals
- Implement market data ingestion
- Build bar aggregation service
- Establish data flow from ingestion to storage

### Dependencies
- Phase 0 complete

### Tasks

#### 1.1 Market Data Ingest Service (`cmd/ingest`)

**1.1.1 Provider Abstraction** ✅
- [x] Create provider interface (`internal/data/provider.go`)
  - [x] `Connect()` method
  - [x] `Subscribe(symbols []string)` method
  - [x] `Unsubscribe(symbols []string)` method
  - [x] `Close()` method
- [x] Implement mock provider for testing
- [x] Create provider factory/registry

**1.1.2 WebSocket Connection Management** ✅
- [x] Implement WebSocket client wrapper
- [x] Connection retry logic with exponential backoff
- [x] Heartbeat/ping-pong handling
- [x] Reconnection strategy
- [x] Connection state monitoring

**1.1.3 Data Normalization** ✅
- [x] Create tick normalizer (`internal/data/normalizer.go`)
  - [x] Normalize different provider formats to common `Tick` struct
  - [x] Handle trade vs quote messages
  - [x] Timestamp normalization (UTC)
  - [x] Price/volume normalization
- [x] Add unit tests for normalization

**1.1.4 Stream Publishing** ✅
- [x] Implement Redis Streams publisher (`internal/pubsub/redis_stream.go`)
  - [x] Partition by symbol (hash-based)
  - [x] Batch publishing for efficiency
  - [x] Error handling and retries
  - [x] Metrics for publish rate/latency
- [ ] Alternative: Kafka publisher (optional, can defer)
- [x] Add configuration for stream names/partitions

**1.1.5 Ingest Service Main** ✅
- [x] Implement main service loop
- [x] Graceful shutdown handling
- [x] Health check endpoint
- [x] Metrics endpoint
- [x] Configuration loading
- [x] Integration tests with mock provider

**1.1.6 Real Provider Implementations**
- [ ] Implement Alpaca provider (`internal/data/alpaca_provider.go`)
  - [ ] WebSocket connection to Alpaca stream API
  - [ ] Authentication (API key/secret)
  - [ ] Subscribe/unsubscribe to symbols
  - [ ] Handle Alpaca-specific message formats (trades, quotes, bars)
  - [ ] Error handling and reconnection logic
  - [ ] Rate limiting compliance
- [ ] Implement Polygon.io provider (optional) (`internal/data/polygon_provider.go`)
  - [ ] WebSocket connection to Polygon.io stream
  - [ ] Authentication (API key)
  - [ ] Subscribe/unsubscribe to symbols
  - [ ] Handle Polygon-specific message formats
  - [ ] Error handling and reconnection logic
- [ ] Update normalizer for provider-specific formats
  - [ ] Alpaca message format handling
  - [ ] Polygon.io message format handling
  - [ ] Provider-specific timestamp formats
  - [ ] Provider-specific price/volume formats
- [ ] Update provider factory to register real providers
- [ ] Integration tests with real providers (sandbox/test mode)
  - [ ] Test with Alpaca sandbox environment
  - [ ] Test with Polygon.io test environment (if implemented)
  - [ ] Verify data normalization accuracy
  - [ ] Test reconnection scenarios
- [ ] Provider fallback logic (optional, can defer to later phase)
  - [ ] Automatic failover between providers
  - [ ] Provider health monitoring
  - [ ] Degraded mode handling

#### 1.2 Bar Aggregator Service (`cmd/bars`)

**1.2.1 Bar Aggregation Logic** ✅
- [x] Implement bar builder (`internal/bars/aggregator.go`)
  - [x] Live bar state per symbol (in-memory map)
  - [x] Update logic on tick:
    - [x] Update high/low
    - [x] Update close
    - [x] Accumulate volume
    - [x] Update VWAP numerator/denominator
  - [x] Minute boundary detection
  - [x] Bar finalization logic
- [x] Thread-safe state management
- [x] Unit tests for aggregation logic

**1.2.2 Redis Stream Consumer** ✅
- [x] Implement Redis Stream consumer (`internal/pubsub/stream_consumer.go`)
  - [x] Consumer group support
  - [x] Partition assignment
  - [x] Message acknowledgment
  - [x] Error handling
  - [x] Lag monitoring
- [x] Process ticks from stream
- [x] Update live bars

**1.2.3 Live Bar Publishing** ✅
- [x] Publish live bar snapshots to Redis
  - [x] Key: `livebar:{symbol}`
  - [x] JSON serialization
  - [x] TTL (e.g., 5 minutes)
- [x] Publish finalized bars to Redis Stream
  - [x] Stream: `bars.finalized`
  - [x] Include all bar data

**1.2.4 TimescaleDB Integration** ✅
- [x] Implement TimescaleDB writer (`internal/storage/timescale.go`)
  - [x] Connection pooling
  - [x] Hypertable creation (migration script already exists from Phase 0)
  - [x] Batch insert for finalized bars
  - [x] Async write queue
  - [x] Error handling and retries
- [x] Create migration script (`scripts/migrations/001_create_bars_table.sql`) ✅ (Completed in Phase 0)
- [x] Add metrics for write latency/errors
- [x] Integrate TimescaleDB client into bar publisher
- [x] Unit tests for TimescaleDB client

**1.2.5 Bar Aggregator Service Main** ✅
- [x] Implement main service loop
- [x] Graceful shutdown
- [x] Health checks
- [x] Metrics exposure
- [x] Integration tests
- [x] Wire consumer, aggregator, publisher, and TimescaleDB together

#### 1.3 Testing & Validation ✅
- [x] End-to-end test: Ingest → Bar Aggregator → TimescaleDB
- [x] Integration tests for bars service
- [x] Test minute boundary handling
- [x] Test reconnection scenarios
- [ ] Load test with mock data (1000+ symbols) (deferred to Phase 7)

#### 1.4 Testing & Deployment Infrastructure ✅
- [x] Comprehensive testing documentation (`docs/TESTING_AND_DEPLOYMENT.md`)
- [x] Quick start guide (`docs/QUICK_START.md`)
- [x] Deployment checklist (`docs/DEPLOYMENT_CHECKLIST.md`)
- [x] Performance monitoring guide (`docs/PERFORMANCE_MONITORING.md`)
- [x] RedisInsight connection guide (`docs/REDISINSIGHT_GUIDE.md`)
- [x] Automated deployment script (`scripts/deploy.sh`)
- [x] Service testing script (`scripts/test_services.sh`)
- [x] Deployment verification script (`scripts/verify_deployment.sh`)
- [x] Performance monitoring script (`scripts/monitor_performance.sh`)
- [x] Updated Makefile with deployment commands (`docker-deploy`, `docker-test`, `docker-verify`)
- [x] Database migration via Docker (no local psql required)

### Phase 1 Completion Summary

**Status:** ✅ Complete (Core Data Pipeline)

**Deliverables:**
- ✅ Market Data Ingest Service with provider abstraction, WebSocket management, and Redis Stream publishing
- ✅ Bar Aggregator Service with tick aggregation, Redis Stream consumption, live bar publishing, and TimescaleDB integration
- ✅ Complete end-to-end data flow: Provider → Ingest → Redis Stream → Bars → TimescaleDB
- ✅ Comprehensive testing and deployment infrastructure
- ✅ Performance monitoring tools and documentation
- ✅ RedisInsight integration for development and debugging

**Key Features:**
- Mock provider for testing (generates 10 ticks/second per symbol)
- Real-time bar aggregation with minute boundary detection
- Async batch writes to TimescaleDB with retry logic
- Health checks and metrics for all services
- Automated deployment and testing scripts

**Verification:**
- All services compile and run successfully
- Integration tests pass
- End-to-end data flow verified
- Services deployable via Docker Compose
- Performance metrics available via Prometheus

**Next Steps:**
- Phase 2: Indicator Engine (compute technical indicators from finalized bars)
- Real provider implementations (Alpaca, Polygon.io, Databento.com)

---

## Phase 2: Indicator Engine (Week 4) ✅ COMPLETE

### Goals
- ✅ Compute technical indicators from finalized bars
- ✅ Publish indicators to Redis
- ✅ Support multiple indicator types

### Dependencies
- ✅ Phase 1 complete (bar aggregator publishing finalized bars)

### Tasks

#### 2.1 Indicator Package (`pkg/indicator`)

**2.1.1 Core Indicator Interface** ✅
- [x] Define `Indicator` interface
- [x] Define `Calculator` interface for each indicator type
- [x] Create indicator registry

**2.1.2 Implement Indicators** ✅
- [x] RSI (Relative Strength Index)
  - [x] Window-based calculation
  - [x] Incremental updates
- [x] EMA (Exponential Moving Average)
  - [x] Multiple periods (EMA20, EMA50, EMA200)
  - [x] Incremental calculation
- [x] VWAP (Volume Weighted Average Price)
  - [x] Window-based (5m, 15m, 1h)
  - [x] Incremental updates
- [x] SMA (Simple Moving Average)
  - [x] Multiple periods (SMA20, SMA50, SMA200)
- [x] Volume indicators:
  - [x] Average volume (5m, 15m, 1h windows)
  - [x] Relative volume calculation
- [x] Price change indicators:
  - [x] Price change % (1m, 5m, 15m)
- [x] Unit tests for each indicator (82.6% coverage)

**2.1.3 Indicator State Management** ✅
- [x] Maintain rolling windows per symbol
- [x] Efficient data structures (ring buffers)
- [x] Thread-safe updates

#### 2.2 Indicator Engine Service (`cmd/indicator`)

**2.2.1 Bar Consumer** ✅
- [x] Subscribe to `bars.finalized` stream
- [x] Process finalized bars
- [x] Update indicator windows

**2.2.2 Indicator Computation** ✅
- [x] Compute indicators after bar finalization
- [x] Batch computation for efficiency
- [x] Handle missing data gracefully

**2.2.3 Indicator Publishing** ✅
- [x] Publish indicators to Redis
  - [x] Key: `ind:{symbol}`
  - [x] JSON structure with all indicators
  - [x] TTL (10 minutes)
- [x] Publish to Redis pub/sub channel: `indicators.updated`
- [x] Metrics for computation latency

**2.2.4 Indicator Engine Main** ✅
- [x] Service initialization
- [x] Graceful shutdown
- [x] Health checks
- [x] Metrics
- [x] Integration with Redis Streams

#### 2.3 Testing ✅
- [x] Unit tests for indicator calculations
- [x] Integration test: Bar → Indicator Engine → Redis
- [x] Verify indicator accuracy against known values
- [x] Test with missing/incomplete data

### Phase 2 Completion Summary

**Status:** ✅ Complete

**Deliverables:**
- ✅ Complete indicator package with Calculator interface, Registry, and SymbolState management
- ✅ All indicator implementations: RSI, EMA, SMA, VWAP, Volume Average, Relative Volume, Price Change
- ✅ Indicator Engine service with bar consumer, computation engine, and publisher
- ✅ Complete end-to-end data flow: Finalized Bars → Indicator Engine → Redis (keys + pub/sub)
- ✅ Comprehensive test suite with 82.6% code coverage
- ✅ Health checks and metrics endpoints

**Key Features:**
- Per-symbol calculator instances via factory pattern
- Thread-safe state management with rolling windows
- Real-time indicator computation from finalized bars
- Redis key storage (`ind:{symbol}`) with 10-minute TTL
- Redis pub/sub notifications for real-time updates
- Support for multiple indicator types and periods/windows

**Verification:**
- All code compiles successfully
- All unit tests pass (40+ test cases)
- Integration with Redis Streams working
- Services deployable and ready for Phase 3

**Next Steps:**
- Phase 3: Rule Engine & Scanner Worker (consume indicators and evaluate rules)

---

## Phase 3: Rule Engine & Scanner Worker (Weeks 5-7)

### Phase 3.1 Completion Summary

**Status:** ✅ Complete (Rule Engine Core)

**Deliverables:**
- ✅ Complete rule engine package with types, validation, parser, compiler, metric resolver, and storage
- ✅ Rule data structures (Rule, Condition, CompiledRule, RuleStore interface)
- ✅ JSON rule parser with syntax validation
- ✅ Rule compiler that converts rules into executable functions
- ✅ Metric resolver for direct and computed metric lookups
- ✅ In-memory rule store with thread-safe operations
- ✅ Comprehensive test suite with 87.4% code coverage

**Key Features:**
- Rule validation at parse time and compile time
- Support for all comparison operators (>, <, >=, <=, ==, !=)
- AND logic for multiple conditions (all must match)
- Extensible metric resolver for computed metrics
- Dual storage implementations:
  - In-memory store: Fast, simple, for testing/development
  - Redis store: Shared state across workers, persistent, for production
- Thread-safe operations (in-memory: sync.RWMutex, Redis: atomic operations)
- Full CRUD operations (AddRule, GetRule, UpdateRule, DeleteRule)
- Deep copy of rules to prevent external modifications
- Configurable via `SCANNER_RULE_STORE_TYPE` environment variable

**Verification:**
- All code compiles successfully
- All unit tests pass (50+ test cases)
- 87.4% code coverage
- Ready for Phase 3.2 (Scanner Worker Core)

**Next Steps:**
- Phase 3.2: Scanner Worker Core (symbol state, tick/indicator ingestion, scan loop)

### Phase 3.2 Completion Summary

**Status:** ✅ Complete (Scanner Worker Core)

**Deliverables:**
- ✅ Complete scanner worker core package with all components
- ✅ Symbol state management (StateManager, SymbolState)
- ✅ Tick ingestion (TickConsumer with Redis streams)
- ✅ Indicator ingestion (IndicatorConsumer with Redis pub/sub)
- ✅ Bar finalization handler (BarFinalizationHandler)
- ✅ Scan loop with rule evaluation (<800ms target)
- ✅ Cooldown management (InMemoryCooldownTracker)
- ✅ Alert emission (AlertEmitterImpl)
- ✅ Partitioning & ownership (PartitionManager)
- ✅ State rehydration (Rehydrator)
- ✅ Comprehensive test suite (64.4% coverage)
- ✅ E2E test suite (3 scenarios)
- ✅ Testing guide documentation

**Key Features:**
- Thread-safe state management with RWMutex
- Lock-free snapshot reading for scan loop
- Performance optimizations (sync.Pool, minimal allocations)
- Consistent hashing for symbol partitioning
- Automatic cooldown cleanup
- Dual alert publishing (pub/sub + streams)
- Historical state rehydration on startup
- Comprehensive error handling

**Verification:**
- All code compiles successfully
- All unit tests pass (100+ test cases)
- 64.4% code coverage
- E2E tests passing
- Performance targets met (<800ms scan time)
- Ready for Phase 3.3 (Scanner Worker Service)

**Next Steps:**
- Phase 3.3: Scanner Worker Service (main service integration)

### Phase 3.3 Completion Summary

**Status:** ✅ Complete (Scanner Worker Service)

**Deliverables:**
- ✅ Complete scanner worker service (`cmd/scanner/main.go`)
- ✅ Integration of all Phase 3.2 components
- ✅ State rehydration on startup
- ✅ All consumers started (ticks, indicators, bars)
- ✅ Scan loop running
- ✅ Health check endpoints (/health, /ready, /live)
- ✅ Metrics endpoints (/metrics, /stats)
- ✅ Graceful shutdown
- ✅ Configuration loading from environment variables
- ✅ Worker ID parsing and partition management

**Key Features:**
- Complete service lifecycle management
- Component health monitoring
- Comprehensive statistics endpoint
- Production-ready error handling
- Follows patterns from other services (bars, indicator)

**Verification:**
- All code compiles successfully
- Service builds: `bin/scanner`
- No linter errors
- Ready for deployment and testing

**Next Steps:**
- Phase 3.4: Testing (unit, integration, performance, chaos tests)

### Phase 3.4 Completion Summary

**Status:** ✅ Complete (Testing)

**Deliverables:**
- ✅ Comprehensive unit tests (64.4%+ coverage)
- ✅ Integration/E2E tests (3 scenarios)
- ✅ Performance tests (2000+ symbols, benchmarks)
- ✅ Chaos tests (6 failure scenarios)
- ✅ Performance validation (<800ms target exceeded: 2.7ms for 2000 symbols!)

**Key Results:**
- **Performance:** 2000 symbols scanned in 2.7ms (well under 800ms target)
- **Scalability:** Tested with varying rule counts (1-100 rules)
- **Resilience:** All chaos tests passing
- **Coverage:** Comprehensive test suite covering all failure modes

**Verification:**
- All unit tests passing
- All integration tests passing
- All performance tests passing
- All chaos tests passing
- Benchmarks show excellent performance

**Next Steps:**
- Phase 4: Alert Service & WebSocket Gateway

### Goals
- Implement rule definition and compilation
- Build scanner worker with <1s scan cycle
- Support multiple rule types and conditions

### Dependencies
- Phase 2 complete (indicators available)

### Tasks

#### 3.1 Rule Engine (`internal/rules`) ✅ COMPLETE

**3.1.1 Rule Data Structures** ✅
- [x] Define `Rule` struct (in `internal/models`)
- [x] Define `Condition` struct (in `internal/models`)
- [x] Define `CompiledRule` type (function)
- [x] Rule validation logic (`ValidateRule`, `ValidateCondition`)

**3.1.2 Rule Parser** ✅
- [x] JSON rule parser (`ParseRule`, `ParseRuleFromString`, `ParseRuleFromReader`)
- [x] Parse multiple rules (`ParseRules`)
- [x] Validate rule syntax (`ValidateRuleSyntax`)
- [x] Validate metric references (`ValidateMetricReference`)
- [x] Validate operators and values

**3.1.3 Rule Compilation** ✅
- [x] Implement rule compiler (`internal/rules/compiler.go`)
  - [x] Parse conditions
  - [x] Build closure/function for evaluation
  - [x] Handle metric lookups via MetricResolver
  - [x] AND logic for multiple conditions (all must match)
- [x] Support operators: `>`, `<`, `>=`, `<=`, `==`, `!=`
- [x] Compile-time validation
- [x] Unit tests for compilation (87.4% coverage)

**3.1.4 Metric Resolver** ✅
- [x] Implement metric resolver (`internal/rules/metrics.go`)
  - [x] MetricResolver interface
  - [x] DefaultMetricResolver implementation
  - [x] Support computed metrics (extensible via RegisterComputedMetric)
  - [x] Support direct indicator lookups
- [x] Condition evaluation (`EvaluateCondition`)
- [x] Unit tests

**3.1.5 Rule Storage** ✅
- [x] Rule storage interface (`RuleStore`)
- [x] In-memory rule store (`InMemoryRuleStore`) - for testing/development
- [x] Redis rule store (`RedisRuleStore`) - for production/shared state
- [x] Thread-safe operations (sync.RWMutex for in-memory, Redis atomic ops)
- [x] Full CRUD operations (AddRule, GetRule, UpdateRule, DeleteRule)
- [x] Enable/Disable operations
- [x] GetEnabledRules helper
- [x] Redis set-based rule ID tracking for efficient listing
- [x] Configurable rule store type (memory/redis) via `SCANNER_RULE_STORE_TYPE`
- [x] Unit tests for both implementations (including concurrency tests)

#### 3.2 Scanner Worker Core (`internal/scanner`) ✅ COMPLETE

**3.2.1 Symbol State Management** ✅
- [x] Implement `SymbolState` struct
- [x] Implement state map with RWMutex
- [x] State update methods (thread-safe)
- [x] State snapshot for scanning
- [x] Comprehensive unit tests

**3.2.2 Tick Ingestion** ✅
- [x] Subscribe to tick stream (partitioned)
- [x] Update live bar on tick
- [x] Update VWAP components
- [x] Handle tick ordering
- [x] Buffer management
- [x] Batch processing with configurable batch size
- [x] Comprehensive unit tests

**3.2.3 Indicator Ingestion** ✅
- [x] Subscribe to indicator updates (Redis pub/sub)
- [x] Fetch indicators from Redis keys
- [x] Update symbol state indicators
- [x] Handle indicator stream lag
- [x] Comprehensive unit tests

**3.2.4 Bar Finalization Handler** ✅
- [x] Subscribe to finalized bars (Redis streams)
- [x] Update `lastFinalBars` ring buffer
- [x] Batch processing
- [x] Comprehensive unit tests

**3.2.5 Scan Loop** ✅
- [x] Implement 1-second ticker
- [x] Symbol iteration (snapshot)
- [x] Rule evaluation per symbol
- [x] Cooldown checking
- [x] Alert emission
- [x] Performance optimization:
  - [x] Minimize allocations
  - [x] Use sync.Pool for temporary objects
  - [x] Lock-free reads where possible
- [x] Metrics for scan cycle time
- [x] Comprehensive unit tests (19 test cases)
- [x] Performance tested (<800ms target)

**3.2.6 Cooldown Management** ✅
- [x] Implement cooldown tracker (`InMemoryCooldownTracker`)
- [x] Per-rule, per-symbol cooldown
- [x] Cleanup of expired cooldowns (automatic)
- [x] Thread-safe operations
- [x] Comprehensive unit tests (13 test cases)

**3.2.7 Alert Emission** ✅
- [x] Create alert struct (already in models)
- [x] Generate alert ID (UUID)
- [x] Publish to Redis pubsub/channel
- [x] Publish to Redis Stream (optional)
- [x] Include trace ID
- [x] Metrics for alert emission
- [x] Comprehensive unit tests (9 test cases)

**3.2.8 Partitioning & Ownership** ✅
- [x] Implement symbol partitioning logic
  - [x] Hash-based: `hash(symbol) % worker_count`
  - [x] Consistent hashing (FNV-32a)
- [x] Worker ID assignment
- [x] Partition discovery
- [x] Rebalancing handling (dynamic worker count updates)
- [x] Comprehensive unit tests (11 test cases)

**3.2.9 State Rehydration** ✅
- [x] On startup: load recent bars from TimescaleDB
- [x] Load indicators from Redis
- [x] Initialize symbol state
- [x] Readiness probe (IsReady)
- [x] Comprehensive unit tests (7 test cases)

**3.2.10 E2E Testing** ✅
- [x] Complete flow E2E test
- [x] Partitioning E2E test
- [x] Multiple rules E2E test
- [x] Comprehensive testing guide (`docs/PHASE3_2_E2E_TESTING.md`)

#### 3.3 Scanner Worker Service (`cmd/scanner`) ✅ COMPLETE

**3.3.1 Service Main** ✅
- [x] Initialize worker
- [x] Load rules (placeholder for API integration)
- [x] Set up subscriptions (ticks, indicators, bars)
- [x] Start scan loop
- [x] Graceful shutdown
- [x] Health checks (/health, /ready, /live)
- [x] Metrics endpoint (/metrics, /stats)
- [x] State rehydration on startup
- [x] Component initialization (all Phase 3.2 components)
- [x] Partition manager integration
- [x] Comprehensive error handling

**3.3.2 Configuration** ✅
- [x] Worker ID/partition config (SCANNER_WORKER_ID, SCANNER_WORKER_COUNT)
- [x] Symbol universe config (SCANNER_SYMBOL_UNIVERSE)
- [x] Scan interval config (SCANNER_SCAN_INTERVAL)
- [x] Buffer sizes (SCANNER_BUFFER_SIZE)
- [x] Cooldown defaults (SCANNER_COOLDOWN_DEFAULT)
- [x] Port configuration (SCANNER_PORT, SCANNER_HEALTH_PORT)
- [x] All environment variables documented in env.example

#### 3.4 Testing ✅ COMPLETE

**3.4.1 Unit Tests** ✅
- [x] Rule compilation tests (covered in rules package tests)
- [x] Metric resolution tests (covered in scan_loop_test.go)
- [x] State update tests (covered in state_test.go)
- [x] Cooldown tests (covered in cooldown_test.go)
- [x] Comprehensive unit test coverage (64.4%+)

**3.4.2 Integration Tests** ✅
- [x] End-to-end: Ingest → Bars → Indicators → Scanner → Alerts
- [x] Test rule matching
- [x] Test cooldown enforcement
- [x] Test partition assignment
- [x] E2E test suite (`tests/scanner_e2e_test.go`)
- [x] Comprehensive testing guide (`docs/PHASE3_2_E2E_TESTING.md`)

**3.4.3 Performance Tests** ✅
- [x] Load test with 2000+ symbols (`TestScanLoop_Performance_2000Symbols`)
- [x] Measure scan cycle time (target <800ms) - **Achieved: 2.7ms for 2000 symbols!**
- [x] Test with varying rule counts (`BenchmarkScanLoop_VaryingRuleCounts`)
- [x] Test with tick bursts (`TestScanLoop_Performance_TickBurst`)
- [x] Concurrent state updates (`TestStateManager_Performance_ConcurrentUpdates`)
- [x] Benchmark tests (`BenchmarkScanLoop_2000Symbols`)

**3.4.4 Chaos Tests** ✅
- [x] Worker restart scenarios (`TestChaos_WorkerRestart`)
- [x] Partition rebalancing (`TestChaos_PartitionRebalancing`)
- [x] Network interruptions (`TestChaos_NetworkInterruption`)
- [x] Verify no duplicate alerts (`TestChaos_NoDuplicateAlerts`)
- [x] Concurrent rule updates (`TestChaos_ConcurrentRuleUpdates`)
- [x] High symbol churn (`TestChaos_HighSymbolChurn`)

---

## Phase 4: Alert Service & WebSocket Gateway (Week 8)

### Goals
- Implement alert deduplication and filtering
- Build WebSocket gateway for real-time delivery
- Persist alerts to storage

### Dependencies
- Phase 3 complete (scanner emitting alerts)

### Tasks

#### 4.1 Alert Service (`cmd/alert` or part of API)

**4.1.1 Alert Consumer**
- [ ] Subscribe to alert stream/pubsub
- [ ] Process alerts
- [ ] Deduplication logic
  - [ ] Idempotency keys
  - [ ] Redis-based dedupe (short-term)
  - [ ] Database check (long-term)

**4.1.2 User Filtering**
- [ ] User subscription management
- [ ] Filter alerts by user preferences
- [ ] Symbol watchlists
- [ ] Rule subscriptions

**4.1.3 Cooldown Enforcement**
- [ ] Per-user, per-rule cooldowns
- [ ] Configurable cooldown periods
- [ ] Cooldown storage (Redis)

**4.1.4 Alert Persistence**
- [ ] Write alerts to ClickHouse/TimescaleDB
- [ ] Batch inserts
- [ ] Async writes
- [ ] Create migration script

**4.1.5 Alert Routing**
- [ ] Route to WebSocket gateway
- [ ] Route to email queue (optional)
- [ ] Route to push notification queue (optional)
- [ ] Metrics for routing

#### 4.2 WebSocket Gateway (`cmd/ws_gateway`)

**4.2.1 WebSocket Server**
- [ ] HTTP upgrade handler
- [ ] Connection management
- [ ] Connection lifecycle (connect, disconnect, ping/pong)
- [ ] Connection registry (in-memory or Redis)
- [ ] Graceful shutdown

**4.2.2 Authentication**
- [ ] JWT token validation
- [ ] User identification
- [ ] Connection authorization

**4.2.3 Message Broadcasting**
- [ ] Receive alerts from alert service
- [ ] Filter by user subscriptions
- [ ] Broadcast to connected clients
- [ ] Handle slow clients (buffering/dropping)
- [ ] Metrics for message delivery

**4.2.4 Client Protocol**
- [ ] Define message format (JSON)
- [ ] Subscribe/unsubscribe messages
- [ ] Heartbeat messages
- [ ] Error messages
- [ ] Alert message format

**4.2.5 Connection Management**
- [ ] Connection pool
- [ ] Rate limiting per connection
- [ ] Max connections per user
- [ ] Connection health monitoring

#### 4.3 Testing
- [ ] Unit tests for alert service
- [ ] Unit tests for WebSocket gateway
- [ ] Integration test: Scanner → Alert Service → WebSocket → Client
- [ ] Load test WebSocket connections (1000+ concurrent)
- [ ] Test reconnection scenarios
- [ ] Test message delivery guarantees

---

## Phase 5: REST API Service (Week 9)

### Goals
- Implement REST API for rule management
- Provide alert history endpoints
- User management and authentication

### Dependencies
- Phase 4 complete (alerts being stored)

### Tasks

#### 5.1 API Service (`cmd/api`)

**5.1.1 API Framework Setup**
- [ ] Choose framework (Gin, Echo, or stdlib)
- [ ] Middleware setup:
  - [ ] CORS
  - [ ] Authentication
  - [ ] Request logging
  - [ ] Error handling
  - [ ] Rate limiting

**5.1.2 Authentication**
- [ ] JWT token generation
- [ ] Token validation middleware
- [ ] User context injection
- [ ] OAuth2 integration (optional for MVP)

**5.1.3 Rule Management Endpoints**
- [ ] `GET /api/v1/rules` - List rules
- [ ] `GET /api/v1/rules/:id` - Get rule
- [ ] `POST /api/v1/rules` - Create rule
- [ ] `PUT /api/v1/rules/:id` - Update rule
- [ ] `DELETE /api/v1/rules/:id` - Delete rule
- [ ] `POST /api/v1/rules/:id/validate` - Validate rule
- [ ] Rule ownership/authorization

**5.1.4 Alert History Endpoints**
- [ ] `GET /api/v1/alerts` - List alerts (paginated)
- [ ] `GET /api/v1/alerts/:id` - Get alert details
- [ ] Filtering: by symbol, rule, date range
- [ ] Sorting options

**5.1.5 Rule Persistence Layer (TimescaleDB)**
- [ ] Create `rules` table in TimescaleDB with schema:
  - `id` (UUID, primary key)
  - `name` (string)
  - `description` (text, nullable)
  - `conditions` (JSONB)
  - `cooldown` (integer, seconds)
  - `enabled` (boolean)
  - `created_at` (timestamp)
  - `updated_at` (timestamp)
  - `version` (integer, for versioning)
- [ ] Implement `DatabaseRuleStore` (persistent storage)
- [ ] Rule Management Service: Sync rules from TimescaleDB to Redis cache
- [ ] Redis pub/sub notifications for rule updates (`rules.updated` channel)
- [ ] Scanner workers subscribe to rule updates and reload rules automatically
- [ ] Rule versioning and rollback support
- [ ] Migration path from Redis-only to Database+Cache pattern
- [ ] Background job to sync rules from DB to Redis on startup

**5.1.6 Symbol Management**
- [ ] `GET /api/v1/symbols` - List available symbols
- [ ] `GET /api/v1/symbols/:symbol` - Get symbol info
- [ ] Symbol search/filter

**5.1.7 User Management (Basic)**
- [ ] `GET /api/v1/user/profile` - Get user profile
- [ ] `PUT /api/v1/user/profile` - Update profile
- [ ] User preferences storage

**5.1.8 Health & Metrics**
- [ ] `GET /health` - Health check
- [ ] `GET /metrics` - Prometheus metrics
- [ ] `GET /ready` - Readiness probe

#### 5.2 API Documentation
- [ ] OpenAPI/Swagger specification
- [ ] Generate docs from code
- [ ] Example requests/responses

#### 5.3 Testing
- [ ] Unit tests for handlers
- [ ] Integration tests for API endpoints
- [ ] Authentication tests
- [ ] Load tests for API

---

## Phase 6: Infrastructure & Deployment (Week 10)

### Goals
- Containerize all services
- Set up Kubernetes manifests
- Configure monitoring and logging
- Create deployment documentation

### Dependencies
- All previous phases complete

### Tasks

#### 6.1 Dockerization

**6.1.1 Dockerfiles**
- [ ] Create Dockerfile for each service
- [ ] Multi-stage builds for optimization
- [ ] Use distroless or alpine base images
- [ ] Set up proper user (non-root)
- [ ] Health check instructions

**6.1.2 Docker Compose Updates**
- [ ] Add all services to docker-compose
- [ ] Configure networking
- [ ] Set up volumes for persistence
- [ ] Environment variable management
- [ ] Development vs production configs

#### 6.2 Kubernetes Manifests

**6.2.1 Deployments**
- [ ] Deployment manifests for each service
- [ ] Resource limits and requests
- [ ] Replica counts
- [ ] Rolling update strategy
- [ ] Pod disruption budgets

**6.2.2 Services**
- [ ] Service manifests (ClusterIP, LoadBalancer)
- [ ] Service discovery
- [ ] Port configurations

**6.2.3 ConfigMaps & Secrets**
- [ ] ConfigMap for non-sensitive config
- [ ] Secrets management
- [ ] Environment variable injection

**6.2.4 Horizontal Pod Autoscaling (HPA)**
- [ ] HPA for scanner workers (CPU + custom metrics)
- [ ] HPA for other services
- [ ] Custom metrics (queue depth, scan cycle time)

**6.2.5 Ingress**
- [ ] Ingress for API service
- [ ] Ingress for WebSocket gateway
- [ ] TLS configuration
- [ ] Rate limiting

#### 6.3 Monitoring & Observability

**6.3.1 Prometheus Configuration**
- [ ] ServiceMonitor CRDs for each service
- [ ] Scrape configurations
- [ ] Alert rules (for Prometheus alerts)
- [ ] Recording rules

**6.3.2 Grafana Dashboards**
- [ ] Dashboard for each service
- [ ] System overview dashboard
- [ ] Alert dashboard
- [ ] Performance dashboard (scan cycle times)
- [ ] Export dashboard JSONs

**6.3.3 Logging**
- [ ] Centralized logging setup (Loki or ELK)
- [ ] Log aggregation configuration
- [ ] Log retention policies
- [ ] Structured log parsing

**6.3.4 Tracing**
- [ ] Jaeger or Tempo setup
- [ ] Trace sampling configuration
- [ ] Service mesh integration (optional)

#### 6.4 Database Migrations

**6.4.1 Migration Tooling**
- [ ] Choose migration tool (golang-migrate, etc.)
- [ ] Create migration scripts:
  - [ ] TimescaleDB hypertables
  - [ ] ClickHouse tables
  - [ ] Indexes
- [ ] Migration versioning
- [ ] Rollback scripts

#### 6.5 CI/CD

**6.5.1 CI Pipeline**
- [ ] GitHub Actions / GitLab CI config
- [ ] Run tests on PR
- [ ] Linting (golangci-lint)
- [ ] Security scanning
- [ ] Build Docker images
- [ ] Push to registry

**6.5.2 CD Pipeline**
- [ ] Deployment to staging
- [ ] Deployment to production
- [ ] Blue-green or canary deployment
- [ ] Rollback procedures

#### 6.6 Documentation

**6.6.1 Deployment Guide**
- [ ] Prerequisites
- [ ] Step-by-step deployment instructions
- [ ] Configuration reference
- [ ] Troubleshooting guide

**6.6.2 Operations Guide**
- [ ] Monitoring runbook
- [ ] Incident response procedures
- [ ] Scaling procedures
- [ ] Backup/restore procedures

---

## Phase 7: Testing & Optimization (Week 11)

### Goals
- Comprehensive end-to-end testing
- Performance optimization
- Load testing and capacity planning
- Bug fixes and stability improvements

### Tasks

#### 7.1 End-to-End Testing

**7.1.1 Test Scenarios**
- [ ] Full pipeline test: Ingest → Alerts
- [ ] Multi-worker partitioning test
- [ ] Reconnection and recovery tests
- [ ] Data consistency tests
- [ ] Alert deduplication tests

**7.1.2 Test Infrastructure**
- [ ] Test data generators
- [ ] Mock market data provider
- [ ] Test harness for E2E tests
- [ ] Test environment setup

#### 7.2 Performance Testing

**7.2.1 Load Tests**
- [ ] Test with 2000 symbols
- [ ] Test with 5000 symbols
- [ ] Test with 10000 symbols
- [ ] Measure scan cycle times
- [ ] Measure end-to-end latency
- [ ] Identify bottlenecks

**7.2.2 Stress Tests**
- [ ] Tick burst scenarios
- [ ] High rule count scenarios
- [ ] Many concurrent WebSocket connections
- [ ] Database connection pool exhaustion
- [ ] Memory pressure tests

**7.2.3 Optimization**
- [ ] Profile hot paths
- [ ] Optimize allocations
- [ ] Optimize locking
- [ ] Optimize serialization
- [ ] Tune buffer sizes
- [ ] Tune worker counts

#### 7.3 Stability Testing

**7.3.1 Chaos Engineering**
- [ ] Random pod kills
- [ ] Network partitions
- [ ] Database failures
- [ ] Redis failures
- [ ] High latency injection
- [ ] Verify recovery

**7.3.2 Long-Running Tests**
- [ ] 24-hour stability test
- [ ] Memory leak detection
- [ ] Resource usage monitoring
- [ ] Alert accuracy over time

#### 7.4 Bug Fixes & Refinement
- [ ] Fix identified issues
- [ ] Code review and refactoring
- [ ] Documentation updates
- [ ] Performance tuning based on test results

---

## Phase 8: Production Readiness (Week 12)

### Goals
- Security hardening
- Production configuration
- Documentation completion
- Final validation

### Tasks

#### 8.1 Security

**8.1.1 Security Hardening**
- [ ] Security audit
- [ ] Dependency vulnerability scanning
- [ ] Secrets management review
- [ ] Network security (firewall rules)
- [ ] TLS configuration review
- [ ] Authentication/authorization review

**8.1.2 Compliance**
- [ ] Data retention policies
- [ ] Audit logging
- [ ] PII handling
- [ ] GDPR considerations (if applicable)

#### 8.2 Production Configuration

**8.2.1 Configuration Review**
- [ ] Production config values
- [ ] Resource limits
- [ ] Scaling parameters
- [ ] Timeout values
- [ ] Retry policies

**8.2.2 Backup & Recovery**
- [ ] Backup procedures
- [ ] Recovery procedures
- [ ] Disaster recovery plan
- [ ] Test backup/restore

#### 8.3 Documentation

**8.3.1 User Documentation**
- [ ] API documentation
- [ ] Rule creation guide
- [ ] WebSocket client guide
- [ ] Troubleshooting guide

**8.3.2 Developer Documentation**
- [ ] Architecture diagrams
- [ ] Code structure guide
- [ ] Contributing guidelines
- [ ] Development setup guide

#### 8.4 Final Validation
- [ ] Production readiness checklist
- [ ] Performance validation
- [ ] Security validation
- [ ] Documentation review
- [ ] Stakeholder sign-off

---

## Implementation Priorities

### MVP Must-Haves (Phases 0-5)
1. Project setup and infrastructure
2. Data pipeline (ingest → bars → indicators)
3. Scanner worker with basic rules
4. Alert service and WebSocket gateway
5. Basic REST API

### Nice-to-Haves (Can defer)
- ClickHouse (can use TimescaleDB initially)
- Kafka (can use Redis Streams initially)
- Advanced authentication (OAuth2)
- Email/push notifications
- Advanced monitoring dashboards

---

## Risk Mitigation

### Technical Risks
1. **Scan cycle time > 1s**
   - Mitigation: Early performance testing, optimization, horizontal scaling
   
2. **Data loss on worker restart**
   - Mitigation: Durable streams, state rehydration, idempotency

3. **Partition rebalancing issues**
   - Mitigation: Consumer groups, proper ownership protocol, testing

4. **Memory leaks in long-running workers**
   - Mitigation: Profiling, memory monitoring, regular restarts

### Operational Risks
1. **Market data provider outages**
   - Mitigation: Multiple providers, fallback logic, degraded mode

2. **Database performance**
   - Mitigation: Proper indexing, connection pooling, monitoring

3. **Scaling bottlenecks**
   - Mitigation: Load testing, autoscaling, capacity planning

---

## Success Metrics

### Performance
- Scan cycle time: < 800ms (p95)
- End-to-end latency (tick → alert): < 2s (p95)
- System uptime: > 99.9%

### Scalability
- Support 10,000 symbols
- Support 100+ concurrent WebSocket connections
- Horizontal scaling working

### Reliability
- Zero data loss
- Graceful degradation
- Fast recovery from failures

---

## Timeline Summary

- **Week 1**: Phase 0 (Setup)
- **Weeks 2-3**: Phase 1 (Data Pipeline)
- **Week 4**: Phase 2 (Indicators)
- **Weeks 5-7**: Phase 3 (Scanner Worker)
- **Week 8**: Phase 4 (Alerts & WebSocket)
- **Week 9**: Phase 5 (REST API)
- **Week 10**: Phase 6 (Infrastructure)
- **Week 11**: Phase 7 (Testing & Optimization)
- **Week 12**: Phase 8 (Production Readiness)

**Total: ~12 weeks for full implementation**

---

## Next Steps

1. Review and approve this implementation plan
2. Set up project repository and Phase 0 tasks
3. Begin Phase 0 implementation
4. Set up regular review checkpoints
5. Adjust plan based on learnings

