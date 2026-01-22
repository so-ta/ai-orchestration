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
		Name:        LText("Subflow", "サブフロー"),
		Description: LText("Execute another workflow", "別のワークフローを実行"),
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryControl,
		Icon:        "workflow",
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"workflow_id": {"type": "string", "format": "uuid", "title": "Workflow ID", "description": "ID of the workflow to execute"},
				"workflow_version": {"type": "integer", "title": "Version", "description": "Workflow version to use"}
			}
		}`, `{
			"type": "object",
			"properties": {
				"workflow_id": {"type": "string", "format": "uuid", "title": "ワークフローID", "description": "実行するワークフローのID"},
				"workflow_version": {"type": "integer", "title": "バージョン", "description": "使用するワークフローバージョン"}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("output", "Output", "出力", "Subflow result", "サブフローの結果", true),
		},
		Code:     `return ctx.workflow.run(config.workflow_id, input);`,
		UIConfig: LSchema(`{"icon": "workflow", "color": "#10B981"}`, `{"icon": "workflow", "color": "#10B981"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("SUBFLOW_001", "NOT_FOUND", "見つからない", "Subflow workflow not found", "サブフローワークフローが見つかりません", false),
			LError("SUBFLOW_002", "EXEC_ERROR", "実行エラー", "Subflow execution error", "サブフローの実行エラー", true),
		},
		Enabled: true,
	}
}

func ToolBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "tool",
		Version:     1,
		Name:        LText("Tool", "ツール"),
		Description: LText("Execute external tool/adapter", "外部ツール/アダプターを実行"),
		Category:    domain.BlockCategoryApps,
		Subcategory: domain.BlockSubcategoryWeb,
		Icon:        "wrench",
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"adapter_id": {"type": "string", "title": "Adapter ID", "description": "ID of the adapter to use"},
				"enable_error_port": {
					"type": "boolean",
					"title": "Enable Error Port",
					"description": "Output to dedicated error port on error",
					"default": false
				}
			}
		}`, `{
			"type": "object",
			"properties": {
				"adapter_id": {"type": "string", "title": "アダプターID", "description": "使用するアダプターのID"},
				"enable_error_port": {
					"type": "boolean",
					"title": "エラーハンドルを有効化",
					"description": "エラー発生時に専用のエラーポートに出力します",
					"default": false
				}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("output", "Output", "出力", "Tool execution result", "ツール実行結果", true),
		},
		Code:     `return ctx.adapter.call(config.adapter_id, input);`,
		UIConfig: LSchema(`{"icon": "wrench", "color": "#10B981"}`, `{"icon": "wrench", "color": "#10B981"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("TOOL_001", "ADAPTER_NOT_FOUND", "アダプター未発見", "Adapter not found", "アダプターが見つかりません", false),
			LError("TOOL_002", "EXEC_ERROR", "実行エラー", "Tool execution error", "ツールの実行エラー", true),
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
		Name:            LText("GraphQL", "GraphQL"),
		Description:     LText("Base block for GraphQL API calls", "GraphQL API呼び出しの基盤ブロック"),
		Category:        domain.BlockCategoryApps,
		Subcategory:     domain.BlockSubcategoryWeb,
		Icon:            "hexagon",
		ParentBlockSlug: "rest-api",
		ConfigDefaults: json.RawMessage(`{
			"auth_type": "bearer"
		}`),
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"endpoint": {"type": "string", "title": "GraphQL Endpoint", "description": "GraphQL API endpoint URL"},
				"query": {"type": "string", "title": "GraphQL Query", "description": "GraphQL query string", "x-ui-widget": "textarea"},
				"variables": {"type": "object", "title": "Variables", "description": "Query variables"}
			}
		}`, `{
			"type": "object",
			"properties": {
				"endpoint": {"type": "string", "title": "GraphQLエンドポイント", "description": "GraphQL APIエンドポイントURL"},
				"query": {"type": "string", "title": "GraphQLクエリ", "description": "GraphQLクエリ文字列", "x-ui-widget": "textarea"},
				"variables": {"type": "object", "title": "変数", "description": "クエリ変数"}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{},
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
		UIConfig: LSchema(`{"icon": "hexagon", "color": "#E535AB"}`, `{"icon": "hexagon", "color": "#E535AB"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("GQL_001", "QUERY_REQUIRED", "クエリ必須", "Query is not specified", "クエリが指定されていません", false),
			LError("GQL_002", "GRAPHQL_ERROR", "GraphQLエラー", "GraphQL error", "GraphQLエラー", false),
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
		Name:            LText("API Key (Header)", "APIキー（ヘッダー）"),
		Description:     LText("API Key header authentication base", "APIキーヘッダー認証基盤"),
		Category:        domain.BlockCategoryApps,
		Subcategory:     domain.BlockSubcategoryWeb,
		Icon:            "key",
		ParentBlockSlug: "rest-api",
		ConfigDefaults: json.RawMessage(`{
			"auth_type": "api_key_header"
		}`),
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"api_key": {"type": "string", "title": "API Key", "description": "API key (direct specification)"},
				"header_name": {"type": "string", "title": "Header Name", "description": "Header name for API key", "default": "X-API-Key"}
			}
		}`, `{
			"type": "object",
			"properties": {
				"api_key": {"type": "string", "title": "APIキー", "description": "APIキー（直接指定）"},
				"header_name": {"type": "string", "title": "ヘッダー名", "description": "APIキー用のヘッダー名", "default": "X-API-Key"}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{},
		PreProcess: `
// api_key -> auth_key にマッピング
if (config.api_key && !config.auth_key) {
    config.auth_key = config.api_key;
}
return input;
`,
		UIConfig: LSchema(`{"icon": "key", "color": "#10B981"}`, `{"icon": "key", "color": "#10B981"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("APIKEY_001", "KEY_NOT_CONFIGURED", "キー未設定", "API key is not configured", "APIキーが設定されていません", false),
		},
		Enabled: true,
	}
}

// APIKeyQueryBlock provides API Key query parameter authentication
func APIKeyQueryBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:            "api-key-query",
		Version:         1,
		Name:            LText("API Key (Query)", "APIキー（クエリ）"),
		Description:     LText("API Key query parameter authentication base", "APIキークエリパラメータ認証基盤"),
		Category:        domain.BlockCategoryApps,
		Subcategory:     domain.BlockSubcategoryWeb,
		Icon:            "key",
		ParentBlockSlug: "rest-api",
		ConfigDefaults: json.RawMessage(`{
			"auth_type": "api_key_query"
		}`),
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"api_key": {"type": "string", "title": "API Key", "description": "API key (direct specification)"}
			}
		}`, `{
			"type": "object",
			"properties": {
				"api_key": {"type": "string", "title": "APIキー", "description": "APIキー（直接指定）"}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{},
		PreProcess: `
// api_key -> auth_key にマッピング
if (config.api_key && !config.auth_key) {
    config.auth_key = config.api_key;
}
return input;
`,
		UIConfig: LSchema(`{"icon": "key", "color": "#8B5CF6"}`, `{"icon": "key", "color": "#8B5CF6"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("APIKEY_001", "KEY_NOT_CONFIGURED", "キー未設定", "API key is not configured", "APIキーが設定されていません", false),
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
		Name:            LText("Notion: Create Page", "Notion: ページ作成"),
		Description:     LText("Create a page in Notion", "Notionにページを作成"),
		Category:        domain.BlockCategoryApps,
		Subcategory:     domain.BlockSubcategoryNotion,
		Icon:            "file-text",
		ParentBlockSlug: "notion-api",
		ConfigSchema: LSchema(`{
			"type": "object",
			"required": ["parent_id"],
			"properties": {
				"title": {"type": "string", "title": "Title", "description": "Page title"},
				"api_key": {"type": "string", "title": "API Key"},
				"content": {"type": "string", "title": "Content", "description": "Page content", "x-ui-widget": "textarea"},
				"parent_id": {"type": "string", "title": "Parent ID", "description": "Parent page or database ID"},
				"properties": {"type": "object", "title": "Properties", "description": "Page properties", "x-ui-widget": "output-schema"},
				"parent_type": {"enum": ["page_id", "database_id"], "type": "string", "title": "Parent Type", "description": "Type of parent", "default": "database_id"}
			}
		}`, `{
			"type": "object",
			"required": ["parent_id"],
			"properties": {
				"title": {"type": "string", "title": "タイトル", "description": "ページタイトル"},
				"api_key": {"type": "string", "title": "APIキー"},
				"content": {"type": "string", "title": "本文", "description": "ページ本文", "x-ui-widget": "textarea"},
				"parent_id": {"type": "string", "title": "親ID", "description": "親ページまたはデータベースのID"},
				"properties": {"type": "object", "title": "プロパティ", "description": "ページプロパティ", "x-ui-widget": "output-schema"},
				"parent_type": {"enum": ["page_id", "database_id"], "type": "string", "title": "親タイプ", "description": "親のタイプ", "default": "database_id"}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{},
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
		UIConfig: LSchema(`{"icon": "file-text", "color": "#000000"}`, `{"icon": "file-text", "color": "#000000"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("NOTION_001", "API_KEY_NOT_CONFIGURED", "APIキー未設定", "API Key is not configured", "APIキーが設定されていません", false),
			LError("NOTION_002", "CREATE_FAILED", "作成失敗", "Failed to create page", "ページの作成に失敗しました", true),
			LError("NOTION_003", "INVALID_PARENT", "無効な親", "Invalid parent ID", "無効な親IDです", false),
		},
		Enabled: true,
	}
}

// GSheetsAppendBlock appends rows to Google Sheets
func GSheetsAppendBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:            "gsheets_append",
		Version:         2, // Incremented for google-api inheritance
		Name:            LText("Google Sheets: Append Rows", "Google Sheets: 行追加"),
		Description:     LText("Append rows to Google Sheets", "Google Sheetsに行を追加"),
		Category:        domain.BlockCategoryApps,
		Subcategory:     domain.BlockSubcategoryGoogle,
		Icon:            "table",
		ParentBlockSlug: "google-api",
		ConfigDefaults: json.RawMessage(`{
			"base_url": "https://sheets.googleapis.com/v4/spreadsheets"
		}`),
		ConfigSchema: LSchema(`{
			"type": "object",
			"required": ["spreadsheet_id", "values"],
			"properties": {
				"range": {"type": "string", "title": "Range", "description": "Cell range", "default": "Sheet1!A:Z"},
				"values": {"type": "array", "title": "Values", "description": "Values to append", "x-ui-widget": "output-schema"},
				"api_key": {"type": "string", "title": "API Key"},
				"spreadsheet_id": {"type": "string", "title": "Spreadsheet ID", "description": "Google Sheets spreadsheet ID"},
				"value_input_option": {"enum": ["RAW", "USER_ENTERED"], "type": "string", "title": "Input Option", "description": "How values are interpreted", "default": "USER_ENTERED"}
			}
		}`, `{
			"type": "object",
			"required": ["spreadsheet_id", "values"],
			"properties": {
				"range": {"type": "string", "title": "範囲", "description": "セル範囲", "default": "Sheet1!A:Z"},
				"values": {"type": "array", "title": "値", "description": "追加する値", "x-ui-widget": "output-schema"},
				"api_key": {"type": "string", "title": "APIキー"},
				"spreadsheet_id": {"type": "string", "title": "スプレッドシートID", "description": "Google SheetsのスプレッドシートID"},
				"value_input_option": {"enum": ["RAW", "USER_ENTERED"], "type": "string", "title": "入力形式", "description": "値の解釈方法", "default": "USER_ENTERED"}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{},
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
		UIConfig: LSchema(`{"icon": "table", "color": "#0F9D58"}`, `{"icon": "table", "color": "#0F9D58"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("GSHEETS_001", "API_KEY_NOT_CONFIGURED", "APIキー未設定", "API Key is not configured", "APIキーが設定されていません", false),
			LError("GSHEETS_002", "APPEND_FAILED", "追加失敗", "Failed to append rows", "行の追加に失敗しました", true),
			LError("GSHEETS_003", "INVALID_SPREADSHEET", "スプレッドシート未発見", "Spreadsheet not found", "スプレッドシートが見つかりません", false),
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
		Name:            LText("Web Search", "Web検索"),
		Description:     LText("Perform web search using Tavily API", "Tavily APIでWeb検索を実行"),
		Category:        domain.BlockCategoryApps,
		Subcategory:     domain.BlockSubcategoryWeb,
		Icon:            "search",
		ParentBlockSlug: "rest-api",
		ConfigDefaults: json.RawMessage(`{
			"base_url": "https://api.tavily.com",
			"secret_key": "TAVILY_API_KEY"
		}`),
		ConfigSchema: LSchema(`{
			"type": "object",
			"required": ["query"],
			"properties": {
				"query": {"type": "string", "title": "Search Query", "description": "Search query string"},
				"api_key": {"type": "string", "title": "API Key"},
				"max_results": {"type": "number", "title": "Max Results", "description": "Maximum number of results", "default": 5, "maximum": 20, "minimum": 1},
				"search_depth": {"enum": ["basic", "advanced"], "type": "string", "title": "Search Depth", "description": "Search depth level", "default": "basic"},
				"include_answer": {"type": "boolean", "title": "Include AI Answer", "description": "Include AI-generated answer", "default": true},
				"exclude_domains": {"type": "array", "items": {"type": "string"}, "title": "Exclude Domains", "description": "Domains to exclude"},
				"include_domains": {"type": "array", "items": {"type": "string"}, "title": "Include Domains", "description": "Domains to include"}
			}
		}`, `{
			"type": "object",
			"required": ["query"],
			"properties": {
				"query": {"type": "string", "title": "検索クエリ", "description": "検索クエリ文字列"},
				"api_key": {"type": "string", "title": "APIキー"},
				"max_results": {"type": "number", "title": "最大結果数", "description": "結果の最大数", "default": 5, "maximum": 20, "minimum": 1},
				"search_depth": {"enum": ["basic", "advanced"], "type": "string", "title": "検索深度", "description": "検索深度レベル", "default": "basic"},
				"include_answer": {"type": "boolean", "title": "AI回答を含める", "description": "AI生成の回答を含める", "default": true},
				"exclude_domains": {"type": "array", "items": {"type": "string"}, "title": "除外ドメイン", "description": "除外するドメイン"},
				"include_domains": {"type": "array", "items": {"type": "string"}, "title": "含めるドメイン", "description": "含めるドメイン"}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{},
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
		UIConfig: LSchema(`{"icon": "search", "color": "#4285F4"}`, `{"icon": "search", "color": "#4285F4"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("SEARCH_001", "API_KEY_NOT_CONFIGURED", "APIキー未設定", "API Key is not configured", "APIキーが設定されていません", false),
			LError("SEARCH_002", "SEARCH_FAILED", "検索失敗", "Search failed", "検索に失敗しました", true),
		},
		Enabled: true,
	}
}

// LinearCreateIssueBlock creates an issue in Linear
func LinearCreateIssueBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:            "linear_create_issue",
		Version:         2, // Incremented for linear-api inheritance
		Name:            LText("Linear: Create Issue", "Linear: Issue作成"),
		Description:     LText("Create an issue in Linear", "LinearにIssueを作成"),
		Category:        domain.BlockCategoryApps,
		Subcategory:     domain.BlockSubcategoryLinear,
		Icon:            "check-square",
		ParentBlockSlug: "linear-api",
		ConfigSchema: LSchema(`{
			"type": "object",
			"required": ["team_id", "title"],
			"properties": {
				"title": {"type": "string", "title": "Title", "description": "Issue title"},
				"api_key": {"type": "string", "title": "API Key"},
				"team_id": {"type": "string", "title": "Team ID", "description": "Linear team ID"},
				"priority": {"enum": [0, 1, 2, 3, 4], "type": "number", "title": "Priority", "description": "Issue priority", "default": 0},
				"label_ids": {"type": "array", "items": {"type": "string"}, "title": "Label IDs", "description": "Label IDs to add"},
				"assignee_id": {"type": "string", "title": "Assignee ID", "description": "Assignee user ID"},
				"description": {"type": "string", "title": "Description", "description": "Issue description", "x-ui-widget": "textarea"}
			}
		}`, `{
			"type": "object",
			"required": ["team_id", "title"],
			"properties": {
				"title": {"type": "string", "title": "タイトル", "description": "Issueタイトル"},
				"api_key": {"type": "string", "title": "APIキー"},
				"team_id": {"type": "string", "title": "チームID", "description": "LinearのチームID"},
				"priority": {"enum": [0, 1, 2, 3, 4], "type": "number", "title": "優先度", "description": "Issueの優先度", "default": 0},
				"label_ids": {"type": "array", "items": {"type": "string"}, "title": "ラベルID", "description": "追加するラベルID"},
				"assignee_id": {"type": "string", "title": "担当者ID", "description": "担当者のユーザーID"},
				"description": {"type": "string", "title": "説明", "description": "Issueの説明", "x-ui-widget": "textarea"}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{},
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
		UIConfig: LSchema(`{"icon": "check-square", "color": "#5E6AD2"}`, `{"icon": "check-square", "color": "#5E6AD2"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("LINEAR_001", "API_KEY_NOT_CONFIGURED", "APIキー未設定", "API Key is not configured", "APIキーが設定されていません", false),
			LError("LINEAR_002", "CREATE_FAILED", "作成失敗", "Failed to create issue", "Issueの作成に失敗しました", true),
		},
		Enabled: true,
	}
}

// EmailSendGridBlock sends email via SendGrid API
func EmailSendGridBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:            "email_sendgrid",
		Version:         2, // Incremented for bearer-api inheritance
		Name:            LText("Email (SendGrid)", "メール（SendGrid）"),
		Description:     LText("Send email via SendGrid API", "SendGrid APIでメールを送信"),
		Category:        domain.BlockCategoryApps,
		Subcategory:     domain.BlockSubcategoryEmail,
		Icon:            "mail",
		ParentBlockSlug: "bearer-api",
		ConfigDefaults: json.RawMessage(`{
			"base_url": "https://api.sendgrid.com/v3",
			"secret_key": "SENDGRID_API_KEY"
		}`),
		ConfigSchema: LSchema(`{
			"type": "object",
			"required": ["from_email", "to_email", "subject", "content"],
			"properties": {
				"api_key": {"type": "string", "title": "API Key"},
				"content": {"type": "string", "title": "Content", "description": "Email body", "x-ui-widget": "textarea"},
				"subject": {"type": "string", "title": "Subject", "description": "Email subject"},
				"to_name": {"type": "string", "title": "To Name", "description": "Recipient name"},
				"to_email": {"type": "string", "title": "To Email", "description": "Recipient email address"},
				"from_name": {"type": "string", "title": "From Name", "description": "Sender name"},
				"from_email": {"type": "string", "title": "From Email", "description": "Sender email address"},
				"content_type": {"enum": ["text/plain", "text/html"], "type": "string", "title": "Content Type", "description": "Email content type", "default": "text/plain"}
			}
		}`, `{
			"type": "object",
			"required": ["from_email", "to_email", "subject", "content"],
			"properties": {
				"api_key": {"type": "string", "title": "APIキー"},
				"content": {"type": "string", "title": "本文", "description": "メール本文", "x-ui-widget": "textarea"},
				"subject": {"type": "string", "title": "件名", "description": "メール件名"},
				"to_name": {"type": "string", "title": "受信者名", "description": "受信者の名前"},
				"to_email": {"type": "string", "title": "宛先メール", "description": "受信者のメールアドレス"},
				"from_name": {"type": "string", "title": "送信者名", "description": "送信者の名前"},
				"from_email": {"type": "string", "title": "送信元メール", "description": "送信者のメールアドレス"},
				"content_type": {"enum": ["text/plain", "text/html"], "type": "string", "title": "本文形式", "description": "メールの本文形式", "default": "text/plain"}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{},
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
		UIConfig: LSchema(`{"icon": "mail", "color": "#1A82E2"}`, `{"icon": "mail", "color": "#1A82E2"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("EMAIL_001", "API_KEY_NOT_CONFIGURED", "APIキー未設定", "API Key is not configured", "APIキーが設定されていません", false),
			LError("EMAIL_002", "SEND_FAILED", "送信失敗", "Failed to send email", "メールの送信に失敗しました", true),
			LError("EMAIL_003", "INVALID_EMAIL", "無効なメール", "Invalid email address", "無効なメールアドレスです", false),
		},
		Enabled: true,
	}
}

func EmbeddingBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "embedding",
		Version:     1,
		Name:        LText("Embedding", "埋め込み"),
		Description: LText("Convert text to vector embeddings", "テキストをベクトル埋め込みに変換"),
		Category:    domain.BlockCategoryAI,
		Subcategory: domain.BlockSubcategoryRAG,
		Icon:        "hash",
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"provider": {"type": "string", "enum": ["openai", "cohere", "voyage"], "default": "openai", "title": "Provider", "description": "Embedding provider"},
				"model": {"type": "string", "default": "text-embedding-3-small", "title": "Model", "description": "Embedding model"}
			}
		}`, `{
			"type": "object",
			"properties": {
				"provider": {"type": "string", "enum": ["openai", "cohere", "voyage"], "default": "openai", "title": "プロバイダー", "description": "埋め込みプロバイダー"},
				"model": {"type": "string", "default": "text-embedding-3-small", "title": "モデル", "description": "埋め込みモデル"}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("output", "Output", "出力", "Documents with vectors", "ベクトル付きドキュメント", true),
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
		UIConfig: LSchema(`{"icon": "hash", "color": "#8B5CF6"}`, `{"icon": "hash", "color": "#8B5CF6"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("EMB_001", "PROVIDER_ERROR", "プロバイダーエラー", "Embedding provider API error", "埋め込みプロバイダーのAPIエラー", true),
			LError("EMB_002", "EMPTY_INPUT", "入力が空", "No text provided for embedding", "埋め込み用のテキストが提供されていません", false),
		},
		Enabled: true,
	}
}

func VectorUpsertBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "vector-upsert",
		Version:     1,
		Name:        LText("Vector Upsert", "ベクトルUpsert"),
		Description: LText("Store documents in vector database", "ベクトルデータベースにドキュメントを保存"),
		Category:    domain.BlockCategoryAI,
		Subcategory: domain.BlockSubcategoryRAG,
		Icon:        "database",
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"collection": {"type": "string", "title": "Collection Name", "description": "Collection name (can also be provided via input.collection)"},
				"embedding_provider": {"type": "string", "default": "openai", "title": "Embedding Provider"},
				"embedding_model": {"type": "string", "default": "text-embedding-3-small", "title": "Embedding Model"}
			}
		}`, `{
			"type": "object",
			"properties": {
				"collection": {"type": "string", "title": "コレクション名", "description": "コレクション名（input.collectionでも指定可能）"},
				"embedding_provider": {"type": "string", "default": "openai", "title": "埋め込みプロバイダー"},
				"embedding_model": {"type": "string", "default": "text-embedding-3-small", "title": "埋め込みモデル"}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("output", "Output", "出力", "Upsert result", "Upsert結果", true),
		},
		Code: `
const collection = config.collection || input.collection;
if (!collection) throw new Error('[VEC_001] Collection name is required');
const documents = input.documents;
if (!documents || documents.length === 0) throw new Error('[VEC_002] Documents array is required');
const result = ctx.vector.upsert(collection, documents, {embedding_provider: config.embedding_provider, embedding_model: config.embedding_model});
return {collection, upserted_count: result.upserted_count, ids: result.ids};
`,
		UIConfig: LSchema(`{"icon": "database", "color": "#10B981"}`, `{"icon": "database", "color": "#10B981"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("VEC_001", "COLLECTION_REQUIRED", "コレクション必須", "Collection name is required", "コレクション名が必要です", false),
			LError("VEC_002", "DOCUMENTS_REQUIRED", "ドキュメント必須", "Documents array is required", "ドキュメント配列が必要です", false),
		},
		Enabled: true,
	}
}

func VectorSearchBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "vector-search",
		Version:     1,
		Name:        LText("Vector Search", "ベクトル検索"),
		Description: LText("Search for similar documents in vector database", "ベクトルデータベースで類似ドキュメントを検索"),
		Category:    domain.BlockCategoryAI,
		Subcategory: domain.BlockSubcategoryRAG,
		Icon:        "search",
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"collection": {"type": "string", "title": "Collection Name", "description": "Collection name (can also be provided via input.collection)"},
				"top_k": {"type": "integer", "default": 5, "minimum": 1, "maximum": 100, "title": "Number of Results", "description": "Number of results to return"},
				"threshold": {"type": "number", "minimum": 0, "maximum": 1, "title": "Similarity Threshold", "description": "Minimum similarity score"},
				"include_content": {"type": "boolean", "default": true, "title": "Include Content", "description": "Include document content in results"},
				"embedding_provider": {"type": "string", "default": "openai", "title": "Embedding Provider"},
				"embedding_model": {"type": "string", "default": "text-embedding-3-small", "title": "Embedding Model"}
			}
		}`, `{
			"type": "object",
			"properties": {
				"collection": {"type": "string", "title": "コレクション名", "description": "コレクション名（input.collectionでも指定可能）"},
				"top_k": {"type": "integer", "default": 5, "minimum": 1, "maximum": 100, "title": "結果数", "description": "返す結果の数"},
				"threshold": {"type": "number", "minimum": 0, "maximum": 1, "title": "類似度閾値", "description": "最小類似度スコア"},
				"include_content": {"type": "boolean", "default": true, "title": "コンテンツを含める", "description": "結果にドキュメントコンテンツを含める"},
				"embedding_provider": {"type": "string", "default": "openai", "title": "埋め込みプロバイダー"},
				"embedding_model": {"type": "string", "default": "text-embedding-3-small", "title": "埋め込みモデル"}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("output", "Output", "出力", "Search results", "検索結果", true),
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
		UIConfig: LSchema(`{"icon": "search", "color": "#3B82F6"}`, `{"icon": "search", "color": "#3B82F6"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("VEC_001", "COLLECTION_REQUIRED", "コレクション必須", "Collection name is required", "コレクション名が必要です", false),
			LError("VEC_003", "VECTOR_OR_QUERY_REQUIRED", "ベクトルまたはクエリ必須", "Either vector or query text is required", "ベクトルまたはクエリテキストが必要です", false),
		},
		Enabled: true,
	}
}

func VectorDeleteBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "vector-delete",
		Version:     1,
		Name:        LText("Vector Delete", "ベクトル削除"),
		Description: LText("Delete documents from vector database", "ベクトルデータベースからドキュメントを削除"),
		Category:    domain.BlockCategoryAI,
		Subcategory: domain.BlockSubcategoryRAG,
		Icon:        "trash-2",
		ConfigSchema: LSchema(`{
			"type": "object",
			"required": ["collection"],
			"properties": {
				"collection": {"type": "string", "title": "Collection Name", "description": "Collection name"}
			}
		}`, `{
			"type": "object",
			"required": ["collection"],
			"properties": {
				"collection": {"type": "string", "title": "コレクション名", "description": "コレクション名"}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("output", "Output", "出力", "Delete result", "削除結果", true),
		},
		Code: `
const collection = config.collection || input.collection;
if (!collection) throw new Error('[VEC_001] Collection name is required');
const ids = input.ids || (input.id ? [input.id] : null);
if (!ids || ids.length === 0) throw new Error('[VEC_004] IDs array is required');
const result = ctx.vector.delete(collection, ids);
return {collection, deleted_count: result.deleted_count, requested_ids: ids};
`,
		UIConfig: LSchema(`{"icon": "trash-2", "color": "#EF4444"}`, `{"icon": "trash-2", "color": "#EF4444"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("VEC_001", "COLLECTION_REQUIRED", "コレクション必須", "Collection name is required", "コレクション名が必要です", false),
			LError("VEC_004", "IDS_REQUIRED", "ID必須", "IDs array is required", "ID配列が必要です", false),
		},
		Enabled: true,
	}
}
