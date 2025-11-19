package scanner

import (
	"context"
	"fmt"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/storage"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

// RehydrationConfig holds configuration for state rehydration
type RehydrationConfig struct {
	MaxBarsToLoad      int           // Maximum number of recent bars to load per symbol (default: 200)
	BarLookbackWindow  time.Duration // How far back to look for bars (default: 1 hour)
	IndicatorKeyPrefix string        // Prefix for indicator keys in Redis (default: "ind:")
	RehydrationTimeout time.Duration // Timeout for rehydration operation (default: 30 seconds)
	Symbols            []string      // Symbols to rehydrate (empty = all symbols from indicators)
}

// DefaultRehydrationConfig returns default configuration
func DefaultRehydrationConfig() RehydrationConfig {
	return RehydrationConfig{
		MaxBarsToLoad:      200,
		BarLookbackWindow:  1 * time.Hour,
		IndicatorKeyPrefix: "ind:",
		RehydrationTimeout: 30 * time.Second,
		Symbols:            []string{}, // Empty = discover from Redis
	}
}

// Rehydrator handles state rehydration on startup
type Rehydrator struct {
	config      RehydrationConfig
	stateManager *StateManager
	barStorage  storage.BarStorage
	redis       storage.RedisClient
}

// NewRehydrator creates a new rehydrator
func NewRehydrator(
	config RehydrationConfig,
	stateManager *StateManager,
	barStorage storage.BarStorage,
	redis storage.RedisClient,
) *Rehydrator {
	if stateManager == nil {
		panic("stateManager cannot be nil")
	}
	if barStorage == nil {
		panic("barStorage cannot be nil")
	}
	if redis == nil {
		panic("redis cannot be nil")
	}

	return &Rehydrator{
		config:       config,
		stateManager: stateManager,
		barStorage:  barStorage,
		redis:       redis,
	}
}

// RehydrateState rehydrates the state for all configured symbols
func (r *Rehydrator) RehydrateState(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, r.config.RehydrationTimeout)
	defer cancel()

	logger.Info("Starting state rehydration",
		logger.Int("max_bars", r.config.MaxBarsToLoad),
		logger.Duration("lookback_window", r.config.BarLookbackWindow),
	)

	// Discover symbols if not provided
	symbols := r.config.Symbols
	if len(symbols) == 0 {
		var err error
		symbols, err = r.discoverSymbols(ctx)
		if err != nil {
			return fmt.Errorf("failed to discover symbols: %w", err)
		}
		logger.Info("Discovered symbols for rehydration",
			logger.Int("symbol_count", len(symbols)),
		)
	}

	// Rehydrate each symbol
	rehydratedCount := 0
	for _, symbol := range symbols {
		if err := r.rehydrateSymbol(ctx, symbol); err != nil {
			logger.Error("Failed to rehydrate symbol",
				logger.ErrorField(err),
				logger.String("symbol", symbol),
			)
			// Continue with other symbols even if one fails
			continue
		}
		rehydratedCount++
	}

	logger.Info("State rehydration complete",
		logger.Int("symbols_rehydrated", rehydratedCount),
		logger.Int("total_symbols", len(symbols)),
	)

	return nil
}

// rehydrateSymbol rehydrates state for a single symbol
func (r *Rehydrator) rehydrateSymbol(ctx context.Context, symbol string) error {
	// Load recent bars from TimescaleDB
	if err := r.loadRecentBars(ctx, symbol); err != nil {
		return fmt.Errorf("failed to load recent bars: %w", err)
	}

	// Load indicators from Redis
	if err := r.loadIndicators(ctx, symbol); err != nil {
		return fmt.Errorf("failed to load indicators: %w", err)
	}

	return nil
}

// loadRecentBars loads recent finalized bars from TimescaleDB
func (r *Rehydrator) loadRecentBars(ctx context.Context, symbol string) error {
	// Use GetLatestBars to get the most recent bars
	bars, err := r.barStorage.GetLatestBars(ctx, symbol, r.config.MaxBarsToLoad)
	if err != nil {
		return fmt.Errorf("failed to get bars from storage: %w", err)
	}

	if len(bars) == 0 {
		logger.Debug("No bars found for symbol",
			logger.String("symbol", symbol),
		)
		return nil
	}

	// Update state manager with loaded bars
	for _, bar := range bars {
		if err := r.stateManager.UpdateFinalizedBar(bar); err != nil {
			logger.Warn("Failed to update finalized bar in state",
				logger.ErrorField(err),
				logger.String("symbol", symbol),
				logger.Time("timestamp", bar.Timestamp),
			)
			// Continue with other bars
		}
	}

	logger.Debug("Loaded recent bars for symbol",
		logger.String("symbol", symbol),
		logger.Int("bar_count", len(bars)),
	)

	return nil
}

// loadIndicators loads indicators from Redis
func (r *Rehydrator) loadIndicators(ctx context.Context, symbol string) error {
	key := fmt.Sprintf("%s%s", r.config.IndicatorKeyPrefix, symbol)

	var indicatorData struct {
		Symbol    string                 `json:"symbol"`
		Timestamp time.Time              `json:"timestamp"`
		Values    map[string]float64 `json:"values"`
	}

	err := r.redis.GetJSON(ctx, key, &indicatorData)
	if err != nil {
		// Check if key doesn't exist (indicator not found is OK)
		exists, existsErr := r.redis.Exists(ctx, key)
		if existsErr == nil && !exists {
			logger.Debug("No indicators found for symbol",
				logger.String("symbol", symbol),
			)
			return nil
		}
		// Other errors are real errors
		return fmt.Errorf("failed to get indicator JSON: %w", err)
	}

	if indicatorData.Values == nil || len(indicatorData.Values) == 0 {
		logger.Debug("No indicator values for symbol",
			logger.String("symbol", symbol),
		)
		return nil
	}

	// Update state manager with indicators
	if err := r.stateManager.UpdateIndicators(symbol, indicatorData.Values); err != nil {
		return fmt.Errorf("failed to update indicators in state: %w", err)
	}

	logger.Debug("Loaded indicators for symbol",
		logger.String("symbol", symbol),
		logger.Int("indicator_count", len(indicatorData.Values)),
	)

	return nil
}

// discoverSymbols discovers symbols from Redis indicator keys
// Since RedisClient doesn't have Keys/SCAN, we'll need to use a different approach
// For now, return empty list - symbols should be provided via config
// In production, this could use a Redis set or list to track active symbols
func (r *Rehydrator) discoverSymbols(ctx context.Context) ([]string, error) {
	// TODO: Implement symbol discovery
	// Options:
	// 1. Use a Redis set to track active symbols (e.g., "symbols:active")
	// 2. Use a Redis list
	// 3. Require symbols to be provided via config
	// For now, return empty - symbols must be provided in config
	logger.Warn("Symbol discovery not implemented - symbols must be provided via config")
	return []string{}, nil
}

// RehydrationStatus represents the status of rehydration
type RehydrationStatus struct {
	Completed    bool
	SymbolsCount int
	StartTime    time.Time
	EndTime      time.Time
	Error        error
}

// IsReady checks if rehydration is complete and state is ready
func (r *Rehydrator) IsReady() bool {
	// Check if we have any symbols in state
	return r.stateManager.GetSymbolCount() > 0
}

