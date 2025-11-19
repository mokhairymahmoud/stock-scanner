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


func TestTickConsumer_DeserializeTick(t *testing.T) {
	sm := NewStateManager(10)
	config := pubsub.DefaultStreamConsumerConfig("ticks", "scanner-group", "scanner-1")
	redis := storage.NewMockRedisClient()
	tc := NewTickConsumer(redis, config, sm)

	tick := &models.Tick{
		Symbol:    "AAPL",
		Price:     150.0,
		Size:      100,
		Timestamp: time.Now(),
		Type:      "trade",
	}

	tickJSON, err := json.Marshal(tick)
	if err != nil {
		t.Fatalf("Failed to marshal tick: %v", err)
	}

	msg := storage.StreamMessage{
		ID:     "1-0",
		Values:  map[string]interface{}{"tick": string(tickJSON)},
	}

	deserialized, err := tc.deserializeTick(msg)
	if err != nil {
		t.Fatalf("Failed to deserialize tick: %v", err)
	}

	if deserialized.Symbol != tick.Symbol {
		t.Errorf("Expected symbol %s, got %s", tick.Symbol, deserialized.Symbol)
	}

	if deserialized.Price != tick.Price {
		t.Errorf("Expected price %f, got %f", tick.Price, deserialized.Price)
	}
}

func TestTickConsumer_GetStreams(t *testing.T) {
	sm := NewStateManager(10)
	redis := storage.NewMockRedisClient()

	// Test without partitioning
	config := pubsub.DefaultStreamConsumerConfig("ticks", "scanner-group", "scanner-1")
	config.Partitions = 0
	tc := NewTickConsumer(redis, config, sm)

	streams := tc.getStreams()
	if len(streams) != 1 {
		t.Errorf("Expected 1 stream, got %d", len(streams))
	}

	if streams[0] != "ticks" {
		t.Errorf("Expected stream 'ticks', got '%s'", streams[0])
	}

	// Test with partitioning
	config.Partitions = 3
	tc2 := NewTickConsumer(redis, config, sm)
	streams2 := tc2.getStreams()

	if len(streams2) != 3 {
		t.Errorf("Expected 3 streams, got %d", len(streams2))
	}

	expectedStreams := []string{"ticks:0", "ticks:1", "ticks:2"}
	for i, expected := range expectedStreams {
		if streams2[i] != expected {
			t.Errorf("Expected stream %s, got %s", expected, streams2[i])
		}
	}
}

func TestTickConsumer_ProcessBatch(t *testing.T) {
	sm := NewStateManager(10)
	config := pubsub.DefaultStreamConsumerConfig("ticks", "scanner-group", "scanner-1")
	redis := storage.NewMockRedisClient()
	tc := NewTickConsumer(redis, config, sm)

	// Create test ticks
	ticks := []*models.Tick{
		{
			Symbol:    "AAPL",
			Price:     150.0,
			Size:      100,
			Timestamp: time.Now(),
			Type:      "trade",
		},
		{
			Symbol:    "GOOGL",
			Price:     2500.0,
			Size:      50,
			Timestamp: time.Now(),
			Type:      "trade",
		},
	}

	// Create messages
	messages := make([]storage.StreamMessage, len(ticks))
	for i, tick := range ticks {
		tickJSON, _ := json.Marshal(tick)
		messages[i] = storage.StreamMessage{
			ID:     fmt.Sprintf("%d-0", i),
			Values:  map[string]interface{}{"tick": string(tickJSON)},
		}
	}

	// Process batch
	tc.processBatch("ticks", messages)

	// Verify state was updated
	aaplState := sm.GetState("AAPL")
	if aaplState == nil {
		t.Fatal("Expected AAPL state to exist")
	}

	aaplState.mu.RLock()
	liveBar := aaplState.LiveBar
	aaplState.mu.RUnlock()

	if liveBar == nil {
		t.Error("Expected live bar to exist for AAPL")
	} else if liveBar.Close != 150.0 {
		t.Errorf("Expected close price 150.0, got %f", liveBar.Close)
	}

	// Check stats
	stats := tc.GetStats()
	if stats.TicksProcessed != 2 {
		t.Errorf("Expected 2 ticks processed, got %d", stats.TicksProcessed)
	}
}

func TestTickConsumer_NewTickConsumer(t *testing.T) {
	sm := NewStateManager(10)
	config := pubsub.DefaultStreamConsumerConfig("ticks", "scanner-group", "scanner-1")
	redis := storage.NewMockRedisClient()

	// Test normal creation
	tc := NewTickConsumer(redis, config, sm)
	if tc == nil {
		t.Fatal("Expected tick consumer to be created")
	}

	if tc.stateManager != sm {
		t.Error("Expected state manager to be set")
	}

	// Test panic with nil state manager
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when state manager is nil")
		}
	}()

	NewTickConsumer(redis, config, nil)
}

func TestTickConsumer_IsRunning(t *testing.T) {
	sm := NewStateManager(10)
	config := pubsub.DefaultStreamConsumerConfig("ticks", "scanner-group", "scanner-1")
	redis := storage.NewMockRedisClient()
	tc := NewTickConsumer(redis, config, sm)

	if tc.IsRunning() {
		t.Error("Expected consumer not to be running initially")
	}

	// Note: We can't easily test Start() without a real Redis connection
	// This would require more sophisticated mocking
}

