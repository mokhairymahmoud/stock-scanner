package scanner

import (
	"sync"
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/rules"
)

// mockCooldownTracker is a mock implementation of CooldownTracker for testing
type mockCooldownTracker struct {
	mu         sync.RWMutex
	cooldowns  map[string]time.Time
	cooldownSec int
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

// mockAlertEmitter is a mock implementation of AlertEmitter for testing
type mockAlertEmitter struct {
	mu     sync.RWMutex
	alerts []*models.Alert
	errors map[string]error // Map of ruleID|symbol to error (for testing error cases)
}

func newMockAlertEmitter() *mockAlertEmitter {
	return &mockAlertEmitter{
		alerts: make([]*models.Alert, 0),
		errors: make(map[string]error),
	}
}

func (m *mockAlertEmitter) EmitAlert(alert *models.Alert) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := alert.RuleID + "|" + alert.Symbol
	if err, exists := m.errors[key]; exists {
		return err
	}

	m.alerts = append(m.alerts, alert)
	return nil
}

func (m *mockAlertEmitter) GetAlerts() []*models.Alert {
	m.mu.RLock()
	defer m.mu.RUnlock()

	alerts := make([]*models.Alert, len(m.alerts))
	copy(alerts, m.alerts)
	return alerts
}

func (m *mockAlertEmitter) SetError(ruleID, symbol string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := ruleID + "|" + symbol
	m.errors[key] = err
}

func (m *mockAlertEmitter) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.alerts = m.alerts[:0]
	m.errors = make(map[string]error)
}

func TestScanLoop_NewScanLoop(t *testing.T) {
	sm := NewStateManager(10)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := newMockCooldownTracker()
	alertEmitter := newMockAlertEmitter()

	config := DefaultScanLoopConfig()

	// Test normal creation
	sl := NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, alertEmitter)
	if sl == nil {
		t.Fatal("Expected scan loop to be created")
	}

	if sl.stateManager != sm {
		t.Error("Expected state manager to be set")
	}

	// Test panic with nil state manager
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when state manager is nil")
		}
	}()

	NewScanLoop(config, nil, ruleStore, compiler, cooldownTracker, alertEmitter)
}

func TestScanLoop_ReloadRules(t *testing.T) {
	sm := NewStateManager(10)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := newMockCooldownTracker()
	alertEmitter := newMockAlertEmitter()

	config := DefaultScanLoopConfig()
	sl := NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, alertEmitter)

	// Add a rule
	rule := &models.Rule{
		ID:         "rule-1",
		Name:       "Test Rule",
		Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Cooldown:   300,
		Enabled:    true,
	}

	err := ruleStore.AddRule(rule)
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	// Reload rules
	err = sl.ReloadRules()
	if err != nil {
		t.Fatalf("Failed to reload rules: %v", err)
	}

	// Verify rules were compiled
	sl.rulesMu.RLock()
	compiledCount := len(sl.compiledRules)
	sl.rulesMu.RUnlock()

	if compiledCount != 1 {
		t.Errorf("Expected 1 compiled rule, got %d", compiledCount)
	}
}

func TestScanLoop_Scan_NoSymbols(t *testing.T) {
	sm := NewStateManager(10)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := newMockCooldownTracker()
	alertEmitter := newMockAlertEmitter()

	config := DefaultScanLoopConfig()
	sl := NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, alertEmitter)

	// Add a rule
	rule := &models.Rule{
		ID:         "rule-1",
		Name:       "Test Rule",
		Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Cooldown:   300,
		Enabled:    true,
	}

	ruleStore.AddRule(rule)
	sl.ReloadRules()

	// Scan with no symbols (should not panic)
	sl.Scan()

	stats := sl.GetStats()
	if stats.SymbolsScanned != 0 {
		t.Errorf("Expected 0 symbols scanned, got %d", stats.SymbolsScanned)
	}
}

