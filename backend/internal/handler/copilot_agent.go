package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/engine"
	"github.com/souta/ai-orchestration/internal/middleware"
	"github.com/souta/ai-orchestration/internal/repository"
)

// CopilotAgentHandler handles agent-based copilot API requests
// This handler uses the standard workflow execution engine with the Copilot system workflow,
// which creates Run records that appear in execution history.
type CopilotAgentHandler struct {
	sessionRepo   repository.CopilotSessionRepository
	projectRepo   repository.ProjectRepository
	inlineRunner  *engine.InlineRunnerFactory
	logger        *slog.Logger

	// Cached Copilot project info
	copilotProjectID uuid.UUID
	copilotStartStep *uuid.UUID

	// Active streams for cancellation
	activeStreams sync.Map
}

// NewCopilotAgentHandler creates a new CopilotAgentHandler
func NewCopilotAgentHandler(
	sessionRepo repository.CopilotSessionRepository,
	projectRepo repository.ProjectRepository,
	inlineRunner *engine.InlineRunnerFactory,
	logger *slog.Logger,
) *CopilotAgentHandler {
	return &CopilotAgentHandler{
		sessionRepo:  sessionRepo,
		projectRepo:  projectRepo,
		inlineRunner: inlineRunner,
		logger:       logger,
	}
}

// ============================================================================
// Request/Response types
// ============================================================================

// StartAgentSessionRequest represents the request to start an agent session
type StartAgentSessionRequest struct {
	InitialPrompt string `json:"initial_prompt"`
	Mode          string `json:"mode,omitempty"` // create, enhance, explain
}

// StartAgentSessionResponse represents the response for starting an agent session
type StartAgentSessionResponse struct {
	SessionID string   `json:"session_id"`
	Response  string   `json:"response"`
	ToolsUsed []string `json:"tools_used"`
	Status    string   `json:"status"`
	Phase     string   `json:"phase"`
	Progress  int      `json:"progress"`
}

// SendAgentMessageRequest represents the request to send a message to the agent
type SendAgentMessageRequest struct {
	Content string `json:"content"`
}

// AgentStreamEvent represents an SSE event from the agent
type AgentStreamEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// ============================================================================
// Internal helpers
// ============================================================================

// getCopilotProject retrieves the Copilot system workflow project with steps
func (h *CopilotAgentHandler) getCopilotProject(ctx context.Context) (uuid.UUID, *uuid.UUID, error) {
	if h.copilotProjectID != uuid.Nil {
		return h.copilotProjectID, h.copilotStartStep, nil
	}

	// First get the basic project info to get the ID and tenant
	basicProject, err := h.projectRepo.GetSystemBySlug(ctx, "copilot")
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("copilot workflow not found: %w", err)
	}

	// Then get the full project with steps
	project, err := h.projectRepo.GetWithStepsAndEdges(ctx, basicProject.TenantID, basicProject.ID)
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("failed to load copilot workflow steps: %w", err)
	}

	h.copilotProjectID = project.ID

	// Find start step
	for _, step := range project.Steps {
		if step.Type == domain.StepTypeStart {
			h.copilotStartStep = &step.ID
			break
		}
	}

	return h.copilotProjectID, h.copilotStartStep, nil
}

// ============================================================================
// Handlers
// ============================================================================

