# Indicator Library Alternatives

This document compares alternatives to implementing all technical indicators from scratch.

## Overview

Your current system uses **incremental/streaming indicators** (RSI, EMA, SMA, VWAP) that update per bar. This is optimal for real-time processing but requires implementing each indicator manually.

## Alternatives Comparison

### 1. Go Technical Analysis Libraries

#### 1.1 **Techan** (`github.com/sdcoffey/techan`)
**Best for: Strategy building and comprehensive indicator set**

- **Pros:**
  - Pure Go, no external dependencies
  - Comprehensive indicator set (RSI, MACD, Bollinger Bands, Stochastic, ADX, etc.)
  - **Streaming architecture** - works with Go channels (matches your architecture!)
  - Strategy building framework
  - Profit/trade analysis tools
  - Well-documented
  - Active maintenance

- **Cons:**
  - May need adapter to match your `Calculator` interface
  - Different API design (series-based vs bar-based)
  - Learning curve for series concept

- **Integration Complexity:** Medium
  - Need to adapt their `Series` concept to your `Update(bar)` pattern
  - Can wrap their indicators in your `Calculator` interface

- **Performance:** Good (pure Go, optimized)

- **Recommendation:** ⭐⭐⭐⭐⭐ **Highly Recommended**
  - Best fit for streaming architecture
  - Most comprehensive pure Go solution

---

#### 1.2 **Indicator** (`github.com/cinar/indicator`)
**Best for: Simple integration and backtesting**

- **Pros:**
  - Pure Go, no dependencies
  - Rich indicator set
  - **Backtesting framework included**
  - Customizable strategies
  - Simple API
  - Works with data streams (channels)

- **Cons:**
  - Less mature than Techan
  - May need adapter layer
  - Smaller community

- **Integration Complexity:** Medium-Low
  - Simpler API than Techan
  - Easier to wrap

- **Performance:** Good

- **Recommendation:** ⭐⭐⭐⭐ **Good Alternative**
  - Good if you want backtesting built-in
  - Simpler than Techan

---

#### 1.3 **Gotal** (`github.com/rangertaha/gotal`)
**Best for: TA-Lib-like API in Go**

- **Pros:**
  - Pure Go port of TA-Lib concepts
  - Wide range of indicators
  - TA-Lib-like API (familiar to many traders)
  - Supports incremental computation

- **Cons:**
  - Less popular/maintained
  - May need significant adapter work
  - Documentation may be limited

- **Integration Complexity:** Medium-High
  - Need to adapt batch-oriented functions to streaming

- **Performance:** Good (pure Go)

- **Recommendation:** ⭐⭐⭐ **Consider if TA-Lib familiarity is important**

---

#### 1.4 **Banta** (`github.com/banbox/banta`)
**Best for: High-performance, event-driven systems**

- **Pros:**
  - **High-performance** (state-caching, parallel computation)
  - Event-driven architecture
  - Flexible framework
  - NaN compatibility
  - Good for real-time applications

- **Cons:**
  - Newer library (less battle-tested)
  - May need adapter work
  - Smaller community

- **Integration Complexity:** Medium
  - Event-driven matches your architecture well

- **Performance:** ⭐⭐⭐⭐⭐ **Excellent**

- **Recommendation:** ⭐⭐⭐⭐ **Good for performance-critical systems**

---

#### 1.5 **Tulip Indicators** (C library with CGO)
**Best for: Maximum indicator coverage**

- **Pros:**
  - **100+ indicators** (most comprehensive)
  - Well-tested C library
  - High performance (C implementation)
  - Industry standard

- **Cons:**
  - **CGO overhead** (slower than pure Go)
  - Build complexity (requires C compiler)
  - Cross-compilation issues
  - Batch-oriented (needs adapter)
  - Deployment complexity

- **Integration Complexity:** High
  - Need CGO bindings
  - Need adapter for streaming

- **Performance:** Good (C) but CGO overhead

- **Recommendation:** ⭐⭐⭐ **Only if you need specific indicators not in Go libraries**

---

### 2. External API Services

#### 2.1 **Polygon.io Indicators API**
**Best for: Offloading computation, reducing server load**

- **Pros:**
  - Pre-computed indicators via API
  - No computation on your servers
  - Always up-to-date
  - Wide indicator coverage
  - Managed service (no maintenance)

