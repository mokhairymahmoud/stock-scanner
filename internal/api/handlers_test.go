package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/rules"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
)

func TestRuleHandler_ListRules(t *testing.T) {
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	handler := NewRuleHandler(ruleStore, compiler, nil)

	// Add a test rule
	rule := &models.Rule{
		ID:          "rule-1",
		Name:        "Test Rule",
		Conditions:  []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Cooldown:    300,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	ruleStore.AddRule(rule)

	req := httptest.NewRequest("GET", "/api/v1/rules", nil)
	w := httptest.NewRecorder()

	handler.ListRules(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	rulesList, ok := response["rules"].([]interface{})
	if !ok {
		t.Fatal("Expected 'rules' array in response")
	}

	if len(rulesList) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(rulesList))
	}
}

func TestRuleHandler_CreateRule(t *testing.T) {
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	handler := NewRuleHandler(ruleStore, compiler, nil)

	ruleData := map[string]interface{}{
		"name":       "New Rule",
		"conditions": []map[string]interface{}{
			{"metric": "rsi_14", "operator": "<", "value": 30.0},
		},
		"cooldown": 300,
		"enabled":  true,
	}

	body, _ := json.Marshal(ruleData)
	req := httptest.NewRequest("POST", "/api/v1/rules", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateRule(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var rule models.Rule
	if err := json.Unmarshal(w.Body.Bytes(), &rule); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if rule.ID == "" {
		t.Error("Expected rule ID to be generated")
	}

	if rule.Name != "New Rule" {
		t.Errorf("Expected rule name 'New Rule', got %s", rule.Name)
	}

	// Verify rule was stored
	_, err := ruleStore.GetRule(rule.ID)
	if err != nil {
		t.Errorf("Rule was not stored: %v", err)
	}
}

func TestRuleHandler_GetRule(t *testing.T) {
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	handler := NewRuleHandler(ruleStore, compiler, nil)

	rule := &models.Rule{
		ID:          "rule-1",
		Name:        "Test Rule",
		Conditions:  []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Cooldown:    300,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	ruleStore.AddRule(rule)

	req := httptest.NewRequest("GET", "/api/v1/rules/rule-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "rule-1"})
	w := httptest.NewRecorder()

	handler.GetRule(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var retrievedRule models.Rule
	if err := json.Unmarshal(w.Body.Bytes(), &retrievedRule); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if retrievedRule.ID != "rule-1" {
		t.Errorf("Expected rule ID 'rule-1', got %s", retrievedRule.ID)
	}
}

func TestRuleHandler_GetRule_NotFound(t *testing.T) {
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	handler := NewRuleHandler(ruleStore, compiler, nil)

	req := httptest.NewRequest("GET", "/api/v1/rules/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	w := httptest.NewRecorder()

	handler.GetRule(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestRuleHandler_UpdateRule(t *testing.T) {
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	handler := NewRuleHandler(ruleStore, compiler, nil)

	// Create initial rule
	rule := &models.Rule{
		ID:          "rule-1",
		Name:        "Original Name",
		Conditions:  []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Cooldown:    300,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	ruleStore.AddRule(rule)

	// Update rule
	updateData := map[string]interface{}{
		"name":       "Updated Name",
		"conditions": []map[string]interface{}{
			{"metric": "rsi_14", "operator": "<", "value": 30.0},
		},
		"cooldown": 600,
		"enabled":  true,
	}

	body, _ := json.Marshal(updateData)
	req := httptest.NewRequest("PUT", "/api/v1/rules/rule-1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "rule-1"})
	w := httptest.NewRecorder()

	handler.UpdateRule(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var updatedRule models.Rule
	if err := json.Unmarshal(w.Body.Bytes(), &updatedRule); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if updatedRule.Name != "Updated Name" {
		t.Errorf("Expected rule name 'Updated Name', got %s", updatedRule.Name)
	}

	if updatedRule.Cooldown != 600 {
		t.Errorf("Expected cooldown 600, got %d", updatedRule.Cooldown)
	}
}

func TestRuleHandler_DeleteRule(t *testing.T) {
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	handler := NewRuleHandler(ruleStore, compiler, nil)

	rule := &models.Rule{
		ID:          "rule-1",
		Name:        "Test Rule",
		Conditions:  []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Cooldown:    300,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	ruleStore.AddRule(rule)

	req := httptest.NewRequest("DELETE", "/api/v1/rules/rule-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "rule-1"})
	w := httptest.NewRecorder()

	handler.DeleteRule(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify rule was deleted
	_, err := ruleStore.GetRule("rule-1")
	if err == nil {
		t.Error("Expected rule to be deleted")
	}
}

func TestRuleHandler_ValidateRule(t *testing.T) {
	ruleStore := rules.NewInMemoryRuleStore()
	compiler := rules.NewCompiler(nil)
	handler := NewRuleHandler(ruleStore, compiler, nil)

	rule := &models.Rule{
		ID:          "rule-1",
		Name:        "Test Rule",
		Conditions:  []models.Condition{{Metric: "rsi_14", Operator: "<", Value: 30.0}},
		Cooldown:    300,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	ruleStore.AddRule(rule)

	req := httptest.NewRequest("POST", "/api/v1/rules/rule-1/validate", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "rule-1"})
	w := httptest.NewRecorder()

	handler.ValidateRule(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	valid, ok := response["valid"].(bool)
	if !ok {
		t.Fatal("Expected 'valid' boolean in response")
	}

	if !valid {
		t.Error("Expected rule to be valid")
	}
}

func TestAlertHandler_ListAlerts(t *testing.T) {
	alertStorage := &storage.MockAlertStorage{}
	handler := NewAlertHandler(alertStorage)

	// Add test alerts
	alerts := []*models.Alert{
		{
			ID:        "alert-1",
			RuleID:    "rule-1",
			Symbol:    "AAPL",
			Timestamp: time.Now(),
			Price:     150.0,
		},
		{
			ID:        "alert-2",
			RuleID:    "rule-2",
			Symbol:    "MSFT",
			Timestamp: time.Now(),
			Price:     200.0,
		},
	}
	alertStorage.WriteAlerts(nil, alerts)

	req := httptest.NewRequest("GET", "/api/v1/alerts", nil)
	w := httptest.NewRecorder()

	handler.ListAlerts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	alertsList, ok := response["alerts"].([]interface{})
	if !ok {
		t.Fatal("Expected 'alerts' array in response")
	}

	if len(alertsList) != 2 {
		t.Errorf("Expected 2 alerts, got %d", len(alertsList))
	}
}

func TestAlertHandler_ListAlerts_WithFilter(t *testing.T) {
	alertStorage := &storage.MockAlertStorage{}
	handler := NewAlertHandler(alertStorage)

	// Add test alerts
	alerts := []*models.Alert{
		{
			ID:        "alert-1",
			RuleID:    "rule-1",
			Symbol:    "AAPL",
			Timestamp: time.Now(),
			Price:     150.0,
		},
		{
			ID:        "alert-2",
			RuleID:    "rule-1",
			Symbol:    "MSFT",
			Timestamp: time.Now(),
			Price:     200.0,
		},
	}
	alertStorage.WriteAlerts(nil, alerts)

	req := httptest.NewRequest("GET", "/api/v1/alerts?symbol=AAPL", nil)
	w := httptest.NewRecorder()

	handler.ListAlerts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	alertsList, ok := response["alerts"].([]interface{})
	if !ok {
		t.Fatal("Expected 'alerts' array in response")
	}

	if len(alertsList) != 1 {
		t.Errorf("Expected 1 alert after filtering, got %d", len(alertsList))
	}
}

func TestSymbolHandler_ListSymbols(t *testing.T) {
	symbols := []string{"AAPL", "MSFT", "GOOGL"}
	handler := NewSymbolHandler(symbols)

	req := httptest.NewRequest("GET", "/api/v1/symbols", nil)
	w := httptest.NewRecorder()

	handler.ListSymbols(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	symbolsList, ok := response["symbols"].([]interface{})
	if !ok {
		t.Fatal("Expected 'symbols' array in response")
	}

	if len(symbolsList) != 3 {
		t.Errorf("Expected 3 symbols, got %d", len(symbolsList))
	}
}

func TestSymbolHandler_ListSymbols_WithSearch(t *testing.T) {
	symbols := []string{"AAPL", "MSFT", "GOOGL"}
	handler := NewSymbolHandler(symbols)

	req := httptest.NewRequest("GET", "/api/v1/symbols?search=AA", nil)
	w := httptest.NewRecorder()

	handler.ListSymbols(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	symbolsList, ok := response["symbols"].([]interface{})
	if !ok {
		t.Fatal("Expected 'symbols' array in response")
	}

	if len(symbolsList) != 1 {
		t.Errorf("Expected 1 symbol after search, got %d", len(symbolsList))
	}
}

func TestUserHandler_GetProfile(t *testing.T) {
	handler := NewUserHandler()

	req := httptest.NewRequest("GET", "/api/v1/user/profile", nil)
	req = req.WithContext(req.Context())
	w := httptest.NewRecorder()

	handler.GetProfile(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["user_id"] == nil {
		t.Error("Expected 'user_id' in response")
	}
}

