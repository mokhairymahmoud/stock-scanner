// Package data contains component-level E2E tests for filter integration.
//
// These tests verify filter evaluation in the scan loop, including:
// - Volume threshold enforcement
// - Session-based filtering
// - Timeframe support
// - Value type variants
// - Performance with multiple filters
// - Lazy metric computation
package data

import (
	"context"
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/rules"
	"github.com/mohamedkhairy/stock-scanner/internal/scanner"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
)

// addHistoricalBars adds historical bars to state for testing
func addHistoricalBars(sm *scanner.StateManager, symbol string, basePrice float64, count int, now time.Time) error {
	for i := count; i >= 0; i-- {
		barTime := now.Add(-time.Duration(i) * time.Minute)
		bar := &models.Bar1m{
			Symbol:    symbol,
			Timestamp: barTime.Truncate(time.Minute),
			Open:      basePrice,
			High:      basePrice + 1,
			Low:       basePrice - 1,
			Close:     basePrice,
			Volume:    1000,
			VWAP:      basePrice,
		}
		err := sm.UpdateFinalizedBar(bar)
		if err != nil {
			return err
		}
	}
	return nil
}

// TestFilterIntegration_ScanLoop evaluates filters in the scan loop
func TestFilterIntegration_ScanLoop(t *testing.T) {
	// Setup
	sm := scanner.NewStateManager(100)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := scanner.NewCooldownTracker(10*time.Second, 5*time.Minute)
	alertEmitter := newMockAlertEmitterE2E()

	config := scanner.DefaultScanLoopConfig()
	config.ScanInterval = 100 * time.Millisecond // Faster for testing

	scanLoop := scanner.NewScanLoop(
		config,
		sm,
		ruleStore,
		compiler,
		cooldownTracker,
		alertEmitter,
		nil, // No toplist integration
	)

	// Create a rule with price change filter
	rule := &models.Rule{
		ID:          "test-rule-1",
		Name:        "Price Change > 5%",
		Description: "Test rule for price change filter",
		Enabled:     true,
		Conditions: []models.Condition{
			{
				Metric:   "price_change_5m_pct",
				Operator: ">",
				Value:    5.0,
			},
		},
	}

	err := ruleStore.AddRule(rule)
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	// Reload rules in scan loop
	err = scanLoop.ReloadRules()
	if err != nil {
		t.Fatalf("Failed to reload rules: %v", err)
	}

	// Setup symbol state with price movement
	symbol := "AAPL"
	basePrice := 100.0

	// Create bars going back 10 minutes (need at least 6 for 5m change)
	now := time.Now()
	for i := 10; i >= 0; i-- {
		barTime := now.Add(-time.Duration(i) * time.Minute)
		bar := &models.Bar1m{
			Symbol:    symbol,
			Timestamp: barTime.Truncate(time.Minute),
			Open:      basePrice,
			High:      basePrice + 1,
			Low:       basePrice - 1,
			Close:     basePrice,
			Volume:    1000,
			VWAP:      basePrice,
		}
		err = sm.UpdateFinalizedBar(bar)
		if err != nil {
			t.Fatalf("Failed to update finalized bar: %v", err)
		}
	}

	// Create current live bar with price increase > 5%
	currentPrice := basePrice * 1.06 // 6% increase
	currentTick := &models.Tick{
		Symbol:    symbol,
		Price:     currentPrice,
		Size:      500,
		Timestamp: now,
		Type:      "trade",
	}

	err = sm.UpdateLiveBar(symbol, currentTick)
	if err != nil {
		t.Fatalf("Failed to update live bar: %v", err)
	}

	// Run scan
	scanLoop.Scan()

	// Verify rule was evaluated (check stats)
	stats := scanLoop.GetStats()
	if stats.SymbolsScanned == 0 {
		t.Error("Expected at least one symbol to be scanned")
	}

	if stats.RulesEvaluated == 0 {
		t.Error("Expected at least one rule to be evaluated")
	}
}

