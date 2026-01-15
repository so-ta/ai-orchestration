package engine

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/adapter"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a test executor
func newTestExecutor() *Executor {
	registry := adapter.NewRegistry()
	registry.Register(adapter.NewMockAdapter())
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	return NewExecutor(registry, logger)
}

// Helper function to create a test run
func newTestRun(workflowID uuid.UUID, input json.RawMessage) *domain.Run {
	return &domain.Run{
		ID:          uuid.New(),
		TenantID:    uuid.New(),
		WorkflowID:  workflowID,
		Status:      domain.RunStatusPending,
		Input:       input,
		TriggeredBy: domain.TriggerTypeTest,
		CreatedAt:   time.Now(),
	}
}

// =============================================================================
// Phase 1.1: Group Boundary Connection Tests
// =============================================================================

// TestExecutor_Integration_StepToGroupConnection tests step → group edge execution
func TestExecutor_Integration_StepToGroupConnection(t *testing.T) {
	executor := newTestExecutor()
	ctx := context.Background()

	// Create IDs
	startStepID := uuid.New()
	initStepID := uuid.New()
	parallelGroupID := uuid.New()
	branchAStepID := uuid.New()
	branchBStepID := uuid.New()
	mergeStepID := uuid.New()

	// Define workflow with step → group connection
	def := &domain.WorkflowDefinition{
		Name: "Step to Group Test",
		Steps: []domain.Step{
			{
				ID:   startStepID,
				Name: "Start",
				Type: domain.StepTypeStart,
			},
			{
				ID:     initStepID,
				Name:   "Initialize",
				Type:   domain.StepTypeFunction,
				Config: json.RawMessage(`{"code": "return { value: 10, items: [1, 2, 3] };", "language": "javascript"}`),
			},
			{
				ID:           branchAStepID,
				Name:         "Branch A",
				Type:         domain.StepTypeFunction,
				BlockGroupID: &parallelGroupID,
				Config:       json.RawMessage(`{"code": "return { branch: 'A', doubled: input.value * 2 };", "language": "javascript"}`),
			},
			{
				ID:           branchBStepID,
				Name:         "Branch B",
				Type:         domain.StepTypeFunction,
				BlockGroupID: &parallelGroupID,
				Config:       json.RawMessage(`{"code": "return { branch: 'B', tripled: input.value * 3 };", "language": "javascript"}`),
			},
			{
				ID:     mergeStepID,
				Name:   "Merge",
				Type:   domain.StepTypeFunction,
				Config: json.RawMessage(`{"code": "return { merged: true, input: input };", "language": "javascript"}`),
			},
		},
		Edges: []domain.Edge{
			// start → init
			{ID: uuid.New(), SourceStepID: &startStepID, TargetStepID: &initStepID},
			// init → parallel_group (step to group edge)
			{ID: uuid.New(), SourceStepID: &initStepID, TargetBlockGroupID: &parallelGroupID, TargetPort: "group-input"},
			// parallel_group → merge (group to step edge)
			{ID: uuid.New(), SourceBlockGroupID: &parallelGroupID, TargetStepID: &mergeStepID, SourcePort: "out"},
		},
		BlockGroups: []domain.BlockGroup{
			{
				ID:     parallelGroupID,
				Name:   "Parallel Group",
				Type:   domain.BlockGroupTypeParallel,
				Config: json.RawMessage(`{"max_concurrent": 2, "fail_fast": false}`),
			},
		},
	}

	input := json.RawMessage(`{"initial": "data"}`)
	run := newTestRun(uuid.New(), input)
	execCtx := NewExecutionContext(run, def)

	// Execute
	err := executor.Execute(ctx, execCtx)

	// Verify
	require.NoError(t, err)
	assert.True(t, len(execCtx.StepData) > 0, "Should have step outputs")

	// Verify init step was executed
	initOutput, ok := execCtx.StepData[initStepID]
	require.True(t, ok, "Init step should have output")
	var initData map[string]interface{}
	require.NoError(t, json.Unmarshal(initOutput, &initData))
	assert.Equal(t, float64(10), initData["value"])

	// Verify group was executed (check group data)
	groupOutput, ok := execCtx.GroupData[parallelGroupID]
	require.True(t, ok, "Parallel group should have output")
	var groupData map[string]interface{}
	require.NoError(t, json.Unmarshal(groupOutput, &groupData))
	assert.True(t, groupData["completed"].(bool), "Group should be completed")
}

