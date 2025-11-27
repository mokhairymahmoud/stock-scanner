package scanner

import (
	"sync"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/metrics"
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

	// Session tracking
	CurrentSession MarketSession
	SessionStartTime time.Time

	// Price references
	YesterdayClose float64 // Yesterday's closing price
	TodayOpen      float64 // Today's opening price
	TodayClose     float64 // Today's closing price (set at market close)

	// Session-specific volume tracking
	PremarketVolume int64 // Volume traded during pre-market session
	MarketVolume    int64 // Volume traded during market session
	PostmarketVolume int64 // Volume traded during post-market session

	// Trade count tracking
	TradeCount      int64 // Total trade count (incremented on each tick)
	TradeCountHistory []int64 // Ring buffer for timeframe-based trade counts

	// Candle direction tracking (for consecutive candles filter)
	// Map of timeframe -> direction history (true = green/up, false = red/down)
	CandleDirections map[string][]bool // timeframe -> []bool

	// Metric caching for performance optimization
	// Cache computed metrics with invalidation timestamp
	cachedMetrics     map[string]float64
	cacheTimestamp    time.Time
	cacheInvalidation time.Time // When cache should be invalidated
}

// StateManager manages symbol states for the scanner
type StateManager struct {
	states        map[string]*SymbolState
	mu            sync.RWMutex
	maxFinalBars  int // Maximum number of finalized bars to keep per symbol
	metricRegistry *metrics.Registry // Metric registry for computing metrics
}

