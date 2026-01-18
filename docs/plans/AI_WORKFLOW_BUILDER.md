# AI ワークフロービルダー 実装計画

> **Status**: 📋 計画中
> **Created**: 2026-01-18
> **Author**: AI Agent
> **Depends on**: [PHASE10_COPILOT.md](./PHASE10_COPILOT.md) - 既存 Copilot 機能を拡張

---

## 概要

「〇〇のワークフローを作って」というユーザーの自然言語入力から、AI が対話的にヒアリングを行い、ワークフローを自律的に構築するシステム。

### 設計原則

```
┌─────────────────────────────────────────────────────────────────────────┐
│                  AI ワークフロービルダー アーキテクチャ                     │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ユーザー: 「営業レポートを自動化して」                                     │
│       │                                                                  │
│       ▼                                                                  │
│  ┌─────────────────────────────────────────────────────────────────────┐│
│  │                    Phase 1: ヒアリング                               ││
│  │                                                                       ││
│  │  AI: 「営業レポートの自動化についてお聞きします。」                     ││
│  │      「レポートの最終形式は何ですか？（PDF/Excel/Slack通知）」          ││
│  │      「誰かの承認は必要ですか？」                                       ││
│  │      「どのくらいの頻度で実行しますか？」                               ││
│  │                                                                       ││
│  │  → 内部DSL（WorkflowSpec）として正規化                                ││
│  └─────────────────────────────────────────────────────────────────────┘│
│       │                                                                  │
│       ▼                                                                  │
│  ┌─────────────────────────────────────────────────────────────────────┐│
│  │                    Phase 2: ワークフロー構築                          ││
│  │                                                                       ││
│  │  WorkflowSpec → Steps + Edges + BlockGroups                          ││
│  │                                                                       ││
│  │  - プリセットブロックへのマッピング（最優先）                          ││
│  │  - カスタムブロック候補の検出                                          ││
│  │  - データフロー・スキーマの自動生成                                     ││
│  │  - 条件分岐・例外処理の組み込み                                        ││
│  └─────────────────────────────────────────────────────────────────────┘│
│       │                                                                  │
│       ▼                                                                  │
│  ┌─────────────────────────────────────────────────────────────────────┐│
│  │                    Phase 3: ユーザー提示・ブラッシュアップ             ││
│  │                                                                       ││
│  │  AI: 「こちらのワークフローを作成しました：」                           ││
│  │      「[ワークフロー図を表示]」                                         ││
│  │      「⚠️ Google Sheets 連携はカスタム作成が必要です」                  ││
│  │                                                                       ││
│  │  ユーザー: 「承認ステップを追加して」                                   ││
│  │  AI: 「承認ステップを追加しました。承認者は誰ですか？」                 ││
│  └─────────────────────────────────────────────────────────────────────┘│
│       │                                                                  │
│       ▼                                                                  │
│  ┌─────────────────────────────────────────────────────────────────────┐│
│  │                    Phase 4: 確定・保存                                ││
│  │                                                                       ││
│  │  - Project として保存（draft 状態）                                    ││
│  │  - バージョン履歴として記録                                            ││
│  │  - カスタム作成タスクの生成                                            ││
│  └─────────────────────────────────────────────────────────────────────┘│
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 既存システムとの統合

### 依存コンポーネント

| コンポーネント | 用途 | 現状 |
|---------------|------|------|
| Copilot API | 基盤 API エンドポイント | ✅ 実装済み |
| CopilotSession | セッション管理 | ✅ 実装済み |
| Block Registry | 利用可能ブロック一覧 | ✅ 46 ブロック |
| Workflow Engine | ワークフロー実行 | ✅ 実装済み |
| Project/Step/Edge | データモデル | ✅ 実装済み |

### 拡張ポイント

| 拡張 | 説明 |
|------|------|
| 新規システムワークフロー | `ai-builder-*` ワークフロー群 |
| WorkflowSpec DSL | ヒアリング結果の内部表現 |
| ビルダーエンジン | DSL → Project 変換 |
| 対話型 UI | 既存 CopilotPanel の拡張 |

---

## データモデル

### WorkflowSpec（内部 DSL）

ヒアリング結果を機械的に扱える形式に正規化したもの。

```typescript
interface WorkflowSpec {
  // === メタ情報 ===
  id: string                        // セッション固有ID
  name: string                      // ワークフロー名
  description: string               // 説明

  // === ビジネス要件 ===
  purpose: string                   // 目的・ゴール
  successCriteria: string[]         // 成功条件
  businessDomain: BusinessDomain    // 業務カテゴリ

  // === 開始・終了 ===
  trigger: TriggerSpec              // 開始条件
  completion: CompletionSpec        // 完了条件

  // === 関与者 ===
  actors: ActorSpec[]               // 関与する人物・役割

  // === ステップ ===
  steps: StepSpec[]                 // 処理ステップ

  // === 外部連携 ===
  integrations: IntegrationSpec[]   // 利用ツール・API

  // === 制約 ===
  constraints: ConstraintSpec       // 制約条件

  // === 仮定 ===
  assumptions: Assumption[]         // AI が仮定した前提

