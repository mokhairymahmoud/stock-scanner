package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

var (
	// ErrUnsupportedFormat is returned when the message format is not supported
	ErrUnsupportedFormat = errors.New("unsupported message format")
	// ErrInvalidMessage is returned when the message cannot be parsed
	ErrInvalidMessage = errors.New("invalid message")
)

// Normalizer normalizes market data from different providers to a common format
type Normalizer interface {
	// Normalize converts a raw message to a Tick
	Normalize(rawMessage []byte) (*models.Tick, error)

	// GetProviderName returns the name of the provider this normalizer handles
	GetProviderName() string
}

// DefaultNormalizer is a flexible normalizer that can handle multiple formats
type DefaultNormalizer struct {
	providerName string
}

// NewNormalizer creates a new normalizer for the given provider
func NewNormalizer(providerName string) Normalizer {
	return &DefaultNormalizer{
		providerName: providerName,
	}
}

// GetProviderName returns the provider name
func (n *DefaultNormalizer) GetProviderName() string {
	return n.providerName
}

// Normalize converts a raw message to a Tick
func (n *DefaultNormalizer) Normalize(rawMessage []byte) (*models.Tick, error) {
	if len(rawMessage) == 0 {
		return nil, ErrInvalidMessage
	}

	// Try to parse as JSON first (most providers use JSON)
	var jsonData map[string]interface{}
	if err := json.Unmarshal(rawMessage, &jsonData); err == nil {
		return n.normalizeJSON(jsonData)
	}

	// If not JSON, try other formats
	return nil, fmt.Errorf("%w: message is not in a supported format", ErrUnsupportedFormat)
}

// normalizeJSON normalizes a JSON message to a Tick
func (n *DefaultNormalizer) normalizeJSON(data map[string]interface{}) (*models.Tick, error) {
	// Provider-specific normalization
	switch n.providerName {
	case "alpaca":
		return n.normalizeAlpaca(data)
	case "polygon":
		return n.normalizePolygon(data)
	case "mock":
		return n.normalizeMock(data)
	default:
		// Generic normalization - try common field names
		return n.normalizeGeneric(data)
	}
}

// normalizeAlpaca normalizes Alpaca WebSocket messages
func (n *DefaultNormalizer) normalizeAlpaca(data map[string]interface{}) (*models.Tick, error) {
	tick := &models.Tick{Type: "trade"}

	// Alpaca trade message format
	// Example: {"T":"t","S":"AAPL","p":150.5,"s":100,"t":"2023-01-01T12:00:00Z"}
	if msgType, ok := data["T"].(string); ok {
		if msgType == "t" { // Trade
			tick.Type = "trade"
		} else if msgType == "q" { // Quote
			tick.Type = "quote"
		}
	}

	// Symbol
	if symbol, ok := data["S"].(string); ok {
		tick.Symbol = strings.ToUpper(symbol)
	} else {
		return nil, fmt.Errorf("%w: missing symbol", ErrInvalidMessage)
	}

	// Price (for trades: "p", for quotes: "ap" or "bp")
	if tick.Type == "trade" {
		if price, ok := data["p"].(float64); ok {
			tick.Price = price
		} else if priceStr, ok := data["p"].(string); ok {
			price, err := strconv.ParseFloat(priceStr, 64)
			if err != nil {
				return nil, fmt.Errorf("%w: invalid price: %v", ErrInvalidMessage, err)
			}
			tick.Price = price
		} else {
			return nil, fmt.Errorf("%w: missing price", ErrInvalidMessage)
		}
	} else {
		// Quote: use ask price or bid price
		if ask, ok := data["ap"].(float64); ok {
			tick.Price = ask
			tick.Ask = ask
		} else if askStr, ok := data["ap"].(string); ok {
			ask, err := strconv.ParseFloat(askStr, 64)
			if err == nil {
				tick.Price = ask
				tick.Ask = ask
			}
		}
		if bid, ok := data["bp"].(float64); ok {
			tick.Bid = bid
			if tick.Price == 0 {
				tick.Price = bid
			}
		} else if bidStr, ok := data["bp"].(string); ok {
			bid, err := strconv.ParseFloat(bidStr, 64)
			if err == nil {
				tick.Bid = bid
				if tick.Price == 0 {
					tick.Price = bid
				}
			}
		}
	}

	// Size (for trades: "s", for quotes: "as" or "bs")
	if tick.Type == "trade" {
		if size, ok := data["s"].(float64); ok {
			tick.Size = int64(size)
		} else if size, ok := data["s"].(int64); ok {
			tick.Size = size
		} else if sizeStr, ok := data["s"].(string); ok {
			size, err := strconv.ParseInt(sizeStr, 10, 64)
			if err == nil {
				tick.Size = size
			}
		}
	}

	// Timestamp
	if ts, ok := data["t"].(string); ok {
		parsedTime, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			// Try other formats
			parsedTime, err = time.Parse(time.RFC3339Nano, ts)
			if err != nil {
				parsedTime, err = time.Parse("2006-01-02T15:04:05.999999999Z07:00", ts)
			}
		}
		if err == nil {
			tick.Timestamp = parsedTime.UTC()
		} else {
			// Fallback to current time
			tick.Timestamp = time.Now().UTC()
		}
	} else if ts, ok := data["t"].(float64); ok {
		// Unix timestamp in nanoseconds
		tick.Timestamp = time.Unix(0, int64(ts)).UTC()
	} else if ts, ok := data["t"].(int64); ok {
		// Unix timestamp in nanoseconds
		tick.Timestamp = time.Unix(0, ts).UTC()
	} else {
		// Fallback to current time
		tick.Timestamp = time.Now().UTC()
	}

	// Validate tick
	if err := tick.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidMessage, err)
	}

	return tick, nil
}

