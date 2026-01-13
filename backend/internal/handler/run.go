package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/usecase"
)

// RunHandler handles run HTTP requests
type RunHandler struct {
	runUsecase *usecase.RunUsecase
}

// NewRunHandler creates a new RunHandler
func NewRunHandler(runUsecase *usecase.RunUsecase) *RunHandler {
	return &RunHandler{runUsecase: runUsecase}
}

// CreateRunRequest represents a create run request
type CreateRunRequest struct {
	Input       json.RawMessage `json:"input"`
	TriggeredBy string          `json:"triggered_by,omitempty"` // manual, webhook, schedule, test, internal
	Mode        string          `json:"mode,omitempty"`         // Deprecated: use triggered_by instead (backward compat: "test" maps to triggered_by="test")
	Version     int             `json:"version,omitempty"`      // 0 or omitted means latest
}

// Create handles POST /api/v1/workflows/{workflow_id}/runs
func (h *RunHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	workflowID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow ID", nil)
		return
	}

	var req CreateRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	// Determine triggered_by with backward compatibility for mode
	triggeredBy := domain.TriggerTypeManual // default
	if req.TriggeredBy != "" {
		// New API: use triggered_by directly
		switch req.TriggeredBy {
		case "test":
			triggeredBy = domain.TriggerTypeTest
		case "webhook":
			triggeredBy = domain.TriggerTypeWebhook
		case "schedule":
			triggeredBy = domain.TriggerTypeSchedule
		case "internal":
			triggeredBy = domain.TriggerTypeInternal
		default:
			triggeredBy = domain.TriggerTypeManual
		}
	} else if req.Mode == "test" {
		// Backward compatibility: mode: "test" maps to triggered_by: "test"
		triggeredBy = domain.TriggerTypeTest
	}

	var userIDPtr *uuid.UUID
	if userID := getUserID(r); userID != uuid.Nil {
		userIDPtr = &userID
	}

	run, err := h.runUsecase.Create(r.Context(), usecase.CreateRunInput{
		TenantID:    tenantID,
		WorkflowID:  workflowID,
		Version:     req.Version,
		Input:       req.Input,
		TriggeredBy: triggeredBy,
		UserID:      userIDPtr,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusCreated, run)
}

// List handles GET /api/v1/workflows/{workflow_id}/runs
func (h *RunHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	workflowID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid workflow ID", nil)
		return
	}

	page := parseIntQuery(r, "page", 1)
	limit := parseIntQuery(r, "limit", 20)

	output, err := h.runUsecase.List(r.Context(), usecase.ListRunsInput{
		TenantID:   tenantID,
		WorkflowID: workflowID,
		Page:       page,
		Limit:      limit,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONList(w, http.StatusOK, output.Runs, output.Page, output.Limit, output.Total)
}

// Get handles GET /api/v1/runs/{run_id}
func (h *RunHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	runID, err := uuid.Parse(chi.URLParam(r, "run_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid run ID", nil)
		return
	}

	output, err := h.runUsecase.GetWithDetailsAndDefinition(r.Context(), tenantID, runID)
	if err != nil {
		HandleError(w, err)
		return
	}

	// Flatten the response to include workflow_definition at the same level as run fields
	response := make(map[string]interface{})
	// Include all run fields
	response["id"] = output.Run.ID
	response["tenant_id"] = output.Run.TenantID
	response["workflow_id"] = output.Run.WorkflowID
	response["workflow_version"] = output.Run.WorkflowVersion
	response["status"] = output.Run.Status
	response["run_number"] = output.Run.RunNumber
	response["input"] = output.Run.Input
	response["output"] = output.Run.Output
	response["error"] = output.Run.Error
	response["triggered_by"] = output.Run.TriggeredBy
	response["triggered_by_user"] = output.Run.TriggeredByUser
	response["started_at"] = output.Run.StartedAt
	response["completed_at"] = output.Run.CompletedAt
	response["created_at"] = output.Run.CreatedAt
	response["step_runs"] = output.Run.StepRuns
	// Include workflow definition if available
	if output.WorkflowDefinition != nil {
		response["workflow_definition"] = output.WorkflowDefinition
	}

	JSONData(w, http.StatusOK, response)
}

// Cancel handles POST /api/v1/runs/{run_id}/cancel
func (h *RunHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	runID, err := uuid.Parse(chi.URLParam(r, "run_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid run ID", nil)
		return
	}

	run, err := h.runUsecase.Cancel(r.Context(), tenantID, runID)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, run)
}

