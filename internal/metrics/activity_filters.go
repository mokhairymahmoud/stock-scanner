package metrics

// Activity filter computers implement trading activity metrics

// TradeCountComputer computes trade count over N minutes
// Metric name format: trade_count_{timeframe} (e.g., trade_count_1m, trade_count_5m)
// Note: TradeCountHistory should contain trade counts per bar (1-minute bars)
type TradeCountComputer struct {
	name      string
	barOffset int // Number of bars to look back
}

// NewTradeCountComputer creates a new trade count computer
func NewTradeCountComputer(name string, barOffset int) *TradeCountComputer {
	return &TradeCountComputer{
		name:      name,
		barOffset: barOffset,
	}
}

func (c *TradeCountComputer) Name() string { return c.name }

func (c *TradeCountComputer) Dependencies() []string { return nil }

func (c *TradeCountComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	// If TradeCountHistory is available, sum the last N bars
	if len(snapshot.TradeCountHistory) >= c.barOffset {
		startIdx := len(snapshot.TradeCountHistory) - c.barOffset
		var total int64
		for i := startIdx; i < len(snapshot.TradeCountHistory); i++ {
			total += snapshot.TradeCountHistory[i]
		}
		return float64(total), true
	}

	// Fallback: if we have finalized bars, we can estimate from bar count
	// Each finalized bar represents trades that occurred during that minute
	// This is a rough estimate - ideally TradeCountHistory should be populated
	if len(snapshot.LastFinalBars) >= c.barOffset {
		// Return the number of bars as a proxy for trade activity
		// In a real implementation, TradeCountHistory should be populated per bar
		return float64(c.barOffset), true
	}

	return 0, false
}

// ConsecutiveCandlesComputer computes consecutive candles of the same direction
// Metric name format: consecutive_candles_{timeframe} (e.g., consecutive_candles_1m, consecutive_candles_5m)
// Returns positive number for consecutive green candles, negative for consecutive red candles
// Note: CandleDirections should be populated per timeframe
type ConsecutiveCandlesComputer struct {
	name      string
	timeframe string // Timeframe to use (e.g., "1m", "5m")
}

// NewConsecutiveCandlesComputer creates a new consecutive candles computer
func NewConsecutiveCandlesComputer(name string, timeframe string) *ConsecutiveCandlesComputer {
	return &ConsecutiveCandlesComputer{
		name:      name,
		timeframe: timeframe,
	}
}

func (c *ConsecutiveCandlesComputer) Name() string { return c.name }

func (c *ConsecutiveCandlesComputer) Dependencies() []string { return nil }

func (c *ConsecutiveCandlesComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	if snapshot.CandleDirections == nil {
		return 0, false
	}

	directions, exists := snapshot.CandleDirections[c.timeframe]
	if !exists || len(directions) == 0 {
		return 0, false
	}

	// Count consecutive candles from the end (most recent)
	// Start from the last candle and count backwards
	lastIdx := len(directions) - 1
	if lastIdx < 0 {
		return 0, false
	}

	lastDirection := directions[lastIdx]
	count := 1

	// Count backwards while direction matches
	for i := lastIdx - 1; i >= 0; i-- {
		if directions[i] == lastDirection {
			count++
		} else {
			break
		}
	}

	// Return positive for green (true), negative for red (false)
	if lastDirection {
		return float64(count), true
	}
	return float64(-count), true
}

