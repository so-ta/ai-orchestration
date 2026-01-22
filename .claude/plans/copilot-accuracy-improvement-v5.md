# Copilot機能 精度改善計画 v5

## 概要

v4の実装完了を受け、さらなる精度向上とコスト効率化を目指したプラン。
LLM呼び出しの最適化、検証ロジックの強化、ユーザー体験の向上に焦点を当てる。

---

## 調査で特定された課題

| 領域 | 現状 | 問題点 | 優先度 |
|------|------|--------|--------|
| トークン使用量 | 固定2000-3000トークン | プロンプト長に基づく動的調整がない | P1 |
| ブロック情報提供 | 全ブロックを常に含める | 不要なブロック情報でトークン浪費 | P1 |
| インテント判定 | 単一パターンマッチ | 複合インテントを検出できない | P1 |
| 自動修正範囲 | 5カテゴリのみ | Cycle, DataFlowはLLM依存 | P2 |
| JSON修正 | 常にLLM呼び出し | 軽微なエラーも高コスト | P2 |
| リトライ戦略 | 固定Exponential Backoff | エラー種別に応じた調整がない | P2 |
| データフロー検証 | 基本的な型チェック | 詳細な型互換性検証がない | P2 |
| 信頼度計算 | 固定重み付け | ワークフロー特性に適応しない | P3 |
| エラー提案 | 一般的なメッセージ | 具体的な修正例がない | P3 |
| 言語判定 | なし | 英語ユーザー対応が甘い | P3 |

---

## 実装計画

### Phase 19: 動的トークン最適化（優先度: P1）

#### 19.1 プロンプト長に基づく動的MaxTokens

**ファイル**: `backend/internal/usecase/copilot_llm.go`

```go
// DynamicTokenConfig calculates optimal max_tokens based on prompt length
type DynamicTokenConfig struct {
    MinTokens      int     // 最小トークン数 (1000)
    MaxTokens      int     // 最大トークン数 (4000)
    PromptRatio    float64 // プロンプトに対する出力比率 (0.5)
    ComplexityBonus int    // 複雑なワークフローへのボーナス (500)
}

func CalculateDynamicMaxTokens(promptLength int, stepCount int, config DynamicTokenConfig) int {
    // 基本計算: プロンプト長の50%を出力用に確保
    baseTokens := int(float64(promptLength) * config.PromptRatio)

    // 複雑なワークフローにはボーナス
    if stepCount > 5 {
        baseTokens += config.ComplexityBonus
    }

    // 範囲内に収める
    return clamp(baseTokens, config.MinTokens, config.MaxTokens)
}
```

#### 19.2 推定プロンプトトークン数の計算

```go
// EstimatePromptTokens estimates token count for a prompt
// Approximation: 1 token ≈ 4 characters (English) or 1.5 characters (Japanese)
func EstimatePromptTokens(prompt string) int {
    // 日本語文字の比率を計算
    japaneseRatio := calculateJapaneseRatio(prompt)

    // 混合比率で推定
    avgCharsPerToken := 4.0 - (2.5 * japaneseRatio)
    return int(float64(len(prompt)) / avgCharsPerToken)
}
```

---

### Phase 20: スマートブロック選択（優先度: P1）

#### 20.1 インテント・キーワードベースのブロックフィルタリング

**ファイル**: `backend/internal/usecase/copilot_prompt.go`

```go
// BlockRelevanceScorer scores blocks based on relevance to user request
type BlockRelevanceScorer struct {
    intentWeights   map[CopilotIntent]map[string]float64 // インテント→カテゴリ→重み
    keywordWeights  map[string][]string                   // キーワード→ブロックslug
}

// SelectRelevantBlocks filters blocks based on intent and keywords
func SelectRelevantBlocks(blocks []*domain.BlockDefinition, intent CopilotIntent, message string, maxBlocks int) []*domain.BlockDefinition {
    scorer := NewBlockRelevanceScorer()
    scores := make(map[string]float64)

    for _, block := range blocks {
        score := scorer.CalculateRelevance(block, intent, message)
        scores[block.Slug] = score
    }

    // スコア順にソートして上位maxBlocksを返す
    return topNByScore(blocks, scores, maxBlocks)
}
```

#### 20.2 カテゴリ別の重み付け設定

