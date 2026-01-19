package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/middleware"
	"github.com/souta/ai-orchestration/internal/usecase"
)

// BuilderHandler handles AI Workflow Builder API requests
type BuilderHandler struct {
	usecase    *usecase.BuilderUsecase
	runUsecase *usecase.RunUsecase
}

// NewBuilderHandler creates a new BuilderHandler
func NewBuilderHandler(uc *usecase.BuilderUsecase, runUC *usecase.RunUsecase) *BuilderHandler {
	return &BuilderHandler{usecase: uc, runUsecase: runUC}
}

// ============================================================================
// Request/Response types
// ============================================================================

// StartSessionRequest represents the request body for starting a builder session
type StartSessionRequest struct {
	InitialPrompt string `json:"initial_prompt"`
}

// StartSessionResponse represents the response for starting a builder session
type StartSessionResponse struct {
	SessionID string               `json:"session_id"`
	Status    string               `json:"status"`
	Phase     string               `json:"phase"`
	Progress  int                  `json:"progress"`
	Message   *BuilderMessageDTO   `json:"message,omitempty"`
}

// BuilderMessageDTO represents a message in the builder
type BuilderMessageDTO struct {
	ID                 string   `json:"id"`
	Role               string   `json:"role"`
	Content            string   `json:"content"`
	SuggestedQuestions []string `json:"suggested_questions,omitempty"`
}

// SendMessageRequest represents the request body for sending a message
type SendMessageRequest struct {
	Content string `json:"content"`
}

// SendMessageResponse represents the response for sending a message (async)
type SendMessageResponse struct {
	RunID  string `json:"run_id"`
	Status string `json:"status"`
}

// GetSessionResponse represents the response for getting a session
type GetSessionResponse struct {
	ID              string              `json:"id"`
	Status          string              `json:"status"`
	Phase           string              `json:"hearing_phase"`
	Progress        int                 `json:"hearing_progress"`
	ProjectID       *string             `json:"project_id,omitempty"`
	Messages        []BuilderMessageDTO `json:"messages,omitempty"`
	CreatedAt       string              `json:"created_at"`
	UpdatedAt       string              `json:"updated_at"`
}

// ConstructRequest represents the request body for constructing a workflow
type ConstructRequest struct {
	// Reserved for future options
}

// RefineRequest represents the request body for refining a workflow
type RefineRequest struct {
	Feedback string `json:"feedback"`
}

// FinalizeRequest represents the request body for finalizing a workflow
type FinalizeRequest struct {
	// No additional fields
}

// ============================================================================
// Handlers
// ============================================================================

