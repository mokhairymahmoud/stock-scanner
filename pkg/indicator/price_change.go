package indicator

import (
	"fmt"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

// PriceChange calculates the percentage price change over a time window
type PriceChange struct {
	window    time.Duration
	name      string
	bars      []*models.Bar1m
	ready     bool
	processed int
}

// NewPriceChange creates a new price change calculator
func NewPriceChange(window time.Duration) (*PriceChange, error) {
	if window <= 0 {
		return nil, fmt.Errorf("price change window must be positive, got %v", window)
	}

	name := fmt.Sprintf("price_change_%s_pct", formatDuration(window))

	return &PriceChange{
		window:    window,
		name:      name,
		bars:      make([]*models.Bar1m, 0),
		ready:     false,
		processed: 0,
	}, nil
}

// Name returns the indicator name
func (p *PriceChange) Name() string {
	return p.name
}

// Update processes a new bar and updates the price change calculation
func (p *PriceChange) Update(bar *models.Bar1m) (float64, error) {
	if bar == nil {
		return 0, fmt.Errorf("bar cannot be nil")
	}

	// Add new bar
	p.bars = append(p.bars, bar)
	p.processed++

	// Remove bars outside the window
	cutoffTime := bar.Timestamp.Add(-p.window)
	p.bars = p.filterBarsByTime(cutoffTime)

	// Need at least 2 bars to calculate change
	if len(p.bars) >= 2 {
		p.ready = true
		return p.calculateChange(), nil
	}

	p.ready = false
	return 0, nil
}

// filterBarsByTime removes bars older than the cutoff time
func (p *PriceChange) filterBarsByTime(cutoffTime time.Time) []*models.Bar1m {
	filtered := make([]*models.Bar1m, 0, len(p.bars))
	for _, bar := range p.bars {
		if bar.Timestamp.After(cutoffTime) || bar.Timestamp.Equal(cutoffTime) {
			filtered = append(filtered, bar)
		}
	}
	return filtered
}

// calculateChange computes the percentage price change
func (p *PriceChange) calculateChange() float64 {
	if len(p.bars) < 2 {
		return 0
	}

	// Oldest bar in window
	oldestBar := p.bars[0]
	// Newest bar (current)
	newestBar := p.bars[len(p.bars)-1]

	if oldestBar.Close == 0 {
		return 0
	}

	change := ((newestBar.Close - oldestBar.Close) / oldestBar.Close) * 100.0
	return change
}

// Value returns the current price change percentage
func (p *PriceChange) Value() (float64, error) {
	if !p.ready {
		return 0, fmt.Errorf("price change not ready: need at least 2 bars in window")
	}
	return p.calculateChange(), nil
}

// Reset clears the price change state
func (p *PriceChange) Reset() {
	p.bars = p.bars[:0]
	p.ready = false
	p.processed = 0
}

// IsReady returns true if the price change can be calculated
func (p *PriceChange) IsReady() bool {
	return p.ready
}

// WindowSize returns an estimate based on window duration
func (p *PriceChange) WindowSize() int {
	minutes := int(p.window.Minutes())
	if minutes < 1 {
		return 2 // Minimum 2 bars needed
	}
	return minutes
}

// BarsProcessed returns the number of bars processed
func (p *PriceChange) BarsProcessed() int {
	return p.processed
}

