-- Migration: 010_add_usage_records.sql
-- Purpose: Add tables for LLM API usage tracking and cost management

-- usage_records: Individual API call records
CREATE TABLE usage_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    workflow_id UUID REFERENCES workflows(id),
    run_id UUID REFERENCES runs(id),
    step_run_id UUID REFERENCES step_runs(id),

    -- Provider information
    provider VARCHAR(50) NOT NULL,      -- openai, anthropic, google, etc.
    model VARCHAR(100) NOT NULL,        -- gpt-4o, claude-3-opus, etc.
    operation VARCHAR(50) NOT NULL,     -- chat, completion, embedding, etc.

    -- Token usage
    input_tokens INT NOT NULL DEFAULT 0,
    output_tokens INT NOT NULL DEFAULT 0,
    total_tokens INT NOT NULL DEFAULT 0,

    -- Cost in USD (8 decimal places for precision)
    input_cost_usd DECIMAL(12, 8) NOT NULL DEFAULT 0,
    output_cost_usd DECIMAL(12, 8) NOT NULL DEFAULT 0,
    total_cost_usd DECIMAL(12, 8) NOT NULL DEFAULT 0,

    -- Metadata
    latency_ms INT,
    success BOOLEAN NOT NULL DEFAULT TRUE,
    error_message TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX idx_usage_records_tenant_date ON usage_records(tenant_id, created_at);
CREATE INDEX idx_usage_records_workflow ON usage_records(workflow_id);
CREATE INDEX idx_usage_records_run ON usage_records(run_id);
CREATE INDEX idx_usage_records_provider_model ON usage_records(provider, model);
CREATE INDEX idx_usage_records_created_at ON usage_records(created_at);

-- usage_daily_aggregates: Pre-aggregated daily data for performance
CREATE TABLE usage_daily_aggregates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    workflow_id UUID REFERENCES workflows(id),
    date DATE NOT NULL,
    provider VARCHAR(50) NOT NULL,
    model VARCHAR(100) NOT NULL,

    -- Aggregated metrics
    total_requests INT NOT NULL DEFAULT 0,
    successful_requests INT NOT NULL DEFAULT 0,
    failed_requests INT NOT NULL DEFAULT 0,
    total_input_tokens BIGINT NOT NULL DEFAULT 0,
    total_output_tokens BIGINT NOT NULL DEFAULT 0,
    total_cost_usd DECIMAL(12, 6) NOT NULL DEFAULT 0,
    avg_latency_ms INT,
    min_latency_ms INT,
    max_latency_ms INT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(tenant_id, COALESCE(workflow_id, '00000000-0000-0000-0000-000000000000'::uuid), date, provider, model)
);

CREATE INDEX idx_usage_daily_tenant_date ON usage_daily_aggregates(tenant_id, date);
CREATE INDEX idx_usage_daily_workflow ON usage_daily_aggregates(workflow_id);

-- usage_budgets: Budget settings for cost control
CREATE TABLE usage_budgets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    workflow_id UUID REFERENCES workflows(id),  -- NULL means tenant-wide budget

    -- Budget configuration
    budget_type VARCHAR(50) NOT NULL,           -- daily, monthly
    budget_amount_usd DECIMAL(12, 2) NOT NULL,
    alert_threshold DECIMAL(3, 2) NOT NULL DEFAULT 0.80,  -- Alert at 80% by default
    enabled BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Ensure only one budget per type per scope
    UNIQUE(tenant_id, COALESCE(workflow_id, '00000000-0000-0000-0000-000000000000'::uuid), budget_type)
);

CREATE INDEX idx_usage_budgets_tenant ON usage_budgets(tenant_id);
CREATE INDEX idx_usage_budgets_workflow ON usage_budgets(workflow_id);

-- Comments for documentation
COMMENT ON TABLE usage_records IS 'Individual LLM API call records with token usage and cost';
COMMENT ON TABLE usage_daily_aggregates IS 'Pre-aggregated daily usage data for dashboard performance';
COMMENT ON TABLE usage_budgets IS 'Budget settings for cost control and alerts';

COMMENT ON COLUMN usage_records.provider IS 'LLM provider: openai, anthropic, google, etc.';
COMMENT ON COLUMN usage_records.model IS 'Model identifier: gpt-4o, claude-3-opus, etc.';
COMMENT ON COLUMN usage_records.operation IS 'Operation type: chat, completion, embedding, etc.';
COMMENT ON COLUMN usage_records.total_cost_usd IS 'Total cost in USD with 8 decimal precision';
COMMENT ON COLUMN usage_budgets.alert_threshold IS 'Percentage (0.00-1.00) at which to trigger alert';
