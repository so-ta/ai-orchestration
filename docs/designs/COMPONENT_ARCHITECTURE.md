# コンポーネントアーキテクチャ

このドキュメントでは、AI Orchestrationのフロントエンドコンポーネントアーキテクチャについて説明します。

## 概要

フロントエンドは Vue 3 + Nuxt 3 で構築されており、Composition API を使用したモジュラーな設計を採用しています。

## ワークフローエディタ構成

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           WorkflowEditor                                 │
│                                                                          │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────┐  │
│  │                 │  │                 │  │                         │  │
│  │    DagEditor    │  │  PropertiesPanel│  │     CopilotPanel        │  │
│  │                 │  │                 │  │                         │  │
│  │  ┌───────────┐  │  │  ┌───────────┐  │  │  ┌───────────────────┐  │  │
│  │  │ VueFlow   │  │  │  │ StepForm  │  │  │  │ ChatInterface     │  │  │
│  │  │ Canvas    │  │  │  │           │  │  │  │                   │  │  │
│  │  └───────────┘  │  │  └───────────┘  │  │  └───────────────────┘  │  │
│  │                 │  │                 │  │                         │  │
│  │  ┌───────────┐  │  │  ┌───────────┐  │  │  ┌───────────────────┐  │  │
│  │  │ Minimap   │  │  │  │ EdgeForm  │  │  │  │ ProposalCard      │  │  │
│  │  └───────────┘  │  │  └───────────┘  │  │  └───────────────────┘  │  │
│  │                 │  │                 │  │                         │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────────────┘  │
│                                                                          │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │                        BlockPalette                                │  │
│  └───────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

## 主要コンポーネント

### WorkflowEditor

ワークフロー編集の親コンポーネント。状態管理とコンポーネント間の通信を担当。

| パス | 責務 |
|------|------|
| `components/workflow-editor/WorkflowEditor.vue` | 状態管理、コマンドパターン |

### DagEditor

DAG（有向非巡回グラフ）の視覚的編集を担当。

| パス | 責務 |
|------|------|
| `components/dag-editor/DagEditor.vue` | VueFlow統合、ノード/エッジレンダリング |
| `components/dag-editor/composables/useFlowNodes.ts` | ノード変換 |
| `components/dag-editor/composables/useFlowEdges.ts` | エッジ変換 |
| `components/dag-editor/composables/useVirtualization.ts` | 大規模ワークフロー最適化 |

### PropertiesPanel

選択されたステップ/エッジのプロパティ編集。

| パス | 責務 |
|------|------|
| `components/workflow-editor/PropertiesPanel.vue` | フォーム表示、設定編集 |

### CopilotPanel

AI アシスタント機能の UI。

| パス | 責務 |
|------|------|
| `components/workflow-editor/CopilotTab.vue` | チャットUI、提案表示 |
| `components/workflow-editor/CopilotProposalCard.vue` | 変更提案カード |

## 通信パターン

### イベントフロー

```
1. DagEditor -> WorkflowEditor
   emit('node-selected', stepId)

2. WorkflowEditor -> PropertiesPanel
   :selected-step="selectedStep"

3. CopilotPanel -> WorkflowEditor
   emit('apply-draft', changes)

4. WorkflowEditor -> DagEditor
   :preview-state="previewState"
```

### データフロー

```
                    ┌──────────────────┐
                    │    API Server    │
                    └────────┬─────────┘
                             │
                    ┌────────▼─────────┐
                    │   Composables    │
                    │ (useProjects,    │
                    │  useCopilot)     │
                    └────────┬─────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
     ┌────────▼────────┐ ┌───▼────┐ ┌──────▼───────┐
     │ WorkflowEditor  │ │ Store  │ │ CommandHistory│
     └────────┬────────┘ └───┬────┘ └──────┬───────┘
              │              │              │
              └──────────────┴──────────────┘
                             │
                    ┌────────▼─────────┐
                    │   Child Comps    │
                    └──────────────────┘
```

## Composable 構成

### 状態管理系

| Composable | 責務 |
|------------|------|
| `useProjects` | プロジェクトCRUD |
| `useSteps` | ステップ操作 |
| `useEdges` | エッジ操作 |
| `useBlockDefinitions` | ブロック定義取得 |

### Copilot 系

| Composable | 責務 |
|------------|------|
| `useCopilot` | エントリポイント（re-export） |
| `copilot/types` | 型定義 |
| `useCopilotDraft` | ドラフト/プレビュー管理 |

### Editor 系

| Composable | 責務 |
|------------|------|
| `useCommandHistory` | Undo/Redo |
| `useFlowNodes` | VueFlow ノード変換 |
| `useFlowEdges` | VueFlow エッジ変換 |
| `useVirtualization` | 仮想化（大規模対応） |

## パフォーマンス最適化

### 仮想化

大規模ワークフロー（100+ノード）では、viewport 外のノードをレンダリングしない仮想化を適用。

```typescript
// useVirtualization.ts
const visibleBounds = computed(() => {
  // viewport + buffer 範囲のみレンダリング
})

function isNodeVisible(node: Node): boolean {
  // bounds 内かチェック
}
```

### キャッシング

ノードデータはタイムスタンプベースでキャッシュ:

```typescript
class NodeCache<T> {
  getOrCompute(key: string, timestamp: string, compute: () => T): T
}
```

## ベストプラクティス

### Props/Emits

```vue
<script setup lang="ts">
interface Props {
  projectId: string
  selectedStepId?: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'step-selected': [stepId: string]
  'step-updated': [step: Step]
}>()
</script>
```

### Computed vs Watch

- 派生データには `computed` を使用
- 副作用には `watch` を使用
- `watchEffect` は依存関係が明確な場合のみ

### エラーハンドリング

```typescript
const { data, error, pending } = await useFetch('/api/...')

// テンプレートで
<div v-if="pending">Loading...</div>
<div v-else-if="error">Error: {{ error.message }}</div>
<div v-else>{{ data }}</div>
```

## 関連ドキュメント

- [FRONTEND.md](../FRONTEND.md) - フロントエンド開発ガイド
- [DAG_EDITOR.md](./DAG_EDITOR.md) - DAG エディタ詳細
- [COPILOT.md](../COPILOT.md) - Copilot 機能
