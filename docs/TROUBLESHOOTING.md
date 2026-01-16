# トラブルシューティングガイド

Claude Code がエラーに遭遇した際の対処法リファレンス。

> **Status**: Active
> **Updated**: 2026-01-15

---

## クイック診断

| 症状 | 最初に確認すること | 対処法 |
|------|------------------|--------|
| API 接続エラー | Docker コンテナ状態 | `docker compose ps` |
| DB 接続エラー | PostgreSQL 起動状態 | `make dev-middleware` |
| 型エラー (Go) | import 漏れ | `go mod tidy` |
| 型エラー (TS) | 型定義 | `npm run typecheck` |
| テスト失敗 | サービス再起動 | `docker compose restart api worker` |

---

## データベースエラー

### エラー: connection refused to localhost:5432

**原因**: PostgreSQL コンテナが起動していない

**対処**:
```bash
# middleware を起動
make dev-middleware

# または PostgreSQL のみ起動
docker compose up -d postgres
```

**Claude Code がよくする誤り**:
- DB 接続設定を変更しようとする → **しない**
- 環境変数を修正しようとする → **しない**

---

### エラー: relation "xxx" does not exist

**原因**: マイグレーションが適用されていない

**対処**:
```bash
make db-reset
```

**詳細対処** (特定マイグレーションのみ):
```bash
cd backend
DATABASE_URL="postgres://aio:aio_password@localhost:5432/ai_orchestration?sslmode=disable" \
  migrate -path migrations -database "$DATABASE_URL" up
```

---

### エラー: duplicate key value violates unique constraint

**原因**: seed データと既存データの衝突

**対処**:
```bash
make db-reset  # 全リセット
```

**部分対処**:
```bash
# 特定テーブルのみクリア
docker compose exec postgres psql -U aio -d ai_orchestration -c "TRUNCATE block_definitions CASCADE;"
make db-seed
```

---

### エラー: null value in column "xxx" violates not-null constraint

**原因**: INSERT/UPDATE 時の必須フィールド漏れ

**対処**:
1. エラーメッセージの column 名を確認
2. 対応するドメインモデル（`domain/*.go`）を確認
3. `NOT NULL` のフィールドを必ず設定

**よくある漏れ**:
| Column | 設定場所 |
|--------|---------|
| `tenant_id` | `middleware.GetTenantID(ctx)` |
| `created_at` | DB の DEFAULT または明示的に `time.Now()` |
| `status` | ドメインの初期値（例: `"draft"`, `"pending"`） |

---

## バックエンドエラー (Go)

### エラー: undefined: xxx

**原因**: 関数・変数・型が未定義

**対処**:
1. import 漏れを確認
2. 正しいパッケージをインポート
3. `go mod tidy` を実行

```bash
cd backend && go mod tidy
```

---

### エラー: cannot use xxx (type A) as type B

**原因**: 型の不一致

**対処**:
1. 期待される型を確認
2. 型変換を追加（例: `uuid.UUID` ↔ `string`）

**よくある変換**:
```go
// string → uuid.UUID
id, err := uuid.Parse(idStr)

// uuid.UUID → string
idStr := id.String()

// json.RawMessage → struct
var cfg StepConfig
json.Unmarshal(rawJSON, &cfg)

// struct → json.RawMessage
rawJSON, _ := json.Marshal(cfg)
```

---

### エラー: cannot find package "xxx"

**原因**: 依存パッケージが未インストール

**対処**:
```bash
cd backend && go mod tidy
```

**それでも解決しない場合**:
```bash
cd backend && go get <package-path>
```

---

### エラー: cyclic import

**原因**: パッケージ間の循環参照

**対処**:
1. 依存関係を図示
2. interface を抽出して共通パッケージに移動
3. または依存方向を見直し

**典型例**:
```
handler → usecase → repository
              ↑         │
              └─────────┘  ← NG (循環)

解決: interface を domain/ に移動
handler → usecase → domain/interfaces
repository → domain/interfaces
```

---

### エラー: context deadline exceeded

**原因**: 操作がタイムアウト

