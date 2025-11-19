package indicator

import (
	"fmt"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

// VWAP calculates the Volume Weighted Average Price over a time window
// VWAP = Sum(Price * Volume) / Sum(Volume) over the window
type VWAP struct {
	window    time.Duration // Time window (e.g., 5m, 15m, 1h)
	name      string
	bars      []*models.Bar1m // Bars within the window
	ready     bool
	processed int
}

// NewVWAP creates a new VWAP calculator with the specified time window
func NewVWAP(window time.Duration) (*VWAP, error) {
	if window <= 0 {
		return nil, fmt.Errorf("VWAP window must be positive, got %v", window)
	}

	// Generate name based on window duration
	name := fmt.Sprintf("vwap_%s", formatDuration(window))

	return &VWAP{
		window:    window,
		name:      name,
		bars:      make([]*models.Bar1m, 0),
		ready:     false,
		processed: 0,
	}, nil
}

// Name returns the indicator name
func (v *VWAP) Name() string {
	return v.name
}

// Update processes a new bar and updates the VWAP calculation
func (v *VWAP) Update(bar *models.Bar1m) (float64, error) {
	if bar == nil {
		return 0, fmt.Errorf("bar cannot be nil")
	}

	// Add new bar
	v.bars = append(v.bars, bar)
	v.processed++

	// Remove bars outside the window
	cutoffTime := bar.Timestamp.Add(-v.window)
	v.bars = v.filterBarsByTime(cutoffTime)

	// Check if we have any bars in the window
	if len(v.bars) > 0 {
		v.ready = true
		return v.calculateVWAP(), nil
	}

	v.ready = false
	return 0, nil
}

// filterBarsByTime removes bars older than the cutoff time
func (v *VWAP) filterBarsByTime(cutoffTime time.Time) []*models.Bar1m {
	filtered := make([]*models.Bar1m, 0, len(v.bars))
	for _, bar := range v.bars {
		if bar.Timestamp.After(cutoffTime) || bar.Timestamp.Equal(cutoffTime) {
			filtered = append(filtered, bar)
		}
	}
	return filtered
}

// calculateVWAP computes the VWAP value
func (v *VWAP) calculateVWAP() float64 {
	if len(v.bars) == 0 {
		return 0
	}

	var totalPriceVolume float64
	var totalVolume int64

	for _, bar := range v.bars {
		// Use typical price (HLC/3) or close price
		typicalPrice := (bar.High + bar.Low + bar.Close) / 3.0
		totalPriceVolume += typicalPrice * float64(bar.Volume)
		totalVolume += bar.Volume
	}

	if totalVolume == 0 {
		return 0
	}

	return totalPriceVolume / float64(totalVolume)
}

// Value returns the current VWAP value
func (v *VWAP) Value() (float64, error) {
	if !v.ready {
		return 0, fmt.Errorf("VWAP not ready: no bars in window")
	}
	return v.calculateVWAP(), nil
}

// Reset clears the VWAP state
func (v *VWAP) Reset() {
	v.bars = v.bars[:0]
	v.ready = false
	v.processed = 0
}

// IsReady returns true if the VWAP has enough data
func (v *VWAP) IsReady() bool {
	return v.ready
}

// WindowSize returns an estimate based on window duration (assuming 1-minute bars)
func (v *VWAP) WindowSize() int {
	// Estimate: window duration in minutes (assuming 1-minute bars)
	minutes := int(v.window.Minutes())
	if minutes < 1 {
		return 1
	}
	return minutes
}

// BarsProcessed returns the number of bars processed
func (v *VWAP) BarsProcessed() int {
	return v.processed
}
