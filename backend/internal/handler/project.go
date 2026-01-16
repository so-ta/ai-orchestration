package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/usecase"
)

// ProjectHandler handles project HTTP requests
type ProjectHandler struct {
	projectUsecase *usecase.ProjectUsecase
	auditService   *usecase.AuditService
}

// NewProjectHandler creates a new ProjectHandler
func NewProjectHandler(projectUsecase *usecase.ProjectUsecase, auditService *usecase.AuditService) *ProjectHandler {
	return &ProjectHandler{
		projectUsecase: projectUsecase,
		auditService:   auditService,
	}
}

// CreateProjectRequest represents a create project request
type CreateProjectRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Variables   json.RawMessage `json:"variables,omitempty"`
}

// Create handles POST /api/v1/projects
func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	var req CreateProjectRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	project, err := h.projectUsecase.Create(r.Context(), usecase.CreateProjectInput{
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		Variables:   req.Variables,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionProjectCreate, domain.AuditResourceProject, &project.ID, map[string]interface{}{
		"name": project.Name,
	})

	JSONData(w, http.StatusCreated, project)
}

// List handles GET /api/v1/projects
func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	// Parse query parameters
	page := parseIntQuery(r, "page", 1)
	limit := parseIntQuery(r, "limit", 20)
	statusStr := r.URL.Query().Get("status")

	var status *domain.ProjectStatus
	if statusStr != "" {
		s := domain.ProjectStatus(statusStr)
		status = &s
	}

	output, err := h.projectUsecase.List(r.Context(), usecase.ListProjectsInput{
		TenantID: tenantID,
		Status:   status,
		Page:     page,
		Limit:    limit,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONList(w, http.StatusOK, output.Projects, output.Page, output.Limit, output.Total)
}

// Get handles GET /api/v1/projects/{id}
func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, ok := parseUUID(w, r, "id", "project ID")
	if !ok {
		return
	}

	project, err := h.projectUsecase.GetWithDetails(r.Context(), tenantID, id)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, project)
}

// UpdateProjectRequest represents an update project request
type UpdateProjectRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Variables   json.RawMessage `json:"variables,omitempty"`
}

// Update handles PUT /api/v1/projects/{id}
func (h *ProjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, ok := parseUUID(w, r, "id", "project ID")
	if !ok {
		return
	}

	var req UpdateProjectRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	project, err := h.projectUsecase.Update(r.Context(), usecase.UpdateProjectInput{
		TenantID:    tenantID,
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Variables:   req.Variables,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionProjectUpdate, domain.AuditResourceProject, &project.ID, map[string]interface{}{
		"name": project.Name,
	})

	JSONData(w, http.StatusOK, project)
}

// Delete handles DELETE /api/v1/projects/{id}
func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, ok := parseUUID(w, r, "id", "project ID")
	if !ok {
		return
	}

	if err := h.projectUsecase.Delete(r.Context(), tenantID, id); err != nil {
		HandleError(w, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionProjectDelete, domain.AuditResourceProject, &id, nil)

	w.WriteHeader(http.StatusNoContent)
}

// SaveProjectRequest represents a save project request
type SaveProjectRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Variables   json.RawMessage `json:"variables,omitempty"`
	Steps       []StepData      `json:"steps"`
	Edges       []EdgeData      `json:"edges"`
}

// Save handles POST /api/v1/projects/{id}/save
// Creates a new version and saves the project
func (h *ProjectHandler) Save(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, ok := parseUUID(w, r, "id", "project ID")
	if !ok {
		return
	}

	var req SaveProjectRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	steps, ok := convertStepData(w, req.Steps)
	if !ok {
		return
	}

	edges, ok := convertEdgeData(w, req.Edges)
	if !ok {
		return
	}

	project, err := h.projectUsecase.Save(r.Context(), usecase.SaveProjectInput{
		TenantID:    tenantID,
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Variables:   req.Variables,
		Steps:       steps,
		Edges:       edges,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	// Log audit event (publish)
	logAudit(r.Context(), h.auditService, r, domain.AuditActionProjectPublish, domain.AuditResourceProject, &project.ID, map[string]interface{}{
		"name":    project.Name,
		"version": project.Version,
	})

	JSONData(w, http.StatusOK, project)
}

// SaveDraft handles POST /api/v1/projects/{id}/draft
// Saves the project as draft without creating a new version
func (h *ProjectHandler) SaveDraft(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, ok := parseUUID(w, r, "id", "project ID")
	if !ok {
		return
	}

	var req SaveProjectRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	steps, ok := convertStepData(w, req.Steps)
	if !ok {
		return
	}

	edges, ok := convertEdgeData(w, req.Edges)
	if !ok {
		return
	}

	project, err := h.projectUsecase.SaveDraft(r.Context(), usecase.SaveDraftInput{
		TenantID:    tenantID,
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Variables:   req.Variables,
		Steps:       steps,
		Edges:       edges,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, project)
}

// DiscardDraft handles DELETE /api/v1/projects/{id}/draft
// Discards the draft and returns the saved version
func (h *ProjectHandler) DiscardDraft(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, ok := parseUUID(w, r, "id", "project ID")
	if !ok {
		return
	}

	project, err := h.projectUsecase.DiscardDraft(r.Context(), tenantID, id)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, project)
}

// RestoreVersionRequest represents a restore version request
type RestoreVersionRequest struct {
	Version int `json:"version"`
}

// RestoreVersion handles POST /api/v1/projects/{id}/restore
// Restores a project to a specific version
func (h *ProjectHandler) RestoreVersion(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, ok := parseUUID(w, r, "id", "project ID")
	if !ok {
		return
	}

	var req RestoreVersionRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	if req.Version < 1 {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "version must be at least 1", nil)
		return
	}

	project, err := h.projectUsecase.RestoreVersion(r.Context(), tenantID, id, req.Version)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, project)
}

// Publish handles POST /api/v1/projects/{id}/publish
// Deprecated: Use Save instead. Kept for backward compatibility.
func (h *ProjectHandler) Publish(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, ok := parseUUID(w, r, "id", "project ID")
	if !ok {
		return
	}

	project, err := h.projectUsecase.Publish(r.Context(), tenantID, id)
	if err != nil {
		HandleError(w, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionProjectPublish, domain.AuditResourceProject, &project.ID, map[string]interface{}{
		"name":    project.Name,
		"version": project.Version,
	})

	JSONData(w, http.StatusOK, project)
}

// ListVersions handles GET /api/v1/projects/{id}/versions
func (h *ProjectHandler) ListVersions(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, ok := parseUUID(w, r, "id", "project ID")
	if !ok {
		return
	}

	versions, err := h.projectUsecase.ListVersions(r.Context(), tenantID, id)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, versions)
}

// GetVersion handles GET /api/v1/projects/{id}/versions/{version}
func (h *ProjectHandler) GetVersion(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, ok := parseUUID(w, r, "id", "project ID")
	if !ok {
		return
	}

	version := parseIntQuery(r, "version", 0)
	versionStr := chi.URLParam(r, "version")
	if versionStr != "" {
		version = parseIntFromString(versionStr, 0)
	}

	if version < 1 {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid version number", nil)
		return
	}

	projectVersion, err := h.projectUsecase.GetVersion(r.Context(), tenantID, id, version)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, projectVersion)
}
