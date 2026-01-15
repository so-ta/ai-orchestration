package usecase

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations

type mockWorkflowRepo struct {
	getByIDFunc       func(ctx context.Context, tenantID, id uuid.UUID) (*domain.Workflow, error)
	getSystemBySlugFn func(ctx context.Context, slug string) (*domain.Workflow, error)
}

func (m *mockWorkflowRepo) Create(ctx context.Context, workflow *domain.Workflow) error { return nil }
func (m *mockWorkflowRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Workflow, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, tenantID, id)
	}
	return nil, domain.ErrWorkflowNotFound
}
func (m *mockWorkflowRepo) List(ctx context.Context, tenantID uuid.UUID, filter repository.WorkflowFilter) ([]*domain.Workflow, int, error) {
	return nil, 0, nil
}
func (m *mockWorkflowRepo) Update(ctx context.Context, workflow *domain.Workflow) error { return nil }
func (m *mockWorkflowRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error   { return nil }
func (m *mockWorkflowRepo) GetWithStepsAndEdges(ctx context.Context, tenantID, id uuid.UUID) (*domain.Workflow, error) {
	return nil, nil
}
func (m *mockWorkflowRepo) GetSystemBySlug(ctx context.Context, slug string) (*domain.Workflow, error) {
	if m.getSystemBySlugFn != nil {
		return m.getSystemBySlugFn(ctx, slug)
	}
	return nil, domain.ErrWorkflowNotFound
}

type mockRunRepo struct {
	createFunc        func(ctx context.Context, run *domain.Run) error
	getByIDFunc       func(ctx context.Context, tenantID, id uuid.UUID) (*domain.Run, error)
	listByWorkflowFn  func(ctx context.Context, tenantID, workflowID uuid.UUID, filter repository.RunFilter) ([]*domain.Run, int, error)
	updateFunc        func(ctx context.Context, run *domain.Run) error
	getWithStepRunsFn func(ctx context.Context, tenantID, id uuid.UUID) (*domain.Run, error)
}

func (m *mockRunRepo) Create(ctx context.Context, run *domain.Run) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, run)
	}
	return nil
}
func (m *mockRunRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Run, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, tenantID, id)
	}
	return nil, domain.ErrRunNotFound
}
func (m *mockRunRepo) ListByWorkflow(ctx context.Context, tenantID, workflowID uuid.UUID, filter repository.RunFilter) ([]*domain.Run, int, error) {
	if m.listByWorkflowFn != nil {
		return m.listByWorkflowFn(ctx, tenantID, workflowID, filter)
	}
	return nil, 0, nil
}
func (m *mockRunRepo) Update(ctx context.Context, run *domain.Run) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, run)
	}
	return nil
}
func (m *mockRunRepo) GetWithStepRuns(ctx context.Context, tenantID, id uuid.UUID) (*domain.Run, error) {
	if m.getWithStepRunsFn != nil {
		return m.getWithStepRunsFn(ctx, tenantID, id)
	}
	return nil, domain.ErrRunNotFound
}

type mockVersionRepo struct {
	getByWorkflowAndVersionFn func(ctx context.Context, workflowID uuid.UUID, version int) (*domain.WorkflowVersion, error)
}

func (m *mockVersionRepo) Create(ctx context.Context, version *domain.WorkflowVersion) error {
	return nil
}
func (m *mockVersionRepo) GetByWorkflowAndVersion(ctx context.Context, workflowID uuid.UUID, version int) (*domain.WorkflowVersion, error) {
	if m.getByWorkflowAndVersionFn != nil {
		return m.getByWorkflowAndVersionFn(ctx, workflowID, version)
	}
	return nil, domain.ErrWorkflowVersionNotFound
}
func (m *mockVersionRepo) GetLatestByWorkflow(ctx context.Context, workflowID uuid.UUID) (*domain.WorkflowVersion, error) {
	return nil, nil
}
func (m *mockVersionRepo) ListByWorkflow(ctx context.Context, workflowID uuid.UUID) ([]*domain.WorkflowVersion, error) {
	return nil, nil
}