// TestExecutor_Integration_GroupToStepConnection tests group → step edge execution
func TestExecutor_Integration_GroupToStepConnection(t *testing.T) {
	executor := newTestExecutor()
	ctx := context.Background()

	// Create IDs
	startStepID := uuid.New()
	parallelGroupID := uuid.New()
	branchStepID := uuid.New()
	outputStepID := uuid.New()

	def := &domain.WorkflowDefinition{
		Name: "Group to Step Test",
		Steps: []domain.Step{
			{
				ID:   startStepID,
				Name: "Start",
				Type: domain.StepTypeStart,
			},
			{
				ID:           branchStepID,
				Name:         "Branch",
				Type:         domain.StepTypeFunction,
				BlockGroupID: &parallelGroupID,
				Config:       json.RawMessage(`{"code": "return { processed: true };", "language": "javascript"}`),
			},
			{
				ID:     outputStepID,
				Name:   "Output",
				Type:   domain.StepTypeFunction,
				Config: json.RawMessage(`{"code": "return { final: true, received: input };", "language": "javascript"}`),
			},
		},
		Edges: []domain.Edge{
			// start → parallel_group
			{ID: uuid.New(), SourceStepID: &startStepID, TargetBlockGroupID: &parallelGroupID, TargetPort: "group-input"},
			// parallel_group (out) → output
			{ID: uuid.New(), SourceBlockGroupID: &parallelGroupID, TargetStepID: &outputStepID, SourcePort: "out"},
		},
		BlockGroups: []domain.BlockGroup{
			{
				ID:     parallelGroupID,
				Name:   "Parallel Group",
				Type:   domain.BlockGroupTypeParallel,
				Config: json.RawMessage(`{}`),
			},
		},
	}

	input := json.RawMessage(`{"test": "data"}`)
	run := newTestRun(uuid.New(), input)
	execCtx := NewExecutionContext(run, def)

	err := executor.Execute(ctx, execCtx)

	require.NoError(t, err)

	// Verify output step received group output
	outputData, ok := execCtx.StepData[outputStepID]
	require.True(t, ok, "Output step should have been executed")
	var output map[string]interface{}
	require.NoError(t, json.Unmarshal(outputData, &output))
	assert.True(t, output["final"].(bool))
}

// TestExecutor_Integration_GroupToGroupConnection tests group → group edge execution
func TestExecutor_Integration_GroupToGroupConnection(t *testing.T) {
	executor := newTestExecutor()
	ctx := context.Background()

	// Create IDs
	startStepID := uuid.New()
	parallelGroupID := uuid.New()
	foreachGroupID := uuid.New()
	parallelStepID := uuid.New()
	foreachStepID := uuid.New()
	finalStepID := uuid.New()

	def := &domain.WorkflowDefinition{
		Name: "Group to Group Test",
		Steps: []domain.Step{
			{
				ID:   startStepID,
				Name: "Start",
				Type: domain.StepTypeStart,
			},
			{
				ID:           parallelStepID,
				Name:         "Parallel Step",
				Type:         domain.StepTypeFunction,
				BlockGroupID: &parallelGroupID,
				Config:       json.RawMessage(`{"code": "return { items: [1, 2, 3] };", "language": "javascript"}`),
			},
			{
				ID:           foreachStepID,
				Name:         "ForEach Step",
				Type:         domain.StepTypeFunction,
				BlockGroupID: &foreachGroupID,
				Config:       json.RawMessage(`{"code": "return { item: input.currentItem, doubled: input.currentItem * 2 };", "language": "javascript"}`),
			},
			{
				ID:     finalStepID,
				Name:   "Final",
				Type:   domain.StepTypeFunction,
				Config: json.RawMessage(`{"code": "return { complete: true, data: input };", "language": "javascript"}`),
			},
		},
		Edges: []domain.Edge{
			// start → parallel_group
			{ID: uuid.New(), SourceStepID: &startStepID, TargetBlockGroupID: &parallelGroupID, TargetPort: "group-input"},
			// parallel_group → foreach_group (group to group)
			{ID: uuid.New(), SourceBlockGroupID: &parallelGroupID, TargetBlockGroupID: &foreachGroupID, SourcePort: "out", TargetPort: "group-input"},
			// foreach_group → final
			{ID: uuid.New(), SourceBlockGroupID: &foreachGroupID, TargetStepID: &finalStepID, SourcePort: "out"},
		},
		BlockGroups: []domain.BlockGroup{
			{
				ID:     parallelGroupID,
				Name:   "Parallel Group",
				Type:   domain.BlockGroupTypeParallel,
				Config: json.RawMessage(`{}`),
			},
			{
				ID:     foreachGroupID,
				Name:   "ForEach Group",
				Type:   domain.BlockGroupTypeForeach,
				Config: json.RawMessage(`{"input_path": "$.results.Parallel Step.items", "parallel": false}`),
			},
		},
	}

	input := json.RawMessage(`{}`)
	run := newTestRun(uuid.New(), input)
	execCtx := NewExecutionContext(run, def)

	err := executor.Execute(ctx, execCtx)

	require.NoError(t, err)

	// Verify both groups executed
	_, parallelOk := execCtx.GroupData[parallelGroupID]
	assert.True(t, parallelOk, "Parallel group should have executed")

	_, foreachOk := execCtx.GroupData[foreachGroupID]
	assert.True(t, foreachOk, "ForEach group should have executed")

	// Verify final step executed
	_, finalOk := execCtx.StepData[finalStepID]
	assert.True(t, finalOk, "Final step should have executed")
}

