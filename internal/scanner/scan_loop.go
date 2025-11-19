package scanner

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/rules"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

// CooldownTracker defines the interface for checking and managing rule cooldowns
// This will be fully implemented in Phase 3.2.6
type CooldownTracker interface {
	// IsOnCooldown checks if a rule is on cooldown for a symbol
	IsOnCooldown(ruleID, symbol string) bool
	// RecordCooldown records that a rule fired for a symbol (starts cooldown)
	RecordCooldown(ruleID, symbol string, cooldownSeconds int)
}

// AlertEmitter defines the interface for emitting alerts
// This will be fully implemented in Phase 3.2.7
type AlertEmitter interface {
	// EmitAlert emits an alert for a matched rule
	EmitAlert(alert *models.Alert) error
}

// ScanLoopConfig holds configuration for the scan loop
type ScanLoopConfig struct {
	ScanInterval    time.Duration // How often to run scan (default: 1 second)
	MaxScanTime     time.Duration // Maximum time allowed for a scan cycle (default: 800ms)
	MetricsPoolSize int           // Size of metrics map pool (default: 100)
}

// DefaultScanLoopConfig returns default configuration
func DefaultScanLoopConfig() ScanLoopConfig {
	return ScanLoopConfig{
		ScanInterval:    1 * time.Second,
		MaxScanTime:     800 * time.Millisecond,
		MetricsPoolSize: 100,
	}
}

// ScanLoop is the core scanning engine that evaluates rules against symbol state
type ScanLoop struct {
	config         ScanLoopConfig
	stateManager   *StateManager
	ruleStore      rules.RuleStore
	compiler       *rules.Compiler
	cooldownTracker CooldownTracker
	alertEmitter   AlertEmitter
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	mu             sync.RWMutex
	running        bool
	stats          ScanLoopStats

	// Performance optimization: pool for metrics maps
	metricsPool *sync.Pool

	// Compiled rules cache (updated when rules change)
	compiledRules map[string]rules.CompiledRule
	rulesMu       sync.RWMutex
}

// ScanLoopStats holds statistics about the scan loop
type ScanLoopStats struct {
	ScanCycles        int64
	SymbolsScanned    int64
	RulesEvaluated    int64
	RulesMatched      int64
	AlertsEmitted     int64
	ScanCycleTime     time.Duration // Last scan cycle time
	MaxScanCycleTime  time.Duration // Maximum scan cycle time observed
	MinScanCycleTime  time.Duration // Minimum scan cycle time observed
	AvgScanCycleTime  time.Duration // Average scan cycle time
	ScanCycleTimeSum  time.Duration // Sum of all scan cycle times (for average calculation)
	mu                sync.RWMutex
}

