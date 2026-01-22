# コアバリュー実現のための詳細実装計画

**目標**: 「ユーザーが任意のワークフローを作ろうとしたときに、Copilotが自動でそのワークフローを作成し、すぐに運用に載せられる」

**現状達成度**: 55% → **目標達成度**: 90%以上

---

## Phase 1: 致命的問題の解決（必須）

### 1.1 トリガー有効化機制の実装

#### 背景
現状、`trigger_config.enabled` フィールドは存在するが、UIで管理できず、デフォルト値が `false` のため新規作成したトリガーが自動的に無効化される。

#### 実装タスク

**Backend（Go）**

```
1. API エンドポイント追加
   ファイル: backend/internal/handler/step.go

   PUT /api/v1/projects/{project_id}/steps/{step_id}/trigger/enable
   PUT /api/v1/projects/{project_id}/steps/{step_id}/trigger/disable

   実装内容:
   - step.TriggerConfig の enabled フィールドを更新
   - Schedule の場合は schedules テーブルも連動更新

2. Usecase 層追加
   ファイル: backend/internal/usecase/step.go

   func (u *StepUsecase) EnableTrigger(ctx, tenantID, projectID, stepID uuid.UUID) error
   func (u *StepUsecase) DisableTrigger(ctx, tenantID, projectID, stepID uuid.UUID) error

   実装内容:
   - TriggerType が manual 以外の場合のみ有効
   - Schedule 連動処理（pause/resume）

3. デフォルト値の変更
   ファイル: backend/internal/domain/step.go

   新規作成時のトリガー設定で enabled: true をデフォルトに
   ただし、リリース前は false のままでもOK（明示的な有効化フロー）
```

**Frontend（Vue 3）**

```
4. TriggerConfigPanel にトグルスイッチ追加
   ファイル: frontend/components/workflow-editor/TriggerConfigPanel.vue

   変更内容:
   - enabled 状態を表示するトグルスイッチ追加
   - emit('update:trigger-enabled', boolean) 追加
   - Webhook/Schedule/Slack/Email タイプで表示

5. API 呼び出し用 composable
   ファイル: frontend/composables/useSteps.ts（新規または既存に追加）

   export function useSteps() {
     const enableTrigger = (projectId: string, stepId: string) => ...
     const disableTrigger = (projectId: string, stepId: string) => ...
   }

6. PropertiesPanel での統合
   ファイル: frontend/components/workflow-editor/PropertiesPanel.vue

   - TriggerConfigPanel からの emit 処理
   - 保存ボタンとは別に即時反映（トグル変更時に API 呼び出し）
```

**型定義更新**

```
7. frontend/types/api.ts

   interface TriggerConfig {
     enabled?: boolean  // 明示的に追加
     // ... 既存フィールド
   }
```

#### 見積もり工数
- Backend: 4時間
- Frontend: 3時間
- テスト: 2時間
- **合計: 9時間**

---

### 1.2 テスト実行機能の強化

#### 背景
ステップ単体テスト（`testStepInline`）は実装済みだが、ワークフロー全体のドライラン機能が不完全。

#### 実装タスク

**Backend（Go）**

```
1. ドライランモード追加
   ファイル: backend/internal/usecase/run.go

   type CreateRunInput struct {
     ...
     DryRun bool // 追加: 実際の外部API呼び出しをスキップ
   }

   実装内容:
   - DryRun=true の場合、外部サービス呼び出しをモック
   - 入力検証とフロー確認のみ実施
   - 結果は正常系のシミュレーション値を返す

2. 事前検証 API
   ファイル: backend/internal/handler/run.go

   POST /api/v1/projects/{project_id}/validate

   レスポンス:
   {
     "valid": boolean,
     "errors": [
       { "step_id": "...", "field": "...", "message": "..." }
     ],
     "warnings": [
       { "step_id": "...", "message": "..." }
     ]
   }

   検証内容:
   - スタートブロック存在確認
   - 全ステップ接続確認
   - 必須設定フィールド確認
   - クレデンシャル設定確認
   - 循環参照チェック
```

**Frontend（Vue 3）**

