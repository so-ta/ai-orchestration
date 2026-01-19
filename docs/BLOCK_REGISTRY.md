# ブロックレジストリリファレンス

ブロック定義の API リファレンス（Source of Truth）。

> **Status**: ✅ 実装済み（Unified Block Model）
> **Updated**: 2026-01-15
> **Role**: **API 仕様リファレンス**（このドキュメントが正）
> **See also**: [UNIFIED_BLOCK_MODEL.md](./designs/UNIFIED_BLOCK_MODEL.md) - 設計思想・アーキテクチャ
> **Migration**: `013_add_integration_blocks.sql` - 外部連携ブロック追加
> **RAG Support**: `seed.sql` - RAG ブロック 7 種追加

---

## このドキュメントの責任範囲

| 内容 | 担当 |
|------|------|
| ブロック一覧・仕様 | ✅ **このドキュメント** |
| ctx API 仕様 | ✅ **このドキュメント** |
| エラーコード定義 | ✅ **このドキュメント** |
| コード例 | ✅ **このドキュメント** |
| 設計思想・Why | ❌ [UNIFIED_BLOCK_MODEL.md](./designs/UNIFIED_BLOCK_MODEL.md) を参照 |
| アーキテクチャ | ❌ [UNIFIED_BLOCK_MODEL.md](./designs/UNIFIED_BLOCK_MODEL.md) を参照 |

**Claude Code への指示**: API 仕様はこのドキュメントを参照。設計思想は UNIFIED_BLOCK_MODEL.md を参照。

---

## クイックリファレンス

| 項目 | 値 |
|------|-------|
| テーブル | `block_definitions` |
| システムブロック | `tenant_id = NULL`（46 個: コア 18 + 基盤 10 + 連携 11 + RAG 7） |
| テナントブロック | `tenant_id = UUID` |
| 実行環境 | Goja JavaScript VM |
| バージョン履歴 | `block_versions` テーブル |
| カテゴリ | ai, logic, integration, data, control, utility |

## 概要

Block Registry はワークフローのステップタイプを管理するシステムです。
**Unified Block Model** により、すべてのブロックは JavaScript コードとして統一実行されます。

