package metrics

import (
	"testing"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

func TestChangeComputer(t *testing.T) {
	computer := NewChangeComputer("change_5m", 6)

	tests := []struct {
		name     string
		bars     []*models.Bar1m
		expected float64
		ok       bool
	}{
		{
			name: "Sufficient bars",
			bars: []*models.Bar1m{
				{Close: 100.0}, // 6 bars ago
				{Close: 101.0},
				{Close: 102.0},
				{Close: 103.0},
				{Close: 104.0},
				{Close: 105.0}, // Current
			},
			expected: 5.0, // 105 - 100
			ok:       true,
		},
		{
			name:     "Insufficient bars",
			bars:     []*models.Bar1m{{Close: 100.0}},
			expected: 0,
			ok:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := &SymbolStateSnapshot{
				LastFinalBars: tt.bars,
			}

			result, ok := computer.Compute(snapshot)
			if ok != tt.ok {
				t.Errorf("Compute() ok = %v, want %v", ok, tt.ok)
			}
			if ok && result != tt.expected {
				t.Errorf("Compute() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestChangeFromCloseComputer(t *testing.T) {
	computer := &ChangeFromCloseComputer{}

	tests := []struct {
		name          string
		yesterdayClose float64
		currentPrice  float64
		expected      float64
		ok            bool
	}{
		{
			name:           "Valid calculation",
			yesterdayClose: 100.0,
			currentPrice:  105.0,
			expected:      5.0,
			ok:            true,
		},
		{
			name:           "No yesterday close",
			yesterdayClose: 0,
			currentPrice:  105.0,
			expected:      0,
			ok:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := &SymbolStateSnapshot{
				YesterdayClose: tt.yesterdayClose,
				LiveBar: &models.LiveBar{
					Close: tt.currentPrice,
				},
			}

			result, ok := computer.Compute(snapshot)
			if ok != tt.ok {
				t.Errorf("Compute() ok = %v, want %v", ok, tt.ok)
			}
			if ok && result != tt.expected {
				t.Errorf("Compute() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestChangeFromClosePctComputer(t *testing.T) {
	computer := &ChangeFromClosePctComputer{}

	tests := []struct {
		name           string
		yesterdayClose float64
		currentPrice  float64
		expected       float64
		ok             bool
	}{
		{
			name:           "Valid calculation",
			yesterdayClose: 100.0,
			currentPrice:  105.0,
			expected:      5.0, // 5% increase
			ok:            true,
		},
		{
			name:           "Negative change",
			yesterdayClose: 100.0,
			currentPrice:  95.0,
			expected:      -5.0, // 5% decrease
			ok:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := &SymbolStateSnapshot{
				YesterdayClose: tt.yesterdayClose,
				LiveBar: &models.LiveBar{
					Close: tt.currentPrice,
				},
			}

			result, ok := computer.Compute(snapshot)
			if ok != tt.ok {
				t.Errorf("Compute() ok = %v, want %v", ok, tt.ok)
			}
			if ok && result != tt.expected {
				t.Errorf("Compute() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestChangeFromOpenComputer(t *testing.T) {
	computer := &ChangeFromOpenComputer{}

	tests := []struct {
		name         string
		todayOpen    float64
		currentPrice float64
		expected     float64
		ok           bool
	}{
		{
			name:         "Valid calculation",
			todayOpen:    100.0,
			currentPrice: 105.0,
			expected:    5.0,
			ok:          true,
		},
		{
			name:         "No today open",
			todayOpen:    0,
			currentPrice: 105.0,
			expected:    0,
			ok:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := &SymbolStateSnapshot{
				TodayOpen: tt.todayOpen,
				LiveBar: &models.LiveBar{
					Close: tt.currentPrice,
				},
			}

			result, ok := computer.Compute(snapshot)
			if ok != tt.ok {
				t.Errorf("Compute() ok = %v, want %v", ok, tt.ok)
			}
			if ok && result != tt.expected {
				t.Errorf("Compute() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGapFromCloseComputer(t *testing.T) {
	computer := &GapFromCloseComputer{}

	tests := []struct {
		name           string
		yesterdayClose float64
		todayOpen      float64
		expected       float64
		ok             bool
	}{
		{
			name:           "Gap up",
			yesterdayClose: 100.0,
			todayOpen:      105.0,
			expected:      5.0,
			ok:            true,
		},
		{
			name:           "Gap down",
			yesterdayClose: 100.0,
			todayOpen:      95.0,
			expected:      -5.0,
			ok:            true,
		},
		{
			name:           "Missing data",
			yesterdayClose: 0,
			todayOpen:      105.0,
			expected:      0,
			ok:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := &SymbolStateSnapshot{
				YesterdayClose: tt.yesterdayClose,
				TodayOpen:      tt.todayOpen,
			}

			result, ok := computer.Compute(snapshot)
			if ok != tt.ok {
				t.Errorf("Compute() ok = %v, want %v", ok, tt.ok)
			}
			if ok && result != tt.expected {
				t.Errorf("Compute() = %v, want %v", result, tt.expected)
			}
		})
	}
}

