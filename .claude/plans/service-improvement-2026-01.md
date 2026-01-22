# サービスブラッシュアップ計画（2026年1月）

## コアバリュー
**「ユーザーが任意のワークフローを作ろうとしたときに、Copilotが自動でそのワークフローを作成し、すぐに運用に載せられる」**

---

## 現状評価（2026-01-22 更新）

### 実装完成度

| フェーズ | 機能 | 完成度 | 状態 |
|---------|------|--------|------|
| **作成** | Copilot会話UI | 95% | ✅ |
| **作成** | SSEストリーミング | 95% | ✅ |
| **作成** | ツール実行（15種） | 90% | ✅ |
| **作成** | 提案表示・適用 | 90% | ✅ |
| **作成** | WelcomeDialog（初回導線） | 100% | ✅ 実装済 |
| **運用** | 公開前チェックリスト | 100% | ✅ 実装済 |
| **運用** | ValidateForPublish API | 100% | ✅ 実装済 |
| **運用** | QuickDeploy（ワンクリック運用） | 0% | ❌ 未実装 |
| **運用** | トリガー自動設定 | 0% | ❌ 未実装 |
| **運用** | クレデンシャル自動検出 | 0% | ❌ 未実装 |
| **運用** | テスト実行統合 | 50% | ⚠️ 基盤のみ |
| **運用** | エラー修正サイクル | 0% | ❌ 未実装 |

### 総合評価
- **ワークフロー作成**: 95%完成 ✅
- **ワークフロー運用開始**: 50%実装 ⚠️

---

## 実装済み機能詳細

### 1. WelcomeDialog（Copilot導線）✅

**ファイル**: `frontend/components/workflow-editor/WelcomeDialog.vue`

```
新規プロジェクト作成
    ↓
WelcomeDialogがポップアップ表示
┌─────────────────────────────────────────────────────────────────┐
│    何を自動化しますか？                                          │
│    ────────────────────────────────────────────────             │
│    [毎朝9時にSlackで天気予報を通知したい           ]  →          │
│                                                                 │
│    例: • GitHubのIssueをSlackに通知                              │
│        • 定期的にAPIからデータ取得してNotionに保存                │
│                                                                 │
│    または [空のキャンバスから始める]                              │
└─────────────────────────────────────────────────────────────────┘
    ↓
プロンプト送信 → Copilotサイドバーが開く → ワークフロー生成開始
```

**i18nキー**: `welcomeDialog.*`

### 2. 公開前チェックリスト ✅

**ファイル**: `frontend/components/editor/ReleaseModal.vue`

```
┌─────────────────────────────────────────────────────────────────┐
│  ✅ スタートブロックが存在する                                   │
│  ✅ すべてのブロックが接続されている                              │
│  ✅ 無限ループの可能性なし                                       │
│  ⚠️ 必要なクレデンシャルが設定されている                         │
│                                                                 │
│  [問題を修正] [警告を無視して公開]                                │
└─────────────────────────────────────────────────────────────────┘
```

**バックエンド**: `backend/internal/usecase/project.go:792` - `ValidateForPublish()`

**i18nキー**: `publishChecklist.*`

### 3. Copilot Composables（リファクタリング）✅

**ディレクトリ**: `frontend/composables/copilot/`

| ファイル | 役割 |
|----------|------|
| `useChatScroll.ts` | チャットスクロール制御 |
| `useProposalStatuses.ts` | 提案ステータス管理 |
| `useToolDisplay.ts` | ツール表示ラベル・説明 |
| `toolConverters.ts` | ツール呼び出し→変更変換 |
| `index.ts` | エクスポート |

### 4. Execution Composables ✅

**ディレクトリ**: `frontend/composables/execution/`

| ファイル | 役割 |
|----------|------|
| `useWorkflowExecution.ts` | ワークフロー実行・ポーリング |
| `useExecutionInput.ts` | 実行入力管理 |
| `useStepNavigation.ts` | ステップナビゲーション |
| `useSchemaFields.ts` | スキーマベースフィールド |
| `index.ts` | エクスポート |

### 5. トリガーブロック正規化 ✅

