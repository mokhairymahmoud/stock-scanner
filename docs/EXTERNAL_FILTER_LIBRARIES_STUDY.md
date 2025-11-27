# External Filter Libraries Study

This document analyzes external libraries and services that can compute filters from `Filters.md` without requiring custom implementation.

## Executive Summary

**Best Options:**
1. **Python Microservice (pandas-ta + TA-Lib)** ⭐⭐⭐⭐⭐ - Most comprehensive
2. **TAAPI.IO API** ⭐⭐⭐⭐ - Easiest integration, but network latency
3. **Java Microservice (Ta4j)** ⭐⭐⭐⭐ - Good coverage, better performance than Python
4. **Polygon.io API** ⭐⭐⭐ - Limited to their supported indicators

**Recommendation:** Python microservice with pandas-ta + TA-Lib for maximum coverage with minimal code.

---

## Option Comparison Matrix

| Option | Filter Coverage | Performance | Integration | Cost | Maintenance |
|--------|----------------|-------------|-------------|------|-------------|
| **Python (pandas-ta)** | ⭐⭐⭐⭐⭐ 90%+ | ⭐⭐⭐ | Medium | Free | Medium |
| **Python (TA-Lib)** | ⭐⭐⭐⭐⭐ 100% | ⭐⭐⭐ | Medium | Free | Low |
| **TAAPI.IO API** | ⭐⭐⭐⭐ 80%+ | ⭐⭐ | Easy | $$$ | None |
| **Polygon.io API** | ⭐⭐⭐ 60% | ⭐⭐ | Easy | $$$ | None |
| **Java (Ta4j)** | ⭐⭐⭐⭐ 70% | ⭐⭐⭐⭐ | Medium | Free | Medium |
| **Go (Techan)** | ⭐⭐ 30% | ⭐⭐⭐⭐⭐ | Easy | Free | Low |

---

## Detailed Analysis

### Option 1: Python Microservice (pandas-ta + TA-Lib) ⭐⭐⭐⭐⭐

**Best for: Maximum filter coverage with proven libraries**

#### Libraries

**pandas-ta:**
- 130+ indicators
- Volume indicators (OBV, Volume SMA, etc.)
- Price indicators (all moving averages, RSI, MACD, etc.)
- Range indicators (ATR, Bollinger Bands)
- Pattern recognition
- Built on pandas (fast with NumPy)

**TA-Lib (Python):**
- 200+ indicators
- Industry standard
- Well-tested
- C library (fast)

#### Coverage Analysis

| Filter Category | pandas-ta | TA-Lib | Combined |
|----------------|-----------|--------|----------|
| **Volume Filters** | ⚠️ Partial | ❌ | ⚠️ Partial (need custom for session-based) |
| **Price Filters** | ✅ Yes | ✅ Yes | ✅ Yes |
| **Range Filters** | ✅ Yes | ✅ Yes | ✅ Yes |
| **Technical Indicators** | ✅ Yes | ✅ Yes | ✅ Yes |
| **Trading Activity** | ❌ | ❌ | ❌ (need custom) |
| **Time-Based** | ❌ | ❌ | ❌ (need custom) |
| **Fundamental** | ❌ | ❌ | ❌ (need external data) |

**Estimated Coverage: 70-80%** (technical indicators + basic filters)

#### Architecture

```
Go Indicator Engine → gRPC/HTTP → Python Service → Results
```

**Python Service Structure:**
```python
# services/filter_service/main.py
from fastapi import FastAPI
from pydantic import BaseModel
import pandas_ta as ta
import talib

app = FastAPI()

class BarData(BaseModel):
    symbol: str
    timestamp: str
    open: float
    high: float
    low: float
    close: float
    volume: int

class FilterRequest(BaseModel):
    filter_type: str
    bars: list[BarData]
    params: dict

@app.post("/compute-filter")
async def compute_filter(request: FilterRequest):
    # Convert to pandas DataFrame
    df = pd.DataFrame([bar.dict() for bar in request.bars])
    
    # Compute filter based on type
    if request.filter_type == "rsi_14":
        result = ta.rsi(df['close'], length=14)
    elif request.filter_type == "volume_5m":
        result = df['volume'].rolling(5).sum()
    # ... etc
    
    return {"value": result.iloc[-1]}
```

