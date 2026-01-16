# マルチスタート・プロジェクトモデル設計書

## 概要

### 背景

ユーザーインタビューから以下の要望を受け、Workflowモデルを再設計する：

1. **異なるトリガーで初期ステップだけ変えたい**
   - 「Slackでメンションされたとき」「Webhookされたとき」で多少の初期ステップだけ変えたい
2. **関連する処理を同一画面で管理したい**
   - RAGの書き込みフローと呼び出しフローを同一画面で見たい
3. **複数のワークフローをまとめて編集管理したい**

### 解決策

- **Workflow → Project** に名称変更
- **1つのProjectに複数のStartブロックを配置可能**
- **各StartブロックがTrigger設定を持つ**
- **Flow概念は導入しない（シンプルさ優先）**

---

## 新しい概念モデル

### 用語

| 用語 | 説明 |
|------|------|
| **Project** | DAG全体を含む単位（旧Workflow） |
| **Entry Point** | Trigger設定を持つStartブロック |
| **Step** | DAG内の各ノード（Start, LLM, Tool等） |
| **Edge** | Step間の接続 |
| **BlockGroup** | 制御フロー構造（parallel, try_catch, foreach, while） |

### データ構造

```
Project
├── id, tenant_id, name, description
├── status (draft / published)
├── version
├── variables (共有変数)
├── steps[] (複数のStartブロックを含む)
│   └── Start Step
│       ├── trigger_type (manual, webhook, schedule, slack, ...)
│       └── trigger_config (トリガー固有設定)
├── edges[]
└── block_groups[]
```

### 画面イメージ

```
┌─────────────────────────────────────────────────────────────────────────┐
│ [← Projects] カスタマーサポートBot ▼        [Save] [Publish] [⚙]       │
├─────────────────────────────────────────────────────────────────────────┤
│ [Canvas] [Runs] [Schedules] [Variables]                                 │
├────────────┬────────────────────────────────────────┬───────────────────┤
│ Blocks     │                                        │ Properties        │
│ ──────────│         Project Canvas                 │ ─────────────────│
│ 🔍 Search  │                                        │                   │
│            │   ┌───────┐                           │ Start: Slack受信  │
│ Start      │   │ Start │    ┌───┐    ┌────┐      │ ─────────────────│
│ ┌────────┐│   │ Slack │ →  │LLM│ →  │共通│ → ●   │ Trigger: Slack    │
│ │▶ Start ││   │       │    └───┘    │処理│       │                   │
│ └────────┘│   └───────┘             └────┘       │ Event: app_mention│
│            │                           ↑          │ Channel: #support │
│ AI         │   ┌───────┐              │          │                   │
│ ├─ LLM    │   │ Start │    ┌───┐     │          │ [Test Run]        │
│ ├─ Router │   │Webhook│ →  │変換│ ────┘          │                   │
│ └─ RAG    │   │       │    └───┘                 │                   │
│            │   └───────┘                          │                   │
│ Flow       │                                        │                   │
│ ├─ Cond.  │   ┌───────┐    ┌───┐                 │                   │
│ └─ Switch │   │ Start │ →  │RAG│ → ●             │                   │
│            │   │Schedule   │更新│                 │                   │
│ Apps       │   │(毎日2時)│    └───┘                 │                   │
│ ├─ Slack  │   └───────┘                          │                   │
│ └─ HTTP   │                                        │                   │
└────────────┴────────────────────────────────────────┴───────────────────┘
```

---

## Phase 1: Database Schema

### 削除するテーブル

```sql
DROP TABLE IF EXISTS workflow_versions CASCADE;
DROP TABLE IF EXISTS workflows CASCADE;
DROP TABLE IF EXISTS webhooks CASCADE;  -- trigger_configに統合
```

### 新規テーブル: projects