  // === メタデータ ===
  heardAt: Date                     // ヒアリング完了日時
  version: number                   // スペックバージョン
}

interface TriggerSpec {
  type: 'manual' | 'schedule' | 'webhook' | 'event'
  schedule?: string                 // cron 式
  eventSource?: string              // イベントソース
  description: string               // 自然言語の説明
}

interface CompletionSpec {
  description: string               // 完了条件の説明
  outputs: OutputSpec[]             // 成果物
}

interface OutputSpec {
  name: string
  type: 'document' | 'notification' | 'data' | 'approval' | 'other'
  format?: string                   // PDF, Excel, JSON, etc.
  destination?: string              // 送信先
}

interface ActorSpec {
  role: 'executor' | 'approver' | 'reviewer' | 'viewer'
  description: string               // 役割の説明
  count: 'single' | 'multiple' | 'optional'
}

interface StepSpec {
  id: string
  name: string
  description: string
  type: StepSpecType
  inputs: string[]                  // 入力データの参照
  outputs: string[]                 // 出力データのキー
  config?: Record<string, unknown>  // ブロック固有設定

  // === マッピング結果 ===
  mappedBlock?: {
    slug: string
    confidence: 'high' | 'medium' | 'low'
    customRequired: boolean
    customReason?: string
  }

  // === 分岐・例外 ===
  branches?: BranchSpec[]
  errorHandling?: ErrorHandlingSpec
}

type StepSpecType =
  | 'input'           // データ入力
  | 'transform'       // データ変換
  | 'decision'        // 条件分岐
  | 'action'          // アクション実行
  | 'notification'    // 通知
  | 'approval'        // 承認
  | 'integration'     // 外部連携
  | 'ai'              // AI 処理
  | 'loop'            // ループ処理
  | 'aggregate'       // 集計
  | 'wait'            // 待機

interface BranchSpec {
  condition: string                 // 条件の説明
  targetStepId: string
}

interface ErrorHandlingSpec {
  onError: 'retry' | 'skip' | 'abort' | 'notify' | 'fallback'
  retryCount?: number
  retryDelay?: number               // ms
  fallbackStepId?: string
  notifyTo?: string
}

interface IntegrationSpec {
  service: string                   // Slack, GitHub, etc.
  operation: string                 // send_message, create_issue
  hasCredentials: boolean           // 認証情報の有無
  requiredSecrets: string[]         // 必要なシークレットキー
}

interface ConstraintSpec {
  frequency?: 'once' | 'daily' | 'weekly' | 'monthly' | 'on-demand'
  deadline?: string                 // 期限
  sla?: string                      // SLA
  security?: string[]               // セキュリティ要件
}

interface Assumption {
  id: string
  category: 'trigger' | 'actor' | 'step' | 'integration' | 'constraint'
  description: string               // 仮定の内容
  default: string                   // デフォルト値
  confirmed: boolean                // ユーザー確認済みか
}

type BusinessDomain =
  | 'sales'           // 営業
  | 'development'     // 開発
  | 'hr'              // 人事・採用
  | 'finance'         // 経理・財務
  | 'marketing'       // マーケティング
  | 'support'         // サポート
  | 'operations'      // オペレーション
  | 'personal'        // 個人タスク
  | 'other'           // その他
```

### BuilderSession（ビルダーセッション）

```typescript
interface BuilderSession {
  id: string
  tenantId: string
  userId: string
  status: BuilderSessionStatus

  // === ヒアリング状態 ===
  hearingPhase: HearingPhase
  hearingProgress: number           // 0-100

  // === 生成物 ===
  spec?: WorkflowSpec               // ヒアリング結果
  projectId?: string                // 生成された Project ID

  // === 会話履歴 ===
  messages: BuilderMessage[]

  // === メタデータ ===
  createdAt: Date
  updatedAt: Date
}

type BuilderSessionStatus =
  | 'hearing'         // ヒアリング中
  | 'building'        // ワークフロー構築中
  | 'reviewing'       // ユーザーレビュー中
  | 'refining'        // ブラッシュアップ中
  | 'completed'       // 完了
  | 'abandoned'       // 放棄

type HearingPhase =
  | 'purpose'         // 1.1-1.2 目的・ゴール確認
  | 'conditions'      // 1.3 開始・終了条件
  | 'actors'          // 1.4-1.5 関与者・承認
  | 'frequency'       // 1.6 実行頻度・期限
  | 'integrations'    // 1.7 ツール・システム
  | 'pain_points'     // 1.8 課題・困りごと
  | 'confirmation'    // 1.9 仮定条件の確認
  | 'completed'       // ヒアリング完了

interface BuilderMessage {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  timestamp: Date

  // === メタデータ ===
  phase?: HearingPhase
  extractedData?: Partial<WorkflowSpec>
  suggestedQuestions?: string[]
}
```

### データベーススキーマ

```sql
-- builder_sessions テーブル
CREATE TABLE builder_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    user_id UUID NOT NULL,
    copilot_session_id UUID REFERENCES copilot_sessions(id),

    -- 状態
    status VARCHAR(50) NOT NULL DEFAULT 'hearing',
    hearing_phase VARCHAR(50) NOT NULL DEFAULT 'purpose',
    hearing_progress INTEGER NOT NULL DEFAULT 0,

    -- 生成物
    spec JSONB,
    project_id UUID REFERENCES projects(id),

    -- タイムスタンプ
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_status CHECK (status IN (
        'hearing', 'building', 'reviewing', 'refining', 'completed', 'abandoned'
    )),
    CONSTRAINT valid_phase CHECK (hearing_phase IN (
        'purpose', 'conditions', 'actors', 'frequency',
        'integrations', 'pain_points', 'confirmation', 'completed'
    ))
);

