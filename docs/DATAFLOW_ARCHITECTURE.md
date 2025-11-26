# Data Flow Architecture with Implementation Status

This document describes the complete data flow between all services, Redis, and TimescaleDB, with implementation status tags for each component.

## Legend

- ✅ **IMPLEMENTED** - Fully implemented and functional
- ⚠️ **PARTIAL** - Partially implemented (some features missing)
- ❌ **NOT IMPLEMENTED** - Not yet implemented
- 🔄 **MVP** - Minimal viable implementation (basic functionality works, enhancements pending)

---

## 1. Redis Streams

### Stream: `ticks`
- **Publisher**: ✅ **IMPLEMENTED** - Ingest Service (`cmd/ingest/main.go`)
  - Consumes ticks from market data provider
  - Publishes to `ticks` stream (batched, optionally partitioned)
  - **Note**: ⚠️ Only mock provider implemented; real providers (Alpaca, Polygon.io, Databento) are ❌ NOT IMPLEMENTED
  
- **Consumers**:
  1. ✅ **IMPLEMENTED** - Bars Service (`cmd/bars/main.go`)
     - Consumer Group: `bars-aggregator`
     - Aggregates ticks into 1-minute bars
  2. ✅ **IMPLEMENTED** - Scanner Service (`cmd/scanner/main.go`)
     - Consumer Group: `scanner-group`
     - Updates symbol state with latest tick data

### Stream: `bars.finalized`
- **Publisher**: ✅ **IMPLEMENTED** - Bars Service (`internal/bars/publisher.go`)
  - Publishes finalized 1-minute bars (batched)
  - Also writes to TimescaleDB ✅ **IMPLEMENTED**
  
- **Consumers**:
  1. ✅ **IMPLEMENTED** - Indicator Service (`cmd/indicator/main.go`)
     - Consumer Group: `indicator-engine`
     - Computes indicators (RSI, EMA, SMA, VWAP, etc.)
  2. ✅ **IMPLEMENTED** - Scanner Service (`cmd/scanner/main.go`)
     - Consumer Group: `scanner-group`
     - Updates state with finalized bars

### Stream: `alerts`
- **Publisher**: ✅ **IMPLEMENTED** - Scanner Service (`internal/scanner/alert_emitter.go`)
  - Publishes alerts when rules trigger
  - Also publishes to pub/sub channel `alerts` (for real-time)
  
- **Consumer**: ✅ **IMPLEMENTED** - Alert Service (`cmd/alert/main.go`)
  - Consumer Group: `alert-service`
  - Processes: deduplication → user filtering → cooldown → persistence → routing
  - **Note**: User filtering is 🔄 **MVP** (structure ready, but basic implementation)

### Stream: `alerts.filtered`
- **Publisher**: ✅ **IMPLEMENTED** - Alert Service (`internal/alert/router.go`)
  - Publishes filtered alerts after processing
  
- **Consumer**: ✅ **IMPLEMENTED** - WebSocket Gateway (`cmd/ws_gateway/main.go`)
  - Consumer Group: `ws-gateway`
  - Broadcasts to connected WebSocket clients

---

## 2. Redis Pub/Sub Channels

### Channel: `indicators.updated`
- **Publisher**: ✅ **IMPLEMENTED** - Indicator Service (`internal/indicator/publisher.go`)
  - Publishes when indicator values are computed/updated
  - Message: `{symbol, timestamp}`
  
- **Subscriber**: ✅ **IMPLEMENTED** - Scanner Service (`internal/scanner/indicator_consumer.go`)
  - Fetches full indicator data from Redis key `ind:{symbol}`
  - Updates symbol state

### Channel: `toplists.updated`
- **Publisher**: ✅ **IMPLEMENTED** - Toplist Updater (via Indicator Service or Scanner Service)
  - Publishes when toplist rankings are updated
  - Message: `{toplist_id, toplist_type, timestamp}`
  
- **Subscriber**: ✅ **IMPLEMENTED** - WebSocket Gateway (`internal/wsgateway/hub.go`)
  - Broadcasts toplist updates to subscribed WebSocket clients

### Channel: `alerts`
- **Publisher**: ✅ **IMPLEMENTED** - Scanner Service (`internal/scanner/alert_emitter.go`)
  - Publishes alerts for real-time delivery
  - **Note**: ❌ Not actively subscribed (alerts consumed via Streams instead)

### Channel: `rules.updated`
- **Publisher**: ✅ **IMPLEMENTED** - Rules Sync Service (`internal/rules/sync.go`)
  - Publishes when rules are synced from DB to Redis
  - **Note**: ❌ Currently no subscribers (future: scanner workers could reload rules)

---

## 3. Redis ZSETs (Sorted Sets) - Toplists

