# Unified Block Model - 統一ブロックモデル設計

> **Status**: ✅ Implemented
> **Created**: 2025-01-12
> **Updated**: 2026-01-16
> **Author**: AI Agent

---

## 概要

すべてのブロックを「コード実行」として統一するアーキテクチャ設計。

### 設計原則

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│   実行エンジン = 任意のJavaScriptコード実行                    │
│                                                             │
│   ブロック = コード + UIメタデータ                            │
│                                                             │
│   ブロックタイプの違い = コードテンプレート + 設定UIの違い      │
│                                                             │
│   ctx = http / llm / workflow / human / adapter / ...       │
│       + secrets + env + log()                               │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## アーキテクチャ

### 実行フロー

```
Input (JSON)
    ↓
┌─────────────────────────────────────────┐
│           Block Executor                 │
├─────────────────────────────────────────┤
│  execute(code, input, sandbox) → output  │
└─────────────────────────────────────────┘
    ↓
Output (JSON)
```

### ctx インターフェース（コード内で利用可能なAPI）

ブロック内のコードから利用できるAPIは以下の通り：

```typescript
interface Context {
  // === HTTP ===
  http: {
    get(url: string, options?: RequestOptions): Promise<Response>;
    post(url: string, body: any, options?: RequestOptions): Promise<Response>;
    put(url: string, body: any, options?: RequestOptions): Promise<Response>;
    delete(url: string, options?: RequestOptions): Promise<Response>;
    request(url: string, options: RequestOptions): Promise<Response>;
  };

  // === LLM ===
  llm: {
    chat(provider: string, model: string, request: LLMRequest): Promise<LLMResponse>;
    complete(provider: string, model: string, prompt: string): Promise<string>;
  };

  // === Workflow（サブワークフロー呼び出し） ===
  workflow: {
    run(workflowId: string, input: any): Promise<any>;
  };

  // === Human（人間介入） ===
  human: {
    requestApproval(request: ApprovalRequest): Promise<ApprovalResult>;
    requestInput(request: InputRequest): Promise<InputResult>;
  };

  // === Adapter（登録済み外部連携） ===
  adapter: {
    call(adapterId: string, input: any): Promise<any>;
    list(): AdapterInfo[];
  };

  // === ユーティリティ ===
  secrets: Record<string, string>;
  env: Record<string, string>;
  log(level: 'debug' | 'info' | 'warn' | 'error', message: string, data?: any): void;
}
```

### コード例

```javascript
// HTTP呼び出し
const response = await ctx.http.post('https://api.example.com/users', {
  name: input.name
});

// LLM呼び出し
const answer = await ctx.llm.chat('openai', 'gpt-4', {
  messages: [{ role: 'user', content: input.question }]
});

// サブワークフロー呼び出し
const result = await ctx.workflow.run('workflow-id-123', input);

// 人間介入
const approval = await ctx.human.requestApproval({
  instructions: '承認してください',
  data: input
});

// アダプタ呼び出し
const slackResult = await ctx.adapter.call('slack', {
  channel: '#general',
  message: 'Hello'
});

// シークレット利用
const apiKey = ctx.secrets.OPENAI_API_KEY;

// ログ出力
ctx.log('info', 'Processing started', { inputSize: input.items.length });
```

---

## Block 定義

### データモデル

```typescript
interface Block {
  // === 識別子 ===
  id: string;           // UUID
  tenantId: string;     // テナントID

  // === 基本情報 ===
  name: string;         // 表示名
  description: string;  // 説明
  category: string;     // カテゴリ（UI整理用）

  // === コード ===
  code: string;         // 実行されるJavaScriptコード

  // === スキーマ ===
  inputSchema: JSONSchema;   // 入力スキーマ
  outputSchema: JSONSchema;  // 出力スキーマ

  // === UI メタデータ ===
  ui: {
    icon: string;            // アイコン
    color: string;           // カラー
    configSchema: JSONSchema; // 設定UIのスキーマ
  };

  // === 管理 ===
  isSystem: boolean;     // システムブロック（編集不可の場合true）
  isBuiltin: boolean;    // ビルトイン（削除不可）
  version: number;       // バージョン
  createdAt: string;
  updatedAt: string;
}
```

### システムブロック一覧

| ブロック | code の概要 | 編集可能 |
|---------|------------|---------|
| `start` | `return input` | ❌ |
| `code` | ユーザー定義（任意のJS） | ✅ |
| `http` | `ctx.http.request(...)` | ✅ |
| `llm` | `ctx.llm.chat(...)` | ✅ |
| `tool` | `ctx.adapter.call(...)` | ✅ |
| `branch` | `return {..., __branch: ...}` | ✅ |
| `parallel` | `Promise.all(ctx.workflow.run(...))` | ✅ |
| `subflow` | `ctx.workflow.run(...)` | ✅ |
| `human` | `ctx.human.requestApproval(...)` | ✅ |

---

## Block 継承アーキテクチャ

### 概要

多段継承により、認証パターンやサービス固有の設定を階層的に定義できます。
これにより、新規外部サービス連携を最小限のコードで追加できます。

### 継承階層

