package rules

import (
	"fmt"
	"reflect"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

// ValidateRule validates a rule with enhanced checks
func ValidateRule(rule *models.Rule) error {
	// Use base validation from models
	if err := rule.Validate(); err != nil {
		return err
	}

	// Additional validations
	if rule.Cooldown < 0 {
		return fmt.Errorf("cooldown must be non-negative, got %d", rule.Cooldown)
	}

	// Validate each condition
	for i, cond := range rule.Conditions {
		if err := ValidateCondition(&cond); err != nil {
			return fmt.Errorf("condition %d: %w", i, err)
		}
	}

	return nil
}

// ValidateCondition validates a condition with enhanced checks
func ValidateCondition(cond *models.Condition) error {
	// Use base validation from models
	if err := cond.Validate(); err != nil {
		return err
	}

	// Validate value type
	if cond.Value == nil {
		return fmt.Errorf("condition value cannot be nil")
	}

	// Value should be numeric for comparison operators
	valueType := reflect.TypeOf(cond.Value)
	switch valueType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		// Valid numeric type
		return nil
	case reflect.String:
		// String values are only valid for == and != operators
		if cond.Operator != "==" && cond.Operator != "!=" {
			return fmt.Errorf("string values only support == and != operators, got %s", cond.Operator)
		}
		return nil
	default:
		return fmt.Errorf("unsupported value type: %s", valueType.Kind())
	}
}

// ValidateMetricName validates that a metric name is well-formed
func ValidateMetricName(metric string) error {
	if metric == "" {
		return fmt.Errorf("metric name cannot be empty")
	}

	// Basic validation: metric names should be alphanumeric with underscores
	// This is a simple check - more complex validation can be added later
	for _, r := range metric {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return fmt.Errorf("metric name contains invalid character: %c", r)
		}
	}

	return nil
}

// ValidateOperator validates that an operator is supported
func ValidateOperator(op string) error {
	validOps := map[string]bool{
		">":  true,
		"<":  true,
		">=": true,
		"<=": true,
		"==": true,
		"!=": true,
	}

	if !validOps[op] {
		return fmt.Errorf("unsupported operator: %s (supported: >, <, >=, <=, ==, !=)", op)
	}

	return nil
}