type mockStepRepo struct {
	listByWorkflowFn func(ctx context.Context, tenantID, workflowID uuid.UUID) ([]*domain.Step, error)
}

func (m *mockStepRepo) Create(ctx context.Context, step *domain.Step) error { return nil }
func (m *mockStepRepo) GetByID(ctx context.Context, tenantID, workflowID, id uuid.UUID) (*domain.Step, error) {
	return nil, nil
}
func (m *mockStepRepo) ListByWorkflow(ctx context.Context, tenantID, workflowID uuid.UUID) ([]*domain.Step, error) {
	if m.listByWorkflowFn != nil {
		return m.listByWorkflowFn(ctx, tenantID, workflowID)
	}
	return nil, nil
}
func (m *mockStepRepo) ListByBlockGroup(ctx context.Context, tenantID, blockGroupID uuid.UUID) ([]*domain.Step, error) {
	return nil, nil
}
func (m *mockStepRepo) Update(ctx context.Context, step *domain.Step) error { return nil }
func (m *mockStepRepo) Delete(ctx context.Context, tenantID, workflowID, id uuid.UUID) error {
	return nil
}

type mockEdgeRepo struct {
	listByWorkflowFn func(ctx context.Context, tenantID, workflowID uuid.UUID) ([]*domain.Edge, error)
}

func (m *mockEdgeRepo) Create(ctx context.Context, edge *domain.Edge) error { return nil }
func (m *mockEdgeRepo) GetByID(ctx context.Context, tenantID, workflowID, id uuid.UUID) (*domain.Edge, error) {
	return nil, nil
}
func (m *mockEdgeRepo) ListByWorkflow(ctx context.Context, tenantID, workflowID uuid.UUID) ([]*domain.Edge, error) {
	if m.listByWorkflowFn != nil {
		return m.listByWorkflowFn(ctx, tenantID, workflowID)
	}
	return nil, nil
}
func (m *mockEdgeRepo) Delete(ctx context.Context, tenantID, workflowID, id uuid.UUID) error {
	return nil
}
func (m *mockEdgeRepo) Exists(ctx context.Context, tenantID, workflowID, sourceID, targetID uuid.UUID) (bool, error) {
	return false, nil
}

type mockStepRunRepo struct {
	getLatestByStepFn     func(ctx context.Context, tenantID, runID, stepID uuid.UUID) (*domain.StepRun, error)
	getMaxAttemptForRunFn func(ctx context.Context, tenantID, runID uuid.UUID) (int, error)
	listCompletedByRunFn  func(ctx context.Context, tenantID, runID uuid.UUID) ([]*domain.StepRun, error)
	listByStepFn          func(ctx context.Context, tenantID, runID, stepID uuid.UUID) ([]*domain.StepRun, error)
}

func (m *mockStepRunRepo) Create(ctx context.Context, stepRun *domain.StepRun) error { return nil }
func (m *mockStepRunRepo) GetByID(ctx context.Context, tenantID, runID, id uuid.UUID) (*domain.StepRun, error) {
	return nil, nil
}
func (m *mockStepRunRepo) ListByRun(ctx context.Context, tenantID, runID uuid.UUID) ([]*domain.StepRun, error) {
	return nil, nil
}
func (m *mockStepRunRepo) Update(ctx context.Context, stepRun *domain.StepRun) error { return nil }
func (m *mockStepRunRepo) GetMaxAttempt(ctx context.Context, tenantID, runID, stepID uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockStepRunRepo) GetMaxAttemptForRun(ctx context.Context, tenantID, runID uuid.UUID) (int, error) {
	if m.getMaxAttemptForRunFn != nil {
		return m.getMaxAttemptForRunFn(ctx, tenantID, runID)
	}
	return 0, nil
}
func (m *mockStepRunRepo) GetLatestByStep(ctx context.Context, tenantID, runID, stepID uuid.UUID) (*domain.StepRun, error) {
	if m.getLatestByStepFn != nil {
		return m.getLatestByStepFn(ctx, tenantID, runID, stepID)
	}
	return nil, nil
}
func (m *mockStepRunRepo) ListCompletedByRun(ctx context.Context, tenantID, runID uuid.UUID) ([]*domain.StepRun, error) {
	if m.listCompletedByRunFn != nil {
		return m.listCompletedByRunFn(ctx, tenantID, runID)
	}
	return nil, nil
}
func (m *mockStepRunRepo) ListByStep(ctx context.Context, tenantID, runID, stepID uuid.UUID) ([]*domain.StepRun, error) {
	if m.listByStepFn != nil {
		return m.listByStepFn(ctx, tenantID, runID, stepID)
	}
	return nil, nil
}

