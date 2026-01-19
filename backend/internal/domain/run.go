package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// RunStatus represents the status of a workflow run
type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"
	RunStatusRunning   RunStatus = "running"
	RunStatusCompleted RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
	RunStatusCancelled RunStatus = "cancelled"
)

// TriggerType represents how the run was triggered
type TriggerType string

const (
	TriggerTypeManual   TriggerType = "manual"
	TriggerTypeSchedule TriggerType = "schedule"
	TriggerTypeWebhook  TriggerType = "webhook"
	TriggerTypeTest     TriggerType = "test"     // Test execution from workflow editor
	TriggerTypeInternal TriggerType = "internal" // Internal system calls (e.g., Copilot)
)

// Run represents a project execution
type Run struct {
	ID             uuid.UUID       `json:"id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	ProjectID      uuid.UUID       `json:"project_id"`
	ProjectVersion int             `json:"project_version"`
	StartStepID    *uuid.UUID      `json:"start_step_id,omitempty"` // Which Start block triggered this run
	Status         RunStatus       `json:"status"`
	Input          json.RawMessage `json:"input,omitempty"`
	Output         json.RawMessage `json:"output,omitempty"`
	Error          *string         `json:"error,omitempty"`
	TriggeredBy    TriggerType     `json:"triggered_by"`
	RunNumber      int             `json:"run_number"` // Sequential number per project + triggered_by
	TriggeredByUser *uuid.UUID     `json:"triggered_by_user,omitempty"`
	StartedAt       *time.Time     `json:"started_at,omitempty"`
	CompletedAt     *time.Time     `json:"completed_at,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`

	// Internal trigger metadata (for TriggerTypeInternal)
	TriggerSource   *string         `json:"trigger_source,omitempty"`   // e.g., "copilot", "audit-system"
	TriggerMetadata json.RawMessage `json:"trigger_metadata,omitempty"` // e.g., {"feature": "generate", "user_id": "..."}

	// Error Workflow tracking
	ParentRunID        *uuid.UUID      `json:"parent_run_id,omitempty"`        // Parent run that triggered this error workflow
	ErrorTriggerSource json.RawMessage `json:"error_trigger_source,omitempty"` // Error info from parent run

	// Loaded relations
	StepRuns []StepRun `json:"step_runs,omitempty"`
}

// NewRun creates a new run
func NewRun(tenantID, projectID uuid.UUID, projectVersion int, input json.RawMessage, triggerType TriggerType) *Run {
	return &Run{
		ID:             uuid.New(),
		TenantID:       tenantID,
		ProjectID:      projectID,
		ProjectVersion: projectVersion,
		Status:         RunStatusPending,
		Input:          input,
		TriggeredBy:    triggerType,
		CreatedAt:      time.Now().UTC(),
		// RunNumber is set by DB trigger
	}
}

// NewRunWithStartStep creates a new run with a specific Start step
func NewRunWithStartStep(tenantID, projectID uuid.UUID, projectVersion int, startStepID uuid.UUID, input json.RawMessage, triggerType TriggerType) *Run {
	return &Run{
		ID:             uuid.New(),
		TenantID:       tenantID,
		ProjectID:      projectID,
		ProjectVersion: projectVersion,
		StartStepID:    &startStepID,
		Status:         RunStatusPending,
		Input:          input,
		TriggeredBy:    triggerType,
		CreatedAt:      time.Now().UTC(),
		// RunNumber is set by DB trigger
	}
}

// Start marks the run as started
func (r *Run) Start() {
	now := time.Now().UTC()
	r.Status = RunStatusRunning
	r.StartedAt = &now
}

// Complete marks the run as completed
func (r *Run) Complete(output json.RawMessage) {
	now := time.Now().UTC()
	r.Status = RunStatusCompleted
	r.Output = output
	r.CompletedAt = &now
}

// Fail marks the run as failed
func (r *Run) Fail(err string) {
	now := time.Now().UTC()
	r.Status = RunStatusFailed
	r.Error = &err
	r.CompletedAt = &now
}

// Cancel marks the run as cancelled
func (r *Run) Cancel() {
	now := time.Now().UTC()
	r.Status = RunStatusCancelled
	r.CompletedAt = &now
}

// DurationMs returns the duration in milliseconds
func (r *Run) DurationMs() *int64 {
	if r.StartedAt == nil || r.CompletedAt == nil {
		return nil
	}
	ms := r.CompletedAt.Sub(*r.StartedAt).Milliseconds()
	return &ms
}

// SetInternalTrigger sets the internal trigger metadata
func (r *Run) SetInternalTrigger(source string, metadata map[string]interface{}) error {
	r.TriggerSource = &source
	if metadata != nil {
		metaJSON, err := json.Marshal(metadata)
		if err != nil {
			return err
		}
		r.TriggerMetadata = metaJSON
	}
	return nil
}

// ErrorTriggerInfo contains information about the error that triggered an error workflow
type ErrorTriggerInfo struct {
	OriginalRunID   uuid.UUID `json:"original_run_id"`
	OriginalProject string    `json:"original_project"`
	ErrorStepID     uuid.UUID `json:"error_step_id"`
	ErrorStepName   string    `json:"error_step_name"`
	ErrorMessage    string    `json:"error_message"`
	TriggeredAt     time.Time `json:"triggered_at"`
}

// SetErrorTrigger sets this run as an error workflow run triggered by a parent run
func (r *Run) SetErrorTrigger(parentRunID uuid.UUID, info ErrorTriggerInfo) error {
	r.ParentRunID = &parentRunID
	infoJSON, err := json.Marshal(info)
	if err != nil {
		return err
	}
	r.ErrorTriggerSource = infoJSON
	return nil
}

// GetErrorTriggerInfo returns the error trigger info if this is an error workflow run
func (r *Run) GetErrorTriggerInfo() (*ErrorTriggerInfo, error) {
	if r.ErrorTriggerSource == nil || len(r.ErrorTriggerSource) == 0 {
		return nil, nil
	}
	var info ErrorTriggerInfo
	if err := json.Unmarshal(r.ErrorTriggerSource, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// IsErrorWorkflowRun returns true if this run was triggered by an error workflow
func (r *Run) IsErrorWorkflowRun() bool {
	return r.ParentRunID != nil
}
