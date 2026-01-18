---
name: review-pr
description: |
  PR review workflow after push. Use after creating a pull request to check
  CI status and Codex review results, and handle any required changes.
---

# PR Review Workflow

PRをpush後のレビュー待機・対応フロー。

## 前提

- PRをpushした
- GitHub Actionsが実行中

## 確認手順

### 1. CIステータス確認

```bash
gh pr checks <PR番号>
```

### 2. Codexレビュー結果確認

```bash
gh pr view <PR番号> --comments
```

### 3. 結果に応じた対応

| 結果 | 対応 |
|------|------|
| APPROVE + CI通過 | Mergeを実行 |
| REQUEST_CHANGES | 指摘事項を修正して再push |
| CI失敗 | エラーを修正して再push |

## Merge実行

```bash
gh pr merge <PR番号> --squash --delete-branch
```

## 修正が必要な場合

```bash
# 1. 指摘事項を確認
gh pr view <PR番号> --comments

# 2. コードを修正

# 3. テスト実行
cd backend && go test ./...
cd frontend && npm run check

# 4. コミット＆プッシュ
git add .
git commit -m "fix: レビュー指摘対応"
git push

# 5. 再度レビュー結果を確認（自動実行される）
```

## 注意事項

- REQUEST_CHANGESは全て対応必須
- CI失敗を無視してマージしない
- レビュー結果を待たずにマージしない

## 参考

- [docs/rules/CODEX_REVIEW.md](docs/rules/CODEX_REVIEW.md)
- [docs/rules/GIT_RULES.md](docs/rules/GIT_RULES.md)
