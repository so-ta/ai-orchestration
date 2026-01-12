# AI Orchestration

Multi-tenant SaaS for designing, executing, and monitoring DAG workflows with LLM and tool integrations.

---

## AI-Driven Development Project

**このプロジェクトは人間によるコーディングを一切行わず、すべてをAIエージェントが実装・保守するプロジェクトです。**

### 基本原則

| 原則 | 説明 |
|------|------|
| **完全AI実装** | 設計・実装・テスト・ドキュメント作成すべてをAIが担当 |
| **人間の役割** | 要件定義、レビュー、承認のみ |
| **コンテキスト継続** | 後続エージェントがコンテキストを見失わないよう文書化必須 |

### AIフレンドリードキュメント要件

後続のAIエージェントが即座にコンテキストを把握できるよう、以下を遵守：

1. **明示的な記述**
   - 暗黙知を排除し、すべてを文書化
   - 「なぜ」その設計・実装にしたかを記録
   - 制約条件や前提条件を明記

2. **構造化された情報**
   - テーブル形式での情報整理を優先
   - コードブロックでの具体例提示
   - 階層的な見出し構造

3. **参照可能性**
   - ファイルパスは絶対パスまたはプロジェクトルートからの相対パス
   - 関連ドキュメントへのリンクを明記
   - 検索可能なキーワードを含める

4. **最新性の維持**
   - コード変更時は必ずドキュメント更新
   - 古い情報は削除または更新日を明記
   - バージョン管理との整合性を保つ

### コンテキスト引き継ぎチェックリスト

新しいAIエージェントセッション開始時：

```
1. [ ] CLAUDE.md を読む（このファイル）
2. [ ] docs/INDEX.md で関連ドキュメントを特定
3. [ ] 作業対象のドキュメントを読む
4. [ ] 既存の実装パターンを確認
5. [ ] テスト・検証手順を確認
```

### 意思決定の記録

重要な技術的決定は以下の形式で記録：

```markdown
### Decision: [決定事項]
- **Date**: YYYY-MM-DD
- **Context**: 背景・状況
- **Options**: 検討した選択肢
- **Decision**: 選択した内容
- **Rationale**: 理由
- **Consequences**: 影響・結果
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
| INDEX | Document navigation | [docs/INDEX.md](docs/INDEX.md) |
| BACKEND | Go code structure, interfaces | [docs/BACKEND.md](docs/BACKEND.md) |
| FRONTEND | Vue/Nuxt structure, composables | [docs/FRONTEND.md](docs/FRONTEND.md) |
| API | REST endpoints, schemas | [docs/API.md](docs/API.md) |
| DATABASE | Schema, queries | [docs/DATABASE.md](docs/DATABASE.md) |
| DEPLOYMENT | Docker, K8s, config | [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) |
| DOCUMENTATION_RULES | Doc format, MECE rules | [docs/DOCUMENTATION_RULES.md](docs/DOCUMENTATION_RULES.md) |
| TESTING | Frontend testing rules | [frontend/docs/TESTING.md](frontend/docs/TESTING.md) |
| OpenAPI | Machine-readable spec | [docs/openapi.yaml](docs/openapi.yaml) |
| **UNIFIED_BLOCK_MODEL** | **Block architecture (MUST READ for integrations)** | [docs/designs/UNIFIED_BLOCK_MODEL.md](docs/designs/UNIFIED_BLOCK_MODEL.md) |
| BLOCK_REGISTRY | Block definitions, error codes | [docs/BLOCK_REGISTRY.md](docs/BLOCK_REGISTRY.md) |

**Read these docs before modifying related code.**

## Directory Structure

```
ai-orchestration/
├── CLAUDE.md                 # This file
├── docker-compose.yml        # Dev environment
├── backend/
│   ├── cmd/api/              # API server entry
│   ├── cmd/worker/           # Worker entry
│   ├── internal/
│   │   ├── domain/           # Entities
│   │   ├── usecase/          # Business logic
│   │   ├── handler/          # HTTP handlers
│   │   ├── repository/       # DB operations
│   │   ├── adapter/          # External integrations
│   │   ├── engine/           # DAG executor
│   │   └── middleware/       # Auth
│   ├── pkg/                  # Shared packages
│   ├── migrations/           # SQL migrations
│   └── tests/e2e/            # Integration tests
├── frontend/
│   ├── pages/                # Nuxt pages
│   ├── components/dag-editor/# DAG visual editor
│   ├── composables/          # Vue composables
│   └── plugins/              # Keycloak init
├── deploy/kubernetes/        # K8s manifests
└── docs/                     # Documentation
```

## Commands

### Option A: Full Docker (すべてDockerで実行)

```bash
# Start all services
docker compose up -d

