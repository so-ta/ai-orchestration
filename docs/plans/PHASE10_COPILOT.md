# Phase 10: Copilot 実装計画

> **Status**: ✅ 実装済み
> **Updated**: 2025-01-12
>
> このフェーズは完全に実装済みです。実装の詳細は以下を参照:
> - `backend/internal/domain/copilot.go` - CopilotSession, CopilotMessage
> - `backend/internal/handler/copilot.go` - 20+ API endpoints (同期・非同期両方)
> - `backend/internal/usecase/copilot.go` - Business logic
> - `backend/internal/repository/postgres/copilot_session.go` - DB operations
>
> **実装済み機能:**
> - Generate, Suggest, Diagnose, Optimize, Explain
> - Chat（セッション管理あり）
> - 非同期実行（ポーリング対応）
> - セッション管理（作成・取得・一覧）

## Quick Reference

| Item | Value |
|------|-------|
| アーキテクチャ | メタワークフロー（Copilot機能をワークフローとして実装） |
| 実行方式 | 非同期 + ポーリング |
| TriggerType | `internal`（新規追加） |
| システムWF | `copilot-generate`, `copilot-diagnose`, `copilot-optimize`, `copilot-suggest` |

## 概要

**目的**: AIアシスタントによるワークフロー構築支援機能を提供し、ユーザーが自然言語でワークフローを設計・最適化できるようにする。

**ユースケース例**:
- 「顧客メールを分類してSlackに通知するワークフローを作って」→ 自動生成
- 「このワークフローにエラーハンドリングを追加して」→ 提案
- 「なぜこのステップが失敗したの？」→ 診断・解決策
- 「コストを下げる方法は？」→ 最適化提案

---

## 機能要件

### 1. 主要機能

| 機能 | 説明 |
|------|------|
| **Generate** | 自然言語からワークフロー生成 |
| **Suggest** | 次に追加すべきブロックを提案 |
| **Diagnose** | エラー診断と修正提案 |
| **Optimize** | パフォーマンス・コスト最適化提案 |
| **Explain** | ワークフローの説明生成 |

### 2. インタラクションモード

| モード | 説明 |
|--------|------|
| Chat | サイドパネルでの対話形式 |
| Inline | エディタ内での提案表示 |
| Command | スラッシュコマンド（`/generate`, `/optimize`） |

### 3. コンテキスト認識

- 現在のワークフロー構造
- 選択中のステップ
- 最近の実行結果・エラー
- 使用可能なアダプター一覧
- テナントの変数・シークレット

---

## 技術設計

### アーキテクチャ：メタワークフロー方式

**設計思想**: Copilot機能自体をワークフローとして定義し、サービス内のワークフロー実行エンジンで処理する（ドッグフーディング）。

```
┌─────────────────────────────────────────────────────────────────────┐
│                    メタワークフローアーキテクチャ                      │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  Frontend                                                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐               │
│  │ CopilotPanel │  │ SuggestionUI │  │CommandPalette│               │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘               │
│         │                 │                 │                        │
│         └─────────────────┴─────────────────┘                        │
│                           │                                          │
│                           ▼                                          │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │                    CopilotHandler                                │ │
│  │  POST /copilot/generate → run_id返却                             │ │
│  │  GET /copilot/runs/{id} → 結果取得（ポーリング）                  │ │
│  └──────────────────────────┬──────────────────────────────────────┘ │
│                             │                                        │
│                             ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │              RunUsecase.ExecuteSystemWorkflow()                  │ │
│  │                                                                   │ │
│  │  - slug: "copilot-generate"                                      │ │
│  │  - TriggerType: internal                                          │ │
│  │  - TriggerSource: "copilot"                                       │ │
│  │  - TriggerMetadata: {feature, user_id, session_id}               │ │
│  └──────────────────────────┬──────────────────────────────────────┘ │
│                             │                                        │
│                             ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │            System Workflow: "copilot-generate"                   │ │
│  │                                                                   │ │
│  │  [Start] → [Get Blocks] → [Build Prompt] → [LLM] → [Validate]   │ │
│  │                                                                   │ │
│  └──────────────────────────┬──────────────────────────────────────┘ │
│                             │                                        │
│                             ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │                 Workflow Engine (既存)                           │ │
│  │                                                                   │ │
│  │  Run作成 → Worker実行 → 結果保存                                 │ │
│  └─────────────────────────────────────────────────────────────────┘ │
│                                                                       │
└─────────────────────────────────────────────────────────────────────┘
```

