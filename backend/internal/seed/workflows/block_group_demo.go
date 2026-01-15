package workflows

import "encoding/json"

func (r *Registry) registerBlockGroupDemoWorkflows() {
	r.register(BlockGroupDemoWorkflow())
}

// BlockGroupDemoWorkflow demonstrates all block group types (parallel, try_catch, foreach, while)
// This workflow shows how to use control flow constructs for complex workflow patterns
func BlockGroupDemoWorkflow() *SystemWorkflowDefinition {
	return &SystemWorkflowDefinition{
		ID:          "a0000000-0000-0000-0000-000000000204",
		SystemSlug:  "block-group-demo",
		Name:        "Block Group Demo",
		Description: "Demonstrates block group types: parallel, try_catch, foreach, and while for advanced control flow",
		Version:     4,
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
		// Layout: Left-to-Right flow with groups containing their steps
		BlockGroups: []SystemBlockGroupDefinition{
			// Parallel group: executes multiple branches concurrently
			{
				TempID:    "parallel_group",
				Name:      "Parallel Processing",
				Type:      "parallel",
				PositionX: 400,
				PositionY: 80,
				Width:     280,
				Height:    320,
				Config: json.RawMessage(`{
					"max_concurrent": 3,
					"fail_fast": false
				}`),
			},
			// Try-Catch group: handles errors gracefully
			{
				TempID:    "try_catch_group",
				Name:      "Error Handling",
				Type:      "try_catch",
				PositionX: 900,
				PositionY: 140,
				Width:     240,
				Height:    160,
				Config: json.RawMessage(`{
					"retry_count": 2,
					"retry_delay_ms": 1000
				}`),
			},
			// ForEach group: iterates over array items
			{
				TempID:    "foreach_group",
				Name:      "Item Iterator",
				Type:      "foreach",
				PositionX: 1200,
				PositionY: 140,
				Width:     240,
				Height:    160,
				Config: json.RawMessage(`{
					"input_path": "$.items",
					"parallel": true,
					"max_workers": 5
				}`),
			},
			// While group: repeats until condition is false
			{
				TempID:    "while_group",
				Name:      "Counter Loop",
				Type:      "while",
				PositionX: 1500,
				PositionY: 140,
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
				PositionY: 200,
				Config:    json.RawMessage(`{}`),
			},
			// === Initialize (outside groups) ===
			{
				TempID:    "init",
				Name:      "Initialize",
				Type:      "function",
				PositionX: 200,
				PositionY: 200,
				Config: json.RawMessage(`{
					"code": "return { ...input, counter: 0, max_iterations: input.max_iterations || 5, results: [] };",
					"language": "javascript"
				}`),
			},

			// === Parallel Group Steps (inside group, relative positions) ===
			// Branch A: Process first half
			{
				TempID:           "parallel_branch_a",
				Name:             "Branch A",
				Type:             "function",
				PositionX:        450,
				PositionY:        130,
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
				PositionY:        220,
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
				PositionY:        310,
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
				PositionY: 200,
				BlockSlug: "join",
				Config:    json.RawMessage(`{}`),
			},

			// === Try-Catch Group Steps ===
			{
				TempID:           "try_operation",
				Name:             "Risky Op",
				Type:             "function",
				PositionX:        940,
				PositionY:        200,
				BlockGroupTempID: "try_catch_group",
				Config: json.RawMessage(`{
					"code": "if (input.items && input.items.length === 0) { throw new Error('Empty items array'); } return { success: true, processed: input.items.length };",
					"language": "javascript"
				}`),
			},

			// === ForEach Group Steps ===
			{
				TempID:           "foreach_process",
				Name:             "Process Item",
				Type:             "function",
				PositionX:        1240,
				PositionY:        200,
				BlockGroupTempID: "foreach_group",
				Config: json.RawMessage(`{
					"code": "const item = input.item || input; return { id: item.id, processed_name: (item.name || 'unknown').toUpperCase(), doubled_value: (item.value || 0) * 2 };",
					"language": "javascript"
				}`),
			},

			// === While Group Steps ===
			{
				TempID:           "while_increment",
				Name:             "Increment",
				Type:             "function",
				PositionX:        1540,
				PositionY:        200,
				BlockGroupTempID: "while_group",
				Config: json.RawMessage(`{
					"code": "const counter = (input.counter || 0) + 1; return { ...input, counter: counter, iteration_result: 'Iteration ' + counter };",
					"language": "javascript"
				}`),
			},

			// === Final Output (outside groups) ===
			{
				TempID:    "final_output",
				Name:      "Final Output",
				Type:      "function",
				PositionX: 1800,
				PositionY: 200,
				Config: json.RawMessage(`{
					"code": "return { parallel_results: input.parallel || {}, foreach_results: input.foreach_results || [], while_result: { final_counter: input.counter || 0 }, final_status: 'completed' };",
					"language": "javascript"
				}`),
			},
		},

		Edges: []SystemEdgeDefinition{
			// === External Edges (connecting to group boundaries) ===
			// Start -> Init
			{SourceTempID: "start", TargetTempID: "init", SourcePort: "output"},
			// Init -> Parallel Group (external edge to group IN port)
			{SourceTempID: "init", TargetGroupTempID: "parallel_group", SourcePort: "output", TargetPort: "in"},
			// Parallel Group -> Merge (external edge from group OUT port)
			{SourceGroupTempID: "parallel_group", TargetTempID: "merge_parallel", SourcePort: "out"},
			// Merge -> Try-Catch Group
			{SourceTempID: "merge_parallel", TargetGroupTempID: "try_catch_group", SourcePort: "output", TargetPort: "in"},
			// Try-Catch Group -> ForEach Group (group to group)
			{SourceGroupTempID: "try_catch_group", TargetGroupTempID: "foreach_group", SourcePort: "out", TargetPort: "in"},
			// ForEach Group -> While Group (group to group)
			{SourceGroupTempID: "foreach_group", TargetGroupTempID: "while_group", SourcePort: "out", TargetPort: "in"},
			// While Group -> Final Output
			{SourceGroupTempID: "while_group", TargetTempID: "final_output", SourcePort: "out"},

			// === Internal Edges (within groups - connecting internal steps) ===
			// Note: These edges connect steps that are inside groups to each other
			// The group's IN port connects to the first step(s), and last step(s) connect to OUT port
			// This is handled by the execution engine based on group configuration
		},
	}
}
