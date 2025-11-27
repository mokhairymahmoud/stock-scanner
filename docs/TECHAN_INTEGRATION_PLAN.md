# Techan Integration Plan (Updated for All Filters)

## Overview

This document outlines the plan to integrate the Techan library for technical indicators and implement all filters from `Filters.md` using a two-tier system:
1. **Indicators** (Techan/Custom) - Computed by Indicator Engine
2. **Metrics** (Computed Filters) - Computed by Metrics Registry

## Architecture Overview

### Two-Tier System

```
┌─────────────────────────────────────────────────────────┐
│                    Filter System                        │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌──────────────────────┐  ┌──────────────────────┐  │
│  │  Indicator Engine     │  │  Metrics Registry    │  │
│  │  (Technical Indicators)│  │  (Computed Metrics)  │  │
│  ├──────────────────────┤  ├──────────────────────┤  │
│  │ • RSI (Techan)        │  │ • Volume Filters     │  │
│  │ • EMA (Techan)        │  │ • Price Filters      │  │
│  │ • SMA (Techan)        │  │ • Range Filters      │  │
│  │ • MACD (Techan)       │  │ • Time-Based Filters │  │
│  │ • ATR (Techan)        │  │ • Trading Activity   │  │
│  │ • VWAP (Custom)       │  │ • Fundamental Data   │  │
│  │ • Volume Avg (Custom) │  │ • Distance Metrics   │  │
│  └──────────────────────┘  └──────────────────────┘  │
│           │                          │                │
│           └──────────┬─────────────────┘                │
│                     │                                   │
│            ┌────────▼────────┐                         │
│            │  Symbol State   │                         │
│            │  (Indicators +  │                         │
│            │   State Data)   │                         │
│            └─────────────────┘                         │
│                     │                                   │
│            ┌────────▼────────┐                         │
│            │  Metrics Map    │                         │
│            │  (For Rules)     │                         │
│            └─────────────────┘                         │
└─────────────────────────────────────────────────────────┘
```

### Key Distinctions

**Indicators** (Indicator Engine):
- Technical analysis indicators (RSI, EMA, SMA, MACD, ATR, etc.)
- Computed incrementally from bars
- Stored in `SymbolState.Indicators`
- Use Techan library where possible
- Custom implementations for VWAP, Volume Average

**Metrics** (Metrics Registry):
- Computed filters (volume, price changes, ranges, etc.)
- Computed on-demand from symbol state
- Not stored, computed when needed for rule evaluation
- All custom implementations
- Use `MetricComputer` interface

## Filter Implementation Mapping

### Indicators (Indicator Engine) - Use Techan

| Filter | Type | Implementation |
|--------|------|----------------|
| 4.1 RSI (14) | Indicator | Techan `RSIIndicator` |
| 4.2 ATR (14) | Indicator | Techan `ATRIndicator` |
| 4.3 ATRP (14) | Indicator | Techan ATR + Custom % calculation |
| 4.5 Distance from MA | Indicator | Techan EMA/SMA + Custom distance |

### Metrics (Metrics Registry) - Custom Implementation

| Category | Filters | Implementation |
|----------|---------|----------------|
| Volume (7) | All volume filters | `MetricComputer` implementations |
| Price (8) | All price filters | `MetricComputer` implementations |
| Range (5) | All range filters | `MetricComputer` implementations |
| Trading Activity (2) | Trade count, Consecutive candles | `MetricComputer` implementations |
| Time-Based (5) | Minutes in market, News time, etc. | `MetricComputer` implementations |
| Fundamental (6) | MarketCap, Float, etc. | `MetricComputer` + External data |

## Implementation Plan

### Phase 1: Techan Integration for Indicators

#### 1.1 Add Techan Dependency
```bash
go get github.com/sdcoffey/techan
```

#### 1.2 Create Techan Adapter
**File:** `pkg/indicator/techan_adapter.go`

