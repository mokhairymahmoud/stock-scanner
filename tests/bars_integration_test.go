package data

import (
	"sync"
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/bars"
	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBarsService_Integration(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup aggregator
	agg := bars.NewAggregator()

	// Setup mock Redis client
	mockRedis := storage.NewMockRedisClient()

	// Setup publisher
	publisherConfig := bars.DefaultPublisherConfig()
	publisher := bars.NewPublisher(mockRedis, publisherConfig)
	publisher.Start()
	defer publisher.Stop()

	// Setup mock bar storage
	mockBarStorage := &storage.MockBarStorage{}
	publisher.SetBarStorage(mockBarStorage)

	// Track finalized bars
	finalizedBars := make([]*models.Bar1m, 0)
	var finalizedMu sync.Mutex
	finalizedBarChan := make(chan *models.Bar1m, 10)

	agg.SetOnBarFinal(func(bar *models.Bar1m) {
		finalizedMu.Lock()
		finalizedBars = append(finalizedBars, bar)
		finalizedMu.Unlock()
		finalizedBarChan <- bar
		publisher.PublishFinalizedBar(bar)
	})

	agg.SetOnBarUpdate(func(liveBar *models.LiveBar) {
		publisher.PublishLiveBar(liveBar)
	})

	// Process ticks directly (simulating consumer)
	now := time.Now().Truncate(time.Minute)
	ticks := []*models.Tick{
		{Symbol: "AAPL", Price: 150.0, Size: 100, Timestamp: now, Type: "trade"},
		{Symbol: "AAPL", Price: 151.0, Size: 200, Timestamp: now.Add(10 * time.Second), Type: "trade"},
		{Symbol: "AAPL", Price: 149.0, Size: 50, Timestamp: now.Add(20 * time.Second), Type: "trade"},
	}

	for _, tick := range ticks {
		err := agg.ProcessTick(tick)
		require.NoError(t, err)
	}

	// Wait for callbacks
	time.Sleep(100 * time.Millisecond)

	// Verify aggregator has live bar
	liveBar := agg.GetLiveBar("AAPL")
	require.NotNil(t, liveBar)
	assert.Equal(t, "AAPL", liveBar.Symbol)
	assert.Equal(t, 151.0, liveBar.High)
	assert.Equal(t, 149.0, liveBar.Low)

	// Process tick in next minute to trigger finalization
	nextMinuteTick := &models.Tick{
		Symbol:    "AAPL",
		Price:     160.0,
		Size:      100,
		Timestamp: now.Add(1 * time.Minute),
		Type:      "trade",
	}
	err := agg.ProcessTick(nextMinuteTick)
	require.NoError(t, err)

	// Wait for finalization with timeout
	select {
	case <-finalizedBarChan:
		// Bar was finalized
		finalizedMu.Lock()
		assert.Greater(t, len(finalizedBars), 0, "Should have finalized at least one bar")
		finalizedMu.Unlock()

		// Wait for storage write
		time.Sleep(200 * time.Millisecond)

		// Verify bars were written to storage
		assert.Greater(t, len(mockBarStorage.Bars), 0, "Bars should be written to storage")
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for bar finalization")
	}
}

func TestBarsService_HealthCheck(t *testing.T) {
	// This would test the health check endpoint
	// For now, we test the components individually
	mockRedis := storage.NewMockRedisClient()
	agg := bars.NewAggregator()
	publisher := bars.NewPublisher(mockRedis, bars.DefaultPublisherConfig())
	publisher.Start()
	defer publisher.Stop()

	// Test that components are running
	assert.True(t, publisher.IsRunning())
	assert.Equal(t, 0, agg.GetSymbolCount())
}

