package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
)

// ProjectRepository defines the interface for project persistence
type ProjectRepository interface {
	Create(ctx context.Context, project *domain.Project) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Project, error)
	List(ctx context.Context, tenantID uuid.UUID, filter ProjectFilter) ([]*domain.Project, int, error)
	Update(ctx context.Context, project *domain.Project) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	GetWithStepsAndEdges(ctx context.Context, tenantID, id uuid.UUID) (*domain.Project, error)
	// GetSystemBySlug retrieves a system project by its slug (accessible across all tenants)
	GetSystemBySlug(ctx context.Context, slug string) (*domain.Project, error)
}

// ProjectFilter defines filtering options for project list
type ProjectFilter struct {
	Status *domain.ProjectStatus
	Page   int
	Limit  int
}

// StepRepository defines the interface for step persistence
type StepRepository interface {
	Create(ctx context.Context, step *domain.Step) error
	GetByID(ctx context.Context, tenantID, projectID, id uuid.UUID) (*domain.Step, error)
	ListByProject(ctx context.Context, tenantID, projectID uuid.UUID) ([]*domain.Step, error)
	ListByBlockGroup(ctx context.Context, tenantID, blockGroupID uuid.UUID) ([]*domain.Step, error)
	// ListStartSteps returns all Start blocks in a project
	ListStartSteps(ctx context.Context, tenantID, projectID uuid.UUID) ([]*domain.Step, error)
	// GetStartStepByTriggerType returns a Start block by its trigger type
	GetStartStepByTriggerType(ctx context.Context, tenantID, projectID uuid.UUID, triggerType domain.StepTriggerType) (*domain.Step, error)
	Update(ctx context.Context, step *domain.Step) error
	Delete(ctx context.Context, tenantID, projectID, id uuid.UUID) error
}

// EdgeRepository defines the interface for edge persistence
type EdgeRepository interface {
	Create(ctx context.Context, edge *domain.Edge) error
	GetByID(ctx context.Context, tenantID, projectID, id uuid.UUID) (*domain.Edge, error)
	ListByProject(ctx context.Context, tenantID, projectID uuid.UUID) ([]*domain.Edge, error)
	Delete(ctx context.Context, tenantID, projectID, id uuid.UUID) error
	Exists(ctx context.Context, tenantID, projectID, sourceID, targetID uuid.UUID) (bool, error)
}

// RunRepository defines the interface for run persistence
type RunRepository interface {
	Create(ctx context.Context, run *domain.Run) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Run, error)
	ListByProject(ctx context.Context, tenantID, projectID uuid.UUID, filter RunFilter) ([]*domain.Run, int, error)
	// ListByStartStep returns runs for a specific Start block
	ListByStartStep(ctx context.Context, tenantID, projectID, startStepID uuid.UUID, filter RunFilter) ([]*domain.Run, int, error)
	Update(ctx context.Context, run *domain.Run) error
	GetWithStepRuns(ctx context.Context, tenantID, id uuid.UUID) (*domain.Run, error)
}

// RunFilter defines filtering options for run list
type RunFilter struct {
	Status      *domain.RunStatus
	TriggeredBy *domain.TriggerType
	StartStepID *uuid.UUID // Filter by specific Start block
	Page        int
	Limit       int
}

