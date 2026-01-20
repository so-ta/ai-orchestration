# Copilot機能のシステムワークフロー化 - 移行プラン

## 概要

現在Goコードで実装されているCopilotエージェントをシステムワークフローとして再実装する計画。

**設計方針**: ドッグフーディングの観点から、Copilot専用ブロックは作成せず、一般ユーザーも活用できる汎用ブロックのみで構成する。

---

## 設計原則

1. **ユーザーファースト**: 追加するブロックは一般ユーザーにもユースケースがあること
2. **汎用性**: Copilot固有の機能はブロックではなく設定・プロンプトで実現
3. **透明性**: ユーザーが同等のエージェントを自作できること

---

## 現状分析

### 現状の制限: `ctx.workflow.executeStep` 未実装

`agent`ブロックはツール呼び出しを`ctx.workflow.executeStep()`で行う設計だが、**現在未実装**：

```javascript
// agent block code
if (ctx.workflow && ctx.workflow.executeStep) {
    toolResult = ctx.workflow.executeStep(toolCall.function.name, args);
} else {
    toolResult = { error: 'Tool execution not available' };
}
```

```go
// workflow_service.go - ExecuteStep メソッドは存在しない
type WorkflowServiceImpl struct{}

func (s *WorkflowServiceImpl) Run(workflowID string, input map[string]interface{}) (map[string]interface{}, error) {
    return nil, fmt.Errorf("subflow execution not yet implemented")
}
```

---

## 汎用ブロックベースの設計

### Copilotツールの汎用化マッピング

全ツールを `http` ブロック → 内部API 形式で統一：

| Copilotツール | 内部API | 備考 |
|--------------|---------|------|
| `list_blocks` | `GET /api/v1/blocks` | 新規API追加 |
| `get_block_schema` | `GET /api/v1/blocks/{slug}` | 新規API追加 |
| `search_blocks` | `GET /api/v1/blocks?search=...` | 新規API追加 |
| `list_workflows` | `GET /api/v1/workflows` | 既存API |
| `get_workflow` | `GET /api/v1/workflows/{id}` | 既存API |
| `get_workflow_runs` | `GET /api/v1/workflows/{id}/runs` | 既存API |
| `create_step` | `POST /api/v1/workflows/{id}/steps` | 既存API |
| `update_step` | `PUT /api/v1/workflows/{id}/steps/{id}` | 既存API |
| `delete_step` | `DELETE /api/v1/workflows/{id}/steps/{id}` | 既存API |
| `create_edge` | `POST /api/v1/workflows/{id}/edges` | 既存API |
| `delete_edge` | `DELETE /api/v1/workflows/{id}/edges/{id}` | 既存API |
| `validate_workflow` | `POST /api/v1/workflows/{id}/validate` | 既存API |
| `search_documentation` | `rag-query` ブロック | RAG検索 |
| `diagnose_workflow` | `llm-structured` ブロック | LLM分析 |

### 新規追加が必要なAPI

```
GET  /api/v1/blocks              - ブロック一覧（カテゴリ/検索フィルタ対応）
GET  /api/v1/blocks/{slug}       - ブロック詳細（スキーマ含む）
```

### 結論: Copilot専用ブロックは不要

既存ブロック + 内部API + インフラ改善で対応可能。

---

## 必要なインフラ改善（汎用機能）

以下は Copilot だけでなく、**全ユーザーがエージェントを構築する際に必要な機能**：

### 1. `ctx.workflow.executeStep` の実装

**ユーザーユースケース**:
- カスタムエージェント構築時、ワークフロー内の他ステップをツールとして呼び出したい
- 例: 顧客サポートボットが「FAQ検索」「チケット作成」「通知送信」を動的に選択

**実装内容**:
```go
// WorkflowService インターフェース拡張
type WorkflowService interface {
    Run(workflowID string, input map[string]interface{}) (map[string]interface{}, error)
    ExecuteStep(stepName string, input map[string]interface{}) (map[string]interface{}, error) // 追加
}
```

### 2. `subflow` ブロックの完成

**ユーザーユースケース**:
- 共通処理をサブワークフローとして再利用
- 例: 「通知送信」ワークフローを複数のメインワークフローから呼び出し

**現状**: stub 実装のみ
**対応**: 完全実装

### 3. 内部API認証バイパス（システムワークフロー用）

**ユーザーユースケース**:
- システムワークフローが認証なしでプラットフォームAPIを呼び出し
- ユーザーワークフローは通常の認証が必要