```
http (Level 0: Base)
├── webhook (Level 1: Pattern)
│   ├── slack (Level 2: Concrete)
│   └── discord (Level 2: Concrete)
│
├── rest-api (Level 1: Pattern)
│   ├── bearer-api (Level 2: Auth)
│   │   ├── github-api (Level 3: Service)
│   │   │   ├── github_create_issue (Level 4: Operation)
│   │   │   └── github_add_comment (Level 4: Operation)
│   │   ├── notion-api (Level 3: Service)
│   │   │   ├── notion_query_db (Level 4: Operation)
│   │   │   └── notion_create_page (Level 4: Operation)
│   │   └── email_sendgrid (Level 3: Concrete)
│   ├── api-key-header (Level 2: Auth)
│   │   └── web_search (Level 3: Concrete)
│   └── api-key-query (Level 2: Auth)
│       └── google-api (Level 3: Service)
│           ├── gsheets_append (Level 4: Operation)
│           └── gsheets_read (Level 4: Operation)
│
└── graphql (Level 1: Pattern)
    └── linear-api (Level 2: Service)
        └── linear_create_issue (Level 3: Operation)
```

### 各レベルの責務

| Level | 名称 | 責務 | 例 |
|-------|------|------|-----|
| 0 | Base | 基本的な実行ロジック（Code保持） | `http` |
| 1 | Pattern | 通信パターン、基本エラーハンドリング | `webhook`, `rest-api`, `graphql` |
| 2 | Auth | 認証方式の抽象化 | `bearer-api`, `api-key-header`, `api-key-query` |
| 3 | Service | サービス固有の設定（ベースURL、APIバージョン） | `github-api`, `notion-api` |
| 4+ | Operation | 具体的なAPI操作 | `github_create_issue`, `notion_query_db` |

### 実行フロー（Multi-Level Inheritance）

```
github_create_issue → github-api → bearer-api → rest-api → http

1. PreProcess Chain (child → root):
   github_create_issue.preProcess → github-api.preProcess →
   bearer-api.preProcess → rest-api.preProcess

2. Config Merge (root → child):
   rest-api.configDefaults ← bearer-api.configDefaults ←
   github-api.configDefaults ← github_create_issue.configDefaults
   ← step.config (runtime)

3. Execute Code (from root ancestor: http.code)

4. PostProcess Chain (root → child):
   rest-api.postProcess → bearer-api.postProcess →
   github-api.postProcess → github_create_issue.postProcess
```

### ConfigDefaults マージ順序

```
root ancestor defaults (rest-api)
    ↓ (override)
auth level defaults (bearer-api: auth_type=bearer)
    ↓ (override)
service defaults (github-api: base_url, secret_key)
    ↓ (override)
child defaults (github_create_issue: specific settings)
    ↓ (override)
step config (execution time)
```

### 継承ルール

| ルール | 説明 |
|--------|------|
| コードを持つブロックのみ継承可能 | `Code != ""` |
| 最大継承深度 | 50レベル（実用上は4-5レベル） |
| 循環継承禁止 | A→B→C→A のような循環は不可（トポロジカルソートで検出） |
| テナント分離 | 同一テナント内またはシステムブロックからのみ継承可能 |

### 継承ブロック定義例

```go
// integration.go - github_create_issue
func GitHubCreateIssueBlock() *SystemBlockDefinition {
    return &SystemBlockDefinition{
        Slug:            "github_create_issue",
        Version:         2,
        ParentBlockSlug: "github-api",  // 親ブロック
        PreProcess: `
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
`,
        PostProcess: `
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
`,
        // Code は親（github-api → bearer-api → rest-api → http）から継承
    }
}
```

### Seeder マイグレーションのトポロジカルソート

多段継承を正しく処理するため、Seeder は Kahn's Algorithm によるトポロジカルソートを使用：

```go
// migrator.go - topologicalSort
func topologicalSort(allBlocks []*blocks.SystemBlockDefinition) ([]*blocks.SystemBlockDefinition, error) {
    // 1. Build slug → block map
    blockMap := make(map[string]*blocks.SystemBlockDefinition)

    // 2. Calculate in-degree for each block
    inDegree := make(map[string]int)
    children := make(map[string][]string)

    for _, block := range allBlocks {
        if block.ParentBlockSlug != "" {
            inDegree[block.Slug]++
            children[block.ParentBlockSlug] = append(children[block.ParentBlockSlug], block.Slug)
        }
    }

    // 3. Start with blocks that have no dependencies (in-degree = 0)
    var queue []string
    for slug, degree := range inDegree {
        if degree == 0 {
            queue = append(queue, slug)
        }
    }

    // 4. Process queue, decrementing in-degree of children
    var sorted []*blocks.SystemBlockDefinition
    for len(queue) > 0 {
        slug := queue[0]
        queue = queue[1:]
        sorted = append(sorted, blockMap[slug])

        for _, childSlug := range children[slug] {
            inDegree[childSlug]--
            if inDegree[childSlug] == 0 {
                queue = append(queue, childSlug)
            }
        }
    }

    // 5. Check for cycles
    if len(sorted) != len(allBlocks) {
        return nil, fmt.Errorf("circular dependency detected")
    }

    return sorted, nil
}
```

### 新規サービス追加の簡略化

継承アーキテクチャにより、新規サービス追加が大幅に簡略化されます：

**Before（従来）**: ~50行のコード（認証、エラーハンドリング、URLヘッダー設定を全て手動）

**After（継承使用）**: ~20行のコード（固有のロジックのみ記述）

