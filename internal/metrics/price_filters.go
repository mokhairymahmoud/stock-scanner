package metrics

// Price filter computers implement various price change and gap calculations

// ChangeComputer computes absolute price change over N minutes
// Metric name format: change_{timeframe} (e.g., change_1m, change_5m)
type ChangeComputer struct {
	name      string
	barOffset int // Number of bars to look back
}

// NewChangeComputer creates a new change computer
func NewChangeComputer(name string, barOffset int) *ChangeComputer {
	return &ChangeComputer{
		name:      name,
		barOffset: barOffset,
	}
}

func (c *ChangeComputer) Name() string { return c.name }

func (c *ChangeComputer) Dependencies() []string { return nil }

func (c *ChangeComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	if len(snapshot.LastFinalBars) < c.barOffset {
		return 0, false
	}

	currentBar := snapshot.LastFinalBars[len(snapshot.LastFinalBars)-1]
	pastBar := snapshot.LastFinalBars[len(snapshot.LastFinalBars)-c.barOffset]

	change := currentBar.Close - pastBar.Close
	return change, true
}

// ChangeFromCloseComputer computes change from yesterday's close
type ChangeFromCloseComputer struct{}

func (c *ChangeFromCloseComputer) Name() string { return "change_from_close" }

func (c *ChangeFromCloseComputer) Dependencies() []string { return nil }

func (c *ChangeFromCloseComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	if snapshot.YesterdayClose <= 0 {
		return 0, false
	}

	var currentPrice float64
	if snapshot.LiveBar != nil {
		currentPrice = snapshot.LiveBar.Close
	} else if len(snapshot.LastFinalBars) > 0 {
		currentPrice = snapshot.LastFinalBars[len(snapshot.LastFinalBars)-1].Close
	} else {
		return 0, false
	}

	change := currentPrice - snapshot.YesterdayClose
	return change, true
}

// ChangeFromClosePctComputer computes percentage change from yesterday's close
type ChangeFromClosePctComputer struct{}

func (c *ChangeFromClosePctComputer) Name() string { return "change_from_close_pct" }

func (c *ChangeFromClosePctComputer) Dependencies() []string { return nil }

func (c *ChangeFromClosePctComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	if snapshot.YesterdayClose <= 0 {
		return 0, false
	}

	var currentPrice float64
	if snapshot.LiveBar != nil {
		currentPrice = snapshot.LiveBar.Close
	} else if len(snapshot.LastFinalBars) > 0 {
		currentPrice = snapshot.LastFinalBars[len(snapshot.LastFinalBars)-1].Close
	} else {
		return 0, false
	}

	changePct := ((currentPrice - snapshot.YesterdayClose) / snapshot.YesterdayClose) * 100.0
	return changePct, true
}

// ChangeFromClosePremarketComputer computes change from yesterday's close (premarket only)
type ChangeFromClosePremarketComputer struct{}

func (c *ChangeFromClosePremarketComputer) Name() string { return "change_from_close_premarket" }

func (c *ChangeFromClosePremarketComputer) Dependencies() []string { return nil }

func (c *ChangeFromClosePremarketComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	// Only compute during premarket
	if snapshot.CurrentSession != "premarket" {
		return 0, false
	}

	if snapshot.YesterdayClose <= 0 {
		return 0, false
	}

	var currentPrice float64
	if snapshot.LiveBar != nil {
		currentPrice = snapshot.LiveBar.Close
	} else if len(snapshot.LastFinalBars) > 0 {
		currentPrice = snapshot.LastFinalBars[len(snapshot.LastFinalBars)-1].Close
	} else {
		return 0, false
	}

	change := currentPrice - snapshot.YesterdayClose
	return change, true
}

// ChangeFromClosePremarketPctComputer computes percentage change from yesterday's close (premarket only)
type ChangeFromClosePremarketPctComputer struct{}

func (c *ChangeFromClosePremarketPctComputer) Name() string { return "change_from_close_premarket_pct" }

func (c *ChangeFromClosePremarketPctComputer) Dependencies() []string { return nil }

func (c *ChangeFromClosePremarketPctComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	// Only compute during premarket
	if snapshot.CurrentSession != "premarket" {
		return 0, false
	}

	if snapshot.YesterdayClose <= 0 {
		return 0, false
	}

	var currentPrice float64
	if snapshot.LiveBar != nil {
		currentPrice = snapshot.LiveBar.Close
	} else if len(snapshot.LastFinalBars) > 0 {
		currentPrice = snapshot.LastFinalBars[len(snapshot.LastFinalBars)-1].Close
	} else {
		return 0, false
	}

	changePct := ((currentPrice - snapshot.YesterdayClose) / snapshot.YesterdayClose) * 100.0
	return changePct, true
}