// GetAgentSession handles GET /api/v1/projects/{project_id}/copilot/agent/sessions/{session_id}
func (h *CopilotAgentHandler) GetAgentSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)

	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_PROJECT_ID", "Invalid project ID", nil)
		return
	}

	sessionID, err := uuid.Parse(chi.URLParam(r, "session_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_SESSION_ID", "Invalid session ID", nil)
		return
	}

	session, err := h.sessionRepo.GetWithMessages(ctx, tenantID, sessionID)
	if err != nil {
		if errors.Is(err, domain.ErrCopilotSessionNotFound) {
			Error(w, http.StatusNotFound, "SESSION_NOT_FOUND", "Session not found", nil)
			return
		}
		Error(w, http.StatusInternalServerError, "GET_SESSION_FAILED", err.Error(), nil)
		return
	}

	// Verify session belongs to this project
	if session.ContextProjectID == nil || *session.ContextProjectID != projectID {
		Error(w, http.StatusNotFound, "SESSION_NOT_FOUND", "Session not found in this project", nil)
		return
	}

	// Convert messages to DTO
	messages := make([]map[string]interface{}, 0, len(session.Messages))
	for _, msg := range session.Messages {
		msgDTO := map[string]interface{}{
			"id":         msg.ID.String(),
			"role":       msg.Role,
			"content":    msg.Content,
			"created_at": msg.CreatedAt.Format(time.RFC3339),
		}
		if len(msg.ExtractedData) > 0 {
			msgDTO["extracted_data"] = msg.ExtractedData
		}
		messages = append(messages, msgDTO)
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"id":         session.ID.String(),
		"status":     string(session.Status),
		"phase":      string(session.HearingPhase),
		"progress":   session.HearingProgress,
		"mode":       string(session.Mode),
		"messages":   messages,
		"created_at": session.CreatedAt.Format(time.RFC3339),
		"updated_at": session.UpdatedAt.Format(time.RFC3339),
	})
}

// GetActiveAgentSession handles GET /api/v1/projects/{project_id}/copilot/agent/sessions/active
func (h *CopilotAgentHandler) GetActiveAgentSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)
	userID := middleware.GetUserID(ctx)

	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_PROJECT_ID", "Invalid project ID", nil)
		return
	}

	session, err := h.sessionRepo.GetActiveByUserAndProjectWithMessages(ctx, tenantID, userID.String(), projectID)
	if err != nil {
		// NotFound means no active session - return null
		if errors.Is(err, domain.ErrCopilotSessionNotFound) {
			JSON(w, http.StatusOK, map[string]interface{}{
				"session": nil,
			})
			return
		}
		// Other errors are internal server errors
		Error(w, http.StatusInternalServerError, "GET_ACTIVE_SESSION_FAILED", err.Error(), nil)
		return
	}

	// Convert messages to DTO
	messages := make([]map[string]interface{}, 0, len(session.Messages))
	for _, msg := range session.Messages {
		msgDTO := map[string]interface{}{
			"id":         msg.ID.String(),
			"role":       msg.Role,
			"content":    msg.Content,
			"created_at": msg.CreatedAt.Format(time.RFC3339),
		}
		if len(msg.ExtractedData) > 0 {
			msgDTO["extracted_data"] = msg.ExtractedData
		}
		messages = append(messages, msgDTO)
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"session": map[string]interface{}{
			"id":         session.ID.String(),
			"status":     string(session.Status),
			"phase":      string(session.HearingPhase),
			"progress":   session.HearingProgress,
			"mode":       string(session.Mode),
			"messages":   messages,
			"created_at": session.CreatedAt.Format(time.RFC3339),
			"updated_at": session.UpdatedAt.Format(time.RFC3339),
		},
	})
}

// StartAgentSession handles POST /api/v1/projects/{project_id}/copilot/agent/sessions
// This creates a session and returns immediately. Use the stream endpoint to process the initial message.
func (h *CopilotAgentHandler) StartAgentSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)
	userID := middleware.GetUserID(ctx)

	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_PROJECT_ID", "Invalid project ID", nil)
		return
	}

	var req StartAgentSessionRequest
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

	// Verify project exists
	project, err := h.projectRepo.GetByID(ctx, tenantID, projectID)
	if err != nil || project == nil {
		Error(w, http.StatusNotFound, "PROJECT_NOT_FOUND", "Project not found", nil)
		return
	}

	// Create session
	session := domain.NewCopilotSession(tenantID, userID.String(), &projectID, mode)
	session.Title = truncateString(req.InitialPrompt, 100)

	if err := h.sessionRepo.Create(ctx, session); err != nil {
		Error(w, http.StatusInternalServerError, "CREATE_SESSION_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusCreated, StartAgentSessionResponse{
		SessionID: session.ID.String(),
		Response:  "", // Empty - will be populated via SSE stream
		ToolsUsed: []string{},
		Status:    string(session.Status),
		Phase:     string(session.HearingPhase),
		Progress:  session.HearingProgress,
	})
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}

