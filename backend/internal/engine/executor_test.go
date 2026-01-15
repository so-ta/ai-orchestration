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

// Tests for new step types (Wait, Function, Router, HumanInLoop)

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

	// Create a workflow with wait step
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
		Steps: []domain.Step{startStep, waitStep},
		Edges: []domain.Edge{
			{
				ID:           uuid.New(),
				WorkflowID:   workflowID,
				SourceStepID: &startStep.ID,
				TargetStepID: &waitStep.ID,
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

	// Both steps should have been executed (start, wait)
	assert.Len(t, execCtx.StepRuns, 2)
	assert.Contains(t, execCtx.StepData, startStep.ID)
	assert.Contains(t, execCtx.StepData, waitStep.ID)
}

// Tests for JSON error handling branches (Issue #68)

func TestExecutor_PrepareStepInput_UnmarshalFallback(t *testing.T) {
	// Test that prepareStepInput falls back to raw string when unmarshal fails
	executor := setupTestExecutor()

	run := &domain.Run{
		ID:         uuid.New(),
		WorkflowID: uuid.New(),
		Input:      json.RawMessage(`{"initial": "data"}`),
	}
	def := &domain.WorkflowDefinition{Name: "test"}
	execCtx := NewExecutionContext(run, def)

	// Add invalid JSON as step data to trigger unmarshal fallback
	step1ID := uuid.New()
	execCtx.StepData[step1ID] = json.RawMessage(`not valid json`)

	step := domain.Step{
		ID:   uuid.New(),
		Name: "test-step",
		Type: domain.StepTypeStart,
	}

	// prepareStepInput should not fail, it should fall back to raw string
	input, err := executor.prepareStepInput(execCtx, step)

	require.NoError(t, err)
	require.NotNil(t, input)

	// Result should contain the step data as a string fallback
	var result map[string]interface{}
	err = json.Unmarshal(input, &result)
	require.NoError(t, err)

	// The invalid JSON should be stored as a raw string
	assert.Equal(t, "not valid json", result[step1ID.String()])
}

func TestExecutor_ExecuteJoinStep_UnmarshalFallback(t *testing.T) {
	// Test that executeJoinStep falls back to raw string when unmarshal fails
	executor := setupTestExecutor()

	run := &domain.Run{
		ID:         uuid.New(),
		WorkflowID: uuid.New(),
	}
	def := &domain.WorkflowDefinition{Name: "test"}
	execCtx := NewExecutionContext(run, def)

	// Add both valid and invalid JSON as step data
	step1ID := uuid.New()
	step2ID := uuid.New()
	execCtx.StepData[step1ID] = json.RawMessage(`{"valid": "json"}`)
	execCtx.StepData[step2ID] = json.RawMessage(`invalid json data`)

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

	// Valid JSON should be parsed correctly
	validData := result[step1ID.String()].(map[string]interface{})
	assert.Equal(t, "json", validData["valid"])

	// Invalid JSON should fall back to raw string
	assert.Equal(t, "invalid json data", result[step2ID.String()])
}

func TestExecutor_ExecuteMapStep_MarshalUnmarshalFallbacks(t *testing.T) {
	executor := setupTestExecutor()

	tests := []struct {
		name     string
		parallel bool
	}{
		{name: "sequential execution", parallel: false},
		{name: "parallel execution", parallel: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := domain.Step{
				ID:   uuid.New(),
				Name: "map-test",
				Type: domain.StepTypeMap,
				Config: json.RawMessage(`{
					"input_path": "$.items",
					"adapter_id": "mock",
					"parallel": ` + boolToString(tt.parallel) + `
				}`),
			}

			// Include items that will produce output
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
		})
	}
}

func TestExecutor_ExecuteRouterStep_UnmarshalFallback(t *testing.T) {
	// Test that executeRouterStep falls back to empty map when LLM response is invalid JSON
	// This tests the branch at line 1389-1392
	executor := setupTestExecutor()

	step := domain.Step{
		ID:   uuid.New(),
		Name: "test-router-fallback",
		Type: domain.StepTypeRouter,
		Config: json.RawMessage(`{
			"routes": [
				{"name": "route-a", "description": "Handle case A"},
				{"name": "route-b", "description": "Handle case B"}
			],
			"provider": "mock"
		}`),
	}

	input := json.RawMessage(`{"query": "test input"}`)
	output, err := executor.executeRouterStep(context.Background(), step, input)

	require.NoError(t, err)
	require.NotNil(t, output)

	var result map[string]interface{}
	err = json.Unmarshal(output, &result)
	require.NoError(t, err)

	// Should have a selected route (mock adapter returns a valid response)
	assert.NotEmpty(t, result["selected_route"])
}

