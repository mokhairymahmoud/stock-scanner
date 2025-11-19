package scanner

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

// BarFinalizationHandler consumes finalized bars from Redis streams and updates symbol state
type BarFinalizationHandler struct {
	config       pubsub.StreamConsumerConfig
	redis        storage.RedisClient
	stateManager *StateManager
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	mu           sync.RWMutex
	running      bool
	stats        BarHandlerStats
}

// BarHandlerStats holds statistics about the bar handler
type BarHandlerStats struct {
	BarsProcessed int64
	BarsAcked     int64
	BarsFailed    int64
	LastBarTime   time.Time
	Lag           int64
	mu            sync.RWMutex
}

// NewBarFinalizationHandler creates a new bar finalization handler
func NewBarFinalizationHandler(redis storage.RedisClient, config pubsub.StreamConsumerConfig, stateManager *StateManager) *BarFinalizationHandler {
	if stateManager == nil {
		panic("stateManager cannot be nil")
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &BarFinalizationHandler{
		config:       config,
		redis:        redis,
		stateManager: stateManager,
		ctx:          ctx,
		cancel:       cancel,
		stats:        BarHandlerStats{},
	}
}

// Start starts consuming finalized bars from the stream
func (bh *BarFinalizationHandler) Start() error {
	bh.mu.Lock()
	if bh.running {
		bh.mu.Unlock()
		return fmt.Errorf("bar handler is already running")
	}
	bh.running = true
	bh.mu.Unlock()

	// Determine which streams to consume from (handle partitioning)
	streams := bh.getStreams()

	if len(streams) == 0 {
		return fmt.Errorf("no streams to consume from")
	}

	logger.Info("Starting bar finalization handler",
		logger.String("stream", bh.config.StreamName),
		logger.String("consumer_group", bh.config.ConsumerGroup),
		logger.String("consumer_name", bh.config.ConsumerName),
		logger.Int("streams", len(streams)),
	)

	// Start consuming from each stream
	for _, stream := range streams {
		bh.wg.Add(1)
		go bh.consumeStream(stream)
	}

	return nil
}

// Stop stops the bar handler
func (bh *BarFinalizationHandler) Stop() {
	bh.mu.Lock()
	if !bh.running {
		bh.mu.Unlock()
		return
	}
	bh.running = false
	bh.mu.Unlock()

	logger.Info("Stopping bar finalization handler")

	bh.cancel()
	bh.wg.Wait()

	logger.Info("Bar finalization handler stopped")
}

// IsRunning returns whether the handler is running
func (bh *BarFinalizationHandler) IsRunning() bool {
	bh.mu.RLock()
	defer bh.mu.RUnlock()
	return bh.running
}

// GetStats returns current handler statistics
func (bh *BarFinalizationHandler) GetStats() BarHandlerStats {
	bh.stats.mu.RLock()
	defer bh.stats.mu.RUnlock()

	// Return a copy to avoid returning a struct with a mutex
	return BarHandlerStats{
		BarsProcessed: bh.stats.BarsProcessed,
		BarsAcked:     bh.stats.BarsAcked,
		BarsFailed:    bh.stats.BarsFailed,
		LastBarTime:   bh.stats.LastBarTime,
		Lag:           bh.stats.Lag,
	}
}

// getStreams returns the list of streams to consume from
// Handles partitioning if configured
func (bh *BarFinalizationHandler) getStreams() []string {
	if bh.config.Partitions <= 0 {
		// No partitioning - consume from single stream
		return []string{bh.config.StreamName}
	}

	// Partitioning: consume from multiple partition streams
	streams := make([]string, 0, bh.config.Partitions)
	for i := 0; i < bh.config.Partitions; i++ {
		streamName := fmt.Sprintf("%s:%d", bh.config.StreamName, i)
		streams = append(streams, streamName)
	}

	return streams
}

// consumeStream consumes messages from a single stream
func (bh *BarFinalizationHandler) consumeStream(stream string) {
	defer bh.wg.Done()

	messageChan, err := bh.redis.ConsumeFromStream(bh.ctx, stream, bh.config.ConsumerGroup, bh.config.ConsumerName)
	if err != nil {
		logger.Error("Failed to start consuming from stream",
			logger.ErrorField(err),
			logger.String("stream", stream),
		)
		return
	}

	batch := make([]storage.StreamMessage, 0, bh.config.BatchSize)
	ticker := time.NewTicker(bh.config.AckTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-bh.ctx.Done():
			// Process remaining batch before exiting
			if len(batch) > 0 {
				bh.processBatch(stream, batch)
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
			if len(batch) >= bh.config.BatchSize {
				bh.processBatch(stream, batch)
				batch = batch[:0] // Clear batch
			}

		case <-ticker.C:
			// Process batch on timeout
			if len(batch) > 0 {
				bh.processBatch(stream, batch)
				batch = batch[:0] // Clear batch
			}
		}
	}
}

// processBatch processes a batch of messages
func (bh *BarFinalizationHandler) processBatch(stream string, messages []storage.StreamMessage) {
	if len(messages) == 0 {
		return
	}

	processed := make([]string, 0, len(messages)) // Message IDs to acknowledge
	failed := make([]string, 0)                   // Message IDs that failed

	for _, msg := range messages {
		bar, err := bh.deserializeBar(msg)
		if err != nil {
			logger.Error("Failed to deserialize bar",
				logger.ErrorField(err),
				logger.String("stream", stream),
				logger.String("message_id", msg.ID),
			)
			failed = append(failed, msg.ID)
			bh.incrementFailed()
			continue
		}

		// Validate bar
		if err := bar.Validate(); err != nil {
			logger.Error("Invalid bar",
				logger.ErrorField(err),
				logger.String("symbol", bar.Symbol),
				logger.String("message_id", msg.ID),
			)
			failed = append(failed, msg.ID)
			bh.incrementFailed()
			continue
		}

		// Update state manager with finalized bar
		err = bh.stateManager.UpdateFinalizedBar(bar)
		if err != nil {
			logger.Error("Failed to update finalized bar",
				logger.ErrorField(err),
				logger.String("symbol", bar.Symbol),
				logger.String("message_id", msg.ID),
			)
			failed = append(failed, msg.ID)
			bh.incrementFailed()
			continue
		}

		processed = append(processed, msg.ID)
		bh.incrementProcessed(bar.Timestamp)
	}

	// Acknowledge successfully processed messages
	if len(processed) > 0 {
		bh.acknowledgeMessages(stream, processed)
		bh.incrementAcked(int64(len(processed)))
	}

	// Log failed messages (they will be retried by consumer group)
	if len(failed) > 0 {
		logger.Warn("Some bars failed to process",
			logger.Int("failed_count", len(failed)),
			logger.String("stream", stream),
		)
	}
}

// deserializeBar deserializes a stream message into a Bar1m
func (bh *BarFinalizationHandler) deserializeBar(msg storage.StreamMessage) (*models.Bar1m, error) {
	// The stream publisher stores bars with key "bar"
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
func (bh *BarFinalizationHandler) acknowledgeMessages(stream string, messageIDs []string) {
	ctx, cancel := context.WithTimeout(context.Background(), bh.config.AckTimeout)
	defer cancel()

	for _, id := range messageIDs {
		err := bh.redis.AcknowledgeMessage(ctx, stream, bh.config.ConsumerGroup, id)
		if err != nil {
			logger.Error("Failed to acknowledge message",
				logger.ErrorField(err),
				logger.String("stream", stream),
				logger.String("message_id", id),
			)
		}
	}
}

// incrementProcessed increments the processed bar counter
func (bh *BarFinalizationHandler) incrementProcessed(barTime time.Time) {
	bh.stats.mu.Lock()
	defer bh.stats.mu.Unlock()
	bh.stats.BarsProcessed++
	bh.stats.LastBarTime = barTime
}

// incrementAcked increments the acknowledged bar counter
func (bh *BarFinalizationHandler) incrementAcked(count int64) {
	bh.stats.mu.Lock()
	defer bh.stats.mu.Unlock()
	bh.stats.BarsAcked += count
}

// incrementFailed increments the failed bar counter
func (bh *BarFinalizationHandler) incrementFailed() {
	bh.stats.mu.Lock()
	defer bh.stats.mu.Unlock()
	bh.stats.BarsFailed++
}

