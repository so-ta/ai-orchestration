package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// ============================================================================
// Mock Credential Share Repository
// ============================================================================

type mockCredentialShareRepo struct {
	shares              map[uuid.UUID]*domain.CredentialShare
	byCredentialAndUser map[string]*domain.CredentialShare
	byCredentialAndProj map[string]*domain.CredentialShare
	createErr           error
	getErr              error
}

func newMockCredentialShareRepo() *mockCredentialShareRepo {
	return &mockCredentialShareRepo{
		shares:              make(map[uuid.UUID]*domain.CredentialShare),
		byCredentialAndUser: make(map[string]*domain.CredentialShare),
		byCredentialAndProj: make(map[string]*domain.CredentialShare),
	}
}

func (m *mockCredentialShareRepo) addShare(s *domain.CredentialShare) {
	m.shares[s.ID] = s
	if s.SharedWithUserID != nil {
		key := s.CredentialID.String() + "-" + s.SharedWithUserID.String()
		m.byCredentialAndUser[key] = s
	}
	if s.SharedWithProjectID != nil {
		key := s.CredentialID.String() + "-" + s.SharedWithProjectID.String()
		m.byCredentialAndProj[key] = s
	}
}

func (m *mockCredentialShareRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.CredentialShare, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	s, ok := m.shares[id]
	if !ok {
		return nil, domain.ErrCredentialShareNotFound
	}
	return s, nil
}

func (m *mockCredentialShareRepo) ListByCredential(ctx context.Context, credentialID uuid.UUID) ([]*domain.CredentialShare, error) {
	var result []*domain.CredentialShare
	for _, s := range m.shares {
		if s.CredentialID == credentialID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockCredentialShareRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.CredentialShare, error) {
	var result []*domain.CredentialShare
	for _, s := range m.shares {
		if s.SharedWithUserID != nil && *s.SharedWithUserID == userID {
			if s.ExpiresAt == nil || time.Now().Before(*s.ExpiresAt) {
				result = append(result, s)
			}
		}
	}
	return result, nil
}

func (m *mockCredentialShareRepo) ListByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.CredentialShare, error) {
	var result []*domain.CredentialShare
	for _, s := range m.shares {
		if s.SharedWithProjectID != nil && *s.SharedWithProjectID == projectID {
			if s.ExpiresAt == nil || time.Now().Before(*s.ExpiresAt) {
				result = append(result, s)
			}
		}
	}
	return result, nil
}

func (m *mockCredentialShareRepo) GetByCredentialAndUser(ctx context.Context, credentialID, userID uuid.UUID) (*domain.CredentialShare, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	key := credentialID.String() + "-" + userID.String()
	s, ok := m.byCredentialAndUser[key]
	if !ok {
		return nil, domain.ErrCredentialShareNotFound
	}
	return s, nil
}

func (m *mockCredentialShareRepo) GetByCredentialAndProject(ctx context.Context, credentialID, projectID uuid.UUID) (*domain.CredentialShare, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	key := credentialID.String() + "-" + projectID.String()
	s, ok := m.byCredentialAndProj[key]
	if !ok {
		return nil, domain.ErrCredentialShareNotFound
	}
	return s, nil
}

func (m *mockCredentialShareRepo) Create(ctx context.Context, share *domain.CredentialShare) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.addShare(share)
	return nil
}

func (m *mockCredentialShareRepo) Update(ctx context.Context, share *domain.CredentialShare) error {
	m.shares[share.ID] = share
	return nil
}

func (m *mockCredentialShareRepo) Delete(ctx context.Context, id uuid.UUID) error {
	s, ok := m.shares[id]
	if !ok {
		return domain.ErrCredentialShareNotFound
	}
	delete(m.shares, id)
	if s.SharedWithUserID != nil {
		key := s.CredentialID.String() + "-" + s.SharedWithUserID.String()
		delete(m.byCredentialAndUser, key)
	}
	if s.SharedWithProjectID != nil {
		key := s.CredentialID.String() + "-" + s.SharedWithProjectID.String()
		delete(m.byCredentialAndProj, key)
	}
	return nil
}

