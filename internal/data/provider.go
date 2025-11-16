package data

import (
	"context"
	"errors"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

var (
	// ErrProviderNotConnected is returned when operations are attempted on a disconnected provider
	ErrProviderNotConnected = errors.New("provider is not connected")
	// ErrProviderAlreadyConnected is returned when attempting to connect an already connected provider
	ErrProviderAlreadyConnected = errors.New("provider is already connected")
	// ErrInvalidSymbol is returned when an invalid symbol is provided
	ErrInvalidSymbol = errors.New("invalid symbol")
)

// Provider defines the interface for market data providers
type Provider interface {
	// Connect establishes a connection to the market data provider
	Connect(ctx context.Context) error

	// Subscribe subscribes to market data for the given symbols
	// Returns a channel that will receive Tick messages
	Subscribe(ctx context.Context, symbols []string) (<-chan *models.Tick, error)

	// Unsubscribe unsubscribes from market data for the given symbols
	Unsubscribe(ctx context.Context, symbols []string) error

	// Close closes the connection to the provider
	Close() error

	// IsConnected returns whether the provider is currently connected
	IsConnected() bool

	// GetName returns the name/type of the provider (e.g., "alpaca", "polygon")
	GetName() string
}

// ProviderFactory creates provider instances
type ProviderFactory interface {
	// CreateProvider creates a new provider instance based on the provider type
	CreateProvider(providerType string, config ProviderConfig) (Provider, error)

	// RegisterProvider registers a custom provider factory function
	RegisterProvider(providerType string, factoryFunc func(ProviderConfig) (Provider, error)) error

	// ListProviders returns a list of available provider types
	ListProviders() []string
}

// ProviderConfig holds configuration for a provider
type ProviderConfig struct {
	// Provider-specific configuration
	APIKey    string
	APISecret string
	BaseURL   string
	WSURL     string

	// Connection settings
	ReconnectDelay    int // in seconds
	MaxReconnectDelay int // in seconds
	HeartbeatInterval int // in seconds
}

// DefaultProviderFactory is the default implementation of ProviderFactory
type DefaultProviderFactory struct {
	factories map[string]func(ProviderConfig) (Provider, error)
}

// NewProviderFactory creates a new provider factory
func NewProviderFactory() *DefaultProviderFactory {
	factory := &DefaultProviderFactory{
		factories: make(map[string]func(ProviderConfig) (Provider, error)),
	}

	// Register built-in providers
	factory.RegisterProvider("mock", NewMockProvider)
	// TODO: Register other providers as they are implemented
	// factory.RegisterProvider("alpaca", NewAlpacaProvider)
	// factory.RegisterProvider("polygon", NewPolygonProvider)

	return factory
}

// CreateProvider creates a new provider instance
func (f *DefaultProviderFactory) CreateProvider(providerType string, config ProviderConfig) (Provider, error) {
	factoryFunc, exists := f.factories[providerType]
	if !exists {
		return nil, errors.New("unknown provider type: " + providerType)
	}

	return factoryFunc(config)
}

// RegisterProvider registers a custom provider factory function
func (f *DefaultProviderFactory) RegisterProvider(providerType string, factoryFunc func(ProviderConfig) (Provider, error)) error {
	if _, exists := f.factories[providerType]; exists {
		return errors.New("provider type already registered: " + providerType)
	}
	f.factories[providerType] = factoryFunc
	return nil
}

// ListProviders returns a list of available provider types
func (f *DefaultProviderFactory) ListProviders() []string {
	providers := make([]string, 0, len(f.factories))
	for providerType := range f.factories {
		providers = append(providers, providerType)
	}
	return providers
}
