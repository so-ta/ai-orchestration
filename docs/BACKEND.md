# バックエンド技術リファレンス

Goバックエンドのコード構造、インターフェース、パターン。

> **移行メモ (2026-01)**: WorkflowはProjectに名称変更されました。Projectは異なるトリガータイプ（manual、schedule、webhook）を持つ複数のStartブロックをサポートするようになりました。webhooksテーブルは削除され、Webhook設定はStartブロックのconfigの一部になりました。

## クイックリファレンス

| 項目 | 値 |
|------|-------|
| 言語 | Go 1.22+ |
| アーキテクチャ | クリーンアーキテクチャ (Handler → Usecase → Domain → Repository) |
| エントリーポイント | `cmd/api/main.go`, `cmd/worker/main.go` |
| ドメインモデル | `internal/domain/` |
| APIハンドラー | `internal/handler/` |
| ビジネスロジック | `internal/usecase/` |
| データベース | `internal/repository/postgres/` |
| 外部API | `internal/adapter/` |
| DAGエンジン | `internal/engine/` |

## ディレクトリ構造

```
backend/
├── cmd/
│   ├── api/main.go         # HTTPサーバー、ルーティング、ミドルウェア設定
│   └── worker/main.go      # ジョブコンシューマー、DAGエグゼキューター
├── internal/
│   ├── domain/             # エンティティ、ビジネスルール
│   ├── usecase/            # アプリケーションロジック
│   ├── handler/            # HTTPハンドラー
│   ├── repository/         # データアクセス
│   │   └── postgres/       # PostgreSQL実装
│   ├── adapter/            # 外部サービス連携
│   ├── engine/             # DAG実行エンジン
│   └── middleware/         # HTTPミドルウェア
├── pkg/
│   ├── database/           # DB接続プール
│   ├── redis/              # Redisクライアントラッパー
│   └── telemetry/          # OpenTelemetry SDK
├── migrations/             # SQLマイグレーション
└── tests/e2e/              # 統合テスト
```

## レイヤー依存関係

```
handler -> usecase -> domain
                  -> repository
                  -> adapter
                  -> engine
```

## ドメインモデル

### Project (domain/project.go、旧workflow.go)

```go
type Project struct {
    ID          uuid.UUID
    TenantID    uuid.UUID
    Name        string
    Description string
    Status      ProjectStatus  // "draft" | "published"
    Version     int
    Variables   json.RawMessage  // プロジェクトレベル変数（input_schema/output_schemaを置換）
    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   *time.Time
}

type ProjectStatus string
const (
    ProjectStatusDraft     ProjectStatus = "draft"
    ProjectStatusPublished ProjectStatus = "published"
)
```

> **移行メモ**: `InputSchema`と`OutputSchema`はProjectから削除されました。入出力スキーマはStepのconfigでStartブロックごとに定義されるようになりました。

### Step (domain/step.go)

```go
type Step struct {
    ID        uuid.UUID
    ProjectID uuid.UUID
    Name      string
    Type      StepType
    Config    json.RawMessage
    Position  Position
    CreatedAt time.Time
    UpdatedAt time.Time
}

type StepType string
const (
    StepTypeStart       StepType = "start"        // プロジェクトごとに複数可、trigger_type付き
    StepTypeLLM         StepType = "llm"
    StepTypeTool        StepType = "tool"
    StepTypeCondition   StepType = "condition"
    StepTypeMap         StepType = "map"
    StepTypeJoin        StepType = "join"
    StepTypeSubflow     StepType = "subflow"
    StepTypeLoop        StepType = "loop"
    StepTypeWait        StepType = "wait"
    StepTypeFunction    StepType = "function"
    StepTypeRouter      StepType = "router"
    StepTypeHumanInLoop StepType = "human_in_loop"
)
```

### Step Config スキーマ

#### Start Step（プロジェクトごとに複数サポート）
```json
{
  "trigger_type": "manual|schedule|webhook",
  "trigger_config": {
    "input_mapping": {},
    "webhook_secret": "string",
    "cron": "0 9 * * *",
    "timezone": "Asia/Tokyo"
  },
  "input_schema": {},
  "output_schema": {}
}
```

| トリガータイプ | trigger_configフィールド |
|--------------|----------------------|
| `manual` | 不要 |
| `schedule` | `cron`, `timezone` |
| `webhook` | `webhook_secret`, `input_mapping` |

