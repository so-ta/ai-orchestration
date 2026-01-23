package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
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
	Steps    []*domain.Step  // Steps belonging to this group
	Edges    []*domain.Edge  // Edges in the project (for chain building)
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
		"group_name", bgCtx.Group.Name,
		"group_type", bgCtx.Group.Type,
		"step_count", len(bgCtx.Steps),
		"input_raw", string(bgCtx.Input),
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
	case domain.BlockGroupTypeAgent:
		internalOutput, err = e.executeAgent(ctx, bgCtx)
	default:
		err = fmt.Errorf("unknown block group type: %s (valid types: parallel, try_catch, foreach, while, agent)", bgCtx.Group.Type)
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
// The input parameter is used for tool chain execution where the step input
// comes from LLM tool arguments rather than DAG edges
func (e *BlockGroupExecutor) executeStep(ctx context.Context, execCtx *ExecutionContext, graph *Graph, step *domain.Step, input json.RawMessage) (json.RawMessage, error) {
	ctx, span := tracer.Start(ctx, "block_group.step",
		trace.WithAttributes(
			attribute.String("step_id", step.ID.String()),
			attribute.String("step_name", step.Name),
			attribute.String("step_type", string(step.Type)),
		),
	)
	defer span.End()

	// Set tool input override so prepareStepInput uses the provided input
	// instead of resolving input from DAG edges (which don't exist for tool steps)
	if len(input) > 0 {
		execCtx.mu.Lock()
		if execCtx.ToolInputOverride == nil {
			execCtx.ToolInputOverride = make(map[uuid.UUID]json.RawMessage)
		}
		execCtx.ToolInputOverride[step.ID] = input
		execCtx.mu.Unlock()

		// Clean up the override after execution
		defer func() {
			execCtx.mu.Lock()
			delete(execCtx.ToolInputOverride, step.ID)
			execCtx.mu.Unlock()
		}()
	}

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

// executeAgent executes an AI agent with ReAct loop
// Entry point steps (no incoming edges) become tools that the agent can call
// Each tool executes its chain of connected steps
func (e *BlockGroupExecutor) executeAgent(ctx context.Context, bgCtx *BlockGroupContext) (json.RawMessage, error) {
	ctx, span := tracer.Start(ctx, "block_group.agent")
	defer span.End()

	// Parse config
	var config domain.AgentConfig
	if bgCtx.Group.Config != nil {
		if err := json.Unmarshal(bgCtx.Group.Config, &config); err != nil {
			e.logger.Warn("Failed to parse agent config, using defaults", "error", err)
		}
	}

	// Set defaults
	if config.MaxIterations <= 0 {
		config.MaxIterations = 30 // Increased from 10 for complex agent tasks
	}
	if config.Temperature == 0 {
		config.Temperature = 0.7
	}
	if config.ToolChoice == "" {
		config.ToolChoice = "auto"
	}
	if config.MemoryWindow <= 0 {
		config.MemoryWindow = 20
	}

	span.SetAttributes(
		attribute.String("provider", config.Provider),
		attribute.String("model", config.Model),
		attribute.Int("max_iterations", config.MaxIterations),
	)

	// Build tool chains from entry points (steps with no incoming edges)
	chainBuilder := NewChainBuilder(bgCtx.Steps, bgCtx.Edges)

	// Validate entry point count
	if errMsg := chainBuilder.ValidateEntryPointCount(bgCtx.Group.ID, bgCtx.Group.Type); errMsg != "" {
		return nil, fmt.Errorf(errMsg)
	}

	toolChains := chainBuilder.BuildToolChains(bgCtx.Group.ID)
	if len(toolChains) == 0 {
		return nil, fmt.Errorf("agent group requires at least one entry point step")
	}

	span.SetAttributes(attribute.Int("tool_count", len(toolChains)))

	// Build tool map for quick lookup
	toolChainMap := make(map[string]*ToolChain)
	for _, tc := range toolChains {
		toolChainMap[tc.ToolName] = tc
	}

	// Build OpenAI-style tools from tool chains
	tools := e.buildToolsFromChains(toolChains)

	// Parse input for user message
	var inputData map[string]interface{}
	if err := json.Unmarshal(bgCtx.Input, &inputData); err != nil {
		inputData = map[string]interface{}{"message": string(bgCtx.Input)}
	}

	userMessage := ""
	if msg, ok := inputData["message"].(string); ok {
		userMessage = msg
	} else if content, ok := inputData["content"].(string); ok {
		userMessage = content
	} else {
		// Stringify the entire input
		userMessage = string(bgCtx.Input)
	}

	// Build workflow context for enhanced prompting
	workflowContext := e.buildWorkflowContext(bgCtx)
	enhancedPrompt := config.SystemPrompt
	if workflowContext != "" {
		enhancedPrompt = config.SystemPrompt + "\n\n" + workflowContext
	}

	// Build initial messages with system prompt
	messages := []map[string]interface{}{
		{"role": "system", "content": enhancedPrompt},
	}

	// Add conversation history if memory is enabled
	if config.EnableMemory {
		messages = e.appendHistoryMessages(messages, inputData, config.MemoryWindow)
	}

	// Add current user message
	messages = append(messages, map[string]interface{}{
		"role":    "user",
		"content": userMessage,
	})

	// Create LLM service
	llmService := sandbox.NewLLMService(ctx)

	var finalResponse string
	var lastError error

	// ReAct loop
	for iteration := 0; iteration < config.MaxIterations; iteration++ {
		span.AddEvent("agent_iteration", trace.WithAttributes(attribute.Int("iteration", iteration)))

		e.logger.Info("Agent iteration",
			"group_id", bgCtx.Group.ID,
			"iteration", iteration,
			"message_count", len(messages),
		)

		// Emit thinking event
		e.emitAgentEvent(bgCtx, EventThinking, ThinkingData{
			Iteration: iteration,
			Content:   "推論中...",
		})

		// Build LLM request
		llmRequest := map[string]interface{}{
			"messages":    messages,
			"temperature": config.Temperature,
			"max_tokens":  4096,
		}

		if len(tools) > 0 {
			llmRequest["tools"] = tools
			llmRequest["tool_choice"] = config.ToolChoice
		}

		// Call LLM
		response, err := llmService.Chat(config.Provider, config.Model, llmRequest)
		if err != nil {
			lastError = err
			e.logger.Error("Agent LLM call failed", "iteration", iteration, "error", err)
			break
		}

		content, _ := response["content"].(string)
		toolCalls, hasToolCalls := response["tool_calls"].([]map[string]interface{})

		if !hasToolCalls || len(toolCalls) == 0 {
			// No tool calls - final response
			finalResponse = content
			e.logger.Info("Agent completed with final response",
				"group_id", bgCtx.Group.ID,
				"iterations", iteration+1,
			)
			break
		}

		// Add assistant message with tool calls
		assistantMsg := map[string]interface{}{
			"role":       "assistant",
			"content":    content,
			"tool_calls": toolCalls,
		}
		messages = append(messages, assistantMsg)

		// Execute each tool call
		for _, toolCall := range toolCalls {
			toolCallID, _ := toolCall["id"].(string)
			fn, _ := toolCall["function"].(map[string]interface{})
			toolName, _ := fn["name"].(string)
			argsStr, _ := fn["arguments"].(string)

			e.logger.Info("Agent executing tool",
				"group_id", bgCtx.Group.ID,
				"tool_name", toolName,
				"tool_call_id", toolCallID,
			)

			// Parse arguments
			var toolArgs map[string]interface{}
			if err := json.Unmarshal([]byte(argsStr), &toolArgs); err != nil {
				toolArgs = map[string]interface{}{}
			}

			// Emit tool_call event
			argsJSON, _ := json.Marshal(toolArgs)
			e.emitAgentEvent(bgCtx, EventToolCall, ToolCallData{
				ToolName:   toolName,
				ToolCallID: toolCallID,
				Arguments:  argsJSON,
			})

			// Find and execute the corresponding tool chain
			var toolResult interface{}
			var isError bool
			if toolChain, ok := toolChainMap[toolName]; ok {
				// Validate tool arguments against schema before execution
				if validationErr := validateToolArgs(toolArgs, toolChain.InputSchema); validationErr != nil {
					e.logger.Warn("Tool argument validation failed",
						"tool_name", toolName,
						"missing_fields", validationErr.MissingFields,
					)
					toolResult = map[string]interface{}{
						"error":          validationErr.Message,
						"missing_fields": validationErr.MissingFields,
						"suggestion":     "Please provide all required parameters and retry the tool call.",
					}
					isError = true
				} else {
					toolInput, _ := json.Marshal(toolArgs)
					output, err := e.executeToolChain(ctx, bgCtx, toolChain, toolInput)
					if err != nil {
						toolResult = map[string]interface{}{
							"error": err.Error(),
						}
						isError = true
					} else {
						if err := json.Unmarshal(output, &toolResult); err != nil {
							toolResult = string(output)
						}
						// Check if the tool output contains an error field
						// This handles cases where the tool execution succeeded but returned an error message
						// (e.g., validation errors from create_step like "project_id, name, and type are required")
						if resultMap, ok := toolResult.(map[string]interface{}); ok {
							if _, hasError := resultMap["error"]; hasError {
								isError = true
							}
						}
					}
				}
			} else {
				toolResult = map[string]interface{}{
					"error": fmt.Sprintf("Tool not found: %s", toolName),
				}
				isError = true
			}

			// Add tool result to messages
			toolResultStr, _ := json.Marshal(toolResult)

			// Emit tool_result event
			e.emitAgentEvent(bgCtx, EventToolResult, ToolResultData{
				ToolName:   toolName,
				ToolCallID: toolCallID,
				Result:     toolResultStr,
				IsError:    isError,
			})

			messages = append(messages, map[string]interface{}{
				"role":         "tool",
				"tool_call_id": toolCallID,
				"content":      string(toolResultStr),
			})
		}
	}

	// Build output
	if lastError != nil {
		// Return error via error port
		errorOutput := map[string]interface{}{
			"__error":    true,
			"__port":     "error",
			"error":      lastError.Error(),
			"iterations": config.MaxIterations,
		}
		return json.Marshal(errorOutput)
	}

	if finalResponse == "" {
		finalResponse = "Agent reached maximum iterations without final response."
	}

	// Emit complete event
	e.emitAgentEvent(bgCtx, EventComplete, CompleteData{
		Response:   finalResponse,
		Iterations: len(messages) / 2,
	})

	output := map[string]interface{}{
		"response":      finalResponse,
		"iterations":    len(messages) / 2, // Rough estimate of iterations
		"message_count": len(messages),
	}

	return json.Marshal(output)
}

// emitAgentEvent emits an event for agent execution if an emitter is configured
func (e *BlockGroupExecutor) emitAgentEvent(bgCtx *BlockGroupContext, eventType ExecutionEventType, data interface{}) {
	if bgCtx.ExecCtx == nil || bgCtx.ExecCtx.EventEmitter == nil {
		return
	}
	bgCtx.ExecCtx.EventEmitter.Emit(NewExecutionEvent(bgCtx.ExecCtx.Run.ID, eventType, data))
}

// appendHistoryMessages appends conversation history messages to the message array
// This enables memory for agents by including previous user and assistant messages
// History is limited by memoryWindow (0 = no limit)
func (e *BlockGroupExecutor) appendHistoryMessages(
	messages []map[string]interface{},
	inputData map[string]interface{},
	memoryWindow int,
) []map[string]interface{} {
	history, ok := inputData["history"].([]interface{})
	if !ok || len(history) == 0 {
		return messages
	}

	// Apply memory window limit
	start := 0
	if memoryWindow > 0 && len(history) > memoryWindow {
		start = len(history) - memoryWindow
	}

	// Append history messages
	for _, item := range history[start:] {
		msg, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		role, _ := msg["role"].(string)
		content, _ := msg["content"].(string)

		// Only include user and assistant messages
		if role == "user" || role == "assistant" {
			if content != "" {
				messages = append(messages, map[string]interface{}{
					"role":    role,
					"content": content,
				})
			}
		}
	}

	return messages
}

// buildWorkflowContext builds a context string for the agent with current workflow information
// This helps the agent understand the current state of the workflow being edited
func (e *BlockGroupExecutor) buildWorkflowContext(bgCtx *BlockGroupContext) string {
	// Extract project_id from input
	var inputData map[string]interface{}
	if err := json.Unmarshal(bgCtx.Input, &inputData); err != nil {
		return ""
	}

	projectID, ok := inputData["project_id"].(string)
	if !ok || projectID == "" {
		// Also check workflow_id as an alias
		projectID, ok = inputData["workflow_id"].(string)
		if !ok || projectID == "" {
			return ""
		}
	}

	var context strings.Builder
	context.WriteString("## Current Workflow Context\n\n")
	context.WriteString(fmt.Sprintf("**Project ID:** %s\n\n", projectID))

	// Filter and list steps in this project
	var projectSteps []domain.Step
	for _, step := range bgCtx.Graph.Steps {
		if step.ProjectID.String() == projectID {
			projectSteps = append(projectSteps, step)
		}
	}

	if len(projectSteps) > 0 {
		context.WriteString("### Existing Steps:\n")
		for _, step := range projectSteps {
			context.WriteString(fmt.Sprintf("- **%s** (type: %s, id: %s)\n",
				step.Name, step.Type, step.ID.String()))
		}
		context.WriteString("\n")
	} else {
		context.WriteString("### Existing Steps:\nNo steps in this workflow yet.\n\n")
	}

	// Filter and list edges in this project
	var projectEdges []domain.Edge
	for _, edge := range bgCtx.Graph.AllEdges {
		if edge.ProjectID.String() == projectID {
			projectEdges = append(projectEdges, edge)
		}
	}

	if len(projectEdges) > 0 {
		context.WriteString("### Existing Connections:\n")
		// Build step ID to name map for readable edge display
		stepNameMap := make(map[string]string)
		for _, step := range projectSteps {
			stepNameMap[step.ID.String()] = step.Name
		}
		for _, edge := range projectEdges {
			sourceName := stepNameMap[edge.SourceStepID.String()]
			if sourceName == "" {
				sourceName = edge.SourceStepID.String()
			}
			targetName := stepNameMap[edge.TargetStepID.String()]
			if targetName == "" {
				targetName = edge.TargetStepID.String()
			}
			context.WriteString(fmt.Sprintf("- %s -> %s\n", sourceName, targetName))
		}
		context.WriteString("\n")
	} else {
		context.WriteString("### Existing Connections:\nNo connections yet.\n\n")
	}

	return context.String()
}

// executeToolChain executes a chain of steps and returns the final output
func (e *BlockGroupExecutor) executeToolChain(ctx context.Context, bgCtx *BlockGroupContext, toolChain *ToolChain, input json.RawMessage) (json.RawMessage, error) {
	currentInput := input

	for _, step := range toolChain.Chain {
		output, err := e.executeStep(ctx, bgCtx.ExecCtx, bgCtx.Graph, step, currentInput)
		if err != nil {
			return nil, fmt.Errorf("step %s failed: %w", step.Name, err)
		}
		currentInput = output
	}

	return currentInput, nil
}

// buildToolsFromChains builds OpenAI-style tool definitions from tool chains
func (e *BlockGroupExecutor) buildToolsFromChains(toolChains []*ToolChain) []interface{} {
	var tools []interface{}

	for _, tc := range toolChains {
		// Get parameters schema from entry point or default to empty object
		parameters := map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}
		if tc.InputSchema != nil {
			if schemaMap, ok := tc.InputSchema.(map[string]interface{}); ok {
				parameters = schemaMap
			} else if schemaBytes, ok := tc.InputSchema.(json.RawMessage); ok {
				var schemaMap map[string]interface{}
				if err := json.Unmarshal(schemaBytes, &schemaMap); err == nil {
					parameters = schemaMap
				}
			}
		}

		tool := map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        tc.ToolName,
				"description": tc.Description,
				"parameters":  parameters,
			},
		}
		tools = append(tools, tool)
	}

	return tools
}