func TestScanLoop_Scan_NoRules(t *testing.T) {
	sm := NewStateManager(10)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := newMockCooldownTracker()
	alertEmitter := newMockAlertEmitter()

	config := DefaultScanLoopConfig()
	sl := NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, alertEmitter)

	// Add a symbol but no rules
	tick := &models.Tick{
		Symbol:    "AAPL",
		Price:     150.0,
		Size:      100,
		Timestamp: time.Now(),
		Type:      "trade",
	}
	sm.UpdateLiveBar("AAPL", tick)

	// Scan with no rules (should not panic)
	sl.Scan()

	stats := sl.GetStats()
	if stats.RulesEvaluated != 0 {
		t.Errorf("Expected 0 rules evaluated, got %d", stats.RulesEvaluated)
	}
}

func TestScanLoop_Scan_RuleMatches(t *testing.T) {
	sm := NewStateManager(10)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := newMockCooldownTracker()
	alertEmitter := newMockAlertEmitter()

	config := DefaultScanLoopConfig()
	sl := NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, alertEmitter)

	// Add a rule that matches RSI < 30
	rule := &models.Rule{
		ID:         "rule-1",
		Name:       "RSI Oversold",
		Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Cooldown:   300,
		Enabled:    true,
	}

	ruleStore.AddRule(rule)
	sl.ReloadRules()

	// Add symbol with matching indicators
	indicators := map[string]float64{
		"rsi_14": 25.0, // Matches rule (< 30)
	}
	sm.UpdateIndicators("AAPL", indicators)

	// Add some state
	tick := &models.Tick{
		Symbol:    "AAPL",
		Price:     150.0,
		Size:      100,
		Timestamp: time.Now(),
		Type:      "trade",
	}
	sm.UpdateLiveBar("AAPL", tick)

	// Scan
	sl.Scan()

	// Verify alert was emitted
	alerts := alertEmitter.GetAlerts()
	if len(alerts) != 1 {
		t.Errorf("Expected 1 alert, got %d", len(alerts))
	}

	if alerts[0].Symbol != "AAPL" {
		t.Errorf("Expected alert for AAPL, got %s", alerts[0].Symbol)
	}

	if alerts[0].RuleID != "rule-1" {
		t.Errorf("Expected rule ID 'rule-1', got %s", alerts[0].RuleID)
	}

	// Verify stats
	stats := sl.GetStats()
	if stats.RulesMatched != 1 {
		t.Errorf("Expected 1 rule matched, got %d", stats.RulesMatched)
	}

	if stats.AlertsEmitted != 1 {
		t.Errorf("Expected 1 alert emitted, got %d", stats.AlertsEmitted)
	}
}

func TestScanLoop_Scan_RuleDoesNotMatch(t *testing.T) {
	sm := NewStateManager(10)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := newMockCooldownTracker()
	alertEmitter := newMockAlertEmitter()

	config := DefaultScanLoopConfig()
	sl := NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, alertEmitter)

	// Add a rule that doesn't match
	rule := &models.Rule{
		ID:         "rule-1",
		Name:       "RSI Oversold",
		Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Cooldown:   300,
		Enabled:    true,
	}

	ruleStore.AddRule(rule)
	sl.ReloadRules()

	// Add symbol with non-matching indicators
	indicators := map[string]float64{
		"rsi_14": 65.0, // Doesn't match rule (>= 30)
	}
	sm.UpdateIndicators("AAPL", indicators)

	tick := &models.Tick{
		Symbol:    "AAPL",
		Price:     150.0,
		Size:      100,
		Timestamp: time.Now(),
		Type:      "trade",
	}
	sm.UpdateLiveBar("AAPL", tick)

	// Scan
	sl.Scan()

	// Verify no alert was emitted
	alerts := alertEmitter.GetAlerts()
	if len(alerts) != 0 {
		t.Errorf("Expected 0 alerts, got %d", len(alerts))
	}

	// Verify stats
	stats := sl.GetStats()
	if stats.RulesMatched != 0 {
		t.Errorf("Expected 0 rules matched, got %d", stats.RulesMatched)
	}
}

