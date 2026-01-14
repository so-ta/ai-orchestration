# AI Orchestration - Development Makefile
# Usage: make <target>

.PHONY: help dev dev-api dev-worker dev-frontend stop stop-all restart restart-api restart-worker restart-frontend install-tools

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
	@echo "起動 (ホットリロード):"
	@echo "  make dev             - ミドルウェア + API + Worker + Frontend を起動"
	@echo "  make dev-api         - API のみ起動"
	@echo "  make dev-worker      - Worker のみ起動"
	@echo "  make dev-frontend    - Frontend のみ起動"
	@echo ""
	@echo "再起動:"
	@echo "  make restart         - ミドルウェア確認 + API + Worker + Frontend を再起動"
	@echo "  make restart-api     - API のみ再起動"
	@echo "  make restart-worker  - Worker のみ再起動"
	@echo "  make restart-frontend- Frontend のみ再起動"
	@echo ""
	@echo "停止:"
	@echo "  make stop            - API + Worker + Frontend を停止"
	@echo "  make stop-all        - ミドルウェア含む全サービスを停止"
	@echo ""
	@echo "Database:"
	@echo "  make db-apply        - Apply schema to database"
	@echo "  make db-seed         - Load seed data (SQL)"
	@echo "  make db-reset        - Reset database (drop, recreate, seed)"
	@echo "  make db-export       - Export current schema"
	@echo ""
	@echo "Block Seeding (Go programmatic):"
	@echo "  make seed-blocks          - Migrate block definitions to database (UPSERT)"
	@echo "  make seed-blocks-validate - Validate block definitions only"
	@echo "  make seed-blocks-dry-run  - Show what would be changed"
	@echo ""
	@echo "Testing:"
	@echo "  make test            - Run all tests"
	@echo "  make test-backend    - Run backend tests"
	@echo "  make test-frontend   - Run frontend tests"
	@echo ""
	@echo "Setup:"
	@echo "  make install-tools   - Install air (Go hot reload)"
	@echo ""
	@echo "URLs:"
	@echo "  Frontend:  http://localhost:3000"
	@echo "  API:       http://localhost:8080"
	@echo "  Keycloak:  http://localhost:8180"
	@echo "  Jaeger:    http://localhost:16686"

# Install development tools
install-tools:
	@echo "Installing air for Go hot reload..."
	go install github.com/air-verse/air@latest
	@echo "Done! Air installed at: $(AIR)"

# ============================================================================
# 起動コマンド（ホットリロード）
# ============================================================================

# Start API with hot reload (foreground)
dev-api:
	@echo "Starting API with hot reload..."
	cd backend && \
	DATABASE_URL="postgres://aio:aio_password@localhost:5432/ai_orchestration?sslmode=disable" \
	REDIS_URL="redis://localhost:6379" \
	PORT=8080 \
	AUTH_ENABLED=false \
	TELEMETRY_ENABLED=false \
	$(AIR) -c .air.toml

# Start Worker with hot reload (foreground)
dev-worker:
	@echo "Starting Worker with hot reload..."
	cd backend && \
	DATABASE_URL="postgres://aio:aio_password@localhost:5432/ai_orchestration?sslmode=disable" \
	REDIS_URL="redis://localhost:6379" \
	TELEMETRY_ENABLED=false \
	$(AIR) -c .air.worker.toml

# Start Frontend with hot reload (foreground)
dev-frontend:
	@echo "Starting Frontend with hot reload..."
	cd frontend && npm run dev

# Start all services with tmux (includes middleware)
dev:
	@echo "Starting middleware (docker compose)..."
	@docker compose -f docker-compose.middleware.yml up -d
	@sleep 3
	@if command -v tmux >/dev/null 2>&1; then \
		$(MAKE) dev-tmux; \
	else \
		echo "tmux not found. Please run in separate terminals:"; \
		echo "  make dev-api"; \
		echo "  make dev-worker"; \
		echo "  make dev-frontend"; \
	fi

# Development with tmux
dev-tmux:
	@echo "Starting development environment with tmux..."
	@tmux kill-session -t aio 2>/dev/null || true
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

# ============================================================================
# 再起動コマンド
# ============================================================================

# Restart API (kill and restart in background, then show logs)
restart-api:
	@echo "Restarting API..."
	@pkill -f "air.*\.air\.toml" 2>/dev/null || true
	@pkill -f "tmp/api" 2>/dev/null || true
	@sleep 1
	@mkdir -p backend/tmp
	@cd backend && \
	DATABASE_URL="postgres://aio:aio_password@localhost:5432/ai_orchestration?sslmode=disable" \
	REDIS_URL="redis://localhost:6379" \
	PORT=8080 \
	AUTH_ENABLED=false \
	TELEMETRY_ENABLED=false \
	nohup $(AIR) -c .air.toml > tmp/api.log 2>&1 &
	@sleep 2
	@echo "API restarted. Logs: backend/tmp/api.log"
	@echo "Check: curl -s http://localhost:8080/health"