> **注意**: プロジェクトは複数のStartブロックを持つことができます。各Startブロックは異なるトリガータイプを持つことができます。これは以前のwebhooksテーブルの機能を置き換えます。

#### LLM Step
```json
{
  "provider": "openai|anthropic",
  "model": "gpt-4|claude-3-opus-20240229",
  "prompt": "{{input.field}} を含むテンプレート",
  "temperature": 0.7,
  "max_tokens": 1000
}
```

#### Tool Step
```json
{
  "adapter_id": "mock|http|openai|anthropic",
  "...アダプター固有フィールド"
}
```

#### Condition Step
```json
{
  "expression": "$.field > 10"
}
```

#### Map Step
```json
{
  "input_path": "$.items",
  "parallel": true,
  "max_concurrency": 5
}
```

#### Loop Step
```json
{
  "loop_type": "for|forEach|while|doWhile",
  "count": 10,                    // for: 反復回数
  "input_path": "$.items",        // forEach: 配列へのパス
  "condition": "$.index < 10",    // while/doWhile: 条件式
  "max_iterations": 100,          // 安全制限（デフォルト: 100）
  "adapter_id": "mock"            // オプション: 各反復で実行するアダプター
}
```

ループタイプ:
- `for`: 固定回数の反復
- `forEach`: 配列要素の反復
- `while`: 条件がtrueの間継続（実行前にチェック）
- `doWhile`: 最低1回実行し、その後条件をチェック

#### Wait Step
```json
{
  "duration_ms": 5000,            // ミリ秒単位の遅延
  "until": "2024-01-15T10:00:00Z" // または ISO8601 日時まで待機
}
```

| 制約 | 値 |
|------------|-------|
| 最大待機時間 | 1時間 (3600000 ms) |

#### Function Step
```json
{
  "code": "return input.value * 2",
  "language": "javascript",       // 現在はjavascriptのみ
  "timeout_ms": 5000              // 実行タイムアウト
}
```

| ステータス | 説明 |
|--------|-------------|
| 実装状態 | 部分的 - 警告付きで入力をパススルー |

#### Router Step
```json
{
  "routes": [
    {"name": "support", "description": "カスタマーサポートリクエスト"},
    {"name": "sales", "description": "営業問い合わせ"}
  ],
  "provider": "openai|anthropic", // 分類用LLMプロバイダー
  "model": "gpt-4",               // ルーティング判断用モデル
  "prompt": "この入力を分類してください" // オプションのカスタムプロンプト
}
```

| 動作 | 説明 |
|----------|-------------|
| ルーティング | LLMを使用して入力を分類し、適切なルートを選択 |

#### Human-in-Loop Step
```json
{
  "instructions": "確認して承認してください",
  "timeout_hours": 24,
  "approval_url": true,           // 承認URLを生成
  "notification": {
    "type": "email|slack|webhook",
    "target": "user@example.com"
  },
  "required_fields": [
    {"name": "approved", "type": "boolean", "required": true},
    {"name": "comment", "type": "string", "required": false}
  ]
}
```

| モード | 動作 |
|------|----------|
| Test | 自動承認 |
| Production | 承認受信までワークフロー一時停止 |

### Edge (domain/edge.go)

```go
type Edge struct {
    ID           uuid.UUID
    ProjectID    uuid.UUID
    SourceStepID uuid.UUID
    TargetStepID uuid.UUID
    Condition    string  // オプション: "$.success == true"
    CreatedAt    time.Time
}
```

### BlockGroup (domain/block_group.go)

複数のステップを単一の論理単位にグループ化する制御フロー構造。

```go
type BlockGroup struct {
    ID            uuid.UUID
    ProjectID     uuid.UUID
    Name          string
    Type          BlockGroupType
    Config        json.RawMessage
    ParentGroupID *uuid.UUID      // ネストされたグループ用
    PositionX     int
    PositionY     int
    Width         int
    Height        int
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

type BlockGroupType string
const (
    BlockGroupTypeParallel   BlockGroupType = "parallel"    // 並列実行
    BlockGroupTypeTryCatch   BlockGroupType = "try_catch"   // エラーハンドリング
    BlockGroupTypeIfElse     BlockGroupType = "if_else"     // 条件分岐
    BlockGroupTypeSwitchCase BlockGroupType = "switch_case" // 多分岐ルーティング
    BlockGroupTypeForeach    BlockGroupType = "foreach"     // 配列反復
    BlockGroupTypeWhile      BlockGroupType = "while"       // 条件ループ
)
```

