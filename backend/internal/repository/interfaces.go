package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
)

// WorkflowRepository defines the interface for workflow persistence
type WorkflowRepository interface {
	Create(ctx context.Context, workflow *domain.Workflow) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Workflow, error)
	List(ctx context.Context, tenantID uuid.UUID, filter WorkflowFilter) ([]*domain.Workflow, int, error)
	Update(ctx context.Context, workflow *domain.Workflow) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	GetWithStepsAndEdges(ctx context.Context, tenantID, id uuid.UUID) (*domain.Workflow, error)
}

// WorkflowFilter defines filtering options for workflow list
type WorkflowFilter struct {
	Status *domain.WorkflowStatus
	Page   int
	Limit  int
}

// StepRepository defines the interface for step persistence
type StepRepository interface {
	Create(ctx context.Context, step *domain.Step) error
	GetByID(ctx context.Context, workflowID, id uuid.UUID) (*domain.Step, error)
	ListByWorkflow(ctx context.Context, workflowID uuid.UUID) ([]*domain.Step, error)
	Update(ctx context.Context, step *domain.Step) error
	Delete(ctx context.Context, workflowID, id uuid.UUID) error
}

// EdgeRepository defines the interface for edge persistence
type EdgeRepository interface {
	Create(ctx context.Context, edge *domain.Edge) error
	GetByID(ctx context.Context, workflowID, id uuid.UUID) (*domain.Edge, error)
	ListByWorkflow(ctx context.Context, workflowID uuid.UUID) ([]*domain.Edge, error)
	Delete(ctx context.Context, workflowID, id uuid.UUID) error
	Exists(ctx context.Context, workflowID, sourceID, targetID uuid.UUID) (bool, error)
}

// RunRepository defines the interface for run persistence
type RunRepository interface {
	Create(ctx context.Context, run *domain.Run) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Run, error)
	ListByWorkflow(ctx context.Context, tenantID, workflowID uuid.UUID, filter RunFilter) ([]*domain.Run, int, error)
	Update(ctx context.Context, run *domain.Run) error
	GetWithStepRuns(ctx context.Context, tenantID, id uuid.UUID) (*domain.Run, error)
}

// RunFilter defines filtering options for run list
type RunFilter struct {
	Status *domain.RunStatus
	Mode   *domain.RunMode
	Page   int
	Limit  int
}

// StepRunRepository defines the interface for step run persistence
type StepRunRepository interface {
	Create(ctx context.Context, stepRun *domain.StepRun) error
	GetByID(ctx context.Context, runID, id uuid.UUID) (*domain.StepRun, error)
	ListByRun(ctx context.Context, runID uuid.UUID) ([]*domain.StepRun, error)
	Update(ctx context.Context, stepRun *domain.StepRun) error
}

// WorkflowVersionRepository defines the interface for workflow version persistence
type WorkflowVersionRepository interface {
	Create(ctx context.Context, version *domain.WorkflowVersion) error
	GetByWorkflowAndVersion(ctx context.Context, workflowID uuid.UUID, version int) (*domain.WorkflowVersion, error)
	GetLatestByWorkflow(ctx context.Context, workflowID uuid.UUID) (*domain.WorkflowVersion, error)
	ListByWorkflow(ctx context.Context, workflowID uuid.UUID) ([]*domain.WorkflowVersion, error)
}

// ScheduleRepository defines the interface for schedule persistence
type ScheduleRepository interface {
	Create(ctx context.Context, schedule *domain.Schedule) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Schedule, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID, filter ScheduleFilter) ([]*domain.Schedule, int, error)
	ListByWorkflow(ctx context.Context, tenantID, workflowID uuid.UUID) ([]*domain.Schedule, error)
	Update(ctx context.Context, schedule *domain.Schedule) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	// GetDueSchedules returns schedules that are due to run
	GetDueSchedules(ctx context.Context, limit int) ([]*domain.Schedule, error)
}

