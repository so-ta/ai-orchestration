package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/souta/ai-orchestration/internal/adapter"
	"github.com/souta/ai-orchestration/internal/block/sandbox"
	"github.com/souta/ai-orchestration/internal/domain"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// BlockGroupExecutor handles execution of block groups (control flow constructs)
// Redesigned to support 4 group types: parallel, try_catch, foreach, while
// All groups now use body-only structure with pre_process/post_process for I/O transformation
type BlockGroupExecutor struct {
	registry  *adapter.Registry
	logger    *slog.Logger
	evaluator *ConditionEvaluator
	executor  *Executor         // Reference to main executor for step execution
	sandbox   *sandbox.Sandbox  // Sandbox for executing pre/post_process JS
}

// NewBlockGroupExecutor creates a new block group executor
func NewBlockGroupExecutor(registry *adapter.Registry, logger *slog.Logger, executor *Executor) *BlockGroupExecutor {
	return &BlockGroupExecutor{
		registry:  registry,
		logger:    logger,
		evaluator: NewConditionEvaluator(),
		executor:  executor,
		sandbox:   sandbox.New(sandbox.DefaultConfig()),
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

// BlockGroupResult represents the result of a block group execution
type BlockGroupResult struct {
	Output json.RawMessage `json:"output"`
	Port   string          `json:"port"` // "out" or "error"
	Error  error           `json:"-"`
}

// ExecuteGroup executes a block group based on its type
// Supports 4 types: parallel, try_catch, foreach, while
// Removed: if_else, switch_case (use condition/switch blocks instead)
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

	// 1. Run pre_process to transform external input to internal input
	internalInput, err := e.runPreProcess(ctx, bgCtx.Group, bgCtx.Input)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "pre_process failed")
		return nil, fmt.Errorf("pre_process failed: %w", err)
	}

	// Update context with transformed input
	bgCtx.Input = internalInput

	// 2. Execute group based on type
	var internalOutput json.RawMessage
	switch bgCtx.Group.Type {
	case domain.BlockGroupTypeParallel:
		internalOutput, err = e.executeParallel(ctx, bgCtx)
	case domain.BlockGroupTypeTryCatch:
		internalOutput, err = e.executeTryCatch(ctx, bgCtx)
	case domain.BlockGroupTypeForeach:
		internalOutput, err = e.executeForeach(ctx, bgCtx)
	case domain.BlockGroupTypeWhile:
		internalOutput, err = e.executeWhile(ctx, bgCtx)
	default:
		err = fmt.Errorf("unknown block group type: %s (valid types: parallel, try_catch, foreach, while)", bgCtx.Group.Type)
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// 3. Run post_process to transform internal output to external output
	externalOutput, err := e.runPostProcess(ctx, bgCtx.Group, internalOutput)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "post_process failed")
		return nil, fmt.Errorf("post_process failed: %w", err)
	}

	span.SetStatus(codes.Ok, "block group completed")
	return externalOutput, nil
}

// runPreProcess executes the pre_process JavaScript to transform input
func (e *BlockGroupExecutor) runPreProcess(ctx context.Context, group *domain.BlockGroup, input json.RawMessage) (json.RawMessage, error) {
	if group.PreProcess == nil || *group.PreProcess == "" {
		return input, nil // No transformation
	}

	ctx, span := tracer.Start(ctx, "block_group.pre_process")
	defer span.End()

	// Parse input into map
	var inputMap map[string]interface{}
	if err := json.Unmarshal(input, &inputMap); err != nil {
		// If input is not an object, wrap it
		inputMap = map[string]interface{}{"value": json.RawMessage(input)}
	}

	// Create sandbox context with HTTP client
	sandboxCtx := &sandbox.ExecutionContext{
		HTTP: sandbox.NewHTTPClient(30 * time.Second),
		Logger: func(args ...interface{}) {
			e.logger.Info("pre_process log", "group_id", group.ID, "message", fmt.Sprint(args...))
		},
	}

	// Wrap code to return result
	wrappedCode := wrapTransformCode(*group.PreProcess)

	result, err := e.sandbox.Execute(ctx, wrappedCode, inputMap, sandboxCtx)
	if err != nil {
		return nil, err
	}

	return json.Marshal(result)
}

