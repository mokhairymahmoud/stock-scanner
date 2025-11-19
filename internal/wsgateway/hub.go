package wsgateway

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mohamedkhairy/stock-scanner/internal/config"
	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

// Hub manages WebSocket connections and broadcasts alerts
type Hub struct {
	config         config.WSGatewayConfig
	registry       *ConnectionRegistry
	redis          storage.RedisClient
	alertStream    string
	consumerGroup  string
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	mu             sync.RWMutex
	running        bool
	stats          HubStats
}

// HubStats holds statistics about the hub
type HubStats struct {
	ConnectionsTotal    int64
	ConnectionsActive   int64
	AlertsReceived      int64
	AlertsBroadcast     int64
	AlertsDropped       int64
	MessagesSent        int64
	MessagesFailed      int64
	LastAlertTime       time.Time
	mu                  sync.RWMutex
}

// NewHub creates a new WebSocket hub
func NewHub(config config.WSGatewayConfig, redis storage.RedisClient, alertStream string, consumerGroup string) *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	return &Hub{
		config:        config,
		registry:      NewConnectionRegistry(),
		redis:         redis,
		alertStream:   alertStream,
		consumerGroup: consumerGroup,
		ctx:           ctx,
		cancel:        cancel,
		stats:         HubStats{},
	}
}

// Start starts the hub (consumes alerts and broadcasts)
func (h *Hub) Start() error {
	h.mu.Lock()
	if h.running {
		h.mu.Unlock()
		return nil
	}
	h.running = true
	h.mu.Unlock()

	logger.Info("Starting WebSocket hub",
		logger.String("alert_stream", h.alertStream),
		logger.String("consumer_group", h.consumerGroup),
	)

	// Start consuming alerts from stream
	h.wg.Add(1)
	go h.consumeAlerts()

	// Start connection health monitor
	h.wg.Add(1)
	go h.monitorConnections()

	return nil
}

// Stop stops the hub
func (h *Hub) Stop() {
	h.mu.Lock()
	if !h.running {
		h.mu.Unlock()
		return
	}
	h.running = false
	h.mu.Unlock()

	logger.Info("Stopping WebSocket hub")
	h.cancel()
	h.wg.Wait()
	logger.Info("WebSocket hub stopped")
}

// Register registers a new connection
func (h *Hub) Register(conn *Connection) {
	h.registry.Add(conn)
	h.incrementConnectionsTotal()
	h.incrementConnectionsActive()

	logger.Info("Connection registered",
		logger.String("connection_id", conn.ID),
		logger.String("user_id", conn.UserID),
		logger.Int("total_connections", h.registry.Count()),
	)

	// Start connection handlers
	h.wg.Add(2)
	go h.writePump(conn)
	go h.readPump(conn)
}

// Unregister unregisters a connection
func (h *Hub) Unregister(conn *Connection) {
	h.registry.Remove(conn.ID)
	h.decrementConnectionsActive()
	conn.Close()

	logger.Info("Connection unregistered",
		logger.String("connection_id", conn.ID),
		logger.String("user_id", conn.UserID),
		logger.Int("total_connections", h.registry.Count()),
	)
}

// consumeAlerts consumes alerts from the filtered stream and broadcasts them
func (h *Hub) consumeAlerts() {
	defer h.wg.Done()

	messageChan, err := h.redis.ConsumeFromStream(
		h.ctx,
		h.alertStream,
		h.consumerGroup,
		"ws-gateway-1",
	)
	if err != nil {
		logger.Error("Failed to start consuming alerts",
			logger.ErrorField(err),
			logger.String("stream", h.alertStream),
		)
		return
	}

	for {
		select {
		case <-h.ctx.Done():
			return

		case msg, ok := <-messageChan:
			if !ok {
				logger.Warn("Alert message channel closed")
				return
			}

			// Deserialize alert
			alert, err := h.deserializeAlert(msg)
			if err != nil {
				logger.Error("Failed to deserialize alert",
					logger.ErrorField(err),
					logger.String("message_id", msg.ID),
				)
				continue
			}

			h.incrementAlertsReceived()
			h.broadcastAlert(alert)

			// Acknowledge message
			ackCtx, ackCancel := context.WithTimeout(context.Background(), 5*time.Second)
			err = h.redis.AcknowledgeMessage(ackCtx, h.alertStream, h.consumerGroup, msg.ID)
			ackCancel()
			if err != nil {
				logger.Warn("Failed to acknowledge alert message",
					logger.ErrorField(err),
					logger.String("message_id", msg.ID),
				)
			}
		}
	}
}

// broadcastAlert broadcasts an alert to all subscribed connections
func (h *Hub) broadcastAlert(alert *models.Alert) {
	connections := h.registry.GetAll()
	sent := 0
	dropped := 0

	for _, conn := range connections {
		if conn.ShouldReceiveAlert(alert) {
			err := conn.SendAlert(alert)
			if err != nil {
				dropped++
				logger.Debug("Failed to send alert to connection",
					logger.ErrorField(err),
					logger.String("connection_id", conn.ID),
				)
			} else {
				sent++
				h.incrementMessagesSent()
			}
		}
	}

	h.incrementAlertsBroadcast()
	if dropped > 0 {
		h.incrementAlertsDropped(int64(dropped))
	}

	logger.Debug("Broadcast alert",
		logger.String("alert_id", alert.ID),
		logger.String("symbol", alert.Symbol),
		logger.Int("sent", sent),
		logger.Int("dropped", dropped),
		logger.Int("total_connections", len(connections)),
	)
}

