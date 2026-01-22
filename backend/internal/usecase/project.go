package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// ProjectUsecase handles project business logic
type ProjectUsecase struct {
	projectRepo    repository.ProjectRepository
	stepRepo       repository.StepRepository
	edgeRepo       repository.EdgeRepository
	versionRepo    repository.ProjectVersionRepository
	blockRepo      repository.BlockDefinitionRepository
	blockGroupRepo repository.BlockGroupRepository
}

// NewProjectUsecase creates a new ProjectUsecase
func NewProjectUsecase(
	projectRepo repository.ProjectRepository,
	stepRepo repository.StepRepository,
	edgeRepo repository.EdgeRepository,
	versionRepo repository.ProjectVersionRepository,
	blockRepo repository.BlockDefinitionRepository,
) *ProjectUsecase {
	return &ProjectUsecase{
		projectRepo: projectRepo,
		stepRepo:    stepRepo,
		edgeRepo:    edgeRepo,
		versionRepo: versionRepo,
		blockRepo:   blockRepo,
	}
}

// WithBlockGroupRepo sets the block group repository for port validation
func (u *ProjectUsecase) WithBlockGroupRepo(repo repository.BlockGroupRepository) *ProjectUsecase {
	u.blockGroupRepo = repo
	return u
}

// CreateProjectInput represents input for creating a project
type CreateProjectInput struct {
	TenantID    uuid.UUID
	Name        string
	Description string
	Variables   json.RawMessage
}

// Create creates a new project with an auto-created Start node
func (u *ProjectUsecase) Create(ctx context.Context, input CreateProjectInput) (*domain.Project, error) {
	if input.Name == "" {
		return nil, domain.NewValidationError("name", "name is required")
	}

	project := domain.NewProject(input.TenantID, input.Name, input.Description)

	if err := u.projectRepo.Create(ctx, project); err != nil {
		return nil, err
	}

	// Auto-create Start step for the new project with manual trigger
	triggerType := domain.StepTriggerTypeManual
	startStep := &domain.Step{
		ID:            uuid.New(),
		TenantID:      input.TenantID,
		ProjectID:     project.ID,
		Name:          "Start",
		Type:          domain.StepType("manual_trigger"),
		Config:        json.RawMessage(`{}`),
		TriggerType:   &triggerType,
		TriggerConfig: json.RawMessage(`{}`),
		PositionX:     400,
		PositionY:     50,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	if err := u.stepRepo.Create(ctx, startStep); err != nil {
		// Log error but don't fail project creation
		// The user can manually add a start step if this fails
		return project, nil
	}

	return project, nil
}

// GetByID retrieves a project by ID
func (u *ProjectUsecase) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Project, error) {
	return u.projectRepo.GetByID(ctx, tenantID, id)
}

// GetWithDetails retrieves a project with steps and edges
func (u *ProjectUsecase) GetWithDetails(ctx context.Context, tenantID, id uuid.UUID) (*domain.Project, error) {
	return u.projectRepo.GetWithStepsAndEdges(ctx, tenantID, id)
}

// ListProjectsInput represents input for listing projects
type ListProjectsInput struct {
	TenantID uuid.UUID
	Status   *domain.ProjectStatus
	Page     int
	Limit    int
}

// ListProjectsOutput represents output for listing projects
type ListProjectsOutput struct {
	Projects []*domain.Project
	Total    int
	Page     int
	Limit    int
}

// List lists projects with pagination
func (u *ProjectUsecase) List(ctx context.Context, input ListProjectsInput) (*ListProjectsOutput, error) {
	input.Page, input.Limit = NormalizePagination(input.Page, input.Limit)

	filter := repository.ProjectFilter{
		Status: input.Status,
		Page:   input.Page,
		Limit:  input.Limit,
	}

	projects, total, err := u.projectRepo.List(ctx, input.TenantID, filter)
	if err != nil {
		return nil, err
	}

	return &ListProjectsOutput{
		Projects: projects,
		Total:    total,
		Page:     input.Page,
		Limit:    input.Limit,
	}, nil
}