```go
var intentCategoryWeights = map[CopilotIntent]map[string]float64{
    IntentCreate: {
        "trigger":     1.0,  // トリガーは必須
        "ai":          0.9,  // LLMはよく使う
        "integration": 0.7,  // 外部連携
        "control":     0.5,  // 制御フロー
        "utility":     0.3,  // ユーティリティ
    },
    IntentEnhance: {
        "control":     1.0,  // 制御フローが重要
        "utility":     0.8,
        "ai":          0.6,
        "integration": 0.5,
        "trigger":     0.3,
    },
    IntentDebug: {
        "utility":     1.0,  // ログなどが重要
        "control":     0.8,
        "ai":          0.5,
        "integration": 0.5,
        "trigger":     0.3,
    },
}
```

---

### Phase 21: 複合インテント判定（優先度: P1）

#### 21.1 複数インテントの検出と優先度付け

**ファイル**: `backend/internal/usecase/copilot_prompt.go`

```go
// IntentScore represents a detected intent with its confidence score
type IntentScore struct {
    Intent     CopilotIntent
    Score      float64
    Keywords   []string // マッチしたキーワード
}

// ClassifyWithScores returns multiple intents with confidence scores
func (ic *IntentClassifier) ClassifyWithScores(message string) []IntentScore {
    msg := strings.ToLower(message)
    results := []IntentScore{}

    intentPatterns := map[CopilotIntent][]string{
        IntentCreate:  {"追加", "作成", "作って", "add", "create", "build"},
        IntentEnhance: {"変更", "修正", "更新", "modify", "change", "update"},
        IntentExplain: {"説明", "教えて", "explain", "what", "how"},
        IntentDebug:   {"エラー", "失敗", "動かない", "error", "fail", "bug"},
    }

    for intent, patterns := range intentPatterns {
        matchedKeywords := []string{}
        for _, pattern := range patterns {
            if strings.Contains(msg, pattern) {
                matchedKeywords = append(matchedKeywords, pattern)
            }
        }

        if len(matchedKeywords) > 0 {
            score := float64(len(matchedKeywords)) / float64(len(patterns))
            results = append(results, IntentScore{
                Intent:   intent,
                Score:    score,
                Keywords: matchedKeywords,
            })
        }
    }

    // スコア順にソート
    sort.Slice(results, func(i, j int) bool {
        return results[i].Score > results[j].Score
    })

    return results
}

// GetPrimaryAndSecondaryIntents returns primary and secondary intents
func (ic *IntentClassifier) GetPrimaryAndSecondaryIntents(message string) (primary CopilotIntent, secondary *CopilotIntent) {
    scores := ic.ClassifyWithScores(message)

    if len(scores) == 0 {
        return IntentGeneral, nil
    }

    primary = scores[0].Intent

    if len(scores) > 1 && scores[1].Score > 0.3 {
        secondary = &scores[1].Intent
    }

    return primary, secondary
}
```

---

### Phase 22: 自動修正範囲の拡張（優先度: P2）

#### 22.1 循環参照の自動修正

**ファイル**: `backend/internal/usecase/copilot_autofix.go`

```go
// fixCycle attempts to break a cycle by removing the least important edge
func (af *AutoFixer) fixCycle(output *GenerateProjectOutput, err CopilotValidationError) AutoFixResult {
    // 循環パスを検出
    cyclePath := detectCyclePath(output)
    if len(cyclePath) < 2 {
        return AutoFixResult{Fixed: false}
    }

    // 循環を形成する最後のエッジ（バックエッジ）を特定
    backEdge := findBackEdge(output.Edges, cyclePath)
    if backEdge == nil {
        return AutoFixResult{Fixed: false}
    }

    // エッジを削除
    output.Edges = removeEdge(output.Edges, backEdge)

    return AutoFixResult{
        Fixed:       true,
        Description: fmt.Sprintf("循環を解消: 「%s」→「%s」のエッジを削除", backEdge.SourceTempID, backEdge.TargetTempID),
    }
}

// detectCyclePath returns the path forming a cycle
func detectCyclePath(output *GenerateProjectOutput) []string {
    // DFSで循環パスを返す
    // ...
}
```

#### 22.2 データフローエラーの簡易修正

```go
// fixDataFlow attempts to fix data flow type mismatches
func (af *AutoFixer) fixDataFlow(output *GenerateProjectOutput, err CopilotValidationError) AutoFixResult {
    // mapブロックへの非配列入力の場合、filterを挿入提案
    if strings.Contains(err.Message, "mapブロックは配列入力") {
        // filterステップの挿入を提案
        return AutoFixResult{
            Fixed:       false, // 自動挿入は危険なのでfalse
            Description: "filterブロックを追加して配列に変換することを推奨",
        }
    }

    return AutoFixResult{Fixed: false}
}
```

