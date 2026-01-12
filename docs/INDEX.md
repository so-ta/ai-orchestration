# AI Orchestration - Document Index

> **AI-Driven Development**: このプロジェクトはすべてAIエージェントが実装・保守します。
> 新しいセッション開始時は必ず [CLAUDE.md](../CLAUDE.md) を最初に読んでください。

## Session Start Checklist

```
1. [ ] ../CLAUDE.md を読む（プロジェクト概要・ルール）
2. [ ] このファイルで関連ドキュメントを特定
3. [ ] 作業対象のドキュメントを読む
4. [ ] 既存コードパターンを確認
5. [ ] テスト手順を確認（frontend/docs/TESTING.md）
```

## Quick Reference

| Document | Purpose | When to Read |
|----------|---------|--------------|
| [BACKEND.md](./BACKEND.md) | Go backend structure, interfaces, patterns | Modifying backend code |
| [FRONTEND.md](./FRONTEND.md) | Nuxt/Vue structure, composables, components | Modifying frontend code |
| [FRONTEND.md#dag-editor](./FRONTEND.md#dag-editor-componentdag-editor) | DAG editor collision detection, resize logic | **Modifying DAG editor** |
| [API.md](./API.md) | REST endpoints, request/response schemas | API integration, adding endpoints |
| [DATABASE.md](./DATABASE.md) | Schema, migrations, query patterns | Database operations |
| [DEPLOYMENT.md](./DEPLOYMENT.md) | Docker, Kubernetes, environment config | DevOps, deployment |
| [DOCUMENTATION_RULES.md](./DOCUMENTATION_RULES.md) | Doc format, MECE, templates | Creating/updating documentation |
| [TESTING.md](../frontend/docs/TESTING.md) | Frontend testing rules, Vitest | Frontend code changes |

## System Overview

```
Architecture: Clean Architecture (Handler -> Usecase -> Domain -> Repository)
Tenancy: Multi-tenant with tenant_id isolation
Auth: Keycloak OIDC (JWT)
Queue: Redis-based job queue
Tracing: OpenTelemetry -> Jaeger
```

## Core Concepts

### Workflow
- DAG-based execution graph
- States: `draft` -> `published` (immutable)
- Version tracked for audit

### Step Types
| Type | Description | Config Key Fields |
|------|-------------|-------------------|
| `start` | Entry point | - |
| `llm` | LLM API call | `provider`, `model`, `prompt` |
| `tool` | Adapter execution | `adapter_id`, adapter-specific |
| `condition` | Branch routing (2-way) | `expression` |
| `switch` | Multi-branch routing | `cases`, `default` |
| `map` | Array parallel/sequential | `input_path`, `parallel` |
| `join` | Merge branches | - |
| `subflow` | Nested workflow | `workflow_id` |
| `loop` | Iteration | `loop_type`, `count`, `condition` |
| `filter` | Filter items | `expression` |
| `log` | Debug logging | `message`, `level` |

### Run States
```
pending -> running -> completed
                  -> failed
                  -> cancelled
```

### Adapters
| ID | File | Purpose |
|----|------|---------|
| `mock` | `adapter/mock.go` | Testing |
| `openai` | `adapter/openai.go` | GPT API |
| `anthropic` | `adapter/anthropic.go` | Claude API |
| `http` | `adapter/http.go` | Generic HTTP |

## File Path Conventions

```
backend/
  cmd/api/main.go          # API entrypoint
  cmd/worker/main.go       # Worker entrypoint
  internal/
    domain/                # Entities (Workflow, Step, Run, Edge)
    usecase/               # Business logic
    handler/               # HTTP handlers
    repository/postgres/   # DB operations
    adapter/               # External integrations
    engine/                # DAG executor
    middleware/            # Auth middleware
  pkg/
    database/              # DB connection
    redis/                 # Redis client
    telemetry/             # OpenTelemetry

frontend/
  pages/                   # Nuxt pages
  components/dag-editor/   # DAG visual editor
  composables/             # Vue composables (useAuth, useApi)
  plugins/auth.client.ts   # Keycloak init
```

## Common Operations

### Add New Adapter
1. Create `backend/internal/adapter/{name}.go`
2. Implement `Adapter` interface
3. Register in `adapter/registry.go`
4. Add test in `adapter/{name}_test.go`

### Add New API Endpoint
1. Add handler in `backend/internal/handler/`
2. Add route in `cmd/api/main.go`
3. Add usecase if new business logic needed
4. Update `docs/openapi.yaml`

### Add New Step Type
1. Define in `backend/internal/domain/step.go`
2. Add execution logic in `backend/internal/engine/executor.go`
3. Update frontend step config UI

### Add Database Migration
1. Create SQL in `backend/migrations/`
2. Run: `docker compose exec api migrate -path /migrations -database "$DB_URL" up`

## Test Commands

```bash
# Backend tests
docker compose exec api go test ./...
docker compose exec api go test ./internal/adapter/... -v
docker compose exec api go test ./tests/e2e/... -v

# Frontend tests (REQUIRED before commit)
cd frontend && npm run check       # All checks
cd frontend && npm run typecheck   # TypeScript only
cd frontend && npm run test:run    # Unit tests only
```

## Environment Variables

| Variable | Service | Default | Description |
|----------|---------|---------|-------------|
| `DATABASE_URL` | api, worker | - | PostgreSQL connection |
| `REDIS_URL` | api, worker | - | Redis connection |
| `AUTH_ENABLED` | api | `false` | Enable JWT validation |
| `KEYCLOAK_URL` | api | - | Keycloak base URL |
| `TELEMETRY_ENABLED` | api, worker | `false` | Enable OpenTelemetry |
| `OPENAI_API_KEY` | worker | - | OpenAI API key |
| `ANTHROPIC_API_KEY` | worker | - | Anthropic API key |

## URLs (Development)

| Service | URL |
|---------|-----|
| API | http://localhost:8080 |
| Frontend | http://localhost:3000 |
| Keycloak Admin | http://localhost:8180/admin |
| Jaeger UI | http://localhost:16686 |
