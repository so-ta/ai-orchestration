# AI Orchestration - Development Makefile
# Usage: make <target>

.PHONY: help dev dev-middleware dev-api dev-worker dev-frontend stop stop-all restart restart-api restart-worker restart-frontend install-tools

# Go environment - using goenv
GOENV_ROOT := $(HOME)/.anyenv/envs/goenv
GO_VERSION := 1.25.3
export GOROOT := $(GOENV_ROOT)/versions/$(GO_VERSION)
export GOPATH := $(HOME)/go/$(GO_VERSION)
export PATH := $(GOPATH)/bin:$(GOROOT)/bin:$(PATH)
AIR := $(GOPATH)/bin/air

# PID/Paneファイル管理
PID_DIR := .pids
API_PID := $(PID_DIR)/api.pid
WORKER_PID := $(PID_DIR)/worker.pid
FRONTEND_PID := $(PID_DIR)/frontend.pid
API_PANE := $(PID_DIR)/api.pane
WORKER_PANE := $(PID_DIR)/worker.pane
FRONTEND_PANE := $(PID_DIR)/frontend.pane

# tmux検出
IN_TMUX := $(TMUX)

# Default target
help:
	@echo "AI Orchestration - Development Commands"
	@echo ""
	@echo "起動 (ホットリロード):"
	@echo "  make dev             - 全サービス起動"
	@echo "                         tmux内: api/worker/frontend window作成"
	@echo "                         tmux外: バックグラウンド起動"
	@echo "  make dev-middleware  - ミドルウェアのみ起動 (DB, Redis, Keycloak, Jaeger)"
	@echo "  make dev-api         - API 起動/再起動 (起動済paneがあればそこで再起動)"
	@echo "  make dev-worker      - Worker 起動/再起動 (起動済paneがあればそこで再起動)"
	@echo "  make dev-frontend    - Frontend 起動/再起動 (起動済paneがあればそこで再起動)"
	@echo ""
	@echo "再起動:"
	@echo "  make restart         - 全サービス再起動"
	@echo "  make restart-api     - API 再起動"
	@echo "  make restart-worker  - Worker 再起動"
	@echo "  make restart-frontend- Frontend 再起動"
	@echo ""
	@echo "停止:"
	@echo "  make stop            - アプリサービス停止"
	@echo "  make stop-all        - ミドルウェア含む全停止"
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
	@echo "  API:       http://localhost:8090"
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

# Start middleware (Docker Compose)
dev-middleware:
	@echo "Starting middleware (docker compose)..."
	@docker compose -f docker-compose.middleware.yml up -d
	@sleep 3
	@echo "Middleware started"

# API 起動/再起動 (paneが存在すればそこで再起動、なければフォアグラウンド起動)
dev-api:
	@if [ -f $(API_PANE) ] && tmux list-panes -a -F '#{pane_id}' 2>/dev/null | grep -qF "$$(cat $(API_PANE))"; then \
		pane_id=$$(cat $(API_PANE)); \
		echo "Restarting API in pane $$pane_id..."; \
		tmux send-keys -t "$$pane_id" C-c; \
		sleep 1; \
		tmux send-keys -t "$$pane_id" 'cd $(CURDIR) && make _dev-api-fg' Enter; \
	else \
		$(MAKE) _dev-api-fg; \
	fi

# Worker 起動/再起動 (paneが存在すればそこで再起動、なければフォアグラウンド起動)
dev-worker:
	@if [ -f $(WORKER_PANE) ] && tmux list-panes -a -F '#{pane_id}' 2>/dev/null | grep -qF "$$(cat $(WORKER_PANE))"; then \
		pane_id=$$(cat $(WORKER_PANE)); \
		echo "Restarting Worker in pane $$pane_id..."; \
		tmux send-keys -t "$$pane_id" C-c; \
		sleep 1; \
		tmux send-keys -t "$$pane_id" 'cd $(CURDIR) && make _dev-worker-fg' Enter; \
	else \
		$(MAKE) _dev-worker-fg; \
	fi

