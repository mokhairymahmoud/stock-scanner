# Alert Implementation Status Analysis

This document provides a comprehensive analysis of how alerts from `docs/alerts.md` are currently implemented in the stock scanner system.

## Architecture Overview

### Alert Processing Pipeline

1. **Scanner** (`internal/scanner/scan_loop.go`)
   - Evaluates rules against symbol states
   - Emits alerts when rules match
   - Pre-filters by volume threshold and session

2. **Alert Consumer** (`internal/alert/consumer.go`)
   - Consumes alerts from Redis stream
   - Deduplicates alerts
   - Filters by user preferences (MVP: pass-through)
   - Persists alerts to database
   - Routes to filtered stream

3. **WebSocket Gateway** (`internal/wsgateway/`)
   - Consumes from filtered stream
   - Broadcasts to connected clients

### Metrics System

The system uses a **metric registry** (`internal/metrics/registry.go`) that:
- Registers metric computers (functions that compute metrics from symbol state)
- Computes metrics on-demand during rule evaluation
- Supports dependency tracking (future enhancement)

### Symbol State

Symbol state (`internal/scanner/state.go`) tracks:
- Live bar (current minute being built)
- Last finalized bars (ring buffer, ~200 bars)
- Indicators (EMA, SMA, VWAP, RSI, ATR)
- Session tracking (premarket, market, postmarket)
- Price references (yesterday close, today open/close)
- Session-specific volumes
- Trade counts
- Candle directions (for consecutive candles)

---

## Implementation Status by Alert Category

### 1. Candlestick Pattern Alerts ❌ **NOT IMPLEMENTED**

#### 1.1 Lower Shadow Alert
- **Status**: ❌ Not implemented
- **Required Metrics**: `lower_shadow_ratio_{timeframe}`
- **Current State**: No pattern detection metrics exist
- **Implementation Needed**:
  - Add metric computer for shadow ratio calculation
  - Track previous candle for comparison
  - Support multiple timeframes (1m, 2m, 5m, 15m)

#### 1.2 Upper Shadow Alert
- **Status**: ❌ Not implemented
- **Required Metrics**: `upper_shadow_ratio_{timeframe}`
- **Implementation Needed**: Same as lower shadow

#### 1.3 Bullish Candle Close
- **Status**: ❌ Not implemented
- **Required Metrics**: `is_bullish_candle_{timeframe}` (boolean)
- **Note**: Candle direction is tracked in `CandleDirections` map, but not exposed as metrics
- **Implementation Needed**: Add metric computer that checks `close > open`

#### 1.4 Bearish Candle Close
- **Status**: ❌ Not implemented
- **Required Metrics**: `is_bearish_candle_{timeframe}` (boolean)
- **Implementation Needed**: Add metric computer that checks `close < open`

#### 1.5 Bullish Engulfing Candle
- **Status**: ❌ Not implemented
- **Required Metrics**: `bullish_engulfing_{timeframe}` (boolean)
- **Implementation Needed**:
  - Compare current and previous candle
  - Check engulfing conditions
  - Support timeframes: 1m, 2m, 5m, 15m, 30m, 60m

#### 1.6 Bearish Engulfing Candle
- **Status**: ❌ Not implemented
- **Required Metrics**: `bearish_engulfing_{timeframe}` (boolean)
- **Implementation Needed**: Similar to bullish engulfing

#### 1.7 Bullish Harami Candle
- **Status**: ❌ Not implemented
- **Required Metrics**: `bullish_harami_{timeframe}` (boolean)
- **Implementation Needed**: Pattern detection with containment check

#### 1.8 Bearish Harami Candle
- **Status**: ❌ Not implemented
- **Required Metrics**: `bearish_harami_{timeframe}` (boolean)
- **Implementation Needed**: Pattern detection

#### 1.9 Inside Bar
- **Status**: ❌ Not implemented
- **Required Metrics**: `inside_bar_{timeframe}` (boolean)
- **Implementation Needed**: Range comparison between candles

---

### 2. Price Level Alerts ⚠️ **PARTIALLY IMPLEMENTED**

#### 2.1 Near High/Low of the Day
- **Status**: ⚠️ Partially implemented
- **Current Metrics**: 
  - `range_today` - daily range ($)
  - `range_pct_today` - daily range (%)
  - `position_in_range_today` - position in range (%)
