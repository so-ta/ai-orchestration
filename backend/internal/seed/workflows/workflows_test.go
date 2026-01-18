package workflows

import (
	"testing"
)

func TestRegistry_AllWorkflowsValid(t *testing.T) {
	registry := NewRegistry()

	for _, wf := range registry.GetAll() {
		if err := wf.Validate(); err != nil {
			t.Errorf("Workflow %s failed validation: %v", wf.SystemSlug, err)
		}
	}
}

func TestRegistry_WorkflowCount(t *testing.T) {
	registry := NewRegistry()

	// Expect at least 5 workflows (1 copilot + 1 RAG + 2 demo + 1 block-group-demo)
	// Note: Copilot workflows are unified into a single workflow with 4 entry points
	// Note: RAG workflows are unified into a single workflow with 3 entry points
	// Note: ai-routing-block-demo and control-flow-block-demo were removed
	// because they used join blocks which are no longer supported
	minExpected := 5
	actual := registry.Count()
	if actual < minExpected {
		t.Errorf("Expected at least %d workflows, got %d", minExpected, actual)
	}
}

func TestRegistry_GetBySlug(t *testing.T) {
	registry := NewRegistry()

	// Test known workflows exist
	// Note: Copilot workflows are unified into a single "copilot" workflow
	// Note: RAG workflows are unified into a single "rag" workflow
	// Note: ai-routing-block-demo and control-flow-block-demo were removed
	// because they used join blocks which are no longer supported
	knownSlugs := []string{
		"copilot", // Unified Copilot workflow with 4 entry points
		"rag",     // Unified RAG workflow with 3 entry points
		"comprehensive-block-demo",
		"data-pipeline-block-demo",
		"block-group-demo",
	}

	for _, slug := range knownSlugs {
		wf, ok := registry.GetBySlug(slug)
		if !ok {
			t.Errorf("Workflow %s not found in registry", slug)
			continue
		}
		if wf.SystemSlug != slug {
			t.Errorf("Workflow %s has wrong system_slug: %s", slug, wf.SystemSlug)
		}
	}
}

func TestRegistry_RequiredFields(t *testing.T) {
	registry := NewRegistry()

	for _, wf := range registry.GetAll() {
		t.Run(wf.SystemSlug, func(t *testing.T) {
			if wf.ID == "" {
				t.Error("ID is required")
			}
			if wf.SystemSlug == "" {
				t.Error("SystemSlug is required")
			}
			if wf.Name == "" {
				t.Error("Name is required")
			}
			if wf.Description == "" {
				t.Error("Description is required")
			}
			if !wf.IsSystem {
				t.Error("IsSystem should be true for system workflows")
			}
			if len(wf.Steps) == 0 {
				t.Error("Steps are required")
			}

			// Check that each workflow has a start step
			hasStart := false
			for _, step := range wf.Steps {
				if step.Type == "start" {
					hasStart = true
					break
				}
			}
			if !hasStart {
				t.Error("Workflow must have a start step")
			}
		})
	}
}

