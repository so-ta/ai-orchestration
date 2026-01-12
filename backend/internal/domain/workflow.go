package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// WorkflowStatus represents the status of a workflow
// Note: "draft" status is kept for backward compatibility but the new model uses
// the Draft JSONB field for unsaved changes and Version for saved snapshots
type WorkflowStatus string

const (
	WorkflowStatusDraft     WorkflowStatus = "draft"
	WorkflowStatusPublished WorkflowStatus = "published"
)

// WorkflowDraft represents the draft state of a workflow (unsaved changes)
type WorkflowDraft struct {
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	InputSchema  json.RawMessage `json:"input_schema,omitempty"`
	OutputSchema json.RawMessage `json:"output_schema,omitempty"`
	Steps        []Step          `json:"steps"`
	Edges        []Edge          `json:"edges"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// Workflow represents a DAG workflow
type Workflow struct {
	ID           uuid.UUID       `json:"id"`
	TenantID     uuid.UUID       `json:"tenant_id"`
	Name         string          `json:"name"`
	Description  string          `json:"description,omitempty"`
	Status       WorkflowStatus  `json:"status"`
	Version      int             `json:"version"` // Current saved version (0 = never saved)
	InputSchema  json.RawMessage `json:"input_schema,omitempty"`
	OutputSchema json.RawMessage `json:"output_schema,omitempty"`
	Draft        json.RawMessage `json:"draft,omitempty"` // Draft state (unsaved changes)
	CreatedBy    *uuid.UUID      `json:"created_by,omitempty"`
	PublishedAt  *time.Time      `json:"published_at,omitempty"` // Kept for backward compatibility
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	DeletedAt    *time.Time      `json:"deleted_at,omitempty"`

	// System workflow fields
	IsSystem   bool    `json:"is_system"`             // True for system workflows (e.g., Copilot)
	SystemSlug *string `json:"system_slug,omitempty"` // Unique slug for system workflows (e.g., "copilot-generate")

	// Loaded relations
	Steps []Step `json:"steps,omitempty"`
	Edges []Edge `json:"edges,omitempty"`

	// Indicates if current state is from draft
	HasDraft bool `json:"has_draft"`
}

// NewWorkflow creates a new workflow
func NewWorkflow(tenantID uuid.UUID, name, description string) *Workflow {
	now := time.Now().UTC()
	return &Workflow{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        name,
		Description: description,
		Status:      WorkflowStatusDraft,
		Version:     0, // No versions yet
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// CanEdit returns true if the workflow can be edited
// Workflows are always editable - versioning handles history
func (w *Workflow) CanEdit() bool {
	return true
}

// HasUnsavedDraft returns true if the workflow has unsaved draft changes
func (w *Workflow) HasUnsavedDraft() bool {
	return len(w.Draft) > 0 && string(w.Draft) != "null"
}

// GetDraft unmarshals and returns the draft state
func (w *Workflow) GetDraft() (*WorkflowDraft, error) {
	if !w.HasUnsavedDraft() {
		return nil, nil
	}
	var draft WorkflowDraft
	if err := json.Unmarshal(w.Draft, &draft); err != nil {
		return nil, err
	}
	return &draft, nil
}

// SetDraft marshals and sets the draft state
func (w *Workflow) SetDraft(draft *WorkflowDraft) error {
	if draft == nil {
		w.Draft = nil
		w.HasDraft = false
		return nil
	}
	data, err := json.Marshal(draft)
	if err != nil {
		return err
	}
	w.Draft = data
	w.HasDraft = true
	return nil
}

// ClearDraft clears the draft state
func (w *Workflow) ClearDraft() {
	w.Draft = nil
	w.HasDraft = false
}

// IncrementVersion increments the version number (called on Save)
func (w *Workflow) IncrementVersion() {
	w.Version++
	now := time.Now().UTC()
	w.UpdatedAt = now
	// Set status to published when first version is saved
	if w.Version >= 1 {
		w.Status = WorkflowStatusPublished
		w.PublishedAt = &now
	}
}

// WorkflowVersion represents an immutable snapshot of a saved workflow
type WorkflowVersion struct {
	ID         uuid.UUID       `json:"id"`
	WorkflowID uuid.UUID       `json:"workflow_id"`
	Version    int             `json:"version"`
	Definition json.RawMessage `json:"definition"`
	SavedBy    *uuid.UUID      `json:"saved_by,omitempty"`
	SavedAt    time.Time       `json:"saved_at"`
}

// WorkflowDefinition contains the complete workflow structure for versioning
type WorkflowDefinition struct {
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	InputSchema  json.RawMessage `json:"input_schema,omitempty"`
	OutputSchema json.RawMessage `json:"output_schema,omitempty"`
	Steps        []Step          `json:"steps"`
	Edges        []Edge          `json:"edges"`
}
