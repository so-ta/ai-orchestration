package workflows

import "encoding/json"

func (r *Registry) registerBuilderWorkflows() {
	r.register(BuilderWorkflow())
}

// BuilderWorkflow is a unified AI Workflow Builder with 3 entry points:
// - hearing: Conducts interactive hearing to gather workflow requirements
// - construct: Builds workflow from gathered requirements (WorkflowSpec)
// - refine: Refines existing workflow based on user feedback
func BuilderWorkflow() *SystemWorkflowDefinition {
	return &SystemWorkflowDefinition{
		ID:          "a0000000-0000-0000-0000-000000000002",
		SystemSlug:  "ai-builder",
		Name:        "AI Workflow Builder",
		Description: "AI-assisted workflow building with interactive hearing, automatic construction, and refinement capabilities",
		Version:     2,
		IsSystem:    true,
		Steps: []SystemStepDefinition{
			// ============================
			// Hearing Entry Point (Y=40)
			// ============================
			{
				TempID:      "start_hearing",
				Name:        "Start: Hearing",
				Type:        "start",
				TriggerType: "internal",
				TriggerConfig: json.RawMessage(`{
					"entry_point": "hearing",
					"description": "Conduct interactive hearing to gather workflow requirements"
				}`),
				PositionX: 40,
				PositionY: 40,
				Config: json.RawMessage(`{
					"input_schema": {
						"type": "object",
						"required": ["session_id", "message"],
						"properties": {
							"session_id": {"type": "string", "title": "セッションID"},
							"message": {"type": "string", "title": "ユーザーメッセージ"},
							"tenant_id": {"type": "string", "title": "テナントID"},
							"user_id": {"type": "string", "title": "ユーザーID"}
						}
					}
				}`),
			},
			{
				TempID:    "hearing_get_context",
				Name:      "Get Session Context",
				Type:      "function",
				PositionX: 160,
				PositionY: 40,
				Config: json.RawMessage(`{
					"code": "const session = ctx.builderSessions.get(input.session_id); const blocks = ctx.blocks.list(); return { session: session, blocks: blocks.map(b => ({ slug: b.slug, name: b.name, description: b.description, category: b.category })), message: input.message, tenant_id: input.tenant_id, user_id: input.user_id };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"session": {"type": "object", "title": "セッション情報"},
							"blocks": {"type": "array", "title": "利用可能ブロック"},
							"message": {"type": "string", "title": "ユーザーメッセージ"},
							"tenant_id": {"type": "string"},
							"user_id": {"type": "string"}
						}
					}
				}`),
			},
			{
				TempID:    "hearing_build_prompt",
				Name:      "Build Hearing Prompt",
				Type:      "function",
				PositionX: 280,
				PositionY: 40,
				Config: json.RawMessage(`{
					"code": "const session = input.session; const phase = session?.hearing_phase || 'purpose'; const spec = session?.spec || {}; const messages = session?.messages || []; const historyText = messages.map(m => (m.role === 'user' ? 'ユーザー: ' : 'AI: ') + m.content).join('\\n'); const phasePrompts = { 'purpose': '【Phase: 目的・ゴール確認】\\nユーザーが作りたいワークフローの目的とゴールを確認してください。\\n- 何を達成したいのか\\n- 成功条件は何か\\n- 業務カテゴリ(sales/development/hr/finance/marketing/support/operations/personal/other)', 'conditions': '【Phase: 開始・終了条件】\\n開始トリガーと終了条件を確認してください。\\n- いつ開始するか(手動/定期実行/Webhook/イベント)\\n- スケジュール(cronや\"毎週月曜9時\"など)\\n- 何をもって完了とするか\\n- 成果物は何か', 'actors': '【Phase: 関与者・承認】\\n関与する人物と承認フローを確認してください。\\n- 誰が作業を実行するか\\n- 承認やレビューは必要か\\n- 承認者は誰か\\n- 差し戻し時の扱い', 'frequency': '【Phase: 実行頻度・期限】\\n実行頻度と期限を確認してください。\\n- どのくらいの頻度で実行するか\\n- 期限やSLAはあるか\\n- 緊急時の対応は必要か', 'integrations': '【Phase: ツール・システム連携】\\n使用するツールやシステムを確認してください。\\n- 利用するサービス(Slack/GitHub/Google Sheets等)\\n- 認証情報は設定済みか\\n- データの入力元・出力先', 'pain_points': '【Phase: 課題・困りごと】\\n現在の課題や困りごとを確認してください。\\n- 現在の手作業で困っていること\\n- エラーが起きやすい箇所\\n- 改善したいポイント', 'confirmation': '【Phase: 仮定条件の確認】\\nこれまでの会話から推測した仮定条件を確認してください。\\n- 不明点があれば仮定を提示\\n- ユーザーに確認を求める\\n- 全て確認できたらcompleted' }; const phaseGuide = phasePrompts[phase] || phasePrompts['purpose']; const blocksInfo = input.blocks.slice(0, 30).map(b => '- ' + b.slug + ': ' + b.name + ' (' + b.category + ')').join('\\n'); const prompt = 'あなたはワークフロービルダーAIです。ユーザーとの対話を通じてワークフロー要件をヒアリングします。\\n\\n' + phaseGuide + '\\n\\n## 現在のワークフロー仕様\\n' + JSON.stringify(spec, null, 2) + '\\n\\n## 会話履歴\\n' + (historyText || '(初回)') + '\\n\\n## ユーザーの最新メッセージ\\n' + input.message + '\\n\\n## 利用可能なブロック(参考)\\n' + blocksInfo + '\\n\\n## 出力形式(JSON)\\n{\\n  \"response\": \"ユーザーへの返答メッセージ\",\\n  \"extractedData\": { 「抽出した情報をWorkflowSpec形式で」 },\\n  \"suggestedQuestions\": [\"次に聞くべき質問1\", \"質問2\"],\\n  \"nextPhase\": \"purpose|conditions|actors|frequency|integrations|pain_points|confirmation|completed\",\\n  \"progress\": 0-100の進捗率\\n}\\n\\n重要:\\n- responseは親しみやすく丁寧な日本語で\\n- 一度に聞くのは1-2項目まで\\n- 情報が十分集まったら次のフェーズへ進む\\n- 全フェーズ完了でnextPhase=\"completed\"'; return { prompt: prompt, session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, current_phase: phase, current_spec: spec };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"prompt": {"type": "string"},
							"session_id": {"type": "string"},
							"tenant_id": {"type": "string"},
							"user_id": {"type": "string"},
							"current_phase": {"type": "string"},
							"current_spec": {"type": "object"}
						}
					}
				}`),
			},
			{
				TempID:    "hearing_llm",
				Name:      "Hearing LLM",
				Type:      "llm",
				PositionX: 400,
				PositionY: 40,
				Config: json.RawMessage(`{
					"model": "claude-3-haiku-20240307",
					"provider": "anthropic",
					"max_tokens": 2000,
					"temperature": 0.5,
					"user_prompt": "{{$.prompt}}",
					"system_prompt": "あなたは親切で専門的なワークフロービルダーAIです。ユーザーの要望を丁寧にヒアリングし、最適なワークフローを設計するための情報を収集します。常に有効なJSONで応答してください。"
				}`),
			},
			{
				TempID:    "hearing_parse",
				Name:      "Parse Hearing Response",
				Type:      "function",
				PositionX: 520,
				PositionY: 40,
				Config: json.RawMessage(`{
					"code": "try { let content = input.content || ''; if (content.startsWith('` + "```json" + `')) content = content.slice(7); if (content.startsWith('` + "```" + `')) content = content.slice(3); if (content.endsWith('` + "```" + `')) content = content.slice(0, -3); content = content.trim(); const result = JSON.parse(content); return { success: true, response: result.response || '', extractedData: result.extractedData || {}, suggestedQuestions: result.suggestedQuestions || [], nextPhase: result.nextPhase || 'purpose', progress: result.progress || 0 }; } catch (e) { return { success: false, error: 'Failed to parse LLM response: ' + e.message, response: input.content || '' }; }",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"success": {"type": "boolean"},
							"response": {"type": "string"},
							"extractedData": {"type": "object"},
							"suggestedQuestions": {"type": "array"},
							"nextPhase": {"type": "string"},
							"progress": {"type": "number"},
							"error": {"type": "string"}
						}
					}
				}`),
			},
			{
				TempID:    "hearing_update_session",
				Name:      "Update Builder Session",
				Type:      "function",
				PositionX: 640,
				PositionY: 40,
				Config: json.RawMessage(`{
					"code": "const sessionId = input.session_id; const tenantId = input.tenant_id; const parsed = input.parsed; const currentSpec = input.current_spec || {}; const mergedSpec = { ...currentSpec, ...parsed.extractedData }; ctx.builderSessions.update(sessionId, { hearing_phase: parsed.nextPhase, hearing_progress: parsed.progress, spec: mergedSpec }); ctx.builderSessions.addMessage(sessionId, { role: 'assistant', content: parsed.response, phase: parsed.nextPhase, suggested_questions: parsed.suggestedQuestions, extracted_data: parsed.extractedData }); return { session_id: sessionId, message: { content: parsed.response, suggested_questions: parsed.suggestedQuestions }, phase: parsed.nextPhase, progress: parsed.progress, complete: parsed.nextPhase === 'completed' };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"session_id": {"type": "string"},
							"message": {"type": "object"},
							"phase": {"type": "string"},
							"progress": {"type": "number"},
							"complete": {"type": "boolean"}
						}
					}
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
							"session_id": {"type": "string", "title": "セッションID"},
							"tenant_id": {"type": "string", "title": "テナントID"},
							"user_id": {"type": "string", "title": "ユーザーID"}
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
					"code": "const session = ctx.builderSessions.get(input.session_id); const blocks = ctx.blocks.list(); return { session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id, spec: session?.spec || {}, blocks: blocks.map(b => ({ slug: b.slug, name: b.name, description: b.description, category: b.category, input_schema: b.input_schema, output_schema: b.output_schema })) };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"session_id": {"type": "string"},
							"tenant_id": {"type": "string"},
							"user_id": {"type": "string"},
							"spec": {"type": "object"},
							"blocks": {"type": "array"}
						}
					}
				}`),
			},
			{
				TempID:    "construct_build_prompt",
				Name:      "Build Construction Prompt",
				Type:      "function",
				PositionX: 280,
				PositionY: 160,
				Config: json.RawMessage(`{
					"code": "const spec = input.spec; const blocksInfo = input.blocks.map(b => '- ' + b.slug + ': ' + b.name + ' (' + b.category + ') - ' + (b.description || '')).join('\\n'); const prompt = 'あなたはワークフロー設計AIです。以下の要件からワークフローを構築してください。\\n\\n## ワークフロー要件(WorkflowSpec)\\n' + JSON.stringify(spec, null, 2) + '\\n\\n## 利用可能なブロック\\n' + blocksInfo + '\\n\\n## ステップタイプ\\n- start: エントリーポイント(必須)\\n- llm: AI/LLM呼び出し\\n- tool: 外部アダプタ\\n- condition: 条件分岐(true/false)\\n- switch: 複数分岐\\n- map: 並列配列処理\\n- loop: ループ\\n- wait: 待機\\n- function: カスタムJavaScript\\n- human_in_loop: 人間承認\\n- log: デバッグログ\\n\\n## 出力形式(JSON)\\n{\\n  \"workflow_name\": \"ワークフロー名\",\\n  \"workflow_description\": \"説明\",\\n  \"steps\": [\\n    {\\n      \"temp_id\": \"step_1\",\\n      \"name\": \"ステップ名\",\\n      \"type\": \"start|llm|function|condition|...\",\\n      \"description\": \"ステップの説明\",\\n      \"config\": { ステップ固有の設定 },\\n      \"position_x\": 40,\\n      \"position_y\": 40,\\n      \"block_slug\": \"使用するブロックのslug(あれば)\",\\n      \"custom_required\": false,\\n      \"custom_reason\": \"カスタムが必要な理由(あれば)\"\\n    }\\n  ],\\n  \"edges\": [\\n    {\\n      \"source_temp_id\": \"step_1\",\\n      \"target_temp_id\": \"step_2\",\\n      \"source_port\": \"output\",\\n      \"target_port\": \"input\"\\n    }\\n  ],\\n  \"start_step_id\": \"step_1\",\\n  \"summary\": {\\n    \"total_steps\": 5,\\n    \"has_approval\": false,\\n    \"has_loop\": false,\\n    \"integrations_used\": [\"slack\", \"http\"],\\n    \"custom_blocks_needed\": []\\n  },\\n  \"warnings\": [\"注意事項があれば\"]\\n}\\n\\n重要:\\n- 必ずstartステップを含める\\n- プリセットブロックを最大限活用する\\n- カスタムが必要な場合は理由を明記\\n- position_xとposition_yは40の倍数で、見やすいレイアウトにする'; return { prompt: prompt, session_id: input.session_id, tenant_id: input.tenant_id, user_id: input.user_id };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"prompt": {"type": "string"},
							"session_id": {"type": "string"},
							"tenant_id": {"type": "string"},
							"user_id": {"type": "string"}
						}
					}
				}`),
			},
			{
				TempID:    "construct_llm",
				Name:      "Construct LLM",
				Type:      "llm",
				PositionX: 400,
				PositionY: 160,
				Config: json.RawMessage(`{
					"model": "claude-3-haiku-20240307",
					"provider": "anthropic",
					"max_tokens": 4000,
					"temperature": 0.3,
					"user_prompt": "{{$.prompt}}",
					"system_prompt": "あなたは専門的なワークフロー設計AIです。与えられた要件から最適なワークフロー構造を設計します。常に有効なJSONで応答し、実用的で効率的なワークフローを生成してください。"
				}`),
			},
			{
				TempID:    "construct_parse",
				Name:      "Parse Construction Result",
				Type:      "function",
				PositionX: 520,
				PositionY: 160,
				Config: json.RawMessage(`{
					"code": "try { let content = input.content || ''; if (content.startsWith('` + "```json" + `')) content = content.slice(7); if (content.startsWith('` + "```" + `')) content = content.slice(3); if (content.endsWith('` + "```" + `')) content = content.slice(0, -3); content = content.trim(); const result = JSON.parse(content); if (!result.steps || !Array.isArray(result.steps)) { return { success: false, error: 'Invalid workflow: missing steps array' }; } const validTypes = ['start', 'llm', 'tool', 'condition', 'switch', 'map', 'join', 'subflow', 'loop', 'wait', 'function', 'router', 'human_in_loop', 'filter', 'split', 'aggregate', 'error', 'note', 'log']; result.steps = result.steps.filter(s => validTypes.includes(s.type)); return { success: true, workflow: result }; } catch (e) { return { success: false, error: 'Failed to parse LLM response: ' + e.message }; }",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"success": {"type": "boolean"},
							"workflow": {"type": "object"},
							"error": {"type": "string"}
						}
					}
				}`),
			},
			{
				TempID:    "construct_create_project",
				Name:      "Create Project",
				Type:      "function",
				PositionX: 640,
				PositionY: 160,
				Config: json.RawMessage(`{
					"code": "const workflow = input.workflow; const sessionId = input.session_id; const tenantId = input.tenant_id; const userId = input.user_id; if (!workflow || !workflow.steps) { return { success: false, error: 'No workflow to create' }; } const project = ctx.projects.create({ tenant_id: tenantId, name: workflow.workflow_name || 'AI Generated Workflow', description: workflow.workflow_description || '', status: 'draft', created_by: userId }); const stepIdMap = {}; for (const step of workflow.steps) { const createdStep = ctx.steps.create({ project_id: project.id, name: step.name, type: step.type, config: step.config || {}, position_x: step.position_x || 40, position_y: step.position_y || 40, block_slug: step.block_slug }); stepIdMap[step.temp_id] = createdStep.id; } for (const edge of (workflow.edges || [])) { const sourceId = stepIdMap[edge.source_temp_id]; const targetId = stepIdMap[edge.target_temp_id]; if (sourceId && targetId) { ctx.edges.create({ project_id: project.id, source_step_id: sourceId, target_step_id: targetId, source_port: edge.source_port || 'output', target_port: edge.target_port || 'input' }); } } const startStepId = stepIdMap[workflow.start_step_id]; if (startStepId) { ctx.projects.update(project.id, { start_step_id: startStepId }); } ctx.builderSessions.update(sessionId, { status: 'reviewing', project_id: project.id }); return { success: true, project_id: project.id, summary: workflow.summary, warnings: workflow.warnings, custom_requirements: workflow.steps.filter(s => s.custom_required).map(s => ({ name: s.name, reason: s.custom_reason })) };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"success": {"type": "boolean"},
							"project_id": {"type": "string"},
							"summary": {"type": "object"},
							"warnings": {"type": "array"},
							"custom_requirements": {"type": "array"},
							"error": {"type": "string"}
						}
					}
				}`),
			},

			// ============================
			// Refine Entry Point (Y=280)
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
				PositionY: 280,
				Config: json.RawMessage(`{
					"input_schema": {
						"type": "object",
						"required": ["session_id", "feedback"],
						"properties": {
							"session_id": {"type": "string", "title": "セッションID"},
							"project_id": {"type": "string", "title": "プロジェクトID"},
							"feedback": {"type": "string", "title": "ユーザーフィードバック"},
							"tenant_id": {"type": "string", "title": "テナントID"},
							"user_id": {"type": "string", "title": "ユーザーID"}
						}
					}
				}`),
			},
			{
				TempID:    "refine_get_workflow",
				Name:      "Get Current Workflow",
				Type:      "function",
				PositionX: 160,
				PositionY: 280,
				Config: json.RawMessage(`{
					"code": "const project = ctx.projects.get(input.project_id); const steps = ctx.steps.listByProject(input.project_id); const edges = ctx.edges.listByProject(input.project_id); const blocks = ctx.blocks.list(); return { session_id: input.session_id, project_id: input.project_id, tenant_id: input.tenant_id, user_id: input.user_id, feedback: input.feedback, current_workflow: { name: project?.name, description: project?.description, steps: steps, edges: edges }, blocks: blocks.map(b => ({ slug: b.slug, name: b.name, description: b.description, category: b.category })) };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"session_id": {"type": "string"},
							"project_id": {"type": "string"},
							"tenant_id": {"type": "string"},
							"user_id": {"type": "string"},
							"feedback": {"type": "string"},
							"current_workflow": {"type": "object"},
							"blocks": {"type": "array"}
						}
					}
				}`),
			},
			{
				TempID:    "refine_build_prompt",
				Name:      "Build Refine Prompt",
				Type:      "function",
				PositionX: 280,
				PositionY: 280,
				Config: json.RawMessage(`{
					"code": "const workflow = input.current_workflow; const stepsInfo = (workflow.steps || []).map(s => '- ' + s.name + ' (type: ' + s.type + ', id: ' + s.id + ')').join('\\n'); const edgesInfo = (workflow.edges || []).map(e => '- ' + e.source_step_id + ' -> ' + e.target_step_id).join('\\n'); const blocksInfo = input.blocks.slice(0, 30).map(b => '- ' + b.slug + ': ' + b.name).join('\\n'); const prompt = 'あなたはワークフロー修正AIです。ユーザーのフィードバックに基づいてワークフローを修正してください。\\n\\n## 現在のワークフロー\\n名前: ' + workflow.name + '\\n説明: ' + workflow.description + '\\n\\n### ステップ\\n' + stepsInfo + '\\n\\n### エッジ(接続)\\n' + edgesInfo + '\\n\\n## ユーザーフィードバック\\n' + input.feedback + '\\n\\n## 利用可能なブロック\\n' + blocksInfo + '\\n\\n## 出力形式(JSON)\\n{\\n  \"changes\": [\\n    {\\n      \"type\": \"add_step|remove_step|modify_step|add_edge|remove_edge|modify_config\",\\n      \"step_id\": \"既存ステップID(修正・削除時)\",\\n      \"step\": { 新規ステップ情報(追加時) },\\n      \"edge\": { エッジ情報(エッジ操作時) },\\n      \"config\": { 設定変更(設定修正時) },\\n      \"reason\": \"変更理由\"\\n    }\\n  ],\\n  \"summary\": \"変更内容の要約\",\\n  \"response\": \"ユーザーへの返答メッセージ\"\\n}'; return { prompt: prompt, session_id: input.session_id, project_id: input.project_id, tenant_id: input.tenant_id, user_id: input.user_id, current_workflow: workflow };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"prompt": {"type": "string"},
							"session_id": {"type": "string"},
							"project_id": {"type": "string"},
							"tenant_id": {"type": "string"},
							"user_id": {"type": "string"},
							"current_workflow": {"type": "object"}
						}
					}
				}`),
			},
			{
				TempID:    "refine_llm",
				Name:      "Refine LLM",
				Type:      "llm",
				PositionX: 400,
				PositionY: 280,
				Config: json.RawMessage(`{
					"model": "claude-3-haiku-20240307",
					"provider": "anthropic",
					"max_tokens": 2000,
					"temperature": 0.3,
					"user_prompt": "{{$.prompt}}",
					"system_prompt": "あなたはワークフロー修正AIです。ユーザーのフィードバックを正確に理解し、最小限の変更で要望を実現します。常に有効なJSONで応答してください。"
				}`),
			},
			{
				TempID:    "refine_parse",
				Name:      "Parse Refine Result",
				Type:      "function",
				PositionX: 520,
				PositionY: 280,
				Config: json.RawMessage(`{
					"code": "try { let content = input.content || ''; if (content.startsWith('` + "```json" + `')) content = content.slice(7); if (content.startsWith('` + "```" + `')) content = content.slice(3); if (content.endsWith('` + "```" + `')) content = content.slice(0, -3); content = content.trim(); const result = JSON.parse(content); return { success: true, changes: result.changes || [], summary: result.summary || '', response: result.response || '' }; } catch (e) { return { success: false, error: 'Failed to parse LLM response: ' + e.message }; }",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"success": {"type": "boolean"},
							"changes": {"type": "array"},
							"summary": {"type": "string"},
							"response": {"type": "string"},
							"error": {"type": "string"}
						}
					}
				}`),
			},
			{
				TempID:    "refine_apply_changes",
				Name:      "Apply Changes",
				Type:      "function",
				PositionX: 640,
				PositionY: 280,
				Config: json.RawMessage(`{
					"code": "const projectId = input.project_id; const changes = input.changes || []; const appliedChanges = []; for (const change of changes) { try { switch (change.type) { case 'add_step': const newStep = ctx.steps.create({ project_id: projectId, name: change.step.name, type: change.step.type, config: change.step.config || {}, position_x: change.step.position_x || 40, position_y: change.step.position_y || 40 }); appliedChanges.push({ type: 'add_step', step_id: newStep.id, name: change.step.name }); break; case 'remove_step': ctx.steps.delete(change.step_id); appliedChanges.push({ type: 'remove_step', step_id: change.step_id }); break; case 'modify_step': case 'modify_config': ctx.steps.update(change.step_id, change.config || change.step); appliedChanges.push({ type: 'modify_step', step_id: change.step_id }); break; case 'add_edge': ctx.edges.create({ project_id: projectId, source_step_id: change.edge.source_step_id, target_step_id: change.edge.target_step_id, source_port: change.edge.source_port || 'output', target_port: change.edge.target_port || 'input' }); appliedChanges.push({ type: 'add_edge' }); break; case 'remove_edge': ctx.edges.delete(change.edge.id); appliedChanges.push({ type: 'remove_edge' }); break; } } catch (e) { appliedChanges.push({ type: change.type, error: e.message }); } } ctx.projects.incrementVersion(projectId); ctx.builderSessions.addMessage(input.session_id, { role: 'assistant', content: input.response }); return { success: true, applied_changes: appliedChanges, summary: input.summary, response: input.response };",
					"language": "javascript",
					"output_schema": {
						"type": "object",
						"properties": {
							"success": {"type": "boolean"},
							"applied_changes": {"type": "array"},
							"summary": {"type": "string"},
							"response": {"type": "string"}
						}
					}
				}`),
			},
		},
		Edges: []SystemEdgeDefinition{
			// Hearing flow
			{SourceTempID: "start_hearing", TargetTempID: "hearing_get_context", SourcePort: "output"},
			{SourceTempID: "hearing_get_context", TargetTempID: "hearing_build_prompt", SourcePort: "output"},
			{SourceTempID: "hearing_build_prompt", TargetTempID: "hearing_llm", SourcePort: "output"},
			{SourceTempID: "hearing_llm", TargetTempID: "hearing_parse", SourcePort: "output"},
			{SourceTempID: "hearing_parse", TargetTempID: "hearing_update_session", SourcePort: "output"},

			// Construct flow
			{SourceTempID: "start_construct", TargetTempID: "construct_get_spec", SourcePort: "output"},
			{SourceTempID: "construct_get_spec", TargetTempID: "construct_build_prompt", SourcePort: "output"},
			{SourceTempID: "construct_build_prompt", TargetTempID: "construct_llm", SourcePort: "output"},
			{SourceTempID: "construct_llm", TargetTempID: "construct_parse", SourcePort: "output"},
			{SourceTempID: "construct_parse", TargetTempID: "construct_create_project", SourcePort: "output"},

			// Refine flow
			{SourceTempID: "start_refine", TargetTempID: "refine_get_workflow", SourcePort: "output"},
			{SourceTempID: "refine_get_workflow", TargetTempID: "refine_build_prompt", SourcePort: "output"},
			{SourceTempID: "refine_build_prompt", TargetTempID: "refine_llm", SourcePort: "output"},
			{SourceTempID: "refine_llm", TargetTempID: "refine_parse", SourcePort: "output"},
			{SourceTempID: "refine_parse", TargetTempID: "refine_apply_changes", SourcePort: "output"},
		},
	}
}
