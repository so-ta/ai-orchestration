# Copilot機能 精度改善計画 v4

## ステータス: ✅ 完了 (2026-01-22)

## 概要

v3の実装完了を受け、調査により特定された追加の改善点に焦点を当てたプラン。
運用環境依存の機能は除外し、静的に実装可能な改善のみを含む。

## 実装完了項目

| Phase | 内容 | ステータス |
|-------|------|-----------|
| 13.1 | 循環参照検出アルゴリズム | ✅ 完了 |
| 13.2 | 循環検出のプロンプト指示追加 | ✅ 完了 |
| 14.1 | Few-shot例追加（6パターン） | ✅ 完了 |
| 14.2 | キーワードベース例選択 | ✅ 完了 |
| 15.1 | 即時自動修正関数 | ✅ 完了 |
| 15.2 | 修正フローの統合 | ✅ 完了 |
| 16.1 | ConfigSchema検証拡張 | ✅ 完了 |
| 17.1 | ブロック情報完全提供 | ✅ 完了 |
| 18.1 | データフロー型検証 | ✅ 完了 |

## 新規作成ファイル

- `backend/internal/usecase/copilot_autofix.go` - 即時自動修正機能

## 修正ファイル

- `backend/internal/usecase/copilot_validation.go` - 循環検出、データフロー検証、ConfigSchema拡張
- `backend/internal/usecase/copilot_examples.go` - 6つの新パターン、キーワードベース選択
- `backend/internal/usecase/copilot_prompt.go` - セルフチェックリスト、EnhancedConfigParam
- `backend/internal/usecase/copilot_validation_test.go` - テスト追加

---

## 調査で特定された課題

| 領域 | 現状 | 問題点 | 優先度 |
|------|------|--------|--------|
| 循環参照検出 | 未実装 | A→B→A のようなループを検出できない | P1 |
| Few-shot例 | 6パターン | ループ、LLMチェーン、複雑な分岐が不足 | P1 |
| 自動修正 | LLM再生成のみ | 単純なエラーに対する即時修正がない | P2 |
| ConfigSchema検証 | 必須・型・enum | minimum/maximum/pattern が未実装 | P2 |
| ブロック情報提供 | 基本情報のみ | 完全なスキーマ、デフォルト値が不足 | P2 |
| データフロー型検証 | 未実装 | 出力型と入力型の不一致を検出できない | P3 |

---

## 実装計画

### Phase 13: 循環参照検出（優先度: P1）

#### 13.1 グラフ循環検出アルゴリズム

**ファイル**: `backend/internal/usecase/copilot_validation.go`

```go
// ErrorCategoryCycle は循環参照エラーを表す
const ErrorCategoryCycle ErrorCategory = "cycle"

// validateCycles validates that there are no cycles in the workflow graph
func (v *WorkflowValidator) validateCycles(output *GenerateProjectOutput, result *CopilotValidationResult) {
    // Build adjacency list
    adj := make(map[string][]string)
    for _, edge := range output.Edges {
        adj[edge.SourceTempID] = append(adj[edge.SourceTempID], edge.TargetTempID)
    }

    // Detect cycles using DFS with coloring
    // 0: unvisited, 1: visiting, 2: visited
    color := make(map[string]int)
    var cyclePath []string

    var dfs func(node string, path []string) bool
    dfs = func(node string, path []string) bool {
        if color[node] == 1 {
            // Found a cycle
            cyclePath = append(path, node)
            return true
        }
        if color[node] == 2 {
            return false
        }

        color[node] = 1
        for _, next := range adj[node] {
            if dfs(next, append(path, node)) {
                return true
            }
        }
        color[node] = 2
        return false
    }

    for _, step := range output.Steps {
        if color[step.TempID] == 0 {
            if dfs(step.TempID, nil) {
                // Report cycle error
                result.Errors = append(result.Errors, CopilotValidationError{
                    Field:      "edges",
                    Message:    fmt.Sprintf("循環参照が検出されました: %s", formatCyclePath(cyclePath)),
                    Severity:   CopilotSeverityError,
                    Suggestion: "エッジを削除して循環を解消してください",
                    Category:   ErrorCategoryCycle,
                })
                break
            }
        }
    }
}
```

