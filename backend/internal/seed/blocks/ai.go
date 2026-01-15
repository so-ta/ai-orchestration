package blocks

import (
	"encoding/json"

	"github.com/souta/ai-orchestration/internal/domain"
)

func (r *Registry) registerAIBlocks() {
	r.register(LLMBlock())
	r.register(RouterBlock())
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
				"system_prompt": {"type": "string", "maxLength": 10000}
			}
		}`),
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"message": {"type": "string"},
				"context": {"type": "string"}
			},
			"description": "LLMプロンプトの入力データ"
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
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"message": {"type": "string", "description": "ルーティング判定対象のメッセージ"}
			},
			"description": "ルーティング判定に使用するデータ"
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
