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
		Version:     2, // Bumped version for Agent Group migration
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
				PositionX:        40,
				PositionY:        40,
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
				PositionX:        160,
				PositionY:        40,
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
				PositionX:        280,
				PositionY:        40,
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
				PositionX:        40,
				PositionY:        120,
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
				PositionX:        160,
				PositionY:        120,
				BlockGroupTempID: "copilot_agent_group",
				Config: json.RawMessage(`{
					"code": "if (!input.workflow_id) return { error: 'workflow_id is required' }; const wf = ctx.workflows.get(input.workflow_id); if (!wf) return { error: 'Workflow not found: ' + input.workflow_id }; return wf;",
					"description": "Get detailed information about a specific workflow, including its steps and edges",
					"input_schema": {
						"type": "object",
						"required": ["workflow_id"],
						"properties": {
							"workflow_id": {"type": "string", "description": "The workflow's UUID"}
						}
					}
				}`),
			},

			// Step Tools
			{
				TempID:           "create_step",
				Name:             "create_step",
				Type:             "function",
				PositionX:        40,
				PositionY:        200,
				BlockGroupTempID: "copilot_agent_group",
				Config: json.RawMessage(`{
					"code": "if (!input.project_id || !input.name || !input.type) return { error: 'project_id, name, and type are required' }; const step = ctx.steps.create({ project_id: input.project_id, name: input.name, type: input.type, config: input.config || {}, position_x: input.position_x || 0, position_y: input.position_y || 0, block_definition_id: input.block_definition_id }); return step;",
					"description": "Create a new step in a workflow",
					"input_schema": {
						"type": "object",
						"required": ["project_id", "name", "type"],
						"properties": {
							"project_id": {"type": "string", "description": "The workflow/project ID where the step will be created"},
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
				PositionX:        160,
				PositionY:        200,
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
				PositionX:        280,
				PositionY:        200,
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
				PositionX:        40,
				PositionY:        280,
				BlockGroupTempID: "copilot_agent_group",
				Config: json.RawMessage(`{
					"code": "if (!input.project_id || !input.source_step_id || !input.target_step_id) return { error: 'project_id, source_step_id, and target_step_id are required' }; const edge = ctx.edges.create({ project_id: input.project_id, source_step_id: input.source_step_id, target_step_id: input.target_step_id, source_port: input.source_port || 'output', target_port: input.target_port || 'input', condition: input.condition }); return edge;",
					"description": "Create a connection between two steps",
					"input_schema": {
						"type": "object",
						"required": ["project_id", "source_step_id", "target_step_id"],
						"properties": {
							"project_id": {"type": "string", "description": "The workflow/project ID"},
							"source_step_id": {"type": "string", "description": "UUID of the source step"},
							"target_step_id": {"type": "string", "description": "UUID of the target step"},
							"source_port": {"type": "string", "description": "Output port name (default: 'output')"},
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
				PositionX:        160,
				PositionY:        280,
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

			// Documentation Search (using RAG if available)
			{
				TempID:           "search_documentation",
				Name:             "search_documentation",
				Type:             "function",
				PositionX:        40,
				PositionY:        360,
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
				PositionX:        160,
				PositionY:        360,
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
				Width:     450,
				Height:    450,
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
	return `You are Copilot, an AI assistant for the AI Orchestration platform. You help users build, understand, and improve their workflows.

## Your Capabilities

### Workflow Building (create mode)
When users want to create new workflows:
1. Ask clarifying questions about their use case
2. Suggest appropriate blocks from the available catalog
3. Create steps with proper configuration
4. Connect steps with edges
5. Validate the final workflow

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

## Available Tools

- list_blocks: Get all available blocks
- get_block_schema: Get detailed schema for a specific block
- search_blocks: Search blocks by keyword
- list_workflows: List user's workflows
- get_workflow: Get workflow details
- create_step: Create a new step in a workflow
- update_step: Update an existing step
- delete_step: Delete a step
- create_edge: Connect two steps
- delete_edge: Remove a connection
- search_documentation: Search platform documentation
- validate_workflow: Validate workflow structure

## Guidelines

1. Always ask for confirmation before making changes
2. Explain your reasoning and suggestions
3. Provide step-by-step guidance for complex tasks
4. Use Japanese when the user writes in Japanese
5. Be concise but thorough in explanations

## Block Categories

- ai: LLM, Agent, RAG blocks
- integration: External service integrations (Slack, Discord, GitHub, etc.)
- data: Data transformation and processing
- control: Flow control (conditions, loops, parallel execution)
- utility: Helper blocks (code, function, log, etc.)
- rag: Vector database and retrieval blocks
`
}

// Note: copilotTools() has been removed.
// In Agent Group architecture, child steps automatically become tools.
// Tool definitions are derived from:
// - Step name -> tool name
// - Step config.description -> tool description
// - Step config.input_schema -> tool parameters
