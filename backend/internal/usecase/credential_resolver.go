package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
	"github.com/souta/ai-orchestration/pkg/crypto"
)

// CredentialResolver resolves credentials for block execution
type CredentialResolver struct {
	credentialRepo       repository.CredentialRepository
	systemCredentialRepo repository.SystemCredentialRepository
	encryptor            *crypto.Encryptor
}

// NewCredentialResolver creates a new CredentialResolver
func NewCredentialResolver(
	credentialRepo repository.CredentialRepository,
	systemCredentialRepo repository.SystemCredentialRepository,
	encryptor *crypto.Encryptor,
) *CredentialResolver {
	return &CredentialResolver{
		credentialRepo:       credentialRepo,
		systemCredentialRepo: systemCredentialRepo,
		encryptor:            encryptor,
	}
}

// ResolvedCredentials contains the resolved and decrypted credentials for a block
type ResolvedCredentials struct {
	// Map of credential name to decrypted credential data
	// This is passed to the sandbox as context.credentials
	Credentials map[string]map[string]interface{}
}

// ResolveForStep resolves credentials for a step execution
// It takes the block definition (with required_credentials), the step (with credential_bindings),
// and the tenant ID to resolve tenant-scoped credentials
func (r *CredentialResolver) ResolveForStep(
	ctx context.Context,
	block *domain.BlockDefinition,
	step *domain.Step,
	tenantID uuid.UUID,
) (*ResolvedCredentials, error) {
	// Parse required credentials from block definition
	requiredCreds, err := block.GetRequiredCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to parse required credentials: %w", err)
	}

	// Parse credential bindings from step
	bindings, err := step.GetCredentialBindings()
	if err != nil {
		return nil, fmt.Errorf("failed to parse credential bindings: %w", err)
	}

	result := &ResolvedCredentials{
		Credentials: make(map[string]map[string]interface{}),
	}

	// Resolve each required credential
	for _, req := range requiredCreds {
		var credData map[string]interface{}
		var resolveErr error

		switch req.Scope {
		case domain.CredentialScopeSystem:
			// System credentials are resolved by name
			credData, resolveErr = r.resolveSystemCredential(ctx, req.Name)

		case domain.CredentialScopeTenant:
			// Tenant credentials are resolved by binding
			credID, hasBind := bindings[req.Name]
			if !hasBind {
				if req.Required {
					return nil, fmt.Errorf("required credential '%s' not bound", req.Name)
				}
				continue // Skip optional unbound credentials
			}
			credData, resolveErr = r.resolveTenantCredential(ctx, tenantID, credID)

		default:
			return nil, fmt.Errorf("unknown credential scope: %s", req.Scope)
		}

		if resolveErr != nil {
			if req.Required {
				return nil, fmt.Errorf("failed to resolve credential '%s': %w", req.Name, resolveErr)
			}
			continue // Skip optional credentials that fail to resolve
		}

		result.Credentials[req.Name] = credData
	}

	return result, nil
}

// resolveSystemCredential resolves a system credential by name
func (r *CredentialResolver) resolveSystemCredential(ctx context.Context, name string) (map[string]interface{}, error) {
	cred, err := r.systemCredentialRepo.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}

	// Check status
	if !cred.IsActive() {
		if cred.IsExpired() {
			return nil, domain.ErrSystemCredentialExpired
		}
		return nil, domain.ErrSystemCredentialRevoked
	}

	// Decrypt credential data
	plaintext, err := r.encryptor.Decrypt(&crypto.EncryptedData{
		Ciphertext:   cred.EncryptedData,
		EncryptedDEK: cred.EncryptedDEK,
		DataNonce:    cred.DataNonce,
		DEKNonce:     cred.DEKNonce,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credential: %w", err)
	}

	// Parse credential data
	credData, err := domain.CredentialDataFromJSON(plaintext)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credential data: %w", err)
	}

	return CredentialDataToMap(credData), nil
}

// resolveTenantCredential resolves a tenant credential by ID
func (r *CredentialResolver) resolveTenantCredential(ctx context.Context, tenantID, credID uuid.UUID) (map[string]interface{}, error) {
	cred, err := r.credentialRepo.GetByID(ctx, tenantID, credID)
	if err != nil {
		return nil, err
	}

	// Check status
	if !cred.IsActive() {
		if cred.IsExpired() {
			return nil, domain.ErrCredentialExpired
		}
		return nil, domain.ErrCredentialRevoked
	}

	// Decrypt credential data
	plaintext, err := r.encryptor.Decrypt(&crypto.EncryptedData{
		Ciphertext:   cred.EncryptedData,
		EncryptedDEK: cred.EncryptedDEK,
		DataNonce:    cred.DataNonce,
		DEKNonce:     cred.DEKNonce,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credential: %w", err)
	}

	// Parse credential data
	credData, err := domain.CredentialDataFromJSON(plaintext)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credential data: %w", err)
	}

	return CredentialDataToMap(credData), nil
}

// Note: credentialDataToMap has been moved to helpers.go as CredentialDataToMap

// GetAuthHeader returns the appropriate authentication header for a resolved credential
func (r *CredentialResolver) GetAuthHeader(cred map[string]interface{}) (headerName, headerValue string) {
	credType, _ := cred["type"].(string)

	switch credType {
	case "api_key":
		headerName = "Authorization"
		if name, ok := cred["header_name"].(string); ok && name != "" {
			headerName = name
		}
		apiKey, _ := cred["api_key"].(string)
		prefix, _ := cred["header_prefix"].(string)
		return headerName, prefix + apiKey

	case "bearer":
		accessToken, _ := cred["access_token"].(string)
		return "Authorization", "Bearer " + accessToken

	case "oauth2":
		tokenType := "Bearer"
		if tt, ok := cred["token_type"].(string); ok && tt != "" {
			tokenType = tt
		}
		accessToken, _ := cred["access_token"].(string)
		return "Authorization", tokenType + " " + accessToken

	case "basic":
		username, _ := cred["username"].(string)
		password, _ := cred["password"].(string)
		auth := username + ":" + password
		encoded := base64Encode([]byte(auth))
		return "Authorization", "Basic " + encoded

	default:
		return "", ""
	}
}

// base64Encode encodes bytes to base64 string
func base64Encode(data []byte) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	result := make([]byte, 0, (len(data)+2)/3*4)

	for i := 0; i < len(data); i += 3 {
		var n uint32
		for j := 0; j < 3; j++ {
			n <<= 8
			if i+j < len(data) {
				n |= uint32(data[i+j])
			}
		}

		for j := 0; j < 4; j++ {
			if i*8/6+j > (len(data)*8+5)/6 {
				result = append(result, '=')
			} else {
				result = append(result, charset[(n>>(18-j*6))&0x3F])
			}
		}
	}

	return string(result)
}

// CredentialsToContext converts resolved credentials to a format suitable for sandbox context
// Returns a map where keys are credential names and values contain the credential data
func (r *CredentialResolver) CredentialsToContext(resolved *ResolvedCredentials) map[string]interface{} {
	if resolved == nil {
		return make(map[string]interface{})
	}

	result := make(map[string]interface{})
	for name, data := range resolved.Credentials {
		result[name] = data
	}
	return result
}

// EncryptCredentialData encrypts credential data for storage
func (r *CredentialResolver) EncryptCredentialData(data *domain.CredentialData) (*crypto.EncryptedData, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal credential data: %w", err)
	}

	return r.encryptor.Encrypt(jsonData)
}