#### BlockGroup Config例

```json
// parallel
{ "max_concurrent": 10, "fail_fast": true }

// try_catch
{ "error_types": ["*"], "retry_count": 3 }

// if_else
{ "condition": "$.status == 'active'" }

// foreach
{ "input_path": "$.items", "parallel": true, "max_workers": 5 }

// while
{ "condition": "$.count < 10", "max_iterations": 100 }
```

#### Step グループロール

BlockGroup内のステップは`group_role`フィールドを持ちます:

| ロール | ブロックタイプ | 説明 |
|------|-----------|-------------|
| `body` | parallel, foreach, while | メイン実行ステップ |
| `try` | try_catch | tryブロックステップ |
| `catch` | try_catch | エラーハンドリングステップ |
| `finally` | try_catch | クリーンアップステップ |
| `then` | if_else | trueブランチステップ |
| `else` | if_else | falseブランチステップ |
| `case_N` | switch_case | caseブランチステップ |
| `default` | switch_case | defaultブランチステップ |

### BlockDefinition (domain/block.go)

ブロック定義は継承・拡張可能な再利用可能な実行単位を表します。

```go
type BlockDefinition struct {
    ID            uuid.UUID
    TenantID      *uuid.UUID       // システムブロックはnil
    Slug          string           // 一意識別子
    Name          string
    Description   string
    Category      BlockCategory
    Icon          string
    ConfigSchema  json.RawMessage  // config用JSONスキーマ
    InputSchema   json.RawMessage  // 入力用JSONスキーマ
    OutputSchema  json.RawMessage  // 出力用JSONスキーマ
    OutputPorts   []OutputPort
    ErrorCodes    []ErrorCodeDef

    // 統一ブロックモデル
    Code          string           // サンドボックスで実行されるJavaScriptコード
    UIConfig      json.RawMessage  // UIメタデータ（アイコン、色など）
    IsSystem      bool             // システムブロックは管理者のみ編集可能
    Version       int              // バージョン番号

    // ブロック継承/拡張
    ParentBlockID *uuid.UUID       // 親ブロックへの参照
    ConfigDefaults json.RawMessage // 親のconfigのデフォルト値
    PreProcess    string           // 入力変換用JavaScript
    PostProcess   string           // 出力変換用JavaScript
    InternalSteps []InternalStep   // コンポジットブロック内部ステップ

    // 解決済みフィールド（バックエンドで設定）
    PreProcessChain        []string         // preProcessのチェーン（子→ルート）
    PostProcessChain       []string         // postProcessのチェーン（ルート→子）
    ResolvedCode           string           // ルート祖先からのコード
    ResolvedConfigDefaults json.RawMessage  // マージされたconfigデフォルト

    Enabled    bool
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

type InternalStep struct {
    Type      string          `json:"type"`       // 実行するブロックslug
    Config    json.RawMessage `json:"config"`     // ステップ設定
    OutputKey string          `json:"output_key"` // 出力格納用キー
}

type BlockCategory string
const (
    BlockCategoryAI          BlockCategory = "ai"
    BlockCategoryLogic       BlockCategory = "logic"
    BlockCategoryIntegration BlockCategory = "integration"
    BlockCategoryData        BlockCategory = "data"
    BlockCategoryControl     BlockCategory = "control"
    BlockCategoryUtility     BlockCategory = "utility"
)
```

#### ブロック継承制約

| 制約 | 値 |
|------------|-------|
| コードを持つブロックのみ継承可能 | `Code != ""` |
| 最大継承深度 | 50レベル（実用上は4-5レベル） |
| 循環継承 | 禁止（トポロジカルソートで検出） |
| テナント分離 | 同一テナント内またはシステムブロックからのみ継承可能 |

#### ブロック実行フロー

継承されたブロックを実行する際:
1. **PreProcessチェーン** (子 → ルート): 各preProcessで入力を変換
2. **内部ステップ** (ある場合): 内部ステップを順次実行
3. **コード実行**: ルート祖先からの解決済みコードを実行
4. **PostProcessチェーン** (ルート → 子): 各postProcessで出力を変換

