package toplist

import (
	"context"
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
)

// MockToplistStore is a mock implementation of ToplistStore for testing
type MockToplistStore struct {
	Configs   map[string]*models.ToplistConfig
	UserLists map[string][]*models.ToplistConfig
	GetErr    error
	CreateErr error
	UpdateErr error
	DeleteErr error
}

func NewMockToplistStore() *MockToplistStore {
	return &MockToplistStore{
		Configs:   make(map[string]*models.ToplistConfig),
		UserLists: make(map[string][]*models.ToplistConfig),
	}
}

func (m *MockToplistStore) GetToplistConfig(ctx context.Context, toplistID string) (*models.ToplistConfig, error) {
	if m.GetErr != nil {
		return nil, m.GetErr
	}
	config, exists := m.Configs[toplistID]
	if !exists {
		return nil, &NotFoundError{ToplistID: toplistID}
	}
	return config, nil
}

func (m *MockToplistStore) GetUserToplists(ctx context.Context, userID string) ([]*models.ToplistConfig, error) {
	if m.GetErr != nil {
		return nil, m.GetErr
	}
	return m.UserLists[userID], nil
}

func (m *MockToplistStore) GetEnabledToplists(ctx context.Context) ([]*models.ToplistConfig, error) {
	if m.GetErr != nil {
		return nil, m.GetErr
	}
	var enabled []*models.ToplistConfig
	for _, config := range m.Configs {
		if config.Enabled {
			enabled = append(enabled, config)
		}
	}
	return enabled, nil
}

func (m *MockToplistStore) CreateToplist(ctx context.Context, config *models.ToplistConfig) error {
	if m.CreateErr != nil {
		return m.CreateErr
	}
	m.Configs[config.ID] = config
	if config.UserID != "" {
		m.UserLists[config.UserID] = append(m.UserLists[config.UserID], config)
	}
	return nil
}

func (m *MockToplistStore) UpdateToplist(ctx context.Context, config *models.ToplistConfig) error {
	if m.UpdateErr != nil {
		return m.UpdateErr
	}
	if _, exists := m.Configs[config.ID]; !exists {
		return &NotFoundError{ToplistID: config.ID}
	}
	m.Configs[config.ID] = config
	return nil
}

func (m *MockToplistStore) DeleteToplist(ctx context.Context, toplistID string) error {
	if m.DeleteErr != nil {
		return m.DeleteErr
	}
	if _, exists := m.Configs[toplistID]; !exists {
		return &NotFoundError{ToplistID: toplistID}
	}
	delete(m.Configs, toplistID)
	return nil
}

func (m *MockToplistStore) Close() error {
	return nil
}

type NotFoundError struct {
	ToplistID string
}

func (e *NotFoundError) Error() string {
	return "toplist not found: " + e.ToplistID
}

func TestToplistService_GetToplistRankings(t *testing.T) {
	mockStore := NewMockToplistStore()
	mockRedis := storage.NewMockRedisClient()
	mockUpdater := NewRedisToplistUpdater(mockRedis)
	service := NewToplistService(mockStore, mockRedis, mockUpdater)
	ctx := context.Background()

	// Create a test toplist config
	config := &models.ToplistConfig{
		ID:         "test-1",
		UserID:     "user-123",
		Name:       "Test Toplist",
		Metric:     models.MetricChangePct,
		TimeWindow: models.Window5m,
		SortOrder:  models.SortOrderDesc,
		Enabled:    true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	mockStore.Configs["test-1"] = config

	// Add some test data to Redis
	key := models.GetUserToplistRedisKey("user-123", "test-1")
	mockRedis.ZAdd(ctx, key, 2.5, "AAPL")
	mockRedis.ZAdd(ctx, key, 1.8, "MSFT")
	mockRedis.ZAdd(ctx, key, 3.2, "GOOGL")

	// Get rankings
	rankings, err := service.GetToplistRankings(ctx, "test-1", 10, 0)
	if err != nil {
		t.Fatalf("GetToplistRankings() error = %v", err)
	}

	if len(rankings) != 3 {
		t.Errorf("GetToplistRankings() returned %d rankings, want 3", len(rankings))
	}

	// Verify order (descending)
	if rankings[0].Symbol != "GOOGL" || rankings[0].Value != 3.2 {
		t.Errorf("First ranking = %v, want GOOGL with 3.2", rankings[0])
	}
}

func TestToplistService_GetEnabledToplists(t *testing.T) {
	mockStore := NewMockToplistStore()
	mockRedis := storage.NewMockRedisClient()
	mockUpdater := NewRedisToplistUpdater(mockRedis)
	service := NewToplistService(mockStore, mockRedis, mockUpdater)
	ctx := context.Background()

	// Create enabled and disabled toplists
	enabled1 := &models.ToplistConfig{
		ID:        "enabled-1",
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	enabled2 := &models.ToplistConfig{
		ID:        "enabled-2",
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	disabled := &models.ToplistConfig{
		ID:        "disabled-1",
		Enabled:   false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockStore.Configs["enabled-1"] = enabled1
	mockStore.Configs["enabled-2"] = enabled2
	mockStore.Configs["disabled-1"] = disabled

	// Get enabled toplists
	toplists, err := service.GetEnabledToplists(ctx)
	if err != nil {
		t.Fatalf("GetEnabledToplists() error = %v", err)
	}

	if len(toplists) != 2 {
		t.Errorf("GetEnabledToplists() returned %d toplists, want 2", len(toplists))
	}
}

