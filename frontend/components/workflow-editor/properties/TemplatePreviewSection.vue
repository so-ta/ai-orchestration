<script setup lang="ts">
const { t } = useI18n()

defineProps<{
  variables: string[]
}>()

function formatTemplateVariable(variable: string): string {
  const openBrace = String.fromCharCode(123, 123)
  const closeBrace = String.fromCharCode(125, 125)
  return openBrace + variable + closeBrace
}
</script>

<template>
  <div class="form-section template-preview-section">
    <h4 class="section-title">
      <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/>
        <polyline points="14 2 14 8 20 8"/>
        <path d="M12 18v-6"/>
        <path d="m9 15 3 3 3-3"/>
      </svg>
      {{ t('editor.templatePreview.title') }}
    </h4>
    <p class="template-preview-hint">{{ t('editor.templatePreview.hint') }}</p>
    <div class="template-variables-list">
      <div v-for="variable in variables" :key="variable" class="template-variable-item">
        <code class="variable-name" v-text="formatTemplateVariable(variable)"/>
        <span class="variable-arrow">→</span>
        <span class="variable-placeholder">{{ t('editor.templatePreview.runtimeValue') }}</span>
      </div>
    </div>
    <div class="template-preview-note">
      <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="12" cy="12" r="10"/>
        <line x1="12" y1="16" x2="12" y2="12"/>
        <line x1="12" y1="8" x2="12.01" y2="8"/>
      </svg>
      <span>{{ t('editor.templatePreview.executionNote') }}</span>
    </div>
  </div>
</template>

<style scoped>
.form-section {
  margin-bottom: 1.5rem;
  padding-bottom: 1.5rem;
  border-bottom: 1px solid var(--color-border);
}

.form-section:last-child {
  margin-bottom: 0;
  padding-bottom: 0;
  border-bottom: none;
}

.section-title {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-text);
  margin: 0 0 0.75rem 0;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.template-preview-section {
  background: linear-gradient(135deg, #fefce8 0%, #fef9c3 100%);
  border: 1px solid #fde047;
  border-radius: 8px;
  padding: 0.875rem !important;
  margin-top: 0.5rem;
}

.template-preview-section .section-title {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  color: #854d0e;
  margin-bottom: 0.5rem;
}

.template-preview-section .section-title svg {
  color: #ca8a04;
}

.template-preview-hint {
  font-size: 0.6875rem;
  color: #a16207;
  margin: 0 0 0.75rem 0;
  line-height: 1.4;
}

.template-variables-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  background: rgba(255, 255, 255, 0.7);
  border-radius: 6px;
  padding: 0.75rem;
}

.template-variable-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.75rem;
}

.variable-name {
  font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
  font-size: 0.6875rem;
  background: #fef3c7;
  color: #92400e;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  border: 1px solid #fcd34d;
}

.variable-arrow {
  color: #d97706;
  font-weight: bold;
}

.variable-placeholder {
  font-size: 0.6875rem;
  color: #78716c;
  font-style: italic;
}

.template-preview-note {
  display: flex;
  align-items: flex-start;
  gap: 0.375rem;
  margin-top: 0.75rem;
  padding: 0.5rem 0.625rem;
  background: rgba(255, 255, 255, 0.5);
  border-radius: 4px;
  font-size: 0.625rem;
  color: #78716c;
}

.template-preview-note svg {
  flex-shrink: 0;
  margin-top: 0.125rem;
  color: #a3a3a3;
}
</style>
