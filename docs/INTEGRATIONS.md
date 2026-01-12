# External Service Integrations

> **Status**: ✅ Phase 1 実装済み
> **Updated**: 2025-01-12
> **Migration**: `013_add_integration_blocks.sql`

---

## Overview

AI Orchestrationプラットフォームでサポートする外部サービス連携の一覧です。
すべての連携ブロックはUnified Block Modelに基づき、JavaScriptコードとして実装されています。

---

## 実装済み連携サービス

### 通知・コミュニケーション

| サービス | ブロック | エンドポイント | 必要シークレット | Status |
|---------|---------|---------------|-----------------|--------|
| **Slack** | `slack` | メッセージ送信 | `SLACK_WEBHOOK_URL` | ✅ |
| **Discord** | `discord` | Webhook通知 | `DISCORD_WEBHOOK_URL` | ✅ |
| **Email (SendGrid)** | `email_sendgrid` | メール送信 | `SENDGRID_API_KEY` | ✅ |

### ドキュメント・データ管理

| サービス | ブロック | エンドポイント | 必要シークレット | Status |
|---------|---------|---------------|-----------------|--------|
| **Notion** | `notion_create_page` | ページ作成 | `NOTION_API_KEY` | ✅ |
| **Notion** | `notion_query_db` | データベース検索 | `NOTION_API_KEY` | ✅ |
| **Google Sheets** | `gsheets_append` | 行追加 | `GOOGLE_API_KEY` | ✅ |
| **Google Sheets** | `gsheets_read` | 範囲読み取り | `GOOGLE_API_KEY` | ✅ |

### 開発ツール

| サービス | ブロック | エンドポイント | 必要シークレット | Status |
|---------|---------|---------------|-----------------|--------|
| **GitHub** | `github_create_issue` | Issue作成 | `GITHUB_TOKEN` | ✅ |
| **GitHub** | `github_add_comment` | コメント追加 | `GITHUB_TOKEN` | ✅ |
| **Linear** | `linear_create_issue` | Issue作成 | `LINEAR_API_KEY` | ✅ |

### 検索・情報取得

| サービス | ブロック | エンドポイント | 必要シークレット | Status |
|---------|---------|---------------|-----------------|--------|
| **Tavily** | `web_search` | Web検索 | `TAVILY_API_KEY` | ✅ |

---

## 未実装（今後追加予定）

### Priority 2: 高価値

| サービス | 予定ブロック | エンドポイント | ユースケース |
|---------|-------------|---------------|-------------|
| **Jira** | `jira_create_issue` | チケット作成 | エンタープライズ開発管理 |
| **Jira** | `jira_transition` | ステータス変更 | ワークフロー連携 |
| **Asana** | `asana_create_task` | タスク作成 | プロジェクト管理 |
| **AWS S3** | `s3_put_object` | オブジェクトPUT | ファイル保存 |
| **AWS S3** | `s3_get_object` | オブジェクトGET | ファイル取得 |
| **Google Drive** | `gdrive_upload` | ファイルアップロード | ドキュメント管理 |
| **Google Calendar** | `gcal_create_event` | イベント作成 | スケジュール管理 |
| **HubSpot** | `hubspot_create_contact` | コンタクト作成 | CRM連携 |
| **Microsoft Teams** | `teams_send_message` | メッセージ送信 | エンタープライズ通知 |

### Priority 3: 拡張

| サービス | 予定ブロック | エンドポイント | ユースケース |
|---------|-------------|---------------|-------------|
| **Replicate** | `replicate_run` | モデル実行 | 画像生成・AI処理 |
| **ElevenLabs** | `elevenlabs_tts` | 音声合成 | 音声コンテンツ |
| **Whisper** | `whisper_transcribe` | 文字起こし | 会議録作成 |
| **X (Twitter)** | `twitter_post` | ツイート投稿 | ソーシャル連携 |
| **Stripe** | `stripe_create_invoice` | 請求書作成 | 課金自動化 |
| **PagerDuty** | `pagerduty_trigger` | インシデント作成 | 障害対応 |
| **Airtable** | `airtable_create_record` | レコード作成 | ノーコードDB |
| **Supabase** | `supabase_insert` | データ挿入 | BaaS連携 |
| **Typeform** | `typeform_get_responses` | 回答取得 | アンケート |
| **Calendly** | `calendly_get_events` | 予約取得 | スケジュール |

---

## トリガー（Webhookエントリーポイント）

### 汎用Webhookトリガー

すべての外部サービスからのWebhookは、汎用Webhookトリガーで受信可能です。

```
POST /api/v1/webhooks/{webhook_id}/trigger
X-Webhook-Signature: sha256=<hmac>
Content-Type: application/json
```

### 対応可能な外部イベント

