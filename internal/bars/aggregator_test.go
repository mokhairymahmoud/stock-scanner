package bars

import (
	"sync"
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAggregator_ProcessTick(t *testing.T) {
	agg := NewAggregator()

	tick := &models.Tick{
		Symbol:    "AAPL",
		Price:     150.0,
		Size:      100,
		Timestamp: time.Now(),
		Type:      "trade",
	}

	err := agg.ProcessTick(tick)
	require.NoError(t, err)

	liveBar := agg.GetLiveBar("AAPL")
	require.NotNil(t, liveBar)
	assert.Equal(t, "AAPL", liveBar.Symbol)
	assert.Equal(t, 150.0, liveBar.Open)
	assert.Equal(t, 150.0, liveBar.High)
	assert.Equal(t, 150.0, liveBar.Low)
	assert.Equal(t, 150.0, liveBar.Close)
	assert.Equal(t, int64(100), liveBar.Volume)
}

func TestAggregator_UpdateHighLow(t *testing.T) {
	agg := NewAggregator()
	now := time.Now().Truncate(time.Minute)

	ticks := []*models.Tick{
		{Symbol: "AAPL", Price: 150.0, Size: 100, Timestamp: now, Type: "trade"},
		{Symbol: "AAPL", Price: 151.0, Size: 200, Timestamp: now.Add(10 * time.Second), Type: "trade"},
		{Symbol: "AAPL", Price: 149.0, Size: 50, Timestamp: now.Add(20 * time.Second), Type: "trade"},
		{Symbol: "AAPL", Price: 150.5, Size: 75, Timestamp: now.Add(30 * time.Second), Type: "trade"},
	}

	for _, tick := range ticks {
		err := agg.ProcessTick(tick)
		require.NoError(t, err)
	}

	liveBar := agg.GetLiveBar("AAPL")
	require.NotNil(t, liveBar)
	assert.Equal(t, 150.0, liveBar.Open)
	assert.Equal(t, 151.0, liveBar.High)
	assert.Equal(t, 149.0, liveBar.Low)
	assert.Equal(t, 150.5, liveBar.Close)
	assert.Equal(t, int64(425), liveBar.Volume)
}

func TestAggregator_MinuteBoundaryDetection(t *testing.T) {
	agg := NewAggregator()
	finalizedBars := make([]*models.Bar1m, 0)
	var finalizedMu sync.Mutex

	agg.SetOnBarFinal(func(bar *models.Bar1m) {
		finalizedMu.Lock()
		defer finalizedMu.Unlock()
		finalizedBars = append(finalizedBars, bar)
	})

	now := time.Now().Truncate(time.Minute)

	// Tick in first minute
	tick1 := &models.Tick{
		Symbol:    "AAPL",
		Price:     150.0,
		Size:      100,
		Timestamp: now,
		Type:      "trade",
	}
	err := agg.ProcessTick(tick1)
	require.NoError(t, err)

	// Tick in second minute (should finalize first bar)
	tick2 := &models.Tick{
		Symbol:    "AAPL",
		Price:     151.0,
		Size:      200,
		Timestamp: now.Add(1 * time.Minute),
		Type:      "trade",
	}
	err = agg.ProcessTick(tick2)
	require.NoError(t, err)

	// Wait for callback to complete
	time.Sleep(50 * time.Millisecond)

	finalizedMu.Lock()
	require.Len(t, finalizedBars, 1, "Expected one finalized bar")
	finalizedBar := finalizedBars[0]
	finalizedMu.Unlock()

	assert.Equal(t, "AAPL", finalizedBar.Symbol)
	assert.Equal(t, now, finalizedBar.Timestamp)
	assert.Equal(t, 150.0, finalizedBar.Open)
	assert.Equal(t, 150.0, finalizedBar.Close)
	assert.Equal(t, int64(100), finalizedBar.Volume)

	// Check that new live bar was created
	liveBar := agg.GetLiveBar("AAPL")
	require.NotNil(t, liveBar)
	assert.Equal(t, now.Add(1*time.Minute), liveBar.Timestamp)
	assert.Equal(t, 151.0, liveBar.Open)
}

func TestAggregator_VWAPCalculation(t *testing.T) {
	agg := NewAggregator()
	now := time.Now().Truncate(time.Minute)

	ticks := []*models.Tick{
		{Symbol: "AAPL", Price: 150.0, Size: 100, Timestamp: now, Type: "trade"},
		{Symbol: "AAPL", Price: 151.0, Size: 200, Timestamp: now.Add(10 * time.Second), Type: "trade"},
		{Symbol: "AAPL", Price: 149.0, Size: 300, Timestamp: now.Add(20 * time.Second), Type: "trade"},
	}

	for _, tick := range ticks {
		err := agg.ProcessTick(tick)
		require.NoError(t, err)
	}

	liveBar := agg.GetLiveBar("AAPL")
	require.NotNil(t, liveBar)

	// VWAP = (150*100 + 151*200 + 149*300) / (100+200+300)
	// = (15000 + 30200 + 44700) / 600
	// = 89900 / 600
	// = 149.833...
	expectedVWAP := (150.0*100.0 + 151.0*200.0 + 149.0*300.0) / 600.0

	finalizedBar := liveBar.ToBar1m()
	assert.InDelta(t, expectedVWAP, finalizedBar.VWAP, 0.01)
}

func TestAggregator_MultipleSymbols(t *testing.T) {
	agg := NewAggregator()
	now := time.Now().Truncate(time.Minute)

	// Process ticks for multiple symbols
	ticks := []*models.Tick{
		{Symbol: "AAPL", Price: 150.0, Size: 100, Timestamp: now, Type: "trade"},
		{Symbol: "MSFT", Price: 300.0, Size: 200, Timestamp: now, Type: "trade"},
		{Symbol: "GOOGL", Price: 2500.0, Size: 50, Timestamp: now, Type: "trade"},
	}

	for _, tick := range ticks {
		err := agg.ProcessTick(tick)
		require.NoError(t, err)
	}

	assert.Equal(t, 3, agg.GetSymbolCount())

	aaplBar := agg.GetLiveBar("AAPL")
	require.NotNil(t, aaplBar)
	assert.Equal(t, 150.0, aaplBar.Open)

	msftBar := agg.GetLiveBar("MSFT")
	require.NotNil(t, msftBar)
	assert.Equal(t, 300.0, msftBar.Open)

	googlBar := agg.GetLiveBar("GOOGL")
	require.NotNil(t, googlBar)
	assert.Equal(t, 2500.0, googlBar.Open)
}

func TestAggregator_FinalizeBar(t *testing.T) {
	agg := NewAggregator()
	now := time.Now().Truncate(time.Minute)

	tick := &models.Tick{
		Symbol:    "AAPL",
		Price:     150.0,
		Size:      100,
		Timestamp: now,
		Type:      "trade",
	}

	err := agg.ProcessTick(tick)
	require.NoError(t, err)

	finalizedBar := agg.FinalizeBar("AAPL")
	require.NotNil(t, finalizedBar)
	assert.Equal(t, "AAPL", finalizedBar.Symbol)
	assert.Equal(t, 150.0, finalizedBar.Open)
	assert.Equal(t, 150.0, finalizedBar.Close)

	// Bar should be removed after finalization
	assert.Nil(t, agg.GetLiveBar("AAPL"))
	assert.Equal(t, 0, agg.GetSymbolCount())
}

func TestAggregator_FinalizeAllBars(t *testing.T) {
	agg := NewAggregator()
	now := time.Now().Truncate(time.Minute)

	ticks := []*models.Tick{
		{Symbol: "AAPL", Price: 150.0, Size: 100, Timestamp: now, Type: "trade"},
		{Symbol: "MSFT", Price: 300.0, Size: 200, Timestamp: now, Type: "trade"},
		{Symbol: "GOOGL", Price: 2500.0, Size: 50, Timestamp: now, Type: "trade"},
	}

	for _, tick := range ticks {
		err := agg.ProcessTick(tick)
		require.NoError(t, err)
	}

	assert.Equal(t, 3, agg.GetSymbolCount())

	finalizedBars := agg.FinalizeAllBars()
	assert.Len(t, finalizedBars, 3)
	assert.Equal(t, 0, agg.GetSymbolCount())
}

func TestAggregator_ThreadSafety(t *testing.T) {
	agg := NewAggregator()
	now := time.Now().Truncate(time.Minute)

	var wg sync.WaitGroup
	symbols := []string{"AAPL", "MSFT", "GOOGL", "TSLA", "AMZN"}

	// Concurrently process ticks for different symbols
	for _, symbol := range symbols {
		wg.Add(1)
		go func(sym string) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				tick := &models.Tick{
					Symbol:    sym,
					Price:     100.0 + float64(i),
					Size:      int64(100 + i),
					Timestamp: now.Add(time.Duration(i) * time.Second),
					Type:      "trade",
				}
				_ = agg.ProcessTick(tick)
			}
		}(symbol)
	}

	wg.Wait()

	// Verify all symbols have live bars
	assert.Equal(t, len(symbols), agg.GetSymbolCount())

	for _, symbol := range symbols {
		liveBar := agg.GetLiveBar(symbol)
		require.NotNil(t, liveBar)
		assert.Equal(t, symbol, liveBar.Symbol)
	}
}

