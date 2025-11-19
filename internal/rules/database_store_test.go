package rules

import (
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/config"
	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

func TestDatabaseRuleStore_WithMock(t *testing.T) {
	// For unit tests, we'll use the in-memory store as a proxy
	// Integration tests would use actual database
	ruleStore := NewInMemoryRuleStore()

	rule := &models.Rule{
		ID:          "rule-1",
		Name:        "Test Rule",
		Description: "Test Description",
		Conditions:  []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Cooldown:    300,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Test AddRule
	err := ruleStore.AddRule(rule)
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	// Test GetRule
	retrieved, err := ruleStore.GetRule("rule-1")
	if err != nil {
		t.Fatalf("Failed to get rule: %v", err)
	}

	if retrieved.ID != rule.ID {
		t.Errorf("Expected rule ID %s, got %s", rule.ID, retrieved.ID)
	}

	if retrieved.Name != rule.Name {
		t.Errorf("Expected rule name %s, got %s", rule.Name, retrieved.Name)
	}

	// Test GetAllRules
	allRules, err := ruleStore.GetAllRules()
	if err != nil {
		t.Fatalf("Failed to get all rules: %v", err)
	}

	if len(allRules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(allRules))
	}

	// Test UpdateRule
	rule.Name = "Updated Name"
	rule.UpdatedAt = time.Now()
	err = ruleStore.UpdateRule(rule)
	if err != nil {
		t.Fatalf("Failed to update rule: %v", err)
	}

	retrieved, err = ruleStore.GetRule("rule-1")
	if err != nil {
		t.Fatalf("Failed to get updated rule: %v", err)
	}

	if retrieved.Name != "Updated Name" {
		t.Errorf("Expected rule name 'Updated Name', got %s", retrieved.Name)
	}

	// Test EnableRule/DisableRule
	err = ruleStore.DisableRule("rule-1")
	if err != nil {
		t.Fatalf("Failed to disable rule: %v", err)
	}

	retrieved, err = ruleStore.GetRule("rule-1")
	if err != nil {
		t.Fatalf("Failed to get disabled rule: %v", err)
	}

	if retrieved.Enabled {
		t.Error("Expected rule to be disabled")
	}

	err = ruleStore.EnableRule("rule-1")
	if err != nil {
		t.Fatalf("Failed to enable rule: %v", err)
	}

	retrieved, err = ruleStore.GetRule("rule-1")
	if err != nil {
		t.Fatalf("Failed to get enabled rule: %v", err)
	}

	if !retrieved.Enabled {
		t.Error("Expected rule to be enabled")
	}

	// Test DeleteRule
	err = ruleStore.DeleteRule("rule-1")
	if err != nil {
		t.Fatalf("Failed to delete rule: %v", err)
	}

	_, err = ruleStore.GetRule("rule-1")
	if err == nil {
		t.Error("Expected error for deleted rule")
	}
}

// TestDatabaseRuleStore_NewDatabaseRuleStore tests the constructor
// This would require a real database connection, so we'll skip it in unit tests
func TestDatabaseRuleStore_NewDatabaseRuleStore_InvalidConfig(t *testing.T) {
	invalidConfig := config.DatabaseConfig{
		Host:     "invalid-host",
		Port:     5432,
		User:     "test",
		Password: "test",
		Database: "test",
		SSLMode:  "disable",
	}

	_, err := NewDatabaseRuleStore(invalidConfig)
	if err == nil {
		t.Error("Expected error for invalid database config")
	}
}

