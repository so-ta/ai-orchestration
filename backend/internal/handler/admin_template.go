package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// AdminTemplateHandler handles HTTP requests for block templates (operator-only)
type AdminTemplateHandler struct {
	repo repository.BlockTemplateRepository
}

// NewAdminTemplateHandler creates a new AdminTemplateHandler
func NewAdminTemplateHandler(repo repository.BlockTemplateRepository) *AdminTemplateHandler {
	return &AdminTemplateHandler{repo: repo}
}

// CreateBlockTemplateRequest represents the request body for creating a block template
type CreateBlockTemplateRequest struct {
	Slug         string          `json:"slug"`
	Name         string          `json:"name"`
	Description  string          `json:"description,omitempty"`
	ConfigSchema json.RawMessage `json:"config_schema"`
	ExecutorType string          `json:"executor_type"` // "builtin" or "javascript"
	ExecutorCode string          `json:"executor_code,omitempty"`
}

// BlockTemplateResponse represents the response for a block template
type BlockTemplateResponse struct {
	ID           uuid.UUID       `json:"id"`
	Slug         string          `json:"slug"`
	Name         string          `json:"name"`
	Description  string          `json:"description,omitempty"`
	ConfigSchema json.RawMessage `json:"config_schema"`
	ExecutorType string          `json:"executor_type"`
	ExecutorCode string          `json:"executor_code,omitempty"`
	IsBuiltin    bool            `json:"is_builtin"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// toResponse converts a BlockTemplate to a response
func (h *AdminTemplateHandler) toResponse(template *domain.BlockTemplate) *BlockTemplateResponse {
	return &BlockTemplateResponse{
		ID:           template.ID,
		Slug:         template.Slug,
		Name:         template.Name,
		Description:  template.Description,
		ConfigSchema: template.ConfigSchema,
		ExecutorType: string(template.ExecutorType),
		ExecutorCode: template.ExecutorCode,
		IsBuiltin:    template.IsBuiltin,
		CreatedAt:    template.CreatedAt,
		UpdatedAt:    template.UpdatedAt,
	}
}

// Create creates a new block template
func (h *AdminTemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateBlockTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body", nil)
		return
	}

	// Validate required fields
	if req.Slug == "" {
		Error(w, http.StatusBadRequest, "MISSING_SLUG", "Slug is required", nil)
		return
	}
	if req.Name == "" {
		Error(w, http.StatusBadRequest, "MISSING_NAME", "Name is required", nil)
		return
	}

	// Validate executor type
	execType := domain.TemplateExecutorType(req.ExecutorType)
	if execType != domain.TemplateExecutorBuiltin && execType != domain.TemplateExecutorJavaScript {
		Error(w, http.StatusBadRequest, "INVALID_EXECUTOR_TYPE", "Executor type must be 'builtin' or 'javascript'", nil)
		return
	}

	// JavaScript templates require executor code
	if execType == domain.TemplateExecutorJavaScript && req.ExecutorCode == "" {
		Error(w, http.StatusBadRequest, "MISSING_EXECUTOR_CODE", "Executor code is required for JavaScript templates", nil)
		return
	}

	// Check if slug already exists
	existing, err := h.repo.GetBySlug(r.Context(), req.Slug)
	if err == nil && existing != nil {
		Error(w, http.StatusConflict, "SLUG_EXISTS", "A template with this slug already exists", nil)
		return
	}

	// Create template
	template := domain.NewBlockTemplate(req.Slug, req.Name)
	template.Description = req.Description
	template.ExecutorType = execType
	template.ExecutorCode = req.ExecutorCode
	template.IsBuiltin = false // Custom templates are never builtin

	if req.ConfigSchema != nil && len(req.ConfigSchema) > 0 {
		template.ConfigSchema = req.ConfigSchema
	}

	if err := h.repo.Create(r.Context(), template); err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusCreated, h.toResponse(template))
}

// Get retrieves a block template by ID
func (h *AdminTemplateHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "template_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid template ID", nil)
		return
	}

	template, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, h.toResponse(template))
}

// GetBySlug retrieves a block template by slug
func (h *AdminTemplateHandler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	template, err := h.repo.GetBySlug(r.Context(), slug)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, h.toResponse(template))
}

// List lists all block templates
func (h *AdminTemplateHandler) List(w http.ResponseWriter, r *http.Request) {
	templates, err := h.repo.List(r.Context())
	if err != nil {
		HandleError(w, err)
		return
	}

	responses := make([]*BlockTemplateResponse, len(templates))
	for i, template := range templates {
		responses[i] = h.toResponse(template)
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"items": responses,
		"total": len(responses),
	})
}

// UpdateBlockTemplateRequest represents the request body for updating a block template
type UpdateBlockTemplateRequest struct {
	Slug         string          `json:"slug,omitempty"`
	Name         string          `json:"name,omitempty"`
	Description  string          `json:"description,omitempty"`
	ConfigSchema json.RawMessage `json:"config_schema,omitempty"`
	ExecutorType string          `json:"executor_type,omitempty"`
	ExecutorCode string          `json:"executor_code,omitempty"`
}

// Update updates a block template (only non-builtin templates)
func (h *AdminTemplateHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "template_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid template ID", nil)
		return
	}

	var req UpdateBlockTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body", nil)
		return
	}

	// Get existing template
	template, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}

	// Check if builtin
	if template.IsBuiltin {
		Error(w, http.StatusForbidden, "BUILTIN_TEMPLATE", "Cannot modify built-in templates", nil)
		return
	}

	// Update fields
	if req.Slug != "" {
		template.Slug = req.Slug
	}
	if req.Name != "" {
		template.Name = req.Name
	}
	if req.Description != "" {
		template.Description = req.Description
	}
	if req.ExecutorType != "" {
		execType := domain.TemplateExecutorType(req.ExecutorType)
		if execType != domain.TemplateExecutorBuiltin && execType != domain.TemplateExecutorJavaScript {
			Error(w, http.StatusBadRequest, "INVALID_EXECUTOR_TYPE", "Executor type must be 'builtin' or 'javascript'", nil)
			return
		}
		template.ExecutorType = execType
	}
	if req.ExecutorCode != "" {
		template.ExecutorCode = req.ExecutorCode
	}
	if req.ConfigSchema != nil && len(req.ConfigSchema) > 0 {
		template.ConfigSchema = req.ConfigSchema
	}

	template.UpdatedAt = time.Now().UTC()

	if err := h.repo.Update(r.Context(), template); err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, h.toResponse(template))
}

// Delete deletes a block template (only non-builtin templates)
func (h *AdminTemplateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "template_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid template ID", nil)
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
