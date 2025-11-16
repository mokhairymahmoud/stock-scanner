package data

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mohamedkhairy/stock-scanner/internal/data"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// echoServer is a simple WebSocket echo server for testing
func echoServer(t *testing.T) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("Upgrade error: %v", err)
			return
		}
		defer conn.Close()

		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				break
			}
			if err := conn.WriteMessage(messageType, message); err != nil {
				break
			}
		}
	}))
	return server
}

func TestWebSocketClient_Connect(t *testing.T) {
	server := echoServer(t)
	defer server.Close()

	wsURL := "ws" + server.URL[4:] // Convert http to ws
	config := data.DefaultWebSocketConfig(wsURL)
	config.MaxReconnectAttempts = 3

	client := data.NewWebSocketClient(config)

	// Test initial state
	assert.False(t, client.IsConnected())
	assert.Equal(t, data.StateDisconnected, client.GetState())

	// Test connect
	err := client.Connect()
	require.NoError(t, err)

	// Wait for connection
	time.Sleep(100 * time.Millisecond)
	assert.True(t, client.IsConnected())
	assert.Equal(t, data.StateConnected, client.GetState())

	// Test double connect
	err = client.Connect()
	assert.ErrorIs(t, err, data.ErrWebSocketAlreadyConnected)

	// Cleanup
	client.Close()
	time.Sleep(50 * time.Millisecond)
	assert.False(t, client.IsConnected())
}

func TestWebSocketClient_MessageHandling(t *testing.T) {
	server := echoServer(t)
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	config := data.DefaultWebSocketConfig(wsURL)
	client := data.NewWebSocketClient(config)

	messages := make([][]byte, 0)
	client.SetOnMessage(func(msg []byte) {
		messages = append(messages, msg)
	})

	err := client.Connect()
	require.NoError(t, err)

	// Wait for connection
	time.Sleep(100 * time.Millisecond)
	require.True(t, client.IsConnected())

	// Send a message
	testMessage := []byte("test message")
	err = client.SendMessage(testMessage)
	require.NoError(t, err)

	// Wait for echo
	time.Sleep(100 * time.Millisecond)

	// Check if message was received
	select {
	case msg := <-client.GetMessageChan():
		assert.Equal(t, testMessage, msg)
	case <-time.After(1 * time.Second):
		t.Fatal("Message not received")
	}

	// Check callback was called
	assert.Greater(t, len(messages), 0)

	client.Close()
}

func TestWebSocketClient_Reconnection(t *testing.T) {
	// Test reconnection by using an invalid URL initially, then verify it attempts to reconnect
	config := data.DefaultWebSocketConfig("ws://invalid-url-that-does-not-exist")
	config.ReconnectDelay = 50 * time.Millisecond
	config.MaxReconnectDelay = 200 * time.Millisecond
	config.MaxReconnectAttempts = 3

	client := data.NewWebSocketClient(config)

	err := client.Connect()
	require.NoError(t, err)

	// Wait for reconnection attempts to start
	time.Sleep(500 * time.Millisecond)

	// Verify reconnection attempts are being made
	attempts := client.GetReconnectAttempts()
	assert.Greater(t, attempts, 0, "Reconnection attempts should be made for invalid URL")

	// Verify state is appropriate for reconnection scenario
	state := client.GetState()
	assert.True(t, state == data.StateReconnecting || state == data.StateConnecting || state == data.StateDisconnected,
		"State should reflect reconnection scenario, got: %v", state)

	// Cleanup
	client.Close()
}

func TestWebSocketClient_ExponentialBackoff(t *testing.T) {
	config := data.DefaultWebSocketConfig("ws://invalid-url")
	config.ReconnectDelay = 10 * time.Millisecond
	config.MaxReconnectDelay = 100 * time.Millisecond
	config.MaxReconnectAttempts = 3

	client := data.NewWebSocketClient(config)

	err := client.Connect()
	require.NoError(t, err)

	// Wait for reconnection attempts
	time.Sleep(500 * time.Millisecond)

	attempts := client.GetReconnectAttempts()
	assert.GreaterOrEqual(t, attempts, 1)
	assert.LessOrEqual(t, attempts, 3)

	client.Close()
}

func TestWebSocketClient_Close(t *testing.T) {
	server := echoServer(t)
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	config := data.DefaultWebSocketConfig(wsURL)
	client := data.NewWebSocketClient(config)

	err := client.Connect()
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	require.True(t, client.IsConnected())

	// Close client
	err = client.Close()
	require.NoError(t, err)

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Verify disconnected
	assert.False(t, client.IsConnected())
	assert.Equal(t, data.StateDisconnected, client.GetState())

	// Verify can't send messages
	err = client.SendMessage([]byte("test"))
	assert.ErrorIs(t, err, data.ErrWebSocketNotConnected)
}

func TestWebSocketClient_Callbacks(t *testing.T) {
	server := echoServer(t)
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	config := data.DefaultWebSocketConfig(wsURL)
	client := data.NewWebSocketClient(config)

	onConnectCalled := false
	onDisconnectCalled := false

	client.SetOnConnect(func() {
		onConnectCalled = true
	})
	client.SetOnDisconnect(func(err error) {
		onDisconnectCalled = true
	})

	err := client.Connect()
	require.NoError(t, err)

	// Wait for connection
	time.Sleep(100 * time.Millisecond)
	assert.True(t, onConnectCalled)

	// Close to trigger disconnect callback
	client.Close()
	time.Sleep(50 * time.Millisecond)
	assert.True(t, onDisconnectCalled)
}

func TestWebSocketClient_StateMonitoring(t *testing.T) {
	config := data.DefaultWebSocketConfig("ws://invalid-url")
	config.MaxReconnectAttempts = 1
	client := data.NewWebSocketClient(config)

	// Test initial state
	assert.Equal(t, data.StateDisconnected, client.GetState())
	assert.False(t, client.IsConnected())
	assert.Nil(t, client.GetLastError())

	// Start connection attempt
	err := client.Connect()
	require.NoError(t, err)

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Check state transitions
	state := client.GetState()
	assert.True(t, state == data.StateConnecting || state == data.StateReconnecting || state == data.StateDisconnected)

	client.Close()
}

func TestDefaultWebSocketConfig(t *testing.T) {
	url := "ws://test.example.com"
	config := data.DefaultWebSocketConfig(url)

	assert.Equal(t, url, config.URL)
	assert.Equal(t, 1*time.Second, config.ReconnectDelay)
	assert.Equal(t, 30*time.Second, config.MaxReconnectDelay)
	assert.Equal(t, 30*time.Second, config.HeartbeatInterval)
	assert.Equal(t, 60*time.Second, config.ReadTimeout)
	assert.Equal(t, 10*time.Second, config.WriteTimeout)
	assert.Equal(t, 0, config.MaxReconnectAttempts) // Unlimited
}
