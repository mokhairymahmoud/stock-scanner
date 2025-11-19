package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/storage"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

// IndicatorConsumerConfig holds configuration for the indicator consumer
type IndicatorConsumerConfig struct {
	UpdateChannel      string        // Redis pub/sub channel (default: "indicators.updated")
	IndicatorKeyPrefix string        // Prefix for indicator keys (default: "ind:")
	FetchTimeout       time.Duration // Timeout for fetching indicators from Redis
	BatchSize          int           // Batch size for processing updates
}

// DefaultIndicatorConsumerConfig returns default configuration
func DefaultIndicatorConsumerConfig() IndicatorConsumerConfig {
	return IndicatorConsumerConfig{
		UpdateChannel:      "indicators.updated",
		IndicatorKeyPrefix: "ind:",
		FetchTimeout:       2 * time.Second,
		BatchSize:          100,
	}
}

// IndicatorConsumer consumes indicator updates from Redis pub/sub and updates symbol state
type IndicatorConsumer struct {
	config       IndicatorConsumerConfig
	redis        storage.RedisClient
	stateManager *StateManager
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	mu           sync.RWMutex
	running      bool
	stats        IndicatorConsumerStats
}

// IndicatorConsumerStats holds statistics about the indicator consumer
type IndicatorConsumerStats struct {
	UpdatesReceived  int64
	UpdatesProcessed int64
	UpdatesFailed    int64
	LastUpdateTime   time.Time
	mu               sync.RWMutex
}

