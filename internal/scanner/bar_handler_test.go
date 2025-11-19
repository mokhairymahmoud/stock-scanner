package scanner

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/pubsub"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
)

func TestBarFinalizationHandler_DeserializeBar(t *testing.T) {
	sm := NewStateManager(10)
	config := pubsub.DefaultStreamConsumerConfig("bars.finalized", "scanner-group", "scanner-1")
	redis := storage.NewMockRedisClient()
	bh := NewBarFinalizationHandler(redis, config, sm)

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

	barJSON, err := json.Marshal(bar)
	if err != nil {
		t.Fatalf("Failed to marshal bar: %v", err)
	}

	msg := storage.StreamMessage{
		ID:     "1-0",
		Values: map[string]interface{}{"bar": string(barJSON)},
	}

	deserialized, err := bh.deserializeBar(msg)
	if err != nil {
		t.Fatalf("Failed to deserialize bar: %v", err)
	}

	if deserialized.Symbol != bar.Symbol {
		t.Errorf("Expected symbol %s, got %s", bar.Symbol, deserialized.Symbol)
	}

	if deserialized.Close != bar.Close {
		t.Errorf("Expected close %f, got %f", bar.Close, deserialized.Close)
	}
}

func TestBarFinalizationHandler_GetStreams(t *testing.T) {
	sm := NewStateManager(10)
	redis := storage.NewMockRedisClient()

	// Test without partitioning
	config := pubsub.DefaultStreamConsumerConfig("bars.finalized", "scanner-group", "scanner-1")
	config.Partitions = 0
	bh := NewBarFinalizationHandler(redis, config, sm)

	streams := bh.getStreams()
	if len(streams) != 1 {
		t.Errorf("Expected 1 stream, got %d", len(streams))
	}

	if streams[0] != "bars.finalized" {
		t.Errorf("Expected stream 'bars.finalized', got '%s'", streams[0])
	}

	// Test with partitioning
	config.Partitions = 3
	bh2 := NewBarFinalizationHandler(redis, config, sm)
	streams2 := bh2.getStreams()

	if len(streams2) != 3 {
		t.Errorf("Expected 3 streams, got %d", len(streams2))
	}

	expectedStreams := []string{"bars.finalized:0", "bars.finalized:1", "bars.finalized:2"}
	for i, expected := range expectedStreams {
		if streams2[i] != expected {
			t.Errorf("Expected stream %s, got %s", expected, streams2[i])
		}
	}
}

func TestBarFinalizationHandler_ProcessBatch(t *testing.T) {
	sm := NewStateManager(10)
	config := pubsub.DefaultStreamConsumerConfig("bars.finalized", "scanner-group", "scanner-1")
	redis := storage.NewMockRedisClient()
	bh := NewBarFinalizationHandler(redis, config, sm)

	// Create test bars
	bars := []*models.Bar1m{
		{
			Symbol:    "AAPL",
			Timestamp: time.Now(),
			Open:      150.0,
			High:      152.0,
			Low:       149.0,
			Close:     151.0,
			Volume:    1000,
			VWAP:      150.5,
		},
		{
			Symbol:    "GOOGL",
			Timestamp: time.Now(),
			Open:      2500.0,
			High:      2510.0,
			Low:       2495.0,
			Close:     2505.0,
			Volume:    500,
			VWAP:      2502.5,
		},
	}

	// Create messages
	messages := make([]storage.StreamMessage, len(bars))
	for i, bar := range bars {
		barJSON, _ := json.Marshal(bar)
		messages[i] = storage.StreamMessage{
			ID:     fmt.Sprintf("%d-0", i),
			Values: map[string]interface{}{"bar": string(barJSON)},
		}
	}

	// Process batch
	bh.processBatch("bars.finalized", messages)

	// Verify state was updated
	aaplState := sm.GetState("AAPL")
	if aaplState == nil {
		t.Fatal("Expected AAPL state to exist")
	}

	aaplState.mu.RLock()
	finalBars := aaplState.LastFinalBars
	aaplState.mu.RUnlock()

	if len(finalBars) != 1 {
		t.Errorf("Expected 1 finalized bar for AAPL, got %d", len(finalBars))
	}

	if finalBars[0].Close != 151.0 {
		t.Errorf("Expected close price 151.0, got %f", finalBars[0].Close)
	}

	// Check stats
	stats := bh.GetStats()
	if stats.BarsProcessed != 2 {
		t.Errorf("Expected 2 bars processed, got %d", stats.BarsProcessed)
	}
}

func TestBarFinalizationHandler_NewBarFinalizationHandler(t *testing.T) {
	sm := NewStateManager(10)
	config := pubsub.DefaultStreamConsumerConfig("bars.finalized", "scanner-group", "scanner-1")
	redis := storage.NewMockRedisClient()

	// Test normal creation
	bh := NewBarFinalizationHandler(redis, config, sm)
	if bh == nil {
		t.Fatal("Expected bar handler to be created")
	}

	if bh.stateManager != sm {
		t.Error("Expected state manager to be set")
	}

	// Test panic with nil state manager
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when state manager is nil")
		}
	}()

	NewBarFinalizationHandler(redis, config, nil)
}

func TestBarFinalizationHandler_IsRunning(t *testing.T) {
	sm := NewStateManager(10)
	config := pubsub.DefaultStreamConsumerConfig("bars.finalized", "scanner-group", "scanner-1")
	redis := storage.NewMockRedisClient()
	bh := NewBarFinalizationHandler(redis, config, sm)

	if bh.IsRunning() {
		t.Error("Expected handler not to be running initially")
	}
}

func TestBarFinalizationHandler_UpdateFinalizedBar(t *testing.T) {
	sm := NewStateManager(3) // Keep only 3 bars

	// Add 5 finalized bars
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

		err := sm.UpdateFinalizedBar(bar)
		if err != nil {
			t.Fatalf("Failed to update finalized bar: %v", err)
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
	if finalBars[0].Close != 153.0 {
		t.Errorf("Expected first bar close 153.0, got %f", finalBars[0].Close)
	}

	if finalBars[2].Close != 155.0 {
		t.Errorf("Expected last bar close 155.0, got %f", finalBars[2].Close)
	}
}