- **Cons:**
  - **API costs** (can be expensive at scale)
  - **Latency** (network calls vs in-memory)
  - **Dependency** on external service
  - Rate limits
  - May not match your real-time needs (polling vs streaming)

- **Integration Complexity:** Low-Medium
  - HTTP client integration
  - Need caching layer

- **Performance:** ⚠️ **Network latency** (not ideal for sub-second scanning)

- **Cost:** $$$ (per API call or subscription)

- **Recommendation:** ⭐⭐ **Not recommended for real-time scanning**
  - Good for historical analysis or batch processing
  - Too slow for your 1-second scan loops

---

#### 2.2 **Alpaca Market Data API**
**Best for: Market data + some indicators**

- **Pros:**
  - Already using for market data (if implemented)
  - Some pre-computed indicators available
  - Unified provider

- **Cons:**
  - Limited indicator set
  - API latency
  - Not designed for real-time indicator computation

- **Recommendation:** ⭐⭐ **Not suitable for indicator computation**

---

### 3. Microservice Approach

#### 3.1 **Python Microservice with TA-Lib**
**Best for: Maximum indicator coverage with minimal Go code changes**

- **Architecture:**
  ```
  Go Indicator Engine → gRPC/HTTP → Python Service (TA-Lib) → Results
  ```

- **Pros:**
  - Access to full TA-Lib (200+ indicators)
  - Python TA-Lib is mature and well-tested
  - Minimal changes to Go codebase
  - Can use pandas-ta, ta, or other Python libraries
  - Easy to add new indicators

- **Cons:**
  - **Network latency** (even localhost adds overhead)
  - **Service complexity** (another service to deploy/maintain)
  - **Performance overhead** (serialization, network, Python interpreter)
  - **Resource usage** (Python service + Go service)
  - **Deployment complexity**

- **Integration Complexity:** Medium-High
  - Need gRPC/HTTP client in Go
  - Need Python service
  - Need error handling, retries, etc.

- **Performance:** ⚠️ **Slower than in-process** (network + Python overhead)

- **Recommendation:** ⭐⭐⭐ **Only if you need indicators not available in Go**
  - Good for batch/backtesting
  - Not ideal for real-time streaming

---

### 4. Hybrid Approach (Recommended)

**Combine multiple solutions:**

1. **Use Techan for most indicators** (streaming-friendly, pure Go)
2. **Keep your custom implementations** for simple, performance-critical indicators (RSI, EMA)
3. **Use Python microservice** for complex/rare indicators (only when needed)
4. **Use external APIs** for historical analysis/backtesting

---

## Detailed Comparison Matrix

| Solution | Indicator Count | Streaming Support | Performance | Integration Ease | Maintenance | Cost |
|----------|----------------|-------------------|-------------|------------------|-------------|------|
| **Techan** | 30+ | ✅ Excellent | ⭐⭐⭐⭐ | Medium | Active | Free |
| **Indicator** | 20+ | ✅ Good | ⭐⭐⭐⭐ | Medium | Active | Free |
| **Gotal** | 50+ | ⚠️ Partial | ⭐⭐⭐ | Medium-High | Limited | Free |
| **Banta** | 20+ | ✅ Excellent | ⭐⭐⭐⭐⭐ | Medium | New | Free |
| **Tulip (CGO)** | 100+ | ❌ Batch | ⭐⭐⭐ | High | Active | Free |
| **Polygon API** | 50+ | ❌ Polling | ⭐⭐ | Low | Managed | $$$ |
| **Python Service** | 200+ | ⚠️ Network | ⭐⭐ | Medium-High | Medium | Free |

---

## Recommendations by Use Case

### For Real-Time Streaming (Your Primary Use Case)

**Best Choice: Techan** ⭐⭐⭐⭐⭐
- Streaming architecture matches yours
- Pure Go (no CGO overhead)
- Comprehensive indicator set
- Good performance
- Active maintenance

**Alternative: Banta** ⭐⭐⭐⭐
- If performance is absolutely critical
- Event-driven architecture

### For Maximum Indicator Coverage

**Best Choice: Hybrid**
- Techan for common indicators (real-time)
- Python microservice for rare/complex indicators (on-demand)
- External API for historical analysis

