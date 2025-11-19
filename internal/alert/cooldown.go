package alert

import (
	"context"
	"fmt"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

// CooldownManager handles per-user, per-rule cooldowns
// For MVP, we'll use a simple per-rule, per-symbol cooldown
// In the future, this will be per-user, per-rule
type CooldownManager struct {
	redis storage.RedisClient
	ttl   time.Duration
}

// NewCooldownManager creates a new cooldown manager
func NewCooldownManager(redis storage.RedisClient, ttl time.Duration) *CooldownManager {
	return &CooldownManager{
		redis: redis,
		ttl:   ttl,
	}
}

// GenerateCooldownKey generates a cooldown key for an alert
// Format: cooldown:{user_id}:{rule_id}:{symbol}
// For MVP, we'll use a global user_id "default" or per-rule, per-symbol
func GenerateCooldownKey(alert *models.Alert, userID string) string {
	if userID == "" {
		// MVP: Use rule_id and symbol for cooldown
		return fmt.Sprintf("cooldown:default:%s:%s", alert.RuleID, alert.Symbol)
	}
	return fmt.Sprintf("cooldown:%s:%s:%s", userID, alert.RuleID, alert.Symbol)
}

// IsInCooldown checks if an alert is in cooldown period
func (c *CooldownManager) IsInCooldown(ctx context.Context, alert *models.Alert, userID string) (bool, error) {
	key := GenerateCooldownKey(alert, userID)

	// Check if key exists
	exists, err := c.redis.Exists(ctx, key)
	if err != nil {
		return false, fmt.Errorf("failed to check cooldown: %w", err)
	}

	if exists {
		logger.Debug("Alert in cooldown period",
			logger.String("alert_id", alert.ID),
			logger.String("rule_id", alert.RuleID),
			logger.String("symbol", alert.Symbol),
			logger.String("user_id", userID),
		)
		return true, nil
	}

	return false, nil
}

// SetCooldown sets a cooldown period for an alert
func (c *CooldownManager) SetCooldown(ctx context.Context, alert *models.Alert, userID string) error {
	key := GenerateCooldownKey(alert, userID)

	// Set the key with TTL
	err := c.redis.Set(ctx, key, alert.ID, c.ttl)
	if err != nil {
		return fmt.Errorf("failed to set cooldown: %w", err)
	}

	logger.Debug("Set cooldown for alert",
		logger.String("alert_id", alert.ID),
		logger.String("rule_id", alert.RuleID),
		logger.String("symbol", alert.Symbol),
		logger.String("user_id", userID),
		logger.Duration("ttl", c.ttl),
	)

	return nil
}

// CheckAndSetCooldown checks if in cooldown and sets it if not
// Returns true if alert should be suppressed (in cooldown)
func (c *CooldownManager) CheckAndSetCooldown(ctx context.Context, alert *models.Alert, userID string) (bool, error) {
	inCooldown, err := c.IsInCooldown(ctx, alert, userID)
	if err != nil {
		return false, err
	}

	if inCooldown {
		return true, nil
	}

	// Set cooldown for future alerts
	err = c.SetCooldown(ctx, alert, userID)
	if err != nil {
		logger.Warn("Failed to set cooldown",
			logger.ErrorField(err),
			logger.String("alert_id", alert.ID),
		)
		// Don't fail the operation
	}

	return false, nil
}

