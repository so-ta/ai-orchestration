-- Migration: Add block_definitions table for Block Registry
-- This table stores both built-in and custom block definitions

CREATE TABLE IF NOT EXISTS block_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,  -- NULL = system block
    slug VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(50) NOT NULL,
    icon VARCHAR(50),

    -- JSON Schemas for configuration and data
    config_schema JSONB NOT NULL DEFAULT '{}',
    input_schema JSONB,
    output_schema JSONB,

    -- Executor configuration
    executor_type VARCHAR(20) NOT NULL DEFAULT 'builtin',  -- builtin, http, function
    executor_config JSONB,

    -- Error code definitions
    error_codes JSONB DEFAULT '[]',

    -- Metadata
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- Unique constraint: slug must be unique per tenant (NULL tenant = global/system)
    CONSTRAINT unique_block_slug UNIQUE NULLS NOT DISTINCT (tenant_id, slug),

    -- Valid category values
    CONSTRAINT valid_block_category CHECK (category IN ('ai', 'logic', 'integration', 'data', 'control', 'utility')),

    -- Valid executor types
    CONSTRAINT valid_executor_type CHECK (executor_type IN ('builtin', 'http', 'function'))
);

-- Indexes
CREATE INDEX idx_block_definitions_tenant ON block_definitions(tenant_id);
CREATE INDEX idx_block_definitions_category ON block_definitions(category);
CREATE INDEX idx_block_definitions_enabled ON block_definitions(enabled) WHERE enabled = true;
CREATE INDEX idx_block_definitions_slug ON block_definitions(slug);

-- Insert built-in block definitions
INSERT INTO block_definitions (tenant_id, slug, name, description, category, icon, executor_type, config_schema, error_codes) VALUES
-- Control blocks
(NULL, 'start', 'Start', 'Workflow entry point', 'control', 'play', 'builtin', '{}', '[]'),

-- AI blocks
(NULL, 'llm', 'LLM', 'Execute LLM prompts with various providers', 'ai', 'brain', 'builtin',
    '{"type":"object","properties":{"provider":{"type":"string","enum":["openai","anthropic"]},"model":{"type":"string"},"system_prompt":{"type":"string"},"user_prompt":{"type":"string"},"temperature":{"type":"number","minimum":0,"maximum":2},"max_tokens":{"type":"integer","minimum":1}}}',
    '[{"code":"LLM_001","name":"RATE_LIMIT","description":"Rate limit exceeded","retryable":true},{"code":"LLM_002","name":"INVALID_MODEL","description":"Invalid model specified","retryable":false},{"code":"LLM_003","name":"TOKEN_LIMIT","description":"Token limit exceeded","retryable":false},{"code":"LLM_004","name":"API_ERROR","description":"LLM API error","retryable":true}]'
),

(NULL, 'router', 'Router', 'AI-driven dynamic routing', 'ai', 'git-branch', 'builtin',
    '{"type":"object","properties":{"routes":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"description":{"type":"string"}}}},"provider":{"type":"string"},"model":{"type":"string"}}}',
    '[{"code":"ROUTER_001","name":"NO_MATCH","description":"No matching route found","retryable":false}]'
),

-- Logic blocks
(NULL, 'condition', 'Condition', 'Branch based on expression', 'logic', 'git-branch', 'builtin',
    '{"type":"object","properties":{"expression":{"type":"string","description":"JSONPath expression"}}}',
    '[{"code":"COND_001","name":"INVALID_EXPR","description":"Invalid condition expression","retryable":false},{"code":"COND_002","name":"EVAL_ERROR","description":"Expression evaluation error","retryable":false}]'
),

(NULL, 'switch', 'Switch', 'Multi-branch routing', 'logic', 'shuffle', 'builtin',
    '{"type":"object","properties":{"mode":{"type":"string","enum":["rules","expression"]},"cases":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"expression":{"type":"string"},"is_default":{"type":"boolean"}}}}}}',
    '[{"code":"SWITCH_001","name":"NO_MATCH","description":"No matching case","retryable":false}]'
),

(NULL, 'loop', 'Loop', 'Iterate with for/forEach/while', 'logic', 'repeat', 'builtin',
    '{"type":"object","properties":{"loop_type":{"type":"string","enum":["for","forEach","while","doWhile"]},"count":{"type":"integer"},"input_path":{"type":"string"},"condition":{"type":"string"},"max_iterations":{"type":"integer"}}}',
    '[{"code":"LOOP_001","name":"MAX_ITERATIONS","description":"Maximum iterations exceeded","retryable":false}]'
),

