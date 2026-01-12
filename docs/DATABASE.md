# Database Reference

## Connection

```
Driver: PostgreSQL 16
URL: postgres://user:pass@localhost:5432/ai_orchestration?sslmode=disable
Pool: pgx connection pool
```

## Schema Overview

```
tenants
  └── users
  └── workflows
        └── workflow_versions
        └── steps
        └── edges
        └── block_groups
        └── schedules
        └── webhooks
  └── runs
        └── step_runs
        └── block_group_runs
        └── usage_records
  └── usage_daily_aggregates
  └── usage_budgets
  └── secrets
  └── audit_logs
  └── adapters
  └── block_definitions (※ tenant_id NULL = system)
        └── block_versions
```

## Tables

### tenants

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| name | VARCHAR(255) | NOT NULL | |
| slug | VARCHAR(255) | NOT NULL, UNIQUE | URL-safe identifier |
| settings | JSONB | DEFAULT '{}' | Tenant config |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |
| deleted_at | TIMESTAMPTZ | | Soft delete |

Default tenant: `00000000-0000-0000-0000-000000000001`

### users

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK | Keycloak user ID |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| email | VARCHAR(255) | NOT NULL | |
| name | VARCHAR(255) | | |
| role | VARCHAR(50) | NOT NULL DEFAULT 'viewer' | tenant_admin, builder, operator, viewer |
| last_login_at | TIMESTAMPTZ | | |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |

Unique: (tenant_id, email)

### workflows

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| name | VARCHAR(255) | NOT NULL | |
| description | TEXT | | |
| status | VARCHAR(50) | NOT NULL DEFAULT 'draft' | draft, published |
| version | INTEGER | NOT NULL DEFAULT 1 | Increments on publish |
| input_schema | JSONB | | JSON Schema |
| output_schema | JSONB | | JSON Schema |
| created_by | UUID | FK users(id) | |
| published_at | TIMESTAMPTZ | | |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |
| deleted_at | TIMESTAMPTZ | | Soft delete |

Indexes:
- `idx_workflows_tenant` ON (tenant_id)
- `idx_workflows_status` ON (status)

### workflow_versions

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| workflow_id | UUID | FK workflows(id), NOT NULL | |
| version | INTEGER | NOT NULL | |
| definition | JSONB | NOT NULL | Full snapshot (steps, edges) |
| published_by | UUID | FK users(id) | |
| published_at | TIMESTAMPTZ | DEFAULT NOW() | |

Unique: (workflow_id, version)

### steps

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| workflow_id | UUID | FK workflows(id) ON DELETE CASCADE, NOT NULL | |
| name | VARCHAR(255) | NOT NULL | |
| type | VARCHAR(50) | NOT NULL | llm, tool, condition, map, join, subflow |
| config | JSONB | NOT NULL DEFAULT '{}' | Type-specific config |
| position_x | INTEGER | DEFAULT 0 | UI position |
| position_y | INTEGER | DEFAULT 0 | UI position |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |

### edges

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| workflow_id | UUID | FK workflows(id) ON DELETE CASCADE, NOT NULL | |
| source_step_id | UUID | FK steps(id) ON DELETE CASCADE, NOT NULL | |
| target_step_id | UUID | FK steps(id) ON DELETE CASCADE, NOT NULL | |
| condition | TEXT | | Expression for conditional routing |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |

Unique: (source_step_id, target_step_id)

### block_groups

Control flow constructs that group multiple steps.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| workflow_id | UUID | FK workflows(id) ON DELETE CASCADE, NOT NULL | |
| name | VARCHAR(255) | NOT NULL | Display name |
| type | VARCHAR(50) | NOT NULL | parallel, try_catch, if_else, switch_case, foreach, while |
| config | JSONB | NOT NULL DEFAULT '{}' | Type-specific configuration |
| parent_group_id | UUID | FK block_groups(id) ON DELETE CASCADE | For nested groups |
| position_x | INT | DEFAULT 0 | UI position X |
| position_y | INT | DEFAULT 0 | UI position Y |
| width | INT | DEFAULT 400 | UI width |
| height | INT | DEFAULT 300 | UI height |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |

Indexes:
- `idx_block_groups_workflow` ON (workflow_id)
- `idx_block_groups_parent` ON (parent_group_id)

**Note**: Steps can belong to a block group via `steps.block_group_id` and `steps.group_role`.

### block_group_runs

Execution tracking for block groups.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| run_id | UUID | FK runs(id) ON DELETE CASCADE, NOT NULL | |
| block_group_id | UUID | FK block_groups(id) ON DELETE CASCADE, NOT NULL | |
| status | VARCHAR(50) | DEFAULT 'pending' | pending, running, completed, failed, skipped |
| iteration | INT | DEFAULT 0 | For loop groups |
| input | JSONB | | Group input |
| output | JSONB | | Group output |
| error | TEXT | | Error message |
| started_at | TIMESTAMPTZ | | |
| completed_at | TIMESTAMPTZ | | |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |

Indexes:
- `idx_block_group_runs_run` ON (run_id)
- `idx_block_group_runs_block_group` ON (block_group_id)

### runs

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| workflow_id | UUID | FK workflows(id), NOT NULL | |
| workflow_version | INTEGER | NOT NULL | Snapshot version |
| status | VARCHAR(50) | NOT NULL DEFAULT 'pending' | pending, running, completed, failed, cancelled |
| mode | VARCHAR(50) | NOT NULL DEFAULT 'production' | test, production |
| input | JSONB | | |
| output | JSONB | | |
| error | TEXT | | |
| triggered_by | VARCHAR(50) | NOT NULL DEFAULT 'manual' | manual, schedule, webhook |
| triggered_by_user | UUID | FK users(id) | |
| started_at | TIMESTAMPTZ | | |
| completed_at | TIMESTAMPTZ | | |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |

Indexes:
- `idx_runs_tenant` ON (tenant_id)
- `idx_runs_workflow` ON (workflow_id)
- `idx_runs_status` ON (status)

### step_runs

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| run_id | UUID | FK runs(id) ON DELETE CASCADE, NOT NULL | |
| step_id | UUID | NOT NULL | Reference to step at execution time |
| step_name | VARCHAR(255) | NOT NULL | Snapshot of step name |
| status | VARCHAR(50) | NOT NULL DEFAULT 'pending' | pending, running, completed, failed |
| attempt | INTEGER | NOT NULL DEFAULT 1 | Retry count |
| input | JSONB | | |
| output | JSONB | | |
| error | TEXT | | |
| started_at | TIMESTAMPTZ | | |
| completed_at | TIMESTAMPTZ | | |
| duration_ms | INTEGER | | |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |

Indexes:
- `idx_step_runs_run` ON (run_id)

### schedules

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| workflow_id | UUID | FK workflows(id), NOT NULL | |
| workflow_version | INTEGER | NOT NULL DEFAULT 1 | |
| name | VARCHAR(255) | NOT NULL | |
| description | TEXT | | |
| cron_expression | VARCHAR(100) | NOT NULL | Standard cron format |
| timezone | VARCHAR(50) | NOT NULL DEFAULT 'UTC' | IANA timezone |
| input | JSONB | | Default input for runs |
| status | VARCHAR(50) | NOT NULL DEFAULT 'active' | active, paused |
| next_run_at | TIMESTAMPTZ | | Computed next execution |
| last_run_at | TIMESTAMPTZ | | |
| last_run_id | UUID | FK runs(id) | |
| run_count | INTEGER | NOT NULL DEFAULT 0 | |
| created_by | UUID | FK users(id) | |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |

Indexes:
- `idx_schedules_tenant` ON (tenant_id)
- `idx_schedules_next_run` ON (next_run_at) WHERE status = 'active'

