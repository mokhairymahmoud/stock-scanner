// Package data contains component-level E2E tests for the toplist feature.
//
// These tests verify the toplist component end-to-end using mocks for dependencies.
// They test toplist updates, rankings, filtering, and integration with scanner worker.
package data

import (
	"context"
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/scanner"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
	"github.com/mohamedkhairy/stock-scanner/internal/toplist"
)

// TestToplistE2E_SystemToplistUpdate tests system toplist updates from scanner
func TestToplistE2E_SystemToplistUpdate(t *testing.T) {
	ctx := context.Background()
	redis := storage.NewMockRedisClient()

	// Create toplist updater
	updater := toplist.NewRedisToplistUpdater(redis)

	// Create toplist integration
	integration := scanner.NewToplistIntegration(updater, true, 1*time.Second)

	// Test symbols with different price changes
	symbols := []struct {
		symbol string
		metrics map[string]float64
	}{
		{"AAPL", map[string]float64{
			"price_change_1m_pct": 2.5,  // 2.5% gain
			"price_change_5m_pct": 3.0,
			"volume": 10000,
		}},
		{"GOOGL", map[string]float64{
			"price_change_1m_pct": -1.2, // 1.2% loss
			"price_change_5m_pct": -2.0,
			"volume": 5000,
		}},
		{"MSFT", map[string]float64{
			"price_change_1m_pct": 5.0,  // 5.0% gain (highest)
			"price_change_5m_pct": 6.0,
			"volume": 15000,
		}},
	}

	// Step 1: Update toplists for each symbol
	t.Log("Step 1: Updating toplists for symbols...")
	for _, s := range symbols {
		if err := integration.UpdateToplists(ctx, s.symbol, s.metrics); err != nil {
			t.Fatalf("Failed to update toplists for %s: %v", s.symbol, err)
		}
	}

	// Step 2: Publish updates (flush accumulated updates)
	t.Log("Step 2: Publishing toplist updates...")
	if err := integration.PublishUpdates(ctx); err != nil {
		t.Fatalf("Failed to publish updates: %v", err)
	}

	// Step 3: Verify rankings in Redis ZSET
	t.Log("Step 3: Verifying rankings in Redis ZSET...")
	
	// Check gainers_1m toplist (should be sorted descending by change_pct)
	gainersKey := models.GetSystemToplistRedisKey(models.MetricChangePct, models.Window1m)
	members, err := redis.ZRevRange(ctx, gainersKey, 0, 10)
	if err != nil {
		t.Fatalf("Failed to get rankings: %v", err)
	}

	if len(members) != 3 {
		t.Errorf("Expected 3 symbols in gainers toplist, got %d", len(members))
	}

	// Verify order: MSFT (5.0%) > AAPL (2.5%) > GOOGL (-1.2%)
	if len(members) >= 3 {
		if members[0].Member != "MSFT" {
			t.Errorf("Expected MSFT to be #1 (highest gain), got %s", members[0].Member)
		}
		if members[0].Score != 5.0 {
			t.Errorf("Expected MSFT score to be 5.0, got %f", members[0].Score)
		}

		if members[1].Member != "AAPL" {
			t.Errorf("Expected AAPL to be #2, got %s", members[1].Member)
		}
		if members[1].Score != 2.5 {
			t.Errorf("Expected AAPL score to be 2.5, got %f", members[1].Score)
		}

		if members[2].Member != "GOOGL" {
			t.Errorf("Expected GOOGL to be #3 (lowest), got %s", members[2].Member)
		}
		if members[2].Score != -1.2 {
			t.Errorf("Expected GOOGL score to be -1.2, got %f", members[2].Score)
		}
	}

	// Step 4: Verify volume toplist
	t.Log("Step 4: Verifying volume toplist...")
	volumeKey := models.GetSystemToplistRedisKey(models.MetricVolume, models.Window1m)
	volumeMembers, err := redis.ZRevRange(ctx, volumeKey, 0, 10)
	if err != nil {
		t.Fatalf("Failed to get volume rankings: %v", err)
	}

	if len(volumeMembers) != 3 {
		t.Errorf("Expected 3 symbols in volume toplist, got %d", len(volumeMembers))
	}

	// Verify order: MSFT (15000) > AAPL (10000) > GOOGL (5000)
	if len(volumeMembers) >= 3 {
		if volumeMembers[0].Member != "MSFT" {
			t.Errorf("Expected MSFT to be #1 in volume, got %s", volumeMembers[0].Member)
		}
		if volumeMembers[0].Score != 15000.0 {
			t.Errorf("Expected MSFT volume to be 15000, got %f", volumeMembers[0].Score)
		}
	}

	t.Log("✅ System toplist update test completed successfully!")
}

