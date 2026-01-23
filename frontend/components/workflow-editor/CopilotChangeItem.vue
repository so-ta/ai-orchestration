<script setup lang="ts">
/**
 * CopilotChangeItem.vue
 * Individual change item display for proposal cards
 *
 * Shows change-type specific details:
 * - step:create → Block icon + name + config summary
 * - step:update → Before→After diff display
 * - step:delete → Delete marker with warning color
 * - edge:create/delete → Connection visualization
 */
import type { ProposalChange } from './CopilotProposalCard.vue'

const { t } = useI18n()

// Step lookup info type
export interface StepLookupInfo {
  name: string
  type: string
}

const props = defineProps<{
  change: ProposalChange
  stepLookup?: Map<string, StepLookupInfo>
}>()

// Get display class for change type
const changeClass = computed(() => {
  switch (props.change.type) {
    case 'step:create':
    case 'edge:create':
      return 'added'
    case 'step:update':
      return 'modified'
    case 'step:delete':
    case 'edge:delete':
      return 'deleted'
    default:
      return ''
  }
})

// Get icon type
const iconType = computed(() => {
  switch (props.change.type) {
    case 'step:create':
      return 'plus'
    case 'step:update':
      return 'edit'
    case 'step:delete':
      return 'trash'
    case 'edge:create':
      return 'arrow-right'
    case 'edge:delete':
      return 'x'
    default:
      return 'help'
  }
})

// Get step name from lookup
function getStepName(stepId: string): string {
  const stepInfo = props.stepLookup?.get(stepId)
  return stepInfo?.name || truncateId(stepId)
}

// Get primary label
const label = computed(() => {
  switch (props.change.type) {
    case 'step:create':
      return props.change.name || t('copilot.change.newStep')
    case 'step:update':
      return getStepName(props.change.step_id || '')
    case 'step:delete':
      return getStepName(props.change.step_id || '')
    case 'edge:create':
      return t('copilot.change.connection')
    case 'edge:delete':
      return t('copilot.change.connection')
    default:
      return 'Unknown'
  }
})

// Get detail/secondary info
const detail = computed(() => {
  switch (props.change.type) {
    case 'step:create':
      return getStepTypeLabel(props.change.step_type || '')
    case 'step:update':
      return formatPatch(props.change.patch)
    case 'step:delete':
      return t('copilot.change.willBeDeleted')
    case 'edge:create':
      return `${truncateId(props.change.source_id || '')} → ${truncateId(props.change.target_id || '')}`
    case 'edge:delete':
      return truncateId(props.change.edge_id || '')
    default:
      return ''
  }
})

// Helper: truncate ID for display
function truncateId(id: string): string {
  if (!id) return '???'
  return id.length > 8 ? id.slice(0, 8) : id
}

// Helper: get step type label with human-readable formatting
function getStepTypeLabel(stepType: string): string {
  if (!stepType) return ''
  // Try to get translated label first
  const key = `blockTypes.${stepType}`
  const translated = t(key)
  if (translated !== key) return translated
  // Fallback: format slug to human-readable text
  // discord_send_message → Discord Send Message
  return formatBlockSlug(stepType)
}

