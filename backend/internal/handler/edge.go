package handler

import (
	"net/http"

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
// Either source_step_id or source_block_group_id must be provided
// Either target_step_id or target_block_group_id must be provided
type CreateEdgeRequest struct {
	SourceStepID       string `json:"source_step_id,omitempty"`
	TargetStepID       string `json:"target_step_id,omitempty"`
	SourceBlockGroupID string `json:"source_block_group_id,omitempty"`
	TargetBlockGroupID string `json:"target_block_group_id,omitempty"`
	SourcePort         string `json:"source_port"`
	Condition          string `json:"condition"`
}

// Create handles POST /api/v1/projects/{project_id}/edges
func (h *EdgeHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	projectID, ok := parseUUID(w, r, "id", "project ID")
	if !ok {
		return
	}

	var req CreateEdgeRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	input := usecase.CreateEdgeInput{
		TenantID:   tenantID,
		ProjectID:  projectID,
		SourcePort: req.SourcePort,
		Condition:  req.Condition,
	}

	// Parse source (step or group)
	if req.SourceStepID != "" {
		sourceID, ok := parseUUIDString(w, req.SourceStepID, "source_step_id")
		if !ok {
			return
		}
		input.SourceStepID = &sourceID
	} else if req.SourceBlockGroupID != "" {
		sourceGroupID, ok := parseUUIDString(w, req.SourceBlockGroupID, "source_block_group_id")
		if !ok {
			return
		}
		input.SourceBlockGroupID = &sourceGroupID
	} else {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "source_step_id or source_block_group_id is required", nil)
		return
	}

	// Parse target (step or group)
	if req.TargetStepID != "" {
		targetID, ok := parseUUIDString(w, req.TargetStepID, "target_step_id")
		if !ok {
			return
		}
		input.TargetStepID = &targetID
	} else if req.TargetBlockGroupID != "" {
		targetGroupID, ok := parseUUIDString(w, req.TargetBlockGroupID, "target_block_group_id")
		if !ok {
			return
		}
		input.TargetBlockGroupID = &targetGroupID
	} else {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "target_step_id or target_block_group_id is required", nil)
		return
	}

	edge, err := h.edgeUsecase.Create(r.Context(), input)
	if err != nil {
		HandleErrorL(w, r, err)
		return
	}

	JSONData(w, http.StatusCreated, edge)
}

// List handles GET /api/v1/projects/{project_id}/edges
func (h *EdgeHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	projectID, ok := parseUUID(w, r, "id", "project ID")
	if !ok {
		return
	}

	edges, err := h.edgeUsecase.List(r.Context(), tenantID, projectID)
	if err != nil {
		HandleErrorL(w, r, err)
		return
	}

	JSONData(w, http.StatusOK, edges)
}

// Delete handles DELETE /api/v1/projects/{project_id}/edges/{edge_id}
func (h *EdgeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	projectID, ok := parseUUID(w, r, "id", "project ID")
	if !ok {
		return
	}
	edgeID, ok := parseUUID(w, r, "edge_id", "edge ID")
	if !ok {
		return
	}

	if err := h.edgeUsecase.Delete(r.Context(), tenantID, projectID, edgeID); err != nil {
		HandleErrorL(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