# Frontend 起動/再起動 (paneが存在すればそこで再起動、なければフォアグラウンド起動)
dev-frontend:
	@if [ -f $(FRONTEND_PANE) ] && tmux list-panes -a -F '#{pane_id}' 2>/dev/null | grep -qF "$$(cat $(FRONTEND_PANE))"; then \
		pane_id=$$(cat $(FRONTEND_PANE)); \
		echo "Restarting Frontend in pane $$pane_id..."; \
		tmux send-keys -t "$$pane_id" C-c; \
		sleep 1; \
		tmux send-keys -t "$$pane_id" 'cd $(CURDIR) && make _dev-frontend-fg' Enter; \
	else \
		$(MAKE) _dev-frontend-fg; \
	fi

# 内部用: フォアグラウンド起動コマンド (tmux内ならpane IDを保存)
_dev-api-fg:
	@echo "Starting API with hot reload..."
	@mkdir -p $(PID_DIR)
	@if [ -n "$$TMUX_PANE" ]; then echo "$$TMUX_PANE" > $(API_PANE); fi
	cd backend && \
	DATABASE_URL="postgres://aio:aio_password@localhost:5432/ai_orchestration?sslmode=disable" \
	REDIS_URL="redis://localhost:6379" \
	PORT=8090 \
	AUTH_ENABLED=false \
	TELEMETRY_ENABLED=false \
	$(AIR) -c .air.toml

_dev-worker-fg:
	@echo "Starting Worker with hot reload..."
	@mkdir -p $(PID_DIR)
	@if [ -n "$$TMUX_PANE" ]; then echo "$$TMUX_PANE" > $(WORKER_PANE); fi
	cd backend && \
	DATABASE_URL="postgres://aio:aio_password@localhost:5432/ai_orchestration?sslmode=disable" \
	REDIS_URL="redis://localhost:6379" \
	TELEMETRY_ENABLED=false \
	$(AIR) -c .air.worker.toml

_dev-frontend-fg:
	@echo "Starting Frontend with hot reload..."
	@mkdir -p $(PID_DIR)
	@if [ -n "$$TMUX_PANE" ]; then echo "$$TMUX_PANE" > $(FRONTEND_PANE); fi
	cd frontend && npm run dev

# Start all services with auto-detection
dev: dev-middleware
	@echo "Starting services..."
ifdef IN_TMUX
	@echo "Running in tmux..."
	@# API: 既存paneがあればそこで再起動、なければ新しいwindowを作成
	@if [ -f $(API_PANE) ] && tmux list-panes -a -F '#{pane_id}' 2>/dev/null | grep -qF "$$(cat $(API_PANE))"; then \
		pane_id=$$(cat $(API_PANE)); \
		echo "Restarting API in pane $$pane_id..."; \
		tmux send-keys -t "$$pane_id" C-c; \
		sleep 1; \
		tmux send-keys -t "$$pane_id" 'cd $(CURDIR) && make _dev-api-fg' Enter; \
	else \
		echo "Creating new window for API..."; \
		tmux new-window -n api 'cd $(CURDIR) && make _dev-api-fg'; \
	fi
	@# Worker: 既存paneがあればそこで再起動、なければ新しいwindowを作成
	@if [ -f $(WORKER_PANE) ] && tmux list-panes -a -F '#{pane_id}' 2>/dev/null | grep -qF "$$(cat $(WORKER_PANE))"; then \
		pane_id=$$(cat $(WORKER_PANE)); \
		echo "Restarting Worker in pane $$pane_id..."; \
		tmux send-keys -t "$$pane_id" C-c; \
		sleep 1; \
		tmux send-keys -t "$$pane_id" 'cd $(CURDIR) && make _dev-worker-fg' Enter; \
	else \
		echo "Creating new window for Worker..."; \
		tmux new-window -n worker 'cd $(CURDIR) && make _dev-worker-fg'; \
	fi
	@# Frontend: 既存paneがあればそこで再起動、なければ新しいwindowを作成
	@if [ -f $(FRONTEND_PANE) ] && tmux list-panes -a -F '#{pane_id}' 2>/dev/null | grep -qF "$$(cat $(FRONTEND_PANE))"; then \
		pane_id=$$(cat $(FRONTEND_PANE)); \
		echo "Restarting Frontend in pane $$pane_id..."; \
		tmux send-keys -t "$$pane_id" C-c; \
		sleep 1; \
		tmux send-keys -t "$$pane_id" 'cd $(CURDIR) && make _dev-frontend-fg' Enter; \
	else \
		echo "Creating new window for Frontend..."; \
		tmux new-window -n frontend 'cd $(CURDIR) && make _dev-frontend-fg'; \
	fi
	@echo ""
	@echo "Services started. Restart: make dev-api / dev-worker / dev-frontend"