### メリット

| メリット | 説明 |
|---------|------|
| **ドッグフーディング** | 自サービスの機能を使って自サービスを構築 |
| **カスタマイズ可能** | 管理者がCopilotの動作をワークフローとして編集可能 |
| **一貫性** | すべての処理が同じワークフローエンジンで実行される |
| **可観測性** | Copilot実行もRun/StepRunとして記録・トレース可能 |
| **拡張性** | 新しいCopilot機能もワークフローとして追加可能 |

---

## データモデル拡張

### TriggerType 追加

```go
// domain/run.go
type TriggerType string

const (
    TriggerTypeManual   TriggerType = "manual"   // UI操作
    TriggerTypeSchedule TriggerType = "schedule" // Cron
    TriggerTypeWebhook  TriggerType = "webhook"  // 外部API
    TriggerTypeInternal TriggerType = "internal" // 内部呼び出し（NEW）
)
```

### Run テーブル拡張

```sql
-- Migration: add_trigger_metadata.sql

-- runs テーブル拡張
ALTER TABLE runs ADD COLUMN trigger_source VARCHAR(100);
ALTER TABLE runs ADD COLUMN trigger_metadata JSONB DEFAULT '{}';

-- インデックス追加
CREATE INDEX idx_runs_trigger_source ON runs(trigger_source)
    WHERE trigger_source IS NOT NULL;

-- コメント
COMMENT ON COLUMN runs.trigger_source IS
    'Internal trigger source identifier: copilot, audit-system, etc.';
COMMENT ON COLUMN runs.trigger_metadata IS
    'Additional metadata about the trigger: feature, user_id, session_id, etc.';
```

### Run ドメインモデル拡張

```go
type Run struct {
    // 既存フィールド
    ID              uuid.UUID
    TenantID        uuid.UUID
    WorkflowID      uuid.UUID
    WorkflowVersion int
    Status          RunStatus
    Mode            RunMode
    TriggerType     TriggerType  // manual, schedule, webhook, internal
    Input           json.RawMessage
    Output          json.RawMessage

    // 新規フィールド
    TriggerSource   string          `json:"trigger_source,omitempty"`   // "copilot", "audit", etc.
    TriggerMetadata json.RawMessage `json:"trigger_metadata,omitempty"` // {"feature": "generate", ...}
}
```

### システムワークフロー

```sql
-- workflows テーブル拡張
ALTER TABLE workflows ADD COLUMN is_system BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE workflows ADD COLUMN system_slug VARCHAR(100);

-- ユニーク制約（システムワークフローはslugで一意）
CREATE UNIQUE INDEX idx_workflows_system_slug ON workflows(system_slug)
    WHERE system_slug IS NOT NULL;
```

---

## API設計

### エンドポイント

| Method | Path | 説明 |
|--------|------|------|
| POST | `/api/v1/copilot/generate` | WF生成リクエスト → run_id返却 |
| POST | `/api/v1/copilot/suggest` | 次ステップ提案 → run_id返却 |
| POST | `/api/v1/copilot/diagnose` | エラー診断 → run_id返却 |
| POST | `/api/v1/copilot/optimize` | 最適化提案 → run_id返却 |
| GET | `/api/v1/copilot/runs/{id}` | Copilot実行結果取得（ポーリング用） |

### Request/Response

**ワークフロー生成（非同期）**:

```json
// POST /api/v1/copilot/generate
{
  "prompt": "顧客メールを分類してSlackに通知するワークフローを作って",
  "session_id": "optional-session-id"
}

// Response (即座に返却)
{
  "run_id": "uuid",
  "status": "pending"
}
```

