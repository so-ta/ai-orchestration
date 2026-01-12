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

// RunMode represents the execution mode
type RunMode string

const (
	RunModeTest       RunMode = "test"
	RunModeProduction RunMode = "production"
)

// TriggerType represents how the run was triggered
type TriggerType string

const (
	TriggerTypeManual   TriggerType = "manual"
	TriggerTypeSchedule TriggerType = "schedule"
	TriggerTypeWebhook  TriggerType = "webhook"
	TriggerTypeInternal TriggerType = "internal" // Internal system calls (e.g., Copilot)
)

// Run represents a workflow execution
type Run struct {
	ID              uuid.UUID       `json:"id"`
	TenantID        uuid.UUID       `json:"tenant_id"`
	WorkflowID      uuid.UUID       `json:"workflow_id"`
	WorkflowVersion int             `json:"workflow_version"`
	Status          RunStatus       `json:"status"`
	Mode            RunMode         `json:"mode"`
	Input           json.RawMessage `json:"input,omitempty"`
	Output          json.RawMessage `json:"output,omitempty"`
	Error           *string         `json:"error,omitempty"`
	TriggeredBy     TriggerType     `json:"triggered_by"`
	TriggeredByUser *uuid.UUID      `json:"triggered_by_user,omitempty"`
	StartedAt       *time.Time      `json:"started_at,omitempty"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`

	// Internal trigger metadata (for TriggerTypeInternal)
	TriggerSource   *string         `json:"trigger_source,omitempty"`   // e.g., "copilot", "audit-system"
	TriggerMetadata json.RawMessage `json:"trigger_metadata,omitempty"` // e.g., {"feature": "generate", "user_id": "..."}

	// Loaded relations
	StepRuns []StepRun `json:"step_runs,omitempty"`
}

// NewRun creates a new run
func NewRun(tenantID, workflowID uuid.UUID, workflowVersion int, input json.RawMessage, mode RunMode, triggerType TriggerType) *Run {
	return &Run{
		ID:              uuid.New(),
		TenantID:        tenantID,
		WorkflowID:      workflowID,
		WorkflowVersion: workflowVersion,
		Status:          RunStatusPending,
		Mode:            mode,
		Input:           input,
		TriggeredBy:     triggerType,
		CreatedAt:       time.Now().UTC(),
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
