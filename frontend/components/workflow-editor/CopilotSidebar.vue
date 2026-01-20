<script setup lang="ts">
import { COPILOT_SIDEBAR_WIDTH, COPILOT_SIDEBAR_COLLAPSED_WIDTH } from '~/composables/useEditorState'
import { useBottomOffset } from '~/composables/useFloatingLayout'

defineProps<{
  workflowId: string
}>()

const emit = defineEmits<{
  'changes:applied': []
  'changes:preview': []
}>()

const { t } = useI18n()
const { copilotSidebarOpen, toggleCopilotSidebar } = useEditorState()
const { offset: bottomOffset, isResizing } = useBottomOffset(0)

// Forward events from CopilotTab
function handleChangesApplied() {
  emit('changes:applied')
}

function handleChangesPreview() {
  emit('changes:preview')
}
</script>

<template>
  <div
    class="copilot-sidebar"
    :class="{ open: copilotSidebarOpen }"
    :style="{
      width: copilotSidebarOpen ? `${COPILOT_SIDEBAR_WIDTH}px` : `${COPILOT_SIDEBAR_COLLAPSED_WIDTH}px`,
      bottom: `${bottomOffset}px`,
      transition: isResizing ? 'none' : undefined,
    }"
  >
    <!-- Toggle Button -->
    <button
      class="toggle-btn"
      :class="{ active: copilotSidebarOpen }"
      :title="copilotSidebarOpen ? t('copilot.sidebar.collapse') : t('copilot.sidebar.expand')"
      @click="toggleCopilotSidebar"
    >
      <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M12 2a2 2 0 0 1 2 2c0 .74-.4 1.39-1 1.73V7h1a7 7 0 0 1 7 7h1a1 1 0 0 1 1 1v3a1 1 0 0 1-1 1h-1v1a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-1H2a1 1 0 0 1-1-1v-3a1 1 0 0 1 1-1h1a7 7 0 0 1 7-7h1V5.73A2 2 0 0 1 10 4a2 2 0 0 1 2-2z"/>
        <circle cx="8" cy="14" r="2"/>
        <circle cx="16" cy="14" r="2"/>
      </svg>
      <span v-if="!copilotSidebarOpen" class="toggle-label">AI</span>
    </button>

    <!-- Sidebar Content -->
    <div v-if="copilotSidebarOpen" class="sidebar-content">
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
</template>

<style scoped>
.copilot-sidebar {
  position: fixed;
  top: 0;
  right: 0;
  height: 100%;
  background: var(--color-surface);
  border-left: 1px solid var(--color-border);
  z-index: 150;
  display: flex;
  flex-direction: column;
  transition: width 0.2s ease, bottom 0.2s ease;
  box-shadow: -2px 0 8px rgba(0, 0, 0, 0.1);
}

/* Toggle Button (collapsed state) */
.toggle-btn {
  position: absolute;
  top: 50%;
  left: 0;
  transform: translateX(-100%) translateY(-50%);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.25rem;
  width: 40px;
  padding: 0.75rem 0.5rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-right: none;
  border-radius: 8px 0 0 8px;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s;
  box-shadow: -2px 0 4px rgba(0, 0, 0, 0.1);
}

.toggle-btn:hover {
  color: var(--color-primary);
  background: var(--color-background);
}

.toggle-btn.active {
  color: var(--color-primary);
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.1) 0%, rgba(139, 92, 246, 0.1) 100%);
  border-color: var(--color-primary);
}

.toggle-label {
  font-size: 0.625rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

/* Sidebar Content (expanded state) */
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
</style>
