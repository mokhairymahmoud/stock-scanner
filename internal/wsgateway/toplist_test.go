package wsgateway

import (
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/config"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
)

func TestConnection_SubscribeToplist(t *testing.T) {
	conn := NewConnection("test-conn", "user-123", nil)

	conn.SubscribeToplist("gainers_1m")
	if !conn.IsSubscribedToToplist("gainers_1m") {
		t.Error("SubscribeToplist() failed - toplist not subscribed")
	}
}

func TestConnection_UnsubscribeToplist(t *testing.T) {
	conn := NewConnection("test-conn", "user-123", nil)

	conn.SubscribeToplist("gainers_1m")
	conn.UnsubscribeToplist("gainers_1m")
	if conn.IsSubscribedToToplist("gainers_1m") {
		t.Error("UnsubscribeToplist() failed - toplist still subscribed")
	}
}

func TestConnection_ToplistSubscriptions(t *testing.T) {
	conn := NewConnection("test-conn", "user-123", nil)

	// Subscribe to multiple toplists
	conn.SubscribeToplist("gainers_1m")
	conn.SubscribeToplist("volume_day")
	conn.SubscribeToplist("user-123-custom-1")

	// Verify all subscriptions
	if !conn.IsSubscribedToToplist("gainers_1m") {
		t.Error("Toplist subscription missing: gainers_1m")
	}
	if !conn.IsSubscribedToToplist("volume_day") {
		t.Error("Toplist subscription missing: volume_day")
	}
	if !conn.IsSubscribedToToplist("user-123-custom-1") {
		t.Error("Toplist subscription missing: user-123-custom-1")
	}

	// Unsubscribe from one
	conn.UnsubscribeToplist("volume_day")
	if conn.IsSubscribedToToplist("volume_day") {
		t.Error("Toplist still subscribed after unsubscribe: volume_day")
	}

	// Others should still be subscribed
	if !conn.IsSubscribedToToplist("gainers_1m") {
		t.Error("Other toplist subscription lost: gainers_1m")
	}
	if !conn.IsSubscribedToToplist("user-123-custom-1") {
		t.Error("Other toplist subscription lost: user-123-custom-1")
	}
}

func TestHub_BroadcastToplistUpdate(t *testing.T) {
	mockRedis := storage.NewMockRedisClient()
	cfg := config.WSGatewayConfig{
		Port:            8080,
		HealthCheckPort: 8081,
		ReadTimeout:     60 * time.Second,
		WriteTimeout:    60 * time.Second,
		PingInterval:    30 * time.Second,
		MaxConnections:  1000,
		AlertStream:     "alerts.filtered",
		ConsumerGroup:   "ws-gateway",
	}
	hub := NewHub(cfg, mockRedis, "alerts.filtered", "ws-gateway")

	// Create test connections (without actual websocket connections for unit test)
	conn1 := NewConnection("conn-1", "user-123", nil)
	conn2 := NewConnection("conn-2", "user-456", nil)
	conn3 := NewConnection("conn-3", "user-123", nil)

	// Subscribe to toplists
	conn1.SubscribeToplist("gainers_1m")
	conn2.SubscribeToplist("losers_1m")
	conn3.SubscribeToplist("gainers_1m")

	// Just verify subscription state (don't actually register since we don't have real websockets)
	// In integration tests, we'd test the full flow with real websocket connections
	if !conn1.IsSubscribedToToplist("gainers_1m") {
		t.Error("conn1 should be subscribed to gainers_1m")
	}
	if !conn3.IsSubscribedToToplist("gainers_1m") {
		t.Error("conn3 should be subscribed to gainers_1m")
	}
	if conn2.IsSubscribedToToplist("gainers_1m") {
		t.Error("conn2 should NOT be subscribed to gainers_1m")
	}
}

func TestProtocol_SubscribeToplist(t *testing.T) {
	conn := NewConnection("test-conn", "user-123", nil)

	msg := &ClientMessage{
		Type:   "subscribe_toplist",
		Symbol: "gainers_1m", // Reuse Symbol field for toplist ID
	}

	err := conn.HandleClientMessage(msg)
	if err != nil {
		t.Fatalf("HandleClientMessage() error = %v", err)
	}

	if !conn.IsSubscribedToToplist("gainers_1m") {
		t.Error("HandleClientMessage() failed to subscribe to toplist")
	}
}

func TestProtocol_UnsubscribeToplist(t *testing.T) {
	conn := NewConnection("test-conn", "user-123", nil)
	conn.SubscribeToplist("gainers_1m")

	msg := &ClientMessage{
		Type:   "unsubscribe_toplist",
		Symbol: "gainers_1m",
	}

	err := conn.HandleClientMessage(msg)
	if err != nil {
		t.Fatalf("HandleClientMessage() error = %v", err)
	}

	if conn.IsSubscribedToToplist("gainers_1m") {
		t.Error("HandleClientMessage() failed to unsubscribe from toplist")
	}
}

