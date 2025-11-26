package metrics

// CloseComputer computes close price from last finalized bar
type CloseComputer struct{}

func (c *CloseComputer) Name() string { return "close" }

func (c *CloseComputer) Dependencies() []string { return nil }

func (c *CloseComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	if len(snapshot.LastFinalBars) == 0 {
		return 0, false
	}
	lastBar := snapshot.LastFinalBars[len(snapshot.LastFinalBars)-1]
	return lastBar.Close, true
}

// OpenComputer computes open price from last finalized bar
type OpenComputer struct{}

func (c *OpenComputer) Name() string { return "open" }

func (c *OpenComputer) Dependencies() []string { return nil }

func (c *OpenComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	if len(snapshot.LastFinalBars) == 0 {
		return 0, false
	}
	lastBar := snapshot.LastFinalBars[len(snapshot.LastFinalBars)-1]
	return lastBar.Open, true
}

// HighComputer computes high price from last finalized bar
type HighComputer struct{}

func (c *HighComputer) Name() string { return "high" }

func (c *HighComputer) Dependencies() []string { return nil }

func (c *HighComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	if len(snapshot.LastFinalBars) == 0 {
		return 0, false
	}
	lastBar := snapshot.LastFinalBars[len(snapshot.LastFinalBars)-1]
	return lastBar.High, true
}

// LowComputer computes low price from last finalized bar
type LowComputer struct{}

func (c *LowComputer) Name() string { return "low" }

func (c *LowComputer) Dependencies() []string { return nil }

func (c *LowComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	if len(snapshot.LastFinalBars) == 0 {
		return 0, false
	}
	lastBar := snapshot.LastFinalBars[len(snapshot.LastFinalBars)-1]
	return lastBar.Low, true
}

// VolumeComputer computes volume from last finalized bar
type VolumeComputer struct{}

func (c *VolumeComputer) Name() string { return "volume" }

func (c *VolumeComputer) Dependencies() []string { return nil }

func (c *VolumeComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	if len(snapshot.LastFinalBars) == 0 {
		return 0, false
	}
	lastBar := snapshot.LastFinalBars[len(snapshot.LastFinalBars)-1]
	return float64(lastBar.Volume), true
}

// VWAPComputer computes VWAP from last finalized bar
type VWAPComputer struct{}

func (c *VWAPComputer) Name() string { return "vwap" }

func (c *VWAPComputer) Dependencies() []string { return nil }

func (c *VWAPComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	if len(snapshot.LastFinalBars) == 0 {
		return 0, false
	}
	lastBar := snapshot.LastFinalBars[len(snapshot.LastFinalBars)-1]
	return lastBar.VWAP, true
}

