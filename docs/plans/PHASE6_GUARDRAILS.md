# Phase 6: Guardrails Block 実装計画

## 概要

**目的**: LLM出力のコンテンツ安全検証を行い、有害なコンテンツをフィルタリングする機能を提供する。

**ユースケース例**:
- カスタマーサポートBotが不適切な回答を返さないようにする
- 個人情報（メールアドレス、電話番号等）が出力に含まれないようにマスキング
- 企業の機密情報が外部に漏れないようにブロック

---

## 機能要件

### 1. 検証タイプ

| タイプ | 説明 | 実装方法 |
|--------|------|----------|
| `toxicity` | 有害・攻撃的コンテンツの検出 | OpenAI Moderation API |
| `pii` | 個人情報の検出・マスキング | 正規表現 + LLM判定 |
| `topic` | 特定トピックの検出・ブロック | LLM分類 |
| `jailbreak` | プロンプトインジェクション検出 | パターンマッチ + LLM |
| `custom` | カスタムプロンプトによる検証 | LLM判定 |

### 2. アクション

| アクション | 説明 |
|------------|------|
| `block` | 出力をブロックし、エラーを返す |
| `warn` | 警告フラグを付けて通過 |
| `redact` | 該当部分をマスキング（`****`） |
| `retry` | 再生成を試みる |

### 3. 出力構造

```json
{
  "passed": false,
  "violations": [
    {
      "type": "pii",
      "category": "email",
      "location": "output.message",
      "original": "john@example.com",
      "action_taken": "redacted"
    }
  ],
  "original_content": "Contact john@example.com for help",
  "filtered_content": "Contact **** for help",
  "scores": {
    "toxicity": 0.02,
    "pii_detected": true
  }
}
```

---

## 技術設計

### Step Config Schema

```go
type GuardrailsConfig struct {
    Provider    string            `json:"provider"`    // openai|anthropic|custom
    Model       string            `json:"model"`       // gpt-4o-mini（custom検証用）
    Checks      []GuardrailCheck  `json:"checks"`
    OnViolation string            `json:"on_violation"` // block|passthrough_with_flag|retry
    MaxRetries  int               `json:"max_retries"`  // retry時の最大回数
}

type GuardrailCheck struct {
    Type       string   `json:"type"`       // toxicity|pii|topic|jailbreak|custom
    Threshold  float64  `json:"threshold"`  // 0.0-1.0（toxicity用）
    Categories []string `json:"categories"` // pii: [email, phone, ssn, credit_card]
    Topics     []string `json:"topics"`     // topic: [politics, religion, ...]
    Prompt     string   `json:"prompt"`     // custom: カスタム検証プロンプト
    Action     string   `json:"action"`     // block|warn|redact
}
```

### PII検出パターン（正規表現）

```go
var piiPatterns = map[string]*regexp.Regexp{
    "email":       regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
    "phone_jp":    regexp.MustCompile(`0\d{1,4}-\d{1,4}-\d{4}`),
    "phone_intl":  regexp.MustCompile(`\+\d{1,3}[-.\s]?\d{1,14}`),
    "credit_card": regexp.MustCompile(`\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}`),
    "ssn":         regexp.MustCompile(`\d{3}-\d{2}-\d{4}`),
    "ip_address":  regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`),
}
```

### OpenAI Moderation API 連携

```go
type ModerationResponse struct {
    Results []struct {
        Flagged    bool `json:"flagged"`
        Categories struct {
            Hate            bool `json:"hate"`
            HateThreatening bool `json:"hate/threatening"`
            SelfHarm        bool `json:"self-harm"`
            Sexual          bool `json:"sexual"`
            Violence        bool `json:"violence"`
        } `json:"categories"`
        CategoryScores struct {
            Hate            float64 `json:"hate"`
            HateThreatening float64 `json:"hate/threatening"`
            SelfHarm        float64 `json:"self-harm"`
            Sexual          float64 `json:"sexual"`
            Violence        float64 `json:"violence"`
        } `json:"category_scores"`
    } `json:"results"`
}
```

---

## 実装ステップ

### Step 1: Domain層の拡張（0.5日）

**ファイル**: `backend/internal/domain/step.go`

```go
// StepType追加
const StepTypeGuardrails StepType = "guardrails"

