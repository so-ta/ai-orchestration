package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/souta/ai-orchestration/internal/adapter"
	"github.com/souta/ai-orchestration/internal/domain"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// BlockGroupExecutor handles execution of block groups (control flow constructs)
type BlockGroupExecutor struct {
	registry  *adapter.Registry
	logger    *slog.Logger
	evaluator *ConditionEvaluator
	executor  *Executor // Reference to main executor for step execution
}

// NewBlockGroupExecutor creates a new block group executor
func NewBlockGroupExecutor(registry *adapter.Registry, logger *slog.Logger, executor *Executor) *BlockGroupExecutor {
	return &BlockGroupExecutor{
		registry:  registry,
		logger:    logger,
		evaluator: NewConditionEvaluator(),
		executor:  executor,
	}
}

// BlockGroupContext holds context for block group execution
type BlockGroupContext struct {
	Group    *domain.BlockGroup
	Steps    []*domain.Step // Steps belonging to this group
	Input    json.RawMessage
	ExecCtx  *ExecutionContext
	Graph    *Graph
}

// ExecuteGroup executes a block group based on its type
func (e *BlockGroupExecutor) ExecuteGroup(ctx context.Context, bgCtx *BlockGroupContext) (json.RawMessage, error) {
	ctx, span := tracer.Start(ctx, "block_group.execute",
		trace.WithAttributes(
			attribute.String("group_id", bgCtx.Group.ID.String()),
			attribute.String("group_type", string(bgCtx.Group.Type)),
			attribute.String("group_name", bgCtx.Group.Name),
		),
	)
	defer span.End()

	e.logger.Info("Executing block group",
		"group_id", bgCtx.Group.ID,
		"group_type", bgCtx.Group.Type,
		"step_count", len(bgCtx.Steps),
	)

	var result json.RawMessage
	var err error

	switch bgCtx.Group.Type {
	case domain.BlockGroupTypeParallel:
		result, err = e.executeParallel(ctx, bgCtx)
	case domain.BlockGroupTypeTryCatch:
		result, err = e.executeTryCatch(ctx, bgCtx)
	case domain.BlockGroupTypeIfElse:
		result, err = e.executeIfElse(ctx, bgCtx)
	case domain.BlockGroupTypeSwitchCase:
		result, err = e.executeSwitchCase(ctx, bgCtx)
	case domain.BlockGroupTypeForeach:
		result, err = e.executeForeach(ctx, bgCtx)
	case domain.BlockGroupTypeWhile:
		result, err = e.executeWhile(ctx, bgCtx)
	default:
		err = fmt.Errorf("unknown block group type: %s", bgCtx.Group.Type)
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetStatus(codes.Ok, "block group completed")
	return result, nil
}

// executeParallel executes all steps in parallel and waits for all to complete
func (e *BlockGroupExecutor) executeParallel(ctx context.Context, bgCtx *BlockGroupContext) (json.RawMessage, error) {
	ctx, span := tracer.Start(ctx, "block_group.parallel",
		trace.WithAttributes(
			attribute.Int("step_count", len(bgCtx.Steps)),
		),
	)
	defer span.End()

	// Parse parallel config
	var config domain.ParallelConfig
	if bgCtx.Group.Config != nil {
		if err := json.Unmarshal(bgCtx.Group.Config, &config); err != nil {
			e.logger.Warn("Failed to parse parallel config, using defaults", "error", err)
		}
	}

	// Filter steps that belong to body role
	var bodySteps []*domain.Step
	for _, step := range bgCtx.Steps {
		if step.GroupRole == "" || step.GroupRole == string(domain.GroupRoleBody) {
			bodySteps = append(bodySteps, step)
		}
	}

	if len(bodySteps) == 0 {
		e.logger.Info("No steps to execute in parallel group")
		return json.RawMessage("{}"), nil
	}

	span.SetAttributes(attribute.Int("body_step_count", len(bodySteps)))

	// Determine concurrency limit
	maxConcurrent := config.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = len(bodySteps) // Unlimited = all at once
	}

	// Results collection
	results := make(map[string]interface{})
	var resultsMu sync.Mutex
	var firstError error
	var errorMu sync.Mutex

	// Semaphore for concurrency control
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for _, step := range bodySteps {
		// Check if we should fail fast
		if config.FailFast {
			errorMu.Lock()
			if firstError != nil {
				errorMu.Unlock()
				break
			}
			errorMu.Unlock()
		}

		wg.Add(1)
		go func(s *domain.Step) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Execute step
			output, err := e.executeStep(ctx, bgCtx.ExecCtx, bgCtx.Graph, s, bgCtx.Input)

			if err != nil {
				e.logger.Error("Parallel step failed",
					"step_id", s.ID,
					"step_name", s.Name,
					"error", err,
				)
				if config.FailFast {
					errorMu.Lock()
					if firstError == nil {
						firstError = err
					}
					errorMu.Unlock()
				}
				return
			}

			// Store result
			resultsMu.Lock()
			var outputData interface{}
			if err := json.Unmarshal(output, &outputData); err == nil {
				results[s.Name] = outputData
			} else {
				results[s.Name] = string(output)
			}
			resultsMu.Unlock()

		}(step)
	}

	wg.Wait()

	// Check for fail-fast error
	if config.FailFast && firstError != nil {
		return nil, firstError
	}

	// Build output
	output := map[string]interface{}{
		"results":   results,
		"completed": true,
		"count":     len(bodySteps),
	}

	return json.Marshal(output)
}

