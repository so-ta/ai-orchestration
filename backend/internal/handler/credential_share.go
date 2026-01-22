package handler

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/usecase"
)

// CredentialShareHandler handles HTTP requests for credential sharing
type CredentialShareHandler struct {
	service      *usecase.CredentialShareService
	auditService *usecase.AuditService
}

// NewCredentialShareHandler creates a new CredentialShareHandler
func NewCredentialShareHandler(service *usecase.CredentialShareService, auditService *usecase.AuditService) *CredentialShareHandler {
	return &CredentialShareHandler{
		service:      service,
		auditService: auditService,
	}
}

// ShareWithUserRequest represents the request body for sharing with a user
type ShareWithUserRequest struct {
	CredentialID     uuid.UUID              `json:"credential_id"`
	SharedWithUserID uuid.UUID              `json:"shared_with_user_id"`
	Permission       domain.SharePermission `json:"permission"`
	Note             string                 `json:"note,omitempty"`
	ExpiresAt        *string                `json:"expires_at,omitempty"` // RFC3339 format
}

// ShareWithUser shares a credential with a user
func (h *CredentialShareHandler) ShareWithUser(w http.ResponseWriter, r *http.Request) {
	var req ShareWithUserRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	if req.CredentialID == uuid.Nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "credential_id is required", nil)
		return
	}
	if req.SharedWithUserID == uuid.Nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "shared_with_user_id is required", nil)
		return
	}
	if !req.Permission.IsValid() {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid permission: must be use, edit, or admin", nil)
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
	userID := getUserID(r)

	share, err := h.service.ShareWithUser(r.Context(), usecase.ShareWithUserInput{
		TenantID:         tenantID,
		CredentialID:     req.CredentialID,
		SharedWithUserID: req.SharedWithUserID,
		SharedByUserID:   userID,
		Permission:       req.Permission,
		Note:             req.Note,
		ExpiresAt:        expiresAt,
	})
	if err != nil {
		HandleErrorL(w, r, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionCredentialShareCreate, domain.AuditResourceCredentialShare, &share.ID, map[string]interface{}{
		"credential_id":       req.CredentialID,
		"shared_with_user_id": req.SharedWithUserID,
		"permission":          string(req.Permission),
	})

	JSON(w, http.StatusCreated, usecase.ToShareResponse(share))
}

// ShareWithProjectRequest represents the request body for sharing with a project
type ShareWithProjectRequest struct {
	CredentialID        uuid.UUID              `json:"credential_id"`
	SharedWithProjectID uuid.UUID              `json:"shared_with_project_id"`
	Permission          domain.SharePermission `json:"permission"`
	Note                string                 `json:"note,omitempty"`
	ExpiresAt           *string                `json:"expires_at,omitempty"` // RFC3339 format
}

// ShareWithProject shares a credential with a project
func (h *CredentialShareHandler) ShareWithProject(w http.ResponseWriter, r *http.Request) {
	var req ShareWithProjectRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	if req.CredentialID == uuid.Nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "credential_id is required", nil)
		return
	}
	if req.SharedWithProjectID == uuid.Nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "shared_with_project_id is required", nil)
		return
	}
	if !req.Permission.IsValid() {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid permission: must be use, edit, or admin", nil)
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
	userID := getUserID(r)

	share, err := h.service.ShareWithProject(r.Context(), usecase.ShareWithProjectInput{
		TenantID:            tenantID,
		CredentialID:        req.CredentialID,
		SharedWithProjectID: req.SharedWithProjectID,
		SharedByUserID:      userID,
		Permission:          req.Permission,
		Note:                req.Note,
		ExpiresAt:           expiresAt,
	})
	if err != nil {
		HandleErrorL(w, r, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionCredentialShareCreate, domain.AuditResourceCredentialShare, &share.ID, map[string]interface{}{
		"credential_id":          req.CredentialID,
		"shared_with_project_id": req.SharedWithProjectID,
		"permission":             string(req.Permission),
	})

	JSON(w, http.StatusCreated, usecase.ToShareResponse(share))
}

// UpdateShareRequest represents the request body for updating a share
type UpdateShareRequest struct {
	Permission *domain.SharePermission `json:"permission,omitempty"`
	Note       *string                 `json:"note,omitempty"`
	ExpiresAt  *string                 `json:"expires_at,omitempty"` // RFC3339 format
}

// UpdateShare updates a credential share
func (h *CredentialShareHandler) UpdateShare(w http.ResponseWriter, r *http.Request) {
	shareID, ok := parseUUID(w, r, "share_id", "share ID")
	if !ok {
		return
	}

	var req UpdateShareRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	if req.Permission != nil && !req.Permission.IsValid() {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid permission: must be use, edit, or admin", nil)
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
	userID := getUserID(r)

	share, err := h.service.UpdateShare(r.Context(), usecase.UpdateShareInput{
		TenantID:   tenantID,
		ShareID:    shareID,
		UserID:     userID,
		Permission: req.Permission,
		Note:       req.Note,
		ExpiresAt:  expiresAt,
	})
	if err != nil {
		HandleErrorL(w, r, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionCredentialShareUpdate, domain.AuditResourceCredentialShare, &share.ID, map[string]interface{}{
		"credential_id": share.CredentialID,
	})

	JSON(w, http.StatusOK, usecase.ToShareResponse(share))
}

// RevokeShare revokes a credential share
func (h *CredentialShareHandler) RevokeShare(w http.ResponseWriter, r *http.Request) {
	shareID, ok := parseUUID(w, r, "share_id", "share ID")
	if !ok {
		return
	}

	tenantID := getTenantID(r)
	userID := getUserID(r)

	if err := h.service.RevokeShare(r.Context(), tenantID, shareID, userID); err != nil {
		HandleErrorL(w, r, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionCredentialShareDelete, domain.AuditResourceCredentialShare, &shareID, nil)

	w.WriteHeader(http.StatusNoContent)
}

// ListByCredential returns all shares for a credential
func (h *CredentialShareHandler) ListByCredential(w http.ResponseWriter, r *http.Request) {
	credentialID, ok := parseUUID(w, r, "credential_id", "credential ID")
	if !ok {
		return
	}

	tenantID := getTenantID(r)
	userID := getUserID(r)

	shares, err := h.service.ListByCredential(r.Context(), tenantID, credentialID, userID)
	if err != nil {
		HandleErrorL(w, r, err)
		return
	}

	JSON(w, http.StatusOK, usecase.ToShareResponses(shares))
}

// ListMyShares returns all credentials shared with the current user
func (h *CredentialShareHandler) ListMyShares(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	shares, err := h.service.ListByUser(r.Context(), userID)
	if err != nil {
		HandleErrorL(w, r, err)
		return
	}

	JSON(w, http.StatusOK, usecase.ToShareResponses(shares))
}

// ListByProject returns all credentials shared with a project
func (h *CredentialShareHandler) ListByProject(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseUUID(w, r, "project_id", "project ID")
	if !ok {
		return
	}

	shares, err := h.service.ListByProject(r.Context(), projectID)
	if err != nil {
		HandleErrorL(w, r, err)
		return
	}

	JSON(w, http.StatusOK, usecase.ToShareResponses(shares))
}

// GetShare returns a share by ID
func (h *CredentialShareHandler) GetShare(w http.ResponseWriter, r *http.Request) {
	shareID, ok := parseUUID(w, r, "share_id", "share ID")
	if !ok {
		return
	}

	share, err := h.service.GetShareByID(r.Context(), shareID)
	if err != nil {
		HandleErrorL(w, r, err)
		return
	}

	JSON(w, http.StatusOK, usecase.ToShareResponse(share))
}
