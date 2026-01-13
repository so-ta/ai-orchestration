# API Reference

REST API endpoints, request/response schemas, and authentication.

## Quick Reference

| Item | Value |
|------|-------|
| Base URL | `/api/v1` |
| Auth | Bearer JWT |
| Content-Type | `application/json` |
| Tenant (Dev) | `X-Tenant-ID` header |
| Tenant (Prod) | JWT claim |
| Health Check | `GET /health`, `GET /ready` |

## Headers

| Header | Required | Description |
|--------|----------|-------------|
| `Authorization` | Yes* | `Bearer <token>` (*unless AUTH_ENABLED=false) |
| `Content-Type` | Yes | `application/json` |
| `X-Tenant-ID` | Dev only | UUID, required when AUTH_ENABLED=false |
| `X-Request-ID` | No | UUID for tracing |

## Error Response

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable message",
    "details": {}
  }
}
```

| Code | HTTP | Description |
|------|------|-------------|
| `UNAUTHORIZED` | 401 | Invalid/missing token |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `NOT_FOUND` | 404 | Resource not found |
| `VALIDATION_ERROR` | 400 | Invalid request body |
| `CONFLICT` | 409 | Resource conflict |
| `INTERNAL_ERROR` | 500 | Server error |
| `RATE_LIMIT_EXCEEDED` | 429 | Rate limit exceeded |

---

## Rate Limiting

API requests are rate limited at multiple scopes to ensure fair usage.

### Rate Limit Scopes

| Scope | Default Limit | Window | Description |
|-------|--------------|--------|-------------|
| `tenant` | 1000 req | 1 min | Per-tenant limit across all endpoints |
| `workflow` | 100 req | 1 min | Per-workflow limit for run creation |
| `webhook` | 60 req | 1 min | Per-webhook-key limit for trigger endpoint |

### Rate Limit Headers

All responses include rate limit headers:

```
X-RateLimit-tenant-Limit: 1000
X-RateLimit-tenant-Remaining: 999
X-RateLimit-tenant-Reset: 1704067200
```

### Rate Limit Error Response

```json
{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Rate limit exceeded for tenant scope",
    "retry_at": "2024-01-01T00:00:00Z",
    "limit": 1000,
    "scope": "tenant"
  }
}
```

### Configuration

Rate limits can be configured via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `RATE_LIMIT_ENABLED` | `true` | Enable/disable rate limiting |
| `RATE_LIMIT_TENANT` | `1000` | Requests per minute per tenant |
| `RATE_LIMIT_WORKFLOW` | `100` | Requests per minute per workflow |
| `RATE_LIMIT_WEBHOOK` | `60` | Requests per minute per webhook key |

---

## Workflows

### List
```
GET /workflows
```

Query:
| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `status` | string | - | `draft` or `published` |
| `page` | int | 1 | Page number |
| `limit` | int | 20 | Items per page (max 100) |

Response `200`:
```json
{
  "data": [
    {
      "id": "uuid",
      "name": "string",
      "description": "string",
      "status": "draft|published",
      "version": 1,
      "created_at": "ISO8601",
      "updated_at": "ISO8601"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 100
  }
}
```

### Create
```
POST /workflows
```

Request:
```json
{
  "name": "string (required)",
  "description": "string",
  "input_schema": {}
}
```

Response `201`:
```json
{
  "id": "uuid",
  "name": "string",
  "description": "string",
  "status": "draft",
  "version": 1,
  "input_schema": {},
  "created_at": "ISO8601",
  "updated_at": "ISO8601"
}
```

### Get
```
GET /workflows/{id}
```

Response `200`: Same as Create response

### Update
```
PUT /workflows/{id}
```

Constraint: Only `draft` status

Request:
```json
{
  "name": "string",
  "description": "string",
  "input_schema": {}
}
```

Response `200`: Updated workflow

### Delete
```
DELETE /workflows/{id}
```

Response `204`: No content

### Publish
```
POST /workflows/{id}/publish
```

Constraint: Must be `draft` status

Response `200`:
```json
{
  "id": "uuid",
  "status": "published",
  "version": 2,
  "published_at": "ISO8601"
}
```

---

## Steps

### List
```
GET /workflows/{workflow_id}/steps
```

Response `200`:
```json
{
  "data": [
    {
      "id": "uuid",
      "workflow_id": "uuid",
      "name": "string",
      "type": "llm|tool|condition|map|join|subflow",
      "config": {},
      "position": {"x": 0, "y": 0},
      "created_at": "ISO8601",
      "updated_at": "ISO8601"
    }
  ]
}
```

### Create
```
POST /workflows/{workflow_id}/steps
```

Request:
```json
{
  "name": "string (required)",
  "type": "llm|tool|condition|map|join|subflow (required)",
  "config": {},
  "position": {"x": 0, "y": 0}
}
```

Config by type:

**llm**:
```json
{
  "provider": "openai|anthropic",
  "model": "gpt-4|claude-3-opus-20240229",
  "prompt": "string with {{input.field}} templates",
  "temperature": 0.7,
  "max_tokens": 1000
}
```

**tool**:
```json
{
  "adapter_id": "mock|http|openai|anthropic",
  "...adapter_specific"
}
```

**condition**:
```json
{
  "expression": "$.field > 10"
}
```

**map**:
```json
{
  "input_path": "$.items",
  "parallel": true,
  "max_concurrency": 5
}
```

Response `201`: Created step

### Update
```
PUT /workflows/{workflow_id}/steps/{step_id}
```

Request: Same as Create
Response `200`: Updated step

### Delete
```
DELETE /workflows/{workflow_id}/steps/{step_id}
```

Response `204`: No content

---

## Edges

### List
```
GET /workflows/{workflow_id}/edges
```

Response `200`:
```json
{
  "data": [
    {
      "id": "uuid",
      "workflow_id": "uuid",
      "source_step_id": "uuid",
      "target_step_id": "uuid",
      "condition": "string (optional)",
      "created_at": "ISO8601"
    }
  ]
}
```

### Create
```
POST /workflows/{workflow_id}/edges
```

Request:
```json
{
  "source_step_id": "uuid (required)",
  "target_step_id": "uuid (required)",
  "condition": "$.success == true"
}
```

Response `201`: Created edge

Validation:
- Rejects cyclic connections
- Source and target must exist

### Delete
```
DELETE /workflows/{workflow_id}/edges/{edge_id}
```

Response `204`: No content

---

## Block Groups

Block groups are control flow constructs that group multiple steps.

### List
```
GET /workflows/{workflow_id}/block-groups
```

Response `200`:
```json
{
  "data": [
    {
      "id": "uuid",
      "workflow_id": "uuid",
      "name": "Parallel Tasks",
      "type": "parallel",
      "config": { "max_concurrent": 10 },
      "parent_group_id": null,
      "position": { "x": 100, "y": 200 },
      "size": { "width": 400, "height": 300 }
    }
  ]
}
```

### Create
```
POST /workflows/{workflow_id}/block-groups
```

Request:
```json
{
  "name": "Parallel Tasks",
  "type": "parallel|try_catch|if_else|switch_case|foreach|while",
  "config": {},
  "parent_group_id": null,
  "position": { "x": 100, "y": 200 },
  "size": { "width": 400, "height": 300 }
}
```

Response `201`: Created block group

### Get
```
GET /workflows/{workflow_id}/block-groups/{group_id}
```

Response `200`: Block group details

### Update
```
PUT /workflows/{workflow_id}/block-groups/{group_id}
```

Request:
```json
{
  "name": "Updated Name",
  "config": { "max_concurrent": 5 },
  "position": { "x": 150, "y": 250 },
  "size": { "width": 500, "height": 400 }
}
```

Response `200`: Updated block group

### Delete
```
DELETE /workflows/{workflow_id}/block-groups/{group_id}
```

Response `204`: No content

### Add Step to Group
```
POST /workflows/{workflow_id}/block-groups/{group_id}/steps
```

Request:
```json
{
  "step_id": "uuid",
  "group_role": "body|try|catch|finally|then|else|case_0|default"
}
```

Response `200`: Updated step

**Restrictions:**
- `start` steps cannot be added to block groups (returns `400 VALIDATION_ERROR`)

**Possible Errors:**

| Code | Message | Description |
|------|---------|-------------|
| VALIDATION_ERROR | this step type cannot be added to a block group | Start nodes cannot be in groups |
| VALIDATION_ERROR | invalid group role | Invalid group_role for the block group type |
| NOT_FOUND | block group not found | Block group does not exist |
| CONFLICT | published workflow cannot be edited | Workflow is published |

### Get Steps in Group
```
GET /workflows/{workflow_id}/block-groups/{group_id}/steps
```

Response `200`: Array of steps

### Remove Step from Group
```
DELETE /workflows/{workflow_id}/block-groups/{group_id}/steps/{step_id}
```

Response `200`: Updated step (with null block_group_id)

---

## Runs

### Execute
```
POST /workflows/{workflow_id}/runs
```

Request:
```json
{
  "input": {},
  "triggered_by": "manual|test|webhook|schedule|internal",
  "version": 0
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `input` | object | `{}` | Input data for the workflow |
| `triggered_by` | string | `manual` | Trigger type: `manual`, `test`, `webhook`, `schedule`, `internal` |
| `version` | int | 0 | Workflow version to execute (0 = latest) |
| `mode` | string | - | **Deprecated**: Use `triggered_by` instead (`mode: "test"` maps to `triggered_by: "test"`) |

Response `201`:
```json
{
  "id": "uuid",
  "workflow_id": "uuid",
  "workflow_version": 1,
  "status": "pending",
  "triggered_by": "manual",
  "run_number": 1,
  "created_at": "ISO8601"
}
```

### List by Workflow
```
GET /workflows/{workflow_id}/runs
```

Query:
| Param | Type | Default |
|-------|------|---------|
| `status` | string | - |
| `page` | int | 1 |
| `limit` | int | 20 |

Response `200`: Paginated runs

### Get
```
GET /runs/{run_id}
```

Response `200`:
```json
{
  "id": "uuid",
  "workflow_id": "uuid",
  "workflow_version": 1,
  "status": "completed",
  "mode": "production",
  "trigger_type": "manual",
  "input": {},
  "output": {},
  "error": "string (if failed)",
  "started_at": "ISO8601",
  "completed_at": "ISO8601",
  "duration_ms": 1000,
  "step_runs": [
    {
      "id": "uuid",
      "step_id": "uuid",
      "step_name": "string",
      "status": "completed",
      "attempt": 1,
      "input": {},
      "output": {},
      "error": "",
      "started_at": "ISO8601",
      "completed_at": "ISO8601",
      "duration_ms": 500
    }
  ]
}
```

### Cancel
```
POST /runs/{run_id}/cancel
```

Response `200`: Updated run with `status: cancelled`

### Resume From Step
```
POST /runs/{run_id}/resume
```

Resume execution from a specific step through all downstream steps.

Request:
```json
{
  "from_step_id": "uuid (required)",
  "input_override": {}
}
```

Constraint: Run must be `completed` or `failed` status

Response `202`:
```json
{
  "data": {
    "run_id": "uuid",
    "from_step_id": "uuid",
    "steps_to_execute": ["uuid", "uuid", "uuid"]
  }
}
```

### Execute Single Step
```
POST /runs/{run_id}/steps/{step_id}/execute
```

Re-execute only a single step from an existing run.

Request:
```json
{
  "input": {}
}
```

Constraint: Run must be `completed` or `failed` status

Response `202`:
```json
{
  "data": {
    "id": "uuid",
    "run_id": "uuid",
    "step_id": "uuid",
    "step_name": "string",
    "status": "pending",
    "attempt": 2
  }
}
```

### Get Step History
```
GET /runs/{run_id}/steps/{step_id}/history
```

Get all execution history for a specific step in a run.

Response `200`:
```json
{
  "data": [
    {
      "id": "uuid",
      "run_id": "uuid",
      "step_id": "uuid",
      "step_name": "string",
      "status": "completed",
      "attempt": 2,
      "input": {},
      "output": {},
      "error": "",
      "started_at": "ISO8601",
      "completed_at": "ISO8601",
      "duration_ms": 500
    },
    {
      "id": "uuid",
      "run_id": "uuid",
      "step_id": "uuid",
      "step_name": "string",
      "status": "failed",
      "attempt": 1,
      "input": {},
      "output": {},
      "error": "Error message",
      "started_at": "ISO8601",
      "completed_at": "ISO8601",
      "duration_ms": 200
    }
  ]
}
```

### Test Step Inline
```
POST /workflows/{workflow_id}/steps/{step_id}/test
```

Test a single step without requiring an existing run. Creates a temporary run and executes only the specified step.

Request:
```json
{
  "input": {}
}
```

Response `202`:
```json
{
  "data": {
    "run": {
      "id": "uuid",
      "workflow_id": "uuid",
      "status": "running",
      "triggered_by": "test"
    },
    "step_run": {
      "id": "uuid",
      "run_id": "uuid",
      "step_id": "uuid",
      "step_name": "string",
      "status": "pending",
      "attempt": 1
    }
  }
}
```

---

## Schedules

### List
```
GET /workflows/{workflow_id}/schedules
```

Response `200`:
```json
{
  "data": [
    {
      "id": "uuid",
      "workflow_id": "uuid",
      "name": "string",
      "cron": "0 9 * * *",
      "timezone": "Asia/Tokyo",
      "input": {},
      "enabled": true,
      "next_run_at": "ISO8601",
      "created_at": "ISO8601"
    }
  ]
}
```

### Create
```
POST /workflows/{workflow_id}/schedules
```

Request:
```json
{
  "name": "string (required)",
  "cron": "0 9 * * * (required)",
  "timezone": "Asia/Tokyo",
  "input": {},
  "enabled": true,
  "retry_policy": {
    "max_attempts": 3,
    "delay_seconds": 60
  }
}
```

Response `201`: Created schedule

### Update
```
PUT /schedules/{schedule_id}
```

Response `200`: Updated schedule

### Delete
```
DELETE /schedules/{schedule_id}
```

Response `204`: No content

---

## Webhooks

### Create
```
POST /workflows/{workflow_id}/webhooks
```

Request:
```json
{
  "name": "string (required)",
  "input_mapping": {
    "event": "$.action",
    "repo": "$.repository.name"
  }
}
```

Response `201`:
```json
{
  "id": "uuid",
  "workflow_id": "uuid",
  "name": "string",
  "url": "https://api.example.com/webhooks/{id}",
  "secret": "whsec_xxx",
  "input_mapping": {},
  "created_at": "ISO8601"
}
```

### Receive (External)
```
POST /webhooks/{webhook_id}
```

Headers:
| Header | Required | Description |
|--------|----------|-------------|
| `X-Webhook-Signature` | Yes | `sha256=<hmac>` |
| `X-Webhook-Timestamp` | Yes | Unix timestamp |
| `X-Idempotency-Key` | No | Deduplication key |

Request: Any JSON payload

Response `200`:
```json
{
  "run_id": "uuid",
  "status": "pending"
}
```

---

## Adapters

### List
```
GET /adapters
```

Response `200`:
```json
{
  "data": [
    {
      "id": "mock",
      "name": "Mock Adapter",
      "description": "string",
      "input_schema": {},
      "output_schema": {}
    }
  ]
}
```

---

## Audit Logs

### List
```
GET /audit-logs
```

Query:
| Param | Type | Description |
|-------|------|-------------|
| `action` | string | `create`, `update`, `delete`, `publish`, `execute` |
| `resource_type` | string | `workflow`, `run`, `secret` |
| `actor_id` | uuid | User ID |
| `from` | ISO8601 | Start time |
| `to` | ISO8601 | End time |
| `page` | int | Page number |
| `limit` | int | Items per page |

Response `200`:
```json
{
  "data": [
    {
      "id": "uuid",
      "action": "publish",
      "resource_type": "workflow",
      "resource_id": "uuid",
      "actor_id": "uuid",
      "actor_email": "user@example.com",
      "metadata": {},
      "created_at": "ISO8601"
    }
  ],
  "pagination": {}
}
```

---

## Usage & Cost Tracking

### Get Usage Summary
```
GET /usage/summary
```

Query:
| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `period` | string | `month` | `day`, `week`, `month` |

Response `200`:
```json
{
  "data": {
    "period": "month",
    "start_date": "2025-01-01T00:00:00Z",
    "end_date": "2025-01-31T23:59:59Z",
    "total_requests": 1500,
    "total_input_tokens": 500000,
    "total_output_tokens": 200000,
    "total_cost_usd": 15.50,
    "success_rate": 0.98,
    "avg_latency_ms": 850
  }
}
```

### Get Daily Usage
```
GET /usage/daily
```

Query:
| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `start` | ISO8601 | Yes | Start date |
| `end` | ISO8601 | Yes | End date |

Response `200`:
```json
{
  "data": [
    {
      "date": "2025-01-15",
      "total_requests": 150,
      "total_input_tokens": 50000,
      "total_output_tokens": 20000,
      "total_cost_usd": 1.55,
      "provider": "openai",
      "model": "gpt-4o"
    }
  ]
}
```

### Get Usage by Workflow
```
GET /usage/by-workflow
```

Query:
| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `period` | string | `month` | `day`, `week`, `month` |

Response `200`:
```json
{
  "data": [
    {
      "workflow_id": "uuid",
      "workflow_name": "My Workflow",
      "total_requests": 500,
      "total_tokens": 150000,
      "total_cost_usd": 5.25
    }
  ]
}
```

### Get Usage by Model
```
GET /usage/by-model
```

Query:
| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `period` | string | `month` | `day`, `week`, `month` |

Response `200`:
```json
{
  "data": [
    {
      "provider": "openai",
      "model": "gpt-4o",
      "total_requests": 800,
      "total_input_tokens": 300000,
      "total_output_tokens": 100000,
      "total_cost_usd": 10.00,
      "avg_latency_ms": 750
    }
  ]
}
```

### Get Run Usage
```
GET /runs/{run_id}/usage
```

Response `200`:
```json
{
  "data": [
    {
      "id": "uuid",
      "step_run_id": "uuid",
      "provider": "openai",
      "model": "gpt-4o",
      "operation": "chat",
      "input_tokens": 1000,
      "output_tokens": 500,
      "total_tokens": 1500,
      "input_cost_usd": 0.0025,
      "output_cost_usd": 0.005,
      "total_cost_usd": 0.0075,
      "latency_ms": 850,
      "success": true,
      "created_at": "ISO8601"
    }
  ]
}
```

### List Budgets
```
GET /usage/budgets
```

Response `200`:
```json
{
  "data": [
    {
      "id": "uuid",
      "workflow_id": null,
      "budget_type": "monthly",
      "budget_amount_usd": 100.00,
      "alert_threshold": 0.80,
      "enabled": true,
      "created_at": "ISO8601",
      "updated_at": "ISO8601"
    }
  ]
}
```

### Create Budget
```
POST /usage/budgets
```

Request:
```json
{
  "workflow_id": "uuid (optional)",
  "budget_type": "monthly|daily",
  "budget_amount_usd": 100.00,
  "alert_threshold": 0.80
}
```

Response `201`: Created budget

### Update Budget
```
PUT /usage/budgets/{id}
```

Request:
```json
{
  "budget_amount_usd": 150.00,
  "alert_threshold": 0.90,
  "enabled": true
}
```

Response `200`: Updated budget

### Delete Budget
```
DELETE /usage/budgets/{id}
```

Response `204`: No content

### Get Model Pricing
```
GET /usage/pricing
```

Response `200`:
```json
{
  "data": [
    {
      "provider": "openai",
      "model": "gpt-4o",
      "input_cost_per_1k": 0.0025,
      "output_cost_per_1k": 0.01
    },
    {
      "provider": "anthropic",
      "model": "claude-3-opus",
      "input_cost_per_1k": 0.015,
      "output_cost_per_1k": 0.075
    }
  ]
}
```

---

## Admin - System Blocks

管理者専用APIエンドポイント。システムブロックの編集・バージョン管理を行う。

### List System Blocks
```
GET /admin/blocks
```

Response `200`:
```json
{
  "blocks": [
    {
      "id": "uuid",
      "slug": "llm",
      "name": "LLM Call",
      "description": "LLM APIを呼び出す",
      "category": "ai",
      "code": "const response = await ctx.llm.chat(...)",
      "config_schema": {},
      "input_schema": {},
      "output_schema": {},
      "ui_config": {"icon": "brain", "color": "#8B5CF6"},
      "is_system": true,
      "version": 3,
      "enabled": true,
      "created_at": "ISO8601",
      "updated_at": "ISO8601"
    }
  ]
}
```

### Get System Block
```
GET /admin/blocks/{id}
```

Response `200`: System block details

### Update System Block
```
PUT /admin/blocks/{id}
```

Request:
```json
{
  "name": "LLM Call",
  "description": "LLM APIを呼び出す",
  "code": "const response = await ctx.llm.chat(...)",
  "config_schema": {},
  "input_schema": {},
  "output_schema": {},
  "ui_config": {"icon": "brain", "color": "#8B5CF6"},
  "change_summary": "プロンプト処理ロジックを改善"
}
```

Response `200`: Updated block (version incremented)

### List Block Versions
```
GET /admin/blocks/{id}/versions
```

Response `200`:
```json
{
  "versions": [
    {
      "id": "uuid",
      "block_id": "uuid",
      "version": 2,
      "code": "...",
      "config_schema": {},
      "input_schema": {},
      "output_schema": {},
      "ui_config": {},
      "change_summary": "バグ修正",
      "changed_by": "uuid",
      "created_at": "ISO8601"
    }
  ]
}
```

### Get Block Version
```
GET /admin/blocks/{id}/versions/{version}
```

Response `200`: Specific version details

### Rollback Block
```
POST /admin/blocks/{id}/rollback
```

Request:
```json
{
  "version": 2
}
```

Response `200`: Block restored to specified version (new version created)

---

## Health

### Liveness
```
GET /health
```

Response `200`:
```json
{
  "status": "ok"
}
```

### Readiness
```
GET /ready
```

Response `200`:
```json
{
  "status": "ok",
  "components": {
    "database": "ok",
    "redis": "ok"
  }
}
```

Response `503` (unhealthy):
```json
{
  "status": "error",
  "components": {
    "database": "error",
    "redis": "ok"
  }
}
```

---

## cURL Examples

### Create Workflow
```bash
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"name": "Test Workflow"}'
```

### Add Step
```bash
curl -X POST "http://localhost:8080/api/v1/workflows/{id}/steps" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{
    "name": "Step 1",
    "type": "tool",
    "config": {"adapter_id": "mock", "response": {"result": "ok"}}
  }'
```

### Execute Workflow
```bash
curl -X POST "http://localhost:8080/api/v1/workflows/{id}/runs" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"input": {"message": "Hello"}, "mode": "test"}'
```

### With JWT Auth
```bash
# Get token
TOKEN=$(curl -s -X POST http://localhost:8180/realms/ai-orchestration/protocol/openid-connect/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "username=admin@example.com&password=admin123&grant_type=password&client_id=frontend" \
  | jq -r .access_token)

# Use token
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/workflows
```

## Related Documents

- [BACKEND.md](./BACKEND.md) - Backend code structure and handlers
- [DATABASE.md](./DATABASE.md) - Database schema
- [openapi.yaml](./openapi.yaml) - Machine-readable OpenAPI spec
- [DEPLOYMENT.md](./DEPLOYMENT.md) - Environment and authentication setup
