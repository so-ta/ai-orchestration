package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewRun(t *testing.T) {
	tenantID := uuid.New()
	projectID := uuid.New()
	version := 2
	input := json.RawMessage(`{"key": "value"}`)

	run := NewRun(tenantID, projectID, version, input, TriggerTypeManual)

	if run.ID == uuid.Nil {
		t.Error("NewRun() should generate a non-nil UUID")
	}
	if run.TenantID != tenantID {
		t.Errorf("NewRun() TenantID = %v, want %v", run.TenantID, tenantID)
	}
	if run.ProjectID != projectID {
		t.Errorf("NewRun() ProjectID = %v, want %v", run.ProjectID, projectID)
	}
	if run.ProjectVersion != version {
		t.Errorf("NewRun() ProjectVersion = %v, want %v", run.ProjectVersion, version)
	}
	if run.Status != RunStatusPending {
		t.Errorf("NewRun() Status = %v, want %v", run.Status, RunStatusPending)
	}
	if run.TriggeredBy != TriggerTypeManual {
		t.Errorf("NewRun() TriggeredBy = %v, want %v", run.TriggeredBy, TriggerTypeManual)
	}
	if run.CreatedAt.IsZero() {
		t.Error("NewRun() CreatedAt should not be zero")
	}
}

func TestNewRunWithStartStep(t *testing.T) {
	tenantID := uuid.New()
	projectID := uuid.New()
	startStepID := uuid.New()
	version := 1
	input := json.RawMessage(`{}`)

	run := NewRunWithStartStep(tenantID, projectID, version, startStepID, input, TriggerTypeWebhook)

	if run.StartStepID == nil || *run.StartStepID != startStepID {
		t.Error("NewRunWithStartStep() StartStepID mismatch")
	}
	if run.TriggeredBy != TriggerTypeWebhook {
		t.Errorf("NewRunWithStartStep() TriggeredBy = %v, want %v", run.TriggeredBy, TriggerTypeWebhook)
	}
}

func TestRunStatus_Constants(t *testing.T) {
	// Verify all status constants are defined
	statuses := []RunStatus{
		RunStatusPending,
		RunStatusRunning,
		RunStatusCompleted,
		RunStatusFailed,
		RunStatusCancelled,
	}

	expected := []string{"pending", "running", "completed", "failed", "cancelled"}
	for i, s := range statuses {
		if string(s) != expected[i] {
			t.Errorf("RunStatus %v = %v, want %v", i, s, expected[i])
		}
	}
}

func TestRun_Start(t *testing.T) {
	run := NewRun(uuid.New(), uuid.New(), 1, nil, TriggerTypeManual)

	run.Start()

	if run.Status != RunStatusRunning {
		t.Errorf("Start() Status = %v, want %v", run.Status, RunStatusRunning)
	}
	if run.StartedAt == nil {
		t.Error("Start() StartedAt should not be nil")
	}
}

func TestRun_Complete(t *testing.T) {
	run := NewRun(uuid.New(), uuid.New(), 1, nil, TriggerTypeManual)
	run.Start()
	output := json.RawMessage(`{"result": "success"}`)

	run.Complete(output)

	if run.Status != RunStatusCompleted {
		t.Errorf("Complete() Status = %v, want %v", run.Status, RunStatusCompleted)
	}
	if run.CompletedAt == nil {
		t.Error("Complete() CompletedAt should not be nil")
	}
	if string(run.Output) != string(output) {
		t.Errorf("Complete() Output mismatch")
	}
}

func TestRun_Fail(t *testing.T) {
	run := NewRun(uuid.New(), uuid.New(), 1, nil, TriggerTypeManual)
	run.Start()
	errorMsg := "something went wrong"

	run.Fail(errorMsg)

	if run.Status != RunStatusFailed {
		t.Errorf("Fail() Status = %v, want %v", run.Status, RunStatusFailed)
	}
	if run.Error == nil || *run.Error != errorMsg {
		t.Errorf("Fail() Error mismatch")
	}
	if run.CompletedAt == nil {
		t.Error("Fail() CompletedAt should not be nil")
	}
}

