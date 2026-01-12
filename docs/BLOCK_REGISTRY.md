# Block Registry Design

Block definitions, error codes, and system block management.

> **Status**: ✅ Implemented (Unified Block Model)
> **Updated**: 2025-01-12
> **See also**: [UNIFIED_BLOCK_MODEL.md](./designs/UNIFIED_BLOCK_MODEL.md)
> **Migration**: `013_add_integration_blocks.sql` - 外部連携ブロック追加

## Quick Reference

| Item | Value |
|------|-------|
| Table | `block_definitions` |
| System Blocks | `tenant_id = NULL` (18 blocks) |
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
│  │  ├── start, llm, condition, loop, map, join, ...                 │ │
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

    // Metadata
    Enabled        bool
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

type ErrorCodeDef struct {
    Code        string `json:"code"`        // e.g., "LLM_001"
    Name        string `json:"name"`        // e.g., "RATE_LIMIT_EXCEEDED"
    Description string `json:"description"` // Human-readable description
    Retryable   bool   `json:"retryable"`   // Can this error be retried?
}
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
| `loop` | Loop | logic | for/forEach/while イテレーション |
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

### システムブロックのコード例

```javascript
// llm block
const prompt = renderTemplate(config.user_prompt || '', input);
const systemPrompt = config.system_prompt || '';

const response = await ctx.llm.chat(config.provider, config.model, {
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

const response = await ctx.http.request(url, {
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

const response = await ctx.http.post(webhookUrl, payload, {
    headers: { 'Content-Type': 'application/json' }
});

return { success: true, status: response.status };
```

```javascript
// github_create_issue block
const token = config.token || ctx.secrets.GITHUB_TOKEN;
const url = 'https://api.github.com/repos/' + config.owner + '/' + config.repo + '/issues';

const response = await ctx.http.post(url, {
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

const response = await ctx.http.post(webhookUrl, payload, {
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
| Sandbox (ctx) | ✅ 完了 | http, llm, workflow, human, adapter |
| Admin API | ✅ 完了 | バージョン管理、ロールバック |
| Frontend | ✅ 完了 | StepPalette, PropertiesPanel |

## Related Documents

- [UNIFIED_BLOCK_MODEL.md](./designs/UNIFIED_BLOCK_MODEL.md) - **必読**: ブロック統一モデル詳細設計
- [BACKEND.md](./BACKEND.md) - Backend architecture
- [API.md](./API.md) - API documentation
- [DATABASE.md](./DATABASE.md) - Database schema
- [FRONTEND.md](./FRONTEND.md) - Frontend architecture
