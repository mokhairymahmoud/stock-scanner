package metrics

import (
	"testing"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

func TestTradeCountComputer(t *testing.T) {
	tests := []struct {
		name            string
		barOffset       int
		tradeCountHistory []int64
		bars            []*models.Bar1m
		wantValue       float64
		wantOk          bool
	}{
		{
			name:            "insufficient trade count history",
			barOffset:       5,
			tradeCountHistory: []int64{10, 20},
			bars:            createBarsForActivity(3, 100.0),
			wantValue:       0,
			wantOk:          false,
		},
		{
			name:            "sufficient trade count history",
			barOffset:       5,
			tradeCountHistory: []int64{10, 20, 30, 40, 50},
			bars:            createBarsForActivity(5, 100.0),
			wantValue:       150.0, // Sum of last 5: 10+20+30+40+50
			wantOk:          true,
		},
		{
			name:            "fallback to bar count",
			barOffset:       3,
			tradeCountHistory: []int64{}, // Empty history
			bars:            createBarsForActivity(5, 100.0),
			wantValue:       3.0, // Number of bars (barOffset)
			wantOk:          true,
		},
		{
			name:            "insufficient bars for fallback",
			barOffset:       5,
			tradeCountHistory: []int64{},
			bars:            createBarsForActivity(2, 100.0),
			wantValue:       0,
			wantOk:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			computer := NewTradeCountComputer("test_trade_count", tt.barOffset)
			snapshot := &SymbolStateSnapshot{
				LastFinalBars:     tt.bars,
				TradeCountHistory: tt.tradeCountHistory,
			}

			value, ok := computer.Compute(snapshot)
			if ok != tt.wantOk {
				t.Errorf("Compute() ok = %v, want %v", ok, tt.wantOk)
			}
			if ok && value != tt.wantValue {
				t.Errorf("Compute() value = %v, want %v", value, tt.wantValue)
			}
		})
	}
}

func TestConsecutiveCandlesComputer(t *testing.T) {
	tests := []struct {
		name            string
		timeframe       string
		candleDirections map[string][]bool
		wantValue       float64
		wantOk          bool
	}{
		{
			name:            "no candle directions",
			timeframe:       "1m",
			candleDirections: nil,
			wantValue:       0,
			wantOk:          false,
		},
		{
			name:            "timeframe not found",
			timeframe:       "1m",
			candleDirections: map[string][]bool{
				"5m": {true, false},
			},
			wantValue: 0,
			wantOk:    false,
		},
		{
			name:      "consecutive green candles",
			timeframe: "1m",
			candleDirections: map[string][]bool{
				"1m": {false, false, true, true, true}, // Last 3 are green
			},
			wantValue: 3.0, // Positive for green
			wantOk:    true,
		},
		{
			name:      "consecutive red candles",
			timeframe: "1m",
			candleDirections: map[string][]bool{
				"1m": {true, true, false, false, false}, // Last 3 are red
			},
			wantValue: -3.0, // Negative for red
			wantOk:    true,
		},
		{
			name:      "single candle",
			timeframe: "1m",
			candleDirections: map[string][]bool{
				"1m": {true},
			},
			wantValue: 1.0,
			wantOk:    true,
		},
		{
			name:      "mixed candles ending with green",
			timeframe: "1m",
			candleDirections: map[string][]bool{
				"1m": {false, true, false, true, true}, // Last 2 are green
			},
			wantValue: 2.0,
			wantOk:    true,
		},
		{
			name:      "mixed candles ending with red",
			timeframe: "1m",
			candleDirections: map[string][]bool{
				"1m": {true, false, true, false, false}, // Last 2 are red
			},
			wantValue: -2.0,
			wantOk:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			computer := NewConsecutiveCandlesComputer("test_consecutive", tt.timeframe)
			snapshot := &SymbolStateSnapshot{
				CandleDirections: tt.candleDirections,
			}

			value, ok := computer.Compute(snapshot)
			if ok != tt.wantOk {
				t.Errorf("Compute() ok = %v, want %v", ok, tt.wantOk)
			}
			if ok && value != tt.wantValue {
				t.Errorf("Compute() value = %v, want %v", value, tt.wantValue)
			}
		})
	}
}

// Helper functions

func createBarsForActivity(count int, basePrice float64) []*models.Bar1m {
	bars := make([]*models.Bar1m, count)
	for i := 0; i < count; i++ {
		bars[i] = &models.Bar1m{
			Symbol:    "TEST",
			Open:      basePrice,
			High:      basePrice + 5,
			Low:       basePrice - 5,
			Close:     basePrice,
			Volume:    1000,
		}
	}
	return bars
}

