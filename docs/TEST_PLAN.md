# Test Plan - AI Orchestration

> **Last Updated**: 2026-01-12
> **Status**: Active

このドキュメントはAI Orchestrationプロジェクトのテスト計画と、テストカバレッジを維持するためのルールを定義します。

## Table of Contents

1. [Current Coverage Analysis](#current-coverage-analysis)
2. [Test Strategy](#test-strategy)
3. [Implementation Phases](#implementation-phases)
4. [Coverage Maintenance Rules](#coverage-maintenance-rules)
5. [Test Templates](#test-templates)
6. [Related Documents](#related-documents)

---

## Current Coverage Analysis

### Summary (2026-01-12時点)

| Area | Test Files | Coverage | Status |
|------|-----------|----------|--------|
| **Backend** | 17 files | ~20% | Partial |
| **Frontend** | 1 file | <1% | Critical |
| **E2E** | 1 file | ~25% | Critical |

### Backend Coverage by Package

| Package | Source Files | Test Files | Coverage | Priority |
|---------|:------------:|:----------:|:--------:|:--------:|
| handler | 17 | 0 | 0% | **CRITICAL** |
| repository/postgres | 18 | 0 | 0% | **CRITICAL** |
| usecase | 14 | 1 | 7% | HIGH |
| domain | 22 | 6 | 27% | OK |
| adapter | 9 | 4 | 44% | OK |
| engine | 7 | 2 | 29% | OK |
| middleware | 3 | 1 | 33% | OK |
| block/sandbox | 3 | 1 | 33% | OK |
| pkg/crypto | 6 | 1 | 17% | MEDIUM |

### Frontend Coverage

| Area | Files | Test Files | Coverage | Priority |
|------|:-----:|:----------:|:--------:|:--------:|
| composables | 16 | 1 | 6% | **CRITICAL** |
| components | 50+ | 0 | 0% | HIGH |
| pages | 15 | 0 | 0% | HIGH |

### Critical Untested Files

#### Backend (Highest Impact)

| File | LOC | Impact | Risk |
|------|-----|--------|------|
| `handler/copilot.go` | 778 | AI co-pilot endpoints | HIGH |
| `handler/admin_tenant.go` | 489 | Multi-tenant admin | CRITICAL |
| `handler/workflow.go` | 462 | Core workflow CRUD | CRITICAL |
| `usecase/copilot.go` | 36.5KB | AI integration logic | HIGH |
| `usecase/run.go` | 15.9KB | Workflow execution | CRITICAL |
| `repository/postgres/usage.go` | 576 | Usage tracking | HIGH |
| `repository/postgres/tenant.go` | 517 | Tenant isolation | CRITICAL |

#### Frontend (Highest Impact)

| File | LOC | Impact | Risk |
|------|-----|--------|------|
| `composables/useAuth.ts` | 216 | Authentication | CRITICAL |
| `composables/useApi.ts` | 76 | API client | CRITICAL |
| `composables/useWorkflows.ts` | 181 | Workflow CRUD | HIGH |
| `components/dag-editor/DagEditor.vue` | 2,987 | Visual editor | HIGH |

---

## Test Strategy

### Testing Pyramid

```
        /\
       /  \     E2E Tests (10%)
      /----\    - Critical user flows
     /      \   - Cross-service integration
    /--------\
   /          \ Integration Tests (30%)
  /            \ - Handler + Usecase + Repository
 /--------------\
/                \ Unit Tests (60%)
/                  \ - Domain, Usecase, Adapter logic
```

### Test Types

| Type | Scope | Tools | Location |
|------|-------|-------|----------|
| Unit | Single function/method | testify, vitest | `*_test.go`, `__tests__/*.spec.ts` |
| Integration | Multiple components | testify + DB | `tests/integration/` |
| E2E | Full system | testify + HTTP | `tests/e2e/` |
| Frontend Unit | Component/Composable | vitest, vue-test-utils | `__tests__/*.spec.ts` |

### Coverage Targets

| Area | Minimum | Target | Stretch |
|------|---------|--------|---------|
| Backend Unit | 50% | 70% | 85% |
| Backend Integration | 30% | 50% | 70% |
| Frontend Unit | 40% | 60% | 80% |
| E2E Flows | 60% | 80% | 90% |

---

## Implementation Phases

### Phase 1: Critical Foundation (Week 1-2)

**Goal**: 最も影響の大きい領域のテストを追加

#### Backend Handler Tests

| File | Tests to Add | Priority |
|------|--------------|----------|
| `handler/workflow.go` | CRUD validation, auth, errors | P0 |
| `handler/admin_tenant.go` | Tenant operations, isolation | P0 |
| `handler/run.go` | Run lifecycle, state transitions | P0 |
| `handler/step.go` | Step CRUD, type validation | P1 |
| `handler/edge.go` | Edge creation, DAG validation | P1 |

**Test Cases for workflow_handler_test.go:**
```
- TestCreateWorkflow_Success
- TestCreateWorkflow_InvalidJSON
- TestCreateWorkflow_MissingTenantID
- TestCreateWorkflow_EmptyName
- TestGetWorkflow_Success
- TestGetWorkflow_NotFound
- TestGetWorkflow_WrongTenant (isolation)
- TestUpdateWorkflow_Success
- TestUpdateWorkflow_AlreadyPublished
- TestDeleteWorkflow_Success
- TestDeleteWorkflow_HasRuns
- TestPublishWorkflow_Success
- TestPublishWorkflow_InvalidDAG
```

#### Frontend Core Composables

| File | Tests to Add | Priority |
|------|--------------|----------|
| `useAuth.ts` | Auth flow, token refresh, dev mode | P0 |
| `useApi.ts` | HTTP methods, error handling, retry | P0 |
| `useWorkflows.ts` | CRUD operations, state management | P1 |
| `useRuns.ts` | Run polling, state updates | P1 |

### Phase 2: Business Logic (Week 3-4)

**Goal**: ビジネスロジック層のテストカバレッジ向上

#### Backend Usecase Tests

| File | Tests to Add | Priority |
|------|--------------|----------|
| `usecase/run.go` | Execution orchestration, state machine | P0 |
| `usecase/copilot.go` | AI suggestions, error handling | P0 |
| `usecase/credential_resolver.go` | Credential resolution, secrets | P1 |
| `usecase/schedule.go` | Schedule management, triggers | P1 |
| `usecase/webhook.go` | Webhook handling, validation | P1 |

#### Backend Repository Tests

| File | Tests to Add | Priority |
|------|--------------|----------|
| `repository/postgres/tenant.go` | Tenant isolation, CRUD | P0 |
| `repository/postgres/workflow.go` | Workflow persistence, versioning | P0 |
| `repository/postgres/run.go` | Run state management | P0 |
| `repository/postgres/usage.go` | Usage tracking, aggregation | P1 |

### Phase 3: UI Components (Week 5-6)

**Goal**: フロントエンドコンポーネントのテスト追加

#### Critical Components

| Component | Tests to Add | Priority |
|-----------|--------------|----------|
| `DagEditor.vue` | Node operations, edge connections | P0 |
| `PropertiesPanel.vue` | Form validation, config updates | P1 |
| `DynamicConfigForm.vue` | Schema rendering, validation | P1 |
| `RunViewer.vue` | State display, log rendering | P2 |

### Phase 4: E2E Coverage (Week 7-8)

**Goal**: E2Eテストでクリティカルフローを網羅

#### E2E Test Scenarios

| Scenario | Description | Priority |
|----------|-------------|----------|
| Workflow CRUD | Create, update, publish, delete | P0 |
| Run Lifecycle | Execute, monitor, cancel, retry | P0 |
| LLM Step Execution | OpenAI/Anthropic integration | P0 |
| Map/Join Patterns | Parallel execution, merge | P1 |
| Block Groups | Group creation, nesting | P1 |
| Multi-tenant Isolation | Cross-tenant access prevention | P0 |
| Error Recovery | Failed step handling, retry | P1 |
| Credentials | Secret resolution, masking | P1 |

---

## Coverage Maintenance Rules

### Rule 1: 新規コード = 新規テスト (Mandatory)

**新しいコードを追加する場合、必ず対応するテストを追加すること。**

| 追加するコード | 必要なテスト |
|--------------|-------------|
| 新規Handler | Handler unit tests + request validation |
| 新規Usecase | Usecase unit tests + edge cases |
| 新規Repository | Repository tests (DB mock or test container) |
| 新規Adapter | Adapter tests with mock external service |
| 新規Composable | Composable unit tests |
| 新規Component | Component tests (mount, props, events) |

### Rule 2: バグ修正 = 回帰テスト (Mandatory)

**バグを修正する場合、必ず回帰テストを追加すること。**

```
1. バグを再現するテストを書く（失敗することを確認）
2. コードを修正
3. テストがパスすることを確認
4. 関連するエッジケースのテストも追加
```

### Rule 3: カバレッジ閾値 (CI/CD)

以下の閾値を下回る変更はマージをブロックする：

| Area | Minimum Threshold |
|------|------------------|
| Backend Unit | 50% (新規ファイル: 70%) |
| Frontend Unit | 40% (新規ファイル: 60%) |
| E2E Critical Paths | 80% |

### Rule 4: テストファイル命名規則

| Language | Pattern | Example |
|----------|---------|---------|
| Go | `{name}_test.go` | `workflow_test.go` |
| TypeScript | `{name}.spec.ts` | `useAuth.spec.ts` |
| E2E (Go) | `{feature}_test.go` | `workflow_e2e_test.go` |

### Rule 5: テストの品質基準

#### 良いテストの条件

| 基準 | 説明 |
|------|------|
| 独立性 | 他のテストに依存しない |
| 再現性 | 何度実行しても同じ結果 |
| 高速 | 単体テストは100ms以内 |
| 明確性 | テスト名から目的がわかる |
| 単一責任 | 1テスト = 1アサーション（原則） |

#### テストケースの命名規則

```go
// Go: Test{Function}_{Scenario}_{ExpectedResult}
func TestCreateWorkflow_EmptyName_ReturnsValidationError(t *testing.T)
func TestExecuteStep_ConditionTrue_FollowsTrueBranch(t *testing.T)
```

```typescript
// TypeScript: describe('function') + it('should...')
describe('useAuth', () => {
  it('should refresh token before expiry', async () => {...})
  it('should use dev mode when AUTH_ENABLED is false', async () => {...})
})
```

### Rule 6: モック使用ガイドライン

| 何をモック | 何をモックしない |
|----------|----------------|
| 外部API（OpenAI, Anthropic） | ドメインロジック |
| データベース接続 | ビジネスルール |
| 時間（time.Now()） | 計算ロジック |
| ファイルシステム | データ変換 |
| 環境変数 | バリデーション |

### Rule 7: テスト実行タイミング

| タイミング | 実行するテスト |
|-----------|--------------|
| コード保存時 | 関連するユニットテスト |
| コミット前 | 全ユニットテスト |
| PR作成時 | 全ユニット + 統合テスト |
| マージ前 | 全テスト + E2E |
| デプロイ前 | 全テスト + E2E + スモークテスト |

---

## Test Templates

### Go Handler Test Template

```go
package handler_test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestWorkflowHandler_Create(t *testing.T) {
    tests := []struct {
        name           string
        requestBody    interface{}
        tenantID       string
        expectedStatus int
        expectedError  string
    }{
        {
            name:           "success",
            requestBody:    map[string]interface{}{"name": "Test Workflow"},
            tenantID:       "tenant-1",
            expectedStatus: http.StatusCreated,
        },
        {
            name:           "missing tenant ID",
            requestBody:    map[string]interface{}{"name": "Test"},
            tenantID:       "",
            expectedStatus: http.StatusBadRequest,
            expectedError:  "tenant_id required",
        },
        {
            name:           "empty name",
            requestBody:    map[string]interface{}{"name": ""},
            tenantID:       "tenant-1",
            expectedStatus: http.StatusBadRequest,
            expectedError:  "name is required",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup
            handler := setupTestHandler(t)
            body, _ := json.Marshal(tt.requestBody)
            req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows", bytes.NewReader(body))
            req.Header.Set("Content-Type", "application/json")
            if tt.tenantID != "" {
                req.Header.Set("X-Tenant-ID", tt.tenantID)
            }
            rec := httptest.NewRecorder()

            // Execute
            handler.CreateWorkflow(rec, req)

            // Assert
            assert.Equal(t, tt.expectedStatus, rec.Code)
            if tt.expectedError != "" {
                var resp map[string]interface{}
                json.Unmarshal(rec.Body.Bytes(), &resp)
                assert.Contains(t, resp["error"], tt.expectedError)
            }
        })
    }
}
```

### Go Repository Test Template

```go
package repository_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestWorkflowRepository_Create(t *testing.T) {
    // Setup test database (use test container or mock)
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)
    repo := postgres.NewWorkflowRepository(db)
    ctx := context.Background()

    t.Run("creates workflow successfully", func(t *testing.T) {
        workflow := &domain.Workflow{
            TenantID: "tenant-1",
            Name:     "Test Workflow",
        }

        err := repo.Create(ctx, workflow)

        require.NoError(t, err)
        assert.NotEmpty(t, workflow.ID)
        assert.NotZero(t, workflow.CreatedAt)
    })

    t.Run("enforces tenant isolation", func(t *testing.T) {
        // Create in tenant-1
        workflow := &domain.Workflow{TenantID: "tenant-1", Name: "Private"}
        repo.Create(ctx, workflow)

        // Try to access from tenant-2
        ctx2 := context.WithValue(ctx, "tenant_id", "tenant-2")
        _, err := repo.GetByID(ctx2, workflow.ID)

        assert.Error(t, err)
        assert.Contains(t, err.Error(), "not found")
    })
}
```

### TypeScript Composable Test Template

```typescript
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

// Mock dependencies
vi.mock('~/composables/useApi', () => ({
  useApi: () => ({
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  }),
}))

describe('useWorkflows', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  describe('fetchWorkflows', () => {
    it('should fetch and store workflows', async () => {
      const mockWorkflows = [
        { id: '1', name: 'Workflow 1' },
        { id: '2', name: 'Workflow 2' },
      ]
      const { useApi } = await import('~/composables/useApi')
      vi.mocked(useApi().get).mockResolvedValue({ data: mockWorkflows })

      const { workflows, fetchWorkflows } = useWorkflows()
      await fetchWorkflows()

      expect(workflows.value).toEqual(mockWorkflows)
    })

    it('should handle fetch error gracefully', async () => {
      const { useApi } = await import('~/composables/useApi')
      vi.mocked(useApi().get).mockRejectedValue(new Error('Network error'))

      const { error, fetchWorkflows } = useWorkflows()
      await fetchWorkflows()

      expect(error.value).toBeTruthy()
    })
  })
})
```

### E2E Test Template

```go
package e2e_test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestWorkflowE2E_CreateAndExecute(t *testing.T) {
    client := setupE2EClient(t)
    tenantID := "test-tenant"

    // 1. Create workflow
    createResp, err := client.Post("/api/v1/workflows", map[string]interface{}{
        "name": "E2E Test Workflow",
    }, tenantID)
    require.NoError(t, err)
    assert.Equal(t, http.StatusCreated, createResp.StatusCode)

    var workflow struct {
        ID string `json:"id"`
    }
    json.NewDecoder(createResp.Body).Decode(&workflow)

    // 2. Add start step
    _, err = client.Post("/api/v1/workflows/"+workflow.ID+"/steps", map[string]interface{}{
        "name": "Start",
        "type": "start",
    }, tenantID)
    require.NoError(t, err)

    // 3. Add tool step
    // ... add more steps

    // 4. Publish workflow
    _, err = client.Post("/api/v1/workflows/"+workflow.ID+"/publish", nil, tenantID)
    require.NoError(t, err)

    // 5. Execute workflow
    runResp, err := client.Post("/api/v1/workflows/"+workflow.ID+"/runs", map[string]interface{}{
        "input": map[string]interface{}{"test": "data"},
        "mode":  "test",
    }, tenantID)
    require.NoError(t, err)
    assert.Equal(t, http.StatusCreated, runResp.StatusCode)

    // 6. Wait for completion and verify
    // ...
}
```

---

## CI/CD Integration

### GitHub Actions Workflow

```yaml
# .github/workflows/test.yml
name: Tests

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  backend-test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16
        env:
          POSTGRES_DB: test
          POSTGRES_USER: test
          POSTGRES_PASSWORD: test
        ports:
          - 5432:5432
      redis:
        image: redis:7
        ports:
          - 6379:6379

    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Run unit tests
        run: |
          cd backend
          go test -v -race -coverprofile=coverage.out ./...

      - name: Check coverage threshold
        run: |
          cd backend
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          if (( $(echo "$COVERAGE < 50" | bc -l) )); then
            echo "Coverage $COVERAGE% is below 50% threshold"
            exit 1
          fi

  frontend-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'

      - name: Install dependencies
        run: cd frontend && npm ci

      - name: Run checks
        run: cd frontend && npm run check

      - name: Check coverage
        run: |
          cd frontend
          npm run test:coverage
          # Parse and verify coverage threshold

  e2e-test:
    runs-on: ubuntu-latest
    needs: [backend-test, frontend-test]
    steps:
      - uses: actions/checkout@v4
      - name: Start services
        run: docker compose up -d
      - name: Wait for services
        run: sleep 30
      - name: Run E2E tests
        run: |
          cd backend
          go test -v ./tests/e2e/...
```

---

## Related Documents

| Document | Description |
|----------|-------------|
| [CLAUDE.md](../CLAUDE.md) | プロジェクトルール全般 |
| [BACKEND.md](./BACKEND.md) | バックエンド構造 |
| [FRONTEND.md](./FRONTEND.md) | フロントエンド構造 |
| [frontend/docs/TESTING.md](../frontend/docs/TESTING.md) | フロントエンドテストルール |
| [DOCUMENTATION_RULES.md](./DOCUMENTATION_RULES.md) | ドキュメント規約 |
