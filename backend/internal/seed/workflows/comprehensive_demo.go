package workflows

import "encoding/json"

func (r *Registry) registerComprehensiveWorkflows() {
	r.register(ComprehensiveBlockDemoWorkflow())
}

func (r *Registry) registerDemoWorkflows() {
	r.register(DataPipelineBlockDemoWorkflow())
	// Note: AIRoutingBlockDemoWorkflow and ControlFlowBlockDemoWorkflow were removed
	// because they use branching (router/condition) with multiple output edges outside Block Groups,
	// which is no longer supported. Use Block Groups for parallel/branching flows.
}

// ComprehensiveBlockDemoWorkflow demonstrates all non-external-integration blocks
// This workflow tests core AI, Logic, Control, Data, and Utility blocks
func ComprehensiveBlockDemoWorkflow() *SystemWorkflowDefinition {
	return &SystemWorkflowDefinition{
		ID:          "a0000000-0000-0000-0000-000000000200",
		SystemSlug:  "comprehensive-block-demo",
		Name:        "Comprehensive Block Demo",
		Description: "A workflow demonstrating all non-external-integration blocks including AI, Logic, Control, Data, RAG, and Utility blocks",
		Version:     1,
		IsSystem:    true,
		InputSchema: json.RawMessage(`{
			"type": "object",
			"required": ["message", "items"],
			"properties": {
				"message": {
					"type": "string",
					"title": "メッセージ",
					"description": "処理するメッセージ"
				},
				"items": {
					"type": "array",
					"title": "アイテム",
					"description": "処理対象のアイテム配列",
					"items": {"type": "object"}
				}
			}
		}`),
		OutputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"result": {"type": "object"},
				"processed_count": {"type": "integer"},
				"llm_response": {"type": "string"}
			}
		}`),
		Steps: []SystemStepDefinition{
			// === Control: Start ===
			{
				TempID:    "start",
				Name:      "Start",
				Type:      "start",
				PositionX: 400,
				PositionY: 50,
				Config: json.RawMessage(`{
					"input_schema": {
						"type": "object",
						"required": ["message", "items"],
						"properties": {
							"message": {"type": "string"},
							"items": {"type": "array"}
						}
					}
				}`),
			},
			// === Utility: Log ===
			{
				TempID:    "log_input",
				Name:      "Log Input",
				Type:      "log",
				PositionX: 400,
				PositionY: 150,
				BlockSlug: "log",
				Config: json.RawMessage(`{
					"level": "info",
					"message": "Processing started with {{$.items.length}} items"
				}`),
			},
			// === Utility: Function (data preparation) ===
			{
				TempID:    "prepare_data",
				Name:      "Prepare Data",
				Type:      "function",
				PositionX: 400,
				PositionY: 250,
				Config: json.RawMessage(`{
					"code": "const items = input.items || []; const enhanced = items.map((item, idx) => ({...item, index: idx, processed_at: new Date().toISOString()})); return { ...input, enhanced_items: enhanced, item_count: enhanced.length };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"enhanced_items": {"type": "array"},
							"item_count": {"type": "integer"}
						}
					}
				}`),
			},
			// === Data: Filter (filter items based on criteria) ===
			{
				TempID:    "filter_items",
				Name:      "Filter Items",
				Type:      "filter",
				PositionX: 400,
				PositionY: 350,
				BlockSlug: "filter",
				Config: json.RawMessage(`{
					"expression": "$.index < 10"
				}`),
			},
			// === Data: Map processing ===
			{
				TempID:    "map_items",
				Name:      "Map Items",
				Type:      "map",
				PositionX: 400,
				PositionY: 450,
				BlockSlug: "map",
				Config: json.RawMessage(`{
					"input_path": "items",
					"parallel": false,
					"max_workers": 5
				}`),
			},
			// === Data: Split into batches ===
			{
				TempID:    "split_batches",
				Name:      "Split Batches",
				Type:      "split",
				PositionX: 400,
				PositionY: 550,
				BlockSlug: "split",
				Config: json.RawMessage(`{
					"input_path": "items",
					"batch_size": 3
				}`),
			},
			// === Aggregate results ===
			{
				TempID:    "aggregate_results",
				Name:      "Aggregate Results",
				Type:      "aggregate",
				PositionX: 400,
				PositionY: 650,
				BlockSlug: "aggregate",
				Config: json.RawMessage(`{
					"operations": [
						{"field": "index", "operation": "count", "output_field": "total_count"},
						{"field": "index", "operation": "max", "output_field": "max_index"}
					]
				}`),
			},
			// === AI: LLM for processing ===
			{
				TempID:    "llm_process",
				Name:      "LLM Process",
				Type:      "llm",
				PositionX: 400,
				PositionY: 750,
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
			// === Utility: Code block (final processing) ===
			{
				TempID:    "final_code",
				Name:      "Final Processing",
				Type:      "code",
				PositionX: 400,
				PositionY: 850,
				BlockSlug: "code",
				Config: json.RawMessage(`{
					"code": "const result = { status: 'completed', llm_summary: input.content || 'No summary', total_processed: input.total_count || 0, timestamp: new Date().toISOString() }; return result;",
					"output_schema": {
						"type": "object",
						"properties": {
							"status": {"type": "string"},
							"llm_summary": {"type": "string"},
							"total_processed": {"type": "integer"},
							"timestamp": {"type": "string"}
						}
					}
				}`),
			},
			// === Control: Wait (brief pause before final output) ===
			{
				TempID:    "wait_brief",
				Name:      "Brief Wait",
				Type:      "wait",
				PositionX: 400,
				PositionY: 950,
				BlockSlug: "wait",
				Config: json.RawMessage(`{
					"duration_ms": 100
				}`),
			},
			// === Utility: Note (documentation) ===
			{
				TempID:    "note_end",
				Name:      "End Note",
				Type:      "note",
				PositionX: 100,
				PositionY: 1050,
				BlockSlug: "note",
				Config: json.RawMessage(`{
					"content": "This workflow demonstrates all core block types",
					"color": "#10B981"
				}`),
			},
			// === Final output ===
			{
				TempID:    "final_output",
				Name:      "Final Output",
				Type:      "function",
				PositionX: 400,
				PositionY: 1050,
				Config: json.RawMessage(`{
					"code": "return { result: input, processed_count: input.total_processed || 0, llm_response: input.llm_summary || '', completed_at: new Date().toISOString() };",
					"language": "javascript"
				}`),
			},
		},
		Edges: []SystemEdgeDefinition{
			// Start -> Log
			{SourceTempID: "start", TargetTempID: "log_input", SourcePort: "output"},
			// Log -> Prepare
			{SourceTempID: "log_input", TargetTempID: "prepare_data", SourcePort: "output"},
			// Prepare -> Filter
			{SourceTempID: "prepare_data", TargetTempID: "filter_items", SourcePort: "output"},
			// Filter -> Map
			{SourceTempID: "filter_items", TargetTempID: "map_items", SourcePort: "matched"},
			// Map -> Split
			{SourceTempID: "map_items", TargetTempID: "split_batches", SourcePort: "complete"},
			// Split -> Aggregate
			{SourceTempID: "split_batches", TargetTempID: "aggregate_results", SourcePort: "output"},
			// Aggregate -> LLM
			{SourceTempID: "aggregate_results", TargetTempID: "llm_process", SourcePort: "output"},
			// LLM -> Final Code
			{SourceTempID: "llm_process", TargetTempID: "final_code", SourcePort: "output"},
			// Final Code -> Wait
			{SourceTempID: "final_code", TargetTempID: "wait_brief", SourcePort: "output"},
			// Wait -> Final Output
			{SourceTempID: "wait_brief", TargetTempID: "final_output", SourcePort: "output"},
		},
	}
}

// DataPipelineBlockDemoWorkflow demonstrates data processing blocks
func DataPipelineBlockDemoWorkflow() *SystemWorkflowDefinition {
	return &SystemWorkflowDefinition{
		ID:          "a0000000-0000-0000-0000-000000000201",
		SystemSlug:  "data-pipeline-block-demo",
		Name:        "Data Pipeline Block Demo",
		Description: "Demonstrates data processing blocks: split, filter, map, and aggregate",
		Version:     1,
		IsSystem:    true,
		InputSchema: json.RawMessage(`{
			"type": "object",
			"required": ["data"],
			"properties": {
				"data": {
					"type": "array",
					"title": "データ",
					"description": "処理対象のデータ配列",
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
		}`),
		Steps: []SystemStepDefinition{
			{
				TempID:    "start",
				Name:      "Start",
				Type:      "start",
				PositionX: 400,
				PositionY: 50,
				Config:    json.RawMessage(`{}`),
			},
			// Split into batches
			{
				TempID:    "split_data",
				Name:      "Split Data",
				Type:      "split",
				PositionX: 400,
				PositionY: 150,
				BlockSlug: "split",
				Config: json.RawMessage(`{
					"input_path": "data",
					"batch_size": 5
				}`),
			},
			// Filter only valid items
			{
				TempID:    "filter_valid",
				Name:      "Filter Valid",
				Type:      "filter",
				PositionX: 400,
				PositionY: 250,
				BlockSlug: "filter",
				Config: json.RawMessage(`{
					"expression": "$.value > 0"
				}`),
			},
			// Map to process each item
			{
				TempID:    "map_process",
				Name:      "Map Process",
				Type:      "map",
				PositionX: 400,
				PositionY: 350,
				BlockSlug: "map",
				Config: json.RawMessage(`{
					"input_path": "items",
					"parallel": true,
					"max_workers": 10
				}`),
			},
			// Aggregate results
			{
				TempID:    "aggregate_data",
				Name:      "Aggregate Data",
				Type:      "aggregate",
				PositionX: 400,
				PositionY: 450,
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
			// Final function
			{
				TempID:    "format_result",
				Name:      "Format Result",
				Type:      "function",
				PositionX: 400,
				PositionY: 550,
				Config: json.RawMessage(`{
					"code": "return { summary: { total: input.total_value, average: input.avg_value, count: input.count, min: input.min_value, max: input.max_value }, processed_at: new Date().toISOString() };",
					"language": "javascript"
				}`),
			},
		},
		Edges: []SystemEdgeDefinition{
			{SourceTempID: "start", TargetTempID: "split_data", SourcePort: "output"},
			{SourceTempID: "split_data", TargetTempID: "filter_valid", SourcePort: "output"},
			{SourceTempID: "filter_valid", TargetTempID: "map_process", SourcePort: "matched"},
			{SourceTempID: "map_process", TargetTempID: "aggregate_data", SourcePort: "complete"},
			{SourceTempID: "aggregate_data", TargetTempID: "format_result", SourcePort: "output"},
		},
	}
}

// AIRoutingBlockDemoWorkflow demonstrates AI routing blocks
func AIRoutingBlockDemoWorkflow() *SystemWorkflowDefinition {
	return &SystemWorkflowDefinition{
		ID:          "a0000000-0000-0000-0000-000000000202",
		SystemSlug:  "ai-routing-block-demo",
		Name:        "AI Routing Block Demo",
		Description: "Demonstrates AI blocks: router for dynamic routing based on content",
		Version:     1,
		IsSystem:    true,
		InputSchema: json.RawMessage(`{
			"type": "object",
			"required": ["query"],
			"properties": {
				"query": {
					"type": "string",
					"title": "クエリ",
					"description": "ルーティング対象のクエリ"
				}
			}
		}`),
		Steps: []SystemStepDefinition{
			{
				TempID:    "start",
				Name:      "Start",
				Type:      "start",
				PositionX: 400,
				PositionY: 50,
				Config:    json.RawMessage(`{}`),
			},
			// AI Router
			{
				TempID:    "ai_router",
				Name:      "AI Router",
				Type:      "router",
				PositionX: 400,
				PositionY: 150,
				BlockSlug: "router",
				Config: json.RawMessage(`{
					"provider": "mock",
					"model": "test",
					"routes": [
						{"name": "technical", "description": "Technical questions about code, APIs, or systems"},
						{"name": "general", "description": "General knowledge questions"},
						{"name": "creative", "description": "Creative writing or brainstorming requests"}
					]
				}`),
			},
			// Technical route
			{
				TempID:    "technical_llm",
				Name:      "Technical LLM",
				Type:      "llm",
				PositionX: 200,
				PositionY: 300,
				BlockSlug: "llm",
				Config: json.RawMessage(`{
					"provider": "mock",
					"model": "test",
					"system_prompt": "You are a technical expert. Provide detailed technical explanations.",
					"user_prompt": "{{$.query}}",
					"temperature": 0.3
				}`),
			},
			// General route
			{
				TempID:    "general_llm",
				Name:      "General LLM",
				Type:      "llm",
				PositionX: 400,
				PositionY: 300,
				BlockSlug: "llm",
				Config: json.RawMessage(`{
					"provider": "mock",
					"model": "test",
					"system_prompt": "You are a helpful assistant. Provide clear and informative answers.",
					"user_prompt": "{{$.query}}",
					"temperature": 0.5
				}`),
			},
			// Creative route
			{
				TempID:    "creative_llm",
				Name:      "Creative LLM",
				Type:      "llm",
				PositionX: 600,
				PositionY: 300,
				BlockSlug: "llm",
				Config: json.RawMessage(`{
					"provider": "mock",
					"model": "test",
					"system_prompt": "You are a creative writer. Be imaginative and engaging.",
					"user_prompt": "{{$.query}}",
					"temperature": 0.9
				}`),
			},
			// Join results
			{
				TempID:    "join_routes",
				Name:      "Join Routes",
				Type:      "join",
				PositionX: 400,
				PositionY: 450,
				BlockSlug: "join",
				Config:    json.RawMessage(`{}`),
			},
			// Format output
			{
				TempID:    "format_output",
				Name:      "Format Output",
				Type:      "function",
				PositionX: 400,
				PositionY: 550,
				Config: json.RawMessage(`{
					"code": "return { response: input.content, route: input.__branch || 'unknown', query: input.query };",
					"language": "javascript"
				}`),
			},
		},
		Edges: []SystemEdgeDefinition{
			{SourceTempID: "start", TargetTempID: "ai_router", SourcePort: "output"},
			{SourceTempID: "ai_router", TargetTempID: "technical_llm", SourcePort: "technical"},
			{SourceTempID: "ai_router", TargetTempID: "general_llm", SourcePort: "general"},
			{SourceTempID: "ai_router", TargetTempID: "creative_llm", SourcePort: "creative"},
			// Note: "default" port is handled by general_llm via the "general" edge
			// The router block falls back to general when no specific route matches
			{SourceTempID: "technical_llm", TargetTempID: "join_routes", SourcePort: "output"},
			{SourceTempID: "general_llm", TargetTempID: "join_routes", SourcePort: "output"},
			{SourceTempID: "creative_llm", TargetTempID: "join_routes", SourcePort: "output"},
			{SourceTempID: "join_routes", TargetTempID: "format_output", SourcePort: "output"},
		},
	}
}

// ControlFlowBlockDemoWorkflow demonstrates control flow blocks
func ControlFlowBlockDemoWorkflow() *SystemWorkflowDefinition {
	return &SystemWorkflowDefinition{
		ID:          "a0000000-0000-0000-0000-000000000203",
		SystemSlug:  "control-flow-block-demo",
		Name:        "Control Flow Block Demo",
		Description: "Demonstrates control flow blocks: condition, switch, loop, wait, and human_in_loop",
		Version:     1,
		IsSystem:    true,
		InputSchema: json.RawMessage(`{
			"type": "object",
			"required": ["count", "require_approval"],
			"properties": {
				"count": {
					"type": "integer",
					"title": "カウント",
					"description": "ループ回数"
				},
				"require_approval": {
					"type": "boolean",
					"title": "承認必要",
					"description": "人間の承認を必要とするか"
				}
			}
		}`),
		Steps: []SystemStepDefinition{
			{
				TempID:    "start",
				Name:      "Start",
				Type:      "start",
				PositionX: 400,
				PositionY: 50,
				Config:    json.RawMessage(`{}`),
			},
			// Condition check
			{
				TempID:    "check_approval",
				Name:      "Check Approval Required",
				Type:      "condition",
				PositionX: 400,
				PositionY: 150,
				BlockSlug: "condition",
				Config: json.RawMessage(`{
					"expression": "$.require_approval === true"
				}`),
			},
			// Human in loop (if approval required)
			{
				TempID:    "human_approval",
				Name:      "Human Approval",
				Type:      "human_in_loop",
				PositionX: 200,
				PositionY: 250,
				BlockSlug: "human_in_loop",
				Config: json.RawMessage(`{
					"instructions": "Please review and approve the following operation",
					"timeout_hours": 24
				}`),
			},
			// Wait (if no approval required)
			{
				TempID:    "wait_step",
				Name:      "Wait",
				Type:      "wait",
				PositionX: 600,
				PositionY: 250,
				BlockSlug: "wait",
				Config: json.RawMessage(`{
					"duration_ms": 500
				}`),
			},
			// Join paths
			{
				TempID:    "join_approval",
				Name:      "Join Approval",
				Type:      "join",
				PositionX: 400,
				PositionY: 350,
				BlockSlug: "join",
				Config:    json.RawMessage(`{}`),
			},
			// Final output
			{
				TempID:    "final_result",
				Name:      "Final Result",
				Type:      "function",
				PositionX: 400,
				PositionY: 450,
				Config: json.RawMessage(`{
					"code": "return { message: input.message, require_approval: input.require_approval, completed: true };",
					"language": "javascript"
				}`),
			},
		},
		Edges: []SystemEdgeDefinition{
			{SourceTempID: "start", TargetTempID: "check_approval", SourcePort: "output"},
			{SourceTempID: "check_approval", TargetTempID: "human_approval", SourcePort: "true"},
			{SourceTempID: "check_approval", TargetTempID: "wait_step", SourcePort: "false"},
			{SourceTempID: "human_approval", TargetTempID: "join_approval", SourcePort: "approved"},
			{SourceTempID: "wait_step", TargetTempID: "join_approval", SourcePort: "output"},
			{SourceTempID: "join_approval", TargetTempID: "final_result", SourcePort: "output"},
		},
	}
}
