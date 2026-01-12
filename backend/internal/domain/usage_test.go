package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewUsageRecord(t *testing.T) {
	tenantID := uuid.New()
	workflowID := uuid.New()
	runID := uuid.New()
	stepRunID := uuid.New()

	record := NewUsageRecord(
		tenantID,
		&workflowID,
		&runID,
		&stepRunID,
		"openai",
		"gpt-4o",
		"chat",
		1000,
		500,
		nil,
		true,
		"",
	)

	// Verify fields
	if record.ID == uuid.Nil {
		t.Error("NewUsageRecord() ID should not be nil")
	}
	if record.TenantID != tenantID {
		t.Errorf("NewUsageRecord() TenantID = %v, want %v", record.TenantID, tenantID)
	}
	if *record.WorkflowID != workflowID {
		t.Errorf("NewUsageRecord() WorkflowID = %v, want %v", *record.WorkflowID, workflowID)
	}
	if record.Provider != "openai" {
		t.Errorf("NewUsageRecord() Provider = %v, want openai", record.Provider)
	}
	if record.Model != "gpt-4o" {
		t.Errorf("NewUsageRecord() Model = %v, want gpt-4o", record.Model)
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
	if !record.Success {
		t.Error("NewUsageRecord() Success should be true")
	}

	// Verify cost calculation
	expectedInputCost := 1000.0 / 1000.0 * 0.0025  // GPT-4o input rate
	expectedOutputCost := 500.0 / 1000.0 * 0.01    // GPT-4o output rate
	expectedTotalCost := expectedInputCost + expectedOutputCost

	if record.InputCostUSD != expectedInputCost {
		t.Errorf("NewUsageRecord() InputCostUSD = %v, want %v", record.InputCostUSD, expectedInputCost)
	}
	if record.OutputCostUSD != expectedOutputCost {
		t.Errorf("NewUsageRecord() OutputCostUSD = %v, want %v", record.OutputCostUSD, expectedOutputCost)
	}
	if record.TotalCostUSD != expectedTotalCost {
		t.Errorf("NewUsageRecord() TotalCostUSD = %v, want %v", record.TotalCostUSD, expectedTotalCost)
	}
}

func TestNewUsageRecord_WithLatency(t *testing.T) {
	tenantID := uuid.New()
	latencyMs := 150

	record := NewUsageRecord(
		tenantID,
		nil,
		nil,
		nil,
		"anthropic",
		"claude-3-sonnet",
		"chat",
		500,
		200,
		&latencyMs,
		true,
		"",
	)

	if record.LatencyMs == nil {
		t.Error("NewUsageRecord() LatencyMs should not be nil")
	}
	if *record.LatencyMs != latencyMs {
		t.Errorf("NewUsageRecord() LatencyMs = %v, want %v", *record.LatencyMs, latencyMs)
	}
}

func TestNewUsageRecord_WithError(t *testing.T) {
	tenantID := uuid.New()
	errorMsg := "API rate limit exceeded"

	record := NewUsageRecord(
		tenantID,
		nil,
		nil,
		nil,
		"openai",
		"gpt-4o",
		"chat",
		100,
		0,
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

	budget := NewUsageBudget(
		tenantID,
		nil,
		BudgetTypeMonthly,
		100.0,
		0.80,
	)

	if budget.ID == uuid.Nil {
		t.Error("NewUsageBudget() ID should not be nil")
	}
	if budget.TenantID != tenantID {
		t.Errorf("NewUsageBudget() TenantID = %v, want %v", budget.TenantID, tenantID)
	}
	if budget.WorkflowID != nil {
		t.Error("NewUsageBudget() WorkflowID should be nil")
	}
	if budget.BudgetType != BudgetTypeMonthly {
		t.Errorf("NewUsageBudget() BudgetType = %v, want monthly", budget.BudgetType)
	}
	if budget.BudgetAmountUSD != 100.0 {
		t.Errorf("NewUsageBudget() BudgetAmountUSD = %v, want 100.0", budget.BudgetAmountUSD)
	}
	if budget.AlertThreshold != 0.80 {
		t.Errorf("NewUsageBudget() AlertThreshold = %v, want 0.80", budget.AlertThreshold)
	}
	if !budget.Enabled {
		t.Error("NewUsageBudget() Enabled should be true")
	}
}

func TestNewUsageBudget_DefaultThreshold(t *testing.T) {
	tenantID := uuid.New()

	// Test with invalid threshold (should default to 0.80)
	budget := NewUsageBudget(
		tenantID,
		nil,
		BudgetTypeDaily,
		50.0,
		0, // Invalid threshold
	)

	if budget.AlertThreshold != 0.80 {
		t.Errorf("NewUsageBudget() AlertThreshold = %v, want 0.80 (default)", budget.AlertThreshold)
	}

	// Test with threshold > 1 (should default to 0.80)
	budget2 := NewUsageBudget(
		tenantID,
		nil,
		BudgetTypeDaily,
		50.0,
		1.5, // Invalid threshold
	)

	if budget2.AlertThreshold != 0.80 {
		t.Errorf("NewUsageBudget() AlertThreshold = %v, want 0.80 (default)", budget2.AlertThreshold)
	}
}

func TestNewUsageBudget_WithWorkflow(t *testing.T) {
	tenantID := uuid.New()
	workflowID := uuid.New()

	budget := NewUsageBudget(
		tenantID,
		&workflowID,
		BudgetTypeMonthly,
		200.0,
		0.90,
	)

	if budget.WorkflowID == nil {
		t.Error("NewUsageBudget() WorkflowID should not be nil")
	}
	if *budget.WorkflowID != workflowID {
		t.Errorf("NewUsageBudget() WorkflowID = %v, want %v", *budget.WorkflowID, workflowID)
	}
}
