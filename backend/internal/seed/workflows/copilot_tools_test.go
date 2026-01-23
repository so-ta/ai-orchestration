package workflows

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/block/sandbox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockStepsService implements sandbox.StepsService for testing
type MockStepsService struct {
	steps     []map[string]interface{}
	projectID string
}

func NewMockStepsService(projectID string) *MockStepsService {
	return &MockStepsService{
		steps:     []map[string]interface{}{},
		projectID: projectID,
	}
}

func (m *MockStepsService) ListByProject(projectID string) ([]map[string]interface{}, error) {
	if projectID != m.projectID {
		return []map[string]interface{}{}, nil
	}
	return m.steps, nil
}

func (m *MockStepsService) Create(data map[string]interface{}) (map[string]interface{}, error) {
	id := uuid.New().String()
	step := map[string]interface{}{
		"id":         id,
		"project_id": m.projectID,
		"name":       data["name"],
		"type":       data["type"],
		"config":     data["config"],
		"position_x": data["position_x"],
		"position_y": data["position_y"],
		"created_at": time.Now().Format(time.RFC3339),
	}
	m.steps = append(m.steps, step)
	return step, nil
}

func (m *MockStepsService) Update(stepID string, updates map[string]interface{}) error {
	return nil
}

func (m *MockStepsService) Delete(stepID string) error {
	return nil
}

// MockEdgesService implements sandbox.EdgesService for testing
type MockEdgesService struct {
	edges     []map[string]interface{}
	projectID string
}

func NewMockEdgesService(projectID string) *MockEdgesService {
	return &MockEdgesService{
		edges:     []map[string]interface{}{},
		projectID: projectID,
	}
}

func (m *MockEdgesService) ListByProject(projectID string) ([]map[string]interface{}, error) {
	if projectID != m.projectID {
		return []map[string]interface{}{}, nil
	}
	return m.edges, nil
}

func (m *MockEdgesService) Create(data map[string]interface{}) (map[string]interface{}, error) {
	// Check for duplicates
	sourceID := data["source_step_id"].(string)
	targetID := data["target_step_id"].(string)
	for _, e := range m.edges {
		if e["source_step_id"] == sourceID && e["target_step_id"] == targetID {
			return map[string]interface{}{
				"error":     "duplicate edge",
				"duplicate": true,
			}, nil
		}
	}

	id := uuid.New().String()
	edge := map[string]interface{}{
		"id":             id,
		"project_id":     m.projectID,
		"source_step_id": sourceID,
		"target_step_id": targetID,
		"source_port":    data["source_port"],
		"created_at":     time.Now().Format(time.RFC3339),
	}
	m.edges = append(m.edges, edge)
	return edge, nil
}

func (m *MockEdgesService) Delete(edgeID string) error {
	return nil
}

// MockWorkflowsService implements sandbox.WorkflowsService for testing
type MockWorkflowsService struct {
	projectID   string
	startStepID string
}

func NewMockWorkflowsService(projectID, startStepID string) *MockWorkflowsService {
	return &MockWorkflowsService{
		projectID:   projectID,
		startStepID: startStepID,
	}
}

func (m *MockWorkflowsService) Get(projectID string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"id":   projectID,
		"name": "Test Workflow",
	}, nil
}

func (m *MockWorkflowsService) List() ([]map[string]interface{}, error) {
	return []map[string]interface{}{
		{"id": m.projectID, "name": "Test Workflow"},
	}, nil
}

func (m *MockWorkflowsService) GetWithStart(projectID string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"id":            projectID,
		"name":          "Test Workflow",
		"start_step_id": m.startStepID,
	}, nil
}

// extractAddStepCode extracts the JavaScript code from addStepToolConfig
func extractAddStepCode() string {
	config := addStepToolConfig()
	var cfg struct {
		Code string `json:"code"`
	}
	json.Unmarshal([]byte(config), &cfg)
	return cfg.Code
}

// extractAddEdgeCode extracts the JavaScript code from addEdgeToolConfig
func extractAddEdgeCode() string {
	config := addEdgeToolConfig()
	var cfg struct {
		Code string `json:"code"`
	}
	json.Unmarshal([]byte(config), &cfg)
	return cfg.Code
}

