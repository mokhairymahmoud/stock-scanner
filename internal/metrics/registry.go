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
	r.Register(NewPriceChangeComputer("price_change_2m_pct", 3))
	r.Register(NewPriceChangeComputer("price_change_5m_pct", 6))
	r.Register(NewPriceChangeComputer("price_change_15m_pct", 16))
	r.Register(NewPriceChangeComputer("price_change_30m_pct", 31))
	r.Register(NewPriceChangeComputer("price_change_60m_pct", 61))

	// Price filters - Change ($) with timeframes
	r.Register(NewChangeComputer("change_1m", 2))
	r.Register(NewChangeComputer("change_2m", 3))
	r.Register(NewChangeComputer("change_5m", 6))
	r.Register(NewChangeComputer("change_15m", 16))
	r.Register(NewChangeComputer("change_30m", 31))
	r.Register(NewChangeComputer("change_60m", 61))

	// Price filters - Change from Close
	r.Register(&ChangeFromCloseComputer{})
	r.Register(&ChangeFromClosePctComputer{})

	// Price filters - Change from Close (Premarket)
	r.Register(&ChangeFromClosePremarketComputer{})
	r.Register(&ChangeFromClosePremarketPctComputer{})

	// Price filters - Change from Close (Post Market)
	r.Register(&ChangeFromClosePostmarketComputer{})
	r.Register(&ChangeFromClosePostmarketPctComputer{})

	// Price filters - Change from Open
	r.Register(&ChangeFromOpenComputer{})
	r.Register(&ChangeFromOpenPctComputer{})

	// Price filters - Gap from Close
	r.Register(&GapFromCloseComputer{})
	r.Register(&GapFromClosePctComputer{})

	// Volume filters - Session-specific
	r.Register(&PostmarketVolumeComputer{})
	r.Register(&PremarketVolumeComputer{})

	// Volume filters - Absolute Volume with timeframes
	r.Register(NewAbsoluteVolumeComputer("volume_1m", 1))
	r.Register(NewAbsoluteVolumeComputer("volume_2m", 2))
	r.Register(NewAbsoluteVolumeComputer("volume_5m", 5))
	r.Register(NewAbsoluteVolumeComputer("volume_10m", 10))
	r.Register(NewAbsoluteVolumeComputer("volume_15m", 15))
	r.Register(NewAbsoluteVolumeComputer("volume_30m", 30))
	r.Register(NewAbsoluteVolumeComputer("volume_60m", 60))
	r.Register(&DailyVolumeComputer{})

	// Volume filters - Dollar Volume with timeframes
	r.Register(NewDollarVolumeComputer("dollar_volume_1m", 1))
	r.Register(NewDollarVolumeComputer("dollar_volume_5m", 5))
	r.Register(NewDollarVolumeComputer("dollar_volume_15m", 15))
	r.Register(NewDollarVolumeComputer("dollar_volume_60m", 60))
	r.Register(&DailyDollarVolumeComputer{})
}

