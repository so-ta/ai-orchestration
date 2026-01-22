package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewSchedule(t *testing.T) {
	tenantID := uuid.New()
	projectID := uuid.New()
	startStepID := uuid.New()
	projectVersion := 5
	name := "Daily Report"
	cron := "0 9 * * *"
	timezone := "Asia/Tokyo"
	input := json.RawMessage(`{"key": "value"}`)

	schedule := NewSchedule(tenantID, projectID, startStepID, projectVersion, name, cron, timezone, input)

	if schedule.ID == uuid.Nil {
		t.Error("NewSchedule() should generate a non-nil UUID")
	}
	if schedule.TenantID != tenantID {
		t.Errorf("NewSchedule() TenantID = %v, want %v", schedule.TenantID, tenantID)
	}
	if schedule.ProjectID != projectID {
		t.Errorf("NewSchedule() ProjectID = %v, want %v", schedule.ProjectID, projectID)
	}
	if schedule.StartStepID != startStepID {
		t.Errorf("NewSchedule() StartStepID = %v, want %v", schedule.StartStepID, startStepID)
	}
	if schedule.ProjectVersion != projectVersion {
		t.Errorf("NewSchedule() ProjectVersion = %v, want %v", schedule.ProjectVersion, projectVersion)
	}
	if schedule.Name != name {
		t.Errorf("NewSchedule() Name = %v, want %v", schedule.Name, name)
	}
	if schedule.CronExpression != cron {
		t.Errorf("NewSchedule() CronExpression = %v, want %v", schedule.CronExpression, cron)
	}
	if schedule.Timezone != timezone {
		t.Errorf("NewSchedule() Timezone = %v, want %v", schedule.Timezone, timezone)
	}
	if schedule.Status != ScheduleStatusActive {
		t.Errorf("NewSchedule() Status = %v, want %v", schedule.Status, ScheduleStatusActive)
	}
	if schedule.RunCount != 0 {
		t.Errorf("NewSchedule() RunCount = %v, want 0", schedule.RunCount)
	}
}

func TestSchedule_IsActive(t *testing.T) {
	tests := []struct {
		status ScheduleStatus
		want   bool
	}{
		{ScheduleStatusActive, true},
		{ScheduleStatusPaused, false},
		{ScheduleStatusDisabled, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			schedule := &Schedule{Status: tt.status}
			if got := schedule.IsActive(); got != tt.want {
				t.Errorf("IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSchedule_Pause(t *testing.T) {
	schedule := &Schedule{Status: ScheduleStatusActive}

	schedule.Pause()

	if schedule.Status != ScheduleStatusPaused {
		t.Errorf("Pause() Status = %v, want %v", schedule.Status, ScheduleStatusPaused)
	}
}

func TestSchedule_Resume(t *testing.T) {
	schedule := &Schedule{Status: ScheduleStatusPaused}

	schedule.Resume()

	if schedule.Status != ScheduleStatusActive {
		t.Errorf("Resume() Status = %v, want %v", schedule.Status, ScheduleStatusActive)
	}
}

func TestSchedule_Disable(t *testing.T) {
	schedule := &Schedule{Status: ScheduleStatusActive}

	schedule.Disable()

	if schedule.Status != ScheduleStatusDisabled {
		t.Errorf("Disable() Status = %v, want %v", schedule.Status, ScheduleStatusDisabled)
	}
}

func TestSchedule_RecordRun(t *testing.T) {
	schedule := &Schedule{
		Status:   ScheduleStatusActive,
		RunCount: 5,
	}
	runID := uuid.New()
	nextRunAt := time.Now().Add(time.Hour)

	schedule.RecordRun(runID, &nextRunAt)

	if schedule.LastRunID == nil || *schedule.LastRunID != runID {
		t.Error("RecordRun() LastRunID mismatch")
	}
	if schedule.LastRunAt == nil {
		t.Error("RecordRun() LastRunAt should not be nil")
	}
	if schedule.NextRunAt == nil || *schedule.NextRunAt != nextRunAt {
		t.Error("RecordRun() NextRunAt mismatch")
	}
	if schedule.RunCount != 6 {
		t.Errorf("RecordRun() RunCount = %v, want 6", schedule.RunCount)
	}
}

func TestSchedule_UpdateNextRun(t *testing.T) {
	schedule := &Schedule{}
	nextRunAt := time.Now().Add(time.Hour)

	schedule.UpdateNextRun(&nextRunAt)

	if schedule.NextRunAt == nil || *schedule.NextRunAt != nextRunAt {
		t.Error("UpdateNextRun() NextRunAt mismatch")
	}
}
