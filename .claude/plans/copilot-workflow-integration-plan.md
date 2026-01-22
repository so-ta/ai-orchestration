# Copilot ワークフロー統合計画

## 概要

Copilot機能において、Goコードでハードコーディングされている処理をCopilotワークフロー自体に移行し、保守性と拡張性を向上させる計画。

---

## 実装済み

### Phase 0: インテント分類のワークフロー化 ✅

| 項目 | 状態 |
|------|------|
| LLMベースのインテント分類ステップ追加 | 完了 |
| キーワードベースのIntentClassifier非推奨化 | 完了 |
| Agent Groupのシステムプロンプト更新 | 完了 |

---

## 移行対象の分類

### A. 高優先度（即座に効果あり）

| カテゴリ | 現在地 | 移行先 | 工数 |
|---------|--------|--------|------|
| LLM設定 | `copilot_llm.go:52-94` | ワークフロー設定 | 2h |
| ワークフロー例 | `copilot_examples.go:51-300` | DB/設定ファイル | 4h |
| ブロックタイプマッピング | `copilot_autofix.go:304-342` | DB/設定ファイル | 2h |

### B. 中優先度（アーキテクチャ改善）

| カテゴリ | 現在地 | 移行先 | 工数 |
|---------|--------|--------|------|
| プロンプト生成 | `copilot_prompt.go` | テンプレートエンジン | 8h |
| 入力サニタイズ | `copilot_sanitizer.go` | 前処理ステップ | 4h |
| Confidence計算 | `copilot_validation.go:1385-1545` | 後処理ステップ | 3h |

### C. 低優先度（複雑なロジック）

| カテゴリ | 現在地 | 移行先 | 工数 |
|---------|--------|--------|------|
| 検証パイプライン | `copilot_validation.go:77-111` | 検証ワークフロー | 12h |
| Auto-Fix | `copilot_autofix.go` | 修正ワークフロー | 8h |
| サイクル検出 | `copilot_validation.go:770-886` | グラフ検証ステップ | 4h |

---

## Phase 1: LLM設定のワークフロー化

### 現状

```go
// copilot_llm.go:52-94
func GetIntentLLMConfig(intent CopilotIntent) LLMConfig {
    switch intent {
    case IntentCreate:
        return LLMConfig{Temperature: 0.5, MaxTokens: 3000}
    case IntentDebug:
        return LLMConfig{Temperature: 0.1, MaxTokens: 2000}
    // ...
    }
}
```

### 移行後

ワークフローの `set_context` ステップでインテントに基づいてLLM設定を動的に選択:

```json
{
  "temp_id": "select_llm_config",
  "name": "Select LLM Config",
  "type": "switch",
  "config": {
    "expression": "{{intent}}",
    "cases": {
      "create": {"temperature": 0.5, "max_tokens": 3000},
      "debug": {"temperature": 0.1, "max_tokens": 2000},
      "explain": {"temperature": 0.3, "max_tokens": 2500}
    },
    "default": {"temperature": 0.3, "max_tokens": 2000}
  }
}
```

---

## Phase 2: ワークフロー例のDB移行

### 現状

`copilot_examples.go` に6+個のワークフロー例がハードコーディング:
- basic_workflow
- condition_workflow
- switch_workflow
- integration_slack
- error_handling
- approval_workflow
- loop_pattern
- llm_chain
- data_pipeline

### 移行後

1. **新テーブル作成**: `copilot_workflow_examples`

```sql
CREATE TABLE copilot_workflow_examples (
    id UUID PRIMARY KEY,
    category VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    keywords JSONB NOT NULL,  -- ["並列", "配列", "ループ"]
    steps JSONB NOT NULL,
    edges JSONB NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);
```

2. **新ツール追加**: `get_relevant_examples`

```javascript
{
  "code": "const examples = ctx.db.query('copilot_workflow_examples', { keywords: input.keywords, intent: input.intent }); return examples;",
  "description": "Get relevant workflow examples based on intent and keywords"
}
```

---

## Phase 3: 入力サニタイズのワークフロー化

### 現状

`copilot_sanitizer.go` に以下のパイプライン:
1. 長さ制限（4000文字）
2. 制御文字削除
3. 空白正規化
4. 危険パターン無効化

### 移行後

ワークフローの `start` 直後に `sanitize_input` ステップを追加:

```
Start → Sanitize Input → Classify Intent → Set Context → Agent Group
```

```json
{
  "temp_id": "sanitize_input",
  "name": "Sanitize Input",
  "type": "function",
  "config": {
    "code": "const input = ctx.input.message; let sanitized = input.substring(0, 4000); sanitized = sanitized.replace(/[\\x00-\\x08\\x0B\\x0C\\x0E-\\x1F]/g, ''); const dangerous = ['ignore previous', 'system prompt', 'jailbreak']; for (const p of dangerous) { sanitized = sanitized.replace(new RegExp(p, 'gi'), '[' + p + ']'); } return { sanitized_message: sanitized, original_length: input.length, was_truncated: input.length > 4000 };"
  }
}
```

---

## Phase 4: プロンプトテンプレートエンジン

### 現状

`copilot_prompt.go` に複数のプロンプト生成関数:
- `buildProjectGenerationPromptWithCoT()`
- `getIntentInstructions()`
- `formatBlockWithFullSchema()`

### 移行後

1. **プロンプトテンプレート保存先**: `copilot_prompt_templates` テーブル

```sql
CREATE TABLE copilot_prompt_templates (
    id UUID PRIMARY KEY,
    template_key VARCHAR(100) UNIQUE NOT NULL,
    template_content TEXT NOT NULL,
    variables JSONB,  -- 必要な変数リスト
    locale VARCHAR(10) DEFAULT 'ja',
    version INT DEFAULT 1,
    is_active BOOLEAN DEFAULT true
);
```

