package rules

import (
	"testing"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
)

func TestRedisRuleStore_AddRule(t *testing.T) {
	mockRedis := storage.NewMockRedisClient()
	config := DefaultRedisRuleStoreConfig()
	store, err := NewRedisRuleStore(mockRedis, config)
	if err != nil {
		t.Fatalf("Failed to create RedisRuleStore: %v", err)
	}

	rule := &models.Rule{
		ID:         "rule-1",
		Name:       "Test Rule",
		Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Cooldown:   300,
		Enabled:    true,
	}

	err = store.AddRule(rule)
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	// Verify rule was stored
	retrieved, err := store.GetRule("rule-1")
	if err != nil {
		t.Fatalf("Failed to get rule: %v", err)
	}

	if retrieved.ID != rule.ID {
		t.Errorf("Expected rule ID %s, got %s", rule.ID, retrieved.ID)
	}
	if retrieved.Name != rule.Name {
		t.Errorf("Expected rule name %s, got %s", rule.Name, retrieved.Name)
	}
}

func TestRedisRuleStore_GetRule_NotFound(t *testing.T) {
	mockRedis := storage.NewMockRedisClient()
	config := DefaultRedisRuleStoreConfig()
	store, err := NewRedisRuleStore(mockRedis, config)
	if err != nil {
		t.Fatalf("Failed to create RedisRuleStore: %v", err)
	}

	_, err = store.GetRule("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent rule")
	}
}

func TestRedisRuleStore_UpdateRule(t *testing.T) {
	mockRedis := storage.NewMockRedisClient()
	config := DefaultRedisRuleStoreConfig()
	store, err := NewRedisRuleStore(mockRedis, config)
	if err != nil {
		t.Fatalf("Failed to create RedisRuleStore: %v", err)
	}

	rule := &models.Rule{
		ID:         "rule-1",
		Name:       "Test Rule",
		Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Cooldown:   300,
		Enabled:    true,
	}

	err = store.AddRule(rule)
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	// Update rule
	rule.Name = "Updated Rule"
	rule.Conditions = []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 25.0}}

	err = store.UpdateRule(rule)
	if err != nil {
		t.Fatalf("Failed to update rule: %v", err)
	}

	// Verify update
	retrieved, err := store.GetRule("rule-1")
	if err != nil {
		t.Fatalf("Failed to get rule: %v", err)
	}

	if retrieved.Name != "Updated Rule" {
		t.Errorf("Expected rule name 'Updated Rule', got %s", retrieved.Name)
	}
	if retrieved.Conditions[0].Value != 25.0 {
		t.Errorf("Expected condition value 25.0, got %v", retrieved.Conditions[0].Value)
	}
}

func TestRedisRuleStore_DeleteRule(t *testing.T) {
	mockRedis := storage.NewMockRedisClient()
	config := DefaultRedisRuleStoreConfig()
	store, err := NewRedisRuleStore(mockRedis, config)
	if err != nil {
		t.Fatalf("Failed to create RedisRuleStore: %v", err)
	}

	rule := &models.Rule{
		ID:         "rule-1",
		Name:       "Test Rule",
		Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Cooldown:   300,
		Enabled:    true,
	}

	err = store.AddRule(rule)
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	// Delete rule
	err = store.DeleteRule("rule-1")
	if err != nil {
		t.Fatalf("Failed to delete rule: %v", err)
	}

	// Verify deletion
	_, err = store.GetRule("rule-1")
	if err == nil {
		t.Error("Expected error for deleted rule")
	}
}

func TestRedisRuleStore_GetAllRules(t *testing.T) {
	mockRedis := storage.NewMockRedisClient()
	config := DefaultRedisRuleStoreConfig()
	store, err := NewRedisRuleStore(mockRedis, config)
	if err != nil {
		t.Fatalf("Failed to create RedisRuleStore: %v", err)
	}

	// Add multiple rules
	rules := []*models.Rule{
		{ID: "rule-1", Name: "Rule 1", Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}}, Cooldown: 300, Enabled: true},
		{ID: "rule-2", Name: "Rule 2", Conditions: []models.Condition{{Metric: "ema_20", Operator: ">", Value: 100.0}}, Cooldown: 600, Enabled: true},
		{ID: "rule-3", Name: "Rule 3", Conditions: []models.Condition{{Metric: "sma_50", Operator: "<", Value: 50.0}}, Cooldown: 300, Enabled: false},
	}

	for _, rule := range rules {
		err = store.AddRule(rule)
		if err != nil {
			t.Fatalf("Failed to add rule %s: %v", rule.ID, err)
		}
	}

	// Get all rules
	allRules, err := store.GetAllRules()
	if err != nil {
		t.Fatalf("Failed to get all rules: %v", err)
	}

	if len(allRules) != 3 {
		t.Errorf("Expected 3 rules, got %d", len(allRules))
	}
}

