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

// Pagination constants
const (
	DefaultPage      = 1
	DefaultLimit     = 20
	MaxLimit         = 100
	DefaultAuditLimit = 50
)

// NormalizePagination normalizes pagination parameters with default values
func NormalizePagination(page, limit int) (int, int) {
	if page < 1 {
		page = DefaultPage
	}
	if limit < 1 || limit > MaxLimit {
		limit = DefaultLimit
	}
	return page, limit
}

// NormalizePaginationWithLimit normalizes pagination with a custom default limit
func NormalizePaginationWithLimit(page, limit, defaultLimit int) (int, int) {
	if page < 1 {
		page = DefaultPage
	}
	if limit < 1 || limit > MaxLimit {
		limit = defaultLimit
	}
	return page, limit
}

// WorkflowChecker provides common workflow-related validation methods
type WorkflowChecker struct {
	workflowRepo repository.WorkflowRepository
}

// NewWorkflowChecker creates a new WorkflowChecker
func NewWorkflowChecker(repo repository.WorkflowRepository) *WorkflowChecker {
	return &WorkflowChecker{workflowRepo: repo}
}

// CheckExists verifies that a workflow exists
func (c *WorkflowChecker) CheckExists(ctx context.Context, tenantID, workflowID uuid.UUID) (*domain.Workflow, error) {
	workflow, err := c.workflowRepo.GetByID(ctx, tenantID, workflowID)
	if err != nil {
		return nil, err
	}
	return workflow, nil
}

// CheckEditable verifies that a workflow exists and can be edited
func (c *WorkflowChecker) CheckEditable(ctx context.Context, tenantID, workflowID uuid.UUID) (*domain.Workflow, error) {
	workflow, err := c.workflowRepo.GetByID(ctx, tenantID, workflowID)
	if err != nil {
		return nil, err
	}
	if !workflow.CanEdit() {
		return nil, domain.ErrWorkflowNotEditable
	}
	return workflow, nil
}

// CheckPublished verifies that a workflow exists and is published
func (c *WorkflowChecker) CheckPublished(ctx context.Context, tenantID, workflowID uuid.UUID) (*domain.Workflow, error) {
	workflow, err := c.workflowRepo.GetByID(ctx, tenantID, workflowID)
	if err != nil {
		return nil, err
	}
	if workflow.Status != domain.WorkflowStatusPublished {
		return nil, domain.ErrWorkflowNotPublished
	}
	return workflow, nil
}

// CredentialDecryptor provides common credential decryption methods
type CredentialDecryptor struct {
	encryptor *crypto.Encryptor
}

// NewCredentialDecryptor creates a new CredentialDecryptor
func NewCredentialDecryptor(encryptor *crypto.Encryptor) *CredentialDecryptor {
	return &CredentialDecryptor{encryptor: encryptor}
}

// Decrypt decrypts credential data
func (d *CredentialDecryptor) Decrypt(cred EncryptedCredentialData) (*domain.CredentialData, error) {
	encrypted := &crypto.EncryptedData{
		Ciphertext:   cred.GetEncryptedData(),
		EncryptedDEK: cred.GetEncryptedDEK(),
		DataNonce:    cred.GetDataNonce(),
		DEKNonce:     cred.GetDEKNonce(),
	}

	dataJSON, err := d.encryptor.Decrypt(encrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt credential: %w", err)
	}

	data, err := domain.CredentialDataFromJSON(dataJSON)
	if err != nil {
		return nil, fmt.Errorf("parse credential data: %w", err)
	}

	return data, nil
}

// Encrypt encrypts credential data
func (d *CredentialDecryptor) Encrypt(data *domain.CredentialData) (*crypto.EncryptedData, error) {
	dataJSON, err := data.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("serialize credential data: %w", err)
	}

	encrypted, err := d.encryptor.Encrypt(dataJSON)
	if err != nil {
		return nil, fmt.Errorf("encrypt credential: %w", err)
	}

	return encrypted, nil
}

// EncryptedCredentialData interface for credentials that have encrypted data
type EncryptedCredentialData interface {
	GetEncryptedData() []byte
	GetEncryptedDEK() []byte
	GetDataNonce() []byte
	GetDEKNonce() []byte
}

// ExtractInputSchemaFromConfig extracts the input_schema from a step's config JSON
func ExtractInputSchemaFromConfig(config json.RawMessage) json.RawMessage {
	if config == nil || len(config) == 0 {
		return nil
	}

	var configMap map[string]json.RawMessage
	if err := json.Unmarshal(config, &configMap); err != nil {
		return nil
	}

	if inputSchema, ok := configMap["input_schema"]; ok {
		return inputSchema
	}

	return nil
}

// CredentialDataToMap converts CredentialData to a map for sandbox access
// This is a shared utility for both CredentialUsecase and CredentialResolver
func CredentialDataToMap(data *domain.CredentialData) map[string]interface{} {
	result := make(map[string]interface{})

	if data.Type != "" {
		result["type"] = data.Type
	}

	// API Key fields
	if data.APIKey != "" {
		result["api_key"] = data.APIKey
	}
	if data.HeaderName != "" {
		result["header_name"] = data.HeaderName
	}
	if data.HeaderPrefix != "" {
		result["header_prefix"] = data.HeaderPrefix
	}

	// Basic Auth fields
	if data.Username != "" {
		result["username"] = data.Username
	}
	if data.Password != "" {
		result["password"] = data.Password
	}

	// OAuth2 fields
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
		result["expires_at"] = data.ExpiresAt.Unix()
	}
	if len(data.Scopes) > 0 {
		result["scopes"] = data.Scopes
	}

	// Custom fields
	for k, v := range data.Custom {
		result[k] = v
	}

	return result
}
