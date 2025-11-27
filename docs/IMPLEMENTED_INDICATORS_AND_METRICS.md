# Implemented Indicators and Metrics

This document lists all indicators and metrics currently implemented in the stock-scanner project.

## Indicators (Incremental/Streaming)

Indicators are computed incrementally as bars arrive. They maintain internal state and update with each new bar.

### Techan Indicators (via `github.com/sdcoffey/techan`)

#### Momentum Indicators

| Name | Period | Description | Min Bars |
|------|--------|-------------|----------|
| `rsi_9` | 9 | Relative Strength Index (9 period) | 10 |
| `rsi_14` | 14 | Relative Strength Index (14 period) | 15 |
| `rsi_21` | 21 | Relative Strength Index (21 period) | 22 |
| `stoch_14_3_3` | 14,3,3 | Stochastic Oscillator | 14 |

#### Trend Indicators

| Name | Period | Description | Min Bars |
|------|--------|-------------|----------|
| `ema_9` | 9 | Exponential Moving Average (9 period) | 9 |
| `ema_12` | 12 | Exponential Moving Average (12 period) | 12 |
| `ema_20` | 20 | Exponential Moving Average (20 period) | 20 |
| `ema_21` | 21 | Exponential Moving Average (21 period) | 21 |
| `ema_26` | 26 | Exponential Moving Average (26 period) | 26 |
| `ema_50` | 50 | Exponential Moving Average (50 period) | 50 |
| `ema_200` | 200 | Exponential Moving Average (200 period) | 200 |
| `sma_10` | 10 | Simple Moving Average (10 period) | 10 |
| `sma_20` | 20 | Simple Moving Average (20 period) | 20 |
| `sma_50` | 50 | Simple Moving Average (50 period) | 50 |
| `sma_200` | 200 | Simple Moving Average (200 period) | 200 |
| `macd_12_26_9` | 12,26,9 | MACD (Moving Average Convergence Divergence) | 26 |

#### Volatility Indicators

| Name | Period | Description | Min Bars |
|------|--------|-------------|----------|
| `atr_14` | 14 | Average True Range (14 period) | 14 |
| `bb_20_2.0` | 20, 2.0 | Bollinger Bands (20 period, 2.0 std dev) | 20 |

### Custom Indicators

#### Price Indicators

| Name | Window | Description | Min Bars |
|------|--------|-------------|----------|
| `vwap_5m` | 5 minutes | Volume Weighted Average Price (5m window) | 1 |
| `vwap_15m` | 15 minutes | Volume Weighted Average Price (15m window) | 1 |
| `vwap_1h` | 1 hour | Volume Weighted Average Price (1h window) | 1 |

#### Volume Indicators

| Name | Window | Description | Min Bars |
|------|--------|-------------|----------|
| `volume_avg_5m` | 5 minutes | Average Volume (5m window) | 1 |
| `volume_avg_15m` | 15 minutes | Average Volume (15m window) | 1 |
| `volume_avg_1h` | 1 hour | Average Volume (1h window) | 1 |

#### Price Change Indicators

| Name | Window | Description | Min Bars |
|------|--------|-------------|----------|
| `price_change_1m_pct` | 1 minute | Price Change Percentage (1m window) | 2 |
| `price_change_5m_pct` | 5 minutes | Price Change Percentage (5m window) | 2 |
| `price_change_15m_pct` | 15 minutes | Price Change Percentage (15m window) | 2 |

---

## Metrics (On-Demand Computation)

Metrics are computed on-demand from symbol state snapshots. They don't maintain internal state.

### Live Bar Metrics

Computed from the current live (unfinalized) bar:

| Name | Description | Source |
|------|-------------|--------|
| `price` | Current price (from live bar) | `LiveBar.Close` |
| `volume_live` | Current live volume | `LiveBar.Volume` |
| `vwap_live` | Current live VWAP | `LiveBar.VWAP` |

### Finalized Bar Metrics

Computed from the last finalized bar:

| Name | Description | Source |
|------|-------------|--------|
| `close` | Close price | `LastFinalizedBar.Close` |
| `open` | Open price | `LastFinalizedBar.Open` |
| `high` | High price | `LastFinalizedBar.High` |
| `low` | Low price | `LastFinalizedBar.Low` |
| `volume` | Volume | `LastFinalizedBar.Volume` |
| `vwap` | VWAP | `LastFinalizedBar.VWAP` |

### Price Change Metrics

Computed from historical finalized bars:

| Name | Description | Bars Required |
|------|-------------|---------------|
| `price_change_1m_pct` | Price change % over 1 minute | 2 bars |
| `price_change_5m_pct` | Price change % over 5 minutes | 6 bars |
| `price_change_15m_pct` | Price change % over 15 minutes | 16 bars |

**Note:** These metrics are computed from finalized bars using a bar offset, not from indicator values.

---

## Summary Statistics

### Total Indicators: 30
- **Techan Indicators:** 19
  - Momentum: 4 (RSI: 3, Stochastic: 1)
  - Trend: 12 (EMA: 7, SMA: 4, MACD: 1)
  - Volatility: 2 (ATR: 1, Bollinger Bands: 1)
- **Custom Indicators:** 11
  - Price: 3 (VWAP)
  - Volume: 3 (Volume Average)
  - Price Change: 3 (Price Change %)

### Total Metrics: 11
- **Live Bar Metrics:** 3
- **Finalized Bar Metrics:** 6
- **Price Change Metrics:** 3

---

## Usage in Rules and Toplists

### Indicators
Indicators can be referenced directly in rule conditions:
```json
{
  "metric": "rsi_14",
  "operator": "<",
  "value": 30
}
```

### Metrics
Metrics are computed on-demand and can be used in:
- Rule conditions
- Toplist sorting/filtering
- Alert payloads

---

## Implementation Details

### Indicator Engine
- **Location:** `internal/indicator/`
- **Registry:** `internal/indicator/indicator_registration.go`
- **Engine:** `internal/indicator/engine.go`
- **State Management:** `pkg/indicator/state.go`

### Metrics Registry
- **Location:** `internal/metrics/`
- **Registry:** `internal/metrics/registry.go`
- **Computers:** `internal/metrics/*_metrics.go`

### Indicator Types
- **Techan Indicators:** Wrapped via `pkg/indicator/techan_adapter.go`
- **Custom Indicators:** Implemented in `pkg/indicator/` (VWAP, Volume, PriceChange)

---

## Future Additions

See `docs/Filters.md` for planned filters and metrics that are not yet implemented.

