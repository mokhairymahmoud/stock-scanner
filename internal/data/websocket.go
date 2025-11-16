package data

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

var (
	// ErrWebSocketNotConnected is returned when operations are attempted on a disconnected WebSocket
	ErrWebSocketNotConnected = errors.New("websocket is not connected")
	// ErrWebSocketAlreadyConnected is returned when attempting to connect an already connected WebSocket
	ErrWebSocketAlreadyConnected = errors.New("websocket is already connected")
)

// WebSocketState represents the connection state
type WebSocketState int

const (
	StateDisconnected WebSocketState = iota
	StateConnecting
	StateConnected
	StateReconnecting
)

func (s WebSocketState) String() string {
	switch s {
	case StateDisconnected:
		return "disconnected"
	case StateConnecting:
		return "connecting"
	case StateConnected:
		return "connected"
	case StateReconnecting:
		return "reconnecting"
	default:
		return "unknown"
	}
}

// WebSocketConfig holds configuration for WebSocket connections
type WebSocketConfig struct {
	URL                string
	ReconnectDelay     time.Duration
	MaxReconnectDelay  time.Duration
	HeartbeatInterval  time.Duration
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	PingPeriod         time.Duration
	PongWait           time.Duration
	MaxReconnectAttempts int // 0 means unlimited
}

// DefaultWebSocketConfig returns a default WebSocket configuration
func DefaultWebSocketConfig(url string) WebSocketConfig {
	return WebSocketConfig{
		URL:                url,
		ReconnectDelay:     1 * time.Second,
		MaxReconnectDelay:  30 * time.Second,
		HeartbeatInterval:  30 * time.Second,
		ReadTimeout:        60 * time.Second,
		WriteTimeout:       10 * time.Second,
		PingPeriod:         54 * time.Second, // Should be less than PongWait
		PongWait:           60 * time.Second,
		MaxReconnectAttempts: 0, // Unlimited
	}
}

// WebSocketClient is a robust WebSocket client with automatic reconnection
type WebSocketClient struct {
	config      WebSocketConfig
	conn        *websocket.Conn
	state       WebSocketState
	mu          sync.RWMutex
	reconnectAttempts int
	lastError   error

	// Channels
	messageChan chan []byte
	errorChan   chan error
	closeChan   chan struct{}

	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Callbacks
	onConnect    func()
	onDisconnect func(error)
	onMessage    func([]byte)
	onError      func(error)
}

// NewWebSocketClient creates a new WebSocket client
func NewWebSocketClient(config WebSocketConfig) *WebSocketClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &WebSocketClient{
		config:      config,
		state:       StateDisconnected,
		messageChan: make(chan []byte, 100),
		errorChan:   make(chan error, 10),
		closeChan:   make(chan struct{}),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// SetOnConnect sets the callback for when connection is established
func (w *WebSocketClient) SetOnConnect(callback func()) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onConnect = callback
}

// SetOnDisconnect sets the callback for when connection is lost
func (w *WebSocketClient) SetOnDisconnect(callback func(error)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onDisconnect = callback
}

// SetOnMessage sets the callback for incoming messages
func (w *WebSocketClient) SetOnMessage(callback func([]byte)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onMessage = callback
}

// SetOnError sets the callback for errors
func (w *WebSocketClient) SetOnError(callback func(error)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onError = callback
}

// Connect establishes a WebSocket connection with automatic reconnection
func (w *WebSocketClient) Connect() error {
	w.mu.Lock()
	if w.state == StateConnected || w.state == StateConnecting {
		w.mu.Unlock()
		return ErrWebSocketAlreadyConnected
	}
	w.state = StateConnecting
	w.mu.Unlock()

	w.wg.Add(1)
	go w.connectLoop()

	return nil
}

// connectLoop handles connection and reconnection logic
func (w *WebSocketClient) connectLoop() {
	defer w.wg.Done()

	for {
		select {
		case <-w.ctx.Done():
			return
		default:
		}

		// Attempt connection
		err := w.attemptConnection()
		if err == nil {
			// Connection successful, start message handling
			w.wg.Add(2)
			go w.readPump()
			go w.writePump()

			// Notify connection established
			w.mu.RLock()
			onConnect := w.onConnect
			w.mu.RUnlock()
			if onConnect != nil {
				onConnect()
			}

			// Wait for connection to close
			<-w.closeChan

			// Notify disconnection
			w.mu.RLock()
			onDisconnect := w.onDisconnect
			lastError := w.lastError
			w.mu.RUnlock()
			if onDisconnect != nil {
				onDisconnect(lastError)
			}
		} else {
			// Connection failed
			w.mu.Lock()
			w.lastError = err
			w.mu.Unlock()

			w.mu.RLock()
			onError := w.onError
			w.mu.RUnlock()
			if onError != nil {
				onError(err)
			}
		}

		// Check if we should stop reconnecting
		w.mu.RLock()
		attempts := w.reconnectAttempts
		maxAttempts := w.config.MaxReconnectAttempts
		w.mu.RUnlock()

		if maxAttempts > 0 && attempts >= maxAttempts {
			logger.Error("Max reconnection attempts reached, stopping",
				logger.Int("attempts", attempts),
				logger.Int("max", maxAttempts),
			)
			return
		}

		// Calculate backoff delay
		delay := w.calculateBackoff()

		logger.Info("Reconnecting WebSocket",
			logger.String("url", w.config.URL),
			logger.Duration("delay", delay),
			logger.Int("attempt", attempts+1),
		)

		select {
		case <-w.ctx.Done():
			return
		case <-time.After(delay):
			w.mu.Lock()
			w.state = StateReconnecting
			w.reconnectAttempts++
			w.mu.Unlock()
		}
	}
}

