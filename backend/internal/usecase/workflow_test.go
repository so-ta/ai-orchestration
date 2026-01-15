package usecase

import (
	"testing"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestHasCycle(t *testing.T) {
	stepA := domain.Step{ID: uuid.New()}
	stepB := domain.Step{ID: uuid.New()}
	stepC := domain.Step{ID: uuid.New()}
	stepD := domain.Step{ID: uuid.New()}

	tests := []struct {
		name     string
		steps    []domain.Step
		edges    []domain.Edge
		expected bool
	}{
		{
			name:     "empty graph - no cycle",
			steps:    []domain.Step{},
			edges:    []domain.Edge{},
			expected: false,
		},
		{
			name:     "single node - no cycle",
			steps:    []domain.Step{stepA},
			edges:    []domain.Edge{},
			expected: false,
		},
		{
			name:  "linear path - no cycle",
			steps: []domain.Step{stepA, stepB, stepC},
			edges: []domain.Edge{
				{SourceStepID: &stepA.ID, TargetStepID: &stepB.ID},
				{SourceStepID: &stepB.ID, TargetStepID: &stepC.ID},
			},
			expected: false,
		},
		{
			name:  "diamond shape - no cycle",
			steps: []domain.Step{stepA, stepB, stepC, stepD},
			edges: []domain.Edge{
				{SourceStepID: &stepA.ID, TargetStepID: &stepB.ID},
				{SourceStepID: &stepA.ID, TargetStepID: &stepC.ID},
				{SourceStepID: &stepB.ID, TargetStepID: &stepD.ID},
				{SourceStepID: &stepC.ID, TargetStepID: &stepD.ID},
			},
			expected: false,
		},
		{
			name:  "simple cycle (A -> B -> A)",
			steps: []domain.Step{stepA, stepB},
			edges: []domain.Edge{
				{SourceStepID: &stepA.ID, TargetStepID: &stepB.ID},
				{SourceStepID: &stepB.ID, TargetStepID: &stepA.ID},
			},
			expected: true,
		},
		{
			name:  "longer cycle (A -> B -> C -> A)",
			steps: []domain.Step{stepA, stepB, stepC},
			edges: []domain.Edge{
				{SourceStepID: &stepA.ID, TargetStepID: &stepB.ID},
				{SourceStepID: &stepB.ID, TargetStepID: &stepC.ID},
				{SourceStepID: &stepC.ID, TargetStepID: &stepA.ID},
			},
			expected: true,
		},
		{
			name:  "self loop",
			steps: []domain.Step{stepA},
			edges: []domain.Edge{
				{SourceStepID: &stepA.ID, TargetStepID: &stepA.ID},
			},
			expected: true,
		},
		{
			name:  "multiple components with cycle",
			steps: []domain.Step{stepA, stepB, stepC, stepD},
			edges: []domain.Edge{
				{SourceStepID: &stepA.ID, TargetStepID: &stepB.ID},
				{SourceStepID: &stepC.ID, TargetStepID: &stepD.ID},
				{SourceStepID: &stepD.ID, TargetStepID: &stepC.ID}, // Cycle in second component
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasCycle(tt.steps, tt.edges)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasUnconnectedSteps(t *testing.T) {
	stepA := domain.Step{ID: uuid.New()}
	stepB := domain.Step{ID: uuid.New()}
	stepC := domain.Step{ID: uuid.New()}

	tests := []struct {
		name     string
		steps    []domain.Step
		edges    []domain.Edge
		expected bool
	}{
		{
			name:     "empty graph",
			steps:    []domain.Step{},
			edges:    []domain.Edge{},
			expected: false,
		},
		{
			name:     "single step - allowed to be unconnected",
			steps:    []domain.Step{stepA},
			edges:    []domain.Edge{},
			expected: false,
		},
		{
			name:  "all steps connected",
			steps: []domain.Step{stepA, stepB, stepC},
			edges: []domain.Edge{
				{SourceStepID: &stepA.ID, TargetStepID: &stepB.ID},
				{SourceStepID: &stepB.ID, TargetStepID: &stepC.ID},
			},
			expected: false,
		},
		{
			name:  "one unconnected step",
			steps: []domain.Step{stepA, stepB, stepC},
			edges: []domain.Edge{
				{SourceStepID: &stepA.ID, TargetStepID: &stepB.ID},
				// stepC is not connected
			},
			expected: true,
		},
		{
			name:  "multiple unconnected steps",
			steps: []domain.Step{stepA, stepB, stepC},
			edges: []domain.Edge{
				// Only A and B connected
				{SourceStepID: &stepA.ID, TargetStepID: &stepB.ID},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasUnconnectedSteps(tt.steps, tt.edges)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWorkflowUsecase_ValidateDAG(t *testing.T) {
	usecase := NewWorkflowUsecase(nil, nil, nil, nil, nil)

	stepA := domain.Step{ID: uuid.New(), Name: "A"}
	stepB := domain.Step{ID: uuid.New(), Name: "B"}
	stepC := domain.Step{ID: uuid.New(), Name: "C"}

	tests := []struct {
		name        string
		workflow    *domain.Workflow
		expectError bool
		errorType   error
	}{
		{
			name: "valid single-step workflow",
			workflow: &domain.Workflow{
				Steps: []domain.Step{stepA},
				Edges: []domain.Edge{},
			},
			expectError: false,
		},
		{
			name: "valid multi-step linear workflow",
			workflow: &domain.Workflow{
				Steps: []domain.Step{stepA, stepB, stepC},
				Edges: []domain.Edge{
					{SourceStepID: &stepA.ID, TargetStepID: &stepB.ID},
					{SourceStepID: &stepB.ID, TargetStepID: &stepC.ID},
				},
			},
			expectError: false,
		},
		{
			name: "empty workflow",
			workflow: &domain.Workflow{
				Steps: []domain.Step{},
				Edges: []domain.Edge{},
			},
			expectError: true,
		},
		{
			name: "workflow with cycle",
			workflow: &domain.Workflow{
				Steps: []domain.Step{stepA, stepB},
				Edges: []domain.Edge{
					{SourceStepID: &stepA.ID, TargetStepID: &stepB.ID},
					{SourceStepID: &stepB.ID, TargetStepID: &stepA.ID},
				},
			},
			expectError: true,
			errorType:   domain.ErrWorkflowHasCycle,
		},
		{
			name: "workflow with unconnected step",
			workflow: &domain.Workflow{
				Steps: []domain.Step{stepA, stepB, stepC},
				Edges: []domain.Edge{
					{SourceStepID: &stepA.ID, TargetStepID: &stepB.ID},
					// stepC is not connected
				},
			},
			expectError: true,
			errorType:   domain.ErrWorkflowHasUnconnected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := usecase.ValidateDAG(tt.workflow)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// BlockGroups Tests
// =============================================================================

func TestWorkflowUsecase_ValidateDAG_WithBlockGroups(t *testing.T) {
	startStep := domain.Step{ID: uuid.New(), Name: "Start", Type: domain.StepTypeStart}
	initStep := domain.Step{ID: uuid.New(), Name: "Init", Type: domain.StepTypeFunction}
	groupID := uuid.New()
	branchA := domain.Step{ID: uuid.New(), Name: "Branch A", Type: domain.StepTypeFunction, BlockGroupID: &groupID}
	branchB := domain.Step{ID: uuid.New(), Name: "Branch B", Type: domain.StepTypeFunction, BlockGroupID: &groupID}
	// Note: join step has been removed as it is no longer supported.
	// Block Group outputs are already aggregated, so we use a function step to process results.
	processStep := domain.Step{ID: uuid.New(), Name: "Process Results", Type: domain.StepTypeFunction}

	usecase := &WorkflowUsecase{}

	tests := []struct {
		name        string
		workflow    *domain.Workflow
		expectError bool
		description string
	}{
		{
			name: "workflow with block group - all steps connected via step edges",
			workflow: &domain.Workflow{
				Steps: []domain.Step{startStep, initStep, branchA, branchB, processStep},
				Edges: []domain.Edge{
					// All connections via step-to-step edges
					{SourceStepID: &startStep.ID, TargetStepID: &initStep.ID},
					{SourceStepID: &initStep.ID, TargetStepID: &branchA.ID},
					{SourceStepID: &initStep.ID, TargetStepID: &branchB.ID},
					{SourceStepID: &branchA.ID, TargetStepID: &processStep.ID},
					{SourceStepID: &branchB.ID, TargetStepID: &processStep.ID},
				},
				BlockGroups: []domain.BlockGroup{
					{
						ID:     groupID,
						Name:   "Parallel Group",
						Type:   domain.BlockGroupTypeParallel,
					},
				},
			},
			expectError: false,
			description: "ValidateDAG checks step-to-step connectivity only",
		},
		{
			name: "workflow with block group edges - steps in group treated as connected",
			workflow: &domain.Workflow{
				// Steps in a group are considered connected if there's at least
				// one edge going into or out of the group
				Steps: []domain.Step{startStep, branchA, branchB},
				Edges: []domain.Edge{
					// start connects to both branches
					{SourceStepID: &startStep.ID, TargetStepID: &branchA.ID},
					{SourceStepID: &startStep.ID, TargetStepID: &branchB.ID},
				},
				BlockGroups: []domain.BlockGroup{
					{
						ID:   groupID,
						Name: "Parallel Group",
						Type: domain.BlockGroupTypeParallel,
					},
				},
			},
			expectError: false,
			description: "Steps with direct step-to-step edges are connected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := usecase.ValidateDAG(tt.workflow)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

func TestWorkflowDefinition_WithBlockGroups(t *testing.T) {
	groupID := uuid.New()
	stepID := uuid.New()

	def := domain.WorkflowDefinition{
		Name:        "Test Workflow",
		Description: "Test Description",
		Steps: []domain.Step{
			{ID: stepID, Name: "Step 1", Type: domain.StepTypeFunction, BlockGroupID: &groupID},
		},
		Edges: []domain.Edge{
			{SourceBlockGroupID: &groupID, TargetStepID: &stepID},
		},
		BlockGroups: []domain.BlockGroup{
			{
				ID:     groupID,
				Name:   "Parallel Group",
				Type:   domain.BlockGroupTypeParallel,
			},
			{
				ID:     uuid.New(),
				Name:   "ForEach Group",
				Type:   domain.BlockGroupTypeForeach,
			},
		},
	}

	// Verify BlockGroups field is properly set
	assert.Len(t, def.BlockGroups, 2)
	assert.Equal(t, "Parallel Group", def.BlockGroups[0].Name)
	assert.Equal(t, domain.BlockGroupTypeParallel, def.BlockGroups[0].Type)
	assert.Equal(t, "ForEach Group", def.BlockGroups[1].Name)
	assert.Equal(t, domain.BlockGroupTypeForeach, def.BlockGroups[1].Type)

	// Verify step references group
	assert.NotNil(t, def.Steps[0].BlockGroupID)
	assert.Equal(t, groupID, *def.Steps[0].BlockGroupID)
}