# Logs
docker compose logs -f api
docker compose logs -f worker

# Rebuild
docker compose up -d --build api worker
```

### Option B: Local Development with Hot Reload (推奨)

Makefileを使用してホットリロード開発環境を起動：

```bash
# 1. ミドルウェア起動 (PostgreSQL, Redis, Keycloak, Jaeger)
make dev-middleware

# 2. API起動 (ホットリロード) - ターミナル1
make dev-api

# 3. Worker起動 (ホットリロード) - ターミナル2
make dev-worker

# 4. Frontend起動 (ホットリロード) - ターミナル3
make dev-frontend

# tmuxがインストールされている場合、すべてを一度に起動
make dev

# すべて停止
make stop
```

**利用可能なMakeターゲット:**

| コマンド | 説明 |
|---------|------|
| `make help` | ヘルプ表示 |
| `make dev` | 全サービス起動（tmux使用） |
| `make dev-middleware` | ミドルウェアのみ起動 |
| `make dev-api` | APIをホットリロードで起動 |
| `make dev-worker` | Workerをホットリロードで起動 |
| `make dev-frontend` | Frontendをホットリロードで起動 |
| `make stop` | 全サービス停止 |
| `make test` | 全テスト実行 |

**手動コマンド（参考）:**

```bash
# Middleware
docker compose -f docker-compose.middleware.yml up -d

# Backend API with hot reload (from project root)
cd backend && air -c .air.toml

# Worker with hot reload (from project root)
cd backend && air -c .air.worker.toml

# Frontend (from frontend/ directory)
npm run dev
```

### Option C: Hybrid Development (フロントエンドローカル + APIはDocker)

ローカルGoのバージョン不一致がある場合、この方法を使用：

```bash
# 1. ミドルウェアとAPI/Workerを Docker で起動
docker compose up -d postgres redis keycloak jaeger api worker

# 2. フロントエンドをローカルで起動
cd frontend && npm run dev
```

**アクセスURL:**
- Frontend: http://localhost:3000
- API: http://localhost:8080
- Keycloak: http://localhost:8180
- Jaeger: http://localhost:16686

**注意:** ローカルGoでバージョン不一致エラーが発生する場合（`compile: version "go1.x.x" does not match go tool version "go1.y.y"`）、APIはDockerで実行してください。

### Tests

```bash
# Backend Test (local)
cd backend && go test ./...
cd backend && go test ./tests/e2e/... -v

# Backend Test (Docker)
docker compose exec api go test ./...