**結果取得（ポーリング）**:

```json
// GET /api/v1/copilot/runs/{id}

// Response (実行中)
{
  "run_id": "uuid",
  "status": "running",
  "started_at": "2025-01-12T10:00:00Z"
}

// Response (完了)
{
  "run_id": "uuid",
  "status": "completed",
  "completed_at": "2025-01-12T10:00:05Z",
  "output": {
    "workflow": {
      "name": "Customer Email Classifier",
      "description": "...",
      "steps": [...],
      "edges": [...]
    },
    "explanation": "このワークフローは..."
  }
}

// Response (失敗)
{
  "run_id": "uuid",
  "status": "failed",
  "error": "LLM API rate limit exceeded"
}
```

---

## Backend実装

### 内部呼び出しインターフェース

```go
// usecase/run.go

type InternalTriggerOptions struct {
    Source   string                 // "copilot", "audit", etc.
    Feature  string                 // "generate", "diagnose", etc.
    Metadata map[string]interface{} // 任意の追加情報
}

func (u *RunUsecase) ExecuteSystemWorkflow(
    ctx context.Context,
    slug string,                    // "copilot-generate"
    input map[string]interface{},
    opts InternalTriggerOptions,
) (*domain.Run, error) {
    // 1. システムワークフロー取得
    workflow, err := u.workflowRepo.GetSystemBySlug(ctx, slug)
    if err != nil {
        return nil, ErrSystemWorkflowNotFound
    }

    // 2. Run作成
    run := domain.NewRun(
        workflow.ID,
        workflow.Version,
        domain.RunModeProduction,
        domain.TriggerTypeInternal,
    )
    run.TriggerSource = opts.Source
    run.TriggerMetadata = toJSON(map[string]interface{}{
        "feature":  opts.Feature,
        "metadata": opts.Metadata,
    })
    run.Input = toJSON(input)

    // 3. 保存 & キュー投入
    if err := u.runRepo.Create(ctx, run); err != nil {
        return nil, err
    }

    // 4. ジョブキューに投入
    if err := u.jobQueue.Enqueue(ctx, run.ID); err != nil {
        return nil, err
    }

    return run, nil
}
```

### CopilotHandler

```go
// handler/copilot.go

type CopilotHandler struct {
    runUsecase usecase.RunUsecase
}

func (h *CopilotHandler) Generate(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Prompt    string `json:"prompt"`
        SessionID string `json:"session_id,omitempty"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    userID := getUserID(r.Context())
    tenantID := getTenantID(r.Context())

    run, err := h.runUsecase.ExecuteSystemWorkflow(
        r.Context(),
        "copilot-generate",
        map[string]interface{}{
            "prompt":    req.Prompt,
            "tenant_id": tenantID.String(),
        },
        usecase.InternalTriggerOptions{
            Source:  "copilot",
            Feature: "generate",
            Metadata: map[string]interface{}{
                "user_id":    userID.String(),
                "session_id": req.SessionID,
            },
        },
    )
    if err != nil {
        respondError(w, err)
        return
    }

    json.NewEncoder(w).Encode(map[string]interface{}{
        "run_id": run.ID,
        "status": run.Status,
    })
}

func (h *CopilotHandler) GetRun(w http.ResponseWriter, r *http.Request) {
    runID := chi.URLParam(r, "id")

    run, err := h.runUsecase.GetByID(r.Context(), uuid.MustParse(runID))
    if err != nil {
        respondError(w, err)
        return
    }

    // Copilot用のレスポンス形式に変換
    resp := map[string]interface{}{
        "run_id": run.ID,
        "status": run.Status,
    }

    if run.StartedAt != nil {
        resp["started_at"] = run.StartedAt
    }
    if run.CompletedAt != nil {
        resp["completed_at"] = run.CompletedAt
    }
    if run.Status == domain.RunStatusCompleted {
        resp["output"] = run.Output
    }
    if run.Status == domain.RunStatusFailed {
        resp["error"] = run.Error
    }

    json.NewEncoder(w).Encode(resp)
}
```

---

## システムワークフロー定義

### copilot-generate

```
[Start]
   ↓