// executeStep executes a single step within a block group context
func (e *BlockGroupExecutor) executeStep(ctx context.Context, execCtx *ExecutionContext, graph *Graph, step *domain.Step, input json.RawMessage) (json.RawMessage, error) {
	ctx, span := tracer.Start(ctx, "block_group.step",
		trace.WithAttributes(
			attribute.String("step_id", step.ID.String()),
			attribute.String("step_name", step.Name),
			attribute.String("step_type", string(step.Type)),
		),
	)
	defer span.End()

	// Use the main executor's step execution logic
	err := e.executor.executeNode(ctx, execCtx, graph, step.ID)
	if err != nil {
		return nil, err
	}

	// Get output from execution context
	execCtx.mu.RLock()
	output := execCtx.StepData[step.ID]
	execCtx.mu.RUnlock()

	return output, nil
}

// executeTryCatch executes try block, and on error executes catch block
func (e *BlockGroupExecutor) executeTryCatch(ctx context.Context, bgCtx *BlockGroupContext) (json.RawMessage, error) {
	ctx, span := tracer.Start(ctx, "block_group.try_catch")
	defer span.End()

	// Parse config
	var config domain.TryCatchConfig
	if bgCtx.Group.Config != nil {
		if err := json.Unmarshal(bgCtx.Group.Config, &config); err != nil {
			e.logger.Warn("Failed to parse try-catch config, using defaults", "error", err)
		}
	}

	// Separate steps by role
	var trySteps, catchSteps, finallySteps []*domain.Step
	for _, step := range bgCtx.Steps {
		switch step.GroupRole {
		case string(domain.GroupRoleTry), string(domain.GroupRoleBody):
			trySteps = append(trySteps, step)
		case string(domain.GroupRoleCatch):
			catchSteps = append(catchSteps, step)
		case string(domain.GroupRoleFinally):
			finallySteps = append(finallySteps, step)
		}
	}

	var output json.RawMessage
	var tryError error

	// Execute try block
	for _, step := range trySteps {
		output, tryError = e.executeStep(ctx, bgCtx.ExecCtx, bgCtx.Graph, step, bgCtx.Input)
		if tryError != nil {
			break
		}
	}

	// If error occurred, execute catch block
	if tryError != nil {
		e.logger.Info("Try block failed, executing catch", "error", tryError)
		span.AddEvent("catch_triggered")

		// Prepare catch input with error info
		catchInput := map[string]interface{}{
			"error":   tryError.Error(),
			"input":   json.RawMessage(bgCtx.Input),
		}
		catchInputJSON, err := json.Marshal(catchInput)
		if err != nil {
			e.logger.Error("Failed to marshal catch input", "error", err)
			catchInputJSON = bgCtx.Input
		}

		for _, step := range catchSteps {
			var stepErr error
			output, stepErr = e.executeStep(ctx, bgCtx.ExecCtx, bgCtx.Graph, step, catchInputJSON)
			if stepErr != nil {
				e.logger.Error("Catch step failed", "step_id", step.ID, "error", stepErr)
			}
		}
	}

	// Always execute finally block
	if len(finallySteps) > 0 {
		span.AddEvent("finally_executing")
		for _, step := range finallySteps {
			if _, err := e.executeStep(ctx, bgCtx.ExecCtx, bgCtx.Graph, step, bgCtx.Input); err != nil {
				e.logger.Error("Finally step failed", "step_id", step.ID, "error", err)
			}
		}
	}

	if output == nil {
		output = json.RawMessage("{}")
	}
	return output, nil
}