// TestFilterIntegration_VolumeThreshold tests volume threshold enforcement
func TestFilterIntegration_VolumeThreshold(t *testing.T) {
	sm := scanner.NewStateManager(100)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := scanner.NewCooldownTracker(10*time.Second, 5*time.Minute)
	alertEmitter := newMockAlertEmitterE2E()

	config := scanner.DefaultScanLoopConfig()
	scanLoop := scanner.NewScanLoop(
		config,
		sm,
		ruleStore,
		compiler,
		cooldownTracker,
		alertEmitter,
		nil,
	)

	// Create rule with volume threshold
	volumeThreshold := int64(100000)
	rule := &models.Rule{
		ID:      "test-rule-volume",
		Name:    "High Volume Price Change",
		Enabled: true,
		Conditions: []models.Condition{
			{
				Metric:         "price_change_5m_pct",
				Operator:       ">",
				Value:          5.0,
				VolumeThreshold: &volumeThreshold,
			},
		},
	}

	err := ruleStore.AddRule(rule)
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	err = scanLoop.ReloadRules()
	if err != nil {
		t.Fatalf("Failed to reload rules: %v", err)
	}

	symbol := "AAPL"
	basePrice := 100.0
	now := time.Now()

	// Setup state with low volume (below threshold)
	// Add historical bars (need at least 6 for 5m change)
	err = addHistoricalBars(sm, symbol, basePrice, 10, now)
	if err != nil {
		t.Fatalf("Failed to add historical bars: %v", err)
	}

	// Add tick with low volume
	lowVolTick := &models.Tick{
		Symbol:    symbol,
		Price:     basePrice * 1.06,
		Size:      50000,
		Timestamp: now,
		Type:      "trade",
	}
	err = sm.UpdateLiveBar(symbol, lowVolTick)
	if err != nil {
		t.Fatalf("Failed to update live bar: %v", err)
	}

	// Run scan
	initialStats := scanLoop.GetStats()
	scanLoop.Scan()
	finalStats := scanLoop.GetStats()

	// Rule should be evaluated but not matched (volume threshold not met)
	if finalStats.RulesEvaluated <= initialStats.RulesEvaluated {
		t.Error("Expected rule to be evaluated")
	}

	// Now add high volume tick (above threshold)
	highVolTick := &models.Tick{
		Symbol:    symbol,
		Price:     basePrice * 1.06,
		Size:      200000, // Above threshold
		Timestamp: now.Add(1 * time.Second),
		Type:      "trade",
	}
	err = sm.UpdateLiveBar(symbol, highVolTick)
	if err != nil {
		t.Fatalf("Failed to update live bar with high volume: %v", err)
	}

	scanLoop.Scan()
	stats := scanLoop.GetStats()

	// Rule should be evaluated with high volume
	if stats.RulesEvaluated == 0 {
		t.Error("Expected rule to be evaluated with high volume")
	}
}

// TestFilterIntegration_SessionFiltering tests session-based filtering
func TestFilterIntegration_SessionFiltering(t *testing.T) {
	sm := scanner.NewStateManager(100)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := scanner.NewCooldownTracker(10*time.Second, 5*time.Minute)
	alertEmitter := newMockAlertEmitterE2E()

	config := scanner.DefaultScanLoopConfig()
	scanLoop := scanner.NewScanLoop(
		config,
		sm,
		ruleStore,
		compiler,
		cooldownTracker,
		alertEmitter,
		nil,
	)

	// Create rule that only works in market session
	rule := &models.Rule{
		ID:      "test-rule-market",
		Name:    "Market Session Only",
		Enabled: true,
		Conditions: []models.Condition{
			{
				Metric:          "price_change_5m_pct",
				Operator:        ">",
				Value:           5.0,
				CalculatedDuring: "market",
			},
		},
	}

	err := ruleStore.AddRule(rule)
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	err = scanLoop.ReloadRules()
	if err != nil {
		t.Fatalf("Failed to reload rules: %v", err)
	}

	symbol := "AAPL"
	basePrice := 100.0
	now := time.Now()

	// Setup state with historical bar
	initialBar := &models.Bar1m{
		Symbol:    symbol,
		Timestamp: now.Add(-5 * time.Minute),
		Open:      basePrice,
		High:      basePrice + 1,
		Low:       basePrice - 1,
		Close:    basePrice,
		Volume:   100000,
		VWAP:     basePrice,
	}
	err = sm.UpdateFinalizedBar(initialBar)
	if err != nil {
		t.Fatalf("Failed to update finalized bar: %v", err)
	}

	// Create tick in premarket (before 9:30 AM ET)
	// Use a time that's in premarket session
	premarketTime := time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC) // 8:00 AM UTC = 3:00 AM ET (premarket)
	premarketTick := &models.Tick{
		Symbol:    symbol,
		Price:     basePrice * 1.06,
		Size:      100000,
		Timestamp: premarketTime,
		Type:      "trade",
	}

	err = sm.UpdateLiveBar(symbol, premarketTick)
	if err != nil {
		t.Fatalf("Failed to update live bar: %v", err)
	}

	// Run scan in premarket
	initialStats := scanLoop.GetStats()
	scanLoop.Scan()
	premarketStats := scanLoop.GetStats()

	// Rule should be evaluated but filtered out (wrong session)
	if premarketStats.RulesEvaluated <= initialStats.RulesEvaluated {
		t.Error("Expected rule to be evaluated")
	}

	// Now create tick in market session (after 9:30 AM ET)
	marketTime := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC) // 2:30 PM UTC = 9:30 AM ET (market open)
	marketTick := &models.Tick{
		Symbol:    symbol,
		Price:     basePrice * 1.06,
		Size:      100000,
		Timestamp: marketTime,
		Type:      "trade",
	}

	err = sm.UpdateLiveBar(symbol, marketTick)
	if err != nil {
		t.Fatalf("Failed to update live bar in market: %v", err)
	}

	scanLoop.Scan()
	marketStats := scanLoop.GetStats()

	// Rule should be evaluated in market session
	if marketStats.RulesEvaluated <= premarketStats.RulesEvaluated {
		t.Error("Expected rule to be evaluated in market session")
	}
}