```go
package indicator

import (
    "fmt"
    "time"
    
    "github.com/mohamedkhairy/stock-scanner/internal/models"
    "github.com/sdcoffey/techan"
)

// TechanCalculator wraps a Techan indicator to implement Calculator interface
type TechanCalculator struct {
    name      string
    series    *techan.TimeSeries
    indicator techan.Indicator
    ready     bool
    period    int
}

// NewTechanCalculator creates a new Techan-based calculator
func NewTechanCalculator(
    name string,
    indicator techan.Indicator,
    period int,
) *TechanCalculator {
    return &TechanCalculator{
        name:      name,
        series:    techan.NewTimeSeries(),
        indicator: indicator,
        period:    period,
        ready:     false,
    }
}

func (t *TechanCalculator) Name() string {
    return t.name
}

func (t *TechanCalculator) Update(bar *models.Bar1m) (float64, error) {
    if bar == nil {
        return 0, fmt.Errorf("bar cannot be nil")
    }
    
    // Convert Bar1m to techan.Candle
    timePeriod := techan.NewTimePeriod(bar.Timestamp, time.Minute)
    candle := techan.NewCandle(timePeriod)
    
    candle.OpenPrice = techan.NewDecimal(bar.Open)
    candle.HighPrice = techan.NewDecimal(bar.High)
    candle.LowPrice = techan.NewDecimal(bar.Low)
    candle.ClosePrice = techan.NewDecimal(bar.Close)
    candle.Volume = techan.NewDecimal(float64(bar.Volume))
    
    t.series.AddCandle(candle)
    
    // Check if we have enough data
    lastIndex := t.series.LastIndex()
    if lastIndex >= t.period-1 {
        t.ready = true
        value := t.indicator.Calculate(lastIndex)
        return value.Float(), nil
    }
    
    return 0, nil
}

func (t *TechanCalculator) Value() (float64, error) {
    if !t.ready {
        return 0, fmt.Errorf("indicator not ready: need at least %d bars", t.period)
    }
    lastIndex := t.series.LastIndex()
    value := t.indicator.Calculate(lastIndex)
    return value.Float(), nil
}

func (t *TechanCalculator) Reset() {
    t.series = techan.NewTimeSeries()
    t.ready = false
}

func (t *TechanCalculator) IsReady() bool {
    return t.ready
}
```

#### 1.3 Create Techan Factory Functions
**File:** `pkg/indicator/techan_factory.go`

```go
package indicator

import (
    "fmt"
    "github.com/sdcoffey/techan"
)

// CalculatorFactory is a function that creates a new calculator instance
type CalculatorFactory func() (Calculator, error)

// CreateTechanRSI creates an RSI indicator using Techan
func CreateTechanRSI(period int) CalculatorFactory {
    return func() (Calculator, error) {
        series := techan.NewTimeSeries()
        closePrice := techan.NewClosePriceIndicator(series)
        rsi := techan.NewRSIIndicator(closePrice, period)
        
        return NewTechanCalculator(
            fmt.Sprintf("rsi_%d", period),
            rsi,
            period,
        ), nil
    }
}

// CreateTechanEMA creates an EMA indicator using Techan
func CreateTechanEMA(period int) CalculatorFactory {
    return func() (Calculator, error) {
        series := techan.NewTimeSeries()
        closePrice := techan.NewClosePriceIndicator(series)
        ema := techan.NewEMAIndicator(closePrice, period)
        
        return NewTechanCalculator(
            fmt.Sprintf("ema_%d", period),
            ema,
            period,
        ), nil
    }
}

// CreateTechanSMA creates an SMA indicator using Techan
func CreateTechanSMA(period int) CalculatorFactory {
    return func() (Calculator, error) {
        series := techan.NewTimeSeries()
        closePrice := techan.NewClosePriceIndicator(series)
        sma := techan.NewSMAIndicator(closePrice, period)
        
        return NewTechanCalculator(
            fmt.Sprintf("sma_%d", period),
            sma,
            period,
        ), nil
    }
}

// CreateTechanMACD creates a MACD indicator using Techan
func CreateTechanMACD(fastPeriod, slowPeriod, signalPeriod int) CalculatorFactory {
    return func() (Calculator, error) {
        series := techan.NewTimeSeries()
        closePrice := techan.NewClosePriceIndicator(series)
        macd := techan.NewMACDIndicator(closePrice, fastPeriod, slowPeriod, signalPeriod)
        
        return NewTechanCalculator(
            fmt.Sprintf("macd_%d_%d_%d", fastPeriod, slowPeriod, signalPeriod),
            macd,
            slowPeriod,
        ), nil
    }
}

// CreateTechanATR creates an ATR indicator using Techan
func CreateTechanATR(period int) CalculatorFactory {
    return func() (Calculator, error) {
        series := techan.NewTimeSeries()
        atr := techan.NewATRIndicator(series, period)
        
        return NewTechanCalculator(
            fmt.Sprintf("atr_%d", period),
            atr,
            period,
        ), nil
    }
}

// CreateTechanBollingerBands creates Bollinger Bands using Techan
func CreateTechanBollingerBands(period int, multiplier float64) CalculatorFactory {
    return func() (Calculator, error) {
        series := techan.NewTimeSeries()
        closePrice := techan.NewClosePriceIndicator(series)
        sma := techan.NewSMAIndicator(closePrice, period)
        bb := techan.NewBollingerBandsIndicator(sma, period, techan.NewDecimal(multiplier))
        
        return NewTechanCalculator(
            fmt.Sprintf("bb_%d_%.1f", period, multiplier),
            sma, // Use SMA as the main indicator
            period,
        ), nil
    }
}

// CreateTechanStochastic creates a Stochastic Oscillator using Techan
func CreateTechanStochastic(kPeriod, dPeriod, smoothK int) CalculatorFactory {
    return func() (Calculator, error) {
        series := techan.NewTimeSeries()
        stochastic := techan.NewStochasticIndicator(series, kPeriod, dPeriod, smoothK)
        
        return NewTechanCalculator(
            fmt.Sprintf("stoch_%d_%d_%d", kPeriod, dPeriod, smoothK),
            stochastic,
            kPeriod,
        ), nil
    }
}
```