else
	@echo "Running in background mode..."
	@mkdir -p backend/tmp frontend/.nuxt $(PID_DIR)
	@cd backend && \
	DATABASE_URL="postgres://aio:aio_password@localhost:5432/ai_orchestration?sslmode=disable" \
	REDIS_URL="redis://localhost:6379" \
	PORT=8090 \
	AUTH_ENABLED=false \
	TELEMETRY_ENABLED=false \
	nohup $(AIR) -c .air.toml > tmp/api.log 2>&1 & echo $$! > ../$(API_PID)
	@cd backend && \
	DATABASE_URL="postgres://aio:aio_password@localhost:5432/ai_orchestration?sslmode=disable" \
	REDIS_URL="redis://localhost:6379" \
	TELEMETRY_ENABLED=false \
	nohup $(AIR) -c .air.worker.toml > tmp/worker.log 2>&1 & echo $$! > ../$(WORKER_PID)
	@cd frontend && nohup npm run dev > .nuxt/dev.log 2>&1 & echo $$! > ../$(FRONTEND_PID)
	@sleep 3
	@echo ""
	@echo "All services started in background!"
	@echo "  API:      http://localhost:8090 (logs: backend/tmp/api.log)"
	@echo "  Worker:   (logs: backend/tmp/worker.log)"
	@echo "  Frontend: http://localhost:3000 (logs: frontend/.nuxt/dev.log)"
	@echo ""
	@echo "View logs: make logs-api / logs-worker / logs-frontend"
	@echo "Stop:      make stop"
endif

# ============================================================================
# 再起動コマンド
# ============================================================================

# Restart API (detects startup method: tmux pane or background process)
restart-api:
	@echo "Restarting API..."
	@if [ -f $(API_PANE) ] && tmux list-panes -a -F '#{pane_id}' 2>/dev/null | grep -qF "$$(cat $(API_PANE))"; then \
		pane_id=$$(cat $(API_PANE)); \
		echo "Restarting in tmux pane $$pane_id..."; \
		tmux send-keys -t "$$pane_id" C-c; \
		sleep 1; \
		tmux send-keys -t "$$pane_id" 'cd $(CURDIR) && make _dev-api-fg' Enter; \
		echo "API restarted in pane $$pane_id"; \
	elif [ -f $(API_PID) ]; then \
		echo "Restarting background process..."; \
		pid=$$(cat $(API_PID)); \
		if kill -0 $$pid 2>/dev/null; then \
			kill $$pid 2>/dev/null || true; \
			sleep 1; \
			kill -9 $$pid 2>/dev/null || true; \
		fi; \
		rm -f $(API_PID); \
		mkdir -p backend/tmp $(PID_DIR); \
		cd backend && \
		DATABASE_URL="postgres://aio:aio_password@localhost:5432/ai_orchestration?sslmode=disable" \
		REDIS_URL="redis://localhost:6379" \
		PORT=8090 \
		AUTH_ENABLED=false \
		TELEMETRY_ENABLED=false \
		nohup $(AIR) -c .air.toml > tmp/api.log 2>&1 & echo $$! > ../$(API_PID); \
		sleep 2; \
		echo "API restarted. Logs: backend/tmp/api.log"; \
	else \
		echo "Error: API not running (no tmux pane or PID file found)"; \
		exit 1; \
	fi