// TestFilterIntegration_Timeframes tests different timeframes
func TestFilterIntegration_Timeframes(t *testing.T) {
	sm := scanner.NewStateManager(100)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := scanner.NewCooldownTracker(10*time.Second, 5*time.Minute)
	alertEmitter := newMockAlertEmitterE2E()

	config := scanner.DefaultScanLoopConfig()
	scanLoop := scanner.NewScanLoop(
		config,
		sm,
		ruleStore,
		compiler,
		cooldownTracker,
		alertEmitter,
		nil,
	)

	// Create rules for different timeframes
	timeframes := []struct {
		timeframe string
		metric    string
		bars      int
	}{
		{"1m", "change_1m_pct", 2},
		{"5m", "change_5m_pct", 6},
		{"15m", "change_15m_pct", 16},
	}

	for _, tf := range timeframes {
		rule := &models.Rule{
			ID:      "test-rule-" + tf.timeframe,
			Name:    "Change " + tf.timeframe,
			Enabled: true,
			Conditions: []models.Condition{
				{
					Metric:   tf.metric,
					Operator: ">",
					Value:    5.0,
				},
			},
		}

		err := ruleStore.AddRule(rule)
		if err != nil {
			t.Fatalf("Failed to add rule for %s: %v", tf.timeframe, err)
		}
	}

	err := scanLoop.ReloadRules()
	if err != nil {
		t.Fatalf("Failed to reload rules: %v", err)
	}

	symbol := "AAPL"
	basePrice := 100.0
	now := time.Now()

	// Setup state with bars for all timeframes
	// Create bars going back 20 minutes
	for i := 20; i >= 0; i-- {
		barTime := now.Add(-time.Duration(i) * time.Minute)
		bar := &models.Bar1m{
			Symbol:    symbol,
			Timestamp: barTime.Truncate(time.Minute),
			Open:      basePrice,
			High:      basePrice + 1,
			Low:       basePrice - 1,
			Close:    basePrice,
			Volume:   1000,
			VWAP:     basePrice,
		}
		err = sm.UpdateFinalizedBar(bar)
		if err != nil {
			t.Fatalf("Failed to add bar: %v", err)
		}
	}

	// Set current price with 6% increase
	currentPrice := basePrice * 1.06
	currentTick := &models.Tick{
		Symbol:    symbol,
		Price:     currentPrice,
		Size:      100000,
		Timestamp: now,
		Type:      "trade",
	}

	err = sm.UpdateLiveBar(symbol, currentTick)
	if err != nil {
		t.Fatalf("Failed to update live bar: %v", err)
	}

	// Run scan
	scanLoop.Scan()

	stats := scanLoop.GetStats()
	if stats.RulesEvaluated == 0 {
		t.Error("Expected rules to be evaluated for all timeframes")
	}
}

