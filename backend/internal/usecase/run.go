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
	workflowRepo repository.WorkflowRepository
	runRepo      repository.RunRepository
	versionRepo  repository.WorkflowVersionRepository
	stepRepo     repository.StepRepository
	edgeRepo     repository.EdgeRepository
	stepRunRepo  repository.StepRunRepository
	queue        *engine.Queue
}

// NewRunUsecase creates a new RunUsecase
func NewRunUsecase(
	workflowRepo repository.WorkflowRepository,
	runRepo repository.RunRepository,
	versionRepo repository.WorkflowVersionRepository,
	stepRepo repository.StepRepository,
	edgeRepo repository.EdgeRepository,
	stepRunRepo repository.StepRunRepository,
	redisClient *redis.Client,
) *RunUsecase {
	return &RunUsecase{
		workflowRepo: workflowRepo,
		runRepo:      runRepo,
		versionRepo:  versionRepo,
		stepRepo:     stepRepo,
		edgeRepo:     edgeRepo,
		stepRunRepo:  stepRunRepo,
		queue:        engine.NewQueue(redisClient),
	}
}

// CreateRunInput represents input for creating a run
type CreateRunInput struct {
	TenantID    uuid.UUID
	WorkflowID  uuid.UUID
	Version     int // 0 means latest version
	Input       json.RawMessage
	TriggeredBy domain.TriggerType // e.g., TriggerTypeManual, TriggerTypeTest
	UserID      *uuid.UUID
}