func (m *mockCredentialShareRepo) DeleteExpired(ctx context.Context) (int, error) {
	count := 0
	for id, s := range m.shares {
		if s.ExpiresAt != nil && time.Now().After(*s.ExpiresAt) {
			delete(m.shares, id)
			count++
		}
	}
	return count, nil
}

// ============================================================================
// Mock Credential Repository for Share Service Tests
// ============================================================================

type mockCredentialRepoForShare struct {
	credentials map[uuid.UUID]*domain.Credential
	credsByName map[string]*domain.Credential
	createErr   error
	getErr      error
}

func newMockCredentialRepoForShare() *mockCredentialRepoForShare {
	return &mockCredentialRepoForShare{
		credentials: make(map[uuid.UUID]*domain.Credential),
		credsByName: make(map[string]*domain.Credential),
	}
}

func (m *mockCredentialRepoForShare) addCredential(cred *domain.Credential) {
	m.credentials[cred.ID] = cred
	key := cred.TenantID.String() + "-" + cred.Name
	m.credsByName[key] = cred
}

func (m *mockCredentialRepoForShare) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Credential, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	cred, ok := m.credentials[id]
	if !ok {
		return nil, domain.ErrCredentialNotFound
	}
	if cred.TenantID != tenantID {
		return nil, domain.ErrCredentialNotFound
	}
	return cred, nil
}

func (m *mockCredentialRepoForShare) GetByName(ctx context.Context, tenantID uuid.UUID, name string) (*domain.Credential, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	key := tenantID.String() + "-" + name
	cred, ok := m.credsByName[key]
	if !ok {
		return nil, domain.ErrCredentialNotFound
	}
	return cred, nil
}

func (m *mockCredentialRepoForShare) List(ctx context.Context, tenantID uuid.UUID, filter repository.CredentialFilter) ([]*domain.Credential, int, error) {
	var result []*domain.Credential
	for _, cred := range m.credentials {
		if cred.TenantID == tenantID {
			result = append(result, cred)
		}
	}
	return result, len(result), nil
}

func (m *mockCredentialRepoForShare) Create(ctx context.Context, cred *domain.Credential) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.addCredential(cred)
	return nil
}

func (m *mockCredentialRepoForShare) Update(ctx context.Context, cred *domain.Credential) error {
	m.credentials[cred.ID] = cred
	key := cred.TenantID.String() + "-" + cred.Name
	m.credsByName[key] = cred
	return nil
}

func (m *mockCredentialRepoForShare) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	cred, ok := m.credentials[id]
	if !ok {
		return domain.ErrCredentialNotFound
	}
	if cred.TenantID != tenantID {
		return domain.ErrCredentialNotFound
	}
	key := cred.TenantID.String() + "-" + cred.Name
	delete(m.credsByName, key)
	delete(m.credentials, id)
	return nil
}

func (m *mockCredentialRepoForShare) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.CredentialStatus) error {
	cred, ok := m.credentials[id]
	if !ok {
		return domain.ErrCredentialNotFound
	}
	if cred.TenantID != tenantID {
		return domain.ErrCredentialNotFound
	}
	cred.Status = status
	return nil
}

// ============================================================================
// Test Helpers for Share Service
// ============================================================================

func createTestCredential(tenantID uuid.UUID, scope domain.OwnerScope, ownerUserID *uuid.UUID) *domain.Credential {
	cred := domain.NewCredential(tenantID, "Test Credential", domain.CredentialTypeAPIKey)
	cred.Scope = scope
	cred.OwnerUserID = ownerUserID
	return cred
}

func createShareService() (*CredentialShareService, *mockCredentialShareRepo, *mockCredentialRepoForShare) {
	shareRepo := newMockCredentialShareRepo()
	credRepo := newMockCredentialRepoForShare()
	service := NewCredentialShareService(shareRepo, credRepo)
	return service, shareRepo, credRepo
}

// ============================================================================
// CredentialShareService Tests
// ============================================================================

