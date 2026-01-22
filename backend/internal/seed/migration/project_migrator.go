package migration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
	"github.com/souta/ai-orchestration/internal/seed/workflows"
)

// ProjectMigrationResult tracks what happened during project migration
type ProjectMigrationResult struct {
	Created   []string // SystemSlugs of newly created projects
	Updated   []string // SystemSlugs of updated projects
	Unchanged []string // SystemSlugs of unchanged projects
	Errors    []error
}

// ProjectDryRunResult shows what would happen during migration without applying
type ProjectDryRunResult struct {
	ToCreate  []string            // SystemSlugs of projects to create
	ToUpdate  []ProjectUpdateInfo // Info about projects to update
	Unchanged []string            // SystemSlugs of unchanged projects
}

// ProjectUpdateInfo provides details about a project update
type ProjectUpdateInfo struct {
	SystemSlug string
	OldVersion int
	NewVersion int
	Reason     string
}

// ProjectMigrator handles project definition migration
type ProjectMigrator struct {
	projectRepo    repository.ProjectRepository
	stepRepo       repository.StepRepository
	edgeRepo       repository.EdgeRepository
	blockRepo      repository.BlockDefinitionRepository
	blockGroupRepo repository.BlockGroupRepository
}

// NewProjectMigrator creates a new project migrator
func NewProjectMigrator(
	projectRepo repository.ProjectRepository,
	stepRepo repository.StepRepository,
	edgeRepo repository.EdgeRepository,
) *ProjectMigrator {
	return &ProjectMigrator{
		projectRepo: projectRepo,
		stepRepo:    stepRepo,
		edgeRepo:    edgeRepo,
	}
}

// WithBlockRepo sets the block definition repository for resolving block slugs
func (m *ProjectMigrator) WithBlockRepo(blockRepo repository.BlockDefinitionRepository) *ProjectMigrator {
	m.blockRepo = blockRepo
	return m
}

// WithBlockGroupRepo sets the block group repository for creating block groups
func (m *ProjectMigrator) WithBlockGroupRepo(blockGroupRepo repository.BlockGroupRepository) *ProjectMigrator {
	m.blockGroupRepo = blockGroupRepo
	return m
}

// Migrate performs UPSERT for all projects in the registry
func (m *ProjectMigrator) Migrate(ctx context.Context, registry *workflows.Registry, tenantID uuid.UUID) (*ProjectMigrationResult, error) {
	result := &ProjectMigrationResult{
		Created:   make([]string, 0),
		Updated:   make([]string, 0),
		Unchanged: make([]string, 0),
		Errors:    make([]error, 0),
	}

	for _, seedProject := range registry.GetAll() {
		action, err := m.upsertProject(ctx, seedProject, tenantID)
		if err != nil {
			result.Errors = append(result.Errors,
				fmt.Errorf("project %s: %w", seedProject.SystemSlug, err))
			continue
		}

		switch action {
		case "created":
			result.Created = append(result.Created, seedProject.SystemSlug)
		case "updated":
			result.Updated = append(result.Updated, seedProject.SystemSlug)
		case "unchanged":
			result.Unchanged = append(result.Unchanged, seedProject.SystemSlug)
		}
	}

	return result, nil
}

// upsertProject creates or updates a single project with its steps and edges
func (m *ProjectMigrator) upsertProject(ctx context.Context, seedProject *workflows.SystemWorkflowDefinition, tenantID uuid.UUID) (string, error) {
	// Parse the project ID from the seed
	projectID, err := uuid.Parse(seedProject.ID)
	if err != nil {
		return "", fmt.Errorf("invalid project ID: %w", err)
	}

	// Look up existing project by ID
	existing, err := m.projectRepo.GetByID(ctx, tenantID, projectID)
	if err != nil {
		if errors.Is(err, domain.ErrProjectNotFound) {
			// Project doesn't exist, create it
			return m.createProject(ctx, seedProject, tenantID, projectID)
		}
		return "", fmt.Errorf("failed to get existing project: %w", err)
	}

	// Check if update is needed
	if m.hasChanges(existing, seedProject) {
		// UPDATE existing project
		return m.updateProject(ctx, existing, seedProject, tenantID)
	}

	return "unchanged", nil
}

