package usecase

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// BlockGroupUsecase handles block group business logic
type BlockGroupUsecase struct {
	workflowRepo   repository.WorkflowRepository
	blockGroupRepo repository.BlockGroupRepository
	stepRepo       repository.StepRepository
}

// NewBlockGroupUsecase creates a new BlockGroupUsecase
func NewBlockGroupUsecase(
	workflowRepo repository.WorkflowRepository,
	blockGroupRepo repository.BlockGroupRepository,
	stepRepo repository.StepRepository,
) *BlockGroupUsecase {
	return &BlockGroupUsecase{
		workflowRepo:   workflowRepo,
		blockGroupRepo: blockGroupRepo,
		stepRepo:       stepRepo,
	}
}

// CreateBlockGroupInput represents input for creating a block group
type CreateBlockGroupInput struct {
	TenantID      uuid.UUID
	WorkflowID    uuid.UUID
	Name          string
	Type          domain.BlockGroupType
	Config        json.RawMessage
	ParentGroupID *uuid.UUID
	PositionX     int
	PositionY     int
	Width         int
	Height        int
}

// Create creates a new block group
func (u *BlockGroupUsecase) Create(ctx context.Context, input CreateBlockGroupInput) (*domain.BlockGroup, error) {
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
	if !input.Type.IsValid() {
		return nil, domain.ErrBlockGroupInvalidType
	}

	// Verify parent group if specified
	if input.ParentGroupID != nil {
		parent, err := u.blockGroupRepo.GetByID(ctx, *input.ParentGroupID)
		if err != nil {
			return nil, err
		}
		if parent.WorkflowID != input.WorkflowID {
			return nil, domain.NewValidationError("parent_group_id", "parent group must be in the same workflow")
		}
	}

	group := domain.NewBlockGroup(input.WorkflowID, input.Name, input.Type)
	if input.Config != nil {
		group.Config = input.Config
	}
	group.ParentGroupID = input.ParentGroupID
	group.SetPosition(input.PositionX, input.PositionY)
	if input.Width > 0 {
		group.Width = input.Width
	}
	if input.Height > 0 {
		group.Height = input.Height
	}

	if err := u.blockGroupRepo.Create(ctx, group); err != nil {
		return nil, err
	}

	return group, nil
}

// GetByID retrieves a block group by ID
func (u *BlockGroupUsecase) GetByID(ctx context.Context, tenantID, workflowID, groupID uuid.UUID) (*domain.BlockGroup, error) {
	// Verify workflow exists
	if _, err := u.workflowRepo.GetByID(ctx, tenantID, workflowID); err != nil {
		return nil, err
	}

	group, err := u.blockGroupRepo.GetByID(ctx, groupID)
	if err != nil {
		return nil, err
	}

	// Verify group belongs to workflow
	if group.WorkflowID != workflowID {
		return nil, domain.ErrBlockGroupNotFound
	}

	return group, nil
}

// List lists block groups for a workflow
func (u *BlockGroupUsecase) List(ctx context.Context, tenantID, workflowID uuid.UUID) ([]*domain.BlockGroup, error) {
	// Verify workflow exists
	if _, err := u.workflowRepo.GetByID(ctx, tenantID, workflowID); err != nil {
		return nil, err
	}
	return u.blockGroupRepo.ListByWorkflow(ctx, workflowID)
}

// UpdateBlockGroupInput represents input for updating a block group
type UpdateBlockGroupInput struct {
	TenantID      uuid.UUID
	WorkflowID    uuid.UUID
	GroupID       uuid.UUID
	Name          string
	Config        json.RawMessage
	ParentGroupID *uuid.UUID
	PositionX     *int
	PositionY     *int
	Width         *int
	Height        *int
}

// Update updates a block group
func (u *BlockGroupUsecase) Update(ctx context.Context, input UpdateBlockGroupInput) (*domain.BlockGroup, error) {
	// Verify workflow is editable
	workflow, err := u.workflowRepo.GetByID(ctx, input.TenantID, input.WorkflowID)
	if err != nil {
		return nil, err
	}
	if !workflow.CanEdit() {
		return nil, domain.ErrWorkflowNotEditable
	}

	group, err := u.blockGroupRepo.GetByID(ctx, input.GroupID)
	if err != nil {
		return nil, err
	}

	// Verify group belongs to workflow
	if group.WorkflowID != input.WorkflowID {
		return nil, domain.ErrBlockGroupNotFound
	}

	if input.Name != "" {
		group.Name = input.Name
	}
	if input.Config != nil {
		group.Config = input.Config
	}
	if input.ParentGroupID != nil {
		// Prevent self-reference
		if *input.ParentGroupID == input.GroupID {
			return nil, domain.NewValidationError("parent_group_id", "block group cannot be its own parent")
		}
		group.ParentGroupID = input.ParentGroupID
	}
	if input.PositionX != nil {
		group.PositionX = *input.PositionX
	}
	if input.PositionY != nil {
		group.PositionY = *input.PositionY
	}
	if input.Width != nil {
		group.Width = *input.Width
	}
	if input.Height != nil {
		group.Height = *input.Height
	}

	if err := u.blockGroupRepo.Update(ctx, group); err != nil {
		return nil, err
	}

	return group, nil
}

