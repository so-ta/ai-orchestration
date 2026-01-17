/**
 * Polling Composable
 *
 * 定期的にデータをフェッチするためのcomposable。
 * ExecutionTabでステップ実行結果のポーリングに使用。
 */

import { ref, onUnmounted } from 'vue'

export interface UsePollingOptions {
  /** ポーリング間隔（ms）デフォルト: 1000ms */
  interval?: number
  /** 最大試行回数（デフォルト: 60） */
  maxAttempts?: number
  /** タイムアウト時のコールバック */
  onTimeout?: () => void
}

export interface UsePollingReturn<T> {
  /** ポーリング中かどうか */
  isPolling: ReturnType<typeof ref<boolean>>
  /** 現在のポーリングID（nullの場合はポーリングしていない） */
  pollingId: ReturnType<typeof ref<string | null>>
  /** ポーリングを開始 */
  start: (id: string, fetcher: () => Promise<T>, onData: (data: T) => boolean | undefined) => void
  /** ポーリングを停止 */
  stop: () => void
}

/**
 * ポーリングのためのcomposable
 *
 * @param options - オプション（間隔、最大試行回数など）
 * @returns ポーリング状態と制御関数
 *
 * @example
 * const { isPolling, pollingId, start, stop } = usePolling({ interval: 1000 })
 *
 * // ポーリング開始
 * start('run-123', async () => {
 *   const response = await api.get(runId)
 *   return response.data
 * }, (data) => {
 *   // trueを返すとポーリングを停止
 *   if (data.status === 'completed') {
 *     toast.success('完了')
 *     return true
 *   }
 *   if (data.status === 'failed') {
 *     toast.error('失敗')
 *     return true
 *   }
 *   return false
 * })
 *
 * // 手動で停止
 * stop()
 */
export function usePolling<T>(options: UsePollingOptions = {}): UsePollingReturn<T> {
  const {
    interval = 1000,
    maxAttempts = 60,
    onTimeout,
  } = options

  const isPolling = ref(false)
  const pollingId = ref<string | null>(null)
  const intervalHandle = ref<ReturnType<typeof setInterval> | null>(null)

  function stop(): void {
    if (intervalHandle.value) {
      clearInterval(intervalHandle.value)
      intervalHandle.value = null
    }
    isPolling.value = false
    pollingId.value = null
  }

  function start(
    id: string,
    fetcher: () => Promise<T>,
    onData: (data: T) => boolean | undefined
  ): void {
    // 既存のポーリングを停止
    stop()

    pollingId.value = id
    isPolling.value = true
    let attempts = 0

    intervalHandle.value = setInterval(async () => {
      attempts++

      if (attempts > maxAttempts) {
        stop()
        onTimeout?.()
        return
      }

      try {
        const data = await fetcher()
        const shouldStop = onData(data)

        if (shouldStop) {
          stop()
        }
      } catch (error) {
        console.error('Polling error:', error)
        // エラー時はポーリングを継続（次回リトライ）
      }
    }, interval)
  }

  // コンポーネントのアンマウント時にクリーンアップ
  onUnmounted(() => {
    stop()
  })

  return {
    isPolling,
    pollingId,
    start,
    stop,
  }
}