// StepRunRepository defines the interface for step run persistence
type StepRunRepository interface {
	Create(ctx context.Context, stepRun *domain.StepRun) error
	GetByID(ctx context.Context, tenantID, runID, id uuid.UUID) (*domain.StepRun, error)
	ListByRun(ctx context.Context, tenantID, runID uuid.UUID) ([]*domain.StepRun, error)
	Update(ctx context.Context, stepRun *domain.StepRun) error

	// GetMaxAttempt returns the highest attempt number for a step in a run
	GetMaxAttempt(ctx context.Context, tenantID, runID, stepID uuid.UUID) (int, error)
	// GetMaxAttemptForRun returns the highest attempt number across all steps in a run
	GetMaxAttemptForRun(ctx context.Context, tenantID, runID uuid.UUID) (int, error)
	// GetMaxSequenceNumberForRun returns the highest sequence number across all steps in a run
	GetMaxSequenceNumberForRun(ctx context.Context, tenantID, runID uuid.UUID) (int, error)
	// GetLatestByStep returns the most recent StepRun for a step in a run
	GetLatestByStep(ctx context.Context, tenantID, runID, stepID uuid.UUID) (*domain.StepRun, error)
	// ListCompletedByRun returns the latest completed StepRun for each step in a run
	ListCompletedByRun(ctx context.Context, tenantID, runID uuid.UUID) ([]*domain.StepRun, error)
	// ListByStep returns all StepRuns for a specific step in a run (for history)
	ListByStep(ctx context.Context, tenantID, runID, stepID uuid.UUID) ([]*domain.StepRun, error)
}

// ProjectVersionRepository defines the interface for project version persistence
type ProjectVersionRepository interface {
	Create(ctx context.Context, version *domain.ProjectVersion) error
	GetByProjectAndVersion(ctx context.Context, projectID uuid.UUID, version int) (*domain.ProjectVersion, error)
	GetLatestByProject(ctx context.Context, projectID uuid.UUID) (*domain.ProjectVersion, error)
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.ProjectVersion, error)
}

// ScheduleRepository defines the interface for schedule persistence
type ScheduleRepository interface {
	Create(ctx context.Context, schedule *domain.Schedule) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Schedule, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID, filter ScheduleFilter) ([]*domain.Schedule, int, error)
	ListByProject(ctx context.Context, tenantID, projectID uuid.UUID) ([]*domain.Schedule, error)
	// ListByStartStep returns schedules for a specific Start block
	ListByStartStep(ctx context.Context, tenantID, projectID, startStepID uuid.UUID) ([]*domain.Schedule, error)
	Update(ctx context.Context, schedule *domain.Schedule) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	// GetDueSchedules returns schedules that are due to run
	GetDueSchedules(ctx context.Context, limit int) ([]*domain.Schedule, error)
}

// ScheduleFilter defines filtering options for schedule list
type ScheduleFilter struct {
	ProjectID   *uuid.UUID
	StartStepID *uuid.UUID // Filter by specific Start block
	Status      *domain.ScheduleStatus
	Page        int
	Limit       int
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
	// ValidateInheritance validates that a block can inherit from the specified parent
	// Checks for circular inheritance, inheritance depth, and parent inheritability
	ValidateInheritance(ctx context.Context, blockID uuid.UUID, parentBlockID uuid.UUID) error
}

// BlockDefinitionFilter defines filtering options for block definition list
type BlockDefinitionFilter struct {
	Category    *domain.BlockCategory
	EnabledOnly bool
	SystemOnly  bool    // If true, only return system blocks (tenant_id IS NULL)
	IsSystem    *bool   // Filter by is_system flag
	Search      *string // Search by name or description
}

// BlockVersionRepository defines the interface for block version persistence
type BlockVersionRepository interface {
	// Create creates a new block version
	Create(ctx context.Context, version *domain.BlockVersion) error
	// GetByID retrieves a block version by ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.BlockVersion, error)
	// GetByBlockAndVersion retrieves a specific version of a block
	GetByBlockAndVersion(ctx context.Context, blockID uuid.UUID, version int) (*domain.BlockVersion, error)
	// ListByBlock retrieves all versions of a block
	ListByBlock(ctx context.Context, blockID uuid.UUID) ([]*domain.BlockVersion, error)
	// GetLatestByBlock retrieves the latest version of a block
	GetLatestByBlock(ctx context.Context, blockID uuid.UUID) (*domain.BlockVersion, error)
}

