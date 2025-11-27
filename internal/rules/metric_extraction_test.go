package rules

import (
	"testing"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

func TestExtractRequiredMetrics(t *testing.T) {
	tests := []struct {
		name     string
		rules    []*models.Rule
		wantSize int
		wantKeys []string
	}{
		{
			name: "single rule with one metric",
			rules: []*models.Rule{
				{
					ID:      "rule1",
					Enabled: true,
					Conditions: []models.Condition{
						{Metric: "price_change_5m_pct", Operator: ">", Value: 5.0},
					},
				},
			},
			wantSize: 1,
			wantKeys: []string{"price_change_5m_pct"},
		},
		{
			name: "single rule with multiple metrics",
			rules: []*models.Rule{
				{
					ID:      "rule1",
					Enabled: true,
					Conditions: []models.Condition{
						{Metric: "price_change_5m_pct", Operator: ">", Value: 5.0},
						{Metric: "volume_daily", Operator: ">", Value: 1000000},
					},
				},
			},
			wantSize: 2,
			wantKeys: []string{"price_change_5m_pct", "volume_daily"},
		},
		{
			name: "multiple rules with overlapping metrics",
			rules: []*models.Rule{
				{
					ID:      "rule1",
					Enabled: true,
					Conditions: []models.Condition{
						{Metric: "price_change_5m_pct", Operator: ">", Value: 5.0},
					},
				},
				{
					ID:      "rule2",
					Enabled: true,
					Conditions: []models.Condition{
						{Metric: "price_change_5m_pct", Operator: "<", Value: -5.0},
						{Metric: "rsi_14", Operator: ">", Value: 70},
					},
				},
			},
			wantSize: 2, // price_change_5m_pct appears twice but only once in set
			wantKeys: []string{"price_change_5m_pct", "rsi_14"},
		},
		{
			name: "disabled rule ignored",
			rules: []*models.Rule{
				{
					ID:      "rule1",
					Enabled: false,
					Conditions: []models.Condition{
						{Metric: "price_change_5m_pct", Operator: ">", Value: 5.0},
					},
				},
			},
			wantSize: 0,
			wantKeys: []string{},
		},
		{
			name: "rule with volume threshold",
			rules: []*models.Rule{
				{
					ID:      "rule1",
					Enabled: true,
					Conditions: []models.Condition{
						{
							Metric:         "price_change_5m_pct",
							Operator:       ">",
							Value:          5.0,
							VolumeThreshold: int64Ptr(100000),
						},
					},
				},
			},
			wantSize: 5, // price_change_5m_pct + 4 volume metrics
			wantKeys: []string{"price_change_5m_pct", "volume_daily", "premarket_volume", "postmarket_volume", "market_volume"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractRequiredMetrics(tt.rules)

			if len(got) != tt.wantSize {
				t.Errorf("ExtractRequiredMetrics() size = %d, want %d", len(got), tt.wantSize)
			}

			for _, key := range tt.wantKeys {
				if !got[key] {
					t.Errorf("ExtractRequiredMetrics() missing key %q", key)
				}
			}
		})
	}
}

func TestExtractRequiredMetricsFromRule(t *testing.T) {
	tests := []struct {
		name     string
		rule     *models.Rule
		wantSize int
		wantKeys []string
	}{
		{
			name: "nil rule",
			rule: nil,
			wantSize: 0,
			wantKeys: []string{},
		},
		{
			name: "disabled rule",
			rule: &models.Rule{
				ID:      "rule1",
				Enabled: false,
				Conditions: []models.Condition{
					{Metric: "price_change_5m_pct", Operator: ">", Value: 5.0},
				},
			},
			wantSize: 0,
			wantKeys: []string{},
		},
		{
			name: "enabled rule with metrics",
			rule: &models.Rule{
				ID:      "rule1",
				Enabled: true,
				Conditions: []models.Condition{
					{Metric: "price_change_5m_pct", Operator: ">", Value: 5.0},
					{Metric: "volume_daily", Operator: ">", Value: 1000000},
				},
			},
			wantSize: 2,
			wantKeys: []string{"price_change_5m_pct", "volume_daily"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractRequiredMetricsFromRule(tt.rule)

			if len(got) != tt.wantSize {
				t.Errorf("ExtractRequiredMetricsFromRule() size = %d, want %d", len(got), tt.wantSize)
			}

			for _, key := range tt.wantKeys {
				if !got[key] {
					t.Errorf("ExtractRequiredMetricsFromRule() missing key %q", key)
				}
			}
		})
	}
}

