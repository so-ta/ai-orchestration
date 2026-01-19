package blocks

import (
	"encoding/json"

	"github.com/souta/ai-orchestration/internal/domain"
)

// registerGroupBlocks registers all group blocks (containers)
func (r *Registry) registerGroupBlocks() {
	r.register(parallelBlock())
	r.register(tryCatchBlock())
	r.register(foreachBlock())
	r.register(whileBlock())
	r.register(agentGroupBlock())
}

// parallelBlock defines the parallel group block
func parallelBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "parallel",
		Version:     1,
		Name:        "Parallel",
		Description: "Execute multiple independent flows concurrently within the group",
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryControl,
		Icon:        "git-branch",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"max_concurrent": {
					"type": "integer",
					"title": "Max Concurrent",
					"description": "Maximum concurrent executions (0 = unlimited)",
					"default": 0
				},
				"fail_fast": {
					"type": "boolean",
					"title": "Fail Fast",
					"description": "Stop all flows on first failure",
					"default": false
				}
			}
		}`),
		InputPorts: DefaultInputPorts(),
		OutputPorts: []domain.OutputPort{
			{Name: "out", Label: "Complete", IsDefault: true},
			{Name: "error", Label: "Error", IsDefault: false},
		},
		Code: `// Parallel execution is handled by the engine
// pre_process: transforms external input to internal input
// post_process: transforms internal outputs to external output
return input;`,
		UIConfig: json.RawMessage(`{
			"icon": "git-branch",
			"color": "#3B82F6",
			"isContainer": true
		}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "PAR_001", Name: "FLOW_FAILED", Description: "One or more flows failed", Retryable: false},
		},
		Enabled:     true,
		GroupKind:   domain.BlockGroupKindParallel,
		IsContainer: true,
	}
}

// tryCatchBlock defines the try-catch group block
func tryCatchBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "try_catch",
		Version:     1,
		Name:        "Try-Catch",
		Description: "Execute body with error handling and retry support",
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryControl,
		Icon:        "shield",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"retry_count": {
					"type": "integer",
					"title": "Retry Count",
					"description": "Number of retries on error (default: 0)",
					"default": 0,
					"minimum": 0,
					"maximum": 10
				},
				"retry_delay_ms": {
					"type": "integer",
					"title": "Retry Delay (ms)",
					"description": "Delay between retries in milliseconds",
					"default": 1000,
					"minimum": 0
				}
			}
		}`),
		InputPorts: DefaultInputPorts(),
		OutputPorts: []domain.OutputPort{
			{Name: "out", Label: "Success", IsDefault: true},
			{Name: "error", Label: "Error", IsDefault: false},
		},
		Code: `// Try-catch execution is handled by the engine
// Body is executed with retry support
// Errors are routed to error port
return input;`,
		UIConfig: json.RawMessage(`{
			"icon": "shield",
			"color": "#EF4444",
			"isContainer": true
		}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "TRY_001", Name: "MAX_RETRIES", Description: "Maximum retries exceeded", Retryable: false},
		},
		Enabled:     true,
		GroupKind:   domain.BlockGroupKindTryCatch,
		IsContainer: true,
	}
}

// foreachBlock defines the foreach group block
func foreachBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "foreach",
		Version:     1,
		Name:        "For Each",
		Description: "Iterate over array elements, executing the same process for each",
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryControl,
		Icon:        "repeat",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"input_path": {
					"type": "string",
					"title": "Input Path",
					"description": "JSONPath to array in input (default: $.items)",
					"default": "$.items"
				},
				"parallel": {
					"type": "boolean",
					"title": "Parallel Execution",
					"description": "Execute iterations in parallel",
					"default": false
				},
				"max_workers": {
					"type": "integer",
					"title": "Max Workers",
					"description": "Maximum parallel workers (0 = unlimited)",
					"default": 0
				}
			}
		}`),
		InputPorts: DefaultInputPorts(),
		OutputPorts: []domain.OutputPort{
			{Name: "out", Label: "Complete", IsDefault: true},
			{Name: "error", Label: "Error", IsDefault: false},
		},
		Code: `// ForEach execution is handled by the engine
// Each iteration receives: { item, index, context }
// Results are aggregated into an array
return input;`,
		UIConfig: json.RawMessage(`{
			"icon": "repeat",
			"color": "#8B5CF6",
			"isContainer": true
		}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "FOR_001", Name: "ITERATION_FAILED", Description: "One or more iterations failed", Retryable: false},
			{Code: "FOR_002", Name: "EMPTY_INPUT", Description: "Input array is empty", Retryable: false},
		},
		Enabled:     true,
		GroupKind:   domain.BlockGroupKindForeach,
		IsContainer: true,
	}
}

