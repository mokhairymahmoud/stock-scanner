package data

import (
	"context"
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockProvider_Connect(t *testing.T) {
	provider, err := NewMockProvider(ProviderConfig{})
	require.NoError(t, err)

	// Test initial state
	assert.False(t, provider.IsConnected())

	// Test connect
	ctx := context.Background()
	err = provider.Connect(ctx)
	require.NoError(t, err)
	assert.True(t, provider.IsConnected())

	// Test double connect
	err = provider.Connect(ctx)
	assert.ErrorIs(t, err, ErrProviderAlreadyConnected)

	// Test close
	err = provider.Close()
	require.NoError(t, err)
	assert.False(t, provider.IsConnected())
}

func TestMockProvider_Subscribe(t *testing.T) {
	provider, err := NewMockProvider(ProviderConfig{})
	require.NoError(t, err)

	ctx := context.Background()
	err = provider.Connect(ctx)
	require.NoError(t, err)

	// Test subscribe before connect
	disconnectedProvider, _ := NewMockProvider(ProviderConfig{})
	_, err = disconnectedProvider.Subscribe(ctx, []string{"AAPL"})
	assert.ErrorIs(t, err, ErrProviderNotConnected)

	// Test subscribe with valid symbols
	symbols := []string{"AAPL", "MSFT", "GOOGL"}
	tickChan, err := provider.Subscribe(ctx, symbols)
	require.NoError(t, err)
	assert.NotNil(t, tickChan)

	// Verify subscribed symbols
	mockProvider := provider.(*MockProvider)
	subscribed := mockProvider.GetSubscribedSymbols()
	assert.Len(t, subscribed, 3)
	for _, symbol := range symbols {
		assert.Contains(t, subscribed, symbol)
	}

	// Test subscribe with empty symbol
	_, err = provider.Subscribe(ctx, []string{""})
	assert.ErrorIs(t, err, ErrInvalidSymbol)

	// Cleanup
	provider.Close()
}

func TestMockProvider_Unsubscribe(t *testing.T) {
	provider, err := NewMockProvider(ProviderConfig{})
	require.NoError(t, err)

	ctx := context.Background()
	err = provider.Connect(ctx)
	require.NoError(t, err)

	// Subscribe to symbols
	symbols := []string{"AAPL", "MSFT", "GOOGL"}
	_, err = provider.Subscribe(ctx, symbols)
	require.NoError(t, err)

	// Unsubscribe from one symbol
	err = provider.Unsubscribe(ctx, []string{"AAPL"})
	require.NoError(t, err)

	// Verify
	mockProvider := provider.(*MockProvider)
	subscribed := mockProvider.GetSubscribedSymbols()
	assert.Len(t, subscribed, 2)
	assert.NotContains(t, subscribed, "AAPL")
	assert.Contains(t, subscribed, "MSFT")
	assert.Contains(t, subscribed, "GOOGL")

	// Cleanup
	provider.Close()
}

func TestMockProvider_TickGeneration(t *testing.T) {
	provider, err := NewMockProvider(ProviderConfig{})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = provider.Connect(ctx)
	require.NoError(t, err)

	// Subscribe to a symbol
	tickChan, err := provider.Subscribe(ctx, []string{"AAPL"})
	require.NoError(t, err)

	// Collect ticks
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
				}
			}
		}
	}()

	// Wait for some ticks
	<-ctx.Done()
	<-done

	// Verify we received ticks
	assert.Greater(t, len(ticks), 0, "Should receive at least one tick")

	// Verify tick properties
	for _, tick := range ticks {
		assert.Equal(t, "AAPL", tick.Symbol)
		assert.Greater(t, tick.Price, 0.0)
		assert.Greater(t, tick.Size, int64(0))
		assert.False(t, tick.Timestamp.IsZero())
		assert.Equal(t, "trade", tick.Type)
		assert.NoError(t, tick.Validate())
	}

	provider.Close()
}

func TestProviderFactory_CreateProvider(t *testing.T) {
	factory := NewProviderFactory()

	// Test creating mock provider
	config := ProviderConfig{
		APIKey: "test-key",
	}
	provider, err := factory.CreateProvider("mock", config)
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, "mock", provider.GetName())

	// Test unknown provider
	_, err = factory.CreateProvider("unknown", config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider type")
}

func TestProviderFactory_RegisterProvider(t *testing.T) {
	factory := NewProviderFactory()

	// Register a custom provider
	customFactory := func(config ProviderConfig) (Provider, error) {
		return NewMockProvider(config)
	}

	err := factory.RegisterProvider("custom", customFactory)
	require.NoError(t, err)

	// Test creating custom provider
	config := ProviderConfig{}
	provider, err := factory.CreateProvider("custom", config)
	require.NoError(t, err)
	assert.NotNil(t, provider)

	// Test duplicate registration
	err = factory.RegisterProvider("custom", customFactory)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestProviderFactory_ListProviders(t *testing.T) {
	factory := NewProviderFactory()

	providers := factory.ListProviders()
	assert.Contains(t, providers, "mock")
	assert.GreaterOrEqual(t, len(providers), 1)
}