// TestAddStep_BasicStepCreation tests basic step creation without edges
func TestAddStep_BasicStepCreation(t *testing.T) {
	sb := sandbox.New(sandbox.DefaultConfig())
	projectID := uuid.New().String()
	stepsService := NewMockStepsService(projectID)
	edgesService := NewMockEdgesService(projectID)
	workflowsService := NewMockWorkflowsService(projectID, "")

	execCtx := &sandbox.ExecutionContext{
		Steps:           stepsService,
		Edges:           edgesService,
		Workflows:       workflowsService,
		TargetProjectID: projectID,
	}

	code := extractAddStepCode()
	input := map[string]interface{}{
		"name": "HTTP Request",
		"type": "http",
		"config": map[string]interface{}{
			"url":    "https://api.example.com",
			"method": "GET",
		},
	}

	result, err := sb.Execute(context.Background(), code, input, execCtx)
	require.NoError(t, err)

	// Verify step was created
	assert.NotEmpty(t, result["step_id"])
	assert.True(t, result["step_created"].(bool))

	// Verify no edges were created (no 'from' specified)
	edges := result["edges"]
	if edges != nil {
		edgeList, ok := edges.([]interface{})
		if ok {
			assert.Empty(t, edgeList)
		}
	}

	// Verify step exists in mock service
	assert.Len(t, stepsService.steps, 1)
	assert.Equal(t, "HTTP Request", stepsService.steps[0]["name"])
	assert.Equal(t, "http", stepsService.steps[0]["type"])
}

// TestAddStep_WithFromParameter tests step creation with automatic edge creation
func TestAddStep_WithFromParameter(t *testing.T) {
	sb := sandbox.New(sandbox.DefaultConfig())
	projectID := uuid.New().String()
	stepsService := NewMockStepsService(projectID)
	edgesService := NewMockEdgesService(projectID)
	workflowsService := NewMockWorkflowsService(projectID, "")

	// Pre-create a source step
	sourceStep, _ := stepsService.Create(map[string]interface{}{
		"project_id": projectID,
		"name":       "Source Step",
		"type":       "manual_trigger",
		"config":     map[string]interface{}{},
	})

	execCtx := &sandbox.ExecutionContext{
		Steps:           stepsService,
		Edges:           edgesService,
		Workflows:       workflowsService,
		TargetProjectID: projectID,
	}

	code := extractAddStepCode()
	input := map[string]interface{}{
		"name": "Process Data",
		"type": "function",
		"from": "Source Step", // Connect from source step by name
	}

	result, err := sb.Execute(context.Background(), code, input, execCtx)
	require.NoError(t, err)

	// Verify step was created
	assert.NotEmpty(t, result["step_id"])
	assert.True(t, result["step_created"].(bool))

	// Verify edge was created
	edges := result["edges"].([]interface{})
	require.Len(t, edges, 1)
	edgeInfo := edges[0].(map[string]interface{})
	assert.NotEmpty(t, edgeInfo["edge_id"])
	assert.True(t, edgeInfo["edge_created"].(bool))
	assert.Equal(t, "Source Step", edgeInfo["from"])

	// Verify edge exists in mock service
	assert.Len(t, edgesService.edges, 1)
	assert.Equal(t, sourceStep["id"], edgesService.edges[0]["source_step_id"])
}

// TestAddStep_Idempotency tests that same name returns existing step
func TestAddStep_Idempotency(t *testing.T) {
	sb := sandbox.New(sandbox.DefaultConfig())
	projectID := uuid.New().String()
	stepsService := NewMockStepsService(projectID)
	edgesService := NewMockEdgesService(projectID)
	workflowsService := NewMockWorkflowsService(projectID, "")

	execCtx := &sandbox.ExecutionContext{
		Steps:           stepsService,
		Edges:           edgesService,
		Workflows:       workflowsService,
		TargetProjectID: projectID,
	}

	code := extractAddStepCode()
	input := map[string]interface{}{
		"name": "My Step",
		"type": "http",
	}

	// First call - should create step
	result1, err := sb.Execute(context.Background(), code, input, execCtx)
	require.NoError(t, err)
	assert.True(t, result1["step_created"].(bool))
	stepID := result1["step_id"].(string)

	// Second call with same name - should return existing step
	result2, err := sb.Execute(context.Background(), code, input, execCtx)
	require.NoError(t, err)
	assert.False(t, result2["step_created"].(bool))
	assert.Equal(t, stepID, result2["step_id"])
	assert.Contains(t, result2["message"].(string), "already exists")

	// Verify only one step was created
	assert.Len(t, stepsService.steps, 1)
}

