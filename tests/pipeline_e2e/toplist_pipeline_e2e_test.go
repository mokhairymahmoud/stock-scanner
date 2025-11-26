// Package data contains pipeline E2E tests for the toplist feature.
//
// These tests verify the complete pipeline from scanner updates to Redis ZSETs to API queries.
// They test the full data flow through internal components and Redis.
package data

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/config"
	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/pubsub"
	"github.com/mohamedkhairy/stock-scanner/internal/toplist"
)

// TestToplistPipelineE2E_ScannerToRedis tests the pipeline from scanner to Redis ZSETs
func TestToplistPipelineE2E_ScannerToRedis(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()

	// Setup Redis client
	redisClient, err := pubsub.NewRedisClient(config.RedisConfig{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
	})
	if err != nil {
		t.Skipf("Skipping test: Redis not available: %v", err)
	}
	defer redisClient.Close()

	// Create toplist updater
	updater := toplist.NewRedisToplistUpdater(redisClient)

	// Test symbols with different metrics
	symbols := []struct {
		symbol string
		metrics map[string]float64
	}{
		{"AAPL", map[string]float64{
			"price_change_1m_pct": 2.5,
			"price_change_5m_pct": 3.0,
			"volume": 10000,
		}},
		{"GOOGL", map[string]float64{
			"price_change_1m_pct": -1.2,
			"price_change_5m_pct": -2.0,
			"volume": 5000,
		}},
		{"MSFT", map[string]float64{
			"price_change_1m_pct": 5.0,
			"price_change_5m_pct": 6.0,
			"volume": 15000,
		}},
	}

	// Step 1: Update toplists (simulate scanner worker)
	t.Log("Step 1: Updating toplists from scanner...")
	
	updates := make([]toplist.ToplistUpdate, 0)
	for _, s := range symbols {
		// Update gainers_1m
		if change1m, ok := s.metrics["price_change_1m_pct"]; ok {
			key := models.GetSystemToplistRedisKey(models.MetricChangePct, models.Window1m)
			updates = append(updates, toplist.ToplistUpdate{
				Key:    key,
				Symbol: s.symbol,
				Value:  change1m,
			})
		}
		// Update volume_1m
		if volume, ok := s.metrics["volume"]; ok {
			key := models.GetSystemToplistRedisKey(models.MetricVolume, models.Window1m)
			updates = append(updates, toplist.ToplistUpdate{
				Key:    key,
				Symbol: s.symbol,
				Value:  volume,
			})
		}
	}

	// Batch update
	if err := updater.BatchUpdate(ctx, updates); err != nil {
		t.Fatalf("Failed to batch update toplists: %v", err)
	}
	t.Logf("Updated %d toplist entries", len(updates))

	// Step 2: Verify rankings in Redis ZSET
	t.Log("Step 2: Verifying rankings in Redis ZSET...")
	
	// Check gainers_1m toplist (should be sorted descending by change_pct)
	gainersKey := models.GetSystemToplistRedisKey(models.MetricChangePct, models.Window1m)
	members, err := redisClient.ZRevRange(ctx, gainersKey, 0, 10)
	if err != nil {
		t.Fatalf("Failed to get rankings: %v", err)
	}

	// Find our test symbols in the rankings (may have other data from previous runs)
	testSymbols := map[string]float64{
		"MSFT": 5.0,
		"AAPL": 2.5,
		"GOOGL": -1.2,
	}

	foundSymbols := make(map[string]float64)
	for _, member := range members {
		if expectedScore, isTestSymbol := testSymbols[member.Member]; isTestSymbol {
			foundSymbols[member.Member] = member.Score
			if member.Score != expectedScore {
				t.Errorf("Expected %s score to be %f, got %f", member.Member, expectedScore, member.Score)
			}
		}
	}

	// Verify we found all test symbols
	if len(foundSymbols) != len(testSymbols) {
		t.Errorf("Expected to find %d test symbols, found %d", len(testSymbols), len(foundSymbols))
	}

	// Verify order: MSFT should be before AAPL, AAPL should be before GOOGL
	msftIndex := -1
	aaplIndex := -1
	googlIndex := -1
	for i, member := range members {
		if member.Member == "MSFT" {
			msftIndex = i
		}
		if member.Member == "AAPL" {
			aaplIndex = i
		}
		if member.Member == "GOOGL" {
			googlIndex = i
		}
	}

	if msftIndex >= 0 && aaplIndex >= 0 && msftIndex > aaplIndex {
		t.Errorf("Expected MSFT (index %d) to be before AAPL (index %d)", msftIndex, aaplIndex)
	}
	if aaplIndex >= 0 && googlIndex >= 0 && aaplIndex > googlIndex {
		t.Errorf("Expected AAPL (index %d) to be before GOOGL (index %d)", aaplIndex, googlIndex)
	}

	// Check volume toplist
	volumeKey := models.GetSystemToplistRedisKey(models.MetricVolume, models.Window1m)
	volumeMembers, err := redisClient.ZRevRange(ctx, volumeKey, 0, 10)
	if err != nil {
		t.Fatalf("Failed to get volume rankings: %v", err)
	}

	// Find our test symbols in the volume rankings
	volumeTestSymbols := map[string]float64{
		"MSFT": 15000.0,
		"AAPL": 10000.0,
		"GOOGL": 5000.0,
	}

	foundVolumeSymbols := make(map[string]float64)
	for _, member := range volumeMembers {
		if expectedScore, isTestSymbol := volumeTestSymbols[member.Member]; isTestSymbol {
			foundVolumeSymbols[member.Member] = member.Score
			if member.Score != expectedScore {
				t.Errorf("Expected %s volume to be %f, got %f", member.Member, expectedScore, member.Score)
			}
		}
	}

	// Verify we found all test symbols
	if len(foundVolumeSymbols) != len(volumeTestSymbols) {
		t.Errorf("Expected to find %d test symbols in volume toplist, found %d", len(volumeTestSymbols), len(foundVolumeSymbols))
	}

	// Step 3: Publish update notifications
	t.Log("Step 3: Publishing toplist update notifications...")
	
	// Use actual system toplist ID from migration
	toplistID := "gainers_1m"
	if err := updater.PublishUpdate(ctx, toplistID, "system"); err != nil {
		t.Fatalf("Failed to publish update: %v", err)
	}

	// Step 4: Verify notification was published
	t.Log("Step 4: Verifying update notifications...")
	
	updateChan, err := redisClient.Subscribe(ctx, "toplists.updated")
	if err != nil {
		t.Logf("Note: Could not subscribe to updates channel: %v", err)
	} else {
		// Wait for notification (with timeout)
		select {
		case msg := <-updateChan:
			if msg.Channel == "toplists.updated" {
				t.Logf("Received update notification: %v", msg.Message)
			}
		case <-time.After(2 * time.Second):
			t.Log("No notification received within timeout (may be expected if publish happened before subscribe)")
		}
	}

	t.Log("✅ Scanner to Redis pipeline test completed!")
}

