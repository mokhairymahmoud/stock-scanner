// Package data contains worker scaling performance tests for the stock scanner system.
//
// These tests measure performance across different worker counts (1, 2, 3, 4 workers):
// - Scan cycle time per worker
// - Total throughput (symbols scanned/second across all workers)
// - Load distribution across workers
// - Scaling efficiency (linear vs sub-linear scaling)
//
// See README.md for documentation on running performance tests.
package data

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/rules"
	"github.com/mohamedkhairy/stock-scanner/internal/scanner"
)

// WorkerPerformanceMetrics holds performance metrics for a single worker
type WorkerPerformanceMetrics struct {
	WorkerID         int
	SymbolCount      int
	ScanDuration     time.Duration
	SymbolsPerSecond float64
	RulesEvaluated   int
	RulesMatched     int
	AlertsEmitted    int
}

// ScalingTestResults holds results for a scaling test
type ScalingTestResults struct {
	WorkerCount       int
	TotalSymbols      int
	Workers           []WorkerPerformanceMetrics
	TotalScanTime     time.Duration // Time for all workers to complete
	TotalThroughput   float64       // Total symbols/second across all workers
	AverageScanTime   time.Duration
	ScalingEfficiency float64 // Percentage of linear scaling (1.0 = perfect linear)
}

// TestWorkerScaling_1To4Workers tests performance with 1, 2, 3, and 4 workers
// This test should be run with a longer timeout for large symbol counts:
//
//	go test -timeout 10m ./tests/performance -run TestWorkerScaling_1To4Workers
//
// For 1M symbols, expect ~5-10 minutes depending on hardware
//
// Note: Go uses multiple CPU cores (GOMAXPROCS), so workers can run in parallel.
// However, scaling may not be perfectly linear, and per-worker scan time may
// actually INCREASE with more workers due to:
//   - GC pressure: StateManager.Snapshot() does heavy memory allocation (deep copies
//     of all symbol states). When multiple workers snapshot simultaneously, they all
//     allocate huge amounts of memory at once, causing frequent and longer GC pauses
//     that affect ALL workers.
//   - Memory bandwidth saturation: All workers reading/writing memory simultaneously
//     saturates the memory bus, making each worker slower.
//   - Cache thrashing: Multiple workers accessing different memory regions causes
//     cache line conflicts and reduced cache efficiency.
//
// This is a REAL bottleneck that would occur in production if multiple workers
// run on the same machine. The solution is to run workers on separate machines
// (distributed setup) where each has its own memory bandwidth and GC.
//
// The test measures per-worker throughput efficiency, which should remain above
// 50% of baseline. In a true distributed setup, scaling would be much better.
func TestWorkerScaling_1To4Workers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping worker scaling test in short mode")
	}

	symbolCount := 1000000
	ruleCount := 10
	workerCounts := []int{1, 2, 3, 4}

	// Generate symbols
	symbols := generateSymbols(symbolCount)

	// Generate rules
	testRules := generateRules(ruleCount)

	// Run tests for each worker count
	results := make([]ScalingTestResults, len(workerCounts))
	for i, workerCount := range workerCounts {
		t.Logf("\n=== Testing with %d worker(s) ===", workerCount)
		result := testWorkerCount(t, workerCount, symbols, testRules)
		results[i] = result
		printScalingResults(t, result)
	}

	// Compare results
	t.Logf("\n=== Scaling Comparison ===")
	printScalingComparison(t, results)

	// Verify scaling efficiency
	verifyScalingEfficiency(t, results)
}

