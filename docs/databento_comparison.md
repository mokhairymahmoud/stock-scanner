# Python Databento Script vs Go Stock Scanner Architecture Comparison

## Executive Summary

The Python script (`databento.py`) is a **simple, single-file demonstration** of a price movement scanner using Databento's API. The Go project is a **production-ready, microservices-based architecture** designed for scalability, reliability, and extensibility.

---

## Architecture Comparison

### Python Script (databento.py)

**Architecture**: Monolithic, single-process
- Single Python file with one class (`PriceMovementScanner`)
- Direct API integration with Databento
- In-memory state management
- Simple print-based output

**Components**:
- `PriceMovementScanner` class
  - Fetches previous day closing prices on initialization
  - Subscribes to live market data (MBP-1 messages)
  - Compares current mid price vs previous close
  - Prints alerts when threshold exceeded

### Go Project (stock-scanner)

**Architecture**: Microservices, distributed system
- 7+ independent services
- Event-driven architecture using Redis Streams
- Persistent storage (TimescaleDB)
- Horizontal scaling support

**Components**:
1. **Ingest Service**: Market data ingestion from multiple providers
2. **Bars Service**: Tick aggregation into 1-minute bars
3. **Indicator Service**: Technical indicator computation (RSI, EMA, VWAP)
4. **Scanner Service**: Rule evaluation and alert generation
5. **Alert Service**: Alert processing, deduplication, filtering
6. **WebSocket Gateway**: Real-time alert delivery
7. **API Service**: REST API for rule management and queries

---

## Feature Comparison

| Feature | Python Script | Go Project |
|---------|--------------|------------|
| **Data Source** | Databento only | Multiple providers (Polygon, IEX, dxFeed, Mock) |
| **Symbol Coverage** | ALL_SYMBOLS (~9,000) | Configurable symbol list |
| **Price Comparison** | Previous day close | Multiple timeframes (1m, 5m, 15m, 1d) |
| **Alert Mechanism** | Simple threshold (3%) | Rule-based system with multiple conditions |
| **Alert Delivery** | Print to console | WebSocket, REST API, persistent storage |
| **State Management** | In-memory only | Redis (hot) + TimescaleDB (warm) |
| **Scalability** | Single process | Horizontal scaling (multiple workers) |
| **Persistence** | None | TimescaleDB for bars, alerts, rules |
| **Technical Indicators** | None | RSI, EMA, VWAP, price change, volume metrics |
| **Cooldown** | One-time flag (`is_signal_lit`) | Configurable per-rule cooldowns |
| **Toplists** | None | System and user-custom toplists |
| **Monitoring** | None | Prometheus metrics, health checks, tracing |

---

## Price Movement Detection

### Python Script Approach

```python
# Simple comparison: current mid price vs previous day close
mid = (bid + ask) * PX_SCALE * 0.5
last = self.last_day_lookup[symbol]
abs_r = abs(mid - last) / last

if abs_r > self.pct_threshold and not self.is_signal_lit[symbol]:
    print(f"{symbol} moved by {abs_r * 100:.2f}%")
    self.is_signal_lit[symbol] = True  # One-time alert
```

**Characteristics**:
- Uses **mid price** (bid + ask) / 2
- Compares against **previous day's closing price**
- **One-time alert** per symbol (flag prevents re-alerting)
- **Fixed threshold** (3% default, configurable)

### Go Project Approach

```go
// Multiple metrics computed from bars and indicators
metrics := map[string]float64{
    "price": liveBar.Close,
    "price_change_1m_pct": ((currentBar.Close - prevBar.Close) / prevBar.Close) * 100.0,
    "price_change_5m_pct": ((currentBar.Close - bar5m.Close) / bar5m.Close) * 100.0,
    "price_change_15m_pct": ((currentBar.Close - bar15m.Close) / bar15m.Close) * 100.0,
    "rsi_14": indicators["rsi_14"],
    "ema_20": indicators["ema_20"],
    // ... more metrics
}

// Rule-based evaluation with multiple conditions
if rule.Matches(metrics) && !cooldownTracker.IsOnCooldown(ruleID, symbol) {
    alertEmitter.EmitAlert(alert)
    cooldownTracker.RecordCooldown(ruleID, symbol, rule.Cooldown)
}
```

