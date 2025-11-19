package alert

import (
	"context"
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

func TestUserFilter_FilterAlert(t *testing.T) {
	filter := NewUserFilter()

	alert := &models.Alert{
		ID:        "alert-1",
		RuleID:    "rule-1",
		Symbol:    "AAPL",
		Timestamp: time.Now(),
		Price:     150.0,
		Message:   "Test alert",
	}

	ctx := context.Background()

	// MVP: All alerts should pass through
	pass, err := filter.FilterAlert(ctx, alert)
	if err != nil {
		t.Fatalf("Failed to filter alert: %v", err)
	}
	if !pass {
		t.Error("Expected alert to pass filter (MVP: all pass through)")
	}
}

func TestUserFilter_FilterAlerts(t *testing.T) {
	filter := NewUserFilter()

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

	filtered, err := filter.FilterAlerts(ctx, alerts)
	if err != nil {
		t.Fatalf("Failed to filter alerts: %v", err)
	}

	// MVP: All alerts should pass through
	if len(filtered) != len(alerts) {
		t.Errorf("Expected %d filtered alerts, got %d", len(alerts), len(filtered))
	}
}

