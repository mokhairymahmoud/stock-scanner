package rules

import (
	"testing"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

func TestCompiler_CompileRule(t *testing.T) {
	resolver := NewMetricResolver()
	compiler := NewCompiler(resolver)

	rule := &models.Rule{
		ID:          "rule-1",
		Name:        "RSI Oversold",
		Description: "Alert when RSI is below 30",
		Conditions: []models.Condition{
			{Metric: "rsi_14", Operator: "<", Value: 30.0},
		},
		Cooldown: 300,
		Enabled:  true,
	}

	compiled, err := compiler.CompileRule(rule)
	if err != nil {
		t.Fatalf("CompileRule() error = %v", err)
	}

	if compiled == nil {
		t.Fatal("CompileRule() returned nil")
	}

	// Test with matching metrics
	metrics := map[string]float64{"rsi_14": 25.0}
	matched, err := compiled("AAPL", metrics)
	if err != nil {
		t.Fatalf("compiled rule evaluation error = %v", err)
	}
	if !matched {
		t.Error("Expected rule to match, but it didn't")
	}

	// Test with non-matching metrics
	metrics = map[string]float64{"rsi_14": 35.0}
	matched, err = compiled("AAPL", metrics)
	if err != nil {
		t.Fatalf("compiled rule evaluation error = %v", err)
	}
	if matched {
		t.Error("Expected rule not to match, but it did")
	}
}

func TestCompiler_CompileRule_MultipleConditions(t *testing.T) {
	resolver := NewMetricResolver()
	compiler := NewCompiler(resolver)

	rule := &models.Rule{
		ID:   "rule-2",
		Name: "RSI and Volume",
		Conditions: []models.Condition{
			{Metric: "rsi_14", Operator: "<", Value: 30.0},
			{Metric: "volume_avg_5m", Operator: ">", Value: 1000000.0},
		},
		Enabled: true,
	}

	compiled, err := compiler.CompileRule(rule)
	if err != nil {
		t.Fatalf("CompileRule() error = %v", err)
	}

	// Test with both conditions matching
	metrics := map[string]float64{
		"rsi_14":        25.0,
		"volume_avg_5m": 2000000.0,
	}
	matched, err := compiled("AAPL", metrics)
	if err != nil {
		t.Fatalf("compiled rule evaluation error = %v", err)
	}
	if !matched {
		t.Error("Expected rule to match when both conditions are true")
	}

	// Test with first condition failing
	metrics = map[string]float64{
		"rsi_14":        35.0, // Doesn't match
		"volume_avg_5m": 2000000.0,
	}
	matched, err = compiled("AAPL", metrics)
	if err != nil {
		t.Fatalf("compiled rule evaluation error = %v", err)
	}
	if matched {
		t.Error("Expected rule not to match when first condition fails")
	}

	// Test with second condition failing
	metrics = map[string]float64{
		"rsi_14":        25.0,
		"volume_avg_5m": 500000.0, // Doesn't match
	}
	matched, err = compiled("AAPL", metrics)
	if err != nil {
		t.Fatalf("compiled rule evaluation error = %v", err)
	}
	if matched {
		t.Error("Expected rule not to match when second condition fails")
	}
}

func TestCompiler_CompileRule_InvalidRule(t *testing.T) {
	resolver := NewMetricResolver()
	compiler := NewCompiler(resolver)

	// Test with nil rule
	_, err := compiler.CompileRule(nil)
	if err == nil {
		t.Error("Expected error when compiling nil rule")
	}

	// Test with invalid rule (missing ID)
	invalidRule := &models.Rule{
		Name:       "Test",
		Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
	}
	_, err = compiler.CompileRule(invalidRule)
	if err == nil {
		t.Error("Expected error when compiling invalid rule")
	}
}

func TestCompiler_CompileRule_MissingMetric(t *testing.T) {
	resolver := NewMetricResolver()
	compiler := NewCompiler(resolver)

	rule := &models.Rule{
		ID:   "rule-1",
		Name: "Test",
		Conditions: []models.Condition{
			{Metric: "rsi_14", Operator: "<", Value: 30.0},
		},
		Enabled: true,
	}

	compiled, err := compiler.CompileRule(rule)
	if err != nil {
		t.Fatalf("CompileRule() error = %v", err)
	}

	// Test with missing metric
	metrics := map[string]float64{} // Empty metrics
	matched, err := compiled("AAPL", metrics)
	if err == nil {
		t.Error("Expected error when metric is missing")
	}
	if matched {
		t.Error("Expected rule not to match when metric is missing")
	}
}

func TestCompiler_CompileRules(t *testing.T) {
	resolver := NewMetricResolver()
	compiler := NewCompiler(resolver)

	rules := []*models.Rule{
		{
			ID:   "rule-1",
			Name: "Rule 1",
			Conditions: []models.Condition{
				{Metric: "rsi_14", Operator: "<", Value: 30.0},
			},
			Enabled: true,
		},
		{
			ID:   "rule-2",
			Name: "Rule 2",
			Conditions: []models.Condition{
				{Metric: "ema_20", Operator: ">", Value: 150.0},
			},
			Enabled: true,
		},
	}

	compiled, err := compiler.CompileRules(rules)
	if err != nil {
		t.Fatalf("CompileRules() error = %v", err)
	}

	if len(compiled) != 2 {
		t.Errorf("Expected 2 compiled rules, got %d", len(compiled))
	}

	if compiled["rule-1"] == nil {
		t.Error("Expected rule-1 to be compiled")
	}

	if compiled["rule-2"] == nil {
		t.Error("Expected rule-2 to be compiled")
	}
}

func TestCompiler_CompileEnabledRules(t *testing.T) {
	resolver := NewMetricResolver()
	compiler := NewCompiler(resolver)

	rules := []*models.Rule{
		{
			ID:   "rule-1",
			Name: "Rule 1",
			Conditions: []models.Condition{
				{Metric: "rsi_14", Operator: "<", Value: 30.0},
			},
			Enabled: true,
		},
		{
			ID:   "rule-2",
			Name: "Rule 2",
			Conditions: []models.Condition{
				{Metric: "ema_20", Operator: ">", Value: 150.0},
			},
			Enabled: false, // Disabled
		},
		{
			ID:   "rule-3",
			Name: "Rule 3",
			Conditions: []models.Condition{
				{Metric: "sma_50", Operator: ">", Value: 100.0},
			},
			Enabled: true,
		},
	}

	compiled, err := compiler.CompileEnabledRules(rules)
	if err != nil {
		t.Fatalf("CompileEnabledRules() error = %v", err)
	}

	if len(compiled) != 2 {
		t.Errorf("Expected 2 compiled rules (only enabled), got %d", len(compiled))
	}

	if compiled["rule-1"] == nil {
		t.Error("Expected rule-1 to be compiled")
	}

	if compiled["rule-2"] != nil {
		t.Error("Expected rule-2 NOT to be compiled (disabled)")
	}

	if compiled["rule-3"] == nil {
		t.Error("Expected rule-3 to be compiled")
	}
}

func TestNewCompiler(t *testing.T) {
	// Test with nil resolver (should create default)
	compiler := NewCompiler(nil)
	if compiler == nil {
		t.Fatal("NewCompiler() returned nil")
	}
	if compiler.resolver == nil {
		t.Error("Expected default resolver to be created")
	}

	// Test with custom resolver
	resolver := NewMetricResolver()
	compiler = NewCompiler(resolver)
	if compiler.resolver != resolver {
		t.Error("Expected custom resolver to be used")
	}
}

