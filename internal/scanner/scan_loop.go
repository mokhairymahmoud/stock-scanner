package scanner

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/metrics"
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
	ScanInterval       time.Duration // How often to run scan (default: 1 second)
	MaxScanTime        time.Duration // Maximum time allowed for a scan cycle (default: 800ms)
	MetricsPoolSize    int           // Size of metrics map pool (default: 100)
	RuleReloadInterval time.Duration // How often to reload rules from store (default: 30 seconds)
}

// DefaultScanLoopConfig returns default configuration
func DefaultScanLoopConfig() ScanLoopConfig {
	return ScanLoopConfig{
		ScanInterval:       1 * time.Second,
		MaxScanTime:        800 * time.Millisecond,
		MetricsPoolSize:    100,
		RuleReloadInterval: 30 * time.Second,
	}
}

// ScanLoop is the core scanning engine that evaluates rules against symbol state
type ScanLoop struct {
	config             ScanLoopConfig
	stateManager       *StateManager
	ruleStore          rules.RuleStore
	compiler           *rules.Compiler
	cooldownTracker    CooldownTracker
	alertEmitter       AlertEmitter
	toplistIntegration *ToplistIntegration // Optional toplist integration
	ctx                context.Context
	cancel             context.CancelFunc
	wg                 sync.WaitGroup
	mu                 sync.RWMutex
	running            bool
	stats              ScanLoopStats

	// Performance optimization: pool for metrics maps
	metricsPool *sync.Pool

	// Metric registry for computing metrics
	metricRegistry *metrics.Registry

	// Compiled rules cache (updated when rules change)
	compiledRules map[string]rules.CompiledRule
	rulesMu       sync.RWMutex

	// Required metrics for all active rules (for lazy computation)
	requiredMetrics map[string]bool
	requiredMetricsMu sync.RWMutex

	// Rule reload tracking
	lastRuleReload time.Time
	lastReloadMu   sync.RWMutex
}

// ScanLoopStats holds statistics about the scan loop
type ScanLoopStats struct {
	ScanCycles       int64
	SymbolsScanned   int64
	RulesEvaluated   int64
	RulesMatched     int64
	AlertsEmitted    int64
	ScanCycleTime    time.Duration // Last scan cycle time
	MaxScanCycleTime time.Duration // Maximum scan cycle time observed
	MinScanCycleTime time.Duration // Minimum scan cycle time observed
	AvgScanCycleTime time.Duration // Average scan cycle time
	ScanCycleTimeSum time.Duration // Sum of all scan cycle times (for average calculation)
	mu               sync.RWMutex
}