// TestAddStep_SourceNotFound tests error when source step doesn't exist
func TestAddStep_SourceNotFound(t *testing.T) {
	sb := sandbox.New(sandbox.DefaultConfig())
	projectID := uuid.New().String()
	stepsService := NewMockStepsService(projectID)
	edgesService := NewMockEdgesService(projectID)
	workflowsService := NewMockWorkflowsService(projectID, "")

	execCtx := &sandbox.ExecutionContext{
		Steps:           stepsService,
		Edges:           edgesService,
		Workflows:       workflowsService,
		TargetProjectID: projectID,
	}

	code := extractAddStepCode()
	input := map[string]interface{}{
		"name": "Target Step",
		"type": "http",
		"from": "Non-existent Step",
	}

	result, err := sb.Execute(context.Background(), code, input, execCtx)
	require.NoError(t, err)

	// Should return error
	assert.NotEmpty(t, result["error"])
	assert.Contains(t, result["error"].(string), "Source step not found")
}

// TestAddStep_MultipleFromSources tests step creation with multiple input connections
func TestAddStep_MultipleFromSources(t *testing.T) {
	sb := sandbox.New(sandbox.DefaultConfig())
	projectID := uuid.New().String()
	stepsService := NewMockStepsService(projectID)
	edgesService := NewMockEdgesService(projectID)
	workflowsService := NewMockWorkflowsService(projectID, "")

	// Pre-create two source steps
	stepsService.Create(map[string]interface{}{
		"project_id": projectID,
		"name":       "Source A",
		"type":       "http",
	})
	stepsService.Create(map[string]interface{}{
		"project_id": projectID,
		"name":       "Source B",
		"type":       "http",
	})

	execCtx := &sandbox.ExecutionContext{
		Steps:           stepsService,
		Edges:           edgesService,
		Workflows:       workflowsService,
		TargetProjectID: projectID,
	}

	code := extractAddStepCode()
	input := map[string]interface{}{
		"name": "Merge Step",
		"type": "function",
		"from": []interface{}{"Source A", "Source B"}, // Multiple sources
	}

	result, err := sb.Execute(context.Background(), code, input, execCtx)
	require.NoError(t, err)

	// Verify step was created
	assert.True(t, result["step_created"].(bool))

	// Verify two edges were created
	edges := result["edges"].([]interface{})
	assert.Len(t, edges, 2)
	assert.Len(t, edgesService.edges, 2)
}

// TestAddEdge_BasicEdgeCreation tests basic edge creation between existing steps
func TestAddEdge_BasicEdgeCreation(t *testing.T) {
	sb := sandbox.New(sandbox.DefaultConfig())
	projectID := uuid.New().String()
	stepsService := NewMockStepsService(projectID)
	edgesService := NewMockEdgesService(projectID)

	// Pre-create source and target steps
	stepsService.Create(map[string]interface{}{
		"project_id": projectID,
		"name":       "Source",
		"type":       "http",
	})
	stepsService.Create(map[string]interface{}{
		"project_id": projectID,
		"name":       "Target",
		"type":       "function",
	})

	execCtx := &sandbox.ExecutionContext{
		Steps:           stepsService,
		Edges:           edgesService,
		TargetProjectID: projectID,
	}

	code := extractAddEdgeCode()
	input := map[string]interface{}{
		"from": "Source",
		"to":   "Target",
	}

	result, err := sb.Execute(context.Background(), code, input, execCtx)
	require.NoError(t, err)

	// Verify edge was created
	assert.NotEmpty(t, result["edge_id"])
	assert.True(t, result["created"].(bool))
	assert.Equal(t, "Source", result["from"])
	assert.Equal(t, "Target", result["to"])

	// Verify edge exists in mock service
	assert.Len(t, edgesService.edges, 1)
}

// TestAddEdge_Idempotency tests that duplicate edge creation is handled gracefully
func TestAddEdge_Idempotency(t *testing.T) {
	sb := sandbox.New(sandbox.DefaultConfig())
	projectID := uuid.New().String()
	stepsService := NewMockStepsService(projectID)
	edgesService := NewMockEdgesService(projectID)

	// Pre-create steps
	stepsService.Create(map[string]interface{}{
		"project_id": projectID,
		"name":       "Source",
		"type":       "http",
	})
	stepsService.Create(map[string]interface{}{
		"project_id": projectID,
		"name":       "Target",
		"type":       "function",
	})

	execCtx := &sandbox.ExecutionContext{
		Steps:           stepsService,
		Edges:           edgesService,
		TargetProjectID: projectID,
	}

	code := extractAddEdgeCode()
	input := map[string]interface{}{
		"from": "Source",
		"to":   "Target",
	}

	// First call - should create edge
	result1, err := sb.Execute(context.Background(), code, input, execCtx)
	require.NoError(t, err)
	assert.True(t, result1["created"].(bool))
	edgeID := result1["edge_id"].(string)

	// Second call - should return existing edge
	result2, err := sb.Execute(context.Background(), code, input, execCtx)
	require.NoError(t, err)
	assert.False(t, result2["created"].(bool))
	assert.Equal(t, edgeID, result2["edge_id"])
	assert.Contains(t, result2["message"].(string), "already exists")

	// Verify only one edge exists
	assert.Len(t, edgesService.edges, 1)
}

