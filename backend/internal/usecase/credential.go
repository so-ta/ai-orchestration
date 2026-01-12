package usecase

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
	"github.com/souta/ai-orchestration/pkg/crypto"
)

// CredentialUsecase handles credential business logic
type CredentialUsecase struct {
	credentialRepo repository.CredentialRepository
	encryptor      *crypto.Encryptor
}

// NewCredentialUsecase creates a new CredentialUsecase
func NewCredentialUsecase(
	credentialRepo repository.CredentialRepository,
	encryptor *crypto.Encryptor,
) *CredentialUsecase {
	return &CredentialUsecase{
		credentialRepo: credentialRepo,
		encryptor:      encryptor,
	}
}

// CreateCredentialInput represents input for creating a credential
type CreateCredentialInput struct {
	TenantID       uuid.UUID
	Name           string
	Description    string
	CredentialType domain.CredentialType
	Data           *domain.CredentialData
	Metadata       *domain.CredentialMetadata
	ExpiresAt      *time.Time
}

// Create creates a new credential
func (u *CredentialUsecase) Create(ctx context.Context, input CreateCredentialInput) (*domain.Credential, error) {
	// Validate input
	if input.Name == "" {
		return nil, domain.NewValidationError("name", "name is required")
	}
	if !input.CredentialType.IsValid() {
		return nil, domain.NewValidationError("credential_type", "invalid credential type")
	}
	if input.Data == nil {
		return nil, domain.NewValidationError("data", "credential data is required")
	}

	// Serialize credential data
	dataJSON, err := input.Data.ToJSON()
	if err != nil {
		return nil, domain.NewValidationError("data", "invalid credential data")
	}

	// Encrypt credential data
	encrypted, err := u.encryptor.Encrypt(dataJSON)
	if err != nil {
		return nil, err
	}

	// Create credential
	credential := domain.NewCredential(input.TenantID, input.Name, input.CredentialType)
	credential.Description = input.Description
	credential.EncryptedData = encrypted.Ciphertext
	credential.EncryptedDEK = encrypted.EncryptedDEK
	credential.DataNonce = encrypted.DataNonce
	credential.DEKNonce = encrypted.DEKNonce
	credential.ExpiresAt = input.ExpiresAt

	// Set metadata if provided
	if input.Metadata != nil {
		metadataJSON, err := input.Metadata.ToJSON()
		if err != nil {
			return nil, domain.NewValidationError("metadata", "invalid metadata")
		}
		credential.Metadata = metadataJSON
	}

	if err := u.credentialRepo.Create(ctx, credential); err != nil {
		return nil, err
	}

	return credential, nil
}

// GetByID retrieves a credential by ID (without decrypting data)
func (u *CredentialUsecase) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Credential, error) {
	return u.credentialRepo.GetByID(ctx, tenantID, id)
}

// GetDecrypted retrieves a credential with decrypted data
func (u *CredentialUsecase) GetDecrypted(ctx context.Context, tenantID, id uuid.UUID) (*domain.DecryptedCredential, error) {
	credential, err := u.credentialRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	// Check if credential is active
	if credential.Status != domain.CredentialStatusActive {
		if credential.Status == domain.CredentialStatusExpired {
			return nil, domain.ErrCredentialExpired
		}
		if credential.Status == domain.CredentialStatusRevoked {
			return nil, domain.ErrCredentialRevoked
		}
	}

	// Check if expired
	if credential.IsExpired() {
		// Update status to expired
		_ = u.credentialRepo.UpdateStatus(ctx, tenantID, id, domain.CredentialStatusExpired)
		return nil, domain.ErrCredentialExpired
	}

	// Decrypt credential data
	encrypted := &crypto.EncryptedData{
		Ciphertext:   credential.EncryptedData,
		EncryptedDEK: credential.EncryptedDEK,
		DataNonce:    credential.DataNonce,
		DEKNonce:     credential.DEKNonce,
	}

	dataJSON, err := u.encryptor.Decrypt(encrypted)
	if err != nil {
		return nil, err
	}

	data, err := domain.CredentialDataFromJSON(dataJSON)
	if err != nil {
		return nil, err
	}

	return &domain.DecryptedCredential{
		Credential: credential,
		Data:       data,
	}, nil
}

// ListCredentialsInput represents input for listing credentials
type ListCredentialsInput struct {
	TenantID       uuid.UUID
	CredentialType *domain.CredentialType
	Status         *domain.CredentialStatus
	Page           int
	Limit          int
}

// ListCredentialsOutput represents output for listing credentials
type ListCredentialsOutput struct {
	Credentials []*domain.Credential
	Total       int
	Page        int
	Limit       int
}

// List lists credentials with pagination (without decrypting data)
func (u *CredentialUsecase) List(ctx context.Context, input ListCredentialsInput) (*ListCredentialsOutput, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.Limit < 1 || input.Limit > 100 {
		input.Limit = 20
	}

	filter := repository.CredentialFilter{
		CredentialType: input.CredentialType,
		Status:         input.Status,
		Page:           input.Page,
		Limit:          input.Limit,
	}

	credentials, total, err := u.credentialRepo.List(ctx, input.TenantID, filter)
	if err != nil {
		return nil, err
	}

	return &ListCredentialsOutput{
		Credentials: credentials,
		Total:       total,
		Page:        input.Page,
		Limit:       input.Limit,
	}, nil
}

// UpdateCredentialInput represents input for updating a credential
type UpdateCredentialInput struct {
	TenantID    uuid.UUID
	ID          uuid.UUID
	Name        string
	Description string
	Data        *domain.CredentialData // If provided, re-encrypt
	Metadata    *domain.CredentialMetadata
	ExpiresAt   *time.Time
}