### Phase 2: Indicator Registry System

#### 2.1 Create Indicator Registry
**File:** `internal/indicator/registry.go`

```go
package indicator

import (
    "fmt"
    "sync"
    
    indicatorpkg "github.com/mohamedkhairy/stock-scanner/pkg/indicator"
)

// IndicatorRegistry manages all available indicators (Techan + Custom)
type IndicatorRegistry struct {
    mu        sync.RWMutex
    factories map[string]CalculatorFactory
    metadata  map[string]IndicatorMetadata
}

// IndicatorMetadata contains information about an indicator
type IndicatorMetadata struct {
    Name        string
    Type        string // "techan", "custom"
    Description string
    Parameters  map[string]interface{}
    Category    string // "momentum", "trend", "volatility", "volume", "price"
}

// NewIndicatorRegistry creates a new indicator registry
func NewIndicatorRegistry() *IndicatorRegistry {
    return &IndicatorRegistry{
        factories: make(map[string]CalculatorFactory),
        metadata:  make(map[string]IndicatorMetadata),
    }
}

// Register registers an indicator factory
func (r *IndicatorRegistry) Register(
    name string,
    factory CalculatorFactory,
    metadata IndicatorMetadata,
) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    if _, exists := r.factories[name]; exists {
        return fmt.Errorf("indicator %q already registered", name)
    }
    
    r.factories[name] = factory
    r.metadata[name] = metadata
    return nil
}

// GetFactory returns a factory for an indicator
func (r *IndicatorRegistry) GetFactory(name string) (CalculatorFactory, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    factory, exists := r.factories[name]
    return factory, exists
}

// ListAvailable returns all available indicator names
func (r *IndicatorRegistry) ListAvailable() []string {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    names := make([]string, 0, len(r.factories))
    for name := range r.factories {
        names = append(names, name)
    }
    return names
}

// GetMetadata returns metadata for an indicator
func (r *IndicatorRegistry) GetMetadata(name string) (IndicatorMetadata, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    metadata, exists := r.metadata[name]
    return metadata, exists
}
```

#### 2.2 Register All Indicators
**File:** `internal/indicator/indicator_registration.go`

