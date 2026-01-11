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
        └── schedules
        └── webhooks
  └── runs
        └── step_runs
  └── secrets
  └── audit_logs
  └── adapters
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