#### 13.2 循環検出のプロンプト指示追加

```go
// セルフチェックリストに追加
sb.WriteString("### 6. 循環参照の確認\n")
sb.WriteString("- [ ] ワークフローに循環（A→B→C→A）がないか\n")
sb.WriteString("- [ ] 自己ループ（A→A）がないか\n\n")
```

---

### Phase 14: Few-shot例の大幅拡充（優先度: P1）

#### 14.1 追加するワークフローパターン

**ファイル**: `backend/internal/usecase/copilot_examples.go`

| カテゴリ | 説明 | ポイント |
|---------|------|---------|
| loop | map/join並列処理 | 配列処理と結果集約 |
| llm_chain | LLM連鎖 | 複数LLMの順次実行 |
| nested_condition | ネスト条件分岐 | conditionの入れ子 |
| retry | リトライパターン | エラー時の再試行 |
| data_pipeline | データ変換パイプライン | filter → transform → aggregate |
| webhook_response | Webhook応答 | 外部からのリクエスト処理 |

```go
// ループ処理パターン
var loopWorkflowExample = WorkflowExample{
    Description: "配列データの並列処理（map/join使用）",
    Category:    "loop",
    Steps: []ExampleStep{
        {TempID: "step_1", Name: "トリガー", Type: "manual_trigger", Config: nil},
        {TempID: "step_2", Name: "配列展開", Type: "map", Config: map[string]interface{}{
            "input_path": "$.input.items",
            "parallel":   true,
        }},
        {TempID: "step_3", Name: "各要素をLLMで処理", Type: "llm", Config: map[string]interface{}{
            "provider":    "openai",
            "model":       "gpt-4o-mini",
            "user_prompt": "以下を要約: {{$.item}}",
        }},
        {TempID: "step_4", Name: "結果集約", Type: "join", Config: map[string]interface{}{
            "join_mode": "all",
        }},
        {TempID: "step_5", Name: "結果出力", Type: "log", Config: map[string]interface{}{
            "message": "処理完了: {{$.steps.step_4.output}}",
            "level":   "info",
        }},
    },
    Edges: []ExampleEdge{
        {Source: "step_1", Target: "step_2", SourcePort: "output"},
        {Source: "step_2", Target: "step_3", SourcePort: "item"},
        {Source: "step_3", Target: "step_4", SourcePort: "output"},
        {Source: "step_4", Target: "step_5", SourcePort: "output"},
    },
}

// LLMチェーンパターン
var llmChainExample = WorkflowExample{
    Description: "複数LLMの連鎖実行（要約→翻訳→フォーマット）",
    Category:    "llm_chain",
    Steps: []ExampleStep{
        {TempID: "step_1", Name: "トリガー", Type: "manual_trigger", Config: nil},
        {TempID: "step_2", Name: "要約", Type: "llm", Config: map[string]interface{}{
            "provider":      "openai",
            "model":         "gpt-4o-mini",
            "system_prompt": "あなたは要約の専門家です",
            "user_prompt":   "以下を3行で要約: {{$.input.text}}",
        }},
        {TempID: "step_3", Name: "翻訳", Type: "llm", Config: map[string]interface{}{
            "provider":      "openai",
            "model":         "gpt-4o-mini",
            "system_prompt": "あなたは翻訳者です",
            "user_prompt":   "以下を英語に翻訳: {{$.steps.step_2.output.content}}",
        }},
        {TempID: "step_4", Name: "フォーマット", Type: "llm", Config: map[string]interface{}{
            "provider":      "openai",
            "model":         "gpt-4o-mini",
            "system_prompt": "あなたはフォーマッターです",
            "user_prompt":   "以下をMarkdown形式に整形: {{$.steps.step_3.output.content}}",
        }},
    },
    Edges: []ExampleEdge{
        {Source: "step_1", Target: "step_2", SourcePort: "output"},
        {Source: "step_2", Target: "step_3", SourcePort: "output"},
        {Source: "step_3", Target: "step_4", SourcePort: "output"},
    },
}

// ネスト条件分岐パターン
var nestedConditionExample = WorkflowExample{
    Description: "複合条件分岐（優先度判定→カテゴリ判定）",
    Category:    "nested_condition",
    Steps: []ExampleStep{
        {TempID: "step_1", Name: "トリガー", Type: "manual_trigger", Config: nil},
        {TempID: "step_2", Name: "優先度チェック", Type: "condition", Config: map[string]interface{}{
            "expression": "$.input.priority == 'high'",
        }},
        {TempID: "step_3", Name: "緊急処理", Type: "llm", Config: map[string]interface{}{
            "provider":    "openai",
            "model":       "gpt-4o",
            "user_prompt": "緊急対応: {{$.input.message}}",
        }},
        {TempID: "step_4", Name: "カテゴリチェック", Type: "condition", Config: map[string]interface{}{
            "expression": "$.input.category == 'support'",
        }},
        {TempID: "step_5", Name: "サポート処理", Type: "llm", Config: map[string]interface{}{
            "provider":    "openai",
            "model":       "gpt-4o-mini",
            "user_prompt": "サポート回答: {{$.input.message}}",
        }},
        {TempID: "step_6", Name: "一般処理", Type: "llm", Config: map[string]interface{}{
            "provider":    "openai",
            "model":       "gpt-4o-mini",
            "user_prompt": "一般回答: {{$.input.message}}",
        }},
    },
    Edges: []ExampleEdge{
        {Source: "step_1", Target: "step_2", SourcePort: "output"},
        {Source: "step_2", Target: "step_3", SourcePort: "true"},
        {Source: "step_2", Target: "step_4", SourcePort: "false"},
        {Source: "step_4", Target: "step_5", SourcePort: "true"},
        {Source: "step_4", Target: "step_6", SourcePort: "false"},
    },
}
```

