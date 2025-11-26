.PHONY: help build test test-performance test-worker-scaling test-coverage clean clean-db docker-up docker-up-all docker-down docker-logs docker-logs-service docker-build docker-restart docker-test docker-deploy docker-verify e2e-test validate-phase2 migrate-up fmt lint run-ingest run-bars run-indicator run-scanner run-alert run-ws-gateway run-api deps

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build all services
	@echo "Building all services..."
	@go build -o bin/ingest ./cmd/ingest
	@go build -o bin/bars ./cmd/bars
	@go build -o bin/indicator ./cmd/indicator
	@go build -o bin/scanner ./cmd/scanner
	@go build -o bin/alert ./cmd/alert
	@go build -o bin/ws-gateway ./cmd/ws_gateway
	@go build -o bin/api ./cmd/api

test: ## Run all tests
	@echo "Running tests..."
	@go test -v ./...

test-performance: ## Run performance tests (with extended timeout)
	@echo "Running performance tests..."
	@go test -timeout 10m -v ./tests/performance

test-worker-scaling: ## Run worker scaling tests (with extended timeout for large symbol counts)
	@echo "Running worker scaling tests..."
	@go test -timeout 10m -v ./tests/performance -run TestWorkerScaling

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html

clean-db: ## Clean database
	@echo "Cleaning database..."
	@docker-compose -f config/docker-compose.yaml down -v
	@docker volume rm stock-scanner-timescaledb-data stock-scanner-redis-data stock-scanner-prometheus-data stock-scanner-grafana-data stock-scanner-redisinsight-data stock-scanner-loki-data stock-scanner-promtail-data stock-scanner-jaeger-data 2>/dev/null || true

docker-up: ## Start Docker Compose services (infrastructure only)
	@echo "Starting Docker Compose services (infrastructure)..."
	@docker-compose -f config/docker-compose.yaml up -d redis timescaledb prometheus grafana redisinsight loki promtail jaeger

docker-up-all: ## Start all Docker Compose services (including Go services)
	@echo "Starting all Docker Compose services..."
	@docker-compose -f config/docker-compose.yaml up -d --build

docker-down: ## Stop Docker Compose services
	@echo "Stopping Docker Compose services..."
	@docker-compose -f config/docker-compose.yaml down

docker-logs: ## View Docker Compose logs
	@docker-compose -f config/docker-compose.yaml logs -f

docker-logs-service: ## View logs for a specific service (usage: make docker-logs-service SERVICE=ingest)
	@docker-compose -f config/docker-compose.yaml logs -f $(SERVICE)

docker-build: ## Build Docker images for all services
	@echo "Building Docker images..."
	@docker-compose -f config/docker-compose.yaml build

docker-restart: ## Restart all services
	@docker-compose -f config/docker-compose.yaml restart

docker-test: ## Test all services
	@echo "Testing all services..."
	@if [ -f ./scripts/test_services.sh ]; then \
		./scripts/test_services.sh; \
	else \
		echo "⚠️  test_services.sh not found. Skipping..."; \
	fi

docker-deploy: ## Deploy all services
	@echo "Deploying all services..."
	@./scripts/deploy.sh

docker-verify: ## Verify deployment
	@echo "Verifying deployment..."
	@./scripts/verify_deployment.sh

e2e-test: ## Interactive E2E testing helper
	@./scripts/e2e_test.sh

validate-phase2: ## Validate Phase 2 (Indicator Engine) implementation
	@echo "Validating Phase 2: Indicator Engine..."
	@if [ -f ./scripts/validate_phase2.sh ]; then \
		./scripts/validate_phase2.sh; \
	else \
		echo "⚠️  validate_phase2.sh not found. Skipping..."; \
	fi

migrate-up: ## Run database migrations (uses Docker)
	@echo "Running migrations..."
	@docker exec -i stock-scanner-timescaledb psql -U postgres -d stock_scanner < scripts/migrations/001_create_bars_table.sql || \
		(echo "⚠️  TimescaleDB container not running. Starting infrastructure..." && \
		 docker-compose -f config/docker-compose.yaml up -d timescaledb && \
		 echo "Waiting for database to be ready..." && \
		 sleep 10 && \
		 docker exec -i stock-scanner-timescaledb psql -U postgres -d stock_scanner < scripts/migrations/001_create_bars_table.sql)
	@docker exec -i stock-scanner-timescaledb psql -U postgres -d stock_scanner < scripts/migrations/002_create_alert_history_table.sql || \
		(echo "⚠️  TimescaleDB container not running. Starting infrastructure..." && \
		 docker-compose -f config/docker-compose.yaml up -d timescaledb && \
		 echo "Waiting for database to be ready..." && \
		 sleep 10 && \
		 docker exec -i stock-scanner-timescaledb psql -U postgres -d stock_scanner < scripts/migrations/002_create_alert_history_table.sql)
	@docker exec -i stock-scanner-timescaledb psql -U postgres -d stock_scanner < scripts/migrations/003_create_rules_table.sql || \
		(echo "⚠️  TimescaleDB container not running. Starting infrastructure..." && \
		 docker-compose -f config/docker-compose.yaml up -d timescaledb && \
		 echo "Waiting for database to be ready..." && \
		 sleep 10 && \
		 docker exec -i stock-scanner-timescaledb psql -U postgres -d stock_scanner < scripts/migrations/003_create_rules_table.sql)
	@docker exec -i stock-scanner-timescaledb psql -U postgres -d stock_scanner < scripts/migrations/004_create_toplist_configs_table.sql || \
		(echo "⚠️  TimescaleDB container not running. Starting infrastructure..." && \
		 docker-compose -f config/docker-compose.yaml up -d timescaledb && \
		 echo "Waiting for database to be ready..." && \
		 sleep 10 && \
		 docker exec -i stock-scanner-timescaledb psql -U postgres -d stock_scanner < scripts/migrations/004_create_toplist_configs_table.sql)
## remove cooldown table migration
	@docker exec -i stock-scanner-timescaledb psql -U postgres -d stock_scanner < scripts/migrations/005_remove_cooldown_from_rules.sql || \
		(echo "⚠️  TimescaleDB container not running. Starting infrastructure..." && \
		 docker-compose -f config/docker-compose.yaml up -d timescaledb && \
		 echo "Waiting for database to be ready..." && \
		 sleep 10 && \
		 docker exec -i stock-scanner-timescaledb psql -U postgres -d stock_scanner < scripts/migrations/005_remove_cooldown_from_rules.sql)
## seed system toplists
	@docker exec -i stock-scanner-timescaledb psql -U postgres -d stock_scanner < scripts/migrations/006_seed_system_toplists.sql || \
		(echo "⚠️  TimescaleDB container not running. Starting infrastructure..." && \
		 docker-compose -f config/docker-compose.yaml up -d timescaledb && \
		 echo "Waiting for database to be ready..." && \
		 sleep 10 && \
		 docker exec -i stock-scanner-timescaledb psql -U postgres -d stock_scanner < scripts/migrations/006_seed_system_toplists.sql)

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...

lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run

run-ingest: ## Run ingest service (requires build first)
	@./bin/ingest

run-bars: ## Run bars service (requires build first)
	@./bin/bars

run-indicator: ## Run indicator service (requires build first)
	@./bin/indicator

run-scanner: ## Run scanner service (requires build first)
	@./bin/scanner

run-alert: ## Run alert service (requires build first)
	@./bin/alert

run-ws-gateway: ## Run WebSocket gateway service (requires build first)
	@./bin/ws-gateway

run-api: ## Run API service (requires build first)
	@./bin/api

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

