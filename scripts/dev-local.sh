#!/bin/bash
# Local Development Script
# Runs middleware in Docker, frontend and backend locally

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    print_info "Checking prerequisites..."

    # Check Docker
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed"
        exit 1
    fi

    # Check Go
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed"
        exit 1
    fi
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    print_info "Go version: $GO_VERSION"

    # Check Node.js
    if ! command -v node &> /dev/null; then
        print_error "Node.js is not installed"
        exit 1
    fi
    NODE_VERSION=$(node --version)
    print_info "Node.js version: $NODE_VERSION"

    # Check npm
    if ! command -v npm &> /dev/null; then
        print_error "npm is not installed"
        exit 1
    fi

    print_success "All prerequisites met"
}

# Start middleware containers
start_middleware() {
    print_info "Starting middleware containers..."
    cd "$PROJECT_ROOT"
    docker compose -f docker-compose.middleware.yml up -d

    # Wait for services to be healthy
    print_info "Waiting for PostgreSQL to be ready..."
    until docker compose -f docker-compose.middleware.yml exec -T postgres pg_isready -U aio -d ai_orchestration &> /dev/null; do
        sleep 1
    done
    print_success "PostgreSQL is ready"

    print_info "Waiting for Redis to be ready..."
    until docker compose -f docker-compose.middleware.yml exec -T redis redis-cli ping &> /dev/null; do
        sleep 1
    done
    print_success "Redis is ready"

    print_success "All middleware containers are running"
}

# Stop middleware containers
stop_middleware() {
    print_info "Stopping middleware containers..."
    cd "$PROJECT_ROOT"
    docker compose -f docker-compose.middleware.yml down
    print_success "Middleware containers stopped"
}

# Run backend API
run_api() {
    print_info "Starting backend API..."
    cd "$PROJECT_ROOT/backend"

    # Set environment variables
    export DATABASE_URL="postgres://aio:aio_password@localhost:5432/ai_orchestration?sslmode=disable"
    export REDIS_URL="redis://localhost:6379"
    export PORT=8080
    export AUTH_ENABLED=false
    export TELEMETRY_ENABLED=false

    go run ./cmd/api
}

# Run worker
run_worker() {
    print_info "Starting worker..."
    cd "$PROJECT_ROOT/backend"

    # Set environment variables
    export DATABASE_URL="postgres://aio:aio_password@localhost:5432/ai_orchestration?sslmode=disable"
    export REDIS_URL="redis://localhost:6379"
    export TELEMETRY_ENABLED=false

    # Load API keys from .env if exists
    if [ -f "$PROJECT_ROOT/.env" ]; then
        export $(grep -E '^(OPENAI_API_KEY|ANTHROPIC_API_KEY)=' "$PROJECT_ROOT/.env" | xargs)
    fi

    go run ./cmd/worker
}

# Run frontend
run_frontend() {
    print_info "Starting frontend..."
    cd "$PROJECT_ROOT/frontend"

    # Install dependencies if needed
    if [ ! -d "node_modules" ]; then
        print_info "Installing frontend dependencies..."
        npm install
    fi

    npm run dev
}

# Show help
show_help() {
    echo "Usage: $0 <command>"
    echo ""
    echo "Commands:"
    echo "  middleware:start  Start middleware containers (PostgreSQL, Redis, Keycloak, Jaeger)"
    echo "  middleware:stop   Stop middleware containers"
    echo "  middleware:logs   Show middleware logs"
    echo "  api               Run backend API locally"
    echo "  worker            Run worker locally"
    echo "  frontend          Run frontend locally"
    echo "  check             Check prerequisites"
    echo "  help              Show this help message"
    echo ""
    echo "Example workflow:"
    echo "  1. $0 middleware:start    # Start infrastructure"
    echo "  2. $0 api                 # In terminal 1"
    echo "  3. $0 worker              # In terminal 2"
    echo "  4. $0 frontend            # In terminal 3"
    echo ""
    echo "URLs:"
    echo "  Frontend:  http://localhost:3000"
    echo "  API:       http://localhost:8080"
    echo "  Keycloak:  http://localhost:8180"
    echo "  Jaeger:    http://localhost:16686"
}

# Main
case "${1:-help}" in
    "middleware:start")
        check_prerequisites
        start_middleware
        ;;
    "middleware:stop")
        stop_middleware
        ;;
    "middleware:logs")
        cd "$PROJECT_ROOT"
        docker compose -f docker-compose.middleware.yml logs -f
        ;;
    "api")
        run_api
        ;;
    "worker")
        run_worker
        ;;
    "frontend")
        run_frontend
        ;;
    "check")
        check_prerequisites
        ;;
    "help"|*)
        show_help
        ;;
esac
