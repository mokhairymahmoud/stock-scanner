# Indicator Computation Behavior

## Current Behavior: Computing All Indicators by Default

**Yes, this is normal!** The indicator engine currently computes **all registered indicators** by default, even when no rules are created.

### Why This Happens

Looking at the engine code (`internal/indicator/engine.go`):

```go
// Line 49: requiredIndicators starts empty
requiredIndicators: make(map[string]bool), // Empty = all indicators

// Line 98: If empty, compute all indicators
allIndicators := len(e.requiredIndicators) == 0
```

**Default Behavior:**
- When `requiredIndicators` is empty (default), the engine computes **ALL** registered indicators
- This ensures all indicators are available immediately when rules/toplists are created
- No requirement tracking system is active yet (planned for future)

### What You're Seeing

From your Redis output:
```json
{
  "price_change_15m_pct": -0.23566964282885922,
  "price_change_1m_pct": -1.545640553430384,
  "price_change_5m_pct": -0.23566964282885922,
  "volume_avg_15m": 360790.4,
  "volume_avg_1h": 360790.4,
  "volume_avg_5m": 360790.4,
  "vwap_15m": 343.0656678526916,
  "vwap_1h": 343.0656678526916,
  "vwap_5m": 343.0656678526916
}
```

**What's Computed:**
- ✅ Custom indicators (VWAP, Volume Average, Price Change) - These are ready immediately
- ❌ Techan indicators (RSI, EMA, SMA, ATR, etc.) - **Not showing because they need more bars**

### Why Techan Indicators Aren't Showing

Techan indicators require a minimum number of bars before they become "ready":

| Indicator | Minimum Bars Required |
|-----------|----------------------|
| RSI (14) | 15 bars (14 periods + 1 for first change) |
| EMA (20) | 1 bar (starts immediately) |
| SMA (20) | 20 bars |
| ATR (14) | 14 bars |
| MACD (12,26,9) | 26 bars (slow period) |

**The `GetAllValues()` method only returns indicators where `IsReady() == true`:**

```go
func (s *SymbolState) GetAllValues() map[string]float64 {
    values := make(map[string]float64)
    for name, calc := range s.calculators {
        if calc.IsReady() {  // ← Only ready indicators are included
            if val, err := calc.Value(); err == nil {
                values[name] = val
            }
        }
    }
    return values
}
```

So:
- **Custom indicators** (VWAP, Volume Average, Price Change) are ready immediately → **Shown**
- **Techan indicators** need more bars → **Computed but not shown until ready**

### How to Verify Techan Indicators Are Being Computed

Check the indicator engine logs or add more bars. After 15+ bars, you should see:
- `rsi_14` (after 15 bars)
- `ema_20` (after 1 bar)
- `sma_20` (after 20 bars)
- `atr_14` (after 14 bars)

## Performance Impact

### Current State
- **CPU Usage**: Computing all indicators for all symbols
- **Memory Usage**: Storing calculator instances for all indicators
- **Network**: Publishing all computed indicators to Redis

### Optimization: Requirement-Based Computation

The engine supports dynamic indicator computation, but it's not enabled by default. To enable it:

1. **Implement Requirement Tracker** (from `TECHAN_INTEGRATION_PLAN.md`)
2. **Set Required Indicators** on the engine:

```go
// In cmd/indicator/main.go
requiredIndicators := map[string]bool{
    "rsi_14": true,
    "ema_20": true,
    // Only indicators used in rules/toplists
}
engine.SetRequiredIndicators(requiredIndicators)
```

This will:
- ✅ Only compute indicators that are actually needed
- ✅ Reduce CPU usage
- ✅ Reduce memory usage
- ✅ Reduce network traffic

## Summary

| Question | Answer |
|----------|--------|
| **Is it normal?** | Yes, by default all indicators are computed |
| **Why are they computed without rules?** | Default behavior ensures indicators are ready when rules are created |
| **Why aren't Techan indicators showing?** | They need more bars to be "ready" (RSI needs 15, SMA needs 20, etc.) |
| **Is this a problem?** | Not immediately, but can be optimized with requirement tracking |
| **How to optimize?** | Implement requirement tracking from rules/toplists (planned feature) |

## Next Steps

1. **Short-term**: This is working as designed. Techan indicators will appear once enough bars are processed.

2. **Medium-term**: Implement requirement tracking to only compute needed indicators:
   - Track indicators used in active rules
   - Track indicators used in active toplists
   - Update engine's `requiredIndicators` dynamically

3. **Verification**: After 20+ bars, check Redis again - you should see:
   ```json
   {
     "rsi_14": 45.2,
     "ema_20": 150.5,
     "sma_20": 151.2,
     "atr_14": 2.3,
     // ... plus your existing custom indicators
   }
   ```

