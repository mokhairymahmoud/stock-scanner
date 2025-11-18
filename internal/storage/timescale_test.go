package storage

import (
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/config"
	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestWriteConfigFromBarsConfig(t *testing.T) {
	barsConfig := config.BarsConfig{
		DBWriteBatchSize: 2000,
		DBWriteInterval:  2 * time.Second,
		DBWriteQueueSize: 20000,
		DBMaxRetries:     5,
		DBRetryDelay:     200 * time.Millisecond,
	}

	writeConfig := WriteConfigFromBarsConfig(barsConfig)

	assert.Equal(t, 2000, writeConfig.BatchSize)
	assert.Equal(t, 2*time.Second, writeConfig.Interval)
	assert.Equal(t, 20000, writeConfig.QueueSize)
	assert.Equal(t, 5, writeConfig.MaxRetries)
	assert.Equal(t, 200*time.Millisecond, writeConfig.RetryDelay)
}

// Note: Full integration tests for TimescaleDB client would require a real database
// These should be in a separate integration test file that can be run with a test database
// For now, we test the validation and configuration logic

func TestTimescaleDBClient_WriteBars_Validation(t *testing.T) {
	// Test validation logic
	bars := []*models.Bar1m{
		{
			Symbol:    "AAPL",
			Timestamp: time.Now(),
			Open:      150.0,
			High:      151.0,
			Low:       149.0,
			Close:     150.5,
			Volume:    1000,
			VWAP:      150.25,
		},
		{
			// Invalid bar (missing symbol)
			Timestamp: time.Now(),
			Open:      150.0,
			High:      151.0,
			Low:       149.0,
			Close:     150.5,
			Volume:    1000,
			VWAP:      150.25,
		},
	}

	// Test that invalid bars are filtered out
	validBars := make([]*models.Bar1m, 0, len(bars))
	for _, bar := range bars {
		if err := bar.Validate(); err == nil {
			validBars = append(validBars, bar)
		}
	}

	assert.Len(t, validBars, 1)
	assert.Equal(t, "AAPL", validBars[0].Symbol)
}

// Note: Full integration tests would require a real TimescaleDB instance
// These should be in a separate integration test file that can be run
// with a test database

