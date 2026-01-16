# Database Reference

PostgreSQL schema, migrations, and query patterns.

> **Migration Note (2026-01)**: The `workflows` table has been renamed to `projects`. Projects now support multiple Start blocks, each with its own `trigger_type` and `trigger_config`. The `webhooks` table has been removed; webhook functionality is now part of Start block configuration. The `input_schema`/`output_schema` columns have been replaced with `variables` at the project level.

## Quick Reference

| Item | Value |
|------|-------|
| Driver | PostgreSQL 16 + pgvector |
| Connection URL | `postgres://user:pass@localhost:5432/ai_orchestration?sslmode=disable` |
| Pool | pgx connection pool |
| Migrations | `backend/migrations/` |
| Default Tenant | `00000000-0000-0000-0000-000000000001` |
| Soft Delete | `deleted_at` column |

## Schema Overview

```
tenants
  └── users
  └── projects (formerly workflows)
        └── project_versions (formerly workflow_versions)
        └── steps (multiple start blocks supported)
        └── edges
        └── block_groups
        └── schedules (now requires start_step_id)
  └── runs (now includes start_step_id)
        └── step_runs
        └── block_group_runs
        └── usage_records
  └── usage_daily_aggregates
  └── usage_budgets
  └── secrets
  └── audit_logs
  └── adapters
  └── block_definitions (※ tenant_id NULL = system)
        └── block_versions
  └── vector_collections (RAG)
        └── vector_documents (RAG)
```

> **Note**: The `webhooks` table has been removed. Webhook functionality is now configured via Start block's `trigger_type` and `trigger_config`.

## Tables

### tenants

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| name | VARCHAR(255) | NOT NULL | |
| slug | VARCHAR(255) | NOT NULL, UNIQUE | URL-safe identifier |
| settings | JSONB | DEFAULT '{}' | Tenant config |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |
| deleted_at | TIMESTAMPTZ | | Soft delete |

Default tenant: `00000000-0000-0000-0000-000000000001`

### users

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK | Keycloak user ID |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| email | VARCHAR(255) | NOT NULL | |
| name | VARCHAR(255) | | |
| role | VARCHAR(50) | NOT NULL DEFAULT 'viewer' | tenant_admin, builder, operator, viewer |
| last_login_at | TIMESTAMPTZ | | |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |

Unique: (tenant_id, email)

### projects (formerly workflows)

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| name | VARCHAR(255) | NOT NULL | |
| description | TEXT | | |
| status | VARCHAR(50) | NOT NULL DEFAULT 'draft' | draft, published |
| version | INTEGER | NOT NULL DEFAULT 1 | Increments on publish |
| variables | JSONB | | Project-level variables (replaces input_schema/output_schema) |
| created_by | UUID | FK users(id) | |
| published_at | TIMESTAMPTZ | | |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |
| deleted_at | TIMESTAMPTZ | | Soft delete |

> **Migration Note**: `input_schema` and `output_schema` have been removed. Input/output schemas are now defined per Start block in the `steps` table config.

Indexes:
- `idx_projects_tenant` ON (tenant_id)
- `idx_projects_status` ON (status)

### project_versions (formerly workflow_versions)

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| project_id | UUID | FK projects(id), NOT NULL | |
| version | INTEGER | NOT NULL | |
| definition | JSONB | NOT NULL | Full snapshot (steps, edges) |
| published_by | UUID | FK users(id) | |
| published_at | TIMESTAMPTZ | DEFAULT NOW() | |

Unique: (project_id, version)

### steps

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| project_id | UUID | FK projects(id) ON DELETE CASCADE, NOT NULL | |
| name | VARCHAR(255) | NOT NULL | |
| type | VARCHAR(50) | NOT NULL | start, llm, tool, condition, switch, map, join, subflow, wait, function, router, human_in_loop, filter, split, aggregate, error, note, log |
| config | JSONB | NOT NULL DEFAULT '{}' | Type-specific config (see below for Start block) |
| block_group_id | UUID | FK block_groups(id) ON DELETE SET NULL | Parent block group |
| group_role | VARCHAR(50) | | Role within block group (body only) |
| block_definition_id | UUID | FK block_definitions(id) | Registry block reference |
| credential_bindings | JSONB | DEFAULT '{}' | Mapping of credential names to tenant credential IDs |
| position_x | INTEGER | DEFAULT 0 | UI position |
| position_y | INTEGER | DEFAULT 0 | UI position |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |

**Start Block Config Schema** (for `type = 'start'`):

A project can have multiple Start blocks, each with its own trigger configuration:

```json
{
  "trigger_type": "manual|schedule|webhook",
  "trigger_config": {
    "input_mapping": {},
    "webhook_secret": "string",
    "cron": "0 9 * * *",
    "timezone": "Asia/Tokyo"
  },
  "input_schema": {},
  "output_schema": {}
}
```

| Trigger Type | trigger_config Fields |
|--------------|----------------------|
| `manual` | None required |
| `schedule` | `cron`, `timezone` (schedule also requires entry in schedules table) |
| `webhook` | `webhook_secret`, `input_mapping` |

### edges

Connects steps and/or block groups. Either source/target can be a step or a block group.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| project_id | UUID | FK projects(id) ON DELETE CASCADE, NOT NULL | |
| source_step_id | UUID | FK steps(id) ON DELETE CASCADE | Nullable if source is a group |
| target_step_id | UUID | FK steps(id) ON DELETE CASCADE | Nullable if target is a group |
| source_block_group_id | UUID | FK block_groups(id) ON DELETE CASCADE | Nullable if source is a step |
| target_block_group_id | UUID | FK block_groups(id) ON DELETE CASCADE | Nullable if target is a step |
| source_port | VARCHAR(100) | DEFAULT 'output' | Output port name |
| target_port | VARCHAR(100) | DEFAULT 'input' | Input port name |
| condition | TEXT | | Expression for conditional routing |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |

Unique: edges_unique_connection (one source/target pair)

### block_groups

Control flow constructs that group multiple steps.

> **Updated**: 2026-01-15 - Simplified to 4 types, added pre_process/post_process

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| project_id | UUID | FK projects(id) ON DELETE CASCADE, NOT NULL | |
| name | VARCHAR(255) | NOT NULL | Display name |
| type | VARCHAR(50) | NOT NULL, CHECK | **4 types only**: parallel, try_catch, foreach, while |
| config | JSONB | NOT NULL DEFAULT '{}' | Type-specific configuration |
| parent_group_id | UUID | FK block_groups(id) ON DELETE CASCADE | For nested groups |
| pre_process | TEXT | | JS code: external IN → internal IN |
| post_process | TEXT | | JS code: internal OUT → external OUT |
| position_x | INT | DEFAULT 0 | UI position X |
| position_y | INT | DEFAULT 0 | UI position Y |
| width | INT | DEFAULT 400 | UI width |
| height | INT | DEFAULT 300 | UI height |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |

Indexes:
- `idx_block_groups_project` ON (project_id)
- `idx_block_groups_parent` ON (parent_group_id)

**Type CHECK constraint**: `type IN ('parallel', 'try_catch', 'foreach', 'while')`

**Removed types**: `if_else` (use condition block), `switch_case` (use switch block)

**Note**: Steps can belong to a block group via `steps.block_group_id` and `steps.group_role` (body only).

### block_group_runs

Execution tracking for block groups.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| run_id | UUID | FK runs(id) ON DELETE CASCADE, NOT NULL | |
| block_group_id | UUID | FK block_groups(id) ON DELETE CASCADE, NOT NULL | |
| status | VARCHAR(50) | DEFAULT 'pending' | pending, running, completed, failed, skipped |
| iteration | INT | DEFAULT 0 | For loop groups |
| input | JSONB | | Group input |
| output | JSONB | | Group output |
| error | TEXT | | Error message |
| started_at | TIMESTAMPTZ | | |
| completed_at | TIMESTAMPTZ | | |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |

Indexes:
- `idx_block_group_runs_run` ON (run_id)
- `idx_block_group_runs_block_group` ON (block_group_id)

