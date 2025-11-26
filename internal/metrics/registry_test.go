package metrics

import (
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

func TestRegistry_ComputeAll(t *testing.T) {
	registry := NewRegistry()

	// Create a test snapshot
	snapshot := &SymbolStateSnapshot{
		Symbol: "AAPL",
		LiveBar: &models.LiveBar{
			Symbol:    "AAPL",
			Timestamp: time.Now(),
			Open:      150.0,
			High:      151.0,
			Low:       149.0,
			Close:     150.5,
			Volume:    1000,
			VWAPNum:   150500.0,
			VWAPDenom: 1000.0,
		},
		LastFinalBars: []*models.Bar1m{
			{
				Symbol:    "AAPL",
				Timestamp:  time.Now().Add(-2 * time.Minute),
				Open:      149.0,
				High:      150.0,
				Low:       148.0,
				Close:     149.5,
				Volume:    500,
				VWAP:      149.5,
			},
			{
				Symbol:    "AAPL",
				Timestamp:  time.Now().Add(-1 * time.Minute),
				Open:      149.5,
				High:      150.5,
				Low:       149.0,
				Close:     150.0,
				Volume:    600,
				VWAP:      150.0,
			},
		},
		Indicators: map[string]float64{
			"rsi_14": 65.5,
			"ema_20": 150.2,
		},
		LastTickTime: time.Now(),
		LastUpdate:   time.Now(),
	}

	// Compute all metrics
	result := registry.ComputeAll(snapshot)

	// Verify indicators are copied
	if result["rsi_14"] != 65.5 {
		t.Errorf("Expected rsi_14=65.5, got %f", result["rsi_14"])
	}
	if result["ema_20"] != 150.2 {
		t.Errorf("Expected ema_20=150.2, got %f", result["ema_20"])
	}

	// Verify live bar metrics
	if result["price"] != 150.5 {
		t.Errorf("Expected price=150.5, got %f", result["price"])
	}
	if result["volume_live"] != 1000.0 {
		t.Errorf("Expected volume_live=1000.0, got %f", result["volume_live"])
	}
	if result["vwap_live"] != 150.5 {
		t.Errorf("Expected vwap_live=150.5, got %f", result["vwap_live"])
	}

	// Verify finalized bar metrics
	if result["close"] != 150.0 {
		t.Errorf("Expected close=150.0, got %f", result["close"])
	}
	if result["open"] != 149.5 {
		t.Errorf("Expected open=149.5, got %f", result["open"])
	}
	if result["volume"] != 600.0 {
		t.Errorf("Expected volume=600.0, got %f", result["volume"])
	}

	// Verify price change metrics
	expectedChange1m := ((150.0 - 149.5) / 149.5) * 100.0
	if result["price_change_1m_pct"] != expectedChange1m {
		t.Errorf("Expected price_change_1m_pct=%f, got %f", expectedChange1m, result["price_change_1m_pct"])
	}
}

func TestPriceChangeComputer(t *testing.T) {
	computer := NewPriceChangeComputer("price_change_5m_pct", 6)

	if computer.Name() != "price_change_5m_pct" {
		t.Errorf("Expected name 'price_change_5m_pct', got '%s'", computer.Name())
	}

	// Test with insufficient bars
	snapshot := &SymbolStateSnapshot{
		LastFinalBars: []*models.Bar1m{
			{Close: 150.0},
		},
	}

	_, ok := computer.Compute(snapshot)
	if ok {
		t.Error("Expected false when insufficient bars, got true")
	}

	// Test with sufficient bars
	// For barOffset=6, we need at least 6 bars
	// We look back 6 bars from the end, so with 7 bars:
	// Index 0: 149.0
	// Index 1: 149.5 (6 bars ago from index 6)
	// Index 2: 150.0
	// Index 3: 150.5
	// Index 4: 151.0
	// Index 5: 151.5
	// Index 6: 152.0 (current bar, last in array)
	// So we compare index 6 (152.0) with index 0 (149.0) when barOffset=6
	// Wait, that's not right. Let me check: len=7, barOffset=6
	// currentBar = LastFinalBars[len-1] = LastFinalBars[6] = 152.0
	// pastBar = LastFinalBars[len-6] = LastFinalBars[1] = 149.5
	snapshot.LastFinalBars = []*models.Bar1m{
		{Close: 149.0},
		{Close: 149.5}, // This is 6 bars ago from index 6
		{Close: 150.0},
		{Close: 150.5},
		{Close: 151.0},
		{Close: 151.5},
		{Close: 152.0}, // Current bar (last)
	}

	value, ok := computer.Compute(snapshot)
	if !ok {
		t.Error("Expected true when sufficient bars, got false")
	}
	// We compare current (152.0 at index 6) with bar at index 0 (149.0)
	// Actually: len=7, barOffset=6
	// currentBar = LastFinalBars[6] = 152.0
	// pastBar = LastFinalBars[7-6] = LastFinalBars[1] = 149.5
	expected := ((152.0 - 149.5) / 149.5) * 100.0
	if value != expected {
		t.Errorf("Expected %f, got %f", expected, value)
	}
}

func TestPriceComputer(t *testing.T) {
	computer := &PriceComputer{}

	if computer.Name() != "price" {
		t.Errorf("Expected name 'price', got '%s'", computer.Name())
	}

	// Test without live bar
	snapshot := &SymbolStateSnapshot{}
	_, ok := computer.Compute(snapshot)
	if ok {
		t.Error("Expected false when no live bar, got true")
	}

	// Test with live bar
	snapshot.LiveBar = &models.LiveBar{Close: 150.5}
	value, ok := computer.Compute(snapshot)
	if !ok {
		t.Error("Expected true when live bar exists, got false")
	}
	if value != 150.5 {
		t.Errorf("Expected 150.5, got %f", value)
	}
}

