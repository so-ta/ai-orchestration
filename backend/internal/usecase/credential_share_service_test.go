package usecase

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
)

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
	if resp.Permission != share.Permission {
		t.Errorf("Permission = %s, want %s", resp.Permission, share.Permission)
	}
	if resp.Note != share.Note {
		t.Errorf("Note = %s, want %s", resp.Note, share.Note)
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
	}

	for _, p := range invalidPermissions {
		if p.IsValid() {
			t.Errorf("Permission %s should be invalid", p)
		}
	}
}
