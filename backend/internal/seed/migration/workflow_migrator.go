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

// WorkflowMigrationResult tracks what happened during workflow migration
type WorkflowMigrationResult struct {
	Created   []string // SystemSlugs of newly created workflows
	Updated   []string // SystemSlugs of updated workflows
	Unchanged []string // SystemSlugs of unchanged workflows
	Errors    []error
}

// WorkflowDryRunResult shows what would happen during migration without applying
type WorkflowDryRunResult struct {
	ToCreate  []string             // SystemSlugs of workflows to create
	ToUpdate  []WorkflowUpdateInfo // Info about workflows to update
	Unchanged []string             // SystemSlugs of unchanged workflows
}

// WorkflowUpdateInfo provides details about a workflow update
type WorkflowUpdateInfo struct {
	SystemSlug string
	OldVersion int
	NewVersion int
	Reason     string
}

// WorkflowMigrator handles workflow definition migration
type WorkflowMigrator struct {
	workflowRepo   repository.WorkflowRepository
	stepRepo       repository.StepRepository
	edgeRepo       repository.EdgeRepository
	blockRepo      repository.BlockDefinitionRepository
	blockGroupRepo repository.BlockGroupRepository
}

// NewWorkflowMigrator creates a new workflow migrator
func NewWorkflowMigrator(
	workflowRepo repository.WorkflowRepository,
	stepRepo repository.StepRepository,
	edgeRepo repository.EdgeRepository,
) *WorkflowMigrator {
	return &WorkflowMigrator{
		workflowRepo: workflowRepo,
		stepRepo:     stepRepo,
		edgeRepo:     edgeRepo,
	}
}

// WithBlockRepo sets the block definition repository for resolving block slugs
func (m *WorkflowMigrator) WithBlockRepo(blockRepo repository.BlockDefinitionRepository) *WorkflowMigrator {
	m.blockRepo = blockRepo
	return m
}

// WithBlockGroupRepo sets the block group repository for creating block groups
func (m *WorkflowMigrator) WithBlockGroupRepo(blockGroupRepo repository.BlockGroupRepository) *WorkflowMigrator {
	m.blockGroupRepo = blockGroupRepo
	return m
}

// Migrate performs UPSERT for all workflows in the registry
func (m *WorkflowMigrator) Migrate(ctx context.Context, registry *workflows.Registry, tenantID uuid.UUID) (*WorkflowMigrationResult, error) {
	result := &WorkflowMigrationResult{
		Created:   make([]string, 0),
		Updated:   make([]string, 0),
		Unchanged: make([]string, 0),
		Errors:    make([]error, 0),
	}

	for _, seedWorkflow := range registry.GetAll() {
		action, err := m.upsertWorkflow(ctx, seedWorkflow, tenantID)
		if err != nil {
			result.Errors = append(result.Errors,
				fmt.Errorf("workflow %s: %w", seedWorkflow.SystemSlug, err))
			continue
		}

		switch action {
		case "created":
			result.Created = append(result.Created, seedWorkflow.SystemSlug)
		case "updated":
			result.Updated = append(result.Updated, seedWorkflow.SystemSlug)
		case "unchanged":
			result.Unchanged = append(result.Unchanged, seedWorkflow.SystemSlug)
		}
	}

	return result, nil
}

// upsertWorkflow creates or updates a single workflow with its steps and edges
func (m *WorkflowMigrator) upsertWorkflow(ctx context.Context, seedWorkflow *workflows.SystemWorkflowDefinition, tenantID uuid.UUID) (string, error) {
	// Parse the workflow ID from the seed
	workflowID, err := uuid.Parse(seedWorkflow.ID)
	if err != nil {
		return "", fmt.Errorf("invalid workflow ID: %w", err)
	}

	// Look up existing workflow by ID
	existing, err := m.workflowRepo.GetByID(ctx, tenantID, workflowID)
	if err != nil {
		if errors.Is(err, domain.ErrWorkflowNotFound) {
			// Workflow doesn't exist, create it
			return m.createWorkflow(ctx, seedWorkflow, tenantID, workflowID)
		}
		return "", fmt.Errorf("failed to get existing workflow: %w", err)
	}

	// Check if update is needed
	if m.hasChanges(existing, seedWorkflow) {
		// UPDATE existing workflow
		return m.updateWorkflow(ctx, existing, seedWorkflow, tenantID)
	}

	return "unchanged", nil
}

