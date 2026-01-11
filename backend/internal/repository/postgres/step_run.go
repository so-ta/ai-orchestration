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
		INSERT INTO step_runs (id, run_id, step_id, step_name, status, attempt, input, output, error, started_at, completed_at, duration_ms, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := r.pool.Exec(ctx, query,
		sr.ID, sr.RunID, sr.StepID, sr.StepName, sr.Status, sr.Attempt,
		sr.Input, sr.Output, sr.Error, sr.StartedAt, sr.CompletedAt, sr.DurationMs, sr.CreatedAt,
	)
	return err
}

// GetByID retrieves a step run by ID
func (r *StepRunRepository) GetByID(ctx context.Context, runID, id uuid.UUID) (*domain.StepRun, error) {
	query := `
		SELECT id, run_id, step_id, step_name, status, attempt, input, output, error, started_at, completed_at, duration_ms, created_at
		FROM step_runs
		WHERE run_id = $1 AND id = $2
	`
	var sr domain.StepRun
	err := r.pool.QueryRow(ctx, query, runID, id).Scan(
		&sr.ID, &sr.RunID, &sr.StepID, &sr.StepName, &sr.Status, &sr.Attempt,
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
func (r *StepRunRepository) ListByRun(ctx context.Context, runID uuid.UUID) ([]*domain.StepRun, error) {
	query := `
		SELECT id, run_id, step_id, step_name, status, attempt, input, output, error, started_at, completed_at, duration_ms, created_at
		FROM step_runs
		WHERE run_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stepRuns []*domain.StepRun
	for rows.Next() {
		var sr domain.StepRun
		if err := rows.Scan(
			&sr.ID, &sr.RunID, &sr.StepID, &sr.StepName, &sr.Status, &sr.Attempt,
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
		WHERE id = $9
	`
	_, err := r.pool.Exec(ctx, query,
		sr.Status, sr.Attempt, sr.Input, sr.Output, sr.Error, sr.StartedAt, sr.CompletedAt, sr.DurationMs,
		sr.ID,
	)
	return err
}