func TestScanLoop_Scan_CooldownPreventsAlert(t *testing.T) {
	sm := NewStateManager(10)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := newMockCooldownTracker()
	alertEmitter := newMockAlertEmitter()

	config := DefaultScanLoopConfig()
	sl := NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, alertEmitter)

	// Add a rule
	rule := &models.Rule{
		ID:         "rule-1",
		Name:       "RSI Oversold",
		Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Cooldown:   300,
		Enabled:    true,
	}

	ruleStore.AddRule(rule)
	sl.ReloadRules()

	// Add symbol with matching indicators
	indicators := map[string]float64{
		"rsi_14": 25.0,
	}
	sm.UpdateIndicators("AAPL", indicators)

	tick := &models.Tick{
		Symbol:    "AAPL",
		Price:     150.0,
		Size:      100,
		Timestamp: time.Now(),
		Type:      "trade",
	}
	sm.UpdateLiveBar("AAPL", tick)

	// Set cooldown
	cooldownTracker.RecordCooldown("rule-1", "AAPL", 300)

	// Scan
	sl.Scan()

	// Verify no alert was emitted (cooldown prevents it)
	alerts := alertEmitter.GetAlerts()
	if len(alerts) != 0 {
		t.Errorf("Expected 0 alerts (cooldown), got %d", len(alerts))
	}
}

func TestScanLoop_Scan_MultipleSymbols(t *testing.T) {
	sm := NewStateManager(10)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := newMockCooldownTracker()
	alertEmitter := newMockAlertEmitter()

	config := DefaultScanLoopConfig()
	sl := NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, alertEmitter)

	// Add a rule
	rule := &models.Rule{
		ID:         "rule-1",
		Name:       "RSI Oversold",
		Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Cooldown:   300,
		Enabled:    true,
	}

	ruleStore.AddRule(rule)
	sl.ReloadRules()

	// Add multiple symbols
	symbols := []string{"AAPL", "GOOGL", "MSFT"}
	for _, symbol := range symbols {
		indicators := map[string]float64{
			"rsi_14": 25.0, // All match
		}
		sm.UpdateIndicators(symbol, indicators)

		tick := &models.Tick{
			Symbol:    symbol,
			Price:     150.0,
			Size:      100,
			Timestamp: time.Now(),
			Type:      "trade",
		}
		sm.UpdateLiveBar(symbol, tick)
	}

	// Scan
	sl.Scan()

	// Verify alerts for all symbols
	alerts := alertEmitter.GetAlerts()
	if len(alerts) != len(symbols) {
		t.Errorf("Expected %d alerts, got %d", len(symbols), len(alerts))
	}

	// Verify stats
	stats := sl.GetStats()
	if stats.SymbolsScanned != int64(len(symbols)) {
		t.Errorf("Expected %d symbols scanned, got %d", len(symbols), stats.SymbolsScanned)
	}
}

func TestScanLoop_Scan_MultipleRules(t *testing.T) {
	sm := NewStateManager(10)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := newMockCooldownTracker()
	alertEmitter := newMockAlertEmitter()

	config := DefaultScanLoopConfig()
	sl := NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, alertEmitter)

	// Add multiple rules
	rules := []*models.Rule{
		{
			ID:         "rule-1",
			Name:       "RSI Oversold",
			Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
			Cooldown:   300,
			Enabled:    true,
		},
		{
			ID:         "rule-2",
			Name:       "RSI Overbought",
			Conditions: []models.Condition{{Metric: "rsi_14", Operator: ">", Value: 70.0}},
			Cooldown:   300,
			Enabled:    true,
		},
	}

	for _, rule := range rules {
		ruleStore.AddRule(rule)
	}
	sl.ReloadRules()

	// Add symbol that matches first rule
	indicators := map[string]float64{
		"rsi_14": 25.0, // Matches rule-1, not rule-2
	}
	sm.UpdateIndicators("AAPL", indicators)

	tick := &models.Tick{
		Symbol:    "AAPL",
		Price:     150.0,
		Size:      100,
		Timestamp: time.Now(),
		Type:      "trade",
	}
	sm.UpdateLiveBar("AAPL", tick)

	// Scan
	sl.Scan()

	// Verify only one alert (rule-1 matches)
	alerts := alertEmitter.GetAlerts()
	if len(alerts) != 1 {
		t.Errorf("Expected 1 alert, got %d", len(alerts))
	}

	if alerts[0].RuleID != "rule-1" {
		t.Errorf("Expected rule ID 'rule-1', got %s", alerts[0].RuleID)
	}

	// Verify stats
	stats := sl.GetStats()
	if stats.RulesEvaluated != 2 { // Both rules evaluated
		t.Errorf("Expected 2 rules evaluated, got %d", stats.RulesEvaluated)
	}

	if stats.RulesMatched != 1 { // Only one matched
		t.Errorf("Expected 1 rule matched, got %d", stats.RulesMatched)
	}
}