### System Toplists
- **Key Pattern**: `toplist:system:{metric}:{window}`
- **Examples**:
  - `toplist:system:change_pct:1m` - 1-minute gainers/losers
  - `toplist:system:volume:1m` - 1-minute volume leaders
  - `toplist:system:rsi:1m` - RSI rankings
  - `toplist:system:relative_volume:5m` - Relative volume rankings
  - `toplist:system:vwap_dist:5m` - VWAP distance rankings

- **Writers**:
  - ✅ **IMPLEMENTED** - Scanner Service (price change, volume)
  - ✅ **IMPLEMENTED** - Indicator Service (RSI, relative volume, VWAP distance)
  
- **Readers**:
  - ✅ **IMPLEMENTED** - API Service (via Toplist Service)
  - ✅ **IMPLEMENTED** - WebSocket Gateway (via pub/sub notifications)

### User Toplists
- **Key Pattern**: `toplist:user:{userID}:{toplistID}`
- **Writers**: ✅ **IMPLEMENTED** - Scanner/Indicator Services (when user rules trigger)
- **Readers**: ✅ **IMPLEMENTED** - API Service, WebSocket Gateway

---

## 4. Redis Keys (Key-Value)

### Indicator Storage
- **Key Pattern**: `ind:{symbol}`
- **Writer**: ✅ **IMPLEMENTED** - Indicator Service
- **Reader**: ✅ **IMPLEMENTED** - Scanner Service (via pub/sub notification)
- **TTL**: 10 minutes
- **Value**: `{symbol, timestamp, values: {rsi_14, ema_20, ...}}`

### Live Bar Snapshots
- **Key Pattern**: `livebar:{symbol}`
- **Writer**: ✅ **IMPLEMENTED** - Bars Service
- **Reader**: ⚠️ **PARTIAL** - API Service (optional, not fully exposed)
- **TTL**: 5 minutes
- **Value**: Current live bar (not yet finalized)

### Rule Cache
- **Key Pattern**: `rule:{ruleID}`
- **Writer**: ✅ **IMPLEMENTED** - Rules Sync Service
- **Reader**: ✅ **IMPLEMENTED** - Scanner Service
- **Value**: Compiled rule definition

### Toplist Config Cache
- **Key Pattern**: `toplist:config:{toplistID}`
- **Writer**: ✅ **IMPLEMENTED** - API Service (Toplist Service)
- **Reader**: ✅ **IMPLEMENTED** - API Service (Toplist Service)
- **TTL**: 1 hour
- **Value**: Toplist configuration

---

## 5. TimescaleDB Tables

### Table: `bars_1m`
- **Status**: ✅ **IMPLEMENTED** (Migration: `001_create_bars_table.sql`)
- **Writers**:
  - ✅ **IMPLEMENTED** - Bars Service (async batch writes when bars are finalized)
- **Readers**:
  - ✅ **IMPLEMENTED** - Scanner Service (rehydration on startup - reads latest 200 bars per symbol)
  - ⚠️ **PARTIAL** - API Service (⚠️ TODO: bars data access endpoint not implemented - see `cmd/api/main.go:153`)

### Table: `alert_history`
- **Status**: ✅ **IMPLEMENTED** (Migration: `002_create_alert_history_table.sql`)
- **Writers**:
  - ✅ **IMPLEMENTED** - Alert Service (async batch writes after deduplication/filtering/cooldown)
- **Readers**:
  - ✅ **IMPLEMENTED** - API Service (queries for alert history with filtering)

### Table: `rules`
- **Status**: ✅ **IMPLEMENTED** (Migration: `003_create_rules_table.sql`)
- **Writers**:
  - ✅ **IMPLEMENTED** - API Service (create/update/delete rules)
- **Readers**:
  - ✅ **IMPLEMENTED** - API Service (read rules)
  - ✅ **IMPLEMENTED** - Rules Sync Service (syncs rules from DB to Redis cache)

### Table: `toplist_configs`
- **Status**: ✅ **IMPLEMENTED** (Migration: `004_create_toplist_configs_table.sql`)
- **Writers**:
  - ✅ **IMPLEMENTED** - API Service (create/update/delete user toplists)
- **Readers**:
  - ✅ **IMPLEMENTED** - API Service (read toplist configurations)

---

## Complete Data Flow Diagram (with Implementation Status)

