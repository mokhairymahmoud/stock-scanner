package scanner

import (
	"sync"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

// SymbolState represents the current state of a symbol for scanning
// This is different from models.SymbolState - this is the scanner's internal state
type SymbolState struct {
	Symbol        string
	LiveBar       *models.LiveBar
	LastFinalBars []*models.Bar1m // Ring buffer of recent finalized bars
	Indicators    map[string]float64
	LastTickTime  time.Time
	LastUpdate    time.Time
	mu            sync.RWMutex
}

// StateManager manages symbol states for the scanner
type StateManager struct {
	states     map[string]*SymbolState
	mu         sync.RWMutex
	maxFinalBars int // Maximum number of finalized bars to keep per symbol
}

// NewStateManager creates a new state manager
func NewStateManager(maxFinalBars int) *StateManager {
	if maxFinalBars <= 0 {
		maxFinalBars = 200 // Default: keep last 200 finalized bars (about 3+ hours)
	}

	return &StateManager{
		states:       make(map[string]*SymbolState),
		maxFinalBars: maxFinalBars,
	}
}

// GetOrCreateState gets an existing symbol state or creates a new one
func (sm *StateManager) GetOrCreateState(symbol string) *SymbolState {
	sm.mu.RLock()
	state, exists := sm.states[symbol]
	sm.mu.RUnlock()

	if exists {
		return state
	}

	// Create new state
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Double-check after acquiring write lock
	if state, exists := sm.states[symbol]; exists {
		return state
	}

	state = &SymbolState{
		Symbol:        symbol,
		LastFinalBars: make([]*models.Bar1m, 0, sm.maxFinalBars),
		Indicators:    make(map[string]float64),
	}

	sm.states[symbol] = state
	return state
}

// GetState gets a symbol state (returns nil if not found)
func (sm *StateManager) GetState(symbol string) *SymbolState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.states[symbol]
}

// UpdateLiveBar updates the live bar for a symbol with a tick
func (sm *StateManager) UpdateLiveBar(symbol string, tick *models.Tick) error {
	if tick == nil {
		return nil
	}

	state := sm.GetOrCreateState(symbol)

	state.mu.Lock()
	defer state.mu.Unlock()

	// Initialize live bar if needed
	// Use tick's timestamp to determine which minute it belongs to
	tickTime := tick.Timestamp
	minuteStart := time.Date(tickTime.Year(), tickTime.Month(), tickTime.Day(), tickTime.Hour(), tickTime.Minute(), 0, 0, tickTime.Location())

	if state.LiveBar == nil || !state.LiveBar.Timestamp.Equal(minuteStart) {
		// New minute - create new live bar
		state.LiveBar = &models.LiveBar{
			Symbol:    symbol,
			Timestamp: minuteStart,
		}
	}

	// Update live bar with tick
	state.LiveBar.Update(tick)
	state.LastTickTime = tick.Timestamp
	state.LastUpdate = time.Now()

	return nil
}

// UpdateFinalizedBar adds a finalized bar to the symbol state
func (sm *StateManager) UpdateFinalizedBar(bar *models.Bar1m) error {
	if bar == nil {
		return nil
	}

	state := sm.GetOrCreateState(bar.Symbol)

	state.mu.Lock()
	defer state.mu.Unlock()

	// Add to ring buffer
	state.LastFinalBars = append(state.LastFinalBars, bar)
	if len(state.LastFinalBars) > sm.maxFinalBars {
		// Remove oldest bar (ring buffer behavior)
		copy(state.LastFinalBars, state.LastFinalBars[1:])
		state.LastFinalBars = state.LastFinalBars[:len(state.LastFinalBars)-1]
	}

	// Clear live bar if it matches this finalized bar's timestamp
	if state.LiveBar != nil && state.LiveBar.Timestamp.Equal(bar.Timestamp) {
		state.LiveBar = nil
	}

	state.LastUpdate = time.Now()

	return nil
}

// UpdateIndicators updates indicator values for a symbol
func (sm *StateManager) UpdateIndicators(symbol string, indicators map[string]float64) error {
	state := sm.GetOrCreateState(symbol)

	state.mu.Lock()
	defer state.mu.Unlock()

	// Update indicators
	if state.Indicators == nil {
		state.Indicators = make(map[string]float64)
	}

	for key, value := range indicators {
		state.Indicators[key] = value
	}

	state.LastUpdate = time.Now()

	return nil
}

