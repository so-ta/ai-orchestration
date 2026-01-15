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
