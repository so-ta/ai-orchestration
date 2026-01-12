# Sim.ai互換機能 実装計画

## 概要

Sim.aiの主要機能を参考に、AI Orchestrationプラットフォームを拡張する。

## 現在の機能 vs 追加予定機能

### 既存Step Types

| Type | 説明 | 状態 |
|------|------|------|
| `llm` | LLM API呼び出し | ✅ 実装済み |
| `tool` | アダプター実行 | ✅ 実装済み |
| `condition` | 条件分岐 | ✅ 実装済み |
| `map` | 配列の並列/逐次処理 | ✅ 実装済み |
| `join` | 出力のマージ | ✅ 実装済み |
| `subflow` | サブワークフロー | ✅ 実装済み |

### 追加Step Types

| Phase | Type | 説明 | 状態 |
|-------|------|------|------|
| 1 | `loop` | for/forEach/while/do-while | ✅ 実装済み |
| 2 | `human_in_loop` | 人間の承認ゲート | ✅ 実装済み |
| 3 | `wait` | 遅延/タイムアウト | ✅ 実装済み |
| 4 | `function` | カスタムJS/TS実行 | ✅ 実装済み（パススルー） |
| 5 | `router` | AI動的ルーティング | ✅ 実装済み |
| 6 | `guardrails` | コンテンツ安全検証 | 📋 未実装 |
| 7 | `evaluator` | 出力品質評価 | 📋 未実装 |

---

## 実装完了ノート

### 2024-01: Phase 1-5 実装完了

以下が実装済み：

**バックエンド**
- `backend/internal/domain/step.go` - 新規ステップタイプとConfig構造体
- `backend/internal/engine/executor.go` - 実行ロジック（executeLoopStep, executeWaitStep等）
- `backend/internal/engine/executor_test.go` - ユニットテスト

**フロントエンド**
- `frontend/types/api.ts` - StepType定義
- `frontend/components/dag-editor/DagEditor.vue` - ノードカラー
- `frontend/pages/workflows/[id].vue` - Step設定フォーム

**ドキュメント**
- `docs/BACKEND.md` - Step Config Schemas
- `CLAUDE.md` - Step Types テーブル

**注意点**
- `function` ステップ: JavaScript実行は未実装（入力パススルー）
- `human_in_loop` ステップ: テストモードでは自動承認、本番モードではpending状態

---

## Phase 1: Loop Block

### 仕様

4つのループタイプをサポート：

1. **for** - 固定回数の繰り返し
2. **forEach** - 配列の各要素を処理
3. **while** - 条件が真の間繰り返し
4. **do-while** - 最低1回実行し、条件が真の間繰り返し

### Config Schema

```json
{
  "loop_type": "for|forEach|while|doWhile",
  "count": 10,                    // for: 繰り返し回数
  "input_path": "$.items",        // forEach: 配列パス
  "condition": "$.index < 10",    // while/doWhile: 継続条件
  "max_iterations": 100,          // 無限ループ防止
  "inner_steps": ["step_id_1"]    // ループ内で実行するステップ
}
```

### 出力

```json
{
  "results": [...],           // 各イテレーションの結果
  "iterations": 10,           // 実行回数
  "completed": true           // 正常完了したか
}
```

### 実装ファイル

1. `backend/internal/domain/step.go` - StepTypeLoop追加
2. `backend/internal/engine/executor.go` - executeLoopStep追加
3. `frontend/components/dag-editor/` - LoopノードUI
4. `docs/BACKEND.md` - ドキュメント更新

---

## Phase 2: Human in the Loop

### 仕様

ワークフロー実行を一時停止し、人間の介入を待つ。

### Config Schema

```json
{
  "timeout_hours": 24,           // タイムアウト時間
  "notification": {
    "type": "email|slack|webhook",
    "target": "..."
  },
  "approval_url": true,          // 承認URLを生成
  "required_fields": [           // 承認時に必要な入力
    {"name": "approved", "type": "boolean"},
    {"name": "comment", "type": "string"}
  ]
}
```

### フロー

1. ステップ到達時にRun状態を `waiting_approval` に変更
2. 通知を送信（設定されている場合）
3. 承認URLまたはAPIエンドポイントで待機
4. 承認/却下後に実行を再開

---

## Phase 3: Wait Block

### 仕様

指定時間だけ実行を遅延させる。

### Config Schema

```json
{
  "duration_ms": 5000,           // 遅延時間（ミリ秒）
  "until": "2024-01-01T00:00:00Z" // 特定時刻まで待機
}
```

---

## Phase 4: Function Block

### 仕様

カスタムJavaScript/TypeScriptコードを実行。

