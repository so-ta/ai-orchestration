# ブロック機能 UX改善プラン Phase 4

## 概要

Phase 1-3で実装した機能を踏まえ、さらなるUX向上のための改善プランを策定。

### 実装済み機能（Phase 1-3）
- ✅ 警告バッジの改善（未設定フィールド数表示）
- ✅ 必須フィールド警告の統一
- ✅ SetupWizard自動表示
- ✅ i18nハードコード修正
- ✅ ブロックプレビューツールチップ
- ✅ スマート検索（ファジーマッチング、エイリアス）

---

## Phase 4: Quick Wins（即効性の高い改善）

### 4.1 検索結果カウント表示
**優先度**: 高 | **工数**: 小

**現状の問題**:
- 検索実行時に結果数が表示されない
- ユーザーは検索が機能しているか分かりにくい

**改善内容**:
- パレット検索バーに「N件の結果」バッジを表示
- 0件の場合は「見つかりません」メッセージ

**対象ファイル**:
- `frontend/components/workflow-editor/StepPalette.vue`

---

### 4.2 検索マッチハイライト
**優先度**: 高 | **工数**: 小

**現状の問題**:
- 検索結果でどの部分がマッチしたか分からない
- 特にエイリアス検索時に混乱する

**改善内容**:
- ブロック名・説明文でマッチ箇所をハイライト
- `<mark>`タグでスタイリング

**対象ファイル**:
- `frontend/components/workflow-editor/palette/PaletteItem.vue`
- `frontend/composables/useBlocks.ts`（マッチ情報を返す）

---

### 4.3 ツールチップ表示速度の改善
**優先度**: 中 | **工数**: 極小

**現状の問題**:
- 400msの遅延は体感的に遅い
- ユーザーがホバーを止めてしまう

**改善内容**:
- 遅延を200msに短縮
- ドラッグ開始時は即座に非表示

**対象ファイル**:
- `frontend/components/workflow-editor/palette/PaletteItem.vue`

---

### 4.4 未設定フィールド名のツールチップ表示
**優先度**: 高 | **工数**: 小

**現状の問題**:
- 警告バッジは数だけ表示
- どのフィールドが未設定か分からない

**改善内容**:
- バッジホバー時に未設定フィールド名をリスト表示
- 「model, prompt, ...」のような形式

**対象ファイル**:
- `frontend/components/dag-editor/composables/useFlowNodes.ts`
- 新規: `frontend/components/dag-editor/WarningBadgeTooltip.vue`

---

### 4.5 ノードコンテキストメニュー
**優先度**: 高 | **工数**: 中

**現状の問題**:
- 右クリックで操作できない
- 複製・削除に複数ステップ必要

**改善内容**:
- 右クリックメニュー追加
  - 複製 (Ctrl+D)
  - 削除 (Delete)
  - 設定を開く
  - 接続を解除
- キーボードショートカット対応

**対象ファイル**:
- `frontend/components/dag-editor/DagEditor.vue`
- 新規: `frontend/components/dag-editor/NodeContextMenu.vue`

---

### 4.6 必須フィールドインジケーター強化
**優先度**: 中 | **工数**: 小

**現状の問題**:
- アスタリスク(*)が小さく見づらい
- 未入力時の視覚的フィードバックが弱い

**改善内容**:
- 必須フィールドラベルに赤ドットを追加
- 未入力かつtouched時にフィールド枠線を赤く
- ラベルに「必須」バッジオプション

**対象ファイル**:
- `frontend/components/workflow-editor/config/ConfigFieldRenderer.vue`
- `frontend/components/workflow-editor/config/widgets/*.vue`

---

## Phase 5: Medium Value（中程度の価値・工数）

### 5.1 お気に入りブロック機能
**優先度**: 中 | **工数**: 中

**現状の問題**:
- よく使うブロックを毎回検索する必要
- パーソナライズができない

**改善内容**:
- ブロックをお気に入り登録（localStorage保存）
- パレット最上部に「お気に入り」セクション
- 最近使用したブロック履歴（5件）

**対象ファイル**:
- 新規: `frontend/composables/useBlockFavorites.ts`
- `frontend/components/workflow-editor/StepPalette.vue`
- `frontend/components/workflow-editor/palette/PaletteItem.vue`

