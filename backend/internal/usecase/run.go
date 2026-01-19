package usecase

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/engine"
	"github.com/souta/ai-orchestration/internal/repository"
)

// RunUsecase handles run business logic
type RunUsecase struct {
	projectRepo repository.ProjectRepository
	runRepo     repository.RunRepository
	versionRepo repository.ProjectVersionRepository
	stepRepo    repository.StepRepository
	edgeRepo    repository.EdgeRepository
	stepRunRepo repository.StepRunRepository
	queue       *engine.Queue
}

// NewRunUsecase creates a new RunUsecase
func NewRunUsecase(
	projectRepo repository.ProjectRepository,
	runRepo repository.RunRepository,
	versionRepo repository.ProjectVersionRepository,
	stepRepo repository.StepRepository,
	edgeRepo repository.EdgeRepository,
	stepRunRepo repository.StepRunRepository,
	redisClient *redis.Client,
) *RunUsecase {
	return &RunUsecase{
		projectRepo: projectRepo,
		runRepo:     runRepo,
		versionRepo: versionRepo,
		stepRepo:    stepRepo,
		edgeRepo:    edgeRepo,
		stepRunRepo: stepRunRepo,
		queue:       engine.NewQueue(redisClient),
	}
}

// CreateRunInput represents input for creating a run
type CreateRunInput struct {
	TenantID    uuid.UUID
	ProjectID   uuid.UUID
	Version     int // 0 means latest version
	Input       json.RawMessage
	TriggeredBy domain.TriggerType // e.g., TriggerTypeManual, TriggerTypeTest
	UserID      *uuid.UUID
	StartStepID *uuid.UUID // Required: which Start block to execute from
}

// Create creates and enqueues a new run
func (u *RunUsecase) Create(ctx context.Context, input CreateRunInput) (*domain.Run, error) {
	// Validate start_step_id is required
	if input.StartStepID == nil {
		return nil, domain.NewValidationError("start_step_id", "start_step_id is required")
	}

	// Get project
	project, err := u.projectRepo.GetByID(ctx, input.TenantID, input.ProjectID)
	if err != nil {
		return nil, err
	}

	// Determine which version to use
	// 0 means use latest (current project version)
	version := input.Version
	if version == 0 {
		version = project.Version
	}

	// Validate that the requested version exists (if specific version requested)
	if input.Version > 0 && u.versionRepo != nil {
		_, err := u.versionRepo.GetByProjectAndVersion(ctx, project.ID, version)
		if err != nil {
			return nil, err
		}
	}

	// Validate input against Start step's input_schema
	if err := u.validateProjectInput(ctx, input.TenantID, project.ID, input.Input); err != nil {
		return nil, err
	}

	// Create run
	run := domain.NewRun(
		input.TenantID,
		project.ID,
		version,
		input.Input,
		input.TriggeredBy,
	)
	run.TriggeredByUser = input.UserID
	run.StartStepID = input.StartStepID

	if err := u.runRepo.Create(ctx, run); err != nil {
		return nil, err
	}

	// Enqueue job
	job := &engine.Job{
		TenantID:       input.TenantID,
		ProjectID:      project.ID,
		ProjectVersion: version,
		RunID:          run.ID,
		Input:          input.Input,
		TargetStepID:   input.StartStepID, // StartStepID is used as TargetStepID for execution
	}
	if err := u.queue.Enqueue(ctx, job); err != nil {
		return nil, err
	}

	return run, nil
}

// GetByID retrieves a run by ID
func (u *RunUsecase) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Run, error) {
	return u.runRepo.GetByID(ctx, tenantID, id)
}

// GetWithDetails retrieves a run with step runs
func (u *RunUsecase) GetWithDetails(ctx context.Context, tenantID, id uuid.UUID) (*domain.Run, error) {
	return u.runRepo.GetWithStepRuns(ctx, tenantID, id)
}

// RunWithDefinitionOutput represents a run with its project definition
type RunWithDefinitionOutput struct {
	Run               *domain.Run               `json:"run"`
	ProjectDefinition *domain.ProjectDefinition `json:"project_definition,omitempty"`
}

