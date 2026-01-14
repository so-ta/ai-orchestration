package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// AdminTenantHandler handles HTTP requests for tenant management (operator-only)
type AdminTenantHandler struct {
	repo repository.TenantRepository
}

// NewAdminTenantHandler creates a new AdminTenantHandler
func NewAdminTenantHandler(repo repository.TenantRepository) *AdminTenantHandler {
	return &AdminTenantHandler{repo: repo}
}

// CreateTenantRequest represents the request body for creating a tenant
type CreateTenantRequest struct {
	Name         string                    `json:"name"`
	Slug         string                    `json:"slug"`
	Plan         string                    `json:"plan,omitempty"`
	OwnerEmail   string                    `json:"owner_email,omitempty"`
	OwnerName    string                    `json:"owner_name,omitempty"`
	BillingEmail string                    `json:"billing_email,omitempty"`
	Metadata     *domain.TenantMetadata    `json:"metadata,omitempty"`
	FeatureFlags *domain.TenantFeatureFlags `json:"feature_flags,omitempty"`
	Limits       *domain.TenantLimits      `json:"limits,omitempty"`
}

// UpdateTenantRequest represents the request body for updating a tenant
type UpdateTenantRequest struct {
	Name         string                    `json:"name,omitempty"`
	Slug         string                    `json:"slug,omitempty"`
	Plan         string                    `json:"plan,omitempty"`
	OwnerEmail   string                    `json:"owner_email,omitempty"`
	OwnerName    string                    `json:"owner_name,omitempty"`
	BillingEmail string                    `json:"billing_email,omitempty"`
	Metadata     *domain.TenantMetadata    `json:"metadata,omitempty"`
	FeatureFlags *domain.TenantFeatureFlags `json:"feature_flags,omitempty"`
	Limits       *domain.TenantLimits      `json:"limits,omitempty"`
}

// SuspendTenantRequest represents the request body for suspending a tenant
type SuspendTenantRequest struct {
	Reason string `json:"reason"`
}

// TenantResponse represents the response for a tenant
type TenantResponse struct {
	ID              uuid.UUID                 `json:"id"`
	Name            string                    `json:"name"`
	Slug            string                    `json:"slug"`
	Status          string                    `json:"status"`
	Plan            string                    `json:"plan"`
	OwnerEmail      string                    `json:"owner_email,omitempty"`
	OwnerName       string                    `json:"owner_name,omitempty"`
	BillingEmail    string                    `json:"billing_email,omitempty"`
	Settings        json.RawMessage           `json:"settings"`
	Metadata        json.RawMessage           `json:"metadata"`
	FeatureFlags    json.RawMessage           `json:"feature_flags"`
	Limits          json.RawMessage           `json:"limits"`
	SuspendedAt     *time.Time                `json:"suspended_at,omitempty"`
	SuspendedReason string                    `json:"suspended_reason,omitempty"`
	CreatedAt       time.Time                 `json:"created_at"`
	UpdatedAt       time.Time                 `json:"updated_at"`
	Stats           *domain.TenantStats       `json:"stats,omitempty"`
}

// toResponse converts a Tenant to a response
func (h *AdminTenantHandler) toResponse(tenant *domain.Tenant, stats *domain.TenantStats) *TenantResponse {
	return &TenantResponse{
		ID:              tenant.ID,
		Name:            tenant.Name,
		Slug:            tenant.Slug,
		Status:          string(tenant.Status),
		Plan:            string(tenant.Plan),
		OwnerEmail:      tenant.OwnerEmail,
		OwnerName:       tenant.OwnerName,
		BillingEmail:    tenant.BillingEmail,
		Settings:        tenant.Settings,
		Metadata:        tenant.Metadata,
		FeatureFlags:    tenant.FeatureFlags,
		Limits:          tenant.Limits,
		SuspendedAt:     tenant.SuspendedAt,
		SuspendedReason: tenant.SuspendedReason,
		CreatedAt:       tenant.CreatedAt,
		UpdatedAt:       tenant.UpdatedAt,
		Stats:           stats,
	}
}

