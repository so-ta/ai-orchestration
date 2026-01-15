# AI Orchestration

Multi-tenant SaaS for designing, executing, and monitoring DAG workflows with LLM and tool integrations.

---

## AI-Driven Development Project

**このプロジェクトは人間によるコーディングを一切行わず、すべてをAIエージェントが実装・保守するプロジェクトです。**

| 原則 | 説明 |
|------|------|
| **完全AI実装** | 設計・実装・テスト・ドキュメント作成すべてをAIが担当 |
| **人間の役割** | 要件定義、レビュー、承認のみ（作業中の確認は不要） |
| **コンテキスト継続** | 後続エージェントがコンテキストを見失わないよう文書化必須 |
| **自律的作業** | 人間への確認・承認要求は最小限に。判断できることは自分で判断して進める |

### 人間への確認が不要なケース

AIエージェントは以下の場合、人間に確認を求めず自律的に作業を進めること：

- 実装方針が明確な場合
- 既存パターンに従う変更
- テストが通る修正
- ドキュメントの更新
- リファクタリング
- バグ修正

### 人間への確認が必要なケース

以下の場合のみ、人間に確認を求める：

- 要件が曖昧で複数の解釈が可能な場合
- 破壊的変更（API変更、DB スキーマ変更など）
- セキュリティに関わる判断
- 外部サービスの課金に影響する変更

### Session Start Checklist

```
1. [ ] CLAUDE.md を読む（このファイル）
2. [ ] docs/INDEX.md で関連ドキュメントを特定
3. [ ] 作業対象のドキュメントを読む
4. [ ] 既存の実装パターンを確認
5. [ ] テスト・検証手順を確認
```

---

## Quick Reference

| Item | Value |
|------|-------|
| Backend | Go 1.22+ |
| Frontend | Vue 3 + Nuxt 3 |
| Database | PostgreSQL 16 |
| Cache/Queue | Redis 7 |
| Auth | Keycloak 24 (OIDC) |
| Tracing | OpenTelemetry + Jaeger |

## Documentation Index

| Document | Purpose | Path |
|----------|---------|------|
| **INDEX** | **Document navigation (MUST READ)** | [docs/INDEX.md](docs/INDEX.md) |
| BACKEND | Go code structure, interfaces | [docs/BACKEND.md](docs/BACKEND.md) |
| FRONTEND | Vue/Nuxt structure, composables | [docs/FRONTEND.md](docs/FRONTEND.md) |
| API | REST endpoints, schemas | [docs/API.md](docs/API.md) |
| DATABASE | Schema, queries | [docs/DATABASE.md](docs/DATABASE.md) |
| BLOCK_REGISTRY | Block definitions | [docs/BLOCK_REGISTRY.md](docs/BLOCK_REGISTRY.md) |
| UNIFIED_BLOCK_MODEL | Block architecture | [docs/designs/UNIFIED_BLOCK_MODEL.md](docs/designs/UNIFIED_BLOCK_MODEL.md) |

**詳細ルール（作業種類に応じて参照）:**

| Rule Document | When to Read |
|---------------|--------------|
| [WORKFLOW_RULES](docs/rules/WORKFLOW_RULES.md) | 開発作業全般 |
| [GIT_RULES](docs/rules/GIT_RULES.md) | コミット、PR、コンフリクト解消 |
| [TESTING_RULES](docs/rules/TESTING_RULES.md) | テスト作成・実行 |
| [DOCUMENTATION_SYNC](docs/rules/DOCUMENTATION_SYNC.md) | ドキュメント更新 |
| [CODEX_REVIEW](docs/rules/CODEX_REVIEW.md) | PR作成後のレビューフロー |

---

## Directory Structure

```
ai-orchestration/
├── CLAUDE.md                 # This file (entry point)
├── backend/
│   ├── cmd/api/              # API server entry
│   ├── cmd/worker/           # Worker entry
│   ├── internal/             # Domain, usecase, handler, repository, adapter, engine
│   ├── schema/               # DB schema (schema.sql, seed.sql)
│   └── tests/e2e/            # Integration tests
├── frontend/
│   ├── pages/                # Nuxt pages
│   ├── components/dag-editor/# DAG visual editor
│   └── composables/          # Vue composables
├── docs/                     # Documentation
│   ├── INDEX.md              # Navigation
│   └── rules/                # Development rules
└── .claude/commands/         # Custom slash commands
```

---

## Commands

### Development

```bash
# Full Docker
docker compose up -d

# Hot Reload (recommended)
make dev              # All services (tmux)
make dev-middleware   # PostgreSQL, Redis, Keycloak, Jaeger
make dev-api          # API with hot reload
make dev-worker       # Worker with hot reload
make dev-frontend     # Frontend with hot reload

# Stop
make stop
```