// BlockGroupRepository defines the interface for block group persistence
type BlockGroupRepository interface {
	Create(ctx context.Context, group *domain.BlockGroup) error
	GetByID(ctx context.Context, tenantID, projectID, id uuid.UUID) (*domain.BlockGroup, error)
	ListByProject(ctx context.Context, tenantID, projectID uuid.UUID) ([]*domain.BlockGroup, error)
	ListByParent(ctx context.Context, tenantID, projectID, parentID uuid.UUID) ([]*domain.BlockGroup, error)
	Update(ctx context.Context, group *domain.BlockGroup) error
	Delete(ctx context.Context, tenantID, projectID, id uuid.UUID) error
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
	Scope          *domain.OwnerScope // Filter by ownership scope
	ProjectID      *uuid.UUID         // Filter by project (for scope=project)
	OwnerUserID    *uuid.UUID         // Filter by owner user (for scope=personal)
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

// CopilotSessionRepository defines the interface for copilot session persistence
type CopilotSessionRepository interface {
	// Create creates a new copilot session
	Create(ctx context.Context, session *domain.CopilotSession) error
	// GetByID retrieves a copilot session by ID
	GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.CopilotSession, error)
	// GetWithMessages retrieves a session with all its messages
	GetWithMessages(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.CopilotSession, error)
	// GetActiveByUser retrieves the most recent active session for a user (global, no project context)
	GetActiveByUser(ctx context.Context, tenantID uuid.UUID, userID string) (*domain.CopilotSession, error)
	// GetActiveByUserAndProject retrieves the active session for a user and project
	GetActiveByUserAndProject(ctx context.Context, tenantID uuid.UUID, userID string, projectID uuid.UUID) (*domain.CopilotSession, error)
	// ListByUser retrieves all sessions for a user (global, no project context)
	ListByUser(ctx context.Context, tenantID uuid.UUID, userID string, filter CopilotSessionFilter) ([]*domain.CopilotSession, int, error)
	// ListByUserAndProject retrieves all sessions for a user and project
	ListByUserAndProject(ctx context.Context, tenantID uuid.UUID, userID string, projectID uuid.UUID) ([]*domain.CopilotSession, error)
	// Update updates a copilot session
	Update(ctx context.Context, session *domain.CopilotSession) error
	// AddMessage adds a message to a session
	AddMessage(ctx context.Context, message *domain.CopilotMessage) error
	// UpdateStatus updates the session status
	UpdateStatus(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, status domain.CopilotSessionStatus) error
	// UpdatePhase updates the hearing phase and progress
	UpdatePhase(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, phase domain.CopilotPhase, progress int) error
	// SetSpec sets the workflow spec
	SetSpec(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, spec []byte) error
	// SetProjectID sets the generated project ID
	SetProjectID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, projectID uuid.UUID) error
	// Delete deletes a copilot session
	Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error
}

// CopilotSessionFilter defines filtering options for copilot session list
type CopilotSessionFilter struct {
	Status *domain.CopilotSessionStatus
	Page   int
	Limit  int
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
	// GetByProject retrieves usage data grouped by project
	GetByProject(ctx context.Context, tenantID uuid.UUID, period string) ([]domain.ProjectUsage, error)
	// GetByModel retrieves usage data grouped by model
	GetByModel(ctx context.Context, tenantID uuid.UUID, period string) (map[string]domain.ModelUsage, error)
	// GetByRun retrieves all usage records for a specific run
	GetByRun(ctx context.Context, tenantID, runID uuid.UUID) ([]domain.UsageRecord, error)
	// AggregateDailyData aggregates raw usage data into daily aggregates
	AggregateDailyData(ctx context.Context, date time.Time) error
	// GetCurrentSpend retrieves current spend for budget checking
	GetCurrentSpend(ctx context.Context, tenantID uuid.UUID, projectID *uuid.UUID, budgetType domain.BudgetType) (float64, error)
}

