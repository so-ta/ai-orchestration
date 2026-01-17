# バックエンドテストガイドライン

> **最終更新**: 2026-01-12
> **関連**: [TESTING.md](./TESTING.md), [BACKEND.md](./BACKEND.md)

このドキュメントはGoバックエンドのテストガイドラインを定義します。

## 目次

1. [テスト構造](#テスト構造)
2. [テストパターン](#テストパターン)
3. [パッケージ別ガイドライン](#パッケージ別ガイドライン)
4. [テストテンプレート](#テストテンプレート)
5. [モックガイドライン](#モックガイドライン)
6. [テストコマンド](#テストコマンド)

---

## テスト構造

### ディレクトリレイアウト

```
backend/
├── internal/
│   ├── handler/
│   │   ├── workflow.go
│   │   └── workflow_test.go       # ソースと同じディレクトリ
│   ├── usecase/
│   │   ├── workflow.go
│   │   └── workflow_test.go
│   ├── repository/postgres/
│   │   ├── workflow.go
│   │   └── workflow_test.go
│   ├── adapter/
│   │   ├── openai.go
│   │   └── openai_test.go
│   └── engine/
│       ├── executor.go
│       └── executor_test.go
├── tests/
│   ├── e2e/                       # エンドツーエンドテスト
│   │   └── workflow_test.go
│   └── integration/               # 統合テスト（将来）
│       └── workflow_test.go
└── pkg/
    └── crypto/
        ├── encryptor.go
        └── encryptor_test.go
```

### 命名規則

| 項目 | 規則 | 例 |
|------|------------|---------|
| テストファイル | `{source}_test.go` | `workflow_test.go` |
| テスト関数 | `Test{関数名}_{シナリオ}` | `TestCreateWorkflow_EmptyName` |
| テーブルテスト名 | 説明的、小文字 | `"empty name returns error"` |
| モック | `Mock{インターフェース}` | `MockWorkflowRepository` |
| ヘルパー | `setup{コンポーネント}`, `create{フィクスチャ}` | `setupTestHandler`, `createTestWorkflow` |

---

## テストパターン

### テーブル駆動テスト（推奨）

```go
func TestCreateWorkflow(t *testing.T) {
    tests := []struct {
        name           string
        input          CreateWorkflowInput
        expectedError  string
        expectedStatus int
    }{
        {
            name:           "success",
            input:          CreateWorkflowInput{Name: "Test", TenantID: "tenant-1"},
            expectedStatus: http.StatusCreated,
        },
        {
            name:           "empty name",
            input:          CreateWorkflowInput{Name: "", TenantID: "tenant-1"},
            expectedError:  "name is required",
            expectedStatus: http.StatusBadRequest,
        },
        {
            name:           "missing tenant",
            input:          CreateWorkflowInput{Name: "Test", TenantID: ""},
            expectedError:  "tenant_id is required",
            expectedStatus: http.StatusBadRequest,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // セットアップ
            h := setupTestHandler(t)

            // 実行
            resp, err := h.CreateWorkflow(tt.input)

            // 検証
            if tt.expectedError != "" {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.expectedError)
            } else {
                assert.NoError(t, err)
            }
            assert.Equal(t, tt.expectedStatus, resp.StatusCode)
        })
    }
}
```

### 関連シナリオのサブテスト

```go
func TestWorkflowUsecase(t *testing.T) {
    t.Run("Create", func(t *testing.T) {
        t.Run("success", func(t *testing.T) { ... })
        t.Run("duplicate name", func(t *testing.T) { ... })
    })

    t.Run("Update", func(t *testing.T) {
        t.Run("success", func(t *testing.T) { ... })
        t.Run("not found", func(t *testing.T) { ... })
        t.Run("already published", func(t *testing.T) { ... })
    })

    t.Run("Publish", func(t *testing.T) {
        t.Run("success", func(t *testing.T) { ... })
        t.Run("invalid DAG", func(t *testing.T) { ... })
    })
}
```

### テストフィクスチャ

```go
// fixtures_test.go
func createTestWorkflow(t *testing.T, opts ...WorkflowOption) *domain.Workflow {
    t.Helper()

    w := &domain.Workflow{
        ID:       uuid.New().String(),
        TenantID: "test-tenant",
        Name:     "Test Workflow",
        Status:   domain.WorkflowStatusDraft,
    }

    for _, opt := range opts {
        opt(w)
    }

    return w
}

type WorkflowOption func(*domain.Workflow)

func WithName(name string) WorkflowOption {
    return func(w *domain.Workflow) { w.Name = name }
}

func WithStatus(status domain.WorkflowStatus) WorkflowOption {
    return func(w *domain.Workflow) { w.Status = status }
}
```

---

## パッケージ別ガイドライン

### Handler テスト

**目的**: HTTP リクエスト/レスポンスの検証

**テスト対象**:
- リクエストのバリデーション
- レスポンスフォーマット
- HTTPステータスコード
- ヘッダー処理（認証、テナントID）
- エラーレスポンス形式

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
        body           string
        tenantID       string
        expectedStatus int
        expectedBody   map[string]interface{}
    }{
        {
            name:           "success",
            body:           `{"name": "Test Workflow"}`,
            tenantID:       "tenant-1",
            expectedStatus: http.StatusCreated,
        },
        {
            name:           "invalid JSON",
            body:           `{invalid}`,
            tenantID:       "tenant-1",
            expectedStatus: http.StatusBadRequest,
            expectedBody:   map[string]interface{}{"error": map[string]interface{}{"code": "INVALID_JSON"}},
        },
        {
            name:           "missing tenant ID",
            body:           `{"name": "Test"}`,
            tenantID:       "",
            expectedStatus: http.StatusBadRequest,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // セットアップ
            mockUsecase := &MockWorkflowUsecase{}
            handler := NewWorkflowHandler(mockUsecase)

            req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows", bytes.NewBufferString(tt.body))
            req.Header.Set("Content-Type", "application/json")
            if tt.tenantID != "" {
                req.Header.Set("X-Tenant-ID", tt.tenantID)
            }
            rec := httptest.NewRecorder()

            // 実行
            handler.Create(rec, req)

            // 検証
            assert.Equal(t, tt.expectedStatus, rec.Code)
            if tt.expectedBody != nil {
                var resp map[string]interface{}
                json.Unmarshal(rec.Body.Bytes(), &resp)
                assert.Equal(t, tt.expectedBody, resp)
            }
        })
    }
}
```

### Usecase テスト

**目的**: ビジネスロジックの検証

**テスト対象**:
- ビジネスルールの適用
- 状態遷移
- バリデーション
- エラーハンドリング
- 外部サービス呼び出し（モック）

```go
package usecase_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestWorkflowUsecase_Publish(t *testing.T) {
    tests := []struct {
        name          string
        workflowID    string
        setupMock     func(*MockWorkflowRepo)
        expectedError string
    }{
        {
            name:       "success",
            workflowID: "wf-1",
            setupMock: func(m *MockWorkflowRepo) {
                m.On("GetByID", mock.Anything, "wf-1").Return(&domain.Workflow{
                    ID:     "wf-1",
                    Status: domain.WorkflowStatusDraft,
                    Steps:  []domain.Step{{Type: "start"}, {Type: "tool"}},
                }, nil)
                m.On("Update", mock.Anything, mock.AnythingOfType("*domain.Workflow")).Return(nil)
            },
        },
        {
            name:       "already published",
            workflowID: "wf-1",
            setupMock: func(m *MockWorkflowRepo) {
                m.On("GetByID", mock.Anything, "wf-1").Return(&domain.Workflow{
                    ID:     "wf-1",
                    Status: domain.WorkflowStatusPublished,
                }, nil)
            },
            expectedError: "workflow is already published",
        },
        {
            name:       "invalid DAG - no start node",
            workflowID: "wf-1",
            setupMock: func(m *MockWorkflowRepo) {
                m.On("GetByID", mock.Anything, "wf-1").Return(&domain.Workflow{
                    ID:     "wf-1",
                    Status: domain.WorkflowStatusDraft,
                    Steps:  []domain.Step{{Type: "tool"}}, // start なし
                }, nil)
            },
            expectedError: "workflow must have exactly one start node",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockRepo := &MockWorkflowRepo{}
            tt.setupMock(mockRepo)
            uc := usecase.NewWorkflowUsecase(mockRepo)

            err := uc.Publish(context.Background(), tt.workflowID)

            if tt.expectedError != "" {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.expectedError)
            } else {
                assert.NoError(t, err)
            }
            mockRepo.AssertExpectations(t)
        })
    }
}
```

### Repository テスト

**目的**: データベース操作の検証

**テスト対象**:
- CRUD操作
- クエリの正確性
- テナント分離
- トランザクション
- ページネーション

**方法**: テストコンテナまたはインメモリDB

```go
package repository_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func setupTestDB(t *testing.T) *sql.DB {
    t.Helper()

    ctx := context.Background()
    container, err := postgres.Run(ctx,
        "postgres:16",
        postgres.WithDatabase("test"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
    )
    require.NoError(t, err)

    t.Cleanup(func() {
        container.Terminate(ctx)
    })

    connStr, err := container.ConnectionString(ctx, "sslmode=disable")
    require.NoError(t, err)

    db, err := sql.Open("postgres", connStr)
    require.NoError(t, err)

    // マイグレーション実行
    runMigrations(t, db)

    return db
}

func TestWorkflowRepository_Create(t *testing.T) {
    db := setupTestDB(t)
    repo := postgres.NewWorkflowRepository(db)
    ctx := context.Background()

    t.Run("creates workflow", func(t *testing.T) {
        workflow := &domain.Workflow{
            TenantID: "tenant-1",
            Name:     "Test Workflow",
        }

        err := repo.Create(ctx, workflow)

        require.NoError(t, err)
        assert.NotEmpty(t, workflow.ID)

        // DBで検証
        fetched, err := repo.GetByID(ctx, workflow.ID)
        require.NoError(t, err)
        assert.Equal(t, workflow.Name, fetched.Name)
    })

    t.Run("enforces tenant isolation", func(t *testing.T) {
        // tenant-1 でワークフローを作成
        workflow := &domain.Workflow{TenantID: "tenant-1", Name: "Private"}
        repo.Create(ctx, workflow)

        // tenant-2 のコンテキストで取得を試みる
        ctx2 := context.WithValue(ctx, "tenant_id", "tenant-2")
        _, err := repo.GetByID(ctx2, workflow.ID)

        assert.Error(t, err) // 見つからないはず
    })
}
```

### Adapter テスト

**目的**: 外部サービス連携の検証

**テスト対象**:
- API呼び出しの正確性
- リクエスト/レスポンス変換
- エラーハンドリング
- タイムアウト
- リトライ

```go
package adapter_test

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestOpenAIAdapter_Complete(t *testing.T) {
    tests := []struct {
        name           string
        mockResponse   string
        mockStatus     int
        expectedResult string
        expectedError  string
    }{
        {
            name:           "success",
            mockResponse:   `{"choices":[{"message":{"content":"Hello!"}}]}`,
            mockStatus:     http.StatusOK,
            expectedResult: "Hello!",
        },
        {
            name:          "rate limited",
            mockResponse:  `{"error":{"message":"Rate limit exceeded"}}`,
            mockStatus:    http.StatusTooManyRequests,
            expectedError: "rate limit",
        },
        {
            name:          "invalid API key",
            mockResponse:  `{"error":{"message":"Invalid API key"}}`,
            mockStatus:    http.StatusUnauthorized,
            expectedError: "unauthorized",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // モックサーバーセットアップ
            server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                // リクエスト検証
                assert.Equal(t, "POST", r.Method)
                assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
                assert.Contains(t, r.Header.Get("Authorization"), "Bearer ")

                w.WriteHeader(tt.mockStatus)
                w.Write([]byte(tt.mockResponse))
            }))
            defer server.Close()

            adapter := openai.NewAdapter(openai.Config{
                BaseURL: server.URL,
                APIKey:  "test-key",
            })

            result, err := adapter.Complete(context.Background(), "Hello")

            if tt.expectedError != "" {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.expectedError)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expectedResult, result)
            }
        })
    }
}
```

### Engine テスト

**目的**: DAG実行ロジックの検証

**テスト対象**:
- ステップ実行順序
- 条件分岐
- 並列実行（Map）
- 結合（Join）
- エラー伝播
- 状態管理

```go
func TestExecutor_MapStep(t *testing.T) {
    tests := []struct {
        name           string
        input          []interface{}
        parallel       bool
        expectedOutput []interface{}
    }{
        {
            name:           "sequential processing",
            input:          []interface{}{"a", "b", "c"},
            parallel:       false,
            expectedOutput: []interface{}{"A", "B", "C"},
        },
        {
            name:           "parallel processing",
            input:          []interface{}{"a", "b", "c"},
            parallel:       true,
            expectedOutput: []interface{}{"A", "B", "C"}, // 順序は異なる可能性あり
        },
        {
            name:           "empty input",
            input:          []interface{}{},
            parallel:       false,
            expectedOutput: []interface{}{},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            executor := setupTestExecutor(t)

            step := &domain.Step{
                Type: "map",
                Config: map[string]interface{}{
                    "input_path": "$.items",
                    "parallel":   tt.parallel,
                },
            }

            ctx := &ExecutionContext{
                Input: map[string]interface{}{"items": tt.input},
            }

            result, err := executor.ExecuteStep(ctx, step)

            assert.NoError(t, err)
            assert.ElementsMatch(t, tt.expectedOutput, result.([]interface{}))
        })
    }
}
```

---

## モックガイドライン

### インターフェースベースのモック

```go
// Repository インターフェース
type WorkflowRepository interface {
    Create(ctx context.Context, workflow *domain.Workflow) error
    GetByID(ctx context.Context, id string) (*domain.Workflow, error)
    Update(ctx context.Context, workflow *domain.Workflow) error
    Delete(ctx context.Context, id string) error
}

