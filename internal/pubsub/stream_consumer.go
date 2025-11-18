package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

// StreamConsumerConfig holds configuration for the stream consumer
type StreamConsumerConfig struct {
	StreamName      string
	ConsumerGroup   string
	ConsumerName    string
	Partitions      int // Number of partitions to consume from (0 = no partitioning)
	BatchSize       int // Number of messages to process before acknowledging
	ProcessTimeout  time.Duration
	AckTimeout      time.Duration
	MaxRetries      int
	RetryDelay      time.Duration
	BlockTime       time.Duration // Block time for XReadGroup
}

// DefaultStreamConsumerConfig returns default configuration
func DefaultStreamConsumerConfig(streamName, consumerGroup, consumerName string) StreamConsumerConfig {
	return StreamConsumerConfig{
		StreamName:     streamName,
		ConsumerGroup:  consumerGroup,
		ConsumerName:   consumerName,
		Partitions:     0, // No partitioning by default
		BatchSize:      100,
		ProcessTimeout: 5 * time.Second,
		AckTimeout:     10 * time.Second,
		MaxRetries:     3,
		RetryDelay:     1 * time.Second,
		BlockTime:      1 * time.Second,
	}
}

// StreamConsumer consumes ticks from Redis streams and processes them
type StreamConsumer struct {
	config     StreamConsumerConfig
	redis      storage.RedisClient
	aggregator AggregatorInterface // Interface to avoid circular dependency
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.RWMutex
	running    bool
	stats      ConsumerStats
}

// AggregatorInterface defines the interface for the bar aggregator
// This avoids circular dependency between pubsub and bars packages
type AggregatorInterface interface {
	ProcessTick(tick *models.Tick) error
}

// ConsumerStats holds statistics about the consumer
type ConsumerStats struct {
	MessagesProcessed int64
	MessagesAcked    int64
	MessagesFailed   int64
	LastMessageTime  time.Time
	Lag              int64 // Approximate lag in messages
	mu               sync.RWMutex
}

// NewStreamConsumer creates a new stream consumer
func NewStreamConsumer(redis storage.RedisClient, config StreamConsumerConfig) *StreamConsumer {
	ctx, cancel := context.WithCancel(context.Background())

	return &StreamConsumer{
		config: config,
		redis:  redis,
		ctx:    ctx,
		cancel: cancel,
		stats:  ConsumerStats{},
	}
}

// SetAggregator sets the aggregator to process ticks
func (c *StreamConsumer) SetAggregator(aggregator AggregatorInterface) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.aggregator = aggregator
}

// Start starts consuming from the stream
func (c *StreamConsumer) Start() error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return fmt.Errorf("consumer is already running")
	}
	c.running = true
	c.mu.Unlock()

	// Determine which streams to consume from
	streams := c.getStreams()

	logger.Info("Starting stream consumer",
		logger.String("stream", c.config.StreamName),
		logger.String("group", c.config.ConsumerGroup),
		logger.String("consumer", c.config.ConsumerName),
		logger.Int("partitions", c.config.Partitions),
		logger.Int("stream_count", len(streams)),
	)

	// Start consumer goroutine for each stream
	for _, stream := range streams {
		c.wg.Add(1)
		go c.consumeStream(stream)
	}

	return nil
}

// Stop stops the consumer
func (c *StreamConsumer) Stop() {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return
	}
	c.running = false
	c.mu.Unlock()

	logger.Info("Stopping stream consumer")
	c.cancel()
	c.wg.Wait()
	logger.Info("Stream consumer stopped")
}

// getStreams returns the list of streams to consume from
func (c *StreamConsumer) getStreams() []string {
	if c.config.Partitions == 0 {
		return []string{c.config.StreamName}
	}

	// If partitioning is enabled, consume from all partitions
	streams := make([]string, c.config.Partitions)
	for i := 0; i < c.config.Partitions; i++ {
		streams[i] = fmt.Sprintf("%s.p%d", c.config.StreamName, i)
	}
	return streams
}

// consumeStream consumes messages from a single stream
func (c *StreamConsumer) consumeStream(stream string) {
	defer c.wg.Done()

	// Create consumer group for this stream
	err := c.createConsumerGroup(stream)
	if err != nil {
		logger.Error("Failed to create consumer group",
			logger.ErrorField(err),
			logger.String("stream", stream),
		)
		return
	}

	messageChan, err := c.redis.ConsumeFromStream(c.ctx, stream, c.config.ConsumerGroup, c.config.ConsumerName)
	if err != nil {
		logger.Error("Failed to start consuming from stream",
			logger.ErrorField(err),
			logger.String("stream", stream),
		)
		return
	}

	batch := make([]storage.StreamMessage, 0, c.config.BatchSize)
	ticker := time.NewTicker(c.config.AckTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			// Process remaining batch before exiting
			if len(batch) > 0 {
				c.processBatch(stream, batch)
			}
			return

		case msg, ok := <-messageChan:
			if !ok {
				logger.Warn("Message channel closed",
					logger.String("stream", stream),
				)
				return
			}

			batch = append(batch, msg)

			// Process batch if it's full
			if len(batch) >= c.config.BatchSize {
				c.processBatch(stream, batch)
				batch = batch[:0] // Clear batch
			}

		case <-ticker.C:
			// Process batch on timeout
			if len(batch) > 0 {
				c.processBatch(stream, batch)
				batch = batch[:0] // Clear batch
			}
		}
	}
}