// BudgetRepository defines the interface for budget persistence
type BudgetRepository interface {
	// Create creates a new budget
	Create(ctx context.Context, budget *domain.UsageBudget) error
	// GetByID retrieves a budget by ID
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.UsageBudget, error)
	// List retrieves all budgets for a tenant
	List(ctx context.Context, tenantID uuid.UUID) ([]*domain.UsageBudget, error)
	// GetByProject retrieves budget for a specific project
	GetByProject(ctx context.Context, tenantID uuid.UUID, projectID *uuid.UUID, budgetType domain.BudgetType) (*domain.UsageBudget, error)
	// Update updates a budget
	Update(ctx context.Context, budget *domain.UsageBudget) error
	// Delete deletes a budget
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

// TenantRepository defines the interface for tenant persistence
type TenantRepository interface {
	// Create creates a new tenant
	Create(ctx context.Context, tenant *domain.Tenant) error
	// GetByID retrieves a tenant by ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error)
	// GetBySlug retrieves a tenant by slug
	GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error)
	// List retrieves tenants with optional filtering
	List(ctx context.Context, filter TenantFilter) ([]*domain.Tenant, int, error)
	// Update updates a tenant
	Update(ctx context.Context, tenant *domain.Tenant) error
	// Delete soft-deletes a tenant
	Delete(ctx context.Context, id uuid.UUID) error
	// UpdateStatus updates the status of a tenant
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.TenantStatus, reason string) error
	// GetStats retrieves aggregated statistics for a tenant
	GetStats(ctx context.Context, id uuid.UUID) (*domain.TenantStats, error)
	// GetAllStats retrieves aggregated statistics for all tenants
	GetAllStats(ctx context.Context) (map[uuid.UUID]*domain.TenantStats, error)
}

// TenantFilter defines filtering options for tenant list
type TenantFilter struct {
	Status         *domain.TenantStatus
	Plan           *domain.TenantPlan
	Search         string
	Page           int
	Limit          int
	IncludeDeleted bool
}


// UserRepository defines the interface for user persistence
type UserRepository interface {
	// GetByID retrieves a user by ID
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.User, error)
	// GetVariables retrieves only the variables for a user
	GetVariables(ctx context.Context, tenantID, id uuid.UUID) (map[string]interface{}, error)
	// UpdateVariables updates only the variables for a user
	UpdateVariables(ctx context.Context, tenantID, id uuid.UUID, variables map[string]interface{}) error
}

// ============================================================================
// OAuth2 Repositories
// ============================================================================

// OAuth2ProviderRepository defines the interface for OAuth2 provider persistence
type OAuth2ProviderRepository interface {
	// GetByID retrieves an OAuth2 provider by ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.OAuth2Provider, error)
	// GetBySlug retrieves an OAuth2 provider by slug
	GetBySlug(ctx context.Context, slug string) (*domain.OAuth2Provider, error)
	// List retrieves all OAuth2 providers
	List(ctx context.Context) ([]*domain.OAuth2Provider, error)
	// ListPresets retrieves all preset OAuth2 providers
	ListPresets(ctx context.Context) ([]*domain.OAuth2Provider, error)
	// Create creates a new OAuth2 provider
	Create(ctx context.Context, provider *domain.OAuth2Provider) error
	// Update updates an OAuth2 provider
	Update(ctx context.Context, provider *domain.OAuth2Provider) error
	// Delete deletes an OAuth2 provider
	Delete(ctx context.Context, id uuid.UUID) error
}

// OAuth2AppRepository defines the interface for OAuth2 app persistence
type OAuth2AppRepository interface {
	// GetByID retrieves an OAuth2 app by ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.OAuth2App, error)
	// GetByTenantAndProvider retrieves an OAuth2 app by tenant and provider
	GetByTenantAndProvider(ctx context.Context, tenantID, providerID uuid.UUID) (*domain.OAuth2App, error)
	// ListByTenant retrieves all OAuth2 apps for a tenant
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.OAuth2App, error)
	// Create creates a new OAuth2 app
	Create(ctx context.Context, app *domain.OAuth2App) error
	// Update updates an OAuth2 app
	Update(ctx context.Context, app *domain.OAuth2App) error
	// Delete deletes an OAuth2 app
	Delete(ctx context.Context, id uuid.UUID) error
}