```go
package indicator

import (
    "fmt"
    "time"
    
    indicatorpkg "github.com/mohamedkhairy/stock-scanner/pkg/indicator"
)

// RegisterAllIndicators registers all available indicators (Techan + Custom)
func RegisterAllIndicators(registry *IndicatorRegistry) error {
    // Register Techan indicators
    if err := registerTechanIndicators(registry); err != nil {
        return err
    }
    
    // Register custom indicators (not in Techan)
    if err := registerCustomIndicators(registry); err != nil {
        return err
    }
    
    return nil
}

// registerTechanIndicators registers all Techan indicators
func registerTechanIndicators(registry *IndicatorRegistry) error {
    // RSI indicators
    rsiPeriods := []int{9, 14, 21}
    for _, period := range rsiPeriods {
        name := fmt.Sprintf("rsi_%d", period)
        if err := registry.Register(name,
            indicatorpkg.CreateTechanRSI(period),
            IndicatorMetadata{
                Name:        name,
                Type:        "techan",
                Description: fmt.Sprintf("Relative Strength Index (%d period)", period),
                Category:    "momentum",
                Parameters:  map[string]interface{}{"period": period},
            },
        ); err != nil {
            return err
        }
    }
    
    // EMA indicators
    emaPeriods := []int{9, 12, 20, 21, 26, 50, 200}
    for _, period := range emaPeriods {
        name := fmt.Sprintf("ema_%d", period)
        if err := registry.Register(name,
            indicatorpkg.CreateTechanEMA(period),
            IndicatorMetadata{
                Name:        name,
                Type:        "techan",
                Description: fmt.Sprintf("Exponential Moving Average (%d period)", period),
                Category:    "trend",
                Parameters:  map[string]interface{}{"period": period},
            },
        ); err != nil {
            return err
        }
    }
    
    // SMA indicators
    smaPeriods := []int{10, 20, 50, 200}
    for _, period := range smaPeriods {
        name := fmt.Sprintf("sma_%d", period)
        if err := registry.Register(name,
            indicatorpkg.CreateTechanSMA(period),
            IndicatorMetadata{
                Name:        name,
                Type:        "techan",
                Description: fmt.Sprintf("Simple Moving Average (%d period)", period),
                Category:    "trend",
                Parameters:  map[string]interface{}{"period": period},
            },
        ); err != nil {
            return err
        }
    }
    
    // MACD
    if err := registry.Register("macd_12_26_9",
        indicatorpkg.CreateTechanMACD(12, 26, 9),
        IndicatorMetadata{
            Name:        "macd_12_26_9",
            Type:        "techan",
            Description: "MACD (12, 26, 9)",
            Category:    "trend",
            Parameters: map[string]interface{}{
                "fast_period":  12,
                "slow_period":  26,
                "signal_period": 9,
            },
        },
    ); err != nil {
        return err
    }
    
    // ATR
    atrPeriods := []int{14}
    for _, period := range atrPeriods {
        name := fmt.Sprintf("atr_%d", period)
        if err := registry.Register(name,
            indicatorpkg.CreateTechanATR(period),
            IndicatorMetadata{
                Name:        name,
                Type:        "techan",
                Description: fmt.Sprintf("Average True Range (%d period)", period),
                Category:    "volatility",
                Parameters:  map[string]interface{}{"period": period},
            },
        ); err != nil {
            return err
        }
    }
    
    // Bollinger Bands
    if err := registry.Register("bb_20_2.0",
        indicatorpkg.CreateTechanBollingerBands(20, 2.0),
        IndicatorMetadata{
            Name:        "bb_20_2.0",
            Type:        "techan",
            Description: "Bollinger Bands (20 period, 2.0 std dev)",
            Category:    "volatility",
            Parameters: map[string]interface{}{
                "period":    20,
                "multiplier": 2.0,
            },
        },
    ); err != nil {
        return err
    }
    
    // Stochastic
    if err := registry.Register("stoch_14_3_3",
        indicatorpkg.CreateTechanStochastic(14, 3, 3),
        IndicatorMetadata{
            Name:        "stoch_14_3_3",
            Type:        "techan",
            Description: "Stochastic Oscillator (14, 3, 3)",
            Category:    "momentum",
            Parameters: map[string]interface{}{
                "k_period": 14,
                "d_period": 3,
                "smooth_k": 3,
            },
        },
    ); err != nil {
        return err
    }
    
    return nil
}

// registerCustomIndicators registers custom indicators (not in Techan)
func registerCustomIndicators(registry *IndicatorRegistry) error {
    // VWAP indicators
    vwapWindows := []time.Duration{
        5 * time.Minute,
        15 * time.Minute,
        1 * time.Hour,
    }
    for _, window := range vwapWindows {
        name := fmt.Sprintf("vwap_%s", formatDuration(window))
        window := window // Capture loop variable
        if err := registry.Register(name,
            func() (indicatorpkg.Calculator, error) {
                return indicatorpkg.NewVWAP(window)
            },
            IndicatorMetadata{
                Name:        name,
                Type:        "custom",
                Description: fmt.Sprintf("Volume Weighted Average Price (%s window)", window),
                Category:    "price",
                Parameters:  map[string]interface{}{"window": window.String()},
            },
        ); err != nil {
            return err
        }
    }
    
    // Volume average indicators
    volumeWindows := []time.Duration{
        5 * time.Minute,
        15 * time.Minute,
        1 * time.Hour,
    }
    for _, window := range volumeWindows {
        name := fmt.Sprintf("volume_avg_%s", formatDuration(window))
        window := window
        if err := registry.Register(name,
            func() (indicatorpkg.Calculator, error) {
                return indicatorpkg.NewVolumeAverage(window)
            },
            IndicatorMetadata{
                Name:        name,
                Type:        "custom",
                Description: fmt.Sprintf("Average Volume (%s window)", window),
                Category:    "volume",
                Parameters:  map[string]interface{}{"window": window.String()},
            },
        ); err != nil {
            return err
        }
    }
    
    return nil
}

// formatDuration formats a duration for use in indicator names
func formatDuration(d time.Duration) string {
    minutes := int(d.Minutes())
    if minutes < 60 {
        return fmt.Sprintf("%dm", minutes)
    }
    hours := minutes / 60
    if hours < 24 {
        return fmt.Sprintf("%dh", hours)
    }
    days := hours / 24
    return fmt.Sprintf("%dd", days)
}
```

