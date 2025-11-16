.PHONY: help build test clean run-ingest run-bars run-indicator run-scanner run-ws-gateway run-api docker-up docker-down migrate-up migrate-down

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
	@go build -o bin/ws-gateway ./cmd/ws_gateway
	@go build -o bin/api ./cmd/api

test: ## Run all tests
	@echo "Running tests..."
	@go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html

docker-up: ## Start Docker Compose services
	@echo "Starting Docker Compose services..."
	@docker-compose -f config/docker-compose.yaml up -d

docker-down: ## Stop Docker Compose services
	@echo "Stopping Docker Compose services..."
	@docker-compose -f config/docker-compose.yaml down

docker-logs: ## View Docker Compose logs
	@docker-compose -f config/docker-compose.yaml logs -f

migrate-up: ## Run database migrations
	@echo "Running migrations..."
	@psql -h localhost -U postgres -d stock_scanner -f scripts/migrations/001_create_bars_table.sql

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

run-ws-gateway: ## Run WebSocket gateway service (requires build first)
	@./bin/ws-gateway

run-api: ## Run API service (requires build first)
	@./bin/api

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

