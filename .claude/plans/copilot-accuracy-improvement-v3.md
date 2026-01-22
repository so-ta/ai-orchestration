# Copilot機能 精度改善計画 v3

## 概要

Phase 4-7（LLM最適化、検証パイプライン、CoT、Confidence）の実装完了。本プランは実装済み機能の改善・最適化に焦点を当てる。

---

## 実装済み機能一覧

| Phase | 機能 | ファイル | 状態 |
|-------|------|---------|------|
| 4.1 | Intent別LLMパラメータ | `copilot_llm.go` | ✅ 完了 |
| 4.2 | リトライ機構（指数バックオフ） | `copilot_llm.go` | ✅ 完了 |
| 4.3 | JSONパースエラー修正 | `copilot_llm.go` | ✅ 完了 |
| 5.1 | 検証パイプライン構造 | `copilot_validation.go` | ✅ 完了 |
| 5.2 | ConfigSchema型検証 | `copilot_validation.go` | ✅ 完了 |
| 5.3 | Refinementループ | `copilot_validation.go` | ✅ 完了 |
| 6.1 | Chain-of-Thoughtプロンプティング | `copilot_prompt.go` | ✅ 完了 |
| 6.2 | セルフバリデーション指示 | `copilot_prompt.go` | ✅ 完了 |
| 7.1 | Confidence活用 | `copilot_validation.go` | ✅ 完了 |

---

## 改善計画

### Phase 4改善: LLM呼び出しの調整

#### 4.1a Intent別パラメータの微調整

**ファイル**: `backend/internal/usecase/copilot_llm.go`

| Intent | 現状Temperature | 検討項目 |
|--------|-----------------|---------|
| Create | 0.5 | 複雑なワークフローでは0.4に下げて安定性向上 |
| Debug | 0.1 | 十分低い、変更不要 |
| Explain | 0.3 | 適切 |
| Enhance | 0.4 | 適切 |
| Search | 0.2 | 適切 |

**テスト方針**:
- 同じプロンプトを複数回実行し、出力のばらつきを測定
- Temperature値と出力安定性の相関を検証

#### 4.2a リトライ設定の最適化

```go
// 現状
MaxRetries:    3,
InitialDelay:  time.Second,
MaxDelay:      30 * time.Second,
BackoffFactor: 2.0,

// 検討: API負荷が低い場合の高速化
// InitialDelay: 500ms に短縮の検討
```

---

### Phase 5改善: 検証パイプラインの強化

#### 5.1a 検証エラーメッセージの改善

**現状**: 汎用的なエラーメッセージ
**改善**: ユーザーフレンドリーな具体的メッセージ

```go
// 現状
"step %d (%s) is missing required config field: %s"

// 改善案
"ステップ「%s」の設定で必須フィールド「%s」が不足しています。\n推奨値: %s"
```

#### 5.2a ConfigSchema検証の拡張

現在の検証項目:
- ✅ 必須フィールドの存在確認
- ✅ フィールドの型チェック

追加検討:
- [ ] enum値の検証（許可された値のリストとの照合）
- [ ] 数値の範囲検証（minimum/maximum）
- [ ] 文字列パターン検証（pattern）

#### 5.3a Refinementプロンプトの改善

**現状**: 汎用的な修正指示

**改善案**: エラータイプ別の具体的指示

```go
func buildRefinementPrompt(output *GenerateProjectOutput, result CopilotValidationResult) string {
    // エラータイプ別の具体的修正指示を生成
    switch classifyPrimaryError(result.Errors) {
    case "missing_field":
        return buildMissingFieldRefinementPrompt(output, result)
    case "invalid_port":
        return buildInvalidPortRefinementPrompt(output, result)
    case "disconnected":
        return buildDisconnectedRefinementPrompt(output, result)
    default:
        return buildGenericRefinementPrompt(output, result)
    }
}
```

---

### Phase 6改善: プロンプトの最適化

#### 6.1a Chain-of-Thoughtの効果測定

**測定方法**:
1. CoTありなしで同じプロンプトを実行
2. 生成結果の検証エラー率を比較
3. 生成にかかる時間を比較

#### 6.2a セルフバリデーション指示の強化

**現状の指示**:
```
生成後、以下を自己チェックしてください:
- すべての必須フィールドが設定されているか
- source_portがブロックの出力ポートに対応しているか
```

**強化案**:
```
生成後、以下を自己チェックし、問題があれば修正してください:
1. トリガーブロック: manual_trigger, webhook, cron_schedule のいずれか1つが含まれているか
2. 必須フィールド: 各ブロックのconfigSchemaを参照し、requiredフィールドが全て設定されているか
3. エッジ接続: すべてのステップが接続され、孤立したステップがないか
4. 出力ポート: source_portがソースブロックのoutputPortsに存在するか
5. 条件分岐: conditionブロックにはtrue/falseの両方の出力エッジがあるか
```

