package indicator

import (
	"math"
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

func TestRSI_NewRSI(t *testing.T) {
	// Valid period
	rsi, err := NewRSI(14)
	if err != nil {
		t.Fatalf("Failed to create RSI: %v", err)
	}
	if rsi == nil {
		t.Fatal("RSI is nil")
	}
	if rsi.Name() != "rsi_14" {
		t.Errorf("Expected name 'rsi_14', got '%s'", rsi.Name())
	}

	// Invalid period
	_, err = NewRSI(1)
	if err == nil {
		t.Error("Expected error for period < 2")
	}
}

func TestRSI_Update(t *testing.T) {
	rsi, _ := NewRSI(14)

	// First bar - should not be ready
	bar1 := &models.Bar1m{
		Symbol:    "AAPL",
		Timestamp: time.Now(),
		Close:     100.0,
	}

	val, err := rsi.Update(bar1)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if val != 0 {
		t.Errorf("Expected 0 for first bar, got %f", val)
	}
	if rsi.IsReady() {
		t.Error("RSI should not be ready after first bar")
	}

	// Add 14 more bars with gains
	for i := 2; i <= 15; i++ {
		bar := &models.Bar1m{
			Symbol:    "AAPL",
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			Close:     100.0 + float64(i), // Increasing price
		}
		val, err = rsi.Update(bar)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}
	}

	// Should be ready now
	if !rsi.IsReady() {
		t.Error("RSI should be ready after 15 bars")
	}

	// RSI should be high (mostly gains)
	val, err = rsi.Value()
	if err != nil {
		t.Fatalf("Value() failed: %v", err)
	}
	if val < 50 || val > 100 {
		t.Errorf("Expected RSI between 50-100 for gains, got %f", val)
	}
}

func TestRSI_Reset(t *testing.T) {
	rsi, _ := NewRSI(14)

	// Add some bars
	for i := 0; i < 15; i++ {
		bar := &models.Bar1m{
			Symbol:    "AAPL",
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			Close:     100.0 + float64(i),
		}
		_, _ = rsi.Update(bar)
	}

	rsi.Reset()

	if rsi.IsReady() {
		t.Error("RSI should not be ready after reset")
	}

	val, err := rsi.Value()
	if err == nil {
		t.Errorf("Expected error after reset, got value %f", val)
	}
}

func TestRSI_AllGains(t *testing.T) {
	rsi, _ := NewRSI(14)

	// Create bars with all gains
	baseTime := time.Now()
	bar := &models.Bar1m{
		Symbol:    "AAPL",
		Timestamp: baseTime,
		Close:     100.0,
	}
	_, _ = rsi.Update(bar)

	for i := 1; i <= 14; i++ {
		bar := &models.Bar1m{
			Symbol:    "AAPL",
			Timestamp: baseTime.Add(time.Duration(i) * time.Minute),
			Close:     100.0 + float64(i)*2, // Always increasing
		}
		_, _ = rsi.Update(bar)
	}

	val, _ := rsi.Value()
	// RSI should be very high (close to 100) for all gains
	if val < 90 {
		t.Errorf("Expected high RSI for all gains, got %f", val)
	}
}

func TestRSI_AllLosses(t *testing.T) {
	rsi, _ := NewRSI(14)

	// Create bars with all losses
	baseTime := time.Now()
	bar := &models.Bar1m{
		Symbol:    "AAPL",
		Timestamp: baseTime,
		Close:     100.0,
	}
	_, _ = rsi.Update(bar)

	for i := 1; i <= 14; i++ {
		bar := &models.Bar1m{
			Symbol:    "AAPL",
			Timestamp: baseTime.Add(time.Duration(i) * time.Minute),
			Close:     100.0 - float64(i)*2, // Always decreasing
		}
		_, _ = rsi.Update(bar)
	}

	val, _ := rsi.Value()
	// RSI should be very low (close to 0) for all losses
	if val > 10 {
		t.Errorf("Expected low RSI for all losses, got %f", val)
	}
}

func TestRSI_Clamp(t *testing.T) {
	rsi, _ := NewRSI(14)

	// Test that RSI values are clamped to 0-100
	val, _ := rsi.Value()
	if math.IsNaN(val) || math.IsInf(val, 0) {
		t.Error("RSI should not be NaN or Inf")
	}
	if val < 0 || val > 100 {
		t.Errorf("RSI should be between 0-100, got %f", val)
	}
}

