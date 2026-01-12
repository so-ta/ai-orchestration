-- Migration: 014_copilot_system_workflows.sql
-- Description: Create system workflows for Copilot functionality
-- These workflows are executed internally by the Copilot feature

-- ============================================
-- 1. Copilot Generate Workflow
-- ============================================

-- System workflow: copilot-generate
-- Generates workflow structures from natural language descriptions
INSERT INTO workflows (
    id, tenant_id, name, description, status, version,
    is_system, system_slug, created_at, updated_at, published_at
) VALUES (
    'a0000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000001',  -- Default tenant for system workflows
    'Copilot: Generate Workflow',
    'Generates workflow structures from natural language descriptions using AI',
    'published',
    1,
    TRUE,
    'copilot-generate',
    NOW(),
    NOW(),
    NOW()
) ON CONFLICT DO NOTHING;

-- Steps for copilot-generate workflow
INSERT INTO steps (id, workflow_id, name, type, config, position_x, position_y, created_at, updated_at) VALUES
-- Start step
('b0000001-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001',
 'Start', 'start', '{}', 400, 50, NOW(), NOW()),

-- Get available blocks
('b0000002-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001',
 'Get Available Blocks', 'function',
 '{"code": "const blocks = context.blocks.list(); return { blocks: blocks.map(b => ({ slug: b.slug, name: b.name, description: b.description, category: b.category })) };", "language": "javascript"}',
 400, 200, NOW(), NOW()),

-- Build prompt for LLM
('b0000003-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001',
 'Build Prompt', 'function',
 '{"code": "const blocksInfo = input.blocks.map(b => `- ${b.slug}: ${b.name} (${b.category}) - ${b.description || \"\"}`).join(\"\\n\");\nconst prompt = `You are an AI workflow generator. Generate a workflow based on the user description.\\n\\n## Available Blocks\\n${blocksInfo}\\n\\n## Available Step Types\\n- start: Entry point (required)\\n- llm: AI/LLM call\\n- tool: External adapter\\n- condition: Binary branch (true/false)\\n- switch: Multi-way branch\\n- map: Parallel array processing\\n- loop: Iteration\\n- wait: Delay\\n- function: Custom JavaScript\\n- log: Debug logging\\n\\n## User Request\\n${input.prompt}\\n\\n## Output Format (JSON)\\n{\\n  \"response\": \"Explanation\",\\n  \"steps\": [{\"temp_id\": \"step_1\", \"name\": \"Step Name\", \"type\": \"start\", \"description\": \"\", \"config\": {}, \"position_x\": 400, \"position_y\": 50}],\\n  \"edges\": [{\"source_temp_id\": \"step_1\", \"target_temp_id\": \"step_2\", \"source_port\": \"default\"}],\\n  \"start_step_id\": \"step_1\"\\n}\\n\\nGenerate a valid workflow JSON. Always include a start step.`;\nreturn { prompt: prompt };", "language": "javascript"}',
 400, 350, NOW(), NOW()),

-- Call LLM to generate workflow
('b0000004-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001',
 'Generate with LLM', 'llm',
 '{"provider": "openai", "model": "gpt-4o-mini", "system_prompt": "You are an AI workflow generator. Always respond with valid JSON.", "user_prompt": "{{$.prompt}}", "temperature": 0.3, "max_tokens": 4000}',
 400, 500, NOW(), NOW()),

-- Parse and validate the response
('b0000005-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001',
 'Parse & Validate', 'function',
 '{"code": "try { let content = input.content || \"\"; if (content.startsWith(\"```json\")) content = content.slice(7); if (content.startsWith(\"```\")) content = content.slice(3); if (content.endsWith(\"```\")) content = content.slice(0, -3); content = content.trim(); const result = JSON.parse(content); if (!result.steps || !Array.isArray(result.steps)) { return { error: \"Invalid workflow: missing steps array\" }; } const validTypes = [\"start\", \"llm\", \"tool\", \"condition\", \"switch\", \"map\", \"join\", \"subflow\", \"loop\", \"wait\", \"function\", \"router\", \"human_in_loop\", \"filter\", \"split\", \"aggregate\", \"error\", \"note\", \"log\"]; result.steps = result.steps.filter(s => validTypes.includes(s.type)); return result; } catch (e) { return { error: \"Failed to parse LLM response: \" + e.message }; }", "language": "javascript"}',
 400, 650, NOW(), NOW())
ON CONFLICT DO NOTHING;

-- Edges for copilot-generate workflow
INSERT INTO edges (id, workflow_id, source_step_id, target_step_id, source_port, created_at) VALUES
('c0000001-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001',
 'b0000001-0000-0000-0000-000000000001', 'b0000002-0000-0000-0000-000000000001', 'default', NOW()),