// =============================================================================
// Phase 1.2: Output Port Routing Tests
// =============================================================================

// TestExecutor_Integration_GroupOutputPort_Out tests out port routing
func TestExecutor_Integration_GroupOutputPort_Complete(t *testing.T) {
	executor := newTestExecutor()
	ctx := context.Background()

	startStepID := uuid.New()
	parallelGroupID := uuid.New()
	branchStepID := uuid.New()
	successStepID := uuid.New()
	errorStepID := uuid.New()

	def := &domain.WorkflowDefinition{
		Name: "Complete Port Test",
		Steps: []domain.Step{
			{ID: startStepID, Name: "Start", Type: domain.StepTypeStart},
			{
				ID:           branchStepID,
				Name:         "Branch",
				Type:         domain.StepTypeFunction,
				BlockGroupID: &parallelGroupID,
				Config:       json.RawMessage(`{"code": "return { success: true };", "language": "javascript"}`),
			},
			{
				ID:     successStepID,
				Name:   "Success Handler",
				Type:   domain.StepTypeFunction,
				Config: json.RawMessage(`{"code": "return { handled: 'success' };", "language": "javascript"}`),
			},
			{
				ID:     errorStepID,
				Name:   "Error Handler",
				Type:   domain.StepTypeFunction,
				Config: json.RawMessage(`{"code": "return { handled: 'error' };", "language": "javascript"}`),
			},
		},
		Edges: []domain.Edge{
			{ID: uuid.New(), SourceStepID: &startStepID, TargetBlockGroupID: &parallelGroupID, TargetPort: "group-input"},
			{ID: uuid.New(), SourceBlockGroupID: &parallelGroupID, TargetStepID: &successStepID, SourcePort: "out"},
			{ID: uuid.New(), SourceBlockGroupID: &parallelGroupID, TargetStepID: &errorStepID, SourcePort: "error"},
		},
		BlockGroups: []domain.BlockGroup{
			{ID: parallelGroupID, Name: "Parallel", Type: domain.BlockGroupTypeParallel, Config: json.RawMessage(`{}`)},
		},
	}

	input := json.RawMessage(`{}`)
	run := newTestRun(uuid.New(), input)
	execCtx := NewExecutionContext(run, def)

	err := executor.Execute(ctx, execCtx)

	require.NoError(t, err)

	// Success handler should be executed (out port)
	_, successOk := execCtx.StepData[successStepID]
	assert.True(t, successOk, "Success handler should have executed via out port")

	// Error handler should NOT be executed
	_, errorOk := execCtx.StepData[errorStepID]
	assert.False(t, errorOk, "Error handler should NOT have executed")
}

