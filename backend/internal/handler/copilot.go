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

// ============================================================================
// Builder Request/Response types (workflow generation)
// ============================================================================

// StartCopilotSessionRequest represents the request body for starting a copilot session
type StartCopilotSessionRequest struct {
	InitialPrompt string `json:"initial_prompt"`
	Mode          string `json:"mode,omitempty"` // create, enhance, explain
}

// StartCopilotSessionResponse represents the response for starting a copilot session
type StartCopilotSessionResponse struct {
	SessionID string              `json:"session_id"`
	Status    string              `json:"status"`
	Phase     string              `json:"phase"`
	Progress  int                 `json:"progress"`
	Message   *CopilotMessageDTO  `json:"message,omitempty"`
}

// CopilotMessageDTO represents a message in the copilot
type CopilotMessageDTO struct {
	ID                 string   `json:"id"`
	Role               string   `json:"role"`
	Content            string   `json:"content"`
	SuggestedQuestions []string `json:"suggested_questions,omitempty"`
}

// SendCopilotMessageRequest represents the request body for sending a message
type SendCopilotMessageRequest struct {
	Content string `json:"content"`
}

// SendCopilotMessageResponse represents the response for sending a message (async)
type SendCopilotMessageResponse struct {
	RunID  string `json:"run_id"`
	Status string `json:"status"`
}

// GetCopilotSessionResponse represents the response for getting a session
type GetCopilotSessionResponse struct {
	ID               string              `json:"id"`
	Status           string              `json:"status"`
	Phase            string              `json:"hearing_phase"`
	Progress         int                 `json:"hearing_progress"`
	Mode             string              `json:"mode"`
	ContextProjectID *string             `json:"context_project_id,omitempty"`
	ProjectID        *string             `json:"project_id,omitempty"`
	Messages         []CopilotMessageDTO `json:"messages,omitempty"`
	CreatedAt        string              `json:"created_at"`
	UpdatedAt        string              `json:"updated_at"`
}

// ============================================================================
// Builder Handlers (workflow generation via hearing process)
// ============================================================================

// StartCopilotSession handles POST /api/v1/projects/{project_id}/copilot/sessions
func (h *CopilotHandler) StartCopilotSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)
	userID := middleware.GetUserID(ctx)

	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_PROJECT_ID", "Invalid project ID", nil)
		return
	}

	var req StartCopilotSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	if req.InitialPrompt == "" {
		Error(w, http.StatusBadRequest, "INVALID_REQUEST", "initial_prompt is required", nil)
		return
	}

	// Parse mode
	mode := domain.CopilotSessionModeCreate
	if req.Mode != "" {
		switch req.Mode {
		case "create":
			mode = domain.CopilotSessionModeCreate
		case "enhance":
			mode = domain.CopilotSessionModeEnhance
		case "explain":
			mode = domain.CopilotSessionModeExplain
		default:
			Error(w, http.StatusBadRequest, "INVALID_MODE", "Invalid mode. Use: create, enhance, explain", nil)
			return
		}
	}

	input := usecase.StartSessionInput{
		TenantID:         tenantID,
		UserID:           userID.String(),
		ContextProjectID: &projectID,
		Mode:             mode,
		InitialPrompt:    req.InitialPrompt,
	}

	output, err := h.usecase.StartSession(ctx, input)
	if err != nil {
		Error(w, http.StatusInternalServerError, "START_SESSION_FAILED", err.Error(), nil)
		return
	}

	resp := StartCopilotSessionResponse{
		SessionID: output.Session.ID.String(),
		Status:    string(output.Session.Status),
		Phase:     string(output.Session.HearingPhase),
		Progress:  output.Session.HearingProgress,
	}

	if output.Message != nil {
		resp.Message = &CopilotMessageDTO{
			ID:      output.Message.ID.String(),
			Role:    output.Message.Role,
			Content: output.Message.Content,
		}
		if output.SuggestedQuestions != nil {
			resp.Message.SuggestedQuestions = output.SuggestedQuestions
		}
	}

	JSON(w, http.StatusCreated, resp)
}

