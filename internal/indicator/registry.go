package indicator

import (
	"fmt"
	"sync"
)

// IndicatorRegistry manages all available indicators (Techan + Custom)
type IndicatorRegistry struct {
	mu        sync.RWMutex
	factories map[string]CalculatorFactory
	metadata  map[string]IndicatorMetadata
}

// IndicatorMetadata contains information about an indicator
type IndicatorMetadata struct {
	Name        string
	Type        string // "techan", "custom"
	Description string
	Parameters  map[string]interface{}
	Category    string // "momentum", "trend", "volatility", "volume", "price"
}

// NewIndicatorRegistry creates a new indicator registry
func NewIndicatorRegistry() *IndicatorRegistry {
	return &IndicatorRegistry{
		factories: make(map[string]CalculatorFactory),
		metadata:  make(map[string]IndicatorMetadata),
	}
}

// Register registers an indicator factory
func (r *IndicatorRegistry) Register(
	name string,
	factory CalculatorFactory,
	metadata IndicatorMetadata,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[name]; exists {
		return fmt.Errorf("indicator %q already registered", name)
	}

	r.factories[name] = factory
	r.metadata[name] = metadata
	return nil
}

// GetFactory returns a factory for an indicator
func (r *IndicatorRegistry) GetFactory(name string) (CalculatorFactory, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	factory, exists := r.factories[name]
	return factory, exists
}

// ListAvailable returns all available indicator names
func (r *IndicatorRegistry) ListAvailable() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// GetMetadata returns metadata for an indicator
func (r *IndicatorRegistry) GetMetadata(name string) (IndicatorMetadata, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	metadata, exists := r.metadata[name]
	return metadata, exists
}

// GetAllMetadata returns all indicator metadata
func (r *IndicatorRegistry) GetAllMetadata() map[string]IndicatorMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]IndicatorMetadata, len(r.metadata))
	for name, metadata := range r.metadata {
		result[name] = metadata
	}
	return result
}

