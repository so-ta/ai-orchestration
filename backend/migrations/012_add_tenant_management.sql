-- Migration: 012_add_tenant_management.sql
-- Purpose: Extend tenants table for comprehensive tenant management

-- Add status column (active, suspended, pending, inactive)
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS status VARCHAR(50) NOT NULL DEFAULT 'active';

-- Add plan column (free, starter, professional, enterprise)
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS plan VARCHAR(50) NOT NULL DEFAULT 'free';

-- Add owner information
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS owner_email VARCHAR(255);
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS owner_name VARCHAR(255);
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS billing_email VARCHAR(255);

-- Add metadata (industry, company size, notes, etc.)
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}';

-- Add feature flags (copilot_enabled, custom_blocks, sso_enabled, etc.)
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS feature_flags JSONB DEFAULT '{}';

-- Add resource limits (max_workflows, max_runs_per_day, max_users, etc.)
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS limits JSONB DEFAULT '{}';

-- Add suspension tracking
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS suspended_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS suspended_reason TEXT;

-- Create indexes for filtering
CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenants(status);
CREATE INDEX IF NOT EXISTS idx_tenants_plan ON tenants(plan);
CREATE INDEX IF NOT EXISTS idx_tenants_owner_email ON tenants(owner_email);

-- Add comments for documentation
COMMENT ON COLUMN tenants.status IS 'Tenant status: active, suspended, pending, inactive';
COMMENT ON COLUMN tenants.plan IS 'Subscription plan: free, starter, professional, enterprise';
COMMENT ON COLUMN tenants.owner_email IS 'Primary contact email for the tenant';
COMMENT ON COLUMN tenants.owner_name IS 'Primary contact name for the tenant';
COMMENT ON COLUMN tenants.billing_email IS 'Email for billing notifications';
COMMENT ON COLUMN tenants.metadata IS 'Additional tenant metadata (industry, company_size, website, country, notes)';
COMMENT ON COLUMN tenants.feature_flags IS 'Feature flags: copilot_enabled, advanced_analytics, custom_blocks, api_access, sso_enabled, audit_logs, max_concurrent_runs';
COMMENT ON COLUMN tenants.limits IS 'Resource limits: max_workflows, max_runs_per_day, max_users, max_credentials, max_storage_mb, retention_days';
COMMENT ON COLUMN tenants.suspended_at IS 'Timestamp when tenant was suspended';
COMMENT ON COLUMN tenants.suspended_reason IS 'Reason for tenant suspension';

-- Update default tenant with default feature flags and limits
UPDATE tenants
SET
    feature_flags = '{"copilot_enabled": true, "advanced_analytics": true, "custom_blocks": true, "api_access": true, "sso_enabled": false, "audit_logs": true, "max_concurrent_runs": 10}'::jsonb,
    limits = '{"max_workflows": 100, "max_runs_per_day": 1000, "max_users": 50, "max_credentials": 100, "max_storage_mb": 10240, "retention_days": 90}'::jsonb
WHERE id = '00000000-0000-0000-0000-000000000001'
  AND feature_flags = '{}'::jsonb;