-- Data blocks
(NULL, 'map', 'Map', 'Process array items in parallel', 'data', 'layers', 'builtin',
    '{"type":"object","properties":{"input_path":{"type":"string"},"parallel":{"type":"boolean"},"max_workers":{"type":"integer"}}}',
    '[{"code":"MAP_001","name":"INVALID_PATH","description":"Invalid input path","retryable":false}]'
),

(NULL, 'join', 'Join', 'Merge multiple branches', 'data', 'git-merge', 'builtin', '{}', '[]'),

(NULL, 'filter', 'Filter', 'Filter items by condition', 'data', 'filter', 'builtin',
    '{"type":"object","properties":{"expression":{"type":"string"},"keep_all":{"type":"boolean"}}}',
    '[{"code":"FILTER_001","name":"INVALID_EXPR","description":"Invalid filter expression","retryable":false}]'
),

(NULL, 'split', 'Split', 'Split into batches', 'data', 'scissors', 'builtin',
    '{"type":"object","properties":{"batch_size":{"type":"integer","minimum":1},"input_path":{"type":"string"}}}',
    '[]'
),

(NULL, 'aggregate', 'Aggregate', 'Aggregate data operations', 'data', 'database', 'builtin',
    '{"type":"object","properties":{"group_by":{"type":"string"},"operations":{"type":"array","items":{"type":"object","properties":{"operation":{"type":"string","enum":["sum","count","avg","min","max","first","last","concat"]},"field":{"type":"string"},"output_field":{"type":"string"}}}}}}',
    '[]'
),

-- Integration blocks
(NULL, 'tool', 'Tool', 'Execute external tool/adapter', 'integration', 'wrench', 'builtin',
    '{"type":"object","properties":{"adapter_id":{"type":"string"}}}',
    '[{"code":"TOOL_001","name":"ADAPTER_NOT_FOUND","description":"Adapter not found","retryable":false},{"code":"TOOL_002","name":"EXEC_ERROR","description":"Tool execution error","retryable":true}]'
),

(NULL, 'subflow', 'Subflow', 'Execute another workflow', 'integration', 'workflow', 'builtin',
    '{"type":"object","properties":{"workflow_id":{"type":"string","format":"uuid"},"workflow_version":{"type":"integer"}}}',
    '[{"code":"SUBFLOW_001","name":"NOT_FOUND","description":"Subflow workflow not found","retryable":false},{"code":"SUBFLOW_002","name":"EXEC_ERROR","description":"Subflow execution error","retryable":true}]'
),

-- Control blocks (continued)
(NULL, 'wait', 'Wait', 'Pause execution', 'control', 'clock', 'builtin',
    '{"type":"object","properties":{"duration_ms":{"type":"integer","minimum":0},"until":{"type":"string","format":"date-time"}}}',
    '[]'
),

(NULL, 'human_in_loop', 'Human in Loop', 'Wait for human approval', 'control', 'user-check', 'builtin',
    '{"type":"object","properties":{"instructions":{"type":"string"},"timeout_hours":{"type":"integer"},"approval_url":{"type":"boolean"}}}',
    '[{"code":"HIL_001","name":"TIMEOUT","description":"Human approval timeout","retryable":false},{"code":"HIL_002","name":"REJECTED","description":"Human rejected","retryable":false}]'
),

(NULL, 'error', 'Error', 'Stop workflow with error', 'control', 'alert-circle', 'builtin',
    '{"type":"object","properties":{"error_type":{"type":"string"},"error_message":{"type":"string"},"error_code":{"type":"string"}}}',
    '[]'
),

-- Utility blocks
(NULL, 'function', 'Function', 'Execute custom JavaScript', 'utility', 'code', 'builtin',
    '{"type":"object","properties":{"code":{"type":"string"},"language":{"type":"string","enum":["javascript"]},"timeout_ms":{"type":"integer"}}}',
    '[{"code":"FUNC_001","name":"SYNTAX_ERROR","description":"JavaScript syntax error","retryable":false},{"code":"FUNC_002","name":"RUNTIME_ERROR","description":"JavaScript runtime error","retryable":false},{"code":"FUNC_003","name":"TIMEOUT","description":"Function execution timeout","retryable":false}]'
),

(NULL, 'note', 'Note', 'Documentation/comment', 'utility', 'file-text', 'builtin',
    '{"type":"object","properties":{"content":{"type":"string"},"color":{"type":"string"}}}',
    '[]'
);

-- Add trigger for updated_at
CREATE OR REPLACE FUNCTION update_block_definitions_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_block_definitions_updated_at
    BEFORE UPDATE ON block_definitions
    FOR EACH ROW
    EXECUTE FUNCTION update_block_definitions_updated_at();
