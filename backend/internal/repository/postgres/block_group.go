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

// BlockGroupRunRepository implements repository for block group runs
type BlockGroupRunRepository struct {
	pool *pgxpool.Pool
}

// NewBlockGroupRunRepository creates a new BlockGroupRunRepository
func NewBlockGroupRunRepository(pool *pgxpool.Pool) *BlockGroupRunRepository {
	return &BlockGroupRunRepository{pool: pool}
}

// Create creates a new block group run
func (r *BlockGroupRunRepository) Create(ctx context.Context, gr *domain.BlockGroupRun) error {
	query := `
		INSERT INTO block_group_runs (id, tenant_id, run_id, block_group_id, status, iteration, input, output, error, started_at, completed_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := r.pool.Exec(ctx, query,
		gr.ID, gr.TenantID, gr.RunID, gr.BlockGroupID, gr.Status, gr.Iteration,
		gr.Input, gr.Output, gr.Error, gr.StartedAt, gr.CompletedAt, gr.CreatedAt,
	)
	return err
}

// GetByID retrieves a block group run by ID
func (r *BlockGroupRunRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.BlockGroupRun, error) {
	query := `
		SELECT id, tenant_id, run_id, block_group_id, status, iteration, input, output, error, started_at, completed_at, created_at
		FROM block_group_runs
		WHERE id = $1 AND tenant_id = $2
	`
	var gr domain.BlockGroupRun
	err := r.pool.QueryRow(ctx, query, id, tenantID).Scan(
		&gr.ID, &gr.TenantID, &gr.RunID, &gr.BlockGroupID, &gr.Status, &gr.Iteration,
		&gr.Input, &gr.Output, &gr.Error, &gr.StartedAt, &gr.CompletedAt, &gr.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrBlockGroupRunNotFound
	}
	if err != nil {
		return nil, err
	}
	return &gr, nil
}

// ListByRun retrieves all block group runs for a workflow run
func (r *BlockGroupRunRepository) ListByRun(ctx context.Context, tenantID, runID uuid.UUID) ([]*domain.BlockGroupRun, error) {
	query := `
		SELECT id, tenant_id, run_id, block_group_id, status, iteration, input, output, error, started_at, completed_at, created_at
		FROM block_group_runs
		WHERE run_id = $1 AND tenant_id = $2
		ORDER BY created_at
	`
	rows, err := r.pool.Query(ctx, query, runID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []*domain.BlockGroupRun
	for rows.Next() {
		var gr domain.BlockGroupRun
		if err := rows.Scan(
			&gr.ID, &gr.TenantID, &gr.RunID, &gr.BlockGroupID, &gr.Status, &gr.Iteration,
			&gr.Input, &gr.Output, &gr.Error, &gr.StartedAt, &gr.CompletedAt, &gr.CreatedAt,
		); err != nil {
			return nil, err
		}
		runs = append(runs, &gr)
	}

	return runs, nil
}

// Update updates a block group run
func (r *BlockGroupRunRepository) Update(ctx context.Context, gr *domain.BlockGroupRun) error {
	query := `
		UPDATE block_group_runs
		SET status = $1, iteration = $2, input = $3, output = $4, error = $5, started_at = $6, completed_at = $7
		WHERE id = $8 AND tenant_id = $9
	`
	result, err := r.pool.Exec(ctx, query,
		gr.Status, gr.Iteration, gr.Input, gr.Output, gr.Error, gr.StartedAt, gr.CompletedAt,
		gr.ID, gr.TenantID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrBlockGroupRunNotFound
	}
	return nil
}
