# ドキュメントガイドライン

ドキュメント作成・管理・同期の統合ガイド。

> **Status**: Active
> **Updated**: 2026-01-15
> **Related**: [INDEX.md](./INDEX.md)

---

## ドキュメント原則

### 1. 単一の情報源（Single Source of Truth）

各情報の正は 1 箇所のみ。他の場所では参照のみ。

| 情報 | 正（Source of Truth） |
|------|---------------------|
| 実装状態 | [INDEX.md](./INDEX.md) の Implementation Status 表 |
| ブロック API 仕様 | [BLOCK_REGISTRY.md](./BLOCK_REGISTRY.md) |
| ブロック設計思想 | [UNIFIED_BLOCK_MODEL.md](./designs/UNIFIED_BLOCK_MODEL.md) |
| DB スキーマ | [DATABASE.md](./DATABASE.md) |
| API エンドポイント | [API.md](./API.md) |

### 2. MECE（相互排他的・全体網羅）

- **相互排他的（Mutually Exclusive）**: ドキュメント間に重複なし
- **全体網羅（Collectively Exhaustive）**: 全情報をどこかに記載

### 3. 例で示す（Show, Don't Tell）

ルールだけでなく、正しい例と間違った例を必ず添える。

---

## ドキュメント階層

```
CLAUDE.md (エントリーポイント)
    ↓
docs/INDEX.md (ナビゲーション)
    ├── Technical Documentation
    │   ├── BACKEND.md      # Go 構造 + Canonical Patterns
    │   ├── FRONTEND.md     # Vue 構造 + Canonical Patterns
    │   ├── API.md          # REST API 仕様
    │   ├── DATABASE.md     # DB スキーマ + Query Patterns
    │   ├── DEPLOYMENT.md   # デプロイ・環境
    │   ├── BLOCK_REGISTRY.md  # ブロック仕様（API）
    │   └── TROUBLESHOOTING.md # エラー対処法
    │
    ├── Development Rules
    │   ├── rules/WORKFLOW_RULES.md  # 開発ルール
    │   ├── rules/GIT_RULES.md       # Git ルール
    │   ├── rules/CODEX_REVIEW.md    # レビュールール
    │   └── DOCUMENTATION.md         # このファイル
    │
    ├── Testing
    │   └── TESTING.md       # テスト統合ガイド
    │
    └── Architecture
        └── designs/*.md     # 設計ドキュメント
```

---

## 更新マッピング（コード変更時の更新ルール）

### コード変更 → ドキュメント更新