```sql
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) DEFAULT 'draft' CHECK (status IN ('draft', 'published')),
    version INT DEFAULT 0,

    -- 共有変数
    variables JSONB DEFAULT '{}',

    -- ドラフト
    draft JSONB,

    -- システムプロジェクト
    is_system BOOLEAN DEFAULT FALSE,
    system_slug VARCHAR(100),

    -- メタデータ
    created_by UUID REFERENCES users(id),
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_projects_tenant ON projects(tenant_id);
CREATE INDEX idx_projects_status ON projects(status);
CREATE INDEX idx_projects_deleted ON projects(deleted_at) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_projects_system_slug ON projects(system_slug) WHERE system_slug IS NOT NULL;

COMMENT ON TABLE projects IS 'プロジェクト: 複数のStartブロックを持つDAG';
COMMENT ON COLUMN projects.variables IS 'プロジェクト全体で共有する変数';
```

### 新規テーブル: project_versions

```sql
CREATE TABLE project_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    version INT NOT NULL,
    definition JSONB NOT NULL,
    saved_by UUID REFERENCES users(id),
    saved_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (project_id, version)
);

CREATE INDEX idx_project_versions_project ON project_versions(project_id);

COMMENT ON TABLE project_versions IS 'プロジェクトのバージョン履歴';
```

### 変更テーブル: steps

```sql
-- workflow_id → project_id に変更
-- trigger_type, trigger_config を追加（Startブロック用）

ALTER TABLE steps DROP CONSTRAINT IF EXISTS steps_workflow_id_fkey;
ALTER TABLE steps DROP COLUMN IF EXISTS workflow_id;

ALTER TABLE steps ADD COLUMN project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE;
ALTER TABLE steps ADD COLUMN trigger_type VARCHAR(50);
ALTER TABLE steps ADD COLUMN trigger_config JSONB DEFAULT '{}';

ALTER TABLE steps ADD CONSTRAINT steps_trigger_type_check CHECK (
    trigger_type IS NULL OR
    trigger_type IN ('manual', 'webhook', 'schedule', 'slack', 'discord', 'email', 'internal', 'api')
);

CREATE INDEX idx_steps_project ON steps(project_id);
CREATE INDEX idx_steps_trigger_type ON steps(trigger_type) WHERE trigger_type IS NOT NULL;

COMMENT ON COLUMN steps.trigger_type IS 'Startブロックのトリガー種別（type=startの場合のみ）';
COMMENT ON COLUMN steps.trigger_config IS 'トリガー固有の設定（type=startの場合のみ）';
```

### 変更テーブル: edges

```sql
ALTER TABLE edges DROP CONSTRAINT IF EXISTS edges_workflow_id_fkey;
ALTER TABLE edges DROP COLUMN IF EXISTS workflow_id;

ALTER TABLE edges ADD COLUMN project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE;

CREATE INDEX idx_edges_project ON edges(project_id);
```

### 変更テーブル: block_groups

```sql
ALTER TABLE block_groups DROP CONSTRAINT IF EXISTS block_groups_workflow_id_fkey;
ALTER TABLE block_groups DROP COLUMN IF EXISTS workflow_id;

ALTER TABLE block_groups ADD COLUMN project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE;

CREATE INDEX idx_block_groups_project ON block_groups(project_id);
```

### 変更テーブル: runs

```sql
ALTER TABLE runs DROP CONSTRAINT IF EXISTS runs_workflow_id_fkey;
ALTER TABLE runs DROP COLUMN IF EXISTS workflow_id;
ALTER TABLE runs DROP COLUMN IF EXISTS workflow_version;

ALTER TABLE runs ADD COLUMN project_id UUID NOT NULL REFERENCES projects(id);
ALTER TABLE runs ADD COLUMN project_version INT NOT NULL;
ALTER TABLE runs ADD COLUMN start_step_id UUID REFERENCES steps(id);

CREATE INDEX idx_runs_project ON runs(project_id);
CREATE INDEX idx_runs_start_step ON runs(start_step_id) WHERE start_step_id IS NOT NULL;

COMMENT ON COLUMN runs.start_step_id IS 'どのStartブロックから実行されたか';
```

### 変更テーブル: schedules