#### 14.2 キーワードベースの例選択強化

```go
// キーワードマッピングを拡充
var keywordToCategory = map[string][]string{
    "loop":             {"並列", "配列", "ループ", "繰り返し", "map", "join", "each", "forEach"},
    "llm_chain":        {"連鎖", "チェーン", "多段", "順番", "LLM2回", "LLM3回", "chain"},
    "nested_condition": {"ネスト", "入れ子", "複数条件", "条件の中に条件", "優先度"},
    "retry":            {"リトライ", "再試行", "失敗時", "エラー時", "retry"},
    "data_pipeline":    {"変換", "フィルター", "集計", "データ処理", "パイプライン"},
    "webhook_response": {"webhook", "外部連携", "API", "リクエスト"},
}

func GetExamplesForUserMessage(intent CopilotIntent, message string) []WorkflowExample {
    examples := GetExamplesForIntent(intent)
    messageLower := strings.ToLower(message)

    // キーワードマッチングで追加例を選択
    for category, keywords := range keywordToCategory {
        for _, keyword := range keywords {
            if strings.Contains(messageLower, keyword) {
                if example := GetExampleByCategory(category); example != nil {
                    examples = appendIfNotExists(examples, example)
                }
                break
            }
        }
    }

    return examples
}
```

---

### Phase 15: 即時自動修正機能（優先度: P2）

#### 15.1 LLM不要の即時修正関数

**ファイル**: `backend/internal/usecase/copilot_autofix.go`（新規）