---

### Phase 7改善: Confidence計算の精緻化

#### 7.1a Confidence計算式の調整

**現状**:
```go
// 基礎スコア計算
score := 1.0
score *= 1.0 - (0.2 * float64(len(result.Errors)))     // エラー
score *= 1.0 - (0.05 * float64(len(result.Warnings)))  // 警告
```

**改善案**: エラーの重大度別重み付け

```go
func CalculateConfidenceV2(output *GenerateProjectOutput, result CopilotValidationResult) float64 {
    score := 1.0

    for _, err := range result.Errors {
        switch err.Severity {
        case SeverityError:
            score -= 0.25  // 重大エラー
        case SeverityWarning:
            score -= 0.10  // 警告
        case SeverityInfo:
            score -= 0.02  // 情報
        }
    }

    // 構造的妥当性ボーナス
    if hasTrigger(output) {
        score += 0.05
    }
    if isFullyConnected(output) {
        score += 0.05
    }

    return clamp(score, 0.0, 1.0)
}
```

#### 7.1b Confirmation閾値の動的調整

**現状**: 固定閾値 0.7

**改善案**: ワークフロー複雑度に応じた動的閾値

```go
func GetConfirmationThreshold(output *GenerateProjectOutput) float64 {
    base := 0.7

    // ステップ数による調整
    if len(output.Steps) > 5 {
        base += 0.05  // 複雑なワークフローは慎重に
    }

    // 条件分岐を含む場合
    if hasConditionalSteps(output) {
        base += 0.05
    }

    return min(base, 0.85)
}
```

---

## テスト計画

### 単体テスト

```bash
cd backend && go test ./internal/usecase/... -run TestCopilot -v
```

### テストケース一覧

| テストケース | 検証内容 |
|-------------|---------|
| TestLLMConfigByIntent | Intent別パラメータが正しく設定されるか |
| TestRetryMechanism | リトライが指数バックオフで動作するか |
| TestJSONRecovery | 不正JSON時に修正リクエストが送信されるか |
| TestValidationPipeline | 5段階検証が順番に実行されるか |
| TestRefinementLoop | 検証エラー時に修正ループが動作するか |
| TestConfidenceCalculation | Confidence計算が正しいか |
| TestConfirmationThreshold | 閾値判定が正しいか |

### 統合テストシナリオ

1. **基本ワークフロー生成**: トリガー → LLM → 通知
2. **条件分岐付きワークフロー**: condition使用（true/false両エッジ）
3. **必須フィールド欠落からの自動修正**: 修正ループの動作確認
4. **低Confidence時の確認要求**: ConfirmationRequired=trueの判定

---

## 効果測定指標

| 指標 | 測定方法 | 目標 |
|-----|---------|-----|
| 1回生成での妥当性 | 検証エラー0のケース率 | 70%+ |
| 修正ループ成功率 | Refinement後のエラー解消率 | 85%+ |
| Confidence精度 | Confidence<0.7での実際のエラー率 | 80%+ |
| API呼び出し効率 | リトライ発生率 | 5%以下 |

---

## 実装順序

| 順序 | 項目 | 工数 | 優先度 |
|------|------|------|--------|
| 1 | 5.1a エラーメッセージ改善 | 2h | 高 |
| 2 | 5.3a Refinementプロンプト改善 | 3h | 高 |
| 3 | 6.2a セルフバリデーション指示強化 | 1h | 中 |
| 4 | 7.1a Confidence計算調整 | 2h | 中 |
| 5 | 7.1b 動的閾値調整 | 2h | 中 |
| 6 | 5.2a ConfigSchema検証拡張 | 4h | 低 |
| 7 | テスト追加 | 4h | 高 |

**合計: 約18時間**

---

## 将来的な拡張計画

### Phase 8: Block情報の高度化（優先度: 高）

#### 8.1 Output Portの詳細情報追加

**ファイル**: `backend/internal/usecase/copilot_prompt.go`

```go
type EnrichedOutputPort struct {
    Name        string                 `json:"name"`
    IsDefault   bool                   `json:"is_default"`
    Type        string                 `json:"type"`
    Schema      map[string]interface{} `json:"schema,omitempty"`
    Description string                 `json:"description"`
}

func formatOutputPortDetailed(port domain.OutputPort) EnrichedOutputPort {
    // OutputPortのスキーマを解析して型情報を抽出
    // object型の場合はプロパティ構造も含める
}
```

#### 8.2 ConfigDefaultsの活用

```go
type BlockInfoForPrompt struct {
    // ... 既存フィールド
    ConfigDefaults     map[string]interface{} `json:"config_defaults"`
    RecommendedValues  map[string]string      `json:"recommended_values"`
}
```

