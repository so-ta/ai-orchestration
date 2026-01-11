package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/usecase"
)

// StepHandler handles step HTTP requests
type StepHandler struct {
	stepUsecase *usecase.StepUsecase
}

// NewStepHandler creates a new StepHandler
func NewStepHandler(stepUsecase *usecase.StepUsecase) *StepHandler {
	return &StepHandler{stepUsecase: stepUsecase}
}

// CreateStepRequest represents a create step request
type CreateStepRequest struct {
	Name     string          `json:"name"`
	Type     string          `json:"type"`
	Config   json.RawMessage `json:"config"`
	Position struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"position"`
}

// Create handles POST /api/v1/workflows/{id}/steps
func (h *StepHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	workflowID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow ID", nil)
		return
	}

	var req CreateStepRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	step, err := h.stepUsecase.Create(r.Context(), usecase.CreateStepInput{
		TenantID:   tenantID,
		WorkflowID: workflowID,
		Name:       req.Name,
		Type:       domain.StepType(req.Type),
		Config:     req.Config,
		PositionX:  req.Position.X,
		PositionY:  req.Position.Y,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusCreated, step)
}

// List handles GET /api/v1/workflows/{workflow_id}/steps
func (h *StepHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	workflowID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow ID", nil)
		return
	}

	steps, err := h.stepUsecase.List(r.Context(), tenantID, workflowID)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, steps)
}

// UpdateStepRequest represents an update step request
type UpdateStepRequest struct {
	Name     string          `json:"name"`
	Type     string          `json:"type"`
	Config   json.RawMessage `json:"config"`
	Position *struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"position"`
}

// Update handles PUT /api/v1/workflows/{workflow_id}/steps/{step_id}
func (h *StepHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	workflowID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow ID", nil)
		return
	}
	stepID, err := uuid.Parse(chi.URLParam(r, "step_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid step ID", nil)
		return
	}

	var req UpdateStepRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	input := usecase.UpdateStepInput{
		TenantID:   tenantID,
		WorkflowID: workflowID,
		StepID:     stepID,
		Name:       req.Name,
		Type:       domain.StepType(req.Type),
		Config:     req.Config,
	}
	if req.Position != nil {
		input.PositionX = &req.Position.X
		input.PositionY = &req.Position.Y
	}

	step, err := h.stepUsecase.Update(r.Context(), input)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, step)
}

// Delete handles DELETE /api/v1/workflows/{workflow_id}/steps/{step_id}
func (h *StepHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	workflowID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow ID", nil)
		return
	}
	stepID, err := uuid.Parse(chi.URLParam(r, "step_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid step ID", nil)
		return
	}

	if err := h.stepUsecase.Delete(r.Context(), tenantID, workflowID, stepID); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