```go
package usecase

import (
    "github.com/souta/ai-orchestration/internal/domain"
)

// AutoFixer provides immediate fixes for common validation errors
type AutoFixer struct {
    blocks map[string]*domain.BlockDefinition
}

// NewAutoFixer creates a new AutoFixer
func NewAutoFixer(blocks []*domain.BlockDefinition) *AutoFixer {
    blockMap := make(map[string]*domain.BlockDefinition)
    for _, b := range blocks {
        blockMap[b.Slug] = b
    }
    return &AutoFixer{blocks: blockMap}
}

// AutoFixResult represents the result of an auto-fix attempt
type AutoFixResult struct {
    Fixed       bool   `json:"fixed"`
    Description string `json:"description"`
}

// TryAutoFix attempts to automatically fix validation errors without LLM
func (af *AutoFixer) TryAutoFix(output *GenerateProjectOutput, errors []CopilotValidationError) ([]AutoFixResult, *GenerateProjectOutput) {
    results := make([]AutoFixResult, 0)
    modified := *output // Create a copy

    for _, err := range errors {
        switch err.Category {
        case ErrorCategoryMissingField:
            if result := af.fixMissingField(&modified, err); result.Fixed {
                results = append(results, result)
            }

        case ErrorCategoryInvalidPort:
            if result := af.fixInvalidPort(&modified, err); result.Fixed {
                results = append(results, result)
            }

        case ErrorCategoryStructure:
            if result := af.fixStructure(&modified, err); result.Fixed {
                results = append(results, result)
            }
        }
    }

    return results, &modified
}

// fixMissingField fills in missing required fields with defaults
func (af *AutoFixer) fixMissingField(output *GenerateProjectOutput, err CopilotValidationError) AutoFixResult {
    // Find the step
    for i := range output.Steps {
        step := &output.Steps[i]
        if step.Name == err.StepName || step.Type == err.BlockType {
            block := af.blocks[step.Type]
            if block == nil || block.ConfigDefaults == nil {
                continue
            }

            // Get default value from block definition
            var defaults map[string]interface{}
            if err := json.Unmarshal(block.ConfigDefaults, &defaults); err != nil {
                continue
            }

            // Extract field name from error
            fieldName := extractFieldName(err.Field)
            if defaultValue, ok := defaults[fieldName]; ok {
                if step.Config == nil {
                    step.Config = make(map[string]interface{})
                }
                step.Config[fieldName] = defaultValue
                return AutoFixResult{
                    Fixed:       true,
                    Description: fmt.Sprintf("ステップ「%s」の「%s」にデフォルト値を設定", step.Name, fieldName),
                }
            }
        }
    }
    return AutoFixResult{Fixed: false}
}

// fixInvalidPort fixes invalid source_port to default
func (af *AutoFixer) fixInvalidPort(output *GenerateProjectOutput, err CopilotValidationError) AutoFixResult {
    // Find the edge and fix source_port
    for i := range output.Edges {
        edge := &output.Edges[i]

        // Find source step
        var sourceStep *GeneratedStep
        for j := range output.Steps {
            if output.Steps[j].TempID == edge.SourceTempID {
                sourceStep = &output.Steps[j]
                break
            }
        }
        if sourceStep == nil {
            continue
        }

        // Handle condition blocks
        if sourceStep.Type == "condition" {
            if edge.SourcePort != "true" && edge.SourcePort != "false" {
                // Default to "output" -> should be "true" for first edge
                edge.SourcePort = "true"
                return AutoFixResult{
                    Fixed:       true,
                    Description: fmt.Sprintf("「%s」→「%s」のsource_portを「true」に修正", edge.SourceTempID, edge.TargetTempID),
                }
            }
        }

        // Handle other blocks - default to "output"
        block := af.blocks[sourceStep.Type]
        if block != nil && len(block.OutputPorts) > 0 {
            for _, port := range block.OutputPorts {
                if port.IsDefault {
                    edge.SourcePort = port.Name
                    return AutoFixResult{
                        Fixed:       true,
                        Description: fmt.Sprintf("「%s」→「%s」のsource_portをデフォルト「%s」に修正", edge.SourceTempID, edge.TargetTempID, port.Name),
                    }
                }
            }
        }
    }
    return AutoFixResult{Fixed: false}
}

// fixStructure fixes structural issues like missing start_step_id
func (af *AutoFixer) fixStructure(output *GenerateProjectOutput, err CopilotValidationError) AutoFixResult {
    if err.Field == "start_step_id" && output.StartStepID == "" && len(output.Steps) > 0 {
        // Find trigger step
        for _, step := range output.Steps {
            if strings.Contains(step.Type, "trigger") {
                output.StartStepID = step.TempID
                return AutoFixResult{
                    Fixed:       true,
                    Description: fmt.Sprintf("start_step_idを「%s」に設定", step.TempID),
                }
            }
        }
        // Fallback to first step
        output.StartStepID = output.Steps[0].TempID
        return AutoFixResult{
            Fixed:       true,
            Description: fmt.Sprintf("start_step_idを最初のステップ「%s」に設定", output.Steps[0].TempID),
        }
    }
    return AutoFixResult{Fixed: false}
}
```

