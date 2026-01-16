<script setup lang="ts">
const { t } = useI18n()

export interface ScriptConfig {
  enabled: boolean
  language: 'javascript' | 'python'
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

const updateLanguage = (language: 'javascript' | 'python') => {
  config.value = { ...config.value, language }
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
      <!-- Language Selector -->
      <div class="form-group">
        <label class="form-label">{{ t('flow.scripts.prescript.language') }}</label>
        <div class="language-tabs">
          <button
            :class="['language-tab', { active: config.language === 'javascript' }]"
            :disabled="disabled"
            @click="updateLanguage('javascript')"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
              <path d="M0 0h24v24H0V0zm22.034 18.276c-.175-1.095-.888-2.015-3.003-2.873-.736-.345-1.554-.585-1.797-1.14-.091-.33-.105-.51-.046-.705.15-.646.915-.84 1.515-.66.39.12.75.42.976.9 1.034-.676 1.034-.676 1.755-1.125-.27-.42-.404-.601-.586-.78-.63-.705-1.469-1.065-2.834-1.034l-.705.089c-.676.165-1.32.525-1.71 1.005-1.14 1.291-.811 3.541.569 4.471 1.365 1.02 3.361 1.244 3.616 2.205.24 1.17-.87 1.545-1.966 1.41-.811-.18-1.26-.586-1.755-1.336l-1.83 1.051c.21.48.45.689.81 1.109 1.74 1.756 6.09 1.666 6.871-1.004.029-.09.24-.705.074-1.65l.046.067zm-8.983-7.245h-2.248c0 1.938-.009 3.864-.009 5.805 0 1.232.063 2.363-.138 2.711-.33.689-1.18.601-1.566.48-.396-.196-.597-.466-.83-.855-.063-.105-.11-.196-.127-.196l-1.825 1.125c.305.63.75 1.172 1.324 1.517.855.51 2.004.675 3.207.405.783-.226 1.458-.691 1.811-1.411.51-.93.402-2.07.397-3.346.012-2.054 0-4.109 0-6.179l.004-.056z"/>
            </svg>
            JavaScript
          </button>
          <button
            :class="['language-tab', { active: config.language === 'python' }]"
            :disabled="disabled"
            @click="updateLanguage('python')"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
              <path d="M14.25.18l.9.2.73.26.59.3.45.32.34.34.25.34.16.33.1.3.04.26.02.2-.01.13V8.5l-.05.63-.13.55-.21.46-.26.38-.3.31-.33.25-.35.19-.35.14-.33.1-.3.07-.26.04-.21.02H8.77l-.69.05-.59.14-.5.22-.41.27-.33.32-.27.35-.2.36-.15.37-.1.35-.07.32-.04.27-.02.21v3.06H3.17l-.21-.03-.28-.07-.32-.12-.35-.18-.36-.26-.36-.36-.35-.46-.32-.59-.28-.73-.21-.88-.14-1.05-.05-1.23.06-1.22.16-1.04.24-.87.32-.71.36-.57.4-.44.42-.33.42-.24.4-.16.36-.1.32-.05.24-.01h.16l.06.01h8.16v-.83H6.18l-.01-2.75-.02-.37.05-.34.11-.31.17-.28.25-.26.31-.23.38-.2.44-.18.51-.15.58-.12.64-.1.71-.06.77-.04.84-.02 1.27.05zm-6.3 1.98l-.23.33-.08.41.08.41.23.34.33.22.41.09.41-.09.33-.22.23-.34.08-.41-.08-.41-.23-.33-.33-.22-.41-.09-.41.09zm13.09 3.95l.28.06.32.12.35.18.36.27.36.35.35.47.32.59.28.73.21.88.14 1.04.05 1.23-.06 1.23-.16 1.04-.24.86-.32.71-.36.57-.4.45-.42.33-.42.24-.4.16-.36.09-.32.05-.24.02-.16-.01h-8.22v.82h5.84l.01 2.76.02.36-.05.34-.11.31-.17.29-.25.25-.31.24-.38.2-.44.17-.51.15-.58.13-.64.09-.71.07-.77.04-.84.01-1.27-.04-1.07-.14-.9-.2-.73-.25-.59-.3-.45-.33-.34-.34-.25-.34-.16-.33-.1-.3-.04-.25-.02-.2.01-.13v-5.34l.05-.64.13-.54.21-.46.26-.38.3-.32.33-.24.35-.2.35-.14.33-.1.3-.06.26-.04.21-.02.13-.01h5.84l.69-.05.59-.14.5-.21.41-.28.33-.32.27-.35.2-.36.15-.36.1-.35.07-.32.04-.28.02-.21V6.07h2.09l.14.01zm-6.47 14.25l-.23.33-.08.41.08.41.23.33.33.23.41.08.41-.08.33-.23.23-.33.08-.41-.08-.41-.23-.33-.33-.23-.41-.08-.41.08z"/>
            </svg>
            Python
          </button>
        </div>
      </div>

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
          <p><strong>JavaScript:</strong> <code>input</code>, <code>config</code>, <code>ctx.secrets</code> が利用可能</p>
          <p><strong>Python:</strong> <code>input</code>, <code>config</code>, <code>secrets</code> が利用可能</p>
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

.language-tabs {
  display: flex;
  gap: 0.5rem;
}

.language-tab {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.375rem 0.75rem;
  font-size: 0.75rem;
  font-weight: 500;
  background: var(--color-surface-raised);
  border: 1px solid var(--color-border);
  border-radius: 6px;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.2s;
}

.language-tab:hover:not(:disabled) {
  border-color: var(--color-border-hover);
  color: var(--color-text);
}

.language-tab.active {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: white;
}

.language-tab:disabled {
  opacity: 0.5;
  cursor: not-allowed;
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
