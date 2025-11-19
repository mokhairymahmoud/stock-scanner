package indicator

import (
	"encoding/json"
	"fmt"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

// BarProcessorInterface defines the interface for processing bars
// This allows the stream consumer to process bars instead of ticks
type BarProcessorInterface interface {
	ProcessBar(bar *models.Bar1m) error
}

// BarConsumerAdapter adapts the stream consumer to process bars
type BarConsumerAdapter struct {
	processor BarProcessorInterface
}

// NewBarConsumerAdapter creates a new bar consumer adapter
func NewBarConsumerAdapter(processor BarProcessorInterface) *BarConsumerAdapter {
	return &BarConsumerAdapter{
		processor: processor,
	}
}

// DeserializeBar deserializes a stream message into a Bar1m
func (a *BarConsumerAdapter) DeserializeBar(msg storage.StreamMessage) (*models.Bar1m, error) {
	// The stream publisher stores bars with key "bar"
	barJSON, ok := msg.Values["bar"].(string)
	if !ok {
		// Try to find any string value (fallback)
		for _, v := range msg.Values {
			if str, ok := v.(string); ok {
				barJSON = str
				break
			}
		}
		if barJSON == "" {
			return nil, fmt.Errorf("no bar data found in message")
		}
	}

	var bar models.Bar1m
	err := json.Unmarshal([]byte(barJSON), &bar)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal bar: %w", err)
	}

	return &bar, nil
}

// ProcessMessage processes a stream message containing a bar
func (a *BarConsumerAdapter) ProcessMessage(msg storage.StreamMessage) error {
	bar, err := a.DeserializeBar(msg)
	if err != nil {
		logger.Error("Failed to deserialize bar",
			logger.ErrorField(err),
			logger.String("stream", msg.Stream),
			logger.String("message_id", msg.ID),
		)
		return err
	}

	if a.processor == nil {
		return fmt.Errorf("no processor set")
	}

	err = a.processor.ProcessBar(bar)
	if err != nil {
		logger.Error("Failed to process bar",
			logger.ErrorField(err),
			logger.String("symbol", bar.Symbol),
			logger.String("message_id", msg.ID),
		)
		return err
	}

	return nil
}

