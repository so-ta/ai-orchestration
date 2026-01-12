# Phase 10: Copilot 実装計画

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

### アーキテクチャ

```
┌─────────────────────────────────────────────────────────┐
│                    Frontend                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │  CopilotChat │  │ SuggestionUI │  │ CommandPalette│   │
│  └──────────────┘  └──────────────┘  └──────────────┘   │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│                    Backend API                           │
│  ┌──────────────────────────────────────────────────┐   │
│  │              CopilotHandler                       │   │
│  └──────────────────────────────────────────────────┘   │
│                           │                              │
│  ┌──────────────────────────────────────────────────┐   │
│  │              CopilotUsecase                       │   │
│  │  - Generate  - Suggest  - Diagnose  - Optimize    │   │
│  └──────────────────────────────────────────────────┘   │
│                           │                              │
│  ┌──────────────────────────────────────────────────┐   │
│  │            PromptBuilder + LLM Adapter            │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

### プロンプト設計

**ファイル**: `backend/internal/adapter/copilot_prompts.go`

```go
package adapter

const SystemPromptGenerate = `
You are an AI workflow builder assistant. Your task is to convert natural language descriptions into structured workflow definitions.

## Available Step Types
- llm: LLM API calls (OpenAI, Anthropic)
- tool: External tool/adapter execution
- condition: Conditional branching
- map: Parallel array processing
- loop: Iteration (for, forEach, while)
- wait: Delay/timer
- human_in_loop: Human approval gate
- router: AI-based dynamic routing
- guardrails: Content safety checks
- evaluator: Output quality evaluation

## Available Adapters
{{range .Adapters}}
- {{.ID}}: {{.Description}}
{{end}}

## Output Format
Return a JSON object with:
{
  "workflow": {
    "name": "Workflow Name",
    "description": "Description",
    "steps": [...],
    "edges": [...]
  },
  "explanation": "Step-by-step explanation of how this workflow works"
}

## Guidelines
1. Create minimal viable workflows - avoid unnecessary complexity
2. Use appropriate step types for each task
3. Include error handling where appropriate
4. Consider performance and cost implications
5. Add clear step names and descriptions
`

const SystemPromptSuggest = `
You are an AI assistant helping users build workflows. Based on the current workflow state, suggest the next step to add.

## Current Workflow
{{.WorkflowJSON}}

## Last Added Step
{{.LastStep}}

## Task
Suggest 2-3 possible next steps that would logically follow. Consider:
1. What typically comes next in this type of workflow
2. Error handling needs
3. Data transformation needs
4. Output/notification needs

Return JSON:
{
  "suggestions": [
    {
      "type": "step_type",
      "name": "Suggested Step Name",
      "description": "Why this step is recommended",
      "config": {...}
    }
  ]
}
`

const SystemPromptDiagnose = `
You are an AI debugging assistant. Analyze the workflow execution error and provide diagnosis.

## Workflow Definition
{{.WorkflowJSON}}

## Failed Step
{{.FailedStepJSON}}

## Error Information
{{.ErrorMessage}}
{{.StackTrace}}

## Recent Step Outputs
{{.RecentOutputs}}

## Task
1. Identify the root cause of the failure
2. Explain what went wrong in simple terms
3. Provide specific fix recommendations
4. Suggest preventive measures

Return JSON:
{
  "diagnosis": {
    "root_cause": "Clear explanation of what caused the error",
    "category": "config_error|input_error|api_error|logic_error|timeout|unknown",
    "severity": "high|medium|low"
  },
  "fixes": [
    {
      "description": "What to fix",
      "steps": ["Step 1", "Step 2"],
      "code_change": {...}  // Optional: actual config changes
    }
  ],
  "prevention": ["Tip 1", "Tip 2"]
}
`

