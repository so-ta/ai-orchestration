# Codex PR Review Workflow

AIエージェントがPRをpushした後のレビューフロー。

---

## ワークフロー概要

```
1. AIエージェントがコードを変更
   ↓
2. git push でリモートにプッシュ
   ↓
3. PRを作成（または既存PRに追加コミット）
   ↓
4. GitHub Actions で Codex Review + CI が自動実行
   ↓
5. PRコメントにレビュー結果が投稿される
   ↓
6. レビュー結果とCI結果を確認
   ↓
7a. APPROVE（承認）かつ CI通過 → AIエージェントがMergeを実行
7b. REQUEST_CHANGES（要修正）またはCI失敗 → 修正して再push
   ↓
8. 7b の場合、手順 4-7 を繰り返す（承認されるまで）
```

---

## 確認すべきレビュー結果

| 判定 | 意味 | 対応 |
|------|------|------|
| **APPROVE** | 問題なし | CIも通過していればMergeを実行 |
| **REQUEST_CHANGES** | 修正が必要 | 指摘事項を修正して再push |
| **COMMENT** | コメントのみ | 内容を確認し、必要に応じて対応 |

---

## 修正→再レビューのループ

```
while (レビュー結果 != APPROVE) {
  1. レビューコメントの「要修正」セクションを確認
  2. 指摘された問題を修正
  3. git add && git commit && git push
  4. Codex Review の再実行を待つ
  5. 新しいレビュー結果を確認
}
```

---

## 重要な注意事項

| ルール | 説明 |
|--------|------|
| **レビュー待ち必須** | pushしたら必ずCodexレビュー完了を待つ |
| **CI待ち必須** | レビューと同時にCIの結果も確認する |
| **全指摘対応** | REQUEST_CHANGESの指摘は全て対応する |
| **再レビュー確認** | 修正後は必ず新しいレビュー結果を確認 |
| **AIがMerge実行** | APPROVE + CI通過を確認後、AIエージェントがMergeを実行する |
| **日本語コメント** | Codexは日本語でレビューコメントを出力する |

---

## レビュー結果の確認方法

```bash
# PRのコメントを確認
gh pr view <PR番号> --comments

# CIの状況を確認
gh pr checks <PR番号>

# 最新のワークフロー実行状況を確認
gh run list --limit 5

# ワークフローのログを確認
gh run view <run_id> --log
```

---

## Mergeの実行

APPROVE + CI通過を確認後、以下のコマンドでMergeを実行：

```bash
# PRをMerge（squash merge推奨）
gh pr merge <PR番号> --squash --delete-branch
```

---

## 関連ファイル

| ファイル | 説明 |
|---------|------|
| `.github/workflows/ci.yml` | CIワークフロー定義（テスト・ビルド） |
| `.github/workflows/codex-review.yml` | Codexレビューワークフロー定義 |
| `.github/codex/prompts/review.md` | レビュープロンプト（チェック項目） |
| `AGENTS.md` | Codex用レビューガイドライン |

---

## Related Documents

- [GIT_RULES.md](./GIT_RULES.md) - Git操作ルール
- [WORKFLOW_RULES.md](./WORKFLOW_RULES.md) - 開発ワークフロー
