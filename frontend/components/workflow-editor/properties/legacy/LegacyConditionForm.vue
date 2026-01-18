<script setup lang="ts">
import { useExpressionHelpers, expressionTemplates } from '../../composables/useExpressionHelpers'

const { t } = useI18n()

interface StepConfig {
  expression?: string
  [key: string]: unknown
}

const props = defineProps<{
  modelValue: StepConfig
  disabled?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: StepConfig): void
}>()

const localConfig = ref<StepConfig>({ ...props.modelValue })

watch(() => props.modelValue, (newVal) => {
  localConfig.value = { ...newVal }
}, { deep: true })

const { insertExpression } = useExpressionHelpers(localConfig)

function handleInsertExpression(expr: string) {
  insertExpression(expr)
  emit('update:modelValue', { ...localConfig.value })
}

function updateField(field: keyof StepConfig, value: unknown) {
  emit('update:modelValue', { ...props.modelValue, [field]: value })
}
</script>

<template>
  <div class="form-section">
    <h4 class="section-title">{{ t('stepConfig.condition.title') }}</h4>

    <div class="form-group">
      <label class="form-label">{{ t('stepConfig.condition.expression') }}</label>
      <textarea
        :value="modelValue.expression"
        class="form-input form-textarea code-input"
        rows="2"
        :placeholder="t('stepConfig.condition.expressionPlaceholder')"
        :disabled="disabled"
        @input="updateField('expression', ($event.target as HTMLTextAreaElement).value)"
      />
      <p class="form-hint">{{ t('stepConfig.condition.expressionHint') }}</p>
    </div>

    <!-- Expression helpers -->
    <div class="expression-helpers">
      <span class="helper-label">{{ t('stepConfig.expressionHelpers.title') }}:</span>
      <div class="helper-chips">
        <button type="button" class="helper-chip" :disabled="disabled" @click="handleInsertExpression(expressionTemplates.equals)">
          ==
        </button>
        <button type="button" class="helper-chip" :disabled="disabled" @click="handleInsertExpression(expressionTemplates.notEquals)">
          !=
        </button>
        <button type="button" class="helper-chip" :disabled="disabled" @click="handleInsertExpression(expressionTemplates.greaterThan)">
          &gt;
        </button>
        <button type="button" class="helper-chip" :disabled="disabled" @click="handleInsertExpression(expressionTemplates.lessThan)">
          &lt;
        </button>
        <button type="button" class="helper-chip" :disabled="disabled" @click="handleInsertExpression(expressionTemplates.exists)">
          {{ t('stepConfig.expressionHelpers.exists') }}
        </button>
      </div>
    </div>

    <!-- Output ports display -->
    <div class="output-ports-preview">
      <div class="ports-header">
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/>
          <polyline points="12 6 12 12 16 14"/>
        </svg>
        <span>{{ t('stepConfig.outputPorts') }}</span>
      </div>
      <div class="condition-preview">
        <div class="condition-branch">
          <span class="branch-label branch-true">{{ t('stepConfig.condition.trueBranch') }}</span>
          <span class="branch-type">boolean</span>
          <span class="branch-desc">{{ t('stepConfig.condition.trueBranchDesc') }}</span>
        </div>
        <div class="condition-branch">
          <span class="branch-label branch-false">{{ t('stepConfig.condition.falseBranch') }}</span>
          <span class="branch-type">boolean</span>
          <span class="branch-desc">{{ t('stepConfig.condition.falseBranchDesc') }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.form-section {
  margin-bottom: 1.5rem;
  padding-bottom: 1.5rem;
  border-bottom: 1px solid var(--color-border);
}

.section-title {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-text);
  margin: 0 0 0.75rem 0;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.form-group {
  margin-bottom: 0.875rem;
}

.form-label {
  display: block;
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--color-text);
  margin-bottom: 0.375rem;
}

.form-input {
  width: 100%;
  padding: 0.5rem 0.75rem;
  font-size: 0.8125rem;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: white;
  color: var(--color-text);
  transition: border-color 0.15s;
}

.form-input:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.form-input:disabled {
  background: var(--color-surface);
  cursor: not-allowed;
}

.form-textarea {
  resize: vertical;
  min-height: 80px;
}

.code-input {
  font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
  font-size: 0.75rem;
}

.form-hint {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  margin-top: 0.25rem;
}

.expression-helpers {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  margin-top: 0.5rem;
  padding: 0.5rem;
  background: var(--color-background);
  border-radius: 6px;
}

.helper-label {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
}

.helper-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 0.25rem;
}

.helper-chip {
  padding: 0.25rem 0.5rem;
  font-size: 0.6875rem;
  font-family: 'SF Mono', Monaco, monospace;
  background: white;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.15s;
}

.helper-chip:hover:not(:disabled) {
  background: var(--color-primary);
  color: white;
  border-color: var(--color-primary);
}

.helper-chip:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.output-ports-preview {
  margin-top: 0.75rem;
}

.ports-header {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.6875rem;
  font-weight: 600;
  color: var(--color-text-secondary);
  margin-bottom: 0.5rem;
}

.condition-preview {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  margin-top: 0.75rem;
  padding: 0.75rem;
  background: var(--color-background);
  border-radius: 6px;
}

.condition-branch {
  display: flex;
  align-items: center;
  gap: 0.625rem;
}

.branch-label {
  font-size: 0.6875rem;
  font-weight: 600;
  padding: 0.125rem 0.5rem;
  border-radius: 4px;
}

.branch-true {
  background: #dcfce7;
  color: #16a34a;
}

.branch-false {
  background: #fee2e2;
  color: #dc2626;
}

.branch-type {
  font-size: 0.625rem;
  font-family: 'SF Mono', Monaco, monospace;
  color: var(--color-primary);
  background: rgba(59, 130, 246, 0.1);
  padding: 0.125rem 0.375rem;
  border-radius: 3px;
}

.branch-desc {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
}
</style>