func TestWorkflow_Validate(t *testing.T) {
	tests := []struct {
		name    string
		wf      *SystemWorkflowDefinition
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid workflow",
			wf: &SystemWorkflowDefinition{
				SystemSlug: "test-workflow",
				Name:       "Test Workflow",
				Steps: []SystemStepDefinition{
					{TempID: "step_1", Type: "start", Name: "Start"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing system_slug",
			wf: &SystemWorkflowDefinition{
				Name: "Test",
				Steps: []SystemStepDefinition{
					{TempID: "step_1", Type: "start", Name: "Start"},
				},
			},
			wantErr: true,
			errMsg:  "system_slug is required",
		},
		{
			name: "missing name",
			wf: &SystemWorkflowDefinition{
				SystemSlug: "test",
				Steps: []SystemStepDefinition{
					{TempID: "step_1", Type: "start", Name: "Start"},
				},
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "missing steps",
			wf: &SystemWorkflowDefinition{
				SystemSlug: "test",
				Name:       "Test",
				Steps:      []SystemStepDefinition{},
			},
			wantErr: true,
			errMsg:  "at least one step is required",
		},
		{
			name: "missing start step",
			wf: &SystemWorkflowDefinition{
				SystemSlug: "test",
				Name:       "Test",
				Steps: []SystemStepDefinition{
					{TempID: "step_1", Type: "function", Name: "Func"},
				},
			},
			wantErr: true,
			errMsg:  "workflow must have a start step",
		},
		{
			name: "duplicate temp_id",
			wf: &SystemWorkflowDefinition{
				SystemSlug: "test",
				Name:       "Test",
				Steps: []SystemStepDefinition{
					{TempID: "step_1", Type: "start", Name: "Start"},
					{TempID: "step_1", Type: "function", Name: "Func"},
				},
			},
			wantErr: true,
			errMsg:  "duplicate temp_id",
		},
		{
			name: "invalid edge source",
			wf: &SystemWorkflowDefinition{
				SystemSlug: "test",
				Name:       "Test",
				Steps: []SystemStepDefinition{
					{TempID: "step_1", Type: "start", Name: "Start"},
				},
				Edges: []SystemEdgeDefinition{
					{SourceTempID: "invalid", TargetTempID: "step_1"},
				},
			},
			wantErr: true,
			errMsg:  "invalid source_temp_id",
		},
		{
			name: "invalid edge target",
			wf: &SystemWorkflowDefinition{
				SystemSlug: "test",
				Name:       "Test",
				Steps: []SystemStepDefinition{
					{TempID: "step_1", Type: "start", Name: "Start"},
				},
				Edges: []SystemEdgeDefinition{
					{SourceTempID: "step_1", TargetTempID: "invalid"},
				},
			},
			wantErr: true,
			errMsg:  "invalid target_temp_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.wf.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !containsString(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestBlockGroupDemoWorkflow(t *testing.T) {
	registry := NewRegistry()

	wf, ok := registry.GetBySlug("block-group-demo")
	if !ok {
		t.Fatal("block-group-demo workflow not found")
	}

	// Check block groups exist
	if len(wf.BlockGroups) == 0 {
		t.Error("BlockGroups should not be empty")
	}

	// Check all 4 block group types are present
	expectedTypes := map[string]bool{
		"parallel":  false,
		"try_catch": false,
		"foreach":   false,
		"while":     false,
	}

	for _, bg := range wf.BlockGroups {
		if _, exists := expectedTypes[bg.Type]; exists {
			expectedTypes[bg.Type] = true
		}
	}

	for groupType, found := range expectedTypes {
		if !found {
			t.Errorf("Block group type %s not found in workflow", groupType)
		}
	}

	// Check steps with BlockGroupTempID are properly configured
	stepsInGroups := 0
	for _, step := range wf.Steps {
		if step.BlockGroupTempID != "" {
			stepsInGroups++
			// Verify the referenced group exists
			groupFound := false
			for _, bg := range wf.BlockGroups {
				if bg.TempID == step.BlockGroupTempID {
					groupFound = true
					break
				}
			}
			if !groupFound {
				t.Errorf("Step %s references non-existent block group %s", step.Name, step.BlockGroupTempID)
			}
		}
	}

	if stepsInGroups == 0 {
		t.Error("Expected at least one step inside a block group")
	}

	// Validate the workflow
	if err := wf.Validate(); err != nil {
		t.Errorf("Workflow validation failed: %v", err)
	}
}

func TestBlockGroupDefinitionFields(t *testing.T) {
	registry := NewRegistry()

	wf, ok := registry.GetBySlug("block-group-demo")
	if !ok {
		t.Fatal("block-group-demo workflow not found")
	}

	for _, bg := range wf.BlockGroups {
		t.Run(bg.TempID, func(t *testing.T) {
			if bg.TempID == "" {
				t.Error("TempID is required")
			}
			if bg.Name == "" {
				t.Error("Name is required")
			}
			if bg.Type == "" {
				t.Error("Type is required")
			}
			// Width and Height should have defaults or be set
			if bg.Width == 0 && bg.Height == 0 {
				// This is OK - defaults will be applied during migration
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
