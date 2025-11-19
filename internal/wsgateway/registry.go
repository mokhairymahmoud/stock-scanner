package wsgateway

import (
	"sync"
)

// ConnectionRegistry manages all active WebSocket connections
type ConnectionRegistry struct {
	connections map[string]*Connection // connection_id -> connection
	byUser      map[string]map[string]*Connection // user_id -> connection_id -> connection
	mu          sync.RWMutex
}

// NewConnectionRegistry creates a new connection registry
func NewConnectionRegistry() *ConnectionRegistry {
	return &ConnectionRegistry{
		connections: make(map[string]*Connection),
		byUser:      make(map[string]map[string]*Connection),
	}
}

// Add adds a connection to the registry
func (r *ConnectionRegistry) Add(conn *Connection) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.connections[conn.ID] = conn
	
	if r.byUser[conn.UserID] == nil {
		r.byUser[conn.UserID] = make(map[string]*Connection)
	}
	r.byUser[conn.UserID][conn.ID] = conn
}

// Remove removes a connection from the registry
func (r *ConnectionRegistry) Remove(connectionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	conn, exists := r.connections[connectionID]
	if !exists {
		return
	}
	
	delete(r.connections, connectionID)
	
	if userConns, exists := r.byUser[conn.UserID]; exists {
		delete(userConns, connectionID)
		if len(userConns) == 0 {
			delete(r.byUser, conn.UserID)
		}
	}
}

// Get retrieves a connection by ID
func (r *ConnectionRegistry) Get(connectionID string) (*Connection, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	conn, exists := r.connections[connectionID]
	return conn, exists
}

// GetByUser retrieves all connections for a user
func (r *ConnectionRegistry) GetByUser(userID string) []*Connection {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	userConns, exists := r.byUser[userID]
	if !exists {
		return nil
	}
	
	connections := make([]*Connection, 0, len(userConns))
	for _, conn := range userConns {
		connections = append(connections, conn)
	}
	return connections
}

// GetAll retrieves all connections
func (r *ConnectionRegistry) GetAll() []*Connection {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	connections := make([]*Connection, 0, len(r.connections))
	for _, conn := range r.connections {
		connections = append(connections, conn)
	}
	return connections
}

// Count returns the total number of connections
func (r *ConnectionRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.connections)
}

// CountByUser returns the number of connections for a user
func (r *ConnectionRegistry) CountByUser(userID string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	userConns, exists := r.byUser[userID]
	if !exists {
		return 0
	}
	return len(userConns)
}

