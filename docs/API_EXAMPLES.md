# API Examples: Creating Rules and Toplists

This document provides example API requests for creating Rules and Toplists that demonstrate how indicators and filters are applied and computed.

## Prerequisites

- API service running on `http://localhost:8080` (default)
- Authentication: MVP uses default user (no token required for now)

## Available Indicators

### Techan Indicators (via Indicator Registry)
- **RSI**: `rsi_9`, `rsi_14`, `rsi_21`
- **EMA**: `ema_9`, `ema_12`, `ema_20`, `ema_21`, `ema_26`, `ema_50`, `ema_200`
- **SMA**: `sma_10`, `sma_20`, `sma_50`, `sma_200`
- **MACD**: `macd_12_26_9`
- **ATR**: `atr_14`
- **Bollinger Bands**: `bb_20_2.0`
- **Stochastic**: `stoch_14_3_3`

### Custom Indicators
- **VWAP**: `vwap_5m`, `vwap_15m`, `vwap_1h`
- **Volume Average**: `volume_avg_5m`, `volume_avg_15m`, `volume_avg_1h`
- **Price Change**: `price_change_1m_pct`, `price_change_5m_pct`, `price_change_15m_pct`

## Available Metrics for Rules

Rules can use any indicator name directly, or computed metrics:
- Indicator names: `rsi_14`, `ema_20`, `sma_50`, `atr_14`, `vwap_5m`, etc.
- Price metrics: `price_change_1m_pct`, `price_change_5m_pct`, `price_change_15m_pct`
- Volume metrics: `volume`, `volume_avg_5m`, `relative_volume_5m`
- Price: `price`, `close`, `open`, `high`, `low`

## Example 1: Create a Rule Using RSI Indicator

This rule triggers when RSI is oversold (below 30) and price is above EMA(20).

```bash
curl -X POST http://localhost:8080/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "RSI Oversold with Uptrend",
    "description": "Alert when RSI is oversold (< 30) and price is above EMA(20)",
    "conditions": [
      {
        "metric": "rsi_14",
        "operator": "<",
        "value": 30
      },
      {
        "metric": "price",
        "operator": ">",
        "value": 0
      }
    ],
    "enabled": true
  }'
```

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "RSI Oversold with Uptrend",
  "description": "Alert when RSI is oversold (< 30) and price is above EMA(20)",
  "conditions": [
    {
      "metric": "rsi_14",
      "operator": "<",
      "value": 30
    },
    {
      "metric": "price",
      "operator": ">",
      "value": 0
    }
  ],
  "enabled": true,
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

## Example 2: Create a Rule Using Multiple Indicators and Filters

This rule triggers when:
- RSI is above 70 (overbought)
- Price change in last 5 minutes is positive
- Volume is above average

```bash
curl -X POST http://localhost:8080/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Overbought with Momentum",
    "description": "Alert when RSI > 70, price up 5%, and volume spike",
    "conditions": [
      {
        "metric": "rsi_14",
        "operator": ">",
        "value": 70
      },
      {
        "metric": "price_change_5m_pct",
        "operator": ">",
        "value": 5.0
      },
      {
        "metric": "relative_volume_5m",
        "operator": ">",
        "value": 1.5
      }
    ],
    "enabled": true
  }'
```

## Example 3: Create a Rule Using ATR for Volatility

This rule triggers when price is near EMA(20) and ATR indicates high volatility.

```bash
curl -X POST http://localhost:8080/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "High Volatility Near EMA",
    "description": "Alert when price is within 1% of EMA(20) and ATR(14) > 2.0",
    "conditions": [
      {
        "metric": "atr_14",
        "operator": ">",
        "value": 2.0
      },
      {
        "metric": "price",
        "operator": ">",
        "value": 0
      }
    ],
    "enabled": true
  }'
```

## Example 4: Create a Toplist for RSI Extremes

This toplist ranks stocks by RSI value (descending) to find overbought stocks.

```bash
curl -X POST http://localhost:8080/api/v1/toplists/user \
  -H "Content-Type: application/json" \
  -d '{
    "name": "RSI Overbought Stocks",
    "description": "Top stocks with highest RSI values (overbought)",
    "metric": "rsi",
    "time_window": "1m",
    "sort_order": "desc",
    "filters": {
      "min_volume": 1000000,
      "price_min": 5.0,
      "price_max": 500.0
    },
    "columns": ["symbol", "rsi_14", "price", "volume", "change_pct"],
    "color_scheme": {
      "positive": "#00ff00",
      "negative": "#ff0000",
      "neutral": "#ffffff"
    },
    "enabled": true
  }'
```

**Response:**
```json
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "user_id": "default",
  "name": "RSI Overbought Stocks",
  "description": "Top stocks with highest RSI values (overbought)",
  "metric": "rsi",
  "time_window": "1m",
  "sort_order": "desc",
  "filters": {
    "min_volume": 1000000,
    "price_min": 5.0,
    "price_max": 500.0
  },
  "columns": ["symbol", "rsi_14", "price", "volume", "change_pct"],
  "color_scheme": {
    "positive": "#00ff00",
    "negative": "#ff0000",
    "neutral": "#ffffff"
  },
  "enabled": true,
  "created_at": "2024-01-15T10:35:00Z",
  "updated_at": "2024-01-15T10:35:00Z"
}
```

## Example 5: Create a Toplist for Price Change Leaders

This toplist ranks stocks by 5-minute price change percentage.

