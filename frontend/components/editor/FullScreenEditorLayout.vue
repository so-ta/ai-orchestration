<script setup lang="ts">
/**
 * FullScreenEditorLayout.vue
 * 折りたたみ可能な3カラムレイアウト
 *
 * 機能:
 * - 3カラムレイアウト（StepPalette | Canvas | PropertiesPanel）
 * - 左右サイドバーの折りたたみトグルボタン
 * - リサイズハンドル
 * - 折りたたみ時はアイコンのみ表示（48px幅）
 */

const { t } = useI18n()

const props = defineProps<{
  leftWidth: number
  rightWidth: number
  leftCollapsed: boolean
  rightCollapsed: boolean
}>()

const emit = defineEmits<{
  (e: 'update:leftWidth' | 'update:rightWidth', width: number): void
  (e: 'update:leftCollapsed' | 'update:rightCollapsed', collapsed: boolean): void
}>()

// Resize state
const isResizingLeft = ref(false)
const isResizingRight = ref(false)
const startX = ref(0)
const startWidth = ref(0)

// Collapsed width
const COLLAPSED_WIDTH = 48

// Computed actual widths
const actualLeftWidth = computed(() =>
  props.leftCollapsed ? COLLAPSED_WIDTH : props.leftWidth
)
const actualRightWidth = computed(() =>
  props.rightCollapsed ? COLLAPSED_WIDTH : props.rightWidth
)

// Handle resize start for left divider
function startResizeLeft(event: MouseEvent) {
  if (props.leftCollapsed) return
  isResizingLeft.value = true
  startX.value = event.clientX
  startWidth.value = props.leftWidth
  document.addEventListener('mousemove', handleMouseMove)
  document.addEventListener('mouseup', stopResize)
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
}

// Handle resize start for right divider
function startResizeRight(event: MouseEvent) {
  if (props.rightCollapsed) return
  isResizingRight.value = true
  startX.value = event.clientX
  startWidth.value = props.rightWidth
  document.addEventListener('mousemove', handleMouseMove)
  document.addEventListener('mouseup', stopResize)
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
}

function handleMouseMove(event: MouseEvent) {
  if (isResizingLeft.value) {
    const delta = event.clientX - startX.value
    const newWidth = Math.max(200, Math.min(500, startWidth.value + delta))
    emit('update:leftWidth', newWidth)
  } else if (isResizingRight.value) {
    const delta = startX.value - event.clientX
    const newWidth = Math.max(280, Math.min(600, startWidth.value + delta))
    emit('update:rightWidth', newWidth)
  }
}

function stopResize() {
  isResizingLeft.value = false
  isResizingRight.value = false
  document.removeEventListener('mousemove', handleMouseMove)
  document.removeEventListener('mouseup', stopResize)
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
}

function toggleLeftCollapsed() {
  emit('update:leftCollapsed', !props.leftCollapsed)
}

function toggleRightCollapsed() {
  emit('update:rightCollapsed', !props.rightCollapsed)
}

onUnmounted(() => {
  document.removeEventListener('mousemove', handleMouseMove)
  document.removeEventListener('mouseup', stopResize)
})
</script>

