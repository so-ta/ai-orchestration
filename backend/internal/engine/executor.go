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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/souta/ai-orchestration/internal/adapter"
	"github.com/souta/ai-orchestration/internal/block/sandbox"
	"github.com/souta/ai-orchestration/internal/domain"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("ai-orchestration/engine")

// BlockDefinitionGetter is an interface for getting block definitions
type BlockDefinitionGetter interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.BlockDefinition, error)
	GetBySlug(ctx context.Context, tenantID *uuid.UUID, slug string) (*domain.BlockDefinition, error)
}

// Executor executes a workflow DAG
type Executor struct {
	registry      *adapter.Registry
	logger        *slog.Logger
	evaluator     *ConditionEvaluator
	sandbox       *sandbox.Sandbox
	usageRecorder *UsageRecorder
	pool          *pgxpool.Pool           // Database pool for sandbox services
	blockDefRepo  BlockDefinitionGetter   // Repository for custom block definitions
}

// ExecutorOption is a functional option for Executor
type ExecutorOption func(*Executor)

// WithUsageRecorder sets the usage recorder for the executor
func WithUsageRecorder(recorder *UsageRecorder) ExecutorOption {
	return func(e *Executor) {
		e.usageRecorder = recorder
	}
}

// WithDatabase sets the database pool for sandbox services
func WithDatabase(pool *pgxpool.Pool) ExecutorOption {
	return func(e *Executor) {
		e.pool = pool
	}
}

// WithBlockDefinitionRepository sets the block definition repository for custom block execution
func WithBlockDefinitionRepository(repo BlockDefinitionGetter) ExecutorOption {
	return func(e *Executor) {
		e.blockDefRepo = repo
	}
}

// NewExecutor creates a new executor
func NewExecutor(registry *adapter.Registry, logger *slog.Logger, opts ...ExecutorOption) *Executor {
	e := &Executor{
		registry:  registry,
		logger:    logger,
		evaluator: NewConditionEvaluator(),
		sandbox:   sandbox.New(sandbox.DefaultConfig()),
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// ExecutionContext holds the context for a workflow execution
type ExecutionContext struct {
	Run             *domain.Run
	Definition      *domain.WorkflowDefinition
	StepRuns        map[uuid.UUID]*domain.StepRun
	StepData        map[uuid.UUID]json.RawMessage // step outputs
	StepOutputPorts map[uuid.UUID]string          // output port used by each step (for port-based routing)
	GroupData       map[uuid.UUID]json.RawMessage // block group outputs
	InjectedOutputs map[string]json.RawMessage    // pre-injected outputs for partial execution
	sequenceCounter int                           // counter for step execution order within an attempt
	mu              sync.RWMutex
}

// NewExecutionContext creates a new execution context
func NewExecutionContext(run *domain.Run, def *domain.WorkflowDefinition) *ExecutionContext {
	return &ExecutionContext{
		Run:             run,
		Definition:      def,
		StepRuns:        make(map[uuid.UUID]*domain.StepRun),
		StepData:        make(map[uuid.UUID]json.RawMessage),
		StepOutputPorts: make(map[uuid.UUID]string),
		GroupData:       make(map[uuid.UUID]json.RawMessage),
		InjectedOutputs: make(map[string]json.RawMessage),
	}
}

// NextSequenceNumber returns the next sequence number for step execution order
func (ec *ExecutionContext) NextSequenceNumber() int {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.sequenceCounter++
	return ec.sequenceCounter
}

// SetSequenceCounter sets the sequence counter (used for resuming execution)
func (ec *ExecutionContext) SetSequenceCounter(value int) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.sequenceCounter = value
}

// InjectPreviousOutputs injects outputs from a previous run for partial execution
func (ec *ExecutionContext) InjectPreviousOutputs(outputs map[string]json.RawMessage) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	ec.InjectedOutputs = outputs
	// Also add to StepData so they're available during execution
	for stepIDStr, output := range outputs {
		if stepID, err := uuid.Parse(stepIDStr); err == nil {
			ec.StepData[stepID] = output
		}
	}
}

// ExecuteSingleStep executes only one step with the given input
func (e *Executor) ExecuteSingleStep(ctx context.Context, execCtx *ExecutionContext, stepID uuid.UUID, input json.RawMessage) (*domain.StepRun, error) {
	ctx, span := tracer.Start(ctx, "workflow.execute_single_step",
		trace.WithAttributes(
			attribute.String("run_id", execCtx.Run.ID.String()),
			attribute.String("step_id", stepID.String()),
		),
	)
	defer span.End()

	// Find the step in the definition
	var targetStep *domain.Step
	for i := range execCtx.Definition.Steps {
		if execCtx.Definition.Steps[i].ID == stepID {
			targetStep = &execCtx.Definition.Steps[i]
			break
		}
	}
	if targetStep == nil {
		err := fmt.Errorf("step not found: %s", stepID)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	e.logger.Info("Executing single step",
		"run_id", execCtx.Run.ID,
		"step_id", stepID,
		"step_name", targetStep.Name,
	)

	// Create step run with sequence number
	seqNum := execCtx.NextSequenceNumber()
	stepRun := domain.NewStepRun(execCtx.Run.TenantID, execCtx.Run.ID, targetStep.ID, targetStep.Name, seqNum)

	execCtx.mu.Lock()
	execCtx.StepRuns[targetStep.ID] = stepRun
	execCtx.mu.Unlock()

	// Use provided input or prepare from context
	var stepInput json.RawMessage
	if input != nil && len(input) > 0 {
		stepInput = input
	} else {
		var err error
		stepInput, err = e.prepareStepInput(execCtx, *targetStep)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare step input: %w", err)
		}
	}
	stepRun.Start(stepInput)

	// Execute step using unified dispatch
	output, err := e.dispatchStepExecution(ctx, execCtx, *targetStep, stepRun, stepInput)

	if err != nil {
		stepRun.Fail(err.Error())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		e.logger.Error("Single step execution failed",
			"run_id", execCtx.Run.ID,
			"step_id", stepID,
			"error", err,
		)
		return stepRun, err
	}

	stepRun.Complete(output)

	execCtx.mu.Lock()
	execCtx.StepData[targetStep.ID] = output
	execCtx.mu.Unlock()

	span.SetStatus(codes.Ok, "single step completed")
	e.logger.Info("Single step completed",
		"run_id", execCtx.Run.ID,
		"step_id", stepID,
	)

	return stepRun, nil
}

