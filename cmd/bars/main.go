package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/mohamedkhairy/stock-scanner/internal/bars"
	"github.com/mohamedkhairy/stock-scanner/internal/config"
	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/pubsub"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
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

	logger.Info("Starting bars aggregator service",
		logger.String("port", fmt.Sprintf("%d", cfg.Bars.Port)),
		logger.String("health_port", fmt.Sprintf("%d", cfg.Bars.HealthCheckPort)),
		logger.String("consumer_group", cfg.Bars.ConsumerGroup),
		logger.String("stream", cfg.Ingest.StreamName),
	)

	// Initialize Redis client
	redisClient, err := pubsub.NewRedisClient(cfg.Redis)
	if err != nil {
		logger.Fatal("Failed to initialize Redis client",
			logger.ErrorField(err),
		)
	}
	defer redisClient.Close()

	// Initialize TimescaleDB client
	writeConfig := storage.WriteConfigFromBarsConfig(cfg.Bars)
	dbClient, err := storage.NewTimescaleDBClient(cfg.Database, writeConfig)
	if err != nil {
		logger.Fatal("Failed to initialize TimescaleDB client",
			logger.ErrorField(err),
		)
	}
	defer dbClient.Close()

	// Start TimescaleDB write queue processor
	if err := dbClient.Start(); err != nil {
		logger.Fatal("Failed to start TimescaleDB client",
			logger.ErrorField(err),
		)
	}

	// Initialize bar aggregator
	aggregator := bars.NewAggregator()

	// Initialize bar publisher
	publisherConfig := bars.DefaultPublisherConfig()
	publisher := bars.NewPublisher(redisClient, publisherConfig)
	publisher.SetBarStorage(dbClient) // Wire TimescaleDB storage
	if err := publisher.Start(); err != nil {
		logger.Fatal("Failed to start bar publisher",
			logger.ErrorField(err),
		)
	}
	defer publisher.Stop()

	// Set up aggregator callbacks
	aggregator.SetOnBarFinal(func(bar *models.Bar1m) {
		// Publish finalized bar (to Redis Stream and TimescaleDB)
		if err := publisher.PublishFinalizedBar(bar); err != nil {
			logger.Error("Failed to publish finalized bar",
				logger.ErrorField(err),
				logger.String("symbol", bar.Symbol),
			)
		}
	})

	// Initialize stream consumer
	consumerConfig := pubsub.DefaultStreamConsumerConfig(
		cfg.Ingest.StreamName,
		cfg.Bars.ConsumerGroup,
		fmt.Sprintf("bars-consumer-%d", os.Getpid()),
	)
	// Use partitions from ingest config if available, otherwise default to 0
	consumerConfig.Partitions = 0 // Can be configured via environment variable if needed
	consumerConfig.BatchSize = cfg.Bars.BatchSize
	consumerConfig.ProcessTimeout = 5 * time.Second
	consumerConfig.AckTimeout = 10 * time.Second

	consumer := pubsub.NewStreamConsumer(redisClient, consumerConfig)
	consumer.SetAggregator(aggregator)

	// Start stream consumer
	if err := consumer.Start(); err != nil {
		logger.Fatal("Failed to start stream consumer",
			logger.ErrorField(err),
		)
	}
	defer consumer.Stop()

	logger.Info("Bars aggregator service started",
		logger.String("stream", cfg.Ingest.StreamName),
		logger.String("consumer_group", cfg.Bars.ConsumerGroup),
		logger.Int("partitions", consumerConfig.Partitions),
	)

	// Setup health and metrics server
	var wg sync.WaitGroup
	healthRouter := setupHealthAndMetricsServer(cfg, aggregator, consumer, publisher, dbClient)
	healthServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Bars.HealthCheckPort),
		Handler:      healthRouter,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("Starting health and metrics server",
			logger.Int("port", cfg.Bars.HealthCheckPort),
		)
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Health and metrics server failed",
				logger.ErrorField(err),
			)
		}
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	logger.Info("Shutting down bars aggregator service")

	// Shut down HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := healthServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Health server shutdown failed", logger.ErrorField(err))
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Finalize all remaining live bars
	finalizedBars := aggregator.FinalizeAllBars()
	if len(finalizedBars) > 0 {
		logger.Info("Finalizing remaining live bars on shutdown",
			logger.Int("count", len(finalizedBars)),
		)
		for _, bar := range finalizedBars {
			publisher.PublishFinalizedBar(bar)
		}
		// Give some time for final writes
		time.Sleep(500 * time.Millisecond)
	}

	logger.Info("Bars aggregator service stopped")
}

// setupHealthAndMetricsServer sets up HTTP endpoints for health checks and metrics
func setupHealthAndMetricsServer(
	cfg *config.Config,
	aggregator *bars.Aggregator,
	consumer *pubsub.StreamConsumer,
	publisher *bars.Publisher,
	dbClient *storage.TimescaleDBClient,
) *mux.Router {
	router := mux.NewRouter()

	// Health check endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		status := http.StatusOK
		healthStatus := map[string]interface{}{
			"status":    "UP",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"checks": map[string]interface{}{
				"consumer": map[string]interface{}{
					"status":  "ok",
					"running": consumer.IsRunning(),
					"stats":   consumer.GetStats(),
				},
				"aggregator": map[string]interface{}{
					"status":       "ok",
					"symbol_count": aggregator.GetSymbolCount(),
				},
				"publisher": map[string]interface{}{
					"status":  "ok",
					"running": publisher.IsRunning(),
				},
				"database": map[string]interface{}{
					"status":  "ok",
					"running": dbClient.IsRunning(),
				},
			},
		}

		// Check if any component is not running
		if !consumer.IsRunning() || !publisher.IsRunning() || !dbClient.IsRunning() {
			status = http.StatusServiceUnavailable
			healthStatus["status"] = "DOWN"
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(healthStatus)
	}).Methods("GET")

	// Readiness probe
	router.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if consumer.IsRunning() && publisher.IsRunning() && dbClient.IsRunning() {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("READY"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("NOT READY"))
		}
	}).Methods("GET")

	// Liveness probe
	router.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("LIVE"))
	}).Methods("GET")

	// Metrics endpoint
	router.Handle("/metrics", promhttp.Handler())

	return router
}