func TestAggregator_InvalidTick(t *testing.T) {
	agg := NewAggregator()

	// Invalid tick (missing symbol)
	invalidTick := &models.Tick{
		Price:     150.0,
		Size:      100,
		Timestamp: time.Now(),
		Type:      "trade",
	}

	err := agg.ProcessTick(invalidTick)
	assert.Error(t, err)

	// Should not create a live bar
	assert.Nil(t, agg.GetLiveBar(""))
}

func TestAggregator_GetAllLiveBars(t *testing.T) {
	agg := NewAggregator()
	now := time.Now().Truncate(time.Minute)

	ticks := []*models.Tick{
		{Symbol: "AAPL", Price: 150.0, Size: 100, Timestamp: now, Type: "trade"},
		{Symbol: "MSFT", Price: 300.0, Size: 200, Timestamp: now, Type: "trade"},
	}

	for _, tick := range ticks {
		err := agg.ProcessTick(tick)
		require.NoError(t, err)
	}

	allBars := agg.GetAllLiveBars()
	assert.Len(t, allBars, 2)
	assert.NotNil(t, allBars["AAPL"])
	assert.NotNil(t, allBars["MSFT"])

	// Modifying returned bars should not affect internal state
	allBars["AAPL"].Open = 999.0
	aaplBar := agg.GetLiveBar("AAPL")
	assert.Equal(t, 150.0, aaplBar.Open) // Original value unchanged
}

