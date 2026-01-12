-- Migration: 011_unified_block_model.sql
-- Purpose: Extend block_definitions for Unified Block Model (code-based execution)
-- See: docs/designs/UNIFIED_BLOCK_MODEL.md

-- Add new columns for code-based execution
ALTER TABLE block_definitions
    ADD COLUMN IF NOT EXISTS code TEXT,
    ADD COLUMN IF NOT EXISTS ui_config JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS is_system BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS version INTEGER NOT NULL DEFAULT 1;

-- Add comments
COMMENT ON COLUMN block_definitions.code IS 'JavaScript code executed in sandbox. All blocks are code-based.';
COMMENT ON COLUMN block_definitions.ui_config IS 'UI metadata: icon, color, configSchema for workflow editor';
COMMENT ON COLUMN block_definitions.is_system IS 'System blocks can only be edited by admins';
COMMENT ON COLUMN block_definitions.version IS 'Version number, incremented on each update';

-- Create block_versions table for version history
CREATE TABLE IF NOT EXISTS block_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    block_id UUID NOT NULL REFERENCES block_definitions(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,

    -- Snapshot of block at this version
    code TEXT NOT NULL,
    config_schema JSONB NOT NULL,
    input_schema JSONB,
    output_schema JSONB,
    ui_config JSONB NOT NULL,

    -- Change tracking
    change_summary TEXT,
    changed_by UUID,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(block_id, version)
);

CREATE INDEX idx_block_versions_block_id ON block_versions(block_id);
CREATE INDEX idx_block_versions_created_at ON block_versions(created_at);

COMMENT ON TABLE block_versions IS 'Version history for block definitions, enables rollback';

-- Update existing system blocks with code
-- start block
UPDATE block_definitions SET
    code = 'return input;',
    ui_config = '{"icon": "play", "color": "#10B981"}',
    is_system = TRUE
WHERE slug = 'start' AND tenant_id IS NULL;

-- llm block
UPDATE block_definitions SET
    code = $code$
const prompt = renderTemplate(config.user_prompt || '', input);
const systemPrompt = config.system_prompt || '';

const response = await ctx.llm.chat(config.provider, config.model, {
    messages: [
        ...(systemPrompt ? [{ role: 'system', content: systemPrompt }] : []),
        { role: 'user', content: prompt }
    ],
    temperature: config.temperature ?? 0.7,
    maxTokens: config.max_tokens ?? 1000
});

return {
    content: response.content,
    usage: response.usage
};
$code$,
    ui_config = '{"icon": "brain", "color": "#8B5CF6"}',
    is_system = TRUE
WHERE slug = 'llm' AND tenant_id IS NULL;

-- condition block
UPDATE block_definitions SET
    code = $code$
const result = evaluate(config.expression, input);
return {
    ...input,
    __branch: result ? 'then' : 'else'
};
$code$,
    ui_config = '{"icon": "git-branch", "color": "#F59E0B"}',
    is_system = TRUE
WHERE slug = 'condition' AND tenant_id IS NULL;

-- switch block
UPDATE block_definitions SET
    code = $code$
let matchedCase = null;

for (const c of config.cases || []) {
    if (c.is_default) {
        matchedCase = matchedCase || c.name;
        continue;
    }
    if (evaluate(c.expression, input)) {
        matchedCase = c.name;
        break;
    }
}

return {
    ...input,
    __branch: matchedCase || 'default'
};
$code$,
    ui_config = '{"icon": "shuffle", "color": "#F59E0B"}',
    is_system = TRUE
WHERE slug = 'switch' AND tenant_id IS NULL;

-- loop block
UPDATE block_definitions SET
    code = $code$
const results = [];
const maxIter = config.max_iterations || 100;

if (config.loop_type === 'for') {
    for (let i = 0; i < (config.count || 0) && i < maxIter; i++) {
        results.push({ index: i, ...input });
    }
} else if (config.loop_type === 'forEach') {
    const items = getPath(input, config.input_path) || [];
    for (let i = 0; i < items.length && i < maxIter; i++) {
        results.push({ item: items[i], index: i, ...input });
    }
} else if (config.loop_type === 'while') {
    let i = 0;
    while (evaluate(config.condition, input) && i < maxIter) {
        results.push({ index: i, ...input });
        i++;
    }
}

return { ...input, results, iterations: results.length };
$code$,
    ui_config = '{"icon": "repeat", "color": "#F59E0B"}',
    is_system = TRUE
