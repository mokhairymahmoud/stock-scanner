package models

import (
	"encoding/json"
	"time"
)

// ToplistMetric represents the metric type for ranking
type ToplistMetric string

const (
	MetricChangePct      ToplistMetric = "change_pct"
	MetricVolume         ToplistMetric = "volume"
	MetricRSI            ToplistMetric = "rsi"
	MetricRelativeVolume ToplistMetric = "relative_volume"
	MetricVWAPDist       ToplistMetric = "vwap_dist"
)

// ToplistTimeWindow represents the time window for metric calculation
type ToplistTimeWindow string

const (
	Window1m  ToplistTimeWindow = "1m"
	Window5m  ToplistTimeWindow = "5m"
	Window15m ToplistTimeWindow = "15m"
	Window1h  ToplistTimeWindow = "1h"
	Window1d  ToplistTimeWindow = "1d"
)

// ToplistSortOrder represents the sort order for rankings
type ToplistSortOrder string

const (
	SortOrderAsc  ToplistSortOrder = "asc"
	SortOrderDesc ToplistSortOrder = "desc"
)

// SystemToplistType represents predefined system toplist types
type SystemToplistType string

const (
	SystemToplistGainers1m  SystemToplistType = "gainers_1m"
	SystemToplistGainers5m   SystemToplistType = "gainers_5m"
	SystemToplistGainers15m  SystemToplistType = "gainers_15m"
	SystemToplistGainers1h   SystemToplistType = "gainers_1h"
	SystemToplistGainers1d   SystemToplistType = "gainers_1d"
	SystemToplistLosers1m    SystemToplistType = "losers_1m"
	SystemToplistLosers5m    SystemToplistType = "losers_5m"
	SystemToplistLosers15m   SystemToplistType = "losers_15m"
	SystemToplistLosers1h    SystemToplistType = "losers_1h"
	SystemToplistLosers1d    SystemToplistType = "losers_1d"
	SystemToplistVolume1m    SystemToplistType = "volume_1m"
	SystemToplistVolume5m   SystemToplistType = "volume_5m"
	SystemToplistVolume15m   SystemToplistType = "volume_15m"
	SystemToplistVolume1h    SystemToplistType = "volume_1h"
	SystemToplistVolume1d    SystemToplistType = "volume_1d"
	SystemToplistRSIHigh     SystemToplistType = "rsi_high"
	SystemToplistRSILow      SystemToplistType = "rsi_low"
	SystemToplistRelVolume   SystemToplistType = "relative_volume"
	SystemToplistVWAPDistHigh SystemToplistType = "vwap_dist_high"
	SystemToplistVWAPDistLow  SystemToplistType = "vwap_dist_low"
)

// ToplistFilter represents filtering criteria for a toplist
type ToplistFilter struct {
	MinVolume  *int64   `json:"min_volume,omitempty"`
	MinChangePct *float64 `json:"min_change_pct,omitempty"`
	PriceMin   *float64 `json:"price_min,omitempty"`
	PriceMax   *float64 `json:"price_max,omitempty"`
	Exchange   *string  `json:"exchange,omitempty"`
	MarketCapMin *int64 `json:"market_cap_min,omitempty"`
	MarketCapMax *int64 `json:"market_cap_max,omitempty"`
}

// ToplistColorScheme represents color coding configuration
type ToplistColorScheme struct {
	Positive string `json:"positive,omitempty"` // Color for positive values (e.g., "#00ff00")
	Negative string `json:"negative,omitempty"` // Color for negative values (e.g., "#ff0000")
	Neutral  string `json:"neutral,omitempty"`  // Color for neutral values (e.g., "#ffffff")
}

