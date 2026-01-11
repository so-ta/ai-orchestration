# Block Registry Design

## Overview

Block Registryはワークフローのステップタイプを動的に管理するシステムです。
ビルトインブロックとカスタムブロックの両方をサポートし、エラーコードによる統一的なエラーハンドリングを提供します。

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Block Registry                               │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌─────────────────────┐    ┌─────────────────────┐                 │
│  │   Built-in Blocks   │    │   Custom Blocks     │                 │
│  │   (Code-defined)    │    │   (DB-defined)      │                 │
│  │                     │    │                     │                 │
│  │  - llm              │    │  - slack_message    │                 │
│  │  - condition        │    │  - github_pr        │                 │
│  │  - loop             │    │  - custom_http      │                 │
│  │  - tool             │    │                     │                 │
│  └─────────────────────┘    └─────────────────────┘                 │
│            │                          │                              │
│            └──────────┬───────────────┘                              │
│                       ▼                                              │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                   BlockDefinition                            │    │
│  │  - slug (unique identifier)                                  │    │
│  │  - name, description, category, icon                         │    │
│  │  - config_schema (JSON Schema)                               │    │
│  │  - input_schema, output_schema                               │    │
│  │  - executor_type (builtin | custom)                          │    │
│  │  - executor_config                                           │    │
│  │  - error_codes (定義されたエラーコード)                        │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                       │                                              │
│                       ▼                                              │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                   BlockExecutor                              │    │
│  │  Execute(ctx, input, config) -> (output, error)              │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                       │
└─────────────────────────────────────────────────────────────────────┘
```

## Data Model

### BlockDefinition

```go
type BlockDefinition struct {
    ID             uuid.UUID       // Unique ID
    TenantID       *uuid.UUID      // NULL = system block, otherwise tenant-specific
    Slug           string          // Unique identifier (e.g., "llm", "slack_message")
    Name           string          // Display name
    Description    string          // Block description
    Category       string          // ai, logic, integration, data, control, utility
    Icon           string          // Icon identifier

    // Schemas (JSON Schema format)
    ConfigSchema   json.RawMessage // Configuration options for the block
    InputSchema    json.RawMessage // Expected input structure
    OutputSchema   json.RawMessage // Output structure

    // Executor
    ExecutorType   string          // "builtin" or "http" or "function"
    ExecutorConfig json.RawMessage // Executor-specific config

    // Error handling
    ErrorCodes     []ErrorCodeDef  // Defined error codes for this block

    // Metadata
    Enabled        bool
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

type ErrorCodeDef struct {
    Code        string `json:"code"`        // e.g., "LLM_001"
    Name        string `json:"name"`        // e.g., "RATE_LIMIT_EXCEEDED"
    Description string `json:"description"` // Human-readable description
    Retryable   bool   `json:"retryable"`   // Can this error be retried?
}
```

### Database Schema

```sql
CREATE TABLE block_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id),  -- NULL = system block
    slug VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(50) NOT NULL,
    icon VARCHAR(50),

    config_schema JSONB NOT NULL DEFAULT '{}',
    input_schema JSONB,
    output_schema JSONB,

    executor_type VARCHAR(20) NOT NULL,  -- builtin, http, function
    executor_config JSONB,

    error_codes JSONB DEFAULT '[]',

    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(tenant_id, slug),  -- Unique per tenant (NULL tenant = global)
    CONSTRAINT valid_category CHECK (category IN ('ai', 'logic', 'integration', 'data', 'control', 'utility'))
);

CREATE INDEX idx_block_definitions_tenant ON block_definitions(tenant_id);
CREATE INDEX idx_block_definitions_category ON block_definitions(category);
CREATE INDEX idx_block_definitions_enabled ON block_definitions(enabled);
```

## Error Code System

### Standard Error Code Format

```
{BLOCK}_{NUMBER}_{TYPE}

Examples:
- LLM_001_RATE_LIMIT     - LLM rate limit exceeded
- LLM_002_INVALID_MODEL  - Invalid model specified
- HTTP_001_TIMEOUT       - HTTP request timeout
- HTTP_002_CONN_REFUSED  - Connection refused
- COND_001_INVALID_EXPR  - Invalid condition expression
```

### BlockError Structure

```go
type BlockError struct {
    Code       string          `json:"code"`        // Error code (e.g., "LLM_001")
    Message    string          `json:"message"`     // Human-readable message
    Details    json.RawMessage `json:"details"`     // Additional error details
    Retryable  bool            `json:"retryable"`   // Can this be retried?
    RetryAfter *time.Duration  `json:"retry_after"` // Suggested retry delay
}

func (e *BlockError) Error() string {
    return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}
