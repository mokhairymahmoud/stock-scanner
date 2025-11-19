package rules

import (
	"context"
	"fmt"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

const (
	// DefaultRedisRuleKeyPrefix is the default prefix for rule keys in Redis
	DefaultRedisRuleKeyPrefix = "rules:"
	// DefaultRedisRuleSetKey is the default key for the set of all rule IDs
	DefaultRedisRuleSetKey = "rules:ids"
	// DefaultRedisRuleTTL is the default TTL for rule keys (1 hour)
	DefaultRedisRuleTTL = 1 * time.Hour
)

// RedisRuleStoreConfig holds configuration for RedisRuleStore
type RedisRuleStoreConfig struct {
	KeyPrefix string        // Prefix for rule keys (default: "rules:")
	SetKey    string        // Key for the set of all rule IDs (default: "rules:ids")
	TTL       time.Duration // TTL for rule keys (default: 1 hour)
}

// DefaultRedisRuleStoreConfig returns default configuration
func DefaultRedisRuleStoreConfig() RedisRuleStoreConfig {
	return RedisRuleStoreConfig{
		KeyPrefix: DefaultRedisRuleKeyPrefix,
		SetKey:    DefaultRedisRuleSetKey,
		TTL:       DefaultRedisRuleTTL,
	}
}

// RedisRuleStore is a Redis-backed implementation of RuleStore
// Rules are stored as JSON in Redis keys with the pattern: rules:{rule_id}
// A Redis set maintains all rule IDs for efficient listing
type RedisRuleStore struct {
	redis  storage.RedisClient
	config RedisRuleStoreConfig
	ctx    context.Context
}

// NewRedisRuleStore creates a new Redis-backed rule store
func NewRedisRuleStore(redis storage.RedisClient, config RedisRuleStoreConfig) (*RedisRuleStore, error) {
	if redis == nil {
		return nil, fmt.Errorf("redis client cannot be nil")
	}

	if config.KeyPrefix == "" {
		config.KeyPrefix = DefaultRedisRuleKeyPrefix
	}
	if config.SetKey == "" {
		config.SetKey = DefaultRedisRuleSetKey
	}
	if config.TTL <= 0 {
		config.TTL = DefaultRedisRuleTTL
	}

	return &RedisRuleStore{
		redis:  redis,
		config: config,
		ctx:    context.Background(),
	}, nil
}

// GetRule retrieves a rule by ID from Redis
func (s *RedisRuleStore) GetRule(id string) (*models.Rule, error) {
	if id == "" {
		return nil, fmt.Errorf("rule ID cannot be empty")
	}

	key := s.config.KeyPrefix + id

	var rule models.Rule
	err := s.redis.GetJSON(s.ctx, key, &rule)
	if err != nil {
		// Check if key doesn't exist
		exists, existsErr := s.redis.Exists(s.ctx, key)
		if existsErr == nil && !exists {
			return nil, fmt.Errorf("rule not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get rule from Redis: %w", err)
	}

	// Validate rule
	if err := ValidateRule(&rule); err != nil {
		return nil, fmt.Errorf("invalid rule data in Redis: %w", err)
	}

	return &rule, nil
}

// GetAllRules retrieves all rules from Redis
func (s *RedisRuleStore) GetAllRules() ([]*models.Rule, error) {
	// Get all rule IDs from the set
	ruleIDs, err := s.redis.SetMembers(s.ctx, s.config.SetKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get rule IDs from Redis: %w", err)
	}

	if len(ruleIDs) == 0 {
		return []*models.Rule{}, nil
	}

	// Fetch all rules
	rules := make([]*models.Rule, 0, len(ruleIDs))
	for _, id := range ruleIDs {
		rule, err := s.GetRule(id)
		if err != nil {
			logger.Warn("Failed to get rule",
				logger.String("rule_id", id),
				logger.ErrorField(err),
			)
			continue // Skip invalid rules
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

// GetEnabledRules retrieves all enabled rules from Redis
func (s *RedisRuleStore) GetEnabledRules() ([]*models.Rule, error) {
	allRules, err := s.GetAllRules()
	if err != nil {
		return nil, err
	}

	enabledRules := make([]*models.Rule, 0)
	for _, rule := range allRules {
		if rule.Enabled {
			enabledRules = append(enabledRules, rule)
		}
	}

	return enabledRules, nil
}

// AddRule adds a new rule to Redis
func (s *RedisRuleStore) AddRule(rule *models.Rule) error {
	if rule == nil {
		return fmt.Errorf("rule cannot be nil")
	}

	// Validate rule
	if err := ValidateRule(rule); err != nil {
		return fmt.Errorf("invalid rule: %w", err)
	}

	// Check if rule already exists
	exists, err := s.redis.Exists(s.ctx, s.config.KeyPrefix+rule.ID)
	if err != nil {
		return fmt.Errorf("failed to check if rule exists: %w", err)
	}
	if exists {
		return fmt.Errorf("rule already exists: %s", rule.ID)
	}

	// Set timestamps if not set
	now := time.Now()
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = now
	}
	if rule.UpdatedAt.IsZero() {
		rule.UpdatedAt = now
	}

	// Store rule in Redis
	key := s.config.KeyPrefix + rule.ID
	err = s.redis.Set(s.ctx, key, rule, s.config.TTL)
	if err != nil {
		return fmt.Errorf("failed to store rule in Redis: %w", err)
	}

	// Add rule ID to the set
	err = s.redis.SetAdd(s.ctx, s.config.SetKey, rule.ID)
	if err != nil {
		// Try to clean up the rule key if set operation fails
		s.redis.Delete(s.ctx, key)
		return fmt.Errorf("failed to add rule ID to set: %w", err)
	}

	logger.Debug("Added rule to Redis",
		logger.String("rule_id", rule.ID),
		logger.String("rule_name", rule.Name),
	)

	return nil
}

// UpdateRule updates an existing rule in Redis
func (s *RedisRuleStore) UpdateRule(rule *models.Rule) error {
	if rule == nil {
		return fmt.Errorf("rule cannot be nil")
	}

	// Validate rule
	if err := ValidateRule(rule); err != nil {
		return fmt.Errorf("invalid rule: %w", err)
	}

	// Check if rule exists
	existing, err := s.GetRule(rule.ID)
	if err != nil {
		return fmt.Errorf("rule not found: %s", rule.ID)
	}

	// Preserve CreatedAt
	rule.CreatedAt = existing.CreatedAt
	// Update UpdatedAt
	rule.UpdatedAt = time.Now()

	// Update rule in Redis
	key := s.config.KeyPrefix + rule.ID
	err = s.redis.Set(s.ctx, key, rule, s.config.TTL)
	if err != nil {
		return fmt.Errorf("failed to update rule in Redis: %w", err)
	}

	logger.Debug("Updated rule in Redis",
		logger.String("rule_id", rule.ID),
		logger.String("rule_name", rule.Name),
	)

	return nil
}

// DeleteRule deletes a rule from Redis
func (s *RedisRuleStore) DeleteRule(id string) error {
	if id == "" {
		return fmt.Errorf("rule ID cannot be empty")
	}

	key := s.config.KeyPrefix + id

	// Check if rule exists
	exists, err := s.redis.Exists(s.ctx, key)
	if err != nil {
		return fmt.Errorf("failed to check if rule exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("rule not found: %s", id)
	}

	// Delete rule key
	err = s.redis.Delete(s.ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete rule from Redis: %w", err)
	}

	// Remove rule ID from set
	err = s.redis.SetRemove(s.ctx, s.config.SetKey, id)
	if err != nil {
		logger.Warn("Failed to remove rule ID from set",
			logger.String("rule_id", id),
			logger.ErrorField(err),
		)
		// Don't fail the operation if set removal fails
	}

	logger.Debug("Deleted rule from Redis",
		logger.String("rule_id", id),
	)

	return nil
}

// EnableRule enables a rule
func (s *RedisRuleStore) EnableRule(id string) error {
	return s.setRuleEnabled(id, true)
}

// DisableRule disables a rule
func (s *RedisRuleStore) DisableRule(id string) error {
	return s.setRuleEnabled(id, false)
}

// setRuleEnabled sets the enabled state of a rule
func (s *RedisRuleStore) setRuleEnabled(id string, enabled bool) error {
	if id == "" {
		return fmt.Errorf("rule ID cannot be empty")
	}

	rule, err := s.GetRule(id)
	if err != nil {
		return err
	}

	rule.Enabled = enabled
	rule.UpdatedAt = time.Now()

	// Update rule in Redis
	key := s.config.KeyPrefix + id
	err = s.redis.Set(s.ctx, key, rule, s.config.TTL)
	if err != nil {
		return fmt.Errorf("failed to update rule enabled state: %w", err)
	}

	logger.Debug("Updated rule enabled state",
		logger.String("rule_id", id),
		logger.Bool("enabled", enabled),
	)

	return nil
}