// ToplistConfig represents a user-custom toplist configuration
type ToplistConfig struct {
	ID          string             `json:"id"`
	UserID      string             `json:"user_id"` // Empty for system toplists
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Metric      ToplistMetric      `json:"metric"`
	TimeWindow  ToplistTimeWindow   `json:"time_window"`
	SortOrder   ToplistSortOrder    `json:"sort_order"`
	Filters     *ToplistFilter      `json:"filters,omitempty"`
	Columns     []string            `json:"columns,omitempty"` // Display columns
	ColorScheme *ToplistColorScheme `json:"color_scheme,omitempty"`
	Enabled     bool                `json:"enabled"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

// Validate validates a ToplistConfig
func (tc *ToplistConfig) Validate() error {
	if tc.ID == "" {
		return ErrInvalidToplistID
	}
	if tc.Name == "" {
		return ErrInvalidToplistName
	}
	if tc.Metric == "" {
		return ErrInvalidToplistMetric
	}
	if tc.TimeWindow == "" {
		return ErrInvalidToplistTimeWindow
	}
	if tc.SortOrder == "" {
		return ErrInvalidToplistSortOrder
	}
	
	// Validate metric
	validMetrics := map[ToplistMetric]bool{
		MetricChangePct:      true,
		MetricVolume:         true,
		MetricRSI:            true,
		MetricRelativeVolume: true,
		MetricVWAPDist:       true,
	}
	if !validMetrics[tc.Metric] {
		return ErrInvalidToplistMetric
	}
	
	// Validate time window
	validWindows := map[ToplistTimeWindow]bool{
		Window1m:  true,
		Window5m:  true,
		Window15m: true,
		Window1h:  true,
		Window1d:  true,
	}
	if !validWindows[tc.TimeWindow] {
		return ErrInvalidToplistTimeWindow
	}
	
	// Validate sort order
	validSortOrders := map[ToplistSortOrder]bool{
		SortOrderAsc:  true,
		SortOrderDesc: true,
	}
	if !validSortOrders[tc.SortOrder] {
		return ErrInvalidToplistSortOrder
	}
	
	return nil
}

// IsSystemToplist returns true if this is a system toplist (no user_id)
func (tc *ToplistConfig) IsSystemToplist() bool {
	return tc.UserID == ""
}

// ToplistRanking represents a single symbol ranking entry
type ToplistRanking struct {
	Symbol   string                 `json:"symbol"`
	Rank     int                    `json:"rank"`
	Value    float64                `json:"value"` // The metric value used for ranking
	Metadata map[string]interface{} `json:"metadata,omitempty"` // Additional data (price, volume, etc.)
}

// ToplistUpdate represents a real-time toplist update message
type ToplistUpdate struct {
	ToplistID   string           `json:"toplist_id"`
	ToplistType string           `json:"toplist_type"` // "system" or "user"
	Rankings    []ToplistRanking `json:"rankings"`
	Timestamp   time.Time        `json:"timestamp"`
}

// Validate validates a ToplistUpdate
func (tu *ToplistUpdate) Validate() error {
	if tu.ToplistID == "" {
		return ErrInvalidToplistID
	}
	if tu.ToplistType != "system" && tu.ToplistType != "user" {
		return ErrInvalidToplistType
	}
	if tu.Timestamp.IsZero() {
		return ErrInvalidTimestamp
	}
	return nil
}

// GetRedisKey returns the Redis key for a system toplist
func GetSystemToplistRedisKey(metric ToplistMetric, window ToplistTimeWindow) string {
	return "toplist:" + string(metric) + ":" + string(window)
}

// GetUserToplistRedisKey returns the Redis key for a user toplist
func GetUserToplistRedisKey(userID string, toplistID string) string {
	return "toplist:user:" + userID + ":" + toplistID
}

// GetToplistConfigRedisKey returns the Redis key for caching a toplist configuration
func GetToplistConfigRedisKey(toplistID string) string {
	return "toplist:config:" + toplistID
}

// GetSystemToplistType returns the system toplist type for a given metric and window
func GetSystemToplistType(metric ToplistMetric, window ToplistTimeWindow, isGainer bool) SystemToplistType {
	if metric == MetricChangePct {
		if isGainer {
			switch window {
			case Window1m:
				return SystemToplistGainers1m
			case Window5m:
				return SystemToplistGainers5m
			case Window15m:
				return SystemToplistGainers15m
			case Window1h:
				return SystemToplistGainers1h
			case Window1d:
				return SystemToplistGainers1d
			}
		} else {
			switch window {
			case Window1m:
				return SystemToplistLosers1m
			case Window5m:
				return SystemToplistLosers5m
			case Window15m:
				return SystemToplistLosers15m
			case Window1h:
				return SystemToplistLosers1h
			case Window1d:
				return SystemToplistLosers1d
			}
		}
	} else if metric == MetricVolume {
		switch window {
		case Window1m:
			return SystemToplistVolume1m
		case Window5m:
			return SystemToplistVolume5m
		case Window15m:
			return SystemToplistVolume15m
		case Window1h:
			return SystemToplistVolume1h
		case Window1d:
			return SystemToplistVolume1d
		}
	} else if metric == MetricRSI {
		if isGainer {
			return SystemToplistRSIHigh
		}
		return SystemToplistRSILow
	} else if metric == MetricRelativeVolume {
		return SystemToplistRelVolume
	} else if metric == MetricVWAPDist {
		if isGainer {
			return SystemToplistVWAPDistHigh
		}
		return SystemToplistVWAPDistLow
	}
	return ""
}

// ToplistConfigFromJSON creates a ToplistConfig from JSON bytes
func ToplistConfigFromJSON(data []byte) (*ToplistConfig, error) {
	var config ToplistConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// ToJSON converts a ToplistConfig to JSON bytes
func (tc *ToplistConfig) ToJSON() ([]byte, error) {
	return json.Marshal(tc)
}

