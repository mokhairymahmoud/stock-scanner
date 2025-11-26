# Architecture Diagram Comparison

This document compares the provided architecture diagram with the actual codebase implementation.

## Overall Assessment

**Status**: ⚠️ **MOSTLY MATCHES** with some important differences

The diagram captures the overall data flow correctly, but there are several key differences in service naming, architecture, and some missing data flows.

---

## Key Differences

### 1. ❌ Service Naming Differences

| Diagram | Actual Codebase | Status |
|---------|----------------|--------|
| "Market Data Ingestion" | **Ingest Service** | ✅ Same functionality, different name |
| "Bar Aggregator (Apache Flink)" | **Bars Service** (Go service, NOT Apache Flink) | ❌ **MAJOR DIFFERENCE**: Not using Apache Flink |
| "Scanner Workers" (shown as separate services) | **Scanner Service** (single service, horizontally scalable) | ⚠️ Same functionality, but shown as single scalable service |
| "Toplist service" (shown as separate) | **Toplist Updater** (integrated into Indicator/Scanner) | ⚠️ Not a separate service, integrated |

### 2. ❌ Missing Data Flows in Diagram

The diagram is **missing** these important data flows:

1. **Scanner Service also consumes `ticks` stream**
   - Diagram shows: Scanner only consumes `bars.finalized` and `indicators.updated`
   - Reality: Scanner consumes **three sources**:
     - ✅ `ticks` stream (for real-time tick updates)
     - ✅ `bars.finalized` stream (for finalized bars)
     - ✅ `indicators.updated` pub/sub (for indicator updates)

2. **Rules caching flow**
   - Diagram shows: Alert Service reads rules from DB directly
   - Reality: Rules are **cached in Redis** (`rule:{id}` keys)
   - Flow: API Service → DB → Rules Sync Service → Redis cache → Scanner/Alert Service

3. **Rehydration flow**
   - Diagram: Not shown
   - Reality: Scanner Service reads from `bars_1m` table on startup for state rehydration

### 3. ⚠️ Incorrect/Incomplete Details

| Aspect | Diagram Shows | Actual Implementation |
|--------|---------------|----------------------|
| **Bar Aggregator Technology** | Apache Flink | Go service (no Flink) |
| **Alert Service Rules Access** | Reads from DB directly | Reads from Redis cache (`rule:{id}`) |
| **Rules.updated Subscriber** | Alert Service subscribes | ❌ No subscribers yet (published but not consumed) |
| **Toplist Service** | Separate service | Integrated into Indicator/Scanner services |
| **Scanner Architecture** | Multiple worker services shown | Single service, horizontally scalable via consumer groups |

### 4. ✅ Correctly Shown

The diagram correctly shows:
- ✅ Market data → Ingest → `ticks` stream
- ✅ Bars Service → `bars.finalized` stream + DB write
- ✅ Indicator Service → `ind:{sym}` keys + `indicators.updated` pub/sub
- ✅ Scanner → `alerts` stream + `alerts` pub/sub
- ✅ Alert Service → `alerts.filtered` stream + DB write
- ✅ WebSocket Gateway → consumes `alerts.filtered` + `toplists.updated`
- ✅ Toplist ZSET updates
- ✅ API Service → Rules CRUD + Alert history queries

---

## Detailed Comparison

### Data Flow: Market Data → Bars

**Diagram:**
```
Market Data Provider → Market Data Ingestion → stream:ticks → Bar Aggregator (Apache Flink) → stream:bars.finalized + DB
```

**Actual:**
```
Market Data Provider → Ingest Service → stream:ticks → Bars Service (Go) → stream:bars.finalized + TimescaleDB bars_1m
```

**Difference**: ❌ Not using Apache Flink, it's a Go service

---

### Data Flow: Scanner Service

**Diagram Shows:**
- Scanner Workers consume:
  - `stream:bars.finalized`
  - `pub/sub indicators.updated`
- Scanner Workers produce:
  - `stream:alerts`
  - `pub/sub: alerts realtime`

**Actual Implementation:**
- Scanner Service consumes:
  - ✅ `stream:ticks` (MISSING in diagram)
  - ✅ `stream:bars.finalized`
  - ✅ `pub/sub indicators.updated`
- Scanner Service produces:
  - ✅ `stream:alerts`
  - ✅ `pub/sub: alerts` (for real-time)

**Difference**: ⚠️ Diagram missing `ticks` stream consumption by Scanner

---

### Data Flow: Alert Service

**Diagram Shows:**
- Alert Service:
  - Consumes `stream:alerts`
  - Reads rules from DB (rules table)
  - Listens to `pub/sub rules.updated`
  - Produces `stream:alert.filtered`
  - Writes to Alert history table

