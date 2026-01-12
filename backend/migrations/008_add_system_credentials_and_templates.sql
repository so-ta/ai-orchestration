-- Migration: Add system credentials, block templates, and credential binding support
-- This enables:
--   1. Operator-managed system credentials (separate from tenant credentials)
--   2. Block templates for common patterns (http_api, graphql, llm, etc.)
--   3. Credential requirements declaration in block definitions
--   4. Credential binding in workflow steps

-- ============================================================================
-- 1. System Credentials (Operator-managed, used by system blocks)
-- ============================================================================
CREATE TABLE IF NOT EXISTS system_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL UNIQUE,
    description TEXT,

    -- Credential type (oauth2, api_key, basic, bearer, custom)
    credential_type VARCHAR(50) NOT NULL,

    -- Encrypted credential data (AES-256-GCM)
    encrypted_data BYTEA NOT NULL,
    encrypted_dek BYTEA NOT NULL,
    data_nonce BYTEA NOT NULL,
    dek_nonce BYTEA NOT NULL,

    -- Non-sensitive metadata
    metadata JSONB NOT NULL DEFAULT '{}',

    -- Token expiration (for OAuth2)
    expires_at TIMESTAMPTZ,

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'active',

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- Valid credential types
    CONSTRAINT valid_system_credential_type CHECK (credential_type IN ('oauth2', 'api_key', 'basic', 'bearer', 'custom')),

    -- Valid status values
    CONSTRAINT valid_system_credential_status CHECK (status IN ('active', 'expired', 'revoked', 'error'))
);

-- Indexes for system_credentials
CREATE INDEX idx_system_credentials_type ON system_credentials(credential_type);
CREATE INDEX idx_system_credentials_status ON system_credentials(status);

-- Trigger for updated_at
CREATE OR REPLACE FUNCTION update_system_credentials_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_system_credentials_updated_at
    BEFORE UPDATE ON system_credentials
    FOR EACH ROW
    EXECUTE FUNCTION update_system_credentials_updated_at();

-- ============================================================================
-- 2. Block Templates (Reusable patterns for block definitions)
-- ============================================================================
CREATE TABLE IF NOT EXISTS block_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(200) NOT NULL,
    description TEXT,

    -- Template configuration schema (what users configure when using this template)
    config_schema JSONB NOT NULL DEFAULT '{}',

    -- Template executor code (Go code reference or JavaScript)
    executor_type VARCHAR(20) NOT NULL DEFAULT 'builtin',  -- builtin, javascript
    executor_code TEXT,  -- For javascript templates

    -- Whether this is a built-in template (cannot be deleted)
    is_builtin BOOLEAN NOT NULL DEFAULT false,

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert built-in templates
INSERT INTO block_templates (slug, name, description, config_schema, executor_type, is_builtin) VALUES
(
    'http_api',
    'HTTP API Call',
    'REST API call template with configurable method, URL, headers, and body',
    '{
        "type": "object",
        "properties": {
            "method": {"type": "string", "enum": ["GET", "POST", "PUT", "PATCH", "DELETE"], "default": "GET"},
            "url": {"type": "string", "description": "URL with {{variable}} placeholders"},
            "headers": {"type": "object", "additionalProperties": {"type": "string"}},
            "body": {"type": "object", "description": "Request body (for POST/PUT/PATCH)"},
            "timeout_ms": {"type": "integer", "default": 30000}
        },
        "required": ["method", "url"]
    }',
    'builtin',
    true
),
(
    'graphql',
    'GraphQL Query',
    'GraphQL query/mutation template',
    '{
        "type": "object",
        "properties": {
            "endpoint": {"type": "string"},
            "query": {"type": "string"},
            "variables": {"type": "object"},
            "operation_name": {"type": "string"}
        },
        "required": ["endpoint", "query"]
    }',
    'builtin',
    true
),
(
    'transform',
    'Data Transform',
    'Transform data using JavaScript expression or JQ-like syntax',
    '{
        "type": "object",
        "properties": {
            "expression": {"type": "string", "description": "JavaScript expression or JQ path"},
            "mode": {"type": "string", "enum": ["javascript", "jq"], "default": "javascript"}
        },
        "required": ["expression"]
    }',
    'builtin',
    true
),
(
    'llm_call',
    'LLM API Call',
    'Call LLM provider API (OpenAI, Anthropic, etc.)',
    '{
        "type": "object",
        "properties": {
            "provider": {"type": "string", "enum": ["openai", "anthropic", "custom"]},
            "model": {"type": "string"},
            "system_prompt": {"type": "string"},
            "user_prompt_template": {"type": "string"},
            "temperature": {"type": "number", "minimum": 0, "maximum": 2},
            "max_tokens": {"type": "integer"}
        },
        "required": ["provider", "model"]
    }',
    'builtin',
    true
);