### webhooks

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| workflow_id | UUID | FK workflows(id), NOT NULL | |
| name | VARCHAR(255) | NOT NULL | |
| description | TEXT | | |
| secret | VARCHAR(255) | NOT NULL | HMAC signing key |
| input_mapping | JSONB | | JSONPath mapping |
| enabled | BOOLEAN | NOT NULL DEFAULT true | |
| last_triggered_at | TIMESTAMPTZ | | |
| trigger_count | INTEGER | NOT NULL DEFAULT 0 | |
| created_by | UUID | FK users(id) | |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |

### adapters

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | VARCHAR(100) | PK | mock, openai, anthropic, http |
| tenant_id | UUID | FK tenants(id) | NULL = global |
| name | VARCHAR(255) | NOT NULL | |
| description | TEXT | | |
| type | VARCHAR(50) | NOT NULL | builtin, custom |
| config | JSONB | | Default config |
| input_schema | JSONB | | JSON Schema |
| output_schema | JSONB | | JSON Schema |
| enabled | BOOLEAN | NOT NULL DEFAULT true | |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |

### block_definitions

ブロック定義（Unified Block Model）。システムブロックとテナントカスタムブロックを管理。

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT gen_random_uuid() | |
| tenant_id | UUID | FK tenants(id) | NULL = system block |
| slug | VARCHAR(100) | NOT NULL | Unique identifier |
| name | VARCHAR(255) | NOT NULL | Display name |
| description | TEXT | | |
| category | VARCHAR(50) | NOT NULL | ai, logic, integration, data, control, utility |
| icon | VARCHAR(50) | | Icon identifier |
| config_schema | JSONB | NOT NULL DEFAULT '{}' | Config JSON Schema |
| input_schema | JSONB | | Input JSON Schema |
| output_schema | JSONB | | Output JSON Schema |
| executor_type | VARCHAR(20) | NOT NULL | builtin, http, function, code |
| executor_config | JSONB | | Legacy executor config |
| **code** | TEXT | | **JavaScript code (Unified Block Model)** |
| **ui_config** | JSONB | NOT NULL DEFAULT '{}' | **{icon, color, configSchema}** |
| **is_system** | BOOLEAN | NOT NULL DEFAULT FALSE | **System block = admin only** |
| **version** | INTEGER | NOT NULL DEFAULT 1 | **Version number** |
| error_codes | JSONB | DEFAULT '[]' | Error code definitions |
| enabled | BOOLEAN | DEFAULT true | |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |

Unique: (tenant_id, slug)

Indexes:
- `idx_block_definitions_tenant` ON (tenant_id)
- `idx_block_definitions_category` ON (category)
- `idx_block_definitions_enabled` ON (enabled)

**See**: [UNIFIED_BLOCK_MODEL.md](./designs/UNIFIED_BLOCK_MODEL.md)

### block_versions

ブロック定義のバージョン履歴。ロールバック機能をサポート。

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT gen_random_uuid() | |
| block_id | UUID | FK block_definitions(id) ON DELETE CASCADE, NOT NULL | |
| version | INTEGER | NOT NULL | Version number |
| code | TEXT | NOT NULL | Code snapshot |
| config_schema | JSONB | NOT NULL | Config schema snapshot |
| input_schema | JSONB | | Input schema snapshot |
| output_schema | JSONB | | Output schema snapshot |
| ui_config | JSONB | NOT NULL | UI config snapshot |
| change_summary | TEXT | | Change description |
| changed_by | UUID | | User who made the change |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | |

Unique: (block_id, version)

Indexes:
- `idx_block_versions_block_id` ON (block_id)
- `idx_block_versions_created_at` ON (created_at)

### usage_records