// deserializeAlert deserializes a stream message into an Alert
func (h *Hub) deserializeAlert(msg storage.StreamMessage) (*models.Alert, error) {
	// Try to get alert from message values
	alertValue, ok := msg.Values["alert"]
	if !ok {
		return nil, fmt.Errorf("alert field not found in message")
	}

	alertStr, ok := alertValue.(string)
	if !ok {
		return nil, fmt.Errorf("alert field is not a string")
	}

	var alert models.Alert
	if err := json.Unmarshal([]byte(alertStr), &alert); err != nil {
		return nil, fmt.Errorf("failed to unmarshal alert: %w", err)
	}

	return &alert, nil
}

// writePump pumps messages from the hub to the WebSocket connection
func (h *Hub) writePump(conn *Connection) {
	defer h.wg.Done()
	defer h.Unregister(conn)

	ticker := time.NewTicker(h.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return

		case message, ok := <-conn.Send:
			conn.Conn.SetWriteDeadline(time.Now().Add(h.config.WriteTimeout))
			if !ok {
				// Channel closed
				conn.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := conn.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current message
			n := len(conn.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-conn.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			conn.Conn.SetWriteDeadline(time.Now().Add(h.config.WriteTimeout))
			if err := conn.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump pumps messages from the WebSocket connection to the hub
func (h *Hub) readPump(conn *Connection) {
	defer h.wg.Done()
	defer h.Unregister(conn)

	conn.Conn.SetReadDeadline(time.Now().Add(h.config.ReadTimeout))
	conn.Conn.SetPongHandler(func(string) error {
		conn.UpdateLastPong()
		conn.Conn.SetReadDeadline(time.Now().Add(h.config.ReadTimeout))
		return nil
	})

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Debug("WebSocket error",
					logger.ErrorField(err),
					logger.String("connection_id", conn.ID),
				)
			}
			break
		}

		// Parse client message
		var clientMsg ClientMessage
		if err := json.Unmarshal(message, &clientMsg); err != nil {
			conn.SendError("invalid_message", "failed to parse message")
			continue
		}

		// Handle client message
		if err := conn.HandleClientMessage(&clientMsg); err != nil {
			logger.Debug("Failed to handle client message",
				logger.ErrorField(err),
				logger.String("connection_id", conn.ID),
			)
		}
	}
}

// monitorConnections monitors connection health and removes stale connections
func (h *Hub) monitorConnections() {
	defer h.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return

		case <-ticker.C:
			connections := h.registry.GetAll()
			now := time.Now()
			staleThreshold := h.config.ReadTimeout * 2

			for _, conn := range connections {
				lastPong := conn.GetLastPong()
				if now.Sub(lastPong) > staleThreshold {
					logger.Info("Removing stale connection",
						logger.String("connection_id", conn.ID),
						logger.String("user_id", conn.UserID),
						logger.Duration("idle_time", now.Sub(lastPong)),
					)
					h.Unregister(conn)
				}
			}
		}
	}
}

// GetStats returns hub statistics
func (h *Hub) GetStats() HubStats {
	h.stats.mu.RLock()
	defer h.stats.mu.RUnlock()

	// Update active connections count
	h.stats.ConnectionsActive = int64(h.registry.Count())

	// Return a copy
	return HubStats{
		ConnectionsTotal:  h.stats.ConnectionsTotal,
		ConnectionsActive: int64(h.registry.Count()),
		AlertsReceived:    h.stats.AlertsReceived,
		AlertsBroadcast:   h.stats.AlertsBroadcast,
		AlertsDropped:     h.stats.AlertsDropped,
		MessagesSent:      h.stats.MessagesSent,
		MessagesFailed:    h.stats.MessagesFailed,
		LastAlertTime:     h.stats.LastAlertTime,
	}
}

// Stats increment methods
func (h *Hub) incrementConnectionsTotal() {
	h.stats.mu.Lock()
	defer h.stats.mu.Unlock()
	h.stats.ConnectionsTotal++
}

func (h *Hub) incrementConnectionsActive() {
	h.stats.mu.Lock()
	defer h.stats.mu.Unlock()
	h.stats.ConnectionsActive++
}

func (h *Hub) decrementConnectionsActive() {
	h.stats.mu.Lock()
	defer h.stats.mu.Unlock()
	if h.stats.ConnectionsActive > 0 {
		h.stats.ConnectionsActive--
	}
}

func (h *Hub) incrementAlertsReceived() {
	h.stats.mu.Lock()
	defer h.stats.mu.Unlock()
	h.stats.AlertsReceived++
	h.stats.LastAlertTime = time.Now()
}

func (h *Hub) incrementAlertsBroadcast() {
	h.stats.mu.Lock()
	defer h.stats.mu.Unlock()
	h.stats.AlertsBroadcast++
}

func (h *Hub) incrementAlertsDropped(count int64) {
	h.stats.mu.Lock()
	defer h.stats.mu.Unlock()
	h.stats.AlertsDropped += count
}

func (h *Hub) incrementMessagesSent() {
	h.stats.mu.Lock()
	defer h.stats.mu.Unlock()
	h.stats.MessagesSent++
}

func (h *Hub) incrementMessagesFailed() {
	h.stats.mu.Lock()
	defer h.stats.mu.Unlock()
	h.stats.MessagesFailed++
}