// createProject creates a new system project with steps and edges
func (m *ProjectMigrator) createProject(ctx context.Context, seedProject *workflows.SystemWorkflowDefinition, tenantID uuid.UUID, projectID uuid.UUID) (string, error) {
	now := time.Now().UTC()

	// Create project
	project := &domain.Project{
		ID:          projectID,
		TenantID:    tenantID,
		Name:        seedProject.Name,
		Description: seedProject.Description,
		Status:      domain.ProjectStatusPublished,
		Version:     seedProject.Version,
		Variables:   seedProject.InputSchema, // Seed definition uses InputSchema field name
		IsSystem:    true,
		SystemSlug:  &seedProject.SystemSlug,
		PublishedAt: &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := m.projectRepo.Create(ctx, project); err != nil {
		return "", fmt.Errorf("failed to create project: %w", err)
	}

	// Create block groups and build temp_id -> actual_id mapping
	groupIDMap, err := m.createBlockGroups(ctx, seedProject, tenantID, projectID)
	if err != nil {
		return "", fmt.Errorf("failed to create block groups: %w", err)
	}

	// Create steps and build temp_id -> actual_id mapping
	stepIDMap, err := m.createSteps(ctx, seedProject, tenantID, projectID, groupIDMap)
	if err != nil {
		return "", fmt.Errorf("failed to create steps: %w", err)
	}

	// Create edges using the step ID and group ID mappings
	if err := m.createEdgesWithGroupMap(ctx, seedProject, tenantID, projectID, stepIDMap, groupIDMap); err != nil {
		return "", fmt.Errorf("failed to create edges: %w", err)
	}

	return "created", nil
}

// createBlockGroups creates all block groups for a project and returns a temp_id -> group_id mapping
func (m *ProjectMigrator) createBlockGroups(ctx context.Context, seedProject *workflows.SystemWorkflowDefinition, tenantID uuid.UUID, projectID uuid.UUID) (map[string]uuid.UUID, error) {
	groupIDMap := make(map[string]uuid.UUID)

	if len(seedProject.BlockGroups) == 0 || m.blockGroupRepo == nil {
		return groupIDMap, nil
	}

	now := time.Now().UTC()

	// First pass: create all groups without parent references
	for _, seedGroup := range seedProject.BlockGroups {
		groupID := uuid.New()
		groupIDMap[seedGroup.TempID] = groupID

		width := seedGroup.Width
		if width == 0 {
			width = 400
		}
		height := seedGroup.Height
		if height == 0 {
			height = 300
		}

		var preProcess, postProcess *string
		if seedGroup.PreProcess != "" {
			preProcess = &seedGroup.PreProcess
		}
		if seedGroup.PostProcess != "" {
			postProcess = &seedGroup.PostProcess
		}

		group := &domain.BlockGroup{
			ID:          groupID,
			TenantID:    tenantID,
			ProjectID:   projectID,
			Name:        seedGroup.Name,
			Type:        domain.BlockGroupType(seedGroup.Type),
			Config:      seedGroup.Config,
			PreProcess:  preProcess,
			PostProcess: postProcess,
			PositionX:   seedGroup.PositionX,
			PositionY:   seedGroup.PositionY,
			Width:       width,
			Height:      height,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		// Set parent group if specified
		if seedGroup.ParentTempID != "" {
			if parentID, ok := groupIDMap[seedGroup.ParentTempID]; ok {
				group.ParentGroupID = &parentID
			}
		}

		if err := m.blockGroupRepo.Create(ctx, group); err != nil {
			return nil, fmt.Errorf("failed to create block group %s: %w", seedGroup.Name, err)
		}
	}

	return groupIDMap, nil
}

// createSteps creates all steps for a project and returns a temp_id -> step_id mapping
func (m *ProjectMigrator) createSteps(ctx context.Context, seedProject *workflows.SystemWorkflowDefinition, tenantID uuid.UUID, projectID uuid.UUID, groupIDMap map[string]uuid.UUID) (map[string]uuid.UUID, error) {
	stepIDMap := make(map[string]uuid.UUID)
	now := time.Now().UTC()

	for _, seedStep := range seedProject.Steps {
		stepID := uuid.New()
		stepIDMap[seedStep.TempID] = stepID

		var blockDefID *uuid.UUID

		// Resolve block ID from slug
		if seedStep.BlockSlug != "" && m.blockRepo != nil {
			// Look up block by slug (system blocks have tenant_id = NULL)
			block, err := m.blockRepo.GetBySlug(ctx, nil, seedStep.BlockSlug)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve block slug %s: %w", seedStep.BlockSlug, err)
			}
			if block != nil {
				blockDefID = &block.ID
			}
		}

		// Resolve block group ID from temp_id
		var blockGroupID *uuid.UUID
		if seedStep.BlockGroupTempID != "" {
			if gid, ok := groupIDMap[seedStep.BlockGroupTempID]; ok {
				blockGroupID = &gid
			}
		}

		// Set trigger type for start steps
		var triggerType *domain.StepTriggerType
		if seedStep.TriggerType != "" {
			tt := domain.StepTriggerType(seedStep.TriggerType)
			triggerType = &tt
		}

		// Set tool definition fields for Agent Group entry points
		var toolName, toolDescription *string
		if seedStep.ToolName != "" {
			toolName = &seedStep.ToolName
		}
		if seedStep.ToolDescription != "" {
			toolDescription = &seedStep.ToolDescription
		}

		step := &domain.Step{
			ID:                 stepID,
			TenantID:           tenantID,
			ProjectID:          projectID,
			Name:               seedStep.Name,
			Type:               domain.StepType(seedStep.Type),
			Config:             seedStep.Config,
			TriggerType:        triggerType,
			TriggerConfig:      seedStep.TriggerConfig,
			PositionX:          seedStep.PositionX,
			PositionY:          seedStep.PositionY,
			BlockDefinitionID:  blockDefID,
			BlockGroupID:       blockGroupID,
			GroupRole:          "body", // Default role for steps in block groups
			CredentialBindings: json.RawMessage(`{}`),
			ToolName:           toolName,
			ToolDescription:    toolDescription,
			ToolInputSchema:    seedStep.ToolInputSchema,
			CreatedAt:          now,
			UpdatedAt:          now,
		}

		if err := m.stepRepo.Create(ctx, step); err != nil {
			return nil, fmt.Errorf("failed to create step %s: %w", seedStep.Name, err)
		}
	}

	return stepIDMap, nil
}

// createEdges creates all edges for a project
func (m *ProjectMigrator) createEdges(ctx context.Context, seedProject *workflows.SystemWorkflowDefinition, tenantID uuid.UUID, projectID uuid.UUID, stepIDMap map[string]uuid.UUID) error {
	// Build group ID map from current project's block groups
	groupIDMap := make(map[string]uuid.UUID)
	if m.blockGroupRepo != nil {
		groups, err := m.blockGroupRepo.ListByProject(ctx, tenantID, projectID)
		if err == nil {
			for _, group := range groups {
				// Match by name to find temp_id (from seed)
				for _, seedGroup := range seedProject.BlockGroups {
					if seedGroup.Name == group.Name {
						groupIDMap[seedGroup.TempID] = group.ID
						break
					}
				}
			}
		}
	}

	return m.createEdgesWithGroupMap(ctx, seedProject, tenantID, projectID, stepIDMap, groupIDMap)
}

// createEdgesWithGroupMap creates all edges for a project with explicit group ID mapping
func (m *ProjectMigrator) createEdgesWithGroupMap(ctx context.Context, seedProject *workflows.SystemWorkflowDefinition, tenantID uuid.UUID, projectID uuid.UUID, stepIDMap map[string]uuid.UUID, groupIDMap map[string]uuid.UUID) error {
	now := time.Now().UTC()

	// Build step type map for port validation
	stepTypeMap := make(map[string]string) // tempID -> type
	for _, step := range seedProject.Steps {
		stepTypeMap[step.TempID] = step.Type
	}

	// Build group type map for port validation
	groupTypeMap := make(map[string]string) // tempID -> type
	for _, group := range seedProject.BlockGroups {
		groupTypeMap[group.TempID] = group.Type
	}

	for _, seedEdge := range seedProject.Edges {
		var sourceStepID, targetStepID *uuid.UUID
		var sourceGroupID, targetGroupID *uuid.UUID

		// Resolve source (step or group)
		if seedEdge.SourceTempID != "" {
			id, ok := stepIDMap[seedEdge.SourceTempID]
			if !ok {
				return fmt.Errorf("invalid source_temp_id: %s", seedEdge.SourceTempID)
			}
			sourceStepID = &id
		} else if seedEdge.SourceGroupTempID != "" {
			id, ok := groupIDMap[seedEdge.SourceGroupTempID]
			if !ok {
				return fmt.Errorf("invalid source_group_temp_id: %s", seedEdge.SourceGroupTempID)
			}
			sourceGroupID = &id
		}

		// Resolve target (step or group)
		if seedEdge.TargetTempID != "" {
			id, ok := stepIDMap[seedEdge.TargetTempID]
			if !ok {
				return fmt.Errorf("invalid target_temp_id: %s", seedEdge.TargetTempID)
			}
			targetStepID = &id
		} else if seedEdge.TargetGroupTempID != "" {
			id, ok := groupIDMap[seedEdge.TargetGroupTempID]
			if !ok {
				return fmt.Errorf("invalid target_group_temp_id: %s", seedEdge.TargetGroupTempID)
			}
			targetGroupID = &id
		}

		// Validate source port if block repo is available
		if m.blockRepo != nil && seedEdge.SourcePort != "" {
			var blockSlug string
			if seedEdge.SourceTempID != "" {
				blockSlug = stepTypeMap[seedEdge.SourceTempID]
			} else if seedEdge.SourceGroupTempID != "" {
				blockSlug = groupTypeMap[seedEdge.SourceGroupTempID]
			}
			if blockSlug != "" {
				if err := m.validateSourcePort(ctx, seedEdge.SourcePort, blockSlug); err != nil {
					return fmt.Errorf("edge source port validation failed for %s->%s: %w", seedEdge.SourceTempID+seedEdge.SourceGroupTempID, seedEdge.TargetTempID+seedEdge.TargetGroupTempID, err)
				}
			}
		}

		var condition *string
		if seedEdge.Condition != "" {
			condition = &seedEdge.Condition
		}

		edge := &domain.Edge{
			ID:                 uuid.New(),
			TenantID:           tenantID,
			ProjectID:          projectID,
			SourceStepID:       sourceStepID,
			TargetStepID:       targetStepID,
			SourceBlockGroupID: sourceGroupID,
			TargetBlockGroupID: targetGroupID,
			SourcePort:         seedEdge.SourcePort,
			Condition:          condition,
			CreatedAt:          now,
		}

		if err := m.edgeRepo.Create(ctx, edge); err != nil {
			return fmt.Errorf("failed to create edge: %w", err)
		}
	}

	return nil
}

// validateSourcePort validates that the source port exists in the block definition
func (m *ProjectMigrator) validateSourcePort(ctx context.Context, sourcePort, blockSlug string) error {
	blockDef, err := m.blockRepo.GetBySlug(ctx, nil, blockSlug)
	if err != nil {
		return err
	}

	// Check current block's output ports
	for _, port := range blockDef.OutputPorts {
		if port.Name == sourcePort {
			return nil
		}
	}

	// Check inherited ports from parent blocks
	allPorts := m.getInheritedOutputPorts(ctx, blockDef)
	for _, port := range allPorts {
		if port.Name == sourcePort {
			return nil
		}
	}

	return fmt.Errorf("source port '%s' not found in block '%s' output ports (available: %v)", sourcePort, blockSlug, getPortNames(allPorts))
}

// getInheritedOutputPorts recursively collects output ports from parent blocks
func (m *ProjectMigrator) getInheritedOutputPorts(ctx context.Context, blockDef *domain.BlockDefinition) []domain.OutputPort {
	var allPorts []domain.OutputPort

	// Start with current block's ports
	allPorts = append(allPorts, blockDef.OutputPorts...)

	// Recursively get parent block's ports
	if blockDef.ParentBlockID != nil {
		parentBlock, err := m.blockRepo.GetByID(ctx, *blockDef.ParentBlockID)
		if err == nil && parentBlock != nil {
			parentPorts := m.getInheritedOutputPorts(ctx, parentBlock)
			// Add parent ports that don't exist in current block
			for _, pp := range parentPorts {
				found := false
				for _, cp := range allPorts {
					if cp.Name == pp.Name {
						found = true
						break
					}
				}
				if !found {
					allPorts = append(allPorts, pp)
				}
			}
		}
	}

	return allPorts
}

// getPortNames extracts port names from output ports for error messages
func getPortNames(ports []domain.OutputPort) []string {
	names := make([]string, len(ports))
	for i, p := range ports {
		names[i] = p.Name
	}
	return names
}

// updateProject updates an existing system project
func (m *ProjectMigrator) updateProject(ctx context.Context, existing *domain.Project, seedProject *workflows.SystemWorkflowDefinition, tenantID uuid.UUID) (string, error) {
	now := time.Now().UTC()

	// Update project fields
	existing.Name = seedProject.Name
	existing.Description = seedProject.Description
	existing.Version = seedProject.Version
	existing.Variables = seedProject.InputSchema // Seed definition uses InputSchema field name
	existing.UpdatedAt = now

	if err := m.projectRepo.Update(ctx, existing); err != nil {
		return "", fmt.Errorf("failed to update project: %w", err)
	}

	// Delete existing steps, edges, and block groups, then recreate
	// This is simpler than trying to diff and update individual items
	existingSteps, err := m.stepRepo.ListByProject(ctx, tenantID, existing.ID)
	if err != nil {
		return "", fmt.Errorf("failed to list existing steps: %w", err)
	}

	existingEdges, err := m.edgeRepo.ListByProject(ctx, tenantID, existing.ID)
	if err != nil {
		return "", fmt.Errorf("failed to list existing edges: %w", err)
	}

	// Delete edges first (due to foreign key constraints)
	for _, edge := range existingEdges {
		if err := m.edgeRepo.Delete(ctx, tenantID, existing.ID, edge.ID); err != nil {
			return "", fmt.Errorf("failed to delete edge: %w", err)
		}
	}

	// Delete steps (before block groups due to foreign key)
	for _, step := range existingSteps {
		if err := m.stepRepo.Delete(ctx, tenantID, existing.ID, step.ID); err != nil {
			return "", fmt.Errorf("failed to delete step: %w", err)
		}
	}

	// Delete existing block groups
	if m.blockGroupRepo != nil {
		existingGroups, err := m.blockGroupRepo.ListByProject(ctx, tenantID, existing.ID)
		if err != nil {
			return "", fmt.Errorf("failed to list existing block groups: %w", err)
		}
		for _, group := range existingGroups {
			if err := m.blockGroupRepo.Delete(ctx, tenantID, existing.ID, group.ID); err != nil {
				return "", fmt.Errorf("failed to delete block group: %w", err)
			}
		}
	}

	// Recreate block groups, steps, and edges
	groupIDMap, err := m.createBlockGroups(ctx, seedProject, tenantID, existing.ID)
	if err != nil {
		return "", fmt.Errorf("failed to create block groups: %w", err)
	}

	stepIDMap, err := m.createSteps(ctx, seedProject, tenantID, existing.ID, groupIDMap)
	if err != nil {
		return "", fmt.Errorf("failed to create steps: %w", err)
	}

	if err := m.createEdgesWithGroupMap(ctx, seedProject, tenantID, existing.ID, stepIDMap, groupIDMap); err != nil {
		return "", fmt.Errorf("failed to create edges: %w", err)
	}

	return "updated", nil
}

// hasChanges compares existing project with seed definition
func (m *ProjectMigrator) hasChanges(existing *domain.Project, seed *workflows.SystemWorkflowDefinition) bool {
	// Compare version first
	if existing.Version != seed.Version {
		return true
	}

	// Compare key fields
	if existing.Name != seed.Name {
		return true
	}
	if existing.Description != seed.Description {
		return true
	}

	// Compare JSON fields (Variables vs InputSchema)
	if !jsonEqual(existing.Variables, seed.InputSchema) {
		return true
	}

	return false
}

// DryRun checks what would happen during migration without applying
func (m *ProjectMigrator) DryRun(ctx context.Context, registry *workflows.Registry, tenantID uuid.UUID) (*ProjectDryRunResult, error) {
	result := &ProjectDryRunResult{
		ToCreate:  make([]string, 0),
		ToUpdate:  make([]ProjectUpdateInfo, 0),
		Unchanged: make([]string, 0),
	}

	for _, seedProject := range registry.GetAll() {
		projectID, err := uuid.Parse(seedProject.ID)
		if err != nil {
			return nil, fmt.Errorf("invalid project ID for %s: %w", seedProject.SystemSlug, err)
		}

		existing, err := m.projectRepo.GetByID(ctx, tenantID, projectID)
		if err != nil {
			if errors.Is(err, domain.ErrProjectNotFound) {
				result.ToCreate = append(result.ToCreate, seedProject.SystemSlug)
				continue
			}
			return nil, fmt.Errorf("failed to get existing project %s: %w", seedProject.SystemSlug, err)
		}

		if m.hasChanges(existing, seedProject) {
			reason := m.describeChanges(existing, seedProject)
			result.ToUpdate = append(result.ToUpdate, ProjectUpdateInfo{
				SystemSlug: seedProject.SystemSlug,
				OldVersion: existing.Version,
				NewVersion: seedProject.Version,
				Reason:     reason,
			})
		} else {
			result.Unchanged = append(result.Unchanged, seedProject.SystemSlug)
		}
	}

	return result, nil
}

// describeChanges returns a human-readable description of what changed
func (m *ProjectMigrator) describeChanges(existing *domain.Project, seed *workflows.SystemWorkflowDefinition) string {
	changes := make([]string, 0)

	if existing.Version != seed.Version {
		changes = append(changes, "version")
	}
	if existing.Name != seed.Name {
		changes = append(changes, "name")
	}
	if existing.Description != seed.Description {
		changes = append(changes, "description")
	}
	if !jsonEqual(existing.Variables, seed.InputSchema) {
		changes = append(changes, "variables")
	}

	if len(changes) == 0 {
		return "steps/edges changed"
	}

	result := changes[0]
	for i := 1; i < len(changes); i++ {
		result += ", " + changes[i]
	}
	return result
}
