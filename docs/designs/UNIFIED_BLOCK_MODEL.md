# Unified Block Model - 統一ブロックモデル設計

> **Status**: Draft
> **Created**: 2025-01-12
> **Author**: AI Agent

---

## 概要

すべてのブロックを「コード実行」として統一するアーキテクチャ設計。

### 設計原則

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│   実行エンジン = コード実行のみ                               │
│                                                             │
│   ブロック = コード + UIメタデータ                            │
│                                                             │
│   ブロックタイプの違い = コードテンプレート + 設定UIの違い      │
│                                                             │
│   Sandbox = call() + secrets + env + log()                  │
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

### Sandbox インターフェース

```typescript
interface Sandbox {
  /**
   * 統一された外部呼び出し
   * すべての外部リソースへのアクセスはこれを通す
   */
  call(target: string, input: any): Promise<any>;

  /**
   * シークレット参照
   */
  secrets: Record<string, string>;

  /**
   * 環境変数参照
   */
  env: Record<string, string>;

  /**
   * ログ出力
   */
  log(level: 'debug' | 'info' | 'warn' | 'error', message: string, data?: any): void;
}
```

### call() プロトコル

| プロトコル | 形式 | 用途 | 例 |
|-----------|-----|------|-----|
| `https://` | URL | HTTPS呼び出し | `call('https://api.example.com/v1/users', {...})` |
| `http://` | URL | HTTP呼び出し | `call('http://internal-api/health', {})` |
| `llm://` | `llm://{provider}/{model}` | LLM API | `call('llm://openai/gpt-4', {messages: [...]})` |
| `adapter://` | `adapter://{id}` | 登録済みアダプタ | `call('adapter://slack', {channel: '...'})` |
| `workflow://` | `workflow://{id}` | サブワークフロー | `call('workflow://abc-123', {...})` |
| `human://` | `human://{type}` | 人間介入 | `call('human://approval', {instructions: '...'})` |

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
| `code` | ユーザー定義 | ✅ |
| `http` | `await call('https://...', input)` | ✅ |
| `llm` | `await call('llm://...', {...})` | ✅ |
| `tool` | `await call('adapter://...', input)` | ✅ |
| `branch` | `return {...input, __branch: eval(...)}` | ✅ |
| `parallel` | `await Promise.all(...)` | ✅ |
| `subflow` | `await call('workflow://...', input)` | ✅ |
| `human` | `await call('human://...', {...})` | ✅ |

---

## システムブロックの code テンプレート

### start

```javascript
// 入力をそのまま出力（バリデーション可）
async function execute(input, ctx) {
  return input;
}
```

### http

```javascript
// HTTP呼び出し
async function execute(input, ctx) {
  const url = config.url.replace(/\$\{([^}]+)\}/g, (_, key) => {
    return getPath(input, key) ?? '';
  });

  const response = await ctx.call(url, {
    method: config.method || 'POST',
    headers: config.headers || {},
    body: config.body ? renderTemplate(config.body, input) : input
  });

  return response;
}
```

### llm

