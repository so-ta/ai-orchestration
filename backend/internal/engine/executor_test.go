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

func setupTestExecutor() *Executor {
	registry := adapter.NewRegistry()
	registry.Register(adapter.NewMockAdapter())
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewExecutor(registry, logger)
}

func TestExecutor_ExecuteMapStep_Simple(t *testing.T) {
	executor := setupTestExecutor()

	step := domain.Step{
		ID:   uuid.New(),
		Name: "map-test",
		Type: domain.StepTypeMap,
		Config: json.RawMessage(`{
			"input_path": "$.items"
		}`),
	}

	input := json.RawMessage(`{
		"items": [1, 2, 3, 4, 5]
	}`)

	output, err := executor.executeMapStep(context.Background(), step, input)

	require.NoError(t, err)
	require.NotNil(t, output)

	var result map[string]interface{}
	err = json.Unmarshal(output, &result)
	require.NoError(t, err)

	assert.Equal(t, float64(5), result["count"])
	assert.Equal(t, true, result["mapped"])

	items := result["items"].([]interface{})
	assert.Len(t, items, 5)
}

func TestExecutor_ExecuteMapStep_WithAdapter(t *testing.T) {
	executor := setupTestExecutor()

	step := domain.Step{
		ID:   uuid.New(),
		Name: "map-with-adapter",
		Type: domain.StepTypeMap,
		Config: json.RawMessage(`{
			"input_path": "$.items",
			"adapter_id": "mock",
			"parallel": false
		}`),
	}

	input := json.RawMessage(`{
		"items": [{"value": 1}, {"value": 2}]
	}`)

	output, err := executor.executeMapStep(context.Background(), step, input)

	require.NoError(t, err)
	require.NotNil(t, output)

	var result map[string]interface{}
	err = json.Unmarshal(output, &result)
	require.NoError(t, err)

	assert.Equal(t, float64(2), result["count"])
	assert.Equal(t, float64(2), result["success_count"])
	assert.Equal(t, float64(0), result["error_count"])
}

func TestExecutor_ExecuteMapStep_Parallel(t *testing.T) {
	executor := setupTestExecutor()

	step := domain.Step{
		ID:   uuid.New(),
		Name: "map-parallel",
		Type: domain.StepTypeMap,
		Config: json.RawMessage(`{
			"input_path": "$.items",
			"adapter_id": "mock",
			"parallel": true,
			"max_workers": 5
		}`),
	}

	// Create array with 10 items
	items := make([]map[string]int, 10)
	for i := 0; i < 10; i++ {
		items[i] = map[string]int{"value": i}
	}
	inputData := map[string]interface{}{"items": items}
	input, _ := json.Marshal(inputData)

	output, err := executor.executeMapStep(context.Background(), step, input)

	require.NoError(t, err)
	require.NotNil(t, output)

	var result map[string]interface{}
	err = json.Unmarshal(output, &result)
	require.NoError(t, err)

	assert.Equal(t, float64(10), result["count"])
	assert.Equal(t, float64(10), result["success_count"])
}

func TestExecutor_ExecuteMapStep_InvalidPath(t *testing.T) {
	executor := setupTestExecutor()

	step := domain.Step{
		ID:   uuid.New(),
		Name: "map-invalid",
		Type: domain.StepTypeMap,
		Config: json.RawMessage(`{
			"input_path": "$.nonexistent"
		}`),
	}

	input := json.RawMessage(`{"other": "data"}`)

	output, err := executor.executeMapStep(context.Background(), step, input)

	assert.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "failed to resolve input path")
}

func TestExecutor_ExecuteMapStep_NotArray(t *testing.T) {
	executor := setupTestExecutor()

	step := domain.Step{
		ID:   uuid.New(),
		Name: "map-not-array",
		Type: domain.StepTypeMap,
		Config: json.RawMessage(`{
			"input_path": "$.data"
		}`),
	}

	input := json.RawMessage(`{"data": "not an array"}`)

	output, err := executor.executeMapStep(context.Background(), step, input)

	assert.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "does not resolve to an array")
}