| サービス | イベント例 | 設定方法 |
|---------|----------|---------|
| **GitHub** | Push, PR, Issue | Repository Settings → Webhooks |
| **Slack** | メンション, メッセージ | Slack App → Event Subscriptions |
| **Discord** | メッセージ | Bot → Outgoing Webhooks |
| **Notion** | ページ更新 | Notion Integration → Webhooks |
| **Stripe** | 支払い完了 | Dashboard → Webhooks |
| **Linear** | Issue更新 | Settings → Webhooks |

### Webhook入力マッピング

外部サービスのペイロードをワークフロー入力にマッピング:

```json
{
  "input_mapping": {
    "event_type": "$.action",
    "repo_name": "$.repository.name",
    "sender": "$.sender.login"
  }
}
```

---

## ブロック実装詳細

### Slack

```javascript
// config: { webhook_url?, channel?, message, username?, icon_emoji?, blocks? }
const webhookUrl = config.webhook_url || ctx.secrets.SLACK_WEBHOOK_URL;

const payload = {
    text: renderTemplate(config.message, input),
    channel: config.channel,
    username: config.username,
    icon_emoji: config.icon_emoji,
    blocks: config.blocks  // Block Kit対応
};

const response = await ctx.http.post(webhookUrl, payload);
return { success: true, status: response.status };
```

**設定オプション:**
| パラメータ | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `webhook_url` | string | - | Incoming Webhook URL |
| `channel` | string | - | 投稿先チャンネル |
| `message` | string | ✅ | メッセージ（テンプレート可） |
| `username` | string | - | ボット表示名 |
| `icon_emoji` | string | - | アイコン絵文字 |
| `blocks` | array | - | Block Kit形式 |

### Discord

```javascript
// config: { webhook_url?, content, username?, avatar_url?, embeds? }
const webhookUrl = config.webhook_url || ctx.secrets.DISCORD_WEBHOOK_URL;

const payload = {
    content: renderTemplate(config.content, input),
    username: config.username,
    avatar_url: config.avatar_url,
    embeds: config.embeds  // リッチ埋め込み対応
};

const response = await ctx.http.post(webhookUrl, payload);
return { success: true, status: response.status };
```

### Notion: ページ作成

```javascript
// config: { api_key?, parent_type, parent_id, title?, properties?, content? }
const apiKey = config.api_key || ctx.secrets.NOTION_API_KEY;

const payload = {
    parent: { [config.parent_type]: config.parent_id },
    properties: config.properties || {
        title: { title: [{ text: { content: renderTemplate(config.title, input) } }] }
    },
    children: config.content ? [
        { type: 'paragraph', paragraph: { rich_text: [{ text: { content: renderTemplate(config.content, input) } }] } }
    ] : undefined
};

const response = await ctx.http.post('https://api.notion.com/v1/pages', payload, {
    headers: { 'Authorization': 'Bearer ' + apiKey, 'Notion-Version': '2022-06-28' }
});

return { id: response.body.id, url: response.body.url };
```

### GitHub: Issue作成

```javascript
// config: { token?, owner, repo, title, body?, labels?, assignees? }
const token = config.token || ctx.secrets.GITHUB_TOKEN;
const url = `https://api.github.com/repos/${config.owner}/${config.repo}/issues`;

const response = await ctx.http.post(url, {
    title: renderTemplate(config.title, input),
    body: renderTemplate(config.body, input),
    labels: config.labels,
    assignees: config.assignees
}, {
    headers: { 'Authorization': 'Bearer ' + token, 'Accept': 'application/vnd.github+json' }
});

return { number: response.body.number, html_url: response.body.html_url };
```

### Web検索 (Tavily)

```javascript
// config: { api_key?, query, search_depth?, max_results?, include_answer? }
const apiKey = config.api_key || ctx.secrets.TAVILY_API_KEY;

const response = await ctx.http.post('https://api.tavily.com/search', {
    api_key: apiKey,
    query: renderTemplate(config.query, input),
    search_depth: config.search_depth || 'basic',
    max_results: config.max_results || 5,
    include_answer: config.include_answer !== false
});

return { answer: response.body.answer, results: response.body.results };
```

### Email (SendGrid)

```javascript
// config: { api_key?, from_email, from_name?, to_email, to_name?, subject, content_type?, content }
const apiKey = config.api_key || ctx.secrets.SENDGRID_API_KEY;

const response = await ctx.http.post('https://api.sendgrid.com/v3/mail/send', {
    personalizations: [{ to: [{ email: renderTemplate(config.to_email, input) }] }],
    from: { email: config.from_email, name: config.from_name },
    subject: renderTemplate(config.subject, input),
    content: [{ type: config.content_type || 'text/plain', value: renderTemplate(config.content, input) }]
}, {
    headers: { 'Authorization': 'Bearer ' + apiKey }
});

return { success: true };
```

### Linear: Issue作成

```javascript
// config: { api_key?, team_id, title, description?, priority?, label_ids?, assignee_id? }
const apiKey = config.api_key || ctx.secrets.LINEAR_API_KEY;

