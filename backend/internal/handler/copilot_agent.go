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
	"github.com/souta/ai-orchestration/internal/copilot/agent"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/middleware"
)

// CopilotAgentHandler handles agent-based copilot API requests
type CopilotAgentHandler struct {
	agentUsecase *agent.AgentUsecase

	// Active streams for cancellation
	activeStreams sync.Map
}

// NewCopilotAgentHandler creates a new CopilotAgentHandler
func NewCopilotAgentHandler(agentUsecase *agent.AgentUsecase) *CopilotAgentHandler {
	return &CopilotAgentHandler{
		agentUsecase: agentUsecase,
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
// Handlers
// ============================================================================

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

	// Create session only (don't run agent yet - that will happen via SSE stream)
	output, err := h.agentUsecase.CreateAgentSessionOnly(ctx, agent.StartAgentSessionInput{
		TenantID:         tenantID,
		UserID:           userID.String(),
		ContextProjectID: &projectID,
		Mode:             mode,
		InitialPrompt:    req.InitialPrompt,
	})
	if err != nil {
		if errors.Is(err, domain.ErrProjectNotFound) {
			Error(w, http.StatusNotFound, "PROJECT_NOT_FOUND", "Project not found", nil)
			return
		}
		Error(w, http.StatusInternalServerError, "START_SESSION_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusCreated, StartAgentSessionResponse{
		SessionID: output.Session.ID.String(),
		Response:  "", // Empty - will be populated via SSE stream
		ToolsUsed: []string{},
		Status:    string(output.Session.Status),
		Phase:     string(output.Session.HearingPhase),
		Progress:  output.Session.HearingProgress,
	})
}

// SendAgentMessage handles POST /api/v1/projects/{project_id}/copilot/agent/sessions/{session_id}/messages
func (h *CopilotAgentHandler) SendAgentMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)
	userID := middleware.GetUserID(ctx)

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

	output, err := h.agentUsecase.RunAgent(ctx, agent.RunAgentInput{
		TenantID:  tenantID,
		UserID:    userID.String(),
		SessionID: sessionID,
		Message:   req.Content,
	})
	if err != nil {
		Error(w, http.StatusInternalServerError, "AGENT_RUN_FAILED", err.Error(), nil)
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"session_id":  output.SessionID.String(),
		"response":    output.Response,
		"tools_used":  output.ToolsUsed,
		"iterations":  output.Iterations,
		"total_tokens": output.TotalTokens,
	})
}

// StreamAgentMessage handles GET /api/v1/projects/{project_id}/copilot/agent/sessions/{session_id}/stream
// This endpoint establishes an SSE connection for streaming agent responses
func (h *CopilotAgentHandler) StreamAgentMessage(w http.ResponseWriter, r *http.Request) {
	reqCtx := r.Context()
	tenantID := middleware.GetTenantID(reqCtx)
	userID := middleware.GetUserID(reqCtx)

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

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Create a new context that is NOT derived from the request context.
	// This bypasses the chi Timeout middleware's deadline.
	// We'll manually watch for client disconnection via reqCtx.Done().
	streamCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Watch for client disconnection from the original request context
	go func() {
		<-reqCtx.Done()
		cancel() // Cancel our stream context when the client disconnects
	}()

	// Store cancel function for this stream
	streamKey := fmt.Sprintf("%s-%s", sessionID.String(), userID.String())
	h.activeStreams.Store(streamKey, cancel)
	defer h.activeStreams.Delete(streamKey)

	// Create event channel
	events := make(chan agent.Event, 100)

	// Start agent in goroutine
	go func() {
		defer close(events)

		_, err := h.agentUsecase.RunAgentWithStreaming(streamCtx, agent.RunAgentInput{
			TenantID:  tenantID,
			UserID:    userID.String(),
			SessionID: sessionID,
			Message:   message,
		}, events)

		if err != nil {
			slog.Error("agent streaming failed", "error", err, "session_id", sessionID)
			// Send error event
			errData, _ := json.Marshal(map[string]string{"error": err.Error()})
			select {
			case events <- agent.Event{
				Type:      agent.EventTypeError,
				Timestamp: time.Now(),
				Data:      errData,
			}:
			default:
			}
		}
	}()

	// Stream events to client
	flusher, ok := w.(http.Flusher)
	if !ok {
		Error(w, http.StatusInternalServerError, "STREAMING_NOT_SUPPORTED", "Streaming not supported", nil)
		return
	}

	// Heartbeat ticker to keep the connection alive during long operations
	heartbeat := time.NewTicker(10 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-streamCtx.Done():
			// Client disconnected or cancelled
			return
		case <-heartbeat.C:
			// Send heartbeat to keep connection alive
			fmt.Fprintf(w, "event: ping\ndata: {}\n\n")
			flusher.Flush()
		case event, ok := <-events:
			if !ok {
				// Channel closed, send done event
				fmt.Fprintf(w, "event: done\ndata: {}\n\n")
				flusher.Flush()
				return
			}

			// Format SSE event
			eventData := AgentStreamEvent{
				Type: string(event.Type),
				Data: event.Data,
			}
			data, _ := json.Marshal(eventData)

			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, string(data))
			flusher.Flush()
		}
	}
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
func (h *CopilotAgentHandler) GetAvailableTools(w http.ResponseWriter, r *http.Request) {
	tools := h.agentUsecase.GetAvailableTools()
	JSON(w, http.StatusOK, map[string]interface{}{
		"tools": tools,
		"count": len(tools),
	})
}
