package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/engine"
	"github.com/souta/ai-orchestration/internal/middleware"
	"github.com/souta/ai-orchestration/internal/usecase"
)

// RunStreamHandler handles SSE streaming for workflow execution
type RunStreamHandler struct {
	runUsecase    *usecase.RunUsecase
	runnerFactory *engine.InlineRunnerFactory
}

// NewRunStreamHandler creates a new run stream handler
func NewRunStreamHandler(runUsecase *usecase.RunUsecase, factory *engine.InlineRunnerFactory) *RunStreamHandler {
	return &RunStreamHandler{
		runUsecase:    runUsecase,
		runnerFactory: factory,
	}
}

// StreamRunExecution handles GET /runs/{run_id}/stream
// This endpoint is available for ALL workflows, not just Copilot
// It streams execution events via Server-Sent Events (SSE)
// Note: This endpoint monitors an existing run - it doesn't start execution
func (h *RunStreamHandler) StreamRunExecution(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)

	runIDStr := chi.URLParam(r, "run_id")
	runID, err := uuid.Parse(runIDStr)
	if err != nil {
		http.Error(w, "Invalid run_id", http.StatusBadRequest)
		return
	}

	// Verify run exists and belongs to tenant
	run, err := h.runUsecase.GetByID(ctx, tenantID, runID)
	if err != nil {
		http.Error(w, "Run not found", http.StatusNotFound)
		return
	}
	if run == nil {
		http.Error(w, "Run not found", http.StatusNotFound)
		return
	}

	// Set SSE headers
	h.setSSEHeaders(w)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send initial connected event with run status
	h.sendSSEEvent(w, flusher, "connected", map[string]interface{}{
		"run_id":     runID.String(),
		"status":     run.Status,
		"project_id": run.ProjectID.String(),
	})

	// If run is already completed or failed, send the final status immediately
	if run.Status == domain.RunStatusCompleted || run.Status == domain.RunStatusFailed {
		eventType := "run:completed"
		if run.Status == domain.RunStatusFailed {
			eventType = "run:failed"
		}
		h.sendSSEEvent(w, flusher, eventType, map[string]interface{}{
			"status": run.Status,
			"output": run.Output,
			"error":  run.Error,
		})
		h.sendSSEEvent(w, flusher, "stream_end", map[string]interface{}{
			"reason": "run_already_completed",
		})
		return
	}

	// For pending/running runs, poll for updates
	// This is a simple polling approach - can be enhanced with pub/sub later
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer heartbeatTicker.Stop()

	lastStatus := run.Status

	for {
		select {
		case <-ticker.C:
			// Poll for run status updates
			run, err = h.runUsecase.GetByID(ctx, tenantID, runID)
			if err != nil {
				h.sendSSEEvent(w, flusher, "error", map[string]interface{}{
					"error": "Failed to get run status",
				})
				return
			}

			// Send status update if changed
			if run.Status != lastStatus {
				h.sendSSEEvent(w, flusher, "status_change", map[string]interface{}{
					"old_status": lastStatus,
					"new_status": run.Status,
				})
				lastStatus = run.Status
			}

			// Check for terminal states
			if run.Status == domain.RunStatusCompleted {
				h.sendSSEEvent(w, flusher, "run:completed", map[string]interface{}{
					"output": run.Output,
				})
				h.sendSSEEvent(w, flusher, "stream_end", map[string]interface{}{
					"reason": "run:completed",
				})
				return
			}
			if run.Status == domain.RunStatusFailed {
				h.sendSSEEvent(w, flusher, "run:failed", map[string]interface{}{
					"error": run.Error,
				})
				h.sendSSEEvent(w, flusher, "stream_end", map[string]interface{}{
					"reason": "run:failed",
				})
				return
			}
			if run.Status == domain.RunStatusCancelled {
				h.sendSSEEvent(w, flusher, "run:cancelled", map[string]interface{}{})
				h.sendSSEEvent(w, flusher, "stream_end", map[string]interface{}{
					"reason": "run:cancelled",
				})
				return
			}

		case <-heartbeatTicker.C:
			h.sendSSEEvent(w, flusher, "heartbeat", map[string]interface{}{
				"timestamp": time.Now().Unix(),
			})

		case <-r.Context().Done():
			return
		}
	}
}

