# Testing Guide

テスト作成・実行の統合ガイド。

> **Status**: Active
> **Updated**: 2026-01-15
> **Related**: [BACKEND_TESTING.md](./BACKEND_TESTING.md), [frontend/docs/TESTING.md](../frontend/docs/TESTING.md)

---

## Quick Reference

| Environment | Command | When to Run |
|-------------|---------|-------------|
| Backend | `cd backend && go test ./...` | Go コード変更後 |
| Frontend | `cd frontend && npm run check` | TS/Vue コード変更後 |
| E2E | `cd backend && go test ./tests/e2e/... -v` | 統合テスト時 |

**コミット前の必須チェック**:
```bash
# Backend 変更時
cd backend && go test ./...

# Frontend 変更時
cd frontend && npm run check  # = typecheck + lint + test

# 両方変更時
cd backend && go test ./... && cd ../frontend && npm run check
```

---

## Testing Priority Matrix

Claude Code はこの優先度に従ってテストを作成する。

### 必須テスト対象（必ずカバー）

| 優先度 | 対象 | 理由 |
|--------|------|------|
| 1 | Domain Logic | ビジネスルールの検証 |
| 2 | Error Paths | 全エラーパスの確認 |
| 3 | Validation | 入力値の境界条件 |
| 4 | Security | 認証・認可・テナント分離 |

### 推奨テスト対象

| 優先度 | 対象 | 理由 |
|--------|------|------|
| 5 | Repository Query | 複雑な SQL の動作確認 |
| 6 | Handler Parsing | リクエストバインディング |
| 7 | API Integration | 外部 API 呼び出し |

### テスト不要

- 単純な getter/setter
- フレームワークが保証する動作
- 設定ファイルの読み込み

---

## Backend Testing (Go)

### テストファイル配置

```
backend/
├── internal/
│   ├── domain/
│   │   ├── workflow.go
│   │   └── workflow_test.go      # 同じパッケージ内
│   ├── usecase/
│   │   ├── workflow.go
│   │   └── workflow_test.go
│   └── repository/
│       └── postgres/
│           ├── workflow.go
│           └── workflow_test.go  # DB モック使用
└── tests/
    └── e2e/
        └── workflow_test.go      # 統合テスト
```

### Table-Driven Tests パターン（必須）

```go
func TestWorkflowUsecase_Create(t *testing.T) {
    tests := []struct {
        name    string
        input   *CreateWorkflowInput
        setup   func(*mockRepo)  // モックの設定
        want    *domain.Workflow
        wantErr error
    }{
        // 正常系
        {
            name:  "valid input creates workflow",
            input: &CreateWorkflowInput{Name: "Test"},
            setup: func(m *mockRepo) {
                m.On("Create", mock.Anything, mock.Anything).Return(nil)
            },
            want: &domain.Workflow{Name: "Test", Status: "draft"},
        },
        // 異常系 - 必須フィールド
        {
            name:    "empty name returns validation error",
            input:   &CreateWorkflowInput{Name: ""},
            wantErr: domain.ErrValidation,
        },
        // 異常系 - 境界値
        {
            name:  "max length name succeeds",
            input: &CreateWorkflowInput{Name: strings.Repeat("a", 255)},
            setup: func(m *mockRepo) {
                m.On("Create", mock.Anything, mock.Anything).Return(nil)
            },
            want: &domain.Workflow{Status: "draft"},
        },
        {
            name:    "over max length fails",
            input:   &CreateWorkflowInput{Name: strings.Repeat("a", 256)},
            wantErr: domain.ErrValidation,
        },
        // 異常系 - DB エラー
        {
            name:  "repository error returns error",
            input: &CreateWorkflowInput{Name: "Test"},
            setup: func(m *mockRepo) {
                m.On("Create", mock.Anything, mock.Anything).Return(errors.New("db error"))
            },
            wantErr: errors.New("create workflow: db error"),
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            repo := new(mockRepo)
            if tt.setup != nil {
                tt.setup(repo)
            }
            uc := usecase.NewWorkflowUsecase(repo)

            got, err := uc.Create(context.Background(), tenantID, tt.input)

            if tt.wantErr != nil {
                assert.Error(t, err)
                if errors.Is(tt.wantErr, domain.ErrValidation) {
                    assert.ErrorIs(t, err, domain.ErrValidation)
                }
                return
            }

            assert.NoError(t, err)
            assert.Equal(t, tt.want.Status, got.Status)
        })
    }
}
```

### 必須テストケース

