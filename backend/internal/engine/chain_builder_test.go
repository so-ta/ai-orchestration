package engine

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChainBuilder_FindEntryPoints(t *testing.T) {
	groupID := uuid.New()

	t.Run("single entry point", func(t *testing.T) {
		// [Step1] -> [Step2]
		step1 := &domain.Step{ID: uuid.New(), Name: "Step1", BlockGroupID: &groupID}
		step2 := &domain.Step{ID: uuid.New(), Name: "Step2", BlockGroupID: &groupID}

		edges := []*domain.Edge{
			{SourceStepID: &step1.ID, TargetStepID: &step2.ID},
		}

		cb := NewChainBuilder([]*domain.Step{step1, step2}, edges)
		entryPoints := cb.FindEntryPoints(groupID)

		require.Len(t, entryPoints, 1)
		assert.Equal(t, "Step1", entryPoints[0].Name)
	})

	t.Run("multiple entry points", func(t *testing.T) {
		// [Step1] -> [Step3]
		// [Step2] (standalone)
		step1 := &domain.Step{ID: uuid.New(), Name: "Step1", BlockGroupID: &groupID}
		step2 := &domain.Step{ID: uuid.New(), Name: "Step2", BlockGroupID: &groupID}
		step3 := &domain.Step{ID: uuid.New(), Name: "Step3", BlockGroupID: &groupID}

		edges := []*domain.Edge{
			{SourceStepID: &step1.ID, TargetStepID: &step3.ID},
		}

		cb := NewChainBuilder([]*domain.Step{step1, step2, step3}, edges)
		entryPoints := cb.FindEntryPoints(groupID)

		require.Len(t, entryPoints, 2)
		names := []string{entryPoints[0].Name, entryPoints[1].Name}
		assert.Contains(t, names, "Step1")
		assert.Contains(t, names, "Step2")
	})

	t.Run("external edge ignored", func(t *testing.T) {
		// Step outside group -> [Step1]
		step1 := &domain.Step{ID: uuid.New(), Name: "Step1", BlockGroupID: &groupID}
		externalStep := &domain.Step{ID: uuid.New(), Name: "External"}

		edges := []*domain.Edge{
			{SourceStepID: &externalStep.ID, TargetStepID: &step1.ID},
		}

		cb := NewChainBuilder([]*domain.Step{step1, externalStep}, edges)
		entryPoints := cb.FindEntryPoints(groupID)

		require.Len(t, entryPoints, 1)
		assert.Equal(t, "Step1", entryPoints[0].Name)
	})
}

func TestChainBuilder_BuildChain(t *testing.T) {
	groupID := uuid.New()

	t.Run("linear chain", func(t *testing.T) {
		// [Step1] -> [Step2] -> [Step3]
		step1 := &domain.Step{ID: uuid.New(), Name: "Step1", BlockGroupID: &groupID}
		step2 := &domain.Step{ID: uuid.New(), Name: "Step2", BlockGroupID: &groupID}
		step3 := &domain.Step{ID: uuid.New(), Name: "Step3", BlockGroupID: &groupID}

		edges := []*domain.Edge{
			{SourceStepID: &step1.ID, TargetStepID: &step2.ID},
			{SourceStepID: &step2.ID, TargetStepID: &step3.ID},
		}

		cb := NewChainBuilder([]*domain.Step{step1, step2, step3}, edges)
		chain := cb.BuildChain(step1)

		require.Len(t, chain, 3)
		assert.Equal(t, "Step1", chain[0].Name)
		assert.Equal(t, "Step2", chain[1].Name)
		assert.Equal(t, "Step3", chain[2].Name)
	})

	t.Run("single step chain", func(t *testing.T) {
		step1 := &domain.Step{ID: uuid.New(), Name: "Step1", BlockGroupID: &groupID}

		cb := NewChainBuilder([]*domain.Step{step1}, nil)
		chain := cb.BuildChain(step1)

		require.Len(t, chain, 1)
		assert.Equal(t, "Step1", chain[0].Name)
	})
}

func TestChainBuilder_BuildToolChains(t *testing.T) {
	groupID := uuid.New()
	toolName := "my_tool"
	toolDesc := "My tool description"

	t.Run("with tool definition fields", func(t *testing.T) {
		step1 := &domain.Step{
			ID:              uuid.New(),
			Name:            "Step1",
			BlockGroupID:    &groupID,
			ToolName:        &toolName,
			ToolDescription: &toolDesc,
			ToolInputSchema: json.RawMessage(`{"type": "object", "properties": {"query": {"type": "string"}}}`),
		}
		step2 := &domain.Step{ID: uuid.New(), Name: "Step2", BlockGroupID: &groupID}

		edges := []*domain.Edge{
			{SourceStepID: &step1.ID, TargetStepID: &step2.ID},
		}

		cb := NewChainBuilder([]*domain.Step{step1, step2}, edges)
		toolChains := cb.BuildToolChains(groupID)

		require.Len(t, toolChains, 1)
		tc := toolChains[0]

		assert.Equal(t, "my_tool", tc.ToolName)
		assert.Equal(t, "My tool description", tc.Description)
		assert.NotNil(t, tc.InputSchema)
		assert.Len(t, tc.Chain, 2)
	})

	t.Run("fallback to step name", func(t *testing.T) {
		step1 := &domain.Step{ID: uuid.New(), Name: "search_blocks", BlockGroupID: &groupID}

		cb := NewChainBuilder([]*domain.Step{step1}, nil)
		toolChains := cb.BuildToolChains(groupID)

		require.Len(t, toolChains, 1)
		tc := toolChains[0]

		assert.Equal(t, "search_blocks", tc.ToolName)
		assert.Equal(t, "search_blocks", tc.Description)
	})
}

func TestChainBuilder_ValidateEntryPointCount(t *testing.T) {
	groupID := uuid.New()

	t.Run("agent group requires at least one", func(t *testing.T) {
		cb := NewChainBuilder([]*domain.Step{}, nil)
		err := cb.ValidateEntryPointCount(groupID, domain.BlockGroupTypeAgent)
		assert.Equal(t, "agent group requires at least one entry point step", err)
	})

	t.Run("foreach requires exactly one", func(t *testing.T) {
		step1 := &domain.Step{ID: uuid.New(), Name: "Step1", BlockGroupID: &groupID}
		step2 := &domain.Step{ID: uuid.New(), Name: "Step2", BlockGroupID: &groupID}

		cb := NewChainBuilder([]*domain.Step{step1, step2}, nil)
		err := cb.ValidateEntryPointCount(groupID, domain.BlockGroupTypeForeach)
		assert.Equal(t, "foreach group requires exactly one entry point", err)
	})

	t.Run("valid single entry point for foreach", func(t *testing.T) {
		step1 := &domain.Step{ID: uuid.New(), Name: "Step1", BlockGroupID: &groupID}

		cb := NewChainBuilder([]*domain.Step{step1}, nil)
		err := cb.ValidateEntryPointCount(groupID, domain.BlockGroupTypeForeach)
		assert.Empty(t, err)
	})

	t.Run("multiple entry points valid for parallel", func(t *testing.T) {
		step1 := &domain.Step{ID: uuid.New(), Name: "Step1", BlockGroupID: &groupID}
		step2 := &domain.Step{ID: uuid.New(), Name: "Step2", BlockGroupID: &groupID}

		cb := NewChainBuilder([]*domain.Step{step1, step2}, nil)
		err := cb.ValidateEntryPointCount(groupID, domain.BlockGroupTypeParallel)
		assert.Empty(t, err)
	})
}
