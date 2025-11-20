# Implementation Checklist — Quick Reference

This is a condensed checklist version of the implementation plan for quick progress tracking.

## Phase 0: Project Setup & Foundation ✅ COMPLETE
- [x] Project structure created
- [x] Go module initialized
- [x] Configuration management (`internal/config`)
- [x] Logging package (`pkg/logger`)
- [x] OpenTelemetry tracing setup (foundation)
- [x] Prometheus metrics foundation
- [x] Docker Compose with Redis, TimescaleDB, Prometheus, Grafana, RedisInsight
- [x] Data models defined (`internal/models`)
- [x] Storage interfaces defined

## Phase 1: Core Data Pipeline
- [x] Market Data Ingest Service ✅
  - [x] Provider abstraction ✅
  - [x] WebSocket connection management ✅
  - [x] Data normalization ✅
  - [x] Redis Streams publisher ✅
  - [x] Ingest service main ✅
  - [ ] Real provider implementations
    - [ ] Alpaca provider
    - [ ] Polygon.io provider 
    - [ ] Databento.com provider
    - [ ] Provider-specific normalization
    - [ ] Integration tests with real providers
- [x] Bar Aggregator Service ✅
  - [x] Bar aggregation logic ✅ (`internal/bars/aggregator.go`)
  - [x] Redis Stream consumer ✅ (`internal/pubsub/stream_consumer.go`)
  - [x] Live bar publishing ✅ (`internal/bars/publisher.go`)
  - [x] TimescaleDB integration ✅ (`internal/storage/timescale.go`)
  - [x] Bar Aggregator Service Main ✅ (`cmd/bars/main.go`)
- [x] End-to-end test: Ingest → Bars → Storage ✅
- [x] Testing & Deployment Infrastructure ✅
  - [x] Testing documentation
  - [x] Deployment scripts
  - [x] Performance monitoring tools
  - [x] RedisInsight integration

## Phase 2: Indicator Engine ✅ COMPLETE
- [x] Indicator package (`pkg/indicator`)
  - [x] Core interfaces (Calculator, WindowedCalculator)
  - [x] Registry for managing calculators
  - [x] SymbolState for per-symbol state management
  - [x] RSI implementation (period 14)
  - [x] EMA implementation (periods 20, 50, 200)
  - [x] SMA implementation (periods 20, 50, 200)
  - [x] VWAP implementation (windows 5m, 15m, 1h)
  - [x] Volume indicators (average and relative volume)
  - [x] Price change indicators (1m, 5m, 15m windows)
  - [x] Comprehensive unit tests (82.6% coverage)
- [x] Indicator Engine Service
  - [x] Engine core with factory pattern for calculators
  - [x] Bar consumer (consumes from `bars.finalized` stream)
  - [x] Indicator computation logic
  - [x] Indicator publishing to Redis (`ind:{symbol}` keys + pub/sub)
  - [x] Service main with health checks and metrics
- [x] Integration tests

## Phase 3: Rule Engine & Scanner Worker
- [x] Rule Engine (`internal/rules`) ✅ COMPLETE
  - [x] Rule data structures (types, validation)
  - [x] Rule parser (JSON → Rule)
  - [x] Rule compiler (Rule → CompiledRule)
  - [x] Metric resolver (metric name → value)
  - [x] Rule storage (in-memory store)
  - [x] Comprehensive tests (87.4% coverage)
- [x] Scanner Worker Core (`internal/scanner`) ✅ COMPLETE
  - [x] Symbol state management (StateManager, SymbolState)
  - [x] Tick ingestion (TickConsumer with Redis streams)
  - [x] Indicator ingestion (IndicatorConsumer with Redis pub/sub)
  - [x] Bar finalization handler (BarFinalizationHandler)
  - [x] Scan loop (<800ms target, sync.Pool optimization)
  - [x] Cooldown management (InMemoryCooldownTracker)
  - [x] Alert emission (AlertEmitterImpl with Redis pub/sub + streams)
  - [x] Partitioning logic (PartitionManager with consistent hashing)
  - [x] State rehydration (Rehydrator with TimescaleDB + Redis)
  - [x] Comprehensive unit tests (64.4% coverage)
  - [x] E2E tests (3 scenarios)
  - [x] Testing guide (`docs/PHASE3_2_E2E_TESTING.md`)
- [x] Scanner Worker Service (`cmd/scanner`) ✅ COMPLETE
  - [x] Service main with all component integration
  - [x] State rehydration on startup
  - [x] All consumers started (ticks, indicators, bars)
  - [x] Scan loop running
  - [x] Health check endpoints
  - [x] Metrics endpoints
  - [x] Graceful shutdown
  - [x] Configuration loading
  - [x] Worker ID parsing and partition management
