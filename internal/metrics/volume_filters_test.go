package metrics

import (
	"testing"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

func TestPostmarketVolumeComputer(t *testing.T) {
	computer := &PostmarketVolumeComputer{}

	tests := []struct {
		name          string
		postmarketVol int64
		expected      float64
	}{
		{
			name:          "Valid volume",
			postmarketVol: 1000000,
			expected:      1000000.0,
		},
		{
			name:          "Zero volume",
			postmarketVol: 0,
			expected:      0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := &SymbolStateSnapshot{
				PostmarketVolume: tt.postmarketVol,
			}

			result, ok := computer.Compute(snapshot)
			if !ok {
				t.Errorf("Compute() ok = false, want true")
			}
			if result != tt.expected {
				t.Errorf("Compute() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPremarketVolumeComputer(t *testing.T) {
	computer := &PremarketVolumeComputer{}

	tests := []struct {
		name         string
		premarketVol int64
		expected     float64
	}{
		{
			name:         "Valid volume",
			premarketVol: 500000,
			expected:     500000.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := &SymbolStateSnapshot{
				PremarketVolume: tt.premarketVol,
			}

			result, ok := computer.Compute(snapshot)
			if !ok {
				t.Errorf("Compute() ok = false, want true")
			}
			if result != tt.expected {
				t.Errorf("Compute() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAbsoluteVolumeComputer(t *testing.T) {
	computer := NewAbsoluteVolumeComputer("volume_5m", 5)

	tests := []struct {
		name     string
		bars     []*models.Bar1m
		expected float64
		ok       bool
	}{
		{
			name: "Sufficient bars",
			bars: []*models.Bar1m{
				{Volume: 1000},
				{Volume: 2000},
				{Volume: 3000},
				{Volume: 4000},
				{Volume: 5000}, // Last 5 bars
			},
			expected: 15000.0, // Sum of last 5
			ok:       true,
		},
		{
			name:     "Insufficient bars",
			bars:     []*models.Bar1m{{Volume: 1000}},
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

func TestDollarVolumeComputer(t *testing.T) {
	computer := NewDollarVolumeComputer("dollar_volume_3m", 3)

	tests := []struct {
		name     string
		bars     []*models.Bar1m
		expected float64
		ok       bool
	}{
		{
			name: "With VWAP",
			bars: []*models.Bar1m{
				{VWAP: 100.0, Volume: 1000}, // 100,000
				{VWAP: 101.0, Volume: 2000}, // 202,000
				{VWAP: 102.0, Volume: 3000}, // 306,000
			},
			expected: 608000.0, // Sum
			ok:       true,
		},
		{
			name: "Without VWAP (uses Close)",
			bars: []*models.Bar1m{
				{VWAP: 0, Close: 100.0, Volume: 1000},
				{VWAP: 0, Close: 101.0, Volume: 2000},
				{VWAP: 0, Close: 102.0, Volume: 3000},
			},
			expected: 608000.0,
			ok:       true,
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

func TestDailyVolumeComputer(t *testing.T) {
	computer := &DailyVolumeComputer{}

	tests := []struct {
		name     string
		bars     []*models.Bar1m
		liveBar  *models.LiveBar
		expected float64
	}{
		{
			name: "With finalized bars only",
			bars: []*models.Bar1m{
				{Volume: 1000},
				{Volume: 2000},
				{Volume: 3000},
			},
			liveBar:  nil,
			expected: 6000.0,
		},
		{
			name: "With live bar",
			bars: []*models.Bar1m{
				{Volume: 1000},
				{Volume: 2000},
			},
			liveBar: &models.LiveBar{
				Volume: 500,
			},
			expected: 3500.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := &SymbolStateSnapshot{
				LastFinalBars: tt.bars,
				LiveBar:       tt.liveBar,
			}

			result, ok := computer.Compute(snapshot)
			if !ok {
				t.Errorf("Compute() ok = false, want true")
			}
			if result != tt.expected {
				t.Errorf("Compute() = %v, want %v", result, tt.expected)
			}
		})
	}
}