### Run (domain/run.go)

```go
type Run struct {
    ID             uuid.UUID
    ProjectID      uuid.UUID
    ProjectVersion int
    StartStepID    uuid.UUID       // この実行をトリガーしたStartブロック
    TenantID       uuid.UUID
    Status         RunStatus
    Mode           RunMode
    TriggerType    TriggerType
    Input          json.RawMessage
    Output         json.RawMessage
    Error          string
    StartedAt      *time.Time
    CompletedAt    *time.Time
    CreatedAt      time.Time
}

type RunStatus string
const (
    RunStatusPending   RunStatus = "pending"
    RunStatusRunning   RunStatus = "running"
    RunStatusCompleted RunStatus = "completed"
    RunStatusFailed    RunStatus = "failed"
    RunStatusCancelled RunStatus = "cancelled"
)

type RunMode string
const (
    RunModeTest       RunMode = "test"
    RunModeProduction RunMode = "production"
)

type TriggerType string
const (
    TriggerTypeManual   TriggerType = "manual"
    TriggerTypeSchedule TriggerType = "schedule"
    TriggerTypeWebhook  TriggerType = "webhook"
)
```

### StepRun (domain/step_run.go)

```go
type StepRun struct {
    ID          uuid.UUID
    RunID       uuid.UUID
    StepID      uuid.UUID
    StepName    string
    Status      RunStatus
    Attempt     int
    Input       json.RawMessage
    Output      json.RawMessage
    Error       string
    StartedAt   *time.Time
    CompletedAt *time.Time
    DurationMS  int64
}
```

## インターフェース

### Repository インターフェース (repository/interfaces.go)

```go
type ProjectRepository interface {
    Create(ctx context.Context, p *domain.Project) error
    GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Project, error)
    List(ctx context.Context, tenantID uuid.UUID, filter ProjectFilter) ([]*domain.Project, error)
    Update(ctx context.Context, p *domain.Project) error
    Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

type StepRepository interface {
    Create(ctx context.Context, s *domain.Step) error
    GetByProjectID(ctx context.Context, projectID uuid.UUID) ([]*domain.Step, error)
    GetStartBlocks(ctx context.Context, projectID uuid.UUID) ([]*domain.Step, error)  // 全Startブロックを取得
    Update(ctx context.Context, s *domain.Step) error
    Delete(ctx context.Context, id uuid.UUID) error
}

type EdgeRepository interface {
    Create(ctx context.Context, e *domain.Edge) error
    GetByProjectID(ctx context.Context, projectID uuid.UUID) ([]*domain.Edge, error)
    Delete(ctx context.Context, id uuid.UUID) error
}

type RunRepository interface {
    Create(ctx context.Context, r *domain.Run) error
    GetByID(ctx context.Context, id uuid.UUID) (*domain.Run, error)
    Update(ctx context.Context, r *domain.Run) error
    ListByProjectID(ctx context.Context, projectID uuid.UUID) ([]*domain.Run, error)
    ListByStartStepID(ctx context.Context, startStepID uuid.UUID) ([]*domain.Run, error)  // Startブロックでフィルタ
}

type StepRunRepository interface {
    Create(ctx context.Context, sr *domain.StepRun) error
    Update(ctx context.Context, sr *domain.StepRun) error
    GetByRunID(ctx context.Context, runID uuid.UUID) ([]*domain.StepRun, error)
}
```

### Adapter インターフェース (adapter/adapter.go)

```go
type Adapter interface {
    ID() string
    Name() string
    Execute(ctx context.Context, req *Request) (*Response, error)
    InputSchema() json.RawMessage
    OutputSchema() json.RawMessage
}

type Request struct {
    Input      json.RawMessage
    Config     json.RawMessage
    SecretRefs map[string]string
}

type Response struct {
    Output   json.RawMessage
    Metadata ResponseMetadata
}

type ResponseMetadata struct {
    DurationMS   int64
    TokensUsed   int
    Cost         float64
    ProviderMeta json.RawMessage
}
```

## アダプター実装

### MockAdapter (adapter/mock.go)

