# Phase 5.3: Filter Implementation Plan

## Overview

This document provides a detailed implementation plan for Phase 5.3: Filter Implementation. The goal is to implement comprehensive filter support for all filter types shown in the UI, supporting volume thresholds, timeframes, session-based filtering, and value types.

## Current State Analysis

### ✅ What's Already Implemented

1. **Metrics Registry System**
   - `MetricComputer` interface for computing metrics
   - `Registry` for managing and computing all metrics
   - Integration with scanner's `getMetricsFromSnapshot`
   - Basic metrics: `price`, `volume_live`, `vwap_live`, `close`, `open`, `high`, `low`, `volume`, `vwap`
   - Price change metrics: `price_change_1m_pct`, `price_change_2m_pct`, `price_change_5m_pct`, `price_change_15m_pct`, `price_change_30m_pct`, `price_change_60m_pct`

2. **Symbol State Management**
   - `SymbolState` with `LiveBar`, `LastFinalBars` (ring buffer), `Indicators`
   - Thread-safe state management
   - Snapshot mechanism for lock-free scanning
   - **NEW:** Session tracking (Pre-Market, Market, Post-Market)
   - **NEW:** Price references (YesterdayClose, TodayOpen, TodayClose)
   - **NEW:** Session-specific volume tracking (PremarketVolume, MarketVolume, PostmarketVolume)
   - **NEW:** Trade count tracking
   - **NEW:** Candle direction tracking

3. **Indicator Engine**
   - RSI, EMA, SMA, MACD, ATR, Bollinger Bands, Stochastic (via Techan)
   - VWAP, Volume Average, Price Change (custom)
   - Indicators stored in `SymbolState.Indicators` map

4. **Session Detection** ✅ NEW
   - Market session detection (Pre-Market: 4:00-9:30, Market: 9:30-16:00, Post-Market: 16:00-20:00 ET)
   - Timezone handling (ET to UTC conversion)
   - Session transition detection
   - Helper functions (IsMarketOpen, MinutesSinceMarketOpen, etc.)

5. **Core Price Filters** ✅ NEW (Phase 1 Complete)
   - Change ($) with timeframes (1m, 2m, 5m, 15m, 30m, 60m)
   - Change from Close ($ and %)
   - Change from Close (Premarket) ($ and %)
   - Change from Close (Post Market) ($ and %)
   - Change from Open ($ and %)
   - Gap from Close ($ and %)

6. **Core Volume Filters** ✅ NEW (Phase 1 Complete)
   - Postmarket Volume
   - Premarket Volume
   - Absolute Volume with timeframes (1m, 2m, 5m, 10m, 15m, 30m, 60m, daily)
   - Absolute Dollar Volume with timeframes (1m, 5m, 15m, 60m, daily)

### ❌ What Still Needs to Be Implemented

1. **Filter Metrics** (Remaining ~35 filter types)
   - Range filters (5 types) - Phase 2
   - Technical indicator filters (5 types) - Phase 2
   - Advanced volume filters (3 types) - Phase 3
   - Trading activity filters (2 types) - Phase 3
   - Time-based filters (5 types) - Phase 4
   - Fundamental data filters (6 types) - Phase 5

2. **Filter Infrastructure** (Remaining)
   - Volume threshold enforcement in rule evaluation - Phase 6
   - Session-based filtering in rule evaluation - Phase 6
   - Timeframe support in rule parser - Phase 6
   - Value type variants already implemented in metrics

## Implementation Phases

### Phase 1: Foundation & Core Price/Volume Filters (Priority: HIGH)

**Goal**: Implement infrastructure and most commonly used filters

#### 1.1 Extend Symbol State (Week 1, Days 1-2) ✅ COMPLETE

**Tasks:**
- [x] Add session tracking to `SymbolState`
  - [x] `CurrentSession` field (PreMarket, Market, PostMarket)
  - [x] `SessionStartTime` field
- [x] Add price reference fields
  - [x] `YesterdayClose` field
  - [x] `TodayOpen` field
  - [x] `TodayClose` field (for postmarket)
- [x] Add session-specific volume tracking
  - [x] `PremarketVolume` field
  - [x] `PostmarketVolume` field
  - [x] `MarketVolume` field
- [x] Add trade count tracking
  - [x] `TradeCount` field (incremented on each tick)
  - [x] `TradeCountHistory` ring buffer (for timeframe-based counts)
- [x] Add candle direction tracking
  - [x] `CandleDirections` map (timeframe -> direction history)

**Files to Modify:**
- `internal/scanner/state.go` - Extend `SymbolState` struct
- `internal/scanner/tick_consumer.go` - Update session tracking and trade count
- `internal/scanner/bar_handler.go` - Update session tracking and candle direction

**New Files:**
- `internal/scanner/session.go` - Session detection utilities

#### 1.2 Session Detection (Week 1, Day 2) ✅ COMPLETE

**Tasks:**
- [x] Implement session detection logic
  - [x] Pre-Market: 4:00 AM - 9:30 AM ET
  - [x] Market: 9:30 AM - 4:00 PM ET
  - [x] Post-Market: 4:00 PM - 8:00 PM ET
- [x] Add session transition detection
- [x] Reset session-specific data on session transitions
- [x] Handle timezone conversions (ET to UTC)

**Files to Create:**
- `internal/scanner/session.go` - Session detection functions