## アーキテクチャ

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Unified Block Model                               │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │                   block_definitions テーブル                     │ │
│  │                                                                   │ │
│  │  System Blocks (tenant_id = NULL)                                │ │
│  │  ├── start, llm, condition, switch, map, function, ...           │ │
│  │  └── 全ユーザーに提供、管理者のみ編集可                           │ │
│  │                                                                   │ │
│  │  Tenant Blocks (tenant_id = UUID)                                │ │
│  │  ├── カスタムブロック（テナント専用）                             │ │
│  │  └── ユーザーが作成・編集可能                                     │ │
│  └─────────────────────────────────────────────────────────────────┘ │
│                           │                                          │
│                           ▼                                          │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │                   Sandbox Executor (Goja VM)                     │ │
│  │                                                                   │ │
│  │  ctx = {                                                          │ │
│  │    http:     { get, post, put, delete, request }                 │ │
│  │    llm:      { chat, complete }                                   │ │
│  │    workflow: { run, executeStep }                                  │ │
│  │    human:    { requestApproval }                                  │ │
│  │    adapter:  { call, list }                                       │ │
│  │    secrets:  Record<string, string>                               │ │
│  │    env:      Record<string, string>                               │ │
│  │    log:      (level, message, data) => void                       │ │
│  │    embedding: { embed }                    // RAG                 │ │
│  │    vector:   { upsert, query, delete, listCollections } // RAG   │ │
│  │  }                                                                │ │
│  └─────────────────────────────────────────────────────────────────┘ │
│                                                                       │
└─────────────────────────────────────────────────────────────────────┘
```

## データモデル

### BlockDefinition

```go
type BlockDefinition struct {
    ID             uuid.UUID       // 一意の ID
    TenantID       *uuid.UUID      // NULL = システムブロック、それ以外はテナント固有
    Slug           string          // 一意の識別子（例: "llm", "discord"）
    Name           string          // 表示名
    Description    string          // ブロックの説明
    Category       string          // ai, logic, integration, data, control, utility

    // === Unified Block Model フィールド ===
    Code           string          // サンドボックスで実行される JavaScript コード
    UIConfig       json.RawMessage // {icon, color, configSchema}
    IsSystem       bool            // システムブロック = 管理者のみ編集可
    Version        int             // バージョン番号、更新時にインクリメント

    // スキーマ（JSON Schema 形式）
    ConfigSchema   json.RawMessage // ブロックの設定オプション
    InputSchema    json.RawMessage // 期待される入力構造
    OutputSchema   json.RawMessage // 出力構造

    // エラーハンドリング
    ErrorCodes     []ErrorCodeDef  // このブロックの定義済みエラーコード

    // === ブロック継承/拡張フィールド ===
    ParentBlockID  *uuid.UUID      // 継承用の親ブロック参照
    ConfigDefaults json.RawMessage // 親の config_schema のデフォルト値
    PreProcess     string          // 入力変換用 JavaScript コード
    PostProcess    string          // 出力変換用 JavaScript コード
    InternalSteps  []InternalStep  // 複合ブロックの内部ステップ

    // === 解決済みフィールド（バックエンドで設定） ===
    PreProcessChain        []string        // preProcess コードのチェーン（子→ルート）
    PostProcessChain       []string        // postProcess コードのチェーン（ルート→子）
    ResolvedCode           string          // ルート祖先からのコード
    ResolvedConfigDefaults json.RawMessage // チェーンからマージされた設定デフォルト

    // メタデータ
    Enabled        bool
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

type InternalStep struct {
    Type      string          `json:"type"`       // 実行するブロックの slug
    Config    json.RawMessage `json:"config"`     // ステップ設定
    OutputKey string          `json:"output_key"` // 出力を格納するキー
}

type ErrorCodeDef struct {
    Code        string `json:"code"`        // 例: "LLM_001"
    Name        string `json:"name"`        // 例: "RATE_LIMIT_EXCEEDED"
    Description string `json:"description"` // 人間が読める説明
    Retryable   bool   `json:"retryable"`   // このエラーはリトライ可能か？
}
```

### ブロック継承/拡張

ブロック継承により、既存ブロックを拡張して再利用可能なブロックを作成できます。
**多段継承**により、認証パターンやサービス固有の設定を階層的に定義できます。

#### 継承階層アーキテクチャ

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    階層的ブロック継承                                     │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  http (Level 0: ベース)                                                  │
│  ├── webhook (Level 1: パターン)                                         │
│  │   ├── slack (Level 2: 具体)                                          │
│  │   └── discord (Level 2: 具体)                                        │
│  │                                                                       │
│  ├── rest-api (Level 1: パターン)                                        │
│  │   ├── bearer-api (Level 2: 認証)                                     │
│  │   │   ├── github-api (Level 3: サービス)                              │
│  │   │   │   ├── github_create_issue (Level 4: 操作)                    │
│  │   │   │   └── github_add_comment (Level 4: 操作)                     │
│  │   │   ├── notion-api (Level 3: サービス)                              │
│  │   │   │   ├── notion_query_db (Level 4: 操作)                        │
│  │   │   │   └── notion_create_page (Level 4: 操作)                     │
│  │   │   └── email_sendgrid (Level 3: 具体)                             │
│  │   ├── api-key-header (Level 2: 認証)                                 │
│  │   ├── api-key-query (Level 2: 認証)                                  │
│  │   │   └── google-api (Level 3: サービス)                              │
│  │   │       ├── gsheets_append (Level 4: 操作)                         │
│  │   │       └── gsheets_read (Level 4: 操作)                           │
│  │   └── web_search (Level 2: 具体)                                     │
│  │                                                                       │
│  └── graphql (Level 1: パターン) ← rest-api を継承                       │
│      └── linear-api (Level 2: サービス)                                  │
│          └── linear_create_issue (Level 3: 操作)                        │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

#### 各レベルの責務

| Level | 名称 | 責務 | 例 |
|-------|------|------|-----|
| 0 | ベース | 基本的な実行ロジック | `http` |
| 1 | パターン | 通信パターン、基本認証 | `webhook`, `rest-api`, `graphql` |
| 2 | 認証 | 認証方式の抽象化 | `bearer-api`, `api-key-header`, `api-key-query` |
| 3 | サービス | サービス固有の設定 | `github-api`, `notion-api`, `google-api` |
| 4+ | 操作 | 具体的な操作 | `github_create_issue`, `notion_query_db` |

#### 継承の仕組み

```
┌──────────────────────────────────────────────────────────────────┐
│          多段継承の実行フロー                                      │
├──────────────────────────────────────────────────────────────────┤
│                                                                    │
│  github_create_issue → github-api → bearer-api → rest-api → http │
│                                                                    │
│  実行順序:                                                         │
│  1. PreProcess チェーン（子 → ルート）:                            │
│     github_create_issue.preProcess → github-api.preProcess →      │
│     bearer-api.preProcess → rest-api.preProcess                   │
│                                                                    │
│  2. Config マージ（ルート → 子）:                                  │
│     rest-api.configDefaults ← bearer-api.configDefaults ←        │
│     github-api.configDefaults ← github_create_issue.configDefaults│
│     ← step.config（実行時）                                       │
│                                                                    │
│  3. コード実行（ルート祖先から: http.code）                         │
│                                                                    │
│  4. PostProcess チェーン（ルート → 子）:                           │
│     rest-api.postProcess → bearer-api.postProcess →               │
│     github-api.postProcess → github_create_issue.postProcess      │
│                                                                    │
└──────────────────────────────────────────────────────────────────┘
```

#### 継承ルール

| ルール | 説明 |
|--------|------|
| コードを持つブロックのみ継承可能 | `Code != ""` |
| 最大継承深度 | 50 レベル（実用上は 4-5 レベル） |
| 循環継承禁止 | A→B→C→A のような循環は不可（トポロジカルソートで検出） |
| テナント分離 | 同一テナント内またはシステムブロックからのみ継承可能 |
| マイグレーション順序 | トポロジカルソートにより親ブロックが先に処理される |

#### ConfigDefaults のマージ順序

```
ルート祖先のデフォルト (rest-api)
    ↓（上書き）
認証レベルのデフォルト (bearer-api: auth_type=bearer)
    ↓（上書き）
サービスのデフォルト (github-api: base_url, secret_key)
    ↓（上書き）
子のデフォルト (github_create_issue: 固有の設定)
    ↓（上書き）
ステップ設定（実行時）
```

#### 継承ブロックの例（新アーキテクチャ）

```javascript
// github_create_issue (github-api → bearer-api → rest-api → http から継承)

// ConfigDefaults（親からのマージ）:
// rest-api より: { auth_type: "bearer" }
// github-api より: { base_url: "https://api.github.com", secret_key: "GITHUB_TOKEN" }

// PreProcess（このブロック固有）:
const payload = {
    title: renderTemplate(config.title, input),
    body: config.body ? renderTemplate(config.body, input) : undefined,
    labels: config.labels,
    assignees: config.assignees
};
return {
    ...input,
    url: '/repos/' + config.owner + '/' + config.repo + '/issues',
    method: 'POST',
    body: payload
};

// PostProcess（このブロック固有）:
if (input.status >= 400) {
    const errorMsg = input.body?.message || 'Unknown error';
    throw new Error('[GITHUB_002] Issue作成失敗: ' + errorMsg);
}
return {
    id: input.body.id,
    number: input.body.number,
    url: input.body.url,
    html_url: input.body.html_url
};

// 親の PreProcess チェーン（自動実行）:
// 1. github-api: GitHub API ヘッダー追加 (Accept, X-GitHub-Api-Version)
// 2. bearer-api: token → auth_key マッピング
// 3. rest-api: Authorization: Bearer ヘッダー追加、URL 構築
// 4. http: 実際の HTTP リクエスト実行

// 親の PostProcess チェーン（自動実行）:
// 1. rest-api: レート制限・エラーステータスチェック
// 2. github-api: 404 エラーのカスタムメッセージ
```

#### 新規サービス追加の例

```javascript
// 例: Jira Issue 作成を追加（約 20 行で実装可能）

// Step 1: jira-api 基盤ブロック作成
{
    slug: "jira-api",
    parent_block_slug: "bearer-api",
    config_defaults: {
        "base_url": "https://{domain}.atlassian.net/rest/api/3",
        "secret_key": "JIRA_API_TOKEN"
    },
    pre_process: `
        // Basic Auth 用のヘッダー変換
        const email = ctx.secrets.JIRA_EMAIL;
        const token = config.auth_key || ctx.secrets[config.secret_key];
        const basicAuth = btoa(email + ':' + token);
        return {
            ...input,
            headers: { ...input.headers, 'Authorization': 'Basic ' + basicAuth }
        };
    `
}

// Step 2: jira_create_issue 操作ブロック作成
{
    slug: "jira_create_issue",
    parent_block_slug: "jira-api",
    pre_process: `
        return {
            url: '/issue',
            method: 'POST',
            body: {
                fields: {
                    project: { key: config.project_key },
                    summary: renderTemplate(config.summary, input),
                    issuetype: { name: config.issue_type || 'Task' }
                }
            }
        };
    `,
    post_process: `
        return { key: input.body.key, id: input.body.id };
    `
}
```

#### InternalSteps（複合ブロック）

複数のブロックを順次実行する複合ブロックを作成できます：

```javascript
// enriched-http ブロック
// internal_steps:
[
    {
        "type": "function",
        "config": { "code": "return { timestamp: Date.now(), ...input }" },
        "output_key": "enriched"
    },
    {
        "type": "http",
        "config": {},  // マージされた設定を使用
        "output_key": "response"
    },
    {
        "type": "function",
        "config": { "code": "return { ...input.response, enriched: input.enriched }" },
        "output_key": "final"
    }
]
// 出力は internal_steps の結果がマージされた状態
```

### データベーススキーマ

```sql
-- block_definitions テーブル（Unified Block Model 対応）
CREATE TABLE block_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id),  -- NULL = システムブロック
    slug VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(50) NOT NULL,
    icon VARCHAR(50),

    -- === Unified Block Model カラム ===
    code TEXT,                              -- JavaScript コード
    ui_config JSONB NOT NULL DEFAULT '{}',  -- {icon, color, configSchema}
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    version INTEGER NOT NULL DEFAULT 1,

    -- スキーマ
    config_schema JSONB NOT NULL DEFAULT '{}',
    input_schema JSONB,
    output_schema JSONB,

    error_codes JSONB DEFAULT '[]',

    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(tenant_id, slug),
    CONSTRAINT valid_category CHECK (category IN ('ai', 'logic', 'integration', 'data', 'control', 'utility'))
);

