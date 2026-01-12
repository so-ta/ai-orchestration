# Frontend Technical Reference

Vue 3 / Nuxt 3 frontend structure, composables, and components.

## Quick Reference

| Item | Value |
|------|-------|
| Framework | Nuxt 3 + Vue 3 |
| Language | TypeScript |
| State Management | Composables (Vue 3 Composition API) |
| DAG Editor | @vue-flow/core |
| Auth | Keycloak OIDC |
| Pages | `pages/` (file-based routing) |
| Components | `components/` |
| Composables | `composables/` |
| Types | `types/api.ts` |

## Directory Structure

```
frontend/
├── app.vue                 # Root component
├── nuxt.config.ts          # Nuxt configuration
├── pages/                  # File-based routing
│   ├── index.vue           # Dashboard /
│   ├── workflows/
│   │   ├── index.vue       # List /workflows
│   │   ├── new.vue         # Create /workflows/new
│   │   └── [id].vue        # Detail /workflows/:id
│   ├── runs/
│   │   ├── index.vue       # List /runs
│   │   └── [id].vue        # Detail /runs/:id
│   ├── schedules/
│   │   └── index.vue       # List /schedules
│   └── settings/
│       └── index.vue       # Settings /settings
├── components/
│   └── dag-editor/         # DAG visual editor
├── composables/            # Vue 3 composables
│   ├── useAuth.ts          # Keycloak auth
│   ├── useApi.ts           # API client
│   ├── useWorkflows.ts     # Workflow operations
│   └── useRuns.ts          # Run operations
├── plugins/
│   └── auth.client.ts      # Keycloak initialization
├── layouts/
│   └── default.vue         # Base layout
├── types/
│   └── api.ts              # TypeScript interfaces
└── assets/css/
    └── main.css            # Global styles
```

## Configuration (nuxt.config.ts)

```typescript
export default defineNuxtConfig({
  runtimeConfig: {
    public: {
      apiBase: 'http://localhost:8080',
      keycloak: {
        url: 'http://localhost:8180',
        realm: 'ai-orchestration',
        clientId: 'frontend'
      }
    }
  }
})
```

## Type Definitions (types/api.ts)

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

// Auto-injects Authorization header from useAuth
// Auto-injects X-Tenant-ID header
```

### useWorkflows (composables/useWorkflows.ts)

```typescript
const {
  workflows,        // Ref<Workflow[]>
  loading,          // Ref<boolean>
  error,            // Ref<string | null>
  fetchWorkflows,   // () => Promise<void>
  createWorkflow,   // (data: CreateWorkflowInput) => Promise<Workflow>
  updateWorkflow,   // (id: string, data: UpdateWorkflowInput) => Promise<Workflow>
  deleteWorkflow,   // (id: string) => Promise<void>
  publishWorkflow,  // (id: string) => Promise<Workflow>

  // Steps
  fetchSteps,       // (workflowId: string) => Promise<Step[]>
  createStep,       // (workflowId: string, data: CreateStepInput) => Promise<Step>
  updateStep,       // (workflowId: string, stepId: string, data: UpdateStepInput) => Promise<Step>
  deleteStep,       // (workflowId: string, stepId: string) => Promise<void>

  // Edges
  fetchEdges,       // (workflowId: string) => Promise<Edge[]>
  createEdge,       // (workflowId: string, data: CreateEdgeInput) => Promise<Edge>
  deleteEdge        // (workflowId: string, edgeId: string) => Promise<void>
} = useWorkflows()
```

### useRuns (composables/useRuns.ts)

```typescript
const {
  runs,           // Ref<Run[]>
  currentRun,     // Ref<Run | null>
  loading,        // Ref<boolean>
  error,          // Ref<string | null>
  fetchRuns,      // (workflowId?: string) => Promise<void>
  fetchRun,       // (runId: string) => Promise<Run>
  executeWorkflow,// (workflowId: string, input: object, mode: 'test' | 'production') => Promise<Run>
  cancelRun,      // (runId: string) => Promise<void>
  resumeRun       // (runId: string) => Promise<Run>
} = useRuns()
```

## Components

### DAG Editor (components/dag-editor/)

Built with [@vue-flow/core](https://vueflow.dev/)

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
| Prop | Type | Description |
|------|------|-------------|
| `workflowId` | `string` | Workflow ID to edit |
| `readonly` | `boolean` | Disable editing |

Events:
| Event | Payload | Description |
|-------|---------|-------------|
| `step-added` | `Step` | New step created |
| `step-updated` | `Step` | Step config changed |
| `step-deleted` | `string` | Step ID deleted |
| `edge-added` | `Edge` | Connection created |
| `edge-deleted` | `string` | Edge ID deleted |

#### Block Group Push Logic

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

#### Group Resize Logic

グループブロックのリサイズ時の挙動と衝突判定。

**重要な前提条件**

```
1. Vue Flowでは親ノード（グループ）の子ノードは相対座標で管理される
2. グループリサイズ時、Vue Flowは子ノードを自動的に移動させる
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
| 位置補正のタイミング | `onGroupResize`でリアルタイム補正必須。`onGroupResizeEnd`のみでは視覚的なジャンプが発生 |
| 衝突判定の順序 | `fullyInside` → `fullyOutside` → `onBoundary` の順で判定 |
| 押出方向 | 境界と重なるブロックは最短距離で外側に押し出す |
| イベント発火順 | `group:update` → `group:resize-complete` の順で発火必須 |
| 相対座標 vs 絶対座標 | 子ノードはparentNodeからの相対座標、押出後は絶対座標に変換 |