// UpdateProjectInput represents input for updating a project
type UpdateProjectInput struct {
	TenantID    uuid.UUID
	ID          uuid.UUID
	Name        string
	Description string
	Variables   json.RawMessage
}

// Update updates a project
func (u *ProjectUsecase) Update(ctx context.Context, input UpdateProjectInput) (*domain.Project, error) {
	project, err := u.projectRepo.GetByID(ctx, input.TenantID, input.ID)
	if err != nil {
		return nil, err
	}

	if !project.CanEdit() {
		return nil, domain.ErrProjectNotEditable
	}

	if input.Name != "" {
		project.Name = input.Name
	}
	project.Description = input.Description

	if err := u.projectRepo.Update(ctx, project); err != nil {
		return nil, err
	}

	return project, nil
}

// Delete deletes a project
// System projects cannot be deleted
func (u *ProjectUsecase) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	// First, check if the project exists and is a system project
	project, err := u.projectRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	// Prevent deletion of system projects
	if project.IsSystem {
		return domain.ErrForbidden
	}

	return u.projectRepo.Delete(ctx, tenantID, id)
}

// SaveProjectInput represents input for saving a project
type SaveProjectInput struct {
	TenantID    uuid.UUID
	ID          uuid.UUID
	Name        string
	Description string
	Variables   json.RawMessage
	Steps       []domain.Step
	Edges       []domain.Edge
	BlockGroups []domain.BlockGroup
}

