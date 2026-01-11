<script setup lang="ts">
const props = defineProps<{
  leftWidth: number
  rightWidth: number
}>()

const emit = defineEmits<{
  (e: 'update:leftWidth', width: number): void
  (e: 'update:rightWidth', width: number): void
}>()

// Resize state
const isResizingLeft = ref(false)
const isResizingRight = ref(false)
const startX = ref(0)
const startWidth = ref(0)

// Handle resize start for left divider
function startResizeLeft(event: MouseEvent) {
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

onUnmounted(() => {
  document.removeEventListener('mousemove', handleMouseMove)
  document.removeEventListener('mouseup', stopResize)
})
</script>

<template>
  <div class="editor-layout">
    <!-- Left Sidebar (Step Palette) -->
    <aside
      class="editor-sidebar-left"
      :style="{ width: leftWidth + 'px' }"
    >
      <slot name="palette" />
    </aside>

    <!-- Left Resize Divider -->
    <div
      :class="['resize-divider', { dragging: isResizingLeft }]"
      @mousedown="startResizeLeft"
    />

    <!-- Canvas (DAG Editor) -->
    <main class="editor-canvas">
      <slot name="canvas" />
    </main>

    <!-- Right Resize Divider -->
    <div
      :class="['resize-divider', { dragging: isResizingRight }]"
      @mousedown="startResizeRight"
    />

    <!-- Right Sidebar (Properties Panel) -->
    <aside
      class="editor-sidebar-right"
      :style="{ width: rightWidth + 'px' }"
    >
      <slot name="properties" />
    </aside>
  </div>
</template>

<style scoped>
.editor-layout {
  display: flex;
  height: calc(100vh - 280px);
  min-height: 400px;
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
}

.editor-sidebar-left {
  border-right: 1px solid var(--color-border);
}

.editor-sidebar-right {
  border-left: 1px solid var(--color-border);
}

.editor-canvas {
  flex: 1;
  min-width: 400px;
  overflow: hidden;
}

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

/* Responsive: Hide sidebars on smaller screens */
@media (max-width: 1200px) {
  .editor-sidebar-left {
    width: 240px !important;
  }
  .editor-sidebar-right {
    width: 300px !important;
  }
}

@media (max-width: 1024px) {
  .editor-layout {
    flex-direction: column;
    height: auto;
  }

  .editor-sidebar-left,
  .editor-sidebar-right {
    width: 100% !important;
    height: auto;
    max-height: 300px;
    border-left: none;
    border-right: none;
  }

  .editor-sidebar-left {
    border-bottom: 1px solid var(--color-border);
  }

  .editor-sidebar-right {
    border-top: 1px solid var(--color-border);
  }

  .editor-canvas {
    min-height: 400px;
  }

  .resize-divider {
    display: none;
  }
}
</style>
