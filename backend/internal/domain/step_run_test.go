package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewStepRun(t *testing.T) {
	tenantID := uuid.New()
	runID := uuid.New()
	stepID := uuid.New()
	stepName := "Test Step"
	sequenceNumber := 1

	stepRun := NewStepRun(tenantID, runID, stepID, stepName, sequenceNumber)

	if stepRun.ID == uuid.Nil {
		t.Error("NewStepRun() should generate a non-nil UUID")
	}
	if stepRun.TenantID != tenantID {
		t.Errorf("NewStepRun() TenantID = %v, want %v", stepRun.TenantID, tenantID)
	}
	if stepRun.RunID != runID {
		t.Errorf("NewStepRun() RunID = %v, want %v", stepRun.RunID, runID)
	}
	if stepRun.StepID != stepID {
		t.Errorf("NewStepRun() StepID = %v, want %v", stepRun.StepID, stepID)
	}
	if stepRun.StepName != stepName {
		t.Errorf("NewStepRun() StepName = %v, want %v", stepRun.StepName, stepName)
	}
	if stepRun.Status != StepRunStatusPending {
		t.Errorf("NewStepRun() Status = %v, want %v", stepRun.Status, StepRunStatusPending)
	}
	if stepRun.Attempt != 1 {
		t.Errorf("NewStepRun() Attempt = %v, want 1", stepRun.Attempt)
	}
	if stepRun.SequenceNumber != sequenceNumber {
		t.Errorf("NewStepRun() SequenceNumber = %v, want %v", stepRun.SequenceNumber, sequenceNumber)
	}
}

func TestNewStepRunWithAttempt(t *testing.T) {
	tenantID := uuid.New()
	runID := uuid.New()
	stepID := uuid.New()
	stepName := "Retry Step"
	attempt := 3
	sequenceNumber := 2

	stepRun := NewStepRunWithAttempt(tenantID, runID, stepID, stepName, attempt, sequenceNumber)

	if stepRun.Attempt != attempt {
		t.Errorf("NewStepRunWithAttempt() Attempt = %v, want %v", stepRun.Attempt, attempt)
	}
	if stepRun.SequenceNumber != sequenceNumber {
		t.Errorf("NewStepRunWithAttempt() SequenceNumber = %v, want %v", stepRun.SequenceNumber, sequenceNumber)
	}
}

func TestStepRunStatus_Constants(t *testing.T) {
	statuses := []StepRunStatus{
		StepRunStatusPending,
		StepRunStatusRunning,
		StepRunStatusCompleted,
		StepRunStatusFailed,
		StepRunStatusSkipped,
	}

	expected := []string{"pending", "running", "completed", "failed", "skipped"}
	for i, s := range statuses {
		if string(s) != expected[i] {
			t.Errorf("StepRunStatus %v = %v, want %v", i, s, expected[i])
		}
	}
}

func TestStepRun_Start(t *testing.T) {
	stepRun := NewStepRun(uuid.New(), uuid.New(), uuid.New(), "Test", 1)
	input := json.RawMessage(`{"key": "value"}`)

	stepRun.Start(input)

	if stepRun.Status != StepRunStatusRunning {
		t.Errorf("Start() Status = %v, want %v", stepRun.Status, StepRunStatusRunning)
	}
	if stepRun.StartedAt == nil {
		t.Error("Start() StartedAt should not be nil")
	}
	if string(stepRun.Input) != string(input) {
		t.Error("Start() Input mismatch")
	}
}

func TestStepRun_Complete(t *testing.T) {
	stepRun := NewStepRun(uuid.New(), uuid.New(), uuid.New(), "Test", 1)
	stepRun.Start(nil)
	output := json.RawMessage(`{"result": "done"}`)

	stepRun.Complete(output)

	if stepRun.Status != StepRunStatusCompleted {
		t.Errorf("Complete() Status = %v, want %v", stepRun.Status, StepRunStatusCompleted)
	}
	if stepRun.CompletedAt == nil {
		t.Error("Complete() CompletedAt should not be nil")
	}
	if stepRun.DurationMs == nil {
		t.Error("Complete() DurationMs should be set")
	}
	if string(stepRun.Output) != string(output) {
		t.Error("Complete() Output mismatch")
	}
}

func TestStepRun_Fail(t *testing.T) {
	stepRun := NewStepRun(uuid.New(), uuid.New(), uuid.New(), "Test", 1)
	stepRun.Start(nil)
	errorMsg := "step failed"

	stepRun.Fail(errorMsg)

	if stepRun.Status != StepRunStatusFailed {
		t.Errorf("Fail() Status = %v, want %v", stepRun.Status, StepRunStatusFailed)
	}
	if stepRun.Error != errorMsg {
		t.Errorf("Fail() Error = %v, want %v", stepRun.Error, errorMsg)
	}
	if stepRun.CompletedAt == nil {
		t.Error("Fail() CompletedAt should not be nil")
	}
	if stepRun.DurationMs == nil {
		t.Error("Fail() DurationMs should be set")
	}
}