| 変更内容 | 更新必須ドキュメント |
|----------|---------------------|
| 新規ブロック追加 | BLOCK_REGISTRY.md |
| DB スキーマ変更 | DATABASE.md |
| API エンドポイント追加 | API.md, openapi.yaml |
| バックエンド構造変更 | BACKEND.md |
| フロントエンド構造変更 | FRONTEND.md |
| 新規機能完了 | INDEX.md (Implementation Status) |
| 開発ルール変更 | rules/*.md |

### ドキュメント更新チェックリスト

コード変更時に確認：

```
1. [ ] 影響するドキュメントを特定
2. [ ] 該当ドキュメントを更新
3. [ ] 関連する参照を確認（壊れたリンクがないか）
4. [ ] 例・サンプルコードが最新か確認
```

---

## ドキュメント作成ルール

### 新規ドキュメントを作成する基準

以下の場合のみ新規ドキュメントを作成：

| ケース | 作成可否 |
|--------|---------|
| 新しい設計が必要 | ✅ designs/ に作成 |
| 新しい機能計画 | ✅ plans/ に作成 |
| 既存ドキュメントが巨大化 | ✅ 分割を検討 |
| 一時的なメモ | ❌ ドキュメント化不要 |
| 特定 PR の説明 | ❌ PR description に記載 |

### ドキュメントテンプレート

```markdown
# [Document Title]

[1-2行の説明]

> **Status**: Draft | Active | Deprecated
> **Updated**: YYYY-MM-DD
> **Related**: [関連ドキュメント](./xxx.md)

---

## Overview

[概要説明]

## Quick Reference

| Item | Value |
|------|-------|
| xxx | yyy |

## [Main Sections]

...

## Related Documents

- [Doc1](./doc1.md) - 説明
- [Doc2](./doc2.md) - 説明
```

---

## 執筆ガイドライン

### Claude Code 最適化ルール

1. **明確な構造**: 表・リスト・コードブロックを活用
2. **具体例必須**: 抽象的な説明だけでなく、コード例を添える
3. **禁止事項を明示**: 「やってはいけないこと」を明確に
4. **Why を説明**: ルールの背景・理由を記載

### 良い例

```markdown
## Handler パターン

```go
// ✅ 正しいパターン
func (h *Handler) Create(c echo.Context) error {
    ctx := c.Request().Context()
    // ...
}

// ❌ 禁止パターン
func (h *Handler) Create(c echo.Context) error {
    ctx := context.Background()  // NG: トレース途切れ
    // ...
}
```

**Why**: `c.Request().Context()` を使わないと OpenTelemetry トレースが途切れる
```

### 悪い例

```markdown
## Handler パターン

Handler は適切にコンテキストを扱ってください。

<!-- 具体例がない、何が「適切」かわからない -->
```

---

## ドキュメントメンテナンス

### 定期レビュー

| 頻度 | 対象 | 確認内容 |
|------|------|---------|
| PR ごと | 関連ドキュメント | 変更に追従しているか |
| 月次 | 全ドキュメント | 古い情報がないか |
| 四半期 | 構造 | 再編成が必要か |

### 廃止ドキュメントの処理

```markdown
# [Deprecated Document Title]

> **⚠️ DEPRECATED**: このドキュメントは廃止されました。
> **代替**: [新しいドキュメント](./new-doc.md) を参照してください。
> **廃止日**: YYYY-MM-DD
```

---

## ドキュメント依存関係

### 参照関係図

```
CLAUDE.md
    → INDEX.md
        → BACKEND.md → API.md, DATABASE.md
        → FRONTEND.md → API.md
        → BLOCK_REGISTRY.md → UNIFIED_BLOCK_MODEL.md
        → TESTING.md → BACKEND.md, FRONTEND.md
        → rules/WORKFLOW_RULES.md → TROUBLESHOOTING.md
```

### 循環参照の禁止

A → B → C → A のような循環参照は禁止。
常に一方向の参照関係を維持する。

---

## 矛盾の解決

ドキュメント間で矛盾を発見した場合：

### 解決手順

1. **正を特定**: 上記「Single Source of Truth」表を参照
2. **正のドキュメントが正しいか確認**: コードと照合
3. **正のドキュメントに合わせて他を修正**
4. **修正履歴を残す**

### 報告フォーマット

```markdown
## Discrepancy Report

**発見日**: YYYY-MM-DD
**発見場所**: [ドキュメントA](./a.md) vs [ドキュメントB](./b.md)
**矛盾内容**: A では X だが、B では Y と記載
**正**: A（コードと照合済み）
**対処**: B を修正済み
```

---

## AI駆動ドキュメンテーション

このプロジェクトは AI エージェントが実装・保守する。
ドキュメントも AI エージェントが更新する前提で設計されている。

### AI エージェントへの要件

1. **コード変更時は必ずドキュメント更新**
2. **暗黙の知識を残さない**: 口頭説明や「見ればわかる」は禁止
3. **後続エージェントがコンテキストを失わないよう記録**
4. **セッション終了時に未完了事項を明記**

### ドキュメント品質チェック

PR 作成時に確認：

```
1. [ ] 関連ドキュメントを更新したか
2. [ ] 例・サンプルコードが動作するか
3. [ ] リンクが切れていないか
4. [ ] 矛盾する記述がないか
5. [ ] 後続 AI エージェントが理解できるか
```

---

## 関連ドキュメント

- [INDEX.md](./INDEX.md) - ドキュメントナビゲーション
- [WORKFLOW_RULES.md](./rules/WORKFLOW_RULES.md) - 開発ワークフロー
- [GIT_RULES.md](./rules/GIT_RULES.md) - コミット・PR ルール
- [CODEX_REVIEW.md](./rules/CODEX_REVIEW.md) - レビュールール
