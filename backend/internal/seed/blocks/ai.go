package blocks

import (
	"encoding/json"

	"github.com/souta/ai-orchestration/internal/domain"
)

func (r *Registry) registerAIBlocks() {
	r.register(LLMBlock())
	r.register(LLMJSONBlock())
	r.register(LLMStructuredBlock())
	r.register(RouterBlock())
	r.register(MemoryBufferBlock())
}

func LLMBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "llm",
		Version:     1,
		Name:        LText("LLM", "LLM"),
		Description: LText("Execute LLM prompts with various providers", "様々なプロバイダーでLLMプロンプトを実行"),
		Category:    domain.BlockCategoryAI,
		Subcategory: domain.BlockSubcategoryChat,
		Icon:        "brain",
		ConfigSchema: LSchema(`{
			"type": "object",
			"required": ["provider", "model", "user_prompt"],
			"properties": {
				"model": {"type": "string", "title": "Model"},
				"provider": {
					"enum": ["openai", "anthropic", "mock"],
					"type": "string",
					"title": "Provider",
					"default": "openai"
				},
				"max_tokens": {"type": "integer", "default": 4096, "maximum": 128000},
				"temperature": {"type": "number", "default": 0.7, "maximum": 2},
				"user_prompt": {"type": "string", "maxLength": 50000},
				"system_prompt": {"type": "string", "maxLength": 10000},
				"enable_error_port": {
					"type": "boolean",
					"title": "Enable Error Port",
					"description": "Output to dedicated error port on error",
					"default": false
				}
			}
		}`, `{
			"type": "object",
			"required": ["provider", "model", "user_prompt"],
			"properties": {
				"model": {"type": "string", "title": "モデル"},
				"provider": {
					"enum": ["openai", "anthropic", "mock"],
					"type": "string",
					"title": "プロバイダー",
					"default": "openai"
				},
				"max_tokens": {"type": "integer", "default": 4096, "maximum": 128000},
				"temperature": {"type": "number", "default": 0.7, "maximum": 2},
				"user_prompt": {"type": "string", "maxLength": 50000},
				"system_prompt": {"type": "string", "maxLength": 10000},
				"enable_error_port": {
					"type": "boolean",
					"title": "エラーハンドルを有効化",
					"description": "エラー発生時に専用のエラーポートに出力します",
					"default": false
				}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithSchema("output", "Output", "出力", "LLM response", "LLMの応答", true,
				json.RawMessage(`{"type": "object", "properties": {"content": {"type": "string"}, "tokens_used": {"type": "number"}}}`)),
		},
		Code: `
const prompt = renderTemplate(config.user_prompt || '', input);
const systemPrompt = config.system_prompt || '';
const response = ctx.llm.chat(config.provider, config.model, {
    messages: [
        ...(systemPrompt ? [{ role: 'system', content: systemPrompt }] : []),
        { role: 'user', content: prompt }
    ],
    temperature: config.temperature ?? 0.7,
    maxTokens: config.max_tokens ?? 1000
});
return {
    content: response.content,
    usage: response.usage
};
`,
		UIConfig: LSchema(`{
			"icon": "brain",
			"color": "#8B5CF6",
			"groups": [
				{"id": "model", "icon": "robot", "title": "Model Settings"},
				{"id": "prompt", "icon": "message", "title": "Prompt"}
			],
			"fieldGroups": {
				"model": "model",
				"provider": "model",
				"user_prompt": "prompt"
			},
			"fieldOverrides": {
				"user_prompt": {"rows": 8, "widget": "textarea"}
			}
		}`, `{
			"icon": "brain",
			"color": "#8B5CF6",
			"groups": [
				{"id": "model", "icon": "robot", "title": "モデル設定"},
				{"id": "prompt", "icon": "message", "title": "プロンプト"}
			],
			"fieldGroups": {
				"model": "model",
				"provider": "model",
				"user_prompt": "prompt"
			},
			"fieldOverrides": {
				"user_prompt": {"rows": 8, "widget": "textarea"}
			}
		}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("LLM_001", "RATE_LIMIT", "レート制限", "Rate limit exceeded", "レート制限を超過しました", true),
			LError("LLM_002", "INVALID_MODEL", "無効なモデル", "Invalid model specified", "無効なモデルが指定されました", false),
			LError("LLM_003", "TOKEN_LIMIT", "トークン制限", "Token limit exceeded", "トークン制限を超過しました", false),
			LError("LLM_004", "API_ERROR", "APIエラー", "LLM API error", "LLM APIエラー", true),
		},
		RequiredCredentials: json.RawMessage(`[{"name": "llm_api_key", "type": "api_key", "scope": "system", "required": true, "description": "LLM Provider API Key"}]`),
		Enabled:             true,
		TestCases: []BlockTestCase{
			{
				Name:   "basic LLM call",
				Input:  map[string]interface{}{"message": "Hello"},
				Config: map[string]interface{}{"provider": "mock", "model": "test", "user_prompt": "Say hello"},
				ExpectedOutput: map[string]interface{}{
					"content": "Mock LLM response",
				},
			},
		},
	}
}

