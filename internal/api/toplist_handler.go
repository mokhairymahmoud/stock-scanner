package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/toplist"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

// ToplistHandler handles toplist management endpoints
type ToplistHandler struct {
	toplistService *toplist.ToplistService
	toplistStore   toplist.ToplistStore
}

// NewToplistHandler creates a new toplist handler
func NewToplistHandler(toplistService *toplist.ToplistService, toplistStore toplist.ToplistStore) *ToplistHandler {
	return &ToplistHandler{
		toplistService: toplistService,
		toplistStore:   toplistStore,
	}
}

// ListToplists handles GET /api/v1/toplists
// Returns both system and user-custom toplists
func (h *ToplistHandler) ListToplists(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := getUserID(r) // Get user ID from context (set by auth middleware)

	// Get user's toplists
	userToplists, err := h.toplistStore.GetUserToplists(ctx, userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve user toplists")
		return
	}

	// Get system toplists (enabled ones)
	systemToplists, err := h.toplistStore.GetEnabledToplists(ctx, "")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve system toplists")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"system_toplists": systemToplists,
		"user_toplists":   userToplists,
		"count":           len(systemToplists) + len(userToplists),
	})
}

// GetSystemToplist handles GET /api/v1/toplists/system/:type
func (h *ToplistHandler) GetSystemToplist(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	toplistType := vars["type"]

	// Parse query parameters
	limit := parseIntQuery(r, "limit", 50, 1, 500)
	offset := parseIntQuery(r, "offset", 0, 0, 10000)

	// Map system toplist type to metric and window
	metric, window, err := parseSystemToplistType(toplistType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid toplist type: "+err.Error())
		return
	}

	ctx := r.Context()

	// Create a temporary config for system toplist (not stored in DB)
	tempConfig := &models.ToplistConfig{
		ID:         toplistType,
		UserID:     "", // System toplist
		Metric:     metric,
		TimeWindow: window,
		SortOrder:  models.SortOrderDesc, // System toplists are typically descending
		Enabled:    true,
	}

	// Get rankings using the config
	rankings, err := h.toplistService.GetRankingsByConfig(ctx, tempConfig, limit, offset, nil)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve toplist rankings")
		return
	}

	// Get total count
	total, err := h.toplistService.GetCountByConfig(ctx, tempConfig)
	if err != nil {
		total = int64(len(rankings))
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"toplist_type": toplistType,
		"rankings":      rankings,
		"pagination": map[string]interface{}{
			"limit":  limit,
			"offset": offset,
			"total":  total,
		},
	})
}

// ListUserToplists handles GET /api/v1/toplists/user
func (h *ToplistHandler) ListUserToplists(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := getUserID(r)

	toplists, err := h.toplistStore.GetUserToplists(ctx, userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve user toplists")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"toplists": toplists,
		"count":    len(toplists),
	})
}

// CreateUserToplist handles POST /api/v1/toplists/user
func (h *ToplistHandler) CreateUserToplist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := getUserID(r)

	var config models.ToplistConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Generate ID if not provided
	if config.ID == "" {
		config.ID = uuid.New().String()
	}

	// Set user ID
	config.UserID = userID

	// Set timestamps
	now := time.Now()
	if config.CreatedAt.IsZero() {
		config.CreatedAt = now
	}
	config.UpdatedAt = now

	// Validate config
	if err := config.Validate(); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid toplist configuration: "+err.Error())
		return
	}

	// Create toplist
	if err := h.toplistStore.CreateToplist(ctx, &config); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create toplist")
		return
	}

	logger.Info("Toplist created",
		logger.String("toplist_id", config.ID),
		logger.String("user_id", userID),
		logger.String("name", config.Name),
	)

	respondWithJSON(w, http.StatusCreated, config)
}

// GetUserToplist handles GET /api/v1/toplists/user/:id
func (h *ToplistHandler) GetUserToplist(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	toplistID := vars["id"]
	ctx := r.Context()
	userID := getUserID(r)

	// Get toplist config
	config, err := h.toplistStore.GetToplistConfig(ctx, toplistID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Toplist not found")
		return
	}

	// Verify ownership
	if config.UserID != userID {
		respondWithError(w, http.StatusForbidden, "Access denied")
		return
	}

	respondWithJSON(w, http.StatusOK, config)
}

// UpdateUserToplist handles PUT /api/v1/toplists/user/:id
func (h *ToplistHandler) UpdateUserToplist(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	toplistID := vars["id"]
	ctx := r.Context()
	userID := getUserID(r)

	// Get existing toplist
	existingConfig, err := h.toplistStore.GetToplistConfig(ctx, toplistID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Toplist not found")
		return
	}

	// Verify ownership
	if existingConfig.UserID != userID {
		respondWithError(w, http.StatusForbidden, "Access denied")
		return
	}

	var config models.ToplistConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Ensure ID and user ID match
	config.ID = toplistID
	config.UserID = userID
	config.CreatedAt = existingConfig.CreatedAt
	config.UpdatedAt = time.Now()

	// Validate config
	if err := config.Validate(); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid toplist configuration: "+err.Error())
		return
	}

	// Update toplist
	if err := h.toplistStore.UpdateToplist(ctx, &config); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update toplist")
		return
	}

	logger.Info("Toplist updated",
		logger.String("toplist_id", toplistID),
		logger.String("user_id", userID),
	)

	respondWithJSON(w, http.StatusOK, config)
}

