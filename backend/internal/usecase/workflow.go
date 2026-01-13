package usecase

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// WorkflowUsecase handles workflow business logic
type WorkflowUsecase struct {
	workflowRepo repository.WorkflowRepository
	stepRepo     repository.StepRepository
	edgeRepo     repository.EdgeRepository
	versionRepo  repository.WorkflowVersionRepository
}

// NewWorkflowUsecase creates a new WorkflowUsecase
func NewWorkflowUsecase(
	workflowRepo repository.WorkflowRepository,
	stepRepo repository.StepRepository,
	edgeRepo repository.EdgeRepository,
	versionRepo repository.WorkflowVersionRepository,
) *WorkflowUsecase {
	return &WorkflowUsecase{
		workflowRepo: workflowRepo,
		stepRepo:     stepRepo,
		edgeRepo:     edgeRepo,
		versionRepo:  versionRepo,
	}
}

// CreateWorkflowInput represents input for creating a workflow
type CreateWorkflowInput struct {
	TenantID    uuid.UUID
	Name        string
	Description string
	InputSchema json.RawMessage
}

// Create creates a new workflow with an auto-created Start node
func (u *WorkflowUsecase) Create(ctx context.Context, input CreateWorkflowInput) (*domain.Workflow, error) {
	if input.Name == "" {
		return nil, domain.NewValidationError("name", "name is required")
	}

	workflow := domain.NewWorkflow(input.TenantID, input.Name, input.Description)
	workflow.InputSchema = input.InputSchema

	if err := u.workflowRepo.Create(ctx, workflow); err != nil {
		return nil, err
	}

	// Auto-create Start step for the new workflow
	startStep := domain.NewStep(input.TenantID, workflow.ID, "Start", domain.StepTypeStart, json.RawMessage(`{}`))
	startStep.SetPosition(400, 50) // Center-top position

	if err := u.stepRepo.Create(ctx, startStep); err != nil {
		// Log error but don't fail workflow creation
		// The user can manually add a start step if this fails
		return workflow, nil
	}

	return workflow, nil
}

// GetByID retrieves a workflow by ID
func (u *WorkflowUsecase) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Workflow, error) {
	return u.workflowRepo.GetByID(ctx, tenantID, id)
}

// GetWithDetails retrieves a workflow with steps and edges
func (u *WorkflowUsecase) GetWithDetails(ctx context.Context, tenantID, id uuid.UUID) (*domain.Workflow, error) {
	return u.workflowRepo.GetWithStepsAndEdges(ctx, tenantID, id)
}

// ListWorkflowsInput represents input for listing workflows
type ListWorkflowsInput struct {
	TenantID uuid.UUID
	Status   *domain.WorkflowStatus
	Page     int
	Limit    int
}

// ListWorkflowsOutput represents output for listing workflows
type ListWorkflowsOutput struct {
	Workflows []*domain.Workflow
	Total     int
	Page      int
	Limit     int
}

// List lists workflows with pagination
func (u *WorkflowUsecase) List(ctx context.Context, input ListWorkflowsInput) (*ListWorkflowsOutput, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.Limit < 1 || input.Limit > 100 {
		input.Limit = 20
	}

	filter := repository.WorkflowFilter{
		Status: input.Status,
		Page:   input.Page,
		Limit:  input.Limit,
	}

	workflows, total, err := u.workflowRepo.List(ctx, input.TenantID, filter)
	if err != nil {
		return nil, err
	}

	return &ListWorkflowsOutput{
		Workflows: workflows,
		Total:     total,
		Page:      input.Page,
		Limit:     input.Limit,
	}, nil
}

// UpdateWorkflowInput represents input for updating a workflow
type UpdateWorkflowInput struct {
	TenantID    uuid.UUID
	ID          uuid.UUID
	Name        string
	Description string
	InputSchema json.RawMessage
}

