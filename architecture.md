# Architecture — Real‑Time Trading Scanner (MVP)

## Purpose
This document describes the architecture for the MVP real‑time trading scanner. It is written to guide engineers and Copilot-style assistants implementing the system. The focus is on a production-minded, horizontally scalable, low-latency design that finishes a full scan cycle in <1s.

**Last Updated**: Architecture refined with detailed service boundaries, storage strategy, communication patterns, and Toplist feature.

---

## High-level Components

```
[Market Data Provider]
        |
        v
[Market Data Ingest Service] --> (Redis Streams) --> [Bar Aggregator] --> TimescaleDB (finished 1m bars)
                                         |                                      |
                                         v                                      |
                                  Indicator Engine                              |
                                         |                                      |
                                   (Redis pub/sub)                              |
                                         |                                      |
                 +-----------------------+------------------------+            |
                 |                        |                       |            |
            [Scanner Worker A]       [Scanner Worker B]      [Scanner Worker N]
                 |                        |                       |
                 +-----------(alerts)-----+-----------------------+
                                         |
                                 [Alert Service]
                                         |
                    +--------------------+--------------------+
                    |                    |                    |
            [Alert Queue]          [TimescaleDB]      [WebSocket Gateway]
            (Redis Stream)         (alert history)           |
                                                             |
                                                        Users (Web / Mobile)
                                                             ^
                                                             |
                                                       [API Service] <--- (Redis ZSETs) --- [Toplists]
                                                             |
                                                             v
                                                    [Toplist Service] (new)
                                                             |
                                                             +---> User-configurable toplists
                                                             +---> Real-time updates via WebSocket
                                                             +---> Filtering & sorting

```

### Additional Services

- **Rule Management Service**: Manages rule storage, validation, and distribution
- **API Service**: REST API for rule management, alert history, toplist management, and system management
- **Toplist Service**: Manages user-configurable toplists, real-time ranking, and WebSocket updates

### Components Explained

- **Market Data Ingest Service**
  - Connects to data providers (Polygon, IEX, dxFeed).
  - Normalizes ticks/trades/quotes to common format.
  - Publishes to Redis Streams with partition key = symbol (hash-based).
  - Handles reconnection, backoff, and provider failover.
  - **Decision**: Redis Streams for MVP (simpler ops, migration path to Kafka available).

- **Bar Aggregator**
  - Builds current live 1m bars (open/high/low/close/volume/vwap) in-memory per symbol.
  - Writes live bar snapshots to Redis key `livebar:{symbol}` (TTL: 5 minutes).
  - When minute boundary passes: finalizes bar, writes to TimescaleDB (async batch), publishes `bars.finalized` event to Redis Stream.
  - Handles backpressure and tick ordering.

- **Indicator Engine**
  - Consumes finalized bars from `bars.finalized` stream.
  - Computes complex indicators: RSI, EMA (multiple periods), VWAP (multiple windows).
  - Writes indicators to Redis key `ind:{symbol}` (TTL: 10 minutes).
  - Publishes indicator updates to Redis pub/sub channel `indicators.updated` (for real-time workers).
  - **Decision**: Complex indicators pre-computed; simple metrics computed in workers.

- **Scanner Workers**
  - Core scanning processes (multiple instances, horizontally scalable).
  - Each worker assigned partition of symbols via consistent hash: `hash(symbol) % worker_count`.
  - **Data ingestion strategy**:
    - Subscribe to tick stream for assigned symbols (build live bars locally).
    - Subscribe to indicator updates via Redis pub/sub (for complex indicators).
    - Compute simple metrics locally (price change %, volume ratios).
  - Maintain all required data in local memory (live bar, recent finalized bars, indicators).
  - Execute rule evaluation every scan interval (1s).
  - Per-worker cooldown tracking (in-memory, fast path).
  - Per-worker cooldown tracking (in-memory, fast path).
  - Publish matches to `alerts` Redis Stream with idempotency key.
  - **Toplist Updates**:
    - Updates Redis Sorted Sets (ZSET) for simple metrics (e.g., `toplist:change_1m`).
    - Updates both system-wide toplists and user-custom toplists.
    - Uses pipelining to minimize RTT.
    - Publishes toplist updates to Redis pub/sub channel `toplists.updated` for real-time delivery.


