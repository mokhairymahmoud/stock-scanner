package metrics

// Volume filter computers implement various volume calculations

// PostmarketVolumeComputer computes postmarket volume
type PostmarketVolumeComputer struct{}

func (c *PostmarketVolumeComputer) Name() string { return "postmarket_volume" }

func (c *PostmarketVolumeComputer) Dependencies() []string { return nil }

func (c *PostmarketVolumeComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	return float64(snapshot.PostmarketVolume), true
}

// PremarketVolumeComputer computes premarket volume
type PremarketVolumeComputer struct{}

func (c *PremarketVolumeComputer) Name() string { return "premarket_volume" }

func (c *PremarketVolumeComputer) Dependencies() []string { return nil }

func (c *PremarketVolumeComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	return float64(snapshot.PremarketVolume), true
}

// AbsoluteVolumeComputer computes absolute volume over N minutes
// Metric name format: volume_{timeframe} (e.g., volume_1m, volume_5m)
type AbsoluteVolumeComputer struct {
	name      string
	barOffset int // Number of bars to sum
}

// NewAbsoluteVolumeComputer creates a new absolute volume computer
func NewAbsoluteVolumeComputer(name string, barOffset int) *AbsoluteVolumeComputer {
	return &AbsoluteVolumeComputer{
		name:      name,
		barOffset: barOffset,
	}
}

func (c *AbsoluteVolumeComputer) Name() string { return c.name }

func (c *AbsoluteVolumeComputer) Dependencies() []string { return nil }

func (c *AbsoluteVolumeComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	if len(snapshot.LastFinalBars) < c.barOffset {
		return 0, false
	}

	// Sum volume from last N bars
	var totalVolume int64
	startIdx := len(snapshot.LastFinalBars) - c.barOffset
	for i := startIdx; i < len(snapshot.LastFinalBars); i++ {
		totalVolume += snapshot.LastFinalBars[i].Volume
	}

	return float64(totalVolume), true
}

// DollarVolumeComputer computes dollar volume (price * volume) over N minutes
// Metric name format: dollar_volume_{timeframe} (e.g., dollar_volume_1m, dollar_volume_5m)
type DollarVolumeComputer struct {
	name      string
	barOffset int // Number of bars to sum
}

// NewDollarVolumeComputer creates a new dollar volume computer
func NewDollarVolumeComputer(name string, barOffset int) *DollarVolumeComputer {
	return &DollarVolumeComputer{
		name:      name,
		barOffset: barOffset,
	}
}

func (c *DollarVolumeComputer) Name() string { return c.name }

func (c *DollarVolumeComputer) Dependencies() []string { return nil }

func (c *DollarVolumeComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	if len(snapshot.LastFinalBars) < c.barOffset {
		return 0, false
	}

	// Sum dollar volume from last N bars
	// Dollar volume = price * volume (using VWAP as price proxy, or close price)
	var totalDollarVolume float64
	startIdx := len(snapshot.LastFinalBars) - c.barOffset
	for i := startIdx; i < len(snapshot.LastFinalBars); i++ {
		bar := snapshot.LastFinalBars[i]
		// Use VWAP if available, otherwise use close price
		price := bar.VWAP
		if price <= 0 {
			price = bar.Close
		}
		dollarVolume := price * float64(bar.Volume)
		totalDollarVolume += dollarVolume
	}

	return totalDollarVolume, true
}

// DailyVolumeComputer computes daily volume (sum of all bars today)
type DailyVolumeComputer struct{}

func (c *DailyVolumeComputer) Name() string { return "volume_daily" }

func (c *DailyVolumeComputer) Dependencies() []string { return nil }

func (c *DailyVolumeComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	// Sum all finalized bars (they should all be from today)
	var totalVolume int64
	for _, bar := range snapshot.LastFinalBars {
		totalVolume += bar.Volume
	}

	// Also add live bar volume if available
	if snapshot.LiveBar != nil {
		totalVolume += snapshot.LiveBar.Volume
	}

	return float64(totalVolume), true
}

// DailyDollarVolumeComputer computes daily dollar volume
type DailyDollarVolumeComputer struct{}

func (c *DailyDollarVolumeComputer) Name() string { return "dollar_volume_daily" }

func (c *DailyDollarVolumeComputer) Dependencies() []string { return nil }

func (c *DailyDollarVolumeComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	var totalDollarVolume float64

	// Sum dollar volume from all finalized bars
	for _, bar := range snapshot.LastFinalBars {
		price := bar.VWAP
		if price <= 0 {
			price = bar.Close
		}
		totalDollarVolume += price * float64(bar.Volume)
	}

	// Also add live bar dollar volume if available
	if snapshot.LiveBar != nil {
		price := snapshot.LiveBar.Close
		if snapshot.LiveBar.VWAPDenom > 0 {
			price = snapshot.LiveBar.VWAPNum / snapshot.LiveBar.VWAPDenom
		}
		totalDollarVolume += price * float64(snapshot.LiveBar.Volume)
	}

	return totalDollarVolume, true
}