const SystemPromptOptimize = `
You are an AI optimization assistant. Analyze the workflow and suggest improvements.

## Workflow Definition
{{.WorkflowJSON}}

## Usage Statistics
- Total runs: {{.TotalRuns}}
- Avg duration: {{.AvgDuration}}ms
- Avg cost: ${{.AvgCost}}
- Failure rate: {{.FailureRate}}%

## Task
Analyze and suggest optimizations for:
1. Performance (latency reduction)
2. Cost (cheaper models, fewer API calls)
3. Reliability (error handling, retries)
4. Maintainability (simplification, documentation)

Return JSON:
{
  "optimizations": [
    {
      "category": "performance|cost|reliability|maintainability",
      "title": "Optimization Title",
      "description": "Detailed explanation",
      "impact": "high|medium|low",
      "effort": "high|medium|low",
      "changes": {...}  // Optional: specific changes
    }
  ],
  "summary": "Overall assessment and top recommendations"
}
`
```

### API設計

| Method | Path | 説明 |
|--------|------|------|
| POST | `/api/v1/copilot/generate` | 自然言語からWF生成 |
| POST | `/api/v1/copilot/suggest` | 次ステップ提案 |
| POST | `/api/v1/copilot/diagnose` | エラー診断 |
| POST | `/api/v1/copilot/optimize` | 最適化提案 |
| POST | `/api/v1/copilot/explain` | WF説明生成 |
| POST | `/api/v1/copilot/chat` | 汎用チャット |

### Request/Response

**ワークフロー生成**:
```json
// POST /api/v1/copilot/generate
{
  "prompt": "顧客からの問い合わせメールを受信し、AIで分類して、緊急の場合はSlack通知、それ以外はチケット作成",
  "context": {
    "available_adapters": ["slack", "email", "jira"]
  }
}

// Response
{
  "workflow": {
    "name": "Customer Inquiry Handler",
    "description": "Receives customer emails, classifies them with AI, and routes urgent ones to Slack while creating tickets for others",
    "steps": [
      {
        "id": "step-1",
        "name": "Receive Email",
        "type": "tool",
        "config": {"adapter_id": "email", "action": "receive"}
      },
      {
        "id": "step-2",
        "name": "Classify with AI",
        "type": "llm",
        "config": {
          "provider": "openai",
          "model": "gpt-4o-mini",
          "prompt": "Classify the following customer inquiry as 'urgent' or 'normal':\n\n{{$.email.body}}"
        }
      },
      {
        "id": "step-3",
        "name": "Check Urgency",
        "type": "condition",
        "config": {"expression": "$.classification == 'urgent'"}
      },
      {
        "id": "step-4",
        "name": "Send Slack Alert",
        "type": "tool",
        "config": {"adapter_id": "slack", "channel": "#urgent"}
      },
      {
        "id": "step-5",
        "name": "Create Ticket",
        "type": "tool",
        "config": {"adapter_id": "jira", "action": "create_issue"}
      }
    ],
    "edges": [
      {"source": "step-1", "target": "step-2"},
      {"source": "step-2", "target": "step-3"},
      {"source": "step-3", "target": "step-4", "condition": "true"},
      {"source": "step-3", "target": "step-5", "condition": "false"}
    ]
  },
  "explanation": "このワークフローは以下の流れで動作します：\n1. メールアダプターで受信メールを取得\n2. GPT-4o-miniで問い合わせ内容を分析し、緊急度を判定\n3. 条件分岐で緊急/通常を振り分け\n4. 緊急の場合はSlackの#urgentチャンネルに通知\n5. 通常の場合はJiraにチケットを作成"
}
```

**エラー診断**:
```json
// POST /api/v1/copilot/diagnose
{
  "run_id": "uuid",
  "step_run_id": "uuid"
}