func TestStepRun_Skip(t *testing.T) {
	stepRun := NewStepRun(uuid.New(), uuid.New(), uuid.New(), "Test", 1)

	stepRun.Skip()

	if stepRun.Status != StepRunStatusSkipped {
		t.Errorf("Skip() Status = %v, want %v", stepRun.Status, StepRunStatusSkipped)
	}
}

func TestStepRun_Retry(t *testing.T) {
	stepRun := NewStepRun(uuid.New(), uuid.New(), uuid.New(), "Test", 1)
	stepRun.Start(json.RawMessage(`{}`))
	stepRun.Fail("initial error")
	originalAttempt := stepRun.Attempt

	stepRun.Retry()

	if stepRun.Attempt != originalAttempt+1 {
		t.Errorf("Retry() Attempt = %v, want %v", stepRun.Attempt, originalAttempt+1)
	}
	if stepRun.Status != StepRunStatusPending {
		t.Errorf("Retry() Status = %v, want %v", stepRun.Status, StepRunStatusPending)
	}
	if stepRun.Error != "" {
		t.Error("Retry() should clear Error")
	}
	if stepRun.StartedAt != nil {
		t.Error("Retry() should clear StartedAt")
	}
	if stepRun.CompletedAt != nil {
		t.Error("Retry() should clear CompletedAt")
	}
	if stepRun.DurationMs != nil {
		t.Error("Retry() should clear DurationMs")
	}
}

func TestStepRun_PinnedInput(t *testing.T) {
	stepRun := NewStepRun(uuid.New(), uuid.New(), uuid.New(), "Test", 1)
	regularInput := json.RawMessage(`{"regular": true}`)
	pinnedInput := json.RawMessage(`{"pinned": true}`)

	stepRun.Start(regularInput)

	// Without pinned input
	if stepRun.HasPinnedInput() {
		t.Error("HasPinnedInput() should return false initially")
	}
	if string(stepRun.GetEffectiveInput()) != string(regularInput) {
		t.Error("GetEffectiveInput() should return regular input when not pinned")
	}

	// With pinned input
	stepRun.SetPinnedInput(pinnedInput)
	if !stepRun.HasPinnedInput() {
		t.Error("HasPinnedInput() should return true after setting pinned input")
	}
	if string(stepRun.GetEffectiveInput()) != string(pinnedInput) {
		t.Error("GetEffectiveInput() should return pinned input when set")
	}
}

func TestStepRun_StreamingOutput(t *testing.T) {
	stepRun := NewStepRun(uuid.New(), uuid.New(), uuid.New(), "Test", 1)

	// Initial state
	chunks, err := stepRun.GetStreamingChunks()
	if err != nil {
		t.Fatalf("GetStreamingChunks() error = %v", err)
	}
	if chunks != nil {
		t.Error("GetStreamingChunks() should return nil initially")
	}

	// Append chunks
	err = stepRun.AppendStreamingChunk("Hello", "text")
	if err != nil {
		t.Fatalf("AppendStreamingChunk() error = %v", err)
	}
	err = stepRun.AppendStreamingChunk(" World", "text")
	if err != nil {
		t.Fatalf("AppendStreamingChunk() error = %v", err)
	}

	chunks, err = stepRun.GetStreamingChunks()
	if err != nil {
		t.Fatalf("GetStreamingChunks() error = %v", err)
	}
	if len(chunks) != 2 {
		t.Errorf("GetStreamingChunks() returned %d chunks, want 2", len(chunks))
	}
	if chunks[0].Chunk != "Hello" {
		t.Errorf("First chunk = %v, want Hello", chunks[0].Chunk)
	}
	if chunks[1].Chunk != " World" {
		t.Errorf("Second chunk = %v, want ' World'", chunks[1].Chunk)
	}
}

func TestStepRun_DurationCalculation(t *testing.T) {
	stepRun := NewStepRun(uuid.New(), uuid.New(), uuid.New(), "Test", 1)

	// Start and wait a bit
	startTime := time.Now().Add(-100 * time.Millisecond)
	stepRun.StartedAt = &startTime
	stepRun.Status = StepRunStatusRunning

	// Complete
	stepRun.Complete(nil)

	if stepRun.DurationMs == nil {
		t.Error("DurationMs should be set after completion")
	}
	if *stepRun.DurationMs < 100 {
		t.Errorf("DurationMs = %v, expected at least 100ms", *stepRun.DurationMs)
	}
}
