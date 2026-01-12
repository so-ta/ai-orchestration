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
| 6 | `guardrails` | コンテンツ安全検証 | 📋 未実装 | [設計書](./plans/PHASE6_GUARDRAILS.md) |
| 7 | `evaluator` | 出力品質評価 | 📋 未実装 | [設計書](./plans/PHASE7_EVALUATOR.md) |

---

## 未実装機能の詳細設計

Phase 6-10 の詳細設計は以下のドキュメントを参照：

| Phase | Feature | Plan Document |
|-------|---------|---------------|
| 6 | Guardrails | [PHASE6_GUARDRAILS.md](./plans/PHASE6_GUARDRAILS.md) |
| 7 | Evaluator | [PHASE7_EVALUATOR.md](./plans/PHASE7_EVALUATOR.md) |
| 8 | Variables System | [PHASE8_VARIABLES.md](./plans/PHASE8_VARIABLES.md) |
| 9 | Cost Tracking | [PHASE9_COST_TRACKING.md](./plans/PHASE9_COST_TRACKING.md) |
| 10 | Copilot | [PHASE10_COPILOT.md](./plans/PHASE10_COPILOT.md) |

**推奨実装順序**: Phase 8 → 9 → 6 → 7 → 10

---

## 実装済み機能の参照

Phase 1-5 の詳細仕様は正式ドキュメントに統合済み：

| Phase | Step Type | 正式ドキュメント |
|-------|-----------|-----------------|
| 1 | `loop` | [BACKEND.md - Loop Step](./BACKEND.md#step-config-schemas) |
| 2 | `human_in_loop` | [BACKEND.md - Human-in-Loop Step](./BACKEND.md#step-config-schemas) |
| 3 | `wait` | [BACKEND.md - Wait Step](./BACKEND.md#step-config-schemas) |
| 4 | `function` | [BACKEND.md - Function Step](./BACKEND.md#step-config-schemas) |
| 5 | `router` | [BACKEND.md - Router Step](./BACKEND.md#step-config-schemas) |

**実装上の注意点**
- `function` ステップ: JavaScript実行はパススルー実装（入力をそのまま返す）
- `human_in_loop` ステップ: テストモードでは自動承認、本番モードではpending状態

**関連コード**
- Backend: `backend/internal/domain/step.go`, `backend/internal/engine/executor.go`
- Frontend: `frontend/types/api.ts`, `frontend/pages/workflows/[id].vue`

