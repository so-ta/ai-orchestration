-- 013_add_integration_blocks.sql
-- 外部サービス連携ブロックの追加
-- Created: 2025-01-12

-- ============================================================================
-- Slack: メッセージ送信
-- ============================================================================
INSERT INTO block_definitions (
    id, tenant_id, slug, name, description, category, icon,
    executor_type, config_schema, input_schema, output_schema,
    error_codes, code, ui_config, is_system, enabled
) VALUES (
    gen_random_uuid(),
    NULL,
    'slack',
    'Slack',
    'Slackチャンネルにメッセージを送信',
    'integration',
    'message-square',
    'builtin',
    '{
        "type": "object",
        "properties": {
            "webhook_url": {
                "type": "string",
                "title": "Webhook URL",
                "description": "Slack Incoming Webhook URL（空の場合はシークレットSLACK_WEBHOOK_URLを使用）"
            },
            "channel": {
                "type": "string",
                "title": "チャンネル",
                "description": "投稿先チャンネル（#general など）"
            },
            "message": {
                "type": "string",
                "title": "メッセージ",
                "description": "送信するメッセージ（テンプレート使用可: ${input.field}）",
                "x-ui-widget": "textarea"
            },
            "username": {
                "type": "string",
                "title": "表示名",
                "description": "ボットの表示名"
            },
            "icon_emoji": {
                "type": "string",
                "title": "アイコン絵文字",
                "description": "ボットのアイコン（:robot_face: など）"
            },
            "blocks": {
                "type": "array",
                "title": "Block Kit",
                "description": "Slack Block Kit形式のメッセージ（JSON）",
                "x-ui-widget": "json"
            }
        },
        "required": ["message"]
    }',
    '{"type": "object"}',
    '{
        "type": "object",
        "properties": {
            "success": {"type": "boolean"},
            "status": {"type": "number"}
        }
    }',
    '[
        {"code": "SLACK_001", "name": "WEBHOOK_NOT_CONFIGURED", "description": "Webhook URLが設定されていません", "retryable": false},
        {"code": "SLACK_002", "name": "SEND_FAILED", "description": "メッセージ送信に失敗しました", "retryable": true},
        {"code": "SLACK_003", "name": "INVALID_WEBHOOK", "description": "無効なWebhook URL", "retryable": false}
    ]',
    $code$
const webhookUrl = config.webhook_url || ctx.secrets.SLACK_WEBHOOK_URL;
if (!webhookUrl) {
    throw new Error('[SLACK_001] Webhook URLが設定されていません');
}

const payload = {
    text: renderTemplate(config.message, input)
};

if (config.channel) {
    payload.channel = config.channel;
}
if (config.username) {
    payload.username = config.username;
}
if (config.icon_emoji) {
    payload.icon_emoji = config.icon_emoji;
}
if (config.blocks && config.blocks.length > 0) {
    payload.blocks = config.blocks;
}

const response = await ctx.http.post(webhookUrl, payload, {
    headers: { 'Content-Type': 'application/json' }
});

if (response.status >= 400) {
    throw new Error('[SLACK_002] Slack送信失敗: ' + response.status);
}

return { success: true, status: response.status };
    $code$,
    '{"icon": "message-square", "color": "#4A154B"}',
    TRUE,
    TRUE
)
ON CONFLICT (tenant_id, slug) WHERE tenant_id IS NULL
DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    config_schema = EXCLUDED.config_schema,
    input_schema = EXCLUDED.input_schema,
    output_schema = EXCLUDED.output_schema,
    error_codes = EXCLUDED.error_codes,
    code = EXCLUDED.code,
    ui_config = EXCLUDED.ui_config,
    updated_at = NOW();

