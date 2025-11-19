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

// TickConsumer consumes ticks from Redis streams and updates symbol state
type TickConsumer struct {
	config       pubsub.StreamConsumerConfig
	redis        storage.RedisClient
	stateManager *StateManager
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	mu           sync.RWMutex
	running      bool
	stats        TickConsumerStats
}

// TickConsumerStats holds statistics about the tick consumer
type TickConsumerStats struct {
	TicksProcessed int64
	TicksAcked     int64
	TicksFailed    int64
	LastTickTime   time.Time
	Lag            int64
	mu             sync.RWMutex
}

// NewTickConsumer creates a new tick consumer
func NewTickConsumer(redis storage.RedisClient, config pubsub.StreamConsumerConfig, stateManager *StateManager) *TickConsumer {
	if stateManager == nil {
		panic("stateManager cannot be nil")
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &TickConsumer{
		config:       config,
		redis:        redis,
		stateManager: stateManager,
		ctx:          ctx,
		cancel:       cancel,
		stats:        TickConsumerStats{},
	}
}

// Start starts consuming ticks from the stream
func (tc *TickConsumer) Start() error {
	tc.mu.Lock()
	if tc.running {
		tc.mu.Unlock()
		return fmt.Errorf("tick consumer is already running")
	}
	tc.running = true
	tc.mu.Unlock()

	// Determine which streams to consume from (handle partitioning)
	streams := tc.getStreams()

	if len(streams) == 0 {
		return fmt.Errorf("no streams to consume from")
	}

	logger.Info("Starting tick consumer",
		logger.String("stream", tc.config.StreamName),
		logger.String("consumer_group", tc.config.ConsumerGroup),
		logger.String("consumer_name", tc.config.ConsumerName),
		logger.Int("streams", len(streams)),
	)

	// Start consuming from each stream
	for _, stream := range streams {
		tc.wg.Add(1)
		go tc.consumeStream(stream)
	}

	return nil
}

// Stop stops the tick consumer
func (tc *TickConsumer) Stop() {
	tc.mu.Lock()
	if !tc.running {
		tc.mu.Unlock()
		return
	}
	tc.running = false
	tc.mu.Unlock()

	logger.Info("Stopping tick consumer")

	tc.cancel()
	tc.wg.Wait()

	logger.Info("Tick consumer stopped")
}

// IsRunning returns whether the consumer is running
func (tc *TickConsumer) IsRunning() bool {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.running
}

// GetStats returns current consumer statistics
func (tc *TickConsumer) GetStats() TickConsumerStats {
	tc.stats.mu.RLock()
	defer tc.stats.mu.RUnlock()

	// Return a copy to avoid returning a struct with a mutex
	return TickConsumerStats{
		TicksProcessed: tc.stats.TicksProcessed,
		TicksAcked:     tc.stats.TicksAcked,
		TicksFailed:    tc.stats.TicksFailed,
		LastTickTime:   tc.stats.LastTickTime,
		Lag:            tc.stats.Lag,
	}
}

// getStreams returns the list of streams to consume from
// Handles partitioning if configured
func (tc *TickConsumer) getStreams() []string {
	if tc.config.Partitions <= 0 {
		// No partitioning - consume from single stream
		return []string{tc.config.StreamName}
	}

	// Partitioning: consume from multiple partition streams
	streams := make([]string, 0, tc.config.Partitions)
	for i := 0; i < tc.config.Partitions; i++ {
		streamName := fmt.Sprintf("%s:%d", tc.config.StreamName, i)
		streams = append(streams, streamName)
	}

	return streams
}

// consumeStream consumes messages from a single stream
func (tc *TickConsumer) consumeStream(stream string) {
	defer tc.wg.Done()

	messageChan, err := tc.redis.ConsumeFromStream(tc.ctx, stream, tc.config.ConsumerGroup, tc.config.ConsumerName)
	if err != nil {
		logger.Error("Failed to start consuming from stream",
			logger.ErrorField(err),
			logger.String("stream", stream),
		)
		return
	}

	batch := make([]storage.StreamMessage, 0, tc.config.BatchSize)
	ticker := time.NewTicker(tc.config.AckTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-tc.ctx.Done():
			// Process remaining batch before exiting
			if len(batch) > 0 {
				tc.processBatch(stream, batch)
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
			if len(batch) >= tc.config.BatchSize {
				tc.processBatch(stream, batch)
				batch = batch[:0] // Clear batch
			}

		case <-ticker.C:
			// Process batch on timeout
			if len(batch) > 0 {
				tc.processBatch(stream, batch)
				batch = batch[:0] // Clear batch
			}
		}
	}
}

// processBatch processes a batch of messages
func (tc *TickConsumer) processBatch(stream string, messages []storage.StreamMessage) {
	if len(messages) == 0 {
		return
	}

	processed := make([]string, 0, len(messages)) // Message IDs to acknowledge
	failed := make([]string, 0)                   // Message IDs that failed

	for _, msg := range messages {
		tick, err := tc.deserializeTick(msg)
		if err != nil {
			logger.Error("Failed to deserialize tick",
				logger.ErrorField(err),
				logger.String("stream", stream),
				logger.String("message_id", msg.ID),
			)
			failed = append(failed, msg.ID)
			tc.incrementFailed()
			continue
		}

		// Update state manager with tick
		err = tc.stateManager.UpdateLiveBar(tick.Symbol, tick)
		if err != nil {
			logger.Error("Failed to update live bar",
				logger.ErrorField(err),
				logger.String("symbol", tick.Symbol),
				logger.String("message_id", msg.ID),
			)
			failed = append(failed, msg.ID)
			tc.incrementFailed()
			continue
		}

		processed = append(processed, msg.ID)
		tc.incrementProcessed(tick.Timestamp)
	}

	// Acknowledge successfully processed messages
	if len(processed) > 0 {
		tc.acknowledgeMessages(stream, processed)
		tc.incrementAcked(int64(len(processed)))
	}

	// Log failed messages (they will be retried by consumer group)
	if len(failed) > 0 {
		logger.Warn("Some ticks failed to process",
			logger.Int("failed_count", len(failed)),
			logger.String("stream", stream),
		)
	}
}

// deserializeTick deserializes a stream message into a Tick
func (tc *TickConsumer) deserializeTick(msg storage.StreamMessage) (*models.Tick, error) {
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

	// Validate tick
	if err := tick.Validate(); err != nil {
		return nil, fmt.Errorf("invalid tick: %w", err)
	}

	return &tick, nil
}

// acknowledgeMessages acknowledges a batch of messages
func (tc *TickConsumer) acknowledgeMessages(stream string, messageIDs []string) {
	ctx, cancel := context.WithTimeout(context.Background(), tc.config.AckTimeout)
	defer cancel()

	for _, id := range messageIDs {
		err := tc.redis.AcknowledgeMessage(ctx, stream, tc.config.ConsumerGroup, id)
		if err != nil {
			logger.Error("Failed to acknowledge message",
				logger.ErrorField(err),
				logger.String("stream", stream),
				logger.String("message_id", id),
			)
		}
	}
}

// incrementProcessed increments the processed tick counter
func (tc *TickConsumer) incrementProcessed(tickTime time.Time) {
	tc.stats.mu.Lock()
	defer tc.stats.mu.Unlock()
	tc.stats.TicksProcessed++
	tc.stats.LastTickTime = tickTime
}

// incrementAcked increments the acknowledged tick counter
func (tc *TickConsumer) incrementAcked(count int64) {
	tc.stats.mu.Lock()
	defer tc.stats.mu.Unlock()
	tc.stats.TicksAcked += count
}

// incrementFailed increments the failed tick counter
func (tc *TickConsumer) incrementFailed() {
	tc.stats.mu.Lock()
	defer tc.stats.mu.Unlock()
	tc.stats.TicksFailed++
}