-- block_versions テーブル（バージョン履歴）
CREATE TABLE block_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    block_id UUID NOT NULL REFERENCES block_definitions(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,

    -- スナップショット
    code TEXT NOT NULL,
    config_schema JSONB NOT NULL,
    input_schema JSONB,
    output_schema JSONB,
    ui_config JSONB NOT NULL,

    -- 変更追跡
    change_summary TEXT,
    changed_by UUID,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(block_id, version)
);

CREATE INDEX idx_block_definitions_tenant ON block_definitions(tenant_id);
CREATE INDEX idx_block_definitions_category ON block_definitions(category);
CREATE INDEX idx_block_definitions_enabled ON block_definitions(enabled);
CREATE INDEX idx_block_versions_block_id ON block_versions(block_id);
```

## 入力スキーマ

### 概要

`input_schema` は各ブロックが期待する入力データの構造を定義します。
ワークフロー実行時に、開始ステップの `input_schema` を基に入力フォームが自動生成されます。

### 用途

1. **実行時の入力フォーム生成**: 開始ステップの次のブロックの `input_schema` から動的にフォームを生成
2. **入力値のバリデーション**: 実行前に入力データの形式をチェック
3. **ドキュメント**: ブロックが期待する入力形式を開発者に伝達

### 形式（JSON Schema）

```json
{
  "type": "object",
  "description": "ブロックへの入力データの説明",
  "properties": {
    "items": {
      "type": "array",
      "description": "処理対象の配列"
    },
    "query": {
      "type": "string",
      "description": "検索クエリ"
    }
  },
  "required": ["items"],
  "examples": [
    { "items": [1, 2, 3], "query": "example" }
  ]
}
```

### ブロック種別ごとの input_schema

| カテゴリ | ブロック例 | input_schema 内容 |
|---------|-----------|------------------|
| **Control** | condition, switch | 条件評価対象のデータ |
| **Data** | map, filter | `items` 配列（required） |
| **Integration** | slack, discord | テンプレートで参照可能なデータ |
| **AI** | llm, router | プロンプトで参照するコンテキスト |
| **Utility** | function, code | 任意のオブジェクト |

### フロントエンドでの活用

```vue
<!-- RunDialog.vue -->
<DynamicConfigForm
  v-model="inputValues"
  :schema="firstStepBlock.input_schema"
  @validation-change="handleValidation"
