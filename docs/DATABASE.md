# データベースリファレンス

PostgreSQL スキーマ、マイグレーション、クエリパターン。

> **マイグレーション注記 (2026-01)**: `workflows` テーブルは `projects` にリネームされました。プロジェクトは複数の Start ブロックをサポートし、各 Start ブロックは独自の `trigger_type` と `trigger_config` を持ちます。`webhooks` テーブルは削除され、Webhook 機能は Start ブロックの設定に統合されました。`input_schema`/`output_schema` カラムはプロジェクトレベルの `variables` に置き換えられました。

## クイックリファレンス

| 項目 | 値 |
|------|-------|
| ドライバー | PostgreSQL 16 + pgvector |
| 接続 URL | `postgres://user:pass@localhost:5432/ai_orchestration?sslmode=disable` |
| プール | pgx コネクションプール |
| マイグレーション | `backend/migrations/` |
| デフォルトテナント | `00000000-0000-0000-0000-000000000001` |
| ソフトデリート | `deleted_at` カラム |

## スキーマ概要

```
tenants
  └── users
  └── projects（旧 workflows）
        └── project_versions（旧 workflow_versions）
        └── steps（複数の Start ブロックをサポート）
        └── edges
        └── block_groups
        └── schedules（start_step_id が必須）
  └── runs（start_step_id を含む）
        └── step_runs
        └── block_group_runs
        └── usage_records
  └── usage_daily_aggregates
  └── usage_budgets
  └── secrets
  └── credentials
        └── credential_shares
        └── oauth2_connections
  └── audit_logs
  └── adapters
  └── block_definitions（※ tenant_id NULL = システムブロック）
        └── block_versions
  └── vector_collections（RAG）
        └── vector_documents（RAG）
  └── copilot_sessions
        └── copilot_messages

oauth2_providers（グローバル）
  └── oauth2_apps（テナント固有）
```

> **注記**: `webhooks` テーブルは削除されました。Webhook 機能は Start ブロックの `trigger_type` と `trigger_config` で設定されます。

## テーブル

### tenants

| カラム | 型 | 制約 | 説明 |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| name | VARCHAR(255) | NOT NULL | |
| slug | VARCHAR(255) | NOT NULL, UNIQUE | URL セーフな識別子 |
| settings | JSONB | DEFAULT '{}' | テナント設定 |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |
| deleted_at | TIMESTAMPTZ | | ソフトデリート |

デフォルトテナント: `00000000-0000-0000-0000-000000000001`

### users

| カラム | 型 | 制約 | 説明 |
|--------|------|-------------|-------------|
| id | UUID | PK | Keycloak ユーザー ID |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| email | VARCHAR(255) | NOT NULL | |
| name | VARCHAR(255) | | |
| role | VARCHAR(50) | NOT NULL DEFAULT 'viewer' | tenant_admin, builder, operator, viewer |
| last_login_at | TIMESTAMPTZ | | |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |

ユニーク: (tenant_id, email)

### projects（旧 workflows）

| カラム | 型 | 制約 | 説明 |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| name | VARCHAR(255) | NOT NULL | |
| description | TEXT | | |
| status | VARCHAR(50) | NOT NULL DEFAULT 'draft' | draft, published |
| version | INTEGER | NOT NULL DEFAULT 1 | 公開時にインクリメント |
| variables | JSONB | | プロジェクトレベル変数（input_schema/output_schema を置換） |
| created_by | UUID | FK users(id) | |
| published_at | TIMESTAMPTZ | | |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |
| deleted_at | TIMESTAMPTZ | | ソフトデリート |

> **マイグレーション注記**: `input_schema` と `output_schema` は削除されました。入出力スキーマは `steps` テーブルの Start ブロック config 内で定義されます。

インデックス:
- `idx_projects_tenant` ON (tenant_id)
- `idx_projects_status` ON (status)

### project_versions（旧 workflow_versions）

| カラム | 型 | 制約 | 説明 |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| project_id | UUID | FK projects(id), NOT NULL | |
| version | INTEGER | NOT NULL | |
| definition | JSONB | NOT NULL | 完全なスナップショット（steps, edges） |
| published_by | UUID | FK users(id) | |
| published_at | TIMESTAMPTZ | DEFAULT NOW() | |

ユニーク: (project_id, version)

### steps