-- ============================================================================
-- Discord: Webhook通知
-- ============================================================================
INSERT INTO block_definitions (
    id, tenant_id, slug, name, description, category, icon,
    executor_type, config_schema, input_schema, output_schema,
    error_codes, code, ui_config, is_system, enabled
) VALUES (
    gen_random_uuid(),
    NULL,
    'discord',
    'Discord',
    'Discord Webhookにメッセージを送信',
    'integration',
    'message-circle',
    'builtin',
    '{
        "type": "object",
        "properties": {
            "webhook_url": {
                "type": "string",
                "title": "Webhook URL",
                "description": "Discord Webhook URL（空の場合はシークレットDISCORD_WEBHOOK_URLを使用）"
            },
            "content": {
                "type": "string",
                "title": "メッセージ",
                "description": "送信するメッセージ（テンプレート使用可）",
                "x-ui-widget": "textarea"
            },
            "username": {
                "type": "string",
                "title": "ユーザー名",
                "description": "Webhookの表示名を上書き"
            },
            "avatar_url": {
                "type": "string",
                "title": "アバターURL",
                "description": "Webhookのアバター画像URL"
            },
            "embeds": {
                "type": "array",
                "title": "Embeds",
                "description": "リッチな埋め込みメッセージ（JSON配列）",
                "x-ui-widget": "json"
            }
        },
        "required": ["content"]
    }',
    '{"type": "object"}',
    '{
        "type": "object",
        "properties": {
            "success": {"type": "boolean"},
            "status": {"type": "number"}
        }
    }',
    '[
        {"code": "DISCORD_001", "name": "WEBHOOK_NOT_CONFIGURED", "description": "Webhook URLが設定されていません", "retryable": false},
        {"code": "DISCORD_002", "name": "SEND_FAILED", "description": "メッセージ送信に失敗しました", "retryable": true},
        {"code": "DISCORD_003", "name": "RATE_LIMITED", "description": "レート制限に達しました", "retryable": true}
    ]',
    $code$
const webhookUrl = config.webhook_url || ctx.secrets.DISCORD_WEBHOOK_URL;
if (!webhookUrl) {
    throw new Error('[DISCORD_001] Webhook URLが設定されていません');
}

const payload = {
    content: renderTemplate(config.content, input)
};

if (config.username) {
    payload.username = config.username;
}
if (config.avatar_url) {
    payload.avatar_url = config.avatar_url;
}
if (config.embeds && config.embeds.length > 0) {
    payload.embeds = config.embeds;
}

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
    $code$,
    '{"icon": "message-circle", "color": "#5865F2"}',
    TRUE,
    TRUE
)
ON CONFLICT (tenant_id, slug) WHERE tenant_id IS NULL
DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    config_schema = EXCLUDED.config_schema,
    input_schema = EXCLUDED.input_schema,
    output_schema = EXCLUDED.output_schema,
    error_codes = EXCLUDED.error_codes,
    code = EXCLUDED.code,
    ui_config = EXCLUDED.ui_config,
    updated_at = NOW();

-- ============================================================================
-- Notion: ページ作成
-- ============================================================================
INSERT INTO block_definitions (
    id, tenant_id, slug, name, description, category, icon,
    executor_type, config_schema, input_schema, output_schema,
    error_codes, code, ui_config, is_system, enabled
) VALUES (
    gen_random_uuid(),
    NULL,
    'notion_create_page',
    'Notion: ページ作成',
    'Notionにページを作成',
    'integration',
    'file-text',
    'builtin',
    '{
        "type": "object",
        "properties": {
            "api_key": {
                "type": "string",
                "title": "API Key",
                "description": "Notion API Key（空の場合はシークレットNOTION_API_KEYを使用）"
            },
            "parent_type": {
                "type": "string",
                "title": "親タイプ",
                "enum": ["page_id", "database_id"],
                "default": "database_id"
            },
            "parent_id": {
                "type": "string",
                "title": "親ID",
                "description": "親ページまたはデータベースのID"
            },
            "title": {
                "type": "string",
                "title": "タイトル",
                "description": "ページタイトル（テンプレート使用可）"
            },
            "properties": {
                "type": "object",
                "title": "プロパティ",
                "description": "DBプロパティ（JSON形式）",
                "x-ui-widget": "json"
            },
            "content": {
                "type": "string",
                "title": "本文",
                "description": "ページの本文コンテンツ（テンプレート使用可）",
                "x-ui-widget": "textarea"
            }
        },
        "required": ["parent_id"]
    }',
    '{"type": "object"}',
    '{
        "type": "object",
        "properties": {
            "id": {"type": "string"},
            "url": {"type": "string"},
            "created_time": {"type": "string"}
        }
    }',
    '[
        {"code": "NOTION_001", "name": "API_KEY_NOT_CONFIGURED", "description": "API Keyが設定されていません", "retryable": false},
        {"code": "NOTION_002", "name": "CREATE_FAILED", "description": "ページ作成に失敗しました", "retryable": true},
        {"code": "NOTION_003", "name": "INVALID_PARENT", "description": "無効な親IDです", "retryable": false}
    ]',
    $code$
