# Filter Implementation Guide

This document provides a comprehensive overview of all filters that the stock scanner application needs to support, along with implementation steps for each filter category.

## Filter Configuration Structure

Each filter supports the following configuration options:

1. **Volume Threshold**: Minimum volume required for the filter to be evaluated
2. **Calculated During**: When the filter is active (Pre-Market, Market, Post-Market)
3. **Timeframes**: For time-based metrics (e.g., 1 min, 5 min, 15 min, daily)
4. **Value Types**: For metrics that can be expressed in different units ($ or %)

## Filter Categories

### 1. Volume Filters

#### 1.1 Postmarket Volume
- **Description**: The volume traded in the postmarket session (during premarket session, postmarket and premarket volume are added together).
- **Volume Threshold**: Configurable (default: 0)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Implementation Steps**:
  1. Track postmarket volume separately in `SymbolState`
  2. Add `postmarket_volume` field to state
  3. Update volume tracking logic to distinguish market sessions
  4. Add metric resolver: `postmarket_volume`
  5. Support session-based filtering in scan loop

#### 1.2 Premarket Volume
- **Description**: The volume traded in the premarket session.
- **Volume Threshold**: Configurable (default: 0)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Implementation Steps**:
  1. Track premarket volume separately in `SymbolState`
  2. Add `premarket_volume` field to state
  3. Update volume tracking logic to distinguish market sessions
  4. Add metric resolver: `premarket_volume`
  5. Support session-based filtering in scan loop

#### 1.3 Absolute Volume
- **Description**: Totalled volume of the stock in a given period of time (e.g., last 5 minutes or the current day).
- **Volume Threshold**: Configurable (default: 0)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Timeframes**: 1 min, 2 min, 5 min, 10 min, 15 min, 30 min, 60 min, daily
- **Implementation Steps**:
  1. Extend `getMetricsFromSnapshot` to compute volume over timeframes
  2. Sum volume from finalized bars for specified timeframe
  3. Add metrics: `volume_1m`, `volume_5m`, `volume_15m`, `volume_60m`, `volume_daily`
  4. Support timeframe selection in rule conditions

