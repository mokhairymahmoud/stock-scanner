package indicator

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/storage"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

// Publisher publishes indicators to Redis
type Publisher struct {
	redis         storage.RedisClient
	config        PublisherConfig
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	mu            sync.RWMutex
	running       bool
	updateChannel chan string // Channel for symbol updates
}

// PublisherConfig holds configuration for the indicator publisher
type PublisherConfig struct {
	IndicatorKeyPrefix string        // Prefix for indicator keys (default: "ind:")
	IndicatorTTL       time.Duration // TTL for indicators (default: 10 minutes)
	UpdateChannel      string        // Redis pub/sub channel for indicator updates (default: "indicators.updated")
	UpdateInterval     time.Duration // How often to check for updates (default: 1 second)
	BatchSize          int           // Batch size for updates (default: 100)
}

// DefaultPublisherConfig returns default configuration
func DefaultPublisherConfig() PublisherConfig {
	return PublisherConfig{
		IndicatorKeyPrefix: "ind:",
		IndicatorTTL:       10 * time.Minute,
		UpdateChannel:      "indicators.updated",
		UpdateInterval:     1 * time.Second,
		BatchSize:          100,
	}
}

// NewPublisher creates a new indicator publisher
func NewPublisher(redis storage.RedisClient, config PublisherConfig) *Publisher {
	ctx, cancel := context.WithCancel(context.Background())

	return &Publisher{
		redis:         redis,
		config:        config,
		ctx:           ctx,
		cancel:        cancel,
		updateChannel: make(chan string, config.BatchSize),
	}
}

// Start starts the publisher
func (p *Publisher) Start() error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return fmt.Errorf("publisher is already running")
	}
	p.running = true
	p.mu.Unlock()

	logger.Info("Starting indicator publisher",
		logger.String("indicator_prefix", p.config.IndicatorKeyPrefix),
		logger.String("update_channel", p.config.UpdateChannel),
	)

	// Start update processor
	p.wg.Add(1)
	go p.updateProcessor()

	return nil
}

// Stop stops the publisher
func (p *Publisher) Stop() {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}
	p.running = false
	p.mu.Unlock()

	logger.Info("Stopping indicator publisher")
	p.cancel()
	close(p.updateChannel)
	p.wg.Wait()
	logger.Info("Indicator publisher stopped")
}

// PublishIndicators publishes indicator values for a symbol
func (p *Publisher) PublishIndicators(symbol string, indicators map[string]float64) error {
	if len(indicators) == 0 {
		return nil // Nothing to publish
	}

	key := fmt.Sprintf("%s%s", p.config.IndicatorKeyPrefix, symbol)

	// Create indicator data structure
	indicatorData := map[string]interface{}{
		"symbol":    symbol,
		"timestamp": time.Now().UTC(),
		"values":    indicators,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Publish to Redis key
	err := p.redis.Set(ctx, key, indicatorData, p.config.IndicatorTTL)
	if err != nil {
		logger.Error("Failed to publish indicators",
			logger.ErrorField(err),
			logger.String("symbol", symbol),
			logger.String("key", key),
		)
		return fmt.Errorf("failed to publish indicators: %w", err)
	}

	// Publish to pub/sub channel for real-time notifications
	updateMsg := map[string]interface{}{
		"symbol":    symbol,
		"timestamp": time.Now().UTC(),
	}
	updateJSON, err := json.Marshal(updateMsg)
	if err != nil {
		logger.Warn("Failed to marshal update message",
			logger.ErrorField(err),
			logger.String("symbol", symbol),
		)
	} else {
		err = p.redis.Publish(ctx, p.config.UpdateChannel, string(updateJSON))
		if err != nil {
			logger.Warn("Failed to publish indicator update",
				logger.ErrorField(err),
				logger.String("symbol", symbol),
				logger.String("channel", p.config.UpdateChannel),
			)
			// Don't fail the whole operation if pub/sub fails
		}
	}

	logger.Debug("Published indicators",
		logger.String("symbol", symbol),
		logger.Int("indicator_count", len(indicators)),
	)

	return nil
}

// QueueUpdate queues a symbol for indicator update
func (p *Publisher) QueueUpdate(symbol string) {
	select {
	case p.updateChannel <- symbol:
	default:
		// Channel full, log warning but don't block
		logger.Warn("Update channel full, dropping symbol",
			logger.String("symbol", symbol),
		)
	}
}

// updateProcessor processes queued indicator updates
func (p *Publisher) updateProcessor() {
	defer p.wg.Done()

	// This will be called by the engine when indicators are computed
	// For now, it's a placeholder for future batching logic
	ticker := time.NewTicker(p.config.UpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			// Process any pending updates
			// This is a placeholder - actual processing happens in PublishIndicators
		}
	}
}

// IsRunning returns whether the publisher is running
func (p *Publisher) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}
