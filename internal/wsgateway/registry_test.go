package wsgateway

import (
	"testing"
)

func TestConnectionRegistry_AddRemove(t *testing.T) {
	registry := NewConnectionRegistry()

	// Create mock connection
	conn := &Connection{
		ID:     "conn-1",
		UserID: "user-1",
	}

	// Add connection
	registry.Add(conn)

	// Verify connection exists
	retrieved, exists := registry.Get("conn-1")
	if !exists {
		t.Error("Expected connection to exist")
	}
	if retrieved.ID != "conn-1" {
		t.Errorf("Expected connection ID %s, got %s", "conn-1", retrieved.ID)
	}

	// Verify count
	if registry.Count() != 1 {
		t.Errorf("Expected 1 connection, got %d", registry.Count())
	}

	// Remove connection
	registry.Remove("conn-1")

	// Verify connection removed
	_, exists = registry.Get("conn-1")
	if exists {
		t.Error("Expected connection to be removed")
	}

	if registry.Count() != 0 {
		t.Errorf("Expected 0 connections, got %d", registry.Count())
	}
}

func TestConnectionRegistry_GetByUser(t *testing.T) {
	registry := NewConnectionRegistry()

	conn1 := &Connection{ID: "conn-1", UserID: "user-1"}
	conn2 := &Connection{ID: "conn-2", UserID: "user-1"}
	conn3 := &Connection{ID: "conn-3", UserID: "user-2"}

	registry.Add(conn1)
	registry.Add(conn2)
	registry.Add(conn3)

	// Get connections for user-1
	user1Conns := registry.GetByUser("user-1")
	if len(user1Conns) != 2 {
		t.Errorf("Expected 2 connections for user-1, got %d", len(user1Conns))
	}

	// Get connections for user-2
	user2Conns := registry.GetByUser("user-2")
	if len(user2Conns) != 1 {
		t.Errorf("Expected 1 connection for user-2, got %d", len(user2Conns))
	}

	// Get connections for non-existent user
	user3Conns := registry.GetByUser("user-3")
	if user3Conns != nil && len(user3Conns) != 0 {
		t.Errorf("Expected 0 connections for user-3, got %d", len(user3Conns))
	}
}

func TestConnectionRegistry_CountByUser(t *testing.T) {
	registry := NewConnectionRegistry()

	conn1 := &Connection{ID: "conn-1", UserID: "user-1"}
	conn2 := &Connection{ID: "conn-2", UserID: "user-1"}

	registry.Add(conn1)
	registry.Add(conn2)

	count := registry.CountByUser("user-1")
	if count != 2 {
		t.Errorf("Expected 2 connections for user-1, got %d", count)
	}

	count = registry.CountByUser("user-2")
	if count != 0 {
		t.Errorf("Expected 0 connections for user-2, got %d", count)
	}
}

func TestConnectionRegistry_GetAll(t *testing.T) {
	registry := NewConnectionRegistry()

	conn1 := &Connection{ID: "conn-1", UserID: "user-1"}
	conn2 := &Connection{ID: "conn-2", UserID: "user-2"}

	registry.Add(conn1)
	registry.Add(conn2)

	all := registry.GetAll()
	if len(all) != 2 {
		t.Errorf("Expected 2 connections, got %d", len(all))
	}
}

// Helper to create a mock WebSocket connection for testing
func createMockConnection(id, userID string) *Connection {
	// Create a dummy websocket.Conn - in real tests, you'd use a proper mock
	// For now, we'll just create the Connection struct without the actual websocket
	return &Connection{
		ID:            id,
		UserID:        userID,
		Conn:          nil, // In real tests, use a mock websocket.Conn
		Send:          make(chan []byte, 256),
		Subscriptions: make(map[string]bool),
	}
}

