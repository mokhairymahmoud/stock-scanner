package rules

import (
	"testing"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

func TestExtractTimeframe(t *testing.T) {
	tests := []struct {
		name      string
		metric    string
		wantFrame string
	}{
		{"1 minute", "change_1m", "1m"},
		{"5 minute", "change_5m", "5m"},
		{"15 minute", "volume_15m", "15m"},
		{"60 minute", "range_60m", "60m"},
		{"daily", "volume_daily", "daily"},
		{"today", "range_today", "today"},
		{"5 days", "avg_volume_5d", "5d"},
		{"no timeframe", "price", ""},
		{"with pct", "change_5m_pct", "5m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractTimeframe(tt.metric)
			if got != tt.wantFrame {
				t.Errorf("ExtractTimeframe(%q) = %q, want %q", tt.metric, got, tt.wantFrame)
			}
		})
	}
}

func TestExtractValueType(t *testing.T) {
	tests := []struct {
		name     string
		metric   string
		wantType string
	}{
		{"percentage", "change_pct", "%"},
		{"percentage with timeframe", "change_5m_pct", "%"},
		{"absolute", "change", "$"},
		{"absolute with timeframe", "change_5m", "$"},
		{"range pct", "range_pct_5m", "%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractValueType(tt.metric)
			if got != tt.wantType {
				t.Errorf("ExtractValueType(%q) = %q, want %q", tt.metric, got, tt.wantType)
			}
		})
	}
}

func TestCheckVolumeThreshold(t *testing.T) {
	tests := []struct {
		name      string
		metrics   map[string]float64
		threshold *int64
		want      bool
	}{
		{
			name:      "no threshold",
			metrics:   map[string]float64{"volume_daily": 1000},
			threshold: nil,
			want:      true,
		},
		{
			name:      "zero threshold",
			metrics:   map[string]float64{"volume_daily": 1000},
			threshold: int64Ptr(0),
			want:      true,
		},
		{
			name:      "daily volume meets threshold",
			metrics:   map[string]float64{"volume_daily": 100000},
			threshold: int64Ptr(50000),
			want:      true,
		},
		{
			name:      "daily volume below threshold",
			metrics:   map[string]float64{"volume_daily": 10000},
			threshold: int64Ptr(50000),
			want:      false,
		},
		{
			name:      "premarket volume meets threshold",
			metrics:   map[string]float64{"premarket_volume": 50000},
			threshold: int64Ptr(25000),
			want:      true,
		},
		{
			name:      "estimated from 1m volume",
			metrics:   map[string]float64{"volume_1m": 1000}, // 1000 * 390 = 390000
			threshold: int64Ptr(200000),
			want:      true,
		},
		{
			name:      "no volume metrics",
			metrics:   map[string]float64{"price": 100.0},
			threshold: int64Ptr(50000),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckVolumeThreshold(tt.metrics, tt.threshold)
			if got != tt.want {
				t.Errorf("CheckVolumeThreshold() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckSessionFilter(t *testing.T) {
	tests := []struct {
		name            string
		currentSession  string
		calculatedDuring string
		want            bool
	}{
		{"no filter", "market", "", true},
		{"all sessions", "market", "all", true},
		{"market session match", "market", "market", true},
		{"market session mismatch", "premarket", "market", false},
		{"premarket match", "premarket", "premarket", true},
		{"postmarket match", "postmarket", "postmarket", true},
		{"closed session", "closed", "market", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckSessionFilter(tt.currentSession, tt.calculatedDuring)
			if got != tt.want {
				t.Errorf("CheckSessionFilter(%q, %q) = %v, want %v", tt.currentSession, tt.calculatedDuring, got, tt.want)
			}
		})
	}
}

func TestEnrichCondition(t *testing.T) {
	tests := []struct {
		name     string
		cond     *models.Condition
		wantTimeframe string
		wantValueType string
		wantCalculatedDuring string
	}{
		{
			name: "enrich from metric name",
			cond: &models.Condition{
				Metric: "change_5m_pct",
			},
			wantTimeframe: "5m",
			wantValueType: "%",
			wantCalculatedDuring: "all",
		},
		{
			name: "preserve existing values",
			cond: &models.Condition{
				Metric: "change_5m_pct",
				Timeframe: "15m",
				ValueType: "$",
				CalculatedDuring: "market",
			},
			wantTimeframe: "15m", // Preserved
			wantValueType: "$",    // Preserved
			wantCalculatedDuring: "market", // Preserved
		},
		{
			name: "set defaults",
			cond: &models.Condition{
				Metric: "price",
			},
			wantTimeframe: "",
			wantValueType: "$",
			wantCalculatedDuring: "all",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			EnrichCondition(tt.cond)
			if tt.cond.Timeframe != tt.wantTimeframe {
				t.Errorf("Timeframe = %q, want %q", tt.cond.Timeframe, tt.wantTimeframe)
			}
			if tt.cond.ValueType != tt.wantValueType {
				t.Errorf("ValueType = %q, want %q", tt.cond.ValueType, tt.wantValueType)
			}
			if tt.cond.CalculatedDuring != tt.wantCalculatedDuring {
				t.Errorf("CalculatedDuring = %q, want %q", tt.cond.CalculatedDuring, tt.wantCalculatedDuring)
			}
		})
	}
}

func TestValidateFilterConfig(t *testing.T) {
	tests := []struct {
		name    string
		cond    *models.Condition
		wantErr bool
	}{
		{
			name: "valid config",
			cond: &models.Condition{
				CalculatedDuring: "market",
				ValueType:        "$",
				VolumeThreshold:  int64Ptr(10000),
			},
			wantErr: false,
		},
		{
			name: "invalid calculated_during",
			cond: &models.Condition{
				CalculatedDuring: "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid value_type",
			cond: &models.Condition{
				ValueType: "invalid",
			},
			wantErr: true,
		},
		{
			name: "negative volume threshold",
			cond: &models.Condition{
				VolumeThreshold: int64Ptr(-1),
			},
			wantErr: true,
		},
		{
			name: "zero volume threshold (valid)",
			cond: &models.Condition{
				VolumeThreshold: int64Ptr(0),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilterConfig(tt.cond)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFilterConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function
func int64Ptr(v int64) *int64 {
	return &v
}