- **Alert Service**
  - Consumes alerts from `alerts` Redis Stream.
  - **Deduplication**: Uses idempotency keys to prevent duplicates (cross-worker safety).
  - **User filtering**: Applies user subscriptions, symbol watchlists, rule filters.
  - **Cooldown enforcement**: Per-user, per-rule cooldowns (Redis-backed for consistency).
  - Persists alerts to TimescaleDB (alert_history table).
  - Publishes filtered alerts to `alerts.filtered` Redis Stream for WebSocket Gateway.
  - **Decision**: Separated from WebSocket Gateway for better scalability.

- **WebSocket Gateway**
  - Consumes filtered alerts from `alerts.filtered` stream.
  - Consumes toplist updates from `toplists.updated` pub/sub channel.
  - Manages WebSocket connections (connection pool, heartbeat, ping/pong).
  - Authenticates connections via JWT tokens.
  - Broadcasts alerts to subscribed clients.
  - Broadcasts toplist updates to subscribed clients.
  - Supports subscriptions to both alerts and toplists.
  - Handles slow clients (buffering, dropping if needed).
  - Metrics for connection count, message delivery latency.

- **Rule Management Service**
  - Stores rules in TimescaleDB (rules table).
  - Caches active rules in Redis for fast access.
  - Validates rule syntax and metric references.
  - Distributes rule updates to workers via Redis pub/sub `rules.updated`.
  - Supports rule versioning and rollback.
  - **Decision**: Database + Cache pattern for rules.

- **API Service**
  - REST API with versioning (`/api/v1/...`).
  - Endpoints: rule management, alert history, symbol info, user profile, toplist management.
  - Toplist endpoints:
    - List available toplist types
    - Get system toplists (gainers, losers, volume, etc.)
    - Create/update/delete user-custom toplists
    - Get toplist rankings with filtering and sorting
  - Authentication via JWT tokens.
  - Rate limiting per user.
  - Health checks and metrics.

- **Toplist Service** (New Component)
  - Manages user-configurable toplists stored in TimescaleDB.
  - Toplist configurations include:
    - Metric to rank by (change_pct, volume, rsi, etc.)
    - Time window (1m, 5m, 15m, 1h, 1d)
    - Filters (min volume, price range, exchange, etc.)
    - Sort order (ascending/descending)
    - Column display preferences
  - Queries Redis ZSETs for real-time rankings.
  - Publishes toplist updates to Redis pub/sub for WebSocket delivery.
  - Supports both system-wide toplists (predefined) and user-custom toplists.
  - Caches toplist configurations in Redis for fast access.

- **Storage Strategy**
  - **Redis** (Tier 1 - Hot):
    - Live bars: `livebar:{symbol}` (TTL: 5 min)
    - Indicators: `ind:{symbol}` (TTL: 10 min)
    - Rules cache: `rules:{rule_id}` (TTL: 1 hour)
    - Cooldown state: `cooldown:{user_id}:{rule_id}:{symbol}` (TTL: cooldown duration)
    - Pub/sub channels: `indicators.updated`, `rules.updated`, `toplists.updated`
    - Streams: `ticks`, `bars.finalized`, `alerts`, `alerts.filtered`
    - **System Toplists**: `toplist:{metric}:{window}` (ZSET, e.g., `toplist:change_pct:1m`)
    - **User Toplists**: `toplist:user:{user_id}:{toplist_id}` (ZSET, for user-custom toplists)
    - **Toplist Configs**: `toplist:config:{toplist_id}` (Hash, cached toplist configuration)
  - **TimescaleDB** (Tier 2 - Warm):
    - Finalized 1m bars (hypertable, retention: 1 year)
    - Alert history (retention: 1 year)
    - Rules (persistent storage)
    - **Toplist configurations** (user-custom toplists, retention: permanent)
    - **Decision**: Single database for MVP (defer ClickHouse to post-MVP).
  - **S3** (Tier 3 - Cold, post-MVP):
    - Long-term archives (> 1 year)
    - Parquet files for analytics

- **Infrastructure**
  - Kubernetes for orchestration.
  - Prometheus + Grafana for metrics.
  - Loki for logs (ELK deferred).
  - OpenTelemetry for traces (Jaeger backend).

---

## Data Flow (step-by-step)