### Phase 3: Metrics Registry for Filters

#### 3.1 Extend Metrics Registry
**File:** `internal/metrics/registry.go` (enhance existing)

The existing `MetricComputer` interface is perfect for implementing filters. We need to:

1. **Create metric computers for all filter categories**
2. **Register them in the metrics registry**
3. **Ensure proper dependency ordering**

#### 3.2 Create Filter Metric Computers

**Volume Filters:**
- `internal/metrics/volume_filters.go`
  - `PostmarketVolumeComputer`
  - `PremarketVolumeComputer`
  - `AbsoluteVolumeComputer`
  - `AbsoluteDollarVolumeComputer`
  - `AverageVolumeComputer`
  - `RelativeVolumeComputer`
  - `RelativeVolumeSameTimeComputer`

**Price Filters:**
- `internal/metrics/price_filters.go`
  - `ChangeComputer`
  - `ChangeFromCloseComputer`
  - `ChangeFromOpenComputer`
  - `PercentageChangeComputer`
  - `GapFromCloseComputer`

**Range Filters:**
- `internal/metrics/range_filters.go`
  - `RangeComputer`
  - `PercentageRangeComputer`
  - `BiggestRangeComputer`
  - `RelativeRangeComputer` (uses ATR from indicators)
  - `PositionInRangeComputer`

**Trading Activity Filters:**
- `internal/metrics/trading_activity_filters.go`
  - `TradeCountComputer`
  - `ConsecutiveCandlesComputer`

**Time-Based Filters:**
- `internal/metrics/time_filters.go`
  - `MinutesInMarketComputer`
  - `MinutesSinceNewsComputer`
  - `HoursSinceNewsComputer`
  - `DaysSinceNewsComputer`
  - `DaysUntilEarningsComputer`

**Fundamental Data Filters:**
- `internal/metrics/fundamental_filters.go`
  - `InstitutionalOwnershipComputer`
  - `MarketCapComputer`
  - `SharesOutstandingComputer`
  - `ShortInterestComputer`
  - `ShortRatioComputer`
  - `FloatComputer`

**Distance Metrics:**
- `internal/metrics/distance_filters.go`
  - `VWAPDistanceComputer` (uses VWAP from indicators)
  - `MADistanceComputer` (uses EMA/SMA from indicators)
  - `ATRPComputer` (uses ATR from indicators)

#### 3.3 Example Metric Computer Implementation

**File:** `internal/metrics/volume_filters.go`