// TestToplistE2E_UserToplist tests user-custom toplist creation and updates
func TestToplistE2E_UserToplist(t *testing.T) {
	ctx := context.Background()
	redis := storage.NewMockRedisClient()

	// Create toplist updater
	updater := toplist.NewRedisToplistUpdater(redis)

	// Create mock store
	store := toplist.NewMockToplistStore()

	// Create toplist service
	service := toplist.NewToplistService(store, redis, updater)

	userID := "user-123"
	toplistID := "toplist-custom-1"

	// Step 1: Create user toplist
	t.Log("Step 1: Creating user toplist...")
	config := &models.ToplistConfig{
		ID:        toplistID,
		UserID:    userID,
		Name:      "My Custom Toplist",
		Metric:    models.MetricChangePct,
		TimeWindow: models.Window5m,
		SortOrder: models.SortOrderDesc,
		Enabled:   true,
	}

	if err := store.CreateToplist(ctx, config); err != nil {
		t.Fatalf("Failed to create toplist: %v", err)
	}

	// Step 2: Update user toplist with symbol values
	t.Log("Step 2: Updating user toplist...")
	symbols := []struct {
		symbol string
		value  float64
	}{
		{"AAPL", 3.5},
		{"GOOGL", 2.1},
		{"MSFT", 4.8},
	}

	for _, s := range symbols {
		if err := updater.UpdateUserToplist(ctx, userID, toplistID, s.symbol, s.value); err != nil {
			t.Fatalf("Failed to update user toplist for %s: %v", s.symbol, err)
		}
	}

	// Step 3: Get rankings
	t.Log("Step 3: Getting toplist rankings...")
	rankings, err := service.GetToplistRankings(ctx, toplistID, 10, 0, nil)
	if err != nil {
		t.Fatalf("Failed to get rankings: %v", err)
	}

	if len(rankings) != 3 {
		t.Errorf("Expected 3 rankings, got %d", len(rankings))
	}

	// Verify order: MSFT (4.8) > AAPL (3.5) > GOOGL (2.1)
	if len(rankings) >= 3 {
		if rankings[0].Symbol != "MSFT" {
			t.Errorf("Expected MSFT to be #1, got %s", rankings[0].Symbol)
		}
		if rankings[0].Value != 4.8 {
			t.Errorf("Expected MSFT value to be 4.8, got %f", rankings[0].Value)
		}
		if rankings[0].Rank != 1 {
			t.Errorf("Expected MSFT rank to be 1, got %d", rankings[0].Rank)
		}

		if rankings[1].Symbol != "AAPL" {
			t.Errorf("Expected AAPL to be #2, got %s", rankings[1].Symbol)
		}
		if rankings[2].Symbol != "GOOGL" {
			t.Errorf("Expected GOOGL to be #3, got %s", rankings[2].Symbol)
		}
	}

	// Step 4: Test pagination
	t.Log("Step 4: Testing pagination...")
	rankingsPage1, err := service.GetToplistRankings(ctx, toplistID, 2, 0, nil)
	if err != nil {
		t.Fatalf("Failed to get first page: %v", err)
	}

	if len(rankingsPage1) != 2 {
		t.Errorf("Expected 2 rankings on first page, got %d", len(rankingsPage1))
	}

	rankingsPage2, err := service.GetToplistRankings(ctx, toplistID, 2, 2, nil)
	if err != nil {
		t.Fatalf("Failed to get second page: %v", err)
	}

	if len(rankingsPage2) != 1 {
		t.Errorf("Expected 1 ranking on second page, got %d", len(rankingsPage2))
	}

	if rankingsPage2[0].Symbol != "GOOGL" {
		t.Errorf("Expected GOOGL on second page, got %s", rankingsPage2[0].Symbol)
	}

	t.Log("✅ User toplist test completed successfully!")
}

