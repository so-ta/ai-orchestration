<script setup lang="ts">
import { onClickOutside } from '@vueuse/core'
import { useBottomOffset, useCopilotOffset } from '~/composables/useFloatingLayout'

const { t } = useI18n()

const props = defineProps<{
  zoom: number
  panelOpen?: boolean
}>()

// ボトムパネルを考慮した下端オフセット（リサイズ中はアニメーション無効）
const { offset: bottomOffset, isResizing } = useBottomOffset(16)

// Copilot Sidebar を考慮した右端オフセット
const copilotOffset = useCopilotOffset(16)

// パネル開閉時の right 計算（Copilot Sidebar + FloatingRightPanel を考慮）
const rightOffset = computed(() => {
  // 基本: copilotOffset (CopilotSidebar開時は 320+16=336, 閉時は 16)
  // panelOpen時: さらに 360px (FloatingRightPanel幅) + 12px (gap) を追加
  if (props.panelOpen) {
    return copilotOffset.value + 360 + 12
  }
  return copilotOffset.value
})

const emit = defineEmits<{
  zoomIn: []
  zoomOut: []
  zoomReset: []
  setZoom: [level: number]
}>()

// Zoom percentage display
const zoomPercent = computed(() => Math.round(props.zoom * 100))

// Zoom presets
const showZoomMenu = ref(false)
const zoomMenuRef = ref<HTMLElement | null>(null)

const zoomPresets = [50, 75, 100, 125, 150, 200]

onClickOutside(zoomMenuRef, () => {
  showZoomMenu.value = false
})

function selectZoomPreset(preset: number) {
  emit('setZoom', preset / 100)
  showZoomMenu.value = false
}
</script>

<template>
  <div class="floating-zoom" :class="{ 'no-transition': isResizing }" :style="{ bottom: bottomOffset + 'px', right: rightOffset + 'px' }">
    <!-- Zoom Controls -->
    <div class="zoom-controls">
      <!-- Zoom Out -->
      <button
        class="zoom-btn"
        :title="t('editor.zoomOut')"
        @click="emit('zoomOut')"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="5" y1="12" x2="19" y2="12" />
        </svg>
      </button>

      <!-- Zoom Percent (clickable for presets) -->
      <div ref="zoomMenuRef" class="zoom-percent-wrapper">
        <button
          class="zoom-percent"
          @click="showZoomMenu = !showZoomMenu"
        >
          {{ zoomPercent }}%
        </button>

        <Transition name="dropdown">
          <div v-if="showZoomMenu" class="zoom-menu">
            <button
              v-for="preset in zoomPresets"
              :key="preset"
              class="zoom-menu-item"
              :class="{ active: zoomPercent === preset }"
              @click="selectZoomPreset(preset)"
            >
              {{ preset }}%
            </button>
          </div>
        </Transition>
      </div>

      <!-- Zoom In -->
      <button
        class="zoom-btn"
        :title="t('editor.zoomIn')"
        @click="emit('zoomIn')"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="12" y1="5" x2="12" y2="19" />
          <line x1="5" y1="12" x2="19" y2="12" />
        </svg>
      </button>
    </div>
  </div>
</template>

<style scoped>
.floating-zoom {
  position: fixed;
  right: 16px;
  z-index: 100;

  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 8px;
  transition: right 0.3s ease, bottom 0.2s ease;
}

.floating-zoom.no-transition {
  transition: none;
}

/* Zoom Controls */
.zoom-controls {
  display: flex;
  align-items: center;
  gap: 2px;

  padding: 6px 8px;
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(12px);
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 10px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

.zoom-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  background: transparent;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  color: #6b7280;
  transition: all 0.15s;
}

.zoom-btn:hover {
  background: rgba(0, 0, 0, 0.05);
  color: #111827;
}

/* Zoom Percent */
.zoom-percent-wrapper {
  position: relative;
}

.zoom-percent {
  min-width: 48px;
  padding: 4px 8px;
  background: transparent;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  font-weight: 500;
  color: #374151;
  text-align: center;
  transition: background 0.15s;
}

.zoom-percent:hover {
  background: rgba(0, 0, 0, 0.05);
}

/* Zoom Menu */
.zoom-menu {
  position: absolute;
  bottom: calc(100% + 8px);
  left: 50%;
  transform: translateX(-50%);
  z-index: 10;

  min-width: 100px;
  padding: 4px;
  background: white;
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 10px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
}

.zoom-menu-item {
  display: block;
  width: 100%;
  padding: 8px 12px;
  background: none;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  color: #374151;
  text-align: center;
  cursor: pointer;
  transition: background 0.1s;
}

.zoom-menu-item:hover {
  background: #f3f4f6;
}

.zoom-menu-item.active {
  background: rgba(59, 130, 246, 0.1);
  color: #3b82f6;
  font-weight: 500;
}

/* Transitions */
.dropdown-enter-active,
.dropdown-leave-active {
  transition: all 0.15s ease;
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(8px);
}
</style>