**Actual Implementation:**
- Alert Service:
  - ✅ Consumes `stream:alerts`
  - ✅ Reads rules from **Redis cache** (`rule:{id}` keys), NOT directly from DB
  - ❌ Does NOT subscribe to `rules.updated` (channel exists but no subscriber)
  - ✅ Produces `stream:alerts.filtered` (note: `alerts.filtered`, not `alert.filtered`)
  - ✅ Writes to Alert history table

**Differences**: 
- ⚠️ Rules come from Redis cache, not DB directly
- ❌ No active subscription to `rules.updated`

---

### Data Flow: Toplist

**Diagram Shows:**
- Separate "Toplist service" updates ZSET and publishes `toplist.updated`

**Actual Implementation:**
- Toplist functionality is **integrated** into:
  - Indicator Service (updates RSI, relative volume, VWAP distance toplists)
  - Scanner Service (updates price change, volume toplists)
- Both publish `toplists.updated` (note: plural "toplists", not "toplist")

**Difference**: ⚠️ Not a separate service, integrated into existing services

---

### Data Flow: API Service

**Diagram Shows:**
- API Service:
  - CRUD on rules table
  - READ queries on Alert history table

**Actual Implementation:**
- API Service:
  - ✅ CRUD on rules table
  - ✅ READ queries on Alert history table
  - ✅ CRUD on toplist_configs table (MISSING in diagram)
  - ⚠️ Missing: bars data access endpoint (TODO)

**Difference**: ⚠️ Diagram missing toplist_configs table operations

---

## Stream/Channel Name Differences

| Component | Diagram Name | Actual Name | Match? |
|-----------|--------------|-------------|--------|
| Stream | `stream:ticks` | `ticks` | ✅ (prefix is just notation) |
| Stream | `stream:bars.finalized` | `bars.finalized` | ✅ |
| Stream | `stream:alerts` | `alerts` | ✅ |
| Stream | `stream:alert.filtered` | `alerts.filtered` | ⚠️ Diagram has typo: "alert" vs "alerts" |
| Pub/Sub | `pub/sub indicators.updated` | `indicators.updated` | ✅ |
| Pub/Sub | `pub/sub: alerts realtime` | `alerts` | ✅ |
| Pub/Sub | `pub/sub rules.updated` | `rules.updated` | ✅ |
| Pub/Sub | `pub/sub toplist.updated` | `toplists.updated` | ⚠️ Diagram: singular "toplist", actual: plural "toplists" |
| Redis Key | `ind:{sym}` | `ind:{symbol}` | ✅ (same pattern) |
| Redis Key | `rule:{id}` | `rule:{ruleID}` | ✅ (same pattern) |
| ZSET | `ZSET TopList:*` | `toplist:system:*` or `toplist:user:*` | ✅ (same concept) |

---

## Missing Components in Diagram

1. ❌ **Rehydration Flow**: Scanner reads from `bars_1m` table on startup
2. ❌ **Rules Sync Service**: Syncs rules from DB to Redis cache
3. ❌ **Toplist Configs Table**: API Service manages `toplist_configs` table
4. ❌ **Live Bar Keys**: `livebar:{symbol}` keys in Redis (not shown)
5. ❌ **Toplist Config Cache**: `toplist:config:{id}` keys in Redis (not shown)

---

## Summary

### What Matches ✅
- Overall data flow direction
- Stream names (with minor typos)
- Pub/sub channel usage
- Database table operations
- WebSocket Gateway functionality
- Toplist ZSET concept

### What's Different ⚠️
- **Apache Flink**: Not used, it's a Go service
- **Scanner consumes ticks**: Missing from diagram
- **Rules caching**: Diagram shows DB access, actual uses Redis cache
- **Toplist service**: Not separate, integrated
- **Stream name typo**: `alert.filtered` vs `alerts.filtered`

### What's Missing ❌
- Scanner Service consuming `ticks` stream
- Rehydration flow (DB → Scanner on startup)
- Rules Sync Service
- Toplist configs table operations
- Live bar keys in Redis

---

## Recommendation

The diagram should be updated to:
1. Remove "Apache Flink" reference (use "Bars Service" or "Bar Aggregator Service")
2. Add `ticks` stream consumption by Scanner Service
3. Show rules caching flow (DB → Redis → Services)
4. Show rehydration flow
5. Fix stream name: `alerts.filtered` (not `alert.filtered`)
6. Fix pub/sub name: `toplists.updated` (not `toplist.updated`)
7. Show Toplist as integrated (not separate service)
8. Add missing Redis keys (livebar, toplist config cache)

The core architecture is correct, but these details need updating for accuracy.