```
┌─────────────────┐
│ Market Data     │
│ Provider        │
└────────┬────────┘
         │ Ticks
         ▼
┌─────────────────┐
│ Ingest Service  │ ✅ IMPLEMENTED
│                 │ ⚠️ PARTIAL: Mock provider only
│                 │ ❌ Real providers (Alpaca/Polygon/Databento) not implemented
└────────┬────────┘
         │ Publishes
         ▼
    ┌─────────┐
    │ Stream: │
    │  ticks  │
    └────┬────┘
         │
    ┌────┴─────────────────────────────┐
    │                                   │
    ▼                                   ▼
┌──────────────┐              ┌─────────────────┐
│ Bars Service │ ✅          │ Scanner Service │ ✅
│ (bars-       │ IMPLEMENTED │ (scanner-group)│ IMPLEMENTED
│ aggregator)  │             └────────┬────────┘
└──────┬───────┘                     │
       │                              │ Updates State
       │ Aggregates                   │
       │                              │
       │ Writes (async batch)         │
       ▼                              │
┌─────────────────┐                  │
│ TimescaleDB     │ ✅                │
│ bars_1m table   │ IMPLEMENTED       │
└─────────────────┘                  │
       │                              │
       │ Publishes                    │
       ▼                              │
┌──────────────┐                     │
│ Stream:      │                     │
│bars.finalized│                     │
└──────┬───────┘                     │
       │                              │
    ┌──┴───────────────────────────────┴──┐
    │                                       │
    ▼                                       ▼
┌─────────────────┐              ┌─────────────────┐
│ Indicator       │ ✅          │ Scanner Service │ ✅
│ Service         │ IMPLEMENTED  │ (bar handler)   │ IMPLEMENTED
│ (indicator-     │             └─────────────────┘
│  engine)        │
└────────┬────────┘
         │ Computes Indicators
         │
         ▼
    ┌────────────┐
    │ Redis Key: │
    │ ind:{sym}  │
    └─────┬──────┘
          │
          │ Publishes
          ▼
    ┌──────────────────┐
    │ Pub/Sub:         │
    │indicators.updated│
    └─────────┬─────────┘
              │
              ▼
    ┌─────────────────┐
    │ Scanner Service │ ✅ IMPLEMENTED
    │ (indicator      │
    │  consumer)      │
    └─────────┬───────┘
              │
              │ Evaluates Rules
              │
              ▼
    ┌─────────────────┐
    │ Scanner Service │ ✅ IMPLEMENTED
    │ (alert emitter) │
    └─────────┬───────┘
              │
    ┌─────────┴──────────┐
    │                    │
    ▼                    ▼
┌──────────┐    ┌──────────────────┐
│ Stream:  │    │ Pub/Sub: alerts  │
│  alerts  │    │ (real-time)      │
└────┬─────┘    └──────────────────┘
     │
     ▼
┌─────────────────┐
│ Alert Service   │ ✅ IMPLEMENTED
│ (alert-service) │ 🔄 MVP: User filtering basic
└─────────┬───────┘
          │
          │ Deduplicates, Filters, Cooldown
          │
          │ Writes (async batch)
          ▼
    ┌─────────────────┐
    │ TimescaleDB     │ ✅ IMPLEMENTED
    │ alert_history   │
    │ table           │
    └─────────────────┘
          │
          │ Publishes
          ▼
    ┌──────────────────┐
    │ Stream:          │
    │ alerts.filtered  │
    └─────────┬─────────┘
              │
              ▼
    ┌─────────────────┐
    │ WebSocket       │ ✅ IMPLEMENTED
    │ Gateway         │
    │ (ws-gateway)    │
    └─────────────────┘

┌───────────────────────────────────────────┐
│ REHYDRATION FLOW (Scanner Startup)         │
└───────────────────────────────────────────┘
              │
              │ Reads latest bars
              ▼
    ┌─────────────────┐
    │ TimescaleDB     │ ✅ IMPLEMENTED
    │ bars_1m table   │
    │ (GetLatestBars) │
    └─────────┬───────┘
              │
              │ + Redis indicators
              │ ⚠️ PARTIAL: Symbol discovery not implemented
              │    (must provide symbols via config)
              ▼
    ┌─────────────────┐
    │ Scanner Service │ ✅ IMPLEMENTED
    │ (Rehydrator)    │ ⚠️ PARTIAL: Symbol discovery missing
    └─────────────────┘

┌───────────────────────────────────────────┐
│ API SERVICE DATA FLOW                     │
└───────────────────────────────────────────┘
              │
    ┌─────────┴──────────┐
    │                   │
    ▼                   ▼
┌──────────────┐  ┌─────────────────┐
│ TimescaleDB  │  │ TimescaleDB     │
│ rules table  │  │ alert_history   │ ✅ IMPLEMENTED
│ (CRUD)       │  │ (Read queries)  │
└──────┬───────┘  └─────────────────┘
       │
       │ ⚠️ PARTIAL: bars_1m read not exposed
       │    (TODO in cmd/api/main.go:153)
       │
       │ Syncs to Redis
       ▼
┌──────────────┐
│ Redis Cache  │
│ rule:{id}    │
└──────┬───────┘
       │
       │ Publishes
       ▼
┌──────────────────┐
│ Pub/Sub:         │ ✅ IMPLEMENTED
│ rules.updated    │ ❌ No subscribers yet
│ (future use)     │
└──────────────────┘

┌───────────────────────────────────────────┐
│ TOPLIST FLOW                              │
└───────────────────────────────────────────┘
              │
    ┌─────────┴──────────┐
    │                   │
    ▼                   ▼
┌──────────────┐  ┌─────────────────┐
│ Scanner/     │  │ Indicator      │ ✅ IMPLEMENTED
│ Indicator    │  │ Service        │
│ Services     │  │                │
└──────┬───────┘  └────────┬────────┘
       │                   │
       │ Updates            │
       ▼                   ▼
┌──────────────┐  ┌─────────────────┐
│ Redis ZSET   │  │ Redis ZSET      │ ✅ IMPLEMENTED
│ toplist:*    │  │ toplist:*       │
└──────┬───────┘  └────────┬────────┘
       │                   │
       └─────────┬─────────┘
                 │
                 │ Publishes
                 ▼
         ┌──────────────────┐
         │ Pub/Sub:         │ ✅ IMPLEMENTED
         │toplists.updated  │
         └─────────┬─────────┘
                   │
                   ▼
         ┌─────────────────┐
         │ WebSocket       │ ✅ IMPLEMENTED
         │ Gateway         │
         └─────────────────┘
```

