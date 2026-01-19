package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ProjectStatus represents the status of a project
type ProjectStatus string

const (
	ProjectStatusDraft     ProjectStatus = "draft"
	ProjectStatusPublished ProjectStatus = "published"
)

// ProjectDraft represents the draft state of a project (unsaved changes)
type ProjectDraft struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Variables   json.RawMessage `json:"variables,omitempty"`
	Steps       []Step          `json:"steps"`
	Edges       []Edge          `json:"edges"`
	BlockGroups []BlockGroup    `json:"block_groups,omitempty"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// Project represents a multi-start DAG project
// A project can have multiple Start blocks, each with its own trigger configuration
type Project struct {
	ID          uuid.UUID       `json:"id"`
	TenantID    uuid.UUID       `json:"tenant_id"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Status      ProjectStatus   `json:"status"`
	Version     int             `json:"version"`             // Current saved version (0 = never saved)
	Variables   json.RawMessage `json:"variables,omitempty"` // Project-level shared variables
	Draft       json.RawMessage `json:"draft,omitempty"`     // Draft state (unsaved changes)
	CreatedBy   *uuid.UUID      `json:"created_by,omitempty"`
	PublishedAt *time.Time      `json:"published_at,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	DeletedAt   *time.Time      `json:"deleted_at,omitempty"`

	// System project fields
	IsSystem   bool    `json:"is_system"`             // True for system projects (e.g., Copilot)
	SystemSlug *string `json:"system_slug,omitempty"` // Unique slug for system projects (e.g., "copilot-generate")

	// Error Workflow configuration
	ErrorWorkflowID     *uuid.UUID      `json:"error_workflow_id,omitempty"`     // Project to execute on failure
	ErrorWorkflowConfig json.RawMessage `json:"error_workflow_config,omitempty"` // Error workflow configuration

	// Loaded relations
	Steps       []Step       `json:"steps,omitempty"`
	Edges       []Edge       `json:"edges,omitempty"`
	BlockGroups []BlockGroup `json:"block_groups,omitempty"`

	// Indicates if current state is from draft
	HasDraft bool `json:"has_draft"`
}

// NewProject creates a new project
func NewProject(tenantID uuid.UUID, name, description string) *Project {
	now := time.Now().UTC()
	return &Project{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        name,
		Description: description,
		Status:      ProjectStatusDraft,
		Version:     0, // No versions yet
		Variables:   json.RawMessage(`{}`),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// CanEdit returns true if the project can be edited
// Projects are always editable - versioning handles history
func (p *Project) CanEdit() bool {
	return true
}

// HasUnsavedDraft returns true if the project has unsaved draft changes
func (p *Project) HasUnsavedDraft() bool {
	return len(p.Draft) > 0 && string(p.Draft) != "null"
}

// GetDraft unmarshals and returns the draft state
func (p *Project) GetDraft() (*ProjectDraft, error) {
	if !p.HasUnsavedDraft() {
		return nil, nil
	}
	var draft ProjectDraft
	if err := json.Unmarshal(p.Draft, &draft); err != nil {
		return nil, err
	}
	return &draft, nil
}

// SetDraft marshals and sets the draft state
func (p *Project) SetDraft(draft *ProjectDraft) error {
	if draft == nil {
		p.Draft = nil
		p.HasDraft = false
		return nil
	}
	data, err := json.Marshal(draft)
	if err != nil {
		return err
	}
	p.Draft = data
	p.HasDraft = true
	return nil
}

// ClearDraft clears the draft state
func (p *Project) ClearDraft() {
	p.Draft = nil
	p.HasDraft = false
}

// IncrementVersion increments the version number (called on Save)
func (p *Project) IncrementVersion() {
	p.Version++
	now := time.Now().UTC()
	p.UpdatedAt = now
	// Set status to published when first version is saved
	if p.Version >= 1 {
		p.Status = ProjectStatusPublished
		p.PublishedAt = &now
	}
}

// GetStartSteps returns all Start blocks in this project
func (p *Project) GetStartSteps() []Step {
	var startSteps []Step
	for _, step := range p.Steps {
		if step.Type == StepTypeStart {
			startSteps = append(startSteps, step)
		}
	}
	return startSteps
}

// ProjectVersion represents an immutable snapshot of a saved project
type ProjectVersion struct {
	ID         uuid.UUID       `json:"id"`
	ProjectID  uuid.UUID       `json:"project_id"`
	Version    int             `json:"version"`
	Definition json.RawMessage `json:"definition"`
	SavedBy    *uuid.UUID      `json:"saved_by,omitempty"`
	SavedAt    time.Time       `json:"saved_at"`
}

// ProjectDefinition contains the complete project structure for versioning
type ProjectDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Variables   json.RawMessage `json:"variables,omitempty"`
	Steps       []Step          `json:"steps"`
	Edges       []Edge          `json:"edges"`
	BlockGroups []BlockGroup    `json:"block_groups,omitempty"`
}