```sql
ALTER TABLE schedules DROP CONSTRAINT IF EXISTS schedules_workflow_id_fkey;
ALTER TABLE schedules DROP COLUMN IF EXISTS workflow_id;
ALTER TABLE schedules DROP COLUMN IF EXISTS workflow_version;

ALTER TABLE schedules ADD COLUMN project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE;
ALTER TABLE schedules ADD COLUMN start_step_id UUID NOT NULL REFERENCES steps(id) ON DELETE CASCADE;

CREATE INDEX idx_schedules_project ON schedules(project_id);
CREATE INDEX idx_schedules_start_step ON schedules(start_step_id);

COMMENT ON COLUMN schedules.start_step_id IS '実行対象のStartブロック';
```

### 変更テーブル: run_number_sequences

```sql
DROP TABLE IF EXISTS run_number_sequences;

CREATE TABLE run_number_sequences (
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    triggered_by VARCHAR(50) NOT NULL,
    next_number INT DEFAULT 1,
    PRIMARY KEY (project_id, triggered_by)
);
```

### 変更テーブル: usage_records

```sql
ALTER TABLE usage_records DROP COLUMN IF EXISTS workflow_id;
ALTER TABLE usage_records ADD COLUMN project_id UUID REFERENCES projects(id);
CREATE INDEX idx_usage_records_project ON usage_records(project_id) WHERE project_id IS NOT NULL;
```

### 変更テーブル: usage_daily_aggregates

```sql
ALTER TABLE usage_daily_aggregates DROP COLUMN IF EXISTS workflow_id;
ALTER TABLE usage_daily_aggregates ADD COLUMN project_id UUID REFERENCES projects(id);
```

### 変更テーブル: usage_budgets

```sql
ALTER TABLE usage_budgets DROP COLUMN IF EXISTS workflow_id;
ALTER TABLE usage_budgets ADD COLUMN project_id UUID REFERENCES projects(id);
```

### 変更テーブル: copilot_sessions

```sql
ALTER TABLE copilot_sessions DROP COLUMN IF EXISTS workflow_id;
ALTER TABLE copilot_sessions ADD COLUMN project_id UUID REFERENCES projects(id);
```

### 削除テーブル: webhooks

```sql
-- Webhook設定はStartブロックのtrigger_configに統合
DROP TABLE IF EXISTS webhooks CASCADE;
```

### trigger_config の構造

```json
// trigger_type = 'manual' の場合
{}

// trigger_type = 'webhook' の場合
{
  "secret": "whsec_xxxxxxxxxxxx",
  "input_mapping": {
    "message": "$.body.text",
    "user_id": "$.headers.X-User-ID"
  },
  "allowed_ips": ["10.0.0.0/8"]
}

// trigger_type = 'schedule' の場合
{
  "cron_expression": "0 9 * * MON-FRI",
  "timezone": "Asia/Tokyo",
  "input": {"type": "scheduled"}
}

// trigger_type = 'slack' の場合
{
  "event_types": ["app_mention", "message"],
  "channel_filter": ["C12345"],
  "input_mapping": {
    "text": "$.event.text",
    "user": "$.event.user"
  }
}

// trigger_type = 'internal' の場合
{
  "description": "他のプロジェクトから呼び出し可能"
}
```

---

## Phase 2: Backend Domain

### 新規: domain/project.go