// モック実装
type MockWorkflowRepository struct {
    mock.Mock
}

func (m *MockWorkflowRepository) Create(ctx context.Context, workflow *domain.Workflow) error {
    args := m.Called(ctx, workflow)
    return args.Error(0)
}

func (m *MockWorkflowRepository) GetByID(ctx context.Context, id string) (*domain.Workflow, error) {
    args := m.Called(ctx, id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*domain.Workflow), args.Error(1)
}
```

### HTTP モックサーバー

```go
func createMockAPIServer(t *testing.T, responses map[string]mockResponse) *httptest.Server {
    t.Helper()

    return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        key := r.Method + " " + r.URL.Path
        if resp, ok := responses[key]; ok {
            w.WriteHeader(resp.status)
            json.NewEncoder(w).Encode(resp.body)
        } else {
            w.WriteHeader(http.StatusNotFound)
        }
    }))
}
```

### モックすべきもの vs モックしないもの

| モックする | モックしない |
|------|------------|
| 外部 API（OpenAI 等） | ドメインロジック |
| データベース接続 | 純粋関数 |
| 時刻（`time.Now()`） | データ変換 |
| ファイルシステム | バリデーションルール |
| ネットワーク呼び出し | ビジネスルール |
| 環境変数 | 計算処理 |

---

## テストコマンド

### テストの実行

```bash
# 全テスト
cd backend && go test ./...

