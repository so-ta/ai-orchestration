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

## Phase 6-7: Guardrails / Evaluator

後続フェーズで実装予定。

---

## Phase 8: Variables System

### 仕様

ワークフロー全体で使用可能な変数システム。

- 環境変数
- ワークフロー変数
- シークレット管理

---

## Phase 9: Cost Tracking

### 仕様

API使用量とコストの追跡。

- トークン使用量の記録
- コスト計算（プロバイダー別）
- ダッシュボード表示

---

## Phase 10: Copilot

### 仕様

AIアシスタントによるワークフロー構築支援。

- 自然言語でのワークフロー説明
- 自動ブロック追加
- ベストプラクティス提案

---

## 実装優先度の理由

1. **Loop** - 既存のmapステップを補完し、より柔軟な繰り返し処理を実現
2. **Human in Loop** - 承認フローは企業利用で必須
3. **Wait** - 実装が比較的簡単で有用
4. **Function** - カスタムロジックの需要が高い
5. **Router** - AI活用の差別化要因
6-7. **Guardrails/Evaluator** - 品質保証機能
8-10. **Variables/Cost/Copilot** - 付加価値機能
