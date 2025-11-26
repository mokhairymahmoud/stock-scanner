package metrics

// PriceComputer computes current price from live bar
type PriceComputer struct{}

func (c *PriceComputer) Name() string { return "price" }

func (c *PriceComputer) Dependencies() []string { return nil }

func (c *PriceComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	if snapshot.LiveBar == nil {
		return 0, false
	}
	return snapshot.LiveBar.Close, true
}

// VolumeLiveComputer computes live volume from live bar
type VolumeLiveComputer struct{}

func (c *VolumeLiveComputer) Name() string { return "volume_live" }

func (c *VolumeLiveComputer) Dependencies() []string { return nil }

func (c *VolumeLiveComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	if snapshot.LiveBar == nil {
		return 0, false
	}
	return float64(snapshot.LiveBar.Volume), true
}

// VWAPLiveComputer computes live VWAP from live bar
type VWAPLiveComputer struct{}

func (c *VWAPLiveComputer) Name() string { return "vwap_live" }

func (c *VWAPLiveComputer) Dependencies() []string { return nil }

func (c *VWAPLiveComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	if snapshot.LiveBar == nil || snapshot.LiveBar.VWAPDenom == 0 {
		return 0, false
	}
	return snapshot.LiveBar.VWAPNum / snapshot.LiveBar.VWAPDenom, true
}

