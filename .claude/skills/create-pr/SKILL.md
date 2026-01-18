---
name: create-pr
description: |
  Create a pull request after code changes are complete and reviewed.
  Use proactively after self-review passes and documentation is updated.
---

# Create Pull Request

自己レビュー完了後のPR作成。

## 前提条件

- self-review が完了している
- テストがすべてパスしている
- 必要なドキュメント更新が完了している

## 手順

### 1. 最終確認

```bash
# 変更内容の確認
git status
git diff --stat

# テスト最終確認
cd backend && go test ./...
cd frontend && npm run check
```

### 2. コミット

```bash
git add .
git commit -m "<type>: <description>"
```

コミットメッセージ形式:
- `feat:` 新機能
- `fix:` バグ修正
- `refactor:` リファクタリング
- `docs:` ドキュメント
- `test:` テスト
- `chore:` その他

### 3. プッシュ

```bash
git push -u origin <branch-name>
```

### 4. PR作成

```bash
gh pr create --title "<type>: <description>" --body "## Summary

- 変更内容の要約

## Changes

- 具体的な変更点

## Test Plan

- テスト方法"
```

## PR作成後

PR作成後は review-pr スキルでCIとCodexレビューの結果を確認する。

## 注意事項

- ローカルCIが通っていない状態でpushしない
- コミットメッセージは明確で簡潔に
- PRの説明は変更の「なぜ」を含める