func TestExecutor_ExecuteFilterStep_ErrorHandling(t *testing.T) {
	// Tests filter step error handling for marshal and evaluation errors
	// This tests the branches at lines 1580-1584 (marshal) and 1585-1591 (evaluation)
	executor := setupTestExecutor()

	tests := []struct {
		name                  string
		expression            string
		input                 string
		expectedOriginalCount float64
		expectedFilteredCount float64
	}{
		{
			name:                  "normal filter with valid expression",
			expression:            "$.value > 0",
			input:                 `[{"value": 1}, {"value": -1}, {"value": 5}]`,
			expectedOriginalCount: 3,
			expectedFilteredCount: 2, // Only items with value > 0
		},
		{
			name:                  "filter skips items when evaluation fails on missing field",
			expression:            "$.nonexistent.field > 0",
			input:                 `[{"value": 1}, {"other": 2}, {"value": 3}]`,
			expectedOriginalCount: 3,
			expectedFilteredCount: 0, // All items skipped due to evaluation error
		},
		{
			name:                  "filter with all items passing condition",
			expression:            "$.value >= 0",
			input:                 `[{"value": 0}, {"value": 1}, {"value": 2}]`,
			expectedOriginalCount: 3,
			expectedFilteredCount: 3,
		},
		{
			name:                  "filter with no items passing condition",
			expression:            "$.value > 100",
			input:                 `[{"value": 1}, {"value": 2}, {"value": 3}]`,
			expectedOriginalCount: 3,
			expectedFilteredCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := domain.Step{
				ID:   uuid.New(),
				Name: "test-filter",
				Type: domain.StepTypeFilter,
				Config: json.RawMessage(`{
					"expression": "` + tt.expression + `"
				}`),
			}

			output, err := executor.executeFilterStep(context.Background(), step, json.RawMessage(tt.input))

			require.NoError(t, err)
			require.NotNil(t, output)

			var result map[string]interface{}
			err = json.Unmarshal(output, &result)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedOriginalCount, result["original_count"])
			assert.Equal(t, tt.expectedFilteredCount, result["filtered_count"])
		})
	}
}

// Helper function
func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// ============================================================================
// Block Definition Execution Tests
// ============================================================================

// Note: executeBlockDefinition uses sandbox.Execute which wraps code.
// The wrapCustomBlockCode in executor.go adds config and renderTemplate setup,
// then sandbox.wrapCode wraps again for IIFE. This is the expected behavior.

func TestExecutor_ExecuteBlockDefinition_PassThrough(t *testing.T) {
	executor := setupTestExecutor()

	tenantID := uuid.New()
	// Block with no code, should pass through input
	blockDef := &domain.BlockDefinition{
		ID:       uuid.New(),
		TenantID: &tenantID,
		Slug:     "passthrough-block",
		Name:     "Passthrough Block",
		Category: domain.BlockCategoryUtility,
		Code:     "", // No code
	}

	step := domain.Step{
		ID:   uuid.New(),
		Name: "test-passthrough",
		Type: "passthrough-block",
	}

	run := &domain.Run{
		ID:       uuid.New(),
		TenantID: tenantID,
	}

	execCtx := NewExecutionContext(run, nil)
	input := json.RawMessage(`{"original": "data", "value": 42}`)

	output, err := executor.executeBlockDefinition(context.Background(), execCtx, step, blockDef, input)

	require.NoError(t, err)
	require.NotNil(t, output, "output should not be nil")

	var result map[string]interface{}
	err = json.Unmarshal(output, &result)
	require.NoError(t, err)

	// Should pass through input data
	assert.Equal(t, "data", result["original"])
	assert.Equal(t, float64(42), result["value"])
}

func TestExecutor_ExecuteBlockDefinition_NoCodePassThrough(t *testing.T) {
	// When a block has no code and no internal steps, input should pass through
	executor := setupTestExecutor()

	tenantID := uuid.New()
	blockDef := &domain.BlockDefinition{
		ID:       uuid.New(),
		TenantID: &tenantID,
		Slug:     "passthrough-block",
		Name:     "Pass Through Block",
		Category: domain.BlockCategoryUtility,
		// No code, no internal steps - input passes through
	}

	step := domain.Step{
		ID:   uuid.New(),
		Name: "test-passthrough",
		Type: "passthrough-block",
	}

	run := &domain.Run{
		ID:       uuid.New(),
		TenantID: tenantID,
	}

	execCtx := NewExecutionContext(run, nil)
	input := json.RawMessage(`{"value": "original", "extra": 123}`)

	output, err := executor.executeBlockDefinition(context.Background(), execCtx, step, blockDef, input)

	require.NoError(t, err)
	require.NotNil(t, output)

	var result map[string]interface{}
	err = json.Unmarshal(output, &result)
	require.NoError(t, err)

	// Input should pass through unchanged
	assert.Equal(t, "original", result["value"])
	assert.Equal(t, float64(123), result["extra"])
}