// OAuth2ConnectionRepository defines the interface for OAuth2 connection persistence
type OAuth2ConnectionRepository interface {
	// GetByID retrieves an OAuth2 connection by ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.OAuth2Connection, error)
	// GetByCredentialID retrieves an OAuth2 connection by credential ID
	GetByCredentialID(ctx context.Context, credentialID uuid.UUID) (*domain.OAuth2Connection, error)
	// GetByState retrieves an OAuth2 connection by state (for callback)
	GetByState(ctx context.Context, state string) (*domain.OAuth2Connection, error)
	// ListByApp retrieves all OAuth2 connections for an app
	ListByApp(ctx context.Context, oauth2AppID uuid.UUID) ([]*domain.OAuth2Connection, error)
	// ListExpiring retrieves connections that will expire within the given duration
	ListExpiring(ctx context.Context, within time.Duration) ([]*domain.OAuth2Connection, error)
	// Create creates a new OAuth2 connection
	Create(ctx context.Context, connection *domain.OAuth2Connection) error
	// Update updates an OAuth2 connection
	Update(ctx context.Context, connection *domain.OAuth2Connection) error
	// Delete deletes an OAuth2 connection
	Delete(ctx context.Context, id uuid.UUID) error
}

// ============================================================================
// Credential Share Repository
// ============================================================================

// CredentialShareRepository defines the interface for credential share persistence
type CredentialShareRepository interface {
	// GetByID retrieves a credential share by ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.CredentialShare, error)
	// ListByCredential retrieves all shares for a credential
	ListByCredential(ctx context.Context, credentialID uuid.UUID) ([]*domain.CredentialShare, error)
	// ListByUser retrieves all shares with a specific user
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.CredentialShare, error)
	// ListByProject retrieves all shares with a specific project
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.CredentialShare, error)
	// GetByCredentialAndUser retrieves a share by credential and user
	GetByCredentialAndUser(ctx context.Context, credentialID, userID uuid.UUID) (*domain.CredentialShare, error)
	// GetByCredentialAndProject retrieves a share by credential and project
	GetByCredentialAndProject(ctx context.Context, credentialID, projectID uuid.UUID) (*domain.CredentialShare, error)
	// Create creates a new credential share
	Create(ctx context.Context, share *domain.CredentialShare) error
	// Update updates a credential share
	Update(ctx context.Context, share *domain.CredentialShare) error
	// Delete deletes a credential share
	Delete(ctx context.Context, id uuid.UUID) error
	// DeleteExpired deletes all expired shares
	DeleteExpired(ctx context.Context) (int, error)
}

// CredentialAccessFilter defines filtering options for available credentials
type CredentialAccessFilter struct {
	TenantID       uuid.UUID
	UserID         uuid.UUID
	ProjectID      *uuid.UUID
	CredentialType *domain.CredentialType
	RequiredScope  *domain.CredentialScope // system or tenant
}

// ============================================================================
// N8N-Style Feature Repositories (Phase 2-4)
// ============================================================================

// AgentMemoryRepository defines the interface for agent memory persistence
type AgentMemoryRepository interface {
	Create(ctx context.Context, memory *domain.AgentMemory) error
	CreateBatch(ctx context.Context, memories []*domain.AgentMemory) error
	GetByRunAndStep(ctx context.Context, runID, stepID uuid.UUID) ([]*domain.AgentMemory, error)
	GetLastNByRunAndStep(ctx context.Context, runID, stepID uuid.UUID, n int) ([]*domain.AgentMemory, error)
	GetNextSequenceNumber(ctx context.Context, runID, stepID uuid.UUID) (int, error)
	DeleteByRunAndStep(ctx context.Context, runID, stepID uuid.UUID) error
	DeleteByRun(ctx context.Context, runID uuid.UUID) error
}

