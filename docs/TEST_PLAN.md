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
| **Backend** | 20 files | ~30% | Improving |
| **Frontend** | 3 files | ~10% | Improving |
| **E2E** | 1 file | ~25% | Critical |

### Phase 1 完了項目 (2026-01-12)

| 追加したテスト | テスト数 | ファイル |
|--------------|---------|---------|
| Backend Handler (workflow) | 47 | `handler/workflow_test.go` |
| Backend Repository (workflow) | 20 | `repository/postgres/workflow_test.go` |
| Frontend useApi | 14 | `composables/__tests__/useApi.spec.ts` |
| Frontend useAuth | 9 | `composables/__tests__/useAuth.spec.ts` |
| **合計** | **90** | |

### Backend Coverage by Package

| Package | Source Files | Test Files | Coverage | Priority |
|---------|:------------:|:----------:|:--------:|:--------:|
| handler | 17 | 1 | 6% | **CRITICAL** |
| repository/postgres | 18 | 1 | 6% | **CRITICAL** |
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
| composables | 16 | 3 | 19% | HIGH |
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

### Phase 1: Critical Foundation ✅ COMPLETED (2026-01-12)

**Goal**: 最も影響の大きい領域のテストを追加

#### 完了項目

| File | 追加したテスト | Status |
|------|--------------|--------|
| `handler/workflow_test.go` | 47 tests (CRUD validation, auth, errors) | ✅ Done |
| `repository/postgres/workflow_test.go` | 20 tests (CRUD, tenant isolation) | ✅ Done |
| `composables/__tests__/useApi.spec.ts` | 14 tests (HTTP methods, headers) | ✅ Done |
| `composables/__tests__/useAuth.spec.ts` | 9 tests (roles, dev mode) | ✅ Done |

#### 追加した設計改善

- `backend/internal/repository/postgres/db.go` - DB interface for testability
- pgxmock dependency for mock database testing

---

### Phase 2: Business Logic (次フェーズ)

**Goal**: ビジネスロジック層のテストカバレッジ向上

#### 2.1 Backend Handler Tests (残り)

| File | LOC | 追加テスト | 見積テスト数 | Priority |
|------|-----|-----------|-------------|----------|
| `handler/run.go` | 462 | Run lifecycle, state transitions | 30-40 | P0 |
| `handler/admin_tenant.go` | 489 | Tenant operations, isolation | 25-35 | P0 |
| `handler/step.go` | 334 | Step CRUD, type validation | 20-30 | P1 |
| `handler/edge.go` | 205 | Edge creation, validation | 15-20 | P1 |
| `handler/copilot.go` | 778 | AI endpoints, streaming | 30-40 | P1 |

**Test Cases for run_handler_test.go:**
```
- TestCreateRun_Success
- TestCreateRun_WorkflowNotPublished
- TestCreateRun_InvalidInput
- TestGetRun_Success
- TestGetRun_NotFound
- TestGetRun_WrongTenant
- TestListRuns_Pagination
- TestListRuns_FilterByStatus
- TestCancelRun_Success
- TestCancelRun_AlreadyCompleted
- TestRetryRun_Success
- TestRetryRun_NotFailed
```

#### 2.2 Backend Repository Tests (残り)

| File | LOC | 追加テスト | 見積テスト数 | Priority |
|------|-----|-----------|-------------|----------|
| `repository/postgres/run.go` | 624 | Run state management | 25-35 | P0 |
| `repository/postgres/tenant.go` | 517 | Tenant isolation, CRUD | 20-30 | P0 |
| `repository/postgres/step.go` | 358 | Step persistence | 15-20 | P1 |
| `repository/postgres/edge.go` | 187 | Edge persistence | 10-15 | P1 |
| `repository/postgres/usage.go` | 576 | Usage tracking | 20-25 | P2 |

**Implementation Note:**
- 既存の `db.go` DB interface を他の repository にも適用
- `NewXxxRepositoryWithDB()` constructor を追加してテスト可能に

#### 2.3 Backend Usecase Tests

| File | LOC | 追加テスト | 見積テスト数 | Priority |
|------|-----|-----------|-------------|----------|
| `usecase/run.go` | 15.9KB | Execution orchestration | 40-50 | P0 |
| `usecase/workflow.go` | 582 | Workflow business logic | 20-25 | P1 |
| `usecase/credential_resolver.go` | 286 | Secret resolution | 15-20 | P1 |
| `usecase/schedule.go` | 201 | Schedule management | 10-15 | P2 |
| `usecase/webhook.go` | 246 | Webhook handling | 10-15 | P2 |

**Test Approach:**
- Repository を interface 化してモック注入
- domain error mapping のテスト
- エッジケース（同時実行、タイムアウト等）

---

### Phase 3: Frontend Components

