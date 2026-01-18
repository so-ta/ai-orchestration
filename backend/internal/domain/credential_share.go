package domain

import (
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// CredentialShare - Sharing credentials between users and projects
// ============================================================================

// SharePermission represents the permission level for a credential share
type SharePermission string

const (
	// SharePermissionUse allows using the credential but not viewing details
	SharePermissionUse SharePermission = "use"
	// SharePermissionEdit allows viewing and updating the credential
	SharePermissionEdit SharePermission = "edit"
	// SharePermissionAdmin allows full control including delete and re-share
	SharePermissionAdmin SharePermission = "admin"
)

// ValidSharePermissions returns all valid share permissions
func ValidSharePermissions() []SharePermission {
	return []SharePermission{
		SharePermissionUse,
		SharePermissionEdit,
		SharePermissionAdmin,
	}
}

// IsValid checks if the share permission is valid
func (p SharePermission) IsValid() bool {
	for _, valid := range ValidSharePermissions() {
		if p == valid {
			return true
		}
	}
	return false
}

// CanView returns whether this permission allows viewing credential details
func (p SharePermission) CanView() bool {
	return p == SharePermissionEdit || p == SharePermissionAdmin
}

// CanEdit returns whether this permission allows editing the credential
func (p SharePermission) CanEdit() bool {
	return p == SharePermissionEdit || p == SharePermissionAdmin
}

// CanAdmin returns whether this permission allows admin operations
func (p SharePermission) CanAdmin() bool {
	return p == SharePermissionAdmin
}

// CredentialShare represents a credential share with a user or project
type CredentialShare struct {
	ID           uuid.UUID `json:"id"`
	CredentialID uuid.UUID `json:"credential_id"`

	// Share target (one must be set)
	SharedWithUserID    *uuid.UUID `json:"shared_with_user_id,omitempty"`
	SharedWithProjectID *uuid.UUID `json:"shared_with_project_id,omitempty"`

	// Permission and metadata
	Permission     SharePermission `json:"permission"`
	SharedByUserID uuid.UUID       `json:"shared_by_user_id"`
	Note           string          `json:"note,omitempty"`

	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// Relations (populated by repository)
	SharedWithUser    *User    `json:"shared_with_user,omitempty"`
	SharedWithProject *Project `json:"shared_with_project,omitempty"`
	SharedByUser      *User    `json:"shared_by_user,omitempty"`
}

// NewCredentialShareWithUser creates a new share with a user
func NewCredentialShareWithUser(credentialID, sharedWithUserID, sharedByUserID uuid.UUID, permission SharePermission) *CredentialShare {
	now := time.Now().UTC()
	return &CredentialShare{
		ID:               uuid.New(),
		CredentialID:     credentialID,
		SharedWithUserID: &sharedWithUserID,
		Permission:       permission,
		SharedByUserID:   sharedByUserID,
		CreatedAt:        now,
	}
}

// NewCredentialShareWithProject creates a new share with a project
func NewCredentialShareWithProject(credentialID, sharedWithProjectID, sharedByUserID uuid.UUID, permission SharePermission) *CredentialShare {
	now := time.Now().UTC()
	return &CredentialShare{
		ID:                  uuid.New(),
		CredentialID:        credentialID,
		SharedWithProjectID: &sharedWithProjectID,
		Permission:          permission,
		SharedByUserID:      sharedByUserID,
		CreatedAt:           now,
	}
}

// IsExpired checks if the share has expired
func (s *CredentialShare) IsExpired() bool {
	if s.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*s.ExpiresAt)
}

// IsSharedWithUser returns true if this share is with a user
func (s *CredentialShare) IsSharedWithUser() bool {
	return s.SharedWithUserID != nil
}

// IsSharedWithProject returns true if this share is with a project
func (s *CredentialShare) IsSharedWithProject() bool {
	return s.SharedWithProjectID != nil
}

// Validate validates the credential share
func (s *CredentialShare) Validate() error {
	// Exactly one target must be set
	hasUser := s.SharedWithUserID != nil
	hasProject := s.SharedWithProjectID != nil

	if hasUser == hasProject {
		// Both set or neither set
		return ErrCredentialShareNotFound
	}

	if !s.Permission.IsValid() {
		return ErrCredentialShareNotFound
	}

	return nil
}

// ============================================================================
// CredentialWithAccess - Credential with access information for listing
// ============================================================================

// CredentialWithAccess represents a credential with access context
type CredentialWithAccess struct {
	*Credential

	// Access information
	AccessSource    CredentialAccessSource `json:"access_source"`     // own, shared
	SharePermission *SharePermission       `json:"share_permission,omitempty"` // Set if access_source is "shared"
	SharedByUserID  *uuid.UUID             `json:"shared_by_user_id,omitempty"`

	// OAuth2 connection info (if oauth2 type)
	OAuth2AccountEmail string                  `json:"oauth2_account_email,omitempty"`
	OAuth2AccountName  string                  `json:"oauth2_account_name,omitempty"`
	OAuth2Status       *OAuth2ConnectionStatus `json:"oauth2_status,omitempty"`
}

// CredentialAccessSource indicates how the user has access to the credential
type CredentialAccessSource string

const (
	CredentialAccessSourceOwn    CredentialAccessSource = "own"    // User owns or has direct access
	CredentialAccessSourceShared CredentialAccessSource = "shared" // Shared with user/project
)

// MaskedPreview returns a masked preview of the credential value
// Shows only last 6 characters for security
func (c *CredentialWithAccess) MaskedPreview() string {
	// This would be populated during retrieval from a separate secure operation
	return "••••••••"
}
