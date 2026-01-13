package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/souta/ai-orchestration/internal/domain"
)

// StepRunRepository implements repository.StepRunRepository
type StepRunRepository struct {
	pool *pgxpool.Pool
}

// NewStepRunRepository creates a new StepRunRepository
func NewStepRunRepository(pool *pgxpool.Pool) *StepRunRepository {
	return &StepRunRepository{pool: pool}
}

// Create creates a new step run
func (r *StepRunRepository) Create(ctx context.Context, sr *domain.StepRun) error {
	query := `
		INSERT INTO step_runs (id, tenant_id, run_id, step_id, step_name, status, attempt, input, output, error, started_at, completed_at, duration_ms, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err := r.pool.Exec(ctx, query,
		sr.ID, sr.TenantID, sr.RunID, sr.StepID, sr.StepName, sr.Status, sr.Attempt,
		sr.Input, sr.Output, sr.Error, sr.StartedAt, sr.CompletedAt, sr.DurationMs, sr.CreatedAt,
	)
	return err
}

// GetByID retrieves a step run by ID
func (r *StepRunRepository) GetByID(ctx context.Context, tenantID, runID, id uuid.UUID) (*domain.StepRun, error) {
	query := `
		SELECT id, tenant_id, run_id, step_id, step_name, status, attempt, input, output, error, started_at, completed_at, duration_ms, created_at
		FROM step_runs
		WHERE id = $1 AND run_id = $2 AND tenant_id = $3
	`
	var sr domain.StepRun
	err := r.pool.QueryRow(ctx, query, id, runID, tenantID).Scan(
		&sr.ID, &sr.TenantID, &sr.RunID, &sr.StepID, &sr.StepName, &sr.Status, &sr.Attempt,
		&sr.Input, &sr.Output, &sr.Error, &sr.StartedAt, &sr.CompletedAt, &sr.DurationMs, &sr.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrStepRunNotFound
	}
	if err != nil {
		return nil, err
	}
	return &sr, nil
}

// ListByRun retrieves all step runs for a given run
func (r *StepRunRepository) ListByRun(ctx context.Context, tenantID, runID uuid.UUID) ([]*domain.StepRun, error) {
	query := `
		SELECT id, tenant_id, run_id, step_id, step_name, status, attempt, input, output, error, started_at, completed_at, duration_ms, created_at
		FROM step_runs
		WHERE run_id = $1 AND tenant_id = $2
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, runID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stepRuns []*domain.StepRun
	for rows.Next() {
		var sr domain.StepRun
		if err := rows.Scan(
			&sr.ID, &sr.TenantID, &sr.RunID, &sr.StepID, &sr.StepName, &sr.Status, &sr.Attempt,
			&sr.Input, &sr.Output, &sr.Error, &sr.StartedAt, &sr.CompletedAt, &sr.DurationMs, &sr.CreatedAt,
		); err != nil {
			return nil, err
		}
		stepRuns = append(stepRuns, &sr)
	}

	return stepRuns, nil
}

// Update updates a step run
func (r *StepRunRepository) Update(ctx context.Context, sr *domain.StepRun) error {
	query := `
		UPDATE step_runs
		SET status = $1, attempt = $2, input = $3, output = $4, error = $5, started_at = $6, completed_at = $7, duration_ms = $8
		WHERE id = $9 AND tenant_id = $10
	`
	result, err := r.pool.Exec(ctx, query,
		sr.Status, sr.Attempt, sr.Input, sr.Output, sr.Error, sr.StartedAt, sr.CompletedAt, sr.DurationMs,
		sr.ID, sr.TenantID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrStepRunNotFound
	}
	return nil
}

// GetMaxAttempt returns the highest attempt number for a step in a run
func (r *StepRunRepository) GetMaxAttempt(ctx context.Context, tenantID, runID, stepID uuid.UUID) (int, error) {
	query := `SELECT COALESCE(MAX(attempt), 0) FROM step_runs WHERE run_id = $1 AND step_id = $2 AND tenant_id = $3`
	var maxAttempt int
	err := r.pool.QueryRow(ctx, query, runID, stepID, tenantID).Scan(&maxAttempt)
	if err != nil {
		return 0, err
	}
	return maxAttempt, nil
}