-- builder_messages テーブル
CREATE TABLE builder_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES builder_sessions(id) ON DELETE CASCADE,

    role VARCHAR(20) NOT NULL,
    content TEXT NOT NULL,

    -- メタデータ
    phase VARCHAR(50),
    extracted_data JSONB,
    suggested_questions JSONB,

    -- タイムスタンプ
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_role CHECK (role IN ('user', 'assistant', 'system'))
);

-- インデックス
CREATE INDEX idx_builder_sessions_tenant ON builder_sessions(tenant_id);
CREATE INDEX idx_builder_sessions_user ON builder_sessions(user_id);
CREATE INDEX idx_builder_sessions_status ON builder_sessions(status);
CREATE INDEX idx_builder_messages_session ON builder_messages(session_id);
```

---

## システムワークフロー

### ai-builder-hearing（ヒアリングワークフロー）

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    ai-builder-hearing                                    │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  [Start]                                                                 │
│     │                                                                    │
│     ▼                                                                    │
│  [Get Session Context]                                                   │
│     │ session_id, current_phase, previous_messages, partial_spec        │
│     │                                                                    │
│     ▼                                                                    │
│  [Get Available Blocks]                                                  │
│     │ blocks.list() → ブロック一覧（マッピング用）                        │
│     │                                                                    │
│     ▼                                                                    │
│  [Build Hearing Prompt]                                                  │
│     │ フェーズに応じたプロンプト生成                                      │
│     │                                                                    │
│     ▼                                                                    │
│  [LLM Call]                                                              │
│     │ provider: anthropic, model: claude-sonnet-4-20250514               │
│     │ structured output: { response, extractedData, nextPhase, ... }    │
│     │                                                                    │
│     ▼                                                                    │
│  [Parse & Validate]                                                      │
│     │ JSON 解析、WorkflowSpec への部分マージ                             │
│     │                                                                    │
│     ▼                                                                    │
│  [Update Session]                                                        │
│     │ phase, progress, spec の更新                                       │
│     │                                                                    │
│     ▼                                                                    │
│  [Return Response]                                                       │
│     │ { message, suggestedQuestions, phase, progress, complete }        │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### ai-builder-construct（ワークフロー構築ワークフロー）

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    ai-builder-construct                                  │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  [Start]                                                                 │
│     │ input: { session_id, spec }                                       │
│     │                                                                    │
│     ▼                                                                    │
│  [Map Steps to Blocks] ─────────────────────────────────────────┐       │
│     │ 各 StepSpec をプリセットブロックにマッピング              │       │
│     │                                                            │       │
│     ▼                                                            │       │
│  [foreach: spec.steps]                                           │       │
│     │                                                            │       │
│     ├─► [Find Best Match]                                        │       │
│     │      │ ブロックレジストリから最適なブロックを検索          │       │
│     │      │ LLM によるセマンティックマッチング                  │       │
│     │      │                                                      │       │
│     │      ▼                                                      │       │
│     │   [Check Capability]                                       │       │
│     │      │ プリセットで実現可能か判定                          │       │
│     │      │ カスタム必要な場合は理由を記録                      │       │
│     │      │                                                      │       │
│     │      ▼                                                      │       │
│     │   [Generate Config]                                        │       │
│     │      │ ブロック設定の生成                                  │       │
│     │      │                                                      │       │
│     └◄─────┘                                                      │       │
│                                                                   │       │
│     ▼  ◄─────────────────────────────────────────────────────────┘       │
│  [Build Data Flow]                                                       │
│     │ ステップ間のデータフロー定義                                       │
│     │ スキーマ推論                                                       │
│     │                                                                    │
│     ▼                                                                    │
│  [Add Control Flow]                                                      │
│     │ 分岐・例外処理・ループの追加                                       │
│     │ BlockGroup の生成                                                  │
│     │                                                                    │
│     ▼                                                                    │
│  [Generate Edges]                                                        │
│     │ ステップ間の接続（Edge）生成                                       │
│     │                                                                    │
│     ▼                                                                    │
│  [Create Project]                                                        │
│     │ Project, Steps, Edges, BlockGroups として保存                      │
│     │ status: draft                                                      │
│     │                                                                    │
│     ▼                                                                    │
│  [Generate Summary]                                                      │
│     │ ユーザー向け説明の生成                                             │
│     │ カスタム作成必要箇所のリスト                                       │
│     │                                                                    │
│     ▼                                                                    │
│  [Return Result]                                                         │
│     │ { projectId, summary, customRequirements, warnings }              │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### ai-builder-refine（ブラッシュアップワークフロー）

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    ai-builder-refine                                     │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  [Start]                                                                 │
│     │ input: { session_id, project_id, user_feedback }                  │
│     │                                                                    │
│     ▼                                                                    │
│  [Get Current Project]                                                   │
│     │ 現在のワークフロー定義を取得                                       │
│     │                                                                    │
│     ▼                                                                    │
│  [Parse Feedback]                                                        │
│     │ ユーザーフィードバックを解析                                       │
│     │ 差分編集の内容を特定                                               │
│     │                                                                    │
│     ▼                                                                    │
│  [switch: feedback_type] ────────────────────────┐                       │
│     │                                            │                       │
│     ├─► add_step     → [Add Step]                │                       │
│     ├─► remove_step  → [Remove Step]             │                       │
│     ├─► modify_step  → [Modify Step]             │                       │
│     ├─► add_approval → [Add Approval Step]       │                       │
│     ├─► change_flow  → [Reorganize Flow]         │                       │
│     └─► other        → [LLM Interpret]           │                       │
│                                            │     │                       │
│     ▼  ◄─────────────────────────────────────────┘                       │
│  [Update Project]                                                        │
│     │ 変更を Project に適用                                              │
│     │ バージョンインクリメント                                           │
│     │                                                                    │
│     ▼                                                                    │
│  [Validate Changes]                                                      │
│     │ 変更後のワークフローを検証                                         │
│     │                                                                    │
│     ▼                                                                    │
│  [Generate Diff Summary]                                                 │
│     │ 変更内容のサマリー生成                                             │
│     │                                                                    │
│     ▼                                                                    │
│  [Return Result]                                                         │
│     │ { changes, newVersion, summary }                                  │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## API 設計

### エンドポイント一覧

| Method | Path | 説明 |
|--------|------|------|
| POST | `/api/v1/builder/sessions` | ビルダーセッション開始 |
| GET | `/api/v1/builder/sessions/{id}` | セッション取得 |
| POST | `/api/v1/builder/sessions/{id}/messages` | メッセージ送信（ヒアリング進行） |
| POST | `/api/v1/builder/sessions/{id}/construct` | ワークフロー構築開始 |
| POST | `/api/v1/builder/sessions/{id}/refine` | ブラッシュアップ |
| POST | `/api/v1/builder/sessions/{id}/finalize` | 確定・保存 |
| DELETE | `/api/v1/builder/sessions/{id}` | セッション破棄 |

### Request/Response

#### セッション開始

```json
// POST /api/v1/builder/sessions
{
  "initial_prompt": "営業レポートを自動化するワークフローを作りたい"
}

