package scanner

import (
	"context"
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
)

// mockBarStorage is a mock implementation of BarStorage for testing
type mockBarStorage struct {
	bars map[string][]*models.Bar1m
}

func newMockBarStorage() *mockBarStorage {
	return &mockBarStorage{
		bars: make(map[string][]*models.Bar1m),
	}
}

func (m *mockBarStorage) WriteBars(ctx context.Context, bars []*models.Bar1m) error {
	for _, bar := range bars {
		m.bars[bar.Symbol] = append(m.bars[bar.Symbol], bar)
	}
	return nil
}

func (m *mockBarStorage) GetBars(ctx context.Context, symbol string, start, end time.Time) ([]*models.Bar1m, error) {
	symbolBars := m.bars[symbol]
	result := make([]*models.Bar1m, 0)
	for _, bar := range symbolBars {
		if !bar.Timestamp.Before(start) && !bar.Timestamp.After(end) {
			result = append(result, bar)
		}
	}
	return result, nil
}

func (m *mockBarStorage) GetLatestBars(ctx context.Context, symbol string, limit int) ([]*models.Bar1m, error) {
	symbolBars := m.bars[symbol]
	if len(symbolBars) == 0 {
		return []*models.Bar1m{}, nil
	}

	// Return last N bars
	start := len(symbolBars) - limit
	if start < 0 {
		start = 0
	}
	return symbolBars[start:], nil
}

func (m *mockBarStorage) Close() error {
	return nil
}

func TestNewRehydrator(t *testing.T) {
	sm := NewStateManager(10)
	barStorage := newMockBarStorage()
	redis := storage.NewMockRedisClient()
	config := DefaultRehydrationConfig()

	// Test normal creation
	rehydrator := NewRehydrator(config, sm, barStorage, redis)
	if rehydrator == nil {
		t.Fatal("Expected rehydrator to be created")
	}

	// Test panic with nil state manager
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when state manager is nil")
		}
	}()

	NewRehydrator(config, nil, barStorage, redis)
}

func TestRehydrator_LoadRecentBars(t *testing.T) {
	sm := NewStateManager(10)
	barStorage := newMockBarStorage()
	redis := storage.NewMockRedisClient()
	config := DefaultRehydrationConfig()
	rehydrator := NewRehydrator(config, sm, barStorage, redis)

	// Add some bars to storage
	symbol := "AAPL"
	bars := []*models.Bar1m{
		{
			Symbol:    symbol,
			Timestamp: time.Now().Add(-10 * time.Minute),
			Open:      150.0,
			High:      152.0,
			Low:       149.0,
			Close:     151.0,
			Volume:    1000,
			VWAP:      150.5,
		},
		{
			Symbol:    symbol,
			Timestamp: time.Now().Add(-5 * time.Minute),
			Open:      151.0,
			High:      153.0,
			Low:       150.0,
			Close:     152.0,
			Volume:    1200,
			VWAP:      151.5,
		},
	}

	barStorage.WriteBars(context.Background(), bars)

	// Load bars
	err := rehydrator.loadRecentBars(context.Background(), symbol)
	if err != nil {
		t.Fatalf("Failed to load recent bars: %v", err)
	}

	// Verify bars were loaded into state
	state := sm.GetState(symbol)
	if state == nil {
		t.Fatal("Expected symbol state to exist")
	}

	if len(state.LastFinalBars) != len(bars) {
		t.Errorf("Expected %d bars in state, got %d", len(bars), len(state.LastFinalBars))
	}
}

func TestRehydrator_LoadIndicators(t *testing.T) {
	sm := NewStateManager(10)
	barStorage := newMockBarStorage()
	redis := storage.NewMockRedisClient()
	config := DefaultRehydrationConfig()
	rehydrator := NewRehydrator(config, sm, barStorage, redis)

	symbol := "AAPL"
	indicators := map[string]float64{
		"rsi_14": 65.5,
		"ema_20": 150.2,
	}

	indicatorData := map[string]interface{}{
		"symbol":    symbol,
		"timestamp": time.Now().UTC(),
		"values":    indicators,
	}

	key := config.IndicatorKeyPrefix + symbol
	redis.Set(context.Background(), key, indicatorData, 0)

	// Load indicators
	err := rehydrator.loadIndicators(context.Background(), symbol)
	if err != nil {
		t.Fatalf("Failed to load indicators: %v", err)
	}

	// Verify indicators were loaded
	state := sm.GetState(symbol)
	if state == nil {
		t.Fatal("Expected symbol state to exist")
	}

	if state.Indicators["rsi_14"] != 65.5 {
		t.Errorf("Expected rsi_14 = 65.5, got %f", state.Indicators["rsi_14"])
	}

	if state.Indicators["ema_20"] != 150.2 {
		t.Errorf("Expected ema_20 = 150.2, got %f", state.Indicators["ema_20"])
	}
}

