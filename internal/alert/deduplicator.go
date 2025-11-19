package alert

import (
	"context"
	"fmt"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

// Deduplicator handles alert deduplication using idempotency keys
type Deduplicator struct {
	redis storage.RedisClient
	ttl   time.Duration
}

// NewDeduplicator creates a new deduplicator
func NewDeduplicator(redis storage.RedisClient, ttl time.Duration) *Deduplicator {
	return &Deduplicator{
		redis: redis,
		ttl:   ttl,
	}
}

// GenerateIdempotencyKey generates an idempotency key for an alert
// Format: {rule_id}:{symbol}:{timestamp_rounded_to_second}
func GenerateIdempotencyKey(alert *models.Alert) string {
	// Round timestamp to nearest second for idempotency
	roundedTime := alert.Timestamp.Truncate(time.Second)
	return fmt.Sprintf("%s:%s:%d", alert.RuleID, alert.Symbol, roundedTime.Unix())
}

// IsDuplicate checks if an alert is a duplicate based on its idempotency key
func (d *Deduplicator) IsDuplicate(ctx context.Context, alert *models.Alert) (bool, error) {
	key := GenerateIdempotencyKey(alert)
	redisKey := fmt.Sprintf("alert:dedupe:%s", key)

	// Check if key exists
	exists, err := d.redis.Exists(ctx, redisKey)
	if err != nil {
		return false, fmt.Errorf("failed to check duplicate: %w", err)
	}

	if exists {
		logger.Debug("Duplicate alert detected",
			logger.String("alert_id", alert.ID),
			logger.String("rule_id", alert.RuleID),
			logger.String("symbol", alert.Symbol),
			logger.String("idempotency_key", key),
		)
		return true, nil
	}

	// Set the key with TTL to mark as seen
	err = d.redis.Set(ctx, redisKey, alert.ID, d.ttl)
	if err != nil {
		logger.Warn("Failed to set deduplication key",
			logger.ErrorField(err),
			logger.String("alert_id", alert.ID),
		)
		// Don't fail the operation if we can't set the key
		// The alert will still be processed, but may result in duplicates
	}

	return false, nil
}

