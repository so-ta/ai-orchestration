package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/souta/ai-orchestration/internal/domain"
)

// BlockVersionRepository implements repository.BlockVersionRepository
type BlockVersionRepository struct {
	pool *pgxpool.Pool
}

// NewBlockVersionRepository creates a new BlockVersionRepository
func NewBlockVersionRepository(pool *pgxpool.Pool) *BlockVersionRepository {
	return &BlockVersionRepository{pool: pool}
}

func (r *BlockVersionRepository) Create(ctx context.Context, version *domain.BlockVersion) error {
	query := `
		INSERT INTO block_versions (
			id, block_id, version,
			code, config_schema, input_schema, output_schema, ui_config,
			change_summary, changed_by, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.pool.Exec(ctx, query,
		version.ID,
		version.BlockID,
		version.Version,
		version.Code,
		version.ConfigSchema,
		version.InputSchema,
		version.OutputSchema,
		version.UIConfig,
		version.ChangeSummary,
		version.ChangedBy,
		version.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create block version: %w", err)
	}

	return nil
}

func (r *BlockVersionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.BlockVersion, error) {
	query := `
		SELECT id, block_id, version,
			   code, config_schema, input_schema, output_schema, ui_config,
			   change_summary, changed_by, created_at
		FROM block_versions
		WHERE id = $1
	`

	version := &domain.BlockVersion{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&version.ID,
		&version.BlockID,
		&version.Version,
		&version.Code,
		&version.ConfigSchema,
		&version.InputSchema,
		&version.OutputSchema,
		&version.UIConfig,
		&version.ChangeSummary,
		&version.ChangedBy,
		&version.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get block version: %w", err)
	}

	return version, nil
}

func (r *BlockVersionRepository) GetByBlockAndVersion(ctx context.Context, blockID uuid.UUID, versionNum int) (*domain.BlockVersion, error) {
	query := `
		SELECT id, block_id, version,
			   code, config_schema, input_schema, output_schema, ui_config,
			   change_summary, changed_by, created_at
		FROM block_versions
		WHERE block_id = $1 AND version = $2
	`

	version := &domain.BlockVersion{}
	err := r.pool.QueryRow(ctx, query, blockID, versionNum).Scan(
		&version.ID,
		&version.BlockID,
		&version.Version,
		&version.Code,
		&version.ConfigSchema,
		&version.InputSchema,
		&version.OutputSchema,
		&version.UIConfig,
		&version.ChangeSummary,
		&version.ChangedBy,
		&version.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get block version by block and version: %w", err)
	}

	return version, nil
}

func (r *BlockVersionRepository) ListByBlock(ctx context.Context, blockID uuid.UUID) ([]*domain.BlockVersion, error) {
	query := `
		SELECT id, block_id, version,
			   code, config_schema, input_schema, output_schema, ui_config,
			   change_summary, changed_by, created_at
		FROM block_versions
		WHERE block_id = $1
		ORDER BY version DESC
	`

	rows, err := r.pool.Query(ctx, query, blockID)
	if err != nil {
		return nil, fmt.Errorf("failed to list block versions: %w", err)
	}
	defer rows.Close()

	var versions []*domain.BlockVersion
	for rows.Next() {
		version := &domain.BlockVersion{}
		err := rows.Scan(
			&version.ID,
			&version.BlockID,
			&version.Version,
			&version.Code,
			&version.ConfigSchema,
			&version.InputSchema,
			&version.OutputSchema,
			&version.UIConfig,
			&version.ChangeSummary,
			&version.ChangedBy,
			&version.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan block version: %w", err)
		}
		versions = append(versions, version)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating block versions: %w", err)
	}

	return versions, nil
}

func (r *BlockVersionRepository) GetLatestByBlock(ctx context.Context, blockID uuid.UUID) (*domain.BlockVersion, error) {
	query := `
		SELECT id, block_id, version,
			   code, config_schema, input_schema, output_schema, ui_config,
			   change_summary, changed_by, created_at
		FROM block_versions
		WHERE block_id = $1
		ORDER BY version DESC
		LIMIT 1
	`

	version := &domain.BlockVersion{}
	err := r.pool.QueryRow(ctx, query, blockID).Scan(
		&version.ID,
		&version.BlockID,
		&version.Version,
		&version.Code,
		&version.ConfigSchema,
		&version.InputSchema,
		&version.OutputSchema,
		&version.UIConfig,
		&version.ChangeSummary,
		&version.ChangedBy,
		&version.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest block version: %w", err)
	}

	return version, nil
}
