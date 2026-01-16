# Backend Technical Reference

Go backend code structure, interfaces, and patterns.

## Quick Reference

| Item | Value |
|------|-------|
| Language | Go 1.22+ |
| Architecture | Clean Architecture (Handler → Usecase → Domain → Repository) |
| Entry Points | `cmd/api/main.go`, `cmd/worker/main.go` |
| Domain Models | `internal/domain/` |
| API Handlers | `internal/handler/` |
| Business Logic | `internal/usecase/` |
| Database | `internal/repository/postgres/` |
| External APIs | `internal/adapter/` |
| DAG Engine | `internal/engine/` |

## Directory Structure

```
backend/
├── cmd/
│   ├── api/main.go         # HTTP server, routing, middleware setup
│   └── worker/main.go      # Job consumer, DAG executor
├── internal/
│   ├── domain/             # Entities, business rules
│   ├── usecase/            # Application logic
│   ├── handler/            # HTTP handlers
│   ├── repository/         # Data access
│   │   └── postgres/       # PostgreSQL implementation
│   ├── adapter/            # External service integrations
│   ├── engine/             # DAG execution engine
│   └── middleware/         # HTTP middleware
├── pkg/
│   ├── database/           # DB connection pool
│   ├── redis/              # Redis client wrapper
│   └── telemetry/          # OpenTelemetry SDK
├── migrations/             # SQL migrations
└── tests/e2e/              # Integration tests
```

## Layer Dependencies

```
handler -> usecase -> domain
                  -> repository
                  -> adapter
                  -> engine
```

## Domain Models

### Workflow (domain/workflow.go)

```go
type Workflow struct {
    ID          uuid.UUID
    TenantID    uuid.UUID
    Name        string
    Description string
    Status      WorkflowStatus  // "draft" | "published"
    Version     int
    InputSchema json.RawMessage
    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   *time.Time
}

type WorkflowStatus string
const (
    WorkflowStatusDraft     WorkflowStatus = "draft"
    WorkflowStatusPublished WorkflowStatus = "published"
)
```

### Step (domain/step.go)

