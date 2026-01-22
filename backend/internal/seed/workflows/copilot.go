package workflows

import "encoding/json"

func (r *Registry) registerCopilotWorkflows() {
	r.register(CopilotWorkflow())
}

// CopilotWorkflow is the system workflow for the Copilot AI assistant.
// It uses the Agent BlockGroup with child steps as tools to help users:
// - Build and modify workflows
// - Understand platform features
// - Get help with block configuration
//
// Architecture:
// - Start → Set Context → Agent Group (with tool steps inside)
// - Tool steps inside the Agent Group are automatically available as tools
// - The agent uses ReAct loop to call tools and generate responses
//
// This workflow serves as a dogfooding example - it's built using
// the same blocks available to all users, demonstrating that
// sophisticated AI agents can be created with the platform.
func CopilotWorkflow() *SystemWorkflowDefinition {
	return &SystemWorkflowDefinition{
		ID:          "a0000000-0000-0000-0000-000000000201",
		SystemSlug:  "copilot",
		Name:        "Copilot AI Assistant",
		Description: "AI assistant for workflow building and platform guidance",
		Version:     16,
		IsSystem:    true,
		Steps: []SystemStepDefinition{
			// ============================
			// Start - Main Entry Point
			// ============================
			{
				TempID:      "start",
				Name:        "Start",
				Type:        "start",
				TriggerType: "internal",
				TriggerConfig: json.RawMessage(`{
					"entry_point": "chat",
					"description": "Copilot chat entry point"
				}`),
				PositionX: 40,
				PositionY: 300,
				Config: json.RawMessage(`{
					"input_schema": {
						"type": "object",
						"required": ["message"],
						"properties": {
							"message": {"type": "string", "description": "User's message"},
							"mode": {"type": "string", "enum": ["create", "explain", "enhance"], "description": "Copilot mode"},
							"workflow_id": {"type": "string", "description": "Target workflow ID (for enhance mode)"},
							"session_id": {"type": "string", "description": "Session ID for memory"}
						}
					}
				}`),
			},

			// ============================
			// Set Variables - Context injection
			// ============================
			{
				TempID:    "set_context",
				Name:      "Set Context",
				Type:      "set-variables",
				PositionX: 160,
				PositionY: 300,
				BlockSlug: "set-variables",
				Config: json.RawMessage(`{
					"variables": [
						{"name": "mode", "value": "{{$.mode}}", "type": "string"},
						{"name": "workflow_id", "value": "{{$.workflow_id}}", "type": "string"},
						{"name": "session_id", "value": "{{$.session_id}}", "type": "string"},
						{"name": "user_message", "value": "{{$.message}}", "type": "string"}
					],
					"merge_input": true
				}`),
			},

			// ============================
			// Tool Steps (inside Agent Group)
			// These steps are child steps of the Agent Group and become tools automatically
			// ============================

			// Block Tools
			{
				TempID:           "list_blocks",
				Name:             "list_blocks",
				Type:             "function",
				PositionX:        300,
				PositionY:        140,
				BlockGroupTempID: "copilot_agent_group",
				Config: json.RawMessage(`{
					"code": "const blocks = ctx.blocks.list(); return { blocks: blocks.map(b => ({ slug: b.slug, name: b.name, category: b.category, description: b.description })) };",
					"description": "Get a list of all available blocks with their basic information (slug, name, category, description)"
				}`),
			},
			{
				TempID:           "get_block_schema",
				Name:             "get_block_schema",
				Type:             "function",
				PositionX:        460,
				PositionY:        140,
				BlockGroupTempID: "copilot_agent_group",
				Config: json.RawMessage(`{
					"code": "if (!input.slug) return { error: 'slug is required' }; const block = ctx.blocks.getWithSchema(input.slug); if (!block) return { error: 'Block not found: ' + input.slug }; return block;",
					"description": "Get the detailed configuration schema for a specific block",
					"input_schema": {
						"type": "object",
						"required": ["slug"],
						"properties": {
							"slug": {"type": "string", "description": "The block's slug identifier (e.g., 'llm', 'http', 'slack')"}
						}
					}
				}`),
			},

			// Workflow Tools
			{
				TempID:           "list_workflows",
				Name:             "list_workflows",
				Type:             "function",
				PositionX:        300,
				PositionY:        240,
				BlockGroupTempID: "copilot_agent_group",
				Config: json.RawMessage(`{
					"code": "const workflows = ctx.workflows.list(); return { workflows: workflows.map(w => ({ id: w.id, name: w.name, description: w.description })), count: workflows.length };",
					"description": "Get a list of all workflows accessible to the user"
				}`),
			},
			{
				TempID:           "get_workflow",
				Name:             "get_workflow",
				Type:             "function",
				PositionX:        460,
				PositionY:        240,
				BlockGroupTempID: "copilot_agent_group",
				Config: json.RawMessage(`{
					"code": "const wfId = input.workflow_id || ctx.targetProjectId; if (!wfId) return { error: 'workflow_id is required (or Copilot must be opened from a workflow)' }; const wf = ctx.workflows.get(wfId); if (!wf) return { error: 'Workflow not found: ' + wfId }; return wf;",
					"description": "Get detailed information about a specific workflow. If workflow_id is not provided, uses the current workflow.",
					"input_schema": {
						"type": "object",
						"properties": {
							"workflow_id": {"type": "string", "description": "The workflow's UUID (optional - defaults to current workflow)"}
						}
					}
				}`),
			},

			// Step Tools
			{
				TempID:           "update_step",
				Name:             "update_step",
				Type:             "function",
				PositionX:        460,
				PositionY:        340,
				BlockGroupTempID: "copilot_agent_group",
				Config: json.RawMessage(`{
					"code": "if (!input.step_id) return { error: 'step_id is required' }; const updates = {}; if (input.name) updates.name = input.name; if (input.config) updates.config = input.config; if (input.position_x !== undefined) updates.position_x = input.position_x; if (input.position_y !== undefined) updates.position_y = input.position_y; const step = ctx.steps.update(input.step_id, updates); return step;",
					"description": "Update an existing step's configuration or position",
					"input_schema": {
						"type": "object",
						"required": ["step_id"],
						"properties": {
							"step_id": {"type": "string", "description": "The step's UUID"},
							"name": {"type": "string", "description": "New name for the step"},
							"config": {"type": "object", "description": "Updated configuration"},
							"position_x": {"type": "integer", "description": "New X position"},
							"position_y": {"type": "integer", "description": "New Y position"}
						}
					}
				}`),
			},
			{
				TempID:           "delete_step",
				Name:             "delete_step",
				Type:             "function",
				PositionX:        620,
				PositionY:        340,
				BlockGroupTempID: "copilot_agent_group",
				Config: json.RawMessage(`{
					"code": "if (!input.step_id) return { error: 'step_id is required' }; ctx.steps.delete(input.step_id); return { success: true, deleted_step_id: input.step_id };",
					"description": "Delete a step from the workflow",
					"input_schema": {
						"type": "object",
						"required": ["step_id"],
						"properties": {
							"step_id": {"type": "string", "description": "The step's UUID to delete"}
						}
					}
				}`),
			},

			// Edge Tools
			{
				TempID:           "delete_edge",
				Name:             "delete_edge",
				Type:             "function",
				PositionX:        460,
				PositionY:        440,
				BlockGroupTempID: "copilot_agent_group",
				Config: json.RawMessage(`{
					"code": "if (!input.edge_id) return { error: 'edge_id is required' }; ctx.edges.delete(input.edge_id); return { success: true, deleted_edge_id: input.edge_id };",
					"description": "Delete a connection between steps",
					"input_schema": {
						"type": "object",
						"required": ["edge_id"],
						"properties": {
							"edge_id": {"type": "string", "description": "The edge's UUID to delete"}
						}
					}
				}`),
			},

			// Unified Workflow Structure Creation - Single API for steps + connections
			{
				TempID:           "create_workflow_structure",
				Name:             "create_workflow_structure",
				Type:             "function",
				PositionX:        620,
				PositionY:        440,
				BlockGroupTempID: "copilot_agent_group",
				Config:           json.RawMessage(createWorkflowStructureToolConfig()),
			},

			// Documentation Search (using RAG if available)
			{
				TempID:           "search_documentation",
				Name:             "search_documentation",
				Type:             "function",
				PositionX:        300,
				PositionY:        540,
				BlockGroupTempID: "copilot_agent_group",
				Config: json.RawMessage(`{
					"code": "if (!input.query) return { error: 'query is required' }; if (ctx.vector && ctx.embedding) { try { const embedding = ctx.embedding.embed('openai', 'text-embedding-3-small', [input.query]); const results = ctx.vector.query('platform-docs', embedding.vectors[0], { topK: 5 }); return { results: results.matches || [], query: input.query }; } catch(e) { return { error: 'Documentation search not available', query: input.query }; } } return { error: 'Vector service not available', query: input.query };",
					"description": "Search platform documentation for relevant information",
					"input_schema": {
						"type": "object",
						"required": ["query"],
						"properties": {
							"query": {"type": "string", "description": "Search query for documentation"}
						}
					}
				}`),
			},

			// Workflow Validation
			{
				TempID:           "validate_workflow",
				Name:             "validate_workflow",
				Type:             "function",
				PositionX:        460,
				PositionY:        540,
				BlockGroupTempID: "copilot_agent_group",
				Config: json.RawMessage(`{
					"code": "if (!input.workflow_id) return { error: 'workflow_id is required' }; const wf = ctx.workflows.get(input.workflow_id); if (!wf) return { error: 'Workflow not found: ' + input.workflow_id, valid: false }; const errors = []; const steps = wf.steps || []; const startSteps = steps.filter(s => s.type === 'start'); if (startSteps.length === 0) errors.push('No start step found'); const stepIds = new Set(steps.map(s => s.id)); for (const edge of (wf.edges || [])) { if (!stepIds.has(edge.source_step_id)) errors.push('Edge references non-existent source step'); if (!stepIds.has(edge.target_step_id)) errors.push('Edge references non-existent target step'); } return { valid: errors.length === 0, errors: errors, step_count: steps.length, edge_count: (wf.edges || []).length };",
					"description": "Validate a workflow's structure and identify potential issues",
					"input_schema": {
						"type": "object",
						"required": ["workflow_id"],
						"properties": {
							"workflow_id": {"type": "string", "description": "The workflow's UUID to validate"}
						}
					}
				}`),
			},

			// Semantic Block Search (Vector Search)
			{
				TempID:           "search_blocks",
				Name:             "search_blocks",
				Type:             "function",
				PositionX:        580,
				PositionY:        540,
				BlockGroupTempID: "copilot_agent_group",
				Config: json.RawMessage(`{
					"code": "if (!input.query) return { error: 'query is required' }; if (!ctx.embedding || !ctx.vector) return { fallback: true, message: 'Vector search not available, use list_blocks instead', blocks: ctx.blocks.list().filter(b => { const q = input.query.toLowerCase(); return (b.name && b.name.toLowerCase().includes(q)) || (b.description && b.description.toLowerCase().includes(q)) || (b.slug && b.slug.toLowerCase().includes(q)) || (input.category && b.category === input.category); }).slice(0, input.limit || 5).map(b => ({ slug: b.slug, name: b.name, category: b.category, description: b.description })) }; try { const embedding = ctx.embedding.embed('openai', 'text-embedding-3-small', [input.query]); if (!embedding || !embedding.vectors || !embedding.vectors[0]) return { error: 'Failed to generate embedding' }; const filter = input.category ? { category: { '$eq': input.category } } : null; const results = ctx.vector.query('block-embeddings', embedding.vectors[0], { topK: input.limit || 5, filter: filter }); if (!results || !results.matches) return { error: 'Vector search returned no results' }; return { blocks: results.matches.map(m => ({ slug: m.metadata.slug, name: m.metadata.name, category: m.metadata.category, description: m.metadata.description, score: m.score })), query: input.query }; } catch(e) { return { error: 'Search failed: ' + e.message, query: input.query }; }",
					"description": "Search for relevant blocks using semantic search. Use this when you're unsure which block to use or when the user describes functionality in natural language. Falls back to text matching if vector search is unavailable.",
					"input_schema": {
						"type": "object",
						"required": ["query"],
						"properties": {
							"query": {"type": "string", "description": "Natural language description of what you want the block to do (e.g., 'send a message to Slack', 'call an external API')"},
							"category": {"type": "string", "enum": ["trigger", "integration", "ai", "control", "data", "utility"], "description": "Optional category to filter results"},
							"limit": {"type": "integer", "description": "Maximum number of results to return (default: 5, max: 10)"}
						}
					}
				}`),
			},

			// Web Search Tools (for external API documentation lookup)
			{
				TempID:           "web_search",
				Name:             "web_search",
				Type:             "function",
				PositionX:        620,
				PositionY:        540,
				BlockGroupTempID: "copilot_agent_group",
				Config: json.RawMessage(`{
					"code": "if (!input.query) return { error: 'query is required' }; if (!ctx.search || !ctx.search.isConfigured()) return { error: 'Web search is not configured (TAVILY_API_KEY not set). Use fetch_url with known documentation URLs instead.' }; try { const results = ctx.search.search(input.query, input.num_results || 5); return { results: results }; } catch(e) { return { error: 'Search failed: ' + e.message }; }",
					"description": "Search the web using Tavily API. Use this to find official API documentation URLs for services that don't have preset blocks.",
					"input_schema": {
						"type": "object",
						"required": ["query"],
						"properties": {
							"query": {"type": "string", "description": "Search query (e.g., 'Stripe API documentation', 'Twilio REST API reference')"},
							"num_results": {"type": "integer", "description": "Number of results to return (1-10, default: 5)"}
						}
					}
				}`),
			},
			{
				TempID:           "fetch_url",
				Name:             "fetch_url",
				Type:             "function",
				PositionX:        740,
				PositionY:        540,
				BlockGroupTempID: "copilot_agent_group",
				Config: json.RawMessage(`{
					"code": "if (!input.url) return { error: 'url is required' }; try { const response = ctx.http.get(input.url, { headers: { 'User-Agent': 'CopilotBot/1.0 (Workflow Builder)', 'Accept': 'text/html,application/json,text/plain' } }); if (response.status !== 200) return { error: 'Failed to fetch: HTTP ' + response.status, url: input.url, status: response.status }; let content = response.data; if (typeof content === 'object') content = JSON.stringify(content, null, 2); if (content.length > 50000) content = content.substring(0, 50000) + '\\n\\n...(truncated, content exceeded 50KB limit)'; return { url: input.url, content: content, status: response.status }; } catch(e) { return { error: 'Fetch failed: ' + e.message, url: input.url }; }",
					"description": "Fetch content from a URL. Use this to retrieve API documentation from official websites. Useful for configuring HTTP blocks with external APIs.",
					"input_schema": {
						"type": "object",
						"required": ["url"],
						"properties": {
							"url": {"type": "string", "description": "The URL to fetch (e.g., https://docs.stripe.com/api, https://api.slack.com/methods)"}
						}
					}
				}`),
			},

			// ============================
			// E2E Workflow Management Tools
			// ============================

			// Get workflow status for progress tracking
			{
				TempID:           "get_workflow_status",
				Name:             "get_workflow_status",
				Type:             "function",
				PositionX:        300,
				PositionY:        640,
				BlockGroupTempID: "copilot_agent_group",
				Config: json.RawMessage(`{
					"code": "const wfId = ctx.targetProjectId; if (!wfId) return { error: 'No target project' }; const wf = ctx.workflows.get(wfId); if (!wf) return { error: 'Workflow not found' }; const steps = wf.steps || []; const startSteps = steps.filter(s => s.type === 'start'); const hasStart = startSteps.length > 0; const startStep = startSteps[0]; let triggerType = null; let triggerConfigured = false; if (startStep && startStep.trigger_type) { triggerType = startStep.trigger_type; triggerConfigured = startStep.trigger_type !== 'manual'; } const integrationSteps = steps.filter(s => ['slack', 'discord', 'github', 'notion', 'google-sheets', 'email'].includes(s.type)); const requiredCredentials = integrationSteps.map(s => ({ stepId: s.id, stepName: s.name, service: s.type, isConfigured: !!s.credential_id })); const unconfiguredCreds = requiredCredentials.filter(c => !c.isConfigured); let phase = 'creation'; if (steps.length > 1) phase = 'configuration'; if (triggerConfigured) phase = 'setup'; if (unconfiguredCreds.length === 0 && triggerConfigured) phase = 'validation'; if (wf.status === 'published') phase = 'deploy'; return { workflowId: wfId, name: wf.name, status: wf.status, currentPhase: phase, stepCount: steps.length, hasTrigger: hasStart, triggerType: triggerType, triggerConfigured: triggerConfigured, requiredCredentials: requiredCredentials, unconfiguredCredentialsCount: unconfiguredCreds.length, isPublished: wf.status === 'published', canPublish: steps.length > 0 && hasStart };",
					"description": "Get current workflow status including phase, trigger configuration, and required credentials. Use this to track progress and determine next steps.",
					"input_schema": {
						"type": "object",
						"properties": {}
					}
				}`),
			},

			// Configure trigger
			{
				TempID:           "configure_trigger",
				Name:             "configure_trigger",
				Type:             "function",
				PositionX:        460,
				PositionY:        640,
				BlockGroupTempID: "copilot_agent_group",
				Config: json.RawMessage(`{
					"code": "const wfId = ctx.targetProjectId; if (!wfId) return { error: 'No target project' }; if (!input.trigger_type) return { error: 'trigger_type is required (schedule, webhook, manual, slack_event)' }; const wf = ctx.workflows.get(wfId); if (!wf) return { error: 'Workflow not found' }; const steps = wf.steps || []; const startStep = steps.find(s => s.type === 'start'); if (!startStep) return { error: 'No start step found. Create a workflow structure first.' }; const config = {}; if (input.trigger_type === 'schedule') { config.schedule = input.schedule || '0 9 * * *'; config.timezone = input.timezone || 'Asia/Tokyo'; } else if (input.trigger_type === 'webhook') { config.method = input.method || 'POST'; } const updated = ctx.steps.update(startStep.id, { trigger_type: input.trigger_type, trigger_config: config }); return { success: true, stepId: startStep.id, triggerType: input.trigger_type, config: config };",
					"description": "Configure the workflow trigger (schedule, webhook, manual, or slack_event). Updates the start step's trigger configuration.",
					"input_schema": {
						"type": "object",
						"required": ["trigger_type"],
						"properties": {
							"trigger_type": {"type": "string", "enum": ["schedule", "webhook", "manual", "slack_event"], "description": "Type of trigger"},
							"schedule": {"type": "string", "description": "Cron expression for schedule trigger (e.g., '0 9 * * *' for daily 9am)"},
							"timezone": {"type": "string", "description": "Timezone for schedule (default: Asia/Tokyo)"},
							"method": {"type": "string", "description": "HTTP method for webhook trigger (default: POST)"}
						}
					}
				}`),
			},

			// List required credentials
			{
				TempID:           "list_required_credentials",
				Name:             "list_required_credentials",
				Type:             "function",
				PositionX:        620,
				PositionY:        640,
				BlockGroupTempID: "copilot_agent_group",
				Config: json.RawMessage(`{
					"code": "const wfId = ctx.targetProjectId; if (!wfId) return { error: 'No target project' }; const wf = ctx.workflows.get(wfId); if (!wf) return { error: 'Workflow not found' }; const steps = wf.steps || []; const integrationTypes = ['slack', 'discord', 'github', 'notion', 'google-sheets', 'email', 'openai', 'anthropic']; const integrationSteps = steps.filter(s => integrationTypes.includes(s.type) || (s.config && s.config.requires_credential)); const credentials = integrationSteps.map(s => ({ stepId: s.id, stepName: s.name, service: s.type, serviceName: s.type.charAt(0).toUpperCase() + s.type.slice(1), isConfigured: !!s.credential_id, credentialId: s.credential_id || null })); const configured = credentials.filter(c => c.isConfigured); const unconfigured = credentials.filter(c => !c.isConfigured); return { total: credentials.length, configured: configured.length, unconfigured: unconfigured.length, credentials: credentials };",
					"description": "List all credentials required by the workflow and their configuration status.",
					"input_schema": {
						"type": "object",
						"properties": {}
					}
				}`),
			},

			// Link credential to step
			{
				TempID:           "link_credential",
				Name:             "link_credential",
				Type:             "function",
				PositionX:        740,
				PositionY:        640,
				BlockGroupTempID: "copilot_agent_group",
				Config: json.RawMessage(`{
					"code": "if (!input.step_id) return { error: 'step_id is required' }; if (!input.credential_id) return { error: 'credential_id is required' }; const updated = ctx.steps.update(input.step_id, { credential_id: input.credential_id }); if (!updated || updated.error) return { error: 'Failed to link credential: ' + (updated && updated.error ? updated.error : 'Unknown error') }; return { success: true, stepId: input.step_id, credentialId: input.credential_id };",
					"description": "Link a credential to a step that requires authentication.",
					"input_schema": {
						"type": "object",
						"required": ["step_id", "credential_id"],
						"properties": {
							"step_id": {"type": "string", "description": "The step's UUID"},
							"credential_id": {"type": "string", "description": "The credential's UUID to link"}
						}
					}
				}`),
			},

			// Test workflow
			{
				TempID:           "test_workflow",
				Name:             "test_workflow",
				Type:             "function",
				PositionX:        300,
				PositionY:        740,
				BlockGroupTempID: "copilot_agent_group",
				Config: json.RawMessage(`{
					"code": "const wfId = ctx.targetProjectId; if (!wfId) return { error: 'No target project' }; const testInput = input.test_input || {}; try { const run = ctx.workflows.run(wfId, { input: testInput, mode: 'test' }); if (!run || run.error) return { error: 'Failed to start test run: ' + (run && run.error ? run.error : 'Unknown error') }; return { success: true, runId: run.id, status: 'started', message: 'Test run started. Check the execution tab for results.' }; } catch(e) { return { error: 'Test failed: ' + e.message }; }",
					"description": "Start a test run of the workflow to validate it works correctly.",
					"input_schema": {
						"type": "object",
						"properties": {
							"test_input": {"type": "object", "description": "Optional test input data for the workflow"}
						}
					}
				}`),
			},

			// Publish workflow
			{
				TempID:           "publish_workflow",
				Name:             "publish_workflow",
				Type:             "function",
				PositionX:        460,
				PositionY:        740,
				BlockGroupTempID: "copilot_agent_group",
				Config: json.RawMessage(`{
					"code": "const wfId = ctx.targetProjectId; if (!wfId) return { error: 'No target project' }; const wf = ctx.workflows.get(wfId); if (!wf) return { error: 'Workflow not found' }; const steps = wf.steps || []; if (steps.length === 0) return { error: 'Cannot publish empty workflow' }; const startSteps = steps.filter(s => s.type === 'start'); if (startSteps.length === 0) return { error: 'Cannot publish workflow without a start step' }; try { const result = ctx.workflows.publish(wfId); if (!result || result.error) return { error: 'Failed to publish: ' + (result && result.error ? result.error : 'Unknown error') }; return { success: true, workflowId: wfId, status: 'published', version: result.version || 1, message: 'Workflow published successfully!' }; } catch(e) { return { error: 'Publish failed: ' + e.message }; }",
					"description": "Publish the workflow to make it active and ready for production use.",
					"input_schema": {
						"type": "object",
						"properties": {}
					}
				}`),
			},

			// Workflow Readiness Check - validates all required fields are configured
			{
				TempID:           "check_workflow_readiness",
				Name:             "check_workflow_readiness",
				Type:             "function",
				PositionX:        620,
				PositionY:        740,
				BlockGroupTempID: "copilot_agent_group",
				Config:           json.RawMessage(checkWorkflowReadinessToolConfig()),
			},
		},
		Edges: []SystemEdgeDefinition{
			// Main flow: Start → Set Context → Agent Group
			{SourceTempID: "start", TargetTempID: "set_context"},
			// Note: TargetPort is empty to skip port validation (group ports are handled differently)
			{SourceTempID: "set_context", TargetGroupTempID: "copilot_agent_group"},
		},
		BlockGroups: []SystemBlockGroupDefinition{
			// ============================
			// Copilot Agent Group
			// Child steps become tools automatically
			// ============================
			{
				TempID:    "copilot_agent_group",
				Name:      "Copilot Agent",
				Type:      "agent",
				PositionX: 280,
				PositionY: 80,
				Width:     500,
				Height:    580,
				Config:    json.RawMessage(copilotAgentGroupConfig()),
			},
		},
	}
}