/>
```

1. 実行ボタンクリック → RunDialog 表示
2. 開始ステップの次のブロックの `input_schema` を取得
3. `DynamicConfigForm` で入力フォームを動的生成
4. ユーザーが入力 → バリデーション
5. 実行開始（入力値を `runs.create()` に渡す）

## エラーコードシステム

### 標準エラーコード形式

```
{BLOCK}_{NUMBER}_{TYPE}

例:
- LLM_001_RATE_LIMIT     - LLM レート制限超過
- LLM_002_INVALID_MODEL  - 無効なモデル指定
- HTTP_001_TIMEOUT       - HTTP リクエストタイムアウト
- HTTP_002_CONN_REFUSED  - 接続拒否
- COND_001_INVALID_EXPR  - 無効な条件式
- DISCORD_001_WEBHOOK_ERROR - Discord Webhook エラー
```

### BlockError 構造

```go
type BlockError struct {
    Code       string          `json:"code"`        // エラーコード（例: "LLM_001"）
    Message    string          `json:"message"`     // 人間が読めるメッセージ
    Details    json.RawMessage `json:"details"`     // 追加のエラー詳細
    Retryable  bool            `json:"retryable"`   // リトライ可能か？
    RetryAfter *time.Duration  `json:"retry_after"` // 推奨リトライ遅延
}

func (e *BlockError) Error() string {
    return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}
```

### エラーコードカテゴリ

| カテゴリ | 範囲 | 説明 |
|----------|-------|-------------|
| SYSTEM   | 000-099 | システムレベルエラー |
| CONFIG   | 100-199 | 設定エラー |
| INPUT    | 200-299 | 入力バリデーションエラー |
| EXEC     | 300-399 | 実行エラー |
| OUTPUT   | 400-499 | 出力処理エラー |
| AUTH     | 500-599 | 認証/認可エラー |
| RATE     | 600-699 | レート制限エラー |
| TIMEOUT  | 700-799 | タイムアウトエラー |

## システムブロック

システムブロック（`tenant_id = NULL`）は全ユーザーに提供されます。

### 現在のシステムブロック一覧

| Slug | 名前 | カテゴリ | コード概要 |
|------|------|----------|----------|
| `start` | Start | control | `return input;` |
| `llm` | LLM | ai | `ctx.llm.chat(...)` |
| `condition` | Condition | logic | `return {..., __branch: result ? 'then' : 'else'}` |
| `switch` | Switch | logic | 多分岐ルーティング |
| `map` | Map | data | 配列並列処理 |
| `filter` | Filter | data | 配列フィルタリング |
| `split` | Split | data | バッチ分割 |
| `aggregate` | Aggregate | data | データ集約 |
| `tool` | Tool | integration | `ctx.adapter.call(...)` |
| `http` | HTTP Request | integration | `ctx.http.request(...)` |
| `subflow` | Subflow | control | `ctx.workflow.run(...)` |
| `wait` | Wait | control | 遅延・タイマー |
| `human_in_loop` | Human in Loop | control | `ctx.human.requestApproval(...)` |
| `error` | Error | control | `throw new Error(...)` |
| `router` | Router | ai | AI 分類ルーティング |
| `note` | Note | utility | ドキュメント用（`return input;`） |
| `code` | Code | utility | ユーザー定義 JavaScript |

> **注記**: `join` ブロックは廃止されました。Block Group 外での分岐ブロック（Condition/Switch）の複数出力は禁止されており、Block Group 内では出力が自動的に集約されるため、join ブロックは不要になりました。

### 基盤/パターンブロック一覧（継承階層用）

これらのブロックは具体的な連携ブロックの親として機能し、認証やエラーハンドリングを共通化します。

| Slug | 名前 | カテゴリ | 親 | 説明 |
|------|------|----------|-----|------|
| `webhook` | Webhook | integration | `http` | Webhook POST 通知パターン |
| `rest-api` | REST API | integration | `http` | REST API with 認証（Bearer/API Key 対応） |
| `graphql` | GraphQL | integration | `rest-api` | GraphQL API 呼び出しパターン |
| `bearer-api` | Bearer Token API | integration | `rest-api` | Bearer Token 認証 API |
| `api-key-header` | API Key Header | integration | `rest-api` | API Key Header ベース認証 |
| `api-key-query` | API Key Query | integration | `rest-api` | API Key Query パラメータ認証 |
| `github-api` | GitHub API | integration | `bearer-api` | GitHub API 共通設定 |
| `notion-api` | Notion API | integration | `bearer-api` | Notion API 共通設定 |
| `google-api` | Google API | integration | `api-key-query` | Google API 共通設定 |
| `linear-api` | Linear API | integration | `graphql` | Linear GraphQL API 共通設定 |

### 外部連携ブロック一覧

| Slug | 名前 | 親ブロック | 説明 | 必要シークレット |
|------|------|-----------|------|-----------------|
| `slack` | Slack | `webhook` | Slack チャンネルにメッセージ送信 | `SLACK_WEBHOOK_URL` |
| `discord` | Discord | `webhook` | Discord Webhook に通知 | `DISCORD_WEBHOOK_URL` |
| `github_create_issue` | GitHub: Issue 作成 | `github-api` | GitHub に Issue を作成 | `GITHUB_TOKEN` |
| `github_add_comment` | GitHub: コメント追加 | `github-api` | Issue/PR にコメント追加 | `GITHUB_TOKEN` |
| `notion_create_page` | Notion: ページ作成 | `notion-api` | Notion にページを作成 | `NOTION_API_KEY` |
| `notion_query_db` | Notion: DB 検索 | `notion-api` | Notion データベースを検索 | `NOTION_API_KEY` |
| `gsheets_append` | Google Sheets: 行追加 | `google-api` | スプレッドシートに行を追加 | `GOOGLE_API_KEY` |
| `gsheets_read` | Google Sheets: 読み取り | `google-api` | スプレッドシートから読み取り | `GOOGLE_API_KEY` |
| `email_sendgrid` | Email (SendGrid) | `api-key-header` | SendGrid でメール送信 | `SENDGRID_API_KEY` |
| `web_search` | Web 検索 | `api-key-header` | Tavily API で Web 検索 | `TAVILY_API_KEY` |
| `linear_create_issue` | Linear: Issue 作成 | `linear-api` | Linear に Issue を作成 | `LINEAR_API_KEY` |

### RAG ブロック一覧

| Slug | 名前 | カテゴリ | 説明 | 必要シークレット |
|------|------|----------|------|-----------------|
| `embedding` | Embedding | ai | テキストをベクトルに変換 | `OPENAI_API_KEY`, `COHERE_API_KEY`, `VOYAGE_API_KEY` |
| `vector-upsert` | Vector Upsert | data | ドキュメントをベクトル DB に保存 | - |
| `vector-search` | Vector Search | data | 類似ドキュメントを検索（ハイブリッド検索対応） | - |
| `vector-delete` | Vector Delete | data | ベクトル DB からドキュメント削除 | - |
| `doc-loader` | Document Loader | data | URL/テキストからドキュメント取得 | - |
| `text-splitter` | Text Splitter | data | テキストをチャンクに分割 | - |
| `rag-query` | RAG Query | ai | RAG 検索+LLM 生成（一括処理） | `OPENAI_API_KEY` |

### RAG ブロック エラーコード一覧

| コード | 名前 | ブロック | リトライ可 | 説明 |
|------|------|-------|-----------|-------------|
| `EMB_001` | PROVIDER_ERROR | embedding | ✅ | Embedding プロバイダー API エラー |
| `EMB_002` | EMPTY_INPUT | embedding | ❌ | Embedding 用のテキストがない |
| `VEC_001` | COLLECTION_REQUIRED | vector-* | ❌ | コレクション名が必須 |
| `VEC_002` | DOCUMENTS_REQUIRED | vector-upsert | ❌ | ドキュメント配列が必須 |
| `VEC_003` | VECTOR_OR_QUERY_REQUIRED | vector-search | ❌ | ベクトルまたはクエリテキストが必須 |
| `VEC_004` | IDS_REQUIRED | vector-delete | ❌ | ID 配列が必須 |
| `DOC_001` | FETCH_ERROR | doc-loader | ✅ | URL 取得失敗（SSRF 保護を含む） |
| `DOC_002` | EMPTY_CONTENT | doc-loader | ❌ | コンテンツがない |
| `TXT_001` | EMPTY_TEXT | text-splitter | ❌ | 分割用のテキストがない |
| `RAG_001` | QUERY_REQUIRED | rag-query | ❌ | クエリテキストが必須 |
| `RAG_002` | COLLECTION_REQUIRED | rag-query | ❌ | コレクション名が必須 |

### Goja ランタイム制約（重要）

ブロックコードは Goja JavaScript VM で実行されます。以下の制約があります：

| 制約 | 説明 | 対処法 |
|------|------|--------|
| **`await` 禁止** | goja は `await` キーワードをサポートしない | `ctx.*` メソッドは同期的に呼び出す |
| **`async function` 禁止** | async 関数定義不可 | 通常の `function` を使用 |
| **`async () =>` 禁止** | async アロー関数不可 | 通常の `() =>` を使用 |

#### なぜ同期的に動作するか

`ctx.llm.chat()`, `ctx.http.get()` などのメソッドは、Go 側で非同期処理を行い、結果が返るまでブロックします。
JavaScript 側からは同期的な関数呼び出しに見えます。

```javascript
// ❌ NG: await は使用不可
const response = await ctx.llm.chat(...);