// Test helper to create RunUsecase with mocks (bypassing Queue)
func newTestRunUsecase(
	workflowRepo *mockWorkflowRepo,
	runRepo *mockRunRepo,
	versionRepo *mockVersionRepo,
	stepRepo *mockStepRepo,
	edgeRepo *mockEdgeRepo,
	stepRunRepo *mockStepRunRepo,
) *RunUsecase {
	return &RunUsecase{
		workflowRepo: workflowRepo,
		runRepo:      runRepo,
		versionRepo:  versionRepo,
		stepRepo:     stepRepo,
		edgeRepo:     edgeRepo,
		stepRunRepo:  stepRunRepo,
		queue:        nil, // Queue tests need Redis, skip for unit tests
	}
}

// Tests

func TestRunUsecase_GetByID(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	runID := uuid.New()
	workflowID := uuid.New()

	expectedRun := &domain.Run{
		ID:         runID,
		TenantID:   tenantID,
		WorkflowID: workflowID,
		Status:     domain.RunStatusCompleted,
	}

	runRepo := &mockRunRepo{
		getByIDFunc: func(ctx context.Context, tid, id uuid.UUID) (*domain.Run, error) {
			if tid == tenantID && id == runID {
				return expectedRun, nil
			}
			return nil, domain.ErrRunNotFound
		},
	}

	uc := newTestRunUsecase(nil, runRepo, nil, nil, nil, nil)

	t.Run("success", func(t *testing.T) {
		run, err := uc.GetByID(ctx, tenantID, runID)
		require.NoError(t, err)
		assert.Equal(t, expectedRun, run)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := uc.GetByID(ctx, tenantID, uuid.New())
		assert.ErrorIs(t, err, domain.ErrRunNotFound)
	})
}

func TestRunUsecase_GetWithDetails(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	runID := uuid.New()

	stepRuns := []domain.StepRun{
		{ID: uuid.New(), RunID: runID, Status: domain.StepRunStatusCompleted},
	}
	expectedRun := &domain.Run{
		ID:       runID,
		TenantID: tenantID,
		Status:   domain.RunStatusCompleted,
		StepRuns: stepRuns,
	}

	runRepo := &mockRunRepo{
		getWithStepRunsFn: func(ctx context.Context, tid, id uuid.UUID) (*domain.Run, error) {
			if tid == tenantID && id == runID {
				return expectedRun, nil
			}
			return nil, domain.ErrRunNotFound
		},
	}

	uc := newTestRunUsecase(nil, runRepo, nil, nil, nil, nil)

	run, err := uc.GetWithDetails(ctx, tenantID, runID)
	require.NoError(t, err)
	assert.Equal(t, expectedRun, run)
	assert.Len(t, run.StepRuns, 1)
}

