package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/souta/ai-orchestration/internal/domain"
)

// ErrorWorkflowConfig represents the configuration for error workflow triggering
type ErrorWorkflowConfig struct {
	TriggerOn    []string               `json:"trigger_on"`    // ["failed", "cancelled", "timeout"]
	InputMapping map[string]string      `json:"input_mapping"` // Custom input mapping
	Enabled      bool                   `json:"enabled"`
}

// DefaultErrorWorkflowConfig returns the default configuration
func DefaultErrorWorkflowConfig() ErrorWorkflowConfig {
	return ErrorWorkflowConfig{
		TriggerOn: []string{"failed"},
		Enabled:   true,
	}
}

// ParseErrorWorkflowConfig parses the error workflow config from JSON
func ParseErrorWorkflowConfig(data json.RawMessage) (*ErrorWorkflowConfig, error) {
	if len(data) == 0 || string(data) == "{}" || string(data) == "null" {
		config := DefaultErrorWorkflowConfig()
		return &config, nil
	}

	var config ErrorWorkflowConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Set defaults if not specified
	if len(config.TriggerOn) == 0 {
		config.TriggerOn = []string{"failed"}
	}

	return &config, nil
}

// ShouldTriggerErrorWorkflow determines if an error workflow should be triggered
func ShouldTriggerErrorWorkflow(project *domain.Project, run *domain.Run) bool {
	// No error workflow configured
	if project.ErrorWorkflowID == nil {
		return false
	}

	// Don't trigger error workflow for error workflow runs (prevent infinite loops)
	if run.IsErrorWorkflowRun() {
		return false
	}

	config, err := ParseErrorWorkflowConfig(project.ErrorWorkflowConfig)
	if err != nil || !config.Enabled {
		return false
	}

	// Check if current run status matches trigger conditions
	runStatus := string(run.Status)
	for _, trigger := range config.TriggerOn {
		if trigger == runStatus {
			return true
		}
	}

	return false
}

// ErrorWorkflowInput represents the input passed to an error workflow
type ErrorWorkflowInput struct {
	OriginalRunID     uuid.UUID              `json:"original_run_id"`
	OriginalProjectID uuid.UUID              `json:"original_project_id"`
	OriginalProject   string                 `json:"original_project"`
	ErrorStepID       *uuid.UUID             `json:"error_step_id,omitempty"`
	ErrorStepName     string                 `json:"error_step_name,omitempty"`
	ErrorMessage      string                 `json:"error_message"`
	OriginalInput     json.RawMessage        `json:"original_input,omitempty"`
	OriginalOutput    json.RawMessage        `json:"original_output,omitempty"`
	TriggeredAt       time.Time              `json:"triggered_at"`
	RunStatus         string                 `json:"run_status"`
	Custom            map[string]interface{} `json:"custom,omitempty"`
}

// BuildErrorWorkflowInput builds the input for an error workflow
func BuildErrorWorkflowInput(
	project *domain.Project,
	run *domain.Run,
	failedStepRun *domain.StepRun,
	config *ErrorWorkflowConfig,
) (*ErrorWorkflowInput, error) {
	input := &ErrorWorkflowInput{
		OriginalRunID:     run.ID,
		OriginalProjectID: run.ProjectID,
		OriginalProject:   project.Name,
		OriginalInput:     run.Input,
		OriginalOutput:    run.Output,
		TriggeredAt:       time.Now().UTC(),
		RunStatus:         string(run.Status),
	}

	// Add error information
	if run.Error != nil {
		input.ErrorMessage = *run.Error
	}

	// Add failed step information if available
	if failedStepRun != nil {
		input.ErrorStepID = &failedStepRun.StepID
		input.ErrorStepName = failedStepRun.StepName
		if failedStepRun.Error != "" {
			input.ErrorMessage = failedStepRun.Error
		}
	}

	// Apply custom input mapping if configured
	if config != nil && len(config.InputMapping) > 0 {
		input.Custom = make(map[string]interface{})
		// Custom mapping would be applied here based on config
		// This is a placeholder for future advanced mapping logic
	}

	return input, nil
}

// CreateErrorWorkflowRun creates a new run for the error workflow
func CreateErrorWorkflowRun(
	ctx context.Context,
	errorProject *domain.Project,
	parentRun *domain.Run,
	input *ErrorWorkflowInput,
) (*domain.Run, error) {
	// Serialize input
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal error workflow input: %w", err)
	}

	// Create new run
	errorRun := domain.NewRun(
		parentRun.TenantID,
		errorProject.ID,
		errorProject.Version,
		inputJSON,
		domain.TriggerTypeInternal,
	)

	// Set error trigger source
	triggerInfo := domain.ErrorTriggerInfo{
		OriginalRunID:   parentRun.ID,
		OriginalProject: input.OriginalProject,
		ErrorMessage:    input.ErrorMessage,
		TriggeredAt:     input.TriggeredAt,
	}
	if input.ErrorStepID != nil {
		triggerInfo.ErrorStepID = *input.ErrorStepID
		triggerInfo.ErrorStepName = input.ErrorStepName
	}

	if err := errorRun.SetErrorTrigger(parentRun.ID, triggerInfo); err != nil {
		return nil, fmt.Errorf("failed to set error trigger: %w", err)
	}

	// Set internal trigger metadata
	if err := errorRun.SetInternalTrigger("error-workflow", map[string]interface{}{
		"parent_run_id":  parentRun.ID.String(),
		"error_step_id":  input.ErrorStepID,
		"error_step_name": input.ErrorStepName,
	}); err != nil {
		return nil, fmt.Errorf("failed to set internal trigger: %w", err)
	}

	return errorRun, nil
}