2. **テンプレートキー例**:
   - `cot_instruction_ja`
   - `self_validation_checklist`
   - `intent_instruction_create`
   - `block_schema_format`
   - `json_output_format`

3. **新ツール**: `render_prompt`

```javascript
{
  "code": "const template = ctx.db.get('copilot_prompt_templates', { key: input.template_key }); return ctx.template.render(template.content, input.variables);",
  "description": "Render a prompt template with variables"
}
```

---

## Phase 5: 検証パイプラインのモジュール化

### 現状

`copilot_validation.go` に7段階の検証:
1. 基本構造検証
2. ステップ検証
3. エッジ検証
4. ConfigSchema検証
5. 接続性検証
6. サイクル検出
7. データフロー型検証

### 移行後

**オプション A: 検証サブワークフロー**

```
Validation Start → Parallel [
    Structure Validator,
    Step Validator,
    Edge Validator,
    Schema Validator,
    Connectivity Validator,
    Cycle Detector,
    DataFlow Validator
] → Aggregate Results → Return
```

**オプション B: 検証ツールの分離**

各検証を独立したツールとして実装し、Agent Groupから呼び出し可能に:

```javascript
// validate_structure
// validate_steps
// validate_edges
// validate_schemas
// detect_cycles
// validate_dataflow
```

### 推奨

**オプション B** を推奨。理由:
- 既存アーキテクチャとの親和性が高い
- 検証の粒度を制御可能
- Agent Groupが必要な検証のみを選択的に実行可能

---

## Phase 6: Auto-Fixのワークフロー化

### 現状

`copilot_autofix.go` に5カテゴリの自動修正:
1. 欠落フィールドの修正
2. 無効ポートの修正
3. 構造の修正
4. ブロックタイプの修正
5. 未接続ステップの修正

### 移行後

**修正ワークフローパターン**:

```
Validation Error → Classify Error → Switch [
    missing_field → Fix Missing Field,
    invalid_port → Fix Invalid Port,
    structure → Fix Structure,
    invalid_block → Fix Block Type,
    disconnected → Fix Connectivity
] → Re-validate → Loop until valid or max_retries
```

**ブロックタイプマッピングのDB化**:

```sql
CREATE TABLE copilot_block_type_mappings (
    id UUID PRIMARY KEY,
    input_type VARCHAR(100) NOT NULL,
    output_type VARCHAR(100) NOT NULL,
    confidence DECIMAL(3,2) DEFAULT 1.0,
    is_active BOOLEAN DEFAULT true
);

-- 例
INSERT INTO copilot_block_type_mappings (input_type, output_type) VALUES
('trigger', 'manual-trigger'),
('ai', 'llm'),
('if', 'condition'),
('delay', 'wait'),
('http', 'http'),
('cron', 'schedule-trigger');
```

---

## 実装順序

| 順序 | Phase | タスク | 工数 | 効果 |
|------|-------|--------|------|------|
| 1 | 1 | LLM設定のワークフロー化 | 2h | 設定変更の即時反映 |
| 2 | 2 | ワークフロー例のDB移行 | 4h | 例の追加が容易 |
| 3 | 3 | 入力サニタイズのワークフロー化 | 4h | セキュリティ強化 |
| 4 | 5-B | 検証ツールの分離 | 8h | 検証の柔軟性 |
| 5 | 6 | Auto-Fixのワークフロー化 | 8h | 修正ロジックの拡張性 |
| 6 | 4 | プロンプトテンプレートエンジン | 8h | 多言語対応・A/Bテスト |

**合計: 約34時間**

---

## 移行後のアーキテクチャ

```
┌──────────────────────────────────────────────────────────────┐
│                     Copilot Workflow                          │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  Start                                                        │
│    ↓                                                          │
│  Sanitize Input  ←── DB: dangerous_patterns                   │
│    ↓                                                          │
│  Classify Intent (LLM-based)                                  │
│    ↓                                                          │
│  Select LLM Config ←── DB: intent_llm_configs                 │
│    ↓                                                          │
│  Set Context                                                  │
│    ↓                                                          │
│  Agent Group                                                  │
│    ├── list_blocks                                            │
│    ├── get_block_schema                                       │
│    ├── create_workflow_structure                              │
│    ├── validate_* (分離された検証ツール群)                       │
│    ├── auto_fix_* (分離された修正ツール群)                       │
│    ├── get_relevant_examples ←── DB: workflow_examples        │
│    ├── render_prompt ←── DB: prompt_templates                 │
│    └── ...                                                    │
│                                                               │
└──────────────────────────────────────────────────────────────┘
```

---

## 期待される効果

| 指標 | 現状 | 移行後 |
|------|------|--------|
| 設定変更のデプロイ | 再ビルド必須 | DB更新のみ |
| 多言語対応 | ハードコーディング | テンプレート切替 |
| A/Bテスト | 困難 | 容易 |
| 検証ルール追加 | Go修正 | ツール追加 |
| ワークフロー例追加 | Go修正 | DB挿入 |
| テスト容易性 | Go単体テスト | ワークフローテスト |

---

## リスクと対策

| リスク | 対策 |
|--------|------|
| パフォーマンス低下 | キャッシュ層の追加 |
| 複雑性増加 | ドキュメント整備、段階的移行 |
| 既存機能の破壊 | 並行稼働期間を設けて検証 |
| DBスキーマ変更 | マイグレーションスクリプト作成 |

---

## 次のステップ

1. Phase 1（LLM設定）の実装開始
2. 必要なDBテーブルの設計レビュー
3. 移行テスト計画の策定