func TestRunUsecase_List(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	workflowID := uuid.New()

	runs := []*domain.Run{
		{ID: uuid.New(), TenantID: tenantID, WorkflowID: workflowID, Status: domain.RunStatusCompleted},
		{ID: uuid.New(), TenantID: tenantID, WorkflowID: workflowID, Status: domain.RunStatusFailed},
	}

	runRepo := &mockRunRepo{
		listByWorkflowFn: func(ctx context.Context, tid, wid uuid.UUID, filter repository.RunFilter) ([]*domain.Run, int, error) {
			return runs, len(runs), nil
		},
	}

	uc := newTestRunUsecase(nil, runRepo, nil, nil, nil, nil)

	t.Run("default pagination", func(t *testing.T) {
		input := ListRunsInput{
			TenantID:   tenantID,
			WorkflowID: workflowID,
		}
		output, err := uc.List(ctx, input)
		require.NoError(t, err)
		assert.Len(t, output.Runs, 2)
		assert.Equal(t, 1, output.Page)
		assert.Equal(t, 20, output.Limit)
	})

	t.Run("custom pagination", func(t *testing.T) {
		input := ListRunsInput{
			TenantID:   tenantID,
			WorkflowID: workflowID,
			Page:       2,
			Limit:      10,
		}
		output, err := uc.List(ctx, input)
		require.NoError(t, err)
		assert.Equal(t, 2, output.Page)
		assert.Equal(t, 10, output.Limit)
	})

	t.Run("limit capped at 100", func(t *testing.T) {
		input := ListRunsInput{
			TenantID:   tenantID,
			WorkflowID: workflowID,
			Limit:      500,
		}
		output, err := uc.List(ctx, input)
		require.NoError(t, err)
		assert.Equal(t, 20, output.Limit) // Should be capped to default 20
	})
}

func TestRunUsecase_Cancel(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	runID := uuid.New()

	tests := []struct {
		name        string
		status      domain.RunStatus
		expectError bool
		errorType   error
	}{
		{
			name:        "cancel pending run",
			status:      domain.RunStatusPending,
			expectError: false,
		},
		{
			name:        "cancel running run",
			status:      domain.RunStatusRunning,
			expectError: false,
		},
		{
			name:        "cannot cancel completed run",
			status:      domain.RunStatusCompleted,
			expectError: true,
			errorType:   domain.ErrRunNotCancellable,
		},
		{
			name:        "cannot cancel failed run",
			status:      domain.RunStatusFailed,
			expectError: true,
			errorType:   domain.ErrRunNotCancellable,
		},
		{
			name:        "cannot cancel already cancelled run",
			status:      domain.RunStatusCancelled,
			expectError: true,
			errorType:   domain.ErrRunNotCancellable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := &domain.Run{
				ID:       runID,
				TenantID: tenantID,
				Status:   tt.status,
			}

			runRepo := &mockRunRepo{
				getByIDFunc: func(ctx context.Context, tid, id uuid.UUID) (*domain.Run, error) {
					return run, nil
				},
				updateFunc: func(ctx context.Context, r *domain.Run) error {
					return nil
				},
			}

			uc := newTestRunUsecase(nil, runRepo, nil, nil, nil, nil)

			result, err := uc.Cancel(ctx, tenantID, runID)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, domain.RunStatusCancelled, result.Status)
			}
		})
	}
}

func TestRunUsecase_GetStepHistory(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	runID := uuid.New()
	stepID := uuid.New()

	stepRuns := []*domain.StepRun{
		{ID: uuid.New(), RunID: runID, StepID: stepID, Attempt: 1, Status: domain.StepRunStatusFailed},
		{ID: uuid.New(), RunID: runID, StepID: stepID, Attempt: 2, Status: domain.StepRunStatusCompleted},
	}

	runRepo := &mockRunRepo{
		getByIDFunc: func(ctx context.Context, tid, id uuid.UUID) (*domain.Run, error) {
			if tid == tenantID && id == runID {
				return &domain.Run{ID: runID, TenantID: tenantID}, nil
			}
			return nil, domain.ErrRunNotFound
		},
	}

	stepRunRepo := &mockStepRunRepo{
		listByStepFn: func(ctx context.Context, tid, rid, sid uuid.UUID) ([]*domain.StepRun, error) {
			if rid == runID && sid == stepID {
				return stepRuns, nil
			}
			return nil, nil
		},
	}

	uc := newTestRunUsecase(nil, runRepo, nil, nil, nil, stepRunRepo)

	t.Run("success", func(t *testing.T) {
		history, err := uc.GetStepHistory(ctx, tenantID, runID, stepID)
		require.NoError(t, err)
		assert.Len(t, history, 2)
		assert.Equal(t, 1, history[0].Attempt)
		assert.Equal(t, 2, history[1].Attempt)
	})

	t.Run("run not found", func(t *testing.T) {
		_, err := uc.GetStepHistory(ctx, tenantID, uuid.New(), stepID)
		assert.ErrorIs(t, err, domain.ErrRunNotFound)
	})
}

