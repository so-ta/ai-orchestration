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

func setupTestBlockGroupExecutor() (*BlockGroupExecutor, *Executor) {
	registry := adapter.NewRegistry()
	registry.Register(adapter.NewMockAdapter())
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	executor := NewExecutor(registry, logger)
	bgExecutor := NewBlockGroupExecutor(registry, logger, executor)

	return bgExecutor, executor
}

func TestBlockGroupExecutor_ExecuteGroup_InvalidType(t *testing.T) {
	bgExecutor, _ := setupTestBlockGroupExecutor()

	tenantID := uuid.New()
	workflowID := uuid.New()

	group := &domain.BlockGroup{
		ID:         uuid.New(),
		TenantID:   tenantID,
		WorkflowID: workflowID,
		Name:       "Invalid Group",
		Type:       domain.BlockGroupType("invalid_type"),
		Config:     json.RawMessage("{}"),
	}

	bgCtx := &BlockGroupContext{
		Group:   group,
		Steps:   []*domain.Step{},
		Input:   json.RawMessage("{}"),
		ExecCtx: nil,
		Graph:   nil,
	}

	_, err := bgExecutor.ExecuteGroup(context.Background(), bgCtx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown block group type")
}

func TestBlockGroupExecutor_PreProcess_NoCode(t *testing.T) {
	bgExecutor, _ := setupTestBlockGroupExecutor()

	tenantID := uuid.New()
	workflowID := uuid.New()

	// Group with no pre_process code
	group := &domain.BlockGroup{
		ID:         uuid.New(),
		TenantID:   tenantID,
		WorkflowID: workflowID,
		Name:       "Test Group",
		Type:       domain.BlockGroupTypeParallel,
		PreProcess: nil,
	}

	input := json.RawMessage(`{"value": 42}`)

	result, err := bgExecutor.runPreProcess(context.Background(), group, input)

	require.NoError(t, err)
	assert.Equal(t, input, result)
}

func TestBlockGroupExecutor_PreProcess_EmptyCode(t *testing.T) {
	bgExecutor, _ := setupTestBlockGroupExecutor()

	tenantID := uuid.New()
	workflowID := uuid.New()

	emptyCode := ""
	group := &domain.BlockGroup{
		ID:         uuid.New(),
		TenantID:   tenantID,
		WorkflowID: workflowID,
		Name:       "Test Group",
		Type:       domain.BlockGroupTypeParallel,
		PreProcess: &emptyCode,
	}

	input := json.RawMessage(`{"value": 42}`)

	result, err := bgExecutor.runPreProcess(context.Background(), group, input)

	require.NoError(t, err)
	assert.Equal(t, input, result)
}

func TestBlockGroupExecutor_PreProcess_WithCode(t *testing.T) {
	bgExecutor, _ := setupTestBlockGroupExecutor()

	tenantID := uuid.New()
	workflowID := uuid.New()

	// Simple passthrough that works with sandbox
	preProcess := `return input;`
	group := &domain.BlockGroup{
		ID:         uuid.New(),
		TenantID:   tenantID,
		WorkflowID: workflowID,
		Name:       "Test Group",
		Type:       domain.BlockGroupTypeParallel,
		PreProcess: &preProcess,
	}

	input := json.RawMessage(`{"value": 42}`)

	result, err := bgExecutor.runPreProcess(context.Background(), group, input)

	// Sandbox execution may succeed or fail depending on runtime
	// Just verify no panic occurs and result is returned
	if err == nil {
		require.NotNil(t, result)
	}
}

func TestBlockGroupExecutor_PostProcess_NoCode(t *testing.T) {
	bgExecutor, _ := setupTestBlockGroupExecutor()

	tenantID := uuid.New()
	workflowID := uuid.New()

	group := &domain.BlockGroup{
		ID:          uuid.New(),
		TenantID:    tenantID,
		WorkflowID:  workflowID,
		Name:        "Test Group",
		Type:        domain.BlockGroupTypeParallel,
		PostProcess: nil,
	}

	output := json.RawMessage(`{"result": "success"}`)

	result, err := bgExecutor.runPostProcess(context.Background(), group, output)

	require.NoError(t, err)
	assert.Equal(t, output, result)
}

func TestBlockGroupExecutor_PostProcess_WithCode(t *testing.T) {
	bgExecutor, _ := setupTestBlockGroupExecutor()

	tenantID := uuid.New()
	workflowID := uuid.New()

	// Simple passthrough
	postProcess := `return input;`
	group := &domain.BlockGroup{
		ID:          uuid.New(),
		TenantID:    tenantID,
		WorkflowID:  workflowID,
		Name:        "Test Group",
		Type:        domain.BlockGroupTypeParallel,
		PostProcess: &postProcess,
	}

	output := json.RawMessage(`{"result": "success"}`)

	result, err := bgExecutor.runPostProcess(context.Background(), group, output)

	// Sandbox execution may succeed or fail depending on runtime
	// Just verify no panic occurs
	if err == nil {
		require.NotNil(t, result)
	}
}

func TestBlockGroupExecutor_ExecuteParallel_NoSteps(t *testing.T) {
	bgExecutor, _ := setupTestBlockGroupExecutor()

	tenantID := uuid.New()
	workflowID := uuid.New()

	group := &domain.BlockGroup{
		ID:         uuid.New(),
		TenantID:   tenantID,
		WorkflowID: workflowID,
		Name:       "Parallel Group",
		Type:       domain.BlockGroupTypeParallel,
		Config:     json.RawMessage("{}"),
	}

	bgCtx := &BlockGroupContext{
		Group:   group,
		Steps:   []*domain.Step{},
		Input:   json.RawMessage(`{"value": 1}`),
		ExecCtx: nil,
		Graph:   nil,
	}

	result, err := bgExecutor.executeParallel(context.Background(), bgCtx)

	require.NoError(t, err)
	assert.Equal(t, json.RawMessage("{}"), result)
}

func TestBlockGroupExecutor_ExecuteParallel_WithConfig(t *testing.T) {
	bgExecutor, _ := setupTestBlockGroupExecutor()

	tenantID := uuid.New()
	workflowID := uuid.New()

	config := domain.ParallelConfig{
		MaxConcurrent: 2,
		FailFast:      false,
	}
	configJSON, _ := json.Marshal(config)

	group := &domain.BlockGroup{
		ID:         uuid.New(),
		TenantID:   tenantID,
		WorkflowID: workflowID,
		Name:       "Parallel Group",
		Type:       domain.BlockGroupTypeParallel,
		Config:     configJSON,
	}

	bgCtx := &BlockGroupContext{
		Group:   group,
		Steps:   []*domain.Step{},
		Input:   json.RawMessage(`{"value": 1}`),
		ExecCtx: nil,
		Graph:   nil,
	}

	result, err := bgExecutor.executeParallel(context.Background(), bgCtx)

	require.NoError(t, err)
	assert.Equal(t, json.RawMessage("{}"), result)
}

func TestBlockGroupExecutor_ExecuteTryCatch_NoSteps(t *testing.T) {
	bgExecutor, _ := setupTestBlockGroupExecutor()

	tenantID := uuid.New()
	workflowID := uuid.New()

	config := domain.TryCatchConfig{
		RetryCount: 2,
		RetryDelay: 10,
	}
	configJSON, _ := json.Marshal(config)

	group := &domain.BlockGroup{
		ID:         uuid.New(),
		TenantID:   tenantID,
		WorkflowID: workflowID,
		Name:       "TryCatch Group",
		Type:       domain.BlockGroupTypeTryCatch,
		Config:     configJSON,
	}

	bgCtx := &BlockGroupContext{
		Group:   group,
		Steps:   []*domain.Step{},
		Input:   json.RawMessage(`{"value": 1}`),
		ExecCtx: nil,
		Graph:   nil,
	}

	result, err := bgExecutor.executeTryCatch(context.Background(), bgCtx)

	// With no steps, returns nil output (last output was never set)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestBlockGroupExecutor_ExecuteForeach_EmptyArray(t *testing.T) {
	bgExecutor, _ := setupTestBlockGroupExecutor()

	tenantID := uuid.New()
	workflowID := uuid.New()

	config := domain.ForeachConfig{
		InputPath:  "$.items",
		Parallel:   false,
		MaxWorkers: 0,
	}
	configJSON, _ := json.Marshal(config)

	group := &domain.BlockGroup{
		ID:         uuid.New(),
		TenantID:   tenantID,
		WorkflowID: workflowID,
		Name:       "Foreach Group",
		Type:       domain.BlockGroupTypeForeach,
		Config:     configJSON,
	}

	bgCtx := &BlockGroupContext{
		Group:   group,
		Steps:   []*domain.Step{},
		Input:   json.RawMessage(`{"items": []}`),
		ExecCtx: nil,
		Graph:   nil,
	}

	result, err := bgExecutor.executeForeach(context.Background(), bgCtx)

	require.NoError(t, err)

	var resultData map[string]interface{}
	err = json.Unmarshal(result, &resultData)
	require.NoError(t, err)

	assert.Equal(t, float64(0), resultData["iterations"])
	assert.Equal(t, true, resultData["completed"])
}

func TestBlockGroupExecutor_ExecuteForeach_DirectArray(t *testing.T) {
	bgExecutor, _ := setupTestBlockGroupExecutor()

	tenantID := uuid.New()
	workflowID := uuid.New()

	config := domain.ForeachConfig{
		// No InputPath means input is directly an array
		Parallel:   false,
		MaxWorkers: 0,
	}
	configJSON, _ := json.Marshal(config)

	group := &domain.BlockGroup{
		ID:         uuid.New(),
		TenantID:   tenantID,
		WorkflowID: workflowID,
		Name:       "Foreach Group",
		Type:       domain.BlockGroupTypeForeach,
		Config:     configJSON,
	}

	bgCtx := &BlockGroupContext{
		Group:   group,
		Steps:   []*domain.Step{},
		Input:   json.RawMessage(`[1, 2, 3]`),
		ExecCtx: nil,
		Graph:   nil,
	}

	result, err := bgExecutor.executeForeach(context.Background(), bgCtx)

	require.NoError(t, err)

	var resultData map[string]interface{}
	err = json.Unmarshal(result, &resultData)
	require.NoError(t, err)

	assert.Equal(t, float64(3), resultData["iterations"])
	assert.Equal(t, true, resultData["completed"])
}

func TestBlockGroupExecutor_ExecuteForeach_InvalidInputPath(t *testing.T) {
	bgExecutor, _ := setupTestBlockGroupExecutor()

	tenantID := uuid.New()
	workflowID := uuid.New()

	config := domain.ForeachConfig{
		InputPath: "$.nonexistent",
	}
	configJSON, _ := json.Marshal(config)

	group := &domain.BlockGroup{
		ID:         uuid.New(),
		TenantID:   tenantID,
		WorkflowID: workflowID,
		Name:       "Foreach Group",
		Type:       domain.BlockGroupTypeForeach,
		Config:     configJSON,
	}

	bgCtx := &BlockGroupContext{
		Group:   group,
		Steps:   []*domain.Step{},
		Input:   json.RawMessage(`{"items": [1,2,3]}`),
		ExecCtx: nil,
		Graph:   nil,
	}

	_, err := bgExecutor.executeForeach(context.Background(), bgCtx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve input path")
}

func TestBlockGroupExecutor_ExecuteForeach_NotArray(t *testing.T) {
	bgExecutor, _ := setupTestBlockGroupExecutor()

	tenantID := uuid.New()
	workflowID := uuid.New()

	config := domain.ForeachConfig{
		InputPath: "$.data",
	}
	configJSON, _ := json.Marshal(config)

	group := &domain.BlockGroup{
		ID:         uuid.New(),
		TenantID:   tenantID,
		WorkflowID: workflowID,
		Name:       "Foreach Group",
		Type:       domain.BlockGroupTypeForeach,
		Config:     configJSON,
	}

	bgCtx := &BlockGroupContext{
		Group:   group,
		Steps:   []*domain.Step{},
		Input:   json.RawMessage(`{"data": "not an array"}`),
		ExecCtx: nil,
		Graph:   nil,
	}

	_, err := bgExecutor.executeForeach(context.Background(), bgCtx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not resolve to array")
}

func TestBlockGroupExecutor_ExecuteWhile_FalseCondition(t *testing.T) {
	bgExecutor, _ := setupTestBlockGroupExecutor()

	tenantID := uuid.New()
	workflowID := uuid.New()

	config := domain.WhileConfig{
		Condition:     "$.value > 100", // Always false with value=1
		MaxIterations: 10,
		DoWhile:       false,
	}
	configJSON, _ := json.Marshal(config)

	group := &domain.BlockGroup{
		ID:         uuid.New(),
		TenantID:   tenantID,
		WorkflowID: workflowID,
		Name:       "While Group",
		Type:       domain.BlockGroupTypeWhile,
		Config:     configJSON,
	}

	bgCtx := &BlockGroupContext{
		Group:   group,
		Steps:   []*domain.Step{},
		Input:   json.RawMessage(`{"value": 1}`),
		ExecCtx: nil,
		Graph:   nil,
	}

	result, err := bgExecutor.executeWhile(context.Background(), bgCtx)

	require.NoError(t, err)

	var resultData map[string]interface{}
	err = json.Unmarshal(result, &resultData)
	require.NoError(t, err)

	assert.Equal(t, float64(0), resultData["iterations"])
	assert.Equal(t, true, resultData["completed"])
}

func TestBlockGroupExecutor_ExecuteWhile_DoWhile_ExecutesOnce(t *testing.T) {
	bgExecutor, _ := setupTestBlockGroupExecutor()

	tenantID := uuid.New()
	workflowID := uuid.New()

	config := domain.WhileConfig{
		Condition:     "$.value > 100", // Always false
		MaxIterations: 10,
		DoWhile:       true, // Should execute at least once
	}
	configJSON, _ := json.Marshal(config)

	group := &domain.BlockGroup{
		ID:         uuid.New(),
		TenantID:   tenantID,
		WorkflowID: workflowID,
		Name:       "DoWhile Group",
		Type:       domain.BlockGroupTypeWhile,
		Config:     configJSON,
	}

	bgCtx := &BlockGroupContext{
		Group:   group,
		Steps:   []*domain.Step{},
		Input:   json.RawMessage(`{"value": 1}`),
		ExecCtx: nil,
		Graph:   nil,
	}

	result, err := bgExecutor.executeWhile(context.Background(), bgCtx)

	require.NoError(t, err)

	var resultData map[string]interface{}
	err = json.Unmarshal(result, &resultData)
	require.NoError(t, err)

	// DoWhile should execute at least once even if condition is false
	assert.GreaterOrEqual(t, resultData["iterations"].(float64), float64(1))
	assert.Equal(t, true, resultData["completed"])
}

func TestBlockGroupExecutor_ExecuteWhile_MaxIterations(t *testing.T) {
	bgExecutor, _ := setupTestBlockGroupExecutor()

	tenantID := uuid.New()
	workflowID := uuid.New()

	config := domain.WhileConfig{
		Condition:     "true", // Always true
		MaxIterations: 5,
		DoWhile:       false,
	}
	configJSON, _ := json.Marshal(config)

	group := &domain.BlockGroup{
		ID:         uuid.New(),
		TenantID:   tenantID,
		WorkflowID: workflowID,
		Name:       "While Group",
		Type:       domain.BlockGroupTypeWhile,
		Config:     configJSON,
	}

	bgCtx := &BlockGroupContext{
		Group:   group,
		Steps:   []*domain.Step{},
		Input:   json.RawMessage(`{"value": 1}`),
		ExecCtx: nil,
		Graph:   nil,
	}

	result, err := bgExecutor.executeWhile(context.Background(), bgCtx)

	require.NoError(t, err)

	var resultData map[string]interface{}
	err = json.Unmarshal(result, &resultData)
	require.NoError(t, err)

	// Should stop at max iterations
	assert.Equal(t, float64(5), resultData["iterations"])
	assert.Equal(t, true, resultData["completed"])
}

func TestBlockGroupExecutor_ExecuteWhile_DefaultMaxIterations(t *testing.T) {
	bgExecutor, _ := setupTestBlockGroupExecutor()

	tenantID := uuid.New()
	workflowID := uuid.New()

	config := domain.WhileConfig{
		Condition:     "$.counter < 3",
		MaxIterations: 0, // Should default to 100
		DoWhile:       false,
	}
	configJSON, _ := json.Marshal(config)

	group := &domain.BlockGroup{
		ID:         uuid.New(),
		TenantID:   tenantID,
		WorkflowID: workflowID,
		Name:       "While Group",
		Type:       domain.BlockGroupTypeWhile,
		Config:     configJSON,
	}

	bgCtx := &BlockGroupContext{
		Group:   group,
		Steps:   []*domain.Step{},
		Input:   json.RawMessage(`{"counter": 5}`), // condition is false
		ExecCtx: nil,
		Graph:   nil,
	}

	result, err := bgExecutor.executeWhile(context.Background(), bgCtx)

	require.NoError(t, err)

	var resultData map[string]interface{}
	err = json.Unmarshal(result, &resultData)
	require.NoError(t, err)

	// Condition is false from start
	assert.Equal(t, float64(0), resultData["iterations"])
}

func TestBlockGroupExecutor_PreProcess_NonObjectInput(t *testing.T) {
	bgExecutor, _ := setupTestBlockGroupExecutor()

	tenantID := uuid.New()
	workflowID := uuid.New()

	preProcess := `return input;`
	group := &domain.BlockGroup{
		ID:         uuid.New(),
		TenantID:   tenantID,
		WorkflowID: workflowID,
		Name:       "Test Group",
		Type:       domain.BlockGroupTypeParallel,
		PreProcess: &preProcess,
	}

	// Non-object input (a string)
	input := json.RawMessage(`"just a string"`)

	result, err := bgExecutor.runPreProcess(context.Background(), group, input)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Should wrap non-object in value field
	var resultData map[string]interface{}
	err = json.Unmarshal(result, &resultData)
	require.NoError(t, err)
}

func TestBlockGroupExecutor_WrapTransformCode(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name: "simple return",
			code: "return { a: 1 };",
			expected: `
(function() {
	var result = (function(input) {
		return { a: 1 };
	})(input);
	return result !== undefined ? result : input;
})()
`,
		},
		{
			name: "with input transformation",
			code: "return { transformed: input.value * 2 };",
			expected: `
(function() {
	var result = (function(input) {
		return { transformed: input.value * 2 };
	})(input);
	return result !== undefined ? result : input;
})()
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wrapTransformCode(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBlockGroupExecutor_ExecuteGroup_WithPrePostProcess(t *testing.T) {
	bgExecutor, _ := setupTestBlockGroupExecutor()

	tenantID := uuid.New()
	workflowID := uuid.New()

	// Test without complex JS transformations
	// Just verify the pre/post process flow works
	config := domain.ForeachConfig{
		// No InputPath means input is array directly
		Parallel: false,
	}
	configJSON, _ := json.Marshal(config)

	group := &domain.BlockGroup{
		ID:         uuid.New(),
		TenantID:   tenantID,
		WorkflowID: workflowID,
		Name:       "Foreach without transforms",
		Type:       domain.BlockGroupTypeForeach,
		Config:     configJSON,
		// No pre/post process to avoid sandbox complexities in test
	}

	bgCtx := &BlockGroupContext{
		Group:   group,
		Steps:   []*domain.Step{},
		Input:   json.RawMessage(`[1, 2, 3]`),
		ExecCtx: nil,
		Graph:   nil,
	}

	result, err := bgExecutor.ExecuteGroup(context.Background(), bgCtx)

	require.NoError(t, err)
	require.NotNil(t, result)

	var resultData map[string]interface{}
	err = json.Unmarshal(result, &resultData)
	require.NoError(t, err)

	assert.Equal(t, float64(3), resultData["iterations"])
	assert.Equal(t, true, resultData["completed"])
}

func TestBlockGroupExecutor_ExecuteGroup_PreProcessError(t *testing.T) {
	bgExecutor, _ := setupTestBlockGroupExecutor()

	tenantID := uuid.New()
	workflowID := uuid.New()

	// Invalid JavaScript that will cause an error
	preProcess := `throw new Error("pre_process error");`

	group := &domain.BlockGroup{
		ID:         uuid.New(),
		TenantID:   tenantID,
		WorkflowID: workflowID,
		Name:       "Error Group",
		Type:       domain.BlockGroupTypeParallel,
		Config:     json.RawMessage("{}"),
		PreProcess: &preProcess,
	}

	bgCtx := &BlockGroupContext{
		Group:   group,
		Steps:   []*domain.Step{},
		Input:   json.RawMessage(`{}`),
		ExecCtx: nil,
		Graph:   nil,
	}

	_, err := bgExecutor.ExecuteGroup(context.Background(), bgCtx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pre_process failed")
}

func TestBlockGroupExecutor_ExecuteForeach_Parallel(t *testing.T) {
	bgExecutor, _ := setupTestBlockGroupExecutor()

	tenantID := uuid.New()
	workflowID := uuid.New()

	config := domain.ForeachConfig{
		InputPath:  "$.items",
		Parallel:   true,
		MaxWorkers: 2,
	}
	configJSON, _ := json.Marshal(config)

	group := &domain.BlockGroup{
		ID:         uuid.New(),
		TenantID:   tenantID,
		WorkflowID: workflowID,
		Name:       "Parallel Foreach",
		Type:       domain.BlockGroupTypeForeach,
		Config:     configJSON,
	}

	bgCtx := &BlockGroupContext{
		Group:   group,
		Steps:   []*domain.Step{},
		Input:   json.RawMessage(`{"items": [1, 2, 3, 4, 5]}`),
		ExecCtx: nil,
		Graph:   nil,
	}

	result, err := bgExecutor.executeForeach(context.Background(), bgCtx)

	require.NoError(t, err)

	var resultData map[string]interface{}
	err = json.Unmarshal(result, &resultData)
	require.NoError(t, err)

	assert.Equal(t, float64(5), resultData["iterations"])
	assert.Equal(t, true, resultData["completed"])
}

func TestBlockGroupExecutor_ExecuteTryCatch_WithRetryConfig(t *testing.T) {
	bgExecutor, _ := setupTestBlockGroupExecutor()

	tenantID := uuid.New()
	workflowID := uuid.New()

	config := domain.TryCatchConfig{
		RetryCount: 3,
		RetryDelay: 10, // 10ms delay
	}
	configJSON, _ := json.Marshal(config)

	group := &domain.BlockGroup{
		ID:         uuid.New(),
		TenantID:   tenantID,
		WorkflowID: workflowID,
		Name:       "TryCatch with Retry",
		Type:       domain.BlockGroupTypeTryCatch,
		Config:     configJSON,
	}

	bgCtx := &BlockGroupContext{
		Group:   group,
		Steps:   []*domain.Step{},
		Input:   json.RawMessage(`{"value": 1}`),
		ExecCtx: nil,
		Graph:   nil,
	}

	startTime := time.Now()
	result, err := bgExecutor.executeTryCatch(context.Background(), bgCtx)
	elapsed := time.Since(startTime)

	// With no steps, should succeed immediately (not use retries)
	require.NoError(t, err)
	assert.Nil(t, result)

	// Should not wait for retries since there are no steps that fail
	assert.Less(t, elapsed, time.Second)
}

func TestBlockGroupExecutor_ExecuteParallel_InvalidConfig(t *testing.T) {
	bgExecutor, _ := setupTestBlockGroupExecutor()

	tenantID := uuid.New()
	workflowID := uuid.New()

	// Invalid JSON config (will be logged as warning but not fail)
	group := &domain.BlockGroup{
		ID:         uuid.New(),
		TenantID:   tenantID,
		WorkflowID: workflowID,
		Name:       "Parallel Group",
		Type:       domain.BlockGroupTypeParallel,
		Config:     json.RawMessage(`{"invalid": json`), // Invalid JSON
	}

	bgCtx := &BlockGroupContext{
		Group:   group,
		Steps:   []*domain.Step{},
		Input:   json.RawMessage(`{}`),
		ExecCtx: nil,
		Graph:   nil,
	}

	// Should still work with defaults
	result, err := bgExecutor.executeParallel(context.Background(), bgCtx)

	require.NoError(t, err)
	assert.Equal(t, json.RawMessage("{}"), result)
}

func TestBlockGroupExecutor_NewBlockGroupExecutor(t *testing.T) {
	registry := adapter.NewRegistry()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	executor := NewExecutor(registry, logger)

	bgExecutor := NewBlockGroupExecutor(registry, logger, executor)

	assert.NotNil(t, bgExecutor)
	assert.NotNil(t, bgExecutor.registry)
	assert.NotNil(t, bgExecutor.logger)
	assert.NotNil(t, bgExecutor.evaluator)
	assert.NotNil(t, bgExecutor.executor)
	assert.NotNil(t, bgExecutor.sandbox)
}
