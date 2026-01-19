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
	"github.com/joho/godotenv"
	"github.com/souta/ai-orchestration/internal/adapter"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/engine"
	"github.com/souta/ai-orchestration/internal/repository/postgres"
	"github.com/souta/ai-orchestration/pkg/database"
	redispkg "github.com/souta/ai-orchestration/pkg/redis"
)

func main() {
	// Load .env file (try multiple locations)
	for _, path := range []string{"../.env", ".env"} {
		if err := godotenv.Load(path); err == nil {
			log.Printf("Loaded .env from: %s", path)
			break
		}
	}

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
	projectRepo := postgres.NewProjectRepository(pool)
	runRepo := postgres.NewRunRepository(pool)
	stepRunRepo := postgres.NewStepRunRepository(pool)
	versionRepo := postgres.NewProjectVersionRepository(pool)
	usageRepo := postgres.NewUsageRepository(pool)
	blockDefRepo := postgres.NewBlockDefinitionRepository(pool)

	// Initialize adapter registry
	registry := adapter.NewRegistry()
	registry.Register(adapter.NewMockAdapter())
	registry.Register(adapter.NewOpenAIAdapter())
	registry.Register(adapter.NewAnthropicAdapter())
	registry.Register(adapter.NewHTTPAdapter())

	// Initialize usage recorder for cost tracking
	usageRecorder := engine.NewUsageRecorder(usageRepo, logger)

	// Initialize executor with usage recorder, database pool, and block definition repository
	executor := engine.NewExecutor(registry, logger,
		engine.WithUsageRecorder(usageRecorder),
		engine.WithDatabase(pool),
		engine.WithBlockDefinitionRepository(blockDefRepo),
	)

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
					log.Println("Dequeue timeout, no job")
					continue // timeout, no job
				}

				// Debug: log all job fields including ProjectTenantID
				projectTenantIDStr := "nil"
				if job.ProjectTenantID != nil {
					projectTenantIDStr = job.ProjectTenantID.String()
				}
				logger.Info("Processing job",
					"job_id", job.ID,
					"run_id", job.RunID,
					"project_id", job.ProjectID,
					"job_tenant_id", job.TenantID,
					"project_tenant_id", projectTenantIDStr,
				)

				// Process job
				if err := processJob(ctx, job, projectRepo, runRepo, stepRunRepo, versionRepo, executor, logger); err != nil {
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
	projectRepo *postgres.ProjectRepository,
	runRepo *postgres.RunRepository,
	stepRunRepo *postgres.StepRunRepository,
	versionRepo *postgres.ProjectVersionRepository,
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

	// For system projects, use the project's tenant_id to fetch the project
	// This allows runs created by different tenants to execute system projects
	projectTenantID := job.TenantID
	if job.ProjectTenantID != nil {
		projectTenantID = *job.ProjectTenantID
	}

	// Get project definition based on execution mode
	var def *domain.ProjectDefinition

	if executionMode == engine.ExecutionModeSingleStep || executionMode == engine.ExecutionModeResume {
		// For partial execution, use versioned definition
		version, err := versionRepo.GetByProjectAndVersion(ctx, job.ProjectID, job.ProjectVersion)
		if err != nil {
			// Fallback to current project if version not found
			logger.Warn("Version not found, falling back to current project",
				"project_id", job.ProjectID,
				"version", job.ProjectVersion,
				"error", err,
			)
			project, err := projectRepo.GetWithStepsAndEdges(ctx, projectTenantID, job.ProjectID)
			if err != nil {
				return err
			}
			def = &domain.ProjectDefinition{
				Name:        project.Name,
				Description: project.Description,
				Variables:   project.Variables,
				Steps:       project.Steps,
				Edges:       project.Edges,
				BlockGroups: project.BlockGroups,
			}
		} else {
			if err := json.Unmarshal(version.Definition, &def); err != nil {
				return err
			}
		}
	} else {
		// For full execution, use current project
		project, err := projectRepo.GetWithStepsAndEdges(ctx, projectTenantID, job.ProjectID)
		if err != nil {
			return err
		}
		def = &domain.ProjectDefinition{
			Name:        project.Name,
			Description: project.Description,
			Variables:   project.Variables,
			Steps:       project.Steps,
			Edges:       project.Edges,
			BlockGroups: project.BlockGroups,
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

		// Check if step exists in the definition, if not fallback to current workflow
		stepExistsInDef := false
		for _, s := range def.Steps {
			if s.ID == *job.TargetStepID {
				stepExistsInDef = true
				break
			}
		}

		// If step not found in version definition, try current project (for steps not in flow)
		if !stepExistsInDef {
			logger.Info("Step not found in version definition, trying current project",
				"step_id", job.TargetStepID,
				"project_id", job.ProjectID,
			)
			currentProject, err := projectRepo.GetWithStepsAndEdges(ctx, projectTenantID, job.ProjectID)
			if err != nil {
				return err
			}
			// Look for the step in current project
			for _, s := range currentProject.Steps {
				if s.ID == *job.TargetStepID {
					// Add the step to definition for execution
					def.Steps = append(def.Steps, s)
					// Update execution context with new definition
					execCtx = engine.NewExecutionContext(run, def)
					if job.InjectedOutputs != nil && len(job.InjectedOutputs) > 0 {
						execCtx.InjectPreviousOutputs(job.InjectedOutputs)
					}
					stepExistsInDef = true
					logger.Info("Step found in current workflow",
						"step_id", job.TargetStepID,
						"step_name", s.Name,
					)
					break
				}
			}
		}

		if !stepExistsInDef {
			logger.Error("Step not found in version or current project",
				"step_id", job.TargetStepID,
			)
			return domain.ErrStepNotFound
		}

		logger.Info("Executing single step",
			"run_id", job.RunID,
			"step_id", job.TargetStepID,
		)

		// Start run (set started_at)
		run.Start()
		if err := runRepo.Update(ctx, run); err != nil {
			return err
		}

		// Get max attempt for the entire run (Run-level unique)
		maxAttempt, err := stepRunRepo.GetMaxAttemptForRun(ctx, run.TenantID, job.RunID)
		if err != nil {
			logger.Warn("Failed to get max attempt, defaulting to 0", "error", err)
			maxAttempt = 0
		}
		newAttempt := maxAttempt + 1

		// Get max sequence number for the run and set counter
		maxSeq, err := stepRunRepo.GetMaxSequenceNumberForRun(ctx, run.TenantID, job.RunID)
		if err != nil {
			logger.Warn("Failed to get max sequence number, defaulting to 0", "error", err)
			maxSeq = 0
		}
		execCtx.SetSequenceCounter(maxSeq)

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

		// Update run status for single step execution
		if execErr != nil {
			run.Fail(execErr.Error())
		} else {
			// Use step output as run output for single step execution
			var output json.RawMessage
			if stepRun != nil && stepRun.Output != nil {
				output = stepRun.Output
			}
			run.Complete(output)
		}

		if err := runRepo.Update(ctx, run); err != nil {
			logger.Error("Failed to update run status", "run_id", run.ID, "error", err)
		}

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

		// Start run (set started_at)
		run.Start()
		if err := runRepo.Update(ctx, run); err != nil {
			return err
		}

		// Get max attempt for the entire run (Run-level unique)
		maxAttempt, err := stepRunRepo.GetMaxAttemptForRun(ctx, run.TenantID, job.RunID)
		if err != nil {
			logger.Warn("Failed to get max attempt, defaulting to 0", "error", err)
			maxAttempt = 0
		}
		newAttempt := maxAttempt + 1

		// Get max sequence number for the run and set counter
		maxSeq, err := stepRunRepo.GetMaxSequenceNumberForRun(ctx, run.TenantID, job.RunID)
		if err != nil {
			logger.Warn("Failed to get max sequence number, defaulting to 0", "error", err)
			maxSeq = 0
		}
		execCtx.SetSequenceCounter(maxSeq)

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

		// Update run status for resume execution
		if execErr != nil {
			run.Fail(execErr.Error())
		} else {
			// Collect output from terminal steps
			var output json.RawMessage
			if len(execCtx.StepData) > 0 {
				terminalSteps := findTerminalSteps(def.Steps, def.Edges)
				if len(terminalSteps) > 0 {
					outputs := make(map[string]interface{})
					for _, stepID := range terminalSteps {
						if data, ok := execCtx.StepData[stepID]; ok {
							var stepOutput interface{}
							json.Unmarshal(data, &stepOutput)
							outputs[stepID.String()] = stepOutput
						}
					}
					if len(outputs) == 1 {
						for _, v := range outputs {
							output, _ = json.Marshal(v)
						}
					} else {
						output, _ = json.Marshal(outputs)
					}
				} else {
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

	default:
		// Full execution (existing behavior)
		// Start run
		run.Start()
		if err := runRepo.Update(ctx, run); err != nil {
			return err
		}

		// Get max attempt for the entire run (Run-level unique)
		maxAttempt, err := stepRunRepo.GetMaxAttemptForRun(ctx, run.TenantID, job.RunID)
		if err != nil {
			logger.Warn("Failed to get max attempt, defaulting to 0", "error", err)
			maxAttempt = 0
		}
		newAttempt := maxAttempt + 1

		// Execute project DAG
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
		if edge.SourceStepID != nil {
			hasOutgoing[*edge.SourceStepID] = true
		}
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
