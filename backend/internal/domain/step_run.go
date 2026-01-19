package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// StepRunStatus represents the status of a step run
type StepRunStatus string

const (
	StepRunStatusPending   StepRunStatus = "pending"
	StepRunStatusRunning   StepRunStatus = "running"
	StepRunStatusCompleted StepRunStatus = "completed"
	StepRunStatusFailed    StepRunStatus = "failed"
	StepRunStatusSkipped   StepRunStatus = "skipped"
)

// StepRun represents a single step execution within a run
type StepRun struct {
	ID             uuid.UUID       `json:"id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	RunID          uuid.UUID       `json:"run_id"`
	StepID         uuid.UUID       `json:"step_id"`
	StepName       string          `json:"step_name"`
	Status         StepRunStatus   `json:"status"`
	Attempt        int             `json:"attempt"`
	SequenceNumber int             `json:"sequence_number"`
	Input          json.RawMessage `json:"input,omitempty"`
	Output         json.RawMessage `json:"output,omitempty"`
	Error          string          `json:"error,omitempty"`
	StartedAt      *time.Time      `json:"started_at,omitempty"`
	CompletedAt    *time.Time      `json:"completed_at,omitempty"`
	DurationMs     *int            `json:"duration_ms,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`

	// Debug features
	PinnedInput     json.RawMessage `json:"pinned_input,omitempty"`     // Pinned input for debugging/replay
	StreamingOutput json.RawMessage `json:"streaming_output,omitempty"` // Streaming output chunks
}

// NewStepRun creates a new step run
func NewStepRun(tenantID, runID, stepID uuid.UUID, stepName string, sequenceNumber int) *StepRun {
	return &StepRun{
		ID:             uuid.New(),
		TenantID:       tenantID,
		RunID:          runID,
		StepID:         stepID,
		StepName:       stepName,
		Status:         StepRunStatusPending,
		Attempt:        1,
		SequenceNumber: sequenceNumber,
		CreatedAt:      time.Now().UTC(),
	}
}

// NewStepRunWithAttempt creates a new step run with a specific attempt number (for re-execution)
func NewStepRunWithAttempt(tenantID, runID, stepID uuid.UUID, stepName string, attempt, sequenceNumber int) *StepRun {
	return &StepRun{
		ID:             uuid.New(),
		TenantID:       tenantID,
		RunID:          runID,
		StepID:         stepID,
		StepName:       stepName,
		Status:         StepRunStatusPending,
		Attempt:        attempt,
		SequenceNumber: sequenceNumber,
		CreatedAt:      time.Now().UTC(),
	}
}

// Start marks the step run as started
func (sr *StepRun) Start(input json.RawMessage) {
	now := time.Now().UTC()
	sr.Status = StepRunStatusRunning
	sr.Input = input
	sr.StartedAt = &now
}

// Complete marks the step run as completed
func (sr *StepRun) Complete(output json.RawMessage) {
	now := time.Now().UTC()
	sr.Status = StepRunStatusCompleted
	sr.Output = output
	sr.CompletedAt = &now
	if sr.StartedAt != nil {
		ms := int(now.Sub(*sr.StartedAt).Milliseconds())
		sr.DurationMs = &ms
	}
}

// Fail marks the step run as failed
func (sr *StepRun) Fail(err string) {
	now := time.Now().UTC()
	sr.Status = StepRunStatusFailed
	sr.Error = err
	sr.CompletedAt = &now
	if sr.StartedAt != nil {
		ms := int(now.Sub(*sr.StartedAt).Milliseconds())
		sr.DurationMs = &ms
	}
}

// Skip marks the step run as skipped
func (sr *StepRun) Skip() {
	sr.Status = StepRunStatusSkipped
}

// Retry increments the attempt counter and resets status
func (sr *StepRun) Retry() {
	sr.Attempt++
	sr.Status = StepRunStatusPending
	sr.Error = ""
	sr.StartedAt = nil
	sr.CompletedAt = nil
	sr.DurationMs = nil
}

// SetPinnedInput sets the pinned input for debugging
func (sr *StepRun) SetPinnedInput(input json.RawMessage) {
	sr.PinnedInput = input
}

// GetEffectiveInput returns the pinned input if set, otherwise the regular input
func (sr *StepRun) GetEffectiveInput() json.RawMessage {
	if len(sr.PinnedInput) > 0 && string(sr.PinnedInput) != "null" {
		return sr.PinnedInput
	}
	return sr.Input
}

// HasPinnedInput returns true if the step run has a pinned input
func (sr *StepRun) HasPinnedInput() bool {
	return len(sr.PinnedInput) > 0 && string(sr.PinnedInput) != "null"
}

// StreamingChunk represents a single chunk of streaming output
type StreamingChunk struct {
	Chunk     string    `json:"chunk"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // "text", "json", "error"
}

// AppendStreamingChunk appends a chunk to the streaming output
func (sr *StepRun) AppendStreamingChunk(chunk string, chunkType string) error {
	newChunk := StreamingChunk{
		Chunk:     chunk,
		Timestamp: time.Now().UTC(),
		Type:      chunkType,
	}

	var chunks []StreamingChunk
	if len(sr.StreamingOutput) > 0 {
		if err := json.Unmarshal(sr.StreamingOutput, &chunks); err != nil {
			return err
		}
	}

	chunks = append(chunks, newChunk)
	data, err := json.Marshal(chunks)
	if err != nil {
		return err
	}

	sr.StreamingOutput = data
	return nil
}

// GetStreamingChunks returns the streaming output chunks
func (sr *StepRun) GetStreamingChunks() ([]StreamingChunk, error) {
	if len(sr.StreamingOutput) == 0 {
		return nil, nil
	}

	var chunks []StreamingChunk
	if err := json.Unmarshal(sr.StreamingOutput, &chunks); err != nil {
		return nil, err
	}

	return chunks, nil
}
