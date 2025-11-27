# Phase 5.3: Filter Implementation Plan - Summary

## Quick Overview

**Goal**: Implement 50+ filter types with support for volume thresholds, timeframes, session-based filtering, and value types.

**Timeline**: 4 weeks (can be adjusted based on priorities)

**Current State**: 
- ✅ Metrics registry system in place
- ✅ Basic metrics implemented (price, volume, VWAP, price_change_1m/5m/15m_pct)
- ✅ Symbol state management working
- ❌ Need to extend SymbolState for sessions, historical data, etc.
- ❌ Need to implement 50+ filter metrics

## Implementation Phases

### Phase 1: Foundation & Core Filters (Week 1) - HIGH PRIORITY
**Focus**: Infrastructure + most commonly used filters

1. **Extend Symbol State**
   - Session tracking (Pre-Market, Market, Post-Market)
   - Price references (yesterday's close, today's open/close)
   - Session-specific volume tracking
   - Trade count tracking
   - Candle direction tracking

2. **Session Detection**
   - Pre-Market: 4:00 AM - 9:30 AM ET
   - Market: 9:30 AM - 4:00 PM ET
   - Post-Market: 4:00 PM - 8:00 PM ET

3. **Core Price Filters** (8 types)
   - Change ($) with timeframes
   - Change from Close ($ and %)
   - Change from Close (Premarket/Post Market)
   - Change from Open ($ and %)
   - Extended Percentage Change (%)
   - Gap from Close ($ and %)

4. **Core Volume Filters** (4 types)
   - Postmarket Volume
   - Premarket Volume
   - Absolute Volume (all timeframes)
   - Absolute Dollar Volume (all timeframes)

### Phase 2: Range & Technical Indicators (Week 2) - HIGH PRIORITY
**Focus**: Range calculations and indicator distances

1. **Range Filters** (3 types)
   - Range ($) with timeframes
   - Percentage Range (%) with timeframes
   - Position in Range (%) with timeframes

2. **Technical Indicator Filters** (5 types)
   - Extended RSI(14) with multiple timeframes
   - ATR(14) calculation
   - ATRP(14) calculation
   - Extended Distance from VWAP
   - Distance from Moving Average (all MA types)

### Phase 3: Advanced Volume & Trading Activity (Week 2) - MEDIUM PRIORITY
**Focus**: Relative volume and trading activity metrics

1. **Advanced Volume Filters** (3 types)
   - Average Volume (5d, 10d, 20d)
   - Relative Volume (%) with forecasting
   - Relative Volume (%) at Same Time

2. **Trading Activity Filters** (2 types)
   - Trade Count with timeframes
   - Consecutive Candles (green/red counting)

### Phase 4: Time-Based & Relative Range (Week 3) - MEDIUM PRIORITY
**Focus**: Time calculations and relative range metrics

1. **Time-Based Filters** (5 types)
   - Minutes in Market
   - Minutes/Hours/Days Since News
   - Days Until Earnings

2. **Range Filters** (2 types)
   - Relative Range (%) vs ATR(14)
   - Biggest Range (%) (3m, 6m, 1y)

### Phase 5: Fundamental Data (Week 3) - LOW PRIORITY
**Focus**: External data integration (can be deferred)

1. **Fundamental Data Integration**
   - Provider interface
   - Mock provider for testing
   - External provider integration (optional)

2. **Fundamental Filters** (6 types)
   - Institutional Ownership
   - MarketCap
   - Shares Outstanding
   - Short Interest (%)
   - Short Ratio
   - Float

### Phase 6: Filter Configuration (Week 4) - HIGH PRIORITY
**Focus**: Support for filter configuration options

1. **Volume Threshold Enforcement**
   - Pre-filtering in scan loop
   - Per-filter threshold configuration

2. **Session-Based Filtering**
   - "Calculated During" configuration
   - Session check in scan loop

3. **Timeframe Support**
   - Timeframe in metric naming
   - Timeframe validation
   - Timeframe in metric resolver

4. **Value Type Support**
   - Both $ and % variants
   - Value type validation

### Phase 7: Performance Optimization (Week 4) - MEDIUM PRIORITY
**Focus**: Maintain <800ms scan cycle target

1. **Metric Computation Optimization**
   - Lazy computation
   - Metric caching
   - Batch computations

2. **Historical Data Management**
   - Efficient storage
   - Data expiration
   - Memory limits

3. **Performance Testing**
   - Scan cycle time with all filters
   - Varying symbol/rule counts
   - Benchmarks

## Key Design Decisions

### 1. Metric Computer Pattern
- Each filter type is a `MetricComputer` implementation
- Metrics registered in `Registry`
- Computed on-demand from `SymbolStateSnapshot`
- Supports dependencies and ordering

### 2. Symbol State Extensions
- Add session tracking fields
- Add price reference fields (yesterday close, today open)
- Add session-specific volume tracking
- Add trade count and candle direction tracking
- Use ring buffers for historical data

### 3. Session Detection
- Centralized session detection utility
- Updates on tick/bar processing
- Handles timezone conversions (ET to UTC)
- Resets session-specific data on transitions

