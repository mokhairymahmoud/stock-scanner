package scanner

import (
	"testing"
	"time"
)

func TestNewCooldownTracker(t *testing.T) {
	ct := NewCooldownTracker(10*time.Second, 5*time.Minute)
	if ct == nil {
		t.Fatal("Expected cooldown tracker to be created")
	}

	if ct.globalCooldown != 10*time.Second {
		t.Errorf("Expected global cooldown 10s, got %v", ct.globalCooldown)
	}

	if ct.cleanupInterval != 5*time.Minute {
		t.Errorf("Expected cleanup interval 5m, got %v", ct.cleanupInterval)
	}

	// Test default cleanup interval
	ct2 := NewCooldownTracker(10*time.Second, 0)
	if ct2.cleanupInterval != 5*time.Minute {
		t.Errorf("Expected default cleanup interval 5m, got %v", ct2.cleanupInterval)
	}
}

func TestCooldownTracker_IsOnCooldown(t *testing.T) {
	ct := NewCooldownTracker(10*time.Second, 5*time.Minute)

	// Not on cooldown initially
	if ct.IsOnCooldown("rule-1", "AAPL") {
		t.Error("Expected not to be on cooldown initially")
	}

	// Record cooldown (cooldownSeconds parameter is ignored, uses global cooldown)
	ct.RecordCooldown("rule-1", "AAPL", 0)

	// Should be on cooldown
	if !ct.IsOnCooldown("rule-1", "AAPL") {
		t.Error("Expected to be on cooldown")
	}

	// Different rule-symbol pair should not be on cooldown
	if ct.IsOnCooldown("rule-2", "AAPL") {
		t.Error("Expected different rule not to be on cooldown")
	}

	if ct.IsOnCooldown("rule-1", "GOOGL") {
		t.Error("Expected different symbol not to be on cooldown")
	}
}

func TestCooldownTracker_RecordCooldown(t *testing.T) {
	ct := NewCooldownTracker(10*time.Second, 5*time.Minute)

	// Record cooldown (uses global cooldown, parameter ignored)
	ct.RecordCooldown("rule-1", "AAPL", 0)

	// Verify cooldown end time
	cooldownEnd := ct.GetCooldownEnd("rule-1", "AAPL")
	if cooldownEnd.IsZero() {
		t.Error("Expected cooldown end time to be set")
	}

	// Cooldown should be in the future
	now := time.Now()
	if cooldownEnd.Before(now) {
		t.Error("Expected cooldown end time to be in the future")
	}

	// Cooldown should be approximately 10 seconds from now (global cooldown)
	expectedEnd := now.Add(10 * time.Second)
	diff := cooldownEnd.Sub(expectedEnd)
	if diff < -1*time.Second || diff > 1*time.Second {
		t.Errorf("Expected cooldown end to be approximately 10s from now, got %v", diff)
	}
}

func TestCooldownTracker_CooldownExpires(t *testing.T) {
	ct := NewCooldownTracker(10*time.Second, 5*time.Minute)

	// Record a very short cooldown
	ct.RecordCooldown("rule-1", "AAPL", 1) // 1 second

	// Should be on cooldown immediately
	if !ct.IsOnCooldown("rule-1", "AAPL") {
		t.Error("Expected to be on cooldown immediately")
	}

	// Wait for cooldown to expire
	time.Sleep(1100 * time.Millisecond)

	// Should not be on cooldown anymore
	if ct.IsOnCooldown("rule-1", "AAPL") {
		t.Error("Expected cooldown to have expired")
	}
}

func TestCooldownTracker_ClearCooldown(t *testing.T) {
	ct := NewCooldownTracker(10*time.Second, 5*time.Minute)

	// Record cooldown
	ct.RecordCooldown("rule-1", "AAPL", 0)

	// Verify on cooldown
	if !ct.IsOnCooldown("rule-1", "AAPL") {
		t.Error("Expected to be on cooldown")
	}

	// Clear cooldown
	ct.ClearCooldown("rule-1", "AAPL")

	// Should not be on cooldown anymore
	if ct.IsOnCooldown("rule-1", "AAPL") {
		t.Error("Expected cooldown to be cleared")
	}
}

