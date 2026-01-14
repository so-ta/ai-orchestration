package domain

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTenant(t *testing.T) {
	tests := []struct {
		name     string
		tenantName string
		slug     string
		plan     TenantPlan
		wantPlan TenantPlan
	}{
		{
			name:       "create tenant with free plan",
			tenantName: "Test Tenant",
			slug:       "test-tenant",
			plan:       TenantPlanFree,
			wantPlan:   TenantPlanFree,
		},
		{
			name:       "create tenant with starter plan",
			tenantName: "Starter Tenant",
			slug:       "starter-tenant",
			plan:       TenantPlanStarter,
			wantPlan:   TenantPlanStarter,
		},
		{
			name:       "create tenant with professional plan",
			tenantName: "Pro Tenant",
			slug:       "pro-tenant",
			plan:       TenantPlanProfessional,
			wantPlan:   TenantPlanProfessional,
		},
		{
			name:       "create tenant with enterprise plan",
			tenantName: "Enterprise Tenant",
			slug:       "enterprise-tenant",
			plan:       TenantPlanEnterprise,
			wantPlan:   TenantPlanEnterprise,
		},
		{
			name:       "create tenant with empty plan defaults to free",
			tenantName: "Default Tenant",
			slug:       "default-tenant",
			plan:       "",
			wantPlan:   TenantPlanFree,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenant, err := NewTenant(tt.tenantName, tt.slug, tt.plan)
			require.NoError(t, err)
			require.NotNil(t, tenant)

			assert.NotEmpty(t, tenant.ID)
			assert.Equal(t, tt.tenantName, tenant.Name)
			assert.Equal(t, tt.slug, tenant.Slug)
			assert.Equal(t, tt.wantPlan, tenant.Plan)
			assert.Equal(t, TenantStatusActive, tenant.Status)

			// Verify FeatureFlags is valid JSON
			var flags TenantFeatureFlags
			err = json.Unmarshal(tenant.FeatureFlags, &flags)
			require.NoError(t, err)

			// Verify Limits is valid JSON
			var limits TenantLimits
			err = json.Unmarshal(tenant.Limits, &limits)
			require.NoError(t, err)

			// Verify Settings and Metadata are initialized
			assert.Equal(t, json.RawMessage("{}"), tenant.Settings)
			assert.Equal(t, json.RawMessage("{}"), tenant.Metadata)

			// Verify timestamps are set
			assert.False(t, tenant.CreatedAt.IsZero())
			assert.False(t, tenant.UpdatedAt.IsZero())
		})
	}
}

func TestNewTenant_FeatureFlags(t *testing.T) {
	tests := []struct {
		name     string
		plan     TenantPlan
		checkFunc func(t *testing.T, flags TenantFeatureFlags)
	}{
		{
			name: "free plan has limited features",
			plan: TenantPlanFree,
			checkFunc: func(t *testing.T, flags TenantFeatureFlags) {
				assert.False(t, flags.CopilotEnabled)
				assert.False(t, flags.AdvancedAnalytics)
				assert.False(t, flags.CustomBlocks)
				assert.Equal(t, 2, flags.MaxConcurrentRuns)
			},
		},
		{
			name: "enterprise plan has all features",
			plan: TenantPlanEnterprise,
			checkFunc: func(t *testing.T, flags TenantFeatureFlags) {
				assert.True(t, flags.CopilotEnabled)
				assert.True(t, flags.AdvancedAnalytics)
				assert.True(t, flags.CustomBlocks)
				assert.True(t, flags.SSOEnabled)
				assert.True(t, flags.AuditLogs)
				assert.Equal(t, 50, flags.MaxConcurrentRuns)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenant, err := NewTenant("Test", "test", tt.plan)
			require.NoError(t, err)

			flags, err := tenant.GetFeatureFlags()
			require.NoError(t, err)

			tt.checkFunc(t, *flags)
		})
	}
}

func TestNewTenant_Limits(t *testing.T) {
	tests := []struct {
		name     string
		plan     TenantPlan
		checkFunc func(t *testing.T, limits TenantLimits)
	}{
		{
			name: "free plan has low limits",
			plan: TenantPlanFree,
			checkFunc: func(t *testing.T, limits TenantLimits) {
				assert.Equal(t, 5, limits.MaxWorkflows)
				assert.Equal(t, 50, limits.MaxRunsPerDay)
				assert.Equal(t, 3, limits.MaxUsers)
				assert.Equal(t, 7, limits.RetentionDays)
			},
		},
		{
			name: "enterprise plan has unlimited resources",
			plan: TenantPlanEnterprise,
			checkFunc: func(t *testing.T, limits TenantLimits) {
				assert.Equal(t, -1, limits.MaxWorkflows)
				assert.Equal(t, -1, limits.MaxRunsPerDay)
				assert.Equal(t, -1, limits.MaxUsers)
				assert.Equal(t, 365, limits.RetentionDays)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenant, err := NewTenant("Test", "test", tt.plan)
			require.NoError(t, err)

			limits, err := tenant.GetLimits()
			require.NoError(t, err)

			tt.checkFunc(t, *limits)
		})
	}
}

func TestTenantStatus_IsValid(t *testing.T) {
	tests := []struct {
		status TenantStatus
		valid  bool
	}{
		{TenantStatusActive, true},
		{TenantStatusSuspended, true},
		{TenantStatusPending, true},
		{TenantStatusInactive, true},
		{TenantStatus("invalid"), false},
		{TenantStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.status.IsValid())
		})
	}
}

func TestTenantPlan_IsValid(t *testing.T) {
	tests := []struct {
		plan  TenantPlan
		valid bool
	}{
		{TenantPlanFree, true},
		{TenantPlanStarter, true},
		{TenantPlanProfessional, true},
		{TenantPlanEnterprise, true},
		{TenantPlan("invalid"), false},
		{TenantPlan(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.plan), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.plan.IsValid())
		})
	}
}

func TestTenant_SuspendAndActivate(t *testing.T) {
	tenant, err := NewTenant("Test", "test", TenantPlanFree)
	require.NoError(t, err)

	assert.True(t, tenant.IsActive())
	assert.False(t, tenant.IsSuspended())
	assert.Nil(t, tenant.SuspendedAt)
	assert.Empty(t, tenant.SuspendedReason)

	// Suspend
	tenant.Suspend("Payment overdue")
	assert.False(t, tenant.IsActive())
	assert.True(t, tenant.IsSuspended())
	assert.NotNil(t, tenant.SuspendedAt)
	assert.Equal(t, "Payment overdue", tenant.SuspendedReason)

	// Activate
	tenant.Activate()
	assert.True(t, tenant.IsActive())
	assert.False(t, tenant.IsSuspended())
	assert.Nil(t, tenant.SuspendedAt)
	assert.Empty(t, tenant.SuspendedReason)
}
