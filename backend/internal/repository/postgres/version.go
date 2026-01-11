package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/souta/ai-orchestration/internal/domain"
)

// WorkflowVersionRepository implements repository.WorkflowVersionRepository
type WorkflowVersionRepository struct {
	pool *pgxpool.Pool
}

// NewWorkflowVersionRepository creates a new WorkflowVersionRepository
func NewWorkflowVersionRepository(pool *pgxpool.Pool) *WorkflowVersionRepository {
	return &WorkflowVersionRepository{pool: pool}
}

// Create creates a new workflow version snapshot
func (r *WorkflowVersionRepository) Create(ctx context.Context, v *domain.WorkflowVersion) error {
	query := `
		INSERT INTO workflow_versions (id, workflow_id, version, definition, saved_by, saved_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, query,
		v.ID, v.WorkflowID, v.Version, v.Definition, v.SavedBy, v.SavedAt,
	)
	return err
}

// GetByWorkflowAndVersion retrieves a specific version of a workflow
func (r *WorkflowVersionRepository) GetByWorkflowAndVersion(ctx context.Context, workflowID uuid.UUID, version int) (*domain.WorkflowVersion, error) {
	query := `
		SELECT id, workflow_id, version, definition, saved_by, saved_at
		FROM workflow_versions
		WHERE workflow_id = $1 AND version = $2
	`
	var v domain.WorkflowVersion
	err := r.pool.QueryRow(ctx, query, workflowID, version).Scan(
		&v.ID, &v.WorkflowID, &v.Version, &v.Definition, &v.SavedBy, &v.SavedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrWorkflowVersionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// GetLatestByWorkflow retrieves the latest version of a workflow
func (r *WorkflowVersionRepository) GetLatestByWorkflow(ctx context.Context, workflowID uuid.UUID) (*domain.WorkflowVersion, error) {
	query := `
		SELECT id, workflow_id, version, definition, saved_by, saved_at
		FROM workflow_versions
		WHERE workflow_id = $1
		ORDER BY version DESC
		LIMIT 1
	`
	var v domain.WorkflowVersion
	err := r.pool.QueryRow(ctx, query, workflowID).Scan(
		&v.ID, &v.WorkflowID, &v.Version, &v.Definition, &v.SavedBy, &v.SavedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrWorkflowVersionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// ListByWorkflow retrieves all versions of a workflow
func (r *WorkflowVersionRepository) ListByWorkflow(ctx context.Context, workflowID uuid.UUID) ([]*domain.WorkflowVersion, error) {
	query := `
		SELECT id, workflow_id, version, definition, saved_by, saved_at
		FROM workflow_versions
		WHERE workflow_id = $1
		ORDER BY version DESC
	`
	rows, err := r.pool.Query(ctx, query, workflowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []*domain.WorkflowVersion
	for rows.Next() {
		var v domain.WorkflowVersion
		if err := rows.Scan(
			&v.ID, &v.WorkflowID, &v.Version, &v.Definition, &v.SavedBy, &v.SavedAt,
		); err != nil {
			return nil, err
		}
		versions = append(versions, &v)
	}

	return versions, nil
}
