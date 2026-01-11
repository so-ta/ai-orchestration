package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// BlockHandler handles block definition HTTP requests
type BlockHandler struct {
	blockRepo repository.BlockDefinitionRepository
}

// NewBlockHandler creates a new BlockHandler
func NewBlockHandler(blockRepo repository.BlockDefinitionRepository) *BlockHandler {
	return &BlockHandler{blockRepo: blockRepo}
}

// CreateBlockRequest represents a create block definition request
type CreateBlockRequest struct {
	Slug           string          `json:"slug"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	Category       string          `json:"category"`
	Icon           string          `json:"icon"`
	ConfigSchema   json.RawMessage `json:"config_schema"`
	InputSchema    json.RawMessage `json:"input_schema"`
	OutputSchema   json.RawMessage `json:"output_schema"`
	ExecutorType   string          `json:"executor_type"`
	ExecutorConfig json.RawMessage `json:"executor_config"`
}

// List handles GET /api/v1/blocks
func (h *BlockHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	// Parse query parameters
	categoryStr := r.URL.Query().Get("category")
	enabledOnly := r.URL.Query().Get("enabled") == "true"

	filter := repository.BlockDefinitionFilter{
		EnabledOnly: enabledOnly,
	}

	if categoryStr != "" {
		category := domain.BlockCategory(categoryStr)
		if !category.IsValid() {
			Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid category", nil)
			return
		}
		filter.Category = &category
	}

	blocks, err := h.blockRepo.List(r.Context(), &tenantID, filter)
	if err != nil {
		HandleError(w, err)
		return
	}

	// Group blocks by category for frontend convenience
	type BlockListResponse struct {
		Blocks []*domain.BlockDefinition `json:"blocks"`
	}

	JSONData(w, http.StatusOK, BlockListResponse{Blocks: blocks})
}

// Get handles GET /api/v1/blocks/{slug}
func (h *BlockHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	slug := chi.URLParam(r, "slug")

	block, err := h.blockRepo.GetBySlug(r.Context(), &tenantID, slug)
	if err != nil {
		HandleError(w, err)
		return
	}

	if block == nil {
		Error(w, http.StatusNotFound, "NOT_FOUND", "block not found", nil)
		return
	}

	JSONData(w, http.StatusOK, block)
}

// Create handles POST /api/v1/blocks
func (h *BlockHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	var req CreateBlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	// Validate required fields
	if req.Slug == "" || req.Name == "" || req.Category == "" {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "slug, name, and category are required", nil)
		return
	}

	// Validate category
	category := domain.BlockCategory(req.Category)
	if !category.IsValid() {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid category", nil)
		return
	}

	// Validate executor type
	executorType := domain.ExecutorType(req.ExecutorType)
	if req.ExecutorType != "" && !executorType.IsValid() {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid executor_type", nil)
		return
	}
	if req.ExecutorType == "" {
		executorType = domain.ExecutorTypeHTTP // Default for custom blocks
	}

	// Check for existing block with same slug
	existing, err := h.blockRepo.GetBySlug(r.Context(), &tenantID, req.Slug)
	if err != nil {
		HandleError(w, err)
		return
	}
	if existing != nil {
		Error(w, http.StatusConflict, "CONFLICT", "block with this slug already exists", nil)
		return
	}

	// Create block definition
	block := domain.NewBlockDefinition(&tenantID, req.Slug, req.Name, category)
	block.Description = req.Description
	block.Icon = req.Icon
	block.ExecutorType = executorType

	if req.ConfigSchema != nil {
		block.ConfigSchema = req.ConfigSchema
	}
	if req.InputSchema != nil {
		block.InputSchema = req.InputSchema
	}
	if req.OutputSchema != nil {
		block.OutputSchema = req.OutputSchema
	}
	if req.ExecutorConfig != nil {
		block.ExecutorConfig = req.ExecutorConfig
	}

	if err := h.blockRepo.Create(r.Context(), block); err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusCreated, block)
}

// UpdateBlockRequest represents an update block definition request
type UpdateBlockRequest struct {
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	Icon           string          `json:"icon"`
	ConfigSchema   json.RawMessage `json:"config_schema"`
	InputSchema    json.RawMessage `json:"input_schema"`
	OutputSchema   json.RawMessage `json:"output_schema"`
	ExecutorConfig json.RawMessage `json:"executor_config"`
	Enabled        *bool           `json:"enabled"`
}

// Update handles PUT /api/v1/blocks/{slug}
func (h *BlockHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	slug := chi.URLParam(r, "slug")

	var req UpdateBlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	// Get existing block
	block, err := h.blockRepo.GetBySlug(r.Context(), &tenantID, slug)
	if err != nil {
		HandleError(w, err)
		return
	}
	if block == nil {
		Error(w, http.StatusNotFound, "NOT_FOUND", "block not found", nil)
		return
	}

	// System blocks cannot be modified
	if block.IsSystemBlock() {
		Error(w, http.StatusForbidden, "FORBIDDEN", "system blocks cannot be modified", nil)
		return
	}

	// Update fields
	if req.Name != "" {
		block.Name = req.Name
	}
	if req.Description != "" {
		block.Description = req.Description
	}
	if req.Icon != "" {
		block.Icon = req.Icon
	}
	if req.ConfigSchema != nil {
		block.ConfigSchema = req.ConfigSchema
	}
	if req.InputSchema != nil {
		block.InputSchema = req.InputSchema
	}
	if req.OutputSchema != nil {
		block.OutputSchema = req.OutputSchema
	}
	if req.ExecutorConfig != nil {
		block.ExecutorConfig = req.ExecutorConfig
	}
	if req.Enabled != nil {
		block.Enabled = *req.Enabled
	}

	if err := h.blockRepo.Update(r.Context(), block); err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, block)
}

// Delete handles DELETE /api/v1/blocks/{slug}
func (h *BlockHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	slug := chi.URLParam(r, "slug")

	// Get existing block
	block, err := h.blockRepo.GetBySlug(r.Context(), &tenantID, slug)
	if err != nil {
		HandleError(w, err)
		return
	}
	if block == nil {
		Error(w, http.StatusNotFound, "NOT_FOUND", "block not found", nil)
		return
	}

	// System blocks cannot be deleted
	if block.IsSystemBlock() {
		Error(w, http.StatusForbidden, "FORBIDDEN", "system blocks cannot be deleted", nil)
		return
	}

	if err := h.blockRepo.Delete(r.Context(), block.ID); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