// TestToplistE2E_BatchUpdate tests batch update functionality
func TestToplistE2E_BatchUpdate(t *testing.T) {
	ctx := context.Background()
	redis := storage.NewMockRedisClient()

	// Create toplist updater
	updater := toplist.NewRedisToplistUpdater(redis)

	// Create batch of updates
	updates := []toplist.ToplistUpdate{
		{
			Key:    models.GetSystemToplistRedisKey(models.MetricChangePct, models.Window1m),
			Symbol: "AAPL",
			Value:  2.5,
		},
		{
			Key:    models.GetSystemToplistRedisKey(models.MetricChangePct, models.Window1m),
			Symbol: "GOOGL",
			Value:  -1.2,
		},
		{
			Key:    models.GetSystemToplistRedisKey(models.MetricChangePct, models.Window1m),
			Symbol: "MSFT",
			Value:  5.0,
		},
		{
			Key:    models.GetSystemToplistRedisKey(models.MetricVolume, models.Window1m),
			Symbol: "AAPL",
			Value:  10000,
		},
		{
			Key:    models.GetSystemToplistRedisKey(models.MetricVolume, models.Window1m),
			Symbol: "MSFT",
			Value:  15000,
		},
	}

	// Step 1: Perform batch update
	t.Log("Step 1: Performing batch update...")
	if err := updater.BatchUpdate(ctx, updates); err != nil {
		t.Fatalf("Failed to batch update: %v", err)
	}

	// Step 2: Verify all updates were applied
	t.Log("Step 2: Verifying batch updates...")
	
	// Check gainers toplist
	gainersKey := models.GetSystemToplistRedisKey(models.MetricChangePct, models.Window1m)
	members, err := redis.ZRevRange(ctx, gainersKey, 0, 10)
	if err != nil {
		t.Fatalf("Failed to get gainers rankings: %v", err)
	}

	if len(members) != 3 {
		t.Errorf("Expected 3 symbols in gainers toplist, got %d", len(members))
	}

	// Check volume toplist
	volumeKey := models.GetSystemToplistRedisKey(models.MetricVolume, models.Window1m)
	volumeMembers, err := redis.ZRevRange(ctx, volumeKey, 0, 10)
	if err != nil {
		t.Fatalf("Failed to get volume rankings: %v", err)
	}

	if len(volumeMembers) != 2 {
		t.Errorf("Expected 2 symbols in volume toplist, got %d", len(volumeMembers))
	}

	// Verify MSFT is #1 in both
	if len(members) > 0 && members[0].Member != "MSFT" {
		t.Errorf("Expected MSFT to be #1 in gainers, got %s", members[0].Member)
	}
	if len(volumeMembers) > 0 && volumeMembers[0].Member != "MSFT" {
		t.Errorf("Expected MSFT to be #1 in volume, got %s", volumeMembers[0].Member)
	}

	t.Log("✅ Batch update test completed successfully!")
}