設定:
```json
{
  "response": {"key": "value"},
  "delay_ms": 100,
  "error": "オプションのエラーメッセージ",
  "status_code": 200
}
```

### OpenAIAdapter (adapter/openai.go)

設定:
```json
{
  "model": "gpt-4",
  "messages": [{"role": "user", "content": "..."}],
  "temperature": 0.7,
  "max_tokens": 1000
}
```

環境変数: `OPENAI_API_KEY`

### AnthropicAdapter (adapter/anthropic.go)

設定:
```json
{
  "model": "claude-3-opus-20240229",
  "messages": [{"role": "user", "content": "..."}],
  "max_tokens": 1000
}
```

環境変数: `ANTHROPIC_API_KEY`

### HTTPAdapter (adapter/http.go)

設定:
```json
{
  "url": "https://api.example.com/endpoint",
  "method": "POST",
  "headers": {"Authorization": "Bearer {{secret.api_key}}"},
  "body": {"data": "{{input.data}}"},
  "timeout_ms": 30000
}
```

## DAGエンジン (engine/executor.go)

### 実行フロー

1. プロジェクト定義（ステップ、エッジ）をロード
2. 実行するStartブロックを特定（`start_step_id`から）
3. 指定されたStartブロックから実行グラフを構築
4. トポロジカル順序でステップを実行
5. 分岐を処理（conditionステップ）
6. 並列実行を処理（mapステップ）
7. 出力を収集し、実行ステータスを更新

> **注意**: プロジェクトは複数のStartブロックを持つことができるため、実行エンジンはどのサブグラフを実行するか知るために`start_step_id`が必要です。

### 条件式構文 (engine/condition.go)

```
$.field == "value"     # 文字列等価
$.field != "value"     # 文字列不等価
$.field > 10           # 数値比較
$.field >= 10
$.field < 10
$.field <= 10
$.nested.field         # ネストされたパスアクセス
$.field                # truthy チェック
```

### ジョブキュー (engine/queue.go)

キュー名: `project:jobs`

ジョブペイロード:
```json
{
  "run_id": "uuid",
  "project_id": "uuid",
  "start_step_id": "uuid",
  "tenant_id": "uuid"
}
```

## ミドルウェア

### 認証ミドルウェア (middleware/auth.go)

```go
// JWTから抽出:
// - tenant_id (クレーム: "tenant_id" または resource_access から)
// - user_id (クレーム: "sub")
// - email (クレーム: "email")
// - roles (クレーム: "realm_access.roles")

// コンテキストキー:
ctx.Value("tenant_id").(uuid.UUID)
ctx.Value("user_id").(string)
ctx.Value("email").(string)
ctx.Value("roles").([]string)
```

バイパス: 開発モードでは`AUTH_ENABLED=false`を設定するか`X-Tenant-ID`ヘッダーを使用。

## テレメトリ (pkg/telemetry/)

### 初期化

```go
cleanup, err := telemetry.Init(ctx, telemetry.Config{
    ServiceName: "api",
    Endpoint:    "jaeger:4318",
    Enabled:     true,
})
defer cleanup()
```

### Span作成

```go
ctx, span := telemetry.StartSpan(ctx, "operation_name")
defer span.End()

span.SetAttributes(
    attribute.String("workflow_id", id.String()),
)
```

## エラーハンドリング

### ドメインエラー (domain/errors.go)

```go
var (
    ErrNotFound       = errors.New("not found")
    ErrValidation     = errors.New("validation error")
    ErrUnauthorized   = errors.New("unauthorized")
    ErrForbidden      = errors.New("forbidden")
    ErrConflict       = errors.New("conflict")
    ErrCyclicDAG      = errors.New("cyclic DAG detected")
    ErrInvalidConfig  = errors.New("invalid step config")
)
```

### ハンドラーエラーレスポンス

```go
func respondError(w http.ResponseWriter, code string, message string, status int) {
    json.NewEncoder(w).Encode(map[string]interface{}{
        "error": map[string]string{
            "code":    code,
            "message": message,
        },
    })
}
```

## テストパターン

### ユニットテスト

```go
func TestWorkflowUsecase_Create(t *testing.T) {
    repo := &mockWorkflowRepo{}
    uc := usecase.NewWorkflowUsecase(repo)

    w, err := uc.Create(ctx, &domain.Workflow{Name: "test"})

    assert.NoError(t, err)
    assert.NotEmpty(t, w.ID)
}
```