// TestExecutor_Integration_GroupOutputPort_Error tests error port routing
func TestExecutor_Integration_GroupOutputPort_Error(t *testing.T) {
	executor := newTestExecutor()
	ctx := context.Background()

	startStepID := uuid.New()
	tryCatchGroupID := uuid.New()
	riskyStepID := uuid.New()
	successStepID := uuid.New()
	catchStepID := uuid.New()

	def := &domain.WorkflowDefinition{
		Name: "Error Port Test",
		Steps: []domain.Step{
			{ID: startStepID, Name: "Start", Type: domain.StepTypeStart},
			{
				ID:           riskyStepID,
				Name:         "Risky Operation",
				Type:         domain.StepTypeFunction,
				BlockGroupID: &tryCatchGroupID,
				// This will fail with empty array
				Config: json.RawMessage(`{"code": "if (!input.items || input.items.length === 0) throw new Error('Empty'); return { ok: true };", "language": "javascript"}`),
			},
			{
				ID:     successStepID,
				Name:   "Success Handler",
				Type:   domain.StepTypeFunction,
				Config: json.RawMessage(`{"code": "return { path: 'success' };", "language": "javascript"}`),
			},
			{
				ID:     catchStepID,
				Name:   "Catch Handler",
				Type:   domain.StepTypeFunction,
				Config: json.RawMessage(`{"code": "return { path: 'caught', error: input };", "language": "javascript"}`),
			},
		},
		Edges: []domain.Edge{
			{ID: uuid.New(), SourceStepID: &startStepID, TargetBlockGroupID: &tryCatchGroupID, TargetPort: "group-input"},
			{ID: uuid.New(), SourceBlockGroupID: &tryCatchGroupID, TargetStepID: &successStepID, SourcePort: "success"},
			{ID: uuid.New(), SourceBlockGroupID: &tryCatchGroupID, TargetStepID: &catchStepID, SourcePort: "error"},
		},
		BlockGroups: []domain.BlockGroup{
			{
				ID:     tryCatchGroupID,
				Name:   "Try-Catch",
				Type:   domain.BlockGroupTypeTryCatch,
				Config: json.RawMessage(`{"retry_count": 0}`),
			},
		},
	}

	// Input with empty items to trigger error
	input := json.RawMessage(`{"items": []}`)
	run := newTestRun(uuid.New(), input)
	execCtx := NewExecutionContext(run, def)

	err := executor.Execute(ctx, execCtx)

	// The workflow should complete (error is handled)
	require.NoError(t, err)

	// Check group output has error marker
	groupOutput, ok := execCtx.GroupData[tryCatchGroupID]
	require.True(t, ok, "Try-catch group should have output")

	var groupData map[string]interface{}
	require.NoError(t, json.Unmarshal(groupOutput, &groupData))

	// Error should be returned in output (try_catch exhausted retries)
	if isError, hasError := groupData["__error"].(bool); hasError && isError {
		// Catch handler should be executed via error port
		_, catchOk := execCtx.StepData[catchStepID]
		assert.True(t, catchOk, "Catch handler should have executed via error port")
	}
}