# 特定パッケージ
go test ./internal/handler/...

# 詳細出力
go test -v ./...

# カバレッジ付き
go test -cover ./...

# カバレッジレポート生成
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# 特定テスト実行
go test -run TestWorkflowHandler_Create ./internal/handler/...

# レースディテクター付き
go test -race ./...

# E2E テスト
go test -v ./tests/e2e/...
```

### カバレッジコマンド

```bash
# カバレッジ閾値チェック
go test -coverprofile=coverage.out ./...
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
if (( $(echo "$COVERAGE < 50" | bc -l) )); then
    echo "Coverage below threshold: $COVERAGE%"
    exit 1
fi
```

### 便利なテストフラグ

| フラグ | 用途 | 例 |
|------|---------|---------|
| `-v` | 詳細出力 | `go test -v` |
| `-run` | 特定テスト実行 | `go test -run TestCreate` |
| `-count` | N回実行 | `go test -count=10` |
| `-race` | レース検出 | `go test -race` |
| `-timeout` | タイムアウト設定 | `go test -timeout 30s` |
| `-short` | 長いテストをスキップ | `go test -short` |
| `-parallel` | 並列数設定 | `go test -parallel 4` |

---

## 関連ドキュメント

| ドキュメント | 説明 |
|----------|-------------|
| [TESTING.md](./TESTING.md) | テスト統合ガイド |
| [BACKEND.md](./BACKEND.md) | バックエンドアーキテクチャ |
| [frontend/docs/TESTING.md](../frontend/docs/TESTING.md) | フロントエンドテストルール |
