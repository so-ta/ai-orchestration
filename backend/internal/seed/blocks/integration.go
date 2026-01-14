package blocks

import (
	"encoding/json"

	"github.com/souta/ai-orchestration/internal/domain"
)

func (r *Registry) registerIntegrationBlocks() {
	r.register(HTTPBlock())
	r.register(SubflowBlock())
	r.register(ToolBlock())
	r.register(SlackBlock())
	r.register(DiscordBlock())
	r.register(NotionQueryDBBlock())
	r.register(NotionCreatePageBlock())
	r.register(GSheetsAppendBlock())
	r.register(GSheetsReadBlock())
	r.register(GitHubCreateIssueBlock())
	r.register(GitHubAddCommentBlock())
	r.register(WebSearchBlock())
	r.register(LinearCreateIssueBlock())
	r.register(EmailSendGridBlock())
	r.register(LoopBlock())
	r.register(EmbeddingBlock())
	r.register(VectorUpsertBlock())
	r.register(VectorSearchBlock())
	r.register(VectorDeleteBlock())
}

func HTTPBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "http",
		Version:     1,
		Name:        "HTTP Request",
		Description: "Make HTTP API calls",
		Category:    domain.BlockCategoryIntegration,
		Icon:        "globe",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"url": {"type": "string"},
				"body": {"type": "object"},
				"method": {"enum": ["GET", "POST", "PUT", "DELETE", "PATCH"], "type": "string"},
				"headers": {"type": "object"}
			}
		}`),
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"body": {"type": "object", "description": "リクエストボディ"},
				"query": {"type": "object", "description": "クエリパラメータ"},
				"headers": {"type": "object", "description": "追加ヘッダー"}
			},
			"description": "URL/ボディのテンプレートで参照可能なデータ"
		}`),
		InputPorts:  []domain.InputPort{},
		OutputPorts: []domain.OutputPort{},
		Code: `
const url = renderTemplate(config.url, input);
const response = await ctx.http.request(url, {
    method: config.method || 'GET',
    headers: config.headers || {},
    body: config.body ? renderTemplate(JSON.stringify(config.body), input) : null
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
		Category:    domain.BlockCategoryIntegration,
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
		Code:     `return await ctx.workflow.run(config.workflow_id, input);`,
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
		Category:    domain.BlockCategoryIntegration,
		Icon:        "wrench",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"adapter_id": {"type": "string"}
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
		Code:     `return await ctx.adapter.call(config.adapter_id, input);`,
		UIConfig: json.RawMessage(`{"icon": "wrench", "color": "#10B981"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "TOOL_001", Name: "ADAPTER_NOT_FOUND", Description: "Adapter not found", Retryable: false},
			{Code: "TOOL_002", Name: "EXEC_ERROR", Description: "Tool execution error", Retryable: true},
		},
		Enabled: true,
	}
}

func SlackBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "slack",
		Version:     1,
		Name:        "Slack",
		Description: "Slackチャンネルにメッセージを送信",
		Category:    domain.BlockCategoryIntegration,
		Icon:        "message-square",
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
		Code: `
const webhookUrl = config.webhook_url || ctx.secrets.SLACK_WEBHOOK_URL;
if (!webhookUrl) {
    throw new Error('[SLACK_001] Webhook URLが設定されていません');
}
const payload = {
    text: renderTemplate(config.message, input)
};
if (config.channel) payload.channel = config.channel;
if (config.username) payload.username = config.username;
if (config.icon_emoji) payload.icon_emoji = config.icon_emoji;
if (config.blocks && config.blocks.length > 0) payload.blocks = config.blocks;
const response = await ctx.http.post(webhookUrl, payload, {
    headers: { 'Content-Type': 'application/json' }
});
if (response.status >= 400) {
    throw new Error('[SLACK_002] Slack送信失敗: ' + response.status);
}
return { success: true, status: response.status };
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

func DiscordBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "discord",
		Version:     1,
		Name:        "Discord",
		Description: "Discord Webhookにメッセージを送信",
		Category:    domain.BlockCategoryIntegration,
		Icon:        "message-circle",
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
		Code: `
