# Filters vs Techan Library Analysis

This document analyzes which filters from `Filters.md` can be implemented using the Techan library versus which require custom implementation.

## Summary

**Techan can help with:** ~5 filters (12%)
**Require custom implementation:** ~35 filters (88%)

Techan is useful for **technical indicators only**, but most filters are:
- Custom aggregations (volume, price changes)
- Range calculations (high/low tracking)
- External data (fundamental, news, earnings)
- Time-based calculations
- Pattern detection

---

## Detailed Analysis by Category

### 1. Volume Filters (7 filters) ❌ **Cannot use Techan**

All volume filters require **custom aggregation logic**, not technical indicators:

| Filter | Techan? | Reason |
|--------|---------|--------|
| 1.1 Postmarket Volume | ❌ | Custom session-based volume tracking |
| 1.2 Premarket Volume | ❌ | Custom session-based volume tracking |
| 1.3 Absolute Volume | ❌ | Simple sum of volume over timeframe |
| 1.4 Absolute Dollar Volume | ❌ | Simple sum of (price × volume) |
| 1.5 Average Volume | ❌ | Average of daily volumes (custom aggregation) |
| 1.6 Relative Volume (%) | ❌ | Custom ratio calculation with forecasting |
| 1.7 Relative Volume at Same Time | ❌ | Custom time-of-day pattern matching |

**Implementation:** Custom logic in scanner/metrics computation layer.

---

### 2. Price Filters (8 filters) ❌ **Cannot use Techan**

All price filters are **simple price calculations**, not technical indicators:

| Filter | Techan? | Reason |
|--------|---------|--------|
| 2.1 Price ($) | ❌ | Current price (already available) |
| 2.2 Change ($) | ❌ | Simple subtraction: `current - price_N_minutes_ago` |
| 2.3 Change from Close | ❌ | Simple subtraction: `current - yesterday_close` |
| 2.4 Change from Close (Premarket) | ❌ | Same as 2.3 with session check |
| 2.5 Change from Close (Post Market) | ❌ | Same as 2.3 with session check |
| 2.6 Change from Open | ❌ | Simple subtraction: `current - today_open` |
| 2.7 Percentage Change (%) | ❌ | Simple percentage: `((current - old) / old) * 100` |
| 2.8 Gap from Close | ❌ | Simple subtraction: `today_open - yesterday_close` |

**Implementation:** Custom logic in metrics computation (already partially implemented).

---

### 3. Range Filters (5 filters) ❌ **Cannot use Techan**

All range filters require **high/low tracking over timeframes**, not technical indicators:

| Filter | Techan? | Reason |
|--------|---------|--------|
| 3.1 Range ($) | ❌ | Custom: `max(high) - min(low)` over timeframe |
| 3.2 Percentage Range (%) | ❌ | Custom: `((high - low) / low) * 100` |
| 3.3 Biggest Range (%) | ❌ | Custom: Maximum range over historical period |
| 3.4 Relative Range (%) | ⚠️ **Partial** | Needs ATR(14) from Techan + custom range calculation |
| 3.5 Position in Range (%) | ❌ | Custom: `((current - low) / (high - low)) * 100` |

**Note:** Filter 3.4 (Relative Range) can use Techan's ATR, but still needs custom range calculation.

**Implementation:** Custom logic tracking high/low over timeframes.

---

### 4. Technical Indicator Filters (5 filters) ✅ **Can use Techan**

These are actual **technical indicators** that Techan provides:

| Filter | Techan? | Techan Indicator | Notes |
|--------|---------|-----------------|-------|
| 4.1 RSI (14) | ✅ | `RSIIndicator` | Techan has RSI, supports different periods |
| 4.2 ATR (Average True Range) | ✅ | `ATRIndicator` | Techan has ATR, supports different periods |
| 4.3 ATRP (ATR Percentage) | ⚠️ **Partial** | ATR + custom | Use Techan ATR, then calculate `(ATR / Close) * 100` |
| 4.4 Distance from VWAP | ❌ | N/A | VWAP is custom (already implemented), distance is simple math |
| 4.5 Distance from Moving Average | ✅ | `EMAIndicator`, `SMAIndicator` | Techan has EMA/SMA, distance is simple math |

