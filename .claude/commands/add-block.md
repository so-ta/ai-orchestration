# Add New Block Workflow

新規ブロック（Discord, Slack, Notion等の外部連携）を追加する際のワークフロー。

## 必読ドキュメント

以下を**必ず先に読む**こと：

1. `docs/designs/UNIFIED_BLOCK_MODEL.md` - Block execution architecture
2. `docs/BLOCK_REGISTRY.md` - Existing block definitions
3. `backend/migrations/011_unified_block_model.sql` - 既存パターン確認

## 標準手順（Migration追加）

**ほとんどのブロックはこの方式で追加する:**

1. Migrationファイル作成: `backend/migrations/XXX_{name}_block.sql`
2. `block_definitions`テーブルにINSERT
   - `tenant_id = NULL` でシステムブロック
   - `code`にJavaScriptコード（`ctx.http`等を使用）
   - `ui_config`にアイコン・カラー・設定スキーマ
3. Migration実行: `make db-reset`
4. `docs/BLOCK_REGISTRY.md` を更新

## コード例

```sql
INSERT INTO block_definitions (tenant_id, slug, name, category, code, ui_config, is_system)
VALUES (
  NULL,  -- システムブロック
  'discord',
  'Discord通知',
  'integration',
  $code$
    const webhookUrl = config.webhook_url || ctx.secrets.DISCORD_WEBHOOK_URL;
    const payload = { content: renderTemplate(config.message, input) };
    return await ctx.http.post(webhookUrl, payload);
  $code$,
  '{"icon": "message-circle", "color": "#5865F2", "configSchema": {...}}',
  TRUE
);
```

## ctx インターフェース

| Interface | Purpose |
|-----------|---------|
| `ctx.http` | HTTP requests (GET, POST, etc.) |
| `ctx.llm` | LLM API calls |
| `ctx.workflow` | Workflow control |
| `ctx.human` | Human-in-loop interactions |
| `ctx.secrets` | Access to stored secrets |

## Go Adapterが必要な例外ケース

| ケース | 理由 |
|--------|------|
| LLMプロバイダー追加 | `ctx.llm`経由で呼び出すため |
| 複雑な認証フロー | OAuth2等、JSでは困難な場合 |
| バイナリ処理 | 画像・ファイル処理等 |

Go Adapter追加が必要な場合:
1. Create `backend/internal/adapter/{name}.go`
2. Implement `Adapter` interface
3. Register in registry
4. Add test `{name}_test.go`
5. Update `docs/BACKEND.md`

## チェックリスト

```
[ ] UNIFIED_BLOCK_MODEL.md を読んだ
[ ] 既存ブロックのパターンを確認した
[ ] Migrationファイルを作成した
[ ] make db-reset でテスト
[ ] BLOCK_REGISTRY.md を更新した
[ ] テストを追加した
```