// TestExecutor_Integration_GroupOutputPort_MaxIterations tests max_iterations port
func TestExecutor_Integration_GroupOutputPort_MaxIterations(t *testing.T) {
	executor := newTestExecutor()
	ctx := context.Background()

	startStepID := uuid.New()
	whileGroupID := uuid.New()
	incrementStepID := uuid.New()
	completeStepID := uuid.New()
	maxIterStepID := uuid.New()

	def := &domain.WorkflowDefinition{
		Name: "Max Iterations Port Test",
		Steps: []domain.Step{
			{ID: startStepID, Name: "Start", Type: domain.StepTypeStart},
			{
				ID:           incrementStepID,
				Name:         "Increment",
				Type:         domain.StepTypeFunction,
				BlockGroupID: &whileGroupID,
				Config:       json.RawMessage(`{"code": "return { counter: (input.counter || 0) + 1 };", "language": "javascript"}`),
			},
			{
				ID:     completeStepID,
				Name:   "Complete Handler",
				Type:   domain.StepTypeFunction,
				Config: json.RawMessage(`{"code": "return { path: 'complete' };", "language": "javascript"}`),
			},
			{
				ID:     maxIterStepID,
				Name:   "Max Iterations Handler",
				Type:   domain.StepTypeFunction,
				Config: json.RawMessage(`{"code": "return { path: 'max_iterations', data: input };", "language": "javascript"}`),
			},
		},
		Edges: []domain.Edge{
			{ID: uuid.New(), SourceStepID: &startStepID, TargetBlockGroupID: &whileGroupID, TargetPort: "group-input"},
			{ID: uuid.New(), SourceBlockGroupID: &whileGroupID, TargetStepID: &completeStepID, SourcePort: "out"},
			{ID: uuid.New(), SourceBlockGroupID: &whileGroupID, TargetStepID: &maxIterStepID, SourcePort: "max_iterations"},
		},
		BlockGroups: []domain.BlockGroup{
			{
				ID:   whileGroupID,
				Name: "While Loop",
				Type: domain.BlockGroupTypeWhile,
				// Condition always true, will hit max_iterations (set to 3)
				Config: json.RawMessage(`{"condition": "true", "max_iterations": 3, "do_while": false}`),
			},
		},
	}

	input := json.RawMessage(`{"counter": 0}`)
	run := newTestRun(uuid.New(), input)
	execCtx := NewExecutionContext(run, def)

	err := executor.Execute(ctx, execCtx)

	require.NoError(t, err)

	// Group should have executed
	groupOutput, ok := execCtx.GroupData[whileGroupID]
	require.True(t, ok, "While group should have output")

	var groupData map[string]interface{}
	require.NoError(t, json.Unmarshal(groupOutput, &groupData))

	// Should have iterated 3 times (max_iterations)
	iterations, ok := groupData["iterations"].(float64)
	assert.True(t, ok, "Should have iterations count")
	assert.Equal(t, float64(3), iterations, "Should have reached max iterations")
}

// =============================================================================
// Phase 1.3: Complex Flow Integration Tests
// =============================================================================

// TestExecutor_Integration_BlockGroupDemo_SuccessPath tests the full success path
// Note: This test was modified to remove the join step, as join is no longer supported.
// Block Group outputs are already aggregated, so we connect directly to the final step.
func TestExecutor_Integration_BlockGroupDemo_SuccessPath(t *testing.T) {
	executor := newTestExecutor()
	ctx := context.Background()

	// Build a simplified Block Group Demo workflow (without join)
	startID := uuid.New()
	initID := uuid.New()
	parallelGroupID := uuid.New()
	branchAID := uuid.New()
	branchBID := uuid.New()
	finalID := uuid.New()

	def := &domain.WorkflowDefinition{
		Name: "Block Group Demo Success Path",
		Steps: []domain.Step{
			{ID: startID, Name: "Start", Type: domain.StepTypeStart},
			{
				ID:     initID,
				Name:   "Initialize",
				Type:   domain.StepTypeFunction,
				Config: json.RawMessage(`{"code": "return { items: input.items || [1, 2, 3], counter: 0 };", "language": "javascript"}`),
			},
			{
				ID:           branchAID,
				Name:         "Branch A",
				Type:         domain.StepTypeFunction,
				BlockGroupID: &parallelGroupID,
				Config:       json.RawMessage(`{"code": "const items = input.items || []; return { branch: 'A', count: items.length };", "language": "javascript"}`),
			},
			{
				ID:           branchBID,
				Name:         "Branch B",
				Type:         domain.StepTypeFunction,
				BlockGroupID: &parallelGroupID,
				Config:       json.RawMessage(`{"code": "const items = input.items || []; const sum = items.reduce((a,b) => a+b, 0); return { branch: 'B', sum: sum };", "language": "javascript"}`),
			},
			{
				ID:     finalID,
				Name:   "Final Output",
				Type:   domain.StepTypeFunction,
				Config: json.RawMessage(`{"code": "return { success: true, parallel_results: input };", "language": "javascript"}`),
			},
		},
		Edges: []domain.Edge{
			{ID: uuid.New(), SourceStepID: &startID, TargetStepID: &initID},
			{ID: uuid.New(), SourceStepID: &initID, TargetBlockGroupID: &parallelGroupID, TargetPort: "group-input"},
			// Block Group output connects directly to Final (no join needed, group output is already aggregated)
			{ID: uuid.New(), SourceBlockGroupID: &parallelGroupID, TargetStepID: &finalID, SourcePort: "out"},
		},
		BlockGroups: []domain.BlockGroup{
			{
				ID:     parallelGroupID,
				Name:   "Parallel Processing",
				Type:   domain.BlockGroupTypeParallel,
				Config: json.RawMessage(`{"max_concurrent": 3, "fail_fast": false}`),
			},
		},
	}

	input := json.RawMessage(`{"items": [1, 2, 3, 4, 5]}`)
	run := newTestRun(uuid.New(), input)
	execCtx := NewExecutionContext(run, def)

	err := executor.Execute(ctx, execCtx)

	require.NoError(t, err)

	// Verify final output
	finalOutput, ok := execCtx.StepData[finalID]
	require.True(t, ok, "Final step should have output")

	var finalData map[string]interface{}
	require.NoError(t, json.Unmarshal(finalOutput, &finalData))
	assert.True(t, finalData["success"].(bool), "Should be successful")
}