// ✅ OK: 同期的に呼び出す（内部でブロッキング）
const response = ctx.llm.chat(...);
```

#### バリデーション

seeder コマンドはブロックコードをバリデーションし、`await`/`async` の使用を検出します：

```bash
go run ./cmd/seeder --validate
```

バリデーションエラー例：
```
❌ Block Validation Errors:
   [rag-query.code] goja runtime incompatibility: goja does not support 'await' keyword. Use synchronous ctx.* methods instead (they are blocking)
```

### システムブロックのコード例

```javascript
// llm ブロック
const prompt = renderTemplate(config.user_prompt || '', input);
const systemPrompt = config.system_prompt || '';

const response = ctx.llm.chat(config.provider, config.model, {
    messages: [
        ...(systemPrompt ? [{ role: 'system', content: systemPrompt }] : []),
        { role: 'user', content: prompt }
    ],
    temperature: config.temperature ?? 0.7,
    maxTokens: config.max_tokens ?? 1000
});

return {
    content: response.content,
    usage: response.usage
};
```

```javascript
// http ブロック
const url = renderTemplate(config.url, input);

const response = ctx.http.request(url, {
    method: config.method || 'GET',
    headers: config.headers || {},
    body: config.body ? renderTemplate(JSON.stringify(config.body), input) : null
});

return response;
```

### 外部連携ブロックのコード例

```javascript
// slack ブロック
const webhookUrl = config.webhook_url || ctx.secrets.SLACK_WEBHOOK_URL;
if (!webhookUrl) {
    throw new Error('[SLACK_001] Webhook URL が設定されていません');
}

const payload = {
    text: renderTemplate(config.message, input),
    channel: config.channel,
    username: config.username,
    icon_emoji: config.icon_emoji
};

const response = ctx.http.post(webhookUrl, payload, {
    headers: { 'Content-Type': 'application/json' }
});

return { success: true, status: response.status };
```

```javascript
// github_create_issue ブロック
const token = config.token || ctx.secrets.GITHUB_TOKEN;
const url = 'https://api.github.com/repos/' + config.owner + '/' + config.repo + '/issues';

