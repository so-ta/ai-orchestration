package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/usecase"
)

// BlockGroupHandler handles block group HTTP requests
type BlockGroupHandler struct {
	blockGroupUsecase *usecase.BlockGroupUsecase
}

// NewBlockGroupHandler creates a new BlockGroupHandler
func NewBlockGroupHandler(blockGroupUsecase *usecase.BlockGroupUsecase) *BlockGroupHandler {
	return &BlockGroupHandler{blockGroupUsecase: blockGroupUsecase}
}

// CreateBlockGroupRequest represents a create block group request
type CreateBlockGroupRequest struct {
	Name          string          `json:"name"`
	Type          string          `json:"type"`
	Config        json.RawMessage `json:"config"`
	ParentGroupID *string         `json:"parent_group_id,omitempty"`
	Position      struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"position"`
	Size struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"size"`
}

// Create handles POST /api/v1/workflows/{id}/block-groups
func (h *BlockGroupHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	workflowID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow ID", nil)
		return
	}

	var req CreateBlockGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	var parentGroupID *uuid.UUID
	if req.ParentGroupID != nil {
		id, err := uuid.Parse(*req.ParentGroupID)
		if err != nil {
			Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid parent group ID", nil)
			return
		}
		parentGroupID = &id
	}

	group, err := h.blockGroupUsecase.Create(r.Context(), usecase.CreateBlockGroupInput{
		TenantID:      tenantID,
		WorkflowID:    workflowID,
		Name:          req.Name,
		Type:          domain.BlockGroupType(req.Type),
		Config:        req.Config,
		ParentGroupID: parentGroupID,
		PositionX:     req.Position.X,
		PositionY:     req.Position.Y,
		Width:         req.Size.Width,
		Height:        req.Size.Height,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusCreated, group)
}

// List handles GET /api/v1/workflows/{id}/block-groups
func (h *BlockGroupHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	workflowID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow ID", nil)
		return
	}

	groups, err := h.blockGroupUsecase.List(r.Context(), tenantID, workflowID)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, groups)
}

// Get handles GET /api/v1/workflows/{id}/block-groups/{group_id}
func (h *BlockGroupHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	workflowID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow ID", nil)
		return
	}
	groupID, err := uuid.Parse(chi.URLParam(r, "group_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid group ID", nil)
		return
	}

	group, err := h.blockGroupUsecase.GetByID(r.Context(), tenantID, workflowID, groupID)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, group)
}

// UpdateBlockGroupRequest represents an update block group request
type UpdateBlockGroupRequest struct {
	Name          string          `json:"name,omitempty"`
	Config        json.RawMessage `json:"config,omitempty"`
	ParentGroupID *string         `json:"parent_group_id,omitempty"`
	Position      *struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"position,omitempty"`
	Size *struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"size,omitempty"`
}

// Update handles PUT /api/v1/workflows/{id}/block-groups/{group_id}
func (h *BlockGroupHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	workflowID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow ID", nil)
		return
	}
	groupID, err := uuid.Parse(chi.URLParam(r, "group_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid group ID", nil)
		return
	}

	var req UpdateBlockGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	input := usecase.UpdateBlockGroupInput{
		TenantID:   tenantID,
		WorkflowID: workflowID,
		GroupID:    groupID,
		Name:       req.Name,
		Config:     req.Config,
	}

	if req.ParentGroupID != nil {
		id, err := uuid.Parse(*req.ParentGroupID)
		if err != nil {
			Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid parent group ID", nil)
			return
		}
		input.ParentGroupID = &id
	}

	if req.Position != nil {
		input.PositionX = &req.Position.X
		input.PositionY = &req.Position.Y
	}

	if req.Size != nil {
		input.Width = &req.Size.Width
		input.Height = &req.Size.Height
	}

	group, err := h.blockGroupUsecase.Update(r.Context(), input)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, group)
}

// Delete handles DELETE /api/v1/workflows/{id}/block-groups/{group_id}
func (h *BlockGroupHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	workflowID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow ID", nil)
		return
	}
	groupID, err := uuid.Parse(chi.URLParam(r, "group_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid group ID", nil)
		return
	}

	if err := h.blockGroupUsecase.Delete(r.Context(), tenantID, workflowID, groupID); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AddStepToGroupRequest represents a request to add a step to a block group
type AddStepToGroupRequest struct {
	StepID    string `json:"step_id"`
	GroupRole string `json:"group_role"`
}

// AddStepToGroup handles POST /api/v1/workflows/{id}/block-groups/{group_id}/steps
func (h *BlockGroupHandler) AddStepToGroup(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	workflowID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow ID", nil)
		return
	}
	groupID, err := uuid.Parse(chi.URLParam(r, "group_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid group ID", nil)
		return
	}

	var req AddStepToGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	stepID, err := uuid.Parse(req.StepID)
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid step ID", nil)
		return
	}

	step, err := h.blockGroupUsecase.AddStepToGroup(r.Context(), usecase.AddStepToGroupInput{
		TenantID:   tenantID,
		WorkflowID: workflowID,
		StepID:     stepID,
		GroupID:    groupID,
		GroupRole:  domain.GroupRole(req.GroupRole),
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, step)
}

// RemoveStepFromGroup handles DELETE /api/v1/workflows/{id}/block-groups/{group_id}/steps/{step_id}
func (h *BlockGroupHandler) RemoveStepFromGroup(w http.ResponseWriter, r *http.Request) {
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

	step, err := h.blockGroupUsecase.RemoveStepFromGroup(r.Context(), tenantID, workflowID, stepID)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, step)
}

// GetStepsByGroup handles GET /api/v1/workflows/{id}/block-groups/{group_id}/steps
func (h *BlockGroupHandler) GetStepsByGroup(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	workflowID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow ID", nil)
		return
	}
	groupID, err := uuid.Parse(chi.URLParam(r, "group_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid group ID", nil)
		return
	}

	steps, err := h.blockGroupUsecase.GetStepsByGroup(r.Context(), tenantID, workflowID, groupID)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, steps)
}