**ファイル**: `backend/internal/usecase/step.go`, `backend/internal/domain/step.go`

```go
// トリガーブロック（manual_trigger, schedule_trigger, webhook_trigger）を
// "start" タイプに正規化し、TriggerTypeフィールドに種別を保存
if domain.IsTriggerBlockSlug(string(input.Type)) {
    stepType = domain.StepTypeStart
    input.TriggerType = string(domain.GetTriggerTypeFromSlug(string(input.Type)))
}
```

### 6. Domain Tests ✅

**新規テストファイル**:
- `backend/internal/domain/step_test.go`
- `backend/internal/domain/project_test.go`
- `backend/internal/domain/block_group_test.go`
- 他10ファイル

**テスト数**: 391テスト（全て成功）

---

## 残り実装項目

### Phase 1: P0（即時対応）

#### 1.1 QuickDeploy（ワンクリック運用開始）

**目的**: Copilot作成後、1クリックで運用開始

**実装内容**:
```
Copilot: 「ワークフローを作成しました」
    ↓
[運用開始] ボタン表示
    ↓
クリック → 以下を自動実行:
  1. クレデンシャル要件チェック
  2. 未設定なら OAuth 誘導ダイアログ
  3. トリガー設定の自動補完（CRON生成等）
  4. ワークフロー自動Publish
  5. テスト実行（オプション）
```

**ファイル変更**:
- `frontend/components/workflow-editor/CopilotProposalCard.vue` - 「運用開始」ボタン追加
- `backend/internal/usecase/copilot.go` - `QuickDeploy()` メソッド追加
- `backend/internal/handler/copilot_agent.go` - エンドポイント追加

**API設計**:
```go
// POST /api/v1/workflows/{id}/quick-deploy
type QuickDeployRequest struct {
    TestInput map[string]interface{} `json:"test_input,omitempty"`
    SkipTest  bool                   `json:"skip_test,omitempty"`
}

type QuickDeployResponse struct {
    Success            bool                    `json:"success"`
    WorkflowStatus     string                  `json:"workflow_status"`
    MissingCredentials []CredentialRequirement `json:"missing_credentials,omitempty"`
    TriggerConfigured  bool                    `json:"trigger_configured"`
    TestRunID          *string                 `json:"test_run_id,omitempty"`
    Warnings           []string                `json:"warnings,omitempty"`
}
```

#### 1.2 トリガー自動設定

**目的**: 「毎日9時」などの自然言語からCRON式を自動生成

**実装内容**:
- Copilotがヒアリング中にトリガー情報を抽出
- 自然言語→CRON式変換（LLM使用）
- `trigger_config.cron_expression` に自動設定

**ファイル変更**:
- `backend/internal/usecase/copilot.go` - `AutoConfigureTrigger()` 追加
- `frontend/components/workflow-editor/TriggerConfigPanel.vue` - 自然言語入力対応

---

### Phase 2: P1（1-2週間）

#### 2.1 クレデンシャル自動検出

**目的**: ワークフロー内で必要なクレデンシャルを自動検出

**実装内容**:
```go
// copilot.go
func (u *CopilotUsecase) DetectCredentialRequirements(
    ctx context.Context,
    workflowID uuid.UUID,
) ([]CredentialRequirement, error) {
    // ステップを走査
    // BlockDefinitionのcredential_schemaを確認
    // 必要なクレデンシャルをリスト化
}

type CredentialRequirement struct {
    BlockSlug      string `json:"block_slug"`
    StepID         string `json:"step_id"`
    StepName       string `json:"step_name"`
    CredentialType string `json:"credential_type"`  // "oauth", "api_key"
    ServiceName    string `json:"service_name"`     // "Slack", "Discord"
    IsConfigured   bool   `json:"is_configured"`
    OAuthURL       string `json:"oauth_url,omitempty"`
}
```

#### 2.2 OAuth誘導フロー

**目的**: 未接続のサービスへのOAuth接続を誘導

**実装内容**:
- QuickDeploy時に未設定クレデンシャルを検出
- モーダルでOAuth接続を促す
- 接続完了後に自動binding