// NewStateManager creates a new state manager
func NewStateManager(maxFinalBars int) *StateManager {
	if maxFinalBars <= 0 {
		maxFinalBars = 200 // Default: keep last 200 finalized bars (about 3+ hours)
	}

	return &StateManager{
		states:         make(map[string]*SymbolState),
		maxFinalBars:   maxFinalBars,
		metricRegistry: metrics.NewRegistry(),
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
		Symbol:           symbol,
		LastFinalBars:    make([]*models.Bar1m, 0, sm.maxFinalBars),
		Indicators:       make(map[string]float64),
		CurrentSession:   SessionClosed,
		TradeCountHistory: make([]int64, 0),
		CandleDirections: make(map[string][]bool),
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

	// Check and update session
	newSession := GetMarketSession(tick.Timestamp)
	if newSession != state.CurrentSession {
		// Session changed - reset session-specific data if needed
		sm.handleSessionTransition(state, newSession, tick.Timestamp)
	}

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

	// Invalidate metric cache (data has changed)
	// Note: We invalidate on every tick update, but cache can still help
	// when multiple rules need the same metrics in a single scan cycle
	state.invalidateMetricCache()

	// Increment trade count
	state.TradeCount++

	// Update session-specific volume
	sm.updateSessionVolume(state, tick.Size, newSession)

	return nil
}

// handleSessionTransition handles transitions between market sessions
func (sm *StateManager) handleSessionTransition(state *SymbolState, newSession MarketSession, t time.Time) {
	oldSession := state.CurrentSession
	state.CurrentSession = newSession
	state.SessionStartTime = t

	// Reset session-specific volumes when transitioning to a new session
	// (except when transitioning from premarket to market - keep premarket volume)
	if newSession == SessionMarket && oldSession == SessionPreMarket {
		// Keep premarket volume, reset market volume
		state.MarketVolume = 0
	} else if newSession == SessionPostMarket && oldSession == SessionMarket {
		// Keep market volume, reset postmarket volume
		state.PostmarketVolume = 0
	} else if newSession == SessionPreMarket {
		// Reset all volumes when starting premarket (new day)
		state.PremarketVolume = 0
		state.MarketVolume = 0
		state.PostmarketVolume = 0
		// Store yesterday's close and today's open
		if state.TodayClose > 0 {
			state.YesterdayClose = state.TodayClose
		}
		// Today's open will be set when first bar is finalized
	}

	// Reset trade count at start of each session
	state.TradeCount = 0
}

// updateSessionVolume updates the appropriate session-specific volume counter
func (sm *StateManager) updateSessionVolume(state *SymbolState, volume int64, session MarketSession) {
	switch session {
	case SessionPreMarket:
		state.PremarketVolume += volume
	case SessionMarket:
		state.MarketVolume += volume
	case SessionPostMarket:
		state.PostmarketVolume += volume
	}
}

// UpdateFinalizedBar adds a finalized bar to the symbol state
func (sm *StateManager) UpdateFinalizedBar(bar *models.Bar1m) error {
	if bar == nil {
		return nil
	}

	state := sm.GetOrCreateState(bar.Symbol)

	state.mu.Lock()
	defer state.mu.Unlock()

	// Check and update session
	newSession := GetMarketSession(bar.Timestamp)
	if newSession != state.CurrentSession {
		sm.handleSessionTransition(state, newSession, bar.Timestamp)
	}

	// Track today's open (first bar of the day)
	if state.TodayOpen == 0 {
		state.TodayOpen = bar.Open
	}

	// Track today's close (last bar before market close)
	if newSession == SessionMarket || (newSession == SessionPostMarket && state.TodayClose == 0) {
		state.TodayClose = bar.Close
	}

	// Track candle direction (green = close > open, red = close < open)
	isGreen := bar.Close > bar.Open
	sm.updateCandleDirection(state, "1m", isGreen)

	// Store trade count for this bar in history
	// TradeCount represents trades that occurred during this bar's timeframe
	if state.TradeCountHistory == nil {
		state.TradeCountHistory = make([]int64, 0, sm.maxFinalBars)
	}
	state.TradeCountHistory = append(state.TradeCountHistory, state.TradeCount)
	// Keep only last maxFinalBars entries (ring buffer behavior)
	if len(state.TradeCountHistory) > sm.maxFinalBars {
		copy(state.TradeCountHistory, state.TradeCountHistory[1:])
		state.TradeCountHistory = state.TradeCountHistory[:len(state.TradeCountHistory)-1]
	}
	// Reset trade count for next bar (will be incremented on next tick)
	state.TradeCount = 0

	// Invalidate metric cache (data has changed)
	state.invalidateMetricCache()

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

// updateCandleDirection updates the candle direction history for a timeframe
func (sm *StateManager) updateCandleDirection(state *SymbolState, timeframe string, isGreen bool) {
	if state.CandleDirections == nil {
		state.CandleDirections = make(map[string][]bool)
	}

	directions, exists := state.CandleDirections[timeframe]
	if !exists {
		directions = make([]bool, 0, 100) // Pre-allocate for efficiency
	}

	// Add new direction
	directions = append(directions, isGreen)

	// Keep only last 100 candles (adjust based on needs)
	maxCandles := 100
	if len(directions) > maxCandles {
		directions = directions[len(directions)-maxCandles:]
	}

	state.CandleDirections[timeframe] = directions
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
// This method uses the metric registry to compute metrics consistently
func (sm *StateManager) GetMetrics(symbol string) map[string]float64 {
	state := sm.GetState(symbol)
	if state == nil {
		return make(map[string]float64)
	}

	state.mu.RLock()
	defer state.mu.RUnlock()

	// Convert state to metric snapshot
	metricSnapshot := &metrics.SymbolStateSnapshot{
		Symbol:           symbol,
		LiveBar:          state.LiveBar,
		LastFinalBars:    state.LastFinalBars,
		Indicators:       state.Indicators,
		LastTickTime:     state.LastTickTime,
		LastUpdate:       state.LastUpdate,
		CurrentSession:   string(state.CurrentSession),
		SessionStartTime: state.SessionStartTime,
		YesterdayClose:   state.YesterdayClose,
		TodayOpen:        state.TodayOpen,
		TodayClose:       state.TodayClose,
		PremarketVolume:  state.PremarketVolume,
		MarketVolume:     state.MarketVolume,
		PostmarketVolume: state.PostmarketVolume,
		TradeCount:       state.TradeCount,
	}

	// Copy trade count history
	if len(state.TradeCountHistory) > 0 {
		metricSnapshot.TradeCountHistory = make([]int64, len(state.TradeCountHistory))
		copy(metricSnapshot.TradeCountHistory, state.TradeCountHistory)
	}

	// Copy candle directions
	if len(state.CandleDirections) > 0 {
		metricSnapshot.CandleDirections = make(map[string][]bool)
		for k, v := range state.CandleDirections {
			directions := make([]bool, len(v))
			copy(directions, v)
			metricSnapshot.CandleDirections[k] = directions
		}
	}

	// Use metric registry to compute all metrics
	return sm.metricRegistry.ComputeAll(metricSnapshot)
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

	// Session tracking
	CurrentSession MarketSession
	SessionStartTime time.Time

	// Price references
	YesterdayClose float64
	TodayOpen      float64
	TodayClose     float64

	// Session-specific volume tracking
	PremarketVolume int64
	MarketVolume    int64
	PostmarketVolume int64

	// Trade count tracking
	TradeCount      int64
	TradeCountHistory []int64

	// Candle direction tracking
	CandleDirections map[string][]bool
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
			Symbol:           symbol,
			LastTickTime:     state.LastTickTime,
			LastUpdate:       state.LastUpdate,
			CurrentSession:   state.CurrentSession,
			SessionStartTime: state.SessionStartTime,
			YesterdayClose:   state.YesterdayClose,
			TodayOpen:        state.TodayOpen,
			TodayClose:       state.TodayClose,
			PremarketVolume:  state.PremarketVolume,
			MarketVolume:     state.MarketVolume,
			PostmarketVolume: state.PostmarketVolume,
			TradeCount:       state.TradeCount,
		}

		// Copy trade count history
		if len(state.TradeCountHistory) > 0 {
			symbolSnapshot.TradeCountHistory = make([]int64, len(state.TradeCountHistory))
			copy(symbolSnapshot.TradeCountHistory, state.TradeCountHistory)
		}

		// Copy candle directions
		if len(state.CandleDirections) > 0 {
			symbolSnapshot.CandleDirections = make(map[string][]bool)
			for k, v := range state.CandleDirections {
				directions := make([]bool, len(v))
				copy(directions, v)
				symbolSnapshot.CandleDirections[k] = directions
			}
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