| カラム | 型 | 制約 | 説明 |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| project_id | UUID | FK projects(id) ON DELETE CASCADE, NOT NULL | |
| name | VARCHAR(255) | NOT NULL | |
| type | VARCHAR(50) | NOT NULL | start, llm, tool, condition, switch, map, join, subflow, wait, function, router, human_in_loop, filter, split, aggregate, error, note, log |
| config | JSONB | NOT NULL DEFAULT '{}' | 型固有の設定（Start ブロックについては下記参照） |
| block_group_id | UUID | FK block_groups(id) ON DELETE SET NULL | 親ブロックグループ |
| group_role | VARCHAR(50) | | ブロックグループ内の役割（body のみ） |
| block_definition_id | UUID | FK block_definitions(id) | レジストリブロック参照 |
| credential_bindings | JSONB | DEFAULT '{}' | クレデンシャル名からテナントクレデンシャル ID へのマッピング |
| position_x | INTEGER | DEFAULT 0 | UI 位置 |
| position_y | INTEGER | DEFAULT 0 | UI 位置 |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |

**Start ブロック Config スキーマ**（`type = 'start'` の場合）:

プロジェクトは複数の Start ブロックを持つことができ、各ブロックは独自のトリガー設定を持ちます:

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

| トリガータイプ | trigger_config フィールド |
|--------------|----------------------|
| `manual` | 必須フィールドなし |
| `schedule` | `cron`, `timezone`（schedules テーブルへのエントリも必要） |
| `webhook` | `webhook_secret`, `input_mapping` |

### edges

ステップおよび/またはブロックグループを接続します。ソース/ターゲットはステップまたはブロックグループのいずれかです。

| カラム | 型 | 制約 | 説明 |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| project_id | UUID | FK projects(id) ON DELETE CASCADE, NOT NULL | |
| source_step_id | UUID | FK steps(id) ON DELETE CASCADE | ソースがグループの場合は Null |
| target_step_id | UUID | FK steps(id) ON DELETE CASCADE | ターゲットがグループの場合は Null |
| source_block_group_id | UUID | FK block_groups(id) ON DELETE CASCADE | ソースがステップの場合は Null |
| target_block_group_id | UUID | FK block_groups(id) ON DELETE CASCADE | ターゲットがステップの場合は Null |
| source_port | VARCHAR(100) | DEFAULT 'output' | 出力ポート名 |
| target_port | VARCHAR(100) | DEFAULT 'input' | 入力ポート名 |
| condition | TEXT | | 条件分岐ルーティング用の式 |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |

ユニーク: edges_unique_connection（ソース/ターゲットペアは一意）

### block_groups

複数のステップをグループ化する制御フロー構造。

> **更新**: 2026-01-15 - 4 タイプに簡素化、pre_process/post_process を追加

| カラム | 型 | 制約 | 説明 |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| project_id | UUID | FK projects(id) ON DELETE CASCADE, NOT NULL | |
| name | VARCHAR(255) | NOT NULL | 表示名 |
| type | VARCHAR(50) | NOT NULL, CHECK | **4 タイプのみ**: parallel, try_catch, foreach, while |
| config | JSONB | NOT NULL DEFAULT '{}' | タイプ固有の設定 |
| parent_group_id | UUID | FK block_groups(id) ON DELETE CASCADE | ネストされたグループ用 |
| pre_process | TEXT | | JS コード: 外部 IN → 内部 IN |
| post_process | TEXT | | JS コード: 内部 OUT → 外部 OUT |
| position_x | INT | DEFAULT 0 | UI 位置 X |
| position_y | INT | DEFAULT 0 | UI 位置 Y |
| width | INT | DEFAULT 400 | UI 幅 |
| height | INT | DEFAULT 300 | UI 高さ |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |

インデックス:
- `idx_block_groups_project` ON (project_id)
- `idx_block_groups_parent` ON (parent_group_id)

**タイプ CHECK 制約**: `type IN ('parallel', 'try_catch', 'foreach', 'while')`

**削除されたタイプ**: `if_else`（condition ブロックを使用）、`switch_case`（switch ブロックを使用）

**注記**: ステップは `steps.block_group_id` と `steps.group_role`（body のみ）を通じてブロックグループに所属できます。

### block_group_runs

ブロックグループの実行追跡。

