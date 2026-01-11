# API Reference

## Base

```
URL: /api/v1
Auth: Bearer <JWT>
Content-Type: application/json
Tenant: X-Tenant-ID header (dev mode) or JWT claim (prod)
```

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
  "mode": "test|production"
}
```

Response `201`:
```json
{
  "id": "uuid",
  "workflow_id": "uuid",
  "workflow_version": 1,
  "status": "pending",
  "mode": "test|production",
  "trigger_type": "manual",
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

### Resume
```
POST /runs/{run_id}/resume
```

Constraint: Must be `failed` status
Response `200`: New run from failed step

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
