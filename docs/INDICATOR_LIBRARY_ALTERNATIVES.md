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

#### 3.2 **Java Microservice with Ta4j** ⭐⭐⭐⭐
**Best for: Comprehensive indicator set with better performance than Python**

- **Architecture:**
  ```
  Go Indicator Engine → gRPC/HTTP → Java Service (Ta4j) → Results
  ```

- **Pros:**
  - **130+ indicators** (more than Techan, less than TA-Lib)
  - **Streaming-friendly** - Ta4j uses `BarSeries` concept (similar to your architecture!)
  - **Better performance than Python** (JVM is faster than Python interpreter)
  - **Strategy engine built-in** - can evaluate trading strategies
  - **Pure Java** (no C dependencies, easier deployment than CGO)
  - **Well-documented** and actively maintained
  - **Flexible numeric precision** - supports BigDecimal or DoubleNum for performance
  - **Backtesting framework** included
  - **Minimal dependencies** (cleaner than Python ecosystem)

- **Cons:**
  - **Network latency** (even localhost adds overhead)
  - **Service complexity** (another service to deploy/maintain)
  - **JVM overhead** (memory footprint, startup time)
  - **Resource usage** (Java service + Go service)
  - **Deployment complexity** (need Java runtime)
  - **Language barrier** (Java vs Go team expertise)

- **Integration Complexity:** Medium-High
  - Need gRPC/HTTP client in Go
  - Need Java service (Spring Boot, Quarkus, or plain Java)
  - Need error handling, retries, connection pooling
  - Need to convert between Go models and Java models

- **Performance:** ⭐⭐⭐ **Better than Python, worse than in-process Go**
  - JVM is faster than Python
  - But still has network + serialization overhead
  - JVM warmup time (first requests slower)

- **Ta4j Streaming Architecture:**
  ```java
  // Ta4j uses BarSeries (similar to your bar concept)
  BarSeries series = new BaseBarSeries();
  
  // Add bars incrementally (matches your Update(bar) pattern!)
  series.addBar(bar);
  
  // Calculate indicators on the series
  Indicator<Num> rsi = new RSIIndicator(closePrice, 14);
  Num rsiValue = rsi.getValue(series.getEndIndex());
  ```

- **Recommendation:** ⭐⭐⭐⭐ **Best microservice option if you need more indicators**
  - **Better than Python** for performance
  - **More indicators than Techan** (130+ vs 30+)
  - **Streaming-friendly** architecture
  - Good for real-time if network latency is acceptable
  - Excellent for backtesting/strategy development

