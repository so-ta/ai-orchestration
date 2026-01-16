package blocks

import (
	"encoding/json"

	"github.com/souta/ai-orchestration/internal/domain"
)

func (r *Registry) registerIntegrationBlocks() {
	// === Level 0: Base blocks ===
	r.register(HTTPBlock())
	r.register(SubflowBlock())
	r.register(ToolBlock())

	// === Level 1: Foundation pattern blocks ===
	r.register(WebhookBlock())
	r.register(RestAPIBlock())
	r.register(GraphQLBlock())

	// === Level 2: Authentication pattern blocks ===
	r.register(BearerAPIBlock())
	r.register(APIKeyHeaderBlock())
	r.register(APIKeyQueryBlock())

	// === Level 3: Service-specific base blocks ===
	r.register(GitHubAPIBlock())
	r.register(NotionAPIBlock())
	r.register(GoogleAPIBlock())
	r.register(LinearAPIBlock())

	// === Level 4+: Concrete operation blocks ===
	// Webhook-based
	r.register(SlackBlock())
	r.register(DiscordBlock())
	// GitHub
	r.register(GitHubCreateIssueBlock())
	r.register(GitHubAddCommentBlock())
	// Notion
	r.register(NotionQueryDBBlock())
	r.register(NotionCreatePageBlock())
	// Google Sheets
	r.register(GSheetsAppendBlock())
	r.register(GSheetsReadBlock())
	// Other API-based
	r.register(WebSearchBlock())
	r.register(EmailSendGridBlock())
	r.register(LinearCreateIssueBlock())

	// === RAG/Vector blocks (no inheritance) ===
	r.register(EmbeddingBlock())
	r.register(VectorUpsertBlock())
	r.register(VectorSearchBlock())
	r.register(VectorDeleteBlock())
}

func HTTPBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "http",
		Version:     2, // Incremented for inheritance support
		Name:        "HTTP Request",
		Description: "Make HTTP API calls",
		Category:    domain.BlockCategoryApps,
		Subcategory: domain.BlockSubcategoryWeb,
		Icon:        "globe",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"url": {"type": "string"},
				"body": {"type": "object"},
				"method": {"enum": ["GET", "POST", "PUT", "DELETE", "PATCH"], "type": "string"},
				"headers": {"type": "object"},
				"enable_error_port": {
					"type": "boolean",
					"title": "エラーハンドルを有効化",
					"description": "エラー発生時に専用のエラーポートに出力します",
					"default": false
				}
			}
		}`),
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"url": {"type": "string", "description": "リクエストURL（継承ブロック用）"},
				"body": {"type": "object", "description": "リクエストボディ"},
				"method": {"type": "string", "description": "HTTPメソッド（継承ブロック用）"},
				"headers": {"type": "object", "description": "追加ヘッダー"}
			},
			"description": "URL/ボディのテンプレートで参照可能なデータ。継承ブロックはPreProcessでurl/body/method/headersを設定可能"
		}`),
		InputPorts:  []domain.InputPort{},
		OutputPorts: []domain.OutputPort{},
		Code: `
// Support inheritance: input can override config values
const url = renderTemplate(input.url || config.url, input);
const method = input.method || config.method || 'GET';
const headers = Object.assign({}, config.headers || {}, input.headers || {});
const body = input.body !== undefined ? input.body : config.body;
const response = ctx.http.request(url, {
    method: method,
    headers: headers,
    body: body ? (typeof body === 'string' ? body : renderTemplate(JSON.stringify(body), input)) : null
});
return response;
`,
		UIConfig: json.RawMessage(`{"icon": "globe", "color": "#3B82F6"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "HTTP_001", Name: "CONNECTION_ERROR", Description: "Failed to connect", Retryable: true},
			{Code: "HTTP_002", Name: "TIMEOUT", Description: "Request timeout", Retryable: true},
		},
		Enabled: true,
	}
}

func SubflowBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "subflow",
		Version:     1,
		Name:        "Subflow",
		Description: "Execute another workflow",
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryControl,
		Icon:        "workflow",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"workflow_id": {"type": "string", "format": "uuid"},
				"workflow_version": {"type": "integer"}
			}
		}`),
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"input": {"type": "object", "description": "サブフローに渡す入力データ"}
			},
			"description": "サブフローに渡すデータ"
		}`),
		InputPorts: []domain.InputPort{
			{Name: "input", Label: "Input", Schema: json.RawMessage(`{"type": "any"}`), Required: false, Description: "Input data for subflow"},
		},
		OutputPorts: []domain.OutputPort{
			{Name: "output", Label: "Output", IsDefault: true, Description: "Subflow result"},
		},
		Code:     `return ctx.workflow.run(config.workflow_id, input);`,
		UIConfig: json.RawMessage(`{"icon": "workflow", "color": "#10B981"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "SUBFLOW_001", Name: "NOT_FOUND", Description: "Subflow workflow not found", Retryable: false},
			{Code: "SUBFLOW_002", Name: "EXEC_ERROR", Description: "Subflow execution error", Retryable: true},
		},
		Enabled: true,
	}
}

func ToolBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "tool",
		Version:     1,
		Name:        "Tool",
		Description: "Execute external tool/adapter",
		Category:    domain.BlockCategoryApps,
		Subcategory: domain.BlockSubcategoryWeb,
		Icon:        "wrench",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"adapter_id": {"type": "string"},
				"enable_error_port": {
					"type": "boolean",
					"title": "エラーハンドルを有効化",
					"description": "エラー発生時に専用のエラーポートに出力します",
					"default": false
				}
			}
		}`),
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"params": {"type": "object", "description": "ツールに渡すパラメータ"}
			},
			"description": "ツールに渡すデータ"
		}`),
		InputPorts: []domain.InputPort{
			{Name: "input", Label: "Input", Schema: json.RawMessage(`{"type": "any"}`), Required: false, Description: "Input data for the tool"},
		},
		OutputPorts: []domain.OutputPort{
			{Name: "output", Label: "Output", IsDefault: true, Description: "Tool execution result"},
		},
		Code:     `return ctx.adapter.call(config.adapter_id, input);`,
		UIConfig: json.RawMessage(`{"icon": "wrench", "color": "#10B981"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "TOOL_001", Name: "ADAPTER_NOT_FOUND", Description: "Adapter not found", Retryable: false},
			{Code: "TOOL_002", Name: "EXEC_ERROR", Description: "Tool execution error", Retryable: true},
		},
		Enabled: true,
	}
}

// =============================================================================
// Level 1: Foundation Pattern Blocks
// =============================================================================

// WebhookBlock provides a base for webhook-style POST notifications
func WebhookBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:            "webhook",
		Version:         1,
		Name:            "Webhook",
		Description:     "Webhook POST通知の基盤ブロック",
		Category:        domain.BlockCategoryApps,
		Subcategory:     domain.BlockSubcategoryWeb,
		Icon:            "send",
		ParentBlockSlug: "http",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"webhook_url": {"type": "string", "title": "Webhook URL"},
				"secret_key": {"type": "string", "title": "シークレットキー名", "description": "ctx.secretsから取得するキー名"}
			}
		}`),
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"payload": {"type": "object", "description": "送信するペイロード"}
			}
		}`),
		OutputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"success": {"type": "boolean"},
				"status": {"type": "number"}
			}
		}`),
		InputPorts:  []domain.InputPort{},
		OutputPorts: []domain.OutputPort{},
		PreProcess: `
const url = config.webhook_url || (config.secret_key ? ctx.secrets[config.secret_key] : null);
if (!url) {
    throw new Error('[WEBHOOK_001] Webhook URLが設定されていません');
}
return {
    ...input,
    url: url,
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: input.payload || input
};
`,
		PostProcess: `
if (input.status >= 400) {
    throw new Error('[WEBHOOK_002] Webhook送信失敗: ' + input.status);
}
return { success: true, status: input.status };
`,
		UIConfig: json.RawMessage(`{"icon": "send", "color": "#6366F1"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "WEBHOOK_001", Name: "URL_NOT_CONFIGURED", Description: "Webhook URLが設定されていません", Retryable: false},
			{Code: "WEBHOOK_002", Name: "SEND_FAILED", Description: "送信に失敗しました", Retryable: true},
		},
		Enabled: true,
	}
}

// RestAPIBlock provides a base for REST API calls with authentication
func RestAPIBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:            "rest-api",
		Version:         1,
		Name:            "REST API",
		Description:     "REST API呼び出しの基盤ブロック（認証サポート）",
		Category:        domain.BlockCategoryApps,
		Subcategory:     domain.BlockSubcategoryWeb,
		Icon:            "cloud",
		ParentBlockSlug: "http",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"base_url": {"type": "string", "title": "ベースURL"},
				"auth_type": {"type": "string", "enum": ["none", "bearer", "api_key_header", "api_key_query"], "title": "認証タイプ", "default": "none"},
				"auth_key": {"type": "string", "title": "認証キー（直接指定）"},
				"secret_key": {"type": "string", "title": "シークレットキー名"},
				"header_name": {"type": "string", "title": "ヘッダー名", "default": "X-API-Key", "description": "api_key_header使用時のヘッダー名"}
			}
		}`),
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"url": {"type": "string", "description": "リクエストURL（base_urlからの相対パス可）"},
				"method": {"type": "string", "description": "HTTPメソッド"},
				"headers": {"type": "object", "description": "追加ヘッダー"},
				"body": {"type": "object", "description": "リクエストボディ"}
			}
		}`),
		InputPorts:  []domain.InputPort{},
		OutputPorts: []domain.OutputPort{},
		PreProcess: `
// 認証キー解決
const authValue = config.auth_key || (config.secret_key ? ctx.secrets[config.secret_key] : null);

// ヘッダー構築
const headers = { ...(input.headers || {}) };
const authType = config.auth_type || 'none';

if (authType === 'bearer' && authValue) {
    headers['Authorization'] = 'Bearer ' + authValue;
} else if (authType === 'api_key_header' && authValue) {
    const headerName = config.header_name || 'X-API-Key';
    headers[headerName] = authValue;
}

// URL構築
let url = input.url || '';
if (config.base_url) {
    url = config.base_url + url;
}
if (authType === 'api_key_query' && authValue) {
    url += (url.includes('?') ? '&' : '?') + 'key=' + encodeURIComponent(authValue);
}

return { ...input, url, headers };
`,
		PostProcess: `
// レート制限チェック
if (input.status === 429) {
    throw new Error('[REST_003] レート制限に達しました');
}
// エラーステータス
if (input.status >= 400) {
    const errorMsg = input.body?.message || input.body?.error || 'Unknown error';
    throw new Error('[REST_002] APIエラー: ' + errorMsg);
}
return input;
`,
		UIConfig: json.RawMessage(`{"icon": "cloud", "color": "#0EA5E9"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "REST_001", Name: "AUTH_FAILED", Description: "認証に失敗しました", Retryable: false},
			{Code: "REST_002", Name: "API_ERROR", Description: "APIエラー", Retryable: true},
			{Code: "REST_003", Name: "RATE_LIMITED", Description: "レート制限に達しました", Retryable: true},
		},
		Enabled: true,
	}
}

