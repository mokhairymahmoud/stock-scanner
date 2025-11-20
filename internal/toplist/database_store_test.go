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
	configs map[string]*models.ToplistConfig
}

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
		TimeWindow: models.Window1m,
		SortOrder:  models.SortOrderDesc,
		Enabled:    true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	mockStore.CreateToplist(ctx, config)

	// Add some test data to Redis
	key := models.GetUserToplistRedisKey("user-123", "test-1")
	mockRedis.ZAdd(ctx, key, 2.5, "AAPL")
	mockRedis.ZAdd(ctx, key, 1.8, "MSFT")
	mockRedis.ZAdd(ctx, key, 3.2, "GOOGL")

	// Get rankings
	rankings, err := service.GetToplistRankings(ctx, "test-1", 10, 0, nil)
	if err != nil {
		t.Fatalf("GetToplistRankings() error = %v", err)
	}

	if len(rankings) != 3 {
		t.Errorf("GetToplistRankings() returned %d rankings, want 3", len(rankings))
	}

	// Verify rankings are in descending order (highest first)
	if rankings[0].Symbol != "GOOGL" || rankings[0].Value != 3.2 {
		t.Errorf("First ranking should be GOOGL with value 3.2, got %s with value %v", rankings[0].Symbol, rankings[0].Value)
	}
}

func TestToplistService_GetToplistCount(t *testing.T) {
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
		TimeWindow: models.Window1m,
		SortOrder:  models.SortOrderDesc,
		Enabled:    true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	mockStore.CreateToplist(ctx, config)

	// Add some test data to Redis
	key := models.GetUserToplistRedisKey("user-123", "test-1")
	mockRedis.ZAdd(ctx, key, 2.5, "AAPL")
	mockRedis.ZAdd(ctx, key, 1.8, "MSFT")
	mockRedis.ZAdd(ctx, key, 3.2, "GOOGL")

	// Get count
	count, err := service.GetToplistCount(ctx, "test-1")
	if err != nil {
		t.Fatalf("GetToplistCount() error = %v", err)
	}

	if count != 3 {
		t.Errorf("GetToplistCount() = %d, want 3", count)
	}
}

func TestToplistService_CacheToplistConfig(t *testing.T) {
	mockStore := NewMockToplistStore()
	mockRedis := storage.NewMockRedisClient()
	mockUpdater := NewRedisToplistUpdater(mockRedis)
	service := NewToplistService(mockStore, mockRedis, mockUpdater)
	ctx := context.Background()

	config := &models.ToplistConfig{
		ID:         "test-1",
		UserID:     "user-123",
		Name:       "Test Toplist",
		Metric:     models.MetricChangePct,
		TimeWindow: models.Window1m,
		SortOrder:  models.SortOrderDesc,
		Enabled:    true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err := service.CacheToplistConfig(ctx, config)
	if err != nil {
		t.Fatalf("CacheToplistConfig() error = %v", err)
	}

	// Verify cache
	cached, err := service.GetCachedToplistConfig(ctx, "test-1")
	if err != nil {
		t.Fatalf("GetCachedToplistConfig() error = %v", err)
	}

	if cached == nil {
		t.Error("GetCachedToplistConfig() returned nil, expected config")
		return
	}

	if cached.ID != config.ID {
		t.Errorf("GetCachedToplistConfig() ID = %s, want %s", cached.ID, config.ID)
	}
}
