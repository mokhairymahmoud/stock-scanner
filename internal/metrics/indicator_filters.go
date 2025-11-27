package metrics

// Indicator filter computers implement distance calculations from technical indicators

// ATRPComputer computes ATR Percentage (ATR / Close * 100)
// Metric name format: atrp_14_{timeframe} (e.g., atrp_14_1m, atrp_14_daily)
// Note: Currently uses atr_14 from indicators. For daily, we may need atr_14_daily later.
type ATRPComputer struct {
	name      string
	atrKey    string // Key to look up ATR in indicators (e.g., "atr_14")
}

// NewATRPComputer creates a new ATRP computer
func NewATRPComputer(name string, atrKey string) *ATRPComputer {
	return &ATRPComputer{
		name:   name,
		atrKey: atrKey,
	}
}

func (c *ATRPComputer) Name() string { return c.name }

func (c *ATRPComputer) Dependencies() []string {
	return []string{c.atrKey}
}

func (c *ATRPComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	// Get ATR from indicators
	atrValue, exists := snapshot.Indicators[c.atrKey]
	if !exists || atrValue <= 0 {
		return 0, false
	}

	// Get current close price
	var closePrice float64
	if snapshot.LiveBar != nil {
		closePrice = snapshot.LiveBar.Close
	} else if len(snapshot.LastFinalBars) > 0 {
		closePrice = snapshot.LastFinalBars[len(snapshot.LastFinalBars)-1].Close
	} else {
		return 0, false
	}

	if closePrice <= 0 {
		return 0, false
	}

	atrp := (atrValue / closePrice) * 100.0
	return atrp, true
}

// VWAPDistanceComputer computes distance from VWAP ($)
// Metric name format: vwap_dist_{timeframe} (e.g., vwap_dist_5m, vwap_dist_15m)
type VWAPDistanceComputer struct {
	name      string
	vwapKey   string // Key to look up VWAP in indicators (e.g., "vwap_5m")
}

// NewVWAPDistanceComputer creates a new VWAP distance computer
func NewVWAPDistanceComputer(name string, vwapKey string) *VWAPDistanceComputer {
	return &VWAPDistanceComputer{
		name:    name,
		vwapKey: vwapKey,
	}
}

func (c *VWAPDistanceComputer) Name() string { return c.name }

func (c *VWAPDistanceComputer) Dependencies() []string {
	return []string{c.vwapKey}
}

func (c *VWAPDistanceComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	// Get VWAP from indicators
	vwapValue, exists := snapshot.Indicators[c.vwapKey]
	if !exists || vwapValue <= 0 {
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

	distance := currentPrice - vwapValue
	if distance < 0 {
		distance = -distance // Absolute distance
	}
	return distance, true
}

// VWAPDistancePctComputer computes distance from VWAP (%)
// Metric name format: vwap_dist_{timeframe}_pct
type VWAPDistancePctComputer struct {
	name      string
	vwapKey   string
}

// NewVWAPDistancePctComputer creates a new VWAP distance percentage computer
func NewVWAPDistancePctComputer(name string, vwapKey string) *VWAPDistancePctComputer {
	return &VWAPDistancePctComputer{
		name:    name,
		vwapKey: vwapKey,
	}
}

func (c *VWAPDistancePctComputer) Name() string { return c.name }

func (c *VWAPDistancePctComputer) Dependencies() []string {
	return []string{c.vwapKey}
}

func (c *VWAPDistancePctComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	// Get VWAP from indicators
	vwapValue, exists := snapshot.Indicators[c.vwapKey]
	if !exists || vwapValue <= 0 {
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

	distancePct := ((currentPrice - vwapValue) / vwapValue) * 100.0
	if distancePct < 0 {
		distancePct = -distancePct // Absolute percentage
	}
	return distancePct, true
}

// MADistanceComputer computes distance from Moving Average (%)
// Metric name format: ma_dist_{ma_type}_{timeframe}_pct
// Example: ma_dist_ema9_5m_pct, ma_dist_sma20_daily_pct
type MADistanceComputer struct {
	name    string
	maKey   string // Key to look up MA in indicators (e.g., "ema_9", "sma_20")
}

// NewMADistanceComputer creates a new MA distance computer
func NewMADistanceComputer(name string, maKey string) *MADistanceComputer {
	return &MADistanceComputer{
		name:  name,
		maKey: maKey,
	}
}

func (c *MADistanceComputer) Name() string { return c.name }

func (c *MADistanceComputer) Dependencies() []string {
	return []string{c.maKey}
}

func (c *MADistanceComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	// Get MA value from indicators
	maValue, exists := snapshot.Indicators[c.maKey]
	if !exists || maValue <= 0 {
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

	distancePct := ((currentPrice - maValue) / maValue) * 100.0
	return distancePct, true
}