func TestScanLoop_Scan_DisabledRule(t *testing.T) {
	sm := NewStateManager(10)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := newMockCooldownTracker()
	alertEmitter := newMockAlertEmitter()

	config := DefaultScanLoopConfig()
	sl := NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, alertEmitter)

	// Add a disabled rule
	rule := &models.Rule{
		ID:         "rule-1",
		Name:       "RSI Oversold",
		Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Cooldown:   300,
		Enabled:    false, // Disabled
	}

	ruleStore.AddRule(rule)
	sl.ReloadRules()

	// Add symbol with matching indicators
	indicators := map[string]float64{
		"rsi_14": 25.0,
	}
	sm.UpdateIndicators("AAPL", indicators)

	tick := &models.Tick{
		Symbol:    "AAPL",
		Price:     150.0,
		Size:      100,
		Timestamp: time.Now(),
		Type:      "trade",
	}
	sm.UpdateLiveBar("AAPL", tick)

	// Scan
	sl.Scan()

	// Verify no alert (rule is disabled)
	alerts := alertEmitter.GetAlerts()
	if len(alerts) != 0 {
		t.Errorf("Expected 0 alerts (rule disabled), got %d", len(alerts))
	}

	// Verify no rules were compiled
	sl.rulesMu.RLock()
	compiledCount := len(sl.compiledRules)
	sl.rulesMu.RUnlock()

	if compiledCount != 0 {
		t.Errorf("Expected 0 compiled rules, got %d", compiledCount)
	}
}

func TestScanLoop_Scan_Performance(t *testing.T) {
	sm := NewStateManager(10)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := newMockCooldownTracker()
	alertEmitter := newMockAlertEmitter()

	config := DefaultScanLoopConfig()
	sl := NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, alertEmitter)

	// Add a rule
	rule := &models.Rule{
		ID:         "rule-1",
		Name:       "Test Rule",
		Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Cooldown:   300,
		Enabled:    true,
	}

	ruleStore.AddRule(rule)
	sl.ReloadRules()

	// Add multiple symbols
	for i := 0; i < 100; i++ {
		symbol := "SYMBOL" + string(rune('A'+i%26)) + string(rune('0'+i/26))
		indicators := map[string]float64{
			"rsi_14": 25.0,
		}
		sm.UpdateIndicators(symbol, indicators)

		tick := &models.Tick{
			Symbol:    symbol,
			Price:     150.0,
			Size:      100,
			Timestamp: time.Now(),
			Type:      "trade",
		}
		sm.UpdateLiveBar(symbol, tick)
	}

	// Scan and measure time
	start := time.Now()
	sl.Scan()
	scanTime := time.Since(start)

	// Verify scan completed quickly (should be well under 800ms for 100 symbols)
	if scanTime > 800*time.Millisecond {
		t.Errorf("Scan took too long: %v (target: <800ms)", scanTime)
	}

	stats := sl.GetStats()
	if stats.ScanCycleTime > 800*time.Millisecond {
		t.Errorf("Scan cycle time too high: %v (target: <800ms)", stats.ScanCycleTime)
	}
}