#### 8.3 ブロック間依存関係の提示

```go
type BlockCompatibility struct {
    BlockSlug           string   `json:"block_slug"`
    RecommendedBefore   []string `json:"recommended_before"`
    RecommendedAfter    []string `json:"recommended_after"`
    IncompatibleWith    []string `json:"incompatible_with"`
}
```

---

### Phase 9: Few-shot例の拡充（優先度: 高）

#### 9.1 追加するワークフロー例

**ファイル**: `backend/internal/usecase/copilot_examples.go`

| カテゴリ | 説明 | ポイント |
|---------|------|---------|
| parallel | 並列処理（map/join） | 配列データの並列処理と結果集約 |
| loop | ループ制御 | whileループ、retry with backoff |
| multi_condition | 複合条件分岐 | 複数conditionのチェーン |
| data_transform | データ変換 | filter → map → aggregate |
| error_recovery | エラー回復 | try-catch-retry パターン |
| credential_flow | 認証フロー | OAuth、API Key設定を含む |

```go
// 並列処理例
{
    Description: "並列処理ワークフロー（map/join使用）",
    Category:    "parallel",
    Steps: []ExampleStep{
        {TempID: "step_1", Name: "開始", Type: "manual_trigger"},
        {TempID: "step_2", Name: "配列分割", Type: "map", Config: map[string]interface{}{
            "input_path": "$.input.items",
            "parallel":   true,
        }},
        {TempID: "step_3", Name: "各要素処理", Type: "llm", Config: map[string]interface{}{
            "provider": "openai",
            "model":    "gpt-4o-mini",
            "user_prompt": "Process: {{$.item}}",
        }},
        {TempID: "step_4", Name: "結果集約", Type: "join", Config: map[string]interface{}{
            "join_mode": "all",
        }},
    },
    Edges: []ExampleEdge{
        {Source: "step_1", Target: "step_2", SourcePort: "output"},
        {Source: "step_2", Target: "step_3", SourcePort: "item"},
        {Source: "step_3", Target: "step_4", SourcePort: "output"},
    },
}
```

#### 9.2 Intent別の例選択ロジック強化

```go
func GetExamplesForIntent(intent CopilotIntent, userMessage string) []WorkflowExample {
    keywords := extractKeywords(userMessage)
    if contains(keywords, []string{"並列", "parallel", "map", "配列"}) {
        examples = append(examples, parallelExample)
    }
    if contains(keywords, []string{"ループ", "繰り返し", "retry"}) {
        examples = append(examples, loopExample)
    }
    // ...
}
```

---

### Phase 10: ブロック選択とテンプレート補完（優先度: 高）

#### 10.1 ブロック選択アルゴリズムの改善

**ファイル**: `backend/internal/usecase/copilot.go`

```go
type BlockSelectionScore struct {
    CategoryMatch      float64 // カテゴリ一致度
    UserMentionScore   float64 // ユーザーメッセージでの言及度
    DependencyScore    float64 // ブロック間依存関係スコア（静的定義）
}

func (cb *ContextBuilder) SelectRelevantBlocks(
    ctx context.Context,
    intent CopilotIntent,
    message string,
    allBlocks []*domain.BlockDefinition,
) []*domain.BlockDefinition {
    // ユーザーメッセージとブロック定義のみを使用
    // 総合スコアで上位N件を返す
    return topNByScore(scores, 15)
}
```

#### 10.2 テンプレート変数の補完強化

**ファイル**: `backend/internal/usecase/copilot_prompt.go`

```go
type TemplateVariableSuggestion struct {
    Variable    string `json:"variable"`
    Type        string `json:"type"`
    Description string `json:"description"`
    Example     string `json:"example"`
}

func SuggestTemplateVariables(
    existingSteps []GeneratedStep,
    currentStepIndex int,
    blockDefinitions map[string]*domain.BlockDefinition,
) []TemplateVariableSuggestion {
    // 現在のステップより前のステップの出力スキーマ（ブロック定義から取得）を分析
    // 利用可能なテンプレート変数を提案
    // ※運用データではなく、ブロック定義の静的情報のみを使用
}
```

---

### Phase 11: エラータイプ別修正戦略（優先度: 中）

#### 11.1 エラー分類と修正戦略マッピング

**ファイル**: `backend/internal/usecase/copilot_validation.go`

