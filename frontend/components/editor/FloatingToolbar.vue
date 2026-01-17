<script setup lang="ts">
import { useBottomOffset } from '~/composables/useFloatingLayout'

defineProps<{
  readonly?: boolean
}>()

// ボトムパネルを考慮した下端オフセット（リサイズ中はアニメーション無効）
const { offset: bottomOffset, isResizing } = useBottomOffset(12)
</script>

<template>
  <div
    class="floating-toolbar"
    :class="{ 'no-transition': isResizing }"
    :style="{ bottom: bottomOffset + 'px' }"
  >
    <StepPalette :readonly="readonly" />
  </div>
</template>

<style scoped>
.floating-toolbar {
  position: fixed;
  top: 68px;
  left: 12px;
  z-index: 99;
  transition: bottom 0.2s ease;

  width: 260px;
  display: flex;
  flex-direction: column;

  background: rgba(255, 255, 255, 0.98);
  backdrop-filter: blur(12px);
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 12px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
  overflow: hidden;
}

.floating-toolbar.no-transition {
  transition: none;
}
</style>
