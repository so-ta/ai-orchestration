-- AI Orchestration - Initial Database Schema
-- This file is executed on first PostgreSQL startup

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Tenants table
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Users table (synced from Keycloak)
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    role VARCHAR(50) NOT NULL DEFAULT 'viewer',
    last_login_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, email)
);

-- Workflows table
CREATE TABLE IF NOT EXISTS workflows (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    version INTEGER NOT NULL DEFAULT 0,
    input_schema JSONB,
    output_schema JSONB,
    draft JSONB,
    created_by UUID REFERENCES users(id),
    published_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Workflow versions (immutable snapshots created on each Save)
CREATE TABLE IF NOT EXISTS workflow_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_id UUID NOT NULL REFERENCES workflows(id),
    version INTEGER NOT NULL,
    definition JSONB NOT NULL,
    saved_by UUID REFERENCES users(id),
    saved_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(workflow_id, version)
);

-- Steps table
CREATE TABLE IF NOT EXISTS steps (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    config JSONB NOT NULL DEFAULT '{}',
    position_x INTEGER DEFAULT 0,
    position_y INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Edges table (connections between steps)
CREATE TABLE IF NOT EXISTS edges (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    source_step_id UUID NOT NULL REFERENCES steps(id) ON DELETE CASCADE,
    target_step_id UUID NOT NULL REFERENCES steps(id) ON DELETE CASCADE,
    condition TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(source_step_id, target_step_id)
);

-- Runs table
CREATE TABLE IF NOT EXISTS runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    workflow_id UUID NOT NULL REFERENCES workflows(id),
    workflow_version INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    mode VARCHAR(50) NOT NULL DEFAULT 'production',
    input JSONB,
    output JSONB,
    error TEXT,
    triggered_by VARCHAR(50) NOT NULL DEFAULT 'manual',
    triggered_by_user UUID REFERENCES users(id),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Step runs table
CREATE TABLE IF NOT EXISTS step_runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    run_id UUID NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    step_id UUID NOT NULL,
    step_name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    attempt INTEGER NOT NULL DEFAULT 1,
    input JSONB,
    output JSONB,
    error TEXT,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Schedules table
CREATE TABLE IF NOT EXISTS schedules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    workflow_id UUID NOT NULL REFERENCES workflows(id),
    workflow_version INTEGER NOT NULL DEFAULT 1,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    cron_expression VARCHAR(100) NOT NULL,
    timezone VARCHAR(50) NOT NULL DEFAULT 'UTC',
    input JSONB,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    next_run_at TIMESTAMP WITH TIME ZONE,
    last_run_at TIMESTAMP WITH TIME ZONE,
    last_run_id UUID REFERENCES runs(id),
    run_count INTEGER NOT NULL DEFAULT 0,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Webhooks table
CREATE TABLE IF NOT EXISTS webhooks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    workflow_id UUID NOT NULL REFERENCES workflows(id),
    workflow_version INTEGER NOT NULL DEFAULT 0, -- 0 means latest
    name VARCHAR(255) NOT NULL,
    description TEXT,
    secret VARCHAR(255) NOT NULL,
    input_mapping JSONB,
    enabled BOOLEAN NOT NULL DEFAULT true,
    last_triggered_at TIMESTAMP WITH TIME ZONE,
    trigger_count INTEGER NOT NULL DEFAULT 0,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Adapters table
CREATE TABLE IF NOT EXISTS adapters (
    id VARCHAR(100) PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL,
    config JSONB,
    input_schema JSONB,
    output_schema JSONB,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Secrets table
CREATE TABLE IF NOT EXISTS secrets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    encrypted_value TEXT NOT NULL,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

-- Audit logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    actor_id UUID,
    actor_email VARCHAR(255),
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_id UUID,
    metadata JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_workflows_tenant ON workflows(tenant_id);
CREATE INDEX idx_workflows_status ON workflows(status);
CREATE INDEX idx_runs_tenant ON runs(tenant_id);
CREATE INDEX idx_runs_workflow ON runs(workflow_id);
CREATE INDEX idx_runs_status ON runs(status);
CREATE INDEX idx_step_runs_run ON step_runs(run_id);
CREATE INDEX idx_schedules_tenant ON schedules(tenant_id);
CREATE INDEX idx_schedules_next_run ON schedules(next_run_at) WHERE status = 'active';
CREATE INDEX idx_audit_logs_tenant ON audit_logs(tenant_id);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at);

-- Insert default tenant
INSERT INTO tenants (id, name, slug, settings)
VALUES ('00000000-0000-0000-0000-000000000001', 'Default Tenant', 'default-tenant', '{"data_retention_days": 30}')
ON CONFLICT (slug) DO NOTHING;

-- Insert mock adapter
INSERT INTO adapters (id, name, description, type, config, input_schema, output_schema)
VALUES (
    'mock',
    'Mock Adapter',
    'Mock adapter for testing and development',
    'builtin',
    '{"delay_ms": 100, "success_rate": 0.95}',
    '{"type": "object", "properties": {"message": {"type": "string"}}}',
    '{"type": "object", "properties": {"result": {"type": "string"}}}'
)
ON CONFLICT (id) DO NOTHING;