// GraphQLBlock provides a base for GraphQL API calls
func GraphQLBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:            "graphql",
		Version:         1,
		Name:            "GraphQL",
		Description:     "GraphQL API呼び出しの基盤ブロック",
		Category:        domain.BlockCategoryApps,
		Subcategory:     domain.BlockSubcategoryWeb,
		Icon:            "hexagon",
		ParentBlockSlug: "rest-api",
		ConfigDefaults: json.RawMessage(`{
			"auth_type": "bearer"
		}`),
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"endpoint": {"type": "string", "title": "GraphQLエンドポイント"},
				"query": {"type": "string", "title": "GraphQLクエリ", "x-ui-widget": "textarea"},
				"variables": {"type": "object", "title": "変数"}
			}
		}`),
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"query": {"type": "string", "description": "GraphQLクエリ（configより優先）"},
				"variables": {"type": "object", "description": "変数（configとマージ）"}
			}
		}`),
		InputPorts:  []domain.InputPort{},
		OutputPorts: []domain.OutputPort{},
		PreProcess: `
const query = input.query || config.query;
if (!query) {
    throw new Error('[GQL_001] GraphQLクエリが指定されていません');
}
const variables = { ...(config.variables || {}), ...(input.variables || {}) };
const body = { query, variables };
const url = config.endpoint || input.url || '';
return { ...input, url, method: 'POST', body };
`,
		PostProcess: `
// GraphQLエラーチェック
if (input.body?.errors && input.body.errors.length > 0) {
    throw new Error('[GQL_002] ' + input.body.errors[0].message);
}
return input.body?.data || input;
`,
		UIConfig: json.RawMessage(`{"icon": "hexagon", "color": "#E535AB"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "GQL_001", Name: "QUERY_REQUIRED", Description: "クエリが指定されていません", Retryable: false},
			{Code: "GQL_002", Name: "GRAPHQL_ERROR", Description: "GraphQLエラー", Retryable: false},
		},
		Enabled: true,
	}
}

// =============================================================================
// Level 2: Authentication Pattern Blocks
// =============================================================================

// BearerAPIBlock provides Bearer token authentication
func BearerAPIBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:            "bearer-api",
		Version:         1,
		Name:            "Bearer API",
		Description:     "Bearer Token認証API基盤",
		Category:        domain.BlockCategoryApps,
		Subcategory:     domain.BlockSubcategoryWeb,
		Icon:            "key",
		ParentBlockSlug: "rest-api",
		ConfigDefaults: json.RawMessage(`{
			"auth_type": "bearer"
		}`),
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"token": {"type": "string", "title": "アクセストークン（直接指定）"}
			}
		}`),
		InputPorts:  []domain.InputPort{},
		OutputPorts: []domain.OutputPort{},
		PreProcess: `
// token -> auth_key にマッピング
if (config.token && !config.auth_key) {
    config.auth_key = config.token;
}
return input;
`,
		UIConfig: json.RawMessage(`{"icon": "key", "color": "#F59E0B"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "BEARER_001", Name: "TOKEN_NOT_CONFIGURED", Description: "トークンが設定されていません", Retryable: false},
		},
		Enabled: true,
	}
}

// APIKeyHeaderBlock provides API Key header authentication
func APIKeyHeaderBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:            "api-key-header",
		Version:         1,
		Name:            "API Key (Header)",
		Description:     "APIキーヘッダー認証基盤",
		Category:        domain.BlockCategoryApps,
		Subcategory:     domain.BlockSubcategoryWeb,
		Icon:            "key",
		ParentBlockSlug: "rest-api",
		ConfigDefaults: json.RawMessage(`{
			"auth_type": "api_key_header"
		}`),
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"api_key": {"type": "string", "title": "APIキー（直接指定）"},
				"header_name": {"type": "string", "title": "ヘッダー名", "default": "X-API-Key"}
			}
		}`),
		InputPorts:  []domain.InputPort{},
		OutputPorts: []domain.OutputPort{},
		PreProcess: `
// api_key -> auth_key にマッピング
if (config.api_key && !config.auth_key) {
    config.auth_key = config.api_key;
}
return input;
`,
		UIConfig: json.RawMessage(`{"icon": "key", "color": "#10B981"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "APIKEY_001", Name: "KEY_NOT_CONFIGURED", Description: "APIキーが設定されていません", Retryable: false},
		},
		Enabled: true,
	}
}

// APIKeyQueryBlock provides API Key query parameter authentication
func APIKeyQueryBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:            "api-key-query",
		Version:         1,
		Name:            "API Key (Query)",
		Description:     "APIキークエリパラメータ認証基盤",
		Category:        domain.BlockCategoryApps,
		Subcategory:     domain.BlockSubcategoryWeb,
		Icon:            "key",
		ParentBlockSlug: "rest-api",
		ConfigDefaults: json.RawMessage(`{
			"auth_type": "api_key_query"
		}`),
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"api_key": {"type": "string", "title": "APIキー（直接指定）"}
			}
		}`),
		InputPorts:  []domain.InputPort{},
		OutputPorts: []domain.OutputPort{},
		PreProcess: `
// api_key -> auth_key にマッピング
if (config.api_key && !config.auth_key) {
    config.auth_key = config.api_key;
}
return input;
`,
		UIConfig: json.RawMessage(`{"icon": "key", "color": "#8B5CF6"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "APIKEY_001", Name: "KEY_NOT_CONFIGURED", Description: "APIキーが設定されていません", Retryable: false},
		},
		Enabled: true,
	}
}

// =============================================================================
// Level 3: Service-Specific Base Blocks
// =============================================================================

// GitHubAPIBlock provides GitHub API base configuration
func GitHubAPIBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:            "github-api",
		Version:         1,
		Name:            "GitHub API",
		Description:     "GitHub API基盤ブロック",
		Category:        domain.BlockCategoryApps,
		Subcategory:     domain.BlockSubcategoryGitHub,
		Icon:            "github",
		ParentBlockSlug: "bearer-api",
		ConfigDefaults: json.RawMessage(`{
			"base_url": "https://api.github.com",
			"secret_key": "GITHUB_TOKEN"
		}`),
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"token": {"type": "string", "title": "アクセストークン"}
			}
		}`),
		InputPorts:  []domain.InputPort{},
		OutputPorts: []domain.OutputPort{},
		PreProcess: `
return {
    ...input,
    headers: {
        ...(input.headers || {}),
        'Accept': 'application/vnd.github+json',
        'X-GitHub-Api-Version': '2022-11-28'
    }
};
`,
		PostProcess: `
if (input.status === 404) {
    throw new Error('[GITHUB_003] リソースが見つかりません');
}
return input;
`,
		UIConfig: json.RawMessage(`{"icon": "github", "color": "#24292F"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "GITHUB_001", Name: "TOKEN_NOT_CONFIGURED", Description: "トークンが設定されていません", Retryable: false},
			{Code: "GITHUB_003", Name: "NOT_FOUND", Description: "リソースが見つかりません", Retryable: false},
		},
		Enabled: true,
	}
}

// NotionAPIBlock provides Notion API base configuration
func NotionAPIBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:            "notion-api",
		Version:         1,
		Name:            "Notion API",
		Description:     "Notion API基盤ブロック",
		Category:        domain.BlockCategoryApps,
		Subcategory:     domain.BlockSubcategoryNotion,
		Icon:            "file-text",
		ParentBlockSlug: "bearer-api",
		ConfigDefaults: json.RawMessage(`{
			"base_url": "https://api.notion.com/v1",
			"secret_key": "NOTION_API_KEY"
		}`),
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"api_key": {"type": "string", "title": "API Key"}
			}
		}`),
		InputPorts:  []domain.InputPort{},
		OutputPorts: []domain.OutputPort{},
		PreProcess: `
// api_key -> token にマッピング（親のbearer-api用）
if (config.api_key && !config.token) {
    config.token = config.api_key;
}
return {
    ...input,
    headers: {
        ...(input.headers || {}),
        'Notion-Version': '2022-06-28'
    }
};
`,
		UIConfig: json.RawMessage(`{"icon": "file-text", "color": "#000000"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "NOTION_001", Name: "API_KEY_NOT_CONFIGURED", Description: "API Keyが設定されていません", Retryable: false},
		},
		Enabled: true,
	}
}

// GoogleAPIBlock provides Google API base configuration
func GoogleAPIBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:            "google-api",
		Version:         1,
		Name:            "Google API",
		Description:     "Google API基盤ブロック",
		Category:        domain.BlockCategoryApps,
		Subcategory:     domain.BlockSubcategoryGoogle,
		Icon:            "cloud",
		ParentBlockSlug: "api-key-query",
		ConfigDefaults: json.RawMessage(`{
			"secret_key": "GOOGLE_API_KEY"
		}`),
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"api_key": {"type": "string", "title": "API Key"}
			}
		}`),
		InputPorts:  []domain.InputPort{},
		OutputPorts: []domain.OutputPort{},
		PostProcess: `
if (input.status === 404) {
    throw new Error('[GOOGLE_003] リソースが見つかりません');
}
if (input.body?.error) {
    throw new Error('[GOOGLE_002] ' + (input.body.error.message || 'Unknown error'));
}
return input;
`,
		UIConfig: json.RawMessage(`{"icon": "cloud", "color": "#4285F4"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "GOOGLE_001", Name: "API_KEY_NOT_CONFIGURED", Description: "API Keyが設定されていません", Retryable: false},
			{Code: "GOOGLE_002", Name: "API_ERROR", Description: "Google APIエラー", Retryable: true},
			{Code: "GOOGLE_003", Name: "NOT_FOUND", Description: "リソースが見つかりません", Retryable: false},
		},
		Enabled: true,
	}
}

// LinearAPIBlock provides Linear API base configuration
func LinearAPIBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:            "linear-api",
		Version:         1,
		Name:            "Linear API",
		Description:     "Linear API基盤ブロック",
		Category:        domain.BlockCategoryApps,
		Subcategory:     domain.BlockSubcategoryLinear,
		Icon:            "check-square",
		ParentBlockSlug: "graphql",
		ConfigDefaults: json.RawMessage(`{
			"endpoint": "https://api.linear.app/graphql",
			"secret_key": "LINEAR_API_KEY"
		}`),
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"api_key": {"type": "string", "title": "API Key"}
			}
		}`),
		InputPorts:  []domain.InputPort{},
		OutputPorts: []domain.OutputPort{},
		PreProcess: `
// api_key -> auth_key にマッピング
if (config.api_key && !config.auth_key) {
    config.auth_key = config.api_key;
}
return input;
`,
		UIConfig: json.RawMessage(`{"icon": "check-square", "color": "#5E6AD2"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "LINEAR_001", Name: "API_KEY_NOT_CONFIGURED", Description: "API Keyが設定されていません", Retryable: false},
		},
		Enabled: true,
	}
}

// =============================================================================
// Level 4+: Concrete Operation Blocks
// =============================================================================

// SlackBlock sends messages to Slack via webhook
func SlackBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:            "slack",
		Version:         3, // Incremented for webhook inheritance
		Name:            "Slack",
		Description:     "Slackチャンネルにメッセージを送信",
		Category:        domain.BlockCategoryApps,
		Subcategory:     domain.BlockSubcategorySlack,
		Icon:            "message-square",
		ParentBlockSlug: "webhook",
		ConfigDefaults: json.RawMessage(`{
			"secret_key": "SLACK_WEBHOOK_URL"
		}`),
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"required": ["message"],
			"properties": {
				"blocks": {"type": "array", "title": "Block Kit", "x-ui-widget": "output-schema"},
				"channel": {"type": "string", "title": "チャンネル"},
				"message": {"type": "string", "title": "メッセージ", "x-ui-widget": "textarea"},
				"username": {"type": "string", "title": "表示名"},
				"icon_emoji": {"type": "string", "title": "アイコン絵文字"},
				"webhook_url": {"type": "string", "title": "Webhook URL"}
			}
		}`),
		InputPorts:  []domain.InputPort{},
		OutputPorts: []domain.OutputPort{},
		OutputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"status": {"type": "number"},
				"success": {"type": "boolean"}
			}
		}`),
		PreProcess: `
