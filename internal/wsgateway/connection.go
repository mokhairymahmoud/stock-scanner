package wsgateway

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

// Connection represents a WebSocket connection with a client
type Connection struct {
	ID                string
	UserID            string
	Conn              *websocket.Conn
	Send              chan []byte
	Subscriptions     map[string]bool // symbol -> subscribed
	ToplistSubscriptions map[string]bool // toplist_id -> subscribed
	mu                sync.RWMutex
	ctx               context.Context
	cancel            context.CancelFunc
	lastPong          time.Time
	createdAt         time.Time
}

// NewConnection creates a new WebSocket connection
func NewConnection(id string, userID string, conn *websocket.Conn) *Connection {
	ctx, cancel := context.WithCancel(context.Background())
	return &Connection{
		ID:                  id,
		UserID:              userID,
		Conn:                conn,
		Send:                make(chan []byte, 256), // Buffered channel
		Subscriptions:       make(map[string]bool),
		ToplistSubscriptions: make(map[string]bool),
		ctx:                 ctx,
		cancel:              cancel,
		createdAt:           time.Now(),
		lastPong:            time.Now(),
	}
}

// Subscribe subscribes to alerts for a symbol
func (c *Connection) Subscribe(symbol string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Subscriptions[symbol] = true
}

// Unsubscribe unsubscribes from alerts for a symbol
func (c *Connection) Unsubscribe(symbol string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.Subscriptions, symbol)
}

// IsSubscribed checks if the connection is subscribed to a symbol
func (c *Connection) IsSubscribed(symbol string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Subscriptions[symbol]
}

// ShouldReceiveAlert checks if the connection should receive an alert
func (c *Connection) ShouldReceiveAlert(alert *models.Alert) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// If no subscriptions, receive all alerts (MVP behavior)
	if len(c.Subscriptions) == 0 {
		return true
	}
	
	// Check if subscribed to this symbol
	return c.Subscriptions[alert.Symbol]
}

// SubscribeToplist subscribes to toplist updates
func (c *Connection) SubscribeToplist(toplistID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ToplistSubscriptions[toplistID] = true
}

// UnsubscribeToplist unsubscribes from toplist updates
func (c *Connection) UnsubscribeToplist(toplistID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.ToplistSubscriptions, toplistID)
}

// IsSubscribedToToplist checks if the connection is subscribed to a toplist
func (c *Connection) IsSubscribedToToplist(toplistID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ToplistSubscriptions[toplistID]
}

// UpdateLastPong updates the last pong time
func (c *Connection) UpdateLastPong() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastPong = time.Now()
}

// GetLastPong returns the last pong time
func (c *Connection) GetLastPong() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastPong
}

// Close closes the connection
func (c *Connection) Close() {
	c.cancel()
	close(c.Send)
	c.Conn.Close()
}

// WriteMessage writes a message to the connection
func (c *Connection) WriteMessage(messageType int, data []byte) error {
	c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return c.Conn.WriteMessage(messageType, data)
}

// ReadMessage reads a message from the connection
func (c *Connection) ReadMessage() (messageType int, p []byte, err error) {
	return c.Conn.ReadMessage()
}

// WriteJSON writes a JSON message to the connection
func (c *Connection) WriteJSON(v interface{}) error {
	c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return c.Conn.WriteJSON(v)
}

// ReadJSON reads a JSON message from the connection
func (c *Connection) ReadJSON(v interface{}) error {
	return c.Conn.ReadJSON(v)
}

// SendAlert sends an alert to the connection
func (c *Connection) SendAlert(alert *models.Alert) error {
	message := map[string]interface{}{
		"type": "alert",
		"data": alert,
	}
	
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	
	select {
	case c.Send <- data:
		return nil
	case <-c.ctx.Done():
		return c.ctx.Err()
	case <-time.After(1 * time.Second):
		logger.Warn("Failed to send alert, channel full",
			logger.String("connection_id", c.ID),
			logger.String("user_id", c.UserID),
		)
		return nil // Drop message if channel is full
	}
}

// SendError sends an error message to the connection
func (c *Connection) SendError(code string, message string) error {
	errorMsg := map[string]interface{}{
		"type":    "error",
		"code":    code,
		"message": message,
	}
	
	data, err := json.Marshal(errorMsg)
	if err != nil {
		return err
	}
	
	select {
	case c.Send <- data:
		return nil
	case <-c.ctx.Done():
		return c.ctx.Err()
	default:
		// Drop error message if channel is full
		return nil
	}
}

