package workflows

import (
	"encoding/json"
)

// SystemWorkflowDefinition represents a system workflow with its steps and edges
type SystemWorkflowDefinition struct {
	// Workflow metadata
	ID          string `json:"id"`           // Fixed UUID for system workflows
	SystemSlug  string `json:"system_slug"`  // Unique slug for system workflows (e.g., "copilot-generate")
	Name        string `json:"name"`         // Display name
	Description string `json:"description"`  // Description
	Version     int    `json:"version"`      // Workflow version

	// Schema
	InputSchema  json.RawMessage `json:"input_schema,omitempty"`
	OutputSchema json.RawMessage `json:"output_schema,omitempty"`

	// Workflow structure
	Steps []SystemStepDefinition `json:"steps"`
	Edges []SystemEdgeDefinition `json:"edges"`

	// Metadata
	IsSystem bool `json:"is_system"` // Always true for system workflows
}

// SystemStepDefinition represents a step within a system workflow
type SystemStepDefinition struct {
	TempID      string          `json:"temp_id"`                 // Temporary ID for edge references (e.g., "step_1")
	Name        string          `json:"name"`                    // Step name
	Type        string          `json:"type"`                    // Block type (e.g., "start", "llm", "function")
	Config      json.RawMessage `json:"config,omitempty"`        // Step configuration
	PositionX   int             `json:"position_x"`              // Canvas X position
	PositionY   int             `json:"position_y"`              // Canvas Y position
	BlockDefID  *string         `json:"block_definition_id"`     // Optional block definition ID reference (deprecated, use BlockSlug)
	BlockSlug   string          `json:"block_slug,omitempty"`    // Block slug reference (resolved to ID at migration time)
}

// SystemEdgeDefinition represents an edge (connection) between steps
type SystemEdgeDefinition struct {
	SourceTempID string `json:"source_temp_id"` // Source step temp_id
	TargetTempID string `json:"target_temp_id"` // Target step temp_id
	SourcePort   string `json:"source_port"`    // Source port name (e.g., "output", "true", "false")
	TargetPort   string `json:"target_port"`    // Target port name (usually empty)
	Condition    string `json:"condition,omitempty"` // Optional condition expression
}

// Validate validates the workflow definition
func (w *SystemWorkflowDefinition) Validate() error {
	if w.SystemSlug == "" {
		return &ValidationError{Field: "system_slug", Message: "system_slug is required"}
	}
	if w.Name == "" {
		return &ValidationError{Field: "name", Message: "name is required"}
	}
	if len(w.Steps) == 0 {
		return &ValidationError{Field: "steps", Message: "at least one step is required"}
	}

	// Check for start step
	hasStart := false
	tempIDs := make(map[string]bool)
	for _, step := range w.Steps {
		if step.TempID == "" {
			return &ValidationError{Field: "steps.temp_id", Message: "temp_id is required for all steps"}
		}
		if tempIDs[step.TempID] {
			return &ValidationError{Field: "steps.temp_id", Message: "duplicate temp_id: " + step.TempID}
		}
		tempIDs[step.TempID] = true
		if step.Type == "start" {
			hasStart = true
		}
	}
	if !hasStart {
		return &ValidationError{Field: "steps", Message: "workflow must have a start step"}
	}

	// Validate edges reference valid steps
	for _, edge := range w.Edges {
		if !tempIDs[edge.SourceTempID] {
			return &ValidationError{Field: "edges.source_temp_id", Message: "invalid source_temp_id: " + edge.SourceTempID}
		}
		if !tempIDs[edge.TargetTempID] {
			return &ValidationError{Field: "edges.target_temp_id", Message: "invalid target_temp_id: " + edge.TargetTempID}
		}
	}

	return nil
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