// List lists all tenants with optional filtering
func (h *AdminTenantHandler) List(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	filter := repository.TenantFilter{
		Page:  1,
		Limit: 20,
	}

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			filter.Page = page
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			filter.Limit = limit
		}
	}

	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		status := domain.TenantStatus(statusStr)
		if status.IsValid() {
			filter.Status = &status
		}
	}

	if planStr := r.URL.Query().Get("plan"); planStr != "" {
		plan := domain.TenantPlan(planStr)
		if plan.IsValid() {
			filter.Plan = &plan
		}
	}

	if search := r.URL.Query().Get("search"); search != "" {
		filter.Search = search
	}

	if r.URL.Query().Get("include_deleted") == "true" {
		filter.IncludeDeleted = true
	}

	// Get tenants
	tenants, total, err := h.repo.List(r.Context(), filter)
	if err != nil {
		HandleError(w, err)
		return
	}

	// Get stats for all tenants
	allStats, err := h.repo.GetAllStats(r.Context())
	if err != nil {
		// Log error but don't fail - stats are optional
		allStats = make(map[uuid.UUID]*domain.TenantStats)
	}

	// Convert to responses
	responses := make([]*TenantResponse, len(tenants))
	for i, tenant := range tenants {
		stats := allStats[tenant.ID]
		responses[i] = h.toResponse(tenant, stats)
	}

	JSONList(w, http.StatusOK, responses, filter.Page, filter.Limit, total)
}

// Create creates a new tenant
func (h *AdminTenantHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body", nil)
		return
	}

	// Validate required fields
	if req.Name == "" {
		Error(w, http.StatusBadRequest, "MISSING_NAME", "Name is required", nil)
		return
	}
	if req.Slug == "" {
		Error(w, http.StatusBadRequest, "MISSING_SLUG", "Slug is required", nil)
		return
	}

	// Validate plan
	plan := domain.TenantPlan(req.Plan)
	if req.Plan != "" && !plan.IsValid() {
		Error(w, http.StatusBadRequest, "INVALID_PLAN", "Invalid plan. Must be one of: free, starter, professional, enterprise", nil)
		return
	}
	if req.Plan == "" {
		plan = domain.TenantPlanFree
	}

	// Check if slug already exists
	existing, err := h.repo.GetBySlug(r.Context(), req.Slug)
	if err != nil && err.Error() != "tenant not found" {
		// Log unexpected DB errors (not "not found" errors)
		slog.Error("failed to check tenant slug existence",
			"slug", req.Slug,
			"error", err,
		)
	}
	if err == nil && existing != nil {
		Error(w, http.StatusConflict, "SLUG_EXISTS", "A tenant with this slug already exists", nil)
		return
	}

	// Create tenant
	tenant := domain.NewTenant(req.Name, req.Slug, plan)
	tenant.OwnerEmail = req.OwnerEmail
	tenant.OwnerName = req.OwnerName
	tenant.BillingEmail = req.BillingEmail

	// Apply custom feature flags if provided
	if req.FeatureFlags != nil {
		flagsJSON, _ := json.Marshal(req.FeatureFlags)
		tenant.FeatureFlags = flagsJSON
	}

	// Apply custom limits if provided
	if req.Limits != nil {
		limitsJSON, _ := json.Marshal(req.Limits)
		tenant.Limits = limitsJSON
	}

	// Apply custom metadata if provided
	if req.Metadata != nil {
		metadataJSON, _ := json.Marshal(req.Metadata)
		tenant.Metadata = metadataJSON
	}

	if err := h.repo.Create(r.Context(), tenant); err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusCreated, h.toResponse(tenant, nil))
}

// Get retrieves a tenant by ID
func (h *AdminTenantHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "tenant_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid tenant ID", nil)
		return
	}

	tenant, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}

	// Get stats (log errors but don't fail the request)
	stats, err := h.repo.GetStats(r.Context(), id)
	if err != nil {
		slog.Error("failed to get tenant stats",
			"tenant_id", id.String(),
			"error", err,
		)
	}

	JSON(w, http.StatusOK, h.toResponse(tenant, stats))
}