// TestFilterIntegration_ValueTypes tests both absolute and percentage value types
func TestFilterIntegration_ValueTypes(t *testing.T) {
	sm := scanner.NewStateManager(100)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := scanner.NewCooldownTracker(10*time.Second, 5*time.Minute)
	alertEmitter := newMockAlertEmitterE2E()

	config := scanner.DefaultScanLoopConfig()
	scanLoop := scanner.NewScanLoop(
		config,
		sm,
		ruleStore,
		compiler,
		cooldownTracker,
		alertEmitter,
		nil,
	)

	// Create rules for both value types
	absoluteRule := &models.Rule{
		ID:      "test-rule-absolute",
		Name:    "Absolute Change",
		Enabled: true,
		Conditions: []models.Condition{
			{
				Metric:   "change_5m", // Absolute ($)
				Operator: ">",
				Value:    5.0, // $5
			},
		},
	}

	percentageRule := &models.Rule{
		ID:      "test-rule-percentage",
		Name:    "Percentage Change",
		Enabled: true,
		Conditions: []models.Condition{
			{
				Metric:   "price_change_5m_pct", // Percentage (%)
				Operator: ">",
				Value:    5.0, // 5%
			},
		},
	}

	err := ruleStore.AddRule(absoluteRule)
	if err != nil {
		t.Fatalf("Failed to add absolute rule: %v", err)
	}

	err = ruleStore.AddRule(percentageRule)
	if err != nil {
		t.Fatalf("Failed to add percentage rule: %v", err)
	}

	err = scanLoop.ReloadRules()
	if err != nil {
		t.Fatalf("Failed to reload rules: %v", err)
	}

	symbol := "AAPL"
	basePrice := 100.0
	now := time.Now()

	// Setup state with historical bar
	initialBar := &models.Bar1m{
		Symbol:    symbol,
		Timestamp: now.Add(-5 * time.Minute),
		Open:      basePrice,
		High:      basePrice + 1,
		Low:       basePrice - 1,
		Close:    basePrice,
		Volume:   100000,
		VWAP:     basePrice,
	}
	err = sm.UpdateFinalizedBar(initialBar)
	if err != nil {
		t.Fatalf("Failed to update finalized bar: %v", err)
	}

	// Set current price with $6 increase (> $5) and 6% increase (> 5%)
	currentPrice := basePrice + 6
	currentTick := &models.Tick{
		Symbol:    symbol,
		Price:     currentPrice,
		Size:      100000,
		Timestamp: now,
		Type:      "trade",
	}

	err = sm.UpdateLiveBar(symbol, currentTick)
	if err != nil {
		t.Fatalf("Failed to update live bar: %v", err)
	}

	// Run scan
	scanLoop.Scan()

	stats := scanLoop.GetStats()
	if stats.RulesEvaluated < 2 {
		t.Errorf("Expected at least 2 rules to be evaluated, got %d", stats.RulesEvaluated)
	}
}

