import { ref, computed, watch, type Ref } from 'vue'

export interface UseExpressionModeOptions<T> {
  modelValue: Ref<T | string | undefined>
  emit: (value: T | string) => void
  parseValue: (text: string) => T | null  // 式テキストから値へ変換（nullはパースエラー）
  formatValue: (value: T) => string       // 値から表示テキストへ変換
  isValidExpression?: (text: string) => boolean  // 式として有効かどうか
}

export function useExpressionMode<T>(options: UseExpressionModeOptions<T>) {
  const { modelValue, emit, parseValue, formatValue, isValidExpression } = options

  // 式モードかどうか
  const isExpressionMode = ref(false)

  // 式モード時のテキスト
  const expressionText = ref('')

  // 値が {{...}} 形式の変数参照かどうかを判定
  const isExpression = computed(() => {
    const v = modelValue.value
    if (typeof v !== 'string') return false
    return v.startsWith('{{') && v.endsWith('}}')
  })

  // 式として有効かどうか（デフォルトは {{...}} 形式をチェック）
  const checkExpression = (text: string): boolean => {
    if (isValidExpression) {
      return isValidExpression(text)
    }
    const trimmed = text.trim()
    return trimmed.startsWith('{{') && trimmed.endsWith('}}')
  }

  // modelValue の変更を監視して式モードを自動切り替え
  watch(modelValue, (v) => {
    if (typeof v === 'string' && checkExpression(v)) {
      isExpressionMode.value = true
      expressionText.value = v
    }
  }, { immediate: true })

  // 式モードをトグル
  function toggleMode() {
    if (isExpressionMode.value) {
      // 式モード → 通常モード
      // 式テキストから値をパース
      const trimmed = expressionText.value.trim()
      if (checkExpression(trimmed)) {
        // 式は維持（通常モードでは表示できないので変換しない）
        isExpressionMode.value = false
        return
      }
      const parsed = parseValue(trimmed)
      if (parsed !== null) {
        emit(parsed)
      }
      isExpressionMode.value = false
    } else {
      // 通常モード → 式モード
      const v = modelValue.value
      if (typeof v === 'string' && checkExpression(v)) {
        expressionText.value = v
      } else if (v !== undefined) {
        expressionText.value = formatValue(v as T)
      } else {
        expressionText.value = ''
      }
      isExpressionMode.value = true
    }
  }

  // 式テキストを更新
  function updateExpression(text: string) {
    expressionText.value = text
    // 変数式または有効な値ならemit
    if (checkExpression(text.trim())) {
      emit(text.trim())
    } else {
      const parsed = parseValue(text)
      if (parsed !== null) {
        emit(parsed)
      } else {
        // パースエラーでも文字列としてemit（バリデーションは呼び出し側で行う）
        emit(text)
      }
    }
  }

  return {
    isExpressionMode,
    expressionText,
    isExpression,
    toggleMode,
    updateExpression
  }
}
