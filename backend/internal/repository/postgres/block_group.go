package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/souta/ai-orchestration/internal/domain"
)

// BlockGroupRepository implements repository for block groups
type BlockGroupRepository struct {
	pool *pgxpool.Pool
}

// NewBlockGroupRepository creates a new BlockGroupRepository
func NewBlockGroupRepository(pool *pgxpool.Pool) *BlockGroupRepository {
	return &BlockGroupRepository{pool: pool}
}

// Create creates a new block group
func (r *BlockGroupRepository) Create(ctx context.Context, g *domain.BlockGroup) error {
	query := `
		INSERT INTO block_groups (id, tenant_id, project_id, name, type, config, parent_group_id, pre_process, post_process, position_x, position_y, width, height, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`
	_, err := r.pool.Exec(ctx, query,
		g.ID, g.TenantID, g.ProjectID, g.Name, g.Type, g.Config,
		g.ParentGroupID, g.PreProcess, g.PostProcess,
		g.PositionX, g.PositionY, g.Width, g.Height,
		g.CreatedAt, g.UpdatedAt,
	)
	return err
}

// GetByID retrieves a block group by ID
func (r *BlockGroupRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.BlockGroup, error) {
	query := `
		SELECT id, tenant_id, project_id, name, type, config, parent_group_id, pre_process, post_process, position_x, position_y, width, height, created_at, updated_at
		FROM block_groups
		WHERE id = $1 AND tenant_id = $2
	`
	var g domain.BlockGroup
	err := r.pool.QueryRow(ctx, query, id, tenantID).Scan(
		&g.ID, &g.TenantID, &g.ProjectID, &g.Name, &g.Type, &g.Config,
		&g.ParentGroupID, &g.PreProcess, &g.PostProcess,
		&g.PositionX, &g.PositionY, &g.Width, &g.Height,
		&g.CreatedAt, &g.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrBlockGroupNotFound
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

// ListByProject retrieves all block groups for a project
func (r *BlockGroupRepository) ListByProject(ctx context.Context, tenantID, projectID uuid.UUID) ([]*domain.BlockGroup, error) {
	query := `
		SELECT id, tenant_id, project_id, name, type, config, parent_group_id, pre_process, post_process, position_x, position_y, width, height, created_at, updated_at
		FROM block_groups
		WHERE project_id = $1 AND tenant_id = $2
		ORDER BY created_at
	`
	rows, err := r.pool.Query(ctx, query, projectID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []*domain.BlockGroup
	for rows.Next() {
		var g domain.BlockGroup
		if err := rows.Scan(
			&g.ID, &g.TenantID, &g.ProjectID, &g.Name, &g.Type, &g.Config,
			&g.ParentGroupID, &g.PreProcess, &g.PostProcess,
			&g.PositionX, &g.PositionY, &g.Width, &g.Height,
			&g.CreatedAt, &g.UpdatedAt,
		); err != nil {
			return nil, err
		}
		groups = append(groups, &g)
	}

	return groups, nil
}

// ListByParent retrieves all child block groups of a parent group
func (r *BlockGroupRepository) ListByParent(ctx context.Context, tenantID, parentID uuid.UUID) ([]*domain.BlockGroup, error) {
	query := `
		SELECT id, tenant_id, project_id, name, type, config, parent_group_id, pre_process, post_process, position_x, position_y, width, height, created_at, updated_at
		FROM block_groups
		WHERE parent_group_id = $1 AND tenant_id = $2
		ORDER BY created_at
	`
	rows, err := r.pool.Query(ctx, query, parentID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []*domain.BlockGroup
	for rows.Next() {
		var g domain.BlockGroup
		if err := rows.Scan(
			&g.ID, &g.TenantID, &g.ProjectID, &g.Name, &g.Type, &g.Config,
			&g.ParentGroupID, &g.PreProcess, &g.PostProcess,
			&g.PositionX, &g.PositionY, &g.Width, &g.Height,
			&g.CreatedAt, &g.UpdatedAt,
		); err != nil {
			return nil, err
		}
		groups = append(groups, &g)
	}

	return groups, nil
}

// Update updates a block group
func (r *BlockGroupRepository) Update(ctx context.Context, g *domain.BlockGroup) error {
	g.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE block_groups
		SET name = $1, type = $2, config = $3, parent_group_id = $4, pre_process = $5, post_process = $6, position_x = $7, position_y = $8, width = $9, height = $10, updated_at = $11
		WHERE id = $12 AND tenant_id = $13
	`
	result, err := r.pool.Exec(ctx, query,
		g.Name, g.Type, g.Config, g.ParentGroupID, g.PreProcess, g.PostProcess,
		g.PositionX, g.PositionY, g.Width, g.Height, g.UpdatedAt,
		g.ID, g.TenantID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrBlockGroupNotFound
	}
	return nil
}

// Delete deletes a block group
func (r *BlockGroupRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	query := `DELETE FROM block_groups WHERE id = $1 AND tenant_id = $2`
	result, err := r.pool.Exec(ctx, query, id, tenantID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrBlockGroupNotFound
	}
	return nil
}
