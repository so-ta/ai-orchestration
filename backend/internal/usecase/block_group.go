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
	projectRepo    repository.ProjectRepository
	blockGroupRepo repository.BlockGroupRepository
	stepRepo       repository.StepRepository
	projectChecker *ProjectChecker
}

// NewBlockGroupUsecase creates a new BlockGroupUsecase
func NewBlockGroupUsecase(
	projectRepo repository.ProjectRepository,
	blockGroupRepo repository.BlockGroupRepository,
	stepRepo repository.StepRepository,
) *BlockGroupUsecase {
	return &BlockGroupUsecase{
		projectRepo:    projectRepo,
		blockGroupRepo: blockGroupRepo,
		stepRepo:       stepRepo,
		projectChecker: NewProjectChecker(projectRepo),
	}
}

// CreateBlockGroupInput represents input for creating a block group
// Supports 4 types: parallel, try_catch, foreach, while
type CreateBlockGroupInput struct {
	TenantID      uuid.UUID
	ProjectID     uuid.UUID
	Name          string
	Type          domain.BlockGroupType
	Config        json.RawMessage
	ParentGroupID *uuid.UUID
	PreProcess    *string // JS: external IN -> internal IN
	PostProcess   *string // JS: internal OUT -> external OUT
	PositionX     int
	PositionY     int
	Width         int
	Height        int
}

// Create creates a new block group
func (u *BlockGroupUsecase) Create(ctx context.Context, input CreateBlockGroupInput) (*domain.BlockGroup, error) {
	// Verify project exists and is editable
	if _, err := u.projectChecker.CheckEditable(ctx, input.TenantID, input.ProjectID); err != nil {
		return nil, err
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
		_, err := u.blockGroupRepo.GetByID(ctx, input.TenantID, input.ProjectID, *input.ParentGroupID)
		if err != nil {
			if err == domain.ErrBlockGroupNotFound {
				return nil, domain.NewValidationError("parent_group_id", "parent group not found in this project")
			}
			return nil, err
		}
	}

	group := domain.NewBlockGroup(input.TenantID, input.ProjectID, input.Name, input.Type)
	if input.Config != nil {
		group.Config = input.Config
	}
	group.ParentGroupID = input.ParentGroupID
	group.PreProcess = input.PreProcess
	group.PostProcess = input.PostProcess
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
func (u *BlockGroupUsecase) GetByID(ctx context.Context, tenantID, projectID, groupID uuid.UUID) (*domain.BlockGroup, error) {
	// Verify project exists
	if _, err := u.projectChecker.CheckExists(ctx, tenantID, projectID); err != nil {
		return nil, err
	}

	return u.blockGroupRepo.GetByID(ctx, tenantID, projectID, groupID)
}

// List lists block groups for a project
func (u *BlockGroupUsecase) List(ctx context.Context, tenantID, projectID uuid.UUID) ([]*domain.BlockGroup, error) {
	// Verify project exists
	if _, err := u.projectChecker.CheckExists(ctx, tenantID, projectID); err != nil {
		return nil, err
	}
	return u.blockGroupRepo.ListByProject(ctx, tenantID, projectID)
}

// UpdateBlockGroupInput represents input for updating a block group
type UpdateBlockGroupInput struct {
	TenantID      uuid.UUID
	ProjectID     uuid.UUID
	GroupID       uuid.UUID
	Name          string
	Config        json.RawMessage
	ParentGroupID *uuid.UUID
	PreProcess    *string // JS: external IN -> internal IN
	PostProcess   *string // JS: internal OUT -> external OUT
	PositionX     *int
	PositionY     *int
	Width         *int
	Height        *int
}

// Update updates a block group
func (u *BlockGroupUsecase) Update(ctx context.Context, input UpdateBlockGroupInput) (*domain.BlockGroup, error) {
	// Verify project is editable
	if _, err := u.projectChecker.CheckEditable(ctx, input.TenantID, input.ProjectID); err != nil {
		return nil, err
	}

	group, err := u.blockGroupRepo.GetByID(ctx, input.TenantID, input.ProjectID, input.GroupID)
	if err != nil {
		return nil, err
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
	if input.PreProcess != nil {
		group.PreProcess = input.PreProcess
	}
	if input.PostProcess != nil {
		group.PostProcess = input.PostProcess
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
func (u *BlockGroupUsecase) Delete(ctx context.Context, tenantID, projectID, groupID uuid.UUID) error {
	// Verify project is editable
	if _, err := u.projectChecker.CheckEditable(ctx, tenantID, projectID); err != nil {
		return err
	}

	// Verify group exists
	if _, err := u.blockGroupRepo.GetByID(ctx, tenantID, projectID, groupID); err != nil {
		return err
	}

	return u.blockGroupRepo.Delete(ctx, tenantID, projectID, groupID)
}

// AddStepToGroupInput represents input for adding a step to a block group
type AddStepToGroupInput struct {
	TenantID  uuid.UUID
	ProjectID uuid.UUID
	StepID    uuid.UUID
	GroupID   uuid.UUID
	GroupRole domain.GroupRole
}

// AddStepToGroup adds a step to a block group
func (u *BlockGroupUsecase) AddStepToGroup(ctx context.Context, input AddStepToGroupInput) (*domain.Step, error) {
	// Verify project is editable
	if _, err := u.projectChecker.CheckEditable(ctx, input.TenantID, input.ProjectID); err != nil {
		return nil, err
	}

	// Verify group exists
	if _, err := u.blockGroupRepo.GetByID(ctx, input.TenantID, input.ProjectID, input.GroupID); err != nil {
		return nil, err
	}

	// Get step
	step, err := u.stepRepo.GetByID(ctx, input.TenantID, input.ProjectID, input.StepID)
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
func (u *BlockGroupUsecase) RemoveStepFromGroup(ctx context.Context, tenantID, projectID, stepID uuid.UUID) (*domain.Step, error) {
	// Verify project is editable
	if _, err := u.projectChecker.CheckEditable(ctx, tenantID, projectID); err != nil {
		return nil, err
	}

	// Get step
	step, err := u.stepRepo.GetByID(ctx, tenantID, projectID, stepID)
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
func (u *BlockGroupUsecase) GetStepsByGroup(ctx context.Context, tenantID, projectID, groupID uuid.UUID) ([]*domain.Step, error) {
	// Verify project exists
	if _, err := u.projectChecker.CheckExists(ctx, tenantID, projectID); err != nil {
		return nil, err
	}

	// Verify group exists
	if _, err := u.blockGroupRepo.GetByID(ctx, tenantID, projectID, groupID); err != nil {
		return nil, err
	}

	// Get steps in group
	steps, err := u.stepRepo.ListByProject(ctx, tenantID, projectID)
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