('c0000002-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001',
 'b0000002-0000-0000-0000-000000000001', 'b0000003-0000-0000-0000-000000000001', 'default', NOW()),
('c0000003-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001',
 'b0000003-0000-0000-0000-000000000001', 'b0000004-0000-0000-0000-000000000001', 'default', NOW()),
('c0000004-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001',
 'b0000004-0000-0000-0000-000000000001', 'b0000005-0000-0000-0000-000000000001', 'default', NOW())
ON CONFLICT DO NOTHING;

-- ============================================
-- 2. Copilot Suggest Workflow
-- ============================================

INSERT INTO workflows (
    id, tenant_id, name, description, status, version,
    is_system, system_slug, created_at, updated_at, published_at
) VALUES (
    'a0000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000001',
    'Copilot: Suggest Steps',
    'Suggests next steps for a workflow based on current structure',
    'published',
    1,
    TRUE,
    'copilot-suggest',
    NOW(),
    NOW(),
    NOW()
) ON CONFLICT DO NOTHING;

-- Steps for copilot-suggest workflow
INSERT INTO steps (id, workflow_id, name, type, config, position_x, position_y, created_at, updated_at) VALUES
('b0000001-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000002',
 'Start', 'start', '{}', 400, 50, NOW(), NOW()),

('b0000002-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000002',
 'Get Workflow Context', 'function',
 '{"code": "const workflow = context.workflows.get(input.workflow_id); const blocks = context.blocks.list(); return { workflow: workflow, blocks: blocks };", "language": "javascript"}',
 400, 200, NOW(), NOW()),

('b0000003-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000002',
 'Build Suggest Prompt', 'function',
 '{"code": "const wf = input.workflow; const blocksInfo = input.blocks.slice(0, 20).map(b => `- ${b.slug}: ${b.name}`).join(\"\\n\"); const stepsInfo = (wf.steps || []).map(s => `- ${s.name} (${s.type})`).join(\"\\n\"); const prompt = `Suggest 2-3 next steps for this workflow.\\n\\n## Current Steps\\n${stepsInfo || \"(empty)\"}\\n\\n## Available Blocks\\n${blocksInfo}\\n\\n## Context\\n${input.context || \"\"}\\n\\nReturn JSON array: [{\"type\": \"...\", \"name\": \"...\", \"description\": \"...\", \"config\": {}, \"reason\": \"...\"}]`; return { prompt: prompt };", "language": "javascript"}',
 400, 350, NOW(), NOW()),

('b0000004-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000002',
 'Suggest with LLM', 'llm',
 '{"provider": "openai", "model": "gpt-4o-mini", "system_prompt": "You are an AI workflow assistant. Return valid JSON array.", "user_prompt": "{{$.prompt}}", "temperature": 0.5, "max_tokens": 2000}',
 400, 500, NOW(), NOW()),

('b0000005-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000002',
 'Parse Suggestions', 'function',
 '{"code": "try { let content = input.content || \"\"; if (content.startsWith(\"```\")) { content = content.replace(/```json?\\n?/g, \"\").replace(/```/g, \"\").trim(); } const suggestions = JSON.parse(content); return { suggestions: Array.isArray(suggestions) ? suggestions : [] }; } catch (e) { return { suggestions: [] }; }", "language": "javascript"}',
 400, 650, NOW(), NOW())
ON CONFLICT DO NOTHING;

-- Edges for copilot-suggest workflow
INSERT INTO edges (id, workflow_id, source_step_id, target_step_id, source_port, created_at) VALUES
('c0000001-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000002',
 'b0000001-0000-0000-0000-000000000002', 'b0000002-0000-0000-0000-000000000002', 'default', NOW()),
('c0000002-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000002',
 'b0000002-0000-0000-0000-000000000002', 'b0000003-0000-0000-0000-000000000002', 'default', NOW()),
('c0000003-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000002',
 'b0000003-0000-0000-0000-000000000002', 'b0000004-0000-0000-0000-000000000002', 'default', NOW()),
('c0000004-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000002',
 'b0000004-0000-0000-0000-000000000002', 'b0000005-0000-0000-0000-000000000002', 'default', NOW())
ON CONFLICT DO NOTHING;

-- ============================================
-- 3. Copilot Diagnose Workflow
-- ============================================

INSERT INTO workflows (
    id, tenant_id, name, description, status, version,
    is_system, system_slug, created_at, updated_at, published_at
) VALUES (
    'a0000000-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000001',
    'Copilot: Diagnose Error',
    'Diagnoses workflow execution errors and suggests fixes',
    'published',
    1,
    TRUE,
    'copilot-diagnose',
    NOW(),
    NOW(),
    NOW()
) ON CONFLICT DO NOTHING;

