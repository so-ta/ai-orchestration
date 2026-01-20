<script setup lang="ts">
/**
 * CopilotPreviewPanel.vue
 * Preview panel for Copilot changes before applying
 *
 * Shows:
 * - Summary of changes (additions, modifications, deletions)
 * - List of individual changes
 * - Apply/Discard buttons
 * - Option to request modifications
 */

import type { ReadonlyCopilotDraft, DraftChange } from '~/composables/useCopilotDraft'

const { t } = useI18n()

const props = defineProps<{
  draft: ReadonlyCopilotDraft
}>()

const emit = defineEmits<{
  apply: []
  discard: []
  modify: [feedback: string]
}>()

// Show modification input
const showModifyInput = ref(false)
const modifyFeedback = ref('')

// Computed: change counts
const additions = computed(() => {
  return props.draft.changes.filter(
    c => c.type === 'step:create' || c.type === 'edge:create'
  ).length
})

const modifications = computed(() => {
  return props.draft.changes.filter(c => c.type === 'step:update').length
})

const deletions = computed(() => {
  return props.draft.changes.filter(
    c => c.type === 'step:delete' || c.type === 'edge:delete'
  ).length
})

// Get display info for a change
function getChangeInfo(change: DraftChange) {
  switch (change.type) {
    case 'step:create':
      return {
        icon: 'plus',
        label: change.name,
        detail: t(`blockTypes.${change.stepType}`, change.stepType),
        class: 'added',
      }
    case 'step:update':
      return {
        icon: 'edit',
        label: change.stepId.slice(0, 8),
        detail: Object.keys(change.patch).join(', '),
        class: 'modified',
      }
    case 'step:delete':
      return {
        icon: 'trash',
        label: change.stepId.slice(0, 8),
        detail: t('copilot.preview.stepDeleted'),
        class: 'deleted',
      }
    case 'edge:create':
      return {
        icon: 'arrow-right',
        label: t('copilot.preview.connection'),
        detail: `${change.sourceId.slice(0, 8)} → ${change.targetId.slice(0, 8)}`,
        class: 'added',
      }
    case 'edge:delete':
      return {
        icon: 'x',
        label: t('copilot.preview.connection'),
        detail: change.edgeId.slice(0, 8),
        class: 'deleted',
      }
    default:
      return {
        icon: 'help',
        label: 'Unknown',
        detail: '',
        class: '',
      }
  }
}

// Handle apply
function handleApply() {
  emit('apply')
}

// Handle discard
function handleDiscard() {
  emit('discard')
}

// Handle modify request
function handleModify() {
  if (modifyFeedback.value.trim()) {
    emit('modify', modifyFeedback.value.trim())
    modifyFeedback.value = ''
    showModifyInput.value = false
  }
}
</script>

