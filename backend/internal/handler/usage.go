package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/usecase"
)

// UsageHandler handles usage HTTP requests
type UsageHandler struct {
	usageUsecase *usecase.UsageUsecase
}

// NewUsageHandler creates a new UsageHandler
func NewUsageHandler(usageUsecase *usecase.UsageUsecase) *UsageHandler {
	return &UsageHandler{usageUsecase: usageUsecase}
}

// GetSummary handles GET /api/v1/usage/summary
func (h *UsageHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "month"
	}

	summary, err := h.usageUsecase.GetSummary(r.Context(), usecase.GetSummaryInput{
		TenantID: tenantID,
		Period:   period,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, summary)
}

// GetDaily handles GET /api/v1/usage/daily
func (h *UsageHandler) GetDaily(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	// Parse start and end dates
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	var start, end time.Time
	now := time.Now()

	if startStr == "" {
		// Default to start of current month
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	} else {
		var err error
		start, err = time.Parse("2006-01-02", startStr)
		if err != nil {
			Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid start date format, use YYYY-MM-DD", nil)
			return
		}
	}

	if endStr == "" {
		// Default to today
		end = time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	} else {
		var err error
		end, err = time.Parse("2006-01-02", endStr)
		if err != nil {
			Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid end date format, use YYYY-MM-DD", nil)
			return
		}
		end = end.AddDate(0, 0, 1) // Include the end date
	}

	daily, err := h.usageUsecase.GetDaily(r.Context(), usecase.GetDailyInput{
		TenantID: tenantID,
		Start:    start,
		End:      end,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, map[string]interface{}{
		"daily": daily,
		"start": start.Format("2006-01-02"),
		"end":   end.AddDate(0, 0, -1).Format("2006-01-02"),
	})
}

// GetByWorkflow handles GET /api/v1/usage/by-workflow
func (h *UsageHandler) GetByWorkflow(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "month"
	}

	workflows, err := h.usageUsecase.GetByWorkflow(r.Context(), usecase.GetByWorkflowInput{
		TenantID: tenantID,
		Period:   period,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, map[string]interface{}{
		"workflows": workflows,
	})
}

// GetByModel handles GET /api/v1/usage/by-model
func (h *UsageHandler) GetByModel(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "month"
	}

	models, err := h.usageUsecase.GetByModel(r.Context(), usecase.GetByModelInput{
		TenantID: tenantID,
		Period:   period,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, map[string]interface{}{
		"models": models,
	})
}

// GetByRun handles GET /api/v1/runs/{run_id}/usage
func (h *UsageHandler) GetByRun(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	runID, err := uuid.Parse(chi.URLParam(r, "run_id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid run ID", nil)
		return
	}

	records, err := h.usageUsecase.GetByRun(r.Context(), usecase.GetByRunInput{
		TenantID: tenantID,
		RunID:    runID,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	// Calculate totals
	var totalCost float64
	var totalInputTokens, totalOutputTokens int
	for _, record := range records {
		totalCost += record.TotalCostUSD
		totalInputTokens += record.InputTokens
		totalOutputTokens += record.OutputTokens
	}

	JSONData(w, http.StatusOK, map[string]interface{}{
		"records":            records,
		"total_cost_usd":     totalCost,
		"total_input_tokens": totalInputTokens,
		"total_output_tokens": totalOutputTokens,
	})
}

// ListBudgets handles GET /api/v1/usage/budgets
func (h *UsageHandler) ListBudgets(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	budgets, err := h.usageUsecase.ListBudgets(r.Context(), usecase.ListBudgetsInput{
		TenantID: tenantID,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, map[string]interface{}{
		"budgets": budgets,
	})
}

// CreateBudgetRequest represents a create budget request
type CreateBudgetRequest struct {
	WorkflowID      *uuid.UUID `json:"workflow_id,omitempty"`
	BudgetType      string     `json:"budget_type"`
	BudgetAmountUSD float64    `json:"budget_amount_usd"`
	AlertThreshold  float64    `json:"alert_threshold,omitempty"`
}

// CreateBudget handles POST /api/v1/usage/budgets
func (h *UsageHandler) CreateBudget(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	var req CreateBudgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	if req.BudgetType != string(domain.BudgetTypeDaily) && req.BudgetType != string(domain.BudgetTypeMonthly) {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "budget_type must be 'daily' or 'monthly'", nil)
		return
	}

	if req.BudgetAmountUSD <= 0 {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "budget_amount_usd must be positive", nil)
		return
	}

	alertThreshold := req.AlertThreshold
	if alertThreshold <= 0 || alertThreshold > 1 {
		alertThreshold = 0.80
	}

	budget, err := h.usageUsecase.CreateBudget(r.Context(), usecase.CreateBudgetInput{
		TenantID:        tenantID,
		WorkflowID:      req.WorkflowID,
		BudgetType:      domain.BudgetType(req.BudgetType),
		BudgetAmountUSD: req.BudgetAmountUSD,
		AlertThreshold:  alertThreshold,
	})
	if err != nil {
		if err == usecase.ErrBudgetAlreadyExists {
			Error(w, http.StatusConflict, "CONFLICT", err.Error(), nil)
			return
		}
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusCreated, budget)
}

// UpdateBudgetRequest represents an update budget request
type UpdateBudgetRequest struct {
	BudgetAmountUSD *float64 `json:"budget_amount_usd,omitempty"`
	AlertThreshold  *float64 `json:"alert_threshold,omitempty"`
	Enabled         *bool    `json:"enabled,omitempty"`
}

// UpdateBudget handles PUT /api/v1/usage/budgets/{id}
func (h *UsageHandler) UpdateBudget(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	budgetID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid budget ID", nil)
		return
	}

	var req UpdateBudgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	budget, err := h.usageUsecase.UpdateBudget(r.Context(), usecase.UpdateBudgetInput{
		TenantID:        tenantID,
		BudgetID:        budgetID,
		BudgetAmountUSD: req.BudgetAmountUSD,
		AlertThreshold:  req.AlertThreshold,
		Enabled:         req.Enabled,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, budget)
}

// DeleteBudget handles DELETE /api/v1/usage/budgets/{id}
func (h *UsageHandler) DeleteBudget(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	budgetID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid budget ID", nil)
		return
	}

	if err := h.usageUsecase.DeleteBudget(r.Context(), usecase.DeleteBudgetInput{
		TenantID: tenantID,
		BudgetID: budgetID,
	}); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetPricing handles GET /api/v1/usage/pricing
func (h *UsageHandler) GetPricing(w http.ResponseWriter, r *http.Request) {
	output, err := h.usageUsecase.GetPricing(r.Context())
	if err != nil {
		HandleError(w, err)
		return
	}

	JSONData(w, http.StatusOK, output)
}
