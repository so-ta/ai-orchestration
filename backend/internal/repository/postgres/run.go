package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// RunRepository implements repository.RunRepository
type RunRepository struct {
	pool *pgxpool.Pool
}

// NewRunRepository creates a new RunRepository
func NewRunRepository(pool *pgxpool.Pool) *RunRepository {
	return &RunRepository{pool: pool}
}

// Create creates a new run
func (r *RunRepository) Create(ctx context.Context, run *domain.Run) error {
	query := `
		INSERT INTO runs (id, tenant_id, workflow_id, workflow_version, status, mode, input,
		                  triggered_by, triggered_by_user, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.pool.Exec(ctx, query,
		run.ID, run.TenantID, run.WorkflowID, run.WorkflowVersion, run.Status, run.Mode,
		run.Input, run.TriggeredBy, run.TriggeredByUser, run.CreatedAt,
	)
	return err
}

// GetByID retrieves a run by ID
func (r *RunRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Run, error) {
	query := `
		SELECT id, tenant_id, workflow_id, workflow_version, status, mode, input, output, error,
		       triggered_by, triggered_by_user, started_at, completed_at, created_at
		FROM runs
		WHERE id = $1 AND tenant_id = $2
	`
	var run domain.Run
	err := r.pool.QueryRow(ctx, query, id, tenantID).Scan(
		&run.ID, &run.TenantID, &run.WorkflowID, &run.WorkflowVersion, &run.Status, &run.Mode,
		&run.Input, &run.Output, &run.Error, &run.TriggeredBy, &run.TriggeredByUser,
		&run.StartedAt, &run.CompletedAt, &run.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrRunNotFound
	}
	if err != nil {
		return nil, err
	}
	return &run, nil
}

// ListByWorkflow retrieves runs for a workflow with pagination
func (r *RunRepository) ListByWorkflow(ctx context.Context, tenantID, workflowID uuid.UUID, filter repository.RunFilter) ([]*domain.Run, int, error) {
	// Count query
	countQuery := `SELECT COUNT(*) FROM runs WHERE tenant_id = $1 AND workflow_id = $2`
	args := []interface{}{tenantID, workflowID}

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// List query
	query := `
		SELECT id, tenant_id, workflow_id, workflow_version, status, mode, input, output, error,
		       triggered_by, triggered_by_user, started_at, completed_at, created_at
		FROM runs
		WHERE tenant_id = $1 AND workflow_id = $2
		ORDER BY created_at DESC
	`

	if filter.Limit > 0 {
		offset := (filter.Page - 1) * filter.Limit
		query += ` LIMIT $3 OFFSET $4`
		args = append(args, filter.Limit, offset)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var runs []*domain.Run
	for rows.Next() {
		var run domain.Run
		if err := rows.Scan(
			&run.ID, &run.TenantID, &run.WorkflowID, &run.WorkflowVersion, &run.Status, &run.Mode,
			&run.Input, &run.Output, &run.Error, &run.TriggeredBy, &run.TriggeredByUser,
			&run.StartedAt, &run.CompletedAt, &run.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		runs = append(runs, &run)
	}

	return runs, total, nil
}

// Update updates a run
func (r *RunRepository) Update(ctx context.Context, run *domain.Run) error {
	query := `
		UPDATE runs
		SET status = $1, output = $2, error = $3, started_at = $4, completed_at = $5
		WHERE id = $6 AND tenant_id = $7
	`
	result, err := r.pool.Exec(ctx, query,
		run.Status, run.Output, run.Error, run.StartedAt, run.CompletedAt,
		run.ID, run.TenantID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrRunNotFound
	}
	return nil
}

// GetWithStepRuns retrieves a run with its step runs
func (r *RunRepository) GetWithStepRuns(ctx context.Context, tenantID, id uuid.UUID) (*domain.Run, error) {
	run, err := r.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, run_id, step_id, step_name, status, attempt, input, output, error,
		       started_at, completed_at, duration_ms, created_at
		FROM step_runs
		WHERE run_id = $1
		ORDER BY created_at
	`
	rows, err := r.pool.Query(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var sr domain.StepRun
		if err := rows.Scan(
			&sr.ID, &sr.RunID, &sr.StepID, &sr.StepName, &sr.Status, &sr.Attempt,
			&sr.Input, &sr.Output, &sr.Error, &sr.StartedAt, &sr.CompletedAt,
			&sr.DurationMs, &sr.CreatedAt,
		); err != nil {
			return nil, err
		}
		run.StepRuns = append(run.StepRuns, sr)
	}

	return run, nil
}
