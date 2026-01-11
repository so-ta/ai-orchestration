package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewEdge(t *testing.T) {
	workflowID := uuid.New()
	sourceID := uuid.New()
	targetID := uuid.New()

	edge := NewEdge(workflowID, sourceID, targetID, "condition == true")

	assert.NotEqual(t, uuid.Nil, edge.ID)
	assert.Equal(t, workflowID, edge.WorkflowID)
	assert.Equal(t, sourceID, edge.SourceStepID)
	assert.Equal(t, targetID, edge.TargetStepID)
	assert.Equal(t, "condition == true", edge.Condition)
	assert.False(t, edge.CreatedAt.IsZero())
}

func TestNewEdge_WithoutCondition(t *testing.T) {
	workflowID := uuid.New()
	sourceID := uuid.New()
	targetID := uuid.New()

	edge := NewEdge(workflowID, sourceID, targetID, "")

	assert.NotEqual(t, uuid.Nil, edge.ID)
	assert.Equal(t, workflowID, edge.WorkflowID)
	assert.Equal(t, sourceID, edge.SourceStepID)
	assert.Equal(t, targetID, edge.TargetStepID)
	assert.Empty(t, edge.Condition)
}
