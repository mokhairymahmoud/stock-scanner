package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/mohamedkhairy/stock-scanner/internal/config"
	"github.com/mohamedkhairy/stock-scanner/internal/pubsub"
	"github.com/mohamedkhairy/stock-scanner/internal/wsgateway"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// MVP: Allow all origins
		// In production, validate origin
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := logger.Init(cfg.LogLevel, cfg.Environment); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting WebSocket gateway service",
		logger.String("port", fmt.Sprintf("%d", cfg.WSGateway.Port)),
		logger.String("health_port", fmt.Sprintf("%d", cfg.WSGateway.HealthCheckPort)),
		logger.Int("max_connections", cfg.WSGateway.MaxConnections),
	)

	// Initialize Redis client
	redisClient, err := pubsub.NewRedisClient(cfg.Redis)
	if err != nil {
		logger.Fatal("Failed to initialize Redis client",
			logger.ErrorField(err),
		)
	}
	defer redisClient.Close()

	// Initialize auth manager
	authManager := wsgateway.NewAuthManager(cfg.WSGateway.JWTSecret)

	// Initialize hub
	hub := wsgateway.NewHub(cfg.WSGateway, redisClient, cfg.WSGateway.AlertStream, cfg.WSGateway.ConsumerGroup)

	// Start hub
	if err := hub.Start(); err != nil {
		logger.Fatal("Failed to start WebSocket hub",
			logger.ErrorField(err),
		)
	}
	defer hub.Stop()

	// Set up HTTP server
	router := mux.NewRouter()

	// WebSocket endpoint
	router.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(hub, authManager, w, r, cfg.WSGateway)
	})

	// Health check endpoints
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	router.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		stats := hub.GetStats()
		if stats.ConnectionsActive > 0 || stats.AlertsReceived > 0 {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "not ready"})
		}
	})

	router.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
	})

	// Stats endpoint
	router.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		stats := hub.GetStats()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	})

	// Metrics endpoint
	router.Handle("/metrics", promhttp.Handler())

	// Start HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.WSGateway.Port),
		Handler: router,
	}

	go func() {
		logger.Info("Starting HTTP server",
			logger.String("addr", server.Addr),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start HTTP server",
				logger.ErrorField(err),
			)
		}
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	logger.Info("Shutting down WebSocket gateway service")

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Error shutting down HTTP server",
			logger.ErrorField(err),
		)
	}

	logger.Info("WebSocket gateway service stopped")
}

// handleWebSocket handles WebSocket connections
func handleWebSocket(hub *wsgateway.Hub, authManager *wsgateway.AuthManager, w http.ResponseWriter, r *http.Request, config config.WSGatewayConfig) {
	// Check max connections
	stats := hub.GetStats()
	if int(stats.ConnectionsActive) >= config.MaxConnections {
		logger.Warn("Max connections reached, rejecting new connection",
			logger.Int("max_connections", config.MaxConnections),
			logger.Int64("active_connections", stats.ConnectionsActive),
		)
		http.Error(w, "Max connections reached", http.StatusServiceUnavailable)
		return
	}

	// Extract and validate JWT token
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		// Try query parameter as fallback
		authHeader = r.URL.Query().Get("token")
		if authHeader != "" {
			authHeader = "Bearer " + authHeader
		}
	}

	var userID string
	tokenString, err := authManager.ExtractTokenFromHeader(authHeader)
	if err != nil {
		// MVP: If no token, use default user
		// In production, this should be required
		logger.Debug("No token provided, using default user",
			logger.ErrorField(err),
		)
		userID = "default"
	} else {
		// Validate token
		userID, err = authManager.ValidateToken(tokenString)
		if err != nil {
			logger.Warn("Invalid token, rejecting connection",
				logger.ErrorField(err),
			)
			http.Error(w, "Invalid authentication token", http.StatusUnauthorized)
			return
		}
	}

	// Upgrade connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("Failed to upgrade connection",
			logger.ErrorField(err),
		)
		return
	}

	// Create connection object
	connectionID := uuid.New().String()
	wsConn := wsgateway.NewConnection(connectionID, userID, conn)

	// Register connection with hub
	hub.Register(wsConn)

	logger.Info("WebSocket connection established",
		logger.String("connection_id", connectionID),
		logger.String("user_id", userID),
		logger.String("remote_addr", r.RemoteAddr),
	)
}
