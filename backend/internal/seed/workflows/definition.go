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
	Steps       []SystemStepDefinition       `json:"steps"`
	Edges       []SystemEdgeDefinition       `json:"edges"`
	BlockGroups []SystemBlockGroupDefinition `json:"block_groups,omitempty"` // Control flow constructs

	// Metadata
	IsSystem bool `json:"is_system"` // Always true for system workflows
}

// SystemStepDefinition represents a step within a system workflow
type SystemStepDefinition struct {
	TempID           string          `json:"temp_id"`                       // Temporary ID for edge references (e.g., "step_1")
	Name             string          `json:"name"`                          // Step name
	Type             string          `json:"type"`                          // Block type (e.g., "start", "llm", "function")
	Config           json.RawMessage `json:"config,omitempty"`              // Step configuration
	TriggerType      string          `json:"trigger_type,omitempty"`        // For Start blocks: manual, webhook, schedule, internal
	TriggerConfig    json.RawMessage `json:"trigger_config,omitempty"`      // For Start blocks: trigger-specific config (includes entry_point)
	PositionX        int             `json:"position_x"`                    // Canvas X position
	PositionY        int             `json:"position_y"`                    // Canvas Y position
	BlockDefID       *string         `json:"block_definition_id"`           // Optional block definition ID reference (deprecated, use BlockSlug)
	BlockSlug        string          `json:"block_slug,omitempty"`          // Block slug reference (resolved to ID at migration time)
	BlockGroupTempID string          `json:"block_group_temp_id,omitempty"` // Parent block group temp_id (for steps inside a group)
	// Agent Group tool definition (for entry point steps within Agent groups)
	ToolName        string          `json:"tool_name,omitempty"`        // Tool name exposed to the agent
	ToolDescription string          `json:"tool_description,omitempty"` // Description of what the tool does
	ToolInputSchema json.RawMessage `json:"tool_input_schema,omitempty"` // JSON Schema for tool parameters
}

// SystemEdgeDefinition represents an edge (connection) between steps or block groups
// Either SourceTempID or SourceGroupTempID must be provided
// Either TargetTempID or TargetGroupTempID must be provided
type SystemEdgeDefinition struct {
	SourceTempID      string `json:"source_temp_id,omitempty"`       // Source step temp_id
	TargetTempID      string `json:"target_temp_id,omitempty"`       // Target step temp_id
	SourceGroupTempID string `json:"source_group_temp_id,omitempty"` // Source block group temp_id
	TargetGroupTempID string `json:"target_group_temp_id,omitempty"` // Target block group temp_id
	SourcePort        string `json:"source_port"`                    // Source port name (e.g., "output", "true", "false", "out")
	TargetPort        string `json:"target_port"`                    // Target port name (e.g., "input", "in")
	Condition         string `json:"condition,omitempty"`            // Optional condition expression
}

// SystemBlockGroupDefinition represents a block group (control flow construct) within a workflow
type SystemBlockGroupDefinition struct {
	TempID        string          `json:"temp_id"`                   // Temporary ID for references (e.g., "group_1")
	Name          string          `json:"name"`                      // Display name
	Type          string          `json:"type"`                      // parallel, try_catch, foreach, while
	Config        json.RawMessage `json:"config,omitempty"`          // Group-specific configuration
	ParentTempID  string          `json:"parent_temp_id,omitempty"`  // Parent group temp_id (for nesting)
	PositionX     int             `json:"position_x"`                // Canvas X position
	PositionY     int             `json:"position_y"`                // Canvas Y position
	Width         int             `json:"width,omitempty"`           // Width (default: 400)
	Height        int             `json:"height,omitempty"`          // Height (default: 300)
	PreProcess    string          `json:"pre_process,omitempty"`     // JS: external IN -> internal IN
	PostProcess   string          `json:"post_process,omitempty"`    // JS: internal OUT -> external OUT
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
	stepTempIDs := make(map[string]bool)
	for _, step := range w.Steps {
		if step.TempID == "" {
			return &ValidationError{Field: "steps.temp_id", Message: "temp_id is required for all steps"}
		}
		if stepTempIDs[step.TempID] {
			return &ValidationError{Field: "steps.temp_id", Message: "duplicate temp_id: " + step.TempID}
		}
		stepTempIDs[step.TempID] = true
		if step.Type == "start" {
			hasStart = true
		}
	}
	if !hasStart {
		return &ValidationError{Field: "steps", Message: "workflow must have a start step"}
	}

	// Collect block group temp IDs
	groupTempIDs := make(map[string]bool)
	for _, group := range w.BlockGroups {
		if group.TempID == "" {
			return &ValidationError{Field: "block_groups.temp_id", Message: "temp_id is required for all block groups"}
		}
		if groupTempIDs[group.TempID] {
			return &ValidationError{Field: "block_groups.temp_id", Message: "duplicate temp_id: " + group.TempID}
		}
		groupTempIDs[group.TempID] = true
	}

	// Validate edges reference valid steps or groups
	for _, edge := range w.Edges {
		// Check source - must have either step or group reference
		hasSource := false
		if edge.SourceTempID != "" {
			if !stepTempIDs[edge.SourceTempID] {
				return &ValidationError{Field: "edges.source_temp_id", Message: "invalid source_temp_id: " + edge.SourceTempID}
			}
			hasSource = true
		}
		if edge.SourceGroupTempID != "" {
			if !groupTempIDs[edge.SourceGroupTempID] {
				return &ValidationError{Field: "edges.source_group_temp_id", Message: "invalid source_group_temp_id: " + edge.SourceGroupTempID}
			}
			hasSource = true
		}
		if !hasSource {
			return &ValidationError{Field: "edges", Message: "edge must have source_temp_id or source_group_temp_id"}
		}

		// Check target - must have either step or group reference
		hasTarget := false
		if edge.TargetTempID != "" {
			if !stepTempIDs[edge.TargetTempID] {
				return &ValidationError{Field: "edges.target_temp_id", Message: "invalid target_temp_id: " + edge.TargetTempID}
			}
			hasTarget = true
		}
		if edge.TargetGroupTempID != "" {
			if !groupTempIDs[edge.TargetGroupTempID] {
				return &ValidationError{Field: "edges.target_group_temp_id", Message: "invalid target_group_temp_id: " + edge.TargetGroupTempID}
			}
			hasTarget = true
		}
		if !hasTarget {
			return &ValidationError{Field: "edges", Message: "edge must have target_temp_id or target_group_temp_id"}
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
