package workflows

import "encoding/json"

func (r *Registry) registerCopilotWorkflows() {
	r.register(CopilotWorkflow())
}

// CopilotWorkflow is the system workflow for the Copilot AI assistant.
// It uses the agent block with tools to help users:
// - Build and modify workflows
// - Understand platform features
// - Get help with block configuration
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
		Version:     1,
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
				PositionY: 200,
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
				PositionY: 200,
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
			// Copilot Agent
			// ============================
			{
				TempID:    "copilot_agent",
				Name:      "Copilot Agent",
				Type:      "agent",
				PositionX: 280,
				PositionY: 200,
				BlockSlug: "agent",
				Config:    json.RawMessage(copilotAgentConfig()),
			},

			// ============================
			// Tool Steps
			// These steps are called by the agent via ctx.workflow.executeStep()
			// ============================

			// Block Tools
			{
				TempID:    "list_blocks",
				Name:      "list_blocks",
				Type:      "function",
				PositionX: 480,
				PositionY: 40,
				Config: json.RawMessage(`{
					"code": "const blocks = ctx.blocks.list(); return { blocks: blocks.map(b => ({ slug: b.slug, name: b.name, category: b.category, description: b.description })) };"
				}`),
			},
			{
				TempID:    "get_block_schema",
				Name:      "get_block_schema",
				Type:      "function",
				PositionX: 600,
				PositionY: 40,
				Config: json.RawMessage(`{
					"code": "if (!input.slug) return { error: 'slug is required' }; const block = ctx.blocks.getWithSchema(input.slug); if (!block) return { error: 'Block not found: ' + input.slug }; return block;"
				}`),
			},
			{
				TempID:    "search_blocks",
				Name:      "search_blocks",
				Type:      "function",
				PositionX: 720,
				PositionY: 40,
				Config: json.RawMessage(`{
					"code": "const query = (input.query || '').toLowerCase(); const blocks = ctx.blocks.list(); const matched = blocks.filter(b => (b.name && b.name.toLowerCase().includes(query)) || (b.description && b.description.toLowerCase().includes(query)) || (b.slug && b.slug.toLowerCase().includes(query))); return { blocks: matched.map(b => ({ slug: b.slug, name: b.name, category: b.category, description: b.description })), count: matched.length };"
				}`),
			},

			// Workflow Tools
			{
				TempID:    "list_workflows",
				Name:      "list_workflows",
				Type:      "function",
				PositionX: 480,
				PositionY: 120,
				Config: json.RawMessage(`{
					"code": "const workflows = ctx.workflows.list(); return { workflows: workflows.map(w => ({ id: w.id, name: w.name, description: w.description })), count: workflows.length };"
				}`),
			},
			{
				TempID:    "get_workflow",
				Name:      "get_workflow",
				Type:      "function",
				PositionX: 600,
				PositionY: 120,
				Config: json.RawMessage(`{
					"code": "if (!input.workflow_id) return { error: 'workflow_id is required' }; const wf = ctx.workflows.get(input.workflow_id); if (!wf) return { error: 'Workflow not found: ' + input.workflow_id }; return wf;"
				}`),
			},

			// Step Tools
			{
				TempID:    "create_step",
				Name:      "create_step",
				Type:      "function",
				PositionX: 480,
				PositionY: 200,
				Config: json.RawMessage(`{
					"code": "if (!input.project_id || !input.name || !input.type) return { error: 'project_id, name, and type are required' }; const step = ctx.steps.create({ project_id: input.project_id, name: input.name, type: input.type, config: input.config || {}, position_x: input.position_x || 0, position_y: input.position_y || 0, block_definition_id: input.block_definition_id }); return step;"
				}`),
			},
			{
				TempID:    "update_step",
				Name:      "update_step",
				Type:      "function",
				PositionX: 600,
				PositionY: 200,
				Config: json.RawMessage(`{
					"code": "if (!input.step_id) return { error: 'step_id is required' }; const updates = {}; if (input.name) updates.name = input.name; if (input.config) updates.config = input.config; if (input.position_x !== undefined) updates.position_x = input.position_x; if (input.position_y !== undefined) updates.position_y = input.position_y; const step = ctx.steps.update(input.step_id, updates); return step;"
				}`),
			},
			{
				TempID:    "delete_step",
				Name:      "delete_step",
				Type:      "function",
				PositionX: 720,
				PositionY: 200,
				Config: json.RawMessage(`{
					"code": "if (!input.step_id) return { error: 'step_id is required' }; ctx.steps.delete(input.step_id); return { success: true, deleted_step_id: input.step_id };"
				}`),
			},

			// Edge Tools
			{
				TempID:    "create_edge",
				Name:      "create_edge",
				Type:      "function",
				PositionX: 480,
				PositionY: 280,
				Config: json.RawMessage(`{
					"code": "if (!input.project_id || !input.source_step_id || !input.target_step_id) return { error: 'project_id, source_step_id, and target_step_id are required' }; const edge = ctx.edges.create({ project_id: input.project_id, source_step_id: input.source_step_id, target_step_id: input.target_step_id, source_port: input.source_port || 'output', target_port: input.target_port || 'input', condition: input.condition }); return edge;"
				}`),
			},
			{
				TempID:    "delete_edge",
				Name:      "delete_edge",
				Type:      "function",
				PositionX: 600,
				PositionY: 280,
				Config: json.RawMessage(`{
					"code": "if (!input.edge_id) return { error: 'edge_id is required' }; ctx.edges.delete(input.edge_id); return { success: true, deleted_edge_id: input.edge_id };"
				}`),
			},

			// Documentation Search (using RAG if available)
			{
				TempID:    "search_documentation",
				Name:      "search_documentation",
				Type:      "function",
				PositionX: 480,
				PositionY: 360,
				Config: json.RawMessage(`{
					"code": "if (!input.query) return { error: 'query is required' }; if (ctx.vector && ctx.embedding) { try { const embedding = ctx.embedding.embed('openai', 'text-embedding-3-small', [input.query]); const results = ctx.vector.query('platform-docs', embedding.vectors[0], { topK: 5 }); return { results: results.matches || [], query: input.query }; } catch(e) { return { error: 'Documentation search not available', query: input.query }; } } return { error: 'Vector service not available', query: input.query };"
				}`),
			},

			// Workflow Validation
			{
				TempID:    "validate_workflow",
				Name:      "validate_workflow",
				Type:      "function",
				PositionX: 600,
				PositionY: 360,
				Config: json.RawMessage(`{
					"code": "if (!input.workflow_id) return { error: 'workflow_id is required' }; const wf = ctx.workflows.get(input.workflow_id); if (!wf) return { error: 'Workflow not found: ' + input.workflow_id, valid: false }; const errors = []; const steps = wf.steps || []; const startSteps = steps.filter(s => s.type === 'start'); if (startSteps.length === 0) errors.push('No start step found'); const stepIds = new Set(steps.map(s => s.id)); for (const edge of (wf.edges || [])) { if (!stepIds.has(edge.source_step_id)) errors.push('Edge references non-existent source step'); if (!stepIds.has(edge.target_step_id)) errors.push('Edge references non-existent target step'); } return { valid: errors.length === 0, errors: errors, step_count: steps.length, edge_count: (wf.edges || []).length };"
				}`),
			},
		},
		Edges: []SystemEdgeDefinition{
			// Main flow
			{SourceTempID: "start", TargetTempID: "set_context"},
			{SourceTempID: "set_context", TargetTempID: "copilot_agent"},
		},
	}
}