**実装内容**:
- システムワークフロー実行時に内部APIトークンを自動付与
- `http` ブロックで `{{ctx.internal_api_token}}` を参照可能に

---

## 必要な汎用ブロック追加

以下のブロックは **一般ユーザーにもユースケースがある** ため追加を検討：

| ブロック | 一般ユーザーのユースケース | Copilotでの用途 |
|---------|--------------------------|----------------|
| `set-variables` | ワークフロー内で変数を設定・変換 | コンテキスト（tenant_id等）の注入 |

**注**: 上記以外の Copilot 専用ブロックは作成しない。

### `set-variables` ブロック仕様

```javascript
// 入力から変数を抽出・変換してコンテキストに設定
{
    slug: "set-variables",
    name: "変数設定",
    description: "ワークフロー内で使用する変数を設定します",
    category: "utility",
    configSchema: {
        type: "object",
        properties: {
            variables: {
                type: "array",
                items: {
                    type: "object",
                    properties: {
                        name: { type: "string" },
                        value: { type: "string" },  // テンプレート式対応
                        type: { enum: ["string", "number", "boolean", "json"] }
                    }
                }
            }
        }
    },
    code: `
        const result = {};
        for (const v of config.variables || []) {
            const value = renderTemplate(v.value, input);
            result[v.name] = v.type === 'number' ? Number(value) :
                            v.type === 'boolean' ? value === 'true' :
                            v.type === 'json' ? JSON.parse(value) : value;
        }
        return { ...input, ...result };
    `
}
```

---

## Copilotワークフロー構成（方式A採用）

### `agent` ブロック + `executeStep`

`ctx.workflow.executeStep` 実装後の構成：

```
[Start]
    ↓
[set-variables] ← tenant_id, project_id, mode 設定
    ↓
[memory-buffer: get] ← 会話履歴取得
    ↓
[agent] ← system_prompt、tools定義
  │
  │  ┌─ ブロック操作系ツール ─┐
  ├── [http: list_blocks]       ← GET /api/v1/blocks
  ├── [http: get_block_schema]  ← GET /api/v1/blocks/{slug}
  ├── [http: search_blocks]     ← GET /api/v1/blocks?search=...
  │
  │  ┌─ ワークフロー参照系ツール ─┐
  ├── [http: list_workflows]    ← GET /api/v1/workflows
  ├── [http: get_workflow]      ← GET /api/v1/workflows/{id}
  ├── [http: get_runs]          ← GET /api/v1/workflows/{id}/runs
  │
  │  ┌─ ワークフロー編集系ツール ─┐
  ├── [http: create_step]       ← POST /api/v1/workflows/{id}/steps
  ├── [http: update_step]       ← PUT /api/v1/workflows/{id}/steps/{id}
  ├── [http: delete_step]       ← DELETE /api/v1/workflows/{id}/steps/{id}
  ├── [http: create_edge]       ← POST /api/v1/workflows/{id}/edges
  ├── [http: delete_edge]       ← DELETE /api/v1/workflows/{id}/edges/{id}
  ├── [http: validate_workflow] ← POST /api/v1/workflows/{id}/validate
  │
  │  ┌─ 情報検索系ツール ─┐
  └── [rag-query: search_docs]  ← ドキュメント検索
    ↓
[memory-buffer: add] ← 応答を履歴に追加
    ↓
[End]
```

---

## 一般ユーザーへの価値

### `executeStep` 実装により可能になるエージェント例

**例1: リサーチ＆記事作成AI**
```
[agent] ← "〇〇についてリサーチして記事を書いて"
  │
  ├── [http: web_search]      ← Tavily/Google検索API
  ├── [http: fetch_page]      ← Webページ取得
  ├── [rag-query: knowledge]  ← 社内ナレッジ検索
  ├── [llm: summarize]        ← 要約生成
  └── [llm: write_article]    ← 記事執筆
```

**例2: 記帳仕訳AI**
```
[agent] ← "この取引を仕訳して"
  │
  ├── [http: get_accounts]        ← 勘定科目マスタ取得
  ├── [http: get_recent_entries]  ← 過去の仕訳パターン取得
  ├── [rag-query: rules]          ← 社内経理ルール検索
  ├── [llm-structured: classify]  ← 勘定科目分類
  └── [http: create_entry]        ← 仕訳登録
```

**例3: カスタマーサポートAI**
```
[agent] ← "ユーザーの問い合わせに回答"
  │
  ├── [rag-query: faq]            ← FAQ検索
  ├── [http: get_user_info]       ← ユーザー情報取得
  ├── [http: get_order_history]   ← 注文履歴取得
  ├── [http: create_ticket]       ← サポートチケット作成
  └── [http: send_notification]   ← 通知送信
```

