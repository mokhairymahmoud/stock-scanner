package rules

import (
	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

// ExtractRequiredMetrics extracts all metric names required by a set of rules
// Returns a set of metric names (as map[string]bool for fast lookup)
func ExtractRequiredMetrics(rules []*models.Rule) map[string]bool {
	requiredMetrics := make(map[string]bool)

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		for _, cond := range rule.Conditions {
			// Add the metric name from the condition
			if cond.Metric != "" {
				requiredMetrics[cond.Metric] = true
			}

			// For volume threshold checks, we need volume metrics
			if cond.VolumeThreshold != nil && *cond.VolumeThreshold > 0 {
				// Add common volume metrics that might be checked
				requiredMetrics["volume_daily"] = true
				requiredMetrics["premarket_volume"] = true
				requiredMetrics["postmarket_volume"] = true
				requiredMetrics["market_volume"] = true
			}
		}
	}

	return requiredMetrics
}

// ExtractRequiredMetricsFromRule extracts all metric names required by a single rule
func ExtractRequiredMetricsFromRule(rule *models.Rule) map[string]bool {
	if rule == nil || !rule.Enabled {
		return make(map[string]bool)
	}

	requiredMetrics := make(map[string]bool)

	for _, cond := range rule.Conditions {
		if cond.Metric != "" {
			requiredMetrics[cond.Metric] = true
		}

		// For volume threshold checks, we need volume metrics
		if cond.VolumeThreshold != nil && *cond.VolumeThreshold > 0 {
			requiredMetrics["volume_daily"] = true
			requiredMetrics["premarket_volume"] = true
			requiredMetrics["postmarket_volume"] = true
			requiredMetrics["market_volume"] = true
		}
	}

	return requiredMetrics
}

