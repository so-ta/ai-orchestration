package domain

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestTenantStatus_IsValid(t *testing.T) {
	validStatuses := []TenantStatus{
		TenantStatusActive,
		TenantStatusSuspended,
		TenantStatusPending,
		TenantStatusInactive,
	}

	for _, status := range validStatuses {
		t.Run(string(status), func(t *testing.T) {
			if !status.IsValid() {
				t.Errorf("IsValid() = false for valid status %v", status)
			}
		})
	}

	invalidStatuses := []TenantStatus{
		TenantStatus("invalid"),
		TenantStatus(""),
	}

	for _, status := range invalidStatuses {
		t.Run(string(status), func(t *testing.T) {
			if status.IsValid() {
				t.Errorf("IsValid() = true for invalid status %v", status)
			}
		})
	}
}

func TestTenantPlan_IsValid(t *testing.T) {
	validPlans := []TenantPlan{
		TenantPlanFree,
		TenantPlanStarter,
		TenantPlanProfessional,
		TenantPlanEnterprise,
	}

	for _, plan := range validPlans {
		t.Run(string(plan), func(t *testing.T) {
			if !plan.IsValid() {
				t.Errorf("IsValid() = false for valid plan %v", plan)
			}
		})
	}

	invalidPlans := []TenantPlan{
		TenantPlan("invalid"),
		TenantPlan(""),
	}

	for _, plan := range invalidPlans {
		t.Run(string(plan), func(t *testing.T) {
			if plan.IsValid() {
				t.Errorf("IsValid() = true for invalid plan %v", plan)
			}
		})
	}
}

func TestNewTenant(t *testing.T) {
	name := "Test Tenant"
	slug := "test-tenant"
	plan := TenantPlanStarter

	tenant, err := NewTenant(name, slug, plan)
	if err != nil {
		t.Fatalf("NewTenant() error = %v", err)
	}

	if tenant.ID == uuid.Nil {
		t.Error("NewTenant() should generate a non-nil UUID")
	}
	if tenant.Name != name {
		t.Errorf("NewTenant() Name = %v, want %v", tenant.Name, name)
	}
	if tenant.Slug != slug {
		t.Errorf("NewTenant() Slug = %v, want %v", tenant.Slug, slug)
	}
	if tenant.Plan != plan {
		t.Errorf("NewTenant() Plan = %v, want %v", tenant.Plan, plan)
	}
	if tenant.Status != TenantStatusActive {
		t.Errorf("NewTenant() Status = %v, want %v", tenant.Status, TenantStatusActive)
	}
}

func TestNewTenant_DefaultPlan(t *testing.T) {
	tenant, err := NewTenant("Test", "test", "")
	if err != nil {
		t.Fatalf("NewTenant() error = %v", err)
	}

	if tenant.Plan != TenantPlanFree {
		t.Errorf("NewTenant() with empty plan = %v, want %v", tenant.Plan, TenantPlanFree)
	}
}

func TestDefaultFeatureFlags(t *testing.T) {
	tests := []struct {
		plan                  TenantPlan
		expectCopilotEnabled  bool
		expectMaxConcurrent   int
	}{
		{TenantPlanFree, false, 2},
		{TenantPlanStarter, true, 10},
		{TenantPlanProfessional, true, 20},
		{TenantPlanEnterprise, true, 50},
	}

	for _, tt := range tests {
		t.Run(string(tt.plan), func(t *testing.T) {
			flags := DefaultFeatureFlags(tt.plan)
			if flags.CopilotEnabled != tt.expectCopilotEnabled {
				t.Errorf("CopilotEnabled = %v, want %v", flags.CopilotEnabled, tt.expectCopilotEnabled)
			}
			if flags.MaxConcurrentRuns != tt.expectMaxConcurrent {
				t.Errorf("MaxConcurrentRuns = %v, want %v", flags.MaxConcurrentRuns, tt.expectMaxConcurrent)
			}
		})
	}
}

func TestDefaultLimits(t *testing.T) {
	tests := []struct {
		plan            TenantPlan
		expectWorkflows int
		expectRunsPerDay int
	}{
		{TenantPlanFree, 5, 50},
		{TenantPlanStarter, 25, 250},
		{TenantPlanProfessional, 100, 1000},
		{TenantPlanEnterprise, -1, -1},
	}

	for _, tt := range tests {
		t.Run(string(tt.plan), func(t *testing.T) {
			limits := DefaultLimits(tt.plan)
			if limits.MaxWorkflows != tt.expectWorkflows {
				t.Errorf("MaxWorkflows = %v, want %v", limits.MaxWorkflows, tt.expectWorkflows)
			}
			if limits.MaxRunsPerDay != tt.expectRunsPerDay {
				t.Errorf("MaxRunsPerDay = %v, want %v", limits.MaxRunsPerDay, tt.expectRunsPerDay)
			}
		})
	}
}

