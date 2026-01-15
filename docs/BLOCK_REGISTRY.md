# Block Registry Reference

ブロック定義の API リファレンス（Source of Truth）。

> **Status**: ✅ Implemented (Unified Block Model)
> **Updated**: 2026-01-15
> **Role**: **API 仕様リファレンス**（このドキュメントが正）
> **See also**: [UNIFIED_BLOCK_MODEL.md](./designs/UNIFIED_BLOCK_MODEL.md) - 設計思想・アーキテクチャ

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

## Quick Reference

| Item | Value |
|------|-------|
| Table | `block_definitions` |
| System Blocks | `tenant_id = NULL` (25 blocks: 18 core + 7 RAG) |
| Tenant Blocks | `tenant_id = UUID` |
| Executor | Goja JavaScript VM |
| Version History | `block_versions` table |
| Categories | ai, logic, integration, data, control, utility |

## Overview

Block Registryはワークフローのステップタイプを管理するシステムです。
**Unified Block Model**により、すべてのブロックはJavaScriptコードとして統一実行されます。

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Unified Block Model                               │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │                   block_definitions テーブル                     │ │
│  │                                                                   │ │
│  │  System Blocks (tenant_id = NULL)                                │ │
│  │  ├── start, llm, condition, switch, map, join, ...               │ │
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
│  │    workflow: { run }                                              │ │
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

## Data Model

### BlockDefinition

```go
type BlockDefinition struct {
    ID             uuid.UUID       // Unique ID
    TenantID       *uuid.UUID      // NULL = system block, otherwise tenant-specific
    Slug           string          // Unique identifier (e.g., "llm", "discord")
    Name           string          // Display name
    Description    string          // Block description
    Category       string          // ai, logic, integration, data, control, utility

    // === Unified Block Model fields ===
    Code           string          // JavaScript code executed in sandbox
    UIConfig       json.RawMessage // {icon, color, configSchema}
    IsSystem       bool            // System blocks = admin only edit
    Version        int             // Version number, incremented on update

    // Schemas (JSON Schema format)
    ConfigSchema   json.RawMessage // Configuration options for the block
    InputSchema    json.RawMessage // Expected input structure
    OutputSchema   json.RawMessage // Output structure

    // Error handling
    ErrorCodes     []ErrorCodeDef  // Defined error codes for this block

    // === Block Inheritance/Extension fields ===
    ParentBlockID  *uuid.UUID      // Reference to parent block for inheritance
    ConfigDefaults json.RawMessage // Default values for parent's config_schema
    PreProcess     string          // JavaScript code for input transformation
    PostProcess    string          // JavaScript code for output transformation
    InternalSteps  []InternalStep  // Composite block internal steps

    // === Resolved fields (populated by backend) ===
    PreProcessChain        []string        // Chain of preProcess code (child→root)
    PostProcessChain       []string        // Chain of postProcess code (root→child)
    ResolvedCode           string          // Code from root ancestor
    ResolvedConfigDefaults json.RawMessage // Merged config defaults from chain

    // Metadata
    Enabled        bool
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

type InternalStep struct {
    Type      string          `json:"type"`       // Block slug to execute
    Config    json.RawMessage `json:"config"`     // Step configuration
    OutputKey string          `json:"output_key"` // Key for storing output
}

type ErrorCodeDef struct {
    Code        string `json:"code"`        // e.g., "LLM_001"
    Name        string `json:"name"`        // e.g., "RATE_LIMIT_EXCEEDED"
    Description string `json:"description"` // Human-readable description
    Retryable   bool   `json:"retryable"`   // Can this error be retried?
}
```

### Block Inheritance/Extension

ブロック継承により、既存ブロックを拡張して再利用可能なブロックを作成できます。

#### 継承の仕組み