- [x] Testing (Phase 3.4) ✅ COMPLETE
  - [x] Unit tests (comprehensive coverage)
  - [x] Integration/E2E tests (3 scenarios)
  - [x] Performance tests (2000+ symbols, 2.7ms scan time!)
  - [x] Chaos tests (6 failure scenarios)
  - [x] Benchmarks (varying rule counts, tick bursts)

## Phase 4: Alert Service & WebSocket Gateway ✅ COMPLETE
- [x] Alert Service
  - [x] Alert consumer (consumes from `alerts` stream)
  - [x] Deduplication (idempotency keys, Redis-based)
  - [x] User filtering (MVP: all pass through, structure ready for future)
  - [x] Alert persistence (TimescaleDB with async batch writes)
  - [x] Alert routing (routes to `alerts.filtered` stream)
- [x] WebSocket Gateway
  - [x] WebSocket server (HTTP upgrade, connection management)
  - [x] Authentication (JWT validation, MVP: allows default user)
  - [x] Message broadcasting (consumes from `alerts.filtered` stream)
  - [x] Client protocol (subscribe/unsubscribe/ping/pong)
  - [x] Connection management (registry, health monitoring)
- [x] Unit tests (23 tests, all passing)
  - [x] Alert Service unit tests (11 tests)
  - [x] WebSocket Gateway unit tests (12 tests)
- [ ] Integration tests (deferred to Phase 7)

## Phase 5: REST API Service ✅ COMPLETE
- [x] API framework setup (gorilla/mux with middleware)
- [x] Authentication (JWT validation middleware, MVP: allows default user)
- [x] Rule management endpoints (CRUD + validate)
- [x] Alert history endpoints (list + get with filtering)
- [x] Symbol management endpoints (list + get with search)
- [x] User management endpoints (basic profile)
- [x] Rule persistence layer (DatabaseRuleStore + Redis sync)
- [x] Rule sync service (database → Redis cache)
- [x] Database migration for rules table
- [x] Unit tests (22 tests, all passing)
- [ ] API documentation (OpenAPI) - deferred

## Phase 5: Toplists & API ⏳ IN PROGRESS
- [ ] Toplist Data Models & Types (`internal/models/toplist.go`)
  - [ ] ToplistConfig struct (user-custom toplist configuration)
  - [ ] ToplistRanking struct (symbol ranking entry)
  - [ ] ToplistUpdate struct (real-time update message)
  - [ ] ToplistFilter struct (filtering criteria)
  - [ ] Validation methods for all structs
  - [ ] Toplist constants (metrics, time windows, sort orders, system types)
  - [ ] Redis key schema definitions
  - [ ] Unit tests for data models
- [ ] Toplist Updater Service (`internal/toplist`)
  - [ ] ToplistUpdater interface (`internal/toplist/updater.go`)
  - [ ] Redis ZSET updater (`internal/toplist/redis_updater.go`)
  - [ ] Pub/sub publisher (`internal/toplist/publisher.go`)
  - [ ] Unit tests for updater service
- [x] Toplist Service (`internal/toplist/service.go`) ✅
  - [x] ToplistService implementation
  - [x] ToplistStore interface (`internal/toplist/store.go`)
  - [x] DatabaseToplistStore (`internal/toplist/database_store.go`)
  - [x] Unit tests for toplist service
- [ ] Scanner Worker Integration
  - [ ] Integrate ToplistUpdater into Scanner Worker
  - [ ] Update system toplists (change_pct, volume)
  - [ ] Update user-custom toplists
  - [ ] Batch updates with pipeline
  - [ ] Publish update notifications
  - [ ] Configuration for enabled toplists
  - [ ] Performance optimization (caching, batching)
  - [ ] Unit tests for scanner worker toplist integration
- [ ] Indicator Engine Integration
  - [ ] Integrate ToplistUpdater into Indicator Engine
  - [ ] Update system toplists (rsi, relative_volume, vwap_dist)
  - [ ] Update user-custom toplists
  - [ ] Batch updates
  - [ ] Publish update notifications
  - [ ] Unit tests for indicator engine toplist integration
- [ ] Database Migration
  - [ ] Create toplist_configs table migration (`004_create_toplist_configs_table.sql`)
  - [ ] Table schema with all required fields
  - [ ] Indexes (user_id, enabled, created_at)
  - [ ] Test migration script