// attemptConnection attempts to establish a WebSocket connection
func (w *WebSocketClient) attemptConnection() error {
	w.mu.Lock()
	w.state = StateConnecting
	w.mu.Unlock()

	logger.Info("Connecting to WebSocket", logger.String("url", w.config.URL))

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(w.config.URL, nil)
	if err != nil {
		w.mu.Lock()
		w.state = StateDisconnected
		w.mu.Unlock()
		return fmt.Errorf("failed to dial WebSocket: %w", err)
	}

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(w.config.PongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(w.config.PongWait))
		return nil
	})

	w.mu.Lock()
	w.conn = conn
	w.state = StateConnected
	w.reconnectAttempts = 0
	w.lastError = nil
	w.closeChan = make(chan struct{})
	w.mu.Unlock()

	logger.Info("WebSocket connected", logger.String("url", w.config.URL))
	return nil
}

// calculateBackoff calculates exponential backoff delay
func (w *WebSocketClient) calculateBackoff() time.Duration {
	w.mu.RLock()
	attempts := w.reconnectAttempts
	baseDelay := w.config.ReconnectDelay
	maxDelay := w.config.MaxReconnectDelay
	w.mu.RUnlock()

	// Exponential backoff: baseDelay * 2^attempts
	delay := baseDelay * time.Duration(1<<uint(attempts))
	if delay > maxDelay {
		delay = maxDelay
	}

	return delay
}

// readPump handles reading messages from the WebSocket
func (w *WebSocketClient) readPump() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.config.PingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		default:
		}

		w.mu.RLock()
		conn := w.conn
		w.mu.RUnlock()

		if conn == nil {
			return
		}

		// Set read deadline
		conn.SetReadDeadline(time.Now().Add(w.config.ReadTimeout))

		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error("WebSocket read error", logger.ErrorField(err))
			}
			w.closeConnection(err)
			return
		}

		if messageType == websocket.TextMessage || messageType == websocket.BinaryMessage {
			// Send to message channel
			select {
			case w.messageChan <- message:
			case <-w.ctx.Done():
				return
			default:
				logger.Warn("Message channel full, dropping message")
			}

			// Call onMessage callback
			w.mu.RLock()
			onMessage := w.onMessage
			w.mu.RUnlock()
			if onMessage != nil {
				onMessage(message)
			}
		}
	}
}

// writePump handles writing messages and ping/pong
func (w *WebSocketClient) writePump() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.config.PingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			// Send ping
			w.mu.RLock()
			conn := w.conn
			w.mu.RUnlock()

			if conn == nil {
				return
			}

			if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(w.config.WriteTimeout)); err != nil {
				logger.Error("Failed to send ping", logger.ErrorField(err))
				w.closeConnection(err)
				return
			}
		}
	}
}

// SendMessage sends a message through the WebSocket
func (w *WebSocketClient) SendMessage(message []byte) error {
	w.mu.RLock()
	conn := w.conn
	state := w.state
	w.mu.RUnlock()

	if state != StateConnected || conn == nil {
		return ErrWebSocketNotConnected
	}

	conn.SetWriteDeadline(time.Now().Add(w.config.WriteTimeout))
	return conn.WriteMessage(websocket.TextMessage, message)
}

// GetMessageChan returns the channel for receiving messages
func (w *WebSocketClient) GetMessageChan() <-chan []byte {
	return w.messageChan
}

// GetErrorChan returns the channel for receiving errors
func (w *WebSocketClient) GetErrorChan() <-chan error {
	return w.errorChan
}

// GetState returns the current connection state
func (w *WebSocketClient) GetState() WebSocketState {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.state
}

// IsConnected returns whether the WebSocket is connected
func (w *WebSocketClient) IsConnected() bool {
	return w.GetState() == StateConnected
}

// GetLastError returns the last error that occurred
func (w *WebSocketClient) GetLastError() error {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.lastError
}

// GetReconnectAttempts returns the number of reconnection attempts
func (w *WebSocketClient) GetReconnectAttempts() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.reconnectAttempts
}

// closeConnection closes the WebSocket connection
func (w *WebSocketClient) closeConnection(err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.conn != nil {
		w.conn.Close()
		w.conn = nil
	}
	w.state = StateDisconnected
	w.lastError = err

	// Only close channel if it's not already closed
	select {
	case <-w.closeChan:
		// Channel already closed, create new one
		w.closeChan = make(chan struct{})
	default:
		// Channel is open, close it
		close(w.closeChan)
		w.closeChan = make(chan struct{})
	}
}

// Close closes the WebSocket connection and stops reconnection attempts
func (w *WebSocketClient) Close() error {
	w.cancel()

	w.mu.Lock()
	if w.conn != nil {
		w.conn.Close()
		w.conn = nil
	}
	w.state = StateDisconnected
	close(w.closeChan)
	w.mu.Unlock()

	w.wg.Wait()
	return nil
}

