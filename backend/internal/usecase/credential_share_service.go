package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// CredentialShareService handles credential sharing operations
type CredentialShareService struct {
	shareRepo      repository.CredentialShareRepository
	credentialRepo repository.CredentialRepository
}

// NewCredentialShareService creates a new CredentialShareService
func NewCredentialShareService(
	shareRepo repository.CredentialShareRepository,
	credentialRepo repository.CredentialRepository,
) *CredentialShareService {
	return &CredentialShareService{
		shareRepo:      shareRepo,
		credentialRepo: credentialRepo,
	}
}

// ShareWithUserInput contains input for sharing a credential with a user
type ShareWithUserInput struct {
	TenantID         uuid.UUID
	CredentialID     uuid.UUID
	SharedWithUserID uuid.UUID
	SharedByUserID   uuid.UUID
	Permission       domain.SharePermission
	Note             string
	ExpiresAt        *time.Time
}

// ShareWithUser shares a credential with a user
func (s *CredentialShareService) ShareWithUser(ctx context.Context, input ShareWithUserInput) (*domain.CredentialShare, error) {
	// Verify credential exists
	cred, err := s.credentialRepo.GetByID(ctx, input.TenantID, input.CredentialID)
	if err != nil {
		return nil, fmt.Errorf("get credential: %w", err)
	}

	// Check permission to share (must be owner or have admin permission)
	if err := s.checkSharePermission(ctx, cred, input.SharedByUserID); err != nil {
		return nil, err
	}

	// Check for existing share
	existing, err := s.shareRepo.GetByCredentialAndUser(ctx, input.CredentialID, input.SharedWithUserID)
	if err == nil && existing != nil {
		return nil, domain.ErrCredentialShareDuplicate
	}

	// Create share
	share := domain.NewCredentialShareWithUser(
		input.CredentialID,
		input.SharedWithUserID,
		input.SharedByUserID,
		input.Permission,
	)
	share.Note = input.Note
	share.ExpiresAt = input.ExpiresAt

	if err := share.Validate(); err != nil {
		return nil, err
	}

	if err := s.shareRepo.Create(ctx, share); err != nil {
		return nil, fmt.Errorf("create share: %w", err)
	}

	return share, nil
}

// ShareWithProjectInput contains input for sharing a credential with a project
type ShareWithProjectInput struct {
	TenantID            uuid.UUID
	CredentialID        uuid.UUID
	SharedWithProjectID uuid.UUID
	SharedByUserID      uuid.UUID
	Permission          domain.SharePermission
	Note                string
	ExpiresAt           *time.Time
}

// ShareWithProject shares a credential with a project
func (s *CredentialShareService) ShareWithProject(ctx context.Context, input ShareWithProjectInput) (*domain.CredentialShare, error) {
	// Verify credential exists
	cred, err := s.credentialRepo.GetByID(ctx, input.TenantID, input.CredentialID)
	if err != nil {
		return nil, fmt.Errorf("get credential: %w", err)
	}

	// Check permission to share
	if err := s.checkSharePermission(ctx, cred, input.SharedByUserID); err != nil {
		return nil, err
	}

	// Check for existing share
	existing, err := s.shareRepo.GetByCredentialAndProject(ctx, input.CredentialID, input.SharedWithProjectID)
	if err == nil && existing != nil {
		return nil, domain.ErrCredentialShareDuplicate
	}

	// Create share
	share := domain.NewCredentialShareWithProject(
		input.CredentialID,
		input.SharedWithProjectID,
		input.SharedByUserID,
		input.Permission,
	)
	share.Note = input.Note
	share.ExpiresAt = input.ExpiresAt

	if err := share.Validate(); err != nil {
		return nil, err
	}

	if err := s.shareRepo.Create(ctx, share); err != nil {
		return nil, fmt.Errorf("create share: %w", err)
	}

	return share, nil
}

// UpdateShareInput contains input for updating a credential share
type UpdateShareInput struct {
	TenantID   uuid.UUID
	ShareID    uuid.UUID
	UserID     uuid.UUID
	Permission *domain.SharePermission
	Note       *string
	ExpiresAt  *time.Time
}

