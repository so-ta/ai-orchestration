package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/usecase"
)

// TemplateHandler handles template HTTP requests
type TemplateHandler struct {
	templateUsecase *usecase.TemplateUsecase
	auditService    *usecase.AuditService
}

// NewTemplateHandler creates a new TemplateHandler
func NewTemplateHandler(templateUsecase *usecase.TemplateUsecase, auditService *usecase.AuditService) *TemplateHandler {
	return &TemplateHandler{
		templateUsecase: templateUsecase,
		auditService:    auditService,
	}
}

// CreateTemplateRequest represents a create template request
type CreateTemplateRequest struct {
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	Category    string                     `json:"category"`
	Tags        []string                   `json:"tags"`
	Definition  json.RawMessage            `json:"definition"`
	Variables   json.RawMessage            `json:"variables"`
	AuthorName  string                     `json:"author_name"`
	Visibility  domain.TemplateVisibility  `json:"visibility"`
}

// Create handles POST /api/v1/templates
func (h *TemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	var req CreateTemplateRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	template, err := h.templateUsecase.Create(r.Context(), usecase.CreateTemplateInput{
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		Tags:        req.Tags,
		Definition:  req.Definition,
		Variables:   req.Variables,
		AuthorName:  req.AuthorName,
		Visibility:  req.Visibility,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionProjectCreate, "template", &template.ID, map[string]interface{}{
		"name": template.Name,
	})

	JSONData(w, http.StatusCreated, template)
}

// CreateFromProjectRequest represents a create from project request
type CreateFromProjectRequest struct {
	ProjectID   string                     `json:"project_id"`
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	Category    string                     `json:"category"`
	Tags        []string                   `json:"tags"`
	AuthorName  string                     `json:"author_name"`
	Visibility  domain.TemplateVisibility  `json:"visibility"`
}

// CreateFromProject handles POST /api/v1/templates/from-project
func (h *TemplateHandler) CreateFromProject(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	var req CreateFromProjectRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid project_id", nil)
		return
	}

	template, err := h.templateUsecase.CreateFromProject(r.Context(), usecase.CreateFromProjectInput{
		TenantID:    tenantID,
		ProjectID:   projectID,
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		Tags:        req.Tags,
		AuthorName:  req.AuthorName,
		Visibility:  req.Visibility,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusCreated, template)
}

// List handles GET /api/v1/templates
func (h *TemplateHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	// Parse query parameters
	page := parseIntQuery(r, "page", 1)
	limit := parseIntQuery(r, "limit", 20)
	category := r.URL.Query().Get("category")
	search := r.URL.Query().Get("search")
	scope := r.URL.Query().Get("scope") // "my", "tenant", "public"

	input := usecase.ListTemplatesInput{
		Page:  page,
		Limit: limit,
	}

	if category != "" {
		input.Category = &category
	}
	if search != "" {
		input.Search = &search
	}

	switch scope {
	case "my", "tenant":
		input.TenantID = &tenantID
	case "public":
		// No tenant filter for public templates
	default:
		// Default: show both tenant and public templates
		input.TenantID = &tenantID
	}

	output, err := h.templateUsecase.List(r.Context(), input)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONList(w, http.StatusOK, output.Templates, output.Page, output.Limit, output.Total)
}

// ListPublic handles GET /api/v1/templates/marketplace
func (h *TemplateHandler) ListPublic(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	page := parseIntQuery(r, "page", 1)
	limit := parseIntQuery(r, "limit", 20)
	category := r.URL.Query().Get("category")
	search := r.URL.Query().Get("search")
	featured := r.URL.Query().Get("featured")

	input := usecase.ListTemplatesInput{
		Page:  page,
		Limit: limit,
	}

	if category != "" {
		input.Category = &category
	}
	if search != "" {
		input.Search = &search
	}
	if featured == "true" {
		isFeatured := true
		input.IsFeatured = &isFeatured
	}

	output, err := h.templateUsecase.List(r.Context(), input)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONList(w, http.StatusOK, output.Templates, output.Page, output.Limit, output.Total)
}

// Get handles GET /api/v1/templates/{id}
func (h *TemplateHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid template ID", nil)
		return
	}

	template, err := h.templateUsecase.GetByID(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, template)
}

// UpdateTemplateRequest represents an update template request
type UpdateTemplateRequest struct {
	Name        string                      `json:"name"`
	Description string                      `json:"description"`
	Category    string                      `json:"category"`
	Tags        []string                    `json:"tags"`
	Definition  json.RawMessage             `json:"definition"`
	Variables   json.RawMessage             `json:"variables"`
	Visibility  *domain.TemplateVisibility  `json:"visibility"`
}

// Update handles PUT /api/v1/templates/{id}
func (h *TemplateHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid template ID", nil)
		return
	}

	var req UpdateTemplateRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	template, err := h.templateUsecase.Update(r.Context(), usecase.UpdateTemplateInput{
		ID:          id,
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		Tags:        req.Tags,
		Definition:  req.Definition,
		Variables:   req.Variables,
		Visibility:  req.Visibility,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, template)
}

// Delete handles DELETE /api/v1/templates/{id}
func (h *TemplateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid template ID", nil)
		return
	}

	if err := h.templateUsecase.Delete(r.Context(), tenantID, id); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UseTemplateRequest represents a use template request
type UseTemplateRequest struct {
	ProjectName string `json:"project_name"`
}

// Use handles POST /api/v1/templates/{id}/use
func (h *TemplateHandler) Use(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid template ID", nil)
		return
	}

	var req UseTemplateRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	project, err := h.templateUsecase.UseTemplate(r.Context(), tenantID, id, req.ProjectName)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusCreated, project)
}

// AddReviewRequest represents an add review request
type AddReviewRequest struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment"`
}

// AddReview handles POST /api/v1/templates/{id}/reviews
func (h *TemplateHandler) AddReview(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	userID := getUserID(r)
	templateID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid template ID", nil)
		return
	}

	var req AddReviewRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	review, err := h.templateUsecase.AddReview(r.Context(), tenantID, templateID, userID, req.Rating, req.Comment)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusCreated, review)
}

// GetReviews handles GET /api/v1/templates/{id}/reviews
func (h *TemplateHandler) GetReviews(w http.ResponseWriter, r *http.Request) {
	templateID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid template ID", nil)
		return
	}

	reviews, err := h.templateUsecase.GetReviews(r.Context(), templateID)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, reviews)
}

// GetCategories handles GET /api/v1/templates/categories
func (h *TemplateHandler) GetCategories(w http.ResponseWriter, r *http.Request) {
	categories := h.templateUsecase.GetCategories()
	JSONData(w, http.StatusOK, categories)
}
