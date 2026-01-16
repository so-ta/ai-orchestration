package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/souta/ai-orchestration/internal/domain"
)

// ProjectVersionRepository implements repository.ProjectVersionRepository
type ProjectVersionRepository struct {
	pool *pgxpool.Pool
}

// NewProjectVersionRepository creates a new ProjectVersionRepository
func NewProjectVersionRepository(pool *pgxpool.Pool) *ProjectVersionRepository {
	return &ProjectVersionRepository{pool: pool}
}

// Create creates a new project version snapshot
func (r *ProjectVersionRepository) Create(ctx context.Context, v *domain.ProjectVersion) error {
	query := `
		INSERT INTO project_versions (id, project_id, version, definition, saved_by, saved_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, query,
		v.ID, v.ProjectID, v.Version, v.Definition, v.SavedBy, v.SavedAt,
	)
	return err
}

// GetByProjectAndVersion retrieves a specific version of a project
func (r *ProjectVersionRepository) GetByProjectAndVersion(ctx context.Context, projectID uuid.UUID, version int) (*domain.ProjectVersion, error) {
	query := `
		SELECT id, project_id, version, definition, saved_by, saved_at
		FROM project_versions
		WHERE project_id = $1 AND version = $2
	`
	var v domain.ProjectVersion
	err := r.pool.QueryRow(ctx, query, projectID, version).Scan(
		&v.ID, &v.ProjectID, &v.Version, &v.Definition, &v.SavedBy, &v.SavedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrProjectVersionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// GetLatestByProject retrieves the latest version of a project
func (r *ProjectVersionRepository) GetLatestByProject(ctx context.Context, projectID uuid.UUID) (*domain.ProjectVersion, error) {
	query := `
		SELECT id, project_id, version, definition, saved_by, saved_at
		FROM project_versions
		WHERE project_id = $1
		ORDER BY version DESC
		LIMIT 1
	`
	var v domain.ProjectVersion
	err := r.pool.QueryRow(ctx, query, projectID).Scan(
		&v.ID, &v.ProjectID, &v.Version, &v.Definition, &v.SavedBy, &v.SavedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrProjectVersionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// ListByProject retrieves all versions of a project
func (r *ProjectVersionRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.ProjectVersion, error) {
	query := `
		SELECT id, project_id, version, definition, saved_by, saved_at
		FROM project_versions
		WHERE project_id = $1
		ORDER BY version DESC
	`
	rows, err := r.pool.Query(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []*domain.ProjectVersion
	for rows.Next() {
		var v domain.ProjectVersion
		if err := rows.Scan(
			&v.ID, &v.ProjectID, &v.Version, &v.Definition, &v.SavedBy, &v.SavedAt,
		); err != nil {
			return nil, err
		}
		versions = append(versions, &v)
	}

	return versions, nil
}