func TestCredentialShareService_ShareWithUser(t *testing.T) {
	service, _, credRepo := createShareService()

	tenantID := uuid.New()
	ownerUserID := uuid.New()
	targetUserID := uuid.New()

	// Create a personal credential owned by ownerUserID
	cred := createTestCredential(tenantID, domain.OwnerScopePersonal, &ownerUserID)
	credRepo.addCredential(cred)

	t.Run("success - owner can share", func(t *testing.T) {
		share, err := service.ShareWithUser(context.Background(), ShareWithUserInput{
			TenantID:         tenantID,
			CredentialID:     cred.ID,
			SharedWithUserID: targetUserID,
			SharedByUserID:   ownerUserID,
			Permission:       domain.SharePermissionUse,
			Note:             "Shared for testing",
		})
		if err != nil {
			t.Fatalf("ShareWithUser() error = %v", err)
		}
		if share.CredentialID != cred.ID {
			t.Errorf("ShareWithUser() credentialID = %s, want %s", share.CredentialID, cred.ID)
		}
		if *share.SharedWithUserID != targetUserID {
			t.Errorf("ShareWithUser() sharedWithUserID = %s, want %s", *share.SharedWithUserID, targetUserID)
		}
		if share.Permission != domain.SharePermissionUse {
			t.Errorf("ShareWithUser() permission = %s, want use", share.Permission)
		}
	})

	t.Run("duplicate share error", func(t *testing.T) {
		// Try to share again with same user
		_, err := service.ShareWithUser(context.Background(), ShareWithUserInput{
			TenantID:         tenantID,
			CredentialID:     cred.ID,
			SharedWithUserID: targetUserID,
			SharedByUserID:   ownerUserID,
			Permission:       domain.SharePermissionEdit,
		})
		if err != domain.ErrCredentialShareDuplicate {
			t.Errorf("ShareWithUser() duplicate error = %v, want ErrCredentialShareDuplicate", err)
		}
	})

	t.Run("credential not found", func(t *testing.T) {
		_, err := service.ShareWithUser(context.Background(), ShareWithUserInput{
			TenantID:         tenantID,
			CredentialID:     uuid.New(), // Non-existent
			SharedWithUserID: targetUserID,
			SharedByUserID:   ownerUserID,
			Permission:       domain.SharePermissionUse,
		})
		if err == nil {
			t.Error("ShareWithUser() expected error for non-existent credential")
		}
	})

	t.Run("share with expiration", func(t *testing.T) {
		newTargetUserID := uuid.New()
		expiresAt := time.Now().Add(24 * time.Hour)

		share, err := service.ShareWithUser(context.Background(), ShareWithUserInput{
			TenantID:         tenantID,
			CredentialID:     cred.ID,
			SharedWithUserID: newTargetUserID,
			SharedByUserID:   ownerUserID,
			Permission:       domain.SharePermissionEdit,
			ExpiresAt:        &expiresAt,
		})
		if err != nil {
			t.Fatalf("ShareWithUser() error = %v", err)
		}
		if share.ExpiresAt == nil {
			t.Error("ShareWithUser() expiresAt should not be nil")
		}
	})
}

func TestCredentialShareService_ShareWithProject(t *testing.T) {
	service, shareRepo, credRepo := createShareService()

	tenantID := uuid.New()
	ownerUserID := uuid.New()
	projectID := uuid.New()

	cred := createTestCredential(tenantID, domain.OwnerScopePersonal, &ownerUserID)
	credRepo.addCredential(cred)

	t.Run("success", func(t *testing.T) {
		share, err := service.ShareWithProject(context.Background(), ShareWithProjectInput{
			TenantID:            tenantID,
			CredentialID:        cred.ID,
			SharedWithProjectID: projectID,
			SharedByUserID:      ownerUserID,
			Permission:          domain.SharePermissionUse,
		})
		if err != nil {
			t.Fatalf("ShareWithProject() error = %v", err)
		}
		if share.SharedWithProjectID == nil || *share.SharedWithProjectID != projectID {
			t.Errorf("ShareWithProject() sharedWithProjectID incorrect")
		}
	})

	t.Run("duplicate error", func(t *testing.T) {
		_, err := service.ShareWithProject(context.Background(), ShareWithProjectInput{
			TenantID:            tenantID,
			CredentialID:        cred.ID,
			SharedWithProjectID: projectID,
			SharedByUserID:      ownerUserID,
			Permission:          domain.SharePermissionEdit,
		})
		if err != domain.ErrCredentialShareDuplicate {
			t.Errorf("ShareWithProject() duplicate error = %v, want ErrCredentialShareDuplicate", err)
		}
	})

	_ = shareRepo // Avoid unused variable warning
}