// LLMJSONBlock provides LLM with automatic JSON output parsing
func LLMJSONBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:            "llm-json",
		Version:         1,
		Name:            LText("LLM (JSON)", "LLM (JSON)"),
		Description:     LText("LLM with automatic JSON output parsing", "自動JSON出力パース付きLLM"),
		Category:        domain.BlockCategoryAI,
		Subcategory:     domain.BlockSubcategoryChat,
		Icon:            "braces",
		ParentBlockSlug: "llm",
		ConfigDefaults: json.RawMessage(`{
			"temperature": 0.3
		}`),
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"json_instruction": {
					"type": "string",
					"title": "JSON Instruction",
					"default": "Always respond with valid JSON only. No markdown, no explanation.",
					"description": "JSON format instruction added to system prompt"
				},
				"strict_parse": {
					"type": "boolean",
					"title": "Strict Parse",
					"default": false,
					"description": "Throw error on parse failure (if false, returns {error: ...})"
				},
				"preserve_input": {
					"type": "boolean",
					"title": "Preserve Input",
					"default": false,
					"description": "Merge original input data with LLM output"
				}
			}
		}`, `{
			"type": "object",
			"properties": {
				"json_instruction": {
					"type": "string",
					"title": "JSON指示",
					"default": "Always respond with valid JSON only. No markdown, no explanation.",
					"description": "システムプロンプトに追加されるJSON形式の指示"
				},
				"strict_parse": {
					"type": "boolean",
					"title": "厳密パース",
					"default": false,
					"description": "パース失敗時にエラーを投げる（falseの場合は{error: ...}を返す）"
				},
				"preserve_input": {
					"type": "boolean",
					"title": "入力を保持",
					"default": false,
					"description": "元の入力データをLLM出力とマージして保持する"
				}
			}
		}`),
		PreProcess: `
