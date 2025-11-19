package rules

import (
	"fmt"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

// Compiler compiles rules into executable functions
type Compiler struct {
	resolver MetricResolver
}

// NewCompiler creates a new rule compiler
func NewCompiler(resolver MetricResolver) *Compiler {
	if resolver == nil {
		resolver = NewMetricResolver()
	}

	return &Compiler{
		resolver: resolver,
	}
}

// CompileRule compiles a rule into a CompiledRule function
func (c *Compiler) CompileRule(rule *models.Rule) (CompiledRule, error) {
	if rule == nil {
		return nil, fmt.Errorf("rule cannot be nil")
	}

	// Validate rule
	if err := ValidateRule(rule); err != nil {
		return nil, fmt.Errorf("invalid rule: %w", err)
	}

	// Store conditions for the compiled function
	conditions := rule.Conditions

	// Create compiled function
	compiled := func(symbol string, metrics map[string]float64) (bool, error) {
		// Evaluate all conditions (AND logic - all must be true)
		for i, cond := range conditions {
			matched, err := EvaluateCondition(&cond, c.resolver, metrics)
			if err != nil {
				return false, fmt.Errorf("condition %d (metric: %s): %w", i, cond.Metric, err)
			}

			// If any condition fails, the rule doesn't match
			if !matched {
				return false, nil
			}
		}

		// All conditions matched
		return true, nil
	}

	return compiled, nil
}

// CompileRules compiles multiple rules into CompiledRule functions
func (c *Compiler) CompileRules(rules []*models.Rule) (map[string]CompiledRule, error) {
	compiled := make(map[string]CompiledRule)

	for _, rule := range rules {
		compiledRule, err := c.CompileRule(rule)
		if err != nil {
			return nil, fmt.Errorf("failed to compile rule %s: %w", rule.ID, err)
		}

		compiled[rule.ID] = compiledRule
	}

	return compiled, nil
}

// CompileEnabledRules compiles only enabled rules
func (c *Compiler) CompileEnabledRules(rules []*models.Rule) (map[string]CompiledRule, error) {
	enabledRules := make([]*models.Rule, 0)

	for _, rule := range rules {
		if rule.Enabled {
			enabledRules = append(enabledRules, rule)
		}
	}

	return c.CompileRules(enabledRules)
}

