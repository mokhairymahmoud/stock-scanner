package indicator

import (
	"sync"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

// SymbolState manages indicator state for a single symbol
type SymbolState struct {
	symbol     string
	mu         sync.RWMutex
	calculators map[string]Calculator
	bars       []*models.Bar1m // Rolling window of bars (ring buffer)
	maxBars    int             // Maximum number of bars to keep
	lastUpdate time.Time
}

// NewSymbolState creates a new symbol state with the specified maximum bars
func NewSymbolState(symbol string, maxBars int) *SymbolState {
	return &SymbolState{
		symbol:     symbol,
		calculators: make(map[string]Calculator),
		bars:       make([]*models.Bar1m, 0, maxBars),
		maxBars:    maxBars,
	}
}

// AddCalculator adds a calculator to this symbol's state
func (s *SymbolState) AddCalculator(calc Calculator) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calculators[calc.Name()] = calc
}

// RemoveCalculator removes a calculator from this symbol's state
func (s *SymbolState) RemoveCalculator(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.calculators, name)
}

// Update processes a new bar and updates all calculators
func (s *SymbolState) Update(bar *models.Bar1m) error {
	if bar == nil {
		return nil
	}

	if bar.Symbol != s.symbol {
		return nil // Ignore bars for different symbols
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Add bar to rolling window
	s.bars = append(s.bars, bar)
	if len(s.bars) > s.maxBars {
		// Remove oldest bar (ring buffer behavior)
		copy(s.bars, s.bars[1:])
		s.bars = s.bars[:len(s.bars)-1]
	}

	// Update all calculators
	for _, calc := range s.calculators {
		_, _ = calc.Update(bar)
	}

	s.lastUpdate = time.Now()
	return nil
}

// GetValue retrieves the current value of an indicator
func (s *SymbolState) GetValue(calculatorName string) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	calc, exists := s.calculators[calculatorName]
	if !exists {
		return 0, nil // Return 0 if calculator not found (not an error)
	}

	return calc.Value()
}

// GetAllValues returns all current indicator values
func (s *SymbolState) GetAllValues() map[string]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	values := make(map[string]float64)
	for name, calc := range s.calculators {
		if calc.IsReady() {
			if val, err := calc.Value(); err == nil {
				values[name] = val
			}
		}
	}

	return values
}

// GetBars returns a copy of the current bars window
func (s *SymbolState) GetBars() []*models.Bar1m {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bars := make([]*models.Bar1m, len(s.bars))
	copy(bars, s.bars)
	return bars
}

// GetLastUpdate returns the time of the last update
func (s *SymbolState) GetLastUpdate() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.lastUpdate
}

// Reset clears all state (useful for rehydration)
func (s *SymbolState) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.bars = s.bars[:0]
	for _, calc := range s.calculators {
		calc.Reset()
	}
	s.lastUpdate = time.Time{}
}

// Rehydrate loads historical bars and updates calculators
// This is useful when a worker restarts and needs to rebuild state
func (s *SymbolState) Rehydrate(bars []*models.Bar1m) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Reset state
	s.bars = s.bars[:0]
	for _, calc := range s.calculators {
		calc.Reset()
	}

	// Process bars in order
	for _, bar := range bars {
		if bar.Symbol != s.symbol {
			continue
		}

		// Add to window
		s.bars = append(s.bars, bar)
		if len(s.bars) > s.maxBars {
			copy(s.bars, s.bars[1:])
			s.bars = s.bars[:len(s.bars)-1]
		}

		// Update calculators
		for _, calc := range s.calculators {
			_, _ = calc.Update(bar)
		}
	}

	if len(bars) > 0 {
		s.lastUpdate = bars[len(bars)-1].Timestamp
	}

	return nil
}

