# Alert Types Implementation Guide

This document provides a comprehensive overview of all alert types that the stock scanner application needs to support, along with detailed configuration options and implementation steps for each alert.

## Alert Configuration Structure

Each alert type supports the following configuration options:

1. **Volume Threshold**: Minimum volume required for the alert to be evaluated (e.g., 400,000, 1,000,000)
2. **Calculated During**: When the alert is active (Pre-Market, Market, Post-Market)
3. **Timeframes**: For time-based alerts (e.g., 1 Min, 2 Min, 5 Min, 15 Min, 30 Min, 60 Min, 4 Hour, 1 Day)
4. **Value Types**: For alerts that can be expressed in different units ($ or %)
5. **Direction**: For alerts that can monitor high or low (e.g., High/Low of the day)
6. **Multiplier**: For alerts that compare current vs average (e.g., Volume spike: 2x, 10x average)

---

## Alert Categories

### 1. Candlestick Pattern Alerts

#### 1.1 Lower Shadow Alert
- **Description**: Triggered when a candle of a given timeframe builds a long lower tail (lower shadow) compared to the body of the candle. The proportion of the tail to the body can be specified.
- **Volume Threshold**: Configurable (default: 400,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Timeframes**: 1 Min, 2 Min, 5 Min, 15 Min
- **Configuration**: Proportion threshold (e.g., shadow/body ratio > 2.0)
- **Implementation Steps**:
  1. Track candle body size: `body = abs(close - open)`
  2. Track lower shadow size: `lower_shadow = min(open, close) - low`
  3. Calculate ratio: `shadow_ratio = lower_shadow / body` (if body > 0)
  4. Add metric: `lower_shadow_ratio_{timeframe}` (e.g., `lower_shadow_ratio_5m`)
  5. Rule condition: `lower_shadow_ratio_5m > 2.0` (shadow is 2x the body)
  6. Support timeframe selection in rule configuration
  7. Check volume threshold before evaluation
  8. Support session-based filtering

#### 1.2 Upper Shadow Alert
- **Description**: Triggered when a candle of a given timeframe builds a long upper tail (upper shadow) compared to the body of the candle. The proportion of the tail to the body can be specified.
- **Volume Threshold**: Configurable (default: 400,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Timeframes**: 1 Min, 2 Min, 5 Min, 15 Min
- **Configuration**: Proportion threshold (e.g., shadow/body ratio > 2.0)
- **Implementation Steps**:
  1. Track candle body size: `body = abs(close - open)`
  2. Track upper shadow size: `upper_shadow = high - max(open, close)`
  3. Calculate ratio: `shadow_ratio = upper_shadow / body` (if body > 0)
  4. Add metric: `upper_shadow_ratio_{timeframe}`
  5. Rule condition: `upper_shadow_ratio_5m > 2.0`
  6. Support timeframe selection
  7. Check volume threshold before evaluation

#### 1.3 Bullish Candle Close
- **Description**: Always triggered when a candle closes green (close > open). Supports different timeframes.
- **Volume Threshold**: Configurable (default: 10,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Timeframes**: 1 Min, 2 Min, 5 Min, 15 Min
- **Implementation Steps**:
  1. Check if candle is bullish: `close > open`
  2. Add metric: `is_bullish_candle_{timeframe}` (boolean: 1.0 = true, 0.0 = false)
  3. Rule condition: `is_bullish_candle_5m == 1.0`
  4. Evaluate on bar finalization
  5. Support timeframe selection

#### 1.4 Bearish Candle Close
- **Description**: Always triggered when a candle closes red (close < open). Supports different timeframes.
- **Volume Threshold**: Configurable (default: 75,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Timeframes**: 1 Min, 2 Min, 5 Min, 15 Min
- **Implementation Steps**:
  1. Check if candle is bearish: `close < open`
  2. Add metric: `is_bearish_candle_{timeframe}` (boolean: 1.0 = true, 0.0 = false)
  3. Rule condition: `is_bearish_candle_5m == 1.0`
  4. Evaluate on bar finalization
  5. Support timeframe selection

#### 1.5 Bullish Engulfing Candle
- **Description**: A reversal pattern triggered when one red candle body is followed by a bigger green candle body that completely engulfs the previous candle.
- **Volume Threshold**: Configurable (default: 400,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Timeframes**: 1 Min, 2 Min, 5 Min, 15 Min, 30 Min, 60 Min
- **Implementation Steps**:
  1. Track previous candle: `prev_body = abs(prev_close - prev_open)`
  2. Track current candle: `curr_body = abs(close - open)`
  3. Check conditions:
     - Previous candle is bearish: `prev_close < prev_open`
     - Current candle is bullish: `close > open`
     - Current body engulfs previous: `curr_body > prev_body`
     - Current open < prev_close AND current close > prev_open
  4. Add metric: `bullish_engulfing_{timeframe}` (boolean)
  5. Rule condition: `bullish_engulfing_5m == 1.0`
  6. Evaluate on bar finalization

#### 1.6 Bearish Engulfing Candle
- **Description**: A reversal pattern triggered when one green candle body is followed by a bigger red candle body that completely engulfs the previous candle.
- **Volume Threshold**: Configurable (default: 400,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Timeframes**: 1 Min, 2 Min, 5 Min, 15 Min, 30 Min, 60 Min
- **Implementation Steps**:
  1. Track previous candle: `prev_body = abs(prev_close - prev_open)`
  2. Track current candle: `curr_body = abs(close - open)`
  3. Check conditions:
     - Previous candle is bullish: `prev_close > prev_open`
     - Current candle is bearish: `close < open`
     - Current body engulfs previous: `curr_body > prev_body`
     - Current open > prev_close AND current close < prev_open
  4. Add metric: `bearish_engulfing_{timeframe}` (boolean)
  5. Rule condition: `bearish_engulfing_5m == 1.0`
  6. Evaluate on bar finalization

#### 1.7 Bullish Harami Candle
- **Description**: A reversal pattern triggered when one red candle body is followed by a smaller green candle body that is entirely contained within the previous candle's body.
- **Volume Threshold**: Configurable (default: 400,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Timeframes**: 1 Min, 2 Min, 5 Min, 15 Min
- **Implementation Steps**:
  1. Check conditions:
     - Previous candle is bearish: `prev_close < prev_open`
     - Current candle is bullish: `close > open`
     - Current body is smaller: `abs(close - open) < abs(prev_close - prev_open)`
     - Current body is inside previous: `prev_open > current_open AND prev_close < current_close` (for bearish prev)
  2. Add metric: `bullish_harami_{timeframe}` (boolean)
  3. Rule condition: `bullish_harami_5m == 1.0`
  4. Evaluate on bar finalization

#### 1.8 Bearish Harami Candle
- **Description**: A reversal pattern triggered when one green candle body is followed by a smaller red candle body that is entirely contained within the previous candle's body.
- **Volume Threshold**: Configurable (default: 400,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Timeframes**: 1 Min, 2 Min, 5 Min, 15 Min
- **Implementation Steps**:
  1. Check conditions:
     - Previous candle is bullish: `prev_close > prev_open`
     - Current candle is bearish: `close < open`
     - Current body is smaller: `abs(close - open) < abs(prev_close - prev_open)`
     - Current body is inside previous: `prev_open < current_open AND prev_close > current_close` (for bullish prev)
  2. Add metric: `bearish_harami_{timeframe}` (boolean)
  3. Rule condition: `bearish_harami_5m == 1.0`
  4. Evaluate on bar finalization

#### 1.9 Inside Bar
- **Description**: Triggered when the current candle is smaller (from low to high) and entirely inside the previous candle's range.
- **Volume Threshold**: Configurable (default: 75,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Timeframes**: 5 Min, 15 Min, 30 Min, 60 Min, 4 Hour, 1 Day
- **Implementation Steps**:
  1. Check conditions:
     - Current high < previous high
     - Current low > previous low
     - Current range is smaller: `(current_high - current_low) < (prev_high - prev_low)`
  2. Add metric: `inside_bar_{timeframe}` (boolean)
  3. Rule condition: `inside_bar_15m == 1.0`
  4. Evaluate on bar finalization

---

### 2. Price Level Alerts

#### 2.1 Near High/Low of the Day
- **Description**: Triggered when the stock is near the high or low of the day, allowing anticipation of breakouts. User can choose between high or low.
- **Volume Threshold**: Configurable (default: 25,000)
- **Calculated During**: Market
- **Direction**: High or Low
- **Configuration**: Proximity threshold (e.g., within 0.5% of high/low)
- **Implementation Steps**:
  1. Track day's high and low from market open (9:30 AM ET)
  2. Calculate distance to high: `dist_to_high_pct = ((high_of_day - current_price) / high_of_day) * 100`
  3. Calculate distance to low: `dist_to_low_pct = ((current_price - low_of_day) / low_of_day) * 100`
  4. Add metrics: `dist_to_high_of_day_pct`, `dist_to_low_of_day_pct`
  5. Rule condition for high: `dist_to_high_of_day_pct < 0.5` (within 0.5% of high)
  6. Rule condition for low: `dist_to_low_of_day_pct < 0.5` (within 0.5% of low)
  7. Reset high/low at market open each day
  8. Support direction selection (High/Low) in rule configuration

#### 2.2 High/Low of the Day (Pre/Post-Market)
- **Description**: Similar to above but includes extended hours (pre-market and post-market) in the high/low calculation.
- **Volume Threshold**: Configurable (default: 1,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Direction**: High or Low
- **Configuration**: Proximity threshold
- **Implementation Steps**:
  1. Track high/low from pre-market start (4:00 AM ET) through post-market end (8:00 PM ET)
  2. Use same metrics as above but include extended hours data
  3. Add metrics: `dist_to_high_of_day_extended_pct`, `dist_to_low_of_day_extended_pct`
  4. Support direction selection
  5. Reset at pre-market start each day

#### 2.3 Near Last High
- **Description**: Triggered when the price is near the latest high that the candlestick chart has formed for a given timeframe.
- **Volume Threshold**: Configurable (default: 400,000)
- **Calculated During**: Market
- **Timeframes**: 1 Min, 2 Min, 5 Min, 15 Min
- **Configuration**: Proximity threshold (e.g., within 0.1% of recent high)
- **Implementation Steps**:
  1. Track recent high over specified timeframe (e.g., last 20 candles for 5m timeframe)
  2. Calculate distance: `dist_to_recent_high_pct = ((recent_high - current_price) / recent_high) * 100`
  3. Add metrics: `dist_to_recent_high_{timeframe}_pct`
  4. Rule condition: `dist_to_recent_high_5m_pct < 0.1`
  5. Update recent high when new high is formed
  6. Support timeframe selection

#### 2.4 Near Last Low
- **Description**: Triggered when the price is near the latest low that the candlestick chart has formed for a given timeframe.
- **Volume Threshold**: Configurable (default: 400,000)
- **Calculated During**: Market
- **Timeframes**: 1 Min, 2 Min, 5 Min, 15 Min
- **Configuration**: Proximity threshold
- **Implementation Steps**:
  1. Track recent low over specified timeframe
  2. Calculate distance: `dist_to_recent_low_pct = ((current_price - recent_low) / recent_low) * 100`
  3. Add metrics: `dist_to_recent_low_{timeframe}_pct`
  4. Rule condition: `dist_to_recent_low_5m_pct < 0.1`
  5. Update recent low when new low is formed
  6. Support timeframe selection

#### 2.5 Break Over Recent High
- **Description**: Triggered when the price breaks over the latest high that the candlestick chart has formed for a given timeframe.
- **Volume Threshold**: Configurable (default: 150,000)
- **Calculated During**: Market
- **Timeframes**: 1 Min, 2 Min, 5 Min, 15 Min, 4 Hour
- **Implementation Steps**:
  1. Track recent high over specified timeframe
  2. Check if current price > recent high
  3. Add metric: `broke_recent_high_{timeframe}` (boolean)
  4. Rule condition: `broke_recent_high_5m == 1.0`
  5. Update recent high after break
  6. Support timeframe selection

#### 2.6 Break Under Recent Low
- **Description**: Triggered when the price breaks under the latest low that the candlestick chart has formed for a given timeframe.
- **Volume Threshold**: Configurable (default: 400,000)
- **Calculated During**: Market
- **Timeframes**: 1 Min, 2 Min, 5 Min, 15 Min
- **Implementation Steps**:
  1. Track recent low over specified timeframe
  2. Check if current price < recent low
  3. Add metric: `broke_recent_low_{timeframe}` (boolean)
  4. Rule condition: `broke_recent_low_5m == 1.0`
  5. Update recent low after break
  6. Support timeframe selection

#### 2.7 Reject Last High
- **Description**: Triggered when the price rejects (bounces off) the latest high that the candlestick chart has formed for a given timeframe.
- **Volume Threshold**: Configurable (default: 1,000,000)
- **Calculated During**: Market
- **Timeframes**: 1 Min, 2 Min, 5 Min, 15 Min, 30 Min, 60 Min
- **Configuration**: Rejection threshold (e.g., price reached within 0.1% of high then dropped by 0.5%)
- **Implementation Steps**:
  1. Track recent high
  2. Detect when price approaches high (within threshold)
  3. Detect subsequent rejection (price drops by specified amount)
  4. Add metric: `rejected_recent_high_{timeframe}` (boolean)
  5. Rule condition: `rejected_recent_high_5m == 1.0`
  6. Support timeframe selection

#### 2.8 Reject Last Low
- **Description**: Triggered when the price rejects (bounces off) the latest low that the candlestick chart has formed for a given timeframe.
- **Volume Threshold**: Configurable (default: 1,000,000)
- **Calculated During**: Market
- **Timeframes**: 1 Min, 2 Min, 5 Min, 15 Min
- **Configuration**: Rejection threshold
- **Implementation Steps**:
  1. Track recent low
  2. Detect when price approaches low (within threshold)
  3. Detect subsequent rejection (price rises by specified amount)
  4. Add metric: `rejected_recent_low_{timeframe}` (boolean)
  5. Rule condition: `rejected_recent_low_5m == 1.0`
  6. Support timeframe selection

#### 2.9 New Candle High
- **Description**: Triggered when the current candle makes a new high compared to the previous candle for a given timeframe.
- **Volume Threshold**: Configurable (default: 10,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Timeframes**: 1 Min, 2 Min, 5 Min, 15 Min, 30 Min, 60 Min, 4 Hour, 1 Day
- **Implementation Steps**:
  1. Compare current candle high to previous candle high
  2. Check if current high > previous high
  3. Add metric: `new_candle_high_{timeframe}` (boolean)
  4. Rule condition: `new_candle_high_5m == 1.0`
  5. Evaluate on bar finalization
  6. Support timeframe selection

#### 2.10 New Candle Low
- **Description**: Triggered when the current candle makes a new low compared to the previous candle for a given timeframe.
- **Volume Threshold**: Configurable (default: 10,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Timeframes**: 1 Min, 2 Min, 5 Min, 15 Min, 30 Min, 60 Min, 4 Hour, 1 Day
- **Implementation Steps**:
  1. Compare current candle low to previous candle low
  2. Check if current low < previous low
  3. Add metric: `new_candle_low_{timeframe}` (boolean)
  4. Rule condition: `new_candle_low_5m == 1.0`
  5. Evaluate on bar finalization
  6. Support timeframe selection

---

### 3. VWAP Alerts

#### 3.1 Through VWAP Alert
- **Description**: Triggered when the price rushes through the VWAP with a candle 3 times bigger than the last candles.
- **Volume Threshold**: Configurable (default: 1,000,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Direction**: Above or Below
- **Configuration**: Candle size multiplier (default: 3x)
- **Implementation Steps**:
  1. Track average candle size over last N candles (e.g., 5 candles)
  2. Calculate current candle size: `candle_size = high - low`
  3. Calculate average candle size: `avg_candle_size = sum(last_N_candle_sizes) / N`
  4. Check if current candle is 3x average: `candle_size >= 3 * avg_candle_size`
  5. Check if price crossed VWAP (above or below based on direction)
  6. Add metric: `through_vwap_{direction}` (boolean, direction: above/below)
  7. Rule condition: `through_vwap_above == 1.0`
  8. Support direction selection

#### 3.2 VWAP Acts as Support
- **Description**: Triggered when the VWAP of a stock acts as support (price comes from above and bounces off VWAP). Based on 1 minute candles.
- **Volume Threshold**: Configurable (default: 1,000,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Timeframes**: 1 Min, 2 Min, 5 Min, 15 Min
- **Configuration**: Bounce threshold (e.g., price touched VWAP then rose by 0.2%)
- **Implementation Steps**:
  1. Track price approaching VWAP from above
  2. Detect when price touches or gets very close to VWAP (within 0.1%)
  3. Detect subsequent bounce (price rises by threshold amount)
  4. Add metric: `vwap_support_{timeframe}` (boolean)
  5. Rule condition: `vwap_support_1m == 1.0`
  6. Support timeframe selection

#### 3.3 VWAP Acts as Resistance
- **Description**: Triggered when the VWAP of a stock acts as resistance (price comes from below and bounces off VWAP). Based on 1 minute candles.
- **Volume Threshold**: Configurable (default: 1,000,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Timeframes**: 1 Min, 2 Min, 5 Min, 15 Min
- **Configuration**: Bounce threshold
- **Implementation Steps**:
  1. Track price approaching VWAP from below
  2. Detect when price touches or gets very close to VWAP (within 0.1%)
  3. Detect subsequent rejection (price drops by threshold amount)
  4. Add metric: `vwap_resistance_{timeframe}` (boolean)
  5. Rule condition: `vwap_resistance_1m == 1.0`
  6. Support timeframe selection

---

### 4. Moving Average Alerts

#### 4.1 Back to EMA Alert
- **Description**: Triggered when the price has some distance to EMA for a while and then comes back to the EMA. Supports different EMAs and timeframes.
- **Volume Threshold**: Configurable (default: 1,000,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **EMA Options**: EMA(9) 1 min, EMA(20) 1 min, EMA(200) 1 min, EMA(9) 5 min, EMA(20) 5 min, EMA(9) 15 min, EMA(21) 15 min
- **Configuration**: Distance threshold (e.g., price was >2% away, now within 0.5%)
- **Implementation Steps**:
  1. Track price distance from EMA over time
  2. Detect when price was far from EMA (distance > threshold)
  3. Detect when price returns to EMA (distance < threshold)
  4. Add metric: `back_to_ema_{ema_type}_{timeframe}` (boolean)
  5. Rule condition: `back_to_ema_ema9_5m == 1.0`
  6. Support EMA type and timeframe selection

#### 4.2 Crossing Above
- **Description**: Triggered if the price of a stock crosses above one of: today's open, yesterday's close, EMA(20), or the current VWAP.
- **Volume Threshold**: Configurable (default: 400,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Crossing Options**: Open, Close, VWAP, EMA(20) 2 min, EMA(9) 5 min, EMA(9) 15 min, EMA(21) 15 min, EMA(9) 60 min, EMA(21) 60 min, EMA(9) daily, EMA(21) daily, EMA(50) daily, SMA(200) daily
- **Implementation Steps**:
  1. Track previous price and current price
  2. Get crossing level value (open, close, VWAP, or EMA/SMA)
  3. Check if previous price < level AND current price > level
  4. Add metric: `crossed_above_{level_type}` (boolean)
  5. Rule condition: `crossed_above_vwap == 1.0` or `crossed_above_ema20_2m == 1.0`
  6. Support level selection in rule configuration

#### 4.3 Crossing Below
- **Description**: Triggered if the price of a stock crosses below one of: today's open, yesterday's close, EMA(20), or the current VWAP.
- **Volume Threshold**: Configurable (default: 400,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Crossing Options**: Same as Crossing Above
- **Implementation Steps**:
  1. Track previous price and current price
  2. Get crossing level value
  3. Check if previous price > level AND current price < level
  4. Add metric: `crossed_below_{level_type}` (boolean)
  5. Rule condition: `crossed_below_vwap == 1.0`
  6. Support level selection

---

### 5. Volume Alerts

#### 5.1 Volume Spike (2)
- **Description**: Triggered when there is a sudden spike in volume. The current volume for a candle is compared to the average volume of the last 2 candles. The multiplier can be specified (e.g., 2 means at least double the average).
- **Volume Threshold**: Configurable (default: 2,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Timeframes**: 1 Min, 2 Min, 5 Min, 15 Min
- **Configuration**: Multiplier (e.g., 1, 1.5, 2, 2.5, 3, etc.)
- **Implementation Steps**:
  1. Calculate average volume of last 2 candles: `avg_vol = (vol[-2] + vol[-1]) / 2`
  2. Get current candle volume
  3. Calculate ratio: `volume_ratio = current_vol / avg_vol`
  4. Check if ratio >= multiplier: `volume_ratio >= 2.0` (for 2x multiplier)
  5. Add metric: `volume_spike_2_{timeframe}` (ratio value)
  6. Rule condition: `volume_spike_2_5m >= 2.0`
  7. Support timeframe and multiplier selection

#### 5.2 Volume Spike (10)
- **Description**: Similar to Volume Spike (2) but compares current volume to the average volume of the last 10 candles.
- **Volume Threshold**: Configurable (default: 10,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Timeframes**: 1 Min, 2 Min, 5 Min, 15 Min
- **Configuration**: Multiplier
- **Implementation Steps**:
  1. Calculate average volume of last 10 candles
  2. Calculate ratio: `volume_ratio = current_vol / avg_vol`
  3. Add metric: `volume_spike_10_{timeframe}` (ratio value)
  4. Rule condition: `volume_spike_10_5m >= 2.0`
  5. Support timeframe and multiplier selection

---

### 6. Price Movement Alerts

#### 6.1 Running Up
- **Description**: Triggered when a stock is running up in the last 60 seconds. Can specify 0.1 steps either as absolute ($) or relative (%) value (min 0.5, e.g., stock has to move 1.5% up within 60s to trigger).
- **Volume Threshold**: Configurable (default: 1,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Value Types**: ($) or (%)
- **Configuration**: Movement threshold (e.g., 1.5% or $0.50)
- **Implementation Steps**:
  1. Track price 60 seconds ago
  2. Calculate change: `change = current_price - price_60s_ago`
  3. Calculate percentage change: `change_pct = (change / price_60s_ago) * 100`
  4. For absolute: Check if `change >= threshold` (e.g., $0.50)
  5. For percentage: Check if `change_pct >= threshold` (e.g., 1.5%)
  6. Add metrics: `running_up_60s` (absolute), `running_up_60s_pct` (percentage)
  7. Rule condition: `running_up_60s_pct >= 1.5`
  8. Support value type selection

#### 6.2 Running Down
- **Description**: Triggered when a stock is running down in the last 60 seconds. Can specify 0.1 steps either as absolute ($) or relative (%) value (min 0.5, e.g., stock has to move 2.3% down within 60s to trigger).
- **Volume Threshold**: Configurable (default: 10,000)
- **Calculated During**: Pre-Market, Market, Post-Market
- **Value Types**: ($) or (%)
- **Configuration**: Movement threshold (e.g., 2.3% or $1.00)
- **Implementation Steps**:
  1. Track price 60 seconds ago
  2. Calculate change: `change = current_price - price_60s_ago`
  3. Calculate percentage change: `change_pct = (change / price_60s_ago) * 100`
  4. For absolute: Check if `change <= -threshold` (negative change)
  5. For percentage: Check if `change_pct <= -threshold` (e.g., -2.3%)
  6. Add metrics: `running_down_60s` (absolute, negative), `running_down_60s_pct` (percentage, negative)
  7. Rule condition: `running_down_60s_pct <= -2.3`
  8. Support value type selection

---

### 7. Opening Range Alerts

#### 7.1 Opening Range Breakout
- **Description**: The opening range is defined by the first candle of a stock after the market opened. This alert is triggered as soon as the first candle breaks over this range for a given timeframe.
- **Volume Threshold**: Configurable (default: 75,000)
- **Calculated During**: Market
- **Timeframes**: 1 Min, 2 Min, 5 Min, 15 Min, 30 Min, 60 Min
- **Implementation Steps**:
  1. Identify first candle after market open (9:30 AM ET)
  2. Store opening range: `range_high = first_candle_high`, `range_low = first_candle_low`
  3. Monitor subsequent candles
  4. Check if current candle breaks above range: `current_high > range_high`
  5. Add metric: `opening_range_breakout_{timeframe}` (boolean)
  6. Rule condition: `opening_range_breakout_5m == 1.0`
  7. Reset opening range at market open each day
  8. Support timeframe selection

#### 7.2 Opening Range Breakdown
- **Description**: The opening range is defined by the first candle of a stock after the market opened. This alert is triggered as soon as the first candle breaks under this range for a given timeframe.
- **Volume Threshold**: Configurable (default: 75,000)
- **Calculated During**: Market
- **Timeframes**: 1 Min, 2 Min, 5 Min, 15 Min, 30 Min, 60 Min
- **Implementation Steps**:
  1. Identify first candle after market open
  2. Store opening range
  3. Check if current candle breaks below range: `current_low < range_low`
  4. Add metric: `opening_range_breakdown_{timeframe}` (boolean)
  5. Rule condition: `opening_range_breakdown_5m == 1.0`
  6. Reset opening range at market open each day
  7. Support timeframe selection

---

## Common Implementation Patterns

### 1. Pattern Detection
All candlestick pattern alerts follow a similar pattern:
- Track previous and current candles from finalized bars
- Compare candle properties (body size, shadows, direction)
- Generate boolean metric when pattern is detected
- Evaluate on bar finalization

### 2. Level Crossing Detection
For crossing alerts (VWAP, EMA, etc.):
- Track previous price and current price
- Compare against level value
- Detect crossing direction (above/below)
- Generate boolean metric on crossing

### 3. Range Tracking
For high/low tracking alerts:
- Maintain rolling window of recent highs/lows
- Update when new high/low is formed
- Calculate distance metrics
- Support timeframe-based windows

### 4. Volume Comparison
For volume spike alerts:
- Calculate average volume over specified window
- Compare current volume to average
- Support configurable multiplier thresholds

### 5. Session-Based Filtering
All alerts support session filtering:
- Pre-Market: 4:00 AM - 9:30 AM ET
- Market: 9:30 AM - 4:00 PM ET
- Post-Market: 4:00 PM - 8:00 PM ET
- Check session before evaluating alert

### 6. Volume Threshold Enforcement
All alerts support volume threshold:
- Check volume threshold before evaluating alert
- Skip evaluation if volume < threshold
- Volume threshold is a pre-filter, not part of metric calculation

---

## Data Storage Requirements

### In-Memory (SymbolState)
- Current price, volume, live bar
- Recent finalized bars (ring buffer)
- Indicators (RSI, EMA, SMA, VWAP)
- Recent highs/lows per timeframe
- Opening range (high/low)
- Previous candle data for pattern detection
- Price history for 60-second movement tracking

### Historical Data (TimescaleDB)
- Daily bars (for opening range, high/low of day)
- Historical volumes (for volume spike calculations)
- Historical highs/lows (for recent high/low tracking)

---

## Implementation Priority

### Phase 1: Core Pattern Alerts (High Priority)
1. Bullish/Bearish Candle Close
2. Bullish/Bearish Engulfing
3. Bullish/Bearish Harami
4. Inside Bar
5. Lower/Upper Shadow

### Phase 2: Price Level Alerts (High Priority)
1. Near High/Low of the Day
2. Break Over/Under Recent High/Low
3. New Candle High/Low
4. Near Last High/Low

### Phase 3: VWAP & Moving Average Alerts (Medium Priority)
1. Through VWAP
2. VWAP Support/Resistance
3. Crossing Above/Below
4. Back to EMA

### Phase 4: Volume & Movement Alerts (Medium Priority)
1. Volume Spike (2) and (10)
2. Running Up/Down

### Phase 5: Opening Range Alerts (Lower Priority)
1. Opening Range Breakout
2. Opening Range Breakdown
3. Reject Last High/Low

---

## Testing Strategy

1. **Unit Tests**: Test each pattern detection algorithm independently
2. **Integration Tests**: Test pattern detection in scan loop with real bar data
3. **Timeframe Tests**: Test all supported timeframes for each alert
4. **Session Tests**: Test Pre-Market, Market, Post-Market behavior
5. **Edge Cases**: Test with missing data, single candles, etc.
6. **Performance Tests**: Ensure pattern detection doesn't slow down scan loop

---

## Performance Considerations

1. **Pattern Detection**: Cache previous candle data to avoid repeated lookups
2. **High/Low Tracking**: Use efficient data structures (ring buffers, sliding windows)
3. **Volume Calculations**: Pre-compute averages where possible
4. **Session Detection**: Cache session state, only recalculate on minute boundaries
5. **Lazy Evaluation**: Only compute metrics when needed for rule evaluation
6. **Batch Processing**: Compute multiple pattern metrics in single pass over bars

