package rules

import (
	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

// CompiledRule is a compiled rule that can be evaluated against symbol state
// Returns true if the rule matches, false otherwise
type CompiledRule func(symbol string, metrics map[string]float64) (bool, error)

// RuleContext holds context for rule evaluation
type RuleContext struct {
	Symbol    string
	Metrics   map[string]float64
	Timestamp int64 // Unix timestamp
}

// EvaluationResult holds the result of rule evaluation
type EvaluationResult struct {
	Matched   bool
	RuleID    string
	RuleName  string
	Symbol    string
	Timestamp int64
	Metrics   map[string]float64 // Metrics that matched
	Error     error
}

// RuleStore defines the interface for storing and retrieving rules
type RuleStore interface {
	// GetRule retrieves a rule by ID
	GetRule(id string) (*models.Rule, error)

	// GetAllRules retrieves all enabled rules
	GetAllRules() ([]*models.Rule, error)

	// AddRule adds a new rule
	AddRule(rule *models.Rule) error

	// UpdateRule updates an existing rule
	UpdateRule(rule *models.Rule) error

	// DeleteRule deletes a rule by ID
	DeleteRule(id string) error

	// EnableRule enables a rule
	EnableRule(id string) error

	// DisableRule disables a rule
	DisableRule(id string) error
}

