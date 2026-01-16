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

// edgeTestWorkflowRepo is a mock implementation of WorkflowRepository for edge tests
type edgeTestWorkflowRepo struct {
	workflows map[uuid.UUID]*domain.Workflow
}

func newEdgeTestWorkflowRepo() *edgeTestWorkflowRepo {
	return &edgeTestWorkflowRepo{
		workflows: make(map[uuid.UUID]*domain.Workflow),
	}
}

func (m *edgeTestWorkflowRepo) Create(ctx context.Context, workflow *domain.Workflow) error {
	m.workflows[workflow.ID] = workflow
	return nil
}

func (m *edgeTestWorkflowRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Workflow, error) {
	if wf, ok := m.workflows[id]; ok {
		if wf.TenantID == tenantID {
			return wf, nil
		}
	}
	return nil, domain.ErrWorkflowNotFound
}

func (m *edgeTestWorkflowRepo) List(ctx context.Context, tenantID uuid.UUID, filter repository.WorkflowFilter) ([]*domain.Workflow, int, error) {
	return nil, 0, nil
}

func (m *edgeTestWorkflowRepo) Update(ctx context.Context, workflow *domain.Workflow) error {
	m.workflows[workflow.ID] = workflow
	return nil
}

func (m *edgeTestWorkflowRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	delete(m.workflows, id)
	return nil
}

func (m *edgeTestWorkflowRepo) GetWithStepsAndEdges(ctx context.Context, tenantID, id uuid.UUID) (*domain.Workflow, error) {
	return m.GetByID(ctx, tenantID, id)
}

func (m *edgeTestWorkflowRepo) GetSystemBySlug(ctx context.Context, slug string) (*domain.Workflow, error) {
	return nil, domain.ErrWorkflowNotFound
}

// edgeTestStepRepo is a mock implementation of StepRepository for edge tests
type edgeTestStepRepo struct {
	steps map[uuid.UUID]*domain.Step
}

func newEdgeTestStepRepo() *edgeTestStepRepo {
	return &edgeTestStepRepo{
		steps: make(map[uuid.UUID]*domain.Step),
	}
}

func (m *edgeTestStepRepo) Create(ctx context.Context, step *domain.Step) error {
	m.steps[step.ID] = step
	return nil
}

func (m *edgeTestStepRepo) GetByID(ctx context.Context, tenantID, workflowID, id uuid.UUID) (*domain.Step, error) {
	if s, ok := m.steps[id]; ok {
		if s.TenantID == tenantID && s.WorkflowID == workflowID {
			return s, nil
		}
	}
	return nil, domain.ErrStepNotFound
}

