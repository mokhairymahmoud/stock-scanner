package rules

import (
	"fmt"
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

func TestInMemoryRuleStore_AddRule(t *testing.T) {
	store := NewInMemoryRuleStore()

	rule := &models.Rule{
		ID:          "rule-1",
		Name:        "Test Rule",
		Description: "Test description",
		Conditions: []models.Condition{
			{Metric: "rsi_14", Operator: "<", Value: 30.0},
		},
		Cooldown: 300,
		Enabled:  true,
	}

	err := store.AddRule(rule)
	if err != nil {
		t.Fatalf("AddRule() error = %v", err)
	}

	// Try to add the same rule again (should fail)
	err = store.AddRule(rule)
	if err == nil {
		t.Error("Expected error when adding duplicate rule")
	}
}

func TestInMemoryRuleStore_GetRule(t *testing.T) {
	store := NewInMemoryRuleStore()

	rule := &models.Rule{
		ID:          "rule-1",
		Name:        "Test Rule",
		Conditions:  []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Cooldown:    300,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := store.AddRule(rule)
	if err != nil {
		t.Fatalf("AddRule() error = %v", err)
	}

	// Get the rule
	retrieved, err := store.GetRule("rule-1")
	if err != nil {
		t.Fatalf("GetRule() error = %v", err)
	}

	if retrieved.ID != rule.ID {
		t.Errorf("Expected ID %s, got %s", rule.ID, retrieved.ID)
	}

	if retrieved.Name != rule.Name {
		t.Errorf("Expected Name %s, got %s", rule.Name, retrieved.Name)
	}

	// Try to get non-existent rule
	_, err = store.GetRule("nonexistent")
	if err == nil {
		t.Error("Expected error when getting non-existent rule")
	}
}

func TestInMemoryRuleStore_GetAllRules(t *testing.T) {
	store := NewInMemoryRuleStore()

	rules := []*models.Rule{
		{
			ID:         "rule-1",
			Name:       "Rule 1",
			Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
			Enabled:    true,
		},
		{
			ID:         "rule-2",
			Name:       "Rule 2",
			Conditions: []models.Condition{{Metric: "ema_20", Operator: ">", Value: 150.0}},
			Enabled:    false,
		},
	}

	for _, rule := range rules {
		if err := store.AddRule(rule); err != nil {
			t.Fatalf("AddRule() error = %v", err)
		}
	}

	allRules, err := store.GetAllRules()
	if err != nil {
		t.Fatalf("GetAllRules() error = %v", err)
	}

	if len(allRules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(allRules))
	}
}

func TestInMemoryRuleStore_GetEnabledRules(t *testing.T) {
	store := NewInMemoryRuleStore()

	rules := []*models.Rule{
		{
			ID:         "rule-1",
			Name:       "Rule 1",
			Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
			Enabled:    true,
		},
		{
			ID:         "rule-2",
			Name:       "Rule 2",
			Conditions: []models.Condition{{Metric: "ema_20", Operator: ">", Value: 150.0}},
			Enabled:    false,
		},
		{
			ID:         "rule-3",
			Name:       "Rule 3",
			Conditions: []models.Condition{{Metric: "sma_50", Operator: ">", Value: 100.0}},
			Enabled:    true,
		},
	}

	for _, rule := range rules {
		if err := store.AddRule(rule); err != nil {
			t.Fatalf("AddRule() error = %v", err)
		}
	}

	enabledRules, err := store.GetEnabledRules()
	if err != nil {
		t.Fatalf("GetEnabledRules() error = %v", err)
	}

	if len(enabledRules) != 2 {
		t.Errorf("Expected 2 enabled rules, got %d", len(enabledRules))
	}
}

func TestInMemoryRuleStore_UpdateRule(t *testing.T) {
	store := NewInMemoryRuleStore()

	rule := &models.Rule{
		ID:         "rule-1",
		Name:       "Test Rule",
		Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Cooldown:   300,
		Enabled:    true,
	}

	err := store.AddRule(rule)
	if err != nil {
		t.Fatalf("AddRule() error = %v", err)
	}

	// Update the rule
	updatedRule := &models.Rule{
		ID:         "rule-1",
		Name:       "Updated Rule",
		Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 25.0}},
		Cooldown:   600,
		Enabled:    true,
	}

	err = store.UpdateRule(updatedRule)
	if err != nil {
		t.Fatalf("UpdateRule() error = %v", err)
	}

	// Verify update
	retrieved, err := store.GetRule("rule-1")
	if err != nil {
		t.Fatalf("GetRule() error = %v", err)
	}

	if retrieved.Name != "Updated Rule" {
		t.Errorf("Expected Name 'Updated Rule', got %s", retrieved.Name)
	}

	// Try to update non-existent rule
	nonExistent := &models.Rule{
		ID:         "nonexistent",
		Name:       "Test",
		Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
	}
	err = store.UpdateRule(nonExistent)
	if err == nil {
		t.Error("Expected error when updating non-existent rule")
	}
}