func TestRehydrator_LoadIndicators_NotFound(t *testing.T) {
	sm := NewStateManager(10)
	barStorage := newMockBarStorage()
	redis := storage.NewMockRedisClient()
	config := DefaultRehydrationConfig()
	rehydrator := NewRehydrator(config, sm, barStorage, redis)

	symbol := "AAPL"

	// Try to load indicators for symbol that doesn't exist
	err := rehydrator.loadIndicators(context.Background(), symbol)
	if err != nil {
		t.Fatalf("Expected no error for missing indicators, got %v", err)
	}
}

func TestRehydrator_RehydrateSymbol(t *testing.T) {
	sm := NewStateManager(10)
	barStorage := newMockBarStorage()
	redis := storage.NewMockRedisClient()
	config := DefaultRehydrationConfig()
	rehydrator := NewRehydrator(config, sm, barStorage, redis)

	symbol := "AAPL"

	// Add bars
	bars := []*models.Bar1m{
		{
			Symbol:    symbol,
			Timestamp: time.Now().Add(-10 * time.Minute),
			Open:      150.0,
			High:      152.0,
			Low:       149.0,
			Close:     151.0,
			Volume:    1000,
			VWAP:      150.5,
		},
	}
	barStorage.WriteBars(context.Background(), bars)

	// Add indicators
	indicators := map[string]float64{
		"rsi_14": 65.5,
	}
	indicatorData := map[string]interface{}{
		"symbol":    symbol,
		"timestamp": time.Now().UTC(),
		"values":    indicators,
	}
	key := config.IndicatorKeyPrefix + symbol
	redis.Set(context.Background(), key, indicatorData, 0)

	// Rehydrate symbol
	err := rehydrator.rehydrateSymbol(context.Background(), symbol)
	if err != nil {
		t.Fatalf("Failed to rehydrate symbol: %v", err)
	}

	// Verify state
	state := sm.GetState(symbol)
	if state == nil {
		t.Fatal("Expected symbol state to exist")
	}

	if len(state.LastFinalBars) != 1 {
		t.Errorf("Expected 1 bar in state, got %d", len(state.LastFinalBars))
	}

	if state.Indicators["rsi_14"] != 65.5 {
		t.Errorf("Expected rsi_14 = 65.5, got %f", state.Indicators["rsi_14"])
	}
}

func TestRehydrator_RehydrateState(t *testing.T) {
	sm := NewStateManager(10)
	barStorage := newMockBarStorage()
	redis := storage.NewMockRedisClient()
	config := DefaultRehydrationConfig()
	config.Symbols = []string{"AAPL", "GOOGL"} // Provide symbols explicitly
	rehydrator := NewRehydrator(config, sm, barStorage, redis)

	// Add bars for both symbols
	bars := []*models.Bar1m{
		{
			Symbol:    "AAPL",
			Timestamp: time.Now().Add(-10 * time.Minute),
			Open:      150.0,
			High:      152.0,
			Low:       149.0,
			Close:     151.0,
			Volume:    1000,
			VWAP:      150.5,
		},
		{
			Symbol:    "GOOGL",
			Timestamp: time.Now().Add(-10 * time.Minute),
			Open:      2500.0,
			High:      2520.0,
			Low:       2490.0,
			Close:     2510.0,
			Volume:    500,
			VWAP:      2505.0,
		},
	}
	barStorage.WriteBars(context.Background(), bars)

	// Rehydrate state
	err := rehydrator.RehydrateState(context.Background())
	if err != nil {
		t.Fatalf("Failed to rehydrate state: %v", err)
	}

	// Verify both symbols have state
	if sm.GetState("AAPL") == nil {
		t.Error("Expected AAPL state to exist")
	}

	if sm.GetState("GOOGL") == nil {
		t.Error("Expected GOOGL state to exist")
	}
}

func TestRehydrator_IsReady(t *testing.T) {
	sm := NewStateManager(10)
	barStorage := newMockBarStorage()
	redis := storage.NewMockRedisClient()
	config := DefaultRehydrationConfig()
	rehydrator := NewRehydrator(config, sm, barStorage, redis)

	// Initially not ready (no symbols)
	if rehydrator.IsReady() {
		t.Error("Expected rehydrator not to be ready initially")
	}

	// Add a symbol
	sm.GetOrCreateState("AAPL")

	// Should be ready now
	if !rehydrator.IsReady() {
		t.Error("Expected rehydrator to be ready after adding symbol")
	}
}

func TestDefaultRehydrationConfig(t *testing.T) {
	config := DefaultRehydrationConfig()

	if config.MaxBarsToLoad != 200 {
		t.Errorf("Expected MaxBarsToLoad 200, got %d", config.MaxBarsToLoad)
	}

	if config.BarLookbackWindow != 1*time.Hour {
		t.Errorf("Expected BarLookbackWindow 1h, got %v", config.BarLookbackWindow)
	}

	if config.IndicatorKeyPrefix != "ind:" {
		t.Errorf("Expected IndicatorKeyPrefix 'ind:', got '%s'", config.IndicatorKeyPrefix)
	}

	if config.RehydrationTimeout != 30*time.Second {
		t.Errorf("Expected RehydrationTimeout 30s, got %v", config.RehydrationTimeout)
	}
}

