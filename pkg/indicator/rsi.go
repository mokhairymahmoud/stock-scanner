package indicator

import (
	"fmt"
	"math"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

// RSI calculates the Relative Strength Index
// RSI = 100 - (100 / (1 + RS))
// where RS = Average Gain / Average Loss over the period
type RSI struct {
	period    int
	name      string
	gains     []float64 // Rolling window of gains
	losses    []float64 // Rolling window of losses
	prevClose float64
	ready     bool
	processed int
	avgGain   float64 // Smoothed average gain
	avgLoss   float64 // Smoothed average loss
}

// NewRSI creates a new RSI calculator with the specified period (typically 14)
func NewRSI(period int) (*RSI, error) {
	if period < 2 {
		return nil, fmt.Errorf("RSI period must be at least 2, got %d", period)
	}

	return &RSI{
		period:    period,
		name:      fmt.Sprintf("rsi_%d", period),
		gains:     make([]float64, 0, period),
		losses:    make([]float64, 0, period),
		prevClose: 0,
		ready:     false,
		processed: 0,
	}, nil
}

// Name returns the indicator name
func (r *RSI) Name() string {
	return r.name
}

// Update processes a new bar and updates the RSI calculation
func (r *RSI) Update(bar *models.Bar1m) (float64, error) {
	if bar == nil {
		return 0, fmt.Errorf("bar cannot be nil")
	}

	// First bar: just store the close price
	if r.prevClose == 0 {
		r.prevClose = bar.Close
		r.processed++
		return 0, nil
	}

	// Calculate price change
	change := bar.Close - r.prevClose
	r.prevClose = bar.Close

	var gain, loss float64
	if change > 0 {
		gain = change
		loss = 0
	} else {
		gain = 0
		loss = -change // Loss is positive
	}

	// For the first period, use simple average
	if r.processed < r.period {
		r.gains = append(r.gains, gain)
		r.losses = append(r.losses, loss)
		r.processed++

		// Check if we have enough data
		if r.processed >= r.period {
			// Calculate initial averages
			var sumGain, sumLoss float64
			for i := 0; i < len(r.gains); i++ {
				sumGain += r.gains[i]
				sumLoss += r.losses[i]
			}
			r.avgGain = sumGain / float64(r.period)
			r.avgLoss = sumLoss / float64(r.period)
			r.ready = true
		} else {
			return 0, nil
		}
	} else {
		// Use Wilder's smoothing method (exponential moving average)
		// New Avg = ((Old Avg * (Period - 1)) + New Value) / Period
		r.avgGain = ((r.avgGain * float64(r.period-1)) + gain) / float64(r.period)
		r.avgLoss = ((r.avgLoss * float64(r.period-1)) + loss) / float64(r.period)
		r.processed++
	}

	// Calculate RSI
	if !r.ready {
		return 0, nil
	}

	return r.calculateRSI(), nil
}

// calculateRSI computes the RSI value
func (r *RSI) calculateRSI() float64 {
	if r.avgLoss == 0 {
		return 100.0 // All gains, no losses
	}

	rs := r.avgGain / r.avgLoss
	rsi := 100.0 - (100.0 / (1.0 + rs))

	// Clamp to 0-100 range
	if math.IsNaN(rsi) || math.IsInf(rsi, 0) {
		return 50.0 // Default to neutral
	}

	return math.Max(0.0, math.Min(100.0, rsi))
}

// Value returns the current RSI value
func (r *RSI) Value() (float64, error) {
	if !r.ready {
		return 0, fmt.Errorf("RSI not ready: need at least %d bars", r.period+1)
	}
	return r.calculateRSI(), nil
}

// Reset clears the RSI state
func (r *RSI) Reset() {
	r.gains = r.gains[:0]
	r.losses = r.losses[:0]
	r.prevClose = 0
	r.ready = false
	r.processed = 0
	r.avgGain = 0
	r.avgLoss = 0
}

// IsReady returns true if the RSI has enough data
func (r *RSI) IsReady() bool {
	return r.ready
}

// WindowSize returns the number of bars required (period + 1 for first change)
func (r *RSI) WindowSize() int {
	return r.period + 1
}

// BarsProcessed returns the number of bars processed
func (r *RSI) BarsProcessed() int {
	return r.processed
}