-- Steps for copilot-diagnose workflow
INSERT INTO steps (id, workflow_id, name, type, config, position_x, position_y, created_at, updated_at) VALUES
('b0000001-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000003',
 'Start', 'start', '{}', 400, 50, NOW(), NOW()),

('b0000002-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000003',
 'Get Run Details', 'function',
 '{"code": "const run = context.runs.get(input.run_id); const stepRuns = context.runs.getStepRuns(input.run_id); const failedSteps = stepRuns.filter(sr => sr.status === \"failed\"); return { run: run, stepRuns: stepRuns, failedSteps: failedSteps };", "language": "javascript"}',
 400, 200, NOW(), NOW()),

('b0000003-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000003',
 'Build Diagnose Prompt', 'function',
 '{"code": "const failedInfo = input.failedSteps.map(sr => `Step: ${sr.step_name}\\nError: ${sr.error || \"Unknown\"}\\nInput: ${JSON.stringify(sr.input || {})}`).join(\"\\n\\n\"); const prompt = `Diagnose this workflow error.\\n\\n## Run Status: ${input.run.status}\\n\\n## Failed Steps\\n${failedInfo || \"No failures found\"}\\n\\nReturn JSON: {\"diagnosis\": {\"root_cause\": \"...\", \"category\": \"config_error|input_error|api_error|logic_error|timeout|unknown\", \"severity\": \"high|medium|low\"}, \"fixes\": [{\"description\": \"...\", \"steps\": [\"...\"]}], \"preventions\": [\"...\"]}`; return { prompt: prompt };", "language": "javascript"}',
 400, 350, NOW(), NOW()),

('b0000004-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000003',
 'Diagnose with LLM', 'llm',
 '{"provider": "openai", "model": "gpt-4o-mini", "system_prompt": "You are an AI debugging assistant. Return valid JSON.", "user_prompt": "{{$.prompt}}", "temperature": 0.3, "max_tokens": 2000}',
 400, 500, NOW(), NOW()),

('b0000005-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000003',
 'Parse Diagnosis', 'function',
 '{"code": "try { let content = input.content || \"\"; if (content.startsWith(\"```\")) { content = content.replace(/```json?\\n?/g, \"\").replace(/```/g, \"\").trim(); } return JSON.parse(content); } catch (e) { return { diagnosis: { root_cause: \"Parse error\", category: \"unknown\", severity: \"low\" }, fixes: [], preventions: [] }; }", "language": "javascript"}',
 400, 650, NOW(), NOW())
ON CONFLICT DO NOTHING;

-- Edges for copilot-diagnose workflow
INSERT INTO edges (id, workflow_id, source_step_id, target_step_id, source_port, created_at) VALUES
('c0000001-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000003',
 'b0000001-0000-0000-0000-000000000003', 'b0000002-0000-0000-0000-000000000003', 'default', NOW()),
('c0000002-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000003',
 'b0000002-0000-0000-0000-000000000003', 'b0000003-0000-0000-0000-000000000003', 'default', NOW()),
('c0000003-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000003',
 'b0000003-0000-0000-0000-000000000003', 'b0000004-0000-0000-0000-000000000003', 'default', NOW()),
('c0000004-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000003',
 'b0000004-0000-0000-0000-000000000003', 'b0000005-0000-0000-0000-000000000003', 'default', NOW())
ON CONFLICT DO NOTHING;

-- ============================================
-- 4. Copilot Optimize Workflow
-- ============================================

INSERT INTO workflows (
    id, tenant_id, name, description, status, version,
    is_system, system_slug, created_at, updated_at, published_at
) VALUES (
    'a0000000-0000-0000-0000-000000000004',
    '00000000-0000-0000-0000-000000000001',
    'Copilot: Optimize Workflow',
    'Suggests optimizations for workflow performance, cost, and reliability',
    'published',
    1,
    TRUE,
    'copilot-optimize',
    NOW(),
    NOW(),
    NOW()
) ON CONFLICT DO NOTHING;

-- Steps for copilot-optimize workflow
INSERT INTO steps (id, workflow_id, name, type, config, position_x, position_y, created_at, updated_at) VALUES
('b0000001-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000004',
 'Start', 'start', '{}', 400, 50, NOW(), NOW()),

('b0000002-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000004',
 'Get Workflow Details', 'function',
 '{"code": "const workflow = context.workflows.get(input.workflow_id); return { workflow: workflow };", "language": "javascript"}',
 400, 200, NOW(), NOW()),