```go
type Step struct {
    ID         uuid.UUID
    WorkflowID uuid.UUID
    Name       string
    Type       StepType
    Config     json.RawMessage
    Position   Position
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

type StepType string
const (
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

### Step Config Schemas

#### LLM Step
```json
{
  "provider": "openai|anthropic",
  "model": "gpt-4|claude-3-opus-20240229",
  "prompt": "template with {{input.field}}",
  "temperature": 0.7,
  "max_tokens": 1000
}
```

#### Tool Step
```json
{
  "adapter_id": "mock|http|openai|anthropic",
  "...adapter_specific_fields"
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
  "count": 10,                    // for: number of iterations
  "input_path": "$.items",        // forEach: path to array
  "condition": "$.index < 10",    // while/doWhile: condition expression
  "max_iterations": 100,          // safety limit (default: 100)
  "adapter_id": "mock"            // optional: adapter to execute per iteration
}
```

Loop types:
- `for`: Fixed count iterations
- `forEach`: Iterate over array elements
- `while`: Continue while condition is true (check before execution)
- `doWhile`: Execute at least once, then check condition

#### Wait Step
```json
{
  "duration_ms": 5000,            // delay in milliseconds
  "until": "2024-01-15T10:00:00Z" // OR wait until ISO8601 datetime
}
```

| Constraint | Value |
|------------|-------|
| Maximum duration | 1 hour (3600000 ms) |

#### Function Step
```json
{
  "code": "return input.value * 2",
  "language": "javascript",       // currently only javascript
  "timeout_ms": 5000              // execution timeout
}
```

| Status | Description |
|--------|-------------|
| Implementation | Partial - passes through input with warning |

#### Router Step
```json
{
  "routes": [
    {"name": "support", "description": "Customer support requests"},
    {"name": "sales", "description": "Sales inquiries"}
  ],
  "provider": "openai|anthropic", // LLM provider for classification
  "model": "gpt-4",               // model for routing decision
  "prompt": "Classify this input" // optional custom prompt
}
```

| Behavior | Description |
|----------|-------------|
| Routing | Uses LLM to classify input and select appropriate route |

#### Human-in-Loop Step
```json
{
  "instructions": "Please review and approve",
  "timeout_hours": 24,
  "approval_url": true,           // generate approval URL
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

| Mode | Behavior |
|------|----------|
| Test | Auto-approved |
| Production | Workflow pauses until approval received |

### Edge (domain/edge.go)

```go
type Edge struct {
    ID           uuid.UUID
    WorkflowID   uuid.UUID
    SourceStepID uuid.UUID
    TargetStepID uuid.UUID
    Condition    string  // Optional: "$.success == true"
    CreatedAt    time.Time
}
```

### BlockGroup (domain/block_group.go)

Control flow construct that groups multiple steps into a single logical unit.

```go
type BlockGroup struct {
    ID            uuid.UUID
    WorkflowID    uuid.UUID
    Name          string
    Type          BlockGroupType
    Config        json.RawMessage
    ParentGroupID *uuid.UUID      // For nested groups
    PositionX     int
    PositionY     int
    Width         int
    Height        int
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

type BlockGroupType string
const (
    BlockGroupTypeParallel   BlockGroupType = "parallel"    // Parallel execution
    BlockGroupTypeTryCatch   BlockGroupType = "try_catch"   // Error handling
    BlockGroupTypeIfElse     BlockGroupType = "if_else"     // Conditional branch
    BlockGroupTypeSwitchCase BlockGroupType = "switch_case" // Multi-branch routing
    BlockGroupTypeForeach    BlockGroupType = "foreach"     // Array iteration
    BlockGroupTypeWhile      BlockGroupType = "while"       // Condition loop
)
```

#### BlockGroup Config Examples

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

#### Step Group Roles

Steps within a BlockGroup have a `group_role` field:

| Role | Block Type | Description |
|------|-----------|-------------|
| `body` | parallel, foreach, while | Main execution steps |
| `try` | try_catch | Try block steps |
| `catch` | try_catch | Error handling steps |
| `finally` | try_catch | Cleanup steps |
| `then` | if_else | True branch steps |
| `else` | if_else | False branch steps |
| `case_N` | switch_case | Case branch steps |
| `default` | switch_case | Default branch steps |

### BlockDefinition (domain/block.go)

Block definitions represent reusable execution units that can be inherited and extended.

```go
type BlockDefinition struct {
    ID            uuid.UUID
    TenantID      *uuid.UUID       // nil for system blocks
    Slug          string           // unique identifier
    Name          string
    Description   string
    Category      BlockCategory
    Icon          string
    ConfigSchema  json.RawMessage  // JSON Schema for config
    InputSchema   json.RawMessage  // JSON Schema for input
    OutputSchema  json.RawMessage  // JSON Schema for output
    InputPorts    []InputPort
    OutputPorts   []OutputPort
    ErrorCodes    []ErrorCodeDef

    // Unified Block Model
    Code          string           // JavaScript code executed in sandbox
    UIConfig      json.RawMessage  // UI metadata (icon, color, etc.)
    IsSystem      bool             // System blocks can only be edited by admins
    Version       int              // Version number

    // Block Inheritance/Extension
    ParentBlockID *uuid.UUID       // Reference to parent block
    ConfigDefaults json.RawMessage // Default values for parent's config
    PreProcess    string           // JavaScript for input transformation
    PostProcess   string           // JavaScript for output transformation
    InternalSteps []InternalStep   // Composite block internal steps

    // Resolved fields (populated by backend)
    PreProcessChain        []string         // Chain of preProcess (child→root)
    PostProcessChain       []string         // Chain of postProcess (root→child)
    ResolvedCode           string           // Code from root ancestor
    ResolvedConfigDefaults json.RawMessage  // Merged config defaults

    Enabled    bool
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

type InternalStep struct {
    Type      string          `json:"type"`       // Block slug to execute
    Config    json.RawMessage `json:"config"`     // Step configuration
    OutputKey string          `json:"output_key"` // Key for storing output
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

#### Block Inheritance Constraints

| Constraint | Value |
|------------|-------|
| Only blocks with code can be inherited | `Code != ""` |
| Maximum inheritance depth | 50 levels（実用上は4-5レベル） |
| Circular inheritance | Not allowed（トポロジカルソートで検出） |
| Tenant isolation | 同一テナント内またはシステムブロックからのみ継承可能 |

#### Block Execution Flow

When executing an inherited block:
1. **PreProcess Chain** (child → root): Transform input through each preProcess
2. **Internal Steps** (if any): Execute internal steps sequentially
3. **Code Execution**: Run the resolved code from root ancestor
4. **PostProcess Chain** (root → child): Transform output through each postProcess

### Run (domain/run.go)

```go
type Run struct {
    ID              uuid.UUID
    WorkflowID      uuid.UUID
    WorkflowVersion int
    TenantID        uuid.UUID
    Status          RunStatus
    Mode            RunMode
    TriggerType     TriggerType
    Input           json.RawMessage
    Output          json.RawMessage
    Error           string
    StartedAt       *time.Time
    CompletedAt     *time.Time
    CreatedAt       time.Time
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

## Interfaces

### Repository Interface (repository/interfaces.go)

```go
type WorkflowRepository interface {
    Create(ctx context.Context, w *domain.Workflow) error
    GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Workflow, error)
    List(ctx context.Context, tenantID uuid.UUID, filter WorkflowFilter) ([]*domain.Workflow, error)
    Update(ctx context.Context, w *domain.Workflow) error
    Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

type StepRepository interface {
    Create(ctx context.Context, s *domain.Step) error
    GetByWorkflowID(ctx context.Context, workflowID uuid.UUID) ([]*domain.Step, error)
    Update(ctx context.Context, s *domain.Step) error
    Delete(ctx context.Context, id uuid.UUID) error
}

type EdgeRepository interface {
    Create(ctx context.Context, e *domain.Edge) error
    GetByWorkflowID(ctx context.Context, workflowID uuid.UUID) ([]*domain.Edge, error)
    Delete(ctx context.Context, id uuid.UUID) error
}

type RunRepository interface {
    Create(ctx context.Context, r *domain.Run) error
    GetByID(ctx context.Context, id uuid.UUID) (*domain.Run, error)
    Update(ctx context.Context, r *domain.Run) error
    ListByWorkflowID(ctx context.Context, workflowID uuid.UUID) ([]*domain.Run, error)
}

type StepRunRepository interface {
    Create(ctx context.Context, sr *domain.StepRun) error
    Update(ctx context.Context, sr *domain.StepRun) error
    GetByRunID(ctx context.Context, runID uuid.UUID) ([]*domain.StepRun, error)
}
```

### Adapter Interface (adapter/adapter.go)

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

## Adapter Implementations

### MockAdapter (adapter/mock.go)

Config:
```json
{
  "response": {"key": "value"},
  "delay_ms": 100,
  "error": "optional error message",
  "status_code": 200
}
```

### OpenAIAdapter (adapter/openai.go)

Config:
```json
{
  "model": "gpt-4",
  "messages": [{"role": "user", "content": "..."}],
  "temperature": 0.7,
  "max_tokens": 1000
}
```

Environment: `OPENAI_API_KEY`

### AnthropicAdapter (adapter/anthropic.go)

Config:
```json
{
  "model": "claude-3-opus-20240229",
  "messages": [{"role": "user", "content": "..."}],
  "max_tokens": 1000
}
```

Environment: `ANTHROPIC_API_KEY`

### HTTPAdapter (adapter/http.go)

Config:
```json
{
  "url": "https://api.example.com/endpoint",
  "method": "POST",
  "headers": {"Authorization": "Bearer {{secret.api_key}}"},
  "body": {"data": "{{input.data}}"},
  "timeout_ms": 30000
}
```

## DAG Engine (engine/executor.go)

### Execution Flow

1. Load workflow definition (steps, edges)
2. Build execution graph
3. Find entry steps (no incoming edges)
4. Execute steps in topological order
5. Handle branching (condition steps)
6. Handle parallel execution (map steps)
7. Collect outputs, update run status

### Condition Expression Syntax (engine/condition.go)

```
$.field == "value"     # String equality
$.field != "value"     # String inequality
$.field > 10           # Numeric comparison
$.field >= 10
$.field < 10
$.field <= 10
$.nested.field         # Nested path access
$.field                # Truthy check
```

### Job Queue (engine/queue.go)

Queue name: `workflow:jobs`

Job payload:
```json
{
  "run_id": "uuid",
  "workflow_id": "uuid",
  "tenant_id": "uuid"
}
```

## Middleware

### Auth Middleware (middleware/auth.go)

```go
// Extracts from JWT:
// - tenant_id (claim: "tenant_id" or from resource_access)
// - user_id (claim: "sub")
// - email (claim: "email")
// - roles (claim: "realm_access.roles")

// Context keys:
ctx.Value("tenant_id").(uuid.UUID)
ctx.Value("user_id").(string)
ctx.Value("email").(string)
ctx.Value("roles").([]string)
```

Bypass: Set `AUTH_ENABLED=false` or use `X-Tenant-ID` header in dev mode.

## Telemetry (pkg/telemetry/)

### Initialization

```go
cleanup, err := telemetry.Init(ctx, telemetry.Config{
    ServiceName: "api",
    Endpoint:    "jaeger:4318",
    Enabled:     true,
})
defer cleanup()
```

### Span Creation

```go
ctx, span := telemetry.StartSpan(ctx, "operation_name")
defer span.End()

span.SetAttributes(
    attribute.String("workflow_id", id.String()),
)
```

## Error Handling

### Domain Errors (domain/errors.go)

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

### Handler Error Response

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

## Testing Patterns

### Unit Test

```go
func TestWorkflowUsecase_Create(t *testing.T) {
    repo := &mockWorkflowRepo{}
    uc := usecase.NewWorkflowUsecase(repo)

    w, err := uc.Create(ctx, &domain.Workflow{Name: "test"})

    assert.NoError(t, err)
    assert.NotEmpty(t, w.ID)
}
```

### E2E Test

```go
func TestWorkflowE2E(t *testing.T) {
    // Setup: create workflow via API
    resp, _ := http.Post(baseURL+"/api/v1/workflows", "application/json", body)

    // Assert
    assert.Equal(t, http.StatusCreated, resp.StatusCode)
}
```

## Build Commands

以下のコマンドは`backend/`ディレクトリ内で実行します：

```bash
cd backend

# Build API
go build -o bin/api ./cmd/api

# Build Worker
go build -o bin/worker ./cmd/worker

# Build Seeder
go build -o bin/seeder ./cmd/seeder

# Run tests
go test ./...

# Run with race detector
go test -race ./...

# Generate mocks (if using mockgen)
go generate ./...
```

## Block Seeding Commands

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

### Seeder マイグレーション処理

Seeder は多段継承を正しく処理するため、Kahn's Algorithm によるトポロジカルソートを使用：

```
http (Level 0)
  ↓ sorted first
rest-api (Level 1)
  ↓
bearer-api (Level 2)
  ↓
github-api (Level 3)
  ↓
github_create_issue (Level 4)
  ↓ sorted last
```

**処理フロー**:
1. すべてのブロック定義を収集
2. 依存関係グラフを構築（`parent_block_slug` → 子ブロック）
3. トポロジカルソートで処理順序を決定
4. 循環依存を検出（エラー時はマイグレーション中止）
5. 親から子の順にUPSERT実行

**See**: `internal/seed/migration/migrator.go` - `topologicalSort()` 関数

## Canonical Code Patterns (必須)

Claude Code はこのセクションのパターンに従ってコードを書くこと。
既存コードが異なるパターンを使っていても、このパターンを優先する。

### Handler パターン

```go
// ✅ 正しいパターン
func (h *WorkflowHandler) Create(c echo.Context) error {
    ctx := c.Request().Context()
    tenantID := middleware.GetTenantID(ctx)

    var req CreateWorkflowRequest
    if err := c.Bind(&req); err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
    }
    if err := c.Validate(&req); err != nil {
        return err // validation middleware handles response
    }

    result, err := h.usecase.Create(ctx, tenantID, req.ToInput())
    if err != nil {
        return h.mapError(err)
    }

    return c.JSON(http.StatusCreated, NewWorkflowResponse(result))
}

// ❌ 禁止パターン
func (h *WorkflowHandler) Create(c echo.Context) error {
    var req CreateWorkflowRequest
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

### Usecase パターン

```go
// ✅ 正しいパターン
func (u *WorkflowUsecase) Create(ctx context.Context, tenantID uuid.UUID, input *CreateWorkflowInput) (*domain.Workflow, error) {
    // 1. バリデーション
    if input.Name == "" {
        return nil, domain.ErrValidation
    }

    // 2. ビジネスロジック
    workflow := &domain.Workflow{
        ID:        uuid.New(),
        TenantID:  tenantID,
        Name:      input.Name,
        Status:    domain.WorkflowStatusDraft,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    // 3. 永続化
    if err := u.repo.Create(ctx, workflow); err != nil {
        return nil, fmt.Errorf("create workflow: %w", err)
    }

    return workflow, nil
}

// ❌ 禁止パターン
func (u *WorkflowUsecase) Create(ctx context.Context, input *CreateWorkflowInput) (*domain.Workflow, error) {
    // tenantID が引数にない → NG
    // ID を外部から受け取る → NG（Usecase 内で生成）
    // time.Now() を外部から受け取る → NG
    workflow := &domain.Workflow{
        ID: input.ID,  // NG
    }
    return u.repo.Create(ctx, workflow)
}
```

**Why**:
- tenantID は必ず Usecase の引数で受け取る（マルチテナント分離）
- ID は Usecase 内で生成（外部からの ID 注入は禁止）
- エラーは `fmt.Errorf("context: %w", err)` でラップ

---

### Repository パターン

```go
// ✅ 正しいパターン
func (r *WorkflowRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Workflow, error) {
    query := `
        SELECT id, tenant_id, name, status, created_at, updated_at
        FROM workflows
        WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
    `

    var w domain.Workflow
    err := r.db.QueryRow(ctx, query, id, tenantID).Scan(
        &w.ID, &w.TenantID, &w.Name, &w.Status, &w.CreatedAt, &w.UpdatedAt,
    )
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, domain.ErrNotFound
        }
        return nil, fmt.Errorf("query workflow: %w", err)
    }

    return &w, nil
}

// ❌ 禁止パターン
func (r *WorkflowRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Workflow, error) {
    // tenant_id フィルタなし → NG（テナント分離違反）
    query := `SELECT * FROM workflows WHERE id = $1`

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

### Domain Error パターン

```go
// ✅ 正しいパターン
func (u *WorkflowUsecase) Publish(ctx context.Context, tenantID, id uuid.UUID) error {
    workflow, err := u.repo.GetByID(ctx, tenantID, id)
    if err != nil {
        return err  // domain.ErrNotFound がそのまま返る
    }

    if workflow.Status == domain.WorkflowStatusPublished {
        return domain.ErrConflict  // 既に公開済み
    }

    steps, err := u.stepRepo.GetByWorkflowID(ctx, workflow.ID)
    if err != nil {
        return fmt.Errorf("get steps: %w", err)
    }

    if len(steps) == 0 {
        return fmt.Errorf("%w: workflow has no steps", domain.ErrValidation)
    }

    // ...
}
```

**標準 Domain Error**:
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
// ✅ 正しいパターン: Table-Driven Tests
func TestWorkflowUsecase_Create(t *testing.T) {
    tests := []struct {
        name    string
        input   *CreateWorkflowInput
        want    *domain.Workflow
        wantErr error
    }{
        // 正常系
        {
            name:  "valid input creates workflow",
            input: &CreateWorkflowInput{Name: "Test Workflow"},
            want:  &domain.Workflow{Name: "Test Workflow", Status: domain.WorkflowStatusDraft},
        },
        // 異常系 - 必須
        {
            name:    "empty name returns validation error",
            input:   &CreateWorkflowInput{Name: ""},
            wantErr: domain.ErrValidation,
        },
        // 境界値
        {
            name:  "max length name succeeds",
            input: &CreateWorkflowInput{Name: strings.Repeat("a", 255)},
            want:  &domain.Workflow{Status: domain.WorkflowStatusDraft},
        },
        {
            name:    "over max length name fails",
            input:   &CreateWorkflowInput{Name: strings.Repeat("a", 256)},
            wantErr: domain.ErrValidation,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            repo := &mockWorkflowRepo{}
            uc := usecase.NewWorkflowUsecase(repo)

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

### JSON 処理パターン

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

### Context 伝播パターン

```go
// ✅ 正しいパターン
func (u *WorkflowUsecase) Execute(ctx context.Context, tenantID, id uuid.UUID) error {
    ctx, span := telemetry.StartSpan(ctx, "WorkflowUsecase.Execute")
    defer span.End()

    span.SetAttributes(
        attribute.String("tenant_id", tenantID.String()),
        attribute.String("workflow_id", id.String()),
    )

    // ctx を全ての呼び出しに伝播
    workflow, err := u.repo.GetByID(ctx, tenantID, id)
    if err != nil {
        span.RecordError(err)
        return err
    }

    // ...
}

// ❌ 禁止パターン
func (u *WorkflowUsecase) Execute(tenantID, id uuid.UUID) error {
    // ctx 引数なし → NG
    ctx := context.Background()  // 新規 ctx 作成 → NG（トレース途切れ）
    // ...
}
```

---

## Related Documents

- [API.md](./API.md) - REST API endpoints and schemas
- [DATABASE.md](./DATABASE.md) - Database schema and queries
- [BLOCK_REGISTRY.md](./BLOCK_REGISTRY.md) - Block definitions and error codes
- [UNIFIED_BLOCK_MODEL.md](./designs/UNIFIED_BLOCK_MODEL.md) - Block execution architecture
- [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) - エラー対処法
