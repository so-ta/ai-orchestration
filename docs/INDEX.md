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
| [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) | エラー対処法 | エラー発生時 |

## Development Rules

作業種類に応じて必要なルールを参照:

| Rule Document | Purpose | When to Read |
|---------------|---------|--------------|
| [WORKFLOW_RULES](./rules/WORKFLOW_RULES.md) | 開発ワークフロー全般（Why/過去の失敗例あり） | すべての開発作業 |
| [GIT_RULES](./rules/GIT_RULES.md) | コミット、PR、コンフリクト解消 | コミット・PR作成時 |
| [TESTING](./TESTING.md) | テスト作成・実行（統合ガイド） | テスト作成・実行時 |
| [DOCUMENTATION](./DOCUMENTATION.md) | ドキュメント作成・同期（統合ガイド） | ドキュメント更新時 |
| [CODEX_REVIEW](./rules/CODEX_REVIEW.md) | PRレビューフロー | PR push後 |

## Testing Documentation

| Document | Purpose | When to Read |
|----------|---------|--------------|
| [TESTING.md](./TESTING.md) | テスト統合ガイド（優先度マトリックス含む） | テスト作成・実行時 |
| [BACKEND_TESTING.md](./BACKEND_TESTING.md) | Go backend testing patterns | Backend test implementation |
| [frontend/docs/TESTING.md](../frontend/docs/TESTING.md) | Frontend testing rules | Frontend code changes |

---

## Architecture Designs

| Design | Description | Status | Document |
|--------|-------------|--------|----------|
| Unified Block Model | ブロック実行の統一モデル | ✅ 実装済み | [UNIFIED_BLOCK_MODEL.md](./designs/UNIFIED_BLOCK_MODEL.md) |
| Block Config Improvement | ブロック設定UI改善 | 📋 設計中 | [BLOCK_CONFIG_IMPROVEMENT.md](./designs/BLOCK_CONFIG_IMPROVEMENT.md) |

## Implementation Status (Single Source of Truth)

> この表が実装状態の正（Source of Truth）です。各計画書内の記載は補助情報です。

| Phase | Feature | Status | Document | Related PR/Commit |
|-------|---------|--------|----------|-------------------|
| Core | Unified Block Model | ✅ Complete | [UNIFIED_BLOCK_MODEL.md](./designs/UNIFIED_BLOCK_MODEL.md) | - |
| Core | Block Group Redesign | ✅ Complete | [BLOCK_GROUP_REDESIGN.md](./designs/BLOCK_GROUP_REDESIGN.md) | - |
| Core | Block Config Improvement | 🚧 Phase 3 Done | [BLOCK_CONFIG_IMPROVEMENT.md](./designs/BLOCK_CONFIG_IMPROVEMENT.md) | - |
| Core | Rich View Output | ✅ Complete | [RICH_VIEW_OUTPUT.md](./designs/RICH_VIEW_OUTPUT.md) | - |
| 6 | Guardrails | 📋 Not Started | [PHASE6_GUARDRAILS.md](./plans/PHASE6_GUARDRAILS.md) | - |
| 7 | Evaluator | 📋 Not Started | [PHASE7_EVALUATOR.md](./plans/PHASE7_EVALUATOR.md) | - |
| 8 | Variables System | ✅ Complete | [PHASE8_VARIABLES.md](./plans/PHASE8_VARIABLES.md) | - |
| 9 | Cost Tracking | ✅ Complete | [PHASE9_COST_TRACKING.md](./plans/PHASE9_COST_TRACKING.md) | - |
| 10 | Copilot | 📋 Not Started | [PHASE10_COPILOT.md](./plans/PHASE10_COPILOT.md) | - |
| Special | RAG Implementation | 🚧 In Progress | [RAG_IMPLEMENTATION_PLAN.md](./plans/RAG_IMPLEMENTATION_PLAN.md) | - |

**ステータス凡例**:
- ✅ Complete: 実装完了
- 🚧 In Progress / Phase X Done: 実装中（フェーズ完了）
- 📋 Not Started: 未実装

**推奨実装順序**: Phase 6 → 7 → 10

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
