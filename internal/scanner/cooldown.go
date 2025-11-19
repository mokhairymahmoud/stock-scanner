package scanner

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CooldownTrackerImpl implements the CooldownTracker interface
// Manages per-rule, per-symbol cooldowns to prevent duplicate alerts
type CooldownTrackerImpl struct {
	mu        sync.RWMutex
	cooldowns map[string]time.Time // Key: "ruleID|symbol", Value: cooldown end time
	cleanupInterval time.Duration   // How often to clean up expired cooldowns
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	running   bool
	stats     CooldownStats
}

// CooldownStats holds statistics about cooldown tracking
type CooldownStats struct {
	CooldownsActive  int64
	CooldownsChecked int64
	CooldownsHit     int64 // Number of times cooldown prevented an alert
	CooldownsExpired int64
	mu               sync.RWMutex
}

// NewCooldownTracker creates a new cooldown tracker
func NewCooldownTracker(cleanupInterval time.Duration) *CooldownTrackerImpl {
	if cleanupInterval <= 0 {
		cleanupInterval = 1 * time.Minute // Default: cleanup every minute
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &CooldownTrackerImpl{
		cooldowns:      make(map[string]time.Time),
		cleanupInterval: cleanupInterval,
		ctx:            ctx,
		cancel:         cancel,
		stats:          CooldownStats{},
	}
}

// Start starts the cooldown tracker cleanup goroutine
func (ct *CooldownTrackerImpl) Start() error {
	ct.mu.Lock()
	if ct.running {
		ct.mu.Unlock()
		return fmt.Errorf("cooldown tracker is already running")
	}
	ct.running = true
	ct.mu.Unlock()

	ct.wg.Add(1)
	go ct.cleanupLoop()

	return nil
}

// Stop stops the cooldown tracker
func (ct *CooldownTrackerImpl) Stop() {
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
func (ct *CooldownTrackerImpl) IsOnCooldown(ruleID, symbol string) bool {
	key := ct.getKey(ruleID, symbol)

	ct.mu.RLock()
	cooldownEnd, exists := ct.cooldowns[key]
	ct.mu.RUnlock()

	if !exists {
		ct.incrementChecked()
		return false
	}

	ct.incrementChecked()

	// Check if cooldown has expired
	if time.Now().After(cooldownEnd) {
		// Cooldown expired, remove it
		ct.mu.Lock()
		delete(ct.cooldowns, key)
		ct.mu.Unlock()
		ct.incrementExpired()
		return false
	}

	// Still on cooldown
	ct.incrementHit()
	return true
}

// RecordCooldown records that a rule fired for a symbol (starts cooldown)
func (ct *CooldownTrackerImpl) RecordCooldown(ruleID, symbol string, cooldownSeconds int) {
	if cooldownSeconds <= 0 {
		return // No cooldown
	}

	key := ct.getKey(ruleID, symbol)
	cooldownEnd := time.Now().Add(time.Duration(cooldownSeconds) * time.Second)

	ct.mu.Lock()
	ct.cooldowns[key] = cooldownEnd
	activeCount := int64(len(ct.cooldowns))
	ct.mu.Unlock()

	ct.updateActiveCount(activeCount)
}

// GetActiveCooldownCount returns the number of active cooldowns
func (ct *CooldownTrackerImpl) GetActiveCooldownCount() int {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return len(ct.cooldowns)
}

// GetStats returns current cooldown statistics
func (ct *CooldownTrackerImpl) GetStats() CooldownStats {
	ct.stats.mu.RLock()
	defer ct.stats.mu.RUnlock()

	// Return a copy
	return CooldownStats{
		CooldownsActive:  ct.stats.CooldownsActive,
		CooldownsChecked: ct.stats.CooldownsChecked,
		CooldownsHit:     ct.stats.CooldownsHit,
		CooldownsExpired: ct.stats.CooldownsExpired,
	}
}

// ClearExpired removes all expired cooldowns
func (ct *CooldownTrackerImpl) ClearExpired() int {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	now := time.Now()
	expiredCount := 0

	for key, cooldownEnd := range ct.cooldowns {
		if now.After(cooldownEnd) {
			delete(ct.cooldowns, key)
			expiredCount++
		}
	}

	if expiredCount > 0 {
		ct.updateActiveCount(int64(len(ct.cooldowns)))
		ct.incrementExpiredBy(int64(expiredCount))
	}

	return expiredCount
}

// ClearAll removes all cooldowns (useful for testing)
func (ct *CooldownTrackerImpl) ClearAll() {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.cooldowns = make(map[string]time.Time)
	ct.updateActiveCount(0)
}

// getKey generates a key for the cooldown map
func (ct *CooldownTrackerImpl) getKey(ruleID, symbol string) string {
	return ruleID + "|" + symbol
}

// cleanupLoop periodically cleans up expired cooldowns
func (ct *CooldownTrackerImpl) cleanupLoop() {
	defer ct.wg.Done()

	ticker := time.NewTicker(ct.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ct.ctx.Done():
			return
		case <-ticker.C:
			ct.ClearExpired()
		}
	}
}

// incrementChecked increments the checked counter
func (ct *CooldownTrackerImpl) incrementChecked() {
	ct.stats.mu.Lock()
	defer ct.stats.mu.Unlock()
	ct.stats.CooldownsChecked++
}

// incrementHit increments the hit counter
func (ct *CooldownTrackerImpl) incrementHit() {
	ct.stats.mu.Lock()
	defer ct.stats.mu.Unlock()
	ct.stats.CooldownsHit++
}

// incrementExpired increments the expired counter
func (ct *CooldownTrackerImpl) incrementExpired() {
	ct.stats.mu.Lock()
	defer ct.stats.mu.Unlock()
	ct.stats.CooldownsExpired++
}

// incrementExpiredBy increments the expired counter by a value
func (ct *CooldownTrackerImpl) incrementExpiredBy(count int64) {
	ct.stats.mu.Lock()
	defer ct.stats.mu.Unlock()
	ct.stats.CooldownsExpired += count
}

// updateActiveCount updates the active cooldown count
func (ct *CooldownTrackerImpl) updateActiveCount(count int64) {
	ct.stats.mu.Lock()
	defer ct.stats.mu.Unlock()
	ct.stats.CooldownsActive = count
}

