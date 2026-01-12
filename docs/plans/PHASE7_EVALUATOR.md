# Phase 7: Evaluator Block 実装計画

## 概要

**目的**: LLM出力の品質を評価し、スコアリング・フィードバックを提供する機能を実装する。

**ユースケース例**:
- 顧客対応Botの回答品質を自動評価
- コンテンツ生成の品質チェック（relevance, accuracy, clarity）
- A/Bテストで複数の出力を比較ランキング
- 品質基準を満たさない場合に再生成を要求

---

## 機能要件

### 1. 評価タイプ

| タイプ | 説明 | 出力 |
|--------|------|------|
| `scoring` | 単一の品質スコア | 0.0-1.0 |
| `criteria` | 複数基準で多面的評価 | 基準ごとのスコア |
| `comparison` | 複数出力の比較ランキング | 順位付け |
| `rubric` | 事前定義ルーブリックで評価 | グレード（A/B/C/D/F） |

### 2. 評価基準（Criteria）例

| 基準 | 説明 |
|------|------|
| `relevance` | 質問・文脈に対する関連性 |
| `accuracy` | 事実の正確性 |
| `clarity` | 明確さ・構造の良さ |
| `completeness` | 回答の完全性 |
| `helpfulness` | 有用性・実用性 |
| `tone` | トーン・スタイルの適切さ |

### 3. 出力構造

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
  ],
  "grade": "B+",
  "confidence": 0.92
}
```

---

## 技術設計

### Step Config Schema

```go
type EvaluatorConfig struct {
    Provider       string              `json:"provider"`        // openai|anthropic
    Model          string              `json:"model"`           // gpt-4o
    EvaluationType string              `json:"evaluation_type"` // scoring|criteria|comparison|rubric
    Criteria       []EvaluationCriterion `json:"criteria"`
    PassThreshold  float64             `json:"pass_threshold"`  // 0.0-1.0
    IncludeFeedback bool               `json:"include_feedback"`
    Rubric         *Rubric             `json:"rubric,omitempty"`
    CompareInputs  []string            `json:"compare_inputs,omitempty"` // comparison用
}

type EvaluationCriterion struct {
    Name        string  `json:"name"`
    Description string  `json:"description"`
    Weight      float64 `json:"weight"` // 0.0-1.0, 合計1.0
}

type Rubric struct {
    Grades []RubricGrade `json:"grades"`
}

type RubricGrade struct {
    Grade       string  `json:"grade"`       // A, B, C, D, F
    MinScore    float64 `json:"min_score"`   // 0.0-1.0
    Description string  `json:"description"`
}
```

### 評価プロンプトテンプレート

```go
const evaluationPromptTemplate = `
You are an expert evaluator. Evaluate the following response based on the given criteria.

## Original Question/Context
{{.Context}}

## Response to Evaluate
{{.Response}}

## Evaluation Criteria
{{range .Criteria}}
- {{.Name}}: {{.Description}} (Weight: {{.Weight}})
{{end}}

## Instructions
1. Evaluate the response against each criterion
2. Provide a score from 0.0 to 1.0 for each criterion
3. Calculate the weighted overall score
4. Provide constructive feedback and suggestions

## Output Format (JSON)
{
  "criteria_scores": {
    "criterion_name": score,
    ...
  },
  "overall_score": weighted_average,
  "feedback": "detailed feedback",
  "suggestions": ["suggestion1", "suggestion2"]
}
`
```

---

## 実装ステップ

### Step 1: Domain層の拡張（0.5日）

**ファイル**: `backend/internal/domain/step.go`

```go
const StepTypeEvaluator StepType = "evaluator"

type EvaluatorConfig struct { ... }
type EvaluationCriterion struct { ... }
type EvaluatorResult struct {
    Passed         bool               `json:"passed"`
    OverallScore   float64            `json:"overall_score"`
    CriteriaScores map[string]float64 `json:"criteria_scores"`
    Feedback       string             `json:"feedback"`
    Suggestions    []string           `json:"suggestions"`
    Grade          string             `json:"grade,omitempty"`
    Confidence     float64            `json:"confidence"`
}
```

### Step 2: Evaluator Adapter作成（1日）

**ファイル**: `backend/internal/adapter/evaluator.go`

```go
type EvaluatorAdapter struct {
    openaiClient    *openai.Client
    anthropicClient *anthropic.Client
}

func (a *EvaluatorAdapter) Evaluate(ctx context.Context, content string, context string, config EvaluatorConfig) (*EvaluatorResult, error) {
    switch config.EvaluationType {
    case "scoring":
        return a.evaluateScoring(ctx, content, context, config)
    case "criteria":
        return a.evaluateCriteria(ctx, content, context, config)
    case "comparison":
        return a.evaluateComparison(ctx, config.CompareInputs, config)
    case "rubric":
        return a.evaluateRubric(ctx, content, context, config)
    }
    return nil, errors.New("unknown evaluation type")
}

