package data

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizer_AlpacaTrade(t *testing.T) {
	normalizer := NewNormalizer("alpaca")

	// Alpaca trade message
	message := map[string]interface{}{
		"T": "t",           // Type: trade
		"S": "AAPL",        // Symbol
		"p": 150.50,        // Price
		"s": 100,           // Size
		"t": "2023-01-01T12:00:00Z",
	}

	msgBytes, err := json.Marshal(message)
	require.NoError(t, err)

	tick, err := normalizer.Normalize(msgBytes)
	require.NoError(t, err)

	assert.Equal(t, "AAPL", tick.Symbol)
	assert.Equal(t, 150.50, tick.Price)
	assert.Equal(t, int64(100), tick.Size)
	assert.Equal(t, "trade", tick.Type)
	assert.False(t, tick.Timestamp.IsZero())
	assert.NoError(t, tick.Validate())
}

func TestNormalizer_AlpacaQuote(t *testing.T) {
	normalizer := NewNormalizer("alpaca")

	// Alpaca quote message
	message := map[string]interface{}{
		"T": "q",           // Type: quote
		"S": "MSFT",        // Symbol
		"ap": 250.75,       // Ask price
		"bp": 250.50,       // Bid price
		"t": "2023-01-01T12:00:00Z",
	}

	msgBytes, err := json.Marshal(message)
	require.NoError(t, err)

	tick, err := normalizer.Normalize(msgBytes)
	require.NoError(t, err)

	assert.Equal(t, "MSFT", tick.Symbol)
	assert.Equal(t, 250.75, tick.Price) // Uses ask price
	assert.Equal(t, 250.75, tick.Ask)
	assert.Equal(t, 250.50, tick.Bid)
	assert.Equal(t, "quote", tick.Type)
	assert.NoError(t, tick.Validate())
}

func TestNormalizer_PolygonTrade(t *testing.T) {
	normalizer := NewNormalizer("polygon")

	// Polygon trade message
	message := map[string]interface{}{
		"ev": "T",                    // Event: Trade
		"sym": "GOOGL",               // Symbol
		"p": 140.25,                  // Price
		"s": 200,                     // Size
		"t": int64(1672574400000000000), // Timestamp in nanoseconds
	}

	msgBytes, err := json.Marshal(message)
	require.NoError(t, err)

	tick, err := normalizer.Normalize(msgBytes)
	require.NoError(t, err)

	assert.Equal(t, "GOOGL", tick.Symbol)
	assert.Equal(t, 140.25, tick.Price)
	assert.Equal(t, int64(200), tick.Size)
	assert.Equal(t, "trade", tick.Type)
	assert.False(t, tick.Timestamp.IsZero())
	assert.NoError(t, tick.Validate())
}

func TestNormalizer_MockFormat(t *testing.T) {
	normalizer := NewNormalizer("mock")

	message := map[string]interface{}{
		"symbol":    "TSLA",
		"price":     300.50,
		"size":      150,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"type":      "trade",
	}

	msgBytes, err := json.Marshal(message)
	require.NoError(t, err)

	tick, err := normalizer.Normalize(msgBytes)
	require.NoError(t, err)

	assert.Equal(t, "TSLA", tick.Symbol)
	assert.Equal(t, 300.50, tick.Price)
	assert.Equal(t, int64(150), tick.Size)
	assert.Equal(t, "trade", tick.Type)
	assert.NoError(t, tick.Validate())
}

func TestNormalizer_GenericFormat(t *testing.T) {
	normalizer := NewNormalizer("unknown")

	// Generic format with common field names
	message := map[string]interface{}{
		"symbol":    "AMZN",
		"price":     120.75,
		"size":      300,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	msgBytes, err := json.Marshal(message)
	require.NoError(t, err)

	tick, err := normalizer.Normalize(msgBytes)
	require.NoError(t, err)

	assert.Equal(t, "AMZN", tick.Symbol)
	assert.Equal(t, 120.75, tick.Price)
	assert.Equal(t, int64(300), tick.Size)
	assert.NoError(t, tick.Validate())
}

func TestNormalizer_InvalidMessage(t *testing.T) {
	normalizer := NewNormalizer("alpaca")

	// Empty message
	_, err := normalizer.Normalize([]byte{})
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidMessage)

	// Invalid JSON
	_, err = normalizer.Normalize([]byte("not json"))
	assert.Error(t, err)

	// Missing required fields
	message := map[string]interface{}{
		"T": "t",
		// Missing symbol and price
	}
	msgBytes, _ := json.Marshal(message)
	_, err = normalizer.Normalize(msgBytes)
	assert.Error(t, err)
}