#### Pros
- ✅ **Maximum coverage** - pandas-ta + TA-Lib covers most filters
- ✅ **Proven libraries** - Well-tested, industry-standard
- ✅ **Easy to extend** - Python ecosystem is rich
- ✅ **No licensing costs** - Open source
- ✅ **Good documentation** - Both libraries well-documented

#### Cons
- ❌ **Network latency** - Even localhost adds 1-5ms overhead
- ❌ **Service complexity** - Another service to deploy/maintain
- ❌ **Performance overhead** - Python interpreter + serialization
- ❌ **Resource usage** - Python service needs memory
- ❌ **Deployment complexity** - Need Python runtime

#### Implementation Effort
- **Development:** 1-2 weeks
- **Integration:** Medium complexity
- **Maintenance:** Medium (Python service + Go client)

#### Cost
- **Library:** Free (open source)
- **Infrastructure:** Additional service (CPU/memory)

---

### Option 2: TAAPI.IO API ⭐⭐⭐⭐

**Best for: Zero-maintenance, API-based solution**

#### Service Overview
- **200+ technical indicators** via REST API
- Pre-computed indicators
- No local computation needed
- Pay-per-use or subscription model

#### Coverage Analysis

| Filter Category | Coverage |
|----------------|----------|
| **Volume Filters** | ⚠️ Limited (basic volume indicators) |
| **Price Filters** | ✅ Yes (price-based indicators) |
| **Range Filters** | ✅ Yes (ATR, Bollinger Bands) |
| **Technical Indicators** | ✅ Yes (all major indicators) |
| **Trading Activity** | ❌ No |
| **Time-Based** | ❌ No |
| **Fundamental** | ❌ No |

**Estimated Coverage: 60-70%** (technical indicators only)

#### API Example

```go
// Go client
type TAAPIClient struct {
    apiKey string
    baseURL string
}

func (c *TAAPIClient) GetRSI(symbol string, period int) (float64, error) {
    url := fmt.Sprintf("%s/rsi?symbol=%s&interval=1m&period=%d", 
        c.baseURL, symbol, period)
    
    resp, err := http.Get(url + "&apikey=" + c.apiKey)
    // Parse response
    return value, nil
}
```

#### Pros
- ✅ **Zero maintenance** - Managed service
- ✅ **Easy integration** - Simple HTTP API
- ✅ **Always up-to-date** - Service maintained by provider
- ✅ **No local computation** - Offloads CPU usage
- ✅ **200+ indicators** - Comprehensive coverage

#### Cons
- ❌ **Network latency** - API calls add 10-50ms (not suitable for real-time)
- ❌ **API costs** - Pay per request or subscription
- ❌ **Rate limits** - May hit limits at scale
- ❌ **Dependency** - External service dependency
- ❌ **Limited customization** - Can't modify calculations
- ❌ **Not suitable for real-time** - Too slow for sub-second scanning

#### Pricing (Estimated)
- **Pay-per-use:** ~$0.001 per indicator call
- **Subscription:** $50-200/month for higher limits
- **At scale (10K symbols, 1 call/sec):** ~$260/month

#### Implementation Effort
- **Development:** 2-3 days
- **Integration:** Easy (HTTP client)
- **Maintenance:** Low (managed service)

---

### Option 3: Polygon.io API ⭐⭐⭐

**Best for: Market data + some indicators**

#### Service Overview
- Market data provider (already considering for data)
- Some pre-computed indicators
- Aggregates API (bars, quotes, trades)
- Technical indicators API (limited)

#### Coverage Analysis

| Filter Category | Coverage |
|----------------|----------|
| **Volume Filters** | ✅ Yes (volume data available) |
| **Price Filters** | ✅ Yes (price data available) |
| **Range Filters** | ✅ Yes (high/low data available) |
| **Technical Indicators** | ⚠️ Limited (basic indicators only) |
| **Trading Activity** | ⚠️ Partial (trade count available) |
| **Time-Based** | ❌ No |
| **Fundamental** | ⚠️ Partial (some fundamental data) |