```
┌──────────────────────────────────────────────────────────────────┐
│                  Block Inheritance Chain                          │
├──────────────────────────────────────────────────────────────────┤
│                                                                    │
│  discord-notify (child)                                           │
│    ├── parent_block_id: http (root)                              │
│    ├── config_defaults: { webhook_url: "...", method: "POST" }   │
│    ├── pre_process: "formats message for Discord"                │
│    └── post_process: null                                        │
│                                                                    │
│  Execution Flow:                                                   │
│  1. preProcess (discord-notify) - Format input                   │
│  2. Resolve config: merge config_defaults with step config       │
│  3. Execute http block code                                       │
│  4. postProcess (discord-notify) - Transform output              │
│                                                                    │
└──────────────────────────────────────────────────────────────────┘
```

#### 継承ルール

| ルール | 説明 |
|--------|------|
| コードを持つブロックのみ継承可能 | `Code != ""` |
| 最大継承深度 | 10レベル |
| 循環継承禁止 | A→B→C→A のような循環は不可 |
| テナント分離 | 同一テナント内またはシステムブロックからのみ継承可能 |

#### ConfigDefaults のマージ順序

```
root ancestor defaults
    ↓ (override)
middle ancestor defaults
    ↓ (override)
child defaults
    ↓ (override)
step config (execution time)
```

#### 継承ブロックの例

```javascript
// discord-error-notify (inherits from http block)
// parent_block_id: http ブロックのUUID
// config_defaults:
{
    "method": "POST",
    "headers": { "Content-Type": "application/json" }
}

// pre_process (入力変換):
const webhookUrl = ctx.secrets.DISCORD_ERROR_WEBHOOK || config.webhook_url;
return {
    url: webhookUrl,
    body: {
        content: "🚨 Error Alert",
        embeds: [{
            title: input.error_type || "Error",
            description: input.message,
            color: 15158332  // Red
        }]
    }
};

// post_process (出力変換):
return {
    success: input.status < 400,
    notified_at: new Date().toISOString()
};
```

#### InternalSteps（複合ブロック）

複数のブロックを順次実行する複合ブロックを作成できます：

```javascript
// enriched-http block
// internal_steps:
[
    {
        "type": "function",
        "config": { "code": "return { timestamp: Date.now(), ...input }" },
        "output_key": "enriched"
    },
    {
        "type": "http",
        "config": {},  // Uses merged config
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

### Database Schema

```sql
-- block_definitions テーブル（Unified Block Model対応）
CREATE TABLE block_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id),  -- NULL = system block
    slug VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(50) NOT NULL,
    icon VARCHAR(50),

    -- === Unified Block Model columns ===
    code TEXT,                              -- JavaScript code
    ui_config JSONB NOT NULL DEFAULT '{}',  -- {icon, color, configSchema}
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    version INTEGER NOT NULL DEFAULT 1,

    -- Schemas
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

    -- Snapshot
    code TEXT NOT NULL,
    config_schema JSONB NOT NULL,
    input_schema JSONB,
    output_schema JSONB,
    ui_config JSONB NOT NULL,

    -- Change tracking
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

## Input Schema

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

## Error Code System

### Standard Error Code Format

```
{BLOCK}_{NUMBER}_{TYPE}

Examples:
- LLM_001_RATE_LIMIT     - LLM rate limit exceeded
- LLM_002_INVALID_MODEL  - Invalid model specified
- HTTP_001_TIMEOUT       - HTTP request timeout
- HTTP_002_CONN_REFUSED  - Connection refused
- COND_001_INVALID_EXPR  - Invalid condition expression
- DISCORD_001_WEBHOOK_ERROR - Discord webhook error
```

### BlockError Structure

```go
type BlockError struct {
    Code       string          `json:"code"`        // Error code (e.g., "LLM_001")
    Message    string          `json:"message"`     // Human-readable message
    Details    json.RawMessage `json:"details"`     // Additional error details
    Retryable  bool            `json:"retryable"`   // Can this be retried?
    RetryAfter *time.Duration  `json:"retry_after"` // Suggested retry delay
}

func (e *BlockError) Error() string {
    return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}
```

