package indicator

import (
	"testing"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

// mockCalculator is a simple mock calculator for testing
type mockCalculator struct {
	name      string
	value     float64
	ready     bool
	processed int
}

func (m *mockCalculator) Name() string {
	return m.name
}

func (m *mockCalculator) Update(bar *models.Bar1m) (float64, error) {
	m.processed++
	m.value = float64(m.processed)
	if m.processed >= 2 {
		m.ready = true
	}
	return m.value, nil
}

func (m *mockCalculator) Value() (float64, error) {
	if !m.ready {
		return 0, nil
	}
	return m.value, nil
}

func (m *mockCalculator) Reset() {
	m.processed = 0
	m.value = 0
	m.ready = false
}

func (m *mockCalculator) IsReady() bool {
	return m.ready
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	calc1 := &mockCalculator{name: "test1"}
	calc2 := &mockCalculator{name: "test2"}

	// Register first calculator
	err := registry.Register(calc1)
	if err != nil {
		t.Fatalf("Failed to register calculator: %v", err)
	}

	// Register second calculator
	err = registry.Register(calc2)
	if err != nil {
		t.Fatalf("Failed to register second calculator: %v", err)
	}

	// Try to register duplicate
	err = registry.Register(calc1)
	if err == nil {
		t.Error("Expected error when registering duplicate calculator")
	}

	// Try to register nil
	err = registry.Register(nil)
	if err == nil {
		t.Error("Expected error when registering nil calculator")
	}
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()

	calc := &mockCalculator{name: "test"}
	_ = registry.Register(calc)

	// Get existing calculator
	retrieved, err := registry.Get("test")
	if err != nil {
		t.Fatalf("Failed to get calculator: %v", err)
	}
	if retrieved != calc {
		t.Error("Retrieved calculator is not the same instance")
	}

	// Get non-existent calculator
	_, err = registry.Get("nonexistent")
	if err == nil {
		t.Error("Expected error when getting non-existent calculator")
	}
}

func TestRegistry_GetAll(t *testing.T) {
	registry := NewRegistry()

	calc1 := &mockCalculator{name: "test1"}
	calc2 := &mockCalculator{name: "test2"}

	_ = registry.Register(calc1)
	_ = registry.Register(calc2)

	all := registry.GetAll()
	if len(all) != 2 {
		t.Errorf("Expected 2 calculators, got %d", len(all))
	}

	if all["test1"] != calc1 {
		t.Error("test1 calculator not found")
	}
	if all["test2"] != calc2 {
		t.Error("test2 calculator not found")
	}
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()

	calc1 := &mockCalculator{name: "test1"}
	calc2 := &mockCalculator{name: "test2"}

	_ = registry.Register(calc1)
	_ = registry.Register(calc2)

	names := registry.List()
	if len(names) != 2 {
		t.Errorf("Expected 2 calculator names, got %d", len(names))
	}

	nameMap := make(map[string]bool)
	for _, name := range names {
		nameMap[name] = true
	}

	if !nameMap["test1"] {
		t.Error("test1 not in list")
	}
	if !nameMap["test2"] {
		t.Error("test2 not in list")
	}
}

func TestRegistry_Unregister(t *testing.T) {
	registry := NewRegistry()

	calc := &mockCalculator{name: "test"}
	_ = registry.Register(calc)

	// Unregister existing calculator
	err := registry.Unregister("test")
	if err != nil {
		t.Fatalf("Failed to unregister calculator: %v", err)
	}

	// Verify it's gone
	_, err = registry.Get("test")
	if err == nil {
		t.Error("Expected error when getting unregistered calculator")
	}

	// Try to unregister non-existent calculator
	err = registry.Unregister("nonexistent")
	if err == nil {
		t.Error("Expected error when unregistering non-existent calculator")
	}
}

func TestRegistry_Clear(t *testing.T) {
	registry := NewRegistry()

	calc1 := &mockCalculator{name: "test1"}
	calc2 := &mockCalculator{name: "test2"}

	_ = registry.Register(calc1)
	_ = registry.Register(calc2)

	registry.Clear()

	if len(registry.GetAll()) != 0 {
		t.Error("Expected registry to be empty after Clear()")
	}
}