### E2Eテスト

```go
func TestWorkflowE2E(t *testing.T) {
    // セットアップ: API経由でワークフロー作成
    resp, _ := http.Post(baseURL+"/api/v1/workflows", "application/json", body)

    // アサート
    assert.Equal(t, http.StatusCreated, resp.StatusCode)
}
```

## ビルドコマンド

以下のコマンドは`backend/`ディレクトリ内で実行します：

```bash
cd backend

# APIをビルド
go build -o bin/api ./cmd/api

# Workerをビルド
go build -o bin/worker ./cmd/worker

# Seederをビルド
go build -o bin/seeder ./cmd/seeder

# テスト実行
go test ./...

# race検出器付きで実行
go test -race ./...

# モック生成（mockgen使用時）
go generate ./...
```

## ブロックシーディングコマンド

プログラム的なブロック定義のマイグレーションコマンドです。

```bash
# ブロック定義をデータベースにマイグレート（UPSERT）
make seed-blocks

# バリデーションのみ実行（DBに書き込まない）
make seed-blocks-validate

# ドライラン（変更内容をプレビュー）
make seed-blocks-dry-run
```

CLIを直接実行する場合：

```bash
cd backend

# マイグレーション実行（DATABASE_URL環境変数が必須）
DATABASE_URL="postgres://aio:aio_password@localhost:5432/ai_orchestration?sslmode=disable" \
  go run ./cmd/seeder

# バリデーションのみ（DB接続不要）
go run ./cmd/seeder -validate

# ドライラン（詳細出力）
DATABASE_URL="postgres://aio:aio_password@localhost:5432/ai_orchestration?sslmode=disable" \
  go run ./cmd/seeder -dry-run -verbose
```

**Note**: `make seed-blocks` コマンドはMakefile内でDATABASE_URLを自動設定します。

### Seederマイグレーション処理

Seederは多段継承を正しく処理するため、Kahn's Algorithmによるトポロジカルソートを使用：

```
http (Level 0)
  ↓ 最初にソート
rest-api (Level 1)
  ↓
bearer-api (Level 2)
  ↓
github-api (Level 3)
  ↓
github_create_issue (Level 4)
  ↓ 最後にソート
```

**処理フロー**:
1. すべてのブロック定義を収集
2. 依存関係グラフを構築（`parent_block_slug` → 子ブロック）
3. トポロジカルソートで処理順序を決定
4. 循環依存を検出（エラー時はマイグレーション中止）
5. 親から子の順にUPSERT実行

**参照**: `internal/seed/migration/migrator.go` - `topologicalSort()` 関数

## 標準コードパターン (必須)

Claude Codeはこのセクションのパターンに従ってコードを書くこと。
既存コードが異なるパターンを使っていても、このパターンを優先する。

### Handlerパターン

```go
// ✅ 正しいパターン
func (h *ProjectHandler) Create(c echo.Context) error {
    ctx := c.Request().Context()
    tenantID := middleware.GetTenantID(ctx)

    var req CreateProjectRequest
    if err := c.Bind(&req); err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
    }
    if err := c.Validate(&req); err != nil {
        return err // バリデーションミドルウェアがレスポンスを処理
    }

    result, err := h.usecase.Create(ctx, tenantID, req.ToInput())
    if err != nil {
        return h.mapError(err)
    }

    return c.JSON(http.StatusCreated, NewProjectResponse(result))
}

// ❌ 禁止パターン
func (h *ProjectHandler) Create(c echo.Context) error {
    var req CreateProjectRequest
    c.Bind(&req)  // エラー無視 → NG

    tenantID, _ := uuid.Parse(c.Request().Header.Get("X-Tenant-ID"))  // middleware 未使用 → NG

    // ctx を作成 → NG（c.Request().Context() を使う）
    ctx := context.Background()

    result, _ := h.usecase.Create(ctx, tenantID, &req)  // エラー無視 → NG
    return c.JSON(200, result)
}
```

**Why**:
- `c.Bind()` のエラーを無視すると不正リクエストが処理される
- `middleware.GetTenantID()` を使わないとテナント分離が壊れる
- `c.Request().Context()` を使わないと OpenTelemetry トレースが途切れる

---

### Usecaseパターン

