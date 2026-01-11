package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/usecase"
)

// EdgeHandler handles edge HTTP requests
type EdgeHandler struct {
	edgeUsecase *usecase.EdgeUsecase
}

// NewEdgeHandler creates a new EdgeHandler
func NewEdgeHandler(edgeUsecase *usecase.EdgeUsecase) *EdgeHandler {
	return &EdgeHandler{edgeUsecase: edgeUsecase}
}

// CreateEdgeRequest represents a create edge request
type CreateEdgeRequest struct {
	SourceStepID string `json:"source_step_id"`
	TargetStepID string `json:"target_step_id"`
	SourcePort   string `json:"source_port"`
	TargetPort   string `json:"target_port"`
	Condition    string `json:"condition"`
}

// Create handles POST /api/v1/workflows/{workflow_id}/edges
func (h *EdgeHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	workflowID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow ID", nil)
		return
	}

	var req CreateEdgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	sourceID, err := uuid.Parse(req.SourceStepID)
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid source_step_id", nil)
		return
	}
	targetID, err := uuid.Parse(req.TargetStepID)
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid target_step_id", nil)
		return
	}

	edge, err := h.edgeUsecase.Create(r.Context(), usecase.CreateEdgeInput{
		TenantID:     tenantID,
		WorkflowID:   workflowID,
		SourceStepID: sourceID,
		TargetStepID: targetID,
		SourcePort:   req.SourcePort,
		TargetPort:   req.TargetPort,
		Condition:    req.Condition,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusCreated, edge)
}

// List handles GET /api/v1/workflows/{workflow_id}/edges
func (h *EdgeHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	workflowID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow ID", nil)
		return
	}

	edges, err := h.edgeUsecase.List(r.Context(), tenantID, workflowID)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, edges)
}

// Delete handles DELETE /api/v1/workflows/{workflow_id}/edges/{edge_id}
func (h *EdgeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	workflowID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow ID", nil)
		return
	}
	edgeID, err := uuid.Parse(chi.URLParam(r, "edge_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid edge ID", nil)
		return
	}

	if err := h.edgeUsecase.Delete(r.Context(), tenantID, workflowID, edgeID); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