---

### Phase 23: JSON事前検証と軽量修正（優先度: P2）

#### 23.1 JSONスキーマに基づく事前検証

**ファイル**: `backend/internal/usecase/copilot_llm.go`

```go
// PreValidateJSON performs lightweight validation before full parsing
func PreValidateJSON(response string) (issues []string, canQuickFix bool) {
    // 1. 基本構造チェック（括弧のバランス）
    if !hasBalancedBraces(response) {
        issues = append(issues, "unbalanced_braces")
    }

    // 2. 必須フィールドの存在チェック（正規表現）
    requiredFields := []string{"steps", "edges", "start_step_id"}
    for _, field := range requiredFields {
        pattern := fmt.Sprintf(`"%s"\s*:`, field)
        if !regexp.MustCompile(pattern).MatchString(response) {
            issues = append(issues, fmt.Sprintf("missing_%s", field))
        }
    }

    // 3. 軽微なエラーはローカル修正可能
    canQuickFix = len(issues) <= 2 && !contains(issues, "unbalanced_braces")

    return issues, canQuickFix
}

// QuickFixJSON performs lightweight fixes without LLM
func QuickFixJSON(response string, issues []string) string {
    result := response

    for _, issue := range issues {
        switch {
        case issue == "trailing_comma":
            result = removeTrailingCommas(result)
        case issue == "missing_quotes":
            result = addMissingQuotes(result)
        case strings.HasPrefix(issue, "missing_"):
            // 必須フィールドの欠落はLLMに委ねる
            continue
        }
    }

    return result
}
```

---

### Phase 24: 高度なリトライ戦略（優先度: P2）

#### 24.1 エラー種別に応じたリトライ設定

**ファイル**: `backend/internal/usecase/copilot_llm.go`

```go
// AdaptiveRetryConfig adapts retry behavior based on error type
type AdaptiveRetryConfig struct {
    RateLimitDelay    time.Duration // Rate Limit時の初期遅延 (5s)
    TimeoutDelay      time.Duration // タイムアウト時の初期遅延 (2s)
    ServerErrorDelay  time.Duration // 5xx時の初期遅延 (3s)
    JitterPercent     float64       // ランダム遅延の割合 (0.2 = 20%)
    MaxTotalTime      time.Duration // リトライ全体の最大時間 (60s)
}

func (u *CopilotUsecase) callLLMWithAdaptiveRetry(ctx context.Context, prompt string, config LLMConfig) (string, error) {
    retryConfig := AdaptiveRetryConfig{
        RateLimitDelay:   5 * time.Second,
        TimeoutDelay:     2 * time.Second,
        ServerErrorDelay: 3 * time.Second,
        JitterPercent:    0.2,
        MaxTotalTime:     60 * time.Second,
    }

    startTime := time.Now()
    var lastErr error

    for attempt := 0; attempt < 5; attempt++ {
        if time.Since(startTime) > retryConfig.MaxTotalTime {
            return "", fmt.Errorf("retry timeout exceeded: %w", lastErr)
        }

        response, err := u.callLLM(ctx, prompt, config)
        if err == nil {
            return response, nil
        }

        lastErr = err
        delay := calculateAdaptiveDelay(err, attempt, retryConfig)

        select {
        case <-ctx.Done():
            return "", ctx.Err()
        case <-time.After(delay):
            continue
        }
    }

    return "", lastErr
}

func calculateAdaptiveDelay(err error, attempt int, config AdaptiveRetryConfig) time.Duration {
    var baseDelay time.Duration

    // エラー種別に応じた基本遅延
    switch {
    case isRateLimitError(err):
        baseDelay = config.RateLimitDelay
    case isTimeoutError(err):
        baseDelay = config.TimeoutDelay
    case isServerError(err):
        baseDelay = config.ServerErrorDelay
    default:
        baseDelay = 2 * time.Second
    }

    // 指数バックオフ
    delay := baseDelay * time.Duration(1<<attempt)

    // Jitter追加
    jitter := time.Duration(float64(delay) * config.JitterPercent * (rand.Float64()*2 - 1))
    delay += jitter

    // 最大30秒
    if delay > 30*time.Second {
        delay = 30 * time.Second
    }

    return delay
}
```

---

### Phase 25: 詳細データフロー型検証（優先度: P2）

#### 25.1 スキーマベースの型互換性マトリックス

**ファイル**: `backend/internal/usecase/copilot_validation.go`

