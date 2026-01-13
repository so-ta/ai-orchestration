-- AI Orchestration Seed Data
-- Initial data for the application
--
-- Usage:
--   psql -U aio -d ai_orchestration -f seed.sql
--
-- Note: Run this AFTER applying schema.sql

\restrict U4nbNOpGbzQDaE5Nbt6A6xfSA28CJLmedtMlJHtwSRT23KSx5kSKoTOYEzhykri
INSERT INTO tenants (id, name, slug, settings, created_at, updated_at, deleted_at, status, plan, owner_email, owner_name, billing_email, metadata, feature_flags, limits, suspended_at, suspended_reason) VALUES ('00000000-0000-0000-0000-000000000001', 'Default Tenant', 'default-tenant', '{"data_retention_days": 30}', '2026-01-12 09:22:38.988109+00', '2026-01-12 09:22:38.988109+00', NULL, 'active', 'free', NULL, NULL, NULL, '{}', '{"api_access": true, "audit_logs": true, "sso_enabled": false, "custom_blocks": true, "copilot_enabled": true, "advanced_analytics": true, "max_concurrent_runs": 10}', '{"max_users": 50, "max_workflows": 100, "max_storage_mb": 10240, "retention_days": 90, "max_credentials": 100, "max_runs_per_day": 1000}', NULL, NULL);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('b860463c-5e1d-4c00-94bd-7cb4ca420b67', NULL, 'subflow', 'Subflow', 'Execute another workflow', 'integration', 'workflow', '{"type": "object", "properties": {"workflow_id": {"type": "string", "format": "uuid"}, "workflow_version": {"type": "integer"}}}', NULL, NULL, '[{"code": "SUBFLOW_001", "name": "NOT_FOUND", "retryable": false, "description": "Subflow workflow not found"}, {"code": "SUBFLOW_002", "name": "EXEC_ERROR", "retryable": true, "description": "Subflow execution error"}]', true, '2026-01-12 09:23:29.067836+00', '2026-01-12 09:23:30.000222+00', '[{"name": "output", "label": "Output", "is_default": true, "description": "Subflow result"}]', '[{"name": "input", "label": "Input", "schema": {"type": "any"}, "required": false, "description": "Input data for subflow"}]', '[]', false, 'return await ctx.workflow.run(config.workflow_id, input);', '{"icon": "workflow", "color": "#10B981"}', true, 1);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('065ed730-c42e-4fdb-9ddf-0a16347926c4', NULL, 'map', 'Map', 'Process array items in parallel', 'data', 'layers', '{"type": "object", "properties": {"parallel": {"type": "boolean"}, "input_path": {"type": "string"}, "max_workers": {"type": "integer"}}}', NULL, NULL, '[{"code": "MAP_001", "name": "INVALID_PATH", "retryable": false, "description": "Invalid input path"}]', true, '2026-01-12 09:23:29.067836+00', '2026-01-12 09:23:29.998717+00', '[{"name": "item", "label": "Item", "is_default": true, "description": "Each mapped item"}, {"name": "complete", "label": "Complete", "is_default": false, "description": "All items processed"}]', '[{"name": "items", "label": "Items", "schema": {"type": "array", "items": {"type": "any"}}, "required": true, "description": "Array of items to process"}]', '[]', false, '
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
', '{"icon": "layers", "color": "#06B6D4"}', true, 1);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('d90aa273-3ee1-4802-8165-aafbe87dcb5a', NULL, 'split', 'Split', 'Split into batches', 'data', 'scissors', '{"type": "object", "properties": {"batch_size": {"type": "integer", "minimum": 1}, "input_path": {"type": "string"}}}', NULL, NULL, '[]', true, '2026-01-12 09:23:29.067836+00', '2026-01-12 09:23:29.999493+00', '[{"name": "output", "label": "Output", "is_default": true, "description": "Split batches"}]', '[{"name": "input", "label": "Input", "schema": {"type": "any"}, "required": true, "description": "Data to split into branches"}]', '[]', false, '
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
', '{"icon": "scissors", "color": "#06B6D4"}', true, 1);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('95ec4521-9a26-4031-80fd-4011d54a0470', NULL, 'function', 'Function', 'Execute custom JavaScript', 'utility', 'code', '{"type": "object", "properties": {"code": {"type": "string"}, "language": {"enum": ["javascript"], "type": "string"}, "timeout_ms": {"type": "integer"}}}', NULL, NULL, '[{"code": "FUNC_001", "name": "SYNTAX_ERROR", "retryable": false, "description": "JavaScript syntax error"}, {"code": "FUNC_002", "name": "RUNTIME_ERROR", "retryable": false, "description": "JavaScript runtime error"}, {"code": "FUNC_003", "name": "TIMEOUT", "retryable": false, "description": "Function execution timeout"}]', true, '2026-01-12 09:23:29.067836+00', '2026-01-12 09:23:30.001214+00', '[{"name": "output", "label": "Output", "is_default": true, "description": "Function result"}]', '[{"name": "input", "label": "Input", "schema": {"type": "any"}, "required": false, "description": "Input data for function"}]', '[]', false, '
// This block executes user-defined code
// The user''s code is stored in config.code and evaluated dynamically
return input;
', '{"icon": "code", "color": "#6366F1"}', true, 1);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('7483993e-c52a-49de-8051-68df5afe6ffb', NULL, 'wait', 'Wait', 'Pause execution', 'control', 'clock', '{"type": "object", "properties": {"until": {"type": "string", "format": "date-time"}, "duration_ms": {"type": "integer", "minimum": 0}}}', NULL, NULL, '[]', true, '2026-01-12 09:23:29.067836+00', '2026-01-12 09:23:30.000437+00', '[{"name": "output", "label": "Output", "is_default": true, "description": "Continues after wait"}]', '[{"name": "input", "label": "Input", "schema": {"type": "any"}, "required": false, "description": "Data to pass through after wait"}]', '[]', false, '
if (config.duration_ms) {
    await new Promise(resolve => setTimeout(resolve, config.duration_ms));
}
return input;
', '{"icon": "clock", "color": "#6B7280"}', true, 1);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('decf6fc1-edff-43dc-90bf-23b76d5ffdcf', NULL, 'condition', 'Condition', 'Branch based on expression', 'logic', 'git-branch', '{"type": "object", "properties": {"expression": {"type": "string", "description": "JSONPath expression"}}}', NULL, NULL, '[{"code": "COND_001", "name": "INVALID_EXPR", "retryable": false, "description": "Invalid condition expression"}, {"code": "COND_002", "name": "EVAL_ERROR", "retryable": false, "description": "Expression evaluation error"}]', true, '2026-01-12 09:23:29.067836+00', '2026-01-12 09:23:29.997933+00', '[{"name": "true", "label": "Yes", "is_default": true, "description": "When condition is true"}, {"name": "false", "label": "No", "is_default": false, "description": "When condition is false"}]', '[{"name": "input", "label": "Input", "schema": {"type": "any"}, "required": true, "description": "Data to evaluate condition against"}]', '[]', false, '
const result = evaluate(config.expression, input);
return {
    ...input,
    __branch: result ? ''then'' : ''else''
};
', '{"icon": "git-branch", "color": "#F59E0B"}', true, 1);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('ce7d25be-8968-4fd8-91ef-a0a87cec7081', NULL, 'human_in_loop', 'Human in Loop', 'Wait for human approval', 'control', 'user-check', '{"type": "object", "properties": {"approval_url": {"type": "boolean"}, "instructions": {"type": "string"}, "timeout_hours": {"type": "integer"}}}', NULL, NULL, '[{"code": "HIL_001", "name": "TIMEOUT", "retryable": false, "description": "Human approval timeout"}, {"code": "HIL_002", "name": "REJECTED", "retryable": false, "description": "Human rejected"}]', true, '2026-01-12 09:23:29.067836+00', '2026-01-12 09:23:30.000697+00', '[{"name": "approved", "label": "Approved", "is_default": true, "description": "When approved"}, {"name": "rejected", "label": "Rejected", "is_default": false, "description": "When rejected"}, {"name": "timeout", "label": "Timeout", "is_default": false, "description": "When timed out"}]', '[{"name": "input", "label": "Input", "schema": {"type": "any"}, "required": true, "description": "Context data for human review"}]', '[]', false, '
return await ctx.human.requestApproval({
    instructions: config.instructions,
    timeout: config.timeout_hours,
    data: input
});
', '{"icon": "user-check", "color": "#EC4899"}', true, 1);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('2319b4e4-2060-4341-b8ab-18c6196b005b', NULL, 'loop', 'Loop', 'Iterate with for/forEach/while', 'logic', 'repeat', '{"type": "object", "properties": {"count": {"type": "integer"}, "condition": {"type": "string"}, "loop_type": {"enum": ["for", "forEach", "while", "doWhile"], "type": "string"}, "input_path": {"type": "string"}, "max_iterations": {"type": "integer"}}}', NULL, NULL, '[{"code": "LOOP_001", "name": "MAX_ITERATIONS", "retryable": false, "description": "Maximum iterations exceeded"}]', true, '2026-01-12 09:23:29.067836+00', '2026-01-12 09:23:29.998477+00', '[{"name": "loop", "label": "Loop Body", "is_default": true, "description": "Each iteration"}, {"name": "complete", "label": "Complete", "is_default": false, "description": "When loop finishes"}]', '[{"name": "input", "label": "Input", "schema": {"type": "any"}, "required": true, "description": "Initial value or array to iterate"}]', '[]', false, '
const results = [];
const maxIter = config.max_iterations || 100;
if (config.loop_type === ''for'') {
    for (let i = 0; i < (config.count || 0) && i < maxIter; i++) {
        results.push({ index: i, ...input });
    }
} else if (config.loop_type === ''forEach'') {
    const items = getPath(input, config.input_path) || [];
    for (let i = 0; i < items.length && i < maxIter; i++) {
        results.push({ item: items[i], index: i, ...input });
    }
} else if (config.loop_type === ''while'') {
    let i = 0;
    while (evaluate(config.condition, input) && i < maxIter) {
        results.push({ index: i, ...input });
        i++;
    }
}
return { ...input, results, iterations: results.length };
', '{"icon": "repeat", "color": "#F59E0B"}', true, 1);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('210a9570-8b4a-49e8-b243-5f7a76c2e5e6', NULL, 'start', 'Start', 'Workflow entry point', 'control', 'play', '{}', NULL, NULL, '[]', true, '2026-01-12 09:23:29.067836+00', '2026-01-12 09:23:29.996184+00', '[{"name": "output", "label": "Output", "is_default": true, "description": "Workflow input data"}]', '[]', '[]', false, 'return input;', '{"icon": "play", "color": "#10B981"}', true, 1);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('942ce139-4c4f-4a9b-91ec-4fa1efdabe1f', NULL, 'note', 'Note', 'Documentation/comment', 'utility', 'file-text', '{"type": "object", "properties": {"color": {"type": "string"}, "content": {"type": "string"}}}', NULL, NULL, '[]', true, '2026-01-12 09:23:29.067836+00', '2026-01-12 09:23:30.001694+00', '[]', '[]', '[]', false, 'return input;', '{"icon": "file-text", "color": "#9CA3AF"}', true, 1);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('b1817047-47fe-4c07-af12-9ecff95b84fd', NULL, 'join', 'Join', 'Merge multiple branches', 'data', 'git-merge', '{}', NULL, NULL, '[]', true, '2026-01-12 09:23:29.067836+00', '2026-01-12 09:23:29.998975+00', '[{"name": "output", "label": "Output", "is_default": true, "description": "Merged data"}]', '[{"name": "input_1", "label": "Input 1", "schema": {"type": "any"}, "required": false, "description": "First branch result"}, {"name": "input_2", "label": "Input 2", "schema": {"type": "any"}, "required": false, "description": "Second branch result"}, {"name": "input_3", "label": "Input 3", "schema": {"type": "any"}, "required": false, "description": "Third branch result"}, {"name": "input_4", "label": "Input 4", "schema": {"type": "any"}, "required": false, "description": "Fourth branch result"}]', '[]', false, 'return input;', '{"icon": "git-merge", "color": "#06B6D4"}', true, 1);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('467509dd-df43-422a-be21-4b70694bb75a', NULL, 'filter', 'Filter', 'Filter items by condition', 'data', 'filter', '{"type": "object", "properties": {"keep_all": {"type": "boolean"}, "expression": {"type": "string"}}}', NULL, NULL, '[{"code": "FILTER_001", "name": "INVALID_EXPR", "retryable": false, "description": "Invalid filter expression"}]', true, '2026-01-12 09:23:29.067836+00', '2026-01-12 09:23:29.999238+00', '[{"name": "matched", "label": "Matched", "is_default": true, "description": "Items matching condition"}, {"name": "unmatched", "label": "Unmatched", "is_default": false, "description": "Items not matching"}]', '[{"name": "items", "label": "Items", "schema": {"type": "array", "items": {"type": "any"}}, "required": true, "description": "Array of items to filter"}]', '[]', false, '
const items = Array.isArray(input) ? input : (input.items || []);
const filtered = items.filter(item => evaluate(config.expression, item));
return {
    items: filtered,
    original_count: items.length,
    filtered_count: filtered.length,
    removed_count: items.length - filtered.length
};
', '{"icon": "filter", "color": "#06B6D4"}', true, 1);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('722549c9-c09e-44ad-873b-bd2b685ef8d9', NULL, 'tool', 'Tool', 'Execute external tool/adapter', 'integration', 'wrench', '{"type": "object", "properties": {"adapter_id": {"type": "string"}}}', NULL, NULL, '[{"code": "TOOL_001", "name": "ADAPTER_NOT_FOUND", "retryable": false, "description": "Adapter not found"}, {"code": "TOOL_002", "name": "EXEC_ERROR", "retryable": true, "description": "Tool execution error"}]', true, '2026-01-12 09:23:29.067836+00', '2026-01-12 09:23:30.000005+00', '[{"name": "output", "label": "Output", "is_default": true, "description": "Tool execution result"}]', '[{"name": "input", "label": "Input", "schema": {"type": "any"}, "required": false, "description": "Input data for the tool"}]', '[]', false, 'return await ctx.adapter.call(config.adapter_id, input);', '{"icon": "wrench", "color": "#10B981"}', true, 1);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('6cd425ce-6314-433a-a827-3ad66a2e5e48', NULL, 'log', 'Log', 'Output log messages for debugging', 'utility', 'terminal', '{"type": "object", "properties": {"data": {"type": "string", "description": "JSON path to include additional data (e.g. $.input)"}, "level": {"enum": ["debug", "info", "warn", "error"], "type": "string", "default": "info", "description": "Log level"}, "message": {"type": "string", "description": "Log message (supports {{$.field}} template variables)"}}}', NULL, NULL, '[]', true, '2026-01-12 09:23:29.764471+00', '2026-01-12 09:23:29.764471+00', '[]', '[]', '[]', false, NULL, '{}', false, 1);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('eab46517-3672-40f6-82fe-88a6313cdce0', NULL, 'switch', 'Switch', 'Multi-branch routing', 'logic', 'shuffle', '{"type": "object", "properties": {"mode": {"enum": ["rules", "expression"], "type": "string"}, "cases": {"type": "array", "items": {"type": "object", "properties": {"name": {"type": "string"}, "expression": {"type": "string"}, "is_default": {"type": "boolean"}}}}}}', NULL, NULL, '[{"code": "SWITCH_001", "name": "NO_MATCH", "retryable": false, "description": "No matching case"}]', true, '2026-01-12 09:23:29.067836+00', '2026-01-12 09:23:29.998206+00', '[{"name": "default", "label": "Default", "is_default": true, "description": "When no case matches"}]', '[{"name": "input", "label": "Input", "schema": {"type": "any"}, "required": true, "description": "Value to switch on"}]', '[]', false, '
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
    __branch: matchedCase || ''default''
};
', '{"icon": "shuffle", "color": "#F59E0B"}', true, 1);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('034d56d2-56c4-48c8-8319-f7b4fdaf79fc', NULL, 'aggregate', 'Aggregate', 'Aggregate data operations', 'data', 'database', '{"type": "object", "properties": {"group_by": {"type": "string"}, "operations": {"type": "array", "items": {"type": "object", "properties": {"field": {"type": "string"}, "operation": {"enum": ["sum", "count", "avg", "min", "max", "first", "last", "concat"], "type": "string"}, "output_field": {"type": "string"}}}}}}', NULL, NULL, '[]', true, '2026-01-12 09:23:29.067836+00', '2026-01-12 09:23:29.999739+00', '[{"name": "output", "label": "Output", "is_default": true, "description": "Aggregated result"}]', '[{"name": "input_1", "label": "Input 1", "schema": {"type": "any"}, "required": false, "description": "First data source"}, {"name": "input_2", "label": "Input 2", "schema": {"type": "any"}, "required": false, "description": "Second data source"}, {"name": "input_3", "label": "Input 3", "schema": {"type": "any"}, "required": false, "description": "Third data source"}, {"name": "input_4", "label": "Input 4", "schema": {"type": "any"}, "required": false, "description": "Fourth data source"}]', '[]', false, '
const items = Array.isArray(input) ? input : (input.items || []);
const result = {};
for (const op of config.operations || []) {
    const values = items.map(item => getPath(item, op.field));
    switch (op.operation) {
        case ''sum'': result[op.output_field] = values.reduce((a, b) => a + b, 0); break;
        case ''count'': result[op.output_field] = values.length; break;
        case ''avg'': result[op.output_field] = values.reduce((a, b) => a + b, 0) / values.length; break;
        case ''min'': result[op.output_field] = Math.min(...values); break;
        case ''max'': result[op.output_field] = Math.max(...values); break;
        case ''first'': result[op.output_field] = values[0]; break;
        case ''last'': result[op.output_field] = values[values.length - 1]; break;
        case ''concat'': result[op.output_field] = values.join(''''); break;
    }
}
return result;
', '{"icon": "database", "color": "#06B6D4"}', true, 1);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('97f46b68-380a-4094-8e38-c3071724795d', NULL, 'error', 'Error', 'Stop workflow with error', 'control', 'alert-circle', '{"type": "object", "properties": {"error_code": {"type": "string"}, "error_type": {"type": "string"}, "error_message": {"type": "string"}}}', NULL, NULL, '[]', true, '2026-01-12 09:23:29.067836+00', '2026-01-12 09:23:30.000962+00', '[]', '[{"name": "error", "label": "Error", "schema": {"type": "object", "properties": {"code": {"type": "string"}, "message": {"type": "string"}}}, "required": true, "description": "Error information to handle"}]', '[]', false, '
throw new Error(config.error_message || ''Workflow stopped with error'');
', '{"icon": "alert-circle", "color": "#EF4444"}', true, 1);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('31a7de23-04aa-4102-affa-aa782a6ddfbf', NULL, 'router', 'Router', 'AI-driven dynamic routing', 'ai', 'git-branch', '{"type": "object", "properties": {"model": {"type": "string"}, "routes": {"type": "array", "items": {"type": "object", "properties": {"name": {"type": "string"}, "description": {"type": "string"}}}}, "provider": {"type": "string"}}}', NULL, NULL, '[{"code": "ROUTER_001", "name": "NO_MATCH", "retryable": false, "description": "No matching route found"}]', true, '2026-01-12 09:23:29.067836+00', '2026-01-12 09:23:30.001445+00', '[{"name": "default", "label": "Default", "is_default": true, "description": "Default route when no match"}]', '[{"name": "input", "label": "Input", "schema": {"type": "string"}, "required": true, "description": "Message to analyze for routing"}]', '[{"name": "llm_api_key", "type": "api_key", "scope": "system", "required": true, "description": "LLM Provider API Key"}]', false, '
const routeDescriptions = (config.routes || []).map(r =>
    `${r.name}: ${r.description}`
).join(''\n'');
const prompt = `Given the following input, select the most appropriate route.
Routes:
${routeDescriptions}
Input: ${JSON.stringify(input)}
Respond with only the route name.`;
const response = await ctx.llm.chat(config.provider || ''openai'', config.model || ''gpt-4'', {
    messages: [{ role: ''user'', content: prompt }]
});
const selectedRoute = response.content.trim();
return {
    ...input,
    __branch: selectedRoute
};
', '{"icon": "git-branch", "color": "#8B5CF6"}', true, 1);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('463514db-ad3a-47b8-9f93-a01d2bcda8bc', NULL, 'http', 'HTTP Request', 'Make HTTP API calls', 'integration', 'globe', '{"type": "object", "properties": {"url": {"type": "string"}, "body": {"type": "object"}, "method": {"enum": ["GET", "POST", "PUT", "DELETE", "PATCH"], "type": "string"}, "headers": {"type": "object"}}}', NULL, NULL, '[{"code": "HTTP_001", "name": "CONNECTION_ERROR", "retryable": true, "description": "Failed to connect"}, {"code": "HTTP_002", "name": "TIMEOUT", "retryable": true, "description": "Request timeout"}]', true, '2026-01-12 09:23:30.001921+00', '2026-01-12 09:23:30.001921+00', '[]', '[]', '[]', false, '
const url = renderTemplate(config.url, input);
const response = await ctx.http.request(url, {
    method: config.method || ''GET'',
    headers: config.headers || {},
    body: config.body ? renderTemplate(JSON.stringify(config.body), input) : null
});
return response;
', '{"icon": "globe", "color": "#3B82F6"}', true, 1);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('37e0928b-6717-4cbf-96f3-ca3928b4d394', NULL, 'code', 'Code', 'Execute custom JavaScript code', 'utility', 'terminal', '{"type": "object", "properties": {"code": {"type": "string", "description": "JavaScript code to execute"}}}', NULL, NULL, '[{"code": "CODE_001", "name": "SYNTAX_ERROR", "retryable": false, "description": "JavaScript syntax error"}, {"code": "CODE_002", "name": "RUNTIME_ERROR", "retryable": false, "description": "JavaScript runtime error"}]', true, '2026-01-12 09:23:30.00232+00', '2026-01-12 09:23:30.00232+00', '[]', '[]', '[]', false, '// User code is dynamically injected\nreturn input;', '{"icon": "terminal", "color": "#6366F1"}', true, 1);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('207aaa19-cd68-4bd4-b42c-d670fae4e77c', NULL, 'slack', 'Slack', 'Slackチャンネルにメッセージを送信', 'integration', 'message-square', '{"type": "object", "required": ["message"], "properties": {"blocks": {"type": "array", "title": "Block Kit", "description": "Slack Block Kit形式のメッセージ（JSON）", "x-ui-widget": "json"}, "channel": {"type": "string", "title": "チャンネル", "description": "投稿先チャンネル（#general など）"}, "message": {"type": "string", "title": "メッセージ", "description": "送信するメッセージ（テンプレート使用可: ${input.field}）", "x-ui-widget": "textarea"}, "username": {"type": "string", "title": "表示名", "description": "ボットの表示名"}, "icon_emoji": {"type": "string", "title": "アイコン絵文字", "description": "ボットのアイコン（:robot_face: など）"}, "webhook_url": {"type": "string", "title": "Webhook URL", "description": "Slack Incoming Webhook URL（空の場合はシークレットSLACK_WEBHOOK_URLを使用）"}}}', '{"type": "object"}', '{"type": "object", "properties": {"status": {"type": "number"}, "success": {"type": "boolean"}}}', '[{"code": "SLACK_001", "name": "WEBHOOK_NOT_CONFIGURED", "retryable": false, "description": "Webhook URLが設定されていません"}, {"code": "SLACK_002", "name": "SEND_FAILED", "retryable": true, "description": "メッセージ送信に失敗しました"}, {"code": "SLACK_003", "name": "INVALID_WEBHOOK", "retryable": false, "description": "無効なWebhook URL"}]', true, '2026-01-12 09:23:30.218182+00', '2026-01-12 09:23:30.218182+00', '[]', '[]', '[]', false, '
const webhookUrl = config.webhook_url || ctx.secrets.SLACK_WEBHOOK_URL;
if (!webhookUrl) {
    throw new Error(''[SLACK_001] Webhook URLが設定されていません'');
}
const payload = {
    text: renderTemplate(config.message, input)
};
if (config.channel) {
    payload.channel = config.channel;
}
if (config.username) {
    payload.username = config.username;
}
if (config.icon_emoji) {
    payload.icon_emoji = config.icon_emoji;
}
if (config.blocks && config.blocks.length > 0) {
    payload.blocks = config.blocks;
}
const response = await ctx.http.post(webhookUrl, payload, {
    headers: { ''Content-Type'': ''application/json'' }
});
if (response.status >= 400) {
    throw new Error(''[SLACK_002] Slack送信失敗: '' + response.status);
}
return { success: true, status: response.status };
    ', '{"icon": "message-square", "color": "#4A154B"}', true, 1);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('4192fef4-6164-412a-920f-f9d9143fb790', NULL, 'discord', 'Discord', 'Discord Webhookにメッセージを送信', 'integration', 'message-circle', '{"type": "object", "required": ["content"], "properties": {"embeds": {"type": "array", "title": "Embeds", "description": "リッチな埋め込みメッセージ（JSON配列）", "x-ui-widget": "json"}, "content": {"type": "string", "title": "メッセージ", "description": "送信するメッセージ（テンプレート使用可）", "x-ui-widget": "textarea"}, "username": {"type": "string", "title": "ユーザー名", "description": "Webhookの表示名を上書き"}, "avatar_url": {"type": "string", "title": "アバターURL", "description": "Webhookのアバター画像URL"}, "webhook_url": {"type": "string", "title": "Webhook URL", "description": "Discord Webhook URL（空の場合はシークレットDISCORD_WEBHOOK_URLを使用）"}}}', '{"type": "object"}', '{"type": "object", "properties": {"status": {"type": "number"}, "success": {"type": "boolean"}}}', '[{"code": "DISCORD_001", "name": "WEBHOOK_NOT_CONFIGURED", "retryable": false, "description": "Webhook URLが設定されていません"}, {"code": "DISCORD_002", "name": "SEND_FAILED", "retryable": true, "description": "メッセージ送信に失敗しました"}, {"code": "DISCORD_003", "name": "RATE_LIMITED", "retryable": true, "description": "レート制限に達しました"}]', true, '2026-01-12 09:23:30.219664+00', '2026-01-12 09:23:30.219664+00', '[]', '[]', '[]', false, '
const webhookUrl = config.webhook_url || ctx.secrets.DISCORD_WEBHOOK_URL;
if (!webhookUrl) {
    throw new Error(''[DISCORD_001] Webhook URLが設定されていません'');
}
const payload = {
    content: renderTemplate(config.content, input)
};
if (config.username) {
    payload.username = config.username;
}
if (config.avatar_url) {
    payload.avatar_url = config.avatar_url;
}
if (config.embeds && config.embeds.length > 0) {
    payload.embeds = config.embeds;
}
const response = await ctx.http.post(webhookUrl, payload, {
    headers: { ''Content-Type'': ''application/json'' }
});
if (response.status === 429) {
    throw new Error(''[DISCORD_003] レート制限に達しました'');
}
if (response.status >= 400) {
    throw new Error(''[DISCORD_002] Discord送信失敗: '' + response.status);
}
return { success: true, status: response.status };
    ', '{"icon": "message-circle", "color": "#5865F2"}', true, 1);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('608c9830-b4b8-42f3-9716-9289b263188c', NULL, 'notion_create_page', 'Notion: ページ作成', 'Notionにページを作成', 'integration', 'file-text', '{"type": "object", "required": ["parent_id"], "properties": {"title": {"type": "string", "title": "タイトル", "description": "ページタイトル（テンプレート使用可）"}, "api_key": {"type": "string", "title": "API Key", "description": "Notion API Key（空の場合はシークレットNOTION_API_KEYを使用）"}, "content": {"type": "string", "title": "本文", "description": "ページの本文コンテンツ（テンプレート使用可）", "x-ui-widget": "textarea"}, "parent_id": {"type": "string", "title": "親ID", "description": "親ページまたはデータベースのID"}, "properties": {"type": "object", "title": "プロパティ", "description": "DBプロパティ（JSON形式）", "x-ui-widget": "json"}, "parent_type": {"enum": ["page_id", "database_id"], "type": "string", "title": "親タイプ", "default": "database_id"}}}', '{"type": "object"}', '{"type": "object", "properties": {"id": {"type": "string"}, "url": {"type": "string"}, "created_time": {"type": "string"}}}', '[{"code": "NOTION_001", "name": "API_KEY_NOT_CONFIGURED", "retryable": false, "description": "API Keyが設定されていません"}, {"code": "NOTION_002", "name": "CREATE_FAILED", "retryable": true, "description": "ページ作成に失敗しました"}, {"code": "NOTION_003", "name": "INVALID_PARENT", "retryable": false, "description": "無効な親IDです"}]', true, '2026-01-12 09:23:30.220201+00', '2026-01-12 09:23:30.220201+00', '[]', '[]', '[]', false, '
const apiKey = config.api_key || ctx.secrets.NOTION_API_KEY;
if (!apiKey) {
    throw new Error(''[NOTION_001] API Keyが設定されていません'');
}
const parentKey = config.parent_type || ''database_id'';
const payload = {
    parent: { [parentKey]: config.parent_id }
};
// プロパティ設定
if (config.properties) {
    payload.properties = config.properties;
} else if (config.title) {
    // シンプルなタイトルのみの場合
    payload.properties = {
        title: {
            title: [{ text: { content: renderTemplate(config.title, input) } }]
        }
    };
}
// 本文コンテンツ（シンプルなparagraph）
if (config.content) {
    payload.children = [
        {
            object: ''block'',
            type: ''paragraph'',
            paragraph: {
                rich_text: [{ type: ''text'', text: { content: renderTemplate(config.content, input) } }]
            }
        }
    ];
}
const response = await ctx.http.post(''https://api.notion.com/v1/pages'', payload, {
    headers: {
        ''Authorization'': ''Bearer '' + apiKey,
        ''Content-Type'': ''application/json'',
        ''Notion-Version'': ''2022-06-28''
    }
});
if (response.status >= 400) {
    const errorMsg = response.body?.message || ''Unknown error'';
    throw new Error(''[NOTION_002] ページ作成失敗: '' + errorMsg);
}
return {
    id: response.body.id,
    url: response.body.url,
    created_time: response.body.created_time
};
    ', '{"icon": "file-text", "color": "#000000"}', true, 1);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('1b182587-c5c1-4bda-aa14-0c456459be1a', NULL, 'notion_query_db', 'Notion: DB検索', 'Notionデータベースを検索', 'integration', 'database', '{"type": "object", "required": ["database_id"], "properties": {"sorts": {"type": "array", "title": "ソート", "description": "ソート条件（JSON配列）", "x-ui-widget": "json"}, "filter": {"type": "object", "title": "フィルター", "description": "Notion Filter形式（JSON）", "x-ui-widget": "json"}, "api_key": {"type": "string", "title": "API Key", "description": "Notion API Key（空の場合はシークレットNOTION_API_KEYを使用）"}, "page_size": {"type": "number", "title": "取得件数", "default": 100, "maximum": 100, "minimum": 1}, "database_id": {"type": "string", "title": "データベースID", "description": "検索対象のデータベースID"}}}', '{"type": "object"}', '{"type": "object", "properties": {"results": {"type": "array"}, "has_more": {"type": "boolean"}, "next_cursor": {"type": "string"}}}', '[{"code": "NOTION_001", "name": "API_KEY_NOT_CONFIGURED", "retryable": false, "description": "API Keyが設定されていません"}, {"code": "NOTION_004", "name": "QUERY_FAILED", "retryable": true, "description": "クエリに失敗しました"}]', true, '2026-01-12 09:23:30.220729+00', '2026-01-12 09:23:30.220729+00', '[]', '[]', '[]', false, '
const apiKey = config.api_key || ctx.secrets.NOTION_API_KEY;
if (!apiKey) {
    throw new Error(''[NOTION_001] API Keyが設定されていません'');
}
const payload = {};
if (config.filter) {
    payload.filter = config.filter;
}
if (config.sorts) {
    payload.sorts = config.sorts;
}
if (config.page_size) {
    payload.page_size = config.page_size;
}
const response = await ctx.http.post(
    ''https://api.notion.com/v1/databases/'' + config.database_id + ''/query'',
    payload,
    {
        headers: {
            ''Authorization'': ''Bearer '' + apiKey,
            ''Content-Type'': ''application/json'',
            ''Notion-Version'': ''2022-06-28''
        }
    }
);
if (response.status >= 400) {
    const errorMsg = response.body?.message || ''Unknown error'';
    throw new Error(''[NOTION_004] クエリ失敗: '' + errorMsg);
}
return {
    results: response.body.results,
    has_more: response.body.has_more,
    next_cursor: response.body.next_cursor
};
    ', '{"icon": "database", "color": "#000000"}', true, 1);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('5eddf8f1-4913-4c29-8934-68c59a72e603', NULL, 'gsheets_append', 'Google Sheets: 行追加', 'Google Sheetsに行を追加', 'integration', 'table', '{"type": "object", "required": ["spreadsheet_id", "values"], "properties": {"range": {"type": "string", "title": "範囲", "default": "Sheet1!A:Z", "description": "シート名と範囲（例: Sheet1!A:Z）"}, "values": {"type": "array", "title": "値", "description": "追加する行データ（2次元配列またはテンプレート）", "x-ui-widget": "json"}, "api_key": {"type": "string", "title": "API Key", "description": "Google API Key（空の場合はシークレットGOOGLE_API_KEYを使用）"}, "spreadsheet_id": {"type": "string", "title": "スプレッドシートID", "description": "URLから取得: /d/{spreadsheet_id}/edit"}, "value_input_option": {"enum": ["RAW", "USER_ENTERED"], "type": "string", "title": "入力形式", "default": "USER_ENTERED"}}}', '{"type": "object"}', '{"type": "object", "properties": {"updated_rows": {"type": "number"}, "updated_cells": {"type": "number"}, "updated_range": {"type": "string"}}}', '[{"code": "GSHEETS_001", "name": "API_KEY_NOT_CONFIGURED", "retryable": false, "description": "API Keyが設定されていません"}, {"code": "GSHEETS_002", "name": "APPEND_FAILED", "retryable": true, "description": "行追加に失敗しました"}, {"code": "GSHEETS_003", "name": "INVALID_SPREADSHEET", "retryable": false, "description": "スプレッドシートが見つかりません"}]', true, '2026-01-12 09:23:30.221106+00', '2026-01-12 09:23:30.221106+00', '[]', '[]', '[]', false, '
const apiKey = config.api_key || ctx.secrets.GOOGLE_API_KEY;
if (!apiKey) {
    throw new Error(''[GSHEETS_001] API Keyが設定されていません'');
}
const range = encodeURIComponent(config.range || ''Sheet1!A:Z'');
const valueInputOption = config.value_input_option || ''USER_ENTERED'';
const url = ''https://sheets.googleapis.com/v4/spreadsheets/'' + config.spreadsheet_id +
    ''/values/'' + range + '':append'' +
    ''?valueInputOption='' + valueInputOption +
    ''&key='' + apiKey;
// 値がテンプレート文字列の場合は展開
let values = config.values;
if (typeof values === ''string'') {
    values = JSON.parse(renderTemplate(values, input));
}
const response = await ctx.http.post(url, { values: values }, {
    headers: { ''Content-Type'': ''application/json'' }
});
if (response.status === 404) {
    throw new Error(''[GSHEETS_003] スプレッドシートが見つかりません'');
}
if (response.status >= 400) {
    const errorMsg = response.body?.error?.message || ''Unknown error'';
    throw new Error(''[GSHEETS_002] 行追加失敗: '' + errorMsg);
}
return {
    updated_range: response.body.updates?.updatedRange,
    updated_rows: response.body.updates?.updatedRows,
    updated_cells: response.body.updates?.updatedCells
};
    ', '{"icon": "table", "color": "#0F9D58"}', true, 1);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('bcfba0d8-bc2e-4460-b385-16b3623912df', NULL, 'gsheets_read', 'Google Sheets: 読み取り', 'Google Sheetsから範囲を読み取り', 'integration', 'table', '{"type": "object", "required": ["spreadsheet_id", "range"], "properties": {"range": {"type": "string", "title": "範囲", "description": "読み取り範囲（例: Sheet1!A1:D10）"}, "api_key": {"type": "string", "title": "API Key", "description": "Google API Key（空の場合はシークレットGOOGLE_API_KEYを使用）"}, "spreadsheet_id": {"type": "string", "title": "スプレッドシートID"}, "major_dimension": {"enum": ["ROWS", "COLUMNS"], "type": "string", "title": "次元", "default": "ROWS"}}}', '{"type": "object"}', '{"type": "object", "properties": {"range": {"type": "string"}, "values": {"type": "array"}}}', '[{"code": "GSHEETS_001", "name": "API_KEY_NOT_CONFIGURED", "retryable": false, "description": "API Keyが設定されていません"}, {"code": "GSHEETS_003", "name": "INVALID_SPREADSHEET", "retryable": false, "description": "スプレッドシートが見つかりません"}, {"code": "GSHEETS_004", "name": "READ_FAILED", "retryable": true, "description": "読み取りに失敗しました"}]', true, '2026-01-12 09:23:30.221551+00', '2026-01-12 09:23:30.221551+00', '[]', '[]', '[]', false, '
const apiKey = config.api_key || ctx.secrets.GOOGLE_API_KEY;
if (!apiKey) {
    throw new Error(''[GSHEETS_001] API Keyが設定されていません'');
}
const range = encodeURIComponent(config.range);
const majorDimension = config.major_dimension || ''ROWS'';
const url = ''https://sheets.googleapis.com/v4/spreadsheets/'' + config.spreadsheet_id +
    ''/values/'' + range +
    ''?majorDimension='' + majorDimension +
    ''&key='' + apiKey;
const response = await ctx.http.get(url);
if (response.status === 404) {
    throw new Error(''[GSHEETS_003] スプレッドシートが見つかりません'');
}
if (response.status >= 400) {
    const errorMsg = response.body?.error?.message || ''Unknown error'';
    throw new Error(''[GSHEETS_004] 読み取り失敗: '' + errorMsg);
}
return {
    range: response.body.range,
    values: response.body.values || []
};
    ', '{"icon": "table", "color": "#0F9D58"}', true, 1);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('08877c27-e58f-413b-ab8e-bffdf113f269', NULL, 'github_create_issue', 'GitHub: Issue作成', 'GitHubリポジトリにIssueを作成', 'integration', 'git-pull-request', '{"type": "object", "required": ["owner", "repo", "title"], "properties": {"body": {"type": "string", "title": "本文", "description": "Issue本文（Markdown、テンプレート使用可）", "x-ui-widget": "textarea"}, "repo": {"type": "string", "title": "リポジトリ", "description": "リポジトリ名"}, "owner": {"type": "string", "title": "オーナー", "description": "リポジトリオーナー（ユーザー名または組織名）"}, "title": {"type": "string", "title": "タイトル", "description": "Issueタイトル（テンプレート使用可）"}, "token": {"type": "string", "title": "アクセストークン", "description": "GitHub Personal Access Token（空の場合はシークレットGITHUB_TOKENを使用）"}, "labels": {"type": "array", "items": {"type": "string"}, "title": "ラベル", "description": "ラベル名の配列"}, "assignees": {"type": "array", "items": {"type": "string"}, "title": "アサイン", "description": "アサインするユーザー名の配列"}}}', '{"type": "object"}', '{"type": "object", "properties": {"id": {"type": "number"}, "url": {"type": "string"}, "number": {"type": "number"}, "html_url": {"type": "string"}}}', '[{"code": "GITHUB_001", "name": "TOKEN_NOT_CONFIGURED", "retryable": false, "description": "トークンが設定されていません"}, {"code": "GITHUB_002", "name": "CREATE_FAILED", "retryable": true, "description": "Issue作成に失敗しました"}, {"code": "GITHUB_003", "name": "REPO_NOT_FOUND", "retryable": false, "description": "リポジトリが見つかりません"}]', true, '2026-01-12 09:23:30.221947+00', '2026-01-12 09:23:30.221947+00', '[]', '[]', '[]', false, '
const token = config.token || ctx.secrets.GITHUB_TOKEN;
if (!token) {
    throw new Error(''[GITHUB_001] トークンが設定されていません'');
}
const payload = {
    title: renderTemplate(config.title, input),
    body: config.body ? renderTemplate(config.body, input) : undefined,
    labels: config.labels,
    assignees: config.assignees
};
const url = ''https://api.github.com/repos/'' + config.owner + ''/'' + config.repo + ''/issues'';
const response = await ctx.http.post(url, payload, {
    headers: {
        ''Authorization'': ''Bearer '' + token,
        ''Accept'': ''application/vnd.github+json'',
        ''X-GitHub-Api-Version'': ''2022-11-28''
    }
});
if (response.status === 404) {
    throw new Error(''[GITHUB_003] リポジトリが見つかりません'');
}
if (response.status >= 400) {
    const errorMsg = response.body?.message || ''Unknown error'';
    throw new Error(''[GITHUB_002] Issue作成失敗: '' + errorMsg);
}
return {
    id: response.body.id,
    number: response.body.number,
    url: response.body.url,
    html_url: response.body.html_url
};
    ', '{"icon": "git-pull-request", "color": "#24292F"}', true, 1);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('825d2ae5-9808-496d-b4e0-dcf2f8e6afe5', NULL, 'github_add_comment', 'GitHub: コメント追加', 'GitHub IssueまたはPRにコメントを追加', 'integration', 'message-square', '{"type": "object", "required": ["owner", "repo", "issue_number", "body"], "properties": {"body": {"type": "string", "title": "コメント本文", "description": "コメント本文（Markdown、テンプレート使用可）", "x-ui-widget": "textarea"}, "repo": {"type": "string", "title": "リポジトリ"}, "owner": {"type": "string", "title": "オーナー"}, "token": {"type": "string", "title": "アクセストークン", "description": "GitHub Personal Access Token（空の場合はシークレットGITHUB_TOKENを使用）"}, "issue_number": {"type": "number", "title": "Issue/PR番号"}}}', '{"type": "object"}', '{"type": "object", "properties": {"id": {"type": "number"}, "url": {"type": "string"}, "html_url": {"type": "string"}}}', '[{"code": "GITHUB_001", "name": "TOKEN_NOT_CONFIGURED", "retryable": false, "description": "トークンが設定されていません"}, {"code": "GITHUB_004", "name": "COMMENT_FAILED", "retryable": true, "description": "コメント追加に失敗しました"}]', true, '2026-01-12 09:23:30.222311+00', '2026-01-12 09:23:30.222311+00', '[]', '[]', '[]', false, '
const token = config.token || ctx.secrets.GITHUB_TOKEN;
if (!token) {
    throw new Error(''[GITHUB_001] トークンが設定されていません'');
}
const url = ''https://api.github.com/repos/'' + config.owner + ''/'' + config.repo +
    ''/issues/'' + config.issue_number + ''/comments'';
const response = await ctx.http.post(url, {
    body: renderTemplate(config.body, input)
}, {
    headers: {
        ''Authorization'': ''Bearer '' + token,
        ''Accept'': ''application/vnd.github+json'',
        ''X-GitHub-Api-Version'': ''2022-11-28''
    }
});
if (response.status >= 400) {
    const errorMsg = response.body?.message || ''Unknown error'';
    throw new Error(''[GITHUB_004] コメント追加失敗: '' + errorMsg);
}
return {
    id: response.body.id,
    url: response.body.url,
    html_url: response.body.html_url
};
    ', '{"icon": "message-square", "color": "#24292F"}', true, 1);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('e1d99e22-5348-4e8b-9c44-a1a7e82cbb83', NULL, 'web_search', 'Web検索', 'Tavily APIでWeb検索を実行', 'integration', 'search', '{"type": "object", "required": ["query"], "properties": {"query": {"type": "string", "title": "検索クエリ", "description": "検索キーワード（テンプレート使用可）"}, "api_key": {"type": "string", "title": "API Key", "description": "Tavily API Key（空の場合はシークレットTAVILY_API_KEYを使用）"}, "max_results": {"type": "number", "title": "最大結果数", "default": 5, "maximum": 20, "minimum": 1}, "search_depth": {"enum": ["basic", "advanced"], "type": "string", "title": "検索深度", "default": "basic"}, "include_answer": {"type": "boolean", "title": "AI回答を含める", "default": true}, "exclude_domains": {"type": "array", "items": {"type": "string"}, "title": "除外ドメイン"}, "include_domains": {"type": "array", "items": {"type": "string"}, "title": "含めるドメイン"}}}', '{"type": "object"}', '{"type": "object", "properties": {"answer": {"type": "string"}, "results": {"type": "array", "items": {"type": "object", "properties": {"url": {"type": "string"}, "score": {"type": "number"}, "title": {"type": "string"}, "content": {"type": "string"}}}}}}', '[{"code": "SEARCH_001", "name": "API_KEY_NOT_CONFIGURED", "retryable": false, "description": "API Keyが設定されていません"}, {"code": "SEARCH_002", "name": "SEARCH_FAILED", "retryable": true, "description": "検索に失敗しました"}]', true, '2026-01-12 09:23:30.222703+00', '2026-01-12 09:23:30.222703+00', '[]', '[]', '[]', false, '
const apiKey = config.api_key || ctx.secrets.TAVILY_API_KEY;
if (!apiKey) {
    throw new Error(''[SEARCH_001] API Keyが設定されていません'');
}
const payload = {
    api_key: apiKey,
    query: renderTemplate(config.query, input),
    search_depth: config.search_depth || ''basic'',
    max_results: config.max_results || 5,
    include_answer: config.include_answer !== false
};
if (config.include_domains && config.include_domains.length > 0) {
    payload.include_domains = config.include_domains;
}
if (config.exclude_domains && config.exclude_domains.length > 0) {
    payload.exclude_domains = config.exclude_domains;
}
const response = await ctx.http.post(''https://api.tavily.com/search'', payload, {
    headers: { ''Content-Type'': ''application/json'' }
});
if (response.status >= 400) {
    throw new Error(''[SEARCH_002] 検索失敗: '' + (response.body?.error || response.status));
}
return {
    answer: response.body.answer,
    results: response.body.results
};
    ', '{"icon": "search", "color": "#4285F4"}', true, 1);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('fa2509a4-f899-4761-a20d-f67122fd2e1d', NULL, 'email_sendgrid', 'Email (SendGrid)', 'SendGrid APIでメールを送信', 'integration', 'mail', '{"type": "object", "required": ["from_email", "to_email", "subject", "content"], "properties": {"api_key": {"type": "string", "title": "API Key", "description": "SendGrid API Key（空の場合はシークレットSENDGRID_API_KEYを使用）"}, "content": {"type": "string", "title": "本文", "description": "メール本文（テンプレート使用可）", "x-ui-widget": "textarea"}, "subject": {"type": "string", "title": "件名", "description": "メール件名（テンプレート使用可）"}, "to_name": {"type": "string", "title": "受信者名"}, "to_email": {"type": "string", "title": "宛先メール", "description": "受信者のメールアドレス（テンプレート使用可）"}, "from_name": {"type": "string", "title": "送信者名"}, "from_email": {"type": "string", "title": "送信元メール", "description": "送信者のメールアドレス"}, "content_type": {"enum": ["text/plain", "text/html"], "type": "string", "title": "本文形式", "default": "text/plain"}}}', '{"type": "object"}', '{"type": "object", "properties": {"success": {"type": "boolean"}, "message_id": {"type": "string"}}}', '[{"code": "EMAIL_001", "name": "API_KEY_NOT_CONFIGURED", "retryable": false, "description": "API Keyが設定されていません"}, {"code": "EMAIL_002", "name": "SEND_FAILED", "retryable": true, "description": "メール送信に失敗しました"}, {"code": "EMAIL_003", "name": "INVALID_EMAIL", "retryable": false, "description": "メールアドレスが無効です"}]', true, '2026-01-12 09:23:30.223087+00', '2026-01-12 09:23:30.223087+00', '[]', '[]', '[]', false, '
const apiKey = config.api_key || ctx.secrets.SENDGRID_API_KEY;
if (!apiKey) {
    throw new Error(''[EMAIL_001] API Keyが設定されていません'');
}
const payload = {
    personalizations: [{
        to: [{
            email: renderTemplate(config.to_email, input),
            name: config.to_name ? renderTemplate(config.to_name, input) : undefined
        }]
    }],
    from: {
        email: config.from_email,
        name: config.from_name
    },
    subject: renderTemplate(config.subject, input),
    content: [{
        type: config.content_type || ''text/plain'',
        value: renderTemplate(config.content, input)
    }]
};
const response = await ctx.http.post(''https://api.sendgrid.com/v3/mail/send'', payload, {
    headers: {
        ''Authorization'': ''Bearer '' + apiKey,
        ''Content-Type'': ''application/json''
    }
});
if (response.status >= 400) {
    const errors = response.body?.errors?.map(e => e.message).join('', '') || ''Unknown error'';
    throw new Error(''[EMAIL_002] メール送信失敗: '' + errors);
}
return {
    success: true,
    message_id: response.headers[''x-message-id'']
};
    ', '{"icon": "mail", "color": "#1A82E2"}', true, 1);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('02fc89f3-62d0-404b-bd10-9dcf9e3ac1c0', NULL, 'linear_create_issue', 'Linear: Issue作成', 'LinearにIssueを作成', 'integration', 'check-square', '{"type": "object", "required": ["team_id", "title"], "properties": {"title": {"type": "string", "title": "タイトル", "description": "Issueタイトル（テンプレート使用可）"}, "api_key": {"type": "string", "title": "API Key", "description": "Linear API Key（空の場合はシークレットLINEAR_API_KEYを使用）"}, "team_id": {"type": "string", "title": "チームID", "description": "LinearチームのID"}, "priority": {"enum": [0, 1, 2, 3, 4], "type": "number", "title": "優先度", "default": 0, "description": "0=なし, 1=緊急, 2=高, 3=中, 4=低"}, "label_ids": {"type": "array", "items": {"type": "string"}, "title": "ラベルID"}, "assignee_id": {"type": "string", "title": "担当者ID"}, "description": {"type": "string", "title": "説明", "description": "Issue説明（Markdown、テンプレート使用可）", "x-ui-widget": "textarea"}}}', '{"type": "object"}', '{"type": "object", "properties": {"id": {"type": "string"}, "url": {"type": "string"}, "identifier": {"type": "string"}}}', '[{"code": "LINEAR_001", "name": "API_KEY_NOT_CONFIGURED", "retryable": false, "description": "API Keyが設定されていません"}, {"code": "LINEAR_002", "name": "CREATE_FAILED", "retryable": true, "description": "Issue作成に失敗しました"}]', true, '2026-01-12 09:23:30.223449+00', '2026-01-12 09:23:30.223449+00', '[]', '[]', '[]', false, '
const apiKey = config.api_key || ctx.secrets.LINEAR_API_KEY;
if (!apiKey) {
    throw new Error(''[LINEAR_001] API Keyが設定されていません'');
}
const mutation = `
mutation IssueCreate($input: IssueCreateInput!) {
    issueCreate(input: $input) {
        success
        issue {
            id
            identifier
            url
        }
    }
}`;
const variables = {
    input: {
        teamId: config.team_id,
        title: renderTemplate(config.title, input),
        description: config.description ? renderTemplate(config.description, input) : undefined,
        priority: config.priority,
        labelIds: config.label_ids,
        assigneeId: config.assignee_id
    }
};
const response = await ctx.http.post(''https://api.linear.app/graphql'', {
    query: mutation,
    variables: variables
}, {
    headers: {
        ''Authorization'': apiKey,
        ''Content-Type'': ''application/json''
    }
});
if (response.status >= 400 || response.body.errors) {
    const errorMsg = response.body.errors?.[0]?.message || ''Unknown error'';
    throw new Error(''[LINEAR_002] Issue作成失敗: '' + errorMsg);
}
const issue = response.body.data.issueCreate.issue;
return {
    id: issue.id,
    identifier: issue.identifier,
    url: issue.url
};
    ', '{"icon": "check-square", "color": "#5E6AD2"}', true, 1);
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, created_at, updated_at, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES ('574b1d48-1b80-4d29-89b4-68c33a9b8976', NULL, 'llm', 'LLM', 'Execute LLM prompts with various providers', 'ai', 'brain', '{"type": "object", "required": ["provider", "model", "user_prompt"], "properties": {"model": {"type": "string", "title": "モデル", "description": "使用するモデル名"}, "provider": {"enum": ["openai", "anthropic", "mock"], "type": "string", "title": "プロバイダー", "default": "openai", "description": "使用するLLMプロバイダーを選択"}, "max_tokens": {"type": "integer", "title": "最大トークン数", "default": 4096, "maximum": 128000, "minimum": 1, "description": "生成する最大トークン数"}, "temperature": {"type": "number", "title": "Temperature", "default": 0.7, "maximum": 2, "minimum": 0, "description": "出力の多様性を制御（0: 決定的、2: 創造的）"}, "user_prompt": {"type": "string", "title": "ユーザープロンプト", "maxLength": 50000, "description": "{{変数名}}で入力データを参照可能"}, "system_prompt": {"type": "string", "title": "システムプロンプト", "maxLength": 10000, "description": "AIの振る舞いを定義するプロンプト"}}}', NULL, NULL, '[{"code": "LLM_001", "name": "RATE_LIMIT", "retryable": true, "description": "Rate limit exceeded"}, {"code": "LLM_002", "name": "INVALID_MODEL", "retryable": false, "description": "Invalid model specified"}, {"code": "LLM_003", "name": "TOKEN_LIMIT", "retryable": false, "description": "Token limit exceeded"}, {"code": "LLM_004", "name": "API_ERROR", "retryable": true, "description": "LLM API error"}]', true, '2026-01-12 09:23:29.067836+00', '2026-01-12 09:23:30.543213+00', '[{"name": "output", "label": "Output", "schema": {"type": "object", "properties": {"content": {"type": "string"}, "tokens_used": {"type": "number"}}}, "is_default": true, "description": "LLM response"}]', '[{"name": "input", "label": "Input", "schema": {"type": "any"}, "required": false, "description": "Data available for prompt template"}]', '[{"name": "llm_api_key", "type": "api_key", "scope": "system", "required": true, "description": "LLM Provider API Key"}]', false, '
const prompt = renderTemplate(config.user_prompt || '''', input);
const systemPrompt = config.system_prompt || '''';
const response = await ctx.llm.chat(config.provider, config.model, {
    messages: [
        ...(systemPrompt ? [{ role: ''system'', content: systemPrompt }] : []),
        { role: ''user'', content: prompt }
    ],
    temperature: config.temperature ?? 0.7,
    maxTokens: config.max_tokens ?? 1000
});
return {
    content: response.content,
    usage: response.usage
};
', '{"icon": "brain", "color": "#8B5CF6", "groups": [{"id": "model", "icon": "🤖", "title": "モデル設定"}, {"id": "prompt", "icon": "💬", "title": "プロンプト"}, {"id": "params", "icon": "⚙️", "title": "パラメータ", "collapsed": true}], "fieldGroups": {"model": "model", "provider": "model", "max_tokens": "params", "temperature": "params", "user_prompt": "prompt", "system_prompt": "prompt"}, "fieldOverrides": {"user_prompt": {"rows": 8, "widget": "textarea"}, "system_prompt": {"rows": 4, "widget": "textarea"}}}', true, 1);
INSERT INTO workflows (id, tenant_id, name, description, status, version, input_schema, output_schema, draft, created_by, published_at, created_at, updated_at, deleted_at, is_system, system_slug) VALUES ('a0000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'Copilot: Generate Workflow', 'Generates workflow structures from natural language descriptions using AI', 'published', 1, NULL, NULL, NULL, NULL, '2026-01-12 09:23:30.439548+00', '2026-01-12 09:23:30.439548+00', '2026-01-12 09:23:30.439548+00', NULL, true, 'copilot-generate');
INSERT INTO workflows (id, tenant_id, name, description, status, version, input_schema, output_schema, draft, created_by, published_at, created_at, updated_at, deleted_at, is_system, system_slug) VALUES ('a0000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'Copilot: Suggest Steps', 'Suggests next steps for a workflow based on current structure', 'published', 1, NULL, NULL, NULL, NULL, '2026-01-12 09:23:30.443322+00', '2026-01-12 09:23:30.443322+00', '2026-01-12 09:23:30.443322+00', NULL, true, 'copilot-suggest');
INSERT INTO workflows (id, tenant_id, name, description, status, version, input_schema, output_schema, draft, created_by, published_at, created_at, updated_at, deleted_at, is_system, system_slug) VALUES ('a0000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'Copilot: Diagnose Error', 'Diagnoses workflow execution errors and suggests fixes', 'published', 1, NULL, NULL, NULL, NULL, '2026-01-12 09:23:30.444644+00', '2026-01-12 09:23:30.444644+00', '2026-01-12 09:23:30.444644+00', NULL, true, 'copilot-diagnose');
INSERT INTO workflows (id, tenant_id, name, description, status, version, input_schema, output_schema, draft, created_by, published_at, created_at, updated_at, deleted_at, is_system, system_slug) VALUES ('a0000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'Copilot: Optimize Workflow', 'Suggests optimizations for workflow performance, cost, and reliability', 'published', 1, NULL, NULL, NULL, NULL, '2026-01-12 09:23:30.445469+00', '2026-01-12 09:23:30.445469+00', '2026-01-12 09:23:30.445469+00', NULL, true, 'copilot-optimize');
INSERT INTO steps (id, workflow_id, name, type, config, position_x, position_y, created_at, updated_at, block_group_id, group_role, credential_bindings, block_definition_id) VALUES ('b0000001-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'Start', 'start', '{}', 400, 50, '2026-01-12 09:23:30.44145+00', '2026-01-12 09:23:30.44145+00', NULL, NULL, '{}', NULL);
INSERT INTO steps (id, workflow_id, name, type, config, position_x, position_y, created_at, updated_at, block_group_id, group_role, credential_bindings, block_definition_id) VALUES ('b0000002-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'Get Available Blocks', 'function', '{"code": "const blocks = context.blocks.list(); return { blocks: blocks.map(b => ({ slug: b.slug, name: b.name, description: b.description, category: b.category })) };", "language": "javascript"}', 400, 200, '2026-01-12 09:23:30.44145+00', '2026-01-12 09:23:30.44145+00', NULL, NULL, '{}', NULL);
INSERT INTO steps (id, workflow_id, name, type, config, position_x, position_y, created_at, updated_at, block_group_id, group_role, credential_bindings, block_definition_id) VALUES ('b0000003-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'Build Prompt', 'function', '{"code": "const blocksInfo = input.blocks.map(b => `- ${b.slug}: ${b.name} (${b.category}) - ${b.description || \"\"}`).join(\"\\n\");\nconst prompt = `You are an AI workflow generator. Generate a workflow based on the user description.\\n\\n## Available Blocks\\n${blocksInfo}\\n\\n## Available Step Types\\n- start: Entry point (required)\\n- llm: AI/LLM call\\n- tool: External adapter\\n- condition: Binary branch (true/false)\\n- switch: Multi-way branch\\n- map: Parallel array processing\\n- loop: Iteration\\n- wait: Delay\\n- function: Custom JavaScript\\n- log: Debug logging\\n\\n## User Request\\n${input.prompt}\\n\\n## Output Format (JSON)\\n{\\n  \"response\": \"Explanation\",\\n  \"steps\": [{\"temp_id\": \"step_1\", \"name\": \"Step Name\", \"type\": \"start\", \"description\": \"\", \"config\": {}, \"position_x\": 400, \"position_y\": 50}],\\n  \"edges\": [{\"source_temp_id\": \"step_1\", \"target_temp_id\": \"step_2\", \"source_port\": \"default\"}],\\n  \"start_step_id\": \"step_1\"\\n}\\n\\nGenerate a valid workflow JSON. Always include a start step.`;\nreturn { prompt: prompt };", "language": "javascript"}', 400, 350, '2026-01-12 09:23:30.44145+00', '2026-01-12 09:23:30.44145+00', NULL, NULL, '{}', NULL);
INSERT INTO steps (id, workflow_id, name, type, config, position_x, position_y, created_at, updated_at, block_group_id, group_role, credential_bindings, block_definition_id) VALUES ('b0000004-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'Generate with LLM', 'llm', '{"model": "gpt-4o-mini", "provider": "openai", "max_tokens": 4000, "temperature": 0.3, "user_prompt": "{{$.prompt}}", "system_prompt": "You are an AI workflow generator. Always respond with valid JSON."}', 400, 500, '2026-01-12 09:23:30.44145+00', '2026-01-12 09:23:30.44145+00', NULL, NULL, '{}', NULL);
INSERT INTO steps (id, workflow_id, name, type, config, position_x, position_y, created_at, updated_at, block_group_id, group_role, credential_bindings, block_definition_id) VALUES ('b0000005-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'Parse & Validate', 'function', '{"code": "try { let content = input.content || \"\"; if (content.startsWith(\"```json\")) content = content.slice(7); if (content.startsWith(\"```\")) content = content.slice(3); if (content.endsWith(\"```\")) content = content.slice(0, -3); content = content.trim(); const result = JSON.parse(content); if (!result.steps || !Array.isArray(result.steps)) { return { error: \"Invalid workflow: missing steps array\" }; } const validTypes = [\"start\", \"llm\", \"tool\", \"condition\", \"switch\", \"map\", \"join\", \"subflow\", \"loop\", \"wait\", \"function\", \"router\", \"human_in_loop\", \"filter\", \"split\", \"aggregate\", \"error\", \"note\", \"log\"]; result.steps = result.steps.filter(s => validTypes.includes(s.type)); return result; } catch (e) { return { error: \"Failed to parse LLM response: \" + e.message }; }", "language": "javascript"}', 400, 650, '2026-01-12 09:23:30.44145+00', '2026-01-12 09:23:30.44145+00', NULL, NULL, '{}', NULL);
INSERT INTO steps (id, workflow_id, name, type, config, position_x, position_y, created_at, updated_at, block_group_id, group_role, credential_bindings, block_definition_id) VALUES ('b0000001-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000002', 'Start', 'start', '{}', 400, 50, '2026-01-12 09:23:30.443723+00', '2026-01-12 09:23:30.443723+00', NULL, NULL, '{}', NULL);
INSERT INTO steps (id, workflow_id, name, type, config, position_x, position_y, created_at, updated_at, block_group_id, group_role, credential_bindings, block_definition_id) VALUES ('b0000002-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000002', 'Get Workflow Context', 'function', '{"code": "const workflow = context.workflows.get(input.workflow_id); const blocks = context.blocks.list(); return { workflow: workflow, blocks: blocks };", "language": "javascript"}', 400, 200, '2026-01-12 09:23:30.443723+00', '2026-01-12 09:23:30.443723+00', NULL, NULL, '{}', NULL);
INSERT INTO steps (id, workflow_id, name, type, config, position_x, position_y, created_at, updated_at, block_group_id, group_role, credential_bindings, block_definition_id) VALUES ('b0000003-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000002', 'Build Suggest Prompt', 'function', '{"code": "const wf = input.workflow; const blocksInfo = input.blocks.slice(0, 20).map(b => `- ${b.slug}: ${b.name}`).join(\"\\n\"); const stepsInfo = (wf.steps || []).map(s => `- ${s.name} (${s.type})`).join(\"\\n\"); const prompt = `Suggest 2-3 next steps for this workflow.\\n\\n## Current Steps\\n${stepsInfo || \"(empty)\"}\\n\\n## Available Blocks\\n${blocksInfo}\\n\\n## Context\\n${input.context || \"\"}\\n\\nReturn JSON array: [{\"type\": \"...\", \"name\": \"...\", \"description\": \"...\", \"config\": {}, \"reason\": \"...\"}]`; return { prompt: prompt };", "language": "javascript"}', 400, 350, '2026-01-12 09:23:30.443723+00', '2026-01-12 09:23:30.443723+00', NULL, NULL, '{}', NULL);
INSERT INTO steps (id, workflow_id, name, type, config, position_x, position_y, created_at, updated_at, block_group_id, group_role, credential_bindings, block_definition_id) VALUES ('b0000004-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000002', 'Suggest with LLM', 'llm', '{"model": "gpt-4o-mini", "provider": "openai", "max_tokens": 2000, "temperature": 0.5, "user_prompt": "{{$.prompt}}", "system_prompt": "You are an AI workflow assistant. Return valid JSON array."}', 400, 500, '2026-01-12 09:23:30.443723+00', '2026-01-12 09:23:30.443723+00', NULL, NULL, '{}', NULL);
INSERT INTO steps (id, workflow_id, name, type, config, position_x, position_y, created_at, updated_at, block_group_id, group_role, credential_bindings, block_definition_id) VALUES ('b0000005-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000002', 'Parse Suggestions', 'function', '{"code": "try { let content = input.content || \"\"; if (content.startsWith(\"```\")) { content = content.replace(/```json?\\n?/g, \"\").replace(/```/g, \"\").trim(); } const suggestions = JSON.parse(content); return { suggestions: Array.isArray(suggestions) ? suggestions : [] }; } catch (e) { return { suggestions: [] }; }", "language": "javascript"}', 400, 650, '2026-01-12 09:23:30.443723+00', '2026-01-12 09:23:30.443723+00', NULL, NULL, '{}', NULL);
INSERT INTO steps (id, workflow_id, name, type, config, position_x, position_y, created_at, updated_at, block_group_id, group_role, credential_bindings, block_definition_id) VALUES ('b0000001-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000003', 'Start', 'start', '{}', 400, 50, '2026-01-12 09:23:30.444897+00', '2026-01-12 09:23:30.444897+00', NULL, NULL, '{}', NULL);
INSERT INTO steps (id, workflow_id, name, type, config, position_x, position_y, created_at, updated_at, block_group_id, group_role, credential_bindings, block_definition_id) VALUES ('b0000002-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000003', 'Get Run Details', 'function', '{"code": "const run = context.runs.get(input.run_id); const stepRuns = context.runs.getStepRuns(input.run_id); const failedSteps = stepRuns.filter(sr => sr.status === \"failed\"); return { run: run, stepRuns: stepRuns, failedSteps: failedSteps };", "language": "javascript"}', 400, 200, '2026-01-12 09:23:30.444897+00', '2026-01-12 09:23:30.444897+00', NULL, NULL, '{}', NULL);
INSERT INTO steps (id, workflow_id, name, type, config, position_x, position_y, created_at, updated_at, block_group_id, group_role, credential_bindings, block_definition_id) VALUES ('b0000003-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000003', 'Build Diagnose Prompt', 'function', '{"code": "const failedInfo = input.failedSteps.map(sr => `Step: ${sr.step_name}\\nError: ${sr.error || \"Unknown\"}\\nInput: ${JSON.stringify(sr.input || {})}`).join(\"\\n\\n\"); const prompt = `Diagnose this workflow error.\\n\\n## Run Status: ${input.run.status}\\n\\n## Failed Steps\\n${failedInfo || \"No failures found\"}\\n\\nReturn JSON: {\"diagnosis\": {\"root_cause\": \"...\", \"category\": \"config_error|input_error|api_error|logic_error|timeout|unknown\", \"severity\": \"high|medium|low\"}, \"fixes\": [{\"description\": \"...\", \"steps\": [\"...\"]}], \"preventions\": [\"...\"]}`; return { prompt: prompt };", "language": "javascript"}', 400, 350, '2026-01-12 09:23:30.444897+00', '2026-01-12 09:23:30.444897+00', NULL, NULL, '{}', NULL);
INSERT INTO steps (id, workflow_id, name, type, config, position_x, position_y, created_at, updated_at, block_group_id, group_role, credential_bindings, block_definition_id) VALUES ('b0000004-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000003', 'Diagnose with LLM', 'llm', '{"model": "gpt-4o-mini", "provider": "openai", "max_tokens": 2000, "temperature": 0.3, "user_prompt": "{{$.prompt}}", "system_prompt": "You are an AI debugging assistant. Return valid JSON."}', 400, 500, '2026-01-12 09:23:30.444897+00', '2026-01-12 09:23:30.444897+00', NULL, NULL, '{}', NULL);
INSERT INTO steps (id, workflow_id, name, type, config, position_x, position_y, created_at, updated_at, block_group_id, group_role, credential_bindings, block_definition_id) VALUES ('b0000005-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000003', 'Parse Diagnosis', 'function', '{"code": "try { let content = input.content || \"\"; if (content.startsWith(\"```\")) { content = content.replace(/```json?\\n?/g, \"\").replace(/```/g, \"\").trim(); } return JSON.parse(content); } catch (e) { return { diagnosis: { root_cause: \"Parse error\", category: \"unknown\", severity: \"low\" }, fixes: [], preventions: [] }; }", "language": "javascript"}', 400, 650, '2026-01-12 09:23:30.444897+00', '2026-01-12 09:23:30.444897+00', NULL, NULL, '{}', NULL);
INSERT INTO steps (id, workflow_id, name, type, config, position_x, position_y, created_at, updated_at, block_group_id, group_role, credential_bindings, block_definition_id) VALUES ('b0000001-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000004', 'Start', 'start', '{}', 400, 50, '2026-01-12 09:23:30.445696+00', '2026-01-12 09:23:30.445696+00', NULL, NULL, '{}', NULL);
INSERT INTO steps (id, workflow_id, name, type, config, position_x, position_y, created_at, updated_at, block_group_id, group_role, credential_bindings, block_definition_id) VALUES ('b0000002-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000004', 'Get Workflow Details', 'function', '{"code": "const workflow = context.workflows.get(input.workflow_id); return { workflow: workflow };", "language": "javascript"}', 400, 200, '2026-01-12 09:23:30.445696+00', '2026-01-12 09:23:30.445696+00', NULL, NULL, '{}', NULL);
INSERT INTO steps (id, workflow_id, name, type, config, position_x, position_y, created_at, updated_at, block_group_id, group_role, credential_bindings, block_definition_id) VALUES ('b0000003-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000004', 'Build Optimize Prompt', 'function', '{"code": "const wf = input.workflow; const stepsInfo = (wf.steps || []).map(s => `- ${s.name} (${s.type}): ${JSON.stringify(s.config || {})}`).join(\"\\n\"); const prompt = `Suggest optimizations for this workflow.\\n\\n## Workflow: ${wf.name}\\n## Steps (${(wf.steps || []).length})\\n${stepsInfo}\\n\\nReturn JSON: {\"optimizations\": [{\"category\": \"performance|cost|reliability|maintainability\", \"title\": \"...\", \"description\": \"...\", \"impact\": \"high|medium|low\", \"effort\": \"high|medium|low\"}], \"summary\": \"...\"}`; return { prompt: prompt };", "language": "javascript"}', 400, 350, '2026-01-12 09:23:30.445696+00', '2026-01-12 09:23:30.445696+00', NULL, NULL, '{}', NULL);
INSERT INTO steps (id, workflow_id, name, type, config, position_x, position_y, created_at, updated_at, block_group_id, group_role, credential_bindings, block_definition_id) VALUES ('b0000004-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000004', 'Optimize with LLM', 'llm', '{"model": "gpt-4o-mini", "provider": "openai", "max_tokens": 2000, "temperature": 0.5, "user_prompt": "{{$.prompt}}", "system_prompt": "You are an AI optimization assistant. Return valid JSON."}', 400, 500, '2026-01-12 09:23:30.445696+00', '2026-01-12 09:23:30.445696+00', NULL, NULL, '{}', NULL);
INSERT INTO steps (id, workflow_id, name, type, config, position_x, position_y, created_at, updated_at, block_group_id, group_role, credential_bindings, block_definition_id) VALUES ('b0000005-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000004', 'Parse Optimizations', 'function', '{"code": "try { let content = input.content || \"\"; if (content.startsWith(\"```\")) { content = content.replace(/```json?\\n?/g, \"\").replace(/```/g, \"\").trim(); } return JSON.parse(content); } catch (e) { return { optimizations: [], summary: \"Parse error\" }; }", "language": "javascript"}', 400, 650, '2026-01-12 09:23:30.445696+00', '2026-01-12 09:23:30.445696+00', NULL, NULL, '{}', NULL);
INSERT INTO edges (id, workflow_id, source_step_id, target_step_id, condition, created_at, source_port, target_port) VALUES ('c0000001-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'b0000001-0000-0000-0000-000000000001', 'b0000002-0000-0000-0000-000000000001', NULL, '2026-01-12 09:23:30.442384+00', 'default', '');
INSERT INTO edges (id, workflow_id, source_step_id, target_step_id, condition, created_at, source_port, target_port) VALUES ('c0000002-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'b0000002-0000-0000-0000-000000000001', 'b0000003-0000-0000-0000-000000000001', NULL, '2026-01-12 09:23:30.442384+00', 'default', '');
INSERT INTO edges (id, workflow_id, source_step_id, target_step_id, condition, created_at, source_port, target_port) VALUES ('c0000003-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'b0000003-0000-0000-0000-000000000001', 'b0000004-0000-0000-0000-000000000001', NULL, '2026-01-12 09:23:30.442384+00', 'default', '');
INSERT INTO edges (id, workflow_id, source_step_id, target_step_id, condition, created_at, source_port, target_port) VALUES ('c0000004-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'b0000004-0000-0000-0000-000000000001', 'b0000005-0000-0000-0000-000000000001', NULL, '2026-01-12 09:23:30.442384+00', 'default', '');
INSERT INTO edges (id, workflow_id, source_step_id, target_step_id, condition, created_at, source_port, target_port) VALUES ('c0000001-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000002', 'b0000001-0000-0000-0000-000000000002', 'b0000002-0000-0000-0000-000000000002', NULL, '2026-01-12 09:23:30.444173+00', 'default', '');
INSERT INTO edges (id, workflow_id, source_step_id, target_step_id, condition, created_at, source_port, target_port) VALUES ('c0000002-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000002', 'b0000002-0000-0000-0000-000000000002', 'b0000003-0000-0000-0000-000000000002', NULL, '2026-01-12 09:23:30.444173+00', 'default', '');
INSERT INTO edges (id, workflow_id, source_step_id, target_step_id, condition, created_at, source_port, target_port) VALUES ('c0000003-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000002', 'b0000003-0000-0000-0000-000000000002', 'b0000004-0000-0000-0000-000000000002', NULL, '2026-01-12 09:23:30.444173+00', 'default', '');
INSERT INTO edges (id, workflow_id, source_step_id, target_step_id, condition, created_at, source_port, target_port) VALUES ('c0000004-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000002', 'b0000004-0000-0000-0000-000000000002', 'b0000005-0000-0000-0000-000000000002', NULL, '2026-01-12 09:23:30.444173+00', 'default', '');
INSERT INTO edges (id, workflow_id, source_step_id, target_step_id, condition, created_at, source_port, target_port) VALUES ('c0000001-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000003', 'b0000001-0000-0000-0000-000000000003', 'b0000002-0000-0000-0000-000000000003', NULL, '2026-01-12 09:23:30.445176+00', 'default', '');
INSERT INTO edges (id, workflow_id, source_step_id, target_step_id, condition, created_at, source_port, target_port) VALUES ('c0000002-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000003', 'b0000002-0000-0000-0000-000000000003', 'b0000003-0000-0000-0000-000000000003', NULL, '2026-01-12 09:23:30.445176+00', 'default', '');
INSERT INTO edges (id, workflow_id, source_step_id, target_step_id, condition, created_at, source_port, target_port) VALUES ('c0000003-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000003', 'b0000003-0000-0000-0000-000000000003', 'b0000004-0000-0000-0000-000000000003', NULL, '2026-01-12 09:23:30.445176+00', 'default', '');
INSERT INTO edges (id, workflow_id, source_step_id, target_step_id, condition, created_at, source_port, target_port) VALUES ('c0000004-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000003', 'b0000004-0000-0000-0000-000000000003', 'b0000005-0000-0000-0000-000000000003', NULL, '2026-01-12 09:23:30.445176+00', 'default', '');
INSERT INTO edges (id, workflow_id, source_step_id, target_step_id, condition, created_at, source_port, target_port) VALUES ('c0000001-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000004', 'b0000001-0000-0000-0000-000000000004', 'b0000002-0000-0000-0000-000000000004', NULL, '2026-01-12 09:23:30.446+00', 'default', '');
INSERT INTO edges (id, workflow_id, source_step_id, target_step_id, condition, created_at, source_port, target_port) VALUES ('c0000002-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000004', 'b0000002-0000-0000-0000-000000000004', 'b0000003-0000-0000-0000-000000000004', NULL, '2026-01-12 09:23:30.446+00', 'default', '');
INSERT INTO edges (id, workflow_id, source_step_id, target_step_id, condition, created_at, source_port, target_port) VALUES ('c0000003-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000004', 'b0000003-0000-0000-0000-000000000004', 'b0000004-0000-0000-0000-000000000004', NULL, '2026-01-12 09:23:30.446+00', 'default', '');
INSERT INTO edges (id, workflow_id, source_step_id, target_step_id, condition, created_at, source_port, target_port) VALUES ('c0000004-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000004', 'b0000004-0000-0000-0000-000000000004', 'b0000005-0000-0000-0000-000000000004', NULL, '2026-01-12 09:23:30.446+00', 'default', '');

-- ============================================================================
-- RAG (Retrieval-Augmented Generation) Block Definitions
-- ============================================================================

-- embedding: Convert text to vector embeddings
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES
('rag00001-0000-0000-0000-000000000001', NULL, 'embedding', 'Embedding', 'Convert text to vector embeddings', 'ai', 'hash',
'{"type": "object", "properties": {"provider": {"type": "string", "enum": ["openai", "cohere", "voyage"], "default": "openai", "title": "Provider"}, "model": {"type": "string", "default": "text-embedding-3-small", "title": "Model"}}}',
'{"type": "object", "properties": {"documents": {"type": "array", "items": {"type": "object", "properties": {"content": {"type": "string"}}}}, "text": {"type": "string"}, "texts": {"type": "array", "items": {"type": "string"}}}}',
'{"type": "object", "properties": {"documents": {"type": "array"}, "vectors": {"type": "array"}, "dimension": {"type": "integer"}}}',
'[{"code": "EMB_001", "name": "PROVIDER_ERROR", "retryable": true, "description": "Embedding provider API error"}, {"code": "EMB_002", "name": "EMPTY_INPUT", "retryable": false, "description": "No text provided for embedding"}]',
true,
'[{"name": "output", "label": "Output", "is_default": true, "description": "Documents with vectors"}]',
'[{"name": "input", "label": "Input", "schema": {"type": "object"}, "required": true, "description": "Documents or text to embed"}]',
'[]', false,
'const documents = input.documents || (input.texts ? input.texts.map(t => ({content: t})) : (input.text ? [{content: input.text}] : []));
if (documents.length === 0) throw new Error(''[EMB_002] No text provided for embedding'');
const provider = config.provider || ''openai'';
const model = config.model || ''text-embedding-3-small'';
const texts = documents.map(d => d.content);
const result = await ctx.embedding.embed(provider, model, texts);
const docsWithVectors = documents.map((doc, i) => ({...doc, vector: result.vectors[i]}));
return {documents: docsWithVectors, vectors: result.vectors, model: result.model, dimension: result.dimension, usage: result.usage};',
'{"icon": "hash", "color": "#8B5CF6"}', true, 1);

-- vector-upsert: Store documents in vector database
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES
('rag00002-0000-0000-0000-000000000001', NULL, 'vector-upsert', 'Vector Upsert', 'Store documents in vector database', 'data', 'database',
'{"type": "object", "required": ["collection"], "properties": {"collection": {"type": "string", "title": "Collection Name"}, "embedding_provider": {"type": "string", "default": "openai", "title": "Embedding Provider"}, "embedding_model": {"type": "string", "default": "text-embedding-3-small", "title": "Embedding Model"}}}',
'{"type": "object", "required": ["documents"], "properties": {"documents": {"type": "array", "items": {"type": "object", "properties": {"content": {"type": "string"}, "metadata": {"type": "object"}, "vector": {"type": "array"}}}}}}',
'{"type": "object", "properties": {"upserted_count": {"type": "integer"}, "ids": {"type": "array", "items": {"type": "string"}}}}',
'[{"code": "VEC_001", "name": "COLLECTION_REQUIRED", "retryable": false, "description": "Collection name is required"}, {"code": "VEC_002", "name": "DOCUMENTS_REQUIRED", "retryable": false, "description": "Documents array is required"}]',
true,
'[{"name": "output", "label": "Output", "is_default": true, "description": "Upsert result"}]',
'[{"name": "documents", "label": "Documents", "schema": {"type": "array"}, "required": true, "description": "Documents to store"}]',
'[]', false,
'const collection = config.collection || input.collection;
if (!collection) throw new Error(''[VEC_001] Collection name is required'');
const documents = input.documents;
if (!documents || documents.length === 0) throw new Error(''[VEC_002] Documents array is required'');
const result = await ctx.vector.upsert(collection, documents, {embedding_provider: config.embedding_provider, embedding_model: config.embedding_model});
return {collection, upserted_count: result.upserted_count, ids: result.ids};',
'{"icon": "database", "color": "#10B981"}', true, 1);

-- vector-search: Search for similar documents
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES
('rag00003-0000-0000-0000-000000000001', NULL, 'vector-search', 'Vector Search', 'Search for similar documents in vector database', 'data', 'search',
'{"type": "object", "required": ["collection"], "properties": {"collection": {"type": "string", "title": "Collection Name"}, "top_k": {"type": "integer", "default": 5, "minimum": 1, "maximum": 100, "title": "Number of Results"}, "threshold": {"type": "number", "minimum": 0, "maximum": 1, "title": "Similarity Threshold"}, "include_content": {"type": "boolean", "default": true, "title": "Include Content"}, "embedding_provider": {"type": "string", "default": "openai"}, "embedding_model": {"type": "string", "default": "text-embedding-3-small"}}}',
'{"type": "object", "properties": {"vector": {"type": "array", "items": {"type": "number"}}, "query": {"type": "string"}}}',
'{"type": "object", "properties": {"matches": {"type": "array"}, "count": {"type": "integer"}}}',
'[{"code": "VEC_001", "name": "COLLECTION_REQUIRED", "retryable": false, "description": "Collection name is required"}, {"code": "VEC_003", "name": "VECTOR_OR_QUERY_REQUIRED", "retryable": false, "description": "Either vector or query text is required"}]',
true,
'[{"name": "output", "label": "Output", "is_default": true, "description": "Search results"}]',
'[{"name": "input", "label": "Input", "schema": {"type": "object"}, "required": true, "description": "Vector or query text"}]',
'[]', false,
'const collection = config.collection || input.collection;
if (!collection) throw new Error(''[VEC_001] Collection name is required'');
let searchVector = input.vector || (input.vectors ? input.vectors[0] : null);
if (!searchVector && input.query) {
  const provider = config.embedding_provider || ''openai'';
  const model = config.embedding_model || ''text-embedding-3-small'';
  const embedResult = await ctx.embedding.embed(provider, model, [input.query]);
  searchVector = embedResult.vectors[0];
}
if (!searchVector) throw new Error(''[VEC_003] Either vector or query text is required'');
const result = await ctx.vector.query(collection, searchVector, {top_k: config.top_k || 5, threshold: config.threshold, include_content: config.include_content !== false});
return {matches: result.matches, count: result.matches.length, collection};',
'{"icon": "search", "color": "#3B82F6"}', true, 1);

-- vector-delete: Delete documents from vector database
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES
('rag00004-0000-0000-0000-000000000001', NULL, 'vector-delete', 'Vector Delete', 'Delete documents from vector database', 'data', 'trash-2',
'{"type": "object", "required": ["collection"], "properties": {"collection": {"type": "string", "title": "Collection Name"}}}',
'{"type": "object", "required": ["ids"], "properties": {"ids": {"type": "array", "items": {"type": "string"}}}}',
'{"type": "object", "properties": {"deleted_count": {"type": "integer"}}}',
'[{"code": "VEC_001", "name": "COLLECTION_REQUIRED", "retryable": false, "description": "Collection name is required"}, {"code": "VEC_004", "name": "IDS_REQUIRED", "retryable": false, "description": "IDs array is required"}]',
true,
'[{"name": "output", "label": "Output", "is_default": true, "description": "Delete result"}]',
'[{"name": "input", "label": "Input", "schema": {"type": "object"}, "required": true, "description": "IDs to delete"}]',
'[]', false,
'const collection = config.collection || input.collection;
if (!collection) throw new Error(''[VEC_001] Collection name is required'');
const ids = input.ids || (input.id ? [input.id] : null);
if (!ids || ids.length === 0) throw new Error(''[VEC_004] IDs array is required'');
const result = await ctx.vector.delete(collection, ids);
return {collection, deleted_count: result.deleted_count, requested_ids: ids};',
'{"icon": "trash-2", "color": "#EF4444"}', true, 1);

-- doc-loader: Load documents from various sources
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES
('rag00005-0000-0000-0000-000000000001', NULL, 'doc-loader', 'Document Loader', 'Load documents from URL, text, or JSON', 'data', 'file-text',
'{"type": "object", "properties": {"source_type": {"type": "string", "enum": ["url", "text", "json"], "default": "url", "title": "Source Type"}, "url": {"type": "string", "title": "URL"}, "content": {"type": "string", "title": "Text Content"}, "strip_html": {"type": "boolean", "default": true, "title": "Strip HTML Tags"}}}',
'{"type": "object", "properties": {"url": {"type": "string"}, "content": {"type": "string"}, "text": {"type": "string"}}}',
'{"type": "object", "properties": {"documents": {"type": "array"}}}',
'[{"code": "DOC_001", "name": "FETCH_ERROR", "retryable": true, "description": "Failed to fetch URL"}, {"code": "DOC_002", "name": "EMPTY_CONTENT", "retryable": false, "description": "No content provided"}]',
true,
'[{"name": "output", "label": "Output", "is_default": true, "description": "Loaded documents"}]',
'[{"name": "input", "label": "Input", "schema": {"type": "object"}, "required": false, "description": "Optional source data"}]',
'[]', false,
E'const sourceType = config.source_type || \'url\';
let content, metadata;
if (sourceType === \'url\') {
  const url = config.url || input.url;
  if (!url) throw new Error(\'[DOC_002] URL is required for url source type\');
  const response = await ctx.http.get(url);
  content = typeof response.data === \'string\' ? response.data : JSON.stringify(response.data);
  metadata = {source: url, source_type: \'url\', content_type: response.headers[\'Content-Type\'], fetched_at: new Date().toISOString()};
} else if (sourceType === \'text\') {
  content = config.content || input.content || input.text;
  if (!content) throw new Error(\'[DOC_002] No content provided\');
  metadata = {source_type: \'text\'};
} else if (sourceType === \'json\') {
  const data = input.data || input;
  content = config.content_path ? getPath(data, config.content_path) : JSON.stringify(data);
  metadata = {source_type: \'json\'};
}
if (config.strip_html && content && content.includes(\'<\')) {
  content = content.replace(/<script[^>]*>[\\s\\S]*?<\\/script>/gi, \'\').replace(/<style[^>]*>[\\s\\S]*?<\\/style>/gi, \'\').replace(/<[^>]+>/g, \' \').replace(/\\s+/g, \' \').trim();
}
return {documents: [{content, metadata, char_count: content.length}]};',
'{"icon": "file-text", "color": "#F59E0B"}', true, 1);

-- text-splitter: Split documents into chunks
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES
('rag00006-0000-0000-0000-000000000001', NULL, 'text-splitter', 'Text Splitter', 'Split documents into smaller chunks', 'data', 'scissors',
'{"type": "object", "properties": {"chunk_size": {"type": "integer", "default": 1000, "minimum": 100, "maximum": 8000, "title": "Chunk Size (chars)"}, "chunk_overlap": {"type": "integer", "default": 200, "minimum": 0, "title": "Overlap (chars)"}, "separator": {"type": "string", "default": "\\n\\n", "title": "Separator"}}}',
'{"type": "object", "properties": {"documents": {"type": "array"}, "content": {"type": "string"}, "text": {"type": "string"}}}',
'{"type": "object", "properties": {"documents": {"type": "array"}, "chunk_count": {"type": "integer"}}}',
'[{"code": "SPLIT_001", "name": "NO_CONTENT", "retryable": false, "description": "No content to split"}]',
true,
'[{"name": "output", "label": "Output", "is_default": true, "description": "Split documents"}]',
'[{"name": "documents", "label": "Documents", "schema": {"type": "array"}, "required": true, "description": "Documents to split"}]',
'[]', false,
E'const documents = input.documents || [{content: input.content || input.text}];
if (!documents || documents.length === 0) throw new Error(\'[SPLIT_001] No content to split\');
const chunkSize = config.chunk_size || 1000;
const chunkOverlap = config.chunk_overlap || 200;
const separator = config.separator || \'\\n\\n\';
function splitText(text, size, overlap, sep) {
  const chunks = [];
  const segments = text.split(sep);
  let current = \'\';
  for (const segment of segments) {
    const combined = current ? current + sep + segment : segment;
    if (combined.length > size && current) {
      chunks.push(current.trim());
      const words = current.split(/\\s+/);
      const overlapWords = Math.ceil(overlap / 6);
      current = words.slice(-overlapWords).join(\' \') + sep + segment;
    } else {
      current = combined;
    }
  }
  if (current.trim()) chunks.push(current.trim());
  return chunks;
}
const result = [];
for (const doc of documents) {
  const chunks = splitText(doc.content || \'\', chunkSize, chunkOverlap, separator);
  for (let i = 0; i < chunks.length; i++) {
    result.push({content: chunks[i], metadata: {...(doc.metadata || {}), chunk_index: i, chunk_total: chunks.length}, char_count: chunks[i].length});
  }
}
return {documents: result, chunk_count: result.length, original_count: documents.length};',
'{"icon": "scissors", "color": "#06B6D4"}', true, 1);

-- rag-query: Combined RAG query (search + augment + LLM)
INSERT INTO block_definitions (id, tenant_id, slug, name, description, category, icon, config_schema, input_schema, output_schema, error_codes, enabled, output_ports, input_ports, required_credentials, is_public, code, ui_config, is_system, version) VALUES
('rag00007-0000-0000-0000-000000000001', NULL, 'rag-query', 'RAG Query', 'Search documents and generate answer with LLM', 'ai', 'message-square',
'{"type": "object", "required": ["collection"], "properties": {"collection": {"type": "string", "title": "Collection Name"}, "top_k": {"type": "integer", "default": 5, "title": "Search Results"}, "embedding_provider": {"type": "string", "default": "openai"}, "embedding_model": {"type": "string", "default": "text-embedding-3-small"}, "llm_provider": {"type": "string", "enum": ["openai", "anthropic"], "default": "openai", "title": "LLM Provider"}, "llm_model": {"type": "string", "default": "gpt-4", "title": "LLM Model"}, "system_prompt": {"type": "string", "title": "System Prompt"}, "temperature": {"type": "number", "default": 0.3, "minimum": 0, "maximum": 2}, "max_tokens": {"type": "integer", "default": 2000}}}',
'{"type": "object", "required": ["query"], "properties": {"query": {"type": "string"}, "question": {"type": "string"}}}',
'{"type": "object", "properties": {"answer": {"type": "string"}, "sources": {"type": "array"}}}',
'[{"code": "RAG_001", "name": "QUERY_REQUIRED", "retryable": false, "description": "Query is required"}, {"code": "RAG_002", "name": "COLLECTION_REQUIRED", "retryable": false, "description": "Collection is required"}]',
true,
'[{"name": "output", "label": "Output", "is_default": true, "description": "Answer with sources"}]',
'[{"name": "query", "label": "Query", "schema": {"type": "string"}, "required": true, "description": "Question to answer"}]',
'[]', false,
E'const query = input.query || input.question;
if (!query) throw new Error(\'[RAG_001] Query is required\');
const collection = config.collection || input.collection;
if (!collection) throw new Error(\'[RAG_002] Collection is required\');
const embeddingProvider = config.embedding_provider || \'openai\';
const embeddingModel = config.embedding_model || \'text-embedding-3-small\';
const llmProvider = config.llm_provider || \'openai\';
const llmModel = config.llm_model || \'gpt-4\';
const topK = config.top_k || 5;
const embedResult = await ctx.embedding.embed(embeddingProvider, embeddingModel, [query]);
const queryVector = embedResult.vectors[0];
const searchResult = await ctx.vector.query(collection, queryVector, {top_k: topK, include_content: true});
const context = searchResult.matches.map((m, i) => \'[\' + (i + 1) + \'] \' + m.content).join(\'\\n\\n---\\n\\n\');
const systemPrompt = config.system_prompt || \'You are a helpful assistant. Answer based on the provided context. Cite sources using [N]. If context lacks relevant info, say so.\';
const userPrompt = \'## Context\\n\\n\' + context + \'\\n\\n## Question\\n\\n\' + query + \'\\n\\n## Answer\';
const llmResponse = await ctx.llm.chat(llmProvider, llmModel, {messages: [{role: \'system\', content: systemPrompt}, {role: \'user\', content: userPrompt}], temperature: config.temperature || 0.3, max_tokens: config.max_tokens || 2000});
return {answer: llmResponse.content, sources: searchResult.matches.map(m => ({id: m.id, score: m.score, content: (m.content || \'\').substring(0, 200) + \'...\', metadata: m.metadata})), usage: {embedding: embedResult.usage, llm: llmResponse.usage}};',
'{"icon": "message-square", "color": "#8B5CF6"}', true, 1);

-- ============================================================================
-- RAG Sample Workflows
-- ============================================================================

-- RAG Sample Workflow 1: Document Indexing Pipeline
-- Input: { "documents": [{ "content": "...", "metadata": {...} }], "collection": "my-docs" }
INSERT INTO workflows (id, tenant_id, name, description, status, version, input_schema, output_schema, draft, created_by, published_at, created_at, updated_at, deleted_at, is_system, system_slug) VALUES
('a0000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000001',
 'RAG: Document Indexing Pipeline',
 'Index documents into vector database for RAG queries. Split documents into chunks, generate embeddings, and store in vector DB.',
 'published', 1,
 '{"type": "object", "required": ["documents", "collection"], "properties": {"documents": {"type": "array", "items": {"type": "object", "properties": {"content": {"type": "string"}, "metadata": {"type": "object"}}}}, "collection": {"type": "string"}}}',
 '{"type": "object", "properties": {"indexed_count": {"type": "integer"}, "chunk_count": {"type": "integer"}}}',
 NULL, NULL, NOW(), NOW(), NOW(), NULL, false, NULL);

-- Steps for Document Indexing Pipeline
INSERT INTO steps (id, workflow_id, name, type, config, position_x, position_y, created_at, updated_at, block_group_id, group_role, credential_bindings, block_definition_id) VALUES
('d0000001-0000-0000-0000-000000000101', 'a0000000-0000-0000-0000-000000000101', 'Start', 'start', '{}', 400, 50, NOW(), NOW(), NULL, NULL, '{}', NULL),
('d0000002-0000-0000-0000-000000000101', 'a0000000-0000-0000-0000-000000000101', 'Split Documents', 'text-splitter', '{"chunk_size": 1000, "chunk_overlap": 200}', 400, 200, NOW(), NOW(), NULL, NULL, '{}', 'rag00006-0000-0000-0000-000000000001'),
('d0000003-0000-0000-0000-000000000101', 'a0000000-0000-0000-0000-000000000101', 'Generate Embeddings', 'embedding', '{"provider": "openai", "model": "text-embedding-3-small"}', 400, 350, NOW(), NOW(), NULL, NULL, '{}', 'rag00001-0000-0000-0000-000000000001'),
('d0000004-0000-0000-0000-000000000101', 'a0000000-0000-0000-0000-000000000101', 'Store in Vector DB', 'vector-upsert', '{"collection": "{{$.collection}}"}', 400, 500, NOW(), NOW(), NULL, NULL, '{}', 'rag00002-0000-0000-0000-000000000001'),
('d0000005-0000-0000-0000-000000000101', 'a0000000-0000-0000-0000-000000000101', 'Return Result', 'function', '{"code": "return { indexed_count: input.upserted_count, chunk_count: input.upserted_count, collection: input.collection, ids: input.ids };", "language": "javascript"}', 400, 650, NOW(), NOW(), NULL, NULL, '{}', NULL);

-- Edges for Document Indexing Pipeline
INSERT INTO edges (id, workflow_id, source_step_id, target_step_id, condition, created_at, source_port, target_port) VALUES
('e0000001-0000-0000-0000-000000000101', 'a0000000-0000-0000-0000-000000000101', 'd0000001-0000-0000-0000-000000000101', 'd0000002-0000-0000-0000-000000000101', NULL, NOW(), 'default', ''),
('e0000002-0000-0000-0000-000000000101', 'a0000000-0000-0000-0000-000000000101', 'd0000002-0000-0000-0000-000000000101', 'd0000003-0000-0000-0000-000000000101', NULL, NOW(), 'default', ''),
('e0000003-0000-0000-0000-000000000101', 'a0000000-0000-0000-0000-000000000101', 'd0000003-0000-0000-0000-000000000101', 'd0000004-0000-0000-0000-000000000101', NULL, NOW(), 'default', ''),
('e0000004-0000-0000-0000-000000000101', 'a0000000-0000-0000-0000-000000000101', 'd0000004-0000-0000-0000-000000000101', 'd0000005-0000-0000-0000-000000000101', NULL, NOW(), 'default', '');

-- RAG Sample Workflow 2: Question Answering
-- Input: { "query": "What is...?", "collection": "my-docs" }
INSERT INTO workflows (id, tenant_id, name, description, status, version, input_schema, output_schema, draft, created_by, published_at, created_at, updated_at, deleted_at, is_system, system_slug) VALUES
('a0000000-0000-0000-0000-000000000102', '00000000-0000-0000-0000-000000000001',
 'RAG: Question Answering',
 'Answer questions using RAG. Searches vector database for relevant documents and generates answer using LLM.',
 'published', 1,
 '{"type": "object", "required": ["query", "collection"], "properties": {"query": {"type": "string"}, "collection": {"type": "string"}}}',
 '{"type": "object", "properties": {"answer": {"type": "string"}, "sources": {"type": "array"}}}',
 NULL, NULL, NOW(), NOW(), NOW(), NULL, false, NULL);

-- Steps for Question Answering
INSERT INTO steps (id, workflow_id, name, type, config, position_x, position_y, created_at, updated_at, block_group_id, group_role, credential_bindings, block_definition_id) VALUES
('d0000001-0000-0000-0000-000000000102', 'a0000000-0000-0000-0000-000000000102', 'Start', 'start', '{}', 400, 50, NOW(), NOW(), NULL, NULL, '{}', NULL),
('d0000002-0000-0000-0000-000000000102', 'a0000000-0000-0000-0000-000000000102', 'RAG Query', 'rag-query', '{"collection": "{{$.collection}}", "top_k": 5, "llm_provider": "openai", "llm_model": "gpt-4o-mini", "system_prompt": "You are a helpful assistant. Answer questions based on the provided context. If the context does not contain enough information, say so clearly."}', 400, 200, NOW(), NOW(), NULL, NULL, '{}', 'rag00007-0000-0000-0000-000000000001');

-- Edges for Question Answering
INSERT INTO edges (id, workflow_id, source_step_id, target_step_id, condition, created_at, source_port, target_port) VALUES
('e0000001-0000-0000-0000-000000000102', 'a0000000-0000-0000-0000-000000000102', 'd0000001-0000-0000-0000-000000000102', 'd0000002-0000-0000-0000-000000000102', NULL, NOW(), 'default', '');

-- RAG Sample Workflow 3: Knowledge Base Chat
-- Input: { "query": "...", "collection": "kb", "chat_history": [] }
INSERT INTO workflows (id, tenant_id, name, description, status, version, input_schema, output_schema, draft, created_by, published_at, created_at, updated_at, deleted_at, is_system, system_slug) VALUES
('a0000000-0000-0000-0000-000000000103', '00000000-0000-0000-0000-000000000001',
 'RAG: Knowledge Base Chat',
 'Interactive chat with knowledge base. Maintains conversation context and retrieves relevant documents for each query.',
 'published', 1,
 '{"type": "object", "required": ["query", "collection"], "properties": {"query": {"type": "string"}, "collection": {"type": "string"}, "chat_history": {"type": "array", "items": {"type": "object", "properties": {"role": {"type": "string"}, "content": {"type": "string"}}}}}}',
 '{"type": "object", "properties": {"answer": {"type": "string"}, "sources": {"type": "array"}, "chat_history": {"type": "array"}}}',
 NULL, NULL, NOW(), NOW(), NOW(), NULL, false, NULL);

-- Steps for Knowledge Base Chat
INSERT INTO steps (id, workflow_id, name, type, config, position_x, position_y, created_at, updated_at, block_group_id, group_role, credential_bindings, block_definition_id) VALUES
('d0000001-0000-0000-0000-000000000103', 'a0000000-0000-0000-0000-000000000103', 'Start', 'start', '{}', 400, 50, NOW(), NOW(), NULL, NULL, '{}', NULL),
('d0000002-0000-0000-0000-000000000103', 'a0000000-0000-0000-0000-000000000103', 'Search Documents', 'vector-search', '{"collection": "{{$.collection}}", "top_k": 5, "include_content": true}', 400, 200, NOW(), NOW(), NULL, NULL, '{}', 'rag00003-0000-0000-0000-000000000001'),
('d0000003-0000-0000-0000-000000000103', 'a0000000-0000-0000-0000-000000000103', 'Build Context', 'function', '{"code": "const context = (input.matches || []).map((m, i) => `[${i+1}] ${m.content}`).join(''\\n\\n---\\n\\n''); const history = (input.chat_history || []).map(h => `${h.role}: ${h.content}`).join(''\\n''); return { context, history, query: input.query, matches: input.matches };", "language": "javascript"}', 400, 350, NOW(), NOW(), NULL, NULL, '{}', NULL),
('d0000004-0000-0000-0000-000000000103', 'a0000000-0000-0000-0000-000000000103', 'Generate Answer', 'llm', '{"provider": "openai", "model": "gpt-4o-mini", "system_prompt": "You are a helpful knowledge base assistant. Answer based on the context provided. Cite sources using [N] notation.", "user_prompt": "## Previous Conversation\\n{{$.history}}\\n\\n## Retrieved Context\\n{{$.context}}\\n\\n## User Question\\n{{$.query}}\\n\\n## Answer", "temperature": 0.3, "max_tokens": 2000}', 400, 500, NOW(), NOW(), NULL, NULL, '{}', NULL),
('d0000005-0000-0000-0000-000000000103', 'a0000000-0000-0000-0000-000000000103', 'Format Response', 'function', '{"code": "const newHistory = [...(input.chat_history || []), {role: ''user'', content: input.query}, {role: ''assistant'', content: input.content}]; return { answer: input.content, sources: (input.matches || []).map(m => ({id: m.id, score: m.score, excerpt: (m.content || '''').substring(0, 150) + ''...''})), chat_history: newHistory };", "language": "javascript"}', 400, 650, NOW(), NOW(), NULL, NULL, '{}', NULL);

-- Edges for Knowledge Base Chat
INSERT INTO edges (id, workflow_id, source_step_id, target_step_id, condition, created_at, source_port, target_port) VALUES
('e0000001-0000-0000-0000-000000000103', 'a0000000-0000-0000-0000-000000000103', 'd0000001-0000-0000-0000-000000000103', 'd0000002-0000-0000-0000-000000000103', NULL, NOW(), 'default', ''),
('e0000002-0000-0000-0000-000000000103', 'a0000000-0000-0000-0000-000000000103', 'd0000002-0000-0000-0000-000000000103', 'd0000003-0000-0000-0000-000000000103', NULL, NOW(), 'default', ''),
('e0000003-0000-0000-0000-000000000103', 'a0000000-0000-0000-0000-000000000103', 'd0000003-0000-0000-0000-000000000103', 'd0000004-0000-0000-0000-000000000103', NULL, NOW(), 'default', ''),
('e0000004-0000-0000-0000-000000000103', 'a0000000-0000-0000-0000-000000000103', 'd0000004-0000-0000-0000-000000000103', 'd0000005-0000-0000-0000-000000000103', NULL, NOW(), 'default', '');

\unrestrict U4nbNOpGbzQDaE5Nbt6A6xfSA28CJLmedtMlJHtwSRT23KSx5kSKoTOYEzhykri