---

## Implementation Status Summary

### Services

| Service | Status | Notes |
|---------|--------|-------|
| **Ingest Service** | ⚠️ **PARTIAL** | Core implemented, but only mock provider. Real providers (Alpaca, Polygon.io, Databento) not implemented |
| **Bars Service** | ✅ **IMPLEMENTED** | Fully functional with TimescaleDB integration |
| **Indicator Service** | ✅ **IMPLEMENTED** | All indicators implemented and tested |
| **Scanner Service** | ✅ **IMPLEMENTED** | Core functionality complete. Symbol discovery not implemented (must provide via config) |
| **Alert Service** | ✅ **IMPLEMENTED** | Fully functional. User filtering is MVP (basic implementation) |
| **WebSocket Gateway** | ✅ **IMPLEMENTED** | Fully functional with toplist support |
| **API Service** | ⚠️ **PARTIAL** | Most endpoints implemented. Missing: bars data access endpoint (TODO) |

### Database Tables

| Table | Status | Notes |
|-------|--------|-------|
| `bars_1m` | ✅ **IMPLEMENTED** | Fully functional, used for rehydration |
| `alert_history` | ✅ **IMPLEMENTED** | Fully functional, used for alert queries |
| `rules` | ✅ **IMPLEMENTED** | Fully functional, synced to Redis |
| `toplist_configs` | ✅ **IMPLEMENTED** | Fully functional, used for user toplists |

### Redis Components

| Component | Status | Notes |
|-----------|--------|-------|
| Streams (ticks, bars.finalized, alerts, alerts.filtered) | ✅ **IMPLEMENTED** | All streams functional |
| Pub/Sub Channels | ✅ **IMPLEMENTED** | All channels functional. `rules.updated` has no subscribers yet |
| ZSETs (Toplists) | ✅ **IMPLEMENTED** | System and user toplists functional |
| Keys (indicators, live bars, rules, toplist configs) | ✅ **IMPLEMENTED** | All key patterns functional |

### Known Limitations / TODOs

1. ⚠️ **Ingest Service**: Real market data providers not implemented (only mock provider)
2. ⚠️ **API Service**: Bars data access endpoint not implemented (TODO in `cmd/api/main.go:153`)
3. ⚠️ **Scanner Service**: Symbol discovery not implemented (must provide symbols via config)
4. 🔄 **Alert Service**: User filtering is MVP (basic structure, needs enhancement)
5. ❌ **Rules Pub/Sub**: `rules.updated` channel has no subscribers (future enhancement)

---

## Data Persistence Strategy

- **Hot Data (Redis)**:
  - Live ticks (streams) ✅
  - Recent bars (streams + live bars in keys) ✅
  - Current indicators (keys with TTL) ✅
  - Active rules (cached in Redis) ✅
  - Toplist rankings (ZSETs) ✅

- **Cold Data (TimescaleDB)**:
  - Historical bars (`bars_1m` hypertable) ✅
  - Alert history (`alert_history` hypertable) ✅
  - Rule definitions (`rules` table) ✅
  - Toplist configurations (`toplist_configs` table) ✅

- **Hybrid**:
  - Scanner rehydration: reads from TimescaleDB + Redis on startup ✅
  - Rules: stored in DB, cached in Redis, synced via pub/sub ✅

This design separates real-time processing (Redis) from historical storage (TimescaleDB) for optimal performance and scalability.