**Goal**: フロントエンドのテストカバレッジ向上

#### 3.1 Composable Tests (残り)

| File | LOC | 追加テスト | 見積テスト数 | Priority |
|------|-----|-----------|-------------|----------|
| `useWorkflows.ts` | 181 | CRUD operations, state | 15-20 | P0 |
| `useRuns.ts` | 270 | Polling, state updates | 20-25 | P0 |
| `useDagEditor.ts` | 356 | Node/edge operations | 25-30 | P1 |
| `useSteps.ts` | 125 | Step CRUD | 10-15 | P1 |
| `useBlocks.ts` | 82 | Block definitions | 8-12 | P2 |

**Test Cases for useWorkflows.spec.ts:**
```typescript
describe('useWorkflows', () => {
  describe('fetchWorkflows', () => {
    it('should fetch and cache workflows')
    it('should handle pagination')
    it('should handle network errors')
  })
  describe('createWorkflow', () => {
    it('should create and add to list')
    it('should validate input')
  })
  describe('updateWorkflow', () => {
    it('should update and refresh')
    it('should handle optimistic update')
  })
  describe('deleteWorkflow', () => {
    it('should remove from list')
    it('should handle deletion errors')
  })
})
```

#### 3.2 Component Tests

| Component | LOC | 追加テスト | 見積テスト数 | Priority |
|-----------|-----|-----------|-------------|----------|
| `DagEditor.vue` | 2,987 | Node/edge operations | 40-50 | P0 |
| `PropertiesPanel.vue` | 486 | Form validation | 15-20 | P1 |
| `DynamicConfigForm.vue` | 389 | Schema rendering | 15-20 | P1 |
| `BlockPalette.vue` | 298 | Block drag/drop | 10-15 | P2 |
| `RunViewer.vue` | 521 | State display, logs | 15-20 | P2 |

**Test Approach (DagEditor.vue):**
```typescript
describe('DagEditor', () => {
  describe('node operations', () => {
    it('should add node on drop')
    it('should remove node on delete key')
    it('should update node position on drag')
    it('should select node on click')
  })
  describe('edge operations', () => {
    it('should create edge between valid ports')
    it('should prevent invalid connections')
    it('should remove edge on delete')
  })
  describe('group operations', () => {
    it('should create group from selection')
    it('should resize group with contents')
    it('should push blocks on boundary collision')
  })
})
```

---

### Phase 4: E2E & Integration Tests

**Goal**: E2Eテストでクリティカルフローを網羅

#### 4.1 E2E Test Scenarios

| Scenario | Description | 見積テスト数 | Priority |
|----------|-------------|-------------|----------|
| Workflow CRUD | Create → Edit → Publish → Delete | 5-8 | P0 |
| Run Lifecycle | Execute → Monitor → Complete/Cancel | 8-12 | P0 |
| LLM Integration | OpenAI/Anthropic step execution | 5-8 | P0 |
| Multi-tenant Isolation | Cross-tenant access prevention | 5-8 | P0 |
| Map/Join Patterns | Parallel execution, merge | 5-8 | P1 |
| Block Groups | Group creation, nesting | 5-8 | P1 |
| Error Recovery | Failed step → Retry → Success | 5-8 | P1 |
| Credentials | Secret resolution, masking | 5-8 | P2 |

#### 4.2 Integration Tests

| Area | 追加テスト | 見積テスト数 | Priority |
|------|-----------|-------------|----------|
| Handler + Usecase + Repository | Full request/response flow | 30-40 | P0 |
| Engine + Adapter | Step execution with mock LLM | 20-30 | P1 |
| Webhook + Scheduler | Trigger mechanisms | 15-20 | P2 |

---

## Phase 実行サマリー

| Phase | 主要タスク | 見積テスト数 | 所要時間目安 |
|-------|-----------|-------------|-------------|
| Phase 1 | ✅ Handler, Repository, Composable 基盤 | 90 | 完了 |
| Phase 2 | Handler残り, Repository残り, Usecase | 200-250 | 2-3日 |
| Phase 3 | Composable残り, Component | 150-200 | 2-3日 |
| Phase 4 | E2E, Integration | 100-150 | 2-3日 |
| **合計** | | **540-690** | **6-9日** |

### 推奨実行順序

```
Phase 2.1 → Phase 2.2 → Phase 2.3 → Phase 3.1 → Phase 3.2 → Phase 4.1 → Phase 4.2
 (Handler)   (Repo)      (Usecase)   (Composable) (Component)  (E2E)     (Integration)
```

### 各Phaseの開始条件

| Phase | 開始条件 |
|-------|---------|
| Phase 2 | 即時開始可能 |
| Phase 3 | Phase 2.1完了後（並行可） |
| Phase 4 | Phase 2, 3 の主要部分完了後 |

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