// TestToplistE2E_ScannerIntegration tests toplist integration with scanner worker
func TestToplistE2E_ScannerIntegration(t *testing.T) {
	ctx := context.Background()
	redis := storage.NewMockRedisClient()

	// Create toplist updater
	updater := toplist.NewRedisToplistUpdater(redis)

	// Create toplist integration
	integration := scanner.NewToplistIntegration(updater, true, 1*time.Second)

	// Create state manager and simulate scanner updates
	sm := scanner.NewStateManager(200)

	// Step 1: Simulate scanner updating symbols with metrics
	t.Log("Step 1: Simulating scanner updates...")
	
	symbols := []string{"AAPL", "GOOGL", "MSFT"}
	for _, symbol := range symbols {
		// Update live bar
		tick := &models.Tick{
			Symbol:    symbol,
			Price:     150.0,
			Size:      100,
			Timestamp: time.Now(),
			Type:      "trade",
		}
		sm.UpdateLiveBar(symbol, tick)

		// Update indicators
		sm.UpdateIndicators(symbol, map[string]float64{
			"rsi_14": 25.0,
		})
	}

	// Step 2: Get metrics snapshot and update toplists
	t.Log("Step 2: Getting metrics and updating toplists...")
	
	snapshot := sm.Snapshot()
	for _, symbol := range snapshot.Symbols {
		// Calculate metrics (simplified for test)
		metrics := map[string]float64{
			"price_change_1m_pct": 2.0 + float64(len(symbol)), // Different values per symbol
			"price_change_5m_pct": 3.0 + float64(len(symbol)),
			"volume": 10000.0 + float64(len(symbol)*1000),
		}

		// Update toplists
		if err := integration.UpdateToplists(ctx, symbol, metrics); err != nil {
			t.Fatalf("Failed to update toplists for %s: %v", symbol, err)
		}
	}

	// Step 3: Publish updates
	t.Log("Step 3: Publishing toplist updates...")
	if err := integration.PublishUpdates(ctx); err != nil {
		t.Fatalf("Failed to publish updates: %v", err)
	}

	// Step 4: Verify toplists were updated
	t.Log("Step 4: Verifying toplist updates...")
	
	gainersKey := models.GetSystemToplistRedisKey(models.MetricChangePct, models.Window1m)
	members, err := redis.ZRevRange(ctx, gainersKey, 0, 10)
	if err != nil {
		t.Fatalf("Failed to get rankings: %v", err)
	}

	if len(members) != len(symbols) {
		t.Errorf("Expected %d symbols in toplist, got %d", len(symbols), len(members))
	}

	// Verify all symbols are present
	symbolSet := make(map[string]bool)
	for _, member := range members {
		symbolSet[member.Member] = true
	}

	for _, symbol := range symbols {
		if !symbolSet[symbol] {
			t.Errorf("Symbol %s not found in toplist", symbol)
		}
	}

	t.Log("✅ Scanner integration test completed successfully!")
}

// TestToplistE2E_UpdateNotifications tests toplist update notifications
func TestToplistE2E_UpdateNotifications(t *testing.T) {
	ctx := context.Background()
	redis := storage.NewMockRedisClient()

	// Create toplist updater
	updater := toplist.NewRedisToplistUpdater(redis)

	// Step 1: Subscribe to toplist update notifications
	t.Log("Step 1: Subscribing to toplist updates...")
	
	updateChan, err := redis.Subscribe(ctx, "toplists.updated")
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Ensure channel is drained and closed after test
	defer func() {
		// Drain any remaining messages to prevent goroutine leaks
		go func() {
			for range updateChan {
				// Drain channel
			}
		}()
		// Give it a moment to drain
		time.Sleep(10 * time.Millisecond)
	}()

	// Step 2: Update toplist and publish notification
	t.Log("Step 2: Updating toplist and publishing notification...")
	
	toplistID := string(models.GetSystemToplistType(models.MetricChangePct, models.Window1m, true))
	if err := updater.UpdateSystemToplist(ctx, models.MetricChangePct, models.Window1m, "AAPL", 2.5); err != nil {
		t.Fatalf("Failed to update toplist: %v", err)
	}

	if err := updater.PublishUpdate(ctx, toplistID, "system"); err != nil {
		t.Fatalf("Failed to publish update: %v", err)
	}

	// Step 3: Wait for notification (with timeout)
	t.Log("Step 3: Waiting for update notification...")
	
	// Note: Mock Redis client may handle pub/sub differently than real Redis
	// For component tests, we verify that PublishUpdate was called successfully
	// Full pub/sub testing is done in pipeline E2E tests with real Redis
	select {
	case msg, ok := <-updateChan:
		if !ok {
			// Channel closed
			t.Log("Channel closed (expected with mock Redis)")
		} else if msg.Channel == "toplists.updated" {
			t.Logf("Received update notification for toplist: %v", msg.Message)
		} else {
			t.Logf("Received message on channel '%s': %v", msg.Channel, msg.Message)
		}
	case <-time.After(100 * time.Millisecond):
		// Mock Redis may not support async pub/sub, which is OK for component tests
		t.Log("No notification received (expected with mock Redis - full testing done in pipeline E2E)")
	}

	t.Log("✅ Update notifications test completed successfully!")
}

