package data

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

// MockProvider is a mock implementation of Provider for testing
type MockProvider struct {
	name       string
	config     ProviderConfig
	connected  bool
	subscribed map[string]bool
	tickChan   chan *models.Tick
	mu         sync.RWMutex
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// NewMockProvider creates a new mock provider
func NewMockProvider(config ProviderConfig) (Provider, error) {
	return &MockProvider{
		name:       "mock",
		config:     config,
		connected:  false,
		subscribed: make(map[string]bool),
		tickChan:   make(chan *models.Tick, 100),
	}, nil
}

// Connect establishes a connection (mock - always succeeds)
func (m *MockProvider) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.connected {
		return ErrProviderAlreadyConnected
	}

	m.connected = true
	return nil
}

// Subscribe subscribes to market data for the given symbols
func (m *MockProvider) Subscribe(ctx context.Context, symbols []string) (<-chan *models.Tick, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.connected {
		return nil, ErrProviderNotConnected
	}

	// Validate symbols
	for _, symbol := range symbols {
		if symbol == "" {
			return nil, ErrInvalidSymbol
		}
		m.subscribed[symbol] = true
	}

	// Start generating mock ticks if not already running
	if m.cancel == nil {
		ctx, cancel := context.WithCancel(ctx)
		m.cancel = cancel
		m.wg.Add(1)
		go m.generateTicks(ctx, symbols)
	} else {
		// Update subscribed symbols
		m.wg.Add(1)
		go m.generateTicks(ctx, symbols)
	}

	return m.tickChan, nil
}

// Unsubscribe unsubscribes from market data for the given symbols
func (m *MockProvider) Unsubscribe(ctx context.Context, symbols []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.connected {
		return ErrProviderNotConnected
	}

	for _, symbol := range symbols {
		delete(m.subscribed, symbol)
	}

	return nil
}

// Close closes the connection
func (m *MockProvider) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.connected {
		return nil
	}

	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}

	m.connected = false
	close(m.tickChan)
	m.wg.Wait()

	return nil
}

// IsConnected returns whether the provider is connected
func (m *MockProvider) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected
}

// GetName returns the provider name
func (m *MockProvider) GetName() string {
	return m.name
}

// generateTicks generates mock tick data for subscribed symbols
func (m *MockProvider) generateTicks(ctx context.Context, symbols []string) {
	defer m.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond) // Generate ticks every 100ms
	defer ticker.Stop()

	// Initialize base prices for each symbol
	basePrices := make(map[string]float64)
	for _, symbol := range symbols {
		basePrices[symbol] = 100.0 + rand.Float64()*200.0 // Random price between 100-300
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.mu.RLock()
			subscribed := make([]string, 0, len(m.subscribed))
			for symbol := range m.subscribed {
				subscribed = append(subscribed, symbol)
			}
			m.mu.RUnlock()

			// Generate a tick for each subscribed symbol
			for _, symbol := range subscribed {
				// Update price with small random walk
				basePrice := basePrices[symbol]
				change := (rand.Float64() - 0.5) * 2.0 // -1 to +1
				newPrice := basePrice + change
				if newPrice < 1.0 {
					newPrice = 1.0
				}
				basePrices[symbol] = newPrice

				// Generate tick
				tick := &models.Tick{
					Symbol:    symbol,
					Price:     newPrice,
					Size:      int64(rand.Intn(1000) + 100), // 100-1100 shares
					Timestamp: time.Now().UTC(),
					Type:      "trade",
				}

				// Validate tick
				if err := tick.Validate(); err != nil {
					continue
				}

				// Send tick (non-blocking)
				select {
				case m.tickChan <- tick:
				case <-ctx.Done():
					return
				default:
					// Channel full, skip this tick
				}
			}
		}
	}
}

// GetSubscribedSymbols returns the list of currently subscribed symbols (for testing)
func (m *MockProvider) GetSubscribedSymbols() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	symbols := make([]string, 0, len(m.subscribed))
	for symbol := range m.subscribed {
		symbols = append(symbols, symbol)
	}
	return symbols
}