const apiKey = config.api_key || ctx.secrets.NOTION_API_KEY;
if (!apiKey) {
    throw new Error('[NOTION_001] API Keyが設定されていません');
}

const parentKey = config.parent_type || 'database_id';
const payload = {
    parent: { [parentKey]: config.parent_id }
};

// プロパティ設定
if (config.properties) {
    payload.properties = config.properties;
} else if (config.title) {
    // シンプルなタイトルのみの場合
    payload.properties = {
        title: {
            title: [{ text: { content: renderTemplate(config.title, input) } }]
        }
    };
}

// 本文コンテンツ（シンプルなparagraph）
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
    $code$,
    '{"icon": "file-text", "color": "#000000"}',
    TRUE,
    TRUE
)
ON CONFLICT (tenant_id, slug) WHERE tenant_id IS NULL
DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    config_schema = EXCLUDED.config_schema,
    input_schema = EXCLUDED.input_schema,
    output_schema = EXCLUDED.output_schema,
    error_codes = EXCLUDED.error_codes,
    code = EXCLUDED.code,
    ui_config = EXCLUDED.ui_config,
    updated_at = NOW();

-- ============================================================================
-- Notion: データベースクエリ
-- ============================================================================
INSERT INTO block_definitions (
    id, tenant_id, slug, name, description, category, icon,
    executor_type, config_schema, input_schema, output_schema,
    error_codes, code, ui_config, is_system, enabled
) VALUES (
    gen_random_uuid(),
    NULL,
    'notion_query_db',
    'Notion: DB検索',
    'Notionデータベースを検索',
    'integration',
    'database',
    'builtin',
    '{
        "type": "object",
        "properties": {
            "api_key": {
                "type": "string",
                "title": "API Key",
                "description": "Notion API Key（空の場合はシークレットNOTION_API_KEYを使用）"
            },
            "database_id": {
                "type": "string",
                "title": "データベースID",
                "description": "検索対象のデータベースID"
            },
            "filter": {
                "type": "object",
                "title": "フィルター",
                "description": "Notion Filter形式（JSON）",
                "x-ui-widget": "json"
            },
            "sorts": {
                "type": "array",
                "title": "ソート",
                "description": "ソート条件（JSON配列）",
                "x-ui-widget": "json"
            },
            "page_size": {
                "type": "number",
                "title": "取得件数",
                "default": 100,
                "minimum": 1,
                "maximum": 100
            }
        },
        "required": ["database_id"]
    }',
    '{"type": "object"}',
    '{
        "type": "object",
        "properties": {
            "results": {"type": "array"},
            "has_more": {"type": "boolean"},
            "next_cursor": {"type": "string"}
        }
    }',
    '[
        {"code": "NOTION_001", "name": "API_KEY_NOT_CONFIGURED", "description": "API Keyが設定されていません", "retryable": false},
        {"code": "NOTION_004", "name": "QUERY_FAILED", "description": "クエリに失敗しました", "retryable": true}
    ]',
    $code$
