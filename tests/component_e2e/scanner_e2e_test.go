// Package data contains component-level E2E tests for the scanner worker.
//
// These tests verify the scanner component end-to-end using mocks for dependencies.
// They test state management, rule evaluation, cooldown, partitioning, and alerts.
//
// For full system E2E tests via API, see e2e_api_test.go
package data

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/rules"
	"github.com/mohamedkhairy/stock-scanner/internal/scanner"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
)

// TestScannerE2E_CompleteFlow tests the complete scanner workflow end-to-end
func TestScannerE2E_CompleteFlow(t *testing.T) {
	// Setup
	ctx := context.Background()
	sm := scanner.NewStateManager(200)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := scanner.NewCooldownTracker(10*time.Second, 5*time.Minute)
	alertEmitter := newMockAlertEmitterE2E()
	redis := storage.NewMockRedisClient()
	barStorage := newMockBarStorageE2E()

	// Initialize components
	config := scanner.DefaultScanLoopConfig()
	toplistIntegration := scanner.NewToplistIntegration(nil, false, 1*time.Second) // Disabled for this test
	scanLoop := scanner.NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, alertEmitter, toplistIntegration)

	// Create rehydrator
	rehydratorConfig := scanner.DefaultRehydrationConfig()
	rehydratorConfig.Symbols = []string{"AAPL", "GOOGL"}
	rehydrator := scanner.NewRehydrator(rehydratorConfig, sm, barStorage, redis)

	// Step 1: Rehydrate state (load historical data)
	t.Log("Step 1: Rehydrating state...")
	
	// Add historical bars
	bars := []*models.Bar1m{
		{
			Symbol:    "AAPL",
			Timestamp: time.Now().Add(-30 * time.Minute),
			Open:      150.0,
			High:      152.0,
			Low:       149.0,
			Close:     151.0,
			Volume:    1000,
			VWAP:      150.5,
		},
		{
			Symbol:    "AAPL",
			Timestamp: time.Now().Add(-25 * time.Minute),
			Open:      151.0,
			High:      153.0,
			Low:       150.0,
			Close:     152.0,
			Volume:    1200,
			VWAP:      151.5,
		},
		{
			Symbol:    "GOOGL",
			Timestamp: time.Now().Add(-30 * time.Minute),
			Open:      2500.0,
			High:      2520.0,
			Low:       2490.0,
			Close:     2510.0,
			Volume:    500,
			VWAP:      2505.0,
		},
	}
	barStorage.WriteBars(ctx, bars)

	// Add indicators
	indicatorsAAPL := map[string]interface{}{
		"symbol":    "AAPL",
		"timestamp": time.Now().UTC(),
		"values": map[string]float64{
			"rsi_14": 25.0, // Oversold - will match rule
			"ema_20": 150.2,
		},
	}
	redis.Set(ctx, "ind:AAPL", indicatorsAAPL, 0)

	indicatorsGOOGL := map[string]interface{}{
		"symbol":    "GOOGL",
		"timestamp": time.Now().UTC(),
		"values": map[string]float64{
			"rsi_14": 75.0, // Overbought - won't match rule
			"ema_20": 2500.2,
		},
	}
	redis.Set(ctx, "ind:GOOGL", indicatorsGOOGL, 0)

	// Rehydrate
	err := rehydrator.RehydrateState(ctx)
	if err != nil {
		t.Fatalf("Failed to rehydrate state: %v", err)
	}

	// Verify rehydration
	if sm.GetSymbolCount() != 2 {
		t.Errorf("Expected 2 symbols after rehydration, got %d", sm.GetSymbolCount())
	}

	// Step 2: Add rules
	t.Log("Step 2: Adding rules...")
	
	rule := &models.Rule{
		ID:   "rule-oversold",
		Name: "RSI Oversold",
		Conditions: []models.Condition{
			{Metric: "rsi_14", Operator: "<", Value: 30.0},
		},
		Cooldown: 10, // 10 seconds
		Enabled:  true,
	}

	err = ruleStore.AddRule(rule)
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	// Reload rules in scan loop
	err = scanLoop.ReloadRules()
	if err != nil {
		t.Fatalf("Failed to reload rules: %v", err)
	}

	// Step 3: Simulate tick ingestion
	t.Log("Step 3: Simulating tick ingestion...")
	
	// Note: In real scenario, TickConsumer would consume from Redis streams
	// For E2E test, we directly update state to simulate ingestion

	// Create tick stream messages
	now := time.Now()
	ticks := []*models.Tick{
		{
			Symbol:    "AAPL",
			Price:     152.5,
			Size:      100,
			Timestamp: now,
			Type:      "trade",
		},
		{
			Symbol:    "GOOGL",
			Price:     2520.0,
			Size:      50,
			Timestamp: now,
			Type:      "trade",
		},
	}

	// Simulate tick updates (directly update state for testing)
	for _, tick := range ticks {
		err := sm.UpdateLiveBar(tick.Symbol, tick)
		if err != nil {
			t.Fatalf("Failed to update live bar: %v", err)
		}
	}

	// Step 4: Simulate indicator updates
	t.Log("Step 4: Simulating indicator updates...")
	
	// Update indicators
	sm.UpdateIndicators("AAPL", map[string]float64{
		"rsi_14": 25.0, // Still oversold
		"ema_20": 150.5,
	})

	sm.UpdateIndicators("GOOGL", map[string]float64{
		"rsi_14": 75.0, // Still overbought
		"ema_20": 2500.5,
	})

	// Step 5: Simulate bar finalization
	t.Log("Step 5: Simulating bar finalization...")
	
	finalizedBar := &models.Bar1m{
		Symbol:    "AAPL",
		Timestamp: time.Now().Add(-1 * time.Minute).Truncate(time.Minute),
		Open:      152.0,
		High:      153.0,
		Low:       151.5,
		Close:     152.5,
		Volume:    1500,
		VWAP:      152.2,
	}

	err = sm.UpdateFinalizedBar(finalizedBar)
	if err != nil {
		t.Fatalf("Failed to update finalized bar: %v", err)
	}

	// Step 6: Run scan loop
	t.Log("Step 6: Running scan loop...")
	
	scanLoop.Scan()

	// Step 7: Verify alerts
	t.Log("Step 7: Verifying alerts...")
	
	alerts := alertEmitter.GetAlerts()
	
	// Should have 1 alert for AAPL (RSI < 30)
	if len(alerts) != 1 {
		t.Errorf("Expected 1 alert, got %d", len(alerts))
	}

	if len(alerts) > 0 {
		alert := alerts[0]
		if alert.Symbol != "AAPL" {
			t.Errorf("Expected alert for AAPL, got %s", alert.Symbol)
		}
		if alert.RuleID != "rule-oversold" {
			t.Errorf("Expected rule ID 'rule-oversold', got %s", alert.RuleID)
		}
	}

	// Step 8: Verify cooldown
	t.Log("Step 8: Verifying cooldown...")
	
	// Run scan again immediately - should not emit alert (cooldown)
	scanLoop.Scan()
	
	newAlerts := alertEmitter.GetAlerts()
	if len(newAlerts) != len(alerts) {
		t.Errorf("Expected no new alerts due to cooldown, got %d total alerts (was %d)", len(newAlerts), len(alerts))
	}

	// Verify cooldown is active
	if !cooldownTracker.IsOnCooldown("rule-oversold", "AAPL") {
		t.Error("Expected rule to be on cooldown")
	}

	// Step 9: Verify stats
	t.Log("Step 9: Verifying statistics...")
	
	stats := scanLoop.GetStats()
	if stats.SymbolsScanned == 0 {
		t.Error("Expected symbols to be scanned")
	}
	if stats.RulesEvaluated == 0 {
		t.Error("Expected rules to be evaluated")
	}
	if stats.RulesMatched == 0 {
		t.Error("Expected rules to match")
	}
	if stats.AlertsEmitted == 0 {
		t.Error("Expected alerts to be emitted")
	}

	t.Log("✅ E2E test completed successfully!")
}

