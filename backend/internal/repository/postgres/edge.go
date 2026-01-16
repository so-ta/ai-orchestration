package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/souta/ai-orchestration/internal/domain"
)

// EdgeRepository implements repository.EdgeRepository
type EdgeRepository struct {
	pool *pgxpool.Pool
}

// NewEdgeRepository creates a new EdgeRepository
func NewEdgeRepository(pool *pgxpool.Pool) *EdgeRepository {
	return &EdgeRepository{pool: pool}
}

// Create creates a new edge
func (r *EdgeRepository) Create(ctx context.Context, e *domain.Edge) error {
	query := `
		INSERT INTO edges (id, tenant_id, project_id, source_step_id, target_step_id, source_block_group_id, target_block_group_id, source_port, target_port, condition, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.pool.Exec(ctx, query,
		e.ID, e.TenantID, e.ProjectID, e.SourceStepID, e.TargetStepID, e.SourceBlockGroupID, e.TargetBlockGroupID, e.SourcePort, e.TargetPort, e.Condition, e.CreatedAt,
	)
	return err
}

// GetByID retrieves an edge by ID
func (r *EdgeRepository) GetByID(ctx context.Context, tenantID, projectID, id uuid.UUID) (*domain.Edge, error) {
	query := `
		SELECT id, tenant_id, project_id, source_step_id, target_step_id, source_block_group_id, target_block_group_id, source_port, target_port, condition, created_at
		FROM edges
		WHERE id = $1 AND project_id = $2 AND tenant_id = $3
	`
	var e domain.Edge
	err := r.pool.QueryRow(ctx, query, id, projectID, tenantID).Scan(
		&e.ID, &e.TenantID, &e.ProjectID, &e.SourceStepID, &e.TargetStepID, &e.SourceBlockGroupID, &e.TargetBlockGroupID, &e.SourcePort, &e.TargetPort, &e.Condition, &e.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrEdgeNotFound
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// ListByProject retrieves all edges for a project
func (r *EdgeRepository) ListByProject(ctx context.Context, tenantID, projectID uuid.UUID) ([]*domain.Edge, error) {
	query := `
		SELECT id, tenant_id, project_id, source_step_id, target_step_id, source_block_group_id, target_block_group_id, source_port, target_port, condition, created_at
		FROM edges
		WHERE project_id = $1 AND tenant_id = $2
	`
	rows, err := r.pool.Query(ctx, query, projectID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []*domain.Edge
	for rows.Next() {
		var e domain.Edge
		if err := rows.Scan(
			&e.ID, &e.TenantID, &e.ProjectID, &e.SourceStepID, &e.TargetStepID, &e.SourceBlockGroupID, &e.TargetBlockGroupID, &e.SourcePort, &e.TargetPort, &e.Condition, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		edges = append(edges, &e)
	}

	return edges, nil
}

// Delete deletes an edge
func (r *EdgeRepository) Delete(ctx context.Context, tenantID, projectID, id uuid.UUID) error {
	query := `DELETE FROM edges WHERE id = $1 AND project_id = $2 AND tenant_id = $3`
	result, err := r.pool.Exec(ctx, query, id, projectID, tenantID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrEdgeNotFound
	}
	return nil
}

// Exists checks if an edge exists between two steps
func (r *EdgeRepository) Exists(ctx context.Context, tenantID, projectID, sourceID, targetID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM edges
			WHERE project_id = $1 AND source_step_id = $2 AND target_step_id = $3 AND tenant_id = $4
		)
	`
	var exists bool
	err := r.pool.QueryRow(ctx, query, projectID, sourceID, targetID, tenantID).Scan(&exists)
	return exists, err
}
