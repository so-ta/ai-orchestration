package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/middleware"
	"github.com/souta/ai-orchestration/internal/usecase"
)

// CopilotHandler handles copilot API requests
type CopilotHandler struct {
	usecase *usecase.CopilotUsecase
}

// NewCopilotHandler creates a new CopilotHandler
func NewCopilotHandler(uc *usecase.CopilotUsecase) *CopilotHandler {
	return &CopilotHandler{usecase: uc}
}

// SuggestRequest represents the request body for suggestions
type SuggestRequest struct {
	WorkflowID string  `json:"workflow_id"`
	StepID     *string `json:"step_id,omitempty"`
	Context    string  `json:"context,omitempty"`
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

	workflowID, err := uuid.Parse(req.WorkflowID)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_WORKFLOW_ID", "Invalid workflow ID", nil)
		return
	}

	var stepID *uuid.UUID
	if req.StepID != nil {
		id, err := uuid.Parse(*req.StepID)
		if err == nil {
			stepID = &id
		}
	}

	input := usecase.SuggestInput{
		TenantID:   tenantID,
		WorkflowID: workflowID,
		StepID:     stepID,
		Context:    req.Context,
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
		if err == nil {
			stepRunID = &id
		}
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
	WorkflowID string  `json:"workflow_id"`
	StepID     *string `json:"step_id,omitempty"`
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

	workflowID, err := uuid.Parse(req.WorkflowID)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_WORKFLOW_ID", "Invalid workflow ID", nil)
		return
	}

	var stepID *uuid.UUID
	if req.StepID != nil {
		id, err := uuid.Parse(*req.StepID)
		if err == nil {
			stepID = &id
		}
	}

	input := usecase.ExplainInput{
		TenantID:   tenantID,
		WorkflowID: workflowID,
		StepID:     stepID,
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
	WorkflowID string `json:"workflow_id"`
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

	workflowID, err := uuid.Parse(req.WorkflowID)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_WORKFLOW_ID", "Invalid workflow ID", nil)
		return
	}

	input := usecase.OptimizeInput{
		TenantID:   tenantID,
		WorkflowID: workflowID,
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
	WorkflowID *string `json:"workflow_id,omitempty"`
	Message    string  `json:"message"`
	Context    string  `json:"context,omitempty"`
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

	var workflowID *uuid.UUID
	if req.WorkflowID != nil {
		id, err := uuid.Parse(*req.WorkflowID)
		if err == nil {
			workflowID = &id
		}
	}

	input := usecase.ChatInput{
		TenantID:   tenantID,
		WorkflowID: workflowID,
		Message:    req.Message,
		Context:    req.Context,
	}

	output, err := h.usecase.Chat(ctx, input)
	if err != nil {
		Error(w, http.StatusInternalServerError, "CHAT_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusOK, output)
}

// SuggestForStep handles POST /api/v1/workflows/{workflow_id}/steps/{step_id}/copilot/suggest
func (h *CopilotHandler) SuggestForStep(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)

	workflowID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_WORKFLOW_ID", "Invalid workflow ID", nil)
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
	json.NewDecoder(r.Body).Decode(&req)

	input := usecase.SuggestInput{
		TenantID:   tenantID,
		WorkflowID: workflowID,
		StepID:     &stepID,
		Context:    req.Context,
	}

	output, err := h.usecase.Suggest(ctx, input)
	if err != nil {
		Error(w, http.StatusInternalServerError, "SUGGEST_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusOK, output)
}

// ExplainStep handles POST /api/v1/workflows/{workflow_id}/steps/{step_id}/copilot/explain
func (h *CopilotHandler) ExplainStep(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)

	workflowID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_WORKFLOW_ID", "Invalid workflow ID", nil)
		return
	}

	stepID, err := uuid.Parse(chi.URLParam(r, "step_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_STEP_ID", "Invalid step ID", nil)
		return
	}

	input := usecase.ExplainInput{
		TenantID:   tenantID,
		WorkflowID: workflowID,
		StepID:     &stepID,
	}

	output, err := h.usecase.Explain(ctx, input)
	if err != nil {
		Error(w, http.StatusInternalServerError, "EXPLAIN_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusOK, output)
}

// ========== Session Management Handlers ==========

// GetOrCreateSession handles GET /api/v1/workflows/{workflow_id}/copilot/session
func (h *CopilotHandler) GetOrCreateSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)
	userID := middleware.GetUserID(ctx)

	workflowID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_WORKFLOW_ID", "Invalid workflow ID", nil)
		return
	}

	input := usecase.GetOrCreateSessionInput{
		TenantID:   tenantID,
		UserID:     userID.String(),
		WorkflowID: workflowID,
	}

	session, err := h.usecase.GetOrCreateSession(ctx, input)
	if err != nil {
		Error(w, http.StatusInternalServerError, "SESSION_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusOK, session)
}

// ListSessions handles GET /api/v1/workflows/{workflow_id}/copilot/sessions
func (h *CopilotHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)
	userID := middleware.GetUserID(ctx)

	workflowID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_WORKFLOW_ID", "Invalid workflow ID", nil)
		return
	}

	input := usecase.ListSessionsInput{
		TenantID:   tenantID,
		UserID:     userID.String(),
		WorkflowID: workflowID,
	}

	sessions, err := h.usecase.ListSessions(ctx, input)
	if err != nil {
		Error(w, http.StatusInternalServerError, "SESSION_LIST_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{"sessions": sessions})
}

// StartNewSession handles POST /api/v1/workflows/{workflow_id}/copilot/sessions/new
func (h *CopilotHandler) StartNewSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)
	userID := middleware.GetUserID(ctx)

	workflowID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_WORKFLOW_ID", "Invalid workflow ID", nil)
		return
	}

	input := usecase.StartNewSessionInput{
		TenantID:   tenantID,
		UserID:     userID.String(),
		WorkflowID: workflowID,
	}

	session, err := h.usecase.StartNewSession(ctx, input)
	if err != nil {
		Error(w, http.StatusInternalServerError, "SESSION_CREATE_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusCreated, session)
}

// GetSessionMessages handles GET /api/v1/workflows/{workflow_id}/copilot/sessions/{session_id}
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

// ChatWithSession handles POST /api/v1/workflows/{workflow_id}/copilot/chat
func (h *CopilotHandler) ChatWithSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)
	userID := middleware.GetUserID(ctx)

	workflowID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_WORKFLOW_ID", "Invalid workflow ID", nil)
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
		if err == nil {
			sessionID = &id
		}
	}

	input := usecase.ChatWithSessionInput{
		TenantID:   tenantID,
		UserID:     userID.String(),
		WorkflowID: workflowID,
		SessionID:  sessionID,
		Message:    req.Message,
		Context:    req.Context,
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

// GenerateWorkflowRequest represents the request body for workflow generation
type GenerateWorkflowRequest struct {
	Description string `json:"description"`
}

// GenerateWorkflow handles POST /api/v1/workflows/{workflow_id}/copilot/generate
func (h *CopilotHandler) GenerateWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)

	workflowID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_WORKFLOW_ID", "Invalid workflow ID", nil)
		return
	}

	var req GenerateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	if req.Description == "" {
		Error(w, http.StatusBadRequest, "EMPTY_DESCRIPTION", "Description cannot be empty", nil)
		return
	}

	input := usecase.GenerateWorkflowInput{
		TenantID:    tenantID,
		WorkflowID:  workflowID,
		Description: req.Description,
	}

	output, err := h.usecase.GenerateWorkflow(ctx, input)
	if err != nil {
		Error(w, http.StatusInternalServerError, "GENERATE_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusOK, output)
}