func TestScanLoop_GetMetricsFromSnapshot(t *testing.T) {
	sm := NewStateManager(10)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := newMockCooldownTracker()
	alertEmitter := newMockAlertEmitter()

	config := DefaultScanLoopConfig()
	sl := NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, alertEmitter)

	// Add finalized bars
	for i := 0; i < 5; i++ {
		bar := &models.Bar1m{
			Symbol:    "AAPL",
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			Open:      float64(150 + i),
			High:      float64(152 + i),
			Low:       float64(149 + i),
			Close:     float64(151 + i),
			Volume:    1000,
			VWAP:      150.5 + float64(i),
		}
		sm.UpdateFinalizedBar(bar)
	}

	// Add indicators
	indicators := map[string]float64{
		"rsi_14": 65.5,
		"ema_20": 150.2,
	}
	sm.UpdateIndicators("AAPL", indicators)

	// Add live bar
	tick := &models.Tick{
		Symbol:    "AAPL",
		Price:     156.0,
		Size:      200,
		Timestamp: time.Now(),
		Type:      "trade",
	}
	sm.UpdateLiveBar("AAPL", tick)

	// Get snapshot
	snapshot := sm.Snapshot()
	symbolSnapshot := snapshot.States["AAPL"]

	// Get metrics
	metrics := sl.getMetricsFromSnapshot(symbolSnapshot)

	// Verify metrics
	if metrics["rsi_14"] != 65.5 {
		t.Errorf("Expected rsi_14 = 65.5, got %f", metrics["rsi_14"])
	}

	if metrics["price"] != 156.0 {
		t.Errorf("Expected price = 156.0, got %f", metrics["price"])
	}

	if metrics["close"] != 155.0 {
		t.Errorf("Expected close = 155.0, got %f", metrics["close"])
	}

	// Verify price change is computed
	if metrics["price_change_1m_pct"] == 0 {
		t.Error("Expected price_change_1m_pct to be computed")
	}

	// Return to pool
	sl.returnMetricsToPool(metrics)
}

func TestScanLoop_IsRunning(t *testing.T) {
	sm := NewStateManager(10)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := newMockCooldownTracker()
	alertEmitter := newMockAlertEmitter()

	config := DefaultScanLoopConfig()
	sl := NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, alertEmitter)

	if sl.IsRunning() {
		t.Error("Expected scan loop not to be running initially")
	}
}

func TestScanLoop_GetStats(t *testing.T) {
	sm := NewStateManager(10)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := newMockCooldownTracker()
	alertEmitter := newMockAlertEmitter()

	config := DefaultScanLoopConfig()
	sl := NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, alertEmitter)

	stats := sl.GetStats()
	if stats.ScanCycles != 0 {
		t.Errorf("Expected 0 scan cycles, got %d", stats.ScanCycles)
	}
}

func TestScanLoop_Scan_NilCooldownTracker(t *testing.T) {
	sm := NewStateManager(10)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	alertEmitter := newMockAlertEmitter()

	config := DefaultScanLoopConfig()
	sl := NewScanLoop(config, sm, ruleStore, compiler, nil, alertEmitter) // nil cooldown tracker

	// Add a rule
	rule := &models.Rule{
		ID:         "rule-1",
		Name:       "RSI Oversold",
		Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Cooldown:   300,
		Enabled:    true,
	}

	ruleStore.AddRule(rule)
	sl.ReloadRules()

	// Add symbol with matching indicators
	indicators := map[string]float64{
		"rsi_14": 25.0,
	}
	sm.UpdateIndicators("AAPL", indicators)

	tick := &models.Tick{
		Symbol:    "AAPL",
		Price:     150.0,
		Size:      100,
		Timestamp: time.Now(),
		Type:      "trade",
	}
	sm.UpdateLiveBar("AAPL", tick)

	// Scan (should work without cooldown tracker)
	sl.Scan()

	// Verify alert was emitted (no cooldown check)
	alerts := alertEmitter.GetAlerts()
	if len(alerts) != 1 {
		t.Errorf("Expected 1 alert, got %d", len(alerts))
	}
}

func TestScanLoop_Scan_NilAlertEmitter(t *testing.T) {
	sm := NewStateManager(10)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := newMockCooldownTracker()

	config := DefaultScanLoopConfig()
	sl := NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, nil) // nil alert emitter

	// Add a rule
	rule := &models.Rule{
		ID:         "rule-1",
		Name:       "RSI Oversold",
		Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Cooldown:   300,
		Enabled:    true,
	}

	ruleStore.AddRule(rule)
	sl.ReloadRules()

	// Add symbol with matching indicators
	indicators := map[string]float64{
		"rsi_14": 25.0,
	}
	sm.UpdateIndicators("AAPL", indicators)

	tick := &models.Tick{
		Symbol:    "AAPL",
		Price:     150.0,
		Size:      100,
		Timestamp: time.Now(),
		Type:      "trade",
	}
	sm.UpdateLiveBar("AAPL", tick)

	// Scan (should not panic without alert emitter)
	sl.Scan()

	// Verify stats show rule matched but no alert emitted
	stats := sl.GetStats()
	if stats.RulesMatched != 1 {
		t.Errorf("Expected 1 rule matched, got %d", stats.RulesMatched)
	}

	if stats.AlertsEmitted != 0 {
		t.Errorf("Expected 0 alerts emitted (no emitter), got %d", stats.AlertsEmitted)
	}
}

