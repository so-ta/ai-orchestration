# Copilot

AI アシスタント機能の概要とアーキテクチャ。

## 概要

Copilot は AI Orchestration プラットフォームの AI アシスタント機能です。ユーザーの自然言語入力からワークフローを生成・編集することができます。

## 機能

| 機能 | 説明 |
|------|------|
| ワークフロー生成 | 自然言語からワークフローを自動生成 |
| ステップ提案 | 次に追加すべきステップを提案 |
| エラー診断 | 実行エラーの原因特定と修正案 |
| 最適化提案 | パフォーマンス/コスト改善の提案 |
| 説明生成 | ワークフローの動作説明を生成 |

## アーキテクチャ

### バックエンド

```
backend/internal/usecase/
├── copilot.go              # メインロジック
├── copilot_prompt.go       # プロンプト構築
├── copilot_sanitizer.go    # 入力サニタイズ（セキュリティ）
├── copilot_validation.go   # 出力検証
├── copilot_examples.go     # Few-shot 例
└── copilot_llm.go          # LLM 呼び出し
```

### フロントエンド

```
frontend/composables/
├── useCopilot.ts           # エントリポイント
├── useCopilotDraft.ts      # ドラフト/プレビュー管理
└── copilot/
    ├── types.ts            # 型定義
    └── toolConverters.ts   # ツール結果変換
```

## セキュリティ

### プロンプトインジェクション対策

ユーザー入力は LLM に送信される前にサニタイズされます:

```go
// copilot_sanitizer.go
func SanitizeUserInput(input string) string {
    // 1. 長さ制限 (4000文字)
    // 2. 制御文字の除去
    // 3. 危険パターンの無害化
}
```

検出される危険パターン:

- `ignore previous`, `ignore above`
- `system:`, `assistant:`
- `jailbreak`, `developer mode`
- など

### 検証ログ

危険なパターンが検出された場合、警告ログが出力されます:

```go
slog.Warn("user input validation warning",
    "error", validationErr,
    "risk_level", AnalyzeInjectionRisk(input))
```

## API エンドポイント

### セッションベース

| Method | Endpoint | 説明 |
|--------|----------|------|
| POST | `/api/v1/copilot/agent/session` | セッション開始 |
| POST | `/api/v1/copilot/agent/session/{id}/message` | メッセージ送信（SSE） |
| GET | `/api/v1/copilot/sessions` | セッション一覧 |
| GET | `/api/v1/copilot/sessions/{id}` | セッション詳細 |

### 同期 API

| Method | Endpoint | 説明 |
|--------|----------|------|
| POST | `/api/v1/copilot/suggest` | ステップ提案 |
| POST | `/api/v1/copilot/diagnose` | エラー診断 |
| POST | `/api/v1/copilot/explain` | 説明生成 |
| POST | `/api/v1/copilot/optimize` | 最適化提案 |

### 非同期 API

| Method | Endpoint | 説明 |
|--------|----------|------|
| POST | `/api/v1/copilot/projects/{id}/generate` | ワークフロー生成（開始） |
| GET | `/api/v1/copilot/runs/{id}` | 実行結果取得 |

## ワークフロー生成フロー

```
User Input
    │
    ▼
┌──────────────────┐
│  SanitizeInput   │  プロンプトインジェクション対策
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  BuildPrompt     │  CoT + Self-validation
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  Call LLM        │  Claude/GPT-4
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  Parse Response  │  JSON 抽出
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  Validate        │  スキーマ検証
└────────┬─────────┘
         │
         ▼
     Response
```

## プロンプト設計

### Chain-of-Thought (CoT)

LLM に思考プロセスを段階的に実行させる:

1. ユーザー要件の分析
2. 適切なブロック選択
3. データフロー設計
4. 設定値決定
5. エッジ接続設計
6. JSON 出力

### Self-Validation

生成前にセルフチェックを実行:

- トリガーブロックの確認
- 必須フィールドの確認
- エッジ接続の確認
- 出力ポートの確認
- 循環参照の確認

## フロントエンド統合

### ドラフト/プレビューモード

Copilot の提案はまず「ドラフト」として蓄積され、ユーザーが承認するまで適用されません:

```typescript
// useCopilotDraft.ts
export type DraftStatus =
  | 'idle'        // 待機中
  | 'collecting'  // 変更収集中
  | 'previewing'  // プレビュー表示中
  | 'applying'    // 適用中
  | 'applied'     // 適用完了
  | 'discarded'   // 破棄
```

### プレビュー表示

DagEditor でプレビュー状態を視覚的に表示:

- 緑: 新規追加されるステップ
- 黄: 変更されるステップ
- 赤(破線): 削除されるステップ

## 設定

### 環境変数

| 変数 | 説明 | デフォルト |
|------|------|-----------|
| `ANTHROPIC_API_KEY` | Anthropic API キー | - |
| `OPENAI_API_KEY` | OpenAI API キー | - |
| `COPILOT_MODEL` | 使用するモデル | `claude-3-5-sonnet-20241022` |
| `COPILOT_MAX_TOKENS` | 最大トークン数 | `4096` |

## 関連ドキュメント

- [COMPONENT_ARCHITECTURE.md](./designs/COMPONENT_ARCHITECTURE.md) - コンポーネント構成
- [COPILOT_AGENT_SSE.md](./designs/COPILOT_AGENT_SSE.md) - SSE 実装詳細
- [API.md](./API.md) - API リファレンス