```go
package metrics

import (
    "time"
)

// AbsoluteVolumeComputer computes absolute volume over a timeframe
type AbsoluteVolumeComputer struct {
    name      string
    timeframe time.Duration
}

// NewAbsoluteVolumeComputer creates a new absolute volume computer
func NewAbsoluteVolumeComputer(timeframe time.Duration) *AbsoluteVolumeComputer {
    return &AbsoluteVolumeComputer{
        name:      formatVolumeMetricName("volume", timeframe),
        timeframe: timeframe,
    }
}

func (c *AbsoluteVolumeComputer) Name() string {
    return c.name
}

func (c *AbsoluteVolumeComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
    if len(snapshot.LastFinalBars) == 0 {
        return 0, false
    }
    
    cutoff := time.Now().Add(-c.timeframe)
    var totalVolume int64
    
    for _, bar := range snapshot.LastFinalBars {
        if bar.Timestamp.After(cutoff) {
            totalVolume += bar.Volume
        }
    }
    
    // Also include live bar if it's within timeframe
    if snapshot.LiveBar != nil {
        liveBarTime := snapshot.LiveBar.Timestamp
        if liveBarTime.After(cutoff) {
            totalVolume += snapshot.LiveBar.Volume
        }
    }
    
    return float64(totalVolume), true
}

func (c *AbsoluteVolumeComputer) Dependencies() []string {
    return []string{} // No dependencies
}

// formatVolumeMetricName formats volume metric names
func formatVolumeMetricName(base string, timeframe time.Duration) string {
    minutes := int(timeframe.Minutes())
    if minutes < 60 {
        return fmt.Sprintf("%s_%dm", base, minutes)
    }
    hours := minutes / 60
    if hours < 24 {
        return fmt.Sprintf("%s_%dh", base, hours)
    }
    days := hours / 24
    return fmt.Sprintf("%s_%dd", base, days)
}
```

### Phase 4: Requirement Discovery (Indicators + Metrics)

#### 4.1 Enhanced Requirement Tracker
**File:** `internal/indicator/requirement_tracker.go`

```go
package indicator

import (
    "context"
    "sync"
    "time"
    
    "github.com/mohamedkhairy/stock-scanner/internal/models"
    "github.com/mohamedkhairy/stock-scanner/internal/rules"
    "github.com/mohamedkhairy/stock-scanner/internal/toplist"
    "github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

// RequirementTracker tracks which indicators and metrics are required
type RequirementTracker struct {
    mu                sync.RWMutex
    requiredIndicators map[string]bool // indicator name -> required
    requiredMetrics    map[string]bool // metric name -> required
    toplistStore      toplist.ToplistStore
    ruleStore         rules.RuleStore
    reloadInterval    time.Duration
    lastReload        time.Time
}

// NewRequirementTracker creates a new requirement tracker
func NewRequirementTracker(
    toplistStore toplist.ToplistStore,
    ruleStore rules.RuleStore,
    reloadInterval time.Duration,
) *RequirementTracker {
    return &RequirementTracker{
        requiredIndicators: make(map[string]bool),
        requiredMetrics:    make(map[string]bool),
        toplistStore:       toplistStore,
        ruleStore:          ruleStore,
        reloadInterval:     reloadInterval,
    }
}

// GetRequiredIndicators returns the set of required indicator names
func (rt *RequirementTracker) GetRequiredIndicators() map[string]bool {
    rt.mu.RLock()
    defer rt.mu.RUnlock()
    
    result := make(map[string]bool)
    for name, required := range rt.requiredIndicators {
        result[name] = required
    }
    return result
}

// GetRequiredMetrics returns the set of required metric names
func (rt *RequirementTracker) GetRequiredMetrics() map[string]bool {
    rt.mu.RLock()
    defer rt.mu.RUnlock()
    
    result := make(map[string]bool)
    for name, required := range rt.requiredMetrics {
        result[name] = required
    }
    return result
}

// ReloadRequirements discovers required indicators and metrics from toplists and rules
func (rt *RequirementTracker) ReloadRequirements(ctx context.Context) error {
    rt.mu.Lock()
    defer rt.mu.Unlock()
    
    requiredIndicators := make(map[string]bool)
    requiredMetrics := make(map[string]bool)
    
    // Discover from toplists
    if rt.toplistStore != nil {
        toplists, err := rt.toplistStore.GetEnabledToplists(ctx, "")
        if err == nil {
            mapper := toplist.NewMetricMapper()
            for _, config := range toplists {
                metricName := mapper.GetMetricName(config)
                if metricName != "" {
                    // Check if it's an indicator or metric
                    if isIndicator(metricName) {
                        requiredIndicators[metricName] = true
                    } else {
                        requiredMetrics[metricName] = true
                    }
                }
                // Special handling for VWAP distance
                if config.Metric == models.MetricVWAPDist {
                    var vwapKey string
                    switch config.TimeWindow {
                    case models.Window5m:
                        vwapKey = "vwap_5m"
                    case models.Window15m:
                        vwapKey = "vwap_15m"
                    case models.Window1h:
                        vwapKey = "vwap_1h"
                    }
                    if vwapKey != "" {
                        requiredIndicators[vwapKey] = true
                    }
                    requiredMetrics["vwap_dist"] = true
                }
            }
        } else {
            logger.Warn("Failed to load toplists for requirement discovery",
                logger.ErrorField(err),
            )
        }
    }
    
    // Discover from rules
    if rt.ruleStore != nil {
        rules, err := rt.ruleStore.GetAllRules()
        if err == nil {
            for _, rule := range rules {
                if !rule.Enabled {
                    continue
                }
                for _, condition := range rule.Conditions {
                    metricName := condition.Metric
                    // Check if it's an indicator or metric
                    if isIndicator(metricName) {
                        requiredIndicators[metricName] = true
                    } else {
                        requiredMetrics[metricName] = true
                    }
                }
            }
        } else {
            logger.Warn("Failed to load rules for requirement discovery",
                logger.ErrorField(err),
            )
        }
    }
    
    rt.requiredIndicators = requiredIndicators
    rt.requiredMetrics = requiredMetrics
    rt.lastReload = time.Now()
    
    logger.Info("Reloaded requirements",
        logger.Int("indicators", len(requiredIndicators)),
        logger.Int("metrics", len(requiredMetrics)),
    )
    
    return nil
}

// isIndicator checks if a metric name refers to an indicator
// Indicators: rsi_14, ema_20, sma_50, macd_12_26_9, atr_14, vwap_5m, etc.
func isIndicator(name string) bool {
    // List of indicator prefixes
    indicatorPrefixes := []string{
        "rsi_", "ema_", "sma_", "macd_", "atr_", "vwap_", "volume_avg_",
        "bb_", "stoch_", "ichimoku_",
    }
    
    for _, prefix := range indicatorPrefixes {
        if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
            return true
        }
    }
    
    return false
}

// ShouldReload checks if requirements should be reloaded
func (rt *RequirementTracker) ShouldReload() bool {
    rt.mu.RLock()
    defer rt.mu.RUnlock()
    return time.Since(rt.lastReload) >= rt.reloadInterval
}
```

