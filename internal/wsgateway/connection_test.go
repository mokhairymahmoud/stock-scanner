package wsgateway

import (
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

func TestConnection_SubscribeUnsubscribe(t *testing.T) {
	conn := &Connection{
		ID:            "conn-1",
		UserID:        "user-1",
		Subscriptions: make(map[string]bool),
	}

	// Subscribe to symbol
	conn.Subscribe("AAPL")
	if !conn.IsSubscribed("AAPL") {
		t.Error("Expected connection to be subscribed to AAPL")
	}

	// Unsubscribe
	conn.Unsubscribe("AAPL")
	if conn.IsSubscribed("AAPL") {
		t.Error("Expected connection to be unsubscribed from AAPL")
	}
}

func TestConnection_ShouldReceiveAlert(t *testing.T) {
	conn := &Connection{
		ID:            "conn-1",
		UserID:        "user-1",
		Subscriptions: make(map[string]bool),
	}

	alert := &models.Alert{
		ID:        "alert-1",
		RuleID:    "rule-1",
		Symbol:    "AAPL",
		Timestamp: time.Now(),
		Price:     150.0,
	}

	// MVP: No subscriptions means receive all alerts
	shouldReceive := conn.ShouldReceiveAlert(alert)
	if !shouldReceive {
		t.Error("Expected connection to receive alert (no subscriptions = all alerts)")
	}

	// Subscribe to specific symbol
	conn.Subscribe("AAPL")
	shouldReceive = conn.ShouldReceiveAlert(alert)
	if !shouldReceive {
		t.Error("Expected connection to receive alert for subscribed symbol")
	}

	// Different symbol should not be received
	alert2 := &models.Alert{
		ID:        "alert-2",
		RuleID:    "rule-2",
		Symbol:    "MSFT",
		Timestamp: time.Now(),
		Price:     200.0,
	}
	shouldReceive = conn.ShouldReceiveAlert(alert2)
	if shouldReceive {
		t.Error("Expected connection not to receive alert for unsubscribed symbol")
	}
}

func TestConnection_UpdateLastPong(t *testing.T) {
	conn := &Connection{
		ID:            "conn-1",
		UserID:        "user-1",
		Subscriptions: make(map[string]bool),
		lastPong:      time.Now().Add(-1 * time.Hour),
	}

	initialPong := conn.GetLastPong()
	time.Sleep(10 * time.Millisecond)

	conn.UpdateLastPong()
	newPong := conn.GetLastPong()

	if !newPong.After(initialPong) {
		t.Error("Expected last pong time to be updated")
	}
}