// testWorkerCount tests performance with a specific number of workers
func testWorkerCount(t *testing.T, workerCount int, symbols []string, testRules []*models.Rule) ScalingTestResults {
	// Create partition managers for each worker
	partitionManagers := make([]*scanner.PartitionManager, workerCount)
	for i := 0; i < workerCount; i++ {
		pm, err := scanner.NewPartitionManager(i, workerCount)
		if err != nil {
			t.Fatalf("Failed to create partition manager for worker %d: %v", i, err)
		}
		partitionManagers[i] = pm
	}

	// Distribute symbols to workers
	workerSymbols := make([][]string, workerCount)
	for _, symbol := range symbols {
		for workerID, pm := range partitionManagers {
			if pm.IsOwned(symbol) {
				workerSymbols[workerID] = append(workerSymbols[workerID], symbol)
				break
			}
		}
	}

	// Log distribution
	totalDistributed := 0
	for i, syms := range workerSymbols {
		pct := float64(len(syms)) / float64(len(symbols)) * 100
		t.Logf("Worker %d: %d symbols (%.1f%%)", i, len(syms), pct)
		totalDistributed += len(syms)
	}
	if totalDistributed != len(symbols) {
		t.Errorf("Symbol distribution mismatch: expected %d, got %d", len(symbols), totalDistributed)
	}

	// Create state managers and scan loops for each worker
	workers := make([]workerSetup, workerCount)
	for i := 0; i < workerCount; i++ {
		workers[i] = setupWorker(t, i, workerSymbols[i], testRules)
	}

	// Run scans concurrently (simulating parallel workers)
	var wg sync.WaitGroup
	metrics := make([]WorkerPerformanceMetrics, workerCount)
	startTime := time.Now()

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			metrics[workerID] = runWorkerScan(t, workerID, workers[workerID])
		}(i)
	}

	wg.Wait()
	totalTime := time.Since(startTime)

	// Calculate results
	result := ScalingTestResults{
		WorkerCount:   workerCount,
		TotalSymbols:  len(symbols),
		Workers:       metrics,
		TotalScanTime: totalTime,
	}

	// Calculate total throughput
	totalSymbolsScanned := 0
	var totalScanDuration time.Duration
	for _, m := range metrics {
		totalSymbolsScanned += m.SymbolCount
		totalScanDuration += m.ScanDuration
	}
	result.TotalThroughput = float64(totalSymbolsScanned) / totalTime.Seconds()
	if workerCount > 0 {
		result.AverageScanTime = totalScanDuration / time.Duration(workerCount)
	}

	return result
}

// workerSetup holds the setup for a single worker
type workerSetup struct {
	StateManager     *scanner.StateManager
	ScanLoop         *scanner.ScanLoop
	PartitionManager *scanner.PartitionManager
	Symbols          []string
}

// setupWorker creates a worker with state manager and scan loop
func setupWorker(t *testing.T, workerID int, symbols []string, testRules []*models.Rule) workerSetup {
	sm := scanner.NewStateManager(200)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := newMockCooldownTracker()
	alertEmitter := newMockAlertEmitter()
	toplistIntegration := scanner.NewToplistIntegration(nil, false, 1*time.Second) // Disabled

	// Add rules
	for _, rule := range testRules {
		ruleStore.AddRule(rule)
	}

	// Initialize state for this worker's symbols
	for _, symbol := range symbols {
		// Add live bar
		tick := &models.Tick{
			Symbol:    symbol,
			Price:     100.0 + float64(workerID)*0.1,
			Size:      100,
			Timestamp: time.Now(),
			Type:      "trade",
		}
		sm.UpdateLiveBar(symbol, tick)

		// Add finalized bars
		for j := 0; j < 10; j++ {
			bar := &models.Bar1m{
				Symbol:    symbol,
				Timestamp: time.Now().Add(-time.Duration(10-j) * time.Minute),
				Open:      100.0,
				High:      101.0,
				Low:       99.0,
				Close:     100.5,
				Volume:    1000,
				VWAP:      100.3,
			}
			sm.UpdateFinalizedBar(bar)
		}

		// Add indicators
		indicators := map[string]float64{
			"rsi_14":  50.0 + float64(workerID%30),
			"ema_20":  100.0 + float64(workerID%10),
			"sma_50":  100.0 + float64(workerID%10),
			"vwap_5m": 100.0 + float64(workerID%5),
		}
		sm.UpdateIndicators(symbol, indicators)
	}

	config := scanner.DefaultScanLoopConfig()
	sl := scanner.NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, alertEmitter, toplistIntegration)
	sl.ReloadRules()

	pm, _ := scanner.NewPartitionManager(workerID, 4) // Max workers for partitioning

	return workerSetup{
		StateManager:     sm,
		ScanLoop:         sl,
		PartitionManager: pm,
		Symbols:          symbols,
	}
}

// runWorkerScan runs a scan for a single worker and returns metrics
func runWorkerScan(t *testing.T, workerID int, setup workerSetup) WorkerPerformanceMetrics {
	start := time.Now()
	setup.ScanLoop.Scan()
	scanDuration := time.Since(start)

	stats := setup.ScanLoop.GetStats()
	symbolCount := setup.StateManager.GetSymbolCount()

	return WorkerPerformanceMetrics{
		WorkerID:         workerID,
		SymbolCount:      symbolCount,
		ScanDuration:     scanDuration,
		SymbolsPerSecond: float64(symbolCount) / scanDuration.Seconds(),
		RulesEvaluated:   int(stats.RulesEvaluated),
		RulesMatched:     int(stats.RulesMatched),
		AlertsEmitted:    int(stats.AlertsEmitted),
	}
}