#### 15.2 修正フローの統合

```go
// validateAndRefineWorkflow に即時修正を統合
func (u *CopilotUsecase) validateAndRefineWorkflow(...) (*GenerateProjectOutput, CopilotValidationResult, error) {
    validator := NewWorkflowValidator(blocks)
    autoFixer := NewAutoFixer(blocks)

    for iteration := 0; iteration < config.MaxIterations; iteration++ {
        result := validator.Validate(output)

        if result.IsValid {
            return output, result, nil
        }

        // Phase 1: Try auto-fix first (no LLM call)
        fixResults, fixedOutput := autoFixer.TryAutoFix(output, result.Errors)
        if len(fixResults) > 0 {
            slog.Info("auto-fix applied", "fixes", len(fixResults))
            output = fixedOutput

            // Re-validate after auto-fix
            result = validator.Validate(output)
            if result.IsValid {
                return output, result, nil
            }
        }

        // Phase 2: Use LLM for remaining errors
        refinedOutput, err := u.refineWorkflow(ctx, output, result, blocks, config.LLMConfig)
        if err != nil {
            return output, result, nil
        }
        output = refinedOutput
    }

    return output, validator.Validate(output), nil
}
```

---

### Phase 16: ConfigSchema検証の拡張（優先度: P2）

#### 16.1 数値範囲・文字列パターン検証

**ファイル**: `backend/internal/usecase/copilot_validation.go`

```go
// validateStepConfigAgainstSchema に追加検証を実装
func validateStepConfigAgainstSchema(...) []CopilotValidationError {
    // ... 既存の検証 ...

    // Check number constraints (minimum/maximum)
    if min, ok := propSchema["minimum"].(float64); ok {
        if numVal, ok := value.(float64); ok && numVal < min {
            errors = append(errors, CopilotValidationError{
                Field:      fieldName,
                Message:    fmt.Sprintf("「%s」の値「%.0f」が最小値「%.0f」未満です", fieldName, numVal, min),
                Severity:   CopilotSeverityError,
                Suggestion: fmt.Sprintf("%.0f以上の値を設定してください", min),
                Category:   ErrorCategoryTypeMismatch,
            })
        }
    }

    if max, ok := propSchema["maximum"].(float64); ok {
        if numVal, ok := value.(float64); ok && numVal > max {
            errors = append(errors, CopilotValidationError{
                Field:      fieldName,
                Message:    fmt.Sprintf("「%s」の値「%.0f」が最大値「%.0f」を超えています", fieldName, numVal, max),
                Severity:   CopilotSeverityError,
                Suggestion: fmt.Sprintf("%.0f以下の値を設定してください", max),
                Category:   ErrorCategoryTypeMismatch,
            })
        }
    }

    // Check string pattern
    if pattern, ok := propSchema["pattern"].(string); ok {
        if strVal, ok := value.(string); ok {
            re, err := regexp.Compile(pattern)
            if err == nil && !re.MatchString(strVal) {
                errors = append(errors, CopilotValidationError{
                    Field:      fieldName,
                    Message:    fmt.Sprintf("「%s」の値がパターン「%s」に一致しません", fieldName, pattern),
                    Severity:   CopilotSeverityWarning,
                    Suggestion: fmt.Sprintf("正規表現「%s」に一致する値を設定してください", pattern),
                    Category:   ErrorCategoryTypeMismatch,
                })
            }
        }
    }

    // Check string length (minLength/maxLength)
    if minLen, ok := propSchema["minLength"].(float64); ok {
        if strVal, ok := value.(string); ok && len(strVal) < int(minLen) {
            errors = append(errors, CopilotValidationError{
                Field:      fieldName,
                Message:    fmt.Sprintf("「%s」の長さが最小長「%d」未満です", fieldName, int(minLen)),
                Severity:   CopilotSeverityWarning,
                Suggestion: fmt.Sprintf("%d文字以上入力してください", int(minLen)),
                Category:   ErrorCategoryTypeMismatch,
            })
        }
    }
}
```

---

### Phase 17: ブロック情報の完全提供（優先度: P2）

#### 17.1 詳細なConfigSchema情報の提供

