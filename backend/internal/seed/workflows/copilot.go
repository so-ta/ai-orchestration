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
		Version:     3, // Bumped version for batch tools and enhanced context
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
				PositionX:        300,
				PositionY:        340,
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

			// Batch Tools - Efficient for creating multiple steps/edges at once
			{
				TempID:           "create_batch_steps",
				Name:             "create_batch_steps",
				Type:             "function",
				PositionX:        620,
				PositionY:        340,
				BlockGroupTempID: "copilot_agent_group",
				Config: json.RawMessage(`{
					"code": "if (!input.project_id || !input.steps || !Array.isArray(input.steps)) return { error: 'project_id and steps array are required' }; const results = []; for (const stepConfig of input.steps) { if (!stepConfig.name || !stepConfig.type) { return { error: 'Each step requires name and type', partial_results: results }; } const step = ctx.steps.create({ project_id: input.project_id, name: stepConfig.name, type: stepConfig.type, config: stepConfig.config || {}, position_x: stepConfig.position_x || 0, position_y: stepConfig.position_y || 0, block_definition_id: stepConfig.block_definition_id }); if (!step || step.error) { return { error: 'Failed to create step: ' + stepConfig.name, partial_results: results }; } results.push({ id: step.id, name: step.name, type: step.type }); } return { success: true, steps_created: results.length, steps: results };",
					"description": "Create multiple steps at once. More efficient than calling create_step multiple times. Returns the created steps with their IDs for edge creation.",
					"input_schema": {
						"type": "object",
						"required": ["project_id", "steps"],
						"properties": {
							"project_id": {"type": "string", "description": "The workflow/project ID"},
							"steps": {
								"type": "array",
								"description": "Array of step configurations to create",
								"items": {
									"type": "object",
									"required": ["name", "type"],
									"properties": {
										"name": {"type": "string", "description": "Step name"},
										"type": {"type": "string", "description": "Step type (llm, http, function, etc.)"},
										"config": {"type": "object", "description": "Step configuration"},
										"position_x": {"type": "integer", "description": "X position on canvas"},
										"position_y": {"type": "integer", "description": "Y position on canvas"},
										"block_definition_id": {"type": "string", "description": "Block definition UUID (optional)"}
									}
								}
							}
						}
					}
				}`),
			},
			{
				TempID:           "create_batch_edges",
				Name:             "create_batch_edges",
				Type:             "function",
				PositionX:        620,
				PositionY:        440,
				BlockGroupTempID: "copilot_agent_group",
				Config: json.RawMessage(`{
					"code": "if (!input.project_id || !input.edges || !Array.isArray(input.edges)) return { error: 'project_id and edges array are required' }; const results = []; for (const edgeConfig of input.edges) { if (!edgeConfig.source_step_id || !edgeConfig.target_step_id) { return { error: 'Each edge requires source_step_id and target_step_id', partial_results: results }; } const edge = ctx.edges.create({ project_id: input.project_id, source_step_id: edgeConfig.source_step_id, target_step_id: edgeConfig.target_step_id, source_port: edgeConfig.source_port || 'output', target_port: edgeConfig.target_port || 'input', condition: edgeConfig.condition }); if (!edge || edge.error) { return { error: 'Failed to create edge from ' + edgeConfig.source_step_id + ' to ' + edgeConfig.target_step_id, partial_results: results }; } results.push({ id: edge.id, source: edgeConfig.source_step_id, target: edgeConfig.target_step_id }); } return { success: true, edges_created: results.length, edges: results };",
					"description": "Create multiple edges (connections) at once. Use after creating multiple steps to connect them efficiently.",
					"input_schema": {
						"type": "object",
						"required": ["project_id", "edges"],
						"properties": {
							"project_id": {"type": "string", "description": "The workflow/project ID"},
							"edges": {
								"type": "array",
								"description": "Array of edge configurations to create",
								"items": {
									"type": "object",
									"required": ["source_step_id", "target_step_id"],
									"properties": {
										"source_step_id": {"type": "string", "description": "Source step ID"},
										"target_step_id": {"type": "string", "description": "Target step ID"},
										"source_port": {"type": "string", "description": "Source port name (default: 'output')"},
										"target_port": {"type": "string", "description": "Target port name (default: 'input')"},
										"condition": {"type": "string", "description": "Optional condition expression for conditional edges"}
									}
								}
							}
						}
					}
				}`),
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
- **condition** (slug: condition): If/else branching based on expressions.
- **switch** (slug: switch): Multi-way branching with multiple cases.
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

- **list_blocks**: Get all available blocks with basic info
- **get_block_schema**: Get detailed configuration schema for a specific block
- **search_blocks**: Search blocks by keyword
- **list_workflows**: List user's workflows
- **get_workflow**: Get workflow details including steps and edges
- **create_step**: Create a new step in a workflow
- **update_step**: Update an existing step's config or position
- **delete_step**: Delete a step
- **create_edge**: Connect two steps
- **delete_edge**: Remove a connection
- **create_batch_steps**: Create multiple steps at once (efficient for complex workflows)
- **create_batch_edges**: Create multiple edges at once (use after batch step creation)
- **search_documentation**: Search platform documentation
- **validate_workflow**: Validate workflow structure

## Workflow Building Guidelines

1. Always use list_blocks or get_block_schema to verify block configurations
2. Create steps with meaningful, descriptive names
3. Position steps logically on the canvas (increment position_x/position_y by ~150-200)
4. Connect steps with edges to define the execution flow
5. Validate the workflow after major changes
6. For multiple steps, prefer create_batch_steps for efficiency
7. Always connect the start step to the first processing step

## Guidelines

1. Always ask for confirmation before making changes
2. Explain your reasoning and suggestions
3. Provide step-by-step guidance for complex tasks
4. Use Japanese when the user writes in Japanese
5. Be concise but thorough in explanations
6. When workflow context is provided, use it to understand the current state
`
}

// Note: copilotTools() has been removed.
// In Agent Group architecture, child steps automatically become tools.
// Tool definitions are derived from:
// - Step name -> tool name
// - Step config.description -> tool description
// - Step config.input_schema -> tool parameters