WHERE slug = 'loop' AND tenant_id IS NULL;

-- map block
UPDATE block_definitions SET
    code = $code$
const items = getPath(input, config.input_path) || [];
const maxWorkers = config.max_workers || 10;

let results;
if (config.parallel) {
    results = await Promise.all(
        items.map((item, index) => ({ item, index, processed: true }))
    );
} else {
    results = items.map((item, index) => ({ item, index, processed: true }));
}

return {
    ...input,
    items: results,
    count: results.length,
    success_count: results.length,
    error_count: 0
};
$code$,
    ui_config = '{"icon": "layers", "color": "#06B6D4"}',
    is_system = TRUE
WHERE slug = 'map' AND tenant_id IS NULL;

-- join block
UPDATE block_definitions SET
    code = 'return input;',
    ui_config = '{"icon": "git-merge", "color": "#06B6D4"}',
    is_system = TRUE
WHERE slug = 'join' AND tenant_id IS NULL;

-- filter block
UPDATE block_definitions SET
    code = $code$
const items = Array.isArray(input) ? input : (input.items || []);
const filtered = items.filter(item => evaluate(config.expression, item));

return {
    items: filtered,
    original_count: items.length,
    filtered_count: filtered.length,
    removed_count: items.length - filtered.length
};
$code$,
    ui_config = '{"icon": "filter", "color": "#06B6D4"}',
    is_system = TRUE
WHERE slug = 'filter' AND tenant_id IS NULL;

-- split block
UPDATE block_definitions SET
    code = $code$
const items = getPath(input, config.input_path) || [];
const batchSize = config.batch_size || 10;
const batches = [];

for (let i = 0; i < items.length; i += batchSize) {
    batches.push(items.slice(i, i + batchSize));
}

return {
    ...input,
    batches,
    batch_count: batches.length,
    total_items: items.length
};
$code$,
    ui_config = '{"icon": "scissors", "color": "#06B6D4"}',
    is_system = TRUE
WHERE slug = 'split' AND tenant_id IS NULL;

-- aggregate block
UPDATE block_definitions SET
    code = $code$
const items = Array.isArray(input) ? input : (input.items || []);
const result = {};

for (const op of config.operations || []) {
    const values = items.map(item => getPath(item, op.field));

    switch (op.operation) {
        case 'sum': result[op.output_field] = values.reduce((a, b) => a + b, 0); break;
        case 'count': result[op.output_field] = values.length; break;
        case 'avg': result[op.output_field] = values.reduce((a, b) => a + b, 0) / values.length; break;
        case 'min': result[op.output_field] = Math.min(...values); break;
        case 'max': result[op.output_field] = Math.max(...values); break;
        case 'first': result[op.output_field] = values[0]; break;
        case 'last': result[op.output_field] = values[values.length - 1]; break;
        case 'concat': result[op.output_field] = values.join(''); break;
    }
}

return result;
$code$,
    ui_config = '{"icon": "database", "color": "#06B6D4"}',
    is_system = TRUE
WHERE slug = 'aggregate' AND tenant_id IS NULL;

-- tool block
UPDATE block_definitions SET
    code = 'return await ctx.adapter.call(config.adapter_id, input);',
    ui_config = '{"icon": "wrench", "color": "#10B981"}',
    is_system = TRUE
WHERE slug = 'tool' AND tenant_id IS NULL;

-- subflow block
UPDATE block_definitions SET
    code = 'return await ctx.workflow.run(config.workflow_id, input);',
    ui_config = '{"icon": "workflow", "color": "#10B981"}',
    is_system = TRUE
WHERE slug = 'subflow' AND tenant_id IS NULL;

-- wait block
UPDATE block_definitions SET
    code = $code$
if (config.duration_ms) {
    await new Promise(resolve => setTimeout(resolve, config.duration_ms));
}
return input;
$code$,
    ui_config = '{"icon": "clock", "color": "#6B7280"}',
    is_system = TRUE
WHERE slug = 'wait' AND tenant_id IS NULL;

-- human_in_loop block
UPDATE block_definitions SET
    code = $code$
return await ctx.human.requestApproval({
    instructions: config.instructions,
    timeout: config.timeout_hours,
    data: input
});
$code$,
    ui_config = '{"icon": "user-check", "color": "#EC4899"}',
    is_system = TRUE
