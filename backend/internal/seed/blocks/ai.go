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
	r.register(AgentBlock())
	r.register(MemoryBufferBlock())
}

func LLMBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "llm",
		Version:     1,
		Name:        "LLM",
		Description: "Execute LLM prompts with various providers",
		Category:    domain.BlockCategoryAI,
		Subcategory: domain.BlockSubcategoryChat,
		Icon:        "brain",
		ConfigSchema: json.RawMessage(`{
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
		InputPorts: []domain.InputPort{
			{Name: "input", Label: "Input", Schema: json.RawMessage(`{"type": "any"}`), Required: false},
		},
		OutputPorts: []domain.OutputPort{
			{
				Name:        "output",
				Label:       "Output",
				Schema:      json.RawMessage(`{"type": "object", "properties": {"content": {"type": "string"}, "tokens_used": {"type": "number"}}}`),
				IsDefault:   true,
				Description: "LLM response",
			},
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
		UIConfig: json.RawMessage(`{
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
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "LLM_001", Name: "RATE_LIMIT", Description: "Rate limit exceeded", Retryable: true},
			{Code: "LLM_002", Name: "INVALID_MODEL", Description: "Invalid model specified", Retryable: false},
			{Code: "LLM_003", Name: "TOKEN_LIMIT", Description: "Token limit exceeded", Retryable: false},
			{Code: "LLM_004", Name: "API_ERROR", Description: "LLM API error", Retryable: true},
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
		Name:            "LLM (JSON)",
		Description:     "LLM with automatic JSON output parsing",
		Category:        domain.BlockCategoryAI,
		Subcategory:     domain.BlockSubcategoryChat,
		Icon:            "braces",
		ParentBlockSlug: "llm",
		ConfigDefaults: json.RawMessage(`{
			"temperature": 0.3
		}`),
		ConfigSchema: json.RawMessage(`{
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
			"try {\n" +
			"    const parsed = JSON.parse(content);\n" +
			"    return {\n" +
			"        ...parsed,\n" +
			"        __raw: input.content,\n" +
			"        __usage: input.usage\n" +
			"    };\n" +
			"} catch (e) {\n" +
			"    if (config.strict_parse) {\n" +
			"        throw new Error('[LLM_JSON_001] Failed to parse JSON: ' + e.message);\n" +
			"    }\n" +
			"    return {\n" +
			"        error: 'JSON parse failed: ' + e.message,\n" +
			"        __raw: input.content,\n" +
			"        __usage: input.usage\n" +
			"    };\n" +
			"}\n",
		UIConfig: json.RawMessage(`{
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
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "LLM_JSON_001", Name: "PARSE_FAILED", Description: "Failed to parse LLM response as JSON", Retryable: false},
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
		Name:            "LLM (Structured)",
		Description:     "LLM with schema-driven structured output and validation",
		Category:        domain.BlockCategoryAI,
		Subcategory:     domain.BlockSubcategoryChat,
		Icon:            "layout-template",
		ParentBlockSlug: "llm-json",
		ConfigDefaults: json.RawMessage(`{
			"strict_parse": true,
			"validate_output": true,
			"include_examples": true
		}`),
		ConfigSchema: json.RawMessage(`{
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
		UIConfig: json.RawMessage(`{
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
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "LLM_STRUCT_001", Name: "VALIDATION_FAILED", Description: "Output validation failed - missing required fields", Retryable: false},
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

// AgentBlock provides an AI agent with ReAct loop and tool calling capabilities
func AgentBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "agent",
		Version:     1,
		Name:        "AI Agent",
		Description: "Autonomous AI agent with ReAct loop, tool calling, and memory management",
		Category:    domain.BlockCategoryAI,
		Subcategory: domain.BlockSubcategoryChat,
		Icon:        "bot",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"required": ["provider", "model", "system_prompt"],
			"properties": {
				"provider": {
					"enum": ["openai", "anthropic"],
					"type": "string",
					"title": "プロバイダー",
					"default": "anthropic"
				},
				"model": {
					"type": "string",
					"title": "モデル",
					"default": "claude-sonnet-4-20250514"
				},
				"system_prompt": {
					"type": "string",
					"title": "システムプロンプト",
					"maxLength": 20000,
					"description": "エージェントの役割と振る舞いを定義"
				},
				"max_iterations": {
					"type": "integer",
					"title": "最大イテレーション",
					"default": 10,
					"minimum": 1,
					"maximum": 50,
					"description": "ReActループの最大反復回数"
				},
				"tools": {
					"type": "array",
					"title": "利用可能なツール",
					"items": {
						"type": "object",
						"properties": {
							"name": {"type": "string"},
							"description": {"type": "string"},
							"parameters": {"type": "object"}
						}
					},
					"description": "エージェントが使用できるツール定義"
				},
				"tool_choice": {
					"enum": ["auto", "none", "required"],
					"type": "string",
					"title": "ツール選択",
					"default": "auto"
				},
				"temperature": {
					"type": "number",
					"title": "温度",
					"default": 0.7,
					"minimum": 0,
					"maximum": 2
				},
				"memory_window": {
					"type": "integer",
					"title": "メモリウィンドウ",
					"default": 20,
					"minimum": 1,
					"maximum": 100,
					"description": "保持する会話履歴の最大数"
				},
				"enable_memory": {
					"type": "boolean",
					"title": "メモリ有効化",
					"default": true,
					"description": "会話履歴を保持する"
				}
			}
		}`),
		InputPorts: []domain.InputPort{
			{Name: "input", Label: "Input", Schema: json.RawMessage(`{"type": "object"}`), Required: true, Description: "User message or task input"},
		},
		OutputPorts: []domain.OutputPort{
			{Name: "output", Label: "Output", IsDefault: true, Description: "Agent final response"},
			{Name: "error", Label: "Error", Description: "Error output"},
		},
		Code: `
// Initialize memory if enabled
const memoryEnabled = config.enable_memory !== false;
const memoryWindow = config.memory_window || 20;
let history = [];

if (memoryEnabled && ctx.memory) {
    history = ctx.memory.getLastN(memoryWindow) || [];
}

// Add user message
const userMessage = input.message || input.content || JSON.stringify(input);
if (memoryEnabled && ctx.memory) {
    ctx.memory.addUser(userMessage);
}
history.push({ role: 'user', content: userMessage });

// Prepare tools
const tools = (config.tools || []).map(function(t) {
    return {
        type: 'function',
        function: {
            name: t.name,
            description: t.description,
            parameters: t.parameters || {}
        }
    };
});

const maxIterations = config.max_iterations || 10;
let finalResponse = null;

// ReAct loop
for (let iteration = 0; iteration < maxIterations; iteration++) {
    // Call LLM
    const llmOptions = {
        messages: [
            { role: 'system', content: config.system_prompt || 'You are a helpful AI assistant.' },
            ...history
        ],
        temperature: config.temperature || 0.7,
        maxTokens: 4096
    };

    if (tools.length > 0) {
        llmOptions.tools = tools;
        llmOptions.tool_choice = config.tool_choice || 'auto';
    }

    const response = ctx.llm.chat(config.provider, config.model, llmOptions);

    // Check for tool calls
    if (response.tool_calls && response.tool_calls.length > 0) {
        // Add assistant message with tool calls
        history.push({
            role: 'assistant',
            content: response.content || '',
            tool_calls: response.tool_calls
        });

        if (memoryEnabled && ctx.memory) {
            ctx.memory.addWithToolCalls('assistant', response.content || '', response.tool_calls);
        }

        // Execute each tool call
        for (const toolCall of response.tool_calls) {
            let toolResult;
            try {
                // Execute tool via workflow if available
                if (ctx.workflow && ctx.workflow.executeStep) {
                    const args = JSON.parse(toolCall.function.arguments || '{}');
                    toolResult = ctx.workflow.executeStep(toolCall.function.name, args);
                } else {
                    toolResult = { error: 'Tool execution not available' };
                }
            } catch (e) {
                toolResult = { error: e.message || 'Tool execution failed' };
            }

            // Add tool result to history
            const toolResultStr = JSON.stringify(toolResult);
            history.push({
                role: 'tool',
                tool_call_id: toolCall.id,
                content: toolResultStr
            });

            if (memoryEnabled && ctx.memory) {
                ctx.memory.addTool(toolResultStr, toolCall.id);
            }
        }
    } else {
        // No tool calls - final response
        finalResponse = response.content;

        if (memoryEnabled && ctx.memory) {
            ctx.memory.addAssistant(finalResponse);
        }

        break;
    }
}

// Return final response
if (finalResponse === null) {
    finalResponse = 'Agent reached maximum iterations without final response.';
}

return {
    response: finalResponse,
    iterations: history.filter(function(m) { return m.role === 'assistant'; }).length,
    memory_count: history.length
};
`,
		UIConfig: json.RawMessage(`{
			"icon": "bot",
			"color": "#10B981",
			"groups": [
				{"id": "model", "icon": "robot", "title": "モデル設定"},
				{"id": "agent", "icon": "bot", "title": "エージェント設定"},
				{"id": "memory", "icon": "database", "title": "メモリ設定"},
				{"id": "tools", "icon": "wrench", "title": "ツール設定"}
			],
			"fieldGroups": {
				"model": "model",
				"provider": "model",
				"system_prompt": "agent",
				"max_iterations": "agent",
				"temperature": "agent",
				"memory_window": "memory",
				"enable_memory": "memory",
				"tools": "tools",
				"tool_choice": "tools"
			},
			"fieldOverrides": {
				"system_prompt": {"rows": 6, "widget": "textarea"}
			}
		}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "AGENT_001", Name: "MAX_ITERATIONS", Description: "Agent reached maximum iterations", Retryable: false},
			{Code: "AGENT_002", Name: "TOOL_ERROR", Description: "Tool execution failed", Retryable: true},
			{Code: "AGENT_003", Name: "LLM_ERROR", Description: "LLM API error", Retryable: true},
		},
		RequiredCredentials: json.RawMessage(`[{"name": "llm_api_key", "type": "api_key", "scope": "system", "required": true, "description": "LLM Provider API Key"}]`),
		Enabled:             true,
		TestCases: []BlockTestCase{
			{
				Name:   "basic agent call",
				Input:  map[string]interface{}{"message": "Hello"},
				Config: map[string]interface{}{"provider": "mock", "model": "test", "system_prompt": "You are helpful"},
				ExpectedOutput: map[string]interface{}{
					"response": "Mock LLM response",
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
		Name:        "Memory Buffer",
		Description: "Manage conversation memory with sliding window",
		Category:    domain.BlockCategoryAI,
		Subcategory: domain.BlockSubcategoryChat,
		Icon:        "database",
		ConfigSchema: json.RawMessage(`{
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
		InputPorts: []domain.InputPort{
			{Name: "input", Label: "Input", Schema: json.RawMessage(`{"type": "object"}`), Required: false, Description: "Message to add (for add operation)"},
		},
		OutputPorts: []domain.OutputPort{
			{Name: "output", Label: "Output", IsDefault: true, Description: "Memory buffer contents or operation result"},
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
		UIConfig: json.RawMessage(`{
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
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "MEMORY_001", Name: "NOT_AVAILABLE", Description: "Memory service not available", Retryable: false},
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
		Name:        "Router",
		Description: "AI-driven dynamic routing",
		Category:    domain.BlockCategoryAI,
		Subcategory: domain.BlockSubcategoryRouting,
		Icon:        "git-branch",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"model": {"type": "string"},
				"routes": {
					"type": "array",
					"items": {
						"type": "object",
						"properties": {
							"name": {"type": "string"},
							"description": {"type": "string"}
						}
					}
				},
				"provider": {"type": "string"}
			}
		}`),
		InputPorts: []domain.InputPort{
			{Name: "input", Label: "Input", Schema: json.RawMessage(`{"type": "string"}`), Required: true, Description: "Message to analyze for routing"},
		},
		OutputPorts: []domain.OutputPort{
			{Name: "default", Label: "Default", IsDefault: true, Description: "Default route when no match"},
			// Dynamic route ports - common route names for AI routing scenarios
			{Name: "technical", Label: "Technical", Description: "Technical/code-related content"},
			{Name: "general", Label: "General", Description: "General knowledge content"},
			{Name: "creative", Label: "Creative", Description: "Creative/brainstorming content"},
			{Name: "support", Label: "Support", Description: "Customer support content"},
			{Name: "sales", Label: "Sales", Description: "Sales-related content"},
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
		UIConfig: json.RawMessage(`{"icon": "git-branch", "color": "#8B5CF6"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "ROUTER_001", Name: "NO_MATCH", Description: "No matching route found", Retryable: false},
		},
		RequiredCredentials: json.RawMessage(`[{"name": "llm_api_key", "type": "api_key", "scope": "system", "required": true, "description": "LLM Provider API Key"}]`),
		Enabled:             true,
	}
}
