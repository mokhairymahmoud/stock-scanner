package models

import (
	"testing"
	"time"
)

func TestTick_Validate(t *testing.T) {
	tests := []struct {
		name    string
		tick    *Tick
		wantErr bool
	}{
		{
			name: "valid tick",
			tick: &Tick{
				Symbol:    "AAPL",
				Price:     150.50,
				Size:      100,
				Timestamp: time.Now(),
				Type:      "trade",
			},
			wantErr: false,
		},
		{
			name: "missing symbol",
			tick: &Tick{
				Price:     150.50,
				Size:      100,
				Timestamp: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "invalid price",
			tick: &Tick{
				Symbol:    "AAPL",
				Price:     0,
				Size:      100,
				Timestamp: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "zero timestamp",
			tick: &Tick{
				Symbol:    "AAPL",
				Price:     150.50,
				Size:      100,
				Timestamp: time.Time{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tick.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Tick.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBar1m_Validate(t *testing.T) {
	tests := []struct {
		name    string
		bar     *Bar1m
		wantErr bool
	}{
		{
			name: "valid bar",
			bar: &Bar1m{
				Symbol:    "AAPL",
				Timestamp: time.Now(),
				Open:      150.0,
				High:      151.0,
				Low:       149.0,
				Close:     150.5,
				Volume:    1000,
				VWAP:      150.25,
			},
			wantErr: false,
		},
		{
			name: "missing symbol",
			bar: &Bar1m{
				Timestamp: time.Now(),
				Open:      150.0,
				High:      151.0,
				Low:       149.0,
				Close:     150.5,
				Volume:    1000,
			},
			wantErr: true,
		},
		{
			name: "high < low",
			bar: &Bar1m{
				Symbol:    "AAPL",
				Timestamp: time.Now(),
				Open:      150.0,
				High:      149.0,
				Low:       151.0,
				Close:     150.5,
				Volume:    1000,
			},
			wantErr: true,
		},
		{
			name: "negative volume",
			bar: &Bar1m{
				Symbol:    "AAPL",
				Timestamp: time.Now(),
				Open:      150.0,
				High:      151.0,
				Low:       149.0,
				Close:     150.5,
				Volume:    -100,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.bar.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Bar1m.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLiveBar_Update(t *testing.T) {
	lb := &LiveBar{
		Symbol:    "AAPL",
		Timestamp: time.Now().Truncate(time.Minute),
	}

	tick1 := &Tick{
		Symbol:    "AAPL",
		Price:     150.0,
		Size:      100,
		Timestamp: time.Now(),
	}

	tick2 := &Tick{
		Symbol:    "AAPL",
		Price:     151.0,
		Size:      200,
		Timestamp: time.Now(),
	}

	tick3 := &Tick{
		Symbol:    "AAPL",
		Price:     149.0,
		Size:      50,
		Timestamp: time.Now(),
	}

	lb.Update(tick1)
	if lb.Open != 150.0 || lb.High != 150.0 || lb.Low != 150.0 || lb.Close != 150.0 {
		t.Errorf("After first tick: Open=%f, High=%f, Low=%f, Close=%f", lb.Open, lb.High, lb.Low, lb.Close)
	}
	if lb.Volume != 100 {
		t.Errorf("Volume after first tick = %d, want 100", lb.Volume)
	}

	lb.Update(tick2)
	if lb.High != 151.0 || lb.Close != 151.0 {
		t.Errorf("After second tick: High=%f, Close=%f", lb.High, lb.Close)
	}
	if lb.Volume != 300 {
		t.Errorf("Volume after second tick = %d, want 300", lb.Volume)
	}

	lb.Update(tick3)
	if lb.Low != 149.0 || lb.Close != 149.0 {
		t.Errorf("After third tick: Low=%f, Close=%f", lb.Low, lb.Close)
	}
	if lb.Volume != 350 {
		t.Errorf("Volume after third tick = %d, want 350", lb.Volume)
	}
}

func TestLiveBar_ToBar1m(t *testing.T) {
	lb := &LiveBar{
		Symbol:    "AAPL",
		Timestamp: time.Now().Truncate(time.Minute),
		Open:      150.0,
		High:      151.0,
		Low:       149.0,
		Close:     150.5,
		Volume:    1000,
		VWAPNum:   150250.0,
		VWAPDenom: 1000.0,
	}

	bar := lb.ToBar1m()
	if bar.Symbol != "AAPL" {
		t.Errorf("Bar symbol = %s, want AAPL", bar.Symbol)
	}
	if bar.VWAP != 150.25 {
		t.Errorf("Bar VWAP = %f, want 150.25", bar.VWAP)
	}
	if bar.Volume != 1000 {
		t.Errorf("Bar volume = %d, want 1000", bar.Volume)
	}
}

func TestRule_Validate(t *testing.T) {
	tests := []struct {
		name    string
		rule    *Rule
		wantErr bool
	}{
		{
			name: "valid rule",
			rule: &Rule{
				ID:   "rule-1",
				Name: "Test Rule",
				Conditions: []Condition{
					{
						Metric:   "rsi_14",
						Operator: ">",
						Value:    70.0,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			rule: &Rule{
				Name: "Test Rule",
				Conditions: []Condition{
					{
						Metric:   "rsi_14",
						Operator: ">",
						Value:    70.0,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing name",
			rule: &Rule{
				ID: "rule-1",
				Conditions: []Condition{
					{
						Metric:   "rsi_14",
						Operator: ">",
						Value:    70.0,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "no conditions",
			rule: &Rule{
				ID:        "rule-1",
				Name:      "Test Rule",
				Conditions: []Condition{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Rule.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCondition_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cond    Condition
		wantErr bool
	}{
		{
			name: "valid condition",
			cond: Condition{
				Metric:   "rsi_14",
				Operator: ">",
				Value:    70.0,
			},
			wantErr: false,
		},
		{
			name: "missing metric",
			cond: Condition{
				Operator: ">",
				Value:    70.0,
			},
			wantErr: true,
		},
		{
			name: "invalid operator",
			cond: Condition{
				Metric:   "rsi_14",
				Operator: "invalid",
				Value:    70.0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cond.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Condition.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAlert_Validate(t *testing.T) {
	tests := []struct {
		name    string
		alert   *Alert
		wantErr bool
	}{
		{
			name: "valid alert",
			alert: &Alert{
				ID:       "alert-1",
				RuleID:   "rule-1",
				RuleName: "Test Rule",
				Symbol:   "AAPL",
				Timestamp: time.Now(),
				Price:    150.50,
				Message:  "Alert triggered",
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			alert: &Alert{
				RuleID:   "rule-1",
				Symbol:   "AAPL",
				Timestamp: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "missing rule ID",
			alert: &Alert{
				ID:       "alert-1",
				Symbol:   "AAPL",
				Timestamp: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "missing symbol",
			alert: &Alert{
				ID:       "alert-1",
				RuleID:   "rule-1",
				Timestamp: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "zero timestamp",
			alert: &Alert{
				ID:       "alert-1",
				RuleID:   "rule-1",
				Symbol:   "AAPL",
				Timestamp: time.Time{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.alert.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Alert.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