func TestRunUsecase_GetWithDetailsAndDefinition(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	runID := uuid.New()
	workflowID := uuid.New()

	stepID := uuid.New()
	definition := domain.WorkflowDefinition{
		Name:        "Test Workflow",
		Description: "Test Description",
		Steps: []domain.Step{
			{ID: stepID, Name: "Step 1", Type: domain.StepTypeTool},
		},
		Edges: []domain.Edge{},
	}
	definitionJSON, _ := json.Marshal(definition)

	run := &domain.Run{
		ID:              runID,
		TenantID:        tenantID,
		WorkflowID:      workflowID,
		WorkflowVersion: 1,
		Status:          domain.RunStatusCompleted,
	}

	version := &domain.WorkflowVersion{
		WorkflowID: workflowID,
		Version:    1,
		Definition: definitionJSON,
	}

	runRepo := &mockRunRepo{
		getWithStepRunsFn: func(ctx context.Context, tid, id uuid.UUID) (*domain.Run, error) {
			if tid == tenantID && id == runID {
				return run, nil
			}
			return nil, domain.ErrRunNotFound
		},
	}

	versionRepo := &mockVersionRepo{
		getByWorkflowAndVersionFn: func(ctx context.Context, wid uuid.UUID, v int) (*domain.WorkflowVersion, error) {
			if wid == workflowID && v == 1 {
				return version, nil
			}
			return nil, domain.ErrWorkflowVersionNotFound
		},
	}

	uc := newTestRunUsecase(nil, runRepo, versionRepo, nil, nil, nil)

	t.Run("with version snapshot", func(t *testing.T) {
		output, err := uc.GetWithDetailsAndDefinition(ctx, tenantID, runID)
		require.NoError(t, err)
		assert.NotNil(t, output.Run)
		assert.NotNil(t, output.WorkflowDefinition)
		assert.Equal(t, "Test Workflow", output.WorkflowDefinition.Name)
		assert.Len(t, output.WorkflowDefinition.Steps, 1)
	})
}

func TestRunUsecase_GetWithDetailsAndDefinition_FallbackToWorkflow(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	runID := uuid.New()
	workflowID := uuid.New()

	run := &domain.Run{
		ID:              runID,
		TenantID:        tenantID,
		WorkflowID:      workflowID,
		WorkflowVersion: 1,
		Status:          domain.RunStatusCompleted,
	}

	workflow := &domain.Workflow{
		ID:          workflowID,
		TenantID:    tenantID,
		Name:        "Fallback Workflow",
		Description: "Fallback Description",
	}

	steps := []*domain.Step{
		{ID: uuid.New(), Name: "Step 1", Type: domain.StepTypeTool},
	}

	runRepo := &mockRunRepo{
		getWithStepRunsFn: func(ctx context.Context, tid, id uuid.UUID) (*domain.Run, error) {
			return run, nil
		},
	}

	// Version repo returns error - triggers fallback
	versionRepo := &mockVersionRepo{
		getByWorkflowAndVersionFn: func(ctx context.Context, wid uuid.UUID, v int) (*domain.WorkflowVersion, error) {
			return nil, domain.ErrWorkflowVersionNotFound
		},
	}

	workflowRepo := &mockWorkflowRepo{
		getByIDFunc: func(ctx context.Context, tid, id uuid.UUID) (*domain.Workflow, error) {
			return workflow, nil
		},
	}

	stepRepo := &mockStepRepo{
		listByWorkflowFn: func(ctx context.Context, tid, wid uuid.UUID) ([]*domain.Step, error) {
			return steps, nil
		},
	}

	edgeRepo := &mockEdgeRepo{
		listByWorkflowFn: func(ctx context.Context, tid, wid uuid.UUID) ([]*domain.Edge, error) {
			return []*domain.Edge{}, nil
		},
	}

	uc := newTestRunUsecase(workflowRepo, runRepo, versionRepo, stepRepo, edgeRepo, nil)

	output, err := uc.GetWithDetailsAndDefinition(ctx, tenantID, runID)
	require.NoError(t, err)
	assert.NotNil(t, output.WorkflowDefinition)
	assert.Equal(t, "Fallback Workflow", output.WorkflowDefinition.Name)
}

