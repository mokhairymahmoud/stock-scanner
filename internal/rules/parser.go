package rules

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

// ParseRule parses a JSON rule definition into a Rule struct
func ParseRule(data []byte) (*models.Rule, error) {
	var rule models.Rule

	if err := json.Unmarshal(data, &rule); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rule: %w", err)
	}

	// Set timestamps if not provided
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = time.Now()
	}
	if rule.UpdatedAt.IsZero() {
		rule.UpdatedAt = time.Now()
	}

	// Validate the parsed rule
	if err := ValidateRule(&rule); err != nil {
		return nil, fmt.Errorf("invalid rule: %w", err)
	}

	return &rule, nil
}

// ParseRuleFromReader parses a rule from an io.Reader
func ParseRuleFromReader(reader io.Reader) (*models.Rule, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read rule data: %w", err)
	}

	return ParseRule(data)
}

// ParseRuleFromString parses a rule from a JSON string
func ParseRuleFromString(jsonStr string) (*models.Rule, error) {
	return ParseRule([]byte(jsonStr))
}

// ParseRules parses multiple rules from JSON array
func ParseRules(data []byte) ([]*models.Rule, error) {
	var rules []*models.Rule

	if err := json.Unmarshal(data, &rules); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rules: %w", err)
	}

	// Validate each rule
	for i, rule := range rules {
		// Set timestamps if not provided
		if rule.CreatedAt.IsZero() {
			rule.CreatedAt = time.Now()
		}
		if rule.UpdatedAt.IsZero() {
			rule.UpdatedAt = time.Now()
		}

		if err := ValidateRule(rule); err != nil {
			return nil, fmt.Errorf("invalid rule at index %d: %w", i, err)
		}
	}

	return rules, nil
}

// ValidateRuleSyntax validates rule syntax without full validation
// Useful for checking JSON structure before full parsing
func ValidateRuleSyntax(data []byte) error {
	var rule map[string]interface{}

	if err := json.Unmarshal(data, &rule); err != nil {
		return fmt.Errorf("invalid JSON syntax: %w", err)
	}

	// Check required fields
	requiredFields := []string{"id", "name", "conditions"}
	for _, field := range requiredFields {
		if _, exists := rule[field]; !exists {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	// Check conditions is an array
	conditions, ok := rule["conditions"].([]interface{})
	if !ok {
		return fmt.Errorf("conditions must be an array")
	}

	if len(conditions) == 0 {
		return fmt.Errorf("conditions array cannot be empty")
	}

	return nil
}

// ValidateMetricReference validates that a metric name is valid
// This checks if the metric name follows the expected format
func ValidateMetricReference(metric string) error {
	if err := ValidateMetricName(metric); err != nil {
		return err
	}

	// Additional validation: check if metric is a known indicator or computed metric
	// For now, we just validate the format
	// Later, we can check against a registry of available metrics

	return nil
}