```go
type ErrorCategory string

const (
    ErrorCategoryMissingField   ErrorCategory = "missing_field"
    ErrorCategoryInvalidPort    ErrorCategory = "invalid_port"
    ErrorCategoryDisconnected   ErrorCategory = "disconnected"
    ErrorCategoryTypeMismatch   ErrorCategory = "type_mismatch"
    ErrorCategoryInvalidBlock   ErrorCategory = "invalid_block"
)

type RepairStrategy struct {
    Category       ErrorCategory
    Temperature    float64
    MaxIterations  int
    PromptTemplate string
    AutoFix        func(*GenerateProjectOutput, CopilotValidationError) bool
}

var RepairStrategies = map[ErrorCategory]RepairStrategy{
    ErrorCategoryMissingField: {
        Temperature:   0.1,
        MaxIterations: 1,
        AutoFix:       autoFixMissingField,
    },
    ErrorCategoryInvalidPort: {
        Temperature:   0.3,
        MaxIterations: 2,
        PromptTemplate: invalidPortPrompt,
    },
    ErrorCategoryDisconnected: {
        Temperature:   0.5,
        MaxIterations: 3,
        PromptTemplate: disconnectedPrompt,
    },
}
```

#### 11.2 自動修正関数の実装

```go
func autoFixMissingField(output *GenerateProjectOutput, err CopilotValidationError) bool {
    // 1. エラーからステップとフィールドを特定
    // 2. ブロック定義からデフォルト値を取得
    // 3. 自動で設定値を追加
}

func autoFixInvalidPort(output *GenerateProjectOutput, err CopilotValidationError) bool {
    // 1. エラーからエッジを特定
    // 2. ソースステップの有効な出力ポートを取得
    // 3. デフォルトポートに修正
}
```

---

### Phase 12: Confidenceスコアの多次元化（優先度: 中）

#### 12.1 多次元スコア計算

```go
type ConfidenceComponents struct {
    StructuralScore    float64 // 構造的妥当性（トリガー有無、接続性）
    ConfigScore        float64 // 設定値の完全性（必須フィールド充足率）
    ComplexityScore    float64 // 複雑さに対する妥当性（ステップ数、分岐数）
}

func CalculateMultiDimensionalConfidence(
    output *GenerateProjectOutput,
    result CopilotValidationResult,
) (float64, ConfidenceComponents) {
    components := ConfidenceComponents{
        StructuralScore: calculateStructuralScore(output, result),
        ConfigScore:     calculateConfigScore(output, result),
        ComplexityScore: calculateComplexityScore(output),
    }

    // 重み付け平均（運用データ不使用）
    confidence := 0.40*components.StructuralScore +
                  0.35*components.ConfigScore +
                  0.25*components.ComplexityScore

    return confidence, components
}
```

#### 12.2 動的閾値調整

```go
func DynamicConfidenceThreshold(
    complexity int,
    hasConditional bool,
) float64 {
    base := 0.7

    if complexity > 5 {
        base += 0.05  // 複雑なワークフローは慎重に
    }
    if hasConditional {
        base += 0.05  // 条件分岐を含む場合も慎重に
    }

    return min(base, 0.85)
}
```

---

## 将来拡張の実装順序

| 順序 | Phase | タスク | 工数 | 効果 |
|------|-------|--------|------|------|
| 1 | 8.1 | Output Portの詳細情報追加 | 3h | 高 |
| 2 | 8.2 | ConfigDefaultsの活用 | 2h | 高 |
| 3 | 8.3 | ブロック間依存関係の提示 | 2h | 中 |
| 4 | 9.1 | Few-shot例の追加（6パターン） | 4h | 高 |
| 5 | 9.2 | Intent別例選択ロジック強化 | 2h | 中 |
| 6 | 10.1 | ブロック選択アルゴリズム改善 | 3h | 高 |
| 7 | 10.2 | テンプレート変数補完強化 | 3h | 中 |
| 8 | 11.1 | エラー分類と修正戦略マッピング | 3h | 中 |
| 9 | 11.2 | 自動修正関数の実装 | 4h | 中 |
| 10 | 12.1 | 多次元Confidenceスコア | 2h | 中 |
| 11 | 12.2 | 動的閾値調整 | 1h | 低 |

**将来拡張合計: 約29時間**

---

## 期待される効果（全Phase完了後）

| 指標 | 現状推定 | Phase 4-7後 | Phase 8-12後 |
|-----|--------|------------|-------------|
| 1回生成での妥当性 | 50% | 70% | 80%+ |
| 検証エラー率 | 50% | 30% | 15% |
| 修正成功率 | 60% | 85% | 90% |
| ユーザー確認率 | 60% | 40% | 25% |

---

## 除外した機能（運用環境依存）

以下の機能は運用データに依存するため、本プランから除外:

- **既存ワークフローからのパターン抽出**: テナント内の過去データに依存
- **ユーザー履歴スコア**: ユーザーの過去成功パターンに依存
- **ユーザーフィードバック機構**: DB記録・学習ループが必要
- **新規ユーザー判定**: ユーザー状態の追跡が必要
