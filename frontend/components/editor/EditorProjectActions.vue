<script setup lang="ts">
/**
 * EditorProjectActions.vue
 * プライマリアクションボタン（Save/Draft/Discard/Run）
 */

import type { Project } from '~/types/api'

const { t } = useI18n()

defineProps<{
  project: Project | null
  saving: boolean
}>()

const emit = defineEmits<{
  (e: 'save' | 'saveDraft' | 'discardDraft' | 'run' | 'autoLayout'): void
}>()
</script>

<template>
  <div class="project-actions">
    <!-- Save button -->
    <button
      class="btn btn-primary"
      :disabled="saving || !project"
      @click="emit('save')"
    >
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z" />
        <polyline points="17 21 17 13 7 13 7 21" />
        <polyline points="7 3 7 8 15 8" />
      </svg>
      <span class="btn-text">{{ t('common.save') }}</span>
    </button>

    <!-- Save Draft button -->
    <button
      class="btn btn-outline"
      :disabled="saving || !project"
      @click="emit('saveDraft')"
    >
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
        <polyline points="14 2 14 8 20 8" />
        <line x1="16" y1="13" x2="8" y2="13" />
        <line x1="16" y1="17" x2="8" y2="17" />
      </svg>
      <span class="btn-text">{{ t('workflows.saveDraft') }}</span>
    </button>

    <!-- Discard Draft button (only if has draft) -->
    <button
      v-if="project?.has_draft"
      class="btn btn-outline btn-warning"
      :disabled="saving"
      @click="emit('discardDraft')"
    >
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <polyline points="3 6 5 6 21 6" />
        <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
      </svg>
      <span class="btn-text">{{ t('workflows.discardDraft') }}</span>
    </button>

    <div class="separator" />

    <!-- Auto Layout button -->
    <button
      class="btn btn-outline"
      :disabled="saving || !project"
      :title="t('editor.autoLayout')"
      @click="emit('autoLayout')"
    >
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <rect x="3" y="3" width="7" height="7" />
        <rect x="14" y="3" width="7" height="7" />
        <rect x="14" y="14" width="7" height="7" />
        <rect x="3" y="14" width="7" height="7" />
      </svg>
      <span class="btn-text">{{ t('editor.autoLayout') }}</span>
    </button>

    <div class="separator" />

    <!-- Run button -->
    <button
      class="btn btn-success"
      :disabled="!project"
      @click="emit('run')"
    >
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <polygon points="5 3 19 12 5 21 5 3" />
      </svg>
      <span class="btn-text">{{ t('projects.run') }}</span>
    </button>
  </div>
</template>

<style scoped>
.project-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.btn {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.375rem 0.75rem;
  border: 1px solid transparent;
  border-radius: var(--radius);
  font-size: 0.8125rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
  white-space: nowrap;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-primary {
  background: var(--color-primary);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  opacity: 0.9;
}

.btn-outline {
  background: transparent;
  border-color: var(--color-border);
  color: var(--color-text);
}

.btn-outline:hover:not(:disabled) {
  background: var(--color-surface);
  border-color: var(--color-border-dark, #d1d5db);
}

.btn-warning {
  color: #d97706;
  border-color: #fbbf24;
}

.btn-warning:hover:not(:disabled) {
  background: #fef3c7;
}

.btn-success {
  background: #22c55e;
  color: white;
}

.btn-success:hover:not(:disabled) {
  background: #16a34a;
}

.separator {
  width: 1px;
  height: 20px;
  background: var(--color-border);
  margin: 0 0.25rem;
}

@media (max-width: 640px) {
  .btn-text {
    display: none;
  }

  .btn {
    padding: 0.375rem;
  }

  .separator {
    display: none;
  }
}
</style>
