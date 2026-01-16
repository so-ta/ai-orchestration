package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// EdgeUsecase handles edge business logic
type EdgeUsecase struct {
	workflowRepo        repository.WorkflowRepository
	stepRepo            repository.StepRepository
	edgeRepo            repository.EdgeRepository
	blockGroupRepo      repository.BlockGroupRepository
	blockDefinitionRepo repository.BlockDefinitionRepository
	workflowChecker     *WorkflowChecker
}

// NewEdgeUsecase creates a new EdgeUsecase
func NewEdgeUsecase(
	workflowRepo repository.WorkflowRepository,
	stepRepo repository.StepRepository,
	edgeRepo repository.EdgeRepository,
) *EdgeUsecase {
	return &EdgeUsecase{
		workflowRepo:    workflowRepo,
		stepRepo:        stepRepo,
		edgeRepo:        edgeRepo,
		workflowChecker: NewWorkflowChecker(workflowRepo),
	}
}

// WithBlockGroupRepo sets the block group repository for the edge usecase
func (u *EdgeUsecase) WithBlockGroupRepo(repo repository.BlockGroupRepository) *EdgeUsecase {
	u.blockGroupRepo = repo
	return u
}

// WithBlockDefinitionRepo sets the block definition repository for port validation
func (u *EdgeUsecase) WithBlockDefinitionRepo(repo repository.BlockDefinitionRepository) *EdgeUsecase {
	u.blockDefinitionRepo = repo
	return u
}

// CreateEdgeInput represents input for creating an edge
// Either SourceStepID or SourceBlockGroupID must be provided
// Either TargetStepID or TargetBlockGroupID must be provided
type CreateEdgeInput struct {
	TenantID           uuid.UUID
	WorkflowID         uuid.UUID
	SourceStepID       *uuid.UUID
	TargetStepID       *uuid.UUID
	SourceBlockGroupID *uuid.UUID
	TargetBlockGroupID *uuid.UUID
	SourcePort         string
	TargetPort         string
	Condition          string
}