- **Missing Metrics**: 
  - `dist_to_high_of_day_pct` - distance to high of day (%)
  - `dist_to_low_of_day_pct` - distance to low of day (%)
- **Implementation Status**: 
  - High/low tracking exists in `DailyRangeComputer` but not exposed as separate metrics
  - Need to add distance metrics
  - Need to support direction selection (High/Low)

#### 2.2 High/Low of the Day (Pre/Post-Market)
- **Status**: ❌ Not implemented
- **Required Metrics**: `dist_to_high_of_day_extended_pct`, `dist_to_low_of_day_extended_pct`
- **Implementation Needed**: Track high/low from 4:00 AM to 8:00 PM ET

#### 2.3 Near Last High
- **Status**: ❌ Not implemented
- **Required Metrics**: `dist_to_recent_high_{timeframe}_pct`
- **Implementation Needed**: 
  - Track recent high over rolling window
  - Calculate distance percentage
  - Support timeframes: 1m, 2m, 5m, 15m

#### 2.4 Near Last Low
- **Status**: ❌ Not implemented
- **Required Metrics**: `dist_to_recent_low_{timeframe}_pct`
- **Implementation Needed**: Similar to near last high

#### 2.5 Break Over Recent High
- **Status**: ❌ Not implemented
- **Required Metrics**: `broke_recent_high_{timeframe}` (boolean)
- **Implementation Needed**: Detect price crossing above recent high

#### 2.6 Break Under Recent Low
- **Status**: ❌ Not implemented
- **Required Metrics**: `broke_recent_low_{timeframe}` (boolean)
- **Implementation Needed**: Detect price crossing below recent low

#### 2.7 Reject Last High
- **Status**: ❌ Not implemented
- **Required Metrics**: `rejected_recent_high_{timeframe}` (boolean)
- **Implementation Needed**: 
  - Track price approaching high
  - Detect subsequent rejection
  - State machine for rejection detection

#### 2.8 Reject Last Low
- **Status**: ❌ Not implemented
- **Required Metrics**: `rejected_recent_low_{timeframe}` (boolean)
- **Implementation Needed**: Similar to reject last high

#### 2.9 New Candle High
- **Status**: ❌ Not implemented
- **Required Metrics**: `new_candle_high_{timeframe}` (boolean)
- **Implementation Needed**: Compare current candle high to previous

#### 2.10 New Candle Low
- **Status**: ❌ Not implemented
- **Required Metrics**: `new_candle_low_{timeframe}` (boolean)
- **Implementation Needed**: Compare current candle low to previous

---

### 3. VWAP Alerts ⚠️ **PARTIALLY IMPLEMENTED**

#### 3.1 Through VWAP Alert
- **Status**: ❌ Not implemented
- **Required Metrics**: `through_vwap_{direction}` (boolean)
- **Current Metrics**: 
  - `vwap_dist_{timeframe}` - distance from VWAP ($)
  - `vwap_dist_{timeframe}_pct` - distance from VWAP (%)
- **Missing**: 
  - Candle size comparison (3x average)
  - Crossing detection
  - Direction selection (above/below)

#### 3.2 VWAP Acts as Support
- **Status**: ❌ Not implemented
- **Required Metrics**: `vwap_support_{timeframe}` (boolean)
- **Implementation Needed**: 
  - State machine to track price approaching from above
  - Detect touch (within 0.1%)
  - Detect bounce (price rises by threshold)

#### 3.3 VWAP Acts as Resistance
- **Status**: ❌ Not implemented
- **Required Metrics**: `vwap_resistance_{timeframe}` (boolean)
- **Implementation Needed**: Similar to support, but from below

---

### 4. Moving Average Alerts ⚠️ **PARTIALLY IMPLEMENTED**

#### 4.1 Back to EMA Alert
- **Status**: ❌ Not implemented
- **Required Metrics**: `back_to_ema_{ema_type}_{timeframe}` (boolean)
- **Current Metrics**: 
  - `ma_dist_{ma_type}_{timeframe}_pct` - distance from MA (%)
- **Missing**: 
  - State tracking (was far, now close)
  - Multiple EMA options support

#### 4.2 Crossing Above
- **Status**: ❌ Not implemented
- **Required Metrics**: `crossed_above_{level_type}` (boolean)
- **Current Metrics**: 
  - `change_from_open` - change from today's open ($)
  - `change_from_close` - change from yesterday's close ($)
  - `ma_dist_*_pct` - distance from MA (%)
