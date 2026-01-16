<script setup lang="ts">
const { t } = useI18n()

export interface ScriptConfig {
  enabled: boolean
  code: string
}

const props = defineProps<{
  modelValue: ScriptConfig
  title: string
  description: string
  placeholder?: string
  disabled?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: ScriptConfig): void
}>()

const config = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const updateEnabled = (enabled: boolean) => {
  config.value = { ...config.value, enabled }
}

const updateCode = (code: string) => {
  config.value = { ...config.value, code }
}
</script>

<template>
  <div class="script-editor">
    <!-- Header with Toggle -->
    <div class="script-header">
      <div class="script-info">
        <h5 class="script-title">{{ title }}</h5>
        <p class="script-description">{{ description }}</p>
      </div>
      <label class="toggle-switch">
        <input
          type="checkbox"
          :checked="config.enabled"
          :disabled="disabled"
          @change="updateEnabled(($event.target as HTMLInputElement).checked)"
        >
        <span class="toggle-slider" />
      </label>
    </div>

    <!-- Editor Content (shown when enabled) -->
    <div v-if="config.enabled" class="script-content">
      <!-- Code Editor -->
      <div class="form-group">
        <label class="form-label">{{ t('flow.scripts.prescript.code') }}</label>
        <textarea
          class="form-input form-textarea code-editor"
          :value="config.code"
          :disabled="disabled"
          :placeholder="placeholder"
          rows="8"
          spellcheck="false"
          @input="updateCode(($event.target as HTMLTextAreaElement).value)"
        />
      </div>

      <!-- Help Text -->
      <div class="script-help">
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/>
          <line x1="12" y1="16" x2="12" y2="12"/>
          <line x1="12" y1="8" x2="12.01" y2="8"/>
        </svg>
        <div class="help-content">
          <p><code>input</code>, <code>config</code>, <code>ctx.secrets</code> が利用可能</p>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.script-editor {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.script-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
}

.script-info {
  flex: 1;
}

.script-title {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--color-text);
  margin: 0 0 0.25rem 0;
}

.script-description {
  font-size: 0.75rem;
  color: var(--color-text-tertiary);
  margin: 0;
}

/* Toggle Switch */
.toggle-switch {
  position: relative;
  display: inline-block;
  width: 36px;
  height: 20px;
  flex-shrink: 0;
}

.toggle-switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: var(--color-border);
  transition: 0.2s;
  border-radius: 20px;
}

.toggle-slider:before {
  position: absolute;
  content: "";
  height: 14px;
  width: 14px;
  left: 3px;
  bottom: 3px;
  background-color: white;
  transition: 0.2s;
  border-radius: 50%;
}

.toggle-switch input:checked + .toggle-slider {
  background-color: var(--color-primary);
}

.toggle-switch input:checked + .toggle-slider:before {
  transform: translateX(16px);
}

.toggle-switch input:disabled + .toggle-slider {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Script Content */
.script-content {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 0.75rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border-light);
  border-radius: 6px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.form-label {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--color-text-secondary);
}

.form-input {
  padding: 0.5rem 0.625rem;
  font-size: 0.8125rem;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-surface);
  color: var(--color-text);
  transition: border-color 0.2s, box-shadow 0.2s;
}

.form-input:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.form-input:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.form-textarea {
  resize: vertical;
  min-height: 100px;
}

.code-editor {
  font-family: var(--font-mono);
  font-size: 0.75rem;
  line-height: 1.5;
  tab-size: 2;
}

.script-help {
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
  padding: 0.625rem;
  background: rgba(59, 130, 246, 0.05);
  border: 1px solid rgba(59, 130, 246, 0.1);
  border-radius: 6px;
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
}

.script-help svg {
  flex-shrink: 0;
  color: var(--color-primary);
  margin-top: 0.125rem;
}

.help-content {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.help-content p {
  margin: 0;
}

.help-content code {
  font-size: 0.625rem;
  background: var(--color-surface-raised);
  padding: 0.125rem 0.25rem;
  border-radius: 3px;
}
</style>
