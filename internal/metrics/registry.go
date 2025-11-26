package metrics

import (
	"fmt"
	"sync"
)

// Registry manages metric computers
type Registry struct {
	mu        sync.RWMutex
	computers map[string]MetricComputer
	ordered   []MetricComputer // Ordered by dependencies
}

// NewRegistry creates a new metric registry and registers built-in metrics
func NewRegistry() *Registry {
	registry := &Registry{
		computers: make(map[string]MetricComputer),
		ordered:   make([]MetricComputer, 0),
	}

	// Register built-in metric computers
	registry.registerBuiltInMetrics()

	return registry
}

// Register registers a metric computer
func (r *Registry) Register(computer MetricComputer) error {
	if computer == nil {
		return fmt.Errorf("computer cannot be nil")
	}

	name := computer.Name()
	if name == "" {
		return fmt.Errorf("computer name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.computers[name]; exists {
		return fmt.Errorf("computer with name %q already registered", name)
	}

	r.computers[name] = computer
	// Rebuild ordered list
	r.rebuildOrdered()

	return nil
}

// ComputeAll computes all registered metrics from a snapshot
func (r *Registry) ComputeAll(snapshot *SymbolStateSnapshot) map[string]float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metrics := make(map[string]float64)

	// First, copy indicators (they're already computed)
	for key, value := range snapshot.Indicators {
		metrics[key] = value
	}

	// Compute metrics in order
	// Note: For now, we use registration order. In the future, we could
	// implement topological sort based on dependencies
	for _, computer := range r.ordered {
		if value, ok := computer.Compute(snapshot); ok {
			metrics[computer.Name()] = value
		}
	}

	return metrics
}

// rebuildOrdered rebuilds the ordered list of computers
// For now, uses registration order. Could be improved with topological sort.
func (r *Registry) rebuildOrdered() {
	r.ordered = make([]MetricComputer, 0, len(r.computers))
	for _, computer := range r.computers {
		r.ordered = append(r.ordered, computer)
	}
}

// registerBuiltInMetrics registers all built-in metric computers
func (r *Registry) registerBuiltInMetrics() {
	// Live bar metrics (no dependencies)
	r.Register(&PriceComputer{})
	r.Register(&VolumeLiveComputer{})
	r.Register(&VWAPLiveComputer{})

	// Finalized bar metrics (no dependencies)
	r.Register(&CloseComputer{})
	r.Register(&OpenComputer{})
	r.Register(&HighComputer{})
	r.Register(&LowComputer{})
	r.Register(&VolumeComputer{})
	r.Register(&VWAPComputer{})

	// Price change metrics (depend on finalized bars)
	r.Register(NewPriceChangeComputer("price_change_1m_pct", 2))
	r.Register(NewPriceChangeComputer("price_change_5m_pct", 6))
	r.Register(NewPriceChangeComputer("price_change_15m_pct", 16))
}