// ChangeFromClosePostmarketComputer computes change from today's close (postmarket only)
type ChangeFromClosePostmarketComputer struct{}

func (c *ChangeFromClosePostmarketComputer) Name() string { return "change_from_close_postmarket" }

func (c *ChangeFromClosePostmarketComputer) Dependencies() []string { return nil }

func (c *ChangeFromClosePostmarketComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	// Only compute during postmarket
	if snapshot.CurrentSession != "postmarket" {
		return 0, false
	}

	if snapshot.TodayClose <= 0 {
		return 0, false
	}

	var currentPrice float64
	if snapshot.LiveBar != nil {
		currentPrice = snapshot.LiveBar.Close
	} else if len(snapshot.LastFinalBars) > 0 {
		currentPrice = snapshot.LastFinalBars[len(snapshot.LastFinalBars)-1].Close
	} else {
		return 0, false
	}

	change := currentPrice - snapshot.TodayClose
	return change, true
}

// ChangeFromClosePostmarketPctComputer computes percentage change from today's close (postmarket only)
type ChangeFromClosePostmarketPctComputer struct{}

func (c *ChangeFromClosePostmarketPctComputer) Name() string { return "change_from_close_postmarket_pct" }

func (c *ChangeFromClosePostmarketPctComputer) Dependencies() []string { return nil }

func (c *ChangeFromClosePostmarketPctComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	// Only compute during postmarket
	if snapshot.CurrentSession != "postmarket" {
		return 0, false
	}

	if snapshot.TodayClose <= 0 {
		return 0, false
	}

	var currentPrice float64
	if snapshot.LiveBar != nil {
		currentPrice = snapshot.LiveBar.Close
	} else if len(snapshot.LastFinalBars) > 0 {
		currentPrice = snapshot.LastFinalBars[len(snapshot.LastFinalBars)-1].Close
	} else {
		return 0, false
	}

	changePct := ((currentPrice - snapshot.TodayClose) / snapshot.TodayClose) * 100.0
	return changePct, true
}

// ChangeFromOpenComputer computes change from today's open
type ChangeFromOpenComputer struct{}

func (c *ChangeFromOpenComputer) Name() string { return "change_from_open" }

func (c *ChangeFromOpenComputer) Dependencies() []string { return nil }

func (c *ChangeFromOpenComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	if snapshot.TodayOpen <= 0 {
		return 0, false
	}

	var currentPrice float64
	if snapshot.LiveBar != nil {
		currentPrice = snapshot.LiveBar.Close
	} else if len(snapshot.LastFinalBars) > 0 {
		currentPrice = snapshot.LastFinalBars[len(snapshot.LastFinalBars)-1].Close
	} else {
		return 0, false
	}

	change := currentPrice - snapshot.TodayOpen
	return change, true
}

// ChangeFromOpenPctComputer computes percentage change from today's open
type ChangeFromOpenPctComputer struct{}

func (c *ChangeFromOpenPctComputer) Name() string { return "change_from_open_pct" }

func (c *ChangeFromOpenPctComputer) Dependencies() []string { return nil }

func (c *ChangeFromOpenPctComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	if snapshot.TodayOpen <= 0 {
		return 0, false
	}

	var currentPrice float64
	if snapshot.LiveBar != nil {
		currentPrice = snapshot.LiveBar.Close
	} else if len(snapshot.LastFinalBars) > 0 {
		currentPrice = snapshot.LastFinalBars[len(snapshot.LastFinalBars)-1].Close
	} else {
		return 0, false
	}

	changePct := ((currentPrice - snapshot.TodayOpen) / snapshot.TodayOpen) * 100.0
	return changePct, true
}

// GapFromCloseComputer computes gap from yesterday's close to today's open
type GapFromCloseComputer struct{}

func (c *GapFromCloseComputer) Name() string { return "gap_from_close" }

func (c *GapFromCloseComputer) Dependencies() []string { return nil }

func (c *GapFromCloseComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	if snapshot.YesterdayClose <= 0 || snapshot.TodayOpen <= 0 {
		return 0, false
	}

	gap := snapshot.TodayOpen - snapshot.YesterdayClose
	return gap, true
}

// GapFromClosePctComputer computes percentage gap from yesterday's close to today's open
type GapFromClosePctComputer struct{}

func (c *GapFromClosePctComputer) Name() string { return "gap_from_close_pct" }

func (c *GapFromClosePctComputer) Dependencies() []string { return nil }

func (c *GapFromClosePctComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	if snapshot.YesterdayClose <= 0 || snapshot.TodayOpen <= 0 {
		return 0, false
	}

	gapPct := ((snapshot.TodayOpen - snapshot.YesterdayClose) / snapshot.YesterdayClose) * 100.0
	return gapPct, true
}

