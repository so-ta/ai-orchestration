package domain

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CredentialType represents the type of credential
type CredentialType string

const (
	CredentialTypeOAuth2 CredentialType = "oauth2"
	CredentialTypeAPIKey CredentialType = "api_key"
	CredentialTypeBasic  CredentialType = "basic"
	CredentialTypeBearer CredentialType = "bearer"
	CredentialTypeCustom CredentialType = "custom"
)

// ValidCredentialTypes returns all valid credential types
func ValidCredentialTypes() []CredentialType {
	return []CredentialType{
		CredentialTypeOAuth2,
		CredentialTypeAPIKey,
		CredentialTypeBasic,
		CredentialTypeBearer,
		CredentialTypeCustom,
	}
}

// IsValid checks if the credential type is valid
func (c CredentialType) IsValid() bool {
	for _, valid := range ValidCredentialTypes() {
		if c == valid {
			return true
		}
	}
	return false
}

// CredentialStatus represents the status of a credential
type CredentialStatus string

const (
	CredentialStatusActive  CredentialStatus = "active"
	CredentialStatusExpired CredentialStatus = "expired"
	CredentialStatusRevoked CredentialStatus = "revoked"
	CredentialStatusError   CredentialStatus = "error"
)

// Credential represents stored authentication credentials
type Credential struct {
	ID             uuid.UUID        `json:"id"`
	TenantID       uuid.UUID        `json:"tenant_id"`
	Name           string           `json:"name"`
	Description    string           `json:"description,omitempty"`
	CredentialType CredentialType   `json:"credential_type"`
	EncryptedData  []byte           `json:"-"` // Never expose in JSON
	EncryptedDEK   []byte           `json:"-"` // Never expose in JSON
	DataNonce      []byte           `json:"-"` // Nonce for data encryption
	DEKNonce       []byte           `json:"-"` // Nonce for DEK encryption
	Metadata       json.RawMessage  `json:"metadata"`
	ExpiresAt      *time.Time       `json:"expires_at,omitempty"`
	Status         CredentialStatus `json:"status"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

// NewCredential creates a new credential
func NewCredential(tenantID uuid.UUID, name string, credType CredentialType) *Credential {
	now := time.Now().UTC()
	return &Credential{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Name:           name,
		CredentialType: credType,
		Metadata:       json.RawMessage("{}"),
		Status:         CredentialStatusActive,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// IsExpired checks if the credential has expired
func (c *Credential) IsExpired() bool {
	if c.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*c.ExpiresAt)
}

// IsActive checks if the credential is active and not expired
func (c *Credential) IsActive() bool {
	return c.Status == CredentialStatusActive && !c.IsExpired()
}

// CredentialData represents decrypted credential data
type CredentialData struct {
	// Common fields
	Type string `json:"type"`

	// API Key
	APIKey       string `json:"api_key,omitempty"`
	HeaderName   string `json:"header_name,omitempty"`   // e.g., "X-API-Key", "Authorization"
	HeaderPrefix string `json:"header_prefix,omitempty"` // e.g., "Bearer ", "Api-Key "

	// Basic Auth
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`

	// OAuth2
	AccessToken  string     `json:"access_token,omitempty"`
	RefreshToken string     `json:"refresh_token,omitempty"`
	TokenType    string     `json:"token_type,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	Scopes       []string   `json:"scopes,omitempty"`

	// Custom (arbitrary key-value pairs)
	Custom map[string]string `json:"custom,omitempty"`
}

// ToJSON serializes CredentialData to JSON
func (c *CredentialData) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}

// CredentialDataFromJSON deserializes CredentialData from JSON
func CredentialDataFromJSON(data []byte) (*CredentialData, error) {
	var cd CredentialData
	if err := json.Unmarshal(data, &cd); err != nil {
		return nil, err
	}
	return &cd, nil
}

// CredentialMetadata represents non-sensitive metadata
type CredentialMetadata struct {
	ServiceName  string `json:"service_name,omitempty"`
	ServiceURL   string `json:"service_url,omitempty"`
	AccountID    string `json:"account_id,omitempty"`
	AccountEmail string `json:"account_email,omitempty"`
	Notes        string `json:"notes,omitempty"`
}

// ToJSON serializes CredentialMetadata to JSON
func (m *CredentialMetadata) ToJSON() (json.RawMessage, error) {
	return json.Marshal(m)
}

// CredentialMetadataFromJSON deserializes CredentialMetadata from JSON
func CredentialMetadataFromJSON(data json.RawMessage) (*CredentialMetadata, error) {
	var cm CredentialMetadata
	if err := json.Unmarshal(data, &cm); err != nil {
		return nil, err
	}
	return &cm, nil
}

// DecryptedCredential represents a credential with decrypted data
// Used internally, never serialized to JSON responses
type DecryptedCredential struct {
	*Credential
	Data *CredentialData `json:"-"`
}

// GetAuthHeader returns the authentication header for HTTP requests
func (dc *DecryptedCredential) GetAuthHeader() (name, value string) {
	if dc.Data == nil {
		return "", ""
	}

	switch dc.Credential.CredentialType {
	case CredentialTypeAPIKey:
		headerName := dc.Data.HeaderName
		if headerName == "" {
			headerName = "Authorization"
		}
		prefix := dc.Data.HeaderPrefix
		return headerName, prefix + dc.Data.APIKey

	case CredentialTypeBearer:
		return "Authorization", "Bearer " + dc.Data.AccessToken

	case CredentialTypeOAuth2:
		tokenType := dc.Data.TokenType
		if tokenType == "" {
			tokenType = "Bearer"
		}
		return "Authorization", tokenType + " " + dc.Data.AccessToken

	case CredentialTypeBasic:
		// Basic auth header is base64(username:password)
		// This should be handled by the HTTP client
		return "Authorization", "Basic " + basicAuthValue(dc.Data.Username, dc.Data.Password)

	default:
		return "", ""
	}
}

// basicAuthValue returns base64 encoded username:password
func basicAuthValue(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// ============================================================================
// SystemCredential - Operator-managed credentials for system blocks
// ============================================================================

// SystemCredential represents operator-managed credentials (not tenant-specific)
// These are used by system blocks and are not accessible by tenants
type SystemCredential struct {
	ID             uuid.UUID        `json:"id"`
	Name           string           `json:"name"`
	Description    string           `json:"description,omitempty"`
	CredentialType CredentialType   `json:"credential_type"`
	EncryptedData  []byte           `json:"-"` // Never expose in JSON
	EncryptedDEK   []byte           `json:"-"` // Never expose in JSON
	DataNonce      []byte           `json:"-"` // Nonce for data encryption
	DEKNonce       []byte           `json:"-"` // Nonce for DEK encryption
	Metadata       json.RawMessage  `json:"metadata"`
	ExpiresAt      *time.Time       `json:"expires_at,omitempty"`
	Status         CredentialStatus `json:"status"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

// NewSystemCredential creates a new system credential
func NewSystemCredential(name string, credType CredentialType) *SystemCredential {
	now := time.Now().UTC()
	return &SystemCredential{
		ID:             uuid.New(),
		Name:           name,
		CredentialType: credType,
		Metadata:       json.RawMessage("{}"),
		Status:         CredentialStatusActive,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// IsExpired checks if the system credential has expired
func (sc *SystemCredential) IsExpired() bool {
	if sc.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*sc.ExpiresAt)
}

// IsActive checks if the system credential is active and not expired
func (sc *SystemCredential) IsActive() bool {
	return sc.Status == CredentialStatusActive && !sc.IsExpired()
}

// DecryptedSystemCredential represents a system credential with decrypted data
type DecryptedSystemCredential struct {
	*SystemCredential
	Data *CredentialData `json:"-"`
}

// GetAuthHeader returns the authentication header for HTTP requests
func (dsc *DecryptedSystemCredential) GetAuthHeader() (name, value string) {
	if dsc.Data == nil {
		return "", ""
	}

	switch dsc.SystemCredential.CredentialType {
	case CredentialTypeAPIKey:
		headerName := dsc.Data.HeaderName
		if headerName == "" {
			headerName = "Authorization"
		}
		prefix := dsc.Data.HeaderPrefix
		return headerName, prefix + dsc.Data.APIKey

	case CredentialTypeBearer:
		return "Authorization", "Bearer " + dsc.Data.AccessToken

	case CredentialTypeOAuth2:
		tokenType := dsc.Data.TokenType
		if tokenType == "" {
			tokenType = "Bearer"
		}
		return "Authorization", tokenType + " " + dsc.Data.AccessToken

	case CredentialTypeBasic:
		return "Authorization", "Basic " + basicAuthValue(dsc.Data.Username, dsc.Data.Password)

	default:
		return "", ""
	}
}

// ============================================================================
// RequiredCredential - Credential requirement declaration for blocks
// ============================================================================

// CredentialScope defines where the credential comes from
type CredentialScope string

const (
	CredentialScopeSystem CredentialScope = "system" // Operator-managed (system_credentials)
	CredentialScopeTenant CredentialScope = "tenant" // Tenant-managed (credentials)
)

// RequiredCredential defines a credential requirement for a block
type RequiredCredential struct {
	Name        string          `json:"name"`        // Identifier used in code (e.g., "api_key")
	Type        CredentialType  `json:"type"`        // Expected credential type
	Scope       CredentialScope `json:"scope"`       // "system" or "tenant"
	Description string          `json:"description"` // Human-readable description
	Required    bool            `json:"required"`    // Whether this credential is mandatory
}

// CredentialBinding maps a required credential name to an actual credential ID
type CredentialBinding struct {
	Name         string    `json:"name"`          // RequiredCredential.Name
	CredentialID uuid.UUID `json:"credential_id"` // Reference to credentials table
}

// ParseRequiredCredentials parses JSON array of required credentials
func ParseRequiredCredentials(data json.RawMessage) ([]RequiredCredential, error) {
	if data == nil || len(data) == 0 || string(data) == "null" {
		return []RequiredCredential{}, nil
	}

	var creds []RequiredCredential
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}
	return creds, nil
}

// ParseCredentialBindings parses JSON object of credential bindings
func ParseCredentialBindings(data json.RawMessage) (map[string]uuid.UUID, error) {
	if data == nil || len(data) == 0 || string(data) == "null" {
		return map[string]uuid.UUID{}, nil
	}

	var bindings map[string]string
	if err := json.Unmarshal(data, &bindings); err != nil {
		return nil, err
	}

	result := make(map[string]uuid.UUID)
	for name, idStr := range bindings {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, err
		}
		result[name] = id
	}
	return result, nil
}