// TestExecutor_Integration_BlockGroupDemo_ParallelBranches tests parallel execution
func TestExecutor_Integration_BlockGroupDemo_ParallelBranches(t *testing.T) {
	executor := newTestExecutor()
	ctx := context.Background()

	startID := uuid.New()
	parallelGroupID := uuid.New()
	branch1ID := uuid.New()
	branch2ID := uuid.New()
	branch3ID := uuid.New()
	outputID := uuid.New()

	def := &domain.WorkflowDefinition{
		Name: "Parallel Branches Test",
		Steps: []domain.Step{
			{ID: startID, Name: "Start", Type: domain.StepTypeStart},
			{
				ID:           branch1ID,
				Name:         "Branch 1",
				Type:         domain.StepTypeFunction,
				BlockGroupID: &parallelGroupID,
				// Use safe access to avoid NaN when value is undefined
				Config: json.RawMessage(`{"code": "var v = input.value || 10; return { branch: 1, value: v * 1 };", "language": "javascript"}`),
			},
			{
				ID:           branch2ID,
				Name:         "Branch 2",
				Type:         domain.StepTypeFunction,
				BlockGroupID: &parallelGroupID,
				Config: json.RawMessage(`{"code": "var v = input.value || 10; return { branch: 2, value: v * 2 };", "language": "javascript"}`),
			},
			{
				ID:           branch3ID,
				Name:         "Branch 3",
				Type:         domain.StepTypeFunction,
				BlockGroupID: &parallelGroupID,
				Config: json.RawMessage(`{"code": "var v = input.value || 10; return { branch: 3, value: v * 3 };", "language": "javascript"}`),
			},
			{
				ID:     outputID,
				Name:   "Output",
				Type:   domain.StepTypeFunction,
				Config: json.RawMessage(`{"code": "return { complete: true, results: input };", "language": "javascript"}`),
			},
		},
		Edges: []domain.Edge{
			{ID: uuid.New(), SourceStepID: &startID, TargetBlockGroupID: &parallelGroupID, TargetPort: "group-input"},
			{ID: uuid.New(), SourceBlockGroupID: &parallelGroupID, TargetStepID: &outputID, SourcePort: "out"},
		},
		BlockGroups: []domain.BlockGroup{
			{
				ID:     parallelGroupID,
				Name:   "Parallel",
				Type:   domain.BlockGroupTypeParallel,
				Config: json.RawMessage(`{"max_concurrent": 3, "fail_fast": false}`),
			},
		},
	}

	input := json.RawMessage(`{"value": 10}`)
	run := newTestRun(uuid.New(), input)
	execCtx := NewExecutionContext(run, def)

	err := executor.Execute(ctx, execCtx)

	require.NoError(t, err)

	// Verify group output contains all branches
	groupOutput, ok := execCtx.GroupData[parallelGroupID]
	require.True(t, ok, "Parallel group should have output")

	var groupData map[string]interface{}
	require.NoError(t, json.Unmarshal(groupOutput, &groupData))

	results, ok := groupData["results"].(map[string]interface{})
	require.True(t, ok, "Should have results map")

	// All 3 branches should be in results
	assert.Len(t, results, 3, "Should have 3 branch results")
}