**Implementation:**
- RSI, ATR, EMA, SMA: Use Techan directly
- ATRP: Use Techan ATR + custom percentage calculation
- VWAP Distance: Custom (VWAP already implemented)
- MA Distance: Use Techan MA + custom distance calculation

---

### 5. Trading Activity Filters (2 filters) ❌ **Cannot use Techan**

These require **pattern detection and counting**, not technical indicators:

| Filter | Techan? | Reason |
|--------|---------|--------|
| 5.1 Trade Count | ❌ | Custom tick counting over timeframe |
| 5.2 Consecutive Candles | ❌ | Custom pattern detection (green/red candles) |

**Implementation:** Custom logic tracking trades and candle patterns.

---

### 6. Time-Based Filters (5 filters) ❌ **Cannot use Techan**

These are **time calculations**, not technical indicators:

| Filter | Techan? | Reason |
|--------|---------|--------|
| 6.1 Minutes in Market | ❌ | Simple time calculation: `now - market_open` |
| 6.2 Minutes Since News | ❌ | Requires external news data + time calculation |
| 6.3 Hours Since News | ❌ | Requires external news data + time calculation |
| 6.4 Days Since News | ❌ | Requires external news data + time calculation |
| 6.5 Days Until Earnings | ❌ | Requires external earnings calendar + time calculation |

**Implementation:** Custom time calculations + external data integration.

---

### 7. Fundamental Data Filters (6 filters) ❌ **Cannot use Techan**

These require **external data sources**, not technical indicators:

| Filter | Techan? | Reason |
|--------|---------|--------|
| 7.1 Institutional Ownership | ❌ | External fundamental data provider |
| 7.2 MarketCap | ❌ | External fundamental data provider |
| 7.3 Shares Outstanding | ❌ | External fundamental data provider |
| 7.4 Short Interest (%) | ❌ | External short interest data provider |
| 7.5 Short Ratio | ❌ | External data + custom calculation |
| 7.6 Float | ❌ | External fundamental data provider |

**Implementation:** External API integration + caching layer.

---

## Techan Indicators Available

Based on Techan documentation, here are the indicators Techan provides:

### ✅ Available in Techan
- **RSI** (Relative Strength Index) - Multiple periods
- **ATR** (Average True Range) - Multiple periods
- **EMA** (Exponential Moving Average) - Multiple periods
- **SMA** (Simple Moving Average) - Multiple periods
- **MACD** (Moving Average Convergence Divergence)
- **Bollinger Bands** (Upper, Middle, Lower)
- **Stochastic Oscillator**
- **Ichimoku Cloud**
- **Volume indicators** (basic volume tracking)

### ❌ Not in Techan (but might be useful)
- **VWAP** (Volume Weighted Average Price) - Custom implementation needed
- **ADX** (Average Directional Index) - Not in Techan
- **Parabolic SAR** - Not in Techan
- **Williams %R** - Not in Techan

---

## Implementation Strategy

### Phase 1: Use Techan for Technical Indicators

**Filters that benefit from Techan:**
1. **RSI (14)** - Filter 4.1
   - Use: `techan.NewRSIIndicator(closePrice, 14)`
   - Support multiple timeframes (1m, 5m, 15m, daily)

2. **ATR (14)** - Filter 4.2
   - Use: `techan.NewATRIndicator(series, 14)`
   - Support multiple timeframes

3. **ATRP (14)** - Filter 4.3
   - Use Techan ATR + custom: `(ATR / Close) * 100`

