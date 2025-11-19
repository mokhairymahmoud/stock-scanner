package scanner

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/rules"
)

// BenchmarkScanLoop_2000Symbols benchmarks the scan loop with 2000 symbols
func BenchmarkScanLoop_2000Symbols(b *testing.B) {
	// Setup
	sm := NewStateManager(200)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := newMockCooldownTracker()
	alertEmitter := newMockAlertEmitter()

	config := DefaultScanLoopConfig()
	sl := NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, alertEmitter)

	// Add 2000 symbols with state
	symbolCount := 2000
	for i := 0; i < symbolCount; i++ {
		symbol := generateSymbol(i)
		
		// Add live bar
		tick := &models.Tick{
			Symbol:    symbol,
			Price:     100.0 + float64(i%100),
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
			"rsi_14":   50.0 + float64(i%30),
			"ema_20":   100.0 + float64(i%10),
			"sma_50":   100.0 + float64(i%10),
			"vwap_5m":  100.0 + float64(i%5),
		}
		sm.UpdateIndicators(symbol, indicators)
	}

	// Add a simple rule
	rule := &models.Rule{
		ID:         "test-rule",
		Name:       "Test Rule",
		Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Cooldown:   300,
		Enabled:    true,
	}
	ruleStore.AddRule(rule)
	sl.ReloadRules()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		sl.Scan()
	}
}

// BenchmarkScanLoop_VaryingRuleCounts benchmarks scan loop with different rule counts
func BenchmarkScanLoop_VaryingRuleCounts(b *testing.B) {
	ruleCounts := []int{1, 10, 50, 100}

	for _, ruleCount := range ruleCounts {
		b.Run(fmt.Sprintf("Rules_%d", ruleCount), func(b *testing.B) {
			// Setup
			sm := NewStateManager(200)
			ruleStore := rules.NewInMemoryRuleStore()
			compiler := rules.NewCompiler(nil)
			cooldownTracker := newMockCooldownTracker()
			alertEmitter := newMockAlertEmitter()

			config := DefaultScanLoopConfig()
			sl := NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, alertEmitter)

			// Add 100 symbols
			for i := 0; i < 100; i++ {
				symbol := generateSymbol(i)
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

			// Add rules
			for i := 0; i < ruleCount; i++ {
				rule := &models.Rule{
					ID:         fmt.Sprintf("rule-%d", i),
					Name:       fmt.Sprintf("Rule %d", i),
					Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
					Cooldown:   300,
					Enabled:    true,
				}
				ruleStore.AddRule(rule)
			}
			sl.ReloadRules()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				sl.Scan()
			}
		})
	}
}

// TestScanLoop_Performance_2000Symbols tests scan loop performance with 2000 symbols
func TestScanLoop_Performance_2000Symbols(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Setup
	sm := NewStateManager(200)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := newMockCooldownTracker()
	alertEmitter := newMockAlertEmitter()

	config := DefaultScanLoopConfig()
	sl := NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, alertEmitter)

	// Add 2000 symbols with state
	symbolCount := 2000
	for i := 0; i < symbolCount; i++ {
		symbol := generateSymbol(i)
		
		// Add live bar
		tick := &models.Tick{
			Symbol:    symbol,
			Price:     100.0 + float64(i%100),
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
			"rsi_14":   50.0 + float64(i%30),
			"ema_20":   100.0 + float64(i%10),
			"sma_50":   100.0 + float64(i%10),
			"vwap_5m":  100.0 + float64(i%5),
		}
		sm.UpdateIndicators(symbol, indicators)
	}

	// Add a simple rule
	rule := &models.Rule{
		ID:         "test-rule",
		Name:       "Test Rule",
		Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Cooldown:   300,
		Enabled:    true,
	}
	ruleStore.AddRule(rule)
	sl.ReloadRules()

	// Run scan and measure time
	start := time.Now()
	sl.Scan()
	duration := time.Since(start)

	// Verify performance target: < 800ms
	if duration > 800*time.Millisecond {
		t.Errorf("Scan took %v, expected < 800ms", duration)
	}

	stats := sl.GetStats()
	t.Logf("Scan performance: %v for %d symbols, %d rules evaluated, %d matched",
		duration, stats.SymbolsScanned, stats.RulesEvaluated, stats.RulesMatched)
}

// TestScanLoop_Performance_TickBurst tests scan loop with tick bursts
func TestScanLoop_Performance_TickBurst(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Setup
	sm := NewStateManager(200)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := newMockCooldownTracker()
	alertEmitter := newMockAlertEmitter()

	config := DefaultScanLoopConfig()
	sl := NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, alertEmitter)

	// Add 500 symbols
	symbolCount := 500
	for i := 0; i < symbolCount; i++ {
		symbol := generateSymbol(i)
		sm.UpdateIndicators(symbol, map[string]float64{"rsi_14": 50.0})
	}

	// Add rule
	rule := &models.Rule{
		ID:         "test-rule",
		Name:       "Test Rule",
		Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Cooldown:   300,
		Enabled:    true,
	}
	ruleStore.AddRule(rule)
	sl.ReloadRules()

	// Simulate tick burst: update all symbols concurrently
	var wg sync.WaitGroup
	burstSize := 1000
	start := time.Now()

	for i := 0; i < burstSize; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			symbol := generateSymbol(idx % symbolCount)
			tick := &models.Tick{
				Symbol:    symbol,
				Price:     100.0 + float64(idx%100),
				Size:      100,
				Timestamp: time.Now(),
				Type:      "trade",
			}
			sm.UpdateLiveBar(symbol, tick)
		}(i)
	}

	wg.Wait()
	updateDuration := time.Since(start)

	// Run scan
	scanStart := time.Now()
	sl.Scan()
	scanDuration := time.Since(scanStart)

	t.Logf("Tick burst performance: %d ticks in %v, scan in %v",
		burstSize, updateDuration, scanDuration)

	// Verify scan still completes in reasonable time even after burst
	if scanDuration > 1*time.Second {
		t.Errorf("Scan took %v after tick burst, expected < 1s", scanDuration)
	}
}

// TestStateManager_Performance_ConcurrentUpdates tests concurrent state updates
func TestStateManager_Performance_ConcurrentUpdates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	sm := NewStateManager(200)
	symbolCount := 1000
	updatesPerSymbol := 100

	var wg sync.WaitGroup
	start := time.Now()

	// Concurrent updates
	for i := 0; i < symbolCount; i++ {
		symbol := generateSymbol(i)
		for j := 0; j < updatesPerSymbol; j++ {
			wg.Add(1)
			go func(sym string, idx int) {
				defer wg.Done()
				tick := &models.Tick{
					Symbol:    sym,
					Price:     100.0 + float64(idx%100),
					Size:      100,
					Timestamp: time.Now(),
					Type:      "trade",
				}
				sm.UpdateLiveBar(sym, tick)
			}(symbol, j)
		}
	}

	wg.Wait()
	duration := time.Since(start)

	t.Logf("Concurrent updates: %d symbols × %d updates = %d total updates in %v",
		symbolCount, updatesPerSymbol, symbolCount*updatesPerSymbol, duration)

	// Verify all symbols have state
	if sm.GetSymbolCount() != symbolCount {
		t.Errorf("Expected %d symbols, got %d", symbolCount, sm.GetSymbolCount())
	}
}

// generateSymbol generates a symbol name from an index
func generateSymbol(index int) string {
	// Generate symbols like SYM0001, SYM0002, etc.
	return fmt.Sprintf("SYM%04d", index)
}