// buildToolsFromSteps builds OpenAI-style tool definitions from child steps
func (e *BlockGroupExecutor) buildToolsFromSteps(steps []*domain.Step, bgCtx *BlockGroupContext) []interface{} {
	var tools []interface{}

	for _, step := range steps {
		// Get tool name (prefer ToolName, fallback to step Name)
		toolName := step.Name
		if step.ToolName != nil && *step.ToolName != "" {
			toolName = *step.ToolName
		}

		// Get description from ToolDescription, step config, or use step name
		description := step.Name
		if step.ToolDescription != nil && *step.ToolDescription != "" {
			description = *step.ToolDescription
		} else {
			var config map[string]interface{}
			if step.Config != nil {
				json.Unmarshal(step.Config, &config)
			}
			if desc, ok := config["description"].(string); ok && desc != "" {
				description = desc
			}
		}

		// Get parameters schema from ToolInputSchema, step config, or default to empty object
		parameters := map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}
		if len(step.ToolInputSchema) > 0 {
			var schemaMap map[string]interface{}
			if err := json.Unmarshal(step.ToolInputSchema, &schemaMap); err == nil {
				parameters = schemaMap
			}
		} else {
			var config map[string]interface{}
			if step.Config != nil {
				json.Unmarshal(step.Config, &config)
			}
			if params, ok := config["input_schema"].(map[string]interface{}); ok {
				parameters = params
			} else if params, ok := config["parameters"].(map[string]interface{}); ok {
				parameters = params
			}
		}

		tool := map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        toolName,
				"description": description,
				"parameters":  parameters,
			},
		}
		tools = append(tools, tool)
	}

	return tools
}

