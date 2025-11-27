package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

// ExtractTimeframe extracts timeframe from metric name (e.g., "change_5m" -> "5m", "change_5m_pct" -> "5m")
// Returns empty string if no timeframe found
func ExtractTimeframe(metricName string) string {
	// First, try regex pattern for numeric timeframes (handles _pct suffix)
	// Pattern: _(\d+[mhd]) optionally followed by _pct or _percent
	re := regexp.MustCompile(`_(\d+[mhd])(?:_(?:pct|percent))?$`)
	matches := re.FindStringSubmatch(metricName)
	if len(matches) > 1 {
		return matches[1]
	}

	// Common timeframe patterns: _1m, _2m, _5m, _15m, _30m, _60m, _daily, _5d, _10d, etc.
	// Check for these patterns, handling _pct suffix
	timeframePatterns := []string{
		"_1m", "_2m", "_5m", "_10m", "_15m", "_30m", "_60m",
		"_1h", "_2h", "_4h",
		"_daily", "_today",
		"_5d", "_10d", "_20d", "_1y",
		"_3m", "_6m", // For biggest_range
	}

	// Remove _pct or _percent suffix if present
	baseName := metricName
	if strings.HasSuffix(metricName, "_pct") {
		baseName = strings.TrimSuffix(metricName, "_pct")
	} else if strings.HasSuffix(metricName, "_percent") {
		baseName = strings.TrimSuffix(metricName, "_percent")
	}

	for _, pattern := range timeframePatterns {
		if strings.HasSuffix(baseName, pattern) {
			return strings.TrimPrefix(pattern, "_")
		}
	}

	return ""
}

// ExtractValueType extracts value type from metric name (e.g., "change_pct" -> "%", "change" -> "$")
// Returns "$" for absolute, "%" for percentage, empty string if ambiguous
func ExtractValueType(metricName string) string {
	// Percentage indicators
	if strings.Contains(metricName, "_pct") || strings.Contains(metricName, "_percent") {
		return "%"
	}

	// Absolute value indicators (default)
	// Most metrics without _pct are absolute
	return "$"
}

// NormalizeMetricName removes timeframe and value type suffixes to get base metric name
// e.g., "change_5m_pct" -> "change"
func NormalizeMetricName(metricName string) string {
	// Remove timeframe
	timeframe := ExtractTimeframe(metricName)
	if timeframe != "" {
		metricName = strings.TrimSuffix(metricName, "_"+timeframe)
	}

	// Remove value type suffix
	metricName = strings.TrimSuffix(metricName, "_pct")
	metricName = strings.TrimSuffix(metricName, "_percent")

	return metricName
}

// CheckVolumeThreshold checks if volume meets the threshold requirement
func CheckVolumeThreshold(metrics map[string]float64, threshold *int64) bool {
	if threshold == nil || *threshold <= 0 {
		return true // No threshold requirement
	}

	// Check daily volume first (most comprehensive)
	if dailyVol, ok := metrics["volume_daily"]; ok && dailyVol > 0 {
		if int64(dailyVol) >= *threshold {
			return true
		}
	}

	// Check premarket volume
	if premarketVol, ok := metrics["premarket_volume"]; ok && premarketVol > 0 {
		if int64(premarketVol) >= *threshold {
			return true
		}
	}

	// Check postmarket volume
	if postmarketVol, ok := metrics["postmarket_volume"]; ok && postmarketVol > 0 {
		if int64(postmarketVol) >= *threshold {
			return true
		}
	}

	// Check recent timeframe volumes (1m, 5m, 15m, 60m) as fallback
	timeframeVolumes := []string{"volume_60m", "volume_15m", "volume_5m", "volume_1m"}
	for _, volMetric := range timeframeVolumes {
		if vol, ok := metrics[volMetric]; ok && vol > 0 {
			// For timeframe volumes, we need to estimate daily volume
			// This is approximate - for 1m volume, multiply by 390 (trading minutes)
			// For 5m, multiply by 78, etc.
			var multiplier float64
			switch volMetric {
			case "volume_1m":
				multiplier = 390.0 // Trading minutes in a day
			case "volume_5m":
				multiplier = 78.0
			case "volume_15m":
				multiplier = 26.0
			case "volume_60m":
				multiplier = 6.5
			default:
				continue
			}
			estimatedDaily := vol * multiplier
			if int64(estimatedDaily) >= *threshold {
				return true
			}
		}
	}

	// Check live bar volume if available (as last resort)
	if liveVol, ok := metrics["volume_live"]; ok && liveVol > 0 {
		// Estimate daily from live volume (very approximate)
		estimatedDaily := liveVol * 390.0
		if int64(estimatedDaily) >= *threshold {
			return true
		}
	}

	return false
}

// CheckSessionFilter checks if current session matches the filter requirement
// currentSession should be one of: "premarket", "market", "postmarket", "closed"
func CheckSessionFilter(currentSession string, calculatedDuring string) bool {
	if calculatedDuring == "" || calculatedDuring == "all" {
		return true // No session filter
	}

	switch calculatedDuring {
	case "premarket":
		return currentSession == "premarket"
	case "market":
		return currentSession == "market"
	case "postmarket":
		return currentSession == "postmarket"
	default:
		// Unknown session filter, allow it (default to true)
		return true
	}
}

// EnrichCondition enriches a condition with extracted timeframe and value type if not specified
func EnrichCondition(cond *models.Condition) {
	// Extract timeframe from metric name if not specified
	if cond.Timeframe == "" {
		cond.Timeframe = ExtractTimeframe(cond.Metric)
	}

	// Extract value type from metric name if not specified
	if cond.ValueType == "" {
		cond.ValueType = ExtractValueType(cond.Metric)
	}

	// Set default calculated_during if not specified
	if cond.CalculatedDuring == "" {
		cond.CalculatedDuring = "all"
	}

	// Set default volume threshold if not specified (0 = no threshold)
	if cond.VolumeThreshold == nil {
		zero := int64(0)
		cond.VolumeThreshold = &zero
	}
}

// ValidateFilterConfig validates filter configuration fields
func ValidateFilterConfig(cond *models.Condition) error {
	// Validate calculated_during
	if cond.CalculatedDuring != "" {
		validSessions := []string{"premarket", "market", "postmarket", "all"}
		valid := false
		for _, v := range validSessions {
			if cond.CalculatedDuring == v {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid calculated_during: %s (must be one of: %v)", cond.CalculatedDuring, validSessions)
		}
	}

	// Validate value_type
	if cond.ValueType != "" {
		validTypes := []string{"$", "%"}
		valid := false
		for _, v := range validTypes {
			if cond.ValueType == v {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid value_type: %s (must be $ or %%)", cond.ValueType)
		}
	}

	// Validate volume_threshold (must be >= 0)
	if cond.VolumeThreshold != nil && *cond.VolumeThreshold < 0 {
		return fmt.Errorf("volume_threshold must be >= 0, got %d", *cond.VolumeThreshold)
	}

	return nil
}