1. **Tick ingestion**
   - Ingest connects to market data websocket feed and receives tick/trade messages.
   - Normalizes messages to common `Tick` format (symbol, price, volume, timestamp).
   - Publishes to Redis Stream `ticks` with partition key = `hash(symbol) % num_partitions`.
   - Ensures ordering: all ticks for a symbol go to same partition.

2. **Live bar updates**
   - Bar Aggregator consumes `ticks` stream (consumer group: `bar-aggregator`).
   - Updates in-memory live bar per symbol:
     - Update high/low/close
     - Accumulate volume
     - Update VWAP numerator/denominator
   - Writes live bar snapshot to Redis key `livebar:{symbol}` (TTL: 5 min, JSON format).
   - On minute boundary (detected by timestamp):
     - Finalizes bar (sets final values)
     - Queues write to TimescaleDB (async batch insert)
     - Publishes `bars.finalized` event to Redis Stream (includes full bar data)

3. **Indicator precompute**
   - Indicator Engine consumes `bars.finalized` stream (consumer group: `indicator-engine`).
   - Updates rolling windows per symbol (maintains last N bars in memory).
   - Computes indicators:
     - **Complex**: RSI, EMA (20, 50, 200), VWAP (5m, 15m, 1h windows)
     - **Simple**: Computed in workers (price change %, volume ratios)
   - Writes indicators to Redis key `ind:{symbol}` (TTL: 10 min, JSON format).
   - Publishes indicator update to Redis pub/sub channel `indicators.updated` (real-time notification).

4. **Worker ingestion & local cache**
   - Scanner Workers subscribe to:
     - **Tick stream**: Consumer group `scanner-workers`, assigned partitions based on symbol hash.
     - **Indicator updates**: Redis pub/sub channel `indicators.updated` (all workers subscribe).
   - On tick arrival:
     - Updates local in-memory live bar (same logic as Bar Aggregator).
     - Updates incremental metrics (vwapNum/vwapDen, last_price, volume counters).
   - On indicator update:
     - Updates local `indicators` map for the symbol.
   - Workers compute simple metrics on-demand:
     - Price change % (1m, 5m, 15m) from finalized bars ring buffer.
     - Relative volume from current volume vs average.
   - **Critical**: Workers never perform network I/O in hot path; all data in-memory.

5. **Rule evaluation & scanning**
   - Every 1 second, each worker runs scan loop:
     - Snapshot symbol list (copy keys under RLock).
     - For each symbol:
       - Acquire RLock on symbol state.
       - Fetch metrics (from indicators map + computed from live/finalized bars).
       - Evaluate compiled rules (function calls, no parsing).
       - Release RLock.
     - For matching rules:
       - Check per-worker cooldown (in-memory map, fast path).
       - Generate idempotency key: `{rule_id}:{symbol}:{timestamp_rounded_to_second}`.
       - Emit alert to `alerts` Redis Stream (includes idempotency key, rule_id, symbol, metrics, timestamp).

6. **Alert processing & delivery**
   - Alert Service consumes `alerts` stream (consumer group: `alert-service`).
   - **Deduplication**: Checks idempotency key in Redis (SET with TTL, at-least-once semantics).
   - **User filtering**:
     - Loads user subscriptions from database (cached in Redis).
     - Filters by symbol watchlists, rule subscriptions.
   - **Cooldown enforcement**: Checks per-user, per-rule cooldown in Redis (key: `cooldown:{user_id}:{rule_id}:{symbol}`).
   - **Persistence**: Writes alert to TimescaleDB `alert_history` table (async batch).
   - **Routing**: Publishes filtered alert to `alerts.filtered` Redis Stream.
   - WebSocket Gateway consumes `alerts.filtered` stream.
   - Gateway looks up connected clients for the user.
   - Broadcasts alert via WebSocket (JSON message).
   - Tracks delivery metrics (latency, success/failure).

