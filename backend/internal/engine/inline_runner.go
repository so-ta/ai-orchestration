package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/souta/ai-orchestration/internal/adapter"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// InlineRunner provides inline workflow execution with event streaming
// This is used for SSE endpoints where we need synchronous execution with real-time events
type InlineRunner struct {
	executor     *Executor
	projectRepo  repository.ProjectRepository
	runRepo      repository.RunRepository
	stepRunRepo  repository.StepRunRepository
	versionRepo  repository.ProjectVersionRepository
	logger       *slog.Logger
}

// NewInlineRunner creates a new inline runner
func NewInlineRunner(
	executor *Executor,
	projectRepo repository.ProjectRepository,
	runRepo repository.RunRepository,
	stepRunRepo repository.StepRunRepository,
	versionRepo repository.ProjectVersionRepository,
	logger *slog.Logger,
) *InlineRunner {
	return &InlineRunner{
		executor:    executor,
		projectRepo: projectRepo,
		runRepo:     runRepo,
		stepRunRepo: stepRunRepo,
		versionRepo: versionRepo,
		logger:      logger,
	}
}

// RunInput represents input for running a workflow inline
type RunInput struct {
	TenantID    uuid.UUID
	ProjectID   uuid.UUID
	RunID       uuid.UUID           // Optional: existing run ID (nil creates new run)
	Input       json.RawMessage
	TriggeredBy domain.TriggerType
	UserID      *uuid.UUID
	StartStepID *uuid.UUID          // Required: which Start block to execute from
}

// RunWithEvents executes a workflow inline with event streaming
// The events channel will receive execution events for SSE streaming
// This method blocks until execution completes
func (r *InlineRunner) RunWithEvents(ctx context.Context, input RunInput, events chan<- ExecutionEvent) (*domain.Run, error) {
	defer func() {
		if events != nil {
			close(events)
		}
	}()

	// Create or get run
	var run *domain.Run
	var err error

	if input.RunID != uuid.Nil {
		run, err = r.runRepo.GetByID(ctx, input.TenantID, input.RunID)
		if err != nil {
			return nil, fmt.Errorf("get run: %w", err)
		}
	} else {
		// Create new run
		run = domain.NewRun(
			input.TenantID,
			input.ProjectID,
			0, // Will be set from project
			input.Input,
			input.TriggeredBy,
		)
		run.TriggeredByUser = input.UserID
		run.StartStepID = input.StartStepID

		// Get project to set version
		project, err := r.projectRepo.GetByID(ctx, input.TenantID, input.ProjectID)
		if err != nil {
			return nil, fmt.Errorf("get project: %w", err)
		}
		run.ProjectVersion = project.Version

		if err := r.runRepo.Create(ctx, run); err != nil {
			return nil, fmt.Errorf("create run: %w", err)
		}
	}

	// Get project definition
	def, err := r.getProjectDefinition(ctx, input.TenantID, input.ProjectID, run.ProjectVersion)
	if err != nil {
		return nil, fmt.Errorf("get project definition: %w", err)
	}

	// Update run status to running
	run.Start()
	if err := r.runRepo.Update(ctx, run); err != nil {
		r.logger.Warn("Failed to update run status", "error", err)
	}

	// Create execution context
	execCtx := NewExecutionContext(run, def)

	// Execute with events
	execErr := r.executor.ExecuteWithEvents(ctx, execCtx, events)

	// Save step runs
	for _, stepRun := range execCtx.StepRuns {
		if err := r.stepRunRepo.Create(ctx, stepRun); err != nil {
			r.logger.Warn("Failed to save step run", "step_run_id", stepRun.ID, "error", err)
		}
	}

	// Update run status
	if execErr != nil {
		run.Fail(execErr.Error())
	} else {
		// Get final output from execution context
		var finalOutput json.RawMessage
		if run.StartStepID != nil {
			// Try to get output from the last executed step
			execCtx.mu.RLock()
			for stepID, output := range execCtx.StepData {
				_ = stepID
				finalOutput = output // Use last output
			}
			execCtx.mu.RUnlock()
		}
		run.Complete(finalOutput)
	}

	if err := r.runRepo.Update(ctx, run); err != nil {
		r.logger.Warn("Failed to update run status", "error", err)
	}

	return run, execErr
}