// normalizePolygon normalizes Polygon.io WebSocket messages
func (n *DefaultNormalizer) normalizePolygon(data map[string]interface{}) (*models.Tick, error) {
	tick := &models.Tick{Type: "trade"}

	// Polygon trade message format
	// Example: {"ev":"T","sym":"AAPL","p":150.5,"s":100,"t":1672574400000000000}
	if ev, ok := data["ev"].(string); ok {
		if ev == "T" { // Trade
			tick.Type = "trade"
		} else if ev == "Q" { // Quote
			tick.Type = "quote"
		}
	}

	// Symbol
	if symbol, ok := data["sym"].(string); ok {
		tick.Symbol = strings.ToUpper(symbol)
	} else {
		return nil, fmt.Errorf("%w: missing symbol", ErrInvalidMessage)
	}

	// Price
	if price, ok := data["p"].(float64); ok {
		tick.Price = price
	} else if priceStr, ok := data["p"].(string); ok {
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid price: %v", ErrInvalidMessage, err)
		}
		tick.Price = price
	} else {
		return nil, fmt.Errorf("%w: missing price", ErrInvalidMessage)
	}

	// Size
	if size, ok := data["s"].(float64); ok {
		tick.Size = int64(size)
	} else if size, ok := data["s"].(int64); ok {
		tick.Size = size
	}

	// Timestamp (nanoseconds)
	if ts, ok := data["t"].(float64); ok {
		tick.Timestamp = time.Unix(0, int64(ts)).UTC()
	} else if ts, ok := data["t"].(int64); ok {
		tick.Timestamp = time.Unix(0, ts).UTC()
	} else {
		tick.Timestamp = time.Now().UTC()
	}

	// Validate tick
	if err := tick.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidMessage, err)
	}

	return tick, nil
}

