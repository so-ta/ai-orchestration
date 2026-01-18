# フロントエンド技術リファレンス

Vue 3 / Nuxt 3 フロントエンドの構造、Composables、コンポーネント。

## クイックリファレンス

| 項目 | 値 |
|------|-------|
| フレームワーク | Nuxt 3 + Vue 3 |
| 言語 | TypeScript |
| 状態管理 | Composables（Vue 3 Composition API） |
| DAG エディタ | @vue-flow/core |
| 認証 | Keycloak OIDC |
| ページ | `pages/`（ファイルベースルーティング） |
| コンポーネント | `components/` |
| Composables | `composables/` |
| 型定義 | `types/api.ts` |

## ディレクトリ構造

```
frontend/
├── app.vue                 # ルートコンポーネント
├── nuxt.config.ts          # Nuxt 設定
├── pages/                  # ファイルベースルーティング
│   ├── index.vue           # ダッシュボード /
│   ├── workflows/
│   │   ├── index.vue       # 一覧 /workflows
│   │   ├── new.vue         # 作成 /workflows/new
│   │   └── [id].vue        # 詳細 /workflows/:id
│   ├── runs/
│   │   ├── index.vue       # 一覧 /runs
│   │   └── [id].vue        # 詳細 /runs/:id
│   ├── schedules/
│   │   └── index.vue       # 一覧 /schedules
│   └── settings/
│       └── index.vue       # 設定 /settings
├── components/
│   └── dag-editor/         # DAG ビジュアルエディタ
├── composables/            # Vue 3 Composables
│   ├── useAuth.ts          # Keycloak 認証
│   ├── useApi.ts           # API クライアント
│   ├── useProjects.ts      # プロジェクト操作
│   ├── useRuns.ts          # Run 操作
│   ├── useBlocks.ts        # ブロック定義・検索ユーティリティ
│   ├── useBlockSearch.ts   # ブロック検索（共通composable）
│   ├── useStoredInput.ts   # localStorage入力値永続化
│   ├── usePolling.ts       # ポーリングロジック
│   ├── useTemplateVariables.ts # テンプレート変数処理
│   └── ...                 # その他のcomposables
├── plugins/
│   └── auth.client.ts      # Keycloak 初期化
├── layouts/
│   └── default.vue         # ベースレイアウト
├── types/
│   └── api.ts              # TypeScript インターフェース
└── assets/css/
    └── main.css            # グローバルスタイル
```

## 設定 (nuxt.config.ts)

```typescript
export default defineNuxtConfig({
  runtimeConfig: {
    public: {
      apiBase: 'http://localhost:8090',
      keycloak: {
        url: 'http://localhost:8180',
        realm: 'ai-orchestration',
        clientId: 'frontend'
      }
    }
  }
})
```

## 型定義 (types/api.ts)

```typescript
interface Workflow {
  id: string
  name: string
  description: string
  status: 'draft' | 'published'
  version: number
  input_schema: object
  created_at: string
  updated_at: string
}

interface Step {
  id: string
  workflow_id: string
  name: string
  type: 'llm' | 'tool' | 'condition' | 'map' | 'join' | 'subflow'
  config: object
  position: { x: number; y: number }
}

interface Edge {
  id: string
  workflow_id: string
  source_step_id: string
  target_step_id: string
  condition?: string
}

interface Run {
  id: string
  workflow_id: string
  workflow_version: number
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'
  mode: 'test' | 'production'
  trigger_type: 'manual' | 'schedule' | 'webhook'
  input: object
  output?: object
  error?: string
  started_at?: string
  completed_at?: string
  created_at: string
  step_runs?: StepRun[]
}

interface StepRun {
  id: string
  run_id: string
  step_id: string
  step_name: string
  status: 'pending' | 'running' | 'completed' | 'failed'
  attempt: number
  input?: object
  output?: object
  error?: string
  duration_ms: number
}

interface Schedule {
  id: string
  workflow_id: string
  name: string
  cron: string
  timezone: string
  input: object
  enabled: boolean
}

interface Webhook {
  id: string
  workflow_id: string
  name: string
  url: string
  secret: string
  input_mapping: object
}
```

