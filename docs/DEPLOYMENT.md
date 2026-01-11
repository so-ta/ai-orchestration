# Deployment Reference

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

## Kubernetes Deployment

### Manifests Structure

```
deploy/kubernetes/
├── namespace.yaml          # ai-orchestration namespace
├── configmap.yaml          # Non-secret config
├── secrets.yaml            # Credentials (base64)
├── api-deployment.yaml     # API Deployment + Service + ServiceAccount
├── worker-deployment.yaml  # Worker Deployment
├── ingress.yaml            # External access
├── hpa.yaml                # Horizontal Pod Autoscaler
└── kustomization.yaml      # Kustomize config
```

### Apply

```bash
# Using kustomize
kubectl apply -k deploy/kubernetes/

# Or individual files
kubectl apply -f deploy/kubernetes/namespace.yaml
kubectl apply -f deploy/kubernetes/configmap.yaml
kubectl apply -f deploy/kubernetes/secrets.yaml
kubectl apply -f deploy/kubernetes/api-deployment.yaml
kubectl apply -f deploy/kubernetes/worker-deployment.yaml
kubectl apply -f deploy/kubernetes/ingress.yaml
kubectl apply -f deploy/kubernetes/hpa.yaml
```

### API Deployment Spec

```yaml
spec:
  replicas: 2
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0

  template:
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        runAsGroup: 1000

      containers:
        - name: api
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: 500m
              memory: 512Mi

          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 30

          readinessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10

          securityContext:
            allowPrivilegeEscalation: false
            readOnlyRootFilesystem: true
            capabilities:
              drop: [ALL]

      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 100
              podAffinityTerm:
                labelSelector:
                  matchLabels:
                    app: api
                topologyKey: kubernetes.io/hostname
```

### Worker Deployment Spec

```yaml
spec:
  replicas: 2

  template:
    spec:
      containers:
        - name: worker
          resources:
            requests:
              cpu: 200m
              memory: 256Mi
            limits:
              cpu: 1000m
              memory: 1Gi

          livenessProbe:
            exec:
              command: ["/app/worker", "health"]
            initialDelaySeconds: 10
            periodSeconds: 30
```

### ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: ai-orchestration-config
  namespace: ai-orchestration
data:
  DATABASE_URL: postgres://user:pass@postgres:5432/ai_orchestration
  REDIS_URL: redis://redis:6379
  KEYCLOAK_URL: http://keycloak:8080
  KEYCLOAK_REALM: ai-orchestration
  AUTH_ENABLED: "true"
  TELEMETRY_ENABLED: "true"
  ENVIRONMENT: production
```

### Secrets

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: ai-orchestration-secrets
  namespace: ai-orchestration
type: Opaque
data:
  OPENAI_API_KEY: <base64>
  ANTHROPIC_API_KEY: <base64>
  DATABASE_PASSWORD: <base64>
```

### HPA

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: api-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: api
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
```

### Ingress

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ai-orchestration
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  rules:
    - host: api.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: api
                port:
                  number: 80
```

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
# Check pod status
kubectl get pods -n ai-orchestration

# View logs
kubectl logs -f deployment/api -n ai-orchestration

# Exec into pod
kubectl exec -it deployment/api -n ai-orchestration -- sh

# Check DB connection
kubectl exec -it deployment/api -n ai-orchestration -- \
  psql $DATABASE_URL -c "SELECT 1"

# Check Redis
kubectl exec -it deployment/api -n ai-orchestration -- \
  redis-cli -u $REDIS_URL PING
```

---

## Scaling Considerations

### API

- Stateless, scale horizontally
- Use HPA with CPU target 70%
- Consider memory-based scaling for large payloads

### Worker

- Scale based on queue depth
- Each worker processes jobs sequentially
- More workers = more parallel job execution

### Database

- Connection pooling essential
- Read replicas for read-heavy workloads
- Consider PgBouncer for connection management

### Redis

- Cluster mode for high availability
- Separate instances for cache vs queue (optional)