<template>
  <div class="copilot-preview-panel">
    <div class="preview-header">
      <h3>{{ t('copilot.preview.title') }}</h3>
      <p class="description">{{ draft.description }}</p>
    </div>

    <!-- Changes Summary -->
    <div class="changes-summary">
      <span v-if="additions > 0" class="badge added">
        +{{ additions }}
      </span>
      <span v-if="modifications > 0" class="badge modified">
        ~{{ modifications }}
      </span>
      <span v-if="deletions > 0" class="badge deleted">
        -{{ deletions }}
      </span>
    </div>

    <!-- Changes List -->
    <div class="changes-list">
      <div
        v-for="(change, idx) in draft.changes"
        :key="idx"
        class="change-item"
        :class="getChangeInfo(change).class"
      >
        <span class="change-icon">
          <!-- Plus icon -->
          <svg v-if="getChangeInfo(change).icon === 'plus'" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="12" y1="5" x2="12" y2="19" />
            <line x1="5" y1="12" x2="19" y2="12" />
          </svg>
          <!-- Edit icon -->
          <svg v-else-if="getChangeInfo(change).icon === 'edit'" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
            <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
          </svg>
          <!-- Trash icon -->
          <svg v-else-if="getChangeInfo(change).icon === 'trash'" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="3 6 5 6 21 6" />
            <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
          </svg>
          <!-- Arrow right icon -->
          <svg v-else-if="getChangeInfo(change).icon === 'arrow-right'" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="5" y1="12" x2="19" y2="12" />
            <polyline points="12 5 19 12 12 19" />
          </svg>
          <!-- X icon -->
          <svg v-else-if="getChangeInfo(change).icon === 'x'" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18" />
            <line x1="6" y1="6" x2="18" y2="18" />
          </svg>
        </span>
        <span class="change-label">{{ getChangeInfo(change).label }}</span>
        <span class="change-detail">{{ getChangeInfo(change).detail }}</span>
      </div>
    </div>

    <!-- Modify Input -->
    <div v-if="showModifyInput" class="modify-input-section">
      <textarea
        v-model="modifyFeedback"
        class="modify-input"
        :placeholder="t('copilot.preview.modifyPlaceholder')"
        rows="2"
      />
      <div class="modify-actions">
        <button class="btn-secondary" @click="showModifyInput = false">
          {{ t('common.cancel') }}
        </button>
        <button
          class="btn-primary"
          :disabled="!modifyFeedback.trim()"
          @click="handleModify"
        >
          {{ t('copilot.preview.sendModification') }}
        </button>
      </div>
    </div>

    <!-- Actions -->
    <div class="actions">
      <button class="btn-ghost" @click="handleDiscard">
        {{ t('copilot.preview.discard') }}
      </button>
      <button
        v-if="!showModifyInput"
        class="btn-secondary"
        @click="showModifyInput = true"
      >
        {{ t('copilot.preview.modify') }}
      </button>
      <button class="btn-primary" @click="handleApply">
        {{ t('copilot.preview.apply') }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.copilot-preview-panel {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  padding: 1rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 12px;
}

.preview-header h3 {
  margin: 0;
  font-size: 0.9375rem;
  font-weight: 600;
  color: var(--color-text);
}

.preview-header .description {
  margin: 0.25rem 0 0;
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
}

/* Changes Summary */
.changes-summary {
  display: flex;
  gap: 0.5rem;
}

.badge {
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
  font-weight: 600;
  border-radius: 4px;
}

.badge.added {
  background: rgba(34, 197, 94, 0.15);
  color: #16a34a;
}

.badge.modified {
  background: rgba(59, 130, 246, 0.15);
  color: #2563eb;
}

.badge.deleted {
  background: rgba(239, 68, 68, 0.15);
  color: #dc2626;
}

/* Changes List */
.changes-list {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  max-height: 200px;
  overflow-y: auto;
}

.change-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  border-radius: 6px;
  font-size: 0.8125rem;
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

.change-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
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

.change-label {
  font-weight: 500;
  color: var(--color-text);
}

.change-detail {
  margin-left: auto;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

/* Modify Input */
.modify-input-section {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.modify-input {
  width: 100%;
  padding: 0.5rem 0.75rem;
  font-size: 0.8125rem;
  font-family: inherit;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  resize: none;
}

.modify-input:focus {
  outline: none;
  border-color: var(--color-primary);
}

.modify-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
}

/* Actions */
.actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  padding-top: 0.5rem;
  border-top: 1px solid var(--color-border);
}

.btn-primary,
.btn-secondary,
.btn-ghost {
  padding: 0.5rem 1rem;
  font-size: 0.8125rem;
  font-weight: 500;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-primary {
  background: var(--color-primary);
  color: white;
  border: none;
}

.btn-primary:hover:not(:disabled) {
  opacity: 0.9;
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-secondary {
  background: var(--color-surface);
  color: var(--color-text);
  border: 1px solid var(--color-border);
}

.btn-secondary:hover {
  background: var(--color-background);
}

.btn-ghost {
  background: transparent;
  color: var(--color-text-secondary);
  border: none;
}

.btn-ghost:hover {
  color: var(--color-text);
  background: var(--color-background);
}
</style>