// Create creates a new edge
func (u *EdgeUsecase) Create(ctx context.Context, input CreateEdgeInput) (*domain.Edge, error) {
	// Verify workflow is editable
	if _, err := u.workflowChecker.CheckEditable(ctx, input.TenantID, input.WorkflowID); err != nil {
		return nil, err
	}

	// Validate: no self-loop for step-to-step edges
	if input.SourceStepID != nil && input.TargetStepID != nil && *input.SourceStepID == *input.TargetStepID {
		return nil, domain.ErrEdgeSelfLoop
	}

	// Validate: no self-loop for group-to-group edges
	if input.SourceBlockGroupID != nil && input.TargetBlockGroupID != nil && *input.SourceBlockGroupID == *input.TargetBlockGroupID {
		return nil, domain.ErrEdgeSelfLoop
	}

	// Verify source exists (step or group) and validate source port
	var sourceStep *domain.Step
	var sourceBlockGroup *domain.BlockGroup
	if input.SourceStepID != nil {
		step, err := u.stepRepo.GetByID(ctx, input.TenantID, input.WorkflowID, *input.SourceStepID)
		if err != nil {
			return nil, err
		}
		sourceStep = step
	} else if input.SourceBlockGroupID != nil && u.blockGroupRepo != nil {
		group, err := u.blockGroupRepo.GetByID(ctx, input.TenantID, *input.SourceBlockGroupID)
		if err != nil {
			return nil, err
		}
		sourceBlockGroup = group
	}

	// Verify target exists (step or group) and validate target port
	var targetStep *domain.Step
	var targetBlockGroup *domain.BlockGroup
	if input.TargetStepID != nil {
		step, err := u.stepRepo.GetByID(ctx, input.TenantID, input.WorkflowID, *input.TargetStepID)
		if err != nil {
			return nil, err
		}
		targetStep = step
	} else if input.TargetBlockGroupID != nil && u.blockGroupRepo != nil {
		group, err := u.blockGroupRepo.GetByID(ctx, input.TenantID, *input.TargetBlockGroupID)
		if err != nil {
			return nil, err
		}
		targetBlockGroup = group
	}

	// Validate ports if block definition repository is available
	if u.blockDefinitionRepo != nil {
		if err := u.validateSourcePort(ctx, input.SourcePort, sourceStep, sourceBlockGroup); err != nil {
			return nil, err
		}
		if err := u.validateTargetPort(ctx, input.TargetPort, targetStep, targetBlockGroup); err != nil {
			return nil, err
		}
	}

	// Create new edge based on connection type
	var newEdge *domain.Edge
	if input.SourceStepID != nil && input.TargetStepID != nil {
		// Step to step
		newEdge = domain.NewEdgeWithPort(input.TenantID, input.WorkflowID, *input.SourceStepID, *input.TargetStepID, input.SourcePort, input.TargetPort, input.Condition)
	} else if input.SourceStepID != nil && input.TargetBlockGroupID != nil {
		// Step to group
		newEdge = domain.NewEdgeToGroup(input.TenantID, input.WorkflowID, *input.SourceStepID, *input.TargetBlockGroupID, input.SourcePort)
	} else if input.SourceBlockGroupID != nil && input.TargetStepID != nil {
		// Group to step
		newEdge = domain.NewEdgeFromGroup(input.TenantID, input.WorkflowID, *input.SourceBlockGroupID, *input.TargetStepID, input.TargetPort)
	} else if input.SourceBlockGroupID != nil && input.TargetBlockGroupID != nil {
		// Group to group
		newEdge = domain.NewGroupToGroupEdge(input.TenantID, input.WorkflowID, *input.SourceBlockGroupID, *input.TargetBlockGroupID)
	}

	// Only check for cycles on step-to-step edges for now
	// TODO: Extend cycle detection to handle groups
	if input.SourceStepID != nil && input.TargetStepID != nil {
		edges, err := u.edgeRepo.ListByWorkflow(ctx, input.TenantID, input.WorkflowID)
		if err != nil {
			return nil, err
		}

		allEdges := append(edges, newEdge)

		steps, err := u.stepRepo.ListByWorkflow(ctx, input.TenantID, input.WorkflowID)
		if err != nil {
			return nil, err
		}

		stepsSlice := make([]domain.Step, len(steps))
		for i, s := range steps {
			stepsSlice[i] = *s
		}
		edgesSlice := make([]domain.Edge, len(allEdges))
		for i, e := range allEdges {
			edgesSlice[i] = *e
		}

		if wouldCreateCycle(stepsSlice, edgesSlice, *input.SourceStepID, *input.TargetStepID) {
			return nil, domain.ErrEdgeCreatesCycle
		}
	}

	if err := u.edgeRepo.Create(ctx, newEdge); err != nil {
		return nil, err
	}

	return newEdge, nil
}

// wouldCreateCycle checks if adding an edge from source to target would create a cycle
func wouldCreateCycle(steps []domain.Step, edges []domain.Edge, source, target uuid.UUID) bool {
	// Build adjacency list (only for step-to-step edges)
	adj := make(map[uuid.UUID][]uuid.UUID)
	for _, edge := range edges {
		if edge.SourceStepID != nil && edge.TargetStepID != nil {
			adj[*edge.SourceStepID] = append(adj[*edge.SourceStepID], *edge.TargetStepID)
		}
	}

	// Check if there's a path from target to source (which would mean adding source->target creates a cycle)
	visited := make(map[uuid.UUID]bool)
	var dfs func(current uuid.UUID) bool
	dfs = func(current uuid.UUID) bool {
		if current == source {
			return true
		}
		if visited[current] {
			return false
		}
		visited[current] = true
		for _, neighbor := range adj[current] {
			if dfs(neighbor) {
				return true
			}
		}
		return false
	}

	return dfs(target)
}

