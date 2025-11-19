package indicator

import (
	"fmt"
	"sync"
)

// Registry manages indicator calculators
type Registry struct {
	mu         sync.RWMutex
	calculators map[string]Calculator
}

// NewRegistry creates a new indicator registry
func NewRegistry() *Registry {
	return &Registry{
		calculators: make(map[string]Calculator),
	}
}

// Register registers a calculator with the registry
func (r *Registry) Register(calc Calculator) error {
	if calc == nil {
		return fmt.Errorf("calculator cannot be nil")
	}

	name := calc.Name()
	if name == "" {
		return fmt.Errorf("calculator name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.calculators[name]; exists {
		return fmt.Errorf("calculator with name %q already registered", name)
	}

	r.calculators[name] = calc
	return nil
}

// Get retrieves a calculator by name
func (r *Registry) Get(name string) (Calculator, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	calc, exists := r.calculators[name]
	if !exists {
		return nil, fmt.Errorf("calculator %q not found", name)
	}

	return calc, nil
}

// GetAll returns all registered calculators
func (r *Registry) GetAll() map[string]Calculator {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]Calculator, len(r.calculators))
	for name, calc := range r.calculators {
		result[name] = calc
	}

	return result
}

// List returns a list of all registered calculator names
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.calculators))
	for name := range r.calculators {
		names = append(names, name)
	}

	return names
}

// Unregister removes a calculator from the registry
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.calculators[name]; !exists {
		return fmt.Errorf("calculator %q not found", name)
	}

	delete(r.calculators, name)
	return nil
}

// Clear removes all calculators from the registry
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.calculators = make(map[string]Calculator)
}