const response = ctx.http.post(url, {
    title: renderTemplate(config.title, input),
    body: renderTemplate(config.body, input),
    labels: config.labels,
    assignees: config.assignees
}, {
    headers: {
        'Authorization': 'Bearer ' + token,
        'Accept': 'application/vnd.github+json'
    }
});

return {
    number: response.body.number,
    html_url: response.body.html_url
};
```

### RAG ブロックのコード例

```javascript
// embedding ブロック
// サポートプロバイダー: openai, cohere, voyage (Phase 3.3)
const texts = Array.isArray(input.texts) ? input.texts : [input.text || input.content];
const result = ctx.embedding.embed(
    config.provider || 'openai',  // 'openai', 'cohere', 'voyage'
    config.model || 'text-embedding-3-small',
    texts
);
return {
    vectors: result.vectors,
    model: result.model,
    dimension: result.dimension,
    usage: result.usage
};

// プロバイダー別の利用可能なモデル:
// OpenAI: text-embedding-3-small (1536d), text-embedding-3-large (3072d)
// Cohere: embed-english-v3.0 (1024d), embed-multilingual-v3.0 (1024d)
// Voyage: voyage-3 (1024d), voyage-3-lite, voyage-code-3
```

```javascript
// vector-upsert ブロック
const documents = (input.documents || [input]).map(doc => ({
    id: doc.id,
    content: doc.content || doc.text,
    metadata: doc.metadata || {},
    vector: doc.vector
}));

const result = ctx.vector.upsert(config.collection, documents, {
    embedding_provider: config.embedding_provider,
    embedding_model: config.embedding_model
});

return { upserted_count: result.upserted_count, ids: result.ids };
```

```javascript
// vector-search ブロック
let queryVector = input.vector;
if (!queryVector && input.query) {
    const embResult = ctx.embedding.embed(
        config.embedding_provider || 'openai',
        config.embedding_model || 'text-embedding-3-small',
        [input.query]
    );
    queryVector = embResult.vectors[0];
}

const result = ctx.vector.query(config.collection, queryVector, {
    top_k: config.top_k || 5,
    threshold: config.threshold,
    filter: config.filter,
    include_content: config.include_content !== false
});

return { matches: result.matches };
```

```javascript
// vector-search 高度なフィルタ付き (Phase 3.1)
// サポート: $eq, $ne, $gt, $gte, $lt, $lte, $in, $nin, $and, $or, $exists, $contains
const result = ctx.vector.query(config.collection, queryVector, {
    top_k: 10,
    filter: {
        "$or": [
            { "category": "news" },
            { "category": "blog" }
        ],
        "score": { "$gte": 0.8 },
        "author": { "$exists": true }
    }
});
```

```javascript
// vector-search ハイブリッド検索付き (Phase 3.2)
// ベクトル類似度 + キーワード検索を RRF で組み合わせ
const result = ctx.vector.query(config.collection, queryVector, {
    top_k: 10,
    keyword: "machine learning",  // ハイブリッド検索を有効化
    hybrid_alpha: 0.7             // 70% ベクトル、30% キーワード
});
```

```javascript
// rag-query ブロック（RAG 検索 + LLM 生成）
// 1. クエリを Embedding
const embResult = ctx.embedding.embed(
    config.embedding_provider || 'openai',
    config.embedding_model || 'text-embedding-3-small',
    [input.query]
);

// 2. ベクトル検索
const searchResult = ctx.vector.query(config.collection, embResult.vectors[0], {
    top_k: config.top_k || 5,
    include_content: true
});

// 3. 取得したドキュメントからコンテキストを構築
const context = searchResult.matches.map(m => m.content).join('\n\n---\n\n');

// 4. LLM でレスポンスを生成
const systemPrompt = config.system_prompt ||
    '以下のコンテキストに基づいて質問に回答してください。コンテキストに答えがない場合は、そう伝えてください。';
const userPrompt = 'コンテキスト:\n' + context + '\n\n質問: ' + input.query;

const response = ctx.llm.chat(
    config.llm_provider || 'openai',
    config.llm_model || 'gpt-4o-mini',
    {
        messages: [
            { role: 'system', content: systemPrompt },
            { role: 'user', content: userPrompt }
        ],
        temperature: config.temperature ?? 0.7
    }
);

return {
    answer: response.content,
    sources: searchResult.matches.map(m => ({ id: m.id, score: m.score, content: m.content })),
    usage: response.usage
};
```

## 新規ブロックの追加

### 標準手順（Migration による追加）

**⚠️ 必ず先に [UNIFIED_BLOCK_MODEL.md](./designs/UNIFIED_BLOCK_MODEL.md) を読むこと**

1. **Migration ファイル作成**: `backend/migrations/XXX_{name}_block.sql`

2. **INSERT 文作成**:

```sql
INSERT INTO block_definitions (
    id, tenant_id, slug, name, description, category, icon,
    config_schema, error_codes, code, ui_config, is_system, enabled
) VALUES (
    gen_random_uuid(),
    NULL,  -- システムブロック（全ユーザーに提供）
    'discord',
    'Discord 通知',
    'Discord Webhook にメッセージを送信',
    'integration',
    'message-circle',
    '{
        "type": "object",
        "properties": {
            "webhook_url": {"type": "string", "title": "Webhook URL"},
            "message": {"type": "string", "title": "メッセージ"}
        },
        "required": ["message"]
    }',
    '[{"code": "DISCORD_001", "name": "WEBHOOK_ERROR", "description": "Webhook 呼び出し失敗", "retryable": true}]',
    $code$
const webhookUrl = config.webhook_url || ctx.secrets.DISCORD_WEBHOOK_URL;
if (!webhookUrl) {
    throw new Error('Webhook URL が設定されていません');
}

