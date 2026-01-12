# AI Orchestration - Development Makefile
# Usage: make <target>

.PHONY: help dev dev-all dev-middleware dev-api dev-worker dev-frontend stop install-tools

# Go environment - using goenv
GOENV_ROOT := $(HOME)/.anyenv/envs/goenv
GO_VERSION := 1.25.3
export GOROOT := $(GOENV_ROOT)/versions/$(GO_VERSION)
export GOPATH := $(HOME)/go/$(GO_VERSION)
export PATH := $(GOPATH)/bin:$(GOROOT)/bin:$(PATH)
AIR := $(GOPATH)/bin/air

# Default target
help:
	@echo "AI Orchestration - Development Commands"
	@echo ""
	@echo "Quick Start:"
	@echo "  make dev          - Start middleware + all services with hot reload"
	@echo "  make stop         - Stop all services"
	@echo ""
	@echo "Individual Services:"
	@echo "  make dev-middleware  - Start middleware only (PostgreSQL, Redis, Keycloak, Jaeger)"
	@echo "  make dev-api         - Start API with hot reload"
	@echo "  make dev-worker      - Start Worker with hot reload"
	@echo "  make dev-frontend    - Start Frontend with hot reload"
	@echo ""
	@echo "Database:"
	@echo "  make db-apply     - Apply schema to database"
	@echo "  make db-seed      - Load seed data (initial data)"
	@echo "  make db-reset     - Reset database (drop, recreate, seed)"
	@echo "  make db-export    - Export current schema for backup"
	@echo ""
	@echo "Setup:"
	@echo "  make install-tools   - Install development tools (air for Go hot reload)"
	@echo ""
	@echo "Testing:"
	@echo "  make test-backend    - Run backend tests"
	@echo "  make test-frontend   - Run frontend tests"
	@echo "  make test            - Run all tests"
	@echo ""
	@echo "URLs:"
	@echo "  Frontend:  http://localhost:3000"
	@echo "  API:       http://localhost:8080"
	@echo "  Keycloak:  http://localhost:8180"
	@echo "  Jaeger:    http://localhost:16686"

# Install development tools
install-tools:
	@echo "Installing air for Go hot reload..."
	@echo "Using Go: $(GOROOT)"
	go install github.com/air-verse/air@latest
	@echo "Done! Air installed at: $(AIR)"

# Start middleware only
dev-middleware:
	docker compose -f docker-compose.middleware.yml up -d
	@echo "Middleware started. Waiting for services to be healthy..."
	@sleep 5
	@docker compose -f docker-compose.middleware.yml ps

# Stop middleware
stop-middleware:
	docker compose -f docker-compose.middleware.yml down

# Start API with hot reload
dev-api:
	@echo "Starting API with hot reload..."
	cd backend && \
	DATABASE_URL="postgres://aio:aio_password@localhost:5432/ai_orchestration?sslmode=disable" \
	REDIS_URL="redis://localhost:6379" \
	PORT=8080 \
	AUTH_ENABLED=false \
	TELEMETRY_ENABLED=false \
	$(AIR) -c .air.toml

# Start Worker with hot reload
dev-worker:
	@echo "Starting Worker with hot reload..."
	cd backend && \
	DATABASE_URL="postgres://aio:aio_password@localhost:5432/ai_orchestration?sslmode=disable" \
	REDIS_URL="redis://localhost:6379" \
	TELEMETRY_ENABLED=false \
	$(AIR) -c .air.worker.toml

# Start Frontend with hot reload
dev-frontend:
	@echo "Starting Frontend with hot reload..."
	cd frontend && npm run dev

# Start all services (requires multiple terminals or use tmux)
dev-all: dev-middleware
	@echo ""
	@echo "Middleware started!"
	@echo ""
	@echo "Now run these commands in separate terminals:"
	@echo "  Terminal 1: make dev-api"
	@echo "  Terminal 2: make dev-worker"
	@echo "  Terminal 3: make dev-frontend"

# Full development environment with tmux (if available)
dev:
	@if command -v tmux >/dev/null 2>&1; then \
		$(MAKE) dev-tmux; \
	else \
		$(MAKE) dev-all; \
	fi

# Development with tmux
dev-tmux: dev-middleware
	@echo "Starting development environment with tmux..."
	@sleep 3
	tmux new-session -d -s aio -n api 'make dev-api'
	tmux new-window -t aio -n worker 'make dev-worker'
	tmux new-window -t aio -n frontend 'make dev-frontend'
	tmux select-window -t aio:api
	@echo ""
	@echo "Development environment started in tmux session 'aio'"
	@echo "Attach with: tmux attach -t aio"
	@echo ""
	@echo "Tmux shortcuts:"
	@echo "  Ctrl+b n    - Next window"
	@echo "  Ctrl+b p    - Previous window"
	@echo "  Ctrl+b d    - Detach"
	@echo "  Ctrl+b &    - Kill window"

# Stop all services
stop:
	@echo "Stopping all services..."
	-docker compose -f docker-compose.middleware.yml down
	-tmux kill-session -t aio 2>/dev/null || true
	@echo "All services stopped"

# Backend tests
test-backend:
	cd backend && go test ./...

# Backend E2E tests
test-backend-e2e:
	cd backend && go test ./tests/e2e/... -v

# Frontend tests
test-frontend:
	cd frontend && npm run check

# All tests
test: test-backend test-frontend

# Logs
logs-middleware:
	docker compose -f docker-compose.middleware.yml logs -f

logs-api:
	@tail -f backend/tmp/api.log 2>/dev/null || echo "API not running or no log file"

# Database management
DB_USER := aio
DB_NAME := ai_orchestration

# Apply schema (creates all tables)
db-apply:
	@echo "Applying schema..."
	@docker compose exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) -f /dev/stdin < backend/schema/schema.sql
	@echo "Schema applied successfully!"

# Load seed data (initial data)
db-seed:
	@echo "Loading seed data..."
	@docker compose exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) -f /dev/stdin < backend/schema/seed.sql
	@echo "Seed data loaded!"

# Reset database (drop all tables and recreate with seed data)
db-reset:
	@echo "Resetting database..."
	@docker compose exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public; CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";"
	@$(MAKE) db-apply
	@$(MAKE) db-seed
	@echo "Database reset complete!"

# Export current schema (for reference/backup)
db-export:
	@echo "Exporting current schema..."
	@docker compose exec -T postgres pg_dump -U $(DB_USER) -d $(DB_NAME) --schema-only --no-owner --no-privileges \
		> backend/schema/schema_exported.sql
	@echo "Schema exported to backend/schema/schema_exported.sql"

# Clean
clean:
	rm -rf backend/tmp
	rm -rf frontend/.nuxt
	rm -rf frontend/node_modules/.cache
