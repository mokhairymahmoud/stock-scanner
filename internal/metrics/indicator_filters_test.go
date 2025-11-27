package metrics

import (
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

func TestATRPComputer(t *testing.T) {
	tests := []struct {
		name       string
		atrKey     string
		bars       []*models.Bar1m
		liveBar    *models.LiveBar
		indicators map[string]float64
		wantValue  float64
		wantOk     bool
	}{
		{
			name:       "no ATR indicator",
			atrKey:     "atr_14",
			bars:       createBarsForIndicator(5, 100.0),
			indicators: map[string]float64{},
			wantValue:  0,
			wantOk:     false,
		},
		{
			name:   "ATRP calculation with live bar",
			atrKey: "atr_14",
			bars:   createBarsForIndicator(5, 100.0),
			liveBar: &models.LiveBar{
				Close: 100.0,
			},
			indicators: map[string]float64{
				"atr_14": 5.0, // ATR is 5
			},
			wantValue: 5.0, // (5/100)*100 = 5%
			wantOk:    true,
		},
		{
			name:   "ATRP calculation with finalized bar",
			atrKey: "atr_14",
			bars: []*models.Bar1m{
				createBarForIndicator(100.0, 105.0, 95.0, 100.0, 1000),
			},
			indicators: map[string]float64{
				"atr_14": 10.0, // ATR is 10
			},
			wantValue: 10.0, // (10/100)*100 = 10%
			wantOk:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			computer := NewATRPComputer("test_atrp", tt.atrKey)
			snapshot := &SymbolStateSnapshot{
				LastFinalBars: tt.bars,
				LiveBar:       tt.liveBar,
				Indicators:    tt.indicators,
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

func TestVWAPDistanceComputer(t *testing.T) {
	tests := []struct {
		name       string
		vwapKey    string
		bars       []*models.Bar1m
		liveBar    *models.LiveBar
		indicators map[string]float64
		wantValue  float64
		wantOk     bool
	}{
		{
			name:       "no VWAP indicator",
			vwapKey:    "vwap_5m",
			bars:       createBarsForIndicator(5, 100.0),
			indicators: map[string]float64{},
			wantValue:  0,
			wantOk:     false,
		},
		{
			name:   "VWAP distance above VWAP",
			vwapKey: "vwap_5m",
			bars:   createBarsForIndicator(5, 100.0),
			liveBar: &models.LiveBar{
				Close: 105.0,
			},
			indicators: map[string]float64{
				"vwap_5m": 100.0,
			},
			wantValue: 5.0, // |105-100| = 5
			wantOk:    true,
		},
		{
			name:   "VWAP distance below VWAP",
			vwapKey: "vwap_5m",
			bars:   createBarsForIndicator(5, 100.0),
			liveBar: &models.LiveBar{
				Close: 95.0,
			},
			indicators: map[string]float64{
				"vwap_5m": 100.0,
			},
			wantValue: 5.0, // |95-100| = 5 (absolute)
			wantOk:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			computer := NewVWAPDistanceComputer("test_vwap_dist", tt.vwapKey)
			snapshot := &SymbolStateSnapshot{
				LastFinalBars: tt.bars,
				LiveBar:       tt.liveBar,
				Indicators:    tt.indicators,
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

func TestVWAPDistancePctComputer(t *testing.T) {
	tests := []struct {
		name       string
		vwapKey    string
		bars       []*models.Bar1m
		liveBar    *models.LiveBar
		indicators map[string]float64
		wantValue  float64
		wantOk     bool
	}{
		{
			name:       "no VWAP indicator",
			vwapKey:    "vwap_5m",
			bars:       createBarsForIndicator(5, 100.0),
			indicators: map[string]float64{},
			wantValue:  0,
			wantOk:     false,
		},
		{
			name:   "VWAP distance percentage above VWAP",
			vwapKey: "vwap_5m",
			bars:   createBarsForIndicator(5, 100.0),
			liveBar: &models.LiveBar{
				Close: 105.0,
			},
			indicators: map[string]float64{
				"vwap_5m": 100.0,
			},
			wantValue: 5.0, // |(105-100)/100|*100 = 5%
			wantOk:    true,
		},
		{
			name:   "VWAP distance percentage below VWAP",
			vwapKey: "vwap_5m",
			bars:   createBarsForIndicator(5, 100.0),
			liveBar: &models.LiveBar{
				Close: 95.0,
			},
			indicators: map[string]float64{
				"vwap_5m": 100.0,
			},
			wantValue: 5.0, // |(95-100)/100|*100 = 5% (absolute)
			wantOk:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			computer := NewVWAPDistancePctComputer("test_vwap_dist_pct", tt.vwapKey)
			snapshot := &SymbolStateSnapshot{
				LastFinalBars: tt.bars,
				LiveBar:       tt.liveBar,
				Indicators:    tt.indicators,
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

func TestMADistanceComputer(t *testing.T) {
	tests := []struct {
		name       string
		maKey      string
		bars       []*models.Bar1m
		liveBar    *models.LiveBar
		indicators map[string]float64
		wantValue  float64
		wantOk     bool
	}{
		{
			name:       "no MA indicator",
			maKey:      "ema_9",
			bars:       createBarsForIndicator(5, 100.0),
			indicators: map[string]float64{},
			wantValue:  0,
			wantOk:     false,
		},
		{
			name:   "MA distance percentage above MA",
			maKey:  "ema_9",
			bars:  createBarsForIndicator(5, 100.0),
			liveBar: &models.LiveBar{
				Close: 110.0,
			},
			indicators: map[string]float64{
				"ema_9": 100.0,
			},
			wantValue: 10.0, // (110-100)/100*100 = 10%
			wantOk:    true,
		},
		{
			name:   "MA distance percentage below MA",
			maKey:  "ema_9",
			bars:  createBarsForIndicator(5, 100.0),
			liveBar: &models.LiveBar{
				Close: 90.0,
			},
			indicators: map[string]float64{
				"ema_9": 100.0,
			},
			wantValue: -10.0, // (90-100)/100*100 = -10%
			wantOk:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			computer := NewMADistanceComputer("test_ma_dist", tt.maKey)
			snapshot := &SymbolStateSnapshot{
				LastFinalBars: tt.bars,
				LiveBar:       tt.liveBar,
				Indicators:    tt.indicators,
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

func createBarForIndicator(open, high, low, close float64, volume int64) *models.Bar1m {
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

func createBarsForIndicator(count int, basePrice float64) []*models.Bar1m {
	bars := make([]*models.Bar1m, count)
	for i := 0; i < count; i++ {
		bars[i] = createBarForIndicator(basePrice, basePrice+5, basePrice-5, basePrice, 1000)
	}
	return bars
}

