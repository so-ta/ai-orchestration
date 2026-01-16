<script setup lang="ts">
/**
 * ProcessEditor - Pre/PostProcess編集コンポーネント
 *
 * 継承ブロックの入力変換（preProcess）と出力変換（postProcess）を編集するタブUI。
 */

const props = withDefaults(defineProps<{
  preProcess: string
  postProcess: string
  preProcessTemplate?: string
  postProcessTemplate?: string
}>(), {
  preProcessTemplate: '',
  postProcessTemplate: '',
})

const emit = defineEmits<{
  'update:preProcess': [value: string]
  'update:postProcess': [value: string]
}>()

const { t } = useI18n()

// Active tab
const activeTab = ref<'pre' | 'post'>('pre')

// Local values for v-model
const localPreProcess = computed({
  get: () => props.preProcess,
  set: (value: string) => emit('update:preProcess', value),
})

const localPostProcess = computed({
  get: () => props.postProcess,
  set: (value: string) => emit('update:postProcess', value),
})

// Apply template
function applyPreTemplate() {
  if (props.preProcessTemplate) {
    localPreProcess.value = props.preProcessTemplate
  }
}

function applyPostTemplate() {
  if (props.postProcessTemplate) {
    localPostProcess.value = props.postProcessTemplate
  }
}
</script>

<template>
  <div class="process-editor">
    <div class="editor-tabs">
      <button
        class="tab-button"
        :class="{ active: activeTab === 'pre' }"
        @click="activeTab = 'pre'"
      >
        <span class="tab-icon">&#8678;</span>
        {{ t('blockEditor.preProcess') }}
      </button>
      <button
        class="tab-button"
        :class="{ active: activeTab === 'post' }"
        @click="activeTab = 'post'"
      >
        <span class="tab-icon">&#8680;</span>
        {{ t('blockEditor.postProcess') }}
      </button>
    </div>

    <div class="editor-content">
      <!-- PreProcess -->
      <div v-show="activeTab === 'pre'" class="editor-pane">
        <div class="help-panel">
          <h4 class="help-title">{{ t('blockEditor.preProcessHelp') }}</h4>
          <ul class="help-list">
            <li><code>input</code>: {{ t('blockEditor.helpInputData') }}</li>
            <li><code>config</code>: {{ t('blockEditor.helpConfigValues') }}</li>
            <li><code>ctx.secrets</code>: {{ t('blockEditor.helpSecrets') }}</li>
            <li><code>return</code>: {{ t('blockEditor.helpPreReturn') }}</li>
          </ul>
        </div>

        <div class="code-section">
          <div class="code-header">
            <label class="code-label">{{ t('blockEditor.preProcessCode') }}</label>
            <button
              v-if="preProcessTemplate"
              class="template-btn"
              @click="applyPreTemplate"
            >
              {{ t('blockEditor.applyTemplate') }}
            </button>
          </div>
          <textarea
            v-model="localPreProcess"
            class="code-editor"
            rows="10"
            spellcheck="false"
            :placeholder="t('blockEditor.preProcessPlaceholder')"
          />
        </div>

        <details class="template-details">
          <summary>{{ t('blockEditor.exampleTemplate') }}</summary>
          <pre class="template-preview">{{ preProcessTemplate || t('blockEditor.noTemplate') }}</pre>
        </details>
      </div>

      <!-- PostProcess -->
      <div v-show="activeTab === 'post'" class="editor-pane">
        <div class="help-panel">
          <h4 class="help-title">{{ t('blockEditor.postProcessHelp') }}</h4>
          <ul class="help-list">
            <li><code>input</code>: {{ t('blockEditor.helpParentOutput') }}</li>
            <li><code>config</code>: {{ t('blockEditor.helpConfigValues') }}</li>
            <li><code>return</code>: {{ t('blockEditor.helpPostReturn') }}</li>
          </ul>
        </div>

        <div class="code-section">
          <div class="code-header">
            <label class="code-label">{{ t('blockEditor.postProcessCode') }}</label>
            <button
              v-if="postProcessTemplate"
              class="template-btn"
              @click="applyPostTemplate"
            >
              {{ t('blockEditor.applyTemplate') }}
            </button>
          </div>
          <textarea
            v-model="localPostProcess"
            class="code-editor"
            rows="10"
            spellcheck="false"
            :placeholder="t('blockEditor.postProcessPlaceholder')"
          />
        </div>

        <details class="template-details">
          <summary>{{ t('blockEditor.exampleTemplate') }}</summary>
          <pre class="template-preview">{{ postProcessTemplate || t('blockEditor.noTemplate') }}</pre>
        </details>
      </div>
    </div>
  </div>
</template>

<style scoped>
.process-editor {
  border: 1px solid var(--color-border);
  border-radius: 0.5rem;
  overflow: hidden;
  margin-top: 1rem;
}

.editor-tabs {
  display: flex;
  border-bottom: 1px solid var(--color-border);
}

.tab-button {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.75rem 1rem;
  background: var(--color-background);
  border: none;
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s;
}

.tab-button:hover {
  background: var(--color-surface);
}

.tab-button.active {
  background: var(--color-surface);
  color: var(--color-primary);
  border-bottom: 2px solid var(--color-primary);
  margin-bottom: -1px;
}

.tab-icon {
  font-size: 1rem;
}

.editor-content {
  background: var(--color-surface);
}

.editor-pane {
  padding: 1rem;
}

.help-panel {
  background: rgba(99, 102, 241, 0.05);
  border: 1px solid rgba(99, 102, 241, 0.1);
  border-radius: 0.375rem;
  padding: 0.75rem;
  margin-bottom: 1rem;
}

.help-title {
  font-size: 0.8125rem;
  font-weight: 600;
  margin: 0 0 0.5rem 0;
  color: var(--color-text);
}

.help-list {
  margin: 0;
  padding-left: 1.25rem;
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
}

.help-list li {
  margin-bottom: 0.25rem;
}

.help-list code {
  background: rgba(99, 102, 241, 0.1);
  padding: 0.125rem 0.25rem;
  border-radius: 0.25rem;
  font-size: 0.75rem;
  font-family: 'Monaco', 'Menlo', monospace;
}

.code-section {
  margin-bottom: 0.75rem;
}

.code-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.5rem;
}

.code-label {
  font-size: 0.875rem;
  font-weight: 500;
}

.template-btn {
  padding: 0.25rem 0.5rem;
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 0.25rem;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s;
}

.template-btn:hover {
  background: var(--color-primary);
  color: white;
  border-color: var(--color-primary);
}

.code-editor {
  width: 100%;
  padding: 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: 0.375rem;
  background: var(--color-background);
  color: var(--color-text);
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.8125rem;
  line-height: 1.5;
  resize: vertical;
}

.code-editor:focus {
  outline: none;
  border-color: var(--color-primary);
}

.template-details {
  margin-top: 0.5rem;
}

.template-details summary {
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
  cursor: pointer;
  padding: 0.5rem 0;
}

.template-details summary:hover {
  color: var(--color-text);
}

.template-preview {
  margin: 0.5rem 0 0 0;
  padding: 0.75rem;
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 0.375rem;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  overflow-x: auto;
  white-space: pre-wrap;
}
</style>