// printScalingResults prints results for a single worker count
func printScalingResults(t *testing.T, result ScalingTestResults) {
	t.Logf("Worker Count: %d", result.WorkerCount)
	t.Logf("Total Symbols: %d", result.TotalSymbols)
	t.Logf("Total Scan Time: %v", result.TotalScanTime)
	t.Logf("Total Throughput: %.2f symbols/second", result.TotalThroughput)
	t.Logf("Average Scan Time: %v", result.AverageScanTime)

	t.Logf("\nPer-Worker Metrics:")
	for _, worker := range result.Workers {
		t.Logf("  Worker %d:", worker.WorkerID)
		t.Logf("    Symbols: %d", worker.SymbolCount)
		t.Logf("    Scan Time: %v", worker.ScanDuration)
		t.Logf("    Throughput: %.2f symbols/second", worker.SymbolsPerSecond)
		t.Logf("    Rules Evaluated: %d", worker.RulesEvaluated)
		t.Logf("    Rules Matched: %d", worker.RulesMatched)
		t.Logf("    Alerts Emitted: %d", worker.AlertsEmitted)
	}
}

// printScalingComparison prints comparison across worker counts
func printScalingComparison(t *testing.T, results []ScalingTestResults) {
	t.Logf("Worker Count | Total Throughput | Avg Scan Time | Per-Worker Throughput | Efficiency")
	t.Logf("-------------|------------------|---------------|------------------------|-----------")

	baselineThroughput := results[0].TotalThroughput
	baselinePerWorker := baselineThroughput / float64(results[0].WorkerCount)

	for _, result := range results {
		// Calculate average per-worker throughput
		var totalPerWorker float64
		for _, worker := range result.Workers {
			totalPerWorker += worker.SymbolsPerSecond
		}
		avgPerWorker := totalPerWorker / float64(len(result.Workers))
		efficiency := (avgPerWorker / baselinePerWorker) * 100

		t.Logf("     %d       |   %8.2f/s     |    %8v    |      %8.2f/s      |   %.1f%%",
			result.WorkerCount, result.TotalThroughput, result.AverageScanTime, avgPerWorker, efficiency)
	}
}

// verifyScalingEfficiency verifies that scaling is reasonable
// Note: Go uses multiple CPU cores, so workers can run in parallel. However,
// scaling may not be perfectly linear due to memory bandwidth, GC pressure,
// or cache contention. We measure per-worker throughput to account for these factors.
func verifyScalingEfficiency(t *testing.T, results []ScalingTestResults) {
	if len(results) < 2 {
		return
	}

	baseline := results[0]
	baselinePerWorkerThroughput := baseline.TotalThroughput / float64(baseline.WorkerCount)

	for i := 1; i < len(results); i++ {
		result := results[i]

		// Calculate average per-worker throughput for this configuration
		var totalPerWorkerThroughput float64
		for _, worker := range result.Workers {
			totalPerWorkerThroughput += worker.SymbolsPerSecond
		}
		avgPerWorkerThroughput := totalPerWorkerThroughput / float64(len(result.Workers))

		// Calculate efficiency based on per-worker throughput (should stay relatively constant)
		// In ideal distributed scenario, per-worker throughput should be similar
		// On single machine, we allow some degradation (at least 50% of baseline per-worker)
		efficiency := (avgPerWorkerThroughput / baselinePerWorkerThroughput) * 100

		t.Logf("Scaling from 1 to %d workers:", result.WorkerCount)
		t.Logf("  Baseline per-worker throughput: %.2f symbols/s", baselinePerWorkerThroughput)
		t.Logf("  Average per-worker throughput: %.2f symbols/s", avgPerWorkerThroughput)
		t.Logf("  Per-worker efficiency: %.1f%%", efficiency)
		t.Logf("  Total throughput: %.2f symbols/s (vs expected %.2f for linear scaling)",
			result.TotalThroughput, baseline.TotalThroughput*float64(result.WorkerCount))

		// Note: Per-worker scan time may INCREASE with more workers due to:
		// - GC pauses: StateManager.Snapshot() does heavy memory allocation (deep copies).
		//   When multiple workers snapshot simultaneously, they all allocate huge amounts
		//   of memory, causing frequent GC pauses that pause ALL goroutines (even across
		//   CPU cores), making each worker slower.
		// - Memory bandwidth saturation: All workers allocating/accessing memory simultaneously
		// - This is a REAL bottleneck that occurs in production on single machines

		// For single-machine testing, per-worker throughput should be at least 50% of baseline
		// This accounts for memory bandwidth saturation, GC pressure, and cache contention
		// Even with multiple CPU cores, these factors can limit scaling on a single machine
		// In a true distributed setup (separate machines), this would be much higher (near 100%)
		if efficiency < 50.0 {
			t.Errorf("Per-worker throughput degraded too much: %.1f%% (expected >= 50%% for single-machine test)", efficiency)
		}
	}
}

