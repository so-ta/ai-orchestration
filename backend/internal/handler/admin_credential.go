package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
	"github.com/souta/ai-orchestration/pkg/crypto"
)

// AdminCredentialHandler handles HTTP requests for system credentials (operator-only)
type AdminCredentialHandler struct {
	repo      repository.SystemCredentialRepository
	encryptor *crypto.Encryptor
}

// NewAdminCredentialHandler creates a new AdminCredentialHandler
func NewAdminCredentialHandler(repo repository.SystemCredentialRepository, encryptor *crypto.Encryptor) *AdminCredentialHandler {
	return &AdminCredentialHandler{repo: repo, encryptor: encryptor}
}

// CreateSystemCredentialRequest represents the request body for creating a system credential
type CreateSystemCredentialRequest struct {
	Name           string                    `json:"name"`
	Description    string                    `json:"description,omitempty"`
	CredentialType domain.CredentialType     `json:"credential_type"`
	Data           *domain.CredentialData    `json:"data"`
	Metadata       *domain.CredentialMetadata `json:"metadata,omitempty"`
	ExpiresAt      *string                   `json:"expires_at,omitempty"` // RFC3339 format
}

// SystemCredentialResponse represents the response for a system credential
type SystemCredentialResponse struct {
	ID             uuid.UUID                `json:"id"`
	Name           string                   `json:"name"`
	Description    string                   `json:"description,omitempty"`
	CredentialType domain.CredentialType    `json:"credential_type"`
	Metadata       json.RawMessage          `json:"metadata"`
	ExpiresAt      *time.Time               `json:"expires_at,omitempty"`
	Status         domain.CredentialStatus  `json:"status"`
	CreatedAt      time.Time                `json:"created_at"`
	UpdatedAt      time.Time                `json:"updated_at"`
}

// toResponse converts a SystemCredential to a safe response (no encrypted data)
func (h *AdminCredentialHandler) toResponse(cred *domain.SystemCredential) *SystemCredentialResponse {
	return &SystemCredentialResponse{
		ID:             cred.ID,
		Name:           cred.Name,
		Description:    cred.Description,
		CredentialType: cred.CredentialType,
		Metadata:       cred.Metadata,
		ExpiresAt:      cred.ExpiresAt,
		Status:         cred.Status,
		CreatedAt:      cred.CreatedAt,
		UpdatedAt:      cred.UpdatedAt,
	}
}

// Create creates a new system credential
func (h *AdminCredentialHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateSystemCredentialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body", nil)
		return
	}

	// Validate required fields
	if req.Name == "" {
		Error(w, http.StatusBadRequest, "MISSING_NAME", "Name is required", nil)
		return
	}
	if !req.CredentialType.IsValid() {
		Error(w, http.StatusBadRequest, "INVALID_CREDENTIAL_TYPE", "Invalid credential type", nil)
		return
	}
	if req.Data == nil {
		Error(w, http.StatusBadRequest, "MISSING_DATA", "Credential data is required", nil)
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			Error(w, http.StatusBadRequest, "INVALID_EXPIRES_AT", "Invalid expires_at format (use RFC3339)", nil)
			return
		}
		expiresAt = &t
	}

	// Encrypt credential data
	dataJSON, err := req.Data.ToJSON()
	if err != nil {
		Error(w, http.StatusInternalServerError, "ENCRYPTION_ERROR", "Failed to serialize credential data", nil)
		return
	}

	encryptedData, err := h.encryptor.Encrypt(dataJSON)
	if err != nil {
		Error(w, http.StatusInternalServerError, "ENCRYPTION_ERROR", "Failed to encrypt credential data", nil)
		return
	}

	// Create credential
	cred := domain.NewSystemCredential(req.Name, req.CredentialType)
	cred.Description = req.Description
	cred.EncryptedData = encryptedData.Ciphertext
	cred.EncryptedDEK = encryptedData.EncryptedDEK
	cred.DataNonce = encryptedData.DataNonce
	cred.DEKNonce = encryptedData.DEKNonce
	cred.ExpiresAt = expiresAt

	// Set metadata
	if req.Metadata != nil {
		metaJSON, err := req.Metadata.ToJSON()
		if err != nil {
			Error(w, http.StatusBadRequest, "INVALID_METADATA", "Invalid metadata", nil)
			return
		}
		cred.Metadata = metaJSON
	}

	if err := h.repo.Create(r.Context(), cred); err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusCreated, h.toResponse(cred))
}