## Composables

### useAuth (composables/useAuth.ts)

```typescript
const {
  isAuthenticated,  // Ref<boolean>
  user,             // Ref<KeycloakUser | null>
  token,            // Ref<string | null>
  login,            // () => Promise<void>
  logout,           // () => Promise<void>
  tenantId          // Ref<string | null>
} = useAuth()

interface KeycloakUser {
  id: string
  email: string
  name: string
  roles: string[]
}
```

### useApi (composables/useApi.ts)

```typescript
const {
  get,    // <T>(path: string) => Promise<T>
  post,   // <T>(path: string, body: object) => Promise<T>
  put,    // <T>(path: string, body: object) => Promise<T>
  del     // (path: string) => Promise<void>
} = useApi()

// useAuth から Authorization ヘッダーを自動注入
// X-Tenant-ID ヘッダーを自動注入
```

### useProjects (composables/useProjects.ts)

> 注: バックエンドAPIは `/workflows` エンドポイントを使用。このcomposableはそれにマッピング。

```typescript
const {
  list,           // (params?) => Promise<PaginatedResponse<Project>>
  get,            // (id: string) => Promise<ApiResponse<Project>>
  create,         // (data) => Promise<ApiResponse<Project>>
  update,         // (id: string, data) => Promise<ApiResponse<Project>>
  remove,         // (id: string) => Promise<void>
  save,           // (id: string, data) => Promise<ApiResponse<Project>> - 新バージョン作成
  saveDraft,      // (id: string, data) => Promise<ApiResponse<Project>> - ドラフト保存
  publish,        // (id: string) => Promise<ApiResponse<Project>>

  // Steps
  listSteps,      // (projectId: string) => Promise<Step[]>
  createStep,     // (projectId: string, data) => Promise<Step>
  updateStep,     // (projectId: string, stepId: string, data) => Promise<Step>
  deleteStep,     // (projectId: string, stepId: string) => Promise<void>

  // Edges
  listEdges,      // (projectId: string) => Promise<Edge[]>
  createEdge,     // (projectId: string, data) => Promise<Edge>
  deleteEdge,     // (projectId: string, edgeId: string) => Promise<void>

  // Execution
  execute,        // (projectId: string, input, mode) => Promise<Run>
} = useProjects()
```

### useRuns (composables/useRuns.ts)

```typescript
const {
  list,           // (workflowId: string, params?) => Promise<PaginatedResponse<Run>>
  get,            // (runId: string) => Promise<ApiResponse<Run>>
  cancel,         // (runId: string) => Promise<void>
  resume,         // (runId: string) => Promise<ApiResponse<Run>>
} = useRuns()
```

### useBlockSearch (composables/useBlockSearch.ts)

ブロック検索ロジックの共通composable。StepPaletteで使用。

```typescript
import { useBlockSearchWithCategory } from '~/composables/useBlockSearch'

const {
  searchQuery,        // Ref<string> - 検索クエリ
  isSearchActive,     // ComputedRef<boolean> - 検索がアクティブか
  clearSearch,        // () => void - 検索をクリア
  activeCategory,     // Ref<BlockCategory> - アクティブカテゴリ
  blocksBySubcategory,// ComputedRef<Record<string, BlockDefinition[]>>
  activeSubcategories,// ComputedRef<BlockSubcategory[]>
} = useBlockSearchWithCategory(blocks)
```

### useStoredInput (composables/useStoredInput.ts)

localStorageを使った入力値の永続化。ExecutionTabで使用。

