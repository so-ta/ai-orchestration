# Testing Rules

テスト作成・実行のルール。AIエージェントは必ず従うこと。

---

## Test Commands

### Backend

```bash
# Unit tests
cd backend && go test ./...

# E2E tests
cd backend && go test ./tests/e2e/... -v

# With race detector
cd backend && go test -race ./...

# Docker environment
docker compose exec api go test ./...
```

### Frontend

```bash
# All checks (REQUIRED before commit)
cd frontend && npm run check

# Individual checks
cd frontend && npm run typecheck   # TypeScript
cd frontend && npm run lint        # ESLint
cd frontend && npm run test:run    # Unit tests

# Coverage report
cd frontend && npm run test:coverage
```

---

## Test Coverage Requirements

### Coverage Thresholds

| Area | Minimum | Target |
|------|---------|--------|
| Backend Unit | 50% | 70% |
| Frontend Unit | 40% | 60% |
| E2E Critical Paths | 60% | 80% |

**新規ファイルは Target を満たすこと。**

### 新規コード = 新規テスト (Mandatory)

| 追加するコード | 必要なテスト |
|--------------|-------------|
| 新規Handler | Handler unit tests + request validation |
| 新規Usecase | Usecase unit tests + edge cases |
| 新規Repository | Repository tests (DB mock or test container) |
| 新規Adapter | Adapter tests with mock external service |
| 新規Composable | Composable unit tests |
| 新規Component | Component tests (mount, props, events) |

### テストなしでマージ禁止

以下のケースではテストなしのコードをマージしない：

- 新規Handler/Usecase/Repository
- バグ修正
- セキュリティ関連の変更
- 外部API連携

---

## Bug Fix Flow

バグ修正時は必ず以下の手順を守ること：

```
1. バグを再現するテストを書く
2. テストが失敗することを確認
3. コードを修正
4. テストがパスすることを確認
5. 関連するエッジケースのテストも追加
```

---

## Test Quality Standards

| 基準 | 説明 |
|------|------|
| 独立性 | 他のテストに依存しない |
| 再現性 | 何度実行しても同じ結果 |
| 高速 | 単体テストは100ms以内 |
| 明確性 | テスト名から目的がわかる |

---

## Frontend Testing Details

### Component Test Example

```typescript
import { describe, it, expect } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import MyComponent from '../MyComponent.vue'

describe('MyComponent', () => {
  it('renders correctly', async () => {
    const component = await mountSuspended(MyComponent, {
      props: { title: 'Test Title' }
    })
    expect(component.text()).toContain('Test Title')
  })
})
```

### Composable Test Example

```typescript
import { describe, it, expect, vi } from 'vitest'
import { useExample } from '../useExample'

describe('useExample', () => {
  it('returns expected value', () => {
    const { value, increment } = useExample()
    expect(value.value).toBe(0)
    increment()
    expect(value.value).toBe(1)
  })
})
```

### API Mock Example

```typescript
import { describe, it, expect, vi } from 'vitest'

vi.mock('~/composables/useApi', () => ({
  useApi: () => ({
    get: vi.fn().mockResolvedValue({ data: [] }),
    post: vi.fn().mockResolvedValue({ data: { id: '123' } }),
  })
}))
```

---

## Frontend Testing Pitfalls

| Issue | Solution |
|-------|----------|
| Vue template expression error | Use `v-pre` directive for code examples |
| Browser API in template | Create wrapper function in `<script>` |
| Missing function parameters | Check composable signatures |
| Type 'unknown' | Use typed composables instead of raw `useApi()` |
| Platform-specific packages | Never add `@rollup/rollup-darwin-*` to dependencies |
| alert/confirm/prompt使用 | **禁止** - `useToast()`を使用 |

### Docker Build Verification

**package.json変更後は必ず確認:**

```bash
docker compose build frontend
```

**発生しやすいケース:**
- ローカルでrollupエラーが出た時に `npm install @rollup/rollup-darwin-arm64` で解決
- → **正しい対処**: `rm -rf node_modules package-lock.json && npm install`

---

## Backend Testing Details

### Unit Test Pattern

```go
func TestWorkflowUsecase_Create(t *testing.T) {
    repo := &mockWorkflowRepo{}
    uc := usecase.NewWorkflowUsecase(repo)

    w, err := uc.Create(ctx, &domain.Workflow{Name: "test"})

    assert.NoError(t, err)
    assert.NotEmpty(t, w.ID)
}
```

### E2E Test Pattern

```go
func TestWorkflowE2E(t *testing.T) {
    resp, _ := http.Post(baseURL+"/api/v1/workflows", "application/json", body)
    assert.Equal(t, http.StatusCreated, resp.StatusCode)
}
```

---

## Test Execution Checklist

コード変更完了前に確認：

```bash
# Backend
cd backend && go test ./...

# Frontend
cd frontend && npm run check

# E2E (重要な変更時)
cd backend && go test ./tests/e2e/... -v
```

---

## Related Documents

- [frontend/docs/TESTING.md](../../frontend/docs/TESTING.md) - Frontend testing details
- [docs/TEST_PLAN.md](../TEST_PLAN.md) - Test plan and coverage
- [docs/BACKEND_TESTING.md](../BACKEND_TESTING.md) - Backend testing patterns