```go
// ✅ 正しいパターン
func (u *ProjectUsecase) Create(ctx context.Context, tenantID uuid.UUID, input *CreateProjectInput) (*domain.Project, error) {
    // 1. バリデーション
    if input.Name == "" {
        return nil, domain.ErrValidation
    }

    // 2. ビジネスロジック
    project := &domain.Project{
        ID:        uuid.New(),
        TenantID:  tenantID,
        Name:      input.Name,
        Status:    domain.ProjectStatusDraft,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    // 3. 永続化
    if err := u.repo.Create(ctx, project); err != nil {
        return nil, fmt.Errorf("create project: %w", err)
    }

    return project, nil
}

// ❌ 禁止パターン
func (u *ProjectUsecase) Create(ctx context.Context, input *CreateProjectInput) (*domain.Project, error) {
    // tenantID が引数にない → NG
    // ID を外部から受け取る → NG（Usecase 内で生成）
    // time.Now() を外部から受け取る → NG
    project := &domain.Project{
        ID: input.ID,  // NG
    }
    return u.repo.Create(ctx, project)
}
```

**Why**:
- tenantID は必ず Usecase の引数で受け取る（マルチテナント分離）
- ID は Usecase 内で生成（外部からの ID 注入は禁止）
- エラーは `fmt.Errorf("context: %w", err)` でラップ

---

### Repositoryパターン

```go
// ✅ 正しいパターン
func (r *ProjectRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Project, error) {
    query := `
        SELECT id, tenant_id, name, status, created_at, updated_at
        FROM projects
        WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
    `

    var p domain.Project
    err := r.db.QueryRow(ctx, query, id, tenantID).Scan(
        &p.ID, &p.TenantID, &p.Name, &p.Status, &p.CreatedAt, &p.UpdatedAt,
    )
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, domain.ErrNotFound
        }
        return nil, fmt.Errorf("query project: %w", err)
    }

    return &p, nil
}

// ❌ 禁止パターン
func (r *ProjectRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
    // tenant_id フィルタなし → NG（テナント分離違反）
    query := `SELECT * FROM projects WHERE id = $1`

    // deleted_at チェックなし → NG（論理削除違反）
    // SELECT * 使用 → NG（カラム明示）

    return r.db.Query(ctx, query, id)  // Scan 漏れ → NG
}
```

**Why**:
- すべてのクエリに `tenant_id` フィルタ必須
- すべてのクエリに `deleted_at IS NULL` 必須（soft delete 対応テーブル）
- `SELECT *` 禁止（カラムを明示）

---

### Domain Errorパターン

```go
// ✅ 正しいパターン
func (u *ProjectUsecase) Publish(ctx context.Context, tenantID, id uuid.UUID) error {
    project, err := u.repo.GetByID(ctx, tenantID, id)
    if err != nil {
        return err  // domain.ErrNotFound がそのまま返る
    }

    if project.Status == domain.ProjectStatusPublished {
        return domain.ErrConflict  // 既に公開済み
    }

    steps, err := u.stepRepo.GetByProjectID(ctx, project.ID)
    if err != nil {
        return fmt.Errorf("get steps: %w", err)
    }

    if len(steps) == 0 {
        return fmt.Errorf("%w: project has no steps", domain.ErrValidation)
    }

    // 少なくとも1つのStartブロックが存在することを検証
    startBlocks, err := u.stepRepo.GetStartBlocks(ctx, project.ID)
    if err != nil {
        return fmt.Errorf("get start blocks: %w", err)
    }

    if len(startBlocks) == 0 {
        return fmt.Errorf("%w: project has no start blocks", domain.ErrValidation)
    }

    // ...
}
```

**標準Domain Error**:
| Error | HTTP Status | 用途 |
|-------|-------------|------|
| `domain.ErrNotFound` | 404 | リソースが存在しない |
| `domain.ErrValidation` | 400 | 入力値が不正 |
| `domain.ErrUnauthorized` | 401 | 認証が必要 |
| `domain.ErrForbidden` | 403 | 権限がない |
| `domain.ErrConflict` | 409 | 状態の競合 |

---

### テストパターン

