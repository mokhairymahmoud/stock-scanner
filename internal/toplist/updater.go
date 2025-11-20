package toplist

import (
	"context"
	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

// ToplistUpdate represents a single toplist update operation
type ToplistUpdate struct {
	Key    string  // Redis key for the toplist
	Symbol string  // Symbol to update
	Value  float64 // Metric value for ranking
}

// ToplistUpdater defines the interface for updating toplists
type ToplistUpdater interface {
	// UpdateSystemToplist updates a system toplist (e.g., gainers_1m, volume_day)
	UpdateSystemToplist(ctx context.Context, metric models.ToplistMetric, window models.ToplistTimeWindow, symbol string, value float64) error

	// UpdateUserToplist updates a user-custom toplist
	UpdateUserToplist(ctx context.Context, userID string, toplistID string, symbol string, value float64) error

	// BatchUpdate performs batch updates using Redis pipeline for efficiency
	BatchUpdate(ctx context.Context, updates []ToplistUpdate) error

	// PublishUpdate publishes a toplist update notification to Redis pub/sub
	PublishUpdate(ctx context.Context, toplistID string, toplistType string) error

	// Close closes the updater
	Close() error
}

