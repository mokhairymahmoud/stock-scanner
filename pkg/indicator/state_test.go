package indicator

import (
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

func TestSymbolState_Update(t *testing.T) {
	state := NewSymbolState("AAPL", 10)

	calc := &mockCalculator{name: "test"}
	state.AddCalculator(calc)

	bar := &models.Bar1m{
		Symbol:    "AAPL",
		Timestamp: time.Now(),
		Open:      100.0,
		High:      105.0,
		Low:       99.0,
		Close:     103.0,
		Volume:    1000,
		VWAP:      102.0,
	}

	err := state.Update(bar)
	if err != nil {
		t.Fatalf("Failed to update state: %v", err)
	}

	bars := state.GetBars()
	if len(bars) != 1 {
		t.Errorf("Expected 1 bar, got %d", len(bars))
	}

	if bars[0].Symbol != "AAPL" {
		t.Error("Bar symbol mismatch")
	}
}

func TestSymbolState_RingBuffer(t *testing.T) {
	state := NewSymbolState("AAPL", 3) // Max 3 bars

	// Add 5 bars, should only keep last 3
	for i := 0; i < 5; i++ {
		bar := &models.Bar1m{
			Symbol:    "AAPL",
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			Close:     float64(i),
		}
		_ = state.Update(bar)
	}

	bars := state.GetBars()
	if len(bars) != 3 {
		t.Errorf("Expected 3 bars (ring buffer limit), got %d", len(bars))
	}

	// Should have bars 2, 3, 4 (last 3)
	if bars[0].Close != 2.0 {
		t.Errorf("Expected first bar close to be 2.0, got %f", bars[0].Close)
	}
	if bars[2].Close != 4.0 {
		t.Errorf("Expected last bar close to be 4.0, got %f", bars[2].Close)
	}
}

func TestSymbolState_GetValue(t *testing.T) {
	state := NewSymbolState("AAPL", 10)

	calc := &mockCalculator{name: "test"}
	state.AddCalculator(calc)

	// Update with 2 bars to make calculator ready
	bar1 := &models.Bar1m{Symbol: "AAPL", Timestamp: time.Now()}
	bar2 := &models.Bar1m{Symbol: "AAPL", Timestamp: time.Now()}

	_ = state.Update(bar1)
	_ = state.Update(bar2)

	value, err := state.GetValue("test")
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}

	if value != 2.0 {
		t.Errorf("Expected value 2.0, got %f", value)
	}

	// Get value for non-existent calculator
	value, err = state.GetValue("nonexistent")
	if err != nil {
		t.Errorf("Expected no error for non-existent calculator, got %v", err)
	}
	if value != 0 {
		t.Errorf("Expected 0 for non-existent calculator, got %f", value)
	}
}

func TestSymbolState_GetAllValues(t *testing.T) {
	state := NewSymbolState("AAPL", 10)

	calc1 := &mockCalculator{name: "test1"}
	calc2 := &mockCalculator{name: "test2"}

	state.AddCalculator(calc1)
	state.AddCalculator(calc2)

	// Update with 2 bars to make calculators ready
	bar1 := &models.Bar1m{Symbol: "AAPL", Timestamp: time.Now()}
	bar2 := &models.Bar1m{Symbol: "AAPL", Timestamp: time.Now()}

	_ = state.Update(bar1)
	_ = state.Update(bar2)

	values := state.GetAllValues()
	if len(values) != 2 {
		t.Errorf("Expected 2 values, got %d", len(values))
	}

	if values["test1"] != 2.0 {
		t.Errorf("Expected test1 value 2.0, got %f", values["test1"])
	}
	if values["test2"] != 2.0 {
		t.Errorf("Expected test2 value 2.0, got %f", values["test2"])
	}
}

func TestSymbolState_Reset(t *testing.T) {
	state := NewSymbolState("AAPL", 10)

	calc := &mockCalculator{name: "test"}
	state.AddCalculator(calc)

	bar := &models.Bar1m{Symbol: "AAPL", Timestamp: time.Now()}
	_ = state.Update(bar)
	_ = state.Update(bar)

	state.Reset()

	bars := state.GetBars()
	if len(bars) != 0 {
		t.Errorf("Expected 0 bars after reset, got %d", len(bars))
	}

	value, _ := state.GetValue("test")
	if value != 0 {
		t.Errorf("Expected value 0 after reset, got %f", value)
	}
}

func TestSymbolState_Rehydrate(t *testing.T) {
	state := NewSymbolState("AAPL", 10)

	calc := &mockCalculator{name: "test"}
	state.AddCalculator(calc)

	// Create historical bars
	bars := make([]*models.Bar1m, 5)
	for i := 0; i < 5; i++ {
		bars[i] = &models.Bar1m{
			Symbol:    "AAPL",
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			Close:     float64(i),
		}
	}

	err := state.Rehydrate(bars)
	if err != nil {
		t.Fatalf("Failed to rehydrate: %v", err)
	}

	stateBars := state.GetBars()
	if len(stateBars) != 5 {
		t.Errorf("Expected 5 bars after rehydration, got %d", len(stateBars))
	}

	// Calculator should have processed all bars
	value, _ := state.GetValue("test")
	if value != 5.0 {
		t.Errorf("Expected value 5.0 after rehydration, got %f", value)
	}
}

func TestSymbolState_IgnoreWrongSymbol(t *testing.T) {
	state := NewSymbolState("AAPL", 10)

	calc := &mockCalculator{name: "test"}
	state.AddCalculator(calc)

	// Update with bar for different symbol
	bar := &models.Bar1m{
		Symbol:    "MSFT",
		Timestamp: time.Now(),
	}

	err := state.Update(bar)
	if err != nil {
		t.Fatalf("Update should not error for wrong symbol: %v", err)
	}

	bars := state.GetBars()
	if len(bars) != 0 {
		t.Errorf("Expected 0 bars (wrong symbol ignored), got %d", len(bars))
	}
}