[Get Available Blocks]
   code: return { blocks: await ctx.blocks.list() }
   ↓
[Get Available Adapters]
   code: return { adapters: await ctx.adapter.list() }
   ↓
[Build Prompt]
   code: // プロンプトテンプレート構築
   ↓
[LLM Call]
   type: llm
   config: { provider: "openai", model: "gpt-4o", prompt: "{{$.prompt}}" }
   ↓
[Parse Response]
   code: return JSON.parse(input.content)
   ↓
[Validate Workflow]
   code: // 構造検証
   ↓
[Return Result]
```

### Migration例

```sql
-- backend/migrations/XXX_copilot_system_workflows.sql

-- 1. copilot-generate ワークフロー
INSERT INTO workflows (
    id, tenant_id, name, description, status,
    is_system, system_slug, version
) VALUES (
    'a0000000-0000-0000-0000-000000000001',
    NULL,  -- システムワークフロー
    'Copilot: Generate Workflow',
    '自然言語からワークフローを生成するシステムワークフロー',
    'published',
    TRUE,
    'copilot-generate',
    1
);

-- 2. ステップ定義
INSERT INTO steps (id, workflow_id, name, type, config, position_x, position_y) VALUES
('b0000001-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001',
 'Start', 'start', '{}', 0, 0),

('b0000002-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001',
 'Get Blocks', 'code',
 '{"code": "return { blocks: await ctx.blocks.list() }"}', 200, 0),

('b0000003-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001',
 'Build Prompt', 'code',
 '{"code": "const prompt = `...`; return { prompt };"}', 400, 0),

('b0000004-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001',
 'Generate with LLM', 'llm',
 '{"provider": "openai", "model": "gpt-4o", "user_prompt": "{{$.prompt}}"}', 600, 0),

('b0000005-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001',
 'Parse & Validate', 'code',
 '{"code": "const parsed = JSON.parse(input.content); return parsed;"}', 800, 0);

-- 3. エッジ定義
INSERT INTO edges (workflow_id, source_step_id, target_step_id) VALUES
('a0000000-0000-0000-0000-000000000001', 'b0000001-0000-0000-0000-000000000001', 'b0000002-0000-0000-0000-000000000001'),
('a0000000-0000-0000-0000-000000000001', 'b0000002-0000-0000-0000-000000000001', 'b0000003-0000-0000-0000-000000000001'),
('a0000000-0000-0000-0000-000000000001', 'b0000003-0000-0000-0000-000000000001', 'b0000004-0000-0000-0000-000000000001'),
('a0000000-0000-0000-0000-000000000001', 'b0000004-0000-0000-0000-000000000001', 'b0000005-0000-0000-0000-000000000001');
```

---

## ctx インターフェース拡張

Copilotワークフロー内で使用する新しいctx機能：

```javascript
// ctx.blocks - ブロック定義操作
ctx.blocks.list()              // ブロック一覧取得
ctx.blocks.get(slug)           // 特定ブロック取得

// ctx.workflows - ワークフロー操作（読み取りのみ）
ctx.workflows.get(id)          // ワークフロー取得
ctx.workflows.list()           // ワークフロー一覧

// ctx.runs - Run操作（読み取りのみ）
ctx.runs.get(id)               // Run取得
ctx.runs.getStepRuns(runId)    // StepRun一覧取得
```

---

## フロントエンド設計

### コンポーネント構成

```
frontend/
├── components/
│   └── copilot/
│       ├── CopilotPanel.vue      # サイドパネル全体
│       ├── CopilotChat.vue       # チャットUI
│       ├── CopilotInput.vue      # 入力フォーム
│       ├── CopilotMessage.vue    # メッセージ表示
│       ├── CopilotLoading.vue    # ローディング表示
│       ├── SuggestionList.vue    # 提案リスト
│       ├── DiagnosisCard.vue     # 診断結果表示
│       └── OptimizationCard.vue  # 最適化提案表示
└── composables/
    └── useCopilot.ts             # Copilot API呼び出し（ポーリング対応）
