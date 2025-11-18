package bars

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

// PublisherConfig holds configuration for the bar publisher
type PublisherConfig struct {
	LiveBarKeyPrefix   string        // Prefix for live bar keys (default: "livebar:")
	LiveBarTTL         time.Duration // TTL for live bars (default: 5 minutes)
	FinalizedStream    string        // Stream name for finalized bars (default: "bars.finalized")
	UpdateInterval     time.Duration // How often to update live bars (default: 1 second)
	BatchSize          int           // Batch size for finalized bars (default: 100)
	BatchTimeout       time.Duration // Timeout for batching finalized bars (default: 100ms)
}

// DefaultPublisherConfig returns default configuration
func DefaultPublisherConfig() PublisherConfig {
	return PublisherConfig{
		LiveBarKeyPrefix: "livebar:",
		LiveBarTTL:        5 * time.Minute,
		FinalizedStream:   "bars.finalized",
		UpdateInterval:    1 * time.Second,
		BatchSize:         100,
		BatchTimeout:      100 * time.Millisecond,
	}
}

// Publisher publishes live bars and finalized bars
type Publisher struct {
	config          PublisherConfig
	redis           storage.RedisClient
	barStorage      storage.BarStorage // Optional TimescaleDB storage
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	mu              sync.RWMutex
	running         bool
	finalizedBatch  []*models.Bar1m
	finalizedMu     sync.Mutex
	finalizedTicker *time.Ticker
}

// NewPublisher creates a new bar publisher
func NewPublisher(redis storage.RedisClient, config PublisherConfig) *Publisher {
	ctx, cancel := context.WithCancel(context.Background())

	return &Publisher{
		config:          config,
		redis:           redis,
		ctx:             ctx,
		cancel:          cancel,
		finalizedBatch:  make([]*models.Bar1m, 0, config.BatchSize),
		finalizedTicker: time.NewTicker(config.BatchTimeout),
	}
}

// SetBarStorage sets the bar storage (e.g., TimescaleDB) for persisting finalized bars
func (p *Publisher) SetBarStorage(barStorage storage.BarStorage) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.barStorage = barStorage
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

	logger.Info("Starting bar publisher",
		logger.String("live_bar_prefix", p.config.LiveBarKeyPrefix),
		logger.String("finalized_stream", p.config.FinalizedStream),
		logger.Duration("update_interval", p.config.UpdateInterval),
	)

	// Start batch processing goroutine for finalized bars
	p.wg.Add(1)
	go p.batchProcessor()

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

	logger.Info("Stopping bar publisher")
	p.cancel()
	p.finalizedTicker.Stop()

	// Flush remaining finalized bars
	p.flushFinalizedBars()

	p.wg.Wait()
	logger.Info("Bar publisher stopped")
}

// PublishLiveBar publishes a live bar snapshot to Redis
func (p *Publisher) PublishLiveBar(liveBar *models.LiveBar) error {
	if liveBar == nil {
		return fmt.Errorf("live bar cannot be nil")
	}

	key := fmt.Sprintf("%s%s", p.config.LiveBarKeyPrefix, liveBar.Symbol)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := p.redis.Set(ctx, key, liveBar, p.config.LiveBarTTL)
	if err != nil {
		logger.Error("Failed to publish live bar",
			logger.ErrorField(err),
			logger.String("symbol", liveBar.Symbol),
			logger.String("key", key),
		)
		return fmt.Errorf("failed to publish live bar: %w", err)
	}

	logger.Debug("Published live bar",
		logger.String("symbol", liveBar.Symbol),
		logger.Float64("close", liveBar.Close),
		logger.Int64("volume", liveBar.Volume),
	)

	return nil
}

// PublishFinalizedBar publishes a finalized bar to Redis Stream (batched)
func (p *Publisher) PublishFinalizedBar(bar *models.Bar1m) error {
	if bar == nil {
		return fmt.Errorf("bar cannot be nil")
	}

	if err := bar.Validate(); err != nil {
		logger.Warn("Invalid bar, skipping",
			logger.ErrorField(err),
			logger.String("symbol", bar.Symbol),
		)
		return fmt.Errorf("invalid bar: %w", err)
	}

	p.finalizedMu.Lock()
	p.finalizedBatch = append(p.finalizedBatch, bar)
	shouldFlush := len(p.finalizedBatch) >= p.config.BatchSize
	p.finalizedMu.Unlock()

	// Flush immediately if batch is full
	if shouldFlush {
		return p.flushFinalizedBars()
	}

	return nil
}

// batchProcessor periodically flushes the finalized bars batch
func (p *Publisher) batchProcessor() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-p.finalizedTicker.C:
			p.flushFinalizedBars()
		}
	}
}

// flushFinalizedBars flushes the current batch of finalized bars to Redis Stream
func (p *Publisher) flushFinalizedBars() error {
	p.finalizedMu.Lock()
	if len(p.finalizedBatch) == 0 {
		p.finalizedMu.Unlock()
		return nil
	}

	// Copy batch and clear
	batch := make([]*models.Bar1m, len(p.finalizedBatch))
	copy(batch, p.finalizedBatch)
	p.finalizedBatch = p.finalizedBatch[:0]
	p.finalizedMu.Unlock()

	if len(batch) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Prepare messages for Redis Stream
	messages := make([]map[string]interface{}, 0, len(batch))
	for _, bar := range batch {
		barJSON, err := json.Marshal(bar)
		if err != nil {
			logger.Error("Failed to marshal bar",
				logger.ErrorField(err),
				logger.String("symbol", bar.Symbol),
			)
			continue
		}

		messages = append(messages, map[string]interface{}{
			"bar": string(barJSON),
		})
	}

	if len(messages) == 0 {
		return nil
	}

	// Publish batch to stream
	err := p.redis.PublishBatchToStream(ctx, p.config.FinalizedStream, messages)
	if err != nil {
		logger.Error("Failed to publish finalized bars batch",
			logger.ErrorField(err),
			logger.String("stream", p.config.FinalizedStream),
			logger.Int("count", len(messages)),
		)
		return fmt.Errorf("failed to publish finalized bars: %w", err)
	}

	logger.Debug("Published finalized bars batch",
		logger.String("stream", p.config.FinalizedStream),
		logger.Int("count", len(messages)),
	)

	// Also write to TimescaleDB if configured
	p.mu.RLock()
	barStorage := p.barStorage
	p.mu.RUnlock()

	if barStorage != nil {
		// Write to database asynchronously (non-blocking)
		go func(bars []*models.Bar1m) {
			if err := barStorage.WriteBars(ctx, bars); err != nil {
				logger.Error("Failed to write bars to storage",
					logger.ErrorField(err),
					logger.Int("count", len(bars)),
				)
			}
		}(batch)
	}

	return nil
}

// GetLiveBar retrieves a live bar from Redis
func (p *Publisher) GetLiveBar(symbol string) (*models.LiveBar, error) {
	key := fmt.Sprintf("%s%s", p.config.LiveBarKeyPrefix, symbol)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var liveBar models.LiveBar
	err := p.redis.GetJSON(ctx, key, &liveBar)
	if err != nil {
		return nil, fmt.Errorf("failed to get live bar: %w", err)
	}

	return &liveBar, nil
}

// IsRunning returns whether the publisher is running
func (p *Publisher) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