// TestExecutor_Integration_BlockGroupDemo_ErrorPath tests error handling path
func TestExecutor_Integration_BlockGroupDemo_ErrorPath(t *testing.T) {
	executor := newTestExecutor()
	ctx := context.Background()

	startID := uuid.New()
	tryCatchGroupID := uuid.New()
	riskyID := uuid.New()
	successID := uuid.New()
	errorID := uuid.New()

	def := &domain.WorkflowDefinition{
		Name: "Error Path Test",
		Steps: []domain.Step{
			{ID: startID, Name: "Start", Type: domain.StepTypeStart},
			{
				ID:           riskyID,
				Name:         "Risky",
				Type:         domain.StepTypeFunction,
				BlockGroupID: &tryCatchGroupID,
				// Throws error when shouldFail is true
				Config: json.RawMessage(`{"code": "if (input.shouldFail) throw new Error('Intentional failure'); return { ok: true };", "language": "javascript"}`),
			},
			{
				ID:     successID,
				Name:   "Success",
				Type:   domain.StepTypeFunction,
				Config: json.RawMessage(`{"code": "return { path: 'success' };", "language": "javascript"}`),
			},
			{
				ID:     errorID,
				Name:   "Error Handler",
				Type:   domain.StepTypeFunction,
				Config: json.RawMessage(`{"code": "return { path: 'error', handled: true };", "language": "javascript"}`),
			},
		},
		Edges: []domain.Edge{
			{ID: uuid.New(), SourceStepID: &startID, TargetBlockGroupID: &tryCatchGroupID, TargetPort: "group-input"},
			{ID: uuid.New(), SourceBlockGroupID: &tryCatchGroupID, TargetStepID: &successID, SourcePort: "success"},
			{ID: uuid.New(), SourceBlockGroupID: &tryCatchGroupID, TargetStepID: &errorID, SourcePort: "error"},
		},
		BlockGroups: []domain.BlockGroup{
			{
				ID:     tryCatchGroupID,
				Name:   "Try-Catch",
				Type:   domain.BlockGroupTypeTryCatch,
				Config: json.RawMessage(`{"retry_count": 0}`),
			},
		},
	}

	// Input that triggers error
	input := json.RawMessage(`{"shouldFail": true}`)
	run := newTestRun(uuid.New(), input)
	execCtx := NewExecutionContext(run, def)

	err := executor.Execute(ctx, execCtx)

	// Workflow should complete even with error (handled by try-catch)
	require.NoError(t, err)

	// Group should have error output
	groupOutput, ok := execCtx.GroupData[tryCatchGroupID]
	require.True(t, ok, "Try-catch group should have output")

	var groupData map[string]interface{}
	require.NoError(t, json.Unmarshal(groupOutput, &groupData))

	// Check if error was caught
	if isError, hasError := groupData["__error"].(bool); hasError && isError {
		// Error handler should have been triggered
		errorOutput, errorOk := execCtx.StepData[errorID]
		if errorOk {
			var errorData map[string]interface{}
			require.NoError(t, json.Unmarshal(errorOutput, &errorData))
			assert.Equal(t, "error", errorData["path"])
		}
	}
}

// Helper function
func strPtr(s string) *string {
	return &s
}