4. **Distance from Moving Average** - Filter 4.5
   - Use: `techan.NewEMAIndicator(closePrice, period)` or `techan.NewSMAIndicator(closePrice, period)`
   - Custom: Calculate distance `((price - ma) / ma) * 100`

5. **Relative Range** - Filter 3.4 (partial)
   - Use Techan ATR(14) daily
   - Custom: Calculate today's range and ratio

### Phase 2: Custom Implementation for Everything Else

**Volume Filters (7):**
- Implement custom volume aggregations
- Track session-based volumes
- Calculate relative volumes

**Price Filters (8):**
- Simple price calculations (already partially implemented)
- Store yesterday's close, today's open
- Calculate changes and gaps

**Range Filters (5):**
- Track high/low over timeframes
- Calculate ranges and positions
- Store historical range data

**Trading Activity (2):**
- Count trades per timeframe
- Detect consecutive candle patterns

**Time-Based (5):**
- Calculate time differences
- Integrate news/earnings data

**Fundamental Data (6):**
- Integrate external data providers
- Cache fundamental data

---

## Code Examples

### Using Techan for RSI

```go
// pkg/indicator/techan_rsi.go
import "github.com/sdcoffey/techan"

func CreateTechanRSI(period int) CalculatorFactory {
    return func() (Calculator, error) {
        series := techan.NewTimeSeries()
        closePrice := techan.NewClosePriceIndicator(series)
        rsi := techan.NewRSIIndicator(closePrice, period)
        
        return &TechanCalculator{
            name:      fmt.Sprintf("rsi_%d", period),
            series:    series,
            indicator: rsi,
            period:    period,
        }, nil
    }
}
```

### Using Techan for ATR

```go
// pkg/indicator/techan_atr.go
func CreateTechanATR(period int) CalculatorFactory {
    return func() (Calculator, error) {
        series := techan.NewTimeSeries()
        atr := techan.NewATRIndicator(series, period)
        
        return &TechanCalculator{
            name:      fmt.Sprintf("atr_%d", period),
            series:    series,
            indicator: atr,
            period:    period,
        }, nil
    }
}
```

### Custom Implementation for Volume Filters

```go
// internal/metrics/volume_metrics.go
func CalculateAbsoluteVolume(bars []*models.Bar1m, timeframe time.Duration) float64 {
    cutoff := time.Now().Add(-timeframe)
    var totalVolume int64
    
    for _, bar := range bars {
        if bar.Timestamp.After(cutoff) {
            totalVolume += bar.Volume
        }
    }
    
    return float64(totalVolume)
}
```

### Custom Implementation for Range Filters

```go
// internal/metrics/range_metrics.go
func CalculateRange(bars []*models.Bar1m, timeframe time.Duration) (float64, float64) {
    cutoff := time.Now().Add(-timeframe)
    var high, low float64
    
    for _, bar := range bars {
        if bar.Timestamp.After(cutoff) {
            if high == 0 || bar.High > high {
                high = bar.High
            }
            if low == 0 || bar.Low < low {
                low = bar.Low
            }
        }
    }
    
    return high, low
}
```

---

## Conclusion

**Techan can help with ~5 filters (12%):**
- RSI (4.1) ✅
- ATR (4.2) ✅
- ATRP (4.3) ⚠️ (partial)
- Distance from MA (4.5) ✅
- Relative Range (3.4) ⚠️ (partial - uses ATR)

**All other filters (~35 filters, 88%) require custom implementation:**
- Volume aggregations
- Price calculations
- Range tracking
- Pattern detection
- Time calculations
- External data integration

**Recommendation:**
1. **Use Techan** for technical indicators (RSI, ATR, EMA, SMA)
2. **Implement custom logic** for all other filters
3. **Combine both** where appropriate (e.g., ATRP uses Techan ATR + custom percentage)

Techan is a **valuable addition** for technical indicators, but it's **not a complete solution** for all filters. Most filters are simple calculations or aggregations that don't require a technical analysis library.

