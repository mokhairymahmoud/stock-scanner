package scanner

import (
	"time"
)

// invalidateMetricCache invalidates the metric cache
// Should be called whenever state data changes
func (s *SymbolState) invalidateMetricCache() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear cache
	s.cachedMetrics = make(map[string]float64)
	s.cacheTimestamp = time.Time{}
	s.cacheInvalidation = time.Time{}
}

// getCachedMetrics retrieves cached metrics if cache is still valid
// Returns nil if cache is invalid or empty
func (s *SymbolState) getCachedMetrics(requiredMetrics map[string]bool, maxAge time.Duration) map[string]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// If no cache or cache is too old, return nil
	if len(s.cachedMetrics) == 0 || s.cacheTimestamp.IsZero() {
		return nil
	}

	// Check if cache is still valid
	if !s.cacheInvalidation.IsZero() && time.Now().After(s.cacheInvalidation) {
		return nil
	}

	// Check if cache age exceeds maxAge
	if maxAge > 0 && time.Since(s.cacheTimestamp) > maxAge {
		return nil
	}

	// Check if all required metrics are in cache
	if requiredMetrics != nil && len(requiredMetrics) > 0 {
		for metric := range requiredMetrics {
			if _, exists := s.cachedMetrics[metric]; !exists {
				// Missing required metric, cache is incomplete
				return nil
			}
		}
	}

	// Cache is valid, return a copy
	result := make(map[string]float64, len(s.cachedMetrics))
	for k, v := range s.cachedMetrics {
		// Only include requested metrics if specified
		if requiredMetrics == nil || requiredMetrics[k] {
			result[k] = v
		}
	}

	return result
}

// setCachedMetrics stores computed metrics in cache
func (s *SymbolState) setCachedMetrics(metrics map[string]float64, maxAge time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Store metrics
	s.cachedMetrics = make(map[string]float64, len(metrics))
	for k, v := range metrics {
		s.cachedMetrics[k] = v
	}

	s.cacheTimestamp = time.Now()
	if maxAge > 0 {
		s.cacheInvalidation = s.cacheTimestamp.Add(maxAge)
	}
}