// runPostProcess executes the post_process JavaScript to transform output
func (e *BlockGroupExecutor) runPostProcess(ctx context.Context, group *domain.BlockGroup, output json.RawMessage) (json.RawMessage, error) {
	if group.PostProcess == nil || *group.PostProcess == "" {
		return output, nil // No transformation
	}

	ctx, span := tracer.Start(ctx, "block_group.post_process")
	defer span.End()

	// Parse output into map
	var outputMap map[string]interface{}
	if err := json.Unmarshal(output, &outputMap); err != nil {
		// If output is not an object, wrap it
		outputMap = map[string]interface{}{"value": json.RawMessage(output)}
	}

	// Create sandbox context with HTTP client
	sandboxCtx := &sandbox.ExecutionContext{
		HTTP: sandbox.NewHTTPClient(30 * time.Second),
		Logger: func(args ...interface{}) {
			e.logger.Info("post_process log", "group_id", group.ID, "message", fmt.Sprint(args...))
		},
	}

	// Wrap code to return result
	wrappedCode := wrapTransformCode(*group.PostProcess)

	result, err := e.sandbox.Execute(ctx, wrappedCode, outputMap, sandboxCtx)
	if err != nil {
		return nil, err
	}

	return json.Marshal(result)
}

// wrapTransformCode wraps pre/post_process code to ensure it returns a value
func wrapTransformCode(code string) string {
	return fmt.Sprintf(`
(function() {
	var result = (function(input) {
		%s
	})(input);
	return result !== undefined ? result : input;
})()
`, code)
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

// executeTryCatch executes body steps with retry support
// Simplified: no more try/catch/finally roles, just body
// Error handling is done via output port (error info returned in output)
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

	// Get body steps only (all steps should have body role now)
	var bodySteps []*domain.Step
	for _, step := range bgCtx.Steps {
		if step.GroupRole == "" || step.GroupRole == string(domain.GroupRoleBody) {
			bodySteps = append(bodySteps, step)
		}
	}

	span.SetAttributes(attribute.Int("body_step_count", len(bodySteps)))
	span.SetAttributes(attribute.Int("retry_count", config.RetryCount))

	var output json.RawMessage
	var lastError error

	// Execute body with retry support
	for attempt := 0; attempt <= config.RetryCount; attempt++ {
		if attempt > 0 {
			e.logger.Info("Retrying try-catch body",
				"attempt", attempt,
				"max_retries", config.RetryCount,
			)
			span.AddEvent("retry_attempt", trace.WithAttributes(attribute.Int("attempt", attempt)))

			// Wait before retry
			if config.RetryDelay > 0 {
				time.Sleep(time.Duration(config.RetryDelay) * time.Millisecond)
			}
		}

		// Execute all body steps
		lastError = nil
		for _, step := range bodySteps {
			var err error
			output, err = e.executeStep(ctx, bgCtx.ExecCtx, bgCtx.Graph, step, bgCtx.Input)
			if err != nil {
				lastError = err
				break
			}
		}

		// If successful, return
		if lastError == nil {
			span.SetAttributes(attribute.Int("successful_attempt", attempt))
			return output, nil
		}
	}

	// All retries exhausted, return error info in output
	e.logger.Warn("Try-catch exhausted all retries",
		"group_id", bgCtx.Group.ID,
		"retries", config.RetryCount,
		"error", lastError,
	)
	span.AddEvent("all_retries_exhausted")

	// Return error information that can be routed via error port
	errorOutput := map[string]interface{}{
		"__error":  true,
		"__port":   "error",
		"error":    lastError.Error(),
		"input":    json.RawMessage(bgCtx.Input),
		"attempts": config.RetryCount + 1,
	}

	return json.Marshal(errorOutput)
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