### Error Code Categories

| Category | Range | Description |
|----------|-------|-------------|
| SYSTEM   | 000-099 | System-level errors |
| CONFIG   | 100-199 | Configuration errors |
| INPUT    | 200-299 | Input validation errors |
| EXEC     | 300-399 | Execution errors |
| OUTPUT   | 400-499 | Output processing errors |
| AUTH     | 500-599 | Authentication/authorization errors |
| RATE     | 600-699 | Rate limiting errors |
| TIMEOUT  | 700-799 | Timeout errors |

## System Blocks

システムブロック（`tenant_id = NULL`）は全ユーザーに提供されます。

### 現在のシステムブロック一覧

| Slug | Name | Category | Code概要 |
|------|------|----------|----------|
| `start` | Start | control | `return input;` |
| `llm` | LLM | ai | `ctx.llm.chat(...)` |
| `condition` | Condition | logic | `return {..., __branch: result ? 'then' : 'else'}` |
| `switch` | Switch | logic | 多分岐ルーティング |
| `map` | Map | data | 配列並列処理 |
| `join` | Join | data | ブランチマージ |
| `filter` | Filter | data | 配列フィルタリング |
| `split` | Split | data | バッチ分割 |
| `aggregate` | Aggregate | data | データ集約 |
| `tool` | Tool | integration | `ctx.adapter.call(...)` |
| `http` | HTTP Request | integration | `ctx.http.request(...)` |
| `subflow` | Subflow | control | `ctx.workflow.run(...)` |
| `wait` | Wait | control | 遅延・タイマー |
| `human_in_loop` | Human in Loop | control | `ctx.human.requestApproval(...)` |
| `error` | Error | control | `throw new Error(...)` |
| `router` | Router | ai | AI分類ルーティング |
| `note` | Note | utility | ドキュメント用（`return input;`） |
| `code` | Code | utility | ユーザー定義JavaScript |

### 外部連携ブロック一覧

| Slug | Name | Category | 説明 | 必要シークレット |
|------|------|----------|------|-----------------|
| `slack` | Slack | integration | Slackチャンネルにメッセージ送信 | `SLACK_WEBHOOK_URL` |
| `discord` | Discord | integration | Discord Webhookに通知 | `DISCORD_WEBHOOK_URL` |
| `notion_create_page` | Notion: ページ作成 | integration | Notionにページを作成 | `NOTION_API_KEY` |
| `notion_query_db` | Notion: DB検索 | integration | Notionデータベースを検索 | `NOTION_API_KEY` |
| `gsheets_append` | Google Sheets: 行追加 | integration | スプレッドシートに行を追加 | `GOOGLE_API_KEY` |
| `gsheets_read` | Google Sheets: 読み取り | integration | スプレッドシートから読み取り | `GOOGLE_API_KEY` |
| `github_create_issue` | GitHub: Issue作成 | integration | GitHubにIssueを作成 | `GITHUB_TOKEN` |
| `github_add_comment` | GitHub: コメント追加 | integration | Issue/PRにコメント追加 | `GITHUB_TOKEN` |
| `web_search` | Web検索 | integration | Tavily APIでWeb検索 | `TAVILY_API_KEY` |
| `email_sendgrid` | Email (SendGrid) | integration | SendGridでメール送信 | `SENDGRID_API_KEY` |
| `linear_create_issue` | Linear: Issue作成 | integration | LinearにIssueを作成 | `LINEAR_API_KEY` |

### RAGブロック一覧

