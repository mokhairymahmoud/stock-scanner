package scanner

import (
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

func TestStateManager_GetOrCreateState(t *testing.T) {
	sm := NewStateManager(10)

	// Get non-existent state (should create)
	state := sm.GetOrCreateState("AAPL")
	if state == nil {
		t.Fatal("Expected state to be created")
	}

	if state.Symbol != "AAPL" {
		t.Errorf("Expected symbol 'AAPL', got '%s'", state.Symbol)
	}

	// Get existing state (should return same)
	state2 := sm.GetOrCreateState("AAPL")
	if state != state2 {
		t.Error("Expected same state instance for same symbol")
	}
}

func TestStateManager_UpdateLiveBar(t *testing.T) {
	sm := NewStateManager(10)

	tick := &models.Tick{
		Symbol:    "AAPL",
		Price:     150.0,
		Size:      100,
		Timestamp: time.Now(),
		Type:      "trade",
	}

	err := sm.UpdateLiveBar("AAPL", tick)
	if err != nil {
		t.Fatalf("UpdateLiveBar() error = %v", err)
	}

	state := sm.GetState("AAPL")
	if state == nil {
		t.Fatal("Expected state to exist")
	}

	state.mu.RLock()
	liveBar := state.LiveBar
	state.mu.RUnlock()

	if liveBar == nil {
		t.Fatal("Expected live bar to be created")
	}

	if liveBar.Close != 150.0 {
		t.Errorf("Expected close price 150.0, got %f", liveBar.Close)
	}

	if liveBar.Volume != 100 {
		t.Errorf("Expected volume 100, got %d", liveBar.Volume)
	}
}

func TestStateManager_UpdateLiveBar_NewMinute(t *testing.T) {
	sm := NewStateManager(10)

	now := time.Now()
	minuteStart := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, now.Location())

	// First tick in minute
	tick1 := &models.Tick{
		Symbol:    "AAPL",
		Price:     150.0,
		Size:      100,
		Timestamp: minuteStart.Add(10 * time.Second),
		Type:      "trade",
	}

	err := sm.UpdateLiveBar("AAPL", tick1)
	if err != nil {
		t.Fatalf("UpdateLiveBar() error = %v", err)
	}

	// Second tick in next minute
	tick2 := &models.Tick{
		Symbol:    "AAPL",
		Price:     151.0,
		Size:      200,
		Timestamp: minuteStart.Add(1*time.Minute + 10*time.Second),
		Type:      "trade",
	}

	err = sm.UpdateLiveBar("AAPL", tick2)
	if err != nil {
		t.Fatalf("UpdateLiveBar() error = %v", err)
	}

	state := sm.GetState("AAPL")
	state.mu.RLock()
	liveBar := state.LiveBar
	state.mu.RUnlock()

	if liveBar == nil {
		t.Fatal("Expected live bar to exist")
	}

	// Should be new minute's bar
	if liveBar.Close != 151.0 {
		t.Errorf("Expected close price 151.0, got %f", liveBar.Close)
	}

	if liveBar.Volume != 200 {
		t.Errorf("Expected volume 200, got %d", liveBar.Volume)
	}
}

func TestStateManager_UpdateFinalizedBar(t *testing.T) {
	sm := NewStateManager(10)

	bar := &models.Bar1m{
		Symbol:    "AAPL",
		Timestamp: time.Now(),
		Open:      150.0,
		High:      152.0,
		Low:       149.0,
		Close:     151.0,
		Volume:    1000,
		VWAP:      150.5,
	}

	err := sm.UpdateFinalizedBar(bar)
	if err != nil {
		t.Fatalf("UpdateFinalizedBar() error = %v", err)
	}

	state := sm.GetState("AAPL")
	if state == nil {
		t.Fatal("Expected state to exist")
	}

	state.mu.RLock()
	finalBars := state.LastFinalBars
	state.mu.RUnlock()

	if len(finalBars) != 1 {
		t.Errorf("Expected 1 finalized bar, got %d", len(finalBars))
	}

	if finalBars[0].Close != 151.0 {
		t.Errorf("Expected close price 151.0, got %f", finalBars[0].Close)
	}
}