// TestAddEdge_ConditionPort tests edge creation with from_port for condition blocks
func TestAddEdge_ConditionPort(t *testing.T) {
	sb := sandbox.New(sandbox.DefaultConfig())
	projectID := uuid.New().String()
	stepsService := NewMockStepsService(projectID)
	edgesService := NewMockEdgesService(projectID)

	// Pre-create condition source step
	stepsService.Create(map[string]interface{}{
		"project_id": projectID,
		"name":       "Check Condition",
		"type":       "condition",
	})
	stepsService.Create(map[string]interface{}{
		"project_id": projectID,
		"name":       "True Branch",
		"type":       "function",
	})

	execCtx := &sandbox.ExecutionContext{
		Steps:           stepsService,
		Edges:           edgesService,
		TargetProjectID: projectID,
	}

	code := extractAddEdgeCode()
	input := map[string]interface{}{
		"from":      "Check Condition",
		"to":        "True Branch",
		"from_port": "true", // Explicit port
	}

	result, err := sb.Execute(context.Background(), code, input, execCtx)
	require.NoError(t, err)

	// Verify edge was created with correct port
	assert.True(t, result["created"].(bool))
	assert.Equal(t, "true", result["from_port"])

	// Verify edge in service has correct port
	assert.Equal(t, "true", edgesService.edges[0]["source_port"])
}

// TestCompleteWorkflowGeneration tests building a complete 3-step workflow
func TestCompleteWorkflowGeneration(t *testing.T) {
	sb := sandbox.New(sandbox.DefaultConfig())
	projectID := uuid.New().String()
	stepsService := NewMockStepsService(projectID)
	edgesService := NewMockEdgesService(projectID)
	workflowsService := NewMockWorkflowsService(projectID, "")

	execCtx := &sandbox.ExecutionContext{
		Steps:           stepsService,
		Edges:           edgesService,
		Workflows:       workflowsService,
		TargetProjectID: projectID,
	}

	addStepCode := extractAddStepCode()

	// Step 1: Create trigger (no 'from')
	input1 := map[string]interface{}{
		"name":   "手動トリガー",
		"type":   "manual_trigger",
		"config": map[string]interface{}{},
	}
	result1, err := sb.Execute(context.Background(), addStepCode, input1, execCtx)
	require.NoError(t, err)
	assert.True(t, result1["step_created"].(bool))
	triggerStepID := result1["step_id"].(string)

	// Step 2: Create HTTP step (from trigger)
	input2 := map[string]interface{}{
		"name": "データ取得",
		"type": "http",
		"from": "手動トリガー",
		"config": map[string]interface{}{
			"url":    "https://api.example.com/data",
			"method": "GET",
		},
	}
	result2, err := sb.Execute(context.Background(), addStepCode, input2, execCtx)
	require.NoError(t, err)
	assert.True(t, result2["step_created"].(bool))
	edges2 := result2["edges"].([]interface{})
	require.Len(t, edges2, 1)
	assert.True(t, edges2[0].(map[string]interface{})["edge_created"].(bool))

	// Step 3: Create Discord notification (from HTTP step)
	input3 := map[string]interface{}{
		"name": "Discord通知",
		"type": "discord",
		"from": "データ取得",
		"config": map[string]interface{}{
			"channel_id": "123456789",
			"message":    "データを取得しました: {{$.url}}",
		},
	}
	result3, err := sb.Execute(context.Background(), addStepCode, input3, execCtx)
	require.NoError(t, err)
	assert.True(t, result3["step_created"].(bool))
	edges3 := result3["edges"].([]interface{})
	require.Len(t, edges3, 1)
	assert.True(t, edges3[0].(map[string]interface{})["edge_created"].(bool))

	// Verify complete workflow structure
	assert.Len(t, stepsService.steps, 3, "Should have 3 steps")
	assert.Len(t, edgesService.edges, 2, "Should have 2 edges")

	// Verify step names and types
	stepNames := map[string]string{}
	for _, step := range stepsService.steps {
		stepNames[step["name"].(string)] = step["type"].(string)
	}
	assert.Equal(t, "manual_trigger", stepNames["手動トリガー"])
	assert.Equal(t, "http", stepNames["データ取得"])
	assert.Equal(t, "discord", stepNames["Discord通知"])

	// Verify edges connect correctly
	triggerToHTTP := false
	httpToDiscord := false
	for _, edge := range edgesService.edges {
		sourceID := edge["source_step_id"].(string)
		targetID := edge["target_step_id"].(string)

		// Find source step
		var sourceStep, targetStep map[string]interface{}
		for _, s := range stepsService.steps {
			if s["id"] == sourceID {
				sourceStep = s
			}
			if s["id"] == targetID {
				targetStep = s
			}
		}

		if sourceStep != nil && targetStep != nil {
			if sourceStep["name"] == "手動トリガー" && targetStep["name"] == "データ取得" {
				triggerToHTTP = true
			}
			if sourceStep["name"] == "データ取得" && targetStep["name"] == "Discord通知" {
				httpToDiscord = true
			}
		}
	}

	assert.True(t, triggerToHTTP, "Edge from trigger to HTTP should exist")
	assert.True(t, httpToDiscord, "Edge from HTTP to Discord should exist")

	// Verify no orphan steps - all non-trigger steps should have incoming edges
	incomingEdges := map[string]int{}
	for _, edge := range edgesService.edges {
		targetID := edge["target_step_id"].(string)
		incomingEdges[targetID]++
	}

	for _, step := range stepsService.steps {
		stepID := step["id"].(string)
		if stepID != triggerStepID {
			assert.GreaterOrEqual(t, incomingEdges[stepID], 1,
				"Step %s should have at least one incoming edge", step["name"])
		}
	}
}

