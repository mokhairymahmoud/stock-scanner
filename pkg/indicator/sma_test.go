package indicator

import (
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

func TestSMA_NewSMA(t *testing.T) {
	// Valid period
	sma, err := NewSMA(20)
	if err != nil {
		t.Fatalf("Failed to create SMA: %v", err)
	}
	if sma == nil {
		t.Fatal("SMA is nil")
	}
	if sma.Name() != "sma_20" {
		t.Errorf("Expected name 'sma_20', got '%s'", sma.Name())
	}

	// Invalid period
	_, err = NewSMA(0)
	if err == nil {
		t.Error("Expected error for period < 1")
	}
}

func TestSMA_Update(t *testing.T) {
	sma, _ := NewSMA(5)

	// Add bars one by one
	for i := 0; i < 4; i++ {
		bar := &models.Bar1m{
			Symbol:    "AAPL",
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			Close:     100.0 + float64(i),
		}
		val, err := sma.Update(bar)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		if i < 4 {
			// Not ready yet
			if sma.IsReady() {
				t.Errorf("SMA should not be ready after %d bars", i+1)
			}
			if val != 0 {
				t.Errorf("Expected 0 for incomplete SMA, got %f", val)
			}
		}
	}

	// 5th bar should make it ready
	bar5 := &models.Bar1m{
		Symbol:    "AAPL",
		Timestamp: time.Now().Add(4 * time.Minute),
		Close:     104.0,
	}
	val, err := sma.Update(bar5)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if !sma.IsReady() {
		t.Error("SMA should be ready after 5 bars")
	}
	// SMA of 100, 101, 102, 103, 104 = 102
	expected := (100.0 + 101.0 + 102.0 + 103.0 + 104.0) / 5.0
	if val != expected {
		t.Errorf("Expected SMA %f, got %f", expected, val)
	}
}

func TestSMA_RollingWindow(t *testing.T) {
	sma, _ := NewSMA(5)

	// Add 10 bars
	for i := 0; i < 10; i++ {
		bar := &models.Bar1m{
			Symbol:    "AAPL",
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			Close:     100.0 + float64(i),
		}
		_, _ = sma.Update(bar)
	}

	// SMA should be average of last 5 bars: 105, 106, 107, 108, 109
	val, _ := sma.Value()
	expected := (105.0 + 106.0 + 107.0 + 108.0 + 109.0) / 5.0
	if val != expected {
		t.Errorf("Expected SMA %f, got %f", expected, val)
	}
}

func TestSMA_Reset(t *testing.T) {
	sma, _ := NewSMA(5)

	// Add some bars
	for i := 0; i < 10; i++ {
		bar := &models.Bar1m{
			Symbol:    "AAPL",
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			Close:     100.0 + float64(i),
		}
		_, _ = sma.Update(bar)
	}

	sma.Reset()

	if sma.IsReady() {
		t.Error("SMA should not be ready after reset")
	}

	val, err := sma.Value()
	if err == nil {
		t.Errorf("Expected error after reset, got value %f", val)
	}
}

func TestSMA_ConstantPrice(t *testing.T) {
	sma, _ := NewSMA(10)

	// Add bars with constant price
	price := 100.0
	for i := 0; i < 10; i++ {
		bar := &models.Bar1m{
			Symbol:    "AAPL",
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			Close:     price,
		}
		_, _ = sma.Update(bar)
	}

	val, _ := sma.Value()
	if val != price {
		t.Errorf("Expected SMA %f for constant price, got %f", price, val)
	}
}

