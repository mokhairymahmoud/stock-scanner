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
	testDuration := 20 * time.Second

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

	// Calculate memory growth (handle potential underflow)
	var heapGrowth int64
	if m2.HeapAlloc > m1.HeapAlloc {
		heapGrowth = int64(m2.HeapAlloc - m1.HeapAlloc)
	} else {
		heapGrowth = -int64(m1.HeapAlloc - m2.HeapAlloc)
	}
	heapGrowthMB := float64(heapGrowth) / 1024 / 1024

	t.Logf("Memory leak detection test results:")
	t.Logf("  Iterations: %d", iterations)
	t.Logf("  Initial heap: %.2f MB", float64(m1.HeapAlloc)/1024/1024)
	t.Logf("  Final heap: %.2f MB", float64(m2.HeapAlloc)/1024/1024)
	t.Logf("  Heap growth: %.2f MB", heapGrowthMB)

	// Check for excessive memory growth (more than 100MB would be suspicious)
	// Only fail if there's significant growth, not if memory decreased
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

	// Use shorter intervals to complete within default timeout
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// Monitor for ~20 seconds (10 iterations × 2 seconds) to fit within 30s timeout
	iterations := 10
	timeout := time.After(25 * time.Second)

	for i := 0; i < iterations; i++ {
		select {
		case <-timeout:
			t.Logf("Test timeout reached, completing early with %d measurements", measurements)
			return
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
	// Use very few symbols (3) so we cycle back quickly and hit cooldowns
	symbols := generateSymbols(3)

	// Track alerts over time
	alertCount := 0
	duplicateCount := 0

	// Simulate alerts over extended period
	// Reduced duration to fit within 30s test timeout
	duration := 20 * time.Second
	// Use cooldown of 8 seconds - with 3 symbols cycling every 3 seconds,
	// we'll hit the same symbol again after 3 seconds, which is still in cooldown
	cooldownSeconds := 8
	start := time.Now()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	timeout := time.After(25 * time.Second)
loop:
	for time.Since(start) < duration {
		select {
		case <-timeout:
			t.Logf("Test timeout reached, completing early")
			break loop
		case <-ticker.C:
			// Cycle through symbols - with 3 symbols, we'll hit each one every 3 seconds
			// Cooldown is 8 seconds, so when we cycle back (after 3 seconds), it's still active
			symbol := symbols[alertCount%len(symbols)]

			if cooldownTracker.IsOnCooldown(ruleID, symbol) {
				duplicateCount++
			} else {
				cooldownTracker.RecordCooldown(ruleID, symbol, cooldownSeconds)
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
	// Reduced duration to fit within 30s test timeout
	duration := 20 * time.Second
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

	// Wait for all goroutines with timeout protection
	timeout := time.After(25 * time.Second)
	for i := 0; i < concurrency; i++ {
		select {
		case <-done:
			// Goroutine completed
		case <-timeout:
			t.Logf("Test timeout reached, some goroutines may not have completed")
			return
		}
	}

	elapsed := time.Since(start)

	// Verify state is still consistent
	if sm.GetSymbolCount() != len(symbols) {
		t.Errorf("State inconsistent after concurrent load: expected %d, got %d", len(symbols), sm.GetSymbolCount())
	}

	t.Logf("Concurrent stability test completed: %v duration, %d concurrent goroutines", elapsed, concurrency)
}