// createWorkflow creates a new system workflow with steps and edges
func (m *WorkflowMigrator) createWorkflow(ctx context.Context, seedWorkflow *workflows.SystemWorkflowDefinition, tenantID uuid.UUID, workflowID uuid.UUID) (string, error) {
	now := time.Now().UTC()

	// Derive input_schema from first executable step's block definition
	inputSchema := m.deriveInputSchemaFromSeed(ctx, seedWorkflow)
	if inputSchema == nil {
		// Fallback to seed's input_schema if derivation fails
		inputSchema = seedWorkflow.InputSchema
	}

	// Create workflow
	workflow := &domain.Workflow{
		ID:           workflowID,
		TenantID:     tenantID,
		Name:         seedWorkflow.Name,
		Description:  seedWorkflow.Description,
		Status:       domain.WorkflowStatusPublished,
		Version:      seedWorkflow.Version,
		InputSchema:  inputSchema,
		OutputSchema: seedWorkflow.OutputSchema,
		IsSystem:     true,
		SystemSlug:   &seedWorkflow.SystemSlug,
		PublishedAt:  &now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := m.workflowRepo.Create(ctx, workflow); err != nil {
		return "", fmt.Errorf("failed to create workflow: %w", err)
	}

	// Create block groups and build temp_id -> actual_id mapping
	groupIDMap, err := m.createBlockGroups(ctx, seedWorkflow, tenantID, workflowID)
	if err != nil {
		return "", fmt.Errorf("failed to create block groups: %w", err)
	}

	// Create steps and build temp_id -> actual_id mapping
	stepIDMap, err := m.createSteps(ctx, seedWorkflow, tenantID, workflowID, groupIDMap)
	if err != nil {
		return "", fmt.Errorf("failed to create steps: %w", err)
	}

	// Create edges using the step ID and group ID mappings
	if err := m.createEdgesWithGroupMap(ctx, seedWorkflow, tenantID, workflowID, stepIDMap, groupIDMap); err != nil {
		return "", fmt.Errorf("failed to create edges: %w", err)
	}

	return "created", nil
}

// createBlockGroups creates all block groups for a workflow and returns a temp_id -> group_id mapping
func (m *WorkflowMigrator) createBlockGroups(ctx context.Context, seedWorkflow *workflows.SystemWorkflowDefinition, tenantID uuid.UUID, workflowID uuid.UUID) (map[string]uuid.UUID, error) {
	groupIDMap := make(map[string]uuid.UUID)

	if len(seedWorkflow.BlockGroups) == 0 || m.blockGroupRepo == nil {
		return groupIDMap, nil
	}

	now := time.Now().UTC()

	// First pass: create all groups without parent references
	for _, seedGroup := range seedWorkflow.BlockGroups {
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
			WorkflowID:  workflowID,
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

// createSteps creates all steps for a workflow and returns a temp_id -> step_id mapping
func (m *WorkflowMigrator) createSteps(ctx context.Context, seedWorkflow *workflows.SystemWorkflowDefinition, tenantID uuid.UUID, workflowID uuid.UUID, groupIDMap map[string]uuid.UUID) (map[string]uuid.UUID, error) {
	stepIDMap := make(map[string]uuid.UUID)
	now := time.Now().UTC()

	for _, seedStep := range seedWorkflow.Steps {
		stepID := uuid.New()
		stepIDMap[seedStep.TempID] = stepID

		var blockDefID *uuid.UUID

		// Resolve block ID from slug (preferred) or use direct ID (deprecated)
		if seedStep.BlockSlug != "" && m.blockRepo != nil {
			// Look up block by slug (system blocks have tenant_id = NULL)
			block, err := m.blockRepo.GetBySlug(ctx, nil, seedStep.BlockSlug)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve block slug %s: %w", seedStep.BlockSlug, err)
			}
			if block != nil {
				blockDefID = &block.ID
			}
		} else if seedStep.BlockDefID != nil {
			// Fallback to direct ID (deprecated)
			parsed, err := uuid.Parse(*seedStep.BlockDefID)
			if err == nil {
				blockDefID = &parsed
			}
		}

		// Resolve block group ID from temp_id
		var blockGroupID *uuid.UUID
		if seedStep.BlockGroupTempID != "" {
			if gid, ok := groupIDMap[seedStep.BlockGroupTempID]; ok {
				blockGroupID = &gid
			}
		}

		step := &domain.Step{
			ID:                 stepID,
			TenantID:           tenantID,
			WorkflowID:         workflowID,
			Name:               seedStep.Name,
			Type:               domain.StepType(seedStep.Type),
			Config:             seedStep.Config,
			PositionX:          seedStep.PositionX,
			PositionY:          seedStep.PositionY,
			BlockDefinitionID:  blockDefID,
			BlockGroupID:       blockGroupID,
			GroupRole:          "body", // Default role for steps in block groups
			CredentialBindings: json.RawMessage(`{}`),
			CreatedAt:          now,
			UpdatedAt:          now,
		}

		if err := m.stepRepo.Create(ctx, step); err != nil {
			return nil, fmt.Errorf("failed to create step %s: %w", seedStep.Name, err)
		}
	}

	return stepIDMap, nil
}

// createEdges creates all edges for a workflow
func (m *WorkflowMigrator) createEdges(ctx context.Context, seedWorkflow *workflows.SystemWorkflowDefinition, tenantID uuid.UUID, workflowID uuid.UUID, stepIDMap map[string]uuid.UUID) error {
	// Build group ID map from current workflow's block groups
	groupIDMap := make(map[string]uuid.UUID)
	if m.blockGroupRepo != nil {
		groups, err := m.blockGroupRepo.ListByWorkflow(ctx, tenantID, workflowID)
		if err == nil {
			for _, group := range groups {
				// Match by name to find temp_id (from seed)
				for _, seedGroup := range seedWorkflow.BlockGroups {
					if seedGroup.Name == group.Name {
						groupIDMap[seedGroup.TempID] = group.ID
						break
					}
				}
			}
		}
	}

	return m.createEdgesWithGroupMap(ctx, seedWorkflow, tenantID, workflowID, stepIDMap, groupIDMap)
}

// createEdgesWithGroupMap creates all edges for a workflow with explicit group ID mapping
func (m *WorkflowMigrator) createEdgesWithGroupMap(ctx context.Context, seedWorkflow *workflows.SystemWorkflowDefinition, tenantID uuid.UUID, workflowID uuid.UUID, stepIDMap map[string]uuid.UUID, groupIDMap map[string]uuid.UUID) error {
	now := time.Now().UTC()

	// Build step type map for port validation
	stepTypeMap := make(map[string]string) // tempID -> type
	for _, step := range seedWorkflow.Steps {
		stepTypeMap[step.TempID] = step.Type
	}

	// Build group type map for port validation
	groupTypeMap := make(map[string]string) // tempID -> type
	for _, group := range seedWorkflow.BlockGroups {
		groupTypeMap[group.TempID] = group.Type
	}

	for _, seedEdge := range seedWorkflow.Edges {
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

		// Validate target port if block repo is available
		if m.blockRepo != nil && seedEdge.TargetPort != "" && seedEdge.TargetPort != "group-input" {
			var blockSlug string
			if seedEdge.TargetTempID != "" {
				blockSlug = stepTypeMap[seedEdge.TargetTempID]
			} else if seedEdge.TargetGroupTempID != "" {
				blockSlug = groupTypeMap[seedEdge.TargetGroupTempID]
			}
			if blockSlug != "" {
				if err := m.validateTargetPort(ctx, seedEdge.TargetPort, blockSlug); err != nil {
					return fmt.Errorf("edge target port validation failed for %s->%s: %w", seedEdge.SourceTempID+seedEdge.SourceGroupTempID, seedEdge.TargetTempID+seedEdge.TargetGroupTempID, err)
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
			WorkflowID:         workflowID,
			SourceStepID:       sourceStepID,
			TargetStepID:       targetStepID,
			SourceBlockGroupID: sourceGroupID,
			TargetBlockGroupID: targetGroupID,
			SourcePort:         seedEdge.SourcePort,
			TargetPort:         seedEdge.TargetPort,
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
func (m *WorkflowMigrator) validateSourcePort(ctx context.Context, sourcePort, blockSlug string) error {
	blockDef, err := m.blockRepo.GetBySlug(ctx, nil, blockSlug)
	if err != nil {
		// If block definition not found, skip validation (legacy blocks)
		if errors.Is(err, domain.ErrBlockDefinitionNotFound) {
			return nil
		}
		return err
	}

	for _, port := range blockDef.OutputPorts {
		if port.Name == sourcePort {
			return nil
		}
	}

	return fmt.Errorf("source port '%s' not found in block '%s' output ports (available: %v)", sourcePort, blockSlug, getPortNames(blockDef.OutputPorts))
}

// validateTargetPort validates that the target port exists in the block definition
func (m *WorkflowMigrator) validateTargetPort(ctx context.Context, targetPort, blockSlug string) error {
	blockDef, err := m.blockRepo.GetBySlug(ctx, nil, blockSlug)
	if err != nil {
		// If block definition not found, skip validation (legacy blocks)
		if errors.Is(err, domain.ErrBlockDefinitionNotFound) {
			return nil
		}
		return err
	}

	for _, port := range blockDef.InputPorts {
		if port.Name == targetPort {
			return nil
		}
	}

	return fmt.Errorf("target port '%s' not found in block '%s' input ports (available: %v)", targetPort, blockSlug, getInputPortNames(blockDef.InputPorts))
}

// getPortNames extracts port names from output ports for error messages
func getPortNames(ports []domain.OutputPort) []string {
	names := make([]string, len(ports))
	for i, p := range ports {
		names[i] = p.Name
	}
	return names
}

// getInputPortNames extracts port names from input ports for error messages
func getInputPortNames(ports []domain.InputPort) []string {
	names := make([]string, len(ports))
	for i, p := range ports {
		names[i] = p.Name
	}
	return names
}

// updateWorkflow updates an existing system workflow
func (m *WorkflowMigrator) updateWorkflow(ctx context.Context, existing *domain.Workflow, seedWorkflow *workflows.SystemWorkflowDefinition, tenantID uuid.UUID) (string, error) {
	now := time.Now().UTC()

	// Derive input_schema from first executable step's block definition
	inputSchema := m.deriveInputSchemaFromSeed(ctx, seedWorkflow)
	if inputSchema == nil {
		// Fallback to seed's input_schema if derivation fails
		inputSchema = seedWorkflow.InputSchema
	}

	// Update workflow fields
	existing.Name = seedWorkflow.Name
	existing.Description = seedWorkflow.Description
	existing.Version = seedWorkflow.Version
	existing.InputSchema = inputSchema
	existing.OutputSchema = seedWorkflow.OutputSchema
	existing.UpdatedAt = now

	if err := m.workflowRepo.Update(ctx, existing); err != nil {
		return "", fmt.Errorf("failed to update workflow: %w", err)
	}

	// Delete existing steps, edges, and block groups, then recreate
	// This is simpler than trying to diff and update individual items
	existingSteps, err := m.stepRepo.ListByWorkflow(ctx, tenantID, existing.ID)
	if err != nil {
		return "", fmt.Errorf("failed to list existing steps: %w", err)
	}

	existingEdges, err := m.edgeRepo.ListByWorkflow(ctx, tenantID, existing.ID)
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
		existingGroups, err := m.blockGroupRepo.ListByWorkflow(ctx, tenantID, existing.ID)
		if err != nil {
			return "", fmt.Errorf("failed to list existing block groups: %w", err)
		}
		for _, group := range existingGroups {
			if err := m.blockGroupRepo.Delete(ctx, tenantID, group.ID); err != nil {
				return "", fmt.Errorf("failed to delete block group: %w", err)
			}
		}
	}

	// Recreate block groups, steps, and edges
	groupIDMap, err := m.createBlockGroups(ctx, seedWorkflow, tenantID, existing.ID)
	if err != nil {
		return "", fmt.Errorf("failed to create block groups: %w", err)
	}

	stepIDMap, err := m.createSteps(ctx, seedWorkflow, tenantID, existing.ID, groupIDMap)
	if err != nil {
		return "", fmt.Errorf("failed to create steps: %w", err)
	}

	if err := m.createEdgesWithGroupMap(ctx, seedWorkflow, tenantID, existing.ID, stepIDMap, groupIDMap); err != nil {
		return "", fmt.Errorf("failed to create edges: %w", err)
	}

	return "updated", nil
}

// hasChanges compares existing workflow with seed definition
func (m *WorkflowMigrator) hasChanges(existing *domain.Workflow, seed *workflows.SystemWorkflowDefinition) bool {
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

	// Compare JSON fields
	if !jsonEqual(existing.InputSchema, seed.InputSchema) {
		return true
	}
	if !jsonEqual(existing.OutputSchema, seed.OutputSchema) {
		return true
	}

	return false
}

// DryRun checks what would happen during migration without applying
func (m *WorkflowMigrator) DryRun(ctx context.Context, registry *workflows.Registry, tenantID uuid.UUID) (*WorkflowDryRunResult, error) {
	result := &WorkflowDryRunResult{
		ToCreate:  make([]string, 0),
		ToUpdate:  make([]WorkflowUpdateInfo, 0),
		Unchanged: make([]string, 0),
	}

	for _, seedWorkflow := range registry.GetAll() {
		workflowID, err := uuid.Parse(seedWorkflow.ID)
		if err != nil {
			return nil, fmt.Errorf("invalid workflow ID for %s: %w", seedWorkflow.SystemSlug, err)
		}

		existing, err := m.workflowRepo.GetByID(ctx, tenantID, workflowID)
		if err != nil {
			if errors.Is(err, domain.ErrWorkflowNotFound) {
				result.ToCreate = append(result.ToCreate, seedWorkflow.SystemSlug)
				continue
			}
			return nil, fmt.Errorf("failed to get existing workflow %s: %w", seedWorkflow.SystemSlug, err)
		}

		if m.hasChanges(existing, seedWorkflow) {
			reason := m.describeChanges(existing, seedWorkflow)
			result.ToUpdate = append(result.ToUpdate, WorkflowUpdateInfo{
				SystemSlug: seedWorkflow.SystemSlug,
				OldVersion: existing.Version,
				NewVersion: seedWorkflow.Version,
				Reason:     reason,
			})
		} else {
			result.Unchanged = append(result.Unchanged, seedWorkflow.SystemSlug)
		}
	}

	return result, nil
}

// describeChanges returns a human-readable description of what changed
func (m *WorkflowMigrator) describeChanges(existing *domain.Workflow, seed *workflows.SystemWorkflowDefinition) string {
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
	if !jsonEqual(existing.InputSchema, seed.InputSchema) {
		changes = append(changes, "input_schema")
	}
	if !jsonEqual(existing.OutputSchema, seed.OutputSchema) {
		changes = append(changes, "output_schema")
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

// deriveInputSchemaFromSeed derives input_schema from the first executable step's block definition
// This ensures workflow.InputSchema always reflects the actual first step's requirements
func (m *WorkflowMigrator) deriveInputSchemaFromSeed(ctx context.Context, seedWorkflow *workflows.SystemWorkflowDefinition) json.RawMessage {
	if m.blockRepo == nil {
		return nil
	}

	// 1. Find Start step
	var startStepTempID string
	for _, step := range seedWorkflow.Steps {
		if step.Type == "start" {
			startStepTempID = step.TempID
			break
		}
	}
	if startStepTempID == "" {
		return nil
	}

	// 2. Find first step after Start (via edge)
	var firstStepTempID string
	for _, edge := range seedWorkflow.Edges {
		if edge.SourceTempID == startStepTempID {
			firstStepTempID = edge.TargetTempID
			break
		}
	}
	if firstStepTempID == "" {
		return nil
	}

	// 3. Get block slug from step
	var blockSlug string
	for _, step := range seedWorkflow.Steps {
		if step.TempID == firstStepTempID {
			blockSlug = step.BlockSlug
			if blockSlug == "" {
				blockSlug = step.Type
			}
			break
		}
	}
	if blockSlug == "" || blockSlug == "start" || blockSlug == "end" {
		return nil
	}

	// 4. Get block definition from repository
	block, err := m.blockRepo.GetBySlug(ctx, nil, blockSlug)
	if err != nil || block == nil {
		return nil
	}

	return block.InputSchema
}