// GetCopilotSession handles GET /api/v1/projects/{project_id}/copilot/sessions/{session_id}
func (h *CopilotHandler) GetCopilotSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)

	sessionID, err := uuid.Parse(chi.URLParam(r, "session_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_SESSION_ID", "Invalid session ID", nil)
		return
	}

	session, err := h.usecase.GetSession(ctx, tenantID, sessionID)
	if err != nil {
		Error(w, http.StatusNotFound, "SESSION_NOT_FOUND", err.Error(), nil)
		return
	}

	resp := GetCopilotSessionResponse{
		ID:        session.ID.String(),
		Status:    string(session.Status),
		Phase:     string(session.HearingPhase),
		Progress:  session.HearingProgress,
		Mode:      string(session.Mode),
		CreatedAt: session.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: session.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if session.ContextProjectID != nil {
		contextProjectIDStr := session.ContextProjectID.String()
		resp.ContextProjectID = &contextProjectIDStr
	}

	if session.ProjectID != nil {
		projectIDStr := session.ProjectID.String()
		resp.ProjectID = &projectIDStr
	}

	for _, msg := range session.Messages {
		dto := CopilotMessageDTO{
			ID:      msg.ID.String(),
			Role:    msg.Role,
			Content: msg.Content,
		}
		if msg.SuggestedQuestions != nil {
			var questions []string
			json.Unmarshal(msg.SuggestedQuestions, &questions)
			dto.SuggestedQuestions = questions
		}
		resp.Messages = append(resp.Messages, dto)
	}

	JSON(w, http.StatusOK, resp)
}

// SendCopilotMessage handles POST /api/v1/projects/{project_id}/copilot/sessions/{session_id}/messages
func (h *CopilotHandler) SendCopilotMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)
	userID := middleware.GetUserID(ctx)

	sessionID, err := uuid.Parse(chi.URLParam(r, "session_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_SESSION_ID", "Invalid session ID", nil)
		return
	}

	var req SendCopilotMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	if req.Content == "" {
		Error(w, http.StatusBadRequest, "INVALID_REQUEST", "content is required", nil)
		return
	}

	// Execute async via system project
	inputData := map[string]interface{}{
		"session_id": sessionID.String(),
		"message":    req.Content,
		"tenant_id":  tenantID.String(),
	}
	if userID != uuid.Nil {
		inputData["user_id"] = userID.String()
	}

	inputJSON, err := json.Marshal(inputData)
	if err != nil {
		Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to marshal input", nil)
		return
	}

	execInput := usecase.ExecuteSystemProjectInput{
		TenantID:      tenantID,
		SystemSlug:    "copilot",
		EntryPoint:    "chat",
		Input:         inputJSON,
		TriggerSource: "copilot",
		TriggerMetadata: map[string]interface{}{
			"feature":    "chat",
			"session_id": sessionID.String(),
		},
	}
	result, err := h.runUsecase.ExecuteSystemProject(ctx, execInput)
	if err != nil {
		Error(w, http.StatusInternalServerError, "EXECUTE_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusAccepted, SendCopilotMessageResponse{
		RunID:  result.RunID.String(),
		Status: "pending",
	})
}

// ConstructCopilotWorkflow handles POST /api/v1/projects/{project_id}/copilot/sessions/{session_id}/construct
func (h *CopilotHandler) ConstructCopilotWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)
	userID := middleware.GetUserID(ctx)

	sessionID, err := uuid.Parse(chi.URLParam(r, "session_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_SESSION_ID", "Invalid session ID", nil)
		return
	}

	// Verify session exists and is in correct state
	session, err := h.usecase.GetSession(ctx, tenantID, sessionID)
	if err != nil {
		Error(w, http.StatusNotFound, "SESSION_NOT_FOUND", err.Error(), nil)
		return
	}

	if session.HearingPhase != domain.CopilotPhaseCompleted {
		Error(w, http.StatusBadRequest, "HEARING_NOT_COMPLETED", "Hearing phase must be completed before construction", nil)
		return
	}

	// Execute async via system project
	inputData := map[string]interface{}{
		"session_id": sessionID.String(),
		"tenant_id":  tenantID.String(),
	}
	if userID != uuid.Nil {
		inputData["user_id"] = userID.String()
	}

	inputJSON, err := json.Marshal(inputData)
	if err != nil {
		Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to marshal input", nil)
		return
	}

	execInput := usecase.ExecuteSystemProjectInput{
		TenantID:      tenantID,
		SystemSlug:    "copilot",
		EntryPoint:    "chat",
		Input:         inputJSON,
		TriggerSource: "copilot",
		TriggerMetadata: map[string]interface{}{
			"feature":    "construct",
			"session_id": sessionID.String(),
		},
	}
	result, err := h.runUsecase.ExecuteSystemProject(ctx, execInput)
	if err != nil {
		Error(w, http.StatusInternalServerError, "EXECUTE_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusAccepted, SendCopilotMessageResponse{
		RunID:  result.RunID.String(),
		Status: "pending",
	})
}

// RefineCopilotRequest represents the request body for refining a workflow
type RefineCopilotRequest struct {
	Feedback string `json:"feedback"`
}

