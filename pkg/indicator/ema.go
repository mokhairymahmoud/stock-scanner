package indicator

import (
	"fmt"
	"math"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

// EMA calculates the Exponential Moving Average
// EMA = (Price - Previous EMA) * Multiplier + Previous EMA
// Multiplier = 2 / (Period + 1)
type EMA struct {
	period    int
	name      string
	multiplier float64
	value     float64
	ready     bool
	processed int
}

// NewEMA creates a new EMA calculator with the specified period
func NewEMA(period int) (*EMA, error) {
	if period < 1 {
		return nil, fmt.Errorf("EMA period must be at least 1, got %d", period)
	}

	multiplier := 2.0 / float64(period+1)

	return &EMA{
		period:    period,
		name:      fmt.Sprintf("ema_%d", period),
		multiplier: multiplier,
		value:     0,
		ready:     false,
		processed: 0,
	}, nil
}

// Name returns the indicator name
func (e *EMA) Name() string {
	return e.name
}

// Update processes a new bar and updates the EMA calculation
func (e *EMA) Update(bar *models.Bar1m) (float64, error) {
	if bar == nil {
		return 0, fmt.Errorf("bar cannot be nil")
	}

	price := bar.Close

	// For the first bar, EMA = price
	if !e.ready {
		e.value = price
		e.ready = true
		e.processed++
		return e.value, nil
	}

	// EMA calculation: (Price - Previous EMA) * Multiplier + Previous EMA
	e.value = (price-e.value)*e.multiplier + e.value
	e.processed++

	// Handle NaN/Inf
	if math.IsNaN(e.value) || math.IsInf(e.value, 0) {
		e.value = price // Fallback to current price
	}

	return e.value, nil
}

// Value returns the current EMA value
func (e *EMA) Value() (float64, error) {
	if !e.ready {
		return 0, fmt.Errorf("EMA not ready: need at least 1 bar")
	}
	return e.value, nil
}

// Reset clears the EMA state
func (e *EMA) Reset() {
	e.value = 0
	e.ready = false
	e.processed = 0
}

// IsReady returns true if the EMA has enough data
func (e *EMA) IsReady() bool {
	return e.ready
}

// WindowSize returns 1 (EMA can start immediately)
func (e *EMA) WindowSize() int {
	return 1
}

// BarsProcessed returns the number of bars processed
func (e *EMA) BarsProcessed() int {
	return e.processed
}