// NewScanLoop creates a new scan loop
func NewScanLoop(
	config ScanLoopConfig,
	stateManager *StateManager,
	ruleStore rules.RuleStore,
	compiler *rules.Compiler,
	cooldownTracker CooldownTracker,
	alertEmitter AlertEmitter,
	toplistIntegration *ToplistIntegration, // Optional
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

	// Initialize metric registry
	metricRegistry := metrics.NewRegistry()

	return &ScanLoop{
		config:             config,
		stateManager:       stateManager,
		ruleStore:          ruleStore,
		compiler:           compiler,
		cooldownTracker:    cooldownTracker,
		alertEmitter:       alertEmitter,
		toplistIntegration: toplistIntegration,
		ctx:                ctx,
		cancel:             cancel,
		metricsPool:        metricsPool,
		metricRegistry:     metricRegistry,
		compiledRules:      make(map[string]rules.CompiledRule),
		requiredMetrics:    make(map[string]bool),
		lastRuleReload:     time.Now(),
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
		SymbolsScanned:   sl.stats.SymbolsScanned,
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

	scanTicker := time.NewTicker(sl.config.ScanInterval)
	defer scanTicker.Stop()

	// Create rule reload ticker if interval is configured
	var ruleReloadTicker *time.Ticker
	var ruleReloadChan <-chan time.Time
	if sl.config.RuleReloadInterval > 0 {
		ruleReloadTicker = time.NewTicker(sl.config.RuleReloadInterval)
		defer ruleReloadTicker.Stop()
		ruleReloadChan = ruleReloadTicker.C
	}

	// Run initial scan immediately
	sl.Scan()

	for {
		select {
		case <-sl.ctx.Done():
			return
		case <-scanTicker.C:
			sl.Scan()
		case <-ruleReloadChan:
			// Periodically reload rules from store
			if err := sl.reloadRules(); err != nil {
				logger.Error("Failed to reload rules during periodic refresh",
					logger.ErrorField(err),
				)
			}
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
		// Only compute metrics that are actually needed by active rules
		metrics := sl.getMetricsFromSnapshot(symbolState, sl.getRequiredMetrics())

		// Get current session for this symbol (as string to avoid import cycle)
		currentSession := string(symbolState.CurrentSession)

		// Evaluate each rule (if any rules exist)
		for ruleID, compiledRule := range compiledRules {
			rulesEvaluated++

			// Get rule details for filter configuration checks
			rule, err := sl.ruleStore.GetRule(ruleID)
			if err != nil {
				logger.Error("Failed to get rule details for filter checks",
					logger.ErrorField(err),
					logger.String("rule_id", ruleID),
				)
				continue
			}

			// Pre-filter: Check volume threshold and session for all conditions
			shouldEvaluate := sl.shouldEvaluateRule(rule, metrics, currentSession)
			if !shouldEvaluate {
				continue // Pre-filter failed, skip rule evaluation
			}

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

			// Rule already retrieved above for filter checks, reuse it

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

				// Record cooldown (using global cooldown, cooldownSeconds parameter is ignored)
				if sl.cooldownTracker != nil {
					sl.cooldownTracker.RecordCooldown(ruleID, symbol, 0)
				}
			}
		}

		// Update toplists if integration is enabled
		if sl.toplistIntegration != nil {
			// Create a copy of metrics for toplist update (since we'll return metrics to pool)
			metricsCopy := make(map[string]float64, len(metrics))
			for k, v := range metrics {
				metricsCopy[k] = v
			}
			if err := sl.toplistIntegration.UpdateToplists(sl.ctx, symbol, metricsCopy); err != nil {
				logger.Debug("Failed to update toplists",
					logger.ErrorField(err),
					logger.String("symbol", symbol),
				)
				// Don't fail scan cycle if toplist update fails
			}
		}

		// Return metrics map to pool
		sl.returnMetricsToPool(metrics)
	}

	// Publish toplist updates after scan cycle
	if sl.toplistIntegration != nil {
		if err := sl.toplistIntegration.PublishUpdates(sl.ctx); err != nil {
			logger.Debug("Failed to publish toplist updates",
				logger.ErrorField(err),
			)
		}
	}

	// Update statistics
	atomic.AddInt64(&sl.stats.SymbolsScanned, symbolsScanned)
	atomic.AddInt64(&sl.stats.RulesEvaluated, rulesEvaluated)
	atomic.AddInt64(&sl.stats.RulesMatched, rulesMatched)
	atomic.AddInt64(&sl.stats.AlertsEmitted, alertsEmitted)
}

// getRequiredMetrics returns the set of metrics required by active rules
func (sl *ScanLoop) getRequiredMetrics() map[string]bool {
	sl.requiredMetricsMu.RLock()
	defer sl.requiredMetricsMu.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]bool, len(sl.requiredMetrics))
	for k, v := range sl.requiredMetrics {
		result[k] = v
	}
	return result
}