func TestNormalizer_TimestampNormalization(t *testing.T) {
	normalizer := NewNormalizer("alpaca")

	// Test RFC3339 format
	message1 := map[string]interface{}{
		"T": "t",
		"S": "AAPL",
		"p": 150.0,
		"s": 100,
		"t": "2023-01-01T12:00:00Z",
	}
	msgBytes1, _ := json.Marshal(message1)
	tick1, err := normalizer.Normalize(msgBytes1)
	require.NoError(t, err)
	assert.True(t, tick1.Timestamp.UTC().Equal(tick1.Timestamp))

	// Test Unix timestamp (nanoseconds)
	message2 := map[string]interface{}{
		"T": "t",
		"S": "AAPL",
		"p": 150.0,
		"s": 100,
		"t": int64(1672574400000000000),
	}
	msgBytes2, _ := json.Marshal(message2)
	tick2, err := normalizer.Normalize(msgBytes2)
	require.NoError(t, err)
	assert.False(t, tick2.Timestamp.IsZero())
	assert.True(t, tick2.Timestamp.UTC().Equal(tick2.Timestamp))
}

func TestNormalizer_PriceVolumeNormalization(t *testing.T) {
	normalizer := NewNormalizer("alpaca")

	// Test string price
	message1 := map[string]interface{}{
		"T": "t",
		"S": "AAPL",
		"p": "150.50", // String price
		"s": "100",    // String size
		"t": time.Now().UTC().Format(time.RFC3339),
	}
	msgBytes1, _ := json.Marshal(message1)
	tick1, err := normalizer.Normalize(msgBytes1)
	require.NoError(t, err)
	assert.Equal(t, 150.50, tick1.Price)
	assert.Equal(t, int64(100), tick1.Size)

	// Test float64 price and int64 size
	message2 := map[string]interface{}{
		"T": "t",
		"S": "AAPL",
		"p": 150.50,
		"s": int64(100),
		"t": time.Now().UTC().Format(time.RFC3339),
	}
	msgBytes2, _ := json.Marshal(message2)
	tick2, err := normalizer.Normalize(msgBytes2)
	require.NoError(t, err)
	assert.Equal(t, 150.50, tick2.Price)
	assert.Equal(t, int64(100), tick2.Size)
}

func TestNormalizeBatch(t *testing.T) {
	normalizer := NewNormalizer("alpaca")

	messages := [][]byte{
		[]byte(`{"T":"t","S":"AAPL","p":150.0,"s":100,"t":"2023-01-01T12:00:00Z"}`),
		[]byte(`{"T":"t","S":"MSFT","p":250.0,"s":200,"t":"2023-01-01T12:00:00Z"}`),
		[]byte(`invalid json`), // This should fail
		[]byte(`{"T":"t","S":"GOOGL","p":140.0,"s":300,"t":"2023-01-01T12:00:00Z"}`),
	}

	ticks, errors := NormalizeBatch(normalizer, messages)

	// Should have 3 successful normalizations and 1 error
	assert.Len(t, ticks, 3)
	assert.Len(t, errors, 1)

	assert.Equal(t, "AAPL", ticks[0].Symbol)
	assert.Equal(t, "MSFT", ticks[1].Symbol)
	assert.Equal(t, "GOOGL", ticks[2].Symbol)
}

func TestNormalizer_GetProviderName(t *testing.T) {
	normalizer := NewNormalizer("alpaca")
	assert.Equal(t, "alpaca", normalizer.GetProviderName())

	normalizer2 := NewNormalizer("polygon")
	assert.Equal(t, "polygon", normalizer2.GetProviderName())
}