```go
// TypeCompatibilityMatrix defines which types can connect to which
var TypeCompatibilityMatrix = map[string]map[string]bool{
    "string": {
        "string": true,
        "any":    true,
    },
    "object": {
        "object":    true,
        "any":       true,
        "string":    false, // 警告
        "condition": true,  // condition入力として有効
    },
    "array": {
        "array": true,
        "any":   true,
        "map":   true, // map入力として有効
    },
    "number": {
        "number":  true,
        "integer": true,
        "any":     true,
    },
}

// validateDataFlowWithMatrix performs detailed type compatibility validation
func (v *WorkflowValidator) validateDataFlowWithMatrix(output *GenerateProjectOutput, result *CopilotValidationResult) {
    for _, edge := range output.Edges {
        sourceType := v.getOutputType(output, edge.SourceTempID, edge.SourcePort)
        targetExpectedType := v.getInputExpectedType(output, edge.TargetTempID)

        if !isTypeCompatible(sourceType, targetExpectedType) {
            result.Errors = append(result.Errors, CopilotValidationError{
                Field:      fmt.Sprintf("edges[%s→%s]", edge.SourceTempID, edge.TargetTempID),
                Message:    fmt.Sprintf("型不一致: 出力「%s」と入力期待「%s」が互換性がありません", sourceType, targetExpectedType),
                Severity:   CopilotSeverityError,
                Suggestion: fmt.Sprintf("変換ブロック（function, filter）を挿入するか、出力型を変更してください"),
                Category:   ErrorCategoryDataFlow,
            })
        }
    }
}
```

---

### Phase 26: 具体的修正例の生成（優先度: P3）

#### 26.1 エラーカテゴリ別の修正コード例

**ファイル**: `backend/internal/usecase/copilot_validation.go`

```go
// GenerateFixExample generates a concrete fix example for an error
func GenerateFixExample(err CopilotValidationError, context *GenerateProjectOutput) string {
    switch err.Category {
    case ErrorCategoryMissingField:
        return generateMissingFieldExample(err)
    case ErrorCategoryInvalidPort:
        return generateInvalidPortExample(err)
    case ErrorCategoryCycle:
        return generateCycleFixExample(err, context)
    default:
        return ""
    }
}

func generateMissingFieldExample(err CopilotValidationError) string {
    field := extractFieldName(err.Field)

    switch err.BlockType {
    case "llm":
        if field == "provider" {
            return `"config": { "provider": "openai", "model": "gpt-4o-mini", "user_prompt": "..." }`
        }
        if field == "model" {
            return `"config": { "provider": "openai", "model": "gpt-4o-mini" }`
        }
    case "condition":
        if field == "expression" {
            return `"config": { "expression": "$.input.value > 0" }`
        }
    }

    return fmt.Sprintf(`"config": { "%s": <値を設定> }`, field)
}
```

---

### Phase 27: 言語検出と多言語対応（優先度: P3）

#### 27.1 ユーザー入力の言語検出

**ファイル**: `backend/internal/usecase/copilot_prompt.go`

```go
// DetectLanguage detects the primary language of the input
func DetectLanguage(text string) string {
    japaneseChars := 0
    englishChars := 0

    for _, r := range text {
        if isJapanese(r) {
            japaneseChars++
        } else if isEnglish(r) {
            englishChars++
        }
    }

    if japaneseChars > englishChars {
        return "ja"
    }
    return "en"
}

func isJapanese(r rune) bool {
    return (r >= 0x3040 && r <= 0x309F) || // ひらがな
           (r >= 0x30A0 && r <= 0x30FF) || // カタカナ
           (r >= 0x4E00 && r <= 0x9FAF)    // 漢字
}
```

#### 27.2 言語別プロンプトテンプレート

```go
var promptTemplates = map[string]map[string]string{
    "ja": {
        "self_check_header":    "## セルフチェック（生成後に必ず確認）",
        "trigger_check":        "トリガーブロックの確認",
        "required_field_check": "必須フィールドの確認",
    },
    "en": {
        "self_check_header":    "## Self-Check (Verify after generation)",
        "trigger_check":        "Trigger Block Verification",
        "required_field_check": "Required Fields Verification",
    },
}

func GetLocalizedPrompt(key, lang string) string {
    if templates, ok := promptTemplates[lang]; ok {
        if text, ok := templates[key]; ok {
            return text
        }
    }
    // フォールバック
    return promptTemplates["en"][key]
}
```

---

