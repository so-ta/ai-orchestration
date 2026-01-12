package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// TenantStatus represents the status of a tenant
type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusSuspended TenantStatus = "suspended"
	TenantStatusPending   TenantStatus = "pending"
	TenantStatusInactive  TenantStatus = "inactive"
)

// IsValid checks if the status is valid
func (s TenantStatus) IsValid() bool {
	switch s {
	case TenantStatusActive, TenantStatusSuspended, TenantStatusPending, TenantStatusInactive:
		return true
	}
	return false
}

// TenantPlan represents the subscription plan
type TenantPlan string

const (
	TenantPlanFree         TenantPlan = "free"
	TenantPlanStarter      TenantPlan = "starter"
	TenantPlanProfessional TenantPlan = "professional"
	TenantPlanEnterprise   TenantPlan = "enterprise"
)

// IsValid checks if the plan is valid
func (p TenantPlan) IsValid() bool {
	switch p {
	case TenantPlanFree, TenantPlanStarter, TenantPlanProfessional, TenantPlanEnterprise:
		return true
	}
	return false
}

// TenantFeatureFlags contains feature flag settings
type TenantFeatureFlags struct {
	CopilotEnabled    bool `json:"copilot_enabled"`
	AdvancedAnalytics bool `json:"advanced_analytics"`
	CustomBlocks      bool `json:"custom_blocks"`
	APIAccess         bool `json:"api_access"`
	SSOEnabled        bool `json:"sso_enabled"`
	AuditLogs         bool `json:"audit_logs"`
	MaxConcurrentRuns int  `json:"max_concurrent_runs"`
}

// DefaultFeatureFlags returns default feature flags for a plan
func DefaultFeatureFlags(plan TenantPlan) TenantFeatureFlags {
	switch plan {
	case TenantPlanEnterprise:
		return TenantFeatureFlags{
			CopilotEnabled:    true,
			AdvancedAnalytics: true,
			CustomBlocks:      true,
			APIAccess:         true,
			SSOEnabled:        true,
			AuditLogs:         true,
			MaxConcurrentRuns: 50,
		}
	case TenantPlanProfessional:
		return TenantFeatureFlags{
			CopilotEnabled:    true,
			AdvancedAnalytics: true,
			CustomBlocks:      true,
			APIAccess:         true,
			SSOEnabled:        false,
			AuditLogs:         true,
			MaxConcurrentRuns: 20,
		}
	case TenantPlanStarter:
		return TenantFeatureFlags{
			CopilotEnabled:    true,
			AdvancedAnalytics: false,
			CustomBlocks:      true,
			APIAccess:         true,
			SSOEnabled:        false,
			AuditLogs:         false,
			MaxConcurrentRuns: 10,
		}
	default: // Free
		return TenantFeatureFlags{
			CopilotEnabled:    false,
			AdvancedAnalytics: false,
			CustomBlocks:      false,
			APIAccess:         false,
			SSOEnabled:        false,
			AuditLogs:         false,
			MaxConcurrentRuns: 2,
		}
	}
}

// TenantLimits contains resource limits
type TenantLimits struct {
	MaxWorkflows   int `json:"max_workflows"`
	MaxRunsPerDay  int `json:"max_runs_per_day"`
	MaxUsers       int `json:"max_users"`
	MaxCredentials int `json:"max_credentials"`
	MaxStorageMB   int `json:"max_storage_mb"`
	RetentionDays  int `json:"retention_days"`
}

// DefaultLimits returns default limits for a plan
func DefaultLimits(plan TenantPlan) TenantLimits {
	switch plan {
	case TenantPlanEnterprise:
		return TenantLimits{
			MaxWorkflows:   -1, // Unlimited
			MaxRunsPerDay:  -1,
			MaxUsers:       -1,
			MaxCredentials: -1,
			MaxStorageMB:   102400, // 100GB
			RetentionDays:  365,
		}
	case TenantPlanProfessional:
		return TenantLimits{
			MaxWorkflows:   100,
			MaxRunsPerDay:  1000,
			MaxUsers:       50,
			MaxCredentials: 100,
			MaxStorageMB:   10240, // 10GB
			RetentionDays:  90,
		}
	case TenantPlanStarter:
		return TenantLimits{
			MaxWorkflows:   25,
			MaxRunsPerDay:  250,
			MaxUsers:       10,
			MaxCredentials: 25,
			MaxStorageMB:   2048, // 2GB
			RetentionDays:  30,
		}
	default: // Free
		return TenantLimits{
			MaxWorkflows:   5,
			MaxRunsPerDay:  50,
			MaxUsers:       3,
			MaxCredentials: 5,
			MaxStorageMB:   512, // 512MB
			RetentionDays:  7,
		}
	}
}

