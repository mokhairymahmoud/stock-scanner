package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/mohamedkhairy/stock-scanner/internal/config"
	"github.com/mohamedkhairy/stock-scanner/internal/pubsub"
	"github.com/mohamedkhairy/stock-scanner/internal/rules"
	"github.com/mohamedkhairy/stock-scanner/internal/scanner"
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

	logger.Info("Starting scanner worker service",
		logger.String("port", fmt.Sprintf("%d", cfg.Scanner.Port)),
		logger.String("health_port", fmt.Sprintf("%d", cfg.Scanner.HealthCheckPort)),
		logger.String("worker_id", cfg.Scanner.WorkerID),
		logger.Int("worker_count", cfg.Scanner.WorkerCount),
		logger.Duration("scan_interval", cfg.Scanner.ScanInterval),
	)

	// Initialize Redis client
	redisClient, err := pubsub.NewRedisClient(cfg.Redis)
	if err != nil {
		logger.Fatal("Failed to initialize Redis client",
			logger.ErrorField(err),
		)
	}
	defer redisClient.Close()

	// Initialize TimescaleDB client (for rehydration)
	writeConfig := storage.WriteConfig{
		BatchSize:  100,
		Interval:   5 * time.Second,
		QueueSize:  1000,
		MaxRetries: 3,
		RetryDelay: 1 * time.Second,
	}
	dbClient, err := storage.NewTimescaleDBClient(cfg.Database, writeConfig)
	if err != nil {
		logger.Fatal("Failed to initialize TimescaleDB client",
			logger.ErrorField(err),
		)
	}
	defer dbClient.Close()

	// Parse worker ID (assume format "worker-1" -> 1)
	workerID := parseWorkerID(cfg.Scanner.WorkerID)
	if workerID < 0 || workerID >= cfg.Scanner.WorkerCount {
		logger.Fatal("Invalid worker ID",
			logger.String("worker_id", cfg.Scanner.WorkerID),
			logger.Int("worker_count", cfg.Scanner.WorkerCount),
		)
	}

	// Initialize partition manager
	partitionManager, err := scanner.NewPartitionManager(workerID, cfg.Scanner.WorkerCount)
	if err != nil {
		logger.Fatal("Failed to create partition manager",
			logger.ErrorField(err),
		)
	}

	// Initialize state manager
	stateManager := scanner.NewStateManager(200) // Keep last 200 finalized bars

	// Initialize rule store (memory or Redis based on config)
	var ruleStore rules.RuleStore
	if cfg.Scanner.RuleStoreType == "redis" {
		redisStoreConfig := rules.DefaultRedisRuleStoreConfig()
		redisStore, err := rules.NewRedisRuleStore(redisClient, redisStoreConfig)
		if err != nil {
			logger.Fatal("Failed to create Redis rule store",
				logger.ErrorField(err),
			)
		}
		ruleStore = redisStore
		logger.Info("Using Redis rule store",
			logger.String("key_prefix", redisStoreConfig.KeyPrefix),
		)
	} else {
		ruleStore = rules.NewInMemoryRuleStore()
		logger.Info("Using in-memory rule store")
	}

	// Initialize rule compiler
	compiler := rules.NewCompiler(nil)

	// Initialize cooldown tracker
	cooldownTracker := scanner.NewCooldownTracker(5 * time.Minute)
	if err := cooldownTracker.Start(); err != nil {
		logger.Fatal("Failed to start cooldown tracker",
			logger.ErrorField(err),
		)
	}
	defer cooldownTracker.Stop()

	// Initialize alert emitter
	alertEmitterConfig := scanner.DefaultAlertEmitterConfig()
	alertEmitter := scanner.NewAlertEmitter(redisClient, alertEmitterConfig)

	// Initialize scan loop
	scanLoopConfig := scanner.DefaultScanLoopConfig()
	scanLoopConfig.ScanInterval = cfg.Scanner.ScanInterval
	scanLoop := scanner.NewScanLoop(
		scanLoopConfig,
		stateManager,
		ruleStore,
		compiler,
		cooldownTracker,
		alertEmitter,
	)

	// Initialize rehydrator
	rehydratorConfig := scanner.DefaultRehydrationConfig()
	rehydratorConfig.Symbols = cfg.Scanner.SymbolUniverse
	rehydrator := scanner.NewRehydrator(rehydratorConfig, stateManager, dbClient, redisClient)

	// Rehydrate state on startup
	logger.Info("Rehydrating state on startup...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := rehydrator.RehydrateState(ctx); err != nil {
		logger.Error("Failed to rehydrate state (continuing anyway)",
			logger.ErrorField(err),
		)
	} else {
		logger.Info("State rehydration complete",
			logger.Int("symbol_count", stateManager.GetSymbolCount()),
		)
	}

	// Load initial rules (for now, empty - rules will be added via API later)
	// TODO: Load rules from database or config file
	logger.Info("No initial rules loaded (rules will be added via API)")

	// Initialize tick consumer
	tickConsumerConfig := pubsub.DefaultStreamConsumerConfig(
		cfg.Ingest.StreamName,
		"scanner-group",
		fmt.Sprintf("scanner-%s", cfg.Scanner.WorkerID),
	)
	tickConsumerConfig.Partitions = 0 // Will be configured based on partitioning
	tickConsumerConfig.BatchSize = cfg.Scanner.BufferSize
	tickConsumerConfig.ProcessTimeout = 5 * time.Second
	tickConsumerConfig.AckTimeout = 10 * time.Second

	tickConsumer := scanner.NewTickConsumer(redisClient, tickConsumerConfig, stateManager)

	// Initialize indicator consumer
	indicatorConsumerConfig := scanner.DefaultIndicatorConsumerConfig()
	indicatorConsumer := scanner.NewIndicatorConsumer(redisClient, indicatorConsumerConfig, stateManager)

	// Initialize bar finalization handler
	barHandlerConfig := pubsub.DefaultStreamConsumerConfig(
		"bars.finalized",
		"scanner-group",
		fmt.Sprintf("scanner-%s", cfg.Scanner.WorkerID),
	)
	barHandlerConfig.Partitions = 0
	barHandlerConfig.BatchSize = cfg.Scanner.BufferSize
	barHandlerConfig.ProcessTimeout = 5 * time.Second
	barHandlerConfig.AckTimeout = 10 * time.Second

	barHandler := scanner.NewBarFinalizationHandler(redisClient, barHandlerConfig, stateManager)

	// Start all consumers
	logger.Info("Starting consumers...")

	if err := tickConsumer.Start(); err != nil {
		logger.Fatal("Failed to start tick consumer",
			logger.ErrorField(err),
		)
	}
	defer tickConsumer.Stop()

	if err := indicatorConsumer.Start(); err != nil {
		logger.Fatal("Failed to start indicator consumer",
			logger.ErrorField(err),
		)
	}
	defer indicatorConsumer.Stop()

	if err := barHandler.Start(); err != nil {
		logger.Fatal("Failed to start bar finalization handler",
			logger.ErrorField(err),
		)
	}
	defer barHandler.Stop()

	// Start scan loop
	logger.Info("Starting scan loop...")
	if err := scanLoop.Start(); err != nil {
		logger.Fatal("Failed to start scan loop",
			logger.ErrorField(err),
		)
	}
	defer scanLoop.Stop()

	logger.Info("Scanner worker service started",
		logger.String("worker_id", cfg.Scanner.WorkerID),
		logger.Int("worker_count", cfg.Scanner.WorkerCount),
		logger.Int("symbol_count", stateManager.GetSymbolCount()),
	)

	// Setup health and metrics server
	var wg sync.WaitGroup
	healthRouter := setupHealthAndMetricsServer(
		cfg,
		stateManager,
		scanLoop,
		tickConsumer,
		indicatorConsumer,
		barHandler,
		cooldownTracker,
		alertEmitter,
		partitionManager,
	)
	healthServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Scanner.HealthCheckPort),
		Handler:      healthRouter,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("Starting health and metrics server",
			logger.Int("port", cfg.Scanner.HealthCheckPort),
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
	logger.Info("Shutting down scanner worker service")

	// Shut down HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := healthServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Health server shutdown failed", logger.ErrorField(err))
	}

	// Wait for all goroutines to finish
	wg.Wait()

	logger.Info("Scanner worker service stopped")
}

