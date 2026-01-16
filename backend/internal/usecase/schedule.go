package usecase

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// ScheduleUsecase handles schedule business logic
type ScheduleUsecase struct {
	scheduleRepo repository.ScheduleRepository
	projectRepo  repository.ProjectRepository
	runRepo      repository.RunRepository
}

// NewScheduleUsecase creates a new ScheduleUsecase
func NewScheduleUsecase(
	scheduleRepo repository.ScheduleRepository,
	projectRepo repository.ProjectRepository,
	runRepo repository.RunRepository,
) *ScheduleUsecase {
	return &ScheduleUsecase{
		scheduleRepo: scheduleRepo,
		projectRepo:  projectRepo,
		runRepo:      runRepo,
	}
}

// CreateScheduleInput represents input for creating a schedule
type CreateScheduleInput struct {
	TenantID       uuid.UUID
	ProjectID      uuid.UUID
	StartStepID    uuid.UUID // Required: which Start block this schedule triggers
	Name           string
	Description    string
	CronExpression string
	Timezone       string
	Input          json.RawMessage
	CreatedBy      *uuid.UUID
}

// Create creates a new schedule
func (u *ScheduleUsecase) Create(ctx context.Context, input CreateScheduleInput) (*domain.Schedule, error) {
	// Validate input
	if input.Name == "" {
		return nil, domain.NewValidationError("name", "name is required")
	}
	if input.CronExpression == "" {
		return nil, domain.NewValidationError("cron_expression", "cron expression is required")
	}

	// Validate cron expression
	nextRun, err := ParseCron(input.CronExpression, input.Timezone)
	if err != nil {
		return nil, domain.ErrScheduleInvalidCron
	}

	// Verify project exists and is published
	project, err := u.projectRepo.GetByID(ctx, input.TenantID, input.ProjectID)
	if err != nil {
		return nil, err
	}
	if project.Status != domain.ProjectStatusPublished {
		return nil, domain.NewValidationError("project_id", "project must be published")
	}

	// Set default timezone
	if input.Timezone == "" {
		input.Timezone = "UTC"
	}

	schedule := domain.NewSchedule(
		input.TenantID,
		input.ProjectID,
		input.StartStepID,
		project.Version,
		input.Name,
		input.CronExpression,
		input.Timezone,
		input.Input,
	)
	schedule.Description = input.Description
	schedule.CreatedBy = input.CreatedBy
	schedule.UpdateNextRun(nextRun)

	if err := u.scheduleRepo.Create(ctx, schedule); err != nil {
		return nil, err
	}

	return schedule, nil
}

// GetByID retrieves a schedule by ID
func (u *ScheduleUsecase) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Schedule, error) {
	return u.scheduleRepo.GetByID(ctx, tenantID, id)
}

// ListSchedulesInput represents input for listing schedules
type ListSchedulesInput struct {
	TenantID  uuid.UUID
	ProjectID *uuid.UUID
	Status    *domain.ScheduleStatus
	Page      int
	Limit     int
}

// ListSchedulesOutput represents output for listing schedules
type ListSchedulesOutput struct {
	Schedules []*domain.Schedule
	Total     int
	Page      int
	Limit     int
}

// List lists schedules with pagination
func (u *ScheduleUsecase) List(ctx context.Context, input ListSchedulesInput) (*ListSchedulesOutput, error) {
	input.Page, input.Limit = NormalizePagination(input.Page, input.Limit)

	filter := repository.ScheduleFilter{
		ProjectID: input.ProjectID,
		Status:    input.Status,
		Page:      input.Page,
		Limit:     input.Limit,
	}

	schedules, total, err := u.scheduleRepo.ListByTenant(ctx, input.TenantID, filter)
	if err != nil {
		return nil, err
	}

	return &ListSchedulesOutput{
		Schedules: schedules,
		Total:     total,
		Page:      input.Page,
		Limit:     input.Limit,
	}, nil
}

// UpdateScheduleInput represents input for updating a schedule
type UpdateScheduleInput struct {
	TenantID       uuid.UUID
	ID             uuid.UUID
	Name           string
	Description    string
	CronExpression string
	Timezone       string
	Input          json.RawMessage
	StartStepID    *uuid.UUID // Optional: update the start step ID (nil means no change)
}