// NewScanLoop creates a new scan loop
func NewScanLoop(
	config ScanLoopConfig,
	stateManager *StateManager,
	ruleStore rules.RuleStore,
	compiler *rules.Compiler,
	cooldownTracker CooldownTracker,
	alertEmitter AlertEmitter,
) *ScanLoop {
	if stateManager == nil {
		panic("stateManager cannot be nil")
	}
	if ruleStore == nil {
		panic("ruleStore cannot be nil")
	}
	if compiler == nil {
		panic("compiler cannot be nil")
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Initialize metrics pool for performance
	metricsPool := &sync.Pool{
		New: func() interface{} {
			return make(map[string]float64, 50) // Pre-allocate capacity
		},
	}

	return &ScanLoop{
		config:         config,
		stateManager:   stateManager,
		ruleStore:      ruleStore,
		compiler:       compiler,
		cooldownTracker: cooldownTracker,
		alertEmitter:   alertEmitter,
		ctx:            ctx,
		cancel:         cancel,
		metricsPool:    metricsPool,
		compiledRules:  make(map[string]rules.CompiledRule),
		stats: ScanLoopStats{
			MinScanCycleTime: time.Hour, // Initialize to large value
		},
	}
}

// Start starts the scan loop
func (sl *ScanLoop) Start() error {
	sl.mu.Lock()
	if sl.running {
		sl.mu.Unlock()
		return fmt.Errorf("scan loop is already running")
	}
	sl.running = true
	sl.mu.Unlock()

	// Load and compile initial rules
	if err := sl.reloadRules(); err != nil {
		return fmt.Errorf("failed to load initial rules: %w", err)
	}

	logger.Info("Starting scan loop",
		logger.Duration("scan_interval", sl.config.ScanInterval),
		logger.Duration("max_scan_time", sl.config.MaxScanTime),
	)

	sl.wg.Add(1)
	go sl.run()

	return nil
}

// Stop stops the scan loop
func (sl *ScanLoop) Stop() {
	sl.mu.Lock()
	if !sl.running {
		sl.mu.Unlock()
		return
	}
	sl.running = false
	sl.mu.Unlock()

	logger.Info("Stopping scan loop")
	sl.cancel()
	sl.wg.Wait()
	logger.Info("Scan loop stopped")
}

// IsRunning returns whether the scan loop is running
func (sl *ScanLoop) IsRunning() bool {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return sl.running
}

// GetStats returns current scan loop statistics
func (sl *ScanLoop) GetStats() ScanLoopStats {
	sl.stats.mu.RLock()
	defer sl.stats.mu.RUnlock()

	// Calculate average
	avgTime := time.Duration(0)
	if sl.stats.ScanCycles > 0 {
		avgTime = sl.stats.ScanCycleTimeSum / time.Duration(sl.stats.ScanCycles)
	}

	// Return a copy
	return ScanLoopStats{
		ScanCycles:       sl.stats.ScanCycles,
		SymbolsScanned:    sl.stats.SymbolsScanned,
		RulesEvaluated:   sl.stats.RulesEvaluated,
		RulesMatched:     sl.stats.RulesMatched,
		AlertsEmitted:    sl.stats.AlertsEmitted,
		ScanCycleTime:    sl.stats.ScanCycleTime,
		MaxScanCycleTime: sl.stats.MaxScanCycleTime,
		MinScanCycleTime: sl.stats.MinScanCycleTime,
		AvgScanCycleTime: avgTime,
	}
}

// ReloadRules reloads and recompiles rules from the rule store
func (sl *ScanLoop) ReloadRules() error {
	return sl.reloadRules()
}

// run is the main scan loop
func (sl *ScanLoop) run() {
	defer sl.wg.Done()

	ticker := time.NewTicker(sl.config.ScanInterval)
	defer ticker.Stop()

	// Run initial scan immediately
	sl.Scan()

	for {
		select {
		case <-sl.ctx.Done():
			return
		case <-ticker.C:
			sl.Scan()
		}
	}
}

// Scan performs a single scan cycle (exported for testing)
func (sl *ScanLoop) Scan() {
	startTime := time.Now()

	// Check if scan takes too long
	defer func() {
		scanTime := time.Since(startTime)
		sl.updateStats(scanTime)

		if scanTime > sl.config.MaxScanTime {
			logger.Warn("Scan cycle exceeded max time",
				logger.Duration("scan_time", scanTime),
				logger.Duration("max_time", sl.config.MaxScanTime),
			)
		}
	}()

	// Get snapshot of all symbol states (lock-free)
	snapshot := sl.stateManager.Snapshot()
	if len(snapshot.Symbols) == 0 {
		return // No symbols to scan
	}

	// Get compiled rules (read lock)
	sl.rulesMu.RLock()
	compiledRules := sl.compiledRules
	sl.rulesMu.RUnlock()

	if len(compiledRules) == 0 {
		return // No rules to evaluate
	}

	// Scan each symbol
	symbolsScanned := int64(0)
	rulesEvaluated := int64(0)
	rulesMatched := int64(0)
	alertsEmitted := int64(0)

	for _, symbol := range snapshot.Symbols {
		symbolState := snapshot.States[symbol]
		if symbolState == nil {
			continue
		}

		symbolsScanned++

		// Get metrics for this symbol (computed from snapshot, no lock needed)
		metrics := sl.getMetricsFromSnapshot(symbolState)

		// Evaluate each rule
		for ruleID, compiledRule := range compiledRules {
			rulesEvaluated++

			// Evaluate rule
			matched, err := compiledRule(symbol, metrics)
			if err != nil {
				logger.Error("Failed to evaluate rule",
					logger.ErrorField(err),
					logger.String("rule_id", ruleID),
					logger.String("symbol", symbol),
				)
				continue
			}

			if !matched {
				continue // Rule didn't match, move to next rule
			}

			rulesMatched++

			// Check cooldown
			if sl.cooldownTracker != nil && sl.cooldownTracker.IsOnCooldown(ruleID, symbol) {
				logger.Debug("Rule on cooldown, skipping alert",
					logger.String("rule_id", ruleID),
					logger.String("symbol", symbol),
				)
				continue
			}

			// Get rule details for alert
			rule, err := sl.ruleStore.GetRule(ruleID)
			if err != nil {
				logger.Error("Failed to get rule details",
					logger.ErrorField(err),
					logger.String("rule_id", ruleID),
				)
				continue
			}

			// Emit alert
			if sl.alertEmitter != nil {
				alert := sl.createAlert(rule, symbol, metrics, symbolState)
				if err := sl.alertEmitter.EmitAlert(alert); err != nil {
					logger.Error("Failed to emit alert",
						logger.ErrorField(err),
						logger.String("rule_id", ruleID),
						logger.String("symbol", symbol),
					)
					continue
				}

				alertsEmitted++

				// Record cooldown
				if sl.cooldownTracker != nil && rule.Cooldown > 0 {
					sl.cooldownTracker.RecordCooldown(ruleID, symbol, rule.Cooldown)
				}
			}
		}

		// Return metrics map to pool
		sl.returnMetricsToPool(metrics)
	}

	// Update statistics
	atomic.AddInt64(&sl.stats.SymbolsScanned, symbolsScanned)
	atomic.AddInt64(&sl.stats.RulesEvaluated, rulesEvaluated)
	atomic.AddInt64(&sl.stats.RulesMatched, rulesMatched)
	atomic.AddInt64(&sl.stats.AlertsEmitted, alertsEmitted)
}

// getMetricsFromSnapshot computes metrics from a symbol state snapshot
// Returns a map that should be returned to pool after use
func (sl *ScanLoop) getMetricsFromSnapshot(snapshot *SymbolStateSnapshot) map[string]float64 {
	// Get metrics map from pool
	metrics := sl.metricsPool.Get().(map[string]float64)

	// Clear map (but keep capacity)
	for k := range metrics {
		delete(metrics, k)
	}

	// Copy indicators
	for key, value := range snapshot.Indicators {
		metrics[key] = value
	}

	// Add computed metrics from live bar
	if snapshot.LiveBar != nil {
		metrics["price"] = snapshot.LiveBar.Close

		// VWAP from live bar
		if snapshot.LiveBar.VWAPDenom > 0 {
			metrics["vwap_live"] = snapshot.LiveBar.VWAPNum / snapshot.LiveBar.VWAPDenom
		}

		// Volume from live bar
		metrics["volume_live"] = float64(snapshot.LiveBar.Volume)
	}

	// Add metrics from last finalized bar if available
	if len(snapshot.LastFinalBars) > 0 {
		lastBar := snapshot.LastFinalBars[len(snapshot.LastFinalBars)-1]
		metrics["close"] = lastBar.Close
		metrics["open"] = lastBar.Open
		metrics["high"] = lastBar.High
		metrics["low"] = lastBar.Low
		metrics["volume"] = float64(lastBar.Volume)
		metrics["vwap"] = lastBar.VWAP
	}

	// Compute price change metrics from finalized bars
	if len(snapshot.LastFinalBars) >= 2 {
		currentBar := snapshot.LastFinalBars[len(snapshot.LastFinalBars)-1]
		prevBar := snapshot.LastFinalBars[len(snapshot.LastFinalBars)-2]

		if prevBar.Close > 0 {
			changePct := ((currentBar.Close - prevBar.Close) / prevBar.Close) * 100.0
			metrics["price_change_1m_pct"] = changePct
		}
	}

	// Compute price change over 5 minutes
	if len(snapshot.LastFinalBars) >= 6 {
		currentBar := snapshot.LastFinalBars[len(snapshot.LastFinalBars)-1]
		bar5m := snapshot.LastFinalBars[len(snapshot.LastFinalBars)-6]

		if bar5m.Close > 0 {
			changePct := ((currentBar.Close - bar5m.Close) / bar5m.Close) * 100.0
			metrics["price_change_5m_pct"] = changePct
		}
	}

	// Compute price change over 15 minutes
	if len(snapshot.LastFinalBars) >= 16 {
		currentBar := snapshot.LastFinalBars[len(snapshot.LastFinalBars)-1]
		bar15m := snapshot.LastFinalBars[len(snapshot.LastFinalBars)-16]

		if bar15m.Close > 0 {
			changePct := ((currentBar.Close - bar15m.Close) / bar15m.Close) * 100.0
			metrics["price_change_15m_pct"] = changePct
		}
	}

	return metrics
}

// returnMetricsToPool returns a metrics map to the pool
func (sl *ScanLoop) returnMetricsToPool(metrics map[string]float64) {
	// Clear the map before returning to pool
	for k := range metrics {
		delete(metrics, k)
	}
	sl.metricsPool.Put(metrics)
}

// reloadRules reloads rules from store and recompiles them
func (sl *ScanLoop) reloadRules() error {
	// Get all rules and filter enabled ones
	allRules, err := sl.ruleStore.GetAllRules()
	if err != nil {
		return fmt.Errorf("failed to get rules: %w", err)
	}

	// Filter enabled rules
	enabledRules := make([]*models.Rule, 0, len(allRules))
	for _, rule := range allRules {
		if rule.Enabled {
			enabledRules = append(enabledRules, rule)
		}
	}

	// Compile rules
	compiled, err := sl.compiler.CompileEnabledRules(enabledRules)
	if err != nil {
		return fmt.Errorf("failed to compile rules: %w", err)
	}

	// Update compiled rules cache (write lock)
	sl.rulesMu.Lock()
	sl.compiledRules = compiled
	sl.rulesMu.Unlock()

	logger.Info("Reloaded and compiled rules",
		logger.Int("rule_count", len(compiled)),
	)

	return nil
}

// createAlert creates an alert from a matched rule
func (sl *ScanLoop) createAlert(
	rule *models.Rule,
	symbol string,
	metrics map[string]float64,
	snapshot *SymbolStateSnapshot,
) *models.Alert {
	// Get current price
	price := 0.0
	if snapshot.LiveBar != nil {
		price = snapshot.LiveBar.Close
	} else if len(snapshot.LastFinalBars) > 0 {
		price = snapshot.LastFinalBars[len(snapshot.LastFinalBars)-1].Close
	}

	// Generate alert ID (simple UUID-like, will be improved in Phase 3.2.7)
	alertID := fmt.Sprintf("%s-%s-%d", rule.ID, symbol, time.Now().UnixNano())

	// Create alert message
	message := fmt.Sprintf("Rule '%s' matched for %s", rule.Name, symbol)

	// Create alert
	alert := &models.Alert{
		ID:        alertID,
		RuleID:    rule.ID,
		RuleName:  rule.Name,
		Symbol:    symbol,
		Timestamp: time.Now(),
		Price:     price,
		Message:   message,
		Metadata: map[string]interface{}{
			"metrics": metrics,
		},
	}

	return alert
}

// updateStats updates scan loop statistics
func (sl *ScanLoop) updateStats(scanTime time.Duration) {
	sl.stats.mu.Lock()
	defer sl.stats.mu.Unlock()

	sl.stats.ScanCycles++
	sl.stats.ScanCycleTime = scanTime
	sl.stats.ScanCycleTimeSum += scanTime

	if scanTime > sl.stats.MaxScanCycleTime {
		sl.stats.MaxScanCycleTime = scanTime
	}

	if scanTime < sl.stats.MinScanCycleTime {
		sl.stats.MinScanCycleTime = scanTime
	}
}

