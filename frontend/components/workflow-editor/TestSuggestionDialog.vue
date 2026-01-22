<script setup lang="ts">
/**
 * TestSuggestionDialog.vue
 * ワークフロー生成後にテスト実行を提案するダイアログ
 */
const { t } = useI18n()

defineProps<{
  modelValue: boolean
  workflowName?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'run-test': []
  'skip': []
}>()

function close() {
  emit('update:modelValue', false)
}

function runTest() {
  emit('run-test')
  close()
}

function skipTest() {
  emit('skip')
  close()
}
</script>

<template>
  <Teleport to="body">
    <div v-if="modelValue" class="test-suggestion-overlay">
      <div class="test-suggestion-dialog">
        <!-- Header -->
        <div class="test-suggestion-header">
          <div class="test-suggestion-icon">
            <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z" />
            </svg>
          </div>
          <h3 class="test-suggestion-title">{{ t('testSuggestion.title') }}</h3>
        </div>

        <!-- Content -->
        <div class="test-suggestion-content">
          <p class="test-suggestion-message">
            {{ t('testSuggestion.message', { name: workflowName || t('testSuggestion.defaultWorkflowName') }) }}
          </p>
          <p class="test-suggestion-hint">
            {{ t('testSuggestion.hint') }}
          </p>
        </div>

        <!-- Actions -->
        <div class="test-suggestion-actions">
          <button class="test-suggestion-btn test-suggestion-btn-secondary" @click="skipTest">
            {{ t('testSuggestion.skip') }}
          </button>
          <button class="test-suggestion-btn test-suggestion-btn-primary" @click="runTest">
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <polygon points="5 3 19 12 5 21 5 3" />
            </svg>
            {{ t('testSuggestion.runTest') }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.test-suggestion-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.test-suggestion-dialog {
  background: var(--color-surface, #fff);
  border-radius: 16px;
  width: 420px;
  max-width: 90vw;
  padding: 2rem;
  text-align: center;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.15);
}

.test-suggestion-header {
  margin-bottom: 1.5rem;
}

.test-suggestion-icon {
  width: 64px;
  height: 64px;
  background: linear-gradient(135deg, #3b82f6 0%, #8b5cf6 100%);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 1rem;
  color: white;
}

.test-suggestion-title {
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--color-text, #1f2937);
  margin: 0;
}

.test-suggestion-content {
  margin-bottom: 1.5rem;
}

.test-suggestion-message {
  font-size: 1rem;
  color: var(--color-text, #1f2937);
  margin: 0 0 0.75rem;
}

.test-suggestion-hint {
  font-size: 0.875rem;
  color: var(--color-text-secondary, #6b7280);
  margin: 0;
}

.test-suggestion-actions {
  display: flex;
  gap: 0.75rem;
  justify-content: center;
}

.test-suggestion-btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1.25rem;
  border-radius: 8px;
  font-size: 0.9375rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}

.test-suggestion-btn-primary {
  background: var(--color-primary, #3b82f6);
  color: white;
  border: none;
}

.test-suggestion-btn-primary:hover {
  background: var(--color-primary-dark, #2563eb);
}

.test-suggestion-btn-secondary {
  background: transparent;
  color: var(--color-text-secondary, #6b7280);
  border: 1px solid var(--color-border, #e5e7eb);
}

.test-suggestion-btn-secondary:hover {
  background: var(--color-background, #f9fafb);
  color: var(--color-text, #1f2937);
}
</style>