### runs

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| project_id | UUID | FK projects(id), NOT NULL | |
| project_version | INTEGER | NOT NULL | Snapshot version |
| start_step_id | UUID | FK steps(id) | Which Start block triggered this run |
| status | VARCHAR(50) | NOT NULL DEFAULT 'pending' | pending, running, completed, failed, cancelled |
| mode | VARCHAR(50) | NOT NULL DEFAULT 'production' | test, production |
| input | JSONB | | |
| output | JSONB | | |
| error | TEXT | | |
| triggered_by | VARCHAR(50) | NOT NULL DEFAULT 'manual' | manual, schedule, webhook |
| triggered_by_user | UUID | FK users(id) | |
| started_at | TIMESTAMPTZ | | |
| completed_at | TIMESTAMPTZ | | |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |

> **Migration Note**: `start_step_id` is now required to identify which Start block triggered the run, since projects can have multiple Start blocks.

Indexes:
- `idx_runs_tenant` ON (tenant_id)
- `idx_runs_project` ON (project_id)
- `idx_runs_start_step` ON (start_step_id)
- `idx_runs_status` ON (status)

### step_runs

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| run_id | UUID | FK runs(id) ON DELETE CASCADE, NOT NULL | |
| step_id | UUID | NOT NULL | Reference to step at execution time |
| step_name | VARCHAR(255) | NOT NULL | Snapshot of step name |
| status | VARCHAR(50) | NOT NULL DEFAULT 'pending' | pending, running, completed, failed |
| attempt | INTEGER | NOT NULL DEFAULT 1 | Retry count |
| input | JSONB | | |
| output | JSONB | | |
| error | TEXT | | |
| started_at | TIMESTAMPTZ | | |
| completed_at | TIMESTAMPTZ | | |
| duration_ms | INTEGER | | |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |

Indexes:
- `idx_step_runs_run` ON (run_id)

### schedules

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| project_id | UUID | FK projects(id), NOT NULL | |
| start_step_id | UUID | FK steps(id), NOT NULL | Which Start block to trigger |
| project_version | INTEGER | NOT NULL DEFAULT 1 | |
| name | VARCHAR(255) | NOT NULL | |
| description | TEXT | | |
| cron_expression | VARCHAR(100) | NOT NULL | Standard cron format |
| timezone | VARCHAR(50) | NOT NULL DEFAULT 'UTC' | IANA timezone |
| input | JSONB | | Default input for runs |
| status | VARCHAR(50) | NOT NULL DEFAULT 'active' | active, paused |
| next_run_at | TIMESTAMPTZ | | Computed next execution |
| last_run_at | TIMESTAMPTZ | | |
| last_run_id | UUID | FK runs(id) | |
| run_count | INTEGER | NOT NULL DEFAULT 0 | |
| created_by | UUID | FK users(id) | |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |

> **Migration Note**: `start_step_id` is now required to specify which Start block the schedule should trigger when it fires.

Indexes:
- `idx_schedules_tenant` ON (tenant_id)
- `idx_schedules_project` ON (project_id)
- `idx_schedules_start_step` ON (start_step_id)
- `idx_schedules_next_run` ON (next_run_at) WHERE status = 'active'

### webhooks (REMOVED)

> **Migration Note**: The `webhooks` table has been removed. Webhook functionality is now configured directly in Start blocks via the `trigger_type` and `trigger_config` fields.
>
> To migrate existing webhooks:
> 1. Create a Start block with `type: 'start'` and `config.trigger_type: 'webhook'`
> 2. Move `secret` to `config.trigger_config.webhook_secret`
> 3. Move `input_mapping` to `config.trigger_config.input_mapping`
> 4. The webhook endpoint becomes `/projects/{project_id}/webhook/{start_step_id}`

### adapters

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | VARCHAR(100) | PK | mock, openai, anthropic, http |
| tenant_id | UUID | FK tenants(id) | NULL = global |
| name | VARCHAR(255) | NOT NULL | |
| description | TEXT | | |
| type | VARCHAR(50) | NOT NULL | builtin, custom |
| config | JSONB | | Default config |
| input_schema | JSONB | | JSON Schema |
| output_schema | JSONB | | JSON Schema |
| enabled | BOOLEAN | NOT NULL DEFAULT true | |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |

### block_definitions

ブロック定義（Unified Block Model）。システムブロックとテナントカスタムブロックを管理。

