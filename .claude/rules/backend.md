---
paths:
  - "backend/**/*.go"
---

# Backend Rules (Go)

## アーキテクチャ

クリーンアーキテクチャ: Handler → Usecase → Domain → Repository

## 必須パターン

### Handler

```go
func (h *ProjectHandler) Create(c echo.Context) error {
    ctx := c.Request().Context()
    tenantID := middleware.GetTenantID(ctx)

    var req CreateProjectRequest
    if err := c.Bind(&req); err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
    }

    result, err := h.usecase.Create(ctx, tenantID, req.ToInput())
    if err != nil {
        return h.mapError(err)
    }

    return c.JSON(http.StatusCreated, NewProjectResponse(result))
}
```

### Usecase

- tenantID は必ず引数で受け取る
- ID は Usecase 内で生成（外部からの注入禁止）
- エラーは `fmt.Errorf("context: %w", err)` でラップ

### Repository

- すべてのクエリに `tenant_id` フィルタ必須
- すべてのクエリに `deleted_at IS NULL` 必須
- `SELECT *` 禁止（カラム明示）

## Domain Error

| Error | HTTP Status |
|-------|-------------|
| `domain.ErrNotFound` | 404 |
| `domain.ErrValidation` | 400 |
| `domain.ErrForbidden` | 403 |
| `domain.ErrConflict` | 409 |

## 禁止事項

- `c.Bind()` のエラー無視
- `context.Background()` の新規作成（トレース途切れ）
- `json.Unmarshal` のエラー無視

## テスト

テーブル駆動テスト必須。カバレッジ: 正常系、必須フィールド欠落、境界値、404、403。

## 参照

詳細は [docs/BACKEND.md](docs/BACKEND.md) を参照。
