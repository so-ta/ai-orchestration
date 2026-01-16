package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// Usage-related errors
var (
	ErrBudgetAlreadyExists = errors.New("budget already exists for this scope and type")
)

// UsageUsecase handles usage-related business logic
type UsageUsecase struct {
	usageRepo  repository.UsageRepository
	budgetRepo repository.BudgetRepository
}

// NewUsageUsecase creates a new UsageUsecase
func NewUsageUsecase(usageRepo repository.UsageRepository, budgetRepo repository.BudgetRepository) *UsageUsecase {
	return &UsageUsecase{
		usageRepo:  usageRepo,
		budgetRepo: budgetRepo,
	}
}

// GetSummaryInput represents input for GetSummary
type GetSummaryInput struct {
	TenantID uuid.UUID
	Period   string // "month", "day", "YYYY-MM", "YYYY-MM-DD"
}

// GetSummary retrieves usage summary for a tenant
func (u *UsageUsecase) GetSummary(ctx context.Context, input GetSummaryInput) (*domain.UsageSummary, error) {
	summary, err := u.usageRepo.GetSummary(ctx, input.TenantID, input.Period)
	if err != nil {
		return nil, err
	}

	// Add budget status if available
	budget, err := u.budgetRepo.GetByProject(ctx, input.TenantID, nil, domain.BudgetTypeMonthly)
	if err == nil && budget != nil && budget.Enabled {
		currentSpend, err := u.usageRepo.GetCurrentSpend(ctx, input.TenantID, nil, domain.BudgetTypeMonthly)
		if err == nil {
			consumedPercent := 0.0
			if budget.BudgetAmountUSD > 0 {
				consumedPercent = currentSpend / budget.BudgetAmountUSD * 100
			}
			summary.Budget = &domain.BudgetStatus{
				MonthlyLimitUSD: &budget.BudgetAmountUSD,
				ConsumedPercent: consumedPercent,
				AlertTriggered:  consumedPercent >= budget.AlertThreshold*100,
			}
		}
	}

	return summary, nil
}

// GetDailyInput represents input for GetDaily
type GetDailyInput struct {
	TenantID uuid.UUID
	Start    time.Time
	End      time.Time
}

// GetDaily retrieves daily usage data
func (u *UsageUsecase) GetDaily(ctx context.Context, input GetDailyInput) ([]domain.DailyUsage, error) {
	return u.usageRepo.GetDaily(ctx, input.TenantID, input.Start, input.End)
}

// GetByProjectInput represents input for GetByProject
type GetByProjectInput struct {
	TenantID uuid.UUID
	Period   string
}

// GetByProject retrieves usage data grouped by project
func (u *UsageUsecase) GetByProject(ctx context.Context, input GetByProjectInput) ([]domain.ProjectUsage, error) {
	return u.usageRepo.GetByProject(ctx, input.TenantID, input.Period)
}

// GetByModelInput represents input for GetByModel
type GetByModelInput struct {
	TenantID uuid.UUID
	Period   string
}

// GetByModel retrieves usage data grouped by model
func (u *UsageUsecase) GetByModel(ctx context.Context, input GetByModelInput) (map[string]domain.ModelUsage, error) {
	return u.usageRepo.GetByModel(ctx, input.TenantID, input.Period)
}

// GetByRunInput represents input for GetByRun
type GetByRunInput struct {
	TenantID uuid.UUID
	RunID    uuid.UUID
}

// GetByRun retrieves usage records for a specific run
func (u *UsageUsecase) GetByRun(ctx context.Context, input GetByRunInput) ([]domain.UsageRecord, error) {
	return u.usageRepo.GetByRun(ctx, input.TenantID, input.RunID)
}

// ListBudgetsInput represents input for ListBudgets
type ListBudgetsInput struct {
	TenantID uuid.UUID
}

// ListBudgets retrieves all budgets for a tenant
func (u *UsageUsecase) ListBudgets(ctx context.Context, input ListBudgetsInput) ([]*domain.UsageBudget, error) {
	return u.budgetRepo.List(ctx, input.TenantID)
}

// CreateBudgetInput represents input for CreateBudget
type CreateBudgetInput struct {
	TenantID        uuid.UUID
	ProjectID       *uuid.UUID
	BudgetType      domain.BudgetType
	BudgetAmountUSD float64
	AlertThreshold  float64
}

// CreateBudget creates a new budget
func (u *UsageUsecase) CreateBudget(ctx context.Context, input CreateBudgetInput) (*domain.UsageBudget, error) {
	// Check if budget already exists
	existing, err := u.budgetRepo.GetByProject(ctx, input.TenantID, input.ProjectID, input.BudgetType)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrBudgetAlreadyExists
	}

	budget := domain.NewUsageBudget(
		input.TenantID,
		input.ProjectID,
		input.BudgetType,
		input.BudgetAmountUSD,
		input.AlertThreshold,
	)

	if err := u.budgetRepo.Create(ctx, budget); err != nil {
		return nil, err
	}

	return budget, nil
}

// UpdateBudgetInput represents input for UpdateBudget
type UpdateBudgetInput struct {
	TenantID        uuid.UUID
	BudgetID        uuid.UUID
	BudgetAmountUSD *float64
	AlertThreshold  *float64
	Enabled         *bool
}

// UpdateBudget updates an existing budget
func (u *UsageUsecase) UpdateBudget(ctx context.Context, input UpdateBudgetInput) (*domain.UsageBudget, error) {
	budget, err := u.budgetRepo.GetByID(ctx, input.TenantID, input.BudgetID)
	if err != nil {
		return nil, err
	}

	if input.BudgetAmountUSD != nil {
		budget.BudgetAmountUSD = *input.BudgetAmountUSD
	}
	if input.AlertThreshold != nil {
		budget.AlertThreshold = *input.AlertThreshold
	}
	if input.Enabled != nil {
		budget.Enabled = *input.Enabled
	}

	if err := u.budgetRepo.Update(ctx, budget); err != nil {
		return nil, err
	}

	return budget, nil
}

// DeleteBudgetInput represents input for DeleteBudget
type DeleteBudgetInput struct {
	TenantID uuid.UUID
	BudgetID uuid.UUID
}

// DeleteBudget deletes a budget
func (u *UsageUsecase) DeleteBudget(ctx context.Context, input DeleteBudgetInput) error {
	return u.budgetRepo.Delete(ctx, input.TenantID, input.BudgetID)
}

// GetPricingOutput represents output for GetPricing
type GetPricingOutput struct {
	Pricing []domain.TokenPricing `json:"pricing"`
}

// GetPricing returns all available pricing configurations
func (u *UsageUsecase) GetPricing(ctx context.Context) (*GetPricingOutput, error) {
	return &GetPricingOutput{
		Pricing: domain.GetAllPricing(),
	}, nil
}