const apiKey = config.api_key || ctx.secrets.NOTION_API_KEY;
if (!apiKey) {
    throw new Error('[NOTION_001] API Keyが設定されていません');
}

const payload = {};
if (config.filter) {
    payload.filter = config.filter;
}
if (config.sorts) {
    payload.sorts = config.sorts;
}
if (config.page_size) {
    payload.page_size = config.page_size;
}

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
    $code$,
    '{"icon": "database", "color": "#000000"}',
    TRUE,
    TRUE
)
ON CONFLICT (tenant_id, slug) WHERE tenant_id IS NULL
DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    config_schema = EXCLUDED.config_schema,
    input_schema = EXCLUDED.input_schema,
    output_schema = EXCLUDED.output_schema,
    error_codes = EXCLUDED.error_codes,
    code = EXCLUDED.code,
    ui_config = EXCLUDED.ui_config,
    updated_at = NOW();

-- ============================================================================
-- Google Sheets: 行追加
-- ============================================================================
INSERT INTO block_definitions (
    id, tenant_id, slug, name, description, category, icon,
    executor_type, config_schema, input_schema, output_schema,
    error_codes, code, ui_config, is_system, enabled
) VALUES (
    gen_random_uuid(),
    NULL,
    'gsheets_append',
    'Google Sheets: 行追加',
    'Google Sheetsに行を追加',
    'integration',
    'table',
    'builtin',
    '{
        "type": "object",
        "properties": {
            "api_key": {
                "type": "string",
                "title": "API Key",
                "description": "Google API Key（空の場合はシークレットGOOGLE_API_KEYを使用）"
            },
            "spreadsheet_id": {
                "type": "string",
                "title": "スプレッドシートID",
                "description": "URLから取得: /d/{spreadsheet_id}/edit"
            },
            "range": {
                "type": "string",
                "title": "範囲",
                "description": "シート名と範囲（例: Sheet1!A:Z）",
                "default": "Sheet1!A:Z"
            },
            "values": {
                "type": "array",
                "title": "値",
                "description": "追加する行データ（2次元配列またはテンプレート）",
                "x-ui-widget": "json"
            },
            "value_input_option": {
                "type": "string",
                "title": "入力形式",
                "enum": ["RAW", "USER_ENTERED"],
                "default": "USER_ENTERED"
            }
        },
        "required": ["spreadsheet_id", "values"]
    }',
    '{"type": "object"}',
    '{
        "type": "object",
        "properties": {
            "updated_range": {"type": "string"},
            "updated_rows": {"type": "number"},
            "updated_cells": {"type": "number"}
        }
    }',
    '[
        {"code": "GSHEETS_001", "name": "API_KEY_NOT_CONFIGURED", "description": "API Keyが設定されていません", "retryable": false},
        {"code": "GSHEETS_002", "name": "APPEND_FAILED", "description": "行追加に失敗しました", "retryable": true},
        {"code": "GSHEETS_003", "name": "INVALID_SPREADSHEET", "description": "スプレッドシートが見つかりません", "retryable": false}
    ]',
    $code$
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

// 値がテンプレート文字列の場合は展開
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
    $code$,
    '{"icon": "table", "color": "#0F9D58"}',
    TRUE,
    TRUE
)
ON CONFLICT (tenant_id, slug) WHERE tenant_id IS NULL
DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    config_schema = EXCLUDED.config_schema,
    input_schema = EXCLUDED.input_schema,
    output_schema = EXCLUDED.output_schema,
    error_codes = EXCLUDED.error_codes,
    code = EXCLUDED.code,
    ui_config = EXCLUDED.ui_config,
    updated_at = NOW();