**対処**:
1. 外部 API 呼び出しのタイムアウト設定を確認
2. DB クエリが重い場合は最適化
3. 必要に応じてタイムアウト値を増加

**タイムアウト設定箇所**:
| 場所 | 設定 |
|------|------|
| HTTP クライアント | `&http.Client{Timeout: 30 * time.Second}` |
| DB クエリ | `context.WithTimeout(ctx, 10*time.Second)` |
| Sandbox | `sandbox.NewRuntime(30 * time.Second)` |

---

## フロントエンドエラー (Vue/Nuxt)

### エラー: Cannot find module 'xxx'

**原因**: node_modules が古い、または破損

**対処**:
```bash
cd frontend
rm -rf node_modules package-lock.json
npm install
```

---

### エラー: Type 'xxx' is not assignable to type 'yyy'

**原因**: TypeScript 型エラー

**対処**:
1. 型定義を確認（`types/api.ts`）
2. 正しい型を使用
3. 必要に応じて型ガードを追加

**よくある修正**:
```typescript
// unknown → 特定の型
const data = response as Workflow

// null チェック
if (workflow.value) {
  // workflow.value は非 null として扱える
}

// オプショナルチェーン
const name = workflow.value?.name ?? 'default'
```

---

### エラー: [Vue warn]: Invalid prop type

**原因**: コンポーネント props の型不一致

**対処**:
1. 親コンポーネントで渡している値の型を確認
2. 子コンポーネントの `defineProps` を確認
3. 型を一致させる

---

### エラー: Hydration mismatch

**原因**: サーバーサイドとクライアントサイドの HTML が不一致

**対処**:
1. `<ClientOnly>` でラップ
2. または `onMounted` で実行
3. `v-if="mounted"` パターンを使用

```vue
<script setup>
const mounted = ref(false)
onMounted(() => { mounted.value = true })
</script>

<template>
  <ClientOnly>
    <DynamicComponent v-if="mounted" />
  </ClientOnly>
</template>
```

---

### エラー: alert/confirm/prompt is not defined (SSR)

**原因**: ブラウザ API を SSR で使用

**対処**:
**禁止**: ブラウザの `alert/confirm/prompt` は使用禁止

**代わりに**:
```typescript
const toast = useToast()
toast.add({ title: 'Success', description: 'Operation completed' })
```

---

### エラー: @rollup/rollup-darwin-* not found

**原因**: ローカルビルド時の rollup 依存問題

**対処**:
```bash
cd frontend
rm -rf node_modules package-lock.json
npm install
```

**禁止事項**:
- `npm install @rollup/rollup-darwin-*` を実行 → **しない**
- package.json に追加 → **しない**

---

## Docker / コンテナエラー

### エラー: port is already allocated

**原因**: 同じポートで別のサービスが起動中

**対処**:
```bash
# 使用中のプロセスを確認
lsof -i :8080

# 停止
docker compose down
# または
docker compose stop <service>
```

---

### エラー: no such service: xxx

**原因**: docker-compose.yml に定義されていないサービス

**対処**:
1. `docker-compose.yml` を確認
2. 正しいサービス名を使用

**サービス一覧**:
```
api, worker, frontend, postgres, redis, keycloak, jaeger
```

---

### 変更が反映されない

**原因**: コンテナが古いイメージを使用中

**対処**:
```bash
# バックエンド変更
docker compose restart api worker

# フロントエンド変更
# → ホットリロードで自動反映（再起動不要）

# スキーマ変更
make db-reset && docker compose restart api worker
```

**Claude Code がよくする誤り**:
- コード変更後に動作確認 → **再起動を忘れる**
- テスト失敗後に修正 → **再起動を忘れる**

---

## Git / PR エラー

### エラー: conflict in xxx

**対処**:
1. `main` から最新を取得: `git fetch origin main`
2. リベース: `git rebase origin/main`
3. コンフリクト解消 → `git add .` → `git rebase --continue`
4. 再 push: `git push --force-with-lease`