# Restart Worker (kill and restart in background)
restart-worker:
	@echo "Restarting Worker..."
	@pkill -f "air.*\.air\.worker\.toml" 2>/dev/null || true
	@pkill -f "tmp/worker" 2>/dev/null || true
	@sleep 1
	@mkdir -p backend/tmp
	@cd backend && \
	DATABASE_URL="postgres://aio:aio_password@localhost:5432/ai_orchestration?sslmode=disable" \
	REDIS_URL="redis://localhost:6379" \
	TELEMETRY_ENABLED=false \
	nohup $(AIR) -c .air.worker.toml > tmp/worker.log 2>&1 &
	@sleep 2
	@echo "Worker restarted. Logs: backend/tmp/worker.log"

# Restart Frontend (kill and restart in background)
restart-frontend:
	@echo "Restarting Frontend..."
	@pkill -f "nuxt" 2>/dev/null || true
	@pkill -f "node.*frontend" 2>/dev/null || true
	@sleep 1
	@mkdir -p frontend/.nuxt
	@cd frontend && nohup npm run dev > .nuxt/dev.log 2>&1 &
	@sleep 3
	@echo "Frontend restarted. Logs: frontend/.nuxt/dev.log"
	@echo "Access: http://localhost:3000"

# Restart all services (includes middleware check)
restart:
	@echo "Ensuring middleware is running..."
	@docker compose -f docker-compose.middleware.yml up -d
	@sleep 2
	@$(MAKE) restart-api
	@$(MAKE) restart-worker
	@$(MAKE) restart-frontend
	@echo ""
	@echo "All services restarted!"
	@echo "  API:      http://localhost:8080"
	@echo "  Frontend: http://localhost:3000"

# ============================================================================
# 停止コマンド
# ============================================================================

# Stop app services (API, Worker, Frontend)
stop:
	@echo "Stopping app services..."
	@tmux kill-session -t aio 2>/dev/null || true
	@pkill -f "air.*\.air\.toml" 2>/dev/null || true
	@pkill -f "air.*\.air\.worker\.toml" 2>/dev/null || true
	@pkill -f "tmp/api" 2>/dev/null || true
	@pkill -f "tmp/worker" 2>/dev/null || true
	@pkill -f "nuxt" 2>/dev/null || true
	@pkill -f "node.*frontend" 2>/dev/null || true
	@echo "App services stopped"

# Stop all services including middleware
stop-all: stop
	@echo "Stopping middleware..."
	@docker compose -f docker-compose.middleware.yml down
	@echo "All services stopped (including middleware)"

# ============================================================================
# テスト
# ============================================================================

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

# ============================================================================
# Database管理
# ============================================================================
DB_USER ?= aio
DB_PASSWORD ?= aio_password
DB_NAME ?= ai_orchestration
DB_CONTAINER ?= aio-postgres

# Apply schema
db-apply:
	@echo "Applying schema..."
	@cat backend/schema/schema.sql | docker exec -i $(DB_CONTAINER) psql -U $(DB_USER) -d $(DB_NAME)
	@echo "Schema applied!"

# Load seed data (SQL)
db-seed:
	@echo "Loading seed data (SQL)..."
	@cat backend/schema/seed.sql | docker exec -i $(DB_CONTAINER) psql -U $(DB_USER) -d $(DB_NAME)
	@echo "Seed data loaded!"

# Seed blocks (Go programmatic seeder)
seed-blocks:
	@echo "Running block seeder..."
	@cd backend && DATABASE_URL="postgres://$(DB_USER):$(DB_PASSWORD)@localhost:5432/$(DB_NAME)?sslmode=disable" go run ./cmd/seeder
	@echo "Block seeding complete!"

# Validate block definitions
seed-blocks-validate:
	@echo "Validating block definitions..."
	@cd backend && go run ./cmd/seeder -validate

# Seed blocks (dry run)
seed-blocks-dry-run:
	@echo "Running block seeder (dry run)..."
	@cd backend && DATABASE_URL="postgres://$(DB_USER):$(DB_PASSWORD)@localhost:5432/$(DB_NAME)?sslmode=disable" go run ./cmd/seeder -dry-run -verbose

# Reset database
db-reset:
	@echo "Resetting database..."
	@docker exec -i $(DB_CONTAINER) psql -U $(DB_USER) -d $(DB_NAME) -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public; CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"; CREATE EXTENSION IF NOT EXISTS vector;"
	@$(MAKE) db-apply
	@$(MAKE) db-seed
	@$(MAKE) seed-blocks
	@echo "Database reset complete!"

# Export schema
db-export:
	@echo "Exporting current schema..."
	@docker exec $(DB_CONTAINER) pg_dump -U $(DB_USER) -d $(DB_NAME) --schema-only --no-owner --no-privileges \
		> backend/schema/schema_exported.sql
	@echo "Schema exported to backend/schema/schema_exported.sql"

# ============================================================================
# ログ表示
# ============================================================================

logs-api:
	@tail -f backend/tmp/api.log 2>/dev/null || echo "API log not found"

logs-worker:
	@tail -f backend/tmp/worker.log 2>/dev/null || echo "Worker log not found"

logs-frontend:
	@tail -f frontend/.nuxt/dev.log 2>/dev/null || echo "Frontend log not found"

# ============================================================================
# クリーンアップ
# ============================================================================

clean:
	rm -rf backend/tmp
	rm -rf frontend/.nuxt
	rm -rf frontend/node_modules/.cache
