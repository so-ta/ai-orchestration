package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// GitSyncUsecase handles git sync business logic
type GitSyncUsecase struct {
	gitSyncRepo repository.ProjectGitSyncRepository
	projectRepo repository.ProjectRepository
}

// NewGitSyncUsecase creates a new GitSyncUsecase
func NewGitSyncUsecase(
	gitSyncRepo repository.ProjectGitSyncRepository,
	projectRepo repository.ProjectRepository,
) *GitSyncUsecase {
	return &GitSyncUsecase{
		gitSyncRepo: gitSyncRepo,
		projectRepo: projectRepo,
	}
}

// CreateGitSyncInput represents input for creating a git sync configuration
type CreateGitSyncInput struct {
	TenantID      uuid.UUID
	ProjectID     uuid.UUID
	RepositoryURL string
	Branch        string
	FilePath      string
	SyncDirection domain.GitSyncDirection
	AutoSync      bool
	CredentialsID *uuid.UUID
}

// Create creates a new git sync configuration
func (u *GitSyncUsecase) Create(ctx context.Context, input CreateGitSyncInput) (*domain.ProjectGitSync, error) {
	// Verify project exists and belongs to tenant
	project, err := u.projectRepo.GetByID(ctx, input.TenantID, input.ProjectID)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, domain.ErrProjectNotFound
	}

	// Check if git sync already exists for this project
	existing, _ := u.gitSyncRepo.GetByProject(ctx, input.ProjectID)
	if existing != nil {
		return nil, domain.NewValidationError("project_id", "git sync configuration already exists for this project")
	}

	// Validate input
	if input.RepositoryURL == "" {
		return nil, domain.NewValidationError("repository_url", "repository URL is required")
	}

	gitSync := domain.NewProjectGitSync(input.TenantID, input.ProjectID, input.RepositoryURL)

	if input.Branch != "" {
		gitSync.Branch = input.Branch
	}
	if input.FilePath != "" {
		gitSync.FilePath = input.FilePath
	}
	if input.SyncDirection != "" {
		gitSync.SyncDirection = input.SyncDirection
	}
	gitSync.AutoSync = input.AutoSync
	if input.CredentialsID != nil {
		gitSync.SetCredentials(*input.CredentialsID)
	}

	if err := u.gitSyncRepo.Create(ctx, gitSync); err != nil {
		return nil, err
	}

	return gitSync, nil
}

// GetByID retrieves a git sync configuration by ID
func (u *GitSyncUsecase) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.ProjectGitSync, error) {
	gitSync, err := u.gitSyncRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Verify tenant access
	if gitSync.TenantID != tenantID {
		return nil, domain.ErrForbidden
	}

	return gitSync, nil
}

// GetByProject retrieves a git sync configuration by project ID
func (u *GitSyncUsecase) GetByProject(ctx context.Context, tenantID, projectID uuid.UUID) (*domain.ProjectGitSync, error) {
	// Verify project exists and belongs to tenant
	project, err := u.projectRepo.GetByID(ctx, tenantID, projectID)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, domain.ErrProjectNotFound
	}

	return u.gitSyncRepo.GetByProject(ctx, projectID)
}

// ListByTenant retrieves all git sync configurations for a tenant
func (u *GitSyncUsecase) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.ProjectGitSync, error) {
	return u.gitSyncRepo.ListByTenant(ctx, tenantID)
}

// UpdateGitSyncInput represents input for updating a git sync configuration
type UpdateGitSyncInput struct {
	ID            uuid.UUID
	TenantID      uuid.UUID
	RepositoryURL *string
	Branch        *string
	FilePath      *string
	SyncDirection *domain.GitSyncDirection
	AutoSync      *bool
	CredentialsID *uuid.UUID
}

// Update updates a git sync configuration
func (u *GitSyncUsecase) Update(ctx context.Context, input UpdateGitSyncInput) (*domain.ProjectGitSync, error) {
	gitSync, err := u.gitSyncRepo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	// Verify tenant access
	if gitSync.TenantID != input.TenantID {
		return nil, domain.ErrForbidden
	}

	if input.RepositoryURL != nil {
		gitSync.RepositoryURL = *input.RepositoryURL
	}
	if input.Branch != nil {
		gitSync.Branch = *input.Branch
	}
	if input.FilePath != nil {
		gitSync.FilePath = *input.FilePath
	}
	if input.SyncDirection != nil {
		gitSync.SyncDirection = *input.SyncDirection
	}
	if input.AutoSync != nil {
		gitSync.AutoSync = *input.AutoSync
	}
	if input.CredentialsID != nil {
		gitSync.SetCredentials(*input.CredentialsID)
	}

	if err := u.gitSyncRepo.Update(ctx, gitSync); err != nil {
		return nil, err
	}

	return gitSync, nil
}

// Delete deletes a git sync configuration
func (u *GitSyncUsecase) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	gitSync, err := u.gitSyncRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Verify tenant access
	if gitSync.TenantID != tenantID {
		return domain.ErrForbidden
	}

	return u.gitSyncRepo.Delete(ctx, id)
}

// TriggerSync triggers a manual sync operation
// This is a placeholder - actual implementation would involve git operations
func (u *GitSyncUsecase) TriggerSync(ctx context.Context, tenantID, id uuid.UUID, operation string) (*domain.GitSyncOperation, error) {
	gitSync, err := u.gitSyncRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Verify tenant access
	if gitSync.TenantID != tenantID {
		return nil, domain.ErrForbidden
	}

	// Validate operation
	if operation != "push" && operation != "pull" {
		return nil, domain.NewValidationError("operation", "operation must be 'push' or 'pull'")
	}

	// Check sync direction compatibility
	switch gitSync.SyncDirection {
	case domain.GitSyncDirectionPush:
		if operation != "push" {
			return nil, domain.NewValidationError("operation", "this configuration only supports push")
		}
	case domain.GitSyncDirectionPull:
		if operation != "pull" {
			return nil, domain.NewValidationError("operation", "this configuration only supports pull")
		}
	}

	// Create sync operation (placeholder - actual implementation would queue the operation)
	syncOp := &domain.GitSyncOperation{
		ID:        uuid.New(),
		GitSyncID: id,
		Operation: operation,
		Status:    "pending",
	}

	// In a real implementation, this would:
	// 1. Queue the sync operation
	// 2. Worker would pick it up and execute git operations
	// 3. Update last_sync_at and last_commit_sha on success

	return syncOp, nil
}
