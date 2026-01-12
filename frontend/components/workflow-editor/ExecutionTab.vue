<script setup lang="ts">
import type { Step, Run, StepRun } from '~/types/api'
import type { ExecutionLog, ABTestConfig, PromptVariant } from '~/types/execution'

const { t } = useI18n()
const runs = useRuns()
const toast = useToast()

const props = defineProps<{
  step: Step | null
  workflowId: string
  latestRun: Run | null
}>()

const emit = defineEmits<{
  (e: 'execute', data: { stepId: string; input: object; mode: 'test' | 'production' }): void
  (e: 'execute-workflow', mode: 'test' | 'production', input: object): void
  (e: 'log', log: ExecutionLog): void
}>()

// Execution state
const executing = ref(false)
const executionMode = ref<'single' | 'workflow' | 'ab-test' | 'prompt-test'>('single')

// Custom input state
const useCustomInput = ref(false)
const customInputJson = ref('{}')
const inputError = ref<string | null>(null)

// A/B Test state
const abTestConfigs = ref<ABTestConfig[]>([
  { id: '1', name: 'Variant A', provider: 'openai', model: 'gpt-4', enabled: true },
  { id: '2', name: 'Variant B', provider: 'anthropic', model: 'claude-3-sonnet', enabled: true },
])

// Prompt test state
const promptVariants = ref<PromptVariant[]>([
  { id: '1', name: 'Original', prompt: '', enabled: true },
])

// Step execution history
const stepHistory = ref<StepRun[]>([])
const loadingHistory = ref(false)

// Parse and validate custom input
function parseCustomInput(): object | null {
  inputError.value = null
  if (!useCustomInput.value) return {}

  try {
    const parsed = JSON.parse(customInputJson.value)
    if (typeof parsed !== 'object' || parsed === null) {
      inputError.value = t('execution.errors.invalidJsonObject')
      return null
    }
    return parsed
  } catch (e) {
    inputError.value = t('execution.errors.invalidJson')
    return null
  }
}

// Execute single step
async function executeStep(mode: 'test' | 'production') {
  if (!props.step || !props.latestRun) {
    toast.error(t('execution.errors.noRunAvailable'))
    return
  }

  const input = parseCustomInput()
  if (input === null) return

  executing.value = true

  // Emit log
  emit('log', {
    id: crypto.randomUUID(),
    timestamp: new Date(),
    level: 'info',
    message: t('execution.logs.startingStep', { name: props.step.name }),
    stepId: props.step.id,
    stepName: props.step.name,
  })

  try {
    const response = await runs.executeSingleStep(
      props.latestRun.id,
      props.step.id,
      Object.keys(input).length > 0 ? input : undefined
    )

    emit('log', {
      id: crypto.randomUUID(),
      timestamp: new Date(),
      level: 'success',
      message: t('execution.logs.stepCompleted', { name: props.step.name }),
      stepId: props.step.id,
      stepName: props.step.name,
      data: response.data,
    })

    toast.success(t('execution.stepExecuted'))
    await loadStepHistory()
  } catch (e) {
    emit('log', {
      id: crypto.randomUUID(),
      timestamp: new Date(),
      level: 'error',
      message: t('execution.logs.stepFailed', { name: props.step?.name || '', error: e instanceof Error ? e.message : 'Unknown error' }),
      stepId: props.step?.id,
      stepName: props.step?.name,
    })
    toast.error(t('execution.errors.executionFailed'), e instanceof Error ? e.message : undefined)
  } finally {
    executing.value = false
  }
}

// Execute entire workflow
async function executeWorkflow(mode: 'test' | 'production') {
  const input = parseCustomInput()
  if (input === null) return

  executing.value = true

  emit('log', {
    id: crypto.randomUUID(),
    timestamp: new Date(),
    level: 'info',
    message: t('execution.logs.startingWorkflow', { mode }),
  })

  try {
    const response = await runs.create(props.workflowId, {
      mode,
      input: Object.keys(input).length > 0 ? input : {}
    })

    emit('log', {
      id: crypto.randomUUID(),
      timestamp: new Date(),
      level: 'success',
      message: t('execution.logs.workflowStarted', { id: response.data.id }),
      data: response.data,
    })

    emit('execute-workflow', mode, input)
    toast.success(t('execution.workflowStarted'))
  } catch (e) {
    emit('log', {
      id: crypto.randomUUID(),
      timestamp: new Date(),
      level: 'error',
      message: t('execution.logs.workflowFailed', { error: e instanceof Error ? e.message : 'Unknown error' }),
    })
    toast.error(t('execution.errors.executionFailed'), e instanceof Error ? e.message : undefined)
  } finally {
    executing.value = false
  }
}