| Slug | Name | Category | 説明 | 必要シークレット |
|------|------|----------|------|-----------------|
| `embedding` | Embedding | ai | テキストをベクトルに変換 | `OPENAI_API_KEY`, `COHERE_API_KEY`, `VOYAGE_API_KEY` |
| `vector-upsert` | Vector Upsert | data | ドキュメントをベクトルDBに保存 | - |
| `vector-search` | Vector Search | data | 類似ドキュメントを検索（ハイブリッド検索対応） | - |
| `vector-delete` | Vector Delete | data | ベクトルDBからドキュメント削除 | - |
| `doc-loader` | Document Loader | data | URL/テキストからドキュメント取得 | - |
| `text-splitter` | Text Splitter | data | テキストをチャンクに分割 | - |
| `rag-query` | RAG Query | ai | RAG検索+LLM生成（一括処理） | `OPENAI_API_KEY` |

### RAGブロック エラーコード一覧

| Code | Name | Block | Retryable | Description |
|------|------|-------|-----------|-------------|
| `EMB_001` | PROVIDER_ERROR | embedding | ✅ | Embedding provider API error |
| `EMB_002` | EMPTY_INPUT | embedding | ❌ | No text provided for embedding |
| `VEC_001` | COLLECTION_REQUIRED | vector-* | ❌ | Collection name is required |
| `VEC_002` | DOCUMENTS_REQUIRED | vector-upsert | ❌ | Documents array is required |
| `VEC_003` | VECTOR_OR_QUERY_REQUIRED | vector-search | ❌ | Either vector or query text is required |
| `VEC_004` | IDS_REQUIRED | vector-delete | ❌ | IDs array is required |
| `DOC_001` | FETCH_ERROR | doc-loader | ✅ | Failed to fetch URL (includes SSRF protection) |
| `DOC_002` | EMPTY_CONTENT | doc-loader | ❌ | No content provided |
| `TXT_001` | EMPTY_TEXT | text-splitter | ❌ | No text provided for splitting |
| `RAG_001` | QUERY_REQUIRED | rag-query | ❌ | Query text is required |
| `RAG_002` | COLLECTION_REQUIRED | rag-query | ❌ | Collection name is required |

### Goja Runtime Constraints (重要)

ブロックコードはGoja JavaScript VMで実行されます。以下の制約があります：

| 制約 | 説明 | 対処法 |
|------|------|--------|
| **`await`禁止** | gojaは`await`キーワードをサポートしない | `ctx.*`メソッドは同期的に呼び出す |
| **`async function`禁止** | async関数定義不可 | 通常の`function`を使用 |
| **`async () =>`禁止** | async arrow function不可 | 通常の`() =>`を使用 |

#### なぜ同期的に動作するか

`ctx.llm.chat()`, `ctx.http.get()`などのメソッドは、Go側で非同期処理を行い、結果が返るまでブロックします。
JavaScript側からは同期的な関数呼び出しに見えます。

```javascript
// ❌ NG: awaitは使用不可
const response = await ctx.llm.chat(...);

// ✅ OK: 同期的に呼び出す（内部でブロッキング）
const response = ctx.llm.chat(...);
```

#### バリデーション

