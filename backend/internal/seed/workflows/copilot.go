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
		Version:     14,
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
			{
				TempID:           "search_blocks",
				Name:             "search_blocks",
				Type:             "function",
				PositionX:        620,
				PositionY:        140,
				BlockGroupTempID: "copilot_agent_group",
				Config: json.RawMessage(`{
					"code": "const query = (input.query || '').toLowerCase(); const blocks = ctx.blocks.list(); const matched = blocks.filter(b => (b.name && b.name.toLowerCase().includes(query)) || (b.description && b.description.toLowerCase().includes(query)) || (b.slug && b.slug.toLowerCase().includes(query))); return { blocks: matched.map(b => ({ slug: b.slug, name: b.name, category: b.category, description: b.description })), count: matched.length };",
					"description": "Search for blocks by keyword in name, description, or slug",
					"input_schema": {
						"type": "object",
						"required": ["query"],
						"properties": {
							"query": {"type": "string", "description": "Search keyword"}
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
				TempID:           "create_step",
				Name:             "create_step",
				Type:             "function",
				PositionX:        300,
				PositionY:        340,
				BlockGroupTempID: "copilot_agent_group",
				Config: json.RawMessage(`{
					"code": "if (!ctx.targetProjectId) return { error: 'No target project - Copilot must be opened from a workflow' }; if (!input.name || !input.type) return { error: 'name and type are required' }; const step = ctx.steps.create({ project_id: ctx.targetProjectId, name: input.name, type: input.type, config: input.config || {}, position_x: input.position_x || 0, position_y: input.position_y || 0, block_definition_id: input.block_definition_id }); return step;",
					"description": "Create a new step in the current workflow (project_id is automatically set)",
					"input_schema": {
						"type": "object",
						"required": ["name", "type"],
						"properties": {
							"name": {"type": "string", "description": "Name of the step"},
							"type": {"type": "string", "description": "Step type (e.g., 'llm', 'http', 'function')"},
							"config": {"type": "object", "description": "Step configuration options"},
							"position_x": {"type": "integer", "description": "X position on canvas"},
							"position_y": {"type": "integer", "description": "Y position on canvas"},
							"block_definition_id": {"type": "string", "description": "UUID of the block definition to use"}
						}
					}
				}`),
			},
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
				TempID:           "create_edge",
				Name:             "create_edge",
				Type:             "function",
				PositionX:        300,
				PositionY:        440,
				BlockGroupTempID: "copilot_agent_group",
				Config: json.RawMessage(`{
					"code": "if (!ctx.targetProjectId) return { error: 'No target project - Copilot must be opened from a workflow' }; if (!input.source_step_id || !input.target_step_id) return { error: 'source_step_id and target_step_id are required' }; const edge = ctx.edges.create({ project_id: ctx.targetProjectId, source_step_id: input.source_step_id, target_step_id: input.target_step_id, source_port: input.source_port || 'output', target_port: input.target_port || 'input', condition: input.condition }); return edge;",
					"description": "Create a connection between two steps in the current workflow. IMPORTANT: For condition blocks, you MUST specify source_port as 'true' or 'false'. For switch blocks, specify the case name.",
					"input_schema": {
						"type": "object",
						"required": ["source_step_id", "target_step_id"],
						"properties": {
							"source_step_id": {"type": "string", "description": "UUID of the source step"},
							"target_step_id": {"type": "string", "description": "UUID of the target step"},
							"source_port": {"type": "string", "description": "Output port name. REQUIRED for condition blocks ('true' or 'false'). Default: 'output'"},
							"target_port": {"type": "string", "description": "Input port name (default: 'input')"},
							"condition": {"type": "string", "description": "Optional condition expression"}
						}
					}
				}`),
			},
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
- Asks to add, create, update, or delete anything → Use create_step, update_step, etc.
- Asks to see or list anything → Use list_blocks, list_workflows, get_workflow
- Asks about a specific block or feature → Use get_block_schema, search_documentation
- Asks to build a workflow → Use create_workflow_structure (single API call)

RESPOND WITH TEXT ONLY when:
- Explaining what you just did (after tool calls complete)
- The request is genuinely ambiguous and you need clarification
- The user asks a question that doesn't require tool use

## CRITICAL: Tool Selection Rules

**NEVER call list_workflows unless the user explicitly asks about workflows.**

**IMPORTANT: When adding a block, you MUST call create_step in the SAME turn. Do NOT wait for the next message.**

**NOTE: All tools automatically operate on the current workflow. You do NOT need to specify project_id.**

### Preset Blocks (use directly with create_step)
These services have preset blocks - use type directly:
- **Integrations**: discord, slack, http, email, notion, github, google-sheets
- **AI/LLM**: llm, llm-chat, llm-structured, agent
- **Control Flow**: condition, switch, loop, map, filter
- **Data Processing**: function, set-variables, template, json-path
- **Utility**: start, log, delay

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
3. **create_step(type="http", config={...})** with:
   - url: https://api.freee.co.jp/api/1/deals
   - method: GET
   - headers: Authorization Bearer token
   - Parameters from documentation

When user asks to ADD/CREATE a block (e.g., "Discordブロックを追加して"):
→ IMMEDIATELY call create_step with:
  - name: descriptive name (e.g., "Discord通知")
  - type: the block slug (e.g., "discord")
  - position_x: 300, position_y: 200 (or as specified)
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

### Example Usage

` + "`" + `` + "`" + `` + "`" + `json
{
  "steps": [
    {"temp_id": "http_step", "name": "HTTP Request", "type": "http", "position": {"x": 300, "y": 200}},
    {"temp_id": "condition_step", "name": "Check Status", "type": "condition", "position": {"x": 500, "y": 200}},
    {"temp_id": "success_notify", "name": "Success Notification", "type": "slack", "position": {"x": 700, "y": 100}},
    {"temp_id": "error_notify", "name": "Error Notification", "type": "discord", "position": {"x": 700, "y": 300}}
  ],
  "connections": [
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

## Common Block Slugs (use directly with create_step)

- discord, slack, http, email, notion, github, google-sheets
- llm, llm-chat, llm-structured, agent
- condition, switch, loop, map, filter
- function, set-variables, template, log, delay, start

## Action Examples

- "Discordブロックを追加して" → IMMEDIATELY call create_step(type="discord", name="Discord通知")
- "LLMブロックを追加" → IMMEDIATELY call create_step(type="llm", name="LLM")
- "ブロック一覧を見せて" → Call list_blocks
- "ワークフローを作成して" → Call create_workflow_structure with steps and connections

## Your Capabilities

### Workflow Building (create mode)
When users want to create new workflows:
1. Immediately use tools to start building
2. If something is unclear, ask ONE specific question while showing progress
3. Use get_block_schema to get proper configuration
4. Use create_workflow_structure to create multiple steps and connections in one call
5. For single step additions, use create_step

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

### Utility Blocks
- **start** (slug: start): Workflow entry point (required for every workflow).
- **log** (slug: log): Log messages for debugging.
- **delay** (slug: delay): Wait for a specified time.

## Available Tools

**Note: All create/modify tools automatically use the current workflow. No project_id needed.**

### Block & Workflow Tools
- **list_blocks**: Get all available blocks with basic info
- **get_block_schema**: Get detailed configuration schema for a specific block
- **search_blocks**: Search blocks by keyword
- **list_workflows**: List user's workflows
- **get_workflow**: Get workflow details including steps and edges
- **create_step**: Create a single step (auto project_id)
- **update_step**: Update an existing step's config or position
- **delete_step**: Delete a step
- **create_edge**: Connect two steps (auto project_id)
- **delete_edge**: Remove a connection
- **create_workflow_structure**: Create multiple steps and connections in one call (PREFERRED, auto project_id)
- **search_documentation**: Search platform documentation
- **validate_workflow**: Validate workflow structure

### External Documentation Tools
- **web_search**: Search the web for API documentation (requires GOOGLE_CUSTOM_SEARCH_API_KEY)
- **fetch_url**: Fetch content from a URL to read API documentation

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
→ create_step with HTTP block configured for Stripe Charges API

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

## Workflow Building Guidelines

1. Use create_workflow_structure for building multi-step workflows (PREFERRED)
2. Use create_step for adding a single block
3. Create steps with meaningful, descriptive names
4. Position steps logically on the canvas (increment x by ~200 for horizontal flow)
5. For condition blocks, always specify from_port as "true" or "false"
6. Validate the workflow after major changes
7. **IMPORTANT: Reuse existing Start step** - When the workflow already has a Start step, use get_workflow first to find its ID, then connect to it using create_edge. Do NOT create a new Start step.
8. For external APIs without preset blocks, use web_search/fetch_url to find documentation

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
		"code": "if (!ctx.targetProjectId) return { error: 'No target project - Copilot must be opened from a workflow' }; const isUUID = (s) => /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(s); const projectId = ctx.targetProjectId; const tempIdToRealId = {}; const createdStepIds = []; const stepTypes = {}; for (const stepConfig of (input.steps || [])) { if (!stepConfig.temp_id || !stepConfig.name || !stepConfig.type) { for (const id of createdStepIds) { try { ctx.steps.delete(id); } catch(e) {} } return { error: 'Each step requires temp_id, name, and type' }; } const pos = stepConfig.position || {}; const step = ctx.steps.create({ project_id: projectId, name: stepConfig.name, type: stepConfig.type, config: stepConfig.config || {}, position_x: pos.x || 0, position_y: pos.y || 0, block_definition_id: stepConfig.block_definition_id }); if (!step || step.error) { for (const id of createdStepIds) { try { ctx.steps.delete(id); } catch(e) {} } return { error: 'Failed to create step: ' + stepConfig.name + (step && step.error ? ' - ' + step.error : '') }; } tempIdToRealId[stepConfig.temp_id] = step.id; createdStepIds.push(step.id); stepTypes[stepConfig.temp_id] = stepConfig.type; } const createdEdgeIds = []; for (const conn of (input.connections || [])) { let sourceId = tempIdToRealId[conn.from]; let targetId = tempIdToRealId[conn.to]; if (!sourceId && isUUID(conn.from)) { sourceId = conn.from; } if (!targetId && isUUID(conn.to)) { targetId = conn.to; } if (!sourceId) { for (const id of createdEdgeIds) { try { ctx.edges.delete(id); } catch(e) {} } for (const id of createdStepIds) { try { ctx.steps.delete(id); } catch(e) {} } return { error: 'Invalid step reference: ' + conn.from }; } if (!targetId) { for (const id of createdEdgeIds) { try { ctx.edges.delete(id); } catch(e) {} } for (const id of createdStepIds) { try { ctx.steps.delete(id); } catch(e) {} } return { error: 'Invalid step reference: ' + conn.to }; } let fromPort = conn.from_port; if (!fromPort) { const sourceType = stepTypes[conn.from]; if (sourceType === 'condition') { fromPort = 'true'; } else if (sourceType === 'switch') { fromPort = 'default'; } else { fromPort = 'output'; } } const edge = ctx.edges.create({ project_id: projectId, source_step_id: sourceId, target_step_id: targetId, source_port: fromPort, target_port: conn.to_port || 'input' }); if (!edge || edge.error) { for (const id of createdEdgeIds) { try { ctx.edges.delete(id); } catch(e) {} } for (const id of createdStepIds) { try { ctx.steps.delete(id); } catch(e) {} } return { error: 'Failed to create connection from ' + conn.from + ' to ' + conn.to + (edge && edge.error ? ' - ' + edge.error : '') }; } createdEdgeIds.push(edge.id); } return { success: true, steps_created: createdStepIds.length, edges_created: createdEdgeIds.length, step_id_mapping: tempIdToRealId };",
		"description": "Create multiple steps and connections in a single API call. Project ID is automatically set. Uses temp_id for new steps. Connections can reference temp_ids OR existing step UUIDs (e.g., to connect to an existing Start step). Transactional: all-or-nothing. Automatic port resolution: condition='true', switch='default', others='output'.",
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

// Note: copilotTools() has been removed.
// In Agent Group architecture, child steps automatically become tools.
// Tool definitions are derived from:
// - Step name -> tool name
// - Step config.description -> tool description
// - Step config.input_schema -> tool parameters
