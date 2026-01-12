# DAG Editor Design Proposal: Minimal Linear

> Notion/Linear/Figma風のクリーンでミニマルなデザイン

## Design Principles

| 原則 | 説明 |
|------|------|
| **Subtle borders** | 1pxの薄いボーダー、目立ちすぎない |
| **Flat appearance** | シャドウは最小限、フラットな見た目 |
| **Color as accent** | 色はアクセントとして控えめに使用 |
| **Typography-first** | タイポグラフィで階層を表現 |
| **Generous spacing** | 十分な余白で読みやすさを確保 |

---

## Color Palette

### Base Colors
```css
--color-bg: #ffffff;
--color-bg-subtle: #fafafa;
--color-border: #e5e5e5;
--color-border-hover: #d4d4d4;
--color-text-primary: #171717;
--color-text-secondary: #737373;
--color-text-tertiary: #a3a3a3;
```

### Accent Colors (Step Types)
```css
--color-start: #22c55e;      /* Green - Start */
--color-llm: #8b5cf6;        /* Purple - AI/LLM */
--color-tool: #3b82f6;       /* Blue - Tool */
--color-condition: #f59e0b;  /* Amber - Condition */
--color-loop: #06b6d4;       /* Cyan - Loop */
--color-subflow: #ec4899;    /* Pink - Subflow */
--color-data: #6366f1;       /* Indigo - Data */
--color-error: #ef4444;      /* Red - Error */
```

### Group Colors
```css
--color-group-parallel: #8b5cf6;   /* Purple */
--color-group-try-catch: #ef4444; /* Red */
--color-group-if-else: #f59e0b;   /* Amber */
--color-group-foreach: #22c55e;   /* Green */
--color-group-while: #06b6d4;     /* Cyan */
```

---

## Block Design

### Before (Current)
```
┌─────────────────────┐
│ █████ LLM ████████ │  ← 色付きヘッダー全体
├─────────────────────┤
│  Block Name         │
└─────────────────────┘
  ↑ 2px border, box-shadow
```

### After (Minimal Linear)
```
┌─────────────────────┐
│ ● LLM              │  ← 小さなドットインジケーター
│   Block Name       │  ← タイポグラフィで階層表現
└─────────────────────┘
  ↑ 1px border, no shadow
```

### CSS Implementation

```css
/* Base Block Style */
.dag-node {
  background: #ffffff;
  border: 1px solid #e5e5e5;
  border-radius: 8px;
  min-width: 180px;
  box-shadow: none;
  transition: border-color 0.15s, background-color 0.15s;
}

.dag-node:hover {
  border-color: #d4d4d4;
  background-color: #fafafa;
}

.dag-node-selected {
  border-color: #3b82f6;
  box-shadow: 0 0 0 1px #3b82f6;
}

/* Block Header - Minimal */
.dag-node-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px 4px;
}

/* Type Indicator (small dot) */
.dag-node-indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

/* Type Label - Subtle */
.dag-node-type {
  font-size: 11px;
  font-weight: 500;
  color: #737373;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

/* Block Name */
.dag-node-label {
  padding: 4px 12px 12px;
  font-size: 14px;
  font-weight: 500;
  color: #171717;
  line-height: 1.4;
}
```

---

## Block Group Design

### Before (Current)
```
┌─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─┐
│ ███ PARALLEL ██████ │  ← 色付きヘッダー
│                     │
│   ┌───┐    ┌───┐    │
│   │ A │    │ B │    │
│   └───┘    └───┘    │
│                     │
└─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─┘
  ↑ dashed border
```

### After (Minimal Linear)
```
┌─────────────────────┐
│ ● Parallel          │  ← 薄いヘッダー、ドットアクセント
├─────────────────────┤
│                     │
│   ┌───┐    ┌───┐    │
│   │ A │    │ B │    │
│   └───┘    └───┘    │
│                     │
└─────────────────────┘
  ↑ solid 1px border, subtle tint background
```

### CSS Implementation

