package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/usecase"
)

// WebhookHandler handles webhook HTTP requests (public, no auth required)
type WebhookHandler struct {
	runUsecase  *usecase.RunUsecase
	stepUsecase *usecase.StepUsecase
}

// NewWebhookHandler creates a new WebhookHandler
func NewWebhookHandler(runUsecase *usecase.RunUsecase, stepUsecase *usecase.StepUsecase) *WebhookHandler {
	return &WebhookHandler{
		runUsecase:  runUsecase,
		stepUsecase: stepUsecase,
	}
}

// WebhookResponse represents the response from webhook trigger
type WebhookResponse struct {
	RunID  string `json:"run_id"`
	Status string `json:"status"`
}

// Trigger handles POST /projects/{project_id}/webhook/{step_id}
// This is a public endpoint that triggers workflow execution via webhook
func (h *WebhookHandler) Trigger(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse project ID
	projectIDStr := chi.URLParam(r, "project_id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid project_id"}`, http.StatusBadRequest)
		return
	}

	// Parse step ID (the webhook trigger step)
	stepIDStr := chi.URLParam(r, "step_id")
	stepID, err := uuid.Parse(stepIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid step_id"}`, http.StatusBadRequest)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error": "failed to read request body"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Get step to retrieve tenant_id and trigger config
	step, err := h.stepUsecase.GetByIDOnly(ctx, stepID)
	if err != nil {
		if err == domain.ErrStepNotFound {
			http.Error(w, `{"error": "webhook not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}

	// Verify project ID matches
	if step.ProjectID != projectID {
		http.Error(w, `{"error": "webhook not found"}`, http.StatusNotFound)
		return
	}

	// Verify this is a webhook trigger step
	if step.TriggerType == nil || *step.TriggerType != domain.StepTriggerTypeWebhook {
		http.Error(w, `{"error": "step is not a webhook trigger"}`, http.StatusBadRequest)
		return
	}

	// Parse trigger config
	var triggerConfig domain.WebhookTriggerConfig
	if step.TriggerConfig != nil {
		if err := json.Unmarshal(step.TriggerConfig, &triggerConfig); err != nil {
			http.Error(w, `{"error": "invalid trigger configuration"}`, http.StatusInternalServerError)
			return
		}
	}

	// Check if webhook is enabled (default: enabled when field is not set)
	// Note: Enabled is a bool, so false means disabled
	if step.TriggerConfig != nil && !triggerConfig.Enabled {
		http.Error(w, `{"error": "webhook is disabled"}`, http.StatusConflict)
		return
	}

	// Verify signature if secret is configured
	if triggerConfig.Secret != "" {
		signature := r.Header.Get("X-Webhook-Signature")
		if signature == "" {
			http.Error(w, `{"error": "missing signature"}`, http.StatusUnauthorized)
			return
		}
		if !verifyWebhookSignature(body, triggerConfig.Secret, signature) {
			http.Error(w, `{"error": "invalid signature"}`, http.StatusUnauthorized)
			return
		}
	}

	// Parse input from request body
	var input json.RawMessage
	if len(body) > 0 {
		input = body
	} else {
		input = json.RawMessage(`{}`)
	}

	// Create run
	run, err := h.runUsecase.Create(ctx, usecase.CreateRunInput{
		TenantID:    step.TenantID,
		ProjectID:   projectID,
		Input:       input,
		TriggeredBy: domain.TriggerTypeWebhook,
		StartStepID: &stepID,
	})
	if err != nil {
		if err == domain.ErrStepNotFound {
			http.Error(w, `{"error": "workflow not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error": "failed to trigger workflow"}`, http.StatusInternalServerError)
		return
	}

	// Return response
	resp := WebhookResponse{
		RunID:  run.ID.String(),
		Status: string(run.Status),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// verifyWebhookSignature verifies HMAC-SHA256 signature
func verifyWebhookSignature(body []byte, secret, signature string) bool {
	// Support both "sha256=xxx" and plain "xxx" formats
	signature = strings.TrimPrefix(signature, "sha256=")

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedMAC))
}
