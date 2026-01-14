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
| Maximum inheritance depth | 10 levels |
| Circular inheritance | Not allowed |

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

```bash
# Build API
go build -o bin/api ./cmd/api

# Build Worker
go build -o bin/worker ./cmd/worker

# Run tests
go test ./...

# Run with race detector
go test -race ./...

# Generate mocks (if using mockgen)
go generate ./...
```

## Related Documents

- [API.md](./API.md) - REST API endpoints and schemas
- [DATABASE.md](./DATABASE.md) - Database schema and queries
- [BLOCK_REGISTRY.md](./BLOCK_REGISTRY.md) - Block definitions and error codes
- [UNIFIED_BLOCK_MODEL.md](./designs/UNIFIED_BLOCK_MODEL.md) - Block execution architecture