# Frontend Test
cd frontend && npm run check       # All checks (typecheck + lint + test)
cd frontend && npm run typecheck   # TypeScript only
cd frontend && npm run test:run    # Unit tests only
```

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

## Core Concepts

### Step Types

| Type | Purpose | Config |
|------|---------|--------|
| `start` | Entry point | - |
| `llm` | LLM call | `provider`, `model`, `prompt` |
| `tool` | Adapter exec | `adapter_id` |
| `condition` | Branch (2-way) | `expression` |
| `switch` | Multi-branch routing | `cases`, `default` |
| `map` | Array parallel | `input_path`, `parallel` |
| `join` | Merge | - |
| `subflow` | Nested workflow | `workflow_id` |
| `loop` | Iteration | `loop_type`, `count`, `condition` |
| `wait` | Delay/Timer | `duration_ms`, `until` |
| `function` | Custom code | `code`, `language` |
| `router` | AI routing | `routes`, `provider`, `model` |
| `human_in_loop` | Approval gate | `instructions`, `timeout_hours` |
| `filter` | Filter items | `expression` |
| `split` | Split into batches | `batch_size` |
| `aggregate` | Aggregate data | `mode` |
| `error` | Stop with error | `message` |
| `note` | Documentation | `content` |
| `log` | Debug logging | `message`, `level` |

### Adapters

| ID | File | Purpose |
|----|------|---------|
| `mock` | adapter/mock.go | Testing |
| `openai` | adapter/openai.go | GPT API |
| `anthropic` | adapter/anthropic.go | Claude API |
| `http` | adapter/http.go | Generic HTTP |

### Run States

```
pending -> running -> completed | failed | cancelled
```

### Condition Expression Syntax

```
$.field == "value"     # equality
$.field != "value"     # inequality
$.field > 10           # numeric
$.nested.field         # nested path
$.field                # truthy
```

## Development Rules

### Workflow (REQUIRED)

1. Read relevant docs before code changes
2. Update docs when changing specs
3. Use TodoWrite for task tracking
4. Run tests after changes
5. **Restart services after code changes** (see below)

### Service Restart After Code Changes (REQUIRED)

**バックエンドコード変更後は、必ず関連サービスを再起動すること。**

| 変更対象 | 再起動コマンド |
|----------|---------------|
| `backend/cmd/api/` | `docker compose restart api` |
| `backend/cmd/worker/` | `docker compose restart worker` |
| `backend/internal/` | `docker compose restart api worker` |
| 両方 | `docker compose restart api worker` |

```bash
# Docker環境の場合
docker compose restart api worker

# ローカル開発（air使用）の場合は自動リロードされる
```

**重要:**
- コード変更だけでは反映されない（特にDocker環境）
- 変更を検証する前に必ず再起動を実行
- フロントエンドはホットリロードで自動反映されるため再起動不要

### Self-Documentation (REQUIRED)

**AI agents MUST maintain documentation autonomously.**

#### Before Any Code Change

```
1. Read docs/INDEX.md to find relevant document
2. Read the relevant document
3. Verify understanding matches implementation
```

#### When Documentation Missing

| Situation | Action |
|-----------|--------|
| No doc for area being modified | Create new doc following DOCUMENTATION_RULES.md |
| Existing doc incomplete | Update existing doc |
| Code contradicts doc | Fix code OR update doc (confirm intent first) |

#### After Code Change

```
1. Update relevant doc in docs/
2. If new feature/module: create docs/{FEATURE}.md
3. If new doc created: update docs/INDEX.md
4. Follow MECE principle (see docs/DOCUMENTATION_RULES.md)
```

#### Documentation Priority

1. **MUST document**: Public interfaces, API changes, config changes
2. **SHOULD document**: Internal architecture decisions, non-obvious patterns
3. **MAY skip**: Trivial implementation details (use code comments)

### Frontend Testing Workflow (REQUIRED)

**AIエージェントは以下のチェックを必ず実行すること：**

```bash
# Frontend directory
cd frontend

# 1. TypeScript check (MUST pass)
npm run typecheck

# 2. Lint check
npm run lint

# 3. Run tests
npm run test:run

