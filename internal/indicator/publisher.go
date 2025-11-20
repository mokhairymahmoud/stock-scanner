package indicator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
	"github.com/mohamedkhairy/stock-scanner/internal/toplist"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

// Publisher publishes indicators to Redis
type Publisher struct {
	redis           storage.RedisClient
	config          PublisherConfig
	toplistUpdater  toplist.ToplistUpdater // Optional toplist updater
	toplistEnabled  bool
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	mu              sync.RWMutex
	running         bool
	updateChannel   chan string // Channel for symbol updates
	toplistUpdates  []toplist.ToplistUpdate // Accumulated toplist updates
	lastToplistPublish time.Time
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
		redis:            redis,
		config:           config,
		ctx:              ctx,
		cancel:           cancel,
		updateChannel:    make(chan string, config.BatchSize),
		toplistUpdates:   make([]toplist.ToplistUpdate, 0, 100),
		lastToplistPublish: time.Now(),
	}
}

// SetToplistUpdater sets the toplist updater for indicator-based toplists
func (p *Publisher) SetToplistUpdater(updater toplist.ToplistUpdater, enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.toplistUpdater = updater
	p.toplistEnabled = enabled
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
	// Pass the map directly - Publish will handle JSON marshaling
	updateMsg := map[string]interface{}{
		"symbol":    symbol,
		"timestamp": time.Now().UTC(),
	}
	err = p.redis.Publish(ctx, p.config.UpdateChannel, updateMsg)
	if err != nil {
		logger.Warn("Failed to publish indicator update",
			logger.ErrorField(err),
			logger.String("symbol", symbol),
			logger.String("channel", p.config.UpdateChannel),
		)
		// Don't fail the whole operation if pub/sub fails
	}

	logger.Debug("Published indicators",
		logger.String("symbol", symbol),
		logger.Int("indicator_count", len(indicators)),
	)

	// Update toplists for complex metrics (RSI, Relative Volume, VWAP Distance)
	if p.toplistEnabled && p.toplistUpdater != nil {
		p.updateToplists(ctx, symbol, indicators)
	}

	return nil
}

// updateToplists updates toplists with indicator values
func (p *Publisher) updateToplists(ctx context.Context, symbol string, indicators map[string]float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Update RSI toplists
	if rsi, ok := indicators["rsi_14"]; ok {
		key := models.GetSystemToplistRedisKey(models.MetricRSI, models.Window1m)
		p.toplistUpdates = append(p.toplistUpdates, toplist.ToplistUpdate{
			Key:    key,
			Symbol: symbol,
			Value:  rsi,
		})
	}

	// Update relative volume toplists
	if relVol, ok := indicators["relative_volume_5m"]; ok {
		key := models.GetSystemToplistRedisKey(models.MetricRelativeVolume, models.Window5m)
		p.toplistUpdates = append(p.toplistUpdates, toplist.ToplistUpdate{
			Key:    key,
			Symbol: symbol,
			Value:  relVol,
		})
	}
	if relVol, ok := indicators["relative_volume_15m"]; ok {
		key := models.GetSystemToplistRedisKey(models.MetricRelativeVolume, models.Window15m)
		p.toplistUpdates = append(p.toplistUpdates, toplist.ToplistUpdate{
			Key:    key,
			Symbol: symbol,
			Value:  relVol,
		})
	}

	// Update VWAP distance toplists (distance from VWAP)
	if vwap, ok := indicators["vwap_5m"]; ok {
		if price, ok := indicators["close"]; ok {
			vwapDist := ((price - vwap) / vwap) * 100.0 // Percentage distance
			key := models.GetSystemToplistRedisKey(models.MetricVWAPDist, models.Window5m)
			p.toplistUpdates = append(p.toplistUpdates, toplist.ToplistUpdate{
				Key:    key,
				Symbol: symbol,
				Value:  vwapDist,
			})
		}
	}

	// Flush updates periodically (every update interval)
	if time.Since(p.lastToplistPublish) >= p.config.UpdateInterval {
		if len(p.toplistUpdates) > 0 {
			updates := p.toplistUpdates
			p.toplistUpdates = p.toplistUpdates[:0] // Clear but keep capacity
			p.lastToplistPublish = time.Now()

			// Flush in background to avoid blocking
			go func() {
				if err := p.toplistUpdater.BatchUpdate(ctx, updates); err != nil {
					logger.Warn("Failed to update toplists from indicators",
						logger.ErrorField(err),
						logger.Int("update_count", len(updates)),
					)
				}

				// Publish update notifications for system toplists
				systemToplists := []struct {
					metric models.ToplistMetric
					window models.ToplistTimeWindow
				}{
					{models.MetricRSI, models.Window1m},
					{models.MetricRelativeVolume, models.Window5m},
					{models.MetricRelativeVolume, models.Window15m},
					{models.MetricVWAPDist, models.Window5m},
				}

				for _, tl := range systemToplists {
					toplistID := string(models.GetSystemToplistType(tl.metric, tl.window, true))
					if err := p.toplistUpdater.PublishUpdate(ctx, toplistID, "system"); err != nil {
						logger.Debug("Failed to publish toplist update",
							logger.ErrorField(err),
							logger.String("toplist_id", toplistID),
						)
					}
				}
			}()
		}
	}
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
