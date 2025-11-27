package metrics

import (
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

func TestRangeComputer(t *testing.T) {
	tests := []struct {
		name      string
		barOffset int
		bars      []*models.Bar1m
		wantValue float64
		wantOk    bool
	}{
		{
			name:      "insufficient bars",
			barOffset: 5,
			bars:      createBarsForRange(3, 100.0),
			wantValue: 0,
			wantOk:    false,
		},
		{
			name:      "range over 5 bars",
			barOffset: 5,
			bars: []*models.Bar1m{
				createBarForRange(100.0, 110.0, 95.0, 105.0, 1000),
				createBarForRange(105.0, 115.0, 100.0, 110.0, 1000),
				createBarForRange(110.0, 120.0, 105.0, 115.0, 1000),
				createBarForRange(115.0, 125.0, 110.0, 120.0, 1000),
				createBarForRange(120.0, 130.0, 115.0, 125.0, 1000),
			},
			wantValue: 35.0, // High: 130, Low: 95, Range: 35
			wantOk:    true,
		},
		{
			name:      "range with decreasing prices",
			barOffset: 3,
			bars: []*models.Bar1m{
				createBarForRange(100.0, 105.0, 95.0, 100.0, 1000),
				createBarForRange(100.0, 102.0, 90.0, 95.0, 1000),
				createBarForRange(95.0, 100.0, 85.0, 90.0, 1000),
			},
			wantValue: 20.0, // High: 105, Low: 85, Range: 20
			wantOk:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			computer := NewRangeComputer("test_range", tt.barOffset)
			snapshot := &SymbolStateSnapshot{
				LastFinalBars: tt.bars,
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

func TestRangePercentageComputer(t *testing.T) {
	tests := []struct {
		name      string
		barOffset int
		bars      []*models.Bar1m
		wantValue float64
		wantOk    bool
	}{
		{
			name:      "insufficient bars",
			barOffset: 5,
			bars:      createBarsForRange(3, 100.0),
			wantValue: 0,
			wantOk:    false,
		},
		{
			name:      "range percentage over 5 bars",
			barOffset: 5,
			bars: []*models.Bar1m{
				createBarForRange(100.0, 110.0, 95.0, 105.0, 1000),
				createBarForRange(105.0, 115.0, 100.0, 110.0, 1000),
				createBarForRange(110.0, 120.0, 105.0, 115.0, 1000),
				createBarForRange(115.0, 125.0, 110.0, 120.0, 1000),
				createBarForRange(120.0, 130.0, 115.0, 125.0, 1000),
			},
			wantValue: 36.84, // High: 130, Low: 95, Range: 35, Range%: (35/95)*100 ≈ 36.84
			wantOk:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			computer := NewRangePercentageComputer("test_range_pct", tt.barOffset)
			snapshot := &SymbolStateSnapshot{
				LastFinalBars: tt.bars,
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

func TestDailyRangeComputer(t *testing.T) {
	tests := []struct {
		name      string
		bars      []*models.Bar1m
		liveBar   *models.LiveBar
		wantValue float64
		wantOk    bool
	}{
		{
			name:      "no bars",
			bars:      []*models.Bar1m{},
			wantValue: 0,
			wantOk:    false,
		},
		{
			name: "range from finalized bars only",
			bars: []*models.Bar1m{
				createBarForRange(100.0, 110.0, 95.0, 105.0, 1000),
				createBarForRange(105.0, 115.0, 100.0, 110.0, 1000),
				createBarForRange(110.0, 120.0, 105.0, 115.0, 1000),
			},
			wantValue: 25.0, // High: 120, Low: 95, Range: 25
			wantOk:    true,
		},
		{
			name: "range including live bar",
			bars: []*models.Bar1m{
				createBarForRange(100.0, 110.0, 95.0, 105.0, 1000),
				createBarForRange(105.0, 115.0, 100.0, 110.0, 1000),
			},
			liveBar: &models.LiveBar{
				High:  125.0,
				Low:   90.0,
				Close: 120.0,
			},
			wantValue: 35.0, // High: 125, Low: 90, Range: 35
			wantOk:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			computer := &DailyRangeComputer{}
			snapshot := &SymbolStateSnapshot{
				LastFinalBars: tt.bars,
				LiveBar:       tt.liveBar,
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

func TestPositionInRangeComputer(t *testing.T) {
	tests := []struct {
		name      string
		barOffset int
		bars      []*models.Bar1m
		liveBar   *models.LiveBar
		wantValue float64
		wantOk    bool
	}{
		{
			name:      "insufficient bars",
			barOffset: 5,
			bars:      createBarsForRange(3, 100.0),
			wantValue: 0,
			wantOk:    false,
		},
		{
			name:      "position at bottom of range",
			barOffset: 3,
			bars: []*models.Bar1m{
				createBarForRange(100.0, 110.0, 95.0, 105.0, 1000),
				createBarForRange(105.0, 115.0, 100.0, 110.0, 1000),
				createBarForRange(110.0, 120.0, 105.0, 115.0, 1000),
			},
			liveBar: &models.LiveBar{
				High:  95.0, // Live bar at the low
				Low:   95.0,
				Close: 95.0, // At the low
			},
			wantValue: 0.0, // (95-95)/(120-95) * 100 = 0%
			wantOk:    true,
		},
		{
			name:      "position at top of range",
			barOffset: 3,
			bars: []*models.Bar1m{
				createBarForRange(100.0, 110.0, 95.0, 105.0, 1000),
				createBarForRange(105.0, 115.0, 100.0, 110.0, 1000),
				createBarForRange(110.0, 120.0, 105.0, 115.0, 1000),
			},
			liveBar: &models.LiveBar{
				Close: 120.0, // At the high
			},
			wantValue: 100.0, // (120-95)/(120-95) * 100 = 100%
			wantOk:    true,
		},
		{
			name:      "position in middle of range",
			barOffset: 3,
			bars: []*models.Bar1m{
				createBarForRange(100.0, 110.0, 95.0, 105.0, 1000),
				createBarForRange(105.0, 115.0, 100.0, 110.0, 1000),
				createBarForRange(110.0, 120.0, 105.0, 115.0, 1000),
			},
			liveBar: &models.LiveBar{
				High:  107.5, // Live bar in the middle
				Low:   107.5,
				Close: 107.5, // Middle: (107.5-95)/(120-95) * 100 = 50%
			},
			wantValue: 50.0,
			wantOk:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			computer := NewPositionInRangeComputer("test_position", tt.barOffset)
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

func TestRelativeRangeComputer(t *testing.T) {
	tests := []struct {
		name       string
		bars       []*models.Bar1m
		liveBar    *models.LiveBar
		indicators map[string]float64
		wantValue  float64
		wantOk     bool
	}{
		{
			name:       "no ATR indicator",
			bars:       createBarsForRange(5, 100.0),
			liveBar:    nil,
			indicators: map[string]float64{},
			wantValue:  0,
			wantOk:     false,
		},
		{
			name: "relative range calculation",
			bars: []*models.Bar1m{
				createBarForRange(100.0, 110.0, 95.0, 105.0, 1000),
				createBarForRange(105.0, 115.0, 100.0, 110.0, 1000),
				createBarForRange(110.0, 120.0, 105.0, 115.0, 1000),
			},
			liveBar: &models.LiveBar{
				High:  125.0,
				Low:   90.0,
				Close: 120.0,
			},
			indicators: map[string]float64{
				"atr_14": 10.0, // ATR is 10 (using atr_14 until daily ATR is implemented)
			},
			wantValue: 350.0, // Today's range: 35, ATR: 10, Relative: (35/10)*100 = 350%
			wantOk:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			computer := &RelativeRangeComputer{}
			snapshot := &SymbolStateSnapshot{
				LastFinalBars: tt.bars,
				LiveBar:       tt.liveBar,
				Indicators:    tt.indicators,
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

// Helper functions

func createBarForRange(open, high, low, close float64, volume int64) *models.Bar1m {
	return &models.Bar1m{
		Symbol:    "TEST",
		Timestamp: time.Now(),
		Open:      open,
		High:      high,
		Low:       low,
		Close:     close,
		Volume:    volume,
	}
}

func createBarsForRange(count int, basePrice float64) []*models.Bar1m {
	bars := make([]*models.Bar1m, count)
	for i := 0; i < count; i++ {
		bars[i] = createBarForRange(basePrice, basePrice+5, basePrice-5, basePrice, 1000)
	}
	return bars
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

