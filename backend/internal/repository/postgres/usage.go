package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/souta/ai-orchestration/internal/domain"
)

// UsageRepository implements repository.UsageRepository
type UsageRepository struct {
	pool *pgxpool.Pool
}

// NewUsageRepository creates a new UsageRepository
func NewUsageRepository(pool *pgxpool.Pool) *UsageRepository {
	return &UsageRepository{pool: pool}
}

// Create creates a new usage record
func (r *UsageRepository) Create(ctx context.Context, record *domain.UsageRecord) error {
	query := `
		INSERT INTO usage_records (
			id, tenant_id, project_id, run_id, step_run_id,
			provider, model, operation,
			input_tokens, output_tokens, total_tokens,
			input_cost_usd, output_cost_usd, total_cost_usd,
			latency_ms, success, error_message, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`
	_, err := r.pool.Exec(ctx, query,
		record.ID, record.TenantID, record.ProjectID, record.RunID, record.StepRunID,
		record.Provider, record.Model, record.Operation,
		record.InputTokens, record.OutputTokens, record.TotalTokens,
		record.InputCostUSD, record.OutputCostUSD, record.TotalCostUSD,
		record.LatencyMs, record.Success, record.ErrorMessage, record.CreatedAt,
	)
	return err
}

// GetByID retrieves a usage record by ID
func (r *UsageRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.UsageRecord, error) {
	query := `
		SELECT id, tenant_id, project_id, run_id, step_run_id,
		       provider, model, operation,
		       input_tokens, output_tokens, total_tokens,
		       input_cost_usd, output_cost_usd, total_cost_usd,
		       latency_ms, success, error_message, created_at
		FROM usage_records
		WHERE id = $1
	`
	var record domain.UsageRecord
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&record.ID, &record.TenantID, &record.ProjectID, &record.RunID, &record.StepRunID,
		&record.Provider, &record.Model, &record.Operation,
		&record.InputTokens, &record.OutputTokens, &record.TotalTokens,
		&record.InputCostUSD, &record.OutputCostUSD, &record.TotalCostUSD,
		&record.LatencyMs, &record.Success, &record.ErrorMessage, &record.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("usage record not found: %s", id)
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetSummary retrieves aggregated usage summary for a tenant
func (r *UsageRepository) GetSummary(ctx context.Context, tenantID uuid.UUID, period string) (*domain.UsageSummary, error) {
	start, end := getPeriodRange(period)

	// Main aggregation query
	mainQuery := `
		SELECT
			COALESCE(SUM(total_cost_usd), 0) as total_cost,
			COUNT(*) as total_requests,
			COALESCE(SUM(input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(output_tokens), 0) as total_output_tokens
		FROM usage_records
		WHERE tenant_id = $1 AND created_at >= $2 AND created_at < $3
	`

	var summary domain.UsageSummary
	summary.Period = period
	summary.ByProvider = make(map[string]domain.ProviderUsage)
	summary.ByModel = make(map[string]domain.ModelUsage)

	err := r.pool.QueryRow(ctx, mainQuery, tenantID, start, end).Scan(
		&summary.TotalCostUSD,
		&summary.TotalRequests,
		&summary.TotalInputTokens,
		&summary.TotalOutputTokens,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	// Provider breakdown
	providerQuery := `
		SELECT provider, COALESCE(SUM(total_cost_usd), 0), COUNT(*)
		FROM usage_records
		WHERE tenant_id = $1 AND created_at >= $2 AND created_at < $3
		GROUP BY provider
	`
	providerRows, err := r.pool.Query(ctx, providerQuery, tenantID, start, end)
	if err != nil {
		return nil, err
	}
	defer providerRows.Close()

	for providerRows.Next() {
		var provider string
		var usage domain.ProviderUsage
		if err := providerRows.Scan(&provider, &usage.CostUSD, &usage.Requests); err != nil {
			return nil, err
		}
		summary.ByProvider[provider] = usage
	}

	// Model breakdown
	modelQuery := `
		SELECT provider, model,
		       COALESCE(SUM(total_cost_usd), 0), COUNT(*),
		       COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0)
		FROM usage_records
		WHERE tenant_id = $1 AND created_at >= $2 AND created_at < $3
		GROUP BY provider, model
	`
	modelRows, err := r.pool.Query(ctx, modelQuery, tenantID, start, end)
	if err != nil {
		return nil, err
	}
	defer modelRows.Close()

	for modelRows.Next() {
		var usage domain.ModelUsage
		var model string
		if err := modelRows.Scan(&usage.Provider, &model, &usage.CostUSD, &usage.Requests, &usage.InputTokens, &usage.OutputTokens); err != nil {
			return nil, err
		}
		summary.ByModel[model] = usage
	}

	return &summary, nil
}

// GetDaily retrieves daily usage data for a date range
func (r *UsageRepository) GetDaily(ctx context.Context, tenantID uuid.UUID, start, end time.Time) ([]domain.DailyUsage, error) {
	query := `
		SELECT DATE(created_at) as date,
		       COALESCE(SUM(total_cost_usd), 0) as total_cost,
		       COUNT(*) as total_requests,
		       COALESCE(SUM(total_tokens), 0) as total_tokens
		FROM usage_records
		WHERE tenant_id = $1 AND created_at >= $2 AND created_at < $3
		GROUP BY DATE(created_at)
		ORDER BY date
	`

	rows, err := r.pool.Query(ctx, query, tenantID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []domain.DailyUsage
	for rows.Next() {
		var usage domain.DailyUsage
		if err := rows.Scan(&usage.Date, &usage.TotalCostUSD, &usage.TotalRequests, &usage.TotalTokens); err != nil {
			return nil, err
		}
		results = append(results, usage)
	}

	return results, nil
}

// GetByProject retrieves usage data grouped by project
func (r *UsageRepository) GetByProject(ctx context.Context, tenantID uuid.UUID, period string) ([]domain.ProjectUsage, error) {
	start, end := getPeriodRange(period)

	query := `
		SELECT u.project_id, COALESCE(p.name, 'Unknown') as project_name,
		       COALESCE(SUM(u.total_cost_usd), 0) as total_cost,
		       COUNT(*) as total_requests,
		       COALESCE(SUM(u.total_tokens), 0) as total_tokens
		FROM usage_records u
		LEFT JOIN projects p ON u.project_id = p.id
		WHERE u.tenant_id = $1 AND u.created_at >= $2 AND u.created_at < $3 AND u.project_id IS NOT NULL
		GROUP BY u.project_id, p.name
		ORDER BY total_cost DESC
	`

	rows, err := r.pool.Query(ctx, query, tenantID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []domain.ProjectUsage
	for rows.Next() {
		var usage domain.ProjectUsage
		if err := rows.Scan(&usage.ProjectID, &usage.ProjectName, &usage.TotalCostUSD, &usage.TotalRequests, &usage.TotalTokens); err != nil {
			return nil, err
		}
		results = append(results, usage)
	}

	return results, nil
}

// GetByModel retrieves usage data grouped by model
func (r *UsageRepository) GetByModel(ctx context.Context, tenantID uuid.UUID, period string) (map[string]domain.ModelUsage, error) {
	start, end := getPeriodRange(period)

	query := `
		SELECT provider, model,
		       COALESCE(SUM(total_cost_usd), 0) as total_cost,
		       COUNT(*) as total_requests,
		       COALESCE(SUM(input_tokens), 0) as input_tokens,
		       COALESCE(SUM(output_tokens), 0) as output_tokens
		FROM usage_records
		WHERE tenant_id = $1 AND created_at >= $2 AND created_at < $3
		GROUP BY provider, model
		ORDER BY total_cost DESC
	`

	rows, err := r.pool.Query(ctx, query, tenantID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make(map[string]domain.ModelUsage)
	for rows.Next() {
		var usage domain.ModelUsage
		var model string
		if err := rows.Scan(&usage.Provider, &model, &usage.CostUSD, &usage.Requests, &usage.InputTokens, &usage.OutputTokens); err != nil {
			return nil, err
		}
		results[model] = usage
	}

	return results, nil
}

// GetByRun retrieves all usage records for a specific run
func (r *UsageRepository) GetByRun(ctx context.Context, tenantID, runID uuid.UUID) ([]domain.UsageRecord, error) {
	query := `
		SELECT id, tenant_id, project_id, run_id, step_run_id,
		       provider, model, operation,
		       input_tokens, output_tokens, total_tokens,
		       input_cost_usd, output_cost_usd, total_cost_usd,
		       latency_ms, success, error_message, created_at
		FROM usage_records
		WHERE tenant_id = $1 AND run_id = $2
		ORDER BY created_at
	`

	rows, err := r.pool.Query(ctx, query, tenantID, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []domain.UsageRecord
	for rows.Next() {
		var record domain.UsageRecord
		if err := rows.Scan(
			&record.ID, &record.TenantID, &record.ProjectID, &record.RunID, &record.StepRunID,
			&record.Provider, &record.Model, &record.Operation,
			&record.InputTokens, &record.OutputTokens, &record.TotalTokens,
			&record.InputCostUSD, &record.OutputCostUSD, &record.TotalCostUSD,
			&record.LatencyMs, &record.Success, &record.ErrorMessage, &record.CreatedAt,
		); err != nil {
			return nil, err
		}
		results = append(results, record)
	}

	return results, nil
}

// AggregateDailyData aggregates raw usage data into daily aggregates
func (r *UsageRepository) AggregateDailyData(ctx context.Context, date time.Time) error {
	dateStr := date.Format("2006-01-02")

	query := `
		INSERT INTO usage_daily_aggregates (
			id, tenant_id, project_id, date, provider, model,
			total_requests, successful_requests, failed_requests,
			total_input_tokens, total_output_tokens, total_cost_usd,
			avg_latency_ms, min_latency_ms, max_latency_ms,
			created_at, updated_at
		)
		SELECT
			gen_random_uuid(),
			tenant_id,
			project_id,
			DATE(created_at),
			provider,
			model,
			COUNT(*),
			COUNT(*) FILTER (WHERE success = TRUE),
			COUNT(*) FILTER (WHERE success = FALSE),
			COALESCE(SUM(input_tokens), 0),
			COALESCE(SUM(output_tokens), 0),
			COALESCE(SUM(total_cost_usd), 0),
			AVG(latency_ms)::INT,
			MIN(latency_ms),
			MAX(latency_ms),
			NOW(),
			NOW()
		FROM usage_records
		WHERE DATE(created_at) = $1
		GROUP BY tenant_id, project_id, DATE(created_at), provider, model
		ON CONFLICT (tenant_id, COALESCE(project_id, '00000000-0000-0000-0000-000000000000'::uuid), date, provider, model)
		DO UPDATE SET
			total_requests = EXCLUDED.total_requests,
			successful_requests = EXCLUDED.successful_requests,
			failed_requests = EXCLUDED.failed_requests,
			total_input_tokens = EXCLUDED.total_input_tokens,
			total_output_tokens = EXCLUDED.total_output_tokens,
			total_cost_usd = EXCLUDED.total_cost_usd,
			avg_latency_ms = EXCLUDED.avg_latency_ms,
			min_latency_ms = EXCLUDED.min_latency_ms,
			max_latency_ms = EXCLUDED.max_latency_ms,
			updated_at = NOW()
	`
	_, err := r.pool.Exec(ctx, query, dateStr)
	return err
}

// GetCurrentSpend retrieves current spend for budget checking
func (r *UsageRepository) GetCurrentSpend(ctx context.Context, tenantID uuid.UUID, projectID *uuid.UUID, budgetType domain.BudgetType) (float64, error) {
	var start time.Time
	now := time.Now()

	switch budgetType {
	case domain.BudgetTypeDaily:
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case domain.BudgetTypeMonthly:
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	default:
		return 0, fmt.Errorf("invalid budget type: %s", budgetType)
	}

	var query string
	var args []interface{}

	if projectID == nil {
		query = `
			SELECT COALESCE(SUM(total_cost_usd), 0)
			FROM usage_records
			WHERE tenant_id = $1 AND created_at >= $2
		`
		args = []interface{}{tenantID, start}
	} else {
		query = `
			SELECT COALESCE(SUM(total_cost_usd), 0)
			FROM usage_records
			WHERE tenant_id = $1 AND project_id = $2 AND created_at >= $3
		`
		args = []interface{}{tenantID, projectID, start}
	}

	var spend float64
	err := r.pool.QueryRow(ctx, query, args...).Scan(&spend)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}

	return spend, nil
}

// BudgetRepository implements repository.BudgetRepository
type BudgetRepository struct {
	pool *pgxpool.Pool
}

// NewBudgetRepository creates a new BudgetRepository
func NewBudgetRepository(pool *pgxpool.Pool) *BudgetRepository {
	return &BudgetRepository{pool: pool}
}

// Create creates a new budget
func (r *BudgetRepository) Create(ctx context.Context, budget *domain.UsageBudget) error {
	query := `
		INSERT INTO usage_budgets (
			id, tenant_id, project_id, budget_type, budget_amount_usd,
			alert_threshold, enabled, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.pool.Exec(ctx, query,
		budget.ID, budget.TenantID, budget.ProjectID, budget.BudgetType,
		budget.BudgetAmountUSD, budget.AlertThreshold, budget.Enabled,
		budget.CreatedAt, budget.UpdatedAt,
	)
	return err
}

// GetByID retrieves a budget by ID
func (r *BudgetRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.UsageBudget, error) {
	query := `
		SELECT id, tenant_id, project_id, budget_type, budget_amount_usd,
		       alert_threshold, enabled, created_at, updated_at
		FROM usage_budgets
		WHERE id = $1 AND tenant_id = $2
	`
	var budget domain.UsageBudget
	err := r.pool.QueryRow(ctx, query, id, tenantID).Scan(
		&budget.ID, &budget.TenantID, &budget.ProjectID, &budget.BudgetType,
		&budget.BudgetAmountUSD, &budget.AlertThreshold, &budget.Enabled,
		&budget.CreatedAt, &budget.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("budget not found: %s", id)
	}
	if err != nil {
		return nil, err
	}
	return &budget, nil
}

// List retrieves all budgets for a tenant
func (r *BudgetRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.UsageBudget, error) {
	query := `
		SELECT id, tenant_id, project_id, budget_type, budget_amount_usd,
		       alert_threshold, enabled, created_at, updated_at
		FROM usage_budgets
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var budgets []*domain.UsageBudget
	for rows.Next() {
		var budget domain.UsageBudget
		if err := rows.Scan(
			&budget.ID, &budget.TenantID, &budget.ProjectID, &budget.BudgetType,
			&budget.BudgetAmountUSD, &budget.AlertThreshold, &budget.Enabled,
			&budget.CreatedAt, &budget.UpdatedAt,
		); err != nil {
			return nil, err
		}
		budgets = append(budgets, &budget)
	}

	return budgets, nil
}

// GetByProject retrieves budget for a specific project
func (r *BudgetRepository) GetByProject(ctx context.Context, tenantID uuid.UUID, projectID *uuid.UUID, budgetType domain.BudgetType) (*domain.UsageBudget, error) {
	var query string
	var args []interface{}

	if projectID == nil {
		query = `
			SELECT id, tenant_id, project_id, budget_type, budget_amount_usd,
			       alert_threshold, enabled, created_at, updated_at
			FROM usage_budgets
			WHERE tenant_id = $1 AND project_id IS NULL AND budget_type = $2
		`
		args = []interface{}{tenantID, budgetType}
	} else {
		query = `
			SELECT id, tenant_id, project_id, budget_type, budget_amount_usd,
			       alert_threshold, enabled, created_at, updated_at
			FROM usage_budgets
			WHERE tenant_id = $1 AND project_id = $2 AND budget_type = $3
		`
		args = []interface{}{tenantID, projectID, budgetType}
	}

	var budget domain.UsageBudget
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&budget.ID, &budget.TenantID, &budget.ProjectID, &budget.BudgetType,
		&budget.BudgetAmountUSD, &budget.AlertThreshold, &budget.Enabled,
		&budget.CreatedAt, &budget.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // No budget set
	}
	if err != nil {
		return nil, err
	}
	return &budget, nil
}

// Update updates a budget
func (r *BudgetRepository) Update(ctx context.Context, budget *domain.UsageBudget) error {
	budget.UpdatedAt = time.Now()
	query := `
		UPDATE usage_budgets
		SET budget_amount_usd = $1, alert_threshold = $2, enabled = $3, updated_at = $4
		WHERE id = $5 AND tenant_id = $6
	`
	result, err := r.pool.Exec(ctx, query,
		budget.BudgetAmountUSD, budget.AlertThreshold, budget.Enabled,
		budget.UpdatedAt, budget.ID, budget.TenantID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("budget not found: %s", budget.ID)
	}
	return nil
}

// Delete deletes a budget
func (r *BudgetRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	query := `DELETE FROM usage_budgets WHERE id = $1 AND tenant_id = $2`
	result, err := r.pool.Exec(ctx, query, id, tenantID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("budget not found: %s", id)
	}
	return nil
}

// getPeriodRange returns start and end times for a period string
// Period format: "YYYY-MM" for month, "YYYY-MM-DD" for day
func getPeriodRange(period string) (start, end time.Time) {
	now := time.Now()

	if period == "" || period == "month" {
		// Current month
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 1, 0)
		return
	}

	if period == "day" {
		// Current day
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 0, 1)
		return
	}

	// Try parsing as YYYY-MM
	t, err := time.Parse("2006-01", period)
	if err == nil {
		start = t
		end = start.AddDate(0, 1, 0)
		return
	}

	// Try parsing as YYYY-MM-DD
	t, err = time.Parse("2006-01-02", period)
	if err == nil {
		start = t
		end = start.AddDate(0, 0, 1)
		return
	}

	// Fallback to current month
	start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	end = start.AddDate(0, 1, 0)
	return
}
