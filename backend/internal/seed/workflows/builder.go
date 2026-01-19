package workflows

import "encoding/json"

func (r *Registry) registerBuilderWorkflows() {
	r.register(BuilderWorkflow())
}

// BuilderWorkflow is a unified AI Workflow Builder with 4 entry points:
// - analysis: AI thinks deeply and proposes assumptions + clarifying questions
// - proposal: Update assumptions based on user answers and finalize spec
// - construct: Builds workflow from gathered requirements (WorkflowSpec)
// - refine: Refines existing workflow based on user feedback
func BuilderWorkflow() *SystemWorkflowDefinition {
	return &SystemWorkflowDefinition{
		ID:          "a0000000-0000-0000-0000-000000000002",
		SystemSlug:  "ai-builder",
		Name:        "AI Workflow Builder",
		Description: "AI-assisted workflow building with deep thinking, proposal-based confirmation, automatic construction, and refinement capabilities",
		Version:     26,
		IsSystem:    true,
		Steps: []SystemStepDefinition{
			// ============================
			// Analysis Entry Point (Y=40)
			// AI thinks deeply and proposes
			// ============================
			{
				TempID:      "start_analysis",
				Name:        "Start: Analysis",
				Type:        "start",
				TriggerType: "internal",
				TriggerConfig: json.RawMessage(`{
					"entry_point": "analysis",
					"description": "AI analyzes input deeply and proposes assumptions + questions"
				}`),
				PositionX: 40,
				PositionY: 40,
				Config: json.RawMessage(`{
					"input_schema": {
						"type": "object",
						"required": ["session_id", "message"],
						"properties": {
							"session_id": {"type": "string", "title": "Session ID"},
							"message": {"type": "string", "title": "User Message"},
							"tenant_id": {"type": "string", "title": "Tenant ID"},
							"user_id": {"type": "string", "title": "User ID"}
						}
					}
				}`),
			},
			{
				TempID:    "analysis_get_context",
				Name:      "Get Session Context",
				Type:      "function",
				PositionX: 160,
				PositionY: 40,
				Config: json.RawMessage(`{
					"code": "const session = ctx.builderSessions.get(input.session_id); const blocks = ctx.blocks.list(); return { session: session, blocks: blocks.map(b => ({ slug: b.slug, name: b.name, description: b.description, category: b.category })), message: input.message, tenant_id: input.tenant_id, user_id: input.user_id };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"session": {"type": "object", "title": "Session Info"},
							"blocks": {"type": "array", "title": "Available Blocks"},
							"message": {"type": "string", "title": "User Message"},
							"tenant_id": {"type": "string"},
							"user_id": {"type": "string"}
						}
					}
				}`),
			},
			{
				TempID:    "analysis_build_prompt",
				Name:      "Build Analysis Prompt",
				Type:      "function",
				PositionX: 280,
				PositionY: 40,
				Config: json.RawMessage(`{
					"code": "const session = input.session; const phase = session?.hearing_phase || 'analysis'; const spec = session?.spec || {}; const messages = session?.messages || []; const historyText = messages.map(m => (m.role === 'user' ? 'User: ' : 'AI: ') + m.content).join('\\n'); const blocksInfo = input.blocks.slice(0, 30).map(b => '- ' + b.slug + ': ' + b.name + ' (' + b.category + ')').join('\\n'); const prompt = 'You are a workflow builder AI with deep thinking capabilities.\\n\\n## Your Task\\nAnalyze the user\\'s request and:\\n1. **Think deeply** about what workflow they need\\n2. **Infer and assume** reasonable defaults for unclear parts\\n3. **Propose a complete WorkflowSpec** based on your analysis\\n4. **List only truly unclear points** as clarifying questions (0-3 max)\\n\\n## Key Principles\\n- **Minimize questions**: Only ask what you absolutely cannot infer\\n- **Be proactive**: Make reasonable assumptions and state them clearly\\n- **Think like an expert**: Use domain knowledge to fill gaps\\n- **Respect user time**: They want you to do the thinking\\n\\n## Current Session\\nPhase: ' + phase + '\\nExisting Spec: ' + JSON.stringify(spec, null, 2) + '\\n\\n## Conversation History\\n' + (historyText || '(First message)') + '\\n\\n## User\\'s Latest Message\\n' + input.message + '\\n\\n## Available Blocks (Reference)\\n' + blocksInfo + '\\n\\n## Output Format (JSON Only)\\n{\\n  \"response\": \"Friendly summary of what you understood and assumed\",\\n  \"thinking\": \"Your internal reasoning process (shown to user)\",\\n  \"proposed_spec\": {\\n    \"name\": \"Workflow name\",\\n    \"description\": \"Workflow description\",\\n    \"purpose\": \"Main goal\",\\n    \"business_domain\": \"sales|development|hr|finance|marketing|support|operations|personal|other\",\\n    \"trigger\": {\\n      \"type\": \"manual|schedule|webhook|event\",\\n      \"schedule\": \"cron expression if scheduled\",\\n      \"description\": \"Trigger description\"\\n    },\\n    \"completion\": {\\n      \"description\": \"What completion looks like\",\\n      \"outputs\": [{\"name\": \"output name\", \"type\": \"document|notification|data|approval|other\"}]\\n    },\\n    \"actors\": [{\"role\": \"executor|approver|reviewer|viewer\", \"description\": \"Role description\", \"count\": \"single|multiple|optional\"}],\\n    \"steps\": [{\\n      \"id\": \"step_1\",\\n      \"name\": \"Step name\",\\n      \"description\": \"What this step does\",\\n      \"type\": \"input|transform|decision|action|notification|wait\"\\n    }],\\n    \"integrations\": [{\"service\": \"Slack|GitHub|etc\", \"operation\": \"What operation\"}],\\n    \"constraints\": {\"frequency\": \"once|daily|weekly|monthly|on-demand\"}\\n  },\\n  \"assumptions\": [\\n    {\\n      \"id\": \"a1\",\\n      \"category\": \"trigger|actor|step|integration|constraint\",\\n      \"description\": \"What you assumed\",\\n      \"default\": \"The default value you chose\",\\n      \"confidence\": \"high|medium|low\",\\n      \"confirmed\": false\\n    }\\n  ],\\n  \"clarifying_points\": [\\n    {\\n      \"id\": \"q1\",\\n      \"question\": \"Question that needs user input\",\\n      \"options\": [\"Option A\", \"Option B\"],\\n      \"required\": true\\n    }\\n  ],\\n  \"nextPhase\": \"analysis|proposal|completed\",\\n  \"progress\": 0-100\\n}\\n\\n## Important Rules\\n- Response must be friendly Japanese\\n- Only ask 0-3 clarifying questions MAX\\n- High confidence assumptions need no confirmation\\n- If you have enough info, set nextPhase to \"proposal\"\\n- Include your thinking process to show transparency'; return { prompt: prompt, session_id: session?.id || input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, current_phase: phase, current_spec: spec };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"prompt": {"type": "string"},
							"session_id": {"type": "string"},
							"tenant_id": {"type": "string"},
							"user_id": {"type": "string"},
							"current_phase": {"type": "string"},
							"current_spec": {"type": "object"}
						}
					}
				}`),
			},
			{
				TempID:    "analysis_llm",
				Name:      "Analysis LLM",
				Type:      "llm",
				PositionX: 400,
				PositionY: 40,
				Config: json.RawMessage(`{
					"model": "claude-sonnet-4-20250514",
					"provider": "anthropic",
					"max_tokens": 4000,
					"temperature": 0.3,
					"user_prompt": "{{$.prompt}}",
					"system_prompt": "You are an expert workflow designer AI. You think deeply about requirements and make intelligent assumptions. You minimize questions to respect the user's time. Always respond in valid JSON format only - no markdown code blocks.\n\nKey principles:\n1. Think like an expert consultant who can infer needs\n2. Make reasonable assumptions and clearly state them\n3. Only ask questions for truly ambiguous critical decisions\n4. Propose a complete workflow spec proactively",
					"passthrough_fields": ["session_id", "tenant_id", "user_id", "current_phase", "current_spec"]
				}`),
			},
			{
				TempID:    "analysis_parse",
				Name:      "Parse Analysis Response",
				Type:      "function",
				PositionX: 520,
				PositionY: 40,
				Config: json.RawMessage(`{
					"code": "try { let content = input.content || ''; if (content.startsWith('` + "```json" + `')) content = content.slice(7); if (content.startsWith('` + "```" + `')) content = content.slice(3); if (content.endsWith('` + "```" + `')) content = content.slice(0, -3); content = content.trim(); let jsonStart = content.indexOf('{'); let jsonEnd = content.lastIndexOf('}'); if (jsonStart >= 0 && jsonEnd > jsonStart) { content = content.slice(jsonStart, jsonEnd + 1); } content = content.split(String.fromCharCode(10)).join(' ').split(String.fromCharCode(13)).join(' '); const result = JSON.parse(content); const nextPhase = result.nextPhase || 'analysis'; const progress = result.progress || (nextPhase === 'proposal' ? 70 : nextPhase === 'completed' ? 100 : 30); return { success: true, response: result.response || '', thinking: result.thinking || '', proposed_spec: result.proposed_spec || {}, assumptions: result.assumptions || [], clarifying_points: result.clarifying_points || [], nextPhase: nextPhase, progress: progress, session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, current_spec: input.current_spec }; } catch (e) { return { success: false, error: 'Failed to parse LLM response: ' + e.message, response: input.content || '', session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, current_spec: input.current_spec }; }",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"success": {"type": "boolean"},
							"response": {"type": "string"},
							"thinking": {"type": "string"},
							"proposed_spec": {"type": "object"},
							"assumptions": {"type": "array"},
							"clarifying_points": {"type": "array"},
							"nextPhase": {"type": "string"},
							"progress": {"type": "number"},
							"error": {"type": "string"},
							"session_id": {"type": "string"},
							"tenant_id": {"type": "string"},
							"user_id": {"type": "string"},
							"current_spec": {"type": "object"}
						}
					}
				}`),
			},
			{
				TempID:    "analysis_update_session",
				Name:      "Update Builder Session",
				Type:      "function",
				PositionX: 640,
				PositionY: 40,
				Config: json.RawMessage(`{
					"code": "const sessionId = input.session_id; const mergedSpec = { ...input.current_spec, ...input.proposed_spec, assumptions: input.assumptions }; ctx.builderSessions.update(sessionId, { hearing_phase: input.nextPhase, hearing_progress: input.progress, spec: mergedSpec }); ctx.builderSessions.addMessage(sessionId, { role: 'assistant', content: input.response, phase: input.nextPhase, extracted_data: { thinking: input.thinking, proposed_spec: input.proposed_spec, assumptions: input.assumptions, clarifying_points: input.clarifying_points } }); return { session_id: sessionId, message: { content: input.response, thinking: input.thinking, proposed_spec: input.proposed_spec, assumptions: input.assumptions, clarifying_points: input.clarifying_points }, phase: input.nextPhase, progress: input.progress, complete: input.nextPhase === 'completed' };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"session_id": {"type": "string"},
							"message": {"type": "object"},
							"phase": {"type": "string"},
							"progress": {"type": "number"},
							"complete": {"type": "boolean"}
						}
					}
				}`),
			},

			// ============================
			// Proposal Entry Point (Y=100)
			// Update assumptions and finalize
			// ============================
			{
				TempID:      "start_proposal",
				Name:        "Start: Proposal",
				Type:        "start",
				TriggerType: "internal",
				TriggerConfig: json.RawMessage(`{
					"entry_point": "proposal",
					"description": "Update assumptions based on user answers and finalize spec"
				}`),
				PositionX: 40,
				PositionY: 100,
				Config: json.RawMessage(`{
					"input_schema": {
						"type": "object",
						"required": ["session_id", "message"],
						"properties": {
							"session_id": {"type": "string", "title": "Session ID"},
							"message": {"type": "string", "title": "User Answers/Feedback"},
							"tenant_id": {"type": "string", "title": "Tenant ID"},
							"user_id": {"type": "string", "title": "User ID"}
						}
					}
				}`),
			},
			{
				TempID:    "proposal_get_context",
				Name:      "Get Proposal Context",
				Type:      "function",
				PositionX: 160,
				PositionY: 100,
				Config: json.RawMessage(`{
					"code": "const session = ctx.builderSessions.get(input.session_id); const blocks = ctx.blocks.list(); return { session: session, blocks: blocks.map(b => ({ slug: b.slug, name: b.name, description: b.description, category: b.category })), message: input.message, tenant_id: input.tenant_id, user_id: input.user_id };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"session": {"type": "object"},
							"blocks": {"type": "array"},
							"message": {"type": "string"},
							"tenant_id": {"type": "string"},
							"user_id": {"type": "string"}
						}
					}
				}`),
			},
			{
				TempID:    "proposal_build_prompt",
				Name:      "Build Proposal Prompt",
				Type:      "function",
				PositionX: 280,
				PositionY: 100,
				Config: json.RawMessage(`{
					"code": "const session = input.session; const spec = session?.spec || {}; const assumptions = spec.assumptions || []; const messages = session?.messages || []; const lastMsg = messages[messages.length - 1]; const clarifyingPoints = lastMsg?.extracted_data?.clarifying_points || []; const historyText = messages.map(m => (m.role === 'user' ? 'User: ' : 'AI: ') + m.content).join('\\n'); const blocksInfo = input.blocks.slice(0, 30).map(b => '- ' + b.slug + ': ' + b.name + ' (' + b.category + ')').join('\\n'); const prompt = 'You are a workflow builder AI finalizing the workflow specification.\\n\\n## Your Task\\n1. Review the user\\'s answers/feedback to your previous questions\\n2. Update assumptions based on their input\\n3. Finalize the complete WorkflowSpec\\n4. If more clarification is needed, ask follow-up (but try to minimize)\\n\\n## Previous Assumptions\\n' + JSON.stringify(assumptions, null, 2) + '\\n\\n## Previous Clarifying Questions\\n' + JSON.stringify(clarifyingPoints, null, 2) + '\\n\\n## Current Spec\\n' + JSON.stringify(spec, null, 2) + '\\n\\n## Conversation History\\n' + historyText + '\\n\\n## User\\'s Latest Response\\n' + input.message + '\\n\\n## Available Blocks\\n' + blocksInfo + '\\n\\n## Output Format (JSON Only)\\n{\\n  \"response\": \"Summary of updates and final confirmation message\",\\n  \"final_spec\": { /* Complete WorkflowSpec with all fields filled */ },\\n  \"assumptions\": [\\n    {\\n      \"id\": \"a1\",\\n      \"category\": \"trigger|actor|step|integration|constraint\",\\n      \"description\": \"What was assumed\",\\n      \"default\": \"Final value\",\\n      \"confidence\": \"high\",\\n      \"confirmed\": true\\n    }\\n  ],\\n  \"nextPhase\": \"proposal|completed\",\\n  \"progress\": 0-100\\n}\\n\\n## Important\\n- Set nextPhase to \"completed\" when spec is finalized\\n- Mark all assumptions as confirmed: true\\n- Response should be friendly Japanese\\n- No more clarifying_points in this phase (finalization only)'; return { prompt: prompt, session_id: session?.id || input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, current_spec: spec };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"prompt": {"type": "string"},
							"session_id": {"type": "string"},
							"tenant_id": {"type": "string"},
							"user_id": {"type": "string"},
							"current_spec": {"type": "object"}
						}
					}
				}`),
			},
			{
				TempID:    "proposal_llm",
				Name:      "Proposal LLM",
				Type:      "llm",
				PositionX: 400,
				PositionY: 100,
				Config: json.RawMessage(`{
					"model": "claude-sonnet-4-20250514",
					"provider": "anthropic",
					"max_tokens": 4000,
					"temperature": 0.2,
					"user_prompt": "{{$.prompt}}",
					"system_prompt": "You are an expert workflow designer finalizing a workflow specification. Update assumptions based on user feedback and produce a complete, ready-to-build spec. Always respond in valid JSON format only - no markdown code blocks.",
					"passthrough_fields": ["session_id", "tenant_id", "user_id", "current_spec"]
				}`),
			},
			{
				TempID:    "proposal_parse",
				Name:      "Parse Proposal Response",
				Type:      "function",
				PositionX: 520,
				PositionY: 100,
				Config: json.RawMessage(`{
					"code": "try { let content = input.content || ''; if (content.startsWith('` + "```json" + `')) content = content.slice(7); if (content.startsWith('` + "```" + `')) content = content.slice(3); if (content.endsWith('` + "```" + `')) content = content.slice(0, -3); content = content.trim(); let jsonStart = content.indexOf('{'); let jsonEnd = content.lastIndexOf('}'); if (jsonStart >= 0 && jsonEnd > jsonStart) { content = content.slice(jsonStart, jsonEnd + 1); } content = content.split(String.fromCharCode(10)).join(' ').split(String.fromCharCode(13)).join(' '); const result = JSON.parse(content); const nextPhase = result.nextPhase || 'completed'; const progress = result.progress || (nextPhase === 'completed' ? 100 : 85); return { success: true, response: result.response || '', final_spec: result.final_spec || input.current_spec, assumptions: result.assumptions || [], nextPhase: nextPhase, progress: progress, session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id }; } catch (e) { return { success: false, error: 'Failed to parse LLM response: ' + e.message, response: input.content || '', session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, final_spec: input.current_spec }; }",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"success": {"type": "boolean"},
							"response": {"type": "string"},
							"final_spec": {"type": "object"},
							"assumptions": {"type": "array"},
							"nextPhase": {"type": "string"},
							"progress": {"type": "number"},
							"error": {"type": "string"},
							"session_id": {"type": "string"},
							"tenant_id": {"type": "string"},
							"user_id": {"type": "string"}
						}
					}
				}`),
			},
			{
				TempID:    "proposal_update_session",
				Name:      "Update Session with Final Spec",
				Type:      "function",
				PositionX: 640,
				PositionY: 100,
				Config: json.RawMessage(`{
					"code": "const sessionId = input.session_id; const finalSpec = { ...input.final_spec, assumptions: input.assumptions }; ctx.builderSessions.update(sessionId, { hearing_phase: input.nextPhase, hearing_progress: input.progress, spec: finalSpec }); ctx.builderSessions.addMessage(sessionId, { role: 'assistant', content: input.response, phase: input.nextPhase, extracted_data: { final_spec: input.final_spec, assumptions: input.assumptions } }); return { session_id: sessionId, message: { content: input.response, final_spec: input.final_spec, assumptions: input.assumptions }, phase: input.nextPhase, progress: input.progress, complete: input.nextPhase === 'completed' };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"session_id": {"type": "string"},
							"message": {"type": "object"},
							"phase": {"type": "string"},
							"progress": {"type": "number"},
							"complete": {"type": "boolean"}
						}
					}
				}`),
			},

			// ============================
			// Construct Entry Point (Y=160)
			// ============================
			{
				TempID:      "start_construct",
				Name:        "Start: Construct",
				Type:        "start",
				TriggerType: "internal",
				TriggerConfig: json.RawMessage(`{
					"entry_point": "construct",
					"description": "Build workflow from gathered requirements"
				}`),
				PositionX: 40,
				PositionY: 160,
				Config: json.RawMessage(`{
					"input_schema": {
						"type": "object",
						"required": ["session_id"],
						"properties": {
							"session_id": {"type": "string", "title": "Session ID"},
							"tenant_id": {"type": "string", "title": "Tenant ID"},
							"user_id": {"type": "string", "title": "User ID"}
						}
					}
				}`),
			},
			{
				TempID:    "construct_get_spec",
				Name:      "Get Workflow Spec",
				Type:      "function",
				PositionX: 160,
				PositionY: 160,
				Config: json.RawMessage(`{
					"code": "const session = ctx.builderSessions.get(input.session_id); const blocks = ctx.blocks.list(); return { session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, spec: session?.spec || {}, blocks: blocks.map(b => ({ slug: b.slug, name: b.name, description: b.description, category: b.category, input_schema: b.input_schema, output_schema: b.output_schema })) };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"session_id": {"type": "string"},
							"tenant_id": {"type": "string"},
							"user_id": {"type": "string"},
							"spec": {"type": "object"},
							"blocks": {"type": "array"}
						}
					}
				}`),
			},
			{
				TempID:    "construct_build_prompt",
				Name:      "Build Construction Prompt",
				Type:      "function",
				PositionX: 280,
				PositionY: 160,
				Config: json.RawMessage(`{
					"code": "const spec = input.spec; const blocksInfo = input.blocks.map(b => '- ' + b.slug + ': ' + b.name + ' (' + b.category + ') - ' + (b.description || '')).join('\\n'); const blockCategories = {}; input.blocks.forEach(b => { if (!blockCategories[b.category]) blockCategories[b.category] = []; blockCategories[b.category].push(b.slug); }); const categoryInfo = Object.entries(blockCategories).map(([cat, slugs]) => '- ' + cat + ': ' + slugs.join(', ')).join('\\n'); const prompt = 'You are a workflow construction AI. Build a concrete workflow from the specification.\\n\\n## Workflow Specification\\n' + JSON.stringify(spec, null, 2) + '\\n\\n## Block Mapping Guide\\nMap each step to blocks with priority:\\n1. Preset blocks (highest priority) - use existing blocks\\n2. Custom required - when preset is insufficient\\n\\n### Blocks by Category\\n' + categoryInfo + '\\n\\n### All Blocks\\n' + blocksInfo + '\\n\\n## Step Types\\n- start: Entry point (required)\\n- llm: AI/LLM call\\n- tool: External adapter (specify block_slug)\\n- condition: Conditional branch (true/false)\\n- switch: Multi-branch routing\\n- map: Parallel array processing\\n- loop: Loop\\n- wait: Wait\\n- function: Custom JavaScript\\n- human_in_loop: Human approval\\n- log: Debug log\\n\\n## Output Format (JSON Only)\\n{\\n  \"workflow_name\": \"Workflow name\",\\n  \"workflow_description\": \"Description\",\\n  \"natural_language_summary\": \"User-friendly explanation (3-5 sentences)\",\\n  \"steps\": [\\n    {\\n      \"temp_id\": \"step_1\",\\n      \"name\": \"Step name\",\\n      \"type\": \"start|llm|tool|function|condition|...\",\\n      \"description\": \"Step description (user-facing)\",\\n      \"config\": { step-specific config },\\n      \"position_x\": 40,\\n      \"position_y\": 40,\\n      \"block_slug\": \"block slug (for tool/llm types)\",\\n      \"mapping_status\": \"preset|custom\",\\n      \"mapping_confidence\": \"high|medium|low\",\\n      \"custom_required\": false,\\n      \"custom_reason\": \"reason if custom needed\",\\n      \"executor\": \"system|user|approver\"\\n    }\\n  ],\\n  \"edges\": [\\n    {\\n      \"source_temp_id\": \"step_1\",\\n      \"target_temp_id\": \"step_2\",\\n      \"source_port\": \"output\",\\n      \"target_port\": \"input\",\\n      \"label\": \"transition description (for conditions)\"\\n    }\\n  ],\\n  \"start_step_id\": \"step_1\",\\n  \"summary\": {\\n    \"total_steps\": 5,\\n    \"preset_steps\": 4,\\n    \"custom_steps\": 1,\\n    \"has_approval\": false,\\n    \"has_loop\": false,\\n    \"integrations_used\": [\"slack\", \"http\"],\\n    \"custom_blocks_needed\": [{\"name\": \"...\", \"reason\": \"...\"}]\\n  },\\n  \"editable_points\": [\"Approver change\", \"Notification method\"],\\n  \"warnings\": [\"Notes if any\"]\\n}\\n\\n## Important\\n- Must include a start step\\n- Maximize use of preset blocks (mapping_status=preset)\\n- Specify reason and confidence for custom blocks\\n- natural_language_summary should be user-friendly, avoid technical jargon\\n- editable_points lists things user can customize later'; return { prompt: prompt, session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, spec: input.spec };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"prompt": {"type": "string"},
							"session_id": {"type": "string"},
							"tenant_id": {"type": "string"},
							"user_id": {"type": "string"},
							"spec": {"type": "object"}
						}
					}
				}`),
			},
			{
				TempID:    "construct_llm",
				Name:      "Construct LLM",
				Type:      "llm",
				PositionX: 400,
				PositionY: 160,
				Config: json.RawMessage(`{
					"model": "claude-sonnet-4-20250514",
					"provider": "anthropic",
					"max_tokens": 4000,
					"temperature": 0.3,
					"user_prompt": "{{$.prompt}}",
					"system_prompt": "You are an expert workflow construction AI. Build optimal workflow structures from specifications. Maximize use of preset blocks. Provide clear, user-friendly explanations. Always respond in valid JSON format only.",
					"passthrough_fields": ["session_id", "tenant_id", "user_id", "spec"]
				}`),
			},
			{
				TempID:    "construct_parse",
				Name:      "Parse Construction Result",
				Type:      "function",
				PositionX: 520,
				PositionY: 160,
				Config: json.RawMessage(`{
					"code": "try { let content = input.content || ''; if (content.startsWith('` + "```json" + `')) content = content.slice(7); if (content.startsWith('` + "```" + `')) content = content.slice(3); if (content.endsWith('` + "```" + `')) content = content.slice(0, -3); content = content.trim(); let jsonStart = content.indexOf('{'); let jsonEnd = content.lastIndexOf('}'); if (jsonStart >= 0 && jsonEnd > jsonStart) { content = content.slice(jsonStart, jsonEnd + 1); } content = content.split(String.fromCharCode(10)).join(' ').split(String.fromCharCode(13)).join(' '); const result = JSON.parse(content); if (!result.steps || !Array.isArray(result.steps)) { return { success: false, error: 'Invalid workflow: missing steps array' }; } const validTypes = ['start', 'llm', 'tool', 'condition', 'switch', 'map', 'join', 'subflow', 'loop', 'wait', 'function', 'router', 'human_in_loop', 'filter', 'split', 'aggregate', 'error', 'note', 'log', 'webhook_trigger']; result.steps = result.steps.filter(s => validTypes.includes(s.type)); const presetSteps = result.steps.filter(s => s.mapping_status === 'preset' || !s.custom_required); const customSteps = result.steps.filter(s => s.mapping_status === 'custom' || s.custom_required); result.summary = result.summary || {}; result.summary.preset_steps = presetSteps.length; result.summary.custom_steps = customSteps.length; return { success: true, workflow: result, session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, spec: input.spec }; } catch (e) { return { success: false, error: 'Failed to parse LLM response: ' + e.message, session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id }; }",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"success": {"type": "boolean"},
							"workflow": {"type": "object"},
							"error": {"type": "string"},
							"session_id": {"type": "string"},
							"tenant_id": {"type": "string"},
							"user_id": {"type": "string"},
							"spec": {"type": "object"}
						}
					}
				}`),
			},
			{
				TempID:    "construct_create_project",
				Name:      "Create Project",
				Type:      "function",
				PositionX: 640,
				PositionY: 160,
				Config: json.RawMessage(`{
					"code": "const workflow = input.workflow; const sessionId = input.session_id; const tenantId = input.tenant_id; const userId = input.user_id; const spec = input.spec; if (!workflow || !workflow.steps) { return { success: false, error: 'No workflow to create' }; } const projectOpts = { tenant_id: tenantId, name: workflow.workflow_name || 'AI Generated Workflow', description: workflow.workflow_description || '', status: 'draft' }; if (userId) { projectOpts.created_by = userId; } const project = ctx.projects.create(projectOpts); const stepIdMap = {}; const startingX = 40; const xSpacing = 120; const baseY = 40; for (let i = 0; i < workflow.steps.length; i++) { const step = workflow.steps[i]; const createdStep = ctx.steps.create({ tenant_id: tenantId, project_id: project.id, name: step.name, type: step.type, config: step.config || {}, position_x: startingX + (i * xSpacing), position_y: baseY, block_slug: step.block_slug }); stepIdMap[step.temp_id] = createdStep.id; } for (const edge of (workflow.edges || [])) { const sourceId = stepIdMap[edge.source_temp_id]; const targetId = stepIdMap[edge.target_temp_id]; if (sourceId && targetId) { ctx.edges.create({ tenant_id: tenantId, project_id: project.id, source_step_id: sourceId, target_step_id: targetId, source_port: edge.source_port || 'output', target_port: edge.target_port || 'input' }); } } ctx.builderSessions.update(sessionId, { status: 'reviewing', project_id: project.id }); const stepMappings = workflow.steps.map(s => ({ name: s.name, type: s.type, block: s.block_slug || null, mapping_status: s.mapping_status || (s.custom_required ? 'custom' : 'preset'), confidence: s.mapping_confidence || 'high', custom_required: s.custom_required || false, custom_reason: s.custom_reason, executor: s.executor || 'system' })); const customRequirements = workflow.steps.filter(s => s.custom_required).map(s => ({ name: s.name, description: s.description, reason: s.custom_reason, inputs: s.config?.input_schema || {}, outputs: s.config?.output_schema || {} })); const nlSummary = workflow.natural_language_summary || workflow.workflow_description; ctx.builderSessions.addMessage(sessionId, { role: 'assistant', content: '## Workflow Created\\n\\n' + nlSummary + '\\n\\n### Step Structure\\n' + stepMappings.map((s, i) => (i+1) + '. ' + s.name + ' (' + (s.mapping_status === 'preset' ? 'Preset' : 'Custom needed') + ')').join('\\n') + (workflow.editable_points?.length > 0 ? '\\n\\n### Customizable Points\\n' + workflow.editable_points.map(p => '- ' + p).join('\\n') : '') + (workflow.warnings?.length > 0 ? '\\n\\n### Notes\\n' + workflow.warnings.map(w => '- ' + w).join('\\n') : '') }); return { success: true, project_id: project.id, natural_language_summary: nlSummary, step_mappings: stepMappings, summary: workflow.summary, editable_points: workflow.editable_points || [], warnings: workflow.warnings || [], custom_requirements: customRequirements, assumptions_used: workflow.assumptions_used || [] };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"success": {"type": "boolean"},
							"project_id": {"type": "string"},
							"natural_language_summary": {"type": "string"},
							"step_mappings": {"type": "array"},
							"summary": {"type": "object"},
							"editable_points": {"type": "array"},
							"warnings": {"type": "array"},
							"custom_requirements": {"type": "array"},
							"assumptions_used": {"type": "array"},
							"error": {"type": "string"}
						}
					}
				}`),
			},

			// ============================
			// Refine Entry Point (Y=280)
			// ============================
			{
				TempID:      "start_refine",
				Name:        "Start: Refine",
				Type:        "start",
				TriggerType: "internal",
				TriggerConfig: json.RawMessage(`{
					"entry_point": "refine",
					"description": "Refine existing workflow based on user feedback"
				}`),
				PositionX: 40,
				PositionY: 280,
				Config: json.RawMessage(`{
					"input_schema": {
						"type": "object",
						"required": ["session_id", "feedback"],
						"properties": {
							"session_id": {"type": "string", "title": "Session ID"},
							"project_id": {"type": "string", "title": "Project ID"},
							"feedback": {"type": "string", "title": "User Feedback"},
							"tenant_id": {"type": "string", "title": "Tenant ID"},
							"user_id": {"type": "string", "title": "User ID"}
						}
					}
				}`),
			},
			{
				TempID:    "refine_get_workflow",
				Name:      "Get Current Workflow",
				Type:      "function",
				PositionX: 160,
				PositionY: 280,
				Config: json.RawMessage(`{
					"code": "const project = ctx.projects.get(input.project_id); const steps = ctx.steps.listByProject(input.project_id); const edges = ctx.edges.listByProject(input.project_id); const blocks = ctx.blocks.list(); return { session_id: input.session_id, project_id: input.project_id, tenant_id: input.tenant_id, user_id: input.user_id, feedback: input.feedback, current_workflow: { name: project?.name, description: project?.description, steps: steps, edges: edges }, blocks: blocks.map(b => ({ slug: b.slug, name: b.name, description: b.description, category: b.category })) };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"session_id": {"type": "string"},
							"project_id": {"type": "string"},
							"tenant_id": {"type": "string"},
							"user_id": {"type": "string"},
							"feedback": {"type": "string"},
							"current_workflow": {"type": "object"},
							"blocks": {"type": "array"}
						}
					}
				}`),
			},
			{
				TempID:    "refine_build_prompt",
				Name:      "Build Refine Prompt",
				Type:      "function",
				PositionX: 280,
				PositionY: 280,
				Config: json.RawMessage(`{
					"code": "const workflow = input.current_workflow; const stepsInfo = (workflow.steps || []).map(s => '- ' + s.name + ' (type: ' + s.type + ', id: ' + s.id + ')').join('\\n'); const edgesInfo = (workflow.edges || []).map(e => '- ' + e.source_step_id + ' -> ' + e.target_step_id).join('\\n'); const blocksInfo = input.blocks.slice(0, 30).map(b => '- ' + b.slug + ': ' + b.name).join('\\n'); const prompt = 'You are a workflow modification AI. Modify the workflow based on user feedback.\\n\\n## Current Workflow\\nName: ' + workflow.name + '\\nDescription: ' + workflow.description + '\\n\\n### Steps\\n' + stepsInfo + '\\n\\n### Edges (Connections)\\n' + edgesInfo + '\\n\\n## User Feedback\\n' + input.feedback + '\\n\\n## Available Blocks\\n' + blocksInfo + '\\n\\n## Output Format (JSON Only)\\n{\\n  \"changes\": [\\n    {\\n      \"type\": \"add_step|remove_step|modify_step|add_edge|remove_edge|modify_config\",\\n      \"step_id\": \"existing step ID (for modify/remove)\",\\n      \"step\": { new step info (for add) },\\n      \"edge\": { edge info (for edge operations) },\\n      \"config\": { config changes (for modify_config) },\\n      \"reason\": \"reason for change\"\\n    }\\n  ],\\n  \"summary\": \"Summary of changes\",\\n  \"response\": \"Response message to user\"\\n}'; return { prompt: prompt, session_id: input.session_id, project_id: input.project_id, tenant_id: input.tenant_id, user_id: input.user_id, current_workflow: workflow };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"prompt": {"type": "string"},
							"session_id": {"type": "string"},
							"project_id": {"type": "string"},
							"tenant_id": {"type": "string"},
							"user_id": {"type": "string"},
							"current_workflow": {"type": "object"}
						}
					}
				}`),
			},
			{
				TempID:    "refine_llm",
				Name:      "Refine LLM",
				Type:      "llm",
				PositionX: 400,
				PositionY: 280,
				Config: json.RawMessage(`{
					"model": "claude-sonnet-4-20250514",
					"provider": "anthropic",
					"max_tokens": 2000,
					"temperature": 0.3,
					"user_prompt": "{{$.prompt}}",
					"system_prompt": "You are a workflow modification AI. Accurately understand user feedback and implement minimal changes to fulfill requirements. Always respond in valid JSON format only.",
					"passthrough_fields": ["session_id", "project_id", "tenant_id", "user_id", "current_workflow"]
				}`),
			},
			{
				TempID:    "refine_parse",
				Name:      "Parse Refine Result",
				Type:      "function",
				PositionX: 520,
				PositionY: 280,
				Config: json.RawMessage(`{
					"code": "try { let content = input.content || ''; if (content.startsWith('` + "```json" + `')) content = content.slice(7); if (content.startsWith('` + "```" + `')) content = content.slice(3); if (content.endsWith('` + "```" + `')) content = content.slice(0, -3); content = content.trim(); let jsonStart = content.indexOf('{'); let jsonEnd = content.lastIndexOf('}'); if (jsonStart >= 0 && jsonEnd > jsonStart) { content = content.slice(jsonStart, jsonEnd + 1); } content = content.split(String.fromCharCode(10)).join(' ').split(String.fromCharCode(13)).join(' '); const result = JSON.parse(content); return { success: true, changes: result.changes || [], summary: result.summary || '', response: result.response || '', session_id: input.session_id, project_id: input.project_id }; } catch (e) { return { success: false, error: 'Failed to parse LLM response: ' + e.message, session_id: input.session_id, project_id: input.project_id }; }",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"success": {"type": "boolean"},
							"changes": {"type": "array"},
							"summary": {"type": "string"},
							"response": {"type": "string"},
							"error": {"type": "string"},
							"session_id": {"type": "string"},
							"project_id": {"type": "string"}
						}
					}
				}`),
			},
			{
				TempID:    "refine_apply_changes",
				Name:      "Apply Changes",
				Type:      "function",
				PositionX: 640,
				PositionY: 280,
				Config: json.RawMessage(`{
					"code": "const projectId = input.project_id; const changes = input.changes || []; const appliedChanges = []; for (const change of changes) { try { switch (change.type) { case 'add_step': const newStep = ctx.steps.create({ project_id: projectId, name: change.step.name, type: change.step.type, config: change.step.config || {}, position_x: change.step.position_x || 40, position_y: change.step.position_y || 40 }); appliedChanges.push({ type: 'add_step', step_id: newStep.id, name: change.step.name }); break; case 'remove_step': ctx.steps.delete(change.step_id); appliedChanges.push({ type: 'remove_step', step_id: change.step_id }); break; case 'modify_step': case 'modify_config': ctx.steps.update(change.step_id, change.config || change.step); appliedChanges.push({ type: 'modify_step', step_id: change.step_id }); break; case 'add_edge': ctx.edges.create({ project_id: projectId, source_step_id: change.edge.source_step_id, target_step_id: change.edge.target_step_id, source_port: change.edge.source_port || 'output', target_port: change.edge.target_port || 'input' }); appliedChanges.push({ type: 'add_edge' }); break; case 'remove_edge': ctx.edges.delete(change.edge.id); appliedChanges.push({ type: 'remove_edge' }); break; } } catch (e) { appliedChanges.push({ type: change.type, error: e.message }); } } ctx.projects.incrementVersion(projectId); ctx.builderSessions.addMessage(input.session_id, { role: 'assistant', content: input.response }); return { success: true, applied_changes: appliedChanges, summary: input.summary, response: input.response };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"success": {"type": "boolean"},
							"applied_changes": {"type": "array"},
							"summary": {"type": "string"},
							"response": {"type": "string"}
						}
					}
				}`),
			},

			// ========================================
			// Agent-Based Construct Entry Point (Y=340)
			// 4-Agent Architecture for high-quality workflow generation
			// ========================================
			{
				TempID:      "start_agent_construct",
				Name:        "Start: Agent Construct",
				Type:        "start",
				TriggerType: "internal",
				TriggerConfig: json.RawMessage(`{
					"entry_point": "agent_construct",
					"description": "Agent-based workflow construction with ConfigSchema awareness"
				}`),
				PositionX: 40,
				PositionY: 340,
				Config: json.RawMessage(`{
					"input_schema": {
						"type": "object",
						"required": ["session_id"],
						"properties": {
							"session_id": {"type": "string", "title": "Session ID"},
							"tenant_id": {"type": "string", "title": "Tenant ID"},
							"user_id": {"type": "string", "title": "User ID"}
						}
					}
				}`),
			},

			// Agent 1: Requirement Analysis
			{
				TempID:    "agent1_get_context",
				Name:      "Agent1: Get Context",
				Type:      "function",
				PositionX: 160,
				PositionY: 340,
				Config: json.RawMessage(`{
					"code": "const session = ctx.builderSessions.get(input.session_id); const blocks = ctx.blocks.list(); return { session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, spec: session?.spec || {}, messages: session?.messages || [], blocks: blocks.map(b => ({ slug: b.slug, name: b.name, description: b.description, category: b.category })) };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"session_id": {"type": "string"},
							"tenant_id": {"type": "string"},
							"user_id": {"type": "string"},
							"spec": {"type": "object"},
							"messages": {"type": "array"},
							"blocks": {"type": "array"}
						}
					}
				}`),
			},
			{
				TempID:    "agent1_build_prompt",
				Name:      "Agent1: Build Requirement Prompt",
				Type:      "function",
				PositionX: 280,
				PositionY: 340,
				Config: json.RawMessage(`{
					"code": "const spec = input.spec; const messages = input.messages; const historyText = messages.map(m => (m.role === 'user' ? 'User: ' : 'AI: ') + m.content).slice(-5).join('\\n'); const blocksInfo = input.blocks.slice(0, 30).map(b => '- ' + b.slug + ': ' + b.name + ' (' + b.category + ')').join('\\n'); const prompt = 'You are Agent 1: Requirement Analyzer.\\n\\n## Your Task\\nAnalyze the workflow specification and:\\n1. Clarify any ambiguous requirements\\n2. Identify missing information\\n3. Generate a clear, structured specification\\n\\n## Current Specification\\n' + JSON.stringify(spec, null, 2) + '\\n\\n## Conversation History\\n' + (historyText || '(No history)') + '\\n\\n## Available Block Categories\\n' + blocksInfo + '\\n\\n## Output Format (JSON Only)\\n{\\n  \"clarified_spec\": {\\n    \"name\": \"Workflow name\",\\n    \"description\": \"Clear description\",\\n    \"purpose\": \"Main goal\",\\n    \"trigger\": {\"type\": \"manual|schedule|webhook|event\", \"description\": \"Trigger details\"},\\n    \"expected_steps\": [\"Step 1 description\", \"Step 2 description\"],\\n    \"integrations_needed\": [\"service names\"],\\n    \"input_requirements\": [\"what data is needed\"],\\n    \"output_requirements\": [\"what should be produced\"]\\n  },\\n  \"assumptions\": [{\"item\": \"what was assumed\", \"default\": \"the value chosen\", \"confidence\": \"high|medium|low\"}],\\n  \"missing_info\": [\"things that could not be determined\"],\\n  \"ready_for_design\": true\\n}\\n\\n## Important\\n- Output valid JSON only\\n- Make reasonable assumptions for unclear parts\\n- Set ready_for_design to true if spec is complete enough'; return { prompt: prompt, session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, original_spec: spec, blocks: input.blocks };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"prompt": {"type": "string"},
							"session_id": {"type": "string"},
							"tenant_id": {"type": "string"},
							"user_id": {"type": "string"},
							"original_spec": {"type": "object"},
							"blocks": {"type": "array"}
						}
					}
				}`),
			},
			{
				TempID:    "agent1_llm",
				Name:      "Agent1: Analyze Requirements",
				Type:      "llm",
				PositionX: 400,
				PositionY: 340,
				Config: json.RawMessage(`{
					"model": "claude-sonnet-4-20250514",
					"provider": "anthropic",
					"max_tokens": 2000,
					"temperature": 0.2,
					"user_prompt": "{{$.prompt}}",
					"system_prompt": "You are a requirement analysis agent. Your job is to clarify and structure workflow requirements. Output valid JSON only - no markdown.",
					"passthrough_fields": ["session_id", "tenant_id", "user_id", "original_spec", "blocks"]
				}`),
			},
			{
				TempID:    "agent1_parse",
				Name:      "Agent1: Parse Result",
				Type:      "function",
				PositionX: 520,
				PositionY: 340,
				Config: json.RawMessage(`{
					"code": "try { let content = input.content || ''; if (content.startsWith('` + "```json" + `')) content = content.slice(7); if (content.startsWith('` + "```" + `')) content = content.slice(3); if (content.endsWith('` + "```" + `')) content = content.slice(0, -3); content = content.trim(); let jsonStart = content.indexOf('{'); let jsonEnd = content.lastIndexOf('}'); if (jsonStart >= 0 && jsonEnd > jsonStart) { content = content.slice(jsonStart, jsonEnd + 1); } content = content.split(String.fromCharCode(10)).join(' ').split(String.fromCharCode(13)).join(' '); const result = JSON.parse(content); return { success: true, clarified_spec: result.clarified_spec || input.original_spec, assumptions: result.assumptions || [], missing_info: result.missing_info || [], ready_for_design: result.ready_for_design !== false, session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, blocks: input.blocks }; } catch (e) { return { success: false, error: e.message, clarified_spec: input.original_spec, session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, blocks: input.blocks }; }",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"success": {"type": "boolean"},
							"clarified_spec": {"type": "object"},
							"assumptions": {"type": "array"},
							"missing_info": {"type": "array"},
							"ready_for_design": {"type": "boolean"},
							"session_id": {"type": "string"},
							"tenant_id": {"type": "string"},
							"user_id": {"type": "string"},
							"blocks": {"type": "array"},
							"error": {"type": "string"}
						}
					}
				}`),
			},

			// Agent 2: Structure Design
			{
				TempID:    "agent2_build_prompt",
				Name:      "Agent2: Build Structure Prompt",
				Type:      "function",
				PositionX: 640,
				PositionY: 340,
				Config: json.RawMessage(`{
					"code": "const spec = input.clarified_spec; const blocksInfo = input.blocks.map(b => '- ' + b.slug + ': ' + b.name + ' (' + b.category + ') - ' + (b.description || '')).join('\\n'); const prompt = 'You are Agent 2: Structure Designer.\\n\\n## Your Task\\nDesign the workflow DAG structure based on the clarified specification.\\n\\n## Clarified Specification\\n' + JSON.stringify(spec, null, 2) + '\\n\\n## Assumptions Made\\n' + JSON.stringify(input.assumptions, null, 2) + '\\n\\n## Available Blocks\\n' + blocksInfo + '\\n\\n## Output Format (JSON Only)\\n{\\n  \"workflow_name\": \"Name\",\\n  \"workflow_description\": \"Description\",\\n  \"step_skeletons\": [\\n    {\\n      \"temp_id\": \"step_1\",\\n      \"name\": \"Step name\",\\n      \"description\": \"What this step does\",\\n      \"type\": \"start|llm|tool|function|condition|...\",\\n      \"block_slug\": \"slug of block to use (if applicable)\",\\n      \"purpose\": \"Why this step exists\",\\n      \"needs_config\": true,\\n      \"depends_on\": [\"step_id that must complete first\"]\\n    }\\n  ],\\n  \"edges\": [\\n    {\\n      \"source\": \"step_1\",\\n      \"target\": \"step_2\",\\n      \"label\": \"condition or description\"\\n    }\\n  ],\\n  \"design_notes\": \"Explanation of design decisions\"\\n}\\n\\n## Step Types\\n- start: Entry point (required first)\\n- llm: AI/LLM call\\n- tool: External adapter using block_slug\\n- function: Custom JavaScript\\n- condition: True/false branch\\n- loop: Iteration\\n\\n## Important\\n- Always start with a \"start\" type step\\n- Set needs_config: true for steps that require configuration\\n- Use block_slug to reference available blocks\\n- Output valid JSON only'; return { prompt: prompt, session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, clarified_spec: spec, assumptions: input.assumptions, blocks: input.blocks };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"prompt": {"type": "string"},
							"session_id": {"type": "string"},
							"tenant_id": {"type": "string"},
							"user_id": {"type": "string"},
							"clarified_spec": {"type": "object"},
							"assumptions": {"type": "array"},
							"blocks": {"type": "array"}
						}
					}
				}`),
			},
			{
				TempID:    "agent2_llm",
				Name:      "Agent2: Design Structure",
				Type:      "llm",
				PositionX: 760,
				PositionY: 340,
				Config: json.RawMessage(`{
					"model": "claude-sonnet-4-20250514",
					"provider": "anthropic",
					"max_tokens": 3000,
					"temperature": 0.3,
					"user_prompt": "{{$.prompt}}",
					"system_prompt": "You are a workflow structure design agent. Design optimal DAG structures for workflows. Output valid JSON only - no markdown.",
					"passthrough_fields": ["session_id", "tenant_id", "user_id", "clarified_spec", "assumptions", "blocks"]
				}`),
			},
			{
				TempID:    "agent2_parse",
				Name:      "Agent2: Parse Structure",
				Type:      "function",
				PositionX: 880,
				PositionY: 340,
				Config: json.RawMessage(`{
					"code": "try { let content = input.content || ''; if (content.startsWith('` + "```json" + `')) content = content.slice(7); if (content.startsWith('` + "```" + `')) content = content.slice(3); if (content.endsWith('` + "```" + `')) content = content.slice(0, -3); content = content.trim(); let jsonStart = content.indexOf('{'); let jsonEnd = content.lastIndexOf('}'); if (jsonStart >= 0 && jsonEnd > jsonStart) { content = content.slice(jsonStart, jsonEnd + 1); } content = content.split(String.fromCharCode(10)).join(' ').split(String.fromCharCode(13)).join(' '); const result = JSON.parse(content); const stepsNeedingConfig = (result.step_skeletons || []).filter(s => s.block_slug || (s.needs_config && s.type !== 'start')); return { success: true, workflow_name: result.workflow_name || 'Generated Workflow', workflow_description: result.workflow_description || '', step_skeletons: result.step_skeletons || [], edges: result.edges || [], design_notes: result.design_notes || '', steps_needing_config: stepsNeedingConfig, current_step_index: 0, configured_steps: [], session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, clarified_spec: input.clarified_spec, blocks: input.blocks }; } catch (e) { return { success: false, error: e.message, session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id }; }",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"success": {"type": "boolean"},
							"workflow_name": {"type": "string"},
							"workflow_description": {"type": "string"},
							"step_skeletons": {"type": "array"},
							"edges": {"type": "array"},
							"design_notes": {"type": "string"},
							"steps_needing_config": {"type": "array"},
							"current_step_index": {"type": "number"},
							"configured_steps": {"type": "array"},
							"session_id": {"type": "string"},
							"tenant_id": {"type": "string"},
							"user_id": {"type": "string"},
							"clarified_spec": {"type": "object"},
							"blocks": {"type": "array"},
							"error": {"type": "string"}
						}
					}
				}`),
			},

			// Agent 3: Step Configuration Loop
			{
				TempID:    "agent3_prepare_step",
				Name:      "Agent3: Prepare Step Config",
				Type:      "function",
				PositionX: 160,
				PositionY: 400,
				Config: json.RawMessage(`{
					"code": "const steps = input.steps_needing_config || []; const idx = input.current_step_index || 0; if (idx >= steps.length) { return { loop_complete: true, configured_steps: input.configured_steps || [], step_skeletons: input.step_skeletons, edges: input.edges, workflow_name: input.workflow_name, workflow_description: input.workflow_description, session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, clarified_spec: input.clarified_spec }; } const currentStep = steps[idx]; const blockInfo = ctx.blocks.getWithSchema(currentStep.block_slug); const previousSteps = input.configured_steps || []; const previousOutputs = previousSteps.map(s => ({ step_id: s.temp_id, step_name: s.name, output_fields: s.output_fields || [] })); return { loop_complete: false, current_step: currentStep, block_info: blockInfo, previous_outputs: previousOutputs, current_step_index: idx, steps_needing_config: steps, configured_steps: input.configured_steps || [], step_skeletons: input.step_skeletons, edges: input.edges, workflow_name: input.workflow_name, workflow_description: input.workflow_description, session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, clarified_spec: input.clarified_spec };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"loop_complete": {"type": "boolean"},
							"current_step": {"type": "object"},
							"block_info": {"type": "object"},
							"previous_outputs": {"type": "array"},
							"current_step_index": {"type": "number"},
							"steps_needing_config": {"type": "array"},
							"configured_steps": {"type": "array"},
							"step_skeletons": {"type": "array"},
							"edges": {"type": "array"},
							"workflow_name": {"type": "string"},
							"workflow_description": {"type": "string"},
							"session_id": {"type": "string"},
							"tenant_id": {"type": "string"},
							"user_id": {"type": "string"},
							"clarified_spec": {"type": "object"}
						}
					}
				}`),
			},
			{
				TempID:    "agent3_check_loop",
				Name:      "Agent3: Check Loop",
				Type:      "condition",
				PositionX: 280,
				PositionY: 400,
				Config: json.RawMessage(`{
					"expression": "$.loop_complete == true",
					"true_label": "All steps configured",
					"false_label": "More steps to configure"
				}`),
			},
			{
				TempID:    "agent3_build_prompt",
				Name:      "Agent3: Build Config Prompt",
				Type:      "function",
				PositionX: 400,
				PositionY: 400,
				Config: json.RawMessage(`{
					"code": "const step = input.current_step; const block = input.block_info; const prevOutputs = input.previous_outputs || []; const outputsInfo = prevOutputs.map(o => '- ' + o.step_name + ': Use {{' + o.step_id + '.field}} to reference').join('\\n'); const requiredFields = block.required_fields || []; const defaultValues = block.resolved_config_defaults || {}; const prompt = 'You are Agent 3: Step Configurator.\\n\\n## Your Task\\nGenerate the config for step \"' + step.name + '\".\\n\\n## Step Information\\n- Name: ' + step.name + '\\n- Description: ' + step.description + '\\n- Purpose: ' + step.purpose + '\\n\\n## Block Information\\n- Slug: ' + block.slug + '\\n- Name: ' + block.name + '\\n\\n## ConfigSchema (IMPORTANT - Follow This Exactly)\\n' + JSON.stringify(block.config_schema, null, 2) + '\\n\\n## Required Fields (Must Be Set)\\n' + (requiredFields.length > 0 ? requiredFields.map(f => '- ' + f).join('\\n') : '(None specified)') + '\\n\\n## Default Values (Can Override)\\n' + JSON.stringify(defaultValues, null, 2) + '\\n\\n## Previous Step Outputs (Use for Template Variables)\\n' + (outputsInfo || '(No previous steps)') + '\\n\\n## Workflow Context\\n' + JSON.stringify(input.clarified_spec, null, 2) + '\\n\\n## Output Format (JSON Only)\\n{\\n  \"config\": {\\n    /* Config object matching the ConfigSchema above */\\n    /* Use {{step_id.field}} for dynamic values from previous steps */\\n  },\\n  \"output_fields\": [\"field1\", \"field2\"],\\n  \"reasoning\": \"Brief explanation of config choices\"\\n}\\n\\n## Important\\n- Follow ConfigSchema structure exactly\\n- Set ALL required fields\\n- Use template variables like {{step_id.field}} for dynamic data\\n- Output valid JSON only'; return { prompt: prompt, current_step: step, block_info: block, current_step_index: input.current_step_index, steps_needing_config: input.steps_needing_config, configured_steps: input.configured_steps, step_skeletons: input.step_skeletons, edges: input.edges, workflow_name: input.workflow_name, workflow_description: input.workflow_description, session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, clarified_spec: input.clarified_spec };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"prompt": {"type": "string"},
							"current_step": {"type": "object"},
							"block_info": {"type": "object"},
							"current_step_index": {"type": "number"},
							"steps_needing_config": {"type": "array"},
							"configured_steps": {"type": "array"},
							"step_skeletons": {"type": "array"},
							"edges": {"type": "array"},
							"workflow_name": {"type": "string"},
							"workflow_description": {"type": "string"},
							"session_id": {"type": "string"},
							"tenant_id": {"type": "string"},
							"user_id": {"type": "string"},
							"clarified_spec": {"type": "object"}
						}
					}
				}`),
			},
			{
				TempID:    "agent3_llm",
				Name:      "Agent3: Generate Config",
				Type:      "llm",
				PositionX: 520,
				PositionY: 400,
				Config: json.RawMessage(`{
					"model": "claude-sonnet-4-20250514",
					"provider": "anthropic",
					"max_tokens": 2000,
					"temperature": 0.2,
					"user_prompt": "{{$.prompt}}",
					"system_prompt": "You are a step configuration agent. Generate valid config objects that match the provided ConfigSchema exactly. Pay close attention to required fields and data types. Output valid JSON only - no markdown.",
					"passthrough_fields": ["current_step", "block_info", "current_step_index", "steps_needing_config", "configured_steps", "step_skeletons", "edges", "workflow_name", "workflow_description", "session_id", "tenant_id", "user_id", "clarified_spec"]
				}`),
			},
			{
				TempID:    "agent3_parse_config",
				Name:      "Agent3: Parse & Validate Config",
				Type:      "function",
				PositionX: 640,
				PositionY: 400,
				Config: json.RawMessage(`{
					"code": "try { let content = input.content || ''; if (content.startsWith('` + "```json" + `')) content = content.slice(7); if (content.startsWith('` + "```" + `')) content = content.slice(3); if (content.endsWith('` + "```" + `')) content = content.slice(0, -3); content = content.trim(); let jsonStart = content.indexOf('{'); let jsonEnd = content.lastIndexOf('}'); if (jsonStart >= 0 && jsonEnd > jsonStart) { content = content.slice(jsonStart, jsonEnd + 1); } content = content.split(String.fromCharCode(10)).join(' ').split(String.fromCharCode(13)).join(' '); const result = JSON.parse(content); const step = input.current_step; const configuredStep = { temp_id: step.temp_id, name: step.name, type: step.type, block_slug: step.block_slug, description: step.description, config: result.config || {}, output_fields: result.output_fields || [], reasoning: result.reasoning || '' }; const newConfigured = [...(input.configured_steps || []), configuredStep]; const nextIndex = (input.current_step_index || 0) + 1; const stepsNeedingConfig = input.steps_needing_config || []; const loopComplete = nextIndex >= stepsNeedingConfig.length; return { success: true, loop_complete: loopComplete, current_step_index: nextIndex, steps_needing_config: stepsNeedingConfig, configured_steps: newConfigured, step_skeletons: input.step_skeletons, edges: input.edges, workflow_name: input.workflow_name, workflow_description: input.workflow_description, session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, clarified_spec: input.clarified_spec }; } catch (e) { const step = input.current_step; const configuredStep = { temp_id: step.temp_id, name: step.name, type: step.type, block_slug: step.block_slug, description: step.description, config: {}, output_fields: [], error: e.message }; const newConfigured = [...(input.configured_steps || []), configuredStep]; const nextIndex = (input.current_step_index || 0) + 1; const stepsNeedingConfig = input.steps_needing_config || []; const loopComplete = nextIndex >= stepsNeedingConfig.length; return { success: false, loop_complete: loopComplete, error: e.message, current_step_index: nextIndex, steps_needing_config: stepsNeedingConfig, configured_steps: newConfigured, step_skeletons: input.step_skeletons, edges: input.edges, workflow_name: input.workflow_name, workflow_description: input.workflow_description, session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, clarified_spec: input.clarified_spec }; }",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"success": {"type": "boolean"},
							"loop_complete": {"type": "boolean"},
							"error": {"type": "string"},
							"current_step_index": {"type": "number"},
							"steps_needing_config": {"type": "array"},
							"configured_steps": {"type": "array"},
							"step_skeletons": {"type": "array"},
							"edges": {"type": "array"},
							"workflow_name": {"type": "string"},
							"workflow_description": {"type": "string"},
							"session_id": {"type": "string"},
							"tenant_id": {"type": "string"},
							"user_id": {"type": "string"},
							"clarified_spec": {"type": "object"}
						}
					}
				}`),
			},

			// Agent 4: Validation & Final Assembly
			{
				TempID:    "agent4_merge_steps",
				Name:      "Agent4: Merge All Steps",
				Type:      "function",
				PositionX: 160,
				PositionY: 460,
				Config: json.RawMessage(`{
					"code": "const skeletons = input.step_skeletons || []; const configured = input.configured_steps || []; const configuredMap = {}; configured.forEach(c => { configuredMap[c.temp_id] = c; }); const mergedSteps = skeletons.map((s, i) => { const cfg = configuredMap[s.temp_id]; return { temp_id: s.temp_id, name: s.name, type: s.type, description: s.description, block_slug: s.block_slug || cfg?.block_slug, config: cfg?.config || s.config || {}, position_x: 40 + (i * 120), position_y: 40 }; }); return { merged_steps: mergedSteps, edges: input.edges, workflow_name: input.workflow_name, workflow_description: input.workflow_description, session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, clarified_spec: input.clarified_spec, configured_count: configured.length, total_steps: skeletons.length };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"merged_steps": {"type": "array"},
							"edges": {"type": "array"},
							"workflow_name": {"type": "string"},
							"workflow_description": {"type": "string"},
							"session_id": {"type": "string"},
							"tenant_id": {"type": "string"},
							"user_id": {"type": "string"},
							"clarified_spec": {"type": "object"},
							"configured_count": {"type": "number"},
							"total_steps": {"type": "number"}
						}
					}
				}`),
			},
			{
				TempID:    "agent4_build_validation_prompt",
				Name:      "Agent4: Build Validation Prompt",
				Type:      "function",
				PositionX: 280,
				PositionY: 460,
				Config: json.RawMessage(`{
					"code": "const steps = input.merged_steps; const edges = input.edges; const stepsInfo = steps.map(s => '- ' + s.temp_id + ': ' + s.name + ' (type: ' + s.type + ')\\n  Config: ' + JSON.stringify(s.config)).join('\\n'); const edgesInfo = edges.map(e => '- ' + e.source + ' -> ' + e.target + (e.label ? ' [' + e.label + ']' : '')).join('\\n'); const prompt = 'You are Agent 4: Workflow Validator.\\n\\n## Your Task\\nValidate the workflow and fix any issues.\\n\\n## Workflow\\nName: ' + input.workflow_name + '\\nDescription: ' + input.workflow_description + '\\n\\n## Steps\\n' + stepsInfo + '\\n\\n## Edges (Connections)\\n' + edgesInfo + '\\n\\n## Specification\\n' + JSON.stringify(input.clarified_spec, null, 2) + '\\n\\n## Validation Checks\\n1. Does workflow have a start step?\\n2. Are all edges valid (source and target exist)?\\n3. Are required configs filled?\\n4. Are template variables valid (reference existing steps)?\\n5. Is the workflow logically complete?\\n\\n## Output Format (JSON Only)\\n{\\n  \"is_valid\": true,\\n  \"issues\": [{\"step_id\": \"...\", \"issue\": \"...\", \"fix\": \"...\"}],\\n  \"fixes_applied\": [{\"step_id\": \"...\", \"field\": \"...\", \"old_value\": ..., \"new_value\": ...}],\\n  \"final_steps\": [/* corrected steps array */],\\n  \"final_edges\": [/* corrected edges array */],\\n  \"summary\": \"Validation summary\"\\n}\\n\\n## Important\\n- If is_valid is false, provide fixes in fixes_applied\\n- final_steps and final_edges should be corrected versions\\n- Output valid JSON only'; return { prompt: prompt, merged_steps: steps, edges: edges, workflow_name: input.workflow_name, workflow_description: input.workflow_description, session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, clarified_spec: input.clarified_spec };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"prompt": {"type": "string"},
							"merged_steps": {"type": "array"},
							"edges": {"type": "array"},
							"workflow_name": {"type": "string"},
							"workflow_description": {"type": "string"},
							"session_id": {"type": "string"},
							"tenant_id": {"type": "string"},
							"user_id": {"type": "string"},
							"clarified_spec": {"type": "object"}
						}
					}
				}`),
			},
			{
				TempID:    "agent4_llm",
				Name:      "Agent4: Validate Workflow",
				Type:      "llm",
				PositionX: 400,
				PositionY: 460,
				Config: json.RawMessage(`{
					"model": "claude-sonnet-4-20250514",
					"provider": "anthropic",
					"max_tokens": 3000,
					"temperature": 0.1,
					"user_prompt": "{{$.prompt}}",
					"system_prompt": "You are a workflow validation agent. Check for errors and fix them. Be thorough but practical. Output valid JSON only - no markdown.",
					"passthrough_fields": ["merged_steps", "edges", "workflow_name", "workflow_description", "session_id", "tenant_id", "user_id", "clarified_spec"]
				}`),
			},
			{
				TempID:    "agent4_parse_validation",
				Name:      "Agent4: Parse Validation",
				Type:      "function",
				PositionX: 520,
				PositionY: 460,
				Config: json.RawMessage(`{
					"code": "try { let content = input.content || ''; if (content.startsWith('` + "```json" + `')) content = content.slice(7); if (content.startsWith('` + "```" + `')) content = content.slice(3); if (content.endsWith('` + "```" + `')) content = content.slice(0, -3); content = content.trim(); let jsonStart = content.indexOf('{'); let jsonEnd = content.lastIndexOf('}'); if (jsonStart >= 0 && jsonEnd > jsonStart) { content = content.slice(jsonStart, jsonEnd + 1); } content = content.split(String.fromCharCode(10)).join(' ').split(String.fromCharCode(13)).join(' '); const result = JSON.parse(content); return { success: true, is_valid: result.is_valid !== false, issues: result.issues || [], fixes_applied: result.fixes_applied || [], final_steps: result.final_steps || input.merged_steps, final_edges: result.final_edges || input.edges, summary: result.summary || '', workflow_name: input.workflow_name, workflow_description: input.workflow_description, session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, clarified_spec: input.clarified_spec }; } catch (e) { return { success: false, error: e.message, is_valid: true, final_steps: input.merged_steps, final_edges: input.edges, workflow_name: input.workflow_name, workflow_description: input.workflow_description, session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, clarified_spec: input.clarified_spec }; }",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"success": {"type": "boolean"},
							"is_valid": {"type": "boolean"},
							"issues": {"type": "array"},
							"fixes_applied": {"type": "array"},
							"final_steps": {"type": "array"},
							"final_edges": {"type": "array"},
							"summary": {"type": "string"},
							"workflow_name": {"type": "string"},
							"workflow_description": {"type": "string"},
							"session_id": {"type": "string"},
							"tenant_id": {"type": "string"},
							"user_id": {"type": "string"},
							"clarified_spec": {"type": "object"},
							"error": {"type": "string"}
						}
					}
				}`),
			},

			// Final: Create Project
			{
				TempID:    "agent_create_project",
				Name:      "Agent: Create Project",
				Type:      "function",
				PositionX: 640,
				PositionY: 460,
				Config: json.RawMessage(`{
					"code": "const sessionId = input.session_id; const tenantId = input.tenant_id; const userId = input.user_id; const steps = input.final_steps || []; const edges = input.final_edges || []; if (!steps.length) { return { success: false, error: 'No steps to create' }; } const projectOpts = { tenant_id: tenantId, name: input.workflow_name || 'AI Generated Workflow', description: input.workflow_description || '', status: 'draft' }; if (userId) { projectOpts.created_by = userId; } const project = ctx.projects.create(projectOpts); const stepIdMap = {}; for (let i = 0; i < steps.length; i++) { const step = steps[i]; const createdStep = ctx.steps.create({ tenant_id: tenantId, project_id: project.id, name: step.name, type: step.type, config: step.config || {}, position_x: step.position_x || (40 + (i * 120)), position_y: step.position_y || 40, block_slug: step.block_slug }); stepIdMap[step.temp_id] = createdStep.id; } for (const edge of edges) { const sourceId = stepIdMap[edge.source]; const targetId = stepIdMap[edge.target]; if (sourceId && targetId) { ctx.edges.create({ tenant_id: tenantId, project_id: project.id, source_step_id: sourceId, target_step_id: targetId, source_port: edge.source_port || 'output', target_port: edge.target_port || 'input' }); } } ctx.builderSessions.update(sessionId, { status: 'reviewing', project_id: project.id }); const summary = 'Agent-based construction complete.\\n\\n' + '**' + input.workflow_name + '**\\n' + (input.workflow_description || '') + '\\n\\n' + '### Steps Created (' + steps.length + ')\\n' + steps.map((s, i) => (i+1) + '. ' + s.name + ' (' + s.type + ')').join('\\n') + (input.summary ? '\\n\\n### Validation Notes\\n' + input.summary : ''); ctx.builderSessions.addMessage(sessionId, { role: 'assistant', content: summary }); return { success: true, project_id: project.id, workflow_name: input.workflow_name, steps_created: steps.length, edges_created: edges.length, validation_summary: input.summary || '', issues_found: input.issues || [], fixes_applied: input.fixes_applied || [] };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"success": {"type": "boolean"},
							"project_id": {"type": "string"},
							"workflow_name": {"type": "string"},
							"steps_created": {"type": "number"},
							"edges_created": {"type": "number"},
							"validation_summary": {"type": "string"},
							"issues_found": {"type": "array"},
							"fixes_applied": {"type": "array"},
							"error": {"type": "string"}
						}
					}
				}`),
			},
		},
		Edges: []SystemEdgeDefinition{
			// Analysis flow
			{SourceTempID: "start_analysis", TargetTempID: "analysis_get_context", SourcePort: "output"},
			{SourceTempID: "analysis_get_context", TargetTempID: "analysis_build_prompt", SourcePort: "output"},
			{SourceTempID: "analysis_build_prompt", TargetTempID: "analysis_llm", SourcePort: "output"},
			{SourceTempID: "analysis_llm", TargetTempID: "analysis_parse", SourcePort: "output"},
			{SourceTempID: "analysis_parse", TargetTempID: "analysis_update_session", SourcePort: "output"},

			// Proposal flow
			{SourceTempID: "start_proposal", TargetTempID: "proposal_get_context", SourcePort: "output"},
			{SourceTempID: "proposal_get_context", TargetTempID: "proposal_build_prompt", SourcePort: "output"},
			{SourceTempID: "proposal_build_prompt", TargetTempID: "proposal_llm", SourcePort: "output"},
			{SourceTempID: "proposal_llm", TargetTempID: "proposal_parse", SourcePort: "output"},
			{SourceTempID: "proposal_parse", TargetTempID: "proposal_update_session", SourcePort: "output"},

			// Construct flow
			{SourceTempID: "start_construct", TargetTempID: "construct_get_spec", SourcePort: "output"},
			{SourceTempID: "construct_get_spec", TargetTempID: "construct_build_prompt", SourcePort: "output"},
			{SourceTempID: "construct_build_prompt", TargetTempID: "construct_llm", SourcePort: "output"},
			{SourceTempID: "construct_llm", TargetTempID: "construct_parse", SourcePort: "output"},
			{SourceTempID: "construct_parse", TargetTempID: "construct_create_project", SourcePort: "output"},

			// Refine flow
			{SourceTempID: "start_refine", TargetTempID: "refine_get_workflow", SourcePort: "output"},
			{SourceTempID: "refine_get_workflow", TargetTempID: "refine_build_prompt", SourcePort: "output"},
			{SourceTempID: "refine_build_prompt", TargetTempID: "refine_llm", SourcePort: "output"},
			{SourceTempID: "refine_llm", TargetTempID: "refine_parse", SourcePort: "output"},
			{SourceTempID: "refine_parse", TargetTempID: "refine_apply_changes", SourcePort: "output"},

			// Agent-based Construct flow
			// Agent 1: Requirements Analysis
			{SourceTempID: "start_agent_construct", TargetTempID: "agent1_get_context", SourcePort: "output"},
			{SourceTempID: "agent1_get_context", TargetTempID: "agent1_build_prompt", SourcePort: "output"},
			{SourceTempID: "agent1_build_prompt", TargetTempID: "agent1_llm", SourcePort: "output"},
			{SourceTempID: "agent1_llm", TargetTempID: "agent1_parse", SourcePort: "output"},
			// Agent 2: Structure Design
			{SourceTempID: "agent1_parse", TargetTempID: "agent2_build_prompt", SourcePort: "output"},
			{SourceTempID: "agent2_build_prompt", TargetTempID: "agent2_llm", SourcePort: "output"},
			{SourceTempID: "agent2_llm", TargetTempID: "agent2_parse", SourcePort: "output"},
			// Agent 3: Step Configuration Loop
			{SourceTempID: "agent2_parse", TargetTempID: "agent3_prepare_step", SourcePort: "output"},
			{SourceTempID: "agent3_prepare_step", TargetTempID: "agent3_check_loop", SourcePort: "output"},
			{SourceTempID: "agent3_check_loop", TargetTempID: "agent3_build_prompt", SourcePort: "false"},      // More steps to configure
			{SourceTempID: "agent3_check_loop", TargetTempID: "agent4_merge_steps", SourcePort: "true"},        // All done, go to Agent 4
			{SourceTempID: "agent3_build_prompt", TargetTempID: "agent3_llm", SourcePort: "output"},
			{SourceTempID: "agent3_llm", TargetTempID: "agent3_parse_config", SourcePort: "output"},
			{SourceTempID: "agent3_parse_config", TargetTempID: "agent3_check_loop", SourcePort: "output"},     // Loop back to check if more steps
			// Agent 4: Validation & Final Assembly
			{SourceTempID: "agent4_merge_steps", TargetTempID: "agent4_build_validation_prompt", SourcePort: "output"},
			{SourceTempID: "agent4_build_validation_prompt", TargetTempID: "agent4_llm", SourcePort: "output"},
			{SourceTempID: "agent4_llm", TargetTempID: "agent4_parse_validation", SourcePort: "output"},
			{SourceTempID: "agent4_parse_validation", TargetTempID: "agent_create_project", SourcePort: "output"},
		},
	}
}