```javascript
// LLM呼び出し
async function execute(input, ctx) {
  const prompt = renderTemplate(config.promptTemplate, input);

  const response = await ctx.call(
    `llm://${config.provider}/${config.model}`,
    {
      messages: [
        ...(config.systemPrompt ? [{ role: 'system', content: config.systemPrompt }] : []),
        { role: 'user', content: prompt }
      ],
      temperature: config.temperature ?? 0.7,
      maxTokens: config.maxTokens ?? 1000
    }
  );

  return {
    content: response.content,
    usage: response.usage
  };
}
```

### branch

```javascript
// 条件分岐
async function execute(input, ctx) {
  const result = evaluate(config.expression, input);

  return {
    ...input,
    __branch: result ? 'then' : 'else'
  };
}
```

### parallel

```javascript
// 並列実行
async function execute(input, ctx) {
  const items = getPath(input, config.inputPath) || [];

  const results = await Promise.all(
    items.map(async (item, index) => {
      // 各アイテムに対してサブブロックを実行
      return await ctx.call(`workflow://${config.subBlockId}`, {
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
}
```

### subflow

```javascript
// サブワークフロー呼び出し
async function execute(input, ctx) {
  return await ctx.call(`workflow://${config.workflowId}`, input);
}
```

### human

```javascript
// 人間介入
async function execute(input, ctx) {
  return await ctx.call('human://approval', {
    instructions: config.instructions,
    timeout: config.timeoutHours,
    data: input,
    approvers: config.approvers
  });
}
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

## 移行計画

### Phase 1: データベース準備

1. `blocks` テーブル作成
2. `block_versions` テーブル作成
3. 既存の Step Types をシステムブロックとして投入

### Phase 2: バックエンド実装

1. Block CRUD API 実装
2. Sandbox 実装（call() プロトコルルーティング）
3. コード実行エンジン統合
4. テスト実行 API 実装

### Phase 3: フロントエンド実装

1. ブロック管理画面（一覧・編集）
2. コードエディタ統合（Monaco Editor）
3. スキーマエディタ
4. テスト実行UI

### Phase 4: ワークフローエディタ統合

1. ブロックパレットをAPIから動的取得
2. ブロック設定UIを configSchema から動的生成
3. 既存ワークフローの移行

### Phase 5: 移行完了

1. 旧 Step Type 実行ロジック削除
2. ドキュメント更新
3. 移行ガイド作成

---

## セキュリティ考慮事項

### Sandbox のセキュリティ

| 脅威 | 対策 |
|-----|------|
| 無限ループ | タイムアウト設定（デフォルト30秒） |
| メモリ消費 | メモリ制限（Gojaの制限） |
| ファイルアクセス | Sandbox内でファイルAPI無効化 |
| ネットワーク制御 | call() 経由のみ許可、直接fetch禁止 |
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

### call() プロトコルルーティング

```go
// backend/internal/sandbox/sandbox.go

package sandbox

import (
    "context"
    "encoding/json"
    "fmt"
    "net/url"
    "strings"
)

// Sandbox はブロック実行時に注入されるコンテキスト
type Sandbox struct {
    ctx        context.Context
    tenantID   string
    secrets    map[string]string
    env        map[string]string
    logs       []LogEntry

    // プロトコルハンドラ
    handlers   map[string]ProtocolHandler
}

// ProtocolHandler は各プロトコルの処理を担当
type ProtocolHandler interface {
    Handle(ctx context.Context, target string, input json.RawMessage) (json.RawMessage, error)
}

// Call は統一された外部呼び出しインターフェース
func (s *Sandbox) Call(target string, input any) (any, error) {
    inputJSON, err := json.Marshal(input)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal input: %w", err)
    }

    // プロトコル判定
    protocol, path := s.parseTarget(target)

    handler, ok := s.handlers[protocol]
    if !ok {
        return nil, fmt.Errorf("unknown protocol: %s", protocol)
    }

    result, err := handler.Handle(s.ctx, path, inputJSON)
    if err != nil {
        return nil, err
    }

    var output any
    if err := json.Unmarshal(result, &output); err != nil {
        return nil, fmt.Errorf("failed to unmarshal output: %w", err)
    }

    return output, nil
}

// parseTarget はターゲット文字列からプロトコルとパスを抽出
func (s *Sandbox) parseTarget(target string) (protocol, path string) {
    // URL形式の場合
    if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
        return "http", target
    }

    // カスタムプロトコル形式: protocol://path
    if idx := strings.Index(target, "://"); idx != -1 {
        return target[:idx], target[idx+3:]
    }

    // デフォルトはHTTP
    return "http", target
}
```

### プロトコルハンドラ実装

#### HTTP ハンドラ

```go
// backend/internal/sandbox/handler_http.go

package sandbox

import (
    "bytes"
    "context"
    "encoding/json"
    "io"
    "net/http"
    "time"
)

type HTTPHandler struct {
    client  *http.Client
    secrets map[string]string
}

func NewHTTPHandler(secrets map[string]string) *HTTPHandler {
    return &HTTPHandler{
        client: &http.Client{
            Timeout: 30 * time.Second,
        },
        secrets: secrets,
    }
}

type HTTPRequest struct {
    Method  string            `json:"method"`
    Headers map[string]string `json:"headers"`
    Body    json.RawMessage   `json:"body"`
}

func (h *HTTPHandler) Handle(ctx context.Context, target string, input json.RawMessage) (json.RawMessage, error) {
    var req HTTPRequest
    if err := json.Unmarshal(input, &req); err != nil {
        // 入力がHTTPRequest形式でない場合、BODYとして扱う
        req = HTTPRequest{
            Method: "POST",
            Body:   input,
        }
    }

    if req.Method == "" {
        req.Method = "POST"
    }

    // シークレットの展開
    target = h.expandSecrets(target)
    for k, v := range req.Headers {
        req.Headers[k] = h.expandSecrets(v)
    }

    var bodyReader io.Reader
    if len(req.Body) > 0 {
        bodyReader = bytes.NewReader(req.Body)
    }

    httpReq, err := http.NewRequestWithContext(ctx, req.Method, target, bodyReader)
    if err != nil {
        return nil, err
    }

    httpReq.Header.Set("Content-Type", "application/json")
    for k, v := range req.Headers {
        httpReq.Header.Set(k, v)
    }

    resp, err := h.client.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    // レスポンスをラップ
    result := map[string]any{
        "status":     resp.StatusCode,
        "statusText": resp.Status,
        "headers":    resp.Header,
        "body":       json.RawMessage(body),
    }

    return json.Marshal(result)
}

func (h *HTTPHandler) expandSecrets(s string) string {
    for k, v := range h.secrets {
        s = strings.ReplaceAll(s, "${secrets."+k+"}", v)
    }
    return s
}
```

#### LLM ハンドラ

```go
// backend/internal/sandbox/handler_llm.go

package sandbox

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"
)

type LLMHandler struct {
    adapters map[string]LLMAdapter
}

type LLMAdapter interface {
    Chat(ctx context.Context, req *LLMRequest) (*LLMResponse, error)
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

func NewLLMHandler(adapters map[string]LLMAdapter) *LLMHandler {
    return &LLMHandler{adapters: adapters}
}

// Handle は llm://provider/model 形式のターゲットを処理
func (h *LLMHandler) Handle(ctx context.Context, target string, input json.RawMessage) (json.RawMessage, error) {
    // target: "openai/gpt-4" or "anthropic/claude-3-opus"
    parts := strings.SplitN(target, "/", 2)
    if len(parts) != 2 {
        return nil, fmt.Errorf("invalid LLM target format: %s (expected provider/model)", target)
    }

    provider, model := parts[0], parts[1]

    adapter, ok := h.adapters[provider]
    if !ok {
        return nil, fmt.Errorf("unknown LLM provider: %s", provider)
    }

    var req LLMRequest
    if err := json.Unmarshal(input, &req); err != nil {
        return nil, fmt.Errorf("invalid LLM request: %w", err)
    }

    // モデルをコンテキストに追加（アダプタが使用）
    ctx = context.WithValue(ctx, "model", model)

    resp, err := adapter.Chat(ctx, &req)
    if err != nil {
        return nil, err
    }

    return json.Marshal(resp)
}
```

#### Workflow ハンドラ（Subflow）

```go
// backend/internal/sandbox/handler_workflow.go

package sandbox

import (
    "context"
    "encoding/json"
    "fmt"
)

type WorkflowHandler struct {
    executor WorkflowExecutor
    tenantID string
}

type WorkflowExecutor interface {
    Execute(ctx context.Context, tenantID, workflowID string, input json.RawMessage) (json.RawMessage, error)
}

func NewWorkflowHandler(executor WorkflowExecutor, tenantID string) *WorkflowHandler {
    return &WorkflowHandler{
        executor: executor,
        tenantID: tenantID,
    }
}

// Handle は workflow://workflow-id 形式のターゲットを処理
func (h *WorkflowHandler) Handle(ctx context.Context, target string, input json.RawMessage) (json.RawMessage, error) {
    workflowID := target

    if workflowID == "" {
        return nil, fmt.Errorf("workflow ID is required")
    }

    return h.executor.Execute(ctx, h.tenantID, workflowID, input)
}
```

#### Human ハンドラ

```go
// backend/internal/sandbox/handler_human.go

package sandbox

import (
    "context"
    "encoding/json"
    "fmt"
)

type HumanHandler struct {
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
    ID           string          `json:"id"`
    TenantID     string          `json:"tenantId"`
    RunID        string          `json:"runId"`
    StepID       string          `json:"stepId"`
    Type         string          `json:"type"` // approval, input, review
    Instructions string          `json:"instructions"`
    Data         json.RawMessage `json:"data"`
    TimeoutHours int             `json:"timeoutHours"`
    Approvers    []string        `json:"approvers"`
    Status       string          `json:"status"` // pending, approved, rejected, timeout
}

type HumanTaskResult struct {
    Approved bool            `json:"approved"`
    Approver string          `json:"approver"`
    Comment  string          `json:"comment"`
    Data     json.RawMessage `json:"data"`
}

type HumanRequest struct {
    Instructions string   `json:"instructions"`
    Timeout      int      `json:"timeout"` // hours
    Data         any      `json:"data"`
    Approvers    []string `json:"approvers"`
}

func NewHumanHandler(store HumanTaskStore, tenantID, runID, stepID string) *HumanHandler {
    return &HumanHandler{
        store:    store,
        tenantID: tenantID,
        runID:    runID,
        stepID:   stepID,
    }
}

// Handle は human://type 形式のターゲットを処理
func (h *HumanHandler) Handle(ctx context.Context, target string, input json.RawMessage) (json.RawMessage, error) {
    taskType := target // "approval", "input", "review"
    if taskType == "" {
        taskType = "approval"
    }

    var req HumanRequest
    if err := json.Unmarshal(input, &req); err != nil {
        return nil, fmt.Errorf("invalid human request: %w", err)
    }

    dataJSON, _ := json.Marshal(req.Data)

    task := &HumanTask{
        TenantID:     h.tenantID,
        RunID:        h.runID,
        StepID:       h.stepID,
        Type:         taskType,
        Instructions: req.Instructions,
        Data:         dataJSON,
        TimeoutHours: req.Timeout,
        Approvers:    req.Approvers,
        Status:       "pending",
    }

    if err := h.store.CreateTask(ctx, task); err != nil {
        return nil, err
    }

    // タスク完了を待機（非同期の場合はここで中断）
    result, err := h.store.WaitForCompletion(ctx, task.ID)
    if err != nil {
        return nil, err
    }

    return json.Marshal(result)
}
```

#### Adapter ハンドラ

```go
// backend/internal/sandbox/handler_adapter.go

package sandbox

import (
    "context"
    "encoding/json"
    "fmt"
)

type AdapterHandler struct {
    registry AdapterRegistry
}

type AdapterRegistry interface {
    Get(id string) (Adapter, bool)
}

type Adapter interface {
    Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error)
}

func NewAdapterHandler(registry AdapterRegistry) *AdapterHandler {
    return &AdapterHandler{registry: registry}
}

// Handle は adapter://adapter-id 形式のターゲットを処理
func (h *AdapterHandler) Handle(ctx context.Context, target string, input json.RawMessage) (json.RawMessage, error) {
    adapterID := target

    adapter, ok := h.registry.Get(adapterID)
    if !ok {
        return nil, fmt.Errorf("unknown adapter: %s", adapterID)
    }

    return adapter.Execute(ctx, input)
}
```

### Sandbox ファクトリ

```go
// backend/internal/sandbox/factory.go

package sandbox

import (
    "context"
)

type SandboxFactory struct {
    llmAdapters     map[string]LLMAdapter
    adapterRegistry AdapterRegistry
    workflowExec    WorkflowExecutor
    humanStore      HumanTaskStore
}

func NewSandboxFactory(
    llmAdapters map[string]LLMAdapter,
    adapterRegistry AdapterRegistry,
    workflowExec WorkflowExecutor,
    humanStore HumanTaskStore,
) *SandboxFactory {
    return &SandboxFactory{
        llmAdapters:     llmAdapters,
        adapterRegistry: adapterRegistry,
        workflowExec:    workflowExec,
        humanStore:      humanStore,
    }
}

type SandboxConfig struct {
    TenantID string
    RunID    string
    StepID   string
    Secrets  map[string]string
    Env      map[string]string
}

func (f *SandboxFactory) Create(ctx context.Context, cfg SandboxConfig) *Sandbox {
    handlers := map[string]ProtocolHandler{
        "http":     NewHTTPHandler(cfg.Secrets),
        "https":    NewHTTPHandler(cfg.Secrets),
        "llm":      NewLLMHandler(f.llmAdapters),
        "adapter":  NewAdapterHandler(f.adapterRegistry),
        "workflow": NewWorkflowHandler(f.workflowExec, cfg.TenantID),
        "human":    NewHumanHandler(f.humanStore, cfg.TenantID, cfg.RunID, cfg.StepID),
    }

    return &Sandbox{
        ctx:      ctx,
        tenantID: cfg.TenantID,
        secrets:  cfg.Secrets,
        env:      cfg.Env,
        handlers: handlers,
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

func (r *Runtime) Execute(ctx context.Context, code string, input json.RawMessage, sandbox *Sandbox) *ExecuteResult {
    result := &ExecuteResult{}

    // タイムアウト付きコンテキスト
    ctx, cancel := context.WithTimeout(ctx, r.timeout)
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

    // ctx.call() を設定
    ctxObj.Set("call", func(call goja.FunctionCall) goja.Value {
        if len(call.Arguments) < 2 {
            panic(vm.ToValue("call requires target and input"))
        }

        target := call.Arguments[0].String()
        callInput := call.Arguments[1].Export()

        output, err := sandbox.Call(target, callInput)
        if err != nil {
            panic(vm.ToValue(err.Error()))
        }

        return vm.ToValue(output)
    })

    // ctx.secrets を設定
    ctxObj.Set("secrets", sandbox.secrets)

    // ctx.env を設定
    ctxObj.Set("env", sandbox.env)

    // ctx.log() を設定
    ctxObj.Set("log", func(call goja.FunctionCall) goja.Value {
        level := "info"
        message := ""
        var data any

        if len(call.Arguments) >= 1 {
            level = call.Arguments[0].String()
        }
        if len(call.Arguments) >= 2 {
            message = call.Arguments[1].String()
        }
        if len(call.Arguments) >= 3 {
            data = call.Arguments[2].Export()
        }

        sandbox.logs = append(sandbox.logs, LogEntry{
            Level:     level,
            Message:   message,
            Data:      data,
            Timestamp: time.Now(),
        })

        return goja.Undefined()
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
            // Simple expression evaluator for conditions
            const match = expression.match(/^\$\.(.+?)\s*(==|!=|>|<|>=|<=)\s*(.+)$/);
            if (match) {
                const [, path, op, rawValue] = match;
                const left = getPath(data, path);
                let right = rawValue.trim();

                // Parse right value
                if (right === 'true') right = true;
                else if (right === 'false') right = false;
                else if (right === 'null') right = null;
                else if (/^".*"$/.test(right) || /^'.*'$/.test(right)) right = right.slice(1, -1);
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
            // Truthy check
            return !!getPath(data, expression.replace(/^\$\.?/, ''));
        }
    `)

    // 割り込み設定（タイムアウト用）
    go func() {
        <-ctx.Done()
        vm.Interrupt("execution timeout")
    }()

    // コード実行
    wrappedCode := fmt.Sprintf(`
        (async function() {
            %s
        })()
    `, code)

    val, err := vm.RunString(wrappedCode)
    if err != nil {
        result.Error = err
        result.Logs = sandbox.logs
        return result
    }

    // Promise の解決を待つ
    promise, ok := val.Export().(*goja.Promise)
    if ok {
        // Promise が解決されるまで待つ
        for promise.State() == goja.PromiseStatePending {
            select {
            case <-ctx.Done():
                result.Error = ctx.Err()
                result.Logs = sandbox.logs
                return result
            default:
                time.Sleep(10 * time.Millisecond)
            }
        }

        if promise.State() == goja.PromiseStateRejected {
            result.Error = fmt.Errorf("promise rejected: %v", promise.Result().Export())
            result.Logs = sandbox.logs
            return result
        }

        val = promise.Result()
    }

    // 結果をJSONに変換
    output := val.Export()
    outputJSON, err := json.Marshal(output)
    if err != nil {
        result.Error = fmt.Errorf("failed to marshal output: %w", err)
        result.Logs = sandbox.logs
        return result
    }

    result.Output = outputJSON
    result.Logs = sandbox.logs
    return result
}

type LogEntry struct {
    Level     string    `json:"level"`
    Message   string    `json:"message"`
    Data      any       `json:"data,omitempty"`
    Timestamp time.Time `json:"timestamp"`
}
```

---

## 関連ドキュメント

- [BACKEND.md](../BACKEND.md) - バックエンド実装パターン
- [API.md](../API.md) - API設計規約
- [DATABASE.md](../DATABASE.md) - データベース設計規約
- [FRONTEND.md](../FRONTEND.md) - フロントエンド実装パターン
