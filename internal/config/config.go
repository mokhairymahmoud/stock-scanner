package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	// Common
	Environment string
	LogLevel    string

	// Database
	Database DatabaseConfig

	// Redis
	Redis RedisConfig

	// Market Data
	MarketData MarketDataConfig

	// Services
	Ingest    IngestConfig
	Bars      BarsConfig
	Indicator IndicatorConfig
	Scanner   ScannerConfig
	Alert     AlertConfig
	WSGateway WSGatewayConfig
	API       APIConfig
}

// DatabaseConfig holds TimescaleDB configuration
type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxConnections  int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host         string
	Port         int
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
}

// MarketDataConfig holds market data provider configuration
type MarketDataConfig struct {
	Provider     string // "alpaca", "polygon", etc.
	APIKey       string
	APISecret    string
	BaseURL      string
	WebSocketURL string
	Symbols      []string
}

// IngestConfig holds ingest service configuration
type IngestConfig struct {
	Port              int
	HealthCheckPort   int
	StreamName        string
	BatchSize         int
	BatchTimeout      time.Duration
	ReconnectDelay    time.Duration
	MaxReconnectDelay time.Duration
}

// BarsConfig holds bar aggregator configuration
type BarsConfig struct {
	Port            int
	HealthCheckPort int
	ConsumerGroup   string
	BatchSize       int
	WriteInterval   time.Duration
	// Database write configuration
	DBWriteBatchSize int
	DBWriteInterval  time.Duration
	DBWriteQueueSize int
	DBMaxRetries     int
	DBRetryDelay     time.Duration
}

// IndicatorConfig holds indicator engine configuration
type IndicatorConfig struct {
	Port            int
	HealthCheckPort int
	ConsumerGroup   string
	UpdateInterval  time.Duration
}

// ScannerConfig holds scanner worker configuration
type ScannerConfig struct {
	Port              int
	HealthCheckPort   int
	WorkerID          string
	WorkerCount       int
	ScanInterval      time.Duration
	SymbolUniverse    []string
	CooldownDefault   time.Duration
	BufferSize        int
	RuleStoreType     string        // "memory" or "redis" (default: "memory")
	RuleReloadInterval time.Duration // How often to reload rules from store (default: 30s)
	EnableToplists    bool          // Enable toplist updates (default: true)
	ToplistUpdateInterval time.Duration // Interval for toplist updates (default: 1s)
}

// WSGatewayConfig holds WebSocket gateway configuration
type WSGatewayConfig struct {
	Port            int
	HealthCheckPort int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	PingInterval    time.Duration
	MaxConnections  int
	JWTSecret       string
	AlertStream     string
	ConsumerGroup   string
}

// AlertConfig holds alert service configuration
type AlertConfig struct {
	Port              int
	HealthCheckPort   int
	ConsumerGroup     string
	StreamName        string
	BatchSize         int
	ProcessTimeout    time.Duration
	DedupeTTL         time.Duration
	CooldownTTL       time.Duration
	FilteredStreamName string
	DBWriteBatchSize  int
	DBWriteInterval   time.Duration
	DBWriteQueueSize  int
	DBMaxRetries      int
	DBRetryDelay      time.Duration
}

// APIConfig holds REST API configuration
type APIConfig struct {
	Port            int
	HealthCheckPort int
	JWTSecret       string
	JWTExpiry       time.Duration
	RateLimitRPS    int
}

