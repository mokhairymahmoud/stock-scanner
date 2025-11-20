package toplist

import (
	"context"
	"testing"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
)

func TestRedisToplistUpdater_UpdateSystemToplist(t *testing.T) {
	mockRedis := storage.NewMockRedisClient()
	updater := NewRedisToplistUpdater(mockRedis)
	ctx := context.Background()

	err := updater.UpdateSystemToplist(ctx, models.MetricChangePct, models.Window1m, "AAPL", 2.5)
	if err != nil {
		t.Fatalf("UpdateSystemToplist() error = %v", err)
	}

	// Verify the key was created
	key := models.GetSystemToplistRedisKey(models.MetricChangePct, models.Window1m)
	score, err := mockRedis.ZScore(ctx, key, "AAPL")
	if err != nil {
		t.Fatalf("ZScore() error = %v", err)
	}
	if score != 2.5 {
		t.Errorf("ZScore() = %v, want %v", score, 2.5)
	}
}

func TestRedisToplistUpdater_UpdateUserToplist(t *testing.T) {
	mockRedis := storage.NewMockRedisClient()
	updater := NewRedisToplistUpdater(mockRedis)
	ctx := context.Background()

	err := updater.UpdateUserToplist(ctx, "user-123", "custom-1", "MSFT", 1.8)
	if err != nil {
		t.Fatalf("UpdateUserToplist() error = %v", err)
	}

	// Verify the key was created
	key := models.GetUserToplistRedisKey("user-123", "custom-1")
	score, err := mockRedis.ZScore(ctx, key, "MSFT")
	if err != nil {
		t.Fatalf("ZScore() error = %v", err)
	}
	if score != 1.8 {
		t.Errorf("ZScore() = %v, want %v", score, 1.8)
	}
}

func TestRedisToplistUpdater_BatchUpdate(t *testing.T) {
	mockRedis := storage.NewMockRedisClient()
	updater := NewRedisToplistUpdater(mockRedis)
	ctx := context.Background()

	updates := []ToplistUpdate{
		{Key: "toplist:change_pct:1m", Symbol: "AAPL", Value: 2.5},
		{Key: "toplist:change_pct:1m", Symbol: "MSFT", Value: 1.8},
		{Key: "toplist:volume:1d", Symbol: "GOOGL", Value: 1000000},
	}

	err := updater.BatchUpdate(ctx, updates)
	if err != nil {
		t.Fatalf("BatchUpdate() error = %v", err)
	}

	// Verify all updates were applied
	score1, _ := mockRedis.ZScore(ctx, "toplist:change_pct:1m", "AAPL")
	if score1 != 2.5 {
		t.Errorf("AAPL score = %v, want %v", score1, 2.5)
	}

	score2, _ := mockRedis.ZScore(ctx, "toplist:change_pct:1m", "MSFT")
	if score2 != 1.8 {
		t.Errorf("MSFT score = %v, want %v", score2, 1.8)
	}

	score3, _ := mockRedis.ZScore(ctx, "toplist:volume:1d", "GOOGL")
	if score3 != 1000000 {
		t.Errorf("GOOGL score = %v, want %v", score3, 1000000)
	}
}

func TestRedisToplistUpdater_BatchUpdate_Empty(t *testing.T) {
	mockRedis := storage.NewMockRedisClient()
	updater := NewRedisToplistUpdater(mockRedis)
	ctx := context.Background()

	err := updater.BatchUpdate(ctx, []ToplistUpdate{})
	if err != nil {
		t.Fatalf("BatchUpdate() with empty slice should not error, got %v", err)
	}
}

func TestRedisToplistUpdater_PublishUpdate(t *testing.T) {
	mockRedis := storage.NewMockRedisClient()
	updater := NewRedisToplistUpdater(mockRedis)
	ctx := context.Background()

	err := updater.PublishUpdate(ctx, "gainers_1m", "system")
	if err != nil {
		t.Fatalf("PublishUpdate() error = %v", err)
	}

	// The mock doesn't store published messages, but we can verify no error occurred
	// In a real test with Redis, we would subscribe and verify the message
}

func TestRedisToplistUpdater_Close(t *testing.T) {
	mockRedis := storage.NewMockRedisClient()
	updater := NewRedisToplistUpdater(mockRedis)

	err := updater.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