```javascript
// 例: Jira Issue作成を追加

// Step 1: jira-api 基盤ブロック作成（1回のみ）
{
    slug: "jira-api",
    parent_block_slug: "bearer-api",
    config_defaults: { "base_url": "https://{domain}.atlassian.net/rest/api/3" }
}

// Step 2: jira_create_issue 操作ブロック作成
{
    slug: "jira_create_issue",
    parent_block_slug: "jira-api",
    pre_process: `return { url: '/issue', method: 'POST', body: {...} };`,
    post_process: `return { key: input.body.key };`
}
```

---

## システムブロックの code テンプレート

### start

```javascript
// 入力をそのまま出力
return input;
```

### code（ユーザー定義）

```javascript
// ユーザーが自由にコードを記述
// 任意のctx APIを利用可能

const result = await ctx.http.get('https://api.example.com/data');
return {
  ...input,
  apiData: result.body
};
```

### http

```javascript
// HTTP呼び出し
const url = renderTemplate(config.url, input);

const response = await ctx.http.request(url, {
  method: config.method || 'POST',
  headers: config.headers || {},
  body: config.body ? renderTemplate(config.body, input) : input
});

return response;
```

### llm

```javascript
// LLM呼び出し
const prompt = renderTemplate(config.promptTemplate, input);

const response = await ctx.llm.chat(config.provider, config.model, {
  messages: [
    ...(config.systemPrompt ? [{ role: 'system', content: config.systemPrompt }] : []),
    { role: 'user', content: prompt }
  ],
  temperature: config.temperature ?? 0.7,
  maxTokens: config.maxTokens ?? 1000
});

return {
  content: response.content,
  usage: response.usage
};
```

### branch

```javascript
// 条件分岐
const result = evaluate(config.expression, input);

return {
  ...input,
  __branch: result ? 'then' : 'else'
};
```

### parallel

```javascript
// 並列実行
const items = getPath(input, config.inputPath) || [];

const results = await Promise.all(
  items.map(async (item, index) => {
    return await ctx.workflow.run(config.subWorkflowId, {
      item,
      index,
      parent: input
    });
  })
);

return {
  ...input,
  results,
  count: results.length
};
```

### subflow

```javascript
// サブワークフロー呼び出し
return await ctx.workflow.run(config.workflowId, input);
```

### human

```javascript
// 人間介入
return await ctx.human.requestApproval({
  instructions: config.instructions,
  timeout: config.timeoutHours,
  data: input,
  approvers: config.approvers
});
```

### tool（アダプタ呼び出し）

```javascript
// 登録済みアダプタを呼び出し
return await ctx.adapter.call(config.adapterId, input);
```

---

## 管理画面設計

### システムブロック管理

```
┌─────────────────────────────────────────────────────────────┐
│  システムブロック管理                           [+ 新規作成]  │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ 🔧 HTTP Request                              [編集][複製]│ │
│  │ HTTP APIを呼び出します                                   │ │
│  │ カテゴリ: External  |  v3  |  システム                   │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                              │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ 🤖 LLM Call                                  [編集][複製]│ │
│  │ LLM APIを呼び出します                                    │ │
│  │ カテゴリ: AI  |  v2  |  システム                         │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                              │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ ⑂ Branch                                    [編集][複製]│ │
│  │ 条件に基づいて分岐します                                  │ │
│  │ カテゴリ: Control Flow  |  v1  |  システム               │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### ブロック編集画面

```
┌─────────────────────────────────────────────────────────────┐
│  ブロック編集: LLM Call                                      │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  [基本情報] [コード] [入力スキーマ] [出力スキーマ] [UI設定]    │
│                                                              │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ 名前: LLM Call                                          │ │
│  │ 説明: LLM APIを呼び出し、応答を取得します                 │ │
│  │ カテゴリ: [AI ▼]                                         │ │
│  │ アイコン: 🤖                                              │ │
│  │ カラー: #8B5CF6                                          │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                              │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ // コードエディタ (Monaco Editor)                        │ │
│  │ async function execute(input, ctx) {                     │ │
│  │   const prompt = renderTemplate(                         │ │
│  │     config.promptTemplate,                               │ │
│  │     input                                                │ │
│  │   );                                                     │ │
│  │                                                          │ │
│  │   const response = await ctx.call(                       │ │
│  │     `llm://${config.provider}/${config.model}`,          │ │
│  │     {                                                    │ │
│  │       messages: [                                        │ │
│  │         { role: 'user', content: prompt }                │ │
│  │       ]                                                  │ │
│  │     }                                                    │ │
│  │   );                                                     │ │
│  │                                                          │ │
│  │   return { content: response.content };                  │ │
│  │ }                                                        │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                              │
│  [キャンセル]                      [テスト実行] [保存]       │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 設定スキーマ編集