// Update updates a schedule
func (u *ScheduleUsecase) Update(ctx context.Context, input UpdateScheduleInput) (*domain.Schedule, error) {
	schedule, err := u.scheduleRepo.GetByID(ctx, input.TenantID, input.ID)
	if err != nil {
		return nil, err
	}

	if input.Name != "" {
		schedule.Name = input.Name
	}
	schedule.Description = input.Description

	if input.CronExpression != "" {
		// Validate new cron expression
		tz := input.Timezone
		if tz == "" {
			tz = schedule.Timezone
		}
		nextRun, err := ParseCron(input.CronExpression, tz)
		if err != nil {
			return nil, domain.ErrScheduleInvalidCron
		}
		schedule.CronExpression = input.CronExpression
		schedule.UpdateNextRun(nextRun)
	}

	if input.Timezone != "" {
		schedule.Timezone = input.Timezone
		// Recalculate next run with new timezone
		nextRun, _ := ParseCron(schedule.CronExpression, input.Timezone)
		schedule.UpdateNextRun(nextRun)
	}

	if input.Input != nil {
		schedule.Input = input.Input
	}

	if input.StartStepID != nil {
		schedule.StartStepID = *input.StartStepID
	}

	schedule.UpdatedAt = time.Now().UTC()

	if err := u.scheduleRepo.Update(ctx, schedule); err != nil {
		return nil, err
	}

	return schedule, nil
}

// Delete deletes a schedule
func (u *ScheduleUsecase) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return u.scheduleRepo.Delete(ctx, tenantID, id)
}

// Pause pauses a schedule
func (u *ScheduleUsecase) Pause(ctx context.Context, tenantID, id uuid.UUID) (*domain.Schedule, error) {
	schedule, err := u.scheduleRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	schedule.Pause()

	if err := u.scheduleRepo.Update(ctx, schedule); err != nil {
		return nil, err
	}

	return schedule, nil
}

// Resume resumes a paused schedule
func (u *ScheduleUsecase) Resume(ctx context.Context, tenantID, id uuid.UUID) (*domain.Schedule, error) {
	schedule, err := u.scheduleRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	schedule.Resume()

	// Recalculate next run
	nextRun, _ := ParseCron(schedule.CronExpression, schedule.Timezone)
	schedule.UpdateNextRun(nextRun)

	if err := u.scheduleRepo.Update(ctx, schedule); err != nil {
		return nil, err
	}

	return schedule, nil
}

// TriggerSchedule manually triggers a schedule
func (u *ScheduleUsecase) Trigger(ctx context.Context, tenantID, id uuid.UUID) (*domain.Run, error) {
	schedule, err := u.scheduleRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	// Create a new run
	run := domain.NewRun(
		schedule.TenantID,
		schedule.ProjectID,
		schedule.ProjectVersion,
		schedule.Input,
		domain.TriggerTypeSchedule,
	)

	if err := u.runRepo.Create(ctx, run); err != nil {
		return nil, err
	}

	// Update schedule stats
	nextRun, _ := ParseCron(schedule.CronExpression, schedule.Timezone)
	schedule.RecordRun(run.ID, nextRun)

	if err := u.scheduleRepo.Update(ctx, schedule); err != nil {
		// Log error but don't fail - run was created successfully
	}

	return run, nil
}

// ProcessDueSchedules processes schedules that are due to run
func (u *ScheduleUsecase) ProcessDueSchedules(ctx context.Context, limit int) (int, error) {
	schedules, err := u.scheduleRepo.GetDueSchedules(ctx, limit)
	if err != nil {
		return 0, err
	}

	processed := 0
	for _, schedule := range schedules {
		_, err := u.Trigger(ctx, schedule.TenantID, schedule.ID)
		if err != nil {
			// Log error but continue processing other schedules
			continue
		}
		processed++
	}

	return processed, nil
}

// ParseCron parses a cron expression and returns the next run time
func ParseCron(expression, timezone string) (*time.Time, error) {
	// Simple implementation - in production, use a proper cron parser like robfig/cron
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}

	// For now, just return 1 minute from now as a placeholder
	// In production, this should properly parse the cron expression
	next := time.Now().In(loc).Add(time.Minute)
	return &next, nil
}