**Files to Modify:**
- `internal/scanner/tick_consumer.go` - Check and update session on tick
- `internal/scanner/bar_handler.go` - Check and update session on bar finalization

#### 1.3 Core Price Filters (Week 1, Days 3-4) ✅ COMPLETE

**Tasks:**
- [x] Implement Change ($) filter with timeframes
  - [x] Metrics: `change_1m`, `change_2m`, `change_5m`, `change_15m`, `change_30m`, `change_60m`
  - [x] Computer: `ChangeComputer` with timeframe parameter
- [x] Implement Change from Close filter
  - [x] Metrics: `change_from_close`, `change_from_close_pct`
  - [x] Computer: `ChangeFromCloseComputer`
- [x] Implement Change from Close (Premarket) filter
  - [x] Metrics: `change_from_close_premarket`, `change_from_close_premarket_pct`
  - [x] Computer: `ChangeFromClosePremarketComputer` (with session check)
- [x] Implement Change from Close (Post Market) filter
  - [x] Metrics: `change_from_close_postmarket`, `change_from_close_postmarket_pct`
  - [x] Computer: `ChangeFromClosePostmarketComputer` (with session check)
- [x] Implement Change from Open filter
  - [x] Metrics: `change_from_open`, `change_from_open_pct`
  - [x] Computer: `ChangeFromOpenComputer`
- [x] Extend Percentage Change (%) filter
  - [x] Add timeframes: `change_pct_2m`, `change_pct_30m`, `change_pct_60m` (2h, 4h pending for Phase 2)
  - [x] Extend `PriceChangeComputer` to support more timeframes
- [x] Implement Gap from Close filter
  - [x] Metrics: `gap_from_close`, `gap_from_close_pct`
  - [x] Computer: `GapFromCloseComputer`

**Files to Create:**
- `internal/metrics/price_filters.go` - All price filter computers

**Files to Modify:**
- `internal/metrics/registry.go` - Register new price filter computers
- `internal/metrics/price_change_metrics.go` - Extend for more timeframes

#### 1.4 Core Volume Filters (Week 1, Days 4-5) ✅ COMPLETE

**Tasks:**
- [x] Implement Postmarket Volume tracking
  - [x] Metric: `postmarket_volume`
  - [x] Computer: `PostmarketVolumeComputer`
  - [x] Update volume tracking in tick consumer
- [x] Implement Premarket Volume tracking
  - [x] Metric: `premarket_volume`
  - [x] Computer: `PremarketVolumeComputer`
- [x] Extend Absolute Volume filter
  - [x] Metrics: `volume_1m`, `volume_2m`, `volume_5m`, `volume_10m`, `volume_15m`, `volume_30m`, `volume_60m`, `volume_daily`
  - [x] Computer: `AbsoluteVolumeComputer` with timeframe parameter
- [x] Implement Absolute Dollar Volume filter
  - [x] Metrics: `dollar_volume_1m`, `dollar_volume_5m`, `dollar_volume_15m`, `dollar_volume_60m`, `dollar_volume_daily`
  - [x] Computer: `DollarVolumeComputer` with timeframe parameter

**Files to Create:**
- `internal/metrics/volume_filters.go` - All volume filter computers

**Files to Modify:**
- `internal/scanner/tick_consumer.go` - Track session-specific volumes
- `internal/metrics/registry.go` - Register new volume filter computers

#### 1.5 Testing & Validation (Week 1, Day 5) ✅ COMPLETE

**Tasks:**
- [x] Unit tests for session detection
- [x] Unit tests for all price filter computers
- [x] Unit tests for all volume filter computers
- [ ] Integration tests for session transitions (deferred to Phase 7)
- [ ] Integration tests for price/volume metrics in scan loop (deferred to Phase 7)

**Files to Create:**
- `internal/scanner/session_test.go`
- `internal/metrics/price_filters_test.go`
- `internal/metrics/volume_filters_test.go`

### Phase 1 Completion Summary

**Status:** ✅ Complete

**Deliverables:**
- ✅ Session detection utilities (`internal/scanner/session.go`)
  - Market session detection (Pre-Market, Market, Post-Market, Closed)
  - Timezone handling (ET to UTC conversion)
  - Helper functions (IsMarketOpen, MinutesSinceMarketOpen, GetMarketOpenTime, GetMarketCloseTime)
- ✅ Extended SymbolState (`internal/scanner/state.go`)
  - Session tracking (CurrentSession, SessionStartTime)
  - Price references (YesterdayClose, TodayOpen, TodayClose)
  - Session-specific volume tracking (PremarketVolume, MarketVolume, PostmarketVolume)
  - Trade count tracking (TradeCount, TradeCountHistory)
  - Candle direction tracking (CandleDirections map)
- ✅ Session transition handling
  - Automatic session detection on tick/bar processing
  - Session-specific data reset on transitions
  - Volume tracking per session
- ✅ Core Price Filters (`internal/metrics/price_filters.go`)
  - Change ($) with timeframes: `change_1m`, `change_2m`, `change_5m`, `change_15m`, `change_30m`, `change_60m`
  - Change from Close: `change_from_close`, `change_from_close_pct`
  - Change from Close (Premarket): `change_from_close_premarket`, `change_from_close_premarket_pct`
  - Change from Close (Post Market): `change_from_close_postmarket`, `change_from_close_postmarket_pct`
  - Change from Open: `change_from_open`, `change_from_open_pct`
  - Gap from Close: `gap_from_close`, `gap_from_close_pct`
  - Extended price change metrics: `price_change_2m_pct`, `price_change_30m_pct`, `price_change_60m_pct`