// executeIfElse executes then or else branch based on condition
func (e *BlockGroupExecutor) executeIfElse(ctx context.Context, bgCtx *BlockGroupContext) (json.RawMessage, error) {
	ctx, span := tracer.Start(ctx, "block_group.if_else")
	defer span.End()

	// Parse config
	var config domain.IfElseConfig
	if bgCtx.Group.Config != nil {
		if err := json.Unmarshal(bgCtx.Group.Config, &config); err != nil {
			e.logger.Warn("Failed to parse if-else config, using defaults", "error", err)
		}
	}

	// Evaluate condition
	conditionResult, err := e.evaluator.Evaluate(config.Condition, bgCtx.Input)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate if condition: %w", err)
	}

	span.SetAttributes(attribute.Bool("condition_result", conditionResult))

	// Separate steps by role
	var thenSteps, elseSteps []*domain.Step
	for _, step := range bgCtx.Steps {
		switch step.GroupRole {
		case string(domain.GroupRoleThen), string(domain.GroupRoleBody):
			thenSteps = append(thenSteps, step)
		case string(domain.GroupRoleElse):
			elseSteps = append(elseSteps, step)
		}
	}

	// Execute appropriate branch
	var stepsToExecute []*domain.Step
	if conditionResult {
		stepsToExecute = thenSteps
		span.AddEvent("then_branch_executing")
	} else {
		stepsToExecute = elseSteps
		span.AddEvent("else_branch_executing")
	}

	var output json.RawMessage
	for _, step := range stepsToExecute {
		output, err = e.executeStep(ctx, bgCtx.ExecCtx, bgCtx.Graph, step, bgCtx.Input)
		if err != nil {
			return nil, err
		}
	}

	if output == nil {
		output = json.RawMessage("{}")
	}
	return output, nil
}

// executeSwitchCase executes the matching case branch
func (e *BlockGroupExecutor) executeSwitchCase(ctx context.Context, bgCtx *BlockGroupContext) (json.RawMessage, error) {
	ctx, span := tracer.Start(ctx, "block_group.switch_case")
	defer span.End()

	// Parse config
	var config domain.SwitchCaseConfig
	if bgCtx.Group.Config != nil {
		if err := json.Unmarshal(bgCtx.Group.Config, &config); err != nil {
			e.logger.Warn("Failed to parse switch-case config, using defaults", "error", err)
		}
	}

	// Group steps by role (case_0, case_1, default, etc.)
	stepsByRole := make(map[string][]*domain.Step)
	for _, step := range bgCtx.Steps {
		role := step.GroupRole
		if role == "" {
			role = string(domain.GroupRoleDefault)
		}
		stepsByRole[role] = append(stepsByRole[role], step)
	}

	// Evaluate expression and find matching case
	var matchedRole string
	for i, caseExpr := range config.Cases {
		caseRole := fmt.Sprintf("case_%d", i)
		result, err := e.evaluator.Evaluate(caseExpr, bgCtx.Input)
		if err == nil && result {
			matchedRole = caseRole
			break
		}
	}

	// If no match, use default
	if matchedRole == "" {
		matchedRole = string(domain.GroupRoleDefault)
	}

	span.SetAttributes(attribute.String("matched_case", matchedRole))

	// Execute matched case steps
	stepsToExecute := stepsByRole[matchedRole]
	var output json.RawMessage
	var err error

	for _, step := range stepsToExecute {
		output, err = e.executeStep(ctx, bgCtx.ExecCtx, bgCtx.Graph, step, bgCtx.Input)
		if err != nil {
			return nil, err
		}
	}

	if output == nil {
		output = json.RawMessage("{}")
	}
	return output, nil
}