// GetWithDetailsAndDefinition retrieves a run with step runs and project definition
func (u *RunUsecase) GetWithDetailsAndDefinition(ctx context.Context, tenantID, id uuid.UUID) (*RunWithDefinitionOutput, error) {
	run, err := u.runRepo.GetWithStepRuns(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	output := &RunWithDefinitionOutput{
		Run: run,
	}

	// Try to get the project definition from the version snapshot
	if u.versionRepo != nil {
		version, err := u.versionRepo.GetByProjectAndVersion(ctx, run.ProjectID, run.ProjectVersion)
		if err == nil && version != nil {
			var definition domain.ProjectDefinition
			if err := json.Unmarshal(version.Definition, &definition); err == nil {
				output.ProjectDefinition = &definition
				return output, nil
			}
		}
	}

	// Fallback: If version snapshot not found, fetch current project definition
	// This handles runs created before version snapshots were implemented
	project, err := u.projectRepo.GetWithStepsAndEdges(ctx, tenantID, run.ProjectID)
	if err == nil && project != nil {
		output.ProjectDefinition = &domain.ProjectDefinition{
			Name:        project.Name,
			Description: project.Description,
			Variables:   project.Variables,
			Steps:       project.Steps,
			Edges:       project.Edges,
			BlockGroups: project.BlockGroups,
		}
	}

	return output, nil
}

// ListRunsInput represents input for listing runs
type ListRunsInput struct {
	TenantID    uuid.UUID
	ProjectID   uuid.UUID
	Status      *domain.RunStatus
	TriggeredBy *domain.TriggerType // Optional filter by trigger type
	Page        int
	Limit       int
}

// ListRunsOutput represents output for listing runs
type ListRunsOutput struct {
	Runs  []*domain.Run
	Total int
	Page  int
	Limit int
}

// List lists runs for a project
func (u *RunUsecase) List(ctx context.Context, input ListRunsInput) (*ListRunsOutput, error) {
	input.Page, input.Limit = NormalizePagination(input.Page, input.Limit)

	filter := repository.RunFilter{
		Status:      input.Status,
		TriggeredBy: input.TriggeredBy,
		Page:        input.Page,
		Limit:       input.Limit,
	}

	runs, total, err := u.runRepo.ListByProject(ctx, input.TenantID, input.ProjectID, filter)
	if err != nil {
		return nil, err
	}

	return &ListRunsOutput{
		Runs:  runs,
		Total: total,
		Page:  input.Page,
		Limit: input.Limit,
	}, nil
}

// Cancel cancels a running project
func (u *RunUsecase) Cancel(ctx context.Context, tenantID, id uuid.UUID) (*domain.Run, error) {
	run, err := u.runRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if run.Status != domain.RunStatusPending && run.Status != domain.RunStatusRunning {
		return nil, domain.ErrRunNotCancellable
	}

	run.Cancel()

	if err := u.runRepo.Update(ctx, run); err != nil {
		return nil, err
	}

	return run, nil
}

// ExecuteSingleStepInput represents input for executing a single step
type ExecuteSingleStepInput struct {
	TenantID uuid.UUID
	RunID    uuid.UUID
	StepID   uuid.UUID
	Input    json.RawMessage // Optional: custom input (nil means use previous input)
}

// ExecuteSingleStep executes only one step from an existing run
func (u *RunUsecase) ExecuteSingleStep(ctx context.Context, input ExecuteSingleStepInput) (*domain.StepRun, error) {
	// 1. Get run and validate status (only completed/failed runs can be re-executed)
	run, err := u.runRepo.GetByID(ctx, input.TenantID, input.RunID)
	if err != nil {
		return nil, err
	}
	if run.Status != domain.RunStatusCompleted && run.Status != domain.RunStatusFailed {
		return nil, domain.ErrRunNotResumable
	}

	// 2. Get project definition from version snapshot
	version, err := u.versionRepo.GetByProjectAndVersion(ctx, run.ProjectID, run.ProjectVersion)
	if err != nil {
		return nil, err
	}

	var definition domain.ProjectDefinition
	if err := json.Unmarshal(version.Definition, &definition); err != nil {
		return nil, err
	}

	// 3. Find the step in the definition
	var targetStep *domain.Step
	for i := range definition.Steps {
		if definition.Steps[i].ID == input.StepID {
			targetStep = &definition.Steps[i]
			break
		}
	}

	// 4. If step not found in version snapshot, try current project (for steps not in flow)
	if targetStep == nil {
		currentSteps, err := u.stepRepo.ListByProject(ctx, input.TenantID, run.ProjectID)
		if err != nil {
			return nil, err
		}
		for _, step := range currentSteps {
			if step.ID == input.StepID {
				targetStep = step
				// Add the step to definition for worker execution
				definition.Steps = append(definition.Steps, *step)
				break
			}
		}
	}

	if targetStep == nil {
		return nil, domain.ErrStepNotFound
	}

	// 4. Determine input (custom or from previous StepRun)
	stepInput := input.Input
	if stepInput == nil {
		lastRun, err := u.stepRunRepo.GetLatestByStep(ctx, input.TenantID, input.RunID, input.StepID)
		if err != nil {
			return nil, err
		}
		stepInput = lastRun.Input
	}

	// 5. Get max attempt number for the entire run and increment
	maxAttempt, err := u.stepRunRepo.GetMaxAttemptForRun(ctx, input.TenantID, input.RunID)
	if err != nil {
		return nil, err
	}
	newAttempt := maxAttempt + 1

	// 6. Collect previous step outputs for injection
	completedRuns, err := u.stepRunRepo.ListCompletedByRun(ctx, input.TenantID, input.RunID)
	if err != nil {
		return nil, err
	}
	injectedOutputs := make(map[string]json.RawMessage)
	for _, sr := range completedRuns {
		injectedOutputs[sr.StepID.String()] = sr.Output
	}

	// 7. Enqueue job
	job := &engine.Job{
		TenantID:        input.TenantID,
		ProjectID:       run.ProjectID,
		ProjectVersion:  run.ProjectVersion,
		RunID:           input.RunID,
		ExecutionMode:   engine.ExecutionModeSingleStep,
		TargetStepID:    &input.StepID,
		StepInput:       stepInput,
		InjectedOutputs: injectedOutputs,
	}
	if err := u.queue.Enqueue(ctx, job); err != nil {
		return nil, err
	}

	// 8. Return new StepRun (note: actual execution happens async in worker)
	// SequenceNumber will be assigned by the executor during actual execution
	return domain.NewStepRunWithAttempt(input.TenantID, input.RunID, input.StepID, targetStep.Name, newAttempt, 0), nil
}

// ResumeFromStepInput represents input for resuming execution from a step
type ResumeFromStepInput struct {
	TenantID      uuid.UUID
	RunID         uuid.UUID
	FromStepID    uuid.UUID
	InputOverride json.RawMessage // Optional: override input for the starting step
}

// ResumeFromStepOutput represents output for resuming execution
type ResumeFromStepOutput struct {
	RunID          uuid.UUID   `json:"run_id"`
	FromStepID     uuid.UUID   `json:"from_step_id"`
	StepsToExecute []uuid.UUID `json:"steps_to_execute"`
}

// ResumeFromStep resumes execution from a specific step through all downstream steps
func (u *RunUsecase) ResumeFromStep(ctx context.Context, input ResumeFromStepInput) (*ResumeFromStepOutput, error) {
	// 1. Get run and validate status
	run, err := u.runRepo.GetByID(ctx, input.TenantID, input.RunID)
	if err != nil {
		return nil, err
	}
	if run.Status != domain.RunStatusCompleted && run.Status != domain.RunStatusFailed {
		return nil, domain.ErrRunNotResumable
	}

	// 2. Get project definition from version snapshot
	version, err := u.versionRepo.GetByProjectAndVersion(ctx, run.ProjectID, run.ProjectVersion)
	if err != nil {
		return nil, err
	}

	var definition domain.ProjectDefinition
	if err := json.Unmarshal(version.Definition, &definition); err != nil {
		return nil, err
	}

	// 3. Find the starting step in the definition
	var found bool
	for i := range definition.Steps {
		if definition.Steps[i].ID == input.FromStepID {
			found = true
			break
		}
	}
	if !found {
		return nil, domain.ErrStepNotFound
	}

	// 4. Collect downstream steps (steps reachable from FromStepID)
	stepsToExecute := collectDownstreamSteps(&definition, input.FromStepID)

	// 5. Collect previous step outputs for injection (steps NOT in stepsToExecute)
	completedRuns, err := u.stepRunRepo.ListCompletedByRun(ctx, input.TenantID, input.RunID)
	if err != nil {
		return nil, err
	}

	// Create a set of steps to execute for quick lookup
	executeSet := make(map[uuid.UUID]bool)
	for _, stepID := range stepsToExecute {
		executeSet[stepID] = true
	}

	injectedOutputs := make(map[string]json.RawMessage)
	for _, sr := range completedRuns {
		// Only inject outputs from steps that won't be re-executed
		if !executeSet[sr.StepID] {
			injectedOutputs[sr.StepID.String()] = sr.Output
		}
	}

	// 6. Determine input for the starting step
	stepInput := input.InputOverride
	if stepInput == nil {
		lastRun, err := u.stepRunRepo.GetLatestByStep(ctx, input.TenantID, input.RunID, input.FromStepID)
		if err == nil && lastRun != nil {
			stepInput = lastRun.Input
		}
	}

	// 7. Enqueue job
	job := &engine.Job{
		TenantID:        input.TenantID,
		ProjectID:       run.ProjectID,
		ProjectVersion:  run.ProjectVersion,
		RunID:           input.RunID,
		ExecutionMode:   engine.ExecutionModeResume,
		TargetStepID:    &input.FromStepID,
		StepInput:       stepInput,
		InjectedOutputs: injectedOutputs,
	}
	if err := u.queue.Enqueue(ctx, job); err != nil {
		return nil, err
	}

	return &ResumeFromStepOutput{
		RunID:          input.RunID,
		FromStepID:     input.FromStepID,
		StepsToExecute: stepsToExecute,
	}, nil
}

// GetStepHistory returns all StepRuns for a specific step in a run
func (u *RunUsecase) GetStepHistory(ctx context.Context, tenantID, runID, stepID uuid.UUID) ([]*domain.StepRun, error) {
	// Validate run exists and belongs to tenant
	_, err := u.runRepo.GetByID(ctx, tenantID, runID)
	if err != nil {
		return nil, err
	}

	return u.stepRunRepo.ListByStep(ctx, tenantID, runID, stepID)
}

// ExecuteSystemProjectInput represents input for executing a system project
type ExecuteSystemProjectInput struct {
	TenantID        uuid.UUID              // Tenant context for the run
	SystemSlug      string                 // System project slug (e.g., "copilot")
	EntryPoint      string                 // Entry point identifier (e.g., "generate", "suggest")
	Input           json.RawMessage        // Project input
	TriggerSource   string                 // Internal caller identifier (e.g., "copilot")
	TriggerMetadata map[string]interface{} // Additional metadata (feature, user_id, session_id, etc.)
	UserID          *uuid.UUID             // User who triggered the execution
}

// ExecuteSystemProjectOutput represents output for system project execution
type ExecuteSystemProjectOutput struct {
	RunID     uuid.UUID `json:"run_id"`
	ProjectID uuid.UUID `json:"project_id"`
	Version   int       `json:"version"`
}

// ExecuteSystemProject executes a system project by its slug and entry point
// This is used for internal system calls (e.g., Copilot meta-project)
// Returns immediately after creating the run - execution happens asynchronously
func (u *RunUsecase) ExecuteSystemProject(ctx context.Context, input ExecuteSystemProjectInput) (*ExecuteSystemProjectOutput, error) {
	// 1. Look up system project by slug
	project, err := u.projectRepo.GetSystemBySlug(ctx, input.SystemSlug)
	if err != nil {
		return nil, err
	}

	// 2. Validate project is published
	if project.Status != domain.ProjectStatusPublished || project.Version < 1 {
		return nil, domain.ErrProjectNotPublished
	}

	// 3. Find the Start block by entry_point
	// For system projects, use the project's tenant ID to find steps
	var startStepID *uuid.UUID
	if input.EntryPoint != "" {
		steps, err := u.stepRepo.ListByProject(ctx, project.TenantID, project.ID)
		if err != nil {
			return nil, err
		}
		startStepID, err = findStartStepByEntryPoint(steps, input.EntryPoint)
		if err != nil {
			return nil, err
		}
	}

	// 4. Create run with internal trigger type
	run := domain.NewRun(
		input.TenantID,
		project.ID,
		project.Version,
		input.Input,
		domain.TriggerTypeInternal,
	)
	run.TriggeredByUser = input.UserID
	run.StartStepID = startStepID

	// 5. Set internal trigger metadata
	if err := run.SetInternalTrigger(input.TriggerSource, input.TriggerMetadata); err != nil {
		return nil, err
	}

	// 6. Save run to database
	if err := u.runRepo.Create(ctx, run); err != nil {
		return nil, err
	}

	// 7. Enqueue job for async execution
	// For system projects, pass the project's tenant_id so the worker can fetch the project
	job := &engine.Job{
		TenantID:        input.TenantID,
		ProjectID:       project.ID,
		ProjectVersion:  project.Version,
		RunID:           run.ID,
		Input:           input.Input,
		TargetStepID:    startStepID,
		ProjectTenantID: &project.TenantID, // System project may belong to different tenant
	}
	if err := u.queue.Enqueue(ctx, job); err != nil {
		return nil, err
	}

	return &ExecuteSystemProjectOutput{
		RunID:     run.ID,
		ProjectID: project.ID,
		Version:   project.Version,
	}, nil
}

// findStartStepByEntryPoint finds a Start block by its entry_point in trigger_config
func findStartStepByEntryPoint(steps []*domain.Step, entryPoint string) (*uuid.UUID, error) {
	for _, step := range steps {
		if step.Type != domain.StepTypeStart {
			continue
		}
		// Parse trigger_config to find entry_point
		if step.TriggerConfig != nil {
			var config map[string]interface{}
			if err := json.Unmarshal(step.TriggerConfig, &config); err != nil {
				continue
			}
			if ep, ok := config["entry_point"].(string); ok && ep == entryPoint {
				return &step.ID, nil
			}
		}
	}
	return nil, domain.NewValidationError("entry_point", "start step with entry_point '"+entryPoint+"' not found")
}

// TestStepInlineInput represents input for inline step testing
type TestStepInlineInput struct {
	TenantID  uuid.UUID
	ProjectID uuid.UUID
	StepID    uuid.UUID
	Input     json.RawMessage // Custom input for testing
	UserID    *uuid.UUID
}

// TestStepInlineOutput represents output for inline step testing
type TestStepInlineOutput struct {
	RunID    uuid.UUID `json:"run_id"`
	StepID   uuid.UUID `json:"step_id"`
	StepName string    `json:"step_name"`
	IsQueued bool      `json:"is_queued"`
}

// TestStepInline creates a test run and executes only a single step
// This allows testing a step without requiring an existing run
func (u *RunUsecase) TestStepInline(ctx context.Context, input TestStepInlineInput) (*TestStepInlineOutput, error) {
	// 1. Get the current project (draft state)
	project, err := u.projectRepo.GetByID(ctx, input.TenantID, input.ProjectID)
	if err != nil {
		return nil, err
	}

	// 2. Get current steps from the project
	steps, err := u.stepRepo.ListByProject(ctx, input.TenantID, input.ProjectID)
	if err != nil {
		return nil, err
	}

	// 3. Find the target step
	var targetStep *domain.Step
	for _, step := range steps {
		if step.ID == input.StepID {
			targetStep = step
			break
		}
	}
	if targetStep == nil {
		return nil, domain.ErrStepNotFound
	}

	// 4. Create a test run with version 0
	// The worker will automatically fall back to current project definition
	// when version 0 is not found (see worker/main.go processJob function)
	run := domain.NewRun(
		input.TenantID,
		project.ID,
		0, // Version 0 indicates inline test with current draft
		input.Input,
		domain.TriggerTypeTest,
	)
	run.TriggeredByUser = input.UserID

	if err := u.runRepo.Create(ctx, run); err != nil {
		return nil, err
	}

	// 5. Enqueue the job for single step execution
	// The worker will:
	// - Try to find version 0 (won't exist)
	// - Fall back to current project definition
	// - Execute only the target step
	job := &engine.Job{
		TenantID:        input.TenantID,
		ProjectID:       project.ID,
		ProjectVersion:  0, // Worker will fall back to current project
		RunID:           run.ID,
		ExecutionMode:   engine.ExecutionModeSingleStep,
		TargetStepID:    &input.StepID,
		StepInput:       input.Input,
		InjectedOutputs: make(map[string]json.RawMessage), // No previous outputs for inline test
	}
	if err := u.queue.Enqueue(ctx, job); err != nil {
		return nil, err
	}

	return &TestStepInlineOutput{
		RunID:    run.ID,
		StepID:   input.StepID,
		StepName: targetStep.Name,
		IsQueued: true,
	}, nil
}

// collectDownstreamSteps collects all steps reachable from the starting step (BFS)
func collectDownstreamSteps(def *domain.ProjectDefinition, startStepID uuid.UUID) []uuid.UUID {
	// Build adjacency list (only for step-to-step edges)
	outEdges := make(map[uuid.UUID][]uuid.UUID)
	for _, edge := range def.Edges {
		if edge.SourceStepID != nil && edge.TargetStepID != nil {
			outEdges[*edge.SourceStepID] = append(outEdges[*edge.SourceStepID], *edge.TargetStepID)
		}
	}

	// BFS to collect all downstream steps
	result := []uuid.UUID{startStepID}
	visited := make(map[uuid.UUID]bool)
	visited[startStepID] = true

	queue := []uuid.UUID{startStepID}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, next := range outEdges[current] {
			if !visited[next] {
				visited[next] = true
				result = append(result, next)
				queue = append(queue, next)
			}
		}
	}

	return result
}

// validateProjectInput validates project input against the project's input_schema
// The input_schema is derived from the first executable step's block definition at save time
func (u *RunUsecase) validateProjectInput(ctx context.Context, tenantID, projectID uuid.UUID, input json.RawMessage) error {
	// In the multi-start project model, input validation is handled by each Start block's trigger_config
	// Project-level Variables are shared across all flows, not input schema
	// TODO: Implement input validation based on the target Start block's trigger_config.input_schema
	return nil
}

// Note: extractInputSchemaFromConfig has been moved to helpers.go as ExtractInputSchemaFromConfig