### Copilotと同じパターンで構築可能

| 構成要素 | Copilot | ユーザーのエージェント |
|---------|---------|---------------------|
| 中核 | `agent` ブロック | `agent` ブロック |
| ツール | `http` → 内部API | `http` → 外部/内部API |
| 知識検索 | `rag-query` | `rag-query` |
| 分析 | `llm-structured` | `llm-structured` |
| 会話履歴 | `memory-buffer` | `memory-buffer` |

---

## 不採用: 方式B（明示的ループ）

`while` + `switch` による明示的な制御フロー：

```
[Start]
    ↓
[set-variables]
    ↓
[memory-buffer: get]
    ↓
┌─→ [llm-structured] ← tools定義、next_action を返す
│       ↓
│   [switch: next_action]
│       ├── "list_workflows" → [http] ──┐
│       ├── "get_workflow" → [http] ────┤
│       ├── "create_step" → [http] ─────┤
│       ├── "search_docs" → [rag-query] ┤
│       ├── "respond" → [format response] → Exit
│       └── default → [error]
│                               ↓
│   [aggregate: tool_results]
│       ↓
│   [condition: should_continue?]
│       ↓ yes
└───────┘
```

**比較**:
| 方式 | メリット | デメリット |
|------|---------|-----------|
| A: agent + executeStep | シンプル、agent ブロックの標準パターン | executeStep 実装が必要 |
| B: 明示的ループ | 現行インフラで実装可能 | 複雑、ワークフローが肥大化 |

**推奨**: 方式A（`executeStep` 実装を優先）

---

## 実装ロードマップ

### Phase 1: インフラ改善（汎用機能）

| タスク | 工数 | 優先度 |
|--------|------|-------|
| `ctx.workflow.executeStep` 実装 | 3日 | 高 |
| ブロックAPI追加（`GET /api/v1/blocks`, `GET /api/v1/blocks/{slug}`） | 1日 | 高 |
| `set-variables` ブロック追加 | 1日 | 高 |
| 内部API認証バイパス（システムワークフロー用） | 2日 | 中 |
| `subflow` ブロック完成 | 2日 | 低 |

### Phase 2: Copilotワークフロー構築

| タスク | 工数 | 優先度 |
|--------|------|-------|
| システムプロンプト整備 | 1日 | 高 |
| ツールステップ定義（http ブロック × 14ツール） | 2日 | 高 |
| RAGコレクション準備（ドキュメント） | 1日 | 中 |
| システムワークフロー定義 | 2日 | 高 |
| ハンドラー接続（既存エンドポイント → ワークフロー実行） | 1日 | 高 |

### Phase 3: テスト・検証

| タスク | 工数 | 優先度 |
|--------|------|-------|
| E2Eテスト | 2日 | 高 |
| パフォーマンス検証 | 1日 | 中 |

**合計**: 約 18日（3-4週間）

---

## 移行後のメリット

### プラットフォームとして

1. **ドッグフーディング**: Copilot が汎用機能のみで構成されることを証明
2. **拡張性**: ユーザーが Copilot と同等のエージェントを自作可能
3. **保守性**: 特殊なコードパスがなくなり、テスト・デバッグが容易

### ユーザーとして

1. **参考実装**: Copilot ワークフローを参考にエージェント構築方法を学習
2. **カスタマイズ**: Copilot をフォークして独自の AI アシスタントを作成
3. **信頼性**: プラットフォーム自身が使う機能は十分にテストされている

---

## リスクと対策

| リスク | 影響 | 対策 |
|--------|------|------|
| `executeStep` 実装の複雑さ | 高 | 段階的実装、まず同期呼び出しのみ |
| システムプロンプトの肥大化 | 中 | ブロックカタログを要約、RAG併用 |
| パフォーマンス低下 | 中 | HTTP呼び出しをバッチ化、キャッシュ |
| ストリーミング非対応 | 低 | 当面は Go 実装を維持、段階的移行 |

---

## 次のアクション

1. ✅ 現状分析完了
2. ⏳ `ctx.workflow.executeStep` 実装設計
3. ⏳ `set-variables` ブロック追加
4. ⏳ システムプロンプト（ブロックカタログ）整備

---

## 参考

- [n8n AI Agents](https://n8n.io/ai-agents/) - ツール統合パターン
- 既存 `agent` ブロック: `backend/internal/seed/blocks/ai.go`
- 既存 Copilot 実装: `backend/internal/copilot/agent/`
