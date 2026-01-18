package workflows

import "encoding/json"

func (r *Registry) registerDemoWorkflows() {
	r.register(DemoWorkflow())
}

// DemoWorkflow demonstrates multiple workflow patterns with multiple entry points
// This workflow consolidates: Block Demo, Data Pipeline, Block Group Demo
// Each entry point demonstrates different block patterns
func DemoWorkflow() *SystemWorkflowDefinition {
	return &SystemWorkflowDefinition{
		ID:          "a0000000-0000-0000-0000-000000000200",
		SystemSlug:  "demo",
		Name:        "Demo Workflows",
		Description: "Demonstrates multiple workflow patterns: block demo, data pipeline, and block groups with multiple entry points",
		Version:     4,
		IsSystem:    true,
		// Note: InputSchema/OutputSchema are defined per entry point in each Start step's config
		// Top-level schemas are optional for multi-entry-point workflows
		InputSchema: json.RawMessage(`{
			"type": "object",
			"description": "This workflow has multiple entry points. See each Start step for specific input schemas.",
			"properties": {}
		}`),
		OutputSchema: json.RawMessage(`{
			"type": "object",
			"description": "Output varies by entry point.",
			"properties": {}
		}`),

		// Block Groups for the block_group entry point
		BlockGroups: []SystemBlockGroupDefinition{
			// Parallel group: executes multiple branches concurrently
			{
				TempID:    "bg_parallel_group",
				Name:      "Parallel Processing",
				Type:      "parallel",
				PositionX: 280,
				PositionY: 400,
				Width:     280,
				Height:    320,
				Config: json.RawMessage(`{
					"max_concurrent": 3,
					"fail_fast": false
				}`),
			},
			// Try-Catch group: handles errors gracefully
			{
				TempID:    "bg_try_catch_group",
				Name:      "Error Handling",
				Type:      "try_catch",
				PositionX: 720,
				PositionY: 400,
				Width:     240,
				Height:    160,
				Config: json.RawMessage(`{
					"retry_count": 2,
					"retry_delay_ms": 1000
				}`),
			},
			// ForEach group: iterates over array items
			{
				TempID:    "bg_foreach_group",
				Name:      "Item Iterator",
				Type:      "foreach",
				PositionX: 1080,
				PositionY: 400,
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
				TempID:    "bg_while_group",
				Name:      "Counter Loop",
				Type:      "while",
				PositionX: 1440,
				PositionY: 400,
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
			// ============================
			// Block Demo Entry Point (横並び: Y=40固定, X増加)
			// Demonstrates: start, log, function, llm, code, wait, note
			// ============================
			{
				TempID:      "start_block_demo",
				Name:        "Start: Block Demo",
				Type:        "start",
				TriggerType: "internal",
				TriggerConfig: json.RawMessage(`{
					"entry_point": "block_demo",
					"description": "Demonstrate core block types"
				}`),
				PositionX: 40,
				PositionY: 40,
				Config: json.RawMessage(`{
					"input_schema": {
						"type": "object",
						"required": ["message", "items"],
						"properties": {
							"message": {"type": "string", "title": "メッセージ"},
							"items": {"type": "array", "title": "アイテム", "items": {"type": "object"}}
						}
					}
				}`),
			},
			{
				TempID:    "block_log_input",
				Name:      "Log Input",
				Type:      "log",
				PositionX: 160,
				PositionY: 40,
				BlockSlug: "log",
				Config: json.RawMessage(`{
					"level": "info",
					"message": "Processing started with {{$.items.length}} items"
				}`),
			},
			{
				TempID:    "block_prepare_data",
				Name:      "Prepare Data",
				Type:      "function",
				PositionX: 280,
				PositionY: 40,
				Config: json.RawMessage(`{
					"code": "const items = input.items || []; const enhanced = items.map((item, idx) => ({...item, index: idx, processed_at: new Date().toISOString()})); return { message: input.message, items: enhanced, item_count: enhanced.length };",
					"language": "javascript"
				}`),
			},
			{
				TempID:    "block_filter_items",
				Name:      "Filter Items",
				Type:      "function",
				PositionX: 400,
				PositionY: 40,
				Config: json.RawMessage(`{
					"code": "const filtered = (input.items || []).filter(item => (item.index || 0) < 10); return { message: input.message, items: filtered, filtered_count: filtered.length };",
					"language": "javascript"
				}`),
			},
			{
				TempID:    "block_transform_items",
				Name:      "Transform Items",
				Type:      "function",
				PositionX: 520,
				PositionY: 40,
				Config: json.RawMessage(`{
					"code": "const transformed = (input.items || []).map(item => ({...item, transformed: true})); return { message: input.message, items: transformed };",
					"language": "javascript"
				}`),
			},
			{
				TempID:    "block_aggregate",
				Name:      "Aggregate Results",
				Type:      "function",
				PositionX: 640,
				PositionY: 40,
				Config: json.RawMessage(`{
					"code": "const items = input.items || []; const total_count = items.length; const max_index = items.reduce((max, item) => Math.max(max, item.index || 0), 0); return { message: input.message, items: items, total_count: total_count, max_index: max_index };",
					"language": "javascript"
				}`),
			},
			{
				TempID:    "block_llm_process",
				Name:      "LLM Process",
				Type:      "llm",
				PositionX: 760,
				PositionY: 40,
				BlockSlug: "llm",
				Config: json.RawMessage(`{
					"provider": "mock",
					"model": "test",
					"user_prompt": "Summarize the following data: {{$.message}}. Total items: {{$.total_count}}",
					"system_prompt": "You are a helpful data summarizer.",
					"temperature": 0.5,
					"max_tokens": 500
				}`),
			},
			{
				TempID:    "block_final_code",
				Name:      "Final Processing",
				Type:      "code",
				PositionX: 880,
				PositionY: 40,
				BlockSlug: "code",
				Config: json.RawMessage(`{
					"code": "const result = { status: 'completed', llm_summary: input.content || 'No summary', total_processed: input.total_count || 0, timestamp: new Date().toISOString() }; return result;"
				}`),
			},
			{
				TempID:    "block_wait",
				Name:      "Brief Wait",
				Type:      "wait",
				PositionX: 1000,
				PositionY: 40,
				BlockSlug: "wait",
				Config: json.RawMessage(`{
					"duration_ms": 100
				}`),
			},
			{
				TempID:    "block_final_output",
				Name:      "Block Demo Output",
				Type:      "function",
				PositionX: 1120,
				PositionY: 40,
				Config: json.RawMessage(`{
					"code": "return { result: input, processed_count: input.total_processed || 0, llm_response: input.llm_summary || '', completed_at: new Date().toISOString() };",
					"language": "javascript"
				}`),
			},
			{
				TempID:    "block_note",
				Name:      "End Note",
				Type:      "note",
				PositionX: 1000,
				PositionY: 160,
				BlockSlug: "note",
				Config: json.RawMessage(`{
					"content": "This workflow demonstrates all core block types",
					"color": "#10B981"
				}`),
			},

			// ============================
			// Data Pipeline Entry Point (横並び: Y=200固定, X増加)
			// Demonstrates: split, filter, map, aggregate
			// ============================
			{
				TempID:      "start_data_pipeline",
				Name:        "Start: Data Pipeline",
				Type:        "start",
				TriggerType: "internal",
				TriggerConfig: json.RawMessage(`{
					"entry_point": "data_pipeline",
					"description": "Data processing pipeline demo"
				}`),
				PositionX: 40,
				PositionY: 200,
				Config: json.RawMessage(`{
					"input_schema": {
						"type": "object",
						"required": ["data"],
						"properties": {
							"data": {
								"type": "array",
								"title": "データ",
								"items": {
									"type": "object",
									"properties": {
										"id": {"type": "integer"},
										"value": {"type": "number"},
										"category": {"type": "string"}
									}
								}
							}
						}
					}
				}`),
			},
			{
				TempID:    "pipeline_split",
				Name:      "Split Data",
				Type:      "split",
				PositionX: 160,
				PositionY: 200,
				BlockSlug: "split",
				Config: json.RawMessage(`{
					"input_path": "data",
					"batch_size": 5
				}`),
			},
			{
				TempID:    "pipeline_filter",
				Name:      "Filter Valid",
				Type:      "filter",
				PositionX: 280,
				PositionY: 200,
				BlockSlug: "filter",
				Config: json.RawMessage(`{
					"expression": "$.value > 0"
				}`),
			},
			{
				TempID:    "pipeline_map",
				Name:      "Map Process",
				Type:      "map",
				PositionX: 400,
				PositionY: 200,
				BlockSlug: "map",
				Config: json.RawMessage(`{
					"input_path": "items",
					"parallel": true,
					"max_workers": 10
				}`),
			},
			{
				TempID:    "pipeline_aggregate",
				Name:      "Aggregate Data",
				Type:      "aggregate",
				PositionX: 520,
				PositionY: 200,
				BlockSlug: "aggregate",
				Config: json.RawMessage(`{
					"operations": [
						{"field": "value", "operation": "sum", "output_field": "total_value"},
						{"field": "value", "operation": "avg", "output_field": "avg_value"},
						{"field": "value", "operation": "count", "output_field": "count"},
						{"field": "value", "operation": "min", "output_field": "min_value"},
						{"field": "value", "operation": "max", "output_field": "max_value"}
					]
				}`),
			},
			{
				TempID:    "pipeline_format",
				Name:      "Format Result",
				Type:      "function",
				PositionX: 640,
				PositionY: 200,
				Config: json.RawMessage(`{
					"code": "return { summary: { total: input.total_value, average: input.avg_value, count: input.count, min: input.min_value, max: input.max_value }, processed_at: new Date().toISOString() };",
					"language": "javascript"
				}`),
			},

			// ============================
			// Block Group Entry Point (横並び: Y=400固定, X増加)
			// Demonstrates: parallel, try_catch, foreach, while block groups
			// ============================
			{
				TempID:      "start_block_group",
				Name:        "Start: Block Group",
				Type:        "start",
				TriggerType: "internal",
				TriggerConfig: json.RawMessage(`{
					"entry_point": "block_group",
					"description": "Demonstrate block group types"
				}`),
				PositionX: 40,
				PositionY: 400,
				Config: json.RawMessage(`{
					"input_schema": {
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
					}
				}`),
			},
			{
				TempID:    "bg_init",
				Name:      "Initialize",
				Type:      "function",
				PositionX: 160,
				PositionY: 400,
				Config: json.RawMessage(`{
					"code": "return { ...input, counter: 0, max_iterations: input.max_iterations || 5, results: [] };",
					"language": "javascript"
				}`),
			},

			// Parallel Group Steps (inside group)
			{
				TempID:           "bg_parallel_branch_a",
				Name:             "Branch A",
				Type:             "function",
				PositionX:        320,
				PositionY:        440,
				BlockGroupTempID: "bg_parallel_group",
				Config: json.RawMessage(`{
					"code": "const items = input.items || []; const half = Math.floor(items.length / 2); return { branch: 'A', processed: items.slice(0, half), count: half };",
					"language": "javascript"
				}`),
			},
			{
				TempID:           "bg_parallel_branch_b",
				Name:             "Branch B",
				Type:             "function",
				PositionX:        320,
				PositionY:        540,
				BlockGroupTempID: "bg_parallel_group",
				Config: json.RawMessage(`{
					"code": "const items = input.items || []; const half = Math.floor(items.length / 2); return { branch: 'B', processed: items.slice(half), count: items.length - half };",
					"language": "javascript"
				}`),
			},
			{
				TempID:           "bg_parallel_branch_c",
				Name:             "Branch C",
				Type:             "function",
				PositionX:        320,
				PositionY:        640,
				BlockGroupTempID: "bg_parallel_group",
				Config: json.RawMessage(`{
					"code": "const items = input.items || []; const values = items.map(i => i.value || 0); const sum = values.reduce((a, b) => a + b, 0); return { branch: 'C', sum: sum, avg: values.length > 0 ? sum / values.length : 0, count: values.length };",
					"language": "javascript"
				}`),
			},

			// Process Parallel Results
			{
				TempID:    "bg_merge_parallel",
				Name:      "Process Results",
				Type:      "function",
				PositionX: 600,
				PositionY: 400,
				Config: json.RawMessage(`{
					"code": "return { ...input, parallel_completed: true };",
					"language": "javascript"
				}`),
			},

			// Parallel Error Handler
			{
				TempID:    "bg_parallel_error",
				Name:      "Parallel Error",
				Type:      "function",
				PositionX: 600,
				PositionY: 760,
				Config: json.RawMessage(`{
					"code": "return { error: 'Parallel processing failed', original_error: input.error || 'Unknown error' };",
					"language": "javascript"
				}`),
			},

			// Try-Catch Group Steps
			{
				TempID:           "bg_try_operation",
				Name:             "Risky Op",
				Type:             "function",
				PositionX:        760,
				PositionY:        440,
				BlockGroupTempID: "bg_try_catch_group",
				Config: json.RawMessage(`{
					"code": "if (input.items && input.items.length === 0) { throw new Error('Empty items array'); } return { success: true, processed: input.items.length };",
					"language": "javascript"
				}`),
			},

			// Catch Handler
			{
				TempID:    "bg_catch_handler",
				Name:      "Catch Handler",
				Type:      "function",
				PositionX: 720,
				PositionY: 600,
				Config: json.RawMessage(`{
					"code": "return { handled: true, message: 'Error was caught and handled', original_error: input.error || 'Unknown' };",
					"language": "javascript"
				}`),
			},

			// ForEach Group Steps
			{
				TempID:           "bg_foreach_process",
				Name:             "Process Item",
				Type:             "function",
				PositionX:        1120,
				PositionY:        440,
				BlockGroupTempID: "bg_foreach_group",
				Config: json.RawMessage(`{
					"code": "const item = input.item || input; return { id: item.id, processed_name: (item.name || 'unknown').toUpperCase(), doubled_value: (item.value || 0) * 2 };",
					"language": "javascript"
				}`),
			},

			// ForEach Error Handler
			{
				TempID:    "bg_foreach_error",
				Name:      "ForEach Error",
				Type:      "function",
				PositionX: 1080,
				PositionY: 600,
				Config: json.RawMessage(`{
					"code": "return { error: 'ForEach iteration failed', item_error: input.error || 'Unknown' };",
					"language": "javascript"
				}`),
			},

			// While Group Steps
			{
				TempID:           "bg_while_increment",
				Name:             "Increment",
				Type:             "function",
				PositionX:        1480,
				PositionY:        440,
				BlockGroupTempID: "bg_while_group",
				Config: json.RawMessage(`{
					"code": "const counter = (input.counter || 0) + 1; return { ...input, counter: counter, iteration_result: 'Iteration ' + counter };",
					"language": "javascript"
				}`),
			},

			// Max Iterations Handler
			{
				TempID:    "bg_max_iterations_handler",
				Name:      "Max Reached",
				Type:      "function",
				PositionX: 1440,
				PositionY: 600,
				Config: json.RawMessage(`{
					"code": "return { warning: 'Maximum iterations reached', final_counter: input.counter || 0 };",
					"language": "javascript"
				}`),
			},

			// Final Output
			{
				TempID:    "bg_final_output",
				Name:      "Block Group Output",
				Type:      "function",
				PositionX: 1760,
				PositionY: 400,
				Config: json.RawMessage(`{
					"code": "return { parallel_completed: input.parallel_completed || false, foreach_results: input.foreach_results || [], while_result: { final_counter: input.counter || 0 }, final_status: 'completed' };",
					"language": "javascript"
				}`),
			},
		},

		Edges: []SystemEdgeDefinition{
			// Block Demo flow
			{SourceTempID: "start_block_demo", TargetTempID: "block_log_input", SourcePort: "output"},
			{SourceTempID: "block_log_input", TargetTempID: "block_prepare_data", SourcePort: "output"},
			{SourceTempID: "block_prepare_data", TargetTempID: "block_filter_items", SourcePort: "output"},
			{SourceTempID: "block_filter_items", TargetTempID: "block_transform_items", SourcePort: "output"},
			{SourceTempID: "block_transform_items", TargetTempID: "block_aggregate", SourcePort: "output"},
			{SourceTempID: "block_aggregate", TargetTempID: "block_llm_process", SourcePort: "output"},
			{SourceTempID: "block_llm_process", TargetTempID: "block_final_code", SourcePort: "output"},
			{SourceTempID: "block_final_code", TargetTempID: "block_wait", SourcePort: "output"},
			{SourceTempID: "block_wait", TargetTempID: "block_final_output", SourcePort: "output"},

			// Data Pipeline flow
			{SourceTempID: "start_data_pipeline", TargetTempID: "pipeline_split", SourcePort: "output"},
			{SourceTempID: "pipeline_split", TargetTempID: "pipeline_filter", SourcePort: "output"},
			{SourceTempID: "pipeline_filter", TargetTempID: "pipeline_map", SourcePort: "matched"},
			{SourceTempID: "pipeline_map", TargetTempID: "pipeline_aggregate", SourcePort: "complete"},
			{SourceTempID: "pipeline_aggregate", TargetTempID: "pipeline_format", SourcePort: "output"},

			// Block Group flow
			{SourceTempID: "start_block_group", TargetTempID: "bg_init", SourcePort: "output"},
			{SourceTempID: "bg_init", TargetGroupTempID: "bg_parallel_group", SourcePort: "output", TargetPort: "group-input"},
			{SourceGroupTempID: "bg_parallel_group", TargetTempID: "bg_merge_parallel", SourcePort: "out"},
			{SourceGroupTempID: "bg_parallel_group", TargetTempID: "bg_parallel_error", SourcePort: "error"},
			{SourceTempID: "bg_merge_parallel", TargetGroupTempID: "bg_try_catch_group", SourcePort: "output", TargetPort: "group-input"},
			{SourceGroupTempID: "bg_try_catch_group", TargetGroupTempID: "bg_foreach_group", SourcePort: "out", TargetPort: "group-input"},
			{SourceGroupTempID: "bg_try_catch_group", TargetTempID: "bg_catch_handler", SourcePort: "error"},
			{SourceGroupTempID: "bg_foreach_group", TargetGroupTempID: "bg_while_group", SourcePort: "out", TargetPort: "group-input"},
			{SourceGroupTempID: "bg_foreach_group", TargetTempID: "bg_foreach_error", SourcePort: "error"},
			{SourceGroupTempID: "bg_while_group", TargetTempID: "bg_final_output", SourcePort: "out"},
			{SourceGroupTempID: "bg_while_group", TargetTempID: "bg_max_iterations_handler", SourcePort: "error"},
		},
	}
}
