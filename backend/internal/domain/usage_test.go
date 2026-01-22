package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewUsageRecord(t *testing.T) {
	tenantID := uuid.New()
	projectID := uuid.New()
	runID := uuid.New()
	stepRunID := uuid.New()
	latency := 500

	record := NewUsageRecord(
		tenantID,
		&projectID,
		&runID,
		&stepRunID,
		"openai",
		"gpt-4",
		"chat_completion",
		1000,
		500,
		&latency,
		true,
		"",
	)

	if record.ID == uuid.Nil {
		t.Error("NewUsageRecord() should generate a non-nil UUID")
	}
	if record.TenantID != tenantID {
		t.Errorf("NewUsageRecord() TenantID = %v, want %v", record.TenantID, tenantID)
	}
	if record.ProjectID == nil || *record.ProjectID != projectID {
		t.Error("NewUsageRecord() ProjectID mismatch")
	}
	if record.RunID == nil || *record.RunID != runID {
		t.Error("NewUsageRecord() RunID mismatch")
	}
	if record.StepRunID == nil || *record.StepRunID != stepRunID {
		t.Error("NewUsageRecord() StepRunID mismatch")
	}
	if record.Provider != "openai" {
		t.Errorf("NewUsageRecord() Provider = %v, want openai", record.Provider)
	}
	if record.Model != "gpt-4" {
		t.Errorf("NewUsageRecord() Model = %v, want gpt-4", record.Model)
	}
	if record.Operation != "chat_completion" {
		t.Errorf("NewUsageRecord() Operation = %v, want chat_completion", record.Operation)
	}
	if record.InputTokens != 1000 {
		t.Errorf("NewUsageRecord() InputTokens = %v, want 1000", record.InputTokens)
	}
	if record.OutputTokens != 500 {
		t.Errorf("NewUsageRecord() OutputTokens = %v, want 500", record.OutputTokens)
	}
	if record.TotalTokens != 1500 {
		t.Errorf("NewUsageRecord() TotalTokens = %v, want 1500", record.TotalTokens)
	}
	if record.LatencyMs == nil || *record.LatencyMs != latency {
		t.Error("NewUsageRecord() LatencyMs mismatch")
	}
	if !record.Success {
		t.Error("NewUsageRecord() Success should be true")
	}
}

func TestNewUsageRecord_Failed(t *testing.T) {
	tenantID := uuid.New()
	errorMsg := "API rate limit exceeded"

	record := NewUsageRecord(
		tenantID,
		nil, nil, nil,
		"openai",
		"gpt-4",
		"chat_completion",
		100, 0,
		nil,
		false,
		errorMsg,
	)

	if record.Success {
		t.Error("NewUsageRecord() Success should be false")
	}
	if record.ErrorMessage != errorMsg {
		t.Errorf("NewUsageRecord() ErrorMessage = %v, want %v", record.ErrorMessage, errorMsg)
	}
}

func TestNewUsageBudget(t *testing.T) {
	tenantID := uuid.New()
	projectID := uuid.New()

	budget := NewUsageBudget(tenantID, &projectID, BudgetTypeMonthly, 100.0, 0.8)

	if budget.ID == uuid.Nil {
		t.Error("NewUsageBudget() should generate a non-nil UUID")
	}
	if budget.TenantID != tenantID {
		t.Errorf("NewUsageBudget() TenantID = %v, want %v", budget.TenantID, tenantID)
	}
	if budget.ProjectID == nil || *budget.ProjectID != projectID {
		t.Error("NewUsageBudget() ProjectID mismatch")
	}
	if budget.BudgetType != BudgetTypeMonthly {
		t.Errorf("NewUsageBudget() BudgetType = %v, want %v", budget.BudgetType, BudgetTypeMonthly)
	}
	if budget.BudgetAmountUSD != 100.0 {
		t.Errorf("NewUsageBudget() BudgetAmountUSD = %v, want 100.0", budget.BudgetAmountUSD)
	}
	if budget.AlertThreshold != 0.8 {
		t.Errorf("NewUsageBudget() AlertThreshold = %v, want 0.8", budget.AlertThreshold)
	}
	if !budget.Enabled {
		t.Error("NewUsageBudget() Enabled should be true")
	}
}

func TestNewUsageBudget_DefaultThreshold(t *testing.T) {
	tenantID := uuid.New()

	tests := []struct {
		name      string
		threshold float64
		want      float64
	}{
		{"zero threshold", 0, 0.8},
		{"negative threshold", -0.5, 0.8},
		{"over 1 threshold", 1.5, 0.8},
		{"valid threshold", 0.9, 0.9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			budget := NewUsageBudget(tenantID, nil, BudgetTypeDaily, 50.0, tt.threshold)
			if budget.AlertThreshold != tt.want {
				t.Errorf("NewUsageBudget() AlertThreshold = %v, want %v", budget.AlertThreshold, tt.want)
			}
		})
	}
}

func TestBudgetType_Constants(t *testing.T) {
	if BudgetTypeDaily != "daily" {
		t.Errorf("BudgetTypeDaily = %v, want daily", BudgetTypeDaily)
	}
	if BudgetTypeMonthly != "monthly" {
		t.Errorf("BudgetTypeMonthly = %v, want monthly", BudgetTypeMonthly)
	}
}
