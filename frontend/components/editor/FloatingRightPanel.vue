<script setup lang="ts">
/**
 * FloatingRightPanel.vue
 * 右側フローティングパネルのラッパー
 *
 * 機能:
 * - 共通の位置、スタイル、アニメーション
 * - ボトムパネルのリサイズに追従
 */
import { useBottomOffset, useCopilotOffset } from '~/composables/useFloatingLayout'

defineProps<{
  /** パネルを表示するか */
  show: boolean
  /** パネルタイトル */
  title?: string
  /** 左にシフトする量（px） */
  shiftLeft?: number
  /** パネルレベル（1=プライマリ, 2=ネスト） */
  level?: 1 | 2
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const { t } = useI18n()

// ボトムパネルを考慮した下端オフセット
const { offset: bottomOffset, isResizing: bottomResizing } = useBottomOffset(12)

// Copilot Sidebar を考慮した右端オフセット
const { value: copilotOffset, isResizing: copilotResizing } = useCopilotOffset(12)

// いずれかのパネルがリサイズ中ならアニメーション無効
const isResizing = computed(() => bottomResizing.value || copilotResizing.value)

// 閉じるボタン
function handleClose() {
  emit('close')
}
</script>

<template>
  <div
    class="floating-right-panel"
    :class="{
      visible: show,
      'no-transition': isResizing,
      'level-2': level === 2
    }"
    :style="{
      bottom: bottomOffset + 'px',
      right: `${copilotOffset + (shiftLeft || 0)}px`
    }"
  >
    <!-- Header -->
    <div v-if="title" class="panel-header">
      <span class="panel-title">{{ title }}</span>
      <button class="close-btn" :title="t('common.close')" @click="handleClose">
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="18" y1="6" x2="6" y2="18" />
          <line x1="6" y1="6" x2="18" y2="18" />
        </svg>
      </button>
    </div>

    <!-- Close button only (no title) -->
    <button v-else class="close-btn absolute" :title="t('common.close')" @click="handleClose">
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <line x1="18" y1="6" x2="6" y2="18" />
        <line x1="6" y1="6" x2="18" y2="18" />
      </svg>
    </button>

    <!-- Content area -->
    <div class="panel-content">
      <slot />
    </div>
  </div>
</template>

<style scoped>
.floating-right-panel {
  position: fixed;
  top: 68px;
  right: 12px;
  width: 360px;
  z-index: 100;

  background: rgba(255, 255, 255, 0.98);
  backdrop-filter: blur(12px);
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 12px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
  overflow: hidden;

  display: flex;
  flex-direction: column;

  transform: translateX(calc(100% + 24px));
  opacity: 0;
  transition:
    transform 0.3s cubic-bezier(0.4, 0, 0.2, 1),
    opacity 0.3s ease,
    right 0.3s cubic-bezier(0.4, 0, 0.2, 1),
    bottom 0.2s ease;
}

.floating-right-panel.visible {
  transform: translateX(0);
  opacity: 1;
}

.floating-right-panel.no-transition {
  transition: transform 0.3s ease, opacity 0.3s ease;
}

.floating-right-panel.level-2 {
  z-index: 101;
  box-shadow: -4px 0 20px rgba(0, 0, 0, 0.12);
}

/* Header */
.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.625rem 0.75rem;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  background: rgba(248, 250, 252, 0.8);
  flex-shrink: 0;
}

.panel-title {
  font-size: 0.8125rem;
  font-weight: 600;
  color: #1e293b;
}

.close-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  padding: 0;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: #94a3b8;
  cursor: pointer;
  transition: all 0.15s;
}

.close-btn:hover {
  background: rgba(0, 0, 0, 0.05);
  color: #64748b;
}

.close-btn.absolute {
  position: absolute;
  top: 8px;
  right: 8px;
  z-index: 10;
}

/* Content */
.panel-content {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
</style>