// RefineCopilotWorkflow handles POST /api/v1/projects/{project_id}/copilot/sessions/{session_id}/refine
func (h *CopilotHandler) RefineCopilotWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)
	userID := middleware.GetUserID(ctx)

	sessionID, err := uuid.Parse(chi.URLParam(r, "session_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_SESSION_ID", "Invalid session ID", nil)
		return
	}

	var req RefineCopilotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	if req.Feedback == "" {
		Error(w, http.StatusBadRequest, "INVALID_REQUEST", "feedback is required", nil)
		return
	}

	// Verify session exists and has a project
	session, err := h.usecase.GetSession(ctx, tenantID, sessionID)
	if err != nil {
		Error(w, http.StatusNotFound, "SESSION_NOT_FOUND", err.Error(), nil)
		return
	}

	if session.ProjectID == nil {
		Error(w, http.StatusBadRequest, "PROJECT_NOT_CREATED", "Workflow must be constructed before refinement", nil)
		return
	}

	// Execute async via system project
	inputData := map[string]interface{}{
		"session_id": sessionID.String(),
		"project_id": session.ProjectID.String(),
		"feedback":   req.Feedback,
		"tenant_id":  tenantID.String(),
	}
	if userID != uuid.Nil {
		inputData["user_id"] = userID.String()
	}

	inputJSON, err := json.Marshal(inputData)
	if err != nil {
		Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to marshal input", nil)
		return
	}

	execInput := usecase.ExecuteSystemProjectInput{
		TenantID:      tenantID,
		SystemSlug:    "copilot",
		EntryPoint:    "chat",
		Input:         inputJSON,
		TriggerSource: "copilot",
		TriggerMetadata: map[string]interface{}{
			"feature":    "refine",
			"session_id": sessionID.String(),
		},
	}
	result, err := h.runUsecase.ExecuteSystemProject(ctx, execInput)
	if err != nil {
		Error(w, http.StatusInternalServerError, "EXECUTE_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusAccepted, SendCopilotMessageResponse{
		RunID:  result.RunID.String(),
		Status: "pending",
	})
}

// FinalizeCopilotSession handles POST /api/v1/projects/{project_id}/copilot/sessions/{session_id}/finalize
func (h *CopilotHandler) FinalizeCopilotSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)

	sessionID, err := uuid.Parse(chi.URLParam(r, "session_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_SESSION_ID", "Invalid session ID", nil)
		return
	}

	err = h.usecase.FinalizeSession(ctx, tenantID, sessionID)
	if err != nil {
		Error(w, http.StatusInternalServerError, "FINALIZE_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"status": "completed",
	})
}

// DeleteCopilotSession handles DELETE /api/v1/projects/{project_id}/copilot/sessions/{session_id}
func (h *CopilotHandler) DeleteCopilotSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)

	sessionID, err := uuid.Parse(chi.URLParam(r, "session_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_SESSION_ID", "Invalid session ID", nil)
		return
	}

	err = h.usecase.DeleteSession(ctx, tenantID, sessionID)
	if err != nil {
		Error(w, http.StatusInternalServerError, "DELETE_FAILED", err.Error(), nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListCopilotSessionsByProject handles GET /api/v1/projects/{project_id}/copilot/sessions
func (h *CopilotHandler) ListCopilotSessionsByProject(w http.ResponseWriter, r *http.Request) {
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
		Error(w, http.StatusInternalServerError, "LIST_FAILED", err.Error(), nil)
		return
	}

	type SessionListItem struct {
		ID               string  `json:"id"`
		Status           string  `json:"status"`
		Phase            string  `json:"hearing_phase"`
		Progress         int     `json:"hearing_progress"`
		Mode             string  `json:"mode"`
		ContextProjectID *string `json:"context_project_id,omitempty"`
		ProjectID        *string `json:"project_id,omitempty"`
		CreatedAt        string  `json:"created_at"`
		UpdatedAt        string  `json:"updated_at"`
	}

	items := make([]SessionListItem, 0, len(sessions))
	for _, s := range sessions {
		item := SessionListItem{
			ID:        s.ID.String(),
			Status:    string(s.Status),
			Phase:     string(s.HearingPhase),
			Progress:  s.HearingProgress,
			Mode:      string(s.Mode),
			CreatedAt: s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: s.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
		if s.ContextProjectID != nil {
			contextProjectIDStr := s.ContextProjectID.String()
			item.ContextProjectID = &contextProjectIDStr
		}
		if s.ProjectID != nil {
			projectIDStr := s.ProjectID.String()
			item.ProjectID = &projectIDStr
		}
		items = append(items, item)
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"sessions": items,
		"total":    len(items),
	})
}

// ============================================================================
// Original Copilot Handlers (suggestions, diagnosis, etc.)
// ============================================================================

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

// ========== Legacy Session Management Handlers (kept for backwards compatibility) ==========

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

// ListSessions handles GET /api/v1/projects/{project_id}/copilot/sessions (legacy)
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

// GetSessionMessages handles GET /api/v1/projects/{project_id}/copilot/sessions/{session_id} (legacy)
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
