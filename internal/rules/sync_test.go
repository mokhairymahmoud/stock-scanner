package rules

import (
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
)

func TestRuleSyncService_SyncRule(t *testing.T) {
	// Create mock stores
	dbStore := NewInMemoryRuleStore()
	redisClient := storage.NewMockRedisClient()
	redisConfig := DefaultRedisRuleStoreConfig()
	redisStore, err := NewRedisRuleStore(redisClient, redisConfig)
	if err != nil {
		t.Fatalf("Failed to create Redis rule store: %v", err)
	}

	syncService := NewRuleSyncService(dbStore, redisStore, redisClient)

	// Add rule to database store
	rule := &models.Rule{
		ID:          "rule-1",
		Name:        "Test Rule",
		Conditions:  []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Cooldown:    300,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err = dbStore.AddRule(rule)
	if err != nil {
		t.Fatalf("Failed to add rule to database: %v", err)
	}

	// Sync rule to Redis
	err = syncService.SyncRule("rule-1")
	if err != nil {
		t.Fatalf("Failed to sync rule: %v", err)
	}

	// Verify rule is in Redis store
	retrieved, err := redisStore.GetRule("rule-1")
	if err != nil {
		t.Fatalf("Failed to get rule from Redis: %v", err)
	}

	if retrieved.ID != rule.ID {
		t.Errorf("Expected rule ID %s, got %s", rule.ID, retrieved.ID)
	}
}

func TestRuleSyncService_SyncAllRules(t *testing.T) {
	// Create mock stores
	dbStore := NewInMemoryRuleStore()
	redisClient := storage.NewMockRedisClient()
	redisConfig := DefaultRedisRuleStoreConfig()
	redisStore, err := NewRedisRuleStore(redisClient, redisConfig)
	if err != nil {
		t.Fatalf("Failed to create Redis rule store: %v", err)
	}

	syncService := NewRuleSyncService(dbStore, redisStore, redisClient)

	// Add multiple rules to database store
	rules := []*models.Rule{
		{
			ID:          "rule-1",
			Name:        "Rule 1",
			Conditions:  []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
			Cooldown:    300,
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "rule-2",
			Name:        "Rule 2",
			Conditions:  []models.Condition{{Metric: "rsi_14", Operator: ">", Value: 70.0}},
			Cooldown:    600,
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	for _, rule := range rules {
		err = dbStore.AddRule(rule)
		if err != nil {
			t.Fatalf("Failed to add rule: %v", err)
		}
	}

	// Sync all rules
	err = syncService.SyncAllRules()
	if err != nil {
		t.Fatalf("Failed to sync all rules: %v", err)
	}

	// Verify all rules are in Redis
	for _, rule := range rules {
		retrieved, err := redisStore.GetRule(rule.ID)
		if err != nil {
			t.Errorf("Failed to get rule %s from Redis: %v", rule.ID, err)
			continue
		}

		if retrieved.ID != rule.ID {
			t.Errorf("Expected rule ID %s, got %s", rule.ID, retrieved.ID)
		}
	}
}

func TestRuleSyncService_DeleteRuleFromRedis(t *testing.T) {
	// Create mock stores
	dbStore := NewInMemoryRuleStore()
	redisClient := storage.NewMockRedisClient()
	redisConfig := DefaultRedisRuleStoreConfig()
	redisStore, err := NewRedisRuleStore(redisClient, redisConfig)
	if err != nil {
		t.Fatalf("Failed to create Redis rule store: %v", err)
	}

	syncService := NewRuleSyncService(dbStore, redisStore, redisClient)

	// Add rule to Redis
	rule := &models.Rule{
		ID:          "rule-1",
		Name:        "Test Rule",
		Conditions:  []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Cooldown:    300,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err = redisStore.AddRule(rule)
	if err != nil {
		t.Fatalf("Failed to add rule to Redis: %v", err)
	}

	// Delete rule from Redis
	err = syncService.DeleteRuleFromRedis("rule-1")
	if err != nil {
		t.Fatalf("Failed to delete rule from Redis: %v", err)
	}

	// Verify rule is deleted
	_, err = redisStore.GetRule("rule-1")
	if err == nil {
		t.Error("Expected error for deleted rule")
	}
}

