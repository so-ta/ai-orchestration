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
	ProjectID      string          `json:"project_id"`
	Name           string          `json:"name"`
	Description    string          `json:"description,omitempty"`
	CronExpression string          `json:"cron_expression"`
	Timezone       string          `json:"timezone,omitempty"`
	Input          json.RawMessage `json:"input,omitempty"`
	StartStepID    *string         `json:"start_step_id,omitempty"` // Optional: start execution from a specific step
}

// Create creates a new schedule
func (h *ScheduleHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateScheduleRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	projectID, ok := parseUUIDString(w, req.ProjectID, "project ID")
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

	// Parse start_step_id (required for multi-start projects)
	if req.StartStepID == nil || *req.StartStepID == "" {
		HandleError(w, domain.NewValidationError("start_step_id", "start_step_id is required"))
		return
	}
	startStepID, ok := parseUUIDString(w, *req.StartStepID, "start_step_id")
	if !ok {
		return
	}

	schedule, err := h.usecase.Create(r.Context(), usecase.CreateScheduleInput{
		TenantID:       tenantID,
		ProjectID:      projectID,
		Name:           req.Name,
		Description:    req.Description,
		CronExpression: req.CronExpression,
		Timezone:       req.Timezone,
		Input:          req.Input,
		StartStepID:    startStepID,
		CreatedBy:      createdBy,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionScheduleCreate, domain.AuditResourceSchedule, &schedule.ID, map[string]interface{}{
		"name":       schedule.Name,
		"project_id": projectID,
		"cron":       schedule.CronExpression,
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

	// Optional project filter
	if projectIDStr := r.URL.Query().Get("project_id"); projectIDStr != "" {
		if projectID, err := uuid.Parse(projectIDStr); err == nil {
			input.ProjectID = &projectID
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
	StartStepID    *string         `json:"start_step_id,omitempty"` // Optional: start execution from a specific step
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

	// Parse start_step_id if provided
	var startStepID *uuid.UUID
	if req.StartStepID != nil && *req.StartStepID != "" {
		stepID, ok := parseUUIDString(w, *req.StartStepID, "start_step_id")
		if !ok {
			return
		}
		startStepID = &stepID
	}

	schedule, err := h.usecase.Update(r.Context(), usecase.UpdateScheduleInput{
		TenantID:       tenantID,
		ID:             id,
		Name:           req.Name,
		Description:    req.Description,
		CronExpression: req.CronExpression,
		Timezone:       req.Timezone,
		Input:          req.Input,
		StartStepID:    startStepID,
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