func TestScanLoop_Scan_MetricsPoolReuse(t *testing.T) {
	sm := NewStateManager(10)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := newMockCooldownTracker()
	alertEmitter := newMockAlertEmitter()

	config := DefaultScanLoopConfig()
	sl := NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, alertEmitter)

	// Add a rule
	rule := &models.Rule{
		ID:         "rule-1",
		Name:       "Test Rule",
		Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Cooldown:   300,
		Enabled:    true,
	}

	ruleStore.AddRule(rule)
	sl.ReloadRules()

	// Add multiple symbols
	for i := 0; i < 10; i++ {
		symbol := "SYMBOL" + string(rune('A'+i))
		indicators := map[string]float64{
			"rsi_14": 25.0,
		}
		sm.UpdateIndicators(symbol, indicators)

		tick := &models.Tick{
			Symbol:    symbol,
			Price:     150.0,
			Size:      100,
			Timestamp: time.Now(),
			Type:      "trade",
		}
		sm.UpdateLiveBar(symbol, tick)
	}

	// Scan multiple times to test pool reuse
	for i := 0; i < 5; i++ {
		sl.Scan()
	}

	// Verify no panics and stats are correct
	stats := sl.GetStats()
	if stats.ScanCycles != 5 {
		t.Errorf("Expected 5 scan cycles, got %d", stats.ScanCycles)
	}
}

func TestScanLoop_Scan_ComplexRule(t *testing.T) {
	sm := NewStateManager(10)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := newMockCooldownTracker()
	alertEmitter := newMockAlertEmitter()

	config := DefaultScanLoopConfig()
	sl := NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, alertEmitter)

	// Add a rule with multiple conditions (AND logic)
	rule := &models.Rule{
		ID:   "rule-1",
		Name: "Complex Rule",
		Conditions: []models.Condition{
			{Metric: "rsi_14", Operator: "<", Value: 30.0},
			{Metric: "price_change_5m_pct", Operator: ">", Value: 1.0},
		},
		Cooldown: 300,
		Enabled:  true,
	}

	ruleStore.AddRule(rule)
	sl.ReloadRules()

	// Add finalized bars for price change calculation
	for i := 0; i < 6; i++ {
		bar := &models.Bar1m{
			Symbol:    "AAPL",
			Timestamp: time.Now().Add(time.Duration(i-5) * time.Minute),
			Open:      float64(150 + i),
			High:      float64(152 + i),
			Low:       float64(149 + i),
			Close:     float64(151 + i),
			Volume:    1000,
			VWAP:      150.5 + float64(i),
		}
		sm.UpdateFinalizedBar(bar)
	}

	// Add indicators (matching first condition)
	indicators := map[string]float64{
		"rsi_14": 25.0, // < 30, matches
	}
	sm.UpdateIndicators("AAPL", indicators)

	tick := &models.Tick{
		Symbol:    "AAPL",
		Price:     160.0, // Price change > 1% (151 -> 160 = ~6%)
		Size:      100,
		Timestamp: time.Now(),
		Type:      "trade",
	}
	sm.UpdateLiveBar("AAPL", tick)

	// Scan
	sl.Scan()

	// Verify alert was emitted (both conditions match)
	alerts := alertEmitter.GetAlerts()
	if len(alerts) != 1 {
		t.Errorf("Expected 1 alert, got %d", len(alerts))
	}
}