```

### useCopilot.ts（ポーリング対応）

```typescript
export function useCopilot() {
  const messages = ref<CopilotMessage[]>([])
  const polling = ref(false)
  const currentRunId = ref<string | null>(null)
  const { $api } = useNuxtApp()

  async function generate(prompt: string): Promise<CopilotResult | null> {
    // 1. リクエスト送信
    const { run_id } = await $api.post('/copilot/generate', { prompt })
    currentRunId.value = run_id

    // ユーザーメッセージ追加
    messages.value.push({
      id: nanoid(),
      role: 'user',
      type: 'text',
      content: prompt
    })

    // ローディングメッセージ追加
    const loadingMsg: CopilotMessage = {
      id: nanoid(),
      role: 'assistant',
      type: 'loading',
      content: 'ワークフローを生成中...'
    }
    messages.value.push(loadingMsg)

    // 2. ポーリング開始
    polling.value = true

    while (polling.value) {
      await sleep(1000)  // 1秒間隔

      const run = await $api.get(`/copilot/runs/${run_id}`)

      if (run.status === 'completed') {
        // ローディングを結果に置換
        const idx = messages.value.findIndex(m => m.id === loadingMsg.id)
        if (idx >= 0) {
          messages.value[idx] = {
            id: loadingMsg.id,
            role: 'assistant',
            type: 'workflow',
            content: run.output.explanation,
            data: run.output.workflow
          }
        }
        polling.value = false
        return run.output

      } else if (run.status === 'failed') {
        // ローディングをエラーに置換
        const idx = messages.value.findIndex(m => m.id === loadingMsg.id)
        if (idx >= 0) {
          messages.value[idx] = {
            id: loadingMsg.id,
            role: 'assistant',
            type: 'error',
            content: run.error
          }
        }
        polling.value = false
        return null
      }
      // running の場合は継続
    }

    return null
  }

  function cancel() {
    polling.value = false
    currentRunId.value = null
  }

  // 他の機能（diagnose, optimize, suggest）も同様のパターン

  return {
    messages,
    polling,
    currentRunId,
    generate,
    cancel
  }
}
```

### CopilotPanel.vue

```vue
<template>
  <aside class="copilot-panel" :class="{ open: isOpen }">
    <header class="copilot-header">
      <h3>
        <Icon name="sparkles" />
        Copilot
      </h3>
      <button @click="close">
        <Icon name="x" />
      </button>
    </header>

    <div class="copilot-content">
      <!-- チャット履歴 -->
      <div class="messages" ref="messagesRef">
        <CopilotMessage
          v-for="msg in messages"
          :key="msg.id"
          :message="msg"
          @apply="applyWorkflow"
        />
      </div>

      <!-- クイックアクション -->
      <div class="quick-actions" v-if="messages.length === 0">
        <button @click="showGenerateInput = true">
          <Icon name="wand" />
          ワークフロー生成
        </button>
        <button @click="optimizeWorkflow" :disabled="!hasWorkflow || polling">
          <Icon name="zap" />
          最適化提案
        </button>
        <button @click="diagnoseError" :disabled="!hasError || polling">
          <Icon name="bug" />
          エラー診断
        </button>
      </div>
    </div>

    <footer class="copilot-footer">
      <CopilotInput
        v-model="input"
        @submit="handleSubmit"
        :loading="polling"
        :disabled="polling"
        placeholder="ワークフローを説明してください..."
      />
      <button v-if="polling" @click="cancel" class="cancel-btn">
        キャンセル
      </button>
    </footer>
  </aside>
</template>

<script setup lang="ts">
const { messages, polling, generate, cancel } = useCopilot()
const { workflow, lastError } = useWorkflowEditor()

const hasWorkflow = computed(() => workflow.value?.steps?.length > 0)
const hasError = computed(() => !!lastError.value)