### Phase 5: Update Metrics Registry Registration

#### 5.1 Register All Filter Metrics
**File:** `internal/metrics/registry.go` (update `registerBuiltInMetrics`)

```go
// registerBuiltInMetrics registers all built-in metric computers
func (r *Registry) registerBuiltInMetrics() {
    // Basic metrics (already exist)
    r.Register(&PriceComputer{})
    r.Register(&VolumeLiveComputer{})
    r.Register(&VWAPLiveComputer{})
    r.Register(&CloseComputer{})
    r.Register(&OpenComputer{})
    r.Register(&HighComputer{})
    r.Register(&LowComputer{})
    r.Register(&VolumeComputer{})
    r.Register(&VWAPComputer{})
    
    // Price change metrics (already exist)
    r.Register(NewPriceChangeComputer("price_change_1m_pct", 2))
    r.Register(NewPriceChangeComputer("price_change_5m_pct", 6))
    r.Register(NewPriceChangeComputer("price_change_15m_pct", 16))
    
    // Volume filters
    r.Register(NewAbsoluteVolumeComputer(1 * time.Minute))
    r.Register(NewAbsoluteVolumeComputer(5 * time.Minute))
    r.Register(NewAbsoluteVolumeComputer(15 * time.Minute))
    r.Register(NewAbsoluteVolumeComputer(60 * time.Minute))
    r.Register(NewAbsoluteDollarVolumeComputer(1 * time.Minute))
    r.Register(NewAbsoluteDollarVolumeComputer(5 * time.Minute))
    // ... register all volume filters
    
    // Price filters
    r.Register(&ChangeFromCloseComputer{})
    r.Register(&ChangeFromOpenComputer{})
    r.Register(&GapFromCloseComputer{})
    // ... register all price filters
    
    // Range filters
    r.Register(NewRangeComputer(5 * time.Minute))
    r.Register(NewRangeComputer(15 * time.Minute))
    // ... register all range filters
    
    // Distance metrics (depend on indicators)
    r.Register(&VWAPDistanceComputer{})
    r.Register(&MADistanceComputer{})
    r.Register(&ATRPComputer{})
    
    // Trading activity filters
    r.Register(NewTradeCountComputer(5 * time.Minute))
    r.Register(NewConsecutiveCandlesComputer(1 * time.Minute))
    
    // Time-based filters
    r.Register(&MinutesInMarketComputer{})
    // ... register all time-based filters
    
    // Fundamental filters (will need external data integration)
    r.Register(&MarketCapComputer{})
    r.Register(&FloatComputer{})
    // ... register all fundamental filters
}
```