| ケース | 説明 |
|--------|------|
| 正常系 | 最低1ケース |
| 必須フィールド欠落 | 各必須フィールドで1ケースずつ |
| 不正な値 | 型違い、範囲外 |
| 境界値 | 最小値、最大値、空 |
| 存在しないリソース | 404 相当 |
| 権限エラー | 403 相当 |
| DB エラー | Repository のエラー伝播 |

### モック作成

```go
// mockgen を使用
//go:generate mockgen -source=repository.go -destination=mock_repository.go -package=repository

// または手動
type mockWorkflowRepo struct {
    mock.Mock
}

func (m *mockWorkflowRepo) Create(ctx context.Context, w *domain.Workflow) error {
    args := m.Called(ctx, w)
    return args.Error(0)
}
```

### E2E テスト

```go
func TestWorkflowE2E_CRUD(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping e2e test")
    }

    // Setup
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)

    client := &http.Client{}
    baseURL := "http://localhost:8080"

    // Create
    body := `{"name": "E2E Test Workflow"}`
    req, _ := http.NewRequest("POST", baseURL+"/api/v1/workflows", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Tenant-ID", testTenantID)

    resp, err := client.Do(req)
    require.NoError(t, err)
    require.Equal(t, http.StatusCreated, resp.StatusCode)

    var created Workflow
    json.NewDecoder(resp.Body).Decode(&created)
    resp.Body.Close()

    // Verify
    assert.NotEmpty(t, created.ID)
    assert.Equal(t, "E2E Test Workflow", created.Name)

    // Cleanup
    // ...
}
```

---

## Frontend Testing (Vue/Nuxt)

### テストファイル配置

```
frontend/
├── composables/
│   ├── useWorkflows.ts
│   └── useWorkflows.test.ts
├── components/
│   └── dag-editor/
│       ├── DagEditor.vue
│       └── DagEditor.test.ts
└── vitest.config.ts
```

### Composable テスト

```typescript
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useWorkflows } from './useWorkflows'

// API モック
vi.mock('./useApi', () => ({
  useApi: () => ({
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    del: vi.fn(),
  }),
}))

describe('useWorkflows', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('fetchWorkflows', () => {
    it('fetches workflows successfully', async () => {
      // Arrange
      const mockWorkflows = [
        { id: '1', name: 'Test Workflow', status: 'draft' },
      ]
      const { get } = useApi()
      vi.mocked(get).mockResolvedValue({ workflows: mockWorkflows })

      // Act
      const { workflows, loading, error, fetchWorkflows } = useWorkflows()
      expect(loading.value).toBe(false)

      const fetchPromise = fetchWorkflows()
      expect(loading.value).toBe(true)

      await fetchPromise

      // Assert
      expect(loading.value).toBe(false)
      expect(error.value).toBeNull()
      expect(workflows.value).toEqual(mockWorkflows)
    })

    it('handles API error', async () => {
      // Arrange
      const { get } = useApi()
      vi.mocked(get).mockRejectedValue(new Error('Network error'))

      // Act
      const { error, fetchWorkflows } = useWorkflows()
      await fetchWorkflows()

      // Assert
      expect(error.value).toBe('Network error')
    })

    it('handles empty response', async () => {
      // Arrange
      const { get } = useApi()
      vi.mocked(get).mockResolvedValue({ workflows: [] })

      // Act
      const { workflows, fetchWorkflows } = useWorkflows()
      await fetchWorkflows()

      // Assert
      expect(workflows.value).toEqual([])
    })
  })
})
```

### Component テスト

```typescript
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import StepNode from './StepNode.vue'

describe('StepNode', () => {
  it('renders step name', () => {
    const wrapper = mount(StepNode, {
      props: {
        data: {
          id: '1',
          name: 'Test Step',
          type: 'llm',
        },
      },
    })

    expect(wrapper.text()).toContain('Test Step')
  })

  it('emits click event', async () => {
    const wrapper = mount(StepNode, {
      props: {
        data: { id: '1', name: 'Test', type: 'llm' },
      },
    })

    await wrapper.trigger('click')

    expect(wrapper.emitted('click')).toBeTruthy()
  })

  it('shows different icon for each type', () => {
    const types = ['llm', 'tool', 'condition'] as const

    types.forEach((type) => {
      const wrapper = mount(StepNode, {
        props: { data: { id: '1', name: 'Test', type } },
      })

      // Each type should have a unique icon
      expect(wrapper.find('.step-icon').exists()).toBe(true)
    })
  })
})
```

### テストコマンド

```bash
# 全テスト実行
npm run test

# 単一ファイル
npm run test -- useWorkflows.test.ts

# Watch モード
npm run test:watch

# カバレッジ
npm run test:coverage
```