**Estimated Coverage: 50-60%** (data available, but need to compute filters)

#### Pros
- ✅ **Unified provider** - Already using for market data
- ✅ **Volume/Price data** - Raw data available
- ✅ **Some indicators** - Basic technical indicators
- ✅ **Fundamental data** - MarketCap, Float, etc.

#### Cons
- ❌ **Limited indicators** - Not comprehensive
- ❌ **API latency** - Network calls
- ❌ **Cost** - API subscription required
- ❌ **Still need computation** - Most filters need custom logic
- ❌ **Not suitable for real-time** - API calls too slow

#### Pricing
- **Starter:** $99/month
- **Developer:** $199/month
- **Advanced:** $499/month

---

### Option 4: Java Microservice (Ta4j) ⭐⭐⭐⭐

**Best for: Better performance than Python, good indicator coverage**

#### Library Overview
- **130+ indicators**
- Strategy engine
- Backtesting framework
- Pure Java (no C dependencies)

#### Coverage Analysis

| Filter Category | Coverage |
|----------------|----------|
| **Volume Filters** | ⚠️ Partial (basic volume tracking) |
| **Price Filters** | ✅ Yes (price-based indicators) |
| **Range Filters** | ✅ Yes (ATR, range calculations) |
| **Technical Indicators** | ✅ Yes (130+ indicators) |
| **Trading Activity** | ⚠️ Partial (pattern detection) |
| **Time-Based** | ❌ No |
| **Fundamental** | ❌ No |

**Estimated Coverage: 60-70%** (technical indicators + basic filters)

#### Architecture

```
Go Indicator Engine → gRPC → Java Service (Spring Boot) → Results
```

#### Pros
- ✅ **Better performance** - JVM faster than Python (2-5x)
- ✅ **130+ indicators** - Good coverage
- ✅ **Strategy engine** - Built-in backtesting
- ✅ **Pure Java** - No C dependencies

#### Cons
- ❌ **Network latency** - Still has overhead
- ❌ **Service complexity** - Another service to maintain
- ❌ **JVM overhead** - Memory footprint
- ❌ **Less coverage than Python** - Fewer indicators than pandas-ta

#### Implementation Effort
- **Development:** 1-2 weeks
- **Integration:** Medium complexity
- **Maintenance:** Medium

---

### Option 5: Go Libraries (Techan, etc.) ⭐⭐

**Best for: In-process, no network overhead**

#### Coverage Analysis

| Filter Category | Coverage |
|----------------|----------|
| **Volume Filters** | ❌ No |
| **Price Filters** | ❌ No |
| **Range Filters** | ❌ No |
| **Technical Indicators** | ✅ Yes (30+ indicators) |
| **Trading Activity** | ❌ No |
| **Time-Based** | ❌ No |
| **Fundamental** | ❌ No |

**Estimated Coverage: 20-30%** (technical indicators only)

#### Pros
- ✅ **No network overhead** - In-process
- ✅ **Best performance** - Native Go
- ✅ **Simple deployment** - Single binary

#### Cons
- ❌ **Limited coverage** - Only technical indicators
- ❌ **Need custom code** - Most filters still need implementation

**Note:** This is already in the integration plan. Good for indicators, but doesn't help with most filters.

---

## Filter-by-Filter Coverage Analysis

### Volume Filters (7 filters)

| Filter | pandas-ta | TA-Lib | TAAPI.IO | Polygon | Ta4j | Techan |
|--------|-----------|--------|----------|---------|------|--------|
| Postmarket Volume | ❌ | ❌ | ❌ | ⚠️ Data | ❌ | ❌ |
| Premarket Volume | ❌ | ❌ | ❌ | ⚠️ Data | ❌ | ❌ |
| Absolute Volume | ⚠️ Custom | ❌ | ❌ | ✅ Data | ⚠️ Custom | ❌ |
| Dollar Volume | ⚠️ Custom | ❌ | ❌ | ✅ Data | ⚠️ Custom | ❌ |
| Average Volume | ✅ | ❌ | ❌ | ✅ Data | ⚠️ Custom | ❌ |
| Relative Volume | ⚠️ Custom | ❌ | ❌ | ⚠️ Data | ⚠️ Custom | ❌ |
| Relative Volume Same Time | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |

**Winner:** Polygon.io (provides data), but still need custom computation

### Price Filters (8 filters)

| Filter | pandas-ta | TA-Lib | TAAPI.IO | Polygon | Ta4j | Techan |
|--------|-----------|--------|----------|---------|------|--------|
| Price | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Change ($) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Change from Close | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Change from Open | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Percentage Change | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Gap from Close | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

**Winner:** All provide basic price calculations

### Range Filters (5 filters)

| Filter | pandas-ta | TA-Lib | TAAPI.IO | Polygon | Ta4j | Techan |
|--------|-----------|--------|----------|---------|------|--------|
| Range ($) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Percentage Range | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Biggest Range | ⚠️ Custom | ⚠️ Custom | ❌ | ⚠️ Data | ⚠️ Custom | ❌ |
| Relative Range | ✅ (ATR) | ✅ (ATR) | ✅ | ⚠️ Data | ✅ (ATR) | ✅ (ATR) |
| Position in Range | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

**Winner:** pandas-ta/TA-Lib (comprehensive)

### Technical Indicators (5 filters)

| Filter | pandas-ta | TA-Lib | TAAPI.IO | Polygon | Ta4j | Techan |
|--------|-----------|--------|----------|---------|------|--------|
| RSI | ✅ | ✅ | ✅ | ⚠️ Limited | ✅ | ✅ |
| ATR | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ |
| ATRP | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ |
| VWAP Distance | ⚠️ Custom | ⚠️ Custom | ❌ | ⚠️ Data | ⚠️ Custom | ⚠️ Custom |
| MA Distance | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ |

**Winner:** pandas-ta/TA-Lib (all covered)

### Trading Activity (2 filters)

| Filter | pandas-ta | TA-Lib | TAAPI.IO | Polygon | Ta4j | Techan |
|--------|-----------|--------|----------|---------|------|--------|
| Trade Count | ❌ | ❌ | ❌ | ✅ Data | ❌ | ❌ |
| Consecutive Candles | ⚠️ Custom | ❌ | ❌ | ⚠️ Data | ⚠️ Custom | ❌ |

**Winner:** Polygon.io (provides trade data)

### Time-Based (5 filters)

| Filter | pandas-ta | TA-Lib | TAAPI.IO | Polygon | Ta4j | Techan |
|--------|-----------|--------|----------|---------|------|--------|
| Minutes in Market | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| Minutes Since News | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| Days Until Earnings | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |

**Winner:** None - All require custom implementation + external data

### Fundamental (6 filters)

| Filter | pandas-ta | TA-Lib | TAAPI.IO | Polygon | Ta4j | Techan |
|--------|-----------|--------|----------|---------|------|--------|
| MarketCap | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ |
| Float | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ |
| Shares Outstanding | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ |
| Short Interest | ❌ | ❌ | ❌ | ⚠️ Limited | ❌ | ❌ |

**Winner:** Polygon.io (fundamental data API)

---

## Recommendation Matrix

### For Maximum Coverage (90%+)
**Python Microservice (pandas-ta + TA-Lib)** ⭐⭐⭐⭐⭐
- Covers most technical indicators and basic filters
- Still need custom code for: session-based volumes, time-based, fundamental data
- **Estimated effort saved: 60-70%**

### For Easiest Integration
**TAAPI.IO API** ⭐⭐⭐⭐
- Zero code, just API calls
- But too slow for real-time (10-50ms per call)
- **Best for:** Batch processing, backtesting
- **Not suitable for:** Real-time scanning

### For Best Performance
**Go (Techan) + Custom** ⭐⭐⭐
- Already in plan
- Best performance (no network)
- But limited coverage (30%)
- **Best for:** Real-time indicators only

### For Unified Provider
**Polygon.io API** ⭐⭐⭐
- Market data + some indicators + fundamental data
- But still need custom computation for most filters
- **Best for:** If already using Polygon for market data

---