| カラム | 型 | 制約 | 説明 |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| run_id | UUID | FK runs(id) ON DELETE CASCADE, NOT NULL | |
| block_group_id | UUID | FK block_groups(id) ON DELETE CASCADE, NOT NULL | |
| status | VARCHAR(50) | DEFAULT 'pending' | pending, running, completed, failed, skipped |
| iteration | INT | DEFAULT 0 | ループグループ用 |
| input | JSONB | | グループ入力 |
| output | JSONB | | グループ出力 |
| error | TEXT | | エラーメッセージ |
| started_at | TIMESTAMPTZ | | |
| completed_at | TIMESTAMPTZ | | |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |

インデックス:
- `idx_block_group_runs_run` ON (run_id)
- `idx_block_group_runs_block_group` ON (block_group_id)

### runs

| カラム | 型 | 制約 | 説明 |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| project_id | UUID | FK projects(id), NOT NULL | |
| project_version | INTEGER | NOT NULL | スナップショットバージョン |
| start_step_id | UUID | FK steps(id) | この Run をトリガーした Start ブロック |
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

> **マイグレーション注記**: `start_step_id` は、プロジェクトが複数の Start ブロックを持つことができるため、どの Start ブロックが Run をトリガーしたかを識別するために必須です。

インデックス:
- `idx_runs_tenant` ON (tenant_id)
- `idx_runs_project` ON (project_id)
- `idx_runs_start_step` ON (start_step_id)
- `idx_runs_status` ON (status)

### step_runs

| カラム | 型 | 制約 | 説明 |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| run_id | UUID | FK runs(id) ON DELETE CASCADE, NOT NULL | |
| step_id | UUID | NOT NULL | 実行時のステップ参照 |
| step_name | VARCHAR(255) | NOT NULL | ステップ名のスナップショット |
| status | VARCHAR(50) | NOT NULL DEFAULT 'pending' | pending, running, completed, failed |
| attempt | INTEGER | NOT NULL DEFAULT 1 | リトライ回数 |
| input | JSONB | | |
| output | JSONB | | |
| error | TEXT | | |
| started_at | TIMESTAMPTZ | | |
| completed_at | TIMESTAMPTZ | | |
| duration_ms | INTEGER | | |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |

インデックス:
- `idx_step_runs_run` ON (run_id)

### schedules

| カラム | 型 | 制約 | 説明 |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| project_id | UUID | FK projects(id), NOT NULL | |
| start_step_id | UUID | FK steps(id), NOT NULL | トリガーする Start ブロック |
| project_version | INTEGER | NOT NULL DEFAULT 1 | |
| name | VARCHAR(255) | NOT NULL | |
| description | TEXT | | |
| cron_expression | VARCHAR(100) | NOT NULL | 標準 cron 形式 |
| timezone | VARCHAR(50) | NOT NULL DEFAULT 'UTC' | IANA タイムゾーン |
| input | JSONB | | Run のデフォルト入力 |
| status | VARCHAR(50) | NOT NULL DEFAULT 'active' | active, paused |
| next_run_at | TIMESTAMPTZ | | 計算された次回実行時刻 |
| last_run_at | TIMESTAMPTZ | | |
| last_run_id | UUID | FK runs(id) | |
| run_count | INTEGER | NOT NULL DEFAULT 0 | |
| created_by | UUID | FK users(id) | |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |

> **マイグレーション注記**: `start_step_id` は、スケジュール発火時にどの Start ブロックをトリガーするかを指定するために必須です。

インデックス:
- `idx_schedules_tenant` ON (tenant_id)
- `idx_schedules_project` ON (project_id)
- `idx_schedules_start_step` ON (start_step_id)
- `idx_schedules_next_run` ON (next_run_at) WHERE status = 'active'

### webhooks（削除済み）

> **マイグレーション注記**: `webhooks` テーブルは削除されました。Webhook 機能は Start ブロックの `trigger_type` と `trigger_config` フィールドで直接設定されます。
>
> 既存の Webhook を移行するには:
> 1. `type: 'start'` と `config.trigger_type: 'webhook'` を持つ Start ブロックを作成
> 2. `secret` を `config.trigger_config.webhook_secret` に移動
> 3. `input_mapping` を `config.trigger_config.input_mapping` に移動
> 4. Webhook エンドポイントは `/projects/{project_id}/webhook/{start_step_id}` になります

### adapters

