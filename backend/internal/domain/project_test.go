package domain

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestNewProject(t *testing.T) {
	tenantID := uuid.New()
	name := "Test Project"
	description := "Test Description"

	project := NewProject(tenantID, name, description)

	if project.ID == uuid.Nil {
		t.Error("NewProject() should generate a non-nil UUID")
	}
	if project.TenantID != tenantID {
		t.Errorf("NewProject() TenantID = %v, want %v", project.TenantID, tenantID)
	}
	if project.Name != name {
		t.Errorf("NewProject() Name = %v, want %v", project.Name, name)
	}
	if project.Description != description {
		t.Errorf("NewProject() Description = %v, want %v", project.Description, description)
	}
	if project.Status != ProjectStatusDraft {
		t.Errorf("NewProject() Status = %v, want %v", project.Status, ProjectStatusDraft)
	}
	if project.Version != 0 {
		t.Errorf("NewProject() Version = %v, want 0", project.Version)
	}
	if project.CreatedAt.IsZero() {
		t.Error("NewProject() CreatedAt should not be zero")
	}
	if project.UpdatedAt.IsZero() {
		t.Error("NewProject() UpdatedAt should not be zero")
	}
}

func TestProject_IncrementVersion(t *testing.T) {
	project := NewProject(uuid.New(), "Test", "Desc")
	originalVersion := project.Version

	project.IncrementVersion()

	if project.Version != originalVersion+1 {
		t.Errorf("IncrementVersion() Version = %v, want %v", project.Version, originalVersion+1)
	}
	if project.Status != ProjectStatusPublished {
		t.Errorf("IncrementVersion() Status = %v, want %v", project.Status, ProjectStatusPublished)
	}
	if project.PublishedAt == nil {
		t.Error("IncrementVersion() PublishedAt should not be nil")
	}
}

func TestProject_CanEdit(t *testing.T) {
	project := NewProject(uuid.New(), "Test", "")

	if !project.CanEdit() {
		t.Error("CanEdit() should return true")
	}

	// Even after incrementing version, should still be editable
	project.IncrementVersion()
	if !project.CanEdit() {
		t.Error("CanEdit() should return true even after publishing")
	}
}

func TestProject_HasUnsavedDraft(t *testing.T) {
	tests := []struct {
		name  string
		draft json.RawMessage
		want  bool
	}{
		{"nil draft", nil, false},
		{"null draft", json.RawMessage("null"), false},
		{"empty draft", json.RawMessage("{}"), true},
		{"valid draft", json.RawMessage(`{"name":"test"}`), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := &Project{Draft: tt.draft}
			if got := project.HasUnsavedDraft(); got != tt.want {
				t.Errorf("HasUnsavedDraft() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProject_SetDraft(t *testing.T) {
	project := NewProject(uuid.New(), "Test", "")
	draft := &ProjectDraft{
		Name:        "New Name",
		Description: "New Description",
	}

	err := project.SetDraft(draft)
	if err != nil {
		t.Fatalf("SetDraft() error = %v", err)
	}

	if !project.HasDraft {
		t.Error("SetDraft() should set HasDraft to true")
	}

	got, err := project.GetDraft()
	if err != nil {
		t.Fatalf("GetDraft() error = %v", err)
	}

	if got.Name != draft.Name {
		t.Errorf("GetDraft() Name = %v, want %v", got.Name, draft.Name)
	}
}

func TestProject_ClearDraft(t *testing.T) {
	project := NewProject(uuid.New(), "Test", "")
	project.SetDraft(&ProjectDraft{Name: "Draft"})

	project.ClearDraft()

	if project.HasDraft {
		t.Error("ClearDraft() should set HasDraft to false")
	}
	if project.Draft != nil {
		t.Error("ClearDraft() should set Draft to nil")
	}
}

func TestProject_GetStartSteps(t *testing.T) {
	project := NewProject(uuid.New(), "Test", "")
	project.Steps = []Step{
		{ID: uuid.New(), Type: StepTypeStart, Name: "Start 1"},
		{ID: uuid.New(), Type: StepTypeLLM, Name: "LLM"},
		{ID: uuid.New(), Type: StepTypeStart, Name: "Start 2"},
		{ID: uuid.New(), Type: StepTypeTool, Name: "Tool"},
	}

	startSteps := project.GetStartSteps()

	if len(startSteps) != 2 {
		t.Errorf("GetStartSteps() returned %d steps, want 2", len(startSteps))
	}

	for _, step := range startSteps {
		if step.Type != StepTypeStart {
			t.Errorf("GetStartSteps() returned non-start step: %v", step.Type)
		}
	}
}

func TestProjectStatus_Validity(t *testing.T) {
	tests := []struct {
		status ProjectStatus
		want   bool
	}{
		{ProjectStatusDraft, true},
		{ProjectStatusPublished, true},
		{ProjectStatus("invalid"), false},
		{ProjectStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			isValid := tt.status == ProjectStatusDraft || tt.status == ProjectStatusPublished
			if isValid != tt.want {
				t.Errorf("ProjectStatus validity = %v, want %v", isValid, tt.want)
			}
		})
	}
}

func TestProject_GetDraft_Nil(t *testing.T) {
	project := NewProject(uuid.New(), "Test", "")

	draft, err := project.GetDraft()
	if err != nil {
		t.Fatalf("GetDraft() error = %v", err)
	}
	if draft != nil {
		t.Error("GetDraft() should return nil for project without draft")
	}
}

func TestProject_SetDraft_Nil(t *testing.T) {
	project := NewProject(uuid.New(), "Test", "")
	project.SetDraft(&ProjectDraft{Name: "Draft"})

	err := project.SetDraft(nil)
	if err != nil {
		t.Fatalf("SetDraft(nil) error = %v", err)
	}

	if project.HasDraft {
		t.Error("SetDraft(nil) should set HasDraft to false")
	}
}
