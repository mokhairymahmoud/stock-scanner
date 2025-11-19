package scanner

import (
	"fmt"
	"testing"
)

func TestNewPartitionManager(t *testing.T) {
	// Test normal creation
	pm, err := NewPartitionManager(0, 4)
	if err != nil {
		t.Fatalf("Failed to create partition manager: %v", err)
	}

	if pm == nil {
		t.Fatal("Expected partition manager to be created")
	}

	if pm.workerID != 0 {
		t.Errorf("Expected worker ID 0, got %d", pm.workerID)
	}

	if pm.totalWorkers != 4 {
		t.Errorf("Expected total workers 4, got %d", pm.totalWorkers)
	}

	// Test invalid worker ID
	_, err = NewPartitionManager(-1, 4)
	if err == nil {
		t.Error("Expected error for negative worker ID")
	}

	// Test invalid total workers
	_, err = NewPartitionManager(0, 0)
	if err == nil {
		t.Error("Expected error for zero total workers")
	}

	// Test worker ID >= total workers
	_, err = NewPartitionManager(4, 4)
	if err == nil {
		t.Error("Expected error when worker ID >= total workers")
	}
}

func TestPartitionManager_GetPartition(t *testing.T) {
	pm, err := NewPartitionManager(0, 4)
	if err != nil {
		t.Fatalf("Failed to create partition manager: %v", err)
	}

	// Test consistent hashing
	symbols := []string{"AAPL", "GOOGL", "MSFT", "TSLA", "AMZN"}
	partitions := make(map[int]bool)

	for _, symbol := range symbols {
		partition := pm.GetPartition(symbol)
		if partition < 0 || partition >= 4 {
			t.Errorf("Partition %d for symbol %s is out of range [0, 4)", partition, symbol)
		}
		partitions[partition] = true
	}

	// Verify partitions are in valid range (distribution may vary, that's OK)
	// Note: It's possible all symbols hash to the same partition, which is valid

	// Test consistency: same symbol should always get same partition
	for i := 0; i < 10; i++ {
		partition1 := pm.GetPartition("AAPL")
		partition2 := pm.GetPartition("AAPL")
		if partition1 != partition2 {
			t.Errorf("Partition should be consistent, got %d and %d", partition1, partition2)
		}
	}

	// Test empty symbol
	partition := pm.GetPartition("")
	if partition != 0 {
		t.Errorf("Expected partition 0 for empty symbol, got %d", partition)
	}
}

func TestPartitionManager_IsOwned(t *testing.T) {
	pm, err := NewPartitionManager(1, 4)
	if err != nil {
		t.Fatalf("Failed to create partition manager: %v", err)
	}

	// Test symbols that belong to this worker
	symbols := []string{"AAPL", "GOOGL", "MSFT", "TSLA", "AMZN"}
	for _, symbol := range symbols {
		partition := pm.GetPartition(symbol)
		isOwned := pm.IsOwned(symbol)
		if isOwned != (partition == 1) {
			t.Errorf("IsOwned(%s) = %v, but partition is %d (expected %v)", symbol, isOwned, partition, partition == 1)
		}
	}

	// Test empty symbol
	if pm.IsOwned("") {
		t.Error("Expected empty symbol not to be owned")
	}
}

func TestPartitionManager_AssignedSymbols(t *testing.T) {
	pm, err := NewPartitionManager(0, 4)
	if err != nil {
		t.Fatalf("Failed to create partition manager: %v", err)
	}

	// Initially no assigned symbols
	if pm.GetAssignedSymbolCount() != 0 {
		t.Errorf("Expected 0 assigned symbols, got %d", pm.GetAssignedSymbolCount())
	}

	// Add symbols
	symbols := []string{"AAPL", "GOOGL", "MSFT"}
	for _, symbol := range symbols {
		pm.AddAssignedSymbol(symbol)
	}

	if pm.GetAssignedSymbolCount() != len(symbols) {
		t.Errorf("Expected %d assigned symbols, got %d", len(symbols), pm.GetAssignedSymbolCount())
	}

	// Verify all are assigned
	for _, symbol := range symbols {
		if !pm.IsAssigned(symbol) {
			t.Errorf("Expected symbol %s to be assigned", symbol)
		}
	}

	// Get assigned symbols
	assigned := pm.GetAssignedSymbols()
	if len(assigned) != len(symbols) {
		t.Errorf("Expected %d assigned symbols, got %d", len(symbols), len(assigned))
	}

	// Remove a symbol
	pm.RemoveAssignedSymbol("AAPL")
	if pm.IsAssigned("AAPL") {
		t.Error("Expected AAPL not to be assigned after removal")
	}

	if pm.GetAssignedSymbolCount() != len(symbols)-1 {
		t.Errorf("Expected %d assigned symbols, got %d", len(symbols)-1, pm.GetAssignedSymbolCount())
	}

	// Clear all
	pm.ClearAssignedSymbols()
	if pm.GetAssignedSymbolCount() != 0 {
		t.Errorf("Expected 0 assigned symbols after clear, got %d", pm.GetAssignedSymbolCount())
	}
}

