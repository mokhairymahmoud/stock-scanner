package indicator

import (
	"fmt"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

// SMA calculates the Simple Moving Average
// SMA = Sum of prices over period / period
type SMA struct {
	period    int
	name      string
	prices    []float64 // Rolling window of prices
	ready     bool
	processed int
}

// NewSMA creates a new SMA calculator with the specified period
func NewSMA(period int) (*SMA, error) {
	if period < 1 {
		return nil, fmt.Errorf("SMA period must be at least 1, got %d", period)
	}

	return &SMA{
		period:    period,
		name:      fmt.Sprintf("sma_%d", period),
		prices:    make([]float64, 0, period),
		ready:     false,
		processed: 0,
	}, nil
}

// Name returns the indicator name
func (s *SMA) Name() string {
	return s.name
}

// Update processes a new bar and updates the SMA calculation
func (s *SMA) Update(bar *models.Bar1m) (float64, error) {
	if bar == nil {
		return 0, fmt.Errorf("bar cannot be nil")
	}

	price := bar.Close

	// Add price to window
	s.prices = append(s.prices, price)
	s.processed++

	// Remove oldest if we exceed period
	if len(s.prices) > s.period {
		copy(s.prices, s.prices[1:])
		s.prices = s.prices[:len(s.prices)-1]
	}

	// Check if we have enough data
	if len(s.prices) >= s.period {
		s.ready = true
		return s.calculateSMA(), nil
	}

	return 0, nil
}

// calculateSMA computes the SMA value
func (s *SMA) calculateSMA() float64 {
	if len(s.prices) == 0 {
		return 0
	}

	var sum float64
	for _, price := range s.prices {
		sum += price
	}

	return sum / float64(len(s.prices))
}

// Value returns the current SMA value
func (s *SMA) Value() (float64, error) {
	if !s.ready {
		return 0, fmt.Errorf("SMA not ready: need at least %d bars", s.period)
	}
	return s.calculateSMA(), nil
}

// Reset clears the SMA state
func (s *SMA) Reset() {
	s.prices = s.prices[:0]
	s.ready = false
	s.processed = 0
}

// IsReady returns true if the SMA has enough data
func (s *SMA) IsReady() bool {
	return s.ready
}

// WindowSize returns the period (number of bars required)
func (s *SMA) WindowSize() int {
	return s.period
}

// BarsProcessed returns the number of bars processed
func (s *SMA) BarsProcessed() int {
	return s.processed
}