-- ============================================================================
-- 3. Update block_definitions - Add credential requirements and template support
-- ============================================================================

-- Add template reference (for template-based blocks)
ALTER TABLE block_definitions
ADD COLUMN IF NOT EXISTS template_id UUID REFERENCES block_templates(id) ON DELETE SET NULL;

-- Add template configuration (when using a template)
ALTER TABLE block_definitions
ADD COLUMN IF NOT EXISTS template_config JSONB;

-- Add custom code (for code-based blocks, hidden for system blocks)
ALTER TABLE block_definitions
ADD COLUMN IF NOT EXISTS custom_code TEXT;

-- Add required credentials declaration
-- Format: [{"name": "api_key", "type": "api_key", "scope": "system|tenant", "description": "...", "required": true}]
ALTER TABLE block_definitions
ADD COLUMN IF NOT EXISTS required_credentials JSONB DEFAULT '[]';

-- Add visibility flag for tenant blocks (whether other tenants can see/use)
-- System blocks (tenant_id = NULL) are always visible to all
ALTER TABLE block_definitions
ADD COLUMN IF NOT EXISTS is_public BOOLEAN DEFAULT false;

-- Remove old default_credential_id column (replaced by required_credentials)
ALTER TABLE block_definitions
DROP COLUMN IF EXISTS default_credential_id;

-- ============================================================================
-- 4. Update steps - Add credential bindings
-- ============================================================================

-- Add credential bindings for steps
-- Format: {"credential_name": "uuid-of-tenant-credential", ...}
ALTER TABLE steps
ADD COLUMN IF NOT EXISTS credential_bindings JSONB DEFAULT '{}';

-- Add block_definition_id reference (optional, for registry-based blocks)
ALTER TABLE steps
ADD COLUMN IF NOT EXISTS block_definition_id UUID REFERENCES block_definitions(id) ON DELETE SET NULL;

-- ============================================================================
-- 5. Comments for documentation
-- ============================================================================
COMMENT ON TABLE system_credentials IS 'Operator-managed credentials for system blocks (not accessible by tenants)';
COMMENT ON TABLE block_templates IS 'Reusable block templates (http_api, graphql, transform, etc.)';
COMMENT ON COLUMN block_definitions.required_credentials IS 'JSON array declaring required credentials: [{name, type, scope, description, required}]';
COMMENT ON COLUMN block_definitions.template_id IS 'Reference to block_templates for template-based blocks';
COMMENT ON COLUMN block_definitions.custom_code IS 'Custom JavaScript code for code-based blocks (hidden in system blocks)';
COMMENT ON COLUMN block_definitions.is_public IS 'Whether tenant block is visible to other tenants';
COMMENT ON COLUMN steps.credential_bindings IS 'Mapping of credential names to tenant credential IDs';
COMMENT ON COLUMN steps.block_definition_id IS 'Reference to block_definitions registry';

-- ============================================================================
-- 6. Update existing system blocks with required_credentials
-- ============================================================================

-- LLM block needs API key
UPDATE block_definitions
SET required_credentials = '[
    {"name": "llm_api_key", "type": "api_key", "scope": "system", "description": "LLM Provider API Key", "required": true}
]'::jsonb
WHERE slug = 'llm' AND tenant_id IS NULL;

-- Router block needs API key (uses LLM for routing)
UPDATE block_definitions
SET required_credentials = '[
    {"name": "llm_api_key", "type": "api_key", "scope": "system", "description": "LLM Provider API Key", "required": true}
]'::jsonb
WHERE slug = 'router' AND tenant_id IS NULL;

-- Tool block - credential depends on adapter, so leave empty (handled by adapter)
UPDATE block_definitions
SET required_credentials = '[]'::jsonb
WHERE slug = 'tool' AND tenant_id IS NULL;
