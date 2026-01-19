package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/souta/ai-orchestration/internal/domain"
)

// ProjectGitSyncRepository handles git sync persistence
type ProjectGitSyncRepository struct {
	db *pgxpool.Pool
}

// NewProjectGitSyncRepository creates a new ProjectGitSyncRepository
func NewProjectGitSyncRepository(db *pgxpool.Pool) *ProjectGitSyncRepository {
	return &ProjectGitSyncRepository{db: db}
}

// Create creates a new git sync configuration
func (r *ProjectGitSyncRepository) Create(ctx context.Context, gitSync *domain.ProjectGitSync) error {
	query := `
		INSERT INTO project_git_sync (
			id, tenant_id, project_id, repository_url, branch, file_path,
			sync_direction, auto_sync, last_sync_at, last_commit_sha,
			credentials_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := r.db.Exec(ctx, query,
		gitSync.ID,
		gitSync.TenantID,
		gitSync.ProjectID,
		gitSync.RepositoryURL,
		gitSync.Branch,
		gitSync.FilePath,
		gitSync.SyncDirection,
		gitSync.AutoSync,
		gitSync.LastSyncAt,
		gitSync.LastCommitSHA,
		gitSync.CredentialsID,
		gitSync.CreatedAt,
		gitSync.UpdatedAt,
	)

	return err
}

// GetByID retrieves a git sync configuration by ID
func (r *ProjectGitSyncRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ProjectGitSync, error) {
	query := `
		SELECT id, tenant_id, project_id, repository_url, branch, file_path,
		       sync_direction, auto_sync, last_sync_at, last_commit_sha,
		       credentials_id, created_at, updated_at
		FROM project_git_sync
		WHERE id = $1
	`

	gitSync := &domain.ProjectGitSync{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&gitSync.ID,
		&gitSync.TenantID,
		&gitSync.ProjectID,
		&gitSync.RepositoryURL,
		&gitSync.Branch,
		&gitSync.FilePath,
		&gitSync.SyncDirection,
		&gitSync.AutoSync,
		&gitSync.LastSyncAt,
		&gitSync.LastCommitSHA,
		&gitSync.CredentialsID,
		&gitSync.CreatedAt,
		&gitSync.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return gitSync, nil
}

// GetByProject retrieves a git sync configuration by project ID
func (r *ProjectGitSyncRepository) GetByProject(ctx context.Context, projectID uuid.UUID) (*domain.ProjectGitSync, error) {
	query := `
		SELECT id, tenant_id, project_id, repository_url, branch, file_path,
		       sync_direction, auto_sync, last_sync_at, last_commit_sha,
		       credentials_id, created_at, updated_at
		FROM project_git_sync
		WHERE project_id = $1
	`

	gitSync := &domain.ProjectGitSync{}
	err := r.db.QueryRow(ctx, query, projectID).Scan(
		&gitSync.ID,
		&gitSync.TenantID,
		&gitSync.ProjectID,
		&gitSync.RepositoryURL,
		&gitSync.Branch,
		&gitSync.FilePath,
		&gitSync.SyncDirection,
		&gitSync.AutoSync,
		&gitSync.LastSyncAt,
		&gitSync.LastCommitSHA,
		&gitSync.CredentialsID,
		&gitSync.CreatedAt,
		&gitSync.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return gitSync, nil
}

// ListByTenant retrieves all git sync configurations for a tenant
func (r *ProjectGitSyncRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.ProjectGitSync, error) {
	query := `
		SELECT id, tenant_id, project_id, repository_url, branch, file_path,
		       sync_direction, auto_sync, last_sync_at, last_commit_sha,
		       credentials_id, created_at, updated_at
		FROM project_git_sync
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gitSyncs []*domain.ProjectGitSync
	for rows.Next() {
		gitSync := &domain.ProjectGitSync{}
		err := rows.Scan(
			&gitSync.ID,
			&gitSync.TenantID,
			&gitSync.ProjectID,
			&gitSync.RepositoryURL,
			&gitSync.Branch,
			&gitSync.FilePath,
			&gitSync.SyncDirection,
			&gitSync.AutoSync,
			&gitSync.LastSyncAt,
			&gitSync.LastCommitSHA,
			&gitSync.CredentialsID,
			&gitSync.CreatedAt,
			&gitSync.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		gitSyncs = append(gitSyncs, gitSync)
	}

	return gitSyncs, rows.Err()
}

// Update updates a git sync configuration
func (r *ProjectGitSyncRepository) Update(ctx context.Context, gitSync *domain.ProjectGitSync) error {
	query := `
		UPDATE project_git_sync
		SET repository_url = $2, branch = $3, file_path = $4,
		    sync_direction = $5, auto_sync = $6, credentials_id = $7, updated_at = $8
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		gitSync.ID,
		gitSync.RepositoryURL,
		gitSync.Branch,
		gitSync.FilePath,
		gitSync.SyncDirection,
		gitSync.AutoSync,
		gitSync.CredentialsID,
		gitSync.UpdatedAt,
	)

	return err
}

// Delete deletes a git sync configuration
func (r *ProjectGitSyncRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM project_git_sync WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// UpdateLastSync updates the last sync timestamp and commit SHA
func (r *ProjectGitSyncRepository) UpdateLastSync(ctx context.Context, id uuid.UUID, commitSHA string) error {
	now := time.Now().UTC()
	query := `
		UPDATE project_git_sync
		SET last_sync_at = $2, last_commit_sha = $3, updated_at = $2
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id, now, commitSHA)
	return err
}
