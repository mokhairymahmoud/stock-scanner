package scanner

import (
	"context"
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/storage"
)

func TestIndicatorConsumer_FetchIndicators(t *testing.T) {
	sm := NewStateManager(10)
	config := DefaultIndicatorConsumerConfig()
	redis := storage.NewMockRedisClient()
	ic := NewIndicatorConsumer(redis, config, sm)

	// Set up indicator data in Redis
	indicatorData := map[string]interface{}{
		"symbol":    "AAPL",
		"timestamp": time.Now().UTC(),
		"values": map[string]interface{}{
			"rsi_14":  65.5,
			"ema_20":  150.2,
			"vwap_5m": 149.8,
		},
	}

	ctx := context.Background()
	key := "ind:AAPL"
	err := redis.Set(ctx, key, indicatorData, 10*time.Minute)
	if err != nil {
		t.Fatalf("Failed to set indicator data: %v", err)
	}

	// Fetch indicators
	indicators, err := ic.fetchIndicators("AAPL")
	if err != nil {
		t.Fatalf("Failed to fetch indicators: %v", err)
	}

	if len(indicators) != 3 {
		t.Errorf("Expected 3 indicators, got %d", len(indicators))
	}

	if indicators["rsi_14"] != 65.5 {
		t.Errorf("Expected rsi_14 = 65.5, got %f", indicators["rsi_14"])
	}

	if indicators["ema_20"] != 150.2 {
		t.Errorf("Expected ema_20 = 150.2, got %f", indicators["ema_20"])
	}
}

func TestIndicatorConsumer_NewIndicatorConsumer(t *testing.T) {
	sm := NewStateManager(10)
	config := DefaultIndicatorConsumerConfig()
	redis := storage.NewMockRedisClient()

	// Test normal creation
	ic := NewIndicatorConsumer(redis, config, sm)
	if ic == nil {
		t.Fatal("Expected indicator consumer to be created")
	}

	if ic.stateManager != sm {
		t.Error("Expected state manager to be set")
	}

	// Test panic with nil state manager
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when state manager is nil")
		}
	}()

	NewIndicatorConsumer(redis, config, nil)
}

func TestIndicatorConsumer_IsRunning(t *testing.T) {
	sm := NewStateManager(10)
	config := DefaultIndicatorConsumerConfig()
	redis := storage.NewMockRedisClient()
	ic := NewIndicatorConsumer(redis, config, sm)

	if ic.IsRunning() {
		t.Error("Expected consumer not to be running initially")
	}
}

func TestIndicatorConsumer_GetStats(t *testing.T) {
	sm := NewStateManager(10)
	config := DefaultIndicatorConsumerConfig()
	redis := storage.NewMockRedisClient()
	ic := NewIndicatorConsumer(redis, config, sm)

	stats := ic.GetStats()
	if stats.UpdatesReceived != 0 {
		t.Errorf("Expected 0 updates received, got %d", stats.UpdatesReceived)
	}

	ic.incrementReceived()
	ic.incrementProcessed()

	stats = ic.GetStats()
	if stats.UpdatesReceived != 1 {
		t.Errorf("Expected 1 update received, got %d", stats.UpdatesReceived)
	}

	if stats.UpdatesProcessed != 1 {
		t.Errorf("Expected 1 update processed, got %d", stats.UpdatesProcessed)
	}
}

func TestIndicatorConsumer_UpdateIndicators(t *testing.T) {
	sm := NewStateManager(10)
	config := DefaultIndicatorConsumerConfig()
	redis := storage.NewMockRedisClient()
	ic := NewIndicatorConsumer(redis, config, sm)

	// Set up indicator data in Redis
	indicatorData := map[string]interface{}{
		"symbol":    "AAPL",
		"timestamp": time.Now().UTC(),
		"values": map[string]interface{}{
			"rsi_14":  65.5,
			"ema_20":  150.2,
		},
	}

	ctx := context.Background()
	key := "ind:AAPL"
	err := redis.Set(ctx, key, indicatorData, 10*time.Minute)
	if err != nil {
		t.Fatalf("Failed to set indicator data: %v", err)
	}

	// Fetch and update indicators
	indicators, err := ic.fetchIndicators("AAPL")
	if err != nil {
		t.Fatalf("Failed to fetch indicators: %v", err)
	}

	err = sm.UpdateIndicators("AAPL", indicators)
	if err != nil {
		t.Fatalf("Failed to update indicators: %v", err)
	}

	// Verify state was updated
	state := sm.GetState("AAPL")
	if state == nil {
		t.Fatal("Expected AAPL state to exist")
	}

	state.mu.RLock()
	stateIndicators := state.Indicators
	state.mu.RUnlock()

	if len(stateIndicators) != 2 {
		t.Errorf("Expected 2 indicators in state, got %d", len(stateIndicators))
	}

	if stateIndicators["rsi_14"] != 65.5 {
		t.Errorf("Expected rsi_14 = 65.5, got %f", stateIndicators["rsi_14"])
	}
}

func TestDefaultIndicatorConsumerConfig(t *testing.T) {
	config := DefaultIndicatorConsumerConfig()

	if config.UpdateChannel != "indicators.updated" {
		t.Errorf("Expected UpdateChannel 'indicators.updated', got '%s'", config.UpdateChannel)
	}

	if config.IndicatorKeyPrefix != "ind:" {
		t.Errorf("Expected IndicatorKeyPrefix 'ind:', got '%s'", config.IndicatorKeyPrefix)
	}

	if config.FetchTimeout != 2*time.Second {
		t.Errorf("Expected FetchTimeout 2s, got %v", config.FetchTimeout)
	}
}