7. **Toplist Maintenance**
   - **Scanner Workers**:
     - As they process symbols, they calculate simple metrics (change %, volume).
     - Periodically (e.g., every 1s or batch), update Redis ZSETs:
       - System toplists: `ZADD toplist:change_pct:1m <value> <symbol>`
       - System toplists: `ZADD toplist:volume:day <value> <symbol>`
       - User-custom toplists: Based on user configurations, update relevant ZSETs
     - After batch update, publish to `toplists.updated` pub/sub channel.
   - **Indicator Engine**:
     - Updates complex metric toplists (e.g., RSI, Relative Volume).
     - Updates both system and user-custom toplists that use these metrics.
   - **Toplist Service**:
     - Loads user-custom toplist configurations from TimescaleDB.
     - Caches configurations in Redis for fast access.
     - Queries Redis ZSETs to compute rankings for user toplists.
     - Applies filters (min volume, price range, etc.) before ranking.
     - Publishes updates to `toplists.updated` pub/sub channel.
   - **Expiration**:
     - Redis keys can have TTL, or workers can remove stale symbols (optional for MVP).
   - **Consumption**:
     - API Service queries ZSETs (`ZREVRANGE`) to return top N symbols.
     - WebSocket Gateway subscribes to `toplists.updated` and broadcasts to clients.

---

## Partitioning & Scaling

### Partitioning Strategy
- **Symbol-based partitioning**: Consistent hash of symbol → partition assignment.
- **Formula**: `partition = hash(symbol) % num_partitions`
- **Guarantees**: All ticks for a symbol go to same partition (ordering preserved).
- **Redis Streams**: Use `XADD` with partition key, or multiple streams (one per partition).
- **Partition count**: Start with 8-16 partitions, scale to 32+ as needed.

### Worker Scaling
- **Horizontal scaling**: Add/remove worker instances, consumer groups rebalance automatically.
- **Autoscaling triggers** (HPA):
  - CPU utilization > 70% (scale up)
  - CPU utilization < 30% (scale down, with cooldown)
  - Consumer lag > 1000 messages (scale up)
  - Scan cycle time > 900ms (p95, scale up)
- **Min replicas**: 2 (for availability)
- **Max replicas**: 20 (adjust based on symbol count)
- **Scaling speed**: Scale up fast (30s), scale down slow (5min cooldown).

### State Sharding
- Each worker maintains local state for assigned symbols only.
- **Consumer groups**: Redis Streams consumer groups handle partition assignment automatically.
- **Rebalancing**: On worker join/leave, consumer group rebalances partitions.
- **State rehydration**: On rebalance, worker loads state from:
  - Redis `livebar:{symbol}` (recent live bars)
  - TimescaleDB (last N finalized bars)
  - Redis `ind:{symbol}` (recent indicators)
- **Heartbeat**: Workers report health every 10s (readiness probe).

---

## Latency & Performance Targets

- **Scan cycle target**: 1 second or less. Aim for <800ms average with 200ms safety slack.
- **Per-worker budget**:
  - Let S = symbols_per_worker, R = rules_per_symbol.
  - CPU cost ≈ S * R * simple comparisons per cycle.
  - Keep R small or compile rules into optimized ops; use vectorized evaluation if needed.
- **I/O restrictions**:
  - No synchronous DB calls in scan loop.
  - No Redis GETs in the loop: ingest updates local memory via pubsub.
  - Bulk updates and writes are async.

---

## Reliability & Fault Tolerance

### Stream Durability
- **Redis Streams**: Enable AOF (Append-Only File) persistence for durability.
- **Replay capability**: Consumer groups track last read position, allow replay on failure.
- **At-least-once delivery**: Idempotent processing handles duplicates.

### Failure Modes & Recovery

#### Market Data Ingest Failure
- **Detection**: Connection timeout, heartbeat failure.
- **Recovery**: Exponential backoff reconnection (1s, 2s, 4s, 8s, max 60s).
- **Fallback**: Switch to alternate provider if configured.
- **Degraded mode**: Mark system as degraded, notify users.

#### Bar Aggregator Failure
- **Recovery**: Restart service, consumer group resumes from last position.
- **State**: No persistent state (rebuilds from tick stream).
- **Impact**: Temporary loss of live bar updates (workers continue with last known state).

#### Indicator Engine Failure
- **Recovery**: Restart service, consumer group resumes.
- **State**: Rebuilds indicator windows from finalized bars (may take minutes).
- **Impact**: Workers use stale indicators until engine catches up.

#### Scanner Worker Failure
- **Recovery**: Kubernetes restarts pod, consumer group rebalances.
- **State rehydration**:
  1. Load recent live bars from Redis (last 5 minutes).
  2. Load last 60 finalized bars from TimescaleDB.
  3. Load recent indicators from Redis.
  4. Resume processing (may miss alerts during rehydration).
- **Readiness probe**: Worker reports ready only after rehydration completes.

