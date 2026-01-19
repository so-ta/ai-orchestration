package sandbox

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockBuilderSessionsService is a mock implementation of BuilderSessionsService
type MockBuilderSessionsService struct {
	GetFunc        func(sessionID string) (map[string]interface{}, error)
	UpdateFunc     func(sessionID string, updates map[string]interface{}) error
	AddMessageFunc func(sessionID string, message map[string]interface{}) error
}

func (m *MockBuilderSessionsService) Get(sessionID string) (map[string]interface{}, error) {
	if m.GetFunc != nil {
		return m.GetFunc(sessionID)
	}
	return nil, nil
}

func (m *MockBuilderSessionsService) Update(sessionID string, updates map[string]interface{}) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(sessionID, updates)
	}
	return nil
}

func (m *MockBuilderSessionsService) AddMessage(sessionID string, message map[string]interface{}) error {
	if m.AddMessageFunc != nil {
		return m.AddMessageFunc(sessionID, message)
	}
	return nil
}

// MockProjectsService is a mock implementation of ProjectsService
type MockProjectsService struct {
	GetFunc              func(projectID string) (map[string]interface{}, error)
	CreateFunc           func(data map[string]interface{}) (map[string]interface{}, error)
	UpdateFunc           func(projectID string, updates map[string]interface{}) error
	IncrementVersionFunc func(projectID string) error
}

func (m *MockProjectsService) Get(projectID string) (map[string]interface{}, error) {
	if m.GetFunc != nil {
		return m.GetFunc(projectID)
	}
	return nil, nil
}

func (m *MockProjectsService) Create(data map[string]interface{}) (map[string]interface{}, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(data)
	}
	return nil, nil
}

func (m *MockProjectsService) Update(projectID string, updates map[string]interface{}) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(projectID, updates)
	}
	return nil
}

func (m *MockProjectsService) IncrementVersion(projectID string) error {
	if m.IncrementVersionFunc != nil {
		return m.IncrementVersionFunc(projectID)
	}
	return nil
}

// MockStepsService is a mock implementation of StepsService
type MockStepsService struct {
	CreateFunc        func(data map[string]interface{}) (map[string]interface{}, error)
	UpdateFunc        func(stepID string, updates map[string]interface{}) error
	DeleteFunc        func(stepID string) error
	ListByProjectFunc func(projectID string) ([]map[string]interface{}, error)
}

func (m *MockStepsService) ListByProject(projectID string) ([]map[string]interface{}, error) {
	if m.ListByProjectFunc != nil {
		return m.ListByProjectFunc(projectID)
	}
	return nil, nil
}

func (m *MockStepsService) Create(data map[string]interface{}) (map[string]interface{}, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(data)
	}
	return nil, nil
}

func (m *MockStepsService) Update(stepID string, updates map[string]interface{}) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(stepID, updates)
	}
	return nil
}

func (m *MockStepsService) Delete(stepID string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(stepID)
	}
	return nil
}

// MockEdgesService is a mock implementation of EdgesService
type MockEdgesService struct {
	CreateFunc        func(data map[string]interface{}) (map[string]interface{}, error)
	DeleteFunc        func(edgeID string) error
	ListByProjectFunc func(projectID string) ([]map[string]interface{}, error)
}

func (m *MockEdgesService) ListByProject(projectID string) ([]map[string]interface{}, error) {
	if m.ListByProjectFunc != nil {
		return m.ListByProjectFunc(projectID)
	}
	return nil, nil
}

func (m *MockEdgesService) Create(data map[string]interface{}) (map[string]interface{}, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(data)
	}
	return nil, nil
}

func (m *MockEdgesService) Delete(edgeID string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(edgeID)
	}
	return nil
}