func TestCooldownTracker_ClearAllCooldowns(t *testing.T) {
	ct := NewCooldownTracker(10*time.Second, 5*time.Minute)

	// Record multiple cooldowns
	ct.RecordCooldown("rule-1", "AAPL", 0)
	ct.RecordCooldown("rule-1", "GOOGL", 10)
	ct.RecordCooldown("rule-2", "AAPL", 10)

	if ct.GetCooldownCount() != 3 {
		t.Errorf("Expected 3 cooldowns, got %d", ct.GetCooldownCount())
	}

	// Clear all
	ct.ClearAllCooldowns()

	if ct.GetCooldownCount() != 0 {
		t.Errorf("Expected 0 cooldowns after clear, got %d", ct.GetCooldownCount())
	}

	// Verify all are cleared
	if ct.IsOnCooldown("rule-1", "AAPL") {
		t.Error("Expected cooldown to be cleared")
	}

	if ct.IsOnCooldown("rule-1", "GOOGL") {
		t.Error("Expected cooldown to be cleared")
	}

	if ct.IsOnCooldown("rule-2", "AAPL") {
		t.Error("Expected cooldown to be cleared")
	}
}

func TestCooldownTracker_GetCooldownCount(t *testing.T) {
	ct := NewCooldownTracker(10*time.Second, 5*time.Minute)

	if ct.GetCooldownCount() != 0 {
		t.Errorf("Expected 0 cooldowns initially, got %d", ct.GetCooldownCount())
	}

	// Add cooldowns
	ct.RecordCooldown("rule-1", "AAPL", 0)
	ct.RecordCooldown("rule-1", "GOOGL", 10)

	if ct.GetCooldownCount() != 2 {
		t.Errorf("Expected 2 cooldowns, got %d", ct.GetCooldownCount())
	}

	// Add same rule-symbol (should overwrite, not add)
	ct.RecordCooldown("rule-1", "AAPL", 20)

	if ct.GetCooldownCount() != 2 {
		t.Errorf("Expected 2 cooldowns (overwrite), got %d", ct.GetCooldownCount())
	}
}

func TestCooldownTracker_InvalidInputs(t *testing.T) {
	ct := NewCooldownTracker(10*time.Second, 5*time.Minute)

	// Empty rule ID or symbol should not record cooldown
	ct.RecordCooldown("", "AAPL", 10)
	ct.RecordCooldown("rule-1", "", 10)
	ct.RecordCooldown("rule-1", "AAPL", 0) // Parameter ignored, uses global cooldown

	if ct.GetCooldownCount() != 0 {
		t.Errorf("Expected 0 cooldowns with invalid inputs, got %d", ct.GetCooldownCount())
	}

	// Empty inputs should return false for IsOnCooldown
	if ct.IsOnCooldown("", "AAPL") {
		t.Error("Expected false for empty rule ID")
	}

	if ct.IsOnCooldown("rule-1", "") {
		t.Error("Expected false for empty symbol")
	}
}

func TestCooldownTracker_Concurrency(t *testing.T) {
	ct := NewCooldownTracker(10*time.Second, 5*time.Minute)

	// Test concurrent writes and reads
	done := make(chan bool)
	symbols := []string{"AAPL", "GOOGL", "MSFT", "TSLA", "AMZN"}

	// Concurrent writes
	for _, symbol := range symbols {
		go func(sym string) {
			for i := 0; i < 100; i++ {
				ct.RecordCooldown("rule-1", sym, 10)
				ct.IsOnCooldown("rule-1", sym)
			}
			done <- true
		}(symbol)
	}

	// Concurrent reads
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				for _, symbol := range symbols {
					ct.IsOnCooldown("rule-1", symbol)
					ct.GetCooldownCount()
				}
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < len(symbols)+5; i++ {
		<-done
	}

	// Verify final state
	if ct.GetCooldownCount() != len(symbols) {
		t.Errorf("Expected %d cooldowns, got %d", len(symbols), ct.GetCooldownCount())
	}
}

func TestCooldownTracker_StartStop(t *testing.T) {
	ct := NewCooldownTracker(10*time.Second, 100*time.Millisecond) // Short cleanup interval for testing

	// Start tracker
	err := ct.Start()
	if err != nil {
		t.Fatalf("Failed to start cooldown tracker: %v", err)
	}

	// Record a short cooldown
	ct.RecordCooldown("rule-1", "AAPL", 1) // 1 second

	// Wait for cleanup to run (should clean up expired cooldowns)
	time.Sleep(150 * time.Millisecond)

	// Stop tracker
	ct.Stop()

	// Try to start again (should work)
	err = ct.Start()
	if err != nil {
		t.Fatalf("Failed to start cooldown tracker again: %v", err)
	}

	ct.Stop()
}