func TestStateManager_UpdateFinalizedBar_RingBuffer(t *testing.T) {
	sm := NewStateManager(3) // Keep only 3 bars

	// Add 5 bars
	for i := 0; i < 5; i++ {
		bar := &models.Bar1m{
			Symbol:    "AAPL",
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			Close:     float64(150 + i),
			Volume:    1000,
		}

		err := sm.UpdateFinalizedBar(bar)
		if err != nil {
			t.Fatalf("UpdateFinalizedBar() error = %v", err)
		}
	}

	state := sm.GetState("AAPL")
	state.mu.RLock()
	finalBars := state.LastFinalBars
	state.mu.RUnlock()

	// Should only keep last 3 bars
	if len(finalBars) != 3 {
		t.Errorf("Expected 3 finalized bars, got %d", len(finalBars))
	}

	// Should have bars 2, 3, 4 (indices 0, 1, 2)
	if finalBars[0].Close != 152.0 {
		t.Errorf("Expected first bar close 152.0, got %f", finalBars[0].Close)
	}

	if finalBars[2].Close != 154.0 {
		t.Errorf("Expected last bar close 154.0, got %f", finalBars[2].Close)
	}
}

func TestStateManager_UpdateIndicators(t *testing.T) {
	sm := NewStateManager(10)

	indicators := map[string]float64{
		"rsi_14":  65.5,
		"ema_20":  150.2,
		"vwap_5m": 149.8,
	}

	err := sm.UpdateIndicators("AAPL", indicators)
	if err != nil {
		t.Fatalf("UpdateIndicators() error = %v", err)
	}

	state := sm.GetState("AAPL")
	if state == nil {
		t.Fatal("Expected state to exist")
	}

	state.mu.RLock()
	stateIndicators := state.Indicators
	state.mu.RUnlock()

	if len(stateIndicators) != 3 {
		t.Errorf("Expected 3 indicators, got %d", len(stateIndicators))
	}

	if stateIndicators["rsi_14"] != 65.5 {
		t.Errorf("Expected rsi_14 = 65.5, got %f", stateIndicators["rsi_14"])
	}
}

func TestStateManager_GetMetrics(t *testing.T) {
	sm := NewStateManager(10)

	// Add finalized bars
	for i := 0; i < 5; i++ {
		bar := &models.Bar1m{
			Symbol:    "AAPL",
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			Open:      float64(150 + i),
			High:      float64(152 + i),
			Low:       float64(149 + i),
			Close:     float64(151 + i),
			Volume:    1000,
			VWAP:      150.5 + float64(i),
		}

		sm.UpdateFinalizedBar(bar)
	}

	// Add indicators
	indicators := map[string]float64{
		"rsi_14":  65.5,
		"ema_20":  150.2,
	}
	sm.UpdateIndicators("AAPL", indicators)

	// Add live bar
	tick := &models.Tick{
		Symbol:    "AAPL",
		Price:     156.0,
		Size:      200,
		Timestamp: time.Now(),
		Type:      "trade",
	}
	sm.UpdateLiveBar("AAPL", tick)

	metrics := sm.GetMetrics("AAPL")

	// Check indicators
	if metrics["rsi_14"] != 65.5 {
		t.Errorf("Expected rsi_14 = 65.5, got %f", metrics["rsi_14"])
	}

	// Check price from live bar
	if metrics["price"] != 156.0 {
		t.Errorf("Expected price = 156.0, got %f", metrics["price"])
	}

	// Check close from last finalized bar
	if metrics["close"] != 155.0 {
		t.Errorf("Expected close = 155.0, got %f", metrics["close"])
	}

	// Check price change 1m
	if metrics["price_change_1m_pct"] == 0 {
		t.Error("Expected price_change_1m_pct to be computed")
	}
}

