package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/usecase"
)

// WorkflowHandler handles workflow HTTP requests
type WorkflowHandler struct {
	workflowUsecase *usecase.WorkflowUsecase
	auditService    *usecase.AuditService
}

// NewWorkflowHandler creates a new WorkflowHandler
func NewWorkflowHandler(workflowUsecase *usecase.WorkflowUsecase, auditService *usecase.AuditService) *WorkflowHandler {
	return &WorkflowHandler{
		workflowUsecase: workflowUsecase,
		auditService:    auditService,
	}
}

// CreateRequest represents a create workflow request
type CreateWorkflowRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// Create handles POST /api/v1/workflows
func (h *WorkflowHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	var req CreateWorkflowRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	workflow, err := h.workflowUsecase.Create(r.Context(), usecase.CreateWorkflowInput{
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		InputSchema: req.InputSchema,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionWorkflowCreate, domain.AuditResourceWorkflow, &workflow.ID, map[string]interface{}{
		"name": workflow.Name,
	})

	JSONData(w, http.StatusCreated, workflow)
}

// List handles GET /api/v1/workflows
func (h *WorkflowHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	// Parse query parameters
	page := parseIntQuery(r, "page", 1)
	limit := parseIntQuery(r, "limit", 20)
	statusStr := r.URL.Query().Get("status")

	var status *domain.WorkflowStatus
	if statusStr != "" {
		s := domain.WorkflowStatus(statusStr)
		status = &s
	}

	output, err := h.workflowUsecase.List(r.Context(), usecase.ListWorkflowsInput{
		TenantID: tenantID,
		Status:   status,
		Page:     page,
		Limit:    limit,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONList(w, http.StatusOK, output.Workflows, output.Page, output.Limit, output.Total)
}

// Get handles GET /api/v1/workflows/{id}
func (h *WorkflowHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, ok := parseUUID(w, r, "id", "workflow ID")
	if !ok {
		return
	}

	workflow, err := h.workflowUsecase.GetWithDetails(r.Context(), tenantID, id)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, workflow)
}

// UpdateRequest represents an update workflow request
type UpdateWorkflowRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// Update handles PUT /api/v1/workflows/{id}
func (h *WorkflowHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, ok := parseUUID(w, r, "id", "workflow ID")
	if !ok {
		return
	}

	var req UpdateWorkflowRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	workflow, err := h.workflowUsecase.Update(r.Context(), usecase.UpdateWorkflowInput{
		TenantID:    tenantID,
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		InputSchema: req.InputSchema,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionWorkflowUpdate, domain.AuditResourceWorkflow, &workflow.ID, map[string]interface{}{
		"name": workflow.Name,
	})

	JSONData(w, http.StatusOK, workflow)
}

// Delete handles DELETE /api/v1/workflows/{id}
func (h *WorkflowHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, ok := parseUUID(w, r, "id", "workflow ID")
	if !ok {
		return
	}

	if err := h.workflowUsecase.Delete(r.Context(), tenantID, id); err != nil {
		HandleError(w, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionWorkflowDelete, domain.AuditResourceWorkflow, &id, nil)

	w.WriteHeader(http.StatusNoContent)
}

// SaveWorkflowRequest represents a save workflow request
type SaveWorkflowRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
	Steps       []StepData      `json:"steps"`
	Edges       []EdgeData      `json:"edges"`
}

// Save handles POST /api/v1/workflows/{id}/save
// Creates a new version and saves the workflow
func (h *WorkflowHandler) Save(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, ok := parseUUID(w, r, "id", "workflow ID")
	if !ok {
		return
	}

	var req SaveWorkflowRequest
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

	workflow, err := h.workflowUsecase.Save(r.Context(), usecase.SaveWorkflowInput{
		TenantID:    tenantID,
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		InputSchema: req.InputSchema,
		Steps:       steps,
		Edges:       edges,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	// Log audit event (publish)
	logAudit(r.Context(), h.auditService, r, domain.AuditActionWorkflowPublish, domain.AuditResourceWorkflow, &workflow.ID, map[string]interface{}{
		"name":    workflow.Name,
		"version": workflow.Version,
	})

	JSONData(w, http.StatusOK, workflow)
}

// SaveDraft handles POST /api/v1/workflows/{id}/draft
// Saves the workflow as draft without creating a new version
func (h *WorkflowHandler) SaveDraft(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, ok := parseUUID(w, r, "id", "workflow ID")
	if !ok {
		return
	}

	var req SaveWorkflowRequest
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

	workflow, err := h.workflowUsecase.SaveDraft(r.Context(), usecase.SaveDraftInput{
		TenantID:    tenantID,
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		InputSchema: req.InputSchema,
		Steps:       steps,
		Edges:       edges,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, workflow)
}

// DiscardDraft handles DELETE /api/v1/workflows/{id}/draft
// Discards the draft and returns the saved version
func (h *WorkflowHandler) DiscardDraft(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, ok := parseUUID(w, r, "id", "workflow ID")
	if !ok {
		return
	}

	workflow, err := h.workflowUsecase.DiscardDraft(r.Context(), tenantID, id)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, workflow)
}

// RestoreVersionRequest represents a restore version request
type RestoreVersionRequest struct {
	Version int `json:"version"`
}

// RestoreVersion handles POST /api/v1/workflows/{id}/restore
// Restores a workflow to a specific version
func (h *WorkflowHandler) RestoreVersion(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, ok := parseUUID(w, r, "id", "workflow ID")
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

	workflow, err := h.workflowUsecase.RestoreVersion(r.Context(), tenantID, id, req.Version)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, workflow)
}

// Publish handles POST /api/v1/workflows/{id}/publish
// Deprecated: Use Save instead. Kept for backward compatibility.
func (h *WorkflowHandler) Publish(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, ok := parseUUID(w, r, "id", "workflow ID")
	if !ok {
		return
	}

	workflow, err := h.workflowUsecase.Publish(r.Context(), tenantID, id)
	if err != nil {
		HandleError(w, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionWorkflowPublish, domain.AuditResourceWorkflow, &workflow.ID, map[string]interface{}{
		"name":    workflow.Name,
		"version": workflow.Version,
	})

	JSONData(w, http.StatusOK, workflow)
}

// ListVersions handles GET /api/v1/workflows/{id}/versions
func (h *WorkflowHandler) ListVersions(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, ok := parseUUID(w, r, "id", "workflow ID")
	if !ok {
		return
	}

	versions, err := h.workflowUsecase.ListVersions(r.Context(), tenantID, id)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, versions)
}

// GetVersion handles GET /api/v1/workflows/{id}/versions/{version}
func (h *WorkflowHandler) GetVersion(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, ok := parseUUID(w, r, "id", "workflow ID")
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

	workflowVersion, err := h.workflowUsecase.GetVersion(r.Context(), tenantID, id, version)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, workflowVersion)
}