// createConsumerGroup creates a consumer group for the stream
func (c *StreamConsumer) createConsumerGroup(stream string) error {
	// The consumer group creation is handled by RedisClientImpl.ConsumeFromStream
	// This is a placeholder for any additional setup if needed
	return nil
}

// processBatch processes a batch of messages
func (c *StreamConsumer) processBatch(stream string, messages []storage.StreamMessage) {
	if len(messages) == 0 {
		return
	}

	processed := make([]string, 0, len(messages)) // Message IDs to acknowledge
	failed := make([]string, 0)                  // Message IDs that failed

	for _, msg := range messages {
		tick, err := c.deserializeTick(msg)
		if err != nil {
			logger.Error("Failed to deserialize tick",
				logger.ErrorField(err),
				logger.String("stream", stream),
				logger.String("message_id", msg.ID),
			)
			failed = append(failed, msg.ID)
			c.incrementFailed()
			continue
		}

		// Process tick through aggregator
		c.mu.RLock()
		aggregator := c.aggregator
		c.mu.RUnlock()

		if aggregator == nil {
			logger.Warn("No aggregator set, skipping tick",
				logger.String("symbol", tick.Symbol),
			)
			failed = append(failed, msg.ID)
			continue
		}

		err = aggregator.ProcessTick(tick)
		if err != nil {
			logger.Error("Failed to process tick",
				logger.ErrorField(err),
				logger.String("symbol", tick.Symbol),
				logger.String("message_id", msg.ID),
			)
			failed = append(failed, msg.ID)
			c.incrementFailed()
			continue
		}

		processed = append(processed, msg.ID)
		c.incrementProcessed()
	}

	// Acknowledge successfully processed messages
	if len(processed) > 0 {
		c.acknowledgeMessages(stream, processed)
		c.incrementAcked(int64(len(processed)))
	}

	// Log failed messages (they will be retried by consumer group)
	if len(failed) > 0 {
		logger.Warn("Some messages failed to process",
			logger.Int("failed_count", len(failed)),
			logger.String("stream", stream),
		)
	}
}

// deserializeTick deserializes a stream message into a Tick
func (c *StreamConsumer) deserializeTick(msg storage.StreamMessage) (*models.Tick, error) {
	// The stream publisher stores ticks with key "tick"
	tickJSON, ok := msg.Values["tick"].(string)
	if !ok {
		// Try to find any string value (fallback)
		for _, v := range msg.Values {
			if str, ok := v.(string); ok {
				tickJSON = str
				break
			}
		}
		if tickJSON == "" {
			return nil, fmt.Errorf("no tick data found in message")
		}
	}

	var tick models.Tick
	err := json.Unmarshal([]byte(tickJSON), &tick)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal tick: %w", err)
	}

	return &tick, nil
}

// acknowledgeMessages acknowledges a batch of messages
func (c *StreamConsumer) acknowledgeMessages(stream string, messageIDs []string) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.AckTimeout)
	defer cancel()

	for _, id := range messageIDs {
		err := c.redis.AcknowledgeMessage(ctx, stream, c.config.ConsumerGroup, id)
		if err != nil {
			logger.Error("Failed to acknowledge message",
				logger.ErrorField(err),
				logger.String("stream", stream),
				logger.String("message_id", id),
			)
		}
	}
}

// incrementProcessed increments the processed message counter
func (c *StreamConsumer) incrementProcessed() {
	c.stats.mu.Lock()
	defer c.stats.mu.Unlock()
	c.stats.MessagesProcessed++
	c.stats.LastMessageTime = time.Now()
}

// incrementAcked increments the acknowledged message counter
func (c *StreamConsumer) incrementAcked(count int64) {
	c.stats.mu.Lock()
	defer c.stats.mu.Unlock()
	c.stats.MessagesAcked += count
}

// incrementFailed increments the failed message counter
func (c *StreamConsumer) incrementFailed() {
	c.stats.mu.Lock()
	defer c.stats.mu.Unlock()
	c.stats.MessagesFailed++
}

// GetStats returns current consumer statistics
func (c *StreamConsumer) GetStats() ConsumerStats {
	c.stats.mu.RLock()
	defer c.stats.mu.RUnlock()
	return c.stats
}

// IsRunning returns whether the consumer is running
func (c *StreamConsumer) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