-- ============================================================================
-- Google Sheets: 範囲読み取り
-- ============================================================================
INSERT INTO block_definitions (
    id, tenant_id, slug, name, description, category, icon,
    executor_type, config_schema, input_schema, output_schema,
    error_codes, code, ui_config, is_system, enabled
) VALUES (
    gen_random_uuid(),
    NULL,
    'gsheets_read',
    'Google Sheets: 読み取り',
    'Google Sheetsから範囲を読み取り',
    'integration',
    'table',
    'builtin',
    '{
        "type": "object",
        "properties": {
            "api_key": {
                "type": "string",
                "title": "API Key",
                "description": "Google API Key（空の場合はシークレットGOOGLE_API_KEYを使用）"
            },
            "spreadsheet_id": {
                "type": "string",
                "title": "スプレッドシートID"
            },
            "range": {
                "type": "string",
                "title": "範囲",
                "description": "読み取り範囲（例: Sheet1!A1:D10）"
            },
            "major_dimension": {
                "type": "string",
                "title": "次元",
                "enum": ["ROWS", "COLUMNS"],
                "default": "ROWS"
            }
        },
        "required": ["spreadsheet_id", "range"]
    }',
    '{"type": "object"}',
    '{
        "type": "object",
        "properties": {
            "range": {"type": "string"},
            "values": {"type": "array"}
        }
    }',
    '[
        {"code": "GSHEETS_001", "name": "API_KEY_NOT_CONFIGURED", "description": "API Keyが設定されていません", "retryable": false},
        {"code": "GSHEETS_003", "name": "INVALID_SPREADSHEET", "description": "スプレッドシートが見つかりません", "retryable": false},
        {"code": "GSHEETS_004", "name": "READ_FAILED", "description": "読み取りに失敗しました", "retryable": true}
    ]',
    $code$
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
    $code$,
    '{"icon": "table", "color": "#0F9D58"}',
    TRUE,
    TRUE
)
ON CONFLICT (tenant_id, slug) WHERE tenant_id IS NULL
DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    config_schema = EXCLUDED.config_schema,
    input_schema = EXCLUDED.input_schema,
    output_schema = EXCLUDED.output_schema,
    error_codes = EXCLUDED.error_codes,
    code = EXCLUDED.code,
    ui_config = EXCLUDED.ui_config,
    updated_at = NOW();

-- ============================================================================
-- GitHub: Issue作成
-- ============================================================================
INSERT INTO block_definitions (
    id, tenant_id, slug, name, description, category, icon,
    executor_type, config_schema, input_schema, output_schema,
    error_codes, code, ui_config, is_system, enabled
) VALUES (
    gen_random_uuid(),
    NULL,
    'github_create_issue',
    'GitHub: Issue作成',
    'GitHubリポジトリにIssueを作成',
    'integration',
    'git-pull-request',
    'builtin',
    '{
        "type": "object",
        "properties": {
            "token": {
                "type": "string",
                "title": "アクセストークン",
                "description": "GitHub Personal Access Token（空の場合はシークレットGITHUB_TOKENを使用）"
            },
            "owner": {
                "type": "string",
                "title": "オーナー",
                "description": "リポジトリオーナー（ユーザー名または組織名）"
            },
            "repo": {
                "type": "string",
                "title": "リポジトリ",
                "description": "リポジトリ名"
            },
            "title": {
                "type": "string",
                "title": "タイトル",
                "description": "Issueタイトル（テンプレート使用可）"
            },
            "body": {
                "type": "string",
                "title": "本文",
                "description": "Issue本文（Markdown、テンプレート使用可）",
                "x-ui-widget": "textarea"
            },
            "labels": {
                "type": "array",
                "title": "ラベル",
                "description": "ラベル名の配列",
                "items": {"type": "string"}
            },
            "assignees": {
                "type": "array",
                "title": "アサイン",
                "description": "アサインするユーザー名の配列",
                "items": {"type": "string"}
            }
        },
        "required": ["owner", "repo", "title"]
    }',
    '{"type": "object"}',
    '{
        "type": "object",
        "properties": {
            "id": {"type": "number"},
            "number": {"type": "number"},
            "url": {"type": "string"},
            "html_url": {"type": "string"}
        }
    }',
    '[
        {"code": "GITHUB_001", "name": "TOKEN_NOT_CONFIGURED", "description": "トークンが設定されていません", "retryable": false},
        {"code": "GITHUB_002", "name": "CREATE_FAILED", "description": "Issue作成に失敗しました", "retryable": true},
        {"code": "GITHUB_003", "name": "REPO_NOT_FOUND", "description": "リポジトリが見つかりません", "retryable": false}
    ]',
    $code$
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
    $code$,
    '{"icon": "git-pull-request", "color": "#24292F"}',
    TRUE,
    TRUE
)
ON CONFLICT (tenant_id, slug) WHERE tenant_id IS NULL
DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    config_schema = EXCLUDED.config_schema,
    input_schema = EXCLUDED.input_schema,
    output_schema = EXCLUDED.output_schema,
    error_codes = EXCLUDED.error_codes,
    code = EXCLUDED.code,
    ui_config = EXCLUDED.ui_config,
    updated_at = NOW();