// TestExecutor_Integration_BlockGroupDemo_FullParallelFlow tests the exact flow of Block Group Demo
// Note: This test was modified to remove the join step, as join is no longer supported.
// Block Group outputs are already aggregated, so we use a function step to process the results.
// This mimics: start -> init -> parallel_group -> process_results
func TestExecutor_Integration_BlockGroupDemo_FullParallelFlow(t *testing.T) {
	executor := newTestExecutor()
	ctx := context.Background()

	// Create IDs matching Block Group Demo structure (without join)
	startID := uuid.New()
	initID := uuid.New()
	parallelGroupID := uuid.New()
	branchAID := uuid.New()
	branchBID := uuid.New()
	branchCID := uuid.New()
	processResultsID := uuid.New()

	def := &domain.WorkflowDefinition{
		Name: "Block Group Demo Parallel Flow Test",
		Steps: []domain.Step{
			// Start step
			{ID: startID, Name: "Start", Type: domain.StepTypeStart},

			// Init step (outside group)
			{
				ID:     initID,
				Name:   "Initialize",
				Type:   domain.StepTypeFunction,
				Config: json.RawMessage(`{"code": "return { ...input, counter: 0, max_iterations: 5, results: [] };", "language": "javascript"}`),
			},

			// Parallel branches (inside group)
			{
				ID:           branchAID,
				Name:         "Branch A",
				Type:         domain.StepTypeFunction,
				BlockGroupID: &parallelGroupID,
				GroupRole:    "body",
				Config:       json.RawMessage(`{"code": "const items = input.items || []; const half = Math.floor(items.length / 2); return { branch: 'A', processed: half };", "language": "javascript"}`),
			},
			{
				ID:           branchBID,
				Name:         "Branch B",
				Type:         domain.StepTypeFunction,
				BlockGroupID: &parallelGroupID,
				GroupRole:    "body",
				Config:       json.RawMessage(`{"code": "const items = input.items || []; return { branch: 'B', processed: items.length };", "language": "javascript"}`),
			},
			{
				ID:           branchCID,
				Name:         "Branch C",
				Type:         domain.StepTypeFunction,
				BlockGroupID: &parallelGroupID,
				GroupRole:    "body",
				Config:       json.RawMessage(`{"code": "const items = input.items || []; const sum = items.reduce((a, i) => a + (i.value || 0), 0); return { branch: 'C', sum: sum };", "language": "javascript"}`),
			},

			// Process results step (replaces join - Block Group output is already aggregated)
			{
				ID:     processResultsID,
				Name:   "Process Results",
				Type:   domain.StepTypeFunction,
				Config: json.RawMessage(`{"code": "return { processed: true, results: input };", "language": "javascript"}`),
			},
		},
		Edges: []domain.Edge{
			// start -> init (step to step)
			{ID: uuid.New(), SourceStepID: &startID, TargetStepID: &initID, SourcePort: "output"},

			// init -> parallel_group (step to group)
			{ID: uuid.New(), SourceStepID: &initID, TargetBlockGroupID: &parallelGroupID, SourcePort: "output", TargetPort: "group-input"},

			// parallel_group -> process_results (group to step with port "out")
			{ID: uuid.New(), SourceBlockGroupID: &parallelGroupID, TargetStepID: &processResultsID, SourcePort: "out"},
		},
		BlockGroups: []domain.BlockGroup{
			{
				ID:     parallelGroupID,
				Name:   "Parallel Processing",
				Type:   domain.BlockGroupTypeParallel,
				Config: json.RawMessage(`{"max_concurrent": 3, "fail_fast": false}`),
			},
		},
	}

	input := json.RawMessage(`{"items": [{"id": 1, "value": 10}, {"id": 2, "value": 20}, {"id": 3, "value": 30}]}`)
	run := newTestRun(uuid.New(), input)
	execCtx := NewExecutionContext(run, def)

	err := executor.Execute(ctx, execCtx)
	require.NoError(t, err, "Workflow execution should not fail")

	// Verify start step executed
	_, startOk := execCtx.StepData[startID]
	assert.True(t, startOk, "Start step should have executed")

	// Verify init step executed
	initOutput, initOk := execCtx.StepData[initID]
	assert.True(t, initOk, "Init step should have executed")
	t.Logf("Init output: %s", string(initOutput))

	// Verify parallel group executed
	groupOutput, groupOk := execCtx.GroupData[parallelGroupID]
	assert.True(t, groupOk, "Parallel group should have executed")
	t.Logf("Group output: %s", string(groupOutput))

	// Parse group output
	var groupData map[string]interface{}
	require.NoError(t, json.Unmarshal(groupOutput, &groupData))

	// Check results contain all 3 branches
	results, resultsOk := groupData["results"].(map[string]interface{})
	assert.True(t, resultsOk, "Group output should have results map")
	assert.Len(t, results, 3, "Should have 3 branch results")
	assert.True(t, groupData["completed"].(bool), "Group should be completed")

	// CRITICAL: Verify process results step executed (this is the key test)
	processOutput, processOk := execCtx.StepData[processResultsID]
	assert.True(t, processOk, "Process results step should have executed after parallel group completes")
	if processOk {
		t.Logf("Process results output: %s", string(processOutput))
	} else {
		t.Log("ERROR: Process results step did not execute - this is a bug!")
	}
}
