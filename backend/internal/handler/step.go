package handler

import (
	"encoding/json"
	"net/http"

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
	Name               string          `json:"name"`
	Type               string          `json:"type"`
	Config             json.RawMessage `json:"config"`
	TriggerType        string          `json:"trigger_type,omitempty"`        // For start blocks: manual, webhook, schedule
	TriggerConfig      json.RawMessage `json:"trigger_config,omitempty"`      // Configuration for the trigger
	CredentialBindings json.RawMessage `json:"credential_bindings,omitempty"` // Mapping of credential names to credential IDs
	Position           struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"position"`
}

// validateCredentialBindings validates the credential_bindings JSON structure
// Expected format: {"credential_name": "credential_uuid", ...}
func validateCredentialBindings(data json.RawMessage) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	var bindings map[string]string
	if err := json.Unmarshal(data, &bindings); err != nil {
		return domain.ErrValidation
	}
	// Validate UUID format for each credential ID
	for _, credID := range bindings {
		if credID == "" {
			continue
		}
		if _, err := uuid.Parse(credID); err != nil {
			return domain.ErrValidation
		}
	}
	return nil
}

// Create handles POST /api/v1/projects/{id}/steps
func (h *StepHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	projectID, ok := parseUUID(w, r, "id", "project ID")
	if !ok {
		return
	}

	var req CreateStepRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	// Validate credential_bindings format
	if err := validateCredentialBindings(req.CredentialBindings); err != nil {
		HandleError(w, err)
		return
	}

	step, err := h.stepUsecase.Create(r.Context(), usecase.CreateStepInput{
		TenantID:           tenantID,
		ProjectID:          projectID,
		Name:               req.Name,
		Type:               domain.StepType(req.Type),
		Config:             req.Config,
		TriggerType:        req.TriggerType,
		TriggerConfig:      req.TriggerConfig,
		CredentialBindings: req.CredentialBindings,
		PositionX:          req.Position.X,
		PositionY:          req.Position.Y,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusCreated, step)
}

// List handles GET /api/v1/projects/{project_id}/steps
func (h *StepHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	projectID, ok := parseUUID(w, r, "id", "project ID")
	if !ok {
		return
	}

	steps, err := h.stepUsecase.List(r.Context(), tenantID, projectID)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, steps)
}

// UpdateStepRequest represents an update step request
type UpdateStepRequest struct {
	Name               string          `json:"name"`
	Type               string          `json:"type"`
	Config             json.RawMessage `json:"config"`
	TriggerType        string          `json:"trigger_type,omitempty"`        // For start blocks: manual, webhook, schedule
	TriggerConfig      json.RawMessage `json:"trigger_config,omitempty"`      // Configuration for the trigger
	CredentialBindings json.RawMessage `json:"credential_bindings,omitempty"` // Mapping of credential names to credential IDs
	Position           *struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"position"`
}

// Update handles PUT /api/v1/projects/{project_id}/steps/{step_id}
func (h *StepHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	projectID, ok := parseUUID(w, r, "id", "project ID")
	if !ok {
		return
	}
	stepID, ok := parseUUID(w, r, "step_id", "step ID")
	if !ok {
		return
	}

	var req UpdateStepRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	// Validate credential_bindings format
	if err := validateCredentialBindings(req.CredentialBindings); err != nil {
		HandleError(w, err)
		return
	}

	input := usecase.UpdateStepInput{
		TenantID:           tenantID,
		ProjectID:          projectID,
		StepID:             stepID,
		Name:               req.Name,
		Type:               domain.StepType(req.Type),
		Config:             req.Config,
		TriggerType:        req.TriggerType,
		TriggerConfig:      req.TriggerConfig,
		CredentialBindings: req.CredentialBindings,
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

// Delete handles DELETE /api/v1/projects/{project_id}/steps/{step_id}
func (h *StepHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	projectID, ok := parseUUID(w, r, "id", "project ID")
	if !ok {
		return
	}
	stepID, ok := parseUUID(w, r, "step_id", "step ID")
	if !ok {
		return
	}

	if err := h.stepUsecase.Delete(r.Context(), tenantID, projectID, stepID); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
