# AI Orchestration

Multi-tenant SaaS for DAG workflows with LLM integrations.

## AI-Driven Development

このプロジェクトはすべてAIエージェントが実装・保守する。人間の役割は要件定義・レビュー・承認のみ。

**自律的に進める**: 実装方針が明確、既存パターンに従う変更、テストが通る修正、ドキュメント更新、リファクタリング、バグ修正

**人間に確認する**: 要件が曖昧、破壊的変更（API/DBスキーマ）、セキュリティ判断、外部サービス課金への影響

---

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

# Database
make db-reset         # Drop, apply schema, seed
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

---

## Documentation

詳細は [docs/INDEX.md](docs/INDEX.md) を参照。

| Document | Purpose |
|----------|---------|
| BACKEND.md | Go code structure |
| FRONTEND.md | Vue/Nuxt structure |
| API.md | REST endpoints |
| DATABASE.md | Schema, queries |
| BLOCK_REGISTRY.md | Block definitions |

---

## Workflow Rules

### コード変更完了後

Claudeは以下を自動的に実行する（スキルが自動選択される）:

1. **self-review** - コード自己検証、テスト実行
2. **update-docs** - 必要に応じてドキュメント更新
3. **create-pr** - PR作成

### PR作成後

1. **review-pr** - CIとCodexレビュー結果を確認、必要に応じて修正

### 禁止事項

| 禁止 | 理由 |
|------|------|
| ローカルCI未実行でpush | CIの失敗を防ぐ |
| レビュー結果を待たずにマージ | 品質担保 |
| REQUEST_CHANGESを無視 | 指摘は全て対応必須 |