### 4. Timeframe Support
- Metric naming: `{metric}_{timeframe}` (e.g., `change_5m`, `volume_15m`)
- Timeframe validation in rule parser
- Support in metric resolver

### 5. Value Type Support
- Both absolute ($) and percentage (%) variants
- Two metrics: `{metric}` and `{metric}_pct`
- Rule conditions can reference either

### 6. Volume Threshold
- Pre-filtering step (before rule evaluation)
- Per-filter configuration
- Optional in rule conditions

### 7. Session-Based Filtering
- "Calculated During" configuration per filter
- Session check before rule evaluation
- Skip evaluation if not in configured session

## File Structure

### New Files to Create

**Session & State:**
- `internal/scanner/session.go` - Session detection utilities
- `internal/scanner/historical_data.go` - Historical data management
- `internal/scanner/external_data.go` - External data interface

**Metric Computers:**
- `internal/metrics/price_filters.go` - Price filter computers
- `internal/metrics/volume_filters.go` - Volume filter computers
- `internal/metrics/advanced_volume_filters.go` - Advanced volume computers
- `internal/metrics/range_filters.go` - Range filter computers
- `internal/metrics/indicator_filters.go` - Indicator distance computers
- `internal/metrics/activity_filters.go` - Trading activity computers
- `internal/metrics/time_filters.go` - Time-based filter computers
- `internal/metrics/fundamental_filters.go` - Fundamental filter computers

**External Data:**
- `internal/data/fundamental_provider.go` - Fundamental data provider interface
- `internal/data/mock_fundamental_provider.go` - Mock implementation

**Tests:**
- `internal/scanner/session_test.go`
- `internal/metrics/*_filters_test.go` (one per filter category)
- `internal/rules/filter_config_test.go`
- `tests/performance/filter_performance_test.go`

### Files to Modify

**Core:**
- `internal/scanner/state.go` - Extend SymbolState
- `internal/scanner/tick_consumer.go` - Session tracking, trade count
- `internal/scanner/bar_handler.go` - Session tracking, candle direction
- `internal/scanner/scan_loop.go` - Volume threshold, session checks

**Metrics:**
- `internal/metrics/registry.go` - Register all new computers
- `internal/metrics/price_change_metrics.go` - Extend for more timeframes

**Rules:**
- `internal/models/models.go` - Add filter configuration fields
- `internal/rules/parser.go` - Parse filter configuration
- `internal/rules/validation.go` - Validate filter configuration
- `internal/rules/metrics.go` - Support timeframes/value types

**Indicators:**
- `pkg/indicator/techan_adapter.go` - Add ATR if needed
- `internal/indicator/engine.go` - Add ATR calculation

## Testing Strategy

### Unit Tests
- Each metric computer: comprehensive tests
- All timeframes for timeframe-based metrics
- Edge cases (missing data, zero values)
- Session transitions
- Value type variants

### Integration Tests
- Metric computation in scan loop
- Filter evaluation with real rules
- Session-based filtering
- Volume threshold enforcement
- Timeframe/value type selection

### Performance Tests
- Scan cycle time with all filters
- Varying symbol counts (1000, 2000, 5000)
- Varying rule counts (1, 10, 50, 100)
- Benchmark metric computation
- Target: <800ms scan cycle

## Success Criteria

1. ✅ All 50+ filter types implemented
2. ✅ Volume threshold, timeframe, session support working
3. ✅ Performance targets maintained (<800ms scan cycle)
4. ✅ Comprehensive test coverage (>80%)
5. ✅ Documentation updated

## Risk Mitigation

### Performance
- Lazy metric computation
- Caching computed metrics
- Profiling and optimization
- Batch computations

### Memory
- Efficient ring buffers
- Data expiration/cleanup
- Limit historical data storage

### External Data
- Aggressive caching
- Mock providers for testing
- Graceful degradation

## Recommended Approach

1. **Start with Phase 1** (Foundation & Core Filters)
   - Most commonly used filters
   - Establishes infrastructure
   - Quick wins

2. **Continue with Phase 2** (Range & Technical Indicators)
   - High-value filters
   - Builds on Phase 1

3. **Phase 3-5** can be done in parallel or deferred
   - Advanced features
   - External data can be mocked initially

4. **Phase 6** (Configuration) is critical
   - Enables full filter functionality
   - Should be done early if possible

5. **Phase 7** (Optimization) is ongoing
   - Profile as you go
   - Optimize hot paths
   - Final performance validation

## Questions to Consider

1. **Priority**: Which filters are most important for MVP?
   - Focus on Phase 1-2 first?
   - Defer Phase 5 (Fundamental Data)?

2. **External Data**: When to integrate real providers?
   - Mock initially, integrate later?
   - Integrate Alpha Vantage/Polygon.io now?

3. **Performance**: Acceptable scan cycle time?
   - Current target: <800ms
   - Can we relax for MVP?

4. **Testing**: Test coverage target?
   - Minimum: 80%
   - Can we start with 70% and increase?

## Next Steps

1. ✅ Review this plan
2. ✅ Decide on priorities (which phases to do first)
3. ✅ Start Phase 1: Foundation & Core Filters
4. ✅ Set up weekly review checkpoints
5. ✅ Adjust plan based on learnings

