package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/middleware"
	"github.com/souta/ai-orchestration/internal/usecase"
)

// CopilotHandler handles copilot API requests
type CopilotHandler struct {
	usecase    *usecase.CopilotUsecase
	runUsecase *usecase.RunUsecase
}

// NewCopilotHandler creates a new CopilotHandler
func NewCopilotHandler(uc *usecase.CopilotUsecase, runUC *usecase.RunUsecase) *CopilotHandler {
	return &CopilotHandler{usecase: uc, runUsecase: runUC}
}

// SuggestRequest represents the request body for suggestions
type SuggestRequest struct {
	ProjectID string  `json:"project_id"`
	StepID    *string `json:"step_id,omitempty"`
	Context   string  `json:"context,omitempty"`
}

// Suggest handles POST /api/v1/copilot/suggest
func (h *CopilotHandler) Suggest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)

	var req SuggestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_PROJECT_ID", "Invalid project ID", nil)
		return
	}

	var stepID *uuid.UUID
	if req.StepID != nil {
		id, err := uuid.Parse(*req.StepID)
		if err != nil {
			Error(w, http.StatusBadRequest, "INVALID_STEP_ID", "Invalid step ID", nil)
			return
		}
		stepID = &id
	}

	input := usecase.SuggestInput{
		TenantID:  tenantID,
		ProjectID: projectID,
		StepID:    stepID,
		Context:   req.Context,
	}

	output, err := h.usecase.Suggest(ctx, input)
	if err != nil {
		Error(w, http.StatusInternalServerError, "SUGGEST_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusOK, output)
}

// DiagnoseRequest represents the request body for diagnosis
type DiagnoseRequest struct {
	RunID     string  `json:"run_id"`
	StepRunID *string `json:"step_run_id,omitempty"`
}

// Diagnose handles POST /api/v1/copilot/diagnose
func (h *CopilotHandler) Diagnose(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)

	var req DiagnoseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	runID, err := uuid.Parse(req.RunID)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_RUN_ID", "Invalid run ID", nil)
		return
	}

	var stepRunID *uuid.UUID
	if req.StepRunID != nil {
		id, err := uuid.Parse(*req.StepRunID)
		if err != nil {
			Error(w, http.StatusBadRequest, "INVALID_STEP_RUN_ID", "Invalid step run ID", nil)
			return
		}
		stepRunID = &id
	}

	input := usecase.DiagnoseInput{
		TenantID:  tenantID,
		RunID:     runID,
		StepRunID: stepRunID,
	}

	output, err := h.usecase.Diagnose(ctx, input)
	if err != nil {
		Error(w, http.StatusInternalServerError, "DIAGNOSE_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusOK, output)
}

// ExplainRequest represents the request body for explanation
type ExplainRequest struct {
	ProjectID string  `json:"project_id"`
	StepID    *string `json:"step_id,omitempty"`
}

// Explain handles POST /api/v1/copilot/explain
func (h *CopilotHandler) Explain(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)

	var req ExplainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_PROJECT_ID", "Invalid project ID", nil)
		return
	}

	var stepID *uuid.UUID
	if req.StepID != nil {
		id, err := uuid.Parse(*req.StepID)
		if err != nil {
			Error(w, http.StatusBadRequest, "INVALID_STEP_ID", "Invalid step ID", nil)
			return
		}
		stepID = &id
	}

	input := usecase.ExplainInput{
		TenantID:  tenantID,
		ProjectID: projectID,
		StepID:    stepID,
	}

	output, err := h.usecase.Explain(ctx, input)
	if err != nil {
		Error(w, http.StatusInternalServerError, "EXPLAIN_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusOK, output)
}

// OptimizeRequest represents the request body for optimization
type OptimizeRequest struct {
	ProjectID string `json:"project_id"`
}

// Optimize handles POST /api/v1/copilot/optimize
func (h *CopilotHandler) Optimize(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)

	var req OptimizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_PROJECT_ID", "Invalid project ID", nil)
		return
	}

	input := usecase.OptimizeInput{
		TenantID:  tenantID,
		ProjectID: projectID,
	}

	output, err := h.usecase.Optimize(ctx, input)
	if err != nil {
		Error(w, http.StatusInternalServerError, "OPTIMIZE_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusOK, output)
}

// ChatRequest represents the request body for chat
type ChatRequest struct {
	ProjectID *string `json:"project_id,omitempty"`
	Message   string  `json:"message"`
	Context   string  `json:"context,omitempty"`
}

// Chat handles POST /api/v1/copilot/chat
func (h *CopilotHandler) Chat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	if req.Message == "" {
		Error(w, http.StatusBadRequest, "EMPTY_MESSAGE", "Message cannot be empty", nil)
		return
	}

	var projectID *uuid.UUID
	if req.ProjectID != nil {
		id, err := uuid.Parse(*req.ProjectID)
		if err != nil {
			Error(w, http.StatusBadRequest, "INVALID_PROJECT_ID", "Invalid project ID", nil)
			return
		}
		projectID = &id
	}

	input := usecase.ChatInput{
		TenantID:  tenantID,
		ProjectID: projectID,
		Message:   req.Message,
		Context:   req.Context,
	}

	output, err := h.usecase.Chat(ctx, input)
	if err != nil {
		Error(w, http.StatusInternalServerError, "CHAT_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusOK, output)
}

// SuggestForStep handles POST /api/v1/projects/{project_id}/steps/{step_id}/copilot/suggest
func (h *CopilotHandler) SuggestForStep(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)

	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_PROJECT_ID", "Invalid project ID", nil)
		return
	}

	stepID, err := uuid.Parse(chi.URLParam(r, "step_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_STEP_ID", "Invalid step ID", nil)
		return
	}

	var req struct {
		Context string `json:"context,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	input := usecase.SuggestInput{
		TenantID:  tenantID,
		ProjectID: projectID,
		StepID:    &stepID,
		Context:   req.Context,
	}

	output, err := h.usecase.Suggest(ctx, input)
	if err != nil {
		Error(w, http.StatusInternalServerError, "SUGGEST_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusOK, output)
}

// ExplainStep handles POST /api/v1/projects/{project_id}/steps/{step_id}/copilot/explain
func (h *CopilotHandler) ExplainStep(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)

	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_PROJECT_ID", "Invalid project ID", nil)
		return
	}

	stepID, err := uuid.Parse(chi.URLParam(r, "step_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_STEP_ID", "Invalid step ID", nil)
		return
	}

	input := usecase.ExplainInput{
		TenantID:  tenantID,
		ProjectID: projectID,
		StepID:    &stepID,
	}

	output, err := h.usecase.Explain(ctx, input)
	if err != nil {
		Error(w, http.StatusInternalServerError, "EXPLAIN_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusOK, output)
}

// ========== Session Management Handlers ==========

// GetOrCreateSession handles GET /api/v1/projects/{project_id}/copilot/session
func (h *CopilotHandler) GetOrCreateSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)
	userID := middleware.GetUserID(ctx)

	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_PROJECT_ID", "Invalid project ID", nil)
		return
	}

	input := usecase.GetOrCreateSessionInput{
		TenantID:  tenantID,
		UserID:    userID.String(),
		ProjectID: projectID,
	}

	session, err := h.usecase.GetOrCreateSession(ctx, input)
	if err != nil {
		Error(w, http.StatusInternalServerError, "SESSION_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusOK, session)
}

// ListSessions handles GET /api/v1/projects/{project_id}/copilot/sessions
func (h *CopilotHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)
	userID := middleware.GetUserID(ctx)

	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_PROJECT_ID", "Invalid project ID", nil)
		return
	}

	input := usecase.ListSessionsInput{
		TenantID:  tenantID,
		UserID:    userID.String(),
		ProjectID: projectID,
	}

	sessions, err := h.usecase.ListSessions(ctx, input)
	if err != nil {
		Error(w, http.StatusInternalServerError, "SESSION_LIST_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{"sessions": sessions})
}

// StartNewSession handles POST /api/v1/projects/{project_id}/copilot/sessions/new
func (h *CopilotHandler) StartNewSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)
	userID := middleware.GetUserID(ctx)

	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_PROJECT_ID", "Invalid project ID", nil)
		return
	}

	input := usecase.StartNewSessionInput{
		TenantID:  tenantID,
		UserID:    userID.String(),
		ProjectID: projectID,
	}

	session, err := h.usecase.StartNewSession(ctx, input)
	if err != nil {
		Error(w, http.StatusInternalServerError, "SESSION_CREATE_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusCreated, session)
}

// GetSessionMessages handles GET /api/v1/projects/{project_id}/copilot/sessions/{session_id}
func (h *CopilotHandler) GetSessionMessages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)

	sessionID, err := uuid.Parse(chi.URLParam(r, "session_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_SESSION_ID", "Invalid session ID", nil)
		return
	}

	session, err := h.usecase.GetSessionWithMessages(ctx, tenantID, sessionID)
	if err != nil {
		Error(w, http.StatusInternalServerError, "SESSION_GET_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusOK, session)
}

// ChatWithSessionRequest represents the request body for chat with session
type ChatWithSessionRequest struct {
	Message   string  `json:"message"`
	SessionID *string `json:"session_id,omitempty"`
	Context   string  `json:"context,omitempty"`
}

// ChatWithSession handles POST /api/v1/projects/{project_id}/copilot/chat
func (h *CopilotHandler) ChatWithSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)
	userID := middleware.GetUserID(ctx)

	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_PROJECT_ID", "Invalid project ID", nil)
		return
	}

	var req ChatWithSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	if req.Message == "" {
		Error(w, http.StatusBadRequest, "EMPTY_MESSAGE", "Message cannot be empty", nil)
		return
	}

	var sessionID *uuid.UUID
	if req.SessionID != nil {
		id, err := uuid.Parse(*req.SessionID)
		if err != nil {
			Error(w, http.StatusBadRequest, "INVALID_SESSION_ID", "Invalid session ID", nil)
			return
		}
		sessionID = &id
	}

	input := usecase.ChatWithSessionInput{
		TenantID:  tenantID,
		UserID:    userID.String(),
		ProjectID: projectID,
		SessionID: sessionID,
		Message:   req.Message,
		Context:   req.Context,
	}

	output, session, err := h.usecase.ChatWithSession(ctx, input)
	if err != nil {
		Error(w, http.StatusInternalServerError, "CHAT_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"response":    output.Response,
		"suggestions": output.Suggestions,
		"session":     session,
	})
}

// GenerateProjectRequest represents the request body for project generation
type GenerateProjectRequest struct {
	Description string `json:"description"`
}

// GenerateProject handles POST /api/v1/projects/{project_id}/copilot/generate
func (h *CopilotHandler) GenerateProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)

	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_PROJECT_ID", "Invalid project ID", nil)
		return
	}

	var req GenerateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	if req.Description == "" {
		Error(w, http.StatusBadRequest, "EMPTY_DESCRIPTION", "Description cannot be empty", nil)
		return
	}

	input := usecase.GenerateProjectInput{
		TenantID:    tenantID,
		ProjectID:   projectID,
		Description: req.Description,
	}

	output, err := h.usecase.GenerateProject(ctx, input)
	if err != nil {
		Error(w, http.StatusInternalServerError, "GENERATE_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusOK, output)
}

// ========== Async System Project Endpoints (Meta-Project Architecture) ==========

// AsyncGenerateRequest represents the request body for async project generation
type AsyncGenerateRequest struct {
	Prompt    string `json:"prompt"`
	SessionID string `json:"session_id,omitempty"`
}

// AsyncGenerateProject handles POST /api/v1/copilot/async/generate
// Returns run_id immediately, client polls GET /copilot/runs/{id} for result
func (h *CopilotHandler) AsyncGenerateProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)
	userID := middleware.GetUserID(ctx)

	var req AsyncGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	if req.Prompt == "" {
		Error(w, http.StatusBadRequest, "EMPTY_PROMPT", "Prompt cannot be empty", nil)
		return
	}

	// Build input for system project
	inputData, err := json.Marshal(map[string]interface{}{
		"prompt":    req.Prompt,
		"tenant_id": tenantID.String(),
	})
	if err != nil {
		slog.Error("failed to marshal input data", "error", err)
		Error(w, http.StatusInternalServerError, "MARSHAL_ERROR", "Failed to marshal input data", nil)
		return
	}

	result, err := h.runUsecase.ExecuteSystemProject(ctx, usecase.ExecuteSystemProjectInput{
		TenantID:      tenantID,
		SystemSlug:    "copilot",
		EntryPoint:    "generate",
		Input:         inputData,
		TriggerSource: "copilot",
		TriggerMetadata: map[string]interface{}{
			"feature":    "generate",
			"user_id":    userID.String(),
			"session_id": req.SessionID,
		},
		UserID: &userID,
	})

	if err != nil {
		Error(w, http.StatusInternalServerError, "GENERATE_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusAccepted, map[string]interface{}{
		"run_id": result.RunID,
		"status": "pending",
	})
}

// AsyncSuggestRequest represents the request body for async suggestions
type AsyncSuggestRequest struct {
	ProjectID string `json:"project_id"`
	Context   string `json:"context,omitempty"`
}

// AsyncSuggest handles POST /api/v1/copilot/async/suggest
func (h *CopilotHandler) AsyncSuggest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)
	userID := middleware.GetUserID(ctx)

	var req AsyncSuggestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	if req.ProjectID == "" {
		Error(w, http.StatusBadRequest, "EMPTY_PROJECT_ID", "Project ID cannot be empty", nil)
		return
	}

	inputData, err := json.Marshal(map[string]interface{}{
		"project_id": req.ProjectID,
		"context":    req.Context,
		"tenant_id":  tenantID.String(),
	})
	if err != nil {
		slog.Error("failed to marshal input data", "error", err)
		Error(w, http.StatusInternalServerError, "MARSHAL_ERROR", "Failed to marshal input data", nil)
		return
	}

	result, err := h.runUsecase.ExecuteSystemProject(ctx, usecase.ExecuteSystemProjectInput{
		TenantID:      tenantID,
		SystemSlug:    "copilot",
		EntryPoint:    "suggest",
		Input:         inputData,
		TriggerSource: "copilot",
		TriggerMetadata: map[string]interface{}{
			"feature":    "suggest",
			"user_id":    userID.String(),
			"project_id": req.ProjectID,
		},
		UserID: &userID,
	})

	if err != nil {
		Error(w, http.StatusInternalServerError, "SUGGEST_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusAccepted, map[string]interface{}{
		"run_id": result.RunID,
		"status": "pending",
	})
}

// AsyncDiagnoseRequest represents the request body for async diagnosis
type AsyncDiagnoseRequest struct {
	RunID string `json:"run_id"`
}

// AsyncDiagnose handles POST /api/v1/copilot/async/diagnose
func (h *CopilotHandler) AsyncDiagnose(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)
	userID := middleware.GetUserID(ctx)

	var req AsyncDiagnoseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	if req.RunID == "" {
		Error(w, http.StatusBadRequest, "EMPTY_RUN_ID", "Run ID cannot be empty", nil)
		return
	}

	inputData, err := json.Marshal(map[string]interface{}{
		"run_id":    req.RunID,
		"tenant_id": tenantID.String(),
	})
	if err != nil {
		slog.Error("failed to marshal input data", "error", err)
		Error(w, http.StatusInternalServerError, "MARSHAL_ERROR", "Failed to marshal input data", nil)
		return
	}

	result, err := h.runUsecase.ExecuteSystemProject(ctx, usecase.ExecuteSystemProjectInput{
		TenantID:      tenantID,
		SystemSlug:    "copilot",
		EntryPoint:    "diagnose",
		Input:         inputData,
		TriggerSource: "copilot",
		TriggerMetadata: map[string]interface{}{
			"feature":       "diagnose",
			"user_id":       userID.String(),
			"target_run_id": req.RunID,
		},
		UserID: &userID,
	})

	if err != nil {
		Error(w, http.StatusInternalServerError, "DIAGNOSE_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusAccepted, map[string]interface{}{
		"run_id": result.RunID,
		"status": "pending",
	})
}

// AsyncOptimizeRequest represents the request body for async optimization
type AsyncOptimizeRequest struct {
	ProjectID string `json:"project_id"`
}

// AsyncOptimize handles POST /api/v1/copilot/async/optimize
func (h *CopilotHandler) AsyncOptimize(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)
	userID := middleware.GetUserID(ctx)

	var req AsyncOptimizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	if req.ProjectID == "" {
		Error(w, http.StatusBadRequest, "EMPTY_PROJECT_ID", "Project ID cannot be empty", nil)
		return
	}

	inputData, err := json.Marshal(map[string]interface{}{
		"project_id": req.ProjectID,
		"tenant_id":  tenantID.String(),
	})
	if err != nil {
		slog.Error("failed to marshal input data", "error", err)
		Error(w, http.StatusInternalServerError, "MARSHAL_ERROR", "Failed to marshal input data", nil)
		return
	}

	result, err := h.runUsecase.ExecuteSystemProject(ctx, usecase.ExecuteSystemProjectInput{
		TenantID:      tenantID,
		SystemSlug:    "copilot",
		EntryPoint:    "optimize",
		Input:         inputData,
		TriggerSource: "copilot",
		TriggerMetadata: map[string]interface{}{
			"feature":    "optimize",
			"user_id":    userID.String(),
			"project_id": req.ProjectID,
		},
		UserID: &userID,
	})

	if err != nil {
		Error(w, http.StatusInternalServerError, "OPTIMIZE_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusAccepted, map[string]interface{}{
		"run_id": result.RunID,
		"status": "pending",
	})
}

// GetCopilotRun handles GET /api/v1/copilot/runs/{id}
// Used for polling the result of async copilot operations
func (h *CopilotHandler) GetCopilotRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)

	runID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_RUN_ID", "Invalid run ID", nil)
		return
	}

	run, err := h.runUsecase.GetByID(ctx, tenantID, runID)
	if err != nil {
		Error(w, http.StatusNotFound, "RUN_NOT_FOUND", "Run not found", nil)
		return
	}

	response := map[string]interface{}{
		"run_id": run.ID,
		"status": run.Status,
	}

	if run.StartedAt != nil {
		response["started_at"] = run.StartedAt
	}
	if run.CompletedAt != nil {
		response["completed_at"] = run.CompletedAt
	}
	if run.Status == domain.RunStatusCompleted && run.Output != nil {
		var output interface{}
		if err := json.Unmarshal(run.Output, &output); err == nil {
			response["output"] = output
		}
	}
	if run.Status == domain.RunStatusFailed && run.Error != nil {
		response["error"] = *run.Error
	}

	JSON(w, http.StatusOK, response)
}