// Save saves a project and creates a new version snapshot
// This replaces the old "Publish" functionality
func (u *ProjectUsecase) Save(ctx context.Context, input SaveProjectInput) (*domain.Project, error) {
	project, err := u.projectRepo.GetByID(ctx, input.TenantID, input.ID)
	if err != nil {
		return nil, err
	}

	// Update project fields
	if input.Name != "" {
		project.Name = input.Name
	}
	project.Description = input.Description
	if input.Variables != nil {
		project.Variables = input.Variables
	}
	project.Steps = input.Steps
	project.Edges = input.Edges

	// Validate DAG before saving
	if err := u.ValidateDAG(project); err != nil {
		return nil, err
	}

	// Validate edge ports if block group repository is available
	if u.blockGroupRepo != nil {
		// Get block groups from database for validation
		blockGroups, err := u.blockGroupRepo.ListByProject(ctx, input.TenantID, input.ID)
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
	if err := u.deleteAndRecreateStepsEdges(ctx, input.TenantID, project.ID, input.Steps, input.Edges); err != nil {
		return nil, err
	}

	// Increment version
	project.IncrementVersion()

	// Clear any existing draft
	project.ClearDraft()

	// Reload block groups from database for version snapshot
	reloadedProject, err := u.projectRepo.GetWithStepsAndEdges(ctx, input.TenantID, input.ID)
	if err != nil {
		return nil, err
	}

	// Create project definition snapshot
	definition := domain.ProjectDefinition{
		Name:        project.Name,
		Description: project.Description,
		Variables:   project.Variables,
		Steps:       input.Steps,
		Edges:       input.Edges,
		BlockGroups: reloadedProject.BlockGroups,
	}

	definitionJSON, err := json.Marshal(definition)
	if err != nil {
		return nil, err
	}

	// Create version record
	projectVersion := &domain.ProjectVersion{
		ID:         uuid.New(),
		ProjectID:  project.ID,
		Version:    project.Version,
		Definition: definitionJSON,
		SavedAt:    time.Now().UTC(),
	}

	// Save version snapshot
	if err := u.versionRepo.Create(ctx, projectVersion); err != nil {
		return nil, err
	}

	// Update project
	if err := u.projectRepo.Update(ctx, project); err != nil {
		return nil, err
	}

	// Reload with steps and edges
	return u.projectRepo.GetWithStepsAndEdges(ctx, input.TenantID, input.ID)
}

// Publish publishes a project by creating a new version with current steps and edges.
// This is a convenience method that fetches current data and calls Save.
func (u *ProjectUsecase) Publish(ctx context.Context, tenantID, projectID uuid.UUID) (*domain.Project, error) {
	// Get project with current steps and edges
	project, err := u.projectRepo.GetWithStepsAndEdges(ctx, tenantID, projectID)
	if err != nil {
		return nil, err
	}

	// Call Save with current data
	return u.Save(ctx, SaveProjectInput{
		TenantID:    tenantID,
		ID:          projectID,
		Name:        project.Name,
		Description: project.Description,
		Variables:   project.Variables,
		Steps:       project.Steps,
		Edges:       project.Edges,
	})
}

// SaveDraftInput represents input for saving a project as draft
type SaveDraftInput struct {
	TenantID    uuid.UUID
	ID          uuid.UUID
	Name        string
	Description string
	Variables   json.RawMessage
	Steps       []domain.Step
	Edges       []domain.Edge
}

// SaveDraft saves a project as draft without creating a new version
// Draft changes are not validated and not persisted to steps/edges tables
func (u *ProjectUsecase) SaveDraft(ctx context.Context, input SaveDraftInput) (*domain.Project, error) {
	project, err := u.projectRepo.GetByID(ctx, input.TenantID, input.ID)
	if err != nil {
		return nil, err
	}

	// Validate edge ports if block group repository is available
	if u.blockGroupRepo != nil {
		// Get block groups from database for validation
		blockGroups, err := u.blockGroupRepo.ListByProject(ctx, input.TenantID, input.ID)
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
	draft := &domain.ProjectDraft{
		Name:        input.Name,
		Description: input.Description,
		Variables:   input.Variables,
		Steps:       input.Steps,
		Edges:       input.Edges,
		UpdatedAt:   time.Now().UTC(),
	}

	// Set draft
	if err := project.SetDraft(draft); err != nil {
		return nil, err
	}

	// Update project (only draft field changes)
	if err := u.projectRepo.Update(ctx, project); err != nil {
		return nil, err
	}

	// Return project with draft data applied
	project.Name = draft.Name
	project.Description = draft.Description
	project.Variables = draft.Variables
	project.Steps = draft.Steps
	project.Edges = draft.Edges
	project.HasDraft = true

	return project, nil
}

// DiscardDraft discards the draft changes and returns the saved version
func (u *ProjectUsecase) DiscardDraft(ctx context.Context, tenantID, id uuid.UUID) (*domain.Project, error) {
	project, err := u.projectRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	// Clear draft
	project.ClearDraft()

	// Update project
	if err := u.projectRepo.Update(ctx, project); err != nil {
		return nil, err
	}

	// Reload with steps and edges from database (not draft)
	return u.getProjectWithStepsEdgesFromDB(ctx, tenantID, id)
}

// RestoreVersion restores a project to a specific version
// This creates a new version based on the restored version's definition
func (u *ProjectUsecase) RestoreVersion(ctx context.Context, tenantID, projectID uuid.UUID, targetVersion int) (*domain.Project, error) {
	// Get the version to restore
	version, err := u.versionRepo.GetByProjectAndVersion(ctx, projectID, targetVersion)
	if err != nil {
		return nil, err
	}

	// Parse the definition
	var definition domain.ProjectDefinition
	if err := json.Unmarshal(version.Definition, &definition); err != nil {
		return nil, err
	}

	// Save as new version
	return u.Save(ctx, SaveProjectInput{
		TenantID:    tenantID,
		ID:          projectID,
		Name:        definition.Name,
		Description: definition.Description,
		Variables:   definition.Variables,
		Steps:       definition.Steps,
		Edges:       definition.Edges,
	})
}

// deleteAndRecreateStepsEdges deletes existing steps and edges, then recreates them
func (u *ProjectUsecase) deleteAndRecreateStepsEdges(ctx context.Context, tenantID, projectID uuid.UUID, steps []domain.Step, edges []domain.Edge) error {
	// Delete all existing edges first (due to foreign key constraints)
	existingEdges, err := u.edgeRepo.ListByProject(ctx, tenantID, projectID)
	if err != nil {
		return err
	}
	for _, edge := range existingEdges {
		if err := u.edgeRepo.Delete(ctx, tenantID, projectID, edge.ID); err != nil {
			return err
		}
	}

	// Delete all existing steps
	existingSteps, err := u.stepRepo.ListByProject(ctx, tenantID, projectID)
	if err != nil {
		return err
	}
	for _, step := range existingSteps {
		if err := u.stepRepo.Delete(ctx, tenantID, projectID, step.ID); err != nil {
			return err
		}
	}

	// Create new steps
	for i := range steps {
		steps[i].TenantID = tenantID
		steps[i].ProjectID = projectID
		if err := u.stepRepo.Create(ctx, &steps[i]); err != nil {
			return err
		}
	}

	// Create new edges
	for i := range edges {
		edges[i].TenantID = tenantID
		edges[i].ProjectID = projectID
		if err := u.edgeRepo.Create(ctx, &edges[i]); err != nil {
			return err
		}
	}

	return nil
}

// getProjectWithStepsEdgesFromDB gets project with steps and edges directly from DB (ignoring draft)
func (u *ProjectUsecase) getProjectWithStepsEdgesFromDB(ctx context.Context, tenantID, id uuid.UUID) (*domain.Project, error) {
	project, err := u.projectRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	// Get steps
	steps, err := u.stepRepo.ListByProject(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	project.Steps = make([]domain.Step, len(steps))
	for i, s := range steps {
		project.Steps[i] = *s
	}

	// Get edges
	edges, err := u.edgeRepo.ListByProject(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	project.Edges = make([]domain.Edge, len(edges))
	for i, e := range edges {
		project.Edges[i] = *e
	}

	return project, nil
}

// GetVersion retrieves a specific version of a project
func (u *ProjectUsecase) GetVersion(ctx context.Context, tenantID, projectID uuid.UUID, version int) (*domain.ProjectVersion, error) {
	// First verify the project exists and belongs to the tenant
	_, err := u.projectRepo.GetByID(ctx, tenantID, projectID)
	if err != nil {
		return nil, err
	}

	return u.versionRepo.GetByProjectAndVersion(ctx, projectID, version)
}

// ListVersions retrieves all versions of a project
func (u *ProjectUsecase) ListVersions(ctx context.Context, tenantID, projectID uuid.UUID) ([]*domain.ProjectVersion, error) {
	// First verify the project exists and belongs to the tenant
	_, err := u.projectRepo.GetByID(ctx, tenantID, projectID)
	if err != nil {
		return nil, err
	}

	return u.versionRepo.ListByProject(ctx, projectID)
}

// ValidateDAG validates the project DAG structure
func (u *ProjectUsecase) ValidateDAG(project *domain.Project) error {
	if len(project.Steps) == 0 {
		return domain.NewValidationError("steps", "project must have at least one step")
	}

	// Check for cycles using DFS
	if hasCycle(project.Steps, project.Edges) {
		return domain.ErrProjectHasCycle
	}

	// Check for unconnected steps (except for single-step projects)
	if len(project.Steps) > 1 {
		if hasUnconnectedSteps(project.Steps, project.Edges) {
			return domain.ErrProjectHasUnconnected
		}
	}

	// Check for branching blocks (condition/switch) with multiple outputs outside Block Groups
	if err := validateBranchingBlocksInGroups(project.Steps, project.Edges); err != nil {
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
			return domain.ErrProjectBranchOutsideGroup
		}
	}

	return nil
}

// validateEdgePorts validates that all edges reference valid ports
func (u *ProjectUsecase) validateEdgePorts(ctx context.Context, steps []domain.Step, edges []domain.Edge, blockGroups []domain.BlockGroup) error {
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
	}

	return nil
}

// validateSourcePort validates that the source port exists in the block definition
func (u *ProjectUsecase) validateSourcePort(ctx context.Context, sourcePort string, sourceStepID, sourceGroupID *uuid.UUID, stepMap map[uuid.UUID]*domain.Step, groupMap map[uuid.UUID]*domain.BlockGroup) error {
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

// getBlockDefinitionForStep retrieves the block definition for a step
func (u *ProjectUsecase) getBlockDefinitionForStep(ctx context.Context, step *domain.Step) (*domain.BlockDefinition, error) {
	if step.BlockDefinitionID == nil {
		return nil, domain.ErrBlockDefinitionNotFound
	}
	return u.blockRepo.GetByID(ctx, *step.BlockDefinitionID)
}

// getBlockDefinitionForGroup retrieves the block definition for a block group
func (u *ProjectUsecase) getBlockDefinitionForGroup(ctx context.Context, group *domain.BlockGroup) (*domain.BlockDefinition, error) {
	// Use group type as slug
	return u.blockRepo.GetBySlug(ctx, nil, string(group.Type))
}

// ValidationCheck represents a single validation check result
type ValidationCheck struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Status  string `json:"status"` // "passed", "warning", "error"
	Message string `json:"message,omitempty"`
}

// ValidationResult represents the result of ValidateForPublish
type ValidationResult struct {
	Checks       []ValidationCheck `json:"checks"`
	CanPublish   bool              `json:"can_publish"`
	ErrorCount   int               `json:"error_count"`
	WarningCount int               `json:"warning_count"`
}

// ValidateForPublish validates a project before publishing
// Returns a list of checks with their status
func (u *ProjectUsecase) ValidateForPublish(ctx context.Context, tenantID, projectID uuid.UUID) (*ValidationResult, error) {
	// Get project with steps and edges
	project, err := u.projectRepo.GetWithStepsAndEdges(ctx, tenantID, projectID)
	if err != nil {
		return nil, err
	}

	result := &ValidationResult{
		Checks:     make([]ValidationCheck, 0),
		CanPublish: true,
	}

	// Check 1: Start block exists
	startCheck := ValidationCheck{
		ID:     "hasStartBlock",
		Label:  "Start block exists",
		Status: "passed",
	}
	hasStartBlock := false
	startBlockTypes := []domain.StepType{
		domain.StepTypeStart,
		"manual_trigger",
		"schedule_trigger",
		"webhook_trigger",
	}
	for _, step := range project.Steps {
		for _, startType := range startBlockTypes {
			if step.Type == startType {
				hasStartBlock = true
				break
			}
		}
		if hasStartBlock {
			break
		}
	}
	if !hasStartBlock {
		startCheck.Status = "error"
		startCheck.Message = "Add a start block to begin the workflow"
		result.CanPublish = false
		result.ErrorCount++
	}
	result.Checks = append(result.Checks, startCheck)

	// Check 2: All blocks connected
	connectedCheck := ValidationCheck{
		ID:     "allConnected",
		Label:  "All blocks are connected",
		Status: "passed",
	}
	if len(project.Steps) > 1 {
		connectedIDs := make(map[uuid.UUID]bool)
		for _, edge := range project.Edges {
			if edge.SourceStepID != nil {
				connectedIDs[*edge.SourceStepID] = true
			}
			if edge.TargetStepID != nil {
				connectedIDs[*edge.TargetStepID] = true
			}
		}
		unconnectedCount := 0
		for _, step := range project.Steps {
			if !connectedIDs[step.ID] {
				unconnectedCount++
			}
		}
		if unconnectedCount > 0 {
			connectedCheck.Status = "warning"
			connectedCheck.Message = fmt.Sprintf("%d unconnected block(s)", unconnectedCount)
			result.WarningCount++
		}
	}
	result.Checks = append(result.Checks, connectedCheck)

	// Check 3: No infinite loops
	loopCheck := ValidationCheck{
		ID:     "noLoop",
		Label:  "No infinite loop detected",
		Status: "passed",
	}
	if hasCycle(project.Steps, project.Edges) {
		loopCheck.Status = "error"
		loopCheck.Message = "Circular reference detected in the workflow"
		result.CanPublish = false
		result.ErrorCount++
	}
	result.Checks = append(result.Checks, loopCheck)

	// Check 4: Credentials configured
	credentialsCheck := ValidationCheck{
		ID:     "credentialsSet",
		Label:  "Required credentials are set",
		Status: "passed",
	}
	missingCredentials := 0
	for i := range project.Steps {
		step := &project.Steps[i]
		blockDef, err := u.getBlockDefinitionForStep(ctx, step)
		if err != nil || blockDef == nil {
			continue
		}
		reqCreds, err := blockDef.GetRequiredCredentials()
		if err != nil || len(reqCreds) == 0 {
			continue
		}
		bindings, err := step.GetCredentialBindings()
		if err != nil {
			continue
		}
		for _, required := range reqCreds {
			if !required.Required {
				continue
			}
			if _, ok := bindings[required.Name]; !ok {
				missingCredentials++
			}
		}
	}
	if missingCredentials > 0 {
		credentialsCheck.Status = "warning"
		credentialsCheck.Message = fmt.Sprintf("%d missing credential binding(s)", missingCredentials)
		result.WarningCount++
	}
	result.Checks = append(result.Checks, credentialsCheck)

	// Check 5: Trigger configuration
	triggerCheck := ValidationCheck{
		ID:     "triggerConfigured",
		Label:  "Trigger is properly configured",
		Status: "passed",
	}
	triggerConfigured := false
	triggerEnabled := false
	for _, step := range project.Steps {
		if step.Type == domain.StepTypeStart || domain.IsTriggerBlockSlug(string(step.Type)) {
			triggerConfigured = true
			if step.TriggerType != nil {
				// Check if trigger is enabled (for non-manual triggers)
				if *step.TriggerType == domain.StepTriggerTypeManual {
					triggerEnabled = true // Manual triggers are always "enabled"
				} else if len(step.TriggerConfig) > 0 {
					var config map[string]interface{}
					if err := json.Unmarshal(step.TriggerConfig, &config); err == nil {
						if enabled, ok := config["enabled"].(bool); ok && enabled {
							triggerEnabled = true
						}
					}
				}
			}
			break
		}
	}
	if !triggerConfigured {
		triggerCheck.Status = "warning"
		triggerCheck.Message = "No trigger block found"
		result.WarningCount++
	} else if !triggerEnabled {
		triggerCheck.Status = "warning"
		triggerCheck.Message = "Trigger is configured but not enabled"
		result.WarningCount++
	}
	result.Checks = append(result.Checks, triggerCheck)

	// Check 6: Required step config fields (check for empty required fields)
	configCheck := ValidationCheck{
		ID:     "requiredConfigSet",
		Label:  "Required step configurations are set",
		Status: "passed",
	}
	missingConfig := 0
	for i := range project.Steps {
		step := &project.Steps[i]
		blockDef, err := u.getBlockDefinitionForStep(ctx, step)
		if err != nil || blockDef == nil {
			continue
		}
		// Check if config schema has required fields
		if blockDef.ConfigSchema != nil {
			var schema map[string]interface{}
			if err := json.Unmarshal(blockDef.ConfigSchema, &schema); err == nil {
				if required, ok := schema["required"].([]interface{}); ok && len(required) > 0 {
					var stepConfig map[string]interface{}
					if err := json.Unmarshal(step.Config, &stepConfig); err == nil {
						for _, reqField := range required {
							if fieldName, ok := reqField.(string); ok {
								if _, exists := stepConfig[fieldName]; !exists {
									missingConfig++
								}
							}
						}
					}
				}
			}
		}
	}
	if missingConfig > 0 {
		configCheck.Status = "warning"
		configCheck.Message = fmt.Sprintf("%d step(s) have missing required configuration", missingConfig)
		result.WarningCount++
	}
	result.Checks = append(result.Checks, configCheck)

	return result, nil
}