// whileBlock defines the while group block
func whileBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "while",
		Version:     1,
		Name:        "While",
		Description: "Repeat body execution while condition is true",
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryControl,
		Icon:        "rotate-cw",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"condition": {
					"type": "string",
					"title": "Condition",
					"description": "Condition expression (e.g., $.counter < $.target)"
				},
				"max_iterations": {
					"type": "integer",
					"title": "Max Iterations",
					"description": "Safety limit to prevent infinite loops",
					"default": 100,
					"minimum": 1,
					"maximum": 10000
				},
				"do_while": {
					"type": "boolean",
					"title": "Do-While Mode",
					"description": "Execute body at least once before checking condition",
					"default": false
				}
			},
			"required": ["condition"]
		}`),
		InputPorts: DefaultInputPorts(),
		OutputPorts: []domain.OutputPort{
			{Name: "out", Label: "Done", IsDefault: true},
			{Name: "error", Label: "Error", IsDefault: false},
		},
		Code: `// While execution is handled by the engine
// Body output becomes next iteration input
// Loop exits when condition is false
return input;`,
		UIConfig: json.RawMessage(`{
			"icon": "rotate-cw",
			"color": "#F59E0B",
			"isContainer": true
		}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "WHL_001", Name: "MAX_ITERATIONS", Description: "Maximum iterations exceeded", Retryable: false},
			{Code: "WHL_002", Name: "CONDITION_ERROR", Description: "Failed to evaluate condition", Retryable: false},
		},
		Enabled:     true,
		GroupKind:   domain.BlockGroupKindWhile,
		IsContainer: true,
	}
}

// agentGroupBlock defines the agent group block
// Child steps become tools that the AI agent can call
func agentGroupBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "agent-group",
		Version:     1,
		Name:        "Agent",
		Description: "AI agent with ReAct loop - child steps become callable tools",
		Category:    domain.BlockCategoryAI,
		Subcategory: domain.BlockSubcategoryAgent,
		Icon:        "bot",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"required": ["provider", "model", "system_prompt"],
			"properties": {
				"provider": {
					"type": "string",
					"title": "Provider",
					"enum": ["openai", "anthropic"],
					"default": "anthropic",
					"description": "LLM provider"
				},
				"model": {
					"type": "string",
					"title": "Model",
					"default": "claude-sonnet-4-20250514",
					"description": "Model ID (e.g., claude-sonnet-4-20250514, gpt-4)"
				},
				"system_prompt": {
					"type": "string",
					"title": "System Prompt",
					"maxLength": 50000,
					"description": "System prompt defining the agent's behavior and capabilities"
				},
				"max_iterations": {
					"type": "integer",
					"title": "Max Iterations",
					"default": 10,
					"minimum": 1,
					"maximum": 50,
					"description": "Maximum ReAct loop iterations"
				},
				"temperature": {
					"type": "number",
					"title": "Temperature",
					"default": 0.7,
					"minimum": 0,
					"maximum": 2,
					"description": "LLM temperature (0-2)"
				},
				"tool_choice": {
					"type": "string",
					"title": "Tool Choice",
					"enum": ["auto", "none", "required"],
					"default": "auto",
					"description": "How the agent should use tools"
				},
				"enable_memory": {
					"type": "boolean",
					"title": "Enable Memory",
					"default": false,
					"description": "Enable conversation memory across runs"
				},
				"memory_window": {
					"type": "integer",
					"title": "Memory Window",
					"default": 20,
					"minimum": 1,
					"maximum": 100,
					"description": "Number of messages to keep in memory"
				}
			}
		}`),
		InputPorts: []domain.InputPort{
			{Name: "in", Label: "Input", Required: true, Description: "User message or task input"},
		},
		OutputPorts: []domain.OutputPort{
			{Name: "out", Label: "Response", IsDefault: true, Description: "Agent's final response"},
			{Name: "error", Label: "Error", IsDefault: false, Description: "Error output"},
		},
		Code: `// Agent execution is handled by the engine's executeAgent()
// Child steps become tools that the agent can call
return input;`,
		UIConfig: json.RawMessage(`{
			"icon": "bot",
			"color": "#10B981",
			"isContainer": true,
			"groups": [
				{"id": "model", "icon": "robot", "title": "Model Settings"},
				{"id": "agent", "icon": "bot", "title": "Agent Settings"},
				{"id": "memory", "icon": "database", "title": "Memory Settings"}
			],
			"fieldGroups": {
				"provider": "model",
				"model": "model",
				"system_prompt": "agent",
				"max_iterations": "agent",
				"temperature": "agent",
				"tool_choice": "agent",
				"enable_memory": "memory",
				"memory_window": "memory"
			},
			"fieldOverrides": {
				"system_prompt": {"rows": 8, "widget": "textarea"}
			}
		}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "AGENT_001", Name: "MAX_ITERATIONS", Description: "Agent reached maximum iterations", Retryable: false},
			{Code: "AGENT_002", Name: "TOOL_ERROR", Description: "Tool execution failed", Retryable: true},
			{Code: "AGENT_003", Name: "LLM_ERROR", Description: "LLM API error", Retryable: true},
		},
		RequiredCredentials: json.RawMessage(`[{"name": "llm_api_key", "type": "api_key", "scope": "system", "required": true, "description": "LLM Provider API Key"}]`),
		Enabled:             true,
		GroupKind:           domain.BlockGroupKindAgent,
		IsContainer:         true,
	}
}