```go
package domain

type ProjectStatus string

const (
    ProjectStatusDraft     ProjectStatus = "draft"
    ProjectStatusPublished ProjectStatus = "published"
)

type Project struct {
    ID          uuid.UUID       `json:"id"`
    TenantID    uuid.UUID       `json:"tenant_id"`
    Name        string          `json:"name"`
    Description string          `json:"description,omitempty"`
    Status      ProjectStatus   `json:"status"`
    Version     int             `json:"version"`
    Variables   json.RawMessage `json:"variables,omitempty"`
    Draft       json.RawMessage `json:"draft,omitempty"`
    HasDraft    bool            `json:"has_draft"`

    IsSystem    bool    `json:"is_system"`
    SystemSlug  *string `json:"system_slug,omitempty"`

    CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
    PublishedAt *time.Time `json:"published_at,omitempty"`
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
    DeletedAt   *time.Time `json:"deleted_at,omitempty"`

    // Relations
    Steps       []Step       `json:"steps,omitempty"`
    Edges       []Edge       `json:"edges,omitempty"`
    BlockGroups []BlockGroup `json:"block_groups,omitempty"`
}

type ProjectDraft struct {
    Name        string          `json:"name"`
    Description string          `json:"description,omitempty"`
    Variables   json.RawMessage `json:"variables,omitempty"`
    Steps       []Step          `json:"steps"`
    Edges       []Edge          `json:"edges"`
    BlockGroups []BlockGroup    `json:"block_groups,omitempty"`
    UpdatedAt   time.Time       `json:"updated_at"`
}

type ProjectVersion struct {
    ID         uuid.UUID       `json:"id"`
    ProjectID  uuid.UUID       `json:"project_id"`
    Version    int             `json:"version"`
    Definition json.RawMessage `json:"definition"`
    SavedBy    *uuid.UUID      `json:"saved_by,omitempty"`
    SavedAt    time.Time       `json:"saved_at"`
}

type ProjectDefinition struct {
    Name        string          `json:"name"`
    Description string          `json:"description,omitempty"`
    Variables   json.RawMessage `json:"variables,omitempty"`
    Steps       []Step          `json:"steps"`
    Edges       []Edge          `json:"edges"`
    BlockGroups []BlockGroup    `json:"block_groups,omitempty"`
}
```

### 変更: domain/step.go

```go
type TriggerType string

const (
    TriggerTypeManual   TriggerType = "manual"
    TriggerTypeWebhook  TriggerType = "webhook"
    TriggerTypeSchedule TriggerType = "schedule"
    TriggerTypeSlack    TriggerType = "slack"
    TriggerTypeDiscord  TriggerType = "discord"
    TriggerTypeEmail    TriggerType = "email"
    TriggerTypeInternal TriggerType = "internal"
    TriggerTypeAPI      TriggerType = "api"
)

type Step struct {
    ID                uuid.UUID       `json:"id"`
    TenantID          uuid.UUID       `json:"tenant_id"`
    ProjectID         uuid.UUID       `json:"project_id"`  // WorkflowID → ProjectID
    Name              string          `json:"name"`
    Type              StepType        `json:"type"`
    Config            json.RawMessage `json:"config"`

    // Start ブロック専用
    TriggerType   *TriggerType    `json:"trigger_type,omitempty"`
    TriggerConfig json.RawMessage `json:"trigger_config,omitempty"`

    BlockGroupID      *uuid.UUID      `json:"block_group_id,omitempty"`
    GroupRole         string          `json:"group_role,omitempty"`
    PositionX         int             `json:"position_x"`
    PositionY         int             `json:"position_y"`
    BlockDefinitionID *uuid.UUID      `json:"block_definition_id,omitempty"`
    CredentialBindings json.RawMessage `json:"credential_bindings,omitempty"`
    CreatedAt         time.Time       `json:"created_at"`
    UpdatedAt         time.Time       `json:"updated_at"`
}
```

### 変更: domain/edge.go

```go
type Edge struct {
    ID                 uuid.UUID  `json:"id"`
    TenantID           uuid.UUID  `json:"tenant_id"`
    ProjectID          uuid.UUID  `json:"project_id"`  // WorkflowID → ProjectID
    // ... 他は変更なし
}
```

### 変更: domain/block_group.go

```go
type BlockGroup struct {
    ID            uuid.UUID       `json:"id"`
    TenantID      uuid.UUID       `json:"tenant_id"`
    ProjectID     uuid.UUID       `json:"project_id"`  // WorkflowID → ProjectID
    // ... 他は変更なし
}
```

### 変更: domain/run.go

