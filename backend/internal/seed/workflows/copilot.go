package workflows

import "encoding/json"

func (r *Registry) registerCopilotWorkflows() {
	r.register(CopilotWorkflow())
}

// CopilotWorkflow is a unified Copilot workflow with 4 entry points:
// - generate: Generates workflow structures from natural language
// - suggest: Suggests next steps for a workflow
// - diagnose: Diagnoses workflow execution errors
// - optimize: Suggests optimizations for workflow performance
func CopilotWorkflow() *SystemWorkflowDefinition {
	return &SystemWorkflowDefinition{
		ID:          "a0000000-0000-0000-0000-000000000001",
		SystemSlug:  "copilot",
		Name:        "Copilot Workflows",
		Description: "AI-assisted workflow building with multiple entry points: generate, suggest, diagnose, optimize",
		Version:     1,
		IsSystem:    true,
		Steps: []SystemStepDefinition{
			// ============================
			// Generate Entry Point (横並び: Y=40固定, X増加)
			// ============================
			{
				TempID:      "start_generate",
				Name:        "Start: Generate",
				Type:        "start",
				TriggerType: "internal",
				TriggerConfig: json.RawMessage(`{
					"entry_point": "generate",
					"description": "Generate workflow from natural language"
				}`),
				PositionX: 40,
				PositionY: 40,
				Config: json.RawMessage(`{
					"input_schema": {
						"type": "object",
						"required": ["prompt"],
						"properties": {
							"prompt": {
								"type": "string",
								"title": "説明",
								"description": "生成したいワークフローの説明を入力してください"
							}
						}
					}
				}`),
			},
			{
				TempID:    "generate_get_blocks",
				Name:      "Get Available Blocks",
				Type:      "function",
				PositionX: 160,
				PositionY: 40,
				Config: json.RawMessage(`{
					"code": "const blocks = context.blocks.list(); return { blocks: blocks.map(b => ({ slug: b.slug, name: b.name, description: b.description, category: b.category })) };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"blocks": {"type": "array", "title": "ブロック一覧", "description": "利用可能なブロックの配列"}
						},
						"required": ["blocks"]
					}
				}`),
			},
			{
				TempID:    "generate_build_prompt",
				Name:      "Build Prompt",
				Type:      "function",
				PositionX: 280,
				PositionY: 40,
				Config: json.RawMessage(`{
					"code": "const blocksInfo = input.blocks.map(b => ` + "`" + `- ${b.slug}: ${b.name} (${b.category}) - ${b.description || \"\"}` + "`" + `).join(\"\\n\");\nconst prompt = ` + "`" + `You are an AI workflow generator. Generate a workflow based on the user description.\n\n## Available Blocks\n${blocksInfo}\n\n## Available Step Types\n- start: Entry point (required)\n- llm: AI/LLM call\n- tool: External adapter\n- condition: Binary branch (true/false)\n- switch: Multi-way branch\n- map: Parallel array processing\n- loop: Iteration\n- wait: Delay\n- function: Custom JavaScript\n- log: Debug logging\n\n## User Request\n${input.prompt}\n\n## Output Format (JSON)\n{\n  \"response\": \"Explanation\",\n  \"steps\": [{\"temp_id\": \"step_1\", \"name\": \"Step Name\", \"type\": \"start\", \"description\": \"\", \"config\": {}, \"position_x\": 400, \"position_y\": 50}],\n  \"edges\": [{\"source_temp_id\": \"step_1\", \"target_temp_id\": \"step_2\", \"source_port\": \"default\"}],\n  \"start_step_id\": \"step_1\"\n}\n\nGenerate a valid workflow JSON. Always include a start step.` + "`" + `;\nreturn { prompt: prompt };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"prompt": {"type": "string", "title": "プロンプト", "description": "LLMに送信するプロンプト"}
						},
						"required": ["prompt"]
					}
				}`),
			},
			{
				TempID:    "generate_llm",
				Name:      "Generate with LLM",
				Type:      "llm",
				PositionX: 400,
				PositionY: 40,
				Config: json.RawMessage(`{
					"model": "gpt-4o-mini",
					"provider": "openai",
					"max_tokens": 4000,
					"temperature": 0.3,
					"user_prompt": "{{$.prompt}}",
					"system_prompt": "You are an AI workflow generator. Always respond with valid JSON."
				}`),
			},
			{
				TempID:    "generate_parse",
				Name:      "Parse & Validate",
				Type:      "function",
				PositionX: 520,
				PositionY: 40,
				Config: json.RawMessage(`{
					"code": "try { let content = input.content || \"\"; if (content.startsWith(\"` + "```json" + `\")) content = content.slice(7); if (content.startsWith(\"` + "```" + `\")) content = content.slice(3); if (content.endsWith(\"` + "```" + `\")) content = content.slice(0, -3); content = content.trim(); const result = JSON.parse(content); if (!result.steps || !Array.isArray(result.steps)) { return { error: \"Invalid workflow: missing steps array\" }; } const validTypes = [\"start\", \"llm\", \"tool\", \"condition\", \"switch\", \"map\", \"join\", \"subflow\", \"loop\", \"wait\", \"function\", \"router\", \"human_in_loop\", \"filter\", \"split\", \"aggregate\", \"error\", \"note\", \"log\"]; result.steps = result.steps.filter(s => validTypes.includes(s.type)); return result; } catch (e) { return { error: \"Failed to parse LLM response: \" + e.message }; }",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"response": {"type": "string", "title": "レスポンス"},
							"steps": {"type": "array", "title": "ステップ", "description": "生成されたステップの配列"},
							"edges": {"type": "array", "title": "エッジ", "description": "ステップ間の接続"},
							"start_step_id": {"type": "string", "title": "開始ステップID"},
							"error": {"type": "string", "title": "エラー"}
						}
					}
				}`),
			},

			// ============================
			// Suggest Entry Point (横並び: Y=160固定, X増加)
			// ============================
			{
				TempID:      "start_suggest",
				Name:        "Start: Suggest",
				Type:        "start",
				TriggerType: "internal",
				TriggerConfig: json.RawMessage(`{
					"entry_point": "suggest",
					"description": "Suggest next steps for a workflow"
				}`),
				PositionX: 40,
				PositionY: 160,
				Config: json.RawMessage(`{
					"input_schema": {
						"type": "object",
						"required": ["workflow_id"],
						"properties": {
							"workflow_id": {"type": "string", "title": "ワークフローID", "description": "ステップを提案するワークフローのID"},
							"context": {"type": "string", "title": "コンテキスト", "description": "追加のコンテキスト情報（オプション）"}
						}
					}
				}`),
			},
			{
				TempID:    "suggest_get_context",
				Name:      "Get Workflow Context",
				Type:      "function",
				PositionX: 160,
				PositionY: 160,
				Config: json.RawMessage(`{
					"code": "const workflow = context.workflows.get(input.workflow_id); const blocks = context.blocks.list(); return { workflow: workflow, blocks: blocks };",
					"language": "javascript"
				}`),
			},
			{
				TempID:    "suggest_build_prompt",
				Name:      "Build Suggest Prompt",
				Type:      "function",
				PositionX: 280,
				PositionY: 160,
				Config: json.RawMessage(`{
					"code": "const wf = input.workflow; const blocksInfo = input.blocks.slice(0, 20).map(b => ` + "`" + `- ${b.slug}: ${b.name}` + "`" + `).join(\"\\n\"); const stepsInfo = (wf.steps || []).map(s => ` + "`" + `- ${s.name} (${s.type})` + "`" + `).join(\"\\n\"); const prompt = ` + "`" + `Suggest 2-3 next steps for this workflow.\n\n## Current Steps\n${stepsInfo || \"(empty)\"}\n\n## Available Blocks\n${blocksInfo}\n\n## Context\n${input.context || \"\"}\n\nReturn JSON array: [{\"type\": \"...\", \"name\": \"...\", \"description\": \"...\", \"config\": {}, \"reason\": \"...\"}]` + "`" + `; return { prompt: prompt };",
					"language": "javascript"
				}`),
			},
			{
				TempID:    "suggest_llm",
				Name:      "Suggest with LLM",
				Type:      "llm",
				PositionX: 400,
				PositionY: 160,
				Config: json.RawMessage(`{
					"model": "gpt-4o-mini",
					"provider": "openai",
					"max_tokens": 2000,
					"temperature": 0.5,
					"user_prompt": "{{$.prompt}}",
					"system_prompt": "You are an AI workflow assistant. Return valid JSON array."
				}`),
			},
			{
				TempID:    "suggest_parse",
				Name:      "Parse Suggestions",
				Type:      "function",
				PositionX: 520,
				PositionY: 160,
				Config: json.RawMessage(`{
					"code": "try { let content = input.content || \"\"; if (content.startsWith(\"` + "```" + `\")) { content = content.replace(/` + "```json?\\n?" + `/g, \"\").replace(/` + "```" + `/g, \"\").trim(); } const suggestions = JSON.parse(content); return { suggestions: Array.isArray(suggestions) ? suggestions : [] }; } catch (e) { return { suggestions: [] }; }",
					"language": "javascript"
				}`),
			},

			// ============================
			// Diagnose Entry Point (横並び: Y=280固定, X増加)
			// ============================
			{
				TempID:      "start_diagnose",
				Name:        "Start: Diagnose",
				Type:        "start",
				TriggerType: "internal",
				TriggerConfig: json.RawMessage(`{
					"entry_point": "diagnose",
					"description": "Diagnose workflow execution errors"
				}`),
				PositionX: 40,
				PositionY: 280,
				Config: json.RawMessage(`{
					"input_schema": {
						"type": "object",
						"required": ["run_id"],
						"properties": {
							"run_id": {"type": "string", "title": "実行ID", "description": "診断する実行のID"}
						}
					}
				}`),
			},
			{
				TempID:    "diagnose_get_run",
				Name:      "Get Run Details",
				Type:      "function",
				PositionX: 160,
				PositionY: 280,
				Config: json.RawMessage(`{
					"code": "const run = context.runs.get(input.run_id); const stepRuns = context.runs.getStepRuns(input.run_id); const failedSteps = stepRuns.filter(sr => sr.status === \"failed\"); return { run: run, stepRuns: stepRuns, failedSteps: failedSteps };",
					"language": "javascript"
				}`),
			},
			{
				TempID:    "diagnose_build_prompt",
				Name:      "Build Diagnose Prompt",
				Type:      "function",
				PositionX: 280,
				PositionY: 280,
				Config: json.RawMessage(`{
					"code": "const failedInfo = input.failedSteps.map(sr => ` + "`" + `Step: ${sr.step_name}\nError: ${sr.error || \"Unknown\"}\nInput: ${JSON.stringify(sr.input || {})}` + "`" + `).join(\"\\n\\n\"); const prompt = ` + "`" + `Diagnose this workflow error.\n\n## Run Status: ${input.run.status}\n\n## Failed Steps\n${failedInfo || \"No failures found\"}\n\nReturn JSON: {\"diagnosis\": {\"root_cause\": \"...\", \"category\": \"config_error|input_error|api_error|logic_error|timeout|unknown\", \"severity\": \"high|medium|low\"}, \"fixes\": [{\"description\": \"...\", \"steps\": [\"...\"]}], \"preventions\": [\"...\"]}` + "`" + `; return { prompt: prompt };",
					"language": "javascript"
				}`),
			},
			{
				TempID:    "diagnose_llm",
				Name:      "Diagnose with LLM",
				Type:      "llm",
				PositionX: 400,
				PositionY: 280,
				Config: json.RawMessage(`{
					"model": "gpt-4o-mini",
					"provider": "openai",
					"max_tokens": 2000,
					"temperature": 0.3,
					"user_prompt": "{{$.prompt}}",
					"system_prompt": "You are an AI debugging assistant. Return valid JSON."
				}`),
			},
			{
				TempID:    "diagnose_parse",
				Name:      "Parse Diagnosis",
				Type:      "function",
				PositionX: 520,
				PositionY: 280,
				Config: json.RawMessage(`{
					"code": "try { let content = input.content || \"\"; if (content.startsWith(\"` + "```" + `\")) { content = content.replace(/` + "```json?\\n?" + `/g, \"\").replace(/` + "```" + `/g, \"\").trim(); } return JSON.parse(content); } catch (e) { return { diagnosis: { root_cause: \"Parse error\", category: \"unknown\", severity: \"low\" }, fixes: [], preventions: [] }; }",
					"language": "javascript"
				}`),
			},

			// ============================
			// Optimize Entry Point (横並び: Y=400固定, X増加)
			// ============================
			{
				TempID:      "start_optimize",
				Name:        "Start: Optimize",
				Type:        "start",
				TriggerType: "internal",
				TriggerConfig: json.RawMessage(`{
					"entry_point": "optimize",
					"description": "Suggest optimizations for workflow performance"
				}`),
				PositionX: 40,
				PositionY: 400,
				Config: json.RawMessage(`{
					"input_schema": {
						"type": "object",
						"required": ["workflow_id"],
						"properties": {
							"workflow_id": {"type": "string", "title": "ワークフローID", "description": "最適化するワークフローのID"}
						}
					}
				}`),
			},
			{
				TempID:    "optimize_get_workflow",
				Name:      "Get Workflow Details",
				Type:      "function",
				PositionX: 160,
				PositionY: 400,
				Config: json.RawMessage(`{
					"code": "const workflow = context.workflows.get(input.workflow_id); return { workflow: workflow };",
					"language": "javascript"
				}`),
			},
			{
				TempID:    "optimize_build_prompt",
				Name:      "Build Optimize Prompt",
				Type:      "function",
				PositionX: 280,
				PositionY: 400,
				Config: json.RawMessage(`{
					"code": "const wf = input.workflow; const stepsInfo = (wf.steps || []).map(s => ` + "`" + `- ${s.name} (${s.type}): ${JSON.stringify(s.config || {})}` + "`" + `).join(\"\\n\"); const prompt = ` + "`" + `Suggest optimizations for this workflow.\n\n## Workflow: ${wf.name}\n## Steps (${(wf.steps || []).length})\n${stepsInfo}\n\nReturn JSON: {\"optimizations\": [{\"category\": \"performance|cost|reliability|maintainability\", \"title\": \"...\", \"description\": \"...\", \"impact\": \"high|medium|low\", \"effort\": \"high|medium|low\"}], \"summary\": \"...\"}` + "`" + `; return { prompt: prompt };",
					"language": "javascript"
				}`),
			},
			{
				TempID:    "optimize_llm",
				Name:      "Optimize with LLM",
				Type:      "llm",
				PositionX: 400,
				PositionY: 400,
				Config: json.RawMessage(`{
					"model": "gpt-4o-mini",
					"provider": "openai",
					"max_tokens": 2000,
					"temperature": 0.5,
					"user_prompt": "{{$.prompt}}",
					"system_prompt": "You are an AI optimization assistant. Return valid JSON."
				}`),
			},
			{
				TempID:    "optimize_parse",
				Name:      "Parse Optimizations",
				Type:      "function",
				PositionX: 520,
				PositionY: 400,
				Config: json.RawMessage(`{
					"code": "try { let content = input.content || \"\"; if (content.startsWith(\"` + "```" + `\")) { content = content.replace(/` + "```json?\\n?" + `/g, \"\").replace(/` + "```" + `/g, \"\").trim(); } return JSON.parse(content); } catch (e) { return { optimizations: [], summary: \"Parse error\" }; }",
					"language": "javascript"
				}`),
			},
		},
		Edges: []SystemEdgeDefinition{
			// Generate flow
			{SourceTempID: "start_generate", TargetTempID: "generate_get_blocks", SourcePort: "output"},
			{SourceTempID: "generate_get_blocks", TargetTempID: "generate_build_prompt", SourcePort: "output"},
			{SourceTempID: "generate_build_prompt", TargetTempID: "generate_llm", SourcePort: "output"},
			{SourceTempID: "generate_llm", TargetTempID: "generate_parse", SourcePort: "output"},

			// Suggest flow
			{SourceTempID: "start_suggest", TargetTempID: "suggest_get_context", SourcePort: "output"},
			{SourceTempID: "suggest_get_context", TargetTempID: "suggest_build_prompt", SourcePort: "output"},
			{SourceTempID: "suggest_build_prompt", TargetTempID: "suggest_llm", SourcePort: "output"},
			{SourceTempID: "suggest_llm", TargetTempID: "suggest_parse", SourcePort: "output"},

			// Diagnose flow
			{SourceTempID: "start_diagnose", TargetTempID: "diagnose_get_run", SourcePort: "output"},
			{SourceTempID: "diagnose_get_run", TargetTempID: "diagnose_build_prompt", SourcePort: "output"},
			{SourceTempID: "diagnose_build_prompt", TargetTempID: "diagnose_llm", SourcePort: "output"},
			{SourceTempID: "diagnose_llm", TargetTempID: "diagnose_parse", SourcePort: "output"},

			// Optimize flow
			{SourceTempID: "start_optimize", TargetTempID: "optimize_get_workflow", SourcePort: "output"},
			{SourceTempID: "optimize_get_workflow", TargetTempID: "optimize_build_prompt", SourcePort: "output"},
			{SourceTempID: "optimize_build_prompt", TargetTempID: "optimize_llm", SourcePort: "output"},
			{SourceTempID: "optimize_llm", TargetTempID: "optimize_parse", SourcePort: "output"},
		},
	}
}