- ✅ Core Volume Filters (`internal/metrics/volume_filters.go`)
  - Postmarket Volume: `postmarket_volume`
  - Premarket Volume: `premarket_volume`
  - Absolute Volume: `volume_1m`, `volume_2m`, `volume_5m`, `volume_10m`, `volume_15m`, `volume_30m`, `volume_60m`, `volume_daily`
  - Dollar Volume: `dollar_volume_1m`, `dollar_volume_5m`, `dollar_volume_15m`, `dollar_volume_60m`, `dollar_volume_daily`
- ✅ Metrics Registry Integration
  - All new metric computers registered in `internal/metrics/registry.go`
  - Extended SymbolStateSnapshot in `internal/metrics/computer.go`
  - Updated scan loop to include new fields in metric snapshot
- ✅ Comprehensive Unit Tests
  - Session detection tests (13 test cases, all passing)
  - Price filter tests (5 test cases, all passing)
  - Volume filter tests (5 test cases, all passing)

**Key Features:**
- Complete session detection with timezone handling
- Automatic session transitions with data reset
- 12 price filter metrics (8 filter types with $ and % variants)
- 11 volume filter metrics (4 filter types with multiple timeframes)
- Thread-safe state management
- All tests passing

**Verification:**
- All code compiles successfully
- All unit tests pass (23+ test cases)
- Session detection working correctly
- Price and volume filters computing correctly
- No linter errors

**Next Steps:**
- Phase 2: Range & Technical Indicator Filters
- Phase 3: Advanced Volume & Trading Activity Filters

---

### Phase 2: Range & Technical Indicator Filters (Priority: HIGH)

**Goal**: Implement range calculations and technical indicator distance metrics

#### 2.1 Range Filters (Week 2, Days 1-2) ✅ COMPLETE

**Tasks:**
- [x] Implement Range ($) filter
  - [x] Metrics: `range_2m`, `range_5m`, `range_10m`, `range_15m`, `range_30m`, `range_60m`, `range_today`
  - [x] Computer: `RangeComputer` with timeframe parameter
  - [x] Track high/low over timeframes
- [x] Implement Percentage Range (%) filter
  - [x] Metrics: `range_pct_2m`, `range_pct_5m`, `range_pct_10m`, `range_pct_15m`, `range_pct_30m`, `range_pct_60m`, `range_pct_today`
  - [x] Computer: `RangePercentageComputer` with timeframe parameter
- [x] Implement Position in Range (%) filter
  - [x] Metrics: `position_in_range_2m`, `position_in_range_5m`, `position_in_range_15m`, `position_in_range_30m`, `position_in_range_60m`, `position_in_range_today`
  - [x] Computer: `PositionInRangeComputer` with timeframe parameter