```
┌─────────────────────────────────────────────────────────────┐
│  UI設定（configSchema）                                      │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ワークフローエディタでユーザーが設定する項目を定義します       │
│                                                              │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ + フィールド追加                                          │ │
│  │                                                          │ │
│  │ ├── provider (string)                        [編集][削除]│ │
│  │ │   表示名: プロバイダー                                  │ │
│  │ │   選択肢: openai, anthropic, google                    │ │
│  │ │                                                        │ │
│  │ ├── model (string)                           [編集][削除]│ │
│  │ │   表示名: モデル                                        │ │
│  │ │   依存: provider によって選択肢が変わる                  │ │
│  │ │                                                        │ │
│  │ ├── promptTemplate (string)                  [編集][削除]│ │
│  │ │   表示名: プロンプトテンプレート                         │ │
│  │ │   UI: textarea                                         │ │
│  │ │   プレースホルダー: ${input.message}                    │ │
│  │ │                                                        │ │
│  │ └── temperature (number)                     [編集][削除]│ │
│  │     表示名: Temperature                                   │ │
│  │     範囲: 0 - 2                                           │ │
│  │     デフォルト: 0.7                                        │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## データベーススキーマ

### blocks テーブル

```sql
CREATE TABLE blocks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),

    -- 基本情報
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100) NOT NULL DEFAULT 'custom',

    -- コード
    code TEXT NOT NULL,

    -- スキーマ (JSONB)
    input_schema JSONB NOT NULL DEFAULT '{}',
    output_schema JSONB NOT NULL DEFAULT '{}',

    -- UI メタデータ (JSONB)
    ui_config JSONB NOT NULL DEFAULT '{}',
    -- ui_config: {
    --   "icon": "🤖",
    --   "color": "#8B5CF6",
    --   "configSchema": { ... }
    -- }

    -- 管理フラグ
    is_system BOOLEAN NOT NULL DEFAULT false,
    is_builtin BOOLEAN NOT NULL DEFAULT false,

    -- バージョン管理
    version INTEGER NOT NULL DEFAULT 1,

    -- タイムスタンプ
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- 制約
    UNIQUE (tenant_id, name)
);

-- インデックス
CREATE INDEX idx_blocks_tenant_id ON blocks(tenant_id);
CREATE INDEX idx_blocks_category ON blocks(tenant_id, category);
CREATE INDEX idx_blocks_is_system ON blocks(tenant_id, is_system);
```

### block_versions テーブル（履歴管理）

```sql
CREATE TABLE block_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    block_id UUID NOT NULL REFERENCES blocks(id) ON DELETE CASCADE,

    -- バージョン情報
    version INTEGER NOT NULL,

    -- スナップショット
    code TEXT NOT NULL,
    input_schema JSONB NOT NULL,
    output_schema JSONB NOT NULL,
    ui_config JSONB NOT NULL,

    -- 変更情報
    change_summary TEXT,
    changed_by UUID,

    -- タイムスタンプ
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- 制約
    UNIQUE (block_id, version)
);

CREATE INDEX idx_block_versions_block_id ON block_versions(block_id);
```

---

## API 設計

### エンドポイント

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/blocks` | ブロック一覧取得 |
| GET | `/api/v1/blocks/:id` | ブロック詳細取得 |
| POST | `/api/v1/blocks` | ブロック作成 |
| PUT | `/api/v1/blocks/:id` | ブロック更新 |
| DELETE | `/api/v1/blocks/:id` | ブロック削除 |
| POST | `/api/v1/blocks/:id/duplicate` | ブロック複製 |
| POST | `/api/v1/blocks/:id/test` | ブロックテスト実行 |
| GET | `/api/v1/blocks/:id/versions` | バージョン履歴取得 |
| POST | `/api/v1/blocks/:id/rollback/:version` | バージョンロールバック |

### リクエスト/レスポンス例

#### GET /api/v1/blocks

```json
{
  "blocks": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "LLM Call",
      "description": "LLM APIを呼び出します",
      "category": "ai",
      "isSystem": true,
      "isBuiltin": true,
      "version": 2,
      "ui": {
        "icon": "🤖",
        "color": "#8B5CF6"
      }
    }
  ],
  "total": 15
}
```

#### POST /api/v1/blocks

```json
{
  "name": "Custom API Call",
  "description": "カスタムAPIを呼び出す",
  "category": "external",
  "code": "async function execute(input, ctx) {\n  return await ctx.call(config.url, input);\n}",
  "inputSchema": {
    "type": "object",
    "properties": {
      "data": { "type": "object" }
    }
  },
  "outputSchema": {
    "type": "object"
  },
  "ui": {
    "icon": "🔗",
    "color": "#10B981",
    "configSchema": {
      "type": "object",
      "properties": {
        "url": {
          "type": "string",
          "title": "API URL",
          "format": "uri"
        }
      },
      "required": ["url"]
    }
  }
}
```

#### POST /api/v1/blocks/:id/test

```json
{
  "input": {
    "message": "Hello, world!"
  },
  "config": {
    "provider": "openai",
    "model": "gpt-4",
    "promptTemplate": "Translate: ${input.message}"
  }
}
```

Response:

```json
{
  "success": true,
  "output": {
    "content": "こんにちは、世界！"
  },
  "executionTime": 1234,
  "logs": [
    { "level": "info", "message": "Calling LLM API", "timestamp": "..." }
  ]
}
```

---

## 実装状況

### Phase 1: データベース準備 ✅

- `block_definitions` テーブル拡張（code, ui_config, is_system, version カラム追加）
- `block_versions` テーブル作成（バージョン履歴管理）
- マイグレーション: `011_unified_block_model.sql`

### Phase 2: バックエンド実装 ✅

- Domain: `domain/block.go` - BlockVersion エンティティ追加
- Repository: `repository/postgres/block_version.go` - バージョン履歴CRUD
- Usecase: `usecase/block.go` - システムブロック管理ロジック
- Handler: `handler/block.go` - 管理者API追加

### Phase 3: Sandbox実装 ✅

