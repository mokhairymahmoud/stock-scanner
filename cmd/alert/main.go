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

	"github.com/gorilla/mux"
	"github.com/mohamedkhairy/stock-scanner/internal/alert"
	"github.com/mohamedkhairy/stock-scanner/internal/config"
	"github.com/mohamedkhairy/stock-scanner/internal/pubsub"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

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

	logger.Info("Starting alert service",
		logger.String("port", fmt.Sprintf("%d", cfg.Alert.Port)),
		logger.String("health_port", fmt.Sprintf("%d", cfg.Alert.HealthCheckPort)),
		logger.String("stream", cfg.Alert.StreamName),
		logger.String("consumer_group", cfg.Alert.ConsumerGroup),
	)

	// Initialize Redis client
	redisClient, err := pubsub.NewRedisClient(cfg.Redis)
	if err != nil {
		logger.Fatal("Failed to initialize Redis client",
			logger.ErrorField(err),
		)
	}
	defer redisClient.Close()

	// Initialize alert service components
	deduplicator := alert.NewDeduplicator(redisClient, cfg.Alert.DedupeTTL)
	filter := alert.NewUserFilter()
	cooldown := alert.NewCooldownManager(redisClient, cfg.Alert.CooldownTTL)

	// Initialize alert persister
	writeConfig := alert.WriteConfig{
		BatchSize:  cfg.Alert.DBWriteBatchSize,
		Interval:   cfg.Alert.DBWriteInterval,
		QueueSize:  cfg.Alert.DBWriteQueueSize,
		MaxRetries: cfg.Alert.DBMaxRetries,
		RetryDelay: cfg.Alert.DBRetryDelay,
	}
	persister, err := alert.NewAlertPersister(cfg.Database, writeConfig)
	if err != nil {
		logger.Fatal("Failed to initialize alert persister",
			logger.ErrorField(err),
		)
	}
	defer persister.Close()

	// Start persister
	if err := persister.Start(); err != nil {
		logger.Fatal("Failed to start alert persister",
			logger.ErrorField(err),
		)
	}

	// Initialize router
	router := alert.NewRouter(redisClient, cfg.Alert.FilteredStreamName, 5*time.Second)

	// Initialize consumer
	consumer := alert.NewConsumer(
		cfg.Alert,
		redisClient,
		deduplicator,
		filter,
		cooldown,
		persister,
		router,
	)

	// Start consumer
	if err := consumer.Start(); err != nil {
		logger.Fatal("Failed to start alert consumer",
			logger.ErrorField(err),
		)
	}
	defer consumer.Stop()

	// Set up HTTP server for health checks and metrics
	routerMux := mux.NewRouter()

	// Health check endpoints
	routerMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	routerMux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		// Check if consumer is running
		stats := consumer.GetStats()
		if stats.AlertsReceived > 0 || time.Since(stats.LastAlertTime) < 5*time.Minute {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "not ready"})
		}
	})

	routerMux.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
	})

	// Stats endpoint
	routerMux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		stats := consumer.GetStats()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	})

	// Metrics endpoint
	routerMux.Handle("/metrics", promhttp.Handler())

	// Start HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Alert.HealthCheckPort),
		Handler: routerMux,
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
	logger.Info("Shutting down alert service")

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Error shutting down HTTP server",
			logger.ErrorField(err),
		)
	}

	logger.Info("Alert service stopped")
}

