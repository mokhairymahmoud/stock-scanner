package indicator

import (
	"fmt"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

// VolumeAverage calculates the average volume over a time window
type VolumeAverage struct {
	window    time.Duration
	name      string
	bars      []*models.Bar1m
	ready     bool
	processed int
}

// NewVolumeAverage creates a new volume average calculator
func NewVolumeAverage(window time.Duration) (*VolumeAverage, error) {
	if window <= 0 {
		return nil, fmt.Errorf("volume average window must be positive, got %v", window)
	}

	name := fmt.Sprintf("volume_avg_%s", formatDuration(window))

	return &VolumeAverage{
		window:    window,
		name:      name,
		bars:      make([]*models.Bar1m, 0),
		ready:     false,
		processed: 0,
	}, nil
}

// Name returns the indicator name
func (v *VolumeAverage) Name() string {
	return v.name
}

// Update processes a new bar and updates the volume average
func (v *VolumeAverage) Update(bar *models.Bar1m) (float64, error) {
	if bar == nil {
		return 0, fmt.Errorf("bar cannot be nil")
	}

	// Add new bar
	v.bars = append(v.bars, bar)
	v.processed++

	// Remove bars outside the window
	cutoffTime := bar.Timestamp.Add(-v.window)
	v.bars = v.filterBarsByTime(cutoffTime)

	// Check if we have any bars
	if len(v.bars) > 0 {
		v.ready = true
		return v.calculateAverage(), nil
	}

	v.ready = false
	return 0, nil
}

// filterBarsByTime removes bars older than the cutoff time
func (v *VolumeAverage) filterBarsByTime(cutoffTime time.Time) []*models.Bar1m {
	filtered := make([]*models.Bar1m, 0, len(v.bars))
	for _, bar := range v.bars {
		if bar.Timestamp.After(cutoffTime) || bar.Timestamp.Equal(cutoffTime) {
			filtered = append(filtered, bar)
		}
	}
	return filtered
}

// calculateAverage computes the average volume
func (v *VolumeAverage) calculateAverage() float64 {
	if len(v.bars) == 0 {
		return 0
	}

	var totalVolume int64
	for _, bar := range v.bars {
		totalVolume += bar.Volume
	}

	return float64(totalVolume) / float64(len(v.bars))
}

// Value returns the current average volume
func (v *VolumeAverage) Value() (float64, error) {
	if !v.ready {
		return 0, fmt.Errorf("volume average not ready: no bars in window")
	}
	return v.calculateAverage(), nil
}

// Reset clears the volume average state
func (v *VolumeAverage) Reset() {
	v.bars = v.bars[:0]
	v.ready = false
	v.processed = 0
}

// IsReady returns true if the volume average has enough data
func (v *VolumeAverage) IsReady() bool {
	return v.ready
}

// WindowSize returns an estimate based on window duration
func (v *VolumeAverage) WindowSize() int {
	minutes := int(v.window.Minutes())
	if minutes < 1 {
		return 1
	}
	return minutes
}

// BarsProcessed returns the number of bars processed
func (v *VolumeAverage) BarsProcessed() int {
	return v.processed
}

// RelativeVolume calculates the current volume relative to average volume
// This stores the last calculated relative volume value
type RelativeVolume struct {
	volumeAvg *VolumeAverage
	name      string
	lastValue float64
	lastBar   *models.Bar1m
	ready     bool
}

// NewRelativeVolume creates a new relative volume calculator
func NewRelativeVolume(avgWindow time.Duration) (*RelativeVolume, error) {
	volumeAvg, err := NewVolumeAverage(avgWindow)
	if err != nil {
		return nil, err
	}

	return &RelativeVolume{
		volumeAvg: volumeAvg,
		name:      fmt.Sprintf("relative_volume_%s", formatDuration(avgWindow)),
		ready:     false,
	}, nil
}

// Name returns the indicator name
func (r *RelativeVolume) Name() string {
	return r.name
}

// Update processes a new bar and updates the relative volume
func (r *RelativeVolume) Update(bar *models.Bar1m) (float64, error) {
	if bar == nil {
		return 0, fmt.Errorf("bar cannot be nil")
	}

	// Update the underlying volume average
	_, err := r.volumeAvg.Update(bar)
	if err != nil {
		return 0, err
	}

	if !r.volumeAvg.IsReady() {
		return 0, nil
	}

	// Calculate relative volume
	avgVol, _ := r.volumeAvg.Value()
	if avgVol == 0 {
		return 0, nil
	}

	r.lastValue = float64(bar.Volume) / avgVol
	r.lastBar = bar
	r.ready = true
	return r.lastValue, nil
}

// Value returns the current relative volume
func (r *RelativeVolume) Value() (float64, error) {
	if !r.ready {
		return 0, fmt.Errorf("relative volume not ready")
	}
	return r.lastValue, nil
}

// Reset clears the relative volume state
func (r *RelativeVolume) Reset() {
	r.volumeAvg.Reset()
	r.lastValue = 0
	r.lastBar = nil
	r.ready = false
}

// IsReady returns true if the relative volume can be calculated
func (r *RelativeVolume) IsReady() bool {
	return r.ready
}