**関連ファイル**

| ファイル | 役割 |
|----------|------|
| `components/dag-editor/DagEditor.vue` | リサイズハンドラ、衝突判定ロジック |
| `pages/workflows/[id].vue` | `handleGroupResizeComplete`でAPI永続化 |

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

## Pages

| Page | Path | Features |
|------|------|----------|
| Dashboard | `pages/index.vue` | Recent workflows, Recent runs (status badges), Quick actions |
| Workflow List | `pages/workflows/index.vue` | Filter by status, Search by name, Create/Edit/Delete/Publish |
| Workflow Detail | `pages/workflows/[id].vue` | DAG Editor + Config Panel + Run button |
| Run List | `pages/runs/index.vue` | Workflow name, Status badge, Trigger type, Duration |
| Run Detail | `pages/runs/[id].vue` | Run metadata, Step timeline, Step I/O, Error details |
| Schedules | `pages/schedules/index.vue` | CRUD for scheduled executions |
| Settings | `pages/settings/index.vue` | Tenant configuration |

### Workflow Detail Layout

```
+------------------+------------------+
|  DAG Editor      |  Config Panel    |
|                  |  - Step config   |
|                  |  - Run button    |
+------------------+------------------+
```

## Auth Plugin (plugins/auth.client.ts)

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

## Routing

| Path | Page | Auth Required |
|------|------|---------------|
| `/` | Dashboard | Yes |
| `/workflows` | Workflow list | Yes |
| `/workflows/new` | Create workflow | Yes (builder+) |
| `/workflows/:id` | Workflow detail | Yes |
| `/runs` | Run list | Yes |
| `/runs/:id` | Run detail | Yes |
| `/schedules` | Schedule list | Yes |
| `/settings` | Settings | Yes (admin) |

## Styling

- Utility-first CSS
- CSS variables for theming
- Component-scoped styles with `<style scoped>`

## Build Commands

```bash
# Development
npm run dev

# Build
npm run build

# Preview production build
npm run preview

# Lint
npm run lint

# Type check
npm run typecheck
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `NUXT_PUBLIC_API_BASE` | Backend API URL |
| `NUXT_PUBLIC_KEYCLOAK_URL` | Keycloak URL |
| `NUXT_PUBLIC_KEYCLOAK_REALM` | Keycloak realm |
| `NUXT_PUBLIC_KEYCLOAK_CLIENT_ID` | Keycloak client ID |

## Related Documents

- [API.md](./API.md) - REST API endpoints and schemas
- [TESTING.md](../frontend/docs/TESTING.md) - Frontend testing rules
- [BACKEND.md](./BACKEND.md) - Backend architecture
- [DEPLOYMENT.md](./DEPLOYMENT.md) - Docker and Kubernetes deployment