-- ============================================================================
-- GitHub: コメント追加
-- ============================================================================
INSERT INTO block_definitions (
    id, tenant_id, slug, name, description, category, icon,
    executor_type, config_schema, input_schema, output_schema,
    error_codes, code, ui_config, is_system, enabled
) VALUES (
    gen_random_uuid(),
    NULL,
    'github_add_comment',
    'GitHub: コメント追加',
    'GitHub IssueまたはPRにコメントを追加',
    'integration',
    'message-square',
    'builtin',
    '{
        "type": "object",
        "properties": {
            "token": {
                "type": "string",
                "title": "アクセストークン",
                "description": "GitHub Personal Access Token（空の場合はシークレットGITHUB_TOKENを使用）"
            },
            "owner": {
                "type": "string",
                "title": "オーナー"
            },
            "repo": {
                "type": "string",
                "title": "リポジトリ"
            },
            "issue_number": {
                "type": "number",
                "title": "Issue/PR番号"
            },
            "body": {
                "type": "string",
                "title": "コメント本文",
                "description": "コメント本文（Markdown、テンプレート使用可）",
                "x-ui-widget": "textarea"
            }
        },
        "required": ["owner", "repo", "issue_number", "body"]
    }',
    '{"type": "object"}',
    '{
        "type": "object",
        "properties": {
            "id": {"type": "number"},
            "url": {"type": "string"},
            "html_url": {"type": "string"}
        }
    }',
    '[
        {"code": "GITHUB_001", "name": "TOKEN_NOT_CONFIGURED", "description": "トークンが設定されていません", "retryable": false},
        {"code": "GITHUB_004", "name": "COMMENT_FAILED", "description": "コメント追加に失敗しました", "retryable": true}
    ]',
    $code$
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
    $code$,
    '{"icon": "message-square", "color": "#24292F"}',
    TRUE,
    TRUE
)
ON CONFLICT (tenant_id, slug) WHERE tenant_id IS NULL
DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    config_schema = EXCLUDED.config_schema,
    input_schema = EXCLUDED.input_schema,
    output_schema = EXCLUDED.output_schema,
    error_codes = EXCLUDED.error_codes,
    code = EXCLUDED.code,
    ui_config = EXCLUDED.ui_config,
    updated_at = NOW();