func TestExecutor_ExecuteBlockDefinition_WithConfigDefaults(t *testing.T) {
	executor := setupTestExecutor()

	tenantID := uuid.New()
	blockDef := &domain.BlockDefinition{
		ID:                     uuid.New(),
		TenantID:               &tenantID,
		Slug:                   "config-defaults-block",
		Name:                   "Config Defaults Block",
		Category:               domain.BlockCategoryUtility,
		ResolvedConfigDefaults: json.RawMessage(`{"multiplier": 10, "prefix": "test_"}`),
	}

	step := domain.Step{
		ID:     uuid.New(),
		Name:   "test-config-defaults",
		Type:   "config-defaults-block",
		Config: json.RawMessage(`{"multiplier": 5}`), // Override multiplier, keep prefix
	}

	run := &domain.Run{
		ID:       uuid.New(),
		TenantID: tenantID,
	}

	execCtx := NewExecutionContext(run, nil)
	input := json.RawMessage(`{"value": 100}`)

	output, err := executor.executeBlockDefinition(context.Background(), execCtx, step, blockDef, input)

	require.NoError(t, err)
	require.NotNil(t, output)

	// Without code, input is passed through
	var result map[string]interface{}
	err = json.Unmarshal(output, &result)
	require.NoError(t, err)

	assert.Equal(t, float64(100), result["value"])
}

func TestExecutor_ExecuteCustomBlockStep_TenantValidation(t *testing.T) {
	executor := setupTestExecutor()

	tenantID := uuid.New()
	otherTenantID := uuid.New()

	// Create a block definition for a different tenant
	blockDef := &domain.BlockDefinition{
		ID:       uuid.New(),
		TenantID: &otherTenantID, // Different tenant
		Slug:     "other-tenant-block",
		Name:     "Other Tenant Block",
		Category: domain.BlockCategoryUtility,
	}

	step := domain.Step{
		ID:                uuid.New(),
		Name:              "test-tenant",
		Type:              "other-tenant-block",
		BlockDefinitionID: &blockDef.ID,
	}

	// Current run belongs to tenantID, not otherTenantID
	run := &domain.Run{
		ID:       uuid.New(),
		TenantID: tenantID,
	}

	execCtx := NewExecutionContext(run, nil)
	input := json.RawMessage(`{}`)

	// executeBlockDefinition itself doesn't check tenant (that's done in executeCustomBlockStep)
	// So this should succeed
	output, err := executor.executeBlockDefinition(context.Background(), execCtx, step, blockDef, input)

	require.NoError(t, err)
	require.NotNil(t, output)
}

func TestExecutor_ExecuteBlockDefinition_NilExecCtx_SystemBlock(t *testing.T) {
	executor := setupTestExecutor()

	// System block (no tenant)
	blockDef := &domain.BlockDefinition{
		ID:       uuid.New(),
		TenantID: nil, // System block
		Slug:     "system-block",
		Name:     "System Block",
		Category: domain.BlockCategoryUtility,
	}

	step := domain.Step{
		ID:   uuid.New(),
		Name: "test-system",
		Type: "system-block",
	}

	input := json.RawMessage(`{"data": "system"}`)

	// System blocks should work even without execCtx (passthrough)
	output, err := executor.executeBlockDefinition(context.Background(), nil, step, blockDef, input)

	require.NoError(t, err)
	require.NotNil(t, output)

	var result map[string]interface{}
	err = json.Unmarshal(output, &result)
	require.NoError(t, err)

	assert.Equal(t, "system", result["data"])
}

func TestExecutor_ExecuteBlockDefinition_EmptyInput(t *testing.T) {
	executor := setupTestExecutor()

	tenantID := uuid.New()
	blockDef := &domain.BlockDefinition{
		ID:       uuid.New(),
		TenantID: &tenantID,
		Slug:     "empty-input-block",
		Name:     "Empty Input Block",
		Category: domain.BlockCategoryUtility,
	}

	step := domain.Step{
		ID:   uuid.New(),
		Name: "test-empty",
		Type: "empty-input-block",
	}

	run := &domain.Run{
		ID:       uuid.New(),
		TenantID: tenantID,
	}

	execCtx := NewExecutionContext(run, nil)
	input := json.RawMessage(`{}`)

	output, err := executor.executeBlockDefinition(context.Background(), execCtx, step, blockDef, input)

	require.NoError(t, err)
	require.NotNil(t, output)

	var result map[string]interface{}
	err = json.Unmarshal(output, &result)
	require.NoError(t, err)

	// Empty input should be passed through
	assert.Empty(t, result)
}

// mockBlockDefinitionGetter is a mock implementation of BlockDefinitionGetter for testing
type mockBlockDefinitionGetter struct {
	blocks map[uuid.UUID]*domain.BlockDefinition
}