### Config Schema

```json
{
  "code": "return input.value * 2;",
  "language": "javascript",
  "timeout_ms": 5000
}
```

### セキュリティ考慮

- サンドボックス実行（goja等のJSランタイム）
- リソース制限（CPU、メモリ、実行時間）
- ネットワークアクセス制限

---

## Phase 5: Router Block

### 仕様

LLMを使用して動的にルーティング先を決定。

### Config Schema

```json
{
  "routes": [
    {"name": "support", "description": "Customer support questions"},
    {"name": "sales", "description": "Sales inquiries"},
    {"name": "technical", "description": "Technical issues"}
  ],
  "model": "gpt-4o-mini",
  "prompt": "Classify the following request..."
}
```

---

## Phase 6: Guardrails Block

### 仕様

LLM出力のコンテンツ安全検証を行い、有害なコンテンツをフィルタリング。

### Config Schema

```json
{
  "provider": "openai|anthropic|custom",
  "model": "gpt-4o-mini",
  "checks": [
    {
      "type": "toxicity",
      "threshold": 0.8,
      "action": "block|warn|flag"
    },
    {
      "type": "pii",
      "categories": ["email", "phone", "ssn", "credit_card"],
      "action": "redact|block"
    },
    {
      "type": "custom",
      "prompt": "Check if the content contains any confidential information...",
      "fail_keywords": ["confidential", "secret"]
    }
  ],
  "on_violation": "block|passthrough_with_flag|retry"
}
```

### 検証タイプ

| タイプ | 説明 |
|--------|------|
| `toxicity` | 有害・攻撃的コンテンツの検出 |
| `pii` | 個人情報（PII）の検出・マスキング |
| `custom` | カスタムプロンプトによる検証 |
| `topic` | 特定トピックの検出・ブロック |
| `jailbreak` | プロンプトインジェクション検出 |

### 出力

```json
{
  "passed": false,
  "violations": [
    {
      "type": "pii",
      "category": "email",
      "location": "output.message",
      "action_taken": "redacted"
    }
  ],
  "original_content": "...",
  "filtered_content": "..."
}
```

### 実装ファイル

| ファイル | 変更内容 |
|----------|----------|
| `backend/internal/domain/step.go` | StepTypeGuardrails, GuardrailsConfig追加 |
| `backend/internal/engine/executor.go` | executeGuardrailsStep追加 |
| `backend/internal/adapter/guardrails.go` | 検証ロジック実装 |
| `frontend/pages/workflows/[id].vue` | 設定UI追加 |

### 依存関係

- OpenAI Moderation API または Anthropic Content Filter
- 正規表現ベースのPII検出ライブラリ

### 工数見積

- バックエンド: 3日
- フロントエンド: 1日
- テスト: 1日

---

## Phase 7: Evaluator Block

### 仕様

LLM出力の品質を評価し、スコアリング・フィードバックを提供。

### Config Schema

```json
{
  "provider": "openai|anthropic",
  "model": "gpt-4o",
  "evaluation_type": "scoring|comparison|criteria",
  "criteria": [
    {
      "name": "relevance",
      "description": "How relevant is the response to the question?",
      "weight": 0.4
    },
    {
      "name": "accuracy",
      "description": "Is the information factually correct?",
      "weight": 0.4
    },
    {
      "name": "clarity",
      "description": "Is the response clear and well-structured?",
      "weight": 0.2
    }
  ],
  "pass_threshold": 0.7,
  "include_feedback": true
}
```

### 評価タイプ

| タイプ | 説明 |
|--------|------|
| `scoring` | 0-1のスコアで品質評価 |
| `comparison` | 複数の出力を比較してランキング |
| `criteria` | 複数の基準で多面的に評価 |
| `rubric` | 事前定義されたルーブリックで評価 |

### 出力

```json
{
  "passed": true,
  "overall_score": 0.85,
  "criteria_scores": {
    "relevance": 0.9,
    "accuracy": 0.8,
    "clarity": 0.85
  },
  "feedback": "The response is highly relevant and accurate. Consider adding more structure for clarity.",
  "suggestions": [
    "Add bullet points for key information",
    "Include a brief summary at the end"
  ]
}
```

### 実装ファイル

| ファイル | 変更内容 |
|----------|----------|
| `backend/internal/domain/step.go` | StepTypeEvaluator, EvaluatorConfig追加 |
| `backend/internal/engine/executor.go` | executeEvaluatorStep追加 |
| `backend/internal/adapter/evaluator.go` | 評価ロジック実装 |
| `frontend/pages/workflows/[id].vue` | 設定UI追加 |

### 工数見積