func TestStateManager_Snapshot(t *testing.T) {
	sm := NewStateManager(10)

	// Add state for multiple symbols
	for _, symbol := range []string{"AAPL", "GOOGL", "MSFT"} {
		tick := &models.Tick{
			Symbol:    symbol,
			Price:     150.0,
			Size:      100,
			Timestamp: time.Now(),
			Type:      "trade",
		}
		sm.UpdateLiveBar(symbol, tick)

		indicators := map[string]float64{
			"rsi_14": 65.5,
		}
		sm.UpdateIndicators(symbol, indicators)
	}

	snapshot := sm.Snapshot()

	if len(snapshot.Symbols) != 3 {
		t.Errorf("Expected 3 symbols in snapshot, got %d", len(snapshot.Symbols))
	}

	if len(snapshot.States) != 3 {
		t.Errorf("Expected 3 states in snapshot, got %d", len(snapshot.States))
	}

	// Verify snapshot data
	aaplSnapshot := snapshot.States["AAPL"]
	if aaplSnapshot == nil {
		t.Fatal("Expected AAPL state in snapshot")
	}

	if aaplSnapshot.LiveBar == nil {
		t.Error("Expected live bar in snapshot")
	}

	if aaplSnapshot.Indicators["rsi_14"] != 65.5 {
		t.Errorf("Expected rsi_14 = 65.5 in snapshot, got %f", aaplSnapshot.Indicators["rsi_14"])
	}
}

func TestStateManager_GetSymbolCount(t *testing.T) {
	sm := NewStateManager(10)

	if sm.GetSymbolCount() != 0 {
		t.Errorf("Expected 0 symbols, got %d", sm.GetSymbolCount())
	}

	sm.GetOrCreateState("AAPL")
	sm.GetOrCreateState("GOOGL")

	if sm.GetSymbolCount() != 2 {
		t.Errorf("Expected 2 symbols, got %d", sm.GetSymbolCount())
	}
}

func TestStateManager_RemoveSymbol(t *testing.T) {
	sm := NewStateManager(10)

	sm.GetOrCreateState("AAPL")
	sm.GetOrCreateState("GOOGL")

	sm.RemoveSymbol("AAPL")

	if sm.GetSymbolCount() != 1 {
		t.Errorf("Expected 1 symbol after removal, got %d", sm.GetSymbolCount())
	}

	if sm.GetState("AAPL") != nil {
		t.Error("Expected AAPL to be removed")
	}

	if sm.GetState("GOOGL") == nil {
		t.Error("Expected GOOGL to still exist")
	}
}

func TestStateManager_Clear(t *testing.T) {
	sm := NewStateManager(10)

	sm.GetOrCreateState("AAPL")
	sm.GetOrCreateState("GOOGL")

	sm.Clear()

	if sm.GetSymbolCount() != 0 {
		t.Errorf("Expected 0 symbols after clear, got %d", sm.GetSymbolCount())
	}
}

func TestStateManager_Concurrency(t *testing.T) {
	sm := NewStateManager(10)

	// Test concurrent updates
	done := make(chan bool)
	symbols := []string{"AAPL", "GOOGL", "MSFT", "TSLA", "AMZN"}

	// Concurrent writes
	for _, symbol := range symbols {
		go func(sym string) {
			for i := 0; i < 100; i++ {
				tick := &models.Tick{
					Symbol:    sym,
					Price:     150.0 + float64(i),
					Size:      100,
					Timestamp: time.Now(),
					Type:      "trade",
				}
				sm.UpdateLiveBar(sym, tick)
			}
			done <- true
		}(symbol)
	}

	// Concurrent reads
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				for _, symbol := range symbols {
					sm.GetMetrics(symbol)
				}
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < len(symbols)+5; i++ {
		<-done
	}

	// Verify final state
	if sm.GetSymbolCount() != len(symbols) {
		t.Errorf("Expected %d symbols, got %d", len(symbols), sm.GetSymbolCount())
	}
}