#### Alert Service Failure
- **Recovery**: Restart service, consumer group resumes from last position.
- **Deduplication**: Idempotency keys prevent duplicate alerts.
- **Impact**: Temporary alert delivery delay (alerts queued in stream).

#### WebSocket Gateway Failure
- **Recovery**: Restart service, reconnect WebSocket clients.
- **Client reconnection**: Clients implement exponential backoff.
- **Missed alerts**: Clients can query alert history API for missed period.

#### Database Failure (TimescaleDB)
- **Read replicas**: Use read replicas for queries (alert history API).
- **Write queue**: Buffer writes in memory, retry with exponential backoff.
- **Degraded mode**: Continue processing, queue writes, resume when DB recovers.

#### Redis Failure
- **High availability**: Redis Sentinel or Redis Cluster for failover.
- **Impact**: Critical (all hot state in Redis).
- **Recovery**: Failover to replica (automatic with Sentinel).
- **Data loss**: Acceptable for hot state (TTL-based), reload from TimescaleDB.

### Circuit Breakers
- **External dependencies**: Market data providers, databases.
- **Threshold**: 5 failures in 30s → open circuit.
- **Recovery**: Half-open after 60s, close on success.
- **Fallback**: Degraded mode, cached data, error responses.

### Idempotency
- **Alert idempotency**: Idempotency keys prevent duplicate alerts.
- **Bar writes**: Idempotent (upsert by symbol + timestamp).
- **Rule updates**: Versioned, atomic updates.

---

## Security & Compliance

### Network Security
- **TLS**: All external endpoints use TLS 1.3 (API, WebSocket).
- **mTLS**: Internal service-to-service communication (optional for MVP, use service mesh).
- **Network policies**: Kubernetes network policies restrict inter-pod communication.
- **Firewall**: Restrict database access to application pods only.

### Authentication & Authorization
- **API authentication**: JWT tokens (issued by auth service or API service).
- **WebSocket authentication**: JWT token in connection handshake.
- **Token validation**: Validate JWT signature, expiration, issuer.
- **Authorization**: Role-based access control (RBAC) for API endpoints.
- **Multi-tenancy**: **MVP decision**: Single-tenant (add multi-tenancy post-MVP with migration path).

### Data Protection
- **Encryption at rest**: Database encryption (TimescaleDB, Redis AOF).
- **Encryption in transit**: TLS for all connections.
- **Secrets management**: Kubernetes Secrets or external secret manager (Vault).
- **PII handling**: No PII in logs, alert history (user IDs only).

### Audit & Compliance
- **Audit logs**: All rule changes, alert issuances, user actions logged.
- **Log retention**: 90 days for audit logs, 30 days for application logs.
- **Data retention**:
  - Live bars: 5 minutes (Redis TTL)
  - Indicators: 10 minutes (Redis TTL)
  - Finalized bars: 1 year (TimescaleDB)
  - Alert history: 1 year (TimescaleDB)
  - Archives: > 1 year to S3 (post-MVP)
- **Data deletion**: Automated deletion after retention period.
- **GDPR**: Right to deletion, data export (post-MVP feature).

---

## Service Communication Patterns

### Event-Driven (Data Flow)
- **Redis Streams**: Ticks, finalized bars, alerts (durable, ordered).
- **Redis Pub/Sub**: Real-time notifications (indicators, rules) (ephemeral, fast).
- **Pattern**: Producer → Stream → Consumer Group → Processing.

### Synchronous (Control Plane)
- **REST API**: API Service endpoints (rule management, alert history).
- **Health checks**: HTTP `/health`, `/ready` endpoints.
- **Metrics**: Prometheus `/metrics` endpoint.

### Service Discovery
- **Kubernetes**: DNS-based service discovery (`service-name.namespace.svc.cluster.local`).
- **No service mesh for MVP**: Use Kubernetes native networking.
- **Post-MVP**: Consider Istio/Linkerd for advanced features.

---

## Observability & Monitoring

### Metrics (Prometheus)

#### Critical SLO Metrics
- `scan_cycle_seconds` (histogram): Scan loop duration (p50, p95, p99)
  - **SLO**: p95 < 800ms
