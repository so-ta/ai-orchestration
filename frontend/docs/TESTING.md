# Frontend Testing Rules

このドキュメントはフロントエンドのテストワークフローとルールを定義します。
AIエージェントはこれらのルールに従ってコード変更を行う必要があります。

## Testing Framework

| Tool | Purpose |
|------|---------|
| Vitest | Unit/Integration tests |
| @vue/test-utils | Vue component testing |
| @nuxt/test-utils | Nuxt-specific testing |
| happy-dom | DOM environment |

## Commands

```bash
# Run tests in watch mode
npm run test

# Run tests once (CI mode)
npm run test:run

# Run tests with coverage
npm run test:coverage

# Full check (typecheck + lint + test)
npm run check
```

## Required Workflow for Code Changes

### Before Committing Code

AIエージェントは以下の手順を**必ず**実行すること：

1. **TypeScript型チェック**
   ```bash
   npm run typecheck
   ```
   - エラーが0件であることを確認
   - エラーがある場合は修正してから次に進む

2. **ESLintチェック**
   ```bash
   npm run lint
   ```
   - 警告・エラーを確認し修正

3. **テスト実行**
   ```bash
   npm run test:run
   ```
   - すべてのテストがパスすることを確認

4. **ブラウザ確認**
   - 開発サーバーが起動している場合はブラウザで動作確認
   - コンソールエラーがないことを確認

### Quick Check Command

すべてのチェックを一括実行：
```bash
npm run check
```

## Test File Structure

```
frontend/
├── components/
│   ├── MyComponent.vue
│   └── __tests__/
│       └── MyComponent.spec.ts
├── composables/
│   ├── useExample.ts
│   └── __tests__/
│       └── useExample.spec.ts
├── pages/
│   ├── index.vue
│   └── __tests__/
│       └── index.spec.ts
└── tests/
    └── utils/           # Test utilities
        └── setup.ts
```

## Writing Tests

### Component Test Example

```typescript
import { describe, it, expect } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import MyComponent from '../MyComponent.vue'

describe('MyComponent', () => {
  it('renders correctly', async () => {
    const component = await mountSuspended(MyComponent, {
      props: {
        title: 'Test Title'
      }
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

// Mock the API
vi.mock('~/composables/useApi', () => ({
  useApi: () => ({
    get: vi.fn().mockResolvedValue({ data: [] }),
    post: vi.fn().mockResolvedValue({ data: { id: '123' } }),
  })
}))
```

## Agent Rules (REQUIRED)

### 1. No Code Without Verification

**絶対にブラウザ確認なしでコードを完了としない**

- コード変更後は必ず `npm run typecheck` を実行
- TypeScriptエラーがある場合は修正するまで完了としない
- 可能な限りブラウザでの動作確認を行う

### 2. Test-First for Bug Fixes

バグ修正時は以下の手順を守ること：

1. バグを再現するテストを書く
2. テストが失敗することを確認
3. コードを修正
4. テストがパスすることを確認

### 3. Mandatory Checks Before Completion

コード変更が完了したと報告する前に：

```bash
npm run check
```

このコマンドが成功しない限り、コード変更は完了とみなさない。

### 4. Error Handling

- TypeScriptエラーは無視しない
- `any` 型の使用は最小限に
- 型エラーは根本原因を修正する（キャストで回避しない）

### 5. Common Pitfalls to Avoid

| Issue | Solution |
|-------|----------|
| Vue template expression error | Use `v-pre` directive for code examples |
| Browser API in template | Create wrapper function in `<script>` |
| Missing function parameters | Check composable signatures |
| Type 'unknown' | Use typed composables instead of raw `useApi()` |
| Platform-specific packages | Never add `@rollup/rollup-darwin-*` or similar to dependencies |
| alert/confirm/prompt使用 | **禁止** - AIブラウザ操作をブロック。`useToast()`を使用 |

### 6. Docker Build Verification

**コード変更後はDockerビルドも確認すること：**

```bash
# プロジェクトルートで実行
docker compose build frontend
```

**禁止事項：**
- プラットフォーム固有パッケージ（`@rollup/rollup-darwin-arm64`等）をdependenciesに追加
- `npm install <package>` で意図せず追加されたパッケージをそのままにする
- ローカル環境のみで動作確認してDockerビルドを無視する

**発生しやすいケース：**
- ローカルでrollupエラーが出た時に `npm install @rollup/rollup-darwin-arm64` で解決
  - → **正しい対処**: `rm -rf node_modules package-lock.json && npm install`

## CI Integration

将来的にはCI/CDパイプラインで以下を自動実行：

```yaml
# .github/workflows/frontend-test.yml
name: Frontend Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
      - run: cd frontend && npm ci
      - run: cd frontend && npm run check
```

## Coverage Goals

| Metric | Target |
|--------|--------|
| Statements | 70% |
| Branches | 60% |
| Functions | 70% |
| Lines | 70% |

## Related Documentation

- [FRONTEND.md](./FRONTEND.md) - Frontend architecture
- [CLAUDE.md](../CLAUDE.md) - Project conventions