// AgentChatSessionRepository defines the interface for agent chat session persistence
type AgentChatSessionRepository interface {
	Create(ctx context.Context, session *domain.AgentChatSession) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.AgentChatSession, error)
	ListByProject(ctx context.Context, projectID uuid.UUID, filter AgentChatSessionFilter) ([]*domain.AgentChatSession, int, error)
	ListByUser(ctx context.Context, userID string, filter AgentChatSessionFilter) ([]*domain.AgentChatSession, int, error)
	Update(ctx context.Context, session *domain.AgentChatSession) error
	Close(ctx context.Context, id uuid.UUID) error
}

// AgentChatSessionFilter defines filtering options for agent chat session list
type AgentChatSessionFilter struct {
	Status *domain.AgentChatSessionStatus
	Page   int
	Limit  int
}

// ProjectTemplateRepository defines the interface for project template persistence
type ProjectTemplateRepository interface {
	Create(ctx context.Context, template *domain.ProjectTemplate) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.ProjectTemplate, error)
	List(ctx context.Context, filter TemplateFilter) ([]*domain.ProjectTemplate, int, error)
	ListPublic(ctx context.Context, filter TemplateFilter) ([]*domain.ProjectTemplate, int, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID, filter TemplateFilter) ([]*domain.ProjectTemplate, int, error)
	Update(ctx context.Context, template *domain.ProjectTemplate) error
	Delete(ctx context.Context, id uuid.UUID) error
	IncrementDownloadCount(ctx context.Context, id uuid.UUID) error
}

// TemplateFilter defines filtering options for template list
type TemplateFilter struct {
	Category   *string
	Tags       []string
	Search     *string
	IsFeatured *bool
	MinRating  *float64
	Visibility *domain.TemplateVisibility
	Page       int
	Limit      int
}

// TemplateReviewRepository defines the interface for template review persistence
type TemplateReviewRepository interface {
	Create(ctx context.Context, review *domain.TemplateReview) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.TemplateReview, error)
	ListByTemplate(ctx context.Context, templateID uuid.UUID) ([]*domain.TemplateReview, error)
	GetByTemplateAndUser(ctx context.Context, templateID, userID uuid.UUID) (*domain.TemplateReview, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// ProjectGitSyncRepository defines the interface for git sync persistence
type ProjectGitSyncRepository interface {
	Create(ctx context.Context, gitSync *domain.ProjectGitSync) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.ProjectGitSync, error)
	GetByProject(ctx context.Context, projectID uuid.UUID) (*domain.ProjectGitSync, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.ProjectGitSync, error)
	Update(ctx context.Context, gitSync *domain.ProjectGitSync) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateLastSync(ctx context.Context, id uuid.UUID, commitSHA string) error
}

// CustomBlockPackageRepository defines the interface for custom block package persistence
type CustomBlockPackageRepository interface {
	Create(ctx context.Context, pkg *domain.CustomBlockPackage) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.CustomBlockPackage, error)
	GetByNameAndVersion(ctx context.Context, tenantID uuid.UUID, name, version string) (*domain.CustomBlockPackage, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID, filter BlockPackageFilter) ([]*domain.CustomBlockPackage, int, error)
	Update(ctx context.Context, pkg *domain.CustomBlockPackage) error
	Delete(ctx context.Context, id uuid.UUID) error
	Publish(ctx context.Context, id uuid.UUID) error
	Deprecate(ctx context.Context, id uuid.UUID) error
}

// BlockPackageFilter defines filtering options for block package list
type BlockPackageFilter struct {
	Status *domain.BlockPackageStatus
	Search *string
	Page   int
	Limit  int
}