- `tick_processing_latency_seconds` (histogram): Tick ingestion to bar update
- `alert_delivery_latency_seconds` (histogram): Alert generation to WebSocket delivery
- `worker_queue_depth` (gauge): Pending messages per worker
- `consumer_lag` (gauge): Messages behind in stream per consumer group

#### Service Health Metrics
- `http_requests_total` (counter): API request count by endpoint, status
- `websocket_connections` (gauge): Active WebSocket connections
- `alerts_emitted_total` (counter): Alerts generated by rule, symbol
- `alerts_delivered_total` (counter): Alerts delivered to clients
- `errors_total` (counter): Errors by service, type

#### Resource Metrics
- CPU, memory usage per service
- Database connection pool usage
- Redis memory usage

### Tracing (OpenTelemetry)
- **Sampling rate**: 1% of requests (adjustable per service).
- **Key spans**:
  - Tick ingestion → bar update → indicator → worker → alert
  - Alert generation → alert service → WebSocket delivery
- **Trace context**: Propagated via headers, correlation IDs in logs.

### Logging
- **Structured logs**: JSON format with correlation IDs.
- **Log levels**: ERROR, WARN, INFO, DEBUG.
- **Aggregation**: Loki for log aggregation and querying.
- **Retention**: 30 days for application logs, 90 days for audit logs.

### Alerting (Prometheus Alerts)
- Scan cycle time > 900ms (p95)
- Consumer lag > 5000 messages
- Worker CPU > 85%
- Database connection pool > 80% utilized
- Redis memory > 80% utilized
- Service down (health check failure)

---

## Operational Playbook

### Daily Operations
- Monitor dashboards: scan cycle times, consumer lag, error rates.
- Review alert delivery metrics: latency, success rate.
- Check database and Redis resource usage.

### Scaling Operations
- **Scale up workers**: When scan cycle > 800ms (p95) or consumer lag > 1000.
- **Scale down workers**: When CPU < 30% for 5 minutes (cooldown).
- **Database scaling**: Monitor connection pool, query latency; add read replicas if needed.

### Incident Response

#### High Scan Cycle Time
1. Check worker CPU/memory usage.
2. Check consumer lag (backlog of messages).
3. Scale up workers (increase replicas).
4. If still high: reduce symbols per worker, optimize rules.

#### Data Provider Outage
1. Ingest service detects connection failure.
2. Attempts reconnection with backoff.
3. After 5 failures: switches to fallback provider (if configured).
4. Marks system as degraded, notifies users via API status endpoint.
5. When primary recovers: switches back, clears degraded flag.

#### Worker Failure
1. Kubernetes restarts pod automatically.
2. Worker rehydrates state (loads from Redis + TimescaleDB).
3. Consumer group rebalances partitions.
4. Monitor for duplicate alerts (idempotency keys prevent issues).
5. Check logs for rehydration errors.

#### Database Failure
1. Check database health (connection errors in logs).
2. If read replica available: switch reads to replica.
3. Queue writes in memory, retry with backoff.
4. When database recovers: drain write queue.
5. Verify data consistency.

### Performance Tuning
- **Worker buffer sizes**: Tune tick channel buffer based on tick rate.
- **Batch sizes**: Tune database batch insert sizes (balance latency vs throughput).
- **Connection pools**: Tune database and Redis connection pool sizes.
- **Rule optimization**: Profile rule evaluation, optimize hot paths.

---

## API Design

### Versioning
- **URL versioning**: `/api/v1/...`, `/api/v2/...`
- **Breaking changes**: New version, deprecate old version with 6-month notice.
- **Non-breaking changes**: Add fields, extend endpoints (backward compatible).

### Rate Limiting
- **Per-user limits**: 100 requests/minute for API endpoints.
- **Per-IP limits**: 1000 requests/minute (DDoS protection).
- **WebSocket**: No rate limit (connection-based).

### Endpoints (MVP)

#### Rules
- `GET /api/v1/rules` - List rules (paginated)
- `GET /api/v1/rules/:id` - Get rule details
- `POST /api/v1/rules` - Create rule
- `PUT /api/v1/rules/:id` - Update rule
- `DELETE /api/v1/rules/:id` - Delete rule
- `POST /api/v1/rules/:id/validate` - Validate rule syntax

#### Alerts
- `GET /api/v1/alerts` - List alert history (paginated, filtered)
- `GET /api/v1/alerts/:id` - Get alert details

