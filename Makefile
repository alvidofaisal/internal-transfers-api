.PHONY: help dev build test test-unit test-integration migrate clean docker-build docker-up docker-down fmt lint perf

# Default target
help:
	@echo "Available commands:"
	@echo "  dev              - Start local development environment"
	@echo "  build            - Build the application binary"
	@echo "  test             - Run all tests"
	@echo "  test-unit        - Run unit tests only"
	@echo "  test-integration - Run integration tests only"
	@echo "  migrate          - Run database migrations"
	@echo "  fmt              - Format Go code"
	@echo "  lint             - Run linter"
	@echo "  perf             - Run performance tests"
	@echo "  docker-build     - Build Docker image"
	@echo "  docker-up        - Start Docker Compose services"
	@echo "  docker-down      - Stop Docker Compose services"
	@echo "  clean            - Clean build artifacts"

# Development environment
dev: docker-up
	@echo "Development environment is running at http://localhost:8080"
	@echo "Health check: http://localhost:8080/healthz"

# Build the application
build:
	@echo "Building application..."
	go build -o bin/server cmd/server/main.go

# Run all tests
test: test-unit test-integration

# Run unit tests
test-unit:
	@echo "Running unit tests..."
	go test -v -race -coverprofile=coverage.out ./internal/...

# Run integration tests (with testcontainers)
test-integration:
	@echo "Running integration tests..."
	go test -v -race -tags=integration ./test/...

# Database migrations (placeholder for now)
migrate:
	@echo "Running database migrations..."
	@echo "Migrations will be implemented in the next phase"

# Format Go code
fmt:
	@echo "Formatting Go code..."
	go fmt ./...
	goimports -w .

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run ./...

# Performance tests
perf:
	@echo "Running performance tests..."
	@if [ ! -f perf/transfer.js ]; then echo "Performance tests will be implemented in the next phase"; else k6 run perf/transfer.js; fi

# Docker commands
docker-build:
	@echo "Building Docker image..."
	docker build -t internal-transfers-api:latest .

docker-up:
	@echo "Starting Docker Compose services..."
	@echo "Make sure POSTGRES_DB, POSTGRES_USER, POSTGRES_PASSWORD are set"
	docker compose up -d
	@echo "Waiting for services to be ready..."
	@for i in $$(seq 1 30); do \
		if docker compose exec postgres pg_isready -U postgres >/dev/null 2>&1; then \
			echo "Database is ready!"; \
			exit 0; \
		fi; \
		echo "Waiting... ($$i/30)"; \
		sleep 2; \
	done; \
	echo "Database failed to start within 60 seconds" && exit 1

docker-down:
	@echo "Stopping Docker Compose services..."
	docker compose down

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out
	docker compose down -v --remove-orphans
	docker system prune -f

# Install development dependencies
deps:
	@echo "Installing development dependencies..."
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run the server locally (without Docker)
run:
	@echo "Starting server locally..."
	@echo "Make sure to set database environment variables (DB_HOST, DB_USER, DB_PASSWORD, etc.)"
	go run cmd/server/main.go 