func TestInMemoryRuleStore_DeleteRule(t *testing.T) {
	store := NewInMemoryRuleStore()

	rule := &models.Rule{
		ID:         "rule-1",
		Name:       "Test Rule",
		Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Enabled:    true,
	}

	err := store.AddRule(rule)
	if err != nil {
		t.Fatalf("AddRule() error = %v", err)
	}

	// Delete the rule
	err = store.DeleteRule("rule-1")
	if err != nil {
		t.Fatalf("DeleteRule() error = %v", err)
	}

	// Try to get the deleted rule
	_, err = store.GetRule("rule-1")
	if err == nil {
		t.Error("Expected error when getting deleted rule")
	}

	// Try to delete non-existent rule
	err = store.DeleteRule("nonexistent")
	if err == nil {
		t.Error("Expected error when deleting non-existent rule")
	}
}

func TestInMemoryRuleStore_EnableDisableRule(t *testing.T) {
	store := NewInMemoryRuleStore()

	rule := &models.Rule{
		ID:         "rule-1",
		Name:       "Test Rule",
		Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Enabled:    false,
	}

	err := store.AddRule(rule)
	if err != nil {
		t.Fatalf("AddRule() error = %v", err)
	}

	// Enable the rule
	err = store.EnableRule("rule-1")
	if err != nil {
		t.Fatalf("EnableRule() error = %v", err)
	}

	retrieved, err := store.GetRule("rule-1")
	if err != nil {
		t.Fatalf("GetRule() error = %v", err)
	}
	if !retrieved.Enabled {
		t.Error("Expected rule to be enabled")
	}

	// Disable the rule
	err = store.DisableRule("rule-1")
	if err != nil {
		t.Fatalf("DisableRule() error = %v", err)
	}

	retrieved, err = store.GetRule("rule-1")
	if err != nil {
		t.Fatalf("GetRule() error = %v", err)
	}
	if retrieved.Enabled {
		t.Error("Expected rule to be disabled")
	}
}

func TestInMemoryRuleStore_Count(t *testing.T) {
	store := NewInMemoryRuleStore()

	if store.Count() != 0 {
		t.Errorf("Expected count 0, got %d", store.Count())
	}

	rule := &models.Rule{
		ID:         "rule-1",
		Name:       "Test Rule",
		Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Enabled:    true,
	}

	err := store.AddRule(rule)
	if err != nil {
		t.Fatalf("AddRule() error = %v", err)
	}

	if store.Count() != 1 {
		t.Errorf("Expected count 1, got %d", store.Count())
	}
}

func TestInMemoryRuleStore_Clear(t *testing.T) {
	store := NewInMemoryRuleStore()

	rule := &models.Rule{
		ID:         "rule-1",
		Name:       "Test Rule",
		Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Enabled:    true,
	}

	err := store.AddRule(rule)
	if err != nil {
		t.Fatalf("AddRule() error = %v", err)
	}

	store.Clear()

	if store.Count() != 0 {
		t.Errorf("Expected count 0 after clear, got %d", store.Count())
	}
}

func TestInMemoryRuleStore_Concurrency(t *testing.T) {
	store := NewInMemoryRuleStore()

	// Test concurrent access
	done := make(chan bool)
	go func() {
		for i := 0; i < 100; i++ {
			rule := &models.Rule{
				ID:         fmt.Sprintf("rule-%d", i),
				Name:       "Test Rule",
				Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
				Enabled:    true,
			}
			store.AddRule(rule)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			store.GetRule(fmt.Sprintf("rule-%d", i))
		}
		done <- true
	}()

	<-done
	<-done

	if store.Count() != 100 {
		t.Errorf("Expected count 100, got %d", store.Count())
	}
}