func TestCollectDownstreamSteps(t *testing.T) {
	stepA := uuid.New()
	stepB := uuid.New()
	stepC := uuid.New()
	stepD := uuid.New()

	tests := []struct {
		name           string
		definition     *domain.WorkflowDefinition
		startStepID    uuid.UUID
		expectedLength int
	}{
		{
			name: "single step - no downstream",
			definition: &domain.WorkflowDefinition{
				Steps: []domain.Step{{ID: stepA}},
				Edges: []domain.Edge{},
			},
			startStepID:    stepA,
			expectedLength: 1,
		},
		{
			name: "linear path",
			definition: &domain.WorkflowDefinition{
				Steps: []domain.Step{{ID: stepA}, {ID: stepB}, {ID: stepC}},
				Edges: []domain.Edge{
					{SourceStepID: &stepA, TargetStepID: &stepB},
					{SourceStepID: &stepB, TargetStepID: &stepC},
				},
			},
			startStepID:    stepA,
			expectedLength: 3,
		},
		{
			name: "start from middle",
			definition: &domain.WorkflowDefinition{
				Steps: []domain.Step{{ID: stepA}, {ID: stepB}, {ID: stepC}},
				Edges: []domain.Edge{
					{SourceStepID: &stepA, TargetStepID: &stepB},
					{SourceStepID: &stepB, TargetStepID: &stepC},
				},
			},
			startStepID:    stepB,
			expectedLength: 2, // stepB and stepC
		},
		{
			name: "diamond shape from top",
			definition: &domain.WorkflowDefinition{
				Steps: []domain.Step{{ID: stepA}, {ID: stepB}, {ID: stepC}, {ID: stepD}},
				Edges: []domain.Edge{
					{SourceStepID: &stepA, TargetStepID: &stepB},
					{SourceStepID: &stepA, TargetStepID: &stepC},
					{SourceStepID: &stepB, TargetStepID: &stepD},
					{SourceStepID: &stepC, TargetStepID: &stepD},
				},
			},
			startStepID:    stepA,
			expectedLength: 4,
		},
		{
			name: "diamond shape from branch",
			definition: &domain.WorkflowDefinition{
				Steps: []domain.Step{{ID: stepA}, {ID: stepB}, {ID: stepC}, {ID: stepD}},
				Edges: []domain.Edge{
					{SourceStepID: &stepA, TargetStepID: &stepB},
					{SourceStepID: &stepA, TargetStepID: &stepC},
					{SourceStepID: &stepB, TargetStepID: &stepD},
					{SourceStepID: &stepC, TargetStepID: &stepD},
				},
			},
			startStepID:    stepB,
			expectedLength: 2, // stepB and stepD
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collectDownstreamSteps(tt.definition, tt.startStepID)
			assert.Len(t, result, tt.expectedLength)
			// First element should always be the start step
			assert.Equal(t, tt.startStepID, result[0])
		})
	}
}