// Delete deletes a block group
func (u *BlockGroupUsecase) Delete(ctx context.Context, tenantID, workflowID, groupID uuid.UUID) error {
	// Verify workflow is editable
	workflow, err := u.workflowRepo.GetByID(ctx, tenantID, workflowID)
	if err != nil {
		return err
	}
	if !workflow.CanEdit() {
		return domain.ErrWorkflowNotEditable
	}

	group, err := u.blockGroupRepo.GetByID(ctx, groupID)
	if err != nil {
		return err
	}

	// Verify group belongs to workflow
	if group.WorkflowID != workflowID {
		return domain.ErrBlockGroupNotFound
	}

	return u.blockGroupRepo.Delete(ctx, groupID)
}

// AddStepToGroupInput represents input for adding a step to a block group
type AddStepToGroupInput struct {
	TenantID   uuid.UUID
	WorkflowID uuid.UUID
	StepID     uuid.UUID
	GroupID    uuid.UUID
	GroupRole  domain.GroupRole
}

// AddStepToGroup adds a step to a block group
func (u *BlockGroupUsecase) AddStepToGroup(ctx context.Context, input AddStepToGroupInput) (*domain.Step, error) {
	// Verify workflow is editable
	workflow, err := u.workflowRepo.GetByID(ctx, input.TenantID, input.WorkflowID)
	if err != nil {
		return nil, err
	}
	if !workflow.CanEdit() {
		return nil, domain.ErrWorkflowNotEditable
	}

	// Verify group exists
	group, err := u.blockGroupRepo.GetByID(ctx, input.GroupID)
	if err != nil {
		return nil, err
	}
	if group.WorkflowID != input.WorkflowID {
		return nil, domain.ErrBlockGroupNotFound
	}

	// Get step
	step, err := u.stepRepo.GetByID(ctx, input.WorkflowID, input.StepID)
	if err != nil {
		return nil, err
	}

	// Validate step type - start nodes cannot be added to groups
	if step.Type == domain.StepTypeStart {
		return nil, domain.ErrStepCannotBeInGroup
	}

	// Validate group role
	if !input.GroupRole.IsValid() {
		return nil, domain.NewValidationError("group_role", "invalid group role")
	}

	// Update step
	step.BlockGroupID = &input.GroupID
	step.GroupRole = string(input.GroupRole)

	if err := u.stepRepo.Update(ctx, step); err != nil {
		return nil, err
	}

	return step, nil
}

// RemoveStepFromGroup removes a step from its block group
func (u *BlockGroupUsecase) RemoveStepFromGroup(ctx context.Context, tenantID, workflowID, stepID uuid.UUID) (*domain.Step, error) {
	// Verify workflow is editable
	workflow, err := u.workflowRepo.GetByID(ctx, tenantID, workflowID)
	if err != nil {
		return nil, err
	}
	if !workflow.CanEdit() {
		return nil, domain.ErrWorkflowNotEditable
	}

	// Get step
	step, err := u.stepRepo.GetByID(ctx, workflowID, stepID)
	if err != nil {
		return nil, err
	}

	// Remove from group
	step.BlockGroupID = nil
	step.GroupRole = ""

	if err := u.stepRepo.Update(ctx, step); err != nil {
		return nil, err
	}

	return step, nil
}

// GetStepsByGroup retrieves all steps in a block group
func (u *BlockGroupUsecase) GetStepsByGroup(ctx context.Context, tenantID, workflowID, groupID uuid.UUID) ([]*domain.Step, error) {
	// Verify workflow exists
	if _, err := u.workflowRepo.GetByID(ctx, tenantID, workflowID); err != nil {
		return nil, err
	}

	// Verify group exists
	group, err := u.blockGroupRepo.GetByID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if group.WorkflowID != workflowID {
		return nil, domain.ErrBlockGroupNotFound
	}

	// Get steps in group
	steps, err := u.stepRepo.ListByWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	// Filter by group
	var result []*domain.Step
	for _, step := range steps {
		if step.BlockGroupID != nil && *step.BlockGroupID == groupID {
			result = append(result, step)
		}
	}

	return result, nil
}