-- ============================================================================
-- Web Search: Tavily
-- ============================================================================
INSERT INTO block_definitions (
    id, tenant_id, slug, name, description, category, icon,
    executor_type, config_schema, input_schema, output_schema,
    error_codes, code, ui_config, is_system, enabled
) VALUES (
    gen_random_uuid(),
    NULL,
    'web_search',
    'Web検索',
    'Tavily APIでWeb検索を実行',
    'integration',
    'search',
    'builtin',
    '{
        "type": "object",
        "properties": {
            "api_key": {
                "type": "string",
                "title": "API Key",
                "description": "Tavily API Key（空の場合はシークレットTAVILY_API_KEYを使用）"
            },
            "query": {
                "type": "string",
                "title": "検索クエリ",
                "description": "検索キーワード（テンプレート使用可）"
            },
            "search_depth": {
                "type": "string",
                "title": "検索深度",
                "enum": ["basic", "advanced"],
                "default": "basic"
            },
            "max_results": {
                "type": "number",
                "title": "最大結果数",
                "default": 5,
                "minimum": 1,
                "maximum": 20
            },
            "include_answer": {
                "type": "boolean",
                "title": "AI回答を含める",
                "default": true
            },
            "include_domains": {
                "type": "array",
                "title": "含めるドメイン",
                "items": {"type": "string"}
            },
            "exclude_domains": {
                "type": "array",
                "title": "除外ドメイン",
                "items": {"type": "string"}
            }
        },
        "required": ["query"]
    }',
    '{"type": "object"}',
    '{
        "type": "object",
        "properties": {
            "answer": {"type": "string"},
            "results": {
                "type": "array",
                "items": {
                    "type": "object",
                    "properties": {
                        "title": {"type": "string"},
                        "url": {"type": "string"},
                        "content": {"type": "string"},
                        "score": {"type": "number"}
                    }
                }
            }
        }
    }',
    '[
        {"code": "SEARCH_001", "name": "API_KEY_NOT_CONFIGURED", "description": "API Keyが設定されていません", "retryable": false},
        {"code": "SEARCH_002", "name": "SEARCH_FAILED", "description": "検索に失敗しました", "retryable": true}
    ]',
    $code$
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
    $code$,
    '{"icon": "search", "color": "#4285F4"}',
    TRUE,
    TRUE
)
ON CONFLICT (tenant_id, slug) WHERE tenant_id IS NULL
DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    config_schema = EXCLUDED.config_schema,
    input_schema = EXCLUDED.input_schema,
    output_schema = EXCLUDED.output_schema,
    error_codes = EXCLUDED.error_codes,
    code = EXCLUDED.code,
    ui_config = EXCLUDED.ui_config,
    updated_at = NOW();

-- ============================================================================
-- Email: SendGrid
-- ============================================================================
INSERT INTO block_definitions (
    id, tenant_id, slug, name, description, category, icon,
    executor_type, config_schema, input_schema, output_schema,
    error_codes, code, ui_config, is_system, enabled
) VALUES (
    gen_random_uuid(),
    NULL,
    'email_sendgrid',
    'Email (SendGrid)',
    'SendGrid APIでメールを送信',
    'integration',
    'mail',
    'builtin',
    '{
        "type": "object",
        "properties": {
            "api_key": {
                "type": "string",
                "title": "API Key",
                "description": "SendGrid API Key（空の場合はシークレットSENDGRID_API_KEYを使用）"
            },
            "from_email": {
                "type": "string",
                "title": "送信元メール",
                "description": "送信者のメールアドレス"
            },
            "from_name": {
                "type": "string",
                "title": "送信者名"
            },
            "to_email": {
                "type": "string",
                "title": "宛先メール",
                "description": "受信者のメールアドレス（テンプレート使用可）"
            },
            "to_name": {
                "type": "string",
                "title": "受信者名"
            },
            "subject": {
                "type": "string",
                "title": "件名",
                "description": "メール件名（テンプレート使用可）"
            },
            "content_type": {
                "type": "string",
                "title": "本文形式",
                "enum": ["text/plain", "text/html"],
                "default": "text/plain"
            },
            "content": {
                "type": "string",
                "title": "本文",
                "description": "メール本文（テンプレート使用可）",
                "x-ui-widget": "textarea"
            }
        },
        "required": ["from_email", "to_email", "subject", "content"]
    }',
    '{"type": "object"}',
    '{
        "type": "object",
        "properties": {
            "success": {"type": "boolean"},
            "message_id": {"type": "string"}
        }
    }',
    '[
        {"code": "EMAIL_001", "name": "API_KEY_NOT_CONFIGURED", "description": "API Keyが設定されていません", "retryable": false},
        {"code": "EMAIL_002", "name": "SEND_FAILED", "description": "メール送信に失敗しました", "retryable": true},
        {"code": "EMAIL_003", "name": "INVALID_EMAIL", "description": "メールアドレスが無効です", "retryable": false}
    ]',
    $code$
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
    $code$,
    '{"icon": "mail", "color": "#1A82E2"}',
    TRUE,
    TRUE
)
ON CONFLICT (tenant_id, slug) WHERE tenant_id IS NULL
DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    config_schema = EXCLUDED.config_schema,
    input_schema = EXCLUDED.input_schema,
    output_schema = EXCLUDED.output_schema,
    error_codes = EXCLUDED.error_codes,
    code = EXCLUDED.code,
    ui_config = EXCLUDED.ui_config,
    updated_at = NOW();