- `block/sandbox/sandbox.go` - ctx サービスインターフェース拡張
  - LLMService: LLM API呼び出し
  - WorkflowService: サブワークフロー実行
  - HumanService: 人間介入リクエスト
  - AdapterService: アダプタ呼び出し

### Phase 4: フロントエンド実装 ✅

- 管理画面: `pages/admin/system-blocks.vue`
- Composable: `composables/useBlocks.ts` - useAdminBlocks()
- 型定義: `types/api.ts` - BlockDefinition拡張

### Phase 5: ワークフローエディタ統合 ✅

- 既存の `engine/executor.go` が sandbox を使用
- function ステップタイプで統合済み

### Phase 6: ドキュメント更新 ✅

- 本ドキュメント更新
- API.md に管理者API追加

### Phase 7: 多段継承アーキテクチャ ✅

- 基盤/パターンブロック10個追加（webhook, rest-api, graphql, bearer-api, api-key-header, api-key-query, github-api, notion-api, google-api, linear-api）
- 既存11個の外部連携ブロックを継承アーキテクチャにマイグレーション
- Seeder にトポロジカルソート（Kahn's Algorithm）を実装
- 最大継承深度50、循環依存検出

---

## セキュリティ考慮事項

### Sandbox のセキュリティ

| 脅威 | 対策 |
|-----|------|
| 無限ループ | タイムアウト設定（デフォルト30秒） |
| メモリ消費 | メモリ制限（Gojaの制限） |
| ファイルアクセス | Sandbox内でファイルAPI無効化 |
| ネットワーク制御 | ctx.http 経由のみ許可、直接fetch禁止 |
| シークレット漏洩 | ログにシークレット値を出力しない |
| コードインジェクション | ユーザー入力のサニタイズ |

### 権限管理

| 操作 | 必要権限 |
|-----|---------|
| ブロック閲覧 | `blocks:read` |
| ブロック作成 | `blocks:write` |
| システムブロック編集 | `blocks:admin` |
| ブロック削除 | `blocks:delete` |

---

## Sandbox 実装詳細

### Context 構造体

```go
// backend/internal/sandbox/context.go

package sandbox

import (
    "context"
    "encoding/json"
    "time"
)

// Context はブロック実行時に注入されるコンテキスト（JSの ctx オブジェクト）
type Context struct {
    goCtx    context.Context
    tenantID string

    // 各サービスへのアクセス
    HTTP     *HTTPService
    LLM      *LLMService
    Workflow *WorkflowService
    Human    *HumanService
    Adapter  *AdapterService

    // ユーティリティ
    Secrets map[string]string
    Env     map[string]string
    logs    []LogEntry
}

type LogEntry struct {
    Level     string    `json:"level"`
    Message   string    `json:"message"`
    Data      any       `json:"data,omitempty"`
    Timestamp time.Time `json:"timestamp"`
}

func (c *Context) Log(level, message string, data any) {
    c.logs = append(c.logs, LogEntry{
        Level:     level,
        Message:   message,
        Data:      data,
        Timestamp: time.Now(),
    })
}

func (c *Context) GetLogs() []LogEntry {
    return c.logs
}
```

### HTTP サービス

```go
// backend/internal/sandbox/http_service.go

package sandbox

import (
    "bytes"
    "context"
    "encoding/json"
    "io"
    "net/http"
    "time"
)

type HTTPService struct {
    client *http.Client
}

func NewHTTPService() *HTTPService {
    return &HTTPService{
        client: &http.Client{Timeout: 30 * time.Second},
    }
}

type RequestOptions struct {
    Method  string            `json:"method"`
    Headers map[string]string `json:"headers"`
    Body    any               `json:"body"`
}

type HTTPResponse struct {
    Status     int               `json:"status"`
    StatusText string            `json:"statusText"`
    Headers    map[string]string `json:"headers"`
    Body       any               `json:"body"`
}

func (s *HTTPService) Request(ctx context.Context, url string, opts RequestOptions) (*HTTPResponse, error) {
    method := opts.Method
    if method == "" {
        method = "GET"
    }

    var bodyReader io.Reader
    if opts.Body != nil {
        bodyBytes, _ := json.Marshal(opts.Body)
        bodyReader = bytes.NewReader(bodyBytes)
    }

    req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
    if err != nil {
        return nil, err
    }

    req.Header.Set("Content-Type", "application/json")
    for k, v := range opts.Headers {
        req.Header.Set(k, v)
    }

    resp, err := s.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)
    var bodyJSON any
    json.Unmarshal(body, &bodyJSON)

    headers := make(map[string]string)
    for k, v := range resp.Header {
        if len(v) > 0 {
            headers[k] = v[0]
        }
    }

    return &HTTPResponse{
        Status:     resp.StatusCode,
        StatusText: resp.Status,
        Headers:    headers,
        Body:       bodyJSON,
    }, nil
}

func (s *HTTPService) Get(ctx context.Context, url string, opts *RequestOptions) (*HTTPResponse, error) {
    o := RequestOptions{Method: "GET"}
    if opts != nil {
        o.Headers = opts.Headers
    }
    return s.Request(ctx, url, o)
}

func (s *HTTPService) Post(ctx context.Context, url string, body any, opts *RequestOptions) (*HTTPResponse, error) {
    o := RequestOptions{Method: "POST", Body: body}
    if opts != nil {
        o.Headers = opts.Headers
    }
    return s.Request(ctx, url, o)
}

func (s *HTTPService) Put(ctx context.Context, url string, body any, opts *RequestOptions) (*HTTPResponse, error) {
    o := RequestOptions{Method: "PUT", Body: body}
    if opts != nil {
        o.Headers = opts.Headers
    }
    return s.Request(ctx, url, o)
}

func (s *HTTPService) Delete(ctx context.Context, url string, opts *RequestOptions) (*HTTPResponse, error) {
    o := RequestOptions{Method: "DELETE"}
    if opts != nil {
        o.Headers = opts.Headers
    }
    return s.Request(ctx, url, o)
}
```

