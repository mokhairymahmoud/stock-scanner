package rules

import (
	"fmt"
	"sync"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

// InMemoryRuleStore is an in-memory implementation of RuleStore
type InMemoryRuleStore struct {
	mu    sync.RWMutex
	rules map[string]*models.Rule
}

// NewInMemoryRuleStore creates a new in-memory rule store
func NewInMemoryRuleStore() *InMemoryRuleStore {
	return &InMemoryRuleStore{
		rules: make(map[string]*models.Rule),
	}
}

// GetRule retrieves a rule by ID
func (s *InMemoryRuleStore) GetRule(id string) (*models.Rule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rule, exists := s.rules[id]
	if !exists {
		return nil, fmt.Errorf("rule not found: %s", id)
	}

	// Return a copy to prevent external modifications
	return copyRule(rule), nil
}

// GetAllRules retrieves all rules
func (s *InMemoryRuleStore) GetAllRules() ([]*models.Rule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rules := make([]*models.Rule, 0, len(s.rules))
	for _, rule := range s.rules {
		rules = append(rules, copyRule(rule))
	}

	return rules, nil
}

// GetEnabledRules retrieves all enabled rules
func (s *InMemoryRuleStore) GetEnabledRules() ([]*models.Rule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rules := make([]*models.Rule, 0)
	for _, rule := range s.rules {
		if rule.Enabled {
			rules = append(rules, copyRule(rule))
		}
	}

	return rules, nil
}

// AddRule adds a new rule
func (s *InMemoryRuleStore) AddRule(rule *models.Rule) error {
	if rule == nil {
		return fmt.Errorf("rule cannot be nil")
	}

	// Validate rule
	if err := ValidateRule(rule); err != nil {
		return fmt.Errorf("invalid rule: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if rule already exists
	if _, exists := s.rules[rule.ID]; exists {
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

	// Store a copy
	s.rules[rule.ID] = copyRule(rule)

	return nil
}

// UpdateRule updates an existing rule
func (s *InMemoryRuleStore) UpdateRule(rule *models.Rule) error {
	if rule == nil {
		return fmt.Errorf("rule cannot be nil")
	}

	// Validate rule
	if err := ValidateRule(rule); err != nil {
		return fmt.Errorf("invalid rule: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if rule exists
	existing, exists := s.rules[rule.ID]
	if !exists {
		return fmt.Errorf("rule not found: %s", rule.ID)
	}

	// Preserve CreatedAt
	rule.CreatedAt = existing.CreatedAt
	// Update UpdatedAt
	rule.UpdatedAt = time.Now()

	// Store a copy
	s.rules[rule.ID] = copyRule(rule)

	return nil
}

// DeleteRule deletes a rule by ID
func (s *InMemoryRuleStore) DeleteRule(id string) error {
	if id == "" {
		return fmt.Errorf("rule ID cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.rules[id]; !exists {
		return fmt.Errorf("rule not found: %s", id)
	}

	delete(s.rules, id)

	return nil
}

// EnableRule enables a rule
func (s *InMemoryRuleStore) EnableRule(id string) error {
	return s.setRuleEnabled(id, true)
}

// DisableRule disables a rule
func (s *InMemoryRuleStore) DisableRule(id string) error {
	return s.setRuleEnabled(id, false)
}

// setRuleEnabled sets the enabled state of a rule
func (s *InMemoryRuleStore) setRuleEnabled(id string, enabled bool) error {
	if id == "" {
		return fmt.Errorf("rule ID cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	rule, exists := s.rules[id]
	if !exists {
		return fmt.Errorf("rule not found: %s", id)
	}

	// Update enabled state
	rule.Enabled = enabled
	rule.UpdatedAt = time.Now()

	return nil
}

// Count returns the number of rules in the store
func (s *InMemoryRuleStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.rules)
}

// Clear removes all rules from the store
func (s *InMemoryRuleStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.rules = make(map[string]*models.Rule)
}

// copyRule creates a deep copy of a rule
func copyRule(rule *models.Rule) *models.Rule {
	if rule == nil {
		return nil
	}

	copied := &models.Rule{
		ID:          rule.ID,
		Name:        rule.Name,
		Description: rule.Description,
		Conditions:  make([]models.Condition, len(rule.Conditions)),
		Cooldown:    rule.Cooldown,
		Enabled:     rule.Enabled,
		CreatedAt:   rule.CreatedAt,
		UpdatedAt:   rule.UpdatedAt,
	}

	// Copy conditions
	for i, cond := range rule.Conditions {
		copied.Conditions[i] = models.Condition{
			Metric:   cond.Metric,
			Operator: cond.Operator,
			Value:    cond.Value, // Value is interface{}, so this is a shallow copy
		}
	}

	return copied
}