// Response
{
  "diagnosis": {
    "root_cause": "OpenAI APIのレート制限に達しました。短時間に多くのリクエストを送信したため、429エラーが発生しています。",
    "category": "api_error",
    "severity": "medium"
  },
  "fixes": [
    {
      "description": "リトライロジックを追加する",
      "steps": [
        "LLMステップの設定で max_retries を 3 に設定",
        "retry_delay_ms を 1000 に設定"
      ],
      "code_change": {
        "step_id": "step-2",
        "config_patch": {
          "max_retries": 3,
          "retry_delay_ms": 1000
        }
      }
    },
    {
      "description": "より安価なモデルに変更してレート制限を回避",
      "steps": [
        "gpt-4o から gpt-4o-mini に変更"
      ]
    }
  ],
  "prevention": [
    "大量実行前にバッチサイズを制限する",
    "mapステップで parallel: false を使用して順次実行する"
  ]
}
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
│       ├── SuggestionList.vue    # 提案リスト
│       ├── DiagnosisCard.vue     # 診断結果表示
│       └── OptimizationCard.vue  # 最適化提案表示
└── composables/
    └── useCopilot.ts             # Copilot API呼び出し
```

### CopilotPanel.vue

```vue
<template>
  <aside class="copilot-panel" :class="{ open: isOpen }">
    <header class="copilot-header">
      <h3>
        <Icon name="sparkles" />
        {{ $t('copilot.title') }}
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
          @apply="applyChange"
        />
      </div>

      <!-- クイックアクション -->
      <div class="quick-actions" v-if="messages.length === 0">
        <button @click="generateWorkflow">
          <Icon name="wand" />
          {{ $t('copilot.generateWorkflow') }}
        </button>
        <button @click="optimizeWorkflow" :disabled="!hasWorkflow">
          <Icon name="zap" />
          {{ $t('copilot.optimize') }}
        </button>
        <button @click="diagnoseError" :disabled="!hasError">
          <Icon name="bug" />
          {{ $t('copilot.diagnose') }}
        </button>
      </div>
    </div>

    <footer class="copilot-footer">
      <CopilotInput
        v-model="input"
        @submit="sendMessage"
        :loading="loading"
        :placeholder="$t('copilot.inputPlaceholder')"
      />
    </footer>
  </aside>
</template>

<script setup lang="ts">
const { messages, sendMessage, loading } = useCopilot()
const { workflow, selectedStep, lastError } = useWorkflowEditor()

const hasWorkflow = computed(() => workflow.value?.steps?.length > 0)
const hasError = computed(() => !!lastError.value)

async function generateWorkflow() {
  await sendMessage({
    type: 'generate',
    prompt: input.value
  })
}

async function applyChange(change: WorkflowChange) {
  // エディタにワークフロー変更を適用
  emit('apply-change', change)
}
</script>
```

### useCopilot.ts

```typescript
export function useCopilot() {
  const messages = ref<CopilotMessage[]>([])
  const loading = ref(false)
  const { $api } = useNuxtApp()

  async function generate(prompt: string): Promise<GenerateResponse> {
    loading.value = true
    try {
      const response = await $api.post('/copilot/generate', { prompt })
      messages.value.push({
        id: nanoid(),
        role: 'assistant',
        type: 'workflow',
        content: response.explanation,
        data: response.workflow
      })
      return response
    } finally {
      loading.value = false
    }
  }

  async function suggest(workflowId: string, lastStepId?: string): Promise<SuggestResponse> {
    loading.value = true
    try {
      const response = await $api.post('/copilot/suggest', {
        workflow_id: workflowId,
        last_step_id: lastStepId
      })
      return response
    } finally {
      loading.value = false
    }
  }

  async function diagnose(runId: string, stepRunId?: string): Promise<DiagnoseResponse> {
    loading.value = true
    try {
      const response = await $api.post('/copilot/diagnose', {
        run_id: runId,
        step_run_id: stepRunId
      })
      messages.value.push({
        id: nanoid(),
        role: 'assistant',
        type: 'diagnosis',
        content: response.diagnosis.root_cause,
        data: response
      })
      return response
    } finally {
      loading.value = false
    }
  }

  async function optimize(workflowId: string): Promise<OptimizeResponse> {
    loading.value = true
    try {
      const response = await $api.post('/copilot/optimize', {
        workflow_id: workflowId
      })
      messages.value.push({
        id: nanoid(),
        role: 'assistant',
        type: 'optimization',
        content: response.summary,
        data: response.optimizations
      })
      return response
    } finally {
      loading.value = false
    }
  }

  return {
    messages,
    loading,
    generate,
    suggest,
    diagnose,
    optimize
  }
}
```

---

## 実装ステップ

### Step 1: プロンプトテンプレート作成（1日）

**ファイル**: `backend/internal/adapter/copilot_prompts.go`

- Generate, Suggest, Diagnose, Optimize各プロンプト
- テンプレート変数の埋め込みロジック

### Step 2: Usecase実装（2日）

**ファイル**: `backend/internal/usecase/copilot.go`

```go
type CopilotUsecase struct {
    llmAdapter   adapter.LLMAdapter
    workflowRepo repository.WorkflowRepository
    runRepo      repository.RunRepository
    usageRepo    repository.UsageRepository
}