- **When to Choose Ta4j over Techan:**
  - Need indicators not in Techan (check Ta4j's 130+ list)
  - Want built-in strategy engine
  - Need backtesting framework
  - Team has Java expertise
  - Can accept microservice architecture

- **When to Choose Techan over Ta4j:**
  - Want everything in Go (no microservice)
  - Need maximum performance (no network overhead)
  - Simpler deployment (one language stack)
  - Techan's 30+ indicators cover your needs

---

### 4. Hybrid Approach (Recommended)

**Combine multiple solutions:**

1. **Use Techan for most indicators** (streaming-friendly, pure Go)
2. **Keep your custom implementations** for simple, performance-critical indicators (RSI, EMA)
3. **Use Java/Ta4j microservice** for additional indicators not in Techan (better than Python)
4. **Use external APIs** for historical analysis/backtesting

---

## Ta4j Deep Dive

### Why Ta4j is Interesting for Your Use Case

**Ta4j** (`github.com/ta4j/ta4j`) is a Java library that's particularly interesting because:

1. **Streaming Architecture Match:**
   - Ta4j uses `BarSeries` which is conceptually similar to your bar-based system
   - Supports incremental updates (add bars one at a time)
   - Indicators calculate on-demand from the series
   - This matches your `Update(bar)` pattern better than batch-oriented libraries

2. **Comprehensive Indicator Set:**
   - 130+ indicators (vs Techan's 30+)
   - Includes: Aroon, ATR, Bollinger Bands, MACD, Parabolic SAR, Stochastic, Williams %R, and many more
   - Well-tested implementations

3. **Strategy Engine:**
   - Built-in strategy building and evaluation
   - Backtesting framework
   - Rule-based strategy execution
   - Could complement your rule engine

4. **Performance Considerations:**
   - JVM is faster than Python (2-5x typically)
   - But still slower than in-process Go
   - Network overhead is the main bottleneck

### Ta4j Architecture Example

```java
// Java service (Spring Boot example)
@RestController
public class IndicatorController {
    
    @PostMapping("/indicator/rsi")
    public RsiResponse calculateRsi(@RequestBody BarRequest request) {
        // Create BarSeries from request
        BarSeries series = new BaseBarSeries();
        for (BarData bar : request.getBars()) {
            series.addBar(
                ZonedDateTime.parse(bar.getTimestamp()),
                bar.getOpen(), bar.getHigh(), 
                bar.getLow(), bar.getClose(), 
                bar.getVolume()
            );
        }
        
        // Calculate RSI
        ClosePriceIndicator closePrice = new ClosePriceIndicator(series);
        RSIIndicator rsi = new RSIIndicator(closePrice, 14);
        Num value = rsi.getValue(series.getEndIndex());
        
        return new RsiResponse(value.doubleValue());
    }
}
```

```go
// Go client adapter
type Ta4jCalculator struct {
    name string
    client *Ta4jClient // gRPC or HTTP client
    symbol string
    bars []*models.Bar1m
}

func (t *Ta4jCalculator) Update(bar *models.Bar1m) (float64, error) {
    t.bars = append(t.bars, bar)
    // Keep rolling window
    if len(t.bars) > maxBars {
        t.bars = t.bars[1:]
    }
    
    // Call Java service when ready
    if len(t.bars) >= t.period {
        return t.client.CalculateRSI(t.symbol, t.bars)
    }
    return 0, nil
}
```

### Ta4j vs Techan Direct Comparison

| Feature | Ta4j (Java) | Techan (Go) |
|---------|-------------|-------------|
| **Indicators** | 130+ | 30+ |
| **Language** | Java | Go |
| **Integration** | Microservice | Direct (same process) |
| **Performance** | Network + JVM | In-process (fastest) |
| **Streaming** | ✅ BarSeries | ✅ Channels/Series |
| **Strategy Engine** | ✅ Built-in | ⚠️ Basic |
| **Backtesting** | ✅ Built-in | ❌ Need separate |
| **Deployment** | Java runtime needed | Single Go binary |
| **Memory** | JVM overhead (~100MB+) | Minimal |
| **Startup Time** | JVM warmup | Instant |

### When to Choose Ta4j

**Choose Ta4j if:**
- ✅ You need indicators not in Techan (check Ta4j's list)
- ✅ You want built-in strategy engine and backtesting
- ✅ You can accept microservice architecture
- ✅ Your team has Java expertise
- ✅ Network latency is acceptable (< 10ms localhost)
- ✅ You want more indicators than Techan offers

**Stick with Techan if:**
- ✅ Techan's 30+ indicators cover your needs
- ✅ You want everything in Go (simpler stack)
- ✅ Maximum performance is critical (no network overhead)
- ✅ You prefer single-language deployment
- ✅ You want minimal resource usage

### Ta4j Integration Strategy

**Option 1: Hybrid Approach (Recommended)**
```
Go Indicator Engine
├── Techan (30+ indicators) → In-process, fast
└── Ta4j Service (130+ indicators) → On-demand, slower
```

- Use Techan for common indicators (RSI, MACD, etc.)
- Use Ta4j service for rare/complex indicators
- Cache results to minimize network calls

**Option 2: Ta4j-Only Microservice**
```
Go Indicator Engine → Ta4j Service (all indicators)
```

- Simpler (one indicator source)
- But all indicators have network overhead
- Good if you need many indicators not in Techan

### Performance Benchmarks (Estimated)

Based on typical microservice performance:

| Operation | Techan (Go) | Ta4j (Java Service) | Python (TA-Lib Service) |
|-----------|-------------|---------------------|-------------------------|
| **RSI Calculation** | ~0.1ms | ~2-5ms | ~5-10ms |
| **MACD Calculation** | ~0.2ms | ~3-6ms | ~8-15ms |
| **Memory per Symbol** | ~1KB | ~5KB + JVM overhead | ~10KB + Python overhead |
| **Startup Time** | Instant | 2-5 seconds (JVM) | 1-3 seconds |

*Note: Network latency dominates microservice performance. Localhost gRPC adds ~1-2ms, HTTP adds ~2-5ms.*

---

## Detailed Comparison Matrix

| Solution | Indicator Count | Streaming Support | Performance | Integration Ease | Maintenance | Cost |
|----------|----------------|-------------------|-------------|------------------|-------------|------|
| **Techan** | 30+ | ✅ Excellent | ⭐⭐⭐⭐ | Medium | Active | Free |
| **Indicator** | 20+ | ✅ Good | ⭐⭐⭐⭐ | Medium | Active | Free |
| **Gotal** | 50+ | ⚠️ Partial | ⭐⭐⭐ | Medium-High | Limited | Free |
| **Banta** | 20+ | ✅ Excellent | ⭐⭐⭐⭐⭐ | Medium | New | Free |
| **Tulip (CGO)** | 100+ | ❌ Batch | ⭐⭐⭐ | High | Active | Free |
| **Ta4j (Java)** | 130+ | ✅ Excellent | ⭐⭐⭐ | Medium-High | Active | Free |
| **Python Service** | 200+ | ⚠️ Network | ⭐⭐ | Medium-High | Medium | Free |
| **Polygon API** | 50+ | ❌ Polling | ⭐⭐ | Low | Managed | $$$ |

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
- **Ta4j Java microservice** for additional indicators (better performance than Python)
- External API for historical analysis

### For Simplest Integration

**Best Choice: Indicator library** ⭐⭐⭐⭐
- Simpler API than Techan
- Backtesting included
- Pure Go

### For Backtesting/Historical Analysis

**Best Choice: Ta4j Java Service** ⭐⭐⭐⭐⭐
- Built-in backtesting framework
- Strategy engine included
- Better performance than Python
- 130+ indicators

**Alternative: Python Service with TA-Lib** ⭐⭐⭐⭐
- Full TA-Lib access (200+ indicators)
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

### Option B: Techan + Ta4j Java Service
- **Development Time:** 1 week
- **Performance:** Good (Techan) + Better (Java faster than Python)
- **Maintenance:** Medium (two systems, but Java more maintainable)
- **Indicator Coverage:** 160+ (Techan 30+ + Ta4j 130+)
- **Risk:** Medium

### Option B2: Techan + Python Service
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

**Alternative: Ta4j Java Microservice** (if you need more indicators)

**Reasons:**
1. ✅ 130+ indicators (4x more than Techan)
2. ✅ Streaming-friendly (BarSeries matches your architecture)
3. ✅ Better performance than Python
4. ✅ Built-in strategy engine and backtesting
5. ✅ Well-maintained and documented

**Trade-offs:**
- ❌ Requires microservice (network latency)
- ❌ JVM overhead (memory, startup)
- ❌ Deployment complexity (Java runtime)
- ❌ Language barrier (Java vs Go)

**Implementation Plan (Ta4j):**
1. Create Java microservice with Spring Boot/Quarkus (2-3 days)
2. Implement gRPC/HTTP API for indicator computation
3. Create Go client wrapper for your `Calculator` interface
4. Use for indicators not in Techan
5. Consider caching layer to reduce network calls

**Recommendation:** Start with Techan, add Ta4j service only if you need indicators not available in Techan.

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