func TestPartitionManager_UpdateWorkerCount(t *testing.T) {
	pm, err := NewPartitionManager(1, 4)
	if err != nil {
		t.Fatalf("Failed to create partition manager: %v", err)
	}

	// Add some symbols
	symbols := []string{"AAPL", "GOOGL", "MSFT", "TSLA", "AMZN"}
	for _, symbol := range symbols {
		if pm.IsOwned(symbol) {
			pm.AddAssignedSymbol(symbol)
		}
	}

	// Update to 8 workers
	err = pm.UpdateWorkerCount(8)
	if err != nil {
		t.Fatalf("Failed to update worker count: %v", err)
	}

	if pm.GetTotalWorkers() != 8 {
		t.Errorf("Expected total workers 8, got %d", pm.GetTotalWorkers())
	}

	// Symbols should be recalculated
	// Some symbols that were owned may no longer be owned
	for _, symbol := range symbols {
		if pm.IsAssigned(symbol) && !pm.IsOwned(symbol) {
			t.Errorf("Symbol %s is assigned but not owned after worker count update", symbol)
		}
	}

	// Test invalid worker count
	err = pm.UpdateWorkerCount(0)
	if err == nil {
		t.Error("Expected error for zero worker count")
	}

	// Test worker ID >= total workers
	err = pm.UpdateWorkerCount(1)
	if err == nil {
		t.Error("Expected error when worker ID >= total workers")
	}
}

func TestPartitionManager_GetPartitionDistribution(t *testing.T) {
	pm, err := NewPartitionManager(0, 4)
	if err != nil {
		t.Fatalf("Failed to create partition manager: %v", err)
	}

	symbols := []string{"AAPL", "GOOGL", "MSFT", "TSLA", "AMZN", "NVDA", "META", "NFLX"}
	distribution := pm.GetPartitionDistribution(symbols)

	// Verify all symbols are distributed
	total := 0
	for _, count := range distribution {
		total += count
	}

	if total != len(symbols) {
		t.Errorf("Expected %d symbols in distribution, got %d", len(symbols), total)
	}

	// Verify partitions are in valid range
	for partition := range distribution {
		if partition < 0 || partition >= 4 {
			t.Errorf("Invalid partition %d in distribution", partition)
		}
	}
}

func TestPartitionManager_Concurrency(t *testing.T) {
	pm, err := NewPartitionManager(0, 4)
	if err != nil {
		t.Fatalf("Failed to create partition manager: %v", err)
	}

	// Test concurrent access
	done := make(chan bool)
	symbols := []string{"AAPL", "GOOGL", "MSFT", "TSLA", "AMZN"}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			for _, symbol := range symbols {
				pm.GetPartition(symbol)
				pm.IsOwned(symbol)
				pm.GetTotalWorkers()
				pm.GetWorkerID()
			}
			done <- true
		}()
	}

	// Concurrent writes
	for i := 0; i < 5; i++ {
		go func(idx int) {
			symbol := symbols[idx%len(symbols)]
			pm.AddAssignedSymbol(symbol)
			pm.IsAssigned(symbol)
			pm.RemoveAssignedSymbol(symbol)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 15; i++ {
		<-done
	}
}

func TestPartitionManager_ConsistentHashing(t *testing.T) {
	// Test that hashing is consistent across different partition managers
	pm1, _ := NewPartitionManager(0, 4)
	pm2, _ := NewPartitionManager(0, 4)

	symbols := []string{"AAPL", "GOOGL", "MSFT", "TSLA", "AMZN"}

	for _, symbol := range symbols {
		partition1 := pm1.GetPartition(symbol)
		partition2 := pm2.GetPartition(symbol)

		if partition1 != partition2 {
			t.Errorf("Partition should be consistent for symbol %s, got %d and %d", symbol, partition1, partition2)
		}
	}
}

func TestPartitionManager_EmptySymbol(t *testing.T) {
	pm, err := NewPartitionManager(0, 4)
	if err != nil {
		t.Fatalf("Failed to create partition manager: %v", err)
	}

	// Empty symbol should not be assigned
	pm.AddAssignedSymbol("")
	if pm.GetAssignedSymbolCount() != 0 {
		t.Error("Expected empty symbol not to be assigned")
	}

	// Empty symbol should not be owned
	if pm.IsOwned("") {
		t.Error("Expected empty symbol not to be owned")
	}

	// Empty symbol should return partition 0
	partition := pm.GetPartition("")
	if partition != 0 {
		t.Errorf("Expected partition 0 for empty symbol, got %d", partition)
	}
}

func TestHashSymbolSHA256(t *testing.T) {
	// Test that SHA256 hash is consistent
	hash1 := HashSymbolSHA256("AAPL")
	hash2 := HashSymbolSHA256("AAPL")

	if hash1 != hash2 {
		t.Errorf("SHA256 hash should be consistent, got %d and %d", hash1, hash2)
	}

	// Test different symbols produce different hashes
	hash3 := HashSymbolSHA256("GOOGL")
	if hash1 == hash3 {
		t.Error("Different symbols should produce different hashes")
	}
}

func TestPartitionManager_Balance(t *testing.T) {
	// Test partition balance with many symbols
	pm, err := NewPartitionManager(0, 4)
	if err != nil {
		t.Fatalf("Failed to create partition manager: %v", err)
	}

	// Generate many symbols
	symbols := make([]string, 100)
	for i := 0; i < 100; i++ {
		symbols[i] = fmt.Sprintf("SYMBOL%d", i)
	}

	distribution := pm.GetPartitionDistribution(symbols)

	// Check balance (should be roughly equal)
	minCount := 100
	maxCount := 0
	for _, count := range distribution {
		if count < minCount {
			minCount = count
		}
		if count > maxCount {
			maxCount = count
		}
	}

	// Allow some variance (at least 15 symbols per partition, at most 35)
	if minCount < 15 {
		t.Errorf("Partition balance too poor: min count %d", minCount)
	}
	if maxCount > 35 {
		t.Errorf("Partition balance too poor: max count %d", maxCount)
	}
}