// getProjectDefinition retrieves the project definition for a given version
func (r *InlineRunner) getProjectDefinition(ctx context.Context, tenantID, projectID uuid.UUID, version int) (*domain.ProjectDefinition, error) {
	// Try to get from version repo first
	if r.versionRepo != nil && version > 0 {
		versionData, err := r.versionRepo.GetByProjectAndVersion(ctx, projectID, version)
		if err == nil && versionData != nil {
			var def domain.ProjectDefinition
			if err := json.Unmarshal(versionData.Definition, &def); err == nil {
				return &def, nil
			}
		}
	}

	// Fallback to current project
	project, err := r.projectRepo.GetWithStepsAndEdges(ctx, tenantID, projectID)
	if err != nil {
		return nil, err
	}

	return &domain.ProjectDefinition{
		Name:        project.Name,
		Description: project.Description,
		Variables:   project.Variables,
		Steps:       project.Steps,
		Edges:       project.Edges,
		BlockGroups: project.BlockGroups,
	}, nil
}

// CreateExecutorForInlineExecution creates an executor for inline execution
// This is a helper for creating an executor with the same configuration as the worker
func CreateExecutorForInlineExecution(
	pool *pgxpool.Pool,
	blockDefRepo repository.BlockDefinitionRepository,
	logger *slog.Logger,
) *Executor {
	// Create a registry with basic adapters for inline execution
	// For most workflows (especially Copilot), the sandbox-based execution is used
	// which doesn't require adapters
	registry := adapter.NewRegistry()

	// Register common adapters
	registry.Register(adapter.NewMockAdapter())
	registry.Register(adapter.NewOpenAIAdapter())
	registry.Register(adapter.NewAnthropicAdapter())
	registry.Register(adapter.NewHTTPAdapter())

	return NewExecutor(registry, logger,
		WithDatabase(pool),
		WithBlockDefinitionRepository(blockDefRepo),
	)
}

// InlineRunnerFactory creates InlineRunner instances with proper configuration
type InlineRunnerFactory struct {
	pool         *pgxpool.Pool
	projectRepo  repository.ProjectRepository
	runRepo      repository.RunRepository
	stepRunRepo  repository.StepRunRepository
	versionRepo  repository.ProjectVersionRepository
	blockDefRepo repository.BlockDefinitionRepository
	logger       *slog.Logger
	executor     *Executor
}

// NewInlineRunnerFactory creates a new inline runner factory
func NewInlineRunnerFactory(
	pool *pgxpool.Pool,
	projectRepo repository.ProjectRepository,
	runRepo repository.RunRepository,
	stepRunRepo repository.StepRunRepository,
	versionRepo repository.ProjectVersionRepository,
	blockDefRepo repository.BlockDefinitionRepository,
	logger *slog.Logger,
) *InlineRunnerFactory {
	return &InlineRunnerFactory{
		pool:         pool,
		projectRepo:  projectRepo,
		runRepo:      runRepo,
		stepRunRepo:  stepRunRepo,
		versionRepo:  versionRepo,
		blockDefRepo: blockDefRepo,
		logger:       logger,
	}
}

// Create creates a new InlineRunner
// The executor is lazily initialized on first call
func (f *InlineRunnerFactory) Create() *InlineRunner {
	if f.executor == nil {
		f.executor = CreateExecutorForInlineExecution(f.pool, f.blockDefRepo, f.logger)
	}
	return NewInlineRunner(
		f.executor,
		f.projectRepo,
		f.runRepo,
		f.stepRunRepo,
		f.versionRepo,
		f.logger,
	)
}

// ExecutionTimeout is the default timeout for inline execution
const ExecutionTimeout = 5 * time.Minute