---

### 5.2 リアルタイムバリデーション
**優先度**: 高 | **工数**: 中

**現状の問題**:
- エラーはblur後にのみ表示
- 入力中のフィードバックがない

**改善内容**:
- 入力中（debounce 300ms）にバリデーション実行
- 有効な入力時は緑チェックマーク表示
- 無効な入力時は即座にエラー表示（touched関係なく）

**対象ファイル**:
- `frontend/components/workflow-editor/config/composables/useValidation.ts`
- `frontend/components/workflow-editor/config/widgets/*.vue`

---

### 5.3 ワークフロー完成度チェックリスト
**優先度**: 高 | **工数**: 中

**現状の問題**:
- ワークフローが「完成」かどうか分からない
- 公開前に何をすべきか不明確

**改善内容**:
- FloatingHeaderに完成度インジケーター追加
- チェックリストパネル
  - ✅ トリガー設定済み
  - ✅ 全ステップ設定完了
  - ✅ 全接続完了
  - ⚠️ テスト未実行
- クリックで問題箇所にジャンプ

**対象ファイル**:
- 新規: `frontend/components/workflow-editor/CompletionChecklist.vue`
- 新規: `frontend/composables/useWorkflowCompletion.ts`
- `frontend/components/editor/FloatingHeader.vue`

---

### 5.4 エッジラベル表示
**優先度**: 中 | **工数**: 中

**現状の問題**:
- 分岐条件がエッジ上で見えない
- true/false, case名などが不明確

**改善内容**:
- エッジ中央にラベルバッジ表示
- Condition: "true" / "false"
- Switch: case名
- Router: ルート名
- ホバーで詳細表示

**対象ファイル**:
- `frontend/components/dag-editor/composables/useFlowEdges.ts`
- `frontend/components/dag-editor/DagEditor.vue`

---

### 5.5 クイック編集モード
**優先度**: 中 | **工数**: 中

**現状の問題**:
- ステップ名変更にサイドパネルを開く必要
- 簡単な編集に手間がかかる

**改善内容**:
- ノードダブルクリックで名前をインライン編集
- Enter で確定、Escape でキャンセル
- フォーカス外れで自動保存

**対象ファイル**:
- `frontend/components/dag-editor/DagEditor.vue`
- `frontend/components/dag-editor/composables/useFlowNodes.ts`

---

### 5.6 フォーム入力進捗インジケーター
**優先度**: 中 | **工数**: 小

**現状の問題**:
- 長いフォームで残りフィールド数が不明
- 完了までの見通しが立たない

**改善内容**:
- PropertiesPanelヘッダーに進捗バー
- 「3/8 設定済み」のようなテキスト
- 必須フィールドのみカウント

**対象ファイル**:
- `frontend/components/workflow-editor/PropertiesPanel.vue`
- `frontend/components/workflow-editor/config/DynamicConfigForm.vue`

---

## Phase 6: High Value（高価値・高工数）

### 6.1 ブロック設定ウィザード
**優先度**: 高 | **工数**: 大

**現状の問題**:
- 複雑なブロック（LLM等）の設定が難しい
- 初心者には設定項目が多すぎる

**改善内容**:
- 複雑ブロック用のステップバイステップウィザード
- LLMブロック: モデル選択 → システムプロンプト → パラメータ
- 各ステップで説明とプレビュー
- スキップ可能（上級者向け）

**対象ブロック**:
- LLM, Router, Tool, HTTP, Subflow

**対象ファイル**:
- 新規: `frontend/components/workflow-editor/wizards/LLMWizard.vue`
- 新規: `frontend/components/workflow-editor/wizards/WizardBase.vue`

---

### 6.2 接続バリデーションフィードバック
**優先度**: 中 | **工数**: 中

**現状の問題**:
- 無効な接続を試みても分かりにくい
- なぜ接続できないか説明がない

**改善内容**:
- ドラッグ中に接続可能なノードをハイライト
- 無効なターゲットは赤枠・禁止アイコン
- 接続失敗時にトースト通知