const mutation = `mutation($input: IssueCreateInput!) { issueCreate(input: $input) { issue { id identifier url } } }`;

const response = await ctx.http.post('https://api.linear.app/graphql', {
    query: mutation,
    variables: {
        input: {
            teamId: config.team_id,
            title: renderTemplate(config.title, input),
            description: renderTemplate(config.description, input),
            priority: config.priority
        }
    }
}, {
    headers: { 'Authorization': apiKey }
});

return response.body.data.issueCreate.issue;
```

---

## シークレット管理

### 必要なシークレット一覧

| シークレット名 | サービス | 取得方法 |
|---------------|---------|---------|
| `SLACK_WEBHOOK_URL` | Slack | Slack App → Incoming Webhooks |
| `DISCORD_WEBHOOK_URL` | Discord | Server Settings → Integrations → Webhooks |
| `NOTION_API_KEY` | Notion | My Integrations → Create new integration |
| `GOOGLE_API_KEY` | Google Sheets | Google Cloud Console → APIs & Services → Credentials |
| `GITHUB_TOKEN` | GitHub | Settings → Developer settings → Personal access tokens |
| `TAVILY_API_KEY` | Tavily | tavily.com → Dashboard → API Keys |
| `SENDGRID_API_KEY` | SendGrid | Settings → API Keys → Create API Key |
| `LINEAR_API_KEY` | Linear | Settings → API → Personal API keys |

### シークレットの設定方法

```bash
# API経由
curl -X POST http://localhost:8080/api/v1/credentials \
  -H "X-Tenant-ID: {tenant_id}" \
  -H "Content-Type: application/json" \
  -d '{"name": "SLACK_WEBHOOK_URL", "value": "https://hooks.slack.com/services/..."}'
```

---

## エラーコード一覧

### 通知系

| コード | 名前 | 説明 | リトライ可 |
|--------|------|------|-----------|
| `SLACK_001` | WEBHOOK_NOT_CONFIGURED | Webhook URLが未設定 | ❌ |
| `SLACK_002` | SEND_FAILED | 送信失敗 | ✅ |
| `DISCORD_001` | WEBHOOK_NOT_CONFIGURED | Webhook URLが未設定 | ❌ |
| `DISCORD_002` | SEND_FAILED | 送信失敗 | ✅ |
| `DISCORD_003` | RATE_LIMITED | レート制限 | ✅ |
| `EMAIL_001` | API_KEY_NOT_CONFIGURED | API Keyが未設定 | ❌ |
| `EMAIL_002` | SEND_FAILED | 送信失敗 | ✅ |

### ドキュメント系

| コード | 名前 | 説明 | リトライ可 |
|--------|------|------|-----------|
| `NOTION_001` | API_KEY_NOT_CONFIGURED | API Keyが未設定 | ❌ |
| `NOTION_002` | CREATE_FAILED | ページ作成失敗 | ✅ |
| `NOTION_003` | INVALID_PARENT | 無効な親ID | ❌ |
| `NOTION_004` | QUERY_FAILED | クエリ失敗 | ✅ |
| `GSHEETS_001` | API_KEY_NOT_CONFIGURED | API Keyが未設定 | ❌ |
| `GSHEETS_002` | APPEND_FAILED | 行追加失敗 | ✅ |
| `GSHEETS_003` | INVALID_SPREADSHEET | スプレッドシートが見つからない | ❌ |
| `GSHEETS_004` | READ_FAILED | 読み取り失敗 | ✅ |

### 開発ツール系

| コード | 名前 | 説明 | リトライ可 |
|--------|------|------|-----------|
| `GITHUB_001` | TOKEN_NOT_CONFIGURED | トークンが未設定 | ❌ |
| `GITHUB_002` | CREATE_FAILED | Issue作成失敗 | ✅ |
| `GITHUB_003` | REPO_NOT_FOUND | リポジトリが見つからない | ❌ |
| `GITHUB_004` | COMMENT_FAILED | コメント追加失敗 | ✅ |
| `LINEAR_001` | API_KEY_NOT_CONFIGURED | API Keyが未設定 | ❌ |
| `LINEAR_002` | CREATE_FAILED | Issue作成失敗 | ✅ |

### 検索系

| コード | 名前 | 説明 | リトライ可 |
|--------|------|------|-----------|
| `SEARCH_001` | API_KEY_NOT_CONFIGURED | API Keyが未設定 | ❌ |
| `SEARCH_002` | SEARCH_FAILED | 検索失敗 | ✅ |

---

## 関連ドキュメント

- [BLOCK_REGISTRY.md](./BLOCK_REGISTRY.md) - ブロック定義一覧
- [UNIFIED_BLOCK_MODEL.md](./designs/UNIFIED_BLOCK_MODEL.md) - ブロック実装アーキテクチャ
- [API.md](./API.md) - Webhook API仕様
- [BACKEND.md](./BACKEND.md) - バックエンド実装
