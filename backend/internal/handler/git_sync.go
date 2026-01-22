package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/usecase"
)

// GitSyncHandler handles git sync HTTP requests
type GitSyncHandler struct {
	gitSyncUsecase *usecase.GitSyncUsecase
	auditService   *usecase.AuditService
}

// NewGitSyncHandler creates a new GitSyncHandler
func NewGitSyncHandler(gitSyncUsecase *usecase.GitSyncUsecase, auditService *usecase.AuditService) *GitSyncHandler {
	return &GitSyncHandler{
		gitSyncUsecase: gitSyncUsecase,
		auditService:   auditService,
	}
}

// CreateGitSyncRequest represents a create git sync request
type CreateGitSyncRequest struct {
	ProjectID     string                   `json:"project_id"`
	RepositoryURL string                   `json:"repository_url"`
	Branch        string                   `json:"branch"`
	FilePath      string                   `json:"file_path"`
	SyncDirection domain.GitSyncDirection  `json:"sync_direction"`
	AutoSync      bool                     `json:"auto_sync"`
	CredentialsID *string                  `json:"credentials_id,omitempty"`
}

// Create handles POST /api/v1/git-sync
func (h *GitSyncHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	var req CreateGitSyncRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid project_id", nil)
		return
	}

	var credentialsID *uuid.UUID
	if req.CredentialsID != nil && *req.CredentialsID != "" {
		id, err := uuid.Parse(*req.CredentialsID)
		if err != nil {
			Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid credentials_id", nil)
			return
		}
		credentialsID = &id
	}

	gitSync, err := h.gitSyncUsecase.Create(r.Context(), usecase.CreateGitSyncInput{
		TenantID:      tenantID,
		ProjectID:     projectID,
		RepositoryURL: req.RepositoryURL,
		Branch:        req.Branch,
		FilePath:      req.FilePath,
		SyncDirection: req.SyncDirection,
		AutoSync:      req.AutoSync,
		CredentialsID: credentialsID,
	})
	if err != nil {
		HandleErrorL(w, r, err)
		return
	}

	JSONData(w, http.StatusCreated, gitSync)
}

// Get handles GET /api/v1/git-sync/{id}
func (h *GitSyncHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid git sync ID", nil)
		return
	}

	gitSync, err := h.gitSyncUsecase.GetByID(r.Context(), tenantID, id)
	if err != nil {
		HandleErrorL(w, r, err)
		return
	}

	JSONData(w, http.StatusOK, gitSync)
}

// GetByProject handles GET /api/v1/workflows/{id}/git-sync
func (h *GitSyncHandler) GetByProject(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	projectID, ok := parseUUID(w, r, "id", "project ID")
	if !ok {
		return
	}

	gitSync, err := h.gitSyncUsecase.GetByProject(r.Context(), tenantID, projectID)
	if err != nil {
		HandleErrorL(w, r, err)
		return
	}

	JSONData(w, http.StatusOK, gitSync)
}

// List handles GET /api/v1/git-sync
func (h *GitSyncHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	gitSyncs, err := h.gitSyncUsecase.ListByTenant(r.Context(), tenantID)
	if err != nil {
		HandleErrorL(w, r, err)
		return
	}

	JSONData(w, http.StatusOK, gitSyncs)
}

// UpdateGitSyncRequest represents an update git sync request
type UpdateGitSyncRequest struct {
	RepositoryURL *string                  `json:"repository_url,omitempty"`
	Branch        *string                  `json:"branch,omitempty"`
	FilePath      *string                  `json:"file_path,omitempty"`
	SyncDirection *domain.GitSyncDirection `json:"sync_direction,omitempty"`
	AutoSync      *bool                    `json:"auto_sync,omitempty"`
	CredentialsID *string                  `json:"credentials_id,omitempty"`
}

// Update handles PUT /api/v1/git-sync/{id}
func (h *GitSyncHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid git sync ID", nil)
		return
	}

	var req UpdateGitSyncRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	var credentialsID *uuid.UUID
	if req.CredentialsID != nil && *req.CredentialsID != "" {
		parsed, err := uuid.Parse(*req.CredentialsID)
		if err != nil {
			Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid credentials_id", nil)
			return
		}
		credentialsID = &parsed
	}

	gitSync, err := h.gitSyncUsecase.Update(r.Context(), usecase.UpdateGitSyncInput{
		ID:            id,
		TenantID:      tenantID,
		RepositoryURL: req.RepositoryURL,
		Branch:        req.Branch,
		FilePath:      req.FilePath,
		SyncDirection: req.SyncDirection,
		AutoSync:      req.AutoSync,
		CredentialsID: credentialsID,
	})
	if err != nil {
		HandleErrorL(w, r, err)
		return
	}

	JSONData(w, http.StatusOK, gitSync)
}

// Delete handles DELETE /api/v1/git-sync/{id}
func (h *GitSyncHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid git sync ID", nil)
		return
	}

	if err := h.gitSyncUsecase.Delete(r.Context(), tenantID, id); err != nil {
		HandleErrorL(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// TriggerSyncRequest represents a trigger sync request
type TriggerSyncRequest struct {
	Operation string `json:"operation"` // "push" or "pull"
}

// TriggerSync handles POST /api/v1/git-sync/{id}/sync
func (h *GitSyncHandler) TriggerSync(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid git sync ID", nil)
		return
	}

	var req TriggerSyncRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	syncOp, err := h.gitSyncUsecase.TriggerSync(r.Context(), tenantID, id, req.Operation)
	if err != nil {
		HandleErrorL(w, r, err)
		return
	}

	JSONData(w, http.StatusAccepted, syncOp)
}
