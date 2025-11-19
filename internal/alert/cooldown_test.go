package alert

import (
	"context"
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
)

func TestCooldownManager_CheckAndSetCooldown(t *testing.T) {
	redis := storage.NewMockRedisClient()
	cooldown := NewCooldownManager(redis, 5*time.Minute)

	alert := &models.Alert{
		ID:        "alert-1",
		RuleID:    "rule-1",
		Symbol:    "AAPL",
		Timestamp: time.Now(),
		Price:     150.0,
	}

	ctx := context.Background()
	userID := "user-1"

	// First check should not be in cooldown
	inCooldown, err := cooldown.CheckAndSetCooldown(ctx, alert, userID)
	if err != nil {
		t.Fatalf("Failed to check cooldown: %v", err)
	}
	if inCooldown {
		t.Error("Expected alert not to be in cooldown on first check")
	}

	// Second check should be in cooldown
	inCooldown, err = cooldown.CheckAndSetCooldown(ctx, alert, userID)
	if err != nil {
		t.Fatalf("Failed to check cooldown: %v", err)
	}
	if !inCooldown {
		t.Error("Expected alert to be in cooldown on second check")
	}
}

func TestCooldownManager_DifferentUsers(t *testing.T) {
	redis := storage.NewMockRedisClient()
	cooldown := NewCooldownManager(redis, 5*time.Minute)

	alert := &models.Alert{
		ID:        "alert-1",
		RuleID:    "rule-1",
		Symbol:    "AAPL",
		Timestamp: time.Now(),
		Price:     150.0,
	}

	ctx := context.Background()

	// Set cooldown for user-1
	inCooldown, err := cooldown.CheckAndSetCooldown(ctx, alert, "user-1")
	if err != nil {
		t.Fatalf("Failed to check cooldown: %v", err)
	}
	if inCooldown {
		t.Error("Expected alert not to be in cooldown for user-1")
	}

	// Check for user-2 should not be in cooldown (different user)
	inCooldown, err = cooldown.CheckAndSetCooldown(ctx, alert, "user-2")
	if err != nil {
		t.Fatalf("Failed to check cooldown: %v", err)
	}
	if inCooldown {
		t.Error("Expected alert not to be in cooldown for user-2 (different user)")
	}
}

func TestCooldownManager_GenerateCooldownKey(t *testing.T) {
	alert := &models.Alert{
		ID:        "alert-1",
		RuleID:    "rule-1",
		Symbol:    "AAPL",
		Timestamp: time.Now(),
	}

	key1 := GenerateCooldownKey(alert, "user-1")
	key2 := GenerateCooldownKey(alert, "user-2")
	key3 := GenerateCooldownKey(alert, "user-1")

	// Different users should have different keys
	if key1 == key2 {
		t.Error("Expected different cooldown keys for different users")
	}

	// Same user should have same key
	if key1 != key3 {
		t.Error("Expected same cooldown key for same user")
	}

	// Test default user
	keyDefault := GenerateCooldownKey(alert, "")
	if keyDefault == "" {
		t.Error("Expected non-empty cooldown key for default user")
	}
}