const payload = {
    content: renderTemplate(config.message, input)
};

const response = ctx.http.post(webhookUrl, payload, {
    headers: { 'Content-Type': 'application/json' }
});

if (response.status >= 400) {
    throw new Error('Discord API error: ' + response.status);
}

return { success: true, status: response.status };
    $code$,
    '{"icon": "message-circle", "color": "#5865F2"}',
    TRUE,
    TRUE
)
ON CONFLICT (tenant_id, slug) WHERE tenant_id IS NULL
DO UPDATE SET
    code = EXCLUDED.code,
    config_schema = EXCLUDED.config_schema,
    ui_config = EXCLUDED.ui_config;
```

3. **Migration 実行**:
```bash
docker compose exec api migrate -path /migrations -database "$DATABASE_URL" up
```

4. **このドキュメントを更新**: システムブロック一覧に追加

### YAML 形式でのブロック追加（推奨）

外部 API 連携ブロックは YAML ファイルで定義できます。宣言的な `request`/`response` 設定により、JavaScript コードなしで HTTP API 呼び出しを実現できます。

**YAML ファイルの配置場所**: `backend/internal/seed/blocks/yaml/`

**基本構造**:

```yaml
---
slug: github_create_issue
version: 3
name: "GitHub: Issue作成"
description: GitHubリポジトリにIssueを作成
category: apps
subcategory: github
icon: git-pull-request
parent_block_slug: github-api
enabled: true

config_schema:
  type: object
  required: [owner, repo, title]
  properties:
    owner:
      type: string
      title: オーナー
    repo:
      type: string
      title: リポジトリ
    title:
      type: string
      title: タイトル

# 宣言的リクエスト設定 - PreProcess の代わり
request:
  url: "/repos/{{owner}}/{{repo}}/issues"
  method: POST
  body:
    title: "{{input.title}}"
    body: "{{input.body}}"
  query_params:
    param1: "{{value}}"  # クエリパラメータは自動でエンコードされる

# 宣言的レスポンス設定 - PostProcess の代わり
response:
  success_status: [200, 201]
  output_mapping:
    id: body.id
    url: body.url

error_codes:
  - code: GITHUB_002
    name: CREATE_FAILED
    description: Issue作成に失敗しました
    retryable: true
```

#### テンプレート変数

| 構文 | 説明 | 例 |
|------|------|-----|
| `{{field}}` | config の値 | `{{owner}}` |
| `{{input.field}}` | 入力データの値 | `{{input.title}}` |
| `{{secret.KEY}}` | シークレット（認証情報） | `{{secret.GITHUB_TOKEN}}` |

#### URL パス変数の自動エンコード

URL テンプレート内の変数は自動的に URL エンコードされます（RFC 3986 準拠）。

```yaml
request:
  url: "/spreadsheets/{{spreadsheet_id}}/values/{{range}}"
  # range が "Sheet1!A1:B10" の場合、自動的に "Sheet1%21A1:B10" にエンコード
```

- 既にエンコード済みの値（`%20` 等を含む）は二重エンコードされません
- クエリパラメータ（`query_params`）も自動エンコードされます

#### オブジェクト形式による omit_empty

リクエストボディのオプショナルフィールドは、オブジェクト形式で `omit_empty: true` を指定することで、空の場合に自動的に除外できます:

```yaml
request:
  url: "/databases/{{database_id}}/query"
  method: POST
  body:
    # オブジェクト形式: value + omit_empty
    filter:
      value: "{{filter}}"
      omit_empty: true      # filter が空/null の場合、フィールドを省略
    sorts:
      value: "{{sorts}}"
      omit_empty: true      # sorts が空配列の場合、フィールドを省略
    # 通常形式: 常に含まれる
    page_size: "{{page_size}}"
```

**omit_empty の判定基準**:

| 値の型 | 空と判定される条件 |
|--------|---------------------|
| 文字列 | `""` (空文字列) |
| 配列 | `[]` (空配列) |
| オブジェクト | `{}` (空オブジェクト) |
| null/未定義 | 常に空 |
| boolean | 空とは判定されない（`false` も含める） |
| 数値 | 空とは判定されない（`0` も含める） |

**ユースケース**: Notion API、Google API など、オプショナルパラメータを持つ API で、未指定時にフィールド自体を送信したくない場合に使用します。

#### 複数ブロックを1ファイルに定義

YAML の `---` セパレータで複数のブロックを1ファイルに定義できます:

```yaml
---
slug: service-api
parent_block_slug: bearer-api
# 基盤ブロックの定義...
---
slug: service_operation1
parent_block_slug: service-api
# 操作ブロック1の定義...
---
slug: service_operation2
parent_block_slug: service-api
# 操作ブロック2の定義...
```

### Go Adapter が必要なケース（例外）

以下の場合のみ、Go Adapter を実装：

| ケース | 理由 |
|--------|------|
| LLM プロバイダー追加 | `ctx.llm` 経由で呼び出すため |
| 複雑な認証フロー | OAuth2 等、JS では困難な場合 |
| バイナリ処理 | 画像・ファイル処理等 |

Go Adapter 追加手順:
1. `backend/internal/adapter/{name}.go` を作成
2. `Adapter` インターフェースを実装
3. レジストリに登録
4. テスト `{name}_test.go` を追加
5. docs/BACKEND.md を更新

## API エンドポイント

### テナント API

```
GET    /api/v1/blocks                    # リスト（システム + テナントブロック）
GET    /api/v1/blocks/{slug}             # 詳細取得
POST   /api/v1/blocks                    # カスタムブロック作成
PUT    /api/v1/blocks/{slug}             # 更新（テナント用のみ）
DELETE /api/v1/blocks/{slug}             # 削除（カスタムのみ）
```

### 管理者 API（システムブロック管理）

```
GET    /api/v1/admin/blocks              # システムブロック一覧
GET    /api/v1/admin/blocks/{id}         # 詳細
PUT    /api/v1/admin/blocks/{id}         # システムブロック編集
GET    /api/v1/admin/blocks/{id}/versions # バージョン履歴
POST   /api/v1/admin/blocks/{id}/rollback # ロールバック
```

## フロントエンド統合

### ブロックパレット

```typescript
interface BlockDefinition {
    slug: string
    name: string
    description: string
    category: 'ai' | 'logic' | 'integration' | 'data' | 'control' | 'utility'
    icon: string
    code?: string           // Unified Block Model
    ui_config?: UIConfig    // {icon, color, configSchema}
    is_system?: boolean
    version?: number
    configSchema: JSONSchema
    inputSchema: JSONSchema
    outputSchema: JSONSchema
    errorCodes: ErrorCodeDef[]
}