// copilotAgentGroupConfig returns the configuration for the Copilot Agent Group
// Note: Tools are not defined here - child steps automatically become tools
func copilotAgentGroupConfig() string {
	config := map[string]interface{}{
		"provider":       "anthropic",
		"model":          "claude-sonnet-4-20250514",
		"max_iterations": 15,
		"temperature":    0.7,
		"enable_memory":  true,
		"memory_window":  30,
		"tool_choice":    "auto",
		"system_prompt":  copilotSystemPrompt(),
	}
	jsonBytes, _ := json.Marshal(config)
	return string(jsonBytes)
}

func copilotSystemPrompt() string {
	return `You are Copilot, an AI assistant for the AI Orchestration platform.

## MOST IMPORTANT RULES

1. **NEVER introduce yourself** - Do NOT start with "Hello, I'm Copilot" or similar greetings
2. **ALWAYS use tools first** - Your first action should be calling a tool, not writing text
3. **Respond with text ONLY after completing tool calls** - Explain what you did, not what you will do

## When to Use Tools vs Text

USE TOOLS when the user:
- Asks to add, create, update, or delete anything → Use create_workflow_structure, update_step, delete_step, delete_edge
- Asks to see or list anything → Use list_blocks, list_workflows, get_workflow
- Asks about a specific block or feature → Use get_block_schema, search_documentation
- Asks to build a workflow → Use create_workflow_structure (single API call for steps and connections)

RESPOND WITH TEXT ONLY when:
- Explaining what you just did (after tool calls complete)
- The request is genuinely ambiguous and you need clarification
- The user asks a question that doesn't require tool use

## CRITICAL: Tool Selection Rules

**NEVER call list_workflows unless the user explicitly asks about workflows.**

**IMPORTANT: When adding blocks, ALWAYS use create_workflow_structure. It works for both single and multiple steps.**

**NOTE: All tools automatically operate on the current workflow. You do NOT need to specify project_id.**

### Preset Blocks (use with create_workflow_structure)
These services have preset blocks - use type directly:
- **Trigger Blocks**: schedule-trigger, manual-trigger, webhook-trigger (REQUIRED as first step in new workflows)
- **Integrations**: discord, slack, http, email, notion, github, google-sheets
- **AI/LLM**: llm, llm-chat, llm-structured, agent
- **Control Flow**: condition, switch, loop, map, filter
- **Data Processing**: function, set-variables, template, json-path
- **Utility**: log, delay

### External APIs (MUST use web_search BEFORE creating blocks)
**CRITICAL: Before creating ANY block for a service NOT in the preset list, you MUST:**

**STOP** - Check if the service name (freee, Stripe, Twilio, etc.) is in the preset list above.
**If NOT in the list:**
1. **FIRST** call **web_search** to find the API documentation
2. **THEN** call **fetch_url** to read the documentation
3. **FINALLY** create an **http** block with the correct configuration

**IMPORTANT: DO NOT create a block with type="{service_name}" if it's not a preset.**
For example: type="freee" will FAIL because freee is not a preset block.
You must use type="http" and configure it based on the API documentation.

**Services that ALWAYS require web_search first:**
- freee, Stripe, Twilio, Airtable, Salesforce, HubSpot, Shopify, Zendesk
- Any cloud service API (AWS, GCP, Azure services)
- Any service not explicitly listed in "Preset Blocks" above

**Correct flow for "freeeで仕訳を取得":**
1. **web_search("freee 会計API 仕訳 ドキュメント")** → Find developer.freee.co.jp
2. **fetch_url("https://developer.freee.co.jp/docs/accounting/reference")** → Read API spec
3. **create_workflow_structure** with single http step configured for freee API

When user asks to ADD/CREATE a block (e.g., "Discordブロックを追加して"):
→ IMMEDIATELY call create_workflow_structure with:
  - steps: [{"temp_id": "discord", "name": "Discord通知", "type": "discord", "position": {"x": 300, "y": 200}}]
  - connections: [] (or connect to existing step using its UUID)
→ Do NOT call list_blocks first - you already know common block slugs

When user asks to SEE/LIST blocks:
→ Call list_blocks (NOT list_workflows)

When user explicitly asks about workflows:
→ ONLY THEN call list_workflows

## Building Workflows with create_workflow_structure

**ALWAYS use create_workflow_structure when creating multiple steps and connections.**

This is the PREFERRED tool for building workflows because:
- Single API call (faster, fewer errors)
- Uses temp_id pattern for easy step referencing
- Automatic port resolution for condition/switch blocks
- Transactional: all-or-nothing (no partial failures)
- Automatic project_id (no need to specify)

### IMPORTANT: Trigger Block Selection

**Every new workflow MUST start with a trigger block.** Choose based on execution pattern:
- **schedule-trigger**: For scheduled/periodic execution (e.g., "毎日", "毎週", "every hour")
- **manual-trigger**: For user-initiated execution (e.g., "手動で実行", "ボタンで実行")
- **webhook-trigger**: For external API triggers (e.g., "Webhookで", "外部から呼び出し")

**NEVER use type="start"** - always use one of the trigger blocks above.

### Example Usage (ALWAYS include config!)

` + "`" + `` + "`" + `` + "`" + `json
{
  "steps": [
    {
      "temp_id": "trigger",
      "name": "スケジュール実行",
      "type": "schedule-trigger",
      "position": {"x": 100, "y": 200},
      "config": {"schedule": "0 9 * * *", "timezone": "Asia/Tokyo"}
    },
    {
      "temp_id": "http_step",
      "name": "API呼び出し",
      "type": "http",
      "position": {"x": 300, "y": 200},
      "config": {"url": "https://api.example.com/data", "method": "GET"}
    },
    {
      "temp_id": "condition_step",
      "name": "ステータス確認",
      "type": "condition",
      "position": {"x": 500, "y": 200},
      "config": {"expression": "{{status}} === 200"}
    },
    {
      "temp_id": "success_notify",
      "name": "成功通知",
      "type": "slack",
      "position": {"x": 700, "y": 100},
      "config": {"channel": "#notifications", "message": "成功: {{data}}"}
    },
    {
      "temp_id": "error_notify",
      "name": "エラー通知",
      "type": "discord",
      "position": {"x": 700, "y": 300},
      "config": {"channel_id": "123456789", "message": "エラー発生: {{error}}"}
    }
  ],
  "connections": [
    {"from": "trigger", "to": "http_step"},
    {"from": "http_step", "to": "condition_step"},
    {"from": "condition_step", "to": "success_notify", "from_port": "true"},
    {"from": "condition_step", "to": "error_notify", "from_port": "false"}
  ]
}
` + "`" + `` + "`" + `` + "`" + `

### Port Auto-Resolution

- **condition blocks**: Use "true" or "false" for from_port
- **switch blocks**: Use case values or "default" for from_port
- **other blocks**: Default from_port is "output"
- All blocks: Default to_port is "input"

## Common Block Slugs (use with create_workflow_structure)

- **Triggers**: schedule-trigger, manual-trigger, webhook-trigger (REQUIRED as first step)
- discord, slack, http, email, notion, github, google-sheets
- llm, llm-chat, llm-structured, agent
- condition, switch, loop, map, filter
- function, set-variables, template, log, delay

## Action Examples (ALWAYS include config!)

- "Discordブロックを追加して":
  1. get_block_schema("discord") → get required fields
  2. create_workflow_structure({steps: [{temp_id: "discord", name: "Discord通知", type: "discord", config: {channel_id: "...", message: "通知メッセージ"}}], connections: []})

- "LLMブロックを追加":
  1. get_block_schema("llm") → get required fields
  2. create_workflow_structure({steps: [{temp_id: "llm", name: "AI処理", type: "llm", config: {provider: "openai", model: "gpt-4o", user_prompt: "{{input}}を処理してください"}}], connections: []})

- "Slackに通知を送るブロックを追加":
  1. get_block_schema("slack") → get required fields
  2. create_workflow_structure({steps: [{temp_id: "slack", name: "Slack通知", type: "slack", config: {channel: "#general", message: "{{result}}"}}], connections: []})

- "ブロック一覧を見せて" → Call list_blocks

- "ワークフローを作成して" →
  1. Get schemas for each block type needed
  2. Call create_workflow_structure with steps (including config) and connections

## Your Capabilities

### Workflow Building (create mode)
When users want to create new workflows:
1. Immediately use tools to start building
2. If something is unclear, ask ONE specific question while showing progress
3. Use get_block_schema to get proper configuration
4. Use create_workflow_structure to create steps and connections (works for both single and multiple steps)

### Platform Guidance (explain mode)
When users ask about the platform:
1. Search documentation for relevant information
2. Explain block functionality and configuration
3. Provide examples and best practices
4. Guide users through complex features

### Workflow Enhancement (enhance mode)
When users want to improve existing workflows:
1. Analyze the current workflow structure
2. Identify potential improvements
3. Suggest optimizations (performance, reliability, cost)
4. Implement changes with user approval

## Common Blocks Quick Reference

Use this reference for fast access to common blocks. For detailed configuration, always call get_block_schema.

### AI/LLM Blocks
- **llm** (slug: llm): Text generation with AI models. Supports OpenAI, Anthropic, Google, etc.
- **llm-chat** (slug: llm-chat): Conversational AI with message history support.
- **llm-structured** (slug: llm-structured): AI with structured JSON output.

### Tool/Integration Blocks
- **http** (slug: http): Make HTTP requests to external APIs.
- **slack** (slug: slack): Send messages to Slack channels.
- **discord** (slug: discord): Send messages to Discord channels.
- **email** (slug: email): Send emails via SMTP.
- **notion** (slug: notion): Interact with Notion databases and pages.

### Control Flow Blocks
- **condition** (slug: condition): If/else branching based on expressions. Ports: true, false
- **switch** (slug: switch): Multi-way branching with multiple cases. Ports: case values, default
- **loop** (slug: loop): Iterate with count or condition.
- **map** (slug: map): Transform each item in an array.
- **filter** (slug: filter): Filter array items by condition.

### Data Processing Blocks
- **function** (slug: function): Execute custom JavaScript code.
- **set-variables** (slug: set-variables): Set and transform variables.
- **template** (slug: template): Generate text from templates.
- **json-path** (slug: json-path): Extract data using JSONPath expressions.

### Trigger Blocks (REQUIRED as first step in workflows)
- **schedule-trigger** (slug: schedule-trigger): Time-based trigger. Use for scheduled/periodic workflows.
- **manual-trigger** (slug: manual-trigger): Manual execution trigger. Use when user triggers manually.
- **webhook-trigger** (slug: webhook-trigger): HTTP webhook trigger. Use for external API callbacks.

### Utility Blocks
- **log** (slug: log): Log messages for debugging.
- **delay** (slug: delay): Wait for a specified time.

## CRITICAL: Config Generation Rules

**ALWAYS call get_block_schema BEFORE creating any step with config.**

When creating steps:
1. Call get_block_schema(slug) to get the config schema
2. Read required fields and defaults from the response
3. Generate config based on the schema
4. Include all required fields in the step config

### Two-Step Pattern (MANDATORY for configured steps)

**Step 1: Get schema**
→ get_block_schema("llm")
← Returns: {required_fields: ["provider", "model", "user_prompt"], resolved_config_defaults: {"temperature": 0.7, ...}}

**Step 2: Create step with config**
→ create_workflow_structure({
    steps: [{
      temp_id: "llm",
      name: "LLM処理",
      type: "llm",
      config: {
        provider: "openai",
        model: "gpt-4o",
        user_prompt: "{{input}}を分析してください",
        temperature: 0.7
      }
    }]
  })

### Config Examples by Block Type

**LLM Block:**
` + "```" + `json
{
  "temp_id": "llm",
  "name": "AI分析",
  "type": "llm",
  "config": {
    "provider": "openai",
    "model": "gpt-4o",
    "user_prompt": "{{data}}を分析してください",
    "system_prompt": "あなたは分析アシスタントです",
    "temperature": 0.7
  }
}
` + "```" + `

**HTTP Block:**
` + "```" + `json
{
  "temp_id": "http",
  "name": "API呼び出し",
  "type": "http",
  "config": {
    "url": "https://api.example.com/v1/data",
    "method": "POST",
    "headers": {"Content-Type": "application/json"},
    "body": "{\"query\": \"{{input}}\"}"
  }
}
` + "```" + `

**Slack Block:**
` + "```" + `json
{
  "temp_id": "slack",
  "name": "Slack通知",
  "type": "slack",
  "config": {
    "channel": "#notifications",
    "message": "処理完了: {{result}}"
  }
}
` + "```" + `

**Condition Block:**
` + "```" + `json
{
  "temp_id": "check",
  "name": "条件分岐",
  "type": "condition",
  "config": {
    "expression": "{{status}} === 'success'"
  }
}
` + "```" + `

## Available Tools

**Note: All create/modify tools automatically use the current workflow. No project_id needed.**

### Block Discovery & Search Tools
- **search_blocks**: Semantic search for blocks by natural language description (USE THIS when unsure which block to use!)
- **list_blocks**: Get all available blocks with basic info
- **get_block_schema**: Get detailed configuration schema for a specific block (CALL THIS FIRST before creating steps!)

### Workflow Tools
- **list_workflows**: List user's workflows
- **get_workflow**: Get workflow details including steps and edges
- **update_step**: Update an existing step's config or position
- **delete_step**: Delete a step
- **delete_edge**: Remove a connection
- **create_workflow_structure**: Create steps and connections in one call (auto project_id) - works for single or multiple steps
- **check_workflow_readiness**: Check if all steps have required fields configured (CALL AFTER creating steps!)
- **search_documentation**: Search platform documentation
- **validate_workflow**: Validate workflow structure

### External Documentation Tools
- **web_search**: Search the web for API documentation (requires TAVILY_API_KEY)
- **fetch_url**: Fetch content from a URL to read API documentation

## Recommended Flow for Unknown Block Types

When the user asks for something and you're not sure which block to use:

1. **search_blocks** with natural language query
   → search_blocks("send notification to Slack channel")
   ← Returns: [{slug: "slack", name: "Slack", score: 0.95}, ...]

2. **get_block_schema** for the best match
   → get_block_schema("slack")
   ← Returns: {required_fields: ["channel", "message"], config_schema: {...}}

3. **create_workflow_structure** with full config
   → create_workflow_structure({steps: [{type: "slack", config: {...}}], ...})

4. **check_workflow_readiness** to verify configuration
   → check_workflow_readiness()
   ← If issues found, fix with update_step, then re-check

This four-step pattern ensures you always use the right block with correct configuration.
   → create_workflow_structure({steps: [{type: "slack", config: {...}}], ...})

This three-step pattern ensures you always use the right block with correct configuration.

## Fetching External API Documentation

When the user wants to integrate with a service that doesn't have a preset block (e.g., Stripe, Twilio, custom APIs):

1. **Search for documentation** using web_search:
   - Query like "{service} API documentation" or "{service} REST API reference"
   - Review the search results to find official documentation URLs

2. **Fetch the documentation** using fetch_url:
   - Retrieve content from the found URL
   - If web_search is unavailable, try common URL patterns directly:
     - https://docs.{service}.com/api
     - https://developer.{service}.com/
     - https://api.{service}.com/docs

3. **Create HTTP block configuration**:
   - Parse the documentation to understand endpoints, authentication, parameters
   - Create an HTTP block with the correct URL, method, headers, and body

Example: "Stripe API で決済を作成するHTTPブロックを追加"
→ web_search("Stripe API create payment documentation")
→ fetch_url("https://docs.stripe.com/api/charges/create")
→ create_workflow_structure with HTTP block configured for Stripe Charges API

## CRITICAL: Template Variables & Data Flow

**Understanding how data flows between steps is ESSENTIAL for creating working workflows.**

### Data Flow Principle
- Each step receives **input** from the previous step's **output** (via edges)
- The previous step's output becomes the current step's input **directly**
- NO step name prefixes are used - reference fields directly from input

### Template Variable Syntax
- ` + "`" + `{{field}}` + "`" + ` - Reference a field from the input
- ` + "`" + `{{$.field}}` + "`" + ` - Same as above (JSONPath style)
- ` + "`" + `{{nested.field}}` + "`" + ` - Access nested fields
- ` + "`" + `{{$org.var}}` + "`" + ` - Organization-level variable (from tenant settings)
- ` + "`" + `{{$project.var}}` + "`" + ` - Project-level variable

### Correct vs Incorrect Patterns

**WRONG (using step name prefix):**
` + "```" + `
LLM step prompt: "Analyze: {{webhook.issue.title}}"
Switch expression: "{{ai_analysis.team}}"
Slack message: "Result: {{llm_output.summary}}"
` + "```" + `

**CORRECT (direct field reference):**
` + "```" + `
LLM step prompt: "Analyze: {{issue.title}}"     // webhook payload has 'issue' object
Switch expression: "{{team}}"                    // LLM output has 'team' field
Slack message: "Result: {{summary}}"             // Previous step output has 'summary'
` + "```" + `

### Example: GitHub Webhook → LLM → Switch Flow

1. **Start block (Webhook trigger)**
   - Receives webhook payload: ` + "`" + `{"issue": {"title": "...", "body": "..."}, "repository": {...}}` + "`" + `
   - Outputs the payload as-is

2. **LLM Structured block**
   - Input: webhook payload (from Start block)
   - Config prompt: ` + "`" + `"Analyze this issue:\nTitle: {{issue.title}}\nBody: {{issue.body}}"` + "`" + `
   - Output (from schema): ` + "`" + `{"team": "frontend", "priority": "high", "summary": "..."}` + "`" + `

3. **Switch block**
   - Input: LLM output
   - Expression: ` + "`" + `{{team}}` + "`" + ` (NOT ` + "`" + `{{llm.team}}` + "`" + ` or ` + "`" + `{{ai_analysis.team}}` + "`" + `)
   - Cases: ` + "`" + `{"frontend": "frontend", "backend": "backend"}` + "`" + `

4. **Slack block (after switch)**
   - Input: passed through from LLM (via switch)
   - Message: ` + "`" + `"Priority: {{priority}}\nSummary: {{summary}}"` + "`" + `

### Block Output Patterns

| Block Type | Output Format |
|------------|---------------|
| **start** | Webhook payload or manual input as-is |
| **llm** | ` + "`" + `{"content": "generated text"}` + "`" + ` |
| **llm-structured** | The JSON object matching the schema |
| **http** | Response body (parsed JSON or raw text) |
| **function** | Whatever the JavaScript returns |
| **set-variables** | Merged variables object |
| **condition/switch** | Passes input through to the selected branch |

### Key Rules

1. **NEVER use step names in templates** - Use field names from the actual data
2. **Reference fields directly** - ` + "`" + `{{field}}` + "`" + ` not ` + "`" + `{{stepName.field}}` + "`" + `
3. **Know your data structure** - Understand what each block outputs
4. **Switch/Condition pass through** - They forward input to the selected branch unchanged

### Complete Example: LLM → Slack Workflow

This is a complete, working example for "LLMでテキスト分析してSlackに通知" request:

` + "```" + `json
{
  "steps": [
    {
      "temp_id": "trigger",
      "name": "手動実行",
      "type": "manual-trigger",
      "position": {"x": 100, "y": 200},
      "config": {}
    },
    {
      "temp_id": "analyze",
      "name": "LLM分析",
      "type": "llm",
      "position": {"x": 300, "y": 200},
      "config": {
        "provider": "openai",
        "model": "gpt-4o",
        "user_prompt": "以下のテキストを分析してください:\n{{text}}",
        "system_prompt": "あなたは優秀な分析アシスタントです。簡潔に要点をまとめてください。"
      }
    },
    {
      "temp_id": "notify",
      "name": "Slack通知",
      "type": "slack",
      "position": {"x": 500, "y": 200},
      "config": {
        "channel": "#notifications",
        "message": "分析結果:\n{{content}}"
      }
    }
  ],
  "connections": [
    {"from": "trigger", "to": "analyze"},
    {"from": "analyze", "to": "notify"}
  ]
}
` + "```" + `

**Key points:**
- ` + "`" + `{{text}}` + "`" + ` in LLM prompt references the input field from trigger
- ` + "`" + `{{content}}` + "`" + ` in Slack message references the LLM output field (llm block outputs ` + "`" + `{"content": "..."}` + "`" + `)
- Each config includes all required fields for the block type

## Auto-Fix Workflow (MANDATORY)

After creating workflow steps, you MUST:
1. Call **check_workflow_readiness** to verify all steps are properly configured
2. If issues are found, fix them automatically using **update_step**
3. Re-check until ` + "`" + `ready: true` + "`" + `

### Auto-Fix Example Flow

` + "```" + `
→ check_workflow_readiness()
← { ready: false, issues: [
    { step_id: "abc-123", step_name: "LLM分析", field: "user_prompt", message: "Required field is empty" }
  ]}

→ update_step({
    step_id: "abc-123",
    config: { user_prompt: "{{text}}を分析してください" }
  })

→ check_workflow_readiness()
← { ready: true, issues: [] }

→ Report to user: "ワークフローを作成しました。すぐに実行できます。"
` + "```" + `

### Field Value Generation Guidelines

| Field | Generation Strategy |
|-------|---------------------|
| user_prompt | Use ` + "`" + `{{field}}` + "`" + ` from input or describe the task |
| channel | Use generic ` + "`" + `#notifications` + "`" + ` or ask user |
| message | Include ` + "`" + `{{content}}` + "`" + ` or relevant output fields |
| url | Ask user for specific URL |
| expression | Generate based on expected data structure |

## Workflow Building Guidelines

1. ALWAYS call get_block_schema BEFORE creating steps to understand required fields
2. ALWAYS use create_workflow_structure for adding blocks (works for single or multiple steps)
3. ALWAYS include config in step definitions - block defaults are merged automatically but you should provide user-specific values
4. Create steps with meaningful, descriptive names
5. Position steps logically on the canvas (increment x by ~200 for horizontal flow)
6. For condition blocks, always specify from_port as "true" or "false"
7. Validate the workflow after major changes
8. **IMPORTANT: Reuse existing Start step** - When connecting to an existing step, use its UUID in connections.from
9. For external APIs without preset blocks, use web_search/fetch_url to find documentation

**Note on Config Defaults:**
- Block definition defaults are automatically merged with your provided config
- However, you MUST still call get_block_schema to understand what fields are required
- User-provided values in config always override defaults
- Always provide meaningful values for user-facing fields (prompts, messages, etc.)

## Guidelines

1. **Take action first** - When the user's request is clear, use tools immediately
2. Only ask for confirmation for destructive operations (delete) or ambiguous requests
3. Explain what you did after taking action
4. Use Japanese when the user writes in Japanese
5. Be concise - show results, not lengthy explanations
6. When workflow context is provided, use it to understand the current state
7. For unknown APIs, search and fetch documentation before configuring HTTP blocks
`
}

