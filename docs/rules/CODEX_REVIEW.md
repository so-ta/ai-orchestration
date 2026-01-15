# Codex PR Review Workflow

AIエージェントがPRをpushした後のレビューフロー。

---

## ワークフロー概要

```
1. AIエージェントがコードを変更
   ↓
2. ブランチ作成（mainブランチでない場合）
   - 現在のブランチがmainの場合: 新しいブランチを作成してチェックアウト
   - 既にfeatureブランチの場合: そのまま継続
   ↓
3. ローカルCIを実行（必須）
   - Backend変更: go test ./...
   - Frontend変更: npm run check
   ↓
4. ローカルCIが全てパスしたことを確認
   ↓
5. git push でリモートにプッシュ（-u でupstream設定）
   ↓
6. PRを作成（または既存PRに追加コミット）
   ↓
7. GitHub Actions で Codex Review + CI が自動実行
   ↓
8. PRコメントにレビュー結果が投稿される
   ↓
9. レビュー結果とCI結果を確認
   ↓
10a. APPROVE（承認）かつ CI通過 → AIエージェントがMergeを実行
10b. REQUEST_CHANGES（要修正）またはCI失敗 → 修正して再push（手順3から）
   ↓
11. 10b の場合、手順 3-10 を繰り返す（承認されるまで）
```

---

## ブランチ作成ルール

**mainブランチで直接作業している場合は、必ず新しいブランチを作成してからpushすること。**

```bash
# 現在のブランチを確認
git branch --show-current

# mainブランチの場合、新しいブランチを作成
git checkout -b feature/your-feature-name

# または修正の場合
git checkout -b fix/your-fix-name

# リモートにプッシュ（upstreamを設定）
git push -u origin feature/your-feature-name
```

| ブランチ命名規則 | 用途 |
|-----------------|------|
| `feature/xxx` | 新機能追加 |
| `fix/xxx` | バグ修正 |
| `refactor/xxx` | リファクタリング |
| `docs/xxx` | ドキュメント更新 |
| `test/xxx` | テスト追加・修正 |

**重要**: mainブランチに直接pushすることは禁止されています。

---

## Push前のローカルCI実行（必須）

**pushする前に、必ずローカルでCI相当のチェックを実行すること。**

詳細は [GIT_RULES.md](./GIT_RULES.md#push前のローカルci実行必須) を参照。

```bash
# Backend変更がある場合
cd backend && go test ./...

# Frontend変更がある場合
cd frontend && npm run check

# 両方変更がある場合
(cd backend && go test ./...) && (cd frontend && npm run check)
```

| ルール | 説明 |
|--------|------|
| **ローカルCI必須** | pushする前に必ずローカルCIを実行 |
| **失敗時はpush禁止** | ローカルCIが通らない状態でpushしない |
| **修正後は再実行** | 修正を加えたら再度ローカルCIを実行 |

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

## 再レビュー時のルール

### PRコメントの確認（必須）

再レビューを行う際は、**必ずPRのコメントを確認すること**。

```bash
# PRのコメントを確認
gh pr view <PR番号> --comments
```

PRコメントには以下の情報が含まれる可能性がある：
- 対応範囲外の説明
- 過去の指摘への回答
- 技術的な背景説明

### 対応範囲外の指摘

PR説明またはコメントで「対応範囲外」と明記されている項目については：

| 対応 | 説明 |
|------|------|
| **レビューで指摘しない** | 対応範囲外と明記されている内容はレビュー指摘から除外 |
| **Issueに起票する** | 将来対応が必要な場合はIssueとして起票 |

```bash
# Issue作成例
gh issue create --title "対応範囲外の改善: XXX" --body "PR #123 で対応範囲外とされた項目"
```

### 過去の誤った指摘

過去のレビューで誤った指摘があった場合：

| 状況 | 対応 |
|------|------|
| 指摘が技術的に誤っていた | 無視して良い |
| 指摘がプロジェクトルールに反していた | 無視して良い |
| 指摘が既に解決済み | 無視して良い |

**重要**: 誤った指摘に対応するために無駄な修正を行う必要はない。

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