### LLM サービス

```go
// backend/internal/sandbox/llm_service.go

package sandbox

import (
    "context"
)

type LLMService struct {
    adapters map[string]LLMAdapter
}

type LLMAdapter interface {
    Chat(ctx context.Context, model string, req *LLMRequest) (*LLMResponse, error)
}

type LLMRequest struct {
    Messages    []Message `json:"messages"`
    Temperature float64   `json:"temperature"`
    MaxTokens   int       `json:"maxTokens"`
}

type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type LLMResponse struct {
    Content string   `json:"content"`
    Usage   LLMUsage `json:"usage"`
}

type LLMUsage struct {
    PromptTokens     int `json:"promptTokens"`
    CompletionTokens int `json:"completionTokens"`
    TotalTokens      int `json:"totalTokens"`
}

func NewLLMService(adapters map[string]LLMAdapter) *LLMService {
    return &LLMService{adapters: adapters}
}

func (s *LLMService) Chat(ctx context.Context, provider, model string, req *LLMRequest) (*LLMResponse, error) {
    adapter, ok := s.adapters[provider]
    if !ok {
        return nil, fmt.Errorf("unknown LLM provider: %s", provider)
    }
    return adapter.Chat(ctx, model, req)
}

func (s *LLMService) Complete(ctx context.Context, provider, model, prompt string) (string, error) {
    resp, err := s.Chat(ctx, provider, model, &LLMRequest{
        Messages: []Message{{Role: "user", Content: prompt}},
    })
    if err != nil {
        return "", err
    }
    return resp.Content, nil
}
```

### Workflow サービス

```go
// backend/internal/sandbox/workflow_service.go

package sandbox

import (
    "context"
    "encoding/json"
)

type WorkflowService struct {
    executor WorkflowExecutor
    tenantID string
}

type WorkflowExecutor interface {
    Execute(ctx context.Context, tenantID, workflowID string, input json.RawMessage) (json.RawMessage, error)
}

func NewWorkflowService(executor WorkflowExecutor, tenantID string) *WorkflowService {
    return &WorkflowService{executor: executor, tenantID: tenantID}
}

func (s *WorkflowService) Run(ctx context.Context, workflowID string, input any) (any, error) {
    inputJSON, _ := json.Marshal(input)
    resultJSON, err := s.executor.Execute(ctx, s.tenantID, workflowID, inputJSON)
    if err != nil {
        return nil, err
    }
    var result any
    json.Unmarshal(resultJSON, &result)
    return result, nil
}
```

### Human サービス

```go
// backend/internal/sandbox/human_service.go

package sandbox

import (
    "context"
)

type HumanService struct {
    store    HumanTaskStore
    tenantID string
    runID    string
    stepID   string
}

type HumanTaskStore interface {
    CreateTask(ctx context.Context, task *HumanTask) error
    WaitForCompletion(ctx context.Context, taskID string) (*HumanTaskResult, error)
}

type HumanTask struct {
    ID           string   `json:"id"`
    TenantID     string   `json:"tenantId"`
    RunID        string   `json:"runId"`
    StepID       string   `json:"stepId"`
    Type         string   `json:"type"`
    Instructions string   `json:"instructions"`
    Data         any      `json:"data"`
    TimeoutHours int      `json:"timeoutHours"`
    Approvers    []string `json:"approvers"`
    Status       string   `json:"status"`
}

type ApprovalRequest struct {
    Instructions string   `json:"instructions"`
    Timeout      int      `json:"timeout"`
    Data         any      `json:"data"`
    Approvers    []string `json:"approvers"`
}

type HumanTaskResult struct {
    Approved bool   `json:"approved"`
    Approver string `json:"approver"`
    Comment  string `json:"comment"`
    Data     any    `json:"data"`
}

func NewHumanService(store HumanTaskStore, tenantID, runID, stepID string) *HumanService {
    return &HumanService{store: store, tenantID: tenantID, runID: runID, stepID: stepID}
}

func (s *HumanService) RequestApproval(ctx context.Context, req ApprovalRequest) (*HumanTaskResult, error) {
    task := &HumanTask{
        TenantID:     s.tenantID,
        RunID:        s.runID,
        StepID:       s.stepID,
        Type:         "approval",
        Instructions: req.Instructions,
        Data:         req.Data,
        TimeoutHours: req.Timeout,
        Approvers:    req.Approvers,
        Status:       "pending",
    }

    if err := s.store.CreateTask(ctx, task); err != nil {
        return nil, err
    }

    return s.store.WaitForCompletion(ctx, task.ID)
}
```

### Adapter サービス

```go
// backend/internal/sandbox/adapter_service.go

package sandbox

import (
    "context"
    "encoding/json"
)

type AdapterService struct {
    registry AdapterRegistry
}

type AdapterRegistry interface {
    Get(id string) (Adapter, bool)
    List() []AdapterInfo
}

type Adapter interface {
    Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error)
}

type AdapterInfo struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description"`
}