// executeForeach executes body steps for each item in the input array
func (e *BlockGroupExecutor) executeForeach(ctx context.Context, bgCtx *BlockGroupContext) (json.RawMessage, error) {
	ctx, span := tracer.Start(ctx, "block_group.foreach")
	defer span.End()

	// Parse config
	var config domain.ForeachConfig
	if bgCtx.Group.Config != nil {
		if err := json.Unmarshal(bgCtx.Group.Config, &config); err != nil {
			e.logger.Warn("Failed to parse foreach config, using defaults", "error", err)
		}
	}

	// Get items from input
	var items []interface{}
	if config.InputPath != "" {
		var inputData map[string]interface{}
		if err := json.Unmarshal(bgCtx.Input, &inputData); err != nil {
			return nil, fmt.Errorf("failed to parse input: %w", err)
		}
		resolved, err := e.evaluator.ResolveValue(config.InputPath, inputData)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve input path: %w", err)
		}
		var ok bool
		items, ok = resolved.([]interface{})
		if !ok {
			return nil, fmt.Errorf("input path does not resolve to array")
		}
	} else {
		if err := json.Unmarshal(bgCtx.Input, &items); err != nil {
			return nil, fmt.Errorf("input is not an array: %w", err)
		}
	}

	span.SetAttributes(attribute.Int("item_count", len(items)))

	// Get body steps
	var bodySteps []*domain.Step
	for _, step := range bgCtx.Steps {
		if step.GroupRole == "" || step.GroupRole == string(domain.GroupRoleBody) {
			bodySteps = append(bodySteps, step)
		}
	}

	// Execute for each item
	results := make([]interface{}, 0, len(items))

	if config.Parallel && len(items) > 0 {
		// Parallel execution
		maxWorkers := config.MaxWorkers
		if maxWorkers <= 0 {
			maxWorkers = len(items)
		}

		resultsChan := make(chan struct {
			index  int
			result interface{}
		}, len(items))

		sem := make(chan struct{}, maxWorkers)
		var wg sync.WaitGroup

		for i, item := range items {
			wg.Add(1)
			go func(idx int, itm interface{}) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				iterInput := map[string]interface{}{
					"index":       idx,
					"currentItem": itm,
					"items":       items,
				}
				iterInputJSON, err := json.Marshal(iterInput)
				if err != nil {
					e.logger.Error("Failed to marshal foreach iteration input", "index", idx, "error", err)
					resultsChan <- struct {
						index  int
						result interface{}
					}{idx, nil}
					return
				}

				var lastOutput interface{}
				for _, step := range bodySteps {
					output, err := e.executeStep(ctx, bgCtx.ExecCtx, bgCtx.Graph, step, iterInputJSON)
					if err != nil {
						e.logger.Error("Foreach step failed", "index", idx, "step_id", step.ID, "error", err)
						continue
					}
					if output != nil {
						if err := json.Unmarshal(output, &lastOutput); err != nil {
							e.logger.Warn("Failed to unmarshal foreach step output", "index", idx, "step_id", step.ID, "error", err)
						}
					}
				}

				resultsChan <- struct {
					index  int
					result interface{}
				}{idx, lastOutput}
			}(i, item)
		}

		wg.Wait()
		close(resultsChan)

		// Collect results in order
		resultsMap := make(map[int]interface{})
		for r := range resultsChan {
			resultsMap[r.index] = r.result
		}
		for i := 0; i < len(items); i++ {
			results = append(results, resultsMap[i])
		}
	} else {
		// Sequential execution
		for i, item := range items {
			iterInput := map[string]interface{}{
				"index":       i,
				"currentItem": item,
				"items":       items,
			}
			iterInputJSON, err := json.Marshal(iterInput)
			if err != nil {
				e.logger.Error("Failed to marshal foreach iteration input", "index", i, "error", err)
				results = append(results, nil)
				continue
			}

			var lastOutput interface{}
			for _, step := range bodySteps {
				output, err := e.executeStep(ctx, bgCtx.ExecCtx, bgCtx.Graph, step, iterInputJSON)
				if err != nil {
					e.logger.Error("Foreach step failed", "index", i, "step_id", step.ID, "error", err)
					continue
				}
				if output != nil {
					if err := json.Unmarshal(output, &lastOutput); err != nil {
						e.logger.Warn("Failed to unmarshal foreach step output", "index", i, "step_id", step.ID, "error", err)
					}
				}
			}
			results = append(results, lastOutput)
		}
	}

	output := map[string]interface{}{
		"results":    results,
		"iterations": len(items),
		"completed":  true,
	}

	return json.Marshal(output)
}