```typescript
import { useStoredInput } from '~/composables/useStoredInput'

const { save, load, clear, getKey } = useStoredInput({
  keyPrefix: 'aio:input:workflow-123'
})

// 使用例
save('step-1', { message: 'Hello' })  // 保存
const data = load('step-1')            // 読み込み
clear('step-1')                        // クリア
```

### usePolling (composables/usePolling.ts)

定期的にデータをフェッチするためのcomposable。

```typescript
import { usePolling } from '~/composables/usePolling'

const { isPolling, pollingId, start, stop } = usePolling<Run>({
  interval: 1000,     // ポーリング間隔（ms）
  maxAttempts: 60,    // 最大試行回数
  onTimeout: () => toast.warning('Timeout'),
})

// ポーリング開始
start('run-123', async () => {
  const response = await api.get(runId)
  return response.data
}, (data) => {
  if (data.status === 'completed') return true  // 停止
  return false  // 継続
})
```

### useTemplateVariables (composables/useTemplateVariables.ts)

テンプレート変数の検出・解決。ExecutionTabで使用。

```typescript
import { useTemplateVariables } from '~/composables/useTemplateVariables'

const configRef = computed(() => props.step?.config)
const {
  variables,        // ComputedRef<string[]> - 検出された変数
  formatVariable,   // (name: string) => string - {{name}} 形式にフォーマット
  resolveVariable,  // (path: string, context: object) => string - 値を解決
  createPreview,    // (context: object) => TemplatePreviewItem[] - プレビュー生成
} = useTemplateVariables(configRef)
```

## コンポーネント

### 動的設定フォーム (components/workflow-editor/config/)

スキーマ駆動のフォーム生成。JSON Schema から自動的にフォームを生成。

```
components/workflow-editor/config/
├── DynamicConfigForm.vue       # メインコンポーネント
├── ConfigFieldRenderer.vue     # フィールドレンダラー
├── widgets/
│   ├── TextWidget.vue          # テキスト入力
│   ├── TextareaWidget.vue      # 複数行テキスト
│   ├── NumberWidget.vue        # 数値入力
│   ├── SelectWidget.vue        # セレクトボックス
│   ├── CheckboxWidget.vue      # チェックボックス
│   ├── ArrayWidget.vue         # 配列エディタ
│   └── KeyValueWidget.vue      # キー・バリュー入力
├── composables/
│   ├── useSchemaParser.ts      # スキーマ解析
│   └── useValidation.ts        # ajv バリデーション
└── types/
    └── config-schema.ts        # 型定義
```

**型推論ルール**

| JSON Schema | 推論されるウィジェット |
|-------------|----------------------|
| `type: "string"` | TextWidget |
| `type: "string"` + `enum` | SelectWidget |
| `type: "string"` + `maxLength > 100` | TextareaWidget |
| `type: "string"` + `format: "uri"` | URL 入力 |
| `type: "number"` / `type: "integer"` | NumberWidget |
| `type: "boolean"` | CheckboxWidget |
| `type: "array"` | ArrayWidget |
| `type: "object"` + `additionalProperties` | KeyValueWidget |

**使用例**

```vue
<DynamicConfigForm
  :schema="configSchema"
  :value="nodeConfig"
  @update:value="handleConfigUpdate"
  @validation-error="handleValidationError"
/>
```

**設計詳細**: [designs/BLOCK_CONFIG_IMPROVEMENT.md](./designs/BLOCK_CONFIG_IMPROVEMENT.md)

### DAG エディタ (components/dag-editor/)