func TestCredentialShareService_UpdateShare(t *testing.T) {
	service, shareRepo, credRepo := createShareService()

	tenantID := uuid.New()
	ownerUserID := uuid.New()
	targetUserID := uuid.New()

	cred := createTestCredential(tenantID, domain.OwnerScopePersonal, &ownerUserID)
	credRepo.addCredential(cred)

	// Create initial share
	share := domain.NewCredentialShareWithUser(cred.ID, targetUserID, ownerUserID, domain.SharePermissionUse)
	shareRepo.addShare(share)

	t.Run("update permission", func(t *testing.T) {
		newPermission := domain.SharePermissionAdmin
		updated, err := service.UpdateShare(context.Background(), UpdateShareInput{
			TenantID:   tenantID,
			ShareID:    share.ID,
			UserID:     ownerUserID,
			Permission: &newPermission,
		})
		if err != nil {
			t.Fatalf("UpdateShare() error = %v", err)
		}
		if updated.Permission != domain.SharePermissionAdmin {
			t.Errorf("UpdateShare() permission = %s, want admin", updated.Permission)
		}
	})

	t.Run("update note", func(t *testing.T) {
		newNote := "Updated note"
		updated, err := service.UpdateShare(context.Background(), UpdateShareInput{
			TenantID: tenantID,
			ShareID:  share.ID,
			UserID:   ownerUserID,
			Note:     &newNote,
		})
		if err != nil {
			t.Fatalf("UpdateShare() error = %v", err)
		}
		if updated.Note != newNote {
			t.Errorf("UpdateShare() note = %s, want %s", updated.Note, newNote)
		}
	})

	t.Run("share not found", func(t *testing.T) {
		newPermission := domain.SharePermissionEdit
		_, err := service.UpdateShare(context.Background(), UpdateShareInput{
			TenantID:   tenantID,
			ShareID:    uuid.New(),
			UserID:     ownerUserID,
			Permission: &newPermission,
		})
		if err != domain.ErrCredentialShareNotFound {
			t.Errorf("UpdateShare() error = %v, want ErrCredentialShareNotFound", err)
		}
	})
}

func TestCredentialShareService_RevokeShare(t *testing.T) {
	service, shareRepo, credRepo := createShareService()

	tenantID := uuid.New()
	ownerUserID := uuid.New()
	targetUserID := uuid.New()

	cred := createTestCredential(tenantID, domain.OwnerScopePersonal, &ownerUserID)
	credRepo.addCredential(cred)

	share := domain.NewCredentialShareWithUser(cred.ID, targetUserID, ownerUserID, domain.SharePermissionUse)
	shareRepo.addShare(share)

	t.Run("success", func(t *testing.T) {
		err := service.RevokeShare(context.Background(), tenantID, share.ID, ownerUserID)
		if err != nil {
			t.Fatalf("RevokeShare() error = %v", err)
		}

		// Verify deleted
		_, err = service.GetShareByID(context.Background(), share.ID)
		if err != domain.ErrCredentialShareNotFound {
			t.Errorf("GetShareByID() after revoke should return not found")
		}
	})

	t.Run("not found", func(t *testing.T) {
		err := service.RevokeShare(context.Background(), tenantID, uuid.New(), ownerUserID)
		if err != domain.ErrCredentialShareNotFound {
			t.Errorf("RevokeShare() error = %v, want ErrCredentialShareNotFound", err)
		}
	})
}

