package bars

import (
	"sync"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

// Aggregator aggregates ticks into 1-minute bars
type Aggregator struct {
	mu          sync.RWMutex
	liveBars    map[string]*models.LiveBar // Map of symbol -> current live bar
	onBarFinal  func(*models.Bar1m)       // Callback when a bar is finalized
	onBarUpdate func(*models.LiveBar)      // Callback when a live bar is updated
}

// NewAggregator creates a new bar aggregator
func NewAggregator() *Aggregator {
	return &Aggregator{
		liveBars: make(map[string]*models.LiveBar),
	}
}

// SetOnBarFinal sets the callback function to be called when a bar is finalized
func (a *Aggregator) SetOnBarFinal(callback func(*models.Bar1m)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onBarFinal = callback
}

// SetOnBarUpdate sets the callback function to be called when a live bar is updated
func (a *Aggregator) SetOnBarUpdate(callback func(*models.LiveBar)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onBarUpdate = callback
}

// ProcessTick processes a tick and updates the corresponding live bar
func (a *Aggregator) ProcessTick(tick *models.Tick) error {
	if tick == nil {
		return nil
	}

	if err := tick.Validate(); err != nil {
		logger.Warn("Invalid tick, skipping",
			logger.ErrorField(err),
			logger.String("symbol", tick.Symbol),
		)
		return err
	}

	// Truncate timestamp to minute boundary
	minuteStart := tick.Timestamp.Truncate(time.Minute)

	a.mu.Lock()
	defer a.mu.Unlock()

	// Get or create live bar for this symbol
	liveBar, exists := a.liveBars[tick.Symbol]

	// Check if we need to finalize the previous bar (minute boundary crossed)
	if exists && !liveBar.Timestamp.Equal(minuteStart) {
		// Minute boundary crossed - finalize the old bar
		finalizedBar := liveBar.ToBar1m()
		if a.onBarFinal != nil {
			// Call callback outside of lock to avoid deadlock
			go a.onBarFinal(finalizedBar)
		}

		logger.Debug("Bar finalized",
			logger.String("symbol", tick.Symbol),
			logger.Time("timestamp", liveBar.Timestamp),
			logger.Float64("open", liveBar.Open),
			logger.Float64("close", liveBar.Close),
			logger.Int64("volume", liveBar.Volume),
		)

		// Create new live bar for the new minute
		liveBar = &models.LiveBar{
			Symbol:    tick.Symbol,
			Timestamp: minuteStart,
		}
		a.liveBars[tick.Symbol] = liveBar
	} else if !exists {
		// First tick for this symbol - create new live bar
		liveBar = &models.LiveBar{
			Symbol:    tick.Symbol,
			Timestamp: minuteStart,
		}
		a.liveBars[tick.Symbol] = liveBar
	}

	// Update the live bar with the tick
	liveBar.Update(tick)

	// Call update callback if set
	if a.onBarUpdate != nil {
		// Create a copy to avoid holding lock during callback
		liveBarCopy := *liveBar
		go a.onBarUpdate(&liveBarCopy)
	}

	return nil
}

// GetLiveBar returns the current live bar for a symbol
func (a *Aggregator) GetLiveBar(symbol string) *models.LiveBar {
	a.mu.RLock()
	defer a.mu.RUnlock()

	liveBar, exists := a.liveBars[symbol]
	if !exists {
		return nil
	}

	// Return a copy to avoid external modification
	liveBarCopy := *liveBar
	return &liveBarCopy
}

// GetAllLiveBars returns all current live bars
func (a *Aggregator) GetAllLiveBars() map[string]*models.LiveBar {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Return a copy of the map
	result := make(map[string]*models.LiveBar, len(a.liveBars))
	for symbol, liveBar := range a.liveBars {
		liveBarCopy := *liveBar
		result[symbol] = &liveBarCopy
	}

	return result
}

// FinalizeBar manually finalizes a bar for a symbol (useful for cleanup or testing)
func (a *Aggregator) FinalizeBar(symbol string) *models.Bar1m {
	a.mu.Lock()
	defer a.mu.Unlock()

	liveBar, exists := a.liveBars[symbol]
	if !exists {
		return nil
	}

	finalizedBar := liveBar.ToBar1m()
	delete(a.liveBars, symbol)

	if a.onBarFinal != nil {
		go a.onBarFinal(finalizedBar)
	}

	return finalizedBar
}

// FinalizeAllBars finalizes all current live bars (useful for shutdown)
func (a *Aggregator) FinalizeAllBars() []*models.Bar1m {
	a.mu.Lock()
	defer a.mu.Unlock()

	finalizedBars := make([]*models.Bar1m, 0, len(a.liveBars))
	for symbol, liveBar := range a.liveBars {
		finalizedBar := liveBar.ToBar1m()
		finalizedBars = append(finalizedBars, finalizedBar)

		if a.onBarFinal != nil {
			go a.onBarFinal(finalizedBar)
		}

		logger.Debug("Bar finalized on shutdown",
			logger.String("symbol", symbol),
			logger.Time("timestamp", liveBar.Timestamp),
		)
	}

	// Clear all live bars
	a.liveBars = make(map[string]*models.LiveBar)

	return finalizedBars
}

// GetSymbolCount returns the number of symbols with active live bars
func (a *Aggregator) GetSymbolCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.liveBars)
}