// ScheduleFilter defines filtering options for schedule list
type ScheduleFilter struct {
	WorkflowID *uuid.UUID
	Status     *domain.ScheduleStatus
	Page       int
	Limit      int
}

// WebhookRepository defines the interface for webhook persistence
type WebhookRepository interface {
	Create(ctx context.Context, webhook *domain.Webhook) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Webhook, error)
	// GetByIDForTrigger retrieves webhook by ID without tenant check (for public trigger endpoint)
	GetByIDForTrigger(ctx context.Context, id uuid.UUID) (*domain.Webhook, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID, filter WebhookFilter) ([]*domain.Webhook, int, error)
	ListByWorkflow(ctx context.Context, tenantID, workflowID uuid.UUID) ([]*domain.Webhook, error)
	Update(ctx context.Context, webhook *domain.Webhook) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

// WebhookFilter defines filtering options for webhook list
type WebhookFilter struct {
	WorkflowID *uuid.UUID
	Enabled    *bool
	Page       int
	Limit      int
}

// AuditLogRepository defines the interface for audit log persistence
type AuditLogRepository interface {
	Create(ctx context.Context, log *domain.AuditLog) error
	ListByTenant(ctx context.Context, tenantID uuid.UUID, filter AuditLogFilter) ([]*domain.AuditLog, int, error)
	ListByResource(ctx context.Context, tenantID uuid.UUID, resourceType domain.AuditResourceType, resourceID uuid.UUID) ([]*domain.AuditLog, error)
}

// AuditLogFilter defines filtering options for audit log list
type AuditLogFilter struct {
	ActorID      *uuid.UUID
	Action       *domain.AuditAction
	ResourceType *domain.AuditResourceType
	ResourceID   *uuid.UUID
	StartTime    *time.Time
	EndTime      *time.Time
	Page         int
	Limit        int
}

// BlockDefinitionRepository defines the interface for block definition persistence
type BlockDefinitionRepository interface {
	// Create creates a new block definition
	Create(ctx context.Context, block *domain.BlockDefinition) error
	// GetByID retrieves a block definition by ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.BlockDefinition, error)
	// GetBySlug retrieves a block definition by slug (tenant-specific or system)
	GetBySlug(ctx context.Context, tenantID *uuid.UUID, slug string) (*domain.BlockDefinition, error)
	// List retrieves block definitions with optional filtering
	List(ctx context.Context, tenantID *uuid.UUID, filter BlockDefinitionFilter) ([]*domain.BlockDefinition, error)
	// Update updates a block definition
	Update(ctx context.Context, block *domain.BlockDefinition) error
	// Delete deletes a block definition (only custom blocks)
	Delete(ctx context.Context, id uuid.UUID) error
}

// BlockDefinitionFilter defines filtering options for block definition list
type BlockDefinitionFilter struct {
	Category    *domain.BlockCategory
	ExecutorType *domain.ExecutorType
	EnabledOnly bool
	SystemOnly  bool // If true, only return system blocks (tenant_id IS NULL)
}

// BlockGroupRepository defines the interface for block group persistence
type BlockGroupRepository interface {
	Create(ctx context.Context, group *domain.BlockGroup) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.BlockGroup, error)
	ListByWorkflow(ctx context.Context, workflowID uuid.UUID) ([]*domain.BlockGroup, error)
	ListByParent(ctx context.Context, parentID uuid.UUID) ([]*domain.BlockGroup, error)
	Update(ctx context.Context, group *domain.BlockGroup) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// BlockGroupRunRepository defines the interface for block group run persistence
type BlockGroupRunRepository interface {
	Create(ctx context.Context, run *domain.BlockGroupRun) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.BlockGroupRun, error)
	ListByRun(ctx context.Context, runID uuid.UUID) ([]*domain.BlockGroupRun, error)
	Update(ctx context.Context, run *domain.BlockGroupRun) error
}