func (u *CopilotUsecase) Generate(ctx context.Context, input GenerateInput) (*GenerateOutput, error) {
    // 1. 利用可能なアダプター一覧を取得
    // 2. プロンプト構築
    // 3. LLM呼び出し
    // 4. JSON解析・検証
    // 5. ワークフロー構造を返却
}

func (u *CopilotUsecase) Diagnose(ctx context.Context, input DiagnoseInput) (*DiagnoseOutput, error) {
    // 1. Run, StepRun情報取得
    // 2. ワークフロー定義取得
    // 3. エラー情報収集
    // 4. プロンプト構築
    // 5. LLM呼び出し
    // 6. 診断結果を返却
}
```

### Step 3: Handler実装（0.5日）

**ファイル**: `backend/internal/handler/copilot.go`

### Step 4: フロントエンド - 基本UI（2日）

**ファイル**:
- `frontend/components/copilot/CopilotPanel.vue`
- `frontend/components/copilot/CopilotChat.vue`
- `frontend/composables/useCopilot.ts`

### Step 5: フロントエンド - 機能統合（2日）

- ワークフローエディタへの統合
- 提案の適用機能
- 診断結果からの修正適用

### Step 6: テスト・調整（2日）

- プロンプトの調整
- エッジケースのハンドリング
- UXの改善

---

## テスト計画

### ユニットテスト

| テスト | 内容 |
|--------|------|
| プロンプト生成 | テンプレート変数の正しい埋め込み |
| JSON解析 | LLM出力の解析 |
| ワークフロー検証 | 生成されたWFの構造検証 |

### E2Eテスト

1. 自然言語からWF生成 → 有効なWF構造
2. エラー診断 → 適切な診断結果
3. 最適化提案 → 実行可能な提案

### プロンプト評価

- 生成されたWFの妥当性
- 診断の正確性
- 提案の実用性

---

## リスクと対策

| リスク | 対策 |
|--------|------|
| LLM出力が不正なJSON | リトライ + フォールバック |
| 生成されたWFが実行不可 | 構造検証 + ユーザー確認 |
| コストが高い | 軽量モデルのデフォルト使用 |
| レスポンス遅延 | ストリーミング対応 |

---

## 将来の拡張

- **ストリーミング応答**: リアルタイムで生成結果を表示
- **マルチターン会話**: コンテキストを保持した対話
- **テンプレートライブラリ**: よく使うパターンの保存・共有
- **学習機能**: ユーザーのフィードバックを反映

---

## 工数見積

| タスク | 工数 |
|--------|------|
| プロンプトテンプレート | 1日 |
| Usecase実装 | 2日 |
| Handler実装 | 0.5日 |
| フロントエンド基本UI | 2日 |
| フロントエンド統合 | 2日 |
| テスト・調整 | 2日 |
| ドキュメント | 0.5日 |
| **合計** | **10日** |
