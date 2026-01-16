# 外部サービス連携

> **Status**: ✅ Phase 1 実装済み
> **Updated**: 2025-01-12
> **Migration**: `013_add_integration_blocks.sql`

---

## 概要

AI Orchestrationプラットフォームでサポートする外部サービス連携の一覧です。
すべての連携ブロックはUnified Block Modelに基づき、JavaScriptコードとして実装されています。

---

## 実装済み連携サービス

### 通知・コミュニケーション

| サービス | ブロック | エンドポイント | 必要シークレット | 状態 |
|---------|---------|---------------|-----------------|--------|
| **Slack** | `slack` | メッセージ送信 | `SLACK_WEBHOOK_URL` | ✅ |
| **Discord** | `discord` | Webhook通知 | `DISCORD_WEBHOOK_URL` | ✅ |
| **Email (SendGrid)** | `email_sendgrid` | メール送信 | `SENDGRID_API_KEY` | ✅ |

### ドキュメント・データ管理

| サービス | ブロック | エンドポイント | 必要シークレット | 状態 |
|---------|---------|---------------|-----------------|--------|
| **Notion** | `notion_create_page` | ページ作成 | `NOTION_API_KEY` | ✅ |
| **Notion** | `notion_query_db` | データベース検索 | `NOTION_API_KEY` | ✅ |
| **Google Sheets** | `gsheets_append` | 行追加 | `GOOGLE_API_KEY` | ✅ |
| **Google Sheets** | `gsheets_read` | 範囲読み取り | `GOOGLE_API_KEY` | ✅ |

### 開発ツール

| サービス | ブロック | エンドポイント | 必要シークレット | 状態 |
|---------|---------|---------------|-----------------|--------|
| **GitHub** | `github_create_issue` | Issue作成 | `GITHUB_TOKEN` | ✅ |
| **GitHub** | `github_add_comment` | コメント追加 | `GITHUB_TOKEN` | ✅ |
| **Linear** | `linear_create_issue` | Issue作成 | `LINEAR_API_KEY` | ✅ |

### 検索・情報取得

| サービス | ブロック | エンドポイント | 必要シークレット | 状態 |
|---------|---------|---------------|-----------------|--------|
| **Tavily** | `web_search` | Web検索 | `TAVILY_API_KEY` | ✅ |

---

## 未実装（今後追加予定）

### 優先度 2: 高価値

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

### 優先度 3: 拡張

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

各ブロックの技術仕様は以下を参照：

- **ブロック定義・設定スキーマ**: [BLOCK_REGISTRY.md](./BLOCK_REGISTRY.md#external-integration-blocks)
- **実装コード**: `backend/schema/seed.sql` (Single Source of Truth)
- **アーキテクチャ**: [UNIFIED_BLOCK_MODEL.md](./designs/UNIFIED_BLOCK_MODEL.md)

### 設定オプション概要

| ブロック | 主要パラメータ | 必須 |
|---------|--------------|------|
| `slack` | `message`, `webhook_url`/secret | message |
| `discord` | `content`, `webhook_url`/secret | content |
| `email_sendgrid` | `to_email`, `subject`, `content`, `from_email` | all |
| `notion_create_page` | `parent_type`, `parent_id`, `title` | all |
| `notion_query_db` | `database_id` | database_id |
| `gsheets_append` | `spreadsheet_id`, `range`, `values` | all |
| `gsheets_read` | `spreadsheet_id`, `range` | all |
| `github_create_issue` | `owner`, `repo`, `title` | all |
| `github_add_comment` | `owner`, `repo`, `issue_number`, `body` | all |
| `linear_create_issue` | `team_id`, `title` | all |
| `web_search` | `query` | query |

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