func TestRedisRuleStore_GetEnabledRules(t *testing.T) {
	mockRedis := storage.NewMockRedisClient()
	config := DefaultRedisRuleStoreConfig()
	store, err := NewRedisRuleStore(mockRedis, config)
	if err != nil {
		t.Fatalf("Failed to create RedisRuleStore: %v", err)
	}

	// Add rules with different enabled states
	rules := []*models.Rule{
		{ID: "rule-1", Name: "Rule 1", Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}}, Cooldown: 300, Enabled: true},
		{ID: "rule-2", Name: "Rule 2", Conditions: []models.Condition{{Metric: "ema_20", Operator: ">", Value: 100.0}}, Cooldown: 600, Enabled: false},
		{ID: "rule-3", Name: "Rule 3", Conditions: []models.Condition{{Metric: "sma_50", Operator: "<", Value: 50.0}}, Cooldown: 300, Enabled: true},
	}

	for _, rule := range rules {
		err = store.AddRule(rule)
		if err != nil {
			t.Fatalf("Failed to add rule %s: %v", rule.ID, err)
		}
	}

	// Get enabled rules
	enabledRules, err := store.GetEnabledRules()
	if err != nil {
		t.Fatalf("Failed to get enabled rules: %v", err)
	}

	if len(enabledRules) != 2 {
		t.Errorf("Expected 2 enabled rules, got %d", len(enabledRules))
	}
}

func TestRedisRuleStore_EnableDisableRule(t *testing.T) {
	mockRedis := storage.NewMockRedisClient()
	config := DefaultRedisRuleStoreConfig()
	store, err := NewRedisRuleStore(mockRedis, config)
	if err != nil {
		t.Fatalf("Failed to create RedisRuleStore: %v", err)
	}

	rule := &models.Rule{
		ID:         "rule-1",
		Name:       "Test Rule",
		Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Cooldown:   300,
		Enabled:    false,
	}

	err = store.AddRule(rule)
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	// Enable rule
	err = store.EnableRule("rule-1")
	if err != nil {
		t.Fatalf("Failed to enable rule: %v", err)
	}

	retrieved, err := store.GetRule("rule-1")
	if err != nil {
		t.Fatalf("Failed to get rule: %v", err)
	}
	if !retrieved.Enabled {
		t.Error("Expected rule to be enabled")
	}

	// Disable rule
	err = store.DisableRule("rule-1")
	if err != nil {
		t.Fatalf("Failed to disable rule: %v", err)
	}

	retrieved, err = store.GetRule("rule-1")
	if err != nil {
		t.Fatalf("Failed to get rule: %v", err)
	}
	if retrieved.Enabled {
		t.Error("Expected rule to be disabled")
	}
}

func TestRedisRuleStore_NewRedisRuleStore_NilRedis(t *testing.T) {
	config := DefaultRedisRuleStoreConfig()
	_, err := NewRedisRuleStore(nil, config)
	if err == nil {
		t.Error("Expected error for nil Redis client")
	}
}

func TestRedisRuleStore_DefaultConfig(t *testing.T) {
	config := DefaultRedisRuleStoreConfig()
	if config.KeyPrefix != DefaultRedisRuleKeyPrefix {
		t.Errorf("Expected key prefix %s, got %s", DefaultRedisRuleKeyPrefix, config.KeyPrefix)
	}
	if config.SetKey != DefaultRedisRuleSetKey {
		t.Errorf("Expected set key %s, got %s", DefaultRedisRuleSetKey, config.SetKey)
	}
	if config.TTL != DefaultRedisRuleTTL {
		t.Errorf("Expected TTL %v, got %v", DefaultRedisRuleTTL, config.TTL)
	}
}

func TestRedisRuleStore_EmptyConfig(t *testing.T) {
	mockRedis := storage.NewMockRedisClient()
	config := RedisRuleStoreConfig{} // Empty config
	store, err := NewRedisRuleStore(mockRedis, config)
	if err != nil {
		t.Fatalf("Failed to create RedisRuleStore with empty config: %v", err)
	}

	// Should use defaults
	if store.config.KeyPrefix != DefaultRedisRuleKeyPrefix {
		t.Errorf("Expected default key prefix, got %s", store.config.KeyPrefix)
	}
}