---

## Bug Fix Testing Flow

バグ修正時は TDD アプローチを使用。

### 手順

```
1. 再現テストを作成
   └── 現在の（バグがある）挙動を期待値として書く

2. テストが失敗することを確認
   └── 「このバグがある」ことをテストで証明

3. バグを修正（最小限の変更）

4. テストが成功することを確認

5. エッジケーステストを追加
   └── 類似のバグを防ぐ

6. 既存テストがパスすることを確認
   └── リグレッションがないことを確認
```

### 例

```go
// Step 1: 再現テスト作成（失敗するはず）
func TestWorkflowPublish_WithNoSteps_ShouldFail(t *testing.T) {
    // バグ: ステップがないワークフローを公開できてしまう
    repo := setupTestRepo()
    uc := usecase.NewWorkflowUsecase(repo)

    // ステップなしのワークフロー
    workflow := &domain.Workflow{ID: uuid.New(), Status: "draft"}
    repo.Create(context.Background(), workflow)

    // 公開しようとするとエラーになるべき
    err := uc.Publish(context.Background(), workflow.TenantID, workflow.ID)

    assert.Error(t, err)  // 現在は成功してしまう → このテストは失敗する
    assert.ErrorIs(t, err, domain.ErrValidation)
}

// Step 3: バグ修正後、テストが成功するようになる

// Step 5: エッジケース追加
func TestWorkflowPublish_EdgeCases(t *testing.T) {
    tests := []struct {
        name      string
        stepCount int
        wantErr   bool
    }{
        {"no steps", 0, true},
        {"one step", 1, false},
        {"many steps", 100, false},
    }
    // ...
}
```

---

## Coverage Requirements

| 領域 | 最小カバレッジ | 推奨カバレッジ |
|------|--------------|--------------|
| Domain | 80% | 90% |
| Usecase | 70% | 85% |
| Handler | 50% | 70% |
| Repository | 40% | 60% |
| E2E | 60% | 80% |

### カバレッジ確認

```bash
# Backend
cd backend && go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Frontend
cd frontend && npm run test:coverage
```

---

## Common Testing Mistakes

### ❌ 避けるべきパターン

```go
// 1. エラーを無視
result, _ := uc.Create(ctx, input)  // エラーチェックなし → NG

// 2. ハードコードされた値
assert.Equal(t, "abc123", result.ID)  // UUID は固定値ではない → NG

// 3. 外部依存
func TestAPI(t *testing.T) {
    resp, _ := http.Get("http://localhost:8080/api/...")  // 外部サービス依存 → NG
}

// 4. テスト間の依存
var globalWorkflow *Workflow  // テスト間で状態共有 → NG
```

### ✅ 正しいパターン

```go
// 1. エラーを常にチェック
result, err := uc.Create(ctx, input)
require.NoError(t, err)

// 2. 動的な値には適切なアサーション
assert.NotEmpty(t, result.ID)
assert.True(t, uuid.Validate(result.ID) == nil)

// 3. モックを使用
mockRepo := new(mockWorkflowRepo)
mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

// 4. テストごとに独立したセットアップ
func TestXxx(t *testing.T) {
    repo := setupTestRepo()  // 各テストで独立
    // ...
}
```

---

## Test Data Management

### Fixture パターン

```go
// testdata/workflows.go
package testdata

func ValidWorkflow() *domain.Workflow {
    return &domain.Workflow{
        ID:       uuid.New(),
        TenantID: testTenantID,
        Name:     "Test Workflow",
        Status:   "draft",
    }
}

func PublishedWorkflow() *domain.Workflow {
    w := ValidWorkflow()
    w.Status = "published"
    return w
}
```

### クリーンアップ

```go
func TestXxx(t *testing.T) {
    db := setupTestDB(t)
    t.Cleanup(func() {
        cleanupTestDB(t, db)
    })

    // テスト本体
}
```

---

## CI Integration

### GitHub Actions での実行

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Backend tests
        run: |
          cd backend
          go test -race -coverprofile=coverage.out ./...

      - name: Frontend tests
        run: |
          cd frontend
          npm ci
          npm run check
```

---

## Related Documents

- [BACKEND.md](./BACKEND.md) - テストパターン（Canonical Code Patterns）
- [FRONTEND.md](./FRONTEND.md) - フロントエンドテストパターン
- [WORKFLOW_RULES.md](./rules/WORKFLOW_RULES.md) - 開発ワークフロー
- [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) - テストエラー対処法