// Update updates a tenant
func (h *AdminTenantHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "tenant_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid tenant ID", nil)
		return
	}

	var req UpdateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body", nil)
		return
	}

	// Get existing tenant
	tenant, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}

	// Update fields
	if req.Name != "" {
		tenant.Name = req.Name
	}
	if req.Slug != "" {
		// Check if new slug already exists
		existing, err := h.repo.GetBySlug(r.Context(), req.Slug)
		if err != nil && err.Error() != "tenant not found" {
			// Log unexpected DB errors (not "not found" errors)
			slog.Error("failed to check tenant slug existence",
				"slug", req.Slug,
				"error", err,
			)
		}
		if err == nil && existing != nil && existing.ID != tenant.ID {
			Error(w, http.StatusConflict, "SLUG_EXISTS", "A tenant with this slug already exists", nil)
			return
		}
		tenant.Slug = req.Slug
	}
	if req.Plan != "" {
		plan := domain.TenantPlan(req.Plan)
		if !plan.IsValid() {
			Error(w, http.StatusBadRequest, "INVALID_PLAN", "Invalid plan. Must be one of: free, starter, professional, enterprise", nil)
			return
		}
		tenant.Plan = plan
	}
	if req.OwnerEmail != "" {
		tenant.OwnerEmail = req.OwnerEmail
	}
	if req.OwnerName != "" {
		tenant.OwnerName = req.OwnerName
	}
	if req.BillingEmail != "" {
		tenant.BillingEmail = req.BillingEmail
	}
	if req.FeatureFlags != nil {
		flagsJSON, _ := json.Marshal(req.FeatureFlags)
		tenant.FeatureFlags = flagsJSON
	}
	if req.Limits != nil {
		limitsJSON, _ := json.Marshal(req.Limits)
		tenant.Limits = limitsJSON
	}
	if req.Metadata != nil {
		metadataJSON, _ := json.Marshal(req.Metadata)
		tenant.Metadata = metadataJSON
	}

	if err := h.repo.Update(r.Context(), tenant); err != nil {
		HandleError(w, err)
		return
	}

	// Get updated stats (log errors but don't fail the request)
	stats, err := h.repo.GetStats(r.Context(), id)
	if err != nil {
		slog.Error("failed to get tenant stats",
			"tenant_id", id.String(),
			"error", err,
		)
	}

	JSON(w, http.StatusOK, h.toResponse(tenant, stats))
}

// Delete soft-deletes a tenant
func (h *AdminTenantHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "tenant_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid tenant ID", nil)
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Suspend suspends a tenant
func (h *AdminTenantHandler) Suspend(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "tenant_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid tenant ID", nil)
		return
	}

	var req SuspendTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body", nil)
		return
	}

	if req.Reason == "" {
		Error(w, http.StatusBadRequest, "MISSING_REASON", "Suspension reason is required", nil)
		return
	}

	if err := h.repo.UpdateStatus(r.Context(), id, domain.TenantStatusSuspended, req.Reason); err != nil {
		HandleError(w, err)
		return
	}

	// Get updated tenant
	tenant, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, h.toResponse(tenant, nil))
}

// Activate activates a tenant
func (h *AdminTenantHandler) Activate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "tenant_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid tenant ID", nil)
		return
	}

	if err := h.repo.UpdateStatus(r.Context(), id, domain.TenantStatusActive, ""); err != nil {
		HandleError(w, err)
		return
	}

	// Get updated tenant
	tenant, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, h.toResponse(tenant, nil))
}

// GetStats retrieves statistics for a tenant
func (h *AdminTenantHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "tenant_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid tenant ID", nil)
		return
	}

	// Verify tenant exists
	_, err = h.repo.GetByID(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}

	stats, err := h.repo.GetStats(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, stats)
}

// GetOverviewStats retrieves overview statistics for all tenants
func (h *AdminTenantHandler) GetOverviewStats(w http.ResponseWriter, r *http.Request) {
	// Get tenant counts by status
	allTenants, total, err := h.repo.List(r.Context(), repository.TenantFilter{
		Limit: 10000, // Get all
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	// Count by status and plan
	statusCounts := map[string]int{
		"active":    0,
		"suspended": 0,
		"pending":   0,
		"inactive":  0,
	}
	planCounts := map[string]int{
		"free":         0,
		"starter":      0,
		"professional": 0,
		"enterprise":   0,
	}

	for _, tenant := range allTenants {
		statusCounts[string(tenant.Status)]++
		planCounts[string(tenant.Plan)]++
	}

	// Get aggregated stats
	allStats, err := h.repo.GetAllStats(r.Context())
	if err != nil {
		allStats = make(map[uuid.UUID]*domain.TenantStats)
	}

	// Calculate totals
	var totalWorkflows, totalRuns, totalRunsThisMonth int
	var totalCost, costThisMonth float64

	for _, stats := range allStats {
		totalWorkflows += stats.WorkflowCount
		totalRuns += stats.RunCount
		totalRunsThisMonth += stats.RunsThisMonth
		totalCost += stats.TotalCostUSD
		costThisMonth += stats.CostThisMonth
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"total_tenants":       total,
		"status_counts":       statusCounts,
		"plan_counts":         planCounts,
		"total_workflows":     totalWorkflows,
		"total_runs":          totalRuns,
		"total_runs_this_month": totalRunsThisMonth,
		"total_cost_usd":      totalCost,
		"cost_this_month":     costThisMonth,
	})
}