func TestSandbox_BuilderSessionsService_Get(t *testing.T) {
	sb := New(DefaultConfig())

	mockService := &MockBuilderSessionsService{
		GetFunc: func(sessionID string) (map[string]interface{}, error) {
			return map[string]interface{}{
				"id":              sessionID,
				"status":          "hearing",
				"hearing_phase":   "analysis",
				"hearing_progress": 10,
			}, nil
		},
	}

	execCtx := &ExecutionContext{
		BuilderSessions: mockService,
	}

	code := `
		const session = ctx.builderSessions.get("test-session-id");
		return { id: session.id, status: session.status };
	`

	result, err := sb.Execute(context.Background(), code, map[string]interface{}{}, execCtx)
	require.NoError(t, err)
	assert.Equal(t, "test-session-id", result["id"])
	assert.Equal(t, "hearing", result["status"])
}

func TestSandbox_BuilderSessionsService_Update(t *testing.T) {
	sb := New(DefaultConfig())

	var updatedSession string
	var updatedData map[string]interface{}

	mockService := &MockBuilderSessionsService{
		UpdateFunc: func(sessionID string, updates map[string]interface{}) error {
			updatedSession = sessionID
			updatedData = updates
			return nil
		},
	}

	execCtx := &ExecutionContext{
		BuilderSessions: mockService,
	}

	code := `
		const result = ctx.builderSessions.update("test-session-id", {
			status: "completed",
			hearing_phase: "completed"
		});
		return result;
	`

	result, err := sb.Execute(context.Background(), code, map[string]interface{}{}, execCtx)
	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
	assert.Equal(t, "test-session-id", updatedSession)
	assert.Equal(t, "completed", updatedData["status"])
	assert.Equal(t, "completed", updatedData["hearing_phase"])
}

func TestSandbox_BuilderSessionsService_AddMessage(t *testing.T) {
	sb := New(DefaultConfig())

	var addedSession string
	var addedMessage map[string]interface{}

	mockService := &MockBuilderSessionsService{
		AddMessageFunc: func(sessionID string, message map[string]interface{}) error {
			addedSession = sessionID
			addedMessage = message
			return nil
		},
	}

	execCtx := &ExecutionContext{
		BuilderSessions: mockService,
	}

	code := `
		const result = ctx.builderSessions.addMessage("test-session-id", {
			role: "assistant",
			content: "Hello, how can I help you?"
		});
		return result;
	`

	result, err := sb.Execute(context.Background(), code, map[string]interface{}{}, execCtx)
	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
	assert.Equal(t, "test-session-id", addedSession)
	assert.Equal(t, "assistant", addedMessage["role"])
	assert.Equal(t, "Hello, how can I help you?", addedMessage["content"])
}

func TestSandbox_ProjectsService_Get(t *testing.T) {
	sb := New(DefaultConfig())

	mockService := &MockProjectsService{
		GetFunc: func(projectID string) (map[string]interface{}, error) {
			return map[string]interface{}{
				"id":     projectID,
				"name":   "Test Project",
				"status": "draft",
			}, nil
		},
	}

	execCtx := &ExecutionContext{
		Projects: mockService,
	}

	code := `
		const project = ctx.projects.get("test-project-id");
		return { id: project.id, name: project.name };
	`

	result, err := sb.Execute(context.Background(), code, map[string]interface{}{}, execCtx)
	require.NoError(t, err)
	assert.Equal(t, "test-project-id", result["id"])
	assert.Equal(t, "Test Project", result["name"])
}

func TestSandbox_ProjectsService_Create(t *testing.T) {
	sb := New(DefaultConfig())

	mockService := &MockProjectsService{
		CreateFunc: func(data map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"id":     "new-project-id",
				"name":   data["name"],
				"status": "draft",
			}, nil
		},
	}

	execCtx := &ExecutionContext{
		Projects: mockService,
	}

	code := `
		const project = ctx.projects.create({
			name: "New Project",
			description: "A new project"
		});
		return { id: project.id, name: project.name };
	`

	result, err := sb.Execute(context.Background(), code, map[string]interface{}{}, execCtx)
	require.NoError(t, err)
	assert.Equal(t, "new-project-id", result["id"])
	assert.Equal(t, "New Project", result["name"])
}