func TestCredentialShareService_ListByCredential(t *testing.T) {
	service, shareRepo, credRepo := createShareService()

	tenantID := uuid.New()
	ownerUserID := uuid.New()

	cred := createTestCredential(tenantID, domain.OwnerScopePersonal, &ownerUserID)
	credRepo.addCredential(cred)

	// Create multiple shares
	share1 := domain.NewCredentialShareWithUser(cred.ID, uuid.New(), ownerUserID, domain.SharePermissionUse)
	share2 := domain.NewCredentialShareWithUser(cred.ID, uuid.New(), ownerUserID, domain.SharePermissionEdit)
	share3 := domain.NewCredentialShareWithProject(cred.ID, uuid.New(), ownerUserID, domain.SharePermissionAdmin)
	shareRepo.addShare(share1)
	shareRepo.addShare(share2)
	shareRepo.addShare(share3)

	t.Run("returns all shares for credential", func(t *testing.T) {
		shares, err := service.ListByCredential(context.Background(), tenantID, cred.ID, ownerUserID)
		if err != nil {
			t.Fatalf("ListByCredential() error = %v", err)
		}
		if len(shares) != 3 {
			t.Errorf("ListByCredential() got %d shares, want 3", len(shares))
		}
	})
}

func TestCredentialShareService_ListByUser(t *testing.T) {
	service, shareRepo, credRepo := createShareService()

	tenantID := uuid.New()
	ownerUserID := uuid.New()
	targetUserID := uuid.New()

	cred1 := createTestCredential(tenantID, domain.OwnerScopePersonal, &ownerUserID)
	cred2 := createTestCredential(tenantID, domain.OwnerScopePersonal, &ownerUserID)
	credRepo.addCredential(cred1)
	credRepo.addCredential(cred2)

	share1 := domain.NewCredentialShareWithUser(cred1.ID, targetUserID, ownerUserID, domain.SharePermissionUse)
	share2 := domain.NewCredentialShareWithUser(cred2.ID, targetUserID, ownerUserID, domain.SharePermissionEdit)
	shareRepo.addShare(share1)
	shareRepo.addShare(share2)

	t.Run("returns shares for user", func(t *testing.T) {
		shares, err := service.ListByUser(context.Background(), targetUserID)
		if err != nil {
			t.Fatalf("ListByUser() error = %v", err)
		}
		if len(shares) != 2 {
			t.Errorf("ListByUser() got %d shares, want 2", len(shares))
		}
	})

	t.Run("excludes expired shares", func(t *testing.T) {
		// Create expired share
		expiredTime := time.Now().Add(-time.Hour)
		expiredShare := domain.NewCredentialShareWithUser(cred1.ID, uuid.New(), ownerUserID, domain.SharePermissionUse)
		expiredShare.ExpiresAt = &expiredTime
		shareRepo.addShare(expiredShare)

		shares, err := service.ListByUser(context.Background(), *expiredShare.SharedWithUserID)
		if err != nil {
			t.Fatalf("ListByUser() error = %v", err)
		}
		if len(shares) != 0 {
			t.Errorf("ListByUser() should exclude expired shares, got %d", len(shares))
		}
	})
}

func TestCredentialShareService_ListByProject(t *testing.T) {
	service, shareRepo, credRepo := createShareService()

	tenantID := uuid.New()
	ownerUserID := uuid.New()
	projectID := uuid.New()

	cred1 := createTestCredential(tenantID, domain.OwnerScopePersonal, &ownerUserID)
	cred2 := createTestCredential(tenantID, domain.OwnerScopePersonal, &ownerUserID)
	credRepo.addCredential(cred1)
	credRepo.addCredential(cred2)

	share1 := domain.NewCredentialShareWithProject(cred1.ID, projectID, ownerUserID, domain.SharePermissionUse)
	share2 := domain.NewCredentialShareWithProject(cred2.ID, projectID, ownerUserID, domain.SharePermissionEdit)
	shareRepo.addShare(share1)
	shareRepo.addShare(share2)

	t.Run("returns shares for project", func(t *testing.T) {
		shares, err := service.ListByProject(context.Background(), projectID)
		if err != nil {
			t.Fatalf("ListByProject() error = %v", err)
		}
		if len(shares) != 2 {
			t.Errorf("ListByProject() got %d shares, want 2", len(shares))
		}
	})
}