// SendAgentMessage handles POST /api/v1/projects/{project_id}/copilot/agent/sessions/{session_id}/messages
// Note: This is a synchronous endpoint. For streaming, use the /stream endpoint.
func (h *CopilotAgentHandler) SendAgentMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)
	userID := middleware.GetUserID(ctx)
	var userIDPtr *uuid.UUID
	if userID != uuid.Nil {
		userIDPtr = &userID
	}

	sessionID, err := uuid.Parse(chi.URLParam(r, "session_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_SESSION_ID", "Invalid session ID", nil)
		return
	}

	var req SendAgentMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	if req.Content == "" {
		Error(w, http.StatusBadRequest, "INVALID_REQUEST", "content is required", nil)
		return
	}

	// Get session
	session, err := h.sessionRepo.GetWithMessages(ctx, tenantID, sessionID)
	if err != nil {
		if errors.Is(err, domain.ErrCopilotSessionNotFound) {
			Error(w, http.StatusNotFound, "SESSION_NOT_FOUND", "Session not found", nil)
			return
		}
		Error(w, http.StatusInternalServerError, "GET_SESSION_FAILED", err.Error(), nil)
		return
	}

	// Get Copilot project
	copilotProjectID, startStepID, err := h.getCopilotProject(ctx)
	if err != nil {
		h.logger.Error("Failed to get copilot project", "error", err)
		Error(w, http.StatusInternalServerError, "COPILOT_WORKFLOW_NOT_FOUND", "Copilot workflow not found", nil)
		return
	}

	// Build workflow input with conversation history
	input := map[string]interface{}{
		"message":    req.Content,
		"session_id": sessionID.String(),
		"mode":       string(session.Mode),
	}
	if session.ContextProjectID != nil {
		input["project_id"] = session.ContextProjectID.String()
		input["workflow_id"] = session.ContextProjectID.String() // For ctx.targetProjectId in tools
	}
	// Add conversation history for memory-enabled agents
	input["history"] = convertMessagesToHistory(session.Messages)
	inputJSON, _ := json.Marshal(input)

	// Execute workflow synchronously (without SSE streaming)
	runner := h.inlineRunner.Create()
	run, err := runner.RunWithEvents(ctx, engine.RunInput{
		TenantID:    tenantID,
		ProjectID:   copilotProjectID,
		Input:       inputJSON,
		TriggeredBy: domain.TriggerTypeInternal,
		UserID:      userIDPtr,
		StartStepID: startStepID,
	}, nil) // nil = no event streaming

	if err != nil {
		Error(w, http.StatusInternalServerError, "AGENT_RUN_FAILED", err.Error(), nil)
		return
	}

	// Extract response from run output
	var response string
	var toolsUsed []string
	if run.Output != nil {
		var output map[string]interface{}
		if err := json.Unmarshal(run.Output, &output); err == nil {
			if resp, ok := output["response"].(string); ok {
				response = resp
			}
			if tools, ok := output["tools_used"].([]interface{}); ok {
				for _, t := range tools {
					if ts, ok := t.(string); ok {
						toolsUsed = append(toolsUsed, ts)
					}
				}
			}
		}
	}

	// Save messages to session
	h.saveMessages(ctx, tenantID, sessionID, req.Content, response, nil)

	JSON(w, http.StatusOK, map[string]interface{}{
		"session_id":   sessionID.String(),
		"response":     response,
		"tools_used":   toolsUsed,
		"run_id":       run.ID.String(), // Include run ID for history tracking
	})
}

