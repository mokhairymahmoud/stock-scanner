package scanner

import (
	"testing"
	"time"
)

func TestCooldownTrackerImpl_IsOnCooldown(t *testing.T) {
	ct := NewCooldownTracker(1 * time.Minute)

	// Initially not on cooldown
	if ct.IsOnCooldown("rule-1", "AAPL") {
		t.Error("Expected not to be on cooldown initially")
	}

	// Record cooldown
	ct.RecordCooldown("rule-1", "AAPL", 300) // 5 minutes

	// Should be on cooldown
	if !ct.IsOnCooldown("rule-1", "AAPL") {
		t.Error("Expected to be on cooldown")
	}

	// Different symbol should not be on cooldown
	if ct.IsOnCooldown("rule-1", "GOOGL") {
		t.Error("Expected different symbol not to be on cooldown")
	}

	// Different rule should not be on cooldown
	if ct.IsOnCooldown("rule-2", "AAPL") {
		t.Error("Expected different rule not to be on cooldown")
	}
}

func TestCooldownTrackerImpl_RecordCooldown(t *testing.T) {
	ct := NewCooldownTracker(1 * time.Minute)

	// Record cooldown
	ct.RecordCooldown("rule-1", "AAPL", 60) // 1 minute

	// Verify it's active
	if !ct.IsOnCooldown("rule-1", "AAPL") {
		t.Error("Expected to be on cooldown after recording")
	}

	// Record zero cooldown (should not set)
	ct.RecordCooldown("rule-2", "GOOGL", 0)
	if ct.IsOnCooldown("rule-2", "GOOGL") {
		t.Error("Expected zero cooldown not to be set")
	}

	// Record negative cooldown (should not set)
	ct.RecordCooldown("rule-3", "MSFT", -10)
	if ct.IsOnCooldown("rule-3", "MSFT") {
		t.Error("Expected negative cooldown not to be set")
	}
}

func TestCooldownTrackerImpl_CooldownExpiration(t *testing.T) {
	ct := NewCooldownTracker(1 * time.Minute)

	// Record a very short cooldown
	ct.RecordCooldown("rule-1", "AAPL", 1) // 1 second

	// Should be on cooldown
	if !ct.IsOnCooldown("rule-1", "AAPL") {
		t.Error("Expected to be on cooldown")
	}

	// Wait for cooldown to expire
	time.Sleep(1100 * time.Millisecond)

	// Should no longer be on cooldown
	if ct.IsOnCooldown("rule-1", "AAPL") {
		t.Error("Expected cooldown to have expired")
	}
}

func TestCooldownTrackerImpl_ClearExpired(t *testing.T) {
	ct := NewCooldownTracker(1 * time.Minute)

	// Record multiple cooldowns
	ct.RecordCooldown("rule-1", "AAPL", 60)
	ct.RecordCooldown("rule-2", "GOOGL", 1) // Very short

	// Wait for one to expire
	time.Sleep(1100 * time.Millisecond)

	// Clear expired
	expiredCount := ct.ClearExpired()

	if expiredCount != 1 {
		t.Errorf("Expected 1 expired cooldown, got %d", expiredCount)
	}

	// Verify expired one is gone
	if ct.IsOnCooldown("rule-2", "GOOGL") {
		t.Error("Expected expired cooldown to be cleared")
	}

	// Verify active one is still there
	if !ct.IsOnCooldown("rule-1", "AAPL") {
		t.Error("Expected active cooldown to remain")
	}
}

func TestCooldownTrackerImpl_ClearAll(t *testing.T) {
	ct := NewCooldownTracker(1 * time.Minute)

	// Record multiple cooldowns
	ct.RecordCooldown("rule-1", "AAPL", 60)
	ct.RecordCooldown("rule-2", "GOOGL", 60)

	// Clear all
	ct.ClearAll()

	// Verify all are cleared
	if ct.IsOnCooldown("rule-1", "AAPL") {
		t.Error("Expected cooldown to be cleared")
	}

	if ct.IsOnCooldown("rule-2", "GOOGL") {
		t.Error("Expected cooldown to be cleared")
	}

	if ct.GetActiveCooldownCount() != 0 {
		t.Errorf("Expected 0 active cooldowns, got %d", ct.GetActiveCooldownCount())
	}
}

func TestCooldownTrackerImpl_GetActiveCooldownCount(t *testing.T) {
	ct := NewCooldownTracker(1 * time.Minute)

	if ct.GetActiveCooldownCount() != 0 {
		t.Errorf("Expected 0 active cooldowns initially, got %d", ct.GetActiveCooldownCount())
	}

	// Record multiple cooldowns
	ct.RecordCooldown("rule-1", "AAPL", 60)
	ct.RecordCooldown("rule-1", "GOOGL", 60)
	ct.RecordCooldown("rule-2", "AAPL", 60)

	if ct.GetActiveCooldownCount() != 3 {
		t.Errorf("Expected 3 active cooldowns, got %d", ct.GetActiveCooldownCount())
	}
}

