---
paths:
  - "frontend/**/*.{ts,vue}"
---

# Frontend Rules (Vue 3 / Nuxt 3)

## Composable パターン

```typescript
// API呼び出し（状態なし）
export function useProjects() {
  const api = useApi()
  async function list() {
    return api.get<PaginatedResponse<Project>>('/workflows')
  }
  return { list }
}

// 状態を持つcomposable
export function useProjectList() {
  const projects = ref<Project[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

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
```

## コンポーネント

```vue
<script setup lang="ts">
interface Props {
  projectId: string
}
const props = defineProps<Props>()
const emit = defineEmits<{
  'step-updated': [step: Step]
}>()

onMounted(async () => {
  // 初期化はonMountedで
})
</script>
```

## 禁止事項

| 禁止 | 代替 |
|------|------|
| `alert()`, `confirm()` | Toast / Modal |
| トップレベル `await` | `onMounted` |
| 直接 `fetch` | `useApi()` |
| `ref` を readonly なしで返す | `readonly(ref)` |

## SSR対応

- ブラウザAPI（window等）は `onMounted` 内で使用
- ブラウザ専用コンポーネントは `<ClientOnly>` でラップ

## DAG Editor

- 座標計算をキャッシュしない
- 親子関係の座標変換（相対/絶対）を正しく行う
- リサイズ中はリアルタイム補正

## 参照

詳細は [docs/FRONTEND.md](docs/FRONTEND.md) を参照。