[@vue-flow/core](https://vueflow.dev/) を使用

```vue
<template>
  <VueFlow
    :nodes="nodes"
    :edges="edges"
    @node-click="onNodeClick"
    @edge-click="onEdgeClick"
    @connect="onConnect"
  >
    <template #node-step="{ data }">
      <StepNode :data="data" />
    </template>
  </VueFlow>
</template>
```

Props:
| Prop | 型 | 説明 |
|------|------|-------------|
| `workflowId` | `string` | 編集するワークフロー ID |
| `readonly` | `boolean` | 編集を無効化 |

Events:
| イベント | ペイロード | 説明 |
|-------|---------|-------------|
| `step-added` | `Step` | 新規ステップ作成 |
| `step-updated` | `Step` | ステップ設定変更 |
| `step-deleted` | `string` | ステップ ID 削除 |
| `edge-added` | `Edge` | 接続作成 |
| `edge-deleted` | `string` | エッジ ID 削除 |

#### ブロックグループ押出ロジック

ブロック/ブロックグループが移動したときに境界線と被った場合の押出ロジック。

**押出方向の統一**

カスケード押出では、最初の押出方向がすべての後続押出に適用されます。

```
方向タイプ: 'left' | 'right' | 'up' | 'down'

例: 最初の押出が 'right' の場合
  Group A → Block B → Group C → Block D
    ↓         ↓         ↓         ↓
  right     right     right     right
```

**押出ロジックのフロー**

```
┌─────────────────────────────────────────────────────────────────┐
│ 1. 移動主体（グループ/ブロック）がドラッグ終了                     │
├─────────────────────────────────────────────────────────────────┤
│ 2. 衝突検出                                                      │
│    - findDropZone(): グループの境界/内部判定                      │
│    - findGroupCollision(): グループ同士の衝突                     │
│    - findGroupBoundaryCollision(): ブロックとグループ境界の衝突   │
├─────────────────────────────────────────────────────────────────┤
│ 3. 押出処理 (cascadeDirection を使用)                            │
│    - calculatePushPosition(): 統一方向で新位置を計算              │
│    - 最初の押出で cascadeDirection を設定                         │
│    - 以降の押出はすべて同じ方向を使用                             │
├─────────────────────────────────────────────────────────────────┤
│ 4. カスケード処理 (最大10回)                                      │
│    while (groupsToPush.length > 0) {                             │
│      - キューからグループを取得                                   │
│      - calculatePushPosition(cascadeDirection) で押出            │
│      - 新位置での衝突をチェック → キューに追加                    │
│    }                                                              │
└─────────────────────────────────────────────────────────────────┘
```

**挙動マトリックス**

| 移動主体 | 衝突対象 | 挙動 |
|---------|---------|------|
| グループ | グループ境界 | 移動グループがスナップ → `wasGroupPushed=true` |
| グループ | ブロック | ブロックを押出（グループが押されていた場合は常に外側） |
| グループ | グループ | 相手グループを押出 → カスケード |
| ブロック | グループ境界 | ブロックをスナップ（内側/外側） |
| 押出ブロック | グループ境界 | 相手グループを押出 → カスケード |
| 押出グループ | ブロック | ブロックを押出 → カスケード |
| 押出グループ | グループ | 相手グループを押出 → カスケード |

**主要関数**

| 関数 | 目的 |
|-----|------|
| `calculatePushPosition()` | 統一方向で押出位置を計算 |
| `determinePushDirection()` | 相対位置から押出方向を決定 |
| `findGroupCollision()` | グループ同士の衝突検出 |
| `findGroupBoundaryCollision()` | ブロックとグループ境界の衝突検出 |
| `snapToValidPosition()` | 境界上のブロックを有効な位置にスナップ |

#### グループリサイズロジック

グループブロックのリサイズ時の挙動と衝突判定。

**重要な前提条件**

```
1. Vue Flow では親ノード（グループ）の子ノードは相対座標で管理される
2. グループリサイズ時、Vue Flow は子ノードを自動的に移動させる
3. ブロックの「絶対位置」を維持するには、リアルタイムで位置補正が必要
4. グループのネストは非対応（グループ内にグループをドロップすると外側にスナップ）
```

**リサイズイベントフロー**

```
┌─────────────────────────────────────────────────────────────────┐
│ 1. onGroupResizeStart                                           │
│    - グループの初期位置を記録                                    │
│    - 子ノードの初期相対座標を記録                                │
├─────────────────────────────────────────────────────────────────┤
│ 2. onGroupResize (ドラッグ中に連続発火)                          │
│    - グループ位置のデルタ(差分)を計算                            │
│    - 子ノードの相対座標を逆方向に補正                            │
│    → 結果：子ノードは絶対座標上で静止して見える                  │
├─────────────────────────────────────────────────────────────────┤
│ 3. onGroupResizeEnd                                              │
│    - 衝突判定を実行                                              │
│    - 3つのケースに分類して処理                                   │
│    - group:update と group:resize-complete イベントを発火        │
└─────────────────────────────────────────────────────────────────┘
```

**衝突判定の3ケース分類**

リサイズ完了時、各ブロックを以下の3ケースに分類：

| ケース | 条件 | 処理 |
|--------|------|------|
| `fullyInside` | 内部有効エリアに完全に収まる | 変更なし（グループ内に維持） |
| `fullyOutside` | グループと全く重ならない | グループから除外、位置は維持 |
| `onBoundary` | 境界線と重なる（部分的に内外） | グループから除外、外側に押し出し |

```typescript
// 判定ロジック
const fullyInside = stepLeft >= innerLeft && stepRight <= innerRight &&
  stepTop >= innerTop && stepBottom <= innerBottom

const fullyOutside = stepRight <= newX || stepLeft >= newX + newWidth ||
  stepBottom <= newY || stepTop >= newY + newHeight

const onBoundary = !fullyInside && !fullyOutside
```

**位置補正の仕組み**

```
リサイズ前:
  グループ位置: (100, 100)
  ブロック相対座標: (50, 50) → 絶対座標 (150, 150)

左上からリサイズでグループが (80, 80) に移動:
  デルタ: (-20, -20)
  ブロック相対座標を補正: (50 - (-20), 50 - (-20)) = (70, 70)
  新しい絶対座標: (80 + 70, 80 + 70) = (150, 150) ← 変化なし
```

**修正時の注意点**

| 注意点 | 説明 |
|--------|------|
| 位置補正のタイミング | `onGroupResize` でリアルタイム補正必須。`onGroupResizeEnd` のみでは視覚的なジャンプが発生 |
| 衝突判定の順序 | `fullyInside` → `fullyOutside` → `onBoundary` の順で判定 |
| 押出方向 | 境界と重なるブロックは最短距離で外側に押し出す |
| イベント発火順 | `group:update` → `group:resize-complete` の順で発火必須 |
| 相対座標 vs 絶対座標 | 子ノードは parentNode からの相対座標、押出後は絶対座標に変換 |

**関連ファイル**

| ファイル | 役割 |
|----------|------|
| `components/dag-editor/DagEditor.vue` | リサイズハンドラ、衝突判定ロジック |
| `pages/workflows/[id].vue` | `handleGroupResizeComplete` で API 永続化 |

**デバッグ時のチェックポイント**

```
1. リサイズ中にブロックが動いていないか確認
   → onGroupResize の位置補正が正しく動作しているか

2. リサイズ完了後にブロック位置がジャンプしないか確認
   → onGroupResizeEnd で余計な位置変更をしていないか

3. 境界と重ならないブロックの位置が変わっていないか確認
   → fullyOutside のケースが正しく処理されているか

4. 押し出されたブロックがグループから除外されているか確認
   → parentNode が undefined に設定されているか
```

## ページ

| ページ | パス | 機能 |
|------|------|----------|
| ダッシュボード | `pages/index.vue` | 最近のワークフロー、最近の Run（ステータスバッジ）、クイックアクション |
| ワークフロー一覧 | `pages/workflows/index.vue` | ステータスでフィルタ、名前で検索、作成/編集/削除/公開 |
| ワークフロー詳細 | `pages/workflows/[id].vue` | DAG エディタ + 設定パネル + 実行ボタン |
| Run 一覧 | `pages/runs/index.vue` | ワークフロー名、ステータスバッジ、トリガータイプ、所要時間 |
| Run 詳細 | `pages/runs/[id].vue` | Run メタデータ、ステップタイムライン、ステップ I/O、エラー詳細 |
| スケジュール | `pages/schedules/index.vue` | スケジュール実行の CRUD |
| 設定 | `pages/settings/index.vue` | テナント設定 |

### ワークフロー詳細レイアウト

```
+------------------+------------------+
|  DAG エディタ     |  設定パネル      |
|                  |  - ステップ設定   |
|                  |  - 実行ボタン     |
+------------------+------------------+
```

## 認証プラグイン (plugins/auth.client.ts)

```typescript
export default defineNuxtPlugin(async () => {
  const config = useRuntimeConfig()

  const keycloak = new Keycloak({
    url: config.public.keycloak.url,
    realm: config.public.keycloak.realm,
    clientId: config.public.keycloak.clientId
  })

  await keycloak.init({
    onLoad: 'check-sso',
    silentCheckSsoRedirectUri: window.location.origin + '/silent-check-sso.html'
  })

  return {
    provide: { keycloak }
  }
})
```

## ルーティング

| パス | ページ | 認証必須 |
|------|------|---------------|
| `/` | ダッシュボード | はい |
| `/workflows` | ワークフロー一覧 | はい |
| `/workflows/new` | ワークフロー作成 | はい（builder 以上） |
| `/workflows/:id` | ワークフロー詳細 | はい |
| `/runs` | Run 一覧 | はい |
| `/runs/:id` | Run 詳細 | はい |
| `/schedules` | スケジュール一覧 | はい |
| `/settings` | 設定 | はい（admin） |

## スタイリング

- ユーティリティファースト CSS
- テーマ用の CSS 変数
- `<style scoped>` でコンポーネントスコープスタイル

## ビルドコマンド

```bash
# 開発
npm run dev

# ビルド
npm run build

# 本番ビルドのプレビュー
npm run preview

# リント
npm run lint

# 型チェック
npm run typecheck
```

## 環境変数

| 変数 | 説明 |
|----------|-------------|
| `NUXT_PUBLIC_API_BASE` | バックエンド API URL |
| `NUXT_PUBLIC_KEYCLOAK_URL` | Keycloak URL |
| `NUXT_PUBLIC_KEYCLOAK_REALM` | Keycloak レルム |
| `NUXT_PUBLIC_KEYCLOAK_CLIENT_ID` | Keycloak クライアント ID |

## 正規コードパターン（必須）

Claude Code はこのセクションのパターンに従ってコードを書くこと。
既存コードが異なるパターンを使っていても、このパターンを優先する。

### Composable パターン

```typescript
// ✅ 正しいパターン: API呼び出しを行うシンプルなcomposable
export function useProjects() {
  const api = useApi()

  // APIメソッドを直接返す（状態を持たない）
  async function list(params?: { status?: string }) {
    return api.get<PaginatedResponse<Project>>('/workflows')
  }

  async function get(id: string) {
    return api.get<ApiResponse<Project>>(`/workflows/${id}`)
  }

  return { list, get }
}

// ✅ 正しいパターン: 状態を持つcomposable
export function useProjectList() {
  const projects = ref<Project[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const { list } = useProjects()

  async function fetchProjects() {
    loading.value = true
    error.value = null
    try {
      const response = await list()
      projects.value = response.data || []
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Unknown error'
    } finally {
      loading.value = false
    }
  }

  return {
    projects: readonly(projects),
    loading: readonly(loading),
    error: readonly(error),
    fetchProjects,
  }
}

// ❌ 禁止パターン
export function useProjects() {
  const projects = ref<Project[]>([])
  const { get } = useApi()

  async function fetchProjects() {
    // loading 状態なし → NG
    // error ハンドリングなし → NG
    const data = await get('/api/v1/workflows')
    projects.value = data.workflows  // 型チェックなし → NG
  }

  return {
    projects,  // readonly でない → NG（外部から変更可能）
    fetchProjects,
  }
}
```

**Why**:
- `loading` / `error` 状態は UI に必須（ローディング表示、エラー表示）
- `readonly()` で外部からの直接変更を防ぐ
- try-catch で予期せぬエラーをキャッチ

---

### コンポーネントパターン (Vue 3 script setup)

```vue
<!-- ✅ 正しいパターン -->
<script setup lang="ts">
interface Props {
  projectId: string
  readonly?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  readonly: false
})

const emit = defineEmits<{
  'step-updated': [step: Step]
  'error': [message: string]
}>()

const projectsApi = useProjects()

// 初期化は onMounted で
onMounted(async () => {
  await projectsApi.get(props.projectId)
})

// イベントハンドラは明示的な関数で
function handleStepUpdate(step: Step) {
  emit('step-updated', step)
}
</script>

<!-- ❌ 禁止パターン -->
<script setup>
// TypeScript なし → NG
// Props の型定義なし → NG
const props = defineProps(['workflowId'])

// トップレベルで await → NG（SSR で問題）
await fetchWorkflows()

// インライン関数 → NG（再レンダリングで再生成）
</script>
```

**Why**:
- TypeScript で型安全性を確保
- `onMounted` で SSR 対応
- 明示的な関数でパフォーマンス最適化

---

### API 呼び出しパターン

```typescript
// ✅ 正しいパターン
async function createWorkflow(input: CreateWorkflowInput): Promise<Workflow> {
  const { post } = useApi()

  try {
    const result = await post<Workflow>('/api/v1/workflows', input)
    return result
  } catch (e) {
    if (e instanceof ApiError) {
      // API エラーは具体的に処理
      if (e.status === 400) {
        throw new Error(`Validation error: ${e.message}`)
      }
      if (e.status === 409) {
        throw new Error('Workflow already exists')
      }
    }
    throw e
  }
}

// ❌ 禁止パターン
async function createWorkflow(input) {
  // 型なし → NG
  const result = await fetch('/api/v1/workflows', {
    method: 'POST',
    body: JSON.stringify(input),
  })
  // ステータスチェックなし → NG
  // useApi 未使用 → NG（認証ヘッダー欠落）
  return result.json()
}
```

**Why**:
- `useApi` は認証ヘッダー (`Authorization`, `X-Tenant-ID`) を自動付与
- 直接 `fetch` を使うと認証が欠落する

---

### リアクティブパターン

```typescript
// ✅ 正しいパターン
const workflowId = computed(() => props.workflowId)
const selectedStep = ref<Step | null>(null)

// computed で派生状態を管理
const isValid = computed(() => {
  return selectedStep.value !== null &&
         selectedStep.value.name.length > 0
})

// watch は最小限に
watch(workflowId, async (newId) => {
  if (newId) {
    await fetchWorkflow(newId)
  }
}, { immediate: true })

// ❌ 禁止パターン
let workflowId = props.workflowId  // ref/computed なし → NG（リアクティブでない）

const isValid = ref(false)  // 派生状態を ref で管理 → NG
watch(selectedStep, () => {
  isValid.value = selectedStep.value !== null  // 手動同期 → NG
})
```

**Why**:
- `computed` は自動的に依存関係を追跡
- 手動同期は同期漏れのリスクがある

---

### SSR 対応パターン

```typescript
// ✅ 正しいパターン
// ブラウザ専用 API は onMounted で
onMounted(() => {
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
})

// または ClientOnly でラップ
```

```vue
<template>
  <ClientOnly>
    <DagEditor :workflow-id="id" />
  </ClientOnly>
</template>
```

```typescript
// ❌ 禁止パターン
// トップレベルでブラウザ API → NG
const width = window.innerWidth  // SSR でエラー

// alert/confirm/prompt → NG
if (confirm('Delete?')) { ... }
```

**禁止: ブラウザダイアログ**

```typescript
// ❌ 禁止: alert, confirm, prompt
alert('Error occurred')
confirm('Are you sure?')
prompt('Enter name')

// ✅ 代わりに使う: Toast / Modal
const toast = useToast()
toast.add({ title: 'Error', description: 'Operation failed', color: 'red' })

// ✅ 確認ダイアログ
const { open } = useConfirmDialog()
const confirmed = await open({
  title: 'Delete Workflow',
  message: 'This action cannot be undone.',
})
```

---

### イベントハンドリングパターン

```typescript
// ✅ 正しいパターン
function handleNodeDragStop(event: NodeDragEvent) {
  const { node } = event
  const position = { x: node.position.x, y: node.position.y }

  // 即座に API 更新
  updateStepPosition(node.id, position)
}

// debounce が必要な場合
const debouncedSave = useDebounceFn(async (value: string) => {
  await saveConfig(value)
}, 300)

// ❌ 禁止パターン
function handleNodeDragStop(event) {
  // 型なし → NG
  // setTimeout で遅延 → NG（useDebounceFn を使う）
  setTimeout(() => {
    updateStepPosition(event.node.id, event.node.position)
  }, 300)
}
```

---

### DAG Editor 特別ルール

```typescript
// ✅ 座標計算は毎回再計算
function getCurrentBounds(node: Node): Bounds {
  return calculateBounds(node)  // キャッシュしない
}

// ❌ 座標をキャッシュ → NG
const cachedBounds = computed(() => calculateBounds(node))
// グループリサイズ時に古い座標が使われる

// ✅ グループ内ノードの位置は相対座標で管理
function getAbsolutePosition(node: Node): Position {
  if (node.parentNode) {
    const parent = findNode(node.parentNode)
    return {
      x: parent.position.x + node.position.x,
      y: parent.position.y + node.position.y,
    }
  }
  return node.position
}

// ❌ 絶対座標と相対座標を混同 → NG
```

**DAG Editor 修正時のチェックリスト**:
1. [ ] 座標計算をキャッシュしていないか
2. [ ] 親子関係の座標変換を正しく行っているか
3. [ ] リサイズ中のリアルタイム補正があるか
4. [ ] `onMounted` でイベントリスナーを設定しているか
5. [ ] `onUnmounted` でクリーンアップしているか

---

### テストパターン

```typescript
// ✅ 正しいパターン
describe('useProjectList', () => {
  it('fetches projects successfully', async () => {
    // Arrange
    const mockProjects = [{ id: '1', name: 'Test' }]
    vi.mocked(useApi).mockReturnValue({
      get: vi.fn().mockResolvedValue({ data: mockProjects }),
    })

    // Act
    const { projects, fetchProjects, loading, error } = useProjectList()
    await fetchProjects()

    // Assert
    expect(projects.value).toEqual(mockProjects)
    expect(loading.value).toBe(false)
    expect(error.value).toBeNull()
  })

  it('handles API error', async () => {
    // Arrange
    vi.mocked(useApi).mockReturnValue({
      get: vi.fn().mockRejectedValue(new Error('Network error')),
    })

    // Act
    const { error, fetchProjects } = useProjectList()
    await fetchProjects()

    // Assert
    expect(error.value).toBe('Network error')
  })
})
```

**テストカバレッジ必須項目**:
1. 正常系（データ取得成功）
2. API エラー（ネットワークエラー、400、500）
3. ローディング状態の遷移
4. 空データ

---

## 関連ドキュメント

- [API.md](./API.md) - REST API エンドポイントとスキーマ
- [TESTING.md](../frontend/docs/TESTING.md) - フロントエンドテストルール
- [BACKEND.md](./BACKEND.md) - バックエンドアーキテクチャ
- [DEPLOYMENT.md](./DEPLOYMENT.md) - Docker と Kubernetes デプロイ
- [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) - エラー対処法
