package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/usecase"
)

// BlockPackageHandler handles block package HTTP requests
type BlockPackageHandler struct {
	packageUsecase *usecase.BlockPackageUsecase
	auditService   *usecase.AuditService
}

// NewBlockPackageHandler creates a new BlockPackageHandler
func NewBlockPackageHandler(packageUsecase *usecase.BlockPackageUsecase, auditService *usecase.AuditService) *BlockPackageHandler {
	return &BlockPackageHandler{
		packageUsecase: packageUsecase,
		auditService:   auditService,
	}
}

// CreatePackageRequest represents a create package request
type CreatePackageRequest struct {
	Name         string                         `json:"name"`
	Version      string                         `json:"version"`
	Description  string                         `json:"description"`
	Blocks       []domain.PackageBlockDefinition `json:"blocks"`
	Dependencies []domain.PackageDependency     `json:"dependencies"`
}

// Create handles POST /api/v1/block-packages
func (h *BlockPackageHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	userID := getUserID(r)

	var req CreatePackageRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	pkg, err := h.packageUsecase.Create(r.Context(), usecase.CreatePackageInput{
		TenantID:     tenantID,
		Name:         req.Name,
		Version:      req.Version,
		Description:  req.Description,
		Blocks:       req.Blocks,
		Dependencies: req.Dependencies,
		CreatedBy:    &userID,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusCreated, pkg)
}

// Get handles GET /api/v1/block-packages/{id}
func (h *BlockPackageHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid package ID", nil)
		return
	}

	pkg, err := h.packageUsecase.GetByID(r.Context(), tenantID, id)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, pkg)
}

// List handles GET /api/v1/block-packages
func (h *BlockPackageHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	// Parse query parameters
	page := parseIntQuery(r, "page", 1)
	limit := parseIntQuery(r, "limit", 20)
	statusStr := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")

	input := usecase.ListPackagesInput{
		TenantID: tenantID,
		Page:     page,
		Limit:    limit,
	}

	if statusStr != "" {
		status := domain.BlockPackageStatus(statusStr)
		input.Status = &status
	}
	if search != "" {
		input.Search = &search
	}

	output, err := h.packageUsecase.List(r.Context(), input)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONList(w, http.StatusOK, output.Packages, output.Page, output.Limit, output.Total)
}

// UpdatePackageRequest represents an update package request
type UpdatePackageRequest struct {
	Description  *string                         `json:"description,omitempty"`
	Blocks       []domain.PackageBlockDefinition `json:"blocks,omitempty"`
	Dependencies []domain.PackageDependency      `json:"dependencies,omitempty"`
	BundleURL    *string                         `json:"bundle_url,omitempty"`
}

// Update handles PUT /api/v1/block-packages/{id}
func (h *BlockPackageHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid package ID", nil)
		return
	}

	var req UpdatePackageRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	pkg, err := h.packageUsecase.Update(r.Context(), usecase.UpdatePackageInput{
		ID:           id,
		TenantID:     tenantID,
		Description:  req.Description,
		Blocks:       req.Blocks,
		Dependencies: req.Dependencies,
		BundleURL:    req.BundleURL,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, pkg)
}

// Delete handles DELETE /api/v1/block-packages/{id}
func (h *BlockPackageHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid package ID", nil)
		return
	}

	if err := h.packageUsecase.Delete(r.Context(), tenantID, id); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Publish handles POST /api/v1/block-packages/{id}/publish
func (h *BlockPackageHandler) Publish(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid package ID", nil)
		return
	}

	pkg, err := h.packageUsecase.Publish(r.Context(), tenantID, id)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, pkg)
}

// Deprecate handles POST /api/v1/block-packages/{id}/deprecate
func (h *BlockPackageHandler) Deprecate(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid package ID", nil)
		return
	}

	pkg, err := h.packageUsecase.Deprecate(r.Context(), tenantID, id)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, pkg)
}
