package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// EdgeUsecase handles edge business logic
type EdgeUsecase struct {
	workflowRepo repository.WorkflowRepository
	stepRepo     repository.StepRepository
	edgeRepo     repository.EdgeRepository
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

// CreateEdgeInput represents input for creating an edge
type CreateEdgeInput struct {
	TenantID     uuid.UUID
	WorkflowID   uuid.UUID
	SourceStepID uuid.UUID
	TargetStepID uuid.UUID
	SourcePort   string
	TargetPort   string
	Condition    string
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

	// Validate: no self-loop
	if input.SourceStepID == input.TargetStepID {
		return nil, domain.ErrEdgeSelfLoop
	}

	// Verify both steps exist
	if _, err := u.stepRepo.GetByID(ctx, input.TenantID, input.WorkflowID, input.SourceStepID); err != nil {
		return nil, err
	}
	if _, err := u.stepRepo.GetByID(ctx, input.TenantID, input.WorkflowID, input.TargetStepID); err != nil {
		return nil, err
	}

	// Check for duplicate
	exists, err := u.edgeRepo.Exists(ctx, input.TenantID, input.WorkflowID, input.SourceStepID, input.TargetStepID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, domain.ErrEdgeDuplicate
	}

	// Check if adding this edge would create a cycle
	edges, err := u.edgeRepo.ListByWorkflow(ctx, input.TenantID, input.WorkflowID)
	if err != nil {
		return nil, err
	}

	// Temporarily add the new edge and check for cycle
	newEdge := domain.NewEdgeWithPort(input.TenantID, input.WorkflowID, input.SourceStepID, input.TargetStepID, input.SourcePort, input.TargetPort, input.Condition)
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

	if wouldCreateCycle(stepsSlice, edgesSlice, input.SourceStepID, input.TargetStepID) {
		return nil, domain.ErrEdgeCreatesCycle
	}

	if err := u.edgeRepo.Create(ctx, newEdge); err != nil {
		return nil, err
	}

	return newEdge, nil
}

// wouldCreateCycle checks if adding an edge from source to target would create a cycle
func wouldCreateCycle(steps []domain.Step, edges []domain.Edge, source, target uuid.UUID) bool {
	// Build adjacency list
	adj := make(map[uuid.UUID][]uuid.UUID)
	for _, edge := range edges {
		adj[edge.SourceStepID] = append(adj[edge.SourceStepID], edge.TargetStepID)
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
