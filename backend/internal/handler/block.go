package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
	"github.com/souta/ai-orchestration/internal/usecase"
)

// BlockHandler handles block definition HTTP requests
type BlockHandler struct {
	blockRepo    repository.BlockDefinitionRepository
	blockUsecase *usecase.BlockUsecase
}

// NewBlockHandler creates a new BlockHandler
func NewBlockHandler(blockRepo repository.BlockDefinitionRepository, blockUsecase *usecase.BlockUsecase) *BlockHandler {
	return &BlockHandler{
		blockRepo:    blockRepo,
		blockUsecase: blockUsecase,
	}
}

// InternalStepRequest represents an internal step in a composite block
type InternalStepRequest struct {
	Type      string          `json:"type"`       // Block slug to execute
	Config    json.RawMessage `json:"config"`     // Configuration for the step
	OutputKey string          `json:"output_key"` // Key to store this step's output
}

// CreateBlockRequest represents a create block definition request
type CreateBlockRequest struct {
	Slug         string          `json:"slug"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Category     string          `json:"category"`
	Icon         string          `json:"icon"`
	ConfigSchema json.RawMessage `json:"config_schema"`
	OutputSchema json.RawMessage `json:"output_schema"`
	Code         string          `json:"code"`
	UIConfig     json.RawMessage `json:"ui_config"`

	// Block Inheritance/Extension fields
	ParentBlockID  *string               `json:"parent_block_id,omitempty"`
	ConfigDefaults json.RawMessage       `json:"config_defaults,omitempty"`
	PreProcess     string                `json:"pre_process,omitempty"`
	PostProcess    string                `json:"post_process,omitempty"`
	InternalSteps  []InternalStepRequest `json:"internal_steps,omitempty"`
}

// List handles GET /api/v1/blocks
// Supports query parameters:
// - category: filter by category (e.g., "ai", "integration")
// - enabled: filter by enabled status ("true" for enabled only)
// - search: search by name or description
func (h *BlockHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	// Parse query parameters
	categoryStr := r.URL.Query().Get("category")
	enabledOnly := r.URL.Query().Get("enabled") == "true"
	searchStr := r.URL.Query().Get("search")

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

	if searchStr != "" {
		filter.Search = &searchStr
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
	block.Code = req.Code

	if req.ConfigSchema != nil {
		block.ConfigSchema = req.ConfigSchema
	}
	if req.OutputSchema != nil {
		block.OutputSchema = req.OutputSchema
	}
	if req.UIConfig != nil {
		block.UIConfig = req.UIConfig
	}

	// Handle inheritance fields
	if req.ParentBlockID != nil && *req.ParentBlockID != "" {
		parentID, err := uuid.Parse(*req.ParentBlockID)
		if err != nil {
			Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid parent_block_id format", nil)
			return
		}

		// Validate parent block exists and can be inherited
		parentBlock, err := h.blockRepo.GetByID(r.Context(), parentID)
		if err != nil {
			HandleError(w, err)
			return
		}
		if parentBlock == nil {
			Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "parent block not found", nil)
			return
		}
		if !parentBlock.CanBeInherited() {
			Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "parent block cannot be inherited (no code)", nil)
			return
		}

		block.ParentBlockID = &parentID

		// Validate inheritance (circular reference and depth)
		if err := h.blockRepo.ValidateInheritance(r.Context(), block.ID, parentID); err != nil {
			switch err {
			case domain.ErrCircularInheritance:
				Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "circular inheritance detected", nil)
				return
			case domain.ErrInheritanceDepthExceeded:
				Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "inheritance depth exceeded maximum limit", nil)
				return
			default:
				HandleError(w, err)
				return
			}
		}
	}

	if req.ConfigDefaults != nil {
		block.ConfigDefaults = req.ConfigDefaults
	}
	block.PreProcess = req.PreProcess
	block.PostProcess = req.PostProcess

	// Convert internal steps
	if len(req.InternalSteps) > 0 {
		block.InternalSteps = make([]domain.InternalStep, len(req.InternalSteps))
		for i, step := range req.InternalSteps {
			// Validate each internal step's block type exists
			stepBlock, err := h.blockRepo.GetBySlug(r.Context(), &tenantID, step.Type)
			if err != nil {
				HandleError(w, err)
				return
			}
			if stepBlock == nil {
				Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "internal step block type not found: "+step.Type, nil)
				return
			}

			block.InternalSteps[i] = domain.InternalStep{
				Type:      step.Type,
				Config:    step.Config,
				OutputKey: step.OutputKey,
			}
		}
	}

	if err := h.blockRepo.Create(r.Context(), block); err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusCreated, block)
}

// UpdateBlockRequest represents an update block definition request
type UpdateBlockRequest struct {
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Icon         string          `json:"icon"`
	ConfigSchema json.RawMessage `json:"config_schema"`
	OutputSchema json.RawMessage `json:"output_schema"`
	Code         *string         `json:"code"`
	UIConfig     json.RawMessage `json:"ui_config"`
	Enabled      *bool           `json:"enabled"`

	// Block Inheritance/Extension fields
	ParentBlockID  *string               `json:"parent_block_id,omitempty"`
	ConfigDefaults json.RawMessage       `json:"config_defaults,omitempty"`
	PreProcess     *string               `json:"pre_process,omitempty"`
	PostProcess    *string               `json:"post_process,omitempty"`
	InternalSteps  []InternalStepRequest `json:"internal_steps,omitempty"`
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
	if req.OutputSchema != nil {
		block.OutputSchema = req.OutputSchema
	}
	if req.Code != nil {
		block.Code = *req.Code
	}
	if req.UIConfig != nil {
		block.UIConfig = req.UIConfig
	}
	if req.Enabled != nil {
		block.Enabled = *req.Enabled
	}

	// Handle inheritance fields
	if req.ParentBlockID != nil {
		if *req.ParentBlockID == "" {
			// Clear parent
			block.ParentBlockID = nil
		} else {
			parentID, err := uuid.Parse(*req.ParentBlockID)
			if err != nil {
				Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid parent_block_id format", nil)
				return
			}

			// Validate parent block exists and can be inherited
			parentBlock, err := h.blockRepo.GetByID(r.Context(), parentID)
			if err != nil {
				HandleError(w, err)
				return
			}
			if parentBlock == nil {
				Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "parent block not found", nil)
				return
			}
			if !parentBlock.CanBeInherited() {
				Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "parent block cannot be inherited (no code)", nil)
				return
			}

			// Validate inheritance (circular reference and depth)
			if err := h.blockRepo.ValidateInheritance(r.Context(), block.ID, parentID); err != nil {
				switch err {
				case domain.ErrCircularInheritance:
					Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "circular inheritance detected", nil)
					return
				case domain.ErrInheritanceDepthExceeded:
					Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "inheritance depth exceeded maximum limit", nil)
					return
				default:
					HandleError(w, err)
					return
				}
			}

			block.ParentBlockID = &parentID
		}
	}

	if req.ConfigDefaults != nil {
		block.ConfigDefaults = req.ConfigDefaults
	}
	if req.PreProcess != nil {
		block.PreProcess = *req.PreProcess
	}
	if req.PostProcess != nil {
		block.PostProcess = *req.PostProcess
	}

	// Update internal steps if provided
	if req.InternalSteps != nil {
		block.InternalSteps = make([]domain.InternalStep, len(req.InternalSteps))
		for i, step := range req.InternalSteps {
			// Validate each internal step's block type exists
			stepBlock, err := h.blockRepo.GetBySlug(r.Context(), &tenantID, step.Type)
			if err != nil {
				HandleError(w, err)
				return
			}
			if stepBlock == nil {
				Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "internal step block type not found: "+step.Type, nil)
				return
			}

			block.InternalSteps[i] = domain.InternalStep{
				Type:      step.Type,
				Config:    step.Config,
				OutputKey: step.OutputKey,
			}
		}
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

// ============================================================================
// Admin endpoints for system block management
// ============================================================================

// ListSystemBlocks handles GET /api/v1/admin/blocks
func (h *BlockHandler) ListSystemBlocks(w http.ResponseWriter, r *http.Request) {
	blocks, err := h.blockUsecase.ListSystemBlocks(r.Context())
	if err != nil {
		HandleError(w, err)
		return
	}

	type SystemBlockListResponse struct {
		Blocks []*domain.BlockDefinition `json:"blocks"`
	}

	JSONData(w, http.StatusOK, SystemBlockListResponse{Blocks: blocks})
}

// UpdateSystemBlockRequest represents a request to update a system block
type UpdateSystemBlockRequest struct {
	Name          *string         `json:"name"`
	Description   *string         `json:"description"`
	Code          *string         `json:"code"`
	ConfigSchema  json.RawMessage `json:"config_schema"`
	OutputSchema  json.RawMessage `json:"output_schema"`
	UIConfig      json.RawMessage `json:"ui_config"`
	ChangeSummary string          `json:"change_summary"`
}

// UpdateSystemBlock handles PUT /api/v1/admin/blocks/{id}
func (h *BlockHandler) UpdateSystemBlock(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid block id", nil)
		return
	}

	var req UpdateSystemBlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	// Get user ID from context (if available)
	var changedBy *uuid.UUID
	if userID := getUserID(r); userID != uuid.Nil {
		changedBy = &userID
	}

	input := usecase.UpdateSystemBlockInput{
		BlockID:       id,
		Name:          req.Name,
		Description:   req.Description,
		Code:          req.Code,
		ConfigSchema:  req.ConfigSchema,
		OutputSchema:  req.OutputSchema,
		UIConfig:      req.UIConfig,
		ChangeSummary: req.ChangeSummary,
		ChangedBy:     changedBy,
	}

	block, err := h.blockUsecase.UpdateSystemBlock(r.Context(), input)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, block)
}

// GetSystemBlock handles GET /api/v1/admin/blocks/{id}
func (h *BlockHandler) GetSystemBlock(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid block id", nil)
		return
	}

	block, err := h.blockRepo.GetByID(r.Context(), id)
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

// ListBlockVersions handles GET /api/v1/admin/blocks/{id}/versions
func (h *BlockHandler) ListBlockVersions(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid block id", nil)
		return
	}

	versions, err := h.blockUsecase.GetBlockVersions(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}

	type VersionListResponse struct {
		Versions []*domain.BlockVersion `json:"versions"`
	}

	JSONData(w, http.StatusOK, VersionListResponse{Versions: versions})
}

// GetBlockVersion handles GET /api/v1/admin/blocks/{id}/versions/{version}
func (h *BlockHandler) GetBlockVersion(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid block id", nil)
		return
	}

	versionStr := chi.URLParam(r, "version")
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid version number", nil)
		return
	}

	blockVersion, err := h.blockUsecase.GetBlockVersion(r.Context(), id, version)
	if err != nil {
		HandleError(w, err)
		return
	}
	if blockVersion == nil {
		Error(w, http.StatusNotFound, "NOT_FOUND", "version not found", nil)
		return
	}

	JSONData(w, http.StatusOK, blockVersion)
}

// RollbackBlockRequest represents a request to rollback a block
type RollbackBlockRequest struct {
	Version int `json:"version"`
}

// RollbackBlock handles POST /api/v1/admin/blocks/{id}/rollback
func (h *BlockHandler) RollbackBlock(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid block id", nil)
		return
	}

	var req RollbackBlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	if req.Version <= 0 {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "version must be positive", nil)
		return
	}

	// Get user ID from context (if available)
	var changedBy *uuid.UUID
	if userID := getUserID(r); userID != uuid.Nil {
		changedBy = &userID
	}

	input := usecase.RollbackSystemBlockInput{
		BlockID:   id,
		Version:   req.Version,
		ChangedBy: changedBy,
	}

	block, err := h.blockUsecase.RollbackSystemBlock(r.Context(), input)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, block)
}