func TestTenant_GetFeatureFlags(t *testing.T) {
	tenant, _ := NewTenant("Test", "test", TenantPlanStarter)

	flags, err := tenant.GetFeatureFlags()
	if err != nil {
		t.Fatalf("GetFeatureFlags() error = %v", err)
	}

	if !flags.CopilotEnabled {
		t.Error("GetFeatureFlags() CopilotEnabled should be true for Starter plan")
	}
}

func TestTenant_GetLimits(t *testing.T) {
	tenant, _ := NewTenant("Test", "test", TenantPlanStarter)

	limits, err := tenant.GetLimits()
	if err != nil {
		t.Fatalf("GetLimits() error = %v", err)
	}

	if limits.MaxWorkflows != 25 {
		t.Errorf("GetLimits() MaxWorkflows = %v, want 25", limits.MaxWorkflows)
	}
}

func TestTenant_GetMetadata(t *testing.T) {
	tenant, _ := NewTenant("Test", "test", TenantPlanFree)

	metadata, err := tenant.GetMetadata()
	if err != nil {
		t.Fatalf("GetMetadata() error = %v", err)
	}

	// Default metadata should be empty
	if metadata.Industry != "" {
		t.Errorf("GetMetadata() Industry = %v, want empty", metadata.Industry)
	}
}

func TestTenant_GetVariables(t *testing.T) {
	tenant, _ := NewTenant("Test", "test", TenantPlanFree)

	// Test with default empty variables
	vars, err := tenant.GetVariables()
	if err != nil {
		t.Fatalf("GetVariables() error = %v", err)
	}
	if len(vars) != 0 {
		t.Errorf("GetVariables() len = %v, want 0", len(vars))
	}

	// Test with custom variables
	tenant.Variables = json.RawMessage(`{"key": "value"}`)
	vars, err = tenant.GetVariables()
	if err != nil {
		t.Fatalf("GetVariables() error = %v", err)
	}
	if vars["key"] != "value" {
		t.Errorf("GetVariables() key = %v, want value", vars["key"])
	}
}

func TestTenant_Suspend(t *testing.T) {
	tenant, _ := NewTenant("Test", "test", TenantPlanFree)
	reason := "Payment overdue"

	tenant.Suspend(reason)

	if tenant.Status != TenantStatusSuspended {
		t.Errorf("Suspend() Status = %v, want %v", tenant.Status, TenantStatusSuspended)
	}
	if tenant.SuspendedReason != reason {
		t.Errorf("Suspend() SuspendedReason = %v, want %v", tenant.SuspendedReason, reason)
	}
	if tenant.SuspendedAt == nil {
		t.Error("Suspend() SuspendedAt should not be nil")
	}
}

func TestTenant_Activate(t *testing.T) {
	tenant, _ := NewTenant("Test", "test", TenantPlanFree)
	tenant.Suspend("Test")

	tenant.Activate()

	if tenant.Status != TenantStatusActive {
		t.Errorf("Activate() Status = %v, want %v", tenant.Status, TenantStatusActive)
	}
	if tenant.SuspendedAt != nil {
		t.Error("Activate() SuspendedAt should be nil")
	}
	if tenant.SuspendedReason != "" {
		t.Error("Activate() SuspendedReason should be empty")
	}
}

func TestTenant_IsActive(t *testing.T) {
	tests := []struct {
		status TenantStatus
		want   bool
	}{
		{TenantStatusActive, true},
		{TenantStatusSuspended, false},
		{TenantStatusPending, false},
		{TenantStatusInactive, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			tenant := &Tenant{Status: tt.status}
			if got := tenant.IsActive(); got != tt.want {
				t.Errorf("IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTenant_IsSuspended(t *testing.T) {
	tests := []struct {
		status TenantStatus
		want   bool
	}{
		{TenantStatusActive, false},
		{TenantStatusSuspended, true},
		{TenantStatusPending, false},
		{TenantStatusInactive, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			tenant := &Tenant{Status: tt.status}
			if got := tenant.IsSuspended(); got != tt.want {
				t.Errorf("IsSuspended() = %v, want %v", got, tt.want)
			}
		})
	}
}