// Response
{
  "session_id": "uuid",
  "status": "hearing",
  "phase": "purpose",
  "progress": 10,
  "message": {
    "id": "msg-uuid",
    "role": "assistant",
    "content": "営業レポートの自動化をお手伝いします。\n\nまず、このワークフローの目的を確認させてください。\n最終的にどのような状態になれば成功と言えますか？\n\n例：\n- レポートがPDFで生成される\n- 上司に自動送信される\n- Slackで通知される",
    "suggested_questions": [
      "レポートの最終形式は何ですか？",
      "レポートは誰に届きますか？"
    ]
  }
}
```

#### メッセージ送信

```json
// POST /api/v1/builder/sessions/{id}/messages
{
  "content": "毎週月曜日にPDFで上司に送信したい"
}

// Response (polling 用の run_id を返却)
{
  "run_id": "uuid",
  "status": "pending"
}

// GET /api/v1/copilot/runs/{run_id} (ポーリング)
// Response (完了時)
{
  "run_id": "uuid",
  "status": "completed",
  "output": {
    "message": {
      "id": "msg-uuid",
      "role": "assistant",
      "content": "承知しました。毎週月曜日にPDFを生成して上司に送信するのですね。\n\n次に、このワークフローを開始するトリガーを確認させてください。\n月曜日の何時に実行しますか？また、データのソースは何ですか？",
      "extracted_data": {
        "trigger": {
          "type": "schedule",
          "schedule": "0 9 * * 1",
          "description": "毎週月曜日"
        },
        "completion": {
          "outputs": [
            { "name": "週次レポート", "type": "document", "format": "PDF" }
          ]
        }
      }
    },
    "session": {
      "phase": "conditions",
      "progress": 25
    }
  }
}
```

#### ワークフロー構築

```json
// POST /api/v1/builder/sessions/{id}/construct
// (spec は自動的にセッションから取得)

// Response
{
  "run_id": "uuid",
  "status": "pending"
}

