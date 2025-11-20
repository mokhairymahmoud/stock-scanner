// Package data contains stability tests for the stock scanner system.
//
// These tests verify long-running stability and resource usage:
// - Long-running stability (configurable duration, default 1 hour)
// - Memory leak detection
// - Resource usage monitoring
// - Alert accuracy over time
// - Concurrent stability
//
// See README.md for documentation on running performance tests.
package data

import (
	"runtime"
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/scanner"
)

// TestStability_LongRunning tests system stability over extended period
// This test should be run with a longer timeout: go test -timeout 30m
func TestStability_LongRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running stability test in short mode")
	}

	// Run for 1 hour (or shorter for CI)
	testDuration := 1 * time.Hour
	if testing.Verbose() {
		// In CI, run for shorter duration
		testDuration = 10 * time.Minute
	}

	sm := scanner.NewStateManager(200)
	symbols := generateSymbols(100)

	// Initialize state
	for _, symbol := range symbols {
		tick := &models.Tick{
			Symbol:    symbol,
			Price:     100.0,
			Size:      100,
			Timestamp: time.Now(),
			Type:      "trade",
		}
		sm.UpdateLiveBar(symbol, tick)
	}

	start := time.Now()
	updateCount := 0
	scanCount := 0
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Run for test duration
	for time.Since(start) < testDuration {
		select {
		case <-ticker.C:
			// Update state
			for _, symbol := range symbols {
				tick := &models.Tick{
					Symbol:    symbol,
					Price:     100.0 + float64(updateCount%100)*0.1,
					Size:      100,
					Timestamp: time.Now(),
					Type:      "trade",
				}
				sm.UpdateLiveBar(symbol, tick)
				updateCount++
			}

			// Simulate scan loop
			sm.Snapshot()
			scanCount++

			// Log progress every minute
			if scanCount%60 == 0 {
				t.Logf("Stability test progress: %v elapsed, %d updates, %d scans", 
					time.Since(start), updateCount, scanCount)
			}
		}
	}

	elapsed := time.Since(start)
	t.Logf("Long-running stability test results:")
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Total updates: %d", updateCount)
	t.Logf("  Total scans: %d", scanCount)
	t.Logf("  Updates per second: %.2f", float64(updateCount)/elapsed.Seconds())
	t.Logf("  Scans per second: %.2f", float64(scanCount)/elapsed.Seconds())

	// Verify state is still consistent
	if sm.GetSymbolCount() != len(symbols) {
		t.Errorf("State lost symbols during long run: expected %d, got %d", len(symbols), sm.GetSymbolCount())
	}
}

// TestStability_MemoryLeakDetection tests for memory leaks
func TestStability_MemoryLeakDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	sm := scanner.NewStateManager(200)
	symbols := generateSymbols(100)

	// Get initial memory stats
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Run many iterations
	iterations := 1000
	for i := 0; i < iterations; i++ {
		// Update state
		for _, symbol := range symbols {
			tick := &models.Tick{
				Symbol:    symbol,
				Price:     100.0 + float64(i)*0.01,
				Size:      100,
				Timestamp: time.Now(),
				Type:      "trade",
			}
			sm.UpdateLiveBar(symbol, tick)
		}

		// Simulate scan
		sm.Snapshot()

		// Force GC periodically
		if i%100 == 0 {
			runtime.GC()
		}
	}

	// Get final memory stats
	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Calculate memory growth
	heapGrowth := m2.HeapAlloc - m1.HeapAlloc
	heapGrowthMB := float64(heapGrowth) / 1024 / 1024

	t.Logf("Memory leak detection test results:")
	t.Logf("  Iterations: %d", iterations)
	t.Logf("  Initial heap: %.2f MB", float64(m1.HeapAlloc)/1024/1024)
	t.Logf("  Final heap: %.2f MB", float64(m2.HeapAlloc)/1024/1024)
	t.Logf("  Heap growth: %.2f MB", heapGrowthMB)

	// Check for excessive memory growth (more than 100MB would be suspicious)
	if heapGrowthMB > 100 {
		t.Errorf("Potential memory leak detected: heap grew by %.2f MB", heapGrowthMB)
	}
}