func NewAdapterService(registry AdapterRegistry) *AdapterService {
    return &AdapterService{registry: registry}
}

func (s *AdapterService) Call(ctx context.Context, adapterID string, input any) (any, error) {
    adapter, ok := s.registry.Get(adapterID)
    if !ok {
        return nil, fmt.Errorf("unknown adapter: %s", adapterID)
    }

    inputJSON, _ := json.Marshal(input)
    resultJSON, err := adapter.Execute(ctx, inputJSON)
    if err != nil {
        return nil, err
    }

    var result any
    json.Unmarshal(resultJSON, &result)
    return result, nil
}

func (s *AdapterService) List() []AdapterInfo {
    return s.registry.List()
}
```

### Context ファクトリ

```go
// backend/internal/sandbox/factory.go

package sandbox

import (
    "context"
)

type ContextFactory struct {
    llmAdapters     map[string]LLMAdapter
    adapterRegistry AdapterRegistry
    workflowExec    WorkflowExecutor
    humanStore      HumanTaskStore
}

func NewContextFactory(
    llmAdapters map[string]LLMAdapter,
    adapterRegistry AdapterRegistry,
    workflowExec WorkflowExecutor,
    humanStore HumanTaskStore,
) *ContextFactory {
    return &ContextFactory{
        llmAdapters:     llmAdapters,
        adapterRegistry: adapterRegistry,
        workflowExec:    workflowExec,
        humanStore:      humanStore,
    }
}

type ContextConfig struct {
    TenantID string
    RunID    string
    StepID   string
    Secrets  map[string]string
    Env      map[string]string
}

func (f *ContextFactory) Create(goCtx context.Context, cfg ContextConfig) *Context {
    return &Context{
        goCtx:    goCtx,
        tenantID: cfg.TenantID,
        HTTP:     NewHTTPService(),
        LLM:      NewLLMService(f.llmAdapters),
        Workflow: NewWorkflowService(f.workflowExec, cfg.TenantID),
        Human:    NewHumanService(f.humanStore, cfg.TenantID, cfg.RunID, cfg.StepID),
        Adapter:  NewAdapterService(f.adapterRegistry),
        Secrets:  cfg.Secrets,
        Env:      cfg.Env,
        logs:     []LogEntry{},
    }
}
```

### JavaScript ランタイム統合

```go
// backend/internal/sandbox/runtime.go

package sandbox

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/dop251/goja"
)

type Runtime struct {
    timeout time.Duration
}

func NewRuntime(timeout time.Duration) *Runtime {
    return &Runtime{timeout: timeout}
}

type ExecuteResult struct {
    Output json.RawMessage
    Logs   []LogEntry
    Error  error
}