```

### Error Code Categories

| Category | Range | Description |
|----------|-------|-------------|
| SYSTEM   | 000-099 | System-level errors |
| CONFIG   | 100-199 | Configuration errors |
| INPUT    | 200-299 | Input validation errors |
| EXEC     | 300-399 | Execution errors |
| OUTPUT   | 400-499 | Output processing errors |
| AUTH     | 500-599 | Authentication/authorization errors |
| RATE     | 600-699 | Rate limiting errors |
| TIMEOUT  | 700-799 | Timeout errors |

## Built-in Blocks

Built-in blocks are registered at application startup:

```go
// Built-in block definitions (seeded on startup)
var BuiltinBlocks = []BlockDefinition{
    {
        Slug:         "start",
        Name:         "Start",
        Category:     "control",
        ExecutorType: "builtin",
        ErrorCodes:   []ErrorCodeDef{},
    },
    {
        Slug:         "llm",
        Name:         "LLM",
        Category:     "ai",
        ExecutorType: "builtin",
        ErrorCodes: []ErrorCodeDef{
            {Code: "LLM_001", Name: "RATE_LIMIT", Description: "Rate limit exceeded", Retryable: true},
            {Code: "LLM_002", Name: "INVALID_MODEL", Description: "Invalid model", Retryable: false},
            {Code: "LLM_003", Name: "TOKEN_LIMIT", Description: "Token limit exceeded", Retryable: false},
            {Code: "LLM_004", Name: "API_ERROR", Description: "LLM API error", Retryable: true},
        },
    },
    // ... more built-in blocks
}
```

## Custom Block Types

### 1. HTTP Block (Low-code)

UIから設定可能なHTTPリクエストブロック:

```json
{
    "slug": "my_api_call",
    "name": "My API Call",
    "category": "integration",
    "executor_type": "http",
    "executor_config": {
        "method": "POST",
        "url": "https://api.example.com/endpoint",
        "headers": {
            "Authorization": "Bearer {{secrets.api_key}}"
        },
        "body_template": "{\"data\": {{input.data}}}"
    }
}
```

### 2. Function Block (Code)

JavaScriptで実装するカスタムロジック:

```json
{
    "slug": "custom_transform",
    "name": "Custom Transform",
    "category": "data",
    "executor_type": "function",
    "executor_config": {
        "code": "return { transformed: input.data.map(x => x * 2) }",
        "language": "javascript",
        "timeout_ms": 5000
    }
}
```

### 3. Builtin Block (Go Code)

Go codeで実装されるブロック（開発者のみ）:

```go
// backend/internal/block/custom/slack.go
type SlackBlock struct{}

func (b *SlackBlock) ID() string { return "slack_message" }

func (b *SlackBlock) Execute(ctx context.Context, req *BlockRequest) (*BlockResponse, error) {
    // Implementation
}
```

## API Endpoints

### List Block Definitions

```
GET /api/v1/blocks
Query: ?category=ai&enabled=true
Response: {
    "data": [
        {
            "slug": "llm",
            "name": "LLM",
            "category": "ai",
            "config_schema": {...},
            "error_codes": [...]
        }
    ]
}
```

### Get Block Definition

```
GET /api/v1/blocks/{slug}
Response: { "data": BlockDefinition }
```

### Create Custom Block (Tenant)

```
POST /api/v1/blocks
Body: {
    "slug": "my_custom_block",
    "name": "My Custom Block",
    "category": "integration",
    "executor_type": "http",
    "executor_config": {...},
    "config_schema": {...}
}
```

### Update Custom Block

```
PUT /api/v1/blocks/{slug}
Body: { ...updates }
```

### Delete Custom Block

```
DELETE /api/v1/blocks/{slug}
```

## Frontend Integration

### Block Palette

```typescript
interface BlockDefinition {
    slug: string
    name: string
    description: string
    category: 'ai' | 'logic' | 'integration' | 'data' | 'control' | 'utility'
    icon: string
    configSchema: JSONSchema
    inputSchema: JSONSchema
    outputSchema: JSONSchema
    errorCodes: ErrorCodeDef[]
}

// Composable
function useBlocks() {
    const blocks = ref<BlockDefinition[]>([])

    async function loadBlocks() {
        const res = await api.get('/blocks')
        blocks.value = res.data
    }

    function getBlocksByCategory(category: string) {
        return blocks.value.filter(b => b.category === category)
    }

    return { blocks, loadBlocks, getBlocksByCategory }
}
```

### Dynamic Config Form

Block config formはJSON Schemaから動的に生成:

```vue
<template>
  <DynamicForm
    :schema="selectedBlock.configSchema"
    v-model="stepConfig"
  />
</template>
```

## Migration Strategy

1. **Phase 1**: Create block_definitions table with built-in blocks seeded
2. **Phase 2**: Migrate existing StepType to use block registry
3. **Phase 3**: Add custom block creation UI
4. **Phase 4**: Add HTTP/Function executor support

## Related Documents

- [BACKEND.md](BACKEND.md) - Backend architecture
- [API.md](API.md) - API documentation
- [FRONTEND.md](FRONTEND.md) - Frontend architecture