- バックエンド: 2日
- フロントエンド: 1日
- テスト: 1日

---

## Phase 8: Variables System

### 仕様

ワークフロー全体で使用可能な変数システム。3層の変数スコープをサポート。

### 変数スコープ

| スコープ | 説明 | 例 |
|----------|------|-----|
| `system` | システム全体で共有 | APIキー、共通設定 |
| `workflow` | ワークフロー固有 | ワークフロー設定、定数 |
| `run` | 実行時のみ有効 | 入力パラメータ、中間結果 |

### データモデル

```sql
-- ワークフロー変数
CREATE TABLE workflow_variables (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    workflow_id UUID REFERENCES workflows(id), -- NULLでテナント共通
    name VARCHAR(255) NOT NULL,
    value_type VARCHAR(50) NOT NULL, -- string|number|boolean|json|secret
    value TEXT, -- 暗号化される場合あり
    is_secret BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, workflow_id, name)
);
```

### 変数参照構文

```
${var.workflow.api_timeout}     -- ワークフロー変数
${var.system.openai_model}      -- システム変数
${var.run.user_input}           -- 実行時変数
${secret.openai_api_key}        -- シークレット（マスク表示）
```

### API エンドポイント

| Method | Path | 説明 |
|--------|------|------|
| GET | `/api/v1/variables` | 変数一覧 |
| POST | `/api/v1/variables` | 変数作成 |
| PUT | `/api/v1/variables/{id}` | 変数更新 |
| DELETE | `/api/v1/variables/{id}` | 変数削除 |
| GET | `/api/v1/workflows/{id}/variables` | ワークフロー変数一覧 |

### 実装ファイル

| ファイル | 変更内容 |
|----------|----------|
| `backend/internal/domain/variable.go` | Variable, VariableScope 定義 |
| `backend/internal/repository/postgres/variable.go` | CRUD実装 |
| `backend/internal/usecase/variable.go` | ビジネスロジック |
| `backend/internal/handler/variable.go` | APIハンドラ |
| `backend/internal/engine/executor.go` | 変数解決ロジック |
| `backend/migrations/XXX_add_variables.sql` | DBマイグレーション |
| `frontend/pages/workflows/[id]/variables.vue` | 変数管理UI |
| `frontend/composables/useVariables.ts` | API呼び出し |

### セキュリティ考慮

- シークレット変数は暗号化して保存
- 変数値はログに出力しない
- シークレットはUI上でマスク表示（`****`）
- テナント間の変数分離を厳密に

### 工数見積

- バックエンド: 4日
- フロントエンド: 2日
- テスト: 2日

---

## Phase 9: Cost Tracking

### 仕様

API使用量とコストの追跡・可視化。

### データモデル

```sql
CREATE TABLE usage_records (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    workflow_id UUID REFERENCES workflows(id),
    run_id UUID REFERENCES runs(id),
    step_run_id UUID REFERENCES step_runs(id),
    provider VARCHAR(50) NOT NULL, -- openai|anthropic|...
    model VARCHAR(100) NOT NULL,
    operation VARCHAR(50) NOT NULL, -- chat|completion|embedding|...
    input_tokens INT NOT NULL DEFAULT 0,
    output_tokens INT NOT NULL DEFAULT 0,
    total_tokens INT NOT NULL DEFAULT 0,
    cost_usd DECIMAL(10, 6) NOT NULL DEFAULT 0,
    latency_ms INT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_usage_records_tenant_date ON usage_records(tenant_id, created_at);
CREATE INDEX idx_usage_records_workflow ON usage_records(workflow_id);
```

### コスト計算

```go
var tokenPricing = map[string]map[string]float64{
    "openai": {
        "gpt-4o":       0.005,  // per 1K input tokens
        "gpt-4o-mini":  0.00015,
        "gpt-4-turbo":  0.01,
    },
    "anthropic": {
        "claude-3-opus":   0.015,
        "claude-3-sonnet": 0.003,
        "claude-3-haiku":  0.00025,
    },
}
```

### API エンドポイント

| Method | Path | 説明 |
|--------|------|------|
| GET | `/api/v1/usage` | 使用量サマリー |
| GET | `/api/v1/usage/daily` | 日別使用量 |
| GET | `/api/v1/usage/by-workflow` | ワークフロー別使用量 |
| GET | `/api/v1/usage/by-model` | モデル別使用量 |
| GET | `/api/v1/runs/{id}/usage` | Run別使用量詳細 |

### ダッシュボード表示項目

- 今月のコスト合計
- 日別コスト推移グラフ
- ワークフロー別コストランキング
- モデル別使用量内訳
- トークン使用量推移

### 実装ファイル

