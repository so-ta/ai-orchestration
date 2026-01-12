package main

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/adapter"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/engine"
	"github.com/souta/ai-orchestration/internal/repository/postgres"
	"github.com/souta/ai-orchestration/pkg/database"
	redispkg "github.com/souta/ai-orchestration/pkg/redis"
)

func main() {
	log.Println("Starting AI Orchestration Worker...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Database connection
	dbURL := getEnv("DATABASE_URL", "postgres://aio:aio_password@localhost:5432/ai_orchestration?sslmode=disable")
	pool, err := database.NewPool(ctx, database.DefaultConfig(dbURL))
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()
	log.Println("Connected to database")

	// Redis connection
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379")
	redisClient, err := redispkg.NewClient(ctx, &redispkg.Config{URL: redisURL})
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()
	log.Println("Connected to Redis")

	// Initialize repositories
	workflowRepo := postgres.NewWorkflowRepository(pool)
	runRepo := postgres.NewRunRepository(pool)
	stepRunRepo := postgres.NewStepRunRepository(pool)
	versionRepo := postgres.NewWorkflowVersionRepository(pool)

	// Initialize adapter registry
	registry := adapter.NewRegistry()
	registry.Register(adapter.NewMockAdapter())
	registry.Register(adapter.NewOpenAIAdapter())
	registry.Register(adapter.NewAnthropicAdapter())
	registry.Register(adapter.NewHTTPAdapter())

	// Initialize executor
	executor := engine.NewExecutor(registry, logger)

	// Initialize queue
	queue := engine.NewQueue(redisClient)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Worker loop
	go func() {
		log.Println("Worker is running. Waiting for jobs...")
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Dequeue job with 5 second timeout
				job, err := queue.Dequeue(ctx, 5*time.Second)
				if err != nil {
					logger.Error("Failed to dequeue job", "error", err)
					continue
				}
				if job == nil {
					continue // timeout, no job
				}

				logger.Info("Processing job",
					"job_id", job.ID,
					"run_id", job.RunID,
					"workflow_id", job.WorkflowID,
				)

				// Process job
				if err := processJob(ctx, job, workflowRepo, runRepo, stepRunRepo, versionRepo, executor, logger); err != nil {
					logger.Error("Job processing failed",
						"job_id", job.ID,
						"run_id", job.RunID,
						"error", err,
					)
				}
			}
		}
	}()

	<-quit

	log.Println("Shutting down worker...")
	cancel()

	// Give time for cleanup
	time.Sleep(2 * time.Second)
	log.Println("Worker exited gracefully")
}

