package rules

import (
	"context"
	"fmt"

	"github.com/mohamedkhairy/stock-scanner/internal/storage"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

// RuleSyncService syncs rules from database to Redis cache
// This allows scanner workers to pick up rule updates via Redis
type RuleSyncService struct {
	dbStore    RuleStore
	redisStore *RedisRuleStore
	redis      storage.RedisClient
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewRuleSyncService creates a new rule sync service
func NewRuleSyncService(dbStore RuleStore, redisStore *RedisRuleStore, redis storage.RedisClient) *RuleSyncService {
	ctx, cancel := context.WithCancel(context.Background())
	return &RuleSyncService{
		dbStore:    dbStore,
		redisStore: redisStore,
		redis:      redis,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// SyncAllRules syncs all rules from database to Redis
func (s *RuleSyncService) SyncAllRules() error {
	rules, err := s.dbStore.GetAllRules()
	if err != nil {
		return fmt.Errorf("failed to get rules from database: %w", err)
	}

	logger.Info("Syncing rules to Redis",
		logger.Int("count", len(rules)),
	)

	// Sync each rule to Redis
	for _, rule := range rules {
		if err := s.redisStore.AddRule(rule); err != nil {
			logger.Warn("Failed to sync rule to Redis",
				logger.ErrorField(err),
				logger.String("rule_id", rule.ID),
			)
			// Continue with other rules
		}
	}

	return nil
}

// SyncRule syncs a single rule from database to Redis
func (s *RuleSyncService) SyncRule(ruleID string) error {
	rule, err := s.dbStore.GetRule(ruleID)
	if err != nil {
		return fmt.Errorf("failed to get rule from database: %w", err)
	}

	if err := s.redisStore.AddRule(rule); err != nil {
		return fmt.Errorf("failed to sync rule to Redis: %w", err)
	}

	return nil
}

// DeleteRuleFromRedis removes a rule from Redis
func (s *RuleSyncService) DeleteRuleFromRedis(ruleID string) error {
	if err := s.redisStore.DeleteRule(ruleID); err != nil {
		return fmt.Errorf("failed to delete rule from Redis: %w", err)
	}

	return nil
}

// Stop stops the sync service
func (s *RuleSyncService) Stop() {
	s.cancel()
}

