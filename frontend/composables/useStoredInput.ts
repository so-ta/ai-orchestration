/**
 * Stored Input Composable
 *
 * localStorageを使った入力値の永続化を提供する。
 * ExecutionTabでステップごとの入力値を保存・復元するために使用。
 */

export interface UseStoredInputOptions {
  /** ストレージキーのプレフィックス */
  keyPrefix: string
}

export interface UseStoredInputReturn {
  /** 入力値を保存 */
  save: (id: string, data: Record<string, unknown>) => void
  /** 入力値を読み込み */
  load: (id: string) => Record<string, unknown> | null
  /** 入力値をクリア */
  clear: (id: string) => void
  /** ストレージキーを生成 */
  getKey: (id: string) => string
}

/**
 * ローカルストレージを使った入力値の永続化
 *
 * @param options - オプション（キープレフィックス）
 * @returns save, load, clear 関数
 *
 * @example
 * const { save, load, clear } = useStoredInput({ keyPrefix: 'aio:input:workflow-123' })
 *
 * // 保存
 * save('step-1', { message: 'Hello' })
 *
 * // 読み込み
 * const data = load('step-1')
 *
 * // クリア
 * clear('step-1')
 */
export function useStoredInput(options: UseStoredInputOptions): UseStoredInputReturn {
  const { keyPrefix } = options

  function getKey(id: string): string {
    return `${keyPrefix}:${id}`
  }

  function save(id: string, data: Record<string, unknown>): void {
    try {
      localStorage.setItem(getKey(id), JSON.stringify(data))
    } catch {
      // Ignore storage errors (quota exceeded, etc.)
    }
  }

  function load(id: string): Record<string, unknown> | null {
    try {
      const stored = localStorage.getItem(getKey(id))
      if (stored) {
        return JSON.parse(stored)
      }
    } catch {
      // Ignore storage errors (parse error, etc.)
    }
    return null
  }

  function clear(id: string): void {
    try {
      localStorage.removeItem(getKey(id))
    } catch {
      // Ignore storage errors
    }
  }

  return {
    save,
    load,
    clear,
    getKey,
  }
}
