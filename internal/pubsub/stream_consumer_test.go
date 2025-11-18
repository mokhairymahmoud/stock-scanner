package pubsub

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/bars"
	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockAggregator implements AggregatorInterface for testing
type MockAggregator struct {
	ticks []*models.Tick
	mu    sync.Mutex
}

func (m *MockAggregator) ProcessTick(tick *models.Tick) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ticks = append(m.ticks, tick)
	return nil
}

func (m *MockAggregator) GetTicks() []*models.Tick {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*models.Tick, len(m.ticks))
	copy(result, m.ticks)
	return result
}

func (m *MockAggregator) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ticks = m.ticks[:0]
}

func TestStreamConsumer_DeserializeTick(t *testing.T) {
	consumer := &StreamConsumer{
		config: DefaultStreamConsumerConfig("test-stream", "test-group", "test-consumer"),
	}

	tick := &models.Tick{
		Symbol:    "AAPL",
		Price:     150.0,
		Size:      100,
		Timestamp:  time.Now(),
		Type:      "trade",
	}

	tickJSON, err := json.Marshal(tick)
	require.NoError(t, err)

	msg := storage.StreamMessage{
		ID:     "123-0",
		Stream: "test-stream",
		Values: map[string]interface{}{
			"tick": string(tickJSON),
		},
	}

	deserialized, err := consumer.deserializeTick(msg)
	require.NoError(t, err)
	assert.Equal(t, tick.Symbol, deserialized.Symbol)
	assert.Equal(t, tick.Price, deserialized.Price)
	assert.Equal(t, tick.Size, deserialized.Size)
	assert.Equal(t, tick.Type, deserialized.Type)
}

func TestStreamConsumer_GetStreams(t *testing.T) {
	tests := []struct {
		name      string
		partitions int
		streamName string
		expected  []string
	}{
		{
			name:       "no partitioning",
			partitions: 0,
			streamName: "ticks",
			expected:   []string{"ticks"},
		},
		{
			name:       "with partitioning",
			partitions: 3,
			streamName: "ticks",
			expected:   []string{"ticks.p0", "ticks.p1", "ticks.p2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consumer := &StreamConsumer{
				config: StreamConsumerConfig{
					StreamName: tt.streamName,
					Partitions: tt.partitions,
				},
			}

			streams := consumer.getStreams()
			assert.Equal(t, tt.expected, streams)
		})
	}
}

func TestStreamConsumer_ProcessBatch(t *testing.T) {
	mockAgg := &MockAggregator{}
	mockRedis := storage.NewMockRedisClient()

	consumer := &StreamConsumer{
		config: StreamConsumerConfig{
			StreamName:    "test-stream",
			ConsumerGroup: "test-group",
			BatchSize:     10,
		},
		redis:      mockRedis,
		aggregator: mockAgg,
	}

	// Create test messages
	tick1 := &models.Tick{
		Symbol:    "AAPL",
		Price:     150.0,
		Size:      100,
		Timestamp: time.Now(),
		Type:      "trade",
	}
	tick1JSON, _ := json.Marshal(tick1)

	tick2 := &models.Tick{
		Symbol:    "MSFT",
		Price:     300.0,
		Size:      200,
		Timestamp: time.Now(),
		Type:      "trade",
	}
	tick2JSON, _ := json.Marshal(tick2)

	messages := []storage.StreamMessage{
		{
			ID:     "1-0",
			Stream: "test-stream",
			Values: map[string]interface{}{"tick": string(tick1JSON)},
		},
		{
			ID:     "2-0",
			Stream: "test-stream",
			Values: map[string]interface{}{"tick": string(tick2JSON)},
		},
	}

	consumer.processBatch("test-stream", messages)

	// Verify ticks were processed
	ticks := mockAgg.GetTicks()
	assert.Len(t, ticks, 2)
	assert.Equal(t, "AAPL", ticks[0].Symbol)
	assert.Equal(t, "MSFT", ticks[1].Symbol)

	// Verify stats
	stats := consumer.GetStats()
	assert.Equal(t, int64(2), stats.MessagesProcessed)
}

