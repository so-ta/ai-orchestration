package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/usecase"
)

// RunHandler handles run HTTP requests
type RunHandler struct {
	runUsecase   *usecase.RunUsecase
	auditService *usecase.AuditService
}

// NewRunHandler creates a new RunHandler
func NewRunHandler(runUsecase *usecase.RunUsecase, auditService *usecase.AuditService) *RunHandler {
	return &RunHandler{
		runUsecase:   runUsecase,
		auditService: auditService,
	}
}

// CreateRunRequest represents a create run request
type CreateRunRequest struct {
	Input       json.RawMessage `json:"input"`
	TriggeredBy string          `json:"triggered_by,omitempty"` // manual, webhook, schedule, test, internal
	Mode        string          `json:"mode,omitempty"`         // Deprecated: use triggered_by instead (backward compat: "test" maps to triggered_by="test")
	Version     int             `json:"version,omitempty"`      // 0 or omitted means latest
}

// RunWithDefinitionResponse represents a run response with workflow definition
type RunWithDefinitionResponse struct {
	*domain.Run
	WorkflowDefinition interface{} `json:"workflow_definition,omitempty"`
}

// Create handles POST /api/v1/workflows/{workflow_id}/runs
func (h *RunHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	workflowID, ok := parseUUID(w, r, "id", "workflow ID")
	if !ok {
		return
	}

	var req CreateRunRequest
	if !decodeJSONBody(w, r, &req) {
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

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionRunCreate, domain.AuditResourceRun, &run.ID, map[string]interface{}{
		"workflow_id":  workflowID,
		"triggered_by": string(triggeredBy),
	})

	JSONData(w, http.StatusCreated, run)
}

// List handles GET /api/v1/workflows/{workflow_id}/runs
func (h *RunHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	workflowID, ok := parseUUID(w, r, "id", "workflow ID")
	if !ok {
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
	runID, ok := parseUUID(w, r, "run_id", "run ID")
	if !ok {
		return
	}

	output, err := h.runUsecase.GetWithDetailsAndDefinition(r.Context(), tenantID, runID)
	if err != nil {
		HandleError(w, err)
		return
	}

	response := &RunWithDefinitionResponse{
		Run:                output.Run,
		WorkflowDefinition: output.WorkflowDefinition,
	}

	JSONData(w, http.StatusOK, response)
}

// Cancel handles POST /api/v1/runs/{run_id}/cancel
func (h *RunHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	runID, ok := parseUUID(w, r, "run_id", "run ID")
	if !ok {
		return
	}

	run, err := h.runUsecase.Cancel(r.Context(), tenantID, runID)
	if err != nil {
		HandleError(w, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionRunCancel, domain.AuditResourceRun, &runID, nil)

	JSONData(w, http.StatusOK, run)
}

// ExecuteSingleStepRequest represents a request to execute a single step
type ExecuteSingleStepRequest struct {
	Input json.RawMessage `json:"input,omitempty"` // Optional: custom input (nil means use previous input)
}

// ExecuteSingleStep handles POST /api/v1/runs/{run_id}/steps/{step_id}/execute
func (h *RunHandler) ExecuteSingleStep(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	runID, ok := parseUUID(w, r, "run_id", "run ID")
	if !ok {
		return
	}
	stepID, ok := parseUUID(w, r, "step_id", "step ID")
	if !ok {
		return
	}

	var req ExecuteSingleStepRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
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
	runID, ok := parseUUID(w, r, "run_id", "run ID")
	if !ok {
		return
	}

	var req ResumeFromStepRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	fromStepID, ok := parseUUIDString(w, req.FromStepID, "from_step_id")
	if !ok {
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
	runID, ok := parseUUID(w, r, "run_id", "run ID")
	if !ok {
		return
	}
	stepID, ok := parseUUID(w, r, "step_id", "step ID")
	if !ok {
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
	workflowID, ok := parseUUID(w, r, "id", "workflow ID")
	if !ok {
		return
	}
	stepID, ok := parseUUID(w, r, "step_id", "step ID")
	if !ok {
		return
	}

	var req TestStepInlineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
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