func TestCooldownTracker_CleanupExpired(t *testing.T) {
	ct := NewCooldownTracker(100 * time.Millisecond)

	// Start tracker
	ct.Start()
	defer ct.Stop()

	// Record a very short cooldown
	ct.RecordCooldown("rule-1", "AAPL", 1) // 1 second

	// Record a longer cooldown
	ct.RecordCooldown("rule-2", "GOOGL", 10) // 10 seconds

	if ct.GetCooldownCount() != 2 {
		t.Errorf("Expected 2 cooldowns, got %d", ct.GetCooldownCount())
	}

	// Wait for first cooldown to expire and cleanup to run
	time.Sleep(1200 * time.Millisecond)

	// First cooldown should be cleaned up
	if ct.IsOnCooldown("rule-1", "AAPL") {
		t.Error("Expected expired cooldown to be cleaned up")
	}

	// Second cooldown should still be active
	if !ct.IsOnCooldown("rule-2", "GOOGL") {
		t.Error("Expected active cooldown to still be present")
	}

	// Count should be 1
	if ct.GetCooldownCount() != 1 {
		t.Errorf("Expected 1 cooldown after cleanup, got %d", ct.GetCooldownCount())
	}
}

func TestCooldownTracker_MultipleRulesSameSymbol(t *testing.T) {
	ct := NewCooldownTracker(10*time.Second, 5*time.Minute)

	// Record cooldowns for different rules on same symbol
	ct.RecordCooldown("rule-1", "AAPL", 0)
	ct.RecordCooldown("rule-2", "AAPL", 20)
	ct.RecordCooldown("rule-3", "AAPL", 30)

	if ct.GetCooldownCount() != 3 {
		t.Errorf("Expected 3 cooldowns, got %d", ct.GetCooldownCount())
	}

	// All should be on cooldown
	if !ct.IsOnCooldown("rule-1", "AAPL") {
		t.Error("Expected rule-1 to be on cooldown")
	}

	if !ct.IsOnCooldown("rule-2", "AAPL") {
		t.Error("Expected rule-2 to be on cooldown")
	}

	if !ct.IsOnCooldown("rule-3", "AAPL") {
		t.Error("Expected rule-3 to be on cooldown")
	}
}

func TestCooldownTracker_SameRuleDifferentSymbols(t *testing.T) {
	ct := NewCooldownTracker(10*time.Second, 5*time.Minute)

	// Record cooldowns for same rule on different symbols
	ct.RecordCooldown("rule-1", "AAPL", 0)
	ct.RecordCooldown("rule-1", "GOOGL", 20)
	ct.RecordCooldown("rule-1", "MSFT", 30)

	if ct.GetCooldownCount() != 3 {
		t.Errorf("Expected 3 cooldowns, got %d", ct.GetCooldownCount())
	}

	// All should be on cooldown
	if !ct.IsOnCooldown("rule-1", "AAPL") {
		t.Error("Expected AAPL to be on cooldown")
	}

	if !ct.IsOnCooldown("rule-1", "GOOGL") {
		t.Error("Expected GOOGL to be on cooldown")
	}

	if !ct.IsOnCooldown("rule-1", "MSFT") {
		t.Error("Expected MSFT to be on cooldown")
	}
}

func TestCooldownTracker_OverwriteCooldown(t *testing.T) {
	ct := NewCooldownTracker(10*time.Second, 5*time.Minute)

	// Record initial cooldown
	ct.RecordCooldown("rule-1", "AAPL", 0)
	firstEnd := ct.GetCooldownEnd("rule-1", "AAPL")

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Record new cooldown (should overwrite)
	ct.RecordCooldown("rule-1", "AAPL", 0)
	secondEnd := ct.GetCooldownEnd("rule-1", "AAPL")

	// Second end should be later than first
	if !secondEnd.After(firstEnd) {
		t.Error("Expected new cooldown to extend the end time")
	}

	// Count should still be 1
	if ct.GetCooldownCount() != 1 {
		t.Errorf("Expected 1 cooldown after overwrite, got %d", ct.GetCooldownCount())
	}
}