// TestAddStep_RequiredParameters tests error handling for missing required parameters
func TestAddStep_RequiredParameters(t *testing.T) {
	sb := sandbox.New(sandbox.DefaultConfig())
	projectID := uuid.New().String()
	stepsService := NewMockStepsService(projectID)
	edgesService := NewMockEdgesService(projectID)
	workflowsService := NewMockWorkflowsService(projectID, "")

	execCtx := &sandbox.ExecutionContext{
		Steps:           stepsService,
		Edges:           edgesService,
		Workflows:       workflowsService,
		TargetProjectID: projectID,
	}

	code := extractAddStepCode()

	t.Run("missing name", func(t *testing.T) {
		input := map[string]interface{}{
			"type": "http",
		}
		result, err := sb.Execute(context.Background(), code, input, execCtx)
		require.NoError(t, err)
		assert.Contains(t, result["error"].(string), "required")
	})

	t.Run("missing type", func(t *testing.T) {
		input := map[string]interface{}{
			"name": "Test Step",
		}
		result, err := sb.Execute(context.Background(), code, input, execCtx)
		require.NoError(t, err)
		assert.Contains(t, result["error"].(string), "required")
	})

	t.Run("missing project", func(t *testing.T) {
		execCtxNoProject := &sandbox.ExecutionContext{
			Steps:     stepsService,
			Edges:     edgesService,
			Workflows: workflowsService,
			// TargetProjectID not set
		}
		input := map[string]interface{}{
			"name": "Test Step",
			"type": "http",
		}
		result, err := sb.Execute(context.Background(), code, input, execCtxNoProject)
		require.NoError(t, err)
		assert.Contains(t, result["error"].(string), "No target project")
	})
}

// TestAddStep_ConditionBlockAutoPort tests automatic port resolution for condition blocks
func TestAddStep_ConditionBlockAutoPort(t *testing.T) {
	sb := sandbox.New(sandbox.DefaultConfig())
	projectID := uuid.New().String()
	stepsService := NewMockStepsService(projectID)
	edgesService := NewMockEdgesService(projectID)
	workflowsService := NewMockWorkflowsService(projectID, "")

	// Pre-create a condition step
	stepsService.Create(map[string]interface{}{
		"project_id": projectID,
		"name":       "Check Status",
		"type":       "condition",
	})

	execCtx := &sandbox.ExecutionContext{
		Steps:           stepsService,
		Edges:           edgesService,
		Workflows:       workflowsService,
		TargetProjectID: projectID,
	}

	code := extractAddStepCode()
	input := map[string]interface{}{
		"name": "On True",
		"type": "function",
		"from": "Check Status",
		// No from_port specified - should auto-resolve to 'true' for condition blocks
	}

	result, err := sb.Execute(context.Background(), code, input, execCtx)
	require.NoError(t, err)

	assert.True(t, result["step_created"].(bool))
	edges := result["edges"].([]interface{})
	require.Len(t, edges, 1)

	// Verify the edge uses 'true' port (auto-resolved for condition blocks)
	edge := edgesService.edges[0]
	assert.Equal(t, "true", edge["source_port"])
}

