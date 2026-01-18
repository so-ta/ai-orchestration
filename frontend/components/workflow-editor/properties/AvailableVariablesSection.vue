<script setup lang="ts">
import type { AvailableVariable } from '../composables/useAvailableVariables'

const { t } = useI18n()

defineProps<{
  variables: AvailableVariable[]
}>()

function formatTemplateVariable(variable: string): string {
  const openBrace = String.fromCharCode(123, 123)
  const closeBrace = String.fromCharCode(125, 125)
  return openBrace + variable + closeBrace
}
</script>

<template>
  <div class="form-section available-variables-section">
    <h4 class="section-title">
      <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
        <polyline points="7 10 12 15 17 10"/>
        <line x1="12" y1="15" x2="12" y2="3"/>
      </svg>
      {{ t('stepConfig.availableVariables.title') }}
    </h4>
    <p class="section-description">
      {{ t('stepConfig.availableVariables.description') }}
    </p>
    <div class="available-variables-list">
      <div v-for="variable in variables" :key="variable.path" class="available-variable-item">
        <div class="variable-header">
          <code class="variable-path">{{ formatTemplateVariable(variable.path) }}</code>
          <code class="variable-type">{{ variable.type }}</code>
        </div>
        <div class="variable-meta">
          <span class="variable-source">{{ variable.source }}</span>
          <span v-if="variable.title && variable.title !== variable.source" class="variable-title">{{ variable.title }}</span>
        </div>
        <div v-if="variable.description" class="variable-description">
          {{ variable.description }}
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

.section-description {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  margin: 0 0 0.75rem 0;
  line-height: 1.4;
}

.available-variables-section {
  background: linear-gradient(135deg, #f0fdf4 0%, #dcfce7 100%);
  border: 1px solid #86efac;
  border-radius: 8px;
  padding: 0.875rem !important;
  margin-top: 0.5rem;
}

.available-variables-section .section-title {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  color: #166534;
  margin-bottom: 0.5rem;
}

.available-variables-section .section-title svg {
  color: #22c55e;
}

.available-variables-section .section-description {
  color: #15803d;
}

.available-variables-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  background: rgba(255, 255, 255, 0.7);
  border-radius: 6px;
  padding: 0.75rem;
  max-height: 200px;
  overflow-y: auto;
}

.available-variable-item {
  padding: 0.5rem;
  background: white;
  border: 1px solid #bbf7d0;
  border-radius: 4px;
}

.variable-header {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  flex-wrap: wrap;
}

.variable-path {
  font-size: 0.6875rem;
  font-family: 'SF Mono', Monaco, monospace;
  background: #dcfce7;
  color: #166534;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  border: 1px solid #86efac;
  word-break: break-all;
}

.variable-type {
  font-size: 0.5625rem;
  font-family: 'SF Mono', Monaco, monospace;
  color: var(--color-text-secondary);
  background: var(--color-background);
  padding: 0.125rem 0.25rem;
  border-radius: 3px;
}

.variable-meta {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-top: 0.25rem;
  font-size: 0.625rem;
}

.variable-source {
  color: #15803d;
  font-weight: 500;
}

.variable-title {
  color: var(--color-text-secondary);
}

.variable-description {
  font-size: 0.5625rem;
  color: var(--color-text-secondary);
  margin-top: 0.25rem;
  line-height: 1.4;
}
</style>