**ファイル**: `backend/internal/usecase/copilot_prompt.go`

```go
// EnhancedConfigParam represents detailed config parameter information
type EnhancedConfigParam struct {
    Name        string        `json:"name"`
    Type        string        `json:"type"`
    Required    bool          `json:"required"`
    Default     interface{}   `json:"default,omitempty"`
    Enum        []string      `json:"enum,omitempty"`
    Minimum     *float64      `json:"minimum,omitempty"`
    Maximum     *float64      `json:"maximum,omitempty"`
    MinLength   *int          `json:"min_length,omitempty"`
    MaxLength   *int          `json:"max_length,omitempty"`
    Pattern     string        `json:"pattern,omitempty"`
    Description string        `json:"description,omitempty"`
    Examples    []interface{} `json:"examples,omitempty"`
}

// extractEnhancedConfigParams extracts detailed config information
func extractEnhancedConfigParams(schema json.RawMessage, defaults json.RawMessage) []EnhancedConfigParam {
    var params []EnhancedConfigParam
    // ... implementation
    return params
}

// formatBlockWithFullSchema formats a block with complete schema information
func formatBlockWithFullSchema(block *domain.BlockDefinition) string {
    var sb strings.Builder

    sb.WriteString(fmt.Sprintf("## %s (%s)\n", block.Slug, block.Name))
    sb.WriteString(fmt.Sprintf("説明: %s\n\n", block.Description))

    // Config parameters with full constraints
    params := extractEnhancedConfigParams(block.ConfigSchema, block.ConfigDefaults)
    if len(params) > 0 {
        sb.WriteString("### 設定パラメータ\n")
        for _, p := range params {
            sb.WriteString(fmt.Sprintf("- **%s** (%s)", p.Name, p.Type))
            if p.Required {
                sb.WriteString(" [必須]")
            }
            sb.WriteString("\n")

            if p.Description != "" {
                sb.WriteString(fmt.Sprintf("  - 説明: %s\n", p.Description))
            }
            if p.Default != nil {
                sb.WriteString(fmt.Sprintf("  - デフォルト: %v\n", p.Default))
            }
            if len(p.Enum) > 0 {
                sb.WriteString(fmt.Sprintf("  - 許可値: %s\n", strings.Join(p.Enum, ", ")))
            }
            if p.Minimum != nil {
                sb.WriteString(fmt.Sprintf("  - 最小値: %.0f\n", *p.Minimum))
            }
            if p.Maximum != nil {
                sb.WriteString(fmt.Sprintf("  - 最大値: %.0f\n", *p.Maximum))
            }
            if len(p.Examples) > 0 {
                sb.WriteString(fmt.Sprintf("  - 例: %v\n", p.Examples[0]))
            }
        }
    }

    // Output ports
    if len(block.OutputPorts) > 0 {
        sb.WriteString("\n### 出力ポート\n")
        for _, port := range block.OutputPorts {
            defaultMark := ""
            if port.IsDefault {
                defaultMark = " [デフォルト]"
            }
            sb.WriteString(fmt.Sprintf("- **%s**%s: %s\n", port.Name, defaultMark, port.Description))
        }
    }

    return sb.String()
}
```

---

### Phase 18: データフロー型検証（優先度: P3）

#### 18.1 出力型と入力型の整合性チェック

**ファイル**: `backend/internal/usecase/copilot_validation.go`

