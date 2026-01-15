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
}

// parallelBlock defines the parallel group block
func parallelBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "parallel",
		Version:     1,
		Name:        "Parallel",
		Description: "Execute multiple independent flows concurrently within the group",
		Category:    domain.BlockCategoryGroup,
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
		Category:    domain.BlockCategoryGroup,
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
		Category:    domain.BlockCategoryGroup,
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
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"items": {
					"type": "array",
					"description": "Array of items to iterate"
				}
			},
			"required": ["items"]
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
		Category:    domain.BlockCategoryGroup,
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