func processJob(
	ctx context.Context,
	job *engine.Job,
	workflowRepo *postgres.WorkflowRepository,
	runRepo *postgres.RunRepository,
	stepRunRepo *postgres.StepRunRepository,
	versionRepo *postgres.WorkflowVersionRepository,
	executor *engine.Executor,
	logger *slog.Logger,
) error {
	// Get run
	run, err := runRepo.GetByID(ctx, job.TenantID, job.RunID)
	if err != nil {
		return err
	}

	// Determine execution mode (default to full if not specified)
	executionMode := job.ExecutionMode
	if executionMode == "" {
		executionMode = engine.ExecutionModeFull
	}

	// Get workflow definition based on execution mode
	var def *domain.WorkflowDefinition

	if executionMode == engine.ExecutionModeSingleStep || executionMode == engine.ExecutionModeResume {
		// For partial execution, use versioned definition
		version, err := versionRepo.GetByWorkflowAndVersion(ctx, job.WorkflowID, job.WorkflowVersion)
		if err != nil {
			// Fallback to current workflow if version not found
			logger.Warn("Version not found, falling back to current workflow",
				"workflow_id", job.WorkflowID,
				"version", job.WorkflowVersion,
				"error", err,
			)
			workflow, err := workflowRepo.GetWithStepsAndEdges(ctx, job.TenantID, job.WorkflowID)
			if err != nil {
				return err
			}
			def = &domain.WorkflowDefinition{
				Name:        workflow.Name,
				Description: workflow.Description,
				InputSchema: workflow.InputSchema,
				Steps:       workflow.Steps,
				Edges:       workflow.Edges,
			}
		} else {
			if err := json.Unmarshal(version.Definition, &def); err != nil {
				return err
			}
		}
	} else {
		// For full execution, use current workflow
		workflow, err := workflowRepo.GetWithStepsAndEdges(ctx, job.TenantID, job.WorkflowID)
		if err != nil {
			return err
		}
		def = &domain.WorkflowDefinition{
			Name:        workflow.Name,
			Description: workflow.Description,
			InputSchema: workflow.InputSchema,
			Steps:       workflow.Steps,
			Edges:       workflow.Edges,
		}
	}

	// Create execution context
	execCtx := engine.NewExecutionContext(run, def)

	// Inject previous outputs for partial execution
	if job.InjectedOutputs != nil && len(job.InjectedOutputs) > 0 {
		execCtx.InjectPreviousOutputs(job.InjectedOutputs)
	}

	// Execute based on mode
	var execErr error

	switch executionMode {
	case engine.ExecutionModeSingleStep:
		// Single step execution - don't change run status, only execute one step
		if job.TargetStepID == nil {
			return domain.ErrStepNotFound
		}

		logger.Info("Executing single step",
			"run_id", job.RunID,
			"step_id", job.TargetStepID,
		)

		// Get max attempt for the entire run (Run-level unique)
		maxAttempt, _ := stepRunRepo.GetMaxAttemptForRun(ctx, job.RunID)
		newAttempt := maxAttempt + 1

		// Execute the single step
		stepRun, err := executor.ExecuteSingleStep(ctx, execCtx, *job.TargetStepID, job.StepInput)
		if err != nil {
			execErr = err
		}

		// Update attempt number
		if stepRun != nil {
			stepRun.Attempt = newAttempt
			// Save step run
			if err := stepRunRepo.Create(ctx, stepRun); err != nil {
				logger.Error("Failed to save step run",
					"run_id", run.ID,
					"step_id", stepRun.StepID,
					"error", err,
				)
			}
		}

		// Don't update run status for single step execution
		return execErr

	case engine.ExecutionModeResume:
		// Resume execution from a specific step
		if job.TargetStepID == nil {
			return domain.ErrStepNotFound
		}

		logger.Info("Resuming execution from step",
			"run_id", job.RunID,
			"from_step_id", job.TargetStepID,
		)

		// Get max attempt for the entire run (Run-level unique)
		maxAttempt, _ := stepRunRepo.GetMaxAttemptForRun(ctx, job.RunID)
		newAttempt := maxAttempt + 1

		// Execute from step
		execErr = executor.ExecuteFromStep(ctx, execCtx, *job.TargetStepID, job.StepInput)

		// Persist step runs to database (all steps in this resume get the same attempt number)
		for _, stepRun := range execCtx.StepRuns {
			stepRun.Attempt = newAttempt

			if err := stepRunRepo.Create(ctx, stepRun); err != nil {
				logger.Error("Failed to save step run",
					"run_id", run.ID,
					"step_id", stepRun.StepID,
					"error", err,
				)
			}
		}

		// Don't update run status for resume execution
		return execErr

	default:
		// Full execution (existing behavior)
		// Start run
		run.Start()
		if err := runRepo.Update(ctx, run); err != nil {
			return err
		}

		// Get max attempt for the entire run (Run-level unique)
		maxAttempt, _ := stepRunRepo.GetMaxAttemptForRun(ctx, job.RunID)
		newAttempt := maxAttempt + 1

		// Execute workflow
		execErr = executor.Execute(ctx, execCtx)

		// Persist step runs to database (all steps in this execution get the same attempt number)
		for _, stepRun := range execCtx.StepRuns {
			stepRun.Attempt = newAttempt
			if err := stepRunRepo.Create(ctx, stepRun); err != nil {
				logger.Error("Failed to save step run",
					"run_id", run.ID,
					"step_id", stepRun.StepID,
					"error", err,
				)
			}
		}

		// Update run status
		if execErr != nil {
			run.Fail(execErr.Error())
		} else {
			// Collect final output from terminal nodes (nodes with no outgoing edges)
			var output json.RawMessage
			if len(execCtx.StepData) > 0 {
				// Find terminal nodes (steps with no outgoing edges)
				terminalSteps := findTerminalSteps(def.Steps, def.Edges)

				// If we have terminal steps, use their outputs
				if len(terminalSteps) > 0 {
					outputs := make(map[string]interface{})
					for _, stepID := range terminalSteps {
						if data, ok := execCtx.StepData[stepID]; ok {
							var stepOutput interface{}
							json.Unmarshal(data, &stepOutput)
							outputs[stepID.String()] = stepOutput
						}
					}
					// If only one terminal step, use its output directly
					if len(outputs) == 1 {
						for _, v := range outputs {
							output, _ = json.Marshal(v)
						}
					} else {
						output, _ = json.Marshal(outputs)
					}
				} else {
					// Fallback: use last executed step output
					for _, data := range execCtx.StepData {
						output = data
					}
				}
			}
			run.Complete(output)
		}

		if err := runRepo.Update(ctx, run); err != nil {
			logger.Error("Failed to update run status", "run_id", run.ID, "error", err)
		}

		return execErr
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// findTerminalSteps returns step IDs that have no outgoing edges
func findTerminalSteps(steps []domain.Step, edges []domain.Edge) []uuid.UUID {
	// Build set of steps that have outgoing edges
	hasOutgoing := make(map[uuid.UUID]bool)
	for _, edge := range edges {
		hasOutgoing[edge.SourceStepID] = true
	}

	// Find steps with no outgoing edges
	var terminal []uuid.UUID
	for _, step := range steps {
		if !hasOutgoing[step.ID] {
			terminal = append(terminal, step.ID)
		}
	}
	return terminal
}