**対象ファイル**:
- `frontend/components/dag-editor/DagEditor.vue`
- `frontend/components/dag-editor/composables/useFlowEdges.ts`

---

### 6.3 ブロックテンプレートライブラリ
**優先度**: 中 | **工数**: 大

**現状の問題**:
- 同じパターンを毎回最初から構築
- ベストプラクティスが共有されない

**改善内容**:
- 設定済みブロックをテンプレート保存
- システムテンプレート（公式パターン）
- ユーザーテンプレート（カスタム）
- パレットから直接追加

**対象ファイル**:
- 新規: `frontend/components/workflow-editor/palette/TemplateLibrary.vue`
- 新規: `frontend/composables/useBlockTemplates.ts`
- バックエンドAPI追加が必要

---

### 6.4 キーボードナビゲーション強化
**優先度**: 低 | **工数**: 中

**現状の問題**:
- マウス操作が主で効率が悪い
- アクセシビリティ対応が不十分

**改善内容**:
- Tab/Shift+Tabでノード間移動
- Enter で選択ノードの設定を開く
- Ctrl+矢印 でノード位置調整
- キーボードショートカット一覧パネル

**対象ファイル**:
- `frontend/components/dag-editor/DagEditor.vue`
- 新規: `frontend/composables/useKeyboardNavigation.ts`

---

### 6.5 ノードグループ化
**優先度**: 低 | **工数**: 大

**現状の問題**:
- 大規模ワークフローが見づらい
- 関連ステップをまとめられない

**改善内容**:
- 複数ノードを選択してグループ化
- グループの折りたたみ/展開
- グループ名とカラー設定
- サブフロー化オプション

**対象ファイル**:
- `frontend/components/dag-editor/DagEditor.vue`
- 新規: `frontend/components/dag-editor/NodeGroup.vue`
- バックエンドのGroup機能拡張

---

## 実装優先順位

### 即座に実装（1-2日）
1. **4.3** ツールチップ速度改善
2. **4.1** 検索結果カウント表示
3. **4.2** 検索マッチハイライト
4. **4.4** 未設定フィールド名ツールチップ

### 短期（1週間以内）
5. **4.6** 必須フィールドインジケーター強化
6. **4.5** ノードコンテキストメニュー
7. **5.6** フォーム入力進捗インジケーター

### 中期（2-3週間）
8. **5.3** ワークフロー完成度チェックリスト
9. **5.2** リアルタイムバリデーション
10. **5.1** お気に入りブロック機能
11. **5.4** エッジラベル表示
12. **5.5** クイック編集モード

### 長期（1ヶ月以上）
13. **6.1** ブロック設定ウィザード
14. **6.2** 接続バリデーションフィードバック
15. **6.3** ブロックテンプレートライブラリ
16. **6.4** キーボードナビゲーション強化
17. **6.5** ノードグループ化

---

## 技術的考慮事項

### パフォーマンス
- 検索ハイライトは仮想スクロール対応が必要な場合あり
- リアルタイムバリデーションはdebounce必須
- 大規模ワークフローでのコンテキストメニュー表示位置計算

### i18n対応
- 新規UI要素はすべてja.json/en.jsonに追加
- エラーメッセージの多言語化

### アクセシビリティ
- ARIA属性の適切な設定
- キーボード操作のサポート
- スクリーンリーダー対応

### テスト
- 新規composableのユニットテスト
- コンポーネントのスナップショットテスト
- E2Eテスト（Playwright）の追加検討

---

## 成功指標

| 指標 | 現状 | 目標 |
|------|------|------|
| ブロック追加までの平均時間 | 未計測 | 3秒以内 |
| 設定完了率 | 未計測 | 90%以上 |
| 検索使用率 | 未計測 | 50%以上 |
| ユーザーエラー発生率 | 未計測 | 20%削減 |

---

## 次のアクション

Phase 4の即座に実装可能な項目から着手することを推奨:

1. **4.3** ツールチップ速度改善 → 1行変更で完了
2. **4.1** 検索結果カウント表示 → StepPaletteに追加
3. **4.2** 検索マッチハイライト → PaletteItemの表示ロジック変更
4. **4.4** 未設定フィールド名ツールチップ → バッジコンポーネント拡張