// StreamAgentMessage handles GET /api/v1/projects/{project_id}/copilot/agent/sessions/{session_id}/stream
// This endpoint establishes an SSE connection for streaming agent responses.
// It executes the Copilot workflow via the standard workflow engine, creating a Run record.
func (h *CopilotAgentHandler) StreamAgentMessage(w http.ResponseWriter, r *http.Request) {
	reqCtx := r.Context()
	tenantID := middleware.GetTenantID(reqCtx)
	userID := middleware.GetUserID(reqCtx)
	var userIDPtr *uuid.UUID
	if userID != uuid.Nil {
		userIDPtr = &userID
	}

	sessionID, err := uuid.Parse(chi.URLParam(r, "session_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_SESSION_ID", "Invalid session ID", nil)
		return
	}

	// Get message from query parameter for GET request
	message := r.URL.Query().Get("message")
	if message == "" {
		Error(w, http.StatusBadRequest, "INVALID_REQUEST", "message query parameter is required", nil)
		return
	}

	// Get session
	session, err := h.sessionRepo.GetWithMessages(reqCtx, tenantID, sessionID)
	if err != nil {
		if errors.Is(err, domain.ErrCopilotSessionNotFound) {
			Error(w, http.StatusNotFound, "SESSION_NOT_FOUND", "Session not found", nil)
			return
		}
		Error(w, http.StatusInternalServerError, "GET_SESSION_FAILED", err.Error(), nil)
		return
	}

	// Get Copilot project
	copilotProjectID, startStepID, err := h.getCopilotProject(reqCtx)
	if err != nil {
		h.logger.Error("Failed to get copilot project", "error", err)
		Error(w, http.StatusInternalServerError, "COPILOT_WORKFLOW_NOT_FOUND", "Copilot workflow not found", nil)
		return
	}

	if startStepID == nil {
		Error(w, http.StatusInternalServerError, "COPILOT_WORKFLOW_INVALID", "Copilot workflow has no start step", nil)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Create a new context that is NOT derived from the request context.
	// This bypasses the chi Timeout middleware's deadline.
	streamCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Watch for client disconnection from the original request context
	go func() {
		<-reqCtx.Done()
		cancel()
	}()

	// Store cancel function for this stream
	streamKey := fmt.Sprintf("%s-%s", sessionID.String(), userID.String())
	h.activeStreams.Store(streamKey, cancel)
	defer h.activeStreams.Delete(streamKey)

	// Create event channel for workflow execution
	eventChan := make(chan engine.ExecutionEvent, 100)

	// Build workflow input with conversation history
	input := map[string]interface{}{
		"message":    message,
		"session_id": sessionID.String(),
		"mode":       string(session.Mode),
	}
	if session.ContextProjectID != nil {
		input["project_id"] = session.ContextProjectID.String()
		input["workflow_id"] = session.ContextProjectID.String() // For ctx.targetProjectId in tools
	}
	// Add conversation history for memory-enabled agents
	input["history"] = convertMessagesToHistory(session.Messages)
	inputJSON, _ := json.Marshal(input)

	// Start workflow execution in goroutine
	type runResult struct {
		run *domain.Run
		err error
	}
	runResultChan := make(chan runResult, 1)

	go func() {
		runner := h.inlineRunner.Create()
		run, err := runner.RunWithEvents(streamCtx, engine.RunInput{
			TenantID:    tenantID,
			ProjectID:   copilotProjectID,
			Input:       inputJSON,
			TriggeredBy: domain.TriggerTypeInternal,
			UserID:      userIDPtr,
			StartStepID: startStepID,
		}, eventChan)
		runResultChan <- runResult{run: run, err: err}
	}()

	// Stream events to client
	flusher, ok := w.(http.Flusher)
	if !ok {
		Error(w, http.StatusInternalServerError, "STREAMING_NOT_SUPPORTED", "Streaming not supported", nil)
		return
	}

	// Heartbeat ticker to keep the connection alive
	heartbeat := time.NewTicker(10 * time.Second)
	defer heartbeat.Stop()

	// Track tool executions for saving to session
	var toolExecutions []map[string]interface{}
	var finalResponse string

	for {
		select {
		case <-streamCtx.Done():
			return

		case <-heartbeat.C:
			fmt.Fprintf(w, "event: ping\ndata: {}\n\n")
			flusher.Flush()

		case event, ok := <-eventChan:
			if !ok {
				// Channel closed, get final result
				result := <-runResultChan

				if result.err != nil {
					h.sendSSEEvent(w, flusher, "error", map[string]interface{}{
						"error": result.err.Error(),
					})
				}

				// If we didn't capture finalResponse from events, try from run output
				if finalResponse == "" && result.run != nil && result.run.Output != nil {
					var output map[string]interface{}
					if err := json.Unmarshal(result.run.Output, &output); err == nil {
						if resp, ok := output["response"].(string); ok {
							finalResponse = resp
						}
					}
				}

				// Send complete event with run_id
				if result.run != nil {
					h.sendSSEEvent(w, flusher, "complete", map[string]interface{}{
						"response": finalResponse,
						"run_id":   result.run.ID.String(),
					})
				}

				// Save messages to session
				h.saveMessages(streamCtx, tenantID, sessionID, message, finalResponse, toolExecutions)

				// Send done event
				fmt.Fprintf(w, "event: done\ndata: {}\n\n")
				flusher.Flush()
				return
			}

			// Extract response from complete event
			if event.Type == engine.EventComplete {
				var completeData map[string]interface{}
				if err := json.Unmarshal(event.Data, &completeData); err == nil {
					if resp, ok := completeData["response"].(string); ok && resp != "" {
						finalResponse = resp
					}
				}
			}

			// Map workflow event to SSE event
			h.forwardWorkflowEvent(w, flusher, event, &toolExecutions)
		}
	}
}

// forwardWorkflowEvent converts a workflow execution event to an SSE event
func (h *CopilotAgentHandler) forwardWorkflowEvent(w http.ResponseWriter, flusher http.Flusher, event engine.ExecutionEvent, toolExecutions *[]map[string]interface{}) {
	var eventType string
	var eventData json.RawMessage

	switch event.Type {
	case engine.EventThinking:
		eventType = "thinking"
		eventData = event.Data

	case engine.EventToolCall:
		eventType = "tool_call"
		eventData = event.Data
		// Track tool call
		var toolCall map[string]interface{}
		if err := json.Unmarshal(event.Data, &toolCall); err == nil {
			*toolExecutions = append(*toolExecutions, map[string]interface{}{
				"type":      "call",
				"tool_name": toolCall["tool_name"],
				"arguments": toolCall["arguments"],
				"timestamp": event.Timestamp,
			})
		}

	case engine.EventToolResult:
		eventType = "tool_result"
		eventData = event.Data
		// Track tool result
		var toolResult map[string]interface{}
		if err := json.Unmarshal(event.Data, &toolResult); err == nil {
			*toolExecutions = append(*toolExecutions, map[string]interface{}{
				"type":      "result",
				"tool_name": toolResult["tool_name"],
				"result":    toolResult["result"],
				"is_error":  toolResult["is_error"],
				"timestamp": event.Timestamp,
			})
		}

	case engine.EventPartialText:
		eventType = "partial_text"
		eventData = event.Data

	case engine.EventStepStarted:
		eventType = "step_started"
		eventData = event.Data

	case engine.EventStepCompleted:
		eventType = "step_completed"
		eventData = event.Data

	case engine.EventComplete:
		eventType = "complete"
		eventData = event.Data

	default:
		// Skip unknown events
		return
	}

	// Wrap in AgentStreamEvent format for frontend compatibility
	wrapped := AgentStreamEvent{
		Type: eventType,
		Data: eventData,
	}
	data, _ := json.Marshal(wrapped)

	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(data))
	flusher.Flush()
}