// TestScannerE2E_Partitioning tests partitioning functionality
func TestScannerE2E_Partitioning(t *testing.T) {
	// Create partition manager
	pm, err := scanner.NewPartitionManager(1, 4)
	if err != nil {
		t.Fatalf("Failed to create partition manager: %v", err)
	}

	symbols := []string{"AAPL", "GOOGL", "MSFT", "TSLA", "AMZN", "NVDA", "META", "NFLX"}
	
	// Test ownership
	ownedCount := 0
	for _, symbol := range symbols {
		if pm.IsOwned(symbol) {
			ownedCount++
			pm.AddAssignedSymbol(symbol)
		}
	}

	t.Logf("Worker 1 owns %d out of %d symbols", ownedCount, len(symbols))

	// Verify assigned symbols
	assigned := pm.GetAssignedSymbols()
	if len(assigned) != ownedCount {
		t.Errorf("Expected %d assigned symbols, got %d", ownedCount, len(assigned))
	}

	// Test distribution
	distribution := pm.GetPartitionDistribution(symbols)
	total := 0
	for partition, count := range distribution {
		total += count
		t.Logf("Partition %d: %d symbols", partition, count)
	}

	if total != len(symbols) {
		t.Errorf("Expected %d symbols in distribution, got %d", len(symbols), total)
	}
}

