package handler

import (
	"net/http"
	"time"

	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/usecase"
)

// CredentialHandler handles HTTP requests for credentials
type CredentialHandler struct {
	usecase      *usecase.CredentialUsecase
	auditService *usecase.AuditService
}

// NewCredentialHandler creates a new CredentialHandler
func NewCredentialHandler(uc *usecase.CredentialUsecase, auditService *usecase.AuditService) *CredentialHandler {
	return &CredentialHandler{
		usecase:      uc,
		auditService: auditService,
	}
}

// CreateCredentialRequest represents the request body for creating a credential
type CreateCredentialRequest struct {
	Name           string                    `json:"name"`
	Description    string                    `json:"description,omitempty"`
	CredentialType domain.CredentialType     `json:"credential_type"`
	Data           *domain.CredentialData    `json:"data"`
	Metadata       *domain.CredentialMetadata `json:"metadata,omitempty"`
	ExpiresAt      *string                   `json:"expires_at,omitempty"` // RFC3339 format
}

// Create creates a new credential
func (h *CredentialHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateCredentialRequest
	if !decodeJSONBody(w, r, &req) {
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

	tenantID := getTenantID(r)

	credential, err := h.usecase.Create(r.Context(), usecase.CreateCredentialInput{
		TenantID:       tenantID,
		Name:           req.Name,
		Description:    req.Description,
		CredentialType: req.CredentialType,
		Data:           req.Data,
		Metadata:       req.Metadata,
		ExpiresAt:      expiresAt,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionCredentialCreate, domain.AuditResourceCredential, &credential.ID, map[string]interface{}{
		"name":            credential.Name,
		"credential_type": string(credential.CredentialType),
	})

	// Return safe response (no encrypted data)
	JSON(w, http.StatusCreated, h.usecase.ToResponse(credential))
}

// Get retrieves a credential by ID
func (h *CredentialHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "credential_id", "credential ID")
	if !ok {
		return
	}

	tenantID := getTenantID(r)

	credential, err := h.usecase.GetByID(r.Context(), tenantID, id)
	if err != nil {
		HandleError(w, err)
		return
	}

	// Return safe response (no encrypted data)
	JSON(w, http.StatusOK, h.usecase.ToResponse(credential))
}

// List lists credentials
func (h *CredentialHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	input := usecase.ListCredentialsInput{
		TenantID: tenantID,
		Page:     parseIntQuery(r, "page", 1),
		Limit:    parseIntQuery(r, "limit", 20),
	}

	// Optional credential type filter
	if credTypeStr := r.URL.Query().Get("credential_type"); credTypeStr != "" {
		credType := domain.CredentialType(credTypeStr)
		if credType.IsValid() {
			input.CredentialType = &credType
		}
	}

	// Optional status filter
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		status := domain.CredentialStatus(statusStr)
		input.Status = &status
	}

	output, err := h.usecase.List(r.Context(), input)
	if err != nil {
		HandleError(w, err)
		return
	}

	// Return safe responses (no encrypted data)
	responses := h.usecase.ToResponses(output.Credentials)
	JSONList(w, http.StatusOK, responses, output.Page, output.Limit, output.Total)
}

// UpdateCredentialRequest represents the request body for updating a credential
type UpdateCredentialRequest struct {
	Name        string                     `json:"name,omitempty"`
	Description string                     `json:"description,omitempty"`
	Data        *domain.CredentialData     `json:"data,omitempty"`
	Metadata    *domain.CredentialMetadata `json:"metadata,omitempty"`
	ExpiresAt   *string                    `json:"expires_at,omitempty"` // RFC3339 format
}

// Update updates a credential
func (h *CredentialHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "credential_id", "credential ID")
	if !ok {
		return
	}

	var req UpdateCredentialRequest
	if !decodeJSONBody(w, r, &req) {
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

	tenantID := getTenantID(r)

	credential, err := h.usecase.Update(r.Context(), usecase.UpdateCredentialInput{
		TenantID:    tenantID,
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Data:        req.Data,
		Metadata:    req.Metadata,
		ExpiresAt:   expiresAt,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionCredentialUpdate, domain.AuditResourceCredential, &credential.ID, map[string]interface{}{
		"name": credential.Name,
	})

	JSON(w, http.StatusOK, h.usecase.ToResponse(credential))
}

// Delete deletes a credential
func (h *CredentialHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "credential_id", "credential ID")
	if !ok {
		return
	}

	tenantID := getTenantID(r)

	if err := h.usecase.Delete(r.Context(), tenantID, id); err != nil {
		HandleError(w, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionCredentialDelete, domain.AuditResourceCredential, &id, nil)

	w.WriteHeader(http.StatusNoContent)
}

// Revoke revokes a credential
func (h *CredentialHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "credential_id", "credential ID")
	if !ok {
		return
	}

	tenantID := getTenantID(r)

	credential, err := h.usecase.Revoke(r.Context(), tenantID, id)
	if err != nil {
		HandleError(w, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionCredentialRevoke, domain.AuditResourceCredential, &credential.ID, nil)

	JSON(w, http.StatusOK, h.usecase.ToResponse(credential))
}

// Activate activates a credential
func (h *CredentialHandler) Activate(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "credential_id", "credential ID")
	if !ok {
		return
	}

	tenantID := getTenantID(r)

	credential, err := h.usecase.Activate(r.Context(), tenantID, id)
	if err != nil {
		HandleError(w, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionCredentialActivate, domain.AuditResourceCredential, &credential.ID, nil)

	JSON(w, http.StatusOK, h.usecase.ToResponse(credential))
}
