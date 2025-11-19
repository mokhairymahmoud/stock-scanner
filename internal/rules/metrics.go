package rules

import (
	"fmt"
	"math"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

// MetricResolver resolves metric names to their values
type MetricResolver interface {
	// ResolveMetric resolves a metric name to its numeric value
	ResolveMetric(metric string, metrics map[string]float64) (float64, error)
}

// DefaultMetricResolver is the default implementation of MetricResolver
type DefaultMetricResolver struct {
	computedMetrics map[string]ComputedMetricFunc
}

// ComputedMetricFunc is a function that computes a metric value from available metrics
type ComputedMetricFunc func(metrics map[string]float64) (float64, error)

// NewMetricResolver creates a new metric resolver
func NewMetricResolver() *DefaultMetricResolver {
	resolver := &DefaultMetricResolver{
		computedMetrics: make(map[string]ComputedMetricFunc),
	}

	// Register built-in computed metrics
	resolver.registerBuiltInMetrics()

	return resolver
}

// RegisterComputedMetric registers a computed metric function
func (r *DefaultMetricResolver) RegisterComputedMetric(name string, fn ComputedMetricFunc) error {
	if name == "" {
		return fmt.Errorf("metric name cannot be empty")
	}
	if fn == nil {
		return fmt.Errorf("computed metric function cannot be nil")
	}

	if err := ValidateMetricName(name); err != nil {
		return fmt.Errorf("invalid metric name: %w", err)
	}

	r.computedMetrics[name] = fn
	return nil
}

// ResolveMetric resolves a metric name to its numeric value
func (r *DefaultMetricResolver) ResolveMetric(metric string, metrics map[string]float64) (float64, error) {
	if metric == "" {
		return 0, fmt.Errorf("metric name cannot be empty")
	}

	// First, try direct lookup in metrics map
	if value, exists := metrics[metric]; exists {
		return value, nil
	}

	// Then, try computed metrics
	if computedFn, exists := r.computedMetrics[metric]; exists {
		return computedFn(metrics)
	}

	// Metric not found
	return 0, fmt.Errorf("metric '%s' not found", metric)
}

// registerBuiltInMetrics registers built-in computed metrics
func (r *DefaultMetricResolver) registerBuiltInMetrics() {
	// Price change metrics are computed from finalized bars (handled in scanner)
	// Volume ratios are computed from current volume vs average (handled in scanner)
	// For now, we register placeholders - actual computation happens in scanner

	// Example: price_change_5m_pct would be computed from bars
	// This is handled in the scanner worker, not here
	// The resolver just looks up values that are already computed
}

// EvaluateCondition evaluates a condition against metrics
func EvaluateCondition(cond *models.Condition, resolver MetricResolver, metrics map[string]float64) (bool, error) {
	if cond == nil {
		return false, fmt.Errorf("condition cannot be nil")
	}

	// Resolve metric value
	metricValue, err := resolver.ResolveMetric(cond.Metric, metrics)
	if err != nil {
		return false, fmt.Errorf("failed to resolve metric '%s': %w", cond.Metric, err)
	}

	// Get comparison value
	comparisonValue, err := getNumericValue(cond.Value)
	if err != nil {
		return false, fmt.Errorf("invalid comparison value: %w", err)
	}

	// Evaluate condition based on operator
	switch cond.Operator {
	case ">":
		return metricValue > comparisonValue, nil
	case "<":
		return metricValue < comparisonValue, nil
	case ">=":
		return metricValue >= comparisonValue, nil
	case "<=":
		return metricValue <= comparisonValue, nil
	case "==":
		// For ==, we need to handle both numeric and string comparisons
		if isNumeric(cond.Value) {
			return math.Abs(metricValue-comparisonValue) < 0.0001, nil // Float comparison with epsilon
		}
		// String comparison (for future use)
		return false, fmt.Errorf("string comparison not yet supported for numeric metrics")
	case "!=":
		if isNumeric(cond.Value) {
			return math.Abs(metricValue-comparisonValue) >= 0.0001, nil
		}
		return false, fmt.Errorf("string comparison not yet supported for numeric metrics")
	default:
		return false, fmt.Errorf("unsupported operator: %s", cond.Operator)
	}
}

// getNumericValue converts a value to float64
func getNumericValue(value interface{}) (float64, error) {
	if value == nil {
		return 0, fmt.Errorf("value cannot be nil")
	}

	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", value)
	}
}

// isNumeric checks if a value is numeric
func isNumeric(value interface{}) bool {
	if value == nil {
		return false
	}

	switch value.(type) {
	case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	default:
		return false
	}
}

