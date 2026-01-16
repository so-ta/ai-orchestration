package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ScheduleStatus represents the status of a schedule
type ScheduleStatus string

const (
	ScheduleStatusActive   ScheduleStatus = "active"
	ScheduleStatusPaused   ScheduleStatus = "paused"
	ScheduleStatusDisabled ScheduleStatus = "disabled"
)

// Schedule represents a scheduled project execution
type Schedule struct {
	ID             uuid.UUID       `json:"id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	ProjectID      uuid.UUID       `json:"project_id"`
	ProjectVersion int             `json:"project_version"`
	StartStepID    uuid.UUID       `json:"start_step_id"` // Required: which Start block this schedule triggers
	Name           string          `json:"name"`
	Description    string          `json:"description,omitempty"`
	CronExpression string          `json:"cron_expression"`
	Timezone       string          `json:"timezone"`
	Input          json.RawMessage `json:"input,omitempty"`
	Status         ScheduleStatus  `json:"status"`
	NextRunAt      *time.Time      `json:"next_run_at,omitempty"`
	LastRunAt      *time.Time      `json:"last_run_at,omitempty"`
	LastRunID      *uuid.UUID      `json:"last_run_id,omitempty"`
	RunCount       int             `json:"run_count"`
	CreatedBy      *uuid.UUID      `json:"created_by,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// NewSchedule creates a new schedule
func NewSchedule(
	tenantID, projectID, startStepID uuid.UUID,
	projectVersion int,
	name, cronExpression, timezone string,
	input json.RawMessage,
) *Schedule {
	now := time.Now().UTC()
	return &Schedule{
		ID:             uuid.New(),
		TenantID:       tenantID,
		ProjectID:      projectID,
		ProjectVersion: projectVersion,
		StartStepID:    startStepID,
		Name:           name,
		CronExpression: cronExpression,
		Timezone:       timezone,
		Input:          input,
		Status:         ScheduleStatusActive,
		RunCount:       0,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// IsActive returns true if the schedule is active
func (s *Schedule) IsActive() bool {
	return s.Status == ScheduleStatusActive
}

// Pause pauses the schedule
func (s *Schedule) Pause() {
	s.Status = ScheduleStatusPaused
	s.UpdatedAt = time.Now().UTC()
}

// Resume resumes a paused schedule
func (s *Schedule) Resume() {
	s.Status = ScheduleStatusActive
	s.UpdatedAt = time.Now().UTC()
}

// Disable disables the schedule
func (s *Schedule) Disable() {
	s.Status = ScheduleStatusDisabled
	s.UpdatedAt = time.Now().UTC()
}

// RecordRun records that a run was triggered
func (s *Schedule) RecordRun(runID uuid.UUID, nextRunAt *time.Time) {
	now := time.Now().UTC()
	s.LastRunAt = &now
	s.LastRunID = &runID
	s.NextRunAt = nextRunAt
	s.RunCount++
	s.UpdatedAt = now
}

// UpdateNextRun updates the next run time
func (s *Schedule) UpdateNextRun(nextRunAt *time.Time) {
	s.NextRunAt = nextRunAt
	s.UpdatedAt = time.Now().UTC()
}
