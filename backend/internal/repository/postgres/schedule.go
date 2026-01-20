package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// ScheduleRepository implements repository.ScheduleRepository
type ScheduleRepository struct {
	pool *pgxpool.Pool
}

// NewScheduleRepository creates a new ScheduleRepository
func NewScheduleRepository(pool *pgxpool.Pool) *ScheduleRepository {
	return &ScheduleRepository{pool: pool}
}

func (r *ScheduleRepository) Create(ctx context.Context, schedule *domain.Schedule) error {
	query := `
		INSERT INTO schedules (
			id, tenant_id, project_id, project_version, start_step_id, name, description,
			cron_expression, timezone, input, status, next_run_at, created_by,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	_, err := r.pool.Exec(ctx, query,
		schedule.ID,
		schedule.TenantID,
		schedule.ProjectID,
		schedule.ProjectVersion,
		schedule.StartStepID,
		schedule.Name,
		schedule.Description,
		schedule.CronExpression,
		schedule.Timezone,
		schedule.Input,
		schedule.Status,
		schedule.NextRunAt,
		schedule.CreatedBy,
		schedule.CreatedAt,
		schedule.UpdatedAt,
	)

	return err
}

func (r *ScheduleRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Schedule, error) {
	query := `
		SELECT id, tenant_id, project_id, project_version, start_step_id, name, description,
			   cron_expression, timezone, input, status, next_run_at, last_run_at,
			   last_run_id, run_count, created_by, created_at, updated_at
		FROM schedules
		WHERE tenant_id = $1 AND id = $2
	`

	var schedule domain.Schedule
	err := r.pool.QueryRow(ctx, query, tenantID, id).Scan(
		&schedule.ID,
		&schedule.TenantID,
		&schedule.ProjectID,
		&schedule.ProjectVersion,
		&schedule.StartStepID,
		&schedule.Name,
		&schedule.Description,
		&schedule.CronExpression,
		&schedule.Timezone,
		&schedule.Input,
		&schedule.Status,
		&schedule.NextRunAt,
		&schedule.LastRunAt,
		&schedule.LastRunID,
		&schedule.RunCount,
		&schedule.CreatedBy,
		&schedule.CreatedAt,
		&schedule.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, domain.ErrScheduleNotFound
	}
	if err != nil {
		return nil, err
	}

	return &schedule, nil
}

func (r *ScheduleRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, filter repository.ScheduleFilter) ([]*domain.Schedule, int, error) {
	// Count query
	countQuery := `SELECT COUNT(*) FROM schedules WHERE tenant_id = $1`
	countArgs := []interface{}{tenantID}
	argIndex := 2

	if filter.ProjectID != nil {
		countQuery += fmt.Sprintf(` AND project_id = $%d`, argIndex)
		countArgs = append(countArgs, *filter.ProjectID)
		argIndex++
	}
	if filter.Status != nil {
		countQuery += fmt.Sprintf(` AND status = $%d`, argIndex)
		countArgs = append(countArgs, *filter.Status)
	}

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count schedules: %w", err)
	}

	// List query
	query := `
		SELECT id, tenant_id, project_id, project_version, start_step_id, name, description,
			   cron_expression, timezone, input, status, next_run_at, last_run_at,
			   last_run_id, run_count, created_by, created_at, updated_at
		FROM schedules
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}
	argIdx := 2

	if filter.ProjectID != nil {
		query += fmt.Sprintf(` AND project_id = $%d`, argIdx)
		args = append(args, *filter.ProjectID)
		argIdx++
	}
	if filter.Status != nil {
		query += fmt.Sprintf(` AND status = $%d`, argIdx)
		args = append(args, *filter.Status)
		argIdx++
	}

	query += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, argIdx, argIdx+1)
	page := filter.Page
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * filter.Limit
	args = append(args, filter.Limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list schedules: %w", err)
	}
	defer rows.Close()

	var schedules []*domain.Schedule
	for rows.Next() {
		var s domain.Schedule
		err := rows.Scan(
			&s.ID, &s.TenantID, &s.ProjectID, &s.ProjectVersion,
			&s.StartStepID, &s.Name, &s.Description, &s.CronExpression, &s.Timezone,
			&s.Input, &s.Status, &s.NextRunAt, &s.LastRunAt,
			&s.LastRunID, &s.RunCount, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan schedule: %w", err)
		}
		schedules = append(schedules, &s)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate schedules: %w", err)
	}

	return schedules, total, nil
}

