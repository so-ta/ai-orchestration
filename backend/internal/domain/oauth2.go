package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// OAuth2Provider - OAuth2 provider configuration (preset or custom)
// ============================================================================

// OAuth2Provider represents an OAuth2 provider configuration
type OAuth2Provider struct {
	ID   uuid.UUID `json:"id"`
	Slug string    `json:"slug"` // e.g., "google", "github", "slack"
	Name string    `json:"name"`

	// Endpoints
	AuthorizationURL string `json:"authorization_url"`
	TokenURL         string `json:"token_url"`
	RevokeURL        string `json:"revoke_url,omitempty"`
	UserinfoURL      string `json:"userinfo_url,omitempty"`

	// Configuration
	PKCERequired  bool     `json:"pkce_required"`
	DefaultScopes []string `json:"default_scopes"`

	// UI
	IconURL          string `json:"icon_url,omitempty"`
	DocumentationURL string `json:"documentation_url,omitempty"`

	// Metadata
	IsPreset  bool      `json:"is_preset"` // true = system-defined, false = custom
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewOAuth2Provider creates a new custom OAuth2 provider
func NewOAuth2Provider(slug, name, authURL, tokenURL string) *OAuth2Provider {
	now := time.Now().UTC()
	return &OAuth2Provider{
		ID:               uuid.New(),
		Slug:             slug,
		Name:             name,
		AuthorizationURL: authURL,
		TokenURL:         tokenURL,
		DefaultScopes:    []string{},
		IsPreset:         false,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// ============================================================================
// OAuth2App - Tenant-specific OAuth2 application configuration
// ============================================================================

// OAuth2App represents a tenant's OAuth2 application configuration
type OAuth2App struct {
	ID         uuid.UUID `json:"id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	ProviderID uuid.UUID `json:"provider_id"`

	// Encrypted client credentials
	EncryptedClientID     []byte `json:"-"`
	EncryptedClientSecret []byte `json:"-"`
	ClientIDNonce         []byte `json:"-"`
	ClientSecretNonce     []byte `json:"-"`

	// Customization
	CustomScopes []string `json:"custom_scopes,omitempty"`
	RedirectURI  string   `json:"redirect_uri,omitempty"`

	// Status
	Status OAuth2AppStatus `json:"status"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relations (populated by repository)
	Provider *OAuth2Provider `json:"provider,omitempty"`
}

// OAuth2AppStatus represents the status of an OAuth2 app
type OAuth2AppStatus string

const (
	OAuth2AppStatusActive   OAuth2AppStatus = "active"
	OAuth2AppStatusDisabled OAuth2AppStatus = "disabled"
)

// NewOAuth2App creates a new OAuth2 app for a tenant
func NewOAuth2App(tenantID, providerID uuid.UUID) *OAuth2App {
	now := time.Now().UTC()
	return &OAuth2App{
		ID:         uuid.New(),
		TenantID:   tenantID,
		ProviderID: providerID,
		Status:     OAuth2AppStatusActive,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// GetScopes returns custom scopes if set, otherwise provider's default scopes
func (a *OAuth2App) GetScopes() []string {
	if len(a.CustomScopes) > 0 {
		return a.CustomScopes
	}
	if a.Provider != nil {
		return a.Provider.DefaultScopes
	}
	return []string{}
}

// ============================================================================
// OAuth2Connection - Individual OAuth2 token connection
// ============================================================================

// OAuth2Connection represents an OAuth2 token connection linked to a credential
type OAuth2Connection struct {
	ID           uuid.UUID `json:"id"`
	CredentialID uuid.UUID `json:"credential_id"`
	OAuth2AppID  uuid.UUID `json:"oauth2_app_id"`

	// Encrypted tokens
	EncryptedAccessToken  []byte `json:"-"`
	EncryptedRefreshToken []byte `json:"-"`
	AccessTokenNonce      []byte `json:"-"`
	RefreshTokenNonce     []byte `json:"-"`
	TokenType             string `json:"token_type"` // Usually "Bearer"

	// Expiration
	AccessTokenExpiresAt  *time.Time `json:"access_token_expires_at,omitempty"`
	RefreshTokenExpiresAt *time.Time `json:"refresh_token_expires_at,omitempty"`

	// OAuth2 flow state (temporary, used during authorization)
	State        string `json:"-"` // CSRF protection
	CodeVerifier string `json:"-"` // PKCE

	// Account information from userinfo endpoint
	AccountID    string          `json:"account_id,omitempty"`
	AccountEmail string          `json:"account_email,omitempty"`
	AccountName  string          `json:"account_name,omitempty"`
	RawUserinfo  json.RawMessage `json:"raw_userinfo,omitempty"`

	// Status
	Status        OAuth2ConnectionStatus `json:"status"`
	LastRefreshAt *time.Time             `json:"last_refresh_at,omitempty"`
	LastUsedAt    *time.Time             `json:"last_used_at,omitempty"`
	ErrorMessage  string                 `json:"error_message,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// OAuth2ConnectionStatus represents the status of an OAuth2 connection
type OAuth2ConnectionStatus string

const (
	OAuth2ConnectionStatusPending   OAuth2ConnectionStatus = "pending"
	OAuth2ConnectionStatusConnected OAuth2ConnectionStatus = "connected"
	OAuth2ConnectionStatusExpired   OAuth2ConnectionStatus = "expired"
	OAuth2ConnectionStatusRevoked   OAuth2ConnectionStatus = "revoked"
	OAuth2ConnectionStatusError     OAuth2ConnectionStatus = "error"
)

// NewOAuth2Connection creates a new OAuth2 connection
func NewOAuth2Connection(credentialID, oauth2AppID uuid.UUID) *OAuth2Connection {
	now := time.Now().UTC()
	return &OAuth2Connection{
		ID:           uuid.New(),
		CredentialID: credentialID,
		OAuth2AppID:  oauth2AppID,
		TokenType:    "Bearer",
		Status:       OAuth2ConnectionStatusPending,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// IsAccessTokenExpired checks if the access token has expired (with 5 minute buffer)
func (c *OAuth2Connection) IsAccessTokenExpired() bool {
	if c.AccessTokenExpiresAt == nil {
		return false
	}
	// 5 minute buffer for proactive refresh
	return time.Now().Add(5 * time.Minute).After(*c.AccessTokenExpiresAt)
}

// IsRefreshTokenExpired checks if the refresh token has expired
func (c *OAuth2Connection) IsRefreshTokenExpired() bool {
	if c.RefreshTokenExpiresAt == nil {
		return false // No expiration = never expires
	}
	return time.Now().After(*c.RefreshTokenExpiresAt)
}

// CanRefresh checks if the connection can be refreshed
func (c *OAuth2Connection) CanRefresh() bool {
	return c.Status == OAuth2ConnectionStatusConnected &&
		c.EncryptedRefreshToken != nil &&
		!c.IsRefreshTokenExpired()
}

// MarkAsConnected updates the connection status to connected
func (c *OAuth2Connection) MarkAsConnected() {
	c.Status = OAuth2ConnectionStatusConnected
	c.ErrorMessage = ""
	c.UpdatedAt = time.Now().UTC()
}

// MarkAsExpired updates the connection status to expired
func (c *OAuth2Connection) MarkAsExpired() {
	c.Status = OAuth2ConnectionStatusExpired
	c.UpdatedAt = time.Now().UTC()
}

// MarkAsError updates the connection status to error
func (c *OAuth2Connection) MarkAsError(errMsg string) {
	c.Status = OAuth2ConnectionStatusError
	c.ErrorMessage = errMsg
	c.UpdatedAt = time.Now().UTC()
}

// MarkAsRevoked updates the connection status to revoked
func (c *OAuth2Connection) MarkAsRevoked() {
	c.Status = OAuth2ConnectionStatusRevoked
	c.UpdatedAt = time.Now().UTC()
}

// UpdateTokens updates the access and refresh tokens after a refresh
func (c *OAuth2Connection) UpdateTokens(encryptedAccessToken, encryptedRefreshToken, accessNonce, refreshNonce []byte, expiresAt *time.Time) {
	now := time.Now().UTC()
	c.EncryptedAccessToken = encryptedAccessToken
	c.AccessTokenNonce = accessNonce
	c.AccessTokenExpiresAt = expiresAt
	c.LastRefreshAt = &now

	if encryptedRefreshToken != nil {
		c.EncryptedRefreshToken = encryptedRefreshToken
		c.RefreshTokenNonce = refreshNonce
	}

	c.Status = OAuth2ConnectionStatusConnected
	c.ErrorMessage = ""
	c.UpdatedAt = now
}

// RecordUsage records when the token was last used
func (c *OAuth2Connection) RecordUsage() {
	now := time.Now().UTC()
	c.LastUsedAt = &now
	c.UpdatedAt = now
}

// ============================================================================
// OAuth2 Authorization Request/Response
// ============================================================================

// OAuth2AuthorizationRequest represents an authorization request
type OAuth2AuthorizationRequest struct {
	ProviderSlug string      `json:"provider_slug"`
	Scope        OwnerScope  `json:"scope"`
	ProjectID    *uuid.UUID  `json:"project_id,omitempty"`
	Name         string      `json:"name"`
	Scopes       []string    `json:"scopes,omitempty"` // Additional scopes
}

// OAuth2AuthorizationResponse represents the response for starting authorization
type OAuth2AuthorizationResponse struct {
	AuthorizationURL string `json:"authorization_url"`
	State            string `json:"state"`
}

// OAuth2TokenResponse represents the response from token endpoint
type OAuth2TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`

	// Error fields
	Error            string `json:"error,omitempty"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// OAuth2UserinfoResponse represents common fields from userinfo endpoint
type OAuth2UserinfoResponse struct {
	Sub           string `json:"sub,omitempty"`
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
	Name          string `json:"name,omitempty"`
	Picture       string `json:"picture,omitempty"`

	// GitHub specific
	Login string `json:"login,omitempty"`
	ID    int64  `json:"id,omitempty"`

	// Full response for storage
	Raw json.RawMessage `json:"-"`
}

// GetAccountID returns the account ID from various providers
func (u *OAuth2UserinfoResponse) GetAccountID() string {
	if u.Sub != "" {
		return u.Sub
	}
	if u.Login != "" {
		return u.Login
	}
	if u.ID != 0 {
		return string(rune(u.ID))
	}
	return ""
}

// GetDisplayName returns the display name from various providers
func (u *OAuth2UserinfoResponse) GetDisplayName() string {
	if u.Name != "" {
		return u.Name
	}
	if u.Login != "" {
		return u.Login
	}
	if u.Email != "" {
		return u.Email
	}
	return ""
}
