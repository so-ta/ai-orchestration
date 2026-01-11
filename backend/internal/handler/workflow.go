package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/usecase"
)

// WorkflowHandler handles workflow HTTP requests
type WorkflowHandler struct {
	workflowUsecase *usecase.WorkflowUsecase
}

// NewWorkflowHandler creates a new WorkflowHandler
func NewWorkflowHandler(workflowUsecase *usecase.WorkflowUsecase) *WorkflowHandler {
	return &WorkflowHandler{workflowUsecase: workflowUsecase}
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
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
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow ID", nil)
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
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow ID", nil)
		return
	}

	var req UpdateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
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

	JSONData(w, http.StatusOK, workflow)
}

// Delete handles DELETE /api/v1/workflows/{id}
func (h *WorkflowHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow ID", nil)
		return
	}

	if err := h.workflowUsecase.Delete(r.Context(), tenantID, id); err != nil {
		HandleError(w, err)
		return
	}

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

// StepData represents step data in save request
type StepData struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Type      string          `json:"type"`
	Config    json.RawMessage `json:"config"`
	PositionX int             `json:"position_x"`
	PositionY int             `json:"position_y"`
}

// EdgeData represents edge data in save request
type EdgeData struct {
	ID           string  `json:"id"`
	SourceStepID string  `json:"source_step_id"`
	TargetStepID string  `json:"target_step_id"`
	Condition    *string `json:"condition"`
}

// Save handles POST /api/v1/workflows/{id}/save
// Creates a new version and saves the workflow
func (h *WorkflowHandler) Save(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow ID", nil)
		return
	}

	var req SaveWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	// Convert step data to domain steps
	steps := make([]domain.Step, len(req.Steps))
	for i, s := range req.Steps {
		stepID, err := uuid.Parse(s.ID)
		if err != nil {
			Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid step ID: "+s.ID, nil)
			return
		}
		steps[i] = domain.Step{
			ID:        stepID,
			Name:      s.Name,
			Type:      domain.StepType(s.Type),
			Config:    s.Config,
			PositionX: s.PositionX,
			PositionY: s.PositionY,
		}
	}

	// Convert edge data to domain edges
	edges := make([]domain.Edge, len(req.Edges))
	for i, e := range req.Edges {
		edgeID, err := uuid.Parse(e.ID)
		if err != nil {
			Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid edge ID: "+e.ID, nil)
			return
		}
		sourceID, err := uuid.Parse(e.SourceStepID)
		if err != nil {
			Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid source step ID: "+e.SourceStepID, nil)
			return
		}
		targetID, err := uuid.Parse(e.TargetStepID)
		if err != nil {
			Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid target step ID: "+e.TargetStepID, nil)
			return
		}
		condition := ""
		if e.Condition != nil {
			condition = *e.Condition
		}
		edges[i] = domain.Edge{
			ID:           edgeID,
			SourceStepID: sourceID,
			TargetStepID: targetID,
			Condition:    condition,
		}
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

	JSONData(w, http.StatusOK, workflow)
}

// SaveDraft handles POST /api/v1/workflows/{id}/draft
// Saves the workflow as draft without creating a new version
func (h *WorkflowHandler) SaveDraft(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow ID", nil)
		return
	}

	var req SaveWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	// Convert step data to domain steps
	steps := make([]domain.Step, len(req.Steps))
	for i, s := range req.Steps {
		stepID, err := uuid.Parse(s.ID)
		if err != nil {
			Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid step ID: "+s.ID, nil)
			return
		}
		steps[i] = domain.Step{
			ID:        stepID,
			Name:      s.Name,
			Type:      domain.StepType(s.Type),
			Config:    s.Config,
			PositionX: s.PositionX,
			PositionY: s.PositionY,
		}
	}

	// Convert edge data to domain edges
	edges := make([]domain.Edge, len(req.Edges))
	for i, e := range req.Edges {
		edgeID, err := uuid.Parse(e.ID)
		if err != nil {
			Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid edge ID: "+e.ID, nil)
			return
		}
		sourceID, err := uuid.Parse(e.SourceStepID)
		if err != nil {
			Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid source step ID: "+e.SourceStepID, nil)
			return
		}
		targetID, err := uuid.Parse(e.TargetStepID)
		if err != nil {
			Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid target step ID: "+e.TargetStepID, nil)
			return
		}
		condition := ""
		if e.Condition != nil {
			condition = *e.Condition
		}
		edges[i] = domain.Edge{
			ID:           edgeID,
			SourceStepID: sourceID,
			TargetStepID: targetID,
			Condition:    condition,
		}
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
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow ID", nil)
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
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow ID", nil)
		return
	}

	var req RestoreVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
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
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow ID", nil)
		return
	}

	workflow, err := h.workflowUsecase.Publish(r.Context(), tenantID, id)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, workflow)
}

// ListVersions handles GET /api/v1/workflows/{id}/versions
func (h *WorkflowHandler) ListVersions(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow ID", nil)
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
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow ID", nil)
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
