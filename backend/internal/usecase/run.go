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
	queue        *engine.Queue
}

// NewRunUsecase creates a new RunUsecase
func NewRunUsecase(
	workflowRepo repository.WorkflowRepository,
	runRepo repository.RunRepository,
	versionRepo repository.WorkflowVersionRepository,
	stepRepo repository.StepRepository,
	edgeRepo repository.EdgeRepository,
	redisClient *redis.Client,
) *RunUsecase {
	return &RunUsecase{
		workflowRepo: workflowRepo,
		runRepo:      runRepo,
		versionRepo:  versionRepo,
		stepRepo:     stepRepo,
		edgeRepo:     edgeRepo,
		queue:        engine.NewQueue(redisClient),
	}
}

// CreateRunInput represents input for creating a run
type CreateRunInput struct {
	TenantID   uuid.UUID
	WorkflowID uuid.UUID
	Version    int // 0 means latest version
	Input      json.RawMessage
	Mode       domain.RunMode
	UserID     *uuid.UUID
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

	// Create run
	run := domain.NewRun(
		input.TenantID,
		workflow.ID,
		version,
		input.Input,
		input.Mode,
		domain.TriggerTypeManual,
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
	workflow, err := u.workflowRepo.GetByID(ctx, tenantID, run.WorkflowID)
	if err == nil && workflow != nil {
		stepPtrs, _ := u.stepRepo.ListByWorkflow(ctx, run.WorkflowID)
		edgePtrs, _ := u.edgeRepo.ListByWorkflow(ctx, run.WorkflowID)

		// Convert pointer slices to value slices
		steps := make([]domain.Step, len(stepPtrs))
		for i, s := range stepPtrs {
			steps[i] = *s
		}
		edges := make([]domain.Edge, len(edgePtrs))
		for i, e := range edgePtrs {
			edges[i] = *e
		}

		output.WorkflowDefinition = &domain.WorkflowDefinition{
			Name:        workflow.Name,
			Description: workflow.Description,
			InputSchema: workflow.InputSchema,
			Steps:       steps,
			Edges:       edges,
		}
	}

	return output, nil
}

// ListRunsInput represents input for listing runs
type ListRunsInput struct {
	TenantID   uuid.UUID
	WorkflowID uuid.UUID
	Status     *domain.RunStatus
	Mode       *domain.RunMode
	Page       int
	Limit      int
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
		Status: input.Status,
		Mode:   input.Mode,
		Page:   input.Page,
		Limit:  input.Limit,
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