### Phase 28: 適応型信頼度計算（優先度: P3）

#### 28.1 ワークフロー特性に基づく動的重み付け

**ファイル**: `backend/internal/usecase/copilot_validation.go`

```go
// AdaptiveConfidenceWeights calculates weights based on workflow characteristics
func AdaptiveConfidenceWeights(output *GenerateProjectOutput) (structural, config, complexity float64) {
    stepCount := len(output.Steps)
    hasCondition := false
    hasLoop := false

    for _, step := range output.Steps {
        if step.Type == "condition" || step.Type == "switch" {
            hasCondition = true
        }
        if step.Type == "map" || step.Type == "loop" {
            hasLoop = true
        }
    }

    // 基本重み
    structural = 0.40
    config = 0.35
    complexity = 0.25

    // 条件分岐があれば構造スコアを重視
    if hasCondition {
        structural += 0.05
        complexity -= 0.05
    }

    // ループがあれば設定スコアを重視
    if hasLoop {
        config += 0.05
        structural -= 0.05
    }

    // 大規模ワークフローでは複雑度スコアを重視
    if stepCount > 8 {
        complexity += 0.05
        config -= 0.05
    }

    return structural, config, complexity
}
```

---

## 実装順序

| 順序 | Phase | タスク | 工数 | 効果 |
|------|-------|--------|------|------|
| 1 | 19.1 | 動的トークン計算 | 2h | 高 |
| 2 | 19.2 | プロンプトトークン推定 | 1h | 中 |
| 3 | 20.1 | スマートブロック選択 | 3h | 高 |
| 4 | 20.2 | カテゴリ別重み設定 | 1h | 中 |
| 5 | 21.1 | 複合インテント判定 | 3h | 高 |
| 6 | 22.1 | 循環参照の自動修正 | 3h | 高 |
| 7 | 22.2 | データフローエラー修正 | 2h | 中 |
| 8 | 23.1 | JSON事前検証 | 3h | 高 |
| 9 | 24.1 | 適応型リトライ | 3h | 中 |
| 10 | 25.1 | 型互換性マトリックス | 4h | 高 |
| 11 | 26.1 | 具体的修正例生成 | 3h | 中 |
| 12 | 27.1 | 言語検出 | 2h | 低 |
| 13 | 27.2 | 多言語テンプレート | 2h | 低 |
| 14 | 28.1 | 適応型信頼度計算 | 2h | 中 |

**合計: 約34時間**

---

## 期待される効果

| 指標 | v4実装後 | v5実装後 | 改善 |
|-----|---------|---------|------|
| LLM呼び出しコスト | 100% | 70-75% | 25-30%削減 |
| 生成ワークフローエラー率 | 15% | 5-8% | 50-60%削減 |
| ユーザー確認要否率 | 30% | 15-20% | 30-50%削減 |
| 自動修正カバー率 | 60% | 80% | +33% |
| レスポンス時間 | 8秒 | 5-6秒 | 25-30%短縮 |

---

## テストケース追加

```go
// TestDynamicTokenCalculation tests dynamic token allocation
func TestDynamicTokenCalculation(t *testing.T) {
    tests := []struct{
        name         string
        promptLength int
        stepCount    int
        wantMin      int
        wantMax      int
    }{
        {"short prompt", 500, 2, 1000, 1500},
        {"medium prompt", 2000, 5, 1500, 2500},
        {"long prompt complex", 5000, 10, 3000, 4000},
    }
    // ...
}

// TestSmartBlockSelection tests relevant block selection
func TestSmartBlockSelection(t *testing.T) {
    // ...
}

// TestCompositeIntentClassification tests multi-intent detection
func TestCompositeIntentClassification(t *testing.T) {
    tests := []struct{
        name           string
        message        string
        wantPrimary    CopilotIntent
        wantSecondary  *CopilotIntent
    }{
        {"create only", "ワークフローを作成して", IntentCreate, nil},
        {"create and explain", "作成して説明して", IntentCreate, &IntentExplain},
        {"fix and debug", "エラーを修正して", IntentDebug, &IntentEnhance},
    }
    // ...
}
```

---

## 除外した機能（高コスト・運用依存）

以下の機能は実装コストが高い、または運用データに依存するため除外:

- **セマンティックキャッシング**: インフラ構築が必要
- **強化学習フィードバックループ**: 運用データ収集が必要
- **複数LLMモデルA/Bテスト**: 別途評価基盤が必要
- **ユーザー別カスタム設定**: DB設計変更が必要
