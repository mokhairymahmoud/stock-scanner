package pubsub

import (
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamPublisher_Publish(t *testing.T) {
	mockRedis := storage.NewMockRedisClient()
	config := DefaultStreamPublisherConfig("test-stream")
	config.BatchSize = 10
	config.BatchTimeout = 100 * time.Millisecond

	publisher := NewStreamPublisher(mockRedis, config)
	publisher.Start()
	defer publisher.Close()

	// Publish a tick
	tick := &models.Tick{
		Symbol:    "AAPL",
		Price:     150.0,
		Size:      100,
		Timestamp: time.Now().UTC(),
		Type:      "trade",
	}

	err := publisher.Publish(tick)
	require.NoError(t, err)

	// Wait for batch timeout
	time.Sleep(150 * time.Millisecond)

	// Verify batch was flushed
	assert.Equal(t, 0, publisher.GetBatchSize())
}

func TestStreamPublisher_BatchFlush(t *testing.T) {
	mockRedis := storage.NewMockRedisClient()
	config := DefaultStreamPublisherConfig("test-stream")
	config.BatchSize = 5
	config.BatchTimeout = 1 * time.Second

	publisher := NewStreamPublisher(mockRedis, config)
	publisher.Start()
	defer publisher.Close()

	// Publish multiple ticks to fill batch
	for i := 0; i < 5; i++ {
		tick := &models.Tick{
			Symbol:    "AAPL",
			Price:     150.0 + float64(i),
			Size:      100,
			Timestamp: time.Now().UTC(),
			Type:      "trade",
		}
		err := publisher.Publish(tick)
		require.NoError(t, err)
	}

	// Batch should be flushed immediately when full
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, publisher.GetBatchSize())
}

func TestStreamPublisher_Partitioning(t *testing.T) {
	mockRedis := storage.NewMockRedisClient()
	config := DefaultStreamPublisherConfig("test-stream")
	config.BatchSize = 10
	config.Partitions = 4

	publisher := NewStreamPublisher(mockRedis, config)
	publisher.Start()
	defer publisher.Close()

	// Publish ticks for different symbols
	symbols := []string{"AAPL", "MSFT", "GOOGL", "TSLA"}
	for _, symbol := range symbols {
		tick := &models.Tick{
			Symbol:    symbol,
			Price:     150.0,
			Size:      100,
			Timestamp: time.Now().UTC(),
			Type:      "trade",
		}
		err := publisher.Publish(tick)
		require.NoError(t, err)
	}

	// Flush
	err := publisher.Flush()
	require.NoError(t, err)

	// Verify partitioning (each symbol should go to a partition)
	// We can't directly verify which partition, but we can verify the publisher works
	assert.Equal(t, 0, publisher.GetBatchSize())
}

func TestStreamPublisher_InvalidTick(t *testing.T) {
	mockRedis := storage.NewMockRedisClient()
	config := DefaultStreamPublisherConfig("test-stream")
	publisher := NewStreamPublisher(mockRedis, config)
	publisher.Start()
	defer publisher.Close()

	// Publish nil tick
	err := publisher.Publish(nil)
	assert.Error(t, err)

	// Publish invalid tick
	invalidTick := &models.Tick{
		Symbol: "", // Invalid: empty symbol
		Price:   150.0,
		Size:    100,
	}
	err = publisher.Publish(invalidTick)
	assert.Error(t, err)
}

func TestStreamPublisher_Flush(t *testing.T) {
	mockRedis := storage.NewMockRedisClient()
	config := DefaultStreamPublisherConfig("test-stream")
	config.BatchSize = 100
	config.BatchTimeout = 10 * time.Second

	publisher := NewStreamPublisher(mockRedis, config)
	publisher.Start()
	defer publisher.Close()

	// Publish a tick
	tick := &models.Tick{
		Symbol:    "AAPL",
		Price:     150.0,
		Size:      100,
		Timestamp: time.Now().UTC(),
		Type:      "trade",
	}

	err := publisher.Publish(tick)
	require.NoError(t, err)

	// Verify batch has item
	assert.Equal(t, 1, publisher.GetBatchSize())

	// Flush manually
	err = publisher.Flush()
	require.NoError(t, err)

	// Verify batch is empty
	assert.Equal(t, 0, publisher.GetBatchSize())
}

func TestStreamPublisher_Close(t *testing.T) {
	mockRedis := storage.NewMockRedisClient()
	config := DefaultStreamPublisherConfig("test-stream")
	config.BatchSize = 100
	config.BatchTimeout = 10 * time.Second

	publisher := NewStreamPublisher(mockRedis, config)
	publisher.Start()

	// Publish a tick
	tick := &models.Tick{
		Symbol:    "AAPL",
		Price:     150.0,
		Size:      100,
		Timestamp: time.Now().UTC(),
		Type:      "trade",
	}

	err := publisher.Publish(tick)
	require.NoError(t, err)

	// Close should flush remaining items
	err = publisher.Close()
	require.NoError(t, err)

	// Verify batch is empty
	assert.Equal(t, 0, publisher.GetBatchSize())
}

func TestStreamPublisher_GetPartitionStreamName(t *testing.T) {
	config := DefaultStreamPublisherConfig("test-stream")
	config.Partitions = 4

	publisher := NewStreamPublisher(storage.NewMockRedisClient(), config)

	// Test with partitioning
	assert.Equal(t, "test-stream.p0", publisher.GetPartitionStreamName(0))
	assert.Equal(t, "test-stream.p1", publisher.GetPartitionStreamName(1))
	assert.Equal(t, "test-stream.p2", publisher.GetPartitionStreamName(2))
	assert.Equal(t, "test-stream.p3", publisher.GetPartitionStreamName(3))

	// Test without partitioning
	config2 := DefaultStreamPublisherConfig("test-stream")
	config2.Partitions = 0
	publisher2 := NewStreamPublisher(storage.NewMockRedisClient(), config2)
	assert.Equal(t, "test-stream", publisher2.GetPartitionStreamName(0))
}

func TestStreamPublisher_RetryOnError(t *testing.T) {
	mockRedis := storage.NewMockRedisClient()
	mockRedis.PublishErr = assert.AnError // Simulate error

	config := DefaultStreamPublisherConfig("test-stream")
	config.BatchSize = 1
	config.RetryAttempts = 2
	config.RetryDelay = 10 * time.Millisecond

	publisher := NewStreamPublisher(mockRedis, config)
	publisher.Start()
	defer publisher.Close()

	tick := &models.Tick{
		Symbol:    "AAPL",
		Price:     150.0,
		Size:      100,
		Timestamp: time.Now().UTC(),
		Type:      "trade",
	}

	// Publish will trigger flush when batch is full (size=1)
	// Since PublishErr is set, flush will fail after retries
	err := publisher.Publish(tick)
	// Error is expected because batch is full and flush fails
	require.Error(t, err, "Expected error when PublishErr is set and batch is flushed")
}

func TestDefaultStreamPublisherConfig(t *testing.T) {
	config := DefaultStreamPublisherConfig("test-stream")

	assert.Equal(t, "test-stream", config.StreamName)
	assert.Equal(t, 100, config.BatchSize)
	assert.Equal(t, 100*time.Millisecond, config.BatchTimeout)
	assert.Equal(t, 0, config.Partitions)
	assert.Equal(t, 3, config.RetryAttempts)
	assert.Equal(t, 100*time.Millisecond, config.RetryDelay)
}