// GET /api/v1/copilot/runs/{run_id} (ポーリング)
// Response (完了時)
{
  "run_id": "uuid",
  "status": "completed",
  "output": {
    "project_id": "proj-uuid",
    "summary": {
      "name": "週次営業レポート自動化",
      "description": "毎週月曜日に営業データを集計し、PDFレポートを生成して上司に送信します。",
      "steps_count": 5,
      "has_approval": false,
      "trigger": "schedule (毎週月曜 9:00)"
    },
    "step_mappings": [
      { "name": "データ取得", "block": "http", "confidence": "high" },
      { "name": "データ変換", "block": "function", "confidence": "high" },
      { "name": "PDF生成", "block": null, "custom_required": true, "reason": "PDF生成ブロックが未実装" },
      { "name": "メール送信", "block": "email_sendgrid", "confidence": "high" }
    ],
    "custom_requirements": [
      {
        "name": "PDF生成ブロック",
        "description": "HTMLからPDFを生成するカスタムブロック",
        "inputs": { "html": "string" },
        "outputs": { "pdf_url": "string" },
        "estimated_effort": "medium"
      }
    ],
    "warnings": [
      "SENDGRID_API_KEY シークレットの設定が必要です"
    ]
  }
}
```

---

## ヒアリングフロー詳細

### Phase 1.1-1.2: 目的・ゴール確認

```typescript
const purposePhasePrompt = `
あなたはワークフロービルダーAIです。
ユーザーが「${userInput}」というワークフローを作りたいと言っています。

以下の観点でヒアリングを行ってください：

1. **依頼内容の特定**
   - 「${userInput}」が指す具体的な業務・作業を確認
   - 抽象的な場合は、業務カテゴリを提示して選択させる

2. **ゴールの確認**
   - このワークフローで最終的に何が達成されれば成功か
   - 成果物があれば、その形式や状態を確認

【出力形式】
{
  "response": "ユーザーへの返答メッセージ",
  "extractedData": {
    "purpose": "推測された目的",
    "businessDomain": "sales|development|hr|...",
    "successCriteria": ["成功条件1", ...]
  },
  "suggestedQuestions": ["次に聞くべき質問1", ...],
  "nextPhase": "purpose|conditions",
  "confidence": "high|medium|low"
}
`;
```

### Phase 1.3: 開始・終了条件

```typescript
const conditionsPhasePrompt = `
現在のワークフロー仕様：
${JSON.stringify(currentSpec, null, 2)}

以下を確認してください：

1. **開始条件**
   - いつ・どのタイミングでこのフローが始まるか
   - トリガーの種類（手動、定期実行、Webhook、イベント）

2. **終了条件**
   - どの状態になったら完了とみなすか
   - 明確でなければ一般的な条件を仮置き

【出力形式】
{
  "response": "...",
  "extractedData": {
    "trigger": { "type": "...", "schedule": "...", "description": "..." },
    "completion": { "description": "...", "outputs": [...] }
  },
  ...
}
`;
```

### Phase 1.4-1.5: 関与者・承認

```typescript
const actorsPhasePrompt = `
現在のワークフロー仕様：
${JSON.stringify(currentSpec, null, 2)}

以下を確認してください：

1. **作業者**
   - 作業を実行する人（担当者）
   - 複数人いる場合は役割単位で整理

2. **承認・レビュー**
   - 承認やレビューが必要か
   - 必要な場合：承認者、タイミング、差し戻し時の扱い

【出力形式】
{
  "response": "...",
  "extractedData": {
    "actors": [
      { "role": "executor|approver|reviewer", "description": "...", "count": "single|multiple" }
    ]
  },
  ...
}
`;
```

### Phase 1.6-1.9: 頻度・ツール・課題・確認

同様のパターンで各フェーズのプロンプトを定義。

---

## ブロックマッピングロジック

### マッピングアルゴリズム

```typescript
interface BlockMappingResult {
  blockSlug: string | null
  confidence: 'high' | 'medium' | 'low'
  customRequired: boolean
  customReason?: string
  suggestedConfig?: Record<string, unknown>
}

async function mapStepToBlock(
  step: StepSpec,
  availableBlocks: BlockDefinition[],
  llmService: LLMService
): Promise<BlockMappingResult> {
  // 1. ルールベースの直接マッチング
  const directMatch = findDirectMatch(step, availableBlocks)
  if (directMatch) {
    return {
      blockSlug: directMatch.slug,
      confidence: 'high',
      customRequired: false
    }
  }

  // 2. セマンティックマッチング（LLM使用）
  const semanticMatch = await findSemanticMatch(step, availableBlocks, llmService)
  if (semanticMatch.confidence !== 'low') {
    return semanticMatch
  }

  // 3. カスタムブロック候補として報告
  return {
    blockSlug: null,
    confidence: 'low',
    customRequired: true,
    customReason: analyzeCustomRequirement(step)
  }
}

function findDirectMatch(step: StepSpec, blocks: BlockDefinition[]): BlockDefinition | null {
  const typeToBlockMap: Record<StepSpecType, string[]> = {
    'input': ['start'],
    'transform': ['function', 'map', 'filter'],
    'decision': ['condition', 'switch', 'router'],
    'action': ['function', 'http'],
    'notification': ['slack', 'discord', 'email_sendgrid'],
    'approval': ['human_in_loop'],
    'integration': ['http', 'github_create_issue', 'notion_create_page', ...],
    'ai': ['llm', 'router', 'rag-query'],
    'loop': ['foreach', 'while'],  // BlockGroup
    'aggregate': ['aggregate'],
    'wait': ['wait']
  }

  const candidates = typeToBlockMap[step.type] || []
  for (const slug of candidates) {
    const block = blocks.find(b => b.slug === slug)
    if (block && matchesCapabilities(step, block)) {
      return block
    }
  }
  return null
}

