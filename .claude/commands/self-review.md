# Self Review Workflow

PR作成前の自己レビュー・セルフバリデーション。

## 目的

- PR作成前にコード品質を確認
- CIで失敗しそうな問題を事前に検出
- レビュー指摘を事前に防ぐ

## 実行手順

### 1. 変更内容の確認

```bash
git status
git diff --stat
git diff
```

### 2. ローカルCI実行

#### Backend変更がある場合

```bash
cd backend && go test ./...
cd backend && go vet ./...
```

#### Frontend変更がある場合

```bash
cd frontend && npm run check
```

### 3. コードレビュー観点でのチェック

以下の観点で変更内容を確認：

| 観点 | チェック項目 |
|------|-------------|
| 機能性 | 要件を満たしているか |
| エラーハンドリング | nil/undefined、エラー処理は適切か |
| セキュリティ | 入力検証、認可チェックは適切か |
| パフォーマンス | N+1クエリ、不要なループはないか |
| 可読性 | 命名、構造は明確か |
| テスト | テストは十分か、エッジケースはカバーしているか |

### 4. プロジェクト規約チェック

- [ ] Canonical Patternに従っているか（BACKEND.md / FRONTEND.md参照）
- [ ] ドキュメント更新が必要な場合、更新したか
- [ ] コミットメッセージがGIT_RULESに従っているか

### 5. 問題発見時の対応

問題を発見した場合は、PR作成前に修正する。

```bash
# 修正後
git add .
git commit --amend  # または新規commit
```

## 自動チェックリスト

```
[ ] git statusで未追跡ファイルを確認した
[ ] Backend: go test ./... が通る
[ ] Backend: go vet ./... が通る
[ ] Frontend: npm run check が通る
[ ] 変更差分を確認した
[ ] セキュリティ観点で問題ないことを確認した
[ ] 不要なデバッグコード（console.log等）を削除した
[ ] ドキュメントを更新した（必要な場合）
```

## よくある指摘パターン（事前防止）

| パターン | 確認方法 |
|---------|---------|
| 未使用import | go vet / eslint |
| フォーマット不統一 | gofmt / prettier |
| エラー無視 | `_ = err` の検索 |
| TODO残り | `TODO` `FIXME` の検索 |
| console.log残り | `console.log` の検索 |
| ハードコード | 定数、環境変数の確認 |

## 完了条件

すべてのチェックが通り、自信を持ってPRを作成できる状態になったら完了。

## 次のステップ

```bash
# PR作成
git push -u origin <branch-name>
gh pr create --title "タイトル" --body "説明"

# PR作成後は /review-pr を実行
```

## 参考

- [docs/rules/CODEX_REVIEW.md](docs/rules/CODEX_REVIEW.md)
- [docs/rules/GIT_RULES.md](docs/rules/GIT_RULES.md)
- [docs/TESTING.md](docs/TESTING.md)