#### Symbols
- `GET /api/v1/symbols` - List available symbols
- `GET /api/v1/symbols/:symbol` - Get symbol info

#### System
- `GET /health` - Health check
- `GET /ready` - Readiness probe
- `GET /metrics` - Prometheus metrics

#### Toplists
- `GET /api/v1/toplists` - List available toplist types (system + user-custom)
- `GET /api/v1/toplists/system/:type?limit=10&offset=0` - Get system toplist (e.g., `gainers_1m`, `losers_1m`, `volume_day`)
- `GET /api/v1/toplists/user` - List user's custom toplists
- `POST /api/v1/toplists/user` - Create user-custom toplist
- `GET /api/v1/toplists/user/:id` - Get user-custom toplist details and rankings
- `PUT /api/v1/toplists/user/:id` - Update user-custom toplist configuration
- `DELETE /api/v1/toplists/user/:id` - Delete user-custom toplist
- `GET /api/v1/toplists/user/:id/rankings?limit=50&offset=0` - Get toplist rankings with filters


---

## Toplist Feature — Detailed Specification

### Overview
The Toplist feature provides real-time ranking and monitoring of stocks based on various metrics, similar to chartswatcher.com. Users can view system-wide toplists (predefined) or create custom toplists with their own criteria, filters, and display preferences.

### Key Features

1. **System Toplists** (Predefined)
   - **Gainers**: Stocks with highest price change % (1m, 5m, 15m, 1h, 1d)
   - **Losers**: Stocks with lowest price change % (1m, 5m, 15m, 1h, 1d)
   - **Volume Leaders**: Stocks with highest trading volume (1m, 5m, 15m, 1h, 1d)
   - **RSI Extremes**: Stocks with highest/lowest RSI values
   - **Relative Volume**: Stocks with highest relative volume ratios
   - **VWAP Distance**: Stocks furthest from VWAP (above/below)

2. **User-Custom Toplists**
   - Users can create custom toplists with:
     - **Metric**: Any available metric (change_pct, volume, rsi, relative_volume, vwap_dist, etc.)
     - **Time Window**: 1m, 5m, 15m, 1h, 1d
     - **Sort Order**: Ascending or descending
     - **Filters**:
       - Minimum daily volume threshold
       - Price range (min/max)
       - Exchange filter (NYSE, NASDAQ, etc.)
       - Market cap range
       - Sector/industry filters (future enhancement)
     - **Display Columns**: Customizable columns to show (price, change%, volume, RSI, etc.)
     - **Color Schemes**: Custom color coding for different value ranges

3. **Real-Time Updates**
   - Toplists update every 1 second (aligned with scan cycle)
   - Updates delivered via WebSocket for subscribed clients
   - REST API provides snapshot queries
   - Efficient updates using Redis ZSETs (sorted sets)

### Data Flow

1. **Toplist Update Generation**
   - Scanner Workers calculate metrics for assigned symbols
   - Workers update Redis ZSETs:
     - System toplists: `toplist:{metric}:{window}` (e.g., `toplist:change_pct:1m`)
     - User toplists: `toplist:user:{user_id}:{toplist_id}`
   - Workers publish update notification to `toplists.updated` pub/sub channel

2. **Toplist Service Processing**
   - Toplist Service subscribes to `toplists.updated` channel
   - For user-custom toplists:
     - Loads toplist configuration from cache (Redis) or database
     - Queries relevant Redis ZSETs
     - Applies filters (min volume, price range, etc.)
     - Computes final rankings
     - Publishes update to `toplists.updated` channel with toplist ID

3. **WebSocket Delivery**
   - WebSocket Gateway subscribes to `toplists.updated` channel
   - For each update:
     - Checks which clients are subscribed to the toplist
     - Broadcasts update message to subscribed clients
   - Message format:
     ```json
     {
       "type": "toplist_update",
       "data": {
         "toplist_id": "user_123_custom_1",
         "toplist_type": "user",
         "rankings": [
           {"symbol": "AAPL", "rank": 1, "value": 2.5, "metadata": {...}},
           {"symbol": "MSFT", "rank": 2, "value": 2.3, "metadata": {...}}
         ],
         "timestamp": "2024-01-01T12:00:00Z"
       }
     }
     ```

4. **API Queries**
   - Clients can query toplist rankings via REST API
   - API queries Redis ZSETs directly (`ZREVRANGE` for top N)
   - Applies filters server-side
   - Returns paginated results