// TestAddStep_SwitchBlockAutoPort tests automatic port resolution for switch blocks
func TestAddStep_SwitchBlockAutoPort(t *testing.T) {
	sb := sandbox.New(sandbox.DefaultConfig())
	projectID := uuid.New().String()
	stepsService := NewMockStepsService(projectID)
	edgesService := NewMockEdgesService(projectID)
	workflowsService := NewMockWorkflowsService(projectID, "")

	// Pre-create a switch step
	stepsService.Create(map[string]interface{}{
		"project_id": projectID,
		"name":       "Route Request",
		"type":       "switch",
	})

	execCtx := &sandbox.ExecutionContext{
		Steps:           stepsService,
		Edges:           edgesService,
		Workflows:       workflowsService,
		TargetProjectID: projectID,
	}

	code := extractAddStepCode()
	input := map[string]interface{}{
		"name": "Default Handler",
		"type": "function",
		"from": "Route Request",
		// No from_port specified - should auto-resolve to 'default' for switch blocks
	}

	result, err := sb.Execute(context.Background(), code, input, execCtx)
	require.NoError(t, err)

	assert.True(t, result["step_created"].(bool))
	edges := result["edges"].([]interface{})
	require.Len(t, edges, 1)

	// Verify the edge uses 'default' port (auto-resolved for switch blocks)
	edge := edgesService.edges[0]
	assert.Equal(t, "default", edge["source_port"])
}

// TestAddStep_FindByUUID tests that steps can be found by UUID
func TestAddStep_FindByUUID(t *testing.T) {
	sb := sandbox.New(sandbox.DefaultConfig())
	projectID := uuid.New().String()
	stepsService := NewMockStepsService(projectID)
	edgesService := NewMockEdgesService(projectID)
	workflowsService := NewMockWorkflowsService(projectID, "")

	// Pre-create a source step and get its UUID
	sourceStep, _ := stepsService.Create(map[string]interface{}{
		"project_id": projectID,
		"name":       "Source Step",
		"type":       "http",
	})
	sourceID := sourceStep["id"].(string)

	execCtx := &sandbox.ExecutionContext{
		Steps:           stepsService,
		Edges:           edgesService,
		Workflows:       workflowsService,
		TargetProjectID: projectID,
	}

	code := extractAddStepCode()
	input := map[string]interface{}{
		"name": "Target Step",
		"type": "function",
		"from": sourceID, // Use UUID instead of name
	}

	result, err := sb.Execute(context.Background(), code, input, execCtx)
	require.NoError(t, err)

	assert.True(t, result["step_created"].(bool))
	edges := result["edges"].([]interface{})
	require.Len(t, edges, 1)

	// Verify the edge connects to the correct source
	edge := edgesService.edges[0]
	assert.Equal(t, sourceID, edge["source_step_id"])
}

// TestAddEdge_RequiredParameters tests error handling for add_edge
func TestAddEdge_RequiredParameters(t *testing.T) {
	sb := sandbox.New(sandbox.DefaultConfig())
	projectID := uuid.New().String()
	stepsService := NewMockStepsService(projectID)
	edgesService := NewMockEdgesService(projectID)

	execCtx := &sandbox.ExecutionContext{
		Steps:           stepsService,
		Edges:           edgesService,
		TargetProjectID: projectID,
	}

	code := extractAddEdgeCode()

	t.Run("missing from", func(t *testing.T) {
		input := map[string]interface{}{
			"to": "Target",
		}
		result, err := sb.Execute(context.Background(), code, input, execCtx)
		require.NoError(t, err)
		assert.Contains(t, result["error"].(string), "required")
	})

	t.Run("missing to", func(t *testing.T) {
		input := map[string]interface{}{
			"from": "Source",
		}
		result, err := sb.Execute(context.Background(), code, input, execCtx)
		require.NoError(t, err)
		assert.Contains(t, result["error"].(string), "required")
	})

	t.Run("source not found", func(t *testing.T) {
		stepsService.Create(map[string]interface{}{
			"project_id": projectID,
			"name":       "Target",
			"type":       "function",
		})
		input := map[string]interface{}{
			"from": "Non-existent",
			"to":   "Target",
		}
		result, err := sb.Execute(context.Background(), code, input, execCtx)
		require.NoError(t, err)
		assert.Contains(t, result["error"].(string), "Source step not found")
	})

	t.Run("target not found", func(t *testing.T) {
		stepsService.Create(map[string]interface{}{
			"project_id": projectID,
			"name":       "Source",
			"type":       "http",
		})
		input := map[string]interface{}{
			"from": "Source",
			"to":   "Non-existent",
		}
		result, err := sb.Execute(context.Background(), code, input, execCtx)
		require.NoError(t, err)
		assert.Contains(t, result["error"].(string), "Target step not found")
	})
}