```
3. TestTab の強化
   ファイル: frontend/components/workflow-editor/properties/TestTab.vue

   変更内容:
   - 「ドライラン」ボタン追加
   - 検証結果の表示（エラー/警告をステップごとに表示）
   - 結果からエラーステップへのジャンプ機能

4. ExecutionTab の強化
   ファイル: frontend/components/workflow-editor/ExecutionTab.vue

   変更内容:
   - 「ワークフロー検証」ボタン追加
   - 検証結果をリスト表示
   - 「すべて修正して再検証」フロー

5. useWorkflowValidation composable
   ファイル: frontend/composables/test/useWorkflowValidation.ts（新規）

   export function useWorkflowValidation() {
     const validating = ref(false)
     const validationResult = ref<ValidationResult | null>(null)

     const validate = async (projectId: string) => { ... }
     const jumpToError = (stepId: string) => { ... }
   }

6. 検証結果コンポーネント
   ファイル: frontend/components/workflow-editor/test/ValidationResultPanel.vue（新規）

   Props:
   - result: ValidationResult
   Emits:
   - 'jump-to-step': (stepId: string)
```

#### 見積もり工数
- Backend: 6時間
- Frontend: 5時間
- テスト: 3時間
- **合計: 14時間**

---

### 1.3 PublishChecklist の拡張

#### 実装タスク

```
1. チェック項目追加
   ファイル: frontend/components/workflow-editor/PublishChecklistModal.vue

   追加するチェック:
   - トリガー設定検証（manual以外の場合、設定が有効か）
   - Webhook URL の有効性（存在確認）
   - Schedule の Cron 式検証
   - 入力スキーマの設定確認

2. トリガー有効化確認ステップ
   リリース完了後に「トリガーを有効化しますか？」確認
   - 「はい」→ トリガー有効化 API 呼び出し
   - 「あとで」→ 無効のままリリース

3. Backend 検証 API 統合
   PublishChecklist で validate API を呼び出し
   結果をチェックリストに反映
```

#### 見積もり工数
- Frontend: 4時間
- テスト: 1時間
- **合計: 5時間**

---

## Phase 2: Copilot 品質向上

### 2.1 ブロック設定自動生成

#### 背景
現在の Copilot はステップタイプ（slack, google_sheets 等）のマッピングのみ。
具体的な config（channel_id, message_template 等）は空のまま。

#### 実装タスク

**Backend（Go）**

```
1. ブロック設定スキーマの活用
   ファイル: backend/internal/usecase/copilot.go

   mapStepsToBlocks() 改修:
   - BlockDefinition.ConfigSchema を取得
   - LLM に「このスキーマに基づいて config を生成」指示
   - 生成された config をスキーマ検証

2. プロンプト拡張
   ファイル: backend/internal/usecase/copilot_prompt.go

   ステップ生成時のプロンプトに追加:
   ```
   ブロック定義:
   - block_slug: slack
   - config_schema: { channel: string (required), message: string (required) }

   ユーザーの要件に基づいて、具体的な config 値を提案してください。
   不明な場合は placeholder（{{user_input}} 等）を使用してください。
   ```

3. Config 検証 API
   ファイル: backend/internal/handler/step.go

   POST /api/v1/blocks/{block_slug}/validate-config

   リクエスト: { "config": { ... } }
   レスポンス: {
     "valid": boolean,
     "errors": [{ "field": "...", "message": "..." }]
   }
```

**Frontend（Vue 3）**

```
4. Copilot 生成結果の表示改善
   ファイル: frontend/components/workflow-editor/copilot/ProposalPreview.vue（既存改修）

   - 生成された config をプレビュー表示
   - 不完全なフィールドをハイライト
   - 「設定を編集」ボタンで PropertiesPanel にジャンプ

5. 不完全設定の警告表示
   ファイル: frontend/components/dag-editor/DagEditor.vue

   - 設定不完全なノードに警告アイコン表示
   - ホバーで不足フィールド一覧
```

#### 見積もり工数
- Backend: 8時間
- Frontend: 4時間
- テスト: 3時間
- **合計: 15時間**

---

### 2.2 生成後の設定ウィザード

#### 背景
Copilot がワークフローを生成した後、ユーザーは手動で各ステップの設定を確認・補完する必要がある。
この作業を対話的にガイドする。

#### 実装タスク