async function handleSubmit() {
  if (!input.value.trim() || polling.value) return

  const prompt = input.value
  input.value = ''

  await generate(prompt)
}

function applyWorkflow(workflowData: WorkflowDefinition) {
  emit('apply-workflow', workflowData)
}
</script>
```

---

## 実装ステップ

### Phase 1: 基盤整備（2日）

| タスク | 工数 |
|--------|------|
| TriggerType に `internal` 追加 | 0.5日 |
| runs テーブル拡張（trigger_source, trigger_metadata） | 0.5日 |
| workflows テーブル拡張（is_system, system_slug） | 0.5日 |
| ExecuteSystemWorkflow メソッド実装 | 0.5日 |

### Phase 2: システムワークフロー（2.5日）

| タスク | 工数 |
|--------|------|
| ctx 拡張（blocks.list, workflows.get, runs.get） | 0.5日 |
| copilot-generate ワークフロー定義 | 1日 |
| copilot-diagnose, copilot-optimize, copilot-suggest 定義 | 1日 |

### Phase 3: API実装（1日）

| タスク | 工数 |
|--------|------|
| CopilotHandler 実装 | 0.5日 |
| ルーティング設定 | 0.5日 |

### Phase 4: フロントエンド（2.5日）

| タスク | 工数 |
|--------|------|
| useCopilot.ts（ポーリング対応） | 1日 |
| CopilotPanel, CopilotChat コンポーネント | 1日|
| ワークフローエディタへの統合 | 0.5日 |

### Phase 5: テスト・調整（2日）

| タスク | 工数 |
|--------|------|
| 単体テスト | 0.5日 |
| E2Eテスト | 0.5日 |
| プロンプト調整 | 1日 |

---

## 工数見積

| フェーズ | 工数 |
|---------|------|
| Phase 1: 基盤整備 | 2日 |
| Phase 2: システムワークフロー | 2.5日 |
| Phase 3: API実装 | 1日 |
| Phase 4: フロントエンド | 2.5日 |
| Phase 5: テスト・調整 | 2日 |
| **合計** | **10日** |

---

## テスト計画

### ユニットテスト

| テスト | 内容 |
|--------|------|
| ExecuteSystemWorkflow | システムWF実行、TriggerType/Source記録 |
| プロンプト生成 | テンプレート変数の正しい埋め込み |
| JSON解析 | LLM出力の解析 |
| ワークフロー検証 | 生成されたWFの構造検証 |

### E2Eテスト

1. POST /copilot/generate → run_id取得
2. GET /copilot/runs/{id} ポーリング → status: completed
3. output.workflow の構造検証

### 結果確認

```sql
-- Copilot実行履歴の確認
SELECT
    id, status, trigger_type, trigger_source,
    trigger_metadata->>'feature' as feature,
    created_at
FROM runs
WHERE trigger_type = 'internal'
  AND trigger_source = 'copilot'
ORDER BY created_at DESC;
```

---

## リスクと対策

| リスク | 対策 |
|--------|------|
| LLM出力が不正なJSON | リトライ + フォールバック |
| 生成されたWFが実行不可 | 構造検証ステップ |
| コストが高い | 軽量モデルのデフォルト使用 |
| ポーリング負荷 | 間隔調整、タイムアウト設定 |
| システムWFの循環呼び出し | ガードチェック実装 |

---

## 将来の拡張

| 機能 | 説明 |
|------|------|
| **ストリーミング応答** | SSE/WebSocketでリアルタイム表示 |
| **マルチターン会話** | コンテキストを保持した対話 |
| **テンプレートライブラリ** | よく使うパターンの保存・共有 |
| **学習機能** | ユーザーのフィードバックを反映 |
| **外部公開** | システムWFにWebhook作成で外部API化 |

---

## Related Documents

- [BACKEND.md](../BACKEND.md) - Backend architecture
- [API.md](../API.md) - API documentation
- [DATABASE.md](../DATABASE.md) - Database schema
- [UNIFIED_BLOCK_MODEL.md](../designs/UNIFIED_BLOCK_MODEL.md) - Block execution architecture