// createWorkflowStructureToolConfig returns the configuration for the create_workflow_structure tool
// This tool creates multiple steps and connections in a single API call with:
// - temp_id pattern for easy step referencing
// - Automatic port resolution based on block type
// - Transactional behavior (all-or-nothing)
// - Automatic project_id from ctx.targetProjectId (no need to specify)
func createWorkflowStructureToolConfig() string {
	return `{
		"code": "if (!ctx.targetProjectId) return { error: 'No target project - Copilot must be opened from a workflow' }; const isUUID = (s) => /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(s); const TRIGGER_TYPES = ['schedule-trigger', 'manual-trigger', 'webhook-trigger']; const projectId = ctx.targetProjectId; const tempIdToRealId = {}; const createdStepIds = []; const stepTypes = {}; let triggerStepId = null; for (const stepConfig of (input.steps || [])) { if (!stepConfig.temp_id || !stepConfig.name || !stepConfig.type) { for (const id of createdStepIds) { try { ctx.steps.delete(id); } catch(e) {} } return { error: 'Each step requires temp_id, name, and type' }; } const pos = stepConfig.position || {}; const step = ctx.steps.create({ project_id: projectId, name: stepConfig.name, type: stepConfig.type, config: stepConfig.config || {}, position_x: pos.x || 0, position_y: pos.y || 0, block_definition_id: stepConfig.block_definition_id }); if (!step || step.error) { for (const id of createdStepIds) { try { ctx.steps.delete(id); } catch(e) {} } return { error: 'Failed to create step: ' + stepConfig.name + (step && step.error ? ' - ' + step.error : '') }; } tempIdToRealId[stepConfig.temp_id] = step.id; createdStepIds.push(step.id); stepTypes[stepConfig.temp_id] = stepConfig.type; if (TRIGGER_TYPES.includes(stepConfig.type)) { triggerStepId = step.id; } } const createdEdgeIds = []; if (triggerStepId) { const wf = ctx.workflows.getWithStart(projectId); if (wf && wf.start_step_id) { const autoEdge = ctx.edges.create({ project_id: projectId, source_step_id: wf.start_step_id, target_step_id: triggerStepId, source_port: 'output', target_port: 'input' }); if (autoEdge && autoEdge.id) { createdEdgeIds.push(autoEdge.id); } } } for (const conn of (input.connections || [])) { let sourceId = tempIdToRealId[conn.from]; let targetId = tempIdToRealId[conn.to]; if (!sourceId && isUUID(conn.from)) { sourceId = conn.from; } if (!targetId && isUUID(conn.to)) { targetId = conn.to; } if (!sourceId) { for (const id of createdEdgeIds) { try { ctx.edges.delete(id); } catch(e) {} } for (const id of createdStepIds) { try { ctx.steps.delete(id); } catch(e) {} } return { error: 'Invalid step reference: ' + conn.from }; } if (!targetId) { for (const id of createdEdgeIds) { try { ctx.edges.delete(id); } catch(e) {} } for (const id of createdStepIds) { try { ctx.steps.delete(id); } catch(e) {} } return { error: 'Invalid step reference: ' + conn.to }; } let fromPort = conn.from_port; if (!fromPort) { const sourceType = stepTypes[conn.from]; if (sourceType === 'condition') { fromPort = 'true'; } else if (sourceType === 'switch') { fromPort = 'default'; } else { fromPort = 'output'; } } const edge = ctx.edges.create({ project_id: projectId, source_step_id: sourceId, target_step_id: targetId, source_port: fromPort, target_port: conn.to_port || 'input' }); if (!edge || edge.error) { for (const id of createdEdgeIds) { try { ctx.edges.delete(id); } catch(e) {} } for (const id of createdStepIds) { try { ctx.steps.delete(id); } catch(e) {} } return { error: 'Failed to create connection from ' + conn.from + ' to ' + conn.to + (edge && edge.error ? ' - ' + edge.error : '') }; } createdEdgeIds.push(edge.id); } return { success: true, steps_created: createdStepIds.length, edges_created: createdEdgeIds.length, step_id_mapping: tempIdToRealId };",
		"description": "Create multiple steps and connections in a single API call. Project ID is automatically set. Uses temp_id for new steps. Connections can reference temp_ids OR existing step UUIDs (e.g., to connect to an existing Start step). Transactional: all-or-nothing. Automatic port resolution: condition='true', switch='default', others='output'. IMPORTANT: Block defaults are automatically merged with provided config - but you should still call get_block_schema first to understand required fields and provide appropriate values.",
		"input_schema": {
			"type": "object",
			"required": ["steps"],
			"properties": {
				"steps": {
					"type": "array",
					"description": "Array of step configurations to create",
					"items": {
						"type": "object",
						"required": ["temp_id", "name", "type"],
						"properties": {
							"temp_id": {"type": "string", "description": "Temporary ID for referencing in connections (e.g., 'step_1', 'http_request')"},
							"name": {"type": "string", "description": "Step name"},
							"type": {"type": "string", "description": "Step type (llm, http, condition, etc.)"},
							"config": {"type": "object", "description": "Step configuration"},
							"position": {
								"type": "object",
								"description": "Position on canvas",
								"properties": {
									"x": {"type": "integer", "description": "X position"},
									"y": {"type": "integer", "description": "Y position"}
								}
							},
							"block_definition_id": {"type": "string", "description": "Block definition UUID (optional)"}
						}
					}
				},
				"connections": {
					"type": "array",
					"description": "Array of connections between steps using temp_ids",
					"items": {
						"type": "object",
						"required": ["from", "to"],
						"properties": {
							"from": {"type": "string", "description": "Source step temp_id"},
							"to": {"type": "string", "description": "Target step temp_id"},
							"from_port": {"type": "string", "description": "Source port (auto-resolved: condition='true', switch='default', others='output')"},
							"to_port": {"type": "string", "description": "Target port (default: 'input')"}
						}
					}
				}
			}
		}
	}`
}