> **Updated**: 2026-01-15 - Phase B: グループブロック統合（group_kind, is_container 追加）

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT gen_random_uuid() | |
| tenant_id | UUID | FK tenants(id) | NULL = system block |
| slug | VARCHAR(100) | NOT NULL | Unique identifier |
| name | VARCHAR(255) | NOT NULL | Display name |
| description | TEXT | | |
| category | VARCHAR(50) | NOT NULL, CHECK | ai, flow, apps, custom, **group** |
| subcategory | VARCHAR(50) | CHECK | chat, rag, routing, branching, data, control, utility, slack, discord, notion, github, google, linear, email, web |
| icon | VARCHAR(50) | | Icon identifier |
| config_schema | JSONB | NOT NULL DEFAULT '{}' | Config JSON Schema |
| input_schema | JSONB | | Input JSON Schema |
| output_schema | JSONB | | Output JSON Schema |
| code | TEXT | | JavaScript code (Unified Block Model) |
| ui_config | JSONB | NOT NULL DEFAULT '{}' | {icon, color, configSchema} |
| is_system | BOOLEAN | NOT NULL DEFAULT FALSE | System block = admin only |
| version | INTEGER | NOT NULL DEFAULT 1 | Version number |
| error_codes | JSONB | DEFAULT '[]' | Error code definitions |
| group_kind | VARCHAR(50) | CHECK | **Phase B**: parallel, try_catch, foreach, while (グループブロック用) |
| is_container | BOOLEAN | NOT NULL DEFAULT FALSE | **Phase B**: TRUE = 他のステップを含むことができる |
| enabled | BOOLEAN | DEFAULT true | |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |

Unique: (tenant_id, slug)

Indexes:
- `idx_block_definitions_tenant` ON (tenant_id)
- `idx_block_definitions_category` ON (category)
- `idx_block_definitions_enabled` ON (enabled)

**Constraints**:
- `valid_block_category`: category IN ('ai', 'flow', 'apps', 'custom', 'group')
- `valid_block_subcategory`: subcategory IS NULL OR subcategory IN ('chat', 'rag', 'routing', 'branching', 'data', 'control', 'utility', 'slack', 'discord', 'notion', 'github', 'google', 'linear', 'email', 'web')
- `valid_group_kind`: group_kind IS NULL OR group_kind IN ('parallel', 'try_catch', 'foreach', 'while')

**Group Blocks (Phase B)**:
- `category = 'group'` かつ `is_container = TRUE` のブロックはグループブロック
- Block Palette からドラッグ＆ドロップで配置可能
- システムブロック: parallel, try_catch, foreach, while

**See**: [UNIFIED_BLOCK_MODEL.md](./designs/UNIFIED_BLOCK_MODEL.md), [BLOCK_GROUP_REDESIGN.md](./designs/BLOCK_GROUP_REDESIGN.md)

### block_versions

ブロック定義のバージョン履歴。ロールバック機能をサポート。

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT gen_random_uuid() | |
| block_id | UUID | FK block_definitions(id) ON DELETE CASCADE, NOT NULL | |
| version | INTEGER | NOT NULL | Version number |
| code | TEXT | NOT NULL | Code snapshot |
| config_schema | JSONB | NOT NULL | Config schema snapshot |
| input_schema | JSONB | | Input schema snapshot |
| output_schema | JSONB | | Output schema snapshot |
| ui_config | JSONB | NOT NULL | UI config snapshot |
| change_summary | TEXT | | Change description |
| changed_by | UUID | | User who made the change |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | |

Unique: (block_id, version)

Indexes:
- `idx_block_versions_block_id` ON (block_id)
- `idx_block_versions_created_at` ON (created_at)

### vector_collections

RAG用ベクトルコレクション。テナントごとに分離されたベクトルデータを管理。

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT gen_random_uuid() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | ⚠️ テナント分離必須 |
| name | VARCHAR(100) | NOT NULL | コレクション名（テナント内でユニーク） |
| description | TEXT | | |
| embedding_provider | VARCHAR(50) | DEFAULT 'openai' | 使用するEmbeddingプロバイダー |
| embedding_model | VARCHAR(100) | DEFAULT 'text-embedding-3-small' | 使用するモデル |
| dimension | INT | NOT NULL DEFAULT 1536 | ベクトル次元数 |
| document_count | INT | DEFAULT 0 | ドキュメント数（キャッシュ） |
| metadata | JSONB | DEFAULT '{}' | カスタムメタデータ |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |

Unique: (tenant_id, name)

Indexes:
- `idx_vector_collections_tenant` ON (tenant_id)

### vector_documents

RAG用ベクトルドキュメント。pgvector拡張を使用。

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT gen_random_uuid() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | ⚠️ テナント分離必須 |
| collection_id | UUID | FK vector_collections(id) ON DELETE CASCADE, NOT NULL | 所属コレクション |
| content | TEXT | NOT NULL | ドキュメント本文 |
| metadata | JSONB | DEFAULT '{}' | カスタムメタデータ |
| embedding | vector(1536) | | pgvectorベクトル型 |
| source_url | TEXT | | 元URLなど |
| source_type | VARCHAR(50) | | api, file, web |
| chunk_index | INT | | チャンク分割時のインデックス |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |

Indexes:
- `idx_vector_documents_tenant_collection` ON (tenant_id, collection_id) - 複合インデックス
- `idx_vector_documents_embedding` ON (embedding) USING ivfflat WITH (lists = 100) - 類似検索用
- `idx_vector_documents_metadata` ON (metadata) USING gin - メタデータフィルタ用

**Note**: pgvector拡張が必要です（`CREATE EXTENSION IF NOT EXISTS vector;`）

### usage_records

Individual LLM API call records for cost tracking.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| project_id | UUID | FK projects(id) | Nullable for non-project calls |
| run_id | UUID | FK runs(id) | |
| step_run_id | UUID | FK step_runs(id) | |
| provider | VARCHAR(50) | NOT NULL | openai, anthropic, google |
| model | VARCHAR(100) | NOT NULL | gpt-4o, claude-3-opus, etc. |
| operation | VARCHAR(50) | NOT NULL | chat, completion, embedding |
| input_tokens | INT | NOT NULL DEFAULT 0 | Prompt tokens |
| output_tokens | INT | NOT NULL DEFAULT 0 | Completion tokens |
| total_tokens | INT | NOT NULL DEFAULT 0 | input + output |
| input_cost_usd | DECIMAL(12, 8) | NOT NULL DEFAULT 0 | Cost for input tokens |
| output_cost_usd | DECIMAL(12, 8) | NOT NULL DEFAULT 0 | Cost for output tokens |
| total_cost_usd | DECIMAL(12, 8) | NOT NULL DEFAULT 0 | Total cost |
| latency_ms | INT | | Response time |
| success | BOOLEAN | NOT NULL DEFAULT TRUE | Whether call succeeded |
| error_message | TEXT | | Error details if failed |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | |

Indexes:
- `idx_usage_records_tenant_created` ON (tenant_id, created_at DESC)
- `idx_usage_records_project` ON (project_id) WHERE project_id IS NOT NULL
- `idx_usage_records_run` ON (run_id) WHERE run_id IS NOT NULL

### usage_daily_aggregates

Pre-aggregated daily usage for dashboard performance.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| project_id | UUID | FK projects(id) | NULL for tenant-wide aggregate |
| date | DATE | NOT NULL | Aggregation date |
| provider | VARCHAR(50) | NOT NULL | |
| model | VARCHAR(100) | NOT NULL | |
| total_requests | INT | NOT NULL DEFAULT 0 | |
| total_input_tokens | BIGINT | NOT NULL DEFAULT 0 | |
| total_output_tokens | BIGINT | NOT NULL DEFAULT 0 | |
| total_cost_usd | DECIMAL(12, 6) | NOT NULL DEFAULT 0 | |
| avg_latency_ms | INT | | |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | |

Unique: (tenant_id, project_id, date, provider, model)

Indexes:
- `idx_usage_daily_tenant_date` ON (tenant_id, date DESC)

### usage_budgets

Budget limits and alert thresholds.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| project_id | UUID | FK projects(id) | NULL for tenant-wide budget |
| budget_type | VARCHAR(50) | NOT NULL | monthly, daily |
| budget_amount_usd | DECIMAL(12, 2) | NOT NULL | Budget limit |
| alert_threshold | DECIMAL(3, 2) | NOT NULL DEFAULT 0.80 | 0.0-1.0, triggers alert |
| enabled | BOOLEAN | NOT NULL DEFAULT TRUE | |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | |

Indexes:
- `idx_usage_budgets_tenant` ON (tenant_id)
- `idx_usage_budgets_project` ON (project_id) WHERE project_id IS NOT NULL

### secrets

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| name | VARCHAR(255) | NOT NULL | |
| encrypted_value | TEXT | NOT NULL | AES-256 encrypted |
| created_by | UUID | FK users(id) | |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |

Unique: (tenant_id, name)

### audit_logs

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| actor_id | UUID | | User who performed action |
| actor_email | VARCHAR(255) | | |
| action | VARCHAR(100) | NOT NULL | create, update, delete, publish, execute |
| resource_type | VARCHAR(100) | NOT NULL | project, run, secret |
| resource_id | UUID | | |
| metadata | JSONB | | Additional context |
| ip_address | INET | | |
| user_agent | TEXT | | |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |

Indexes:
- `idx_audit_logs_tenant` ON (tenant_id)
- `idx_audit_logs_created` ON (created_at)

## Canonical Query Patterns (必須)

Claude Code はこのセクションのパターンに従ってクエリを書くこと。

### 必須ルール

| ルール | 説明 | 違反時のリスク |
|--------|------|---------------|
| `tenant_id` フィルタ必須 | すべての SELECT/UPDATE/DELETE に必須 | テナント分離違反（データ漏洩） |
| `deleted_at IS NULL` 必須 | soft delete 対応テーブルで必須 | 削除済みデータを取得 |
| `SELECT *` 禁止 | カラムを明示的に指定 | スキーマ変更時に壊れる |
| プレースホルダー必須 | `$1`, `$2` を使用 | SQL インジェクション |

### 正しいパターン vs 禁止パターン

```sql
-- ✅ 正しいパターン
SELECT id, tenant_id, name, status, created_at, updated_at
FROM projects
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL;

-- ❌ 禁止パターン
SELECT * FROM projects WHERE id = $1;
-- 問題: SELECT *, tenant_id なし, deleted_at なし
```

---

## Query Patterns

### List Projects (with tenant isolation)

```sql
SELECT *
FROM projects
WHERE tenant_id = $1
  AND deleted_at IS NULL
ORDER BY updated_at DESC
LIMIT $2 OFFSET $3;
```

### Get Project with Steps and Edges

```sql
-- Project
SELECT * FROM projects WHERE id = $1 AND tenant_id = $2;

-- Steps (including multiple Start blocks)
SELECT * FROM steps WHERE project_id = $1 ORDER BY created_at;

-- Edges
SELECT * FROM edges WHERE project_id = $1;
```

### Get Start Blocks for Project

```sql
-- Get all Start blocks with their trigger configurations
SELECT id, name, config
FROM steps
WHERE project_id = $1
  AND type = 'start'
ORDER BY created_at;
```

### Get Run with StepRuns

```sql
SELECT r.*, json_agg(sr.*) AS step_runs
FROM runs r
LEFT JOIN step_runs sr ON sr.run_id = r.id
WHERE r.id = $1
GROUP BY r.id;
```

### Find Active Schedules Due

```sql
SELECT *
FROM schedules
WHERE status = 'active'
  AND next_run_at <= NOW()
ORDER BY next_run_at;
```

### Count Runs by Status (for dashboard)

```sql
SELECT status, COUNT(*) as count
FROM runs
WHERE tenant_id = $1
  AND created_at >= $2
GROUP BY status;
```

### Get Usage Summary by Period

```sql
SELECT
    COUNT(*) as total_requests,
    COALESCE(SUM(input_tokens), 0) as total_input_tokens,
    COALESCE(SUM(output_tokens), 0) as total_output_tokens,
    COALESCE(SUM(total_cost_usd), 0) as total_cost_usd,
    AVG(CASE WHEN success THEN 1 ELSE 0 END) as success_rate,
    AVG(latency_ms) as avg_latency_ms
FROM usage_records
WHERE tenant_id = $1
  AND created_at >= $2
  AND created_at < $3;
```

### Get Usage by Model