### For Simplest Integration

**Best Choice: Indicator library** ⭐⭐⭐⭐
- Simpler API than Techan
- Backtesting included
- Pure Go

### For Backtesting/Historical Analysis

**Best Choice: Python Service with TA-Lib** ⭐⭐⭐⭐
- Full TA-Lib access
- Easy to add new indicators
- Good for batch processing

---

## Implementation Strategy

### Phase 1: Integrate Techan (Recommended)

1. **Add Techan dependency:**
   ```bash
   go get github.com/sdcoffey/techan
   ```

2. **Create adapter wrapper:**
   ```go
   // pkg/indicator/techan_adapter.go
   type TechanCalculator struct {
       name string
       series *techan.TimeSeries
       indicator techan.Indicator
   }
   
   func (t *TechanCalculator) Update(bar *models.Bar1m) (float64, error) {
       // Convert bar to techan.Candle
       candle := techan.NewCandle(techan.NewTimePeriod(
           bar.Timestamp, time.Minute,
       ))
       candle.OpenPrice = bar.Open
       candle.HighPrice = bar.High
       candle.LowPrice = bar.Low
       candle.ClosePrice = bar.Close
       candle.Volume = float64(bar.Volume)
       
       t.series.AddCandle(candle)
       
       if t.series.LastIndex() >= t.indicator.Period() {
           return t.indicator.Calculate(t.series.LastIndex()).Float(), nil
       }
       return 0, nil
   }
   ```

3. **Register new indicators:**
   ```go
   // MACD example
   macd := techan.NewMACDIndicator(closePrices, 12, 26, 9)
   calc := NewTechanCalculator("macd", series, macd)
   ```

### Phase 2: Add Python Service (Optional, for rare indicators)

1. Create Python service with FastAPI/gRPC
2. Implement TA-Lib wrapper
3. Add Go client for on-demand indicator computation
4. Use only for indicators not in Techan

### Phase 3: Keep Custom for Performance-Critical

- Keep your custom RSI, EMA, SMA (they're optimized for your use case)
- Use Techan for everything else

---

## Cost-Benefit Analysis

### Option A: Pure Techan Integration
- **Development Time:** 2-3 days
- **Performance:** Excellent (pure Go)
- **Maintenance:** Low (library maintained)
- **Indicator Coverage:** 30+ (covers 90% of use cases)
- **Risk:** Low

### Option B: Techan + Python Service
- **Development Time:** 1 week
- **Performance:** Good (Techan) + Slower (Python for rare cases)
- **Maintenance:** Medium (two systems)
- **Indicator Coverage:** 200+ (covers 100% of use cases)
- **Risk:** Medium

### Option C: External API Only
- **Development Time:** 1-2 days
- **Performance:** Poor (network latency)
- **Maintenance:** Low (managed service)
- **Indicator Coverage:** 50+
- **Cost:** High (API fees)
- **Risk:** High (dependency, latency)

---

## Final Recommendation

**Use Techan as primary library** ⭐⭐⭐⭐⭐

**Reasons:**
1. ✅ Streaming architecture matches yours perfectly
2. ✅ Pure Go (no CGO, no external dependencies)
3. ✅ Comprehensive indicator set (covers most needs)
4. ✅ Good performance
5. ✅ Active maintenance and community
6. ✅ Easy to integrate with your `Calculator` interface

**Implementation Plan:**
1. Integrate Techan (2-3 days)
2. Create adapter for your `Calculator` interface
3. Replace/add indicators using Techan
4. Keep your optimized custom indicators (RSI, EMA) if they perform better
5. Add Python service later only if you need specific indicators not in Techan

**Expected Outcome:**
- 30+ indicators available immediately
- No performance degradation
- Minimal code changes
- Easy to add new indicators

---

## Next Steps

1. **Evaluate Techan:** Review their documentation and examples
2. **Proof of Concept:** Implement one indicator (e.g., MACD) using Techan adapter
3. **Benchmark:** Compare Techan performance with your custom implementations
4. **Decide:** Proceed with Techan or explore alternatives

Would you like me to:
- Create a proof-of-concept Techan integration?
- Set up a comparison benchmark?
- Research specific indicators you need?