```go
type Run struct {
    ID              uuid.UUID       `json:"id"`
    TenantID        uuid.UUID       `json:"tenant_id"`
    ProjectID       uuid.UUID       `json:"project_id"`       // WorkflowID → ProjectID
    ProjectVersion  int             `json:"project_version"`  // WorkflowVersion → ProjectVersion
    StartStepID     *uuid.UUID      `json:"start_step_id,omitempty"`  // 新規追加
    // ... 他は変更なし
}
```

### 削除ファイル

- `domain/workflow.go` → 削除

---

## Phase 3: Backend Repository

### 新規インターフェース

```go
type ProjectRepository interface {
    Create(ctx context.Context, project *domain.Project) error
    GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Project, error)
    GetWithDetails(ctx context.Context, tenantID, id uuid.UUID) (*domain.Project, error)
    List(ctx context.Context, tenantID uuid.UUID, filter ProjectFilter) ([]*domain.Project, int, error)
    Update(ctx context.Context, project *domain.Project) error
    Delete(ctx context.Context, tenantID, id uuid.UUID) error
    GetSystemBySlug(ctx context.Context, slug string) (*domain.Project, error)
}

type ProjectFilter struct {
    Status *domain.ProjectStatus
    Search string
    Page   int
    Limit  int
}

type ProjectVersionRepository interface {
    Create(ctx context.Context, version *domain.ProjectVersion) error
    GetByProjectAndVersion(ctx context.Context, projectID uuid.UUID, version int) (*domain.ProjectVersion, error)
    GetLatestByProject(ctx context.Context, projectID uuid.UUID) (*domain.ProjectVersion, error)
    ListByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.ProjectVersion, error)
}
```

### 変更: 既存Repository

| Repository | 変更内容 |
|------------|----------|
| StepRepository | `ListByWorkflow` → `ListByProject` |
| EdgeRepository | `ListByWorkflow` → `ListByProject` |
| BlockGroupRepository | `ListByWorkflow` → `ListByProject` |
| RunRepository | `ListByWorkflow` → `ListByProject` |
| ScheduleRepository | `workflow_id` → `project_id`, `start_step_id` 追加 |

### 削除ファイル

- `repository/postgres/workflow.go` → 削除
- `repository/postgres/version.go` → 削除

---

## Phase 4: Backend Usecase

### 新規: usecase/project.go

```go
type ProjectUsecase struct {
    projectRepo        repository.ProjectRepository
    stepRepo           repository.StepRepository
    edgeRepo           repository.EdgeRepository
    versionRepo        repository.ProjectVersionRepository
    blockGroupRepo     repository.BlockGroupRepository
}

// Create - プロジェクト作成
func (u *ProjectUsecase) Create(ctx context.Context, input CreateProjectInput) (*domain.Project, error)

// GetByID - プロジェクト取得
func (u *ProjectUsecase) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Project, error)

// GetWithDetails - Steps, Edges, BlockGroups付きで取得
func (u *ProjectUsecase) GetWithDetails(ctx context.Context, tenantID, id uuid.UUID) (*domain.Project, error)

// List - プロジェクト一覧
func (u *ProjectUsecase) List(ctx context.Context, input ListProjectsInput) (*ListProjectsOutput, error)

// Update - プロジェクト更新
func (u *ProjectUsecase) Update(ctx context.Context, tenantID, id uuid.UUID, input UpdateProjectInput) (*domain.Project, error)

// Delete - プロジェクト削除
func (u *ProjectUsecase) Delete(ctx context.Context, tenantID, id uuid.UUID) error

// SaveDraft - ドラフト保存
func (u *ProjectUsecase) SaveDraft(ctx context.Context, tenantID, id uuid.UUID, draft *domain.ProjectDraft) error

// PublishDraft - 公開（バージョン作成）
func (u *ProjectUsecase) PublishDraft(ctx context.Context, tenantID, id uuid.UUID) (*domain.Project, error)

// DiscardDraft - ドラフト破棄
func (u *ProjectUsecase) DiscardDraft(ctx context.Context, tenantID, id uuid.UUID) error

// UpdateVariables - 共有変数更新
func (u *ProjectUsecase) UpdateVariables(ctx context.Context, tenantID, id uuid.UUID, variables json.RawMessage) error
```