// TestAddEdge_CannotConnectToTrigger tests that edges cannot be created TO trigger steps
func TestAddEdge_CannotConnectToTrigger(t *testing.T) {
	sb := sandbox.New(sandbox.DefaultConfig())
	projectID := uuid.New().String()
	stepsService := NewMockStepsService(projectID)
	edgesService := NewMockEdgesService(projectID)

	triggerTypes := []string{"manual_trigger", "schedule_trigger", "webhook_trigger", "start"}

	for _, triggerType := range triggerTypes {
		t.Run("cannot connect to "+triggerType, func(t *testing.T) {
			// Clear previous steps
			freshStepsService := NewMockStepsService(projectID)
			freshEdgesService := NewMockEdgesService(projectID)

			// Create a source step (non-trigger)
			freshStepsService.Create(map[string]interface{}{
				"project_id": projectID,
				"name":       "Source Step",
				"type":       "http",
			})

			// Create a trigger step as target
			freshStepsService.Create(map[string]interface{}{
				"project_id": projectID,
				"name":       "Trigger Step",
				"type":       triggerType,
			})

			execCtx := &sandbox.ExecutionContext{
				Steps:           freshStepsService,
				Edges:           freshEdgesService,
				TargetProjectID: projectID,
			}

			code := extractAddEdgeCode()
			input := map[string]interface{}{
				"from": "Source Step",
				"to":   "Trigger Step",
			}

			result, err := sb.Execute(context.Background(), code, input, execCtx)
			require.NoError(t, err)

			// Should return an error
			assert.NotNil(t, result["error"], "Should error when connecting to %s", triggerType)
			assert.Contains(t, result["error"].(string), "Cannot connect to a trigger step")

			// No edge should be created
			assert.Len(t, freshEdgesService.edges, 0, "No edge should be created when target is a trigger")
		})
	}

	// But connecting FROM a trigger TO a non-trigger should work
	t.Run("can connect from trigger to non-trigger", func(t *testing.T) {
		freshStepsService := NewMockStepsService(projectID)
		freshEdgesService := NewMockEdgesService(projectID)

		// Create a trigger step as source
		freshStepsService.Create(map[string]interface{}{
			"project_id": projectID,
			"name":       "Trigger",
			"type":       "manual_trigger",
		})

		// Create a non-trigger step as target
		freshStepsService.Create(map[string]interface{}{
			"project_id": projectID,
			"name":       "Process",
			"type":       "http",
		})

		execCtx := &sandbox.ExecutionContext{
			Steps:           freshStepsService,
			Edges:           freshEdgesService,
			TargetProjectID: projectID,
		}

		code := extractAddEdgeCode()
		input := map[string]interface{}{
			"from": "Trigger",
			"to":   "Process",
		}

		result, err := sb.Execute(context.Background(), code, input, execCtx)
		require.NoError(t, err)

		// Should succeed
		assert.Nil(t, result["error"])
		assert.True(t, result["created"].(bool))
		assert.Len(t, freshEdgesService.edges, 1)
	})

	_ = stepsService // silence unused
	_ = edgesService // silence unused
}