# 4. Or run all checks at once
npm run check
```

| Check | Command | Required |
|-------|---------|----------|
| TypeScript | `npm run typecheck` | **必須** |
| ESLint | `npm run lint` | 推奨 |
| Unit Tests | `npm run test:run` | 推奨 |
| All Checks | `npm run check` | **完了前必須** |
| Docker Build | `docker compose build frontend` | **package.json変更時必須** |

**禁止事項：**
- TypeScriptエラーを無視してコードを完了とすること
- ブラウザ確認なしでUI変更を完了とすること
- 型エラーをキャストで回避すること（根本原因を修正）
- プラットフォーム固有パッケージ（`@rollup/rollup-darwin-*`等）をdependenciesに追加

**詳細は [frontend/docs/TESTING.md](frontend/docs/TESTING.md) を参照**

### DAG Editor Modification (REQUIRED)

**ワークフローエディタ（DAGエディタ）を修正する場合、必ず以下を確認すること：**

```
1. [ ] docs/FRONTEND.md の「Block Group Push Logic」セクションを読む
2. [ ] docs/FRONTEND.md の「Group Resize Logic」セクションを読む
3. [ ] Vue Flowの親子ノード関係（相対座標 vs 絶対座標）を理解する
4. [ ] 衝突判定ロジックの3ケース分類を理解する
```

**重要な注意点：**

| 領域 | 注意点 |
|------|--------|
| 座標系 | 子ノードは親からの相対座標。押出後は絶対座標に変換必須 |
| リサイズ | `onGroupResize`でリアルタイム位置補正必須（視覚的ジャンプ防止） |
| 衝突判定 | `fullyInside`, `fullyOutside`, `onBoundary`の3ケースで処理 |
| イベント順序 | `group:update` → `group:resize-complete`の順で発火必須 |
| ネスト | グループのネストは非対応（外側にスナップ） |

**関連ファイル：**
- `components/dag-editor/DagEditor.vue` - 衝突判定、リサイズハンドラ
- `pages/workflows/[id].vue` - イベントハンドラ、API永続化

### Bug Fix Flow

1. Write failing test reproducing bug
2. Verify test fails
3. Fix code
4. Verify test passes

### Task Type Decision (判断フローガイド)

作業開始前にタスクの種類と影響範囲を判別する。

**タスク種類の判別:**

| リクエスト内容 | 対応方針 |
|--------------|---------|
| バグ修正 | 再現テスト作成 → 修正 → テスト確認 |
| 新機能追加 | 既存パターン確認 → 実装 → テスト → ドキュメント |
| リファクタリング | 既存テスト確認 → リファクタ → 振る舞い不変を確認 |
| ドキュメント更新 | 関連コードを確認 → 更新 |
| 調査・質問 | コードを読んで回答（変更不要） |

**影響範囲の判別:**

| 変更対象 | 対応方針 |
|---------|---------|
| 単一ファイルのみ | 直接修正 |
| 複数ファイル（同一パッケージ） | パッケージ内で完結させる |
| 複数パッケージ | 影響範囲を全て確認してから着手 |
| 不明 | 影響範囲を調査してから着手 |

### Commit Message Convention (詳細)

**形式:**
```
<type>: <summary>

