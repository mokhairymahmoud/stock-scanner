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
	"github.com/mohamedkhairy/stock-scanner/internal/api"
	"github.com/mohamedkhairy/stock-scanner/internal/config"
	"github.com/mohamedkhairy/stock-scanner/internal/pubsub"
	"github.com/mohamedkhairy/stock-scanner/internal/rules"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
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

	logger.Info("Starting REST API service",
		logger.String("port", fmt.Sprintf("%d", cfg.API.Port)),
		logger.String("health_port", fmt.Sprintf("%d", cfg.API.HealthCheckPort)),
		logger.Int("rate_limit_rps", cfg.API.RateLimitRPS),
	)

	// Initialize Redis client (for rule store if using Redis)
	redisClient, err := pubsub.NewRedisClient(cfg.Redis)
	if err != nil {
		logger.Fatal("Failed to initialize Redis client",
			logger.ErrorField(err),
		)
	}
	defer redisClient.Close()

	// Initialize rule store (use database store for persistence)
	ruleStore, err := rules.NewDatabaseRuleStore(cfg.Database)
	if err != nil {
		logger.Fatal("Failed to initialize rule store",
			logger.ErrorField(err),
		)
	}
	defer ruleStore.Close()

	// Initialize Redis rule store for caching
	redisStoreConfig := rules.DefaultRedisRuleStoreConfig()
	redisRuleStore, err := rules.NewRedisRuleStore(redisClient, redisStoreConfig)
	if err != nil {
		logger.Fatal("Failed to initialize Redis rule store",
			logger.ErrorField(err),
		)
	}

	// Initialize rule sync service
	syncService := rules.NewRuleSyncService(ruleStore, redisRuleStore, redisClient)

	// Sync rules from database to Redis on startup
	logger.Info("Syncing rules from database to Redis...")
	if err := syncService.SyncAllRules(); err != nil {
		logger.Warn("Failed to sync rules on startup",
			logger.ErrorField(err),
		)
		// Don't fail startup if sync fails
	}

	// Initialize rule compiler
	metricResolver := rules.NewMetricResolver()
	compiler := rules.NewCompiler(metricResolver)

	// Initialize alert storage
	alertStorage, err := storage.NewTimescaleAlertStorage(cfg.Database)
	if err != nil {
		logger.Fatal("Failed to initialize alert storage",
			logger.ErrorField(err),
		)
	}
	defer alertStorage.Close()

	// Initialize toplist store
	toplistStore, err := toplist.NewDatabaseToplistStore(cfg.Database)
	if err != nil {
		logger.Fatal("Failed to initialize toplist store",
			logger.ErrorField(err),
		)
	}
	defer toplistStore.Close()

	// Initialize toplist service
	toplistUpdater := toplist.NewRedisToplistUpdater(redisClient)
	toplistService := toplist.NewToplistService(toplistStore, redisClient, toplistUpdater)

	// Initialize handlers
	ruleHandler := api.NewRuleHandler(ruleStore, compiler, syncService)
	alertHandler := api.NewAlertHandler(alertStorage)
	symbolHandler := api.NewSymbolHandler(cfg.MarketData.Symbols)
	userHandler := api.NewUserHandler()
	toplistHandler := api.NewToplistHandler(toplistService, toplistStore)

	// Set up router
	router := mux.NewRouter()

	// API v1 routes
	v1 := router.PathPrefix("/api/v1").Subrouter()

	// Rule management endpoints
	v1.HandleFunc("/rules", ruleHandler.ListRules).Methods("GET")
	v1.HandleFunc("/rules", ruleHandler.CreateRule).Methods("POST")
	v1.HandleFunc("/rules/{id}", ruleHandler.GetRule).Methods("GET")
	v1.HandleFunc("/rules/{id}", ruleHandler.UpdateRule).Methods("PUT")
	v1.HandleFunc("/rules/{id}", ruleHandler.DeleteRule).Methods("DELETE")
	v1.HandleFunc("/rules/{id}/validate", ruleHandler.ValidateRule).Methods("POST")

	// Alert history endpoints
	v1.HandleFunc("/alerts", alertHandler.ListAlerts).Methods("GET")
	v1.HandleFunc("/alerts/{id}", alertHandler.GetAlert).Methods("GET")

	// Symbol management endpoints
	v1.HandleFunc("/symbols", symbolHandler.ListSymbols).Methods("GET")
	v1.HandleFunc("/symbols/{symbol}", symbolHandler.GetSymbol).Methods("GET")

	// User management endpoints
	v1.HandleFunc("/user/profile", userHandler.GetProfile).Methods("GET")
	v1.HandleFunc("/user/profile", userHandler.UpdateProfile).Methods("PUT")

	// Toplist endpoints
	v1.HandleFunc("/toplists", toplistHandler.ListToplists).Methods("GET")
	v1.HandleFunc("/toplists/system/{type}", toplistHandler.GetSystemToplist).Methods("GET")
	v1.HandleFunc("/toplists/user", toplistHandler.ListUserToplists).Methods("GET")
	v1.HandleFunc("/toplists/user", toplistHandler.CreateUserToplist).Methods("POST")
	v1.HandleFunc("/toplists/user/{id}", toplistHandler.GetUserToplist).Methods("GET")
	v1.HandleFunc("/toplists/user/{id}", toplistHandler.UpdateUserToplist).Methods("PUT")
	v1.HandleFunc("/toplists/user/{id}", toplistHandler.DeleteUserToplist).Methods("DELETE")
	v1.HandleFunc("/toplists/user/{id}/rankings", toplistHandler.GetToplistRankings).Methods("GET")

	// Health check endpoints
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	router.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		// Try to query rules table to check database connectivity
		_, err := ruleStore.GetAllRules()
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "not ready"})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	})

	router.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
	})

	// Metrics endpoint
	router.Handle("/metrics", promhttp.Handler())

	// Apply middleware
	middlewares := api.ChainMiddleware(
		api.CORSMiddleware(),
		api.LoggingMiddleware(),
		api.ErrorHandlingMiddleware(),
		api.AuthMiddleware(cfg.API.JWTSecret),
		api.RateLimitMiddleware(cfg.API.RateLimitRPS),
	)

	handler := middlewares(router)

	// Start HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.API.Port),
		Handler: handler,
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
	logger.Info("Shutting down REST API service")

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Error shutting down HTTP server",
			logger.ErrorField(err),
		)
	}

	logger.Info("REST API service stopped")
}