// BenchmarkWorkerScaling benchmarks different worker counts
func BenchmarkWorkerScaling(b *testing.B) {
	symbolCount := 2000
	ruleCount := 10
	workerCounts := []int{1, 2, 3, 4}

	symbols := generateSymbols(symbolCount)
	testRules := generateRules(ruleCount)

	for _, workerCount := range workerCounts {
		b.Run(fmt.Sprintf("Workers_%d", workerCount), func(b *testing.B) {
			// Setup workers
			partitionManagers := make([]*scanner.PartitionManager, workerCount)
			for i := 0; i < workerCount; i++ {
				pm, _ := scanner.NewPartitionManager(i, workerCount)
				partitionManagers[i] = pm
			}

			workerSymbols := make([][]string, workerCount)
			for _, symbol := range symbols {
				for workerID, pm := range partitionManagers {
					if pm.IsOwned(symbol) {
						workerSymbols[workerID] = append(workerSymbols[workerID], symbol)
						break
					}
				}
			}

			workers := make([]workerSetup, workerCount)
			for i := 0; i < workerCount; i++ {
				workers[i] = setupWorkerBenchmark(b, i, workerSymbols[i], testRules)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				for j := 0; j < workerCount; j++ {
					wg.Add(1)
					go func(workerID int) {
						defer wg.Done()
						workers[workerID].ScanLoop.Scan()
					}(j)
				}
				wg.Wait()
			}
		})
	}
}

// setupWorkerBenchmark is a simplified setup for benchmarks
func setupWorkerBenchmark(b *testing.B, workerID int, symbols []string, testRules []*models.Rule) workerSetup {
	sm := scanner.NewStateManager(200)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := newMockCooldownTracker()
	alertEmitter := newMockAlertEmitter()
	toplistIntegration := scanner.NewToplistIntegration(nil, false, 1*time.Second)

	for _, rule := range testRules {
		ruleStore.AddRule(rule)
	}

	for _, symbol := range symbols {
		tick := &models.Tick{
			Symbol:    symbol,
			Price:     100.0,
			Size:      100,
			Timestamp: time.Now(),
			Type:      "trade",
		}
		sm.UpdateLiveBar(symbol, tick)
		sm.UpdateIndicators(symbol, map[string]float64{"rsi_14": 50.0})
	}

	config := scanner.DefaultScanLoopConfig()
	sl := scanner.NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, alertEmitter, toplistIntegration)
	sl.ReloadRules()

	pm, _ := scanner.NewPartitionManager(workerID, 4)
	return workerSetup{
		StateManager:     sm,
		ScanLoop:         sl,
		PartitionManager: pm,
		Symbols:          symbols,
	}
}

// TestWorkerScaling_LoadDistribution verifies load is evenly distributed
func TestWorkerScaling_LoadDistribution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load distribution test in short mode")
	}

	symbolCount := 2000
	workerCounts := []int{1, 2, 3, 4}

	symbols := generateSymbols(symbolCount)

	for _, workerCount := range workerCounts {
		t.Run(fmt.Sprintf("Workers_%d", workerCount), func(t *testing.T) {
			// Create partition managers
			partitionManagers := make([]*scanner.PartitionManager, workerCount)
			for i := 0; i < workerCount; i++ {
				pm, err := scanner.NewPartitionManager(i, workerCount)
				if err != nil {
					t.Fatalf("Failed to create partition manager: %v", err)
				}
				partitionManagers[i] = pm
			}

			// Count symbols per worker
			workerSymbolCounts := make([]int, workerCount)
			for _, symbol := range symbols {
				for workerID, pm := range partitionManagers {
					if pm.IsOwned(symbol) {
						workerSymbolCounts[workerID]++
						break
					}
				}
			}

			// Calculate distribution
			expectedPerWorker := symbolCount / workerCount
			maxDeviation := float64(expectedPerWorker) * 0.2 // Allow 20% deviation

			t.Logf("Load distribution for %d workers:", workerCount)
			for i, count := range workerSymbolCounts {
				deviation := float64(count - expectedPerWorker)
				deviationPct := (deviation / float64(expectedPerWorker)) * 100
				t.Logf("  Worker %d: %d symbols (expected: %d, deviation: %.1f%%)",
					i, count, expectedPerWorker, deviationPct)

				if abs(deviation) > maxDeviation {
					t.Errorf("Worker %d has unbalanced load: %d symbols (expected ~%d, deviation: %.1f%%)",
						i, count, expectedPerWorker, deviationPct)
				}
			}
		})
	}
}