// Composable
function useBlocks() {
    const blocks = ref<BlockDefinition[]>([])

    async function loadBlocks() {
        const res = await api.get('/blocks')
        blocks.value = res.data
    }

    function getBlocksByCategory(category: string) {
        return blocks.value.filter(b => b.category === category)
    }

    return { blocks, loadBlocks, getBlocksByCategory }
}
```

### 動的設定フォーム

ブロック設定フォームは JSON Schema から動的に生成:

```vue
<template>
  <DynamicForm
    :schema="selectedBlock.configSchema"
    v-model="stepConfig"
  />
</template>
```

## 実装状況

| フェーズ | 状態 | 説明 |
|-------|--------|-------------|
| DB スキーマ | ✅ 完了 | `block_definitions`, `block_versions` テーブル |
| システムブロック | ✅ 完了 | 18 個のシステムブロック登録済み |
| 基盤ブロック | ✅ 完了 | 10 個の基盤/パターンブロック（継承階層） |
| 連携ブロック | ✅ 完了 | 11 個の外部連携ブロック（継承アーキテクチャに移行済み） |
| RAG ブロック | ✅ 完了 | 7 個の RAG ブロック（seed.sql） |
| サンドボックス (ctx) | ✅ 完了 | http, llm, workflow, human, adapter, embedding, vector |
| 管理者 API | ✅ 完了 | バージョン管理、ロールバック |
| フロントエンド | ✅ 完了 | StepPalette, PropertiesPanel |
| 多段継承 | ✅ 完了 | トポロジカルソート、最大深度 50 |

## ブロックグループ（制御フロー構造）

> **Updated**: 2026-01-15
> **Phase A + B Complete**: グループブロックは BlockDefinition に統合されました
> **See also**: [BLOCK_GROUP_REDESIGN.md](./designs/BLOCK_GROUP_REDESIGN.md)

ブロックグループは複数のステップの制御フローを管理するコンテナ構造です。入出力変換用の `pre_process`/`post_process` を持つブロックと同様の機能を提供します。

**Phase B: BlockDefinition 統合**

グループブロックは `block_definitions` テーブルで管理され、以下のフィールドで区別されます：
- `category`: `"group"`
- `group_kind`: `parallel` | `try_catch` | `foreach` | `while`
- `is_container`: `true`

これにより、グループブロックも通常のブロックと同様に Block Palette から選択・配置できます。

### ブロックグループタイプ（4 タイプのみ）

| タイプ | 説明 | 設定プロパティ |
|------|-------------|-------------------|
| `parallel` | 複数の独立したフローを並行実行 | `max_concurrent`, `fail_fast` |
| `try_catch` | リトライサポート付きエラーハンドリング | `retry_count`, `retry_delay_ms` |
| `foreach` | 配列要素に対して同じ処理を反復 | `input_path`, `parallel`, `max_workers` |
| `while` | 条件ベースのループ | `condition`, `max_iterations`, `do_while` |

### 削除されたタイプ

| タイプ | 代替 |
|------|-------------|
| `if_else` | `condition` システムブロックを使用 |
| `switch_case` | `switch` システムブロックを使用 |

### グループロール

すべてのグループは **`body` ロールのみ**を使用します。以前のロール（`try`, `catch`, `finally`, `then`, `else`, `default`, `case_N`）は削除されました。

エラーハンドリングは出力ポート経由で行います：
- `out` - 通常出力
- `error` - エラー出力（外部のエラーハンドリングブロックに接続）

### Pre/Post Process

通常のブロックと同様に、グループも JavaScript 変換をサポートします：

```javascript
// pre_process: 外部 IN → 内部 IN
return { ...input, timestamp: Date.now() };

// post_process: 内部 OUT → 外部 OUT
return { result: output.data, processed: true };
```

### ネスト

グループはネストできます（例: parallel 内に while）：

```
parallel
├── body
│   ├── step1
│   └── while（ネスト）
│       └── body
│           ├── step2
│           └── step3
└── body
    └── step4
```

### ブロックグループ vs システムブロック

| 機能 | ブロックグループ | condition/switch ブロック |
|---------|-------------|----------------------|
| ステップを含む | はい（body） | いいえ |
| Pre/Post Process | はい | いいえ（インラインロジック） |
| 出力ポート | out, error | then/else, case_N |
| ネスト | はい | N/A |
| ユースケース | 複雑な制御フロー | シンプルな分岐 |

## 関連ドキュメント

- [UNIFIED_BLOCK_MODEL.md](./designs/UNIFIED_BLOCK_MODEL.md) - **必読**: ブロック統一モデル詳細設計
- [BACKEND.md](./BACKEND.md) - バックエンドアーキテクチャ
- [API.md](./API.md) - API ドキュメント
- [DATABASE.md](./DATABASE.md) - データベーススキーマ
- [FRONTEND.md](./FRONTEND.md) - フロントエンドアーキテクチャ