// Config構造体追加
type GuardrailsConfig struct { ... }
type GuardrailCheck struct { ... }
```

### Step 2: Guardrails Adapter作成（1.5日）

**ファイル**: `backend/internal/adapter/guardrails.go`

```go
type GuardrailsAdapter struct {
    openaiClient    *openai.Client
    anthropicClient *anthropic.Client
}

func (a *GuardrailsAdapter) Check(ctx context.Context, content string, config GuardrailsConfig) (*GuardrailsResult, error) {
    result := &GuardrailsResult{Passed: true}

    for _, check := range config.Checks {
        switch check.Type {
        case "toxicity":
            violation := a.checkToxicity(ctx, content, check)
            if violation != nil {
                result.Violations = append(result.Violations, violation)
            }
        case "pii":
            violations := a.checkPII(content, check)
            result.Violations = append(result.Violations, violations...)
        case "custom":
            violation := a.checkCustom(ctx, content, check, config.Model)
            if violation != nil {
                result.Violations = append(result.Violations, violation)
            }
        // ...
        }
    }

    result.Passed = len(result.Violations) == 0
    return result, nil
}
```

### Step 3: Executor拡張（0.5日）

**ファイル**: `backend/internal/engine/executor.go`

```go
func (e *Executor) executeGuardrailsStep(ctx context.Context, execCtx *ExecutionContext, step *domain.Step, input json.RawMessage) error {
    var config domain.GuardrailsConfig
    if err := json.Unmarshal(step.Config, &config); err != nil {
        return err
    }

    // 入力コンテンツを取得
    content := extractContent(input)

    // Guardrails検証
    result, err := e.guardrailsAdapter.Check(ctx, content, config)
    if err != nil {
        return err
    }

    // アクション実行
    switch config.OnViolation {
    case "block":
        if !result.Passed {
            return domain.ErrGuardrailsViolation
        }
    case "passthrough_with_flag":
        // フラグ付きで通過
    case "retry":
        // 再試行ロジック（LLMステップとの連携必要）
    }

    return e.saveStepOutput(execCtx, step.ID, result)
}
```

### Step 4: テスト作成（0.5日）

**ファイル**: `backend/internal/adapter/guardrails_test.go`

```go
func TestGuardrailsAdapter_CheckPII(t *testing.T) {
    tests := []struct {
        name     string
        content  string
        expected []string
    }{
        {"email", "Contact john@example.com", []string{"email"}},
        {"phone", "Call 03-1234-5678", []string{"phone_jp"}},
        {"clean", "Hello world", nil},
    }
    // ...
}

func TestGuardrailsAdapter_CheckToxicity(t *testing.T) {
    // OpenAI Moderation APIのモック
}
```

### Step 5: フロントエンド UI（1日）

**ファイル**: `frontend/pages/workflows/[id].vue`

Guardrailsステップの設定フォーム:
- 検証タイプの選択（チェックボックス）
- 各タイプのしきい値設定
- PIIカテゴリの選択
- カスタムプロンプト入力
- 違反時のアクション選択

---

## テスト計画

### ユニットテスト

| テスト | 内容 |
|--------|------|
| PII検出 | 各パターン（email, phone, etc.）の検出・マスキング |
| Toxicity | OpenAI APIモックでのスコア判定 |
| Custom | LLMモックでのカスタム検証 |
| 複合検証 | 複数チェックの組み合わせ |

### E2Eテスト

1. PIIを含む入力 → redactアクションでマスキング確認
2. Toxicコンテンツ → blockアクションでエラー確認
3. クリーンな入力 → 通過確認

---

## 依存関係

| 依存 | 理由 |
|------|------|
| OpenAI API | Moderation API, カスタム検証用LLM |
| 正規表現ライブラリ | PII検出（Go標準regexp） |

---

## リスクと対策

| リスク | 対策 |
|--------|------|
| OpenAI API障害 | フォールバック（正規表現のみ） |
| 誤検出（False Positive） | しきい値調整UI提供 |
| 処理遅延 | キャッシュ、並列処理 |

---

## 工数見積

| タスク | 工数 |
|--------|------|
| Domain層 | 0.5日 |
| Adapter実装 | 1.5日 |
| Executor拡張 | 0.5日 |
| テスト | 0.5日 |
| フロントエンド | 1日 |
| ドキュメント | 0.5日 |
| **合計** | **4.5日** |