func TestCredentialShareService_CleanupExpired(t *testing.T) {
	service, shareRepo, credRepo := createShareService()

	tenantID := uuid.New()
	ownerUserID := uuid.New()

	cred := createTestCredential(tenantID, domain.OwnerScopePersonal, &ownerUserID)
	credRepo.addCredential(cred)

	// Create active and expired shares
	activeShare := domain.NewCredentialShareWithUser(cred.ID, uuid.New(), ownerUserID, domain.SharePermissionUse)
	shareRepo.addShare(activeShare)

	expiredTime := time.Now().Add(-time.Hour)
	expiredShare1 := domain.NewCredentialShareWithUser(cred.ID, uuid.New(), ownerUserID, domain.SharePermissionUse)
	expiredShare1.ExpiresAt = &expiredTime
	expiredShare2 := domain.NewCredentialShareWithUser(cred.ID, uuid.New(), ownerUserID, domain.SharePermissionUse)
	expiredShare2.ExpiresAt = &expiredTime
	shareRepo.addShare(expiredShare1)
	shareRepo.addShare(expiredShare2)

	t.Run("removes expired shares", func(t *testing.T) {
		count, err := service.CleanupExpired(context.Background())
		if err != nil {
			t.Fatalf("CleanupExpired() error = %v", err)
		}
		if count != 2 {
			t.Errorf("CleanupExpired() count = %d, want 2", count)
		}

		// Verify active share still exists
		_, err = service.GetShareByID(context.Background(), activeShare.ID)
		if err != nil {
			t.Errorf("Active share should still exist")
		}
	})
}

// ============================================================================
// Domain Model Tests
// ============================================================================

func TestSharePermissions(t *testing.T) {
	tests := []struct {
		name       string
		permission domain.SharePermission
		canView    bool
		canEdit    bool
		canAdmin   bool
	}{
		{"use", domain.SharePermissionUse, false, false, false},
		{"edit", domain.SharePermissionEdit, true, true, false},
		{"admin", domain.SharePermissionAdmin, true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.permission.CanView() != tt.canView {
				t.Errorf("%s.CanView() = %v, want %v", tt.name, tt.permission.CanView(), tt.canView)
			}
			if tt.permission.CanEdit() != tt.canEdit {
				t.Errorf("%s.CanEdit() = %v, want %v", tt.name, tt.permission.CanEdit(), tt.canEdit)
			}
			if tt.permission.CanAdmin() != tt.canAdmin {
				t.Errorf("%s.CanAdmin() = %v, want %v", tt.name, tt.permission.CanAdmin(), tt.canAdmin)
			}
		})
	}
}

func TestSharePermissionIsValid(t *testing.T) {
	validPermissions := []domain.SharePermission{
		domain.SharePermissionUse,
		domain.SharePermissionEdit,
		domain.SharePermissionAdmin,
	}

	for _, p := range validPermissions {
		if !p.IsValid() {
			t.Errorf("Permission %s should be valid", p)
		}
	}

	invalidPermissions := []domain.SharePermission{
		"invalid",
		"read",
		"write",
		"",
		"USE", // case sensitive
	}

	for _, p := range invalidPermissions {
		if p.IsValid() {
			t.Errorf("Permission %s should be invalid", p)
		}
	}
}

func TestCredentialShare_IsExpired(t *testing.T) {
	t.Run("not expired - no expiration", func(t *testing.T) {
		share := &domain.CredentialShare{}
		if share.IsExpired() {
			t.Error("Share with no expiration should not be expired")
		}
	})

	t.Run("not expired - future expiration", func(t *testing.T) {
		future := time.Now().Add(time.Hour)
		share := &domain.CredentialShare{ExpiresAt: &future}
		if share.IsExpired() {
			t.Error("Share with future expiration should not be expired")
		}
	})

	t.Run("expired - past expiration", func(t *testing.T) {
		past := time.Now().Add(-time.Hour)
		share := &domain.CredentialShare{ExpiresAt: &past}
		if !share.IsExpired() {
			t.Error("Share with past expiration should be expired")
		}
	})
}

