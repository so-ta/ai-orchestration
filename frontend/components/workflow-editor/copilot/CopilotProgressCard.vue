<script setup lang="ts">
/**
 * CopilotProgressCard.vue
 *
 * Claude Code-style progress checklist displayed within chat messages.
 * Shows workflow preparation status with collapsible items and action buttons.
 */
import type {
  WorkflowProgressStatus,
  ChecklistItem,
  ChecklistItemStatus,
  InlineAction,
} from './types'

const { t } = useI18n()

const props = defineProps<{
  progress: WorkflowProgressStatus
  collapsed?: boolean
}>()

const emit = defineEmits<{
  'item-action': [itemId: string, action: InlineAction]
  'toggle-collapse': []
}>()

// Local state
const isCollapsed = ref(props.collapsed ?? false)

// Sync with prop
watch(() => props.collapsed, (val) => {
  if (val !== undefined) {
    isCollapsed.value = val
  }
})

// Status icon mapping
function getStatusIcon(status: ChecklistItemStatus): string {
  switch (status) {
    case 'completed':
      return '✅'
    case 'in_progress':
      return '▶️'
    case 'skipped':
      return '⏭️'
    case 'error':
      return '❌'
    default:
      return '⬜'
  }
}

// Status class mapping
function getStatusClass(status: ChecklistItemStatus): string {
  switch (status) {
    case 'completed':
      return 'status-completed'
    case 'in_progress':
      return 'status-in-progress'
    case 'skipped':
      return 'status-skipped'
    case 'error':
      return 'status-error'
    default:
      return 'status-pending'
  }
}

// Handle item action click
function handleItemAction(item: ChecklistItem) {
  if (item.action) {
    emit('item-action', item.id, item.action)
  }
}

// Toggle collapse
function toggleCollapse() {
  isCollapsed.value = !isCollapsed.value
  emit('toggle-collapse')
}

// Computed progress percentage
const progressPercent = computed(() => {
  if (props.progress.totalCount === 0) return 0
  return Math.round((props.progress.completedCount / props.progress.totalCount) * 100)
})

// Computed status label
const statusLabel = computed(() => {
  if (props.progress.isComplete) {
    return t('copilot.progress.allComplete')
  }
  return t('copilot.progress.inProgress', {
    completed: props.progress.completedCount,
    total: props.progress.totalCount,
  })
})
</script>

<template>
  <div class="progress-card" :class="{ 'is-collapsed': isCollapsed, 'is-complete': progress.isComplete }">
    <!-- Header -->
    <div class="progress-header" @click="toggleCollapse">
      <div class="header-content">
        <span class="header-icon">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M9 11l3 3L22 4" />
            <path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11" />
          </svg>
        </span>
        <span class="header-title">{{ t('copilot.progress.title') }}</span>
        <span class="header-status">{{ statusLabel }}</span>
      </div>
      <button class="collapse-btn" :aria-label="isCollapsed ? t('common.expand') : t('common.collapse')">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="14"
          height="14"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          class="chevron"
          :class="{ rotated: isCollapsed }"
        >
          <polyline points="6 9 12 15 18 9" />
        </svg>
      </button>
    </div>

    <!-- Progress bar -->
    <div v-if="!isCollapsed" class="progress-bar-container">
      <div class="progress-bar" :style="{ width: `${progressPercent}%` }" />
    </div>

    <!-- Checklist items -->
    <div v-if="!isCollapsed" class="checklist">
      <div
        v-for="item in progress.items"
        :key="item.id"
        class="checklist-item"
        :class="getStatusClass(item.status)"
      >
        <div class="item-main">
          <span class="item-icon">{{ getStatusIcon(item.status) }}</span>
          <span class="item-label">{{ item.label }}</span>
          <span v-if="item.description" class="item-description">{{ item.description }}</span>
          <button
            v-if="item.action && item.status !== 'completed'"
            class="item-action-btn"
            @click.stop="handleItemAction(item)"
          >
            {{ item.action.title || t('copilot.progress.configure') }}
          </button>
        </div>

        <!-- Child items -->
        <div v-if="item.children && item.children.length > 0" class="child-items">
          <div
            v-for="child in item.children"
            :key="child.id"
            class="child-item"
            :class="getStatusClass(child.status)"
          >
            <span class="child-connector">├─</span>
            <span class="item-icon">{{ getStatusIcon(child.status) }}</span>
            <span class="item-label">{{ child.label }}</span>
            <button
              v-if="child.action && child.status !== 'completed'"
              class="item-action-btn small"
              @click.stop="handleItemAction(child)"
            >
              {{ child.action.title || t('copilot.progress.configure') }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Collapsed summary -->
    <div v-if="isCollapsed && progress.isComplete" class="collapsed-summary complete">
      {{ t('copilot.progress.allComplete') }} ✅
    </div>
    <div v-else-if="isCollapsed" class="collapsed-summary">
      {{ statusLabel }}
    </div>
  </div>
</template>

<style scoped>
.progress-card {
  display: flex;
  flex-direction: column;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 10px;
  overflow: hidden;
  transition: all 0.2s ease;
}

.progress-card.is-complete {
  border-color: var(--color-success);
  background: rgba(34, 197, 94, 0.05);
}

/* Header */
.progress-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem 1rem;
  cursor: pointer;
  user-select: none;
  transition: background 0.15s;
}

