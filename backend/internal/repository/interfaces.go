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

	// GetMaxAttempt returns the highest attempt number for a step in a run
	GetMaxAttempt(ctx context.Context, runID, stepID uuid.UUID) (int, error)
	// GetMaxAttemptForRun returns the highest attempt number across all steps in a run
	GetMaxAttemptForRun(ctx context.Context, runID uuid.UUID) (int, error)
	// GetLatestByStep returns the most recent StepRun for a step in a run
	GetLatestByStep(ctx context.Context, runID, stepID uuid.UUID) (*domain.StepRun, error)
	// ListCompletedByRun returns the latest completed StepRun for each step in a run
	ListCompletedByRun(ctx context.Context, runID uuid.UUID) ([]*domain.StepRun, error)
	// ListByStep returns all StepRuns for a specific step in a run (for history)
	ListByStep(ctx context.Context, runID, stepID uuid.UUID) ([]*domain.StepRun, error)
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

// CredentialRepository defines the interface for credential persistence
type CredentialRepository interface {
	// Create creates a new credential
	Create(ctx context.Context, credential *domain.Credential) error
	// GetByID retrieves a credential by ID
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Credential, error)
	// GetByName retrieves a credential by name within a tenant
	GetByName(ctx context.Context, tenantID uuid.UUID, name string) (*domain.Credential, error)
	// List retrieves credentials with optional filtering
	List(ctx context.Context, tenantID uuid.UUID, filter CredentialFilter) ([]*domain.Credential, int, error)
	// Update updates a credential
	Update(ctx context.Context, credential *domain.Credential) error
	// Delete deletes a credential
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	// UpdateStatus updates the status of a credential
	UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.CredentialStatus) error
}

// CredentialFilter defines filtering options for credential list
type CredentialFilter struct {
	CredentialType *domain.CredentialType
	Status         *domain.CredentialStatus
	Page           int
	Limit          int
}

// SystemCredentialRepository defines the interface for system credential persistence
// System credentials are operator-managed and used by system blocks (not accessible by tenants)
type SystemCredentialRepository interface {
	// Create creates a new system credential
	Create(ctx context.Context, credential *domain.SystemCredential) error
	// GetByID retrieves a system credential by ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.SystemCredential, error)
	// GetByName retrieves a system credential by name
	GetByName(ctx context.Context, name string) (*domain.SystemCredential, error)
	// List retrieves all system credentials
	List(ctx context.Context) ([]*domain.SystemCredential, error)
	// ListByType retrieves system credentials by type
	ListByType(ctx context.Context, credType domain.CredentialType) ([]*domain.SystemCredential, error)
	// Update updates a system credential
	Update(ctx context.Context, credential *domain.SystemCredential) error
	// Delete deletes a system credential
	Delete(ctx context.Context, id uuid.UUID) error
	// UpdateStatus updates the status of a system credential
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.CredentialStatus) error
}

// BlockTemplateRepository defines the interface for block template persistence
type BlockTemplateRepository interface {
	// Create creates a new block template
	Create(ctx context.Context, template *domain.BlockTemplate) error
	// GetByID retrieves a block template by ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.BlockTemplate, error)
	// GetBySlug retrieves a block template by slug
	GetBySlug(ctx context.Context, slug string) (*domain.BlockTemplate, error)
	// List retrieves all block templates
	List(ctx context.Context) ([]*domain.BlockTemplate, error)
	// Update updates a block template (only non-builtin templates)
	Update(ctx context.Context, template *domain.BlockTemplate) error
	// Delete deletes a block template (only non-builtin templates)
	Delete(ctx context.Context, id uuid.UUID) error
}

// CopilotSessionRepository defines the interface for copilot session persistence
type CopilotSessionRepository interface {
	// Create creates a new copilot session
	Create(ctx context.Context, session *domain.CopilotSession) error
	// GetByID retrieves a copilot session by ID
	GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.CopilotSession, error)
	// GetActiveByUserAndWorkflow retrieves the active session for a user and workflow
	GetActiveByUserAndWorkflow(ctx context.Context, tenantID uuid.UUID, userID string, workflowID uuid.UUID) (*domain.CopilotSession, error)
	// GetWithMessages retrieves a session with all its messages
	GetWithMessages(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.CopilotSession, error)
	// ListByUserAndWorkflow retrieves all sessions for a user and workflow
	ListByUserAndWorkflow(ctx context.Context, tenantID uuid.UUID, userID string, workflowID uuid.UUID) ([]*domain.CopilotSession, error)
	// Update updates a copilot session
	Update(ctx context.Context, session *domain.CopilotSession) error
	// AddMessage adds a message to a session
	AddMessage(ctx context.Context, message *domain.CopilotMessage) error
	// CloseSession marks a session as inactive
	CloseSession(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error
}

// UsageRepository defines the interface for usage record persistence
type UsageRepository interface {
	// Create creates a new usage record
	Create(ctx context.Context, record *domain.UsageRecord) error
	// GetByID retrieves a usage record by ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.UsageRecord, error)
	// GetSummary retrieves aggregated usage summary for a tenant
	GetSummary(ctx context.Context, tenantID uuid.UUID, period string) (*domain.UsageSummary, error)
	// GetDaily retrieves daily usage data for a date range
	GetDaily(ctx context.Context, tenantID uuid.UUID, start, end time.Time) ([]domain.DailyUsage, error)
	// GetByWorkflow retrieves usage data grouped by workflow
	GetByWorkflow(ctx context.Context, tenantID uuid.UUID, period string) ([]domain.WorkflowUsage, error)
	// GetByModel retrieves usage data grouped by model
	GetByModel(ctx context.Context, tenantID uuid.UUID, period string) (map[string]domain.ModelUsage, error)
	// GetByRun retrieves all usage records for a specific run
	GetByRun(ctx context.Context, tenantID, runID uuid.UUID) ([]domain.UsageRecord, error)
	// AggregateDailyData aggregates raw usage data into daily aggregates
	AggregateDailyData(ctx context.Context, date time.Time) error
	// GetCurrentSpend retrieves current spend for budget checking
	GetCurrentSpend(ctx context.Context, tenantID uuid.UUID, workflowID *uuid.UUID, budgetType domain.BudgetType) (float64, error)
}

// BudgetRepository defines the interface for budget persistence
type BudgetRepository interface {
	// Create creates a new budget
	Create(ctx context.Context, budget *domain.UsageBudget) error
	// GetByID retrieves a budget by ID
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.UsageBudget, error)
	// List retrieves all budgets for a tenant
	List(ctx context.Context, tenantID uuid.UUID) ([]*domain.UsageBudget, error)
	// GetByWorkflow retrieves budget for a specific workflow
	GetByWorkflow(ctx context.Context, tenantID uuid.UUID, workflowID *uuid.UUID, budgetType domain.BudgetType) (*domain.UsageBudget, error)
	// Update updates a budget
	Update(ctx context.Context, budget *domain.UsageBudget) error
	// Delete deletes a budget
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}