| カラム | 型 | 制約 | 説明 |
|--------|------|-------------|-------------|
| id | VARCHAR(100) | PK | mock, openai, anthropic, http |
| tenant_id | UUID | FK tenants(id) | NULL = グローバル |
| name | VARCHAR(255) | NOT NULL | |
| description | TEXT | | |
| type | VARCHAR(50) | NOT NULL | builtin, custom |
| config | JSONB | | デフォルト設定 |
| input_schema | JSONB | | JSON スキーマ |
| output_schema | JSONB | | JSON スキーマ |
| enabled | BOOLEAN | NOT NULL DEFAULT true | |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |

### block_definitions

ブロック定義（Unified Block Model）。システムブロックとテナントカスタムブロックを管理。

> **更新**: 2026-01-15 - Phase B: グループブロック統合（group_kind, is_container 追加）

| カラム | 型 | 制約 | 説明 |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT gen_random_uuid() | |
| tenant_id | UUID | FK tenants(id) | NULL = システムブロック |
| slug | VARCHAR(100) | NOT NULL | ユニーク識別子 |
| name | VARCHAR(255) | NOT NULL | 表示名 |
| description | TEXT | | |
| category | VARCHAR(50) | NOT NULL, CHECK | ai, flow, apps, custom, **group** |
| subcategory | VARCHAR(50) | CHECK | chat, rag, routing, branching, data, control, utility, slack, discord, notion, github, google, linear, email, web |
| icon | VARCHAR(50) | | アイコン識別子 |
| config_schema | JSONB | NOT NULL DEFAULT '{}' | Config JSON スキーマ |
| input_schema | JSONB | | 入力 JSON スキーマ |
| output_schema | JSONB | | 出力 JSON スキーマ |
| code | TEXT | | JavaScript コード（Unified Block Model） |
| ui_config | JSONB | NOT NULL DEFAULT '{}' | {icon, color, configSchema} |
| is_system | BOOLEAN | NOT NULL DEFAULT FALSE | システムブロック = 管理者のみ |
| version | INTEGER | NOT NULL DEFAULT 1 | バージョン番号 |
| error_codes | JSONB | DEFAULT '[]' | エラーコード定義 |
| group_kind | VARCHAR(50) | CHECK | **Phase B**: parallel, try_catch, foreach, while（グループブロック用） |
| is_container | BOOLEAN | NOT NULL DEFAULT FALSE | **Phase B**: TRUE = 他のステップを含むことができる |
| enabled | BOOLEAN | DEFAULT true | |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |

ユニーク: (tenant_id, slug)

インデックス:
- `idx_block_definitions_tenant` ON (tenant_id)
- `idx_block_definitions_category` ON (category)
- `idx_block_definitions_enabled` ON (enabled)

**制約**:
- `valid_block_category`: category IN ('ai', 'flow', 'apps', 'custom', 'group')
- `valid_block_subcategory`: subcategory IS NULL OR subcategory IN ('chat', 'rag', 'routing', 'branching', 'data', 'control', 'utility', 'slack', 'discord', 'notion', 'github', 'google', 'linear', 'email', 'web')
- `valid_group_kind`: group_kind IS NULL OR group_kind IN ('parallel', 'try_catch', 'foreach', 'while')

**グループブロック（Phase B）**:
- `category = 'group'` かつ `is_container = TRUE` のブロックはグループブロック
- Block Palette からドラッグ＆ドロップで配置可能
- システムブロック: parallel, try_catch, foreach, while

**参照**: [UNIFIED_BLOCK_MODEL.md](./designs/UNIFIED_BLOCK_MODEL.md), [BLOCK_GROUP_REDESIGN.md](./designs/BLOCK_GROUP_REDESIGN.md)

### block_versions

ブロック定義のバージョン履歴。ロールバック機能をサポート。

| カラム | 型 | 制約 | 説明 |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT gen_random_uuid() | |
| block_id | UUID | FK block_definitions(id) ON DELETE CASCADE, NOT NULL | |
| version | INTEGER | NOT NULL | バージョン番号 |
| code | TEXT | NOT NULL | コードスナップショット |
| config_schema | JSONB | NOT NULL | Config スキーマスナップショット |
| input_schema | JSONB | | 入力スキーマスナップショット |
| output_schema | JSONB | | 出力スキーマスナップショット |
| ui_config | JSONB | NOT NULL | UI 設定スナップショット |
| change_summary | TEXT | | 変更説明 |
| changed_by | UUID | | 変更者ユーザー |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | |

ユニーク: (block_id, version)

インデックス:
- `idx_block_versions_block_id` ON (block_id)
- `idx_block_versions_created_at` ON (created_at)