// Load loads configuration from environment variables
// It automatically loads .env file if it exists in the current directory or parent directories
func Load() (*Config, error) {
	// Try to load .env file (ignore error if it doesn't exist)
	_ = godotenv.Load()

	cfg := &Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnvAsInt("DB_PORT", 5432),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "postgres"),
			Database:        getEnv("DB_NAME", "stock_scanner"),
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			MaxConnections:  getEnvAsInt("DB_MAX_CONNECTIONS", 25),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			Host:         getEnv("REDIS_HOST", "localhost"),
			Port:         getEnvAsInt("REDIS_PORT", 6379),
			Password:     getEnv("REDIS_PASSWORD", ""),
			DB:           getEnvAsInt("REDIS_DB", 0),
			PoolSize:     getEnvAsInt("REDIS_POOL_SIZE", 10),
			MinIdleConns: getEnvAsInt("REDIS_MIN_IDLE_CONNS", 5),
		},
		MarketData: MarketDataConfig{
			Provider:     getEnv("MARKET_DATA_PROVIDER", "alpaca"),
			APIKey:       getEnv("MARKET_DATA_API_KEY", ""),
			APISecret:    getEnv("MARKET_DATA_API_SECRET", ""),
			BaseURL:      getEnv("MARKET_DATA_BASE_URL", ""),
			WebSocketURL: getEnv("MARKET_DATA_WS_URL", ""),
			Symbols:      getEnvAsStringSlice("MARKET_DATA_SYMBOLS", []string{}),
		},
		Ingest: IngestConfig{
			Port:              getEnvAsInt("INGEST_PORT", 8080),
			HealthCheckPort:   getEnvAsInt("INGEST_HEALTH_PORT", 8081),
			StreamName:        getEnv("INGEST_STREAM_NAME", "ticks"),
			BatchSize:         getEnvAsInt("INGEST_BATCH_SIZE", 100),
			BatchTimeout:      getEnvAsDuration("INGEST_BATCH_TIMEOUT", 100*time.Millisecond),
			ReconnectDelay:    getEnvAsDuration("INGEST_RECONNECT_DELAY", 1*time.Second),
			MaxReconnectDelay: getEnvAsDuration("INGEST_MAX_RECONNECT_DELAY", 30*time.Second),
		},
		Bars: BarsConfig{
			Port:            getEnvAsInt("BARS_PORT", 8082),
			HealthCheckPort: getEnvAsInt("BARS_HEALTH_PORT", 8083),
			ConsumerGroup:   getEnv("BARS_CONSUMER_GROUP", "bars-aggregator"),
			BatchSize:       getEnvAsInt("BARS_BATCH_SIZE", 1000),
			WriteInterval:   getEnvAsDuration("BARS_WRITE_INTERVAL", 1*time.Second),
			// Database write configuration
			DBWriteBatchSize: getEnvAsInt("BARS_DB_WRITE_BATCH_SIZE", 1000),
			DBWriteInterval:  getEnvAsDuration("BARS_DB_WRITE_INTERVAL", 1*time.Second),
			DBWriteQueueSize: getEnvAsInt("BARS_DB_WRITE_QUEUE_SIZE", 10000),
			DBMaxRetries:     getEnvAsInt("BARS_DB_MAX_RETRIES", 3),
			DBRetryDelay:     getEnvAsDuration("BARS_DB_RETRY_DELAY", 100*time.Millisecond),
		},
		Indicator: IndicatorConfig{
			Port:            getEnvAsInt("INDICATOR_PORT", 8084),
			HealthCheckPort: getEnvAsInt("INDICATOR_HEALTH_PORT", 8085),
			ConsumerGroup:   getEnv("INDICATOR_CONSUMER_GROUP", "indicator-engine"),
			UpdateInterval:  getEnvAsDuration("INDICATOR_UPDATE_INTERVAL", 1*time.Second),
		},
		Scanner: ScannerConfig{
			Port:              getEnvAsInt("SCANNER_PORT", 8086),
			HealthCheckPort:   getEnvAsInt("SCANNER_HEALTH_PORT", 8087),
			WorkerID:          getEnv("SCANNER_WORKER_ID", "worker-1"),
			WorkerCount:       getEnvAsInt("SCANNER_WORKER_COUNT", 1),
			ScanInterval:      getEnvAsDuration("SCANNER_SCAN_INTERVAL", 1*time.Second),
			SymbolUniverse:    getEnvAsStringSlice("SCANNER_SYMBOL_UNIVERSE", []string{}),
			CooldownDefault:   getEnvAsDuration("SCANNER_COOLDOWN_DEFAULT", 5*time.Minute),
			BufferSize:        getEnvAsInt("SCANNER_BUFFER_SIZE", 1000),
			RuleStoreType:     getEnv("SCANNER_RULE_STORE_TYPE", "memory"), // "memory" or "redis"
			RuleReloadInterval: getEnvAsDuration("SCANNER_RULE_RELOAD_INTERVAL", 30*time.Second),
			EnableToplists:    getEnvAsBool("SCANNER_ENABLE_TOPLISTS", true),
			ToplistUpdateInterval: getEnvAsDuration("SCANNER_TOPLIST_UPDATE_INTERVAL", 1*time.Second),
		},
		Alert: AlertConfig{
			Port:              getEnvAsInt("ALERT_PORT", 8092),
			HealthCheckPort:   getEnvAsInt("ALERT_HEALTH_PORT", 8093),
			ConsumerGroup:     getEnv("ALERT_CONSUMER_GROUP", "alert-service"),
			StreamName:        getEnv("ALERT_STREAM_NAME", "alerts"),
			BatchSize:         getEnvAsInt("ALERT_BATCH_SIZE", 100),
			ProcessTimeout:     getEnvAsDuration("ALERT_PROCESS_TIMEOUT", 5*time.Second),
			DedupeTTL:          getEnvAsDuration("ALERT_DEDUPE_TTL", 1*time.Hour),
			CooldownTTL:        getEnvAsDuration("ALERT_COOLDOWN_TTL", 5*time.Minute),
			FilteredStreamName: getEnv("ALERT_FILTERED_STREAM_NAME", "alerts.filtered"),
			DBWriteBatchSize:   getEnvAsInt("ALERT_DB_WRITE_BATCH_SIZE", 100),
			DBWriteInterval:    getEnvAsDuration("ALERT_DB_WRITE_INTERVAL", 5*time.Second),
			DBWriteQueueSize:   getEnvAsInt("ALERT_DB_WRITE_QUEUE_SIZE", 1000),
			DBMaxRetries:       getEnvAsInt("ALERT_DB_MAX_RETRIES", 3),
			DBRetryDelay:       getEnvAsDuration("ALERT_DB_RETRY_DELAY", 1*time.Second),
		},
		WSGateway: WSGatewayConfig{
			Port:            getEnvAsInt("WS_GATEWAY_PORT", 8088),
			HealthCheckPort: getEnvAsInt("WS_GATEWAY_HEALTH_PORT", 8089),
			ReadTimeout:     getEnvAsDuration("WS_GATEWAY_READ_TIMEOUT", 60*time.Second),
			WriteTimeout:    getEnvAsDuration("WS_GATEWAY_WRITE_TIMEOUT", 10*time.Second),
			PingInterval:    getEnvAsDuration("WS_GATEWAY_PING_INTERVAL", 30*time.Second),
			MaxConnections:  getEnvAsInt("WS_GATEWAY_MAX_CONNECTIONS", 1000),
			JWTSecret:       getEnv("WS_GATEWAY_JWT_SECRET", ""),
			AlertStream:     getEnv("WS_GATEWAY_ALERT_STREAM", "alerts.filtered"),
			ConsumerGroup:   getEnv("WS_GATEWAY_CONSUMER_GROUP", "ws-gateway"),
		},
		API: APIConfig{
			Port:            getEnvAsInt("API_PORT", 8090),
			HealthCheckPort: getEnvAsInt("API_HEALTH_PORT", 8091),
			JWTSecret:       getEnv("API_JWT_SECRET", ""),
			JWTExpiry:       getEnvAsDuration("API_JWT_EXPIRY", 24*time.Hour),
			RateLimitRPS:    getEnvAsInt("API_RATE_LIMIT_RPS", 100),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Database.Host == "" {
		return fmt.Errorf("DB_HOST is required")
	}
	if c.Redis.Host == "" {
		return fmt.Errorf("REDIS_HOST is required")
	}
	if len(c.MarketData.Symbols) == 0 {
		return fmt.Errorf("MARKET_DATA_SYMBOLS must contain at least one symbol")
	}
	if c.MarketData.APIKey == "" {
		return fmt.Errorf("MARKET_DATA_API_KEY is required")
	}
	return nil
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return boolValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return duration
}

func getEnvAsStringSlice(key string, defaultValue []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	if value == "" {
		return defaultValue
	}
	// Split by comma and trim spaces
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return defaultValue
	}
	return result
}