```go
// ErrorCategoryDataFlow はデータフロー型不一致エラーを表す
const ErrorCategoryDataFlow ErrorCategory = "data_flow"

// validateDataFlow validates that output types match expected input types
func (v *WorkflowValidator) validateDataFlow(output *GenerateProjectOutput, result *CopilotValidationResult) {
    // Build step output type map
    stepOutputTypes := make(map[string]string) // step_id -> output_type

    for _, step := range output.Steps {
        block := v.blocks[step.Type]
        if block == nil {
            continue
        }

        // Get default output port type
        for _, port := range block.OutputPorts {
            if port.IsDefault {
                outputType := extractTypeFromSchema(port.Schema)
                stepOutputTypes[step.TempID] = outputType
                break
            }
        }
    }

    // Check each edge for type compatibility
    for _, edge := range output.Edges {
        sourceType := stepOutputTypes[edge.SourceTempID]
        if sourceType == "" || sourceType == "any" {
            continue
        }

        // Find target step and check expected input type
        var targetStep *GeneratedStep
        for i := range output.Steps {
            if output.Steps[i].TempID == edge.TargetTempID {
                targetStep = &output.Steps[i]
                break
            }
        }
        if targetStep == nil {
            continue
        }

        // Check if target block expects a specific type
        targetBlock := v.blocks[targetStep.Type]
        if targetBlock == nil {
            continue
        }

        // For condition blocks, input should be evaluable
        if targetStep.Type == "condition" && sourceType != "object" && sourceType != "string" {
            result.Warnings = append(result.Warnings, CopilotValidationError{
                Field:      fmt.Sprintf("edges[%s→%s]", edge.SourceTempID, edge.TargetTempID),
                Message:    fmt.Sprintf("conditionブロックへの入力型「%s」は評価困難な可能性があります", sourceType),
                Severity:   CopilotSeverityWarning,
                Suggestion: "条件式が適切に評価できることを確認してください",
                Category:   ErrorCategoryDataFlow,
            })
        }
    }
}
```

---

## 実装順序

| 順序 | Phase | タスク | 工数 | 効果 |
|------|-------|--------|------|------|
| 1 | 13.1 | 循環参照検出アルゴリズム | 2h | 高 |
| 2 | 13.2 | 循環検出のプロンプト指示追加 | 0.5h | 中 |
| 3 | 14.1 | Few-shot例追加（6パターン） | 4h | 高 |
| 4 | 14.2 | キーワードベース例選択 | 2h | 高 |
| 5 | 15.1 | 即時自動修正関数 | 4h | 高 |
| 6 | 15.2 | 修正フローの統合 | 2h | 高 |
| 7 | 16.1 | ConfigSchema検証拡張 | 3h | 中 |
| 8 | 17.1 | ブロック情報完全提供 | 3h | 中 |
| 9 | 18.1 | データフロー型検証 | 4h | 中 |

**合計: 約24.5時間**

---

## 期待される効果

| 指標 | v3実装後 | v4実装後 | 改善 |
|-----|---------|---------|------|
| 1回生成での妥当性 | 70% | 85% | +15% |
| 循環参照検出率 | 0% | 100% | +100% |
| 即時修正成功率 | 0% | 60% | +60% |
| LLM修正呼び出し削減 | - | 40% | 新規 |
| Few-shot適合率 | 60% | 85% | +25% |

---

## テストケース追加

```go
// TestValidateCycles 循環参照検出テスト
func TestValidateCycles(t *testing.T) {
    tests := []struct{
        name    string
        edges   []GeneratedEdge
        wantErr bool
    }{
        {
            name: "no cycle",
            edges: []GeneratedEdge{
                {SourceTempID: "a", TargetTempID: "b"},
                {SourceTempID: "b", TargetTempID: "c"},
            },
            wantErr: false,
        },
        {
            name: "simple cycle",
            edges: []GeneratedEdge{
                {SourceTempID: "a", TargetTempID: "b"},
                {SourceTempID: "b", TargetTempID: "a"},
            },
            wantErr: true,
        },
        {
            name: "complex cycle",
            edges: []GeneratedEdge{
                {SourceTempID: "a", TargetTempID: "b"},
                {SourceTempID: "b", TargetTempID: "c"},
                {SourceTempID: "c", TargetTempID: "a"},
            },
            wantErr: true,
        },
    }
    // ...
}

// TestAutoFixer 即時修正テスト
func TestAutoFixer(t *testing.T) {
    // ...
}

// TestGetExamplesForUserMessage キーワードベース例選択テスト
func TestGetExamplesForUserMessage(t *testing.T) {
    // ...
}
```

---

## 除外した機能（運用環境依存）

以下の機能は運用データに依存するため、本プランから除外:

- **実行履歴に基づくパターン学習**: 過去の実行結果に依存
- **ユーザー別の成功パターン推薦**: ユーザー履歴に依存
- **エラー発生頻度に基づく重み付け**: 統計データに依存
- **A/Bテストによる最適化**: 運用環境でのテストが必要
