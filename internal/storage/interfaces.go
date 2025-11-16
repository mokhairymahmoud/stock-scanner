package storage

import (
	"context"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

// BarStorage defines the interface for bar storage operations
type BarStorage interface {
	// WriteBars writes finalized bars to storage
	WriteBars(ctx context.Context, bars []*models.Bar1m) error

	// GetBars retrieves bars for a symbol within a time range
	GetBars(ctx context.Context, symbol string, start, end time.Time) ([]*models.Bar1m, error)

	// GetLatestBars retrieves the latest N bars for a symbol
	GetLatestBars(ctx context.Context, symbol string, limit int) ([]*models.Bar1m, error)

	// Close closes the storage connection
	Close() error
}

// AlertStorage defines the interface for alert storage operations
type AlertStorage interface {
	// WriteAlert writes an alert to storage
	WriteAlert(ctx context.Context, alert *models.Alert) error

	// WriteAlerts writes multiple alerts to storage (batch operation)
	WriteAlerts(ctx context.Context, alerts []*models.Alert) error

	// GetAlerts retrieves alerts with filtering options
	GetAlerts(ctx context.Context, filter AlertFilter) ([]*models.Alert, error)

	// GetAlert retrieves a single alert by ID
	GetAlert(ctx context.Context, alertID string) (*models.Alert, error)

	// Close closes the storage connection
	Close() error
}

// AlertFilter defines filtering options for alert queries
type AlertFilter struct {
	Symbol    string
	RuleID    string
	StartTime time.Time
	EndTime   time.Time
	Limit     int
	Offset    int
}

// RedisClient defines the interface for Redis operations
type RedisClient interface {
	// Stream operations
	PublishToStream(ctx context.Context, stream string, key string, value interface{}) error
	PublishBatchToStream(ctx context.Context, stream string, messages []map[string]interface{}) error
	ConsumeFromStream(ctx context.Context, stream string, group string, consumer string) (<-chan StreamMessage, error)
	AcknowledgeMessage(ctx context.Context, stream string, group string, id string) error

	// Key-value operations
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	GetJSON(ctx context.Context, key string, dest interface{}) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)

	// Set operations
	SetAdd(ctx context.Context, key string, members ...string) error
	SetMembers(ctx context.Context, key string) ([]string, error)
	SetRemove(ctx context.Context, key string, members ...string) error

	// Pub/Sub operations
	Publish(ctx context.Context, channel string, message interface{}) error
	Subscribe(ctx context.Context, channels ...string) (<-chan PubSubMessage, error)

	// Close closes the Redis connection
	Close() error
}

// StreamMessage represents a message from a Redis stream
type StreamMessage struct {
	ID     string
	Stream string
	Values map[string]interface{}
}

// PubSubMessage represents a message from Redis pub/sub
type PubSubMessage struct {
	Channel string
	Message string
}