### Tests

```bash
# Backend
cd backend && go test ./...
cd backend && go test ./tests/e2e/... -v

# Frontend (REQUIRED before commit)
cd frontend && npm run check   # typecheck + lint + test
```

### Database

```bash
make db-reset   # Drop, apply schema, seed
make db-apply   # Apply schema only
make db-seed    # Seed data only
```

---

## URLs (Development)

| Service | URL |
|---------|-----|
| API | http://localhost:8080 |
| Frontend | http://localhost:3000 |
| Keycloak | http://localhost:8180 |
| Jaeger | http://localhost:16686 |

## Test Credentials

| User | Password | Role |
|------|----------|------|
| admin@example.com | admin123 | tenant_admin |
| builder@example.com | builder123 | builder |

Default tenant ID: `00000000-0000-0000-0000-000000000001`

---

## Essential Rules (MUST FOLLOW)

### 0. CODEX REVIEW WORKFLOW（最重要・絶対遵守）

**作業完了後は、必ず[CODEX_REVIEW](docs/rules/CODEX_REVIEW.md)のワークフローに従うプルリクエストを作成すること。**

```
⚠️ これは絶対に守るべきルールです。違反は許可されません。

1. 作業完了後、git commit
2. ローカルCIを実行して全てパスすることを確認（必須）
   - Backend変更: cd backend && go test ./...
   - Frontend変更: cd frontend && npm run check
3. ローカルCI通過後、git push してプルリクエストを作成
4. プルリクエスト後は必ず Codex Review + CI の完了を待つ
5. REQUEST_CHANGES があれば全て修正して再push（手順2から）。変更外の場合は当該箇所を確認して修正。レビューに「対応する必要がない」場合は、その旨をIssueのコメントに記載して7へ
6. 修正後したら再度pushを行い 4 へ
7. プルリクエストをMergeする
```

| 禁止事項 | 理由 |
|----------|------|
| ローカルCI未実行でpush | CIの失敗を事前に防ぐため |
| レビュー結果を待たずにマージ | 品質担保のため |
| REQUEST_CHANGESを無視してマージ | 指摘事項は全て対応必須 |
| CI失敗を無視してマージ | テスト・ビルド通過必須 |

---

### 1. Always Read Docs First

```
Before modifying code:
1. Read docs/INDEX.md
2. Read relevant document for your task
3. Understand existing patterns
```

### 2. Always Run Tests

```bash
# Backend changes
cd backend && go test ./...

# Frontend changes
cd frontend && npm run check

# Service restart (Docker)
docker compose restart api worker
```

### 3. Always Update Documentation

| Change Type | Update |
|-------------|--------|
| New block/integration | BLOCK_REGISTRY.md |
| DB schema | DATABASE.md |
| API endpoint | API.md, openapi.yaml |
| Backend structure | BACKEND.md |
| Frontend structure | FRONTEND.md |

---

## Custom Slash Commands

作業種類に応じたコマンドを使用:

| Command | Purpose |
|---------|---------|
| `/add-block` | 新規ブロック追加のワークフロー |
| `/fix-bug` | バグ修正のワークフロー |
| `/add-feature` | 新機能追加のワークフロー |
| `/review-pr` | PRレビュー待機フロー |

---

## Environment Variables

| Variable | Service | Description |
|----------|---------|-------------|
| `DATABASE_URL` | api, worker | PostgreSQL |
| `REDIS_URL` | api, worker | Redis |
| `AUTH_ENABLED` | api | Enable JWT (default: false) |
| `OPENAI_API_KEY` | worker | OpenAI key |
| `ANTHROPIC_API_KEY` | worker | Anthropic key |

---

## API Quick Test

```bash
# Create workflow
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"name": "Test"}'

# Execute workflow
curl -X POST "http://localhost:8080/api/v1/workflows/{id}/runs" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"input": {}, "mode": "test"}'
```

---

## Decision Record Template

重要な技術的決定は以下の形式で記録：

```markdown
### Decision: [決定事項]
- **Date**: YYYY-MM-DD
- **Context**: 背景・状況
- **Decision**: 選択した内容
- **Rationale**: 理由
```

---

## Implementation Status

All phases complete (Phase 1-8):
- Workflow CRUD, Steps, Edges, DAG execution engine
- Adapters (Mock, OpenAI, Anthropic, HTTP)
- Schedules, Webhooks, Keycloak OIDC auth
- OpenTelemetry tracing, E2E tests
