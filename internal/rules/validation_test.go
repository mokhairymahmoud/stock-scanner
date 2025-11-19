package rules

import (
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

func TestValidateRule(t *testing.T) {
	tests := []struct {
		name    string
		rule    *models.Rule
		wantErr bool
	}{
		{
			name: "valid rule",
			rule: &models.Rule{
				ID:          "rule-1",
				Name:        "Test Rule",
				Description: "Test description",
				Conditions: []models.Condition{
					{Metric: "rsi_14", Operator: "<", Value: 30.0},
				},
				Cooldown:  300,
				Enabled:   true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "negative cooldown",
			rule: &models.Rule{
				ID:          "rule-2",
				Name:        "Test Rule",
				Conditions:  []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
				Cooldown:    -1,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			wantErr: true,
		},
		{
			name: "invalid condition",
			rule: &models.Rule{
				ID:          "rule-3",
				Name:        "Test Rule",
				Conditions:  []models.Condition{{Metric: "", Operator: "<", Value: 30.0}},
				Cooldown:   300,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRule(tt.rule)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRule() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCondition(t *testing.T) {
	tests := []struct {
		name    string
		cond    *models.Condition
		wantErr bool
	}{
		{
			name:    "valid numeric condition",
			cond:    &models.Condition{Metric: "rsi_14", Operator: "<", Value: 30.0},
			wantErr: false,
		},
		{
			name:    "valid string condition with ==",
			cond:    &models.Condition{Metric: "symbol", Operator: "==", Value: "AAPL"},
			wantErr: false,
		},
		{
			name:    "invalid string condition with >",
			cond:    &models.Condition{Metric: "symbol", Operator: ">", Value: "AAPL"},
			wantErr: true,
		},
		{
			name:    "nil value",
			cond:    &models.Condition{Metric: "rsi_14", Operator: "<", Value: nil},
			wantErr: true,
		},
		{
			name:    "invalid operator",
			cond:    &models.Condition{Metric: "rsi_14", Operator: "~", Value: 30.0},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCondition(tt.cond)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCondition() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateMetricName(t *testing.T) {
	tests := []struct {
		name    string
		metric  string
		wantErr bool
	}{
		{"valid metric", "rsi_14", false},
		{"valid metric with numbers", "ema_20", false},
		{"valid metric uppercase", "RSI_14", false},
		{"empty metric", "", true},
		{"invalid character", "rsi-14", true},
		{"invalid character space", "rsi 14", true},
		{"invalid character dot", "rsi.14", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMetricName(tt.metric)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMetricName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateOperator(t *testing.T) {
	tests := []struct {
		name    string
		op      string
		wantErr bool
	}{
		{">", ">", false},
		{"<", "<", false},
		{">=", ">=", false},
		{"<=", "<=", false},
		{"==", "==", false},
		{"!=", "!=", false},
		{"invalid", "~", true},
		{"invalid", ">=", false}, // >= is valid
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOperator(tt.op)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOperator() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