// sendSSEEvent sends a single SSE event
func (h *CopilotAgentHandler) sendSSEEvent(w http.ResponseWriter, flusher http.Flusher, eventType string, data interface{}) {
	dataBytes, _ := json.Marshal(data)
	wrapped := AgentStreamEvent{
		Type: eventType,
		Data: dataBytes,
	}
	wrappedBytes, _ := json.Marshal(wrapped)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(wrappedBytes))
	flusher.Flush()
}

// saveMessages saves user and assistant messages to the session
func (h *CopilotAgentHandler) saveMessages(ctx context.Context, tenantID, sessionID uuid.UUID, userMessage, assistantResponse string, toolExecutions []map[string]interface{}) {
	// Add user message
	userMsg := domain.NewCopilotMessage(sessionID, "user", userMessage)
	if err := h.sessionRepo.AddMessage(ctx, userMsg); err != nil {
		h.logger.Warn("Failed to save user message", "error", err)
	}

	// Add assistant message with tool execution history
	assistantMsg := domain.NewCopilotMessage(sessionID, "assistant", assistantResponse)
	if len(toolExecutions) > 0 {
		extractedData, _ := json.Marshal(map[string]interface{}{
			"tool_executions": toolExecutions,
		})
		assistantMsg.ExtractedData = extractedData
	}
	if err := h.sessionRepo.AddMessage(ctx, assistantMsg); err != nil {
		h.logger.Warn("Failed to save assistant message", "error", err)
	}
}