// DeleteUserToplist handles DELETE /api/v1/toplists/user/:id
func (h *ToplistHandler) DeleteUserToplist(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	toplistID := vars["id"]
	ctx := r.Context()
	userID := getUserID(r)

	// Get existing toplist to verify ownership
	config, err := h.toplistStore.GetToplistConfig(ctx, toplistID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Toplist not found")
		return
	}

	// Verify ownership
	if config.UserID != userID {
		respondWithError(w, http.StatusForbidden, "Access denied")
		return
	}

	// Delete toplist
	if err := h.toplistStore.DeleteToplist(ctx, toplistID); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete toplist")
		return
	}

	logger.Info("Toplist deleted",
		logger.String("toplist_id", toplistID),
		logger.String("user_id", userID),
	)

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Toplist deleted"})
}

// GetToplistRankings handles GET /api/v1/toplists/user/:id/rankings
func (h *ToplistHandler) GetToplistRankings(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	toplistID := vars["id"]
	ctx := r.Context()
	userID := getUserID(r)

	// Get toplist config to verify ownership
	config, err := h.toplistStore.GetToplistConfig(ctx, toplistID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Toplist not found")
		return
	}

	// Verify ownership
	if config.UserID != userID {
		respondWithError(w, http.StatusForbidden, "Access denied")
		return
	}

	// Parse query parameters
	limit := parseIntQuery(r, "limit", 50, 1, 500)
	offset := parseIntQuery(r, "offset", 0, 0, 10000)

	// Parse filters from query
	var filters *models.ToplistFilter
	if minVolStr := r.URL.Query().Get("min_volume"); minVolStr != "" {
		if minVol, err := strconv.ParseInt(minVolStr, 10, 64); err == nil {
			if filters == nil {
				filters = &models.ToplistFilter{}
			}
			filters.MinVolume = &minVol
		}
	}
	if priceMinStr := r.URL.Query().Get("price_min"); priceMinStr != "" {
		if priceMin, err := strconv.ParseFloat(priceMinStr, 64); err == nil {
			if filters == nil {
				filters = &models.ToplistFilter{}
			}
			filters.PriceMin = &priceMin
		}
	}
	if priceMaxStr := r.URL.Query().Get("price_max"); priceMaxStr != "" {
		if priceMax, err := strconv.ParseFloat(priceMaxStr, 64); err == nil {
			if filters == nil {
				filters = &models.ToplistFilter{}
			}
			filters.PriceMax = &priceMax
		}
	}

	// Get rankings
	rankings, err := h.toplistService.GetToplistRankings(ctx, toplistID, limit, offset, filters)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve rankings")
		return
	}

	// Get total count
	total, err := h.toplistService.GetToplistCount(ctx, toplistID)
	if err != nil {
		total = int64(len(rankings))
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"toplist_id": toplistID,
		"name":       config.Name,
		"rankings":   rankings,
		"pagination": map[string]interface{}{
			"limit":  limit,
			"offset": offset,
			"total":  total,
		},
	})
}

// Helper functions

func getUserID(r *http.Request) string {
	// Get user ID from context (set by auth middleware)
	userID := r.Context().Value("user_id")
	if userID == nil {
		return "default" // MVP: default user
	}
	return userID.(string)
}

func parseIntQuery(r *http.Request, key string, defaultValue, min, max int) int {
	valueStr := r.URL.Query().Get(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil || value < min || value > max {
		return defaultValue
	}
	return value
}

func parseSystemToplistType(toplistType string) (models.ToplistMetric, models.ToplistTimeWindow, error) {
	// Map system toplist types to metrics and windows
	switch models.SystemToplistType(toplistType) {
	case models.SystemToplistGainers1m, models.SystemToplistLosers1m:
		return models.MetricChangePct, models.Window1m, nil
	case models.SystemToplistGainers5m, models.SystemToplistLosers5m:
		return models.MetricChangePct, models.Window5m, nil
	case models.SystemToplistGainers15m, models.SystemToplistLosers15m:
		return models.MetricChangePct, models.Window15m, nil
	case models.SystemToplistGainers1h, models.SystemToplistLosers1h:
		return models.MetricChangePct, models.Window1h, nil
	case models.SystemToplistGainers1d, models.SystemToplistLosers1d:
		return models.MetricChangePct, models.Window1d, nil
	case models.SystemToplistVolume1m:
		return models.MetricVolume, models.Window1m, nil
	case models.SystemToplistVolume5m:
		return models.MetricVolume, models.Window5m, nil
	case models.SystemToplistVolume15m:
		return models.MetricVolume, models.Window15m, nil
	case models.SystemToplistVolume1h:
		return models.MetricVolume, models.Window1h, nil
	case models.SystemToplistVolume1d:
		return models.MetricVolume, models.Window1d, nil
	case models.SystemToplistRSIHigh, models.SystemToplistRSILow:
		return models.MetricRSI, models.Window1m, nil
	case models.SystemToplistRelVolume:
		return models.MetricRelativeVolume, models.Window15m, nil
	case models.SystemToplistVWAPDistHigh, models.SystemToplistVWAPDistLow:
		return models.MetricVWAPDist, models.Window5m, nil
	default:
		return "", "", fmt.Errorf("unknown system toplist type: %s", toplistType)
	}
}