func (a *EvaluatorAdapter) evaluateCriteria(ctx context.Context, content, context string, config EvaluatorConfig) (*EvaluatorResult, error) {
    prompt := buildEvaluationPrompt(content, context, config.Criteria)

    // LLM呼び出し
    response, err := a.callLLM(ctx, config.Provider, config.Model, prompt)
    if err != nil {
        return nil, err
    }

    // JSONパース
    var result EvaluatorResult
    if err := json.Unmarshal([]byte(response), &result); err != nil {
        return nil, err
    }

    // 加重平均スコア計算
    result.OverallScore = calculateWeightedScore(result.CriteriaScores, config.Criteria)
    result.Passed = result.OverallScore >= config.PassThreshold

    return &result, nil
}
```

### Step 3: Executor拡張（0.5日）

**ファイル**: `backend/internal/engine/executor.go`

```go
func (e *Executor) executeEvaluatorStep(ctx context.Context, execCtx *ExecutionContext, step *domain.Step, input json.RawMessage) error {
    var config domain.EvaluatorConfig
    if err := json.Unmarshal(step.Config, &config); err != nil {
        return err
    }

    // 評価対象コンテンツの取得
    content := extractContent(input)

    // 文脈の取得（オプション）
    context := extractContext(input)

    // 評価実行
    result, err := e.evaluatorAdapter.Evaluate(ctx, content, context, config)
    if err != nil {
        return err
    }

    // 結果を保存
    return e.saveStepOutput(execCtx, step.ID, result)
}
```

### Step 4: テスト作成（0.5日）

**ファイル**: `backend/internal/adapter/evaluator_test.go`

```go
func TestEvaluatorAdapter_EvaluateCriteria(t *testing.T) {
    // LLMモック
    mockClient := &MockLLMClient{
        Response: `{
            "criteria_scores": {"relevance": 0.9, "accuracy": 0.8},
            "feedback": "Good response",
            "suggestions": []
        }`,
    }

    adapter := &EvaluatorAdapter{openaiClient: mockClient}

    config := EvaluatorConfig{
        EvaluationType: "criteria",
        Criteria: []EvaluationCriterion{
            {Name: "relevance", Weight: 0.5},
            {Name: "accuracy", Weight: 0.5},
        },
        PassThreshold: 0.7,
    }

    result, err := adapter.Evaluate(ctx, "test content", "test context", config)

    assert.NoError(t, err)
    assert.True(t, result.Passed)
    assert.Equal(t, 0.85, result.OverallScore)
}
```

### Step 5: フロントエンド UI（1日）

**ファイル**: `frontend/pages/workflows/[id].vue`

Evaluatorステップの設定フォーム:
- 評価タイプの選択（ドロップダウン）
- 評価基準の追加・編集（動的フォーム）
  - 基準名
  - 説明
  - 重み（スライダー）
- 合格しきい値（スライダー）
- フィードバック含めるかどうか（チェックボックス）
- ルーブリック設定（rubricタイプ時）

---

## 評価結果の活用パターン

### パターン1: 品質ゲート

```
[LLM] → [Evaluator] → [Condition: score >= 0.8] → [Output]
                                ↓ (不合格)
                            [Retry LLM]
```

### パターン2: フィードバックループ

```
[LLM] → [Evaluator] → [LLM: Improve based on feedback] → [Output]
```

### パターン3: A/Bテスト

```
[LLM A] ─┐
         ├→ [Evaluator: comparison] → [Best Output]
[LLM B] ─┘
```

---

## テスト計画

### ユニットテスト

| テスト | 内容 |
|--------|------|
| Scoring評価 | 単一スコア計算の正確性 |
| Criteria評価 | 複数基準の加重平均計算 |
| Comparison評価 | 複数入力のランキング |
| Rubric評価 | グレード判定 |
| しきい値判定 | passed/failedの正確性 |

### E2Eテスト

1. 高品質回答 → passed = true
2. 低品質回答 → passed = false, feedback付き
3. 複数基準評価 → 全基準のスコアが返る

---

## 依存関係

| 依存 | 理由 |
|------|------|
| OpenAI/Anthropic API | 評価用LLM呼び出し |
| JSON Schema | 出力フォーマット検証 |

---

## リスクと対策

| リスク | 対策 |
|--------|------|
| 評価の一貫性が低い | 温度パラメータを低く設定（0.0-0.3） |
| 評価コストが高い | 軽量モデル（gpt-4o-mini）をデフォルトに |
| 評価が主観的 | 明確な基準説明を促すUI |

---

## 工数見積

| タスク | 工数 |
|--------|------|
| Domain層 | 0.5日 |
| Adapter実装 | 1日 |
| Executor拡張 | 0.5日 |
| テスト | 0.5日 |
| フロントエンド | 1日 |
| ドキュメント | 0.5日 |
| **合計** | **4日** |
