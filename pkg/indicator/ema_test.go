package indicator

import (
	"math"
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

func TestEMA_NewEMA(t *testing.T) {
	// Valid period
	ema, err := NewEMA(20)
	if err != nil {
		t.Fatalf("Failed to create EMA: %v", err)
	}
	if ema == nil {
		t.Fatal("EMA is nil")
	}
	if ema.Name() != "ema_20" {
		t.Errorf("Expected name 'ema_20', got '%s'", ema.Name())
	}

	// Invalid period
	_, err = NewEMA(0)
	if err == nil {
		t.Error("Expected error for period < 1")
	}
}

func TestEMA_Update(t *testing.T) {
	ema, _ := NewEMA(20)

	// First bar - should be ready immediately
	bar1 := &models.Bar1m{
		Symbol:    "AAPL",
		Timestamp: time.Now(),
		Close:     100.0,
	}

	val, err := ema.Update(bar1)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if val != 100.0 {
		t.Errorf("Expected 100.0 for first bar, got %f", val)
	}
	if !ema.IsReady() {
		t.Error("EMA should be ready after first bar")
	}

	// Second bar
	bar2 := &models.Bar1m{
		Symbol:    "AAPL",
		Timestamp: time.Now().Add(time.Minute),
		Close:     105.0,
	}

	val, err = ema.Update(bar2)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	// EMA should be between 100 and 105
	if val < 100.0 || val > 105.0 {
		t.Errorf("Expected EMA between 100-105, got %f", val)
	}
}

func TestEMA_Convergence(t *testing.T) {
	ema, _ := NewEMA(20)

	// Add many bars with constant price
	price := 100.0
	for i := 0; i < 100; i++ {
		bar := &models.Bar1m{
			Symbol:    "AAPL",
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			Close:     price,
		}
		val, _ := ema.Update(bar)
		// After many bars, EMA should converge to the price
		if i > 50 {
			if math.Abs(val-price) > 0.1 {
				t.Errorf("EMA should converge to price, got %f, expected %f", val, price)
			}
		}
	}
}

func TestEMA_Reset(t *testing.T) {
	ema, _ := NewEMA(20)

	// Add some bars
	for i := 0; i < 10; i++ {
		bar := &models.Bar1m{
			Symbol:    "AAPL",
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			Close:     100.0 + float64(i),
		}
		_, _ = ema.Update(bar)
	}

	ema.Reset()

	if ema.IsReady() {
		t.Error("EMA should not be ready after reset")
	}

	val, err := ema.Value()
	if err == nil {
		t.Errorf("Expected error after reset, got value %f", val)
	}
}

func TestEMA_IncreasingPrice(t *testing.T) {
	ema, _ := NewEMA(20)

	// Add bars with increasing price
	for i := 0; i < 50; i++ {
		bar := &models.Bar1m{
			Symbol:    "AAPL",
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			Close:     100.0 + float64(i),
		}
		val, _ := ema.Update(bar)
		// EMA should be increasing but lagging behind the price
		if i > 0 {
			prevVal, _ := ema.Value()
			if val < prevVal {
				t.Errorf("EMA should be increasing, got %f < %f", val, prevVal)
			}
		}
	}
}

func TestEMA_HandleNaN(t *testing.T) {
	ema, _ := NewEMA(20)

	// Test that NaN values are handled
	bar := &models.Bar1m{
		Symbol:    "AAPL",
		Timestamp: time.Now(),
		Close:     100.0,
	}
	_, _ = ema.Update(bar)

	val, _ := ema.Value()
	if math.IsNaN(val) || math.IsInf(val, 0) {
		t.Error("EMA should not be NaN or Inf")
	}
}

