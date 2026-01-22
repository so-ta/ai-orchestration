package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/usecase"
)

// AuditHandler handles HTTP requests for audit logs
type AuditHandler struct {
	service *usecase.AuditService
}

// NewAuditHandler creates a new AuditHandler
func NewAuditHandler(service *usecase.AuditService) *AuditHandler {
	return &AuditHandler{service: service}
}

// List lists audit logs
func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	input := usecase.ListAuditLogsInput{
		TenantID: tenantID,
		Page:     parseIntQuery(r, "page", 1),
		Limit:    parseIntQuery(r, "limit", 50),
	}

	// Optional filters
	if actorIDStr := r.URL.Query().Get("actor_id"); actorIDStr != "" {
		if actorID, err := uuid.Parse(actorIDStr); err == nil {
			input.ActorID = &actorID
		}
	}

	if actionStr := r.URL.Query().Get("action"); actionStr != "" {
		action := domain.AuditAction(actionStr)
		input.Action = &action
	}

	if resourceTypeStr := r.URL.Query().Get("resource_type"); resourceTypeStr != "" {
		resourceType := domain.AuditResourceType(resourceTypeStr)
		input.ResourceType = &resourceType
	}

	if resourceIDStr := r.URL.Query().Get("resource_id"); resourceIDStr != "" {
		if resourceID, err := uuid.Parse(resourceIDStr); err == nil {
			input.ResourceID = &resourceID
		}
	}

	if startTimeStr := r.URL.Query().Get("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			input.StartTime = &startTime
		}
	}

	if endTimeStr := r.URL.Query().Get("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			input.EndTime = &endTime
		}
	}

	output, err := h.service.List(r.Context(), input)
	if err != nil {
		HandleErrorL(w, r, err)
		return
	}

	JSONList(w, http.StatusOK, output.Logs, output.Page, output.Limit, output.Total)
}

// GetByResource gets audit logs for a specific resource
func (h *AuditHandler) GetByResource(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	resourceTypeStr := chi.URLParam(r, "resource_type")
	resourceIDStr := chi.URLParam(r, "resource_id")

	resourceType := domain.AuditResourceType(resourceTypeStr)
	resourceID, err := uuid.Parse(resourceIDStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid resource ID", nil)
		return
	}

	logs, err := h.service.ListByResource(r.Context(), tenantID, resourceType, resourceID)
	if err != nil {
		HandleErrorL(w, r, err)
		return
	}

	JSON(w, http.StatusOK, logs)
}
