/**
 * useProjectUrlSync.ts
 * URLとプロジェクト状態の同期
 *
 * 機能:
 * - URLクエリパラメータ (?project=xxx) からプロジェクトIDを取得
 * - プロジェクトIDをURLに設定
 * - lastProjectIdからの復元
 */

export function useProjectUrlSync() {
  const route = useRoute()
  const router = useRouter()

  // Get project ID from URL query
  const projectIdFromUrl = computed(() => {
    const projectParam = route.query.project
    if (typeof projectParam === 'string' && projectParam) {
      return projectParam
    }
    return null
  })

  // Set project ID in URL
  function setProjectInUrl(projectId: string | null) {
    if (projectId) {
      router.replace({
        path: route.path,
        query: { project: projectId },
      })
    } else {
      router.replace({
        path: route.path,
        query: {},
      })
    }
  }

  return {
    projectIdFromUrl,
    setProjectInUrl,
  }
}