### 変更: 既存Usecase

| Usecase | 変更内容 |
|---------|----------|
| StepUsecase | `workflowID` → `projectID`、`UpdateTrigger` メソッド追加 |
| EdgeUsecase | `workflowID` → `projectID` |
| BlockGroupUsecase | `workflowID` → `projectID` |
| RunUsecase | `workflowID` → `projectID`、`startStepID` パラメータ追加 |
| ScheduleUsecase | `workflowID` → `projectID`、`startStepID` 必須化 |

### 削除ファイル

- `usecase/workflow.go` → 削除

---

## Phase 5: Backend Handler

### 新規API一覧

```
# Projects
POST   /api/v1/projects                              # 作成
GET    /api/v1/projects                              # 一覧
GET    /api/v1/projects/{projectId}                  # 詳細
PUT    /api/v1/projects/{projectId}                  # 更新
DELETE /api/v1/projects/{projectId}                  # 削除
POST   /api/v1/projects/{projectId}/draft            # ドラフト保存
POST   /api/v1/projects/{projectId}/publish          # 公開
DELETE /api/v1/projects/{projectId}/draft            # ドラフト破棄
PUT    /api/v1/projects/{projectId}/variables        # 共有変数更新
GET    /api/v1/projects/{projectId}/versions         # バージョン一覧

# Steps
POST   /api/v1/projects/{projectId}/steps            # 作成
GET    /api/v1/projects/{projectId}/steps            # 一覧
PUT    /api/v1/projects/{projectId}/steps/{stepId}   # 更新
DELETE /api/v1/projects/{projectId}/steps/{stepId}   # 削除
PUT    /api/v1/projects/{projectId}/steps/{stepId}/trigger  # トリガー設定

# Edges
POST   /api/v1/projects/{projectId}/edges            # 作成
GET    /api/v1/projects/{projectId}/edges            # 一覧
DELETE /api/v1/projects/{projectId}/edges/{edgeId}   # 削除

# BlockGroups
POST   /api/v1/projects/{projectId}/block-groups
GET    /api/v1/projects/{projectId}/block-groups
PUT    /api/v1/projects/{projectId}/block-groups/{groupId}
DELETE /api/v1/projects/{projectId}/block-groups/{groupId}

# Runs
POST   /api/v1/projects/{projectId}/runs             # 実行（startStepIdを指定）
GET    /api/v1/projects/{projectId}/runs             # 履歴
GET    /api/v1/runs/{runId}                          # 詳細
POST   /api/v1/runs/{runId}/cancel                   # キャンセル

# Schedules
POST   /api/v1/projects/{projectId}/schedules        # 作成（startStepId必須）
GET    /api/v1/projects/{projectId}/schedules        # 一覧
PUT    /api/v1/schedules/{scheduleId}                # 更新
DELETE /api/v1/schedules/{scheduleId}                # 削除

# Webhook受信
POST   /api/v1/webhooks/projects/{projectId}/steps/{startStepId}
```

### 削除ファイル

- `handler/workflow.go` → 削除
- `handler/webhook.go` → 削除（Webhook受信はproject.goに統合）

---

## Phase 6: Backend Engine

### 変更: engine/executor.go

```go
type ExecutionContext struct {
    Run         *domain.Run
    Project     *domain.Project           // Workflow → Project
    Definition  *domain.ProjectDefinition // WorkflowDefinition → ProjectDefinition
    StartStepID uuid.UUID                 // どのStartから開始するか
    // ... 他は変更なし
}

// Execute - 指定されたStartStepから実行
func (e *Executor) Execute(ctx context.Context, execCtx *ExecutionContext) error {
    // StartStepIDで指定されたStartブロックから開始
}

// findStartNodes - 全Startブロックではなく、指定されたStartのみ返す
func (e *Executor) findStartNodes(execCtx *ExecutionContext) []uuid.UUID {
    return []uuid.UUID{execCtx.StartStepID}
}
```

