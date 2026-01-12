package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/souta/ai-orchestration/internal/domain"
)

// BlockTemplateRepository implements repository.BlockTemplateRepository
type BlockTemplateRepository struct {
	pool *pgxpool.Pool
}

// NewBlockTemplateRepository creates a new BlockTemplateRepository
func NewBlockTemplateRepository(pool *pgxpool.Pool) *BlockTemplateRepository {
	return &BlockTemplateRepository{pool: pool}
}

// Create creates a new block template
func (r *BlockTemplateRepository) Create(ctx context.Context, template *domain.BlockTemplate) error {
	query := `
		INSERT INTO block_templates (
			id, slug, name, description, config_schema,
			executor_type, executor_code, is_builtin, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.pool.Exec(ctx, query,
		template.ID,
		template.Slug,
		template.Name,
		template.Description,
		template.ConfigSchema,
		template.ExecutorType,
		template.ExecutorCode,
		template.IsBuiltin,
		template.CreatedAt,
		template.UpdatedAt,
	)
	return err
}

// GetByID retrieves a block template by ID
func (r *BlockTemplateRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.BlockTemplate, error) {
	query := `
		SELECT id, slug, name, description, config_schema,
			executor_type, executor_code, is_builtin, created_at, updated_at
		FROM block_templates
		WHERE id = $1
	`

	template := &domain.BlockTemplate{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&template.ID,
		&template.Slug,
		&template.Name,
		&template.Description,
		&template.ConfigSchema,
		&template.ExecutorType,
		&template.ExecutorCode,
		&template.IsBuiltin,
		&template.CreatedAt,
		&template.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrBlockTemplateNotFound
	}
	if err != nil {
		return nil, err
	}
	return template, nil
}

// GetBySlug retrieves a block template by slug
func (r *BlockTemplateRepository) GetBySlug(ctx context.Context, slug string) (*domain.BlockTemplate, error) {
	query := `
		SELECT id, slug, name, description, config_schema,
			executor_type, executor_code, is_builtin, created_at, updated_at
		FROM block_templates
		WHERE slug = $1
	`

	template := &domain.BlockTemplate{}
	err := r.pool.QueryRow(ctx, query, slug).Scan(
		&template.ID,
		&template.Slug,
		&template.Name,
		&template.Description,
		&template.ConfigSchema,
		&template.ExecutorType,
		&template.ExecutorCode,
		&template.IsBuiltin,
		&template.CreatedAt,
		&template.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrBlockTemplateNotFound
	}
	if err != nil {
		return nil, err
	}
	return template, nil
}

// List retrieves all block templates
func (r *BlockTemplateRepository) List(ctx context.Context) ([]*domain.BlockTemplate, error) {
	query := `
		SELECT id, slug, name, description, config_schema,
			executor_type, executor_code, is_builtin, created_at, updated_at
		FROM block_templates
		ORDER BY is_builtin DESC, name
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*domain.BlockTemplate
	for rows.Next() {
		template := &domain.BlockTemplate{}
		err := rows.Scan(
			&template.ID,
			&template.Slug,
			&template.Name,
			&template.Description,
			&template.ConfigSchema,
			&template.ExecutorType,
			&template.ExecutorCode,
			&template.IsBuiltin,
			&template.CreatedAt,
			&template.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		templates = append(templates, template)
	}

	return templates, rows.Err()
}

// Update updates a block template (only non-builtin templates)
func (r *BlockTemplateRepository) Update(ctx context.Context, template *domain.BlockTemplate) error {
	// First check if it's a builtin template
	existing, err := r.GetByID(ctx, template.ID)
	if err != nil {
		return err
	}
	if existing.IsBuiltin {
		return domain.ErrBlockTemplateIsBuiltin
	}

	query := `
		UPDATE block_templates
		SET slug = $2, name = $3, description = $4, config_schema = $5,
			executor_type = $6, executor_code = $7, updated_at = $8
		WHERE id = $1 AND is_builtin = false
	`

	result, err := r.pool.Exec(ctx, query,
		template.ID,
		template.Slug,
		template.Name,
		template.Description,
		template.ConfigSchema,
		template.ExecutorType,
		template.ExecutorCode,
		template.UpdatedAt,
	)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrBlockTemplateNotFound
	}
	return nil
}

// Delete deletes a block template (only non-builtin templates)
func (r *BlockTemplateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// First check if it's a builtin template
	existing, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing.IsBuiltin {
		return domain.ErrBlockTemplateIsBuiltin
	}

	query := `DELETE FROM block_templates WHERE id = $1 AND is_builtin = false`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrBlockTemplateNotFound
	}
	return nil
}
