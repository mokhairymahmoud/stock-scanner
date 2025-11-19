package alert

import (
	"context"
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
)

func TestDeduplicator_IsDuplicate(t *testing.T) {
	redis := storage.NewMockRedisClient()
	deduplicator := NewDeduplicator(redis, 1*time.Hour)

	alert := &models.Alert{
		ID:        "alert-1",
		RuleID:    "rule-1",
		Symbol:    "AAPL",
		Timestamp: time.Now(),
		Price:     150.0,
		Message:   "Test alert",
	}

	ctx := context.Background()

	// First check should not be duplicate
	isDuplicate, err := deduplicator.IsDuplicate(ctx, alert)
	if err != nil {
		t.Fatalf("Failed to check duplicate: %v", err)
	}
	if isDuplicate {
		t.Error("Expected alert not to be duplicate on first check")
	}

	// Second check should be duplicate
	isDuplicate, err = deduplicator.IsDuplicate(ctx, alert)
	if err != nil {
		t.Fatalf("Failed to check duplicate: %v", err)
	}
	if !isDuplicate {
		t.Error("Expected alert to be duplicate on second check")
	}
}

func TestDeduplicator_GenerateIdempotencyKey(t *testing.T) {
	alert1 := &models.Alert{
		ID:        "alert-1",
		RuleID:    "rule-1",
		Symbol:    "AAPL",
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		Price:     150.0,
	}

	alert2 := &models.Alert{
		ID:        "alert-2",
		RuleID:    "rule-1",
		Symbol:    "AAPL",
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 500000000, time.UTC), // Same second, different nanosecond
		Price:     151.0,
	}

	alert3 := &models.Alert{
		ID:        "alert-3",
		RuleID:    "rule-1",
		Symbol:    "AAPL",
		Timestamp: time.Date(2024, 1, 1, 12, 0, 1, 0, time.UTC), // Different second
		Price:     152.0,
	}

	key1 := GenerateIdempotencyKey(alert1)
	key2 := GenerateIdempotencyKey(alert2)
	key3 := GenerateIdempotencyKey(alert3)

	// Same rule, symbol, and second should generate same key (truncated to second)
	if key1 != key2 {
		t.Errorf("Expected same idempotency key for alerts in same second, got %s and %s", key1, key2)
	}

	// Different second should generate different key
	if key1 == key3 {
		t.Errorf("Expected different idempotency key for alerts in different seconds, got %s for both", key1)
	}
}

func TestDeduplicator_DifferentSymbols(t *testing.T) {
	redis := storage.NewMockRedisClient()
	deduplicator := NewDeduplicator(redis, 1*time.Hour)

	alert1 := &models.Alert{
		ID:        "alert-1",
		RuleID:    "rule-1",
		Symbol:    "AAPL",
		Timestamp: time.Now(),
		Price:     150.0,
	}

	alert2 := &models.Alert{
		ID:        "alert-2",
		RuleID:    "rule-1",
		Symbol:    "MSFT",
		Timestamp: time.Now(),
		Price:     200.0,
	}

	ctx := context.Background()

	// Check first alert
	isDuplicate, err := deduplicator.IsDuplicate(ctx, alert1)
	if err != nil {
		t.Fatalf("Failed to check duplicate: %v", err)
	}
	if isDuplicate {
		t.Error("Expected alert1 not to be duplicate")
	}

	// Check second alert (different symbol) should not be duplicate
	isDuplicate, err = deduplicator.IsDuplicate(ctx, alert2)
	if err != nil {
		t.Fatalf("Failed to check duplicate: %v", err)
	}
	if isDuplicate {
		t.Error("Expected alert2 not to be duplicate (different symbol)")
	}
}