async function findSemanticMatch(
  step: StepSpec,
  blocks: BlockDefinition[],
  llmService: LLMService
): Promise<BlockMappingResult> {
  const prompt = `
以下のステップ仕様に最も適したブロックを選んでください。

【ステップ仕様】
名前: ${step.name}
説明: ${step.description}
種類: ${step.type}

【利用可能なブロック】
${blocks.map(b => `- ${b.slug}: ${b.description}`).join('\n')}

【出力形式】
{
  "blockSlug": "最適なブロックのslug、なければnull",
  "confidence": "high|medium|low",
  "reason": "選択理由",
  "suggestedConfig": { 設定の提案 }
}
`

  const response = await llmService.chat({
    messages: [{ role: 'user', content: prompt }],
    responseFormat: 'json'
  })

  return JSON.parse(response.content)
}
```

### カスタムブロック要件の分析

```typescript
function analyzeCustomRequirement(step: StepSpec): string {
  const reasons: string[] = []

  // 未対応の外部API
  if (step.type === 'integration' && !isKnownIntegration(step)) {
    reasons.push(`未対応の外部サービス連携: ${step.description}`)
  }

  // 複雑な変換
  if (step.type === 'transform' && hasComplexTransformation(step)) {
    reasons.push('複雑なデータ変換ロジックが必要')
  }

  // ドメイン固有の計算
  if (requiresDomainSpecificLogic(step)) {
    reasons.push('ドメイン固有の計算・ロジックが必要')
  }

  return reasons.join('; ') || '汎用ブロックでは表現できない処理'
}
```

---

## フロントエンド設計

### コンポーネント構成

```
frontend/
├── components/
│   └── builder/
│       ├── BuilderPanel.vue           # メインパネル
│       ├── BuilderChat.vue            # チャットUI
│       ├── BuilderMessage.vue         # メッセージ表示
│       ├── BuilderProgress.vue        # ヒアリング進捗
│       ├── BuilderAssumptions.vue     # 仮定条件表示
│       ├── WorkflowPreview.vue        # ワークフロープレビュー
│       ├── StepMappingList.vue        # ブロックマッピング結果
│       ├── CustomRequirements.vue     # カスタム要件リスト
│       └── BuilderActions.vue         # アクションボタン
└── composables/
    └── useBuilder.ts                  # ビルダーAPI呼び出し