WHERE slug = 'human_in_loop' AND tenant_id IS NULL;

-- error block
UPDATE block_definitions SET
    code = $code$
throw new Error(config.error_message || 'Workflow stopped with error');
$code$,
    ui_config = '{"icon": "alert-circle", "color": "#EF4444"}',
    is_system = TRUE
WHERE slug = 'error' AND tenant_id IS NULL;

-- function block
UPDATE block_definitions SET
    code = $code$
// This block executes user-defined code
// The user's code is stored in config.code and evaluated dynamically
return input;
$code$,
    ui_config = '{"icon": "code", "color": "#6366F1"}',
    is_system = TRUE
WHERE slug = 'function' AND tenant_id IS NULL;

-- router block
UPDATE block_definitions SET
    code = $code$
const routeDescriptions = (config.routes || []).map(r =>
    `${r.name}: ${r.description}`
).join('\n');

const prompt = `Given the following input, select the most appropriate route.

Routes:
${routeDescriptions}

Input: ${JSON.stringify(input)}

Respond with only the route name.`;

const response = await ctx.llm.chat(config.provider || 'openai', config.model || 'gpt-4', {
    messages: [{ role: 'user', content: prompt }]
});

const selectedRoute = response.content.trim();
return {
    ...input,
    __branch: selectedRoute
};
$code$,
    ui_config = '{"icon": "git-branch", "color": "#8B5CF6"}',
    is_system = TRUE
WHERE slug = 'router' AND tenant_id IS NULL;

-- note block
UPDATE block_definitions SET
    code = 'return input;',
    ui_config = '{"icon": "file-text", "color": "#9CA3AF"}',
    is_system = TRUE
WHERE slug = 'note' AND tenant_id IS NULL;

-- Add http block (new)
INSERT INTO block_definitions (tenant_id, slug, name, description, category, icon, executor_type, config_schema, error_codes, code, ui_config, is_system)
VALUES (
    NULL,
    'http',
    'HTTP Request',
    'Make HTTP API calls',
    'integration',
    'globe',
    'builtin',
    '{"type":"object","properties":{"url":{"type":"string"},"method":{"type":"string","enum":["GET","POST","PUT","DELETE","PATCH"]},"headers":{"type":"object"},"body":{"type":"object"}}}',
    '[{"code":"HTTP_001","name":"CONNECTION_ERROR","description":"Failed to connect","retryable":true},{"code":"HTTP_002","name":"TIMEOUT","description":"Request timeout","retryable":true}]',
    $code$
const url = renderTemplate(config.url, input);

const response = await ctx.http.request(url, {
    method: config.method || 'GET',
    headers: config.headers || {},
    body: config.body ? renderTemplate(JSON.stringify(config.body), input) : null
});

return response;
$code$,
    '{"icon": "globe", "color": "#3B82F6"}',
    TRUE
)
ON CONFLICT (tenant_id, slug) DO UPDATE SET
    code = EXCLUDED.code,
    ui_config = EXCLUDED.ui_config,
    is_system = EXCLUDED.is_system;

-- Add code block (user-defined code execution)
INSERT INTO block_definitions (tenant_id, slug, name, description, category, icon, executor_type, config_schema, error_codes, code, ui_config, is_system)
VALUES (
    NULL,
    'code',
    'Code',
    'Execute custom JavaScript code',
    'utility',
    'terminal',
    'builtin',
    '{"type":"object","properties":{"code":{"type":"string","description":"JavaScript code to execute"}}}',
    '[{"code":"CODE_001","name":"SYNTAX_ERROR","description":"JavaScript syntax error","retryable":false},{"code":"CODE_002","name":"RUNTIME_ERROR","description":"JavaScript runtime error","retryable":false}]',
    '// User code is dynamically injected\nreturn input;',
    '{"icon": "terminal", "color": "#6366F1"}',
    TRUE
)
ON CONFLICT (tenant_id, slug) DO UPDATE SET
    code = EXCLUDED.code,
    ui_config = EXCLUDED.ui_config,
    is_system = EXCLUDED.is_system;

-- Update executor_type constraint to include 'code'
ALTER TABLE block_definitions DROP CONSTRAINT IF EXISTS valid_executor_type;
ALTER TABLE block_definitions ADD CONSTRAINT valid_executor_type
    CHECK (executor_type IN ('builtin', 'http', 'function', 'code'));