// Helper: format block slug to human-readable text
function formatBlockSlug(slug: string): string {
  if (!slug) return ''
  return slug
    .split('_')
    .map(word => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ')
}

// Helper: format patch object for display
function formatPatch(patch?: Record<string, unknown>): string {
  if (!patch) return ''
  const keys = Object.keys(patch)
  if (keys.length === 0) return ''
  if (keys.length <= 2) {
    return keys.join(', ')
  }
  return `${keys.slice(0, 2).join(', ')} +${keys.length - 2}`
}

// Check if this is a step create with config
const hasConfigPreview = computed(() => {
  return props.change.type === 'step:create' && props.change.config
})

// Maximum items to show in preview
const MAX_PREVIEW_ITEMS = 5
const MAX_VALUE_LENGTH = 50

// Get config preview items (limited to 5 items)
const configPreviewItems = computed(() => {
  if (!hasConfigPreview.value || !props.change.config) return []
  const entries = Object.entries(props.change.config)
  return entries.slice(0, MAX_PREVIEW_ITEMS).map(([key, value]) => ({
    key,
    value: formatConfigValue(value, MAX_VALUE_LENGTH),
  }))
})

// Count remaining config items not shown
const remainingConfigCount = computed(() => {
  if (!props.change.config) return 0
  return Math.max(0, Object.keys(props.change.config).length - MAX_PREVIEW_ITEMS)
})

// Helper: format config value for display
function formatConfigValue(value: unknown, maxLength = 50): string {
  if (value === null || value === undefined) return 'null'
  if (typeof value === 'string') {
    return value.length > maxLength ? value.slice(0, maxLength) + '...' : value
  }
  if (typeof value === 'number' || typeof value === 'boolean') {
    return String(value)
  }
  if (Array.isArray(value)) {
    return `[${value.length} items]`
  }
  if (typeof value === 'object') {
    return '{...}'
  }
  return String(value)
}

// Expand/Collapse state
const isExpanded = ref(false)

// Check if this item has expandable details
const hasDetails = computed(() => {
  return props.change.type === 'step:create' && props.change.config && Object.keys(props.change.config).length > 0
})

// Formatted config for expanded view
const formattedConfig = computed(() => {
  if (!props.change.config) return ''
  return JSON.stringify(props.change.config, null, 2)
})

// Toggle expand state
function toggleExpand() {
  if (hasDetails.value) {
    isExpanded.value = !isExpanded.value
  }
}
</script>

<template>
  <div class="change-item" :class="[changeClass, { expanded: isExpanded, clickable: hasDetails }]" @click="toggleExpand">
    <span class="change-icon">
      <!-- Plus icon -->
      <svg v-if="iconType === 'plus'" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <line x1="12" y1="5" x2="12" y2="19" />
        <line x1="5" y1="12" x2="19" y2="12" />
      </svg>
      <!-- Edit icon -->
      <svg v-else-if="iconType === 'edit'" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
        <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
      </svg>
      <!-- Trash icon -->
      <svg v-else-if="iconType === 'trash'" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <polyline points="3 6 5 6 21 6" />
        <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
      </svg>
      <!-- Arrow right icon -->
      <svg v-else-if="iconType === 'arrow-right'" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <line x1="5" y1="12" x2="19" y2="12" />
        <polyline points="12 5 19 12 12 19" />
      </svg>
      <!-- X icon -->
      <svg v-else-if="iconType === 'x'" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <line x1="18" y1="6" x2="6" y2="18" />
        <line x1="6" y1="6" x2="18" y2="18" />
      </svg>
    </span>

    <div class="change-content">
      <div class="change-header">
        <div class="change-main">
          <span class="change-label">{{ label }}</span>
          <span v-if="detail" class="change-detail">{{ detail }}</span>
        </div>
        <!-- Expand toggle button -->
        <button v-if="hasDetails" class="expand-toggle" @click.stop="toggleExpand">
          <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" :class="{ rotated: isExpanded }">
            <polyline points="6 9 12 15 18 9" />
          </svg>
        </button>
      </div>

      <!-- Config preview for step:create (collapsed view) -->
      <div v-if="hasConfigPreview && configPreviewItems.length > 0 && !isExpanded" class="config-preview">
        <div
          v-for="item in configPreviewItems"
          :key="item.key"
          class="config-item"
        >
          <span class="config-key">{{ item.key }}:</span>
          <span class="config-value">{{ item.value }}</span>
        </div>
        <div v-if="remainingConfigCount > 0" class="config-more">
          {{ t('copilot.change.andMore', { count: remainingConfigCount }) }}
        </div>
      </div>

      <!-- Expanded config details -->
      <Transition name="expand">
        <div v-if="isExpanded && hasDetails" class="change-details">
          <pre class="config-json">{{ formattedConfig }}</pre>
        </div>
      </Transition>
    </div>
  </div>
</template>

<style scoped>
.change-item {
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  border-radius: 6px;
  font-size: 0.8125rem;
  transition: background 0.15s ease;
}

.change-item.clickable {
  cursor: pointer;
}

.change-item.clickable:hover {
  filter: brightness(0.97);
}

.change-item.added {
  background: rgba(34, 197, 94, 0.08);
  border-left: 3px solid #22c55e;
}

.change-item.modified {
  background: rgba(59, 130, 246, 0.08);
  border-left: 3px solid #3b82f6;
}

.change-item.deleted {
  background: rgba(239, 68, 68, 0.08);
  border-left: 3px solid #ef4444;
}

/* Icon */
.change-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  flex-shrink: 0;
  margin-top: 1px;
}

.change-item.added .change-icon {
  color: #22c55e;
}

.change-item.modified .change-icon {
  color: #3b82f6;
}

.change-item.deleted .change-icon {
  color: #ef4444;
}

/* Content */
.change-content {
  flex: 1;
  min-width: 0;
}

.change-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.change-main {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
  flex: 1;
  min-width: 0;
}

.change-label {
  font-weight: 500;
  color: var(--color-text);
}

.change-detail {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  margin-left: auto;
}

/* Expand toggle */
.expand-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  padding: 0;
  background: transparent;
  border: none;
  color: var(--color-text-secondary);
  cursor: pointer;
  border-radius: 4px;
  transition: all 0.15s;
  flex-shrink: 0;
}

.expand-toggle:hover {
  background: var(--color-background);
  color: var(--color-text);
}

.expand-toggle svg {
  transition: transform 0.2s ease;
}

.expand-toggle svg.rotated {
  transform: rotate(180deg);
}

/* Config preview */
.config-preview {
  margin-top: 0.375rem;
  padding-top: 0.375rem;
  border-top: 1px dashed var(--color-border);
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.config-item {
  display: flex;
  gap: 0.25rem;
  font-size: 0.6875rem;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.config-key {
  color: var(--color-text-secondary);
  flex-shrink: 0;
}

.config-value {
  color: var(--color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.config-more {
  font-size: 0.6875rem;
  color: var(--color-text-tertiary);
  font-style: italic;
  margin-top: 0.125rem;
}

/* Expanded details */
.change-details {
  margin-top: 0.5rem;
  padding-top: 0.5rem;
  border-top: 1px dashed var(--color-border);
}

.config-json {
  margin: 0;
  padding: 0.5rem;
  background: var(--color-background);
  border-radius: 4px;
  font-size: 0.6875rem;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  color: var(--color-text);
  overflow-x: auto;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 200px;
  overflow-y: auto;
}

/* Expand transition */
.expand-enter-active,
.expand-leave-active {
  transition: all 0.2s ease;
  overflow: hidden;
}

.expand-enter-from,
.expand-leave-to {
  opacity: 0;
  max-height: 0;
}

.expand-enter-to,
.expand-leave-from {
  opacity: 1;
  max-height: 300px;
}
</style>