func TestCooldownTrackerImpl_GetStats(t *testing.T) {
	ct := NewCooldownTracker(1 * time.Minute)

	// Record and check cooldowns
	ct.RecordCooldown("rule-1", "AAPL", 60)
	ct.IsOnCooldown("rule-1", "AAPL")
	ct.IsOnCooldown("rule-1", "GOOGL") // Not on cooldown

	stats := ct.GetStats()

	if stats.CooldownsActive != 1 {
		t.Errorf("Expected 1 active cooldown, got %d", stats.CooldownsActive)
	}

	if stats.CooldownsChecked != 2 {
		t.Errorf("Expected 2 cooldown checks, got %d", stats.CooldownsChecked)
	}

	if stats.CooldownsHit != 1 {
		t.Errorf("Expected 1 cooldown hit, got %d", stats.CooldownsHit)
	}
}

func TestCooldownTrackerImpl_Concurrency(t *testing.T) {
	ct := NewCooldownTracker(1 * time.Minute)

	// Test concurrent access
	done := make(chan bool)
	symbols := []string{"AAPL", "GOOGL", "MSFT", "TSLA", "AMZN"}

	// Concurrent writes
	for _, symbol := range symbols {
		go func(sym string) {
			for i := 0; i < 100; i++ {
				ct.RecordCooldown("rule-1", sym, 60)
				ct.IsOnCooldown("rule-1", sym)
			}
			done <- true
		}(symbol)
	}

	// Wait for all goroutines
	for i := 0; i < len(symbols); i++ {
		<-done
	}

	// Verify final state
	if ct.GetActiveCooldownCount() != len(symbols) {
		t.Errorf("Expected %d active cooldowns, got %d", len(symbols), ct.GetActiveCooldownCount())
	}
}

func TestCooldownTrackerImpl_CleanupLoop(t *testing.T) {
	ct := NewCooldownTracker(100 * time.Millisecond) // Very short cleanup interval

	// Start cleanup loop
	err := ct.Start()
	if err != nil {
		t.Fatalf("Failed to start cooldown tracker: %v", err)
	}
	defer ct.Stop()

	// Record a short cooldown
	ct.RecordCooldown("rule-1", "AAPL", 1) // 1 second

	// Wait for it to expire and cleanup to run
	time.Sleep(1200 * time.Millisecond)

	// Verify it was cleaned up
	if ct.IsOnCooldown("rule-1", "AAPL") {
		t.Error("Expected expired cooldown to be cleaned up")
	}

	stats := ct.GetStats()
	if stats.CooldownsExpired == 0 {
		t.Error("Expected some expired cooldowns to be tracked")
	}
}

func TestCooldownTrackerImpl_MultipleRulesSameSymbol(t *testing.T) {
	ct := NewCooldownTracker(1 * time.Minute)

	// Record cooldowns for different rules on same symbol
	ct.RecordCooldown("rule-1", "AAPL", 60)
	ct.RecordCooldown("rule-2", "AAPL", 60)

	// Both should be on cooldown
	if !ct.IsOnCooldown("rule-1", "AAPL") {
		t.Error("Expected rule-1 to be on cooldown")
	}

	if !ct.IsOnCooldown("rule-2", "AAPL") {
		t.Error("Expected rule-2 to be on cooldown")
	}

	// Verify they are independent
	ct.ClearAll()
	ct.RecordCooldown("rule-1", "AAPL", 60)

	if !ct.IsOnCooldown("rule-1", "AAPL") {
		t.Error("Expected rule-1 to be on cooldown")
	}

	if ct.IsOnCooldown("rule-2", "AAPL") {
		t.Error("Expected rule-2 not to be on cooldown")
	}
}

func TestCooldownTrackerImpl_SameRuleDifferentSymbols(t *testing.T) {
	ct := NewCooldownTracker(1 * time.Minute)

	// Record cooldowns for same rule on different symbols
	ct.RecordCooldown("rule-1", "AAPL", 60)
	ct.RecordCooldown("rule-1", "GOOGL", 60)

	// Both should be on cooldown
	if !ct.IsOnCooldown("rule-1", "AAPL") {
		t.Error("Expected AAPL to be on cooldown")
	}

	if !ct.IsOnCooldown("rule-1", "GOOGL") {
		t.Error("Expected GOOGL to be on cooldown")
	}

	// Verify they are independent
	ct.ClearAll()
	ct.RecordCooldown("rule-1", "AAPL", 60)

	if !ct.IsOnCooldown("rule-1", "AAPL") {
		t.Error("Expected AAPL to be on cooldown")
	}

	if ct.IsOnCooldown("rule-1", "GOOGL") {
		t.Error("Expected GOOGL not to be on cooldown")
	}
}

func TestCooldownTrackerImpl_StartStop(t *testing.T) {
	ct := NewCooldownTracker(100 * time.Millisecond)

	// Start
	err := ct.Start()
	if err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	// Try to start again (should fail)
	err = ct.Start()
	if err == nil {
		t.Error("Expected error when starting already running tracker")
	}

	// Stop
	ct.Stop()

	// Should be able to start again after stop
	err = ct.Start()
	if err != nil {
		t.Fatalf("Failed to start after stop: %v", err)
	}

	ct.Stop()
}

func TestNewCooldownTracker(t *testing.T) {
	// Test with default cleanup interval
	ct := NewCooldownTracker(0)
	if ct.cleanupInterval != 1*time.Minute {
		t.Errorf("Expected default cleanup interval 1m, got %v", ct.cleanupInterval)
	}

	// Test with custom cleanup interval
	ct2 := NewCooldownTracker(30 * time.Second)
	if ct2.cleanupInterval != 30*time.Second {
		t.Errorf("Expected cleanup interval 30s, got %v", ct2.cleanupInterval)
	}
}