const payload = {
    text: renderTemplate(config.message, input)
};
if (config.channel) payload.channel = config.channel;
if (config.username) payload.username = config.username;
if (config.icon_emoji) payload.icon_emoji = config.icon_emoji;
if (config.blocks && config.blocks.length > 0) payload.blocks = config.blocks;
return { ...input, payload };
`,
		PostProcess: `
if (input.status >= 400) {
    throw new Error('[SLACK_002] Slack送信失敗: ' + input.status);
}
return { success: true, status: input.status };
`,
		UIConfig: json.RawMessage(`{"icon": "message-square", "color": "#4A154B"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "SLACK_001", Name: "WEBHOOK_NOT_CONFIGURED", Description: "Webhook URLが設定されていません", Retryable: false},
			{Code: "SLACK_002", Name: "SEND_FAILED", Description: "メッセージ送信に失敗しました", Retryable: true},
			{Code: "SLACK_003", Name: "INVALID_WEBHOOK", Description: "無効なWebhook URL", Retryable: false},
		},
		Enabled: true,
	}
}

// DiscordBlock sends messages to Discord via webhook
func DiscordBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:            "discord",
		Version:         3, // Incremented for webhook inheritance
		Name:            "Discord",
		Description:     "Discord Webhookにメッセージを送信",
		Category:        domain.BlockCategoryApps,
		Subcategory:     domain.BlockSubcategoryDiscord,
		Icon:            "message-circle",
		ParentBlockSlug: "webhook",
		ConfigDefaults: json.RawMessage(`{
			"secret_key": "DISCORD_WEBHOOK_URL"
		}`),
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"required": ["content"],
			"properties": {
				"embeds": {"type": "array", "title": "Embeds", "x-ui-widget": "output-schema"},
				"content": {"type": "string", "title": "メッセージ", "x-ui-widget": "textarea"},
				"username": {"type": "string", "title": "ユーザー名"},
				"avatar_url": {"type": "string", "title": "アバターURL"},
				"webhook_url": {"type": "string", "title": "Webhook URL"}
			}
		}`),
		InputPorts:  []domain.InputPort{},
		OutputPorts: []domain.OutputPort{},
		OutputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"status": {"type": "number"},
				"success": {"type": "boolean"}
			}
		}`),
		PreProcess: `
const payload = {
    content: renderTemplate(config.content, input)
};
if (config.username) payload.username = config.username;
if (config.avatar_url) payload.avatar_url = config.avatar_url;
if (config.embeds && config.embeds.length > 0) payload.embeds = config.embeds;
return { ...input, payload };
`,
		PostProcess: `
if (input.status === 429) {
    throw new Error('[DISCORD_003] レート制限に達しました');
}
if (input.status >= 400) {
    throw new Error('[DISCORD_002] Discord送信失敗: ' + input.status);
}
return { success: true, status: input.status };
`,
		UIConfig: json.RawMessage(`{"icon": "message-circle", "color": "#5865F2"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "DISCORD_001", Name: "WEBHOOK_NOT_CONFIGURED", Description: "Webhook URLが設定されていません", Retryable: false},
			{Code: "DISCORD_002", Name: "SEND_FAILED", Description: "メッセージ送信に失敗しました", Retryable: true},
			{Code: "DISCORD_003", Name: "RATE_LIMITED", Description: "レート制限に達しました", Retryable: true},
		},
		Enabled: true,
	}
}

// NotionQueryDBBlock queries a Notion database
func NotionQueryDBBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:            "notion_query_db",
		Version:         2, // Incremented for notion-api inheritance
		Name:            "Notion: DB検索",
		Description:     "Notionデータベースを検索",
		Category:        domain.BlockCategoryApps,
		Subcategory:     domain.BlockSubcategoryNotion,
		Icon:            "database",
		ParentBlockSlug: "notion-api",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"required": ["database_id"],
			"properties": {
				"sorts": {"type": "array", "title": "ソート", "x-ui-widget": "output-schema"},
				"filter": {"type": "object", "title": "フィルター", "x-ui-widget": "output-schema"},
				"api_key": {"type": "string", "title": "API Key"},
				"page_size": {"type": "number", "title": "取得件数", "default": 100, "maximum": 100, "minimum": 1},
				"database_id": {"type": "string", "title": "データベースID"}
			}
		}`),
		InputPorts:  []domain.InputPort{},
		OutputPorts: []domain.OutputPort{},
		OutputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"results": {"type": "array"},
				"has_more": {"type": "boolean"},
				"next_cursor": {"type": "string"}
			}
		}`),
		PreProcess: `
const payload = {};
if (config.filter) payload.filter = config.filter;
if (config.sorts) payload.sorts = config.sorts;
if (config.page_size) payload.page_size = config.page_size;
return {
    ...input,
    url: '/databases/' + config.database_id + '/query',
    method: 'POST',
    body: payload
};
`,
		PostProcess: `
