# Development Workflow Rules

開発作業全般のルール。AIエージェントは必ず従うこと。

## Task Type Decision

作業開始前にタスクの種類と影響範囲を判別する。

### タスク種類の判別

| リクエスト内容 | 対応方針 |
|--------------|---------|
| バグ修正 | 再現テスト作成 → 修正 → テスト確認 |
| 新機能追加 | 既存パターン確認 → 実装 → テスト → ドキュメント |
| リファクタリング | 既存テスト確認 → リファクタ → 振る舞い不変を確認 |
| ドキュメント更新 | 関連コードを確認 → 更新 |
| 調査・質問 | コードを読んで回答（変更不要） |

### 影響範囲の判別

| 変更対象 | 対応方針 |
|---------|---------|
| 単一ファイルのみ | 直接修正 |
| 複数ファイル（同一パッケージ） | パッケージ内で完結させる |
| 複数パッケージ | 影響範囲を全て確認してから着手 |
| 不明 | 影響範囲を調査してから着手 |

---

## Before Any Code Change

```
1. Read docs/INDEX.md to find relevant document
2. Read the relevant document
3. Verify understanding matches implementation
```

---

## Service Restart After Code Changes (REQUIRED)

**バックエンドコード変更後は、必ず関連サービスを再起動すること。**

| 変更対象 | 再起動コマンド |
|----------|---------------|
| `backend/cmd/api/` | `docker compose restart api` |
| `backend/cmd/worker/` | `docker compose restart worker` |
| `backend/internal/` | `docker compose restart api worker` |
| 両方 | `docker compose restart api worker` |

**重要:**
- コード変更だけでは反映されない（特にDocker環境）
- 変更を検証する前に必ず再起動を実行
- フロントエンドはホットリロードで自動反映されるため再起動不要
- ローカル開発（air使用）の場合は自動リロードされる

---

## Self-Verification Checklist

作業完了時に確認:

### Backend

- [ ] `go build ./...` がパス
- [ ] `go test ./...` がパス
- [ ] 不要なデバッグコードを削除
- [ ] サービスを再起動して動作確認

### Frontend

- [ ] `npm run typecheck` がパス
- [ ] `npm run lint` がパス
- [ ] ブラウザで動作確認

### 共通

- [ ] コーディング規約に違反していないか
- [ ] テストを追加・更新したか
- [ ] ドキュメントを更新したか（必要な場合）
- [ ] コミットメッセージが規約に従っている

---

## Code Conventions

### Go

- gofmt/goimports
- Explicit error handling (no `_` ignore)
- log/slog for logging
- testify for tests

### Vue/TS

- Composition API
- `<script setup lang="ts">`
- PascalCase components
- `use` prefix for composables
- **ブラウザのalert/confirm/promptは使用禁止** - `useToast()`を使用

---

## Error Handling Decision

### ビルドエラー

| エラー種類 | 対処法 |
|-----------|--------|
| importエラー | `go mod tidy` を実行 |
| 型エラー | 型定義を確認、必要に応じて修正 |
| undefinedエラー | 関数・変数の定義漏れを確認 |
| 循環参照エラー | パッケージ構成を見直し |

### テストエラー

| エラー種類 | 対処法 |
|-----------|--------|
| アサーション失敗（期待値が間違い） | テストを修正 |
| アサーション失敗（実装が間違い） | 実装を修正 |
| パニック発生 | nil チェック、境界値を確認 |
| タイムアウト | 無限ループ、デッドロックを確認 |

### 解決できない場合

1. エラーメッセージを正確に記録
2. 再現手順を整理
3. 関連するコードを特定
4. ユーザーに報告し、追加情報を求める

---

## User Confirmation Required

以下のケースでは、ユーザーに確認してから進める:

| ケース | 確認内容 |
|--------|---------|
| 破壊的変更 | 既存の振る舞いを変更してよいか |
| 大規模リファクタリング | 方針を確認 |
| 外部API仕様の不明点 | 仕様の詳細 |
| セキュリティに関わる変更 | 認証・認可の要件 |
| 複数の実装方法がある | どのアプローチを取るか |

---

## Session Management

### セッション終了時の引き継ぎ

```
1. 未完了タスクをTodoWriteで記録
2. 実装した内容のドキュメント更新
3. 既知の問題・課題を明記
4. 次のアクションを明確に記載
```

### 禁止事項

- ドキュメント更新なしでのセッション終了
- 暗黙的な前提に依存した実装
- 口頭説明でしか伝わらない設計決定

---

## DAG Editor Modification

**ワークフローエディタ（DAGエディタ）を修正する場合、必ず以下を確認すること：**

```
1. [ ] docs/FRONTEND.md の「Block Group Push Logic」セクションを読む
2. [ ] docs/FRONTEND.md の「Group Resize Logic」セクションを読む
3. [ ] Vue Flowの親子ノード関係（相対座標 vs 絶対座標）を理解する
4. [ ] 衝突判定ロジックの3ケース分類を理解する
```

| 領域 | 注意点 |
|------|--------|
| 座標系 | 子ノードは親からの相対座標。押出後は絶対座標に変換必須 |
| リサイズ | `onGroupResize`でリアルタイム位置補正必須 |
| 衝突判定 | `fullyInside`, `fullyOutside`, `onBoundary`の3ケースで処理 |
| イベント順序 | `group:update` → `group:resize-complete`の順で発火必須 |
| ネスト | グループのネストは非対応（外側にスナップ） |

**関連ファイル:**
- `components/dag-editor/DagEditor.vue` - 衝突判定、リサイズハンドラ
- `pages/workflows/[id].vue` - イベントハンドラ、API永続化

---

## Rules with Context (Why and Past Failures)

各ルールの背景と過去の失敗例を記載。Claude Code がルールの重要性を理解するため。

### Rule 1: バックエンドコード変更後の再起動必須

**Why:**
Docker コンテナは起動時にバイナリをロードする。コード変更してもコンテナ内の古いバイナリが動作し続ける。

**過去の失敗例:**
```
症状: バグ修正のコードを書いたのに、動作確認で修正前の挙動が再現
原因: `docker compose restart api worker` を実行していなかった
結果: 「修正したのに直っていない」と誤認し、さらに不要な修正を重ねた
```

**対策:**
- コード変更後、テスト実行前に必ず再起動
- ローカル開発（air使用）の場合は自動リロードされるが、Docker 環境では手動再起動が必須

---

### Rule 2: 座標計算のキャッシュ禁止（DAG Editor）

**Why:**
グループのリサイズやドラッグ時、内部要素の座標は親の変更に連動して再計算される必要がある。
`computed` でキャッシュすると、依存関係が正しく追跡されず古い座標が返される。

**過去の失敗例:**
```
症状: BlockGroup をドラッグした時、内部ブロックが正しく追従しない
原因: `getBoundingBox()` の結果を `computed` でキャッシュしていた
結果: グループリサイズ後も古い座標を返し、ドラッグ時に位置がずれた
```

**対策:**
```typescript
// ❌ 禁止
const cachedBounds = computed(() => getBoundingBox(node))

// ✅ 正しい
function getCurrentBounds(node: Node) {
  return getBoundingBox(node) // 毎回再計算
}
```

---

### Rule 3: テナント分離の徹底

**Why:**
マルチテナント SaaS では、テナント A のデータをテナント B が参照できてはならない。
`tenant_id` フィルタを忘れると、セキュリティ違反となる。

**過去の失敗例:**
```
症状: ワークフロー一覧で他テナントのワークフローが表示された
原因: Repository クエリで `WHERE tenant_id = $2` を忘れていた
結果: セキュリティインシデント（データ漏洩）
```

