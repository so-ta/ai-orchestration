<script setup lang="ts">
import {
  COPILOT_SIDEBAR_MIN_WIDTH,
  COPILOT_SIDEBAR_MAX_WIDTH,
} from '~/composables/useEditorState'
import { useBottomOffset } from '~/composables/useFloatingLayout'
import type { ProposalChange } from './CopilotProposalCard.vue'

defineProps<{
  workflowId: string
}>()

const emit = defineEmits<{
  'changes:applied': [changes: ProposalChange[]]
  'changes:preview': []
}>()

const { t } = useI18n()
const {
  copilotSidebarOpen,
  copilotSidebarWidth,
  copilotSidebarResizing,
  toggleCopilotSidebar,
  setCopilotSidebarWidth,
  setCopilotSidebarResizing,
} = useEditorState()
const { offset: bottomOffset, isResizing: bottomPanelResizing } = useBottomOffset(0)

// Resize state (local)
const startX = ref(0)
const startWidth = ref(0)

// Resize handlers
function startResize(e: MouseEvent) {
  setCopilotSidebarResizing(true)
  startX.value = e.clientX
  startWidth.value = copilotSidebarWidth.value
  document.addEventListener('mousemove', handleResize)
  document.addEventListener('mouseup', stopResize)
}

function handleResize(e: MouseEvent) {
  if (!copilotSidebarResizing.value) return
  // Dragging left increases width, dragging right decreases
  const delta = startX.value - e.clientX
  const newWidth = Math.max(
    COPILOT_SIDEBAR_MIN_WIDTH,
    Math.min(COPILOT_SIDEBAR_MAX_WIDTH, startWidth.value + delta)
  )
  setCopilotSidebarWidth(newWidth)
}

function stopResize() {
  setCopilotSidebarResizing(false)
  document.removeEventListener('mousemove', handleResize)
  document.removeEventListener('mouseup', stopResize)
}

// Forward events from CopilotTab
function handleChangesApplied(changes: ProposalChange[]) {
  emit('changes:applied', changes)
}

function handleChangesPreview() {
  emit('changes:preview')
}
</script>

<template>
  <Transition name="slide-right">
    <div
      v-if="copilotSidebarOpen"
      class="copilot-sidebar"
      :class="{ resizing: copilotSidebarResizing }"
      :style="{
        width: `${copilotSidebarWidth}px`,
        bottom: `${bottomOffset}px`,
        transition: (copilotSidebarResizing || bottomPanelResizing) ? 'none' : undefined,
      }"
    >
      <!-- Resize Handle -->
      <div class="resize-handle" @mousedown="startResize">
        <div class="resize-line" />
      </div>

      <!-- Sidebar Content -->
      <div class="sidebar-content">
        <!-- Header -->
        <div class="sidebar-header">
          <h3 class="sidebar-title">{{ t('copilot.sidebar.title') }}</h3>
          <button
            class="collapse-btn"
            :title="t('copilot.sidebar.collapse')"
            @click="toggleCopilotSidebar"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="9 18 15 12 9 6"/>
            </svg>
          </button>
        </div>

        <!-- Copilot Content -->
        <div class="sidebar-body">
          <CopilotTab
            :workflow-id="workflowId"
            @changes:applied="handleChangesApplied"
            @changes:preview="handleChangesPreview"
          />
        </div>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.copilot-sidebar {
  position: fixed;
  top: 0;
  right: 0;
  /* bottom is set dynamically via :style */
  background: var(--color-surface);
  border-left: 1px solid var(--color-border);
  z-index: 150;
  display: flex;
  flex-direction: column;
  transition: width 0.2s ease, bottom 0.2s ease;
  box-shadow: -2px 0 8px rgba(0, 0, 0, 0.1);
}

.copilot-sidebar.resizing {
  transition: none;
  user-select: none;
}

/* Resize Handle */
.resize-handle {
  position: absolute;
  top: 0;
  left: -4px;
  bottom: 0;
  width: 8px;
  cursor: ew-resize;
  z-index: 10;
  display: flex;
  align-items: center;
  justify-content: center;
}

.resize-handle:hover .resize-line,
.copilot-sidebar.resizing .resize-line {
  background: #3b82f6;
  width: 3px;
}

.resize-line {
  width: 2px;
  height: 40px;
  background: rgba(0, 0, 0, 0.1);
  border-radius: 2px;
  transition: all 0.15s;
}

/* Sidebar Content */
.sidebar-content {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem 1rem;
  border-bottom: 1px solid var(--color-border);
  background: var(--color-background);
  flex-shrink: 0;
}

.sidebar-title {
  margin: 0;
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--color-text);
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.sidebar-title::before {
  content: '';
  display: inline-block;
  width: 8px;
  height: 8px;
  background: linear-gradient(135deg, var(--color-primary) 0%, #8b5cf6 100%);
  border-radius: 50%;
}

.collapse-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  padding: 0;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 6px;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s;
}

.collapse-btn:hover {
  background: var(--color-surface);
  border-color: var(--color-border);
  color: var(--color-text);
}

.sidebar-body {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  padding: 0.75rem;
}

/* Deep styles for CopilotTab inside sidebar */
.sidebar-body :deep(.copilot-tab) {
  height: 100%;
}

.sidebar-body :deep(.chat-section) {
  height: 100%;
}

.sidebar-body :deep(.chat-messages) {
  padding: 0.5rem;
}

/* Slide animation */
.slide-right-enter-active,
.slide-right-leave-active {
  transition: transform 0.25s ease, opacity 0.25s ease;
}

.slide-right-enter-from,
.slide-right-leave-to {
  transform: translateX(100%);
  opacity: 0;
}

.slide-right-enter-to,
.slide-right-leave-from {
  transform: translateX(0);
  opacity: 1;
}
</style>
