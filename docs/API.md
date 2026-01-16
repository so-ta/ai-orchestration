# APIリファレンス

REST APIエンドポイント、リクエスト/レスポンススキーマ、認証についてのドキュメント。

> **移行メモ (2026-01)**: WorkflowはProjectに名称変更されました。Projectは複数のStartブロックをサポートし、それぞれ独自のトリガー設定を持ちます。webhooksテーブルは削除され、Webhook機能はStartブロックの`trigger_config`で設定するようになりました。

## クイックリファレンス

| 項目 | 値 |
|------|-------|
| ベースURL | `/api/v1` |
| 認証 | Bearer JWT |
| Content-Type | `application/json` |
| テナント (開発) | `X-Tenant-ID` ヘッダー |
| テナント (本番) | JWT クレーム |
| ヘルスチェック | `GET /health`, `GET /ready` |

## ヘッダー

| ヘッダー | 必須 | 説明 |
|--------|----------|-------------|
| `Authorization` | はい* | `Bearer <token>` (*AUTH_ENABLED=false以外) |
| `Content-Type` | はい | `application/json` |
| `X-Tenant-ID` | 開発のみ | UUID、AUTH_ENABLED=false時に必須 |
| `X-Request-ID` | いいえ | トレーシング用UUID |

## エラーレスポンス

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "人間が読めるメッセージ",
    "details": {}
  }
}
```

| コード | HTTP | 説明 |
|------|------|-------------|
| `UNAUTHORIZED` | 401 | 無効/欠落トークン |
| `FORBIDDEN` | 403 | 権限不足 |
| `NOT_FOUND` | 404 | リソースが見つからない |
| `VALIDATION_ERROR` | 400 | 無効なリクエストボディ |
| `SCHEMA_VALIDATION_ERROR` | 400 | 入力がStartブロックのinput_schemaと一致しない |
| `CONFLICT` | 409 | リソースの競合 |
| `INVALID_STATE` | 409 | 操作に無効な状態（実行がキャンセル/再開不可、スケジュールが無効など） |
| `INTERNAL_ERROR` | 500 | サーバーエラー |
| `RATE_LIMIT_EXCEEDED` | 429 | レート制限超過 |

### スキーマ検証エラーレスポンス

入力データがStartブロックの`input_schema`と一致しない場合、APIは詳細な検証エラーを返します：

```json
{
  "error": {
    "code": "SCHEMA_VALIDATION_ERROR",
    "message": "入力検証に失敗しました",
    "details": {
      "errors": [
        {
          "field": "email",
          "message": "emailは必須です"
        },
        {
          "field": "age",
          "message": "ageはinteger型である必要があります"
        }
      ]
    }
  }
}
```

このエラーは以下で返されます：
- `POST /projects/{project_id}/runs` - 実行入力がStartブロックのinput_schemaと一致しない場合
- Webhookトリガー - Webhookペイロード（Startブロックのtrigger_config内のinput_mapping適用後）がinput_schemaと一致しない場合

---

## レート制限

APIリクエストは公平な使用を確保するため、複数のスコープでレート制限されます。

### レート制限スコープ

| スコープ | デフォルト制限 | ウィンドウ | 説明 |
|-------|--------------|--------|-------------|
| `tenant` | 1000 req | 1分 | 全エンドポイントでのテナントごとの制限 |
| `project` | 100 req | 1分 | 実行作成のプロジェクトごとの制限 |
| `webhook` | 60 req | 1分 | トリガーエンドポイントのWebhookキーごとの制限 |

### レート制限ヘッダー

すべてのレスポンスにレート制限ヘッダーが含まれます：

```
X-RateLimit-tenant-Limit: 1000
X-RateLimit-tenant-Remaining: 999
X-RateLimit-tenant-Reset: 1704067200
```

### レート制限エラーレスポンス

```json
{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "tenantスコープのレート制限を超過しました",
    "retry_at": "2024-01-01T00:00:00Z",
    "limit": 1000,
    "scope": "tenant"
  }
}
```

### 設定

レート制限は環境変数で設定できます：

| 変数 | デフォルト | 説明 |
|----------|---------|-------------|
| `RATE_LIMIT_ENABLED` | `true` | レート制限の有効化/無効化 |
| `RATE_LIMIT_TENANT` | `1000` | テナントごとの1分あたりのリクエスト数 |
| `RATE_LIMIT_PROJECT` | `100` | プロジェクトごとの1分あたりのリクエスト数 |
| `RATE_LIMIT_WEBHOOK` | `60` | Webhookキーごとの1分あたりのリクエスト数 |

---

## Projects

Projects（旧Workflows）はDAG定義の主要な組織単位です。プロジェクトは複数のStartブロックを持つことができ、それぞれ独自のトリガータイプ（manual、schedule、webhook）を持ちます。

### 一覧取得
```
GET /projects
```

クエリ：
| パラメータ | 型 | デフォルト | 説明 |
|-------|------|---------|-------------|
| `status` | string | - | `draft` または `published` |
| `page` | int | 1 | ページ番号 |
| `limit` | int | 20 | 1ページあたりの件数（最大100） |

レスポンス `200`：
```json
{
  "data": [
    {
      "id": "uuid",
      "name": "string",
      "description": "string",
      "status": "draft|published",
      "version": 1,
      "variables": {},
      "created_at": "ISO8601",
      "updated_at": "ISO8601"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 100
  }
}
```

### 作成
```
POST /projects
```

リクエスト：
```json
{
  "name": "string (必須)",
  "description": "string",
  "variables": {}
}
```

> **注意**: `input_schema`と`output_schema`はプロジェクトレベルの`variables`に置き換えられました。入出力スキーマはStartブロックごとに定義されるようになりました。

レスポンス `201`：
```json
{
  "id": "uuid",
  "name": "string",
  "description": "string",
  "status": "draft",
  "version": 1,
  "variables": {},
  "created_at": "ISO8601",
  "updated_at": "ISO8601"
}
```

### 取得
```
GET /projects/{id}
```

レスポンス `200`: 作成レスポンスと同じ

### 更新
```
PUT /projects/{id}
```

制約: `draft`ステータスのみ

リクエスト：
```json
{
  "name": "string",
  "description": "string",
  "variables": {}
}
```

レスポンス `200`: 更新されたプロジェクト

### 削除
```
DELETE /projects/{id}
```

レスポンス `204`: コンテンツなし

### 公開
```
POST /projects/{id}/publish
```

制約: `draft`ステータスである必要がある

レスポンス `200`：
```json
{
  "id": "uuid",
  "status": "published",
  "version": 2,
  "published_at": "ISO8601"
}
```

---

## Steps

### 一覧取得
```
GET /projects/{project_id}/steps
```

レスポンス `200`：
```json
{
  "data": [
    {
      "id": "uuid",
      "project_id": "uuid",
      "name": "string",
      "type": "start|llm|tool|condition|map|join|subflow",
      "config": {},
      "position": {"x": 0, "y": 0},
      "created_at": "ISO8601",
      "updated_at": "ISO8601"
    }
  ]
}
```

### 作成
```
POST /projects/{project_id}/steps
```

リクエスト：
```json
{
  "name": "string (必須)",
  "type": "start|llm|tool|condition|map|join|subflow (必須)",
  "config": {},
  "position": {"x": 0, "y": 0}
}
```

タイプ別の設定：

**start** (プロジェクトごとに複数のStartブロックをサポート)：
```json
{
  "trigger_type": "manual|schedule|webhook",
  "trigger_config": {
    "input_schema": {},
    "input_mapping": {},
    "webhook_secret": "string",
    "cron": "0 9 * * *",
    "timezone": "Asia/Tokyo"
  },
  "input_schema": {},
  "output_schema": {}
}
```

> **注意**: 各Startブロックは異なるトリガータイプを持つことができます。WebhookとScheduleの設定は、別テーブルではなくStartブロックの`trigger_config`の一部になりました。

**llm**：
```json
{
  "provider": "openai|anthropic",
  "model": "gpt-4|claude-3-opus-20240229",
  "prompt": "{{input.field}} テンプレートを含む文字列",
  "temperature": 0.7,
  "max_tokens": 1000
}
```

**tool**：
```json
{
  "adapter_id": "mock|http|openai|anthropic",
  "...アダプター固有"
}
```

**condition**：
```json
{
  "expression": "$.field > 10"
}
```

**map**：
```json
{
  "input_path": "$.items",
  "parallel": true,
  "max_concurrency": 5
}
```

レスポンス `201`: 作成されたステップ

### 更新
```
PUT /projects/{project_id}/steps/{step_id}
```

リクエスト: 作成と同じ
レスポンス `200`: 更新されたステップ

### 削除
```
DELETE /projects/{project_id}/steps/{step_id}
```

レスポンス `204`: コンテンツなし

---

## Edges

### 一覧取得
```
GET /projects/{project_id}/edges
```

レスポンス `200`：
```json
{
  "data": [
    {
      "id": "uuid",
      "project_id": "uuid",
      "source_step_id": "uuid",
      "target_step_id": "uuid",
      "condition": "string (オプション)",
      "created_at": "ISO8601"
    }
  ]
}
```

### 作成
```
POST /projects/{project_id}/edges
```

リクエスト：
```json
{
  "source_step_id": "uuid (必須)",
  "target_step_id": "uuid (必須)",
  "condition": "$.success == true"
}
```

レスポンス `201`: 作成されたエッジ

検証：
- 循環接続を拒否
- ソースとターゲットが存在する必要がある

### 削除
```
DELETE /projects/{project_id}/edges/{edge_id}
```

レスポンス `204`: コンテンツなし

---

## Block Groups

Block Groupsは複数のステップをグループ化する制御フロー構造です。

> **更新**: 2026-01-15
> 4タイプのみに簡略化: `parallel`, `try_catch`, `foreach`, `while`
> 削除: `if_else` (`condition`ブロックを使用), `switch_case` (`switch`ブロックを使用)
> すべてのグループは`body`ロールのみを使用し、変換には`pre_process`/`post_process`を使用。

### グループタイプ

| タイプ | 説明 | 設定 |
|------|-------------|--------|
| `parallel` | 複数のフローを並列実行 | `max_concurrent`, `fail_fast` |
| `try_catch` | リトライサポート付きエラーハンドリング | `retry_count`, `retry_delay_ms` |
| `foreach` | 配列要素の反復処理 | `input_path`, `parallel`, `max_workers` |
| `while` | 条件ベースのループ | `condition`, `max_iterations`, `do_while` |

### 一覧取得
```
GET /projects/{project_id}/block-groups
```

レスポンス `200`：
```json
{
  "data": [
    {
      "id": "uuid",
      "project_id": "uuid",
      "name": "並列タスク",
      "type": "parallel",
      "config": { "max_concurrent": 10, "fail_fast": false },
      "parent_group_id": null,
      "pre_process": "return { ...input, timestamp: Date.now() };",
      "post_process": "return { result: output.data };",
      "position": { "x": 100, "y": 200 },
      "size": { "width": 400, "height": 300 }
    }
  ]
}
```

### 作成
```
POST /projects/{project_id}/block-groups
```

リクエスト：
```json
{
  "name": "並列タスク",
  "type": "parallel|try_catch|foreach|while",
  "config": {},
  "parent_group_id": null,
  "pre_process": "return input;",
  "post_process": "return output;",
  "position": { "x": 100, "y": 200 },
  "size": { "width": 400, "height": 300 }
}
```

| フィールド | 型 | 必須 | 説明 |
|-------|------|----------|-------------|
| `name` | string | はい | 表示名 |
| `type` | string | はい | `parallel`, `try_catch`, `foreach`, `while` のいずれか |
| `config` | object | いいえ | タイプ固有の設定 |
| `parent_group_id` | uuid | いいえ | ネストされたグループ用 |
| `pre_process` | string | いいえ | JSコード: 外部IN → 内部IN |
| `post_process` | string | いいえ | JSコード: 内部OUT → 外部OUT |
| `position` | object | はい | `{ x, y }` 座標 |
| `size` | object | はい | `{ width, height }` 寸法 |

レスポンス `201`: 作成されたブロックグループ

### 取得
```
GET /projects/{project_id}/block-groups/{group_id}
```

レスポンス `200`: ブロックグループの詳細

### 更新
```
PUT /projects/{project_id}/block-groups/{group_id}
```

リクエスト：
```json
{
  "name": "更新された名前",
  "config": { "max_concurrent": 5 },
  "pre_process": "return { ...input, modified: true };",
  "post_process": "return output;",
  "position": { "x": 150, "y": 250 },
  "size": { "width": 500, "height": 400 }
}
```

レスポンス `200`: 更新されたブロックグループ

### 削除
```
DELETE /projects/{project_id}/block-groups/{group_id}
```

レスポンス `204`: コンテンツなし

### グループにステップを追加
```
POST /projects/{project_id}/block-groups/{group_id}/steps
```

リクエスト：
```json
{
  "step_id": "uuid",
  "group_role": "body"
}
```

> **注意**: `body`ロールのみがサポートされています。他のロールは削除されました。

レスポンス `200`: 更新されたステップ

**制限:**
- `start`ステップはブロックグループに追加できません（`400 VALIDATION_ERROR`を返す）

**発生する可能性のあるエラー:**

| コード | メッセージ | 説明 |
|------|---------|-------------|
| VALIDATION_ERROR | このステップタイプはブロックグループに追加できません | Startノードはグループに入れられない |
| VALIDATION_ERROR | 無効なグループロール | `body`ロールのみが有効 |
| NOT_FOUND | ブロックグループが見つかりません | ブロックグループが存在しない |
| CONFLICT | 公開済みプロジェクトは編集できません | プロジェクトが公開済み |

### グループ内のステップを取得
```
GET /projects/{project_id}/block-groups/{group_id}/steps
```

レスポンス `200`: ステップの配列

### グループからステップを削除
```
DELETE /projects/{project_id}/block-groups/{group_id}/steps/{step_id}
```

レスポンス `200`: 更新されたステップ（block_group_idがnull）

---

## Runs

### 実行
```
POST /projects/{project_id}/runs
```

リクエスト：
```json
{
  "input": {},
  "start_step_id": "uuid",
  "triggered_by": "manual|test|webhook|schedule|internal",
  "version": 0
}
```

| フィールド | 型 | デフォルト | 説明 |
|-------|------|---------|-------------|
| `input` | object | `{}` | 実行の入力データ |
| `start_step_id` | uuid | - | **複数Startプロジェクトでは必須**: トリガーするStartブロックを指定 |
| `triggered_by` | string | `manual` | トリガータイプ: `manual`, `test`, `webhook`, `schedule`, `internal` |
| `version` | int | 0 | 実行するプロジェクトバージョン（0 = 最新） |
| `mode` | string | - | **非推奨**: 代わりに`triggered_by`を使用（`mode: "test"`は`triggered_by: "test"`にマップ） |

> **注意**: プロジェクトは複数のStartブロックを持つことができます。実行を実行する際、プロジェクトに複数のStartブロックがある場合は`start_step_id`でどのStartブロックを使用するか指定する必要があります。

レスポンス `201`：
```json
{
  "id": "uuid",
  "project_id": "uuid",
  "project_version": 1,
  "start_step_id": "uuid",
  "status": "pending",
  "triggered_by": "manual",
  "run_number": 1,
  "created_at": "ISO8601"
}
```

### プロジェクト別一覧取得
```
GET /projects/{project_id}/runs
```

クエリ：
| パラメータ | 型 | デフォルト |
|-------|------|---------|
| `status` | string | - |
| `start_step_id` | uuid | - |
| `page` | int | 1 |
| `limit` | int | 20 |

レスポンス `200`: ページネーションされた実行一覧

### 取得
```
GET /runs/{run_id}
```

レスポンス `200`：
```json
{
  "id": "uuid",
  "project_id": "uuid",
  "project_version": 1,
  "start_step_id": "uuid",
  "status": "completed",
  "mode": "production",
  "trigger_type": "manual",
  "input": {},
  "output": {},
  "error": "string (失敗時)",
  "started_at": "ISO8601",
  "completed_at": "ISO8601",
  "duration_ms": 1000,
  "step_runs": [
    {
      "id": "uuid",
      "step_id": "uuid",
      "step_name": "string",
      "status": "completed",
      "attempt": 1,
      "input": {},
      "output": {},
      "error": "",
      "started_at": "ISO8601",
      "completed_at": "ISO8601",
      "duration_ms": 500
    }
  ]
}
```

### キャンセル
```
POST /runs/{run_id}/cancel
```

レスポンス `200`: `status: cancelled`で更新された実行

**エラーレスポンス:**

| コード | HTTP | 条件 |
|------|------|-----------|
| `NOT_FOUND` | 404 | 実行が存在しない |
| `INVALID_STATE` | 409 | 実行がキャンセル可能な状態にない（すでに完了またはキャンセル済み等） |

### ステップから再開
```
POST /runs/{run_id}/resume
```

特定のステップからすべての下流ステップまで実行を再開します。

リクエスト：
```json
{
  "from_step_id": "uuid (必須)",
  "input_override": {}
}
```

制約: 実行は`completed`または`failed`ステータスである必要がある

レスポンス `202`：
```json
{
  "data": {
    "run_id": "uuid",
    "from_step_id": "uuid",
    "steps_to_execute": ["uuid", "uuid", "uuid"]
  }
}
```

**エラーレスポンス:**

| コード | HTTP | 条件 |
|------|------|-----------|
| `NOT_FOUND` | 404 | 実行が存在しない |
| `INVALID_STATE` | 409 | 実行が再開可能な状態にない（`completed`または`failed`である必要がある） |

### 単一ステップを実行
```
POST /runs/{run_id}/steps/{step_id}/execute
```

既存の実行から単一ステップのみを再実行します。

リクエスト：
```json
{
  "input": {}
}
```

制約: 実行は`completed`または`failed`ステータスである必要がある

レスポンス `202`：
```json
{
  "data": {
    "id": "uuid",
    "run_id": "uuid",
    "step_id": "uuid",
    "step_name": "string",
    "status": "pending",
    "attempt": 2
  }
}
```

### ステップ履歴を取得
```
GET /runs/{run_id}/steps/{step_id}/history
```

実行内の特定のステップのすべての実行履歴を取得します。

レスポンス `200`：
```json
{
  "data": [
    {
      "id": "uuid",
      "run_id": "uuid",
      "step_id": "uuid",
      "step_name": "string",
      "status": "completed",
      "attempt": 2,
      "input": {},
      "output": {},
      "error": "",
      "started_at": "ISO8601",
      "completed_at": "ISO8601",
      "duration_ms": 500
    },
    {
      "id": "uuid",
      "run_id": "uuid",
      "step_id": "uuid",
      "step_name": "string",
      "status": "failed",
      "attempt": 1,
      "input": {},
      "output": {},
      "error": "エラーメッセージ",
      "started_at": "ISO8601",
      "completed_at": "ISO8601",
      "duration_ms": 200
    }
  ]
}
```

### ステップをインラインでテスト
```
POST /projects/{project_id}/steps/{step_id}/test
```

既存の実行を必要とせずに単一ステップをテストします。一時的な実行を作成し、指定されたステップのみを実行します。

リクエスト：
```json
{
  "input": {}
}
```

レスポンス `202`：
```json
{
  "data": {
    "run": {
      "id": "uuid",
      "project_id": "uuid",
      "status": "running",
      "triggered_by": "test"
    },
    "step_run": {
      "id": "uuid",
      "run_id": "uuid",
      "step_id": "uuid",
      "step_name": "string",
      "status": "pending",
      "attempt": 1
    }
  }
}
```

---

## Schedules

スケジュールはプロジェクト内の特定のStartブロックにリンクされるようになりました。スケジュールがトリガーされると、指定されたStartブロックを実行します。

### 一覧取得
```
GET /projects/{project_id}/schedules
```

レスポンス `200`：
```json
{
  "data": [
    {
      "id": "uuid",
      "project_id": "uuid",
      "start_step_id": "uuid",
      "name": "string",
      "cron": "0 9 * * *",
      "timezone": "Asia/Tokyo",
      "input": {},
      "enabled": true,
      "next_run_at": "ISO8601",
      "created_at": "ISO8601"
    }
  ]
}
```

### 作成
```
POST /projects/{project_id}/schedules
```

リクエスト：
```json
{
  "name": "string (必須)",
  "start_step_id": "uuid (必須)",
  "cron": "0 9 * * * (必須)",
  "timezone": "Asia/Tokyo",
  "input": {},
  "enabled": true,
  "retry_policy": {
    "max_attempts": 3,
    "delay_seconds": 60
  }
}
```

> **注意**: `start_step_id`は必須で、プロジェクト内のStartブロックを参照する必要があります。これにより、スケジュールが発火したときにどのStartブロックがトリガーされるかが決まります。

レスポンス `201`: 作成されたスケジュール

### 更新
```
PUT /schedules/{schedule_id}
```

レスポンス `200`: 更新されたスケジュール

### 削除
```
DELETE /schedules/{schedule_id}
```

レスポンス `204`: コンテンツなし

---

## Webhooks

> **移行メモ**: スタンドアロンのwebhooksテーブルは削除されました。Webhook機能はStartブロックで`trigger_type: "webhook"`と`trigger_config`を通じて直接設定されるようになりました。

### Webhook設定 (Startブロック経由)

Webhookトリガーを作成するには、Startブロックを以下のように作成または更新します：

```json
{
  "name": "Webhookトリガー",
  "type": "start",
  "config": {
    "trigger_type": "webhook",
    "trigger_config": {
      "webhook_secret": "whsec_xxx",
      "input_mapping": {
        "event": "$.action",
        "repo": "$.repository.name"
      }
    },
    "input_schema": {}
  }
}
```

### Webhook受信 (外部)
```
POST /projects/{project_id}/webhook/{start_step_id}
```

ヘッダー：
| ヘッダー | 必須 | 説明 |
|--------|----------|-------------|
| `X-Webhook-Signature` | はい | `sha256=<hmac>` |
| `X-Webhook-Timestamp` | はい | Unixタイムスタンプ |
| `X-Idempotency-Key` | いいえ | 重複排除キー |

リクエスト: 任意のJSONペイロード

レスポンス `200`：
```json
{
  "run_id": "uuid",
  "status": "pending"
}
```

---

## Blocks

ワークフローステップ用のブロック定義。ブロックはシステムブロック（組み込み）またはテナント固有のカスタムブロックにできます。ブロックは再利用可能な設定のための継承をサポートします。

### 一覧取得
```
GET /blocks
```

クエリ：
| パラメータ | 型 | 説明 |
|-------|------|-------------|
| `category` | string | カテゴリでフィルタ: `ai`, `flow`, `apps`, `custom` |
| `subcategory` | string | サブカテゴリでフィルタ: `chat`, `rag`, `routing`, `branching`, `data`, `control`, `utility`, `slack`, `discord`, `notion`, `github`, `google`, `linear`, `email`, `web` |
| `enabled` | bool | 有効なブロックのみをフィルタ |

レスポンス `200`：
```json
{
  "blocks": [
    {
      "id": "uuid",
      "tenant_id": "uuid",
      "slug": "llm",
      "name": "LLM呼び出し",
      "description": "LLMプロバイダーを呼び出す",
      "category": "ai",
      "subcategory": "chat",
      "icon": "brain",
      "config_schema": {},
      "input_schema": {},
      "output_schema": {},
      "input_ports": [],
      "output_ports": [],
      "error_codes": [],
      "code": "...",
      "ui_config": {},
      "is_system": true,
      "version": 1,
      "parent_block_id": null,
      "config_defaults": {},
      "pre_process": "",
      "post_process": "",
      "internal_steps": [],
      "pre_process_chain": [],
      "post_process_chain": [],
      "resolved_code": "",
      "resolved_config_defaults": {},
      "enabled": true,
      "created_at": "ISO8601",
      "updated_at": "ISO8601"
    }
  ]
}
```

### 取得
```
GET /blocks/{slug}
```

レスポンス `200`: 単一ブロック定義

### 作成
```
POST /blocks
```

リクエスト：
```json
{
  "slug": "string (必須)",
  "name": "string (必須)",
  "description": "string",
  "category": "ai|flow|apps|custom (必須)",
  "subcategory": "chat|rag|routing|branching|data|control|utility|slack|discord|notion|github|google|linear|email|web (オプション)",
  "icon": "string",
  "config_schema": {},
  "input_schema": {},
  "output_schema": {},
  "code": "string",
  "ui_config": {},
  "parent_block_id": "uuid (オプション)",
  "config_defaults": {},
  "pre_process": "string",
  "post_process": "string",
  "internal_steps": [
    {
      "type": "block-slug",
      "config": {},
      "output_key": "step1"
    }
  ]
}
```

**ブロック継承/拡張フィールド:**

| フィールド | 型 | 説明 |
|-------|------|-------------|
| `parent_block_id` | uuid | 継承用の親ブロックへの参照（コードを持つブロックのみ継承可能） |
| `config_defaults` | object | 親のconfig_schemaのデフォルト値（親のデフォルトを上書き） |
| `pre_process` | string | メインコードの前に実行されるJavaScriptコード（入力変換用） |
| `post_process` | string | メインコードの後に実行されるJavaScriptコード（出力変換用） |
| `internal_steps` | array | ブロック内で順次実行されるステップの配列 |

**解決済みフィールド（継承ブロック用にバックエンドで設定）:**

| フィールド | 型 | 説明 |
|-------|------|-------------|
| `pre_process_chain` | string[] | preProcessコードのチェーン（子 → ルート） |
| `post_process_chain` | string[] | postProcessコードのチェーン（ルート → 子） |
| `resolved_code` | string | ルート祖先からのコード |
| `resolved_config_defaults` | object | 継承チェーンからマージされた設定デフォルト |

レスポンス `201`: 作成されたブロック

**検証エラー:**

| コード | メッセージ | 説明 |
|------|---------|-------------|
| VALIDATION_ERROR | 循環継承が検出されました | ブロックが循環継承を作成する |
| VALIDATION_ERROR | 継承深度が最大制限を超えました | 継承チェーンが10レベルを超える |
| VALIDATION_ERROR | 親ブロックは継承できません（コードなし） | 親ブロックに継承するコードがない |
| CONFLICT | このslugのブロックはすでに存在します | Slugがすでに使用されている |

### 更新
```
PUT /blocks/{slug}
```

リクエスト：
```json
{
  "name": "string",
  "description": "string",
  "icon": "string",
  "config_schema": {},
  "input_schema": {},
  "output_schema": {},
  "code": "string",
  "ui_config": {},
  "enabled": true,
  "parent_block_id": "uuid (クリアするにはnull)",
  "config_defaults": {},
  "pre_process": "string",
  "post_process": "string",
  "internal_steps": []
}
```

レスポンス `200`: 更新されたブロック

### 削除
```
DELETE /blocks/{slug}
```

レスポンス `204`: コンテンツなし

---

## Adapters

### 一覧取得
```
GET /adapters
```

レスポンス `200`：
```json
{
  "data": [
    {
      "id": "mock",
      "name": "モックアダプター",
      "description": "string",
      "input_schema": {},
      "output_schema": {}
    }
  ]
}
```

---

## Audit Logs

### 一覧取得
```
GET /audit-logs
```

クエリ：
| パラメータ | 型 | 説明 |
|-------|------|-------------|
| `action` | string | `create`, `update`, `delete`, `publish`, `execute` |
| `resource_type` | string | `project`, `run`, `secret` |
| `actor_id` | uuid | ユーザーID |
| `from` | ISO8601 | 開始時刻 |
| `to` | ISO8601 | 終了時刻 |
| `page` | int | ページ番号 |
| `limit` | int | 1ページあたりの件数 |

レスポンス `200`：
```json
{
  "data": [
    {
      "id": "uuid",
      "action": "publish",
      "resource_type": "project",
      "resource_id": "uuid",
      "actor_id": "uuid",
      "actor_email": "user@example.com",
      "metadata": {},
      "created_at": "ISO8601"
    }
  ],
  "pagination": {}
}
```

---

## 使用量とコスト追跡

### 使用量サマリーを取得
```
GET /usage/summary
```

クエリ：
| パラメータ | 型 | デフォルト | 説明 |
|-------|------|---------|-------------|
| `period` | string | `month` | `day`, `week`, `month` |

レスポンス `200`：
```json
{
  "data": {
    "period": "month",
    "start_date": "2025-01-01T00:00:00Z",
    "end_date": "2025-01-31T23:59:59Z",
    "total_requests": 1500,
    "total_input_tokens": 500000,
    "total_output_tokens": 200000,
    "total_cost_usd": 15.50,
    "success_rate": 0.98,
    "avg_latency_ms": 850
  }
}
```

### 日次使用量を取得
```
GET /usage/daily
```

クエリ：
| パラメータ | 型 | 必須 | 説明 |
|-------|------|----------|-------------|
| `start` | ISO8601 | はい | 開始日 |
| `end` | ISO8601 | はい | 終了日 |

レスポンス `200`：
```json
{
  "data": [
    {
      "date": "2025-01-15",
      "total_requests": 150,
      "total_input_tokens": 50000,
      "total_output_tokens": 20000,
      "total_cost_usd": 1.55,
      "provider": "openai",
      "model": "gpt-4o"
    }
  ]
}
```

### プロジェクト別使用量を取得
```
GET /usage/by-project
```

クエリ：
| パラメータ | 型 | デフォルト | 説明 |
|-------|------|---------|-------------|
| `period` | string | `month` | `day`, `week`, `month` |

レスポンス `200`：
```json
{
  "data": [
    {
      "project_id": "uuid",
      "project_name": "マイプロジェクト",
      "total_requests": 500,
      "total_tokens": 150000,
      "total_cost_usd": 5.25
    }
  ]
}
```

### モデル別使用量を取得
```
GET /usage/by-model
```

クエリ：
| パラメータ | 型 | デフォルト | 説明 |
|-------|------|---------|-------------|
| `period` | string | `month` | `day`, `week`, `month` |

レスポンス `200`：
```json
{
  "data": [
    {
      "provider": "openai",
      "model": "gpt-4o",
      "total_requests": 800,
      "total_input_tokens": 300000,
      "total_output_tokens": 100000,
      "total_cost_usd": 10.00,
      "avg_latency_ms": 750
    }
  ]
}
```

### 実行使用量を取得
```
GET /runs/{run_id}/usage
```

レスポンス `200`：
```json
{
  "data": [
    {
      "id": "uuid",
      "step_run_id": "uuid",
      "provider": "openai",
      "model": "gpt-4o",
      "operation": "chat",
      "input_tokens": 1000,
      "output_tokens": 500,
      "total_tokens": 1500,
      "input_cost_usd": 0.0025,
      "output_cost_usd": 0.005,
      "total_cost_usd": 0.0075,
      "latency_ms": 850,
      "success": true,
      "created_at": "ISO8601"
    }
  ]
}
```

### 予算一覧
```
GET /usage/budgets
```

レスポンス `200`：
```json
{
  "data": [
    {
      "id": "uuid",
      "project_id": null,
      "budget_type": "monthly",
      "budget_amount_usd": 100.00,
      "alert_threshold": 0.80,
      "enabled": true,
      "created_at": "ISO8601",
      "updated_at": "ISO8601"
    }
  ]
}
```

### 予算作成
```
POST /usage/budgets
```

リクエスト：
```json
{
  "project_id": "uuid (オプション)",
  "budget_type": "monthly|daily",
  "budget_amount_usd": 100.00,
  "alert_threshold": 0.80
}
```

レスポンス `201`: 作成された予算

### 予算更新
```
PUT /usage/budgets/{id}
```

リクエスト：
```json
{
  "budget_amount_usd": 150.00,
  "alert_threshold": 0.90,
  "enabled": true
}
```

レスポンス `200`: 更新された予算

### 予算削除
```
DELETE /usage/budgets/{id}
```

レスポンス `204`: コンテンツなし

### モデル料金を取得
```
GET /usage/pricing
```

レスポンス `200`：
```json
{
  "data": [
    {
      "provider": "openai",
      "model": "gpt-4o",
      "input_cost_per_1k": 0.0025,
      "output_cost_per_1k": 0.01
    },
    {
      "provider": "anthropic",
      "model": "claude-3-opus",
      "input_cost_per_1k": 0.015,
      "output_cost_per_1k": 0.075
    }
  ]
}
```

---

## 管理者 - システムブロック

管理者専用APIエンドポイント。システムブロックの編集・バージョン管理を行う。

### システムブロック一覧
```
GET /admin/blocks
```

レスポンス `200`：
```json
{
  "blocks": [
    {
      "id": "uuid",
      "slug": "llm",
      "name": "LLM呼び出し",
      "description": "LLM APIを呼び出す",
      "category": "ai",
      "subcategory": "chat",
      "code": "const response = await ctx.llm.chat(...)",
      "config_schema": {},
      "input_schema": {},
      "output_schema": {},
      "ui_config": {"icon": "brain", "color": "#8B5CF6"},
      "is_system": true,
      "version": 3,
      "enabled": true,
      "created_at": "ISO8601",
      "updated_at": "ISO8601"
    }
  ]
}
```

### システムブロック取得
```
GET /admin/blocks/{id}
```

レスポンス `200`: システムブロックの詳細

### システムブロック更新
```
PUT /admin/blocks/{id}
```

リクエスト：
```json
{
  "name": "LLM呼び出し",
  "description": "LLM APIを呼び出す",
  "code": "const response = await ctx.llm.chat(...)",
  "config_schema": {},
  "input_schema": {},
  "output_schema": {},
  "ui_config": {"icon": "brain", "color": "#8B5CF6"},
  "change_summary": "プロンプト処理ロジックを改善"
}
```

レスポンス `200`: 更新されたブロック（バージョンがインクリメント）

### ブロックバージョン一覧
```
GET /admin/blocks/{id}/versions
```

レスポンス `200`：
```json
{
  "versions": [
    {
      "id": "uuid",
      "block_id": "uuid",
      "version": 2,
      "code": "...",
      "config_schema": {},
      "input_schema": {},
      "output_schema": {},
      "ui_config": {},
      "change_summary": "バグ修正",
      "changed_by": "uuid",
      "created_at": "ISO8601"
    }
  ]
}
```

### ブロックバージョン取得
```
GET /admin/blocks/{id}/versions/{version}
```

レスポンス `200`: 特定バージョンの詳細

### ブロックロールバック
```
POST /admin/blocks/{id}/rollback
```

リクエスト：
```json
{
  "version": 2
}
```

レスポンス `200`: 指定されたバージョンに復元されたブロック（新しいバージョンが作成される）

---

## ヘルス

### Liveness
```
GET /health
```

レスポンス `200`：
```json
{
  "status": "ok"
}
```

### Readiness
```
GET /ready
```

レスポンス `200`：
```json
{
  "status": "ok",
  "components": {
    "database": "ok",
    "redis": "ok"
  }
}
```

レスポンス `503` (異常時)：
```json
{
  "status": "error",
  "components": {
    "database": "error",
    "redis": "ok"
  }
}
```

---

## cURLサンプル

### プロジェクト作成
```bash
curl -X POST http://localhost:8080/api/v1/projects \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"name": "テストプロジェクト"}'
```

### ステップ追加
```bash
curl -X POST "http://localhost:8080/api/v1/projects/{id}/steps" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{
    "name": "ステップ1",
    "type": "tool",
    "config": {"adapter_id": "mock", "response": {"result": "ok"}}
  }'
```

### プロジェクト実行
```bash
curl -X POST "http://localhost:8080/api/v1/projects/{id}/runs" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"input": {"message": "こんにちは"}, "start_step_id": "{start_step_uuid}", "triggered_by": "test"}'
```

### JWT認証あり
```bash
# トークンを取得
TOKEN=$(curl -s -X POST http://localhost:8180/realms/ai-orchestration/protocol/openid-connect/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "username=admin@example.com&password=admin123&grant_type=password&client_id=frontend" \
  | jq -r .access_token)

# トークンを使用
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/projects
```

## 関連ドキュメント

- [BACKEND.md](./BACKEND.md) - バックエンドのコード構造とハンドラー
- [DATABASE.md](./DATABASE.md) - データベーススキーマ
- [openapi.yaml](./openapi.yaml) - 機械可読なOpenAPI仕様
- [DEPLOYMENT.md](./DEPLOYMENT.md) - 環境と認証のセットアップ