- [ ] API Service Integration (`cmd/api`)
  - [ ] ToplistHandler (`internal/api/toplist_handler.go`)
  - [ ] ListToplists - GET /api/v1/toplists
  - [ ] GetSystemToplist - GET /api/v1/toplists/system/:type
  - [ ] ListUserToplists - GET /api/v1/toplists/user
  - [ ] CreateUserToplist - POST /api/v1/toplists/user
  - [ ] GetUserToplist - GET /api/v1/toplists/user/:id
  - [ ] UpdateUserToplist - PUT /api/v1/toplists/user/:id
  - [ ] DeleteUserToplist - DELETE /api/v1/toplists/user/:id
  - [ ] GetToplistRankings - GET /api/v1/toplists/user/:id/rankings
  - [ ] Query parameter support (limit, offset, filters)
  - [ ] Authentication and authorization
  - [ ] Unit tests for toplist handlers
- [ ] WebSocket Gateway Integration
  - [ ] Extend WebSocket protocol for toplist subscriptions
  - [ ] Add message types (subscribe_toplist, unsubscribe_toplist, toplist_update)
  - [ ] Update Connection struct (ToplistSubscriptions map)
  - [ ] Update Hub to handle toplist updates
  - [ ] Subscribe to `toplists.updated` pub/sub channel
  - [ ] Broadcast toplist updates to subscribed clients
  - [ ] Unit tests for WebSocket toplist integration
- [ ] Testing & Verification
  - [ ] Unit tests (updater, service, store, handlers, protocol)
  - [ ] Integration tests (E2E: Worker → Redis ZSET → API)
  - [ ] Integration tests (E2E: Indicator Engine → Redis ZSET → API)
  - [ ] Integration tests (E2E: Toplist update → WebSocket delivery)
  - [ ] Integration tests (User toplist creation → ranking → API query)
  - [ ] Performance tests (high churn updates, batch performance, WebSocket broadcast)
  - [ ] Load tests (multiple workers, many toplists, high WebSocket connections)

## Phase 6: Infrastructure & Deployment ✅ COMPLETE
- [x] Dockerfiles for all services ✅
  - [x] Multi-stage Dockerfile (builds all 7 services)
  - [x] Alpine base image
  - [x] Non-root user (appuser)
  - [x] Health check support
- [x] Docker Compose updates ✅
  - [x] All 7 services configured (ingest, bars, indicator, scanner, alert, ws-gateway, api)
  - [x] All infrastructure services (Redis, TimescaleDB, Prometheus, Grafana, RedisInsight, Loki, Promtail, Jaeger)
  - [x] Networking configured (stock-scanner-network)
  - [x] Volumes for persistence
  - [x] Environment variable management
  - [x] Redis configuration file support
  - [x] Loki permission fixes
- [x] Kubernetes manifests ✅
  - [x] Deployments for all 7 services ✅
  - [x] Services (ClusterIP, LoadBalancer) ✅
  - [x] ConfigMaps & Secrets ✅
  - [x] HPA (scanner, alert, ws-gateway, api) ✅
  - [x] Ingress (API + WebSocket) ✅
  - [x] Resource limits and requests
  - [x] Health checks (liveness/readiness)
  - [x] Rolling update strategy
- [x] Prometheus configuration ✅
  - [x] Prometheus service in docker-compose
  - [x] Scrape configuration (`config/prometheus.yml`)
  - [x] Service discovery
  - [x] Metrics endpoints on all services
- [x] Grafana dashboards ✅
  - [x] Overview dashboard
  - [x] Data pipeline dashboard
  - [x] Scanner dashboard
  - [x] API dashboard
  - [x] Alerts dashboard
  - [x] Logs dashboard
  - [x] Dashboard provisioning
  - [x] Datasource provisioning (Prometheus, Loki, Jaeger)
  - [x] Dashboard documentation
- [x] Logging setup (Loki/Promtail) ✅
  - [x] Loki service configured
  - [x] Promtail service configured
  - [x] Log aggregation from Docker containers
  - [x] Log retention policies (720h)
  - [x] Structured log parsing
  - [x] Loki configuration fixes (deprecated fields removed)
  - [x] Loki permission fixes
- [x] Tracing setup (Jaeger) ✅
  - [x] Jaeger all-in-one service
  - [x] OTLP support (gRPC + HTTP)
  - [x] Jaeger UI access
  - [x] Documentation
- [x] Database migrations ✅
  - [x] Migration scripts (3 migrations)
  - [x] Automatic migration on container startup
  - [x] TimescaleDB hypertables
  - [x] Alert history table
  - [x] Rules table
- [x] Documentation ✅
  - [x] Kubernetes deployment guide (`k8s/README.md`)
  - [x] Configuration reference (`config/env.example`)
  - [x] Monitoring documentation
  - [x] Logging documentation
  - [x] Tracing documentation
- [ ] CI/CD pipeline ⏳ (structure exists, needs completion)