| ファイル | 変更内容 |
|----------|----------|
| `backend/internal/domain/usage.go` | UsageRecord定義 |
| `backend/internal/repository/postgres/usage.go` | CRUD・集計クエリ |
| `backend/internal/usecase/usage.go` | 集計ロジック |
| `backend/internal/handler/usage.go` | APIハンドラ |
| `backend/internal/adapter/*.go` | 使用量記録フック追加 |
| `backend/migrations/XXX_add_usage_records.sql` | DBマイグレーション |
| `frontend/pages/usage/index.vue` | ダッシュボード |
| `frontend/components/usage/CostChart.vue` | グラフコンポーネント |

### 工数見積

- バックエンド: 3日
- フロントエンド: 3日
- テスト: 1日

---

## Phase 10: Copilot

### 仕様

AIアシスタントによるワークフロー構築支援。

### 機能一覧

| 機能 | 説明 |
|------|------|
| 自然言語入力 | 「メールを分類してSlackに通知」→ワークフロー生成 |
| ブロック提案 | 次に追加すべきブロックを提案 |
| エラー診断 | 実行エラーの原因分析と修正提案 |
| 最適化提案 | パフォーマンス・コスト最適化のアドバイス |
| テンプレート推薦 | 類似ユースケースのテンプレート提案 |

### API エンドポイント

| Method | Path | 説明 |
|--------|------|------|
| POST | `/api/v1/copilot/generate` | 自然言語からワークフロー生成 |
| POST | `/api/v1/copilot/suggest` | 次のブロック提案 |
| POST | `/api/v1/copilot/diagnose` | エラー診断 |
| POST | `/api/v1/copilot/optimize` | 最適化提案 |

### Request/Response 例

**ワークフロー生成:**

```json
// Request
{
  "prompt": "顧客からの問い合わせメールを受信し、AIで分類して、緊急の場合はSlack通知、それ以外はチケット作成"
}

// Response
{
  "workflow": {
    "name": "Customer Inquiry Handler",
    "steps": [
      {"type": "tool", "name": "Receive Email", "config": {...}},
      {"type": "llm", "name": "Classify Inquiry", "config": {...}},
      {"type": "condition", "name": "Is Urgent?", "config": {...}},
      {"type": "tool", "name": "Send Slack", "config": {...}},
      {"type": "tool", "name": "Create Ticket", "config": {...}}
    ],
    "edges": [...]
  },
  "explanation": "このワークフローは以下の流れで動作します..."
}
```

### 実装ファイル

| ファイル | 変更内容 |
|----------|----------|
| `backend/internal/usecase/copilot.go` | Copilotロジック |
| `backend/internal/handler/copilot.go` | APIハンドラ |
| `backend/internal/adapter/copilot_prompts.go` | プロンプトテンプレート |
| `frontend/components/copilot/CopilotChat.vue` | チャットUI |
| `frontend/components/copilot/SuggestionPanel.vue` | 提案パネル |
| `frontend/pages/workflows/[id].vue` | Copilot統合 |

### 工数見積

- バックエンド: 5日
- フロントエンド: 4日
- テスト: 2日

---

## 実装ロードマップ

### 推奨実装順序

| 順序 | Phase | 理由 |
|------|-------|------|
| 1 | Phase 8: Variables | 他機能の基盤となる |
| 2 | Phase 9: Cost Tracking | 運用に必須 |
| 3 | Phase 6: Guardrails | 本番運用の安全性 |
| 4 | Phase 7: Evaluator | 品質保証 |
| 5 | Phase 10: Copilot | 最も複雑、最後に実装 |

### 合計工数見積

| Phase | バックエンド | フロントエンド | テスト | 合計 |
|-------|------------|--------------|--------|------|
| Phase 6 | 3日 | 1日 | 1日 | 5日 |
| Phase 7 | 2日 | 1日 | 1日 | 4日 |
| Phase 8 | 4日 | 2日 | 2日 | 8日 |
| Phase 9 | 3日 | 3日 | 1日 | 7日 |
| Phase 10 | 5日 | 4日 | 2日 | 11日 |
| **合計** | **17日** | **11日** | **7日** | **35日** |

---

## 実装優先度の理由

1. **Loop** - 既存のmapステップを補完し、より柔軟な繰り返し処理を実現
2. **Human in Loop** - 承認フローは企業利用で必須
3. **Wait** - 実装が比較的簡単で有用
4. **Function** - カスタムロジックの需要が高い
5. **Router** - AI活用の差別化要因
6-7. **Guardrails/Evaluator** - 品質保証機能
8-10. **Variables/Cost/Copilot** - 付加価値機能
