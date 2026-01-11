# Frontend Technical Reference

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

## Pages

### Dashboard (pages/index.vue)

Displays:
- Recent workflows
- Recent runs (status badges)
- Quick actions

### Workflow List (pages/workflows/index.vue)

Features:
- Filter by status (draft/published)
- Search by name
- Create new button
- Actions: edit, delete, publish

### Workflow Detail (pages/workflows/[id].vue)

Layout:
```
+------------------+------------------+
|  DAG Editor      |  Config Panel    |
|                  |  - Step config   |
|                  |  - Run button    |
+------------------+------------------+
```

### Run List (pages/runs/index.vue)

Columns:
- Workflow name
- Status (badge)
- Trigger type
- Started at
- Duration

### Run Detail (pages/runs/[id].vue)

Displays:
- Run metadata
- Step timeline (visual)
- Step input/output (collapsible)
- Error details (if failed)

### Schedules (pages/schedules/index.vue)

CRUD for scheduled executions.

### Settings (pages/settings/index.vue)

Tenant configuration.

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