// parseWorkerID parses worker ID from string format "worker-1" -> 1
func parseWorkerID(workerIDStr string) int {
	// Try to extract number from "worker-1" format
	if len(workerIDStr) > 7 && workerIDStr[:7] == "worker-" {
		id, err := strconv.Atoi(workerIDStr[7:])
		if err == nil {
			return id
		}
	}

	// Try direct integer parse
	id, err := strconv.Atoi(workerIDStr)
	if err == nil {
		return id
	}

	return -1
}

// setupHealthAndMetricsServer sets up HTTP endpoints for health checks and metrics
func setupHealthAndMetricsServer(
	cfg *config.Config,
	stateManager *scanner.StateManager,
	scanLoop *scanner.ScanLoop,
	tickConsumer *scanner.TickConsumer,
	indicatorConsumer *scanner.IndicatorConsumer,
	barHandler *scanner.BarFinalizationHandler,
	cooldownTracker *scanner.InMemoryCooldownTracker,
	alertEmitter *scanner.AlertEmitterImpl,
	partitionManager *scanner.PartitionManager,
) *mux.Router {
	router := mux.NewRouter()

	// Health check endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		status := http.StatusOK
		healthStatus := map[string]interface{}{
			"status":    "UP",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"worker": map[string]interface{}{
				"id":    cfg.Scanner.WorkerID,
				"count": cfg.Scanner.WorkerCount,
			},
			"checks": map[string]interface{}{
				"state_manager": map[string]interface{}{
					"status":       "ok",
					"symbol_count": stateManager.GetSymbolCount(),
				},
				"scan_loop": map[string]interface{}{
					"status":  "ok",
					"running": scanLoop.IsRunning(),
					"stats":   scanLoop.GetStats(),
				},
				"tick_consumer": map[string]interface{}{
					"status":  "ok",
					"running": tickConsumer.IsRunning(),
					"stats":   tickConsumer.GetStats(),
				},
				"indicator_consumer": map[string]interface{}{
					"status":  "ok",
					"running": indicatorConsumer.IsRunning(),
					"stats":   indicatorConsumer.GetStats(),
				},
				"bar_handler": map[string]interface{}{
					"status":  "ok",
					"running": barHandler.IsRunning(),
					"stats":   barHandler.GetStats(),
				},
				"cooldown_tracker": map[string]interface{}{
					"status":        "ok",
					"cooldown_count": cooldownTracker.GetCooldownCount(),
				},
				"alert_emitter": map[string]interface{}{
					"status": "ok",
					"stats":  alertEmitter.GetStats(),
				},
				"partition_manager": map[string]interface{}{
					"status":         "ok",
					"worker_id":      partitionManager.GetWorkerID(),
					"total_workers":  partitionManager.GetTotalWorkers(),
					"assigned_count": partitionManager.GetAssignedSymbolCount(),
				},
			},
		}

		// Check if any critical component is not running
		if !scanLoop.IsRunning() || !tickConsumer.IsRunning() || !indicatorConsumer.IsRunning() || !barHandler.IsRunning() {
			status = http.StatusServiceUnavailable
			healthStatus["status"] = "DOWN"
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(healthStatus)
	}).Methods("GET")

	// Readiness probe
	router.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		// Service is ready if scan loop and all consumers are running
		if scanLoop.IsRunning() &&
			tickConsumer.IsRunning() &&
			indicatorConsumer.IsRunning() &&
			barHandler.IsRunning() {
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

	// Stats endpoint (detailed statistics)
	router.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		stats := map[string]interface{}{
			"state_manager": map[string]interface{}{
				"symbol_count": stateManager.GetSymbolCount(),
			},
			"scan_loop": scanLoop.GetStats(),
			"tick_consumer": tickConsumer.GetStats(),
			"indicator_consumer": indicatorConsumer.GetStats(),
			"bar_handler": barHandler.GetStats(),
			"cooldown_tracker": map[string]interface{}{
				"cooldown_count": cooldownTracker.GetCooldownCount(),
			},
			"alert_emitter": alertEmitter.GetStats(),
			"partition_manager": map[string]interface{}{
				"worker_id":      partitionManager.GetWorkerID(),
				"total_workers":  partitionManager.GetTotalWorkers(),
				"assigned_count": partitionManager.GetAssignedSymbolCount(),
				"assigned_symbols": partitionManager.GetAssignedSymbols(),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	}).Methods("GET")

	return router
}