```
1. SetupWizard コンポーネント
   ファイル: frontend/components/workflow-editor/copilot/SetupWizard.vue（新規）

   フロー:
   Step 1: トリガー設定確認
     - トリガータイプ確認
     - 必要な設定入力（Webhook secret, Cron 式等）

   Step 2: 認証情報設定
     - 必要なクレデンシャル一覧
     - 未設定のものは OAuth 接続ボタン表示

   Step 3: ステップ設定確認
     - 不完全な設定があるステップを順番に表示
     - 各ステップの必須フィールドを入力

   Step 4: テスト実行
     - ドライラン実行
     - 結果確認

   Step 5: 完了
     - 「リリースする」または「あとで編集」

2. useCopilotSetup composable
   ファイル: frontend/composables/copilot/useCopilotSetup.ts（新規）

   export function useCopilotSetup() {
     const currentStep = ref(1)
     const incompleteSteps = ref<Step[]>([])
     const missingCredentials = ref<string[]>([])

     const analyzeWorkflow = async () => { ... }
     const goToStep = (n: number) => { ... }
     const completeSetup = async () => { ... }
   }

3. Copilot 完了時のフック
   ファイル: frontend/components/workflow-editor/CopilotTab.vue

   ワークフロー生成完了時に SetupWizard をモーダル表示
```

#### 見積もり工数
- Frontend: 10時間
- テスト: 2時間
- **合計: 12時間**

---

### 2.3 テスト実行の自動提案

#### 実装タスク

```
1. 生成完了時のプロンプト
   ファイル: frontend/components/workflow-editor/copilot/SetupWizard.vue

   最終ステップで:
   - 「テスト実行を行いますか？」確認
   - サンプル入力の自動生成（入力スキーマから）
   - ワンクリックでテスト実行

2. サンプル入力生成
   ファイル: backend/internal/usecase/step.go

   func (u *StepUsecase) GenerateSampleInput(ctx, stepID uuid.UUID) (json.RawMessage, error)

   - InputSchema から型に基づいたサンプル値生成
   - string → "sample_string"
   - number → 123
   - boolean → true
   - array → [sample_item]

3. Frontend 統合
   - LLM に「このワークフローをテストするためのサンプル入力を生成して」と依頼
   - 生成されたサンプルをプリセットとして使用可能に
```

#### 見積もり工数
- Backend: 3時間
- Frontend: 3時間
- **合計: 6時間**

---

## Phase 3: UX 改善

### 3.1 オンボーディング強化

#### 背景
WelcomeDialog でスキップした後、ユーザーが何をすべきか分からない。

#### 実装タスク

```
1. WelcomeDialog の改修
   ファイル: frontend/components/workflow-editor/WelcomeDialog.vue

   選択肢:
   A. Copilot で作成（推奨）
      → Copilot サイドバーを開く

   B. テンプレートから作成
      → テンプレートギャラリーを開く

   C. 空のワークフローから作成
      → チュートリアルツアー開始

2. チュートリアルツアー
   ファイル: frontend/components/workflow-editor/TutorialTour.vue（新規）

   ステップ:
   1. 「ここからブロックを追加」（StepPalette にハイライト）
   2. 「スタートブロックを選択」（Start タブを示す）
   3. 「トリガーを設定」（TriggerConfigPanel を示す）
   4. 「ブロックを接続」（Edge 作成を示す）
   5. 「テスト実行」（TestTab を示す）
   6. 「リリース」（Release ボタンを示す）

   ライブラリ: vue-tour または custom 実装

3. コンテキストヘルプ
   各パネルに「?」アイコン追加
   クリックで該当機能の説明をポップオーバー表示
```

#### 見積もり工数
- Frontend: 8時間
- テスト: 2時間
- **合計: 10時間**

---

### 3.2 トリガーダッシュボード

#### 背景
運用中のワークフローのトリガー状態を一覧で確認・管理できない。

#### 実装タスク

**Backend（Go）**

