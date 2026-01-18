---
name: add-test
description: |
  Add test cases to the codebase. Use when writing new tests, improving test coverage,
  or when the user asks to add tests for specific functionality.
---

# Add Test Cases

テストケースの追加。

## Backend (Go)

### テストファイルの配置

```
backend/internal/<package>/<file>_test.go
```

### テスト構造

```go
func Test<Function>_<Scenario>(t *testing.T) {
    // Arrange
    // テストデータの準備

    // Act
    // テスト対象の実行

    // Assert
    // 結果の検証
}
```

### テーブル駆動テスト

```go
func Test<Function>(t *testing.T) {
    tests := []struct {
        name    string
        input   <Type>
        want    <Type>
        wantErr bool
    }{
        {"valid case", input1, want1, false},
        {"error case", input2, nil, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Function(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("got = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### テスト実行

```bash
cd backend && go test ./...
cd backend && go test ./internal/<package>/... -v
```

## Frontend (TypeScript/Vue)

### テストファイルの配置

```
frontend/tests/<component>.test.ts
frontend/composables/<composable>.test.ts
```

### テスト構造

```typescript
import { describe, it, expect } from 'vitest'

describe('<Component/Function>', () => {
  it('should <expected behavior>', () => {
    // Arrange
    // Act
    // Assert
    expect(result).toBe(expected)
  })
})
```

### テスト実行

```bash
cd frontend && npm run test:run
cd frontend && npm run test:run -- --grep "<pattern>"
```

## テストのベストプラクティス

| 観点 | ガイドライン |
|------|-------------|
| カバレッジ | 正常系、異常系、境界値をカバー |
| 独立性 | 各テストは独立して実行可能に |
| 可読性 | テスト名で何をテストしているか明確に |
| 保守性 | テストデータは最小限に |

## 次のステップ

テスト追加後、全テストがパスすることを確認し、PR作成へ進む。