// UpdateShare updates a credential share
func (s *CredentialShareService) UpdateShare(ctx context.Context, input UpdateShareInput) (*domain.CredentialShare, error) {
	// Get existing share
	share, err := s.shareRepo.GetByID(ctx, input.ShareID)
	if err != nil {
		return nil, err
	}

	// Verify credential exists and user has admin permission
	cred, err := s.credentialRepo.GetByID(ctx, input.TenantID, share.CredentialID)
	if err != nil {
		return nil, fmt.Errorf("get credential: %w", err)
	}

	if err := s.checkSharePermission(ctx, cred, input.UserID); err != nil {
		return nil, err
	}

	// Update fields
	if input.Permission != nil {
		share.Permission = *input.Permission
	}
	if input.Note != nil {
		share.Note = *input.Note
	}
	if input.ExpiresAt != nil {
		share.ExpiresAt = input.ExpiresAt
	}

	if err := share.Validate(); err != nil {
		return nil, err
	}

	if err := s.shareRepo.Update(ctx, share); err != nil {
		return nil, fmt.Errorf("update share: %w", err)
	}

	return share, nil
}

// RevokeShare revokes a credential share
func (s *CredentialShareService) RevokeShare(ctx context.Context, tenantID, shareID, userID uuid.UUID) error {
	// Get existing share
	share, err := s.shareRepo.GetByID(ctx, shareID)
	if err != nil {
		return err
	}

	// Verify credential exists and user has admin permission
	cred, err := s.credentialRepo.GetByID(ctx, tenantID, share.CredentialID)
	if err != nil {
		return fmt.Errorf("get credential: %w", err)
	}

	if err := s.checkSharePermission(ctx, cred, userID); err != nil {
		return err
	}

	return s.shareRepo.Delete(ctx, shareID)
}

// ListByCredential returns all shares for a credential
func (s *CredentialShareService) ListByCredential(ctx context.Context, tenantID, credentialID, userID uuid.UUID) ([]*domain.CredentialShare, error) {
	// Verify credential exists
	cred, err := s.credentialRepo.GetByID(ctx, tenantID, credentialID)
	if err != nil {
		return nil, fmt.Errorf("get credential: %w", err)
	}

	// Check permission to view shares (must be owner or have admin permission)
	if err := s.checkViewSharesPermission(ctx, cred, userID); err != nil {
		return nil, err
	}

	return s.shareRepo.ListByCredential(ctx, credentialID)
}

// ListByUser returns all shares for a user (credentials shared with them)
func (s *CredentialShareService) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.CredentialShare, error) {
	return s.shareRepo.ListByUser(ctx, userID)
}

// ListByProject returns all shares for a project
func (s *CredentialShareService) ListByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.CredentialShare, error) {
	return s.shareRepo.ListByProject(ctx, projectID)
}

// GetShareByID returns a share by ID
func (s *CredentialShareService) GetShareByID(ctx context.Context, shareID uuid.UUID) (*domain.CredentialShare, error) {
	return s.shareRepo.GetByID(ctx, shareID)
}

// CheckAccess checks if a user has access to a credential
func (s *CredentialShareService) CheckAccess(ctx context.Context, tenantID, credentialID, userID uuid.UUID, projectID *uuid.UUID) (*domain.SharePermission, error) {
	// Get credential
	cred, err := s.credentialRepo.GetByID(ctx, tenantID, credentialID)
	if err != nil {
		return nil, err
	}

	// Check if user is owner
	if s.isOwner(cred, userID) {
		adminPerm := domain.SharePermissionAdmin
		return &adminPerm, nil
	}

	// Check direct user share
	share, err := s.shareRepo.GetByCredentialAndUser(ctx, credentialID, userID)
	if err == nil && share != nil && !share.IsExpired() {
		return &share.Permission, nil
	}

	// Check project share if project ID provided
	if projectID != nil {
		share, err = s.shareRepo.GetByCredentialAndProject(ctx, credentialID, *projectID)
		if err == nil && share != nil && !share.IsExpired() {
			return &share.Permission, nil
		}
	}

	// Check organization-scope credential (available to all tenant users)
	if cred.Scope == domain.OwnerScopeOrganization {
		usePerm := domain.SharePermissionUse
		return &usePerm, nil
	}

	return nil, domain.ErrCredentialAccessDenied
}