func newMockBlockDefinitionGetter() *mockBlockDefinitionGetter {
	return &mockBlockDefinitionGetter{
		blocks: make(map[uuid.UUID]*domain.BlockDefinition),
	}
}

func (m *mockBlockDefinitionGetter) Add(block *domain.BlockDefinition) {
	m.blocks[block.ID] = block
}

func (m *mockBlockDefinitionGetter) GetByID(ctx context.Context, id uuid.UUID) (*domain.BlockDefinition, error) {
	if block, ok := m.blocks[id]; ok {
		return block, nil
	}
	return nil, nil
}

func (m *mockBlockDefinitionGetter) GetBySlug(ctx context.Context, tenantID *uuid.UUID, slug string) (*domain.BlockDefinition, error) {
	for _, block := range m.blocks {
		if block.Slug == slug {
			return block, nil
		}
	}
	return nil, nil
}

func setupTestExecutorWithBlockRepo(repo BlockDefinitionGetter) *Executor {
	registry := adapter.NewRegistry()
	registry.Register(adapter.NewMockAdapter())
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewExecutor(registry, logger, WithBlockDefinitionRepository(repo))
}

func TestExecutor_ExecuteCustomBlockStep_NilExecCtx_TenantBlock_Error(t *testing.T) {
	// Test that tenant-specific blocks fail when execCtx is nil
	mockRepo := newMockBlockDefinitionGetter()

	tenantID := uuid.New()
	blockDef := &domain.BlockDefinition{
		ID:       uuid.New(),
		TenantID: &tenantID, // Tenant-specific block
		Slug:     "tenant-block",
		Name:     "Tenant Block",
		Category: domain.BlockCategoryUtility,
	}
	mockRepo.Add(blockDef)

	executor := setupTestExecutorWithBlockRepo(mockRepo)

	step := domain.Step{
		ID:                uuid.New(),
		Name:              "test-tenant-block-nil-ctx",
		Type:              "tenant-block",
		BlockDefinitionID: &blockDef.ID,
	}

	input := json.RawMessage(`{"data": "test"}`)

	// Call with nil execCtx - should fail for tenant-specific block
	_, err := executor.executeCustomBlockStep(context.Background(), nil, step, input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "tenant-specific block")
	assert.Contains(t, err.Error(), "requires execution context")
	assert.Contains(t, err.Error(), blockDef.Slug) // Error should include block slug
}

func TestExecutor_ExecuteCustomBlockStep_NilRun_TenantBlock_Error(t *testing.T) {
	// Test that tenant-specific blocks fail when execCtx.Run is nil
	mockRepo := newMockBlockDefinitionGetter()

	tenantID := uuid.New()
	blockDef := &domain.BlockDefinition{
		ID:       uuid.New(),
		TenantID: &tenantID, // Tenant-specific block
		Slug:     "tenant-block-nil-run",
		Name:     "Tenant Block Nil Run",
		Category: domain.BlockCategoryUtility,
	}
	mockRepo.Add(blockDef)

	executor := setupTestExecutorWithBlockRepo(mockRepo)

	step := domain.Step{
		ID:                uuid.New(),
		Name:              "test-tenant-block-nil-run",
		Type:              "tenant-block-nil-run",
		BlockDefinitionID: &blockDef.ID,
	}

	input := json.RawMessage(`{"data": "test"}`)

	// Create execCtx with nil Run
	execCtx := &ExecutionContext{
		Run: nil, // Nil run
	}

	// Call with nil Run in execCtx - should fail for tenant-specific block
	_, err := executor.executeCustomBlockStep(context.Background(), execCtx, step, input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "tenant-specific block")
	assert.Contains(t, err.Error(), "requires execution context")
}

func TestExecutor_ExecuteCustomBlockStep_SystemBlock_NilExecCtx_Success(t *testing.T) {
	// Test that system blocks (no tenant) work with nil execCtx
	mockRepo := newMockBlockDefinitionGetter()

	blockDef := &domain.BlockDefinition{
		ID:       uuid.New(),
		TenantID: nil, // System block (no tenant)
		Slug:     "system-block-test",
		Name:     "System Block Test",
		Category: domain.BlockCategoryUtility,
	}
	mockRepo.Add(blockDef)

	executor := setupTestExecutorWithBlockRepo(mockRepo)

	step := domain.Step{
		ID:                uuid.New(),
		Name:              "test-system-block",
		Type:              "system-block-test",
		BlockDefinitionID: &blockDef.ID,
	}

	input := json.RawMessage(`{"data": "system test"}`)

	// System blocks should work even with nil execCtx
	output, err := executor.executeCustomBlockStep(context.Background(), nil, step, input)

	require.NoError(t, err)
	require.NotNil(t, output)

	var result map[string]interface{}
	err = json.Unmarshal(output, &result)
	require.NoError(t, err)

	assert.Equal(t, "system test", result["data"])
}