seederコマンドはブロックコードをバリデーションし、`await`/`async`の使用を検出します：

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
// llm block
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
// http block
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
// slack block
const webhookUrl = config.webhook_url || ctx.secrets.SLACK_WEBHOOK_URL;
if (!webhookUrl) {
    throw new Error('[SLACK_001] Webhook URLが設定されていません');
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
// github_create_issue block
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

### RAGブロックのコード例

```javascript
// embedding block
// Supported providers: openai, cohere, voyage (Phase 3.3)
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

// Available models by provider:
// OpenAI: text-embedding-3-small (1536d), text-embedding-3-large (3072d)
// Cohere: embed-english-v3.0 (1024d), embed-multilingual-v3.0 (1024d)
// Voyage: voyage-3 (1024d), voyage-3-lite, voyage-code-3
```

```javascript
// vector-upsert block
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
// vector-search block
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
// vector-search with advanced filters (Phase 3.1)
// Supports: $eq, $ne, $gt, $gte, $lt, $lte, $in, $nin, $and, $or, $exists, $contains
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
// vector-search with hybrid search (Phase 3.2)
// Combines vector similarity + keyword search using RRF
const result = ctx.vector.query(config.collection, queryVector, {
    top_k: 10,
    keyword: "machine learning",  // Enable hybrid search
    hybrid_alpha: 0.7             // 70% vector, 30% keyword
});
```

```javascript
// rag-query block (RAG検索+LLM生成)
// 1. Embedding query
const embResult = ctx.embedding.embed(
    config.embedding_provider || 'openai',
    config.embedding_model || 'text-embedding-3-small',
    [input.query]
);

// 2. Vector search
const searchResult = ctx.vector.query(config.collection, embResult.vectors[0], {
    top_k: config.top_k || 5,
    include_content: true
});

// 3. Build context from retrieved documents
const context = searchResult.matches.map(m => m.content).join('\n\n---\n\n');

// 4. Generate response with LLM
const systemPrompt = config.system_prompt ||
    'Answer the question based on the following context. If the answer is not in the context, say so.';
const userPrompt = 'Context:\n' + context + '\n\nQuestion: ' + input.query;

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

## Adding New Blocks

### 標準手順（Migrationによる追加）

**⚠️ 必ず先に [UNIFIED_BLOCK_MODEL.md](./designs/UNIFIED_BLOCK_MODEL.md) を読むこと**

1. **Migrationファイル作成**: `backend/migrations/XXX_{name}_block.sql`

2. **INSERT文作成**:

```sql
INSERT INTO block_definitions (
    id, tenant_id, slug, name, description, category, icon,
    config_schema, error_codes, code, ui_config, is_system, enabled
) VALUES (
    gen_random_uuid(),
    NULL,  -- システムブロック（全ユーザーに提供）
    'discord',
    'Discord通知',
    'Discord Webhookにメッセージを送信',
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
    '[{"code": "DISCORD_001", "name": "WEBHOOK_ERROR", "description": "Webhook呼び出し失敗", "retryable": true}]',
    $code$
const webhookUrl = config.webhook_url || ctx.secrets.DISCORD_WEBHOOK_URL;
if (!webhookUrl) {
    throw new Error('Webhook URLが設定されていません');
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

3. **Migration実行**:
```bash
docker compose exec api migrate -path /migrations -database "$DATABASE_URL" up
```

4. **このドキュメントを更新**: システムブロック一覧に追加

### Go Adapterが必要なケース（例外）

以下の場合のみ、Go Adapterを実装：

| ケース | 理由 |
|--------|------|
| LLMプロバイダー追加 | `ctx.llm`経由で呼び出すため |
| 複雑な認証フロー | OAuth2等、JSでは困難な場合 |
| バイナリ処理 | 画像・ファイル処理等 |

Go Adapter追加手順:
1. Create `backend/internal/adapter/{name}.go`
2. Implement `Adapter` interface
3. Register in registry
4. Add test `{name}_test.go`
5. Update docs/BACKEND.md

## API Endpoints

### Tenant API

```
GET    /api/v1/blocks                    # リスト（システム + テナントブロック）
GET    /api/v1/blocks/{slug}             # 詳細取得
POST   /api/v1/blocks                    # カスタムブロック作成
PUT    /api/v1/blocks/{slug}             # 更新（テナント用のみ）
DELETE /api/v1/blocks/{slug}             # 削除（カスタムのみ）
```

### Admin API（システムブロック管理）

```
GET    /api/v1/admin/blocks              # システムブロック一覧
GET    /api/v1/admin/blocks/{id}         # 詳細
PUT    /api/v1/admin/blocks/{id}         # システムブロック編集
GET    /api/v1/admin/blocks/{id}/versions # バージョン履歴
POST   /api/v1/admin/blocks/{id}/rollback # ロールバック
```

## Frontend Integration

### Block Palette

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

### Dynamic Config Form

Block config formはJSON Schemaから動的に生成:

```vue
<template>
  <DynamicForm
    :schema="selectedBlock.configSchema"
    v-model="stepConfig"
  />
</template>
```

## Implementation Status

| Phase | Status | Description |
|-------|--------|-------------|
| DB Schema | ✅ 完了 | `block_definitions`, `block_versions` テーブル |
| System Blocks | ✅ 完了 | 18個のシステムブロック登録済み |
| Integration Blocks | ✅ 完了 | 11個の外部連携ブロック（013_add_integration_blocks.sql） |
| RAG Blocks | ✅ 完了 | 7個のRAGブロック（seed.sql） |
| Sandbox (ctx) | ✅ 完了 | http, llm, workflow, human, adapter, embedding, vector |
| Admin API | ✅ 完了 | バージョン管理、ロールバック |
| Frontend | ✅ 完了 | StepPalette, PropertiesPanel |

## Block Groups (Control Flow Constructs)

> **Updated**: 2026-01-15
> **Phase A + B Complete**: グループブロックはBlockDefinitionに統合されました
> **See also**: [BLOCK_GROUP_REDESIGN.md](./designs/BLOCK_GROUP_REDESIGN.md)

Block Groups are container constructs that manage control flow for multiple steps. They provide similar functionality to blocks with `pre_process`/`post_process` for input/output transformation.

**Phase B: BlockDefinition統合**

グループブロックは `block_definitions` テーブルで管理され、以下のフィールドで区別されます：
- `category`: `"group"`
- `group_kind`: `parallel` | `try_catch` | `foreach` | `while`
- `is_container`: `true`

これにより、グループブロックも通常のブロックと同様にBlock Paletteから選択・配置できます。

### Block Group Types (4 types only)

| Type | Description | Config Properties |
|------|-------------|-------------------|
| `parallel` | Execute multiple independent flows concurrently | `max_concurrent`, `fail_fast` |
| `try_catch` | Error handling with retry support | `retry_count`, `retry_delay_ms` |
| `foreach` | Iterate same process over array elements | `input_path`, `parallel`, `max_workers` |
| `while` | Condition-based loop | `condition`, `max_iterations`, `do_while` |

### Removed Types

| Type | Alternative |
|------|-------------|
| `if_else` | Use `condition` system block |
| `switch_case` | Use `switch` system block |

### Group Role

All groups now use **`body` role only**. Previous roles (`try`, `catch`, `finally`, `then`, `else`, `default`, `case_N`) have been removed.

Error handling is now done via output ports:
- `out` - Normal output
- `error` - Error output (connects to external error handling blocks)

### Pre/Post Process

Similar to regular blocks, groups support JavaScript transformation:

```javascript
// pre_process: external IN → internal IN
return { ...input, timestamp: Date.now() };

// post_process: internal OUT → external OUT
return { result: output.data, processed: true };
```

### Nesting

Groups can be nested (e.g., while inside parallel):

```
parallel
├── body
│   ├── step1
│   └── while (nested)
│       └── body
│           ├── step2
│           └── step3
└── body
    └── step4
```

### Block Group vs System Blocks

| Feature | Block Group | condition/switch Block |
|---------|-------------|----------------------|
| Contains steps | Yes (body) | No |
| Pre/Post Process | Yes | No (inline logic) |
| Output Ports | out, error | then/else, case_N |
| Nesting | Yes | N/A |
| Use Case | Complex control flow | Simple branching |

## Related Documents

- [UNIFIED_BLOCK_MODEL.md](./designs/UNIFIED_BLOCK_MODEL.md) - **必読**: ブロック統一モデル詳細設計
- [BACKEND.md](./BACKEND.md) - Backend architecture
- [API.md](./API.md) - API documentation
- [DATABASE.md](./DATABASE.md) - Database schema
- [FRONTEND.md](./FRONTEND.md) - Frontend architecture
