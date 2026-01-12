package domain

import (
	"time"

	"github.com/google/uuid"
)

// BudgetType represents the type of budget period
type BudgetType string

const (
	BudgetTypeDaily   BudgetType = "daily"
	BudgetTypeMonthly BudgetType = "monthly"
)

// UsageRecord represents a single LLM API call record
type UsageRecord struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	WorkflowID  *uuid.UUID `json:"workflow_id,omitempty"`
	RunID       *uuid.UUID `json:"run_id,omitempty"`
	StepRunID   *uuid.UUID `json:"step_run_id,omitempty"`

	// Provider information
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	Operation string `json:"operation"`

	// Token usage
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`

	// Cost in USD
	InputCostUSD  float64 `json:"input_cost_usd"`
	OutputCostUSD float64 `json:"output_cost_usd"`
	TotalCostUSD  float64 `json:"total_cost_usd"`

	// Metadata
	LatencyMs    *int   `json:"latency_ms,omitempty"`
	Success      bool   `json:"success"`
	ErrorMessage string `json:"error_message,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

// NewUsageRecord creates a new UsageRecord with calculated costs
func NewUsageRecord(
	tenantID uuid.UUID,
	workflowID, runID, stepRunID *uuid.UUID,
	provider, model, operation string,
	inputTokens, outputTokens int,
	latencyMs *int,
	success bool,
	errorMessage string,
) *UsageRecord {
	inputCost, outputCost, totalCost := CalculateCost(provider, model, inputTokens, outputTokens)

	return &UsageRecord{
		ID:            uuid.New(),
		TenantID:      tenantID,
		WorkflowID:    workflowID,
		RunID:         runID,
		StepRunID:     stepRunID,
		Provider:      provider,
		Model:         model,
		Operation:     operation,
		InputTokens:   inputTokens,
		OutputTokens:  outputTokens,
		TotalTokens:   inputTokens + outputTokens,
		InputCostUSD:  inputCost,
		OutputCostUSD: outputCost,
		TotalCostUSD:  totalCost,
		LatencyMs:     latencyMs,
		Success:       success,
		ErrorMessage:  errorMessage,
		CreatedAt:     time.Now(),
	}
}

// UsageDailyAggregate represents pre-aggregated daily usage data
type UsageDailyAggregate struct {
	ID         uuid.UUID  `json:"id"`
	TenantID   uuid.UUID  `json:"tenant_id"`
	WorkflowID *uuid.UUID `json:"workflow_id,omitempty"`
	Date       time.Time  `json:"date"`
	Provider   string     `json:"provider"`
	Model      string     `json:"model"`

	// Aggregated metrics
	TotalRequests      int     `json:"total_requests"`
	SuccessfulRequests int     `json:"successful_requests"`
	FailedRequests     int     `json:"failed_requests"`
	TotalInputTokens   int64   `json:"total_input_tokens"`
	TotalOutputTokens  int64   `json:"total_output_tokens"`
	TotalCostUSD       float64 `json:"total_cost_usd"`
	AvgLatencyMs       *int    `json:"avg_latency_ms,omitempty"`
	MinLatencyMs       *int    `json:"min_latency_ms,omitempty"`
	MaxLatencyMs       *int    `json:"max_latency_ms,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UsageBudget represents a budget configuration
type UsageBudget struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	WorkflowID      *uuid.UUID `json:"workflow_id,omitempty"`
	BudgetType      BudgetType `json:"budget_type"`
	BudgetAmountUSD float64    `json:"budget_amount_usd"`
	AlertThreshold  float64    `json:"alert_threshold"` // 0.00 - 1.00
	Enabled         bool       `json:"enabled"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// NewUsageBudget creates a new UsageBudget
func NewUsageBudget(
	tenantID uuid.UUID,
	workflowID *uuid.UUID,
	budgetType BudgetType,
	budgetAmountUSD float64,
	alertThreshold float64,
) *UsageBudget {
	if alertThreshold <= 0 || alertThreshold > 1 {
		alertThreshold = 0.80 // Default to 80%
	}

	return &UsageBudget{
		ID:              uuid.New(),
		TenantID:        tenantID,
		WorkflowID:      workflowID,
		BudgetType:      budgetType,
		BudgetAmountUSD: budgetAmountUSD,
		AlertThreshold:  alertThreshold,
		Enabled:         true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

// UsageSummary represents aggregated usage data for a period
type UsageSummary struct {
	Period           string            `json:"period"`
	TotalCostUSD     float64           `json:"total_cost_usd"`
	TotalRequests    int               `json:"total_requests"`
	TotalInputTokens int64             `json:"total_input_tokens"`
	TotalOutputTokens int64            `json:"total_output_tokens"`
	ByProvider       map[string]ProviderUsage `json:"by_provider"`
	ByModel          map[string]ModelUsage    `json:"by_model"`
	Budget           *BudgetStatus            `json:"budget,omitempty"`
}

// ProviderUsage represents usage data for a single provider
type ProviderUsage struct {
	CostUSD  float64 `json:"cost_usd"`
	Requests int     `json:"requests"`
}

// ModelUsage represents usage data for a single model
type ModelUsage struct {
	Provider     string  `json:"provider"`
	CostUSD      float64 `json:"cost_usd"`
	Requests     int     `json:"requests"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
}

// BudgetStatus represents current budget consumption status
type BudgetStatus struct {
	MonthlyLimitUSD  *float64 `json:"monthly_limit_usd,omitempty"`
	DailyLimitUSD    *float64 `json:"daily_limit_usd,omitempty"`
	ConsumedPercent  float64  `json:"consumed_percent"`
	AlertTriggered   bool     `json:"alert_triggered"`
}

// DailyUsage represents usage data for a single day
type DailyUsage struct {
	Date         time.Time `json:"date"`
	TotalCostUSD float64   `json:"total_cost_usd"`
	TotalRequests int      `json:"total_requests"`
	TotalTokens  int64     `json:"total_tokens"`
}

// WorkflowUsage represents usage data for a single workflow
type WorkflowUsage struct {
	WorkflowID   uuid.UUID `json:"workflow_id"`
	WorkflowName string    `json:"workflow_name"`
	TotalCostUSD float64   `json:"total_cost_usd"`
	TotalRequests int      `json:"total_requests"`
	TotalTokens  int64     `json:"total_tokens"`
}
