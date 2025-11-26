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
	updater        toplist.ToplistUpdater
	store          toplist.ToplistStore
	mapper         *toplist.MetricMapper
	enabled        bool
	updateInterval time.Duration
	reloadInterval time.Duration
	lastPublish    time.Time
	lastReload     time.Time
	updates        []toplist.ToplistUpdate
	toplists       []*models.ToplistConfig // Cached enabled toplists
	mu             sync.RWMutex
}

// NewToplistIntegration creates a new toplist integration
func NewToplistIntegration(updater toplist.ToplistUpdater, store toplist.ToplistStore, enabled bool, updateInterval time.Duration) *ToplistIntegration {
	return &ToplistIntegration{
		updater:        updater,
		store:          store,
		mapper:         toplist.NewMetricMapper(),
		enabled:        enabled,
		updateInterval: updateInterval,
		reloadInterval: 30 * time.Second, // Reload toplists every 30 seconds
		lastPublish:    time.Now(),
		lastReload:     time.Time{}, // Will trigger immediate reload
		updates:        make([]toplist.ToplistUpdate, 0, 100),
		toplists:       make([]*models.ToplistConfig, 0),
	}
}

// reloadToplists reloads enabled toplists from the store
func (ti *ToplistIntegration) reloadToplists(ctx context.Context) error {
	// Load all enabled toplists (system and user)
	toplists, err := ti.store.GetEnabledToplists(ctx, "")
	if err != nil {
		return err
	}

	ti.mu.Lock()
	ti.toplists = toplists
	ti.lastReload = time.Now()
	ti.mu.Unlock()

	logger.Info("Reloaded toplists",
		logger.Int("count", len(toplists)),
	)
	if len(toplists) > 0 {
		for _, tl := range toplists {
			logger.Info("Loaded toplist",
				logger.String("id", tl.ID),
				logger.String("name", tl.Name),
				logger.String("user_id", tl.UserID),
				logger.String("metric", string(tl.Metric)),
				logger.String("time_window", string(tl.TimeWindow)),
				logger.Bool("enabled", tl.Enabled),
			)
		}
	}
	return nil
}

// UpdateToplists updates toplists with metrics from a symbol
// This accumulates updates and they are flushed by PublishUpdates
func (ti *ToplistIntegration) UpdateToplists(ctx context.Context, symbol string, metrics map[string]float64) error {
	if !ti.enabled {
		return nil
	}

	// Reload toplists periodically
	ti.mu.RLock()
	needsReload := time.Since(ti.lastReload) >= ti.reloadInterval
	toplists := ti.toplists
	ti.mu.RUnlock()

	if needsReload || len(toplists) == 0 {
		logger.Info("Reloading toplists",
			logger.Bool("needs_reload", needsReload),
			logger.Int("cached_count", len(toplists)),
		)
		if err := ti.reloadToplists(ctx); err != nil {
			logger.Warn("Failed to reload toplists",
				logger.ErrorField(err),
			)
			// Continue with cached toplists if reload fails
		} else {
			ti.mu.RLock()
			toplists = ti.toplists
			ti.mu.RUnlock()
			logger.Info("Toplists reloaded, now have",
				logger.Int("count", len(toplists)),
			)
		}
	}

	ti.mu.Lock()
	defer ti.mu.Unlock()

	logger.Info("Updating toplists for symbol",
		logger.String("symbol", symbol),
		logger.Int("toplist_count", len(toplists)),
		logger.Int("metrics_count", len(metrics)),
	)

	// Update all matching toplists dynamically
	for _, config := range toplists {
		// Get metric value for this toplist config
		value, found := ti.mapper.GetMetricValue(config, metrics)
		if !found {
			logger.Debug("Metric not found for toplist",
				logger.String("toplist_id", config.ID),
				logger.String("toplist_name", config.Name),
				logger.String("metric", string(config.Metric)),
				logger.String("time_window", string(config.TimeWindow)),
				logger.String("symbol", symbol),
			)
			continue
		}

		// Get Redis key for this toplist
		key := ti.mapper.GetToplistRedisKey(config)
		logger.Debug("Adding toplist update",
			logger.String("toplist_id", config.ID),
			logger.String("key", key),
			logger.String("symbol", symbol),
			logger.Float64("value", value),
		)
		ti.updates = append(ti.updates, toplist.ToplistUpdate{
			Key:    key,
			Symbol: symbol,
			Value:  value,
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
		logger.Info("Batch updating toplists",
			logger.Int("update_count", len(updates)),
		)
		for _, update := range updates {
			logger.Debug("Toplist update",
				logger.String("key", update.Key),
				logger.String("symbol", update.Symbol),
				logger.Float64("value", update.Value),
			)
		}
		if err := ti.updater.BatchUpdate(ctx, updates); err != nil {
			logger.Warn("Failed to batch update toplists",
				logger.ErrorField(err),
				logger.Int("update_count", len(updates)),
			)
			// Continue to publish notifications even if batch update fails
		} else {
			logger.Info("Successfully batch updated toplists",
				logger.Int("update_count", len(updates)),
			)
		}
	} else {
		logger.Debug("No toplist updates to batch",
			logger.Int("cached_toplist_count", len(ti.toplists)),
		)
	}

	// Publish update notifications (throttled)
	if shouldPublish {
		ti.mu.RLock()
		toplists := ti.toplists
		ti.mu.RUnlock()

		// Publish updates for all enabled toplists
		for _, config := range toplists {
			toplistType := "user"
			if config.IsSystemToplist() {
				toplistType = "system"
			}

			if err := ti.updater.PublishUpdate(ctx, config.ID, toplistType); err != nil {
				logger.Debug("Failed to publish toplist update",
					logger.ErrorField(err),
					logger.String("toplist_id", config.ID),
				)
			}
		}

		ti.mu.Lock()
		ti.lastPublish = time.Now()
		ti.mu.Unlock()
	}

	return nil
}
