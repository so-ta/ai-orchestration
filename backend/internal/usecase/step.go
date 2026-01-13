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
	workflowRepo     repository.WorkflowRepository
	stepRepo         repository.StepRepository
	blockDefRepo     repository.BlockDefinitionRepository
}

// NewStepUsecase creates a new StepUsecase
func NewStepUsecase(
	workflowRepo repository.WorkflowRepository,
	stepRepo repository.StepRepository,
	blockDefRepo repository.BlockDefinitionRepository,
) *StepUsecase {
	return &StepUsecase{
		workflowRepo:     workflowRepo,
		stepRepo:         stepRepo,
		blockDefRepo:     blockDefRepo,
	}
}

// CreateStepInput represents input for creating a step
type CreateStepInput struct {
	TenantID   uuid.UUID
	WorkflowID uuid.UUID
	Name       string
	Type       domain.StepType
	Config     json.RawMessage
	PositionX  int
	PositionY  int
}

// Create creates a new step
func (u *StepUsecase) Create(ctx context.Context, input CreateStepInput) (*domain.Step, error) {
	// Verify workflow exists and is editable
	workflow, err := u.workflowRepo.GetByID(ctx, input.TenantID, input.WorkflowID)
	if err != nil {
		return nil, err
	}
	if !workflow.CanEdit() {
		return nil, domain.ErrWorkflowNotEditable
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

	step := domain.NewStep(input.TenantID, input.WorkflowID, input.Name, input.Type, input.Config)
	if blockDef != nil {
		step.BlockDefinitionID = &blockDef.ID
	}
	step.SetPosition(input.PositionX, input.PositionY)

	if err := u.stepRepo.Create(ctx, step); err != nil {
		return nil, err
	}

	return step, nil
}

// GetByID retrieves a step by ID
func (u *StepUsecase) GetByID(ctx context.Context, tenantID, workflowID, stepID uuid.UUID) (*domain.Step, error) {
	// Verify workflow exists
	if _, err := u.workflowRepo.GetByID(ctx, tenantID, workflowID); err != nil {
		return nil, err
	}
	return u.stepRepo.GetByID(ctx, tenantID, workflowID, stepID)
}

// List lists steps for a workflow
func (u *StepUsecase) List(ctx context.Context, tenantID, workflowID uuid.UUID) ([]*domain.Step, error) {
	// Verify workflow exists
	if _, err := u.workflowRepo.GetByID(ctx, tenantID, workflowID); err != nil {
		return nil, err
	}
	return u.stepRepo.ListByWorkflow(ctx, tenantID, workflowID)
}

// UpdateStepInput represents input for updating a step
type UpdateStepInput struct {
	TenantID   uuid.UUID
	WorkflowID uuid.UUID
	StepID     uuid.UUID
	Name       string
	Type       domain.StepType
	Config     json.RawMessage
	PositionX  *int
	PositionY  *int
}

// Update updates a step
func (u *StepUsecase) Update(ctx context.Context, input UpdateStepInput) (*domain.Step, error) {
	// Verify workflow is editable
	workflow, err := u.workflowRepo.GetByID(ctx, input.TenantID, input.WorkflowID)
	if err != nil {
		return nil, err
	}
	if !workflow.CanEdit() {
		return nil, domain.ErrWorkflowNotEditable
	}

	step, err := u.stepRepo.GetByID(ctx, input.TenantID, input.WorkflowID, input.StepID)
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

	if err := u.stepRepo.Update(ctx, step); err != nil {
		return nil, err
	}

	return step, nil
}

// Delete deletes a step
func (u *StepUsecase) Delete(ctx context.Context, tenantID, workflowID, stepID uuid.UUID) error {
	// Verify workflow is editable
	workflow, err := u.workflowRepo.GetByID(ctx, tenantID, workflowID)
	if err != nil {
		return err
	}
	if !workflow.CanEdit() {
		return domain.ErrWorkflowNotEditable
	}

	return u.stepRepo.Delete(ctx, tenantID, workflowID, stepID)
}