```go
// ✅ 正しいパターン: テーブル駆動テスト
func TestProjectUsecase_Create(t *testing.T) {
    tests := []struct {
        name    string
        input   *CreateProjectInput
        want    *domain.Project
        wantErr error
    }{
        // 正常系
        {
            name:  "有効な入力でプロジェクト作成",
            input: &CreateProjectInput{Name: "Test Project"},
            want:  &domain.Project{Name: "Test Project", Status: domain.ProjectStatusDraft},
        },
        // 異常系 - 必須
        {
            name:    "空の名前でバリデーションエラー",
            input:   &CreateProjectInput{Name: ""},
            wantErr: domain.ErrValidation,
        },
        // 境界値
        {
            name:  "最大長の名前で成功",
            input: &CreateProjectInput{Name: strings.Repeat("a", 255)},
            want:  &domain.Project{Status: domain.ProjectStatusDraft},
        },
        {
            name:    "最大長超過の名前で失敗",
            input:   &CreateProjectInput{Name: strings.Repeat("a", 256)},
            wantErr: domain.ErrValidation,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            repo := &mockProjectRepo{}
            uc := usecase.NewProjectUsecase(repo)

            got, err := uc.Create(ctx, tenantID, tt.input)

            if tt.wantErr != nil {
                assert.ErrorIs(t, err, tt.wantErr)
                return
            }

            assert.NoError(t, err)
            assert.Equal(t, tt.want.Status, got.Status)
        })
    }
}
```

**テストカバレッジ必須項目**:
1. 正常系（最低1ケース）
2. 必須フィールド欠落
3. 不正な値（型違い、範囲外）
4. 境界値（最小値、最大値、空）
5. 存在しないリソース（404）
6. 権限エラー（403）

---

### JSON処理パターン

```go
// ✅ 正しいパターン
func (s *Step) GetConfig() (*LLMConfig, error) {
    var cfg LLMConfig
    if err := json.Unmarshal(s.Config, &cfg); err != nil {
        return nil, fmt.Errorf("unmarshal config: %w", err)
    }
    return &cfg, nil
}

func (s *Step) SetConfig(cfg *LLMConfig) error {
    data, err := json.Marshal(cfg)
    if err != nil {
        return fmt.Errorf("marshal config: %w", err)
    }
    s.Config = data
    return nil
}

// ❌ 禁止パターン
func (s *Step) GetConfig() *LLMConfig {
    var cfg LLMConfig
    json.Unmarshal(s.Config, &cfg)  // エラー無視 → NG
    return &cfg
}
```

---

### Context伝播パターン

```go
// ✅ 正しいパターン
func (u *ProjectUsecase) Execute(ctx context.Context, tenantID, projectID, startStepID uuid.UUID) error {
    ctx, span := telemetry.StartSpan(ctx, "ProjectUsecase.Execute")
    defer span.End()

    span.SetAttributes(
        attribute.String("tenant_id", tenantID.String()),
        attribute.String("project_id", projectID.String()),
        attribute.String("start_step_id", startStepID.String()),
    )

    // ctx を全ての呼び出しに伝播
    project, err := u.repo.GetByID(ctx, tenantID, projectID)
    if err != nil {
        span.RecordError(err)
        return err
    }

    // start_step_id がこのプロジェクトに属することを検証
    startStep, err := u.stepRepo.GetByID(ctx, startStepID)
    if err != nil {
        span.RecordError(err)
        return err
    }

    if startStep.Type != domain.StepTypeStart {
        return fmt.Errorf("%w: specified step is not a start block", domain.ErrValidation)
    }

    // ...
}

// ❌ 禁止パターン
func (u *ProjectUsecase) Execute(tenantID, id uuid.UUID) error {
    // ctx 引数なし → NG
    ctx := context.Background()  // 新規 ctx 作成 → NG（トレース途切れ）
    // start_step_id なし → NG（マルチスタートプロジェクトでは必須）
    // ...
}
```

---

## 関連ドキュメント

- [API.md](./API.md) - REST APIエンドポイントとスキーマ
- [DATABASE.md](./DATABASE.md) - データベーススキーマとクエリ
- [BLOCK_REGISTRY.md](./BLOCK_REGISTRY.md) - ブロック定義とエラーコード
- [UNIFIED_BLOCK_MODEL.md](./designs/UNIFIED_BLOCK_MODEL.md) - ブロック実行アーキテクチャ
- [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) - エラー対処法