// CreateAndStreamRunRequest represents the request body for creating and streaming a run
type CreateAndStreamRunRequest struct {
	ProjectID   string                 `json:"project_id"`
	Input       map[string]interface{} `json:"input"`
	StartStepID string                 `json:"start_step_id"` // Required: which Start block to execute from
}

// CreateAndStreamRun handles POST /runs/stream
// Creates a new run and streams its execution via SSE
// This is a combined endpoint for creating and streaming in one request
func (h *RunStreamHandler) CreateAndStreamRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.GetTenantID(ctx)
	userID := middleware.GetUserID(ctx)

	// Parse request body
	var req CreateAndStreamRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		http.Error(w, "Invalid project_id", http.StatusBadRequest)
		return
	}

	var startStepID *uuid.UUID
	if req.StartStepID != "" {
		id, err := uuid.Parse(req.StartStepID)
		if err != nil {
			http.Error(w, "Invalid start_step_id", http.StatusBadRequest)
			return
		}
		startStepID = &id
	}

	// Set SSE headers
	h.setSSEHeaders(w)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Create event channel
	eventChan := make(chan engine.ExecutionEvent, 100)

	// Create inline runner
	runner := h.runnerFactory.Create()

	// Marshal input
	inputJSON, _ := json.Marshal(req.Input)

	// Execute in goroutine
	runResultChan := make(chan *runResult, 1)
	go func() {
		run, err := runner.RunWithEvents(ctx, engine.RunInput{
			TenantID:    tenantID,
			ProjectID:   projectID,
			Input:       inputJSON,
			TriggeredBy: domain.TriggerTypeManual,
			UserID:      &userID,
			StartStepID: startStepID,
		}, eventChan)
		runResultChan <- &runResult{run: run, err: err}
	}()

	// Send initial event
	h.sendSSEEvent(w, flusher, "execution_started", map[string]interface{}{
		"project_id": projectID.String(),
		"timestamp":  time.Now().Unix(),
	})

	// Send heartbeat every 30 seconds
	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer heartbeatTicker.Stop()

	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				// Channel closed, wait for final result
				result := <-runResultChan
				if result.err != nil {
					h.sendSSEEvent(w, flusher, "error", map[string]interface{}{
						"error": result.err.Error(),
					})
				}
				if result.run != nil {
					h.sendSSEEvent(w, flusher, "run_completed", map[string]interface{}{
						"run_id": result.run.ID.String(),
						"status": result.run.Status,
						"output": result.run.Output,
					})
				}
				h.sendSSEEvent(w, flusher, "stream_end", map[string]interface{}{
					"reason": "execution_completed",
				})
				return
			}

			// Send event
			h.sendSSEEvent(w, flusher, string(event.Type), json.RawMessage(event.Data))

		case <-heartbeatTicker.C:
			h.sendSSEEvent(w, flusher, "heartbeat", map[string]interface{}{
				"timestamp": time.Now().Unix(),
			})

		case <-r.Context().Done():
			return
		}
	}
}

type runResult struct {
	run *domain.Run
	err error
}

// setSSEHeaders sets the required headers for SSE
func (h *RunStreamHandler) setSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no")
}

// sendSSEEvent sends an event in SSE format
func (h *RunStreamHandler) sendSSEEvent(w http.ResponseWriter, flusher http.Flusher, eventType string, data interface{}) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return
	}

	fmt.Fprintf(w, "event: %s\n", eventType)
	fmt.Fprintf(w, "data: %s\n\n", dataBytes)
	flusher.Flush()
}