### 変更: engine/queue.go

```go
type Job struct {
    TenantID    uuid.UUID
    ProjectID   uuid.UUID  // WorkflowID → ProjectID
    StartStepID uuid.UUID  // 追加
    RunID       uuid.UUID
    Mode        string
    Priority    int
}
```

---

## Phase 7: Frontend

### 型定義の変更

```typescript
// types/api.ts

export type ProjectStatus = 'draft' | 'published'

export interface Project {
  id: string
  tenant_id: string
  name: string
  description?: string
  status: ProjectStatus
  version: number
  variables?: Record<string, unknown>
  draft?: ProjectDraft
  has_draft: boolean
  is_system: boolean
  system_slug?: string
  created_by?: string
  published_at?: string
  created_at: string
  updated_at: string

  // Relations
  steps?: Step[]
  edges?: Edge[]
  block_groups?: BlockGroup[]
}

export type TriggerType =
  | 'manual'
  | 'webhook'
  | 'schedule'
  | 'slack'
  | 'discord'
  | 'email'
  | 'internal'
  | 'api'

export interface Step {
  id: string
  tenant_id: string
  project_id: string      // workflow_id → project_id
  name: string
  type: StepType
  config: Record<string, unknown>

  // Start専用
  trigger_type?: TriggerType
  trigger_config?: TriggerConfig

  // ... 他は変更なし
}

export interface Edge {
  id: string
  tenant_id: string
  project_id: string      // workflow_id → project_id
  // ... 他は変更なし
}

export interface BlockGroup {
  id: string
  tenant_id: string
  project_id: string      // workflow_id → project_id
  // ... 他は変更なし
}

export interface Run {
  id: string
  tenant_id: string
  project_id: string        // workflow_id → project_id
  project_version: number   // workflow_version → project_version
  start_step_id?: string    // 追加
  // ... 他は変更なし
}
```

### Composables

```typescript
// composables/useProjects.ts (旧useWorkflows.ts)

export function useProjects() {
  const api = useApi()

  return {
    // Project CRUD
    list(params?: { status?: string; search?: string; page?: number; limit?: number }),
    get(projectId: string),
    create(data: CreateProjectRequest),
    update(projectId: string, data: UpdateProjectRequest),
    delete(projectId: string),

    // Draft
    saveDraft(projectId: string, draft: ProjectDraft),
    publish(projectId: string),
    discardDraft(projectId: string),

    // Variables
    updateVariables(projectId: string, variables: Record<string, unknown>),

    // Versions
    listVersions(projectId: string),
    getVersion(projectId: string, version: number),

    // Steps
    listSteps(projectId: string),
    createStep(projectId: string, step: CreateStepRequest),
    updateStep(projectId: string, stepId: string, data: UpdateStepRequest),
    deleteStep(projectId: string, stepId: string),
    updateStepTrigger(projectId: string, stepId: string, trigger: TriggerConfig),

    // Edges
    listEdges(projectId: string),
    createEdge(projectId: string, edge: CreateEdgeRequest),
    deleteEdge(projectId: string, edgeId: string),

    // BlockGroups
    listBlockGroups(projectId: string),
    createBlockGroup(projectId: string, group: CreateBlockGroupRequest),
    updateBlockGroup(projectId: string, groupId: string, data: UpdateBlockGroupRequest),
    deleteBlockGroup(projectId: string, groupId: string),

    // Runs
    createRun(projectId: string, data: { start_step_id: string; input?: unknown; mode?: string }),
    listRuns(projectId: string, params?: { page?: number; limit?: number }),

    // Schedules
    listSchedules(projectId: string),
    createSchedule(projectId: string, data: CreateScheduleRequest),
  }
}
```

### ページ構成

