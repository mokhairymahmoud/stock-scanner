package toplist

import (
	"context"
	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

// MockToplistStore is a mock implementation of ToplistStore for testing
// Exported for use in other packages
type MockToplistStore struct {
	configs map[string]*models.ToplistConfig
}

// NewMockToplistStore creates a new mock toplist store
func NewMockToplistStore() *MockToplistStore {
	return &MockToplistStore{
		configs: make(map[string]*models.ToplistConfig),
	}
}

func (m *MockToplistStore) GetToplistConfig(ctx context.Context, toplistID string) (*models.ToplistConfig, error) {
	config, exists := m.configs[toplistID]
	if !exists {
		return nil, &NotFoundError{ToplistID: toplistID}
	}
	return config, nil
}

func (m *MockToplistStore) GetUserToplists(ctx context.Context, userID string) ([]*models.ToplistConfig, error) {
	var result []*models.ToplistConfig
	for _, config := range m.configs {
		if config.UserID == userID {
			result = append(result, config)
		}
	}
	return result, nil
}

func (m *MockToplistStore) GetEnabledToplists(ctx context.Context, userID string) ([]*models.ToplistConfig, error) {
	var result []*models.ToplistConfig
	for _, config := range m.configs {
		if !config.Enabled {
			continue
		}
		if userID == "" {
			if config.UserID == "" {
				result = append(result, config)
			}
		} else {
			if config.UserID == userID {
				result = append(result, config)
			}
		}
	}
	return result, nil
}

func (m *MockToplistStore) CreateToplist(ctx context.Context, config *models.ToplistConfig) error {
	m.configs[config.ID] = config
	return nil
}

func (m *MockToplistStore) UpdateToplist(ctx context.Context, config *models.ToplistConfig) error {
	if _, exists := m.configs[config.ID]; !exists {
		return &NotFoundError{ToplistID: config.ID}
	}
	m.configs[config.ID] = config
	return nil
}

func (m *MockToplistStore) DeleteToplist(ctx context.Context, toplistID string) error {
	if _, exists := m.configs[toplistID]; !exists {
		return &NotFoundError{ToplistID: toplistID}
	}
	delete(m.configs, toplistID)
	return nil
}

func (m *MockToplistStore) Close() error {
	return nil
}

// NotFoundError represents a toplist not found error
type NotFoundError struct {
	ToplistID string
}

func (e *NotFoundError) Error() string {
	return "toplist not found: " + e.ToplistID
}

