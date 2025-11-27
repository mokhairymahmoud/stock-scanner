package metrics

// Range filter computers implement range calculations over timeframes

// RangeComputer computes absolute range ($) over N minutes
// Metric name format: range_{timeframe} (e.g., range_2m, range_5m)
type RangeComputer struct {
	name      string
	barOffset int // Number of bars to look back
}

// NewRangeComputer creates a new range computer
func NewRangeComputer(name string, barOffset int) *RangeComputer {
	return &RangeComputer{
		name:      name,
		barOffset: barOffset,
	}
}

func (c *RangeComputer) Name() string { return c.name }

func (c *RangeComputer) Dependencies() []string { return nil }

func (c *RangeComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	if len(snapshot.LastFinalBars) < c.barOffset {
		return 0, false
	}

	// Find high and low over the timeframe
	startIdx := len(snapshot.LastFinalBars) - c.barOffset
	high := snapshot.LastFinalBars[startIdx].High
	low := snapshot.LastFinalBars[startIdx].Low

	for i := startIdx + 1; i < len(snapshot.LastFinalBars); i++ {
		bar := snapshot.LastFinalBars[i]
		if bar.High > high {
			high = bar.High
		}
		if bar.Low < low {
			low = bar.Low
		}
	}

	rangeValue := high - low
	return rangeValue, true
}

// RangePercentageComputer computes percentage range (%) over N minutes
// Metric name format: range_pct_{timeframe} (e.g., range_pct_2m, range_pct_5m)
type RangePercentageComputer struct {
	name      string
	barOffset int
}

// NewRangePercentageComputer creates a new range percentage computer
func NewRangePercentageComputer(name string, barOffset int) *RangePercentageComputer {
	return &RangePercentageComputer{
		name:      name,
		barOffset: barOffset,
	}
}

func (c *RangePercentageComputer) Name() string { return c.name }

func (c *RangePercentageComputer) Dependencies() []string { return nil }

func (c *RangePercentageComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	if len(snapshot.LastFinalBars) < c.barOffset {
		return 0, false
	}

	// Find high and low over the timeframe
	startIdx := len(snapshot.LastFinalBars) - c.barOffset
	high := snapshot.LastFinalBars[startIdx].High
	low := snapshot.LastFinalBars[startIdx].Low

	for i := startIdx + 1; i < len(snapshot.LastFinalBars); i++ {
		bar := snapshot.LastFinalBars[i]
		if bar.High > high {
			high = bar.High
		}
		if bar.Low < low {
			low = bar.Low
		}
	}

	if low <= 0 {
		return 0, false
	}

	rangePct := ((high - low) / low) * 100.0
	return rangePct, true
}

// DailyRangeComputer computes today's range ($)
type DailyRangeComputer struct{}

func (c *DailyRangeComputer) Name() string { return "range_today" }

func (c *DailyRangeComputer) Dependencies() []string { return nil }

func (c *DailyRangeComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	if len(snapshot.LastFinalBars) == 0 {
		return 0, false
	}

	// Find high and low from all finalized bars (they should all be from today)
	high := snapshot.LastFinalBars[0].High
	low := snapshot.LastFinalBars[0].Low

	for _, bar := range snapshot.LastFinalBars {
		if bar.High > high {
			high = bar.High
		}
		if bar.Low < low {
			low = bar.Low
		}
	}

	// Also check live bar if available
	if snapshot.LiveBar != nil {
		if snapshot.LiveBar.High > high {
			high = snapshot.LiveBar.High
		}
		if snapshot.LiveBar.Low < low {
			low = snapshot.LiveBar.Low
		}
	}

	rangeValue := high - low
	return rangeValue, true
}

// DailyRangePercentageComputer computes today's range percentage (%)
type DailyRangePercentageComputer struct{}

func (c *DailyRangePercentageComputer) Name() string { return "range_pct_today" }

func (c *DailyRangePercentageComputer) Dependencies() []string { return nil }

func (c *DailyRangePercentageComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	if len(snapshot.LastFinalBars) == 0 {
		return 0, false
	}

	// Find high and low from all finalized bars
	high := snapshot.LastFinalBars[0].High
	low := snapshot.LastFinalBars[0].Low

	for _, bar := range snapshot.LastFinalBars {
		if bar.High > high {
			high = bar.High
		}
		if bar.Low < low {
			low = bar.Low
		}
	}

	// Also check live bar if available
	if snapshot.LiveBar != nil {
		if snapshot.LiveBar.High > high {
			high = snapshot.LiveBar.High
		}
		if snapshot.LiveBar.Low < low {
			low = snapshot.LiveBar.Low
		}
	}

	if low <= 0 {
		return 0, false
	}

	rangePct := ((high - low) / low) * 100.0
	return rangePct, true
}