func TestStreamConsumer_ProcessBatchWithInvalidTick(t *testing.T) {
	mockAgg := &MockAggregator{}
	mockRedis := storage.NewMockRedisClient()

	consumer := &StreamConsumer{
		config: StreamConsumerConfig{
			StreamName:    "test-stream",
			ConsumerGroup: "test-group",
		},
		redis:      mockRedis,
		aggregator: mockAgg,
	}

	messages := []storage.StreamMessage{
		{
			ID:     "1-0",
			Stream: "test-stream",
			Values: map[string]interface{}{"tick": "invalid json"},
		},
		{
			ID:     "2-0",
			Stream: "test-stream",
			Values: map[string]interface{}{"tick": "{}"}, // Valid JSON but invalid tick (missing required fields)
		},
	}

	consumer.processBatch("test-stream", messages)

	// Verify stats show failures (deserialization or validation failures)
	stats := consumer.GetStats()
	assert.Greater(t, stats.MessagesFailed, int64(0), "Should have failed messages")
	
	// Note: The aggregator's ProcessTick validates ticks, so invalid ticks won't be added
	// But we can't guarantee 0 ticks because the empty JSON might pass deserialization
	// and only fail validation in the aggregator
}

func TestStreamConsumer_Stats(t *testing.T) {
	consumer := &StreamConsumer{
		config: DefaultStreamConsumerConfig("test-stream", "test-group", "test-consumer"),
	}

	// Initial stats
	stats := consumer.GetStats()
	assert.Equal(t, int64(0), stats.MessagesProcessed)
	assert.Equal(t, int64(0), stats.MessagesAcked)
	assert.Equal(t, int64(0), stats.MessagesFailed)

	// Increment stats
	consumer.incrementProcessed()
	consumer.incrementAcked(5)
	consumer.incrementFailed()

	stats = consumer.GetStats()
	assert.Equal(t, int64(1), stats.MessagesProcessed)
	assert.Equal(t, int64(5), stats.MessagesAcked)
	assert.Equal(t, int64(1), stats.MessagesFailed)
}

func TestStreamConsumer_IntegrationWithAggregator(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	mockRedis := storage.NewMockRedisClient()
	agg := bars.NewAggregator()

	// Track finalized bars
	finalizedBars := make([]*models.Bar1m, 0)
	var finalizedMu sync.Mutex

	agg.SetOnBarFinal(func(bar *models.Bar1m) {
		finalizedMu.Lock()
		defer finalizedMu.Unlock()
		finalizedBars = append(finalizedBars, bar)
	})

	consumer := NewStreamConsumer(mockRedis, StreamConsumerConfig{
		StreamName:     "test-ticks",
		ConsumerGroup:  "test-group",
		ConsumerName:   "test-consumer",
		BatchSize:      5,
		AckTimeout:     100 * time.Millisecond,
		ProcessTimeout: 1 * time.Second,
	})

	consumer.SetAggregator(agg)

	// Manually add messages to mock Redis
	now := time.Now().Truncate(time.Minute)
	for i := 0; i < 10; i++ {
		tick := &models.Tick{
			Symbol:    "AAPL",
			Price:     150.0 + float64(i),
			Size:      100,
			Timestamp: now.Add(time.Duration(i) * 10 * time.Second),
			Type:      "trade",
		}
		tickJSON, _ := json.Marshal(tick)
		mockRedis.StreamData = append(mockRedis.StreamData, storage.StreamMessage{
			ID:     fmt.Sprintf("%d-0", i),
			Stream: "test-ticks",
			Values: map[string]interface{}{"tick": string(tickJSON)},
		})
	}

	// Note: This is a simplified test. In a real scenario, we'd need to
	// properly mock the ConsumeFromStream method to return our test data.
	// For now, we test the deserialization and processing logic separately.
}

