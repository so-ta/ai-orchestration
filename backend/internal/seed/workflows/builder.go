package workflows

import "encoding/json"

func (r *Registry) registerBuilderWorkflows() {
	r.register(BuilderWorkflow())
}

// BuilderWorkflow is a unified AI Workflow Builder with 4 entry points:
// - analysis: AI thinks deeply and proposes assumptions + clarifying questions
// - proposal: Update assumptions based on user answers and finalize spec
// - construct: Builds workflow from gathered requirements (WorkflowSpec)
// - refine: Refines existing workflow based on user feedback
//
// This workflow uses llm-structured blocks to minimize boilerplate:
// - Automatic JSON output parsing (inherited from llm-json)
// - Schema-driven structured output with validation
// - Eliminates separate build_prompt and parse steps
func BuilderWorkflow() *SystemWorkflowDefinition {
	// Shared output schema for Analysis phase
	analysisOutputSchema := map[string]interface{}{
		"type": "object",
		"required": []string{"response", "proposed_spec", "assumptions", "nextPhase", "progress"},
		"properties": map[string]interface{}{
			"response":          map[string]interface{}{"type": "string", "title": "応答", "description": "ユーザーへの応答メッセージ"},
			"thinking":          map[string]interface{}{"type": "string", "title": "思考プロセス", "description": "AIの内部思考（ユーザーに表示）"},
			"proposed_spec":     map[string]interface{}{"type": "object", "title": "提案仕様", "description": "提案するワークフロー仕様"},
			"assumptions":       map[string]interface{}{"type": "array", "title": "仮定", "description": "行った仮定のリスト"},
			"clarifying_points": map[string]interface{}{"type": "array", "title": "確認事項", "description": "確認が必要な事項（0-3個まで）"},
			"nextPhase":         map[string]interface{}{"type": "string", "title": "次のフェーズ", "description": "analysis|proposal|completed"},
			"progress":          map[string]interface{}{"type": "number", "title": "進捗", "description": "0-100の進捗率"},
		},
	}

	// Shared output schema for Proposal phase
	proposalOutputSchema := map[string]interface{}{
		"type": "object",
		"required": []string{"response", "final_spec", "assumptions", "nextPhase", "progress"},
		"properties": map[string]interface{}{
			"response":    map[string]interface{}{"type": "string", "title": "応答", "description": "ユーザーへの応答メッセージ"},
			"final_spec":  map[string]interface{}{"type": "object", "title": "最終仕様", "description": "確定したワークフロー仕様"},
			"assumptions": map[string]interface{}{"type": "array", "title": "仮定", "description": "確認された仮定のリスト"},
			"nextPhase":   map[string]interface{}{"type": "string", "title": "次のフェーズ", "description": "proposal|completed"},
			"progress":    map[string]interface{}{"type": "number", "title": "進捗", "description": "0-100の進捗率"},
		},
	}

	// Shared output schema for Construct phase
	constructOutputSchema := map[string]interface{}{
		"type": "object",
		"required": []string{"workflow_name", "steps", "edges"},
		"properties": map[string]interface{}{
			"workflow_name":            map[string]interface{}{"type": "string", "title": "ワークフロー名"},
			"workflow_description":     map[string]interface{}{"type": "string", "title": "説明"},
			"natural_language_summary": map[string]interface{}{"type": "string", "title": "自然言語要約"},
			"steps": map[string]interface{}{
				"type":  "array",
				"title": "ステップ",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"temp_id":            map[string]interface{}{"type": "string"},
						"name":               map[string]interface{}{"type": "string"},
						"type":               map[string]interface{}{"type": "string"},
						"description":        map[string]interface{}{"type": "string"},
						"config":             map[string]interface{}{"type": "object"},
						"block_slug":         map[string]interface{}{"type": "string"},
						"mapping_status":     map[string]interface{}{"type": "string"},
						"mapping_confidence": map[string]interface{}{"type": "string"},
						"custom_required":    map[string]interface{}{"type": "boolean"},
						"custom_reason":      map[string]interface{}{"type": "string"},
						"executor":           map[string]interface{}{"type": "string"},
					},
				},
			},
			"edges": map[string]interface{}{
				"type":  "array",
				"title": "エッジ",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"source_temp_id": map[string]interface{}{"type": "string"},
						"target_temp_id": map[string]interface{}{"type": "string"},
						"source_port":    map[string]interface{}{"type": "string"},
						"target_port":    map[string]interface{}{"type": "string"},
						"label":          map[string]interface{}{"type": "string"},
					},
				},
			},
			"start_step_id":   map[string]interface{}{"type": "string", "title": "開始ステップID"},
			"summary":         map[string]interface{}{"type": "object", "title": "サマリー"},
			"editable_points": map[string]interface{}{"type": "array", "title": "編集可能ポイント"},
			"warnings":        map[string]interface{}{"type": "array", "title": "警告"},
		},
	}

	// Shared output schema for Refine phase
	refineOutputSchema := map[string]interface{}{
		"type": "object",
		"required": []string{"changes", "response"},
		"properties": map[string]interface{}{
			"changes": map[string]interface{}{
				"type":  "array",
				"title": "変更",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"type":    map[string]interface{}{"type": "string"},
						"step_id": map[string]interface{}{"type": "string"},
						"step":    map[string]interface{}{"type": "object"},
						"edge":    map[string]interface{}{"type": "object"},
						"config":  map[string]interface{}{"type": "object"},
						"reason":  map[string]interface{}{"type": "string"},
					},
				},
			},
			"summary":  map[string]interface{}{"type": "string", "title": "変更サマリー"},
			"response": map[string]interface{}{"type": "string", "title": "応答メッセージ"},
		},
	}

	// Agent requirement analysis output schema
	agentRequirementSchema := map[string]interface{}{
		"type": "object",
		"required": []string{"clarified_spec", "ready_for_design"},
		"properties": map[string]interface{}{
			"clarified_spec":   map[string]interface{}{"type": "object", "title": "明確化された仕様"},
			"assumptions":      map[string]interface{}{"type": "array", "title": "仮定"},
			"missing_info":     map[string]interface{}{"type": "array", "title": "不足情報"},
			"ready_for_design": map[string]interface{}{"type": "boolean", "title": "設計準備完了"},
		},
	}

	// Agent structure design output schema
	agentStructureSchema := map[string]interface{}{
		"type": "object",
		"required": []string{"workflow_name", "step_skeletons", "edges"},
		"properties": map[string]interface{}{
			"workflow_name":        map[string]interface{}{"type": "string", "title": "ワークフロー名"},
			"workflow_description": map[string]interface{}{"type": "string", "title": "説明"},
			"step_skeletons": map[string]interface{}{
				"type":  "array",
				"title": "ステップスケルトン",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"temp_id":      map[string]interface{}{"type": "string"},
						"name":         map[string]interface{}{"type": "string"},
						"description":  map[string]interface{}{"type": "string"},
						"type":         map[string]interface{}{"type": "string"},
						"block_slug":   map[string]interface{}{"type": "string"},
						"purpose":      map[string]interface{}{"type": "string"},
						"needs_config": map[string]interface{}{"type": "boolean"},
						"depends_on":   map[string]interface{}{"type": "array"},
					},
				},
			},
			"edges": map[string]interface{}{
				"type":  "array",
				"title": "エッジ",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"source": map[string]interface{}{"type": "string"},
						"target": map[string]interface{}{"type": "string"},
						"label":  map[string]interface{}{"type": "string"},
					},
				},
			},
			"design_notes": map[string]interface{}{"type": "string", "title": "設計メモ"},
		},
	}

	// Agent step config output schema
	agentConfigSchema := map[string]interface{}{
		"type": "object",
		"required": []string{"config"},
		"properties": map[string]interface{}{
			"config":        map[string]interface{}{"type": "object", "title": "設定"},
			"output_fields": map[string]interface{}{"type": "array", "title": "出力フィールド"},
			"reasoning":     map[string]interface{}{"type": "string", "title": "理由"},
		},
	}

	// Agent validation output schema
	agentValidationSchema := map[string]interface{}{
		"type": "object",
		"required": []string{"is_valid", "final_steps", "final_edges"},
		"properties": map[string]interface{}{
			"is_valid":      map[string]interface{}{"type": "boolean", "title": "有効"},
			"issues":        map[string]interface{}{"type": "array", "title": "問題"},
			"fixes_applied": map[string]interface{}{"type": "array", "title": "適用された修正"},
			"final_steps":   map[string]interface{}{"type": "array", "title": "最終ステップ"},
			"final_edges":   map[string]interface{}{"type": "array", "title": "最終エッジ"},
			"summary":       map[string]interface{}{"type": "string", "title": "サマリー"},
		},
	}

	// Convert schemas to JSON
	analysisSchemaJSON, _ := json.Marshal(analysisOutputSchema)
	proposalSchemaJSON, _ := json.Marshal(proposalOutputSchema)
	constructSchemaJSON, _ := json.Marshal(constructOutputSchema)
	refineSchemaJSON, _ := json.Marshal(refineOutputSchema)
	agentRequirementSchemaJSON, _ := json.Marshal(agentRequirementSchema)
	agentStructureSchemaJSON, _ := json.Marshal(agentStructureSchema)
	agentConfigSchemaJSON, _ := json.Marshal(agentConfigSchema)
	agentValidationSchemaJSON, _ := json.Marshal(agentValidationSchema)

	return &SystemWorkflowDefinition{
		ID:          "a0000000-0000-0000-0000-000000000002",
		SystemSlug:  "ai-builder",
		Name:        "AI Workflow Builder",
		Description: "AI-assisted workflow building with deep thinking, proposal-based confirmation, automatic construction, and refinement capabilities",
		Version:     27,
		IsSystem:    true,
		Steps: []SystemStepDefinition{
			// ============================
			// Analysis Entry Point (Y=40)
			// AI thinks deeply and proposes
			// ============================
			{
				TempID:      "start_analysis",
				Name:        "Start: Analysis",
				Type:        "start",
				TriggerType: "internal",
				TriggerConfig: json.RawMessage(`{
					"entry_point": "analysis",
					"description": "AI analyzes input deeply and proposes assumptions + questions"
				}`),
				PositionX: 40,
				PositionY: 40,
				Config: json.RawMessage(`{
					"input_schema": {
						"type": "object",
						"required": ["session_id", "message"],
						"properties": {
							"session_id": {"type": "string", "title": "Session ID"},
							"message": {"type": "string", "title": "User Message"},
							"tenant_id": {"type": "string", "title": "Tenant ID"},
							"user_id": {"type": "string", "title": "User ID"}
						}
					}
				}`),
			},
			{
				TempID:    "analysis_get_context",
				Name:      "Get Session Context",
				Type:      "function",
				PositionX: 160,
				PositionY: 40,
				Config: json.RawMessage(`{
					"code": "const session = ctx.builderSessions.get(input.session_id); const blocks = ctx.blocks.list(); const phase = session?.hearing_phase || 'analysis'; const spec = session?.spec || {}; const messages = session?.messages || []; const historyText = messages.map(m => (m.role === 'user' ? 'User: ' : 'AI: ') + m.content).join('\\n'); const blocksInfo = blocks.slice(0, 30).map(b => '- ' + b.slug + ': ' + b.name + ' (' + b.category + ')').join('\\n'); return { session_id: session?.id || input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, message: input.message, current_phase: phase, current_spec: spec, history_text: historyText, blocks_info: blocksInfo };",
					"language": "javascript"
				}`),
			},
			{
				TempID:    "analysis_llm",
				Name:      "Analysis LLM",
				Type:      "llm-structured",
				PositionX: 280,
				PositionY: 40,
				Config: func() json.RawMessage {
					cfg := map[string]interface{}{
						"model":       "claude-sonnet-4-20250514",
						"provider":    "anthropic",
						"max_tokens":  4000,
						"temperature": 0.3,
						"user_prompt": `You are a workflow builder AI with deep thinking capabilities.

## Your Task
Analyze the user's request and:
1. **Think deeply** about what workflow they need
2. **Infer and assume** reasonable defaults for unclear parts
3. **Propose a complete WorkflowSpec** based on your analysis
4. **List only truly unclear points** as clarifying questions (0-3 max)

## Key Principles
- **Minimize questions**: Only ask what you absolutely cannot infer
- **Be proactive**: Make reasonable assumptions and state them clearly
- **Think like an expert**: Use domain knowledge to fill gaps
- **Respect user time**: They want you to do the thinking

## Current Session
Phase: {{$.current_phase}}
Existing Spec: {{$.current_spec}}

## Conversation History
{{$.history_text}}

## User's Latest Message
{{$.message}}

## Available Blocks (Reference)
{{$.blocks_info}}

## Important Rules
- Response must be friendly Japanese
- Only ask 0-3 clarifying questions MAX
- High confidence assumptions need no confirmation
- If you have enough info, set nextPhase to "proposal"
- Include your thinking process to show transparency`,
						"system_prompt": `You are an expert workflow designer AI. You think deeply about requirements and make intelligent assumptions. You minimize questions to respect the user's time.

Key principles:
1. Think like an expert consultant who can infer needs
2. Make reasonable assumptions and clearly state them
3. Only ask questions for truly ambiguous critical decisions
4. Propose a complete workflow spec proactively`,
						"output_schema":      json.RawMessage(analysisSchemaJSON),
						"validate_output":    true,
						"include_examples":   true,
						"passthrough_fields": []string{"session_id", "tenant_id", "user_id", "current_phase", "current_spec"},
					}
					data, _ := json.Marshal(cfg)
					return data
				}(),
			},
			{
				TempID:    "analysis_update_session",
				Name:      "Update Builder Session",
				Type:      "function",
				PositionX: 400,
				PositionY: 40,
				Config: json.RawMessage(`{
					"code": "const sessionId = input.session_id; const nextPhase = input.nextPhase || 'analysis'; const progress = input.progress || (nextPhase === 'proposal' ? 70 : nextPhase === 'completed' ? 100 : 30); const mergedSpec = { ...input.current_spec, ...input.proposed_spec, assumptions: input.assumptions }; ctx.builderSessions.update(sessionId, { hearing_phase: nextPhase, hearing_progress: progress, spec: mergedSpec }); ctx.builderSessions.addMessage(sessionId, { role: 'assistant', content: input.response, phase: nextPhase, extracted_data: { thinking: input.thinking, proposed_spec: input.proposed_spec, assumptions: input.assumptions, clarifying_points: input.clarifying_points } }); return { session_id: sessionId, message: { content: input.response, thinking: input.thinking, proposed_spec: input.proposed_spec, assumptions: input.assumptions, clarifying_points: input.clarifying_points }, phase: nextPhase, progress: progress, complete: nextPhase === 'completed' };",
					"language": "javascript"
				}`),
			},

			// ============================
			// Proposal Entry Point (Y=100)
			// Update assumptions and finalize
			// ============================
			{
				TempID:      "start_proposal",
				Name:        "Start: Proposal",
				Type:        "start",
				TriggerType: "internal",
				TriggerConfig: json.RawMessage(`{
					"entry_point": "proposal",
					"description": "Update assumptions based on user answers and finalize spec"
				}`),
				PositionX: 40,
				PositionY: 100,
				Config: json.RawMessage(`{
					"input_schema": {
						"type": "object",
						"required": ["session_id", "message"],
						"properties": {
							"session_id": {"type": "string", "title": "Session ID"},
							"message": {"type": "string", "title": "User Answers/Feedback"},
							"tenant_id": {"type": "string", "title": "Tenant ID"},
							"user_id": {"type": "string", "title": "User ID"}
						}
					}
				}`),
			},
			{
				TempID:    "proposal_get_context",
				Name:      "Get Proposal Context",
				Type:      "function",
				PositionX: 160,
				PositionY: 100,
				Config: json.RawMessage(`{
					"code": "const session = ctx.builderSessions.get(input.session_id); const blocks = ctx.blocks.list(); const spec = session?.spec || {}; const assumptions = spec.assumptions || []; const messages = session?.messages || []; const lastMsg = messages[messages.length - 1]; const clarifyingPoints = lastMsg?.extracted_data?.clarifying_points || []; const historyText = messages.map(m => (m.role === 'user' ? 'User: ' : 'AI: ') + m.content).join('\\n'); const blocksInfo = blocks.slice(0, 30).map(b => '- ' + b.slug + ': ' + b.name + ' (' + b.category + ')').join('\\n'); return { session_id: session?.id || input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, message: input.message, current_spec: spec, assumptions: assumptions, clarifying_points: clarifyingPoints, history_text: historyText, blocks_info: blocksInfo };",
					"language": "javascript"
				}`),
			},
			{
				TempID:    "proposal_llm",
				Name:      "Proposal LLM",
				Type:      "llm-structured",
				PositionX: 280,
				PositionY: 100,
				Config: func() json.RawMessage {
					cfg := map[string]interface{}{
						"model":       "claude-sonnet-4-20250514",
						"provider":    "anthropic",
						"max_tokens":  4000,
						"temperature": 0.2,
						"user_prompt": `You are a workflow builder AI finalizing the workflow specification.

## Your Task
1. Review the user's answers/feedback to your previous questions
2. Update assumptions based on their input
3. Finalize the complete WorkflowSpec
4. If more clarification is needed, ask follow-up (but try to minimize)

## Previous Assumptions
{{$.assumptions}}

## Previous Clarifying Questions
{{$.clarifying_points}}

## Current Spec
{{$.current_spec}}

## Conversation History
{{$.history_text}}

## User's Latest Response
{{$.message}}

## Available Blocks
{{$.blocks_info}}

## Important
- Set nextPhase to "completed" when spec is finalized
- Mark all assumptions as confirmed: true
- Response should be friendly Japanese
- No more clarifying_points in this phase (finalization only)`,
						"system_prompt":      "You are an expert workflow designer finalizing a workflow specification. Update assumptions based on user feedback and produce a complete, ready-to-build spec.",
						"output_schema":      json.RawMessage(proposalSchemaJSON),
						"validate_output":    true,
						"include_examples":   true,
						"passthrough_fields": []string{"session_id", "tenant_id", "user_id", "current_spec"},
					}
					data, _ := json.Marshal(cfg)
					return data
				}(),
			},
			{
				TempID:    "proposal_update_session",
				Name:      "Update Session with Final Spec",
				Type:      "function",
				PositionX: 400,
				PositionY: 100,
				Config: json.RawMessage(`{
					"code": "const sessionId = input.session_id; const nextPhase = input.nextPhase || 'completed'; const progress = input.progress || (nextPhase === 'completed' ? 100 : 85); const finalSpec = { ...input.final_spec, assumptions: input.assumptions }; ctx.builderSessions.update(sessionId, { hearing_phase: nextPhase, hearing_progress: progress, spec: finalSpec }); ctx.builderSessions.addMessage(sessionId, { role: 'assistant', content: input.response, phase: nextPhase, extracted_data: { final_spec: input.final_spec, assumptions: input.assumptions } }); return { session_id: sessionId, message: { content: input.response, final_spec: input.final_spec, assumptions: input.assumptions }, phase: nextPhase, progress: progress, complete: nextPhase === 'completed' };",
					"language": "javascript"
				}`),
			},

			// ============================
			// Construct Entry Point (Y=160)
			// ============================
			{
				TempID:      "start_construct",
				Name:        "Start: Construct",
				Type:        "start",
				TriggerType: "internal",
				TriggerConfig: json.RawMessage(`{
					"entry_point": "construct",
					"description": "Build workflow from gathered requirements"
				}`),
				PositionX: 40,
				PositionY: 160,
				Config: json.RawMessage(`{
					"input_schema": {
						"type": "object",
						"required": ["session_id"],
						"properties": {
							"session_id": {"type": "string", "title": "Session ID"},
							"tenant_id": {"type": "string", "title": "Tenant ID"},
							"user_id": {"type": "string", "title": "User ID"}
						}
					}
				}`),
			},
			{
				TempID:    "construct_get_spec",
				Name:      "Get Workflow Spec",
				Type:      "function",
				PositionX: 160,
				PositionY: 160,
				Config: json.RawMessage(`{
					"code": "const session = ctx.builderSessions.get(input.session_id); const blocks = ctx.blocks.list(); const blocksInfo = blocks.map(b => '- ' + b.slug + ': ' + b.name + ' (' + b.category + ') - ' + (b.description || '')).join('\\n'); const blockCategories = {}; blocks.forEach(b => { if (!blockCategories[b.category]) blockCategories[b.category] = []; blockCategories[b.category].push(b.slug); }); const categoryInfo = Object.entries(blockCategories).map(function(entry) { return '- ' + entry[0] + ': ' + entry[1].join(', '); }).join('\\n'); return { session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, spec: session?.spec || {}, blocks_info: blocksInfo, category_info: categoryInfo };",
					"language": "javascript"
				}`),
			},
			{
				TempID:    "construct_llm",
				Name:      "Construct LLM",
				Type:      "llm-structured",
				PositionX: 280,
				PositionY: 160,
				Config: func() json.RawMessage {
					cfg := map[string]interface{}{
						"model":       "claude-sonnet-4-20250514",
						"provider":    "anthropic",
						"max_tokens":  4000,
						"temperature": 0.3,
						"user_prompt": `You are a workflow construction AI. Build a concrete workflow from the specification.

## Workflow Specification
{{$.spec}}

## Block Mapping Guide
Map each step to blocks with priority:
1. Preset blocks (highest priority) - use existing blocks
2. Custom required - when preset is insufficient

### Blocks by Category
{{$.category_info}}

### All Blocks
{{$.blocks_info}}

## Step Types
- start: Entry point (required)
- llm: AI/LLM call
- tool: External adapter (specify block_slug)
- condition: Conditional branch (true/false)
- switch: Multi-branch routing
- map: Parallel array processing
- loop: Loop
- wait: Wait
- function: Custom JavaScript
- human_in_loop: Human approval
- log: Debug log

## Important
- Must include a start step
- Maximize use of preset blocks (mapping_status=preset)
- Specify reason and confidence for custom blocks
- natural_language_summary should be user-friendly, avoid technical jargon
- editable_points lists things user can customize later`,
						"system_prompt":      "You are an expert workflow construction AI. Build optimal workflow structures from specifications. Maximize use of preset blocks. Provide clear, user-friendly explanations.",
						"output_schema":      json.RawMessage(constructSchemaJSON),
						"validate_output":    true,
						"include_examples":   true,
						"passthrough_fields": []string{"session_id", "tenant_id", "user_id", "spec"},
					}
					data, _ := json.Marshal(cfg)
					return data
				}(),
			},
			{
				TempID:    "construct_create_project",
				Name:      "Create Project",
				Type:      "function",
				PositionX: 400,
				PositionY: 160,
				Config: json.RawMessage(`{
					"code": "const sessionId = input.session_id; const tenantId = input.tenant_id; const userId = input.user_id; const spec = input.spec; const steps = input.steps || []; const edges = input.edges || []; if (!steps.length) { return { success: false, error: 'No steps to create' }; } const validTypes = ['start', 'llm', 'tool', 'condition', 'switch', 'map', 'join', 'subflow', 'loop', 'wait', 'function', 'router', 'human_in_loop', 'filter', 'split', 'aggregate', 'error', 'note', 'log', 'webhook_trigger']; const validSteps = steps.filter(s => validTypes.includes(s.type)); const projectOpts = { tenant_id: tenantId, name: input.workflow_name || 'AI Generated Workflow', description: input.workflow_description || '', status: 'draft' }; if (userId) { projectOpts.created_by = userId; } const project = ctx.projects.create(projectOpts); const stepIdMap = {}; const startingX = 40; const xSpacing = 120; const baseY = 40; for (let i = 0; i < validSteps.length; i++) { const step = validSteps[i]; const createdStep = ctx.steps.create({ tenant_id: tenantId, project_id: project.id, name: step.name, type: step.type, config: step.config || {}, position_x: startingX + (i * xSpacing), position_y: baseY, block_slug: step.block_slug }); stepIdMap[step.temp_id] = createdStep.id; } for (const edge of edges) { const sourceId = stepIdMap[edge.source_temp_id]; const targetId = stepIdMap[edge.target_temp_id]; if (sourceId && targetId) { ctx.edges.create({ tenant_id: tenantId, project_id: project.id, source_step_id: sourceId, target_step_id: targetId, source_port: edge.source_port || 'output', target_port: edge.target_port || 'input' }); } } ctx.builderSessions.update(sessionId, { status: 'reviewing', project_id: project.id }); const stepMappings = validSteps.map(s => ({ name: s.name, type: s.type, block: s.block_slug || null, mapping_status: s.mapping_status || (s.custom_required ? 'custom' : 'preset'), confidence: s.mapping_confidence || 'high', custom_required: s.custom_required || false, custom_reason: s.custom_reason, executor: s.executor || 'system' })); const nlSummary = input.natural_language_summary || input.workflow_description; ctx.builderSessions.addMessage(sessionId, { role: 'assistant', content: '## Workflow Created\\n\\n' + nlSummary + '\\n\\n### Step Structure\\n' + stepMappings.map(function(s, i) { return (i+1) + '. ' + s.name + ' (' + (s.mapping_status === 'preset' ? 'Preset' : 'Custom needed') + ')'; }).join('\\n') + (input.editable_points?.length > 0 ? '\\n\\n### Customizable Points\\n' + input.editable_points.map(function(p) { return '- ' + p; }).join('\\n') : '') + (input.warnings?.length > 0 ? '\\n\\n### Notes\\n' + input.warnings.map(function(w) { return '- ' + w; }).join('\\n') : '') }); return { success: true, project_id: project.id, natural_language_summary: nlSummary, step_mappings: stepMappings, summary: input.summary, editable_points: input.editable_points || [], warnings: input.warnings || [] };",
					"language": "javascript"
				}`),
			},

			// ============================
			// Refine Entry Point (Y=220)
			// ============================
			{
				TempID:      "start_refine",
				Name:        "Start: Refine",
				Type:        "start",
				TriggerType: "internal",
				TriggerConfig: json.RawMessage(`{
					"entry_point": "refine",
					"description": "Refine existing workflow based on user feedback"
				}`),
				PositionX: 40,
				PositionY: 220,
				Config: json.RawMessage(`{
					"input_schema": {
						"type": "object",
						"required": ["session_id", "feedback"],
						"properties": {
							"session_id": {"type": "string", "title": "Session ID"},
							"project_id": {"type": "string", "title": "Project ID"},
							"feedback": {"type": "string", "title": "User Feedback"},
							"tenant_id": {"type": "string", "title": "Tenant ID"},
							"user_id": {"type": "string", "title": "User ID"}
						}
					}
				}`),
			},
			{
				TempID:    "refine_get_workflow",
				Name:      "Get Current Workflow",
				Type:      "function",
				PositionX: 160,
				PositionY: 220,
				Config: json.RawMessage(`{
					"code": "const project = ctx.projects.get(input.project_id); const steps = ctx.steps.listByProject(input.project_id); const edges = ctx.edges.listByProject(input.project_id); const blocks = ctx.blocks.list(); const stepsInfo = (steps || []).map(s => '- ' + s.name + ' (type: ' + s.type + ', id: ' + s.id + ')').join('\\n'); const edgesInfo = (edges || []).map(e => '- ' + e.source_step_id + ' -> ' + e.target_step_id).join('\\n'); const blocksInfo = blocks.slice(0, 30).map(b => '- ' + b.slug + ': ' + b.name).join('\\n'); return { session_id: input.session_id, project_id: input.project_id, tenant_id: input.tenant_id, user_id: input.user_id, feedback: input.feedback, workflow_name: project?.name, workflow_description: project?.description, steps_info: stepsInfo, edges_info: edgesInfo, blocks_info: blocksInfo };",
					"language": "javascript"
				}`),
			},
			{
				TempID:    "refine_llm",
				Name:      "Refine LLM",
				Type:      "llm-structured",
				PositionX: 280,
				PositionY: 220,
				Config: func() json.RawMessage {
					cfg := map[string]interface{}{
						"model":       "claude-sonnet-4-20250514",
						"provider":    "anthropic",
						"max_tokens":  2000,
						"temperature": 0.3,
						"user_prompt": `You are a workflow modification AI. Modify the workflow based on user feedback.

## Current Workflow
Name: {{$.workflow_name}}
Description: {{$.workflow_description}}

### Steps
{{$.steps_info}}

### Edges (Connections)
{{$.edges_info}}

## User Feedback
{{$.feedback}}

## Available Blocks
{{$.blocks_info}}

## Change Types
- add_step: Add a new step
- remove_step: Remove an existing step
- modify_step: Modify step configuration
- add_edge: Add a new connection
- remove_edge: Remove a connection
- modify_config: Modify configuration only`,
						"system_prompt":      "You are a workflow modification AI. Accurately understand user feedback and implement minimal changes to fulfill requirements.",
						"output_schema":      json.RawMessage(refineSchemaJSON),
						"validate_output":    true,
						"include_examples":   true,
						"passthrough_fields": []string{"session_id", "project_id", "tenant_id", "user_id"},
					}
					data, _ := json.Marshal(cfg)
					return data
				}(),
			},
			{
				TempID:    "refine_apply_changes",
				Name:      "Apply Changes",
				Type:      "function",
				PositionX: 400,
				PositionY: 220,
				Config: json.RawMessage(`{
					"code": "const projectId = input.project_id; const changes = input.changes || []; const appliedChanges = []; for (const change of changes) { try { switch (change.type) { case 'add_step': const newStep = ctx.steps.create({ project_id: projectId, name: change.step.name, type: change.step.type, config: change.step.config || {}, position_x: change.step.position_x || 40, position_y: change.step.position_y || 40 }); appliedChanges.push({ type: 'add_step', step_id: newStep.id, name: change.step.name }); break; case 'remove_step': ctx.steps.delete(change.step_id); appliedChanges.push({ type: 'remove_step', step_id: change.step_id }); break; case 'modify_step': case 'modify_config': ctx.steps.update(change.step_id, change.config || change.step); appliedChanges.push({ type: 'modify_step', step_id: change.step_id }); break; case 'add_edge': ctx.edges.create({ project_id: projectId, source_step_id: change.edge.source_step_id, target_step_id: change.edge.target_step_id, source_port: change.edge.source_port || 'output', target_port: change.edge.target_port || 'input' }); appliedChanges.push({ type: 'add_edge' }); break; case 'remove_edge': ctx.edges.delete(change.edge.id); appliedChanges.push({ type: 'remove_edge' }); break; } } catch (e) { appliedChanges.push({ type: change.type, error: e.message }); } } ctx.projects.incrementVersion(projectId); ctx.builderSessions.addMessage(input.session_id, { role: 'assistant', content: input.response }); return { success: true, applied_changes: appliedChanges, summary: input.summary, response: input.response };",
					"language": "javascript"
				}`),
			},

			// ========================================
			// Agent-Based Construct Entry Point (Y=280)
			// 4-Agent Architecture for high-quality workflow generation
			// ========================================
			{
				TempID:      "start_agent_construct",
				Name:        "Start: Agent Construct",
				Type:        "start",
				TriggerType: "internal",
				TriggerConfig: json.RawMessage(`{
					"entry_point": "agent_construct",
					"description": "Agent-based workflow construction with ConfigSchema awareness"
				}`),
				PositionX: 40,
				PositionY: 280,
				Config: json.RawMessage(`{
					"input_schema": {
						"type": "object",
						"required": ["session_id"],
						"properties": {
							"session_id": {"type": "string", "title": "Session ID"},
							"tenant_id": {"type": "string", "title": "Tenant ID"},
							"user_id": {"type": "string", "title": "User ID"}
						}
					}
				}`),
			},

			// Agent 1: Requirement Analysis
			{
				TempID:    "agent1_get_context",
				Name:      "Agent1: Get Context",
				Type:      "function",
				PositionX: 160,
				PositionY: 280,
				Config: json.RawMessage(`{
					"code": "const session = ctx.builderSessions.get(input.session_id); const blocks = ctx.blocks.list(); const spec = session?.spec || {}; const messages = session?.messages || []; const historyText = messages.map(m => (m.role === 'user' ? 'User: ' : 'AI: ') + m.content).slice(-5).join('\\n'); const blocksInfo = blocks.slice(0, 30).map(b => '- ' + b.slug + ': ' + b.name + ' (' + b.category + ')').join('\\n'); return { session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, spec: spec, history_text: historyText, blocks_info: blocksInfo, blocks: blocks.map(b => ({ slug: b.slug, name: b.name, description: b.description, category: b.category })) };",
					"language": "javascript"
				}`),
			},
			{
				TempID:    "agent1_llm",
				Name:      "Agent1: Analyze Requirements",
				Type:      "llm-structured",
				PositionX: 280,
				PositionY: 280,
				Config: func() json.RawMessage {
					cfg := map[string]interface{}{
						"model":       "claude-sonnet-4-20250514",
						"provider":    "anthropic",
						"max_tokens":  2000,
						"temperature": 0.2,
						"user_prompt": `You are Agent 1: Requirement Analyzer.

## Your Task
Analyze the workflow specification and:
1. Clarify any ambiguous requirements
2. Identify missing information
3. Generate a clear, structured specification

## Current Specification
{{$.spec}}

## Conversation History
{{$.history_text}}

## Available Block Categories
{{$.blocks_info}}

## Important
- Make reasonable assumptions for unclear parts
- Set ready_for_design to true if spec is complete enough`,
						"system_prompt":      "You are a requirement analysis agent. Your job is to clarify and structure workflow requirements.",
						"output_schema":      json.RawMessage(agentRequirementSchemaJSON),
						"validate_output":    true,
						"include_examples":   true,
						"passthrough_fields": []string{"session_id", "tenant_id", "user_id", "blocks"},
					}
					data, _ := json.Marshal(cfg)
					return data
				}(),
			},

			// Agent 2: Structure Design
			{
				TempID:    "agent2_prepare",
				Name:      "Agent2: Prepare Context",
				Type:      "function",
				PositionX: 400,
				PositionY: 280,
				Config: json.RawMessage(`{
					"code": "const blocksInfo = (input.blocks || []).map(b => '- ' + b.slug + ': ' + b.name + ' (' + b.category + ') - ' + (b.description || '')).join('\\n'); return { session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, clarified_spec: input.clarified_spec || input.spec || {}, assumptions: input.assumptions || [], blocks_info: blocksInfo, blocks: input.blocks };",
					"language": "javascript"
				}`),
			},
			{
				TempID:    "agent2_llm",
				Name:      "Agent2: Design Structure",
				Type:      "llm-structured",
				PositionX: 520,
				PositionY: 280,
				Config: func() json.RawMessage {
					cfg := map[string]interface{}{
						"model":       "claude-sonnet-4-20250514",
						"provider":    "anthropic",
						"max_tokens":  3000,
						"temperature": 0.3,
						"user_prompt": `You are Agent 2: Structure Designer.

## Your Task
Design the workflow DAG structure based on the clarified specification.

## Clarified Specification
{{$.clarified_spec}}

## Assumptions Made
{{$.assumptions}}

## Available Blocks
{{$.blocks_info}}

## Step Types
- start: Entry point (required first)
- llm: AI/LLM call
- tool: External adapter using block_slug
- function: Custom JavaScript
- condition: True/false branch
- loop: Iteration

## Important
- Always start with a "start" type step
- Set needs_config: true for steps that require configuration
- Use block_slug to reference available blocks`,
						"system_prompt":      "You are a workflow structure design agent. Design optimal DAG structures for workflows.",
						"output_schema":      json.RawMessage(agentStructureSchemaJSON),
						"validate_output":    true,
						"include_examples":   true,
						"passthrough_fields": []string{"session_id", "tenant_id", "user_id", "clarified_spec", "assumptions", "blocks"},
					}
					data, _ := json.Marshal(cfg)
					return data
				}(),
			},
			{
				TempID:    "agent2_parse",
				Name:      "Agent2: Parse Structure",
				Type:      "function",
				PositionX: 640,
				PositionY: 280,
				Config: json.RawMessage(`{
					"code": "const stepsNeedingConfig = (input.step_skeletons || []).filter(s => s.block_slug || (s.needs_config && s.type !== 'start')); return { workflow_name: input.workflow_name || 'Generated Workflow', workflow_description: input.workflow_description || '', step_skeletons: input.step_skeletons || [], edges: input.edges || [], design_notes: input.design_notes || '', steps_needing_config: stepsNeedingConfig, current_step_index: 0, configured_steps: [], session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, clarified_spec: input.clarified_spec, blocks: input.blocks };",
					"language": "javascript"
				}`),
			},

			// Agent 3: Step Configuration Loop
			{
				TempID:    "agent3_prepare_step",
				Name:      "Agent3: Prepare Step Config",
				Type:      "function",
				PositionX: 160,
				PositionY: 340,
				Config: json.RawMessage(`{
					"code": "const steps = input.steps_needing_config || []; const idx = input.current_step_index || 0; if (idx >= steps.length) { return { loop_complete: true, configured_steps: input.configured_steps || [], step_skeletons: input.step_skeletons, edges: input.edges, workflow_name: input.workflow_name, workflow_description: input.workflow_description, session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, clarified_spec: input.clarified_spec }; } const currentStep = steps[idx]; const blockInfo = ctx.blocks.getWithSchema(currentStep.block_slug); const previousSteps = input.configured_steps || []; const previousOutputs = previousSteps.map(s => ({ step_id: s.temp_id, step_name: s.name, output_fields: s.output_fields || [] })); return { loop_complete: false, current_step: currentStep, block_info: blockInfo, block_config_schema: blockInfo?.config_schema || {}, block_required_fields: blockInfo?.required_fields || [], block_defaults: blockInfo?.resolved_config_defaults || {}, previous_outputs: previousOutputs, current_step_index: idx, steps_needing_config: steps, configured_steps: input.configured_steps || [], step_skeletons: input.step_skeletons, edges: input.edges, workflow_name: input.workflow_name, workflow_description: input.workflow_description, session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, clarified_spec: input.clarified_spec };",
					"language": "javascript"
				}`),
			},
			{
				TempID:    "agent3_check_loop",
				Name:      "Agent3: Check Loop",
				Type:      "condition",
				PositionX: 280,
				PositionY: 340,
				Config: json.RawMessage(`{
					"expression": "$.loop_complete == true",
					"true_label": "All steps configured",
					"false_label": "More steps to configure"
				}`),
			},
			{
				TempID:    "agent3_llm",
				Name:      "Agent3: Generate Config",
				Type:      "llm-structured",
				PositionX: 400,
				PositionY: 340,
				Config: func() json.RawMessage {
					cfg := map[string]interface{}{
						"model":       "claude-sonnet-4-20250514",
						"provider":    "anthropic",
						"max_tokens":  2000,
						"temperature": 0.2,
						"user_prompt": `You are Agent 3: Step Configurator.

## Your Task
Generate the config for step "{{$.current_step.name}}".

## Step Information
- Name: {{$.current_step.name}}
- Description: {{$.current_step.description}}
- Purpose: {{$.current_step.purpose}}

## Block Information
- Slug: {{$.block_info.slug}}
- Name: {{$.block_info.name}}

## ConfigSchema (IMPORTANT - Follow This Exactly)
{{$.block_config_schema}}

## Required Fields (Must Be Set)
{{$.block_required_fields}}

## Default Values (Can Override)
{{$.block_defaults}}

## Previous Step Outputs (Use for Template Variables)
{{$.previous_outputs}}

## Workflow Context
{{$.clarified_spec}}

## Important
- Follow ConfigSchema structure exactly
- Set ALL required fields
- Use template variables like {{step_id.field}} for dynamic data`,
						"system_prompt":      "You are a step configuration agent. Generate valid config objects that match the provided ConfigSchema exactly. Pay close attention to required fields and data types.",
						"output_schema":      json.RawMessage(agentConfigSchemaJSON),
						"validate_output":    true,
						"include_examples":   true,
						"passthrough_fields": []string{"current_step", "current_step_index", "steps_needing_config", "configured_steps", "step_skeletons", "edges", "workflow_name", "workflow_description", "session_id", "tenant_id", "user_id", "clarified_spec"},
					}
					data, _ := json.Marshal(cfg)
					return data
				}(),
			},
			{
				TempID:    "agent3_process_config",
				Name:      "Agent3: Process Config",
				Type:      "function",
				PositionX: 520,
				PositionY: 340,
				Config: json.RawMessage(`{
					"code": "const step = input.current_step; const configuredStep = { temp_id: step.temp_id, name: step.name, type: step.type, block_slug: step.block_slug, description: step.description, config: input.config || {}, output_fields: input.output_fields || [], reasoning: input.reasoning || '' }; const newConfigured = [...(input.configured_steps || []), configuredStep]; const nextIndex = (input.current_step_index || 0) + 1; const stepsNeedingConfig = input.steps_needing_config || []; const loopComplete = nextIndex >= stepsNeedingConfig.length; return { loop_complete: loopComplete, current_step_index: nextIndex, steps_needing_config: stepsNeedingConfig, configured_steps: newConfigured, step_skeletons: input.step_skeletons, edges: input.edges, workflow_name: input.workflow_name, workflow_description: input.workflow_description, session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, clarified_spec: input.clarified_spec };",
					"language": "javascript"
				}`),
			},

			// Agent 4: Validation & Final Assembly
			{
				TempID:    "agent4_merge_steps",
				Name:      "Agent4: Merge All Steps",
				Type:      "function",
				PositionX: 160,
				PositionY: 400,
				Config: json.RawMessage(`{
					"code": "const skeletons = input.step_skeletons || []; const configured = input.configured_steps || []; const configuredMap = {}; configured.forEach(c => { configuredMap[c.temp_id] = c; }); const mergedSteps = skeletons.map(function(s, i) { const cfg = configuredMap[s.temp_id]; return { temp_id: s.temp_id, name: s.name, type: s.type, description: s.description, block_slug: s.block_slug || (cfg ? cfg.block_slug : null), config: (cfg ? cfg.config : s.config) || {}, position_x: 40 + (i * 120), position_y: 40 }; }); const stepsInfo = mergedSteps.map(s => '- ' + s.temp_id + ': ' + s.name + ' (type: ' + s.type + ')\\n  Config: ' + JSON.stringify(s.config)).join('\\n'); const edgesInfo = (input.edges || []).map(e => '- ' + e.source + ' -> ' + e.target + (e.label ? ' [' + e.label + ']' : '')).join('\\n'); return { merged_steps: mergedSteps, edges: input.edges, workflow_name: input.workflow_name, workflow_description: input.workflow_description, session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, clarified_spec: input.clarified_spec, steps_info: stepsInfo, edges_info: edgesInfo };",
					"language": "javascript"
				}`),
			},
			{
				TempID:    "agent4_llm",
				Name:      "Agent4: Validate Workflow",
				Type:      "llm-structured",
				PositionX: 280,
				PositionY: 400,
				Config: func() json.RawMessage {
					cfg := map[string]interface{}{
						"model":       "claude-sonnet-4-20250514",
						"provider":    "anthropic",
						"max_tokens":  3000,
						"temperature": 0.1,
						"user_prompt": `You are Agent 4: Workflow Validator.

## Your Task
Validate the workflow and fix any issues.

## Workflow
Name: {{$.workflow_name}}
Description: {{$.workflow_description}}

## Steps
{{$.steps_info}}

## Edges (Connections)
{{$.edges_info}}

## Specification
{{$.clarified_spec}}

## Validation Checks
1. Does workflow have a start step?
2. Are all edges valid (source and target exist)?
3. Are required configs filled?
4. Are template variables valid (reference existing steps)?
5. Is the workflow logically complete?

## Important
- If is_valid is false, provide fixes in fixes_applied
- final_steps and final_edges should be corrected versions`,
						"system_prompt":      "You are a workflow validation agent. Check for errors and fix them. Be thorough but practical.",
						"output_schema":      json.RawMessage(agentValidationSchemaJSON),
						"validate_output":    true,
						"include_examples":   true,
						"passthrough_fields": []string{"merged_steps", "edges", "workflow_name", "workflow_description", "session_id", "tenant_id", "user_id", "clarified_spec"},
					}
					data, _ := json.Marshal(cfg)
					return data
				}(),
			},

			// Final: Create Project
			{
				TempID:    "agent_create_project",
				Name:      "Agent: Create Project",
				Type:      "function",
				PositionX: 400,
				PositionY: 400,
				Config: json.RawMessage(`{
					"code": "const sessionId = input.session_id; const tenantId = input.tenant_id; const userId = input.user_id; const steps = input.final_steps || input.merged_steps || []; const edges = input.final_edges || input.edges || []; if (!steps.length) { return { success: false, error: 'No steps to create' }; } const projectOpts = { tenant_id: tenantId, name: input.workflow_name || 'AI Generated Workflow', description: input.workflow_description || '', status: 'draft' }; if (userId) { projectOpts.created_by = userId; } const project = ctx.projects.create(projectOpts); const stepIdMap = {}; for (let i = 0; i < steps.length; i++) { const step = steps[i]; const createdStep = ctx.steps.create({ tenant_id: tenantId, project_id: project.id, name: step.name, type: step.type, config: step.config || {}, position_x: step.position_x || (40 + (i * 120)), position_y: step.position_y || 40, block_slug: step.block_slug }); stepIdMap[step.temp_id] = createdStep.id; } for (const edge of edges) { const sourceId = stepIdMap[edge.source]; const targetId = stepIdMap[edge.target]; if (sourceId && targetId) { ctx.edges.create({ tenant_id: tenantId, project_id: project.id, source_step_id: sourceId, target_step_id: targetId, source_port: edge.source_port || 'output', target_port: edge.target_port || 'input' }); } } ctx.builderSessions.update(sessionId, { status: 'reviewing', project_id: project.id }); const summary = 'Agent-based construction complete.\\n\\n' + '**' + input.workflow_name + '**\\n' + (input.workflow_description || '') + '\\n\\n' + '### Steps Created (' + steps.length + ')\\n' + steps.map(function(s, i) { return (i+1) + '. ' + s.name + ' (' + s.type + ')'; }).join('\\n') + (input.summary ? '\\n\\n### Validation Notes\\n' + input.summary : ''); ctx.builderSessions.addMessage(sessionId, { role: 'assistant', content: summary }); return { success: true, project_id: project.id, workflow_name: input.workflow_name, steps_created: steps.length, edges_created: edges.length, validation_summary: input.summary || '', issues_found: input.issues || [], fixes_applied: input.fixes_applied || [] };",
					"language": "javascript"
				}`),
			},
		},
		Edges: []SystemEdgeDefinition{
			// Analysis flow (simplified: 4 steps instead of 6)
			{SourceTempID: "start_analysis", TargetTempID: "analysis_get_context", SourcePort: "output"},
			{SourceTempID: "analysis_get_context", TargetTempID: "analysis_llm", SourcePort: "output"},
			{SourceTempID: "analysis_llm", TargetTempID: "analysis_update_session", SourcePort: "output"},

			// Proposal flow (simplified: 4 steps instead of 6)
			{SourceTempID: "start_proposal", TargetTempID: "proposal_get_context", SourcePort: "output"},
			{SourceTempID: "proposal_get_context", TargetTempID: "proposal_llm", SourcePort: "output"},
			{SourceTempID: "proposal_llm", TargetTempID: "proposal_update_session", SourcePort: "output"},

			// Construct flow (simplified: 4 steps instead of 6)
			{SourceTempID: "start_construct", TargetTempID: "construct_get_spec", SourcePort: "output"},
			{SourceTempID: "construct_get_spec", TargetTempID: "construct_llm", SourcePort: "output"},
			{SourceTempID: "construct_llm", TargetTempID: "construct_create_project", SourcePort: "output"},

			// Refine flow (simplified: 4 steps instead of 6)
			{SourceTempID: "start_refine", TargetTempID: "refine_get_workflow", SourcePort: "output"},
			{SourceTempID: "refine_get_workflow", TargetTempID: "refine_llm", SourcePort: "output"},
			{SourceTempID: "refine_llm", TargetTempID: "refine_apply_changes", SourcePort: "output"},

			// Agent-based Construct flow
			// Agent 1: Requirements Analysis
			{SourceTempID: "start_agent_construct", TargetTempID: "agent1_get_context", SourcePort: "output"},
			{SourceTempID: "agent1_get_context", TargetTempID: "agent1_llm", SourcePort: "output"},
			// Agent 2: Structure Design
			{SourceTempID: "agent1_llm", TargetTempID: "agent2_prepare", SourcePort: "output"},
			{SourceTempID: "agent2_prepare", TargetTempID: "agent2_llm", SourcePort: "output"},
			{SourceTempID: "agent2_llm", TargetTempID: "agent2_parse", SourcePort: "output"},
			// Agent 3: Step Configuration Loop
			{SourceTempID: "agent2_parse", TargetTempID: "agent3_prepare_step", SourcePort: "output"},
			{SourceTempID: "agent3_prepare_step", TargetTempID: "agent3_check_loop", SourcePort: "output"},
			{SourceTempID: "agent3_check_loop", TargetTempID: "agent3_llm", SourcePort: "false"},          // More steps to configure
			{SourceTempID: "agent3_check_loop", TargetTempID: "agent4_merge_steps", SourcePort: "true"},   // All done, go to Agent 4
			{SourceTempID: "agent3_llm", TargetTempID: "agent3_process_config", SourcePort: "output"},
			{SourceTempID: "agent3_process_config", TargetTempID: "agent3_check_loop", SourcePort: "output"}, // Loop back
			// Agent 4: Validation & Final Assembly
			{SourceTempID: "agent4_merge_steps", TargetTempID: "agent4_llm", SourcePort: "output"},
			{SourceTempID: "agent4_llm", TargetTempID: "agent_create_project", SourcePort: "output"},
		},
	}
}