// TestStability_ResourceUsage monitors resource usage over time
func TestStability_ResourceUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resource usage test in short mode")
	}

	sm := scanner.NewStateManager(200)
	symbols := generateSymbols(1000)

	// Initialize state
	for _, symbol := range symbols {
		tick := &models.Tick{
			Symbol:    symbol,
			Price:     100.0,
			Size:      100,
			Timestamp: time.Now(),
			Type:      "trade",
		}
		sm.UpdateLiveBar(symbol, tick)
	}

	// Monitor resource usage
	var m runtime.MemStats
	start := time.Now()
	measurements := 0

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for i := 0; i < 12; i++ { // Monitor for 1 minute
		select {
		case <-ticker.C:
			runtime.GC()
			runtime.ReadMemStats(&m)

			measurements++
			elapsed := time.Since(start)

			t.Logf("Resource usage (measurement %d, %v elapsed):", measurements, elapsed)
			t.Logf("  Heap allocated: %.2f MB", float64(m.HeapAlloc)/1024/1024)
			t.Logf("  Heap in-use: %.2f MB", float64(m.HeapInuse)/1024/1024)
			t.Logf("  Num GC: %d", m.NumGC)
			t.Logf("  Goroutines: %d", runtime.NumGoroutine())

			// Update state
			for _, symbol := range symbols[:100] { // Update subset
				tick := &models.Tick{
					Symbol:    symbol,
					Price:     100.0 + float64(i)*0.1,
					Size:      100,
					Timestamp: time.Now(),
					Type:      "trade",
				}
				sm.UpdateLiveBar(symbol, tick)
			}

			// Simulate scan
			sm.Snapshot()
		}
	}

	t.Logf("Resource usage monitoring completed: %d measurements", measurements)
}

// TestStability_AlertAccuracy tests alert accuracy over time
func TestStability_AlertAccuracy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping alert accuracy test in short mode")
	}

	cooldownTracker := scanner.NewCooldownTracker(5 * time.Minute)
	ruleID := "rule-test"
	symbols := generateSymbols(100)

	// Track alerts over time
	alertCount := 0
	duplicateCount := 0

	// Simulate alerts over extended period
	duration := 10 * time.Minute
	start := time.Now()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for time.Since(start) < duration {
		select {
		case <-ticker.C:
			// Simulate alert for random symbol
			symbol := symbols[alertCount%len(symbols)]

			if cooldownTracker.IsOnCooldown(ruleID, symbol) {
				duplicateCount++
			} else {
				cooldownTracker.RecordCooldown(ruleID, symbol, 300) // 5 minute cooldown
				alertCount++
			}
		}
	}

	elapsed := time.Since(start)
	duplicateRate := float64(duplicateCount) / float64(alertCount+duplicateCount) * 100

	t.Logf("Alert accuracy test results:")
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Total alerts: %d", alertCount)
	t.Logf("  Duplicates prevented: %d", duplicateCount)
	t.Logf("  Duplicate rate: %.2f%%", duplicateRate)

	// Verify cooldown is working (duplicate rate should be high)
	if duplicateRate < 50 {
		t.Errorf("Cooldown not working effectively: only %.2f%% duplicates prevented", duplicateRate)
	}
}

// TestStability_ConcurrentStability tests stability under concurrent load
func TestStability_ConcurrentStability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent stability test in short mode")
	}

	sm := scanner.NewStateManager(200)
	symbols := generateSymbols(100)

	// Initialize state
	for _, symbol := range symbols {
		tick := &models.Tick{
			Symbol:    symbol,
			Price:     100.0,
			Size:      100,
			Timestamp: time.Now(),
			Type:      "trade",
		}
		sm.UpdateLiveBar(symbol, tick)
	}

	// Run concurrent updates for extended period
	duration := 5 * time.Minute
	concurrency := 10
	start := time.Now()
	done := make(chan bool)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			updateCount := 0
			for time.Since(start) < duration {
				symbol := symbols[(id*10+updateCount)%len(symbols)]
				tick := &models.Tick{
					Symbol:    symbol,
					Price:     100.0 + float64(updateCount)*0.1,
					Size:      100,
					Timestamp: time.Now(),
					Type:      "trade",
				}
				sm.UpdateLiveBar(symbol, tick)
				updateCount++

				// Simulate scan periodically
				if updateCount%100 == 0 {
					sm.Snapshot()
				}

				time.Sleep(10 * time.Millisecond)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < concurrency; i++ {
		<-done
	}

	elapsed := time.Since(start)

	// Verify state is still consistent
	if sm.GetSymbolCount() != len(symbols) {
		t.Errorf("State inconsistent after concurrent load: expected %d, got %d", len(symbols), sm.GetSymbolCount())
	}

	t.Logf("Concurrent stability test completed: %v duration, %d concurrent goroutines", elapsed, concurrency)
}