// SchemaValidationError represents an error from validating tool arguments against a schema
type SchemaValidationError struct {
	Message       string   `json:"message"`
	MissingFields []string `json:"missing_fields,omitempty"`
	InvalidFields []string `json:"invalid_fields,omitempty"`
}

func (e *SchemaValidationError) Error() string {
	return e.Message
}

// validateToolArgs validates tool arguments against a JSON Schema
// Returns nil if validation passes, or a SchemaValidationError if validation fails
func validateToolArgs(args map[string]interface{}, schema interface{}) *SchemaValidationError {
	if schema == nil {
		return nil
	}

	// Parse schema
	var schemaMap map[string]interface{}
	switch s := schema.(type) {
	case map[string]interface{}:
		schemaMap = s
	case json.RawMessage:
		if err := json.Unmarshal(s, &schemaMap); err != nil {
			return nil // Can't parse schema, skip validation
		}
	default:
		return nil // Unknown schema type, skip validation
	}

	// Check required fields
	var missingFields []string
	if required, ok := schemaMap["required"].([]interface{}); ok {
		for _, req := range required {
			fieldName, ok := req.(string)
			if !ok {
				continue
			}
			if _, exists := args[fieldName]; !exists {
				missingFields = append(missingFields, fieldName)
			} else if args[fieldName] == nil || args[fieldName] == "" {
				// Also treat null or empty string as missing for required fields
				missingFields = append(missingFields, fieldName)
			}
		}
	}

	if len(missingFields) > 0 {
		return &SchemaValidationError{
			Message:       fmt.Sprintf("Missing required fields: %s", strings.Join(missingFields, ", ")),
			MissingFields: missingFields,
		}
	}

	return nil
}
