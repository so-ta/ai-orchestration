package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// EdgeUsecase handles edge business logic
type EdgeUsecase struct {
	workflowRepo   repository.WorkflowRepository
	stepRepo       repository.StepRepository
	edgeRepo       repository.EdgeRepository
	blockGroupRepo repository.BlockGroupRepository
}

// NewEdgeUsecase creates a new EdgeUsecase
func NewEdgeUsecase(
	workflowRepo repository.WorkflowRepository,
	stepRepo repository.StepRepository,
	edgeRepo repository.EdgeRepository,
) *EdgeUsecase {
	return &EdgeUsecase{
		workflowRepo: workflowRepo,
		stepRepo:     stepRepo,
		edgeRepo:     edgeRepo,
	}
}

// WithBlockGroupRepo sets the block group repository for the edge usecase
func (u *EdgeUsecase) WithBlockGroupRepo(repo repository.BlockGroupRepository) *EdgeUsecase {
	u.blockGroupRepo = repo
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
	workflow, err := u.workflowRepo.GetByID(ctx, input.TenantID, input.WorkflowID)
	if err != nil {
		return nil, err
	}
	if !workflow.CanEdit() {
		return nil, domain.ErrWorkflowNotEditable
	}

	// Validate: no self-loop for step-to-step edges
	if input.SourceStepID != nil && input.TargetStepID != nil && *input.SourceStepID == *input.TargetStepID {
		return nil, domain.ErrEdgeSelfLoop
	}

	// Validate: no self-loop for group-to-group edges
	if input.SourceBlockGroupID != nil && input.TargetBlockGroupID != nil && *input.SourceBlockGroupID == *input.TargetBlockGroupID {
		return nil, domain.ErrEdgeSelfLoop
	}

	// Verify source exists (step or group)
	if input.SourceStepID != nil {
		if _, err := u.stepRepo.GetByID(ctx, input.TenantID, input.WorkflowID, *input.SourceStepID); err != nil {
			return nil, err
		}
	} else if input.SourceBlockGroupID != nil && u.blockGroupRepo != nil {
		if _, err := u.blockGroupRepo.GetByID(ctx, input.TenantID, *input.SourceBlockGroupID); err != nil {
			return nil, err
		}
	}

	// Verify target exists (step or group)
	if input.TargetStepID != nil {
		if _, err := u.stepRepo.GetByID(ctx, input.TenantID, input.WorkflowID, *input.TargetStepID); err != nil {
			return nil, err
		}
	} else if input.TargetBlockGroupID != nil && u.blockGroupRepo != nil {
		if _, err := u.blockGroupRepo.GetByID(ctx, input.TenantID, *input.TargetBlockGroupID); err != nil {
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
	if _, err := u.workflowRepo.GetByID(ctx, tenantID, workflowID); err != nil {
		return nil, err
	}
	return u.edgeRepo.ListByWorkflow(ctx, tenantID, workflowID)
}

// Delete deletes an edge
func (u *EdgeUsecase) Delete(ctx context.Context, tenantID, workflowID, edgeID uuid.UUID) error {
	// Verify workflow is editable
	workflow, err := u.workflowRepo.GetByID(ctx, tenantID, workflowID)
	if err != nil {
		return err
	}
	if !workflow.CanEdit() {
		return domain.ErrWorkflowNotEditable
	}

	return u.edgeRepo.Delete(ctx, tenantID, workflowID, edgeID)
}
