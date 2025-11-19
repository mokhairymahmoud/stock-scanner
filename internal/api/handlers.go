package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/rules"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

// RuleHandler handles rule management endpoints
type RuleHandler struct {
	ruleStore   rules.RuleStore
	compiler    *rules.Compiler
	syncService *rules.RuleSyncService
}

// NewRuleHandler creates a new rule handler
func NewRuleHandler(ruleStore rules.RuleStore, compiler *rules.Compiler, syncService *rules.RuleSyncService) *RuleHandler {
	return &RuleHandler{
		ruleStore:   ruleStore,
		compiler:    compiler,
		syncService: syncService,
	}
}

// ListRules handles GET /api/v1/rules
func (h *RuleHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	allRules, err := h.ruleStore.GetAllRules()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve rules")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"rules": allRules,
		"count": len(allRules),
	})
}

// GetRule handles GET /api/v1/rules/:id
func (h *RuleHandler) GetRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["id"]

	rule, err := h.ruleStore.GetRule(ruleID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Rule not found")
		return
	}

	respondWithJSON(w, http.StatusOK, rule)
}

// CreateRule handles POST /api/v1/rules
func (h *RuleHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	var rule models.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Generate ID if not provided
	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = now
	}
	rule.UpdatedAt = now

	// Validate rule
	if err := rules.ValidateRule(&rule); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Try to compile rule to ensure it's valid
	_, err := h.compiler.CompileRule(&rule)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to compile rule: "+err.Error())
		return
	}

	// Add rule to store
	if err := h.ruleStore.AddRule(&rule); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create rule")
		return
	}

	// Sync to Redis if sync service is available
	if h.syncService != nil {
		if err := h.syncService.SyncRule(rule.ID); err != nil {
			logger.Warn("Failed to sync rule to Redis",
				logger.ErrorField(err),
				logger.String("rule_id", rule.ID),
			)
			// Don't fail the request if sync fails
		}
	}

	logger.Info("Rule created",
		logger.String("rule_id", rule.ID),
		logger.String("rule_name", rule.Name),
	)

	respondWithJSON(w, http.StatusCreated, rule)
}

// UpdateRule handles PUT /api/v1/rules/:id
func (h *RuleHandler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["id"]

	// Check if rule exists
	existingRule, err := h.ruleStore.GetRule(ruleID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Rule not found")
		return
	}

	var rule models.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Ensure ID matches
	rule.ID = ruleID
	rule.CreatedAt = existingRule.CreatedAt
	rule.UpdatedAt = time.Now()

	// Validate rule
	if err := rules.ValidateRule(&rule); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Try to compile rule to ensure it's valid
	_, err = h.compiler.CompileRule(&rule)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to compile rule: "+err.Error())
		return
	}

	// Update rule in store
	if err := h.ruleStore.UpdateRule(&rule); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update rule")
		return
	}

	// Sync to Redis if sync service is available
	if h.syncService != nil {
		if err := h.syncService.SyncRule(rule.ID); err != nil {
			logger.Warn("Failed to sync rule to Redis",
				logger.ErrorField(err),
				logger.String("rule_id", rule.ID),
			)
			// Don't fail the request if sync fails
		}
	}

	logger.Info("Rule updated",
		logger.String("rule_id", rule.ID),
		logger.String("rule_name", rule.Name),
	)

	respondWithJSON(w, http.StatusOK, rule)
}

// DeleteRule handles DELETE /api/v1/rules/:id
func (h *RuleHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["id"]

	if err := h.ruleStore.DeleteRule(ruleID); err != nil {
		respondWithError(w, http.StatusNotFound, "Rule not found")
		return
	}

	// Remove from Redis if sync service is available
	if h.syncService != nil {
		if err := h.syncService.DeleteRuleFromRedis(ruleID); err != nil {
			logger.Warn("Failed to delete rule from Redis",
				logger.ErrorField(err),
				logger.String("rule_id", ruleID),
			)
			// Don't fail the request if sync fails
		}
	}

	logger.Info("Rule deleted",
		logger.String("rule_id", ruleID),
	)

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Rule deleted"})
}

