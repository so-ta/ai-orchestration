# Documentation Sync Rules

コードとドキュメントの同期ルール。

---

## アーキテクチャ変更時の更新対象マッピング

| 変更内容 | 更新必須ドキュメント |
|---------|-------------------|
| 新規ブロック/外部連携追加 | `BLOCK_REGISTRY.md`, `UNIFIED_BLOCK_MODEL.md`（参照のみ） |
| DBスキーマ変更 | `DATABASE.md` |
| API追加/変更 | `API.md`, `openapi.yaml` |
| バックエンド構造変更 | `BACKEND.md` |
| フロントエンド構造変更 | `FRONTEND.md` |
| 認証/認可変更 | `DEPLOYMENT.md` |

---

## ドキュメント更新チェックリスト

コード変更完了後、以下を確認：

```
1. [ ] 変更した機能に関連するドキュメントを特定
2. [ ] 該当ドキュメントの記載が実装と一致しているか確認
3. [ ] 古い情報があれば更新（「実装済み」等のステータス更新含む）
4. [ ] 相互参照リンクが正しいか確認
5. [ ] docs/INDEX.md に新ドキュメントの参照があるか確認
```

---

## 齟齬防止のための設計原則

| 原則 | 説明 |
|------|------|
| **Single Source of Truth** | 同じ情報を複数箇所に書かない。参照リンクを使う |
| **Status明記** | 設計書には `Status: ✅ 実装済み / 📋 未実装` を明記 |
| **Updated日付** | 重要なドキュメントには更新日を記載 |
| **関連ドキュメントリンク** | Related Documents セクションで相互リンク |

---

## ドキュメントカテゴリと役割

| カテゴリ | パス | 役割 | 実装後の扱い |
|---------|------|------|-------------|
| **正式ドキュメント** | `docs/*.md` | 実装済み機能の仕様 | 維持・更新 |
| **設計書** | `docs/designs/*.md` | アーキテクチャ設計 | 参考資料として残す |
| **プラン** | `docs/plans/*.md` | 未実装の将来計画 | 実装後は正式ドキュメントに統合し削除 |
| **実装コード** | `backend/schema/seed.sql` | ブロック定義のSingle Source of Truth | 常に最新 |

---

## 実装完了時のドキュメント移行フロー

```
1. プラン (plans/*.md) で機能を設計
   ↓
2. 実装完了
   ↓
3. 正式ドキュメントに仕様を統合
   - Step Types → BACKEND.md
   - コンポーネント → FRONTEND.md
   - API → API.md, openapi.yaml
   - ブロック定義 → BLOCK_REGISTRY.md (参照のみ、コードはseed.sql)
   ↓
4. プランから詳細仕様を削除（参照リンクに置換）
   ↓
5. 設計書のStatusを「✅ 実装済み」に更新
```

---

## 齟齬発見時の対応フロー

```
1. 齟齬を発見
   ↓
2. どちらが正しいか確認（実装 vs ドキュメント）
   ↓
3. 実装が正しい場合 → ドキュメントを更新
   ドキュメントが正しい場合 → 実装を修正（またはユーザーに確認）
   ↓
4. 関連ドキュメントも同時に確認・更新
   ↓
5. 齟齬の原因を分析し、再発防止策をCLAUDE.mdに追記
```

---

## ドキュメント間の依存関係

```
CLAUDE.md (エントリーポイント)
  ├── docs/INDEX.md (ナビゲーション)
  │     ├── BACKEND.md
  │     ├── FRONTEND.md
  │     ├── API.md
  │     ├── DATABASE.md
  │     └── designs/UNIFIED_BLOCK_MODEL.md ← BLOCK_REGISTRY.md
  │
  └── 変更時は上記の依存関係を考慮して更新
```

**重要**: アーキテクチャに関わる変更（Unified Block Model等）は、複数ドキュメントに影響する。
変更前に影響範囲を確認し、すべての関連ドキュメントを更新すること。

---

## When Documentation Missing

| Situation | Action |
|-----------|--------|
| No doc for area being modified | Create new doc following DOCUMENTATION_RULES.md |
| Existing doc incomplete | Update existing doc |
| Code contradicts doc | Fix code OR update doc (confirm intent first) |

---

## Documentation Priority

1. **MUST document**: Public interfaces, API changes, config changes
2. **SHOULD document**: Internal architecture decisions, non-obvious patterns
3. **MAY skip**: Trivial implementation details (use code comments)

---

## AIフレンドリードキュメント要件

後続のAIエージェントが即座にコンテキストを把握できるよう、以下を遵守：

### 1. 明示的な記述

- 暗黙知を排除し、すべてを文書化
- 「なぜ」その設計・実装にしたかを記録
- 制約条件や前提条件を明記

### 2. 構造化された情報

- テーブル形式での情報整理を優先
- コードブロックでの具体例提示
- 階層的な見出し構造

### 3. 参照可能性

- ファイルパスは絶対パスまたはプロジェクトルートからの相対パス
- 関連ドキュメントへのリンクを明記
- 検索可能なキーワードを含める

### 4. 最新性の維持

- コード変更時は必ずドキュメント更新
- 古い情報は削除または更新日を明記
- バージョン管理との整合性を保つ

---

## Related Documents

- [DOCUMENTATION_RULES.md](../DOCUMENTATION_RULES.md) - ドキュメント作成ルール
- [INDEX.md](../INDEX.md) - ドキュメントナビゲーション
