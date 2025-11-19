package indicator

import (
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

func TestVWAP_NewVWAP(t *testing.T) {
	// Valid window
	vwap, err := NewVWAP(5 * time.Minute)
	if err != nil {
		t.Fatalf("Failed to create VWAP: %v", err)
	}
	if vwap == nil {
		t.Fatal("VWAP is nil")
	}
	if vwap.Name() != "vwap_5m" {
		t.Errorf("Expected name 'vwap_5m', got '%s'", vwap.Name())
	}

	// Invalid window
	_, err = NewVWAP(0)
	if err == nil {
		t.Error("Expected error for zero window")
	}

	_, err = NewVWAP(-time.Minute)
	if err == nil {
		t.Error("Expected error for negative window")
	}
}

func TestVWAP_Update(t *testing.T) {
	vwap, _ := NewVWAP(5 * time.Minute)

	baseTime := time.Now()

	// Add bars within the window
	for i := 0; i < 5; i++ {
		bar := &models.Bar1m{
			Symbol:    "AAPL",
			Timestamp: baseTime.Add(time.Duration(i) * time.Minute),
			Open:      100.0,
			High:      105.0,
			Low:       99.0,
			Close:     103.0,
			Volume:    1000,
			VWAP:      102.0,
		}
		val, err := vwap.Update(bar)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		// VWAP should be ready after first bar
		if !vwap.IsReady() {
			t.Error("VWAP should be ready after first bar")
		}
		if val <= 0 {
			t.Errorf("Expected positive VWAP, got %f", val)
		}
	}
}

func TestVWAP_WindowExpiration(t *testing.T) {
	vwap, _ := NewVWAP(5 * time.Minute)

	baseTime := time.Now()

	// Add bars within window
	for i := 0; i < 3; i++ {
		bar := &models.Bar1m{
			Symbol:    "AAPL",
			Timestamp: baseTime.Add(time.Duration(i) * time.Minute),
			Close:     100.0,
			Volume:    1000,
		}
		_, _ = vwap.Update(bar)
	}

	// Add bar outside window (6 minutes later)
	bar := &models.Bar1m{
		Symbol:    "AAPL",
		Timestamp: baseTime.Add(6 * time.Minute),
		Close:     110.0,
		Volume:    1000,
	}
	_, _ = vwap.Update(bar)

	// Should only have the last bar in window
	if !vwap.IsReady() {
		t.Error("VWAP should still be ready")
	}
}

func TestVWAP_Calculation(t *testing.T) {
	vwap, _ := NewVWAP(10 * time.Minute)

	baseTime := time.Now()

	// Add bars with known values
	bars := []*models.Bar1m{
		{Timestamp: baseTime, Close: 100.0, High: 102.0, Low: 98.0, Volume: 1000},
		{Timestamp: baseTime.Add(time.Minute), Close: 105.0, High: 107.0, Low: 103.0, Volume: 2000},
		{Timestamp: baseTime.Add(2 * time.Minute), Close: 110.0, High: 112.0, Low: 108.0, Volume: 1500},
	}

	for _, bar := range bars {
		_, _ = vwap.Update(bar)
	}

	val, _ := vwap.Value()
	// VWAP should be calculated using typical price (HLC/3)
	// Bar 1: (102+98+100)/3 = 100, volume 1000
	// Bar 2: (107+103+105)/3 = 104.67, volume 2000
	// Bar 3: (112+108+110)/3 = 110, volume 1500
	// Total price*volume / total volume
	expected := (100.0*1000 + 104.67*2000 + 110.0*1500) / (1000 + 2000 + 1500)
	if val < expected-1 || val > expected+1 {
		t.Errorf("Expected VWAP around %f, got %f", expected, val)
	}
}

func TestVWAP_Reset(t *testing.T) {
	vwap, _ := NewVWAP(5 * time.Minute)

	baseTime := time.Now()
	for i := 0; i < 3; i++ {
		bar := &models.Bar1m{
			Symbol:    "AAPL",
			Timestamp: baseTime.Add(time.Duration(i) * time.Minute),
			Close:     100.0,
			Volume:    1000,
		}
		_, _ = vwap.Update(bar)
	}

	vwap.Reset()

	if vwap.IsReady() {
		t.Error("VWAP should not be ready after reset")
	}

	val, err := vwap.Value()
	if err == nil {
		t.Errorf("Expected error after reset, got value %f", val)
	}
}