func (r *ScheduleRepository) ListByProject(ctx context.Context, tenantID, projectID uuid.UUID) ([]*domain.Schedule, error) {
	query := `
		SELECT id, tenant_id, project_id, project_version, start_step_id, name, description,
			   cron_expression, timezone, input, status, next_run_at, last_run_at,
			   last_run_id, run_count, created_by, created_at, updated_at
		FROM schedules
		WHERE tenant_id = $1 AND project_id = $2
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, tenantID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []*domain.Schedule
	for rows.Next() {
		var s domain.Schedule
		err := rows.Scan(
			&s.ID, &s.TenantID, &s.ProjectID, &s.ProjectVersion,
			&s.StartStepID, &s.Name, &s.Description, &s.CronExpression, &s.Timezone,
			&s.Input, &s.Status, &s.NextRunAt, &s.LastRunAt,
			&s.LastRunID, &s.RunCount, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, &s)
	}

	return schedules, nil
}

func (r *ScheduleRepository) ListByStartStep(ctx context.Context, tenantID, projectID, startStepID uuid.UUID) ([]*domain.Schedule, error) {
	query := `
		SELECT id, tenant_id, project_id, project_version, start_step_id, name, description,
			   cron_expression, timezone, input, status, next_run_at, last_run_at,
			   last_run_id, run_count, created_by, created_at, updated_at
		FROM schedules
		WHERE tenant_id = $1 AND project_id = $2 AND start_step_id = $3
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, tenantID, projectID, startStepID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []*domain.Schedule
	for rows.Next() {
		var s domain.Schedule
		err := rows.Scan(
			&s.ID, &s.TenantID, &s.ProjectID, &s.ProjectVersion,
			&s.StartStepID, &s.Name, &s.Description, &s.CronExpression, &s.Timezone,
			&s.Input, &s.Status, &s.NextRunAt, &s.LastRunAt,
			&s.LastRunID, &s.RunCount, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, &s)
	}

	return schedules, nil
}

func (r *ScheduleRepository) Update(ctx context.Context, schedule *domain.Schedule) error {
	query := `
		UPDATE schedules SET
			name = $3,
			description = $4,
			cron_expression = $5,
			timezone = $6,
			input = $7,
			status = $8,
			next_run_at = $9,
			last_run_at = $10,
			last_run_id = $11,
			run_count = $12,
			updated_at = $13
		WHERE tenant_id = $1 AND id = $2
	`

	result, err := r.pool.Exec(ctx, query,
		schedule.TenantID,
		schedule.ID,
		schedule.Name,
		schedule.Description,
		schedule.CronExpression,
		schedule.Timezone,
		schedule.Input,
		schedule.Status,
		schedule.NextRunAt,
		schedule.LastRunAt,
		schedule.LastRunID,
		schedule.RunCount,
		schedule.UpdatedAt,
	)

	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrScheduleNotFound
	}

	return nil
}

func (r *ScheduleRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	query := `DELETE FROM schedules WHERE tenant_id = $1 AND id = $2`

	result, err := r.pool.Exec(ctx, query, tenantID, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrScheduleNotFound
	}

	return nil
}

func (r *ScheduleRepository) GetDueSchedules(ctx context.Context, limit int) ([]*domain.Schedule, error) {
	query := `
		SELECT id, tenant_id, project_id, project_version, start_step_id, name, description,
			   cron_expression, timezone, input, status, next_run_at, last_run_at,
			   last_run_id, run_count, created_by, created_at, updated_at
		FROM schedules
		WHERE status = $1 AND next_run_at <= $2
		ORDER BY next_run_at ASC
		LIMIT $3
	`

	rows, err := r.pool.Query(ctx, query, domain.ScheduleStatusActive, time.Now().UTC(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []*domain.Schedule
	for rows.Next() {
		var s domain.Schedule
		err := rows.Scan(
			&s.ID, &s.TenantID, &s.ProjectID, &s.ProjectVersion,
			&s.StartStepID, &s.Name, &s.Description, &s.CronExpression, &s.Timezone,
			&s.Input, &s.Status, &s.NextRunAt, &s.LastRunAt,
			&s.LastRunID, &s.RunCount, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, &s)
	}

	return schedules, nil
}