```
1. トリガー状態取得 API
   ファイル: backend/internal/handler/trigger.go（新規）

   GET /api/v1/projects/{project_id}/triggers

   レスポンス:
   {
     "triggers": [
       {
         "step_id": "...",
         "step_name": "Start",
         "trigger_type": "webhook",
         "enabled": true,
         "webhook_url": "https://...",
         "last_triggered_at": "2024-01-01T00:00:00Z",
         "trigger_count_24h": 15
       },
       {
         "step_id": "...",
         "step_name": "Daily Report",
         "trigger_type": "schedule",
         "enabled": true,
         "cron_expression": "0 9 * * *",
         "timezone": "Asia/Tokyo",
         "next_run_at": "2024-01-02T09:00:00+09:00",
         "last_run_at": "2024-01-01T09:00:00+09:00"
       }
     ]
   }

2. Usecase 層
   ファイル: backend/internal/usecase/trigger.go（新規）

   type TriggerUsecase struct { ... }
   func (u *TriggerUsecase) ListByProject(ctx, tenantID, projectID uuid.UUID) ([]TriggerStatus, error)
```

**Frontend（Vue 3）**

```
3. TriggerDashboard コンポーネント
   ファイル: frontend/components/workflow-editor/TriggerDashboard.vue（新規）

   表示内容:
   - トリガー一覧テーブル
     - ステップ名
     - タイプ（アイコン付き）
     - 有効/無効トグル
     - 詳細（Webhook URL, 次回実行等）
     - 24時間のトリガー回数
   - 各行にクイックアクション
     - 「テスト」ボタン
     - 「設定を編集」リンク
     - 「ログを見る」リンク

4. ダッシュボードタブ追加
   ファイル: frontend/components/workflow-editor/ExecutionTab.vue

   タブ追加:
   - 実行履歴（既存）
   - トリガー管理（新規）

5. useTriggers composable
   ファイル: frontend/composables/useTriggers.ts（新規）

   export function useTriggers() {
     const triggers = ref<TriggerStatus[]>([])
     const loading = ref(false)

     const list = async (projectId: string) => { ... }
     const enable = async (projectId: string, stepId: string) => { ... }
     const disable = async (projectId: string, stepId: string) => { ... }
     const testWebhook = async (projectId: string, stepId: string) => { ... }
   }
```

#### 見積もり工数
- Backend: 5時間
- Frontend: 8時間
- テスト: 2時間
- **合計: 15時間**

---

### 3.3 エラーハンドリング改善

#### 背景
実行失敗時の通知がなく、ユーザーが問題に気づけない。

#### 実装タスク

**Backend（Go）**

```
1. 実行失敗通知
   ファイル: backend/internal/usecase/run.go

   Run 完了時（status=failed）に通知:
   - プロジェクト設定で通知先を設定可能
   - 通知チャネル: Email, Slack, Webhook

2. エラーログ詳細化
   ファイル: backend/internal/domain/step_run.go

   StepRun.Error 構造の拡張:
   {
     "code": "CREDENTIAL_EXPIRED",
     "message": "Slack認証が期限切れです",
     "details": { ... },
     "suggestion": "Slack再認証を行ってください",
     "doc_url": "https://..."
   }

3. エラー通知設定 API
   POST /api/v1/projects/{project_id}/notification-settings
   {
     "on_failure": {
       "email": ["admin@example.com"],
       "slack_webhook": "https://hooks.slack.com/..."
     }
   }
```

**Frontend（Vue 3）**

```
4. エラー表示の改善
   ファイル: frontend/components/workflow-editor/execution/StepRunDetail.vue（既存改修）

   - エラーコード別のアイコン
   - 推奨アクションの表示
   - ドキュメントへのリンク
   - 「再試行」ボタン

5. 通知設定画面
   ファイル: frontend/components/workflow-editor/settings/NotificationSettings.vue（新規）

   設定項目:
   - 実行失敗時の通知先（Email）
   - Slack 通知（Webhook URL）
   - 通知条件（全失敗 / 連続失敗のみ）

6. リアルタイム通知（オプション）
   - WebSocket 接続で実行状態をリアルタイム更新
   - 失敗時にトースト通知
```

#### 見積もり工数
- Backend: 8時間
- Frontend: 6時間
- テスト: 2時間
- **合計: 16時間**

---

## 実装スケジュール

### Phase 1（Week 1-2）: 致命的問題の解決
| タスク | 工数 | 担当 | 依存 |
|--------|------|------|------|
| 1.1 トリガー有効化機制 | 9h | - | なし |
| 1.2 テスト実行機能強化 | 14h | - | なし |
| 1.3 PublishChecklist拡張 | 5h | - | 1.1, 1.2 |
| **小計** | **28h** | | |

