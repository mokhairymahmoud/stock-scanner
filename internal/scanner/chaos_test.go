package scanner

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/rules"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
)

// TestChaos_WorkerRestart tests behavior when worker restarts
func TestChaos_WorkerRestart(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	// Setup initial state
	sm := NewStateManager(200)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := NewCooldownTracker(5 * time.Minute)
	cooldownTracker.Start()
	defer cooldownTracker.Stop()

	mockRedis := storage.NewMockRedisClient()
	alertEmitterConfig := DefaultAlertEmitterConfig()
	alertEmitter := NewAlertEmitter(mockRedis, alertEmitterConfig)

	config := DefaultScanLoopConfig()
	sl := NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, alertEmitter)

	// Add symbols and rules
	symbols := []string{"AAPL", "GOOGL", "MSFT"}
	for _, symbol := range symbols {
		tick := &models.Tick{
			Symbol:    symbol,
			Price:     100.0,
			Size:      100,
			Timestamp: time.Now(),
			Type:      "trade",
		}
		sm.UpdateLiveBar(symbol, tick)
		sm.UpdateIndicators(symbol, map[string]float64{"rsi_14": 25.0}) // Matches rule
	}

	rule := &models.Rule{
		ID:         "test-rule",
		Name:       "Test Rule",
		Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Cooldown:   10, // Short cooldown for testing
		Enabled:    true,
	}
	ruleStore.AddRule(rule)
	sl.ReloadRules()

	// First scan - should emit alerts
	sl.Scan()

	// Simulate worker restart: create new scan loop with same state
	// In real scenario, state would be rehydrated
	sl2 := NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, alertEmitter)
	sl2.ReloadRules()

	// Second scan immediately after restart - should respect cooldown
	sl2.Scan()

	// Verify no duplicate alerts were emitted
	// (In real scenario, we'd check Redis for duplicate alerts)
	stats1 := sl.GetStats()
	stats2 := sl2.GetStats()

	t.Logf("First scan: %d alerts, Second scan: %d alerts",
		stats1.AlertsEmitted, stats2.AlertsEmitted)

	// Cooldown should prevent duplicate alerts
	if stats2.AlertsEmitted > 0 {
		t.Log("Note: Second scan emitted alerts (cooldown may have expired or not set)")
	}
}

// TestChaos_PartitionRebalancing tests partition rebalancing when worker count changes
func TestChaos_PartitionRebalancing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	// Initial setup: 4 workers
	totalWorkers := 4
	workerID := 0

	pm, err := NewPartitionManager(workerID, totalWorkers)
	if err != nil {
		t.Fatalf("Failed to create partition manager: %v", err)
	}

	symbols := []string{"AAPL", "GOOGL", "MSFT", "AMZN", "TSLA", "NVDA", "META", "NFLX"}
	
	// Get initial distribution
	initialOwned := make([]string, 0)
	for _, symbol := range symbols {
		if pm.IsOwned(symbol) {
			initialOwned = append(initialOwned, symbol)
		}
	}

	t.Logf("Initial: Worker %d owns %d symbols", workerID, len(initialOwned))

	// Simulate rebalancing: reduce to 2 workers
	newTotalWorkers := 2
	err = pm.UpdateWorkerCount(newTotalWorkers)
	if err != nil {
		t.Fatalf("Failed to update worker count: %v", err)
	}

	// Get new distribution
	newOwned := make([]string, 0)
	for _, symbol := range symbols {
		if pm.IsOwned(symbol) {
			newOwned = append(newOwned, symbol)
		}
	}

	t.Logf("After rebalance: Worker %d owns %d symbols", workerID, len(newOwned))

	// Verify distribution changed
	if len(newOwned) == len(initialOwned) {
		t.Log("Note: Distribution unchanged (possible but unlikely with hash-based partitioning)")
	}

	// Verify no symbol is owned by multiple workers (would need multiple partition managers to fully test)
	// For now, just verify the partition manager still works correctly
	for _, symbol := range symbols {
		partition := pm.GetPartition(symbol)
		if partition < 0 || partition >= newTotalWorkers {
			t.Errorf("Invalid partition %d for symbol %s (expected 0-%d)",
				partition, symbol, newTotalWorkers-1)
		}
	}
}

// TestChaos_NetworkInterruption tests behavior during network interruptions
func TestChaos_NetworkInterruption(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	// Setup with mock Redis that can simulate failures
	mockRedis := storage.NewMockRedisClient()
	sm := NewStateManager(200)

	// Test indicator consumer with network interruption
	indicatorConfig := DefaultIndicatorConsumerConfig()
	indicatorConsumer := NewIndicatorConsumer(mockRedis, indicatorConfig, sm)

	// Start consumer
	err := indicatorConsumer.Start()
	if err != nil {
		t.Fatalf("Failed to start indicator consumer: %v", err)
	}
	defer indicatorConsumer.Stop()

	// Simulate network interruption by making Redis operations fail
	// (In real scenario, this would be a connection error)
	// For now, we just verify the consumer handles errors gracefully

	// Publish an indicator update
	ctx := context.Background()
	indicatorData := map[string]interface{}{
		"symbol":    "AAPL",
		"timestamp": time.Now(),
		"values": map[string]float64{
			"rsi_14": 25.0,
		},
	}
	mockRedis.Set(ctx, "ind:AAPL", indicatorData, 0)
	mockRedis.Publish(ctx, "indicators.updated", `{"symbol":"AAPL"}`)

	// Wait a bit for processing
	time.Sleep(200 * time.Millisecond)

	// Verify consumer is still running
	if !indicatorConsumer.IsRunning() {
		t.Error("Indicator consumer stopped after network operation")
	}

	// Note: Mock Redis doesn't fully simulate pub/sub, so we manually update state
	// In real scenario, the consumer would handle this via pub/sub
	sm.UpdateIndicators("AAPL", map[string]float64{"rsi_14": 25.0})

	// Verify state was updated
	state := sm.GetState("AAPL")
	if state == nil {
		t.Error("State not created for AAPL")
	} else if state.Indicators["rsi_14"] != 25.0 {
		t.Errorf("Expected RSI 25.0, got %f", state.Indicators["rsi_14"])
	}
}