// ValidateRule handles POST /api/v1/rules/:id/validate
func (h *RuleHandler) ValidateRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["id"]

	rule, err := h.ruleStore.GetRule(ruleID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Rule not found")
		return
	}

	// Validate rule syntax
	if err := rules.ValidateRule(rule); err != nil {
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"valid": false,
			"error": err.Error(),
		})
		return
	}

	// Try to compile rule
	_, err = h.compiler.CompileRule(rule)
	if err != nil {
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"valid": false,
			"error": "Failed to compile rule: " + err.Error(),
		})
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"valid": true,
	})
}

// AlertHandler handles alert history endpoints
type AlertHandler struct {
	alertStorage storage.AlertStorage
}

// NewAlertHandler creates a new alert handler
func NewAlertHandler(alertStorage storage.AlertStorage) *AlertHandler {
	return &AlertHandler{
		alertStorage: alertStorage,
	}
}

// ListAlerts handles GET /api/v1/alerts
func (h *AlertHandler) ListAlerts(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	filter := storage.AlertFilter{
		Symbol:    r.URL.Query().Get("symbol"),
		RuleID:    r.URL.Query().Get("rule_id"),
		Limit:     100, // Default limit
		Offset:    0,
	}

	// Parse limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := parseInt(limitStr); err == nil && limit > 0 && limit <= 1000 {
			filter.Limit = limit
		}
	}

	// Parse offset
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := parseInt(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	// Parse date range
	if startStr := r.URL.Query().Get("start_time"); startStr != "" {
		if start, err := time.Parse(time.RFC3339, startStr); err == nil {
			filter.StartTime = start
		}
	}

	if endStr := r.URL.Query().Get("end_time"); endStr != "" {
		if end, err := time.Parse(time.RFC3339, endStr); err == nil {
			filter.EndTime = end
		}
	}

	alerts, err := h.alertStorage.GetAlerts(r.Context(), filter)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve alerts")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"alerts": alerts,
		"count":  len(alerts),
		"limit":   filter.Limit,
		"offset":  filter.Offset,
	})
}

// GetAlert handles GET /api/v1/alerts/:id
func (h *AlertHandler) GetAlert(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	alertID := vars["id"]

	alert, err := h.alertStorage.GetAlert(r.Context(), alertID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve alert")
		return
	}

	if alert == nil {
		respondWithError(w, http.StatusNotFound, "Alert not found")
		return
	}

	respondWithJSON(w, http.StatusOK, alert)
}

// SymbolHandler handles symbol management endpoints
type SymbolHandler struct {
	symbols []string // MVP: hardcoded list, in production this would come from database
}

// NewSymbolHandler creates a new symbol handler
func NewSymbolHandler(symbols []string) *SymbolHandler {
	return &SymbolHandler{
		symbols: symbols,
	}
}

// ListSymbols handles GET /api/v1/symbols
func (h *SymbolHandler) ListSymbols(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	
	var filtered []string
	if search == "" {
		filtered = h.symbols
	} else {
		// Simple case-insensitive search
		searchLower := strings.ToLower(search)
		for _, symbol := range h.symbols {
			if strings.Contains(strings.ToLower(symbol), searchLower) {
				filtered = append(filtered, symbol)
			}
		}
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"symbols": filtered,
		"count":   len(filtered),
	})
}

// GetSymbol handles GET /api/v1/symbols/:symbol
func (h *SymbolHandler) GetSymbol(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	symbol := vars["symbol"]

	// Check if symbol exists
	for _, s := range h.symbols {
		if s == symbol {
			respondWithJSON(w, http.StatusOK, map[string]interface{}{
				"symbol": symbol,
			})
			return
		}
	}

	respondWithError(w, http.StatusNotFound, "Symbol not found")
}

// UserHandler handles user management endpoints
type UserHandler struct {
	// MVP: No user storage, just return default user info
}

// NewUserHandler creates a new user handler
func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

// GetProfile handles GET /api/v1/user/profile
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID := r.Context().Value("user_id")
	if userID == nil {
		userID = "default"
	}

	// MVP: Return basic user profile
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"user_id": userID,
		"email":   "", // MVP: No email storage
		"name":    "", // MVP: No name storage
	})
}

// UpdateProfile handles PUT /api/v1/user/profile
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID := r.Context().Value("user_id")
	if userID == nil {
		userID = "default"
	}

	// MVP: Accept profile update but don't persist
	var profile map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// MVP: Return success but don't actually save
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"user_id": userID,
		"message": "Profile update accepted (MVP: not persisted)",
	})
}

// Helper functions

func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