<body（任意）>
```

**typeの種類:**

| type | 用途 |
|------|------|
| `feat` | 新機能 |
| `fix` | バグ修正 |
| `refactor` | リファクタリング（機能変更なし） |
| `docs` | ドキュメントのみの変更 |
| `test` | テストの追加・修正 |
| `chore` | ビルドプロセス、補助ツールの変更 |

**summaryの書き方:**
- 日本語で記述
- 50文字以内を目安
- 動詞で始める（「追加」「修正」「変更」など）
- 句点（。）は付けない

**コミットのタイミング:**

| 作業状態 | 判断 |
|---------|------|
| 機能が完成した | コミット |
| テストがパスした | コミット |
| 作業途中だが区切りがよい | コミット |
| ビルドが通らない | **コミットしない** |
| テストが失敗している | **コミットしない** |

**コミット粒度:**

| 変更内容 | 粒度 |
|---------|------|
| 単一の目的（バグ修正、機能追加など） | 1コミット |
| 複数の独立した変更 | 変更ごとに分割 |
| リファクタリング + 機能追加 | 別コミットに分割 |

### Error Handling Decision (エラー対処フロー)

**ビルドエラー:**

| エラー種類 | 対処法 |
|-----------|--------|
| importエラー | `go mod tidy` を実行 |
| 型エラー | 型定義を確認、必要に応じて修正 |
| undefinedエラー | 関数・変数の定義漏れを確認 |
| 循環参照エラー | パッケージ構成を見直し |

**テストエラー:**

| エラー種類 | 対処法 |
|-----------|--------|
| アサーション失敗（期待値が間違い） | テストを修正 |
| アサーション失敗（実装が間違い） | 実装を修正 |
| パニック発生 | nil チェック、境界値を確認 |
| タイムアウト | 無限ループ、デッドロックを確認 |

**解決できない場合:**
1. エラーメッセージを正確に記録
2. 再現手順を整理
3. 関連するコードを特定
4. ユーザーに報告し、追加情報を求める

### User Confirmation Required (確認が必要なケース)

以下のケースでは、ユーザーに確認してから進める:

| ケース | 確認内容 |
|--------|---------|
| 破壊的変更 | 既存の振る舞いを変更してよいか |
| 大規模リファクタリング | 方針を確認 |
| 外部API仕様の不明点 | 仕様の詳細 |
| セキュリティに関わる変更 | 認証・認可の要件 |
| 複数の実装方法がある | どのアプローチを取るか |

### Self-Verification Checklist (自己検証)

作業完了時に確認:

**Backend:**
- [ ] `go build ./...` がパス
- [ ] `go test ./...` がパス
- [ ] 不要なデバッグコードを削除
- [ ] サービスを再起動して動作確認

**Frontend:**
- [ ] `npm run typecheck` がパス
- [ ] `npm run lint` がパス
- [ ] ブラウザで動作確認

**共通:**
- [ ] コーディング規約に違反していないか
- [ ] テストを追加・更新したか
- [ ] ドキュメントを更新したか（必要な場合）
- [ ] コミットメッセージが規約に従っている

### Code Conventions

**Go:**
- gofmt/goimports
- Explicit error handling (no `_` ignore)
- log/slog for logging
- testify for tests

**Vue/TS:**
- Composition API
- `<script setup lang="ts">`
- PascalCase components
- `use` prefix for composables
- **ブラウザのalert/confirm/promptは使用禁止** - AIブラウザ操作をブロックするため、`useToast()`を使用

### Git

- Branches: `feature/`, `fix/`, `docs/`
- Commits: Conventional Commits (`feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`)

**禁止事項:**

| 操作 | 理由 |
|------|------|
| `git push --force` to main/master | 履歴破壊 |
| `git reset --hard` on shared branch | 他者の作業を消す |
| シークレットのコミット | セキュリティリスク |
| 巨大なバイナリのコミット | リポジトリ肥大化 |

**注意が必要な操作:**

| 操作 | 注意点 |
|------|--------|
| `git commit --amend` | push済みの場合は禁止 |
| `git rebase` | 共有ブランチでは禁止 |
| `.gitignore` の変更 | 影響範囲を確認 |

### Multi-tenancy

All queries MUST include `tenant_id` filter.

### API Design

- Base: `/api/v1`
- Auth: Bearer JWT
- Dev mode: `X-Tenant-ID` header
- Error: `{"error": {"code": "...", "message": "..."}}`

### Session Management (AI Agent)

**セッション終了時の引き継ぎ：**

```
1. 未完了タスクをTodoWriteで記録
2. 実装した内容のドキュメント更新
3. 既知の問題・課題を明記
4. 次のアクションを明確に記載
```

**禁止事項：**
- ドキュメント更新なしでのセッション終了
- 暗黙的な前提に依存した実装
- 口頭説明でしか伝わらない設計決定

## Environment Variables

| Variable | Service | Description |
|----------|---------|-------------|
| `DATABASE_URL` | api, worker | PostgreSQL |
| `REDIS_URL` | api, worker | Redis |
| `AUTH_ENABLED` | api | Enable JWT (default: false) |
| `TELEMETRY_ENABLED` | api, worker | Enable tracing |
| `OPENAI_API_KEY` | worker | OpenAI key |
| `ANTHROPIC_API_KEY` | worker | Anthropic key |

## API Quick Test

```bash
# Create workflow
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"name": "Test"}'