func (m *edgeTestStepRepo) ListByWorkflow(ctx context.Context, tenantID, workflowID uuid.UUID) ([]*domain.Step, error) {
	var result []*domain.Step
	for _, s := range m.steps {
		if s.TenantID == tenantID && s.WorkflowID == workflowID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *edgeTestStepRepo) Update(ctx context.Context, step *domain.Step) error {
	m.steps[step.ID] = step
	return nil
}

func (m *edgeTestStepRepo) Delete(ctx context.Context, tenantID, workflowID, id uuid.UUID) error {
	delete(m.steps, id)
	return nil
}

func (m *edgeTestStepRepo) ListByBlockGroup(ctx context.Context, tenantID, blockGroupID uuid.UUID) ([]*domain.Step, error) {
	var result []*domain.Step
	for _, s := range m.steps {
		if s.TenantID == tenantID && s.BlockGroupID != nil && *s.BlockGroupID == blockGroupID {
			result = append(result, s)
		}
	}
	return result, nil
}

// edgeTestEdgeRepo is a mock implementation of EdgeRepository for edge tests
type edgeTestEdgeRepo struct {
	edges map[uuid.UUID]*domain.Edge
}

func newEdgeTestEdgeRepo() *edgeTestEdgeRepo {
	return &edgeTestEdgeRepo{
		edges: make(map[uuid.UUID]*domain.Edge),
	}
}

func (m *edgeTestEdgeRepo) Create(ctx context.Context, edge *domain.Edge) error {
	m.edges[edge.ID] = edge
	return nil
}

func (m *edgeTestEdgeRepo) ListByWorkflow(ctx context.Context, tenantID, workflowID uuid.UUID) ([]*domain.Edge, error) {
	var result []*domain.Edge
	for _, e := range m.edges {
		if e.TenantID == tenantID && e.WorkflowID == workflowID {
			result = append(result, e)
		}
	}
	return result, nil
}

func (m *edgeTestEdgeRepo) Delete(ctx context.Context, tenantID, workflowID, id uuid.UUID) error {
	delete(m.edges, id)
	return nil
}

func (m *edgeTestEdgeRepo) GetByID(ctx context.Context, tenantID, workflowID, id uuid.UUID) (*domain.Edge, error) {
	if e, ok := m.edges[id]; ok {
		if e.TenantID == tenantID && e.WorkflowID == workflowID {
			return e, nil
		}
	}
	return nil, domain.ErrEdgeNotFound
}

func (m *edgeTestEdgeRepo) Exists(ctx context.Context, tenantID, workflowID, sourceID, targetID uuid.UUID) (bool, error) {
	for _, e := range m.edges {
		if e.TenantID == tenantID && e.WorkflowID == workflowID {
			if e.SourceStepID != nil && e.TargetStepID != nil {
				if *e.SourceStepID == sourceID && *e.TargetStepID == targetID {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

// edgeTestBlockGroupRepo is a mock implementation of BlockGroupRepository for edge tests
type edgeTestBlockGroupRepo struct {
	groups map[uuid.UUID]*domain.BlockGroup
}

func newEdgeTestBlockGroupRepo() *edgeTestBlockGroupRepo {
	return &edgeTestBlockGroupRepo{
		groups: make(map[uuid.UUID]*domain.BlockGroup),
	}
}

func (m *edgeTestBlockGroupRepo) Create(ctx context.Context, group *domain.BlockGroup) error {
	m.groups[group.ID] = group
	return nil
}

func (m *edgeTestBlockGroupRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.BlockGroup, error) {
	if g, ok := m.groups[id]; ok {
		if g.TenantID == tenantID {
			return g, nil
		}
	}
	return nil, domain.ErrBlockGroupNotFound
}

func (m *edgeTestBlockGroupRepo) ListByWorkflow(ctx context.Context, tenantID, workflowID uuid.UUID) ([]*domain.BlockGroup, error) {
	var result []*domain.BlockGroup
	for _, g := range m.groups {
		if g.TenantID == tenantID && g.WorkflowID == workflowID {
			result = append(result, g)
		}
	}
	return result, nil
}

func (m *edgeTestBlockGroupRepo) ListByParent(ctx context.Context, tenantID, parentID uuid.UUID) ([]*domain.BlockGroup, error) {
	return nil, nil
}

func (m *edgeTestBlockGroupRepo) Update(ctx context.Context, group *domain.BlockGroup) error {
	m.groups[group.ID] = group
	return nil
}

func (m *edgeTestBlockGroupRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	delete(m.groups, id)
	return nil
}

// edgeTestBlockDefRepo is a mock implementation of BlockDefinitionRepository for edge tests
type edgeTestBlockDefRepo struct {
	blocks map[string]*domain.BlockDefinition // key is slug
}

func newEdgeTestBlockDefRepo() *edgeTestBlockDefRepo {
	return &edgeTestBlockDefRepo{
		blocks: make(map[string]*domain.BlockDefinition),
	}
}

func (m *edgeTestBlockDefRepo) Create(ctx context.Context, block *domain.BlockDefinition) error {
	m.blocks[block.Slug] = block
	return nil
}

func (m *edgeTestBlockDefRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.BlockDefinition, error) {
	for _, b := range m.blocks {
		if b.ID == id {
			return b, nil
		}
	}
	return nil, domain.ErrBlockDefinitionNotFound
}

func (m *edgeTestBlockDefRepo) GetBySlug(ctx context.Context, tenantID *uuid.UUID, slug string) (*domain.BlockDefinition, error) {
	if b, ok := m.blocks[slug]; ok {
		return b, nil
	}
	return nil, domain.ErrBlockDefinitionNotFound
}

func (m *edgeTestBlockDefRepo) List(ctx context.Context, tenantID *uuid.UUID, filter repository.BlockDefinitionFilter) ([]*domain.BlockDefinition, error) {
	var result []*domain.BlockDefinition
	for _, b := range m.blocks {
		result = append(result, b)
	}
	return result, nil
}

func (m *edgeTestBlockDefRepo) Update(ctx context.Context, block *domain.BlockDefinition) error {
	m.blocks[block.Slug] = block
	return nil
}

func (m *edgeTestBlockDefRepo) Delete(ctx context.Context, id uuid.UUID) error {
	for slug, b := range m.blocks {
		if b.ID == id {
			delete(m.blocks, slug)
			return nil
		}
	}
	return nil
}

func (m *edgeTestBlockDefRepo) ValidateInheritance(ctx context.Context, blockID uuid.UUID, parentBlockID uuid.UUID) error {
	return nil
}

func TestEdgeUsecase_Create_PortValidation(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	workflowID := uuid.New()

	// Setup repositories
	workflowRepo := newEdgeTestWorkflowRepo()
	stepRepo := newEdgeTestStepRepo()
	edgeRepo := newEdgeTestEdgeRepo()
	blockGroupRepo := newEdgeTestBlockGroupRepo()
	blockDefRepo := newEdgeTestBlockDefRepo()

	// Create workflow
	workflow := &domain.Workflow{
		ID:       workflowID,
		TenantID: tenantID,
		Name:     "Test Workflow",
		Status:   domain.WorkflowStatusDraft,
	}
	require.NoError(t, workflowRepo.Create(ctx, workflow))

	// Create block definitions
	functionBlock := &domain.BlockDefinition{
		ID:   uuid.New(),
		Slug: "function",
		OutputPorts: []domain.OutputPort{
			{Name: "output", Label: "Output", IsDefault: true},
		},
		InputPorts: []domain.InputPort{
			{Name: "input", Label: "Input"},
		},
	}
	require.NoError(t, blockDefRepo.Create(ctx, functionBlock))

	tryCatchBlock := &domain.BlockDefinition{
		ID:   uuid.New(),
		Slug: "try_catch",
		OutputPorts: []domain.OutputPort{
			{Name: "out", Label: "Success", IsDefault: true},
			{Name: "error", Label: "Error"},
		},
		InputPorts: []domain.InputPort{
			{Name: "in", Label: "Input"},
		},
	}
	require.NoError(t, blockDefRepo.Create(ctx, tryCatchBlock))

	// Create steps
	sourceStep := domain.NewStep(tenantID, workflowID, "Source", domain.StepTypeFunction, json.RawMessage(`{}`))
	targetStep := domain.NewStep(tenantID, workflowID, "Target", domain.StepTypeFunction, json.RawMessage(`{}`))
	sourceStep2 := domain.NewStep(tenantID, workflowID, "Source2", domain.StepTypeFunction, json.RawMessage(`{}`))
	sourceStep3 := domain.NewStep(tenantID, workflowID, "Source3", domain.StepTypeFunction, json.RawMessage(`{}`))
	require.NoError(t, stepRepo.Create(ctx, sourceStep))
	require.NoError(t, stepRepo.Create(ctx, targetStep))
	require.NoError(t, stepRepo.Create(ctx, sourceStep2))
	require.NoError(t, stepRepo.Create(ctx, sourceStep3))

	// Create block group
	tryCatchGroup := &domain.BlockGroup{
		ID:         uuid.New(),
		TenantID:   tenantID,
		WorkflowID: workflowID,
		Name:       "Error Handling",
		Type:       domain.BlockGroupTypeTryCatch,
	}
	require.NoError(t, blockGroupRepo.Create(ctx, tryCatchGroup))

	// Create usecase with all repos
	uc := NewEdgeUsecase(workflowRepo, stepRepo, edgeRepo).
		WithBlockGroupRepo(blockGroupRepo).
		WithBlockDefinitionRepo(blockDefRepo)

	t.Run("valid step-to-step edge with valid ports", func(t *testing.T) {
		input := CreateEdgeInput{
			TenantID:     tenantID,
			WorkflowID:   workflowID,
			SourceStepID: &sourceStep.ID,
			TargetStepID: &targetStep.ID,
			SourcePort:   "output",
			TargetPort:   "input",
		}
		edge, err := uc.Create(ctx, input)
		require.NoError(t, err)
		assert.NotNil(t, edge)
	})

	t.Run("invalid source port returns error", func(t *testing.T) {
		input := CreateEdgeInput{
			TenantID:     tenantID,
			WorkflowID:   workflowID,
			SourceStepID: &sourceStep.ID,
			TargetStepID: &targetStep.ID,
			SourcePort:   "nonexistent",
			TargetPort:   "input",
		}
		edge, err := uc.Create(ctx, input)
		assert.ErrorIs(t, err, domain.ErrSourcePortNotFound)
		assert.Nil(t, edge)
	})

	t.Run("invalid target port returns error", func(t *testing.T) {
		input := CreateEdgeInput{
			TenantID:     tenantID,
			WorkflowID:   workflowID,
			SourceStepID: &sourceStep.ID,
			TargetStepID: &targetStep.ID,
			SourcePort:   "output",
			TargetPort:   "nonexistent",
		}
		edge, err := uc.Create(ctx, input)
		assert.ErrorIs(t, err, domain.ErrTargetPortNotFound)
		assert.Nil(t, edge)
	})

	t.Run("group-to-step with valid output port", func(t *testing.T) {
		input := CreateEdgeInput{
			TenantID:           tenantID,
			WorkflowID:         workflowID,
			SourceBlockGroupID: &tryCatchGroup.ID,
			TargetStepID:       &targetStep.ID,
			SourcePort:         "out",
			TargetPort:         "input",
		}
		edge, err := uc.Create(ctx, input)
		require.NoError(t, err)
		assert.NotNil(t, edge)
	})

	t.Run("group-to-step with invalid output port (caught) returns error", func(t *testing.T) {
		input := CreateEdgeInput{
			TenantID:           tenantID,
			WorkflowID:         workflowID,
			SourceBlockGroupID: &tryCatchGroup.ID,
			TargetStepID:       &targetStep.ID,
			SourcePort:         "caught", // This port doesn't exist
			TargetPort:         "input",
		}
		edge, err := uc.Create(ctx, input)
		assert.ErrorIs(t, err, domain.ErrSourcePortNotFound)
		assert.Nil(t, edge)
	})

	t.Run("step-to-group with group-input port is allowed", func(t *testing.T) {
		// Use sourceStep2 to avoid conflict with existing edge from sourceStep
		input := CreateEdgeInput{
			TenantID:           tenantID,
			WorkflowID:         workflowID,
			SourceStepID:       &sourceStep2.ID,
			TargetBlockGroupID: &tryCatchGroup.ID,
			SourcePort:         "output",
			TargetPort:         "group-input",
		}
		edge, err := uc.Create(ctx, input)
		require.NoError(t, err)
		assert.NotNil(t, edge)
	})

	t.Run("empty port skips validation (default port)", func(t *testing.T) {
		// Use sourceStep3 to avoid conflict with existing edges
		input := CreateEdgeInput{
			TenantID:     tenantID,
			WorkflowID:   workflowID,
			SourceStepID: &sourceStep3.ID,
			TargetStepID: &targetStep.ID,
			SourcePort:   "",
			TargetPort:   "",
		}
		edge, err := uc.Create(ctx, input)
		require.NoError(t, err)
		assert.NotNil(t, edge)
	})
}
