---
paths:
  - "backend/**/*_test.go"
  - "frontend/tests/**/*.ts"
  - "frontend/**/*.test.ts"
---

# Testing Rules

## Backend (Go)

### テーブル駆動テスト

```go
func TestProjectUsecase_Create(t *testing.T) {
    tests := []struct {
        name    string
        input   *CreateProjectInput
        want    *domain.Project
        wantErr error
    }{
        {"有効な入力", &CreateProjectInput{Name: "Test"}, &domain.Project{Status: domain.ProjectStatusDraft}, nil},
        {"空の名前", &CreateProjectInput{Name: ""}, nil, domain.ErrValidation},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Arrange, Act, Assert
        })
    }
}
```

### 必須カバレッジ

1. 正常系（最低1ケース）
2. 必須フィールド欠落
3. 不正な値
4. 境界値
5. 存在しないリソース（404）
6. 権限エラー（403）

### 実行コマンド

```bash
cd backend && go test ./...
cd backend && go test -race ./...
```

## Frontend (TypeScript)

### テスト構造

```typescript
describe('useProjectList', () => {
  it('fetches projects successfully', async () => {
    // Arrange
    vi.mocked(useApi).mockReturnValue({
      get: vi.fn().mockResolvedValue({ data: mockProjects }),
    })

    // Act
    const { projects, fetchProjects } = useProjectList()
    await fetchProjects()

    // Assert
    expect(projects.value).toEqual(mockProjects)
  })
})
```

### 必須カバレッジ

1. 正常系
2. APIエラー
3. ローディング状態
4. 空データ

### 実行コマンド

```bash
cd frontend && npm run test:run
cd frontend && npm run check   # typecheck + lint + test
```
