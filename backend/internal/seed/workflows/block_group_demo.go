package workflows

import "encoding/json"

func (r *Registry) registerBlockGroupDemoWorkflows() {
	r.register(BlockGroupDemoWorkflow())
}

// BlockGroupDemoWorkflow demonstrates all block group types (parallel, try_catch, foreach, while)
// This workflow shows how to use control flow constructs for complex workflow patterns
// Each block group has multiple output ports that can be connected to different paths
func BlockGroupDemoWorkflow() *SystemWorkflowDefinition {
	return &SystemWorkflowDefinition{
		ID:          "a0000000-0000-0000-0000-000000000204",
		SystemSlug:  "block-group-demo",
		Name:        "Block Group Demo",
		Description: "Demonstrates block group types: parallel, try_catch, foreach, and while with multiple output ports",
		Version:     5,
		IsSystem:    true,
		InputSchema: json.RawMessage(`{
			"type": "object",
			"required": ["items"],
			"properties": {
				"items": {
					"type": "array",
					"title": "Items",
					"description": "Array of items to process",
					"items": {
						"type": "object",
						"properties": {
							"id": {"type": "integer"},
							"value": {"type": "number"},
							"name": {"type": "string"}
						}
					}
				},
				"max_iterations": {
					"type": "integer",
					"title": "Max Iterations",
					"description": "Maximum iterations for while loop",
					"default": 5
				}
			}
		}`),
		OutputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"parallel_results": {"type": "object"},
				"foreach_results": {"type": "array"},
				"while_result": {"type": "object"},
				"final_status": {"type": "string"}
			}
		}`),

		// Block Groups define the control flow constructs
		// Each group type has specific output ports:
		// - parallel: out (complete), error
		// - try_catch: out (success), error
		// - foreach: out (complete), error
		// - while: out (done), error
		BlockGroups: []SystemBlockGroupDefinition{
			// Parallel group: executes multiple branches concurrently
			// Output ports: out (all succeeded), error (any failed)
			{
				TempID:    "parallel_group",
				Name:      "Parallel Processing",
				Type:      "parallel",
				PositionX: 400,
				PositionY: 50,
				Width:     280,
				Height:    320,
				Config: json.RawMessage(`{
					"max_concurrent": 3,
					"fail_fast": false
				}`),
			},
			// Try-Catch group: handles errors gracefully
			// Output ports: out (success), error (caught)
			{
				TempID:    "try_catch_group",
				Name:      "Error Handling",
				Type:      "try_catch",
				PositionX: 900,
				PositionY: 50,
				Width:     240,
				Height:    160,
				Config: json.RawMessage(`{
					"retry_count": 2,
					"retry_delay_ms": 1000
				}`),
			},
			// ForEach group: iterates over array items
			// Output ports: out (all items processed), error (iteration failed)
			{
				TempID:    "foreach_group",
				Name:      "Item Iterator",
				Type:      "foreach",
				PositionX: 1200,
				PositionY: 50,
				Width:     240,
				Height:    160,
				Config: json.RawMessage(`{
					"input_path": "$.items",
					"parallel": true,
					"max_workers": 5
				}`),
			},
			// While group: repeats until condition is false
			// Output ports: out (condition became false), error (limit reached)
			{
				TempID:    "while_group",
				Name:      "Counter Loop",
				Type:      "while",
				PositionX: 1500,
				PositionY: 50,
				Width:     240,
				Height:    160,
				Config: json.RawMessage(`{
					"condition": "$.counter < $.max_iterations",
					"max_iterations": 100,
					"do_while": false
				}`),
			},
		},

		Steps: []SystemStepDefinition{
			// === Start (outside groups) ===
			{
				TempID:    "start",
				Name:      "Start",
				Type:      "start",
				PositionX: 40,
				PositionY: 180,
				Config:    json.RawMessage(`{}`),
			},
			// === Initialize (outside groups) ===
			{
				TempID:    "init",
				Name:      "Initialize",
				Type:      "function",
				PositionX: 200,
				PositionY: 180,
				Config: json.RawMessage(`{
					"code": "return { ...input, counter: 0, max_iterations: input.max_iterations || 5, results: [] };",
					"language": "javascript"
				}`),
			},

			// === Parallel Group Steps (inside group) ===
			// Branch A: Process first half
			{
				TempID:           "parallel_branch_a",
				Name:             "Branch A",
				Type:             "function",
				PositionX:        450,
				PositionY:        100,
				BlockGroupTempID: "parallel_group",
				Config: json.RawMessage(`{
					"code": "const items = input.items || []; const half = Math.floor(items.length / 2); return { branch: 'A', processed: items.slice(0, half), count: half };",
					"language": "javascript"
				}`),
			},
			// Branch B: Process second half
			{
				TempID:           "parallel_branch_b",
				Name:             "Branch B",
				Type:             "function",
				PositionX:        450,
				PositionY:        190,
				BlockGroupTempID: "parallel_group",
				Config: json.RawMessage(`{
					"code": "const items = input.items || []; const half = Math.floor(items.length / 2); return { branch: 'B', processed: items.slice(half), count: items.length - half };",
					"language": "javascript"
				}`),
			},
			// Branch C: Calculate stats
			{
				TempID:           "parallel_branch_c",
				Name:             "Branch C",
				Type:             "function",
				PositionX:        450,
				PositionY:        280,
				BlockGroupTempID: "parallel_group",
				Config: json.RawMessage(`{
					"code": "const items = input.items || []; const values = items.map(i => i.value || 0); const sum = values.reduce((a, b) => a + b, 0); return { branch: 'C', sum: sum, avg: values.length > 0 ? sum / values.length : 0, count: values.length };",
					"language": "javascript"
				}`),
			},

			// === Merge Parallel Results (outside groups) ===
			{
				TempID:    "merge_parallel",
				Name:      "Merge Results",
				Type:      "join",
				PositionX: 740,
				PositionY: 180,
				BlockSlug: "join",
				Config:    json.RawMessage(`{}`),
			},

			// === Parallel Error Handler (handles parallel group error) ===
			{
				TempID:    "parallel_error",
				Name:      "Parallel Error",
				Type:      "function",
				PositionX: 740,
				PositionY: 420,
				Config: json.RawMessage(`{
					"code": "return { error: 'Parallel processing failed', original_error: input.error || 'Unknown error' };",
					"language": "javascript"
				}`),
			},

			// === Try-Catch Group Steps ===
			{
				TempID:           "try_operation",
				Name:             "Risky Op",
				Type:             "function",
				PositionX:        940,
				PositionY:        110,
				BlockGroupTempID: "try_catch_group",
				Config: json.RawMessage(`{
					"code": "if (input.items && input.items.length === 0) { throw new Error('Empty items array'); } return { success: true, processed: input.items.length };",
					"language": "javascript"
				}`),
			},

			// === Catch Handler (handles caught errors from try_catch) ===
			{
				TempID:    "catch_handler",
				Name:      "Catch Handler",
				Type:      "function",
				PositionX: 900,
				PositionY: 280,
				Config: json.RawMessage(`{
					"code": "return { handled: true, message: 'Error was caught and handled', original_error: input.error || 'Unknown' };",
					"language": "javascript"
				}`),
			},

			// === ForEach Group Steps ===
			{
				TempID:           "foreach_process",
				Name:             "Process Item",
				Type:             "function",
				PositionX:        1240,
				PositionY:        110,
				BlockGroupTempID: "foreach_group",
				Config: json.RawMessage(`{
					"code": "const item = input.item || input; return { id: item.id, processed_name: (item.name || 'unknown').toUpperCase(), doubled_value: (item.value || 0) * 2 };",
					"language": "javascript"
				}`),
			},

			// === ForEach Error Handler ===
			{
				TempID:    "foreach_error",
				Name:      "ForEach Error",
				Type:      "function",
				PositionX: 1200,
				PositionY: 280,
				Config: json.RawMessage(`{
					"code": "return { error: 'ForEach iteration failed', item_error: input.error || 'Unknown' };",
					"language": "javascript"
				}`),
			},

			// === While Group Steps ===
			{
				TempID:           "while_increment",
				Name:             "Increment",
				Type:             "function",
				PositionX:        1540,
				PositionY:        110,
				BlockGroupTempID: "while_group",
				Config: json.RawMessage(`{
					"code": "const counter = (input.counter || 0) + 1; return { ...input, counter: counter, iteration_result: 'Iteration ' + counter };",
					"language": "javascript"
				}`),
			},

			// === Max Iterations Handler ===
			{
				TempID:    "max_iterations_handler",
				Name:      "Max Reached",
				Type:      "function",
				PositionX: 1500,
				PositionY: 280,
				Config: json.RawMessage(`{
					"code": "return { warning: 'Maximum iterations reached', final_counter: input.counter || 0 };",
					"language": "javascript"
				}`),
			},

			// === Final Output (outside groups) ===
			{
				TempID:    "final_output",
				Name:      "Final Output",
				Type:      "function",
				PositionX: 1800,
				PositionY: 180,
				Config: json.RawMessage(`{
					"code": "return { parallel_results: input.parallel || {}, foreach_results: input.foreach_results || [], while_result: { final_counter: input.counter || 0 }, final_status: 'completed' };",
					"language": "javascript"
				}`),
			},

			// === Error Aggregator (collects all error paths) ===
			{
				TempID:    "error_aggregator",
				Name:      "Error Summary",
				Type:      "join",
				PositionX: 1800,
				PositionY: 420,
				BlockSlug: "join",
				Config:    json.RawMessage(`{}`),
			},
		},

		Edges: []SystemEdgeDefinition{
			// === Main Flow (Success Path) ===
			// Start -> Init
			{SourceTempID: "start", TargetTempID: "init", SourcePort: "output"},

			// Init -> Parallel Group (input to group)
			{SourceTempID: "init", TargetGroupTempID: "parallel_group", SourcePort: "output", TargetPort: "group-input"},

			// Parallel Group SUCCESS -> Merge (using 'out' output port)
			{SourceGroupTempID: "parallel_group", TargetTempID: "merge_parallel", SourcePort: "out"},

			// Parallel Group ERROR -> Parallel Error Handler (using 'error' output port)
			{SourceGroupTempID: "parallel_group", TargetTempID: "parallel_error", SourcePort: "error"},

			// Merge -> Try-Catch Group
			{SourceTempID: "merge_parallel", TargetGroupTempID: "try_catch_group", SourcePort: "output", TargetPort: "group-input"},

			// Try-Catch Group SUCCESS -> ForEach Group (using 'out' output port)
			{SourceGroupTempID: "try_catch_group", TargetGroupTempID: "foreach_group", SourcePort: "out", TargetPort: "group-input"},

			// Try-Catch Group ERROR -> Catch Handler (using 'error' output port)
			{SourceGroupTempID: "try_catch_group", TargetTempID: "catch_handler", SourcePort: "error"},

			// ForEach Group COMPLETE -> While Group (using 'out' output port)
			{SourceGroupTempID: "foreach_group", TargetGroupTempID: "while_group", SourcePort: "out", TargetPort: "group-input"},

			// ForEach Group ERROR -> ForEach Error Handler (using 'error' output port)
			{SourceGroupTempID: "foreach_group", TargetTempID: "foreach_error", SourcePort: "error"},

			// While Group COMPLETE -> Final Output (using 'out' output port)
			{SourceGroupTempID: "while_group", TargetTempID: "final_output", SourcePort: "out"},

			// While Group MAX_ITERATIONS -> Max Handler (using 'error' output port)
			{SourceGroupTempID: "while_group", TargetTempID: "max_iterations_handler", SourcePort: "error"},

			// === Error Paths converge to Error Summary ===
			{SourceTempID: "parallel_error", TargetTempID: "error_aggregator", SourcePort: "output"},
			{SourceTempID: "catch_handler", TargetTempID: "error_aggregator", SourcePort: "output"},
			{SourceTempID: "foreach_error", TargetTempID: "error_aggregator", SourcePort: "output"},
			{SourceTempID: "max_iterations_handler", TargetTempID: "error_aggregator", SourcePort: "output"},
		},
	}
}
