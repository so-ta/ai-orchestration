# AGENTS.md - AI Agent Instructions

> このファイルはAIエージェント（OpenAI Codex等）向けの指示書です。
> リポジトリ内でAIエージェントが作業する際のガイドラインを定義します。

---

## プロジェクト概要

**AI Orchestration** - LLMとツール連携を備えたDAGワークフローの設計・実行・監視を行うマルチテナントSaaS。

| Item | Value |
|------|-------|
| Backend | Go 1.22+ (Clean Architecture) |
| Frontend | Vue 3 + Nuxt 3 (Composition API) |
| Database | PostgreSQL 16 |
| Cache/Queue | Redis 7 |
| Auth | Keycloak 24 (OIDC) |
| Tracing | OpenTelemetry + Jaeger |

---

## レビューガイドライン

PRレビュー時にCodexが従うべきガイドラインです。

### 1. コード品質基準

#### Backend (Go)

| 観点 | チェック項目 |
|------|------------|
| **エラーハンドリング** | `_` でエラーを無視していないか |
| **マルチテナント** | 全クエリに `tenant_id` フィルタがあるか |
| **インターフェース** | Repository/Adapter は interface 経由か |
| **ログ** | `log/slog` を使用しているか |
| **テスト** | `testify` を使用、テーブル駆動テストか |

```go
// Good: エラーを明示的に処理
result, err := repo.GetByID(ctx, id)
if err != nil {
    return nil, fmt.Errorf("failed to get workflow: %w", err)
}

// Bad: エラーを無視
result, _ := repo.GetByID(ctx, id)
```

#### Frontend (Vue/TypeScript)

| 観点 | チェック項目 |
|------|------------|
| **Composition API** | `<script setup lang="ts">` を使用 |
| **型安全** | 明示的な型定義、any/unknown 回避 |
| **Composables** | `use` プレフィックス |
| **コンポーネント** | PascalCase 命名 |
| **UI操作** | `alert/confirm/prompt` 禁止、`useToast()` を使用 |

```vue
<!-- Good -->
<script setup lang="ts">
const projectsApi = useProjects()
const projects = ref<Project[]>([])
const loading = ref(false)

onMounted(async () => {
  loading.value = true
  const result = await projectsApi.list()
  projects.value = result.data || []
  loading.value = false
})
</script>

<!-- Bad: Options API -->
<script>
export default {
  data() { return { projects: [] } }
}
</script>
```

### 2. セキュリティレビュー項目

| カテゴリ | チェック項目 |
|---------|------------|
| **SQLインジェクション** | パラメータ化クエリを使用しているか |
| **XSS** | ユーザー入力のサニタイズ |
| **認証・認可** | JWT検証、tenant_id分離 |
| **シークレット** | 環境変数経由、ハードコードなし |
| **入力検証** | 境界値、型チェック |

```go
// Good: パラメータ化クエリ
query := "SELECT * FROM workflows WHERE tenant_id = $1 AND id = $2"
row := db.QueryRow(ctx, query, tenantID, workflowID)

// Bad: 文字列連結（SQLインジェクション脆弱）
query := fmt.Sprintf("SELECT * FROM workflows WHERE id = '%s'", id)
```

### 3. パフォーマンスレビュー項目

| 観点 | チェック項目 |
|------|------------|
| **N+1クエリ** | ループ内でのDB呼び出し |
| **インデックス** | 検索条件にインデックスがあるか |
| **メモリ** | 大きなスライスの事前割り当て |
| **並行処理** | goroutine リーク、デッドロック |

### 4. テストカバレッジ要件

**新規コードにはテストが必須です。**

| 追加コード | 必要なテスト |
|-----------|------------|
| Handler | リクエスト検証、レスポンス形式 |
| Usecase | ビジネスロジック、エッジケース |
| Repository | CRUD、テナント分離 |
| Adapter | 外部API呼び出しのモック |
| Composable | 状態管理、API呼び出し |
| Component | マウント、props、events |

```go
// 必須: テーブル駆動テスト
func TestCreateWorkflow(t *testing.T) {
    tests := []struct {
        name    string
        input   CreateWorkflowInput
        wantErr bool
    }{
        {"valid input", CreateWorkflowInput{Name: "Test"}, false},
        {"empty name", CreateWorkflowInput{Name: ""}, true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // ...
        })
    }
}
```

### 5. ドキュメント要件

| 変更内容 | 更新するドキュメント |
|---------|-------------------|
| 新規API | `docs/API.md`, `docs/openapi.yaml` |
| DBスキーマ | `docs/DATABASE.md` |
| 新規ブロック | `docs/BLOCK_REGISTRY.md` |
| バックエンド構造 | `docs/BACKEND.md` |
| フロントエンド構造 | `docs/FRONTEND.md` |

### 6. アーキテクチャ制約

| 制約 | 説明 |
|------|------|
| **Clean Architecture** | Handler → Usecase → Domain → Repository |
| **Multi-tenancy** | 全データに tenant_id 必須 |
| **Unified Block Model** | 新規ブロックは Migration で追加 |
| **DAG Editor** | 衝突判定ロジックは FRONTEND.md 参照 |

---

## コードスタイル

### コミットメッセージ形式

```
<type>: <summary>

<body（任意）>
```

| type | 用途 |
|------|------|
| `feat` | 新機能 |
| `fix` | バグ修正 |
| `refactor` | リファクタリング |
| `docs` | ドキュメント |
| `test` | テスト |
| `chore` | ビルド、CI/CD |

### ブランチ命名規則

| パターン | 用途 |
|---------|------|
| `feature/xxx` | 新機能 |
| `fix/xxx` | バグ修正 |
| `docs/xxx` | ドキュメント |

---

## レビュー結果出力形式

レビュー結果は以下の形式で出力してください：

```markdown
## サマリー
（変更内容の要約）

## 良い点
（良い変更点）

## 提案
（改善提案）

## 修正必要
（修正が必要な問題）

## 判定
（APPROVE / REQUEST_CHANGES / COMMENT）
```

---

## 関連ドキュメント

| ドキュメント | 目的 |
|-------------|------|
| [CLAUDE.md](./CLAUDE.md) | プロジェクトルール全般 |
| [docs/BACKEND.md](./docs/BACKEND.md) | バックエンド構造 |
| [docs/FRONTEND.md](./docs/FRONTEND.md) | フロントエンド構造 |
| [docs/API.md](./docs/API.md) | API仕様 |
| [docs/TESTING.md](./docs/TESTING.md) | テスト統合ガイド |
| [docs/designs/UNIFIED_BLOCK_MODEL.md](./docs/designs/UNIFIED_BLOCK_MODEL.md) | ブロックアーキテクチャ |