// GetMetrics returns a snapshot of all metrics for a symbol (for rule evaluation)
func (sm *StateManager) GetMetrics(symbol string) map[string]float64 {
	state := sm.GetState(symbol)
	if state == nil {
		return make(map[string]float64)
	}

	state.mu.RLock()
	defer state.mu.RUnlock()

	metrics := make(map[string]float64)

	// Copy indicators
	for key, value := range state.Indicators {
		metrics[key] = value
	}

	// Add computed metrics from live bar
	if state.LiveBar != nil {
		// Current price (from live bar close)
		metrics["price"] = state.LiveBar.Close

		// VWAP from live bar
		if state.LiveBar.VWAPDenom > 0 {
			metrics["vwap_live"] = state.LiveBar.VWAPNum / state.LiveBar.VWAPDenom
		}

		// Volume from live bar
		metrics["volume_live"] = float64(state.LiveBar.Volume)
	}

	// Add metrics from last finalized bar if available
	if len(state.LastFinalBars) > 0 {
		lastBar := state.LastFinalBars[len(state.LastFinalBars)-1]
		metrics["close"] = lastBar.Close
		metrics["open"] = lastBar.Open
		metrics["high"] = lastBar.High
		metrics["low"] = lastBar.Low
		metrics["volume"] = float64(lastBar.Volume)
		metrics["vwap"] = lastBar.VWAP
	}

	// Compute price change metrics from finalized bars
	if len(state.LastFinalBars) >= 2 {
		currentBar := state.LastFinalBars[len(state.LastFinalBars)-1]
		prevBar := state.LastFinalBars[len(state.LastFinalBars)-2]

		if prevBar.Close > 0 {
			changePct := ((currentBar.Close - prevBar.Close) / prevBar.Close) * 100.0
			metrics["price_change_1m_pct"] = changePct
		}
	}

	// Compute price change over 5 minutes (if we have enough bars)
	if len(state.LastFinalBars) >= 6 {
		currentBar := state.LastFinalBars[len(state.LastFinalBars)-1]
		bar5m := state.LastFinalBars[len(state.LastFinalBars)-6]

		if bar5m.Close > 0 {
			changePct := ((currentBar.Close - bar5m.Close) / bar5m.Close) * 100.0
			metrics["price_change_5m_pct"] = changePct
		}
	}

	// Compute price change over 15 minutes (if we have enough bars)
	if len(state.LastFinalBars) >= 16 {
		currentBar := state.LastFinalBars[len(state.LastFinalBars)-1]
		bar15m := state.LastFinalBars[len(state.LastFinalBars)-16]

		if bar15m.Close > 0 {
			changePct := ((currentBar.Close - bar15m.Close) / bar15m.Close) * 100.0
			metrics["price_change_15m_pct"] = changePct
		}
	}

	return metrics
}

// Snapshot creates a snapshot of all symbol states for scanning
// This allows the scan loop to iterate without holding locks
type StateSnapshot struct {
	Symbols []string
	States  map[string]*SymbolStateSnapshot
}

// SymbolStateSnapshot is a snapshot of a single symbol's state
type SymbolStateSnapshot struct {
	Symbol        string
	LiveBar       *models.LiveBar
	LastFinalBars []*models.Bar1m
	Indicators    map[string]float64
	LastTickTime  time.Time
	LastUpdate    time.Time
}

// Snapshot creates a snapshot of all states
func (sm *StateManager) Snapshot() *StateSnapshot {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	snapshot := &StateSnapshot{
		Symbols: make([]string, 0, len(sm.states)),
		States:  make(map[string]*SymbolStateSnapshot),
	}

	for symbol, state := range sm.states {
		state.mu.RLock()

		// Create snapshot of this symbol's state
		symbolSnapshot := &SymbolStateSnapshot{
			Symbol:        symbol,
			LastTickTime:  state.LastTickTime,
			LastUpdate:    state.LastUpdate,
		}

		// Copy live bar
		if state.LiveBar != nil {
			liveBarCopy := *state.LiveBar
			symbolSnapshot.LiveBar = &liveBarCopy
		}

		// Copy finalized bars
		symbolSnapshot.LastFinalBars = make([]*models.Bar1m, len(state.LastFinalBars))
		for i, bar := range state.LastFinalBars {
			barCopy := *bar
			symbolSnapshot.LastFinalBars[i] = &barCopy
		}

		// Copy indicators
		symbolSnapshot.Indicators = make(map[string]float64)
		for key, value := range state.Indicators {
			symbolSnapshot.Indicators[key] = value
		}

		state.mu.RUnlock()

		snapshot.Symbols = append(snapshot.Symbols, symbol)
		snapshot.States[symbol] = symbolSnapshot
	}

	return snapshot
}

// GetSymbolCount returns the number of symbols in the state manager
func (sm *StateManager) GetSymbolCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return len(sm.states)
}

// RemoveSymbol removes a symbol from the state manager
func (sm *StateManager) RemoveSymbol(symbol string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.states, symbol)
}

// Clear removes all symbols from the state manager
func (sm *StateManager) Clear() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.states = make(map[string]*SymbolState)
}