// Run A/B test
async function runABTest() {
  if (!props.step) return

  const enabledConfigs = abTestConfigs.value.filter(c => c.enabled)
  if (enabledConfigs.length < 2) {
    toast.error(t('execution.errors.needTwoVariants'))
    return
  }

  const input = parseCustomInput()
  if (input === null) return

  executing.value = true

  emit('log', {
    id: crypto.randomUUID(),
    timestamp: new Date(),
    level: 'info',
    message: t('execution.logs.startingABTest', { count: enabledConfigs.length }),
    stepId: props.step.id,
    stepName: props.step.name,
  })

  // TODO: Implement actual A/B test execution with different configs
  // For now, just log the intent
  for (const config of enabledConfigs) {
    emit('log', {
      id: crypto.randomUUID(),
      timestamp: new Date(),
      level: 'info',
      message: t('execution.logs.runningVariant', { name: config.name, provider: config.provider, model: config.model }),
    })
  }

  executing.value = false
  toast.info(t('execution.abTestPlaceholder'))
}

// Run prompt test
async function runPromptTest() {
  if (!props.step) return

  const enabledVariants = promptVariants.value.filter(v => v.enabled && v.prompt.trim())
  if (enabledVariants.length === 0) {
    toast.error(t('execution.errors.needPromptVariant'))
    return
  }

  const input = parseCustomInput()
  if (input === null) return

  executing.value = true

  emit('log', {
    id: crypto.randomUUID(),
    timestamp: new Date(),
    level: 'info',
    message: t('execution.logs.startingPromptTest', { count: enabledVariants.length }),
    stepId: props.step.id,
    stepName: props.step.name,
  })

  // TODO: Implement actual prompt test execution
  for (const variant of enabledVariants) {
    emit('log', {
      id: crypto.randomUUID(),
      timestamp: new Date(),
      level: 'info',
      message: t('execution.logs.testingPrompt', { name: variant.name }),
    })
  }

  executing.value = false
  toast.info(t('execution.promptTestPlaceholder'))
}

// Load step execution history
async function loadStepHistory() {
  if (!props.step || !props.latestRun) return

  loadingHistory.value = true
  try {
    const response = await runs.getStepHistory(props.latestRun.id, props.step.id)
    stepHistory.value = response.data
  } catch (e) {
    console.error('Failed to load step history:', e)
  } finally {
    loadingHistory.value = false
  }
}

// A/B Test helpers
function addABVariant() {
  const id = (abTestConfigs.value.length + 1).toString()
  abTestConfigs.value.push({
    id,
    name: `Variant ${String.fromCharCode(65 + abTestConfigs.value.length)}`,
    provider: 'openai',
    model: 'gpt-4',
    enabled: true,
  })
}

function removeABVariant(id: string) {
  abTestConfigs.value = abTestConfigs.value.filter(c => c.id !== id)
}

// Prompt test helpers
function addPromptVariant() {
  const id = (promptVariants.value.length + 1).toString()
  promptVariants.value.push({
    id,
    name: `Variant ${promptVariants.value.length + 1}`,
    prompt: '',
    enabled: true,
  })
}

function removePromptVariant(id: string) {
  promptVariants.value = promptVariants.value.filter(v => v.id !== id)
}

// Copy prompt from current step config
function copyCurrentPrompt(index: number) {
  if (!props.step) return
  const config = props.step.config as Record<string, unknown>
  const prompt = config.prompt as string || ''
  promptVariants.value[index].prompt = prompt
}

// Format step status
function formatStatus(status: string): string {
  return t(`execution.status.${status}`) || status
}

