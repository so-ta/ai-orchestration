package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/usecase"
)

// WebhookHandler handles HTTP requests for webhooks
type WebhookHandler struct {
	usecase *usecase.WebhookUsecase
}

// NewWebhookHandler creates a new WebhookHandler
func NewWebhookHandler(usecase *usecase.WebhookUsecase) *WebhookHandler {
	return &WebhookHandler{usecase: usecase}
}

// CreateWebhookRequest represents the request body for creating a webhook
type CreateWebhookRequest struct {
	WorkflowID   string          `json:"workflow_id"`
	Name         string          `json:"name"`
	Description  string          `json:"description,omitempty"`
	InputMapping json.RawMessage `json:"input_mapping,omitempty"`
}

// Create creates a new webhook
func (h *WebhookHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body", nil)
		return
	}

	workflowID, err := uuid.Parse(req.WorkflowID)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_WORKFLOW_ID", "Invalid workflow ID", nil)
		return
	}

	tenantID := getTenantID(r)
	userID := getUserID(r)

	webhook, err := h.usecase.Create(r.Context(), usecase.CreateWebhookInput{
		TenantID:     tenantID,
		WorkflowID:   workflowID,
		Name:         req.Name,
		Description:  req.Description,
		InputMapping: req.InputMapping,
		CreatedBy:    &userID,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusCreated, webhook)
}

// Get retrieves a webhook by ID
func (h *WebhookHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "webhook_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid webhook ID", nil)
		return
	}

	tenantID := getTenantID(r)

	webhook, err := h.usecase.GetByID(r.Context(), tenantID, id)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, webhook)
}

// List lists webhooks
func (h *WebhookHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	input := usecase.ListWebhooksInput{
		TenantID: tenantID,
		Page:     parseIntQuery(r, "page", 1),
		Limit:    parseIntQuery(r, "limit", 20),
	}

	// Optional workflow filter
	if workflowIDStr := r.URL.Query().Get("workflow_id"); workflowIDStr != "" {
		if workflowID, err := uuid.Parse(workflowIDStr); err == nil {
			input.WorkflowID = &workflowID
		}
	}

	// Optional enabled filter
	if enabledStr := r.URL.Query().Get("enabled"); enabledStr != "" {
		enabled := enabledStr == "true"
		input.Enabled = &enabled
	}

	output, err := h.usecase.List(r.Context(), input)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONList(w, http.StatusOK, output.Webhooks, output.Page, output.Limit, output.Total)
}

// UpdateWebhookRequest represents the request body for updating a webhook
type UpdateWebhookRequest struct {
	Name         string          `json:"name,omitempty"`
	Description  string          `json:"description,omitempty"`
	InputMapping json.RawMessage `json:"input_mapping,omitempty"`
}

// Update updates a webhook
func (h *WebhookHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "webhook_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid webhook ID", nil)
		return
	}

	var req UpdateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body", nil)
		return
	}

	tenantID := getTenantID(r)

	webhook, err := h.usecase.Update(r.Context(), usecase.UpdateWebhookInput{
		TenantID:     tenantID,
		ID:           id,
		Name:         req.Name,
		Description:  req.Description,
		InputMapping: req.InputMapping,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, webhook)
}

// Delete deletes a webhook
func (h *WebhookHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "webhook_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid webhook ID", nil)
		return
	}

	tenantID := getTenantID(r)

	if err := h.usecase.Delete(r.Context(), tenantID, id); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Enable enables a webhook
func (h *WebhookHandler) Enable(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "webhook_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid webhook ID", nil)
		return
	}

	tenantID := getTenantID(r)

	webhook, err := h.usecase.Enable(r.Context(), tenantID, id)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, webhook)
}

// Disable disables a webhook
func (h *WebhookHandler) Disable(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "webhook_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid webhook ID", nil)
		return
	}

	tenantID := getTenantID(r)

	webhook, err := h.usecase.Disable(r.Context(), tenantID, id)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, webhook)
}

// RegenerateSecret regenerates the webhook secret
func (h *WebhookHandler) RegenerateSecret(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "webhook_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid webhook ID", nil)
		return
	}

	tenantID := getTenantID(r)

	webhook, err := h.usecase.RegenerateSecret(r.Context(), tenantID, id)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, webhook)
}

// Trigger handles incoming webhook requests (public endpoint)
func (h *WebhookHandler) Trigger(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "webhook_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid webhook ID", nil)
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_BODY", "Failed to read request body", nil)
		return
	}

	// Get signature from header
	signature := r.Header.Get("X-Webhook-Signature")

	run, err := h.usecase.Trigger(r.Context(), usecase.TriggerWebhookInput{
		WebhookID: id,
		Signature: signature,
		Payload:   body,
	})
	if err != nil {
		switch err {
		case domain.ErrWebhookNotFound:
			Error(w, http.StatusNotFound, "NOT_FOUND", "Webhook not found", nil)
		case domain.ErrWebhookDisabled:
			Error(w, http.StatusForbidden, "WEBHOOK_DISABLED", "Webhook is disabled", nil)
		case domain.ErrWebhookInvalidSecret:
			Error(w, http.StatusUnauthorized, "INVALID_SIGNATURE", "Invalid webhook signature", nil)
		default:
			HandleError(w, err)
		}
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"run_id": run.ID,
		"status": run.Status,
	})
}