// getMetricsFromSnapshot computes metrics from a symbol state snapshot
// If requiredMetrics is nil or empty, computes all metrics (backward compatibility)
// Returns a map that should be returned to pool after use
// Uses caching when possible to avoid recomputation
func (sl *ScanLoop) getMetricsFromSnapshot(snapshot *SymbolStateSnapshot, requiredMetrics map[string]bool) map[string]float64 {
	// Get the actual state for cache access
	// Note: We need to access the state to check cache, but we can't hold locks during computation
	state := sl.stateManager.GetState(snapshot.Symbol)
	
	// Try to get cached metrics (cache valid for 100ms within same scan cycle)
	// This helps when multiple rules need the same metrics
	cacheMaxAge := 100 * time.Millisecond
	if state != nil {
		if cached := state.getCachedMetrics(requiredMetrics, cacheMaxAge); cached != nil {
			// Get metrics map from pool
			metricsMap := sl.metricsPool.Get().(map[string]float64)
			// Clear map (but keep capacity)
			for k := range metricsMap {
				delete(metricsMap, k)
			}
			// Copy cached metrics
			for k, v := range cached {
				metricsMap[k] = v
			}
			return metricsMap
		}
	}

	// Get metrics map from pool
	metricsMap := sl.metricsPool.Get().(map[string]float64)

	// Clear map (but keep capacity)
	for k := range metricsMap {
		delete(metricsMap, k)
	}

	// Convert scanner snapshot to metrics snapshot
	metricSnapshot := &metrics.SymbolStateSnapshot{
		Symbol:           snapshot.Symbol,
		LiveBar:          snapshot.LiveBar,
		LastFinalBars:    snapshot.LastFinalBars,
		Indicators:       snapshot.Indicators,
		LastTickTime:     snapshot.LastTickTime,
		LastUpdate:       snapshot.LastUpdate,
		CurrentSession:   string(snapshot.CurrentSession),
		SessionStartTime: snapshot.SessionStartTime,
		YesterdayClose:   snapshot.YesterdayClose,
		TodayOpen:        snapshot.TodayOpen,
		TodayClose:       snapshot.TodayClose,
		PremarketVolume:  snapshot.PremarketVolume,
		MarketVolume:     snapshot.MarketVolume,
		PostmarketVolume: snapshot.PostmarketVolume,
		TradeCount:       snapshot.TradeCount,
	}

	// Copy trade count history
	if len(snapshot.TradeCountHistory) > 0 {
		metricSnapshot.TradeCountHistory = make([]int64, len(snapshot.TradeCountHistory))
		copy(metricSnapshot.TradeCountHistory, snapshot.TradeCountHistory)
	}

	// Copy candle directions
	if len(snapshot.CandleDirections) > 0 {
		metricSnapshot.CandleDirections = make(map[string][]bool)
		for k, v := range snapshot.CandleDirections {
			directions := make([]bool, len(v))
			copy(directions, v)
			metricSnapshot.CandleDirections[k] = directions
		}
	}

	// Compute only required metrics using registry (lazy computation)
	// If requiredMetrics is nil or empty, compute all (backward compatibility)
	computed := sl.metricRegistry.ComputeMetrics(metricSnapshot, requiredMetrics)

	// Copy computed metrics to pooled map
	for k, v := range computed {
		metricsMap[k] = v
	}

	// Cache computed metrics for future use in same scan cycle
	if state != nil {
		state.setCachedMetrics(computed, cacheMaxAge)
	}

	return metricsMap
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

	// Extract required metrics from enabled rules
	requiredMetrics := rules.ExtractRequiredMetrics(enabledRules)

	// Update compiled rules cache and required metrics (write lock)
	sl.rulesMu.Lock()
	oldCount := len(sl.compiledRules)
	sl.compiledRules = compiled
	sl.rulesMu.Unlock()

	// Update required metrics
	sl.requiredMetricsMu.Lock()
	sl.requiredMetrics = requiredMetrics
	sl.requiredMetricsMu.Unlock()

	// Update last reload time
	sl.lastReloadMu.Lock()
	sl.lastRuleReload = time.Now()
	sl.lastReloadMu.Unlock()

	// Log if rule count changed
	if oldCount != len(compiled) {
		logger.Info("Reloaded and compiled rules",
			logger.Int("old_rule_count", oldCount),
			logger.Int("new_rule_count", len(compiled)),
		)
	} else {
		logger.Debug("Reloaded and compiled rules (no change)",
			logger.Int("rule_count", len(compiled)),
		)
	}

	return nil
}

// shouldEvaluateRule checks if a rule should be evaluated based on filter configuration
// Returns true if volume threshold and session filters pass
func (sl *ScanLoop) shouldEvaluateRule(rule *models.Rule, metrics map[string]float64, currentSession string) bool {
	// Check each condition's filter configuration
	for _, cond := range rule.Conditions {
		// Check volume threshold
		if cond.VolumeThreshold != nil && *cond.VolumeThreshold > 0 {
			if !rules.CheckVolumeThreshold(metrics, cond.VolumeThreshold) {
				return false // Volume threshold not met
			}
		}

		// Check session filter
		if cond.CalculatedDuring != "" && cond.CalculatedDuring != "all" {
			if !rules.CheckSessionFilter(currentSession, cond.CalculatedDuring) {
				return false // Session filter not met
			}
		}
	}

	return true // All pre-filters passed
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