const webhookUrl = config.webhook_url || ctx.secrets.DISCORD_WEBHOOK_URL;
if (!webhookUrl) {
    throw new Error('[DISCORD_001] Webhook URLが設定されていません');
}
const payload = {
    content: renderTemplate(config.content, input)
};
if (config.username) payload.username = config.username;
if (config.avatar_url) payload.avatar_url = config.avatar_url;
if (config.embeds && config.embeds.length > 0) payload.embeds = config.embeds;
const response = await ctx.http.post(webhookUrl, payload, {
    headers: { 'Content-Type': 'application/json' }
});
if (response.status === 429) {
    throw new Error('[DISCORD_003] レート制限に達しました');
}
if (response.status >= 400) {
    throw new Error('[DISCORD_002] Discord送信失敗: ' + response.status);
}
return { success: true, status: response.status };
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

func NotionQueryDBBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "notion_query_db",
		Version:     1,
		Name:        "Notion: DB検索",
		Description: "Notionデータベースを検索",
		Category:    domain.BlockCategoryIntegration,
		Icon:        "database",
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
		Code: `
const apiKey = config.api_key || ctx.secrets.NOTION_API_KEY;
if (!apiKey) {
    throw new Error('[NOTION_001] API Keyが設定されていません');
}
const payload = {};
if (config.filter) payload.filter = config.filter;
if (config.sorts) payload.sorts = config.sorts;
if (config.page_size) payload.page_size = config.page_size;
const response = await ctx.http.post(
    'https://api.notion.com/v1/databases/' + config.database_id + '/query',
    payload,
    {
        headers: {
            'Authorization': 'Bearer ' + apiKey,
            'Content-Type': 'application/json',
            'Notion-Version': '2022-06-28'
        }
    }
);
if (response.status >= 400) {
    const errorMsg = response.body?.message || 'Unknown error';
    throw new Error('[NOTION_004] クエリ失敗: ' + errorMsg);
}
return {
    results: response.body.results,
    has_more: response.body.has_more,
    next_cursor: response.body.next_cursor
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

func NotionCreatePageBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "notion_create_page",
		Version:     1,
		Name:        "Notion: ページ作成",
		Description: "Notionにページを作成",
		Category:    domain.BlockCategoryIntegration,
		Icon:        "file-text",
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
		Code: `