// Update updates a workflow
func (u *WorkflowUsecase) Update(ctx context.Context, input UpdateWorkflowInput) (*domain.Workflow, error) {
	workflow, err := u.workflowRepo.GetByID(ctx, input.TenantID, input.ID)
	if err != nil {
		return nil, err
	}

	if !workflow.CanEdit() {
		return nil, domain.ErrWorkflowNotEditable
	}

	if input.Name != "" {
		workflow.Name = input.Name
	}
	workflow.Description = input.Description
	if input.InputSchema != nil {
		workflow.InputSchema = input.InputSchema
	}

	if err := u.workflowRepo.Update(ctx, workflow); err != nil {
		return nil, err
	}

	return workflow, nil
}

// Delete deletes a workflow
func (u *WorkflowUsecase) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return u.workflowRepo.Delete(ctx, tenantID, id)
}

// SaveWorkflowInput represents input for saving a workflow
type SaveWorkflowInput struct {
	TenantID    uuid.UUID
	ID          uuid.UUID
	Name        string
	Description string
	InputSchema json.RawMessage
	Steps       []domain.Step
	Edges       []domain.Edge
}

// Save saves a workflow and creates a new version snapshot
// This replaces the old "Publish" functionality
func (u *WorkflowUsecase) Save(ctx context.Context, input SaveWorkflowInput) (*domain.Workflow, error) {
	workflow, err := u.workflowRepo.GetByID(ctx, input.TenantID, input.ID)
	if err != nil {
		return nil, err
	}

	// Update workflow fields
	if input.Name != "" {
		workflow.Name = input.Name
	}
	workflow.Description = input.Description
	workflow.InputSchema = input.InputSchema
	workflow.Steps = input.Steps
	workflow.Edges = input.Edges

	// Validate DAG before saving
	if err := u.ValidateDAG(workflow); err != nil {
		return nil, err
	}

	// Delete existing steps and edges, then recreate
	if err := u.deleteAndRecreateStepsEdges(ctx, input.TenantID, workflow.ID, input.Steps, input.Edges); err != nil {
		return nil, err
	}

	// Increment version
	workflow.IncrementVersion()

	// Clear any existing draft
	workflow.ClearDraft()

	// Create workflow definition snapshot
	definition := domain.WorkflowDefinition{
		Name:         workflow.Name,
		Description:  workflow.Description,
		InputSchema:  workflow.InputSchema,
		OutputSchema: workflow.OutputSchema,
		Steps:        input.Steps,
		Edges:        input.Edges,
	}

	definitionJSON, err := json.Marshal(definition)
	if err != nil {
		return nil, err
	}

	// Create version record
	workflowVersion := &domain.WorkflowVersion{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Version:    workflow.Version,
		Definition: definitionJSON,
		SavedAt:    time.Now().UTC(),
	}

	// Save version snapshot
	if err := u.versionRepo.Create(ctx, workflowVersion); err != nil {
		return nil, err
	}

	// Update workflow
	if err := u.workflowRepo.Update(ctx, workflow); err != nil {
		return nil, err
	}

	// Reload with steps and edges
	return u.workflowRepo.GetWithStepsAndEdges(ctx, input.TenantID, input.ID)
}

// SaveDraftInput represents input for saving a workflow as draft
type SaveDraftInput struct {
	TenantID    uuid.UUID
	ID          uuid.UUID
	Name        string
	Description string
	InputSchema json.RawMessage
	Steps       []domain.Step
	Edges       []domain.Edge
}

// SaveDraft saves a workflow as draft without creating a new version
// Draft changes are not validated and not persisted to steps/edges tables
func (u *WorkflowUsecase) SaveDraft(ctx context.Context, input SaveDraftInput) (*domain.Workflow, error) {
	workflow, err := u.workflowRepo.GetByID(ctx, input.TenantID, input.ID)
	if err != nil {
		return nil, err
	}

	// Create draft data
	draft := &domain.WorkflowDraft{
		Name:         input.Name,
		Description:  input.Description,
		InputSchema:  input.InputSchema,
		OutputSchema: workflow.OutputSchema,
		Steps:        input.Steps,
		Edges:        input.Edges,
		UpdatedAt:    time.Now().UTC(),
	}

	// Set draft
	if err := workflow.SetDraft(draft); err != nil {
		return nil, err
	}

	// Update workflow (only draft field changes)
	if err := u.workflowRepo.Update(ctx, workflow); err != nil {
		return nil, err
	}

	// Return workflow with draft data applied
	workflow.Name = draft.Name
	workflow.Description = draft.Description
	workflow.InputSchema = draft.InputSchema
	workflow.Steps = draft.Steps
	workflow.Edges = draft.Edges
	workflow.HasDraft = true

	return workflow, nil
}