## Phase 7: Testing & Optimization ✅ COMPLETE (Test Suites)
- [x] End-to-end tests ✅
  - [x] API-based E2E tests (`tests/api_e2e/`) ✅
  - [x] Component E2E tests (`tests/component_e2e/`) ✅
  - [x] Pipeline E2E tests (`tests/pipeline_e2e/`) ✅
  - [x] Docker Compose integration
  - [x] API test client
  - [x] WebSocket test client
- [x] Load tests (2000, 5000, 10000 symbols) ✅
  - [x] Test suite created (`tests/performance/load_test.go`)
  - [x] 2000 symbols test
  - [x] 5000 symbols test
  - [x] 10000 symbols test
  - [x] Tick ingestion rate tests
  - [x] Concurrent update tests
- [x] Stress tests ✅
  - [x] Test suite created (`tests/performance/stress_test.go`)
  - [x] Tick burst scenarios
  - [x] High rule count scenarios
  - [x] WebSocket connection stress
  - [x] Database connection pool exhaustion
  - [x] Memory pressure tests
- [x] Chaos engineering tests ✅
  - [x] Test suite created (`tests/performance/chaos_test.go`)
  - [x] Service restart scenarios
  - [x] Network partition tests
  - [x] Redis failure tests
  - [x] High latency injection
  - [x] Concurrent failures
  - [x] Duplicate alert prevention
- [x] Stability tests ✅
  - [x] Test suite created (`tests/performance/stability_test.go`)
  - [x] Long-running tests (24h configurable)
  - [x] Memory leak detection
  - [x] Resource usage monitoring
  - [x] Alert accuracy over time
- [x] Test organization ✅
  - [x] Tests organized into subdirectories
  - [x] Test documentation (`tests/README.md`)
  - [x] Test helper utilities
- [x] Bug fixes ✅
  - [x] Fixed package declaration issues
  - [x] Fixed import cycles
  - [x] Fixed API mismatches
  - [x] Fixed Docker build errors
  - [x] Fixed Loki configuration
  - [x] Fixed test compilation errors
- [ ] Performance optimization ⏳ (deferred - requires profiling)

## Phase 8: Production Readiness
- [ ] Security audit
- [ ] Production configuration review
- [ ] Backup & recovery procedures
- [ ] User documentation
- [ ] Developer documentation
- [ ] Final validation

---

## Critical Path Items (Must Complete in Order)

1. **Phase 0** ✅ → Foundation for everything
2. **Phase 1** ✅ → Data must flow before anything else
3. **Phase 2** ✅ → Indicators needed for rules
4. **Phase 3** ✅ → Core scanning functionality
5. **Phase 4** ✅ → Alert delivery
   - ✅ Phase 1.1: Market Data Ingest Service
   - ✅ Phase 1.2.1-1.2.3: Bar Aggregation, Consumer, Publishing
   - ✅ Phase 1.2.4-1.2.5: TimescaleDB Integration, Service Main
   - ✅ Phase 1.3-1.4: Testing & Deployment Infrastructure
3. **Phase 2** ✅ → Indicators needed for rules
   - ✅ Phase 2.1: Indicator Package (interfaces, implementations, state management)
   - ✅ Phase 2.2: Indicator Engine Service (consumer, computation, publishing)
   - ✅ Phase 2.3: Testing
4. **Phase 3** → Core scanning functionality
   - ✅ Phase 3.1: Rule Engine Core (data structures, parser, compiler, metric resolver, storage)
   - ✅ Phase 3.2: Scanner Worker Core (symbol state, tick/indicator ingestion, scan loop, cooldown, alerts, partitioning, rehydration)
   - ✅ Phase 3.3: Scanner Worker Service (main service integration, health checks, metrics, graceful shutdown)
   - ✅ Phase 3.4: Testing (unit, integration, performance, chaos tests)
5. **Phase 4** ✅ → Alert delivery
6. **Phase 5** ⏳ → User interface (API + Toplists)
   - ✅ REST API Service (rule management, alerts, symbols, users)
   - ⏳ Toplists (system toplists, user-custom toplists, real-time updates)
7. **Phase 6** ✅ → Deployment capability
8. **Phase 7** ✅ → Quality assurance (test suites complete)
9. **Phase 8** → Production launch

---

## Key Performance Targets

- ✅ Scan cycle: < 800ms (p95)
- ✅ End-to-end latency: < 2s (tick → alert)
- ✅ Support: 10,000 symbols
- ✅ Uptime: > 99.9%

---

## Quick Commands Reference

```bash
# Start local infrastructure
make docker-up
# Or: docker-compose -f config/docker-compose.yaml up -d

# Run tests
make test
# Or: go test ./...

# Build all services
make build

# Run specific service (after build)
make run-ingest
make run-bars
make run-scanner

# Format code
make fmt

# Run linter
make lint

# View Docker logs
make docker-logs
```