func TestRun_Cancel(t *testing.T) {
	run := NewRun(uuid.New(), uuid.New(), 1, nil, TriggerTypeManual)
	run.Start()

	run.Cancel()

	if run.Status != RunStatusCancelled {
		t.Errorf("Cancel() Status = %v, want %v", run.Status, RunStatusCancelled)
	}
	if run.CompletedAt == nil {
		t.Error("Cancel() CompletedAt should not be nil")
	}
}

func TestRun_DurationMs(t *testing.T) {
	run := NewRun(uuid.New(), uuid.New(), 1, nil, TriggerTypeManual)

	// Not started yet
	if run.DurationMs() != nil {
		t.Error("DurationMs() should return nil when not started")
	}

	// Started but not completed
	startTime := time.Now().Add(-time.Second)
	run.StartedAt = &startTime
	if run.DurationMs() != nil {
		t.Error("DurationMs() should return nil when not completed")
	}

	// Completed
	completedTime := startTime.Add(500 * time.Millisecond)
	run.CompletedAt = &completedTime
	duration := run.DurationMs()
	if duration == nil {
		t.Error("DurationMs() should not return nil when completed")
	} else if *duration < 500 || *duration > 600 {
		t.Errorf("DurationMs() = %v, expected around 500ms", *duration)
	}
}

func TestTriggerType_Constants(t *testing.T) {
	types := []TriggerType{
		TriggerTypeManual,
		TriggerTypeSchedule,
		TriggerTypeWebhook,
		TriggerTypeTest,
		TriggerTypeInternal,
	}

	expected := []string{"manual", "schedule", "webhook", "test", "internal"}
	for i, tt := range types {
		if string(tt) != expected[i] {
			t.Errorf("TriggerType %v = %v, want %v", i, tt, expected[i])
		}
	}
}

func TestRun_SetInternalTrigger(t *testing.T) {
	run := NewRun(uuid.New(), uuid.New(), 1, nil, TriggerTypeInternal)
	source := "copilot"
	metadata := map[string]interface{}{
		"feature": "generate",
	}

	err := run.SetInternalTrigger(source, metadata)
	if err != nil {
		t.Fatalf("SetInternalTrigger() error = %v", err)
	}

	if run.TriggerSource == nil || *run.TriggerSource != source {
		t.Error("SetInternalTrigger() TriggerSource mismatch")
	}
	if run.TriggerMetadata == nil {
		t.Error("SetInternalTrigger() TriggerMetadata should not be nil")
	}
}

func TestRun_IsErrorWorkflowRun(t *testing.T) {
	run := NewRun(uuid.New(), uuid.New(), 1, nil, TriggerTypeManual)

	if run.IsErrorWorkflowRun() {
		t.Error("IsErrorWorkflowRun() should return false for normal run")
	}

	parentRunID := uuid.New()
	run.ParentRunID = &parentRunID

	if !run.IsErrorWorkflowRun() {
		t.Error("IsErrorWorkflowRun() should return true for error workflow run")
	}
}

func TestRun_SetErrorTrigger(t *testing.T) {
	run := NewRun(uuid.New(), uuid.New(), 1, nil, TriggerTypeInternal)
	parentRunID := uuid.New()
	info := ErrorTriggerInfo{
		OriginalRunID:   uuid.New(),
		OriginalProject: "Test Project",
		ErrorStepID:     uuid.New(),
		ErrorStepName:   "LLM Step",
		ErrorMessage:    "API Error",
		TriggeredAt:     time.Now(),
	}

	err := run.SetErrorTrigger(parentRunID, info)
	if err != nil {
		t.Fatalf("SetErrorTrigger() error = %v", err)
	}

	if run.ParentRunID == nil || *run.ParentRunID != parentRunID {
		t.Error("SetErrorTrigger() ParentRunID mismatch")
	}

	gotInfo, err := run.GetErrorTriggerInfo()
	if err != nil {
		t.Fatalf("GetErrorTriggerInfo() error = %v", err)
	}
	if gotInfo.OriginalProject != info.OriginalProject {
		t.Errorf("GetErrorTriggerInfo() OriginalProject = %v, want %v", gotInfo.OriginalProject, info.OriginalProject)
	}
}
