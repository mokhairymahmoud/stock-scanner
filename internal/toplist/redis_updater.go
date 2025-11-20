package toplist

import (
	"context"
	"fmt"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

const (
	// ToplistUpdateChannel is the Redis pub/sub channel for toplist updates
	ToplistUpdateChannel = "toplists.updated"
	// DefaultToplistTTL is the default TTL for toplist ZSET keys (5 minutes)
	DefaultToplistTTL = 5 * time.Minute
)

// RedisToplistUpdater implements ToplistUpdater using Redis ZSETs
type RedisToplistUpdater struct {
	redisClient storage.RedisClient
}

// NewRedisToplistUpdater creates a new Redis-based toplist updater
func NewRedisToplistUpdater(redisClient storage.RedisClient) *RedisToplistUpdater {
	return &RedisToplistUpdater{
		redisClient: redisClient,
	}
}

// UpdateSystemToplist updates a system toplist
func (r *RedisToplistUpdater) UpdateSystemToplist(ctx context.Context, metric models.ToplistMetric, window models.ToplistTimeWindow, symbol string, value float64) error {
	key := models.GetSystemToplistRedisKey(metric, window)
	return r.updateZSet(ctx, key, symbol, value)
}

// UpdateUserToplist updates a user-custom toplist
func (r *RedisToplistUpdater) UpdateUserToplist(ctx context.Context, userID string, toplistID string, symbol string, value float64) error {
	key := models.GetUserToplistRedisKey(userID, toplistID)
	return r.updateZSet(ctx, key, symbol, value)
}

// updateZSet updates a Redis ZSET with a new score for a member
func (r *RedisToplistUpdater) updateZSet(ctx context.Context, key string, symbol string, value float64) error {
	err := r.redisClient.ZAdd(ctx, key, value, symbol)
	if err != nil {
		return fmt.Errorf("failed to update ZSET %s: %w", key, err)
	}

	// Set TTL on the key (refresh it each time we update)
	// Note: We don't have a SetTTL method in the interface, so we'll handle TTL
	// by setting it when we create the key. For now, we'll rely on Redis expiration
	// or handle it separately if needed.

	return nil
}

// BatchUpdate performs batch updates using Redis pipeline
func (r *RedisToplistUpdater) BatchUpdate(ctx context.Context, updates []ToplistUpdate) error {
	if len(updates) == 0 {
		return nil
	}

	// Group updates by key for efficient batching
	updatesByKey := make(map[string]map[string]float64)
	for _, update := range updates {
		if updatesByKey[update.Key] == nil {
			updatesByKey[update.Key] = make(map[string]float64)
		}
		updatesByKey[update.Key][update.Symbol] = update.Value
	}

	// Perform batch updates for each key
	for key, members := range updatesByKey {
		if err := r.redisClient.ZAddBatch(ctx, key, members); err != nil {
			logger.Warn("Failed to batch update toplist",
				logger.ErrorField(err),
				logger.String("key", key),
			)
			// Continue with other keys even if one fails
			continue
		}
	}

	return nil
}

// PublishUpdate publishes a toplist update notification
func (r *RedisToplistUpdater) PublishUpdate(ctx context.Context, toplistID string, toplistType string) error {
	update := map[string]interface{}{
		"toplist_id":   toplistID,
		"toplist_type": toplistType,
		"timestamp":    time.Now().Unix(),
	}

	err := r.redisClient.Publish(ctx, ToplistUpdateChannel, update)
	if err != nil {
		return fmt.Errorf("failed to publish toplist update: %w", err)
	}

	return nil
}

// Close closes the updater (no-op for Redis client, as it's shared)
func (r *RedisToplistUpdater) Close() error {
	// Redis client is shared, don't close it here
	return nil
}