```sql
SELECT
    provider,
    model,
    COUNT(*) as total_requests,
    SUM(input_tokens) as total_input_tokens,
    SUM(output_tokens) as total_output_tokens,
    SUM(total_cost_usd) as total_cost_usd,
    AVG(latency_ms) as avg_latency_ms
FROM usage_records
WHERE tenant_id = $1
  AND created_at >= $2
  AND created_at < $3
GROUP BY provider, model
ORDER BY total_cost_usd DESC;
```

### Vector Similarity Search (RAG)

⚠️ **重要**: すべてのベクトルクエリは`tenant_id`フィルタを必須とする。

```sql
-- コレクション取得/作成
SELECT id FROM vector_collections
WHERE tenant_id = $1 AND name = $2;

-- ベクトル類似検索（コサイン類似度）
SELECT
    vd.id,
    vd.content,
    vd.metadata,
    1 - (vd.embedding <=> $3::vector) as score
FROM vector_documents vd
JOIN vector_collections vc ON vd.collection_id = vc.id
WHERE vc.tenant_id = $1
  AND vc.name = $2
  AND vd.tenant_id = $1
ORDER BY vd.embedding <=> $3::vector
LIMIT $4;

-- メタデータフィルタ付き検索
SELECT
    vd.id,
    vd.content,
    1 - (vd.embedding <=> $3::vector) as score
FROM vector_documents vd
JOIN vector_collections vc ON vd.collection_id = vc.id
WHERE vc.tenant_id = $1
  AND vc.name = $2
  AND vd.tenant_id = $1
  AND vd.metadata->>'source_type' = $4
ORDER BY vd.embedding <=> $3::vector
LIMIT $5;
```

## Migration Commands

```bash
# Apply migrations (using golang-migrate)
docker compose exec api migrate -path /migrations -database "$DATABASE_URL" up

# Rollback last migration
docker compose exec api migrate -path /migrations -database "$DATABASE_URL" down 1

# Force version (dangerous)
docker compose exec api migrate -path /migrations -database "$DATABASE_URL" force VERSION
```

## Soft Delete Pattern

All tenant-owned tables support soft delete via `deleted_at` column:

```sql
-- "Delete"
UPDATE projects SET deleted_at = NOW() WHERE id = $1;

-- Query (exclude deleted)
SELECT * FROM projects WHERE deleted_at IS NULL;

-- Hard delete (admin only)
DELETE FROM projects WHERE id = $1;
```

## Multi-Tenancy Pattern

All queries MUST include `tenant_id`:

```go
func (r *ProjectRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Project, error) {
    return r.db.QueryRow(ctx,
        `SELECT * FROM projects WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
        id, tenantID,
    ).Scan(...)
}
```

## JSONB Column Usage

| Table | Column | Content |
|-------|--------|---------|
| tenants | settings | `{"data_retention_days": 30, "max_concurrent_runs": 10}` |
| projects | variables | Project-level variables |
| steps | config | Step type-specific config (Start blocks include trigger_type, trigger_config, input_schema, output_schema) |
| runs | input | Execution input |
| runs | output | Execution result |
| audit_logs | metadata | Action-specific details |

## Connection Pool Settings

```go
config := pgxpool.Config{
    MaxConns:          25,
    MinConns:          5,
    MaxConnLifetime:   time.Hour,
    MaxConnIdleTime:   30 * time.Minute,
    HealthCheckPeriod: time.Minute,
}
```

## Backup

```bash
# Dump
pg_dump -h localhost -U postgres ai_orchestration > backup.sql

# Restore
psql -h localhost -U postgres ai_orchestration < backup.sql
```

## Related Documents

- [BACKEND.md](./BACKEND.md) - Repository interfaces and data access patterns
- [API.md](./API.md) - API endpoints that interact with database
- [UNIFIED_BLOCK_MODEL.md](./designs/UNIFIED_BLOCK_MODEL.md) - Block definitions schema
- [BLOCK_REGISTRY.md](./BLOCK_REGISTRY.md) - Block definition tables (RAGブロック含む)
- [RAG_IMPLEMENTATION_PLAN.md](./plans/RAG_IMPLEMENTATION_PLAN.md) - RAG機能の設計書
