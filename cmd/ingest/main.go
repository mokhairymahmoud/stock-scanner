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
	"github.com/mohamedkhairy/stock-scanner/internal/data"
	"github.com/mohamedkhairy/stock-scanner/internal/models"
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

	logger.Info("Starting ingest service",
		logger.String("port", fmt.Sprintf("%d", cfg.Ingest.Port)),
		logger.String("health_port", fmt.Sprintf("%d", cfg.Ingest.HealthCheckPort)),
		logger.String("stream", cfg.Ingest.StreamName),
		logger.String("provider", cfg.MarketData.Provider),
	)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Redis client
	redisClient, err := pubsub.NewRedisClient(cfg.Redis)
	if err != nil {
		logger.Fatal("Failed to initialize Redis client",
			logger.ErrorField(err),
		)
	}
	defer redisClient.Close()

	// Initialize stream publisher
	publisherConfig := pubsub.DefaultStreamPublisherConfig(cfg.Ingest.StreamName)
	publisherConfig.BatchSize = cfg.Ingest.BatchSize
	publisherConfig.BatchTimeout = cfg.Ingest.BatchTimeout
	publisherConfig.Partitions = 0 // Can be configured later if needed

	streamPublisher := pubsub.NewStreamPublisher(redisClient, publisherConfig)
	streamPublisher.Start()
	defer streamPublisher.Close()

	// Initialize normalizer
	normalizer := data.NewNormalizer(cfg.MarketData.Provider)

	// Initialize provider factory
	providerFactory := data.NewProviderFactory()

	// Create provider
	providerConfig := data.ProviderConfig{
		APIKey:    cfg.MarketData.APIKey,
		APISecret: cfg.MarketData.APISecret,
		BaseURL:   cfg.MarketData.BaseURL,
		WSURL:     cfg.MarketData.WebSocketURL,
	}

	provider, err := providerFactory.CreateProvider(cfg.MarketData.Provider, providerConfig)
	if err != nil {
		logger.Fatal("Failed to create provider",
			logger.ErrorField(err),
			logger.String("provider", cfg.MarketData.Provider),
		)
	}
	defer provider.Close()

	// Connect to provider
	if err := provider.Connect(ctx); err != nil {
		logger.Fatal("Failed to connect to provider",
			logger.ErrorField(err),
		)
	}

	// Subscribe to symbols
	tickChan, err := provider.Subscribe(ctx, cfg.MarketData.Symbols)
	if err != nil {
		logger.Fatal("Failed to subscribe to symbols",
			logger.ErrorField(err),
		)
	}

	logger.Info("Subscribed to symbols",
		logger.Int("count", len(cfg.MarketData.Symbols)),
		logger.String("symbols", fmt.Sprintf("%v", cfg.MarketData.Symbols)),
	)

	// Start ingestion loop
	var wg sync.WaitGroup
	wg.Add(1)
	go ingestLoop(ctx, &wg, tickChan, normalizer, streamPublisher)

	// Start HTTP server for health checks and metrics
	healthServer := startHealthServer(cfg.Ingest.HealthCheckPort, provider, streamPublisher)
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		healthServer.Shutdown(shutdownCtx)
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	logger.Info("Shutting down ingest service")

	// Cancel context to stop ingestion loop
	cancel()

	// Wait for ingestion loop to finish
	wg.Wait()

	logger.Info("Ingest service stopped")
}

// ingestLoop processes ticks from the provider and publishes them to Redis streams
func ingestLoop(
	ctx context.Context,
	wg *sync.WaitGroup,
	tickChan <-chan *models.Tick,
	normalizer data.Normalizer,
	publisher *pubsub.StreamPublisher,
) {
	defer wg.Done()

	tickCount := 0
	errorCount := 0

	for {
		select {
		case <-ctx.Done():
			logger.Info("Ingestion loop stopped",
				logger.Int("ticks_processed", tickCount),
				logger.Int("errors", errorCount),
			)
			return

		case tick, ok := <-tickChan:
			if !ok {
				logger.Warn("Tick channel closed")
				return
			}

			if tick == nil {
				continue
			}

			// Publish tick directly (already normalized by provider)
			// If provider returns raw messages, we'd normalize here
			if err := publisher.Publish(tick); err != nil {
				errorCount++
				logger.Error("Failed to publish tick",
					logger.ErrorField(err),
					logger.String("symbol", tick.Symbol),
				)
				continue
			}

			tickCount++
			if tickCount%1000 == 0 {
				logger.Debug("Processed ticks",
					logger.Int("count", tickCount),
					logger.Int("errors", errorCount),
				)
			}
		}
	}
}

// startHealthServer starts the HTTP server for health checks and metrics
func startHealthServer(port int, provider data.Provider, publisher *pubsub.StreamPublisher) *http.Server {
	router := mux.NewRouter()

	// Health check endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
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
					"status":     "ok",
					"batch_size": publisher.GetBatchSize(),
				},
			},
		}

		// Determine overall status
		if !provider.IsConnected() {
			health["status"] = "degraded"
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(health)
	}).Methods("GET")

	// Readiness probe
	router.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if provider.IsConnected() {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ready"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("not ready"))
		}
	}).Methods("GET")

	// Liveness probe
	router.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("alive"))
	}).Methods("GET")

	// Metrics endpoint
	router.Handle("/metrics", promhttp.Handler()).Methods("GET")

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		logger.Info("Starting health check server",
			logger.Int("port", port),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Health check server failed",
				logger.ErrorField(err),
			)
		}
	}()

	return server
}