// normalizeMock normalizes mock provider messages
func (n *DefaultNormalizer) normalizeMock(data map[string]interface{}) (*models.Tick, error) {
	// Mock provider already returns Tick struct, but handle JSON if needed
	tick := &models.Tick{Type: "trade"}

	if symbol, ok := data["symbol"].(string); ok {
		tick.Symbol = strings.ToUpper(symbol)
	} else {
		return nil, fmt.Errorf("%w: missing symbol", ErrInvalidMessage)
	}

	if price, ok := data["price"].(float64); ok {
		tick.Price = price
	} else {
		return nil, fmt.Errorf("%w: missing price", ErrInvalidMessage)
	}

	if size, ok := data["size"].(float64); ok {
		tick.Size = int64(size)
	} else if size, ok := data["size"].(int64); ok {
		tick.Size = size
	}

	if ts, ok := data["timestamp"].(string); ok {
		parsedTime, err := time.Parse(time.RFC3339, ts)
		if err == nil {
			tick.Timestamp = parsedTime.UTC()
		} else {
			tick.Timestamp = time.Now().UTC()
		}
	} else {
		tick.Timestamp = time.Now().UTC()
	}

	if tickType, ok := data["type"].(string); ok {
		tick.Type = tickType
	}

	// Validate tick
	if err := tick.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidMessage, err)
	}

	return tick, nil
}

// normalizeGeneric normalizes messages using common field names
func (n *DefaultNormalizer) normalizeGeneric(data map[string]interface{}) (*models.Tick, error) {
	tick := &models.Tick{Type: "trade"}

	// Try common field names
	symbolFields := []string{"symbol", "sym", "S", "ticker", "instrument"}
	for _, field := range symbolFields {
		if val, ok := data[field].(string); ok && val != "" {
			tick.Symbol = strings.ToUpper(val)
			break
		}
	}

	if tick.Symbol == "" {
		return nil, fmt.Errorf("%w: missing symbol", ErrInvalidMessage)
	}

	// Price fields
	priceFields := []string{"price", "p", "last", "close", "ap", "bp"}
	for _, field := range priceFields {
		if val, ok := data[field].(float64); ok && val > 0 {
			tick.Price = val
			break
		} else if valStr, ok := data[field].(string); ok {
			if val, err := strconv.ParseFloat(valStr, 64); err == nil && val > 0 {
				tick.Price = val
				break
			}
		}
	}

	if tick.Price <= 0 {
		return nil, fmt.Errorf("%w: missing or invalid price", ErrInvalidMessage)
	}

	// Size fields
	sizeFields := []string{"size", "s", "volume", "qty", "quantity"}
	for _, field := range sizeFields {
		if val, ok := data[field].(float64); ok {
			tick.Size = int64(val)
			break
		} else if val, ok := data[field].(int64); ok {
			tick.Size = val
			break
		}
	}

	// Timestamp fields
	timestampFields := []string{"timestamp", "t", "time", "ts", "datetime"}
	for _, field := range timestampFields {
		if ts, ok := data[field].(string); ok {
			parsedTime, err := time.Parse(time.RFC3339, ts)
			if err == nil {
				tick.Timestamp = parsedTime.UTC()
				break
			}
		} else if ts, ok := data[field].(float64); ok {
			tick.Timestamp = time.Unix(0, int64(ts)).UTC()
			break
		} else if ts, ok := data[field].(int64); ok {
			tick.Timestamp = time.Unix(0, ts).UTC()
			break
		}
	}

	if tick.Timestamp.IsZero() {
		tick.Timestamp = time.Now().UTC()
	}

	// Type
	if tickType, ok := data["type"].(string); ok {
		tick.Type = tickType
	}

	// Validate tick
	if err := tick.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidMessage, err)
	}

	return tick, nil
}

// NormalizeBatch normalizes multiple messages in batch
func NormalizeBatch(normalizer Normalizer, messages [][]byte) ([]*models.Tick, []error) {
	ticks := make([]*models.Tick, 0, len(messages))
	errors := make([]error, 0)

	for _, msg := range messages {
		tick, err := normalizer.Normalize(msg)
		if err != nil {
			logger.Warn("Failed to normalize message",
				logger.ErrorField(err),
				logger.String("provider", normalizer.GetProviderName()),
			)
			errors = append(errors, err)
			continue
		}
		ticks = append(ticks, tick)
	}

	return ticks, errors
}