**詳細は**: [GIT_RULES.md](./rules/GIT_RULES.md#conflict-resolution)

---

### エラー: pre-commit hook failed

**対処**:
1. エラーメッセージを確認
2. lint/format エラーなら修正
3. 再コミット

```bash
# Go の場合
cd backend && go fmt ./... && go vet ./...

# TypeScript の場合
cd frontend && npm run lint -- --fix
```

---

### CI が失敗

**対処**:
1. CI ログを確認
2. ローカルで同じテストを実行
3. 修正して再 push

```bash
# バックエンド
cd backend && go test ./...

# フロントエンド
cd frontend && npm run check
```

---

## Sandbox / ブロック実行エラー

### エラー: goja does not support 'await'

**原因**: ブロックコードで `await` を使用

**対処**:
`await` を削除し、同期的に呼び出す（内部でブロッキング）

```javascript
// ❌ NG
const response = await ctx.llm.chat(...);

// ✅ OK
const response = ctx.llm.chat(...);
```

---

### エラー: unknown adapter: xxx

**原因**: 存在しないアダプタを呼び出し

**対処**:
1. 利用可能なアダプタを確認: `ctx.adapter.list()`
2. 正しいアダプタ ID を使用

**利用可能なアダプタ**: `mock`, `openai`, `anthropic`, `http`

---

### エラー: execution timeout

**原因**: ブロックコードが時間内に完了しない

**対処**:
1. 無限ループがないか確認
2. 外部 API 呼び出しが長すぎないか確認
3. 必要に応じてタイムアウト設定を増加

---

## DAG エディタエラー

### ブロックがドラッグ後に元の位置に戻る

**原因**: 座標更新が正しく保存されていない

**対処**:
1. `@node-drag-stop` ハンドラを確認
2. API への保存処理が完了しているか確認
3. エラーレスポンスを確認

---

### グループリサイズ後にブロック位置がずれる

**原因**: 相対座標 vs 絶対座標の変換問題

**対処**:
1. [FRONTEND.md](./FRONTEND.md#group-resize-logic) を参照
2. `onGroupResize` でリアルタイム位置補正を確認
3. 子ノードの相対座標計算を確認

---

### エッジが正しく接続されない

**原因**: ポート名の不一致、または無効な接続

**対処**:
1. ソース/ターゲットのポート名を確認
2. エッジのバリデーションロジックを確認
3. 循環参照がないか確認

---

## Claude Code がよくする間違い

### 1. テスト通過後に再起動を忘れる

**症状**: コードを修正してテストは通るが、実際の動作確認で古い挙動

**対処**: バックエンド変更後は必ず:
```bash
docker compose restart api worker
```

---

### 2. 既存パターンを無視して独自実装

**症状**: 動作はするが、既存コードと整合性がない

**対処**: 修正前に必ず:
1. 類似機能の既存実装を検索
2. パターンを踏襲
3. 独自実装は最小限に

---

### 3. ドキュメント更新を忘れる

**症状**: コードと仕様書が乖離

**対処**: [DOCUMENTATION.md](./DOCUMENTATION.md) を参照

---

### 4. 過剰な修正

**症状**: 依頼された修正に加えて、周辺コードも「改善」

**対処**:
- 依頼された範囲のみ修正
- リファクタリングは別タスク

---

## エラー解決テンプレート

エラーに遭遇したら、以下のテンプレートで記録:

```markdown
### エラー: [エラーメッセージ]

**発生状況**:
- 実行したコマンド/操作:
- ファイル:
- 行番号:

**原因**:
[なぜこのエラーが発生したか]

**対処**:
[どう修正したか]

**予防策**:
[今後同じエラーを防ぐには]
```

---

## 関連ドキュメント

- [WORKFLOW_RULES.md](./rules/WORKFLOW_RULES.md) - 開発ワークフロー全般
- [TESTING.md](./TESTING.md) - テスト統合ガイド
- [GIT_RULES.md](./rules/GIT_RULES.md) - Git 操作ルール
- [BACKEND.md](./BACKEND.md) - バックエンド構造
- [FRONTEND.md](./FRONTEND.md) - フロントエンド構造