if (input.status >= 400) {
    const errorMsg = input.body?.message || 'Unknown error';
    throw new Error('[NOTION_004] クエリ失敗: ' + errorMsg);
}
return {
    results: input.body.results,
    has_more: input.body.has_more,
    next_cursor: input.body.next_cursor
};
`,
		UIConfig: json.RawMessage(`{"icon": "database", "color": "#000000"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "NOTION_001", Name: "API_KEY_NOT_CONFIGURED", Description: "API Keyが設定されていません", Retryable: false},
			{Code: "NOTION_004", Name: "QUERY_FAILED", Description: "クエリに失敗しました", Retryable: true},
		},
		Enabled: true,
	}
}

// NotionCreatePageBlock creates a page in Notion
func NotionCreatePageBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:            "notion_create_page",
		Version:         2, // Incremented for notion-api inheritance
		Name:            "Notion: ページ作成",
		Description:     "Notionにページを作成",
		Category:        domain.BlockCategoryApps,
		Subcategory:     domain.BlockSubcategoryNotion,
		Icon:            "file-text",
		ParentBlockSlug: "notion-api",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"required": ["parent_id"],
			"properties": {
				"title": {"type": "string", "title": "タイトル"},
				"api_key": {"type": "string", "title": "API Key"},
				"content": {"type": "string", "title": "本文", "x-ui-widget": "textarea"},
				"parent_id": {"type": "string", "title": "親ID"},
				"properties": {"type": "object", "title": "プロパティ", "x-ui-widget": "output-schema"},
				"parent_type": {"enum": ["page_id", "database_id"], "type": "string", "title": "親タイプ", "default": "database_id"}
			}
		}`),
		InputPorts:  []domain.InputPort{},
		OutputPorts: []domain.OutputPort{},
		OutputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"id": {"type": "string"},
				"url": {"type": "string"},
				"created_time": {"type": "string"}
			}
		}`),
		PreProcess: `
const parentKey = config.parent_type || 'database_id';
const payload = {
    parent: { [parentKey]: config.parent_id }
};
if (config.properties) {
    payload.properties = config.properties;
} else if (config.title) {
    payload.properties = {
        title: {
            title: [{ text: { content: renderTemplate(config.title, input) } }]
        }
    };
}
if (config.content) {
    payload.children = [
        {
            object: 'block',
            type: 'paragraph',
            paragraph: {
                rich_text: [{ type: 'text', text: { content: renderTemplate(config.content, input) } }]
            }
        }
    ];
}
return {
    ...input,
    url: '/pages',
    method: 'POST',
    body: payload
};
`,
		PostProcess: `
if (input.status >= 400) {
    const errorMsg = input.body?.message || 'Unknown error';
    throw new Error('[NOTION_002] ページ作成失敗: ' + errorMsg);
}
return {
    id: input.body.id,
    url: input.body.url,
    created_time: input.body.created_time
};
`,
		UIConfig: json.RawMessage(`{"icon": "file-text", "color": "#000000"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "NOTION_001", Name: "API_KEY_NOT_CONFIGURED", Description: "API Keyが設定されていません", Retryable: false},
			{Code: "NOTION_002", Name: "CREATE_FAILED", Description: "ページ作成に失敗しました", Retryable: true},
			{Code: "NOTION_003", Name: "INVALID_PARENT", Description: "無効な親IDです", Retryable: false},
		},
		Enabled: true,
	}
}

// GSheetsAppendBlock appends rows to Google Sheets
func GSheetsAppendBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:            "gsheets_append",
		Version:         2, // Incremented for google-api inheritance
		Name:            "Google Sheets: 行追加",
		Description:     "Google Sheetsに行を追加",
		Category:        domain.BlockCategoryApps,
		Subcategory:     domain.BlockSubcategoryGoogle,
		Icon:            "table",
		ParentBlockSlug: "google-api",
		ConfigDefaults: json.RawMessage(`{
			"base_url": "https://sheets.googleapis.com/v4/spreadsheets"
		}`),
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"required": ["spreadsheet_id", "values"],
			"properties": {
				"range": {"type": "string", "title": "範囲", "default": "Sheet1!A:Z"},
				"values": {"type": "array", "title": "値", "x-ui-widget": "output-schema"},
				"api_key": {"type": "string", "title": "API Key"},
				"spreadsheet_id": {"type": "string", "title": "スプレッドシートID"},
				"value_input_option": {"enum": ["RAW", "USER_ENTERED"], "type": "string", "title": "入力形式", "default": "USER_ENTERED"}
			}
		}`),
		InputPorts:  []domain.InputPort{},
		OutputPorts: []domain.OutputPort{},
		OutputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"updated_range": {"type": "string"},
				"updated_rows": {"type": "number"},
				"updated_cells": {"type": "number"}
			}
		}`),
		PreProcess: `
const range = encodeURIComponent(config.range || 'Sheet1!A:Z');
const valueInputOption = config.value_input_option || 'USER_ENTERED';
let values = config.values;
if (typeof values === 'string') {
    values = JSON.parse(renderTemplate(values, input));
}
return {
    ...input,
    url: '/' + config.spreadsheet_id + '/values/' + range + ':append?valueInputOption=' + valueInputOption,
    method: 'POST',
    body: { values: values }
};
`,
		PostProcess: `
if (input.status === 404) {
    throw new Error('[GSHEETS_003] スプレッドシートが見つかりません');
}
if (input.status >= 400) {
    const errorMsg = input.body?.error?.message || 'Unknown error';
    throw new Error('[GSHEETS_002] 行追加失敗: ' + errorMsg);
}
return {
    updated_range: input.body.updates?.updatedRange,
    updated_rows: input.body.updates?.updatedRows,
    updated_cells: input.body.updates?.updatedCells
};
`,
		UIConfig: json.RawMessage(`{"icon": "table", "color": "#0F9D58"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "GSHEETS_001", Name: "API_KEY_NOT_CONFIGURED", Description: "API Keyが設定されていません", Retryable: false},
			{Code: "GSHEETS_002", Name: "APPEND_FAILED", Description: "行追加に失敗しました", Retryable: true},
			{Code: "GSHEETS_003", Name: "INVALID_SPREADSHEET", Description: "スプレッドシートが見つかりません", Retryable: false},
		},
		Enabled: true,
	}
}

