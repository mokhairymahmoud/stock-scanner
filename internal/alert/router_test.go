package alert

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
)

func TestRouter_RouteAlert(t *testing.T) {
	redis := storage.NewMockRedisClient()
	router := NewRouter(redis, "alerts.filtered", 5*time.Second)

	alert := &models.Alert{
		ID:        "alert-1",
		RuleID:    "rule-1",
		RuleName:  "Test Rule",
		Symbol:    "AAPL",
		Timestamp: time.Now(),
		Price:     150.0,
		Message:   "Test alert",
	}

	ctx := context.Background()

	err := router.RouteAlert(ctx, alert)
	if err != nil {
		t.Fatalf("Failed to route alert: %v", err)
	}

	// Verify alert was published to stream
	// The mock doesn't track stream messages, but we can verify no error occurred
}

func TestRouter_RouteAlerts(t *testing.T) {
	redis := storage.NewMockRedisClient()
	router := NewRouter(redis, "alerts.filtered", 5*time.Second)

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

	ctx := context.Background()

	err := router.RouteAlerts(ctx, alerts)
	if err != nil {
		t.Fatalf("Failed to route alerts: %v", err)
	}
}

func TestRouter_AlertSerialization(t *testing.T) {
	alert := &models.Alert{
		ID:        "alert-1",
		RuleID:    "rule-1",
		RuleName:  "Test Rule",
		Symbol:    "AAPL",
		Timestamp: time.Now(),
		Price:     150.0,
		Message:   "Test alert",
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}

	// Verify alert can be serialized to JSON
	data, err := json.Marshal(alert)
	if err != nil {
		t.Fatalf("Failed to marshal alert: %v", err)
	}

	// Verify it can be unmarshaled
	var unmarshaled models.Alert
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal alert: %v", err)
	}

	if unmarshaled.ID != alert.ID {
		t.Errorf("Expected alert ID %s, got %s", alert.ID, unmarshaled.ID)
	}
	if unmarshaled.Symbol != alert.Symbol {
		t.Errorf("Expected symbol %s, got %s", alert.Symbol, unmarshaled.Symbol)
	}
}