```

### useBuilder.ts

```typescript
export function useBuilder() {
  const session = ref<BuilderSession | null>(null)
  const messages = ref<BuilderMessage[]>([])
  const isLoading = ref(false)
  const currentRunId = ref<string | null>(null)
  const { $api } = useNuxtApp()

  async function startSession(initialPrompt: string): Promise<void> {
    isLoading.value = true
    try {
      const response = await $api.post('/builder/sessions', {
        initial_prompt: initialPrompt
      })
      session.value = response
      messages.value = [response.message]
    } finally {
      isLoading.value = false
    }
  }

  async function sendMessage(content: string): Promise<void> {
    if (!session.value) return

    // ユーザーメッセージを即座に追加
    const userMessage: BuilderMessage = {
      id: nanoid(),
      role: 'user',
      content,
      timestamp: new Date()
    }
    messages.value.push(userMessage)

    // ローディングメッセージ
    const loadingMessage: BuilderMessage = {
      id: nanoid(),
      role: 'assistant',
      content: '考え中...',
      timestamp: new Date()
    }
    messages.value.push(loadingMessage)

    // APIコール
    const { run_id } = await $api.post(
      `/builder/sessions/${session.value.id}/messages`,
      { content }
    )
    currentRunId.value = run_id

    // ポーリング
    await pollForResult(loadingMessage.id)
  }

  async function pollForResult(loadingMessageId: string): Promise<void> {
    while (currentRunId.value) {
      await sleep(1000)

      const run = await $api.get(`/copilot/runs/${currentRunId.value}`)

      if (run.status === 'completed') {
        // ローディングメッセージを結果で置換
        const idx = messages.value.findIndex(m => m.id === loadingMessageId)
        if (idx >= 0) {
          messages.value[idx] = run.output.message
        }

        // セッション状態更新
        if (run.output.session) {
          session.value = {
            ...session.value!,
            ...run.output.session
          }
        }

        currentRunId.value = null
        return
      }

      if (run.status === 'failed') {
        const idx = messages.value.findIndex(m => m.id === loadingMessageId)
        if (idx >= 0) {
          messages.value[idx] = {
            id: loadingMessageId,
            role: 'assistant',
            content: `エラーが発生しました: ${run.error}`,
            timestamp: new Date()
          }
        }
        currentRunId.value = null
        return
      }
    }
  }

  async function constructWorkflow(): Promise<ConstructionResult | null> {
    if (!session.value) return null

    const { run_id } = await $api.post(
      `/builder/sessions/${session.value.id}/construct`
    )
    currentRunId.value = run_id

    // ポーリングで結果を待つ
    while (currentRunId.value) {
      await sleep(1000)
      const run = await $api.get(`/copilot/runs/${currentRunId.value}`)

      if (run.status === 'completed') {
        currentRunId.value = null
        return run.output
      }

      if (run.status === 'failed') {
        currentRunId.value = null
        throw new Error(run.error)
      }
    }

    return null
  }

  async function refineWorkflow(feedback: string): Promise<RefineResult | null> {
    if (!session.value) return null

    const { run_id } = await $api.post(
      `/builder/sessions/${session.value.id}/refine`,
      { feedback }
    )

    // 同様にポーリング
    // ...
  }

  async function finalizeWorkflow(): Promise<void> {
    if (!session.value) return

    await $api.post(`/builder/sessions/${session.value.id}/finalize`)
    session.value.status = 'completed'
  }

  function cancel(): void {
    currentRunId.value = null
  }

  return {
    session,
    messages,
    isLoading,
    startSession,
    sendMessage,
    constructWorkflow,
    refineWorkflow,
    finalizeWorkflow,
    cancel
  }
}
```

### BuilderPanel.vue

```vue
<template>
  <aside class="builder-panel" :class="{ open: isOpen }">
    <header class="builder-header">
      <h3>
        <Icon name="wand-2" />
        ワークフロービルダー
      </h3>
      <button @click="close">
        <Icon name="x" />
      </button>
    </header>

    <!-- ヒアリング進捗 -->
    <BuilderProgress
      v-if="session?.status === 'hearing'"
      :phase="session.hearingPhase"
      :progress="session.hearingProgress"
    />

    <div class="builder-content">
      <!-- 初期画面 -->
      <div v-if="!session" class="builder-start">
        <p>ワークフローを自然言語で説明してください。AIがヒアリングを行い、最適なワークフローを構築します。</p>
        <textarea
          v-model="initialPrompt"
          placeholder="例: 毎週月曜日に営業データをまとめてレポートを作成し、上司にメールで送信したい"
          rows="4"
        />
        <button @click="start" :disabled="!initialPrompt.trim()">
          ビルダーを開始
        </button>
      </div>

      <!-- チャット -->
      <div v-else class="builder-chat">
        <div class="messages" ref="messagesRef">
          <BuilderMessage
            v-for="msg in messages"
            :key="msg.id"
            :message="msg"
          />
        </div>

        <!-- 仮定条件の確認 -->
        <BuilderAssumptions
          v-if="session.status === 'hearing' && session.hearingPhase === 'confirmation'"
          :assumptions="session.spec?.assumptions || []"
          @confirm="handleAssumptionConfirm"
        />

        <!-- ワークフロープレビュー -->
        <WorkflowPreview
          v-if="session.status === 'reviewing'"
          :project-id="session.projectId"
          :summary="constructionResult?.summary"
        />

        <!-- カスタム要件 -->
        <CustomRequirements
          v-if="constructionResult?.custom_requirements?.length"
          :requirements="constructionResult.custom_requirements"
        />
      </div>
    </div>

    <footer class="builder-footer">
      <!-- ヒアリング中 -->
      <template v-if="session?.status === 'hearing'">
        <input
          v-model="userInput"
          @keypress.enter="sendMessage"
          :placeholder="currentPlaceholder"
          :disabled="isLoading"
        />
        <button @click="sendMessage" :disabled="!userInput.trim() || isLoading">
          送信
        </button>
      </template>

      <!-- ヒアリング完了後 -->
      <template v-else-if="session?.hearingPhase === 'completed'">
        <button @click="construct" :disabled="isLoading" class="primary">
          ワークフローを構築
        </button>
      </template>

      <!-- レビュー中 -->
      <template v-else-if="session?.status === 'reviewing'">
        <input
          v-model="refineFeedback"
          @keypress.enter="refine"
          placeholder="変更したい点を入力..."
          :disabled="isLoading"
        />
        <button @click="refine" :disabled="!refineFeedback.trim() || isLoading">
          変更
        </button>
        <button @click="finalize" :disabled="isLoading" class="primary">
          確定
        </button>
      </template>
    </footer>
  </aside>
</template>

<script setup lang="ts">
const {
  session,
  messages,
  isLoading,
  startSession,
  sendMessage: send,
  constructWorkflow,
  refineWorkflow,
  finalizeWorkflow
} = useBuilder()

const initialPrompt = ref('')
const userInput = ref('')
const refineFeedback = ref('')
const constructionResult = ref<ConstructionResult | null>(null)

async function start() {
  await startSession(initialPrompt.value)
  initialPrompt.value = ''
}

async function sendMessage() {
  if (!userInput.value.trim()) return
  const content = userInput.value
  userInput.value = ''
  await send(content)
}

async function construct() {
  constructionResult.value = await constructWorkflow()
}

async function refine() {
  if (!refineFeedback.value.trim()) return
  const feedback = refineFeedback.value
  refineFeedback.value = ''
  await refineWorkflow(feedback)
}

async function finalize() {
  await finalizeWorkflow()
  emit('workflow-created', session.value?.projectId)
}