const apiKey = config.api_key || ctx.secrets.NOTION_API_KEY;
if (!apiKey) {
    throw new Error('[NOTION_001] API Keyが設定されていません');
}
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
const response = await ctx.http.post('https://api.notion.com/v1/pages', payload, {
    headers: {
        'Authorization': 'Bearer ' + apiKey,
        'Content-Type': 'application/json',
        'Notion-Version': '2022-06-28'
    }
});
if (response.status >= 400) {
    const errorMsg = response.body?.message || 'Unknown error';
    throw new Error('[NOTION_002] ページ作成失敗: ' + errorMsg);
}
return {
    id: response.body.id,
    url: response.body.url,
    created_time: response.body.created_time
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

func GSheetsAppendBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "gsheets_append",
		Version:     1,
		Name:        "Google Sheets: 行追加",
		Description: "Google Sheetsに行を追加",
		Category:    domain.BlockCategoryIntegration,
		Icon:        "table",
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
		Code: `
const apiKey = config.api_key || ctx.secrets.GOOGLE_API_KEY;
if (!apiKey) {
    throw new Error('[GSHEETS_001] API Keyが設定されていません');
}
const range = encodeURIComponent(config.range || 'Sheet1!A:Z');
const valueInputOption = config.value_input_option || 'USER_ENTERED';
const url = 'https://sheets.googleapis.com/v4/spreadsheets/' + config.spreadsheet_id +
    '/values/' + range + ':append' +
    '?valueInputOption=' + valueInputOption +
    '&key=' + apiKey;
let values = config.values;
if (typeof values === 'string') {
    values = JSON.parse(renderTemplate(values, input));
}
const response = await ctx.http.post(url, { values: values }, {
    headers: { 'Content-Type': 'application/json' }
});
if (response.status === 404) {
    throw new Error('[GSHEETS_003] スプレッドシートが見つかりません');
}
if (response.status >= 400) {
    const errorMsg = response.body?.error?.message || 'Unknown error';
    throw new Error('[GSHEETS_002] 行追加失敗: ' + errorMsg);
}
return {
    updated_range: response.body.updates?.updatedRange,
    updated_rows: response.body.updates?.updatedRows,
    updated_cells: response.body.updates?.updatedCells
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

func GSheetsReadBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "gsheets_read",
		Version:     1,
		Name:        "Google Sheets: 読み取り",
		Description: "Google Sheetsから範囲を読み取り",
		Category:    domain.BlockCategoryIntegration,
		Icon:        "table",
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
		Code: `
const apiKey = config.api_key || ctx.secrets.GOOGLE_API_KEY;
if (!apiKey) {
    throw new Error('[GSHEETS_001] API Keyが設定されていません');
}
const range = encodeURIComponent(config.range);
const majorDimension = config.major_dimension || 'ROWS';
const url = 'https://sheets.googleapis.com/v4/spreadsheets/' + config.spreadsheet_id +
    '/values/' + range +
    '?majorDimension=' + majorDimension +
    '&key=' + apiKey;
const response = await ctx.http.get(url);
if (response.status === 404) {
    throw new Error('[GSHEETS_003] スプレッドシートが見つかりません');
}
if (response.status >= 400) {
    const errorMsg = response.body?.error?.message || 'Unknown error';
    throw new Error('[GSHEETS_004] 読み取り失敗: ' + errorMsg);
}
return {
    range: response.body.range,
    values: response.body.values || []
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

func GitHubCreateIssueBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "github_create_issue",
		Version:     1,
		Name:        "GitHub: Issue作成",
		Description: "GitHubリポジトリにIssueを作成",
		Category:    domain.BlockCategoryIntegration,
		Icon:        "git-pull-request",
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
		Code: `
const token = config.token || ctx.secrets.GITHUB_TOKEN;
if (!token) {
    throw new Error('[GITHUB_001] トークンが設定されていません');
}
const payload = {
    title: renderTemplate(config.title, input),
    body: config.body ? renderTemplate(config.body, input) : undefined,
    labels: config.labels,
    assignees: config.assignees
};
const url = 'https://api.github.com/repos/' + config.owner + '/' + config.repo + '/issues';
const response = await ctx.http.post(url, payload, {
    headers: {
        'Authorization': 'Bearer ' + token,
        'Accept': 'application/vnd.github+json',
        'X-GitHub-Api-Version': '2022-11-28'
    }
});
if (response.status === 404) {
    throw new Error('[GITHUB_003] リポジトリが見つかりません');
}
if (response.status >= 400) {
    const errorMsg = response.body?.message || 'Unknown error';
    throw new Error('[GITHUB_002] Issue作成失敗: ' + errorMsg);
}
return {
    id: response.body.id,
    number: response.body.number,
    url: response.body.url,
    html_url: response.body.html_url
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

func GitHubAddCommentBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "github_add_comment",
		Version:     1,
		Name:        "GitHub: コメント追加",
		Description: "GitHub IssueまたはPRにコメントを追加",
		Category:    domain.BlockCategoryIntegration,
		Icon:        "message-square",
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
		Code: `
const token = config.token || ctx.secrets.GITHUB_TOKEN;
if (!token) {
    throw new Error('[GITHUB_001] トークンが設定されていません');
}
const url = 'https://api.github.com/repos/' + config.owner + '/' + config.repo +
    '/issues/' + config.issue_number + '/comments';
const response = await ctx.http.post(url, {
    body: renderTemplate(config.body, input)
}, {
    headers: {
        'Authorization': 'Bearer ' + token,
        'Accept': 'application/vnd.github+json',
        'X-GitHub-Api-Version': '2022-11-28'
    }
});
if (response.status >= 400) {
    const errorMsg = response.body?.message || 'Unknown error';
    throw new Error('[GITHUB_004] コメント追加失敗: ' + errorMsg);
}
return {
    id: response.body.id,
    url: response.body.url,
    html_url: response.body.html_url
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

func WebSearchBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "web_search",
		Version:     1,
		Name:        "Web検索",
		Description: "Tavily APIでWeb検索を実行",
		Category:    domain.BlockCategoryIntegration,
		Icon:        "search",
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
		Code: `
const apiKey = config.api_key || ctx.secrets.TAVILY_API_KEY;
if (!apiKey) {
    throw new Error('[SEARCH_001] API Keyが設定されていません');
}
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
const response = await ctx.http.post('https://api.tavily.com/search', payload, {
    headers: { 'Content-Type': 'application/json' }
});
if (response.status >= 400) {
    throw new Error('[SEARCH_002] 検索失敗: ' + (response.body?.error || response.status));
}
return {
    answer: response.body.answer,
    results: response.body.results
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

func LinearCreateIssueBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "linear_create_issue",
		Version:     1,
		Name:        "Linear: Issue作成",
		Description: "LinearにIssueを作成",
		Category:    domain.BlockCategoryIntegration,
		Icon:        "check-square",
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
		Code: `
const apiKey = config.api_key || ctx.secrets.LINEAR_API_KEY;
if (!apiKey) {
    throw new Error('[LINEAR_001] API Keyが設定されていません');
}
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
const response = await ctx.http.post('https://api.linear.app/graphql', {
    query: mutation,
    variables: variables
}, {
    headers: {
        'Authorization': apiKey,
        'Content-Type': 'application/json'
    }
});
if (response.status >= 400 || response.body.errors) {
    const errorMsg = response.body.errors?.[0]?.message || 'Unknown error';
    throw new Error('[LINEAR_002] Issue作成失敗: ' + errorMsg);
}
const issue = response.body.data.issueCreate.issue;
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

func EmailSendGridBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "email_sendgrid",
		Version:     1,
		Name:        "Email (SendGrid)",
		Description: "SendGrid APIでメールを送信",
		Category:    domain.BlockCategoryIntegration,
		Icon:        "mail",
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
		Code: `
const apiKey = config.api_key || ctx.secrets.SENDGRID_API_KEY;
if (!apiKey) {
    throw new Error('[EMAIL_001] API Keyが設定されていません');
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
const response = await ctx.http.post('https://api.sendgrid.com/v3/mail/send', payload, {
    headers: {
        'Authorization': 'Bearer ' + apiKey,
        'Content-Type': 'application/json'
    }
});
if (response.status >= 400) {
    const errors = response.body?.errors?.map(e => e.message).join(', ') || 'Unknown error';
    throw new Error('[EMAIL_002] メール送信失敗: ' + errors);
}
return {
    success: true,
    message_id: response.headers['x-message-id']
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

func LoopBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "loop",
		Version:     1,
		Name:        "Loop",
		Description: "Iterate with for/forEach/while",
		Category:    domain.BlockCategoryLogic,
		Icon:        "repeat",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"count": {"type": "integer"},
				"condition": {"type": "string"},
				"loop_type": {"enum": ["for", "forEach", "while", "doWhile"], "type": "string"},
				"input_path": {"type": "string"},
				"max_iterations": {"type": "integer"}
			}
		}`),
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"data": {"type": "object", "description": "ループ内で参照可能なデータ"},
				"items": {"type": "array", "description": "forEach時のループ対象配列"}
			},
			"description": "ループ処理に使用するデータ"
		}`),
		InputPorts: []domain.InputPort{
			{Name: "input", Label: "Input", Schema: json.RawMessage(`{"type": "any"}`), Required: true, Description: "Initial value or array to iterate"},
		},
		OutputPorts: []domain.OutputPort{
			{Name: "loop", Label: "Loop Body", IsDefault: true, Description: "Each iteration"},
			{Name: "complete", Label: "Complete", IsDefault: false, Description: "When loop finishes"},
		},
		Code: `