// StartSession handles POST /api/v1/builder/sessions
func (h *BuilderHandler) StartSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)
	userID := middleware.GetUserID(ctx)

	var req StartSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	if req.InitialPrompt == "" {
		Error(w, http.StatusBadRequest, "INVALID_REQUEST", "initial_prompt is required", nil)
		return
	}

	input := usecase.StartBuilderSessionInput{
		TenantID:      tenantID,
		UserID:        userID.String(),
		InitialPrompt: req.InitialPrompt,
	}

	output, err := h.usecase.StartSession(ctx, input)
	if err != nil {
		Error(w, http.StatusInternalServerError, "START_SESSION_FAILED", err.Error(), nil)
		return
	}

	resp := StartSessionResponse{
		SessionID: output.Session.ID.String(),
		Status:    string(output.Session.Status),
		Phase:     string(output.Session.HearingPhase),
		Progress:  output.Session.HearingProgress,
	}

	if output.Message != nil {
		resp.Message = &BuilderMessageDTO{
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

// GetSession handles GET /api/v1/builder/sessions/{id}
func (h *BuilderHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)

	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_SESSION_ID", "Invalid session ID", nil)
		return
	}

	session, err := h.usecase.GetSession(ctx, tenantID, sessionID)
	if err != nil {
		Error(w, http.StatusNotFound, "SESSION_NOT_FOUND", err.Error(), nil)
		return
	}

	resp := GetSessionResponse{
		ID:        session.ID.String(),
		Status:    string(session.Status),
		Phase:     string(session.HearingPhase),
		Progress:  session.HearingProgress,
		CreatedAt: session.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: session.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if session.ProjectID != nil {
		projectIDStr := session.ProjectID.String()
		resp.ProjectID = &projectIDStr
	}

	for _, msg := range session.Messages {
		dto := BuilderMessageDTO{
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

// SendMessage handles POST /api/v1/builder/sessions/{id}/messages
func (h *BuilderHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)
	userID := middleware.GetUserID(ctx)

	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_SESSION_ID", "Invalid session ID", nil)
		return
	}

	var req SendMessageRequest
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
	// Only include user_id if it's a valid (non-zero) UUID
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
		SystemSlug:    "ai-builder",
		EntryPoint:    "proposal",
		Input:         inputJSON,
		TriggerSource: "builder",
		TriggerMetadata: map[string]interface{}{
			"feature":    "proposal",
			"session_id": sessionID.String(),
		},
	}
	// Don't set UserID for system project execution - it's optional and causes FK issues
	// when using development user IDs that don't exist in the database
	result, err := h.runUsecase.ExecuteSystemProject(ctx, execInput)
	if err != nil {
		Error(w, http.StatusInternalServerError, "EXECUTE_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusAccepted, SendMessageResponse{
		RunID:  result.RunID.String(),
		Status: "pending",
	})
}

// Construct handles POST /api/v1/builder/sessions/{id}/construct
func (h *BuilderHandler) Construct(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)
	userID := middleware.GetUserID(ctx)

	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
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

	if session.HearingPhase != domain.HearingPhaseCompleted {
		Error(w, http.StatusBadRequest, "HEARING_NOT_COMPLETED", "Hearing phase must be completed before construction", nil)
		return
	}

	// Execute async via system project
	inputData := map[string]interface{}{
		"session_id": sessionID.String(),
		"tenant_id":  tenantID.String(),
	}
	// Only include user_id if it's a valid (non-zero) UUID
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
		SystemSlug:    "ai-builder",
		EntryPoint:    "agent_construct",
		Input:         inputJSON,
		TriggerSource: "builder",
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

	JSON(w, http.StatusAccepted, SendMessageResponse{
		RunID:  result.RunID.String(),
		Status: "pending",
	})
}

// Refine handles POST /api/v1/builder/sessions/{id}/refine
func (h *BuilderHandler) Refine(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)
	userID := middleware.GetUserID(ctx)

	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_SESSION_ID", "Invalid session ID", nil)
		return
	}

	var req RefineRequest
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
	// Only include user_id if it's a valid (non-zero) UUID
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
		SystemSlug:    "ai-builder",
		EntryPoint:    "refine",
		Input:         inputJSON,
		TriggerSource: "builder",
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

	JSON(w, http.StatusAccepted, SendMessageResponse{
		RunID:  result.RunID.String(),
		Status: "pending",
	})
}

// Finalize handles POST /api/v1/builder/sessions/{id}/finalize
func (h *BuilderHandler) Finalize(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)

	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
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

// DeleteSession handles DELETE /api/v1/builder/sessions/{id}
func (h *BuilderHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)

	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
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

// ListSessions handles GET /api/v1/builder/sessions
func (h *BuilderHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)
	userID := middleware.GetUserID(ctx)

	sessions, total, err := h.usecase.ListSessions(ctx, tenantID, userID.String())
	if err != nil {
		Error(w, http.StatusInternalServerError, "LIST_FAILED", err.Error(), nil)
		return
	}

	type SessionListItem struct {
		ID        string  `json:"id"`
		Status    string  `json:"status"`
		Phase     string  `json:"hearing_phase"`
		Progress  int     `json:"hearing_progress"`
		ProjectID *string `json:"project_id,omitempty"`
		CreatedAt string  `json:"created_at"`
		UpdatedAt string  `json:"updated_at"`
	}

	items := make([]SessionListItem, 0, len(sessions))
	for _, s := range sessions {
		item := SessionListItem{
			ID:        s.ID.String(),
			Status:    string(s.Status),
			Phase:     string(s.HearingPhase),
			Progress:  s.HearingProgress,
			CreatedAt: s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: s.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
		if s.ProjectID != nil {
			projectIDStr := s.ProjectID.String()
			item.ProjectID = &projectIDStr
		}
		items = append(items, item)
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"sessions": items,
		"total":    total,
	})
}