- **Missing**: 
  - Previous price tracking
  - Crossing detection logic
  - Support for multiple level types (Open, Close, VWAP, EMA, SMA)

#### 4.3 Crossing Below
- **Status**: ❌ Not implemented
- **Required Metrics**: `crossed_below_{level_type}` (boolean)
- **Implementation Needed**: Similar to crossing above

---

### 5. Volume Alerts ⚠️ **PARTIALLY IMPLEMENTED**

#### 5.1 Volume Spike (2)
- **Status**: ⚠️ Partially implemented
- **Current Metrics**: 
  - `relative_volume_{timeframe}` - compares to last 10 bars (%)
- **Required Metrics**: `volume_spike_2_{timeframe}` (ratio value)
- **Missing**: 
  - Specific 2-candle average calculation
  - Ratio metric (not percentage)
  - Multiplier threshold support

#### 5.2 Volume Spike (10)
- **Status**: ⚠️ Partially implemented
- **Current Metrics**: `relative_volume_{timeframe}` uses 10 bars
- **Required Metrics**: `volume_spike_10_{timeframe}` (ratio value)
- **Missing**: Ratio format (currently percentage)

---

### 6. Price Movement Alerts ❌ **NOT IMPLEMENTED**

#### 6.1 Running Up
- **Status**: ❌ Not implemented
- **Required Metrics**: `running_up_60s` ($), `running_up_60s_pct` (%)
- **Implementation Needed**: 
  - Track price 60 seconds ago
  - Calculate change (absolute and percentage)
  - Support value type selection ($ or %)

#### 6.2 Running Down
- **Status**: ❌ Not implemented
- **Required Metrics**: `running_down_60s` ($), `running_down_60s_pct` (%)
- **Implementation Needed**: Similar to running up

---

### 7. Opening Range Alerts ❌ **NOT IMPLEMENTED**

#### 7.1 Opening Range Breakout
- **Status**: ❌ Not implemented
- **Required Metrics**: `opening_range_breakout_{timeframe}` (boolean)
- **Implementation Needed**: 
  - Identify first candle after market open (9:30 AM ET)
  - Store opening range (high/low)
  - Detect breakout above range
  - Reset daily

#### 7.2 Opening Range Breakdown
- **Status**: ❌ Not implemented
- **Required Metrics**: `opening_range_breakdown_{timeframe}` (boolean)
- **Implementation Needed**: Similar to breakout, but below range

---

## Infrastructure Components

### ✅ Implemented

1. **Alert Pipeline**
   - Alert emission from scanner
   - Redis stream consumption
   - Deduplication
   - User filtering (MVP: pass-through)
   - Persistence
   - Routing to filtered stream
   - WebSocket broadcasting

2. **Metrics System**
   - Metric registry
   - Metric computers
   - On-demand computation
   - Symbol state snapshots

3. **Rule Evaluation**
   - Rule compilation
   - Metric extraction
   - Pre-filtering (volume threshold, session)
   - Cooldown tracking

4. **Session Tracking**
   - Premarket, Market, Postmarket detection
   - Session transitions
   - Session-specific volume tracking

5. **Price Metrics**
   - Change from open/close
   - Gap calculations
   - Range calculations
   - Position in range

6. **Volume Metrics**
   - Absolute volume (multiple timeframes)
   - Dollar volume
   - Relative volume (simplified)
   - Session-specific volumes

7. **Indicator Metrics**
   - VWAP distance
   - MA distance (EMA, SMA)
   - ATR percentage

### ❌ Missing Infrastructure

1. **Pattern Detection Framework**
   - No candlestick pattern detection
   - No previous candle comparison utilities
   - No pattern state machines

2. **Crossing Detection**
   - No previous price tracking
   - No crossing detection logic
   - No state machines for support/resistance

3. **High/Low Tracking**
   - No rolling window high/low tracking
   - No recent high/low metrics
   - No opening range tracking

4. **Time-Based Price Tracking**
   - No 60-second price history
   - No tick-level price tracking for movement alerts

5. **State Machines**
   - No rejection detection state machines
   - No support/resistance state machines
   - No "back to EMA" state tracking

---

## Implementation Patterns

### Current Patterns

