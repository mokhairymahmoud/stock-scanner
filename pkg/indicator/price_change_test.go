package indicator

import (
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

func TestPriceChange_NewPriceChange(t *testing.T) {
	pc, err := NewPriceChange(5 * time.Minute)
	if err != nil {
		t.Fatalf("Failed to create PriceChange: %v", err)
	}
	if pc == nil {
		t.Fatal("PriceChange is nil")
	}
	if pc.Name() != "price_change_5m_pct" {
		t.Errorf("Expected name 'price_change_5m_pct', got '%s'", pc.Name())
	}
}

func TestPriceChange_Update(t *testing.T) {
	pc, _ := NewPriceChange(5 * time.Minute)

	baseTime := time.Now()

	// First bar
	bar1 := &models.Bar1m{
		Symbol:    "AAPL",
		Timestamp: baseTime,
		Close:     100.0,
	}
	val, err := pc.Update(bar1)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if pc.IsReady() {
		t.Error("PriceChange should not be ready after first bar")
	}
	if val != 0 {
		t.Errorf("Expected 0 for first bar, got %f", val)
	}

	// Second bar - price increase
	bar2 := &models.Bar1m{
		Symbol:    "AAPL",
		Timestamp: baseTime.Add(time.Minute),
		Close:     105.0, // 5% increase
	}
	val, err = pc.Update(bar2)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if !pc.IsReady() {
		t.Error("PriceChange should be ready after second bar")
	}
	if val != 5.0 {
		t.Errorf("Expected 5.0%% change, got %f", val)
	}
}

func TestPriceChange_Decrease(t *testing.T) {
	pc, _ := NewPriceChange(5 * time.Minute)

	baseTime := time.Now()

	bar1 := &models.Bar1m{
		Symbol:    "AAPL",
		Timestamp: baseTime,
		Close:     100.0,
	}
	_, _ = pc.Update(bar1)

	bar2 := &models.Bar1m{
		Symbol:    "AAPL",
		Timestamp: baseTime.Add(time.Minute),
		Close:     95.0, // 5% decrease
	}
	val, _ := pc.Update(bar2)

	if val != -5.0 {
		t.Errorf("Expected -5.0%% change, got %f", val)
	}
}

func TestPriceChange_WindowExpiration(t *testing.T) {
	pc, _ := NewPriceChange(5 * time.Minute)

	baseTime := time.Now()

	// Add bar at start
	bar1 := &models.Bar1m{
		Symbol:    "AAPL",
		Timestamp: baseTime,
		Close:     100.0,
	}
	_, _ = pc.Update(bar1)

	// Add bar within window
	bar2 := &models.Bar1m{
		Symbol:    "AAPL",
		Timestamp: baseTime.Add(3 * time.Minute),
		Close:     105.0,
	}
	_, _ = pc.Update(bar2)

	// Add bar outside window (6 minutes later)
	bar3 := &models.Bar1m{
		Symbol:    "AAPL",
		Timestamp: baseTime.Add(6 * time.Minute),
		Close:     110.0,
	}
	_, _ = pc.Update(bar3)

	// Should calculate change from bar2 to bar3
	val, _ := pc.Value()
	expected := ((110.0 - 105.0) / 105.0) * 100.0
	if val < expected-0.1 || val > expected+0.1 {
		t.Errorf("Expected change around %f, got %f", expected, val)
	}
}

func TestPriceChange_Reset(t *testing.T) {
	pc, _ := NewPriceChange(5 * time.Minute)

	baseTime := time.Now()
	bar1 := &models.Bar1m{
		Symbol:    "AAPL",
		Timestamp: baseTime,
		Close:     100.0,
	}
	_, _ = pc.Update(bar1)

	bar2 := &models.Bar1m{
		Symbol:    "AAPL",
		Timestamp: baseTime.Add(time.Minute),
		Close:     105.0,
	}
	_, _ = pc.Update(bar2)

	pc.Reset()

	if pc.IsReady() {
		t.Error("PriceChange should not be ready after reset")
	}

	val, err := pc.Value()
	if err == nil {
		t.Errorf("Expected error after reset, got value %f", val)
	}
}

