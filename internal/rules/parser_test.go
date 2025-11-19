package rules

import (
	"strings"
	"testing"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

func TestParseRule(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
		validate func(*testing.T, *models.Rule)
	}{
		{
			name: "valid rule",
			json: `{
				"id": "rule-1",
				"name": "RSI Oversold",
				"description": "Alert when RSI is below 30",
				"conditions": [
					{"metric": "rsi_14", "operator": "<", "value": 30.0}
				],
				"cooldown": 300,
				"enabled": true
			}`,
			wantErr: false,
			validate: func(t *testing.T, r *models.Rule) {
				if r.ID != "rule-1" {
					t.Errorf("Expected ID 'rule-1', got '%s'", r.ID)
				}
				if r.Name != "RSI Oversold" {
					t.Errorf("Expected Name 'RSI Oversold', got '%s'", r.Name)
				}
				if len(r.Conditions) != 1 {
					t.Errorf("Expected 1 condition, got %d", len(r.Conditions))
				}
				if r.Cooldown != 300 {
					t.Errorf("Expected cooldown 300, got %d", r.Cooldown)
				}
				if !r.Enabled {
					t.Error("Expected rule to be enabled")
				}
			},
		},
		{
			name: "rule with multiple conditions",
			json: `{
				"id": "rule-2",
				"name": "RSI and Volume",
				"conditions": [
					{"metric": "rsi_14", "operator": "<", "value": 30.0},
					{"metric": "volume_avg_5m", "operator": ">", "value": 1000000.0}
				],
				"cooldown": 600
			}`,
			wantErr: false,
			validate: func(t *testing.T, r *models.Rule) {
				if len(r.Conditions) != 2 {
					t.Errorf("Expected 2 conditions, got %d", len(r.Conditions))
				}
			},
		},
		{
			name: "invalid JSON",
			json: `{
				"id": "rule-1",
				"name": "Test",
				invalid json
			}`,
			wantErr: true,
		},
		{
			name: "missing required field",
			json: `{
				"name": "Test",
				"conditions": [{"metric": "rsi_14", "operator": "<", "value": 30.0}]
			}`,
			wantErr: true,
		},
		{
			name: "invalid condition",
			json: `{
				"id": "rule-1",
				"name": "Test",
				"conditions": [{"metric": "", "operator": "<", "value": 30.0}]
			}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule, err := ParseRule([]byte(tt.json))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && rule != nil && tt.validate != nil {
				tt.validate(t, rule)
			}
		})
	}
}

func TestParseRuleFromString(t *testing.T) {
	jsonStr := `{
		"id": "rule-1",
		"name": "Test Rule",
		"conditions": [{"metric": "rsi_14", "operator": "<", "value": 30.0}],
		"cooldown": 300
	}`

	rule, err := ParseRuleFromString(jsonStr)
	if err != nil {
		t.Fatalf("ParseRuleFromString() error = %v", err)
	}

	if rule.ID != "rule-1" {
		t.Errorf("Expected ID 'rule-1', got '%s'", rule.ID)
	}
}

func TestParseRuleFromReader(t *testing.T) {
	jsonStr := `{
		"id": "rule-1",
		"name": "Test Rule",
		"conditions": [{"metric": "rsi_14", "operator": "<", "value": 30.0}],
		"cooldown": 300
	}`

	reader := strings.NewReader(jsonStr)
	rule, err := ParseRuleFromReader(reader)
	if err != nil {
		t.Fatalf("ParseRuleFromReader() error = %v", err)
	}

	if rule.ID != "rule-1" {
		t.Errorf("Expected ID 'rule-1', got '%s'", rule.ID)
	}
}

func TestParseRules(t *testing.T) {
	jsonStr := `[
		{
			"id": "rule-1",
			"name": "Rule 1",
			"conditions": [{"metric": "rsi_14", "operator": "<", "value": 30.0}],
			"cooldown": 300
		},
		{
			"id": "rule-2",
			"name": "Rule 2",
			"conditions": [{"metric": "ema_20", "operator": ">", "value": 150.0}],
			"cooldown": 600
		}
	]`

	rules, err := ParseRules([]byte(jsonStr))
	if err != nil {
		t.Fatalf("ParseRules() error = %v", err)
	}

	if len(rules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(rules))
	}

	if rules[0].ID != "rule-1" {
		t.Errorf("Expected first rule ID 'rule-1', got '%s'", rules[0].ID)
	}

	if rules[1].ID != "rule-2" {
		t.Errorf("Expected second rule ID 'rule-2', got '%s'", rules[1].ID)
	}
}

func TestParseRule_Timestamps(t *testing.T) {
	jsonStr := `{
		"id": "rule-1",
		"name": "Test Rule",
		"conditions": [{"metric": "rsi_14", "operator": "<", "value": 30.0}],
		"cooldown": 300
	}`

	rule, err := ParseRule([]byte(jsonStr))
	if err != nil {
		t.Fatalf("ParseRule() error = %v", err)
	}

	// Timestamps should be set automatically
	if rule.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}

	if rule.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
}

func TestValidateRuleSyntax(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "valid syntax",
			json:    `{"id": "rule-1", "name": "Test", "conditions": [{"metric": "rsi_14", "operator": "<", "value": 30.0}]}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			json:    `{invalid json}`,
			wantErr: true,
		},
		{
			name:    "missing id",
			json:    `{"name": "Test", "conditions": []}`,
			wantErr: true,
		},
		{
			name:    "missing name",
			json:    `{"id": "rule-1", "conditions": []}`,
			wantErr: true,
		},
		{
			name:    "missing conditions",
			json:    `{"id": "rule-1", "name": "Test"}`,
			wantErr: true,
		},
		{
			name:    "empty conditions",
			json:    `{"id": "rule-1", "name": "Test", "conditions": []}`,
			wantErr: true,
		},
		{
			name:    "conditions not array",
			json:    `{"id": "rule-1", "name": "Test", "conditions": "not an array"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRuleSyntax([]byte(tt.json))
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRuleSyntax() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateMetricReference(t *testing.T) {
	tests := []struct {
		name    string
		metric  string
		wantErr bool
	}{
		{"valid indicator", "rsi_14", false},
		{"valid indicator", "ema_20", false},
		{"valid computed metric", "price_change_5m_pct", false},
		{"empty", "", true},
		{"invalid character", "rsi-14", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMetricReference(tt.metric)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMetricReference() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