func TestExecutor_ExecuteMapStep_DirectArray(t *testing.T) {
	executor := setupTestExecutor()

	step := domain.Step{
		ID:   uuid.New(),
		Name: "map-direct",
		Type: domain.StepTypeMap,
		Config: json.RawMessage(`{}`), // No input_path, use input directly
	}

	input := json.RawMessage(`[1, 2, 3]`)

	output, err := executor.executeMapStep(context.Background(), step, input)

	require.NoError(t, err)
	require.NotNil(t, output)

	var result map[string]interface{}
	err = json.Unmarshal(output, &result)
	require.NoError(t, err)

	assert.Equal(t, float64(3), result["count"])
}

func TestExecutor_ExecuteConditionStep(t *testing.T) {
	executor := setupTestExecutor()

	run := &domain.Run{
		ID:         uuid.New(),
		WorkflowID: uuid.New(),
	}
	def := &domain.WorkflowDefinition{Name: "test"}
	execCtx := NewExecutionContext(run, def)

	step := domain.Step{
		ID:   uuid.New(),
		Name: "condition-test",
		Type: domain.StepTypeCondition,
		Config: json.RawMessage(`{
			"expression": "$.value > 10"
		}`),
	}

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "condition true",
			input:    `{"value": 15}`,
			expected: true,
		},
		{
			name:     "condition false",
			input:    `{"value": 5}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executor.executeConditionStep(context.Background(), execCtx, step, json.RawMessage(tt.input))

			require.NoError(t, err)
			require.NotNil(t, output)

			var result map[string]interface{}
			err = json.Unmarshal(output, &result)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, result["result"])
		})
	}
}

func TestExecutor_ExecuteJoinStep(t *testing.T) {
	executor := setupTestExecutor()

	run := &domain.Run{
		ID:         uuid.New(),
		WorkflowID: uuid.New(),
	}
	def := &domain.WorkflowDefinition{Name: "test"}
	execCtx := NewExecutionContext(run, def)

	// Add some step data
	step1ID := uuid.New()
	step2ID := uuid.New()
	execCtx.StepData[step1ID] = json.RawMessage(`{"a": 1}`)
	execCtx.StepData[step2ID] = json.RawMessage(`{"b": 2}`)

	joinStep := domain.Step{
		ID:     uuid.New(),
		Name:   "join-test",
		Type:   domain.StepTypeJoin,
		Config: json.RawMessage(`{}`),
	}

	output, err := executor.executeJoinStep(context.Background(), execCtx, joinStep)

	require.NoError(t, err)
	require.NotNil(t, output)

	var result map[string]interface{}
	err = json.Unmarshal(output, &result)
	require.NoError(t, err)

	// Should have both step outputs
	assert.Contains(t, result, step1ID.String())
	assert.Contains(t, result, step2ID.String())
}

// Tests for new step types (Loop, Wait, Function, Router, HumanInLoop)

func TestExecutor_ExecuteLoopStep_ForLoop(t *testing.T) {
	executor := setupTestExecutor()

	step := domain.Step{
		ID:   uuid.New(),
		Name: "test-loop",
		Type: domain.StepTypeLoop,
		Config: json.RawMessage(`{
			"loop_type": "for",
			"count": 3
		}`),
	}

	input := json.RawMessage(`{"value": "test"}`)
	output, err := executor.executeLoopStep(context.Background(), step, input)

	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(output, &result)
	require.NoError(t, err)

	assert.Equal(t, float64(3), result["iterations"])
	assert.Equal(t, true, result["completed"])
	assert.Len(t, result["results"].([]interface{}), 3)
}

func TestExecutor_ExecuteLoopStep_ForEachLoop(t *testing.T) {
	executor := setupTestExecutor()

	step := domain.Step{
		ID:   uuid.New(),
		Name: "test-foreach",
		Type: domain.StepTypeLoop,
		Config: json.RawMessage(`{
			"loop_type": "forEach",
			"input_path": "$.items"
		}`),
	}

	input := json.RawMessage(`{"items": ["a", "b", "c", "d"]}`)
	output, err := executor.executeLoopStep(context.Background(), step, input)

	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(output, &result)
	require.NoError(t, err)

	assert.Equal(t, float64(4), result["iterations"])
	assert.Equal(t, true, result["completed"])
}

func TestExecutor_ExecuteLoopStep_WhileLoop(t *testing.T) {
	executor := setupTestExecutor()

	step := domain.Step{
		ID:   uuid.New(),
		Name: "test-while",
		Type: domain.StepTypeLoop,
		Config: json.RawMessage(`{
			"loop_type": "while",
			"condition": "$.index < 3",
			"max_iterations": 10
		}`),
	}

	input := json.RawMessage(`{}`)
	output, err := executor.executeLoopStep(context.Background(), step, input)

	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(output, &result)
	require.NoError(t, err)

	assert.Equal(t, float64(3), result["iterations"])
	assert.Equal(t, true, result["completed"])
}

func TestExecutor_ExecuteLoopStep_DoWhileLoop(t *testing.T) {
	executor := setupTestExecutor()

	step := domain.Step{
		ID:   uuid.New(),
		Name: "test-dowhile",
		Type: domain.StepTypeLoop,
		Config: json.RawMessage(`{
			"loop_type": "doWhile",
			"condition": "$.index < 2",
			"max_iterations": 10
		}`),
	}

	input := json.RawMessage(`{}`)
	output, err := executor.executeLoopStep(context.Background(), step, input)

	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(output, &result)
	require.NoError(t, err)

	// DoWhile executes at least once
	assert.GreaterOrEqual(t, result["iterations"].(float64), float64(1))
	assert.Equal(t, true, result["completed"])
}

func TestExecutor_ExecuteLoopStep_MaxIterations(t *testing.T) {
	executor := setupTestExecutor()

	step := domain.Step{
		ID:   uuid.New(),
		Name: "test-max-iter",
		Type: domain.StepTypeLoop,
		Config: json.RawMessage(`{
			"loop_type": "for",
			"count": 1000,
			"max_iterations": 5
		}`),
	}

	input := json.RawMessage(`{}`)
	output, err := executor.executeLoopStep(context.Background(), step, input)

	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(output, &result)
	require.NoError(t, err)

	// Should be capped at max_iterations
	assert.Equal(t, float64(5), result["iterations"])
}

func TestExecutor_ExecuteLoopStep_WithAdapter(t *testing.T) {
	executor := setupTestExecutor()

	step := domain.Step{
		ID:   uuid.New(),
		Name: "test-loop-adapter",
		Type: domain.StepTypeLoop,
		Config: json.RawMessage(`{
			"loop_type": "for",
			"count": 2,
			"adapter_id": "mock"
		}`),
	}

	input := json.RawMessage(`{"value": "test"}`)
	output, err := executor.executeLoopStep(context.Background(), step, input)

	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(output, &result)
	require.NoError(t, err)

	assert.Equal(t, float64(2), result["iterations"])
	results := result["results"].([]interface{})
	assert.Len(t, results, 2)
}

func TestExecutor_ExecuteWaitStep_Duration(t *testing.T) {
	// Save original timeAfter and restore after test
	originalTimeAfter := timeAfter
	defer func() { timeAfter = originalTimeAfter }()

	// Mock timeAfter to return immediately
	timeAfter = func(ms int64) <-chan time.Time {
		ch := make(chan time.Time, 1)
		ch <- time.Now()
		return ch
	}

	executor := setupTestExecutor()

	step := domain.Step{
		ID:   uuid.New(),
		Name: "test-wait",
		Type: domain.StepTypeWait,
		Config: json.RawMessage(`{
			"duration_ms": 100
		}`),
	}

	input := json.RawMessage(`{"key": "value"}`)
	output, err := executor.executeWaitStep(context.Background(), step, input)

	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(output, &result)
	require.NoError(t, err)

	assert.Equal(t, float64(100), result["waited_ms"])
}

func TestExecutor_ExecuteWaitStep_Until(t *testing.T) {
	// Save original functions and restore after test
	originalTimeAfter := timeAfter
	originalTimeNow := timeNow
	defer func() {
		timeAfter = originalTimeAfter
		timeNow = originalTimeNow
	}()

	// Mock time functions
	fixedNow := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	timeNow = func() time.Time { return fixedNow }
	timeAfter = func(ms int64) <-chan time.Time {
		ch := make(chan time.Time, 1)
		ch <- time.Now()
		return ch
	}

	executor := setupTestExecutor()

	// Set until time 1 hour in the future
	until := fixedNow.Add(1 * time.Hour).Format(time.RFC3339)

	step := domain.Step{
		ID:   uuid.New(),
		Name: "test-wait-until",
		Type: domain.StepTypeWait,
		Config: json.RawMessage(`{
			"until": "` + until + `"
		}`),
	}

	input := json.RawMessage(`{}`)
	output, err := executor.executeWaitStep(context.Background(), step, input)

	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(output, &result)
	require.NoError(t, err)

	// Should wait for 1 hour = 3600000 ms
	assert.Equal(t, float64(3600000), result["waited_ms"])
}

func TestExecutor_ExecuteWaitStep_MaxDuration(t *testing.T) {
	// Save original functions
	originalTimeAfter := timeAfter
	defer func() { timeAfter = originalTimeAfter }()

	timeAfter = func(ms int64) <-chan time.Time {
		ch := make(chan time.Time, 1)
		ch <- time.Now()
		return ch
	}

	executor := setupTestExecutor()

	step := domain.Step{
		ID:   uuid.New(),
		Name: "test-wait-max",
		Type: domain.StepTypeWait,
		Config: json.RawMessage(`{
			"duration_ms": 9999999999
		}`),
	}

	input := json.RawMessage(`{}`)
	output, err := executor.executeWaitStep(context.Background(), step, input)

	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(output, &result)
	require.NoError(t, err)

	// Should be capped at 1 hour (3600000 ms)
	assert.Equal(t, float64(3600000), result["waited_ms"])
}

func TestExecutor_ExecuteWaitStep_ContextCancelled(t *testing.T) {
	executor := setupTestExecutor()

	step := domain.Step{
		ID:   uuid.New(),
		Name: "test-wait-cancel",
		Type: domain.StepTypeWait,
		Config: json.RawMessage(`{
			"duration_ms": 10000
		}`),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	input := json.RawMessage(`{}`)
	_, err := executor.executeWaitStep(ctx, step, input)

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestExecutor_ExecuteFunctionStep(t *testing.T) {
	executor := setupTestExecutor()

	step := domain.Step{
		ID:   uuid.New(),
		Name: "test-function",
		Type: domain.StepTypeFunction,
		Config: json.RawMessage(`{
			"code": "return { result: input.value * 2 }",
			"language": "javascript"
		}`),
	}

	input := json.RawMessage(`{"value": 5}`)
	output, err := executor.executeFunctionStep(context.Background(), nil, step, input)

	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(output, &result)
	require.NoError(t, err)

	// Function should execute and return calculated result
	assert.EqualValues(t, 10, result["result"])
}

func TestExecutor_ExecuteFunctionStep_WithExecuteFunction(t *testing.T) {
	executor := setupTestExecutor()

	step := domain.Step{
		ID:   uuid.New(),
		Name: "test-function-with-execute",
		Type: domain.StepTypeFunction,
		Config: json.RawMessage(`{
			"code": "function execute(input, context) { return { greeting: 'Hello, ' + input.name + '!' }; }"
		}`),
	}

	input := json.RawMessage(`{"name": "World"}`)
	output, err := executor.executeFunctionStep(context.Background(), nil, step, input)

	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(output, &result)
	require.NoError(t, err)

	assert.Equal(t, "Hello, World!", result["greeting"])
}

func TestExecutor_ExecuteFunctionStep_UnsupportedLanguage(t *testing.T) {
	executor := setupTestExecutor()

	step := domain.Step{
		ID:   uuid.New(),
		Name: "test-function-python",
		Type: domain.StepTypeFunction,
		Config: json.RawMessage(`{
			"code": "print('hello')",
			"language": "python"
		}`),
	}

	input := json.RawMessage(`{}`)
	_, err := executor.executeFunctionStep(context.Background(), nil, step, input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported language")
}

func TestExecutor_ExecuteRouterStep_NoAdapter(t *testing.T) {
	executor := setupTestExecutor()

	step := domain.Step{
		ID:   uuid.New(),
		Name: "test-router",
		Type: domain.StepTypeRouter,
		Config: json.RawMessage(`{
			"routes": [
				{"name": "route-a", "description": "Handle case A"},
				{"name": "route-b", "description": "Handle case B"}
			],
			"provider": "nonexistent"
		}`),
	}

	input := json.RawMessage(`{"query": "test input"}`)
	output, err := executor.executeRouterStep(context.Background(), step, input)

	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(output, &result)
	require.NoError(t, err)

	// Should fallback to first route
	assert.Equal(t, "route-a", result["selected_route"])
	assert.Equal(t, true, result["fallback"])
}

func TestExecutor_ExecuteRouterStep_NoRoutes(t *testing.T) {
	executor := setupTestExecutor()

	step := domain.Step{
		ID:   uuid.New(),
		Name: "test-router-empty",
		Type: domain.StepTypeRouter,
		Config: json.RawMessage(`{
			"routes": []
		}`),
	}

	input := json.RawMessage(`{}`)
	_, err := executor.executeRouterStep(context.Background(), step, input)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no routes defined")
}

func TestExecutor_ExecuteRouterStep_WithMockAdapter(t *testing.T) {
	executor := setupTestExecutor()

	step := domain.Step{
		ID:   uuid.New(),
		Name: "test-router-mock",
		Type: domain.StepTypeRouter,
		Config: json.RawMessage(`{
			"routes": [
				{"name": "support", "description": "Customer support requests"},
				{"name": "sales", "description": "Sales inquiries"}
			],
			"provider": "mock"
		}`),
	}

	input := json.RawMessage(`{"query": "I need help with my order"}`)
	output, err := executor.executeRouterStep(context.Background(), step, input)

	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(output, &result)
	require.NoError(t, err)

	// Mock adapter returns generic response, should select first route
	assert.NotEmpty(t, result["selected_route"])
}

func TestExecutor_ExecuteHumanInLoopStep_TestMode(t *testing.T) {
	executor := setupTestExecutor()

	step := domain.Step{
		ID:   uuid.New(),
		Name: "test-human-in-loop",
		Type: domain.StepTypeHumanInLoop,
		Config: json.RawMessage(`{
			"instructions": "Please review this data",
			"timeout_hours": 24
		}`),
	}

	run := &domain.Run{
		ID:          uuid.New(),
		TriggeredBy: domain.TriggerTypeTest,
	}
	execCtx := NewExecutionContext(run, nil)

	input := json.RawMessage(`{"data": "to review"}`)
	output, err := executor.executeHumanInLoopStep(context.Background(), execCtx, step, input)

	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(output, &result)
	require.NoError(t, err)

	// In test mode, should be auto-approved
	assert.Equal(t, "approved", result["status"])
	assert.Equal(t, true, result["auto_approved"])
	assert.Equal(t, "Please review this data", result["instructions"])
}

func TestExecutor_ExecuteHumanInLoopStep_ProductionMode(t *testing.T) {
	executor := setupTestExecutor()

	step := domain.Step{
		ID:   uuid.New(),
		Name: "test-human-in-loop-prod",
		Type: domain.StepTypeHumanInLoop,
		Config: json.RawMessage(`{
			"instructions": "Approve this action",
			"required_fields": [
				{"name": "approved", "type": "boolean", "required": true}
			]
		}`),
	}

	run := &domain.Run{
		ID:          uuid.New(),
		TriggeredBy: domain.TriggerTypeManual,
	}
	execCtx := NewExecutionContext(run, nil)

	input := json.RawMessage(`{}`)
	output, err := executor.executeHumanInLoopStep(context.Background(), execCtx, step, input)

	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(output, &result)
	require.NoError(t, err)

	// In production mode, should be pending
	assert.Equal(t, "pending", result["status"])
	assert.Equal(t, false, result["auto_approved"])
	assert.NotEmpty(t, result["approval_id"])
	assert.NotEmpty(t, result["approval_url"])
}

func TestParseISO8601(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Time
		hasError bool
	}{
		{
			input:    "2024-01-15T10:30:00Z",
			expected: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			hasError: false,
		},
		{
			input:    "2024-01-15T10:30:00",
			expected: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			hasError: false,
		},
		{
			input:    "2024-01-15",
			expected: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			hasError: false,
		},
		{
			input:    "invalid-date",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseISO8601(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestStringContains(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"hello world", "world", true},
		{"HELLO WORLD", "world", true},
		{"hello world", "WORLD", true},
		{"hello", "hello world", false},
		{"", "test", false},
		{"test", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			got := stringContains(tt.s, tt.substr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExecutor_FullWorkflowWithNewSteps(t *testing.T) {
	executor := setupTestExecutor()

	// Save original timeAfter
	originalTimeAfter := timeAfter
	defer func() { timeAfter = originalTimeAfter }()
	timeAfter = func(ms int64) <-chan time.Time {
		ch := make(chan time.Time, 1)
		ch <- time.Now()
		return ch
	}

	workflowID := uuid.New()

	// Create a start node - required entry point
	startStep := domain.Step{
		ID:         uuid.New(),
		WorkflowID: workflowID,
		Name:       "start",
		Type:       domain.StepTypeStart,
		Config:     json.RawMessage(`{}`),
	}

	// Create a workflow with multiple new step types
	loopStep := domain.Step{
		ID:         uuid.New(),
		WorkflowID: workflowID,
		Name:       "loop-step",
		Type:       domain.StepTypeLoop,
		Config: json.RawMessage(`{
			"loop_type": "for",
			"count": 2
		}`),
	}

	waitStep := domain.Step{
		ID:         uuid.New(),
		WorkflowID: workflowID,
		Name:       "wait-step",
		Type:       domain.StepTypeWait,
		Config: json.RawMessage(`{
			"duration_ms": 10
		}`),
	}

	def := &domain.WorkflowDefinition{
		Name:  "test-workflow",
		Steps: []domain.Step{startStep, loopStep, waitStep},
		Edges: []domain.Edge{
			{
				ID:           uuid.New(),
				WorkflowID:   workflowID,
				SourceStepID: startStep.ID,
				TargetStepID: loopStep.ID,
			},
			{
				ID:           uuid.New(),
				WorkflowID:   workflowID,
				SourceStepID: loopStep.ID,
				TargetStepID: waitStep.ID,
			},
		},
	}

	run := &domain.Run{
		ID:          uuid.New(),
		WorkflowID:  workflowID,
		TriggeredBy: domain.TriggerTypeTest,
		Input:       json.RawMessage(`{"initial": "data"}`),
	}

	execCtx := NewExecutionContext(run, def)
	err := executor.Execute(context.Background(), execCtx)

	require.NoError(t, err)

	// All three steps should have been executed (start, loop, wait)
	assert.Len(t, execCtx.StepRuns, 3)
	assert.Contains(t, execCtx.StepData, startStep.ID)
	assert.Contains(t, execCtx.StepData, loopStep.ID)
	assert.Contains(t, execCtx.StepData, waitStep.ID)
}
