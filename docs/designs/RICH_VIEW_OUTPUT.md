# Rich View Output 設計書

> **Status**: ✅ 実装済み
> **Updated**: 2025-01-13

## 概要

Block出力をExtended Markdown形式で表現し、リッチな可視化を可能にする機能。

**目的:**
- 処理の共通化（すべてMarkdownパーサーを通る）
- 複雑なデータ構造を視覚的に理解しやすく表示
- LLMが直接出力できる形式との親和性
- フォールバック可能（拡張未対応でもテキスト表示）

---

## アーキテクチャ

```
BlockOutput.data
    ↓
Markdown変換（共通処理）
    ↓
Extended Markdownレンダラー
    ↓
表示
```

**メリット:**
1. **共通処理**: すべてMarkdownパーサーを通る
2. **フォールバック**: 拡張未対応でもJSONとして表示される
3. **コピー可能**: テキストとしてコピー＆ペーストできる
4. **LLM親和性**: LLMが直接この形式で出力できる
5. **段階的実装**: 拡張コンポーネントを順次追加可能

---

## 出力スキーマ

### BlockOutput

```typescript
interface BlockOutput {
  data: any;                    // 実際のデータ（後続ブロック用）
  markdown?: string;            // Extended Markdown形式の表示用出力
}
```

---

## View Types

### 標準Markdown変換

コードブロックのlanguage指定を使わず、標準Markdownで表現できるもの。

| Type | 変換結果 | 説明 |
|------|----------|------|
| markdown | そのまま | Markdownテキスト |
| table | `\| col \| col \|` | Markdownテーブル |
| image | `![alt](src)` | 画像埋め込み |
| code | ` ```lang ``` ` | コードブロック |
| key-value | `**Label:** value` | Key-Value形式 |

---

### 1. table

Markdownテーブル形式で表現。

**変換例:**
```markdown
| 名前 | メール | 状態 |
|------|--------|------|
| 田中太郎 | tanaka@example.com | Active |
| 山田花子 | yamada@example.com | Pending |
```

**表示イメージ:**
```
┌──────────┬──────────────────────┬─────────┐
│ 名前     │ メール               │ 状態    │
├──────────┼──────────────────────┼─────────┤
│ 田中太郎 │ tanaka@example.com   │ Active  │
│ 山田花子 │ yamada@example.com   │ Pending │
└──────────┴──────────────────────┴─────────┘
```

---

### 2. image

Markdown画像構文で表現。

**変換例:**
```markdown
![生成画像](https://example.com/image.png)
```

---

### 3. code

コードブロックで表現。言語指定によりシンタックスハイライト。

**変換例:**
````markdown
```sql
SELECT * FROM users WHERE status = 'active';
```
````

---

### 4. key-value

太字ラベルと値のペアで表現。

**変換例:**
```markdown
**Status:** 200 OK
**Latency:** 145ms
**URL:** https://api.example.com/users
```

**表示イメージ:**
```
Status:   200 OK
Latency:  145ms
URL:      https://api.example.com/users
```

---

## 拡張構文

コードブロックのlanguage指定を使い、標準Markdownでは表現できないリッチな表示を実現。

### 1. chart

グラフ形式でデータを可視化。

**構文:**
````markdown
```chart
{
  "type": "bar",
  "labels": ["Jan", "Feb", "Mar", "Apr"],
  "datasets": [
    { "label": "売上", "data": [100, 200, 150, 300] }
  ]
}
```
````

**プロパティ:**

| プロパティ | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| type | `"bar" \| "line" \| "pie" \| "doughnut"` | ✅ | グラフ種類 |
| labels | `string[]` | ✅ | X軸ラベル（pie/doughnutでは凡例） |
| datasets | `Dataset[]` | ✅ | データセット配列 |
| height | `number` | | 高さ（px、デフォルト: 300） |
| stacked | `boolean` | | 積み上げ表示（bar, line） |
| showLegend | `boolean` | | 凡例表示（デフォルト: true） |

**Dataset:**
```typescript
interface Dataset {
  label?: string;      // 凡例ラベル
  data: number[];      // 値配列（labelsと同じ長さ）
  color?: string;      // カスタム色（CSS color）
}
```

**使用例:**

````markdown
```chart
{
  "type": "bar",
  "labels": ["GPT-4o", "Claude-3.5", "Gemini"],
  "datasets": [
    { "label": "Latency (ms)", "data": [1250, 980, 1100] },
    { "label": "Cost ($)", "data": [0.008, 0.006, 0.005] }
  ]
}
```
````

