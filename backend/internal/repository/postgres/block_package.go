package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// CustomBlockPackageRepository handles custom block package persistence
type CustomBlockPackageRepository struct {
	db *pgxpool.Pool
}

// NewCustomBlockPackageRepository creates a new CustomBlockPackageRepository
func NewCustomBlockPackageRepository(db *pgxpool.Pool) *CustomBlockPackageRepository {
	return &CustomBlockPackageRepository{db: db}
}

// Create creates a new custom block package
func (r *CustomBlockPackageRepository) Create(ctx context.Context, pkg *domain.CustomBlockPackage) error {
	query := `
		INSERT INTO custom_block_packages (
			id, tenant_id, name, version, description, bundle_url,
			blocks, dependencies, status, created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := r.db.Exec(ctx, query,
		pkg.ID,
		pkg.TenantID,
		pkg.Name,
		pkg.Version,
		pkg.Description,
		pkg.BundleURL,
		pkg.Blocks,
		pkg.Dependencies,
		pkg.Status,
		pkg.CreatedBy,
		pkg.CreatedAt,
		pkg.UpdatedAt,
	)

	return err
}

// GetByID retrieves a custom block package by ID
func (r *CustomBlockPackageRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.CustomBlockPackage, error) {
	query := `
		SELECT id, tenant_id, name, version, description, bundle_url,
		       blocks, dependencies, status, created_by, created_at, updated_at
		FROM custom_block_packages
		WHERE id = $1
	`

	pkg := &domain.CustomBlockPackage{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&pkg.ID,
		&pkg.TenantID,
		&pkg.Name,
		&pkg.Version,
		&pkg.Description,
		&pkg.BundleURL,
		&pkg.Blocks,
		&pkg.Dependencies,
		&pkg.Status,
		&pkg.CreatedBy,
		&pkg.CreatedAt,
		&pkg.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return pkg, nil
}

// GetByNameAndVersion retrieves a custom block package by name and version
func (r *CustomBlockPackageRepository) GetByNameAndVersion(ctx context.Context, tenantID uuid.UUID, name, version string) (*domain.CustomBlockPackage, error) {
	query := `
		SELECT id, tenant_id, name, version, description, bundle_url,
		       blocks, dependencies, status, created_by, created_at, updated_at
		FROM custom_block_packages
		WHERE tenant_id = $1 AND name = $2 AND version = $3
	`

	pkg := &domain.CustomBlockPackage{}
	err := r.db.QueryRow(ctx, query, tenantID, name, version).Scan(
		&pkg.ID,
		&pkg.TenantID,
		&pkg.Name,
		&pkg.Version,
		&pkg.Description,
		&pkg.BundleURL,
		&pkg.Blocks,
		&pkg.Dependencies,
		&pkg.Status,
		&pkg.CreatedBy,
		&pkg.CreatedAt,
		&pkg.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return pkg, nil
}

// ListByTenant retrieves custom block packages for a tenant
func (r *CustomBlockPackageRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, filter repository.BlockPackageFilter) ([]*domain.CustomBlockPackage, int, error) {
	// Build WHERE clause
	conditions := []string{"tenant_id = $1"}
	args := []interface{}{tenantID}
	argIndex := 2

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *filter.Status)
		argIndex++
	}

	if filter.Search != nil && *filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+*filter.Search+"%")
		argIndex++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	// Count query
	countQuery := "SELECT COUNT(*) FROM custom_block_packages " + whereClause
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// List query
	query := `
		SELECT id, tenant_id, name, version, description, bundle_url,
		       blocks, dependencies, status, created_by, created_at, updated_at
		FROM custom_block_packages
	` + whereClause + " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}

	if filter.Page > 0 && filter.Limit > 0 {
		offset := (filter.Page - 1) * filter.Limit
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, offset)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var packages []*domain.CustomBlockPackage
	for rows.Next() {
		pkg := &domain.CustomBlockPackage{}
		err := rows.Scan(
			&pkg.ID,
			&pkg.TenantID,
			&pkg.Name,
			&pkg.Version,
			&pkg.Description,
			&pkg.BundleURL,
			&pkg.Blocks,
			&pkg.Dependencies,
			&pkg.Status,
			&pkg.CreatedBy,
			&pkg.CreatedAt,
			&pkg.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		packages = append(packages, pkg)
	}

	return packages, total, rows.Err()
}

// Update updates a custom block package
func (r *CustomBlockPackageRepository) Update(ctx context.Context, pkg *domain.CustomBlockPackage) error {
	query := `
		UPDATE custom_block_packages
		SET name = $2, version = $3, description = $4, bundle_url = $5,
		    blocks = $6, dependencies = $7, status = $8, updated_at = $9
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		pkg.ID,
		pkg.Name,
		pkg.Version,
		pkg.Description,
		pkg.BundleURL,
		pkg.Blocks,
		pkg.Dependencies,
		pkg.Status,
		pkg.UpdatedAt,
	)

	return err
}

// Delete deletes a custom block package
func (r *CustomBlockPackageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM custom_block_packages WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// Publish marks a package as published
func (r *CustomBlockPackageRepository) Publish(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	query := `UPDATE custom_block_packages SET status = $2, updated_at = $3 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id, domain.BlockPackageStatusPublished, now)
	return err
}

// Deprecate marks a package as deprecated
func (r *CustomBlockPackageRepository) Deprecate(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	query := `UPDATE custom_block_packages SET status = $2, updated_at = $3 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id, domain.BlockPackageStatusDeprecated, now)
	return err
}