.progress-header:hover {
  background: var(--color-background);
}

.header-content {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.header-icon {
  display: flex;
  color: var(--color-primary);
}

.header-title {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--color-text);
}

.header-status {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  margin-left: 0.5rem;
}

.collapse-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  background: transparent;
  border: none;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: color 0.15s;
}

.collapse-btn:hover {
  color: var(--color-text);
}

.chevron {
  transition: transform 0.2s ease;
}

.chevron.rotated {
  transform: rotate(-90deg);
}

/* Progress bar */
.progress-bar-container {
  height: 3px;
  background: var(--color-background);
  margin: 0 1rem;
}

.progress-bar {
  height: 100%;
  background: var(--color-primary);
  border-radius: 2px;
  transition: width 0.3s ease;
}

.is-complete .progress-bar {
  background: var(--color-success);
}

/* Checklist */
.checklist {
  display: flex;
  flex-direction: column;
  padding: 0.75rem 1rem;
  gap: 0.5rem;
}

.checklist-item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.item-main {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.8125rem;
}

.item-icon {
  font-size: 0.875rem;
  width: 1.25rem;
  text-align: center;
  flex-shrink: 0;
}

.item-label {
  color: var(--color-text);
}

.item-description {
  color: var(--color-text-secondary);
  font-size: 0.75rem;
}

.status-completed .item-label {
  color: var(--color-text-secondary);
}

.status-in-progress .item-label {
  color: var(--color-primary);
  font-weight: 500;
}

.status-error .item-label {
  color: var(--color-error);
}

.status-skipped .item-label {
  color: var(--color-text-tertiary);
  text-decoration: line-through;
}

/* Action button */
.item-action-btn {
  margin-left: auto;
  padding: 0.25rem 0.5rem;
  font-size: 0.6875rem;
  font-weight: 500;
  color: var(--color-primary);
  background: rgba(99, 102, 241, 0.1);
  border: none;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.15s;
}

.item-action-btn:hover {
  background: rgba(99, 102, 241, 0.2);
}

.item-action-btn.small {
  padding: 0.125rem 0.375rem;
  font-size: 0.625rem;
}

/* Child items */
.child-items {
  display: flex;
  flex-direction: column;
  margin-left: 1.25rem;
  gap: 0.25rem;
}

.child-item {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.75rem;
}

.child-connector {
  color: var(--color-border);
  font-family: ui-monospace, monospace;
  margin-right: 0.25rem;
}

/* Collapsed summary */
.collapsed-summary {
  padding: 0 1rem 0.75rem;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.collapsed-summary.complete {
  color: var(--color-success);
  font-weight: 500;
}
</style>