// checkWorkflowReadinessToolConfig returns the configuration for the check_workflow_readiness tool
// This tool validates that all steps have their required fields configured
// and returns a list of issues that need to be fixed
func checkWorkflowReadinessToolConfig() string {
	return `{
		"code": "const wfId = ctx.targetProjectId; if (!wfId) return { error: 'No target project' }; const wf = ctx.workflows.get(wfId); if (!wf) return { error: 'Workflow not found' }; const issues = []; const steps = wf.steps || []; for (const step of steps) { if (step.type === 'start') continue; const schema = ctx.blocks.getWithSchema(step.type); if (!schema) continue; const required = schema.required_fields || []; const config = step.config || {}; for (const field of required) { const value = config[field]; if (value === undefined || value === null || value === '') { issues.push({ step_id: step.id, step_name: step.name, step_type: step.type, field: field, current_value: value, message: 'Required field is empty or missing' }); } } } return { ready: issues.length === 0, issues: issues, step_count: steps.length, suggestion: issues.length > 0 ? 'Use update_step to fix each issue. Generate appropriate values based on step context and the examples in the system prompt.' : null };",
		"description": "Check if workflow is ready for execution. Returns issues with step_id for fixing. IMPORTANT: After creating steps, ALWAYS call this to verify configuration. If issues are found, fix them using update_step before reporting success to the user.",
		"input_schema": {
			"type": "object",
			"properties": {}
		}
	}`
}

// Note: copilotTools() has been removed.
// In Agent Group architecture, child steps automatically become tools.
// Tool definitions are derived from:
// - Step name -> tool name
// - Step config.description -> tool description
// - Step config.input_schema -> tool parameters
