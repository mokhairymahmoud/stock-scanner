package scanner

import (
	"context"
	"sync"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/toplist"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

// ToplistIntegration handles toplist updates from scanner worker
type ToplistIntegration struct {
	updater       toplist.ToplistUpdater
	enabled       bool
	updateInterval time.Duration
	lastPublish   time.Time
	updates       []toplist.ToplistUpdate
	mu            sync.RWMutex
}

// NewToplistIntegration creates a new toplist integration
func NewToplistIntegration(updater toplist.ToplistUpdater, enabled bool, updateInterval time.Duration) *ToplistIntegration {
	return &ToplistIntegration{
		updater:       updater,
		enabled:       enabled,
		updateInterval: updateInterval,
		lastPublish:   time.Now(),
		updates:       make([]toplist.ToplistUpdate, 0, 100),
	}
}

// UpdateToplists updates toplists with metrics from a symbol
// This accumulates updates and they are flushed by PublishUpdates
func (ti *ToplistIntegration) UpdateToplists(ctx context.Context, symbol string, metrics map[string]float64) error {
	if !ti.enabled {
		return nil
	}

	ti.mu.Lock()
	defer ti.mu.Unlock()

	// Update system toplists for change_pct
	if change1m, ok := metrics["price_change_1m_pct"]; ok {
		key := models.GetSystemToplistRedisKey(models.MetricChangePct, models.Window1m)
		ti.updates = append(ti.updates, toplist.ToplistUpdate{
			Key:    key,
			Symbol: symbol,
			Value:  change1m,
		})
	}
	if change5m, ok := metrics["price_change_5m_pct"]; ok {
		key := models.GetSystemToplistRedisKey(models.MetricChangePct, models.Window5m)
		ti.updates = append(ti.updates, toplist.ToplistUpdate{
			Key:    key,
			Symbol: symbol,
			Value:  change5m,
		})
	}
	if change15m, ok := metrics["price_change_15m_pct"]; ok {
		key := models.GetSystemToplistRedisKey(models.MetricChangePct, models.Window15m)
		ti.updates = append(ti.updates, toplist.ToplistUpdate{
			Key:    key,
			Symbol: symbol,
			Value:  change15m,
		})
	}

	// Update system toplists for volume (use finalized volume if available, otherwise live)
	if volume, ok := metrics["volume"]; ok {
		key := models.GetSystemToplistRedisKey(models.MetricVolume, models.Window1m)
		ti.updates = append(ti.updates, toplist.ToplistUpdate{
			Key:    key,
			Symbol: symbol,
			Value:  volume,
		})
	} else if volumeLive, ok := metrics["volume_live"]; ok {
		key := models.GetSystemToplistRedisKey(models.MetricVolume, models.Window1m)
		ti.updates = append(ti.updates, toplist.ToplistUpdate{
			Key:    key,
			Symbol: symbol,
			Value:  volumeLive,
		})
	}

	return nil
}

// PublishUpdates flushes accumulated updates and publishes toplist update notifications
func (ti *ToplistIntegration) PublishUpdates(ctx context.Context) error {
	if !ti.enabled {
		return nil
	}

	ti.mu.Lock()
	updates := ti.updates
	ti.updates = ti.updates[:0] // Clear but keep capacity
	shouldPublish := time.Since(ti.lastPublish) >= ti.updateInterval
	ti.mu.Unlock()

	// Batch update all accumulated updates
	if len(updates) > 0 {
		if err := ti.updater.BatchUpdate(ctx, updates); err != nil {
			logger.Warn("Failed to batch update toplists",
				logger.ErrorField(err),
				logger.Int("update_count", len(updates)),
			)
			// Continue to publish notifications even if batch update fails
		}
	}

	// Publish update notifications (throttled)
	if shouldPublish {
		// Publish updates for system toplists
		systemToplists := []struct {
			metric models.ToplistMetric
			window models.ToplistTimeWindow
		}{
			{models.MetricChangePct, models.Window1m},
			{models.MetricChangePct, models.Window5m},
			{models.MetricChangePct, models.Window15m},
			{models.MetricVolume, models.Window1m},
		}

		for _, tl := range systemToplists {
			toplistID := string(models.GetSystemToplistType(tl.metric, tl.window, true))
			if err := ti.updater.PublishUpdate(ctx, toplistID, "system"); err != nil {
				logger.Debug("Failed to publish toplist update",
					logger.ErrorField(err),
					logger.String("toplist_id", toplistID),
				)
			}
		}

		ti.mu.Lock()
		ti.lastPublish = time.Now()
		ti.mu.Unlock()
	}

	return nil
}