1. **Metric Computers**
   - Each metric is a `MetricComputer` interface
   - Computes from `SymbolStateSnapshot`
   - Returns `(value float64, ok bool)`
   - Registered in `Registry.registerBuiltInMetrics()`

2. **Pre-filtering**
   - Volume threshold checked before rule evaluation
   - Session filter checked before rule evaluation
   - Implemented in `ScanLoop.shouldEvaluateRule()`

3. **Bar Finalization**
   - Bars finalized and added to `LastFinalBars` ring buffer
   - Metrics computed from finalized bars
   - Live bar used for current price

4. **Session Management**
   - Session transitions handled in `StateManager.handleSessionTransition()`
   - Session-specific data reset on transitions
   - Session tracked in `SymbolState.CurrentSession`

### Patterns Needed for Missing Alerts

1. **Pattern Detection**
   - Need to compare current and previous candles
   - Need candle body/shadow calculations
   - Need pattern state tracking

2. **Crossing Detection**
   - Need previous price storage
   - Need crossing detection on bar finalization
   - Need level value lookup (VWAP, EMA, etc.)

3. **State Machines**
   - Need state tracking for rejection detection
   - Need state tracking for support/resistance
   - Need state tracking for "back to EMA"

4. **Rolling Windows**
   - Need high/low tracking over rolling windows
   - Need recent high/low updates
   - Need opening range storage

---

## Recommendations

### Priority 1: Foundation
1. **Add Previous Candle Tracking**
   - Store previous candle in `SymbolState`
   - Update on bar finalization
   - Expose in `SymbolStateSnapshot`

2. **Add Crossing Detection Framework**
   - Previous price tracking
   - Crossing detection utilities
   - Level value lookup helpers

3. **Add Pattern Detection Framework**
   - Candle comparison utilities
   - Body/shadow calculations
   - Pattern detection functions

### Priority 2: High-Value Alerts
1. **Candlestick Patterns** (Bullish/Bearish Close, Engulfing)
2. **Price Level Alerts** (Near High/Low, Break Over/Under)
3. **Volume Spikes** (Complete 2 and 10 candle implementations)

### Priority 3: Advanced Alerts
1. **VWAP Support/Resistance**
2. **Crossing Alerts**
3. **Opening Range Alerts**

### Priority 4: Complex State Machines
1. **Rejection Detection**
2. **Back to EMA**
3. **Running Up/Down** (requires tick-level tracking)

---

## Files to Create/Modify

### New Files Needed
- `internal/metrics/pattern_filters.go` - Candlestick pattern metrics
- `internal/metrics/crossing_filters.go` - Crossing detection metrics
- `internal/metrics/high_low_tracking.go` - High/low tracking metrics
- `internal/scanner/pattern_detector.go` - Pattern detection utilities
- `internal/scanner/crossing_detector.go` - Crossing detection utilities

### Files to Modify
- `internal/scanner/state.go` - Add previous candle, high/low tracking, opening range
- `internal/metrics/registry.go` - Register new metric computers
- `internal/metrics/volume_filters.go` - Complete volume spike implementations
- `internal/metrics/price_filters.go` - Add high/low distance metrics

---

## Testing Strategy

1. **Unit Tests**: Test each metric computer independently
2. **Integration Tests**: Test pattern detection in scan loop
3. **Timeframe Tests**: Test all supported timeframes
4. **Session Tests**: Test Pre-Market, Market, Post-Market behavior
5. **Edge Cases**: Missing data, single candles, etc.
6. **Performance Tests**: Ensure pattern detection doesn't slow scan loop

---

## Summary

**Total Alert Types**: 30
- **Fully Implemented**: 0
- **Partially Implemented**: 4 (Volume Spike, Near High/Low, VWAP Distance, MA Distance)
- **Not Implemented**: 26

**Infrastructure Status**:
- ✅ Alert pipeline: Complete
- ✅ Metrics system: Complete
- ✅ Rule evaluation: Complete
- ✅ Session tracking: Complete
- ❌ Pattern detection: Missing
- ❌ Crossing detection: Missing
- ❌ State machines: Missing
- ❌ High/low tracking: Missing

The system has a solid foundation with the alert pipeline and metrics system, but most alert types from `docs/alerts.md` are not yet implemented. The main gaps are in pattern detection, crossing detection, and state machine-based alerts.