func TestAverageVolumeComputer(t *testing.T) {
	tests := []struct {
		name     string
		days     int
		bars     []*models.Bar1m
		liveBar  *models.LiveBar
		wantOk   bool
	}{
		{
			name:   "no bars",
			days:   5,
			bars:   []*models.Bar1m{},
			wantOk: false,
		},
		{
			name: "with finalized bars",
			days: 5,
			bars: []*models.Bar1m{
				{Volume: 1000},
				{Volume: 2000},
				{Volume: 3000},
			},
			wantOk: true,
		},
		{
			name: "with live bar",
			days: 10,
			bars: []*models.Bar1m{
				{Volume: 1000},
				{Volume: 2000},
			},
			liveBar: &models.LiveBar{
				Volume: 500,
			},
			wantOk: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			computer := NewAverageVolumeComputer("test_avg_volume", tt.days)
			snapshot := &SymbolStateSnapshot{
				LastFinalBars: tt.bars,
				LiveBar:       tt.liveBar,
			}

			value, ok := computer.Compute(snapshot)
			if ok != tt.wantOk {
				t.Errorf("Compute() ok = %v, want %v", ok, tt.wantOk)
			}
			if ok && value <= 0 {
				t.Errorf("Compute() value = %v, want > 0", value)
			}
		})
	}
}

func TestRelativeVolumeComputer(t *testing.T) {
	tests := []struct {
		name        string
		barOffset   int
		bars        []*models.Bar1m
		liveBar     *models.LiveBar
		wantValue   float64
		wantOk      bool
	}{
		{
			name:      "insufficient bars",
			barOffset: 10,
			bars:      createBarsForActivity(5, 100.0),
			wantValue: 0,
			wantOk:    false,
		},
		{
			name:      "volume equal to average",
			barOffset: 5,
			bars: []*models.Bar1m{
				{Volume: 1000},
				{Volume: 1000},
				{Volume: 1000},
				{Volume: 1000},
				{Volume: 1000}, // Average: 1000
			},
			liveBar: &models.LiveBar{
				Volume: 1000, // Current: 1000, Relative: 100%
			},
			wantValue: 100.0,
			wantOk:    true,
		},
		{
			name:      "volume double average",
			barOffset: 5,
			bars: []*models.Bar1m{
				{Volume: 1000},
				{Volume: 1000},
				{Volume: 1000},
				{Volume: 1000},
				{Volume: 1000}, // Average: 1000
			},
			liveBar: &models.LiveBar{
				Volume: 2000, // Current: 2000, Relative: 200%
			},
			wantValue: 200.0,
			wantOk:    true,
		},
		{
			name:      "volume half average",
			barOffset: 4,
			bars: []*models.Bar1m{
				{Volume: 1000},
				{Volume: 1000},
				{Volume: 1000},
				{Volume: 1000}, // Average: 1000
			},
			liveBar: &models.LiveBar{
				Volume: 500, // Current: 500, Relative: 50%
			},
			wantValue: 50.0,
			wantOk:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			computer := NewRelativeVolumeComputer("test_relative_volume", tt.barOffset)
			snapshot := &SymbolStateSnapshot{
				LastFinalBars: tt.bars,
				LiveBar:       tt.liveBar,
			}

			value, ok := computer.Compute(snapshot)
			if ok != tt.wantOk {
				t.Errorf("Compute() ok = %v, want %v", ok, tt.wantOk)
			}
			if ok {
				// Allow small floating point differences
				if abs(value-tt.wantValue) > 0.1 {
					t.Errorf("Compute() value = %v, want %v", value, tt.wantValue)
				}
			}
		})
	}
}

func TestRelativeVolumeSameTimeComputer(t *testing.T) {
	tests := []struct {
		name      string
		bars      []*models.Bar1m
		liveBar   *models.LiveBar
		wantOk    bool
	}{
		{
			name:   "insufficient bars",
			bars:   createBarsForActivity(5, 100.0),
			wantOk: false,
		},
		{
			name:   "sufficient bars",
			bars:   createBarsForActivity(10, 100.0),
			wantOk: true,
		},
		{
			name: "with live bar",
			bars: createBarsForActivity(10, 100.0),
			liveBar: &models.LiveBar{
				Volume: 1000,
			},
			wantOk: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			computer := &RelativeVolumeSameTimeComputer{}
			snapshot := &SymbolStateSnapshot{
				LastFinalBars: tt.bars,
				LiveBar:       tt.liveBar,
			}

			value, ok := computer.Compute(snapshot)
			if ok != tt.wantOk {
				t.Errorf("Compute() ok = %v, want %v", ok, tt.wantOk)
			}
			if ok && value <= 0 {
				t.Errorf("Compute() value = %v, want > 0", value)
			}
		})
	}
}