func TestScanLoop_Scan_ComplexRulePartialMatch(t *testing.T) {
	sm := NewStateManager(10)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := newMockCooldownTracker()
	alertEmitter := newMockAlertEmitter()

	config := DefaultScanLoopConfig()
	sl := NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, alertEmitter)

	// Add a rule with multiple conditions
	rule := &models.Rule{
		ID:   "rule-1",
		Name: "Complex Rule",
		Conditions: []models.Condition{
			{Metric: "rsi_14", Operator: "<", Value: 30.0},
			{Metric: "price_change_5m_pct", Operator: ">", Value: 1.0},
		},
		Cooldown: 300,
		Enabled:  true,
	}

	ruleStore.AddRule(rule)
	sl.ReloadRules()

	// Add finalized bars - ensure price change 5m is < 1%
	// Bar 0 (5 minutes ago): close = 150.0
	// Bar 5 (current): close = 150.5 (0.5/150 = 0.33% change, < 1%)
	for i := 0; i < 6; i++ {
		bar := &models.Bar1m{
			Symbol:    "AAPL",
			Timestamp: time.Now().Add(time.Duration(i-5) * time.Minute),
			Open:      150.0,
			High:      152.0,
			Low:       149.0,
			Close:     150.0 + float64(i)*0.1, // Small increments: 150.0, 150.1, 150.2, 150.3, 150.4, 150.5
			Volume:    1000,
			VWAP:      150.5,
		}
		sm.UpdateFinalizedBar(bar)
	}

	// Add indicators (matching first condition)
	indicators := map[string]float64{
		"rsi_14": 25.0, // < 30, matches
	}
	sm.UpdateIndicators("AAPL", indicators)

	tick := &models.Tick{
		Symbol:    "AAPL",
		Price:     150.5, // Same as last finalized bar close
		Size:      100,
		Timestamp: time.Now(),
		Type:      "trade",
	}
	sm.UpdateLiveBar("AAPL", tick)

	// Scan
	sl.Scan()

	// Verify no alert (second condition doesn't match)
	alerts := alertEmitter.GetAlerts()
	if len(alerts) != 0 {
		t.Errorf("Expected 0 alerts (partial match), got %d", len(alerts))
	}
}

func TestScanLoop_Scan_StatsTracking(t *testing.T) {
	sm := NewStateManager(10)
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	cooldownTracker := newMockCooldownTracker()
	alertEmitter := newMockAlertEmitter()

	config := DefaultScanLoopConfig()
	sl := NewScanLoop(config, sm, ruleStore, compiler, cooldownTracker, alertEmitter)

	// Add a rule
	rule := &models.Rule{
		ID:         "rule-1",
		Name:       "Test Rule",
		Conditions: []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Cooldown:   300,
		Enabled:    true,
	}

	ruleStore.AddRule(rule)
	sl.ReloadRules()

	// Add symbol
	indicators := map[string]float64{
		"rsi_14": 25.0,
	}
	sm.UpdateIndicators("AAPL", indicators)

	tick := &models.Tick{
		Symbol:    "AAPL",
		Price:     150.0,
		Size:      100,
		Timestamp: time.Now(),
		Type:      "trade",
	}
	sm.UpdateLiveBar("AAPL", tick)

	// Run multiple scans
	for i := 0; i < 3; i++ {
		sl.Scan()
	}

	// Verify stats
	stats := sl.GetStats()
	if stats.ScanCycles != 3 {
		t.Errorf("Expected 3 scan cycles, got %d", stats.ScanCycles)
	}

	if stats.SymbolsScanned != 3 {
		t.Errorf("Expected 3 symbols scanned, got %d", stats.SymbolsScanned)
	}

	if stats.RulesEvaluated != 3 {
		t.Errorf("Expected 3 rules evaluated, got %d", stats.RulesEvaluated)
	}

	if stats.RulesMatched != 3 {
		t.Errorf("Expected 3 rules matched, got %d", stats.RulesMatched)
	}

	if stats.AlertsEmitted != 1 { // Only first one, rest on cooldown
		t.Errorf("Expected 1 alert emitted (cooldown), got %d", stats.AlertsEmitted)
	}

	// Verify scan cycle time tracking
	if stats.ScanCycleTime == 0 {
		t.Error("Expected scan cycle time to be tracked")
	}

	if stats.MaxScanCycleTime == 0 {
		t.Error("Expected max scan cycle time to be tracked")
	}

	if stats.AvgScanCycleTime == 0 {
		t.Error("Expected avg scan cycle time to be tracked")
	}
}