// GSheetsReadBlock reads data from Google Sheets
func GSheetsReadBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:            "gsheets_read",
		Version:         2, // Incremented for google-api inheritance
		Name:            "Google Sheets: 読み取り",
		Description:     "Google Sheetsから範囲を読み取り",
		Category:        domain.BlockCategoryApps,
		Subcategory:     domain.BlockSubcategoryGoogle,
		Icon:            "table",
		ParentBlockSlug: "google-api",
		ConfigDefaults: json.RawMessage(`{
			"base_url": "https://sheets.googleapis.com/v4/spreadsheets"
		}`),
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"required": ["spreadsheet_id", "range"],
			"properties": {
				"range": {"type": "string", "title": "範囲"},
				"api_key": {"type": "string", "title": "API Key"},
				"spreadsheet_id": {"type": "string", "title": "スプレッドシートID"},
				"major_dimension": {"enum": ["ROWS", "COLUMNS"], "type": "string", "title": "次元", "default": "ROWS"}
			}
		}`),
		InputPorts:  []domain.InputPort{},
		OutputPorts: []domain.OutputPort{},
		OutputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"range": {"type": "string"},
				"values": {"type": "array"}
			}
		}`),
		PreProcess: `
const range = encodeURIComponent(config.range);
const majorDimension = config.major_dimension || 'ROWS';
return {
    ...input,
    url: '/' + config.spreadsheet_id + '/values/' + range + '?majorDimension=' + majorDimension,
    method: 'GET'
};
`,
		PostProcess: `
if (input.status === 404) {
    throw new Error('[GSHEETS_003] スプレッドシートが見つかりません');
}
if (input.status >= 400) {
    const errorMsg = input.body?.error?.message || 'Unknown error';
    throw new Error('[GSHEETS_004] 読み取り失敗: ' + errorMsg);
}
return {
    range: input.body.range,
    values: input.body.values || []
};
`,
		UIConfig: json.RawMessage(`{"icon": "table", "color": "#0F9D58"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "GSHEETS_001", Name: "API_KEY_NOT_CONFIGURED", Description: "API Keyが設定されていません", Retryable: false},
			{Code: "GSHEETS_003", Name: "INVALID_SPREADSHEET", Description: "スプレッドシートが見つかりません", Retryable: false},
			{Code: "GSHEETS_004", Name: "READ_FAILED", Description: "読み取りに失敗しました", Retryable: true},
		},
		Enabled: true,
	}
}

// GitHubCreateIssueBlock creates an issue on GitHub
func GitHubCreateIssueBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:            "github_create_issue",
		Version:         2, // Incremented for github-api inheritance
		Name:            "GitHub: Issue作成",
		Description:     "GitHubリポジトリにIssueを作成",
		Category:        domain.BlockCategoryApps,
		Subcategory:     domain.BlockSubcategoryGitHub,
		Icon:            "git-pull-request",
		ParentBlockSlug: "github-api",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"required": ["owner", "repo", "title"],
			"properties": {
				"body": {"type": "string", "title": "本文", "x-ui-widget": "textarea"},
				"repo": {"type": "string", "title": "リポジトリ"},
				"owner": {"type": "string", "title": "オーナー"},
				"title": {"type": "string", "title": "タイトル"},
				"token": {"type": "string", "title": "アクセストークン"},
				"labels": {"type": "array", "items": {"type": "string"}, "title": "ラベル"},
				"assignees": {"type": "array", "items": {"type": "string"}, "title": "アサイン"}
			}
		}`),
		InputPorts:  []domain.InputPort{},
		OutputPorts: []domain.OutputPort{},
		OutputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"id": {"type": "number"},
				"number": {"type": "number"},
				"url": {"type": "string"},
				"html_url": {"type": "string"}
			}
		}`),
		PreProcess: `
const payload = {
    title: renderTemplate(config.title, input),
    body: config.body ? renderTemplate(config.body, input) : undefined,
    labels: config.labels,
    assignees: config.assignees
};
return {
    ...input,
    url: '/repos/' + config.owner + '/' + config.repo + '/issues',
    method: 'POST',
    body: payload
};
`,
		PostProcess: `
if (input.status >= 400) {
    const errorMsg = input.body?.message || 'Unknown error';
    throw new Error('[GITHUB_002] Issue作成失敗: ' + errorMsg);
}
return {
    id: input.body.id,
    number: input.body.number,
    url: input.body.url,
    html_url: input.body.html_url
};
`,
		UIConfig: json.RawMessage(`{"icon": "git-pull-request", "color": "#24292F"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "GITHUB_001", Name: "TOKEN_NOT_CONFIGURED", Description: "トークンが設定されていません", Retryable: false},
			{Code: "GITHUB_002", Name: "CREATE_FAILED", Description: "Issue作成に失敗しました", Retryable: true},
			{Code: "GITHUB_003", Name: "REPO_NOT_FOUND", Description: "リポジトリが見つかりません", Retryable: false},
		},
		Enabled: true,
	}
}

// GitHubAddCommentBlock adds a comment to a GitHub issue or PR
func GitHubAddCommentBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:            "github_add_comment",
		Version:         2, // Incremented for github-api inheritance
		Name:            "GitHub: コメント追加",
		Description:     "GitHub IssueまたはPRにコメントを追加",
		Category:        domain.BlockCategoryApps,
		Subcategory:     domain.BlockSubcategoryGitHub,
		Icon:            "message-square",
		ParentBlockSlug: "github-api",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"required": ["owner", "repo", "issue_number", "body"],
			"properties": {
				"body": {"type": "string", "title": "コメント本文", "x-ui-widget": "textarea"},
				"repo": {"type": "string", "title": "リポジトリ"},
				"owner": {"type": "string", "title": "オーナー"},
				"token": {"type": "string", "title": "アクセストークン"},
				"issue_number": {"type": "number", "title": "Issue/PR番号"}
			}
		}`),
		InputPorts:  []domain.InputPort{},
		OutputPorts: []domain.OutputPort{},
		OutputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"id": {"type": "number"},
				"url": {"type": "string"},
				"html_url": {"type": "string"}
			}
		}`),
		PreProcess: `
return {
    ...input,
    url: '/repos/' + config.owner + '/' + config.repo + '/issues/' + config.issue_number + '/comments',
    method: 'POST',
    body: { body: renderTemplate(config.body, input) }
};
`,
		PostProcess: `
if (input.status >= 400) {
    const errorMsg = input.body?.message || 'Unknown error';
    throw new Error('[GITHUB_004] コメント追加失敗: ' + errorMsg);
}
return {
    id: input.body.id,
    url: input.body.url,
    html_url: input.body.html_url
};
`,
		UIConfig: json.RawMessage(`{"icon": "message-square", "color": "#24292F"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "GITHUB_001", Name: "TOKEN_NOT_CONFIGURED", Description: "トークンが設定されていません", Retryable: false},
			{Code: "GITHUB_004", Name: "COMMENT_FAILED", Description: "コメント追加に失敗しました", Retryable: true},
		},
		Enabled: true,
	}
}

// WebSearchBlock performs web search using Tavily API
func WebSearchBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:            "web_search",
		Version:         2, // Incremented for rest-api inheritance
		Name:            "Web検索",
		Description:     "Tavily APIでWeb検索を実行",
		Category:        domain.BlockCategoryApps,
		Subcategory:     domain.BlockSubcategoryWeb,
		Icon:            "search",
		ParentBlockSlug: "rest-api",
		ConfigDefaults: json.RawMessage(`{
			"base_url": "https://api.tavily.com",
			"secret_key": "TAVILY_API_KEY"
		}`),
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"required": ["query"],
			"properties": {
				"query": {"type": "string", "title": "検索クエリ"},
				"api_key": {"type": "string", "title": "API Key"},
				"max_results": {"type": "number", "title": "最大結果数", "default": 5, "maximum": 20, "minimum": 1},
				"search_depth": {"enum": ["basic", "advanced"], "type": "string", "title": "検索深度", "default": "basic"},
				"include_answer": {"type": "boolean", "title": "AI回答を含める", "default": true},
				"exclude_domains": {"type": "array", "items": {"type": "string"}, "title": "除外ドメイン"},
				"include_domains": {"type": "array", "items": {"type": "string"}, "title": "含めるドメイン"}
			}
		}`),
		InputPorts:  []domain.InputPort{},
		OutputPorts: []domain.OutputPort{},
		OutputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"answer": {"type": "string"},
				"results": {"type": "array"}
			}
		}`),
		PreProcess: `
