# AI Orchestration - Document Index

> **AI-Driven Development**: このプロジェクトはすべてAIエージェントが実装・保守します。
> 新しいセッション開始時は必ず [CLAUDE.md](../CLAUDE.md) を最初に読んでください。

## Session Start Checklist

```
1. [ ] ../CLAUDE.md を読む（プロジェクト概要）
2. [ ] このファイルで関連ドキュメントを特定
3. [ ] 作業対象のドキュメントを読む
4. [ ] 既存コードパターンを確認
```

---

## Technical Documentation

| Document | Purpose | When to Read |
|----------|---------|--------------|
| [BACKEND.md](./BACKEND.md) | Go backend structure, interfaces, patterns | Modifying backend code |
| [FRONTEND.md](./FRONTEND.md) | Nuxt/Vue structure, composables, components | Modifying frontend code |
| [API.md](./API.md) | REST endpoints, request/response schemas | API integration, adding endpoints |
| [DATABASE.md](./DATABASE.md) | Schema, queries | Database operations |
| [DEPLOYMENT.md](./DEPLOYMENT.md) | Docker, Kubernetes, environment config | DevOps, deployment |
| [BLOCK_REGISTRY.md](./BLOCK_REGISTRY.md) | Block definitions, error codes | **新規ブロック追加時** |
| [INTEGRATIONS.md](./INTEGRATIONS.md) | 外部サービス連携一覧 | 連携ブロック追加・利用時 |

## Development Rules

作業種類に応じて必要なルールを参照:

| Rule Document | Purpose | When to Read |
|---------------|---------|--------------|
| [WORKFLOW_RULES](./rules/WORKFLOW_RULES.md) | 開発ワークフロー全般 | すべての開発作業 |
| [GIT_RULES](./rules/GIT_RULES.md) | コミット、PR、コンフリクト解消 | コミット・PR作成時 |
| [TESTING_RULES](./rules/TESTING_RULES.md) | テスト作成・実行 | テスト作成・実行時 |
| [DOCUMENTATION_SYNC](./rules/DOCUMENTATION_SYNC.md) | ドキュメント同期 | ドキュメント更新時 |
| [CODEX_REVIEW](./rules/CODEX_REVIEW.md) | PRレビューフロー | PR push後 |
| [DOCUMENTATION_RULES.md](./DOCUMENTATION_RULES.md) | ドキュメント作成ルール | 新規ドキュメント作成時 |

## Testing Documentation

| Document | Purpose | When to Read |
|----------|---------|--------------|
| [TEST_PLAN.md](./TEST_PLAN.md) | Test plan, coverage rules | Adding tests, coverage review |
| [BACKEND_TESTING.md](./BACKEND_TESTING.md) | Go backend testing patterns | Backend test implementation |
| [frontend/docs/TESTING.md](../frontend/docs/TESTING.md) | Frontend testing rules | Frontend code changes |

---

## Architecture Designs

| Design | Description | Status | Document |
|--------|-------------|--------|----------|
| Unified Block Model | ブロック実行の統一モデル | ✅ 実装済み | [UNIFIED_BLOCK_MODEL.md](./designs/UNIFIED_BLOCK_MODEL.md) |
| Block Config Improvement | ブロック設定UI改善 | 📋 設計中 | [BLOCK_CONFIG_IMPROVEMENT.md](./designs/BLOCK_CONFIG_IMPROVEMENT.md) |

## Feature Implementation Plans

| Phase | Feature | Status | Plan Document |
|-------|---------|--------|---------------|
| 6 | Guardrails | 📋 未実装 | [PHASE6_GUARDRAILS.md](./plans/PHASE6_GUARDRAILS.md) |
| 7 | Evaluator | 📋 未実装 | [PHASE7_EVALUATOR.md](./plans/PHASE7_EVALUATOR.md) |
| 8 | Variables System | 📋 未実装 | [PHASE8_VARIABLES.md](./plans/PHASE8_VARIABLES.md) |
| 9 | Cost Tracking | 📋 未実装 | [PHASE9_COST_TRACKING.md](./plans/PHASE9_COST_TRACKING.md) |
| 10 | Copilot | 📋 未実装 | [PHASE10_COPILOT.md](./plans/PHASE10_COPILOT.md) |

**推奨実装順序**: Phase 8 → 9 → 6 → 7 → 10

---

## System Overview

```
Architecture: Clean Architecture (Handler -> Usecase -> Domain -> Repository)
Tenancy: Multi-tenant with tenant_id isolation
Auth: Keycloak OIDC (JWT)
Queue: Redis-based job queue
Tracing: OpenTelemetry -> Jaeger
```

## Core Concepts (Quick Reference)

### Workflow States

```
draft -> published (immutable)
```

### Run States

```
pending -> running -> completed | failed | cancelled
```

### Step Types

詳細は [BACKEND.md](./BACKEND.md#domain-models) を参照。

| Type | Purpose |
|------|---------|
| `start` | Entry point |
| `llm` | LLM API call |
| `tool` | Adapter execution |
| `condition` | Branch routing (2-way) |
| `switch` | Multi-branch routing |
| `map` | Array parallel/sequential |
| `join` | Merge branches |
| `subflow` | Nested workflow |
| `loop` | Iteration |
| `filter` | Filter items |
| `log` | Debug logging |

### Adapters

詳細は [BACKEND.md](./BACKEND.md#adapter-implementations) を参照。

| ID | Purpose |
|----|---------|
| `mock` | Testing |
| `openai` | GPT API |
| `anthropic` | Claude API |
| `http` | Generic HTTP |

---

## Common Operations

### Add New Block / Integration

**Use slash command**: `/add-block`

または [.claude/commands/add-block.md](../.claude/commands/add-block.md) を参照。

### Add New API Endpoint

1. Add handler in `backend/internal/handler/`
2. Add route in `cmd/api/main.go`
3. Add usecase if new business logic needed
4. Update `docs/API.md` and `docs/openapi.yaml`

### Add New Step Type

1. Define in `backend/internal/domain/step.go`
2. Add execution logic in `backend/internal/engine/executor.go`
3. Update frontend step config UI
4. Update `docs/BACKEND.md`

### Fix a Bug

**Use slash command**: `/fix-bug`

---

## Test Commands

```bash
# Backend
cd backend && go test ./...
cd backend && go test ./tests/e2e/... -v

# Frontend (REQUIRED before commit)
cd frontend && npm run check
```

---

## URLs (Development)

| Service | URL |
|---------|-----|
| API | http://localhost:8080 |
| Frontend | http://localhost:3000 |
| Keycloak Admin | http://localhost:8180/admin |
| Jaeger UI | http://localhost:16686 |