| URL | 画面 | 説明 |
|-----|------|------|
| `/projects` | プロジェクト一覧 | カード/リスト表示 |
| `/projects/[projectId]` | プロジェクトエディタ | 統合エディタ（タブ: Canvas, Runs, Schedules, Variables） |
| `/runs/[runId]` | 実行詳細 | 実行ログ表示 |
| `/admin/blocks` | ブロック管理 | |
| `/admin/tenants` | テナント管理 | |

### 削除ファイル

- `pages/workflows/*` → 削除
- `composables/useWorkflows.ts` → 削除
- `composables/useWebhooks.ts` → 削除

### 新規コンポーネント

| コンポーネント | 説明 |
|---------------|------|
| `ProjectCard.vue` | プロジェクト一覧のカード |
| `ProjectRunsTab.vue` | Runsタブ |
| `ProjectSchedulesTab.vue` | Schedulesタブ |
| `ProjectVariablesTab.vue` | Variablesタブ |
| `TriggerConfigPanel.vue` | Startブロックのトリガー設定パネル |
| `TriggerBadge.vue` | トリガー種別バッジ |

### 変更コンポーネント

| コンポーネント | 変更内容 |
|---------------|----------|
| `DagEditor.vue` | Startブロックの表示にトリガーバッジ追加 |
| `PropertiesPanel.vue` | Startブロック選択時にTrigger設定タブ追加 |
| `StepPalette.vue` | 変更なし |

---

## Phase 8: ドキュメント更新

| ドキュメント | 更新内容 |
|-------------|----------|
| `docs/INDEX.md` | Project/マルチスタート概念の説明追加 |
| `docs/API.md` | 全APIエンドポイント書き換え |
| `docs/DATABASE.md` | テーブル構造全面書き換え |
| `docs/BACKEND.md` | Domain/Repository/Usecase更新 |
| `docs/FRONTEND.md` | ページ構成・Composables更新 |
| `docs/openapi.yaml` | OpenAPI仕様全面書き換え |
| `CLAUDE.md` | API Quick Test例の更新 |
| `docs/BLOCK_REGISTRY.md` | Startブロックのtrigger設定説明追加 |
| `docs/TESTING.md` | テストデータ構造の更新 |

---

## 実装順序

```
Week 1: DB + Backend Domain/Repository
├── schema.sql 書き換え
├── domain/project.go 作成
├── domain/step.go, edge.go, block_group.go, run.go 変更
├── repository/interfaces.go 変更
├── repository/postgres/project.go 作成
├── repository/postgres/project_version.go 作成
├── 既存repository の workflowID → projectID 変更
└── domain/workflow.go, repository/postgres/workflow.go 削除

Week 2: Backend Usecase/Handler/Engine
├── usecase/project.go 作成
├── 既存usecase の workflowID → projectID 変更
├── handler/project.go 作成
├── 既存handler の workflowID → projectID 変更
├── engine/executor.go 変更
├── engine/queue.go 変更
├── usecase/workflow.go, handler/workflow.go 削除
└── cmd/api/main.go ルーティング変更

Week 3: Frontend
├── types/api.ts 変更
├── composables/useProjects.ts 作成
├── 既存composables の workflowId → projectId 変更
├── pages/projects/index.vue 作成
├── pages/projects/[projectId].vue 作成（統合エディタ）
├── components/project/* 作成
├── DagEditor, PropertiesPanel のStartブロック対応
├── pages/workflows/* 削除
└── composables/useWorkflows.ts 削除

Week 4: ドキュメント + テスト
├── schema/seed.sql 更新
├── docs/*.md 更新
├── E2Eテスト更新
└── 動作確認
```

---

## 変更規模サマリー

| カテゴリ | 新規 | 変更 | 削除 |
|----------|------|------|------|
| **DB テーブル** | 2 (projects, project_versions) | 10 | 3 (workflows, workflow_versions, webhooks) |
| **Backend ファイル** | 4 | 15+ | 4 |
| **Frontend ファイル** | 8 | 10+ | 5 |
| **ドキュメント** | 1 (本設計書) | 8+ | 0 |
