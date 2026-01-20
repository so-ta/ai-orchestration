/**
 * useFloatingLayout.ts
 * フローティング要素のレイアウト管理
 *
 * Workflowエディタ内のフローティング要素（ツールバー、ズームコントロールなど）の
 * 位置計算に必要な情報を提供する。
 *
 * useEditorStateのシングルトンパターンを活用して、
 * provide/injectなしでグローバル状態にアクセスする。
 */

import { computed, type ComputedRef, type Ref } from 'vue'
import { COPILOT_SIDEBAR_WIDTH, COPILOT_SIDEBAR_COLLAPSED_WIDTH, useEditorState } from './useEditorState'

/**
 * 下部フローティング要素のbottom位置とアニメーション状態を計算するヘルパー
 * @param baseOffset ベースのオフセット（px）
 * @returns { offset: 計算されたbottom値（px）, isResizing: リサイズ中かどうか }
 */
export function useBottomOffset(baseOffset = 12): {
  offset: ComputedRef<number>
  isResizing: Ref<boolean>
} {
  // useEditorStateはシングルトンなので、直接状態を取得できる
  const { bottomPanelHeight, bottomPanelCollapsed, bottomPanelResizing } = useEditorState()

  const offset = computed(() => {
    // 折りたたみ時は40px（ヘッダーのみ）、展開時は実際の高さ
    const panelHeight = bottomPanelCollapsed.value ? 40 : bottomPanelHeight.value
    return panelHeight + baseOffset
  })

  return {
    offset,
    isResizing: bottomPanelResizing,
  }
}

/**
 * 右フローティング要素のright位置を計算するヘルパー
 * @param baseOffset ベースのオフセット（px）
 * @param panelOpen 右パネルが開いているか（外部から指定）
 * @returns 計算されたright値（px）
 */
export function useRightOffset(baseOffset = 12, panelOpen?: Ref<boolean>): ComputedRef<number> {
  const { rightPanelWidth } = useEditorState()

  return computed(() => {
    if (panelOpen?.value) {
      return rightPanelWidth.value + baseOffset
    }
    return baseOffset
  })
}

/**
 * Copilot Sidebar のオフセットを計算するヘルパー
 * Copilot Sidebar が開いているときは、他のフローティング要素を左にシフトする
 * @param baseOffset ベースのオフセット（px）
 * @returns 計算されたオフセット値（px）
 */
export function useCopilotOffset(baseOffset = 12): ComputedRef<number> {
  const { copilotSidebarOpen } = useEditorState()

  return computed(() => {
    if (copilotSidebarOpen.value) {
      return COPILOT_SIDEBAR_WIDTH + baseOffset
    }
    return baseOffset
  })
}

// Re-export constants for convenience
export { COPILOT_SIDEBAR_WIDTH, COPILOT_SIDEBAR_COLLAPSED_WIDTH }