// executeWhile executes body steps while condition is true
func (e *BlockGroupExecutor) executeWhile(ctx context.Context, bgCtx *BlockGroupContext) (json.RawMessage, error) {
	ctx, span := tracer.Start(ctx, "block_group.while")
	defer span.End()

	// Parse config
	var config domain.WhileConfig
	if bgCtx.Group.Config != nil {
		if err := json.Unmarshal(bgCtx.Group.Config, &config); err != nil {
			e.logger.Warn("Failed to parse while config, using defaults", "error", err)
		}
	}

	maxIterations := config.MaxIterations
	if maxIterations <= 0 {
		maxIterations = 100
	}

	// Get body steps
	var bodySteps []*domain.Step
	for _, step := range bgCtx.Steps {
		if step.GroupRole == "" || step.GroupRole == string(domain.GroupRoleBody) {
			bodySteps = append(bodySteps, step)
		}
	}

	var results []interface{}
	iterations := 0
	currentInput := bgCtx.Input

	for iterations < maxIterations {
		// For do-while, execute first before checking condition
		if config.DoWhile && iterations == 0 {
			// Execute body
			var lastOutput interface{}
			for _, step := range bodySteps {
				output, err := e.executeStep(ctx, bgCtx.ExecCtx, bgCtx.Graph, step, currentInput)
				if err != nil {
					return nil, err
				}
				if output != nil {
					if err := json.Unmarshal(output, &lastOutput); err != nil {
						e.logger.Warn("Failed to unmarshal while step output", "step_id", step.ID, "error", err)
					}
					currentInput = output
				}
			}
			results = append(results, lastOutput)
			iterations++
		}

		// Check condition
		conditionResult, err := e.evaluator.Evaluate(config.Condition, currentInput)
		if err != nil || !conditionResult {
			break
		}

		// Execute body (for regular while, or subsequent do-while iterations)
		if !config.DoWhile || iterations > 0 {
			var lastOutput interface{}
			for _, step := range bodySteps {
				output, err := e.executeStep(ctx, bgCtx.ExecCtx, bgCtx.Graph, step, currentInput)
				if err != nil {
					return nil, err
				}
				if output != nil {
					if err := json.Unmarshal(output, &lastOutput); err != nil {
						e.logger.Warn("Failed to unmarshal while step output", "step_id", step.ID, "error", err)
					}
					currentInput = output
				}
			}
			results = append(results, lastOutput)
			iterations++
		}
	}

	span.SetAttributes(attribute.Int("iterations", iterations))

	output := map[string]interface{}{
		"results":    results,
		"iterations": iterations,
		"completed":  true,
	}

	return json.Marshal(output)
}