// GetMaxAttemptForRun returns the highest attempt number across all steps in a run
func (r *StepRunRepository) GetMaxAttemptForRun(ctx context.Context, tenantID, runID uuid.UUID) (int, error) {
	query := `SELECT COALESCE(MAX(attempt), 0) FROM step_runs WHERE run_id = $1 AND tenant_id = $2`
	var maxAttempt int
	err := r.pool.QueryRow(ctx, query, runID, tenantID).Scan(&maxAttempt)
	if err != nil {
		return 0, err
	}
	return maxAttempt, nil
}

// GetLatestByStep returns the most recent StepRun for a step in a run
func (r *StepRunRepository) GetLatestByStep(ctx context.Context, tenantID, runID, stepID uuid.UUID) (*domain.StepRun, error) {
	query := `
		SELECT id, tenant_id, run_id, step_id, step_name, status, attempt, input, output, error, started_at, completed_at, duration_ms, created_at
		FROM step_runs
		WHERE run_id = $1 AND step_id = $2 AND tenant_id = $3
		ORDER BY attempt DESC
		LIMIT 1
	`
	var sr domain.StepRun
	err := r.pool.QueryRow(ctx, query, runID, stepID, tenantID).Scan(
		&sr.ID, &sr.TenantID, &sr.RunID, &sr.StepID, &sr.StepName, &sr.Status, &sr.Attempt,
		&sr.Input, &sr.Output, &sr.Error, &sr.StartedAt, &sr.CompletedAt, &sr.DurationMs, &sr.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrStepRunNotFound
	}
	if err != nil {
		return nil, err
	}
	return &sr, nil
}

// ListCompletedByRun returns the latest completed StepRun for each step in a run
func (r *StepRunRepository) ListCompletedByRun(ctx context.Context, tenantID, runID uuid.UUID) ([]*domain.StepRun, error) {
	query := `
		SELECT DISTINCT ON (step_id)
			id, tenant_id, run_id, step_id, step_name, status, attempt, input, output, error, started_at, completed_at, duration_ms, created_at
		FROM step_runs
		WHERE run_id = $1 AND tenant_id = $2 AND status = 'completed'
		ORDER BY step_id, attempt DESC
	`
	rows, err := r.pool.Query(ctx, query, runID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stepRuns []*domain.StepRun
	for rows.Next() {
		var sr domain.StepRun
		if err := rows.Scan(
			&sr.ID, &sr.TenantID, &sr.RunID, &sr.StepID, &sr.StepName, &sr.Status, &sr.Attempt,
			&sr.Input, &sr.Output, &sr.Error, &sr.StartedAt, &sr.CompletedAt, &sr.DurationMs, &sr.CreatedAt,
		); err != nil {
			return nil, err
		}
		stepRuns = append(stepRuns, &sr)
	}

	return stepRuns, nil
}

// ListByStep returns all StepRuns for a specific step in a run (for history)
func (r *StepRunRepository) ListByStep(ctx context.Context, tenantID, runID, stepID uuid.UUID) ([]*domain.StepRun, error) {
	query := `
		SELECT id, tenant_id, run_id, step_id, step_name, status, attempt, input, output, error, started_at, completed_at, duration_ms, created_at
		FROM step_runs
		WHERE run_id = $1 AND step_id = $2 AND tenant_id = $3
		ORDER BY attempt ASC
	`
	rows, err := r.pool.Query(ctx, query, runID, stepID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stepRuns []*domain.StepRun
	for rows.Next() {
		var sr domain.StepRun
		if err := rows.Scan(
			&sr.ID, &sr.TenantID, &sr.RunID, &sr.StepID, &sr.StepName, &sr.Status, &sr.Attempt,
			&sr.Input, &sr.Output, &sr.Error, &sr.StartedAt, &sr.CompletedAt, &sr.DurationMs, &sr.CreatedAt,
		); err != nil {
			return nil, err
		}
		stepRuns = append(stepRuns, &sr)
	}

	return stepRuns, nil
}