-- ============================================================================
-- Linear: Issue作成
-- ============================================================================
INSERT INTO block_definitions (
    id, tenant_id, slug, name, description, category, icon,
    executor_type, config_schema, input_schema, output_schema,
    error_codes, code, ui_config, is_system, enabled
) VALUES (
    gen_random_uuid(),
    NULL,
    'linear_create_issue',
    'Linear: Issue作成',
    'LinearにIssueを作成',
    'integration',
    'check-square',
    'builtin',
    '{
        "type": "object",
        "properties": {
            "api_key": {
                "type": "string",
                "title": "API Key",
                "description": "Linear API Key（空の場合はシークレットLINEAR_API_KEYを使用）"
            },
            "team_id": {
                "type": "string",
                "title": "チームID",
                "description": "LinearチームのID"
            },
            "title": {
                "type": "string",
                "title": "タイトル",
                "description": "Issueタイトル（テンプレート使用可）"
            },
            "description": {
                "type": "string",
                "title": "説明",
                "description": "Issue説明（Markdown、テンプレート使用可）",
                "x-ui-widget": "textarea"
            },
            "priority": {
                "type": "number",
                "title": "優先度",
                "description": "0=なし, 1=緊急, 2=高, 3=中, 4=低",
                "enum": [0, 1, 2, 3, 4],
                "default": 0
            },
            "label_ids": {
                "type": "array",
                "title": "ラベルID",
                "items": {"type": "string"}
            },
            "assignee_id": {
                "type": "string",
                "title": "担当者ID"
            }
        },
        "required": ["team_id", "title"]
    }',
    '{"type": "object"}',
    '{
        "type": "object",
        "properties": {
            "id": {"type": "string"},
            "identifier": {"type": "string"},
            "url": {"type": "string"}
        }
    }',
    '[
        {"code": "LINEAR_001", "name": "API_KEY_NOT_CONFIGURED", "description": "API Keyが設定されていません", "retryable": false},
        {"code": "LINEAR_002", "name": "CREATE_FAILED", "description": "Issue作成に失敗しました", "retryable": true}
    ]',
    $code$
const apiKey = config.api_key || ctx.secrets.LINEAR_API_KEY;
if (!apiKey) {
    throw new Error('[LINEAR_001] API Keyが設定されていません');
}

const mutation = `
mutation IssueCreate($input: IssueCreateInput!) {
    issueCreate(input: $input) {
        success
        issue {
            id
            identifier
            url
        }
    }
}`;

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
    $code$,
    '{"icon": "check-square", "color": "#5E6AD2"}',
    TRUE,
    TRUE
)
ON CONFLICT (tenant_id, slug) WHERE tenant_id IS NULL
DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    config_schema = EXCLUDED.config_schema,
    input_schema = EXCLUDED.input_schema,
    output_schema = EXCLUDED.output_schema,
    error_codes = EXCLUDED.error_codes,
    code = EXCLUDED.code,
    ui_config = EXCLUDED.ui_config,
    updated_at = NOW();