// convertMessagesToHistory converts session messages to the format expected by the workflow engine
// Only user and assistant messages are included (system messages are handled by the agent's system prompt)
func convertMessagesToHistory(messages []domain.CopilotMessage) []map[string]string {
	history := make([]map[string]string, 0, len(messages))
	for _, msg := range messages {
		if msg.Role == "user" || msg.Role == "assistant" {
			history = append(history, map[string]string{
				"role":    msg.Role,
				"content": msg.Content,
			})
		}
	}
	return history
}

// CancelAgentStream handles POST /api/v1/projects/{project_id}/copilot/agent/sessions/{session_id}/cancel
func (h *CopilotAgentHandler) CancelAgentStream(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	sessionID, err := uuid.Parse(chi.URLParam(r, "session_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_SESSION_ID", "Invalid session ID", nil)
		return
	}

	// Find and cancel the stream
	streamKey := fmt.Sprintf("%s-%s", sessionID.String(), userID.String())
	if cancel, ok := h.activeStreams.LoadAndDelete(streamKey); ok {
		cancel.(context.CancelFunc)()
		JSON(w, http.StatusOK, map[string]interface{}{
			"status":  "cancelled",
			"message": "Agent stream cancelled successfully",
		})
		return
	}

	Error(w, http.StatusNotFound, "STREAM_NOT_FOUND", "No active stream found for this session", nil)
}

// GetAvailableTools handles GET /api/v1/copilot/agent/tools
// Note: Tools are now defined as child steps in the Copilot workflow
func (h *CopilotAgentHandler) GetAvailableTools(w http.ResponseWriter, r *http.Request) {
	// Get Copilot project to extract tool definitions from child steps
	ctx := r.Context()
	copilotProjectID, _, err := h.getCopilotProject(ctx)
	if err != nil {
		Error(w, http.StatusInternalServerError, "COPILOT_WORKFLOW_NOT_FOUND", "Copilot workflow not found", nil)
		return
	}

	// Get project with steps
	tenantID := middleware.GetTenantID(ctx)
	project, err := h.projectRepo.GetWithStepsAndEdges(ctx, tenantID, copilotProjectID)
	if err != nil {
		Error(w, http.StatusInternalServerError, "GET_PROJECT_FAILED", err.Error(), nil)
		return
	}

	// Extract tool definitions from function steps in block groups
	tools := make([]map[string]interface{}, 0)
	for _, step := range project.Steps {
		if step.Type == domain.StepTypeFunction && step.BlockGroupID != nil {
			var config map[string]interface{}
			if err := json.Unmarshal(step.Config, &config); err == nil {
				tool := map[string]interface{}{
					"name":        step.Name,
					"description": config["description"],
				}
				if schema, ok := config["input_schema"]; ok {
					tool["input_schema"] = schema
				}
				tools = append(tools, tool)
			}
		}
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"tools": tools,
		"count": len(tools),
	})
}
