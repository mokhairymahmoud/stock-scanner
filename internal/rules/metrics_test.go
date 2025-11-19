package rules

import (
	"fmt"
	"testing"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

func TestDefaultMetricResolver_ResolveMetric(t *testing.T) {
	resolver := NewMetricResolver()

	metrics := map[string]float64{
		"rsi_14":     65.5,
		"ema_20":     150.2,
		"price":      149.8,
		"volume_avg": 1000000.0,
	}

	tests := []struct {
		name      string
		metric    string
		metrics   map[string]float64
		wantValue float64
		wantErr   bool
	}{
		{
			name:      "direct lookup",
			metric:    "rsi_14",
			metrics:   metrics,
			wantValue: 65.5,
			wantErr:   false,
		},
		{
			name:      "another direct lookup",
			metric:    "ema_20",
			metrics:   metrics,
			wantValue: 150.2,
			wantErr:   false,
		},
		{
			name:      "metric not found",
			metric:    "nonexistent",
			metrics:   metrics,
			wantValue: 0,
			wantErr:   true,
		},
		{
			name:      "empty metric name",
			metric:    "",
			metrics:   metrics,
			wantValue: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := resolver.ResolveMetric(tt.metric, tt.metrics)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveMetric() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && value != tt.wantValue {
				t.Errorf("ResolveMetric() value = %v, want %v", value, tt.wantValue)
			}
		})
	}
}

func TestDefaultMetricResolver_RegisterComputedMetric(t *testing.T) {
	resolver := NewMetricResolver()

	// Register a computed metric
	err := resolver.RegisterComputedMetric("test_metric", func(metrics map[string]float64) (float64, error) {
		if val, ok := metrics["rsi_14"]; ok {
			return val * 2, nil
		}
		return 0, fmt.Errorf("rsi_14 not found")
	})

	if err != nil {
		t.Fatalf("RegisterComputedMetric() error = %v", err)
	}

	// Test the computed metric
	metrics := map[string]float64{"rsi_14": 30.0}
	value, err := resolver.ResolveMetric("test_metric", metrics)
	if err != nil {
		t.Fatalf("ResolveMetric() error = %v", err)
	}

	if value != 60.0 {
		t.Errorf("Expected computed metric value 60.0, got %f", value)
	}
}

func TestEvaluateCondition(t *testing.T) {
	resolver := NewMetricResolver()

	metrics := map[string]float64{
		"rsi_14": 25.0,
		"ema_20": 150.0,
	}

	tests := []struct {
		name    string
		cond    *models.Condition
		metrics map[string]float64
		want    bool
		wantErr bool
	}{
		{
			name:    "less than - true",
			cond:    &models.Condition{Metric: "rsi_14", Operator: "<", Value: 30.0},
			metrics: metrics,
			want:    true,
			wantErr: false,
		},
		{
			name:    "less than - false",
			cond:    &models.Condition{Metric: "rsi_14", Operator: "<", Value: 20.0},
			metrics: metrics,
			want:    false,
			wantErr: false,
		},
		{
			name:    "greater than - true",
			cond:    &models.Condition{Metric: "rsi_14", Operator: ">", Value: 20.0},
			metrics: metrics,
			want:    true,
			wantErr: false,
		},
		{
			name:    "greater than or equal - true",
			cond:    &models.Condition{Metric: "rsi_14", Operator: ">=", Value: 25.0},
			metrics: metrics,
			want:    true,
			wantErr: false,
		},
		{
			name:    "less than or equal - true",
			cond:    &models.Condition{Metric: "rsi_14", Operator: "<=", Value: 25.0},
			metrics: metrics,
			want:    true,
			wantErr: false,
		},
		{
			name:    "equal - true",
			cond:    &models.Condition{Metric: "rsi_14", Operator: "==", Value: 25.0},
			metrics: metrics,
			want:    true,
			wantErr: false,
		},
		{
			name:    "not equal - true",
			cond:    &models.Condition{Metric: "rsi_14", Operator: "!=", Value: 30.0},
			metrics: metrics,
			want:    true,
			wantErr: false,
		},
		{
			name:    "metric not found",
			cond:    &models.Condition{Metric: "nonexistent", Operator: "<", Value: 30.0},
			metrics: metrics,
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvaluateCondition(tt.cond, resolver, tt.metrics)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != tt.want {
				t.Errorf("EvaluateCondition() result = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestGetNumericValue(t *testing.T) {
	tests := []struct {
		name    string
		value   interface{}
		want    float64
		wantErr bool
	}{
		{"float64", 30.5, 30.5, false},
		{"float32", float32(30.5), 30.5, false},
		{"int", 30, 30.0, false},
		{"int64", int64(30), 30.0, false},
		{"string", "30", 0, true},
		{"nil", nil, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getNumericValue(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("getNumericValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("getNumericValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNumeric(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  bool
	}{
		{"float64", 30.5, true},
		{"int", 30, true},
		{"int64", int64(30), true},
		{"string", "30", false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNumeric(tt.value); got != tt.want {
				t.Errorf("isNumeric() = %v, want %v", got, tt.want)
			}
		})
	}
}