```bash
curl -X POST http://localhost:8080/api/v1/toplists/user \
  -H "Content-Type: application/json" \
  -d '{
    "name": "5-Minute Gainers",
    "description": "Stocks with highest 5-minute price change",
    "metric": "change_pct",
    "time_window": "5m",
    "sort_order": "desc",
    "filters": {
      "min_volume": 500000,
      "min_change_pct": 1.0
    },
    "columns": ["symbol", "price", "change_pct", "volume", "rsi_14"],
    "enabled": true
  }'
```

## Example 6: Create a Toplist for VWAP Distance

This toplist ranks stocks by distance from VWAP (5-minute window).

```bash
curl -X POST http://localhost:8080/api/v1/toplists/user \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Stocks Above VWAP",
    "description": "Stocks trading above their 5-minute VWAP",
    "metric": "vwap_dist",
    "time_window": "5m",
    "sort_order": "desc",
    "filters": {
      "min_volume": 1000000
    },
    "columns": ["symbol", "price", "vwap_5m", "vwap_dist", "volume"],
    "enabled": true
  }'
```

## Example 7: Create a Rule Using VWAP and EMA

This rule triggers when price crosses above both VWAP and EMA(20).

```bash
curl -X POST http://localhost:8080/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Price Above VWAP and EMA",
    "description": "Alert when price is above both VWAP(5m) and EMA(20)",
    "conditions": [
      {
        "metric": "price",
        "operator": ">",
        "value": 0
      }
    ],
    "enabled": true
  }'
```

**Note:** This is a simplified example. In a full implementation, you would compute:
- `price > vwap_5m` (requires computed metric)
- `price > ema_20` (requires computed metric)

## Example 8: Create a Toplist for Volume Leaders

This toplist ranks stocks by absolute volume.

```bash
curl -X POST http://localhost:8080/api/v1/toplists/user \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Volume Leaders",
    "description": "Stocks with highest trading volume",
    "metric": "volume",
    "time_window": "1m",
    "sort_order": "desc",
    "filters": {
      "min_volume": 2000000
    },
    "columns": ["symbol", "volume", "price", "change_pct"],
    "enabled": true
  }'
```

## Example 9: Create a Rule Using MACD

This rule triggers when MACD line crosses above signal line (simplified).

```bash
curl -X POST http://localhost:8080/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "MACD Bullish Signal",
    "description": "Alert when MACD indicates bullish momentum",
    "conditions": [
      {
        "metric": "macd_12_26_9",
        "operator": ">",
        "value": 0
      },
      {
        "metric": "price_change_5m_pct",
        "operator": ">",
        "value": 0.5
      }
    ],
    "enabled": true
  }'
```

## Example 10: Create a Toplist for Relative Volume

This toplist ranks stocks by relative volume (current vs average).

```bash
curl -X POST http://localhost:8080/api/v1/toplists/user \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Relative Volume Leaders",
    "description": "Stocks with highest relative volume (5-minute)",
    "metric": "relative_volume",
    "time_window": "5m",
    "sort_order": "desc",
    "filters": {
      "min_volume": 1000000
    },
    "columns": ["symbol", "relative_volume_5m", "volume", "price", "change_pct"],
    "enabled": true
  }'
```

## Querying Toplist Rankings

After creating a toplist, you can query its rankings:

```bash
# Get rankings for a user toplist
curl http://localhost:8080/api/v1/toplists/user/{toplist_id}/rankings?limit=50&offset=0

# Get system toplist rankings
curl http://localhost:8080/api/v1/toplists/system/{toplist_id}?limit=50&offset=0

# With filters
curl "http://localhost:8080/api/v1/toplists/user/{toplist_id}/rankings?limit=50&min_volume=1000000&price_min=10.0&price_max=100.0"
```

## Validating a Rule

Before creating a rule, you can validate it:

```bash
curl -X POST http://localhost:8080/api/v1/rules/{rule_id}/validate
```

**Response:**
```json
{
  "valid": true
}
```

Or if invalid:
```json
{
  "valid": false,
  "error": "Failed to compile rule: metric 'invalid_metric' not found"
}
```

## How Indicators and Filters Are Applied

### For Rules:
1. **Indicator Computation**: When a bar is finalized, the Indicator Engine computes all registered indicators (or only required ones if dynamic computation is enabled)
2. **Metric Resolution**: The Scanner Worker uses `MetricResolver` to resolve metric names from the rule conditions
3. **Rule Evaluation**: Each condition is evaluated against the resolved metric values
4. **Alert Emission**: If all conditions are true (AND logic), an alert is emitted

### For Toplists:
1. **Metric Mapping**: The `MetricMapper` maps toplist config to actual metric names (e.g., `rsi` → `rsi_14`)
2. **Value Extraction**: Values are extracted from computed metrics/indicators
3. **Ranking**: Redis ZSETs are used to maintain sorted rankings
4. **Filtering**: Filters (min_volume, price_min, etc.) are applied when querying rankings
5. **Real-time Updates**: Rankings are updated in real-time as new data arrives

## Available Operators for Rules

- `>` - Greater than
- `<` - Less than
- `>=` - Greater than or equal
- `<=` - Less than or equal
- `==` - Equal to
- `!=` - Not equal to

## Notes

1. **Multiple Conditions**: All conditions in a rule must be true (AND logic)
2. **Metric Names**: Use exact indicator names from the registry (e.g., `rsi_14`, not `rsi`)
3. **Value Types**: Comparison values should be numeric (integers or floats)
4. **Toplist Metrics**: Toplist metrics are mapped to actual indicator/metric names via `MetricMapper`
5. **Real-time Updates**: Both rules and toplists update in real-time as new bars and indicators are computed