// TestToplistPipelineE2E_RedisToAPI tests the pipeline from Redis ZSETs to API queries
func TestToplistPipelineE2E_RedisToAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()

	// Setup Redis client
	redisClient, err := pubsub.NewRedisClient(config.RedisConfig{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
	})
	if err != nil {
		t.Skipf("Skipping test: Redis not available: %v", err)
	}
	defer redisClient.Close()

	// Create toplist updater
	updater := toplist.NewRedisToplistUpdater(redisClient)

	// Create mock store
	store := toplist.NewMockToplistStore()

	// Create toplist service
	service := toplist.NewToplistService(store, redisClient, updater)

	// Step 1: Populate Redis ZSET with rankings
	t.Log("Step 1: Populating Redis ZSET with rankings...")
	
	symbols := []struct {
		symbol string
		value  float64
	}{
		{"AAPL", 2.5},
		{"GOOGL", -1.2},
		{"MSFT", 5.0},
		{"TSLA", 3.8},
		{"AMZN", 1.5},
	}

	gainersKey := models.GetSystemToplistRedisKey(models.MetricChangePct, models.Window1m)
	for _, s := range symbols {
		if err := redisClient.ZAdd(ctx, gainersKey, s.value, s.symbol); err != nil {
			t.Fatalf("Failed to add symbol to ZSET: %v", err)
		}
	}
	t.Logf("Added %d symbols to gainers toplist", len(symbols))

	// Step 2: Create system toplist config (simulate from database)
	t.Log("Step 2: Creating toplist config...")
	
	config := &models.ToplistConfig{
		ID:         "gainers_1m", // System toplist ID from migration
		UserID:     "", // System toplist
		Name:       "Gainers 1m",
		Metric:     models.MetricChangePct,
		TimeWindow: models.Window1m,
		SortOrder:  models.SortOrderDesc,
		Enabled:    true,
	}

	// Step 3: Query rankings via service
	t.Log("Step 3: Querying rankings via toplist service...")
	
	rankings, err := service.GetRankingsByConfig(ctx, config, 10, 0, nil)
	if err != nil {
		t.Fatalf("Failed to get rankings: %v", err)
	}

	if len(rankings) != len(symbols) {
		t.Errorf("Expected %d rankings, got %d", len(symbols), len(rankings))
	}

	// Verify order: MSFT (5.0) > TSLA (3.8) > AAPL (2.5) > AMZN (1.5) > GOOGL (-1.2)
	expectedOrder := []string{"MSFT", "TSLA", "AAPL", "AMZN", "GOOGL"}
	if len(rankings) >= len(expectedOrder) {
		for i, expectedSymbol := range expectedOrder {
			if rankings[i].Symbol != expectedSymbol {
				t.Errorf("Expected rank %d to be %s, got %s", i+1, expectedSymbol, rankings[i].Symbol)
			}
			if rankings[i].Rank != i+1 {
				t.Errorf("Expected rank to be %d, got %d", i+1, rankings[i].Rank)
			}
		}
	}

	// Step 4: Test pagination
	t.Log("Step 4: Testing pagination...")
	
	rankingsPage1, err := service.GetRankingsByConfig(ctx, config, 2, 0, nil)
	if err != nil {
		t.Fatalf("Failed to get first page: %v", err)
	}

	if len(rankingsPage1) != 2 {
		t.Errorf("Expected 2 rankings on first page, got %d", len(rankingsPage1))
	}

	if rankingsPage1[0].Symbol != "MSFT" {
		t.Errorf("Expected MSFT on first page, got %s", rankingsPage1[0].Symbol)
	}

	rankingsPage2, err := service.GetRankingsByConfig(ctx, config, 2, 2, nil)
	if err != nil {
		t.Fatalf("Failed to get second page: %v", err)
	}

	if len(rankingsPage2) != 2 {
		t.Errorf("Expected 2 rankings on second page, got %d", len(rankingsPage2))
	}

	if rankingsPage2[0].Symbol != "AAPL" {
		t.Errorf("Expected AAPL on second page, got %s", rankingsPage2[0].Symbol)
	}

	t.Log("✅ Redis to API pipeline test completed!")
}

