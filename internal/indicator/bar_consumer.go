package indicator

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/pubsub"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

// BarConsumer consumes finalized bars from Redis streams
type BarConsumer struct {
	config      pubsub.StreamConsumerConfig
	redis       storage.RedisClient
	processor   BarProcessorInterface
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	mu          sync.RWMutex
	running     bool
	stats       ConsumerStats
}

// ConsumerStats holds statistics about the consumer
type ConsumerStats struct {
	BarsProcessed int64
	BarsAcked     int64
	BarsFailed    int64
	LastBarTime   time.Time
	Lag           int64
	mu            sync.RWMutex
}

// NewBarConsumer creates a new bar consumer
func NewBarConsumer(redis storage.RedisClient, config pubsub.StreamConsumerConfig) *BarConsumer {
	ctx, cancel := context.WithCancel(context.Background())

	return &BarConsumer{
		config: config,
		redis:  redis,
		ctx:    ctx,
		cancel: cancel,
		stats:  ConsumerStats{},
	}
}

// SetProcessor sets the bar processor
func (c *BarConsumer) SetProcessor(processor BarProcessorInterface) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.processor = processor
}

// Start starts consuming from the stream
func (c *BarConsumer) Start() error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return fmt.Errorf("consumer is already running")
	}
	c.running = true
	c.mu.Unlock()

	streams := c.getStreams()
	logger.Info("Starting bar consumer",
		logger.String("stream", c.config.StreamName),
		logger.String("consumer_group", c.config.ConsumerGroup),
		logger.Int("stream_count", len(streams)),
	)

	for _, stream := range streams {
		c.wg.Add(1)
		go c.consumeStream(stream)
	}

	return nil
}

// Stop stops the consumer
func (c *BarConsumer) Stop() {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return
	}
	c.running = false
	c.mu.Unlock()

	logger.Info("Stopping bar consumer")
	c.cancel()
	c.wg.Wait()
	logger.Info("Bar consumer stopped")
}

// IsRunning returns whether the consumer is running
func (c *BarConsumer) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

// GetStats returns consumer statistics
func (c *BarConsumer) GetStats() ConsumerStats {
	c.stats.mu.RLock()
	defer c.stats.mu.RUnlock()
	// Return a copy to avoid lock value copy warning
	return ConsumerStats{
		BarsProcessed: c.stats.BarsProcessed,
		BarsAcked:     c.stats.BarsAcked,
		BarsFailed:    c.stats.BarsFailed,
		LastBarTime:   c.stats.LastBarTime,
		Lag:           c.stats.Lag,
	}
}

// getStreams returns the list of streams to consume from
func (c *BarConsumer) getStreams() []string {
	if c.config.Partitions == 0 {
		return []string{c.config.StreamName}
	}

	streams := make([]string, c.config.Partitions)
	for i := 0; i < c.config.Partitions; i++ {
		streams[i] = fmt.Sprintf("%s.p%d", c.config.StreamName, i)
	}
	return streams
}

// consumeStream consumes messages from a single stream
func (c *BarConsumer) consumeStream(stream string) {
	defer c.wg.Done()

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

// processBatch processes a batch of messages
func (c *BarConsumer) processBatch(stream string, messages []storage.StreamMessage) {
	if len(messages) == 0 {
		return
	}

	processed := make([]string, 0, len(messages))
	failed := make([]string, 0)

	for _, msg := range messages {
		bar, err := c.deserializeBar(msg)
		if err != nil {
			logger.Error("Failed to deserialize bar",
				logger.ErrorField(err),
				logger.String("stream", stream),
				logger.String("message_id", msg.ID),
			)
			failed = append(failed, msg.ID)
			c.incrementFailed()
			continue
		}

		// Process bar through processor
		c.mu.RLock()
		processor := c.processor
		c.mu.RUnlock()

		if processor == nil {
			logger.Warn("No processor set, skipping bar",
				logger.String("symbol", bar.Symbol),
			)
			failed = append(failed, msg.ID)
			continue
		}

		err = processor.ProcessBar(bar)
		if err != nil {
			logger.Error("Failed to process bar",
				logger.ErrorField(err),
				logger.String("symbol", bar.Symbol),
				logger.String("message_id", msg.ID),
			)
			failed = append(failed, msg.ID)
			c.incrementFailed()
			continue
		}

		processed = append(processed, msg.ID)
		c.incrementProcessed()
		c.updateLastBarTime(bar.Timestamp)
	}

	// Acknowledge successfully processed messages
	if len(processed) > 0 {
		c.acknowledgeMessages(stream, processed)
		c.incrementAcked(int64(len(processed)))
	}

	// Log failed messages
	if len(failed) > 0 {
		logger.Warn("Some bars failed to process",
			logger.Int("failed_count", len(failed)),
			logger.String("stream", stream),
		)
	}
}

// deserializeBar deserializes a stream message into a Bar1m
func (c *BarConsumer) deserializeBar(msg storage.StreamMessage) (*models.Bar1m, error) {
	barJSON, ok := msg.Values["bar"].(string)
	if !ok {
		// Try to find any string value (fallback)
		for _, v := range msg.Values {
			if str, ok := v.(string); ok {
				barJSON = str
				break
			}
		}
		if barJSON == "" {
			return nil, fmt.Errorf("no bar data found in message")
		}
	}

	var bar models.Bar1m
	err := json.Unmarshal([]byte(barJSON), &bar)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal bar: %w", err)
	}

	return &bar, nil
}

// acknowledgeMessages acknowledges a batch of messages
func (c *BarConsumer) acknowledgeMessages(stream string, messageIDs []string) {
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

// incrementProcessed increments the processed counter
func (c *BarConsumer) incrementProcessed() {
	c.stats.mu.Lock()
	c.stats.BarsProcessed++
	c.stats.mu.Unlock()
}

// incrementAcked increments the acked counter
func (c *BarConsumer) incrementAcked(count int64) {
	c.stats.mu.Lock()
	c.stats.BarsAcked += count
	c.stats.mu.Unlock()
}

// incrementFailed increments the failed counter
func (c *BarConsumer) incrementFailed() {
	c.stats.mu.Lock()
	c.stats.BarsFailed++
	c.stats.mu.Unlock()
}

// updateLastBarTime updates the last bar timestamp
func (c *BarConsumer) updateLastBarTime(timestamp time.Time) {
	c.stats.mu.Lock()
	c.stats.LastBarTime = timestamp
	c.stats.mu.Unlock()
}

