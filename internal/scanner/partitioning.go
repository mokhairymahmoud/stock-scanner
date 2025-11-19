package scanner

import (
	"crypto/sha256"
	"fmt"
	"hash/fnv"
	"sync"
)

// PartitionManager manages symbol partitioning across multiple workers
type PartitionManager struct {
	workerID      int
	totalWorkers  int
	mu            sync.RWMutex
	assignedSymbols map[string]bool // Symbols assigned to this worker
}

// NewPartitionManager creates a new partition manager
func NewPartitionManager(workerID, totalWorkers int) (*PartitionManager, error) {
	if workerID < 0 {
		return nil, fmt.Errorf("worker ID must be non-negative, got %d", workerID)
	}
	if totalWorkers <= 0 {
		return nil, fmt.Errorf("total workers must be positive, got %d", totalWorkers)
	}
	if workerID >= totalWorkers {
		return nil, fmt.Errorf("worker ID %d must be less than total workers %d", workerID, totalWorkers)
	}

	return &PartitionManager{
		workerID:        workerID,
		totalWorkers:    totalWorkers,
		assignedSymbols: make(map[string]bool),
	}, nil
}

// GetPartition calculates which partition (worker) a symbol belongs to
// Uses consistent hashing: hash(symbol) % totalWorkers
func (pm *PartitionManager) GetPartition(symbol string) int {
	if symbol == "" {
		return 0
	}

	// Use FNV hash for fast, consistent hashing
	h := fnv.New32a()
	h.Write([]byte(symbol))
	hash := h.Sum32()

	partition := int(hash) % pm.totalWorkers
	if partition < 0 {
		partition = -partition
	}

	return partition
}

// IsOwned checks if this worker owns a symbol
func (pm *PartitionManager) IsOwned(symbol string) bool {
	if symbol == "" {
		return false
	}

	partition := pm.GetPartition(symbol)
	return partition == pm.workerID
}

// GetWorkerID returns this worker's ID
func (pm *PartitionManager) GetWorkerID() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.workerID
}

// GetTotalWorkers returns the total number of workers
func (pm *PartitionManager) GetTotalWorkers() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.totalWorkers
}

// AddAssignedSymbol marks a symbol as assigned to this worker
// This is useful for tracking which symbols this worker is responsible for
func (pm *PartitionManager) AddAssignedSymbol(symbol string) {
	if symbol == "" {
		return
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.assignedSymbols[symbol] = true
}

// RemoveAssignedSymbol removes a symbol from assigned symbols
func (pm *PartitionManager) RemoveAssignedSymbol(symbol string) {
	if symbol == "" {
		return
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.assignedSymbols, symbol)
}

// IsAssigned checks if a symbol is assigned to this worker
func (pm *PartitionManager) IsAssigned(symbol string) bool {
	if symbol == "" {
		return false
	}

	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return pm.assignedSymbols[symbol]
}

// GetAssignedSymbols returns a copy of all assigned symbols
func (pm *PartitionManager) GetAssignedSymbols() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	symbols := make([]string, 0, len(pm.assignedSymbols))
	for symbol := range pm.assignedSymbols {
		symbols = append(symbols, symbol)
	}

	return symbols
}

// GetAssignedSymbolCount returns the number of assigned symbols
func (pm *PartitionManager) GetAssignedSymbolCount() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return len(pm.assignedSymbols)
}

// ClearAssignedSymbols clears all assigned symbols
func (pm *PartitionManager) ClearAssignedSymbols() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.assignedSymbols = make(map[string]bool)
}

// UpdateWorkerCount updates the total number of workers
// This can be used for dynamic scaling
func (pm *PartitionManager) UpdateWorkerCount(totalWorkers int) error {
	if totalWorkers <= 0 {
		return fmt.Errorf("total workers must be positive, got %d", totalWorkers)
	}
	if pm.workerID >= totalWorkers {
		return fmt.Errorf("worker ID %d must be less than total workers %d", pm.workerID, totalWorkers)
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.totalWorkers = totalWorkers

	// Recalculate assigned symbols based on new worker count
	// Symbols that are no longer owned by this worker should be removed
	newAssigned := make(map[string]bool)
	for symbol := range pm.assignedSymbols {
		if pm.GetPartition(symbol) == pm.workerID {
			newAssigned[symbol] = true
		}
	}

	pm.assignedSymbols = newAssigned

	return nil
}

// GetPartitionDistribution returns a map of partition -> symbol count
// Useful for monitoring partition balance
func (pm *PartitionManager) GetPartitionDistribution(symbols []string) map[int]int {
	distribution := make(map[int]int)

	for _, symbol := range symbols {
		partition := pm.GetPartition(symbol)
		distribution[partition]++
	}

	return distribution
}

// HashSymbolSHA256 calculates SHA256 hash of a symbol (alternative hashing method)
func HashSymbolSHA256(symbol string) uint32 {
	h := sha256.Sum256([]byte(symbol))
	// Use first 4 bytes for hash
	return uint32(h[0])<<24 | uint32(h[1])<<16 | uint32(h[2])<<8 | uint32(h[3])
}

