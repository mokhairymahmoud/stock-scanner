package wsgateway

import (
	"encoding/json"
	"fmt"

	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

// MessageType represents the type of WebSocket message
type MessageType string

const (
	MessageTypeSubscribe        MessageType = "subscribe"
	MessageTypeUnsubscribe      MessageType = "unsubscribe"
	MessageTypeSubscribeToplist MessageType = "subscribe_toplist"
	MessageTypeUnsubscribeToplist MessageType = "unsubscribe_toplist"
	MessageTypePing             MessageType = "ping"
	MessageTypePong             MessageType = "pong"
)

// ClientMessage represents a message from the client
type ClientMessage struct {
	Type    string          `json:"type"`
	Symbol  string          `json:"symbol,omitempty"`
	Symbols []string        `json:"symbols,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// ServerMessage represents a message to the client
type ServerMessage struct {
	Type    string      `json:"type"`
	Data    interface{} `json:"data,omitempty"`
	Code    string      `json:"code,omitempty"`
	Message string      `json:"message,omitempty"`
}

// HandleClientMessage handles a message from the client
func (c *Connection) HandleClientMessage(msg *ClientMessage) error {
	switch MessageType(msg.Type) {
	case MessageTypeSubscribe:
		if msg.Symbol != "" {
			c.Subscribe(msg.Symbol)
			logger.Debug("Client subscribed to symbol",
				logger.String("connection_id", c.ID),
				logger.String("user_id", c.UserID),
				logger.String("symbol", msg.Symbol),
			)
			return c.SendSuccess("subscribed", map[string]string{"symbol": msg.Symbol})
		} else if len(msg.Symbols) > 0 {
			for _, symbol := range msg.Symbols {
				c.Subscribe(symbol)
			}
			logger.Debug("Client subscribed to symbols",
				logger.String("connection_id", c.ID),
				logger.String("user_id", c.UserID),
				logger.Int("count", len(msg.Symbols)),
			)
			return c.SendSuccess("subscribed", map[string]interface{}{"symbols": msg.Symbols})
		}
		return c.SendError("invalid_request", "symbol or symbols field required")

	case MessageTypeUnsubscribe:
		if msg.Symbol != "" {
			c.Unsubscribe(msg.Symbol)
			logger.Debug("Client unsubscribed from symbol",
				logger.String("connection_id", c.ID),
				logger.String("user_id", c.UserID),
				logger.String("symbol", msg.Symbol),
			)
			return c.SendSuccess("unsubscribed", map[string]string{"symbol": msg.Symbol})
		} else if len(msg.Symbols) > 0 {
			for _, symbol := range msg.Symbols {
				c.Unsubscribe(symbol)
			}
			logger.Debug("Client unsubscribed from symbols",
				logger.String("connection_id", c.ID),
				logger.String("user_id", c.UserID),
				logger.Int("count", len(msg.Symbols)),
			)
			return c.SendSuccess("unsubscribed", map[string]interface{}{"symbols": msg.Symbols})
		}
		return c.SendError("invalid_request", "symbol or symbols field required")

	case MessageTypeSubscribeToplist:
		toplistID := msg.Symbol // Reuse Symbol field for toplist ID
		if toplistID == "" {
			return c.SendError("invalid_request", "toplist_id field required")
		}
		c.SubscribeToplist(toplistID)
		logger.Debug("Client subscribed to toplist",
			logger.String("connection_id", c.ID),
			logger.String("user_id", c.UserID),
			logger.String("toplist_id", toplistID),
		)
		return c.SendSuccess("subscribed_toplist", map[string]string{"toplist_id": toplistID})

	case MessageTypeUnsubscribeToplist:
		toplistID := msg.Symbol // Reuse Symbol field for toplist ID
		if toplistID == "" {
			return c.SendError("invalid_request", "toplist_id field required")
		}
		c.UnsubscribeToplist(toplistID)
		logger.Debug("Client unsubscribed from toplist",
			logger.String("connection_id", c.ID),
			logger.String("user_id", c.UserID),
			logger.String("toplist_id", toplistID),
		)
		return c.SendSuccess("unsubscribed_toplist", map[string]string{"toplist_id": toplistID})

	case MessageTypePing:
		// Respond with pong
		return c.SendPong()

	default:
		return c.SendError("unknown_message_type", fmt.Sprintf("unknown message type: %s", msg.Type))
	}
}

// SendSuccess sends a success message to the client
func (c *Connection) SendSuccess(action string, data interface{}) error {
	message := ServerMessage{
		Type: "success",
		Data: map[string]interface{}{
			"action": action,
			"data":   data,
		},
	}
	return c.WriteJSON(message)
}

// SendPong sends a pong message to the client
func (c *Connection) SendPong() error {
	message := ServerMessage{
		Type: "pong",
	}
	return c.WriteJSON(message)
}