// DiscardDraft discards the draft changes and returns the saved version
func (u *WorkflowUsecase) DiscardDraft(ctx context.Context, tenantID, id uuid.UUID) (*domain.Workflow, error) {
	workflow, err := u.workflowRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	// Clear draft
	workflow.ClearDraft()

	// Update workflow
	if err := u.workflowRepo.Update(ctx, workflow); err != nil {
		return nil, err
	}

	// Reload with steps and edges from database (not draft)
	return u.getWorkflowWithStepsEdgesFromDB(ctx, tenantID, id)
}

// RestoreVersion restores a workflow to a specific version
// This creates a new version based on the restored version's definition
func (u *WorkflowUsecase) RestoreVersion(ctx context.Context, tenantID, workflowID uuid.UUID, targetVersion int) (*domain.Workflow, error) {
	// Get the version to restore
	version, err := u.versionRepo.GetByWorkflowAndVersion(ctx, workflowID, targetVersion)
	if err != nil {
		return nil, err
	}

	// Parse the definition
	var definition domain.WorkflowDefinition
	if err := json.Unmarshal(version.Definition, &definition); err != nil {
		return nil, err
	}

	// Save as new version
	return u.Save(ctx, SaveWorkflowInput{
		TenantID:    tenantID,
		ID:          workflowID,
		Name:        definition.Name,
		Description: definition.Description,
		InputSchema: definition.InputSchema,
		Steps:       definition.Steps,
		Edges:       definition.Edges,
	})
}

// deleteAndRecreateStepsEdges deletes existing steps and edges, then recreates them
func (u *WorkflowUsecase) deleteAndRecreateStepsEdges(ctx context.Context, tenantID, workflowID uuid.UUID, steps []domain.Step, edges []domain.Edge) error {
	// Delete all existing edges first (due to foreign key constraints)
	existingEdges, err := u.edgeRepo.ListByWorkflow(ctx, tenantID, workflowID)
	if err != nil {
		return err
	}
	for _, edge := range existingEdges {
		if err := u.edgeRepo.Delete(ctx, tenantID, workflowID, edge.ID); err != nil {
			return err
		}
	}

	// Delete all existing steps
	existingSteps, err := u.stepRepo.ListByWorkflow(ctx, tenantID, workflowID)
	if err != nil {
		return err
	}
	for _, step := range existingSteps {
		if err := u.stepRepo.Delete(ctx, tenantID, workflowID, step.ID); err != nil {
			return err
		}
	}

	// Create new steps
	for i := range steps {
		steps[i].WorkflowID = workflowID
		if err := u.stepRepo.Create(ctx, &steps[i]); err != nil {
			return err
		}
	}

	// Create new edges
	for i := range edges {
		edges[i].WorkflowID = workflowID
		if err := u.edgeRepo.Create(ctx, &edges[i]); err != nil {
			return err
		}
	}

	return nil
}

