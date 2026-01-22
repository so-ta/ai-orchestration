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
		Name:        LText("Parallel", "並列"),
		Description: LText("Execute multiple independent flows concurrently within the group", "グループ内で複数の独立したフローを同時に実行"),
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryControl,
		Icon:        "git-branch",
		ConfigSchema: LSchema(`{
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
		}`, `{
			"type": "object",
			"properties": {
				"max_concurrent": {
					"type": "integer",
					"title": "最大同時実行数",
					"description": "最大同時実行数（0 = 無制限）",
					"default": 0
				},
				"fail_fast": {
					"type": "boolean",
					"title": "即座に失敗",
					"description": "最初の失敗で全フローを停止",
					"default": false
				}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("out", "Complete", "完了", "All flows completed", "全フロー完了", true),
			LPortWithDesc("error", "Error", "エラー", "Error output", "エラー出力", false),
		},
		Code: `// Parallel execution is handled by the engine
// pre_process: transforms external input to internal input
// post_process: transforms internal outputs to external output
return input;`,
		UIConfig: LSchema(`{
			"icon": "git-branch",
			"color": "#3B82F6",
			"isContainer": true
		}`, `{
			"icon": "git-branch",
			"color": "#3B82F6",
			"isContainer": true
		}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("PAR_001", "FLOW_FAILED", "フロー失敗", "One or more flows failed", "1つ以上のフローが失敗しました", false),
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
		Name:        LText("Try-Catch", "Try-Catch"),
		Description: LText("Execute body with error handling and retry support", "エラーハンドリングとリトライサポート付きで本体を実行"),
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryControl,
		Icon:        "shield",
		ConfigSchema: LSchema(`{
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
		}`, `{
			"type": "object",
			"properties": {
				"retry_count": {
					"type": "integer",
					"title": "リトライ回数",
					"description": "エラー時のリトライ回数（デフォルト: 0）",
					"default": 0,
					"minimum": 0,
					"maximum": 10
				},
				"retry_delay_ms": {
					"type": "integer",
					"title": "リトライ遅延 (ミリ秒)",
					"description": "リトライ間の遅延時間（ミリ秒）",
					"default": 1000,
					"minimum": 0
				}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("out", "Success", "成功", "Successful execution", "実行成功", true),
			LPortWithDesc("error", "Error", "エラー", "Error output", "エラー出力", false),
		},
		Code: `// Try-catch execution is handled by the engine
// Body is executed with retry support
// Errors are routed to error port
return input;`,
		UIConfig: LSchema(`{
			"icon": "shield",
			"color": "#EF4444",
			"isContainer": true
		}`, `{
			"icon": "shield",
			"color": "#EF4444",
			"isContainer": true
		}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("TRY_001", "MAX_RETRIES", "最大リトライ超過", "Maximum retries exceeded", "最大リトライ回数を超過しました", false),
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
		Name:        LText("For Each", "繰り返し"),
		Description: LText("Iterate over array elements, executing the same process for each", "配列要素を反復し、各要素に対して同じ処理を実行"),
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryControl,
		Icon:        "repeat",
		ConfigSchema: LSchema(`{
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
		}`, `{
			"type": "object",
			"properties": {
				"input_path": {
					"type": "string",
					"title": "入力パス",
					"description": "入力内の配列へのJSONPath（デフォルト: $.items）",
					"default": "$.items"
				},
				"parallel": {
					"type": "boolean",
					"title": "並列実行",
					"description": "反復を並列で実行",
					"default": false
				},
				"max_workers": {
					"type": "integer",
					"title": "最大ワーカー数",
					"description": "最大並列ワーカー数（0 = 無制限）",
					"default": 0
				}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("out", "Complete", "完了", "All iterations completed", "全反復完了", true),
			LPortWithDesc("error", "Error", "エラー", "Error output", "エラー出力", false),
		},
		Code: `// ForEach execution is handled by the engine
// Each iteration receives: { item, index, context }
// Results are aggregated into an array
return input;`,
		UIConfig: LSchema(`{
			"icon": "repeat",
			"color": "#8B5CF6",
			"isContainer": true
		}`, `{
			"icon": "repeat",
			"color": "#8B5CF6",
			"isContainer": true
		}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("FOR_001", "ITERATION_FAILED", "反復失敗", "One or more iterations failed", "1つ以上の反復が失敗しました", false),
			LError("FOR_002", "EMPTY_INPUT", "入力が空", "Input array is empty", "入力配列が空です", false),
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
		Name:        LText("While", "While"),
		Description: LText("Repeat body execution while condition is true", "条件が真の間、本体を繰り返し実行"),
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryControl,
		Icon:        "rotate-cw",
		ConfigSchema: LSchema(`{
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
		}`, `{
			"type": "object",
			"properties": {
				"condition": {
					"type": "string",
					"title": "条件",
					"description": "条件式（例: $.counter < $.target）"
				},
				"max_iterations": {
					"type": "integer",
					"title": "最大反復回数",
					"description": "無限ループ防止の安全制限",
					"default": 100,
					"minimum": 1,
					"maximum": 10000
				},
				"do_while": {
					"type": "boolean",
					"title": "Do-Whileモード",
					"description": "条件チェック前に少なくとも1回は本体を実行",
					"default": false
				}
			},
			"required": ["condition"]
		}`),
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("out", "Done", "完了", "Loop completed", "ループ完了", true),
			LPortWithDesc("error", "Error", "エラー", "Error output", "エラー出力", false),
		},
		Code: `// While execution is handled by the engine
// Body output becomes next iteration input
// Loop exits when condition is false
return input;`,
		UIConfig: LSchema(`{
			"icon": "rotate-cw",
			"color": "#F59E0B",
			"isContainer": true
		}`, `{
			"icon": "rotate-cw",
			"color": "#F59E0B",
			"isContainer": true
		}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("WHL_001", "MAX_ITERATIONS", "最大反復超過", "Maximum iterations exceeded", "最大反復回数を超過しました", false),
			LError("WHL_002", "CONDITION_ERROR", "条件エラー", "Failed to evaluate condition", "条件の評価に失敗しました", false),
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
		Name:        LText("Agent", "エージェント"),
		Description: LText("AI agent with ReAct loop - child steps become callable tools", "ReActループを持つAIエージェント - 子ステップが呼び出し可能なツールになります"),
		Category:    domain.BlockCategoryAI,
		Subcategory: domain.BlockSubcategoryAgent,
		Icon:        "bot",
		ConfigSchema: LSchema(`{
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
		}`, `{
			"type": "object",
			"required": ["provider", "model", "system_prompt"],
			"properties": {
				"provider": {
					"type": "string",
					"title": "プロバイダー",
					"enum": ["openai", "anthropic"],
					"default": "anthropic",
					"description": "LLMプロバイダー"
				},
				"model": {
					"type": "string",
					"title": "モデル",
					"default": "claude-sonnet-4-20250514",
					"description": "モデルID（例: claude-sonnet-4-20250514, gpt-4）"
				},
				"system_prompt": {
					"type": "string",
					"title": "システムプロンプト",
					"maxLength": 50000,
					"description": "エージェントの動作と機能を定義するシステムプロンプト"
				},
				"max_iterations": {
					"type": "integer",
					"title": "最大反復回数",
					"default": 10,
					"minimum": 1,
					"maximum": 50,
					"description": "ReActループの最大反復回数"
				},
				"temperature": {
					"type": "number",
					"title": "温度",
					"default": 0.7,
					"minimum": 0,
					"maximum": 2,
					"description": "LLM温度（0-2）"
				},
				"tool_choice": {
					"type": "string",
					"title": "ツール選択",
					"enum": ["auto", "none", "required"],
					"default": "auto",
					"description": "エージェントがツールを使用する方法"
				},
				"enable_memory": {
					"type": "boolean",
					"title": "メモリ有効化",
					"default": false,
					"description": "実行間で会話メモリを有効にする"
				},
				"memory_window": {
					"type": "integer",
					"title": "メモリウィンドウ",
					"default": 20,
					"minimum": 1,
					"maximum": 100,
					"description": "メモリに保持するメッセージ数"
				}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("out", "Response", "応答", "Agent's final response", "エージェントの最終応答", true),
			LPortWithDesc("error", "Error", "エラー", "Error output", "エラー出力", false),
		},
		Code: `// Agent execution is handled by the engine's executeAgent()
// Child steps become tools that the agent can call
return input;`,
		UIConfig: LSchema(`{
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
		}`, `{
			"icon": "bot",
			"color": "#10B981",
			"isContainer": true,
			"groups": [
				{"id": "model", "icon": "robot", "title": "モデル設定"},
				{"id": "agent", "icon": "bot", "title": "エージェント設定"},
				{"id": "memory", "icon": "database", "title": "メモリ設定"}
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
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("AGENT_001", "MAX_ITERATIONS", "最大反復超過", "Agent reached maximum iterations", "エージェントが最大反復回数に達しました", false),
			LError("AGENT_002", "TOOL_ERROR", "ツールエラー", "Tool execution failed", "ツールの実行に失敗しました", true),
			LError("AGENT_003", "LLM_ERROR", "LLMエラー", "LLM API error", "LLM APIエラー", true),
		},
		RequiredCredentials: json.RawMessage(`[{"name": "llm_api_key", "type": "api_key", "scope": "system", "required": true, "description": "LLM Provider API Key"}]`),
		Enabled:             true,
		GroupKind:           domain.BlockGroupKindAgent,
		IsContainer:         true,
	}
}