## Implementation Checklist

### Phase 1: Techan Integration
- [ ] Add Techan dependency
- [ ] Create `pkg/indicator/techan_adapter.go`
- [ ] Create `pkg/indicator/techan_factory.go`
- [ ] Implement all Techan factory functions
- [ ] Add unit tests

### Phase 2: Indicator Registry
- [ ] Create `internal/indicator/registry.go`
- [ ] Create `internal/indicator/indicator_registration.go`
- [ ] Register all Techan indicators
- [ ] Register all custom indicators (VWAP, Volume Average)
- [ ] Add unit tests

### Phase 3: Metrics Registry (Filters)
- [ ] Create `internal/metrics/volume_filters.go`
- [ ] Create `internal/metrics/price_filters.go`
- [ ] Create `internal/metrics/range_filters.go`
- [ ] Create `internal/metrics/trading_activity_filters.go`
- [ ] Create `internal/metrics/time_filters.go`
- [ ] Create `internal/metrics/fundamental_filters.go`
- [ ] Create `internal/metrics/distance_filters.go`
- [ ] Update `internal/metrics/registry.go` to register all filters
- [ ] Add unit tests for each filter category

### Phase 4: Requirement Discovery
- [ ] Update `internal/indicator/requirement_tracker.go`
- [ ] Add indicator vs metric detection
- [ ] Update requirement discovery logic
- [ ] Add unit tests

### Phase 5: Integration
- [ ] Update `internal/indicator/engine.go` for dynamic indicators
- [ ] Update `cmd/indicator/main.go` to use registry
- [ ] Ensure metrics registry is initialized in scanner
- [ ] Add metrics for tracking
- [ ] Add logging

### Phase 6: Cleanup
- [ ] Remove old custom RSI, EMA, SMA implementations
- [ ] Update all tests
- [ ] Update documentation

## Filter Implementation Priority

### Phase 1: Core Filters (High Priority)
1. **Volume Filters:** Absolute Volume, Absolute Dollar Volume
2. **Price Filters:** Change, Change from Close/Open, Percentage Change
3. **Range Filters:** Range, Percentage Range
4. **Technical Indicators:** RSI, ATR, EMA, SMA (via Techan)

### Phase 2: Advanced Filters (Medium Priority)
1. **Volume Filters:** Relative Volume, Average Volume
2. **Range Filters:** Position in Range, Relative Range
3. **Distance Metrics:** VWAP Distance, MA Distance, ATRP
4. **Trading Activity:** Trade Count, Consecutive Candles

### Phase 3: External Data Filters (Lower Priority)
1. **Time-Based Filters:** Minutes in Market, News time
2. **Fundamental Filters:** MarketCap, Float, Shares Outstanding
3. **Earnings/News Integration**

## Key Design Decisions

1. **Indicators vs Metrics:**
   - **Indicators:** Computed incrementally, stored in state (RSI, EMA, ATR, etc.)
   - **Metrics:** Computed on-demand from state (volume, price changes, ranges, etc.)

2. **Techan Usage:**
   - Use Techan for all technical indicators it supports
   - Keep custom implementations only for indicators not in Techan (VWAP, Volume Average)

3. **Filter Implementation:**
   - All filters use `MetricComputer` interface
   - Filters can depend on indicators (e.g., ATRP depends on ATR)
   - Filters are computed on-demand, not stored

4. **Requirement Discovery:**
   - Tracks both indicators and metrics
   - Only computes required indicators (performance optimization)
   - All metrics are available (computed on-demand)

## Benefits

1. **Separation of Concerns:** Indicators (incremental) vs Metrics (on-demand)
2. **Performance:** Only compute required indicators
3. **Flexibility:** Easy to add new filters via `MetricComputer` interface
4. **Techan Integration:** Use proven indicators from Techan library
5. **Comprehensive:** Supports all filters from Filters.md
