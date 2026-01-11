package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestWorkflow_CanEdit(t *testing.T) {
	// All workflows are always editable - versioning handles history
	tests := []struct {
		name     string
		status   WorkflowStatus
		expected bool
	}{
		{
			name:     "draft is editable",
			status:   WorkflowStatusDraft,
			expected: true,
		},
		{
			name:     "published is also editable",
			status:   WorkflowStatusPublished,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Workflow{Status: tt.status}
			assert.Equal(t, tt.expected, w.CanEdit())
		})
	}
}

func TestNewWorkflow(t *testing.T) {
	tenantID := uuid.New()

	w := NewWorkflow(tenantID, "Test Workflow", "Description")

	assert.NotEqual(t, uuid.Nil, w.ID)
	assert.Equal(t, tenantID, w.TenantID)
	assert.Equal(t, "Test Workflow", w.Name)
	assert.Equal(t, "Description", w.Description)
	assert.Equal(t, WorkflowStatusDraft, w.Status)
	assert.Equal(t, 0, w.Version) // New workflow starts at version 0
	assert.False(t, w.CreatedAt.IsZero())
	assert.False(t, w.UpdatedAt.IsZero())
}

func TestWorkflow_IncrementVersion(t *testing.T) {
	t.Run("increment version from 0", func(t *testing.T) {
		w := &Workflow{
			Status:    WorkflowStatusDraft,
			Version:   0,
			UpdatedAt: time.Now().Add(-time.Hour),
		}

		w.IncrementVersion()
		assert.Equal(t, 1, w.Version)
		assert.Equal(t, WorkflowStatusPublished, w.Status)
		assert.NotNil(t, w.PublishedAt)
		assert.True(t, w.UpdatedAt.After(time.Now().Add(-time.Minute)))
	})

	t.Run("increment version from 1", func(t *testing.T) {
		oldPublishedAt := time.Now().Add(-time.Hour)
		w := &Workflow{
			Status:      WorkflowStatusPublished,
			Version:     1,
			PublishedAt: &oldPublishedAt,
			UpdatedAt:   oldPublishedAt,
		}

		w.IncrementVersion()
		assert.Equal(t, 2, w.Version)
		assert.Equal(t, WorkflowStatusPublished, w.Status)
		assert.NotNil(t, w.PublishedAt)
		// PublishedAt should be updated to new time
		assert.True(t, w.PublishedAt.After(oldPublishedAt))
	})
}

func TestWorkflow_Draft(t *testing.T) {
	t.Run("HasUnsavedDraft returns false for nil draft", func(t *testing.T) {
		w := &Workflow{Draft: nil}
		assert.False(t, w.HasUnsavedDraft())
	})

	t.Run("HasUnsavedDraft returns false for null draft", func(t *testing.T) {
		w := &Workflow{Draft: json.RawMessage("null")}
		assert.False(t, w.HasUnsavedDraft())
	})

	t.Run("HasUnsavedDraft returns true for valid draft", func(t *testing.T) {
		w := &Workflow{Draft: json.RawMessage(`{"name":"test"}`)}
		assert.True(t, w.HasUnsavedDraft())
	})

	t.Run("SetDraft and GetDraft", func(t *testing.T) {
		w := &Workflow{}
		draft := &WorkflowDraft{
			Name:        "Draft Name",
			Description: "Draft Description",
			Steps:       []Step{},
			Edges:       []Edge{},
			UpdatedAt:   time.Now(),
		}

		err := w.SetDraft(draft)
		assert.NoError(t, err)
		assert.True(t, w.HasDraft)
		assert.True(t, w.HasUnsavedDraft())

		retrieved, err := w.GetDraft()
		assert.NoError(t, err)
		assert.Equal(t, "Draft Name", retrieved.Name)
		assert.Equal(t, "Draft Description", retrieved.Description)
	})

	t.Run("ClearDraft", func(t *testing.T) {
		w := &Workflow{
			Draft:    json.RawMessage(`{"name":"test"}`),
			HasDraft: true,
		}

		w.ClearDraft()
		assert.Nil(t, w.Draft)
		assert.False(t, w.HasDraft)
		assert.False(t, w.HasUnsavedDraft())
	})
}

func TestWorkflowStatus_Values(t *testing.T) {
	assert.Equal(t, WorkflowStatus("draft"), WorkflowStatusDraft)
	assert.Equal(t, WorkflowStatus("published"), WorkflowStatusPublished)
}