// List lists edges for a workflow
func (u *EdgeUsecase) List(ctx context.Context, tenantID, workflowID uuid.UUID) ([]*domain.Edge, error) {
	// Verify workflow exists
	if _, err := u.workflowChecker.CheckExists(ctx, tenantID, workflowID); err != nil {
		return nil, err
	}
	return u.edgeRepo.ListByWorkflow(ctx, tenantID, workflowID)
}

// Delete deletes an edge
func (u *EdgeUsecase) Delete(ctx context.Context, tenantID, workflowID, edgeID uuid.UUID) error {
	// Verify workflow is editable
	if _, err := u.workflowChecker.CheckEditable(ctx, tenantID, workflowID); err != nil {
		return err
	}

	return u.edgeRepo.Delete(ctx, tenantID, workflowID, edgeID)
}

// validateSourcePort validates that the source port exists in the block definition
func (u *EdgeUsecase) validateSourcePort(ctx context.Context, sourcePort string, step *domain.Step, group *domain.BlockGroup) error {
	// Skip validation if port is empty (default port will be used)
	if sourcePort == "" {
		return nil
	}

	var blockDef *domain.BlockDefinition
	var err error

	if step != nil {
		blockDef, err = u.getBlockDefinitionForStep(ctx, step)
	} else if group != nil {
		blockDef, err = u.getBlockDefinitionForGroup(ctx, group)
	}

	if err != nil {
		// If block definition not found, skip validation (legacy blocks)
		if err == domain.ErrBlockDefinitionNotFound {
			return nil
		}
		return err
	}

	if blockDef == nil {
		return nil
	}

	// Check if the source port exists in output ports
	for _, port := range blockDef.OutputPorts {
		if port.Name == sourcePort {
			return nil
		}
	}

	return domain.ErrSourcePortNotFound
}

// validateTargetPort validates that the target port exists in the block definition
func (u *EdgeUsecase) validateTargetPort(ctx context.Context, targetPort string, step *domain.Step, group *domain.BlockGroup) error {
	// Skip validation if port is empty (default port will be used)
	if targetPort == "" {
		return nil
	}

	// Special case: "group-input" is a virtual port for block groups
	if group != nil && targetPort == "group-input" {
		return nil
	}

	var blockDef *domain.BlockDefinition
	var err error

	if step != nil {
		blockDef, err = u.getBlockDefinitionForStep(ctx, step)
	} else if group != nil {
		blockDef, err = u.getBlockDefinitionForGroup(ctx, group)
	}

	if err != nil {
		// If block definition not found, skip validation (legacy blocks)
		if err == domain.ErrBlockDefinitionNotFound {
			return nil
		}
		return err
	}

	if blockDef == nil {
		return nil
	}

	// Check if the target port exists in input ports
	for _, port := range blockDef.InputPorts {
		if port.Name == targetPort {
			return nil
		}
	}

	return domain.ErrTargetPortNotFound
}

// getBlockDefinitionForStep retrieves the block definition for a step
func (u *EdgeUsecase) getBlockDefinitionForStep(ctx context.Context, step *domain.Step) (*domain.BlockDefinition, error) {
	if step.BlockDefinitionID != nil {
		return u.blockDefinitionRepo.GetByID(ctx, *step.BlockDefinitionID)
	}
	// Use step type as slug for legacy steps
	return u.blockDefinitionRepo.GetBySlug(ctx, nil, string(step.Type))
}

// getBlockDefinitionForGroup retrieves the block definition for a block group
func (u *EdgeUsecase) getBlockDefinitionForGroup(ctx context.Context, group *domain.BlockGroup) (*domain.BlockDefinition, error) {
	// Use group type as slug
	return u.blockDefinitionRepo.GetBySlug(ctx, nil, string(group.Type))
}