// ExecuteFromStep executes from a specific step through all downstream steps
func (e *Executor) ExecuteFromStep(ctx context.Context, execCtx *ExecutionContext, startStepID uuid.UUID, startInput json.RawMessage) error {
	ctx, span := tracer.Start(ctx, "workflow.execute_from_step",
		trace.WithAttributes(
			attribute.String("run_id", execCtx.Run.ID.String()),
			attribute.String("start_step_id", startStepID.String()),
		),
	)
	defer span.End()

	// Verify the starting step exists
	var found bool
	for i := range execCtx.Definition.Steps {
		if execCtx.Definition.Steps[i].ID == startStepID {
			found = true
			break
		}
	}
	if !found {
		err := fmt.Errorf("start step not found: %s", startStepID)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	e.logger.Info("Executing from step",
		"run_id", execCtx.Run.ID,
		"start_step_id", startStepID,
	)

	// Build execution graph
	graph := e.buildGraph(execCtx.Definition)

	// If custom input provided for start step, store it
	if startInput != nil && len(startInput) > 0 {
		execCtx.mu.Lock()
		// Store the custom input to be used when the start step runs
		execCtx.StepData[startStepID] = startInput
		execCtx.mu.Unlock()
	}

	// Execute from start step
	if err := e.executeNodes(ctx, execCtx, graph, []uuid.UUID{startStepID}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "execution from step completed")
	return nil
}

// dispatchStepExecution routes step execution to the appropriate handler based on step type.
// This is the central dispatch point for all step type execution logic.
func (e *Executor) dispatchStepExecution(ctx context.Context, execCtx *ExecutionContext, step domain.Step, stepRun *domain.StepRun, input json.RawMessage) (json.RawMessage, error) {
	switch step.Type {
	case domain.StepTypeStart:
		return e.executeStartStep(ctx, step, input)
	case domain.StepTypeTool:
		return e.executeToolStep(ctx, execCtx, step, stepRun, input)
	case domain.StepTypeLLM:
		return e.executeLLMStep(ctx, execCtx, step, stepRun, input)
	case domain.StepTypeCondition:
		return e.executeConditionStep(ctx, execCtx, step, input)
	case domain.StepTypeMap:
		return e.executeMapStep(ctx, step, input)
	case domain.StepTypeWait:
		return e.executeWaitStep(ctx, step, input)
	case domain.StepTypeFunction:
		return e.executeFunctionStep(ctx, execCtx, step, input)
	case domain.StepTypeRouter:
		return e.executeRouterStep(ctx, step, input)
	case domain.StepTypeHumanInLoop:
		return e.executeHumanInLoopStep(ctx, execCtx, step, input)
	case domain.StepTypeSwitch:
		return e.executeSwitchStep(ctx, execCtx, step, input)
	case domain.StepTypeFilter:
		return e.executeFilterStep(ctx, step, input)
	case domain.StepTypeSplit:
		return e.executeSplitStep(ctx, step, input)
	case domain.StepTypeAggregate:
		return e.executeAggregateStep(ctx, step, input)
	case domain.StepTypeError:
		return e.executeErrorStep(ctx, step, input)
	case domain.StepTypeNote:
		return e.executeNoteStep(ctx, step, input)
	case domain.StepTypeLog:
		return e.executeLogStep(ctx, step, input)
	case domain.StepTypeSubflow:
		// Subflow not yet implemented - return error to ensure workflow fails explicitly.
		// BREAKING CHANGE: Previously passed through input silently. Now returns error
		// to prevent unintended success when subflow execution is expected but not available.
		// TODO: Implement subflow execution when workflow nesting feature is added
		e.logger.Error("Subflow step type is not yet implemented",
			"step_id", step.ID,
			"step_name", step.Name,
		)
		return nil, fmt.Errorf("subflow step type is not yet implemented: step %s (%s)", step.Name, step.ID)
	default:
		return e.executeCustomOrPassthrough(ctx, execCtx, step, input)
	}
}

// executeCustomOrPassthrough handles custom block execution or passes through for unknown types.
func (e *Executor) executeCustomOrPassthrough(ctx context.Context, execCtx *ExecutionContext, step domain.Step, input json.RawMessage) (json.RawMessage, error) {
	// First check by BlockDefinitionID
	if step.BlockDefinitionID != nil {
		return e.executeCustomBlockStep(ctx, execCtx, step, input)
	}
	// Then try to find block definition by step type (slug)
	if e.blockDefRepo != nil {
		return e.executeCustomBlockStepBySlug(ctx, execCtx, step, input)
	}
	// For unimplemented types without block definition, log warning and pass through
	e.logger.Warn("Unknown step type without block definition, passing through input",
		"step_id", step.ID,
		"step_name", step.Name,
		"step_type", step.Type,
	)
	return input, nil
}

// Execute executes the workflow
func (e *Executor) Execute(ctx context.Context, execCtx *ExecutionContext) error {
	ctx, span := tracer.Start(ctx, "workflow.execute",
		trace.WithAttributes(
			attribute.String("run_id", execCtx.Run.ID.String()),
			attribute.String("workflow_id", execCtx.Run.WorkflowID.String()),
			attribute.String("triggered_by", string(execCtx.Run.TriggeredBy)),
		),
	)
	defer span.End()

	e.logger.Info("Starting workflow execution",
		"run_id", execCtx.Run.ID,
		"workflow_id", execCtx.Run.WorkflowID,
	)

	// Build execution graph
	graph := e.buildGraph(execCtx.Definition)

	// Find start nodes (nodes with no incoming edges)
	startNodes := e.findStartNodes(graph)
	if len(startNodes) == 0 {
		err := fmt.Errorf("no start nodes found in workflow")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetAttributes(attribute.Int("start_node_count", len(startNodes)))

	// Execute from start nodes
	if err := e.executeNodes(ctx, execCtx, graph, startNodes); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "workflow completed")
	return nil
}

// Graph represents the execution graph
type Graph struct {
	Steps       map[uuid.UUID]domain.Step
	BlockGroups map[uuid.UUID]domain.BlockGroup
	InEdges     map[uuid.UUID][]domain.Edge // incoming edges to steps
	OutEdges    map[uuid.UUID][]domain.Edge // outgoing edges from steps
	GroupInEdges  map[uuid.UUID][]domain.Edge // incoming edges to groups
	GroupOutEdges map[uuid.UUID][]domain.Edge // outgoing edges from groups
}

func (e *Executor) buildGraph(def *domain.WorkflowDefinition) *Graph {
	graph := &Graph{
		Steps:         make(map[uuid.UUID]domain.Step),
		BlockGroups:   make(map[uuid.UUID]domain.BlockGroup),
		InEdges:       make(map[uuid.UUID][]domain.Edge),
		OutEdges:      make(map[uuid.UUID][]domain.Edge),
		GroupInEdges:  make(map[uuid.UUID][]domain.Edge),
		GroupOutEdges: make(map[uuid.UUID][]domain.Edge),
	}

	for _, step := range def.Steps {
		graph.Steps[step.ID] = step
	}

	for _, group := range def.BlockGroups {
		graph.BlockGroups[group.ID] = group
	}

	// Process all edges including group edges
	for _, edge := range def.Edges {
		// Step-to-step edge
		if edge.SourceStepID != nil && edge.TargetStepID != nil {
			graph.InEdges[*edge.TargetStepID] = append(graph.InEdges[*edge.TargetStepID], edge)
			graph.OutEdges[*edge.SourceStepID] = append(graph.OutEdges[*edge.SourceStepID], edge)
		}
		// Step-to-group edge
		if edge.SourceStepID != nil && edge.TargetBlockGroupID != nil {
			graph.OutEdges[*edge.SourceStepID] = append(graph.OutEdges[*edge.SourceStepID], edge)
			graph.GroupInEdges[*edge.TargetBlockGroupID] = append(graph.GroupInEdges[*edge.TargetBlockGroupID], edge)
		}
		// Group-to-step edge
		if edge.SourceBlockGroupID != nil && edge.TargetStepID != nil {
			graph.GroupOutEdges[*edge.SourceBlockGroupID] = append(graph.GroupOutEdges[*edge.SourceBlockGroupID], edge)
			graph.InEdges[*edge.TargetStepID] = append(graph.InEdges[*edge.TargetStepID], edge)
		}
		// Group-to-group edge
		if edge.SourceBlockGroupID != nil && edge.TargetBlockGroupID != nil {
			graph.GroupOutEdges[*edge.SourceBlockGroupID] = append(graph.GroupOutEdges[*edge.SourceBlockGroupID], edge)
			graph.GroupInEdges[*edge.TargetBlockGroupID] = append(graph.GroupInEdges[*edge.TargetBlockGroupID], edge)
		}
	}

	return graph
}

func (e *Executor) findStartNodes(graph *Graph) []uuid.UUID {
	var startNodes []uuid.UUID
	for stepID, step := range graph.Steps {
		// Only consider nodes of type "start" as entry points
		// This prevents disconnected nodes from being executed
		if step.Type == domain.StepTypeStart {
			startNodes = append(startNodes, stepID)
		}
	}
	return startNodes
}

func (e *Executor) executeNodes(ctx context.Context, execCtx *ExecutionContext, graph *Graph, nodeIDs []uuid.UUID) error {
	// Track completed nodes and groups
	completed := make(map[uuid.UUID]bool)
	completedGroups := make(map[uuid.UUID]bool)
	var mu sync.Mutex

	// Use a WaitGroup to wait for parallel executions
	var wg sync.WaitGroup
	errChan := make(chan error, len(nodeIDs))

	for _, nodeID := range nodeIDs {
		wg.Add(1)
		go func(id uuid.UUID) {
			defer wg.Done()

			// Execute this node
			if err := e.executeNode(ctx, execCtx, graph, id); err != nil {
				errChan <- err
				return
			}

			mu.Lock()
			completed[id] = true
			mu.Unlock()

			// Check for group edges and execute groups
			if err := e.executeNextGroups(ctx, execCtx, graph, id, completed, completedGroups, &mu); err != nil {
				errChan <- err
				return
			}

			// Find next step nodes to execute
			nextNodes := e.findNextNodes(ctx, execCtx, graph, id, completed, completedGroups, &mu)
			if len(nextNodes) > 0 {
				if err := e.executeNodes(ctx, execCtx, graph, nextNodes); err != nil {
					errChan <- err
				}
			}
		}(nodeID)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// executeNextGroups finds and executes block groups that are next in the execution flow
func (e *Executor) executeNextGroups(ctx context.Context, execCtx *ExecutionContext, graph *Graph, currentStepID uuid.UUID, completed, completedGroups map[uuid.UUID]bool, mu *sync.Mutex) error {
	execCtx.mu.RLock()
	currentOutput := execCtx.StepData[currentStepID]
	currentOutputPort := execCtx.StepOutputPorts[currentStepID]
	execCtx.mu.RUnlock()

	// Default output port if not set
	if currentOutputPort == "" {
		currentOutputPort = "output"
	}

	for _, edge := range graph.OutEdges[currentStepID] {
		// Only process step-to-group edges
		if edge.TargetBlockGroupID == nil {
			continue
		}
		groupID := *edge.TargetBlockGroupID

		// Port-based routing for step-to-group edges
		if edge.SourcePort != "" {
			if edge.SourcePort != currentOutputPort {
				continue
			}
		} else {
			// No port specified - only trigger on default "output" port
			if currentOutputPort != "output" {
				continue
			}
		}

		// Check if group already completed and all dependencies are met (single lock)
		shouldExecute := func() bool {
			mu.Lock()
			defer mu.Unlock()

			if completedGroups[groupID] {
				return false
			}

			// Check if all incoming edges to this group are satisfied
			for _, inEdge := range graph.GroupInEdges[groupID] {
				if inEdge.SourceStepID != nil && !completed[*inEdge.SourceStepID] {
					return false
				}
				if inEdge.SourceBlockGroupID != nil && !completedGroups[*inEdge.SourceBlockGroupID] {
					return false
				}
			}
			return true
		}()

		if !shouldExecute {
			continue
		}

		// Execute the block group
		group := graph.BlockGroups[groupID]
		e.logger.Info("Executing block group",
			"group_id", groupID,
			"group_name", group.Name,
			"group_type", group.Type,
		)

		groupOutput, outputPort, err := e.executeBlockGroup(ctx, execCtx, graph, &group, currentOutput)
		if err != nil {
			e.logger.Error("Block group execution failed",
				"group_id", groupID,
				"group_name", group.Name,
				"error", err,
			)
			return fmt.Errorf("block group %s execution failed: %w", group.Name, err)
		}

		mu.Lock()
		completedGroups[groupID] = true
		mu.Unlock()

		execCtx.mu.Lock()
		execCtx.GroupData[groupID] = groupOutput
		execCtx.mu.Unlock()

		// Execute next steps/groups from this group's output
		if err := e.executeFromGroupOutput(ctx, execCtx, graph, groupID, groupOutput, outputPort, completed, completedGroups, mu); err != nil {
			return err
		}
	}

	return nil
}

// executeBlockGroup executes a block group and returns its output
func (e *Executor) executeBlockGroup(ctx context.Context, execCtx *ExecutionContext, graph *Graph, group *domain.BlockGroup, input json.RawMessage) (json.RawMessage, string, error) {
	// Find steps that belong to this group
	var groupSteps []*domain.Step
	for _, step := range graph.Steps {
		if step.BlockGroupID != nil && *step.BlockGroupID == group.ID {
			s := step // Create a copy
			groupSteps = append(groupSteps, &s)
		}
	}

	// Create BlockGroupExecutor
	blockGroupExecutor := NewBlockGroupExecutor(e.registry, e.logger, e)

	// Create context for group execution
	bgCtx := &BlockGroupContext{
		Group:   group,
		Steps:   groupSteps,
		Input:   input,
		ExecCtx: execCtx,
		Graph:   graph,
	}

	// Execute group
	output, err := blockGroupExecutor.ExecuteGroup(ctx, bgCtx)
	if err != nil {
		return nil, "error", err
	}

	// Determine output port based on output content
	// Default to "out" which matches the default output port in block definitions
	outputPort := "out"
	var outputMap map[string]interface{}
	if err := json.Unmarshal(output, &outputMap); err == nil {
		if port, ok := outputMap["__port"].(string); ok {
			outputPort = port
		}
		if isError, ok := outputMap["__error"].(bool); ok && isError {
			outputPort = "error"
		}
	}

	return output, outputPort, nil
}

// executeFromGroupOutput handles execution after a group completes
func (e *Executor) executeFromGroupOutput(ctx context.Context, execCtx *ExecutionContext, graph *Graph, groupID uuid.UUID, output json.RawMessage, outputPort string, completed, completedGroups map[uuid.UUID]bool, mu *sync.Mutex) error {
	// Find edges from this group and execute next nodes
	for _, edge := range graph.GroupOutEdges[groupID] {
		// Check if this edge's source port matches the output port
		if edge.SourcePort != "" {
			if edge.SourcePort != outputPort {
				continue // Skip edges that don't match the output port
			}
		}

		// Execute next step
		if edge.TargetStepID != nil {
			// Store group output for the next step's input preparation
			execCtx.mu.Lock()
			execCtx.StepData[groupID] = output
			execCtx.mu.Unlock()

			if err := e.executeNodes(ctx, execCtx, graph, []uuid.UUID{*edge.TargetStepID}); err != nil {
				return err
			}
		}

		// Execute next group
		if edge.TargetBlockGroupID != nil {
			nextGroupID := *edge.TargetBlockGroupID
			mu.Lock()
			alreadyCompleted := completedGroups[nextGroupID]
			mu.Unlock()

			if !alreadyCompleted {
				nextGroup := graph.BlockGroups[nextGroupID]
				e.logger.Info("Executing next block group from group output",
					"from_group_id", groupID,
					"to_group_id", nextGroupID,
					"group_name", nextGroup.Name,
				)

				groupOutput, nextOutputPort, err := e.executeBlockGroup(ctx, execCtx, graph, &nextGroup, output)
				if err != nil {
					return err
				}

				mu.Lock()
				completedGroups[nextGroupID] = true
				mu.Unlock()

				execCtx.mu.Lock()
				execCtx.GroupData[nextGroupID] = groupOutput
				execCtx.mu.Unlock()

				// Recursively handle output from this group
				if err := e.executeFromGroupOutput(ctx, execCtx, graph, nextGroupID, groupOutput, nextOutputPort, completed, completedGroups, mu); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (e *Executor) findNextNodes(ctx context.Context, execCtx *ExecutionContext, graph *Graph, currentID uuid.UUID, completed, completedGroups map[uuid.UUID]bool, mu *sync.Mutex) []uuid.UUID {
	ctx, span := tracer.Start(ctx, "workflow.find_next_nodes",
		trace.WithAttributes(
			attribute.String("current_step_id", currentID.String()),
		),
	)
	defer span.End()

	var nextNodes []uuid.UUID

	// Get the output and output port of current step
	execCtx.mu.RLock()
	currentOutput := execCtx.StepData[currentID]
	currentOutputPort := execCtx.StepOutputPorts[currentID]
	execCtx.mu.RUnlock()

	// Default output port if not set
	if currentOutputPort == "" {
		currentOutputPort = "output"
	}

	for _, edge := range graph.OutEdges[currentID] {
		// Skip edges to groups (handled separately)
		if edge.TargetBlockGroupID != nil {
			continue
		}

		// Skip edges without step target
		if edge.TargetStepID == nil {
			continue
		}
		targetID := *edge.TargetStepID

		// Port-based routing: check if edge source port matches step output port
		if edge.SourcePort != "" {
			// Edge has explicit port requirement - must match
			if edge.SourcePort != currentOutputPort {
				e.logger.Debug("Edge skipped due to port mismatch",
					"edge_id", edge.ID,
					"edge_port", edge.SourcePort,
					"output_port", currentOutputPort,
				)
				continue
			}
		} else {
			// Edge has no port requirement - only match default "output" port
			// This ensures edges without port specification don't trigger on error/custom ports
			if currentOutputPort != "output" {
				e.logger.Debug("Edge skipped: no port specified, but output is non-default",
					"edge_id", edge.ID,
					"output_port", currentOutputPort,
				)
				continue
			}
		}

		// Evaluate edge condition if present
		if edge.Condition != nil && *edge.Condition != "" {
			condResult, err := e.evaluator.Evaluate(*edge.Condition, currentOutput)
			if err != nil {
				e.logger.Warn("Edge condition evaluation failed, skipping edge",
					"edge_id", edge.ID,
					"source_step_id", currentID,
					"target_step_id", targetID,
					"condition", *edge.Condition,
					"error", err,
				)
				continue // Skip this edge on evaluation error
			}
			if !condResult {
				e.logger.Debug("Edge condition not met, skipping",
					"edge_id", edge.ID,
					"condition", *edge.Condition,
				)
				continue // Condition not met, skip this edge
			}
		}

		// Check if all incoming edges' sources are completed (single lock operation)
		allDependenciesMet := func() bool {
			mu.Lock()
			defer mu.Unlock()

			for _, inEdge := range graph.InEdges[targetID] {
				if inEdge.SourceStepID != nil && !completed[*inEdge.SourceStepID] {
					return false
				}
				if inEdge.SourceBlockGroupID != nil && !completedGroups[*inEdge.SourceBlockGroupID] {
					return false
				}
			}
			return true
		}()

		if allDependenciesMet {
			nextNodes = append(nextNodes, targetID)
		}
	}

	span.SetAttributes(attribute.Int("next_node_count", len(nextNodes)))
	return nextNodes
}

func (e *Executor) executeNode(ctx context.Context, execCtx *ExecutionContext, graph *Graph, nodeID uuid.UUID) error {
	step := graph.Steps[nodeID]

	ctx, span := tracer.Start(ctx, "step.execute",
		trace.WithAttributes(
			attribute.String("step_id", step.ID.String()),
			attribute.String("step_name", step.Name),
			attribute.String("step_type", string(step.Type)),
			attribute.String("run_id", execCtx.Run.ID.String()),
		),
	)
	defer span.End()

	e.logger.Info("Executing step",
		"run_id", execCtx.Run.ID,
		"step_id", step.ID,
		"step_name", step.Name,
		"step_type", step.Type,
	)

	// Create step run with sequence number
	seqNum := execCtx.NextSequenceNumber()
	stepRun := domain.NewStepRun(execCtx.Run.TenantID, execCtx.Run.ID, step.ID, step.Name, seqNum)

	execCtx.mu.Lock()
	execCtx.StepRuns[step.ID] = stepRun
	execCtx.mu.Unlock()

	// Prepare input
	input, err := e.prepareStepInput(execCtx, step)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		stepRun.Fail(fmt.Sprintf("failed to prepare step input: %v", err))
		return fmt.Errorf("failed to prepare step input: %w", err)
	}
	stepRun.Start(input)

	// Execute step using unified dispatch
	output, err := e.dispatchStepExecution(ctx, execCtx, step, stepRun, input)

	// Determine output port (default is "output")
	outputPort := "output"

	if err != nil {
		// Check if error port is enabled in step config
		enableErrorPort := getConfigBool(step.Config, "enable_error_port")
		if enableErrorPort && e.hasEdgeFromPort(graph, step.ID, "error") {
			// Route to error port instead of failing
			errorOutput := map[string]interface{}{
				"error": map[string]interface{}{
					"message": err.Error(),
					"type":    "execution_error",
				},
				"input": json.RawMessage(input),
			}
			output, _ = json.Marshal(errorOutput)
			outputPort = "error"

			stepRun.Complete(output)
			e.logger.Info("Step error routed to error port",
				"run_id", execCtx.Run.ID,
				"step_id", step.ID,
				"error", err.Error(),
			)
		} else {
			// Normal error handling - fail the step
			stepRun.Fail(err.Error())
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			e.logger.Error("Step execution failed",
				"run_id", execCtx.Run.ID,
				"step_id", step.ID,
				"error", err,
			)
			return err
		}
	} else {
		// Check for custom output port in output (for code blocks)
		// extractOutputPortAndData also unwraps { port, data } format to just data
		var cleanedOutput json.RawMessage
		outputPort, cleanedOutput = e.extractOutputPortAndData(step, output)
		output = cleanedOutput
		stepRun.Complete(output)
	}

	execCtx.mu.Lock()
	execCtx.StepData[step.ID] = output
	execCtx.StepOutputPorts[step.ID] = outputPort
	execCtx.mu.Unlock()

	if stepRun.DurationMs != nil {
		span.SetAttributes(attribute.Int64("duration_ms", int64(*stepRun.DurationMs)))
	}
	span.SetStatus(codes.Ok, "step completed")

	e.logger.Info("Step completed",
		"run_id", execCtx.Run.ID,
		"step_id", step.ID,
		"output_port", outputPort,
		"duration_ms", stepRun.DurationMs,
	)

	return nil
}

// getConfigBool extracts a boolean value from step config
func getConfigBool(config json.RawMessage, key string) bool {
	if config == nil {
		return false
	}
	var configMap map[string]interface{}
	if err := json.Unmarshal(config, &configMap); err != nil {
		return false
	}
	if val, ok := configMap[key].(bool); ok {
		return val
	}
	return false
}

// getConfigStringArray extracts a string array from step config
func getConfigStringArray(config json.RawMessage, key string) []string {
	if config == nil {
		return nil
	}
	var configMap map[string]interface{}
	if err := json.Unmarshal(config, &configMap); err != nil {
		return nil
	}
	if val, ok := configMap[key].([]interface{}); ok {
		result := make([]string, 0, len(val))
		for _, v := range val {
			if s, ok := v.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

// hasEdgeFromPort checks if there's an edge from the step with the specified source port
func (e *Executor) hasEdgeFromPort(graph *Graph, stepID uuid.UUID, portName string) bool {
	for _, edge := range graph.OutEdges[stepID] {
		if edge.SourcePort == portName {
			return true
		}
	}
	return false
}

// extractOutputPortAndData extracts the output port and actual data from step output
// For code blocks with custom ports, it looks for { port: "portName", data: {...} } format
// Returns (outputPort, cleanedOutput) where cleanedOutput has port wrapper and __port field removed
func (e *Executor) extractOutputPortAndData(step domain.Step, output json.RawMessage) (string, json.RawMessage) {
	// Check if this is a code/function block with custom output ports
	customPorts := getConfigStringArray(step.Config, "custom_output_ports")
	if len(customPorts) == 0 {
		// Check for __port in output (used by condition/switch blocks)
		var outputMap map[string]interface{}
		if err := json.Unmarshal(output, &outputMap); err == nil {
			if port, ok := outputMap["__port"].(string); ok {
				// Remove __port from output to avoid leaking internal field to next step
				delete(outputMap, "__port")
				cleanedOutput, err := json.Marshal(outputMap)
				if err != nil {
					return port, output // fallback to original if marshal fails
				}
				return port, cleanedOutput
			}
		}
		return "output", output
	}

	// Try to parse as { port: "...", data: {...} } format
	var portOutput struct {
		Port string          `json:"port"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(output, &portOutput); err == nil && portOutput.Port != "" {
		// Validate that the port is in custom_output_ports
		for _, p := range customPorts {
			if p == portOutput.Port {
				// Return the data field content (unwrapped) instead of the full output
				if portOutput.Data != nil && len(portOutput.Data) > 0 {
					return portOutput.Port, portOutput.Data
				}
				return portOutput.Port, output
			}
		}
		// Port not in allowed list - use default
		e.logger.Warn("Invalid output port specified, using default",
			"step_id", step.ID,
			"specified_port", portOutput.Port,
			"allowed_ports", customPorts,
		)
	}

	return "output", output
}

func (e *Executor) prepareStepInput(execCtx *ExecutionContext, step domain.Step) (json.RawMessage, error) {
	execCtx.mu.RLock()
	defer execCtx.mu.RUnlock()

	// No previous step outputs - use workflow input
	if len(execCtx.StepData) == 0 && len(execCtx.GroupData) == 0 {
		return execCtx.Run.Input, nil
	}

	// Find edges pointing to this step (source edges)
	var sourceEdges []domain.Edge
	if execCtx.Definition != nil {
		for _, edge := range execCtx.Definition.Edges {
			// Consider both step-to-step and group-to-step edges
			if edge.TargetStepID != nil && *edge.TargetStepID == step.ID {
				sourceEdges = append(sourceEdges, edge)
			}
		}
	}

	// If exactly one source edge from a step, use that step's output directly (pass-through mode)
	if len(sourceEdges) == 1 {
		edge := sourceEdges[0]
		if edge.SourceStepID != nil {
			sourceStepID := *edge.SourceStepID
			if data, ok := execCtx.StepData[sourceStepID]; ok {
				return data, nil
			}
		}
		// If source is a block group, use group output
		if edge.SourceBlockGroupID != nil {
			sourceGroupID := *edge.SourceBlockGroupID
			if data, ok := execCtx.GroupData[sourceGroupID]; ok {
				return data, nil
			}
			// Also check StepData as group output might be stored there for convenience
			if data, ok := execCtx.StepData[sourceGroupID]; ok {
				return data, nil
			}
		}
		// Fallback to workflow input if source has no data
		return execCtx.Run.Input, nil
	}

	// Multiple sources or no edge info: merge all outputs (for join steps, etc.)
	merged := make(map[string]interface{})
	merged["workflow_input"] = json.RawMessage(execCtx.Run.Input)

	for stepID, data := range execCtx.StepData {
		var stepOutput interface{}
		if err := json.Unmarshal(data, &stepOutput); err != nil {
			e.logger.Warn("Failed to unmarshal step output", "step_id", stepID, "error", err)
			stepOutput = string(data) // fallback to raw string
		}
		merged[stepID.String()] = stepOutput
	}

	for groupID, data := range execCtx.GroupData {
		var groupOutput interface{}
		if err := json.Unmarshal(data, &groupOutput); err != nil {
			e.logger.Warn("Failed to unmarshal group output", "group_id", groupID, "error", err)
			groupOutput = string(data) // fallback to raw string
		}
		merged[groupID.String()] = groupOutput
	}

	result, err := json.Marshal(merged)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal merged output: %w", err)
	}
	return result, nil
}

func (e *Executor) executeStartStep(ctx context.Context, step domain.Step, input json.RawMessage) (json.RawMessage, error) {
	// Start step simply passes through the input data
	// This serves as the entry point for the workflow
	e.logger.Debug("Executing start step", "step_id", step.ID)
	return input, nil
}

func (e *Executor) executeToolStep(ctx context.Context, execCtx *ExecutionContext, step domain.Step, stepRun *domain.StepRun, input json.RawMessage) (json.RawMessage, error) {
	// Parse step config to get adapter ID
	var config struct {
		AdapterID string `json:"adapter_id"`
	}
	if err := json.Unmarshal(step.Config, &config); err != nil {
		return nil, fmt.Errorf("invalid step config: %w", err)
	}

	if config.AdapterID == "" {
		config.AdapterID = "mock"
	}

	// Get adapter
	adp, ok := e.registry.Get(config.AdapterID)
	if !ok {
		return nil, fmt.Errorf("adapter not found: %s", config.AdapterID)
	}

	// Execute adapter
	resp, err := adp.Execute(ctx, &adapter.Request{
		Input:  input,
		Config: step.Config,
	})

	// Record usage if this is an LLM adapter (has token metadata)
	if e.usageRecorder != nil && resp != nil && resp.Metadata != nil {
		// Only record if we have token information (indicates LLM call)
		if _, hasTokens := resp.Metadata["prompt_tokens"]; hasTokens {
			workflowID := execCtx.Run.WorkflowID
			runID := execCtx.Run.ID
			stepRunID := stepRun.ID
			errorMsg := ""
			if err != nil {
				errorMsg = err.Error()
			}
			e.usageRecorder.RecordFromMetadata(
				ctx,
				execCtx.Run.TenantID,
				&workflowID,
				&runID,
				&stepRunID,
				resp.Metadata,
				resp.DurationMs,
				err == nil,
				errorMsg,
			)
		}
	}

	if err != nil {
		return nil, err
	}

	return resp.Output, nil
}

func (e *Executor) executeLLMStep(ctx context.Context, execCtx *ExecutionContext, step domain.Step, stepRun *domain.StepRun, input json.RawMessage) (json.RawMessage, error) {
	// Parse step config to determine which LLM provider to use
	var config struct {
		Provider string `json:"provider"` // openai, anthropic, etc.
	}
	if err := json.Unmarshal(step.Config, &config); err != nil {
		return nil, fmt.Errorf("invalid LLM step config: %w", err)
	}

	// Determine adapter ID (default to OpenAI)
	adapterID := config.Provider
	if adapterID == "" {
		adapterID = "openai"
	}

	// Get adapter
	adp, ok := e.registry.Get(adapterID)
	if !ok {
		// Fall back to mock if adapter not found
		adp, ok = e.registry.Get("mock")
		if !ok {
			return nil, fmt.Errorf("LLM adapter not found: %s", adapterID)
		}
		e.logger.Warn("LLM adapter not found, using mock",
			"requested", adapterID,
			"step_id", step.ID,
		)
	}

	// Execute adapter
	resp, err := adp.Execute(ctx, &adapter.Request{
		Input:  input,
		Config: step.Config,
	})

	// Record usage regardless of success/failure
	if e.usageRecorder != nil && resp != nil {
		workflowID := execCtx.Run.WorkflowID
		runID := execCtx.Run.ID
		stepRunID := stepRun.ID
		errorMsg := ""
		if err != nil {
			errorMsg = err.Error()
		}
		e.usageRecorder.RecordFromMetadata(
			ctx,
			execCtx.Run.TenantID,
			&workflowID,
			&runID,
			&stepRunID,
			resp.Metadata,
			resp.DurationMs,
			err == nil,
			errorMsg,
		)
	}

	if err != nil {
		return nil, err
	}

	return resp.Output, nil
}

func (e *Executor) executeConditionStep(ctx context.Context, execCtx *ExecutionContext, step domain.Step, input json.RawMessage) (json.RawMessage, error) {
	// Parse condition config
	var config struct {
		Expression string `json:"expression"`
	}
	if err := json.Unmarshal(step.Config, &config); err != nil {
		return nil, fmt.Errorf("invalid condition config: %w", err)
	}

	// Evaluate the condition expression
	condResult, evalErr := e.evaluator.Evaluate(config.Expression, input)
	if evalErr != nil {
		e.logger.Warn("Condition evaluation failed, defaulting to true",
			"step_id", step.ID,
			"expression", config.Expression,
			"error", evalErr,
		)
		condResult = true
	}

	result := map[string]interface{}{
		"result":     condResult,
		"expression": config.Expression,
	}

	// Include evaluation error in result for debugging
	if evalErr != nil {
		result["evaluation_error"] = evalErr.Error()
		result["defaulted"] = true
	}

	e.logger.Info("Condition step evaluated",
		"step_id", step.ID,
		"expression", config.Expression,
		"result", condResult,
	)

	return json.Marshal(result)
}

func (e *Executor) executeMapStep(ctx context.Context, step domain.Step, input json.RawMessage) (json.RawMessage, error) {
	// Parse map config
	var config struct {
		InputPath  string `json:"input_path"`  // JSON path to array (e.g., "$.items")
		AdapterID  string `json:"adapter_id"`  // Optional: adapter to apply to each item
		Parallel   bool   `json:"parallel"`    // Execute in parallel
		MaxWorkers int    `json:"max_workers"` // Max parallel workers
	}
	if err := json.Unmarshal(step.Config, &config); err != nil {
		return nil, fmt.Errorf("invalid map config: %w", err)
	}

	// Extract array from input
	var items []interface{}
	if config.InputPath != "" {
		// Parse input data as map
		var inputData map[string]interface{}
		if err := json.Unmarshal(input, &inputData); err != nil {
			return nil, fmt.Errorf("invalid input data for path resolution: %w", err)
		}

		// Use evaluator to resolve path
		resolved, err := e.evaluator.ResolveValue(config.InputPath, inputData)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve input path %s: %w", config.InputPath, err)
		}
		var ok bool
		items, ok = resolved.([]interface{})
		if !ok {
			return nil, fmt.Errorf("input path %s does not resolve to an array", config.InputPath)
		}
	} else {
		// Try to use input directly as array
		if err := json.Unmarshal(input, &items); err != nil {
			return nil, fmt.Errorf("input is not an array and no input_path specified")
		}
	}

	e.logger.Info("Executing map step",
		"step_id", step.ID,
		"item_count", len(items),
		"parallel", config.Parallel,
	)

	// If no adapter specified, just pass through items
	if config.AdapterID == "" {
		result := map[string]interface{}{
			"items":  items,
			"count":  len(items),
			"mapped": true,
		}
		return json.Marshal(result)
	}

	// Get adapter
	adp, ok := e.registry.Get(config.AdapterID)
	if !ok {
		return nil, fmt.Errorf("adapter not found: %s", config.AdapterID)
	}

	// Process items
	results := make([]interface{}, len(items))
	errors := make([]error, len(items))

	if config.Parallel {
		// Parallel execution
		maxWorkers := config.MaxWorkers
		if maxWorkers <= 0 {
			maxWorkers = 10 // Default max workers
		}

		sem := make(chan struct{}, maxWorkers)
		var wg sync.WaitGroup

		for i, item := range items {
			wg.Add(1)
			go func(idx int, itm interface{}) {
				defer wg.Done()
				sem <- struct{}{}        // Acquire
				defer func() { <-sem }() // Release

				itemJSON, err := json.Marshal(itm)
				if err != nil {
					e.logger.Warn("Failed to marshal map item", "index", idx, "error", err)
					errors[idx] = err
					return
				}
				resp, err := adp.Execute(ctx, &adapter.Request{
					Input:  itemJSON,
					Config: step.Config,
				})
				if err != nil {
					errors[idx] = err
					return
				}

				var output interface{}
				if err := json.Unmarshal(resp.Output, &output); err != nil {
					e.logger.Warn("Failed to unmarshal map item output", "index", idx, "error", err)
					output = string(resp.Output)
				}
				results[idx] = output
			}(i, item)
		}
		wg.Wait()
	} else {
		// Sequential execution
		for i, item := range items {
			itemJSON, err := json.Marshal(item)
			if err != nil {
				e.logger.Warn("Failed to marshal map item", "index", i, "error", err)
				errors[i] = err
				continue
			}
			resp, err := adp.Execute(ctx, &adapter.Request{
				Input:  itemJSON,
				Config: step.Config,
			})
			if err != nil {
				errors[i] = err
				continue
			}

			var output interface{}
			if err := json.Unmarshal(resp.Output, &output); err != nil {
				e.logger.Warn("Failed to unmarshal map item output", "index", i, "error", err)
				output = string(resp.Output)
			}
			results[i] = output
		}
	}

	// Check for errors
	var firstError error
	successCount := 0
	for i, err := range errors {
		if err != nil {
			if firstError == nil {
				firstError = err
			}
			e.logger.Warn("Map item failed",
				"step_id", step.ID,
				"item_index", i,
				"error", err,
			)
		} else {
			successCount++
		}
	}

	result := map[string]interface{}{
		"items":         results,
		"count":         len(items),
		"success_count": successCount,
		"error_count":   len(items) - successCount,
	}

	// If all items failed, return error
	if successCount == 0 && len(items) > 0 {
		return nil, fmt.Errorf("all map items failed: %w", firstError)
	}

	return json.Marshal(result)
}

func (e *Executor) executeWaitStep(ctx context.Context, step domain.Step, input json.RawMessage) (json.RawMessage, error) {
	// Parse wait config
	var config domain.WaitStepConfig
	if err := json.Unmarshal(step.Config, &config); err != nil {
		return nil, fmt.Errorf("invalid wait config: %w", err)
	}

	e.logger.Info("Executing wait step",
		"step_id", step.ID,
		"duration_ms", config.DurationMs,
		"until", config.Until,
	)

	// Calculate wait duration
	var waitDuration int64

	if config.Until != "" {
		// Parse ISO8601 datetime
		targetTime, err := parseISO8601(config.Until)
		if err != nil {
			return nil, fmt.Errorf("invalid until time: %w", err)
		}
		waitDuration = targetTime.Sub(timeNow()).Milliseconds()
		if waitDuration < 0 {
			waitDuration = 0 // Already past the target time
		}
	} else {
		waitDuration = config.DurationMs
	}

	// Cap wait duration at 1 hour for safety
	maxWait := int64(3600000) // 1 hour in ms
	if waitDuration > maxWait {
		e.logger.Warn("Wait duration capped", "requested", waitDuration, "max", maxWait)
		waitDuration = maxWait
	}

	// Actually wait
	if waitDuration > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeAfter(waitDuration):
			// Done waiting
		}
	}

	output := map[string]interface{}{
		"waited_ms": waitDuration,
		"input":     json.RawMessage(input),
	}

	return json.Marshal(output)
}

func (e *Executor) executeFunctionStep(ctx context.Context, execCtx *ExecutionContext, step domain.Step, input json.RawMessage) (json.RawMessage, error) {
	// Parse function config
	var config domain.FunctionStepConfig
	if err := json.Unmarshal(step.Config, &config); err != nil {
		return nil, fmt.Errorf("invalid function config: %w", err)
	}

	// Validate language (only JavaScript supported for now)
	language := config.Language
	if language == "" {
		language = "javascript"
	}
	if language != "javascript" && language != "js" {
		return nil, fmt.Errorf("unsupported language: %s (only javascript is supported)", language)
	}

	e.logger.Info("Executing function step",
		"step_id", step.ID,
		"language", language,
	)

	// Parse input to map
	var inputMap map[string]interface{}
	if err := json.Unmarshal(input, &inputMap); err != nil {
		// If input is not a valid JSON object, wrap it
		inputMap = map[string]interface{}{
			"raw": string(input),
		}
	}

	// Create sandbox execution context with HTTP client and logger
	sandboxCtx := &sandbox.ExecutionContext{
		HTTP: sandbox.NewHTTPClient(30 * time.Second),
		Logger: func(args ...interface{}) {
			e.logger.Info("Script log", "step_id", step.ID, "message", fmt.Sprint(args...))
		},
	}

	// Add sandbox services if database pool is available (for Copilot/meta-workflow features)
	if e.pool != nil && execCtx != nil && execCtx.Run != nil {
		sandboxCtx.Blocks = sandbox.NewBlocksService(ctx, e.pool, execCtx.Run.TenantID)
		sandboxCtx.Workflows = sandbox.NewWorkflowsService(ctx, e.pool, execCtx.Run.TenantID)
		sandboxCtx.Runs = sandbox.NewRunsService(ctx, e.pool, execCtx.Run.TenantID)
	}

	// Execute the code in sandbox
	result, err := e.sandbox.Execute(ctx, config.Code, inputMap, sandboxCtx)
	if err != nil {
		e.logger.Error("Function execution failed",
			"step_id", step.ID,
			"error", err,
		)
		return nil, fmt.Errorf("function execution failed: %w", err)
	}

	e.logger.Info("Function execution completed",
		"step_id", step.ID,
	)

	// Marshal result to JSON
	output, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	// Filter output by schema if defined
	if config.OutputSchema != nil && len(config.OutputSchema) > 0 {
		filtered, filterErr := domain.FilterOutputBySchema(output, config.OutputSchema)
		if filterErr != nil {
			e.logger.Warn("Output filtering failed, using original output",
				"step_id", step.ID,
				"error", filterErr,
			)
			return output, nil
		}
		e.logger.Debug("Output filtered by schema",
			"step_id", step.ID,
		)
		return filtered, nil
	}

	return output, nil
}

func (e *Executor) executeRouterStep(ctx context.Context, step domain.Step, input json.RawMessage) (json.RawMessage, error) {
	// Parse router config
	var config domain.RouterStepConfig
	if err := json.Unmarshal(step.Config, &config); err != nil {
		return nil, fmt.Errorf("invalid router config: %w", err)
	}

	e.logger.Info("Executing router step",
		"step_id", step.ID,
		"route_count", len(config.Routes),
	)

	if len(config.Routes) == 0 {
		return nil, fmt.Errorf("router has no routes defined")
	}

	// Build route descriptions for the LLM
	routeDescriptions := make([]string, len(config.Routes))
	for i, route := range config.Routes {
		routeDescriptions[i] = fmt.Sprintf("%d. %s: %s", i+1, route.Name, route.Description)
	}

	// Build classification prompt
	prompt := config.Prompt
	if prompt == "" {
		prompt = "Based on the input, classify which route this should take. Respond with only the route name."
	}

	// Determine LLM provider
	provider := config.Provider
	if provider == "" {
		provider = "openai"
	}

	// Get LLM adapter
	adp, ok := e.registry.Get(provider)
	if !ok {
		// Fallback: just return first route
		e.logger.Warn("Router LLM adapter not found, using first route",
			"step_id", step.ID,
			"provider", provider,
		)
		output := map[string]interface{}{
			"selected_route": config.Routes[0].Name,
			"confidence":     0.0,
			"fallback":       true,
		}
		return json.Marshal(output)
	}

	// Build LLM request
	llmConfig := map[string]interface{}{
		"model": config.Model,
		"messages": []map[string]string{
			{"role": "system", "content": prompt + "\n\nAvailable routes:\n" + stringJoin(routeDescriptions, "\n")},
			{"role": "user", "content": string(input)},
		},
	}
	llmConfigJSON, err := json.Marshal(llmConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal LLM config: %w", err)
	}

	resp, err := adp.Execute(ctx, &adapter.Request{
		Input:  input,
		Config: llmConfigJSON,
	})
	if err != nil {
		// Fallback on error
		e.logger.Warn("Router LLM call failed, using first route",
			"step_id", step.ID,
			"error", err,
		)
		output := map[string]interface{}{
			"selected_route": config.Routes[0].Name,
			"confidence":     0.0,
			"error":          err.Error(),
		}
		return json.Marshal(output)
	}

	// Parse LLM response to determine route
	var llmOutput map[string]interface{}
	if err := json.Unmarshal(resp.Output, &llmOutput); err != nil {
		e.logger.Warn("Failed to parse LLM router response", "error", err)
		llmOutput = map[string]interface{}{}
	}

	// Try to extract route from response
	selectedRoute := config.Routes[0].Name
	if content, ok := llmOutput["content"].(string); ok {
		// Check if response matches any route name
		for _, route := range config.Routes {
			if stringContains(content, route.Name) {
				selectedRoute = route.Name
				break
			}
		}
	}

	output := map[string]interface{}{
		"selected_route": selectedRoute,
		"llm_response":   llmOutput,
	}

	return json.Marshal(output)
}

func (e *Executor) executeHumanInLoopStep(ctx context.Context, execCtx *ExecutionContext, step domain.Step, input json.RawMessage) (json.RawMessage, error) {
	// Parse human-in-loop config
	var config domain.HumanInLoopStepConfig
	if err := json.Unmarshal(step.Config, &config); err != nil {
		return nil, fmt.Errorf("invalid human-in-loop config: %w", err)
	}

	e.logger.Info("Executing human-in-loop step",
		"step_id", step.ID,
		"run_id", execCtx.Run.ID,
	)

	// Generate approval URL
	approvalID := uuid.New().String()
	approvalURL := fmt.Sprintf("/api/v1/runs/%s/approve/%s", execCtx.Run.ID, approvalID)

	// In a real implementation, this would:
	// 1. Store the pending approval in the database
	// 2. Send notification if configured
	// 3. Update run status to "waiting_approval"
	// 4. Return and let the workflow pause
	// 5. Resume when approval is received

	// For now, auto-approve in test mode
	autoApprove := execCtx.Run.TriggeredBy == domain.TriggerTypeTest

	output := map[string]interface{}{
		"approval_id":     approvalID,
		"approval_url":    approvalURL,
		"status":          "pending",
		"auto_approved":   autoApprove,
		"instructions":    config.Instructions,
		"required_fields": config.RequiredFields,
		"input":           json.RawMessage(input),
	}

	if autoApprove {
		output["status"] = "approved"
		output["approved_at"] = timeNow().Format("2006-01-02T15:04:05Z07:00")
		output["approved_by"] = "system (test mode)"
	}

	e.logger.Info("Human-in-loop step completed",
		"step_id", step.ID,
		"approval_id", approvalID,
		"auto_approved", autoApprove,
	)

	return json.Marshal(output)
}

func (e *Executor) executeSwitchStep(ctx context.Context, execCtx *ExecutionContext, step domain.Step, input json.RawMessage) (json.RawMessage, error) {
	// Parse switch config
	var config domain.SwitchStepConfig
	if err := json.Unmarshal(step.Config, &config); err != nil {
		return nil, fmt.Errorf("invalid switch config: %w", err)
	}

	e.logger.Info("Executing switch step",
		"step_id", step.ID,
		"case_count", len(config.Cases),
		"mode", config.Mode,
	)

	if len(config.Cases) == 0 {
		return nil, fmt.Errorf("switch has no cases defined")
	}

	// Evaluate each case in order
	var matchedCase *domain.SwitchCase
	var defaultCase *domain.SwitchCase

	for i := range config.Cases {
		c := &config.Cases[i]
		if c.IsDefault {
			defaultCase = c
			continue
		}

		// Evaluate the case expression
		result, err := e.evaluator.Evaluate(c.Expression, input)
		if err != nil {
			e.logger.Warn("Switch case evaluation failed",
				"step_id", step.ID,
				"case_name", c.Name,
				"expression", c.Expression,
				"error", err,
			)
			continue
		}

		if result {
			matchedCase = c
			break
		}
	}

	// Use default case if no match
	if matchedCase == nil {
		matchedCase = defaultCase
	}

	output := map[string]interface{}{
		"input":        json.RawMessage(input),
		"matched_case": nil,
	}

	if matchedCase != nil {
		output["matched_case"] = matchedCase.Name
		e.logger.Info("Switch case matched",
			"step_id", step.ID,
			"case_name", matchedCase.Name,
		)
	} else {
		e.logger.Info("Switch no case matched",
			"step_id", step.ID,
		)
	}

	return json.Marshal(output)
}

func (e *Executor) executeFilterStep(ctx context.Context, step domain.Step, input json.RawMessage) (json.RawMessage, error) {
	// Parse filter config
	var config domain.FilterStepConfig
	if err := json.Unmarshal(step.Config, &config); err != nil {
		return nil, fmt.Errorf("invalid filter config: %w", err)
	}

	e.logger.Info("Executing filter step",
		"step_id", step.ID,
		"expression", config.Expression,
		"keep_all", config.KeepAll,
	)

	// Parse input as array
	var items []interface{}
	if err := json.Unmarshal(input, &items); err != nil {
		// Try to parse as object with items field
		var inputObj map[string]interface{}
		if err2 := json.Unmarshal(input, &inputObj); err2 != nil {
			return nil, fmt.Errorf("input is not an array or object: %w", err)
		}
		if itemsVal, ok := inputObj["items"].([]interface{}); ok {
			items = itemsVal
		} else {
			// If keep_all mode, evaluate expression against entire input
			if config.KeepAll {
				result, err := e.evaluator.Evaluate(config.Expression, input)
				if err != nil {
					return nil, fmt.Errorf("filter expression evaluation failed: %w", err)
				}
				output := map[string]interface{}{
					"kept":       result,
					"input":      json.RawMessage(input),
					"expression": config.Expression,
				}
				return json.Marshal(output)
			}
			return nil, fmt.Errorf("input does not contain array items")
		}
	}

	// Filter items
	var filteredItems []interface{}
	for i, item := range items {
		itemJSON, err := json.Marshal(item)
		if err != nil {
			e.logger.Warn("Failed to marshal filter item", "step_id", step.ID, "index", i, "error", err)
			continue
		}
		result, err := e.evaluator.Evaluate(config.Expression, itemJSON)
		if err != nil {
			e.logger.Warn("Filter item evaluation failed",
				"step_id", step.ID,
				"error", err,
			)
			continue
		}
		if result {
			filteredItems = append(filteredItems, item)
		}
	}

	output := map[string]interface{}{
		"items":          filteredItems,
		"original_count": len(items),
		"filtered_count": len(filteredItems),
		"removed_count":  len(items) - len(filteredItems),
	}

	e.logger.Info("Filter step completed",
		"step_id", step.ID,
		"original_count", len(items),
		"filtered_count", len(filteredItems),
	)

	return json.Marshal(output)
}

func (e *Executor) executeSplitStep(ctx context.Context, step domain.Step, input json.RawMessage) (json.RawMessage, error) {
	// Parse split config
	var config domain.SplitStepConfig
	if err := json.Unmarshal(step.Config, &config); err != nil {
		return nil, fmt.Errorf("invalid split config: %w", err)
	}

	e.logger.Info("Executing split step",
		"step_id", step.ID,
		"batch_size", config.BatchSize,
		"input_path", config.InputPath,
	)

	// Extract array from input
	var items []interface{}
	if config.InputPath != "" {
		var inputData map[string]interface{}
		if err := json.Unmarshal(input, &inputData); err != nil {
			return nil, fmt.Errorf("invalid input for path resolution: %w", err)
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
		if err := json.Unmarshal(input, &items); err != nil {
			return nil, fmt.Errorf("input is not an array")
		}
	}

	// Set default batch size
	batchSize := config.BatchSize
	if batchSize <= 0 {
		batchSize = 1
	}

	// Split into batches
	var batches [][]interface{}
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}
		batches = append(batches, items[i:end])
	}

	output := map[string]interface{}{
		"batches":      batches,
		"batch_count":  len(batches),
		"total_items":  len(items),
		"batch_size":   batchSize,
	}

	e.logger.Info("Split step completed",
		"step_id", step.ID,
		"batch_count", len(batches),
		"total_items", len(items),
	)

	return json.Marshal(output)
}

func (e *Executor) executeAggregateStep(ctx context.Context, step domain.Step, input json.RawMessage) (json.RawMessage, error) {
	// Parse aggregate config
	var config domain.AggregateStepConfig
	if err := json.Unmarshal(step.Config, &config); err != nil {
		return nil, fmt.Errorf("invalid aggregate config: %w", err)
	}

	e.logger.Info("Executing aggregate step",
		"step_id", step.ID,
		"operation_count", len(config.Operations),
		"group_by", config.GroupBy,
	)

	// Parse input as array
	var items []interface{}
	if err := json.Unmarshal(input, &items); err != nil {
		// Try to parse as object with items field
		var inputObj map[string]interface{}
		if err2 := json.Unmarshal(input, &inputObj); err2 != nil {
			return nil, fmt.Errorf("input is not an array or object: %w", err)
		}
		if itemsVal, ok := inputObj["items"].([]interface{}); ok {
			items = itemsVal
		} else {
			return nil, fmt.Errorf("input does not contain array items")
		}
	}

	// Perform aggregations
	result := make(map[string]interface{})

	for _, op := range config.Operations {
		var value interface{}

		switch op.Operation {
		case "count":
			value = len(items)

		case "sum":
			sum := 0.0
			for _, item := range items {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if v := getNumericValue(itemMap, op.Field); v != nil {
						sum += *v
					}
				}
			}
			value = sum

		case "avg":
			sum := 0.0
			count := 0
			for _, item := range items {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if v := getNumericValue(itemMap, op.Field); v != nil {
						sum += *v
						count++
					}
				}
			}
			if count > 0 {
				value = sum / float64(count)
			} else {
				value = 0.0
			}

		case "min":
			var min *float64
			for _, item := range items {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if v := getNumericValue(itemMap, op.Field); v != nil {
						if min == nil || *v < *min {
							min = v
						}
					}
				}
			}
			if min != nil {
				value = *min
			}

		case "max":
			var max *float64
			for _, item := range items {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if v := getNumericValue(itemMap, op.Field); v != nil {
						if max == nil || *v > *max {
							max = v
						}
					}
				}
			}
			if max != nil {
				value = *max
			}

		case "first":
			if len(items) > 0 {
				if op.Field != "" {
					if itemMap, ok := items[0].(map[string]interface{}); ok {
						value = itemMap[op.Field]
					}
				} else {
					value = items[0]
				}
			}

		case "last":
			if len(items) > 0 {
				if op.Field != "" {
					if itemMap, ok := items[len(items)-1].(map[string]interface{}); ok {
						value = itemMap[op.Field]
					}
				} else {
					value = items[len(items)-1]
				}
			}

		case "concat":
			var parts []string
			for _, item := range items {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if v, ok := itemMap[op.Field].(string); ok {
						parts = append(parts, v)
					}
				}
			}
			sep := op.Separator
			if sep == "" {
				sep = ", "
			}
			value = stringJoin(parts, sep)

		default:
			e.logger.Warn("Unknown aggregate operation",
				"step_id", step.ID,
				"operation", op.Operation,
			)
			continue
		}

		result[op.OutputField] = value
	}

	result["item_count"] = len(items)

	e.logger.Info("Aggregate step completed",
		"step_id", step.ID,
		"item_count", len(items),
	)

	return json.Marshal(result)
}

func (e *Executor) executeErrorStep(ctx context.Context, step domain.Step, input json.RawMessage) (json.RawMessage, error) {
	// Parse error config
	var config domain.ErrorStepConfig
	if err := json.Unmarshal(step.Config, &config); err != nil {
		return nil, fmt.Errorf("invalid error config: %w", err)
	}

	e.logger.Info("Executing error step",
		"step_id", step.ID,
		"error_type", config.ErrorType,
		"error_code", config.ErrorCode,
	)

	// Create and return error
	errMsg := config.ErrorMessage
	if errMsg == "" {
		errMsg = "Workflow stopped by error step"
	}

	return nil, &WorkflowError{
		Type:    config.ErrorType,
		Code:    config.ErrorCode,
		Message: errMsg,
	}
}

func (e *Executor) executeNoteStep(ctx context.Context, step domain.Step, input json.RawMessage) (json.RawMessage, error) {
	// Note step is a no-op, just pass through input
	e.logger.Debug("Executing note step (pass-through)",
		"step_id", step.ID,
	)

	return input, nil
}

// executeLogStep outputs a log message for debugging purposes
func (e *Executor) executeLogStep(ctx context.Context, step domain.Step, input json.RawMessage) (json.RawMessage, error) {
	var config domain.LogStepConfig
	if err := json.Unmarshal(step.Config, &config); err != nil {
		return nil, err
	}

	// Default log level
	level := config.Level
	if level == "" {
		level = "info"
	}

	// Process message template (replace {{$.field}} with actual values)
	message := config.Message
	if input != nil {
		message = substituteLogTemplateVariables(message, input)
	}

	// Build log output
	logOutput := map[string]interface{}{
		"message":   message,
		"level":     level,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	// If data path is specified, extract and include that data
	if config.Data != "" && input != nil {
		var inputData interface{}
		if err := json.Unmarshal(input, &inputData); err == nil {
			if extracted := extractLogJSONPath(inputData, config.Data); extracted != nil {
				logOutput["data"] = extracted
			}
		}
	}

	// Log to executor logger
	switch level {
	case "debug":
		e.logger.Debug("Log step output",
			"step_id", step.ID,
			"message", message,
		)
	case "warn":
		e.logger.Warn("Log step output",
			"step_id", step.ID,
			"message", message,
		)
	case "error":
		e.logger.Error("Log step output",
			"step_id", step.ID,
			"message", message,
		)
	default:
		e.logger.Info("Log step output",
			"step_id", step.ID,
			"message", message,
		)
	}

	// Return log output so it's visible in StepRun
	output, err := json.Marshal(logOutput)
	if err != nil {
		return nil, err
	}

	return output, nil
}

// executeCustomBlockStep executes a custom block defined in the block_definitions table
// This supports the unified block model with inheritance, preProcess/postProcess chains, and internal steps
func (e *Executor) executeCustomBlockStep(ctx context.Context, execCtx *ExecutionContext, step domain.Step, input json.RawMessage) (json.RawMessage, error) {
	if e.blockDefRepo == nil {
		return nil, fmt.Errorf("block definition repository not configured, cannot execute custom block")
	}

	if step.BlockDefinitionID == nil {
		return nil, fmt.Errorf("step has no block definition ID")
	}

	e.logger.Info("Executing custom block step",
		"step_id", step.ID,
		"step_name", step.Name,
		"block_definition_id", step.BlockDefinitionID,
	)

	// Get block definition from repository (with inheritance resolved)
	blockDef, err := e.blockDefRepo.GetByID(ctx, *step.BlockDefinitionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get block definition: %w", err)
	}
	if blockDef == nil {
		return nil, fmt.Errorf("block definition not found: %s", step.BlockDefinitionID)
	}

	// Validate tenant access: block must be either a system block (no tenant) or belong to the same tenant
	if blockDef.TenantID != nil {
		// Tenant-specific blocks require a valid execution context with tenant info
		if execCtx == nil || execCtx.Run == nil {
			return nil, fmt.Errorf("tenant-specific block %q (step: %s) requires execution context with run information", blockDef.Slug, step.Name)
		}
		if *blockDef.TenantID != execCtx.Run.TenantID {
			return nil, fmt.Errorf("block definition %q belongs to a different tenant", blockDef.Slug)
		}
	}

	// Execute the block with unified model
	return e.executeBlockDefinition(ctx, execCtx, step, blockDef, input)
}

// executeBlockDefinition executes a block definition using the unified execution model
// Flow: preProcess chain -> internal_steps -> code -> postProcess chain
func (e *Executor) executeBlockDefinition(ctx context.Context, execCtx *ExecutionContext, step domain.Step, blockDef *domain.BlockDefinition, input json.RawMessage) (json.RawMessage, error) {
	// Parse input to map
	var inputMap map[string]interface{}
	if err := json.Unmarshal(input, &inputMap); err != nil {
		inputMap = map[string]interface{}{
			"raw": string(input),
		}
	}

	// Parse step config and merge with resolved config defaults
	configMap := e.mergeBlockConfig(step.Config, blockDef.GetEffectiveConfigDefaults())

	// Create sandbox execution context
	sandboxCtx := e.createSandboxContext(ctx, execCtx, step.ID, blockDef.Slug)

	// === Phase 1: Execute preProcess chain (child -> root order) ===
	currentInput := inputMap
	if len(blockDef.PreProcessChain) > 0 {
		e.logger.Debug("Executing preProcess chain",
			"step_id", step.ID,
			"block", blockDef.Slug,
			"chain_length", len(blockDef.PreProcessChain),
		)

		for i, preCode := range blockDef.PreProcessChain {
			if preCode == "" {
				continue
			}
			processed, err := e.runHookCode(ctx, preCode, currentInput, configMap, sandboxCtx, "preProcess", i)
			if err != nil {
				e.logger.Error("preProcess hook failed",
					"step_id", step.ID,
					"block", blockDef.Slug,
					"hook_index", i,
					"error", err,
				)
				return nil, fmt.Errorf("preProcess hook %d failed: %w", i, err)
			}
			currentInput = processed
		}
	} else if blockDef.PreProcess != "" {
		// Single preProcess (non-inherited block)
		processed, err := e.runHookCode(ctx, blockDef.PreProcess, currentInput, configMap, sandboxCtx, "preProcess", 0)
		if err != nil {
			return nil, fmt.Errorf("preProcess failed: %w", err)
		}
		currentInput = processed
	}

	var output map[string]interface{}

	// === Phase 2: Execute internal_steps (if any) ===
	if len(blockDef.InternalSteps) > 0 {
		e.logger.Debug("Executing internal steps",
			"step_id", step.ID,
			"block", blockDef.Slug,
			"step_count", len(blockDef.InternalSteps),
		)

		results := make(map[string]interface{})
		stepInput := currentInput

		for i, internalStep := range blockDef.InternalSteps {
			stepOutput, err := e.executeInternalStep(ctx, execCtx, internalStep, stepInput)
			if err != nil {
				e.logger.Error("Internal step failed",
					"step_id", step.ID,
					"block", blockDef.Slug,
					"internal_step_index", i,
					"internal_step_type", internalStep.Type,
					"error", err,
				)
				return nil, fmt.Errorf("internal step %d (%s) failed: %w", i, internalStep.Type, err)
			}

			// Store result by output_key
			if internalStep.OutputKey != "" {
				results[internalStep.OutputKey] = stepOutput
			}

			// Merge results into input for next step
			stepInput = e.mergeInputWithResults(currentInput, results)
		}

		output = results
	}

	// === Phase 3: Execute main code (if any) ===
	effectiveCode := blockDef.GetEffectiveCode()
	if effectiveCode != "" {
		e.logger.Debug("Executing main code",
			"step_id", step.ID,
			"block", blockDef.Slug,
		)

		// Prepare sandbox input
		sandboxInput := currentInput
		sandboxInput["__config"] = configMap

		// If we have internal step results, make them available
		if output != nil {
			sandboxInput["__internal_results"] = output
		}

		wrappedCode := wrapCustomBlockCode(effectiveCode)
		codeResult, err := e.sandbox.Execute(ctx, wrappedCode, sandboxInput, sandboxCtx)
		if err != nil {
			e.logger.Error("Block code execution failed",
				"step_id", step.ID,
				"block", blockDef.Slug,
				"error", err,
			)
			return nil, fmt.Errorf("block code execution failed: %w", err)
		}

		if output == nil {
			// No internal steps, use code result directly
			output = codeResult
		} else {
			// Merge code result with internal step results
			output["_code_result"] = codeResult
		}
	}

	// If no code and no internal steps, pass through input
	if output == nil {
		output = currentInput
	}

	// === Phase 4: Execute postProcess chain (root -> child order) ===
	currentOutput := output
	if len(blockDef.PostProcessChain) > 0 {
		e.logger.Debug("Executing postProcess chain",
			"step_id", step.ID,
			"block", blockDef.Slug,
			"chain_length", len(blockDef.PostProcessChain),
		)

		for i, postCode := range blockDef.PostProcessChain {
			if postCode == "" {
				continue
			}
			processed, err := e.runHookCode(ctx, postCode, currentOutput, configMap, sandboxCtx, "postProcess", i)
			if err != nil {
				e.logger.Error("postProcess hook failed",
					"step_id", step.ID,
					"block", blockDef.Slug,
					"hook_index", i,
					"error", err,
				)
				return nil, fmt.Errorf("postProcess hook %d failed: %w", i, err)
			}
			currentOutput = processed
		}
	} else if blockDef.PostProcess != "" {
		// Single postProcess (non-inherited block)
		processed, err := e.runHookCode(ctx, blockDef.PostProcess, currentOutput, configMap, sandboxCtx, "postProcess", 0)
		if err != nil {
			return nil, fmt.Errorf("postProcess failed: %w", err)
		}
		currentOutput = processed
	}

	e.logger.Info("Custom block execution completed",
		"step_id", step.ID,
		"block", blockDef.Slug,
	)

	// Marshal result to JSON
	result, err := json.Marshal(currentOutput)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	// Filter output by schema if defined in step config
	if configMap != nil {
		if outputSchemaRaw, ok := configMap["output_schema"]; ok && outputSchemaRaw != nil {
			outputSchemaJSON, err := json.Marshal(outputSchemaRaw)
			if err == nil && len(outputSchemaJSON) > 0 {
				filtered, filterErr := domain.FilterOutputBySchema(result, outputSchemaJSON)
				if filterErr != nil {
					e.logger.Warn("Output filtering failed, using original output",
						"step_id", step.ID,
						"block", blockDef.Slug,
						"error", filterErr,
					)
					return result, nil
				}
				return filtered, nil
			}
		}
	}

	return result, nil
}

// mergeBlockConfig merges step config with block's resolved config defaults
func (e *Executor) mergeBlockConfig(stepConfig json.RawMessage, defaults json.RawMessage) map[string]interface{} {
	result := make(map[string]interface{})

	// First apply defaults
	if defaults != nil && len(defaults) > 0 && string(defaults) != "{}" {
		json.Unmarshal(defaults, &result)
	}

	// Then override with step config
	if stepConfig != nil && len(stepConfig) > 0 {
		var stepConfigMap map[string]interface{}
		if err := json.Unmarshal(stepConfig, &stepConfigMap); err == nil {
			for k, v := range stepConfigMap {
				result[k] = v
			}
		}
	}

	return result
}

// createSandboxContext creates a sandbox execution context for block execution
// All services defined in ExecutionContext should be initialized here to prevent
// "undefined" errors when blocks access ctx.* properties in JavaScript.
func (e *Executor) createSandboxContext(ctx context.Context, execCtx *ExecutionContext, stepID uuid.UUID, blockSlug string) *sandbox.ExecutionContext {
	sandboxCtx := &sandbox.ExecutionContext{
		HTTP: sandbox.NewHTTPClient(30 * time.Second),
		Logger: func(args ...interface{}) {
			e.logger.Info("Custom block script log", "step_id", stepID, "block", blockSlug, "message", fmt.Sprint(args...))
		},
	}

	// Initialize LLM service (needed for AI/RAG blocks)
	sandboxCtx.LLM = sandbox.NewLLMService(ctx)

	// Initialize Embedding service (needed for RAG blocks)
	embeddingService := sandbox.NewEmbeddingService(ctx)
	sandboxCtx.Embedding = embeddingService

	// Initialize stub services (return errors if used, but prevent undefined errors)
	sandboxCtx.Workflow = sandbox.NewWorkflowService()
	sandboxCtx.Human = sandbox.NewHumanService()
	sandboxCtx.Adapter = sandbox.NewAdapterService()

	if e.pool != nil && execCtx != nil && execCtx.Run != nil {
		sandboxCtx.Blocks = sandbox.NewBlocksService(ctx, e.pool, execCtx.Run.TenantID)
		sandboxCtx.Workflows = sandbox.NewWorkflowsService(ctx, e.pool, execCtx.Run.TenantID)
		sandboxCtx.Runs = sandbox.NewRunsService(ctx, e.pool, execCtx.Run.TenantID)
		// Initialize Vector service with tenant isolation (needed for RAG blocks)
		sandboxCtx.Vector = sandbox.NewVectorService(ctx, execCtx.Run.TenantID, e.pool, embeddingService)
	}

	return sandboxCtx
}

// runHookCode executes preProcess or postProcess JavaScript code
func (e *Executor) runHookCode(ctx context.Context, code string, input map[string]interface{}, config map[string]interface{}, sandboxCtx *sandbox.ExecutionContext, hookType string, index int) (map[string]interface{}, error) {
	// Wrap hook code with config access
	wrappedCode := fmt.Sprintf(`
var config = input.__config || {};
delete input.__config;
var ctx = {
	log: function(level, message) { console.log('[' + level + '] ' + message); }
};
(function() {
%s
})();
`, code)

	// Prepare input with config
	sandboxInput := make(map[string]interface{})
	for k, v := range input {
		sandboxInput[k] = v
	}
	sandboxInput["__config"] = config

	result, err := e.sandbox.Execute(ctx, wrappedCode, sandboxInput, sandboxCtx)
	if err != nil {
		return nil, err
	}

	// Result is already map[string]interface{} from sandbox.Execute
	return result, nil
}

// executeInternalStep executes a single internal step by slug
func (e *Executor) executeInternalStep(ctx context.Context, execCtx *ExecutionContext, internalStep domain.InternalStep, input map[string]interface{}) (interface{}, error) {
	if e.blockDefRepo == nil {
		return nil, fmt.Errorf("block definition repository not configured")
	}

	// Get the block definition for this step type
	var tenantID *uuid.UUID
	if execCtx != nil && execCtx.Run != nil {
		tenantID = &execCtx.Run.TenantID
	}

	blockDef, err := e.blockDefRepo.GetBySlug(ctx, tenantID, internalStep.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to get block definition for internal step: %w", err)
	}
	if blockDef == nil {
		return nil, fmt.Errorf("block definition not found for type: %s", internalStep.Type)
	}

	// Prepare input JSON
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal internal step input: %w", err)
	}

	// Create a temporary step for execution
	tempStep := domain.Step{
		ID:                uuid.New(),
		Name:              fmt.Sprintf("internal_%s", internalStep.Type),
		Type:              domain.StepType(internalStep.Type),
		Config:            internalStep.Config,
		BlockDefinitionID: &blockDef.ID,
	}

	// Execute the block
	outputJSON, err := e.executeBlockDefinition(ctx, execCtx, tempStep, blockDef, inputJSON)
	if err != nil {
		return nil, err
	}

	// Parse output
	var output interface{}
	if err := json.Unmarshal(outputJSON, &output); err != nil {
		return string(outputJSON), nil
	}

	return output, nil
}

// mergeInputWithResults merges original input with internal step results
func (e *Executor) mergeInputWithResults(input map[string]interface{}, results map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	for k, v := range input {
		merged[k] = v
	}
	for k, v := range results {
		merged[k] = v
	}
	return merged
}

// executeCustomBlockStepBySlug executes a custom block by looking up the block definition by slug (step type)
func (e *Executor) executeCustomBlockStepBySlug(ctx context.Context, execCtx *ExecutionContext, step domain.Step, input json.RawMessage) (json.RawMessage, error) {
	if e.blockDefRepo == nil {
		return input, nil // No repo, pass through
	}

	// Try to find block definition by step type (slug)
	// First try tenant-specific, then system blocks (tenant_id = NULL)
	var blockDef *domain.BlockDefinition
	var err error

	if execCtx != nil && execCtx.Run != nil {
		blockDef, err = e.blockDefRepo.GetBySlug(ctx, &execCtx.Run.TenantID, string(step.Type))
	}
	if err != nil || blockDef == nil {
		// Try system blocks
		blockDef, err = e.blockDefRepo.GetBySlug(ctx, nil, string(step.Type))
	}
	if err != nil || blockDef == nil {
		// No block definition found, pass through
		e.logger.Debug("No block definition found for step type, passing through",
			"step_id", step.ID,
			"step_type", step.Type,
		)
		return input, nil
	}

	// Check if block has code, internal steps, or preProcess/postProcess
	// If none of these, pass through
	if blockDef.GetEffectiveCode() == "" && len(blockDef.InternalSteps) == 0 && blockDef.PreProcess == "" && blockDef.PostProcess == "" && len(blockDef.PreProcessChain) == 0 && len(blockDef.PostProcessChain) == 0 {
		return input, nil
	}

	e.logger.Info("Executing custom block step by slug",
		"step_id", step.ID,
		"step_name", step.Name,
		"step_type", step.Type,
		"block_slug", blockDef.Slug,
	)

	// Use unified execution model
	return e.executeBlockDefinition(ctx, execCtx, step, blockDef, input)
}

// wrapCustomBlockCode wraps custom block code with setup that provides expected globals
func wrapCustomBlockCode(code string) string {
	// Wrapper provides:
	// - config: the step configuration
	// - renderTemplate: simple {{variable}} substitution
	return `
// Setup globals for custom block
var config = input.__config || {};
delete input.__config;

// Simple template rendering function
function renderTemplate(template, data) {
	if (!template) return '';
	if (typeof template !== 'string') return String(template);

	return template.replace(/\{\{([^}]+)\}\}/g, function(match, path) {
		var value = data;
		var parts = path.trim().split('.');
		for (var i = 0; i < parts.length; i++) {
			if (value == null) return '';
			value = value[parts[i]];
		}
		return value != null ? String(value) : '';
	});
}

// Execute the block code
(function() {
` + code + `
})();
`
}

// substituteLogTemplateVariables replaces {{$.path}} patterns in the message with values from input
func substituteLogTemplateVariables(template string, input json.RawMessage) string {
	var inputData interface{}
	if err := json.Unmarshal(input, &inputData); err != nil {
		return template
	}

	// Find and replace all {{$.path}} patterns
	result := template
	for {
		start := strings.Index(result, "{{")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "}}")
		if end == -1 {
			break
		}
		end += start + 2

		path := strings.TrimSpace(result[start+2 : end-2])
		value := extractLogJSONPath(inputData, path)
		var replacement string
		if value != nil {
			switch v := value.(type) {
			case string:
				replacement = v
			default:
				if jsonBytes, err := json.Marshal(v); err == nil {
					replacement = string(jsonBytes)
				}
			}
		}
		result = result[:start] + replacement + result[end:]
	}

	return result
}

// extractLogJSONPath extracts a value from data using a JSON path like $.field or $.nested.field
func extractLogJSONPath(data interface{}, path string) interface{} {
	// Remove leading $. if present
	path = strings.TrimPrefix(path, "$.")
	path = strings.TrimPrefix(path, "$")

	if path == "" {
		return data
	}

	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		if part == "" {
			continue
		}

		switch v := current.(type) {
		case map[string]interface{}:
			var ok bool
			current, ok = v[part]
			if !ok {
				return nil
			}
		default:
			return nil
		}
	}

	return current
}

// WorkflowError represents a custom workflow error from error step
type WorkflowError struct {
	Type    string
	Code    string
	Message string
}

func (e *WorkflowError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Type, e.Code, e.Message)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// getNumericValue extracts a numeric value from a map field
func getNumericValue(m map[string]interface{}, field string) *float64 {
	v, ok := m[field]
	if !ok {
		return nil
	}
	switch val := v.(type) {
	case float64:
		return &val
	case int:
		f := float64(val)
		return &f
	case int64:
		f := float64(val)
		return &f
	default:
		return nil
	}
}

// Helper functions

func parseISO8601(s string) (time.Time, error) {
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time: %s", s)
}

func stringJoin(strs []string, sep string) string {
	return strings.Join(strs, sep)
}

// stringContains performs case-insensitive substring search.
// Both strings are converted to lowercase for comparison.
// Early returns handle edge cases (empty substring, impossible match).
func stringContains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(substr) > len(s) {
		return false
	}
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// Time functions (for testing)
var timeNow = time.Now
var timeAfter = func(ms int64) <-chan time.Time {
	return time.After(time.Duration(ms) * time.Millisecond)
}
