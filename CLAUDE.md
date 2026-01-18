# AI Orchestration

Multi-tenant SaaS for DAG workflows with LLM integrations.

## AI-Driven Development

このプロジェクトはすべてAIエージェントが実装・保守する。

## Tech Stack

| Item | Value |
|------|-------|
| Backend | Go 1.22+ |
| Frontend | Vue 3 + Nuxt 3 |
| Database | PostgreSQL 16 |
| Cache/Queue | Redis 7 |

## Commands

```bash
# Development
make dev-middleware   # DB, Redis, Keycloak, Jaeger
make dev-api          # API with hot reload
make dev-frontend     # Frontend with hot reload

# Tests (REQUIRED before commit)
cd backend && go test ./...
cd frontend && npm run check
```

## URLs

| Service | URL |
|---------|-----|
| API | http://localhost:8080 |
| Frontend | http://localhost:3000 |
| Keycloak | http://localhost:8180 |

## Test Credentials

| User | Password |
|------|----------|
| admin@example.com | admin123 |
| builder@example.com | builder123 |

Tenant ID: `00000000-0000-0000-0000-000000000001`

## Documentation

詳細は [docs/INDEX.md](docs/INDEX.md) を参照。