```css
/* Block Group - Minimal */
.dag-group {
  background: rgba(0, 0, 0, 0.01);
  border: 1px solid #e5e5e5;
  border-radius: 12px;
  position: relative;
}

.dag-group:hover {
  border-color: #d4d4d4;
}

.dag-group-selected {
  border-color: var(--group-color);
  box-shadow: 0 0 0 1px var(--group-color);
}

/* Group Header - Slim */
.dag-group-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-bottom: 1px solid #e5e5e5;
  background: rgba(0, 0, 0, 0.02);
  border-radius: 12px 12px 0 0;
}

/* Group Type Indicator */
.dag-group-indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--group-color);
}

/* Group Type Label */
.dag-group-type {
  font-size: 11px;
  font-weight: 600;
  color: #737373;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

/* Group Name */
.dag-group-name {
  font-size: 13px;
  font-weight: 500;
  color: #171717;
  margin-left: auto;
}

/* Section Dividers - Subtle */
.dag-group-divider-h {
  height: 1px;
  background: #e5e5e5;
  margin: 0 12px;
}

.dag-group-divider-v {
  width: 1px;
  background: #e5e5e5;
}

/* Section Labels */
.dag-group-section-label {
  font-size: 10px;
  font-weight: 600;
  color: var(--group-color);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  opacity: 0.8;
}
```

---

## Handles (Connection Points)

### Before
```css
/* 12px colored circles with visible border */
```

### After
```css
/* Smaller, subtle handles that appear on hover */
.dag-handle {
  width: 8px !important;
  height: 8px !important;
  background: #ffffff !important;
  border: 1.5px solid #d4d4d4 !important;
  border-radius: 50% !important;
  opacity: 0;
  transition: opacity 0.15s, transform 0.15s, border-color 0.15s;
}

.dag-node:hover .dag-handle,
.dag-group:hover .dag-handle {
  opacity: 1;
}

.dag-handle:hover {
  border-color: #3b82f6 !important;
  background: #3b82f6 !important;
  transform: scale(1.25);
}
```

---

## Edges (Connections)

```css
/* Subtle gray edges */
:deep(.vue-flow__edge-path) {
  stroke: #d4d4d4;
  stroke-width: 1.5;
}

:deep(.vue-flow__edge.selected .vue-flow__edge-path) {
  stroke: #3b82f6;
  stroke-width: 2;
}

:deep(.vue-flow__edge:hover .vue-flow__edge-path) {
  stroke: #a3a3a3;
}
```

---

## Status Indicators

実行状態の表示はよりサブタイルに:

```css
/* Status badge - smaller, positioned inside */
.dag-node-status {
  position: absolute;
  top: 8px;
  right: 8px;
  width: 6px;
  height: 6px;
  border-radius: 50%;
}

/* Or use a subtle border glow */
.dag-node-running {
  border-color: #3b82f6;
  animation: pulse-border 2s ease-in-out infinite;
}

.dag-node-completed {
  border-left: 3px solid #22c55e;
}

.dag-node-failed {
  border-left: 3px solid #ef4444;
}

@keyframes pulse-border {
  0%, 100% { box-shadow: 0 0 0 0 rgba(59, 130, 246, 0.4); }
  50% { box-shadow: 0 0 0 4px rgba(59, 130, 246, 0); }
}
```

---

## Background Grid

```css
/* Subtle dot grid */
.dag-editor {
  background-color: #fafafa;
  background-image: radial-gradient(circle, #e5e5e5 1px, transparent 1px);
  background-size: 24px 24px;
}
```

---

## Visual Comparison

### Current Design
- 強い色のヘッダーバー
- 目立つシャドウ
- 破線ボーダー（グループ）
- 大きなハンドル

### Minimal Linear Design
- 小さなドットインジケーター
- シャドウなし/最小限
- ソリッドな薄いボーダー
- ホバー時のみ表示されるハンドル
- タイポグラフィによる階層表現

---

## Implementation Notes

1. **段階的な移行**: 一度に全て変更せず、まずノードから始める
2. **カラー変数**: CSS変数を活用して一貫性を保つ
3. **ホバー状態**: 微妙な変化で操作可能性を示す
4. **アクセシビリティ**: コントラスト比を維持する

---

## Approval Checklist

- [ ] カラーパレットの確認
- [ ] ブロックデザインの確認
- [ ] グループデザインの確認
- [ ] ハンドル/エッジのスタイル確認
- [ ] 実装開始の承認