**Characteristics**:
- Uses **close price** from live bars
- Compares against **multiple timeframes** (1m, 5m, 15m bars)
- **Rule-based system** with complex conditions
- **Configurable cooldowns** per rule
- **Multiple alerts possible** (different rules can trigger)

---

## Data Flow Comparison

### Python Script Data Flow

```
Databento Live API
    ↓
PriceMovementScanner.scan()
    ↓
Compare mid price vs previous close
    ↓
Print alert (if threshold exceeded)
```

**Latency**: Sub-millisecond (direct API callback)
**Throughput**: Limited by single process
**Reliability**: No persistence, no retry logic

### Go Project Data Flow

```
Market Data Provider
    ↓
[Ingest Service] → Redis Streams (ticks)
    ↓
[Bars Service] → TimescaleDB + Redis Streams (bars.finalized)
    ↓
[Indicator Service] → Redis (ind:{symbol}) + Pub/Sub
    ↓
[Scanner Workers] → Evaluate Rules → Redis Streams (alerts)
    ↓
[Alert Service] → Deduplication → TimescaleDB + Redis Streams (alerts.filtered)
    ↓
[WebSocket Gateway] → Real-time delivery to clients
```

**Latency**: <1 second scan cycle (target: <800ms)
**Throughput**: Horizontally scalable (multiple workers)
**Reliability**: Persistent storage, retry logic, deduplication

---

## Key Differences

### 1. **Previous Day Closing Price**

**Python Script**:
- Fetches previous day closing prices **on initialization** using Databento Historical API
- Stores in dictionary: `last_day_lookup: Dict[str, float]`
- Uses `ohlcv-1d` schema to get yesterday's close
- **Note**: Comment mentions handling overnight splits (TODO)

**Go Project**:
- **Currently does NOT fetch previous day closing prices**
- Computes price changes from **finalized 1-minute bars** only
- Price change metrics:
  - `price_change_1m_pct`: Current bar vs previous bar
  - `price_change_5m_pct`: Current bar vs 5 bars ago
  - `price_change_15m_pct`: Current bar vs 15 bars ago
- **Missing**: Previous day close comparison (could be added)

### 2. **Price Source**

**Python Script**:
- Uses **mid price**: `(bid + ask) / 2` from MBP-1 messages
- Handles null prices (when one side of book is empty)

**Go Project**:
- Uses **close price** from live bars (last trade price)
- Could also use bid/ask if needed (data available in Tick model)

### 3. **Alert Deduplication**

**Python Script**:
- Simple flag: `is_signal_lit[symbol] = True`
- **One-time alert** per symbol (never re-alerts for same symbol)

**Go Project**:
- **Per-rule cooldowns**: Each rule has its own cooldown period
- **Idempotency keys**: Prevents duplicate alerts across workers
- **Multiple alerts possible**: Different rules can trigger for same symbol

### 4. **Scalability**

**Python Script**:
- Single process, single thread
- Limited by Python GIL and single API connection
- Cannot scale horizontally

**Go Project**:
- **Horizontal scaling**: Multiple scanner workers
- **Symbol partitioning**: Consistent hash assigns symbols to workers
- **Consumer groups**: Redis Streams handle load balancing
- **State rehydration**: Workers can recover state on restart

### 5. **State Management**

**Python Script**:
- In-memory dictionaries:
  - `symbol_directory`: instrument_id → symbol mapping
  - `last_day_lookup`: symbol → previous close price
  - `is_signal_lit`: symbol → boolean flag
- **Lost on restart**: No persistence

**Go Project**:
- **Hot state** (Redis):
  - Live bars: `livebar:{symbol}` (TTL: 5 min)
  - Indicators: `ind:{symbol}` (TTL: 10 min)
  - Rules cache: `rules:{rule_id}` (TTL: 1 hour)