// TestAddStep_OrphanStepWarning tests that a warning is returned when creating non-trigger steps without 'from'
func TestAddStep_OrphanStepWarning(t *testing.T) {
	sb := sandbox.New(sandbox.DefaultConfig())
	projectID := uuid.New().String()
	stepsService := NewMockStepsService(projectID)
	edgesService := NewMockEdgesService(projectID)
	workflowsService := NewMockWorkflowsService(projectID, "")

	execCtx := &sandbox.ExecutionContext{
		Steps:           stepsService,
		Edges:           edgesService,
		Workflows:       workflowsService,
		TargetProjectID: projectID,
	}

	code := extractAddStepCode()

	// Test 1: Non-trigger step without 'from' should produce a warning
	t.Run("non-trigger without from produces warning", func(t *testing.T) {
		input := map[string]interface{}{
			"name":   "Orphan HTTP Step",
			"type":   "http",
			"config": map[string]interface{}{"url": "https://example.com"},
			// No 'from' parameter
		}

		result, err := sb.Execute(context.Background(), code, input, execCtx)
		require.NoError(t, err)

		// Step should still be created
		assert.True(t, result["step_created"].(bool))
		assert.NotEmpty(t, result["step_id"])

		// But should have a warning
		warning, ok := result["warning"].(string)
		assert.True(t, ok, "Should have a warning")
		assert.Contains(t, warning, "without connection")
	})

	// Test 2: Trigger step without 'from' should NOT produce a warning
	t.Run("trigger without from does not produce warning", func(t *testing.T) {
		freshStepsService := NewMockStepsService(projectID)
		freshEdgesService := NewMockEdgesService(projectID)

		execCtx := &sandbox.ExecutionContext{
			Steps:           freshStepsService,
			Edges:           freshEdgesService,
			Workflows:       workflowsService,
			TargetProjectID: projectID,
		}

		input := map[string]interface{}{
			"name":   "Manual Trigger",
			"type":   "manual_trigger",
			"config": map[string]interface{}{},
			// No 'from' parameter - this is expected for triggers
		}

		result, err := sb.Execute(context.Background(), code, input, execCtx)
		require.NoError(t, err)

		// Step should be created
		assert.True(t, result["step_created"].(bool))

		// Should NOT have a warning (triggers don't need 'from')
		_, hasWarning := result["warning"]
		assert.False(t, hasWarning, "Trigger steps should not warn about missing 'from'")
	})

	// Test 3: Non-trigger step WITH 'from' should NOT produce a warning
	t.Run("non-trigger with from does not produce warning", func(t *testing.T) {
		freshStepsService := NewMockStepsService(projectID)
		freshEdgesService := NewMockEdgesService(projectID)

		// Create a source step
		freshStepsService.Create(map[string]interface{}{
			"project_id": projectID,
			"name":       "Source",
			"type":       "manual_trigger",
		})

		execCtx := &sandbox.ExecutionContext{
			Steps:           freshStepsService,
			Edges:           freshEdgesService,
			Workflows:       workflowsService,
			TargetProjectID: projectID,
		}

		input := map[string]interface{}{
			"name":   "Connected HTTP Step",
			"type":   "http",
			"from":   "Source", // Has 'from' - properly connected
			"config": map[string]interface{}{"url": "https://example.com"},
		}

		result, err := sb.Execute(context.Background(), code, input, execCtx)
		require.NoError(t, err)

		// Step should be created
		assert.True(t, result["step_created"].(bool))

		// Should NOT have a warning (properly connected)
		_, hasWarning := result["warning"]
		assert.False(t, hasWarning, "Properly connected steps should not warn")
	})
}

// TestAddStep_CannotCreateTriggerWithFrom tests that trigger steps cannot be created with 'from' parameter
func TestAddStep_CannotCreateTriggerWithFrom(t *testing.T) {
	sb := sandbox.New(sandbox.DefaultConfig())
	projectID := uuid.New().String()
	workflowsService := NewMockWorkflowsService(projectID, "")

	triggerTypes := []string{"manual_trigger", "schedule_trigger", "webhook_trigger", "start"}

	for _, triggerType := range triggerTypes {
		t.Run("cannot create "+triggerType+" with from", func(t *testing.T) {
			freshStepsService := NewMockStepsService(projectID)
			freshEdgesService := NewMockEdgesService(projectID)

			// Create a source step
			freshStepsService.Create(map[string]interface{}{
				"project_id": projectID,
				"name":       "Source Step",
				"type":       "http",
			})

			execCtx := &sandbox.ExecutionContext{
				Steps:           freshStepsService,
				Edges:           freshEdgesService,
				Workflows:       workflowsService,
				TargetProjectID: projectID,
			}

			code := extractAddStepCode()
			input := map[string]interface{}{
				"name":   "New Trigger",
				"type":   triggerType,
				"from":   "Source Step", // This should be rejected
				"config": map[string]interface{}{},
			}

			result, err := sb.Execute(context.Background(), code, input, execCtx)
			require.NoError(t, err)

			// Should return an error
			assert.NotNil(t, result["error"], "Should error when creating %s with from parameter", triggerType)
			assert.Contains(t, result["error"].(string), "Cannot connect to a trigger step")

			// No step should be created
			assert.Len(t, freshStepsService.steps, 1, "Only the source step should exist")
		})
	}

	// But creating a trigger without 'from' should work
	t.Run("can create trigger without from", func(t *testing.T) {
		freshStepsService := NewMockStepsService(projectID)
		freshEdgesService := NewMockEdgesService(projectID)

		execCtx := &sandbox.ExecutionContext{
			Steps:           freshStepsService,
			Edges:           freshEdgesService,
			Workflows:       workflowsService,
			TargetProjectID: projectID,
		}

		code := extractAddStepCode()
		input := map[string]interface{}{
			"name":   "My Trigger",
			"type":   "manual_trigger",
			"config": map[string]interface{}{},
			// No 'from' parameter
		}

		result, err := sb.Execute(context.Background(), code, input, execCtx)
		require.NoError(t, err)

		// Should succeed
		assert.Nil(t, result["error"])
		assert.True(t, result["step_created"].(bool))
		assert.Len(t, freshStepsService.steps, 1)
	})
}