// TestWorkerScaling_ResourceUsage measures resource usage across worker counts
func TestWorkerScaling_ResourceUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resource usage test in short mode")
	}

	symbolCount := 1000
	ruleCount := 5
	workerCounts := []int{1, 2, 4}

	symbols := generateSymbols(symbolCount)
	testRules := generateRules(ruleCount)

	for _, workerCount := range workerCounts {
		t.Run(fmt.Sprintf("Workers_%d", workerCount), func(t *testing.T) {
			var m1, m2 runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&m1)

			// Setup and run workers
			partitionManagers := make([]*scanner.PartitionManager, workerCount)
			for i := 0; i < workerCount; i++ {
				pm, _ := scanner.NewPartitionManager(i, workerCount)
				partitionManagers[i] = pm
			}

			workerSymbols := make([][]string, workerCount)
			for _, symbol := range symbols {
				for workerID, pm := range partitionManagers {
					if pm.IsOwned(symbol) {
						workerSymbols[workerID] = append(workerSymbols[workerID], symbol)
						break
					}
				}
			}

			workers := make([]workerSetup, workerCount)
			for i := 0; i < workerCount; i++ {
				workers[i] = setupWorker(t, i, workerSymbols[i], testRules)
			}

			// Run scans
			var wg sync.WaitGroup
			for i := 0; i < workerCount; i++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()
					workers[workerID].ScanLoop.Scan()
				}(i)
			}
			wg.Wait()

			runtime.GC()
			runtime.ReadMemStats(&m2)

			// Calculate memory used (handle potential underflow)
			var memUsed float64
			if m2.HeapAlloc > m1.HeapAlloc {
				memUsed = float64(m2.HeapAlloc-m1.HeapAlloc) / 1024 / 1024
			} else {
				memUsed = 0 // Memory decreased (GC happened)
			}
			goroutines := runtime.NumGoroutine()

			t.Logf("Resource usage for %d workers:", workerCount)
			t.Logf("  Memory used: %.2f MB", memUsed)
			t.Logf("  Goroutines: %d", goroutines)
			if workerCount > 0 && memUsed > 0 {
				t.Logf("  Memory per worker: %.2f MB", memUsed/float64(workerCount))
			}
		})
	}
}

// Helper functions

func generateRules(count int) []*models.Rule {
	rules := make([]*models.Rule, count)
	for i := 0; i < count; i++ {
		rules[i] = &models.Rule{
			ID:         fmt.Sprintf("rule-%d", i),
			Name:       fmt.Sprintf("Test Rule %d", i),
			Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
			Cooldown:   300,
			Enabled:    true,
		}
	}
	return rules
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// mockCooldownTracker is a mock implementation for testing
type mockCooldownTracker struct {
	mu        sync.RWMutex
	cooldowns map[string]time.Time
}

func newMockCooldownTracker() *mockCooldownTracker {
	return &mockCooldownTracker{
		cooldowns: make(map[string]time.Time),
	}
}

func (m *mockCooldownTracker) IsOnCooldown(ruleID, symbol string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := ruleID + "|" + symbol
	cooldownEnd, exists := m.cooldowns[key]
	if !exists {
		return false
	}

	return time.Now().Before(cooldownEnd)
}

func (m *mockCooldownTracker) RecordCooldown(ruleID, symbol string, cooldownSeconds int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := ruleID + "|" + symbol
	m.cooldowns[key] = time.Now().Add(time.Duration(cooldownSeconds) * time.Second)
}

func (m *mockCooldownTracker) Start() error { return nil }
func (m *mockCooldownTracker) Stop()        {}

// mockAlertEmitter is a mock implementation for testing
type mockAlertEmitter struct {
	mu     sync.RWMutex
	alerts []*models.Alert
}

func newMockAlertEmitter() *mockAlertEmitter {
	return &mockAlertEmitter{
		alerts: make([]*models.Alert, 0),
	}
}

func (m *mockAlertEmitter) EmitAlert(alert *models.Alert) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alerts = append(m.alerts, alert)
	return nil
}