func TestCredentialShare_IsSharedWithUser(t *testing.T) {
	userID := uuid.New()

	t.Run("shared with user", func(t *testing.T) {
		share := &domain.CredentialShare{SharedWithUserID: &userID}
		if !share.IsSharedWithUser() {
			t.Error("IsSharedWithUser() should return true")
		}
	})

	t.Run("not shared with user", func(t *testing.T) {
		share := &domain.CredentialShare{}
		if share.IsSharedWithUser() {
			t.Error("IsSharedWithUser() should return false")
		}
	})
}

func TestCredentialShare_IsSharedWithProject(t *testing.T) {
	projectID := uuid.New()

	t.Run("shared with project", func(t *testing.T) {
		share := &domain.CredentialShare{SharedWithProjectID: &projectID}
		if !share.IsSharedWithProject() {
			t.Error("IsSharedWithProject() should return true")
		}
	})

	t.Run("not shared with project", func(t *testing.T) {
		share := &domain.CredentialShare{}
		if share.IsSharedWithProject() {
			t.Error("IsSharedWithProject() should return false")
		}
	})
}

// ============================================================================
// Response Conversion Tests
// ============================================================================

func TestToShareResponse(t *testing.T) {
	credentialID := uuid.New()
	userID := uuid.New()
	sharedByUserID := uuid.New()
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)

	share := &domain.CredentialShare{
		ID:               uuid.New(),
		CredentialID:     credentialID,
		SharedWithUserID: &userID,
		Permission:       domain.SharePermissionUse,
		SharedByUserID:   sharedByUserID,
		Note:             "Test share",
		CreatedAt:        now,
		ExpiresAt:        &expiresAt,
	}

	resp := ToShareResponse(share)

	if resp.ID != share.ID {
		t.Errorf("ID = %s, want %s", resp.ID, share.ID)
	}
	if resp.CredentialID != share.CredentialID {
		t.Errorf("CredentialID = %s, want %s", resp.CredentialID, share.CredentialID)
	}
	if resp.SharedWithUserID == nil || *resp.SharedWithUserID != userID {
		t.Errorf("SharedWithUserID = %v, want %s", resp.SharedWithUserID, userID)
	}
	if resp.SharedWithProjectID != nil {
		t.Error("SharedWithProjectID should be nil")
	}
	if resp.Permission != share.Permission {
		t.Errorf("Permission = %s, want %s", resp.Permission, share.Permission)
	}
	if resp.Note != share.Note {
		t.Errorf("Note = %s, want %s", resp.Note, share.Note)
	}
	if resp.ExpiresAt == nil {
		t.Error("ExpiresAt should not be nil")
	}
}

func TestToShareResponses(t *testing.T) {
	shares := []*domain.CredentialShare{
		{
			ID:         uuid.New(),
			Permission: domain.SharePermissionUse,
		},
		{
			ID:         uuid.New(),
			Permission: domain.SharePermissionEdit,
		},
		{
			ID:         uuid.New(),
			Permission: domain.SharePermissionAdmin,
		},
	}

	responses := ToShareResponses(shares)

	if len(responses) != len(shares) {
		t.Errorf("Response length = %d, want %d", len(responses), len(shares))
	}

	for i, resp := range responses {
		if resp.ID != shares[i].ID {
			t.Errorf("Response[%d].ID = %s, want %s", i, resp.ID, shares[i].ID)
		}
		if resp.Permission != shares[i].Permission {
			t.Errorf("Response[%d].Permission = %s, want %s", i, resp.Permission, shares[i].Permission)
		}
	}
}

func TestToShareResponses_Empty(t *testing.T) {
	responses := ToShareResponses([]*domain.CredentialShare{})
	if len(responses) != 0 {
		t.Errorf("Empty input should return empty slice, got %d", len(responses))
	}
}
