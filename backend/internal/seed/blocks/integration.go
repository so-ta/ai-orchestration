package blocks

import (
	"encoding/json"

	"github.com/souta/ai-orchestration/internal/domain"
)

func (r *Registry) registerIntegrationBlocks() {
	// === Level 0: Base blocks ===
	// http, rest-api, bearer-api are defined in YAML (integration_http.yaml)
	r.register(SubflowBlock())
	r.register(ToolBlock())

	// === Level 1: Foundation pattern blocks ===
	// webhook is defined in YAML (integration_webhook.yaml)
	r.register(GraphQLBlock())

	// === Level 2: Authentication pattern blocks ===
	// bearer-api is defined in YAML (integration_http.yaml)
	r.register(APIKeyHeaderBlock())
	r.register(APIKeyQueryBlock())

	// === Level 3: Service-specific base blocks ===
	// github-api is defined in YAML (integration_github.yaml)
	// notion-api is defined in YAML (integration_notion.yaml)
	// linear-api is defined in YAML (integration_linear.yaml)
	// google-api is defined in YAML (integration_google.yaml)

	// === Level 4+: Concrete operation blocks ===
	// slack, discord are defined in YAML (integration_webhook.yaml)
	// github_create_issue, github_add_comment are defined in YAML (integration_github.yaml)
	// Notion: notion_query_db is defined in YAML (integration_notion.yaml)
	r.register(NotionCreatePageBlock())
	// Google Sheets
	r.register(GSheetsAppendBlock())
	// gsheets_read is defined in YAML (integration_google.yaml)
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

// HTTPBlock, WebhookBlock, RestAPIBlock, BearerAPIBlock are defined in YAML files
// See: yaml/integration_http.yaml, yaml/integration_webhook.yaml

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

// WebhookBlock and RestAPIBlock are defined in YAML files
// See: yaml/integration_http.yaml, yaml/integration_webhook.yaml

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

// BearerAPIBlock is defined in YAML files
// See: yaml/integration_http.yaml

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

// GitHubAPIBlock is defined in YAML files
// See: yaml/integration_github.yaml

// NotionAPIBlock is defined in YAML files
// See: yaml/integration_notion.yaml

// LinearAPIBlock is defined in YAML files
// See: yaml/integration_linear.yaml

// GoogleAPIBlock is defined in YAML files
// See: yaml/integration_google.yaml

// =============================================================================
// Level 4+: Concrete Operation Blocks
// =============================================================================

// SlackBlock and DiscordBlock are defined in YAML files
// See: yaml/integration_webhook.yaml

// NotionQueryDBBlock is defined in YAML files
// See: yaml/integration_notion.yaml

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

// GSheetsReadBlock is defined in YAML files
// See: yaml/integration_google.yaml

// GitHubCreateIssueBlock and GitHubAddCommentBlock are defined in YAML files
// See: yaml/integration_github.yaml

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
			"properties": {
				"collection": {"type": "string", "title": "Collection Name", "description": "Collection name (can also be provided via input.collection)"},
				"embedding_provider": {"type": "string", "default": "openai", "title": "Embedding Provider"},
				"embedding_model": {"type": "string", "default": "text-embedding-3-small", "title": "Embedding Model"}
			}
		}`),
		InputPorts: []domain.InputPort{
			{Name: "input", Label: "Input", Schema: json.RawMessage(`{"type": "object"}`), Required: true, Description: "Documents and optional collection name"},
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
			"properties": {
				"collection": {"type": "string", "title": "Collection Name", "description": "Collection name (can also be provided via input.collection)"},
				"top_k": {"type": "integer", "default": 5, "minimum": 1, "maximum": 100, "title": "Number of Results"},
				"threshold": {"type": "number", "minimum": 0, "maximum": 1, "title": "Similarity Threshold"},
				"include_content": {"type": "boolean", "default": true, "title": "Include Content"},
				"embedding_provider": {"type": "string", "default": "openai"},
				"embedding_model": {"type": "string", "default": "text-embedding-3-small"}
			}
		}`),
		InputPorts: []domain.InputPort{
			{Name: "input", Label: "Input", Schema: json.RawMessage(`{"type": "object"}`), Required: true, Description: "Query text, vector, and optional collection name"},
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