// Create creates and enqueues a new run
func (u *RunUsecase) Create(ctx context.Context, input CreateRunInput) (*domain.Run, error) {
	// Get workflow
	workflow, err := u.workflowRepo.GetByID(ctx, input.TenantID, input.WorkflowID)
	if err != nil {
		return nil, err
	}

	// Determine which version to use
	// 0 means use latest (current workflow version)
	version := input.Version
	if version == 0 {
		version = workflow.Version
	}

	// Validate that the requested version exists (if specific version requested)
	if input.Version > 0 && u.versionRepo != nil {
		_, err := u.versionRepo.GetByWorkflowAndVersion(ctx, workflow.ID, version)
		if err != nil {
			return nil, err
		}
	}

	// Validate input against Start step's input_schema
	if err := u.validateWorkflowInput(ctx, input.TenantID, workflow.ID, input.Input); err != nil {
		return nil, err
	}

	// Create run
	run := domain.NewRun(
		input.TenantID,
		workflow.ID,
		version,
		input.Input,
		input.TriggeredBy,
	)
	run.TriggeredByUser = input.UserID

	if err := u.runRepo.Create(ctx, run); err != nil {
		return nil, err
	}

	// Enqueue job
	job := &engine.Job{
		TenantID:        input.TenantID,
		WorkflowID:      workflow.ID,
		WorkflowVersion: version,
		RunID:           run.ID,
		Input:           input.Input,
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

// RunWithDefinitionOutput represents a run with its workflow definition
type RunWithDefinitionOutput struct {
	Run                *domain.Run                `json:"run"`
	WorkflowDefinition *domain.WorkflowDefinition `json:"workflow_definition,omitempty"`
}

// GetWithDetailsAndDefinition retrieves a run with step runs and workflow definition
func (u *RunUsecase) GetWithDetailsAndDefinition(ctx context.Context, tenantID, id uuid.UUID) (*RunWithDefinitionOutput, error) {
	run, err := u.runRepo.GetWithStepRuns(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	output := &RunWithDefinitionOutput{
		Run: run,
	}

	// Try to get the workflow definition from the version snapshot
	if u.versionRepo != nil {
		version, err := u.versionRepo.GetByWorkflowAndVersion(ctx, run.WorkflowID, run.WorkflowVersion)
		if err == nil && version != nil {
			var definition domain.WorkflowDefinition
			if err := json.Unmarshal(version.Definition, &definition); err == nil {
				output.WorkflowDefinition = &definition
				return output, nil
			}
		}
	}

	// Fallback: If version snapshot not found, fetch current workflow definition
	// This handles runs created before version snapshots were implemented
	workflow, err := u.workflowRepo.GetWithStepsAndEdges(ctx, tenantID, run.WorkflowID)
	if err == nil && workflow != nil {
		output.WorkflowDefinition = &domain.WorkflowDefinition{
			Name:        workflow.Name,
			Description: workflow.Description,
			InputSchema: workflow.InputSchema,
			Steps:       workflow.Steps,
			Edges:       workflow.Edges,
			BlockGroups: workflow.BlockGroups,
		}
	}

	return output, nil
}

// ListRunsInput represents input for listing runs
type ListRunsInput struct {
	TenantID    uuid.UUID
	WorkflowID  uuid.UUID
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

// List lists runs for a workflow
func (u *RunUsecase) List(ctx context.Context, input ListRunsInput) (*ListRunsOutput, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.Limit < 1 || input.Limit > 100 {
		input.Limit = 20
	}

	filter := repository.RunFilter{
		Status:      input.Status,
		TriggeredBy: input.TriggeredBy,
		Page:        input.Page,
		Limit:       input.Limit,
	}

	runs, total, err := u.runRepo.ListByWorkflow(ctx, input.TenantID, input.WorkflowID, filter)
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

// Cancel cancels a running workflow
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

	// 2. Get workflow definition from version snapshot
	version, err := u.versionRepo.GetByWorkflowAndVersion(ctx, run.WorkflowID, run.WorkflowVersion)
	if err != nil {
		return nil, err
	}

	var definition domain.WorkflowDefinition
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

	// 4. If step not found in version snapshot, try current workflow (for steps not in flow)
	if targetStep == nil {
		currentSteps, err := u.stepRepo.ListByWorkflow(ctx, input.TenantID, run.WorkflowID)
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
		WorkflowID:      run.WorkflowID,
		WorkflowVersion: run.WorkflowVersion,
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

	// 2. Get workflow definition from version snapshot
	version, err := u.versionRepo.GetByWorkflowAndVersion(ctx, run.WorkflowID, run.WorkflowVersion)
	if err != nil {
		return nil, err
	}

	var definition domain.WorkflowDefinition
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
		WorkflowID:      run.WorkflowID,
		WorkflowVersion: run.WorkflowVersion,
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

// ExecuteSystemWorkflowInput represents input for executing a system workflow
type ExecuteSystemWorkflowInput struct {
	TenantID        uuid.UUID              // Tenant context for the run
	SystemSlug      string                 // System workflow slug (e.g., "copilot-generate")
	Input           json.RawMessage        // Workflow input
	TriggerSource   string                 // Internal caller identifier (e.g., "copilot")
	TriggerMetadata map[string]interface{} // Additional metadata (feature, user_id, session_id, etc.)
	UserID          *uuid.UUID             // User who triggered the execution
}

// ExecuteSystemWorkflowOutput represents output for system workflow execution
type ExecuteSystemWorkflowOutput struct {
	RunID      uuid.UUID `json:"run_id"`
	WorkflowID uuid.UUID `json:"workflow_id"`
	Version    int       `json:"version"`
}

// ExecuteSystemWorkflow executes a system workflow by its slug
// This is used for internal system calls (e.g., Copilot meta-workflow)
// Returns immediately after creating the run - execution happens asynchronously
func (u *RunUsecase) ExecuteSystemWorkflow(ctx context.Context, input ExecuteSystemWorkflowInput) (*ExecuteSystemWorkflowOutput, error) {
	// 1. Look up system workflow by slug
	workflow, err := u.workflowRepo.GetSystemBySlug(ctx, input.SystemSlug)
	if err != nil {
		return nil, err
	}

	// 2. Validate workflow is published
	if workflow.Status != domain.WorkflowStatusPublished || workflow.Version < 1 {
		return nil, domain.ErrWorkflowNotPublished
	}

	// 3. Create run with internal trigger type
	run := domain.NewRun(
		input.TenantID,
		workflow.ID,
		workflow.Version,
		input.Input,
		domain.TriggerTypeInternal,
	)
	run.TriggeredByUser = input.UserID

	// 4. Set internal trigger metadata
	if err := run.SetInternalTrigger(input.TriggerSource, input.TriggerMetadata); err != nil {
		return nil, err
	}

	// 5. Save run to database
	if err := u.runRepo.Create(ctx, run); err != nil {
		return nil, err
	}

	// 6. Enqueue job for async execution
	job := &engine.Job{
		TenantID:        input.TenantID,
		WorkflowID:      workflow.ID,
		WorkflowVersion: workflow.Version,
		RunID:           run.ID,
		Input:           input.Input,
	}
	if err := u.queue.Enqueue(ctx, job); err != nil {
		return nil, err
	}

	return &ExecuteSystemWorkflowOutput{
		RunID:      run.ID,
		WorkflowID: workflow.ID,
		Version:    workflow.Version,
	}, nil
}

// TestStepInlineInput represents input for inline step testing
type TestStepInlineInput struct {
	TenantID   uuid.UUID
	WorkflowID uuid.UUID
	StepID     uuid.UUID
	Input      json.RawMessage // Custom input for testing
	UserID     *uuid.UUID
}

// TestStepInlineOutput represents output for inline step testing
type TestStepInlineOutput struct {
	RunID     uuid.UUID `json:"run_id"`
	StepID    uuid.UUID `json:"step_id"`
	StepName  string    `json:"step_name"`
	IsQueued  bool      `json:"is_queued"`
}

// TestStepInline creates a test run and executes only a single step
// This allows testing a step without requiring an existing run
func (u *RunUsecase) TestStepInline(ctx context.Context, input TestStepInlineInput) (*TestStepInlineOutput, error) {
	// 1. Get the current workflow (draft state)
	workflow, err := u.workflowRepo.GetByID(ctx, input.TenantID, input.WorkflowID)
	if err != nil {
		return nil, err
	}

	// 2. Get current steps from the workflow
	steps, err := u.stepRepo.ListByWorkflow(ctx, input.TenantID, input.WorkflowID)
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
	// The worker will automatically fall back to current workflow definition
	// when version 0 is not found (see worker/main.go processJob function)
	run := domain.NewRun(
		input.TenantID,
		workflow.ID,
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
	// - Fall back to current workflow definition
	// - Execute only the target step
	job := &engine.Job{
		TenantID:        input.TenantID,
		WorkflowID:      workflow.ID,
		WorkflowVersion: 0, // Worker will fall back to current workflow
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
func collectDownstreamSteps(def *domain.WorkflowDefinition, startStepID uuid.UUID) []uuid.UUID {
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

// validateWorkflowInput validates workflow input against the workflow's input_schema
// The input_schema is derived from the first executable step's block definition at save time
func (u *RunUsecase) validateWorkflowInput(ctx context.Context, tenantID, workflowID uuid.UUID, input json.RawMessage) error {
	// Get workflow to access its input_schema
	workflow, err := u.workflowRepo.GetByID(ctx, tenantID, workflowID)
	if err != nil {
		return nil // Skip validation if we can't get workflow
	}

	// Use workflow's input_schema (derived from first step's block definition at save time)
	if workflow.InputSchema == nil || len(workflow.InputSchema) == 0 {
		return nil // No input_schema defined, skip validation
	}

	// Validate input against schema
	return domain.ValidateInputSchema(input, workflow.InputSchema)
}

// extractInputSchemaFromConfig extracts the input_schema from a step's config
func extractInputSchemaFromConfig(config json.RawMessage) json.RawMessage {
	if config == nil || len(config) == 0 {
		return nil
	}

	var configMap map[string]json.RawMessage
	if err := json.Unmarshal(config, &configMap); err != nil {
		return nil
	}

	if inputSchema, ok := configMap["input_schema"]; ok {
		return inputSchema
	}

	return nil
}