## Hybrid Approach (Recommended)

**Combine multiple solutions:**

1. **Use Techan (Go)** for real-time technical indicators
   - RSI, EMA, SMA, MACD, ATR (in-process, fast)

2. **Use Python microservice** for complex/computed filters
   - Volume aggregations
   - Range calculations
   - Pattern detection
   - (On-demand, slower but comprehensive)

3. **Use Polygon.io API** for fundamental data
   - MarketCap, Float, Shares Outstanding
   - (Cached, updated infrequently)

4. **Custom implementation** for:
   - Session-based volumes (premarket/postmarket)
   - Time-based filters (minutes in market)
   - News/earnings integration

**Architecture:**
```
┌─────────────────────────────────────────┐
│  Go Scanner (Real-Time)                 │
│  ├── Techan Indicators (fast)           │
│  └── Custom Metrics (fast)               │
└─────────────────────────────────────────┘
           │
           ├── On-demand → Python Service (complex filters)
           └── Cached → Polygon.io API (fundamental data)
```

---

## Cost-Benefit Analysis

### Option A: Python Microservice
- **Development:** 1-2 weeks
- **Coverage:** 70-80% of filters
- **Performance:** 2-5ms per filter (localhost)
- **Maintenance:** Medium
- **Cost:** Free (open source)
- **ROI:** High (saves 60-70% of development time)

### Option B: TAAPI.IO API
- **Development:** 2-3 days
- **Coverage:** 60-70% of filters
- **Performance:** 10-50ms per filter (network)
- **Maintenance:** Low (managed)
- **Cost:** $50-500/month
- **ROI:** Medium (easy but expensive and slow)

### Option C: Polygon.io API
- **Development:** 1 week
- **Coverage:** 50-60% of filters
- **Performance:** 10-50ms per filter (network)
- **Maintenance:** Low (managed)
- **Cost:** $99-499/month
- **ROI:** Low (limited coverage, still need custom code)

### Option D: Custom Implementation
- **Development:** 4-6 weeks
- **Coverage:** 100% of filters
- **Performance:** <1ms per filter (in-process)
- **Maintenance:** High
- **Cost:** Development time
- **ROI:** Low (most time-consuming)

---

## Final Recommendation

### Primary Recommendation: Python Microservice (pandas-ta + TA-Lib)

**Why:**
1. ✅ **Maximum coverage** - 70-80% of filters
2. ✅ **Proven libraries** - Industry standard
3. ✅ **Easy to extend** - Python ecosystem
4. ✅ **Good ROI** - Saves 60-70% development time
5. ✅ **Flexible** - Can add custom logic easily

**Implementation:**
- Create Python FastAPI service
- Expose gRPC or HTTP endpoints
- Use pandas-ta for most filters
- Use TA-Lib for additional indicators
- Keep Go service for real-time indicators (Techan)

**When to use:**
- Complex volume aggregations
- Range calculations
- Pattern detection
- On-demand computation (not in scan loop)

**When NOT to use:**
- Real-time indicators (use Techan in Go)
- Simple calculations (implement in Go)
- Time-based filters (custom Go code)

### Secondary Recommendation: Hybrid Approach

**Use:**
1. **Techan (Go)** - Real-time technical indicators
2. **Python Service** - Complex filters on-demand
3. **Polygon.io** - Fundamental data (cached)
4. **Custom Go** - Session-based, time-based filters

**This gives you:**
- Best performance (real-time in Go)
- Maximum coverage (Python for complex)
- External data (Polygon for fundamentals)
- Minimal custom code (only session/time filters)

---

## Next Steps

1. **Evaluate Python libraries:**
   - Test pandas-ta with sample data
   - Test TA-Lib Python bindings
   - Benchmark performance

2. **Create proof-of-concept:**
   - Simple Python FastAPI service
   - Implement 5-10 filters
   - Test with Go client

3. **Compare performance:**
   - Python service vs custom Go implementation
   - Network overhead measurement
   - Memory usage comparison

4. **Decision:**
   - If performance acceptable → Use Python service
   - If too slow → Use custom Go implementation
   - Hybrid approach for best of both worlds

