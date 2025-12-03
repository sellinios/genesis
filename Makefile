# =============================================================================
# GENESIS - Makefile
# =============================================================================

.PHONY: help build run dev test clean docker-build docker-up docker-down docker-logs

# Default target
help:
	@echo "Genesis - Dynamic Business Platform"
	@echo ""
	@echo "Usage:"
	@echo "  make build        - Build the Genesis binary"
	@echo "  make run          - Run Genesis locally"
	@echo "  make dev          - Run with hot reload (requires air)"
	@echo "  make test         - Run tests"
	@echo "  make clean        - Clean build artifacts"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build - Build Docker image"
	@echo "  make docker-up    - Start Genesis with Docker Compose"
	@echo "  make docker-down  - Stop Docker Compose"
	@echo "  make docker-logs  - View Docker logs"
	@echo "  make docker-clean - Remove all Docker resources"
	@echo ""
	@echo "Production:"
	@echo "  make prod-up      - Start production environment"
	@echo "  make prod-down    - Stop production environment"

# -----------------------------------------------------------------------------
# Local Development
# -----------------------------------------------------------------------------

build:
	@echo "Building Genesis..."
	@go build -ldflags="-X main.Version=$$(git describe --tags --always --dirty 2>/dev/null || echo dev)" \
		-o bin/genesis ./cmd/server

run: build
	@echo "Starting Genesis..."
	@./bin/genesis

dev:
	@echo "Starting Genesis with hot reload..."
	@air -c .air.toml 2>/dev/null || go run ./cmd/server

test:
	@echo "Running tests..."
	@go test -v ./...

clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@go clean

# -----------------------------------------------------------------------------
# Docker
# -----------------------------------------------------------------------------

docker-build:
	@echo "Building Docker image..."
	@docker build -t genesis:latest .

docker-up:
	@echo "Starting Genesis with Docker Compose..."
	@docker-compose up -d
	@echo ""
	@echo "Genesis is starting..."
	@echo "  - API: http://localhost:8090"
	@echo "  - Health: http://localhost:8090/api/health"
	@echo ""
	@echo "Run 'make docker-logs' to view logs"

docker-down:
	@echo "Stopping Docker Compose..."
	@docker-compose down

docker-logs:
	@docker-compose logs -f

docker-clean:
	@echo "Removing all Docker resources..."
	@docker-compose down -v --rmi local

# -----------------------------------------------------------------------------
# Production
# -----------------------------------------------------------------------------

prod-up:
	@echo "Starting production environment..."
	@docker-compose -f docker-compose.prod.yml up -d

prod-down:
	@echo "Stopping production environment..."
	@docker-compose -f docker-compose.prod.yml down

# -----------------------------------------------------------------------------
# Database
# -----------------------------------------------------------------------------

db-migrate:
	@echo "Running migrations..."
	@psql -h localhost -U genesis -d genesis -f migrations/001_genesis_core.sql
	@psql -h localhost -U genesis -d genesis -f migrations/002_connections_services.sql

db-reset:
	@echo "Resetting database..."
	@docker-compose exec postgres psql -U genesis -d genesis -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	@make db-migrate
