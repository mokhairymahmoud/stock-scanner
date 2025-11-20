package toplist

import (
	"context"
	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

// ToplistStore defines the interface for toplist configuration storage
type ToplistStore interface {
	// GetToplistConfig retrieves a toplist configuration by ID
	GetToplistConfig(ctx context.Context, toplistID string) (*models.ToplistConfig, error)

	// GetUserToplists retrieves all toplists for a user
	GetUserToplists(ctx context.Context, userID string) ([]*models.ToplistConfig, error)

	// GetEnabledToplists retrieves all enabled toplists (for a user or all system toplists)
	GetEnabledToplists(ctx context.Context, userID string) ([]*models.ToplistConfig, error)

	// CreateToplist creates a new toplist configuration
	CreateToplist(ctx context.Context, config *models.ToplistConfig) error

	// UpdateToplist updates an existing toplist configuration
	UpdateToplist(ctx context.Context, config *models.ToplistConfig) error

	// DeleteToplist deletes a toplist configuration
	DeleteToplist(ctx context.Context, toplistID string) error

	// Close closes the store connection
	Close() error
}