# Add step
curl -X POST "http://localhost:8080/api/v1/workflows/{id}/steps" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"name": "Step1", "type": "tool", "config": {"adapter_id": "mock"}}'

# Publish
curl -X POST "http://localhost:8080/api/v1/workflows/{id}/publish" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001"

# Execute
curl -X POST "http://localhost:8080/api/v1/workflows/{id}/runs" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"input": {}, "mode": "test"}'
```

## Common Operations

### Add New Block / Integration (REQUIRED READING)

**⚠️ 外部連携（Discord, Slack, Notion等）を追加する場合、必ず以下を先に読むこと：**

```
1. [ ] docs/designs/UNIFIED_BLOCK_MODEL.md を読む
2. [ ] 既存ブロックの実装パターンを確認（migrations/011_unified_block_model.sql）
3. [ ] ctx インターフェース（http, llm, workflow, human, adapter）を理解する
```

**現在のアーキテクチャ（Unified Block Model）:**

| 方式 | 説明 | 用途 |
|------|------|------|
| **Migration追加** | JavaScriptコードをDBに登録 | **新規ブロック追加の標準方式** |
| Go Adapter追加 | Goでアダプター実装 | LLMプロバイダー等の特殊ケースのみ |

**新規ブロック追加手順（標準）:**

1. Migrationファイル作成: `backend/migrations/XXX_{name}_block.sql`
2. `block_definitions`テーブルにINSERT
   - `tenant_id = NULL` でシステムブロック（全ユーザー提供）
   - `code`にJavaScriptコード（`ctx.http`等を使用）
   - `ui_config`にアイコン・カラー・設定スキーマ
3. Migration実行
4. docs/BLOCK_REGISTRY.md を更新

**コード例（Discord通知ブロック）:**

```sql
INSERT INTO block_definitions (tenant_id, slug, name, category, code, ui_config, is_system)
VALUES (
  NULL,  -- システムブロック
  'discord',
  'Discord通知',
  'integration',
  $code$
    const webhookUrl = config.webhook_url || ctx.secrets.DISCORD_WEBHOOK_URL;
    const payload = { content: renderTemplate(config.message, input) };
    return await ctx.http.post(webhookUrl, payload);
  $code$,
  '{"icon": "message-circle", "color": "#5865F2", "configSchema": {...}}',
  TRUE
);
```

**Go Adapterが必要なケース（例外）:**

| ケース | 理由 |
|--------|------|
| LLMプロバイダー追加 | `ctx.llm`経由で呼び出すため |
| 複雑な認証フロー | OAuth2等、JSでは困難な場合 |
| バイナリ処理 | 画像・ファイル処理等 |

Go Adapter追加が必要な場合のみ:
1. Create `backend/internal/adapter/{name}.go`
2. Implement `Adapter` interface
3. Register in registry
4. Add test `{name}_test.go`
5. Update docs/BACKEND.md

### Add New API Endpoint

1. Add handler in `backend/internal/handler/`
2. Add route in `cmd/api/main.go`
3. Add usecase if needed
4. Update docs/API.md and docs/openapi.yaml

### Add Database Migration

1. Create SQL in `backend/migrations/`
2. Run: `docker compose exec api migrate -path /migrations -database "$DB_URL" up`
3. Update docs/DATABASE.md

## Implementation Status

All phases complete (Phase 1-8):
- Workflow CRUD, Steps, Edges
- DAG execution engine (conditions, map, join)
- Adapters (Mock, OpenAI, Anthropic, HTTP)
- Schedules, Webhooks
- Keycloak OIDC auth
- OpenTelemetry tracing
- E2E tests
- K8s deployment manifests
