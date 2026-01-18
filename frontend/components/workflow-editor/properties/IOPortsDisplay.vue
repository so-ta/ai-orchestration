<script setup lang="ts">
import type { StepType } from '~/types/api'

const { t } = useI18n()

interface Port {
  name: string
  label: string
  description?: string
  schema?: object
  required?: boolean
  is_default?: boolean
}

defineProps<{
  inputPorts?: Port[]
  outputPorts?: Port[]
  stepType: StepType
}>()

function formatSchemaType(schema: object | undefined): string {
  if (!schema) return 'any'
  const s = schema as Record<string, unknown>
  if (s.type === 'array' && s.items) {
    const items = s.items as Record<string, unknown>
    return `${items.type || 'any'}[]`
  }
  return String(s.type || 'any')
}
</script>

<template>
  <div class="form-section">
    <h4 class="section-title">{{ t('stepConfig.ioPorts.title') }}</h4>

    <!-- Input Ports -->
    <div v-if="inputPorts && inputPorts.length > 1" class="io-ports-group">
      <div class="ports-header">
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="15 18 9 12 15 6"/>
        </svg>
        <span>{{ t('stepConfig.ioPorts.inputs') }}</span>
      </div>
      <div class="ports-list">
        <div v-for="port in inputPorts" :key="port.name" class="port-item">
          <span class="port-name">{{ port.label }}</span>
          <code class="port-type">{{ formatSchemaType(port.schema) }}</code>
          <span v-if="port.required" class="port-required">*</span>
          <span v-if="port.description" class="port-desc">{{ port.description }}</span>
        </div>
      </div>
    </div>

    <!-- Output Ports (for blocks not covered by specific config sections) -->
    <div v-if="outputPorts && outputPorts.length > 1 && !['condition', 'switch', 'human_in_loop'].includes(stepType)" class="io-ports-group">
      <div class="ports-header">
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="9 18 15 12 9 6"/>
        </svg>
        <span>{{ t('stepConfig.ioPorts.outputs') }}</span>
      </div>
      <div class="ports-list">
        <div v-for="port in outputPorts" :key="port.name" class="port-item">
          <span :class="['port-name', { 'port-default': port.is_default }]">{{ port.label }}</span>
          <code class="port-type">{{ formatSchemaType(port.schema) }}</code>
          <span v-if="port.is_default" class="port-default-badge">{{ t('stepConfig.ioPorts.default') }}</span>
          <span v-if="port.description" class="port-desc">{{ port.description }}</span>
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

.io-ports-group {
  margin-bottom: 0.75rem;
}

.io-ports-group:last-child {
  margin-bottom: 0;
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

.ports-list {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  padding: 0.5rem;
  background: var(--color-background);
  border-radius: 6px;
}

.port-item {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.375rem;
  padding: 0.25rem 0;
}

.port-name {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--color-text);
}

.port-name.port-default {
  color: var(--color-primary);
}

.port-type {
  font-size: 0.625rem;
  font-family: 'SF Mono', Monaco, monospace;
  color: var(--color-primary);
  background: rgba(59, 130, 246, 0.1);
  padding: 0.125rem 0.375rem;
  border-radius: 3px;
}

.port-required {
  font-size: 0.75rem;
  color: var(--color-error);
  font-weight: 600;
}

.port-desc {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  width: 100%;
}

.port-default-badge {
  font-size: 0.5625rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  padding: 0.125rem 0.375rem;
  background: #e0e7ff;
  color: #4f46e5;
  border-radius: 3px;
}
</style>