### Phase 2（Week 3-4）: Copilot 品質向上
| タスク | 工数 | 担当 | 依存 |
|--------|------|------|------|
| 2.1 ブロック設定自動生成 | 15h | - | なし |
| 2.2 生成後の設定ウィザード | 12h | - | 2.1, 1.2 |
| 2.3 テスト実行の自動提案 | 6h | - | 2.2 |
| **小計** | **33h** | | |

### Phase 3（Week 5-6）: UX 改善
| タスク | 工数 | 担当 | 依存 |
|--------|------|------|------|
| 3.1 オンボーディング強化 | 10h | - | なし |
| 3.2 トリガーダッシュボード | 15h | - | 1.1 |
| 3.3 エラーハンドリング改善 | 16h | - | なし |
| **小計** | **41h** | | |

### 総工数
- **Phase 1**: 28時間（約3.5人日）
- **Phase 2**: 33時間（約4人日）
- **Phase 3**: 41時間（約5人日）
- **合計**: 102時間（約13人日）

---

## ファイル変更一覧

### Backend（新規）
- `backend/internal/handler/trigger.go`
- `backend/internal/usecase/trigger.go`

### Backend（修正）
- `backend/internal/handler/step.go` - トリガー有効化 API
- `backend/internal/handler/run.go` - ドライラン、検証 API
- `backend/internal/usecase/step.go` - EnableTrigger, DisableTrigger
- `backend/internal/usecase/run.go` - DryRun モード
- `backend/internal/usecase/copilot.go` - config 自動生成
- `backend/internal/usecase/copilot_prompt.go` - プロンプト拡張
- `backend/internal/domain/step.go` - エラー構造拡張

### Frontend（新規）
- `frontend/composables/useTriggers.ts`
- `frontend/composables/useSteps.ts`（または既存に追加）
- `frontend/composables/test/useWorkflowValidation.ts`
- `frontend/composables/copilot/useCopilotSetup.ts`
- `frontend/components/workflow-editor/TriggerDashboard.vue`
- `frontend/components/workflow-editor/TutorialTour.vue`
- `frontend/components/workflow-editor/copilot/SetupWizard.vue`
- `frontend/components/workflow-editor/test/ValidationResultPanel.vue`
- `frontend/components/workflow-editor/settings/NotificationSettings.vue`

### Frontend（修正）
- `frontend/types/api.ts` - 型定義追加
- `frontend/components/workflow-editor/TriggerConfigPanel.vue` - 有効化トグル
- `frontend/components/workflow-editor/PropertiesPanel.vue` - 統合
- `frontend/components/workflow-editor/properties/TestTab.vue` - ドライラン
- `frontend/components/workflow-editor/ExecutionTab.vue` - タブ追加
- `frontend/components/workflow-editor/PublishChecklistModal.vue` - チェック追加
- `frontend/components/workflow-editor/WelcomeDialog.vue` - 選択肢改善
- `frontend/components/workflow-editor/CopilotTab.vue` - ウィザード連携
- `frontend/components/dag-editor/DagEditor.vue` - 警告表示

---

## 成功指標

### 定量指標
| 指標 | 現状 | 目標 |
|------|------|------|
| Copilot でのワークフロー完成率 | 不明 | 80%以上 |
| 生成からリリースまでの平均時間 | 不明 | 10分以内 |
| リリース後のトリガー有効化率 | 0% | 90%以上 |
| 初回テスト実行成功率 | 不明 | 70%以上 |

### 定性指標
- ユーザーが「次に何をすべきか」迷わない
- Copilot 生成後の手動調整が最小限
- エラー発生時に原因と対処法が分かる

---

## リスクと対策

| リスク | 影響 | 対策 |
|--------|------|------|
| LLM config 生成の精度不足 | 手動調整が必要 | スキーマ検証 + 設定ウィザードでカバー |
| トリガー有効化の混乱 | 意図せず本番実行 | 明示的な有効化フロー + 確認ダイアログ |
| ドライランの実行コスト | 外部サービス課金 | モック応答 + 制限回数設定 |
| オンボーディングの煩わしさ | スキップされる | スキップ可能 + 再表示オプション |