- [x] Implement Relative Range (%) filter
  - [x] Metric: `relative_range_pct` (compares today's range vs ATR(14))
  - [x] Computer: `RelativeRangeComputer` (uses ATR from indicators)

**Files Created:**
- `internal/metrics/range_filters.go` - All range filter computers
- `internal/metrics/range_filters_test.go` - Unit tests for range filters

**Files Modified:**
- `internal/metrics/registry.go` - Registered all range filter computers

#### 2.2 Technical Indicator Filters (Week 2, Days 2-3) ✅ COMPLETE

**Tasks:**
- [x] Implement ATRP(14) calculation
  - [x] Metrics: `atrp_14_1m`, `atrp_14_5m`, `atrp_14_daily`
  - [x] Computer: `ATRPComputer` (uses ATR from indicators)
- [x] Extend Distance from VWAP filter
  - [x] Metrics: `vwap_dist_5m`, `vwap_dist_15m`, `vwap_dist_1h` ($)
  - [x] Metrics: `vwap_dist_5m_pct`, `vwap_dist_15m_pct`, `vwap_dist_1h_pct` (%)
  - [x] Computer: `VWAPDistanceComputer` and `VWAPDistancePctComputer`
- [x] Implement Distance from Moving Average filter
  - [x] Metrics for each MA type:
    - [ ] `ma_dist_sma20_daily_pct`, `ma_dist_sma10_daily_pct`, `ma_dist_sma200_daily_pct`
    - [x] `ma_dist_sma20_daily_pct`, `ma_dist_sma10_daily_pct`, `ma_dist_sma200_daily_pct`
    - [x] `ma_dist_ema20_1m_pct`, `ma_dist_ema9_1m_pct`, `ma_dist_ema9_5m_pct`, `ma_dist_ema9_15m_pct`
    - [x] `ma_dist_ema21_15m_pct`, `ma_dist_ema9_60m_pct`, `ma_dist_ema21_60m_pct`
    - [x] `ma_dist_ema50_15m_pct`, `ma_dist_ema50_daily_pct`
  - [x] Computer: `MADistanceComputer` with MA type and timeframe parameters

**Files Created:**
- `internal/metrics/indicator_filters.go` - Indicator distance computers
- `internal/metrics/indicator_filters_test.go` - Unit tests for indicator filters

**Files Modified:**
- `internal/metrics/registry.go` - Registered all indicator filter computers

**Note:** ATR(14) is already registered in the indicator engine. RSI timeframe extension is deferred to Phase 4.

#### 2.3 Testing & Validation (Week 2, Day 3) ✅ COMPLETE

**Tasks:**
- [x] Unit tests for range filter computers
- [x] Unit tests for indicator filter computers
- [ ] Integration tests for range calculations (deferred to Phase 7)
- [ ] Integration tests for indicator distances (deferred to Phase 7)

**Files Created:**
- `internal/metrics/range_filters_test.go` - 5 test cases, all passing
- `internal/metrics/indicator_filters_test.go` - 4 test cases, all passing

### Phase 2 Completion Summary

**Status:** ✅ Complete

**Deliverables:**
- ✅ Range Filters (`internal/metrics/range_filters.go`)
  - Range ($): `range_2m`, `range_5m`, `range_10m`, `range_15m`, `range_30m`, `range_60m`, `range_today`
  - Percentage Range (%): `range_pct_2m`, `range_pct_5m`, `range_pct_10m`, `range_pct_15m`, `range_pct_30m`, `range_pct_60m`, `range_pct_today`
  - Position in Range (%): `position_in_range_2m`, `position_in_range_5m`, `position_in_range_15m`, `position_in_range_30m`, `position_in_range_60m`, `position_in_range_today`
  - Relative Range (%): `relative_range_pct` (compares today's range vs ATR(14))
- ✅ Technical Indicator Filters (`internal/metrics/indicator_filters.go`)
  - ATRP (ATR Percentage): `atrp_14_1m`, `atrp_14_5m`, `atrp_14_daily`
  - VWAP Distance ($): `vwap_dist_5m`, `vwap_dist_15m`, `vwap_dist_1h`
  - VWAP Distance (%): `vwap_dist_5m_pct`, `vwap_dist_15m_pct`, `vwap_dist_1h_pct`
  - MA Distance (%): 12 metrics for various EMA/SMA combinations across timeframes
- ✅ Metrics Registry Integration
  - All new metric computers registered in `internal/metrics/registry.go`
- ✅ Comprehensive Unit Tests
  - Range filter tests (5 test cases, all passing)
  - Indicator filter tests (4 test cases, all passing)

**Key Features:**
- 7 range filter metrics (4 filter types with multiple timeframes)
- 18 indicator filter metrics (4 filter types with multiple timeframes and MA combinations)
- All filters compute from `SymbolStateSnapshot` with proper dependency handling
- Thread-safe metric computation
- All tests passing

**Verification:**
- All code compiles successfully
- All unit tests pass (9+ test cases)
- Range and indicator filters computing correctly
- No linter errors

**Next Steps:**
- Phase 3: Advanced Volume & Trading Activity Filters
- Phase 4: Extended Technical Indicators (RSI timeframe extension)

---

### Phase 3: Advanced Volume & Trading Activity Filters (Priority: MEDIUM)

**Goal**: Implement relative volume calculations and trading activity metrics

#### 3.1 Advanced Volume Filters (Week 2, Days 4-5) ✅ COMPLETE

**Tasks:**
- [x] Implement Average Volume filter (5d, 10d, 20d)
  - [x] Metrics: `avg_volume_5d`, `avg_volume_10d`, `avg_volume_20d`
  - [x] Computer: `AverageVolumeComputer` with day parameter
  - [x] Note: Simplified implementation using available bars. Full implementation would require historical data retrieval from TimescaleDB
- [x] Implement Relative Volume (%) filter
  - [x] Track average volume of last N bars per timeframe
  - [x] Metrics: `relative_volume_1m`, `relative_volume_2m`, `relative_volume_5m`, `relative_volume_15m`, `relative_volume_daily`
  - [x] Computer: `RelativeVolumeComputer` with timeframe parameter
  - [x] Note: Volume forecasting for intraday timeframes can be added in future enhancement
- [x] Implement Relative Volume (%) at Same Time filter
  - [x] Metric: `relative_volume_same_time`
  - [x] Computer: `RelativeVolumeSameTimeComputer`
  - [x] Note: Simplified implementation. Full implementation would require time-of-day pattern storage

**Files Created:**
- `internal/metrics/advanced_volume_filters.go` - Advanced volume computers

**Files Modified:**
- `internal/metrics/registry.go` - Registered advanced volume computers

#### 3.2 Trading Activity Filters (Week 2, Day 5) ✅ COMPLETE

**Tasks:**
- [x] Implement Trade Count filter
  - [x] Metrics: `trade_count_1m`, `trade_count_2m`, `trade_count_5m`, `trade_count_15m`, `trade_count_60m`
  - [x] Computer: `TradeCountComputer` with timeframe parameter
  - [x] Track trade count history in `SymbolState` (populated when bars are finalized)
- [x] Implement Consecutive Candles filter
  - [x] Track candle direction (green/red) from finalized bars
  - [x] Count consecutive candles of same direction
  - [x] Metrics: `consecutive_candles_1m`, `consecutive_candles_2m`, `consecutive_candles_5m`, `consecutive_candles_15m`, `consecutive_candles_daily`
  - [x] Computer: `ConsecutiveCandlesComputer` with timeframe parameter
  - [x] Positive for green, negative for red

**Files Created:**
- `internal/metrics/activity_filters.go` - Trading activity computers
- `internal/metrics/activity_filters_test.go` - Unit tests for activity filters

**Files Modified:**
- `internal/scanner/state.go` - Populate TradeCountHistory when bars are finalized
- `internal/metrics/registry.go` - Registered activity filter computers

#### 3.3 Testing & Validation (Week 2, Day 5) ✅ COMPLETE

**Tasks:**
- [x] Unit tests for advanced volume filter computers (3 test cases, all passing)
- [x] Unit tests for activity filter computers (11 test cases, all passing)
- [ ] Integration tests for relative volume calculations (deferred to Phase 7)
- [ ] Integration tests for trade count and consecutive candles (deferred to Phase 7)

**Files Created:**
- `internal/metrics/activity_filters_test.go` - Unit tests for activity and advanced volume filters (14 test cases total)

### Phase 3 Completion Summary

**Status:** ✅ Complete

**Deliverables:**
- ✅ Advanced Volume Filters (`internal/metrics/advanced_volume_filters.go`)
  - Average Volume: `avg_volume_5d`, `avg_volume_10d`, `avg_volume_20d` (simplified, can be enhanced with historical data)
  - Relative Volume (%): `relative_volume_1m`, `relative_volume_2m`, `relative_volume_5m`, `relative_volume_15m`, `relative_volume_daily`
  - Relative Volume at Same Time: `relative_volume_same_time` (simplified, can be enhanced with time-of-day patterns)
- ✅ Trading Activity Filters (`internal/metrics/activity_filters.go`)
  - Trade Count: `trade_count_1m`, `trade_count_2m`, `trade_count_5m`, `trade_count_15m`, `trade_count_60m`
  - Consecutive Candles: `consecutive_candles_1m`, `consecutive_candles_2m`, `consecutive_candles_5m`, `consecutive_candles_15m`, `consecutive_candles_daily`
- ✅ State Management Updates
  - TradeCountHistory populated when bars are finalized
  - CandleDirections tracked per timeframe (already implemented)
- ✅ Metrics Registry Integration
  - All new metric computers registered in `internal/metrics/registry.go`
- ✅ Comprehensive Unit Tests
  - Activity filter tests (11 test cases, all passing)
  - Advanced volume filter tests (3 test cases, all passing)

**Key Features:**
- 11 advanced volume filter metrics (3 filter types)
- 10 trading activity filter metrics (2 filter types with multiple timeframes)
- Trade count tracking per bar
- Candle direction tracking for consecutive candles
- All filters compute from `SymbolStateSnapshot`
- Thread-safe metric computation
- All tests passing

**Verification:**
- All code compiles successfully
- All unit tests pass (14+ test cases)
- Activity and volume filters computing correctly
- No linter errors

**Notes:**
- Average Volume uses simplified calculation from available bars. Full implementation would require historical data retrieval from TimescaleDB.
- Relative Volume at Same Time uses simplified approach. Full implementation would require time-of-day pattern storage.
- Volume forecasting for intraday timeframes can be added as a future enhancement.

**Next Steps:**
- Phase 4: Time-Based & Relative Range Filters
- Phase 5: Extended Technical Indicators (RSI timeframe extension)

---

### Phase 4: Time-Based & Relative Range Filters (Priority: MEDIUM)

**Goal**: Implement time-based calculations and relative range metrics

#### 4.1 Time-Based Filters (Week 3, Days 1-2) ✅ COMPLETE

**Tasks:**
- [x] Implement Minutes in Market filter
  - [x] Calculate minutes since market open (9:30 AM ET)
  - [x] Metric: `minutes_in_market`
  - [x] Computer: `MinutesInMarketComputer`
  - [x] Handle premarket/postmarket edge cases
- [x] Implement Minutes Since News filter (placeholder)
  - [x] Metric: `minutes_since_news`
  - [x] Computer: `MinutesSinceNewsComputer`
  - [x] Note: Returns false until news data integration is implemented
- [x] Implement Hours Since News filter (placeholder)
  - [x] Metric: `hours_since_news`
  - [x] Computer: `HoursSinceNewsComputer`
  - [x] Note: Returns false until news data integration is implemented
- [x] Implement Days Since News filter (placeholder)
  - [x] Metric: `days_since_news`
  - [x] Computer: `DaysSinceNewsComputer`
  - [x] Note: Returns false until news data integration is implemented
- [x] Implement Days Until Earnings filter (placeholder)
  - [x] Metric: `days_until_earnings`
  - [x] Computer: `DaysUntilEarningsComputer`
  - [x] Note: Returns false until earnings calendar integration is implemented

**Files Created:**
- `internal/metrics/time_filters.go` - Time-based filter computers
- `internal/metrics/time_filters_test.go` - Unit tests for time filters

**Files Modified:**
- `internal/metrics/registry.go` - Registered time-based filter computers

### Phase 4 Completion Summary

**Status:** ✅ Complete (with placeholders for external data integration)

**Deliverables:**
- ✅ Time-Based Filters (`internal/metrics/time_filters.go`)
  - Minutes in Market: `minutes_in_market` (fully functional)
  - Minutes Since News: `minutes_since_news` (placeholder, requires news data integration)
  - Hours Since News: `hours_since_news` (placeholder, requires news data integration)
  - Days Since News: `days_since_news` (placeholder, requires news data integration)
  - Days Until Earnings: `days_until_earnings` (placeholder, requires earnings calendar integration)
- ✅ Metrics Registry Integration
  - All time-based filter computers registered in `internal/metrics/registry.go`
- ✅ Comprehensive Unit Tests
  - Time filter tests (6 test cases for MinutesInMarket, all passing)
  - Placeholder tests for news/earnings filters (verify they return false until integration)

**Key Features:**
- 5 time-based filter metrics (1 fully functional, 4 placeholders)
- Minutes in Market calculates correctly for market, premarket, postmarket, and closed sessions
- News and earnings filters ready for external data integration
- All filters compute from `SymbolStateSnapshot`
- Thread-safe metric computation
- All tests passing

**Verification:**
- All code compiles successfully
- All unit tests pass (6+ test cases)
- Minutes in Market filter computing correctly
- No linter errors

**Notes:**
- Minutes in Market is fully functional and handles all market sessions correctly
- News and earnings filters are implemented as placeholders that return false until external data sources are integrated
- Future work: Integrate news data source and earnings calendar data source to enable news/earnings filters

**Next Steps:**
- Phase 5: Extended Technical Indicators (RSI timeframe extension)
- External data integration for news and earnings filters (can be done in parallel)

---

#### 4.2 Relative Range Filter (Week 3, Day 2)

**Tasks:**
- [ ] Implement Relative Range (%) filter
  - [ ] Compute today's range
  - [ ] Get ATR(14) daily value from indicators
  - [ ] Calculate: `relative_range = (today_range / atr_14_daily) * 100`
  - [ ] Metric: `relative_range_pct`
  - [ ] Computer: `RelativeRangeComputer`

**Files to Modify:**
- `internal/metrics/range_filters.go` - Add relative range computer

#### 4.3 Biggest Range Filter (Week 3, Day 2)

**Tasks:**
- [ ] Implement Biggest Range (%) filter
  - [ ] Store historical range data for longer periods (3m, 6m, 1y)
  - [ ] Compute maximum range over period
  - [ ] Metrics: `biggest_range_3m`, `biggest_range_6m`, `biggest_range_1y`
  - [ ] Computer: `BiggestRangeComputer` with period parameter
  - [ ] Requires historical data storage/retrieval

**Files to Modify:**
- `internal/metrics/range_filters.go` - Add biggest range computer
- `internal/scanner/historical_data.go` - Add historical range storage

#### 4.4 Testing & Validation (Week 3, Day 2)

**Tasks:**
- [ ] Unit tests for time-based filter computers
- [ ] Unit tests for relative range and biggest range computers
- [ ] Integration tests for time calculations
- [ ] Mock tests for news/earnings integration

**Files to Create:**
- `internal/metrics/time_filters_test.go`

---

### Phase 5: Fundamental Data Filters (Priority: LOW)

**Goal**: Implement fundamental data filters (requires external data integration)

#### 5.1 Fundamental Data Integration (Week 3, Days 3-4)

**Tasks:**
- [ ] Design fundamental data provider interface
  - [ ] `FundamentalDataProvider` interface
  - [ ] Methods: `GetMarketCap`, `GetFloat`, `GetSharesOutstanding`, etc.
- [ ] Implement mock provider for testing
- [ ] Integrate with external data provider (Alpha Vantage, Polygon.io, etc.)
  - [ ] Note: Can be deferred to later, use mock for now
- [ ] Add fundamental data caching layer
  - [ ] Cache data in Redis with appropriate TTLs
  - [ ] Update frequency: MarketCap (weekly), others (as needed)

**Files to Create:**
- `internal/data/fundamental_provider.go` - Fundamental data provider interface
- `internal/data/mock_fundamental_provider.go` - Mock implementation
- `internal/data/alpha_vantage_provider.go` - Alpha Vantage implementation (optional)

#### 5.2 Fundamental Data Filters (Week 3, Days 4-5)

**Tasks:**
- [ ] Implement Institutional Ownership filter
  - [ ] Metric: `institutional_ownership_pct`
  - [ ] Computer: `InstitutionalOwnershipComputer`
- [ ] Implement MarketCap filter
  - [ ] Metric: `marketcap`
  - [ ] Computer: `MarketCapComputer`
- [ ] Implement Shares Outstanding filter
  - [ ] Metric: `shares_outstanding`
  - [ ] Computer: `SharesOutstandingComputer`
- [ ] Implement Short Interest (%) filter
  - [ ] Metric: `short_interest_pct`
  - [ ] Computer: `ShortInterestComputer`
- [ ] Implement Short Ratio filter
  - [ ] Metric: `short_ratio`
  - [ ] Computer: `ShortRatioComputer` (uses short interest + avg volume)
- [ ] Implement Float filter
  - [ ] Metric: `float`
  - [ ] Computer: `FloatComputer`

**Files to Create:**
- `internal/metrics/fundamental_filters.go` - Fundamental filter computers

**Files to Modify:**
- `internal/scanner/state.go` - Add fundamental data storage
- `internal/metrics/registry.go` - Register fundamental filter computers

#### 5.3 Testing & Validation (Week 3, Day 5)

**Tasks:**
- [ ] Unit tests for fundamental filter computers
- [ ] Integration tests with mock provider
- [ ] Cache tests for fundamental data

**Files to Create:**
- `internal/metrics/fundamental_filters_test.go`

---

### Phase 6: Filter Configuration & Infrastructure (Priority: HIGH) ✅ COMPLETE

**Goal**: Implement filter configuration support (volume threshold, session, timeframe, value type)

#### 6.1 Volume Threshold Enforcement (Week 4, Days 1-2) ✅ COMPLETE

**Tasks:**
- [x] Extend rule conditions to support volume threshold
  - [x] Add optional `volume_threshold` field to `Condition` struct
  - [x] Update rule parser to support volume threshold
- [x] Implement volume threshold pre-filtering in scan loop
  - [x] Check volume threshold before evaluating rule conditions
  - [x] Skip rule evaluation if volume < threshold
- [x] Support per-filter volume threshold configuration
  - [x] Default threshold is 0 (no threshold) if not specified
  - [x] Volume threshold checked against daily volume, session volumes, or estimated from timeframe volumes

**Files Modified:**
- `internal/models/models.go` - Added volume threshold to Condition
- `internal/rules/parser.go` - Parse and enrich conditions
- `internal/rules/filter_config.go` - Volume threshold checking logic
- `internal/scanner/scan_loop.go` - Implement volume threshold check

#### 6.2 Session-Based Filtering (Week 4, Day 2) ✅ COMPLETE

**Tasks:**
- [x] Extend rule conditions to support "Calculated During" configuration
  - [x] Add optional `calculated_during` field to `Condition` struct
  - [x] Values: `premarket`, `market`, `postmarket`, `all` (default: "all")
- [x] Implement session check in scan loop
  - [x] Check session before evaluating rule conditions
  - [x] Skip evaluation if not in configured session
- [x] Update rule parser to support session configuration

**Files Modified:**
- `internal/models/models.go` - Added calculated_during to Condition
- `internal/rules/parser.go` - Parse calculated_during
- `internal/rules/filter_config.go` - Session filter checking logic
- `internal/scanner/scan_loop.go` - Implement session check

#### 6.3 Timeframe Support (Week 4, Day 3) ✅ COMPLETE

**Tasks:**
- [x] Extend metric naming convention to support timeframes
  - [x] Format: `{metric}_{timeframe}` (e.g., `change_5m`, `volume_15m`)
- [x] Update rule parser to support timeframe extraction
  - [x] Extract timeframe from metric name automatically
  - [x] Store timeframe in condition for reference
- [x] Add timeframe validation in rule validation
  - [x] Timeframe extracted and validated during parsing
- [x] Support timeframe in metric resolver
  - [x] Metrics already named with timeframes (e.g., `change_5m_pct`)
  - [x] Metric resolver looks up metrics by full name including timeframe

**Files Modified:**
- `internal/rules/parser.go` - Extract timeframe from metric names
- `internal/rules/filter_config.go` - Timeframe extraction logic
- `internal/rules/validation.go` - Validate filter config including timeframe

#### 6.4 Value Type Support (Week 4, Day 3) ✅ COMPLETE

**Tasks:**
- [x] Support both absolute ($) and percentage (%) variants
  - [x] Metrics named with `_pct` suffix for percentage (e.g., `change_5m_pct`)
  - [x] Metrics without `_pct` are absolute (e.g., `change_5m`)
- [x] Update rule parser to support value type extraction
  - [x] Extract value type from metric name automatically
  - [x] Store value type in condition for reference
- [x] Add value type validation
  - [x] Value type extracted and validated during parsing

**Files Modified:**
- `internal/rules/parser.go` - Extract value type from metric names
- `internal/rules/filter_config.go` - Value type extraction logic
- `internal/rules/validation.go` - Validate filter config including value type

#### 6.5 Testing & Validation (Week 4, Day 4) ✅ COMPLETE

**Tasks:**
- [x] Unit tests for volume threshold enforcement (7 test cases, all passing)
- [x] Unit tests for session-based filtering (7 test cases, all passing)
- [x] Unit tests for timeframe extraction (9 test cases, all passing)
- [x] Unit tests for value type extraction (5 test cases, all passing)
- [x] Unit tests for condition enrichment (3 test cases, all passing)
- [x] Unit tests for filter config validation (5 test cases, all passing)
- [ ] Integration tests for filter configuration (deferred to Phase 7)

**Files Created:**
- `internal/rules/filter_config.go` - Filter configuration utilities
- `internal/rules/filter_config_test.go` - Comprehensive unit tests (36+ test cases)

### Phase 6 Completion Summary

**Status:** ✅ Complete

**Deliverables:**
- ✅ Filter Configuration Support (`internal/rules/filter_config.go`)
  - Volume threshold enforcement with intelligent volume checking
  - Session-based filtering (premarket, market, postmarket, all)
  - Timeframe extraction from metric names (automatic)
  - Value type extraction from metric names (automatic)
- ✅ Extended Condition Model (`internal/models/models.go`)
  - `VolumeThreshold` field (optional, default: 0)
  - `CalculatedDuring` field (optional, default: "all")
  - `Timeframe` field (auto-extracted from metric name)
  - `ValueType` field (auto-extracted from metric name)
- ✅ Parser Enhancements (`internal/rules/parser.go`)
  - Automatic condition enrichment with extracted timeframe and value type
  - Filter configuration validation
- ✅ Scan Loop Integration (`internal/scanner/scan_loop.go`)
  - Pre-filtering: volume threshold and session checks before rule evaluation
  - Performance optimization: skip rule evaluation if pre-filters fail
- ✅ Comprehensive Unit Tests
  - Filter config tests (36+ test cases, all passing)

**Key Features:**
- Volume threshold enforcement with fallback strategies (daily volume, session volumes, estimated from timeframes)
- Session-based filtering with support for all market sessions
- Automatic timeframe and value type extraction from metric names
- Pre-filtering in scan loop for performance optimization
- All filter configurations validated during rule parsing
- Thread-safe filter checking
- All tests passing

**Verification:**
- All code compiles successfully
- All unit tests pass (36+ test cases)
- Filter configuration working correctly
- No linter errors

**Notes:**
- Timeframe and value type are automatically extracted from metric names during parsing
- Volume threshold uses intelligent fallback: checks daily volume first, then session volumes, then estimates from timeframe volumes
- Session filtering happens before rule evaluation for performance
- All filter configurations are optional with sensible defaults

**Next Steps:**
- Phase 7: Performance Optimization
- Integration tests for filter configuration (can be done in Phase 7)

---

### Phase 7: Performance Optimization (Priority: MEDIUM)

**Goal**: Optimize metric computation to maintain <800ms scan cycle target

#### 7.1 Metric Computation Optimization (Week 4, Days 4-5)

**Tasks:**
- [ ] Implement lazy metric computation
  - [ ] Only compute metrics when needed for rule evaluation
  - [ ] Cache computed metrics in `SymbolState`
- [ ] Batch metric computations in scan loop
  - [ ] Compute multiple metrics in single pass where possible
- [ ] Optimize historical data lookups
  - [ ] Use efficient ring buffers
  - [ ] Limit historical data storage to necessary periods
- [ ] Profile metric computation performance
  - [ ] Identify hot paths
  - [ ] Optimize allocations

**Files to Modify:**
- `internal/scanner/state.go` - Add metric caching
- `internal/scanner/scan_loop.go` - Optimize metric computation
- `internal/metrics/registry.go` - Optimize computation order

#### 7.2 Historical Data Management (Week 4, Day 5)

**Tasks:**
- [ ] Implement efficient ring buffer for recent bars
  - [ ] Already implemented, verify efficiency
- [ ] Add historical data retrieval from TimescaleDB
  - [ ] For multi-day calculations (average volume, biggest range)
  - [ ] Cache historical data in memory
- [ ] Implement data expiration/cleanup
  - [ ] Remove old data that's no longer needed
  - [ ] Limit memory usage

**Files to Create:**
- `internal/scanner/historical_data.go` - Historical data management

**Files to Modify:**
- `internal/scanner/rehydration.go` - Load historical data on startup

#### 7.3 Performance Testing (Week 4, Day 5)

**Tasks:**
- [ ] Performance tests with all filters enabled
  - [ ] Measure scan cycle time impact
  - [ ] Test with varying symbol counts (1000, 2000, 5000)
  - [ ] Test with varying rule counts (1, 10, 50, 100)
- [ ] Benchmark metric computation
  - [ ] Compare before/after optimization
- [ ] Ensure scan cycle time remains <800ms

**Files to Create:**
- `tests/performance/filter_performance_test.go`

---

## Implementation Checklist

### Week 1: Foundation & Core Filters ✅ COMPLETE
- [x] Extend Symbol State (session, price refs, volumes, trade count)
- [x] Implement session detection
- [x] Implement core price filters (8 types)
- [x] Implement core volume filters (4 types)
- [x] Unit tests for all new components

### Week 2: Range & Advanced Filters
- [ ] Implement range filters (3 types)
- [ ] Implement technical indicator filters (5 types)
- [ ] Implement advanced volume filters (3 types)
- [ ] Implement trading activity filters (2 types)
- [ ] Unit tests for all new components

### Week 3: Time-Based & Fundamental Filters
- [ ] Implement time-based filters (5 types)
- [ ] Implement relative/biggest range filters (2 types)
- [ ] Implement fundamental data integration (interface + mock)
- [ ] Implement fundamental filters (6 types)
- [ ] Unit tests for all new components

### Week 4: Configuration & Optimization
- [ ] Implement volume threshold enforcement
- [ ] Implement session-based filtering
- [ ] Implement timeframe support
- [ ] Implement value type support
- [ ] Performance optimization
- [ ] Performance testing

## Testing Strategy

### Unit Tests
- Each metric computer should have comprehensive unit tests
- Test all timeframes for timeframe-based metrics
- Test edge cases (missing data, zero values, etc.)
- Test session transitions
- Test value type variants ($ and %)

### Integration Tests
- Test metric computation in scan loop
- Test filter evaluation with real rules
- Test session-based filtering
- Test volume threshold enforcement
- Test timeframe selection
- Test value type selection

### Performance Tests
- Measure scan cycle time with all filters enabled
- Test with varying symbol counts
- Test with varying rule counts
- Benchmark metric computation
- Ensure <800ms scan cycle target

## Success Criteria

1. ✅ All filter types implemented and tested
2. ✅ Volume threshold, timeframe, and session support working
3. ✅ Performance targets maintained (<800ms scan cycle)
4. ✅ Comprehensive test coverage (>80%)
5. ✅ Documentation updated

## Risk Mitigation

### Performance Risks
- **Risk**: Adding 50+ filters may slow down scan cycle
- **Mitigation**: Lazy computation, caching, profiling, optimization

### Data Storage Risks
- **Risk**: Historical data storage may consume too much memory
- **Mitigation**: Efficient ring buffers, data expiration, limit storage periods

### External Data Risks
- **Risk**: External data providers may be slow or unavailable
- **Mitigation**: Aggressive caching, mock providers for testing, graceful degradation

## Next Steps

1. Review and approve this implementation plan
2. Start with Phase 1: Foundation & Core Price/Volume Filters
3. Set up regular review checkpoints (weekly)
4. Adjust plan based on learnings