// CleanupExpired removes expired shares
func (s *CredentialShareService) CleanupExpired(ctx context.Context) (int, error) {
	return s.shareRepo.DeleteExpired(ctx)
}

// Helper methods

// checkSharePermission verifies user can share a credential
func (s *CredentialShareService) checkSharePermission(ctx context.Context, cred *domain.Credential, userID uuid.UUID) error {
	// Owner can always share
	if s.isOwner(cred, userID) {
		return nil
	}

	// Check if user has admin permission via share
	share, err := s.shareRepo.GetByCredentialAndUser(ctx, cred.ID, userID)
	if err != nil {
		return domain.ErrCredentialAccessDenied
	}
	if share.IsExpired() || !share.Permission.CanAdmin() {
		return domain.ErrCredentialAccessDenied
	}

	return nil
}

// checkViewSharesPermission verifies user can view shares for a credential
func (s *CredentialShareService) checkViewSharesPermission(ctx context.Context, cred *domain.Credential, userID uuid.UUID) error {
	// Owner can always view shares
	if s.isOwner(cred, userID) {
		return nil
	}

	// Check if user has edit or admin permission via share
	share, err := s.shareRepo.GetByCredentialAndUser(ctx, cred.ID, userID)
	if err != nil {
		return domain.ErrCredentialAccessDenied
	}
	if share.IsExpired() || !share.Permission.CanView() {
		return domain.ErrCredentialAccessDenied
	}

	return nil
}

// isOwner checks if user owns the credential
func (s *CredentialShareService) isOwner(cred *domain.Credential, userID uuid.UUID) bool {
	// Personal credential - user must be owner
	if cred.Scope == domain.OwnerScopePersonal && cred.OwnerUserID != nil {
		return *cred.OwnerUserID == userID
	}
	// Organization or project scope - check via separate authorization
	// For now, allow org-scope credential owners based on role (handled at handler level)
	return false
}

// ============================================================================
// Response Types
// ============================================================================

// CredentialShareResponse is the API response for a credential share
type CredentialShareResponse struct {
	ID                  uuid.UUID              `json:"id"`
	CredentialID        uuid.UUID              `json:"credential_id"`
	SharedWithUserID    *uuid.UUID             `json:"shared_with_user_id,omitempty"`
	SharedWithProjectID *uuid.UUID             `json:"shared_with_project_id,omitempty"`
	Permission          domain.SharePermission `json:"permission"`
	SharedByUserID      uuid.UUID              `json:"shared_by_user_id"`
	Note                string                 `json:"note,omitempty"`
	CreatedAt           time.Time              `json:"created_at"`
	ExpiresAt           *time.Time             `json:"expires_at,omitempty"`
}

// ToShareResponse converts a domain share to API response
func ToShareResponse(share *domain.CredentialShare) CredentialShareResponse {
	return CredentialShareResponse{
		ID:                  share.ID,
		CredentialID:        share.CredentialID,
		SharedWithUserID:    share.SharedWithUserID,
		SharedWithProjectID: share.SharedWithProjectID,
		Permission:          share.Permission,
		SharedByUserID:      share.SharedByUserID,
		Note:                share.Note,
		CreatedAt:           share.CreatedAt,
		ExpiresAt:           share.ExpiresAt,
	}
}

// ToShareResponses converts domain shares to API responses
func ToShareResponses(shares []*domain.CredentialShare) []CredentialShareResponse {
	responses := make([]CredentialShareResponse, len(shares))
	for i, share := range shares {
		responses[i] = ToShareResponse(share)
	}
	return responses
}