### Toplist Configuration Schema

```json
{
  "id": "user_123_custom_1",
  "user_id": "user_123",
  "name": "High Volume Gainers",
  "description": "Stocks with >2% gain and >1M volume",
  "metric": "change_pct",
  "time_window": "5m",
  "sort_order": "desc",
  "filters": {
    "min_volume": 1000000,
    "min_change_pct": 2.0,
    "price_min": 10.0,
    "price_max": 500.0
  },
  "columns": ["symbol", "price", "change_pct", "volume", "rsi"],
  "color_scheme": {
    "positive": "#00ff00",
    "negative": "#ff0000",
    "neutral": "#ffffff"
  },
  "enabled": true,
  "created_at": "2024-01-01T10:00:00Z",
  "updated_at": "2024-01-01T10:00:00Z"
}
```

### Storage Strategy

- **TimescaleDB**: Stores user-custom toplist configurations (persistent)
- **Redis**: 
  - ZSETs for rankings (hot data, TTL: 5 minutes)
  - Hash for toplist config cache (TTL: 1 hour)
  - Pub/sub channel for real-time updates

### Performance Considerations

- **Batch Updates**: Workers batch ZSET updates to minimize Redis round-trips
- **Caching**: Toplist configurations cached in Redis to avoid DB queries
- **Lazy Evaluation**: User toplists computed on-demand or on update notification
- **Pagination**: API endpoints support limit/offset for large result sets
- **WebSocket Throttling**: Updates throttled to max 1 update/second per toplist

### WebSocket Protocol Extensions

New message types for toplist subscriptions:

**Client → Server:**
```json
{
  "type": "subscribe_toplist",
  "toplist_id": "user_123_custom_1",
  "toplist_type": "user" // or "system"
}
```

```json
{
  "type": "unsubscribe_toplist",
  "toplist_id": "user_123_custom_1"
}
```

**Server → Client:**
```json
{
  "type": "toplist_update",
  "data": {
    "toplist_id": "user_123_custom_1",
    "rankings": [...],
    "timestamp": "..."
  }
}
```

---

## Architecture Decisions & Rationale

### 1. Redis Streams vs Kafka
**Decision**: Redis Streams for MVP
**Rationale**: Simpler operations, sufficient for MVP scale, migration path to Kafka available via abstraction layer.

### 2. Separate Alert Service and WebSocket Gateway
**Decision**: Separate services
**Rationale**: Better scalability, independent scaling, clearer separation of concerns.

### 3. Storage: TimescaleDB only (defer ClickHouse)
**Decision**: Single database for MVP
**Rationale**: Simpler operations, sufficient for MVP scale, add ClickHouse post-MVP for analytics.

### 4. Rule Management: Database + Cache
**Decision**: TimescaleDB + Redis cache
**Rationale**: Persistent storage with fast access, supports complex queries, easy rule updates.

### 5. Worker Data Ingestion: Hybrid
**Decision**: Ticks + complex indicators from engine, simple metrics computed locally
**Rationale**: Balance between CPU usage and data freshness, reduces dependency on Indicator Engine for hot path.

### 6. Alert Deduplication: Hybrid
**Decision**: Per-worker cooldown (fast) + Alert Service idempotency (safety)
**Rationale**: Fast path for common case, safety net for edge cases (rebalancing).

### 7. Multi-tenancy: Single-tenant MVP
**Decision**: Single-tenant for MVP
**Rationale**: Simpler implementation, add multi-tenancy post-MVP with migration path.

### 8. Service Communication: Event-driven + REST
**Decision**: Events for data flow, REST for control
**Rationale**: Best of both worlds, async for performance, sync for control operations.

---

## Future Enhancements (Post-MVP)

1. **Kafka migration**: Replace Redis Streams with Kafka for higher throughput.
2. **ClickHouse integration**: Add for analytics and long-term alert history.
3. **Multi-tenancy**: Add tenant isolation, per-tenant resource limits.
4. **Service mesh**: Add Istio/Linkerd for advanced traffic management.
5. **GraphQL API**: Add for complex queries.
6. **Advanced indicators**: Machine learning indicators, custom indicator plugins.
7. **Alert channels**: Email, SMS, push notifications.
8. **Rule templates**: Pre-built rule templates, rule marketplace.