('b0000003-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000004',
 'Build Optimize Prompt', 'function',
 '{"code": "const wf = input.workflow; const stepsInfo = (wf.steps || []).map(s => `- ${s.name} (${s.type}): ${JSON.stringify(s.config || {})}`).join(\"\\n\"); const prompt = `Suggest optimizations for this workflow.\\n\\n## Workflow: ${wf.name}\\n## Steps (${(wf.steps || []).length})\\n${stepsInfo}\\n\\nReturn JSON: {\"optimizations\": [{\"category\": \"performance|cost|reliability|maintainability\", \"title\": \"...\", \"description\": \"...\", \"impact\": \"high|medium|low\", \"effort\": \"high|medium|low\"}], \"summary\": \"...\"}`; return { prompt: prompt };", "language": "javascript"}',
 400, 350, NOW(), NOW()),

('b0000004-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000004',
 'Optimize with LLM', 'llm',
 '{"provider": "openai", "model": "gpt-4o-mini", "system_prompt": "You are an AI optimization assistant. Return valid JSON.", "user_prompt": "{{$.prompt}}", "temperature": 0.5, "max_tokens": 2000}',
 400, 500, NOW(), NOW()),

('b0000005-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000004',
 'Parse Optimizations', 'function',
 '{"code": "try { let content = input.content || \"\"; if (content.startsWith(\"```\")) { content = content.replace(/```json?\\n?/g, \"\").replace(/```/g, \"\").trim(); } return JSON.parse(content); } catch (e) { return { optimizations: [], summary: \"Parse error\" }; }", "language": "javascript"}',
 400, 650, NOW(), NOW())
ON CONFLICT DO NOTHING;

-- Edges for copilot-optimize workflow
INSERT INTO edges (id, workflow_id, source_step_id, target_step_id, source_port, created_at) VALUES
('c0000001-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000004',
 'b0000001-0000-0000-0000-000000000004', 'b0000002-0000-0000-0000-000000000004', 'default', NOW()),
('c0000002-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000004',
 'b0000002-0000-0000-0000-000000000004', 'b0000003-0000-0000-0000-000000000004', 'default', NOW()),
('c0000003-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000004',
 'b0000003-0000-0000-0000-000000000004', 'b0000004-0000-0000-0000-000000000004', 'default', NOW()),
('c0000004-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000004',
 'b0000004-0000-0000-0000-000000000004', 'b0000005-0000-0000-0000-000000000004', 'default', NOW())
ON CONFLICT DO NOTHING;

-- ============================================
-- 5. Create workflow_versions for each system workflow
-- ============================================

-- Version for copilot-generate
INSERT INTO workflow_versions (id, workflow_id, version, definition, saved_at) VALUES (
    'd0000001-0000-0000-0000-000000000001',
    'a0000000-0000-0000-0000-000000000001',
    1,
    '{"name":"Copilot: Generate Workflow","description":"Generates workflow structures from natural language descriptions using AI","steps":[],"edges":[]}',
    NOW()
) ON CONFLICT DO NOTHING;

-- Version for copilot-suggest
INSERT INTO workflow_versions (id, workflow_id, version, definition, saved_at) VALUES (
    'd0000001-0000-0000-0000-000000000002',
    'a0000000-0000-0000-0000-000000000002',
    1,
    '{"name":"Copilot: Suggest Steps","description":"Suggests next steps for a workflow based on current structure","steps":[],"edges":[]}',
    NOW()
) ON CONFLICT DO NOTHING;

-- Version for copilot-diagnose
INSERT INTO workflow_versions (id, workflow_id, version, definition, saved_at) VALUES (
    'd0000001-0000-0000-0000-000000000003',
    'a0000000-0000-0000-0000-000000000003',
    1,
    '{"name":"Copilot: Diagnose Error","description":"Diagnoses workflow execution errors and suggests fixes","steps":[],"edges":[]}',
    NOW()
) ON CONFLICT DO NOTHING;

-- Version for copilot-optimize
INSERT INTO workflow_versions (id, workflow_id, version, definition, saved_at) VALUES (
    'd0000001-0000-0000-0000-000000000004',
    'a0000000-0000-0000-0000-000000000004',
    1,
    '{"name":"Copilot: Optimize Workflow","description":"Suggests optimizations for workflow performance, cost, and reliability","steps":[],"edges":[]}',
    NOW()
) ON CONFLICT DO NOTHING;

-- Comments for documentation
COMMENT ON TABLE workflows IS 'Workflow definitions. System workflows (is_system=true) are used for internal features like Copilot.';