// Format duration
function formatDuration(ms?: number): string {
  if (!ms) return '-'
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(2)}s`
}

// Watch for step changes
watch(() => props.step, () => {
  if (props.step && props.latestRun) {
    loadStepHistory()
  }
}, { immediate: true })

// Watch for latest run changes
watch(() => props.latestRun, () => {
  if (props.step && props.latestRun) {
    loadStepHistory()
  }
})

// Check if step is LLM type (for A/B and prompt testing)
const isLLMStep = computed(() => {
  return props.step?.type === 'llm' || props.step?.type === 'router'
})
</script>

<template>
  <div class="execution-tab">
    <!-- Mode Selector -->
    <div class="mode-selector">
      <button
        :class="['mode-btn', { active: executionMode === 'single' }]"
        @click="executionMode = 'single'"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polygon points="5 3 19 12 5 21 5 3"></polygon>
        </svg>
        {{ t('execution.modes.single') }}
      </button>
      <button
        :class="['mode-btn', { active: executionMode === 'workflow' }]"
        @click="executionMode = 'workflow'"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"></circle>
          <polygon points="10 8 16 12 10 16 10 8"></polygon>
        </svg>
        {{ t('execution.modes.workflow') }}
      </button>
      <button
        v-if="isLLMStep"
        :class="['mode-btn', { active: executionMode === 'ab-test' }]"
        @click="executionMode = 'ab-test'"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M18 20V10"></path>
          <path d="M12 20V4"></path>
          <path d="M6 20v-6"></path>
        </svg>
        {{ t('execution.modes.abTest') }}
      </button>
      <button
        v-if="isLLMStep"
        :class="['mode-btn', { active: executionMode === 'prompt-test' }]"
        @click="executionMode = 'prompt-test'"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"></path>
          <polyline points="14 2 14 8 20 8"></polyline>
          <line x1="16" y1="13" x2="8" y2="13"></line>
          <line x1="16" y1="17" x2="8" y2="17"></line>
        </svg>
        {{ t('execution.modes.promptTest') }}
      </button>
    </div>

    <!-- Single Step Execution -->
    <div v-if="executionMode === 'single'" class="execution-content">
      <div v-if="!step" class="empty-state">
        <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect>
          <line x1="9" y1="9" x2="15" y2="15"></line>
          <line x1="15" y1="9" x2="9" y2="15"></line>
        </svg>
        <p>{{ t('execution.selectStepToExecute') }}</p>
      </div>

      <div v-else class="step-execution">
        <div class="current-step-info">
          <div class="step-badge" :style="{ backgroundColor: getStepColor(step.type) }">
            {{ step.type }}
          </div>
          <span class="step-name">{{ step.name }}</span>
        </div>

        <!-- Custom Input -->
        <div class="input-section">
          <label class="checkbox-label">
            <input v-model="useCustomInput" type="checkbox">
            <span>{{ t('execution.useCustomInput') }}</span>
          </label>

          <div v-if="useCustomInput" class="input-editor">
            <textarea
              v-model="customInputJson"
              class="json-input"
              rows="4"
              :placeholder="t('execution.inputPlaceholder')"
            ></textarea>
            <p v-if="inputError" class="input-error">{{ inputError }}</p>
          </div>
        </div>

        <!-- Execution Buttons -->
        <div class="execution-buttons">
          <button
            class="btn btn-primary"
            :disabled="executing || !latestRun"
            @click="executeStep('test')"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polygon points="5 3 19 12 5 21 5 3"></polygon>
            </svg>
            {{ executing ? t('execution.executing') : t('execution.executeTest') }}
          </button>
        </div>

        <p v-if="!latestRun" class="warning-text">
          {{ t('execution.noRunWarning') }}
        </p>

        <!-- Step History -->
        <div v-if="stepHistory.length > 0" class="step-history">
          <h4 class="history-title">{{ t('execution.history') }}</h4>
          <div class="history-list">
            <div v-for="run in stepHistory.slice(0, 5)" :key="run.id" class="history-item">
              <span :class="['status-badge', run.status]">{{ formatStatus(run.status) }}</span>
              <span class="history-duration">{{ formatDuration(run.duration_ms) }}</span>
              <span class="history-attempt">{{ t('execution.attempt', { n: run.attempt }) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Workflow Execution -->
    <div v-else-if="executionMode === 'workflow'" class="execution-content">
      <div class="workflow-execution">
        <p class="section-desc">{{ t('execution.workflowDesc') }}</p>

        <!-- Custom Input -->
        <div class="input-section">
          <label class="checkbox-label">
            <input v-model="useCustomInput" type="checkbox">
            <span>{{ t('execution.useCustomInput') }}</span>
          </label>

          <div v-if="useCustomInput" class="input-editor">
            <textarea
              v-model="customInputJson"
              class="json-input"
              rows="4"
              :placeholder="t('execution.inputPlaceholder')"
            ></textarea>
            <p v-if="inputError" class="input-error">{{ inputError }}</p>
          </div>
        </div>

        <!-- Execution Buttons -->
        <div class="execution-buttons">
          <button
            class="btn btn-outline"
            :disabled="executing"
            @click="executeWorkflow('test')"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polygon points="5 3 19 12 5 21 5 3"></polygon>
            </svg>
            {{ t('execution.testRun') }}
          </button>
          <button
            class="btn btn-primary"
            :disabled="executing"
            @click="executeWorkflow('production')"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon>
            </svg>
            {{ t('execution.productionRun') }}
          </button>
        </div>
      </div>
    </div>

    <!-- A/B Test -->
    <div v-else-if="executionMode === 'ab-test'" class="execution-content">
      <div v-if="!step || !isLLMStep" class="empty-state">
        <p>{{ t('execution.selectLLMStep') }}</p>
      </div>

      <div v-else class="ab-test">
        <p class="section-desc">{{ t('execution.abTestDesc') }}</p>

        <div class="variants-list">
          <div v-for="(config, index) in abTestConfigs" :key="config.id" class="variant-item">
            <div class="variant-header">
              <label class="checkbox-label">
                <input v-model="config.enabled" type="checkbox">
                <input
                  v-model="config.name"
                  type="text"
                  class="variant-name-input"
                  :placeholder="t('execution.variantName')"
                >
              </label>
              <button
                v-if="abTestConfigs.length > 2"
                class="btn-icon btn-remove"
                @click="removeABVariant(config.id)"
              >
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="18" y1="6" x2="6" y2="18"></line>
                  <line x1="6" y1="6" x2="18" y2="18"></line>
                </svg>
              </button>
            </div>
            <div class="variant-config">
              <select v-model="config.provider" class="form-input form-input-sm">
                <option value="openai">OpenAI</option>
                <option value="anthropic">Anthropic</option>
                <option value="mock">Mock</option>
              </select>
              <input
                v-model="config.model"
                type="text"
                class="form-input form-input-sm"
                :placeholder="t('execution.modelName')"
              >
            </div>
          </div>
        </div>

        <button class="btn btn-add-variant" @click="addABVariant">
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="12" y1="5" x2="12" y2="19"></line>
            <line x1="5" y1="12" x2="19" y2="12"></line>
          </svg>
          {{ t('execution.addVariant') }}
        </button>

        <!-- Custom Input -->
        <div class="input-section">
          <label class="checkbox-label">
            <input v-model="useCustomInput" type="checkbox">
            <span>{{ t('execution.useCustomInput') }}</span>
          </label>

          <div v-if="useCustomInput" class="input-editor">
            <textarea
              v-model="customInputJson"
              class="json-input"
              rows="3"
              :placeholder="t('execution.inputPlaceholder')"
            ></textarea>
          </div>
        </div>

        <button
          class="btn btn-primary btn-full"
          :disabled="executing || abTestConfigs.filter(c => c.enabled).length < 2"
          @click="runABTest"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M18 20V10"></path>
            <path d="M12 20V4"></path>
            <path d="M6 20v-6"></path>
          </svg>
          {{ t('execution.runABTest') }}
        </button>
      </div>
    </div>

    <!-- Prompt Test -->
    <div v-else-if="executionMode === 'prompt-test'" class="execution-content">
      <div v-if="!step || !isLLMStep" class="empty-state">
        <p>{{ t('execution.selectLLMStep') }}</p>
      </div>

      <div v-else class="prompt-test">
        <p class="section-desc">{{ t('execution.promptTestDesc') }}</p>

        <div class="variants-list">
          <div v-for="(variant, index) in promptVariants" :key="variant.id" class="variant-item">
            <div class="variant-header">
              <label class="checkbox-label">
                <input v-model="variant.enabled" type="checkbox">
                <input
                  v-model="variant.name"
                  type="text"
                  class="variant-name-input"
                  :placeholder="t('execution.variantName')"
                >
              </label>
              <div class="variant-actions">
                <button
                  class="btn-icon btn-copy"
                  :title="t('execution.copyCurrentPrompt')"
                  @click="copyCurrentPrompt(index)"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                    <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
                  </svg>
                </button>
                <button
                  v-if="promptVariants.length > 1"
                  class="btn-icon btn-remove"
                  @click="removePromptVariant(variant.id)"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="18" y1="6" x2="6" y2="18"></line>
                    <line x1="6" y1="6" x2="18" y2="18"></line>
                  </svg>
                </button>
              </div>
            </div>
            <textarea
              v-model="variant.prompt"
              class="prompt-input"
              rows="3"
              :placeholder="t('execution.promptPlaceholder')"
            ></textarea>
          </div>
        </div>

        <button class="btn btn-add-variant" @click="addPromptVariant">
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="12" y1="5" x2="12" y2="19"></line>
            <line x1="5" y1="12" x2="19" y2="12"></line>
          </svg>
          {{ t('execution.addPromptVariant') }}
        </button>

        <!-- Custom Input -->
        <div class="input-section">
          <label class="checkbox-label">
            <input v-model="useCustomInput" type="checkbox">
            <span>{{ t('execution.useCustomInput') }}</span>
          </label>

          <div v-if="useCustomInput" class="input-editor">
            <textarea
              v-model="customInputJson"
              class="json-input"
              rows="3"
              :placeholder="t('execution.inputPlaceholder')"
            ></textarea>
          </div>
        </div>

        <button
          class="btn btn-primary btn-full"
          :disabled="executing || promptVariants.filter(v => v.enabled && v.prompt.trim()).length === 0"
          @click="runPromptTest"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polygon points="5 3 19 12 5 21 5 3"></polygon>
          </svg>
          {{ t('execution.runPromptTest') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
// Step type colors (matching DagEditor)
const stepTypeColors: Record<string, string> = {
  start: '#10b981',
  llm: '#3b82f6',
  tool: '#22c55e',
  condition: '#f59e0b',
  switch: '#eab308',
  map: '#8b5cf6',
  join: '#6366f1',
  subflow: '#ec4899',
  loop: '#14b8a6',
  wait: '#64748b',
  function: '#f97316',
  router: '#a855f7',
  human_in_loop: '#ef4444',
  filter: '#06b6d4',
  split: '#0ea5e9',
  aggregate: '#0284c7',
  error: '#dc2626',
  note: '#9ca3af',
  log: '#10b981',
}

function getStepColor(type: string): string {
  return stepTypeColors[type] || '#64748b'
}
</script>

<style scoped>
.execution-tab {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

/* Mode Selector */
.mode-selector {
  display: flex;
  flex-wrap: wrap;
  gap: 0.25rem;
  padding: 0.5rem;
  background: var(--color-background);
  border-radius: 6px;
  margin-bottom: 0.75rem;
}

.mode-btn {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.375rem 0.5rem;
  font-size: 0.6875rem;
  font-weight: 500;
  color: var(--color-text-secondary);
  background: transparent;
  border: 1px solid transparent;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.15s;
}

.mode-btn:hover {
  color: var(--color-text);
  background: white;
}

.mode-btn.active {
  color: var(--color-primary);
  background: white;
  border-color: var(--color-primary);
  box-shadow: 0 1px 3px rgba(59, 130, 246, 0.1);
}

.mode-btn svg {
  flex-shrink: 0;
}

/* Execution Content */
.execution-content {
  flex: 1;
  overflow-y: auto;
  padding: 0.5rem 0;
}

/* Empty State */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 2rem 1rem;
  text-align: center;
  color: var(--color-text-secondary);
}

.empty-state svg {
  opacity: 0.5;
  margin-bottom: 0.75rem;
}

.empty-state p {
  font-size: 0.8125rem;
  margin: 0;
}

/* Step Execution */
.step-execution,
.workflow-execution,
.ab-test,
.prompt-test {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.current-step-info {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem;
  background: var(--color-background);
  border-radius: 6px;
}

.step-badge {
  font-size: 0.625rem;
  font-weight: 600;
  text-transform: uppercase;
  color: white;
  padding: 0.125rem 0.375rem;
  border-radius: 3px;
}

.step-name {
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--color-text);
}

.section-desc {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  margin: 0;
  line-height: 1.4;
}

/* Input Section */
.input-section {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.8125rem;
  cursor: pointer;
}

.checkbox-label input[type="checkbox"] {
  width: 14px;
  height: 14px;
  cursor: pointer;
}

.input-editor {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.json-input,
.prompt-input {
  width: 100%;
  padding: 0.5rem;
  font-size: 0.75rem;
  font-family: 'SF Mono', Monaco, monospace;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  resize: vertical;
  min-height: 60px;
}

.json-input:focus,
.prompt-input:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.input-error {
  font-size: 0.6875rem;
  color: var(--color-error);
  margin: 0;
}

/* Execution Buttons */
.execution-buttons {
  display: flex;
  gap: 0.5rem;
}

.execution-buttons .btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.375rem;
  font-size: 0.8125rem;
}

.btn-full {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.375rem;
}

.warning-text {
  font-size: 0.6875rem;
  color: var(--color-warning);
  margin: 0;
  padding: 0.5rem;
  background: #fef3c7;
  border-radius: 4px;
}

/* Step History */
.step-history {
  margin-top: 0.5rem;
}

.history-title {
  font-size: 0.6875rem;
  font-weight: 600;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin: 0 0 0.5rem 0;
}

.history-list {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.history-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.375rem 0.5rem;
  background: var(--color-background);
  border-radius: 4px;
  font-size: 0.6875rem;
}

.status-badge {
  font-size: 0.5625rem;
  font-weight: 600;
  text-transform: uppercase;
  padding: 0.125rem 0.375rem;
  border-radius: 3px;
}

.status-badge.completed {
  background: #dcfce7;
  color: #16a34a;
}

.status-badge.failed {
  background: #fee2e2;
  color: #dc2626;
}

.status-badge.running {
  background: #dbeafe;
  color: #2563eb;
}

.status-badge.pending {
  background: #f3f4f6;
  color: #6b7280;
}

.history-duration {
  color: var(--color-text-secondary);
}

.history-attempt {
  color: var(--color-text-secondary);
  margin-left: auto;
}

/* Variants List */
.variants-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.variant-item {
  padding: 0.5rem;
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 6px;
}

.variant-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.5rem;
}

.variant-name-input {
  flex: 1;
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  background: white;
}

.variant-actions {
  display: flex;
  gap: 0.25rem;
}

.variant-config {
  display: flex;
  gap: 0.5rem;
}

.form-input-sm {
  padding: 0.375rem 0.5rem;
  font-size: 0.75rem;
}

.btn-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  padding: 0;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 4px;
  cursor: pointer;
  color: var(--color-text-secondary);
  transition: all 0.15s;
}

.btn-copy:hover {
  background: #dbeafe;
  color: #2563eb;
}

.btn-remove:hover {
  background: #fee2e2;
  color: #dc2626;
}

.btn-add-variant {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.375rem;
  width: 100%;
  padding: 0.5rem;
  font-size: 0.75rem;
  color: var(--color-primary);
  background: transparent;
  border: 1px dashed var(--color-primary);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-add-variant:hover {
  background: rgba(59, 130, 246, 0.05);
}

/* Scrollbar */
.execution-content::-webkit-scrollbar {
  width: 6px;
}

.execution-content::-webkit-scrollbar-track {
  background: transparent;
}

.execution-content::-webkit-scrollbar-thumb {
  background: var(--color-border);
  border-radius: 3px;
}
</style>
