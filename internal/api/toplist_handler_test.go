package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
	"github.com/mohamedkhairy/stock-scanner/internal/toplist"
)

func TestToplistHandler_CreateUserToplist(t *testing.T) {
	mockStore := toplist.NewMockToplistStore()
	mockRedis := storage.NewMockRedisClient()
	mockUpdater := toplist.NewRedisToplistUpdater(mockRedis)
	service := toplist.NewToplistService(mockStore, mockRedis, mockUpdater)
	handler := NewToplistHandler(service, mockStore)

	config := models.ToplistConfig{
		Name:       "Test Toplist",
		Metric:     models.MetricChangePct,
		TimeWindow: models.Window5m,
		SortOrder:  models.SortOrderDesc,
		Enabled:    true,
	}

	body, _ := json.Marshal(config)
	req := httptest.NewRequest("POST", "/api/v1/toplists/user", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), "user_id", "user-123"))
	w := httptest.NewRecorder()

	handler.CreateUserToplist(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("CreateUserToplist() status = %d, want %d", w.Code, http.StatusCreated)
	}

	var response models.ToplistConfig
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.ID == "" {
		t.Error("CreateUserToplist() did not generate ID")
	}
	if response.UserID != "user-123" {
		t.Errorf("CreateUserToplist() UserID = %s, want user-123", response.UserID)
	}
}

func TestToplistHandler_GetUserToplist(t *testing.T) {
	mockStore := toplist.NewMockToplistStore()
	mockRedis := storage.NewMockRedisClient()
	mockUpdater := toplist.NewRedisToplistUpdater(mockRedis)
	service := toplist.NewToplistService(mockStore, mockRedis, mockUpdater)
	handler := NewToplistHandler(service, mockStore)

	// Create a test toplist
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
	mockStore.CreateToplist(context.Background(), config)

	req := httptest.NewRequest("GET", "/api/v1/toplists/user/test-1", nil)
	req = req.WithContext(context.WithValue(req.Context(), "user_id", "user-123"))
	vars := map[string]string{"id": "test-1"}
	req = mux.SetURLVars(req, vars)
	w := httptest.NewRecorder()

	handler.GetUserToplist(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetUserToplist() status = %d, want %d", w.Code, http.StatusOK)
	}

	var response models.ToplistConfig
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.ID != "test-1" {
		t.Errorf("GetUserToplist() ID = %s, want test-1", response.ID)
	}
}

func TestToplistHandler_GetUserToplist_Forbidden(t *testing.T) {
	mockStore := toplist.NewMockToplistStore()
	mockRedis := storage.NewMockRedisClient()
	mockUpdater := toplist.NewRedisToplistUpdater(mockRedis)
	service := toplist.NewToplistService(mockStore, mockRedis, mockUpdater)
	handler := NewToplistHandler(service, mockStore)

	// Create a test toplist owned by different user
	config := &models.ToplistConfig{
		ID:         "test-1",
		UserID:     "user-456",
		Name:       "Test Toplist",
		Metric:     models.MetricChangePct,
		TimeWindow: models.Window5m,
		SortOrder:  models.SortOrderDesc,
		Enabled:    true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	mockStore.CreateToplist(context.Background(), config)

	req := httptest.NewRequest("GET", "/api/v1/toplists/user/test-1", nil)
	req = req.WithContext(context.WithValue(req.Context(), "user_id", "user-123"))
	vars := map[string]string{"id": "test-1"}
	req = mux.SetURLVars(req, vars)
	w := httptest.NewRecorder()

	handler.GetUserToplist(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("GetUserToplist() status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestToplistHandler_ListUserToplists(t *testing.T) {
	mockStore := toplist.NewMockToplistStore()
	mockRedis := storage.NewMockRedisClient()
	mockUpdater := toplist.NewRedisToplistUpdater(mockRedis)
	service := toplist.NewToplistService(mockStore, mockRedis, mockUpdater)
	handler := NewToplistHandler(service, mockStore)

	// Create test toplists
	config1 := &models.ToplistConfig{
		ID:         uuid.New().String(),
		UserID:     "user-123",
		Name:       "Toplist 1",
		Metric:     models.MetricChangePct,
		TimeWindow: models.Window5m,
		SortOrder:  models.SortOrderDesc,
		Enabled:    true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	config2 := &models.ToplistConfig{
		ID:         uuid.New().String(),
		UserID:     "user-123",
		Name:       "Toplist 2",
		Metric:     models.MetricVolume,
		TimeWindow: models.Window1m,
		SortOrder:  models.SortOrderDesc,
		Enabled:    true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	mockStore.CreateToplist(context.Background(), config1)
	mockStore.CreateToplist(context.Background(), config2)

	req := httptest.NewRequest("GET", "/api/v1/toplists/user", nil)
	req = req.WithContext(context.WithValue(req.Context(), "user_id", "user-123"))
	w := httptest.NewRecorder()

	handler.ListUserToplists(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListUserToplists() status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	toplists, ok := response["toplists"].([]interface{})
	if !ok {
		t.Fatal("ListUserToplists() response missing toplists field")
	}

	if len(toplists) != 2 {
		t.Errorf("ListUserToplists() returned %d toplists, want 2", len(toplists))
	}
}

func TestToplistHandler_GetSystemToplist(t *testing.T) {
	mockStore := toplist.NewMockToplistStore()
	mockRedis := storage.NewMockRedisClient()
	mockUpdater := toplist.NewRedisToplistUpdater(mockRedis)
	service := toplist.NewToplistService(mockStore, mockRedis, mockUpdater)
	handler := NewToplistHandler(service, mockStore)

	// Add test data to Redis
	key := models.GetSystemToplistRedisKey(models.MetricChangePct, models.Window1m)
	mockRedis.ZAdd(context.Background(), key, 2.5, "AAPL")
	mockRedis.ZAdd(context.Background(), key, 1.8, "MSFT")
	mockRedis.ZAdd(context.Background(), key, 3.2, "GOOGL")

	req := httptest.NewRequest("GET", "/api/v1/toplists/system/gainers_1m?limit=10", nil)
	vars := map[string]string{"type": "gainers_1m"}
	req = mux.SetURLVars(req, vars)
	w := httptest.NewRecorder()

	handler.GetSystemToplist(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetSystemToplist() status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	rankings, ok := response["rankings"].([]interface{})
	if !ok {
		t.Fatal("GetSystemToplist() response missing rankings field")
	}

	if len(rankings) != 3 {
		t.Errorf("GetSystemToplist() returned %d rankings, want 3", len(rankings))
	}
}