func TestSandbox_ProjectsService_IncrementVersion(t *testing.T) {
	sb := New(DefaultConfig())

	var incrementedProjectID string

	mockService := &MockProjectsService{
		IncrementVersionFunc: func(projectID string) error {
			incrementedProjectID = projectID
			return nil
		},
	}

	execCtx := &ExecutionContext{
		Projects: mockService,
	}

	code := `
		const result = ctx.projects.incrementVersion("test-project-id");
		return result;
	`

	result, err := sb.Execute(context.Background(), code, map[string]interface{}{}, execCtx)
	require.NoError(t, err)
	assert.True(t, result["success"].(bool))
	assert.Equal(t, "test-project-id", incrementedProjectID)
}

func TestSandbox_StepsService_Create(t *testing.T) {
	sb := New(DefaultConfig())

	mockService := &MockStepsService{
		CreateFunc: func(data map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"id":         "new-step-id",
				"project_id": data["project_id"],
				"name":       data["name"],
				"type":       data["type"],
			}, nil
		},
	}

	execCtx := &ExecutionContext{
		Steps: mockService,
	}

	code := `
		const step = ctx.steps.create({
			project_id: "test-project-id",
			name: "Start Step",
			type: "start"
		});
		return { id: step.id, name: step.name };
	`

	result, err := sb.Execute(context.Background(), code, map[string]interface{}{}, execCtx)
	require.NoError(t, err)
	assert.Equal(t, "new-step-id", result["id"])
	assert.Equal(t, "Start Step", result["name"])
}

func TestSandbox_StepsService_ListByProject(t *testing.T) {
	sb := New(DefaultConfig())

	mockService := &MockStepsService{
		ListByProjectFunc: func(projectID string) ([]map[string]interface{}, error) {
			return []map[string]interface{}{
				{"id": "step-1", "name": "Start", "type": "start"},
				{"id": "step-2", "name": "Process", "type": "function"},
			}, nil
		},
	}

	execCtx := &ExecutionContext{
		Steps: mockService,
	}

	code := `
		const steps = ctx.steps.listByProject("test-project-id");
		return { count: steps.length };
	`

	result, err := sb.Execute(context.Background(), code, map[string]interface{}{}, execCtx)
	require.NoError(t, err)
	assert.EqualValues(t, 2, result["count"])
}

func TestSandbox_EdgesService_Create(t *testing.T) {
	sb := New(DefaultConfig())

	mockService := &MockEdgesService{
		CreateFunc: func(data map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"id":              "new-edge-id",
				"project_id":     data["project_id"],
				"source_step_id": data["source_step_id"],
				"target_step_id": data["target_step_id"],
			}, nil
		},
	}

	execCtx := &ExecutionContext{
		Edges: mockService,
	}

	code := `
		const edge = ctx.edges.create({
			project_id: "test-project-id",
			source_step_id: "step-1",
			target_step_id: "step-2"
		});
		return { id: edge.id };
	`

	result, err := sb.Execute(context.Background(), code, map[string]interface{}{}, execCtx)
	require.NoError(t, err)
	assert.Equal(t, "new-edge-id", result["id"])
}

func TestSandbox_EdgesService_ListByProject(t *testing.T) {
	sb := New(DefaultConfig())

	mockService := &MockEdgesService{
		ListByProjectFunc: func(projectID string) ([]map[string]interface{}, error) {
			return []map[string]interface{}{
				{"id": "edge-1", "source_step_id": "step-1", "target_step_id": "step-2"},
			}, nil
		},
	}

	execCtx := &ExecutionContext{
		Edges: mockService,
	}

	code := `
		const edges = ctx.edges.listByProject("test-project-id");
		return { count: edges.length };
	`

	result, err := sb.Execute(context.Background(), code, map[string]interface{}{}, execCtx)
	require.NoError(t, err)
	assert.EqualValues(t, 1, result["count"])
}
