<script setup lang="ts">
/**
 * CopilotProposalCard.vue
 * Inline proposal card displayed within chat messages (Claude Code style)
 *
 * Shows:
 * - Summary of changes (additions, modifications, deletions)
 * - List of individual changes with CopilotChangeItem
 * - Apply/Modify/Discard buttons
 * - Status indicator (applied/discarded)
 */
import CopilotChangeItem, { type StepLookupInfo } from './CopilotChangeItem.vue'

const { t } = useI18n()

// Proposal types matching backend structure
export interface ProposalChangePosition {
  x: number
  y: number
}

export interface ProposalChange {
  type: 'step:create' | 'step:update' | 'step:delete' | 'edge:create' | 'edge:delete'
  temp_id?: string
  step_id?: string
  edge_id?: string
  name?: string
  step_type?: string
  config?: Record<string, unknown>
  position?: ProposalChangePosition
  patch?: Record<string, unknown>
  source_id?: string
  target_id?: string
  source_port?: string
}

export interface Proposal {
  id: string
  status: 'pending' | 'applied' | 'discarded'
  changes: ProposalChange[]
}

const props = defineProps<{
  proposal: Proposal
  messageId?: string
  stepLookup?: Map<string, StepLookupInfo>
}>()

const emit = defineEmits<{
  apply: [proposalId: string]
  discard: [proposalId: string]
  modify: [proposalId: string, feedback: string]
}>()

// Local status (may differ from backend status)
const localStatus = ref<'pending' | 'applied' | 'discarded'>(props.proposal.status)

// Modification UI state
const showModifyInput = ref(false)
const modifyFeedback = ref('')

// Computed: change counts
const additions = computed(() => {
  return props.proposal.changes.filter(
    c => c.type === 'step:create' || c.type === 'edge:create'
  ).length
})

const modifications = computed(() => {
  return props.proposal.changes.filter(c => c.type === 'step:update').length
})

const deletions = computed(() => {
  return props.proposal.changes.filter(
    c => c.type === 'step:delete' || c.type === 'edge:delete'
  ).length
})

// Check if proposal can be actioned
const canAction = computed(() => localStatus.value === 'pending')

// Handle apply
function handleApply() {
  if (!canAction.value) return
  localStatus.value = 'applied'
  emit('apply', props.proposal.id)
}

// Handle discard
function handleDiscard() {
  if (!canAction.value) return
  localStatus.value = 'discarded'
  emit('discard', props.proposal.id)
}

// Handle modify request
function handleModify() {
  if (!modifyFeedback.value.trim()) return
  emit('modify', props.proposal.id, modifyFeedback.value.trim())
  modifyFeedback.value = ''
  showModifyInput.value = false
}

// Cancel modify
function cancelModify() {
  showModifyInput.value = false
  modifyFeedback.value = ''
}

// Watch for external status changes
watch(() => props.proposal.status, (newStatus) => {
  localStatus.value = newStatus
})
</script>

<template>
  <div
    class="proposal-card"
    :class="{
      'status-applied': localStatus === 'applied',
      'status-discarded': localStatus === 'discarded'
    }"
  >
    <!-- Header -->
    <div class="proposal-header">
      <div class="proposal-title">
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
          <polyline points="14 2 14 8 20 8" />
          <line x1="12" y1="18" x2="12" y2="12" />
          <line x1="9" y1="15" x2="15" y2="15" />
        </svg>
        <span>{{ t('copilot.proposal.title') }}</span>
      </div>

      <!-- Status badge -->
      <div v-if="localStatus !== 'pending'" class="status-badge" :class="localStatus">
        <svg v-if="localStatus === 'applied'" xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="20 6 9 17 4 12" />
        </svg>
        <svg v-else xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="18" y1="6" x2="6" y2="18" />
          <line x1="6" y1="6" x2="18" y2="18" />
        </svg>
        <span>{{ t(`copilot.proposal.status.${localStatus}`) }}</span>
      </div>
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
      <CopilotChangeItem
        v-for="(change, idx) in proposal.changes"
        :key="idx"
        :change="change"
        :step-lookup="stepLookup"
      />
    </div>

    <!-- Modify Input (shown when user wants to request modifications) -->
    <div v-if="showModifyInput && canAction" class="modify-section">
      <textarea
        v-model="modifyFeedback"
        class="modify-input"
        :placeholder="t('copilot.proposal.modifyPlaceholder')"
        rows="2"
        @keydown.meta.enter="handleModify"
        @keydown.ctrl.enter="handleModify"
      />
      <div class="modify-actions">
        <button class="btn-ghost" @click="cancelModify">
          {{ t('common.cancel') }}
        </button>
        <button
          class="btn-primary"
          :disabled="!modifyFeedback.trim()"
          @click="handleModify"
        >
          {{ t('copilot.proposal.sendModification') }}
        </button>
      </div>
    </div>

    <!-- Actions -->
    <div v-if="canAction && !showModifyInput" class="actions">
      <button class="btn-ghost" @click="handleDiscard">
        {{ t('copilot.proposal.discard') }}
      </button>
      <button class="btn-secondary" @click="showModifyInput = true">
        {{ t('copilot.proposal.modify') }}
      </button>
      <button class="btn-primary" @click="handleApply">
        {{ t('copilot.proposal.apply') }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.proposal-card {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 0.875rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 10px;
  margin-top: 0.5rem;
  transition: all 0.2s ease;
}

.proposal-card.status-applied {
  border-color: var(--color-success);
  background: rgba(34, 197, 94, 0.05);
}

.proposal-card.status-discarded {
  border-color: var(--color-text-secondary);
  background: var(--color-background);
  opacity: 0.7;
}

/* Header */
.proposal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.proposal-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--color-text);
}

.proposal-title svg {
  color: var(--color-primary);
}

/* Status badge */
.status-badge {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.25rem 0.5rem;
  font-size: 0.6875rem;
  font-weight: 600;
  text-transform: uppercase;
  border-radius: 4px;
}

.status-badge.applied {
  background: rgba(34, 197, 94, 0.15);
  color: #16a34a;
}

.status-badge.discarded {
  background: rgba(107, 114, 128, 0.15);
  color: #6b7280;
}

/* Changes Summary */
.changes-summary {
  display: flex;
  gap: 0.5rem;
}

.badge {
  padding: 0.1875rem 0.5rem;
  font-size: 0.6875rem;
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

/* Modify Section */
.modify-section {
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
  background: var(--color-background);
  color: var(--color-text);
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
  padding: 0.375rem 0.75rem;
  font-size: 0.75rem;
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