// Append JSON instruction to system prompt
const jsonInstruction = config.json_instruction || 'Always respond with valid JSON only. No markdown, no explanation.';
if (config.system_prompt) {
    config.system_prompt = config.system_prompt + '\n\n' + jsonInstruction;
} else {
    config.system_prompt = jsonInstruction;
}
// Store original input for preserve_input option
// Note: We store it in __preserved_for_postprocess field of the result
// so it survives the internal LLM step execution and is available in PostProcess
if (config.preserve_input) {
    const preserved = {};
    for (const key in input) {
        if (!key.startsWith('__')) {
            preserved[key] = input[key];
        }
    }
    // Store in result so it gets passed through
    return { ...input, __preserved_for_postprocess: preserved };
}
return input;
`,
		PostProcess: "// Parse JSON from LLM response\n" +
			"let content = input.content || '';\n" +
			"\n" +
			"// Strip Markdown code blocks\n" +
			"if (content.startsWith('```json')) {\n" +
			"    content = content.slice(7);\n" +
			"} else if (content.startsWith('```')) {\n" +
			"    content = content.slice(3);\n" +
			"}\n" +
			"if (content.endsWith('```')) {\n" +
			"    content = content.slice(0, -3);\n" +
			"}\n" +
			"content = content.trim();\n" +
			"\n" +
			"// Get preserved input from PreProcess (passed through executor)\n" +
			"const preserved = input.__preserved_for_postprocess || {};\n" +
			"\n" +
			"try {\n" +
			"    const parsed = JSON.parse(content);\n" +
			"    // Merge with preserved input if enabled (LLM output takes precedence)\n" +
			"    return {\n" +
			"        ...preserved,\n" +
			"        ...parsed,\n" +
			"        __raw: input.content,\n" +
			"        __usage: input.usage\n" +
			"    };\n" +
			"} catch (e) {\n" +
			"    if (config.strict_parse) {\n" +
			"        throw new Error('[LLM_JSON_001] Failed to parse JSON: ' + e.message);\n" +
			"    }\n" +
			"    return {\n" +
			"        ...preserved,\n" +
			"        error: 'JSON parse failed: ' + e.message,\n" +
			"        __raw: input.content,\n" +
			"        __usage: input.usage\n" +
			"    };\n" +
			"}\n",
		UIConfig: LSchema(`{
			"icon": "braces",
			"color": "#F59E0B",
			"groups": [
				{"id": "model", "icon": "robot", "title": "Model Settings"},
				{"id": "prompt", "icon": "message", "title": "Prompt"},
				{"id": "json", "icon": "braces", "title": "JSON Settings"}
			],
			"fieldGroups": {
				"model": "model",
				"provider": "model",
				"user_prompt": "prompt",
				"system_prompt": "prompt",
				"json_instruction": "json",
				"strict_parse": "json"
			},
			"fieldOverrides": {
				"user_prompt": {"rows": 8, "widget": "textarea"}
			}
		}`, `{
			"icon": "braces",
			"color": "#F59E0B",
			"groups": [
				{"id": "model", "icon": "robot", "title": "モデル設定"},
				{"id": "prompt", "icon": "message", "title": "プロンプト"},
				{"id": "json", "icon": "braces", "title": "JSON設定"}
			],
			"fieldGroups": {
				"model": "model",
				"provider": "model",
				"user_prompt": "prompt",
				"system_prompt": "prompt",
				"json_instruction": "json",
				"strict_parse": "json"
			},
			"fieldOverrides": {
				"user_prompt": {"rows": 8, "widget": "textarea"}
			}
		}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("LLM_JSON_001", "PARSE_FAILED", "パース失敗", "Failed to parse LLM response as JSON", "LLMの応答をJSONとしてパースできませんでした", false),
		},
		Enabled: true,
		TestCases: []BlockTestCase{
			{
				Name:   "valid JSON response",
				Input:  map[string]interface{}{},
				Config: map[string]interface{}{"provider": "mock", "model": "test", "user_prompt": "Return JSON"},
				ExpectedOutput: map[string]interface{}{
					"__raw": "Mock LLM response",
				},
			},
		},
	}
}

// LLMStructuredBlock provides LLM with schema-driven structured output
func LLMStructuredBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:            "llm-structured",
		Version:         1,
		Name:            LText("LLM (Structured)", "LLM (構造化出力)"),
		Description:     LText("LLM with schema-driven structured output and validation", "スキーマ駆動の構造化出力と検証付きLLM"),
		Category:        domain.BlockCategoryAI,
		Subcategory:     domain.BlockSubcategoryChat,
		Icon:            "layout-template",
		ParentBlockSlug: "llm-json",
		ConfigDefaults: json.RawMessage(`{
			"strict_parse": true,
			"validate_output": true,
			"include_examples": true
		}`),
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"output_schema": {
					"type": "object",
					"title": "Output Schema",
					"x-ui-widget": "output-schema",
					"description": "JSON Schema for expected output"
				},
				"validate_output": {
					"type": "boolean",
					"title": "Validate Output",
					"default": true,
					"description": "Validate output against schema"
				},
				"include_examples": {
					"type": "boolean",
					"title": "Include Examples",
					"default": true,
					"description": "Include schema examples in prompt"
				}
			}
		}`, `{
			"type": "object",
			"properties": {
				"output_schema": {
					"type": "object",
					"title": "出力スキーマ",
					"x-ui-widget": "output-schema",
					"description": "期待する出力のJSON Schema"
				},
				"validate_output": {
					"type": "boolean",
					"title": "出力検証",
					"default": true,
					"description": "スキーマに基づいて出力を検証する"
				},
				"include_examples": {
					"type": "boolean",
					"title": "例を含める",
					"default": true,
					"description": "プロンプトにスキーマの例を含める"
				}
			}
		}`),
		PreProcess: `
// Build schema instruction for system prompt
if (config.output_schema && config.output_schema.properties) {
    const schema = config.output_schema;
    const fields = Object.entries(schema.properties).map(function(entry) {
        const name = entry[0];
        const prop = entry[1];
        const required = (schema.required || []).includes(name) ? ' (required)' : '';
        const type = prop.type || 'any';
        const desc = prop.description ? ': ' + prop.description : '';
        return '  - ' + name + ' (' + type + ')' + required + desc;
    }).join('\n');

    let schemaInstruction = '\n\n## Required Output Format\nRespond with a JSON object containing these fields:\n' + fields;

    // Generate example if enabled
    if (config.include_examples !== false) {
        const example = {};
        for (const entry of Object.entries(schema.properties)) {
            const name = entry[0];
            const prop = entry[1];
            if (prop.type === 'string') example[name] = prop.title || 'string value';
            else if (prop.type === 'number') example[name] = 0;
            else if (prop.type === 'boolean') example[name] = true;
            else if (prop.type === 'array') example[name] = [];
            else if (prop.type === 'object') example[name] = {};
            else example[name] = null;
        }
        schemaInstruction = schemaInstruction + '\n\nExample:\n' + JSON.stringify(example, null, 2);
    }

    // Update both json_instruction and system_prompt to ensure schema is visible to LLM
    config.json_instruction = (config.json_instruction || 'Always respond with valid JSON only. No markdown, no explanation.') + schemaInstruction;
    config.system_prompt = (config.system_prompt || '') + schemaInstruction;
}
return input;
`,
		PostProcess: `
// Validate output against schema if enabled
if (config.validate_output !== false && config.output_schema && config.output_schema.required) {
    const required = config.output_schema.required;
    const missing = required.filter(function(field) {
        return input[field] === undefined;
    });
    if (missing.length > 0) {
        throw new Error('[LLM_STRUCT_001] Missing required fields: ' + missing.join(', '));
    }
}

// Type coercion for schema-defined fields
if (config.output_schema && config.output_schema.properties) {
    for (const entry of Object.entries(config.output_schema.properties)) {
        const name = entry[0];
        const prop = entry[1];
        if (input[name] !== undefined && prop.type === 'array' && !Array.isArray(input[name])) {
            input[name] = [input[name]];
        }
    }
}

return input;
`,
		UIConfig: LSchema(`{
			"icon": "layout-template",
			"color": "#8B5CF6",
			"groups": [
				{"id": "model", "icon": "robot", "title": "Model Settings"},
				{"id": "prompt", "icon": "message", "title": "Prompt"},
				{"id": "schema", "icon": "braces", "title": "Output Schema"}
			],
			"fieldGroups": {
				"model": "model",
				"provider": "model",
				"user_prompt": "prompt",
				"system_prompt": "prompt",
				"output_schema": "schema",
				"validate_output": "schema",
				"include_examples": "schema"
			},
			"fieldOverrides": {
				"user_prompt": {"rows": 8, "widget": "textarea"}
			}
		}`, `{
			"icon": "layout-template",
			"color": "#8B5CF6",
			"groups": [
				{"id": "model", "icon": "robot", "title": "モデル設定"},
				{"id": "prompt", "icon": "message", "title": "プロンプト"},
				{"id": "schema", "icon": "braces", "title": "出力スキーマ"}
			],
			"fieldGroups": {
				"model": "model",
				"provider": "model",
				"user_prompt": "prompt",
				"system_prompt": "prompt",
				"output_schema": "schema",
				"validate_output": "schema",
				"include_examples": "schema"
			},
			"fieldOverrides": {
				"user_prompt": {"rows": 8, "widget": "textarea"}
			}
		}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("LLM_STRUCT_001", "VALIDATION_FAILED", "検証失敗", "Output validation failed - missing required fields", "出力検証に失敗しました - 必須フィールドがありません", false),
		},
		Enabled: true,
		TestCases: []BlockTestCase{
			{
				Name:  "structured output with schema",
				Input: map[string]interface{}{},
				Config: map[string]interface{}{
					"provider":    "mock",
					"model":       "test",
					"user_prompt": "Generate structured output",
					"output_schema": map[string]interface{}{
						"type":     "object",
						"required": []string{"name"},
						"properties": map[string]interface{}{
							"name": map[string]interface{}{"type": "string"},
						},
					},
				},
				ExpectedOutput: map[string]interface{}{
					"__raw": "Mock LLM response",
				},
			},
		},
	}
}

// MemoryBufferBlock provides conversation memory management for agents
func MemoryBufferBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "memory-buffer",
		Version:     1,
		Name:        LText("Memory Buffer", "メモリバッファ"),
		Description: LText("Manage conversation memory with sliding window", "スライディングウィンドウで会話メモリを管理"),
		Category:    domain.BlockCategoryAI,
		Subcategory: domain.BlockSubcategoryChat,
		Icon:        "database",
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"window_size": {
					"type": "integer",
					"title": "Window Size",
					"default": 20,
					"minimum": 1,
					"maximum": 100,
					"description": "Maximum number of messages to keep"
				},
				"memory_key": {
					"type": "string",
					"title": "Memory Key",
					"default": "default",
					"description": "Key to distinguish multiple memories"
				},
				"operation": {
					"enum": ["get", "add", "clear"],
					"type": "string",
					"title": "Operation",
					"default": "get",
					"description": "Operation to perform"
				},
				"message_role": {
					"enum": ["user", "assistant", "system"],
					"type": "string",
					"title": "Message Role",
					"default": "user",
					"description": "Role of message to add (for operation=add)"
				}
			}
		}`, `{
			"type": "object",
			"properties": {
				"window_size": {
					"type": "integer",
					"title": "ウィンドウサイズ",
					"default": 20,
					"minimum": 1,
					"maximum": 100,
					"description": "保持するメッセージの最大数"
				},
				"memory_key": {
					"type": "string",
					"title": "メモリキー",
					"default": "default",
					"description": "複数のメモリを区別するためのキー"
				},
				"operation": {
					"enum": ["get", "add", "clear"],
					"type": "string",
					"title": "操作",
					"default": "get",
					"description": "実行する操作"
				},
				"message_role": {
					"enum": ["user", "assistant", "system"],
					"type": "string",
					"title": "メッセージロール",
					"default": "user",
					"description": "追加するメッセージのロール（operation=add時）"
				}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("output", "Output", "出力", "Memory buffer contents or operation result", "メモリバッファの内容または操作結果", true),
		},
		Code: `
const windowSize = config.window_size || 20;
const operation = config.operation || 'get';
const key = config.memory_key || 'default';

if (!ctx.memory) {
    return { error: 'Memory service not available', messages: [] };
}

switch (operation) {
    case 'get':
        const messages = ctx.memory.getLastN(windowSize) || [];
        return {
            messages: messages,
            count: messages.length,
            window_size: windowSize
        };

    case 'add':
        const content = input.message || input.content || JSON.stringify(input);
        const role = config.message_role || 'user';
        ctx.memory.add(role, content);
        return {
            success: true,
            added: { role: role, content: content }
        };

    case 'clear':
        ctx.memory.clear();
        return {
            success: true,
            cleared: true
        };

    default:
        return { error: 'Unknown operation: ' + operation };
}
`,
		UIConfig: LSchema(`{
			"icon": "database",
			"color": "#6366F1",
			"groups": [
				{"id": "buffer", "icon": "database", "title": "Buffer Settings"},
				{"id": "operation", "icon": "settings", "title": "Operation Settings"}
			],
			"fieldGroups": {
				"window_size": "buffer",
				"memory_key": "buffer",
				"operation": "operation",
				"message_role": "operation"
			}
		}`, `{
			"icon": "database",
			"color": "#6366F1",
			"groups": [
				{"id": "buffer", "icon": "database", "title": "バッファ設定"},
				{"id": "operation", "icon": "settings", "title": "操作設定"}
			],
			"fieldGroups": {
				"window_size": "buffer",
				"memory_key": "buffer",
				"operation": "operation",
				"message_role": "operation"
			}
		}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("MEMORY_001", "NOT_AVAILABLE", "利用不可", "Memory service not available", "メモリサービスが利用できません", false),
		},
		Enabled: true,
		TestCases: []BlockTestCase{
			{
				Name:   "get memory buffer",
				Input:  map[string]interface{}{},
				Config: map[string]interface{}{"operation": "get", "window_size": 10},
				ExpectedOutput: map[string]interface{}{
					"messages": []interface{}{},
					"count":    0,
				},
			},
		},
	}
}

func RouterBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "router",
		Version:     1,
		Name:        LText("Router", "ルーター"),
		Description: LText("AI-driven dynamic routing", "AI駆動の動的ルーティング"),
		Category:    domain.BlockCategoryAI,
		Subcategory: domain.BlockSubcategoryRouting,
		Icon:        "git-branch",
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"model": {"type": "string", "title": "Model"},
				"routes": {
					"type": "array",
					"title": "Routes",
					"items": {
						"type": "object",
						"properties": {
							"name": {"type": "string"},
							"description": {"type": "string"}
						}
					}
				},
				"provider": {"type": "string", "title": "Provider"}
			}
		}`, `{
			"type": "object",
			"properties": {
				"model": {"type": "string", "title": "モデル"},
				"routes": {
					"type": "array",
					"title": "ルート",
					"items": {
						"type": "object",
						"properties": {
							"name": {"type": "string"},
							"description": {"type": "string"}
						}
					}
				},
				"provider": {"type": "string", "title": "プロバイダー"}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("default", "Default", "デフォルト", "Default route when no match", "マッチしない場合のデフォルトルート", true),
			LPortWithDesc("technical", "Technical", "技術", "Technical/code-related content", "技術/コード関連コンテンツ", false),
			LPortWithDesc("general", "General", "一般", "General knowledge content", "一般的な知識コンテンツ", false),
			LPortWithDesc("creative", "Creative", "クリエイティブ", "Creative/brainstorming content", "クリエイティブ/ブレインストーミングコンテンツ", false),
			LPortWithDesc("support", "Support", "サポート", "Customer support content", "カスタマーサポートコンテンツ", false),
			LPortWithDesc("sales", "Sales", "営業", "Sales-related content", "営業関連コンテンツ", false),
		},
		Code: `
const routeDescriptions = (config.routes || []).map(r =>
    r.name + ': ' + r.description
).join('\n');
const prompt = 'Given the following input, select the most appropriate route.\nRoutes:\n' + routeDescriptions + '\nInput: ' + JSON.stringify(input) + '\nRespond with only the route name.';
const response = ctx.llm.chat(config.provider || 'openai', config.model || 'gpt-4', {
    messages: [{ role: 'user', content: prompt }]
});
const selectedRoute = response.content.trim();
return {
    ...input,
    __port: selectedRoute,
    __branch: selectedRoute
};
`,
		UIConfig: LSchema(`{"icon": "git-branch", "color": "#8B5CF6"}`, `{"icon": "git-branch", "color": "#8B5CF6"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("ROUTER_001", "NO_MATCH", "マッチなし", "No matching route found", "マッチするルートが見つかりません", false),
		},
		RequiredCredentials: json.RawMessage(`[{"name": "llm_api_key", "type": "api_key", "scope": "system", "required": true, "description": "LLM Provider API Key"}]`),
		Enabled:             true,
	}
}
