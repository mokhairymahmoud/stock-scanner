package indicator

import (
	"context"
	"fmt"
	"sync"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	indicatorpkg "github.com/mohamedkhairy/stock-scanner/pkg/indicator"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

// CalculatorFactory is a function that creates a new calculator instance
type CalculatorFactory func() (indicatorpkg.Calculator, error)

// OnIndicatorsUpdated is a callback function called after indicators are updated
type OnIndicatorsUpdated func(symbol string, indicators map[string]float64)

// Engine processes finalized bars and computes indicators
type Engine struct {
	indicatorRegistry   *IndicatorRegistry // Registry of all available indicators
	requiredIndicators  map[string]bool    // Set of required indicator names (empty = all)
	symbolStates        map[string]*indicatorpkg.SymbolState
	onIndicatorsUpdated OnIndicatorsUpdated // Callback after indicators are updated
	mu                  sync.RWMutex
	ctx                 context.Context
	cancel              context.CancelFunc
	maxBars             int // Maximum bars to keep per symbol
}

// EngineConfig holds configuration for the indicator engine
type EngineConfig struct {
	MaxBars int // Maximum number of bars to keep per symbol (default: 200)
}

// DefaultEngineConfig returns default configuration
func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		MaxBars: 200, // Keep last 200 bars (about 3+ hours of 1-minute bars)
	}
}

// NewEngine creates a new indicator engine
func NewEngine(config EngineConfig, registry *IndicatorRegistry) *Engine {
	ctx, cancel := context.WithCancel(context.Background())

	return &Engine{
		indicatorRegistry:  registry,
		requiredIndicators: make(map[string]bool), // Empty = all indicators
		symbolStates:       make(map[string]*indicatorpkg.SymbolState),
		ctx:                ctx,
		cancel:             cancel,
		maxBars:            config.MaxBars,
	}
}

// SetRequiredIndicators sets which indicators should be computed
// If empty map, all indicators will be computed
func (e *Engine) SetRequiredIndicators(required map[string]bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.requiredIndicators = required
}

// GetRequiredIndicators returns the set of required indicators
func (e *Engine) GetRequiredIndicators() map[string]bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make(map[string]bool)
	for name, required := range e.requiredIndicators {
		result[name] = required
	}
	return result
}

// ProcessBar processes a finalized bar and updates indicators
func (e *Engine) ProcessBar(bar *models.Bar1m) error {
	if bar == nil {
		return fmt.Errorf("bar cannot be nil")
	}

	if err := bar.Validate(); err != nil {
		return fmt.Errorf("invalid bar: %w", err)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Get or create symbol state
	state, exists := e.symbolStates[bar.Symbol]
	if !exists {
		state = indicatorpkg.NewSymbolState(bar.Symbol, e.maxBars)
		e.symbolStates[bar.Symbol] = state

		// Create calculator instances for this symbol using registry
		// If requiredIndicators is empty, create all indicators
		// Otherwise, only create required ones
		allIndicators := len(e.requiredIndicators) == 0
		availableIndicators := e.indicatorRegistry.ListAvailable()

		for _, name := range availableIndicators {
			// Skip if we have required indicators and this one is not required
			if !allIndicators && !e.requiredIndicators[name] {
				continue
			}

			factory, exists := e.indicatorRegistry.GetFactory(name)
			if !exists {
				continue
			}

			calc, err := factory()
			if err != nil {
				logger.Warn("Failed to create calculator",
					logger.String("name", name),
					logger.String("symbol", bar.Symbol),
					logger.ErrorField(err),
				)
				continue
			}
			state.AddCalculator(calc)
		}
	}

	// Update symbol state with the new bar
	err := state.Update(bar)
	if err != nil {
		return err
	}

	// Get updated indicators and call callback
	if e.onIndicatorsUpdated != nil {
		indicators := state.GetAllValues()
		if len(indicators) > 0 {
			e.onIndicatorsUpdated(bar.Symbol, indicators)
		}
	}

	return nil
}

// GetIndicators returns all indicator values for a symbol
func (e *Engine) GetIndicators(symbol string) (map[string]float64, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	state, exists := e.symbolStates[symbol]
	if !exists {
		return nil, fmt.Errorf("symbol %s not found", symbol)
	}

	return state.GetAllValues(), nil
}

// GetAllSymbols returns a list of all symbols being tracked
func (e *Engine) GetAllSymbols() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	symbols := make([]string, 0, len(e.symbolStates))
	for symbol := range e.symbolStates {
		symbols = append(symbols, symbol)
	}
	return symbols
}

// GetSymbolCount returns the number of symbols being tracked
func (e *Engine) GetSymbolCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.symbolStates)
}

// Stop stops the engine
func (e *Engine) Stop() {
	e.cancel()
}

// Context returns the engine's context
func (e *Engine) Context() context.Context {
	return e.ctx
}

// SetOnIndicatorsUpdated sets the callback function called after indicators are updated
func (e *Engine) SetOnIndicatorsUpdated(callback OnIndicatorsUpdated) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onIndicatorsUpdated = callback
}