// TestScannerE2E_MultipleRules tests multiple rules with different conditions
func TestScannerE2E_MultipleRules(t *testing.T) {
	sm := scanner.NewStateManager(200)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := scanner.NewCooldownTracker(10*time.Second, 5*time.Minute)
	alertEmitter := newMockAlertEmitterE2E()
	config := scanner.DefaultScanLoopConfig()
	toplistIntegration := scanner.NewToplistIntegration(nil, false, 1*time.Second) // Disabled for this test
	scanLoop := scanner.NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, alertEmitter, toplistIntegration)

	// Add multiple rules
	rules := []*models.Rule{
		{
			ID:   "rule-oversold",
			Name: "RSI Oversold",
			Conditions: []models.Condition{
				{Metric: "rsi_14", Operator: "<", Value: 30.0},
			},
			Cooldown: 10,
			Enabled:  true,
		},
		{
			ID:   "rule-overbought",
			Name: "RSI Overbought",
			Conditions: []models.Condition{
				{Metric: "rsi_14", Operator: ">", Value: 70.0},
			},
			Cooldown: 10,
			Enabled:  true,
		},
		{
			ID:   "rule-complex",
			Name: "Complex Rule",
			Conditions: []models.Condition{
				{Metric: "rsi_14", Operator: "<", Value: 30.0},
				{Metric: "price_change_5m_pct", Operator: ">", Value: 1.0},
			},
			Cooldown: 10,
			Enabled:  true,
		},
	}

	for _, rule := range rules {
		ruleStore.AddRule(rule)
	}

	scanLoop.ReloadRules()

	// Setup symbol with matching indicators
	symbol := "AAPL"
	
	// Add finalized bars for price change calculation
	for i := 0; i < 6; i++ {
		bar := &models.Bar1m{
			Symbol:    symbol,
			Timestamp: time.Now().Add(time.Duration(i-5) * time.Minute),
			Open:      150.0 + float64(i)*0.1,
			High:      152.0 + float64(i)*0.1,
			Low:       149.0 + float64(i)*0.1,
			Close:     151.0 + float64(i)*0.1,
			Volume:    1000,
			VWAP:      150.5 + float64(i)*0.1,
		}
		sm.UpdateFinalizedBar(bar)
	}

	// Add indicators
	sm.UpdateIndicators(symbol, map[string]float64{
		"rsi_14": 25.0, // Matches oversold rule
	})

	// Add live bar with price change
	tick := &models.Tick{
		Symbol:    symbol,
		Price:     160.0, // Large price change
		Size:      100,
		Timestamp: time.Now(),
		Type:      "trade",
	}
	sm.UpdateLiveBar(symbol, tick)

	// Scan
	scanLoop.Scan()

	// Verify alerts
	alerts := alertEmitter.GetAlerts()
	
	// Should have alerts for oversold rule and possibly complex rule
	if len(alerts) == 0 {
		t.Error("Expected at least one alert")
	}

	// Verify rule IDs
	ruleIDs := make(map[string]bool)
	for _, alert := range alerts {
		ruleIDs[alert.RuleID] = true
	}

	if !ruleIDs["rule-oversold"] {
		t.Error("Expected alert for rule-oversold")
	}
}

// mockAlertEmitterE2E is a mock alert emitter for E2E testing
type mockAlertEmitterE2E struct {
	alerts []*models.Alert
	mu     sync.RWMutex
}

func newMockAlertEmitterE2E() *mockAlertEmitterE2E {
	return &mockAlertEmitterE2E{
		alerts: make([]*models.Alert, 0),
	}
}

func (m *mockAlertEmitterE2E) EmitAlert(alert *models.Alert) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alerts = append(m.alerts, alert)
	return nil
}

func (m *mockAlertEmitterE2E) GetAlerts() []*models.Alert {
	m.mu.RLock()
	defer m.mu.RUnlock()
	alerts := make([]*models.Alert, len(m.alerts))
	copy(alerts, m.alerts)
	return alerts
}

// mockBarStorageE2E is a mock bar storage for E2E testing
type mockBarStorageE2E struct {
	bars map[string][]*models.Bar1m
	mu   sync.RWMutex
}

func newMockBarStorageE2E() *mockBarStorageE2E {
	return &mockBarStorageE2E{
		bars: make(map[string][]*models.Bar1m),
	}
}

func (m *mockBarStorageE2E) WriteBars(ctx context.Context, bars []*models.Bar1m) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, bar := range bars {
		m.bars[bar.Symbol] = append(m.bars[bar.Symbol], bar)
	}
	return nil
}

func (m *mockBarStorageE2E) GetBars(ctx context.Context, symbol string, start, end time.Time) ([]*models.Bar1m, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	symbolBars := m.bars[symbol]
	result := make([]*models.Bar1m, 0)
	for _, bar := range symbolBars {
		if !bar.Timestamp.Before(start) && !bar.Timestamp.After(end) {
			result = append(result, bar)
		}
	}
	return result, nil
}

func (m *mockBarStorageE2E) GetLatestBars(ctx context.Context, symbol string, limit int) ([]*models.Bar1m, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	symbolBars := m.bars[symbol]
	if len(symbolBars) == 0 {
		return []*models.Bar1m{}, nil
	}
	start := len(symbolBars) - limit
	if start < 0 {
		start = 0
	}
	return symbolBars[start:], nil
}

func (m *mockBarStorageE2E) Close() error {
	return nil
}

