package scanner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

// InMemoryCooldownTracker is an in-memory implementation of CooldownTracker
type InMemoryCooldownTracker struct {
	mu              sync.RWMutex
	cooldowns       map[string]time.Time // Key: "ruleID|symbol", Value: cooldown end time
	globalCooldown  time.Duration        // Global cooldown duration for all rules
	cleanupInterval time.Duration
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	running         bool
}

// NewCooldownTracker creates a new in-memory cooldown tracker with global cooldown
func NewCooldownTracker(globalCooldown, cleanupInterval time.Duration) *InMemoryCooldownTracker {
	if cleanupInterval <= 0 {
		cleanupInterval = 5 * time.Minute // Default: cleanup every 5 minutes
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &InMemoryCooldownTracker{
		cooldowns:       make(map[string]time.Time),
		globalCooldown:  globalCooldown,
		cleanupInterval: cleanupInterval,
		ctx:             ctx,
		cancel:          cancel,
	}
}

// Start starts the cooldown tracker (starts cleanup goroutine)
func (ct *InMemoryCooldownTracker) Start() error {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	if ct.running {
		return fmt.Errorf("cooldown tracker is already running")
	}

	ct.running = true

	// Start cleanup goroutine
	ct.wg.Add(1)
	go ct.cleanup()

	return nil
}

// Stop stops the cooldown tracker
func (ct *InMemoryCooldownTracker) Stop() {
	ct.mu.Lock()
	if !ct.running {
		ct.mu.Unlock()
		return
	}
	ct.running = false
	ct.mu.Unlock()

	ct.cancel()
	ct.wg.Wait()
}

// IsOnCooldown checks if a rule is on cooldown for a symbol
func (ct *InMemoryCooldownTracker) IsOnCooldown(ruleID, symbol string) bool {
	if ruleID == "" || symbol == "" {
		return false
	}

	key := ruleID + "|" + symbol

	ct.mu.RLock()
	defer ct.mu.RUnlock()

	cooldownEnd, exists := ct.cooldowns[key]
	if !exists {
		return false
	}

	// Check if cooldown has expired
	return time.Now().Before(cooldownEnd)
}

// RecordCooldown records that a rule fired for a symbol (starts cooldown using global cooldown)
func (ct *InMemoryCooldownTracker) RecordCooldown(ruleID, symbol string, cooldownSeconds int) {
	if ruleID == "" || symbol == "" {
		return
	}

	// Use global cooldown instead of per-rule cooldown
	ct.mu.RLock()
	cooldownDuration := ct.globalCooldown
	ct.mu.RUnlock()

	if cooldownDuration <= 0 {
		return // No cooldown configured
	}

	key := ruleID + "|" + symbol
	cooldownEnd := time.Now().Add(cooldownDuration)

	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.cooldowns[key] = cooldownEnd
}

// GetCooldownEnd returns when the cooldown ends for a rule-symbol pair
// Returns zero time if not on cooldown
func (ct *InMemoryCooldownTracker) GetCooldownEnd(ruleID, symbol string) time.Time {
	if ruleID == "" || symbol == "" {
		return time.Time{}
	}

	key := ruleID + "|" + symbol

	ct.mu.RLock()
	defer ct.mu.RUnlock()

	return ct.cooldowns[key]
}

// ClearCooldown clears the cooldown for a rule-symbol pair
func (ct *InMemoryCooldownTracker) ClearCooldown(ruleID, symbol string) {
	if ruleID == "" || symbol == "" {
		return
	}

	key := ruleID + "|" + symbol

	ct.mu.Lock()
	defer ct.mu.Unlock()

	delete(ct.cooldowns, key)
}

// ClearAllCooldowns clears all cooldowns
func (ct *InMemoryCooldownTracker) ClearAllCooldowns() {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.cooldowns = make(map[string]time.Time)
}

// GetCooldownCount returns the number of active cooldowns
func (ct *InMemoryCooldownTracker) GetCooldownCount() int {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	return len(ct.cooldowns)
}

// cleanup periodically removes expired cooldowns
func (ct *InMemoryCooldownTracker) cleanup() {
	defer ct.wg.Done()

	ticker := time.NewTicker(ct.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ct.ctx.Done():
			return
		case <-ticker.C:
			ct.cleanupExpired()
		}
	}
}

// cleanupExpired removes expired cooldowns
func (ct *InMemoryCooldownTracker) cleanupExpired() {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	now := time.Now()
	expired := make([]string, 0)

	for key, cooldownEnd := range ct.cooldowns {
		if now.After(cooldownEnd) {
			expired = append(expired, key)
		}
	}

	// Remove expired cooldowns
	for _, key := range expired {
		delete(ct.cooldowns, key)
	}

	if len(expired) > 0 {
		logger.Debug("Cleaned up expired cooldowns",
			logger.Int("count", len(expired)),
		)
	}
}
