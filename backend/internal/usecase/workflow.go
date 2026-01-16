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
	workflowRepo   repository.WorkflowRepository
	stepRepo       repository.StepRepository
	edgeRepo       repository.EdgeRepository
	versionRepo    repository.WorkflowVersionRepository
	blockRepo      repository.BlockDefinitionRepository
	blockGroupRepo repository.BlockGroupRepository
}

// NewWorkflowUsecase creates a new WorkflowUsecase
func NewWorkflowUsecase(
	workflowRepo repository.WorkflowRepository,
	stepRepo repository.StepRepository,
	edgeRepo repository.EdgeRepository,
	versionRepo repository.WorkflowVersionRepository,
	blockRepo repository.BlockDefinitionRepository,
) *WorkflowUsecase {
	return &WorkflowUsecase{
		workflowRepo: workflowRepo,
		stepRepo:     stepRepo,
		edgeRepo:     edgeRepo,
		versionRepo:  versionRepo,
		blockRepo:    blockRepo,
	}
}

// WithBlockGroupRepo sets the block group repository for port validation
func (u *WorkflowUsecase) WithBlockGroupRepo(repo repository.BlockGroupRepository) *WorkflowUsecase {
	u.blockGroupRepo = repo
	return u
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
	input.Page, input.Limit = NormalizePagination(input.Page, input.Limit)

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
	BlockGroups []domain.BlockGroup
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
	workflow.Steps = input.Steps
	workflow.Edges = input.Edges

	// Derive input_schema from first executable step's block definition
	// This ensures workflow.InputSchema always reflects the actual first step's requirements
	derivedSchema, _ := u.deriveInputSchemaFromFirstStep(ctx, input.Steps, input.Edges)
	if derivedSchema != nil {
		workflow.InputSchema = derivedSchema
	} else {
		// Fallback to provided input_schema if derivation fails
		workflow.InputSchema = input.InputSchema
	}

	// Validate DAG before saving
	if err := u.ValidateDAG(workflow); err != nil {
		return nil, err
	}

	// Validate edge ports if block group repository is available
	if u.blockGroupRepo != nil {
		// Get block groups from database for validation
		blockGroups, err := u.blockGroupRepo.ListByWorkflow(ctx, input.TenantID, input.ID)
		if err != nil {
			return nil, err
		}
		blockGroupSlice := make([]domain.BlockGroup, len(blockGroups))
		for i, bg := range blockGroups {
			blockGroupSlice[i] = *bg
		}
		if err := u.validateEdgePorts(ctx, input.Steps, input.Edges, blockGroupSlice); err != nil {
			return nil, err
		}
	}

	// Delete existing steps and edges, then recreate
	if err := u.deleteAndRecreateStepsEdges(ctx, input.TenantID, workflow.ID, input.Steps, input.Edges); err != nil {
		return nil, err
	}

	// Increment version
	workflow.IncrementVersion()

	// Clear any existing draft
	workflow.ClearDraft()

	// Reload block groups from database for version snapshot
	reloadedWorkflow, err := u.workflowRepo.GetWithStepsAndEdges(ctx, input.TenantID, input.ID)
	if err != nil {
		return nil, err
	}

	// Create workflow definition snapshot
	definition := domain.WorkflowDefinition{
		Name:         workflow.Name,
		Description:  workflow.Description,
		InputSchema:  workflow.InputSchema,
		OutputSchema: workflow.OutputSchema,
		Steps:        input.Steps,
		Edges:        input.Edges,
		BlockGroups:  reloadedWorkflow.BlockGroups,
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

	// Derive input_schema from first executable step's block definition
	derivedSchema, _ := u.deriveInputSchemaFromFirstStep(ctx, input.Steps, input.Edges)
	inputSchema := input.InputSchema
	if derivedSchema != nil {
		inputSchema = derivedSchema
	}

	// Validate edge ports if block group repository is available
	if u.blockGroupRepo != nil {
		// Get block groups from database for validation
		blockGroups, err := u.blockGroupRepo.ListByWorkflow(ctx, input.TenantID, input.ID)
		if err != nil {
			return nil, err
		}
		blockGroupSlice := make([]domain.BlockGroup, len(blockGroups))
		for i, bg := range blockGroups {
			blockGroupSlice[i] = *bg
		}
		if err := u.validateEdgePorts(ctx, input.Steps, input.Edges, blockGroupSlice); err != nil {
			return nil, err
		}
	}

	// Create draft data
	draft := &domain.WorkflowDraft{
		Name:         input.Name,
		Description:  input.Description,
		InputSchema:  inputSchema,
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
		steps[i].TenantID = tenantID
		steps[i].WorkflowID = workflowID
		if err := u.stepRepo.Create(ctx, &steps[i]); err != nil {
			return err
		}
	}

	// Create new edges
	for i := range edges {
		edges[i].TenantID = tenantID
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

	// Use Save with current data (including BlockGroups)
	return u.Save(ctx, SaveWorkflowInput{
		TenantID:    tenantID,
		ID:          id,
		Name:        workflow.Name,
		Description: workflow.Description,
		InputSchema: workflow.InputSchema,
		Steps:       workflow.Steps,
		Edges:       workflow.Edges,
		BlockGroups: workflow.BlockGroups,
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

	// Check for branching blocks (condition/switch) with multiple outputs outside Block Groups
	if err := validateBranchingBlocksInGroups(workflow.Steps, workflow.Edges); err != nil {
		return err
	}

	return nil
}

// hasCycle checks if the DAG contains a cycle using DFS
func hasCycle(steps []domain.Step, edges []domain.Edge) bool {
	// Build adjacency list (only for step-to-step edges)
	adj := make(map[uuid.UUID][]uuid.UUID)
	for _, edge := range edges {
		if edge.SourceStepID != nil && edge.TargetStepID != nil {
			adj[*edge.SourceStepID] = append(adj[*edge.SourceStepID], *edge.TargetStepID)
		}
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
		if edge.SourceStepID != nil {
			connected[*edge.SourceStepID] = true
		}
		if edge.TargetStepID != nil {
			connected[*edge.TargetStepID] = true
		}
	}

	for _, step := range steps {
		if !connected[step.ID] {
			return true
		}
	}

	return false
}

// validateBranchingBlocksInGroups checks that branching blocks (condition/switch) with multiple output edges
// are contained within a Block Group. This prevents complex parallel flows outside of managed group contexts.
func validateBranchingBlocksInGroups(steps []domain.Step, edges []domain.Edge) error {
	// Count outgoing edges per step (only step-to-step edges)
	outgoingEdgeCount := make(map[uuid.UUID]int)
	for _, edge := range edges {
		if edge.SourceStepID != nil {
			outgoingEdgeCount[*edge.SourceStepID]++
		}
	}

	// Check each branching block
	for _, step := range steps {
		// Only check condition and switch blocks
		if step.Type != domain.StepTypeCondition && step.Type != domain.StepTypeSwitch {
			continue
		}

		// If this branching block has multiple outgoing edges, it must be in a Block Group
		if outgoingEdgeCount[step.ID] > 1 && step.BlockGroupID == nil {
			return domain.ErrWorkflowBranchOutsideGroup
		}
	}

	return nil
}

// deriveInputSchemaFromFirstStep derives input_schema from the first executable step's block definition
// This ensures workflow.InputSchema always reflects the actual input requirements of the first step
func (u *WorkflowUsecase) deriveInputSchemaFromFirstStep(ctx context.Context, steps []domain.Step, edges []domain.Edge) (json.RawMessage, error) {
	// 1. Find Start step
	var startStepID uuid.UUID
	for _, step := range steps {
		if step.Type == domain.StepTypeStart {
			startStepID = step.ID
			break
		}
	}
	if startStepID == uuid.Nil {
		return nil, nil // No Start step found
	}

	// 2. Find first step after Start
	var firstStepID uuid.UUID
	for _, edge := range edges {
		if edge.SourceStepID != nil && *edge.SourceStepID == startStepID && edge.TargetStepID != nil {
			firstStepID = *edge.TargetStepID
			break
		}
	}
	if firstStepID == uuid.Nil {
		return nil, nil // No step after Start
	}

	// 3. Get block slug from step
	var blockSlug domain.StepType
	for _, step := range steps {
		if step.ID == firstStepID {
			blockSlug = step.Type
			break
		}
	}
	if blockSlug == "" || blockSlug == domain.StepTypeStart {
		return nil, nil // Not an executable block
	}

	// 4. Get block definition from repository (nil tenantID for system blocks)
	block, err := u.blockRepo.GetBySlug(ctx, nil, string(blockSlug))
	if err != nil {
		// Block not found - return nil without error
		return nil, nil
	}

	return block.InputSchema, nil
}

// validateEdgePorts validates that all edges reference valid ports
func (u *WorkflowUsecase) validateEdgePorts(ctx context.Context, steps []domain.Step, edges []domain.Edge, blockGroups []domain.BlockGroup) error {
	// Build step map for quick lookup
	stepMap := make(map[uuid.UUID]*domain.Step)
	for i := range steps {
		stepMap[steps[i].ID] = &steps[i]
	}

	// Build block group map for quick lookup
	groupMap := make(map[uuid.UUID]*domain.BlockGroup)
	for i := range blockGroups {
		groupMap[blockGroups[i].ID] = &blockGroups[i]
	}

	for _, edge := range edges {
		// Validate source port
		if edge.SourcePort != "" {
			if err := u.validateSourcePort(ctx, edge.SourcePort, edge.SourceStepID, edge.SourceBlockGroupID, stepMap, groupMap); err != nil {
				return err
			}
		}

		// Validate target port
		if edge.TargetPort != "" {
			if err := u.validateTargetPort(ctx, edge.TargetPort, edge.TargetStepID, edge.TargetBlockGroupID, stepMap, groupMap); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateSourcePort validates that the source port exists in the block definition
func (u *WorkflowUsecase) validateSourcePort(ctx context.Context, sourcePort string, sourceStepID, sourceGroupID *uuid.UUID, stepMap map[uuid.UUID]*domain.Step, groupMap map[uuid.UUID]*domain.BlockGroup) error {
	var blockDef *domain.BlockDefinition
	var err error

	if sourceStepID != nil {
		if step, ok := stepMap[*sourceStepID]; ok {
			blockDef, err = u.getBlockDefinitionForStep(ctx, step)
		}
	} else if sourceGroupID != nil {
		if group, ok := groupMap[*sourceGroupID]; ok {
			blockDef, err = u.getBlockDefinitionForGroup(ctx, group)
		}
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
func (u *WorkflowUsecase) validateTargetPort(ctx context.Context, targetPort string, targetStepID, targetGroupID *uuid.UUID, stepMap map[uuid.UUID]*domain.Step, groupMap map[uuid.UUID]*domain.BlockGroup) error {
	// Special case: "group-input" is a virtual port for block groups
	if targetGroupID != nil && targetPort == "group-input" {
		return nil
	}

	var blockDef *domain.BlockDefinition
	var err error

	if targetStepID != nil {
		if step, ok := stepMap[*targetStepID]; ok {
			blockDef, err = u.getBlockDefinitionForStep(ctx, step)
		}
	} else if targetGroupID != nil {
		if group, ok := groupMap[*targetGroupID]; ok {
			blockDef, err = u.getBlockDefinitionForGroup(ctx, group)
		}
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
func (u *WorkflowUsecase) getBlockDefinitionForStep(ctx context.Context, step *domain.Step) (*domain.BlockDefinition, error) {
	if step.BlockDefinitionID != nil {
		return u.blockRepo.GetByID(ctx, *step.BlockDefinitionID)
	}
	// Use step type as slug for legacy steps
	return u.blockRepo.GetBySlug(ctx, nil, string(step.Type))
}

// getBlockDefinitionForGroup retrieves the block definition for a block group
func (u *WorkflowUsecase) getBlockDefinitionForGroup(ctx context.Context, group *domain.BlockGroup) (*domain.BlockDefinition, error) {
	// Use group type as slug
	return u.blockRepo.GetBySlug(ctx, nil, string(group.Type))
}