const results = [];
const maxIter = config.max_iterations || 100;
if (config.loop_type === 'for') {
    for (let i = 0; i < (config.count || 0) && i < maxIter; i++) {
        results.push({ index: i, ...input });
    }
} else if (config.loop_type === 'forEach') {
    const items = getPath(input, config.input_path) || [];
    for (let i = 0; i < items.length && i < maxIter; i++) {
        results.push({ item: items[i], index: i, ...input });
    }
} else if (config.loop_type === 'while') {
    let i = 0;
    while (evaluate(config.condition, input) && i < maxIter) {
        results.push({ index: i, ...input });
        i++;
    }
}
return { ...input, results, iterations: results.length };
`,
		UIConfig: json.RawMessage(`{"icon": "repeat", "color": "#F59E0B"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "LOOP_001", Name: "MAX_ITERATIONS", Description: "Maximum iterations exceeded", Retryable: false},
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
const result = await ctx.embedding.embed(provider, model, texts);
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
		Category:    domain.BlockCategoryData,
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
const result = await ctx.vector.upsert(collection, documents, {embedding_provider: config.embedding_provider, embedding_model: config.embedding_model});
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
		Category:    domain.BlockCategoryData,
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
  const embedResult = await ctx.embedding.embed(provider, model, [input.query]);
  searchVector = embedResult.vectors[0];
}
if (!searchVector) throw new Error('[VEC_003] Either vector or query text is required');
const result = await ctx.vector.query(collection, searchVector, {top_k: config.top_k || 5, threshold: config.threshold, include_content: config.include_content !== false});
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
		Category:    domain.BlockCategoryData,
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
const result = await ctx.vector.delete(collection, ids);
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