**対策:**
```go
// ❌ 禁止
query := `SELECT * FROM workflows WHERE id = $1`

// ✅ 正しい
query := `SELECT * FROM workflows WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`
```

---

### Rule 4: 論理削除（Soft Delete）の一貫性

**Why:**
`deleted_at` カラムを持つテーブルは論理削除方式。物理削除すると外部キー制約や監査ログに問題が発生する。

**過去の失敗例:**
```
症状: 削除したはずのワークフローが復活した
原因: DELETE 文で物理削除したが、関連テーブルに参照が残っていた
結果: データ不整合、エラー発生
```

**対策:**
```sql
-- ❌ 禁止
DELETE FROM workflows WHERE id = $1

-- ✅ 正しい
UPDATE workflows SET deleted_at = NOW() WHERE id = $1 AND tenant_id = $2
```

---

### Rule 5: ブラウザダイアログ（alert/confirm/prompt）の禁止

**Why:**
1. SSR 環境（Nuxt）ではブラウザ API が存在しない
2. ブロッキングダイアログは UX が悪い
3. スタイリングができない

**過去の失敗例:**
```
症状: ページロード時に「ReferenceError: confirm is not defined」エラー
原因: <script setup> のトップレベルで confirm() を呼び出していた
結果: SSR 時にサーバーサイドでエラー発生、ページが表示されない
```

**対策:**
```typescript
// ❌ 禁止
if (confirm('Delete?')) { ... }

// ✅ 正しい
const toast = useToast()
toast.add({ title: 'Deleted', color: 'green' })
```

---

### Rule 6: Context 伝播の徹底

**Why:**
OpenTelemetry トレースは `context.Context` を通じて親子関係を追跡する。
`context.Background()` で新しいコンテキストを作ると、トレースが途切れて問題の追跡が困難になる。

**過去の失敗例:**
```
症状: Jaeger でワークフロー実行のトレースを見ると、途中で途切れている
原因: Usecase 内で ctx := context.Background() を使っていた
結果: エラー発生時にどこで問題が起きたか特定できない
```

**対策:**
```go
// ❌ 禁止
func (u *Usecase) Execute(tenantID, id uuid.UUID) error {
    ctx := context.Background()  // トレース途切れ
    // ...
}

// ✅ 正しい
func (u *Usecase) Execute(ctx context.Context, tenantID, id uuid.UUID) error {
    ctx, span := telemetry.StartSpan(ctx, "Usecase.Execute")
    defer span.End()
    // ...
}
```

---

### Rule 7: 既存パターンの踏襲

**Why:**
コードベースの一貫性を保つため。異なるパターンが混在すると、読み手の認知負荷が増加し、バグが発生しやすくなる。

**過去の失敗例:**
```
症状: 新しい Handler でエラーハンドリングが他と異なる
原因: 既存の Handler を参照せずに独自実装した
結果: エラーレスポンスの形式が不統一、フロントエンドでエラー処理に失敗
```

**対策:**
```
1. 実装前に類似機能の既存コードを検索
2. パターンを踏襲
3. 独自実装は最小限に
```

---

### Rule 8: ドキュメント更新の同期

**Why:**
コードとドキュメントが乖離すると、次の開発者（または AI エージェント）が誤った情報に基づいて作業する。

**過去の失敗例:**
```
症状: 新しい API エンドポイントを追加したが、フロントエンドから呼び出せない
原因: API.md と openapi.yaml を更新していなかったため、フロントエンド開発者が存在を知らなかった
結果: 重複実装、後からの統合作業が発生
```

**対策:**
- コード変更と同時にドキュメントを更新
- [DOCUMENTATION.md](../DOCUMENTATION.md) の更新マッピングを参照

---

## Related Documents

- [GIT_RULES.md](./GIT_RULES.md) - Git操作ルール
- [TESTING.md](../TESTING.md) - テスト統合ガイド
- [DOCUMENTATION.md](../DOCUMENTATION.md) - ドキュメント作成ガイド
