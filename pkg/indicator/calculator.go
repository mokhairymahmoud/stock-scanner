package indicator

import (
	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

// Calculator is the interface for computing technical indicators
// Each indicator type implements this interface
type Calculator interface {
	// Name returns the unique name of this indicator (e.g., "rsi_14", "ema_20")
	Name() string

	// Update processes a new bar and updates the indicator state
	// Returns the new indicator value, or nil if not enough data
	Update(bar *models.Bar1m) (float64, error)

	// Value returns the current indicator value
	// Returns 0 and error if not enough data has been processed
	Value() (float64, error)

	// Reset clears the indicator state (useful for rehydration or testing)
	Reset()

	// IsReady returns true if the indicator has enough data to produce a valid value
	IsReady() bool
}

// WindowedCalculator extends Calculator for indicators that require a window of bars
type WindowedCalculator interface {
	Calculator

	// WindowSize returns the number of bars required for this indicator
	WindowSize() int

	// BarsProcessed returns the number of bars processed so far
	BarsProcessed() int
}