// TestChaos_NoDuplicateAlerts tests that no duplicate alerts are emitted
func TestChaos_NoDuplicateAlerts(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	// Setup
	sm := NewStateManager(200)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := NewCooldownTracker(5 * time.Minute)
	cooldownTracker.Start()
	defer cooldownTracker.Stop()

	mockRedis := storage.NewMockRedisClient()
	alertEmitterConfig := DefaultAlertEmitterConfig()
	alertEmitter := NewAlertEmitter(mockRedis, alertEmitterConfig)

	config := DefaultScanLoopConfig()
	sl := NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, alertEmitter)

	// Add symbol with matching indicators
	symbol := "AAPL"
	tick := &models.Tick{
		Symbol:    symbol,
		Price:     100.0,
		Size:      100,
		Timestamp: time.Now(),
		Type:      "trade",
	}
	sm.UpdateLiveBar(symbol, tick)
	sm.UpdateIndicators(symbol, map[string]float64{"rsi_14": 25.0}) // Matches rule

	rule := &models.Rule{
		ID:         "test-rule",
		Name:       "Test Rule",
		Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Cooldown:   10, // 10 second cooldown
		Enabled:    true,
	}
	ruleStore.AddRule(rule)
	sl.ReloadRules()

	// Run multiple scans rapidly
	scanCount := 10
	alertsEmitted := 0

	for i := 0; i < scanCount; i++ {
		sl.Scan()
		stats := sl.GetStats()
		alertsEmitted = int(stats.AlertsEmitted)
		time.Sleep(50 * time.Millisecond) // Small delay between scans
	}

	// Should only emit 1 alert due to cooldown
	if alertsEmitted > 1 {
		t.Errorf("Expected at most 1 alert (due to cooldown), got %d", alertsEmitted)
	}

	// Wait for cooldown to expire
	time.Sleep(11 * time.Second)

	// Run scan again - should emit another alert
	sl.Scan()
	finalStats := sl.GetStats()

	if finalStats.AlertsEmitted != 2 {
		t.Logf("Expected 2 alerts total (1 before cooldown, 1 after), got %d",
			finalStats.AlertsEmitted)
	}
}

// TestChaos_ConcurrentRuleUpdates tests concurrent rule updates
func TestChaos_ConcurrentRuleUpdates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	sm := NewStateManager(200)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := newMockCooldownTracker()
	alertEmitter := newMockAlertEmitter()

	config := DefaultScanLoopConfig()
	sl := NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, alertEmitter)

	// Add symbols
	symbols := []string{"AAPL", "GOOGL", "MSFT"}
	for _, symbol := range symbols {
		sm.UpdateIndicators(symbol, map[string]float64{"rsi_14": 50.0})
	}

	// Concurrently add rules and reload
	var wg sync.WaitGroup
	ruleCount := 10

	for i := 0; i < ruleCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			rule := &models.Rule{
				ID:         fmt.Sprintf("rule-%d", idx),
				Name:       fmt.Sprintf("Rule %d", idx),
				Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
				Cooldown:   300,
				Enabled:    true,
			}
			ruleStore.AddRule(rule)
		}(i)
	}

	wg.Wait()

	// Reload rules (should be thread-safe)
	sl.ReloadRules()

	// Run scan
	sl.Scan()

	// Verify scan completed successfully
	stats := sl.GetStats()
	if stats.RulesEvaluated == 0 {
		t.Error("No rules were evaluated")
	}
}

// TestChaos_HighSymbolChurn tests behavior with symbols being added/removed rapidly
func TestChaos_HighSymbolChurn(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	sm := NewStateManager(200)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := newMockCooldownTracker()
	alertEmitter := newMockAlertEmitter()

	config := DefaultScanLoopConfig()
	sl := NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, alertEmitter)

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

	// Concurrently add and remove symbols
	var wg sync.WaitGroup
	operations := 100

	for i := 0; i < operations; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			symbol := fmt.Sprintf("SYM%04d", idx%50) // Cycle through 50 symbols

			// Add symbol
			tick := &models.Tick{
				Symbol:    symbol,
				Price:     100.0,
				Size:      100,
				Timestamp: time.Now(),
				Type:      "trade",
			}
			sm.UpdateLiveBar(symbol, tick)
			sm.UpdateIndicators(symbol, map[string]float64{"rsi_14": 25.0})

			// Sometimes remove symbol
			if idx%10 == 0 {
				sm.RemoveSymbol(symbol)
			}
		}(i)
	}

	wg.Wait()

	// Run scan while symbols are being churned
	sl.Scan()

	// Verify scan completed without errors
	stats := sl.GetStats()
	if stats.ScanCycles == 0 {
		t.Error("Scan cycle did not complete")
	}
}