// getWorkflowWithStepsEdgesFromDB gets workflow with steps and edges directly from DB (ignoring draft)
func (u *WorkflowUsecase) getWorkflowWithStepsEdgesFromDB(ctx context.Context, tenantID, id uuid.UUID) (*domain.Workflow, error) {
	workflow, err := u.workflowRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	// Get steps
	steps, err := u.stepRepo.ListByWorkflow(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	workflow.Steps = make([]domain.Step, len(steps))
	for i, s := range steps {
		workflow.Steps[i] = *s
	}

	// Get edges
	edges, err := u.edgeRepo.ListByWorkflow(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	workflow.Edges = make([]domain.Edge, len(edges))
	for i, e := range edges {
		workflow.Edges[i] = *e
	}

	return workflow, nil
}

// Publish is deprecated - use Save instead
// Kept for backward compatibility
func (u *WorkflowUsecase) Publish(ctx context.Context, tenantID, id uuid.UUID) (*domain.Workflow, error) {
	// Get workflow with current steps and edges
	workflow, err := u.workflowRepo.GetWithStepsAndEdges(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	// Use Save with current data
	return u.Save(ctx, SaveWorkflowInput{
		TenantID:    tenantID,
		ID:          id,
		Name:        workflow.Name,
		Description: workflow.Description,
		InputSchema: workflow.InputSchema,
		Steps:       workflow.Steps,
		Edges:       workflow.Edges,
	})
}

// GetVersion retrieves a specific version of a workflow
func (u *WorkflowUsecase) GetVersion(ctx context.Context, tenantID, workflowID uuid.UUID, version int) (*domain.WorkflowVersion, error) {
	// First verify the workflow exists and belongs to the tenant
	_, err := u.workflowRepo.GetByID(ctx, tenantID, workflowID)
	if err != nil {
		return nil, err
	}

	return u.versionRepo.GetByWorkflowAndVersion(ctx, workflowID, version)
}

// ListVersions retrieves all versions of a workflow
func (u *WorkflowUsecase) ListVersions(ctx context.Context, tenantID, workflowID uuid.UUID) ([]*domain.WorkflowVersion, error) {
	// First verify the workflow exists and belongs to the tenant
	_, err := u.workflowRepo.GetByID(ctx, tenantID, workflowID)
	if err != nil {
		return nil, err
	}

	return u.versionRepo.ListByWorkflow(ctx, workflowID)
}

// ValidateDAG validates the workflow DAG structure
func (u *WorkflowUsecase) ValidateDAG(workflow *domain.Workflow) error {
	if len(workflow.Steps) == 0 {
		return domain.NewValidationError("steps", "workflow must have at least one step")
	}

	// Check for cycles using DFS
	if hasCycle(workflow.Steps, workflow.Edges) {
		return domain.ErrWorkflowHasCycle
	}

	// Check for unconnected steps (except for single-step workflows)
	if len(workflow.Steps) > 1 {
		if hasUnconnectedSteps(workflow.Steps, workflow.Edges) {
			return domain.ErrWorkflowHasUnconnected
		}
	}

	return nil
}

// hasCycle checks if the DAG contains a cycle using DFS
func hasCycle(steps []domain.Step, edges []domain.Edge) bool {
	// Build adjacency list
	adj := make(map[uuid.UUID][]uuid.UUID)
	for _, edge := range edges {
		adj[edge.SourceStepID] = append(adj[edge.SourceStepID], edge.TargetStepID)
	}

	// Track visited states: 0 = unvisited, 1 = visiting, 2 = visited
	state := make(map[uuid.UUID]int)
	for _, step := range steps {
		state[step.ID] = 0
	}

	var dfs func(id uuid.UUID) bool
	dfs = func(id uuid.UUID) bool {
		state[id] = 1 // visiting
		for _, neighbor := range adj[id] {
			if state[neighbor] == 1 {
				return true // back edge found = cycle
			}
			if state[neighbor] == 0 && dfs(neighbor) {
				return true
			}
		}
		state[id] = 2 // visited
		return false
	}

	for _, step := range steps {
		if state[step.ID] == 0 && dfs(step.ID) {
			return true
		}
	}

	return false
}

// hasUnconnectedSteps checks if any step is not connected to the graph
func hasUnconnectedSteps(steps []domain.Step, edges []domain.Edge) bool {
	if len(steps) <= 1 {
		return false
	}

	connected := make(map[uuid.UUID]bool)
	for _, edge := range edges {
		connected[edge.SourceStepID] = true
		connected[edge.TargetStepID] = true
	}

	for _, step := range steps {
		if !connected[step.ID] {
			return true
		}
	}

	return false
}