### vector_collections

RAG 用ベクトルコレクション。テナントごとに分離されたベクトルデータを管理。

| カラム | 型 | 制約 | 説明 |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT gen_random_uuid() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | ⚠️ テナント分離必須 |
| name | VARCHAR(100) | NOT NULL | コレクション名（テナント内でユニーク） |
| description | TEXT | | |
| embedding_provider | VARCHAR(50) | DEFAULT 'openai' | 使用する Embedding プロバイダー |
| embedding_model | VARCHAR(100) | DEFAULT 'text-embedding-3-small' | 使用するモデル |
| dimension | INT | NOT NULL DEFAULT 1536 | ベクトル次元数 |
| document_count | INT | DEFAULT 0 | ドキュメント数（キャッシュ） |
| metadata | JSONB | DEFAULT '{}' | カスタムメタデータ |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |

ユニーク: (tenant_id, name)

インデックス:
- `idx_vector_collections_tenant` ON (tenant_id)

### vector_documents

RAG 用ベクトルドキュメント。pgvector 拡張を使用。

| カラム | 型 | 制約 | 説明 |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT gen_random_uuid() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | ⚠️ テナント分離必須 |
| collection_id | UUID | FK vector_collections(id) ON DELETE CASCADE, NOT NULL | 所属コレクション |
| content | TEXT | NOT NULL | ドキュメント本文 |
| metadata | JSONB | DEFAULT '{}' | カスタムメタデータ |
| embedding | vector(1536) | | pgvector ベクトル型 |
| source_url | TEXT | | 元 URL など |
| source_type | VARCHAR(50) | | api, file, web |
| chunk_index | INT | | チャンク分割時のインデックス |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |

インデックス:
- `idx_vector_documents_tenant_collection` ON (tenant_id, collection_id) - 複合インデックス
- `idx_vector_documents_embedding` ON (embedding) USING ivfflat WITH (lists = 100) - 類似検索用
- `idx_vector_documents_metadata` ON (metadata) USING gin - メタデータフィルタ用

**注記**: pgvector 拡張が必要です（`CREATE EXTENSION IF NOT EXISTS vector;`）

### usage_records

コスト追跡のための個別 LLM API 呼び出しレコード。

| カラム | 型 | 制約 | 説明 |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| project_id | UUID | FK projects(id) | プロジェクト外呼び出しの場合は Null |
| run_id | UUID | FK runs(id) | |
| step_run_id | UUID | FK step_runs(id) | |
| provider | VARCHAR(50) | NOT NULL | openai, anthropic, google |
| model | VARCHAR(100) | NOT NULL | gpt-4o, claude-3-opus など |
| operation | VARCHAR(50) | NOT NULL | chat, completion, embedding |
| input_tokens | INT | NOT NULL DEFAULT 0 | プロンプトトークン |
| output_tokens | INT | NOT NULL DEFAULT 0 | 完了トークン |
| total_tokens | INT | NOT NULL DEFAULT 0 | input + output |
| input_cost_usd | DECIMAL(12, 8) | NOT NULL DEFAULT 0 | 入力トークンのコスト |
| output_cost_usd | DECIMAL(12, 8) | NOT NULL DEFAULT 0 | 出力トークンのコスト |
| total_cost_usd | DECIMAL(12, 8) | NOT NULL DEFAULT 0 | 合計コスト |
| latency_ms | INT | | 応答時間 |
| success | BOOLEAN | NOT NULL DEFAULT TRUE | 呼び出しが成功したかどうか |
| error_message | TEXT | | 失敗時のエラー詳細 |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | |

インデックス:
- `idx_usage_records_tenant_created` ON (tenant_id, created_at DESC)
- `idx_usage_records_project` ON (project_id) WHERE project_id IS NOT NULL
- `idx_usage_records_run` ON (run_id) WHERE run_id IS NOT NULL

### usage_daily_aggregates

ダッシュボードパフォーマンスのための日次使用量集計。

| カラム | 型 | 制約 | 説明 |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| project_id | UUID | FK projects(id) | テナント全体の集計の場合は NULL |
| date | DATE | NOT NULL | 集計日 |
| provider | VARCHAR(50) | NOT NULL | |
| model | VARCHAR(100) | NOT NULL | |
| total_requests | INT | NOT NULL DEFAULT 0 | |
| total_input_tokens | BIGINT | NOT NULL DEFAULT 0 | |
| total_output_tokens | BIGINT | NOT NULL DEFAULT 0 | |
| total_cost_usd | DECIMAL(12, 6) | NOT NULL DEFAULT 0 | |
| avg_latency_ms | INT | | |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | |

ユニーク: (tenant_id, project_id, date, provider, model)

インデックス:
- `idx_usage_daily_tenant_date` ON (tenant_id, date DESC)

### usage_budgets

予算制限とアラートしきい値。

| カラム | 型 | 制約 | 説明 |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| project_id | UUID | FK projects(id) | テナント全体の予算の場合は NULL |
| budget_type | VARCHAR(50) | NOT NULL | monthly, daily |
| budget_amount_usd | DECIMAL(12, 2) | NOT NULL | 予算上限 |
| alert_threshold | DECIMAL(3, 2) | NOT NULL DEFAULT 0.80 | 0.0-1.0、アラートをトリガー |
| enabled | BOOLEAN | NOT NULL DEFAULT TRUE | |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | |

インデックス:
- `idx_usage_budgets_tenant` ON (tenant_id)
- `idx_usage_budgets_project` ON (project_id) WHERE project_id IS NOT NULL

### secrets

| カラム | 型 | 制約 | 説明 |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| name | VARCHAR(255) | NOT NULL | |
| encrypted_value | TEXT | NOT NULL | AES-256 暗号化 |
| created_by | UUID | FK users(id) | |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |

ユニーク: (tenant_id, name)

### audit_logs

| カラム | 型 | 制約 | 説明 |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| actor_id | UUID | | アクションを実行したユーザー |
| actor_email | VARCHAR(255) | | |
| action | VARCHAR(100) | NOT NULL | create, update, delete, publish, execute |
| resource_type | VARCHAR(100) | NOT NULL | project, run, secret |
| resource_id | UUID | | |
| metadata | JSONB | | 追加コンテキスト |
| ip_address | INET | | |
| user_agent | TEXT | | |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |

インデックス:
- `idx_audit_logs_tenant` ON (tenant_id)
- `idx_audit_logs_created` ON (created_at)

### oauth2_providers

OAuth2 プロバイダー設定（プリセットおよびカスタム）。

| カラム | 型 | 制約 | 説明 |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT gen_random_uuid() | |
| slug | VARCHAR(50) | NOT NULL, UNIQUE | プロバイダー識別子 (google, github 等) |
| name | VARCHAR(100) | NOT NULL | 表示名 |
| icon_url | TEXT | | アイコン URL |
| authorization_url | TEXT | NOT NULL | OAuth2 認可エンドポイント |
| token_url | TEXT | NOT NULL | トークンエンドポイント |
| revoke_url | TEXT | | トークン無効化エンドポイント |
| userinfo_url | TEXT | | ユーザー情報エンドポイント |
| pkce_required | BOOLEAN | DEFAULT false | PKCE 必須フラグ |
| default_scopes | TEXT[] | DEFAULT '{}' | デフォルトスコープ |
| documentation_url | TEXT | | ドキュメント URL |
| is_preset | BOOLEAN | DEFAULT false | プリセットプロバイダーフラグ |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |

プリセットプロバイダー: Google, GitHub, Slack, Notion, Linear, Microsoft, Discord, Atlassian

### oauth2_apps

テナント固有の OAuth2 アプリケーション設定。

| カラム | 型 | 制約 | 説明 |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT gen_random_uuid() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| provider_id | UUID | FK oauth2_providers(id), NOT NULL | |
| encrypted_client_id | BYTEA | NOT NULL | AES-256-GCM 暗号化 |
| encrypted_client_secret | BYTEA | NOT NULL | AES-256-GCM 暗号化 |
| client_id_nonce | BYTEA | NOT NULL | 暗号化ノンス |
| client_secret_nonce | BYTEA | NOT NULL | 暗号化ノンス |
| custom_scopes | TEXT[] | | カスタムスコープ |
| redirect_uri | TEXT | | リダイレクト URI |
| status | VARCHAR(20) | DEFAULT 'active' | active, disabled |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |

ユニーク: (tenant_id, provider_id)

### oauth2_connections

個別の OAuth2 トークン接続。

