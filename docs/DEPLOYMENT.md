# Deployment Reference

Docker, Kubernetes, and environment configuration for development and production.

## Quick Reference

| Item | Value |
|------|-------|
| Development | Docker Compose |
| Production | ECS (別途構築) |
| Container Registry | Local build / Custom |
| API Port | 8080 |
| Frontend Port | 3000 |
| Keycloak Port | 8180 |
| Jaeger Port | 16686 |
| Health Endpoint | `/health`, `/ready` |

## Development Environment

### Docker Compose Services

| Service | Image | Port | Description |
|---------|-------|------|-------------|
| postgres | postgres:16-alpine | 5432 | PostgreSQL database |
| redis | redis:7-alpine | 6379 | Cache & job queue |
| keycloak | keycloak:24.0 | 8180 | OIDC authentication |
| api | ./backend | 8080 | Go API server |
| worker | ./backend | - | Job processor |
| frontend | ./frontend | 3000 | Nuxt web UI |
| jaeger | jaegertracing/all-in-one | 16686 | Distributed tracing |

### Commands

```bash
# Start all services
docker compose up -d

# Start with build
docker compose up -d --build

# View logs
docker compose logs -f api
docker compose logs -f worker
docker compose logs -f frontend

# Restart single service
docker compose restart api

# Stop all
docker compose down

# Stop and remove volumes
docker compose down -v

# Rebuild single service
docker compose up -d --build api
```

### Environment Variables

Create `.env` file in project root:

```bash
# LLM API Keys
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...

# Enable telemetry
TELEMETRY_ENABLED=true
```

### Service URLs

| Service | URL |
|---------|-----|
| API | http://localhost:8080 |
| Frontend | http://localhost:3000 |
| Keycloak Admin | http://localhost:8180/admin (admin/admin) |
| Jaeger UI | http://localhost:16686 |
| PostgreSQL | localhost:5432 |
| Redis | localhost:6379 |

### Default Credentials

| Service | User | Password |
|---------|------|----------|
| PostgreSQL | aio | aio_password |
| Keycloak Admin | admin | admin |
| Test User (admin) | admin@example.com | admin123 |
| Test User (builder) | builder@example.com | builder123 |

---

## Production Dockerfile

### Backend

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /worker ./cmd/worker

# Production stage
FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
RUN adduser -D -g '' appuser
WORKDIR /app
COPY --from=builder /api /worker ./
USER appuser
EXPOSE 8080
CMD ["./api"]
```

### Frontend

```dockerfile
# Build stage
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

# Production stage
FROM node:20-alpine
WORKDIR /app
COPY --from=builder /app/.output ./.output
EXPOSE 3000
CMD ["node", ".output/server/index.mjs"]
```

---

## Health Endpoints

### Liveness (/health)

- Returns immediately
- Checks process is alive
- Used by: K8s livenessProbe

```json
{"status": "ok"}
```

### Readiness (/ready)

- Checks dependencies
- Returns 503 if unhealthy
- Used by: K8s readinessProbe

```json
{
  "status": "ok",
  "components": {
    "database": "ok",
    "redis": "ok"
  }
}
```

---

## Monitoring

### OpenTelemetry

Enable: `TELEMETRY_ENABLED=true`

Traces exported to: `OTEL_EXPORTER_OTLP_ENDPOINT`

### Jaeger Setup

```yaml
# docker-compose
jaeger:
  image: jaegertracing/all-in-one:1.54
  ports:
    - "16686:16686"  # UI
    - "4317:4317"    # OTLP gRPC
    - "4318:4318"    # OTLP HTTP
```

### Key Metrics

- Request latency (P50, P95, P99)
- Error rate
- DAG execution duration
- Step execution duration
- Queue depth
- Active runs count

---

## Troubleshooting

### Common Issues

| Issue | Cause | Fix |
|-------|-------|-----|
| API 502 | DB connection failed | Check DATABASE_URL, postgres health |
| Worker not processing | Redis connection | Check REDIS_URL, redis health |
| Auth errors | Keycloak unavailable | Check KEYCLOAK_URL, keycloak health |
| Slow queries | Missing indexes | Check database logs, EXPLAIN |
| OOM killed | Memory limit too low | Increase limits in deployment |

### Debug Commands

```bash
# Docker Compose logs
docker compose logs -f api
docker compose logs -f worker

# Exec into container
docker compose exec api sh

# Check DB connection
docker compose exec api psql $DATABASE_URL -c "SELECT 1"

# Check Redis
docker compose exec api redis-cli -u $REDIS_URL PING
```

---

## Scaling Considerations

| Component | Strategy | Notes |
|-----------|----------|-------|
| API | Horizontal scaling | Stateless, CPU-based autoscaling recommended |
| Worker | Queue-based scaling | Each worker processes sequentially, more workers = more parallelism |
| Database | Connection pooling | PgBouncer recommended, read replicas for read-heavy |
| Redis | Cluster mode | Separate instances for cache vs queue (optional) |

## Related Documents

- [BACKEND.md](./BACKEND.md) - Backend architecture
- [FRONTEND.md](./FRONTEND.md) - Frontend architecture
- [DATABASE.md](./DATABASE.md) - Database schema and connection settings
- [API.md](./API.md) - Health check endpoints
