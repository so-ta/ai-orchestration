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
// - Start → Set Default Config → Classify Intent → Switch Intent → Set Intent Config → Set Context → Agent Group
// - Tool steps inside the Agent Group are automatically available as tools
// - The agent uses ReAct loop to call tools and generate responses
//
// DOGFOODING: This workflow is built using the same blocks available to all users.
// All configuration (LLM settings, thresholds, mappings) is defined in the workflow,
// NOT hardcoded in Go code. This demonstrates that sophisticated AI agents can be
// created entirely through the workflow builder.
//
// Phase 1: LLM Configuration Migration
// - Default LLM config defined in set_default_config step
// - Intent-specific configs defined via switch + set-variables steps
//
// Phase 2: Validation Configuration Migration
// - Confidence thresholds defined in set_default_config
// - Refinement settings defined in workflow variables
func CopilotWorkflow() *SystemWorkflowDefinition {
	return &SystemWorkflowDefinition{
		ID:          "a0000000-0000-0000-0000-000000000201",
		SystemSlug:  "copilot",
		Name:        "Copilot AI Assistant",
		Description: "AI assistant for workflow building and platform guidance",
		Version:     25,
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
			// Phase 1: Set Default Configuration
			// ============================
			// This step replaces hardcoded DefaultLLMConfig(), DefaultLLMRetryConfig(),
			// and DefaultRefinementConfig() in copilot_llm.go and copilot_validation.go
			{
				TempID:    "set_default_config",
				Name:      "Set Default Config",
				Type:      "set-variables",
				PositionX: 100,
				PositionY: 300,
				BlockSlug: "set-variables",
				Config: json.RawMessage(`{
					"variables": [
						{"name": "llm_model", "value": "gpt-4o-mini", "type": "string"},
						{"name": "llm_temperature", "value": 0.3, "type": "number"},
						{"name": "llm_max_tokens", "value": 2000, "type": "number"},
						{"name": "retry_max_retries", "value": 3, "type": "number"},
						{"name": "retry_initial_delay_ms", "value": 1000, "type": "number"},
						{"name": "retry_max_delay_ms", "value": 30000, "type": "number"},
						{"name": "retry_backoff_factor", "value": 2.0, "type": "number"},
						{"name": "confidence_threshold", "value": 0.7, "type": "number"},
						{"name": "max_refinement_retries", "value": 2, "type": "number"},
						{"name": "refinement_temperature", "value": 0.2, "type": "number"}
					],
					"merge_input": true
				}`),
			},

			// ============================
			// Intent Classification - LLM-based intent detection
			// ============================
			{
				TempID:    "classify_intent",
				Name:      "Classify Intent",
				Type:      "llm-structured",
				PositionX: 160,
				PositionY: 300,
				BlockSlug: "llm-structured",
				Config:    json.RawMessage(intentClassificationConfig()),
			},

			// ============================
			// Phase 1: Intent-based Config Switch
			// ============================
			// This step replaces hardcoded GetIntentLLMConfig() in copilot_llm.go
			// Intent to case mapping (internal, not exposed to users):
			//   case_1 = create, case_2 = debug, case_3 = explain,
			//   case_4 = enhance, case_5 = search, default = general
			{
				TempID:    "switch_intent",
				Name:      "Switch Intent",
				Type:      "switch",
				PositionX: 220,
				PositionY: 300,
				BlockSlug: "switch",
				Config: json.RawMessage(`{
					"mode": "rules",
					"cases": [
						{"name": "case_1", "expression": "$.intent == 'create'"},
						{"name": "case_2", "expression": "$.intent == 'debug'"},
						{"name": "case_3", "expression": "$.intent == 'explain'"},
						{"name": "case_4", "expression": "$.intent == 'enhance'"},
						{"name": "case_5", "expression": "$.intent == 'search'"},
						{"name": "default", "is_default": true}
					]
				}`),
			},

			// Intent-specific config steps (Phase 1 migration from GetIntentLLMConfig)
			// NOTE: Each config step re-injects base config values since Classify Intent doesn't merge input
			// Case 1: Create intent - higher temperature for creativity
			{
				TempID:    "set_config_create",
				Name:      "Config: Create",
				Type:      "set-variables",
				PositionX: 280,
				PositionY: 100,
				BlockSlug: "set-variables",
				Config: json.RawMessage(`{
					"variables": [
						{"name": "llm_model", "value": "gpt-4o-mini", "type": "string"},
						{"name": "llm_temperature", "value": 0.5, "type": "number"},
						{"name": "llm_max_tokens", "value": 3000, "type": "number"},
						{"name": "confidence_threshold", "value": 0.7, "type": "number"},
						{"name": "max_refinement_retries", "value": 2, "type": "number"},
						{"name": "refinement_temperature", "value": 0.2, "type": "number"},
						{"name": "retry_max_retries", "value": 3, "type": "number"},
						{"name": "retry_initial_delay_ms", "value": 1000, "type": "number"},
						{"name": "retry_max_delay_ms", "value": 30000, "type": "number"},
						{"name": "retry_backoff_factor", "value": 2, "type": "number"}
					],
					"merge_input": true
				}`),
			},
			// Case 2: Debug intent - low temperature for precision
			{
				TempID:    "set_config_debug",
				Name:      "Config: Debug",
				Type:      "set-variables",
				PositionX: 280,
				PositionY: 180,
				BlockSlug: "set-variables",
				Config: json.RawMessage(`{
					"variables": [
						{"name": "llm_model", "value": "gpt-4o-mini", "type": "string"},
						{"name": "llm_temperature", "value": 0.1, "type": "number"},
						{"name": "llm_max_tokens", "value": 2000, "type": "number"},
						{"name": "confidence_threshold", "value": 0.7, "type": "number"},
						{"name": "max_refinement_retries", "value": 2, "type": "number"},
						{"name": "refinement_temperature", "value": 0.2, "type": "number"},
						{"name": "retry_max_retries", "value": 3, "type": "number"},
						{"name": "retry_initial_delay_ms", "value": 1000, "type": "number"},
						{"name": "retry_max_delay_ms", "value": 30000, "type": "number"},
						{"name": "retry_backoff_factor", "value": 2, "type": "number"}
					],
					"merge_input": true
				}`),
			},
			// Case 3: Explain intent - balanced settings
			{
				TempID:    "set_config_explain",
				Name:      "Config: Explain",
				Type:      "set-variables",
				PositionX: 280,
				PositionY: 260,
				BlockSlug: "set-variables",
				Config: json.RawMessage(`{
					"variables": [
						{"name": "llm_model", "value": "gpt-4o-mini", "type": "string"},
						{"name": "llm_temperature", "value": 0.3, "type": "number"},
						{"name": "llm_max_tokens", "value": 2500, "type": "number"},
						{"name": "confidence_threshold", "value": 0.7, "type": "number"},
						{"name": "max_refinement_retries", "value": 2, "type": "number"},
						{"name": "refinement_temperature", "value": 0.2, "type": "number"},
						{"name": "retry_max_retries", "value": 3, "type": "number"},
						{"name": "retry_initial_delay_ms", "value": 1000, "type": "number"},
						{"name": "retry_max_delay_ms", "value": 30000, "type": "number"},
						{"name": "retry_backoff_factor", "value": 2, "type": "number"}
					],
					"merge_input": true
				}`),
			},
			// Case 4: Enhance intent - moderate creativity
			{
				TempID:    "set_config_enhance",
				Name:      "Config: Enhance",
				Type:      "set-variables",
				PositionX: 280,
				PositionY: 340,
				BlockSlug: "set-variables",
				Config: json.RawMessage(`{
					"variables": [
						{"name": "llm_model", "value": "gpt-4o-mini", "type": "string"},
						{"name": "llm_temperature", "value": 0.4, "type": "number"},
						{"name": "llm_max_tokens", "value": 2500, "type": "number"},
						{"name": "confidence_threshold", "value": 0.7, "type": "number"},
						{"name": "max_refinement_retries", "value": 2, "type": "number"},
						{"name": "refinement_temperature", "value": 0.2, "type": "number"},
						{"name": "retry_max_retries", "value": 3, "type": "number"},
						{"name": "retry_initial_delay_ms", "value": 1000, "type": "number"},
						{"name": "retry_max_delay_ms", "value": 30000, "type": "number"},
						{"name": "retry_backoff_factor", "value": 2, "type": "number"}
					],
					"merge_input": true
				}`),
			},
			// Case 5: Search intent - low temperature for accuracy
			{
				TempID:    "set_config_search",
				Name:      "Config: Search",
				Type:      "set-variables",
				PositionX: 280,
				PositionY: 420,
				BlockSlug: "set-variables",
				Config: json.RawMessage(`{
					"variables": [
						{"name": "llm_model", "value": "gpt-4o-mini", "type": "string"},
						{"name": "llm_temperature", "value": 0.2, "type": "number"},
						{"name": "llm_max_tokens", "value": 1500, "type": "number"},
						{"name": "confidence_threshold", "value": 0.7, "type": "number"},
						{"name": "max_refinement_retries", "value": 2, "type": "number"},
						{"name": "refinement_temperature", "value": 0.2, "type": "number"},
						{"name": "retry_max_retries", "value": 3, "type": "number"},
						{"name": "retry_initial_delay_ms", "value": 1000, "type": "number"},
						{"name": "retry_max_delay_ms", "value": 30000, "type": "number"},
						{"name": "retry_backoff_factor", "value": 2, "type": "number"}
					],
					"merge_input": true
				}`),
			},
			// Default: General intent - use default config values
			{
				TempID:    "set_config_general",
				Name:      "Config: General",
				Type:      "set-variables",
				PositionX: 280,
				PositionY: 500,
				BlockSlug: "set-variables",
				Config: json.RawMessage(`{
					"variables": [
						{"name": "llm_model", "value": "gpt-4o-mini", "type": "string"},
						{"name": "llm_temperature", "value": 0.3, "type": "number"},
						{"name": "llm_max_tokens", "value": 2000, "type": "number"},
						{"name": "confidence_threshold", "value": 0.7, "type": "number"},
						{"name": "max_refinement_retries", "value": 2, "type": "number"},
						{"name": "refinement_temperature", "value": 0.2, "type": "number"},
						{"name": "retry_max_retries", "value": 3, "type": "number"},
						{"name": "retry_initial_delay_ms", "value": 1000, "type": "number"},
						{"name": "retry_max_delay_ms", "value": 30000, "type": "number"},
						{"name": "retry_backoff_factor", "value": 2, "type": "number"}
					],
					"merge_input": true
				}`),
			},

			// ============================
			// Set Variables - Context injection with config
			// ============================
			{
				TempID:    "set_context",
				Name:      "Set Context",
				Type:      "set-variables",
				PositionX: 340,
				PositionY: 300,
				BlockSlug: "set-variables",
				Config: json.RawMessage(`{
					"variables": [
						{"name": "intent", "value": "{{intent}}", "type": "string"},
						{"name": "confidence", "value": "{{confidence}}", "type": "number"},
						{"name": "detected_blocks", "value": "{{detected_blocks}}", "type": "array"},
						{"name": "workflow_id", "value": "{{$.workflow_id}}", "type": "string"},
						{"name": "session_id", "value": "{{$.session_id}}", "type": "string"},
						{"name": "user_message", "value": "{{$.message}}", "type": "string"},
						{"name": "llm_config", "value": {"model": "{{llm_model}}", "temperature": "{{llm_temperature}}", "max_tokens": "{{llm_max_tokens}}"}, "type": "object"},
						{"name": "retry_config", "value": {"max_retries": "{{retry_max_retries}}", "initial_delay_ms": "{{retry_initial_delay_ms}}", "max_delay_ms": "{{retry_max_delay_ms}}", "backoff_factor": "{{retry_backoff_factor}}"}, "type": "object"},
						{"name": "validation_config", "value": {"confidence_threshold": "{{confidence_threshold}}", "max_refinement_retries": "{{max_refinement_retries}}", "refinement_temperature": "{{refinement_temperature}}"}, "type": "object"}
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

			// ============================
			// Phase 3: Auto-Fix Tools (Migration from copilot_autofix.go)
			// ============================

			// Block type mapping tool - replaces findSimilarBlockType() in copilot_autofix.go
			{
				TempID:           "fix_block_type",
				Name:             "fix_block_type",
				Type:             "function",
				PositionX:        740,
				PositionY:        140,
				BlockGroupTempID: "copilot_agent_group",
				Config:           json.RawMessage(fixBlockTypeToolConfig()),
			},

			// Auto-fix validation errors tool
			{
				TempID:           "auto_fix_errors",
				Name:             "auto_fix_errors",
				Type:             "function",
				PositionX:        740,
				PositionY:        240,
				BlockGroupTempID: "copilot_agent_group",
				Config:           json.RawMessage(autoFixErrorsToolConfig()),
			},

			// ============================
			// Phase 4: Security Check Tool (Migration from copilot_sanitizer.go)
			// ============================
			{
				TempID:           "check_security",
				Name:             "check_security",
				Type:             "function",
				PositionX:        740,
				PositionY:        340,
				BlockGroupTempID: "copilot_agent_group",
				Config:           json.RawMessage(checkSecurityToolConfig()),
			},

			// ============================
			// Phase 5: Few-shot Examples Tool (Migration from copilot_examples.go)
			// ============================
			{
				TempID:           "get_relevant_examples",
				Name:             "get_relevant_examples",
				Type:             "function",
				PositionX:        740,
				PositionY:        440,
				BlockGroupTempID: "copilot_agent_group",
				Config:           json.RawMessage(getRelevantExamplesToolConfig()),
			},
		},
		Edges: []SystemEdgeDefinition{
			// Main flow: Start → Set Default Config → Classify Intent → Switch Intent
			{SourceTempID: "start", TargetTempID: "set_default_config"},
			{SourceTempID: "set_default_config", TargetTempID: "classify_intent"},
			{SourceTempID: "classify_intent", TargetTempID: "switch_intent"},

			// Intent-specific branches (from switch to config steps)
			// Mapping: case_1=create, case_2=debug, case_3=explain, case_4=enhance, case_5=search, default=general
			{SourceTempID: "switch_intent", TargetTempID: "set_config_create", SourcePort: "case_1"},
			{SourceTempID: "switch_intent", TargetTempID: "set_config_debug", SourcePort: "case_2"},
			{SourceTempID: "switch_intent", TargetTempID: "set_config_explain", SourcePort: "case_3"},
			{SourceTempID: "switch_intent", TargetTempID: "set_config_enhance", SourcePort: "case_4"},
			{SourceTempID: "switch_intent", TargetTempID: "set_config_search", SourcePort: "case_5"},
			{SourceTempID: "switch_intent", TargetTempID: "set_config_general", SourcePort: "default"},

			// Config steps converge to set_context
			{SourceTempID: "set_config_create", TargetTempID: "set_context"},
			{SourceTempID: "set_config_debug", TargetTempID: "set_context"},
			{SourceTempID: "set_config_explain", TargetTempID: "set_context"},
			{SourceTempID: "set_config_enhance", TargetTempID: "set_context"},
			{SourceTempID: "set_config_search", TargetTempID: "set_context"},
			{SourceTempID: "set_config_general", TargetTempID: "set_context"},

			// Set Context → Agent Group
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
		"model":          "claude-3-5-haiku-20241022",
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

## Context Variables

The following context is available from intent classification:
- **{{intent}}**: Classified user intent (create, explain, enhance, search, debug, general)
- **{{confidence}}**: Classification confidence score (0.0-1.0)
- **{{detected_blocks}}**: Block types detected in the user's message

Use these to optimize your response strategy.

## MOST IMPORTANT RULES

1. **NEVER introduce yourself** - Do NOT start with "Hello, I'm Copilot" or similar greetings
2. **ALWAYS use tools first** - Your first action should be calling a tool, not writing text
3. **Respond with text ONLY after completing tool calls** - Explain what you did, not what you will do
4. **Use intent to guide your approach** - Adapt your tool selection and response based on the classified intent

## Intent-Based Behavior

**When intent = "create"**:
→ Immediately use create_workflow_structure with detected_blocks
→ Call get_block_schema first to ensure proper config

**When intent = "explain"**:
→ Use search_documentation and get_block_schema
→ Provide clear explanations with examples

**When intent = "enhance"**:
→ First get_workflow to understand current state
→ Suggest specific improvements with rationale

**When intent = "search"**:
→ Use search_blocks or list_blocks
→ Filter results based on detected_blocks

**When intent = "debug"**:
→ Use validate_workflow and check_workflow_readiness
→ Identify and explain issues clearly

**When intent = "general"**:
→ Respond conversationally
→ Offer guidance on how you can help

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

### Workflow Building (intent = "create")
When the classified intent is "create":
1. Check detected_blocks for block types the user mentioned
2. Use get_block_schema to understand required configuration
3. Use create_workflow_structure to create steps with proper config
4. If unclear, ask ONE specific question while showing progress

### Platform Guidance (intent = "explain")
When the classified intent is "explain":
1. Search documentation for relevant information
2. Use get_block_schema to explain block configuration
3. Provide examples and best practices
4. Guide users through complex features

### Workflow Enhancement (intent = "enhance")
When the classified intent is "enhance":
1. Analyze the current workflow with get_workflow
2. Identify potential improvements based on context
3. Suggest optimizations (performance, reliability, cost)
4. Implement changes with user approval

### Block Search (intent = "search")
When the classified intent is "search":
1. Use search_blocks with semantic query for best results
2. Filter by category if detected_blocks suggests specific types
3. Present options with brief descriptions

### Debugging (intent = "debug")
When the classified intent is "debug":
1. Use validate_workflow to check structure
2. Use check_workflow_readiness for config issues
3. Explain problems clearly with suggested fixes

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

### Auto-Fix & Validation Tools
- **fix_block_type**: Fix invalid block types by finding similar valid ones (e.g., "gpt" → "llm", "trigger" → "manual-trigger")
- **auto_fix_errors**: Analyze validation errors and get suggested fixes
- **check_security**: Check code/text for security issues (dangerous commands, sensitive data patterns)
- **get_relevant_examples**: Get workflow examples based on intent and keywords (use for reference when creating workflows)

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
		"code": "if (!ctx.targetProjectId) return { error: 'No target project - Copilot must be opened from a workflow' }; const isUUID = (s) => /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(s); const TRIGGER_TYPES = ['schedule-trigger', 'manual-trigger', 'webhook-trigger']; const projectId = ctx.targetProjectId; const tempIdToRealId = {}; const createdStepIds = []; const stepTypes = {}; let triggerStepId = null; for (const stepConfig of (input.steps || [])) { if (!stepConfig.temp_id || !stepConfig.name || !stepConfig.type) { for (const id of createdStepIds) { try { ctx.steps.delete(id); } catch(e) {} } return { error: 'Each step requires temp_id, name, and type' }; } const pos = stepConfig.position || {}; const step = ctx.steps.create({ project_id: projectId, name: stepConfig.name, type: stepConfig.type, config: stepConfig.config || {}, position_x: pos.x || 0, position_y: pos.y || 0, block_definition_id: stepConfig.block_definition_id }); if (!step || step.error) { for (const id of createdStepIds) { try { ctx.steps.delete(id); } catch(e) {} } return { error: 'Failed to create step: ' + stepConfig.name + (step && step.error ? ' - ' + step.error : '') }; } tempIdToRealId[stepConfig.temp_id] = step.id; createdStepIds.push(step.id); stepTypes[stepConfig.temp_id] = stepConfig.type; if (TRIGGER_TYPES.includes(stepConfig.type)) { triggerStepId = step.id; } } const createdEdgeIds = []; if (triggerStepId) { const wf = ctx.workflows.getWithStart(projectId); if (wf && wf.start_step_id) { const autoEdge = ctx.edges.create({ project_id: projectId, source_step_id: wf.start_step_id, target_step_id: triggerStepId, source_port: 'output' }); if (autoEdge && autoEdge.id) { createdEdgeIds.push(autoEdge.id); } } } for (const conn of (input.connections || [])) { let sourceId = tempIdToRealId[conn.from]; let targetId = tempIdToRealId[conn.to]; if (!sourceId && isUUID(conn.from)) { sourceId = conn.from; } if (!targetId && isUUID(conn.to)) { targetId = conn.to; } if (!sourceId) { for (const id of createdEdgeIds) { try { ctx.edges.delete(id); } catch(e) {} } for (const id of createdStepIds) { try { ctx.steps.delete(id); } catch(e) {} } return { error: 'Invalid step reference: ' + conn.from }; } if (!targetId) { for (const id of createdEdgeIds) { try { ctx.edges.delete(id); } catch(e) {} } for (const id of createdStepIds) { try { ctx.steps.delete(id); } catch(e) {} } return { error: 'Invalid step reference: ' + conn.to }; } let fromPort = conn.from_port; if (!fromPort) { const sourceType = stepTypes[conn.from]; if (sourceType === 'condition') { fromPort = 'true'; } else if (sourceType === 'switch') { fromPort = 'default'; } else { fromPort = 'output'; } } const edge = ctx.edges.create({ project_id: projectId, source_step_id: sourceId, target_step_id: targetId, source_port: fromPort }); if (!edge || edge.error) { for (const id of createdEdgeIds) { try { ctx.edges.delete(id); } catch(e) {} } for (const id of createdStepIds) { try { ctx.steps.delete(id); } catch(e) {} } return { error: 'Failed to create connection from ' + conn.from + ' to ' + conn.to + (edge && edge.error ? ' - ' + edge.error : '') }; } createdEdgeIds.push(edge.id); } return { success: true, steps_created: createdStepIds.length, edges_created: createdEdgeIds.length, step_id_mapping: tempIdToRealId };",
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
							"from_port": {"type": "string", "description": "Source port (auto-resolved: condition='true', switch='default', others='output')"}
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

// intentClassificationConfig returns the configuration for the LLM-based intent classification step
// This replaces the hardcoded keyword-based IntentClassifier in Go code
func intentClassificationConfig() string {
	config := map[string]interface{}{
		"provider":       "anthropic",
		"model":          "claude-3-5-haiku-20241022",
		"preserve_input": true, // Keep original input (message, workflow_id, session_id, etc.) merged with output
		// NOTE: The schema instruction is included directly in system_prompt because
		// the llm-structured PreProcess chain may not correctly propagate config changes
		// when Go maps are converted to JavaScript objects.
		"system_prompt": `You are a JSON-only intent classifier. You MUST respond with ONLY a valid JSON object, nothing else. No explanations, no markdown, no text before or after the JSON.

## Intent Categories

- create: User wants to create new steps, workflows, or add blocks
- explain: User wants to understand something about the platform
- enhance: User wants to modify or improve existing workflow
- search: User is searching for blocks or documentation
- debug: User is debugging or troubleshooting
- general: General questions or casual conversation

## Block Types to Detect

Integration: slack, discord, notion, github, http, email, google-sheets
AI/LLM: llm, llm-chat, llm-structured, agent
Control: condition, switch, loop, map, filter
Data: function, set-variables, template, json-path
Trigger: schedule-trigger, manual-trigger, webhook-trigger

## Required Output Format
Respond with a JSON object containing these fields:
  - intent (string) (required): The classified intent category
  - confidence (number) (required): Confidence score from 0.0 to 1.0
  - detected_blocks (array) (required): Block slugs mentioned or implied in the message
  - reasoning (string): Brief explanation of the classification

Example:
{"intent":"create","confidence":0.95,"detected_blocks":["slack"],"reasoning":"User wants to create a Slack notification workflow"}`,
		"user_prompt": "Classify this message and respond with JSON only:\n\n{{$.message}}\n\nRespond with JSON:",
		"temperature": 0.1,
		"output_schema": map[string]interface{}{
			"type":     "object",
			"required": []string{"intent", "confidence", "detected_blocks"},
			"properties": map[string]interface{}{
				"intent": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"create", "explain", "enhance", "search", "debug", "general"},
					"description": "The classified intent category",
				},
				"confidence": map[string]interface{}{
					"type":        "number",
					"minimum":     0,
					"maximum":     1,
					"description": "Confidence score from 0.0 to 1.0",
				},
				"detected_blocks": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Block slugs mentioned or implied in the message",
				},
				"reasoning": map[string]interface{}{
					"type":        "string",
					"description": "Brief explanation of the classification",
				},
			},
		},
	}
	jsonBytes, _ := json.Marshal(config)
	return string(jsonBytes)
}

// Note: copilotTools() has been removed.
// In Agent Group architecture, child steps automatically become tools.
// Tool definitions are derived from:
// - Step name -> tool name
// - Step config.description -> tool description
// - Step config.input_schema -> tool parameters

// ============================================================================
// Phase 3: Auto-Fix Tool Configurations (Migration from copilot_autofix.go)
// ============================================================================

// fixBlockTypeToolConfig returns the configuration for the fix_block_type tool
// This migrates the hardcoded findSimilarBlockType() from copilot_autofix.go
func fixBlockTypeToolConfig() string {
	return `{
		"code": "if (!input.invalid_type) return { error: 'invalid_type is required' }; const invalidLower = input.invalid_type.toLowerCase(); const mappings = { 'trigger': 'manual-trigger', 'start': 'manual-trigger', 'begin': 'manual-trigger', 'ai': 'llm', 'gpt': 'llm', 'openai': 'llm', 'claude': 'llm', 'anthropic': 'llm', 'if': 'condition', 'branch': 'condition', 'conditional': 'condition', 'case': 'switch', 'delay': 'delay', 'sleep': 'delay', 'timer': 'delay', 'api': 'http', 'request': 'http', 'cron': 'schedule-trigger', 'schedule': 'schedule-trigger', 'scheduled': 'schedule-trigger', 'debug': 'log', 'print': 'log', 'console': 'log', 'parallel': 'map', 'foreach': 'map', 'loop': 'loop', 'merge': 'join', 'aggregate': 'aggregate', 'collect': 'join', 'human': 'human-in-loop', 'approval': 'human-in-loop', 'human_approval': 'human-in-loop' }; if (mappings[invalidLower]) { const mapped = mappings[invalidLower]; const block = ctx.blocks.getWithSchema(mapped); if (block) return { fixed: true, original_type: input.invalid_type, suggested_type: mapped, block_name: block.name, block_description: block.description }; } const blocks = ctx.blocks.list(); for (const block of blocks) { const slugLower = block.slug.toLowerCase(); if (slugLower.includes(invalidLower) || invalidLower.includes(slugLower)) { return { fixed: true, original_type: input.invalid_type, suggested_type: block.slug, block_name: block.name, block_description: block.description }; } } return { fixed: false, original_type: input.invalid_type, error: 'No similar block type found', available_types: blocks.slice(0, 20).map(b => b.slug) };",
		"description": "Find a valid block type similar to an invalid one. Uses a mapping table and fuzzy matching to suggest corrections. This replaces hardcoded mappings in copilot_autofix.go.",
		"input_schema": {
			"type": "object",
			"required": ["invalid_type"],
			"properties": {
				"invalid_type": {"type": "string", "description": "The invalid block type to fix (e.g., 'gpt', 'ai', 'trigger')"}
			}
		}
	}`
}

// autoFixErrorsToolConfig returns the configuration for the auto_fix_errors tool
// This migrates auto-fix logic from copilot_autofix.go
func autoFixErrorsToolConfig() string {
	return `{
		"code": "if (!input.errors || !Array.isArray(input.errors)) return { error: 'errors array is required' }; const fixes = []; for (const err of input.errors) { const fix = { error: err, fixed: false }; switch (err.category) { case 'missing_field': if (err.step_id && err.field) { const step = ctx.steps.get(err.step_id); if (step) { const block = ctx.blocks.getWithSchema(step.type); if (block && block.config_defaults && block.config_defaults[err.field] !== undefined) { fix.suggestion = { field: err.field, default_value: block.config_defaults[err.field] }; fix.fixed = true; } } } break; case 'invalid_port': if (err.step_type === 'condition') { fix.suggestion = { valid_ports: ['true', 'false'] }; fix.fixed = true; } else if (err.step_type === 'switch') { fix.suggestion = { valid_ports: ['case values or default'] }; fix.fixed = true; } else { fix.suggestion = { valid_ports: ['output'] }; fix.fixed = true; } break; case 'invalid_block': fix.suggestion = { action: 'Use fix_block_type tool to find a valid replacement' }; break; default: fix.suggestion = { action: 'Manual fix required' }; break; } fixes.push(fix); } return { total: input.errors.length, fixable: fixes.filter(f => f.fixed).length, fixes: fixes };",
		"description": "Analyze validation errors and suggest auto-fixes. Returns fixable errors with suggested corrections. Use update_step to apply fixes.",
		"input_schema": {
			"type": "object",
			"required": ["errors"],
			"properties": {
				"errors": {
					"type": "array",
					"description": "Array of validation errors from check_workflow_readiness",
					"items": {
						"type": "object",
						"properties": {
							"step_id": {"type": "string"},
							"step_name": {"type": "string"},
							"step_type": {"type": "string"},
							"field": {"type": "string"},
							"category": {"type": "string", "enum": ["missing_field", "invalid_port", "invalid_block", "disconnected", "structure"]}
						}
					}
				}
			}
		}
	}`
}

// ============================================================================
// Phase 4: Security Check Tool Configuration (Migration from copilot_sanitizer.go)
// ============================================================================

// checkSecurityToolConfig returns the configuration for the check_security tool
// This migrates the hardcoded dangerousPatterns and suspiciousPatterns from copilot_sanitizer.go
func checkSecurityToolConfig() string {
	return `{
		"code": "if (!input.code && !input.text) return { error: 'code or text is required' }; const content = input.code || input.text; const dangerousPatterns = ['rm -rf', 'rm -f', 'DROP TABLE', 'DELETE FROM', 'TRUNCATE', 'eval(', 'exec(', 'system(', 'os.system', 'subprocess.', 'child_process', '__import__', 'require(\"fs\")', 'fs.unlink', 'fs.rmdir', 'process.exit', 'curl |', 'wget |', 'base64 -d', '; bash', '| sh', 'ignore previous', 'ignore above', 'ignore all', 'disregard previous', 'system:', 'jailbreak', 'dan mode']; const suspiciousPatterns = ['password', 'api_key', 'api-key', 'secret', 'token', 'credential', 'private_key', 'private-key', 'access_key', 'access-key']; const contentLower = content.toLowerCase(); const dangerous = []; const suspicious = []; for (const pattern of dangerousPatterns) { if (contentLower.includes(pattern.toLowerCase())) { dangerous.push({ pattern: pattern, risk: 'high' }); } } for (const pattern of suspiciousPatterns) { if (contentLower.includes(pattern.toLowerCase())) { suspicious.push({ pattern: pattern, risk: 'medium' }); } } const riskLevel = dangerous.length > 0 ? 'high' : (suspicious.length > 0 ? 'medium' : 'low'); return { safe: dangerous.length === 0, risk_level: riskLevel, dangerous_patterns: dangerous, suspicious_patterns: suspicious, recommendation: dangerous.length > 0 ? 'Block contains potentially dangerous code. Review and remove before execution.' : (suspicious.length > 0 ? 'Block may contain sensitive data references. Ensure proper handling.' : 'No security issues detected.') };",
		"description": "Check code or text for security issues. Detects dangerous commands (rm -rf, DROP TABLE, etc.) and suspicious patterns (API keys, passwords). Use this before executing or storing user-provided code.",
		"input_schema": {
			"type": "object",
			"properties": {
				"code": {"type": "string", "description": "Code to check for security issues"},
				"text": {"type": "string", "description": "Text to check for security issues (alternative to code)"}
			}
		}
	}`
}

// ============================================================================
// Phase 5: Few-shot Examples Tool Configuration (Migration from copilot_examples.go)
// ============================================================================

// getRelevantExamplesToolConfig returns the configuration for the get_relevant_examples tool
// This migrates the hardcoded keywordToCategory and WorkflowExamples from copilot_examples.go
func getRelevantExamplesToolConfig() string {
	return `{
		"code": "const intent = input.intent || 'general'; const message = (input.message || '').toLowerCase(); const keywordToCategory = { 'loop': ['並列', '配列', 'ループ', '繰り返し', 'map', 'join', 'each', 'foreach', 'イテレート'], 'llm_chain': ['連鎖', 'チェーン', '多段', '順番', 'chain', 'パイプライン', '連続'], 'nested_condition': ['ネスト', '入れ子', '複数条件', '条件の中に条件', '優先度', '複合条件'], 'retry': ['リトライ', '再試行', '失敗時', 'エラー時', 'retry', '再実行', 'リカバリ'], 'data_pipeline': ['変換', 'フィルター', '集計', 'データ処理', 'パイプライン', 'etl', 'aggregate'], 'webhook_response': ['webhook', '外部連携', 'api呼び出し', 'リクエスト', 'レスポンス', 'コールバック'] }; const examples = { 'basic': { description: '基本的なワークフロー（トリガー → LLM → ログ）', steps: [{ temp_id: 'step_1', name: '開始', type: 'manual-trigger' }, { temp_id: 'step_2', name: 'AI処理', type: 'llm', config: { provider: 'openai', model: 'gpt-4o-mini', user_prompt: '{{$.message}}' } }, { temp_id: 'step_3', name: '結果をログ', type: 'log', config: { message: '処理完了: {{content}}' } }], edges: [{ from: 'step_1', to: 'step_2' }, { from: 'step_2', to: 'step_3' }] }, 'condition': { description: '条件分岐を含むワークフロー', steps: [{ temp_id: 'step_1', name: '開始', type: 'manual-trigger' }, { temp_id: 'step_2', name: '条件チェック', type: 'condition', config: { expression: '{{value}} > 100' } }, { temp_id: 'step_3', name: '高値処理', type: 'log' }, { temp_id: 'step_4', name: '通常処理', type: 'log' }], edges: [{ from: 'step_1', to: 'step_2' }, { from: 'step_2', to: 'step_3', from_port: 'true' }, { from: 'step_2', to: 'step_4', from_port: 'false' }] }, 'integration': { description: '外部連携ワークフロー（Slack通知）', steps: [{ temp_id: 'step_1', name: '開始', type: 'manual-trigger' }, { temp_id: 'step_2', name: 'メッセージ生成', type: 'llm' }, { temp_id: 'step_3', name: 'Slack通知', type: 'slack', config: { channel: '#general' } }], edges: [{ from: 'step_1', to: 'step_2' }, { from: 'step_2', to: 'step_3' }] }, 'loop': { description: '配列データの並列処理（map/join使用）', steps: [{ temp_id: 'step_1', name: 'トリガー', type: 'manual-trigger' }, { temp_id: 'step_2', name: '配列展開', type: 'map' }, { temp_id: 'step_3', name: '各要素処理', type: 'llm' }, { temp_id: 'step_4', name: '結果集約', type: 'join' }], edges: [{ from: 'step_1', to: 'step_2' }, { from: 'step_2', to: 'step_3' }, { from: 'step_3', to: 'step_4' }] }, 'retry': { description: 'リトライパターン（エラー時の再試行）', steps: [{ temp_id: 'step_1', name: 'トリガー', type: 'manual-trigger' }, { temp_id: 'step_2', name: 'API呼び出し', type: 'http' }, { temp_id: 'step_3', name: '成功チェック', type: 'condition' }, { temp_id: 'step_4', name: '成功処理', type: 'log' }, { temp_id: 'step_5', name: '待機', type: 'delay' }], edges: [{ from: 'step_1', to: 'step_2' }, { from: 'step_2', to: 'step_3' }, { from: 'step_3', to: 'step_4', from_port: 'true' }, { from: 'step_3', to: 'step_5', from_port: 'false' }] } }; const result = []; const intentCategories = { 'create': ['basic', 'integration'], 'enhance': ['condition', 'retry'], 'debug': ['condition', 'retry'], 'general': ['basic'] }; const baseCategories = intentCategories[intent] || ['basic']; for (const cat of baseCategories) { if (examples[cat]) result.push(examples[cat]); } for (const [category, keywords] of Object.entries(keywordToCategory)) { for (const keyword of keywords) { if (message.includes(keyword) && examples[category] && !result.find(e => e.description === examples[category].description)) { result.push(examples[category]); break; } } } return { examples: result.slice(0, 3), count: result.length, intent: intent, matched_keywords: [] };",
		"description": "Get relevant workflow examples based on intent and keywords in the user message. Returns up to 3 examples that match the context. Use these examples as templates when creating workflows.",
		"input_schema": {
			"type": "object",
			"properties": {
				"intent": {"type": "string", "enum": ["create", "explain", "enhance", "search", "debug", "general"], "description": "The classified user intent"},
				"message": {"type": "string", "description": "The user's message to extract keywords from"}
			}
		}
	}`
}