| カラム | 型 | 制約 | 説明 |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT gen_random_uuid() | |
| credential_id | UUID | FK credentials(id), NOT NULL | |
| oauth2_app_id | UUID | FK oauth2_apps(id), NOT NULL | |
| encrypted_access_token | BYTEA | | 暗号化アクセストークン |
| encrypted_refresh_token | BYTEA | | 暗号化リフレッシュトークン |
| access_token_nonce | BYTEA | | |
| refresh_token_nonce | BYTEA | | |
| token_type | VARCHAR(50) | DEFAULT 'Bearer' | |
| access_token_expires_at | TIMESTAMPTZ | | |
| refresh_token_expires_at | TIMESTAMPTZ | | |
| state | VARCHAR(255) | | OAuth2 フロー用 CSRF state |
| code_verifier | TEXT | | PKCE コードベリファイア |
| account_id | TEXT | | 外部アカウント ID |
| account_email | TEXT | | 外部アカウントメール |
| account_name | TEXT | | 外部アカウント名 |
| raw_userinfo | JSONB | | userinfo エンドポイントからの生データ |
| status | VARCHAR(20) | DEFAULT 'pending' | pending, connected, expired, revoked, error |
| last_refresh_at | TIMESTAMPTZ | | |
| last_used_at | TIMESTAMPTZ | | |
| error_message | TEXT | | |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |

インデックス:
- `idx_oauth2_connections_credential` ON (credential_id)
- `idx_oauth2_connections_status` ON (status)
- `idx_oauth2_connections_state` ON (state) WHERE state IS NOT NULL
- `idx_oauth2_connections_expires` ON (access_token_expires_at) WHERE status = 'connected'

### credential_shares

認証情報のユーザー/プロジェクト間共有設定。

| カラム | 型 | 制約 | 説明 |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT gen_random_uuid() | |
| credential_id | UUID | FK credentials(id), NOT NULL | |
| shared_with_user_id | UUID | FK users(id) | ユーザーとの共有時 |
| shared_with_project_id | UUID | FK projects(id) | プロジェクトとの共有時 |
| permission | VARCHAR(20) | NOT NULL DEFAULT 'use' | use, edit, admin |
| shared_by_user_id | UUID | FK users(id), NOT NULL | 共有元ユーザー |
| note | TEXT | | 共有メモ |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| expires_at | TIMESTAMPTZ | | 有効期限 |

制約: shared_with_user_id または shared_with_project_id のいずれか一方のみ設定

ユニーク: (credential_id, shared_with_user_id), (credential_id, shared_with_project_id)

### copilot_sessions

AI Copilot セッション。対話型ワークフロー作成/改善用。

| カラム | 型 | 制約 | 説明 |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT gen_random_uuid() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| user_id | VARCHAR(255) | NOT NULL | |
| context_project_id | UUID | FK projects(id) | セッションのコンテキストプロジェクト |
| mode | VARCHAR(50) | NOT NULL DEFAULT 'create' | create, enhance, explain |
| title | VARCHAR(200) | | セッションタイトル |
| status | VARCHAR(50) | NOT NULL DEFAULT 'hearing' | hearing, building, reviewing, refining, completed, abandoned |
| hearing_phase | VARCHAR(50) | NOT NULL DEFAULT 'analysis' | analysis, proposal, completed |
| hearing_progress | INTEGER | NOT NULL DEFAULT 0 | 0-100 |
| spec | JSONB | | WorkflowSpec DSL |
| project_id | UUID | FK projects(id) | 生成されたプロジェクト ID |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |

インデックス:
- `idx_copilot_sessions_tenant` ON (tenant_id)
- `idx_copilot_sessions_user` ON (user_id)
- `idx_copilot_sessions_context_project` ON (context_project_id)

### copilot_messages

Copilot セッション内のメッセージ。

| カラム | 型 | 制約 | 説明 |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT gen_random_uuid() | |
| session_id | UUID | FK copilot_sessions(id), NOT NULL | |
| role | VARCHAR(20) | NOT NULL | user, assistant, system |
| content | TEXT | NOT NULL | メッセージ内容 |
| phase | VARCHAR(50) | | メッセージ作成時のフェーズ |
| extracted_data | JSONB | | ユーザーメッセージから抽出されたデータ |
| suggested_questions | JSONB | | 提案されたフォローアップ質問 |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |

インデックス:
- `idx_copilot_messages_session` ON (session_id)

## 正規クエリパターン（必須）

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

## クエリパターン

### プロジェクト一覧取得（テナント分離あり）