// ExecuteSingleStepRequest represents a request to execute a single step
type ExecuteSingleStepRequest struct {
	Input json.RawMessage `json:"input,omitempty"` // Optional: custom input (nil means use previous input)
}

// ExecuteSingleStep handles POST /api/v1/runs/{run_id}/steps/{step_id}/execute
func (h *RunHandler) ExecuteSingleStep(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	runID, err := uuid.Parse(chi.URLParam(r, "run_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid run ID", nil)
		return
	}
	stepID, err := uuid.Parse(chi.URLParam(r, "step_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid step ID", nil)
		return
	}

	var req ExecuteSingleStepRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	stepRun, err := h.runUsecase.ExecuteSingleStep(r.Context(), usecase.ExecuteSingleStepInput{
		TenantID: tenantID,
		RunID:    runID,
		StepID:   stepID,
		Input:    req.Input,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusAccepted, stepRun)
}

// ResumeFromStepRequest represents a request to resume execution from a step
type ResumeFromStepRequest struct {
	FromStepID    string          `json:"from_step_id"`               // Starting step ID
	InputOverride json.RawMessage `json:"input_override,omitempty"`   // Optional: override input for the starting step
}

// ResumeFromStep handles POST /api/v1/runs/{run_id}/resume
func (h *RunHandler) ResumeFromStep(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	runID, err := uuid.Parse(chi.URLParam(r, "run_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid run ID", nil)
		return
	}

	var req ResumeFromStepRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	fromStepID, err := uuid.Parse(req.FromStepID)
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid from_step_id", nil)
		return
	}

	output, err := h.runUsecase.ResumeFromStep(r.Context(), usecase.ResumeFromStepInput{
		TenantID:      tenantID,
		RunID:         runID,
		FromStepID:    fromStepID,
		InputOverride: req.InputOverride,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusAccepted, output)
}

// GetStepHistory handles GET /api/v1/runs/{run_id}/steps/{step_id}/history
func (h *RunHandler) GetStepHistory(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	runID, err := uuid.Parse(chi.URLParam(r, "run_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid run ID", nil)
		return
	}
	stepID, err := uuid.Parse(chi.URLParam(r, "step_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid step ID", nil)
		return
	}

	stepRuns, err := h.runUsecase.GetStepHistory(r.Context(), tenantID, runID, stepID)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, stepRuns)
}

// TestStepInlineRequest represents a request to test a step inline
type TestStepInlineRequest struct {
	Input json.RawMessage `json:"input"` // Custom input for testing
}

// TestStepInline handles POST /api/v1/workflows/{id}/steps/{step_id}/test
// This allows testing a single step without requiring an existing run
func (h *RunHandler) TestStepInline(w http.ResponseWriter, r *http.Request) {
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

	var req TestStepInlineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	// Set default empty input if not provided
	input := req.Input
	if input == nil {
		input = json.RawMessage(`{}`)
	}

	var userIDPtr *uuid.UUID
	if userID := getUserID(r); userID != uuid.Nil {
		userIDPtr = &userID
	}

	output, err := h.runUsecase.TestStepInline(r.Context(), usecase.TestStepInlineInput{
		TenantID:   tenantID,
		WorkflowID: workflowID,
		StepID:     stepID,
		Input:      input,
		UserID:     userIDPtr,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusAccepted, output)
}