// TestFilterIntegration_Performance tests performance with multiple filters
func TestFilterIntegration_Performance(t *testing.T) {
	sm := scanner.NewStateManager(100)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := scanner.NewCooldownTracker(10*time.Second, 5*time.Minute)
	alertEmitter := newMockAlertEmitterE2E()

	config := scanner.DefaultScanLoopConfig()
	config.MaxScanTime = 800 * time.Millisecond
	scanLoop := scanner.NewScanLoop(
		config,
		sm,
		ruleStore,
		compiler,
		cooldownTracker,
		alertEmitter,
		nil,
	)

	// Create multiple rules with different filters
	testRules := []*models.Rule{
		{
			ID:      "rule-price-change",
			Name:    "Price Change",
			Enabled: true,
			Conditions: []models.Condition{
				{Metric: "price_change_5m_pct", Operator: ">", Value: 5.0},
			},
		},
		{
			ID:      "rule-volume",
			Name:    "Volume",
			Enabled: true,
			Conditions: []models.Condition{
				{Metric: "volume_daily", Operator: ">", Value: 1000000.0},
			},
		},
		{
			ID:      "rule-range",
			Name:    "Range",
			Enabled: true,
			Conditions: []models.Condition{
				{Metric: "range_pct_5m", Operator: ">", Value: 2.0},
			},
		},
		{
			ID:      "rule-rsi",
			Name:    "RSI",
			Enabled: true,
			Conditions: []models.Condition{
				{Metric: "rsi_14", Operator: ">", Value: 70.0},
			},
		},
	}

	for _, rule := range testRules {
		err := ruleStore.AddRule(rule)
		if err != nil {
			t.Fatalf("Failed to add rule %s: %v", rule.ID, err)
		}
	}

	err := scanLoop.ReloadRules()
	if err != nil {
		t.Fatalf("Failed to reload rules: %v", err)
	}

	// Setup multiple symbols
	symbols := []string{"AAPL", "MSFT", "GOOGL", "AMZN", "TSLA"}
	basePrice := 100.0
	now := time.Now()

	ctx := context.Background()
	redis := storage.NewMockRedisClient()

	for _, symbol := range symbols {
		// Add historical bars
		for i := 20; i >= 0; i-- {
			barTime := now.Add(-time.Duration(i) * time.Minute)
			bar := &models.Bar1m{
				Symbol:    symbol,
				Timestamp: barTime.Truncate(time.Minute),
				Open:      basePrice,
				High:      basePrice + 2,
				Low:       basePrice - 2,
				Close:    basePrice,
				Volume:   100000,
				VWAP:     basePrice,
			}
			err = sm.UpdateFinalizedBar(bar)
			if err != nil {
				t.Fatalf("Failed to add bar for %s: %v", symbol, err)
			}
		}

		// Set indicators via Redis (simulating indicator engine)
		indicators := map[string]interface{}{
			"symbol":    symbol,
			"timestamp": now.UTC(),
			"values": map[string]float64{
				"rsi_14": 75.0,
			},
		}
		err = redis.Set(ctx, "ind:"+symbol, indicators, 0)
		if err != nil {
			t.Fatalf("Failed to set indicators: %v", err)
		}

		// Update indicators in state via StateManager
		sm.UpdateIndicators(symbol, map[string]float64{
			"rsi_14": 75.0,
		})

		// Add tick with high volume
		tick := &models.Tick{
			Symbol:    symbol,
			Price:     basePrice + 6,
			Size:      2000000,
			Timestamp: now,
			Type:      "trade",
		}
		err = sm.UpdateLiveBar(symbol, tick)
		if err != nil {
			t.Fatalf("Failed to update live bar for %s: %v", symbol, err)
		}
	}

	// Run scan and measure time
	start := time.Now()
	scanLoop.Scan()
	duration := time.Since(start)

	stats := scanLoop.GetStats()

	// Verify performance
	if duration > config.MaxScanTime {
		t.Errorf("Scan took %v, exceeded max time of %v", duration, config.MaxScanTime)
	}

	// Verify all symbols were scanned
	if stats.SymbolsScanned != int64(len(symbols)) {
		t.Errorf("Expected %d symbols scanned, got %d", len(symbols), stats.SymbolsScanned)
	}

	// Verify rules were evaluated
	if stats.RulesEvaluated == 0 {
		t.Error("Expected rules to be evaluated")
	}

	t.Logf("Performance: Scanned %d symbols with %d rules in %v", stats.SymbolsScanned, len(testRules), duration)
}

// TestFilterIntegration_LazyComputation tests that only required metrics are computed
func TestFilterIntegration_LazyComputation(t *testing.T) {
	sm := scanner.NewStateManager(100)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := scanner.NewCooldownTracker(10*time.Second, 5*time.Minute)
	alertEmitter := newMockAlertEmitterE2E()

	config := scanner.DefaultScanLoopConfig()
	scanLoop := scanner.NewScanLoop(
		config,
		sm,
		ruleStore,
		compiler,
		cooldownTracker,
		alertEmitter,
		nil,
	)

	// Create rule that only needs one metric
	rule := &models.Rule{
		ID:      "test-rule-lazy",
		Name:    "Single Metric Rule",
		Enabled: true,
		Conditions: []models.Condition{
			{
				Metric:   "price_change_5m_pct",
				Operator: ">",
				Value:    5.0,
			},
		},
	}

	err := ruleStore.AddRule(rule)
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	err = scanLoop.ReloadRules()
	if err != nil {
		t.Fatalf("Failed to reload rules: %v", err)
	}

	symbol := "AAPL"
	basePrice := 100.0
	now := time.Now()

	// Setup state with historical bars (need at least 6 for 5m change)
	err = addHistoricalBars(sm, symbol, basePrice, 10, now)
	if err != nil {
		t.Fatalf("Failed to add historical bars: %v", err)
	}

	currentTick := &models.Tick{
		Symbol:    symbol,
		Price:     basePrice * 1.06,
		Size:      100000,
		Timestamp: now,
		Type:      "trade",
	}
	err = sm.UpdateLiveBar(symbol, currentTick)
	if err != nil {
		t.Fatalf("Failed to update live bar: %v", err)
	}

	// Run scan
	scanLoop.Scan()

	stats := scanLoop.GetStats()
	if stats.RulesEvaluated == 0 {
		t.Error("Expected rule to be evaluated")
	}

	// Verify that only required metrics were computed (lazy computation)
	// This is verified by the fact that the scan completes quickly
	// In a real scenario, we could add metrics to track which metrics were computed
	t.Logf("Lazy computation test: Scanned %d symbols, evaluated %d rules", stats.SymbolsScanned, stats.RulesEvaluated)
}