func (r *Runtime) Execute(goCtx context.Context, code string, input json.RawMessage, ctx *Context) *ExecuteResult {
    result := &ExecuteResult{}

    // タイムアウト付きコンテキスト
    goCtx, cancel := context.WithTimeout(goCtx, r.timeout)
    defer cancel()

    vm := goja.New()

    // 危険なグローバルを削除
    vm.Set("eval", goja.Undefined())
    vm.Set("Function", goja.Undefined())

    // input を設定
    var inputObj any
    json.Unmarshal(input, &inputObj)
    vm.Set("input", inputObj)

    // ctx オブジェクトを設定
    ctxObj := vm.NewObject()

    // ctx.http を設定
    httpObj := vm.NewObject()
    httpObj.Set("get", func(url string, opts map[string]any) any {
        resp, err := ctx.HTTP.Get(goCtx, url, nil)
        if err != nil {
            panic(vm.ToValue(err.Error()))
        }
        return resp
    })
    httpObj.Set("post", func(url string, body any, opts map[string]any) any {
        resp, err := ctx.HTTP.Post(goCtx, url, body, nil)
        if err != nil {
            panic(vm.ToValue(err.Error()))
        }
        return resp
    })
    httpObj.Set("put", func(url string, body any, opts map[string]any) any {
        resp, err := ctx.HTTP.Put(goCtx, url, body, nil)
        if err != nil {
            panic(vm.ToValue(err.Error()))
        }
        return resp
    })
    httpObj.Set("delete", func(url string, opts map[string]any) any {
        resp, err := ctx.HTTP.Delete(goCtx, url, nil)
        if err != nil {
            panic(vm.ToValue(err.Error()))
        }
        return resp
    })
    httpObj.Set("request", func(url string, opts map[string]any) any {
        reqOpts := RequestOptions{
            Method:  opts["method"].(string),
            Headers: opts["headers"].(map[string]string),
            Body:    opts["body"],
        }
        resp, err := ctx.HTTP.Request(goCtx, url, reqOpts)
        if err != nil {
            panic(vm.ToValue(err.Error()))
        }
        return resp
    })
    ctxObj.Set("http", httpObj)

    // ctx.llm を設定
    llmObj := vm.NewObject()
    llmObj.Set("chat", func(provider, model string, req map[string]any) any {
        llmReq := &LLMRequest{
            Temperature: req["temperature"].(float64),
            MaxTokens:   int(req["maxTokens"].(float64)),
        }
        // messages の変換
        if msgs, ok := req["messages"].([]any); ok {
            for _, m := range msgs {
                msg := m.(map[string]any)
                llmReq.Messages = append(llmReq.Messages, Message{
                    Role:    msg["role"].(string),
                    Content: msg["content"].(string),
                })
            }
        }
        resp, err := ctx.LLM.Chat(goCtx, provider, model, llmReq)
        if err != nil {
            panic(vm.ToValue(err.Error()))
        }
        return resp
    })
    llmObj.Set("complete", func(provider, model, prompt string) string {
        result, err := ctx.LLM.Complete(goCtx, provider, model, prompt)
        if err != nil {
            panic(vm.ToValue(err.Error()))
        }
        return result
    })
    ctxObj.Set("llm", llmObj)

    // ctx.workflow を設定
    workflowObj := vm.NewObject()
    workflowObj.Set("run", func(workflowID string, input any) any {
        result, err := ctx.Workflow.Run(goCtx, workflowID, input)
        if err != nil {
            panic(vm.ToValue(err.Error()))
        }
        return result
    })
    ctxObj.Set("workflow", workflowObj)

    // ctx.human を設定
    humanObj := vm.NewObject()
    humanObj.Set("requestApproval", func(req map[string]any) any {
        approvalReq := ApprovalRequest{
            Instructions: req["instructions"].(string),
            Timeout:      int(req["timeout"].(float64)),
            Data:         req["data"],
        }
        if approvers, ok := req["approvers"].([]any); ok {
            for _, a := range approvers {
                approvalReq.Approvers = append(approvalReq.Approvers, a.(string))
            }
        }
        result, err := ctx.Human.RequestApproval(goCtx, approvalReq)
        if err != nil {
            panic(vm.ToValue(err.Error()))
        }
        return result
    })
    ctxObj.Set("human", humanObj)

    // ctx.adapter を設定
    adapterObj := vm.NewObject()
    adapterObj.Set("call", func(adapterID string, input any) any {
        result, err := ctx.Adapter.Call(goCtx, adapterID, input)
        if err != nil {
            panic(vm.ToValue(err.Error()))
        }
        return result
    })
    adapterObj.Set("list", func() []AdapterInfo {
        return ctx.Adapter.List()
    })
    ctxObj.Set("adapter", adapterObj)

    // ctx.secrets, ctx.env を設定
    ctxObj.Set("secrets", ctx.Secrets)
    ctxObj.Set("env", ctx.Env)

    // ctx.log を設定
    ctxObj.Set("log", func(level, message string, data any) {
        ctx.Log(level, message, data)
    })

    vm.Set("ctx", ctxObj)

    // ヘルパー関数
    vm.RunString(`
        function getPath(obj, path) {
            if (!path || path === '$') return obj;
            const parts = path.replace(/^\$\.?/, '').split('.');
            let current = obj;
            for (const part of parts) {
                if (current == null) return undefined;
                current = current[part];
            }
            return current;
        }

        function renderTemplate(template, data) {
            return template.replace(/\$\{([^}]+)\}/g, (_, path) => {
                const value = getPath(data, path);
                return value !== undefined ? String(value) : '';
            });
        }

        function evaluate(expression, data) {
            const match = expression.match(/^\$\.(.+?)\s*(==|!=|>|<|>=|<=)\s*(.+)$/);
            if (match) {
                const [, path, op, rawValue] = match;
                const left = getPath(data, path);
                let right = rawValue.trim();
                if (right === 'true') right = true;
                else if (right === 'false') right = false;
                else if (right === 'null') right = null;
                else if (/^["'].*["']$/.test(right)) right = right.slice(1, -1);
                else if (!isNaN(Number(right))) right = Number(right);
                switch (op) {
                    case '==': return left == right;
                    case '!=': return left != right;
                    case '>': return left > right;
                    case '<': return left < right;
                    case '>=': return left >= right;
                    case '<=': return left <= right;
                }
            }
            return !!getPath(data, expression.replace(/^\$\.?/, ''));
        }
    `)

    // タイムアウト割り込み
    go func() {
        <-goCtx.Done()
        vm.Interrupt("execution timeout")
    }()

    // コード実行
    wrappedCode := fmt.Sprintf(`(async function() { %s })()`, code)
    val, err := vm.RunString(wrappedCode)
    if err != nil {
        result.Error = err
        result.Logs = ctx.GetLogs()
        return result
    }

    // Promise 解決待ち
    if promise, ok := val.Export().(*goja.Promise); ok {
        for promise.State() == goja.PromiseStatePending {
            select {
            case <-goCtx.Done():
                result.Error = goCtx.Err()
                result.Logs = ctx.GetLogs()
                return result
            default:
                time.Sleep(10 * time.Millisecond)
            }
        }
        if promise.State() == goja.PromiseStateRejected {
            result.Error = fmt.Errorf("promise rejected: %v", promise.Result().Export())
            result.Logs = ctx.GetLogs()
            return result
        }
        val = promise.Result()
    }

    output := val.Export()
    outputJSON, _ := json.Marshal(output)
    result.Output = outputJSON
    result.Logs = ctx.GetLogs()
    return result
}
```

---

## 関連ドキュメント

- [BACKEND.md](../BACKEND.md) - バックエンド実装パターン
- [API.md](../API.md) - API設計規約
- [DATABASE.md](../DATABASE.md) - データベース設計規約
- [FRONTEND.md](../FRONTEND.md) - フロントエンド実装パターン
