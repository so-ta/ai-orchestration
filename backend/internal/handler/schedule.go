package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/usecase"
)

// ScheduleHandler handles HTTP requests for schedules
type ScheduleHandler struct {
	usecase      *usecase.ScheduleUsecase
	auditService *usecase.AuditService
}

// NewScheduleHandler creates a new ScheduleHandler
func NewScheduleHandler(uc *usecase.ScheduleUsecase, auditService *usecase.AuditService) *ScheduleHandler {
	return &ScheduleHandler{
		usecase:      uc,
		auditService: auditService,
	}
}

// CreateScheduleRequest represents the request body for creating a schedule
type CreateScheduleRequest struct {
	WorkflowID     string          `json:"workflow_id"`
	Name           string          `json:"name"`
	Description    string          `json:"description,omitempty"`
	CronExpression string          `json:"cron_expression"`
	Timezone       string          `json:"timezone,omitempty"`
	Input          json.RawMessage `json:"input,omitempty"`
}

// Create creates a new schedule
func (h *ScheduleHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateScheduleRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	workflowID, ok := parseUUIDString(w, req.WorkflowID, "workflow ID")
	if !ok {
		return
	}

	tenantID := getTenantID(r)
	userID := getUserID(r)

	// Only set CreatedBy if we have a valid user ID
	var createdBy *uuid.UUID
	if userID != uuid.Nil {
		createdBy = &userID
	}

	schedule, err := h.usecase.Create(r.Context(), usecase.CreateScheduleInput{
		TenantID:       tenantID,
		WorkflowID:     workflowID,
		Name:           req.Name,
		Description:    req.Description,
		CronExpression: req.CronExpression,
		Timezone:       req.Timezone,
		Input:          req.Input,
		CreatedBy:      createdBy,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionScheduleCreate, domain.AuditResourceSchedule, &schedule.ID, map[string]interface{}{
		"name":        schedule.Name,
		"workflow_id": workflowID,
		"cron":        schedule.CronExpression,
	})

	JSONData(w, http.StatusCreated, schedule)
}

// Get retrieves a schedule by ID
func (h *ScheduleHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "schedule_id", "schedule ID")
	if !ok {
		return
	}

	tenantID := getTenantID(r)

	schedule, err := h.usecase.GetByID(r.Context(), tenantID, id)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, schedule)
}

// List lists schedules
func (h *ScheduleHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	input := usecase.ListSchedulesInput{
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

	// Optional status filter
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		status := domain.ScheduleStatus(statusStr)
		input.Status = &status
	}

	output, err := h.usecase.List(r.Context(), input)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONList(w, http.StatusOK, output.Schedules, output.Page, output.Limit, output.Total)
}

// UpdateScheduleRequest represents the request body for updating a schedule
type UpdateScheduleRequest struct {
	Name           string          `json:"name,omitempty"`
	Description    string          `json:"description,omitempty"`
	CronExpression string          `json:"cron_expression,omitempty"`
	Timezone       string          `json:"timezone,omitempty"`
	Input          json.RawMessage `json:"input,omitempty"`
}

// Update updates a schedule
func (h *ScheduleHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "schedule_id", "schedule ID")
	if !ok {
		return
	}

	var req UpdateScheduleRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	tenantID := getTenantID(r)

	schedule, err := h.usecase.Update(r.Context(), usecase.UpdateScheduleInput{
		TenantID:       tenantID,
		ID:             id,
		Name:           req.Name,
		Description:    req.Description,
		CronExpression: req.CronExpression,
		Timezone:       req.Timezone,
		Input:          req.Input,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionScheduleUpdate, domain.AuditResourceSchedule, &schedule.ID, map[string]interface{}{
		"name": schedule.Name,
	})

	JSONData(w, http.StatusOK, schedule)
}

// Delete deletes a schedule
func (h *ScheduleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "schedule_id", "schedule ID")
	if !ok {
		return
	}

	tenantID := getTenantID(r)

	if err := h.usecase.Delete(r.Context(), tenantID, id); err != nil {
		HandleError(w, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionScheduleDelete, domain.AuditResourceSchedule, &id, nil)

	w.WriteHeader(http.StatusNoContent)
}

// Pause pauses a schedule
func (h *ScheduleHandler) Pause(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "schedule_id", "schedule ID")
	if !ok {
		return
	}

	tenantID := getTenantID(r)

	schedule, err := h.usecase.Pause(r.Context(), tenantID, id)
	if err != nil {
		HandleError(w, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionSchedulePause, domain.AuditResourceSchedule, &schedule.ID, nil)

	JSONData(w, http.StatusOK, schedule)
}

// Resume resumes a paused schedule
func (h *ScheduleHandler) Resume(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "schedule_id", "schedule ID")
	if !ok {
		return
	}

	tenantID := getTenantID(r)

	schedule, err := h.usecase.Resume(r.Context(), tenantID, id)
	if err != nil {
		HandleError(w, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionScheduleResume, domain.AuditResourceSchedule, &schedule.ID, nil)

	JSONData(w, http.StatusOK, schedule)
}

// Trigger manually triggers a schedule
func (h *ScheduleHandler) Trigger(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "schedule_id", "schedule ID")
	if !ok {
		return
	}

	tenantID := getTenantID(r)

	run, err := h.usecase.Trigger(r.Context(), tenantID, id)
	if err != nil {
		HandleError(w, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionScheduleTrigger, domain.AuditResourceSchedule, &id, map[string]interface{}{
		"run_id": run.ID,
	})

	JSONData(w, http.StatusOK, run)
}