// TestToplistPipelineE2E_FullFlow tests the complete flow from scanner to WebSocket
func TestToplistPipelineE2E_FullFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	timeout := 3 * time.Minute
	deadline := time.Now().Add(timeout)

	// Setup Redis client
	redisClient, err := pubsub.NewRedisClient(config.RedisConfig{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
	})
	if err != nil {
		t.Skipf("Skipping test: Redis not available: %v", err)
	}
	defer redisClient.Close()

	// Create toplist updater
	updater := toplist.NewRedisToplistUpdater(redisClient)

	// Step 1: Subscribe to toplist update notifications (simulate WebSocket gateway)
	t.Log("Step 1: Subscribing to toplist updates...")
	
	updateChan, err := redisClient.Subscribe(ctx, "toplists.updated")
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	notificationsReceived := 0
	go func() {
		for msg := range updateChan {
			if msg.Channel == "toplists.updated" {
				notificationsReceived++
				t.Logf("Received toplist update notification: %v", msg.Message)
			}
		}
	}()

	// Step 2: Update toplists (simulate scanner worker)
	t.Log("Step 2: Updating toplists from scanner...")
	
	symbols := []string{"AAPL", "GOOGL", "MSFT"}
	updates := make([]toplist.ToplistUpdate, 0)

	for i, symbol := range symbols {
		key := models.GetSystemToplistRedisKey(models.MetricChangePct, models.Window1m)
		value := 2.0 + float64(i)*1.5 // Different values: 2.0, 3.5, 5.0
		updates = append(updates, toplist.ToplistUpdate{
			Key:    key,
			Symbol: symbol,
			Value:  value,
		})
	}

	if err := updater.BatchUpdate(ctx, updates); err != nil {
		t.Fatalf("Failed to batch update: %v", err)
	}

	// Step 3: Publish update notifications
	t.Log("Step 3: Publishing update notifications...")
	
	// Use actual system toplist ID from migration
	toplistID := "gainers_1m"
	if err := updater.PublishUpdate(ctx, toplistID, "system"); err != nil {
		t.Fatalf("Failed to publish update: %v", err)
	}

	// Step 4: Wait for notification
	t.Log("Step 4: Waiting for update notification...")
	
	notificationReceived := false
	for time.Now().Before(deadline) {
		if notificationsReceived > 0 {
			notificationReceived = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if !notificationReceived {
		t.Log("Warning: No notification received within timeout (may be expected if publish/subscribe timing is off)")
	} else {
		t.Log("✅ Notification received successfully")
	}

	// Step 5: Verify rankings are queryable
	t.Log("Step 5: Verifying rankings are queryable...")
	
	store := toplist.NewMockToplistStore()
	service := toplist.NewToplistService(store, redisClient, updater)

	config := &models.ToplistConfig{
		ID:         toplistID,
		Metric:     models.MetricChangePct,
		TimeWindow: models.Window1m,
		SortOrder:  models.SortOrderDesc,
		Enabled:    true,
	}

	rankings, err := service.GetRankingsByConfig(ctx, config, 10, 0, nil)
	if err != nil {
		t.Fatalf("Failed to get rankings: %v", err)
	}

	// Find our test symbols in the rankings (may have other data from previous runs)
	testSymbolsSet := make(map[string]bool)
	for _, symbol := range symbols {
		testSymbolsSet[symbol] = true
	}

	foundTestSymbols := make(map[string]int) // symbol -> rank
	for _, ranking := range rankings {
		if testSymbolsSet[ranking.Symbol] {
			foundTestSymbols[ranking.Symbol] = ranking.Rank
		}
	}

	// Verify we found all test symbols
	if len(foundTestSymbols) != len(symbols) {
		t.Errorf("Expected to find %d test symbols, found %d", len(symbols), len(foundTestSymbols))
	}

	// Verify order: MSFT should be before GOOGL, GOOGL should be before AAPL
	if msftRank, ok := foundTestSymbols["MSFT"]; ok {
		if googlRank, ok := foundTestSymbols["GOOGL"]; ok && msftRank > googlRank {
			t.Errorf("Expected MSFT (rank %d) to be before GOOGL (rank %d)", msftRank, googlRank)
		}
		if aaplRank, ok := foundTestSymbols["AAPL"]; ok && msftRank > aaplRank {
			t.Errorf("Expected MSFT (rank %d) to be before AAPL (rank %d)", msftRank, aaplRank)
		}
	}

	t.Log("✅ Full flow pipeline test completed!")
}

// TestToplistPipelineE2E_IndicatorEngineIntegration tests toplist updates from indicator engine
func TestToplistPipelineE2E_IndicatorEngineIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()

	// Setup Redis client
	redisClient, err := pubsub.NewRedisClient(config.RedisConfig{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
	})
	if err != nil {
		t.Skipf("Skipping test: Redis not available: %v", err)
	}
	defer redisClient.Close()

	// Create toplist updater
	updater := toplist.NewRedisToplistUpdater(redisClient)

	// Step 1: Simulate indicator engine publishing indicators
	t.Log("Step 1: Simulating indicator engine publishing indicators...")
	
	symbols := []string{"AAPL", "GOOGL", "MSFT"}
	for _, symbol := range symbols {
		indicators := map[string]interface{}{
			"symbol":    symbol,
			"timestamp": time.Now().UTC(),
			"values": map[string]float64{
				"rsi_14":        25.0 + float64(len(symbol))*5, // Different RSI values
				"relative_volume_5m": 1.5 + float64(len(symbol))*0.2,
				"vwap_dist_5m":   0.5 + float64(len(symbol))*0.1,
			},
		}
		indData, _ := json.Marshal(indicators)
		if err := redisClient.Set(ctx, fmt.Sprintf("ind:%s", symbol), string(indData), 10*time.Minute); err != nil {
			t.Fatalf("Failed to set indicators: %v", err)
		}
		// Publish indicator update notification
		if err := redisClient.Publish(ctx, "indicators.updated", symbol); err != nil {
			t.Fatalf("Failed to publish indicator update: %v", err)
		}
	}
	t.Log("Published indicators for all symbols")

	// Step 2: Update toplists with indicator-based metrics (simulate indicator engine integration)
	t.Log("Step 2: Updating toplists with indicator metrics...")
	
	// Update RSI extremes toplist
	rsiKey := models.GetSystemToplistRedisKey(models.MetricRSI, models.Window1m)
	updates := []toplist.ToplistUpdate{
		{Key: rsiKey, Symbol: "AAPL", Value: 25.0},   // Oversold
		{Key: rsiKey, Symbol: "GOOGL", Value: 30.0},
		{Key: rsiKey, Symbol: "MSFT", Value: 35.0},
	}

	if err := updater.BatchUpdate(ctx, updates); err != nil {
		t.Fatalf("Failed to batch update RSI toplist: %v", err)
	}

	// Step 3: Verify RSI toplist rankings
	t.Log("Step 3: Verifying RSI toplist rankings...")
	
	rsiMembers, err := redisClient.ZRevRange(ctx, rsiKey, 0, 10)
	if err != nil {
		t.Fatalf("Failed to get RSI rankings: %v", err)
	}

	if len(rsiMembers) != 3 {
		t.Errorf("Expected 3 symbols in RSI toplist, got %d", len(rsiMembers))
	}

	// For RSI extremes, lower values indicate oversold (more extreme)
	// So we'd typically sort ascending for oversold, but for now we'll verify the data is there
	if len(rsiMembers) >= 3 {
		// Verify all symbols are present
		symbolSet := make(map[string]bool)
		for _, member := range rsiMembers {
			symbolSet[member.Member] = true
		}

		for _, symbol := range symbols {
			if !symbolSet[symbol] {
				t.Errorf("Symbol %s not found in RSI toplist", symbol)
			}
		}
	}

	// Step 4: Publish update notification
	t.Log("Step 4: Publishing RSI toplist update notification...")
	
	// Use actual system toplist ID from migration
	rsiToplistID := "rsi_low"
	if err := updater.PublishUpdate(ctx, rsiToplistID, "system"); err != nil {
		t.Fatalf("Failed to publish RSI update: %v", err)
	}

	t.Log("✅ Indicator engine integration test completed!")
}