<template>
  <div class="fullscreen-editor-layout">
    <!-- Left Sidebar (Step Palette) -->
    <aside
      :class="['editor-sidebar-left', { collapsed: leftCollapsed }]"
      :style="{ width: actualLeftWidth + 'px' }"
    >
      <!-- Collapse toggle button -->
      <button
        class="collapse-toggle collapse-toggle-left"
        :title="leftCollapsed ? t('editor.expandPalette') : t('editor.collapsePalette')"
        @click="toggleLeftCollapsed"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          :class="{ rotated: leftCollapsed }"
        >
          <polyline points="15 18 9 12 15 6" />
        </svg>
      </button>

      <!-- Content -->
      <div v-show="!leftCollapsed" class="sidebar-content">
        <slot name="palette" />
      </div>

      <!-- Collapsed icon -->
      <div v-show="leftCollapsed" class="collapsed-indicator" @click="toggleLeftCollapsed">
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="3" y="3" width="7" height="7" />
          <rect x="14" y="3" width="7" height="7" />
          <rect x="14" y="14" width="7" height="7" />
          <rect x="3" y="14" width="7" height="7" />
        </svg>
      </div>
    </aside>

    <!-- Left Resize Divider -->
    <div
      v-if="!leftCollapsed"
      :class="['resize-divider', { dragging: isResizingLeft }]"
      @mousedown="startResizeLeft"
    />

    <!-- Canvas (DAG Editor) -->
    <main class="editor-canvas">
      <slot name="canvas" />
    </main>

    <!-- Right Resize Divider -->
    <div
      v-if="!rightCollapsed"
      :class="['resize-divider', { dragging: isResizingRight }]"
      @mousedown="startResizeRight"
    />

    <!-- Right Sidebar (Properties Panel) -->
    <aside
      :class="['editor-sidebar-right', { collapsed: rightCollapsed }]"
      :style="{ width: actualRightWidth + 'px' }"
    >
      <!-- Collapse toggle button -->
      <button
        class="collapse-toggle collapse-toggle-right"
        :title="rightCollapsed ? t('editor.expandProperties') : t('editor.collapseProperties')"
        @click="toggleRightCollapsed"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          :class="{ rotated: !rightCollapsed }"
        >
          <polyline points="9 18 15 12 9 6" />
        </svg>
      </button>

      <!-- Content -->
      <div v-show="!rightCollapsed" class="sidebar-content">
        <slot name="properties" />
      </div>

      <!-- Collapsed icon -->
      <div v-show="rightCollapsed" class="collapsed-indicator" @click="toggleRightCollapsed">
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z" />
          <circle cx="12" cy="12" r="3" />
        </svg>
      </div>
    </aside>
  </div>
</template>

<style scoped>
.fullscreen-editor-layout {
  display: flex;
  height: 100%;
  background: var(--color-background);
  overflow: hidden;
}

.editor-sidebar-left,
.editor-sidebar-right {
  flex-shrink: 0;
  background: var(--color-surface);
  overflow: hidden;
  display: flex;
  flex-direction: column;
  position: relative;
  transition: width 0.2s ease;
}

.editor-sidebar-left {
  border-right: 1px solid var(--color-border);
}

.editor-sidebar-right {
  border-left: 1px solid var(--color-border);
}

.editor-sidebar-left.collapsed,
.editor-sidebar-right.collapsed {
  display: flex;
  align-items: center;
  justify-content: center;
}

.sidebar-content {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.editor-canvas {
  flex: 1;
  min-width: 400px;
  overflow: hidden;
}

/* Collapse toggle buttons */
.collapse-toggle {
  position: absolute;
  top: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  padding: 0;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  background: var(--color-surface);
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s;
  z-index: 10;
}

.collapse-toggle:hover {
  background: var(--color-bg-hover);
  color: var(--color-text);
}

.collapse-toggle-left {
  right: 8px;
}

.collapse-toggle-right {
  left: 8px;
}

.collapse-toggle svg.rotated {
  transform: rotate(180deg);
}

/* Collapsed indicator */
.collapsed-indicator {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: color 0.15s;
}

.collapsed-indicator:hover {
  color: var(--color-text);
}

/* Resize divider */
.resize-divider {
  width: 4px;
  cursor: col-resize;
  background: transparent;
  transition: background 0.15s;
  flex-shrink: 0;
  position: relative;
}

.resize-divider::before {
  content: '';
  position: absolute;
  top: 0;
  left: -4px;
  right: -4px;
  bottom: 0;
}

.resize-divider:hover,
.resize-divider.dragging {
  background: var(--color-primary);
}

/* Responsive: Adjust for smaller screens */
@media (max-width: 1200px) {
  .editor-sidebar-left:not(.collapsed) {
    width: 240px !important;
  }
  .editor-sidebar-right:not(.collapsed) {
    width: 300px !important;
  }
}

@media (max-width: 900px) {
  .fullscreen-editor-layout {
    flex-direction: column;
  }

  .editor-sidebar-left,
  .editor-sidebar-right {
    width: 100% !important;
    height: auto;
    max-height: 200px;
    border-left: none;
    border-right: none;
  }

  .editor-sidebar-left {
    border-bottom: 1px solid var(--color-border);
  }

  .editor-sidebar-right {
    border-top: 1px solid var(--color-border);
  }

  .editor-sidebar-left.collapsed,
  .editor-sidebar-right.collapsed {
    max-height: 48px;
  }

  .editor-canvas {
    min-height: 400px;
  }

  .resize-divider {
    display: none;
  }

  .collapse-toggle {
    display: none;
  }
}
</style>