// TenantMetadata contains additional tenant information
type TenantMetadata struct {
	Industry    string `json:"industry,omitempty"`
	CompanySize string `json:"company_size,omitempty"`
	Website     string `json:"website,omitempty"`
	Country     string `json:"country,omitempty"`
	Notes       string `json:"notes,omitempty"`
}

// Tenant represents a tenant in the system
type Tenant struct {
	ID              uuid.UUID       `json:"id"`
	Name            string          `json:"name"`
	Slug            string          `json:"slug"`
	Status          TenantStatus    `json:"status"`
	Plan            TenantPlan      `json:"plan"`
	OwnerEmail      string          `json:"owner_email,omitempty"`
	OwnerName       string          `json:"owner_name,omitempty"`
	BillingEmail    string          `json:"billing_email,omitempty"`
	Settings        json.RawMessage `json:"settings"`
	Metadata        json.RawMessage `json:"metadata"`
	FeatureFlags    json.RawMessage `json:"feature_flags"`
	Limits          json.RawMessage `json:"limits"`
	SuspendedAt     *time.Time      `json:"suspended_at,omitempty"`
	SuspendedReason string          `json:"suspended_reason,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	DeletedAt       *time.Time      `json:"deleted_at,omitempty"`
}

// NewTenant creates a new tenant with defaults
func NewTenant(name, slug string, plan TenantPlan) *Tenant {
	if plan == "" {
		plan = TenantPlanFree
	}

	featureFlags := DefaultFeatureFlags(plan)
	flagsJSON, _ := json.Marshal(featureFlags)

	limits := DefaultLimits(plan)
	limitsJSON, _ := json.Marshal(limits)

	now := time.Now().UTC()
	return &Tenant{
		ID:           uuid.New(),
		Name:         name,
		Slug:         slug,
		Status:       TenantStatusActive,
		Plan:         plan,
		Settings:     json.RawMessage("{}"),
		Metadata:     json.RawMessage("{}"),
		FeatureFlags: flagsJSON,
		Limits:       limitsJSON,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// GetFeatureFlags parses and returns feature flags
func (t *Tenant) GetFeatureFlags() (*TenantFeatureFlags, error) {
	var flags TenantFeatureFlags
	if err := json.Unmarshal(t.FeatureFlags, &flags); err != nil {
		return nil, err
	}
	return &flags, nil
}

// GetLimits parses and returns limits
func (t *Tenant) GetLimits() (*TenantLimits, error) {
	var limits TenantLimits
	if err := json.Unmarshal(t.Limits, &limits); err != nil {
		return nil, err
	}
	return &limits, nil
}

// GetMetadata parses and returns metadata
func (t *Tenant) GetMetadata() (*TenantMetadata, error) {
	var metadata TenantMetadata
	if err := json.Unmarshal(t.Metadata, &metadata); err != nil {
		return nil, err
	}
	return &metadata, nil
}

// Suspend marks the tenant as suspended
func (t *Tenant) Suspend(reason string) {
	now := time.Now().UTC()
	t.Status = TenantStatusSuspended
	t.SuspendedAt = &now
	t.SuspendedReason = reason
	t.UpdatedAt = now
}

// Activate marks the tenant as active
func (t *Tenant) Activate() {
	t.Status = TenantStatusActive
	t.SuspendedAt = nil
	t.SuspendedReason = ""
	t.UpdatedAt = time.Now().UTC()
}

// IsActive returns true if the tenant is active
func (t *Tenant) IsActive() bool {
	return t.Status == TenantStatusActive
}

// IsSuspended returns true if the tenant is suspended
func (t *Tenant) IsSuspended() bool {
	return t.Status == TenantStatusSuspended
}

// TenantStats contains aggregated statistics for a tenant
type TenantStats struct {
	WorkflowCount      int     `json:"workflow_count"`
	PublishedWorkflows int     `json:"published_workflows"`
	RunCount           int     `json:"run_count"`
	RunsThisMonth      int     `json:"runs_this_month"`
	UserCount          int     `json:"user_count"`
	CredentialCount    int     `json:"credential_count"`
	TotalCostUSD       float64 `json:"total_cost_usd"`
	CostThisMonth      float64 `json:"cost_this_month"`
}