**ファイル変更**:
- `frontend/components/workflow-editor/CredentialSetupModal.vue` - 新規作成
- `backend/internal/usecase/credential.go` - OAuth URL生成

#### 2.3 テスト実行統合

**目的**: ワークフロー作成後すぐにテスト実行可能に

**現状**: `useWorkflowExecution.ts` に実行機能は実装済み

**追加実装**:
- `CopilotProposalCard.vue` に「適用してテスト」ボタン追加
- テスト結果をCopilotチャットにフィードバック

---

### Phase 3: P2（2-3週間）

#### 3.1 エラー修正サイクル

**目的**: 実行エラー発生時にCopilotが自動修正提案

**実装内容**:
```
ワークフロー実行
    ↓
エラー発生
    ↓
Copilotに自動報告
    ↓
修正提案生成
    ↓
「この問題を修正」ボタン
    ↓
自動修正適用
```

**ファイル変更**:
- `frontend/components/workflow-editor/ExecutionTab.vue` - エラー時のCopilot連携
- `backend/internal/usecase/copilot.go` - `DiagnoseError()` 強化

---

## 優先度マトリックス

| 施策 | 効果 | 工数 | 優先度 | 状態 |
|------|------|------|--------|------|
| WelcomeDialog | 高 | 中 | P0 | ✅ 完了 |
| 公開前チェックリスト | 高 | 中 | P0 | ✅ 完了 |
| QuickDeploy | **高** | 中 | **P0** | ❌ 未実装 |
| トリガー自動設定 | **高** | 低 | **P0** | ❌ 未実装 |
| クレデンシャル自動検出 | 高 | 中 | P1 | ❌ 未実装 |
| OAuth誘導フロー | 中 | 中 | P1 | ❌ 未実装 |
| テスト実行統合 | 高 | 低 | P1 | ⚠️ 基盤のみ |
| エラー修正サイクル | 中 | 高 | P2 | ❌ 未実装 |

---

## 実装ロードマップ

```
✅ Week 0（完了）:
├── WelcomeDialog実装
├── PublishChecklist実装
├── ValidateForPublish API
├── Composablesリファクタリング
└── Domain Tests追加

Week 1-2: Phase 1（P0）
├── QuickDeploy API実装
├── QuickDeployボタンUI
├── トリガー自動設定（CRON生成）
└── 統合テスト

Week 3-4: Phase 2（P1）
├── クレデンシャル要件検出
├── CredentialSetupModal
├── OAuth誘導フロー
└── テスト実行統合（「適用してテスト」）

Week 5-6: Phase 3（P2）
├── エラー自動報告
├── Copilot診断強化
└── 修正サイクルUI
```

---

## 成功指標

| 指標 | 以前 | 現在 | 目標 |
|------|------|------|------|
| 作成→運用開始のクリック数 | 5-10回 | 3-5回 | **1回** |
| 作成→運用開始の時間 | 10-30分 | 5-10分 | **1分以内** |
| 手動設定項目数 | 3-5項目 | 1-2項目 | **0項目** |
| 初回テスト成功率 | 測定不能 | 測定不能 | **80%以上** |

---

## Critical Files（残り実装）

| ファイル | 変更内容 | 優先度 |
|----------|----------|--------|
| `backend/internal/usecase/copilot.go` | QuickDeploy, AutoConfigureTrigger追加 | P0 |
| `backend/internal/handler/project.go` | QuickDeployエンドポイント追加 | P0 |
| `frontend/components/workflow-editor/CopilotProposalCard.vue` | 運用開始ボタン追加 | P0 |
| `frontend/components/workflow-editor/CredentialSetupModal.vue` | 新規作成 | P1 |
| `frontend/components/workflow-editor/ExecutionTab.vue` | Copilotエラー連携 | P2 |

---

## 次のアクション

1. **QuickDeploy API設計・実装**
   - バックエンド: `copilot.go` に `QuickDeploy()` メソッド追加
   - フロントエンド: CopilotProposalCardに「運用開始」ボタン追加

2. **トリガー自動設定**
   - 自然言語→CRON変換ロジック実装
   - TriggerConfigPanelへの統合