// NewIndicatorConsumer creates a new indicator consumer
func NewIndicatorConsumer(redis storage.RedisClient, config IndicatorConsumerConfig, stateManager *StateManager) *IndicatorConsumer {
	if stateManager == nil {
		panic("stateManager cannot be nil")
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &IndicatorConsumer{
		config:       config,
		redis:        redis,
		stateManager: stateManager,
		ctx:          ctx,
		cancel:       cancel,
		stats:        IndicatorConsumerStats{},
	}
}

// Start starts consuming indicator updates
func (ic *IndicatorConsumer) Start() error {
	ic.mu.Lock()
	if ic.running {
		ic.mu.Unlock()
		return fmt.Errorf("indicator consumer is already running")
	}
	ic.running = true
	ic.mu.Unlock()

	logger.Info("Starting indicator consumer",
		logger.String("channel", ic.config.UpdateChannel),
		logger.String("indicator_prefix", ic.config.IndicatorKeyPrefix),
	)

	// Subscribe to indicator updates channel
	ic.wg.Add(1)
	go ic.consumeUpdates()

	return nil
}

// Stop stops the indicator consumer
func (ic *IndicatorConsumer) Stop() {
	ic.mu.Lock()
	if !ic.running {
		ic.mu.Unlock()
		return
	}
	ic.running = false
	ic.mu.Unlock()

	logger.Info("Stopping indicator consumer")

	ic.cancel()
	ic.wg.Wait()

	logger.Info("Indicator consumer stopped")
}

// IsRunning returns whether the consumer is running
func (ic *IndicatorConsumer) IsRunning() bool {
	ic.mu.RLock()
	defer ic.mu.RUnlock()
	return ic.running
}

// GetStats returns current consumer statistics
func (ic *IndicatorConsumer) GetStats() IndicatorConsumerStats {
	ic.stats.mu.RLock()
	defer ic.stats.mu.RUnlock()

	// Return a copy to avoid returning a struct with a mutex
	return IndicatorConsumerStats{
		UpdatesReceived:  ic.stats.UpdatesReceived,
		UpdatesProcessed: ic.stats.UpdatesProcessed,
		UpdatesFailed:    ic.stats.UpdatesFailed,
		LastUpdateTime:   ic.stats.LastUpdateTime,
	}
}

// consumeUpdates subscribes to indicator updates and processes them
func (ic *IndicatorConsumer) consumeUpdates() {
	defer ic.wg.Done()

	// Subscribe to indicator updates channel
	msgChan, err := ic.redis.Subscribe(ic.ctx, ic.config.UpdateChannel)
	if err != nil {
		logger.Error("Failed to subscribe to indicator updates",
			logger.ErrorField(err),
			logger.String("channel", ic.config.UpdateChannel),
		)
		return
	}

	logger.Info("Subscribed to indicator updates",
		logger.String("channel", ic.config.UpdateChannel),
	)

	// Process messages
	for {
		select {
		case <-ic.ctx.Done():
			return

		case msg, ok := <-msgChan:
			if !ok {
				logger.Warn("Indicator update channel closed")
				return
			}

			if msg.Channel != ic.config.UpdateChannel {
				continue // Ignore messages from other channels
			}

			ic.incrementReceived()

			// Parse update message
			// The message might be double-encoded (JSON string containing JSON)
			// Try to unmarshal directly first, if that fails, try unmarshaling as a string first
			var updateMsg map[string]interface{}
			messageBytes := []byte(msg.Message)

			// First, try direct unmarshal
			if err := json.Unmarshal(messageBytes, &updateMsg); err != nil {
				// If that fails, try unmarshaling as a string first (double-encoded case)
				var jsonStr string
				if err2 := json.Unmarshal(messageBytes, &jsonStr); err2 == nil {
					// Successfully unmarshaled as string, now unmarshal the inner JSON
					if err3 := json.Unmarshal([]byte(jsonStr), &updateMsg); err3 != nil {
						logger.Error("Failed to unmarshal indicator update message (double-encoded)",
							logger.ErrorField(err3),
							logger.String("message", msg.Message),
						)
						ic.incrementFailed()
						continue
					}
				} else {
					logger.Error("Failed to unmarshal indicator update message",
						logger.ErrorField(err),
						logger.String("message", msg.Message),
					)
					ic.incrementFailed()
					continue
				}
			}

			symbol, ok := updateMsg["symbol"].(string)
			if !ok || symbol == "" {
				logger.Warn("Invalid indicator update message: missing or invalid symbol",
					logger.Any("message", updateMsg),
				)
				ic.incrementFailed()
				continue
			}

			// Fetch full indicator data from Redis
			indicators, err := ic.fetchIndicators(symbol)
			if err != nil {
				logger.Error("Failed to fetch indicators",
					logger.ErrorField(err),
					logger.String("symbol", symbol),
				)
				ic.incrementFailed()
				continue
			}

			// Update state manager
			if err := ic.stateManager.UpdateIndicators(symbol, indicators); err != nil {
				logger.Error("Failed to update indicators in state manager",
					logger.ErrorField(err),
					logger.String("symbol", symbol),
				)
				ic.incrementFailed()
				continue
			}

			ic.incrementProcessed()
			logger.Debug("Updated indicators",
				logger.String("symbol", symbol),
				logger.Int("indicator_count", len(indicators)),
			)
		}
	}
}

// fetchIndicators fetches indicator values from Redis for a symbol
func (ic *IndicatorConsumer) fetchIndicators(symbol string) (map[string]float64, error) {
	key := fmt.Sprintf("%s%s", ic.config.IndicatorKeyPrefix, symbol)

	ctx, cancel := context.WithTimeout(ic.ctx, ic.config.FetchTimeout)
	defer cancel()

	// Fetch indicator data from Redis
	var indicatorData map[string]interface{}
	if err := ic.redis.GetJSON(ctx, key, &indicatorData); err != nil {
		return nil, fmt.Errorf("failed to get indicator data: %w", err)
	}

	// Extract values map
	valuesInterface, ok := indicatorData["values"]
	if !ok {
		return nil, fmt.Errorf("indicator data missing 'values' field")
	}

	valuesMap, ok := valuesInterface.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("indicator 'values' field is not a map")
	}

	// Convert to map[string]float64
	indicators := make(map[string]float64)
	for key, value := range valuesMap {
		switch v := value.(type) {
		case float64:
			indicators[key] = v
		case int:
			indicators[key] = float64(v)
		case int64:
			indicators[key] = float64(v)
		default:
			// Try to convert via JSON unmarshaling
			if jsonBytes, err := json.Marshal(v); err == nil {
				var floatVal float64
				if err := json.Unmarshal(jsonBytes, &floatVal); err == nil {
					indicators[key] = floatVal
				}
			}
		}
	}

	return indicators, nil
}

// incrementReceived increments the received update counter
func (ic *IndicatorConsumer) incrementReceived() {
	ic.stats.mu.Lock()
	defer ic.stats.mu.Unlock()
	ic.stats.UpdatesReceived++
}

// incrementProcessed increments the processed update counter
func (ic *IndicatorConsumer) incrementProcessed() {
	ic.stats.mu.Lock()
	defer ic.stats.mu.Unlock()
	ic.stats.UpdatesProcessed++
	ic.stats.LastUpdateTime = time.Now()
}

// incrementFailed increments the failed update counter
func (ic *IndicatorConsumer) incrementFailed() {
	ic.stats.mu.Lock()
	defer ic.stats.mu.Unlock()
	ic.stats.UpdatesFailed++
}