// Get retrieves a system credential by ID
func (h *AdminCredentialHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "credential_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid credential ID", nil)
		return
	}

	cred, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, h.toResponse(cred))
}

// List lists all system credentials
func (h *AdminCredentialHandler) List(w http.ResponseWriter, r *http.Request) {
	creds, err := h.repo.List(r.Context())
	if err != nil {
		HandleError(w, err)
		return
	}

	responses := make([]*SystemCredentialResponse, len(creds))
	for i, cred := range creds {
		responses[i] = h.toResponse(cred)
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"items": responses,
		"total": len(responses),
	})
}

// UpdateSystemCredentialRequest represents the request body for updating a system credential
type UpdateSystemCredentialRequest struct {
	Name        string                     `json:"name,omitempty"`
	Description string                     `json:"description,omitempty"`
	Data        *domain.CredentialData     `json:"data,omitempty"`
	Metadata    *domain.CredentialMetadata `json:"metadata,omitempty"`
	ExpiresAt   *string                    `json:"expires_at,omitempty"` // RFC3339 format
}

// Update updates a system credential
func (h *AdminCredentialHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "credential_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid credential ID", nil)
		return
	}

	var req UpdateSystemCredentialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body", nil)
		return
	}

	// Get existing credential
	cred, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}

	// Update fields
	if req.Name != "" {
		cred.Name = req.Name
	}
	if req.Description != "" {
		cred.Description = req.Description
	}
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			Error(w, http.StatusBadRequest, "INVALID_EXPIRES_AT", "Invalid expires_at format (use RFC3339)", nil)
			return
		}
		cred.ExpiresAt = &t
	}
	if req.Metadata != nil {
		metaJSON, err := req.Metadata.ToJSON()
		if err != nil {
			Error(w, http.StatusBadRequest, "INVALID_METADATA", "Invalid metadata", nil)
			return
		}
		cred.Metadata = metaJSON
	}

	// Re-encrypt data if provided
	if req.Data != nil {
		dataJSON, err := req.Data.ToJSON()
		if err != nil {
			Error(w, http.StatusInternalServerError, "ENCRYPTION_ERROR", "Failed to serialize credential data", nil)
			return
		}

		encryptedData, err := h.encryptor.Encrypt(dataJSON)
		if err != nil {
			Error(w, http.StatusInternalServerError, "ENCRYPTION_ERROR", "Failed to encrypt credential data", nil)
			return
		}

		cred.EncryptedData = encryptedData.Ciphertext
		cred.EncryptedDEK = encryptedData.EncryptedDEK
		cred.DataNonce = encryptedData.DataNonce
		cred.DEKNonce = encryptedData.DEKNonce
	}

	cred.UpdatedAt = time.Now().UTC()

	if err := h.repo.Update(r.Context(), cred); err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, h.toResponse(cred))
}

// Delete deletes a system credential
func (h *AdminCredentialHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "credential_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid credential ID", nil)
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Revoke revokes a system credential
func (h *AdminCredentialHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "credential_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid credential ID", nil)
		return
	}

	if err := h.repo.UpdateStatus(r.Context(), id, domain.CredentialStatusRevoked); err != nil {
		HandleError(w, err)
		return
	}

	cred, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, h.toResponse(cred))
}

// Activate activates a system credential
func (h *AdminCredentialHandler) Activate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "credential_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid credential ID", nil)
		return
	}

	if err := h.repo.UpdateStatus(r.Context(), id, domain.CredentialStatusActive); err != nil {
		HandleError(w, err)
		return
	}

	cred, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, h.toResponse(cred))
}
