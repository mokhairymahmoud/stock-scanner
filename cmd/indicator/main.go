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
	"github.com/mohamedkhairy/stock-scanner/internal/config"
	"github.com/mohamedkhairy/stock-scanner/internal/indicator"
	"github.com/mohamedkhairy/stock-scanner/internal/pubsub"
	"github.com/mohamedkhairy/stock-scanner/internal/toplist"
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

	logger.Info("Starting indicator engine service",
		logger.String("port", fmt.Sprintf("%d", cfg.Indicator.Port)),
		logger.String("health_port", fmt.Sprintf("%d", cfg.Indicator.HealthCheckPort)),
		logger.String("consumer_group", cfg.Indicator.ConsumerGroup),
	)

	// Initialize Redis client
	redisClient, err := pubsub.NewRedisClient(cfg.Redis)
	if err != nil {
		logger.Fatal("Failed to initialize Redis client",
			logger.ErrorField(err),
		)
	}
	defer redisClient.Close()

	// Initialize indicator registry
	indicatorRegistry := indicator.NewIndicatorRegistry()
	if err := indicator.RegisterAllIndicators(indicatorRegistry); err != nil {
		logger.Fatal("Failed to register indicators",
			logger.ErrorField(err),
		)
	}

	logger.Info("Registered indicators",
		logger.Int("count", len(indicatorRegistry.ListAvailable())),
	)

	// Initialize indicator engine
	engineConfig := indicator.DefaultEngineConfig()
	engine := indicator.NewEngine(engineConfig, indicatorRegistry)

	// Initialize indicator publisher
	publisherConfig := indicator.DefaultPublisherConfig()
	publisher := indicator.NewPublisher(redisClient, publisherConfig)

	// Initialize toplist store and updater
	// Note: Toplist store is optional - if database is unavailable, we'll continue without toplist updates
	var toplistStore toplist.ToplistStore
	toplistStore, err = toplist.NewDatabaseToplistStore(cfg.Database)
	if err != nil {
		logger.Warn("Failed to initialize toplist store, toplist updates will be disabled",
			logger.ErrorField(err),
		)
		toplistStore = nil
	} else {
		defer toplistStore.Close()
	}

	toplistUpdater := toplist.NewRedisToplistUpdater(redisClient)
	publisher.SetToplistUpdater(toplistUpdater, toplistStore != nil)
	if toplistStore != nil {
		publisher.SetToplistStore(toplistStore)
		logger.Info("Toplist integration enabled for indicator engine")
	} else {
		logger.Info("Toplist integration disabled (database unavailable)")
	}

	if err := publisher.Start(); err != nil {
		logger.Fatal("Failed to start indicator publisher",
			logger.ErrorField(err),
		)
	}
	defer publisher.Stop()

	// Set up engine to publish indicators after processing bars
	engine.SetOnIndicatorsUpdated(func(symbol string, indicators map[string]float64) {
		if err := publisher.PublishIndicators(symbol, indicators); err != nil {
			logger.Error("Failed to publish indicators",
				logger.ErrorField(err),
				logger.String("symbol", symbol),
			)
		}
	})

	// Initialize bar consumer
	consumerConfig := pubsub.DefaultStreamConsumerConfig(
		"bars.finalized", // Stream name for finalized bars
		cfg.Indicator.ConsumerGroup,
		fmt.Sprintf("indicator-consumer-%d", os.Getpid()),
	)
	consumerConfig.Partitions = 0 // No partitioning for now
	consumerConfig.BatchSize = 100
	consumerConfig.ProcessTimeout = 5 * time.Second
	consumerConfig.AckTimeout = 10 * time.Second

	barConsumer := indicator.NewBarConsumer(redisClient, consumerConfig)
	barConsumer.SetProcessor(engine)

	// Start bar consumer
	if err := barConsumer.Start(); err != nil {
		logger.Fatal("Failed to start bar consumer",
			logger.ErrorField(err),
		)
	}
	defer barConsumer.Stop()

	logger.Info("Indicator engine service started",
		logger.String("stream", "bars.finalized"),
		logger.String("consumer_group", cfg.Indicator.ConsumerGroup),
	)

	// Setup health and metrics server
	var wg sync.WaitGroup
	healthRouter := setupHealthAndMetricsServer(cfg, engine, barConsumer, publisher)
	healthServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Indicator.HealthCheckPort),
		Handler:      healthRouter,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("Starting health and metrics server",
			logger.Int("port", cfg.Indicator.HealthCheckPort),
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
	logger.Info("Shutting down indicator engine service")

	// Shut down HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := healthServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Health server shutdown failed", logger.ErrorField(err))
	}

	// Wait for all goroutines to finish
	wg.Wait()

	logger.Info("Indicator engine service stopped")
}


// setupHealthAndMetricsServer sets up HTTP endpoints for health checks and metrics
func setupHealthAndMetricsServer(
	cfg *config.Config,
	engine *indicator.Engine,
	consumer *indicator.BarConsumer,
	publisher *indicator.Publisher,
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
				"engine": map[string]interface{}{
					"status":       "ok",
					"symbol_count": engine.GetSymbolCount(),
				},
				"publisher": map[string]interface{}{
					"status":  "ok",
					"running": publisher.IsRunning(),
				},
			},
		}

		// Check if any component is not running
		if !consumer.IsRunning() || !publisher.IsRunning() {
			status = http.StatusServiceUnavailable
			healthStatus["status"] = "DOWN"
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(healthStatus)
	}).Methods("GET")

	// Readiness probe
	router.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if consumer.IsRunning() && publisher.IsRunning() {
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