Individual LLM API call records for cost tracking.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| workflow_id | UUID | FK workflows(id) | Nullable for non-workflow calls |
| run_id | UUID | FK runs(id) | |
| step_run_id | UUID | FK step_runs(id) | |
| provider | VARCHAR(50) | NOT NULL | openai, anthropic, google |
| model | VARCHAR(100) | NOT NULL | gpt-4o, claude-3-opus, etc. |
| operation | VARCHAR(50) | NOT NULL | chat, completion, embedding |
| input_tokens | INT | NOT NULL DEFAULT 0 | Prompt tokens |
| output_tokens | INT | NOT NULL DEFAULT 0 | Completion tokens |
| total_tokens | INT | NOT NULL DEFAULT 0 | input + output |
| input_cost_usd | DECIMAL(12, 8) | NOT NULL DEFAULT 0 | Cost for input tokens |
| output_cost_usd | DECIMAL(12, 8) | NOT NULL DEFAULT 0 | Cost for output tokens |
| total_cost_usd | DECIMAL(12, 8) | NOT NULL DEFAULT 0 | Total cost |
| latency_ms | INT | | Response time |
| success | BOOLEAN | NOT NULL DEFAULT TRUE | Whether call succeeded |
| error_message | TEXT | | Error details if failed |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | |

Indexes:
- `idx_usage_records_tenant_created` ON (tenant_id, created_at DESC)
- `idx_usage_records_workflow` ON (workflow_id) WHERE workflow_id IS NOT NULL
- `idx_usage_records_run` ON (run_id) WHERE run_id IS NOT NULL

### usage_daily_aggregates

Pre-aggregated daily usage for dashboard performance.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| workflow_id | UUID | FK workflows(id) | NULL for tenant-wide aggregate |
| date | DATE | NOT NULL | Aggregation date |
| provider | VARCHAR(50) | NOT NULL | |
| model | VARCHAR(100) | NOT NULL | |
| total_requests | INT | NOT NULL DEFAULT 0 | |
| total_input_tokens | BIGINT | NOT NULL DEFAULT 0 | |
| total_output_tokens | BIGINT | NOT NULL DEFAULT 0 | |
| total_cost_usd | DECIMAL(12, 6) | NOT NULL DEFAULT 0 | |
| avg_latency_ms | INT | | |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | |

Unique: (tenant_id, workflow_id, date, provider, model)

Indexes:
- `idx_usage_daily_tenant_date` ON (tenant_id, date DESC)

### usage_budgets

Budget limits and alert thresholds.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| workflow_id | UUID | FK workflows(id) | NULL for tenant-wide budget |
| budget_type | VARCHAR(50) | NOT NULL | monthly, daily |
| budget_amount_usd | DECIMAL(12, 2) | NOT NULL | Budget limit |
| alert_threshold | DECIMAL(3, 2) | NOT NULL DEFAULT 0.80 | 0.0-1.0, triggers alert |
| enabled | BOOLEAN | NOT NULL DEFAULT TRUE | |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | |

Indexes:
- `idx_usage_budgets_tenant` ON (tenant_id)
- `idx_usage_budgets_workflow` ON (workflow_id) WHERE workflow_id IS NOT NULL

### secrets

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| name | VARCHAR(255) | NOT NULL | |
| encrypted_value | TEXT | NOT NULL | AES-256 encrypted |
| created_by | UUID | FK users(id) | |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | |

Unique: (tenant_id, name)

### audit_logs

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK, DEFAULT uuid_generate_v4() | |
| tenant_id | UUID | FK tenants(id), NOT NULL | |
| actor_id | UUID | | User who performed action |
| actor_email | VARCHAR(255) | | |
| action | VARCHAR(100) | NOT NULL | create, update, delete, publish, execute |
| resource_type | VARCHAR(100) | NOT NULL | workflow, run, secret |
| resource_id | UUID | | |
| metadata | JSONB | | Additional context |
| ip_address | INET | | |
| user_agent | TEXT | | |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | |

Indexes:
- `idx_audit_logs_tenant` ON (tenant_id)
- `idx_audit_logs_created` ON (created_at)

## Query Patterns

### List Workflows (with tenant isolation)

```sql
SELECT *
FROM workflows
WHERE tenant_id = $1
  AND deleted_at IS NULL
ORDER BY updated_at DESC
LIMIT $2 OFFSET $3;
```

### Get Workflow with Steps and Edges