const apiKey = config.api_key || (config.secret_key ? ctx.secrets[config.secret_key] : null);
const payload = {
    api_key: apiKey,
    query: renderTemplate(config.query, input),
    search_depth: config.search_depth || 'basic',
    max_results: config.max_results || 5,
    include_answer: config.include_answer !== false
};
if (config.include_domains && config.include_domains.length > 0) {
    payload.include_domains = config.include_domains;
}
if (config.exclude_domains && config.exclude_domains.length > 0) {
    payload.exclude_domains = config.exclude_domains;
}
return {
    ...input,
    url: '/search',
    method: 'POST',
    body: payload
};
`,
		PostProcess: `
if (input.status >= 400) {
    throw new Error('[SEARCH_002] 検索失敗: ' + (input.body?.error || input.status));
}
return {
    answer: input.body.answer,
    results: input.body.results
};
`,
		UIConfig: json.RawMessage(`{"icon": "search", "color": "#4285F4"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "SEARCH_001", Name: "API_KEY_NOT_CONFIGURED", Description: "API Keyが設定されていません", Retryable: false},
			{Code: "SEARCH_002", Name: "SEARCH_FAILED", Description: "検索に失敗しました", Retryable: true},
		},
		Enabled: true,
	}
}

// LinearCreateIssueBlock creates an issue in Linear
func LinearCreateIssueBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:            "linear_create_issue",
		Version:         2, // Incremented for linear-api inheritance
		Name:            "Linear: Issue作成",
		Description:     "LinearにIssueを作成",
		Category:        domain.BlockCategoryApps,
		Subcategory:     domain.BlockSubcategoryLinear,
		Icon:            "check-square",
		ParentBlockSlug: "linear-api",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"required": ["team_id", "title"],
			"properties": {
				"title": {"type": "string", "title": "タイトル"},
				"api_key": {"type": "string", "title": "API Key"},
				"team_id": {"type": "string", "title": "チームID"},
				"priority": {"enum": [0, 1, 2, 3, 4], "type": "number", "title": "優先度", "default": 0},
				"label_ids": {"type": "array", "items": {"type": "string"}, "title": "ラベルID"},
				"assignee_id": {"type": "string", "title": "担当者ID"},
				"description": {"type": "string", "title": "説明", "x-ui-widget": "textarea"}
			}
		}`),
		InputPorts:  []domain.InputPort{},
		OutputPorts: []domain.OutputPort{},
		OutputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"id": {"type": "string"},
				"identifier": {"type": "string"},
				"url": {"type": "string"}
			}
		}`),
		PreProcess: `
const mutation = 'mutation IssueCreate($input: IssueCreateInput!) { issueCreate(input: $input) { success issue { id identifier url } } }';
const variables = {
    input: {
        teamId: config.team_id,
        title: renderTemplate(config.title, input),
        description: config.description ? renderTemplate(config.description, input) : undefined,
        priority: config.priority,
        labelIds: config.label_ids,
        assigneeId: config.assignee_id
    }
};
return {
    ...input,
    query: mutation,
    variables: variables
};
`,
		PostProcess: `
if (input.status >= 400) {
    throw new Error('[LINEAR_002] Issue作成失敗: ' + input.status);
}
// GraphQL response is already processed by graphql block
const issue = input.issueCreate?.issue || input;
return {
    id: issue.id,
    identifier: issue.identifier,
    url: issue.url
};
`,
		UIConfig: json.RawMessage(`{"icon": "check-square", "color": "#5E6AD2"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "LINEAR_001", Name: "API_KEY_NOT_CONFIGURED", Description: "API Keyが設定されていません", Retryable: false},
			{Code: "LINEAR_002", Name: "CREATE_FAILED", Description: "Issue作成に失敗しました", Retryable: true},
		},
		Enabled: true,
	}
}

// EmailSendGridBlock sends email via SendGrid API
func EmailSendGridBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:            "email_sendgrid",
		Version:         2, // Incremented for bearer-api inheritance
		Name:            "Email (SendGrid)",
		Description:     "SendGrid APIでメールを送信",
		Category:        domain.BlockCategoryApps,
		Subcategory:     domain.BlockSubcategoryEmail,
		Icon:            "mail",
		ParentBlockSlug: "bearer-api",
		ConfigDefaults: json.RawMessage(`{
			"base_url": "https://api.sendgrid.com/v3",
			"secret_key": "SENDGRID_API_KEY"
		}`),
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"required": ["from_email", "to_email", "subject", "content"],
			"properties": {
				"api_key": {"type": "string", "title": "API Key"},
				"content": {"type": "string", "title": "本文", "x-ui-widget": "textarea"},
				"subject": {"type": "string", "title": "件名"},
				"to_name": {"type": "string", "title": "受信者名"},
				"to_email": {"type": "string", "title": "宛先メール"},
				"from_name": {"type": "string", "title": "送信者名"},
				"from_email": {"type": "string", "title": "送信元メール"},
				"content_type": {"enum": ["text/plain", "text/html"], "type": "string", "title": "本文形式", "default": "text/plain"}
			}
		}`),
		InputPorts:  []domain.InputPort{},
		OutputPorts: []domain.OutputPort{},
		OutputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"success": {"type": "boolean"},
				"message_id": {"type": "string"}
			}
		}`),
		PreProcess: `
// api_key -> token にマッピング
if (config.api_key && !config.token) {
    config.token = config.api_key;
}
const payload = {
    personalizations: [{
        to: [{
            email: renderTemplate(config.to_email, input),
            name: config.to_name ? renderTemplate(config.to_name, input) : undefined
        }]
    }],
    from: {
        email: config.from_email,
        name: config.from_name
    },
    subject: renderTemplate(config.subject, input),
    content: [{
        type: config.content_type || 'text/plain',
        value: renderTemplate(config.content, input)
    }]
};
return {
    ...input,
    url: '/mail/send',
    method: 'POST',
    body: payload
};
`,
		PostProcess: `
if (input.status >= 400) {
    const errors = input.body?.errors?.map(e => e.message).join(', ') || 'Unknown error';
    throw new Error('[EMAIL_002] メール送信失敗: ' + errors);
}
return {
    success: true,
    message_id: input.headers?.['x-message-id'] || null
};
`,
		UIConfig: json.RawMessage(`{"icon": "mail", "color": "#1A82E2"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "EMAIL_001", Name: "API_KEY_NOT_CONFIGURED", Description: "API Keyが設定されていません", Retryable: false},
			{Code: "EMAIL_002", Name: "SEND_FAILED", Description: "メール送信に失敗しました", Retryable: true},
			{Code: "EMAIL_003", Name: "INVALID_EMAIL", Description: "メールアドレスが無効です", Retryable: false},
		},
		Enabled: true,
	}
}

func EmbeddingBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "embedding",
		Version:     1,
		Name:        "Embedding",
		Description: "Convert text to vector embeddings",
		Category:    domain.BlockCategoryAI,
		Subcategory: domain.BlockSubcategoryRAG,
		Icon:        "hash",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"provider": {"type": "string", "enum": ["openai", "cohere", "voyage"], "default": "openai", "title": "Provider"},
				"model": {"type": "string", "default": "text-embedding-3-small", "title": "Model"}
			}
		}`),
		InputPorts: []domain.InputPort{
			{Name: "input", Label: "Input", Schema: json.RawMessage(`{"type": "object"}`), Required: true, Description: "Documents or text to embed"},
		},
		OutputPorts: []domain.OutputPort{
			{Name: "output", Label: "Output", IsDefault: true, Description: "Documents with vectors"},
		},
		Code: `
const documents = input.documents || (input.texts ? input.texts.map(t => ({content: t})) : (input.text ? [{content: input.text}] : []));
if (documents.length === 0) throw new Error('[EMB_002] No text provided for embedding');
const provider = config.provider || 'openai';
const model = config.model || 'text-embedding-3-small';
const texts = documents.map(d => d.content);
const result = ctx.embedding.embed(provider, model, texts);
const docsWithVectors = documents.map((doc, i) => ({...doc, vector: result.vectors[i]}));
return {documents: docsWithVectors, vectors: result.vectors, model: result.model, dimension: result.dimension, usage: result.usage};
`,
		UIConfig: json.RawMessage(`{"icon": "hash", "color": "#8B5CF6"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "EMB_001", Name: "PROVIDER_ERROR", Description: "Embedding provider API error", Retryable: true},
			{Code: "EMB_002", Name: "EMPTY_INPUT", Description: "No text provided for embedding", Retryable: false},
		},
		Enabled: true,
	}
}

func VectorUpsertBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "vector-upsert",
		Version:     1,
		Name:        "Vector Upsert",
		Description: "Store documents in vector database",
		Category:    domain.BlockCategoryAI,
		Subcategory: domain.BlockSubcategoryRAG,
		Icon:        "database",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"required": ["collection"],
			"properties": {
				"collection": {"type": "string", "title": "Collection Name"},
				"embedding_provider": {"type": "string", "default": "openai", "title": "Embedding Provider"},
				"embedding_model": {"type": "string", "default": "text-embedding-3-small", "title": "Embedding Model"}
			}
		}`),
		InputPorts: []domain.InputPort{
			{Name: "documents", Label: "Documents", Schema: json.RawMessage(`{"type": "array"}`), Required: true, Description: "Documents to store"},
		},
		OutputPorts: []domain.OutputPort{
			{Name: "output", Label: "Output", IsDefault: true, Description: "Upsert result"},
		},
		Code: `
const collection = config.collection || input.collection;
if (!collection) throw new Error('[VEC_001] Collection name is required');
const documents = input.documents;
if (!documents || documents.length === 0) throw new Error('[VEC_002] Documents array is required');
const result = ctx.vector.upsert(collection, documents, {embedding_provider: config.embedding_provider, embedding_model: config.embedding_model});
return {collection, upserted_count: result.upserted_count, ids: result.ids};
`,
		UIConfig: json.RawMessage(`{"icon": "database", "color": "#10B981"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "VEC_001", Name: "COLLECTION_REQUIRED", Description: "Collection name is required", Retryable: false},
			{Code: "VEC_002", Name: "DOCUMENTS_REQUIRED", Description: "Documents array is required", Retryable: false},
		},
		Enabled: true,
	}
}

func VectorSearchBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "vector-search",
		Version:     1,
		Name:        "Vector Search",
		Description: "Search for similar documents in vector database",
		Category:    domain.BlockCategoryAI,
		Subcategory: domain.BlockSubcategoryRAG,
		Icon:        "search",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"required": ["collection"],
			"properties": {
				"collection": {"type": "string", "title": "Collection Name"},
				"top_k": {"type": "integer", "default": 5, "minimum": 1, "maximum": 100, "title": "Number of Results"},
				"threshold": {"type": "number", "minimum": 0, "maximum": 1, "title": "Similarity Threshold"},
				"include_content": {"type": "boolean", "default": true, "title": "Include Content"},
				"embedding_provider": {"type": "string", "default": "openai"},
				"embedding_model": {"type": "string", "default": "text-embedding-3-small"}
			}
		}`),
		InputPorts: []domain.InputPort{
			{Name: "input", Label: "Input", Schema: json.RawMessage(`{"type": "object"}`), Required: true, Description: "Vector or query text"},
		},
		OutputPorts: []domain.OutputPort{
			{Name: "output", Label: "Output", IsDefault: true, Description: "Search results"},
		},
		Code: `
const collection = config.collection || input.collection;
if (!collection) throw new Error('[VEC_001] Collection name is required');
let searchVector = input.vector || (input.vectors ? input.vectors[0] : null);
if (!searchVector && input.query) {
  const provider = config.embedding_provider || 'openai';
  const model = config.embedding_model || 'text-embedding-3-small';
  const embedResult = ctx.embedding.embed(provider, model, [input.query]);
  searchVector = embedResult.vectors[0];
}
if (!searchVector) throw new Error('[VEC_003] Either vector or query text is required');
const result = ctx.vector.query(collection, searchVector, {top_k: config.top_k || 5, threshold: config.threshold, include_content: config.include_content !== false});
return {matches: result.matches, count: result.matches.length, collection};
`,
		UIConfig: json.RawMessage(`{"icon": "search", "color": "#3B82F6"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "VEC_001", Name: "COLLECTION_REQUIRED", Description: "Collection name is required", Retryable: false},
			{Code: "VEC_003", Name: "VECTOR_OR_QUERY_REQUIRED", Description: "Either vector or query text is required", Retryable: false},
		},
		Enabled: true,
	}
}

func VectorDeleteBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "vector-delete",
		Version:     1,
		Name:        "Vector Delete",
		Description: "Delete documents from vector database",
		Category:    domain.BlockCategoryAI,
		Subcategory: domain.BlockSubcategoryRAG,
		Icon:        "trash-2",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"required": ["collection"],
			"properties": {
				"collection": {"type": "string", "title": "Collection Name"}
			}
		}`),
		InputPorts: []domain.InputPort{
			{Name: "input", Label: "Input", Schema: json.RawMessage(`{"type": "object"}`), Required: true, Description: "IDs to delete"},
		},
		OutputPorts: []domain.OutputPort{
			{Name: "output", Label: "Output", IsDefault: true, Description: "Delete result"},
		},
		Code: `
const collection = config.collection || input.collection;
if (!collection) throw new Error('[VEC_001] Collection name is required');
const ids = input.ids || (input.id ? [input.id] : null);
if (!ids || ids.length === 0) throw new Error('[VEC_004] IDs array is required');
const result = ctx.vector.delete(collection, ids);
return {collection, deleted_count: result.deleted_count, requested_ids: ids};
`,
		UIConfig: json.RawMessage(`{"icon": "trash-2", "color": "#EF4444"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "VEC_001", Name: "COLLECTION_REQUIRED", Description: "Collection name is required", Retryable: false},
			{Code: "VEC_004", Name: "IDS_REQUIRED", Description: "IDs array is required", Retryable: false},
		},
		Enabled: true,
	}
}
