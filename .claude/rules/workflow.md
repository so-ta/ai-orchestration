# Workflow Rules

コード変更時のワークフロールール。

## 自律的に進める

- 実装方針が明確
- 既存パターンに従う変更
- テストが通る修正
- ドキュメント更新
- リファクタリング
- バグ修正

## 人間に確認する

- 要件が曖昧
- 破壊的変更（API/DBスキーマ）
- セキュリティ判断
- 外部サービス課金への影響

## コード変更後の必須フロー

1. ローカルCI実行
   - Backend: `cd backend && go test ./...`
   - Frontend: `cd frontend && npm run check`

2. self-review スキルで自己検証

3. 必要に応じて update-docs スキルでドキュメント更新

4. create-pr スキルでPR作成

## PR作成後

- review-pr スキルでCI/Codexレビュー確認
- REQUEST_CHANGES は全て対応必須

## 禁止事項

| 禁止 | 理由 |
|------|------|
| ローカルCI未実行でpush | CIの失敗を防ぐ |
| レビュー結果を待たずにマージ | 品質担保 |
| REQUEST_CHANGESを無視 | 指摘は全て対応必須 |

## Git ルール

- コミットメッセージ: `<type>: <summary>` 形式
- type: feat, fix, refactor, docs, test, chore
- ブランチ: feature/, fix/, docs/ プレフィックス
- push前に必ずローカルCIを実行

## 参照

- [docs/rules/GIT_RULES.md](docs/rules/GIT_RULES.md)
- [docs/rules/CODEX_REVIEW.md](docs/rules/CODEX_REVIEW.md)
