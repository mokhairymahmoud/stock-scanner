package metrics

// PriceChangeComputer computes price change percentage over N minutes
// barOffset is the number of bars to look back (e.g., 2 for 1m, 6 for 5m, 16 for 15m)
type PriceChangeComputer struct {
	name      string
	barOffset int
}

// NewPriceChangeComputer creates a new price change computer
func NewPriceChangeComputer(name string, barOffset int) *PriceChangeComputer {
	return &PriceChangeComputer{
		name:      name,
		barOffset: barOffset,
	}
}

func (c *PriceChangeComputer) Name() string { return c.name }

func (c *PriceChangeComputer) Dependencies() []string {
	// Price change metrics depend on having finalized bars
	// We don't have a direct dependency on "close" metric, but we need bars
	return nil
}

func (c *PriceChangeComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	if len(snapshot.LastFinalBars) < c.barOffset {
		return 0, false
	}

	currentBar := snapshot.LastFinalBars[len(snapshot.LastFinalBars)-1]
	pastBar := snapshot.LastFinalBars[len(snapshot.LastFinalBars)-c.barOffset]

	if pastBar.Close <= 0 {
		return 0, false
	}

	changePct := ((currentBar.Close - pastBar.Close) / pastBar.Close) * 100.0
	return changePct, true
}

