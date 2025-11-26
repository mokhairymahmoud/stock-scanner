package bars

import (
	"sync"
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublisher_PublishFinalizedBar(t *testing.T) {
	mockRedis := storage.NewMockRedisClient()
	config := DefaultPublisherConfig()
	config.BatchSize = 2 // Small batch for testing
	config.BatchTimeout = 50 * time.Millisecond
	publisher := NewPublisher(mockRedis, config)

	err := publisher.Start()
	require.NoError(t, err)
	defer publisher.Stop()

	bar1 := &models.Bar1m{
		Symbol:    "AAPL",
		Timestamp: time.Now().Truncate(time.Minute),
		Open:      150.0,
		High:      151.0,
		Low:       149.0,
		Close:     150.5,
		Volume:    1000,
		VWAP:      150.25,
	}

	bar2 := &models.Bar1m{
		Symbol:    "MSFT",
		Timestamp: time.Now().Truncate(time.Minute),
		Open:      300.0,
		High:      301.0,
		Low:       299.0,
		Close:     300.5,
		Volume:    2000,
		VWAP:      300.25,
	}

	// Publish first bar (should not flush yet)
	err = publisher.PublishFinalizedBar(bar1)
	require.NoError(t, err)

	// Publish second bar (should trigger flush)
	err = publisher.PublishFinalizedBar(bar2)
	require.NoError(t, err)

	// Wait for batch to be processed
	time.Sleep(100 * time.Millisecond)

	// Verify bars were published to stream
	assert.Greater(t, len(mockRedis.StreamData), 0, "Bars should be published to stream")
}

func TestPublisher_BatchFlush(t *testing.T) {
	mockRedis := storage.NewMockRedisClient()
	config := DefaultPublisherConfig()
	config.BatchSize = 5
	config.BatchTimeout = 100 * time.Millisecond
	publisher := NewPublisher(mockRedis, config)

	err := publisher.Start()
	require.NoError(t, err)
	defer publisher.Stop()

	// Publish 3 bars (less than batch size)
	for i := 0; i < 3; i++ {
		bar := &models.Bar1m{
			Symbol:    "AAPL",
			Timestamp: time.Now().Truncate(time.Minute).Add(time.Duration(i) * time.Minute),
			Open:      150.0 + float64(i),
			High:      151.0 + float64(i),
			Low:       149.0 + float64(i),
			Close:     150.5 + float64(i),
			Volume:    1000,
			VWAP:      150.25 + float64(i),
		}
		err = publisher.PublishFinalizedBar(bar)
		require.NoError(t, err)
	}

	// Wait for timeout flush
	time.Sleep(150 * time.Millisecond)

	// Verify bars were flushed
	assert.Greater(t, len(mockRedis.StreamData), 0, "Bars should be flushed on timeout")
}

func TestPublisher_InvalidBar(t *testing.T) {
	mockRedis := storage.NewMockRedisClient()
	config := DefaultPublisherConfig()
	publisher := NewPublisher(mockRedis, config)

	// Invalid bar (missing symbol)
	invalidBar := &models.Bar1m{
		Timestamp: time.Now(),
		Open:      150.0,
		High:      151.0,
		Low:       149.0,
		Close:     150.5,
		Volume:    1000,
	}

	err := publisher.PublishFinalizedBar(invalidBar)
	assert.Error(t, err, "Should reject invalid bar")
}

func TestPublisher_IntegrationWithAggregator(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	mockRedis := storage.NewMockRedisClient()
	agg := NewAggregator()
	publisher := NewPublisher(mockRedis, DefaultPublisherConfig())

	finalizedBars := make([]*models.Bar1m, 0)
	var finalizedMu sync.Mutex
	finalizedBarChan := make(chan *models.Bar1m, 10)

	// Set up aggregator to publish finalized bars
	agg.SetOnBarFinal(func(bar *models.Bar1m) {
		finalizedMu.Lock()
		finalizedBars = append(finalizedBars, bar)
		finalizedMu.Unlock()
		finalizedBarChan <- bar
		publisher.PublishFinalizedBar(bar)
	})

	err := publisher.Start()
	require.NoError(t, err)
	defer publisher.Stop()

	// Process ticks
	now := time.Now().Truncate(time.Minute)
	ticks := []*models.Tick{
		{Symbol: "AAPL", Price: 150.0, Size: 100, Timestamp: now, Type: "trade"},
		{Symbol: "AAPL", Price: 151.0, Size: 200, Timestamp: now.Add(10 * time.Second), Type: "trade"},
		{Symbol: "AAPL", Price: 149.0, Size: 50, Timestamp: now.Add(20 * time.Second), Type: "trade"},
	}

	for _, tick := range ticks {
		err = agg.ProcessTick(tick)
		require.NoError(t, err)
	}

	// Wait a bit for callbacks
	time.Sleep(100 * time.Millisecond)

	// Verify live bar in aggregator (source of truth)
	liveBar := agg.GetLiveBar("AAPL")
	require.NotNil(t, liveBar, "Live bar should exist in aggregator")
	assert.Equal(t, "AAPL", liveBar.Symbol)
	// The live bar should have the correct high/low from the ticks processed
	assert.Equal(t, 151.0, liveBar.High, "High should be 151.0")
	assert.Equal(t, 149.0, liveBar.Low, "Low should be 149.0")
	assert.Equal(t, int64(350), liveBar.Volume, "Volume should be 350")

	// Process tick in next minute to trigger finalization
	nextMinuteTick := &models.Tick{
		Symbol:    "AAPL",
		Price:     152.0,
		Size:      100,
		Timestamp: now.Add(1 * time.Minute),
		Type:      "trade",
	}
	err = agg.ProcessTick(nextMinuteTick)
	require.NoError(t, err)

	// Wait for finalization with timeout
	select {
	case <-finalizedBarChan:
		// Bar was finalized
		finalizedMu.Lock()
		assert.Greater(t, len(finalizedBars), 0, "Should have finalized at least one bar")
		finalizedMu.Unlock()
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for bar finalization")
	}
}