// copilotAgentConfig returns the configuration for the Copilot agent
func copilotAgentConfig() string {
	config := map[string]interface{}{
		"provider":       "anthropic",
		"model":          "claude-sonnet-4-20250514",
		"max_iterations": 15,
		"temperature":    0.7,
		"enable_memory":  true,
		"memory_window":  30,
		"system_prompt":  copilotSystemPrompt(),
		"tools":          copilotTools(),
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

func copilotTools() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":        "list_blocks",
			"description": "Get a list of all available blocks with their basic information (slug, name, category, description)",
			"parameters": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "get_block_schema",
			"description": "Get the detailed configuration schema for a specific block, including all configurable options",
			"parameters": map[string]interface{}{
				"type":     "object",
				"required": []string{"slug"},
				"properties": map[string]interface{}{
					"slug": map[string]interface{}{
						"type":        "string",
						"description": "The block's slug identifier (e.g., 'llm', 'http', 'slack')",
					},
				},
			},
		},
		{
			"name":        "search_blocks",
			"description": "Search for blocks by keyword in name, description, or slug",
			"parameters": map[string]interface{}{
				"type":     "object",
				"required": []string{"query"},
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search keyword",
					},
				},
			},
		},
		{
			"name":        "list_workflows",
			"description": "Get a list of all workflows accessible to the user",
			"parameters": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "get_workflow",
			"description": "Get detailed information about a specific workflow, including its steps and edges",
			"parameters": map[string]interface{}{
				"type":     "object",
				"required": []string{"workflow_id"},
				"properties": map[string]interface{}{
					"workflow_id": map[string]interface{}{
						"type":        "string",
						"description": "The workflow's UUID",
					},
				},
			},
		},
		{
			"name":        "create_step",
			"description": "Create a new step in a workflow",
			"parameters": map[string]interface{}{
				"type":     "object",
				"required": []string{"project_id", "name", "type"},
				"properties": map[string]interface{}{
					"project_id": map[string]interface{}{
						"type":        "string",
						"description": "The workflow/project ID where the step will be created",
					},
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Name of the step",
					},
					"type": map[string]interface{}{
						"type":        "string",
						"description": "Step type (e.g., 'llm', 'http', 'function')",
					},
					"config": map[string]interface{}{
						"type":        "object",
						"description": "Step configuration options",
					},
					"position_x": map[string]interface{}{
						"type":        "integer",
						"description": "X position on canvas",
					},
					"position_y": map[string]interface{}{
						"type":        "integer",
						"description": "Y position on canvas",
					},
					"block_definition_id": map[string]interface{}{
						"type":        "string",
						"description": "UUID of the block definition to use",
					},
				},
			},
		},
		{
			"name":        "update_step",
			"description": "Update an existing step's configuration or position",
			"parameters": map[string]interface{}{
				"type":     "object",
				"required": []string{"step_id"},
				"properties": map[string]interface{}{
					"step_id": map[string]interface{}{
						"type":        "string",
						"description": "The step's UUID",
					},
					"name": map[string]interface{}{
						"type":        "string",
						"description": "New name for the step",
					},
					"config": map[string]interface{}{
						"type":        "object",
						"description": "Updated configuration",
					},
					"position_x": map[string]interface{}{
						"type":        "integer",
						"description": "New X position",
					},
					"position_y": map[string]interface{}{
						"type":        "integer",
						"description": "New Y position",
					},
				},
			},
		},
		{
			"name":        "delete_step",
			"description": "Delete a step from the workflow",
			"parameters": map[string]interface{}{
				"type":     "object",
				"required": []string{"step_id"},
				"properties": map[string]interface{}{
					"step_id": map[string]interface{}{
						"type":        "string",
						"description": "The step's UUID to delete",
					},
				},
			},
		},
		{
			"name":        "create_edge",
			"description": "Create a connection between two steps",
			"parameters": map[string]interface{}{
				"type":     "object",
				"required": []string{"project_id", "source_step_id", "target_step_id"},
				"properties": map[string]interface{}{
					"project_id": map[string]interface{}{
						"type":        "string",
						"description": "The workflow/project ID",
					},
					"source_step_id": map[string]interface{}{
						"type":        "string",
						"description": "UUID of the source step",
					},
					"target_step_id": map[string]interface{}{
						"type":        "string",
						"description": "UUID of the target step",
					},
					"source_port": map[string]interface{}{
						"type":        "string",
						"description": "Output port name (default: 'output')",
					},
					"target_port": map[string]interface{}{
						"type":        "string",
						"description": "Input port name (default: 'input')",
					},
					"condition": map[string]interface{}{
						"type":        "string",
						"description": "Optional condition expression",
					},
				},
			},
		},
		{
			"name":        "delete_edge",
			"description": "Delete a connection between steps",
			"parameters": map[string]interface{}{
				"type":     "object",
				"required": []string{"edge_id"},
				"properties": map[string]interface{}{
					"edge_id": map[string]interface{}{
						"type":        "string",
						"description": "The edge's UUID to delete",
					},
				},
			},
		},
		{
			"name":        "search_documentation",
			"description": "Search platform documentation for relevant information",
			"parameters": map[string]interface{}{
				"type":     "object",
				"required": []string{"query"},
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query for documentation",
					},
				},
			},
		},
		{
			"name":        "validate_workflow",
			"description": "Validate a workflow's structure and identify potential issues",
			"parameters": map[string]interface{}{
				"type":     "object",
				"required": []string{"workflow_id"},
				"properties": map[string]interface{}{
					"workflow_id": map[string]interface{}{
						"type":        "string",
						"description": "The workflow's UUID to validate",
					},
				},
			},
		},
	}
}
