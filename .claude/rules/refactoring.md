# Refactoring Rules

コードリファクタリングのルール。

## コンポーネント分割

詳細は [docs/rules/COMPONENT_REFACTORING.md](../../docs/rules/COMPONENT_REFACTORING.md) を参照。

### 分割基準

| セクション | 閾値 | 対応 |
|------------|------|------|
| 総行数 | >500行 | 分割検討 |
| 総行数 | >1000行 | 分割必須 |
| script setup | >200行 | composable抽出 |
| template | >300行 | 子コンポーネント分離 |

### 優先順位

1. Composables抽出（ロジック分離）
2. 読み取り専用セクション分離
3. フォームセクション分離（v-model）
4. スタイル分離

## 禁止事項

| 禁止 | 理由 |
|------|------|
| 外部インターフェース変更 | 下位互換性を維持 |
| Props/Emits変更 | 親コンポーネントへの影響 |
| 検証なしのpush | 品質担保 |

## 検証

リファクタリング後は必ず以下を実行:

```bash
cd frontend && npm run check
```
