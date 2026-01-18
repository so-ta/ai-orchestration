---
paths:
  - "backend/migrations/**/*.sql"
---

# Migration Rules

## ファイル命名

```
backend/migrations/XXX_description.sql
```

- XXX: 3桁の連番
- description: スネークケースで内容を説明

## ブロック追加

新規ブロックは `block_definitions` テーブルにINSERT:

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
| `ctx.http` | HTTP requests |
| `ctx.llm` | LLM API calls |
| `ctx.workflow` | Workflow control |
| `ctx.human` | Human-in-loop |
| `ctx.secrets` | Stored secrets |

## 実行コマンド

```bash
make db-reset   # Drop, apply schema, seed
```

## 参照

- [docs/BLOCK_REGISTRY.md](docs/BLOCK_REGISTRY.md)
- [docs/designs/UNIFIED_BLOCK_MODEL.md](docs/designs/UNIFIED_BLOCK_MODEL.md)
