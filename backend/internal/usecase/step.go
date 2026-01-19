package usecase

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// StepUsecase handles step business logic
type StepUsecase struct {
	projectRepo     repository.ProjectRepository
	stepRepo        repository.StepRepository
	blockDefRepo    repository.BlockDefinitionRepository
	projectChecker  *ProjectChecker
}

// NewStepUsecase creates a new StepUsecase
func NewStepUsecase(
	projectRepo repository.ProjectRepository,
	stepRepo repository.StepRepository,
	blockDefRepo repository.BlockDefinitionRepository,
) *StepUsecase {
	return &StepUsecase{
		projectRepo:    projectRepo,
		stepRepo:       stepRepo,
		blockDefRepo:   blockDefRepo,
		projectChecker: NewProjectChecker(projectRepo),
	}
}

// CreateStepInput represents input for creating a step
type CreateStepInput struct {
	TenantID           uuid.UUID
	ProjectID          uuid.UUID
	Name               string
	Type               domain.StepType
	Config             json.RawMessage
	TriggerType        string          // For start blocks: manual, webhook, schedule, etc.
	TriggerConfig      json.RawMessage // Configuration for the trigger
	CredentialBindings json.RawMessage // Mapping of credential names to credential IDs
	PositionX          int
	PositionY          int
}

// Create creates a new step
func (u *StepUsecase) Create(ctx context.Context, input CreateStepInput) (*domain.Step, error) {
	// Verify project exists and is editable
	if _, err := u.projectChecker.CheckEditable(ctx, input.TenantID, input.ProjectID); err != nil {
		return nil, err
	}

	// Validate input
	if input.Name == "" {
		return nil, domain.NewValidationError("name", "name is required")
	}

	// Check if type is a built-in step type or a custom block definition
	var blockDef *domain.BlockDefinition
	if !input.Type.IsValid() {
		// Try to find as a custom block definition
		var err error
		blockDef, err = u.blockDefRepo.GetBySlug(ctx, &input.TenantID, string(input.Type))
		if err != nil || blockDef == nil {
			// Also try system blocks (tenant_id = NULL)
			blockDef, err = u.blockDefRepo.GetBySlug(ctx, nil, string(input.Type))
			if err != nil || blockDef == nil {
				return nil, domain.ErrInvalidStepType
			}
		}
	}

	step := domain.NewStep(input.TenantID, input.ProjectID, input.Name, input.Type, input.Config)
	if blockDef != nil {
		step.BlockDefinitionID = &blockDef.ID
	}
	step.SetPosition(input.PositionX, input.PositionY)

	// Set trigger type/config for start blocks
	if input.TriggerType != "" {
		tt := domain.StepTriggerType(input.TriggerType)
		step.TriggerType = &tt
	}
	if len(input.TriggerConfig) > 0 {
		step.TriggerConfig = input.TriggerConfig
	}

	// Set credential bindings (skip if null or empty)
	if len(input.CredentialBindings) > 0 && string(input.CredentialBindings) != "null" {
		step.CredentialBindings = input.CredentialBindings
	}

	if err := u.stepRepo.Create(ctx, step); err != nil {
		return nil, err
	}

	return step, nil
}

// GetByID retrieves a step by ID
func (u *StepUsecase) GetByID(ctx context.Context, tenantID, projectID, stepID uuid.UUID) (*domain.Step, error) {
	// Verify project exists
	if _, err := u.projectChecker.CheckExists(ctx, tenantID, projectID); err != nil {
		return nil, err
	}
	return u.stepRepo.GetByID(ctx, tenantID, projectID, stepID)
}

// List lists steps for a project
func (u *StepUsecase) List(ctx context.Context, tenantID, projectID uuid.UUID) ([]*domain.Step, error) {
	// Verify project exists
	if _, err := u.projectChecker.CheckExists(ctx, tenantID, projectID); err != nil {
		return nil, err
	}
	return u.stepRepo.ListByProject(ctx, tenantID, projectID)
}

// UpdateStepInput represents input for updating a step
type UpdateStepInput struct {
	TenantID           uuid.UUID
	ProjectID          uuid.UUID
	StepID             uuid.UUID
	Name               string
	Type               domain.StepType
	Config             json.RawMessage
	TriggerType        string          // For start blocks: manual, webhook, schedule, etc.
	TriggerConfig      json.RawMessage // Configuration for the trigger
	CredentialBindings json.RawMessage // Mapping of credential names to credential IDs
	PositionX          *int
	PositionY          *int
}

// Update updates a step
func (u *StepUsecase) Update(ctx context.Context, input UpdateStepInput) (*domain.Step, error) {
	// Verify project is editable
	if _, err := u.projectChecker.CheckEditable(ctx, input.TenantID, input.ProjectID); err != nil {
		return nil, err
	}

	step, err := u.stepRepo.GetByID(ctx, input.TenantID, input.ProjectID, input.StepID)
	if err != nil {
		return nil, err
	}

	if input.Name != "" {
		step.Name = input.Name
	}
	if input.Type != "" && input.Type.IsValid() {
		step.Type = input.Type
	}
	if input.Config != nil {
		step.Config = input.Config
	}
	if input.PositionX != nil {
		step.PositionX = *input.PositionX
	}
	if input.PositionY != nil {
		step.PositionY = *input.PositionY
	}
	// Update trigger type/config for start blocks
	if input.TriggerType != "" {
		tt := domain.StepTriggerType(input.TriggerType)
		step.TriggerType = &tt
	}
	if len(input.TriggerConfig) > 0 {
		step.TriggerConfig = input.TriggerConfig
	}

	// Update credential bindings (skip if null or empty)
	// Note: Tenant authorization for credential IDs is enforced at runtime by CredentialResolver
	if len(input.CredentialBindings) > 0 && string(input.CredentialBindings) != "null" {
		step.CredentialBindings = input.CredentialBindings
	}

	if err := u.stepRepo.Update(ctx, step); err != nil {
		return nil, err
	}

	return step, nil
}

// Delete deletes a step
func (u *StepUsecase) Delete(ctx context.Context, tenantID, projectID, stepID uuid.UUID) error {
	// Verify project is editable
	if _, err := u.projectChecker.CheckEditable(ctx, tenantID, projectID); err != nil {
		return err
	}

	return u.stepRepo.Delete(ctx, tenantID, projectID, stepID)
}