**表示イメージ:**
```
┌─────────────────────────────────────────────┐
│                                             │
│     ████                                    │
│     ████  ████                              │
│     ████  ████  ████                        │
│     ████  ████  ████  ████                  │
│     ────────────────────────                │
│     Jan   Feb   Mar   Apr                   │
│                                             │
│     ■ 売上                                  │
└─────────────────────────────────────────────┘
```

**フォールバック（拡張未対応時）:**
```
chart
{
  "type": "bar",
  "labels": ["Jan", "Feb", "Mar", "Apr"],
  ...
}
```

---

### 2. progress

プログレスバーを表示。

**構文:**
````markdown
```progress
{ "value": 67, "label": "処理進捗" }
```
````

**プロパティ:**

| プロパティ | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| value | `number` | ✅ | 現在値（0-100） |
| label | `string` | | ラベルテキスト |
| color | `string \| "auto"` | | 色（autoは値で変化: 緑→黄→赤） |
| size | `"sm" \| "md" \| "lg"` | | サイズ（デフォルト: md） |

**使用例:**

````markdown
```progress
{ "value": 67, "label": "処理進捗", "color": "auto" }
```
````

**表示イメージ:**
```
┌─────────────────────────────────────────────┐
│ 処理進捗                                     │
│ ████████████████████░░░░░░░░░░  67%         │
└─────────────────────────────────────────────┘
```

**フォールバック（拡張未対応時）:**
```
処理進捗: 67%
```

---

## 実装例：LLM Compare Block

```javascript
// ブロック出力例
const output = {
  data: {
    results: [
      { model: "GPT-4o", response: "...", latency_ms: 1250, cost_usd: 0.008 },
      { model: "Claude-3.5", response: "...", latency_ms: 980, cost_usd: 0.006 }
    ]
  },
  markdown: `
## Model Comparison Results

**Fastest:** Claude-3.5 (980ms)
**Cheapest:** Claude-3.5 ($0.006)

\`\`\`chart
{
  "type": "bar",
  "labels": ["GPT-4o", "Claude-3.5"],
  "datasets": [
    { "label": "Latency (ms)", "data": [1250, 980] },
    { "label": "Cost ($)", "data": [0.008, 0.006] }
  ]
}
\`\`\`

### GPT-4o Response
...

### Claude-3.5 Response
...
`
};
```

---

## UI設計

### タブ切り替え

```
┌─────────────────────────────────────────────────────────────┐
│ Step Output                                                 │
├─────────────────────────────────────────────────────────────┤
│  ┌──────┬──────────┐                                        │
│  │ JSON │ Markdown │  ← markdown が定義されている場合のみ    │
│  └──────┴──────────┘                                        │
│   ▲ デフォルト                                               │
└─────────────────────────────────────────────────────────────┘
```

### Markdownタブ表示

```
┌─────────────────────────────────────────────────────────────┐
│  ┌──────┬──────────┐                                        │
│  │ JSON │ Markdown │                                        │
│  └──────┴──────────┘                                        │
│            ▲                                                 │
│                                                             │
│  ## Model Comparison Results                                │
│                                                             │
│  **Fastest:** Claude-3.5 (980ms)                            │
│  **Cheapest:** Claude-3.5 ($0.006)                          │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │     ████  ████                                       │   │
│  │     ████  ████                                       │   │
│  │     ────────────                                     │   │
│  │     GPT-4o  Claude                                   │   │
│  │     ■ Latency  ■ Cost                               │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ### GPT-4o Response                                        │
│  ...                                                        │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## 実装工数

| タスク | 工数 |
|--------|------|
| TypeScript型定義 | 0.5日 |
| ExtendedMarkdownRenderer | 1日 |
| ChartBlock（Chart.js統合） | 1日 |
| ProgressBlock | 0.5日 |
| タブ切り替えUI | 0.5日 |
| テスト | 0.5日 |
| **合計** | **4日** |

---

## 関連ドキュメント

- [UNIFIED_BLOCK_MODEL.md](UNIFIED_BLOCK_MODEL.md) - ブロック実行アーキテクチャ
- [BACKEND.md](../BACKEND.md) - バックエンドアーキテクチャ
- [FRONTEND.md](../FRONTEND.md) - フロントエンドアーキテクチャ