#### 1.4 Absolute Dollar Volume
- **Description**: Totalled dollar volume of the stock in a given period of time (e.g., last 5 minutes or the current day).
- **Volume Threshold**: Configurable (default: 10,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Timeframes**: 1 min, 5 min, 15 min, 60 min, daily
- **Implementation Steps**:
  1. Compute dollar volume = price × volume for each bar
  2. Sum dollar volume over specified timeframe
  3. Add metrics: `dollar_volume_1m`, `dollar_volume_5m`, `dollar_volume_15m`, `dollar_volume_60m`, `dollar_volume_daily`
  4. Support timeframe selection in rule conditions

#### 1.5 Average Volume
- **Description**: Average daily volume of the last 5, 10, or 20 days.
- **Volume Threshold**: Configurable (default: 0)
- **Calculated During**: Pre-Market, Market
- **Timeframes**: 5 Day, 10 Day, 20 Day
- **Implementation Steps**:
  1. Store historical daily volumes in `SymbolState` (ring buffer)
  2. Compute average from last N days
  3. Add metrics: `avg_volume_5d`, `avg_volume_10d`, `avg_volume_20d`
  4. Requires historical data storage/retrieval

#### 1.6 Relative Volume (%)
- **Description**: The relative volume of the current candle compared to the average volume of the last 10 candles of a given timeframe (e.g., 5 or 15 minute candles). Formula: `relative_volume = (current_candle_volume / average_candle_volume) * 100`. The current candle volume is a calculated forecast: e.g., when 30 seconds of a 1 min candle have passed, the current volume is multiplied by 2 to forecast the volume at the end of the 1 min timeframe. Except for the rel. daily volume, we use the current volume and do not forecast the daily volume.
- **Volume Threshold**: Configurable (default: 1,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Timeframes**: 1 min, 2 min, 5 min, 15 min, daily
- **Implementation Steps**:
  1. Track average volume of last 10 candles per timeframe
  2. Compute current candle volume (with forecast for intraday timeframes)
  3. Calculate relative volume percentage
  4. Add metrics: `relative_volume_1m`, `relative_volume_5m`, `relative_volume_15m`, `relative_volume_daily`
  5. Implement volume forecasting logic for intraday timeframes

#### 1.7 Relative Volume (%) at Same Time
- **Description**: The relative volume of a stock compared to the volume that is normally traded at the same time. So if it's 10:00 and the value is 200%, it means that the stock was traded 2 times the amount it would normally have traded until 10:00 in the last days.
- **Volume Threshold**: Configurable (default: 50,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Implementation Steps**:
  1. Store historical volume patterns by time of day
  2. Compute average volume at same time of day over last N days
  3. Compare current volume to historical average
  4. Add metric: `relative_volume_same_time`
  5. Requires time-of-day pattern storage

### 2. Price Filters

#### 2.1 Price ($)
- **Description**: The current price of a ticker.
- **Volume Threshold**: Configurable (default: 0)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Implementation Steps**:
  1. Already implemented as `price` metric in `getMetricsFromSnapshot`
  2. No additional work needed

#### 2.2 Change ($)
- **Description**: Absolute change of the stock in a given period of time (e.g., last 5 minutes or 30 minutes).
- **Volume Threshold**: Configurable (default: 10,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Timeframes**: 1 min, 2 min, 5 min, 15 min, 30 min, 60 min
- **Implementation Steps**:
  1. Compute absolute price change from finalized bars
  2. Add metrics: `change_1m`, `change_5m`, `change_15m`, `change_30m`, `change_60m`
  3. Formula: `change = current_price - price_N_minutes_ago`

#### 2.3 Change from Close
- **Description**: The difference between yesterday close price and the current price as absolute or relative value.
- **Volume Threshold**: Configurable (default: 1,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Value Types**: ($) or (%)
- **Implementation Steps**:
  1. Store yesterday's close price in `SymbolState`
  2. Compute absolute change: `change_from_close = current_price - yesterday_close`
  3. Compute percentage change: `change_from_close_pct = (change_from_close / yesterday_close) * 100`
  4. Add metrics: `change_from_close`, `change_from_close_pct`
  5. Requires previous day's close storage

#### 2.4 Change from Close (Premarket)
- **Description**: The difference between yesterday close and the current price in premarket as absolute or relative value (does only work in premarket).
- **Volume Threshold**: Configurable (default: 0)
- **Calculated During**: Pre-Market only
- **Value Types**: ($) or (%)
- **Implementation Steps**:
  1. Similar to "Change from Close" but only active during premarket
  2. Add session check in scan loop
  3. Add metrics: `change_from_close_premarket`, `change_from_close_premarket_pct`

#### 2.5 Change from Close (Post Market)
- **Description**: The difference between today close and the current price in postmarket as absolute or relative value (does only work in postmarket).
- **Volume Threshold**: Configurable (default: 0)
- **Calculated During**: Post-Market only
- **Value Types**: ($) or (%)
- **Implementation Steps**:
  1. Store today's close price
  2. Compute change from today's close during postmarket
  3. Add metrics: `change_from_close_postmarket`, `change_from_close_postmarket_pct`
  4. Add session check in scan loop

#### 2.6 Change from Open
- **Description**: The difference between today's open price and the current price as absolute or relative value.
- **Volume Threshold**: Configurable (default: 10,000)
- **Calculated During**: Market, Post-Market
- **Value Types**: ($) or (%)
- **Implementation Steps**:
  1. Store today's open price in `SymbolState`
  2. Compute absolute change: `change_from_open = current_price - today_open`
  3. Compute percentage change: `change_from_open_pct = (change_from_open / today_open) * 100`
  4. Add metrics: `change_from_open`, `change_from_open_pct`

#### 2.7 Percentage Change (%)
- **Description**: Relative change of the stock in a given period of time (e.g., last 5 minutes or 10 days).
- **Volume Threshold**: Configurable (default: 0)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Timeframes**: 1 min, 2 min, 5 min, 10 min, 15 min, 30 min, 60 min, 2 hour, 4 hour, 2 Day, 5 Day, 10 Day, 20 Day
- **Implementation Steps**:
  1. Extend existing `price_change_Nm_pct` metrics
  2. Add support for longer timeframes (hours, days)
  3. Add metrics: `change_pct_1m`, `change_pct_5m`, `change_pct_15m`, `change_pct_60m`, `change_pct_2h`, `change_pct_4h`, `change_pct_2d`, `change_pct_5d`, `change_pct_10d`, `change_pct_20d`
  4. Requires historical bar storage for multi-day calculations

#### 2.8 Gap from Close
- **Description**: The difference between yesterday close price and today's open price as absolute or relative value.
- **Volume Threshold**: Configurable (default: 75,000)
- **Calculated During**: Pre-Market, Market
- **Value Types**: ($) or (%)
- **Implementation Steps**:
  1. Store yesterday's close and today's open
  2. Compute absolute gap: `gap_from_close = today_open - yesterday_close`
  3. Compute percentage gap: `gap_from_close_pct = (gap_from_close / yesterday_close) * 100`
  4. Add metrics: `gap_from_close`, `gap_from_close_pct`

### 3. Range Filters

#### 3.1 Range ($)
- **Description**: Absolute Range (distance between lowest to highest price) of the stock in a given period of time (e.g., last 5 minutes or 10 days).
- **Volume Threshold**: Configurable (default: 0)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Timeframes**: 2 min, 5 min, 10 min, 15 min, 30 min, 60 min, today, 5 Day, 10 Day
- **Implementation Steps**:
  1. Compute range from finalized bars: `range = high - low` over timeframe
  2. Add metrics: `range_2m`, `range_5m`, `range_15m`, `range_30m`, `range_60m`, `range_today`, `range_5d`, `range_10d`
  3. Track high/low over specified period

#### 3.2 Percentage Range (%)
- **Description**: Relative Range (from lowest to highest price) of the stock in a given period of time (e.g., last 5 minutes or 10 days).
- **Volume Threshold**: Configurable (default: 0)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Timeframes**: 2 min, 5 min, 10 min, 15 min, 30 min, 60 min, today, 20 Day
- **Implementation Steps**:
  1. Compute range percentage: `range_pct = ((high - low) / low) * 100`
  2. Add metrics: `range_pct_2m`, `range_pct_5m`, `range_pct_15m`, `range_pct_30m`, `range_pct_60m`, `range_pct_today`, `range_pct_20d`

#### 3.3 Biggest Range (%)
- **Description**: Relative Range (from lowest to highest price) of the stock in a given period of time (e.g., last 5 minutes or 10 days).
- **Volume Threshold**: Configurable (default: 0)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Timeframes**: 3 Month, 6 Month, 1 Year
- **Implementation Steps**:
  1. Store historical range data for longer periods
  2. Compute maximum range over period
  3. Add metrics: `biggest_range_3m`, `biggest_range_6m`, `biggest_range_1y`
  4. Requires historical data storage for months/years

#### 3.4 Relative Range (%)
- **Description**: Compares the range today against the daily ATR(14). E.g., if the ATR is 10 and the stock has a range of 8 you will get 80%. If you set this to e.g. a min. of 200% you will only get stocks that moved twice the normal daily range.
- **Volume Threshold**: Configurable (default: 25,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Implementation Steps**:
  1. Compute today's range
  2. Get ATR(14) daily value
  3. Calculate: `relative_range = (today_range / atr_14_daily) * 100`
  4. Add metric: `relative_range_pct`
  5. Requires ATR(14) daily calculation

#### 3.5 Position in Range (%)
- **Description**: Gives the position of the current price in the range of a given period of time (e.g., 5 or 30 minutes). For example, if the 10 minute range of a stock is from 100 to 110 and the current price is 101 the result would be 10%.
- **Volume Threshold**: Configurable (default: 75,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Timeframes**: 2 min, 5 min, 15 min, 30 min, 60 min, today, 5 Day, 10 Day, 20 Day, 1 Year
- **Implementation Steps**:
  1. Compute high/low over timeframe
  2. Calculate position: `position_in_range = ((current_price - low) / (high - low)) * 100`
  3. Add metrics: `position_in_range_2m`, `position_in_range_5m`, `position_in_range_15m`, `position_in_range_30m`, `position_in_range_60m`, `position_in_range_today`, `position_in_range_5d`, `position_in_range_10d`, `position_in_range_20d`, `position_in_range_1y`

### 4. Technical Indicator Filters

#### 4.1 RSI (14)
- **Description**: The RSI 14 value for different timeframes (e.g., 5 or 60 minutes).
- **Volume Threshold**: Configurable (default: 50,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Timeframes**: 1 min, 2 min, 5 min, 15 min, daily
- **Implementation Steps**:
  1. RSI(14) already implemented in indicator engine
  2. Extend to support multiple timeframes
  3. Add metrics: `rsi_14_1m`, `rsi_14_5m`, `rsi_14_15m`, `rsi_14_daily`
  4. Requires RSI calculation per timeframe

#### 4.2 ATR (Average True Range)
- **Description**: The ATR(14) of a ticker for the given timeframe.
- **Volume Threshold**: Configurable (default: 0)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Timeframes**: 1 min, 5 min, daily
- **Implementation Steps**:
  1. Implement ATR(14) calculation in indicator engine
  2. Support multiple timeframes
  3. Add metrics: `atr_14_1m`, `atr_14_5m`, `atr_14_daily`
  4. ATR formula: `ATR = EMA(True Range, 14)` where `True Range = max(high-low, abs(high-prev_close), abs(low-prev_close))`

#### 4.3 ATRP (Average True Range Percentage)
- **Description**: The ATRP(14) of a ticker for the given timeframe (ATRP = (Average True Range / Close) * 100).
- **Volume Threshold**: Configurable (default: 0)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Timeframes**: 1 min, 5 min, daily
- **Implementation Steps**:
  1. Compute ATR(14) first
  2. Calculate ATRP: `ATRP = (ATR / Close) * 100`
  3. Add metrics: `atrp_14_1m`, `atrp_14_5m`, `atrp_14_daily`

#### 4.4 Distance from VWAP
- **Description**: The distance of the current price to the current VWAP of a stock as absolute or relative value.
- **Volume Threshold**: Configurable (default: 75,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Value Types**: ($) or (%)
- **Implementation Steps**:
  1. VWAP already computed in bars
  2. Compute absolute distance: `vwap_dist = abs(current_price - vwap)`
  3. Compute percentage distance: `vwap_dist_pct = (vwap_dist / vwap) * 100`
  4. Add metrics: `vwap_dist`, `vwap_dist_pct`
  5. Already partially implemented in toplist integration

#### 4.5 Distance from Moving Average
- **Description**: The distance of the current price to the current EMA/SMA of a stock as relative value.
- **Volume Threshold**: Configurable (default: 100,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Moving Average Options**: SMA(20) daily, SMA(10) daily, SMA(200) daily, EMA(20) 1 min, EMA(9) 1 min, EMA(9) 5 min, EMA(9) 15 min, EMA(21) 15 min, EMA(9) 60 min, EMA(21) 60 min, EMA(50) 15 min, EMA(50) daily
- **Implementation Steps**:
  1. EMA/SMA already implemented in indicator engine
  2. Compute distance: `ma_dist_pct = ((current_price - ma_value) / ma_value) * 100`
  3. Add metrics for each MA type: `ma_dist_sma20_daily_pct`, `ma_dist_ema9_5m_pct`, etc.
  4. Support multiple MA configurations

### 5. Trading Activity Filters

#### 5.1 Trade Count
- **Description**: Totalled trade count of the stock in a given period of time (e.g., last 5 minutes).
- **Volume Threshold**: Configurable (default: 0)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Timeframes**: 1 min, 2 min, 5 min, 15 min, 60 min
- **Implementation Steps**:
  1. Track trade count per symbol in `SymbolState`
  2. Increment counter on each tick
  3. Sum trade count over timeframe
  4. Add metrics: `trade_count_1m`, `trade_count_5m`, `trade_count_15m`, `trade_count_60m`
  5. Requires tick counting logic

#### 5.2 Consecutive Candles
- **Description**: The amount of consecutive green or red closed candles. A value of -5 means that there are currently 5 consecutive red closed candles. A positive value like 3 would mean that there are 3 consecutive green closed candles.
- **Volume Threshold**: Configurable (default: 75,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Timeframes**: 1 min, 2 min, 5 min, 15 min, daily
- **Implementation Steps**:
  1. Track candle direction (green/red) from finalized bars
  2. Count consecutive candles of same direction
  3. Positive for green, negative for red
  4. Add metrics: `consecutive_candles_1m`, `consecutive_candles_5m`, `consecutive_candles_15m`, `consecutive_candles_daily`
  5. Requires historical bar analysis

### 6. Time-Based Filters

#### 6.1 Minutes in Market
- **Description**: The amount of minutes since market open. For example 09:45 would return the value 15 since the market is open since 15 minutes.
- **Volume Threshold**: Configurable (default: 0)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Implementation Steps**:
  1. Calculate minutes since market open (9:30 AM ET)
  2. Add metric: `minutes_in_market`
  3. Handle premarket/postmarket edge cases

#### 6.2 Minutes Since News
- **Description**: Number of minutes since last (relevant) News (looks up to 24 hours in the past).
- **Volume Threshold**: Configurable (default: 0)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Implementation Steps**:
  1. Integrate news data source
  2. Store last news timestamp per symbol
  3. Calculate minutes since last news
  4. Add metric: `minutes_since_news`
  5. Requires news data integration

#### 6.3 Hours Since News
- **Description**: Number of hours since last (relevant) News (looks up to 24 hours in the past).
- **Volume Threshold**: Configurable (default: 0)
- **Calculated During**: Pre-Market, Market
- **Implementation Steps**:
  1. Similar to minutes since news
  2. Calculate hours instead of minutes
  3. Add metric: `hours_since_news`

#### 6.4 Days Since News (Number)
- **Description**: Number of days since last (relevant) News (looks up to 3 days in the past).
- **Volume Threshold**: Configurable (default: 0)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Implementation Steps**:
  1. Similar to hours/minutes since news
  2. Calculate days instead
  3. Add metric: `days_since_news`

#### 6.5 Days Until Earnings
- **Description**: Number of days until the next earnings.
- **Volume Threshold**: Configurable (default: 0)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Implementation Steps**:
  1. Integrate earnings calendar data source
  2. Store next earnings date per symbol
  3. Calculate days until earnings
  4. Add metric: `days_until_earnings`
  5. Requires earnings calendar integration

### 7. Fundamental Data Filters

#### 7.1 Institutional Ownership
- **Description**: The relative amount of shares held by institutional traders compared to the float of the Stock.
- **Volume Threshold**: Configurable (default: 0)
- **Calculated During**: Pre-Market, Market
- **Implementation Steps**:
  1. Integrate fundamental data provider (e.g., Alpha Vantage, Polygon.io)
  2. Store institutional ownership percentage per symbol
  3. Add metric: `institutional_ownership_pct`
  4. Requires external data source integration
  5. Cache data (updated infrequently)

#### 7.2 MarketCap
- **Description**: MarketCap of the stock, updated once a week.
- **Volume Threshold**: Configurable (default: 0)
- **Calculated During**: Pre-Market, Market
- **Implementation Steps**:
  1. Integrate fundamental data provider
  2. Store market cap per symbol
  3. Add metric: `marketcap`
  4. Requires external data source integration
  5. Cache data (updated weekly)

#### 7.3 Shares Outstanding
- **Description**: The amount of outstanding shares of a Company.
- **Volume Threshold**: Configurable (default: 0)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Implementation Steps**:
  1. Integrate fundamental data provider
  2. Store shares outstanding per symbol
  3. Add metric: `shares_outstanding`
  4. Requires external data source integration

#### 7.4 Short Interest (%)
- **Description**: The relative amount of shares shorted compared to the float of the Stock (shares shorted/float).
- **Volume Threshold**: Configurable (default: 0)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Implementation Steps**:
  1. Integrate short interest data provider
  2. Store short interest percentage per symbol
  3. Add metric: `short_interest_pct`
  4. Requires external data source integration

#### 7.5 Short Ratio
- **Description**: Also known as the "days to cover" ratio, is the amount of shares short divided by the amount of the average trading volume of a stock.
- **Volume Threshold**: Configurable (default: 0)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Implementation Steps**:
  1. Get short interest and average volume
  2. Calculate: `short_ratio = shares_short / avg_daily_volume`
  3. Add metric: `short_ratio`
  4. Requires short interest and volume data

#### 7.6 Float
- **Description**: The number of shares available for trading of a particular stock.
- **Volume Threshold**: Configurable (default: 0)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Implementation Steps**:
  1. Integrate fundamental data provider
  2. Store float per symbol
  3. Add metric: `float`
  4. Requires external data source integration

## Implementation Priority

### Phase 1: Core Metrics (High Priority)
1. Price filters (Price, Change, Change from Close/Open)
2. Volume filters (Absolute Volume, Absolute Dollar Volume)
3. Range filters (Range, Percentage Range)
4. Basic technical indicators (RSI, VWAP distance)

### Phase 2: Advanced Metrics (Medium Priority)
1. Relative volume calculations
2. Time-based filters (Minutes in Market)
3. Moving average distances
4. Position in range

### Phase 3: External Data Integration (Lower Priority)
1. News integration (Minutes/Hours/Days since news)
2. Earnings calendar integration
3. Fundamental data integration (MarketCap, Float, etc.)

## Common Implementation Patterns

### 1. Timeframe-Based Metrics
All metrics that support timeframes follow a similar pattern:
- Store historical data in `SymbolState.LastFinalBars` (ring buffer)
- Compute metric over specified timeframe
- Add metric with timeframe suffix: `{metric}_{timeframe}`

### 2. Session-Based Filtering
Filters that depend on market session:
- Add session detection logic (Pre-Market: 4:00-9:30, Market: 9:30-16:00, Post-Market: 16:00-20:00 ET)
- Check session in scan loop before evaluating filter
- Store session-specific data in `SymbolState`

### 3. Value Type Support ($ or %)
Filters that support both absolute and percentage values:
- Compute both values
- Add two metrics: `{metric}` (absolute) and `{metric}_pct` (percentage)
- Rule conditions can reference either metric

### 4. Volume Threshold Enforcement
All filters support volume threshold:
- Check volume threshold before evaluating filter
- Skip filter evaluation if volume < threshold
- Volume threshold is a pre-filter, not part of the metric calculation

## Data Storage Requirements

### In-Memory (SymbolState)
- Current price, volume, live bar
- Recent finalized bars (ring buffer)
- Indicators (RSI, EMA, SMA, VWAP)
- Session-specific data (premarket/postmarket volume)

### Historical Data (TimescaleDB)
- Daily bars (for multi-day calculations)
- Historical volumes (for average volume)
- Historical ranges (for biggest range)

### External Data Sources
- News data (last news timestamp per symbol)
- Earnings calendar (next earnings date per symbol)
- Fundamental data (MarketCap, Float, Shares Outstanding, etc.)
- Short interest data

## Testing Strategy

1. **Unit Tests**: Test each metric calculation independently
2. **Integration Tests**: Test metric computation in scan loop
3. **Timeframe Tests**: Test all supported timeframes
4. **Session Tests**: Test Pre-Market, Market, Post-Market behavior
5. **Edge Cases**: Test with missing data, zero values, etc.

## Performance Considerations

1. **Caching**: Cache computed metrics when possible
2. **Lazy Computation**: Only compute metrics when needed for rule evaluation
3. **Batch Processing**: Compute multiple metrics in single pass
4. **Historical Data**: Limit historical data storage to necessary periods
5. **External API Calls**: Cache external data (fundamental, news) aggressively