```sql
-- Workflow
SELECT * FROM workflows WHERE id = $1 AND tenant_id = $2;

-- Steps
SELECT * FROM steps WHERE workflow_id = $1 ORDER BY created_at;

-- Edges
SELECT * FROM edges WHERE workflow_id = $1;
```

### Get Run with StepRuns

```sql
SELECT r.*, json_agg(sr.*) AS step_runs
FROM runs r
LEFT JOIN step_runs sr ON sr.run_id = r.id
WHERE r.id = $1
GROUP BY r.id;
```

### Find Active Schedules Due

```sql
SELECT *
FROM schedules
WHERE status = 'active'
  AND next_run_at <= NOW()
ORDER BY next_run_at;
```

### Count Runs by Status (for dashboard)

```sql
SELECT status, COUNT(*) as count
FROM runs
WHERE tenant_id = $1
  AND created_at >= $2
GROUP BY status;
```

### Get Usage Summary by Period

```sql
SELECT
    COUNT(*) as total_requests,
    COALESCE(SUM(input_tokens), 0) as total_input_tokens,
    COALESCE(SUM(output_tokens), 0) as total_output_tokens,
    COALESCE(SUM(total_cost_usd), 0) as total_cost_usd,
    AVG(CASE WHEN success THEN 1 ELSE 0 END) as success_rate,
    AVG(latency_ms) as avg_latency_ms
FROM usage_records
WHERE tenant_id = $1
  AND created_at >= $2
  AND created_at < $3;
```

### Get Usage by Model

```sql
SELECT
    provider,
    model,
    COUNT(*) as total_requests,
    SUM(input_tokens) as total_input_tokens,
    SUM(output_tokens) as total_output_tokens,
    SUM(total_cost_usd) as total_cost_usd,
    AVG(latency_ms) as avg_latency_ms
FROM usage_records
WHERE tenant_id = $1
  AND created_at >= $2
  AND created_at < $3
GROUP BY provider, model
ORDER BY total_cost_usd DESC;
```

## Migration Commands

```bash
# Apply migrations (using golang-migrate)
docker compose exec api migrate -path /migrations -database "$DATABASE_URL" up

# Rollback last migration
docker compose exec api migrate -path /migrations -database "$DATABASE_URL" down 1

# Force version (dangerous)
docker compose exec api migrate -path /migrations -database "$DATABASE_URL" force VERSION
```

## Soft Delete Pattern

All tenant-owned tables support soft delete via `deleted_at` column:

```sql
-- "Delete"
UPDATE workflows SET deleted_at = NOW() WHERE id = $1;

-- Query (exclude deleted)
SELECT * FROM workflows WHERE deleted_at IS NULL;

-- Hard delete (admin only)
DELETE FROM workflows WHERE id = $1;
```

## Multi-Tenancy Pattern

All queries MUST include `tenant_id`:

```go
func (r *WorkflowRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Workflow, error) {
    return r.db.QueryRow(ctx,
        `SELECT * FROM workflows WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
        id, tenantID,
    ).Scan(...)
}
```

## JSONB Column Usage

| Table | Column | Content |
|-------|--------|---------|
| tenants | settings | `{"data_retention_days": 30, "max_concurrent_runs": 10}` |
| workflows | input_schema | JSON Schema |
| workflows | output_schema | JSON Schema |
| steps | config | Step type-specific config |
| runs | input | Execution input |
| runs | output | Execution result |
| webhooks | input_mapping | JSONPath mappings |
| audit_logs | metadata | Action-specific details |

## Connection Pool Settings

```go
config := pgxpool.Config{
    MaxConns:          25,
    MinConns:          5,
    MaxConnLifetime:   time.Hour,
    MaxConnIdleTime:   30 * time.Minute,
    HealthCheckPeriod: time.Minute,
}
```

## Backup

```bash
# Dump
pg_dump -h localhost -U postgres ai_orchestration > backup.sql

# Restore
psql -h localhost -U postgres ai_orchestration < backup.sql
```