- **Warm state** (TimescaleDB):
  - Finalized bars (1 year retention)
  - Alert history (1 year retention)
  - Rules (persistent)
- **In-memory** (per worker):
  - Symbol state (live bars, finalized bars ring buffer, indicators)
  - Cooldown tracker

---

## What the Go Project Could Learn from Python Script

### 1. **Previous Day Close Comparison**

The Python script's approach of comparing against previous day's closing price is useful for detecting **overnight gaps** and **daily momentum**. The Go project could add this:

**Implementation Suggestion**:
```go
// Add to metrics computation
if previousDayClose > 0 {
    changeFromPrevDay := ((currentPrice - previousDayClose) / previousDayClose) * 100.0
    metrics["price_change_1d_pct"] = changeFromPrevDay
}
```

**Where to fetch**:
- On startup: Query TimescaleDB for last day's closing bar
- Or: Use historical data provider API (like Python script does)
- Or: Store daily closing prices in separate table

### 2. **Mid Price Calculation**

The Python script uses mid price (bid/ask average), which can be more stable than last trade price. The Go project could add this option:

**Implementation Suggestion**:
```go
// In Tick model, already has Bid and Ask fields
// Could compute mid price in bar aggregator or scanner
if tick.Bid > 0 && tick.Ask > 0 {
    midPrice := (tick.Bid + tick.Ask) / 2.0
    // Use midPrice instead of tick.Price for bar updates
}
```

### 3. **Overnight Split Handling**

The Python script has a TODO comment about handling overnight splits. The Go project should also consider this:

**Implementation Suggestion**:
- Query corporate actions API (Databento or other provider)
- Adjust historical prices when splits occur
- Store split-adjusted prices separately

---

## What the Python Script Could Learn from Go Project

### 1. **Rule-Based System**

Instead of fixed threshold, allow complex rules:
```python
# Instead of: if abs_r > threshold
# Allow: if (rsi < 30 and volume > avg_volume * 2) or (price_change_5m > 5%)
```

### 2. **Persistence**

Store alerts, state, and historical data:
- Database for alert history
- Redis for hot state
- File system or S3 for long-term archives

### 3. **Scalability**

- Multiple worker processes
- Symbol partitioning
- Load balancing

### 4. **Monitoring**

- Metrics (Prometheus)
- Health checks
- Distributed tracing
- Logging

### 5. **Alert Delivery**

- WebSocket for real-time delivery
- REST API for querying history
- Multiple delivery channels (email, SMS, etc.)

---

## Recommendations

### For Go Project

1. **Add Previous Day Close Comparison**
   - Fetch previous day's closing prices on startup
   - Store in Redis with TTL (refresh daily)
   - Add `price_change_1d_pct` metric
   - Consider overnight gap detection

2. **Add Mid Price Option**
   - Support both last trade price and mid price
   - Make it configurable per rule or globally
   - Use mid price for more stable comparisons

3. **Consider Databento Integration**
   - The Python script shows Databento is a viable provider
   - Could add Databento as a market data provider option
   - Leverage their historical data API for previous day closes

### For Python Script (if evolving to production)

1. **Add Persistence**
   - Store alerts to database
   - Cache previous day closes in Redis
   - Add state recovery on restart

2. **Add Rule System**
   - Replace fixed threshold with rule engine
   - Support multiple conditions
   - Add cooldown management

3. **Add Scalability**
   - Multiple worker processes
   - Symbol partitioning
   - Load balancing

4. **Add Monitoring**
   - Metrics collection
   - Health checks
   - Logging

---

## Conclusion

The Python script is an **excellent demonstration** of a simple price movement scanner using Databento. It's perfect for:
- Learning Databento API
- Quick prototyping
- Single-user scenarios
- Educational purposes

The Go project is a **production-ready system** designed for:
- High throughput (thousands of symbols)
- Low latency (<1s scan cycles)
- Horizontal scaling
- Multi-user scenarios
- Enterprise reliability

**Key Takeaway**: The Go project could benefit from the Python script's approach to previous day close comparison, while the Python script could evolve toward the Go project's architecture for production use.

