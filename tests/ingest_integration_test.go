package data

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/data"
	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/pubsub"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIngestService_Integration(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup mock Redis
	mockRedis := storage.NewMockRedisClient()

	// Create provider
	provider, err := data.NewMockProvider(data.ProviderConfig{})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Connect provider
	err = provider.Connect(ctx)
	require.NoError(t, err)
	defer provider.Close()

	// Subscribe to symbols
	tickChan, err := provider.Subscribe(ctx, []string{"AAPL", "MSFT"})
	require.NoError(t, err)

	// Create stream publisher
	publisherConfig := pubsub.DefaultStreamPublisherConfig("test-ticks")
	publisherConfig.BatchSize = 10
	publisherConfig.BatchTimeout = 100 * time.Millisecond

	publisher := pubsub.NewStreamPublisher(mockRedis, publisherConfig)
	publisher.Start()
	defer publisher.Close()

	// Process a few ticks
	tickCount := 0
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-ctx.Done():
				done <- true
				return
			case tick, ok := <-tickChan:
				if !ok {
					done <- true
					return
				}
				if tick != nil {
					err := publisher.Publish(tick)
					if err == nil {
						tickCount++
					}
					if tickCount >= 5 {
						done <- true
						return
					}
				}
			}
		}
	}()

	// Wait for ticks
	select {
	case <-done:
	case <-ctx.Done():
	}

	// Verify we processed some ticks
	assert.Greater(t, tickCount, 0, "Should process at least one tick")

	// Flush publisher
	err = publisher.Flush()
	require.NoError(t, err)
}

func TestHealthCheckEndpoint(t *testing.T) {
	// Create mock provider
	provider, err := data.NewMockProvider(data.ProviderConfig{})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = provider.Connect(ctx)
	require.NoError(t, err)
	defer provider.Close()

	// Create mock publisher
	mockRedis := storage.NewMockRedisClient()
	publisher := pubsub.NewStreamPublisher(mockRedis, pubsub.DefaultStreamPublisherConfig("test"))
	publisher.Start()
	defer publisher.Close()

	// Create a simple health check handler
	healthHandler := func(w http.ResponseWriter, r *http.Request) {
		health := map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
			"checks": map[string]interface{}{
				"provider": map[string]interface{}{
					"status":    "ok",
					"connected": provider.IsConnected(),
					"provider":  provider.GetName(),
				},
				"publisher": map[string]interface{}{
					"status":    "ok",
					"batch_size": publisher.GetBatchSize(),
				},
			},
		}

		if !provider.IsConnected() {
			health["status"] = "degraded"
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(health)
	}

	// Test health check
	req, _ := http.NewRequest("GET", "/health", nil)
	w := &mockResponseWriter{}

	healthHandler(w, req)

	assert.Equal(t, http.StatusOK, w.statusCode)
	assert.Contains(t, w.body, "healthy")
	assert.Contains(t, w.body, "provider")
}

type mockResponseWriter struct {
	statusCode int
	body       string
	header     http.Header
}

func (m *mockResponseWriter) Header() http.Header {
	if m.header == nil {
		m.header = make(http.Header)
	}
	return m.header
}

func (m *mockResponseWriter) Write(b []byte) (int, error) {
	m.body = string(b)
	return len(b), nil
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.statusCode = statusCode
}

func TestIngestService_ProviderDisconnection(t *testing.T) {
	// Test that service handles provider disconnection gracefully
	provider, err := data.NewMockProvider(data.ProviderConfig{})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = provider.Connect(ctx)
	require.NoError(t, err)

	// Verify connected
	assert.True(t, provider.IsConnected())

	// Disconnect
	err = provider.Close()
	require.NoError(t, err)

	// Verify disconnected
	assert.False(t, provider.IsConnected())
}

func TestIngestService_StreamPublisherIntegration(t *testing.T) {
	// Test stream publisher with mock provider
	mockRedis := storage.NewMockRedisClient()

	provider, err := data.NewMockProvider(data.ProviderConfig{})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = provider.Connect(ctx)
	require.NoError(t, err)
	defer provider.Close()

	// Subscribe
	tickChan, err := provider.Subscribe(ctx, []string{"AAPL"})
	require.NoError(t, err)

	// Create publisher
	publisherConfig := pubsub.DefaultStreamPublisherConfig("test-stream")
	publisherConfig.BatchSize = 5
	publisherConfig.BatchTimeout = 200 * time.Millisecond

	publisher := pubsub.NewStreamPublisher(mockRedis, publisherConfig)
	publisher.Start()
	defer publisher.Close()

	// Collect ticks and publish
	ticks := make([]*models.Tick, 0)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-ctx.Done():
				done <- true
				return
			case tick, ok := <-tickChan:
				if !ok {
					done <- true
					return
				}
				if tick != nil {
					ticks = append(ticks, tick)
					publisher.Publish(tick)
					if len(ticks) >= 3 {
						done <- true
						return
					}
				}
			}
		}
	}()

	// Wait for ticks
	select {
	case <-done:
	case <-ctx.Done():
	}

	// Verify we got ticks
	assert.GreaterOrEqual(t, len(ticks), 1, "Should receive at least one tick")

	// Flush
	err = publisher.Flush()
	require.NoError(t, err)
}