const currentPlaceholder = computed(() => {
  const phase = session.value?.hearingPhase
  const placeholders: Record<HearingPhase, string> = {
    'purpose': '何を達成したいですか？',
    'conditions': 'いつ、どのように開始・終了しますか？',
    'actors': '誰が関わりますか？',
    'frequency': 'どのくらいの頻度で実行しますか？',
    'integrations': '使用するツールはありますか？',
    'pain_points': '現在の課題は何ですか？',
    'confirmation': '上記の前提でよろしいですか？',
    'completed': ''
  }
  return placeholders[phase || 'purpose'] || 'メッセージを入力...'
})
</script>
```

---

## 実装フェーズ

### Phase 1: データモデル・基盤（3日）

| タスク | 工数 |
|--------|------|
| builder_sessions, builder_messages テーブル作成 | 0.5日 |
| BuilderSession, BuilderMessage ドメインモデル | 0.5日 |
| WorkflowSpec 型定義 | 0.5日 |
| Repository 実装 | 1日 |
| Handler 基盤実装 | 0.5日 |

### Phase 2: ヒアリングワークフロー（4日）

| タスク | 工数 |
|--------|------|
| ai-builder-hearing システムワークフロー定義 | 1日 |
| 各フェーズのプロンプトテンプレート | 1.5日 |
| セッション状態管理ロジック | 0.5日 |
| テスト・調整 | 1日 |

### Phase 3: 構築ワークフロー（5日）

| タスク | 工数 |
|--------|------|
| ai-builder-construct システムワークフロー定義 | 1日 |
| ブロックマッピングロジック | 1.5日 |
| データフロー・エッジ生成ロジック | 1日 |
| Project 生成・保存 | 0.5日 |
| テスト・調整 | 1日 |

### Phase 4: ブラッシュアップ機能（3日）

| タスク | 工数 |
|--------|------|
| ai-builder-refine システムワークフロー定義 | 1日 |
| 差分編集ロジック | 1日 |
| バージョン管理統合 | 0.5日 |
| テスト・調整 | 0.5日 |

### Phase 5: フロントエンド（5日）

| タスク | 工数 |
|--------|------|
| useBuilder.ts Composable | 1日 |
| BuilderPanel, BuilderChat コンポーネント | 1.5日 |
| WorkflowPreview, StepMappingList | 1日 |
| ワークフローエディタへの統合 | 1日 |
| テスト・調整 | 0.5日 |

### Phase 6: E2E テスト・品質向上（3日）

| タスク | 工数 |
|--------|------|
| E2E テストシナリオ作成 | 1日 |
| プロンプトチューニング | 1.5日 |
| エッジケース対応 | 0.5日 |

### 工数サマリー

| フェーズ | 工数 |
|---------|------|
| Phase 1: データモデル・基盤 | 3日 |
| Phase 2: ヒアリングワークフロー | 4日 |
| Phase 3: 構築ワークフロー | 5日 |
| Phase 4: ブラッシュアップ機能 | 3日 |
| Phase 5: フロントエンド | 5日 |
| Phase 6: E2E テスト・品質向上 | 3日 |
| **合計** | **23日** |

---

## プロンプト設計ガイドライン

### ヒアリングプロンプトの原則

1. **1フェーズ1質問**
   - 一度に複数の観点を聞かない
   - 回答を得たら次のフェーズへ

2. **選択肢の提示**
   - 抽象的な場合は具体例を提示
   - 「はい/いいえ」で答えられる質問を混ぜる

3. **仮定の明示**
   - 情報が不足している場合は仮定を提示
   - 「〇〇と仮定してよろしいですか？」

4. **進捗の可視化**
   - 残り何フェーズかを伝える
   - 「あと2つ確認させてください」

### 構築プロンプトの原則

1. **プリセット優先**
   - まず既存ブロックで実現できないか検討
   - カスタムは最終手段

2. **構造化出力**
   - JSON Schema で出力形式を厳密に定義
   - パース失敗時のフォールバック

3. **検証の組み込み**
   - 生成したワークフローの整合性チェック
   - 循環参照、孤立ノードの検出

---

## セキュリティ考慮事項

| 項目 | 対策 |
|------|------|
| プロンプトインジェクション | ユーザー入力のサニタイズ、システムプロンプトの分離 |
| 機密情報の漏洩 | シークレットキー名のみ参照、値は表示しない |
| 不正なワークフロー生成 | 生成後のバリデーション、危険なブロックの制限 |
| セッションハイジャック | テナント・ユーザー ID の検証 |
| 過剰なAPI呼び出し | レート制限、セッションタイムアウト |

---

## 将来の拡張

| 機能 | 説明 |
|------|------|
| **テンプレートライブラリ** | よく使うパターンをテンプレートとして保存・共有 |
| **学習機能** | ユーザーのフィードバックからプロンプト改善 |
| **マルチモーダル入力** | 画像や図からワークフロー生成 |
| **コラボレーション** | 複数ユーザーでの共同編集 |
| **自動最適化** | 実行履歴からの自動改善提案 |

---

## 関連ドキュメント

- [PHASE10_COPILOT.md](./PHASE10_COPILOT.md) - 基盤となる Copilot 機能
- [UNIFIED_BLOCK_MODEL.md](../designs/UNIFIED_BLOCK_MODEL.md) - ブロック定義・実行
- [BLOCK_REGISTRY.md](../BLOCK_REGISTRY.md) - 利用可能ブロック一覧
- [BACKEND.md](../BACKEND.md) - バックエンドアーキテクチャ
- [API.md](../API.md) - API 設計