// Update updates a credential
func (u *CredentialUsecase) Update(ctx context.Context, input UpdateCredentialInput) (*domain.Credential, error) {
	credential, err := u.credentialRepo.GetByID(ctx, input.TenantID, input.ID)
	if err != nil {
		return nil, err
	}

	if input.Name != "" {
		credential.Name = input.Name
	}
	credential.Description = input.Description

	// Re-encrypt data if provided
	if input.Data != nil {
		dataJSON, err := input.Data.ToJSON()
		if err != nil {
			return nil, domain.NewValidationError("data", "invalid credential data")
		}

		encrypted, err := u.encryptor.Encrypt(dataJSON)
		if err != nil {
			return nil, err
		}

		credential.EncryptedData = encrypted.Ciphertext
		credential.EncryptedDEK = encrypted.EncryptedDEK
		credential.DataNonce = encrypted.DataNonce
		credential.DEKNonce = encrypted.DEKNonce
	}

	// Update metadata if provided
	if input.Metadata != nil {
		metadataJSON, err := input.Metadata.ToJSON()
		if err != nil {
			return nil, domain.NewValidationError("metadata", "invalid metadata")
		}
		credential.Metadata = metadataJSON
	}

	if input.ExpiresAt != nil {
		credential.ExpiresAt = input.ExpiresAt
	}

	credential.UpdatedAt = time.Now().UTC()

	if err := u.credentialRepo.Update(ctx, credential); err != nil {
		return nil, err
	}

	return credential, nil
}

// Delete deletes a credential
func (u *CredentialUsecase) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return u.credentialRepo.Delete(ctx, tenantID, id)
}

// Revoke revokes a credential
func (u *CredentialUsecase) Revoke(ctx context.Context, tenantID, id uuid.UUID) (*domain.Credential, error) {
	credential, err := u.credentialRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	credential.Status = domain.CredentialStatusRevoked
	credential.UpdatedAt = time.Now().UTC()

	if err := u.credentialRepo.Update(ctx, credential); err != nil {
		return nil, err
	}

	return credential, nil
}

// Activate activates a credential (e.g., after fixing issues)
func (u *CredentialUsecase) Activate(ctx context.Context, tenantID, id uuid.UUID) (*domain.Credential, error) {
	credential, err := u.credentialRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	// Don't reactivate if expired
	if credential.IsExpired() {
		return nil, domain.ErrCredentialExpired
	}

	credential.Status = domain.CredentialStatusActive
	credential.UpdatedAt = time.Now().UTC()

	if err := u.credentialRepo.Update(ctx, credential); err != nil {
		return nil, err
	}

	return credential, nil
}

// GetByName retrieves a credential by name within a tenant
func (u *CredentialUsecase) GetByName(ctx context.Context, tenantID uuid.UUID, name string) (*domain.Credential, error) {
	return u.credentialRepo.GetByName(ctx, tenantID, name)
}

// GetDecryptedByName retrieves a credential by name with decrypted data
func (u *CredentialUsecase) GetDecryptedByName(ctx context.Context, tenantID uuid.UUID, name string) (*domain.DecryptedCredential, error) {
	credential, err := u.credentialRepo.GetByName(ctx, tenantID, name)
	if err != nil {
		return nil, err
	}

	return u.GetDecrypted(ctx, tenantID, credential.ID)
}

// ToCredentialMap converts CredentialData to map for use in sandbox context
func (u *CredentialUsecase) ToCredentialMap(data *domain.CredentialData) map[string]interface{} {
	result := make(map[string]interface{})

	if data.APIKey != "" {
		result["api_key"] = data.APIKey
	}
	if data.HeaderName != "" {
		result["header_name"] = data.HeaderName
	}
	if data.HeaderPrefix != "" {
		result["header_prefix"] = data.HeaderPrefix
	}
	if data.Username != "" {
		result["username"] = data.Username
	}
	if data.Password != "" {
		result["password"] = data.Password
	}
	if data.AccessToken != "" {
		result["access_token"] = data.AccessToken
	}
	if data.RefreshToken != "" {
		result["refresh_token"] = data.RefreshToken
	}
	if data.TokenType != "" {
		result["token_type"] = data.TokenType
	}
	if data.ExpiresAt != nil {
		result["expires_at"] = data.ExpiresAt.Format(time.RFC3339)
	}
	if len(data.Scopes) > 0 {
		result["scopes"] = data.Scopes
	}
	if len(data.Custom) > 0 {
		for k, v := range data.Custom {
			result[k] = v
		}
	}

	return result
}

// CredentialResponse represents the API response for a credential
type CredentialResponse struct {
	ID             uuid.UUID              `json:"id"`
	TenantID       uuid.UUID              `json:"tenant_id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description,omitempty"`
	CredentialType domain.CredentialType  `json:"credential_type"`
	Metadata       json.RawMessage        `json:"metadata"`
	ExpiresAt      *time.Time             `json:"expires_at,omitempty"`
	Status         domain.CredentialStatus `json:"status"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// ToResponse converts a Credential to a safe response (no encrypted data)
func (u *CredentialUsecase) ToResponse(c *domain.Credential) *CredentialResponse {
	return &CredentialResponse{
		ID:             c.ID,
		TenantID:       c.TenantID,
		Name:           c.Name,
		Description:    c.Description,
		CredentialType: c.CredentialType,
		Metadata:       c.Metadata,
		ExpiresAt:      c.ExpiresAt,
		Status:         c.Status,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
	}
}

// ToResponses converts multiple Credentials to safe responses
func (u *CredentialUsecase) ToResponses(credentials []*domain.Credential) []*CredentialResponse {
	responses := make([]*CredentialResponse, len(credentials))
	for i, c := range credentials {
		responses[i] = u.ToResponse(c)
	}
	return responses
}