```sql
SELECT *
FROM projects
WHERE tenant_id = $1
  AND deleted_at IS NULL
ORDER BY updated_at DESC
LIMIT $2 OFFSET $3;
```

### プロジェクトとステップ・エッジの取得

```sql
-- プロジェクト
SELECT * FROM projects WHERE id = $1 AND tenant_id = $2;

-- ステップ（複数の Start ブロックを含む）
SELECT * FROM steps WHERE project_id = $1 ORDER BY created_at;

-- エッジ
SELECT * FROM edges WHERE project_id = $1;
```

### プロジェクトの Start ブロック取得

```sql
-- トリガー設定を含むすべての Start ブロックを取得
SELECT id, name, config
FROM steps
WHERE project_id = $1
  AND type = 'start'
ORDER BY created_at;
```

### Run と StepRuns の取得

```sql
SELECT r.*, json_agg(sr.*) AS step_runs
FROM runs r
LEFT JOIN step_runs sr ON sr.run_id = r.id
WHERE r.id = $1
GROUP BY r.id;
```

### 実行待ちのアクティブスケジュール検索

```sql
SELECT *
FROM schedules
WHERE status = 'active'
  AND next_run_at <= NOW()
ORDER BY next_run_at;
```

### ステータス別 Run カウント（ダッシュボード用）

```sql
SELECT status, COUNT(*) as count
FROM runs
WHERE tenant_id = $1
  AND created_at >= $2
GROUP BY status;
```

### 期間別使用量サマリー取得

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

### モデル別使用量取得

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

### ベクトル類似検索（RAG）

⚠️ **重要**: すべてのベクトルクエリは `tenant_id` フィルタを必須とする。

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

## マイグレーションコマンド

```bash
# マイグレーション適用（golang-migrate 使用）
docker compose exec api migrate -path /migrations -database "$DATABASE_URL" up

# 直前のマイグレーションをロールバック
docker compose exec api migrate -path /migrations -database "$DATABASE_URL" down 1

# バージョン強制（危険）
docker compose exec api migrate -path /migrations -database "$DATABASE_URL" force VERSION
```

## ソフトデリートパターン

テナント所有のすべてのテーブルは `deleted_at` カラムによるソフトデリートをサポート:

```sql
-- 「削除」
UPDATE projects SET deleted_at = NOW() WHERE id = $1;

-- クエリ（削除済みを除外）
SELECT * FROM projects WHERE deleted_at IS NULL;

-- ハードデリート（管理者のみ）
DELETE FROM projects WHERE id = $1;
```

## マルチテナンシーパターン

すべてのクエリに `tenant_id` を含める必要があります:

```go
func (r *ProjectRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Project, error) {
    return r.db.QueryRow(ctx,
        `SELECT * FROM projects WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
        id, tenantID,
    ).Scan(...)
}
```

## JSONB カラムの使用法

| テーブル | カラム | 内容 |
|-------|--------|---------|
| tenants | settings | `{"data_retention_days": 30, "max_concurrent_runs": 10}` |
| projects | variables | プロジェクトレベル変数 |
| steps | config | ステップタイプ固有の設定（Start ブロックは trigger_type, trigger_config, input_schema, output_schema を含む） |
| runs | input | 実行入力 |
| runs | output | 実行結果 |
| audit_logs | metadata | アクション固有の詳細 |

## コネクションプール設定

```go
config := pgxpool.Config{
    MaxConns:          25,
    MinConns:          5,
    MaxConnLifetime:   time.Hour,
    MaxConnIdleTime:   30 * time.Minute,
    HealthCheckPeriod: time.Minute,
}
```

## バックアップ

```bash
# ダンプ
pg_dump -h localhost -U postgres ai_orchestration > backup.sql

# リストア
psql -h localhost -U postgres ai_orchestration < backup.sql
```

## 関連ドキュメント

- [BACKEND.md](./BACKEND.md) - リポジトリインターフェースとデータアクセスパターン
- [API.md](./API.md) - データベースとやり取りする API エンドポイント
- [UNIFIED_BLOCK_MODEL.md](./designs/UNIFIED_BLOCK_MODEL.md) - ブロック定義スキーマ
- [BLOCK_REGISTRY.md](./BLOCK_REGISTRY.md) - ブロック定義テーブル（RAG ブロック含む）
- [RAG_IMPLEMENTATION_PLAN.md](./plans/RAG_IMPLEMENTATION_PLAN.md) - RAG 機能の設計書