// PositionInRangeComputer computes position of current price in range (%)
// Metric name format: position_in_range_{timeframe}
// Formula: ((current_price - low) / (high - low)) * 100
type PositionInRangeComputer struct {
	name      string
	barOffset int
}

// NewPositionInRangeComputer creates a new position in range computer
func NewPositionInRangeComputer(name string, barOffset int) *PositionInRangeComputer {
	return &PositionInRangeComputer{
		name:      name,
		barOffset: barOffset,
	}
}

func (c *PositionInRangeComputer) Name() string { return c.name }

func (c *PositionInRangeComputer) Dependencies() []string { return nil }

func (c *PositionInRangeComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	if len(snapshot.LastFinalBars) < c.barOffset {
		return 0, false
	}

	// Get current price
	var currentPrice float64
	if snapshot.LiveBar != nil {
		currentPrice = snapshot.LiveBar.Close
	} else if len(snapshot.LastFinalBars) > 0 {
		currentPrice = snapshot.LastFinalBars[len(snapshot.LastFinalBars)-1].Close
	} else {
		return 0, false
	}

	// Find high and low over the timeframe
	startIdx := len(snapshot.LastFinalBars) - c.barOffset
	high := snapshot.LastFinalBars[startIdx].High
	low := snapshot.LastFinalBars[startIdx].Low

	for i := startIdx + 1; i < len(snapshot.LastFinalBars); i++ {
		bar := snapshot.LastFinalBars[i]
		if bar.High > high {
			high = bar.High
		}
		if bar.Low < low {
			low = bar.Low
		}
	}

	// Also check live bar for high/low if available
	if snapshot.LiveBar != nil {
		if snapshot.LiveBar.High > high {
			high = snapshot.LiveBar.High
		}
		if snapshot.LiveBar.Low < low {
			low = snapshot.LiveBar.Low
		}
	}

	if high <= low {
		return 0, false
	}

	position := ((currentPrice - low) / (high - low)) * 100.0
	return position, true
}

// DailyPositionInRangeComputer computes position in today's range (%)
type DailyPositionInRangeComputer struct{}

func (c *DailyPositionInRangeComputer) Name() string { return "position_in_range_today" }

func (c *DailyPositionInRangeComputer) Dependencies() []string { return nil }

func (c *DailyPositionInRangeComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	// Get current price
	var currentPrice float64
	if snapshot.LiveBar != nil {
		currentPrice = snapshot.LiveBar.Close
	} else if len(snapshot.LastFinalBars) > 0 {
		currentPrice = snapshot.LastFinalBars[len(snapshot.LastFinalBars)-1].Close
	} else {
		return 0, false
	}

	// Find high and low from all finalized bars
	if len(snapshot.LastFinalBars) == 0 {
		return 0, false
	}

	high := snapshot.LastFinalBars[0].High
	low := snapshot.LastFinalBars[0].Low

	for _, bar := range snapshot.LastFinalBars {
		if bar.High > high {
			high = bar.High
		}
		if bar.Low < low {
			low = bar.Low
		}
	}

	// Also check live bar
	if snapshot.LiveBar != nil {
		if snapshot.LiveBar.High > high {
			high = snapshot.LiveBar.High
		}
		if snapshot.LiveBar.Low < low {
			low = snapshot.LiveBar.Low
		}
	}

	if high <= low {
		return 0, false
	}

	position := ((currentPrice - low) / (high - low)) * 100.0
	return position, true
}

// RelativeRangeComputer computes relative range (%) vs ATR(14) daily
// Formula: (today_range / atr_14_daily) * 100
type RelativeRangeComputer struct{}

func (c *RelativeRangeComputer) Name() string { return "relative_range_pct" }

func (c *RelativeRangeComputer) Dependencies() []string {
	return []string{"atr_14"} // Depends on ATR(14) indicator (using atr_14 until daily ATR is implemented)
}

func (c *RelativeRangeComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	// Get today's range
	dailyRangeComputer := &DailyRangeComputer{}
	todayRange, ok := dailyRangeComputer.Compute(snapshot)
	if !ok || todayRange <= 0 {
		return 0, false
	}

	// Get ATR(14) from indicators (using atr_14 until daily ATR is implemented)
	atrKey := "atr_14"
	atrValue, exists := snapshot.Indicators[atrKey]
	if !exists || atrValue <= 0 {
		return 0, false
	}

	relativeRange := (todayRange / atrValue) * 100.0
	return relativeRange, true
}

