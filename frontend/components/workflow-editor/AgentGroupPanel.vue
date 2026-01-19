<script setup lang="ts">
import type { BlockGroup, AgentConfig, Step } from '~/types/api'
import { useDebounceFn } from '@vueuse/core'

const props = defineProps<{
  group: BlockGroup
  childSteps?: Step[]
  readonly?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:config', config: AgentConfig): void
  (e: 'close'): void
}>()

// Local state for form
const localConfig = ref<AgentConfig>({
  provider: 'anthropic',
  model: 'claude-sonnet-4-20250514',
  system_prompt: '',
  max_iterations: 10,
  temperature: 0.7,
  tool_choice: 'auto',
  enable_memory: false,
  memory_window: 20,
})

// Initialize from group config
watchEffect(() => {
  if (props.group?.config) {
    const config = props.group.config as AgentConfig
    localConfig.value = {
      provider: config.provider || 'anthropic',
      model: config.model || 'claude-sonnet-4-20250514',
      system_prompt: config.system_prompt || '',
      max_iterations: config.max_iterations || 10,
      temperature: config.temperature || 0.7,
      tool_choice: config.tool_choice || 'auto',
      enable_memory: config.enable_memory || false,
      memory_window: config.memory_window || 20,
    }
  }
})

// Provider options
const providers = [
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'openai', label: 'OpenAI' },
]

// Tool choice options
const toolChoiceOptions = [
  { value: 'auto', label: 'Auto', description: 'Model decides when to use tools' },
  { value: 'required', label: 'Required', description: 'Model must use at least one tool' },
  { value: 'none', label: 'None', description: 'Model cannot use tools' },
]

// Save handler
function handleSave() {
  emit('update:config', { ...localConfig.value })
}

// Debounced auto-save
const debouncedSave = useDebounceFn(handleSave, 500)

// Watch for changes and auto-save
watch(localConfig, () => {
  if (!props.readonly) {
    debouncedSave()
  }
}, { deep: true })
</script>

<template>
  <div class="agent-group-panel">
    <!-- Header -->
    <div class="panel-header">
      <div class="header-content">
        <span class="header-icon">🤖</span>
        <span class="header-title">{{ group.name }}</span>
        <span class="header-type">Agent</span>
      </div>
      <button class="close-btn" @click="emit('close')">
        <Icon name="lucide:x" size="16" />
      </button>
    </div>

    <div class="panel-content">
      <!-- Model Settings Section -->
      <div class="section">
        <div class="section-header">
          <Icon name="lucide:bot" size="14" />
          <span>Model Settings</span>
        </div>

        <!-- Provider -->
        <div class="field">
          <label>Provider</label>
          <select v-model="localConfig.provider" :disabled="readonly">
            <option v-for="p in providers" :key="p.value" :value="p.value">
              {{ p.label }}
            </option>
          </select>
        </div>

        <!-- Model -->
        <div class="field">
          <label>Model</label>
          <input
            v-model="localConfig.model"
            type="text"
            placeholder="e.g., claude-sonnet-4-20250514"
            :disabled="readonly"
          />
        </div>
      </div>

      <!-- Agent Settings Section -->
      <div class="section">
        <div class="section-header">
          <Icon name="lucide:settings" size="14" />
          <span>Agent Settings</span>
        </div>

        <!-- System Prompt -->
        <div class="field">
          <label>System Prompt</label>
          <textarea
            v-model="localConfig.system_prompt"
            rows="8"
            placeholder="Define the agent's behavior and capabilities..."
            :disabled="readonly"
          />
        </div>

        <!-- Max Iterations -->
        <div class="field">
          <label>Max Iterations</label>
          <div class="slider-field">
            <input
              v-model.number="localConfig.max_iterations"
              type="range"
              min="1"
              max="50"
              :disabled="readonly"
            />
            <span class="slider-value">{{ localConfig.max_iterations }}</span>
          </div>
        </div>

        <!-- Temperature -->
        <div class="field">
          <label>Temperature</label>
          <div class="slider-field">
            <input
              v-model.number="localConfig.temperature"
              type="range"
              min="0"
              max="2"
              step="0.1"
              :disabled="readonly"
            />
            <span class="slider-value">{{ localConfig.temperature?.toFixed(1) }}</span>
          </div>
        </div>

        <!-- Tool Choice -->
        <div class="field">
          <label>Tool Choice</label>
          <select v-model="localConfig.tool_choice" :disabled="readonly">
            <option v-for="tc in toolChoiceOptions" :key="tc.value" :value="tc.value">
              {{ tc.label }}
            </option>
          </select>
          <span class="field-description">
            {{ toolChoiceOptions.find(tc => tc.value === localConfig.tool_choice)?.description }}
          </span>
        </div>
      </div>

      <!-- Memory Settings Section -->
      <div class="section">
        <div class="section-header">
          <Icon name="lucide:database" size="14" />
          <span>Memory Settings</span>
        </div>

        <!-- Enable Memory -->
        <div class="field checkbox-field">
          <label>
            <input
              v-model="localConfig.enable_memory"
              type="checkbox"
              :disabled="readonly"
            />
            Enable Conversation Memory
          </label>
        </div>

        <!-- Memory Window (only shown when memory is enabled) -->
        <div v-if="localConfig.enable_memory" class="field">
          <label>Memory Window</label>
          <div class="slider-field">
            <input
              v-model.number="localConfig.memory_window"
              type="range"
              min="1"
              max="100"
              :disabled="readonly"
            />
            <span class="slider-value">{{ localConfig.memory_window }} messages</span>
          </div>
        </div>
      </div>

      <!-- Available Tools Section (Read-only) -->
      <div class="section">
        <div class="section-header">
          <Icon name="lucide:wrench" size="14" />
          <span>Available Tools (Child Steps)</span>
        </div>

        <div v-if="childSteps && childSteps.length > 0" class="tools-list">
          <div v-for="step in childSteps" :key="step.id" class="tool-item">
            <Icon name="lucide:function" size="14" />
            <span class="tool-name">{{ step.name }}</span>
            <span class="tool-type">{{ step.type }}</span>
          </div>
        </div>
        <div v-else class="tools-empty">
          <p>No tools configured.</p>
          <p class="hint">Drag steps into this group to make them available as tools.</p>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.agent-group-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--color-bg-primary);
  border-left: 1px solid var(--color-border);
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
}

