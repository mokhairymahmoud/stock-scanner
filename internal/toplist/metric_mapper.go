package toplist

import (
	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

// MetricMapper maps toplist configurations to actual metric names in computed metrics
type MetricMapper struct{}

// NewMetricMapper creates a new metric mapper
func NewMetricMapper() *MetricMapper {
	return &MetricMapper{}
}

// GetMetricName returns the actual metric name for a given toplist config
// Returns empty string if the metric is not available in computed metrics
func (m *MetricMapper) GetMetricName(config *models.ToplistConfig) string {
	switch config.Metric {
	case models.MetricChangePct:
		switch config.TimeWindow {
		case models.Window1m:
			return "price_change_1m_pct"
		case models.Window5m:
			return "price_change_5m_pct"
		case models.Window15m:
			return "price_change_15m_pct"
		case models.Window1h:
			return "price_change_1h_pct"
		case models.Window1d:
			return "price_change_1d_pct"
		}
	case models.MetricVolume:
		switch config.TimeWindow {
		case models.Window1m:
			// Prefer finalized volume, fallback to live volume
			return "volume" // or "volume_live" as fallback
		case models.Window5m:
			return "volume_5m"
		case models.Window15m:
			return "volume_15m"
		case models.Window1h:
			return "volume_1h"
		case models.Window1d:
			return "volume_1d"
		}
	case models.MetricRSI:
		// RSI is typically 14-period, regardless of window
		// The window might affect which RSI we use, but for now we use rsi_14
		return "rsi_14"
	case models.MetricRelativeVolume:
		switch config.TimeWindow {
		case models.Window5m:
			return "relative_volume_5m"
		case models.Window15m:
			return "relative_volume_15m"
		case models.Window1h:
			return "relative_volume_1h"
		}
	case models.MetricVWAPDist:
		// VWAP distance is calculated from vwap_5m and close price
		// This is a special case that needs to be handled separately
		return "" // Special handling needed
	}
	return ""
}

// GetMetricValue extracts the metric value from a metrics map based on toplist config
// Returns the value and whether it was found
func (m *MetricMapper) GetMetricValue(config *models.ToplistConfig, metrics map[string]float64) (float64, bool) {
	metricName := m.GetMetricName(config)
	if metricName == "" {
		// Special handling for VWAP distance
		if config.Metric == models.MetricVWAPDist {
			return m.getVWAPDistance(config, metrics)
		}
		return 0, false
	}

	// Try primary metric name
	if value, ok := metrics[metricName]; ok {
		return value, true
	}

	// Fallback for volume: try volume_live if volume not found
	if config.Metric == models.MetricVolume && config.TimeWindow == models.Window1m {
		if value, ok := metrics["volume_live"]; ok {
			return value, true
		}
	}

	return 0, false
}

// getVWAPDistance calculates VWAP distance from metrics
func (m *MetricMapper) getVWAPDistance(config *models.ToplistConfig, metrics map[string]float64) (float64, bool) {
	var vwapKey string
	switch config.TimeWindow {
	case models.Window5m:
		vwapKey = "vwap_5m"
	case models.Window15m:
		vwapKey = "vwap_15m"
	case models.Window1h:
		vwapKey = "vwap_1h"
	default:
		return 0, false
	}

	vwap, vwapOk := metrics[vwapKey]
	price, priceOk := metrics["close"]
	if !vwapOk || !priceOk || vwap == 0 {
		return 0, false
	}

	// Calculate percentage distance from VWAP
	vwapDist := ((price - vwap) / vwap) * 100.0
	return vwapDist, true
}

// GetToplistRedisKey returns the Redis key for a toplist config
func (m *MetricMapper) GetToplistRedisKey(config *models.ToplistConfig) string {
	if config.IsSystemToplist() {
		return models.GetSystemToplistRedisKey(config.Metric, config.TimeWindow)
	}
	return models.GetUserToplistRedisKey(config.UserID, config.ID)
}