# Restart Worker (detects startup method: tmux pane or background process)
restart-worker:
	@echo "Restarting Worker..."
	@if [ -f $(WORKER_PANE) ] && tmux list-panes -a -F '#{pane_id}' 2>/dev/null | grep -qF "$$(cat $(WORKER_PANE))"; then \
		pane_id=$$(cat $(WORKER_PANE)); \
		echo "Restarting in tmux pane $$pane_id..."; \
		tmux send-keys -t "$$pane_id" C-c; \
		sleep 1; \
		tmux send-keys -t "$$pane_id" 'cd $(CURDIR) && make _dev-worker-fg' Enter; \
		echo "Worker restarted in pane $$pane_id"; \
	elif [ -f $(WORKER_PID) ]; then \
		echo "Restarting background process..."; \
		pid=$$(cat $(WORKER_PID)); \
		if kill -0 $$pid 2>/dev/null; then \
			kill $$pid 2>/dev/null || true; \
			sleep 1; \
			kill -9 $$pid 2>/dev/null || true; \
		fi; \
		rm -f $(WORKER_PID); \
		mkdir -p backend/tmp $(PID_DIR); \
		cd backend && \
		DATABASE_URL="postgres://aio:aio_password@localhost:5432/ai_orchestration?sslmode=disable" \
		REDIS_URL="redis://localhost:6379" \
		TELEMETRY_ENABLED=false \
		nohup $(AIR) -c .air.worker.toml > tmp/worker.log 2>&1 & echo $$! > ../$(WORKER_PID); \
		sleep 2; \
		echo "Worker restarted. Logs: backend/tmp/worker.log"; \
	else \
		echo "Error: Worker not running (no tmux pane or PID file found)"; \
		exit 1; \
	fi

# Restart Frontend (detects startup method: tmux pane or background process)
restart-frontend:
	@echo "Restarting Frontend..."
	@if [ -f $(FRONTEND_PANE) ] && tmux list-panes -a -F '#{pane_id}' 2>/dev/null | grep -qF "$$(cat $(FRONTEND_PANE))"; then \
		pane_id=$$(cat $(FRONTEND_PANE)); \
		echo "Restarting in tmux pane $$pane_id..."; \
		tmux send-keys -t "$$pane_id" C-c; \
		sleep 1; \
		tmux send-keys -t "$$pane_id" 'cd $(CURDIR) && make _dev-frontend-fg' Enter; \
		echo "Frontend restarted in pane $$pane_id"; \
	elif [ -f $(FRONTEND_PID) ]; then \
		echo "Restarting background process..."; \
		pid=$$(cat $(FRONTEND_PID)); \
		if kill -0 $$pid 2>/dev/null; then \
			kill $$pid 2>/dev/null || true; \
			sleep 1; \
			kill -9 $$pid 2>/dev/null || true; \
		fi; \
		rm -f $(FRONTEND_PID); \
		mkdir -p frontend/.nuxt $(PID_DIR); \
		cd frontend && nohup npm run dev > .nuxt/dev.log 2>&1 & echo $$! > ../$(FRONTEND_PID); \
		sleep 3; \
		echo "Frontend restarted. Logs: frontend/.nuxt/dev.log"; \
	else \
		echo "Error: Frontend not running (no tmux pane or PID file found)"; \
		exit 1; \
	fi

# Restart all services
restart: dev-middleware restart-api restart-worker restart-frontend
	@echo ""
	@echo "All services restarted!"

# ============================================================================
# 停止コマンド
# ============================================================================

# Stop app services (API, Worker, Frontend)
stop:
	@echo "Stopping app services..."
	@# tmux paneがあればC-cを送信
	@if [ -f $(API_PANE) ]; then \
		pane_id=$$(cat $(API_PANE)); \
		tmux send-keys -t "$$pane_id" C-c 2>/dev/null || true; \
		rm -f $(API_PANE); \
	fi
	@if [ -f $(WORKER_PANE) ]; then \
		pane_id=$$(cat $(WORKER_PANE)); \
		tmux send-keys -t "$$pane_id" C-c 2>/dev/null || true; \
		rm -f $(WORKER_PANE); \
	fi
	@if [ -f $(FRONTEND_PANE) ]; then \
		pane_id=$$(cat $(FRONTEND_PANE)); \
		tmux send-keys -t "$$pane_id" C-c 2>/dev/null || true; \
		rm -f $(FRONTEND_PANE); \
	fi
	@# PIDファイルがあれば停止
	@if [ -f $(API_PID) ]; then \
		pid=$$(cat $(API_PID)); \
		kill $$pid 2>/dev/null || true; \
		rm -f $(API_PID); \
	fi
	@if [ -f $(WORKER_PID) ]; then \
		pid=$$(cat $(WORKER_PID)); \
		kill $$pid 2>/dev/null || true; \
		rm -f $(WORKER_PID); \
	fi
	@if [ -f $(FRONTEND_PID) ]; then \
		pid=$$(cat $(FRONTEND_PID)); \
		kill $$pid 2>/dev/null || true; \
		rm -f $(FRONTEND_PID); \
	fi
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
	rm -rf $(PID_DIR)