.header-content {
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-icon {
  font-size: 18px;
}

.header-title {
  font-weight: 600;
  color: var(--color-text-primary);
}

.header-type {
  font-size: 12px;
  color: #10b981;
  background: rgba(16, 185, 129, 0.1);
  padding: 2px 8px;
  border-radius: 4px;
}

.close-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border: none;
  background: transparent;
  border-radius: 4px;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s;
}

.close-btn:hover {
  background: var(--color-bg-hover);
  color: var(--color-text-primary);
}

.panel-content {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
}

.section {
  margin-bottom: 20px;
}

.section-header {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--color-border);
}

.field {
  margin-bottom: 12px;
}

.field label {
  display: block;
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-primary);
  margin-bottom: 6px;
}

.field input[type="text"],
.field select,
.field textarea {
  width: 100%;
  padding: 8px 12px;
  font-size: 13px;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-bg-primary);
  color: var(--color-text-primary);
  transition: border-color 0.15s;
}

.field input[type="text"]:focus,
.field select:focus,
.field textarea:focus {
  outline: none;
  border-color: #10b981;
}

.field textarea {
  resize: vertical;
  min-height: 120px;
  font-family: monospace;
}

.slider-field {
  display: flex;
  align-items: center;
  gap: 12px;
}

.slider-field input[type="range"] {
  flex: 1;
  height: 4px;
  border-radius: 2px;
  background: var(--color-bg-tertiary);
  accent-color: #10b981;
}

.slider-value {
  min-width: 60px;
  text-align: right;
  font-size: 13px;
  color: var(--color-text-secondary);
}

.field-description {
  display: block;
  font-size: 11px;
  color: var(--color-text-tertiary);
  margin-top: 4px;
}

.checkbox-field label {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.checkbox-field input[type="checkbox"] {
  width: 16px;
  height: 16px;
  accent-color: #10b981;
}

.tools-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.tool-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: 6px;
}

.tool-name {
  flex: 1;
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-primary);
}

.tool-type {
  font-size: 11px;
  color: var(--color-text-tertiary);
  background: var(--color-bg-tertiary);
  padding: 2px 6px;
  border-radius: 4px;
}

.tools-empty {
  text-align: center;
  padding: 16px;
  background: var(--color-bg-secondary);
  border: 1px dashed var(--color-border);
  border-radius: 6px;
}

.tools-empty p {
  margin: 0;
  font-size: 13px;
  color: var(--color-text-secondary);
}

.tools-empty .hint {
  margin-top: 8px;
  font-size: 12px;
  color: var(--color-text-tertiary);
}
</style>
